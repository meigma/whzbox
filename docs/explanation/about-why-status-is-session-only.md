---
title: "About why status is session-only"
sidebar_position: 4
description: Understand why the current status command shows cached session data but not live sandbox state.
---

`whzbox status` is intentionally limited to the cached Whizlabs session.

## The command implementation

The CLI command calls `Session.Current`, which reads the stored tokens directly from disk. It does not call the sandbox service.

## Why it does not show live sandbox state

The sandbox adapter has no reliable upstream endpoint for "show me the currently active sandbox." The current Whizlabs play API surface does not expose a clean runtime-status call, so the adapter returns "no active sandbox" for that path.

Because of that gap, the command stays honest by reporting only what it can know reliably from the local machine:

- whether a cached session exists
- when the access token expires
- when the refresh token expires

## Why this is useful anyway

A session-only `status` command still has value:

- it is fast
- it is safe in scripts
- it never refreshes or prompts
- it tells you whether a later authenticated command will probably proceed without re-login

## Related

- [Reference: status](../reference/commands/status.md)
- [How to use whzbox in non-interactive scripts](../how-to/use-whzbox-in-non-interactive-scripts.md)
