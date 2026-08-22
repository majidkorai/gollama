# Gollama Robustness Plan

**Status:** Phases 0–4 complete (`v3.8.0` security, `v3.9.0` correctness, `v4.0.0` concurrency & lifecycle, `v4.1.0` robustness). **Next: Phase 5 (→ v4.2.0, refactor). `v4.1.0` is tagged + pushed + deployed to the gollama VM (2026-08-22 — see Phase 4 deploy notes).**

## ▶ Resume here (read this first)

**Where we are:** Phase 0 (test safety net) and Phase 1 (security) are done, and **Phase 1 is deployed** (v3.8.0 live on the gollama VM 2026-08-20, hermes pipeline updated with the token — see deploy notes below). Phase 1 shipped: loopback-default bind, shared-secret API token, chat-id + model-delete traversal fixes, self-update checksum verification, no silent package installs, systemd unit fixes, bounded API client. Phase 2 is complete: **P2-T1 (merge_reasoning) + P2-T2 (`gollama run` no-op, incl. a pgrep→ps orphan-recovery fix) + P2-T3 (clean shutdown) + P2-T4 (one readiness deadline) + P2-T5 (deterministic fuzzy model match) + P2-T6 (Windows orphan recovery) + P2-T7 (per-GPU utilization + real CPU) all done** — smoke tests passed 2026-08-21 (simultaneous serve+run instances, clean SIGTERM shutdown of both, no leftover processes) and **v3.9.0 is tagged + pushed** (GitHub Actions release building on the self-hosted runner). Phase 3 is complete: **P3-T1 (model-switch coordinator) + P3-T2 (no lock held while sleeping) + P3-T3 (orphan-readiness confirmation) + P3-T4 (sentinel errors) + P3-T5 (streaming client-disconnect) all done** — smoke test passed 2026-08-21 (two-model switch: one start each, one eviction, 2s GPU cooldown, both served, no stop/start loop; SIGTERM → zero leftover processes) and **v4.0.0 is tagged + pushed + deployed** to the gollama VM (2026-08-21, v3.9.0 backed up as `gollama.bak-v390-*`; a live hermes request verified the coordinator cold-start path on the new binary). **Phase 4 is complete (→ v4.1.0, tagged 2026-08-22):** P4-T1 (safe downloads: `.part` files + resume + size verification), P4-T2 (cheap model list: throttled scan + `?refresh=1` + skip complete entries), P4-T3 (log hygiene: tail reads + 10 MB rotation), P4-T4 (error propagation on config/index writes), P4-T5 (TTY-gated download progress). Real resume smoke passed 2026-08-22 (killed a live HF pull mid-download, re-ran → "resuming from N B", `.part` continued). **Phase 4 is tagged + pushed + deployed** to the gollama VM (2026-08-22 — v4.0.0 backed up as `gollama.bak-v400-202608220500`; restart cleanly killed the old model instance with no orphans and gollama auto-relaunched it ready — see Phase 4 deploy notes). **Next: Phase 5 (→ v4.2.0, refactor).**

**How to resume:**
1. Read this file top-to-bottom (phase checkboxes are the source of truth).
2. Confirm green baseline: `go build ./... && go vet ./... && go test ./...` (add `-race` for the server package; proxy tests spawn a dummy `llama-server` shell script and take ~20–35s).
3. Start Phase 5, task P5-T1 (all five Phase 4 tasks are done, smoke test passed, and `v4.1.0` is tagged + pushed + deployed). The P2-T3 stop-check is **done** (2026-08-22 v4.1.0 deploy: restart cleanly killed the running model instance, `pgrep llama-server` showed no orphan of the old PID, gollama auto-relaunched it ready). Phase 5 is a refactor release (no behavior change intended) — work tasks in order; each phase ends with a tag.

