# whzbox — Design

**Module:** `github.com/meigma/whzbox`
**Binary:** `whzbox`
**Purpose:** A single command-line tool that spins up on-demand cloud sandboxes through Whizlabs, fetches their credentials, and verifies they work — starting with AWS, extensible to GCP/Azure/etc.

## 1. Goals & Non-goals

**Goals**

- One command to go from zero to working AWS credentials: `whzbox create aws`
- Persistent, low-friction auth — the user logs in once and stays logged in until the refresh token expires
- Clean, testable hexagonal core with provider-agnostic domain logic
- Beautiful, accessible terminal UX (huh prompts, lipgloss framing, charm log)
- Scriptable — a pipeline can run `whzbox create aws` without a TTY, as long as credentials are cached

**Non-goals (v1)**

- Machine-readable output (`--json`) — deferred
- Writing to `~/.aws/credentials`, `--export`, subshell launch — deferred (stdout text only)
- Multiple concurrent sandboxes per user — designed-for but not exposed
- Config files — only flags and env vars
- Providers beyond AWS — interfaces will exist, only AWS is wired up

## 2. User experience

All examples assume a 120-column terminal with color support.

### 2.1. First run (no cached session)

```
$ whzbox create aws

 ┃ Sign in to Whizlabs
 ┃
 ┃ Email
 ┃ > josh@example.com
 ┃
 ┃ Password
 ┃ > ••••••••••••••••••••
 ┃
 ┃  Submit   Cancel

INFO  logging in                       user=josh@example.com
INFO  session established              expires=2026-04-11T20:00:00Z
INFO  creating aws sandbox             duration=1h
INFO  registering sandbox with account
INFO  verifying credentials            waiting for IAM propagation (attempt 2)
INFO  credentials verified             account=966674480408

╭─ AWS sandbox ready ──────────────────────────────────────────────────╮
│                                                                      │
│  Account       966674480408                                          │
│  User          Whiz_User_341128.45467158                             │
│  ARN           arn:aws:iam::966674480408:user/Whiz_User_341128...    │
│  Region        us-east-1                                             │
│  Expires       2026-04-11 06:01:06 UTC  (in 59m 58s)                 │
│                                                                      │
│  Console       https://966674480408.signin.aws.amazon.com/console    │
│  Username      Whiz_User_341128.45467158                             │
│  Password      4f23d035-90bf-4eb8-b1ac-14ec1b4f8225                  │
│                                                                      │
│  AWS_ACCESS_KEY_ID      AKIAEXAMPLE000000000                         │
│  AWS_SECRET_ACCESS_KEY  wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY     │
│                                                                      │
╰──────────────────────────────────────────────────────────────────────╯

Destroy with:  whzbox destroy
```

### 2.2. Second run (cached, valid session)

```
$ whzbox create aws --duration 3

INFO  session loaded from cache        expires=2026-04-11T20:00:00Z
INFO  creating aws sandbox             duration=3h
INFO  registering sandbox with account
INFO  credentials verified             account=461364615260

╭─ AWS sandbox ready ──────────────────────────────────────────────────╮
│  Account   461364615260                                              │
│  ...                                                                 │
╰──────────────────────────────────────────────────────────────────────╯
```

### 2.3. Cached but expired session → silent refresh

```
$ whzbox create aws

DEBUG  session loaded from cache       expires=2026-04-10T18:00:00Z  (expired)
DEBUG  refreshing session
INFO   session refreshed               expires=2026-04-12T04:30:00Z
INFO   creating aws sandbox            duration=1h
...
```

### 2.4. Cached session unrefreshable → inline re-prompt

```
$ whzbox create aws

DEBUG  session loaded from cache
WARN   session refresh failed          err="refresh token expired"

 ┃ Session expired — please sign in again
 ┃
 ┃ Email
 ┃ > josh@example.com
 ┃ ...
```

### 2.5. `whzbox status`

```
$ whzbox status

 Session
   Email          josh@example.com
   Expires        2026-04-11 20:00:00 UTC  (in 14h 32m)
   Refreshable    yes, until 2026-04-18

 Active sandbox
   Kind           aws
   Account        966674480408
   Started        2026-04-11 05:00:19 UTC
   Expires        2026-04-11 06:00:19 UTC  (in 45m 12s)
```

