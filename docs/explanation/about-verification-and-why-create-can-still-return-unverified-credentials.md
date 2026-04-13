---
title: "About verification and why create can still return unverified credentials"
sidebar_position: 3
description: Understand the post-create AWS STS verification step, IAM propagation retries, and why the CLI still returns credentials on verification failure.
---

After `whzbox` provisions and commits a sandbox, it verifies the returned AWS credentials by calling `sts:GetCallerIdentity`.

## Why verification exists

Provisioning success from Whizlabs is not enough. The CLI wants proof that the credentials are usable against the real cloud provider before it presents them as ready.

## Why verification can fail transiently

Fresh IAM users and access keys can take time to propagate. The AWS verifier retries `GetCallerIdentity` when the error looks like `InvalidClientTokenId`.

The current retry settings are:

- region: `us-east-1`
- max attempts: `15`
- backoff: `3s`

## Why the CLI still returns the sandbox

At the point verification runs:

- the sandbox has already been provisioned
- the commit step has already registered it
- the sandbox is real and billed against the account

Destroying it automatically would hide a real resource that the user may still want to use once propagation catches up. Instead, `create` returns the sandbox data, renders it, and exits with a provider error.

## What happens next

The cached sandbox is written to disk even when verification fails. On a later `create` call, `whzbox` can try verification again before deciding whether to reuse the cache or create a new sandbox.

## Related

- [Reference: create](../reference/commands/create.md)
- [About sandbox caching and reuse](./about-sandbox-caching-and-reuse.md)
