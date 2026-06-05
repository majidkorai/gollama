# gollama 🦙

**Spin up GGUF models in seconds** — a single Go binary that downloads, manages, and runs llama.cpp instances with a terminal chat, web UI, REST API, and full flag control.

Pull any model from HuggingFace, launch it on any port, chat with it in your terminal or browser — all from one command. No dependencies, no Docker, no Python.

## Install

**Linux & macOS** (one line, no Go required):
```bash
curl -fsSL https://raw.githubusercontent.com/majidkorai/gollama/main/install.sh | sh
```

**Windows** (PowerShell):
```powershell
iwr -useb https://raw.githubusercontent.com/majidkorai/gollama/main/install.ps1 | iex
```

The script detects your platform, downloads a pre-built binary (linux/darwin/windows × amd64/arm64), and installs it to `/usr/local/bin`. If no pre-built binary exists, it falls back to building from source.

**Manual build:**
```bash
git clone https://github.com/majidkorai/gollama
cd gollama
go build -o gollama .
sudo cp gollama /usr/local/bin/
```

## Quick Start

```bash
# First run — interactive wizard handles everything
gollama

# Or step by step:
gollama update                  # Install llama-server binary
gollama pull hf.co/...          # Download a model from HuggingFace
gollama serve                   # Open web UI — manage everything from the browser
```

The first-run wizard:
1. Detects your GPU and downloads the right `llama-server` build
2. Offers to pull a popular starter model (Gemma, Qwen, Llama, or skip)
3. Shows next steps to start chatting

## Commands

| Command | Description |
|---------|-------------|
| `gollama` | First-run wizard (auto-setup if fresh install) |
| `gollama update` | Download/update llama-server binary |
| `gollama self-update` | Update gollama itself to the latest version |
| `gollama pull <model>` | Download a GGUF model from HuggingFace |
| `gollama serve [port]` | Web UI + REST API on :9080 (main workflow) |
| `gollama chat <model> [flags]` | Start a terminal chat session (streaming) |
| `gollama list` | List downloaded models with metadata |
| `gollama ps` | List running instances |
| `gollama stop <port>` | Stop an instance |
| `gollama run <model> [flags]` | Run a model server directly (debug/advanced) |

**Examples:**
```bash
gollama serve                                    # Web UI on http://<ip>:9080
gollama pull hf.co/unsloth/Qwen3.5-0.8B-GGUF:Q4_K_M
gollama chat gemma-4-E2B-it-Q4_K_M.gguf          # Terminal chat
gollama run Qwopus3.6-27B-v2-Q4_K_M.gguf         # Debug/advanced
```

## Terminal Chat (`gollama chat`)

For when you don't need the full web UI — a lightweight streaming chat right in your terminal:
- **Token streaming** — see tokens as they're generated (SSE)
- **Reasoning display** — thinking tokens shown in *italic* before the response
- **Multi-line input** — `/open` to start, `/close` to end
- **Chat history** — maintained for context (last 20 messages)
- **Auto port** — picks a free port, no manual setup
- **Graceful exit** — Ctrl+C or `/exit` to stop and clean up

```
$ gollama chat gemma-4-E2B-it-Q4_K_M.gguf
Waiting for model on port 8081...
Chatting with llama-server at http://127.0.0.1:8081
Type /exit or /quit to end the conversation.

>>> What is the capital of France?
The capital of France is Paris.
```

## Web UI

Open **http://<your-ip>:9080** in your browser.

- **Dashboard** — metrics overview (models, instances, tokens/sec), quick launch, pull model
- **Models** — list all downloaded models with metadata (arch, quant, context length)
- **Chat** — full chat workspace with any running instance (proxied, no CORS)
- **Left sidebar** — navigation between views
- **Instance cards** — port, PID, status, tokens/sec, actions (stop, chat, open, logs)
- **Theme toggle** — dark/light mode

## Updating

gollama has two separate update commands:

