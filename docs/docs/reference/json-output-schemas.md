---
title: JSON output schemas
sidebar_position: 4
description: Reference for the JSON emitted by whzbox create and whzbox list.
---

# JSON output schemas

`whzbox` currently implements JSON rendering for `create` and `list`.

## Commands

| Command | Top-level JSON type |
| --- | --- |
| `whzbox create aws --json` | object |
| `whzbox create azure --json` | object |
| `whzbox create gcp --json` | object |
| `whzbox list --json` | array of objects |

## Sandbox object schema

| Field | Type | Description |
| --- | --- | --- |
| `kind` | string | Sandbox kind: `aws`, `azure`, or `gcp`. |
| `slug` | string | Whizlabs provider slug: `aws-sandbox`, `azure-sandbox`, or `gcp-sandbox`. |
| `credentials` | object | Programmatic credentials. Both values are empty for Azure and GCP. |
| `console` | object | Browser console login details. |
| `identity` | object | Verified AWS identity, Azure resource groups, or GCP project metadata. |
| `started_at` | string | Sandbox start time as an RFC 3339 timestamp. |
| `expires_at` | string | Sandbox expiry time as an RFC 3339 timestamp. |

### `credentials`

| Field | Type | Description |
| --- | --- | --- |
| `access_key` | string | AWS access key ID |
| `secret_key` | string | AWS secret access key |

### `console`

| Field | Type | Description |
| --- | --- | --- |
| `url` | string | Provider console sign-in URL |
| `username` | string | Console username |
| `password` | string | Console password |

### `identity`

| Field | Type | Description |
| --- | --- | --- |
| `account` | string | AWS account ID |
| `user_id` | string | AWS user ID |
| `arn` | string | AWS ARN |
| `region` | string | AWS region |
| `project_id` | string | GCP project ID; omitted for AWS |
| `project_name` | string | GCP project name; omitted for AWS |
| `resource_groups` | array of strings | Azure resource groups; omitted for AWS and GCP |

## Example: `create --json`

```json
{
  "kind": "aws",
  "slug": "aws-sandbox",
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
```

## Example: `list --json`

```json
[
  {
    "kind": "aws",
    "slug": "aws-sandbox",
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
]
```

## Example: GCP `create --json`

```json
{
  "kind": "gcp",
  "slug": "gcp-sandbox",
  "credentials": {
    "access_key": "",
    "secret_key": ""
  },
  "console": {
    "url": "https://console.cloud.google.com/",
    "username": "student@example.com",
    "password": "password"
  },
  "identity": {
    "account": "",
    "user_id": "",
    "arn": "",
    "region": "",
    "project_id": "project-12345",
    "project_name": "project-12345"
  },
  "started_at": "2026-04-12T00:00:00Z",
  "expires_at": "2026-04-12T01:00:00Z"
}
```

## Example: Azure `create --json`

```json
{
  "kind": "azure",
  "slug": "azure-sandbox",
  "credentials": {
    "access_key": "",
    "secret_key": ""
  },
  "console": {
    "url": "https://portal.azure.com/",
    "username": "student@example.com",
    "password": "password"
  },
  "identity": {
    "account": "",
    "user_id": "",
    "arn": "",
    "region": "",
    "resource_groups": [
      "rg-compute",
      "rg-network",
      "rg-storage"
    ]
  },
  "started_at": "2026-04-12T00:00:00Z",
  "expires_at": "2026-04-12T03:00:00Z"
}
```

## Notes

- `status`, `login`, `logout`, `destroy`, `exec`, `mfa`, `version`, and `completion` do not have JSON renderers.
- `create --json` can still return a non-zero exit code if verification fails after creation.

## See also

- [How to consume create and list JSON output](../how-to/consume-create-and-list-json-output.md)
- [Reference: create](./commands/create.md)
