---
inclusion: auto
description: Naming conventions, module structure, RCON command patterns, and workflow standards for the factop codebase.
---

# Factop Naming and Code Conventions

These conventions apply to all code in this repository. Follow them when
creating new modules, commands, packages, or files.

## Singular Naming

All names use singular nouns. This applies to:

- Lua module files: `softmod/factop/entity.lua` not `entities.lua`
- Lua module variables: `local entity_mod = {}` not `entities_mod`
- Go client packages: `client/entity/` not `client/entities/`
- Go source files: `entity.go` not `entities.go`
- RCON command prefixes: `/entity-create` not `/entities-create`

Exception: `game-players` stays plural because it returns a list and
`game-player` would imply a single-player operation.

## Softmod Module Structure

Each Lua module in `softmod/factop/` that registers RCON commands must:

1. Require `factop.common` for shared helpers (`rcon_only`, `reply`, `get_surface`, `parse_args`, `parse_area`, `parse_position`, `parse_tile_area`)
2. Guard all commands with `c.rcon_only(cmd)` — commands are not accessible to in-game players
3. Register commands in a `register_commands` function called from both `on_init` and `on_load`
4. Return a table with `on_init`, `on_load`, and optionally `events` keys

`common.lua` is excluded from `add_lib` registration by `softmod.go` and
must not contain event handlers.

## Client Package Structure

Each `softmod/factop/<name>.lua` gets a corresponding `client/<name>/` Go package:

- `New(conn *client.Conn) *Client` constructor
- One method per RCON command
- A `Parse` function for any structured response wire format
- Tests for the parser

## RCON Command Naming

Commands follow the pattern `/<module>-<action>`:

- Module is the singular Lua file name: `entity`, `tile`, `surface`, `resource`, `player`, `game`
- Action is a verb: `create`, `find`, `count`, `destroy`, `fill`, `read`, `remove`, `replace`, `set`, `respawn`, `teleport`, `kick`

## Wire Formats

Responses use compact colon-separated formats parseable with `strings.Split`.
Do not use JSON — Lua has no built-in JSON encoder.

- Tile: `name:x:y`
- Entity: `name:x:y:unit_number`
- Resource: `name:x:y:amount`
- Player: `name:index:x:y:has_character`
- Surface list: `name:index`
- Surface info: `key:value,key:value,...`
- Scalar values: plain text (`123.45`, `Placed 100 tiles`)

## Factorio API Reference

Use `apidoc/factorio-api.md` as the API reference when writing new softmod
Lua code. Regenerate it with `go run ./apidoc` when needed.

## Prototype Validation

Use `client/prototype/` for compile-time validation of entity and tile names.
Regenerate with `go run ./protodump --factorio-dir /opt/factorio/<version>`
after Factorio updates.

## Commit and Push

Use the `modular` command to commit, version bump, and push in one step:

```bash
modular bump "commit message here"
```

Commit messages must always be multi-line: a short summary line followed by
a blank line and then one or more detailed paragraphs describing each change
in the commit. Example:

```
modular bump "Standardize player move UDP format

Update player.lua to emit colon-separated move events that include
surface name and index, matching the project wire format conventions.

Move parseMove into client/player as an exported ParseMove function
returning a typed Move struct for reuse across plugins.

Update pluginone to use player.ParseMove from the client package."
```
