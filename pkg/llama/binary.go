package llama

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/majidkorai/gollama/pkg/model"
)

type BackendOption struct {
	Name   string
	Suffix string
	GPU    bool
}

func DetectGPUBackends() []BackendOption {
	var options []BackendOption
	options = append(options, BackendOption{Name: "CPU", Suffix: "", GPU: false})

	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		cmd := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			label := "CUDA"
			if len(lines) == 1 {
				label = fmt.Sprintf("CUDA (%s)", strings.TrimSpace(lines[0]))
			} else if len(lines) > 1 {
				label = fmt.Sprintf("CUDA (%d GPUs)", len(lines))
			}
			options = append(options, BackendOption{
				Name:   label,
				Suffix: "-cuda",
				GPU:    true,
			})
		}
	}

	if _, err := os.Stat("/opt/rocm"); err == nil {
		options = append(options, BackendOption{
			Name:   "ROCm",
			Suffix: "-rocm-7.2",
			GPU:    true,
		})
	}

	// Only offer Vulkan if the runtime library is available (prevents segfaults on bare metal)
	vulkanAvailable := func() bool {
		if runtime.GOOS == "darwin" {
			return true // macOS uses Metal via the default build, not Vulkan
		}
		// Check for Vulkan loader library
		for _, path := range []string{
			"/usr/lib/libvulkan.so.1",
			"/usr/lib/x86_64-linux-gnu/libvulkan.so.1",
			"/usr/lib/aarch64-linux-gnu/libvulkan.so.1",
			"/usr/local/lib/libvulkan.so.1",
		} {
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
		// Also check via ldconfig cache
		if _, err := os.Stat("/etc/ld.so.cache"); err == nil {
			cmd := exec.Command("ldconfig", "-p")
			out, _ := cmd.Output()
			if strings.Contains(string(out), "libvulkan.so.1") {
				return true
			}
		}
		return false
	}()
	if vulkanAvailable {
		options = append(options, BackendOption{Name: "Vulkan", Suffix: "-vulkan", GPU: true})
	}

	return options
}

func GetReleaseData() (string, map[string]string, error) {
	resp, err := model.HTTPClient.Get("https://api.github.com/repos/ggml-org/llama.cpp/releases/latest")
	if err != nil {
		return "", nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", nil, fmt.Errorf("parsing release: %w", err)
	}

	assets := make(map[string]string)
	for _, a := range release.Assets {
		assets[a.Name] = a.BrowserDownloadURL
	}
	return release.TagName, assets, nil
}

func FindAsset(tagName, kind string, assets map[string]string) (string, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "x86_64" || arch == "amd64" {
		arch = "x64"
	}

	var candidates []string
	base := fmt.Sprintf("llama-%s-bin", tagName)

	switch osName {
	case "linux":
		candidates = []string{
			base + "-ubuntu" + kind + "-" + arch,
			base + "-ubuntu-" + arch + kind,
			base + "-ubuntu-" + arch,
		}
	case "darwin":
		candidates = []string{base + "-macos-" + arch}
	case "windows":
		// Windows patterns: llama-b9630-bin-win-cpu-x64 / win-cuda-12.4-x64 / win-vulkan-x64
		noDash := strings.TrimPrefix(kind, "-")
		candidates = []string{
			base + "-win" + kind + "-" + arch,
			base + "-win-" + arch + kind,
			base + "-win-" + arch,
			base + "-win-" + noDash + "-" + arch,
			base + "-win-cpu-" + arch, // CPU build uses -cpu- in the name
		}
	}

	for _, c := range candidates {
		for name, url := range assets {
			if name == c+".tar.gz" || name == c+".zip" || strings.HasPrefix(name, c+".") {
				return url, nil
			}
			// Windows CUDA assets have version numbers: llama-b9459-bin-win-cuda-12.4-x64.zip
			if strings.HasPrefix(name, c) && strings.HasSuffix(name, arch+".zip") {
				return url, nil
			}
			if strings.HasPrefix(name, c) && strings.HasSuffix(name, arch+".tar.gz") {
				return url, nil
			}
		}
	}
	return "", fmt.Errorf("no matching asset for %s/%s (kind=%s)", osName, arch, kind)
}

