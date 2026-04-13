---
title: login
sidebar_position: 1
description: Reference for the interactive whzbox login command.
---

`whzbox login` signs in to Whizlabs interactively and replaces the stored session tokens.

## Syntax

```text
whzbox login
```

## Arguments

This command takes no positional arguments.

## Flags

This command uses only the global flags.

## Behavior

- Always prompts for credentials through the interactive prompt adapter.
- Ignores any cached refresh token.
- Saves the returned Whizlabs access token, refresh token, expiry timestamps, and user email to the local state file.
- Returns a non-interactive error when no terminal is attached.

## Output

This command writes no structured data to stdout on success.

## Exit behavior

- `0` on success
- `2` for invalid credentials
- `4` if the user aborts the prompt
- `5` if no interactive terminal is available

## See also

- [Reference: global flags and environment variables](../global-flags-and-environment-variables.md)
- [About sessions, refresh, and re-login](../../explanation/about-sessions-refresh-and-re-login.md)
