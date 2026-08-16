# gollama — Agent Context

## Project Overview

Go binary that manages llama.cpp instances. Single-file CLI + web UI + OpenAI-compatible API proxy. Zero external dependencies.

**Version:** v3.0.0
**Module:** `github.com/majidkorai/gollama`
**Go:** 1.23
**Dependencies:** None (stdlib only)

## Key Commands

```bash
go build ./...          # build all packages
go test ./...           # run all tests
go vet ./...            # static analysis
```

**Release builds** (CI does this automatically on `v*` tags):
```bash
GOOS=linux GOARCH=amd64 go build -o gollama-linux-amd64 -ldflags="-s -w -X main.version=vX.Y.Z" .
```

## Project Structure

```
gollama/
├── main.go                    # CLI entry point (cobra-less, raw flag parse)
├── install.sh                 # Linux/macOS installer
├── install.ps1                # Windows installer
├── logo.svg                   # Embedded SVG logo
├── .github/workflows/release.yml  # GitHub Actions: test + matrix build + release
│
├── pkg/
│   ├── chat/
│   │   └── chat.go            # Terminal streaming chat (stdin/stdout)
│   ├── llama/
│   │   └── binary.go          # llama-server binary download/build/update
│   ├── manager/
│   │   ├── manager.go         # Instance lifecycle (start/stop/list/metrics)
│   │   └── manager_test.go
│   ├── model/
│   │   ├── model.go           # Config, model index, HF pull/search/scan
│   │   ├── model_test.go
│   │   ├── gguf.go            # GGUF binary reader + metadata parser
│   │   ├── gguf_test.go
│   │   ├── presets.go         # Flag presets CRUD (separate file ~/.gollama/presets.json)
│   │   ├── presets_test.go
│   │   ├── chat_history.go    # Chat session persistence (~/.gollama/chats/)
│   │   ├── disk_free_unix.go  # freeDiskBytes() via syscall.Statfs
│   │   └── disk_free_windows.go  # freeDiskBytes() stub
│   ├── server/
│   │   └── server.go          # HTTP server: web UI + REST API + OpenAI proxy
│   └── ui/
│       └── ui.go              # Embedded web UI (2170-line const string: HTML/CSS/JS)
```

## Architecture

### Data Flow

```
User → CLI (main.go) → Manager API (manager.go) → llama-server child process
User → Browser → Server HTTP (server.go) → Manager API → llama-server
User → OpenAI Client → /v1/chat/completions (server.go) → Manager auto-launch → llama-server
```

### Key Types

**Config** (`pkg/model/model.go:48`):
```go
type Config struct {
    DefaultFlags  []string            `json:"default_flags"`
    ProxyDefaults []string            `json:"proxy_defaults"`
    Profiles      map[string]Profile  `json:"profiles,omitempty"`
    IdleTTL       int                 `json:"idle_ttl"`
}
```

**Profile** (`pkg/model/model.go:40`):
```go
type Profile struct {
    Model          string            `json:"model,omitempty"`
    Flags          []string          `json:"flags"`
    Description    string            `json:"description,omitempty"`
    StripReasoning *bool             `json:"strip_reasoning,omitempty"`
    MergeReasoning *bool             `json:"merge_reasoning,omitempty"`
    Env            map[string]string `json:"env,omitempty"`
    Type           string            `json:"type,omitempty"` // "text" (default) or "image"

    // Image-specific defaults (optional — user can override in UI/API)
    Steps    *int     `json:"steps,omitempty"`
    Guidance *float64 `json:"guidance,omitempty"`
    Size     *string  `json:"size,omitempty"`
    N        *int     `json:"n,omitempty"`
}
```

**Instance** (`pkg/manager/manager.go:23`):
```go
type Instance struct {
    Port, PID int
    Model, BlobPath, Status string
    Ready bool
    TokensPerSec float64
    Flags []string
    StartedAt, LastActivity time.Time
    TotalTokens int64
    Profile string          // profile name used to launch
    Type    string          // "text" or "image"
    // ... GPU/CPU metrics
}
```

