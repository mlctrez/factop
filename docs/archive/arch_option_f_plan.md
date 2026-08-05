# Option F Implementation Plan

## Phase 1: Event Router in Plugin SDK

### Goal
Plugin authors subscribe to typed game events without manual parsing or prefix matching.

### Changes Required

#### New file: `plugin/events.go`

The event router framework. Hand-written, stable, rarely changes.

```go
package plugin

// EventRouter dispatches parsed UDP events to typed handler functions.
type EventRouter struct {
    nc       *nats.Conn
    sub      *nats.Subscription
    handlers map[string][]func([]byte)
    logger   *slog.Logger
}

// dispatch is the raw udp.incoming subscriber callback.
// It extracts the [event-name] tag and calls registered handlers.
func (r *EventRouter) dispatch(msg *nats.Msg) { ... }
```

Key design decisions:
- Router subscribes once to `udp.incoming` on first handler registration (lazy)
- Handlers are grouped by event name string (`"move"`, `"entity-died"`, etc.)
- Parse errors are logged but don't crash the plugin
- Handlers run synchronously in subscription goroutine (simple, predictable)
- If a plugin needs async processing, the handler can send to a channel

#### New file: `plugin/events_player.go`

Typed registration methods for player events.

```go
func (r *EventRouter) OnPlayerMove(fn func(player.Move)) { ... }
func (r *EventRouter) OnPlayerJoin(fn func(PlayerJoinEvent)) { ... }
func (r *EventRouter) OnPlayerLeave(fn func(PlayerLeaveEvent)) { ... }
func (r *EventRouter) OnPlayerDeath(fn func(PlayerDeathEvent)) { ... }
func (r *EventRouter) OnPlayerRespawn(fn func(PlayerRespawnEvent)) { ... }
```

#### New file: `plugin/events_entity.go`

Typed registration methods for entity events.

```go
func (r *EventRouter) OnEntityDied(fn func(entity.EntityEvent)) { ... }
func (r *EventRouter) OnEntityBuilt(fn func(entity.EntityEvent)) { ... }
func (r *EventRouter) OnEntityMined(fn func(entity.EntityEvent)) { ... }
```

#### New file: `plugin/events_raw.go`

Escape hatch for events not yet covered by typed handlers.

```go
// OnRaw registers a handler for any event matching the given tag.
// The handler receives the full raw UDP message bytes.
func (r *EventRouter) OnRaw(eventTag string, fn func([]byte)) { ... }
```

#### Modified: `plugin/plugin.go`

Add `Events()` to the `Context` interface:

```go
type Context interface {
    Rcon(command string) (string, error)
    Subscribe(subject string, handler func([]byte)) error
    Publish(subject string, data []byte) error
    DataDir() string
    Logger() *slog.Logger
    Done() <-chan struct{}
    Clients() Clients
    Events() *EventRouter    // NEW
    Ctx() context.Context    // NEW (from arch_improvements.md #13)
}
```

#### Modified: `plugin/context.go`

Add event router initialization and `context.Context` support:

```go
type contextImpl struct {
    // ... existing fields ...
    eventsOnce sync.Once
    events     *EventRouter
    ctx        context.Context
    cancel     context.CancelFunc
}

func (c *contextImpl) Events() *EventRouter {
    c.eventsOnce.Do(func() {
        c.events = newEventRouter(c.nc, c.logger)
    })
    return c.events
}

func (c *contextImpl) Ctx() context.Context {
    return c.ctx
}
```

#### New event structs: `plugin/events_types.go`

Simple events that don't warrant their own client package:

```go
type PlayerJoinEvent struct {
    Name  string
    Index int
}

type PlayerLeaveEvent struct {
    Name  string
    Index int
}

type PlayerDeathEvent struct {
    Name  string
    Index int
    Cause string
}

type PlayerRespawnEvent struct {
    Name  string
    Index int
}
```

#### New parsers for simple events: `plugin/events_parse.go`

Parse the `[join]`, `[leave]`, `[death]`, `[respawn]` UDP formats:

