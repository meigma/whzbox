---
title: exec
sidebar_position: 7
description: Reference for whzbox exec, including argv mode, shell mode, subshell mode, and exit-code propagation.
---

`whzbox exec <provider> [-- cmd args...]` runs a child process with sandbox environment variables injected.

## Syntax

```text
whzbox exec <provider> [-- cmd args...]
whzbox exec <provider> -s "<shell command>"
whzbox exec <provider>
```

## Arguments

| Argument | Required | Values | Description |
| --- | --- | --- | --- |
| `provider` | Yes | `aws` | Sandbox provider kind. |
| `cmd args...` | No | Any child argv | Command to run. Omit it to launch a subshell. |

## Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `-s`, `--shell` | boolean | `false` | Treats the single command argument as a shell string and runs `/bin/sh -c`. |

## Behavior

- Reads the cached sandbox directly from the local store.
- Does not refresh tokens.
- Does not prompt for credentials.
- Rejects unknown providers.
- Rejects expired or missing cached sandboxes.
- Appends provider-specific environment variables to the current process environment.
- Propagates the child process exit code.

## Environment variables injected for AWS

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`
- `AWS_DEFAULT_REGION`

## Output

`whzbox exec` does not wrap child output. The child process writes directly to the inherited stdout and stderr streams.

## Exit behavior

- `0` when the child process exits successfully
- child exit code when the child process exits non-zero
- `1` for local execution errors, invalid shell usage, or missing cached sandbox

## See also

- [How to run AWS CLI commands with whzbox exec](../../how-to/run-aws-cli-commands-with-whzbox-exec.md)
- [Reference: exit codes](../exit-codes.md)
