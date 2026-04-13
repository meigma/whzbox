---
title: "How to use whzbox in non-interactive scripts"
sidebar_position: 3
description: Run whzbox safely from automation by relying on cached sessions, JSON output, and explicit non-interactive flags.
---

Use `whzbox` in scripts only with commands that do not require an interactive prompt, or with a valid cached session that was created earlier from a real terminal.

## Prerequisites

- A valid cached session created by `whzbox login` in an interactive terminal
- `jq` if you want to parse JSON examples

## Steps

### 1. Use JSON output for machine-readable data

```bash
sandbox_json=$(whzbox create aws --json)
printf '%s\n' "$sandbox_json" | jq -r '.credentials.access_key'
```

### 2. Use cache-only commands when you do not want auth side effects

```bash
whzbox list --json
whzbox status
whzbox exec aws -- aws sts get-caller-identity
```

`list` and `exec` do not talk to Whizlabs. `status` reads the cached session only.

### 3. Pass `--yes` for non-interactive destroy

```bash
whzbox destroy --yes
```

Without `--yes`, `destroy` returns a non-interactive error instead of waiting on a prompt.

## Verification

Check the command exit status from the shell:

```bash
whzbox exec aws -- false
echo $?
```

The exit code from the child command is propagated by `whzbox exec`.

## Troubleshooting

### Problem: `login` fails in CI or cron

`whzbox login` is interactive only. Run it once from a real terminal first, or provide a pre-existing state directory.

### Problem: `status` does not show a sandbox

That is expected. `status` reports the cached session only.

### Problem: `--json` does not change the output

`--json` is only implemented by `create` and `list`.

## Related

- [Reference: JSON output schemas](../reference/json-output-schemas.md)
- [Reference: exit codes](../reference/exit-codes.md)
- [About why status is session-only](../explanation/about-why-status-is-session-only.md)
