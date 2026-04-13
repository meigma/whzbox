# whzbox

`whzbox` is a Go CLI for Whizlabs cloud sandboxes. It signs in to Whizlabs, creates or reuses a sandbox, verifies the returned credentials, and helps you run commands against that sandbox until you destroy it.

AWS is the only supported provider today.

## Install

Install from [GitHub Releases](https://github.com/meigma/whzbox/releases), Homebrew, Scoop, or from source.

```sh
# Homebrew (macOS/Linux)
brew install meigma/tap/whzbox

# Scoop (Windows)
scoop bucket add meigma https://github.com/meigma/scoop-bucket
scoop install whzbox

# Go
go install github.com/meigma/whzbox/cmd/whzbox@latest
```

Build from a checkout:

```sh
go build -o whzbox ./cmd/whzbox
./whzbox --help
```

## Quick Start

```sh
whzbox login
whzbox create aws
whzbox exec aws -- aws sts get-caller-identity
whzbox destroy --yes
```

Common commands:

- `whzbox create aws` creates an AWS sandbox and prints credentials.
- `whzbox list` shows cached sandboxes from local state.
- `whzbox exec aws -- <command>` runs a command with sandbox credentials injected.
- `whzbox status` shows the cached session.
- `whzbox destroy` tears down the active sandbox.

Run `whzbox <command> --help` for per-command flags.

## Documentation

The root README stays intentionally short. Detailed documentation lives under [`docs/`](docs):

- [Documentation overview](docs/index.md)
- [AI agent skill](SKILL.md)
- [Tutorial: create, use, and destroy your first AWS sandbox](docs/tutorials/create-use-and-destroy-your-first-aws-sandbox.md)
- [How-to guides](docs/how-to)
- [Command reference](docs/reference)
- [Explanation](docs/explanation)

A repo-local [SKILL.md](SKILL.md) is available for AI agents that need a concise, task-oriented guide to using the CLI before drilling into the full docs.

A few behaviors that are easy to miss are documented in detail there:

- `create aws` can reuse an unexpired cached sandbox.
- `exec`, `list`, and `status` read local state; they do not talk to Whizlabs.
- `--json` is implemented by `create` and `list`.

Preview the docs site locally:

```sh
cd docs
npm ci
npm run start
```

## Development

Prerequisites:

- Go 1.26.2 or later
- Node 20 or later for the docs site
- `golangci-lint` for local linting

Common tasks:

```sh
go build -o whzbox ./cmd/whzbox
go test -race ./...
golangci-lint run

cd docs
npm ci
npm run build
```

Run the Whizlabs integration tests only when you have real credentials in a repo-local `.env`:

```sh
go test -tags integration -timeout 5m ./internal/adapters/whizlabs/...
```

Those tests read `USERNAME` and `PASSWORD` from `.env`.

## License

Dual-licensed under [Apache-2.0](LICENSE-APACHE) or [MIT](LICENSE-MIT).