- **`gollama self-update`** — updates the `gollama` binary itself. Downloads the latest release from GitHub and replaces the current binary.
- **`gollama update`** — updates the `llama-server` inference engine. Downloads the latest pre-built binary or rebuilds from source.

Run both to stay fully up to date:
```bash
gollama self-update
gollama update
```

## Configuration

`~/.gollama/config.json` — created automatically on first run with sensible defaults:

```json
{
  "default_flags": ["--flash-attn", "on", "--ctx-size", "4096"]
}
```

If a GPU is detected, `--n-gpu-layers 99` is added automatically. User-provided flags always override config defaults. Edit the file directly to customize.

## Custom Flags

Any llama-server flag works. Pass them after the model name:

```bash
gollama chat model.gguf --flash-attn on --ctx-size 8192 --cont-batching
```

Common flags:

| Flag | Description |
|------|-------------|
| `--n-gpu-layers 99` | Offload all layers to GPU |
| `--tensor-split 12,8` | Manual GPU split (e.g. 3060:12GB, 2080:8GB) |
| `--ctx-size 4096` | Context window |
| `--flash-attn on` | Flash attention (reduces VRAM) |
| `--cont-batching` | Continuous batching |
| `--cache-type-k q4_0` | KV cache quantization (reduces VRAM) |
| `-np N` | Parallel slots |

## Model Metadata

gollama reads GGUF file headers to display model information automatically:

- **Architecture** — llama, gemma, qwen2, etc.
- **Quantization** — Q4_K_M, Q5_K_M, Q8_0, etc. (20+ types recognized)
- **Context length** — max tokens the model supports

Metadata is shown in the web UI and populated when downloading or listing models.

## Architecture

```
~/.gollama/
├── bin/
│   └── llama-server        # Inference engine (auto-downloaded or built)
├── models/
│   └── *.gguf              # Downloaded models
├── logs/
│   └── port-NNNN.log       # Instance logs (viewable from web UI)
├── index.json              # Model registry (with cached metadata)
└── config.json             # Default flags configuration
```

## Notes

- **Ports**: `gollama serve` uses port 9080. `gollama run`/`chat` auto-pick free ports starting from 8081. If a port is busy, the next available port is used automatically.
- **VRAM**: gollama needs free GPU memory. Stop Ollama (`systemctl stop ollama`) before launching instances if both use the same GPUs.
- **Linux CUDA**: pre-built CUDA binaries are not available for Linux on the llama.cpp release page. If CUDA toolkit is detected (`nvcc` in PATH), `gollama update` will build llama-server from source automatically with CUDA support. Otherwise it falls back to Vulkan (also supports NVIDIA GPUs).
- **Dependencies**: On minimal Linux installations, gollama auto-installs missing shared libraries (libgomp1, libatomic1) via apt-get.
- **Multi-instance**: each instance runs on its own port. Chat with any running instance from the web UI or terminal.

## Project Structure

```
gollama/
├── main.go                 # CLI entry point
├── pkg/
│   ├── chat/               # Terminal streaming chat
│   ├── llama/              # llama-server binary management
│   ├── manager/            # Instance lifecycle
│   ├── model/              # Model index, HF pull, GGUF parser
│   ├── server/             # HTTP API + web UI handler
│   └── ui/                 # Embedded web UI (HTML/CSS/JS)
├── install.sh              # Linux/macOS one-liner
├── install.ps1             # Windows one-liner
└── .github/workflows/
    └── release.yml         # Auto-build binaries on tags
```

## Why gollama?

Ollama hides llama.cpp flags and hardcodes defaults. gollama exposes every parameter while keeping convenience — model management, terminal chat, web UI, multi-instance, and full llama-server control. Perfect for multi-GPU setups, MTP testing, or when you need precise control over inference.

## User interface

<img width="1920" height="927" alt="gollama" src="https://github.com/user-attachments/assets/33c8077c-2fe1-491a-a01c-dac79ec85b7e" />
