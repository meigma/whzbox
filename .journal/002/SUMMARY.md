---
id: 002
title: Add Azure console sandbox support
date: 2026-08-12
status: complete
repos_touched: [whzbox]
related_sessions: [001]
---

## Goal
Add Microsoft Azure sandboxes alongside the existing AWS and GCP providers, using a live contract spike to keep the first implementation small and evidence-based.

## Outcome
Goal met. `whzbox` now creates, caches, lists, and destroys Azure console sandboxes, persists their resource groups, and fetches short-lived MFA codes on demand. PR #40 passed all hosted and external checks and was squash-merged to `main` as `8a10387`. Disposable one-hour and three-hour live lifecycle exercises both completed with successful teardown.

## Key Decisions
- Prove the authenticated Whizlabs contract before designing the integration -> Azure uses the same lab lifecycle as GCP, returns three resource groups, and exposes MFA through a separate endpoint.
- Ship Azure as console-only -> the live response contains no tenant, subscription, service principal, client secret, or other credential suitable for `az` or SDK execution.
- Add `whzbox mfa azure` as an on-demand operation -> the returned OTP is short-lived, sensitive, and must never be persisted.
- Reuse the GCP lab lifecycle without introducing a broader provider abstraction -> the shared HTTP shape is real, while provider-specific durations and MFA behavior remain explicit.
- Preserve state schema version 1 -> Azure resource groups are an optional additive field and existing AWS/GCP state remains readable.

## Changes
- `internal/adapters/azureconsole/` - added the console-only Azure provider contract.
- `internal/adapters/whizlabs/` - added Azure create, commit, destroy, and MFA handling plus a disposable live lifecycle acceptance test.
- `internal/core/sandbox/` - added the Azure kind, resource-group identity metadata, and non-persisted MFA service operation.
- `internal/adapters/xdgstore/`, `internal/ui/`, and `internal/cli/` - persisted and rendered Azure metadata, exposed `create azure` and `mfa azure`, and rejected `exec azure` explicitly.
- `README.md`, `SKILL.md`, and `docs/docs/` - documented Azure creation, browser login, MFA, durations, state, JSON output, and console-only limitations.
- `internal/ui/render.go` - reused renderer label constants after golangci-lint 2.12.2 exposed a CI-only `goconst` failure.

## Open Threads
- Programmatic Azure execution remains unavailable unless Whizlabs starts returning a workload identity or equivalent credentials.
- The Azure lab and MFA APIs are undocumented public behavior; re-probe them if Whizlabs changes its frontend contract or lifecycle begins failing.

## Lessons
- The authenticated spike prevented a false Azure CLI design and established the create, MFA, and teardown contracts before production changes.
- Local validation must use the repository-pinned golangci-lint version; a global 2.11.4 binary missed findings enforced by pinned 2.12.2 in CI.

## References
- PR #40: https://github.com/meigma/whzbox/pull/40
- Session 001: `.journal/001/SUMMARY.md`
- Session log: `.journal/002/NOTES.md`
