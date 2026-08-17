# Gollama Robustness Plan

**Status:** Phase 0 complete (2026-08-18) — test safety net in place, `go build`/`go vet`/`go test -race` green.
**Baseline:** v3.7.3, `main.go` + `pkg/{server,manager,model,llama,chat,ui}`, stdlib-only, `go build`/`go vet`/`go test` all green.

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
  - `ScanModels` idempotence, split-file indexing (part 1 only, size = sum), missing-blob filtering, `ResolveModelBlob` exact/path/missing, fuzzy-match nondeterminism documented (P2-T5 fixes).
  - **Bonus finding pinned:** scan path replaces underscores→hyphens *before* the quant-strip regex (which expects underscores), so scanned short names keep the quant suffix.

**Exit:** `go test ./...` green, proxy + transforms covered.

---

## Phase 1 — Security hardening → `v3.8.0`

- [ ] **P1-T1: Bind to loopback by default** (`pkg/server/server.go:77-82`, `main.go` `serve`)
  - `Server` gains `listen string`; `gollama serve [port]` binds `127.0.0.1` by default.
  - Opt out: `--listen 0.0.0.0` flag (and `GOLLAMA_LISTEN` env for the systemd unit).
  - Update `DefaultConfig` default flags: `--host 127.0.0.1` (existing on-disk configs keep their values — no migration).
  - Update `printUsage` + README; the startup banner prints the actual bind address.
- [ ] **P1-T2: Shared-secret API token**
  - `Config` gains `APIToken string`. On first `LoadConfig`, if empty → generate 32 random bytes (hex), save, and the CLI prints it once: `gollama: API token generated — <token> (shown again in Web UI → Settings)`.
  - New middleware in `registerRoutes`: if `cfg.APIToken != ""`, require `Authorization: Bearer <token>` **or** `?token=<token>` (query form for curl/cron simplicity) on all `/api/v1/*` and `/v1/*` routes. UI assets (`/`, `/logo.svg`) stay open — the UI is a viewer; its fetches carry the token.
  - UI: Settings page shows/generates the token; stores it in `localStorage`; attaches header to all `fetch` calls.
  - **Backward compat:** empty token = auth disabled (logs a warning at startup: "no API token — gollama is unauthenticated").
  - `handleRestart` and `handleModelDelete` are the highest-value protected routes; protect all uniformly.
  - Update AGENTS.md "Caller guidance" (hermes/gollama API section) with token usage.
- [ ] **P1-T3: Fix chat-history path traversal** (`pkg/server/server.go:583`, `pkg/model/chat_history.go`)
  - Validate id: `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$` → 400 on mismatch. Applies to GET/PUT/DELETE `/api/v1/chats/{id}`.
  - Defense in depth: `chatPath` also rejects ids containing `/`, `\`, `..`.
- [ ] **P1-T4: Fix model-delete path check** (`pkg/server/server.go:175-180`)
  - Replace `strings.HasPrefix(absPath, modelsDir)` with `rel, err := filepath.Rel(modelsDir, absPath); err == nil && !strings.HasPrefix(rel, "..")`.
- [ ] **P1-T5: Verify `self-update` against release checksums** (`pkg/llama/binary.go:176-290`)
  - After downloading the binary, fetch `https://github.com/majidkorai/gollama/releases/download/<tag>/checksums.txt`, find the sha256 line for our asset name, verify. Abort on mismatch. (CI already ships `checksums.txt`.)
- [ ] **P1-T6: No silent package installs** (`pkg/llama/binary.go:522`)
  - `checkDependencies` asks `Install missing packages: libgomp1, libatomic1? [y/N]` unless `GOLLAMA_AUTO_INSTALL_DEPS=1` or `--yes`-style flag is set. Non-interactive (no TTY) → warn and list the apt command instead of installing.
- [ ] **P1-T7: systemd unit fixes** (`main.go:259-289`, wizard copy at 450-481 — dedupe into one helper)
  - Drop hardcoded `Environment=HOME=/root` (inherit HOME).
  - If uid != 0: install a **user** unit to `~/.config/systemd/user/gollama.service` via `systemctl --user` instead of failing.
  - Document the phase-1 deploy: `gollama.service` needs `ExecStart=%s serve --listen 0.0.0.0` (LAN access) — the token is then the protection.
- [ ] **P1-T8: HTTP client timeout for API calls** (`pkg/model/model.go:352-356`)
  - `model.HTTPClient` stays timeout-free (it carries long downloads); add `model.APIClient` with `Timeout: 30s` and use it for HF search / GitHub release calls (`SearchModels`, `SearchImageModels`, `ListRepoGGUFFiles`, `pullModelInternal` metadata fetch, `GetReleaseData`, `SelfUpdate` release listing).