**Key context for the next session:**
- Architecture review (what/where all the issues are) lives in the conversation that produced this plan; the plan itself is self-contained for execution. The full issue list maps 1:1 to tasks.
- **Test conventions** (follow them for new tests): `t.Setenv("HOME", t.TempDir())` for isolation; dummy `llama-server` = `#!/bin/sh\nsleep 30` script in `$HOME/.gollama/bin/`; fixed ports for warmup tests, ephemeral reserved ports for proxy tests (`startProxyFixture` in `pkg/server/proxy_test.go`); `fakeUpstream` (`pkg/server/fakeupstream_test.go`) simulates the model. **Hermetic managers (P3-T1):** test managers must not run the orphan `ps` scan — use `manager.NewManagerNoRecovery()` (server tests) or raw `&Manager{...}` literals (manager tests), or one test binary's dummy processes get adopted by another binary's manager and stopped mid-switch.
- **Session tooling quirk (important!):** the file-writing pipeline strips angle-bracket pairs from written content. In `pkg/server/transforms_test.go` the think-tag constants are therefore built from hex escapes (`0x3c`/`0x3e`) — keep it that way, and after writing any file containing `think`-style literals, verify on disk with `sed -n`/`grep` before compiling.
- **Behavior quirks pinned by Phase 0 tests** (do not "fix" incidentally — they are scoped to later phases):
  - `/api/v1/chat` drops the `[DONE]` marker → fixed in P5-T1 (UI already tolerates `[DONE]`, `ui.go:2074`).
  - `ScanModels` normalizes underscores→hyphens before the quant-strip regex → scanned short names keep the quant suffix → fixed in P5-T2.
  - ~~`FindInstanceByModel` fuzzy match is map-iteration-nondeterministic → fixed in P2-T5 (test asserts "a valid candidate", not which one).~~ **Fixed (P2-T5):** tiered scoring, lowest-port tie-break, tests assert the exact winner.
  - ~~`merge_reasoning` is dead code (toggle does nothing)~~ → **wired in P2-T1 (done)** (tests in `transforms_test.go` pin intended behavior; note: `mergeReasoningContent` appends reasoning AFTER existing content in the same chunk, "c"+"r"→"cr" — preserved).
  - `ProfileFlags` keeps both `--verbose` and `--no-verbose` (standalone flags aren't key-overridden; llama-server takes the last) → snapshot in `flags_test.go`, replaced by typed flag model in P5-T3.

**Phase 1 deploy (DONE 2026-08-20):** v3.8.0 is live on the gollama VM.
- VM 192.168.1.36: `/usr/local/bin/gollama` is v3.8.0 (v3.7.3 backed up as `gollama.bak-v373-*`); `gollama.service` runs `serve --listen 0.0.0.0`.
- Token: a 64-hex `api_token` was pre-written into `/root/.gollama/config.json` **before** the binary swap (no 401 window for the hermes pipeline); the old config is backed up as `config.json.bak-v380-*`. The token is visible in Web UI → Settings and in hermes as `GOLLAMA_API_KEY` in `~/.hermes/.env`.
- hermes 192.168.1.17 patched (backups `*.bak-v380-*`):
  - `ghost_post_creator.js` — `gollamaToken` in CONFIG + `Authorization: Bearer` header on the image-gen call.
  - `post_watcher.sh` — warmup curl now sends `?token=`.
  - `~/.hermes/config.yaml` — the agent's gollama endpoint is a **named custom provider** (`providers.gollama` with `api:` + `key_env: GOLLAMA_API_KEY`, `model.provider: gollama`). Note: a `key_env` in the `model:` block is NOT honored by hermes (v0.20.0) — inline `provider: custom` resolves to a `no-key-required` placeholder and 401s; the named-provider block is the supported path.
  - **Cron jobs** (`~/.hermes/cron/jobs.json`) pin `provider` per job and that value overrides the model block — both jobs (`telegram-top-stories`, `Daily Blog Post`) had `provider: custom` stored and kept 401ing after the config change. Fixed with `hermes cron edit <id> --provider gollama`, then gateway restart. (If a job ever 401s again, check its stored `provider` first: `hermes cron list` / `jobs.json`.)
- Smoke test passed: no-token → 401, Bearer / `?token=` → 200, UI open on LAN, `hermes chat` round-trip OK, image gen (flux-klein) 200 with token, text model re-warmed after image gen, cron `telegram-top-stories` completed end-to-end after re-pointing both jobs at the `gollama` provider.

**Phase 2 deploy (v3.9.0 — DONE 2026-08-21 13:19 UTC):**
- `gh` had no valid token on this machine, so the binary was built locally from the tag with the exact CI command (`GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v3.9.0" .`) and scp'd to the VM (sha256 `b398ba84…` verified on both ends). Backed up as `gollama.bak-v390-20260821`, installed, `systemctl restart gollama`. No config/token changes needed.
- **Verified on the VM:** `GET /api/v1/version` → `v3.9.0`; live instance JSON carries `gpu_util_per_gpu: [86, 86]` (per-GPU split on the 2×3090) and instantaneous `cpu_percent` (P2-T7); the pre-restart v3.8.0 instance was cleaned up on restart with no orphan left behind (`pgrep llama-server` shows only the new instance); Qwen3.8-27B auto-restarted on demand (port 8081, ready in ~25s) — auto-launch + deterministic matching unaffected.
- **Deferred (model was actively in use at deploy time, GPU 86%):** the explicit `systemctl stop gollama` → `pgrep llama-server` empty check (P2-T3). It is covered by the local smoke test (SIGTERM → `stopped 1 instance(s)`, zero leftover processes); run it whenever the box is idle to confirm on the VM.
- hermes pipeline: no changes required (503/Retry-After contract unchanged; token unchanged).

**Phase 3 smoke + release (v4.0.0 — smoke DONE 2026-08-21 local, VM deploy DONE 2026-08-21 16:36 UTC):**
- Smoke (macOS, dummy HTTP-serving `llama-server` with a 3s simulated load, two indexed models): `curl /v1/chat/completions` for model A, then model B ~4.5s later (after A's response completes). Result: A started exactly once (port 8081) and served (200 in 4.5s); B evicted A exactly once + 2s GPU cooldown, started once (port 8082) and served (200 in 5.6s) — no stop/start loop in the log. Note on "simultaneous": a truly concurrent different-model request queues behind A's switch and evicts A the moment A's response starts, so the first caller's response and the second's eviction race by design (single text slot on the GPU); the staggered smoke is the no-thrash guarantee, and the queue/coalesce behavior itself is pinned by unit tests (`TestCoordinatorQueuesDifferentModels`, `TestCoordinatorCoalescesSameModel`).
- Shutdown: SIGTERM → `stopped 1 instance(s)`, zero leftover processes (gollama + model), port closed.
- **VM deploy (DONE 2026-08-21 16:36 UTC):** `gh` still has no token, so the binary was built locally from the tag (`git checkout v4.0.0 && GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v4.0.0" .`, sha256 `26f0c809…`), scp'd, sha256 re-verified on the VM, v3.9.0 backed up as `gollama.bak-v390-202608211636`, swapped + `systemctl restart gollama`. No config/token changes needed.
- **Verified on the VM:** `GET /api/v1/version` → `v4.0.0`; the old v3.9.0 instance (pid 569664) was gone after the restart (clean P2-T3 shutdown, no orphan); and within ~20s a **live hermes request** (54k-token prompt) exercised the new coordinator cold-start path end-to-end — fresh instance (pid 630926, `started_at` post-restart, not a recovery) loaded in ~25s and served (task 297, ~60 t/s, `gpu_util_per_gpu: [90, 89]`), so the P3-T1 proxy path + P2-T7 metrics are confirmed on v4.0.0 in production.
- **Deferred (model actively in use at deploy time — hermes session running, task 297):** the explicit `systemctl stop gollama` → `pgrep llama-server` empty + `nvidia-smi` VRAM idle check (P2-T3). Covered by the local smoke test (SIGTERM → zero leftover processes); run it whenever the box is idle.
- hermes pipeline: no changes required (503/Retry-After contract unchanged; token unchanged).

**Baseline:** v4.0.0 (Phase 3, deployed 2026-08-21), `main.go` + `pkg/{server,manager,model,llama,chat,ui}`, stdlib-only, `go build`/`go vet`/`go test` all green (plus `GOOS=linux` and `GOOS=windows` cross-builds for the platform-split files).

## Goals

1. **Secure** — gollama is an unauthenticated admin surface (start/stop models, read logs = read prompts, delete files, restart service). Make it safe to expose on a LAN.
2. **Honest** — fix features that silently lie: dead `merge_reasoning`, `gollama run` no-op, orphaned processes on shutdown, UI "starting" forever on long cold starts.
3. **Safe under concurrency** — two agents hitting different models at once can currently thrash the GPU (no lock around model switching); the manager holds its mutex while sleeping.
4. **Durable** — resumable downloads, bounded logs, real error propagation, one source of truth per concern (GGUF parsing, quant regexes, flag handling, SSE loop).
5. **Maintainable** — split `ui.go`, typed flag model, structured logging, meaningful test coverage of the proxy.

## Principles

- **Zero external dependencies** stays sacred (stdlib only).
- **One phase = one release tag** so any regression is bisectable.
- **Backward-compatible where possible**; where a change is breaking (default bind address), provide a flag + env override and document the deploy step.
- Every phase ends with `go build ./... && go vet ./... && go test ./...` green and a manual smoke test on the gollama VM (192.168.1.36).

## Release & deploy strategy

| Phase | Tag | Notes |
|-------|-----|-------|
| 0 | (no tag) | tests only, no behavior change |
| 1 | `v3.8.0` | security |
| 2 | `v3.9.0` | correctness |
| 3 | `v4.0.0` | concurrency + lifecycle (major: shutdown now kills instances) |
| 4 | `v4.1.0` | robustness |
| 5 | `v4.2.0` | refactor (no behavior change intended) |

Deploy per phase: `gollama self-update <tag>` on the VM (or CI artifact), then `gollama restart`.
**Phase 1 deploy requires a manual step** (see P1-T7): update `gollama.service` to pass `--listen 0.0.0.0` (we access the UI over LAN) and note the generated API token for the hermes blog pipeline.

---

## Phase 0 — Safety net (tests before refactors)

No behavior changes. This makes Phases 3–5 safe.

- [x] **P0-T1: Fake upstream llama-server test helper** (`pkg/server/fakeupstream_test.go`)
  - `httptest` server speaking enough OpenAI SSE: streaming chunks, `finish_reason`, trailing `usage` chunk, 503 "loading model" N times before 200, 500 grammar errors N times before 200.
- [x] **P0-T2: Proxy tests** (`pkg/server/proxy_test.go`, unix-only — dummy shell-script llama-server)
  - Non-stream: usage tokens + tps recorded, response passthrough.
  - Stream: initial `: model loading...` comment, `data: [DONE]` exactly once, trailing usage chunk forwarded (regression test for the 2026-08-15 fix).
  - 503-retry loop: upstream returns 503 "loading" twice then 200 → client gets 200 (stream and non-stream).
  - Grammar-error retry with tools: 500 first, 200 on retry.
  - 500 passthrough when no tools.
  - **Bonus finding pinned:** `handleChat` (/api/v1/chat) does NOT forward `[DONE]` (sentDone set before the marker is written) — divergence from the OpenAI proxy documented in `TestHandleChatStream`; P5-T1 merges the loops.
- [x] **P0-T3: Table-driven tests for reasoning transforms** (`pkg/server/transforms_test.go`)
  - `extractThinkStream`: complete/split/start/end think blocks, lone close tag passthrough.
  - `stripContentThinkTags`, `stripReasoningContent`, `convertCompleteThink`, `mergeReasoningContent` (uncalled dead code — tests pin intended behavior for P2-T1).
  - `sanitizeSchemaPatterns` / `stripAdditionalProperties` / `simplifyToolSchemas` on nested schemas.
  - Note: think-tag literals in the test are built from hex escapes (0x3c/0x3e) because session tooling strips angle-bracket pairs from written file content.
- [x] **P0-T4: Flag-merge tests** (`pkg/model/flags_test.go`)
  - `sanitizeFlags`, `ProfileFlags` (override/extend/standalone-both-survive quirk), `ProxyFlags` fallback, `LoadConfig` sanitization.
- [x] **P0-T5: Chat-history tests** (`pkg/model/chat_history_test.go`)
  - Round-trip, auto-title, truncation (byte-based: 60/80 bytes + 3-byte ellipsis), corrupt-file skipping, delete.
- [x] **P0-T6: Model index tests** (`pkg/model/index_test.go`)
  - `ScanModels` idempotence, split-file indexing (part 1 only, size = sum), missing-blob filtering, `ResolveModelBlob` exact/path/missing, fuzzy-match nondeterminism documented (fixed in P2-T5 — `TestResolveModelBlobFuzzyDeterministic` now asserts the exact winner).
  - **Bonus finding pinned:** scan path replaces underscores→hyphens *before* the quant-strip regex (which expects underscores), so scanned short names keep the quant suffix.

**Exit:** `go test ./...` green, proxy + transforms covered.

---

## Phase 1 — Security hardening → `v3.8.0`

- [x] **P1-T1: Bind to loopback by default** (`pkg/server/server.go`, `main.go` `serve`)
  - `Server` gains `listen string`; `gollama serve [port]` binds `127.0.0.1` by default.
  - Opt out: `--listen 0.0.0.0` flag (and `GOLLAMA_LISTEN` env for the systemd unit).
  - Updated `DefaultConfig` default flags: `--host 127.0.0.1` (existing on-disk configs keep their values — no migration).
  - Updated `printUsage` + README; the startup banner prints the actual bind address.
- [x] **P1-T2: Shared-secret API token**
  - `Config` gains `APIToken string` (no `omitempty` — a cleared token persists as `""`). `model.EnsureAPIToken()` generates 32 random bytes (hex) on first start and the CLI prints it once: `gollama: API token generated — <token> (shown again in Web UI → Settings)`.
    - **Design note (marker file):** `EnsureAPIToken` uses a `~/.gollama/api-token-generated` marker to distinguish "never had a token" (generate) from "user cleared it" (disabled — respect). Plain key-presence in config.json can't tell them apart because `LoadConfig` (called by `NewManager`/`EnsureLlamaServer` before serve) writes `api_token: ""` on fresh installs.
  - Middleware in `registerRoutes`: if `cfg.APIToken != ""`, require `Authorization: Bearer <token>` **or** `?token=<token>` (query form for curl/cron simplicity) on all `/api/v1/*` and `/v1/*` routes (constant-time compare). UI assets (`/`, `/logo.svg`) stay open — the UI is a viewer; all its fetches go through a central `apiFetch()` that attaches the stored token, and a 401 raises a token-entry gate.
  - UI: Settings shows the token with Copy / Regenerate / Disable-auth buttons; stores it in `localStorage` (kept in sync from the config GET); `POST /api/v1/config/token` (`regenerate`/`clear`).
  - **Backward compat:** empty token = auth disabled (warns at startup: "no API token — gollama is unauthenticated").
  - All routes protected uniformly, including `handleRestart` and `handleModelDelete`.
  - Updated AGENTS.md "Caller guidance" with token usage.
- [x] **P1-T3: Fix chat-history path traversal** (`pkg/server/server.go`, `pkg/model/chat_history.go`)
  - `model.ValidChatID`: `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$` → 400 on mismatch for GET/PUT/DELETE `/api/v1/chats/{id}`.
  - Defense in depth: `chatPath` maps any invalid id to an inert `.invalid` path.
- [x] **P1-T4: Fix model-delete path check** (`pkg/server/server.go`)
  - `filepath.Rel(modelsDir, absPath)` instead of `HasPrefix` (which admitted sibling dirs like `models-evil` sharing the path prefix).
- [x] **P1-T5: Verify `self-update` against release checksums** (`pkg/llama/binary.go`)
  - After download, fetches `checksums.txt` from the release, matches the asset name, verifies sha256, aborts on mismatch. Missing/unlisted checksum (older releases) → warning, not failure. `expectedChecksum` is a pure function (unit-tested offline; `verifyChecksum` tested against a local server via `checksumURLBase`).
- [x] **P1-T6: No silent package installs** (`pkg/llama/binary.go`)
  - `checkDependencies` asks `Install missing packages: …? [y/N]` unless `GOLLAMA_AUTO_INSTALL_DEPS=1`. Non-interactive (no TTY, stdlib `os.ModeCharDevice` check) → warn + print the apt command instead of installing.
- [x] **P1-T7: systemd unit fixes** (`main.go` — deduped into `installSystemdService`, used by `install-service` and the wizard)
  - Dropped hardcoded `Environment=HOME=/root` (HOME inherited).
  - uid != 0 → **user** unit at `~/.config/systemd/user/gollama.service` via `systemctl --user` (+ `loginctl enable-linger` hint). `restart` and `handleRestart` recognize both unit locations.
  - Phase-1 deploy documented: `gollama.service` needs `ExecStart=… serve --listen 0.0.0.0` for LAN access — the token is the gate.
- [x] **P1-T8: HTTP client timeout for API calls** (`pkg/model/model.go`)
  - `model.HTTPClient` stays timeout-free (long downloads); `model.APIClient` (`Timeout: 30s`) used for `SearchModels`, `SearchImageModels`, `ListRepoGGUFFiles`, `pullModelInternal` metadata fetch, `GetReleaseData`, `SelfUpdate` release listing + checksum fetch.

**Exit:** unauthenticated LAN access closed (token + loopback default), traversal closed, self-update verified.
**Smoke test (done on macOS, 2026-08-20):** fresh `serve` → token generated + printed once, `curl` without token → 401, Bearer + `?token=` → 200, UI open; upgraded config (no `api_token` key) → token added, existing flags preserved; `--listen 0.0.0.0` → LAN IP banner + reachable.
**Deployed to gollama VM 192.168.1.36 (2026-08-20):** token pre-written to config before the binary swap (no 401 window), unit now `serve --listen 0.0.0.0`, hermes consumers patched (see Phase 1 deploy notes above). VM smoke: no-token → 401, Bearer/`?token=` → 200, UI open on LAN, `hermes chat` round-trip OK, flux-klein image gen 200 with token, text model re-warmed after image gen.

---

## Phase 2 — Correctness → `v3.9.0`

- [x] **P2-T1: Make `merge_reasoning` actually work** (`pkg/server/server.go`)
  - Stream path: when `shouldMerge`, call `mergeReasoningContent(data)` instead of pass-through.
  - Non-stream path: merge branch runs `convertCompleteThink` then `mergeReasoningContent` (merge takes precedence over strip on both paths, matching the stream path's check order).
  - `mergeReasoningContent` generalized to handle both `delta` (stream) and `message` (non-stream) shapes, mirroring `stripReasoningContent`.
  - Pinned tests updated: the dead-code NOTE in `transforms_test.go` now says wired; new `TestMergeReasoningContentMessage` covers the message shape; new e2e `TestProxyStreamMergeReasoning` / `TestProxyNonStreamMergeReasoning` (with a `nonStream` override added to `fakeUpstream`) assert reasoning never leaks and lands in `content`.
- [x] **P2-T2: `gollama run` no longer a silent no-op** (`main.go`, `pkg/manager/manager.go`)
  - Removed the `len(mgr.List()) == 0` guard; `run` always starts with port 0 (auto-assign). An explicit `--port` in use → clear "port N is already in use" error from `Start` (unit-tested in `TestStartRejectsRunningSlot`).
  - **Recovery fix found by the smoke test:** the old `recoverOrphans` used `pgrep -a llama-server`, which matches the *executable name* — for a shebang script that is the interpreter (`/bin/sh`), so on macOS orphans were never recovered and a second `run` collided on the same port. Now scans `ps -eo pid,args` full command lines (`isLlamaServerCommandLine`: llama-server basename arg + gollama's `--host` flag), portable across macOS/Linux. Tests: `TestIsLlamaServerCommandLine`, `TestRecoverOrphansFindsScriptInstance`, `TestStartAutoAssignsPastRunningInstance`.
  - Smoke test: `gollama run` ×2 with a live instance → second run logs `recovered orphan instance: port=8081` and starts on 8082.
- [x] **P2-T3: Clean shutdown** (`main.go` `serve`, `pkg/manager`)
  - `Instance` now holds `Cmd *exec.Cmd` + `WaitDone chan struct{}` (both `json:"-"`); the `cmd.Wait()` goroutines in `Start`/`StartImage` close it when the process is fully reaped.
  - `Stop` rewritten: SIGINT → wait ≤500ms on `WaitDone` (or a signal-0 probe for recovered orphans — never a blind `Kill` that could hit a recycled PID) → SIGKILL only if still alive → wait for the reap. The mutex is no longer held while waiting.
  - New `Manager.StopAll() []int` — stops all instances in parallel, returns stopped ports.
  - `serve` on SIGINT/SIGTERM: cancel ctx → `StopAll` bounded to 5s ("stopped N instance(s)" or a warning) → exit.
  - Tests: `TestStopEscalatesToKillWhenSIGINTIgnored` (SIGINT-ignoring script must be SIGKILL'ed after the grace period; waits for a marker file the script writes after its trap line, and skips if the sandbox blocks the child in dyld startup — observed in this session: macOS dyld `notifyDebuggerLoad` block under the DSH sandbox), `TestStopFastPathWhenAlreadyExited` (no 500ms wait for a dead process), `TestStopAll`.
  - Smoke: `gollama serve` with a live instance + SIGTERM → `shutting down... / instance on port 8081 did not exit after SIGINT — sending SIGKILL`, instance gone, clean exit.
- [x] **P2-T4: One readiness deadline** (`pkg/model/model.go`, `pkg/manager/manager.go`, `pkg/server/server.go`)
  - New `model.LoadTimeout()` — the single reader for `GOLLAMA_MODEL_LOAD_TIMEOUT` (seconds), default 5m, invalid/negative values fall back to the default.
  - Manager's text and image ready-poll goroutines use it (previously a hardcoded 120s that disagreed with the server's 5m default).
  - Server's three call sites (`waitForInstanceReady`, the streaming cold-start wait, the post-health 503-retry deadline) now call `model.LoadTimeout()`; the local `modelLoadTimeout` wrapper is gone.
  - Test moved: `TestModelLoadTimeoutEnv` (server) → `TestLoadTimeout` (pkg/model).
- [x] **P2-T5: Deterministic model matching** (`pkg/manager/manager.go`, `pkg/model/model.go`, `pkg/server/server.go`)
  - `FindInstanceByModel` rewritten: collects **all** candidates in ascending-port order and scores them exact-fold/same-blob (tier 0) > short-name-equal (1) > suffix (2) > substring (3); ties broken by lowest port; logs a warning when a substring-only match was used. Empty query returns nil.
  - `ResolveModelBlob` fuzzy tier made deterministic too: short-name candidates are scored over substring candidates, ties within a tier broken by lexicographic index name (the pinned `TestResolveModelBlobFuzzyDocumentsNondeterminism` is now `TestResolveModelBlobFuzzyDeterministic` asserting the exact winner).
  - Image auto-detect with empty `model` and multiple image profiles → 400 listing the profile names (sorted); a single image profile is still auto-selected; explicit `profile` bypasses the check.
  - Tests: `TestFindInstanceByModelDeterministic` (tier precedence, port tie-break over 50 iterations, stopped instances excluded, empty query), `TestImageAutoDetectMultipleProfiles400`.
- [x] **P2-T6: Windows orphan recovery** (`pkg/manager/manager.go`)
  - `recoverOrphansWindows` now delegates per-PID work to `recoverOrphanPidWindows`: if WMI fails to read the command line, it logs and **skips** the process instead of registering a phantom instance under a guessed port.
  - Parsing/registration extracted to pure `registerOrphanFromCommandLine(pid, cmdLine) bool`: only command lines carrying the gollama `--host` flag are recovered (the old code also registered non-gollama processes whose WMI read succeeded but lacked `--host` — same phantom bug), duplicate PIDs are rejected, and a missing `--port` still gets a guessed port.
  - Tests (cross-platform, the parser is pure): `TestRegisterOrphanFromCommandLine` (no--host skip, explicit port + basename model, duplicate PID, guessed port) and `TestRecoverOrphanPidWindowsSkipsSelf` (the test process itself must never register, on any OS).
- [x] **P2-T7: Per-GPU utilization + real CPU** (`pkg/manager/manager.go`, `pkg/manager/cpu_linux.go`, `pkg/manager/cpu_other.go`, `pkg/ui/ui.go`)
  - `queryGpuUtil` now returns `([]float64, bool)` — one entry per GPU from nvidia-smi (CSV parsing split out as pure `parseGpuUtilCSV`). `Instance.GpuUtil` = max across devices (was the average), new `GpuUtilPerGPU []float64` (`gpu_util_per_gpu`) keeps the breakdown; `setGpuUtil` applies a sample (max of the current sample, not a running high-water mark).
  - CPU: on Linux the metrics goroutine samples `/proc/<pid>/stat` utime+stime twice 1s apart (new `cpu_linux.go`: `procCPUTicks` + `cpuTicksPerSec` = 100 USER_HZ — stdlib has no Sysconf) → instantaneous %; `ps` %cpu (a lifetime average) is the fallback and the memory source everywhere (`applyCpuMetrics`). Non-Linux stubs in `cpu_other.go`.
  - UI: instance card badge shows the per-GPU split (`GPU0 92% / GPU1 3%`), falling back to `GPU 92%` for single-GPU hosts without the array.
  - Note: the `Instance` struct block got gofmt-aligned (it was pre-existing dirt in the same block as the new field); the rest of manager.go's gofmt dirt is untouched.
  - Tests: `TestParseProcStatTicks` (incl. comm with spaces/parens), `TestParseGpuUtilCSV`, `TestSetGpuUtil` — all cross-platform (pure parsers).

**Exit:** no feature silently lies; shutdown reclaims VRAM; matching is predictable.
**Smoke test:** kill gollama → `pgrep llama-server` empty; `gollama run` with an instance alive starts a second one; UI shows per-GPU split.

---

## Phase 3 — Concurrency & lifecycle → `v4.0.0` (major)

- [x] **P3-T1: Model-switch coordinator** (new `pkg/manager/coordinator.go` or in `server.go`) — **done**: `Coordinator.SwitchAndStart` with `switchMu` (serializes switches incl. the 60s image-grace wait and the readiness wait) + `mu`-guarded in-flight map `model → chan` (same-model callers coalesce and reuse the result; different models queue); modes `SwitchText`/`SwitchImage`/`SwitchExplicit`; `ErrBusy` (503 + Retry-After: 30) / `ErrSwitchAborted` / `ErrModelExited` / `ErrNotReady` sentinels; >90s waits log loudly. All five entry points rewired (proxy, warmup, instances POST, image generations, CLI chat/run); `Server.switchToModel` deleted. `Manager.ProcessExited` (WaitDone channel, else signal-0 with ESRCH/ErrProcessDone = gone, EPERM = alive) added for fail-fast readiness. `NewManagerNoRecovery()` added: server tests use it so one test binary's dummy llama-server is not adopted (and SIGKILL'ed) by another binary's orphan scan — that cross-binary interference was flaking the full suite.
  - All model switches go through one function: `coordinator.SwitchAndStart(model, profile, ...)`.
  - Internals: a `sync.Mutex` serializing switches + an in-flight map `model → chan *Instance` so concurrent requests for the *same* model coalesce (second caller waits on the channel, then reuses the instance) while different models queue.
  - Entry points rewired: `proxyToInstance`, `handleWarmup`, `handleInstances` (POST), `handleV1ImageGenerations` (text-stop part), CLI `chat`/`run`.
  - `switchToModel`'s up-to-62s image grace wait moves *inside* the lock (it already blocks; making it explicit is the point).
  - Guard: a switch that waits > N seconds (e.g. 90s) logs loudly.
- [x] **P3-T2: Don't hold `m.mu` while sleeping** (`pkg/manager/manager.go:310-334, 574-596, 690-709`) — **done**: candidate port picked + reserved in a new `reserved map[int]bool` under the lock, availability scan/retries (`allocateAvailablePort`/`claimPort`) and `cmd.Start` run outside it, commit + unreserve re-acquire it; ready-metrics snapshot (`queryProcessStats`/`queryGpuUtil`) sampled before the lock; `RecoverOrphans` ps/tasklist/wmic scan runs outside the lock (registration under it). Regression `TestStartParallelNoSamePortNoSerialization`: two parallel blocked Starts get unique ports in ~3.05s (one retry window) not ~6s (two).
  - Port allocation: pick candidate port under lock, **reserve** it in a `reserved map[int]bool`, release lock, do the availability scan/retries outside, then re-acquire and commit (or unreserve on failure).
  - `Start` process spawn: build everything (args, env, log file) under lock is fine, but move `cmd.Start` + the 2s GPU cooldown + health polling outside the lock.
  - `Stop`: signal the process outside the lock (the 500ms sleep is the worst offender); mutate state under lock.
  - Regression test: two `Start` calls in parallel don't serialize on the sleep (assert wall-clock < threshold), and never get the same port.
- [x] **P3-T3: Instance lifecycle state** — **done**: `Cmd`/`WaitDone` landed in P2-T3 (Stop/StopAll/wait-for-exit exact); orphan registration now kicks `confirmOrphanReady` (unix + Windows paths), a background /health poll that confirms readiness + snapshots the one-shot metrics, marks the instance stopped if the process dies first (`ProcessExited`), and downgrades `Ready` to false if it never serves within the load deadline (don't trust forever). Windows registration also sets `Ready: true` to match unix.
- [x] **P3-T4: Sentinel errors** (`pkg/model/model.go:1144` → `pkg/server/server.go:200,263`) — **done**: `model.ErrAlreadyExists` + `model.ErrNotFound` sentinels in pkg/model. `PullModel` returns `ErrAlreadyExists` (was `fmt.Errorf("already_exists")`); `ResolveModelBlob` wraps `ErrNotFound`; `handleModelPull`/`handleModelPullStream` use `errors.Is(err, model.ErrAlreadyExists)` (was `err.Error() == "already_exists"`); `handleModelDelete` now returns 404 for `ErrNotFound` and 500 for a real index-write failure (was a hardcoded 404 for every error).
- [x] **P3-T5: Request-level context on proxy paths** — **verified + pinned**: streaming already ties the upstream request to `r.Context()` via `context.WithCancel` (non-stream uses `WithTimeout(r.Context(), 10m)`); the read loop breaks on `r.Context().Err()`, an upstream read error, or a client-write error, and `defer resp.Body.Close()` + `defer cancel()` run on return. Regression `TestProxyStreamHonorsClientDisconnect`: the fake upstream streams slowly, the test drops the client mid-stream, and asserts the proxy returns promptly AND the upstream request context was canceled (no draining a model no one is listening to).

**Exit:** concurrent different-model requests queue instead of thrashing; UI polls are not blocked by port scans; `systemctl stop gollama` leaves no GPU memory behind. — **met 2026-08-21** (smoke above; queue/coalesce pinned by unit tests; the VM stop-check is deferred to the next idle window — model was in active use at deploy time).
**Smoke test (the important one) — DONE 2026-08-21:** two `curl /v1/chat/completions` for *different* models → first model starts, second queues + evicts + starts, both serve; exactly one start and one stop per model in the logs (no stop/start loop).

---

## Phase 4 — Robustness → `v4.1.0`

- [x] **P4-T1: Safe model downloads** — **done**: downloads write to `<name>.gguf.part` and `os.Rename` to the final name on success. Resume: an existing `.part` opens with `O_APPEND` + `Range: bytes=<done>-` (server returning 200 instead of 206 → truncate + restart from 0); a part already at the remote size is just renamed (no download); a part *larger* than the remote is stale → deleted + restarted. Size verified against the probed remote size before the rename (mismatch → delete `.part` + error). Size probe extracted to `probeRemoteFileSize` (known size → HEAD → ranged GET / Content-Range). Cancel path (ctx) deletes the `.part` (in-loop on copy error + deferred safety net). `ScanModels` now skips `*.part` and zero-byte files. `hfBaseURL` package var is the test seam for hermetic tests. Tests (`pkg/model/pull_test.go`, fake HF server with Range support): fresh download, resume-from-partial (asserts the Range header), stale-oversized-part restart, complete-part finalize-without-download, size-mismatch deletes part, mid-download cancel deletes part, ScanModels skip rules.
- [x] **P4-T2: Cheap `GET /api/v1/models`** — **done**: models-dir scan is throttled (package-level `scanInterval`, default 60s; `scanModelsMaybe` from `ListModels`, `ScanModelsForce` for explicit refresh; `ScanModels` stays the raw always-scan used by tests/CLI). `GET /api/v1/models?refresh=1` forces a scan and the UI ↻ button now sends it (`loadModels(refresh)`). `ListModels` skips `populateModelInfo` when the index entry already has arch/quant/ctx/short-name (no more GGUF header re-parse per list call; incomplete entries still get populated). `model.ResetScanThrottle()` is the test hook. Tests: `TestScanModelsMaybeThrottles` (throttle + force at model level), `TestModelsRefreshParam` (end-to-end: throttled list misses a new file, `?refresh=1` surfaces it).
- [x] **P4-T3: Log hygiene** — **done** (new `pkg/manager/logs.go`): `manager.TailLogFile(path, maxBytes)` — tail via `Seek(-maxBytes, io.SeekEnd)`, drops the partial first line, whole file when smaller; used by `handleInstanceLogs` (256 KB), `instanceLogTail` (64 KB), the instance-exit log diagnosis (64 KB), and `gollama logs` CLI (1 MB) — no more full-file `os.ReadFile`. Rotation: instances write through a `rotatingLogWriter` that renames `port-<port>.log` → `port-<port>.log.1` (one level, overwritten) once it exceeds `instanceLogMaxBytes` (10 MB); `prepareInstanceLog` at launch rotates an oversized previous log or removes small ones (old truncate-on-start behavior). `StopIdle` mtime heuristic documented: it only *extends* `LastActivity`, and since llama-server logs while serving, direct traffic to the llama-server port keeps instances alive. Tests: `TestTailLogFile`, `TestRotatingLogWriterRotates`, `TestPrepareInstanceLogMax`.
- [x] **P4-T4: Error propagation on state writes** — **done**: `SaveConfig`, `unsafeSaveIndex`, and `SaveIndex` now return errors (atomic tmp+rename unchanged); `UpdateIndex` propagates the write error (previously swallowed after a successful `fn`). API handlers surface 500 (`handleConfig` save, `handleConfigToken` regenerate/clear, image-profile install); internal callers log loudly (`LoadConfig` initial/default save, `EnsureAPIToken` token persist, `populateModelInfo` metadata, `doScanModels`, pull's stale-entry cleanup). A downloaded-but-unindexable model now returns an error instead of a silent success. Tests: `TestSaveConfigReturnsErrorOnWriteFailure`, `TestSaveIndexReturnsErrorOnWriteFailure` (block the tmp path, assert the error).
- [x] **P4-T5: Download progress for the systemd/CLI path** — **done**: `ProgressReader` gains a `TTY` field; the `\r` progress bar now renders only when the progress output is a terminal. `isTerminal(f)` (stdlib `f.Stat()` + `ModeCharDevice`) + `progressOutputTTY(w)` detect it (nil→stderr, `*os.File`→that file, other writers→non-TTY); `pullModelInternal` sets `TTY` per download. Non-TTY (systemd/cron/pipes) now get clean output — no `\r` spam — while the start/finish messages still print to stdout. **Note:** the TTY check is on the *progress output* (stderr), not stdout as the plan originally said — that's technically correct for the `gollama pull > out.log` / `2> err.log` edge cases. Tests: `TestProgressReaderTTYControlsRendering`, `TestIsTerminalAndProgressOutputTTY`.

**Exit:** interrupted 40 GB pull resumes; model list is O(index) not O(files × GGUF headers); logs bounded. — **met 2026-08-22.**
**Smoke + release (v4.1.0 — DONE 2026-08-22):**
- **Resume (real, on macOS):** `gollama pull hf.co/Qwen/Qwen2-0.5B-Instruct-GGUF:Q4_K_M` → killed mid-download (`.part` at 16 KB) → re-ran → log shows `resuming from 16375 B`, `.part` grew to 10 MB (Range honored by the real HF CDN). Resume also pinned hermetically by `TestPullModelResumesPartial` (asserts the Range header).
- **Model list:** throttled scan + skip-complete-entries covered by `TestScanModelsMaybeThrottles` + `TestModelsRefreshParam`; `?refresh=1` wired to the UI ↻ button.
- **Logs:** tail reads + 10 MB rotation covered by `TestTailLogFile` / `TestRotatingLogWriterRotates` / `TestPrepareInstanceLogMax`.
- **Errors:** `SaveConfig`/`SaveIndex`/`UpdateIndex` return errors; 500 on API save failure; covered by `TestSaveConfigReturnsErrorOnWriteFailure` / `TestSaveIndexReturnsErrorOnWriteFailure`.
- **Progress:** TTY-gated `\r` bar covered by `TestProgressReaderTTYControlsRendering` / `TestIsTerminalAndProgressOutputTTY`.
- Full suite + `-race` + linux/windows/darwin cross-builds green. **`v4.1.0` tagged + pushed + deployed 2026-08-22.**
- **VM deploy (2026-08-22):** `gh` token still invalid, so the binary was built locally with the exact CI command (`GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v4.1.0" .`, sha256 `9e6f3255…`), scp'd, sha256 re-verified on the VM, v4.0.0 backed up as `gollama.bak-v400-202608220500`, swapped + `systemctl restart gollama`. **Verified:** `GET /api/v1/version` → `v4.1.0`; the restart cleanly killed the old Qwen3.8-27B instance (port 8083, PID 693656) with **no orphan** (`pgrep llama-server` empty for the old PID), and gollama auto-relaunched it (port 8081) which came back `ready: true` with `gpu_util_per_gpu` populated — confirming P2-T3 clean shutdown + P2-T7 metrics + the auto-launch path on v4.1.0 in production. (Note: the model serving the deploying DSH session is itself a gollama instance, so the restart briefly interrupted it; it recovered via cold-start.)
- **Pre-existing bug noted (NOT fixed in this phase):** HF file selection is case-sensitive — `gollama pull ...:Q4_K_M` on a repo whose files are lowercase (`q4_k_m.gguf`) fails to match the quant and falls back to *all* GGUF files (e.g. grabs `fp16`). See the 2026-08-22 smoke run. Candidate for a follow-up fix.

---

## Phase 5 — Architecture & code quality → `v4.2.0`

- [ ] **P5-T1: One SSE streaming proxy** (`pkg/server/server.go:724-884` vs 1699-1841)
  - Extract `proxySSE(w, resp, port, opts{strip, merge, touch})` used by both `handleChat` and `proxyToInstance`; the 503-loading retry loop moves into the shared helper so `handleChat` gets it too.
  - Unify `[DONE]` handling: `handleChat` currently drops the marker (pinned by `TestHandleChatStream`) — the merged loop must send exactly one `data: [DONE]` on both paths. The UI already handles it (`ui.go:2074`: `if (data === '[DONE]') continue;`).
  - P0-T2 tests pin the merged behavior.
- [ ] **P5-T2: One GGUF parser, one quant regex**
  - Delete `ReadBlockCount` (`pkg/model/gguf.go:240-299`); `manager.Start` uses `readGGUFMetadata` (which already yields `BlockCount`) via a cached lookup (cache path→metadata in a process-local `map[string]ggufMetadata` with a mutex).
  - Single exported `StripQuantSuffix(name) string` (merge the three divergent regexes at `model.go:698`, `model.go:782`, `gguf.go:233`); table-driven test over real filenames (incl. `UD-`, `ARM-`, `IQ*` cases).
  - Fix the scan-path ordering bug pinned by P0-T6: `ScanModels` replaces underscores→hyphens *before* the quant-strip regex runs, so scanned short names keep the quant suffix.
- [ ] **P5-T3: Typed flag model** (`pkg/model/presets.go` or new `flags.go`)
  - `type Flags struct { values map[string]*string; order []string; standalone map[string]bool }`
  - `ParseFlags(args []string) Flags` — handles `--k v` **and** `--k=v`, records order.
  - `Flags.Merge(base, override)` (override wins, standalone flags set/unset), `Flags.Args() []string`.
  - Replace `sanitizeFlags`, `ProfileFlags`, and the merge loop in `manager.Start` with this. `standaloneFlags` map stays as data (add `--no-*` derivations where obvious).
  - Full matrix unit tests (this is the piece most likely to break real deployments — keep P0-T4 snapshots as the acceptance diff).
- [ ] **P5-T4: Split `ui.go`** (into the existing-but-empty `web/` dir)
  - `web/index.html`, `web/app.css`, `web/app.js` + `go:embed` in `pkg/ui`. Same zero-build property; real syntax highlighting in editors; `switchView` nav lookup (the `onclick*=` substring gotcha) replaced by `data-view` attributes.
  - Diff-check: served HTML must be byte-identical modulo the embed (a test comparing rendered page to the old const during transition).
- [ ] **P5-T5: `slog`**
  - Replace `log.Printf` with `slog` (text handler to stderr, JSON optional via env). All packages; keep output shape close to current for log-parsing muscle memory.
- [ ] **P5-T6: Small cleanups**
  - Dedupe the systemd-unit writer (P1-T7 leaves two call sites).
  - `chat.WaitForReady` hardcodes `HOME` env for the log path (`pkg/chat/chat.go:49-50`) — use `model.GollamaDir()`.
  - `tmpFileReader` dead code (`pkg/llama/binary.go:805-808`).
  - `main.go` CLI: consider a small hand-rolled arg parser to kill the `os.Args[1]` switch duplication (`serve` port parsing, `--listen` from P1).

**Exit:** no duplicated SSE/GGUF/flag logic; `web/` editable; `go vet` + tests still green; behavior unchanged (v4.2.0 is a refactor release).

---

## Cross-cutting: gollama VM (192.168.1.36) migration notes

1. **After Phase 1 deploy:**
   - `gollama.service`: `ExecStart=/usr/local/bin/gollama serve --listen 0.0.0.0` (keep LAN UI access; token is the gate).
   - Grab the generated token from the Settings page; add `?token=…` (or header) to:
     - hermes `ghost_post_creator.js` image-gen call (`POST /v1/images/generations`),
     - hermes `post_watcher.sh` warmup call if any,
     - any other cron/agent consumers (search hermes scripts for `9080`).
2. **After Phase 3 deploy:** `systemctl stop gollama` will now kill llama-server children — expect a cleaner VRAM state; verify `nvidia-smi` is idle after stop.
3. **General:** `gollama self-update <tag>` then `gollama restart` per phase; keep a `~/.gollama/config.json.bak` (auto-made on save) before each phase.

## Risk register

| Risk | Mitigation |
|------|-----------|
| P1 auth breaks hermes cron mid-phase | Token is opt-in per config; deploy order = tag → update unit → update hermes scripts → restart |
| P3 coordinator changes cold-start timing (visible to agents) | Coalescing only *waits* longer in the same-model case; different-model case was already racy. Heartbeats keep clients alive. |
| P5-T3 flag model subtly changes real launch args | P0-T4 snapshot tests = acceptance gate; diff the exact `llama-server` command lines before/after on the VM for every profile (`openclaw`, `ornith`, `Muse Glimmer`, `deepseek-v4-flash`) |
| P5-T4 UI split regresses the SPA | Byte-identical page test; keep v3.7.3 `ui.go` in git history as reference |
| Breaking the zero-dep constraint under time pressure | Explicitly out of scope for every phase; `go.mod` must stay dependency-free in CI (add a check: `go list -m all` == module only) |

## Out of scope (backlog, not in this plan)

- TUI mode, multi-model chat routing, MCP server, context-window management (existing backlog items).
- Image engine: the Python diffusers app itself (`/opt/image-api/app.py`) is a separate codebase — only its management (ports, switching, env) is touched here.
- Config format versioning (no schema migration system yet — avoid adding one until the config actually needs it).