When nothing is active:

```
$ whzbox status

 Session
   Email          josh@example.com
   Expires        2026-04-11 20:00:00 UTC  (in 14h 32m)

 Active sandbox
   (none)
```

### 2.6. `whzbox destroy`

```
$ whzbox destroy

 ┃ Destroy AWS sandbox 966674480408?
 ┃
 ┃ • All resources in the sandbox will be deleted
 ┃ • This cannot be undone
 ┃
 ┃  Yes, destroy   Cancel

INFO  sandbox destroyed                account=966674480408
```

With `--yes` or piped input: skip confirmation.

### 2.7. `whzbox login` (explicit)

```
$ whzbox login

 ┃ Sign in to Whizlabs
 ┃
 ┃ Email
 ┃ > josh@example.com
 ┃
 ┃ Password
 ┃ > ••••••••••••••••••••

INFO  session established              expires=2026-04-11T20:00:00Z
```

### 2.8. Error: bad password

```
$ whzbox login

 ┃ Sign in to Whizlabs
 ┃ ...

ERROR  login failed                    err="invalid credentials"
exit status 2
```

### 2.9. Error: non-TTY, no cached session

```
$ whzbox create aws < /dev/null

ERROR  no cached session and stdin is not a terminal
  hint: run `whzbox login` from an interactive shell first, or set WHZBOX_EMAIL and WHZBOX_PASSWORD
exit status 5
```

### 2.10. Help

```
$ whzbox --help

whzbox — spin up cloud sandboxes from Whizlabs

Usage:
  whzbox [command]

Available Commands:
  create       Create a new sandbox
  destroy      Destroy the active sandbox
  status       Show session and active sandbox
  login        Sign in to Whizlabs
  logout       Clear the cached session
  version      Print version information
  completion   Generate shell completion scripts
  help         Help about any command

Global Flags:
      --log-level string   debug, info, warn, error (default "info")
  -v, --verbose count      increase log verbosity (repeat for more)
  -q, --quiet              suppress non-essential output
      --no-color           disable colored output
      --yes                assume yes for all confirmations
  -h, --help               help for whzbox
      --version            version for whzbox

Environment:
  WHZBOX_EMAIL         email (skips prompt)
  WHZBOX_PASSWORD      password (skips prompt — prefer stdin)
  WHZBOX_LOG_LEVEL     equivalent to --log-level
  WHZBOX_NO_COLOR      equivalent to --no-color
  WHZBOX_STATE_DIR     override state directory (default $XDG_STATE_HOME/whzbox)

Use "whzbox [command] --help" for more information about a command.
```

## 3. Command surface

| Command | Synopsis |
|---|---|
| `whzbox login` | Prompt for email/password, exchange for tokens, save to state |
| `whzbox logout` | Delete the cached session file |
| `whzbox create <provider>` | Create a sandbox (provider = `aws` for v1). Auto-login if no cached session |
| `whzbox destroy` | Destroy the user's active sandbox |
| `whzbox status` | Show session validity + active sandbox (if any) |
| `whzbox version` | Print version, commit, build time |
| `whzbox completion [bash\|zsh\|fish\|powershell]` | Generate completions |
| `whzbox help [command]` | Help |

**Global flags** (persistent)

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `--log-level` | `WHZBOX_LOG_LEVEL` | `info` | Absolute level. Overrides `-v`. |
| `-v`, `--verbose` | — | `0` | Relative level. `-v` → info, `-vv` → debug, `-vvv` → debug + wire logs |
| `-q`, `--quiet` | — | `false` | Only errors |
| `--no-color` | `WHZBOX_NO_COLOR`, `NO_COLOR` | auto | Force-disable color |
| `--yes` | — | `false` | Skip confirmation prompts |

**`create` flags**

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `--duration` | — | `1h` | Sandbox lifetime; 1h–9h for AWS |

## 4. Configuration

### 4.1. Precedence

```
flags > env vars > defaults
```

(No config file, per product direction.)

### 4.2. Env var prefix

All env vars are prefixed `WHZBOX_` to avoid colliding with native `AWS_*` variables and other tools.

### 4.3. Viper wiring

