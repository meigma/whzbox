---
title: whzbox documentation
sidebar_label: Overview
sidebar_position: 1
slug: /
---

`whzbox` is a Go CLI for Whizlabs cloud sandboxes. The current implementation is AWS-only and centers on a few behaviors that matter when you use it in practice:

- `create aws` can reuse a cached, unexpired sandbox instead of provisioning a new one.
- `exec` and `list` are cache-backed. They do not call Whizlabs.
- `logout` clears session tokens but preserves cached sandboxes.
- `status` is session-only today. It does not report live sandbox state.
- `--json` is only implemented by `create` and `list`.
- `--duration` accepts Go durations but is rounded up to whole hours for the upstream API.

This site is organized with the Diataxis model:

- [Tutorials](/tutorials): learn one complete workflow end to end
- [How-to Guides](/how-to): solve specific tasks quickly
- [Reference](/reference): look up commands, flags, env vars, schemas, and exit codes
- [Explanation](/explanation): understand the design decisions and operational tradeoffs

Start with the tutorial if you have not used `whzbox` before:

- [Tutorial: Create, use, and destroy your first AWS sandbox](./tutorials/create-use-and-destroy-your-first-aws-sandbox.md)
