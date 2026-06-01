package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
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
var version = "0.1.0-dev"

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
			fmt.Println("  e.g. gollama pull hf.co/unsloth/gemma-4-E2B-it-GGUF:Q4_K_M")
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

		go func() {
			for {
				time.Sleep(30 * time.Second)
				for _, inst := range mgr.List() {
					proc, err := os.FindProcess(inst.PID)
					if err != nil {
						mgr.Stop(inst.Port)
						continue
					}
					if err := proc.Signal(syscall.Signal(0)); err != nil {
						mgr.Stop(inst.Port)
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

		srv := server.New(mgr, port)
		go func() {
			if err := srv.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
				os.Exit(1)
			}
		}()

		<-sigCh
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

		inst, err := mgr.Start(modelName, 0, extraArgs)
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
			inst, err := mgr.Start(modelName, 8081, extraArgs)
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
			<-sigCh
			fmt.Println("\nstopping...")
			if err := mgr.Stop(inst.Port); err != nil {
				fmt.Fprintf(os.Stderr, "Error stopping: %v\n", err)
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
		{"gemma-4-E2B-it-GGUF (2B, ~1.6 GB) — fast, great for chat", "hf.co/unsloth/gemma-4-E2B-it-GGUF:Q4_K_M"},
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
	fmt.Println("  Start chatting:     gollama chat <model>")
	fmt.Println("  Open web UI:        gollama serve")
	fmt.Println("  See all commands:   gollama help")
	fmt.Println()
	fmt.Println("Quickstart (chat with your model right now):")
	if choice < len(models) && models[choice].Ref != "" {
		pulled := models[choice].Ref
		fmt.Printf("\n  gollama chat %s\n", pulled)
		fmt.Println()
	}
}

func localIP() string {
	hosts := []string{"hostname", "hostname -I"}
	for _, cmd := range hosts {
		c := exec.Command("sh", "-c", cmd)
		out, err := c.Output()
		if err != nil {
			continue
		}
		ip := strings.Fields(string(out))
		if len(ip) > 0 && net.ParseIP(ip[0]) != nil {
			return ip[0]
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
  gollama list                   List available models
  gollama serve [port]           Start manager with web UI (default :9080)
  gollama chat <model> [flags]   Start a terminal chat session
  gollama run <model> [flags]    Quick-start a model (Ctrl+C to stop)
  gollama ps                     List running instances
  gollama stop <port>            Stop an instance

Examples:
  gollama self-update           # Update gollama binary
  gollama update                # Update llama-server
  gollama pull hf.co/unsloth/gemma-4-E2B-it-GGUF:Q4_K_M
  gollama chat gemma-4-E2B-it-Q4_K_M.gguf
  gollama serve

Tip:
  Models are stored in ~/.gollama/models/
  llama-server binary in ~/.gollama/bin/ (auto-downloaded via 'gollama update')
  For CUDA on Linux: build from source and copy the binary to ~/.gollama/bin/`)
}