A single `*viper.Viper` instance created in the app container, configured with:

```go
vp.SetEnvPrefix("WHZBOX")
vp.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
vp.AutomaticEnv()
```

Config is loaded in the root `PersistentPreRunE` hook so local subcommand flags are visible to `BindPFlags(cmd.Flags())`. A strongly typed `config.Config` struct is unmarshaled from Viper and passed through the dependency container — call sites never touch Viper directly.

## 5. State

### 5.1. Location

`$XDG_STATE_HOME/whzbox/state.json` (overridable via `WHZBOX_STATE_DIR`).

Fallback on systems that don't set `XDG_STATE_HOME`: `~/.local/state/whzbox/state.json`. On macOS we still prefer `~/.local/state/whzbox` over `~/Library/Application Support` to match other modern CLIs (gh, fly, turbo); it's conventional even though not strictly Apple-blessed.

### 5.2. Permissions

- Directory: `0700`
- File: `0600`
- Atomic writes via temp file + `os.Rename` (same filesystem)

### 5.3. Schema

```json
{
  "version": 1,
  "whizlabs": {
    "user_email": "josh@example.com",
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "access_token_expires_at": "2026-04-11T20:00:00Z",
    "refresh_token_expires_at": "2026-04-18T04:30:00Z"
  }
}
```

`version` allows schema migrations without user disruption. Unknown-version files are treated as corrupt and user is asked to re-login.

Note: we deliberately do *not* persist the play.whizlabs.com-scoped JWT — it's cheap to re-derive from the main JWT via the auth exchange, and it has a different lifetime than the main JWT. Keeping one source of truth simplifies expiry logic.

## 6. Authentication flow

### 6.1. Session lifecycle

```
           ┌──────────────────────────┐
           │  CLI command needs auth  │
           └────────────┬─────────────┘
                        │
                        ▼
              ┌──────────────────┐
              │  Load state file │
              └─────┬────────┬───┘
                    │        │
           cache hit│        │no cache
                    ▼        ▼
          ┌──────────────┐  ┌──────────────────┐
          │ Token fresh? │  │ Interactive TTY? │
          └─┬──────────┬─┘  └───┬───────────┬──┘
            │ yes      │no      │yes        │no
            │          │        │           │
            │          ▼        ▼           ▼
            │    ┌───────────┐ ┌─────────┐ ┌──────────┐
            │    │ Try       │ │ huh     │ │ ERROR    │
            │    │ refresh   │ │ prompt  │ │ exit 5   │
            │    └─────┬─────┘ └────┬────┘ └──────────┘
            │          │            │
            │   ok │err│            │
            │      │   │            │
            │      │   ▼            │
            │      │  ┌──────────┐  │
            │      │  │ huh      │◄─┘
            │      │  │ prompt   │
            │      │  └────┬─────┘
            │      │       │
            │      ▼       ▼
            │   ┌──────────────┐
            │   │ POST login   │
            │   └──────┬───────┘
            │          │
            ▼          ▼
         ┌───────────────────┐
         │ Save new tokens   │
         │ Return to caller  │
         └───────────────────┘
```

### 6.2. Refresh

The Whizlabs API exposes `GET /Stage/auth/exchange` with the current access token as `Authorization: Bearer`. This returns a fresh access/refresh pair. We call it:

- **Proactively**: if the stored access token expires in < 10 minutes
- **Reactively**: if a `401 Unauthorized` bubbles up from any downstream call

If exchange fails (e.g., refresh token age > 7d), we fall through to the prompt (or fail if non-interactive).

The exchange endpoint behavior beyond what we've already captured (successful call with Bearer auth returning fresh tokens) needs one 5-minute probe during implementation to confirm it accepts a nearly-expired token. If it doesn't, v1 falls back to re-prompt only and we open an issue.

### 6.3. Play token derivation

The play.whizlabs.com JWT is derived from the main JWT via `POST /api/web/login/user-authentication` on demand, once per command invocation. Not persisted.

## 7. Package layout

