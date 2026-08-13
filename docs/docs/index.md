---
title: whzbox documentation
sidebar_label: Overview
sidebar_position: 1
slug: /
---

# whzbox documentation

`whzbox` is a Go CLI for Whizlabs AWS, Azure, and GCP cloud sandboxes:

- `create aws` returns console and programmatic credentials.
- `create azure` returns browser-console credentials and resource-group metadata. Use `mfa azure` to get a current login code.
- `create gcp` returns browser-console credentials and project metadata. It does not return credentials for `gcloud` or SDKs.
- `create` can reuse a cached, unexpired sandbox instead of provisioning a new one.
- `exec` supports AWS only. `list` supports all three providers.
- Whizlabs permits one active sandbox per user. Destroy it before creating a different provider.
- `logout` clears session tokens but preserves cached sandboxes.
- `status` is session-only today. It does not report live sandbox state.
- `--json` is only implemented by `create` and `list`.
- AWS and Azure `--duration` values are rounded up to whole hours. Azure accepts `1h` through `3h`; GCP accepts `1h` only.

This site is organized with the Diataxis model:

- [Tutorials](tutorials/index.md): learn one complete workflow end to end
- [How-to Guides](how-to/index.md): solve specific tasks quickly
- [Reference](reference/index.md): look up commands, flags, env vars, schemas, and exit codes
- [Explanation](explanation/index.md): understand the design decisions and operational tradeoffs

Start with the tutorial if you have not used `whzbox` before:

- [Tutorial: Create, use, and destroy your first AWS sandbox](./tutorials/create-use-and-destroy-your-first-aws-sandbox.md)
