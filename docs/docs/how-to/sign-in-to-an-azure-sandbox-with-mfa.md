---
title: "How to sign in to an Azure sandbox with MFA"
sidebar_position: 3
description: Create an Azure sandbox and fetch the short-lived MFA code required for console login.
---

# How to sign in to an Azure sandbox with MFA

## Prerequisites

- A Whizlabs account with Azure sandbox access
- The `whzbox` binary on your `PATH`

## Steps

### 1. Create the Azure sandbox

```bash
whzbox create azure
```

Keep the console URL, username, and password from the output.

### 2. Generate an MFA code

Run this immediately before signing in:

```bash
whzbox mfa azure
```

The code expires after about 30 seconds. Run the command again if it expires.

### 3. Sign in to Azure

Open the console URL from `create azure`. Enter the displayed username and password, then enter the MFA code.

### 4. Destroy the sandbox when finished

```bash
whzbox destroy --yes
```

## Troubleshooting

### Problem: there is no active Azure sandbox

`mfa` reads the local Azure cache before it contacts Whizlabs. Create or reuse an Azure sandbox first:

```bash
whzbox create azure
```

### Problem: the MFA code expired

Generate another code. MFA codes are not cached:

```bash
whzbox mfa azure
```

## Related

- [Reference: mfa](../reference/commands/mfa.md)
- [Reference: create](../reference/commands/create.md)
