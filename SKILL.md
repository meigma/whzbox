---
name: whzbox-cli
description: Use when the whzbox CLI is installed and you need to create, reuse, inspect, use, or destroy AWS, Azure, or GCP sandboxes. Covers login, create, Azure MFA, list, status, AWS exec, destroy, JSON output, and cache and session behavior.
---

# whzbox CLI

Use this skill when the `whzbox` CLI is installed and you need an AWS, Azure, or GCP sandbox through Whizlabs.

## Quick model

- `whzbox` is a Go CLI for Whizlabs cloud sandboxes.
- AWS provides console and programmatic credentials.
- Azure provides browser-console credentials and resource groups only. Console login requires `mfa azure`; `exec azure` is unavailable.
- GCP provides browser-console credentials and project metadata only. `exec gcp` is unavailable.
- `login` stores a cached Whizlabs session locally.
- `logout` clears the cached session but can leave cached sandboxes in place.
- `create aws` creates or reuses an AWS sandbox, verifies credentials, and prints them.
- `create azure` creates or reuses an Azure console sandbox and prints its login and resource groups.
- `create gcp` creates or reuses a GCP console sandbox and prints its login and project details.
- `list`, `status`, and `exec` are local-state commands. They do not talk to Whizlabs.
- `destroy` tears down the active sandbox upstream and clears cached sandbox entries locally.

The default goal for an agent is usually:

1. Reuse an active cached sandbox if one already exists.
2. Otherwise create a new AWS sandbox.
3. Run commands through `whzbox exec aws -- ...` when possible instead of manually copying credentials.
4. Destroy the sandbox only when the task or user asks for cleanup.

Whizlabs permits one active sandbox per user. Destroy the current sandbox before creating a different provider.

## Common workflow

Create or reuse a sandbox and run one command in it:

```sh
whzbox login
whzbox create aws
whzbox exec aws -- aws sts get-caller-identity
```

Inspect existing local state before creating a new sandbox:

```sh
whzbox list
whzbox status
```

Clean up when requested:

```sh
whzbox destroy --yes
```

## Command guidance

- `whzbox login`
  - Interactive only.
  - Replaces stored session tokens.
  - Fails in non-interactive environments.

- `whzbox create aws`
  - Main provisioning command.
  - Supports `--duration`.
  - Can reuse an unexpired cached sandbox for the same provider.
  - Supports `--json`.
  - Prefer this over trying to stitch credentials together from cached state yourself.

- `whzbox create gcp`
  - Creates or reuses a browser-console sandbox.
  - Currently supports the default `1h` duration only.
  - Supports `--json`.
  - Does not return a service account, access token, or other credential for `gcloud` or an SDK.

- `whzbox create azure`
  - Creates or reuses a browser-console sandbox.
  - Supports durations from `1h` through `3h`.
  - Supports `--json`.
  - Does not return a service principal, access token, or other credential for `az` or an SDK.

- `whzbox mfa azure`
  - Requires an unexpired cached Azure sandbox and a valid Whizlabs session.
  - At the Microsoft MFA prompt, select **Use a verification code** before entering the generated code.
  - Prints a fresh 30-second console MFA code to stdout.
  - Never caches the code.

- `whzbox logout`
  - Clears the cached session.
  - Can preserve cached sandboxes.

- `whzbox list`
  - Reads cached sandboxes only.
  - Includes expired cached sandboxes.
  - Supports `--json`.

- `whzbox status`
  - Reads the cached session only.
  - Does not show live sandbox state.
  - Does not implement JSON output.

- `whzbox exec aws -- <cmd>`
  - Runs a child process with cached sandbox credentials injected.
  - Fails if no active cached sandbox exists.
  - Use `-s` only when you need shell parsing.
  - Prefer this for AWS CLI calls and one-off commands.

- `whzbox exec gcp`
  - Always fails with a console-only explanation.

- `whzbox exec azure`
  - Always fails with a console-only explanation.

- `whzbox destroy`
  - Destroys the active sandbox upstream.
  - Prompts unless `--yes` is set.
  - In scripts, use `--yes`.

## Machine-readable output

Use JSON only with:

- `whzbox create aws --json`
- `whzbox create azure --json`
- `whzbox create gcp --json`
- `whzbox list --json`

Do not assume `--json` changes output for `status`, `login`, `logout`, `destroy`, `exec`, `mfa`, `version`, or `completion`.

For schemas and examples, read:

- `docs/reference/json-output-schemas.md`
- `docs/how-to/consume-create-and-list-json-output.md`

## Non-interactive use

For automation or scripted use:

- Assume `login` requires a real terminal.
- Prefer `create --json` and `list --json` when parsing output.
- Prefer `exec` when you only need to run a command in the sandbox.
- Use `destroy --yes`.
- Remember that `list`, `status`, and `exec` read local state and do not refresh or prompt.

Read:

- `docs/how-to/use-whzbox-in-non-interactive-scripts.md`

## Where to read next

Start here for orientation:

- `docs/index.md`
- `docs/tutorials/create-use-and-destroy-your-first-aws-sandbox.md`

Open command-specific reference when exact behavior matters:

- `docs/reference/commands/login.md`
- `docs/reference/commands/create.md`
- `docs/reference/commands/logout.md`
- `docs/reference/commands/list.md`
- `docs/reference/commands/status.md`
- `docs/reference/commands/exec.md`
- `docs/reference/commands/mfa.md`
- `docs/reference/commands/destroy.md`

Open explanation docs when the behavior looks surprising:

- `docs/explanation/about-sandbox-caching-and-reuse.md`
- `docs/explanation/about-why-status-is-session-only.md`
- `docs/explanation/about-sessions-refresh-and-re-login.md`
