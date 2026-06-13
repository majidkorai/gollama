package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/majidkorai/gollama/pkg/chat"
	"github.com/majidkorai/gollama/pkg/llama"
	"github.com/majidkorai/gollama/pkg/manager"
	"github.com/majidkorai/gollama/pkg/model"
	"github.com/majidkorai/gollama/pkg/server"
)

// version is set at build time via -ldflags=-X main.version=v0.x.x
// local builds fall back to this default
var version = "0.2.7"

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--version" || os.Args[1] == "-v" {
		if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
			fmt.Printf("gollama %s\n", version)
			return
		}
		if isFreshInstall() {
			runWizard()
			return
		}
		printUsage()
		os.Exit(1)
	}

	if os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		return
	}

	mgr := manager.NewManager()

	switch os.Args[1] {
	case "update":
		if err := llama.EnsureLlamaServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "self-update":
		if err := llama.SelfUpdate(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "pull":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gollama pull hf.co/user/repo:quant")
			fmt.Println("  e.g. gollama pull hf.co/unsloth/Qwen3.5-0.8B-GGUF:Q4_K_M")
			os.Exit(1)
		}
		modelRef := os.Args[2]
		if err := model.PullModel(modelRef); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list":
		models, err := model.ListModels()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(models) == 0 {
			fmt.Println("No models found. Use 'gollama pull <model>' to download one.")
			return
		}
		fmt.Printf("%-40s %-10s %s\n", "Name", "Size", "Source")
		for _, m := range models {
			fmt.Printf("%-40s %-10s %s\n", m.Name, model.FormatSize(m.Size), m.Source)
		}

	case "serve":
		if err := llama.EnsureLlamaServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		ctx, stop := context.WithCancel(context.Background())
		defer stop()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(30 * time.Second):
				}
				for _, inst := range mgr.List() {
					proc, err := os.FindProcess(inst.PID)
					if err != nil {
						mgr.Stop(inst.Port)
						continue
					}
					// Signal 0 is not supported on Windows — always fails
					if runtime.GOOS != "windows" {
						if err := proc.Signal(syscall.Signal(0)); err != nil {
							mgr.Stop(inst.Port)
						}
					}
				}
				mgr.RecoverOrphans()
			}
		}()

		port := "9080"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		portNum, _ := strconv.Atoi(port)
		if portNum > 0 && !func() bool {
			ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portNum))
			if err != nil {
				return false
			}
			ln.Close()
			return true
		}() {
			fmt.Fprintf(os.Stderr, "Warning: port %s is busy, try a different port\n", port)
		}
		fmt.Printf("Web UI: http://%s:%s\n", localIP(), port)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		srv := server.NewWithVersion(mgr, port, version)
		go func() {
			if err := srv.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
				os.Exit(1)
			}
		}()

		<-sigCh
		stop()
		fmt.Println("\nshutting down...")

	case "chat":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gollama chat <model> [flags...]")
			os.Exit(1)
		}
		if err := llama.EnsureLlamaServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
		modelName := os.Args[2]
		extraArgs := os.Args[3:]

		inst, err := mgr.Start(modelName, 0, extraArgs, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Waiting for %s on port %d...\n", inst.Model, inst.Port)
		baseURL := fmt.Sprintf("http://127.0.0.1:%d", inst.Port)
		if err := chat.WaitForReady(baseURL, 30*time.Second); err != nil {
			mgr.Stop(inst.Port)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nStopping...")
			mgr.Stop(inst.Port)
			os.Exit(0)
		}()

		c := chat.New(inst.Port)
		if err := c.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
		}
		mgr.Stop(inst.Port)

	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gollama run <model> [flags...]")
			os.Exit(1)
		}
		if err := llama.EnsureLlamaServer(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
		modelName := os.Args[2]
		extraArgs := os.Args[3:]

		if len(mgr.List()) == 0 {
			inst, err := mgr.Start(modelName, 8081, extraArgs, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Started %s on port %d (PID %d)\n", inst.Model, inst.Port, inst.PID)
			ip := localIP()
			fmt.Printf("Chat: http://%s:%d\n", ip, inst.Port)
			fmt.Printf("Web UI: run 'gollama serve' for http://%s:9080\n", ip)
			fmt.Println("Press Ctrl+C to stop")

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			done := make(chan struct{})
			go func() {
				for {
					time.Sleep(2 * time.Second)
					for _, i := range mgr.List() {
						if i.Port == inst.Port && i.Status != "running" {
							close(done)
							return
						}
					}
				}
			}()
			select {
			case <-sigCh:
				fmt.Println("\nstopping...")
				mgr.Stop(inst.Port)
			case <-done:
				fmt.Println("\ninstance stopped unexpectedly")
				logFile := filepath.Join(model.GollamaDir(), "logs", fmt.Sprintf("port-%d.log", inst.Port))
				if data, err := os.ReadFile(logFile); err == nil {
					lines := strings.Split(string(data), "\n")
					for i := len(lines) - 1; i >= 0 && i > len(lines)-10; i-- {
						line := strings.TrimSpace(lines[i])
						if line != "" && !strings.Contains(line, "\r") {
							fmt.Printf("Error: %s\n", line)
							break
						}
					}
				}
			}
		}

	case "ps":
		instances := mgr.List()
		if len(instances) == 0 {
			fmt.Println("No running instances")
			return
		}
		fmt.Printf("%-5s %-20s %-5s %-8s\n", "Port", "Model", "PID", "Status")
		for _, inst := range instances {
			fmt.Printf("%-5d %-20s %-5d %-8s\n", inst.Port, truncate(inst.Model, 20), inst.PID, inst.Status)
		}

	case "stop":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gollama stop <port>")
			os.Exit(1)
		}
		port, _ := strconv.Atoi(os.Args[2])
		if err := mgr.Stop(port); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Stopped instance on port %d\n", port)

	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gollama delete <model-name>")
			os.Exit(1)
		}
		modelName := os.Args[2]
		var blobPath string
		if err := model.UpdateIndex(func(idx map[string]model.ModelInfo) error {
			info, ok := idx[modelName]
			if !ok {
				return fmt.Errorf("model %q not found", modelName)
			}
			blobPath = info.BlobPath
			delete(idx, modelName)
			return nil
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := os.Remove(blobPath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error deleting file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted model: %s\n", modelName)

	case "logs":
		if len(os.Args) < 3 {
			fmt.Println("Usage: gollama logs <port>")
			os.Exit(1)
		}
		port := os.Args[2]
		logFile := filepath.Join(model.GollamaDir(), "logs", fmt.Sprintf("port-%s.log", port))
		data, err := os.ReadFile(logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading logs for port %s: %v\n", port, err)
			os.Exit(1)
		}
		fmt.Print(string(data))

	default:
		printUsage()
		os.Exit(1)
	}
}

func isFreshInstall() bool {
	if _, err := os.Stat(model.BinDir()); os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(llama.FindLlamaServer()); os.IsNotExist(err) {
		return true
	}
	return false
}

func runWizard() {
	fmt.Print(`
  ╔══════════════════════════════════╗
  ║       gollama — llama.cpp 🦙     ║
  ║  one command to run them all     ║
  ╚══════════════════════════════════╝

`)

	fmt.Print("Let's get you set up in a few steps.\n\n")

	// Step 1: install llama-server
	fmt.Println("Step 1: Install llama-server")
	if err := llama.EnsureLlamaServer(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Println("You can retry later with: gollama update")
		return
	}

	// Step 2: ask about pulling a model
	fmt.Println("\nStep 2: Pull a model")
	fmt.Println("gollama needs a GGUF model to run.")
	models := []struct {
		Label string
		Ref   string
	}{
		{"Qwen3.5-0.8B-GGUF (~664 MB) — runs on almost any hardware", "hf.co/unsloth/Qwen3.5-0.8B-GGUF:Q4_K_M"},
		{"Qwen2.5-7B-Instruct-GGUF (~4.7 GB) — good balance of speed & quality", "hf.co/Qwen/Qwen2.5-7B-Instruct-GGUF:Q4_K_M"},
		{"Llama-3.1-8B-Instruct-GGUF (~5.5 GB) — popular general purpose", "hf.co/lmstudio-community/Meta-Llama-3.1-8B-Instruct-GGUF:Q4_K_M"},
		{"Qwen2.5-14B-Instruct-GGUF (~9 GB) — higher quality, needs more VRAM", "hf.co/Qwen/Qwen2.5-14B-Instruct-GGUF:Q4_K_M"},
		{"Skip — I'll add my own models later", ""},
	}

	fmt.Println("Pick a starter model to download:")
	for i, m := range models {
		fmt.Printf("  [%d] %s\n", i+1, m.Label)
	}
	fmt.Printf("\nChoose (1-%d): ", len(models))

	var choice int
	fmt.Scanf("%d", &choice)
	if choice < 1 || choice > len(models) {
		choice = len(models)
	}
	choice--

	if models[choice].Ref != "" {
		if err := model.PullModel(models[choice].Ref); err != nil {
			fmt.Fprintf(os.Stderr, "Error pulling model: %v\n", err)
			fmt.Println("You can retry later with: gollama pull <model>")
		}
	}

	fmt.Println("\nStep 3: You're ready!")
	fmt.Println()
	fmt.Println("  Web UI:             gollama serve")
	fmt.Println("  Terminal chat:      gollama chat <model>")
	fmt.Println("  See all commands:   gollama help")
	fmt.Println()
	fmt.Println("Quickstart:")
	if choice < len(models) && models[choice].Ref != "" {
		pulled := models[choice].Ref
		fmt.Printf("\n  gollama serve\n")
		fmt.Printf("  gollama chat %s\n", pulled)
		fmt.Println()
	}
}

func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return "localhost"
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func printUsage() {
	fmt.Println(`gollama — llama.cpp instance manager

Usage:
  gollama update                 Download/update llama-server binary
  gollama self-update            Update gollama itself
  gollama pull <model>           Download model from HuggingFace
  gollama serve [port]           Web UI + REST API on :9080 (main workflow)
  gollama chat <model> [flags]   Start a terminal chat session
  gollama list                   List available models
  gollama delete <model>         Delete a downloaded model
  gollama logs <port>            Show instance logs
  gollama ps                     List running instances
  gollama stop <port>            Stop an instance
  gollama run <model> [flags]    Run a model directly (debug/advanced)

Examples:
  gollama serve                 # Web UI on http://<ip>:9080
  gollama self-update           # Update gollama binary
  gollama update                # Update llama-server
  gollama pull hf.co/unsloth/Qwen3.5-0.8B-GGUF:Q4_K_M
  gollama delete my-model       # Remove a model
  gollama logs 8081             # View logs for instance on port 8081

Tip:
  Models are stored in ~/.gollama/models/
  llama-server binary in ~/.gollama/bin/ (auto-downloaded via 'gollama update')
  For CUDA on Linux: build from source and copy the binary to ~/.gollama/bin/`)
}
