# whzbox — Implementation plan

This plan implements `DESIGN.md` in seven phases. Each phase is a self-contained, merge-sized unit that leaves a working binary and green tests. Each later phase depends only on the ones before it.

| # | Phase | Effort | User-visible outcome |
|---|---|---|---|
| 1 | Skeleton & tooling | S | `whzbox --help` prints something, CI is green |
| 2 | Config, logging, theme | S | `whzbox version` prints build info with verbosity + color flags working |
| 3 | Session domain (core) | M | Internal only; fully unit-tested `session.Service` |
| 4 | Auth adapters + `login`/`logout` | L | `whzbox login` / `logout` work end-to-end against real Whizlabs |
| 5 | Sandbox domain (core) | S | Internal only; fully unit-tested `sandbox.Service` |
| 6 | Sandbox adapters + `create`/`destroy`/`status` | L | Full lifecycle: `whzbox create aws` → `destroy` → `status` |
| 7 | Polish: testscript, completion, release | M | Shipping artifact with shell completion and CI release |

**Ground rules across all phases:**

- Every merge leaves `go build ./...`, `go vet ./...`, and `go test ./...` green.
- Every merge leaves a runnable `whzbox` binary — no half-wired commands.
- Core packages (`internal/core/*`) never import outside the standard library or sibling core packages. This is enforced by a lint rule added in phase 1.
- Adapters expose package-private types; only their port implementations are consumed by the core.
- Secrets are never logged — every phase that touches tokens or credentials includes redaction tests.

---

## Phase 1 — Skeleton & tooling

**Goal:** Replace the prototype with the layout from `DESIGN.md` §7, wire up a real Cobra root command, and establish the tooling baseline.

**Scope in**

- Delete `main.go` at repo root (prototype).
- Create `cmd/whzbox/main.go` with `signal.NotifyContext` + `ExecuteContext` pattern.
- Create empty package directories with a single `doc.go` each so `go build ./...` walks them:
  - `internal/cli`, `internal/config`, `internal/logging`, `internal/ui`
  - `internal/core/{session,sandbox,clock}`
  - `internal/adapters/{whizlabs,awsverify,xdgstore,huhprompt}`