func SelfUpdate(version string) error {
	rawOS := runtime.GOOS
	rawArch := runtime.GOARCH

	osMap := map[string]string{"linux": "linux", "darwin": "darwin", "windows": "windows"}
	archMap := map[string]string{"amd64": "amd64", "x86_64": "amd64", "arm64": "arm64", "aarch64": "arm64"}

	osName, ok := osMap[rawOS]
	if !ok {
		return fmt.Errorf("unsupported OS: %s", rawOS)
	}
	archName, ok := archMap[rawArch]
	if !ok {
		return fmt.Errorf("unsupported architecture: %s", rawArch)
	}

	exe := ""
	if osName == "windows" {
		exe = ".exe"
	}

	// Resolve version: if not specified, find latest stable (non-prerelease)
	tag := version
	if tag == "" {
		// List recent releases and find the latest non-prerelease
		resp, err := model.HTTPClient.Get("https://api.github.com/repos/majidkorai/gollama/releases?per_page=10")
		if err != nil {
			return fmt.Errorf("fetching releases: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("fetching releases (HTTP %d)", resp.StatusCode)
		}
		var releases []struct {
			TagName    string `json:"tag_name"`
			Prerelease bool   `json:"prerelease"`
		}
		if json.NewDecoder(resp.Body).Decode(&releases) != nil {
			return fmt.Errorf("parsing releases")
		}
		for _, r := range releases {
			if !r.Prerelease {
				tag = r.TagName
				break
			}
		}
		if tag == "" {
			return fmt.Errorf("no stable release found")
		}
	}

	url := fmt.Sprintf("https://github.com/majidkorai/gollama/releases/download/%s/gollama-%s-%s%s", tag, osName, archName, exe)

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	fmt.Printf("Downloading gollama update for %s/%s...\n", osName, archName)
	resp, err := model.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("update not found (HTTP %d)", resp.StatusCode)
	}

	tmpFile := self + ".new"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("downloading: %w", err)
	}

	if err := os.Chmod(tmpFile, 0755); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("setting permissions: %w", err)
	}

	_ = written

	backup := self + ".old"
	os.Remove(backup)
	if err := os.Rename(self, backup); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := os.Rename(tmpFile, self); err != nil {
		os.Rename(backup, self)
		os.Remove(tmpFile)
		return fmt.Errorf("installing update: %w", err)
	}

	os.Remove(backup)

	cmd := exec.Command(self, "--version")
	verOut, _ := cmd.Output()
	verStr := strings.TrimSpace(string(verOut))
	if verStr == "" {
		verStr = "latest"
	}

	fmt.Printf("Updated gollama to %s (%d bytes)\n", verStr, written)
	return nil
}

func FindLlamaServer() string {
	self := filepath.Join(model.BinDir(), "llama-server")
	if _, err := os.Stat(self); err == nil {
		return self
	}
	if runtime.GOOS == "windows" {
		selfExe := self + ".exe"
		if _, err := os.Stat(selfExe); err == nil {
			return selfExe
		}
	}
	if path, err := exec.LookPath("llama-server"); err == nil {
		return path
	}
	candidates := []string{
		"/usr/local/lib/ollama/llama-server",
		"/usr/local/bin/llama-server",
		"/home/ollama/llama.cpp/build/bin/llama-server",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "llama-server"
}

func EnsureLlamaServer() error {
	tagName, assets, err := GetReleaseData()
	if err != nil {
		return fmt.Errorf("fetching release info: %w", err)
	}

	// Check if latest version is already installed in gollama's bin dir
	installedPath := filepath.Join(model.BinDir(), "llama-server")
	if runtime.GOOS == "windows" {
		installedPath += ".exe"
	}
	if data, err := os.ReadFile(model.VersionFile()); err == nil && string(data) == tagName {
		if _, err := os.Stat(installedPath); err == nil {
			fmt.Printf("llama-server %s already installed at %s\n", tagName, installedPath)
			return nil
		}
	}

	// If found elsewhere (e.g. Homebrew), acknowledge but still download managed version
	other := FindLlamaServer()
	if other != installedPath && other != "llama-server" {
		if _, err := os.Stat(other); err == nil {
			fmt.Printf("Found llama-server at %s, but installing managed version to %s\n", other, model.BinDir())
		}
	}

	backends := DetectGPUBackends()

	fmt.Println("\nAvailable builds for llama.cpp " + tagName + ":")
	for i, b := range backends {
		mark := ""
		if b.GPU {
			mark = " 🚀 (recommended)"
		}
		if i == 0 {
			mark = " (fallback)"
		}
		fmt.Printf("  [%d] %s%s\n", i, b.Name, mark)
	}
	fmt.Printf("\nChoose (0-%d): ", len(backends)-1)

	var choice int
	fmt.Scanf("%d", &choice)
	if choice < 0 || choice >= len(backends) {
		choice = 0
	}

	selected := backends[choice]
	fmt.Printf("Selected: %s\n", selected.Name)

	if selected.Suffix == "-cuda" && runtime.GOOS == "linux" {
		fmt.Println("\nCUDA pre-built binaries are not available for Linux on the release page.")
		if _, err := exec.LookPath("nvcc"); err == nil {
			fmt.Println("CUDA toolkit detected. Building from source...")
			if err := buildLlamaServerCUDA(); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
			os.WriteFile(model.VersionFile(), []byte(tagName), 0644)
			os.WriteFile(model.BackendFile(), []byte(selected.Name), 0644)
			fmt.Printf("\nllama-server %s (%s) built and installed to %s\n", tagName, selected.Name, installedPath)
			return nil
		}
		fmt.Println("CUDA toolkit not found (nvcc missing).")
		fmt.Println("Falling back to Vulkan build which also supports NVIDIA GPUs.")
		selected = backends[len(backends)-1]
	}

	url, err := FindAsset(tagName, selected.Suffix, assets)
	if err != nil {
		return fmt.Errorf("build not found: %w", err)
	}

	model.EnsureDir(model.BinDir())
	ext := ".tar.gz"
	if strings.HasSuffix(url, ".zip") {
		ext = ".zip"
	}
	tmpFile := filepath.Join(model.BinDir(), "llama-server"+ext)
	defer os.Remove(tmpFile)

	fmt.Printf("Downloading ...\n")
	dlResp, err := model.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer dlResp.Body.Close()

	pr := &model.ProgressReader{
		Reader: dlResp.Body,
		Total:  dlResp.ContentLength,
		Name:   "▸",
		Start:  time.Now(),
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, pr)
	out.Close()
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}

	extractDir := model.BinDir()
	var found bool
	if strings.HasSuffix(tmpFile, ".zip") {
		found, err = extractZip(tmpFile, extractDir)
	} else {
		found, err = extractTarGz(tmpFile, extractDir)
	}
	if err != nil {
		return fmt.Errorf("extracting: %w", err)
	}
	if !found {
		return fmt.Errorf("llama-server not found in downloaded archive")
	}
	os.WriteFile(model.VersionFile(), []byte(tagName), 0644)
	os.WriteFile(model.BackendFile(), []byte(selected.Name), 0644)

		if runtime.GOOS == "linux" {
		checkDependencies(installedPath)
	} else if runtime.GOOS == "windows" {
		checkWindowsDependencies()
	}

	log.Printf("llama-server installed: version=%s backend=%s path=%s", tagName, selected.Name, installedPath)
	fmt.Printf("\nllama-server %s (%s) installed to %s\n", tagName, selected.Name, installedPath)
	return nil
}

