---
title: create
sidebar_position: 3
description: Reference for whzbox create, including provider arguments, duration handling, cache reuse, and JSON output.
---

# create

`whzbox create <provider>` provisions a sandbox, registers it with the account, caches it, and renders the result. AWS credentials are verified with STS. Azure and GCP are console-only and have no programmatic credentials to verify.

## Syntax

```text
whzbox create <provider> [flags]
```

## Arguments

| Argument | Required | Values | Description |
| --- | --- | --- | --- |
| `provider` | Yes | `aws`, `azure`, `gcp` | Sandbox provider kind. |

## Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--duration` | duration | `1h` | Requested lifetime. AWS accepts `1h` through `9h`; Azure accepts `1h` through `3h`; GCP accepts `1h` only. Partial AWS and Azure hours round up. |
| `--json` | boolean | `false` | Emits a JSON object instead of the styled terminal renderer. |

## Behavior

- Reuses an unexpired cached sandbox for the same provider when one is already stored locally.
- Rejects creation when a different provider has an unexpired cached sandbox.
- Re-verifies unverified AWS cache entries. Azure and GCP cache entries are console-only and do not have a verification step.
- Uses the session service, which may return a cached access token, refresh it, or prompt for credentials.
- Calls the provider-specific Whizlabs create endpoint and then its commit endpoint.
- Saves the created sandbox to the local cache even when credential verification fails.
- Returns the created sandbox together with a provider error when verification fails after creation.

## Output

- AWS styled output includes identity, console credentials, API credentials, and expiry information.
- Azure styled output includes resource groups, console credentials, expiry information, and instructions to select **Use a verification code** before running `whzbox mfa azure`.
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
