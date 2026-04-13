---
title: destroy
sidebar_position: 4
description: Reference for whzbox destroy, including confirmation behavior and non-interactive use.
---

`whzbox destroy` tears down the currently active sandbox upstream and clears cached sandbox entries locally.

## Syntax

```text
whzbox destroy
```

## Arguments

This command takes no positional arguments.

## Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--yes` | boolean | `false` | Skips the interactive confirmation prompt. Required in non-interactive environments. |

## Behavior

- Shows a confirmation prompt unless `--yes` is set.
- Uses the session service, which may refresh or prompt before the destroy call.
- Calls the Whizlabs destroy endpoint without a local sandbox identifier.
- Clears all cached sandbox entries after a successful destroy.
- Returns a provider error when the upstream reports that no active sandbox exists.

## Output

This command writes no structured data to stdout on success.

## Exit behavior

- `0` on success
- `2` when authentication fails before destroy
- `3` on provider failures, including "no active sandbox"
- `4` if the user aborts the confirmation prompt
- `5` if confirmation is required but no interactive terminal is available

## See also

- [How to destroy a sandbox without prompts](../../how-to/destroy-a-sandbox-without-prompts.md)
- [Reference: exit codes](../exit-codes.md)
