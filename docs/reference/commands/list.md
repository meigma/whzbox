---
title: list
sidebar_position: 5
description: Reference for whzbox list, the cache-backed sandbox listing command.
---

`whzbox list` reads cached sandboxes from the local state file.

## Syntax

```text
whzbox list
```

## Arguments

This command takes no positional arguments.

## Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--json` | boolean | `false` | Emits a JSON array instead of the styled table. |

## Behavior

- Reads only the local sandbox cache.
- Does not refresh tokens.
- Does not prompt for credentials.
- Includes expired cached sandboxes and marks them as `expired` in styled output.

## Output

- Styled output: a table with `KIND`, `ACCOUNT`, `STATUS`, and `EXPIRES`
- JSON output: an array of sandbox objects. See [JSON output schemas](../json-output-schemas.md).

## Exit behavior

- `0` on success
- `1` on local state-file errors

## See also

- [Reference: JSON output schemas](../json-output-schemas.md)
- [Reference: state file location, permissions, and lifecycle](../state-file-location-permissions-and-lifecycle.md)
