# factop

Factorio server operator — a Go service for automating and remotely managing a Factorio headless server.

### Why a softmod instead of a regular mod?

* [grok answer](https://x.com/i/grok/share/9eEbNfDbw9s6PPf7qMjJNIW0l)

### Why this and not [factorio-server-manager](https://github.com/OpenFactorioServerManager/factorio-server-manager)?

* Hot-reload softmod application — stop, patch save, restart is handled automatically
* Embedded [NATS](https://docs.nats.io/nats-concepts/what-is-nats) message bus for all internal and external communication
* Factorio stdin/stdout/stderr exposed as NATS subjects
* RCON managed by the service and exposed over NATS
* Plugin system for extending server behavior with external Go binaries
* UDP bridge for real-time game event telemetry to plugins

## Commands

All commands use NATS request/reply. The CLI in `cmd/main.go` wraps them.

> For brevity, `focmd` is used below. Create an alias (`alias focmd='go run cmd/main.go'`) or build the binary.

### Server Management (`focmd command <name>`)

| Command | Description |
|---------|-------------|
| `status` | Current server state |
| `start` | Start the Factorio server |
| `stop` | Shut down the Factorio server |
| `restart` | Restart the Factorio server |
| `reset` | Delete save and start fresh |
| `latest` | Download the latest Factorio version |
| `list-versions` | List installed Factorio versions |
| `set-version <v>` | Switch Factorio version and restart |

### RCON

| Command | Description |
|---------|-------------|
| `focmd rcon <path>` | Execute Lua file via RCON |
| `focmd prcon <delay> <path>` | Execute Lua file on a repeating timer |
| `focmd lrcon <path>` | Execute via local NATS (localhost) |

### Softmod

`focmd softmod` — packages `softmod/` into a zip, sends it to the server, which stops Factorio, patches the save, and restarts.

### Plugin Management (`focmd plugin <subcommand>`)

| Subcommand | Description |
|------------|-------------|
| `register <name> <binary-path>` | Register a plugin |
| `unregister <name>` | Remove a plugin |
| `start <name>` | Start a plugin |
| `stop <name>` | Stop a plugin |
| `restart <name>` | Restart a plugin |
| `status` | Show all plugin states |
| `list` | List registered plugins |
| `deploy <name> <version> <binary-path>` | Deploy a plugin binary |
| `rollback <name> <version>` | Roll back to a previous version |
| `versions <name>` | List installed versions |
| `version-remove <name> <version>` | Delete a version from disk |

### Monitoring

`focmd watch` — subscribe to `factorio.*`, `udp.*`, `factop.log`, and `plugin.*` NATS subjects in real time.

## Architecture

The service uses `github.com/mlctrez/bind` for dependency injection and component lifecycle. Components are defined in `service/` and wired in `service/service.go`:

| Component | Description |
|-----------|-------------|
| Nats | Embedded NATS server, central message bus |
| Settings | Configuration management (`factop-settings.json`) |
| SoftMod | Injects Lua softmod into the Factorio save zip |
| Factorio | Server process lifecycle with a formal state machine |
| Rcon | RCON connection management, exposed via NATS |
| Command | Server management command dispatch (`factop.command`) |
| PluginManager | External plugin process lifecycle, deployment, versioning (`factop.plugin`) |
| WebServer | HTTP server (port 8080) for health checks |
| UDPBridge | Bridges game UDP traffic (ports 4000/4001) to NATS (`udp.incoming`/`udp.outgoing`) |

### Data Flows

1. **Command & Control** — CLI sends NATS request → service component processes → reply returned
2. **Factorio I/O** — stdout/stderr published to `factorio.stdout`/`factorio.stderr`, stdin writable via `factorio.stdin`
3. **RCON** — detects startup marker on stdout, connects, serves requests on `factop.rcon`
4. **Softmod** — receives zip on `factorio.softmod`, stops server, patches save, restarts
5. **UDP Events** — Lua softmod emits game events via `helpers.send_udp`, UDPBridge publishes to `udp.incoming`, plugins subscribe via NATS
6. **Plugins** — auto-start after Factorio reaches Running state, health-checked every 30s, auto-restart on failure (3 attempts with backoff)

## Project Structure

| Path | Description |
|------|-------------|
| `factop.go` | Service entry point |
| `service/` | Core service components |
| `softmod/` | Lua softmod source (injected into save) |
| `client/` | Typed Go client packages for RCON commands |
| `plugin/` | Plugin SDK for building external plugins |
| `pluginexample/` | Example plugin (lab trail) |
| `cmd/main.go` | CLI tool |
| `apidoc/` | Factorio Lua API doc generator |
| `protodump/` | Prototype name extractor for compile-time validation |

## Getting Started

### Prerequisites

* Go 1.25+

### Build and Deploy

```bash
# Deploy the factop service
focmd service

# Deploy the softmod
focmd softmod

# Build and deploy a plugin
CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/pluginone ./pluginexample/pluginone/
focmd plugin deploy pluginone 0.1.0 /tmp/pluginone
```
