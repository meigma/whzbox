---
title: "How to relocate or inspect the state directory"
sidebar_position: 5
description: Override the whzbox state directory for one shell session or inspect the current state file and its permissions.
---

Use `WHZBOX_STATE_DIR` when you want `whzbox` to read and write state somewhere other than the default XDG path.

## Prerequisites

- The `whzbox` binary on your `PATH`

## Steps

### 1. Override the state directory

```bash
export WHZBOX_STATE_DIR=/tmp/whzbox-demo
whzbox status
```

With this environment variable set, `whzbox` uses `/tmp/whzbox-demo/state.json`.

### 2. Inspect the directory and file permissions

```bash
ls -ld "$WHZBOX_STATE_DIR"
ls -l "$WHZBOX_STATE_DIR/state.json"
```

When the file exists, the expected permissions are:

- directory: `0700`
- file: `0600`

### 3. Inspect the file contents

```bash
jq . "$WHZBOX_STATE_DIR/state.json"
```

The state file contains the cached Whizlabs session and any cached sandboxes.

## Verification

Unset the override and compare:

```bash
unset WHZBOX_STATE_DIR
whzbox status
```

Without the override, `whzbox` falls back to `$XDG_STATE_HOME/whzbox` or `~/.local/state/whzbox`.

## Troubleshooting

### Problem: the file loads with a permissions error

Fix the permissions:

```bash
chmod 700 "$WHZBOX_STATE_DIR"
chmod 600 "$WHZBOX_STATE_DIR/state.json"
```

### Problem: I want to inspect the default path

If `XDG_STATE_HOME` is unset, the default directory is:

```text
~/.local/state/whzbox
```

## Related

- [Reference: state file location, permissions, and lifecycle](../reference/state-file-location-permissions-and-lifecycle.md)
- [Reference: global flags and environment variables](../reference/global-flags-and-environment-variables.md)
