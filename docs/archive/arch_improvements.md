# Factop Architecture Analysis & Improvement Areas

## How It All Fits Together

```
Factorio Server ←→ Go Service (embedded NATS) ←→ Plugins / CLI / Clients
     ↕ RCON (commands)         ↕ NATS subjects          ↕ NATS subjects
     ↕ UDP (events)            ↕ factop.rcon             ↕ udp.incoming
     ↕ stdout/stderr           ↕ factop.command          ↕ plugin.<name>.health
```

The Lua softmod registers RCON commands and emits UDP events. The Go service bridges both into NATS. Clients and plugins connect to NATS and use typed Go packages that format RCON commands and parse wire-format responses.

### Communication: Factorio ↔ Go Service

**RCON (Command Path):**
- Factorio starts with `--rcon-bind` and a random UUID password
- `service/rcon.go` subscribes to `factorio.stdout` and watches for `"Starting RCON interface at IP ADDR"` marker
- On detection, it dials the RCON connection with a large payload limit (16384 bytes) and a nonzero command ID (to skip console noise)
- RCON requests arrive on NATS subject `factop.rcon` → `Rcon.handle()` executes via the RCON connection → replies with the result
- A regex filter (`IgnorePattern`) strips timestamped console messages from RCON output

**UDP (Event Path):**
- Factorio starts with `--enable-lua-udp <port>` (default 4001 outgoing from Factorio's perspective)
- Lua softmod modules call `helpers.send_udp(4000, msg, 0)` to emit events (port 4000 = incoming to the Go service)
- `service/udpbridge.go` listens on UDP port 4000, publishes each packet to NATS `udp.incoming`
- Outgoing: subscribes to NATS `udp.outgoing`, sends UDP packets to port 4001 (for `helpers.recv_udp`)
- `softmod/factop/udp.lua` handles `on_udp_packet_received` and calls `helpers.recv_udp(0)` every 20 ticks

**Stdout/Stderr/Stdin:**
- `service/factorio.go` pipes stdout/stderr through `bufio.Scanner`, publishing each line to `factorio.stdout`/`factorio.stderr`
- Stdin is writable via `factorio.stdin` NATS subject

### NATS Subject Map

| Subject | Purpose |
|---------|---------|
| `factop.rcon` | RCON request/reply (clients send commands here) |
| `factop.command` | Server management (start/stop/restart/reset/versions) |
| `factop.plugin` | Plugin management commands |
| `factop.plugin.deploy.>` | Chunked binary deploys |
| `factop.log` | Structured JSON log messages |
| `factorio.stdout` | Server stdout lines |
| `factorio.stderr` | Server stderr lines |
| `factorio.stdin` | Write to server stdin |
| `factorio.softmod` | Softmod zip upload (triggers stop/patch/restart) |
| `udp.incoming` | Game events from Lua → Go/plugins |
| `udp.outgoing` | Messages from Go/plugins → Lua |
| `plugin.<name>` | Plugin stdout/stderr output |
| `plugin.<name>.health` | Plugin health check request/reply |

### Client Package Structure

**`client/client.go`** - Core connection wrapper:
- `Conn` wraps a `*nats.Conn` with a configurable timeout
- `Dial(natsURL)` connects to NATS
- `Rcon(command)` sends a NATS request to `factop.rcon`, handles timeout and error headers

**Sub-packages** (each mirrors a Lua softmod module):
- Each has a `Client` struct holding `*client.Conn`
- Each has a `New(conn)` constructor
- Methods format RCON command strings, call `c.conn.Rcon(cmd)`, and parse the wire-format response
- Wire formats are compact colon-separated strings (e.g., `name:x:y:unit_number`)

| Package | Lua Module | Commands |
|---------|-----------|----------|
| `client/entity` | `factop/entity.lua` | entity-create, entity-bulk, entity-find, entity-count, entity-destroy |
| `client/game` | `factop/game.lua` | game-players, game-players-all, game-kick, game-remove |
| `client/player` | `factop/player.lua` | player-respawn, player-teleport |
| `client/playerattr` | `factop/playerattr.lua` | playerattr-set/get/del/list/keys |
| `client/resource` | `factop/resource.lua` | resource-count/find/set, pollution-get/set/total |
| `client/surface` | `factop/surface.lua` | surface-list/info/set/generate/delete-chunk/clear-pollution |
| `client/tile` | `factop/tile.lua` | tile-fill/read/remove/replace/checker |

Additionally, `client/entity` and `client/player` include **UDP event parsers** (`ParseEntityEvent`, `ParseMove`) for consuming `udp.incoming` messages.

### Softmod Lua Module Pattern

Each module follows a consistent pattern:
1. `require("factop.common")` for shared helpers (`rcon_only`, `reply`, `parse_area`, etc.)
2. Register RCON commands in `register_commands()` (called from both `on_init` and `on_load`)
3. All commands check `c.rcon_only(cmd)` to reject in-game player invocations
4. Event handlers emit UDP messages via `helpers.send_udp(4000, msg, 0)` with a 512-byte cap
5. Return a module table with `.events`, `.on_init`, `.on_load`, and optionally `.on_nth_tick`

**UDP event formats:**
- `[entity-died] name:x:y:unit_number:surface_name:surface_index:player_index:cause`
- `[entity-built] name:x:y:unit_number:surface_name:surface_index:player_index`
- `[entity-mined] name:x:y:unit_number:surface_name:surface_index:player_index`
- `[move] #<index> <name> name:index:x:y:surface_name:surface_index`
- `[join] #<index> <name>`
- `[leave] #<index> <name>`
- `[death] #<index> <name> by <cause>`
- `[respawn] #<index> <name>`

### Plugin System

**Plugin SDK (`plugin/plugin.go`, `plugin/context.go`):**
- Plugins implement `Plugin` interface: `Setup(Context)` and `Run(Context)`
- Call `plugin.Run(impl)` from `main()` — handles flags, NATS connection, health checks, signal handling
- `Context` provides: `Rcon()`, `Subscribe()`, `Publish()`, `DataDir()`, `Logger()`, `Done()`, `Clients()`
- `Clients()` returns lazily-initialized typed client accessors (all sub-packages)
- Health check: responds on `plugin.<name>.health` with name/version/uptime JSON

**Plugin Manager (`service/plugin.go`, `service/registry.go`):**
- Manages external plugin processes as child processes
- State machine: Stopped → Starting → Running → Stopping (+ Errored)
- Auto-starts enabled plugins when Factorio reaches Running state
- Health checks every 30s; 3 consecutive failures → kill + restart
- Restart with exponential backoff (2s, 4s, 8s, max 3 attempts)
- Supports: register, unregister, start, stop, restart, deploy, rollback, versions, version-remove
- Chunked deploy support for large binaries via `factop.plugin.deploy.<name>.<version>`
- Plugin stdout/stderr piped to NATS `plugin.<name>` subject

### Service Layer

Uses `github.com/mlctrez/bind` for DI and lifecycle management. Components are added in order:
`LogHandler → Nats → Settings → SoftMod → Factorio → Rcon → Command → PluginManager → WebServer → UDPBridge`

Each component implements `bind.Startup` and optionally `bind.Shutdown`. The binder calls `Startup()` in order and `Shutdown()` in reverse.

---

## Areas of Improvement

### 1. `udp.lua` is a conflicting stub

`udp.lua` registers handlers for `on_player_joined_game` and `on_player_changed_position` — but these are empty no-ops (`event_handler` just has commented-out prints). Meanwhile, `player.lua` registers real handlers for the same events. Factorio's `event_handler` library should merge these, but it's confusing and fragile.

**Recommendation:** Either give `udp.lua` a clear purpose (handling `udp.outgoing` → Lua commands from plugins) or remove the duplicate event registrations. Right now it's dead weight that could mask bugs if the event handler library doesn't merge correctly.

### 2. No real `udp.outgoing` → Lua pipeline

The UDPBridge subscribes to `udp.outgoing` and sends UDP to port 4001, and `udp.lua` calls `helpers.recv_udp(0)` every 20 ticks — but `packet_received` is a no-op. Plugins can publish to `udp.outgoing`, but nothing in Lua actually processes those messages.

**Recommendation:** Define a command dispatch protocol for `udp.outgoing`. For example, plugins could send structured messages that trigger Lua-side actions (spawn items, display alerts, etc.) without going through RCON. This would enable low-latency plugin→game communication that doesn't serialize through the single RCON connection.

### 3. RCON is a single-threaded bottleneck

All RCON commands serialize through one mutex-protected connection (`Rcon.handle`). If a plugin does a burst of `entity-bulk` calls while another plugin queries `game-players`, they queue up. The 16KB payload limit and single-connection design means high-throughput plugins (like the lab trail placing tiles on every movement) can starve other consumers.

**Recommendation:** Consider a priority system or at least document the contention model. For plugins that do frequent writes, batch operations (like `entity-bulk`) are already the right pattern — but the client SDK could expose a `BatchRcon` method that sends multiple commands in a single NATS request, letting the service execute them sequentially without per-message NATS overhead.

### 4. Plugin SDK lacks event type routing

Plugins subscribe to raw `udp.incoming` and manually parse the `[event-name]` prefix (see `pluginone` checking `strings.HasPrefix(msg, "[move]")`). Every plugin gets every UDP message and filters locally.

**Recommendation:** Add a typed event subscription layer to the plugin SDK:

```go
ctx.OnPlayerMove(func(mv player.Move) { ... })
ctx.OnEntityDied(func(ev entity.EntityEvent) { ... })
```

This could be implemented purely client-side (subscribe to `udp.incoming`, parse, dispatch) without service changes. It would eliminate boilerplate and parsing errors in every plugin.

### 5. No structured error types from RCON

Client methods return `(string, error)` or `([]T, error)`, but RCON errors are just string messages ("Player not found", "Surface not found"). There's no way to distinguish a "not found" from a "server error" programmatically.

**Recommendation:** Define sentinel errors or an error type in the client package:

```go
var ErrNotFound = errors.New("not found")
var ErrRconDisconnected = errors.New("rcon not connected")
```

The `Conn.Rcon` method could inspect common error patterns and wrap them.

### 6. Wire format fragility with colons in names

The colon-separated wire format (`name:x:y:unit_number`) assumes entity/tile/surface names never contain colons. Factorio mod names can technically contain colons (e.g., `mod-name:entity`).

**Recommendation:** Use `SplitN` with the correct field count (which the parsers already do — good). But the Lua side should also escape or reject names with colons, or switch to a delimiter that's guaranteed safe (like `\x1F` unit separator). This is low-risk today with vanilla Factorio but worth noting for mod compatibility.

### 7. `handleOutgoing` creates a new UDP connection per message

In `udpbridge.go`, every outgoing message dials a fresh UDP connection. For a protocol that might eventually see high throughput (plugin→Lua commands), this is wasteful.

**Recommendation:** Keep a persistent `*net.UDPConn` for outgoing, created at startup. UDP is connectionless anyway — a single `DialUDP` can be reused for all writes.

### 8. Plugin health check has a 30-second blind spot

A plugin transitions from `Starting` → `Running` only on the first successful health check, which fires every 30 seconds. A plugin could be fully operational for 29 seconds before the manager knows.

**Recommendation:** Add an initial health check with a shorter delay (e.g., 3 seconds after launch), then fall back to the 30-second interval. Or let plugins self-report readiness by publishing to a `plugin.<name>.ready` subject.

### 9. `softmod.go` module load ordering is undefined

`BuildControlLua()` walks the embedded filesystem and adds `add_lib()` calls in filesystem order. The comment `// TODO: these may need ordering` acknowledges this. Today it works because modules only depend on `common.lua` (which is excluded from `add_lib`), but if you add a module that depends on another module's storage initialization, you'll hit load-order bugs.

**Recommendation:** Either enforce alphabetical ordering (which is predictable) or add a simple priority mechanism — e.g., a numeric prefix convention (`01_player.lua`, `02_entity.lua`) or a manifest file.

### 10. `goal.lua` uses global-style naming

`goal.lua` uses `factop_goal` as a global table name and `factop_goal_setup` as a global function, unlike every other module which uses `local xxx_mod = {}`. This is inconsistent and pollutes the global namespace.

**Recommendation:** Refactor to match the local module pattern used everywhere else (`local goal_mod = {}`).

### 11. No plugin-to-plugin communication convention

Plugins can publish/subscribe to arbitrary NATS subjects, but there's no documented pattern for inter-plugin messaging. If two plugins need to coordinate (e.g., a "zones" plugin and a "permissions" plugin), they'd have to invent their own subject naming.

**Recommendation:** Document a convention like `plugin.<source>.<topic>` for inter-plugin events, and optionally add a `ctx.PublishEvent(topic, data)` helper that namespaces automatically.

### 12. `client/test/test.go` is a standalone script, not a test

`client/test/test.go` is a `package main` that builds a maze — it's an integration test / demo script, not a Go test. Its location under `client/test/` is misleading.

**Recommendation:** Move it to something like `cmd/mazetest/main.go` or `example/maze/main.go` to clarify its purpose.

### 13. Missing `context.Context` propagation in client SDK

The plugin SDK's `contextImpl` has a `done` channel but no `context.Context`. If a plugin wants to use Go libraries that accept `context.Context` (HTTP clients, database drivers, etc.), they have to wrap the done channel manually.

**Recommendation:** Add a `Context() context.Context` method to the plugin `Context` interface that returns a context cancelled when `done` closes.

---

## Summary of Priorities

| Priority | Area | Effort |
|----------|------|--------|
| High | Typed event routing in plugin SDK | Medium |
| High | Clean up `udp.lua` conflicts | Low |
| Medium | Persistent outgoing UDP connection | Low |
| Medium | Plugin readiness signaling | Low |
| Medium | `context.Context` in plugin SDK | Low |
| Medium | Structured RCON error types | Medium |
| Low | Module load ordering | Low |
| Low | `goal.lua` naming cleanup | Low |
| Low | Move `client/test/` | Low |
| Low | Inter-plugin convention docs | Low |

The biggest wins for plugin authors would be the typed event routing (eliminates boilerplate and parsing in every plugin) and adding `context.Context` support. For the softmod side, cleaning up `udp.lua` and defining the outgoing UDP protocol would unlock bidirectional plugin↔game communication beyond RCON.
