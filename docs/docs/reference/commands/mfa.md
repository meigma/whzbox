---
title: mfa
sidebar_position: 8
description: Reference for generating a short-lived Azure console MFA code.
---

# mfa

`whzbox mfa azure` prints a current MFA code for Azure console login.

## Syntax

```text
whzbox mfa azure
```

## Arguments

| Argument | Required | Values | Description |
| --- | --- | --- | --- |
| `provider` | Yes | `azure` | Sandbox provider kind. |

## Behavior

- Requires an unexpired Azure sandbox in the local cache.
- Ensures the Whizlabs session is valid and may refresh or prompt for login.
- Requests a new code from Whizlabs.
- Does not save the code to the state file.
- Rejects other providers.

At the Microsoft MFA prompt, select **Use a verification code** before you enter the generated code. The default **Approve sign in request** prompt uses Authenticator number matching and does not accept this code.

## Output

The command writes only the MFA code and a newline to stdout. The code is valid for about 30 seconds. The global `--json` flag does not change this output.

## Exit behavior

- `0` on success
- `1` when the Azure cache entry is missing or expired
- `2` when authentication fails
- `3` when Whizlabs cannot generate the code
- `4` if a login prompt is shown and the user aborts it
- `5` if login is required without an interactive terminal

## See also

- [How to sign in to an Azure sandbox with MFA](../../how-to/sign-in-to-an-azure-sandbox-with-mfa.md)
- [Reference: create](./create.md)
