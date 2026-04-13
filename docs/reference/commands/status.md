---
title: status
sidebar_position: 6
description: Reference for whzbox status, the read-only session inspection command.
---

`whzbox status` prints the cached Whizlabs session, if one exists.

## Syntax

```text
whzbox status
```

## Arguments

This command takes no positional arguments.

## Flags

This command uses only the global flags.

## Behavior

- Reads the cached session directly from the local token store.
- Does not refresh tokens.
- Does not prompt for credentials.
- Does not show live sandbox status.

## Output

Styled output only. The output contains:

- `Session`
- `Email` when a session exists
- `Expires`
- `Refresh` when a refresh token is present

When no session exists, the output contains `(not logged in)`.

## Exit behavior

- `0` on success
- `1` on local state-file errors

## See also

- [About why status is session-only](../../explanation/about-why-status-is-session-only.md)
- [Reference: global flags and environment variables](../global-flags-and-environment-variables.md)
