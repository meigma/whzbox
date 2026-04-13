---
title: "How to consume create and list JSON output"
sidebar_position: 7
description: Parse the JSON produced by whzbox create and whzbox list, and avoid assuming JSON mode exists everywhere else.
---

Use `--json` with `create` and `list` when you need stable machine-readable output.

## Prerequisites

- `jq` if you want to use the examples exactly as shown

## Steps

### 1. Capture one created sandbox

```bash
created=$(whzbox create aws --json)
printf '%s\n' "$created" | jq -r '.identity.account'
printf '%s\n' "$created" | jq -r '.credentials.access_key'
```

### 2. Read the cached sandbox list

```bash
whzbox list --json | jq -r '.[] | [.kind, .identity.account, .expires_at] | @tsv'
```

### 3. Treat timestamps as RFC 3339 strings

`started_at` and `expires_at` are emitted from Go `time.Time` values, so they arrive as JSON strings.

## Verification

This prints the JSON type of the list output:

```bash
whzbox list --json | jq -r 'type'
```

You should see:

```text
array
```

## Troubleshooting

### Problem: `status --json` still prints styled output

`--json` is parsed as a global flag, but only `create` and `list` implement JSON rendering today.

### Problem: `create --json` still exits non-zero

If credential verification fails after sandbox creation, `create` still prints the JSON object and returns a provider exit code.

## Related

- [Reference: JSON output schemas](../reference/json-output-schemas.md)
- [Reference: create](../reference/commands/create.md)
- [Reference: list](../reference/commands/list.md)