```
whzbox/
├── cmd/whzbox/                 # main entrypoint
│   └── main.go                 # signal.NotifyContext + ExecuteContext
├── internal/
│   ├── cli/                    # cobra wiring — commands, flags, DI container
│   │   ├── app.go              # *App: deps container (config, logger, services)
│   │   ├── root.go             # NewRootCommand(*App)
│   │   ├── login.go            # NewLoginCommand(*App)
│   │   ├── logout.go
│   │   ├── create.go
│   │   ├── destroy.go
│   │   ├── status.go
│   │   └── version.go
│   ├── core/                   # domain + use cases (no I/O)
│   │   ├── session/
│   │   │   ├── session.go      # Tokens, Session value types
│   │   │   ├── service.go      # Session.EnsureValid, Login, Refresh
│   │   │   └── ports.go        # IdentityProvider, TokenStore, Prompt
│   │   ├── sandbox/
│   │   │   ├── sandbox.go      # Sandbox, Credentials, Identity, Kind
│   │   │   ├── service.go      # Sandbox.Create, Destroy, Status
│   │   │   └── ports.go        # Manager, Provider
│   │   └── clock/
│   │       └── clock.go        # Clock interface + Real impl (for tests)
│   ├── adapters/               # port implementations — touch the outside world
│   │   ├── whizlabs/           # HTTP client → IdentityProvider + Manager
│   │   │   ├── client.go       # base http.Client, json helpers
│   │   │   ├── auth.go         # Login, Refresh (IdentityProvider)
│   │   │   └── sandbox.go      # Create, Commit, Destroy, Active (Manager)
│   │   ├── awsverify/          # AWS STS → CredentialVerifier for kind=aws
│   │   │   └── sts.go
│   │   ├── xdgstore/           # XDG state file → TokenStore
│   │   │   └── store.go
│   │   └── huhprompt/          # huh forms → Prompt
│   │       └── prompt.go
│   ├── config/
│   │   └── config.go           # Config struct + Load(*viper.Viper)
│   ├── logging/
│   │   └── logger.go           # slog.Logger backed by charm log handler
│   └── ui/
│       ├── theme.go            # shared lipgloss palette + huh theme
│       ├── render.go           # credential box, status view, etc.
│       └── tty.go              # isInteractive(), color detection
└── go.mod
```

**Dependency direction** (strictly inward — hexagonal):

```
  cmd/whzbox  ──▶  internal/cli  ──▶  internal/core  ◀──  internal/adapters
                                         ▲
                                         │
                                      ports
```

- `internal/core` imports nothing from `cli`, `adapters`, `config`, `logging`, or `ui`.
- `internal/adapters` imports `core` (to implement its port interfaces) plus outside libs (`net/http`, `aws-sdk-go-v2`, etc.), nothing else from `internal/`.
- `internal/cli` imports `core` (to call use cases) and `adapters` (to construct concrete implementations for injection). This is the only place concrete types are wired.

## 8. Domain model

### 8.1. Session (auth)

```go
// internal/core/session/session.go
package session

type Tokens struct {
    AccessToken           string
    RefreshToken          string
    AccessTokenExpiresAt  time.Time
    RefreshTokenExpiresAt time.Time
    UserEmail             string
}

func (t Tokens) AccessValid(now time.Time) bool
func (t Tokens) AccessNearExpiry(now time.Time, window time.Duration) bool
func (t Tokens) Refreshable(now time.Time) bool
```

### 8.2. Session ports

```go
// internal/core/session/ports.go
package session

type IdentityProvider interface {
    Login(ctx context.Context, email, password string) (Tokens, error)
    Refresh(ctx context.Context, current Tokens) (Tokens, error)
}

type TokenStore interface {
    Load(ctx context.Context) (Tokens, bool, error) // (tokens, found, err)
    Save(ctx context.Context, t Tokens) error
    Clear(ctx context.Context) error
}

type Prompt interface {
    Credentials(ctx context.Context, defaultEmail string) (email, password string, err error)
}

var ErrPromptUnavailable = errors.New("no interactive terminal")
var ErrUserAborted      = errors.New("user aborted")
```

### 8.3. Session service