- Minimal `internal/cli/root.go` with just `NewRootCommand()` returning a bare `whzbox` with `SilenceUsage: true`, `SilenceErrors: true`, a `--help` string, and no subcommands yet.
- Delete `aws-sandbox` binary artifacts if they exist.
- `.gitignore` for `whzbox`, `dist/`, `.DS_Store`, `coverage.out`.
- `.golangci.yml` with:
  - `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `gofmt`, `goimports`, `revive`
  - `depguard` rule enforcing the hexagonal dependency direction: `internal/core/**` may only import stdlib + sibling `internal/core/**`
- `Makefile` (or `justfile`) with `build`, `test`, `lint`, `tidy`, `run` targets.
- `.github/workflows/ci.yml`: build + vet + test + lint on push and PR. Matrix: linux/darwin, Go 1.24.
- Keep existing `go.mod` but run `go mod tidy` to drop the aws-sdk deps until phase 6 re-adds them (or leave them — they're cheap).

**Scope out**

- No cobra subcommands (not even `version`).
- No Viper.
- No logging setup.
- No real business logic anywhere.

**Files created**

```
cmd/whzbox/main.go
internal/cli/root.go
internal/cli/doc.go
internal/config/doc.go
internal/logging/doc.go
internal/ui/doc.go
internal/core/clock/doc.go
internal/core/session/doc.go
internal/core/sandbox/doc.go
internal/adapters/whizlabs/doc.go
internal/adapters/awsverify/doc.go
internal/adapters/xdgstore/doc.go
internal/adapters/huhprompt/doc.go
.golangci.yml
.gitignore
Makefile
.github/workflows/ci.yml
```

**Files deleted**

```
main.go            (prototype)
```

**Tests**

- `cmd/whzbox/main_test.go`: smoke test that `NewRootCommand()` returns non-nil and `--help` exit-zeros.
- Lint passes including the depguard rule (intentionally empty core packages give nothing to catch yet, but the rule must be live).

**Done when**

- `go build ./...` → clean binary at `./whzbox` (or via `make build`)
- `./whzbox --help` prints usage without panic and exit code 0
- `./whzbox --version` — deferred to phase 2
- `make lint` clean
- CI green on GitHub Actions

---

## Phase 2 — Config, logging, theme, `version` command

**Goal:** Real Viper-backed config, `slog.Logger` backed by `charm.land/log/v2`, a shared UI theme, and the first real subcommand (`version`).

**Scope in**

- **`internal/config/config.go`** — `config.Config` struct:
  ```go
  type Config struct {
      LogLevel   string
      Verbose    int
      Quiet      bool
      NoColor    bool
      AssumeYes  bool
      StateDir   string
      Whizlabs   WhizlabsConfig  // BaseURL fields — default to real prod URLs
  }
  func Load(vp *viper.Viper) (Config, error)
  ```
- Viper wiring in `cli/root.go`'s `PersistentPreRunE`:
  - `vp.SetEnvPrefix("WHZBOX")`, `AutomaticEnv`, key replacer for `- → _`
  - `BindPFlags(cmd.Flags())`
  - Fail on `ReadInConfig` errors except `ConfigFileNotFoundError` (even though we won't create a config file, users might). Actually: skip `ReadInConfig` entirely since the design says no config files.
- **`internal/logging/logger.go`** — `New(cfg config.Config) *slog.Logger`:
  - Resolves level from (highest priority first) `--log-level` → `-v` count → `--quiet` → default `info`
  - Returns `slog.New(charmHandler)` where the charm handler is styled with `ui.LogStyles()`
  - Honors `--no-color` and `NO_COLOR` env var
  - Detects non-TTY stderr and disables color automatically
- **`internal/ui/theme.go`** — palette + `HuhTheme(isDark bool)` + `LogStyles()` + `SandboxFrame()` — exactly as in `DESIGN.md` §14
- **`internal/cli/app.go`** — `App` container with `Config`, `Logger`, `Clock` fields. `NewApp(vp *viper.Viper) (*App, error)` — only populates the three foundational fields for now. Service fields get added in later phases.
- **`internal/cli/version.go`** — `newVersionCommand(app **App)`:
  - Prints `whzbox <version> (<commit>) built <time>`
  - Reads from `Version`, `Commit`, `BuildTime` package vars (ldflags-populated; default to `dev`/`none`/`unknown`)
  - Also sets `rootCmd.Version` so `--version` works
- **`Makefile`**: `build` target adds `-ldflags "-X main.Version=... -X main.Commit=... -X main.BuildTime=..."`
- **`cli/root.go`** — persistent flags added:
  - `--log-level`, `-v` (count), `-q`, `--no-color`, `--yes`
  - Every flag bound to Viper with matching key

**Scope out**

- No login, session, sandbox logic.
- No TTY detection helpers beyond what the logger needs internally.

**Files created/modified**

```
internal/config/config.go         (new)
internal/config/config_test.go    (new)
internal/logging/logger.go        (new)
internal/logging/logger_test.go   (new)
internal/ui/theme.go              (new)
internal/cli/app.go               (new)
internal/cli/version.go           (new)
internal/cli/root.go              (modified — flags, PersistentPreRunE, register version)
Makefile                          (modified — ldflags)
go.sum                            (updated — charm packages)
```

**Tests**

- `config_test.go`:
  - Precedence: flag > env > default (assert with `vp.Set`-free flow)
  - `WHZBOX_LOG_LEVEL=debug` resolves correctly
  - Invalid level string returns typed error
- `logger_test.go`:
  - `-q` → `slog.LevelError`
  - `-v` → `slog.LevelInfo`
  - `-vv` → `slog.LevelDebug`
  - `--log-level debug` overrides `-q`
  - `--no-color` propagates to charm handler
- `cli/version_test.go`:
  - `whzbox version` exits 0 and writes to stdout (capture via `cmd.SetOut`)
  - Output contains the injected version string

**Done when**

- `./whzbox version` prints structured build info
- `./whzbox --version` also works (cobra built-in)
- `./whzbox -vv version` shows debug-level log lines from the logger
- `WHZBOX_LOG_LEVEL=error ./whzbox version` suppresses info logs
- `./whzbox --no-color version | cat` produces uncolored output

---

## Phase 3 — Session domain (pure core)

**Goal:** Implement the session domain (`internal/core/session` and `internal/core/clock`) with full unit test coverage against in-memory fakes. Zero adapter code.

**Scope in**

- **`internal/core/clock/clock.go`**:
  ```go
  type Clock interface { Now() time.Time }
  type Real struct{}
  func (Real) Now() time.Time { return time.Now() }
  type Fake struct { T time.Time }
  func (f *Fake) Now() time.Time { return f.T }
  func (f *Fake) Advance(d time.Duration) { f.T = f.T.Add(d) }
  ```
- **`internal/core/session/session.go`**:
  - `Tokens` value type with `AccessValid`, `AccessNearExpiry`, `Refreshable` methods (all take `now time.Time`)
  - `ErrInvalidCredentials`, `ErrSessionExpired`, `ErrPromptUnavailable`, `ErrUserAborted` sentinel errors
- **`internal/core/session/ports.go`**:
  - `IdentityProvider` (`Login`, `Refresh`)
  - `TokenStore` (`Load`, `Save`, `Clear`)
  - `Prompt` (`Credentials`)
- **`internal/core/session/service.go`**:
  - `Service` struct taking all ports + `Clock` + `*slog.Logger`
  - `NewService(...)` constructor
  - `EnsureValid(ctx) (Tokens, error)`:
    1. `store.Load` → if missing, fall to step 4
    2. If `AccessValid` and not `AccessNearExpiry(10*time.Minute)` → return
    3. If `Refreshable` → try `provider.Refresh`; on success save + return
    4. `prompt.Credentials` → `provider.Login` → save + return
    5. If prompt returns `ErrPromptUnavailable` → bubble up
  - `Login(ctx) (Tokens, error)` — always prompts, always calls provider login (no refresh path)
  - `Logout(ctx) error` — `store.Clear`

**Scope out**

- No HTTP, no files, no huh. All three ports have in-memory fakes for tests.
- No CLI wiring yet — `App.Session` stays nil.

**Files created**

```
internal/core/clock/clock.go
internal/core/clock/clock_test.go
internal/core/session/session.go
internal/core/session/session_test.go
internal/core/session/ports.go
internal/core/session/service.go
internal/core/session/service_test.go
internal/core/session/fakes_test.go  (in-memory IdentityProvider, TokenStore, Prompt)
```

**Tests**

- `session_test.go`: expiry math (edge cases around clock skew, near-expiry window)
- `service_test.go` (table-driven):
  - cache hit, fresh → no provider calls, returns cached
  - cache hit, near expiry, refresh ok → provider.Refresh called once, new tokens saved
  - cache hit, expired, refreshable, refresh ok → as above
  - cache hit, refresh fails, prompt ok → fallback login, tokens saved
  - cache hit, refresh fails, no prompt available → `ErrPromptUnavailable`
  - cache miss, prompt ok → login, save
  - cache miss, no prompt → `ErrPromptUnavailable`
  - user aborts prompt → `ErrUserAborted` surfaces
  - `Login()` ignores cache entirely
  - `Logout()` clears store even when empty
- Each test uses a `clock.Fake` to control "now"

**Done when**

- `go test ./internal/core/session/... -count=1 -race` passes
- Coverage on `internal/core/session` ≥ 90%
- `depguard` confirms no non-stdlib imports in `internal/core/session`

---

## Phase 4 — Auth adapters + `login`/`logout` commands

**Goal:** Wire the session core to real I/O. At the end, `whzbox login` and `whzbox logout` work end-to-end against production Whizlabs.

**Scope in**

- **`internal/adapters/whizlabs/client.go`**:
  - `Client` struct with `baseURL`, `playURL`, `*http.Client`, `*slog.Logger`
  - Constructor: `NewClient(cfg config.WhizlabsConfig, logger *slog.Logger) *Client`
  - Private `postJSON[Req, Resp any]` generic helper that handles: marshal, headers (origin, referer, user-agent, content-type, optional bearer, optional x-session-id), read, status check, unmarshal, typed error for 4xx/5xx
  - Secret-redaction middleware for debug logging: logs method+URL+status by default, adds redacted headers+bodies at debug level (authorization → `Bearer <redacted>`, `access_token`/`refresh_token`/`password` fields in JSON → `<redacted>`)
- **`internal/adapters/whizlabs/auth.go`** — implements `session.IdentityProvider`:
  - `Login(ctx, email, password) (session.Tokens, error)`:
    - generates `X-Session-Id` UUID
    - POST `/Stage/auth/login`
    - parses `success`, `data.access_token`, `data.refresh_token`
    - decodes JWT `exp` claims without verifying the signature (we trust the issuer) to populate `Tokens.AccessTokenExpiresAt` / `Tokens.RefreshTokenExpiresAt` / `Tokens.UserEmail`
    - maps `401` / `success=0` → `session.ErrInvalidCredentials`
  - `Refresh(ctx, current) (session.Tokens, error)`:
    - GET `/Stage/auth/exchange` with `Authorization: Bearer <current.AccessToken>`
    - same parse/decode logic
    - `401` → wrapped `session.ErrSessionExpired` so the service falls through to prompt
- **`internal/adapters/xdgstore/store.go`** — implements `session.TokenStore`:
  - `stateDir()` resolves `WHZBOX_STATE_DIR` → `XDG_STATE_HOME/whzbox` → `~/.local/state/whzbox`
  - `New(stateDir string) (*Store, error)` — creates dir with `0700` if missing
  - `Load(ctx)`:
    - file missing → `(Tokens{}, false, nil)`
    - parse error or unknown version → deletes file + returns `false, nil` (self-heal on corruption) *and* logs a warning
    - mode check: file must be `0600`, otherwise refuse to load
  - `Save(ctx, t)`:
    - marshal to `state.json.tmp` with `0600`
    - fsync, then `os.Rename`
  - `Clear(ctx)` — `os.Remove`, ignores `os.ErrNotExist`
- **`internal/ui/tty.go`** — `IsInteractive(f *os.File) bool` using `golang.org/x/term.IsTerminal`
- **`internal/adapters/huhprompt/prompt.go`** — implements `session.Prompt`:
  - Checks `ui.IsInteractive(os.Stdin)` and `ui.IsInteractive(os.Stderr)` at the start of `Credentials` — if either is false, returns `session.ErrPromptUnavailable`
  - Builds a `huh.Form` with two fields (Input + Password), themed via `ui.HuhTheme`
  - Returns `session.ErrUserAborted` when `huh.ErrUserAborted` comes out
- **`internal/cli/app.go`** — extend `NewApp`:
  - Constructs the whizlabs client, xdg store, huh prompt
  - Constructs `session.Service` and exposes it as `app.Session`
- **`internal/cli/login.go`** — `newLoginCommand(app **App)`:
  - `RunE`: calls `(*app).Session.Login(cmd.Context())` and logs `INFO session established` on success
  - No flags
- **`internal/cli/logout.go`** — `newLogoutCommand(app **App)`:
  - `RunE`: calls `(*app).Session.Logout(cmd.Context())` and logs `INFO session cleared`
- **`cmd/whzbox/main.go`** — error-to-exit-code mapping for session errors (exit 2/4/5)
- **Register commands** in `cli/root.go`

**Scope out**

- No sandbox logic yet.
- No httptest fixtures stored on disk — inline fixture strings in tests are fine.

**Files created**

```
internal/adapters/whizlabs/client.go
internal/adapters/whizlabs/client_test.go
internal/adapters/whizlabs/auth.go
internal/adapters/whizlabs/auth_test.go
internal/adapters/whizlabs/fixtures_test.go    (inline JSON fixtures + httptest helper)
internal/adapters/xdgstore/store.go
internal/adapters/xdgstore/store_test.go
internal/adapters/huhprompt/prompt.go
internal/ui/tty.go
internal/cli/login.go
internal/cli/login_test.go
internal/cli/logout.go
internal/cli/logout_test.go
```

**Tests**

- `whizlabs/auth_test.go`:
  - Login happy path (fixture response from the real traffic we captured) — asserts request method, URL, headers, body shape, and parsed token fields
  - Login 401 → `session.ErrInvalidCredentials`
  - Login `success:0` → `ErrInvalidCredentials`
  - Refresh happy path → new tokens returned
  - Refresh 401 → wrapped `ErrSessionExpired`
  - Context cancellation mid-request
- `whizlabs/client_test.go`:
  - Redaction: `Authorization` header is redacted in debug logs
  - Redaction: `password` and `access_token` JSON fields are redacted in debug logs
- `xdgstore/store_test.go`:
  - Load missing file
  - Round-trip save + load
  - File perms after save are `0600`, dir perms are `0700`
  - Wrong perms on load → error
  - Corrupt JSON → deleted + `(_, false, nil)` with warning log
  - Atomic save: `Save` interrupted mid-write leaves old file intact
  - `Clear` on missing file is a no-op
- `cli/login_test.go` + `cli/logout_test.go`:
  - Inject fake session service via `App.Session` override
  - Assert exit codes (0 on success, 2 on `ErrInvalidCredentials`, 4 on `ErrUserAborted`, 5 on `ErrPromptUnavailable`)

**Manual E2E check before merge**

1. `rm -rf ~/.local/state/whzbox`
2. `./whzbox login` → huh prompt appears → enter real credentials → "session established"
3. `cat ~/.local/state/whzbox/state.json` → valid JSON, mode 0600
4. `./whzbox login` again → prompts again (explicit login always prompts)
5. `./whzbox logout` → state file gone
6. `./whzbox login < /dev/null` → exits 5 with clear error

**Done when**

- All manual checks pass
- All tests green, race-clean
- `whzbox login` works against production

---

## Phase 5 — Sandbox domain (pure core)

**Goal:** Implement `internal/core/sandbox` with unit tests against in-memory fakes. Zero adapter code.

**Scope in**

- **`internal/core/sandbox/sandbox.go`** — `Kind`, `Credentials`, `Identity`, `Console`, `Sandbox` types exactly as in `DESIGN.md` §8.4
- **`internal/core/sandbox/ports.go`** — `Manager` and `Provider` interfaces
- **`internal/core/sandbox/errors.go`** — `ErrUnknownKind`, `ErrNoActiveSandbox`, `ErrVerificationFailed`
- **`internal/core/sandbox/service.go`** — `Service` with `Create`, `Destroy`, `Status`:
  - `Create` implements the five-step flow from `DESIGN.md` §8.6:
    1. lookup provider by kind
    2. `session.EnsureValid`
    3. `manager.Create` → get credentials
    4. `manager.Commit` → if this fails, attempt `manager.Destroy` as best-effort cleanup and return the original error
    5. `provider.VerifyCredentials` → attach identity, return
  - `Destroy`: `session.EnsureValid` → `manager.Destroy`; `ErrNoActiveSandbox` passes through
  - `Status`: `session.EnsureValid` → `manager.Active`; returns `nil, nil` for no active sandbox

**Scope out**

- No adapters.
- No CLI wiring.

**Files created**

```
internal/core/sandbox/sandbox.go
internal/core/sandbox/sandbox_test.go
internal/core/sandbox/ports.go
internal/core/sandbox/errors.go
internal/core/sandbox/service.go
internal/core/sandbox/service_test.go
internal/core/sandbox/fakes_test.go
```

**Tests**

- `service_test.go` table-driven:
  - happy path: create → commit → verify → return fully-populated Sandbox with Identity
  - unknown kind → `ErrUnknownKind`
  - session `EnsureValid` fails → error propagated, no manager calls
  - `manager.Create` fails → error propagated, no commit called
  - `manager.Commit` fails → `manager.Destroy` called (best-effort), original commit error returned
  - `manager.Commit` fails + `Destroy` also fails → commit error still returned, destroy error logged
  - `provider.VerifyCredentials` fails → `ErrVerificationFailed` wrapping underlying error; sandbox should still be considered created (explicit decision: do NOT auto-destroy on verification failure — IAM propagation flakiness shouldn't cost the user their sandbox)
  - `Destroy` happy path
  - `Destroy` with no active sandbox → `ErrNoActiveSandbox`
  - `Status` with no active → `(nil, nil)`
  - `Status` with active → returns sandbox populated with `Identity` if manager supplies it, otherwise returns without verification (we don't re-verify on status — too slow)

**Done when**

- Core sandbox test coverage ≥ 90%
- `depguard` confirms no non-stdlib/sibling-core imports
- All existing tests still green

---

## Phase 6 — Sandbox adapters + `create`/`destroy`/`status`

**Goal:** Full lifecycle working. `whzbox create aws --duration 1h` spins up an AWS sandbox, verifies it via STS, and renders the credential box. `whzbox destroy` tears it down. `whzbox status` reports.

**Scope in**

- **Extend `internal/adapters/whizlabs`** with sandbox methods, implementing `sandbox.Manager`:
  - `sandbox.go` file holds the sandbox-specific methods
  - Each method first calls a private `exchangeForPlayToken(ctx, tokens)` that hits `play.whizlabs.com/api/web/login/user-authentication` and returns the play JWT — scoped per-call, not persisted
  - `Create(ctx, tokens, slug, duration)`:
    - POST `/api/web/play-sandbox/play-create-sandbox`
    - Parses response into `*sandbox.Sandbox` with `Kind` left empty (service fills it from the caller's kind)
    - `duration` → whole hours (1–9); validate before sending
  - `Commit(ctx, tokens, slug, duration)` → POST `/play-update-sandbox`
  - `Destroy(ctx, tokens)`:
    - POST `/play-end-sandbox` with `{error_id:"0", type:"stop-sandbox", access_token}`
    - Response message `"Sandbox Not Found"` → `sandbox.ErrNoActiveSandbox`
  - `Active(ctx, tokens)`:
    - POST `/api/web/play-sandbox/play-get-aws-sandbox-content` and inspect response for an active sandbox; if the endpoint doesn't surface active-state (TBD during implementation), we fall back to returning `(nil, nil)` and document the limitation. **Implementation note:** probe this endpoint early in the phase — we didn't capture a clear "is there an active sandbox" response during feasibility. If no clean endpoint exists, `Status` will only show the session and print `(active sandbox state unavailable)` for now.
- **`internal/adapters/awsverify/sts.go`** — `Provider` for `KindAWS`:
  - `New(region string) *STSVerifier` constructor
  - `Kind() → KindAWS`, `Slug() → "aws-sandbox"`
  - `VerifyCredentials`:
    - uses `aws-sdk-go-v2/config` + `credentials.NewStaticCredentialsProvider`
    - calls `sts.GetCallerIdentity`
    - retry loop: up to 15 attempts, 3s between, only on `InvalidClientTokenId` (matches prototype)
    - returns `sandbox.Identity{Account, UserID, ARN, Region}`
- **`internal/ui/render.go`**:
  - `RenderSandbox(w io.Writer, sb *sandbox.Sandbox)` — the credential box from `DESIGN.md` §2.1 (rounded border, padding, successgreen, key/value table inside)
  - `RenderStatus(w io.Writer, session session.Tokens, active *sandbox.Sandbox)` — the status view from §2.5
  - Both use lipgloss styles from `internal/ui/theme.go`
  - Both detect `NO_COLOR` / non-TTY and render plain text
  - All output goes to `w` (pass `cmd.OutOrStdout()` from commands)
- **`internal/cli/app.go`** — extend `NewApp` to construct `sandbox.Service` with the providers map `{KindAWS: awsverify.New("us-east-1")}`
- **`internal/cli/create.go`** — `newCreateCommand(app **App)`:
  - Positional arg: provider name, validated against `{"aws"}` (v1)
  - Flag: `--duration` (default `1h`)
  - Calls `(*app).Sandbox.Create`, renders via `ui.RenderSandbox`
- **`internal/cli/destroy.go`** — `newDestroyCommand(app **App)`:
  - No positional arg (only one active sandbox at a time)
  - Checks `cfg.AssumeYes`; if false and interactive, show `huh.NewConfirm` prompt
  - Calls `(*app).Sandbox.Destroy`, logs `INFO sandbox destroyed`
- **`internal/cli/status.go`** — `newStatusCommand(app **App)`:
  - Loads session tokens directly from the store (via a new `session.Service.Current()` method added in this phase) — does **not** call `EnsureValid` (status shouldn't trigger refresh or prompts)
  - Calls `(*app).Sandbox.Status` only if session is valid
  - Renders via `ui.RenderStatus`
- Add `session.Service.Current(ctx) (Tokens, bool, error)` — thin wrapper over `store.Load` — needed for status
- Register commands in root

**Scope out**

- No sandbox renewal.
- No machine-readable output.
- No writing to `~/.aws/credentials` etc.

**Files created/modified**

```
internal/adapters/whizlabs/sandbox.go          (new)
internal/adapters/whizlabs/sandbox_test.go     (new)
internal/adapters/awsverify/sts.go             (new)
internal/adapters/awsverify/sts_test.go        (new)
internal/adapters/awsverify/mock_test.go       (new — sts client interface for mocking)
internal/ui/render.go                          (new)
internal/ui/render_test.go                     (new — golden-file comparison)
internal/ui/testdata/sandbox.golden            (new)
internal/ui/testdata/status.golden             (new)
internal/cli/create.go                         (new)
internal/cli/create_test.go                    (new)
internal/cli/destroy.go                        (new)
internal/cli/destroy_test.go                   (new)
internal/cli/status.go                         (new)
internal/cli/status_test.go                    (new)
internal/cli/app.go                            (modified — wire sandbox service)
internal/core/session/service.go               (modified — add Current)
internal/core/session/service_test.go          (modified — cover Current)
go.mod / go.sum                                (aws-sdk-go-v2 deps back in)
```

**Tests**

- `whizlabs/sandbox_test.go`:
  - Create: asserts POST to play-create-sandbox with correct body; fixture response returns creds; parsed into `*Sandbox`
  - Create: validates duration is 1–9; rejects 0 or 10
  - Commit: asserts POST to play-update-sandbox
  - Destroy: asserts POST to play-end-sandbox
  - Destroy: `"Sandbox Not Found"` → `ErrNoActiveSandbox`
  - `exchangeForPlayToken` called once per operation, result not cached across operations
- `awsverify/sts_test.go`:
  - Define a tiny `stsClient` interface in the package, inject a mock
  - Happy path: first call succeeds → returns identity
  - Retry path: first two calls return `InvalidClientTokenId`, third succeeds → returns identity after correct backoff
  - Non-retryable error: first call returns `AccessDenied` → returns immediately
  - Max retries exceeded → returns wrapped last error
- `render_test.go`:
  - Golden file comparison for credential box and status view
  - Test with and without color
- `cli/create_test.go`:
  - Injects fake sandbox service
  - Success: exit 0, stdout contains the expected rendered box
  - `ErrVerificationFailed`: exit 3
  - Invalid provider name: exit 1 with clear message
- `cli/destroy_test.go`:
  - Without `--yes` on non-TTY: requires confirmation → fails cleanly
  - With `--yes`: destroys
  - `ErrNoActiveSandbox`: exit 3
- `cli/status_test.go`:
  - No session: prints "not logged in" without prompting
  - Session, no sandbox: prints session block + "(none)"
  - Session, active sandbox: prints both

**Manual E2E check before merge**

1. `./whzbox logout && ./whzbox login` (fresh start)
2. `./whzbox create aws --duration 1h` → huh-less (already logged in) → progress logs → credential box on stdout
3. Copy `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` from output → `aws sts get-caller-identity` with `--region us-east-1` → matches the ARN in the box
4. `./whzbox status` → shows session + active sandbox
5. `./whzbox destroy --yes` → cleanup
6. `./whzbox status` → no active sandbox
7. `./whzbox create aws | cat` → stdout capture works (only box, no log chatter)

**Done when**

- All manual checks pass
- All tests green, race-clean
- No secrets in logs even at `-vvv`
- Goreleaser-ready (phase 7 formalizes this)

---

## Phase 7 — Polish: testscript, completion, release

**Goal:** Ship-ready artifact. Integration tests cover the full command surface via testscript; shell completion generation works; CI builds release artifacts on tag.

**Scope in**

- **testscript end-to-end tests** in `internal/cli/testscript_test.go`:
  - Setup: `testscript.Main` entry point so `whzbox` commands work inside `.txtar` files
  - `testdata/script/login_logout.txtar`: login with fake prompt, status, logout — uses a test-only build tag `testfake` or an env-var override to swap in in-memory adapters
  - `testdata/script/create_destroy.txtar`: full lifecycle with mocked whizlabs + stub sts
  - `testdata/script/non_interactive.txtar`: confirms non-TTY login fails with exit 5
  - `testdata/script/flags.txtar`: covers `-q`, `-v`, `--no-color`, `--yes`
- **Shell completion**:
  - `internal/cli/completion.go` — `newCompletionCommand()` using cobra's built-in `GenBashCompletion` / `GenZshCompletion` / etc.
  - Covers `bash`, `zsh`, `fish`, `powershell`
  - Document in `--help`
- **Goreleaser**:
  - `.goreleaser.yaml` building darwin/linux, amd64/arm64
  - Cosign-signed checksums (if the user has a Cosign key; otherwise skip)
  - Homebrew tap config (deferred — commented out)
  - GitHub Actions release workflow `.github/workflows/release.yml` triggered on `v*` tags
- **Docs**:
  - `README.md` with install instructions, quickstart, examples (reuse ASCII from DESIGN.md §2)
  - `docs/usage.md` for detailed flag and env var reference (optional if README is comprehensive)
- **`SECURITY.md`**: how to report vulnerabilities, note about local credential storage permissions
- **Version stamping**: goreleaser provides `Version`, `Commit`, `BuildTime` via ldflags; phase 2's `cmd/whzbox/main.go` vars are finally populated correctly in release builds
- **Optional: `--version` JSON output** for automation — defer if time-boxed

**Scope out**

- Homebrew / scoop / AUR publishing (tap config stubbed but not live)
- Auto-update checks
- Telemetry (not planned)

**Files created/modified**

```
internal/cli/testscript_test.go
internal/cli/testdata/script/*.txtar
internal/cli/completion.go
internal/cli/root.go                   (register completion command)
.goreleaser.yaml
.github/workflows/release.yml
README.md
SECURITY.md
```

**Tests**

- All testscript scenarios pass
- `make build` includes ldflags and produces a correct `whzbox version`
- `./whzbox completion bash > /tmp/whzbox.bash && bash -c "source /tmp/whzbox.bash"` completes cleanly

**Done when**

- `make release-check` (or equivalent) runs goreleaser in dry-run mode cleanly
- README has install + quickstart + example output
- Pushing a `v0.1.0` tag would produce a GitHub release (actual release gated on user approval)
- All testscript scenarios green

---

## Risks & contingencies

**Risk: `/auth/exchange` doesn't behave like a refresh endpoint for expired tokens.**
Mitigation: Phase 4 includes a 15-minute probe at the start. If exchange only works for not-yet-expired tokens, v1 ships with "silent refresh for near-expiry, prompt for expired" — which is still much better than "prompt every time." The Service is already structured for this: near-expiry is the proactive trigger, expiry-by-reactive-401 is a bonus.

**Risk: Whizlabs has no "active sandbox" query endpoint.**
Mitigation: Phase 6 probes this early. If no clean endpoint exists, `Status` ships showing session info only and prints `(active sandbox state not available)` for now. Issue filed for follow-up.

**Risk: IAM propagation retry loop in awsverify hits the ctx deadline.**
Mitigation: `Service.Create` uses the command's context. We set a generous default timeout (90s) inside the verifier but respect ctx cancellation. If users have sub-minute CI timeouts, they'll need `--duration` anyway and the verify failure is informative.

**Risk: Whizlabs changes their API shape without warning.**
Mitigation: adapter tests use inline fixtures, so changes require a code update — no silent drift. Secret redaction is a test, so accidental logging of new response fields won't slip through.

**Risk: `charm.land/*/v2` import paths change again before we ship.**
Mitigation: pin exact versions in `go.mod`. If the module path changes, the skill's v2 path was validated 2026-04-10 — rerun `charmbracelet` skill before phase 2.

---

## Out of scope for this plan (deferred)

- Machine-readable output (`--json`)
- `~/.aws/credentials` profile writes, `--export`, subshell launch
- Auto-renewal (`whzbox create aws --renew`)
- GCP, Azure, Hyper-V, Power BI, Jupyter providers — interfaces exist, adapters don't
- `whzbox list` for historical sandboxes
- Multi-sandbox concurrency (one per user per command)
- Telemetry / crash reporting
- Homebrew tap, scoop bucket, AUR PKGBUILD
- Config file support
- MFA (Whizlabs doesn't currently require it on login)
- Proxy / custom CA support