**ModelInfo** (`pkg/model/model.go:21`):
```go
type ModelInfo struct {
    Name, ShortName, BlobPath string
    Size int64
    Architecture, Quantization string
    ContextLength uint64
    BlockCount uint32
    Source string
}
```

### Profile Type

Profiles support a `type` field: `"text"` (default) or `"image"`. Image type launches a Python diffusers subprocess instead of llama-server.

Image profiles can bundle generation defaults (`steps`, `guidance`, `size`, `n`) so each model carries sensible defaults. The UI auto-fills these when selecting a profile and the API merges them into the request body if not explicitly provided.

### Image Generation

Gollama supports image generation via Python diffusers subprocesses. Configured through profiles with `"type": "image"`.

**Web UI:** 🎨 Image tab — full playground with prompt, advanced parameters (n, size, steps, guidance, seed), result cards with download/re-generate, lightbox, and history (localStorage). Selecting a profile auto-fills steps/guidance/size/n from the profile.

**Model browser:** 🔍 Browse button searches HF for text-to-image models. "Add" creates an image profile with auto-detected defaults (schnell→4 steps/0 guidance, dev→28 steps/3.5 guidance). The model list shows configured profiles with cached/not-cached status and size.

**Settings:** Separate "Image Profiles" card (no llama.cpp flags shown). Edit mode shows steps (number), guidance (number), size (dropdown with presets + Custom option), n (number). Read-only view displays the configured parameters.

**API endpoints:**
- `POST /v1/images/generations` — OpenAI-compatible, auto-launches image model on first request
- `GET /api/v1/image-models` — list configured image profiles with cache status
- `GET /api/v1/image-models/search?q=` — search HF for image generation models
- `POST /api/v1/image-models/install` — add a new image profile

**API params:** `prompt` (required), `profile`, `model`, `n` (1-8), `size` (e.g. `"1024x1024"`), `steps`, `guidance`, `seed`, `response_format`. If not set in the request, values from the profile's `steps`/`guidance`/`size`/`n` fields are used as defaults.

**Caller guidance (for agents/cron):**
- **Omit `steps`/`guidance`/`size`/`n`** — they fall back to profile defaults. Only send them if you need to override.
- **503 handling:** Model loads ~40s from cold start. Returned with `Retry-After: 5` header. Retry with that delay (5s) — ~8-10 retries should be enough.
- **Image generation time:** ~1.2s per step for FLUX.1-dev (28 steps ≈ 34s).
- **Total time from first request:** ~75s (40s start + 34s gen) when starting from cold.
- **Recommended retry strategy:** Use server's `Retry-After` header (not hardcoded). Retry up to 30 times for 503s, 10 times for other errors/connection failures.

**Automatic sequential GPU switching:** Text models are stopped before image gen loads, and vice versa. This enables single-GPU setups — no GPU env vars needed.

**Example image profile with defaults:**
```json
{
  "profiles": {
    "flux-dev": {
      "model": "black-forest-labs/FLUX.1-dev",
      "type": "image",
      "description": "FLUX.1-dev (28 steps, high quality)",
      "steps": 28,
      "guidance": 3.5,
      "size": "1024x1024",
      "n": 1,
      "env": {
        "HF_TOKEN": "hf_..."
      }
    },
    "flux-schnell": {
      "model": "black-forest-labs/FLUX.1-schnell",
      "type": "image",
      "description": "FLUX.1-schnell (4 steps, fast)",
      "steps": 4,
      "guidance": 0,
      "size": "1024x1024",
      "n": 1
    }
  }
}
```

**API:** `POST /v1/images/generations` — OpenAI-compatible, auto-launches image model on first request (503 + Retry-After: 5). Profile defaults merge into the proxied request.

