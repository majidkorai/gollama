# Gollama Phase 6 — Ops, Freshness & UI Wins → `v4.3.0`

**Status:** Complete (all T1–T6 done, verified). Ready to tag `v4.3.0` and deploy. The robustness plan (Phases 0–5, `v4.2.0`/`v4.2.1`) is complete.

## Why this phase (scope rationale)

Gollama is now the backbone of the personal AI stack (hermes agent, blog pipeline, image gen) on the 2×3090 VM. Two gaps matter for that:

1. **Nothing watches gollama.** If the VM or the service dies, the failure is discovered indirectly (blog post doesn't arrive) — possibly hours later. Uptime Kuma (status.talkingtech.io, Telegram alerts) already runs; gollama just needs a probe.
2. **The llama-server binary is opaque.** The VM runs a custom CUDA build (b396, hand-built from `/opt/llama.cpp`). `EnsureLlamaServer` uses the existing binary "regardless of version tag" (binary.go:413), so there is zero visibility into how stale it is or when rebuilding is worth doing.

Everything else proposed earlier (Prometheus metrics, per-request flags, instance-fingerprint identity) is **explicitly deferred**: no active consumer, single user, revisit on demand.

## ▶ Resume here

**Where we are:** v4.2.1 deployed on the gollama VM (192.168.1.36), working tree clean, `go build`/`go vet`/`go test ./...` green (plus `-race` on `pkg/server`, linux/windows/darwin cross-builds).

**How to resume:**
1. Read this file; task checkboxes are the source of truth.
2. Confirm green baseline: `go build ./... && go vet ./... && go test ./...` (add `-race` for pkg/server; proxy tests ~20–35s).
3. **All of T1–T6 are implemented and verified.** `web/testdata/page_reference.html` is regenerated (includes all UI changes). Remaining: tag `v4.3.0` + deploy (see below + the exit-criteria smoke checks).
4. Tag `v4.3.0` only when the exit criteria are met; deploy with the local-build + scp pattern (the `gh` token on this machine is still invalid — see Phase 4/5 deploy notes for the exact commands).

**Conventions (carried over from the robustness plan):**
- Zero external dependencies (stdlib only).
- Tests: `t.Setenv("HOME", t.TempDir())` isolation; hermetic managers via `manager.NewManagerNoRecovery()`; `fakeUpstream` for proxy tests; pure-parser tests cross-platform.
- Tooling quirk: the session file-write pipeline strips angle-bracket pairs from written content — keep hex-escape style in `transforms_test.go` and verify any HTML-like literals on disk after writing.
- UI: files live in `pkg/ui/web/`; the served page must stay byte-identical to `web/testdata/page_reference.html` (`TestPageMatchesReference`). Regenerate the reference ONCE, after all intentional UI changes in this phase are in (a temporary dump test gated on `GOLLAMA_DUMP_PAGE=1`, see the comment in the test).

## Goals

1. **Alertable** — a dead gollama or VM pages Telegram via Uptime Kuma within a minute, not discovered via a missing blog post.
2. **Honest about staleness** — tell the operator when the llama-server binary lags upstream, without auto-updating a custom build and without false alarms on unparseable versions.
3. **UI wins** — small, high-leverage improvements: freshness badge, chat stop button, copyable OpenAI endpoint, and responsiveness polish (mobile logo, ultrawide cap).

## Release & deploy strategy

| Phase | Tag | Notes |
|-------|-----|-------|
| 6 | `v4.3.0` | ops + freshness + UI wins |

- Deploy: build locally from the tag (`GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=v4.3.0" .`), scp, sha256-verify, back up the old binary, swap, `systemctl restart gollama` — same pattern as v4.1.0–v4.2.0.
- **Post-deploy (operator task, not code):** add a Kuma HTTP monitor for `http://192.168.1.36:9080/healthz` (30s interval, 3 retries, Telegram notify-down). `/healthz` is intentionally token-free.
- No config changes needed for consumers (hermes etc.); the 503/Retry-After contract is untouched.

---

## Tasks

- [x] **T1: `GET /healthz` liveness endpoint** (`pkg/server/server.go`)
  - Registered **outside** `requireAuth` (Kuma has no token): `s.mux.HandleFunc("/healthz", ...)` alongside the open UI routes.
  - Response: `200 {"status":"ok","version":"vX.Y.Z"}`. Nothing else — no model inventory, no instance state (the token is the gate for that; this is a liveness probe, not a status page).
  - Deliberately does NOT check instances: a dead llama-server is not a gollama failure (auto-launch recovers on demand).
  - Tests: `TestHealthzOpen` — 200 without a token even when `api_token` is set, body carries the version; `TestHealthzNotOnAPIPrefix` guard if cheap (it must not be shadowed by the `/api/v1/*` auth middleware).
- [x] **T2: llama-server freshness check** (`pkg/llama/binary.go`, `pkg/server/server.go`, `main.go`)
  - **Cached latest-release lookup:** new `LatestReleaseInfo() (tag, releaseURL string, err error)` in pkg/llama — memoized behind a mutex with a **1-hour TTL** (unauthenticated GitHub rate limit is 60 req/h/IP; the UI polls this on every Settings open). A fresh fetch on cache miss; a *cached error* is returned for a short backoff (e.g. 5 min) so a GitHub outage doesn't hammer the API on every Settings open.
  - **Latest-release endpoint gotcha (fixed):** llama.cpp marks its build releases (`bXXX`) as *prereleases*, so GitHub's `releases/latest` endpoint returns the stale non-prerelease tag (`v0.2.0`), not the current build. `fetchLatestRelease` therefore hits the releases **list** endpoint (`?per_page=15`) and picks the release with the **highest build number** — correct for both the freshness check and `GetReleaseData` (the `gollama update` download path).
  - **Installed version:** `LlamaServerVersion()` (version file) first; if empty, **fallback**: run `llama-server --version` via `FindLlamaServer()` with a 5s timeout and parse the build number from the output. (The VM's custom build has no/odd version file — this makes the check work there too.)
  - **Comparison (pure, tested):** `CompareBuildNumbers(installed, latest string) (behind int, comparable bool)` — extract the build number (`b1234` → 1234, or first run of ≥3 digits); `comparable=false` when either side has none (custom/unknown builds) → the UI shows **no** badge rather than a misleading "outdated".
  - **Endpoint:** extend the existing `GET /api/v1/version` response (already auth-gated, already called by Settings) with: `llama_server_latest`, `llama_server_comparable`, `llama_server_outdated`, `llama_server_builds_behind`, `llama_server_release_url`. On lookup failure: fields present, `llama_server_latest: null` + `llama_server_check_error` — **never 500** because GitHub is unreachable.
  - **CLI:** `gollama llama-version` — prints installed, latest, and status (up to date / N builds behind / unknown), exit 0 in all cases (informational command).
  - Tests: `TestCompareBuildNumbers` (table: equal, behind, ahead, `b`-prefixed, bare numbers, unparseable both/one); endpoint test with a fake GitHub server via a `releaseAPIBase` package-var seam (same pattern as `checksumURLBase` in self-update); TTL-cache test (second call must not re-fetch).
- [x] **T3: UI — llama-server freshness badge** (`pkg/ui/web/index.html`, `app.js`, `app.css`)
  - In Settings → version card, next to the llama-server tag:
    - `outdated && comparable` → amber `.badge` "N BEHIND" wrapping the `llama_server_release_url` (release notes link).
    - up to date → green `.badge` "UP TO DATE".
    - not comparable / unknown / check error → no badge (version tag stands alone).
  - Fetched with the existing `loadSettings()` `/api/v1/version` call — no extra request, no polling.
  - Uses the existing badge classes (`badge-amber`/`badge-green`, LED dot, mono uppercase) — no new design language.
- [x] **T4: UI — stop button for chat streams** (`pkg/ui/web/index.html`, `app.js`, `app.css`)
  - `sendChat()` currently reads the SSE stream with no way to cancel; the server already cancels upstream generation on client disconnect (P3-T5), so this is UI-only.
  - Per-send `AbortController` → pass `{signal}` through `apiFetch` (it already forwards `opts`); while streaming, the send button becomes a **Stop** button (same slot, `.danger` styling); on click → `controller.abort()`.
  - Abort handling: catch `AbortError` distinctly in the read loop — finalize the in-flight message with a "(stopped)" note, **do not** render it as an error, do not push it into `chatHistory` as a complete assistant turn (or push it with the partial content — pick partial-content-preserving; pin the choice in a code comment).
  - Button restores to Send when the stream ends (normal completion, error, or abort).
- [x] **T5: UI — OpenAI endpoint row in Settings** (`pkg/ui/web/index.html`, `app.js`)
  - New detail-row in the version card: `OpenAI endpoint` → `location.origin + "/v1"` with a **Copy** button (existing `fallbackCopy()` — LAN/non-HTTPS safe).
  - One-line hint under it: "add `?token=` or `Authorization: Bearer` (see API token below)".
  - Purely display-side; no backend change.
  - After T3–T5: regenerate `web/testdata/page_reference.html` (dump test), confirm `TestPageMatchesReference` green.

- [x] **T6: UI — responsiveness polish (mobile logo + ultrawide cap)** (`pkg/ui/web/app.css`)
   - **T6-A: mobile logo clipping fix.** At ≤768px the media query forces the 60px sidebar but the "GPU CONSOLE" subtitle (`.logo-sub`) is only hidden by the `.collapsed` class, which mobile never sets — so it renders clipped ("PU CONSOL"). Add `.sidebar .logo-sub` to the mobile `display: none` group (line ~527) so the icon-only rail matches the collapsed desktop state.
   - **T6-B: ultrawide content cap.** `.main` has no max-width, so on wide monitors (2560px) the MODEL select stretches to ~1300px and empty states center in a huge void. Cap the active view at `max-width: 1440px` centered (`margin: 0 auto; width: 100%`) so the console keeps a deliberate width; the faceplate and grid texture stay full-bleed.
   - Both are CSS-only. Verify with device-emulated screenshots: logo fully hidden at 375px (no clipped text), content centered/capped at 2560px, no horizontal overflow at 320px.
   - After T6: regenerate `web/testdata/page_reference.html` (dump test), confirm `TestPageMatchesReference` green.

**Exit criteria (all required for the tag):**
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./pkg/server/` green; linux/windows/darwin cross-builds green; zero-dependency check.
- Local smoke: fresh `serve` → `/healthz` 200 without token (with a token configured); `/api/v1/version` returns freshness fields; `gollama llama-version` prints status; UI: badge renders per state (test by temporarily pointing `releaseAPIBase` at a fake server), Stop button cancels a stream, endpoint row copies.
- VM smoke (post-deploy): `curl -s http://192.168.1.36:9080/healthz` from another LAN host (no token); `gollama llama-version` on the VM shows the custom build's honest status; Settings badge visible; Kuma monitor added and green (operator task).

## Risk register

| Risk | Mitigation |
|------|-----------|
| GitHub API rate limit (60/h unauthenticated) from UI polling | 1h TTL cache + 5-min error backoff in T2; single call site (Settings load) |
| `llama-server --version` fallback hangs or misbehaves on a custom build | 5s exec timeout; failure degrades to "unknown" (no badge), never blocks the endpoint |
| Misleading "outdated" on the VM's custom build | `comparable=false` → no badge; comparison only on parseable build numbers (pinned test) |
| `/healthz` unauthenticated = info leak | Version string only (already public via releases); no models/instances/config |
| UI changes regress the SPA | Single page-reference regeneration at the end; `TestPageMatchesReference` guards |
| Abort handling swallows real errors | `AbortError` matched by name only; everything else keeps current error rendering |

## Deferred (parked, revisit on demand)

- **Per-request llama.cpp flags** on the OpenAI proxy/warmup — profiles cover the stable per-model config; per-request flags only apply at cold start and would no-op most of the time. Revisit if a concrete client need or an external request lands.
- **Flags-aware instance identity** (model+flags fingerprint in the coordinator) — same trigger.
- **Prometheus `/metrics`** — no consumer; the faceplate covers what's glanced at.
- **Response metadata headers** (`X-Gollama-Instance`) — cheap, but no active debugging pain yet.
