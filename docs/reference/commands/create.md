---
title: create
sidebar_position: 3
description: Reference for whzbox create, including provider arguments, duration handling, cache reuse, and JSON output.
---

`whzbox create <provider>` provisions a sandbox, registers it with the account, verifies the credentials, and renders the result.

## Syntax

```text
whzbox create <provider> [flags]
```

## Arguments

| Argument | Required | Values | Description |
| --- | --- | --- | --- |
| `provider` | Yes | `aws` | Sandbox provider kind. |

## Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--duration` | duration | `1h` | Requested lifetime. Values are rounded up to whole hours before the upstream API call. Rounded values must fall between `1h` and `9h`. |
| `--json` | boolean | `false` | Emits a JSON object instead of the styled terminal renderer. |

## Behavior

- Reuses an unexpired cached sandbox for the same provider when one is already stored locally.
- Re-verifies an unexpired cached sandbox if the cached entry is marked unverified.
- Uses the session service, which may return a cached access token, refresh it, or prompt for credentials.
- Calls the Whizlabs create endpoint and then the Whizlabs commit endpoint.
- Saves the created sandbox to the local cache even when credential verification fails.
- Returns the created sandbox together with a provider error when verification fails after creation.

## Output

- Styled output: a rendered box with identity, console credentials, API credentials, and expiry information.
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
