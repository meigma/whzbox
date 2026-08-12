# whzbox

`whzbox` is a Go CLI for Whizlabs cloud sandboxes. It creates, caches, lists, and destroys AWS and GCP sandboxes.

AWS includes programmatic credentials for `whzbox exec`. GCP provides browser-console credentials only.

## Install

Install from [GitHub Releases](https://github.com/meigma/whzbox/releases), Homebrew, Scoop, or from source.

```sh
# Homebrew (macOS/Linux)
brew install meigma/tap/whzbox

# Scoop (Windows)
scoop bucket add meigma https://github.com/meigma/scoop-bucket
scoop install whzbox

# ghd (macOS/Linux releases produced by the current pipeline)
ghd install meigma/whzbox/whzbox

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

For a GCP console sandbox:

```sh
whzbox create gcp
# Open the displayed console URL and sign in with the displayed username and password.
whzbox destroy --yes
```

Common commands:

- `whzbox create aws` creates an AWS sandbox with console and programmatic credentials.
- `whzbox create gcp` creates a GCP sandbox with console credentials and project metadata.
- `whzbox list` shows cached sandboxes from local state.
- `whzbox exec aws -- <command>` runs a command with sandbox credentials injected.
- `whzbox status` shows the cached session.
- `whzbox destroy` tears down the active sandbox.

Run `whzbox <command> --help` for per-command flags.

## Documentation

The root README stays intentionally short. The rendered site is at
[whzbox.meigma.dev](https://whzbox.meigma.dev), with source under [`docs/docs/`](docs/docs):

- [Documentation overview](docs/docs/index.md)
- [AI agent skill](SKILL.md)
- [Tutorial: create, use, and destroy your first AWS sandbox](docs/docs/tutorials/create-use-and-destroy-your-first-aws-sandbox.md)
- [How-to guides](docs/docs/how-to)
- [Command reference](docs/docs/reference)
- [Explanation](docs/docs/explanation)

A repo-local [SKILL.md](SKILL.md) is available for AI agents that need a concise, task-oriented guide to using the CLI before drilling into the full docs.

A few behaviors that are easy to miss are documented in detail there:

- `create aws` and `create gcp` can reuse an unexpired cached sandbox.
- Whizlabs permits one active sandbox, so destroy the current provider before creating another.
- `exec` supports AWS only. `exec gcp` explains the console-only limitation.
- `exec`, `list`, and `status` read local state; they do not talk to Whizlabs.
- `--json` is implemented by `create` and `list`.

Preview the docs site locally:

```sh
cd docs
mise exec -- uv sync --locked
mise exec -- uv run --locked mkdocs serve
```

## Development

Prerequisites:

- [mise](https://mise.jdx.dev) for the repository's locked toolchain

Common tasks:

```sh
mise install
mise exec -- moon run root:check
```

Repository settings are declared in `.github/repository-settings.toml`. Preview
supported GitHub changes before applying them:

```sh
mise exec -- uv run .github/scripts/configure_github_repo.py plan --repo meigma/whzbox
```

Run the Whizlabs integration tests only when you have real credentials in a repo-local `.env`:

```sh
go test -tags integration -timeout 5m ./internal/adapters/whizlabs/...
```

Those tests read `USERNAME` and `PASSWORD` from `.env`.

## License

Dual-licensed under [Apache-2.0](LICENSE-APACHE) or [MIT](LICENSE-MIT).
