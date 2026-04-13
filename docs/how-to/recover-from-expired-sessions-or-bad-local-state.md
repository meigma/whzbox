---
title: "How to recover from expired sessions or bad local state"
sidebar_position: 6
description: Recover from expired tokens, overbroad state-file permissions, or corrupt state-file contents.
---

Use this guide when `whzbox` stops authenticating cleanly or when the local state file is invalid.

## Prerequisites

- Access to the machine where `whzbox` stores its local state

## Steps

### 1. Re-establish the session when tokens have expired

Run:

```bash
whzbox login
```

`login` always prompts and replaces the stored session.

### 2. Fix state-file permissions if they are too broad

If `whzbox` reports that the state file mode is wider than `0600`, fix the permissions:

```bash
state_dir="${WHZBOX_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/whzbox}"
chmod 700 "$state_dir"
chmod 600 "$state_dir/state.json"
```

### 3. Remove the file if you want to start clean

```bash
state_dir="${WHZBOX_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/whzbox}"
rm -f "$state_dir/state.json"
```

The next command behaves like a first run.

## Verification

Check the current session:

```bash
whzbox status
```

If the file is gone or the auth section is empty, `status` prints:

```text
Session
  (not logged in)
```

## Troubleshooting

### Problem: the state file contains bad JSON

`whzbox` removes corrupt JSON automatically the next time it tries to load session tokens.

### Problem: the state file has an unknown schema version or unparsable token timestamps

`whzbox` removes that auth state automatically on load.

### Problem: cached sandbox data is bad

`whzbox` ignores unparsable cached sandbox entries and proceeds as if the cache missed.

## Related

- [Reference: state file location, permissions, and lifecycle](../reference/state-file-location-permissions-and-lifecycle.md)
- [About sessions, refresh, and re-login](../explanation/about-sessions-refresh-and-re-login.md)
