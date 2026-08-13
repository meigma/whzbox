# Azure MFA Resolution Research

Date: 2026-08-12

## Outcome

The Azure feature is functional. The failed acceptance run stopped at Microsoft's preferred Authenticator push/number-matching prompt, while `whzbox mfa azure` correctly generates the account's alternate OATH/TOTP verification code.

On a fresh disposable Azure sandbox, the working flow was:

1. Sign in to Azure with the generated username and password.
2. At **Approve sign in request**, select **Use a verification code**.
3. Run `whzbox mfa azure` and enter the fresh six-digit code in the field that appears.
4. Select **Verify**.

Microsoft accepted the code and the browser reached `https://portal.azure.com/#home` with title **Home - Microsoft Azure**.

## Why the prompt changed

Microsoft Entra's system-preferred authentication chooses the strongest registered method first. Authenticator notification is ranked above TOTP, but users can select another registered method. Number matching is mandatory for Authenticator push notifications and is a different mechanism from a six-digit OATH/TOTP code.

Sources:

- https://learn.microsoft.com/en-us/entra/identity/authentication/concept-system-preferred-authentication
- https://learn.microsoft.com/en-us/entra/identity/authentication/how-to-mfa-number-match
- https://learn.microsoft.com/en-us/entra/identity/authentication/howto-mfa-mfasettings

## Current product mismatch

The adapter calls Whizlabs' `play-az-generate-mfa` endpoint and returns its `otp` value. The CLI and documentation describe entering that code, but they omit the Microsoft method-selection step. This makes the correct TOTP appear incompatible with the default number-matching screen.

No evidence indicates that `whzbox` needs to approve number matching, automate Microsoft login, or call an additional Whizlabs endpoint.

## Recommended release fix

Keep the change small and instructional:

- Change the Azure create guidance to say: at the Microsoft MFA prompt, select **Use a verification code**, then run `whzbox mfa azure`.
- Add the same one-line instruction to `mfa azure` help or stderr guidance while preserving stdout as code-only for scripting.
- Update the Azure sign-in how-to and command reference with the exact button and fallback wording.
- Add a troubleshooting note: if **Use a verification code** is absent, cancel and choose another sign-in method; if TOTP is not offered there, treat that sandbox account as an upstream provisioning fault.

Do not disable system-preferred authentication or number matching. Those are tenant security controls, and the supported alternate TOTP path already works.

## Release gate

After the guidance change, rerun the Azure acceptance slice from the exact release head:

- create Azure sandbox;
- sign in with username and password;
- select **Use a verification code**;
- generate and submit a fresh MFA code;
- prove Azure Portal home loads;
- destroy the sandbox and prove empty local state.

This is sufficient to clear the original Azure acceptance failure; a new broad three-cloud redesign or full acceptance campaign is not required unless the release candidate changes materially.

## Cleanup

- Disposable Azure sandbox destroyed successfully.
- Final isolated `list --json` result contained zero sandboxes.
- Browser tab closed and browser session finalized.
- Credential and MFA values were never included in this report.