```go
func parseJoinEvent(msg string) (PlayerJoinEvent, error) { ... }
func parseLeaveEvent(msg string) (PlayerLeaveEvent, error) { ... }
func parseDeathEvent(msg string) (PlayerDeathEvent, error) { ... }
func parseRespawnEvent(msg string) (PlayerRespawnEvent, error) { ... }
```

#### Modified: `pluginexample/pluginone/main.go`

Refactor to use the event router (demonstrates the improvement):

```go
func (l *labTrail) Run(ctx plugin.Context) error {
    tileClient := ctx.Clients().Tile()

    ctx.Events().OnPlayerMove(func(mv player.Move) {
        tx := int(math.Floor(mv.X))
        ty := int(math.Floor(mv.Y))
        name := labTiles[rand.IntN(len(labTiles))]
        area := tile.Area{X1: tx, Y1: ty, X2: tx + 1, Y2: ty + 1}
        tileClient.Fill(area, name, mv.SurfaceName)
    })

    <-ctx.Done()
    return nil
}
```

### Testing

- Unit tests for each parser function (table-driven, example messages → expected structs)
- Unit test for router dispatch (register handler, feed raw bytes, verify callback fires)
- Integration: update pluginone, deploy, verify it still works

### Files Summary

| Action | File |
|--------|------|
| Create | `plugin/events.go` |
| Create | `plugin/events_player.go` |
| Create | `plugin/events_entity.go` |
| Create | `plugin/events_raw.go` |
| Create | `plugin/events_types.go` |
| Create | `plugin/events_parse.go` |
| Create | `plugin/events_test.go` |
| Modify | `plugin/plugin.go` (add Events/Ctx to interface) |
| Modify | `plugin/context.go` (implement Events/Ctx) |
| Modify | `pluginexample/pluginone/main.go` (use event router) |

---

## Phase 2: Schema Definition

### Goal
Document the current command/event contracts in a machine-readable format that will drive code generation in Phase 3.

### Schema Location

```
schema/
  entity.yaml
  game.yaml
  player.yaml
  playerattr.yaml
  resource.yaml
  surface.yaml
  tile.yaml
```

### Schema Format (YAML)

```yaml
# schema/entity.yaml
module: entity
description: Entity manipulation — CRUD operations for entities on a surface

types:
  Entity:
    fields:
      - name: Name
        wire_name: name
        type: string
      - name: X
        wire_name: x
        type: float64
      - name: Y
        wire_name: y
        type: float64
      - name: UnitNumber
        wire_name: unit_number
        type: uint64
    wire_separator: ":"
    list_separator: ","

  EntityEvent:
    fields:
      - name: Event
        type: string
        source: tag  # extracted from [brackets], not wire payload
      - name: Name
        type: string
      - name: X
        type: float64
      - name: Y
        type: float64
      - name: UnitNumber
        type: uint64
      - name: SurfaceName
        type: string
      - name: SurfaceIndex
        type: int
      - name: PlayerIndex
        type: int
      - name: Cause
        type: string
        optional: true  # only present in entity-died

  Position:
    inline: true  # not a standalone struct, used for arg formatting
    format: "{X},{Y}"
    fields:
      - name: X
        type: float64
      - name: Y
        type: float64

  Area:
    inline: true
    format: "{X1},{Y1},{X2},{Y2}"
    fields:
      - name: X1
        type: float64
      - name: X2
        type: float64
      - name: Y1
        type: float64
      - name: Y2
        type: float64

commands:
  create:
    description: Place a single entity at the given position
    args:
      - name: pos
        type: Position
        required: true
      - name: name
        type: string
        required: true
      - name: force
        type: string
        default: "player"
      - name: direction
        type: direction
        skip: "_"
      - name: surface
        type: surface
    response:
      type: string

  bulk:
    description: Create multiple entities of the same type in batched RCON calls
    custom: true  # generator skips this, hand-written due to batching logic
    args:
      - name: positions
        type: "[]Position"
      - name: name
        type: string
      - name: force
        type: string
      - name: surface
        type: surface
    response:
      type: int  # total created count

  find:
    description: Find entities matching filters in an area
    args:
      - name: area
        type: Area
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
      type: "[]Entity"

  count:
    description: Count entities matching filters in an area
    args:
      - name: area
        type: Area
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
      - name: surface
        type: surface
    response:
      type: int

  destroy:
    description: Remove entities matching filters in an area
    args:
      - name: area
        type: Area
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
      type: string

events:
  entity-died:
    type: EntityEvent
    tag: "entity-died"
    has_cause: true

  entity-built:
    type: EntityEvent
    tag: "entity-built"

  entity-mined:
    type: EntityEvent
    tag: "entity-mined"

test_cases:
  find_response:
    wire: "stone-wall:5.0:3.0:123,laser-turret:7.5:8.0:456"
    expected:
      - { Name: "stone-wall", X: 5.0, Y: 3.0, UnitNumber: 123 }
      - { Name: "laser-turret", X: 7.5, Y: 8.0, UnitNumber: 456 }

  entity_died_event:
    wire: "[entity-died] stone-wall:5.0:3.0:123:nauvis:1:0:small-biter"
    expected:
      Event: "entity-died"
      Name: "stone-wall"
      X: 5.0
      Y: 3.0
      UnitNumber: 123
      SurfaceName: "nauvis"
      SurfaceIndex: 1
      PlayerIndex: 0
      Cause: "small-biter"
```