```go
// internal/core/session/service.go
type Service struct {
    provider IdentityProvider
    store    TokenStore
    prompt   Prompt
    clock    clock.Clock
    log      *slog.Logger
}

// EnsureValid returns a Tokens that is usable right now.
// It loads from the store, refreshes if near expiry, and prompts if necessary.
func (s *Service) EnsureValid(ctx context.Context) (Tokens, error)

// Login is explicit: always prompts (or uses provided creds), replaces stored tokens.
func (s *Service) Login(ctx context.Context) (Tokens, error)

// Logout clears the store.
func (s *Service) Logout(ctx context.Context) error
```

### 8.4. Sandbox

```go
// internal/core/sandbox/sandbox.go
package sandbox

type Kind string

const (
    KindAWS Kind = "aws"
    // KindGCP, KindAzure, KindHyperV, ... (future)
)

type Credentials struct {
    AccessKey string
    SecretKey string
    // Other providers can extend: ServiceAccountJSON, TenantID, etc.
    // For now, AWS-style suffices. Future providers get typed fields.
}

type Identity struct {
    Account string   // AWS account id / GCP project / Azure subscription
    UserID  string   // canonical user id
    ARN     string   // provider-specific resource identifier
    Region  string
}

type Console struct {
    URL      string
    Username string
    Password string
}

type Sandbox struct {
    Kind        Kind
    Slug        string     // whizlabs slug, e.g. "aws-sandbox"
    Credentials Credentials
    Console     Console
    Identity    Identity   // filled in after Verify
    StartedAt   time.Time
    ExpiresAt   time.Time
}
```

### 8.5. Sandbox ports

```go
// internal/core/sandbox/ports.go
package sandbox

// Manager talks to the upstream sandbox broker (Whizlabs).
// Provider-agnostic: sandbox kind is identified by slug.
type Manager interface {
    Create(ctx context.Context, tokens session.Tokens, slug string, duration time.Duration) (*Sandbox, error)
    Commit(ctx context.Context, tokens session.Tokens, slug string, duration time.Duration) error
    Destroy(ctx context.Context, tokens session.Tokens) error
    Active(ctx context.Context, tokens session.Tokens) (*Sandbox, error)
}

// Provider knows the quirks of one sandbox kind: which slug to use,
// and how to verify the resulting credentials work against the real cloud.
type Provider interface {
    Kind() Kind
    Slug() string
    VerifyCredentials(ctx context.Context, creds Credentials) (Identity, error)
}
```

### 8.6. Sandbox service

```go
// internal/core/sandbox/service.go
type Service struct {
    session   *session.Service
    manager   Manager
    providers map[Kind]Provider
    clock     clock.Clock
    log       *slog.Logger
}

func (s *Service) Create(ctx context.Context, kind Kind, duration time.Duration) (*Sandbox, error) {
    prov, ok := s.providers[kind]
    if !ok { return nil, fmt.Errorf("unknown sandbox kind: %s", kind) }

    tokens, err := s.session.EnsureValid(ctx)
    if err != nil { return nil, err }

    sb, err := s.manager.Create(ctx, tokens, prov.Slug(), duration)
    if err != nil { return nil, err }
    sb.Kind = kind

    if err := s.manager.Commit(ctx, tokens, prov.Slug(), duration); err != nil {
        // best-effort cleanup
        _ = s.manager.Destroy(ctx, tokens)
        return nil, err
    }

    id, err := prov.VerifyCredentials(ctx, sb.Credentials)
    if err != nil { return nil, err }
    sb.Identity = id

    return sb, nil
}

func (s *Service) Destroy(ctx context.Context) error
func (s *Service) Status(ctx context.Context) (*Sandbox, error)
```

Adding a new sandbox kind later is two steps:
1. Write a `Provider` implementation (typically in `internal/adapters/<kind>verify/`)
2. Register it in `internal/cli/app.go`'s provider map

The Whizlabs `Manager` already handles all kinds — only the slug differs.

## 9. Adapters

### 9.1. `whizlabs.Client`

A thin HTTP client over `net/http` that:

- Implements `session.IdentityProvider` (login via `/Stage/auth/login`, refresh via `/Stage/auth/exchange`)
- Implements `sandbox.Manager` (create/commit/destroy/active via `play.whizlabs.com/api/web/play-sandbox/*`)
- Handles the main-JWT → play-JWT exchange transparently inside `Create`/`Commit`/`Destroy`/`Active`. Play JWT is not surfaced to the core.
- Uses `context.Context` for deadlines and cancellation on every request
- Returns typed errors the core can inspect (`ErrUnauthorized`, `ErrNotFound`, `ErrConflict`)

