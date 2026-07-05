# gollama 🦙

**Spin up GGUF models in seconds** — a single Go binary that downloads, manages, and runs llama.cpp instances with an OpenAI-compatible API, real-time streaming chat, reasoning display, web UI, REST API, and full flag control.

Pull any model from HuggingFace, launch it on any port, chat with streaming token-by-token responses — all from one command. No dependencies, no Docker, no Python.

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

**Install a specific version** (e.g. release):
```bash
VERSION=v3.0.0 curl -fsSL https://raw.githubusercontent.com/majidkorai/gollama/main/install.sh | sh
```

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
gollama update                  # Install llama-server binary (select GPU backend)
gollama pull hf.co/...          # Download a model from HuggingFace
gollama serve                   # Open web UI — manage everything from the browser
```

The first-run wizard:
1. Detects your GPU and downloads the right `llama-server` build
2. Offers to pull a popular starter model (Gemma, Qwen, Llama, or skip)
3. Shows next steps to start chatting

## Features

- **OpenAI-compatible API** — `/v1/chat/completions`, `/v1/completions`, `/v1/models`. Works with any OpenAI SDK or tool (Flowise, Cursor, continue.dev). Auto-routes by model name. **Auto-launches** the model if not running — like Ollama but with multi-instance.
- **Real-time streaming chat** — tokens arrive as they're generated. Reasoning models show thinking process live.
- **Web UI** — modern dark-themed interface with dashboard, model management, chat, log viewer, settings. Built-in, no extra setup.
- **Searchable flag dropdown** — type to filter 150+ llama-server flags with real-time suggestions, arrow key navigation, and value hints. No more scrolling through a giant list.
- **Settings protected mode** — configuration sections are read-only by default with an Edit button to reveal forms. Prevents accidental changes.
- **Model Profile presets** — named profiles bundling model, flags, env vars, and reasoning strip toggle. Auto-selected by model name in API requests.
- **Model search-as-you-type** — search HuggingFace directly from the pull input. Shows model sizes, likes, and downloads. Click any result to pull.
- **Multi-file GGUF split download** — automatically detects and downloads all parts of split models (e.g. `model-00001-of-00005.gguf` through `model-00005-of-00005.gguf`) with per-part progress.
- **Per-part progress** — terminal and web UI show which part is downloading (`[2/6] Downloading…`).
- **Auto-stop idle instances** — configurable TTL (default 30 min). Stops unused instances to free GPU memory.
- **Model management** — list, pull, delete models from HuggingFace. Click any model to see architecture, quantization, context length, and file path.
- **Multi-instance** — run multiple models on separate ports simultaneously. Restart with modified flags.
- **Live log tail** — view llama-server logs in the UI with auto-refresh.
- **Disk space check** — warns before downloading if insufficient space.
- **CLI & Web UI** — use `gollama chat <model>` in the terminal or open `http://localhost:9080`.

## Commands

| Command | Description |
|---------|-------------|
| `gollama` | First-run wizard (auto-setup if fresh install) |
| `gollama update` | Download/update llama-server binary (re-select GPU backend) |
| `gollama self-update` | Update gollama itself to the latest version |
| `gollama pull <model>` | Download a GGUF model from HuggingFace |
| `gollama serve [port]` | Web UI + REST API on :9080 (main workflow) |
| `gollama chat <model> [flags]` | Start a terminal chat session (streaming) |
| `gollama list` | List downloaded models with metadata |
| `gollama ps` | List running instances |
| `gollama stop <port>` | Stop an instance |
| `gollama restart` | Restart gollama server (stops all instances, uses systemctl if installed as service) |
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

- **Dashboard** — metrics overview (models, instances, tokens/sec), quick launch with searchable flag editor and presets, running instances grid with live status badges
- **Models** — list all downloaded models with metadata badges (arch, quant, context length, file path), search-as-you-type pull input, refresh button
- **Chat** — full chat workspace with any running instance (proxied, no CORS), chat history with save/load/rename/delete, copy button, context meter
- **Settings** — version info, idle TTL, default launch flags and API launch defaults (read-only with Edit button), Model Profiles management with per-profile env vars and strip reasoning toggle
- **Left sidebar** — gradient-accented navigation, theme toggle, collapsible
- **Instance cards** — glass-styled cards with gradient accent bar, port, model, tokens/sec, uptime, idle time, total tokens, flags, model profile badge, actions (stop, restart, chat, open, logs)

## OpenAI-Compatible API

gollama exposes an OpenAI-compatible API at `/v1`. **If the model isn't running, it auto-starts it** — like Ollama, but with multi-instance support. Point any OpenAI SDK or tool at the single fixed endpoint:

```python
from openai import OpenAI
client = OpenAI(base_url="http://<host>:9080/v1", api_key="not-needed")
response = client.chat.completions.create(
    model="gemma-4-12b",    # auto-starts if not running
    messages=[{"role": "user", "content": "Hello!"}]
)
```

**Endpoints:**
| Endpoint | Description |
|----------|-------------|
| `GET /v1/models` | List downloaded models (not running instances) |
| `POST /v1/chat/completions` | Chat completions — auto-starts model if needed (streaming supported) |
| `POST /v1/completions` | Text completions — auto-starts model if needed (streaming supported) |

