---
title: logout
sidebar_position: 2
description: Reference for the whzbox logout command.
---

`whzbox logout` clears the cached Whizlabs session from the local state file.

## Syntax

```text
whzbox logout
```

## Arguments

This command takes no positional arguments.

## Flags

This command uses only the global flags.

## Behavior

- Clears the stored auth section from the state file.
- Preserves cached sandboxes when sandbox entries are present.
- Removes the entire state file when no sandbox cache remains.

## Output

This command writes no structured data to stdout on success.

## Exit behavior

- `0` on success
- `1` on generic local I/O failure

## See also

- [Reference: state file location, permissions, and lifecycle](../state-file-location-permissions-and-lifecycle.md)
- [About sandbox caching and reuse](../../explanation/about-sandbox-caching-and-reuse.md)