### Design Decisions

- `custom: true` marks commands that have logic too complex for generation (batching, retries)
- `skip: "_"` indicates the placeholder used for optional positional args
- `type: surface` is a special type that means "optional string, appended last, resolves to a surface name"
- `type: direction` means "optional Factorio direction enum string"
- `test_cases` section drives generated test files
- `inline: true` types are not standalone Go structs — they're used for argument formatting only

### Files Summary

| Action | File |
|--------|------|
| Create | `schema/entity.yaml` |
| Create | `schema/game.yaml` |
| Create | `schema/player.yaml` |
| Create | `schema/playerattr.yaml` |
| Create | `schema/resource.yaml` |
| Create | `schema/surface.yaml` |
| Create | `schema/tile.yaml` |

### Validation

After writing all schemas, manually verify:
- Every existing Go client method is representable in the schema
- Every existing Lua command is covered
- Wire format field counts match type definitions
- Test cases cover at least one happy path per command response type

---

## Phase 3: Go Client Code Generator

### Goal
Generate Go client packages from schema files, replacing hand-written code.

### Generator Location

```
gen/
  main.go          # entry point, reads schemas, orchestrates generation
  schema.go        # YAML schema parsing into Go types
  template_go.go   # Go code generation templates
  template_test.go # Go test generation templates
  types.go         # shared type mapping (schema types → Go types)
```

### Invocation

```bash
go run ./gen
# or via go:generate directive in a root file:
//go:generate go run ./gen
```

### What It Generates

For each `schema/<module>.yaml`:

**`client/<module>/<module>_gen.go`:**
- Package doc comment from schema description
- Type structs from schema types (unless `inline: true`)
- `Client` struct with `*client.Conn`
- `New(conn)` constructor
- One method per command (unless `custom: true`)
- `Parse` functions for list/struct response types
- `String()` methods for inline types (Area, Position)

**`client/<module>/<module>_gen_test.go`:**
- Table-driven tests from `test_cases` section
- One test function per response parser

**`plugin/events_<module>_gen.go`:**
- Event struct (if not already defined in a client package)
- Parser function for the event wire format
- `OnXxx` registration method on `EventRouter`

### Template Strategy

Use `text/template` with helper functions:

```go
var clientTemplate = template.Must(template.New("client").Funcs(template.FuncMap{
    "goType":     goType,     // schema type → Go type string
    "methodName": methodName, // "find" → "Find"
    "formatCmd":  formatCmd,  // builds the fmt.Sprintf for RCON command
}).Parse(clientTmpl))
```

