---
title: "How to run AWS CLI commands with whzbox exec"
sidebar_position: 2
description: Use whzbox exec to run direct commands, shell strings, or a subshell with cached AWS sandbox credentials.
---

Use `whzbox exec aws` when you want to run a process with the cached sandbox environment injected.

## Prerequisites

- An unexpired cached AWS sandbox
- The `whzbox` binary on your `PATH`
- The AWS CLI installed if you are running AWS CLI examples

## Steps

### 1. Run a direct argv command

```bash
whzbox exec aws -- aws sts get-caller-identity
```

This runs the child process without going through a shell.

### 2. Run a shell string

```bash
whzbox exec aws -s "aws s3 ls | head"
```

With `-s`, `whzbox` runs `/bin/sh -c <command>`.

### 3. Open an interactive subshell

```bash
whzbox exec aws
```

With no command arguments, `whzbox` launches `$SHELL` or `/bin/sh` if `$SHELL` is unset.

## Verification

To inspect the injected environment directly, run:

```bash
whzbox exec aws -- /usr/bin/env | grep '^AWS_'
```

You should see:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`
- `AWS_DEFAULT_REGION`

## Troubleshooting

### Problem: `whzbox exec` says there is no active sandbox

`exec` only reads the local sandbox cache. Create a sandbox first:

```bash
whzbox create aws
```

### Problem: `-s` rejects the command line

`--shell` takes exactly one command argument. Wrap the whole shell command in one quoted string.

## Related

- [Reference: exec](../reference/commands/exec.md)
- [How to use whzbox in non-interactive scripts](./use-whzbox-in-non-interactive-scripts.md)
