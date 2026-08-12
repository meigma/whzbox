---
id: 001
title: Add GCP console sandbox support
date: 2026-08-12
status: complete
repos_touched: [whzbox]
related_sessions: []
---

## Goal
Verify the providers supported by `whzbox`, determine the live Whizlabs GCP sandbox contract, and add the smallest useful GCP integration supported by that contract.

## Outcome
Goal met. `whzbox` now creates, caches, lists, and destroys Whizlabs GCP console sandboxes while preserving AWS behavior and JSON compatibility. PR #39 passed all hosted checks and was squash-merged to `main` as `e9722a6`. A disposable live GCP lifecycle acceptance passed and cleaned up its sandbox.

## Key Decisions
- Probe the undocumented live contract before designing the integration -> Whizlabs routes GCP through its lab API and returns browser credentials plus project metadata, not credentials usable by `gcloud` or SDKs.
- Ship console-only GCP support -> `exec gcp` fails with an explicit explanation instead of pretending console credentials are programmatic credentials.
- Preserve the existing AWS JSON fields -> GCP adds project ID/name while leaving the AWS credential and identity keys compatible.
- Enforce one active provider and a one-hour GCP duration -> these are the account and request shapes proven against the live product.
- Raise the Whizlabs request timeout to four minutes -> observed GCP create and teardown calls can each take close to three minutes.

## Changes
- `internal/adapters/gcpconsole/` - added the console-only GCP provider contract.
- `internal/adapters/whizlabs/` - added GCP lab create, commit, and teardown routes with scrubbed contract tests.
- `internal/core/sandbox/` - added the GCP kind, project metadata, unsupported-verification handling, provider-aware destroy, and the cross-provider active-sandbox guard.
- `internal/adapters/xdgstore/`, `internal/ui/`, and `internal/cli/` - persisted and rendered GCP data, exposed `create gcp`, updated `list`, and rejected `exec gcp` clearly.
- `README.md`, `SKILL.md`, and `docs/docs/` - documented GCP console access, duration, JSON shape, and execution limits.

## Open Threads
- Programmatic GCP execution remains unavailable unless Whizlabs starts returning a service account, OAuth access token, private key, or equivalent credential.
- The GCP lab API is undocumented public behavior; re-probe it if Whizlabs changes the frontend contract or lifecycle begins failing.

## Lessons
- A short live contract spike prevented building a false `gcloud` integration around browser-only credentials.
- Privileged acceptance should remain disposable and redaction-safe: the final live run exercised create, commit, parsing, and destroy without committing the acceptance harness or logging secrets.

## References
- PR #39: https://github.com/meigma/whzbox/pull/39
- Whizlabs GCP sandbox: https://www.whizlabs.com/google-cloud-sandbox/
- Session log: `.journal/001/NOTES.md`
