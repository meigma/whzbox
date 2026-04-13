---
name: whzbox-cli
description: Use when the whzbox CLI is installed and you need to create, reuse, inspect, use, or destroy AWS sandboxes. Covers login, create aws, list, status, exec, destroy, JSON output, and the cache and session behavior that matters for automation.
---

# whzbox CLI

Use this skill when the `whzbox` CLI is installed and you need an AWS sandbox through Whizlabs.

## Quick model

- `whzbox` is a Go CLI for Whizlabs cloud sandboxes.
- AWS is the only supported provider today.
- `login` stores a cached Whizlabs session locally.
- `logout` clears the cached session but can leave cached sandboxes in place.
- `create aws` creates or reuses an AWS sandbox, verifies credentials, and prints them.
- `list`, `status`, and `exec` are local-state commands. They do not talk to Whizlabs.
- `destroy` tears down the active sandbox upstream and clears cached sandbox entries locally.

The default goal for an agent is usually:

1. Reuse an active cached sandbox if one already exists.
2. Otherwise create a new AWS sandbox.
3. Run commands through `whzbox exec aws -- ...` when possible instead of manually copying credentials.
4. Destroy the sandbox only when the task or user asks for cleanup.

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

- `whzbox destroy`
  - Destroys the active sandbox upstream.
  - Prompts unless `--yes` is set.
  - In scripts, use `--yes`.

## Machine-readable output

Use JSON only with:

- `whzbox create aws --json`
- `whzbox list --json`

Do not assume `--json` changes output for `status`, `login`, `logout`, `destroy`, `exec`, `version`, or `completion`.

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
- `docs/reference/commands/destroy.md`

Open explanation docs when the behavior looks surprising:

- `docs/explanation/about-sandbox-caching-and-reuse.md`
- `docs/explanation/about-why-status-is-session-only.md`
- `docs/explanation/about-sessions-refresh-and-re-login.md`
