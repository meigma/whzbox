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

### 2. Select verification-code authentication

Open the console URL from `create azure` and enter the displayed username and password. At the **Approve sign in request** prompt, select **Use a verification code**.

### 3. Generate and enter an MFA code

Run this after the verification-code field appears:

```bash
whzbox mfa azure
```

Enter the displayed code and select **Verify**. The code expires after about 30 seconds. Run the command again if it expires.

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

### Problem: Microsoft asks you to approve the sign-in request

The approval prompt is for Authenticator number matching, not the code from `whzbox`. Select **Use a verification code**. If that option is not visible, select **Sign in another way**, then select **Use a verification code**.

## Related

- [Reference: mfa](../reference/commands/mfa.md)
- [Reference: create](../reference/commands/create.md)
