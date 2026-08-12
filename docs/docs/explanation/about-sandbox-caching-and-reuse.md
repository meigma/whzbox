---
title: "About sandbox caching and reuse"
sidebar_position: 2
description: Understand why whzbox caches sandbox data locally and how that affects create, list, exec, and logout.
---

# About sandbox caching and reuse

`whzbox` caches sandbox data locally because the practical CLI workflow needs the credentials after the initial create call, while the upstream API does not provide a clean "fetch the credentials for the current sandbox" path.

## Why the cache exists

Without a local cache, the credentials printed by `create` would be lost as soon as the terminal session ended. The cache makes later commands possible:

- `list` can show what is still stored locally
- `exec aws` can inject cached AWS credentials into a child process
- `create` can reuse a still-live sandbox instead of provisioning again

## Reuse rules

`create` checks the local cache first:

- no cached sandbox: provision a new one
- cached but expired sandbox: provision a new one
- cached and verified sandbox: reuse it immediately
- cached and unverified AWS sandbox: try verification again before deciding
- cached GCP sandbox: reuse its console credentials without verification

Whizlabs permits one active sandbox per user. If a different unexpired provider is cached, `create` asks you to destroy it first.

## Why `logout` preserves cached sandboxes

Session tokens and sandbox credentials serve different purposes. Clearing the Whizlabs session does not remove cached AWS or GCP credentials, so `logout` preserves cached sandboxes on disk.

That is why this sequence can still work while the sandbox remains valid:

1. `whzbox logout`
2. `whzbox list`
3. `whzbox exec aws -- ...`

## Operational consequence

Commands do not all have the same dependency model:

- `create` and `destroy` need a valid Whizlabs session
- `list` and `exec` need only the local sandbox cache
- `status` needs only the local auth cache

## Related

- [Reference: create](../reference/commands/create.md)
- [Reference: list](../reference/commands/list.md)
- [Reference: exec](../reference/commands/exec.md)
