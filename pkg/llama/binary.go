package llama

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
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

// releaseAPIBase is the GitHub releases LIST endpoint; a package var seam so
// tests can point it at a fake server (same pattern as checksumURLBase). The
// GOLLAMA_RELEASE_API_BASE env var overrides it — this is what cross-package
// tests (pkg/server) use, since they can't touch the private var.
//
// Note: it is the list endpoint (not /releases/latest) on purpose — llama.cpp
// marks its build releases (bXXX) as *prereleases*, so /releases/latest returns
// the stale non-prerelease tag (v0.2.0) instead of the current build. We fetch
// a page of releases and pick the highest build number.
var releaseAPIBase = "https://api.github.com/repos/ggml-org/llama.cpp/releases"

// effectiveReleaseAPIBase resolves the base URL: env override first, then the
// package var. Reading it per call (rather than once) is what lets tests
// change the target and invalidate the release-info cache.
func effectiveReleaseAPIBase() string {
	if u := os.Getenv("GOLLAMA_RELEASE_API_BASE"); u != "" {
		return u
	}
	return releaseAPIBase
}

// fetchLatestRelease performs one GitHub releases-list call and returns the
// release with the highest llama.cpp build number (bXXX): its tag, its
// html_url (release notes page), and the asset map.
//
// llama.cpp marks build releases as *prereleases*, so /releases/latest returns
// the stale non-prerelease tag (v0.2.0), not the current build. We fetch a page
// of releases and pick the highest build number — robust against the stale tag
// and against any out-of-order tags.
func fetchLatestRelease() (tag, htmlURL string, assets map[string]string, err error) {
	url := effectiveReleaseAPIBase()
	if !strings.Contains(url, "?") {
		url += "?per_page=15"
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gollama")

	resp, err := model.APIClient.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", "", nil, fmt.Errorf("parsing releases: %w", err)
	}

	bestN := int64(-1)
	best := -1
	for i := range releases {
		if n := extractBuildNumber(releases[i].TagName); n > bestN {
			bestN, best = n, i
		}
	}
	if best == -1 {
		return "", "", nil, fmt.Errorf("no build-number release found in list")
	}

	r := releases[best]
	assets = make(map[string]string)
	for _, a := range r.Assets {
		assets[a.Name] = a.BrowserDownloadURL
	}
	return r.TagName, r.HTMLURL, assets, nil
}

func GetReleaseData() (string, map[string]string, error) {
	tag, _, assets, err := fetchLatestRelease()
	return tag, assets, err
}

// LatestReleaseInfo returns the latest upstream llama.cpp release tag and its
// release-notes URL, memoized behind a mutex with a 1h TTL. The UI calls this
// on every Settings open, and unauthenticated GitHub is rate-limited to
// 60 req/h/IP, so a fresh fetch happens only on cache miss. A cached error is
// returned for a short backoff so a GitHub outage doesn't hammer the API on
// every call.
var (
	releaseInfoMu    sync.Mutex
	releaseInfoBase  string // base URL the cached result was fetched from
	releaseInfoTag   string
	releaseInfoURL   string
	releaseInfoAt    time.Time
	releaseInfoErr   error
	releaseInfoErrAt time.Time
)

const (
	releaseInfoTTL    = time.Hour
	releaseInfoErrTTL = 5 * time.Minute
)

func LatestReleaseInfo() (tag, releaseURL string, err error) {
	base := effectiveReleaseAPIBase()
	releaseInfoMu.Lock()
	defer releaseInfoMu.Unlock()
	now := time.Now()
	// The cache is only valid for the same base URL; a change (e.g. a test
	// swapping in a fake server) is a cache miss.
	if base == releaseInfoBase && releaseInfoAt.Add(releaseInfoTTL).After(now) {
		return releaseInfoTag, releaseInfoURL, nil
	}
	if base == releaseInfoBase && releaseInfoErr != nil && releaseInfoErrAt.Add(releaseInfoErrTTL).After(now) {
		return "", "", releaseInfoErr
	}
	tag, url, _, err := fetchLatestRelease()
	if err != nil {
		releaseInfoBase, releaseInfoErr, releaseInfoErrAt = base, err, now
		return "", "", err
	}
	releaseInfoBase, releaseInfoTag, releaseInfoURL, releaseInfoAt, releaseInfoErr = base, tag, url, now, nil
	return tag, url, nil
}

