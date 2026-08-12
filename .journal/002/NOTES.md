---
id: 002
title: Awaiting task request
started: 2026-08-12
---

## 2026-08-12 16:19 — Kickoff
Goal for the session: Start a new journal session; the substantive task has not yet been provided.
Current state of the world: Session 001 added and live-validated GCP console sandbox support, and `main` includes the merged work from PR #39.
Plan: Bind this new session to the current task, then wait for the user's request.

## 2026-08-12 16:25 — Azure support proposal
Goal for the session: Propose an evidence-backed, agile plan for adding Microsoft Azure alongside the existing AWS and GCP sandboxes.
Current findings:
- The public Whizlabs frontend currently routes `azure-sandbox` through the lab lifecycle used by GCP: `play-create-lab`, `play-update-user-task-status`, and `play-end-lab`.
- The create payload appears to match GCP's lab payload, while the rendered response uses console URL, username, password, and one or more `resource_group` values.
- Azure login requires a 30-second MFA TOTP fetched separately from `play-az-generate-mfa`; the public page describes a three-hour maximum and one active sandbox at a time.
- Microsoft documents that username/password Azure CLI login does not work for MFA-enabled accounts, so Azure should initially be console-only and `exec azure` should fail explicitly.
- These observations came from public frontend assets only. No authenticated Azure sandbox was created, so the live contract remains the first implementation gate.
Proposed delivery:
1. Run a disposable authenticated contract spike, capture only scrubbed request/response shapes, prove create/commit/MFA/console-login/destroy, and confirm accepted durations and timeout behavior.
2. Add `KindAzure`, resource-group identity metadata, an `azureconsole` provider, and lab-route handling shared with GCP without introducing a broader provider abstraction.
3. Add an on-demand Azure MFA operation and `whzbox mfa azure`; never cache the 30-second code. Keep `exec azure` unsupported unless the live response unexpectedly includes a workload identity.
4. Wire `create azure`, caching, list/render/JSON output, provider-aware destroy, and backward-compatible optional state fields.
5. Add behavior-focused adapter, service, persistence, CLI, renderer, and compatibility tests; update concise user and reference docs.
6. Pass `moon run root:check`, then run a final disposable live Azure lifecycle acceptance and always destroy the sandbox.
Verification: The unchanged repository passes `moon run root:check` on Moon 2.3.5 and Go 1.26.4. `mise exec` was not used because this checkout's config is untrusted; the already-installed tool versions match the repository pins.
