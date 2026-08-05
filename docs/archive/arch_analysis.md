# Architecture Re-imagination: Client ↔ Softmod Interaction

## Current State Assessment

The current architecture was a solid first pass that proved the concept works:

- Lua registers RCON commands with hand-crafted argument parsing
- Go client packages hand-craft command strings and hand-parse colon-separated responses
- Plugins subscribe to raw UDP bytes and manually pattern-match event prefixes
- Each new capability requires touching 3+ files: Lua command, Go client method, Go parser, Go test

This approach has worked for ~12 commands across 7 modules, but it won't scale. Adding 20 more commands means 20 more hand-written parsers, 20 more string-formatting methods, and 20 more opportunities for the Lua and Go sides to drift out of sync.

---

## Core Problems to Solve

1. **Symmetry maintenance** — Lua command signatures and Go client methods must stay in sync manually
2. **Boilerplate amplification** — Every new command requires the same scaffolding in 3 places
3. **Weak type safety** — Wire formats are stringly-typed; parse errors surface at runtime
4. **Event consumption is primitive** — Raw byte subscriptions with manual prefix matching
5. **Single communication channel** — Everything funnels through one RCON connection
6. **No bidirectional plugin↔game flow** — Plugins can't push actions to Lua without RCON

---

## Option A: Code Generation from a Shared Schema

### Concept

Define each command/event once in a schema file. Generate both the Lua command registration and the Go client code from it.

### Schema Example (YAML or similar)

```yaml
modules:
  entity:
    commands:
      create:
        args:
          - name: position
            type: position
            required: true
          - name: entity_name
            type: string
            required: true
          - name: force
            type: string
            default: "player"
          - name: direction
            type: direction
          - name: surface
            type: surface
        response:
          type: string
          format: "Created {name} at {position} unit_number={unit_number}"

      find:
        args:
          - name: area
            type: area
            required: true
          - name: name
            type: string
            skip: "_"
          - name: type
            type: string
            skip: "_"
          - name: force
            type: string
            skip: "_"
          - name: limit
            type: int
            skip: "_"
          - name: surface
            type: surface
        response:
          type: list
          wire: "name:x:y:unit_number"
          struct:
            name: string
            x: float64
            y: float64
            unit_number: uint64

    events:
      entity-died:
        wire: "name:x:y:unit_number:surface_name:surface_index:player_index:cause"
        fields:
          name: string
          x: float64
          y: float64
          unit_number: uint64
          surface_name: string
          surface_index: int
          player_index: int
          cause: string

      entity-built:
        wire: "name:x:y:unit_number:surface_name:surface_index:player_index"
        fields:
          name: string
          x: float64
          y: float64
          unit_number: uint64
          surface_name: string
          surface_index: int
          player_index: int
```

### What Gets Generated

| Output | From Schema |
|--------|-------------|
| `softmod/factop/entity.lua` | Command registration, argument parsing, response formatting |
| `client/entity/entity.go` | Client struct, typed methods, wire format parsers |
| `client/entity/entity_test.go` | Table-driven parser tests with example wire strings |
| `plugin/events_gen.go` | Typed event structs and dispatcher registration |

### Advantages

- **Single source of truth** — schema defines the contract, both sides are always in sync
- **Zero boilerplate for new commands** — add to schema, run `go generate`, done
- **Compile-time correctness** — generated code is always structurally valid
- **Schema can drive documentation** — auto-generate command reference docs
- **Wire format changes propagate instantly** — update schema, regenerate both sides

### Disadvantages

- **Upfront investment** — need to build the generator (probably 500-800 lines of Go)
- **Lua generation complexity** — Lua templates need to handle edge cases (optional args, skip placeholders)
- **Schema language design** — need to be expressive enough without becoming a DSL nightmare
- **Harder to debug generated code** — when something goes wrong, you're reading generated output
- **Custom logic escapes** — some commands (like `entity-bulk` with batching) have logic that doesn't fit a simple template

