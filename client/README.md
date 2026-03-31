# client

Typed Go client for interacting with a factop server over NATS.

## Architecture

```
client/
├── client.go              # Shared Conn type (NATS + RCON-over-NATS)
├── entity/entities.go     # Client for softmod/factop/entity.lua commands
├── player/players.go      # Client for softmod/factop/player.lua commands
├── prototype/prototypes_gen.go # Generated prototype name validation
├── resource/resources.go  # Client for softmod/factop/resource.lua commands
├── surface/surface.go     # Client for softmod/factop/surface.lua commands
├── tile/tiles.go          # Client for softmod/factop/tile.lua commands
└── README.md
```

## Naming Convention

All module names use singular nouns. Both the Lua file and the Go client
package must use the same singular name:

| Lua module | Go package | Client type |
|---|---|---|
| `softmod/factop/tile.lua` | `client/tile` | `tile.Client` |
| `softmod/factop/entity.lua` | `client/entity` | `entity.Client` |
| `softmod/factop/surface.lua` | `client/surface` | `surface.Client` |
| `softmod/factop/resource.lua` | `client/resource` | `resource.Client` |
| `softmod/factop/player.lua` | `client/player` | `player.Client` |

Do not use plural forms for module or package names (e.g. use `entity` not
`entities`, `tile` not `tiles`). RCON command names may still use plurals
where it reads naturally (e.g. `/players-list`) — the naming rule applies
to file and package names only.

When creating a new softmod command module:

1. Use `apidoc/factorio-api.md` as the API reference for Factorio Lua calls.
   Generate or update it with `go run ./apidoc` if needed.
2. Create `softmod/factop/<name>.lua` following the patterns in `softmod/README.md`
   (RCON-only guard, `on_init`/`on_load` registration, compact wire format).
3. Create `client/<name>/` with a `Client` struct, a `New(conn)` constructor,
   typed methods for each command, and a `Parse` function for any structured
   response format.
4. The Lua command names and Go method names should mirror each other:
   `/tiles-fill` → `tiles.Client.Fill()`, `/entities-create` → `entities.Client.Create()`.

## Connection

All sub-packages share a `*client.Conn` which wraps the NATS connection.
The server address is configured once at dial time:

```go
conn, err := client.Dial("nats://factorio:4222")
defer conn.Close()

tc := tile.New(conn)
ec := entity.New(conn)
```

Options:
- `client.WithTimeout(30 * time.Second)` — RCON request timeout (default 10s)

## Wire Formats

Commands use compact text formats to minimize RCON payload size. Each
`client/<name>/` package includes a `Parse` function that converts the
wire format into typed Go structs.

| Module | Wire format | Example |
|---|---|---|
| tiles-read | `name:x:y,...` | `concrete:0:0,grass-1:1:0` |
| entities-find | `name:x:y:unit_number,...` | `iron-chest:5.5:10.5:42` |
| surface-list | `name:index,...` | `nauvis:1,vulcanus:2` |
| surface-info | `key:value,...` | `name:nauvis,always_day:false,...` |
| resources-count | `name:count,...` | `iron-ore:12345,copper-ore:6789` |
| resources-find | `name:x:y:amount,...` | `iron-ore:10.5:20.5:1500` |
| pollution-get | `float` | `123.45` |
| pollution-total | `float` | `5678.90` |

## Filter Placeholders

For commands with optional positional arguments, use `_` to skip a filter
field. The Lua side treats `_` as "no filter". The Go client handles this
automatically through the typed options structs.

## Design Decisions

- **NATS, not direct RCON**: The client talks to NATS, which routes to the
  factop service's RCON handler. This keeps the RCON password internal to
  the server and allows the same NATS bus for all factop communication.
- **Compact wire formats over JSON**: Lua has no built-in JSON encoder.
  Custom compact formats are trivially parseable on both sides and keep
  RCON payloads small.
- **One package per Lua module**: Keeps Go imports focused. You only pull
  in the client code for the commands you actually use.
- **Parse functions are exported**: Useful for testing or processing
  raw RCON output from other sources.