func checkDependencies(binary string) {
	cmd := exec.Command("ldd", binary)
	out, err := cmd.Output()
	if err != nil {
		return
	}

	var missing []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "not found") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				missing = append(missing, parts[0])
			}
		}
	}

	if len(missing) == 0 {
		return
	}

	libMap := map[string]string{
		"libgomp.so.1":    "libgomp1",
		"libatomic.so.1":  "libatomic1",
		"libstdc++.so.6":  "libstdc++6",
		"libm.so.6":       "libc6",
		"libc.so.6":       "libc6",
		"libpthread.so.0": "libc6",
		"librt.so.1":      "libc6",
		"libdl.so.2":      "libc6",
	}

	pkgs := make(map[string]bool)
	for _, lib := range missing {
		if pkg, ok := libMap[lib]; ok {
			pkgs[pkg] = true
		}
	}

	if len(pkgs) == 0 {
		fmt.Printf("Warning: missing shared libraries: %s\n", strings.Join(missing, ", "))
		return
	}

	pkgList := make([]string, 0, len(pkgs))
	for p := range pkgs {
		pkgList = append(pkgList, p)
	}

	if _, err := exec.LookPath("apt-get"); err != nil {
		fmt.Printf("Warning: missing packages: %s (install manually: apt install %s)\n",
			strings.Join(missing, ", "), strings.Join(pkgList, " "))
		return
	}

	fmt.Printf("Installing missing dependencies: %s ...\n", strings.Join(pkgList, " "))
	cmd = exec.Command("apt-get", append([]string{"install", "-y", "-qq"}, pkgList...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to install dependencies: %v\n", err)
	}
}