**Exit:** unauthenticated LAN access closed (token + loopback default), traversal closed, self-update verified.
**Smoke test:** `gollama serve` on the VM → UI reachable via LAN with token in Settings; `curl` without token → 401; hermes image-gen call updated with `?token=` works.

---

## Phase 2 — Correctness → `v3.9.0`

- [ ] **P2-T1: Make `merge_reasoning` actually work** (`pkg/server/server.go:1759-1760, 1832-1837`)
  - Stream path: when `shouldMerge`, call `mergeReasoningContent(data)` instead of pass-through.
  - Non-stream path: after `convertCompleteThink`, if merge → move `reasoning_content` into `content`.
  - P0-T3 tests now assert the *wired* behavior.
- [ ] **P2-T2: `gollama run` no longer a silent no-op** (`main.go:213`)
  - Remove the `len(mgr.List()) == 0` guard; always start (auto-port assignment already handles collisions). If the caller passes an explicit port in use → clear error.
- [ ] **P2-T3: Clean shutdown** (`main.go` `serve`, `pkg/manager`)
  - New `Manager.StopAll() []int` (iterate a snapshot, `Stop` each).
  - `serve` on SIGINT/SIGTERM: cancel ctx → `mgr.StopAll()` (wait ≤ 5s) → exit.
  - `Stop` improvement: after `os.Interrupt` + 500ms, check `cmd.ProcessState` via the `cmd.Wait()` goroutine result instead of blind `Kill`; keep `Kill` only if still alive. (Requires `Instance` to carry its `*exec.Cmd` or a done-channel — store `cmd` in the instance struct.)
- [ ] **P2-T4: One readiness deadline** (`pkg/manager/manager.go:506`)
  - Move the `GOLLAMA_MODEL_LOAD_TIMEOUT` reader to `pkg/manager` (or `pkg/model`) as `LoadTimeout()`; both the manager's health-poll goroutine and the server's `waitForReady` use it. Default 5 min.
- [ ] **P2-T5: Deterministic model matching** (`pkg/manager/manager.go:792-824`, `pkg/server/server.go:1011-1024`)
  - `FindInstanceByModel`: collect **all** candidates, score exact-fold > short-name-equal > suffix > substring; ties broken by lowest port. Log when a substring match was used.
  - Image auto-detect with empty `model` and multiple image profiles → 400 listing profile names (no more map-order lottery).
- [ ] **P2-T6: Windows orphan recovery** (`pkg/manager/manager.go:183-199`)
  - If WMI fails, log and **skip** the process instead of registering it under a guessed port (don't burn ports, don't show phantom instances).
- [ ] **P2-T7: Per-GPU utilization + real CPU** (`pkg/manager/manager.go:842-886`)
  - `queryGpuUtil` returns `[]float64` (per GPU); instance gets `GpuUtil` = max, plus `GpuUtilPerGPU []float64` (UI badge can show `GPU0 92% / GPU1 3%`).
  - CPU: on Linux, sample `/proc/<pid>/stat` (utime+stime) twice 1s apart in the metrics goroutine → instantaneous %; keep `ps` as fallback.

**Exit:** no feature silently lies; shutdown reclaims VRAM; matching is predictable.
**Smoke test:** kill gollama → `pgrep llama-server` empty; `gollama run` with an instance alive starts a second one; UI shows per-GPU split.

---

## Phase 3 — Concurrency & lifecycle → `v4.0.0` (major)

- [ ] **P3-T1: Model-switch coordinator** (new `pkg/manager/coordinator.go` or in `server.go`)
  - All model switches go through one function: `coordinator.SwitchAndStart(model, profile, ...)`.
  - Internals: a `sync.Mutex` serializing switches + an in-flight map `model → chan *Instance` so concurrent requests for the *same* model coalesce (second caller waits on the channel, then reuses the instance) while different models queue.
  - Entry points rewired: `proxyToInstance`, `handleWarmup`, `handleInstances` (POST), `handleV1ImageGenerations` (text-stop part), CLI `chat`/`run`.
  - `switchToModel`'s up-to-62s image grace wait moves *inside* the lock (it already blocks; making it explicit is the point).
  - Guard: a switch that waits > N seconds (e.g. 90s) logs loudly.