### Verdict

Best option if the project is going to grow beyond ~20 commands. The generator pays for itself after the 3rd or 4th generated module.

---

## Option B: Drop RCON Commands, Use `/sc` (Silent Command) Directly

### Concept

Instead of registering custom Factorio commands, execute arbitrary Lua via `/sc` (silent command) through RCON. The Go client constructs Lua snippets that run server-side and return results.

### How It Works Today

```
Go Client → NATS → Rcon handler → RCON "/entity-find 0,0,10,10 _" → Lua command handler → response
```

### Proposed Flow

```
Go Client → NATS → Rcon handler → RCON "/sc <lua code>" → Factorio Lua VM → response via rcon.print
```

### Example

Instead of:
```go
func (c *Client) Find(area Area, opts FindOptions) ([]Entity, error) {
    cmd := fmt.Sprintf("/entity-find %s %s %s %s", area, ...)
    raw, err := c.conn.Rcon(cmd)
    return Parse(raw)
}
```

The client would send:
```go
func (c *Client) Find(area Area, opts FindOptions) ([]Entity, error) {
    lua := fmt.Sprintf(`
        local r=game.surfaces["nauvis"].find_entities_filtered({area={{%g,%g},{%g,%g}}})
        local p={}
        for _,e in ipairs(r) do p[#p+1]=e.name..":"..e.position.x..":"..e.position.y..":"..tostring(e.unit_number or 0) end
        rcon.print(table.concat(p,","))
    `, area.X1, area.Y1, area.X2, area.Y2)
    raw, err := c.conn.Rcon("/sc " + lua)
    return Parse(raw)
}
```

### Advantages

- **No Lua softmod command registration needed** — eliminates half the codebase
- **Full Factorio API access** — any Lua API call is available without pre-registering a command
- **Faster iteration** — change Go code only, no softmod redeploy needed for new commands
- **No command name collisions** — no `commands.add_command` that errors on duplicates
- **Can compose complex operations** — multi-step Lua logic in a single RCON call

### Disadvantages

- **Security boundary disappears** — any RCON consumer can execute arbitrary Lua
- **Lua injection risk** — if inputs aren't sanitized, malformed strings could inject Lua code
- **Payload size limits** — complex Lua snippets eat into the 16KB RCON limit
- **No persistent state between calls** — can't maintain Lua-side state without `storage`
- **Lose event handling** — still need the softmod for event subscriptions and UDP emission
- **Debugging is harder** — Lua errors in `/sc` show up as opaque RCON responses
- **Performance** — `/sc` compiles Lua on every call vs pre-registered commands that are already compiled

### Hybrid Approach

Keep the softmod for **events only** (UDP emission on game events) and use `/sc` for all **queries and mutations**. This eliminates command registration entirely while preserving the event pipeline.

```
Commands: Go → /sc Lua snippets → response
Events:   Lua event handlers → UDP → NATS → plugins
```

### Verdict

Good for rapid prototyping and full API coverage. The security/injection concerns are manageable since RCON is already a trusted channel (only accessible from localhost). However, embedding Lua strings in Go is ugly and error-prone at scale — which leads to Option C.

---

## Option C: Lua Template Library + `/sc` Execution

### Concept

Combine Option B's `/sc` approach with a Go-side Lua template library. Pre-compile Lua snippets as templates, fill in parameters safely, and execute via `/sc`.

### Implementation

```go
// In client/lua package
type Template struct {
    source string
    // compiled template with parameter placeholders
}

// Parameterized, injection-safe
var FindEntities = lua.MustCompile(`
    local filter = {area={{$x1,$y1},{$x2,$y2}}}
    if $name ~= "_" then filter.name = $name end
    if $type ~= "_" then filter.type = $type end
    local r = game.surfaces[$surface].find_entities_filtered(filter)
    local p = {}
    for _, e in ipairs(r) do
        p[#p+1] = e.name..":"..string.format("%.1f",e.position.x)..":"..
                   string.format("%.1f",e.position.y)..":"..tostring(e.unit_number or 0)
    end
    rcon.print(table.concat(p,","))
`)

func (c *Client) Find(area Area, opts FindOptions) ([]Entity, error) {
    lua := FindEntities.Execute(map[string]any{
        "x1": area.X1, "y1": area.Y1, "x2": area.X2, "y2": area.Y2,
        "name": opts.Name, "type": opts.Type, "surface": "nauvis",
    })
    raw, err := c.conn.Rcon("/sc " + lua)
    return Parse(raw)
}
```

### Advantages

- Everything from Option B, plus:
- **Injection-safe** — template engine handles quoting/escaping
- **Readable Lua** — templates are proper Lua, not string concatenation
- **Testable** — can unit test template output without a running server
- **Cacheable** — compiled templates are reusable
- **Could be generated** — combine with Option A's schema to generate templates

### Disadvantages

- **Still need softmod for events** — UDP emission requires persistent event handlers
- **Template engine complexity** — need to handle Lua string escaping, nil vs empty, etc.
- **Two Lua dialects** — event handlers in files, command logic in templates
- **Still hits RCON payload limits** — complex templates produce large payloads

### Verdict

A pragmatic middle ground. Good if you want to eliminate command registration without going full code-gen.

---

## Option D: Dual-Channel Architecture (RCON for queries, UDP for commands)

### Concept

Flip the current model. Instead of everything going through RCON, use:
- **RCON** for queries (read state from the game)
- **UDP outgoing** for mutations (tell the game to do things)

The Lua side would have a command dispatcher that receives structured UDP messages and executes actions.

### Flow

```
Plugin wants to place a tile:
  Plugin → ctx.Publish("udp.outgoing", "tile-fill:0,0,5,5:lab-dark-1:nauvis")
  → UDPBridge → UDP port 4001 → Factorio
  → udp.lua recv_udp → dispatches to tile_mod.fill()
  → (optionally) sends UDP confirmation back

Plugin wants to read tiles:
  Plugin → ctx.Clients().Tile().Read(area) → RCON → response
```

### Lua Dispatcher

```lua
-- udp.lua becomes a real dispatcher
local dispatch = {
    ["tile-fill"] = function(args) ... end,
    ["entity-create"] = function(args) ... end,
    ["surface-set"] = function(args) ... end,
}

factop_udp.packet_received = function(event)
    local cmd, payload = event.payload:match("^(%S+)%s(.+)$")
    if dispatch[cmd] then
        dispatch[cmd](payload)
    end
end
```

### Advantages

- **Non-blocking mutations** — UDP fire-and-forget doesn't wait for RCON lock
- **Reduced RCON contention** — only queries need the mutex-protected connection
- **Natural separation** — reads are synchronous (need response), writes are async (fire and forget)
- **Higher throughput for writes** — no serialization through a single connection
- **Tick-aligned execution** — UDP commands execute on the next `recv_udp` tick (every 20 ticks = 333ms), which is actually appropriate for game mutations

### Disadvantages

- **No confirmation for writes** — unless you build an ack protocol over UDP
- **UDP is unreliable** — packets can be dropped (though localhost UDP rarely drops)
- **Ordering not guaranteed** — multiple writes might execute out of order
- **512-byte payload limit** — can't do bulk operations in a single UDP packet
- **333ms latency floor** — `recv_udp` only runs every 20 ticks
- **Complex error handling** — write failures need a separate error reporting channel

### When This Makes Sense

For write-heavy plugins (like lab trail) that don't need confirmation. The 333ms latency is actually fine for cosmetic/decorative operations. For anything that needs confirmation (teleport a player, verify entity was created), you'd still use RCON.

### Verdict

Interesting for specific use cases but not a wholesale replacement. Best combined with another approach (e.g., Option A + Option D for write-heavy plugins).

---

## Option E: Unified Event Bus with Request/Reply Semantics

### Concept

Instead of separate RCON and UDP channels, build a unified protocol over the UDP bridge that supports both fire-and-forget events AND request/reply semantics.

### Protocol Design

Each message has a header:
```
<msg-id>:<type>:<module>:<action>:<payload>
```

- `msg-id`: unique ID for correlation (0 for fire-and-forget)
- `type`: `req` (expects reply), `evt` (fire-and-forget), `rep` (reply to a request)
- `module`: target module name
- `action`: command within the module
- `payload`: colon-separated args

### Example Exchange

```
Plugin → UDP outgoing:  "42:req:entity:find:0,0,10,10:_:_:_"
Lua receives, executes, responds:
Lua → UDP incoming:     "42:rep:entity:find:wall:5:3:123,turret:7:8:456"
```

### Lua Side

```lua
local pending_replies = {}

factop_udp.packet_received = function(event)
    local id, mtype, module, action, payload = parse_header(event.payload)
    if mtype == "req" then
        local result = dispatch[module][action](payload)
        helpers.send_udp(4000, id..":rep:"..module..":"..action..":"..result, 0)
    end
end
```

### Go Side

```go
// In the UDPBridge or a new component
func (u *UDPBridge) Request(module, action, payload string) (string, error) {
    id := atomic.AddUint64(&u.msgID, 1)
    msg := fmt.Sprintf("%d:req:%s:%s:%s", id, module, action, payload)

    replyCh := u.registerPending(id)
    u.sendUDP(msg)

    select {
    case reply := <-replyCh:
        return reply, nil
    case <-time.After(5 * time.Second):
        u.removePending(id)
        return "", ErrTimeout
    }
}
```

### Advantages

- **Bypasses RCON entirely** — no RCON connection, no mutex, no 16KB limit (UDP is 512 bytes but you can send multiple)
- **Parallel requests** — multiple in-flight requests with correlation IDs
- **Unified channel** — events and commands use the same transport
- **Lower latency for writes** — no RCON round-trip through the TCP connection
- **Scalable** — doesn't bottleneck on a single connection

### Disadvantages

- **512-byte UDP limit** — large queries (find entities in huge area) won't fit in one packet
- **Need fragmentation protocol** — for large responses, need multi-packet assembly
- **20-tick polling** — `recv_udp` runs every 20 ticks, so minimum 333ms latency for requests
- **Complex error handling** — timeouts, lost packets, out-of-order responses
- **Reinventing TCP over UDP** — if you need reliability + ordering + large payloads, you're building TCP
- **Breaking change** — completely different communication model

### Verdict

Elegant in theory, but the 20-tick polling interval and 512-byte limit make it impractical as the primary communication channel. RCON gives you immediate execution and arbitrary payload sizes. This approach only wins if Factorio's UDP implementation improves or if you're willing to accept the latency.

---

## Option F: Hybrid Code-Gen + Event Router (Recommended)

### Concept

Combine the strongest elements:
1. **Schema-driven code generation** (Option A) for command definitions
2. **Keep RCON for commands** (it works, it's synchronous, it's reliable)
3. **Typed event router in plugin SDK** (from arch_improvements.md #4)
4. **Dedicated UDP outgoing dispatcher** for fire-and-forget actions
5. **`/sc` escape hatch** for ad-hoc Lua execution

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                      Plugin Code                             │
│  ctx.OnEntityDied(handler)    ctx.Clients().Entity().Find() │
│  ctx.Fire("tile-fill", ...)   ctx.Exec(luaSnippet)          │
└────────────┬──────────────────────────┬─────────────────────┘
             │                          │
┌────────────▼──────────────────────────▼─────────────────────┐
│                     Plugin SDK                               │
│  EventRouter (parse+dispatch)  │  TypedClients (generated)  │
│  FireAndForget (udp.outgoing)  │  Exec (/sc passthrough)    │
└────────────┬──────────────────────────┬─────────────────────┘
             │                          │
┌────────────▼──────────┐  ┌───────────▼──────────────────────┐
│   NATS: udp.incoming  │  │   NATS: factop.rcon              │
│   (events from Lua)   │  │   (commands to Lua/Factorio)     │
└────────────┬──────────┘  └───────────┬──────────────────────┘
             │                          │
┌────────────▼──────────┐  ┌───────────▼──────────────────────┐
│   UDPBridge           │  │   Rcon service                    │
│   port 4000 listen    │  │   RCON TCP connection             │
└────────────┬──────────┘  └───────────┬──────────────────────┘
             │                          │
┌────────────▼──────────────────────────▼─────────────────────┐
│                  Factorio Server                              │
│  Lua softmod: event handlers → send_udp                      │
│  Lua softmod: registered commands (generated)                │
│  Lua softmod: UDP dispatcher for fire-and-forget             │
│  Factorio /sc: ad-hoc Lua execution                          │
└─────────────────────────────────────────────────────────────┘
```

### What Gets Generated (from schema)

| File | Contents |
|------|----------|
| `softmod/factop/<module>.lua` | RCON command registration + event handlers |
| `client/<module>/<module>_gen.go` | Typed client methods + wire parsers |
| `client/<module>/<module>_gen_test.go` | Parser tests with example wire strings |
| `plugin/events_gen.go` | Event structs + router registration methods |

### What Stays Hand-Written

| File | Contents |
|------|----------|
| `schema/<module>.yaml` | Command/event definitions |
| `softmod/factop/common.lua` | Shared helpers (stable, rarely changes) |
| `softmod/factop/udp.lua` | UDP dispatcher (generic, handles all fire-and-forget) |
| `client/client.go` | Core Conn, Dial, Rcon (stable) |
| `plugin/plugin.go` | SDK entry point, lifecycle (stable) |
| `plugin/context.go` | Context implementation (add event router here) |

### Plugin Author Experience

```go
type myPlugin struct{}

func (p *myPlugin) Setup(ctx plugin.Context) error {
    return nil
}

func (p *myPlugin) Run(ctx plugin.Context) error {
    // Typed event subscription — no manual parsing
    ctx.Events().OnPlayerMove(func(mv player.Move) {
        ctx.Logger().Info("player moved", "name", mv.Name, "x", mv.X, "y", mv.Y)

        // Typed client call — generated, compile-time safe
        area := tile.Area{X1: int(mv.X), Y1: int(mv.Y), X2: int(mv.X)+1, Y2: int(mv.Y)+1}
        ctx.Clients().Tile().Fill(area, "lab-dark-1", mv.SurfaceName)
    })

    ctx.Events().OnEntityDied(func(ev entity.EntityEvent) {
        if ev.Cause == "character" {
            ctx.Logger().Info("player killed entity", "entity", ev.Name, "player", ev.PlayerIndex)
        }
    })

    // Fire-and-forget for non-critical writes (optional, bypasses RCON queue)
    ctx.Fire("tile-fill", "0,0,5,5", "lab-dark-1", "nauvis")

    // Ad-hoc Lua for one-off operations not worth a schema entry
    result, _ := ctx.Exec(`rcon.print(game.tick)`)

    <-ctx.Done()
    return nil
}
```

### Generator Design

The generator (`go run ./gen` or `go generate`) would:

1. Read `schema/*.yaml` files
2. For each module, generate:
   - Lua: command handlers using a template (arg parsing, filter building, response formatting)
   - Go client: struct, constructor, methods, parser
   - Go tests: example wire strings → expected parsed structs
   - Go events: struct per event type, parser, router registration

3. Validate at generation time:
   - All referenced types are known (position, area, surface, string, int, float64, etc.)
   - Wire format field count matches struct field count
   - No duplicate command names across modules

### Event Router Implementation

```go
// In plugin/events.go (hand-written framework, generated registrations)
type EventRouter struct {
    handlers map[string][]func([]byte)
}

func (r *EventRouter) dispatch(data []byte) {
    msg := string(data)
    // Extract event name from [brackets]
    if len(msg) < 2 || msg[0] != '[' {
        return
    }
    close := strings.IndexByte(msg, ']')
    if close < 0 {
        return
    }
    event := msg[1:close]
    for _, h := range r.handlers[event] {
        h(data)
    }
}

// Generated registration methods:
func (r *EventRouter) OnPlayerMove(fn func(player.Move)) {
    r.handlers["move"] = append(r.handlers["move"], func(data []byte) {
        mv, err := player.ParseMove(string(data))
        if err == nil {
            fn(mv)
        }
    })
}

func (r *EventRouter) OnEntityDied(fn func(entity.EntityEvent)) {
    r.handlers["entity-died"] = append(r.handlers["entity-died"], func(data []byte) {
        ev, err := entity.ParseEntityEvent(string(data))
        if err == nil {
            fn(ev)
        }
    })
}
```

### Migration Path

1. **Phase 1:** Build the event router (hand-written, uses existing parsers) — immediate plugin DX improvement
2. **Phase 2:** Write the schema for existing modules (documents current state)
3. **Phase 3:** Build the generator, generate Lua + Go from schema
4. **Phase 4:** Add UDP dispatcher for fire-and-forget writes
5. **Phase 5:** Delete hand-written code replaced by generated code

Each phase is independently valuable and shippable.

---

## Comparison Matrix

| Criteria | A: Schema Gen | B: /sc Direct | C: Lua Templates | D: Dual Channel | E: Unified Bus | F: Hybrid (Rec.) |
|----------|:---:|:---:|:---:|:---:|:---:|:---:|
| Eliminates boilerplate | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| Type safety | ✅ | ❌ | ⚠️ | ⚠️ | ⚠️ | ✅ |
| Sync for queries | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ |
| Async for writes | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| Plugin DX | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ✅ |
| Minimal softmod | ❌ | ✅ | ✅ | ❌ | ❌ | ⚠️ |
| Incremental migration | ⚠️ | ✅ | ✅ | ⚠️ | ❌ | ✅ |
| Handles events | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| Implementation effort | Medium | Low | Medium | Medium | High | Medium |

---

## Recommendation

**Go with Option F (Hybrid Code-Gen + Event Router)** with this execution order:

1. **Now:** Build the event router into the plugin SDK. This is the single highest-impact change for plugin authors and requires zero changes to the softmod or service layer.

2. **Next:** Define the schema format and write schemas for the existing 7 modules. This documents the current state and validates the schema design before writing the generator.

3. **Then:** Build the generator. Start with Go client output only (Lua can come later). This lets you validate generated code against existing hand-written code.

4. **Later:** Add Lua generation. Replace hand-written Lua modules with generated ones.

5. **Eventually:** Add the UDP outgoing dispatcher for fire-and-forget mutations. This is optional and only needed for write-heavy plugins that don't need confirmation.

The `/sc` escape hatch (`ctx.Exec`) can be added at any point as a thin wrapper — it's just `ctx.Rcon("/sc " + lua)` with some string sanitization.

---

## Open Questions

- **Schema format:** YAML? CUE? Go struct tags? A custom DSL? YAML is boring but everyone knows it.
- **Generator language:** Go (using `text/template`) is the obvious choice since the project is Go.
- **Event router location:** Should it live in `plugin/` (only for plugins) or `client/` (usable by any NATS consumer)?
- **Error propagation for fire-and-forget:** Do plugins need eventual confirmation that a UDP mutation succeeded? If yes, the Lua dispatcher should send an ack/nack back.
- **Testing story:** How do you test generated Lua without a running Factorio? Maybe a Lua test harness that mocks the Factorio API?
- **Versioning:** When the schema changes, how do you handle backward compatibility with deployed plugins still expecting the old wire format?