### 9.2. `awsverify.STSVerifier`

Implements `sandbox.Provider` for `KindAWS`:

```go
type STSVerifier struct {
    region       string
    maxAttempts  int
    retryBackoff time.Duration
}

func (v *STSVerifier) Kind() sandbox.Kind { return sandbox.KindAWS }
func (v *STSVerifier) Slug() string       { return "aws-sandbox" }

func (v *STSVerifier) VerifyCredentials(ctx context.Context, c sandbox.Credentials) (sandbox.Identity, error) {
    // aws-sdk-go-v2 STS GetCallerIdentity with retry loop for IAM propagation
    // (matches what the prototype proved)
}
```

### 9.3. `xdgstore.Store`

Implements `session.TokenStore` backed by a JSON file at `$XDG_STATE_HOME/whzbox/state.json`:

- `Load`: reads, json-decodes, returns `(Tokens, true, nil)` or `(_, false, nil)` if file doesn't exist; any parse error surfaces
- `Save`: writes atomic (temp file in same dir + rename), ensures `0600` and parent `0700`
- `Clear`: deletes the file

### 9.4. `huhprompt.Prompt`

Implements `session.Prompt` using `charm.land/huh/v2`:

- Returns `session.ErrPromptUnavailable` immediately if stdin or stderr is not a TTY (checked via the shared `ui.IsInteractive()`)
- Returns `session.ErrUserAborted` on ctrl-c (`huh.ErrUserAborted`)
- Uses the shared theme from `ui/theme.go` for visual consistency with other forms

## 10. CLI layer

### 10.1. Dependency container

```go
// internal/cli/app.go
type App struct {
    Config  config.Config
    Logger  *slog.Logger
    Clock   clock.Clock

    Session *session.Service
    Sandbox *sandbox.Service
}

func NewApp(vp *viper.Viper) (*App, error) {
    cfg, err := config.Load(vp)
    if err != nil { return nil, err }

    logger := logging.New(cfg)
    realClock := clock.Real{}

    // adapters
    whiz := whizlabs.NewClient(cfg.Whizlabs, logger)
    store, err := xdgstore.New(cfg.StateDir)
    if err != nil { return nil, err }
    prompt := huhprompt.New()

    sessionSvc := session.NewService(whiz, store, prompt, realClock, logger)

    awsProv := awsverify.New("us-east-1")
    sandboxSvc := sandbox.NewService(
        sessionSvc,
        whiz, // same client implements Manager
        map[sandbox.Kind]sandbox.Provider{
            sandbox.KindAWS: awsProv,
        },
        realClock,
        logger,
    )

    return &App{
        Config: cfg, Logger: logger, Clock: realClock,
        Session: sessionSvc, Sandbox: sandboxSvc,
    }, nil
}
```

### 10.2. Root command

Following cobra-viper best practice:

```go
// internal/cli/root.go
func NewRootCommand() *cobra.Command {
    vp := viper.New()
    var app *App

    rootCmd := &cobra.Command{
        Use:           "whzbox",
        Short:         "Spin up cloud sandboxes from Whizlabs",
        SilenceUsage:  true,
        SilenceErrors: true,
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            if err := initViper(vp, cmd); err != nil { return err }
            var err error
            app, err = NewApp(vp)
            return err
        },
    }

    // persistent flags → bound in initViper
    rootCmd.PersistentFlags().String("log-level", "info", "log level")
    rootCmd.PersistentFlags().CountP("verbose", "v", "increase verbosity")
    rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress non-essential output")
    rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
    rootCmd.PersistentFlags().Bool("yes", false, "skip confirmation prompts")

    rootCmd.AddCommand(
        newCreateCommand(&app),
        newDestroyCommand(&app),
        newStatusCommand(&app),
        newLoginCommand(&app),
        newLogoutCommand(&app),
        newVersionCommand(),
    )
    return rootCmd
}
```