The `model` field accepts the full name (e.g. `hf.co/unsloth/gemma-4-12b-it-GGUF:Q4_K_M`) or **any substring** like `gemma-4-12b`. If it matches an indexed model, gollama auto-launches it on a free port, waits for it to be ready, then proxies your request.

You can still launch multiple instances manually from the UI — the API will route to an already-running instance if one exists, or auto-start a new one for an unused model.

## Updating

gollama has two separate update commands:

- **`gollama self-update`** — updates the `gollama` binary itself. Downloads the latest release from GitHub and replaces the current binary.
- **`gollama update`** — updates the `llama-server` inference engine. Re-detects your GPU and prompts to select a backend, then downloads the latest pre-built binary or rebuilds from source.

Run both to stay fully up to date:
```bash
gollama self-update
gollama update
```

## Configuration

`~/.gollama/config.json` — created automatically on first run with sensible defaults:

```json
{
  "default_flags": ["--host", "0.0.0.0", "--ctx-size", "2048", "--flash-attn", "on", "--temp", "0.7"],
  "proxy_defaults": [],
  "profiles": {},
  "idle_ttl": 30
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_flags` | string[] | — | Default flags for manual launches (Quick Launch UI / CLI) |
| `proxy_defaults` | string[] | — | Default flags for API auto-launched instances (falls back to `default_flags` if empty) |
| `idle_ttl` | int | 30 | Auto-stop idle instances after N minutes (0 = disable) |
| `profiles` | object | — | Named Model Profiles (see [Model Profiles](#model-profiles) below) |

Edit via the **Settings** page in the web UI or directly in the file. If a GPU is detected, `--n-gpu-layers` is added to the pre-filled form.

### Model Profiles

Model Profiles are named presets that bundle model selection, launch flags, environment variables, and reasoning behavior. When the API proxy receives a request for a model matching a profile's model name, it automatically applies the profile's settings. Manage them from the **Settings** page in the web UI.

```json
{
  "profiles": {
    "coding": {
      "model": "qwen3-coder-next",
      "flags": ["--ctx-size", "81920", "--temp", "1.0"],
      "strip_reasoning": true,
      "env": {
        "CUDA_VISIBLE_DEVICES": "0"
      },
      "description": "Qwen coder, GPU 0 only"
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `model` | string | Model name to auto-select this profile (fuzzy match) |
| `flags` | string[] | Launch flags for this profile |
| `strip_reasoning` | bool | Strip `reasoning_content` from API responses |
| `env` | object | Environment variables applied to the llama-server process |
| `description` | string | Human-readable description (displayed in UI) |

- **Env vars** (`env`) let you control GPU selection (`CUDA_VISIBLE_DEVICES`), thread counts, or any process-level variable — applied per instance.
- **Strip reasoning** (`strip_reasoning`) strips `reasoning_content` fields from streaming and non-streaming API responses. Useful for tools like OpenClaw that don't support reasoning tokens.
- **Auto-selection**: When the API proxy auto-launches a model, it checks if any profile's `model` matches the request and applies that profile's flags, env, and strip setting.

## Custom Flags

Any llama-server flag works. In the web UI, flags are pre-filled from the config file and editable via a searchable dropdown with real-time filtering, value hints, and standalone flag detection. In the CLI, pass them after the model name:

```bash
gollama chat model.gguf --flash-attn on --ctx-size 8192 --cont-batching
```

Common flags (searchable from the UI flag editor):

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

- **Ports**: `gollama serve` uses port 9080. `gollama run`/`chat` auto-pick free ports starting from 8081. The API endpoint at `:9080/v1/chat/completions` auto-launches models — use it as a fixed endpoint like Ollama.
- **Auto-launch**: Calling `/v1/chat/completions` with a model name that's indexed but not running will auto-start it on a free port. The instance appears in the UI and can be stopped/restarted like any manually launched one.
- **VRAM**: gollama needs free GPU memory. Stop any other tool that might be using GPU before launching instances if both use the same GPUs.
- **Linux CUDA**: llama.cpp does not ship pre-built CUDA binaries for Linux. To build from source, install the required tools (`apt install git cmake build-essential nvidia-cuda-toolkit`) and run `gollama update` — it will detect `nvcc` and compile llama-server with CUDA support. Otherwise it falls back to Vulkan (also supports NVIDIA GPUs with good performance).
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

Ollama hides llama.cpp flags and hardcodes defaults. gollama exposes every parameter while keeping convenience — model management, terminal chat, web UI, multi-instance, and full llama-server control.

**Key advantage over Ollama:** Ollama cannot download sharded (multi-file) GGUF models from HuggingFace — it throws an error on any model split across multiple files (common for 100B+ parameter models). This effectively locks you out of the latest and largest open-weight models unless you use their cloud service. Gollama downloads all parts automatically with per-part progress. No gatekeeping — if it's on HuggingFace, you can run it.

Perfect for multi-GPU setups, MTP testing, or when you need precise control over inference without vendor lock-in.

## User interface

<img width="1920" height="927" alt="gollama" src="https://github.com/user-attachments/assets/33c8077c-2fe1-491a-a01c-dac79ec85b7e" />
