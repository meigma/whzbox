---
title: "Tutorial: Create, use, and destroy your first AWS sandbox"
sidebar_position: 1
description: Learn the full whzbox workflow by signing in, creating an AWS sandbox, running a command inside it, and tearing it down.
---

In this tutorial, we will sign in to Whizlabs, create an AWS sandbox, run one command inside it, and destroy it when we are done. By the end, you will have used the main `whzbox` flow from start to finish.

## Prerequisites

- A Whizlabs account
- A real interactive terminal
- The `whzbox` binary on your `PATH`
- The AWS CLI installed if you want to run the `exec` example exactly as shown

If you already have an active sandbox that you want to clear first, use [How to destroy a sandbox without prompts](../how-to/destroy-a-sandbox-without-prompts.md).

## What We Are Building

We will create one temporary AWS sandbox, use its credentials through `whzbox exec`, confirm that `whzbox` cached it locally, and then destroy it.

## Step 1: Sign in

Run:

```bash
whzbox login
```

You should see an interactive prompt for your email and password. When it succeeds, the command returns to the shell without printing an error.

## Step 2: Create an AWS sandbox

Run:

```bash
whzbox create aws
```

You should see a rendered sandbox summary that includes:

- an AWS account number
- an IAM user or user ID
- console login details
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- an expiration timestamp

The output ends with:

```text
Destroy with:  whzbox destroy
```

## Step 3: Confirm that the sandbox is cached locally

Run:

```bash
whzbox list
```

You should see one row for `aws` with a status of `active`.

## Step 4: Run one command inside the sandbox

Run:

```bash
whzbox exec aws -- aws sts get-caller-identity
```

You should see AWS STS output from the child command. The exact JSON varies, but it includes fields such as `Account`, `Arn`, and `UserId`.

## Step 5: Destroy the sandbox

Run:

```bash
whzbox destroy
```

You should see a confirmation prompt. Confirm the action.

When the command completes, run:

```bash
whzbox list
```

You should see:

```text
(no sandboxes cached)
```

## What We Learned

- How to sign in with `whzbox login`
- How to provision an AWS sandbox with `whzbox create aws`
- How to confirm cached local state with `whzbox list`
- How to run a command with sandbox credentials through `whzbox exec`
- How to clean up with `whzbox destroy`

## Next Steps

- [How to create a sandbox with a specific duration](../how-to/create-a-sandbox-with-a-specific-duration.md)
- [How to run AWS CLI commands with whzbox exec](../how-to/run-aws-cli-commands-with-whzbox-exec.md)
- [About sandbox caching and reuse](../explanation/about-sandbox-caching-and-reuse.md)
