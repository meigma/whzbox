# whzbox

A small Go CLI that automates the full lifecycle of cloud sandboxes on [Whizlabs](https://www.whizlabs.com/):
sign in, provision, fetch usable credentials, verify them against the real cloud, and tear everything down —
in a single command.

v1 ships with AWS support. The core is provider-agnostic, so GCP / Azure / Hyper-V are straightforward
follow-on adapters.

```
$ whzbox create aws

INFO  creating aws sandbox            duration=1h
INFO  registering sandbox with account
INFO  credentials verified            account=966674480408

╭─ AWS sandbox ready ──────────────────────────────────────────────╮
│                                                                  │
│  Account                111111111111                             │
│  User                   Whiz_User_341128.68018890                │
│  ARN                    arn:aws:iam::111111111111:user/Whiz_...  │
│  Region                 us-east-1                                │
│                                                                  │
│  Expires                2026-04-11 06:01:06 UTC (in 59m 58s)     │
│                                                                  │
│  Console                https://111111111111.signin.aws.amaz... │
│  Username               Whiz_User_341128.68018890                │
│  Password               4f23d035-90bf-4eb8-b1ac-14ec1b4f8225     │
│                                                                  │
│  AWS_ACCESS_KEY_ID      AKIA6CESJ7UMMW2VYNNH                     │
│  AWS_SECRET_ACCESS_KEY  i9m236UZXNWdU1EEAGskc9s+1Xa+xQYajhM7C... │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯

Destroy with:  whzbox destroy
```

---

## Install

```sh
# Homebrew (macOS/Linux)
brew install meigma/tap/whzbox

# Scoop (Windows)
scoop bucket add meigma https://github.com/meigma/scoop-bucket
scoop install whzbox

# From source (requires Go 1.26+)
go install github.com/meigma/whzbox/cmd/whzbox@latest

# Or clone + build
git clone https://github.com/meigma/whzbox.git
cd whzbox
make build
./whzbox --help
```

Release binaries for Linux and macOS (amd64 + arm64) and Windows (amd64) are attached to each tagged release on GitHub.

## Quickstart

```sh
# 1. Sign in. Credentials are saved encrypted-at-rest only by filesystem perms
#    (0600) under $XDG_STATE_HOME/whzbox/state.json.
whzbox login

# 2. Spin up a sandbox. whzbox prints credentials to stdout once they're verified.
whzbox create aws --duration 2h

# 3. Use the credentials however you like.
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
aws sts get-caller-identity

# 4. Tear it down.
whzbox destroy --yes
```

On a machine that's already logged in, the whole flow is just:

```sh
whzbox create aws && echo "..."
whzbox destroy --yes
```

## Commands

| Command | What it does |
|---|---|
| `whzbox login` | Prompt for email + password, exchange them for an access/refresh pair, persist to disk |
| `whzbox logout` | Clear the cached session |
| `whzbox create <provider>` | Provision a sandbox, register ownership, verify credentials, render them |
| `whzbox destroy` | Tear down the currently active sandbox |
| `whzbox exec <provider> [-- cmd args...]` | Run a command (or drop into `$SHELL`) with the cached sandbox's env vars set |
| `whzbox list` | List cached sandboxes (read-only) |
| `whzbox status` | Show the cached session (read-only, never prompts) |
| `whzbox version` | Print build info |
| `whzbox completion <shell>` | Generate shell completions for bash / zsh / fish / powershell |

Run `whzbox <command> --help` for per-command flags.

### Global flags

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `--log-level` | `WHZBOX_LOG_LEVEL` | `info` | Absolute log level (debug / info / warn / error) |
| `-v`, `--verbose` | — | `0` | Repeatable. `-v` = info, `-vv` = debug |
| `-q`, `--quiet` | — | `false` | Errors only |
| `--no-color` | `WHZBOX_NO_COLOR`, `NO_COLOR` | auto | Disable coloured output |
| `--yes` | — | `false` | Skip confirmation prompts (required for non-interactive `destroy`) |
| `--json` | `WHZBOX_JSON` | `false` | Emit machine-readable JSON instead of styled output (applies to `create`, `list`) |

### create flags

| Flag | Default | Purpose |
|---|---|---|
| `--duration` | `1h` | Sandbox lifetime, between 1h and 9h |

### exec flags

| Flag | Default | Purpose |
|---|---|---|
| `-s`, `--shell` | `false` | Treat the single command argument as a shell string (passed to `sh -c`) |

### Running commands in a sandbox

`whzbox exec` injects the cached sandbox's credentials into a child
process's environment:

```sh
# run a single argv (no shell involved)
whzbox exec aws -- aws sts get-caller-identity

# run a shell pipeline (-s wraps in sh -c)
whzbox exec aws -s "aws s3 ls | head"

# drop into an interactive subshell with AWS_* set
whzbox exec aws
```

AWS sandboxes set `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`,
`AWS_REGION`, and `AWS_DEFAULT_REGION`. The child's exit code is
propagated, so `whzbox exec aws -- false; echo $?` prints `1`.

`whzbox exec` fails if no sandbox is cached for the provider; the error
message tells you to run `whzbox create <provider>` first.

### Environment variables

| Variable | Purpose |
|---|---|
| `WHZBOX_STATE_DIR` | Override the state directory (default: `$XDG_STATE_HOME/whzbox` → `~/.local/state/whzbox`) |
| `WHZBOX_LOG_LEVEL` | Same as `--log-level` |
| `WHZBOX_NO_COLOR` | Same as `--no-color` |

## State & secrets

Tokens are stored in a single JSON file at `$XDG_STATE_HOME/whzbox/state.json` (default: `~/.local/state/whzbox/state.json`).
The file is written with mode `0600` and the containing directory with mode `0700`. whzbox refuses to load the file
if its permissions are wider than `0600`, as a defence against sloppy local backup / sync tools.

If the file is ever corrupted (bad JSON, unknown schema version, unparsable timestamps), whzbox self-heals by deleting
it and prompting for a fresh login on the next invocation.

The refresh token is valid for 7 days and is used to silently extend the session when the access token nears expiry
(10 minutes before the end, by default). Once both tokens are expired, whzbox falls through to the interactive
sign-in prompt.

## Exit codes

Scripts can branch on these:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic error |
| `2` | Authentication error (invalid credentials, session fully expired) |
| `3` | Provider error (sandbox create/destroy failed, STS verification failed) |
| `4` | User aborted an interactive prompt |
| `5` | No interactive terminal available (run `whzbox login` from a real shell, or pre-populate the state file) |

## Scripting

whzbox works cleanly in pipelines — credentials go to **stdout**, logs and prompts to **stderr**:

```sh
# Extract fields from the JSON output with jq
secret=$(whzbox create aws --json | jq -r .credentials.secret_key)
account=$(whzbox list --json | jq -r '.[] | select(.kind=="aws") | .identity.account')

# Non-interactive destroy with an explicit confirmation bypass
whzbox destroy --yes
```

For fully non-interactive environments (CI, cron) you must have a valid cached session — log in once from a
real terminal, then run `whzbox` commands against the resulting state file.

## Architecture

whzbox is a small hexagonal Go application:

```
cmd/whzbox/                  main entrypoint
internal/
  cli/                       cobra commands + dependency container
  core/
    session/                 auth domain (pure, no I/O)
    sandbox/                 sandbox domain (pure, no I/O)
    clock/                   time abstraction
  adapters/
    whizlabs/                HTTP client implementing IdentityProvider + Manager
    awsverify/               STS verifier implementing sandbox.Provider
    xdgstore/                Filesystem TokenStore
    huhprompt/               Interactive credential prompt
  config/                    Viper-backed typed configuration
  logging/                   slog.Logger + charmbracelet/log handler
  ui/                        Shared theme + rendering
```

The core packages have zero non-stdlib imports and are 100% unit-test covered. A `depguard` lint rule enforces
the dependency direction on every change.

See `DESIGN.md` for the full design doc and `PLAN.md` for the phased implementation plan.

## Contributing

Development requires Go 1.26+ and `golangci-lint` v2.

```sh
make build    # build ./whzbox with ldflags
make test     # go test -race ./...
make lint     # golangci-lint run
make tidy     # go mod tidy
```

Integration tests against the real Whizlabs API live under a build tag:

```sh
go test -tags integration -count=1 ./internal/adapters/whizlabs/...
```

They require a repo-local `.env` with `USERNAME` and `PASSWORD`. The `.env` file is gitignored; do not commit it.

## License

Licensed under either of [Apache License, Version 2.0](LICENSE-APACHE) or [MIT License](LICENSE-MIT) at your option.
