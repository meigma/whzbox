---
title: "How to create a sandbox with a specific duration"
sidebar_position: 1
description: Request a specific sandbox lifetime with the create command and understand how partial-hour values are handled.
---

Use `--duration` with `whzbox create aws` to request a sandbox lifetime.

## Prerequisites

- A valid Whizlabs session or an interactive terminal for re-login
- The `whzbox` binary on your `PATH`

## Steps

### 1. Run create with a duration

```bash
whzbox create aws --duration 2h
```

`whzbox` accepts Go duration syntax. The current implementation allows values that round into the range `1h` through `9h`.

### 2. Verify the expiration time

```bash
whzbox list
```

The `EXPIRES` column should reflect the requested lifetime.

## Verification

For a machine-readable check, use:

```bash
whzbox create aws --duration 90m --json | jq -r '.expires_at'
```

`90m` is rounded up before the upstream API call, so the requested lifetime becomes `2h`.

## Troubleshooting

### Problem: the command rejects the duration

`whzbox` rejects values that round below `1h` or above `9h`.

### Problem: a new sandbox was not created

`create` can reuse an unexpired cached sandbox for the same provider. Check the current cache with:

```bash
whzbox list
```

If you need a fresh sandbox, destroy the current one first.

## Related

- [Reference: create](../reference/commands/create.md)
- [About sandbox caching and reuse](../explanation/about-sandbox-caching-and-reuse.md)
