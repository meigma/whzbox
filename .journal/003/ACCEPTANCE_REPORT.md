# whzbox 1.1.0 Release Acceptance Report

Date: 2026-08-12
Candidate: release PR #25, `8a7c98d20773c56f5e0ca8356025c894bf34ed72`
Verdict: **NOT READY FOR RELEASE**

## Executive summary

The exact release candidate is clean and passes all local and hosted automated gates. AWS passed full live acceptance. GCP passed its CLI/API lifecycle and its generated console credentials were accepted by Google, reaching the first-login Workspace Terms of Service gate. Azure passed creation, metadata, cache, duration, rejection, MFA-generation, and teardown checks, and Microsoft accepted its username/password, but the live sign-in then required Authenticator number matching. `whzbox mfa azure` returns a six-digit OATH/TOTP code and the Microsoft page offered no field for it, so the Azure console could not be reached.

Do not merge release PR #25 or cut `v1.1.0` until the Azure MFA contract is re-probed, corrected, and re-accepted through the Azure portal.

## Candidate and automated evidence

- Release PR: https://github.com/meigma/whzbox/pull/25
- Exact head: `8a7c98d20773c56f5e0ca8356025c894bf34ed72`, matching its upstream branch.
- Candidate delta from `main@8a10387`: only `.release-please-manifest.json` and `CHANGELOG.md`.
- Local pinned gate: `mise exec -- moon run root:check --summary minimal` passed all 12 tasks, including formatting, lint, build, race tests, vet, repository-settings tests, and strict documentation build.
- Hosted checks: CI, Go and Actions CodeQL, Kusari Inspector, and Binary Release Dry Run all passed on the exact PR head.
- A `v1.1.0`-identified acceptance binary was built from the exact PR head and its version metadata was verified.
- Live Whizlabs login, refresh, persistence, logout, and interactive CLI login all passed using an isolated state directory with `0700` directory and `0600` file permissions.

## Major-feature matrix

| Surface | Result | Evidence |
| --- | --- | --- |
| Help, version, completion | PASS | Root and command help rendered; zsh completion generated; version identified `v1.1.0` and the exact head. |
| Status, list, JSON, state permissions | PASS | Empty and active states rendered correctly; JSON shapes validated; state modes were `0700`/`0600`. |
| Login and logout | PASS | Live login/refresh passed; interactive login persisted fresh tokens; logout cleared the session while preserving an active AWS cache. |
| Cache reuse | PASS | Repeated same-provider creates returned the same cached identity, timestamps, console data, and credentials without reprovisioning. |
| One-active-provider guard | PASS | AWS->GCP, GCP->Azure, and Azure->AWS attempts were rejected until the active sandbox was destroyed. |
| Duration validation | PASS | GCP `2h` and Azure `4h` were rejected before creation; Azure `90m` rounded to exactly two hours. |
| AWS | PASS | See provider detail below. |
| GCP | PASS | See provider detail below; legal Terms of Service were intentionally not accepted. |
| Azure | **FAIL** | Console MFA mechanism mismatch blocks portal access. |
| Final cleanup | PASS | Final `list --json` returned `[]`; a final upstream destroy returned the documented no-active-sandbox exit code `3`. |

## AWS acceptance

- Created a real one-hour AWS sandbox and validated complete console credentials, programmatic credentials, timestamps, region, and STS identity.
- Validated styled list and JSON list, same-provider reuse, and the different-provider guard.
- Ran `aws sts get-caller-identity` through argv-mode `exec` and matched account, user ID, and ARN to the create result.
- Validated shell-mode `exec`, AWS environment injection, and child exit-code propagation.
- Logged out while AWS remained cached, verified `list` and local `exec` still worked, then interactively logged back in without losing the sandbox.
- Destroyed successfully, verified an empty local cache, and verified a second destroy returned no-active-sandbox.

## GCP acceptance

- Created real one-hour GCP sandboxes and validated console credentials, empty AWS-style credentials, project ID/name, timestamps, styled/JSON list, cache reuse, and provider guards.
- Confirmed `exec gcp` gives the browser-console-only error and `mfa gcp` rejects the unsupported provider.
- A password-safe keyboard-level browser run proved Google accepted the generated username/password and advanced to the first-login Google Workspace Terms of Service gate. The legal terms were not accepted because credential validity was already proven and acceptance was outside the product contract.
- Test-harness incident: two earlier disposable GCP passwords were surfaced by browser DOM snapshots. Each affected sandbox was destroyed immediately; the later sandbox used rotated credentials, was accepted through the safe keyboard-level flow, and was also destroyed. No GCP sandbox remained active.

## Azure acceptance failure

Passed portions:

- Created a live Azure sandbox with `--duration 90m`; the returned lifetime was exactly two hours.
- Validated complete console credentials, empty AWS-style credentials, three resource groups, timestamps, styled/JSON list, cache reuse, and provider guards.
- Confirmed `exec azure` gives the browser-console-only error.
- Generated multiple fresh six-digit MFA codes, including while the Microsoft MFA challenge was pending; codes were not persisted or displayed.
- Microsoft accepted the generated username and password.

Failure:

- Microsoft then displayed **Approve sign in request** and required Authenticator number matching. The page had no input for the six-digit code produced by `whzbox mfa azure`.
- Generating a fresh code while the challenge was pending did not advance the login.
- Microsoft documents browser number matching as an Authenticator push approval flow and distinguishes it from OATH/TOTP verification codes: https://learn.microsoft.com/en-us/entra/identity/authentication/how-to-mfa-number-match and https://learn.microsoft.com/en-us/entra/identity/authentication/howto-mfa-mfasettings
- Therefore the documented `create azure` -> `mfa azure` -> enter code flow does not work against the live Azure sign-in contract as tested.

Required follow-up:

1. Re-probe the current Whizlabs Azure frontend and MFA endpoints to identify how its number-matching challenge is expected to be approved.
2. Correct the CLI and documentation contract, or remove/disable the unsupported Azure sign-in claim until a usable flow exists.
3. Repeat this acceptance from `create azure` through an authenticated Azure portal landing, then destroy and prove zero residual state.

## Cleanup

- Every AWS, GCP, and Azure sandbox created during acceptance was destroyed.
- Final local sandbox state was `[]`.
- Final upstream destroy returned no-active-sandbox with exit code `3`.
- Browser login tabs were closed and the browser session was finalized.
- All 101 temporary acceptance files, including credential-bearing JSON and MFA output, were deleted with their dedicated `/tmp/whzbox-accept.*` directory before handoff.
