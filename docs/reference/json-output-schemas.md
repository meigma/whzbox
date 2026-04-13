---
title: JSON output schemas
sidebar_position: 4
description: Reference for the JSON emitted by whzbox create and whzbox list.
---

`whzbox` currently implements JSON rendering for `create` and `list`.

## Commands

| Command | Top-level JSON type |
| --- | --- |
| `whzbox create aws --json` | object |
| `whzbox list --json` | array of objects |

## Sandbox object schema

| Field | Type | Description |
| --- | --- | --- |
| `kind` | string | Sandbox kind, currently `aws`. |
| `slug` | string | Whizlabs provider slug, currently `aws-sandbox`. |
| `credentials` | object | Programmatic credentials. |
| `console` | object | Browser console login details. |
| `identity` | object | Verified identity data, when available. |
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
| `url` | string | AWS console sign-in URL |
| `username` | string | Console username |
| `password` | string | Console password |

### `identity`

| Field | Type | Description |
| --- | --- | --- |
| `account` | string | AWS account ID |
| `user_id` | string | AWS user ID |
| `arn` | string | AWS ARN |
| `region` | string | AWS region |

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

## Notes

- `status`, `login`, `logout`, `destroy`, `exec`, `version`, and `completion` do not have JSON renderers.
- `create --json` can still return a non-zero exit code if verification fails after creation.

## See also

- [How to consume create and list JSON output](../how-to/consume-create-and-list-json-output.md)
- [Reference: create](./commands/create.md)
