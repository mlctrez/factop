---
inclusion: auto
description: Build and deployment guidelines for compiling Go binaries targeting the factop server.
---

# Server Build Guidelines

The factop server runs a different Linux C library version than the
development workstation. All Go binaries deployed to the server (the
factop service, plugins, etc.) must be compiled with CGO disabled to
produce fully static binaries that avoid glibc compatibility issues.

## Required Build Flags

Set `CGO_ENABLED=0` when building any binary destined for the server:

```bash
CGO_ENABLED=0 go build -o <output> <package>
```

Stripping debug info with `-ldflags "-s -w"` is recommended to reduce
binary size for NATS-based plugin deploys:

```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -o <output> <package>
```

## Examples

Service build and deploy:

```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/factop factop.go
```

Plugin build:

```bash
CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/pluginone ./pluginexample/pluginone/
```

## Build Output Location

Never place compiled binaries inside the source tree. Use `/tmp` or another
directory outside the repository to avoid accidentally committing them to
source control.

## Why Not CGO?

The factop codebase has no dependencies that require CGO. Disabling it
produces a statically linked binary that runs on any Linux regardless of
the installed glibc version.