### Handling `custom: true` Commands

Commands marked `custom: true` are skipped by the generator. The hand-written implementation stays in a non-`_gen.go` file (e.g., `client/entity/bulk.go`). The generated file includes a comment noting which commands are hand-written:

```go
// Code generated by go run ./gen -- DO NOT EDIT.
// Hand-written commands (custom: true): Bulk
```

### Migration Strategy

1. Generate code alongside existing hand-written code (different filenames)
2. Compare generated output against existing implementations
3. Run existing tests against generated code
4. Once validated, delete hand-written files and rename generated files
5. Update `go:generate` directive

### Files Summary

| Action | File |
|--------|------|
| Create | `gen/main.go` |
| Create | `gen/schema.go` |
| Create | `gen/template_go.go` |
| Create | `gen/template_test.go` |
| Create | `gen/types.go` |
| Generate | `client/entity/entity_gen.go` (replaces `entity.go`) |
| Generate | `client/entity/entity_gen_test.go` |
| Generate | `client/game/game_gen.go` (replaces `game.go`) |
| Generate | `client/game/game_gen_test.go` |
| Generate | `client/player/player_gen.go` (replaces parts of `player.go`) |
| Generate | `client/resource/resource_gen.go` (replaces `resource.go`) |
| Generate | `client/surface/surface_gen.go` (replaces `surface.go`) |
| Generate | `client/surface/parse_gen.go` (replaces `parse.go`) |
| Generate | `client/tile/tile_gen.go` (replaces `tile.go`) |
| Generate | `client/playerattr/playerattr_gen.go` (replaces `playerattr.go`) |
| Generate | `plugin/events_entity_gen.go` |
| Generate | `plugin/events_player_gen.go` |
| Delete | Hand-written client files after validation |

---

## Phase 4: Lua Code Generator

### Goal
Generate Lua softmod command modules from the same schema files.

### Generator Addition

Add Lua template to `gen/`:

```
gen/
  template_lua.go  # Lua code generation templates
```

### What It Generates

For each `schema/<module>.yaml`:

**`softmod/factop/<module>_gen.lua`:**
- Module table declaration
- `require("factop.common")`
- `register_commands()` function with all non-custom commands
- Event handler functions (for modules that have events)
- Module table return with `.events`, `.on_init`, `.on_load`

### Lua Template Pattern

Generated commands follow the existing pattern exactly:

```lua
commands.add_command("entity-find", "Find entities. Usage: /entity-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]", function(cmd)
    if not c.rcon_only(cmd) then return end
    local args = c.parse_args(cmd)
    if #args < 1 then c.reply("Usage: /entity-find x1,y1,x2,y2 [name] [type] [force] [limit] [surface]") return end
    local area = c.parse_area(args[1])
    if not area then c.reply("Invalid area format. Use: x1,y1,x2,y2") return end
    -- ... generated filter building and response formatting ...
end)
```

### Handling Custom Commands in Lua

For `custom: true` commands, the generated file includes a comment:
```lua
-- entity-bulk: custom implementation in entity_custom.lua
```

And a separate `<module>_custom.lua` file stays hand-written. The generator's `register_commands()` calls into the custom file:

```lua
local custom = require("factop.entity_custom")

local function register_commands()
    -- Generated commands here...

    -- Custom commands
    custom.register_commands()
end
```

### Migration Strategy

1. Generate Lua alongside existing hand-written Lua
2. Diff generated output against existing (should be nearly identical)
3. Test with a softmod deploy
4. Replace hand-written files
5. Update `softmod.go` embed pattern if needed (should be transparent since it walks the directory)

### Files Summary

