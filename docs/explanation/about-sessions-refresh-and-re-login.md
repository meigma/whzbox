---
title: "About sessions, refresh, and re-login"
sidebar_position: 1
description: Understand how whzbox chooses between cached access tokens, refresh, and interactive re-login.
---

`whzbox` uses a session service to decide whether the current command can proceed with the tokens on disk or needs new credentials.

## The decision order

For commands that require authentication, the service follows this order:

1. load the stored session
2. if the access token is still valid and not near expiry, use it
3. if the access token is near expiry and the refresh token is still valid, refresh it
4. if refresh is not possible or fails, prompt for credentials and log in again

The proactive refresh window is 10 minutes before access-token expiry.

## Why `login` is different

`whzbox login` is an explicit reset of the session. It does not try to refresh. It always prompts and overwrites the stored auth section.

## Why `status` is different

`whzbox status` does not call the session validation flow. It reads the cached session directly so that:

- it never refreshes tokens behind the user's back
- it never prompts unexpectedly
- it remains safe to use in scripts and diagnostics

## Failure modes

- invalid credentials return an authentication error
- a missing TTY during a required prompt returns a non-interactive error
- prompt cancellation returns a user-aborted error

## Related

- [Reference: login](../reference/commands/login.md)
- [Reference: status](../reference/commands/status.md)