// InstalledLlamaServerVersion returns the installed llama-server version: the
// recorded version file first, then a `llama-server --version` exec fallback
// (custom builds like the VM's hand-built binary may lack the version file).
func InstalledLlamaServerVersion() string {
	if v := model.LlamaServerVersion(); v != "" {
		return v
	}
	bin := FindLlamaServer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// buildNumberRe matches a llama.cpp "b1234" build tag.
var buildNumberRe = regexp.MustCompile(`b(\d{3,})`)

// digitsRe matches the first run of >=3 digits (e.g. in --version output).
var digitsRe = regexp.MustCompile(`\d{3,}`)

// extractBuildNumber pulls the build number out of a version-ish string: a
// "b1234" tag, or the first run of >=3 digits. Returns -1 when unparseable.
func extractBuildNumber(s string) int64 {
	if m := buildNumberRe.FindStringSubmatch(s); m != nil {
		if n, err := strconv.ParseInt(m[1], 10, 64); err == nil {
			return n
		}
	}
	if m := digitsRe.FindString(s); m != "" {
		if n, err := strconv.ParseInt(m, 10, 64); err == nil {
			return n
		}
	}
	return -1
}

// CompareBuildNumbers reports how many builds `installed` lags `latest` by, and
// whether the two are comparable. comparable is false when either side has no
// parseable build number (custom/unknown builds) — callers should then show no
// badge rather than a misleading "outdated".
func CompareBuildNumbers(installed, latest string) (behind int, comparable bool) {
	inst := extractBuildNumber(installed)
	lat := extractBuildNumber(latest)
	if inst < 0 || lat < 0 {
		return 0, false
	}
	b := int(lat - inst)
	if b < 0 {
		b = 0
	}
	return b, true
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
		resp, err := model.APIClient.Get("https://api.github.com/repos/majidkorai/gollama/releases?per_page=10")
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
			if !r.Prerelease && !strings.Contains(r.TagName, "-rc") {
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

	// Verify the download against the release's checksums.txt before it
	// replaces the running binary.
	assetName := fmt.Sprintf("gollama-%s-%s%s", osName, archName, exe)
	if err := verifyChecksum(tag, assetName, tmpFile); err != nil {
		os.Remove(tmpFile)
		return err
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
	fmt.Println("Restart gollama to apply the update (gollama restart)")
	return nil
}

// checksumURLBase is the release download base for checksums.txt; a
// variable so tests can point it at a local server.
var checksumURLBase = "https://github.com/majidkorai/gollama/releases/download"

// verifyChecksum fetches checksums.txt from the release (shipped by CI)
// and verifies the sha256 of the downloaded binary. A missing or
// unlisted checksum (older releases) is a warning; a mismatch is fatal —
// the update is aborted before the binary is replaced.
func verifyChecksum(tag, assetName, path string) error {
	url := fmt.Sprintf("%s/%s/checksums.txt", checksumURLBase, tag)
	resp, err := model.APIClient.Get(url)
	if err != nil {
		fmt.Printf("Warning: could not fetch checksums.txt for %s: %v (skipping verification)\n", tag, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Warning: checksums.txt not found for %s (HTTP %d) — skipping verification\n", tag, resp.StatusCode)
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}
	expected := expectedChecksum(string(data), assetName)
	if expected == "" {
		fmt.Printf("Warning: %s not listed in checksums.txt — skipping verification\n", assetName)
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening downloaded binary: %w", err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		f.Close()
		return fmt.Errorf("hashing downloaded binary: %w", err)
	}
	f.Close()
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s (expected %s, got %s) — aborting update", assetName, expected, actual)
	}
	fmt.Printf("Checksum verified: %s\n", actual)
	return nil
}

// expectedChecksum parses a sha256sum-style checksums.txt (as shipped by CI)
// and returns the lowercase hash recorded for assetName, or "" if the asset
// is not listed.
func expectedChecksum(checksumsContent, assetName string) string {
	for _, line := range strings.Split(checksumsContent, "\n") {
		fields := strings.Fields(line)
		// sha256sum output: "<hash>  <name>" (two spaces) or "<hash> *<name>".
		if len(fields) >= 2 {
			name := strings.TrimPrefix(fields[1], "*")
			if name == assetName {
				return strings.ToLower(fields[0])
			}
		}
	}
	return ""
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
	installedPath := filepath.Join(model.BinDir(), "llama-server")
	if runtime.GOOS == "windows" {
		installedPath += ".exe"
	}

	tagName, assets, err := GetReleaseData()
	if err != nil {
		// If release info fails but a binary already exists, proceed silently
		if _, statErr := os.Stat(installedPath); statErr == nil {
			return nil
		}
		if other := FindLlamaServer(); other != "llama-server" {
			if _, statErr := os.Stat(other); statErr == nil {
				return nil
			}
		}
		return fmt.Errorf("fetching release info: %w", err)
	}

	// If binary already exists, use it regardless of version tag
	if _, err := os.Stat(installedPath); err == nil {
		if data, _ := os.ReadFile(model.VersionFile()); len(data) > 0 {
			fmt.Printf("llama-server %s already installed at %s\n", strings.TrimSpace(string(data)), installedPath)
		} else {
			fmt.Printf("llama-server already installed at %s\n", installedPath)
		}
		return nil
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
		fmt.Println("\nllama.cpp does not ship pre-built CUDA binaries for Linux.")
		fmt.Println("To build from source, install build tools and the CUDA toolkit:")
		fmt.Println("  apt install git cmake build-essential nvidia-cuda-toolkit")
		fmt.Println("Then run  gollama update  again to rebuild with CUDA support.")
		if _, err := exec.LookPath("nvcc"); err == nil {
			fmt.Println("\nCUDA toolkit detected — building from source.")
			fmt.Println("This takes 30-60 minutes on most systems (compiling ~200 CUDA kernels).")
			fmt.Println("Press Ctrl+C to abort and pick Vulkan instead (5-second install, same perf).")
			if err := buildLlamaServerCUDA(); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
			os.WriteFile(model.VersionFile(), []byte(tagName), 0644)
			os.WriteFile(model.BackendFile(), []byte(selected.Name), 0644)
			fmt.Printf("\nllama-server %s (%s) built and installed to %s\n", tagName, selected.Name, installedPath)
			return nil
		}
		fmt.Println("\nFalling back to Vulkan, which also supports NVIDIA GPUs with good performance.")
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

	slog.Info("llama-server installed", "version", tagName, "backend", selected.Name, "path", installedPath)
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

	// No silent package installs: explicit opt-in (GOLLAMA_AUTO_INSTALL_DEPS=1)
	// or an interactive yes. Non-interactive sessions without the env var
	// only get the command to run.
	autoInstall := os.Getenv("GOLLAMA_AUTO_INSTALL_DEPS") == "1"
	if !autoInstall {
		fmt.Printf("Install missing packages: %s? [y/N] ", strings.Join(pkgList, " "))
		if !isTerminal(os.Stdin) {
			fmt.Println("(non-interactive — not installing)")
			fmt.Printf("Install manually: sudo apt-get install %s\n", strings.Join(pkgList, " "))
			return
		}
		var ans string
		fmt.Scanln(&ans)
		if a := strings.ToLower(strings.TrimSpace(ans)); a != "y" && a != "yes" {
			fmt.Printf("Install manually: sudo apt-get install %s\n", strings.Join(pkgList, " "))
			return
		}
	}

	fmt.Printf("Installing missing dependencies: %s ...\n", strings.Join(pkgList, " "))
	cmd = exec.Command("apt-get", append([]string{"install", "-y", "-qq"}, pkgList...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to install dependencies: %v\n", err)
	}
}

// isTerminal reports whether f is connected to a TTY (stdlib-only check).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
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

func detectCUDAArch() string {
	capCmd := exec.Command("nvidia-smi", "--query-gpu=compute_cap", "--format=csv,noheader")
	capOut, err := capCmd.Output()
	if err == nil {
		archs := []string{}
		for _, line := range strings.Split(strings.TrimSpace(string(capOut)), "\n") {
			arch := strings.ReplaceAll(strings.TrimSpace(line), ".", "")
			if arch != "" {
				archs = append(archs, arch)
			}
		}
		if len(archs) > 0 {
			return strings.Join(archs, ";")
		}
	}
	return "all"
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
	// Only build for the detected GPU arch instead of all — cuts compile time 5-10x
	cudaArch := detectCUDAArch()
	fmt.Printf("  Targeting CUDA arch %s\n", cudaArch)
	cmake := exec.Command("cmake", "-B", "build",
		"-DGGML_CUDA=ON",
		"-DCMAKE_CUDA_ARCHITECTURES="+cudaArch)
	cmake.Dir = srcDir
	cmake.Stdout = os.Stdout
	cmake.Stderr = os.Stderr
	if err := cmake.Run(); err != nil {
		return fmt.Errorf("cmake configure failed: %w", err)
	}

	fmt.Println("  Building llama-server (this may take a while)...")
	// Limit parallel jobs on low-memory systems — CUDA compilation is RAM-hungry
	jobLimit := runtime.NumCPU()
	if jobLimit > 4 {
		jobLimit = 4
	}
	build := exec.Command("cmake", "--build", "build", "-j", strconv.Itoa(jobLimit), "--target", "llama-server")
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

	// Copy any .so files the binary needs (may be in bin/, lib/, or src/)
	soSeen := map[string]bool{}
	for _, libDir := range []string{
		filepath.Join(srcDir, "build", "bin"),
		filepath.Join(srcDir, "build", "lib"),
		filepath.Join(srcDir, "build", "src"),
		filepath.Join(srcDir, "build"),
	} {
		entries, err := os.ReadDir(libDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), "lib") || !strings.Contains(e.Name(), ".so") {
				continue
			}
			src := filepath.Join(libDir, e.Name())
			dst := filepath.Join(model.BinDir(), e.Name())
			if soSeen[dst] {
				continue
			}
			soSeen[dst] = true
			fi, err := os.Stat(src)
			if err != nil {
				continue
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				link, _ := os.Readlink(src)
				os.Symlink(link, dst)
				continue
			}
			data, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			os.WriteFile(dst, data, 0755)
		}
	}

	// llama.cpp b9720+ builds per-arch CPU libs (libggml-cpu-*.so)
	// but not a generic libggml-cpu.so.0, which libggml.so dlopens at runtime.
	// Create a symlink so the generic soname always resolves.
	cpuVariant := filepath.Join(model.BinDir(), "libggml-cpu-x64.so")
	if _, err := os.Stat(cpuVariant); err == nil {
		os.Symlink("libggml-cpu-x64.so", filepath.Join(model.BinDir(), "libggml-cpu.so.0"))
	}

	fmt.Println("  Build complete.")
	return nil
}