Subcommand constructors take `**App` (pointer to pointer) so they can read it after `PersistentPreRunE` has populated it. Alternatively, each subcommand's `RunE` can call `NewApp(vp)` itself — cleaner but duplicates work across `PersistentPreRunE`. The double-pointer pattern is standard cobra idiom; we'll go with it.

### 10.3. Leaf commands

Thin — they just call into the service:

```go
// internal/cli/create.go
func newCreateCommand(app **App) *cobra.Command {
    var duration time.Duration

    cmd := &cobra.Command{
        Use:   "create <provider>",
        Short: "Create a new sandbox",
        Args:  cobra.ExactArgs(1),
        ValidArgs: []string{"aws"},
        RunE: func(cmd *cobra.Command, args []string) error {
            kind := sandbox.Kind(args[0])
            sb, err := (*app).Sandbox.Create(cmd.Context(), kind, duration)
            if err != nil { return err }
            ui.RenderSandbox(cmd.OutOrStdout(), sb)
            return nil
        },
    }
    cmd.Flags().DurationVar(&duration, "duration", time.Hour, "sandbox lifetime (1h–9h)")
    return cmd
}
```

### 10.4. Stream discipline

Strict separation, per [CLI skill guidance](./DESIGN.md):

- **stdout**: final credential box (`ui.RenderSandbox`), status view, version info — the "data" the user wants
- **stderr**: logs (charm log / slog), huh forms, progress messages, errors
- **stdin**: used only by huh prompts

Piping to `grep` or `tee` captures exactly the user-visible data, not the log chatter. Non-TTY stderr is still safe because charm log degrades color automatically.

## 11. Logging

### 11.1. Stack

`slog.Logger` as the universal logger type flowing through the core, backed by a `charm.land/log/v2` handler for pretty terminal output. The log package provides a `slog.Handler` implementation, so the API inside the core is pure stdlib `slog` while the output remains beautiful.

```go
// internal/logging/logger.go
func New(cfg config.Config) *slog.Logger {
    level := levelFromConfig(cfg)
    h := log.NewWithOptions(os.Stderr, log.Options{
        ReportTimestamp: cfg.LogLevel == slog.LevelDebug,
        Level:           toCharmLevel(level),
    })
    h.SetStyles(ui.LogStyles()) // shared palette
    return slog.New(h)
}
```

### 11.2. Verbosity mapping

| Flag pattern | slog level | What you see |
|---|---|---|
| `-q` | `ERROR` | errors only |
| default | `INFO` | step-by-step progress |
| `-v` | `INFO` | same as default (reserved for future use) |
| `-vv` | `DEBUG` | token lifecycle, api requests, retries |
| `-vvv` | `DEBUG` + HTTP wire log | full request/response bodies (with secret redaction) |

`--log-level` takes absolute precedence when set. `-v` is relative.

Secrets (access tokens, refresh tokens, AWS secret keys, passwords) are never logged. The whizlabs http client wraps requests with a middleware that redacts `Authorization` headers and secret-bearing body fields on log output.

## 12. Error handling & exit codes

Error types in the core:

```go
// internal/core/errors.go (or per-package)
var (
    ErrInvalidCredentials = errors.New("invalid credentials")      // → exit 2
    ErrSessionExpired     = errors.New("session expired")          // → exit 2
    ErrNoActiveSandbox    = errors.New("no active sandbox")        // → exit 3
    ErrVerificationFailed = errors.New("credential verification failed") // → exit 3
    ErrUserAborted        = errors.New("user aborted")             // → exit 4
    ErrNonInteractive     = errors.New("no interactive terminal")  // → exit 5
)
```

The root `main.go` maps errors to exit codes:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    err := cli.NewRootCommand().ExecuteContext(ctx)
    os.Exit(exitCode(err))
}

