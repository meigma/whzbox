---
title: version
sidebar_position: 8
description: Reference for whzbox version and the root --version flag.
---

`whzbox version` prints the build metadata embedded in the current binary.

## Syntax

```text
whzbox version
whzbox --version
```

## Arguments

This command takes no positional arguments.

## Flags

This command uses only the global flags.

## Behavior

- Prints the version string assembled from the embedded `Version`, `Commit`, and `BuildTime` values.
- Uses a lightweight app path that does not initialize the state store or network adapters.
- Binaries built directly from source without linker flags report the fallback values `dev`, `none`, and `unknown`.
- GoReleaser snapshot builds embed `v0.0.0-dev` plus the current commit and build timestamp.
- Tagged GoReleaser release builds embed the Git tag version plus the corresponding commit and build timestamp.

## Output

The output format is:

```text
whzbox <version> (<commit>) built <build-time>
```

When the binary is built without linker flags, the default values are:

- version: `dev`
- commit: `none`
- build time: `unknown`

## Exit behavior

- `0` on success
- `1` on generic command failure

## See also

- [Reference: global flags and environment variables](../global-flags-and-environment-variables.md)
