---
title: State file location, permissions, and lifecycle
sidebar_position: 5
description: Reference for the local state.json file, path resolution, security checks, and how auth and sandbox entries are preserved or cleared.
---

`whzbox` stores local state in a single JSON file named `state.json`.

## Path resolution

The state directory resolves in this order:

1. `WHZBOX_STATE_DIR`
2. `$XDG_STATE_HOME/whzbox`
3. `~/.local/state/whzbox`

The full file path is:

```text
<state-dir>/state.json
```

## Permissions

| Path | Mode |
| --- | --- |
| state directory | `0700` |
| state file | `0600` |

If the file mode is wider than `0600`, `whzbox` refuses to load it.

## On-disk shape

The file contains:

- `version`
- `whizlabs`
- `sandboxes`

Example:

```json
{
  "version": 1,
  "whizlabs": {
    "user_email": "alice@example.com",
    "access_token": "access-token",
    "refresh_token": "refresh-token",
    "access_token_expires_at": "2026-04-12T10:00:00Z",
    "refresh_token_expires_at": "2026-04-19T10:00:00Z"
  },
  "sandboxes": {
    "aws": {
      "kind": "aws",
      "slug": "aws-sandbox",
      "verified": true,
      "credentials": {
        "access_key": "AKIAEXAMPLE",
        "secret_key": "secret"
      },
      "console": {
        "url": "https://111111111111.signin.aws.amazon.com/console",
        "username": "whiz_user",
        "password": "password"
      },
      "identity": {
        "account": "111111111111",
        "user_id": "AIDAEXAMPLE",
        "arn": "arn:aws:iam::111111111111:user/whiz_user",
        "region": "us-east-1"
      },
      "started_at": "2026-04-12T00:00:00Z",
      "expires_at": "2026-04-12T01:00:00Z"
    }
  }
}
```

## Lifecycle rules

### Session entries

- `login` saves the auth section.
- session refresh overwrites only the auth section.
- `logout` clears only the auth section when cached sandboxes are present.
- `logout` removes the entire file when no cached sandboxes remain.

### Sandbox entries

- `create` saves the sandbox entry after provisioning.
- verification failure still saves the sandbox entry.
- `destroy` clears all cached sandboxes after a successful destroy.
- `list` and `exec` read the sandbox cache directly.

## Invalid-state handling

| Condition | Behavior |
| --- | --- |
| bad JSON in auth state | removes the file and treats the session as missing |
| unknown schema version in auth state | removes the file and treats the session as missing |
| unparsable auth timestamps | removes the file and treats the session as missing |
| unparsable sandbox entry | ignores that sandbox entry |
| file mode wider than `0600` | returns an error instead of loading |

## See also

- [Reference: global flags and environment variables](./global-flags-and-environment-variables.md)
- [How to relocate or inspect the state directory](../how-to/relocate-or-inspect-the-state-directory.md)
