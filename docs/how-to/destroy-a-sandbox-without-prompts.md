---
title: "How to destroy a sandbox without prompts"
sidebar_position: 4
description: Destroy the active sandbox from scripts or other non-interactive environments by using --yes.
---

Use `--yes` with `whzbox destroy` to skip the confirmation prompt.

## Prerequisites

- An active Whizlabs session or an interactive terminal for re-login

## Steps

### 1. Destroy the sandbox

```bash
whzbox destroy --yes
```

### 2. Confirm that the cache is empty

```bash
whzbox list
```

## Verification

After a successful destroy, `whzbox list` prints:

```text
(no sandboxes cached)
```

## Troubleshooting

### Problem: destroy still failed

If there is no active sandbox upstream, `destroy` returns a provider error.

### Problem: destroy asked for credentials

`destroy` needs a valid session. If the cached session is expired and cannot be refreshed, log in again:

```bash
whzbox login
```

## Related

- [Reference: destroy](../reference/commands/destroy.md)
- [Reference: exit codes](../reference/exit-codes.md)
