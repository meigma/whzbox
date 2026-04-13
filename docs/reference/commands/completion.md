---
title: completion
sidebar_position: 9
description: Reference for the Cobra-generated whzbox completion command.
---

`whzbox completion <shell>` writes a shell completion script to stdout.

## Syntax

```text
whzbox completion [bash|zsh|fish|powershell]
```

## Arguments

| Argument | Required | Values | Description |
| --- | --- | --- | --- |
| `shell` | Yes | `bash`, `zsh`, `fish`, `powershell` | Target shell for the generated completion script. |

## Flags

This command uses only the global flags.

## Behavior

- Is provided by Cobra's built-in completion command.
- Writes the script to stdout.
- Does not require a state directory or network access.

## Output

Shell-specific completion script text.

## Exit behavior

- `0` on success
- `1` on generic command failure

## See also

- [Reference: global flags and environment variables](../global-flags-and-environment-variables.md)
