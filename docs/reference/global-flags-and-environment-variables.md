---
title: Global flags and environment variables
sidebar_position: 2
description: Reference for shared whzbox flags, their matching environment variables, and configuration precedence.
---

These settings are shared across the CLI.

## Global flags

| Flag | Type | Default | Environment variable | Description |
| --- | --- | --- | --- | --- |
| `--log-level` | string | derived | `WHZBOX_LOG_LEVEL` | Absolute log level. Accepted values: `debug`, `info`, `warn`, `warning`, `error`. |
| `-v`, `--verbose` | count | `0` | `WHZBOX_VERBOSE` | Increases verbosity. `-vv` enables debug logging. |
| `-q`, `--quiet` | boolean | `false` | `WHZBOX_QUIET` | Suppresses non-error log output. |
| `--no-color` | boolean | `false` | `WHZBOX_NO_COLOR` or `NO_COLOR` | Disables ANSI color output. |
| `--yes` | boolean | `false` | `WHZBOX_YES` | Skips confirmation prompts. |
| `--json` | boolean | `false` | `WHZBOX_JSON` | Requests machine-readable JSON. Only `create` and `list` implement JSON output. |

## Environment variables without matching flags

| Variable | Description |
| --- | --- |
| `WHZBOX_STATE_DIR` | Overrides the state directory used for `state.json`. |
| `WHZBOX_WHIZLABS_BASE_URL` | Overrides the main Whizlabs API base URL. Intended for tests or local development. |
| `WHZBOX_WHIZLABS_PLAY_URL` | Overrides the Whizlabs play API base URL. Intended for tests or local development. |

## Configuration precedence

Shared configuration is loaded in this order:

1. command-line flags
2. environment variables
3. built-in defaults

## Logging precedence

After configuration is loaded, logging resolves in this order:

1. `--log-level` or `WHZBOX_LOG_LEVEL`
2. `-v` or `WHZBOX_VERBOSE`
3. `-q` or `WHZBOX_QUIET`
4. default `info`

## Notes

- There is no `--state-dir` flag. Use `WHZBOX_STATE_DIR`.
- `NO_COLOR` is honored even without `--no-color`.

## See also

- [Reference: exit codes](./exit-codes.md)
- [Reference: state file location, permissions, and lifecycle](./state-file-location-permissions-and-lifecycle.md)
