---
title: Exit codes
sidebar_position: 3
description: Reference for whzbox process exit codes, including child-process propagation for exec.
---

`whzbox` maps command errors to process exit codes.

## Standard exit codes

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Generic error |
| `2` | Authentication error |
| `3` | Provider error |
| `4` | User aborted an interactive prompt |
| `5` | No interactive terminal available for a required prompt |

## `exec` child exit codes

`whzbox exec` is special:

- if the child process exits non-zero, `whzbox` returns the same exit code
- `whzbox` does not print its own `Error:` line for that case

Example:

```bash
whzbox exec aws -- false
echo $?
```

The shell prints:

```text
1
```

## Error mapping

| Condition | Exit code |
| --- | --- |
| invalid credentials | `2` |
| session expired and not recoverable | `2` |
| sandbox provider or verification failure | `3` |
| no active sandbox upstream for destroy | `3` |
| user aborts prompt | `4` |
| prompt needed but no TTY is attached | `5` |

## See also

- [Reference: exec](./commands/exec.md)
- [Reference: destroy](./commands/destroy.md)
