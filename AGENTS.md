# gollama — Agent Context

## Project Overview

Go binary that manages llama.cpp instances. CLI + embedded web UI + OpenAI-compatible API proxy. Zero external dependencies.

**Version:** v4.4.1
**Module:** `github.com/majidkorai/gollama`
**Go:** 1.23
**Dependencies:** None (stdlib only)

## Current Plan

**v4.4.1 (2026-09-06) — flag editor fix + b10815 catalog refresh via `tools/flags`:** (1) **Bug fix:** a flag typed into the profile editor's search field that is not in the catalog now re-renders into the **search field** on the next edit (with an amber dotted-underline `flag-unknown` marker + "stored as typed" tooltip) instead of spawning an extra key field; the key field only appears via the dropdown's "Custom…" item (`makeFlagRow` in `app.js`; `app.css` `.flag-search.flag-unknown`). Round-trip `renderFlags`→`collectFlags` is byte-stable for unknown flags. (2) **Reproducible catalog generator:** new `tools/flags` (Go, stdlib-only) — `flags probe <help.txt> <binary>` probes every flag group through the live binary (A: `F --list-devices` → device list=STANDALONE / `invalid argument`=INVALID / "has been removed"=REMOVED; B: `F __gollama_probe__ --list-devices` → device list=VALUED / `error while handling argument`=VALUED / `invalid argument: __gollama_probe__`=STANDALONE; 20s timeout) and `flags parse <help.txt> <probes.tsv> [--dry-run] [--force]` regenerates the three `app.js` blocks + the `model.go` standalone map in place between `BEGIN/END generated flag catalog` markers. Generation rules: every existing catalog name is kept (alias continuity — profiles on disk never churn; a group contributes its existing long aliases, else its first), the standalone set carries **all** long aliases of standalone groups so parsers never swallow the next token, REMOVED/INVALID flags are excluded but reported, UNKNOWN probe results warn + exclude; a shrink tripwire refuses any run that would cut the catalog by more than half without `--force`. Help parsing encodes the b10815 format contract (`common/arg.cpp to_string`: alias column padded to exactly 40, inline help at index 40 iff index 39 is a space, continuations indented 40, dotted names like `--fim-qwen-1.5b-default`). (3) **Catalog: 284→298 flags** against live obelisk b10815 (`92ffcfd0d`): added `--lazy-mode` (the user's original trigger — the live flash profile uses it; replaces the dropped `--tensor-read-lazy`, upstream-renamed in v0.4.0), `--moe-expert-cache{,-inserts}` (local MoE patch), the quick-start defaults `--fim-qwen-*` (7) / `--gpt-oss-*-default` (2) / `--vision-gemma-*-default` (2) / `--embd-gemma-default`, and `--special`; all 29 flags in the live config verified covered. Standalone set 65→98 (every `--no-*` alias form now explicit). Page reference regenerated; full Go suite green; committed Node smoke test `tools/flags/flagrow_smoke.js` (33 checks, `node tools/flags/flagrow_smoke.js` — no deps) guards the row logic against future edits. Deployed to obelisk as v4.4.1.


**v4.4.0 (2026-08-28) — llama.cpp flag list refresh (b10665):** the flag catalog was re-derived against the live VM build (4e97ac86e, b10665+PR27836) by probing every candidate flag through the real binary (VALUED / STANDALONE / INVALID) instead of trusting `--help` text. Dropped from the UI catalog: 5 flags the binary now **rejects** with `error: invalid argument` — `--no-flash-attn`, `--hf-file-v`, `--hf-repo-v`, `--model-vocoder`, `--tts-use-guide-tokens` (the old TTS/vocoder stack is gone) — plus 3 duplicate alias entries (`--log-verbosity`→`--verbosity`, `--no-webui`→`--no-ui`, `--no-webui-mcp-proxy`→`--no-ui-mcp-proxy`). Added 36 new flags with hints, headliners: `--load-mode` (auto/none/mmap/mlock/mmap+mlock/dio — the replacement for the deprecated `--mmap`/`--mlock`/`--direct-io`, which stay in the catalog but are now marked DEPRECATED in their hints, and **stay in both standalone sets** because live configs on the gollama VM still use `--mlock`/`--no-mmap` and the parsers need them to not swallow the next token), `--reasoning-effort` (minimal…xhigh/max — the first-class flag behind the `reasoning_effort` chat_template_kwargs already in use), `--tensor-read-lazy`, `--n-cpu-ffn`, `--kv-unified-per-slot`, `--mmproj-device`, the full CORS quartet, `--tools-runtime` + `--mcp-servers-{config,json}`, `--reasoning-preserve`/`--no-reasoning-preserve`, `--spec-default`, `--spec-synth-{len,rates}` (benchmarking), `--spec-draft-{cpu-mask,cpu-range,cpu-strict,prio,poll}` + batch/draft variants, `--video-{fps,timestamp-interval,ffmpeg-dir}`, and the positive forms `--warmup`/`--slots`/`--kv-offload`/`--mmap` (negatives were already cataloged). The `standaloneFlags` set is mirrored in `pkg/model/model.go` (parse/merge path) and `pkg/ui/web/app.js` (UI path, no derivation — every `--no-*` must be listed explicitly); pinned tests in `flags_model_test.go` that used `--no-flash-attn` as their example now use `--no-host`/`--host` (the quirk they pin — one-way standalone derivation — is unchanged). Page reference regenerated (`TestDumpPage`); full suite green; served page verified byte-identical against a live `serve` on a temp HOME.


**v4.3.2 (2026-08-23) — UD-IQ* quant suffixes:** `knownQuantSuffixes` gained the Unsloth `UD-IQ*` compounds (UD-IQ2_XXS … UD-IQ4_XS, UD-IQ1_S/UD-IQ1_M) so `DeepSeek-V4-Flash-0731-UD-IQ4_XS` scans as `deepseek-v4-flash-0731` instead of leaving a dangling `-ud` in the name. (The v4.3.1 binary deployed to the gollama VM already contains this; the tag was cut before the change, hence v4.3.2.)


**v4.3.1 (2026-08-23) — model-list duplicate fix + GGUF v3 parser repair:** the model list showed split (multi-file) models twice — once per index key — because the pull path and the scan path disagreed on the index key for the same blob (e.g. `deepseek-v4-flash-0731-ud` vs `deepseek-v4-flash-0731-ud-iq4-xs`, same 4-part set). Fixes: (1) one index key per blob file — `collapseIndexBlob`/`dedupIndexByBlob` (pkg/model/model.go) run on every scan, the scan's split branch refreshes an existing entry in place instead of adding a second key, and both pull index sites (fresh + already-exists) collapse to the existing key; `ListModels` has a defensive in-memory dedup (largest size wins, lexicographic tie-break) so a corrupt index never shows one file twice. (2) Pull path indexed split sets with part-1's size only (10 GB instead of 136 GB) — now the sum of all parts, both at fresh-download and re-index time. (3) **GGUF parser was silently failing on every recent file** (v4.3.1): `skipGGUFValue` skipped FLOAT32 as 8 bytes (spec: 4), desynchronizing the metadata stream at the first float key (`general.sampling.*` in modern quant repos) so architecture/quant/ctx badges were empty for *all* models; `ggufTypeNames` was also stale (23: IQ4_XS not IQ2_M, missing I8/I16/I32/I64/F64/IQ1_M@29/BF16@30/TQ1_0/TQ2_0/MXFP4/NVFP4/Q1_0/Q2_0); `<arch>.block_count` and `<arch>.context_length` are now accepted as UINT32 (type 4) in addition to the standard INT32/UINT64 (recent writers, e.g. deepseek4/qwen35). Verified e2e against the real DeepSeek-V4-Flash-0731 UD-IQ4_XS part-1 (arch=deepseek4, ctx=1M, blocks=43) and Qwen3.8-27B (qwen35, 262144, 65). Deployed to the gollama VM; the VM's stale index (69 entries, 59 dead-blob, 2 duplicate-blob groups) was repaired in place (backup `/root/.gollama/index.json.bak-20260823-dedup`) — 10 live models remain.


**`docs/PHASE_6_PLAN.md` (→ `v4.3.0`, ops + freshness + UI wins) — all tasks T1–T6 complete, verified, ready to tag + deploy:** T1 `GET /healthz` liveness endpoint (token-free, for Uptime Kuma); T2 llama-server freshness check (cached GitHub lookup with 1h TTL via `LatestReleaseInfo`, `llama-server --version` exec fallback in `InstalledLlamaServerVersion`, pure `CompareBuildNumbers`, extended `/api/v1/version` response, `gollama llama-version` CLI; `GOLLAMA_RELEASE_API_BASE` env seam for tests); T3 freshness badge in Settings (amber "N BEHIND" / green "UP TO DATE", nothing when not comparable — no false alarms on custom builds); T4 chat stream stop button (AbortController; the server already cancels upstream on client disconnect per P3-T5; partial content preserved on stop); T5 OpenAI endpoint row with Copy in Settings; T6 responsiveness polish (mobile logo subtitle hidden ≤768px; content capped at 1440px centered for ultrawide) + delete-button icon contrast fix (white trash emoji on red via `filter`). Per-request llama.cpp flags, flags-aware instance identity, and Prometheus metrics are explicitly parked in that plan — revisit only on concrete demand.

**`docs/ROBUSTNESS_PLAN.md` (Phases 0–5, complete)** — phased fixes for security, correctness, concurrency, and code-quality issues found in the 2026-08-18 architecture review. Its "Resume here" block documents the test conventions and pinned behavior quirks still in force.

- Phases 0–4 are complete (security `v3.8.0`, correctness `v3.9.0`, concurrency & lifecycle `v4.0.0`, robustness `v4.1.0`); Phase 5 (refactor → `v4.2.0`) is in progress — P5-T1 done (one SSE streaming proxy: shared `proxySSE`/`postToInstance` in `pkg/server/server.go`; `/api/v1/chat` now sends exactly one `[DONE]`, forwards the trailing usage chunk, and runs the same reasoning transforms as the OpenAI proxy); P5-T2 done (one GGUF parser + one quant regex: `ReadBlockCount` deleted in favor of cached `model.GGUFMetadataCached`, exported `model.StripQuantSuffix` replaces three divergent regexes, scan-path short names are now clean); P5-T3 done (typed flag model: `pkg/model/flags.go` `ParseFlags`/`Merge`/`Args` replaces `sanitizeFlags` + the `ProfileFlags` and `manager.Start` merge loops; standalone flags now set/unset their counterpart — effective behavior unchanged); P5-T4 done (UI split: the 3200-line `const Page` is now assembled from `pkg/ui/web/{index.html,app.css,app.js}` via `go:embed`, served page stays a single self-contained HTML doc; `switchView` nav uses `data-view` attributes; byte-identity guarded by `TestPageMatchesReference`); P5-T5 done (all 71 `log.Printf` calls → leveled `slog` calls; `setupLogging()` in main.go configures the logger: text to stderr by default, `GOLLAMA_LOG_FORMAT=json` for JSON, `GOLLAMA_LOG_LEVEL` to override); **P5-T6 done (Phase 5 complete)** (systemd unit-location deduped into `manager.FindGollamaUnit()` in `pkg/manager/service.go`; `chat.WaitForReady` uses `model.GollamaDir()`; `tmpFileReader` dead code removed). **v4.2.0 tagged + pushed + deployed to the gollama VM (2026-08-22) — the robustness plan (Phases 0–5) is complete.**
- Phase 1 behavior changes: `serve` binds `127.0.0.1` by default (`--listen`/`GOLLAMA_LISTEN` to opt out), all `/api/v1/*` + `/v1/*` routes require the config's `api_token` when set, chat ids are validated, model-delete path check hardened, self-update verifies release checksums, dep installs are no longer silent.
- Phase 2 behavior changes (v3.9.0): `merge_reasoning` profiles now actually merge reasoning into content (stream + non-stream); `gollama run` no longer no-ops when an instance is alive (recovers orphans via `ps` scan, starts a second instance); `serve`/`run` shutdown stops all instances (SIGINT → 500ms → SIGKILL, bounded 5s); one readiness deadline everywhere (`model.LoadTimeout`, `GOLLAMA_MODEL_LOAD_TIMEOUT`, default 5m); model matching is deterministic (tiered scoring, lowest-port tie-break; multiple image profiles + no model → 400); Windows orphan recovery no longer registers phantom instances; per-GPU utilization (`gpu_util_per_gpu`) + instantaneous Linux CPU% (`/proc/<pid>/stat` double-sample, `ps` fallback).
- Phase 3 behavior changes (v4.0.0): every model switch goes through one `Coordinator` (`pkg/manager/coordinator.go`) — concurrent requests for the *same* model coalesce onto one process, requests for *different* models queue on the switch lock (no stop/start thrash), image warmup checks idempotency first and defers with 503 + `Retry-After: 30` while a text model has been active <30s; the manager no longer holds its mutex while sleeping (ports are picked + reserved under the lock, availability scans/spawn/metrics run outside it — parallel `Start`s get unique ports); recovered orphans get their `Ready` confirmed by a background health poll (metrics snapshot, dead-process detection, downgrade if it never serves); `model.ErrAlreadyExists`/`model.ErrNotFound` sentinels replace string-matched errors (model delete 404s only when genuinely missing, 500 on real index failures); the streaming proxy cancels the upstream request when the client disconnects. `Manager.NewManagerNoRecovery()` exists for hermetic tests (skips the orphan `ps` scan).
- Phase 4 behavior changes (v4.1.0): model downloads write to `<name>.gguf.part` and rename on success, resume interrupted pulls via HTTP Range (200-instead-of-206 → fresh restart; complete parts finalized without re-download; oversized stale parts restarted), and verify final size before promoting; `ScanModels` skips `*.part`/zero-byte files; the models-dir scan is throttled (60s, `ScanModelsForce`/`?refresh=1` bypass — the UI ↻ button sends it) and `ListModels` skips GGUF header re-parsing for complete index entries; instance logs are tailed (`manager.TailLogFile`) and rotated at 10 MB (`port-N.log.1`, one level); `SaveConfig`/`SaveIndex`/`UpdateIndex` return errors (API handlers → 500, internal callers log loudly); the `\r` download progress bar only renders on a TTY (clean systemd/cron output).
- When working on planned fixes, check the plan's task list and update its checkboxes.
- Tests pin several current behavior quirks on purpose — don't change them incidentally; they are scoped to specific later tasks (fuzzy model-match nondeterminism fixed in P2-T5; the `/api/v1/chat` `[DONE]` drop fixed in P5-T1; the `ScanModels` quant-suffix short name fixed in P5-T2; the `ProfileFlags` standalone both-survive quirk fixed in P5-T3).
- Tooling note: the session file-write pipeline strips angle-bracket pairs from content; think-tag literals in `pkg/server/transforms_test.go` are built from hex escapes for this reason.

## Key Commands

```bash
go build ./...          # build all packages
go test ./...           # run all tests
go vet ./...            # static analysis
```

**Logging (P5-T5):** uses `log/slog`. Text to stderr by default; `GOLLAMA_LOG_FORMAT=json` for JSON; `GOLLAMA_LOG_LEVEL=debug|info|warn|error` to override (default `info`).

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
│   │   ├── coordinator.go     # Model-switch coordinator: serializes switches, coalesces same-model (P3-T1)
│   │   ├── service.go         # FindGollamaUnit(): shared systemd unit-location lookup (P5-T6)
│   │   ├── manager_test.go
│   │   ├── coordinator_test.go
│   │   └── manager_start_test.go
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
│       ├── ui.go              # Assembles Page from embedded web/ files (P5-T4)
│       └── web/
│           ├── index.html     # HTML shell with __GOLLAMA_CSS__ / __GOLLAMA_JS__ placeholders
│           ├── app.css        # All CSS
│           ├── app.js         # All JS
│           └── testdata/
│               └── page_reference.html  # Byte-identity reference (TestPageMatchesReference)
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
    APIToken      string              `json:"api_token"` // guards /api/v1/* + /v1/* when non-empty (no omitempty on purpose)
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

**Coordinator** (`pkg/manager/coordinator.go`, P3-T1):
```go
type SwitchRequest struct {
    Model, Profile, BinaryPath string
    Mode         SwitchMode    // SwitchText | SwitchImage | SwitchExplicit
    Port         int           // explicit mode only; 0 = auto-assign
    Flags        []string
    ReplaceFlags bool
    Env          map[string]string
    WaitReady    bool          // hold the switch until the model serves
    Heartbeat    func()
    ShouldAbort  func() bool   // e.g. client disconnect
}
```
One switch at a time (`switchMu`); same-model callers coalesce via an in-flight map (`mu`). Sentinels: `ErrBusy` (concrete `*BusyError` names the busy model → 503 + `Retry-After: 30`), `ErrSwitchAborted`, `ErrModelExited`, `ErrNotReady`. Entry points: proxy, warmup, instances POST, image generations, CLI chat/run.

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
- **API token (v3.8.0+):** every `/api/v1/*` and `/v1/*` call needs `?token=<api-token>` or `Authorization: Bearer <api-token>` while a token is set (it is, by default). Token lives in Web UI → Settings or `~/.gollama/config.json` (`api_token`).
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
  "default_flags": ["--host","127.0.0.1","--ctx-size","2048","--flash-attn","on","--temp","0.7"],
  "proxy_defaults": [],
  "profiles": {},
  "idle_ttl": 30,
  "api_token": "<64-hex, generated on first start>"
}
```

- `default_flags`: used by Quick Launch form and CLI `gollama run`/`chat`
- `proxy_defaults`: used by API auto-launch; falls back to `default_flags` if empty
- `profiles`: map of named Model Profiles, each bundling model, flags, env vars, type, strip_reasoning
- `idle_ttl`: auto-stop idle instances after N minutes (0=disable)
- `api_token`: shared-secret for all API routes (v3.8.0). Generated once by `model.EnsureAPIToken()` on first start (printed once by `serve`, visible in Web UI → Settings). A `~/.gollama/api-token-generated` marker distinguishes "never had a token" (generate) from "user cleared it" (auth disabled — respected, not regenerated). Empty token = unauthenticated (startup warning).

## Security Model (v3.8.0)

- **Bind:** `gollama serve` listens on `127.0.0.1` by default. `--listen 0.0.0.0` (or `GOLLAMA_LISTEN=0.0.0.0` in systemd) exposes it on the LAN.
- **Auth:** `requireAuth` middleware wraps every `/api/v1/*` and `/v1/*` route. Accepts `Authorization: Bearer <token>` or `?token=<token>`, constant-time compared. Token is read from disk per request (regenerations apply without restart). UI assets (`/`, `/logo.svg`) stay open; the UI routes all fetches through `apiFetch()` (attaches the localStorage token) and shows a token-entry gate on 401.
- **Traversal:** chat ids must match `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$` (400 otherwise); `chatPath` maps invalid ids to an inert path; model delete requires `filepath.Rel(modelsDir, path)` to stay inside the models dir.
- **Self-update:** verifies the downloaded binary against the release `checksums.txt` (`verifyChecksum`); mismatch aborts, missing file warns.
- **Dep installs:** `checkDependencies` prompts `[y/N]` (TTY) or prints the apt command (non-interactive) unless `GOLLAMA_AUTO_INSTALL_DEPS=1`.
- **systemd:** no hardcoded `HOME`; non-root installs a user unit (`~/.config/systemd/user/gollama.service` + `systemctl --user`); `install-service`, the wizard, `restart`, and `handleRestart` all share the unit-location logic.

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
| `/api/v1/warmup` | POST | Pre-warm a model in background (`{"profile":...}` or `{"model":...}`) |
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

### Seamless Cold Start (v3.5.0)

- `waitForInstanceReady` returns an error (including the instance log tail) instead of silently timing out after 60s. Deadline defaults to 5 min, override with `GOLLAMA_MODEL_LOAD_TIMEOUT` (seconds).
- Streaming (`stream: true`) requests get SSE headers immediately plus comment heartbeats (`: model loading...`) every ~10s while the model loads, so the connection stays alive past client/proxy idle timeouts during long cold starts.
- If upstream answers 503 "loading model" after `/health` passes, the proxy retries with heartbeats until the load deadline.
- Start failures surface as 500 JSON (non-stream) or an OpenAI-style SSE error event (stream), including the last lines of the instance log — and the wait fails fast when the model process exits before serving.
- A failed start doesn't poison the port: `manager.Start`/`StartImage` reuse stale (non-running) slot entries instead of returning "port already in use".

### Pre-Warm (v3.6.0)

`POST /api/v1/warmup` — start a model (text or image profile) in the background before it's needed, so agents don't sit through a cold start. Body: `{"profile": "name"}` or `{"model": "model-id"}` (profile auto-detected by model name, same rules as the chat proxy).

- Returns immediately: `200 {"status":"starting","port":N,...}` once the process has spawned, or `200 {"status":"running",...}` if the model is already up (idempotent — no restart).
- Text warmup applies the same switching rules as the chat path (`switchToModel`: stops other text instances, image grace period, 2s GPU cooldown). Image warmup applies the image-generation rule: defers with `503 + Retry-After: 30` while a text model has been active <30s.
- Readiness: poll `GET /api/v1/instances` for the `ready` flag, or just fire the real request — v3.5.0 guarantees it blocks (with heartbeats) until the model is up. Warmup therefore never costs anything; worst case it saves nothing.
- 404 for unknown model/profile, 400 for an empty body.

Usage (e.g. re-warm the text model after image gen in the blog pipeline):
```bash
curl -s -X POST "http://192.168.1.36:9080/api/v1/warmup?token=<api-token>" -d '{"profile":"deepseek-v4-flash"}'
```

## CLI Commands (`main.go`)

| Command | Args | Description |
|---------|------|-------------|
| `gollama` | — | First-run wizard or usage |
| `serve` | `[port] [--listen ADDR]` | Web UI + API server (default :9080, binds 127.0.0.1; `--listen 0.0.0.0` / `GOLLAMA_LISTEN` for LAN) |
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

## UI Architecture (`pkg/ui/web/`)

The web app lives in `pkg/ui/web/`: `index.html` (HTML shell), `app.css` (CSS), `app.js` (JS). `pkg/ui/ui.go` embeds them via `go:embed` and assembles `ui.Page` by inlining the CSS/JS into the `__GOLLAMA_CSS__` / `__GOLLAMA_JS__` placeholders (P5-T4). No framework, no build step; the served page is a single self-contained HTML document (inline `<style>` + `<script>`).

### Views (JS SPA with `switchView()`):
- **Dashboard** (`#view-dashboard`): Quick Launch form + Running Instances grid; polls `loadInstances()` every 5s while visible
- **Models** (`#view-models`): Model list with badges + Pull model with HF search + details modal
- **Chat** (`#view-chat`): SSE streaming chat with reasoning display + history management
- **Image** (`#view-image`): Image generation playground (profile select, prompt, params, results, history)
- **Settings** (`#view-settings`): Version info + API token (copy/regenerate/disable) + idle TTL + default flags + API defaults + Model Profiles + restart

### JS Patterns:
- Navigation: `switchView(name)` — hides all `.view`, shows target, calls `loadInstances()`/`loadModels()`/`loadChats()`/`loadSettings()` depending on view; starts/stops the 5s dashboard poll. Nav buttons carry a `data-view` attribute (P5-T4); `switchView` finds the active button via `.nav-item[data-view="..."]`
- Sidebar: Collapsible (60px), expanded (220px), state in localStorage
- Theme: Dark/light toggle, persisted in localStorage, CSS custom properties; syncs `meta[theme-color]`
- Faceplate: top strip (`#faceLed`, `#faceInst`, `#faceTps`, `#faceModels`, `#faceVersion`, `#faceClock`) updated by `updateFaceplate()` from the instance list + a 1s clock
- Flags: Searchable dropdown of 150+ llama-server flags with auto-complete, hints, standalone flag detection
- Dialogs: Model details, chat history, logs modal, pull progress bar; Escape closes any open modal/lightbox
- Clipboard: `fallbackCopy()` falls back to `execCommand('copy')` when `navigator.clipboard` is unavailable (LAN/non-HTTPS access)

### CSS Design System (v3.7.0 — instrument panel / GPU-rack console):
- Direction A from the 2026-08 UI refresh: dark charcoal (`--bg #0f1115`), no glass/glow, subtle 28px grid texture on `.content`
- Typography: Archivo (body/UI) + JetBrains Mono (labels, values, data) — Google Fonts; mono uppercase micro-labels throughout
- Sharp 3–4px radii, 1px hairline borders, status LED semantics: green = running, amber = starting/warning, red = error, dim = off
- Flat accent `--accent #45d483` (buttons) instead of the old teal→blue gradient; gradients kept only as unused `--accent-gradient` fallback
- Badges are square with a 4px LED dot (`::before`); instance cards are "rack units" with a 2px status stripe (`.inst-card.starting/.stopped/.error`)
- Both dark and light themes via `.light` on `<html>` + `<body>` (JS sets both); keep that pairing
- Responsive breakpoint at 768px (force-collapsed sidebar, single column, faceplate trimmed)
- Nav buttons carry a `data-view` attribute; `switchView` matches on it (P5-T4). Keep the attribute in sync with the view id when adding a view.

## Code Conventions

1. **Zero external dependencies** — stdlib only. No gorilla/mux, no cobra, no chi, no templ.
2. **Web UI in `pkg/ui/web/`** — HTML/CSS/JS live in separate files (`index.html`, `app.css`, `app.js`), assembled into `ui.Page` at load via `go:embed` (P5-T4). No build step; the served page is inlined (CSS in `<style>`, JS in `<script>`), so edits to `web/` files require a rebuild. `TestPageMatchesReference` guards byte-identity against `web/testdata/page_reference.html` — if you intentionally change the served bytes, regenerate that reference (see its test comment).
3. **Model profiles** — stored in `config.json` under `profiles` key. Always named "Model Profiles" in the UI.
4. **Config access** — always call `model.LoadConfig()` (reads from disk every time). Save with `model.SaveConfig()`.
5. **Instance management** — through `manager.Manager` (mutex-protected map of ports). Ports auto-assigned from 8080+.
6. **SSE streaming** — used for chat responses, model pull progress. `Transfer-Encoding: chunked` + `text/event-stream`.
7. **Error responses** — JSON with `{ "error": "message" }` format. Use `jsonResponse()` / `jsonError()` helpers.

## Testing

```bash
go test ./...            # everything (server proxy tests spawn dummy llama-server, ~20-35s; -race on pkg/server for concurrency)
go test ./pkg/model      # FormatSize, DefaultConfig, API token, chat-id validation, Presets, GGUF metadata
go test ./pkg/manager    # NewManager, List, Stop, UpdateTokens
go test ./pkg/llama      # self-update checksum verification (local HTTP server, no network)
```

Tests use temp dirs (`t.Setenv("HOME", t.TempDir())`, no real config). No test fixtures on disk.

## Important Files

- `pkg/ui/web/{index.html,app.css,app.js}` — the web UI (HTML/CSS/JS). All UI changes happen here. `app.js` and `app.css` use `// ── Section ──` comments to navigate.
- `pkg/server/server.go` — all HTTP handlers, route registration, OpenAI proxy logic.
- `pkg/manager/manager.go` — instance lifecycle, metrics polling, idle timeout.
- `pkg/model/model.go` — config management, model index, HF pull/search, flag handling.
- `main.go` — CLI dispatch (switch on os.Args[1]), version injected at build time.
