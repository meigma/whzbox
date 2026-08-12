---
title: create
sidebar_position: 3
description: Reference for whzbox create, including provider arguments, duration handling, cache reuse, and JSON output.
---

# create

`whzbox create <provider>` provisions a sandbox, registers it with the account, caches it, and renders the result. AWS credentials are verified with STS. GCP is console-only and has no programmatic credential to verify.

## Syntax

```text
whzbox create <provider> [flags]
```

## Arguments

| Argument | Required | Values | Description |
| --- | --- | --- | --- |
| `provider` | Yes | `aws`, `gcp` | Sandbox provider kind. |

## Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--duration` | duration | `1h` | Requested lifetime. AWS values are rounded up to whole hours and must fall between `1h` and `9h`. GCP currently accepts `1h` only. |
| `--json` | boolean | `false` | Emits a JSON object instead of the styled terminal renderer. |

## Behavior

- Reuses an unexpired cached sandbox for the same provider when one is already stored locally.
- Rejects creation when a different provider has an unexpired cached sandbox.
- Re-verifies unverified AWS cache entries. GCP cache entries are console-only and do not have a verification step.
- Uses the session service, which may return a cached access token, refresh it, or prompt for credentials.
- Calls the provider-specific Whizlabs create endpoint and then its commit endpoint.
- Saves the created sandbox to the local cache even when credential verification fails.
- Returns the created sandbox together with a provider error when verification fails after creation.

## Output

- AWS styled output includes identity, console credentials, API credentials, and expiry information.
- GCP styled output includes project metadata, console credentials, and expiry information.
- JSON output: a single sandbox object. See [JSON output schemas](../json-output-schemas.md).

## Exit behavior

- `0` on success
- `2` when authentication fails before creation
- `3` on provider failures, including post-create verification failure
- `4` if re-login is prompted and the user aborts it
- `5` if re-login is needed but no interactive terminal is available

## See also

- [Reference: JSON output schemas](../json-output-schemas.md)
- [About verification and why create can still return unverified credentials](../../explanation/about-verification-and-why-create-can-still-return-unverified-credentials.md)
- [About sandbox caching and reuse](../../explanation/about-sandbox-caching-and-reuse.md)
