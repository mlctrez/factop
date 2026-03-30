# apidoc

Fetches the Factorio runtime Lua API documentation from
`https://lua-api.factorio.com/latest/runtime-api.json` and generates a filtered
Markdown file suitable for use as a Kiro steering file or general LLM context.

The API JSON is cached locally to avoid repeated downloads.

## Usage

```bash
go run ./apidoc [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-list` | `false` | List all available class, event, define, concept, global object, and global function names, then exit. Useful for discovering what to filter on. |
| `-classes` | `""` | Comma-separated class names to include (e.g. `LuaPlayer,LuaEntity`). |
| `-events` | `""` | Comma-separated event names to include (e.g. `on_player_created`). |
| `-defines` | `""` | Comma-separated top-level define names to include (e.g. `events`). |
| `-concepts` | `""` | Comma-separated concept names to include (e.g. `MapPosition`). |
| `-globals` | `false` | Include global objects (`game`, `script`, `helpers`, etc.) and global functions. |
| `-output` | `../.kiro/steering/factorio-api.md` | Output path for the generated Markdown file. |
| `-cache-dir` | `.` | Directory to store the cached API JSON. |
| `-max-age` | `24h` | Maximum age of cached data before re-fetching. |
| `-no-cache` | `false` | Bypass cache and always fetch fresh data. |

## Examples

List everything available in the API:

```bash
go run ./apidoc --cache-dir apidoc --list
```

Generate a steering file for softmod development:

```bash
go run ./apidoc --cache-dir apidoc \
  --globals \
  --classes LuaPlayer,LuaGameScript,LuaHelpers,LuaEntity,LuaSurface,LuaForce,LuaGuiElement,LuaBootstrap \
  --events on_player_created,on_player_joined_game,on_player_left_game,on_player_died,on_player_respawned,on_player_changed_position,on_udp_packet_received \
  --defines events
```

Generate directly into the Kiro steering directory:

```bash
go run ./apidoc --cache-dir apidoc \
  --globals \
  --classes LuaPlayer,LuaGameScript \
  --output ../.kiro/steering/factorio-api.md
```

## Output Format

The generated Markdown includes a front-matter block with `inclusion: manual` so
it is only pulled into Kiro context when explicitly referenced via `#` in chat.
This keeps the default context lean while making the full API reference available
on demand.

Sections included depend on the flags provided:

- **Global Objects** — entry points like `game`, `script`, `helpers`
- **Global Functions** — top-level functions like `log()`, `table_size()`
- **Classes** — attributes (with R/W indicators) and methods (with parameter details)
- **Events** — event data fields and descriptions
- **Defines** — enum-like values (e.g. `defines.events.*`)
- **Concepts** — type definitions used across the API