func checkWindowsDependencies() {
	if _, err := os.Stat(`C:\Windows\System32\VCRUNTIME140.dll`); err == nil {
		return
	}

	// Also check SysWOW64 for 32-bit on 64-bit systems
	if _, err := os.Stat(`C:\Windows\SysWOW64\VCRUNTIME140.dll`); err == nil {
		return
	}

	fmt.Println("Visual C++ Redistributable not found. Installing...")
	cmd := exec.Command("winget", "install", "Microsoft.VCRedist.2015+.x64", "--accept-source-agreements", "--silent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Fallback: download and run installer directly
		fmt.Println("winget failed, downloading installer directly...")
		url := "https://aka.ms/vs/17/release/vc_redist.x64.exe"
		tmpFile := filepath.Join(os.TempDir(), "vc_redist.x64.exe")

		out, err := os.Create(tmpFile)
		if err != nil {
			fmt.Printf("Warning: could not download VC++ redist: %v\n", err)
			return
		}

		resp, err := model.HTTPClient.Get(url)
		if err != nil {
			out.Close()
			fmt.Printf("Warning: could not download VC++ redist: %v\n", err)
			return
		}
		defer resp.Body.Close()

		io.Copy(out, resp.Body)
		out.Close()

		install := exec.Command(tmpFile, "/install", "/quiet", "/norestart")
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			fmt.Printf("Warning: VC++ redist installation failed: %v\n", err)
			fmt.Println("Install manually from: https://aka.ms/vcredist")
		}
		os.Remove(tmpFile)
	}
}

func extractTarGz(path, dest string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return false, err
	}
	defer gzr.Close()

	found := false
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}
		name := filepath.Base(header.Name)
		if name == "" || name == "." || name == ".." {
			continue
		}
		outFile := filepath.Join(dest, name)
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(outFile, 0755)
		case tar.TypeSymlink:
			os.Symlink(header.Linkname, outFile)
		default:
			outF, err := os.Create(outFile)
			if err != nil {
				return false, fmt.Errorf("creating %s: %w", name, err)
			}
			if _, err := io.Copy(outF, tr); err != nil {
				outF.Close()
				return false, err
			}
			outF.Close()
			os.Chmod(outFile, os.FileMode(header.Mode))
		}
		if name == "llama-server" || name == "llama-server.exe" {
			found = true
		}
	}
	return found, nil
}

func extractZip(path, dest string) (bool, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false, err
	}
	defer r.Close()

	found := false
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == "" || name == "." || name == ".." {
			continue
		}
		outFile := filepath.Join(dest, name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(outFile, 0755)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return false, fmt.Errorf("opening %s: %w", name, err)
		}

		outF, err := os.Create(outFile)
		if err != nil {
			rc.Close()
			return false, fmt.Errorf("creating %s: %w", name, err)
		}

		_, err = io.Copy(outF, rc)
		rc.Close()
		outF.Close()
		if err != nil {
			return false, fmt.Errorf("writing %s: %w", name, err)
		}

		if name == "llama-server" || name == "llama-server.exe" {
			found = true
		}
	}
	return found, nil
}

func buildLlamaServerCUDA() error {
	for _, tool := range []string{"git", "cmake", "make", "nvcc"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("required tool %q not found in PATH", tool)
		}
	}

	buildDir, err := os.MkdirTemp("", "gollama-build-*")
	if err != nil {
		return fmt.Errorf("creating build dir: %w", err)
	}
	defer os.RemoveAll(buildDir)

	fmt.Println("  Cloning llama.cpp...")
	clone := exec.Command("git", "clone", "--depth", "1",
		"https://github.com/ggml-org/llama.cpp", filepath.Join(buildDir, "llama.cpp"))
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		return fmt.Errorf("cloning llama.cpp: %w", err)
	}

	srcDir := filepath.Join(buildDir, "llama.cpp")
	fmt.Println("  Configuring with cmake (CUDA)...")
	cmake := exec.Command("cmake", "-B", "build",
		"-DGGML_CUDA=ON",
		"-DCMAKE_CUDA_ARCHITECTURES=all")
	cmake.Dir = srcDir
	cmake.Stdout = os.Stdout
	cmake.Stderr = os.Stderr
	if err := cmake.Run(); err != nil {
		return fmt.Errorf("cmake configure failed: %w", err)
	}

	fmt.Println("  Building llama-server (this may take a while)...")
	build := exec.Command("cmake", "--build", "build", "-j", "--target", "llama-server")
	build.Dir = srcDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("cmake build failed: %w", err)
	}

	src := filepath.Join(srcDir, "build", "bin", "llama-server")
	dest := filepath.Join(model.BinDir(), "llama-server")
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading built binary: %w", err)
	}
	if err := os.WriteFile(dest, input, 0755); err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}

	// Also copy any .so files needed by the binary
	libDir := filepath.Join(srcDir, "build", "bin")
	entries, _ := os.ReadDir(libDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".so") {
			data, err := os.ReadFile(filepath.Join(libDir, e.Name()))
			if err != nil {
				continue
			}
			os.WriteFile(filepath.Join(model.BinDir(), e.Name()), data, 0755)
		}
	}

	fmt.Println("  Build complete.")
	return nil
}

func tmpFileReader(path string) io.ReadCloser {
	r, _ := os.Open(path)
	return r
}