- [ ] **P3-T2: Don't hold `m.mu` while sleeping** (`pkg/manager/manager.go:310-334, 574-596, 690-709`)
  - Port allocation: pick candidate port under lock, **reserve** it in a `reserved map[int]bool`, release lock, do the availability scan/retries outside, then re-acquire and commit (or unreserve on failure).
  - `Start` process spawn: build everything (args, env, log file) under lock is fine, but move `cmd.Start` + the 2s GPU cooldown + health polling outside the lock.
  - `Stop`: signal the process outside the lock (the 500ms sleep is the worst offender); mutate state under lock.
  - Regression test: two `Start` calls in parallel don't serialize on the sleep (assert wall-clock < threshold), and never get the same port.
- [ ] **P3-T3: Instance lifecycle state** 
  - `Instance` carries its `*exec.Cmd` + `waitDone chan struct{}` so Stop/StopAll/wait-for-exit are exact instead of signal-blind (supports P2-T3 properly).
  - `recoverOrphans` registers recovered instances with `Ready: true` but also kicks a health poll to confirm (don't trust forever).
- [ ] **P3-T4: Sentinel errors** (`pkg/model/model.go:1144` → `pkg/server/server.go:200,263`)
  - `var ErrAlreadyExists = errors.New("model already exists")` in `pkg/model`; `errors.Is` in handlers. Same pattern for `ErrNotFound` (model/blob) where it's currently string-matched.
- [ ] **P3-T5: Request-level context on proxy paths**
  - `proxyToInstance` non-stream currently uses `context.WithTimeout(r.Context(), 10m)`; ensure *streaming* also honors client disconnect end-to-end (cancel upstream request when the client goes away — `proxyCtx` from `r.Context()` is already there; verify the read loop breaks and the upstream body is closed).

**Exit:** concurrent different-model requests queue instead of thrashing; UI polls are not blocked by port scans; `systemctl stop gollama` leaves no GPU memory behind.
**Smoke test (the important one):** fire two `curl /v1/chat/completions` for *different* models simultaneously → first model starts, second waits, then both serve; no stop/start loop in logs.

---

## Phase 4 — Robustness → `v4.1.0`

- [ ] **P4-T1: Safe model downloads** (`pkg/model/model.go:1163-1259`)
  - Write to `<name>.gguf.part`; on success `os.Rename` to final name.
  - **Resume:** if `.part` exists, `HEAD`/`Content-Range` (size probe already exists at 1171-1193 — reuse), open with `O_APPEND`, send `Range: bytes=<done>-`.
  - Verify final size == expected (when known); size mismatch → delete `.part`, error.
  - `ScanModels` skips `*.part` and zero-byte files.
  - Cancel path (ctx) deletes the `.part`.
- [ ] **P4-T2: Cheap `GET /api/v1/models`** (`pkg/server/server.go:90-100`, `pkg/model/model.go:792-813`)
  - `ScanModels` throttled: runs at startup and at most once per 60s; `?refresh=1` forces it (UI refresh button uses it).
  - `populateModelInfo` skipped when the index entry already has arch/quant/ctx/short-name (it re-parses GGUF headers of *every* file today on *every* list call).
- [ ] **P4-T3: Log hygiene** (`pkg/manager/manager.go`, `pkg/server/server.go:323-340`)
  - Tail by `Seek(-(N*4096), io.SeekEnd)` + buffer instead of `os.ReadFile` of the whole file (logs contain `\r` progress spam and grow unbounded).
  - Rotation: when a log exceeds ~10 MB, rename to `port-<port>.log.1` (one level) and keep writing.
  - Idle heuristic (`StopIdle` mtime check, `manager.go:772-778`): keep, but only let mtime *extend* activity within the TTL window (already the case) — document that direct llama-server traffic keeps instances alive.
- [ ] **P4-T4: Error propagation on state writes** (`pkg/model/model.go:189-209, 380-385`)
  - `SaveConfig` and `unsafeSaveIndex` return errors (atomic tmp+rename already in place); callers log loudly (config) or surface (API 500). At minimum: failed rename → `log.Printf` with the error, never silent.
- [ ] **P4-T5: Download progress for the systemd/CLI path**
  - `gollama pull` from CLI: keep `ProgressReader` stderr output; ensure it also works when stdout is not a TTY (guard the `\r` rendering behind `isatty` — stdlib-only: check `os.Stdout.Stat()` mode).

**Exit:** interrupted 40 GB pull resumes; model list is O(index) not O(files × GGUF headers); logs bounded.
**Smoke test:** `gollama pull` a medium model, kill mid-download, re-run → resumes; `gollama list` is instant with many models.

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
