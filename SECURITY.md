# Security policy

## Reporting a vulnerability

If you discover a security issue in whzbox, please report it by emailing the maintainer directly rather than opening
a public issue. Include a description of the problem, steps to reproduce, and any proof-of-concept you can share.

## Credential handling

whzbox stores the Whizlabs access token, refresh token, and the email of the logged-in user in a single JSON file at:

```
$XDG_STATE_HOME/whzbox/state.json
```

(falling back to `~/.local/state/whzbox/state.json` when `$XDG_STATE_HOME` is not set).

The file is written with mode `0600` and the containing directory with mode `0700`. whzbox refuses to load the file
when its permissions are wider than `0600` — this is a defence-in-depth check against sloppy local backup or sync
tools copying state files around with default-wide perms.

**The file is not encrypted.** Anyone with read access to your home directory can read the tokens. If your threat
model requires encryption at rest (e.g. a shared workstation, or a laptop where disk encryption is not in use),
consider:

- running whzbox inside a user account whose home directory is on an encrypted filesystem, or
- running `whzbox logout` before leaving the machine unattended.

AWS sandbox credentials returned by `whzbox create aws` are only ever printed to stdout and never persisted by whzbox
itself. The sandbox's IAM user is destroyed by Whizlabs when the sandbox expires or when you run `whzbox destroy`.

## Network

whzbox talks to two Whizlabs endpoints over HTTPS:

- `https://fq6dv85p2h.execute-api.us-east-1.amazonaws.com` (authentication)
- `https://play.whizlabs.com` (sandbox provisioning)

and to AWS STS in `us-east-1` during credential verification. There are no other outbound connections, no telemetry,
and no auto-update checks.

## Dependencies

whzbox pins exact versions of every dependency via `go.mod`. Updates go through `go mod tidy` and `go get` on a
reviewed branch. If you discover a vulnerable transitive dependency, please report it through the channel above.