| Action | File |
|--------|------|
| Create | `gen/template_lua.go` |
| Generate | `softmod/factop/entity_gen.lua` (replaces `entity.lua`) |
| Generate | `softmod/factop/game_gen.lua` (replaces `game.lua`) |
| Generate | `softmod/factop/player_gen.lua` (replaces `player.lua`) |
| Generate | `softmod/factop/playerattr_gen.lua` (replaces `playerattr.lua`) |
| Generate | `softmod/factop/resource_gen.lua` (replaces `resource.lua`) |
| Generate | `softmod/factop/surface_gen.lua` (replaces `surface.lua`) |
| Generate | `softmod/factop/tile_gen.lua` (replaces `tile.lua`) |
| Keep | `softmod/factop/common.lua` (hand-written, stable) |
| Keep | `softmod/factop/goal.lua` (hand-written, not schema-driven) |
| Rewrite | `softmod/factop/udp.lua` (becomes the dispatcher in Phase 5) |

---

## Phase 5: UDP Outgoing Dispatcher

### Goal
Enable plugins to send fire-and-forget mutations to the game without blocking on RCON.

### Lua Side Changes

#### Rewrite: `softmod/factop/udp.lua`

Transform from a stub into a real command dispatcher:

```lua
local udp_mod = {}
local c = require("factop.common")

-- Dispatch table: command → handler function
local dispatch = {}

-- Register a handler for a UDP command
function udp_mod.register(cmd, handler)
    dispatch[cmd] = handler
end

-- Process incoming UDP packet
udp_mod.packet_received = function(event)
    local payload = event.payload
    if not payload or payload == "" then return end

    -- Format: command:arg1:arg2:arg3...
    local cmd = payload:match("^([^:]+)")
    if not cmd then return end

    local handler = dispatch[cmd]
    if handler then
        local args_str = payload:sub(#cmd + 2)  -- skip "cmd:"
        local ok, err = pcall(handler, args_str)
        if not ok then
            print("UDP dispatch error for " .. cmd .. ": " .. tostring(err))
        end
    end
end

udp_mod.periodic = function()
    helpers.recv_udp(0)
end

udp_mod.events = {
    [defines.events.on_udp_packet_received] = udp_mod.packet_received,
}

udp_mod.on_nth_tick = {
    [20] = udp_mod.periodic
}

-- Register built-in fire-and-forget commands
udp_mod.on_init = function()
    -- tile-fill: x1,y1,x2,y2:tile_name:surface_name
    udp_mod.register("tile-fill", function(args)
        local parts = {}
        for p in args:gmatch("[^:]+") do parts[#parts + 1] = p end
        if #parts < 2 then return end
        local lt, rb = c.parse_tile_area(parts[1])
        if not lt then return end
        local surface = c.get_surface(parts[3])
        if not surface then return end
        -- reuse tile_mod logic or inline
        local tiles = {}
        for x = lt.x, rb.x - 1 do
            for y = lt.y, rb.y - 1 do
                tiles[#tiles + 1] = { name = parts[2], position = { x, y } }
            end
        end
        if #tiles > 0 then surface.set_tiles(tiles, true, false) end
    end)
end

udp_mod.on_load = udp_mod.on_init

return udp_mod
```

### Go Side Changes

#### New method on plugin Context: `Fire`

```go
// Fire sends a fire-and-forget command via UDP outgoing.
// The command executes on the next Lua recv_udp tick (~333ms).
// No confirmation or error is returned.
func (c *contextImpl) Fire(command string, args ...string) {
    payload := command
    if len(args) > 0 {
        payload += ":" + strings.Join(args, ":")
    }
    _ = c.nc.Publish("udp.outgoing", []byte(payload))
}
```

#### Update Context interface

```go
type Context interface {
    // ... existing methods ...
    Fire(command string, args ...string)
}
```

#### Fix: `service/udpbridge.go` persistent outgoing connection

Replace per-message `DialUDP` with a persistent connection:

```go
type UDPBridge struct {
    // ... existing fields ...
    outgoingConn *net.UDPConn  // persistent, created at startup
}

func (u *UDPBridge) Startup() error {
    // ... existing incoming setup ...

    // Create persistent outgoing connection
    u.outgoingConn, err = net.DialUDP("udp4", nil, u.outgoingAddr)
    if err != nil {
        return fmt.Errorf("failed to create outgoing UDP connection: %w", err)
    }

    // ... rest of startup ...
}

func (u *UDPBridge) handleOutgoing(msg *nats.Msg) {
    if _, err := u.outgoingConn.Write(msg.Data); err != nil {
        u.Error("failed to send UDP packet", "error", err)
    }
}
```