func exitCode(err error) int {
    switch {
    case err == nil:                              return 0
    case errors.Is(err, session.ErrInvalidCredentials),
         errors.Is(err, session.ErrSessionExpired): return 2
    case errors.Is(err, sandbox.ErrVerificationFailed),
         errors.Is(err, sandbox.ErrNoActiveSandbox): return 3
    case errors.Is(err, session.ErrUserAborted):  return 4
    case errors.Is(err, session.ErrPromptUnavailable): return 5
    default:                                      return 1
    }
}
```

Leaf commands return wrapped errors (`fmt.Errorf("create: %w", err)`) so `errors.Is` still walks the chain.

## 13. Testing strategy

The hexagonal split makes each layer mockable:

- **Core services** (`session.Service`, `sandbox.Service`) get unit tests against fake `IdentityProvider`, `TokenStore`, `Prompt`, `Manager`, `Provider`, `Clock` implementations. No I/O.
- **Whizlabs adapter** gets integration tests against `httptest.Server` with captured fixtures from the prototype's real traffic.
- **STS verifier** gets a unit test using a mocked STS interface (`aws-sdk-go-v2` supports this via the smithy middleware stack, or we define a tiny `stsClient` interface).
- **XDG store** gets filesystem tests in `t.TempDir()` covering perms, atomicity, and corrupt state.
- **CLI layer** gets end-to-end tests via `testscript` — real cobra command, fake adapters injected via a build-time hook or a test-only `NewTestApp` factory. `charm.land/huh/v2`'s accessible mode makes form-driven flows deterministic in tests.

Target: >80% coverage in `core/`, >60% overall, zero coverage required in `adapters/` for network-heavy paths but fixture coverage expected.

## 14. UI theme

A single shared palette in `internal/ui/theme.go` keeps huh forms, lipgloss frames, and charm log levels visually consistent.

```go
package ui

import (
    "charm.land/huh/v2"
    "charm.land/lipgloss/v2"
    "charm.land/log/v2"
)

var (
    Accent  = lipgloss.Color("#874BFD") // purple — primary
    Success = lipgloss.Color("#02BA84") // green
    Warn    = lipgloss.Color("#F2B033") // amber
    Danger  = lipgloss.Color("#EF4444") // red
    Dim     = lipgloss.Color("245")
)

func HuhTheme(isDark bool) *huh.Styles {
    t := huh.ThemeBase(isDark)
    t.Focused.Title = t.Focused.Title.Foreground(Accent).Bold(true)
    t.Focused.FocusedButton = t.Focused.FocusedButton.
        Foreground(lipgloss.Color("#FFFDF5")).
        Background(Accent)
    t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(Success)
    return t
}

func LogStyles() *log.Styles {
    s := log.DefaultStyles()
    s.Levels[log.InfoLevel]  = lipgloss.NewStyle().SetString("INFO").Foreground(Accent)
    s.Levels[log.WarnLevel]  = lipgloss.NewStyle().SetString("WARN").Foreground(Warn)
    s.Levels[log.ErrorLevel] = lipgloss.NewStyle().SetString("ERROR").Foreground(Danger)
    s.Levels[log.DebugLevel] = lipgloss.NewStyle().SetString("DEBUG").Foreground(Dim)
    return s
}

func SandboxFrame() lipgloss.Style {
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Success).
        Padding(1, 2)
}
```

## 15. Open questions (for later passes)

1. `--json` machine output — which commands, what schema?
2. `--export`, `--profile`, `--shell` output modes for `create`
3. Refresh endpoint validation — does `/auth/exchange` accept a token within N minutes after expiry? (5-minute probe during implementation)
4. Auto-renew option: `whzbox create aws --renew` that automatically extends the sandbox 5 minutes before expiry
5. `whzbox list` for showing historical sandboxes if Whizlabs exposes that
6. Proper GCP/Azure verifier implementations (trivial once the port exists)

## 16. Implementation order

Roughly, in merge-sized chunks:

1. Project skeleton: `cmd/whzbox`, `internal/{cli,core,adapters,config,logging,ui}` empty packages + `go.mod` tidy
2. Config + logging + theme (no commands yet) — `whzbox version` prints correctly
3. Session core + whizlabs auth adapter + xdg store — `whzbox login` / `logout` work
4. huh prompt adapter + TTY detection — `whzbox login` is pretty and errors cleanly non-interactively
5. Sandbox core + whizlabs manager adapter + aws verifier — `whzbox create aws` / `destroy` / `status` work
6. `ui.RenderSandbox` + `ui.RenderStatus` for the credential box and status view
7. End-to-end testscript tests
8. Docs + shell completion + goreleaser

Each step ends with a working binary, green tests, and a demonstrable user-facing change.