**Lifecycle:** Same as text instances — idle timeout, health checks, log management.

**Paths:**
- `GOLLAMA_IMAGE_PYTHON` env var overrides Python binary (default: `/opt/image-api/.venv/bin/python`)
- `GOLLAMA_IMAGE_APP` env var overrides app script (default: `/opt/image-api/app.py`)

### Config (`~/.gollama/config.json`)

Auto-created on first `LoadConfig()` call. Defaults from `DefaultConfig()`:

```json
{
  "default_flags": ["--host","0.0.0.0","--ctx-size","2048","--flash-attn","on","--temp","0.7"],
  "proxy_defaults": [],
  "profiles": {},
  "idle_ttl": 30
}
```

- `default_flags`: used by Quick Launch form and CLI `gollama run`/`chat`
- `proxy_defaults`: used by API auto-launch; falls back to `default_flags` if empty
- `profiles`: map of named Model Profiles, each bundling model, flags, env vars, type, strip_reasoning
- `idle_ttl`: auto-stop idle instances after N minutes (0=disable)

## API Endpoints (`pkg/server/server.go`)

All routes registered in `registerRoutes()`. Key ones:

| Route | Method | Purpose |
|-------|--------|---------|
| `/api/v1/models` | GET | List downloaded models |
| `/api/v1/models/search` | GET | HF search (`?q=`) |
| `/api/v1/models/delete` | POST | Delete model |
| `/api/v1/models/pull` | POST | Start pull from HF |
| `/api/v1/models/pull/stream` | GET | SSE streaming progress |
| `/api/v1/instances` | GET/POST | List/launch instances |
| `/api/v1/instances/stop` | POST | Stop by `?port=` |
| `/api/v1/instances/logs` | GET | Logs by `?port=` |
| `/api/v1/config` | GET/POST | Read/write config.json |
| `/api/v1/config/default-flags` | GET | Code defaults + GPU auto-detect |
| `/api/v1/presets` | GET/POST/DELETE | Flag presets CRUD |
| `/api/v1/chats` | GET | List chat sessions |
| `/api/v1/chats/{id}` | GET/PUT/DELETE | Single chat CRUD |
| `/api/v1/version` | GET | Version string |
| `/api/v1/restart` | POST | Restart gollama |
| `/v1/chat/completions` | POST | OpenAI-compatible (auto-launches) |
| `/v1/completions` | POST | OpenAI-compatible (auto-launches) |
| `/v1/images/generations` | POST | OpenAI-compatible image generation (auto-launches image profile) |
| `/v1/models` | GET | OpenAI model list |
| `/` | GET | Serves embedded web UI |

### Auto-launch Pattern (OpenAI proxy)

`/v1/chat/completions` and `/v1/completions` auto-start a model if not running:
1. Resolves `profile` from request body or auto-detects by model name
2. Checks for running instance matching model name
3. If none found, auto-starts with profile flags or proxy_defaults
4. Returns 503 + `Retry-After: 5` while starting (client retries)
5. On subsequent retry, proxies the request to the running instance

### Seamless Cold Start (v3.4.x)

- `waitForInstanceReady` returns an error (including the instance log tail) instead of silently timing out after 60s. Deadline defaults to 5 min, override with `GOLLAMA_MODEL_LOAD_TIMEOUT` (seconds).
- Streaming (`stream: true`) requests get SSE headers immediately plus comment heartbeats (`: model loading...`) every ~10s while the model loads, so the connection stays alive past client/proxy idle timeouts during long cold starts.
- If upstream answers 503 "loading model" after `/health` passes, the proxy retries with heartbeats until the load deadline.
- Start failures surface as 500 JSON (non-stream) or an OpenAI-style SSE error event (stream), including the last lines of the instance log — and the wait fails fast when the model process exits before serving.
- A failed start doesn't poison the port: `manager.Start`/`StartImage` reuse stale (non-running) slot entries instead of returning "port already in use".