### Plugin Usage

```go
// Fire-and-forget tile placement (no RCON contention)
ctx.Fire("tile-fill", "0,0,5,5", "lab-dark-1", "nauvis")

// For operations that need confirmation, still use RCON:
result, err := ctx.Clients().Tile().Fill(area, "lab-dark-1", "nauvis")
```

### Files Summary

| Action | File |
|--------|------|
| Rewrite | `softmod/factop/udp.lua` |
| Modify | `plugin/plugin.go` (add Fire to interface) |
| Modify | `plugin/context.go` (implement Fire) |
| Modify | `service/udpbridge.go` (persistent outgoing conn) |

---

## Ancillary Improvements (Can Be Done Anytime)

### Fix `goal.lua` global naming

```lua
-- Before:
factop_goal = {}
factop_goal_setup = function() ...

-- After:
local goal_mod = {}
local function setup() ...
```

### Move `client/test/test.go`

```
client/test/test.go → cmd/mazetest/main.go
```

or

```
client/test/test.go → example/maze/main.go
```

### Add `context.Context` to plugin SDK (included in Phase 1)

Already covered in the Phase 1 context.go changes.

### Improve plugin health check timing

In `service/plugin.go`, add an initial short-delay check:

```go
func (pm *PluginManager) healthCheck(name string) {
    // Initial check after 3 seconds
    select {
    case <-pm.Context.Done():
        return
    case <-time.After(3 * time.Second):
    }
    pm.doHealthCheck(name) // factor out the check logic

    // Then regular 30-second interval
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    // ...
}
```

### Structured RCON errors in `client/client.go`

```go
var (
    ErrRconDisconnected = errors.New("rcon not connected")
    ErrNotFound         = errors.New("not found")
    ErrInvalidArgs      = errors.New("invalid arguments")
)

func (c *Conn) Rcon(command string) (string, error) {
    msg, err := c.nc.Request(RconSubject, []byte(command), c.timeout)
    // ... existing error handling ...
    if msg.Header != nil {
        if errVal := msg.Header.Get("error"); errVal != "" {
            return "", classifyError(errVal)
        }
    }
    return string(msg.Data), nil
}

func classifyError(msg string) error {
    switch {
    case msg == "rcon not connected":
        return ErrRconDisconnected
    case strings.Contains(msg, "not found"):
        return fmt.Errorf("%w: %s", ErrNotFound, msg)
    default:
        return errors.New(msg)
    }
}
```

---

## Execution Order Summary

```
Phase 1: Event Router          (immediate value, no breaking changes)
   ↓
Phase 2: Schema Definition     (documentation, validates design)
   ↓
Phase 3: Go Code Generator     (eliminates boilerplate, adds type safety)
   ↓
Phase 4: Lua Code Generator    (completes the generation story)
   ↓
Phase 5: UDP Dispatcher        (unlocks async mutations)
```

Each phase is independently deployable. Phase 1 can ship tomorrow. Phase 2 is just YAML files. Phase 3 is the biggest effort but has the longest-term payoff. Phase 4 and 5 are optional enhancements that further reduce friction.

---

## Estimated Effort

| Phase | New Files | Modified Files | Estimated Lines | Time |
|-------|-----------|----------------|-----------------|------|
| 1 | 7 | 3 | ~400 | 1-2 days |
| 2 | 7 | 0 | ~600 (YAML) | 1 day |
| 3 | 5 + generated | many (replacements) | ~800 (generator) | 3-4 days |
| 4 | 1 + generated | 0 | ~200 (templates) | 1-2 days |
| 5 | 0 | 4 | ~150 | 1 day |
| Ancillary | 0 | 5-6 | ~100 | half day |

Total: ~8-10 days of focused work for the complete transformation.