## CLI Commands (`main.go`)

| Command | Args | Description |
|---------|------|-------------|
| `gollama` | — | First-run wizard or usage |
| `serve` | `[port]` | Web UI + API server (default :9080) |
| `chat` | `<model> [flags...]` | Terminal streaming chat |
| `run` | `<model> [flags...]` | Run model server directly |
| `pull` | `<model>` | Download GGUF from HF |
| `list` | — | List downloaded models |
| `ps` | — | List running instances |
| `stop` | `<port>` | Stop instance |
| `delete` | `<model>` | Delete model from disk |
| `update` | — | Re-download llama-server |
| `self-update` | `[version]` | Update gollama binary |
| `install-service` | — | Install systemd service |
| `restart` | — | Restart gollama |
| `logs` | `<port>` | Print instance logs |

## UI Architecture (`pkg/ui/ui.go`)

Single `const Page` string (~2170 lines) containing the entire web app: HTML template + CSS + JS. No framework, no build step.

### Views (JS SPA with `switchView()`):
- **Dashboard** (`#view-dashboard`): Metrics cards + Quick Launch form + Running Instances grid
- **Models** (`#view-models`): Model list with badges + Pull model with HF search + details modal
- **Chat** (`#view-chat`): SSE streaming chat with reasoning display + history management
- **Settings** (`#view-settings`): Version info + idle TTL + default flags + API defaults + Model Profiles + restart

### JS Patterns:
- Navigation: `switchView(name)` — hides all `.view`, shows target, calls `loadInstances()`/`loadModels()`/`loadChats()`/`loadSettings()` depending on view
- Sidebar: Collapsible (60px), expanded (220px), state in localStorage
- Theme: Dark/light toggle, persisted in localStorage, CSS custom properties
- Flags: Searchable dropdown of 150+ llama-server flags with auto-complete, hints, standalone flag detection
- Dialogs: Model details, chat history, logs modal, pull progress bar

### CSS Design System (v3.0):
- Dark theme with CSS custom properties
- Accent gradient (`#00e5bf → #00b8ff`) used for buttons, headings, card accents
- Glass effects (`backdrop-filter: blur()`), glow shadows, gradient borders
- Smooth transitions (200ms cubic-bezier)
- Responsive breakpoint at 768px (collapsed sidebar, single column)

## Code Conventions

1. **Zero external dependencies** — stdlib only. No gorilla/mux, no cobra, no chi, no templ.
2. **Single ui.go file** — all HTML/CSS/JS in one Go const string. Edit with care (no template engine).
3. **Model profiles** — stored in `config.json` under `profiles` key. Always named "Model Profiles" in the UI.
4. **Config access** — always call `model.LoadConfig()` (reads from disk every time). Save with `model.SaveConfig()`.
5. **Instance management** — through `manager.Manager` (mutex-protected map of ports). Ports auto-assigned from 8080+.
6. **SSE streaming** — used for chat responses, model pull progress. `Transfer-Encoding: chunked` + `text/event-stream`.
7. **Error responses** — JSON with `{ "error": "message" }` format. Use `jsonResponse()` / `jsonError()` helpers.

## Testing

```bash
go test ./pkg/model      # FormatSize, DefaultConfig, Config paths, ProgressReader, Presets, GGUF metadata
go test ./pkg/manager    # NewManager, List, Stop, UpdateTokens
```

Tests use temp dirs (no real config). No test fixtures on disk.

## Important Files

- `pkg/ui/ui.go` — the giant const string. All UI changes happen here. Look for the `// ── Section ──` comments to navigate.
- `pkg/server/server.go` — all HTTP handlers, route registration, OpenAI proxy logic.
- `pkg/manager/manager.go` — instance lifecycle, metrics polling, idle timeout.
- `pkg/model/model.go` — config management, model index, HF pull/search, flag handling.
- `main.go` — CLI dispatch (switch on os.Args[1]), version injected at build time.
