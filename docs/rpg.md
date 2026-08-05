# Building a Factorio RPG with factop

**This is the product plan** for story/RPG work on factop. Operator CLI and deploy details live in the [root README](../README.md).

---

## What you are building

A Factorio headless server where **what the player does changes the story**: builds, deaths, exploration, and research become inputs to a Go **story plugin**, which responds by messaging the player, spawning threats, reshaping the map, and unlocking the next problem.

Factorio stays the game. factop is the operator. Your RPG is a **plugin**.

```
Player plays Factorio
        │
        ▼
 softmod sends events (move, built, died, …)
        │
        ▼
 factop bridges them onto NATS
        │
        ▼
 RPG plugin decides what happens next
        │
        ▼
 plugin changes the world (message, spawn, teleport, …)
```

---

## What already works

Use this stack; do not redesign it.

| Piece | Role |
|-------|------|
| **factop service** | Runs Factorio, NATS, RCON, softmod deploy, plugins |
| **softmod** | Emits game events over UDP (`softmod/factop/*_gen.lua`, `player.lua`) |
| **codegen** | `go run ./gen` — one annotated Go struct → Go parser + Lua emitter + plugin `OnXxx` |
| **plugin SDK** | `plugin.Run`; subscribe with `ctx.Events().OnMove(...)`, etc. |
| **RCON** | `ctx.Rcon("/sc …")` — run Lua in the game and get a reply (works today) |

**Events you already receive:**  
`move`, `player_joined` / `left` / `died` / `respawned`, `entity_built` / `mined` / `died`, `tile_built` / `mined`, `resource_depleted`, `surface_created` / `deleted`, `game_tick`.

**What is missing for an RPG:** a real story plugin, durable quest state, richer “change the world” helpers, and a few more events (research, chat). World changes already work via `ctx.Rcon("/sc …")`; you can ship content with that and refine later.

---

## Rules of the road

1. **Story logic lives in a Go plugin**, not in Lua.
2. **New game signals = annotate a struct + `go run ./gen`**, not hand-edited dual formats.
3. **Ship a short playable arc before a generic quest framework.**
4. **Prefer one RPG plugin** until the loop feels good; split later if needed.
5. **Do not revive old softmod RCON command modules** as a prerequisite. Use `/sc` (or small helpers you add) when the plugin must change the world.

---

## Development path

### Step 0 — Sandbox (half day)

1. Run factop, deploy softmod, join the game.
2. `focmd watch` — confirm events like `udp.incoming.move` and `udp.incoming.entity_built`.
3. Deploy `pluginexample/pluginone` so a plugin process is known-good.

**Done when:** you can see player actions as NATS events.

---

### Step 1 — First story plugin (the real start)

Create `pluginexample/rpgdemo` (name flexible):

- `Setup`: load/create state file under `ctx.DataDir()` (JSON is enough).
- `Run`: register a few handlers, e.g. `OnMove`, `OnEntityBuilt`, `OnPlayerDied`.
- **State:** `stage` string + a map of flags (`built_furnace`, `first_death`, …).
- **Reactions** with `ctx.Rcon("/sc …")`:
  - `game.print(...)` or `player.print(...)` for story text
  - `surface.create_entity{...}` to spawn a problem
  - optional teleport / give item as you learn the APIs

**Done when:** build a furnace → get a message and a small biter spawn; restart plugin → flags still set.

---

### Step 2 — Playable slice: “First Night” (~1–2 weeks)

One short arc, hard-coded stages are fine:

| Stage | Trigger | Response |
|-------|---------|----------|
| intro | player joined | Welcome + goal text |
| explore | moved far enough from spawn | Flavor + foreshadowing |
| industry | first furnace/miner built | Flag set; schedule or immediate pressure |
| siege | industry flag | Spawn a small attack near the player |
| resolve | spawn cleared or player dies | Branch text; set `first_night_done` |
| tease | done | Hint at the next problem (research / expand) |

**Done when:** a 10–20 minute session feels directed without reading the code.

---

### Step 3 — Content tools (as pain appears)

Add only what the slice forces you to need:

| Need | Approach |
|------|----------|
| Player answers / debug | Event for console chat; simple commands (`rpg status`) |
| Tech gates | Event for research finished |
| Less `/sc` string soup | Small Go helpers that format common Lua snippets |
| Faster one-way actions | Optional: UDP outbound command path later (old notes called this “Fire”; name does not matter) |
| More story without recompiling | Move dialogue/beats into JSON/YAML under `DataDir` |

**Done when:** a second mini-scenario is mostly new data + a few rules, not a rewrite.

---

### Step 4 — Campaign quality

- Reset campaign on new save / explicit command  
- Cap spawns and message spam  
- Survive softmod deploy + plugin restart  
- Multiplayer: decide shared world story vs per-player flags  
- Tag a release when a full session is fun  

---

## How to add a new event (when Step 3 needs one)

1. Add or extend a struct in `client/<area>/` with `wire` tags and `// gen:event` / `// gen:lua` comments (copy an existing event).
2. Run `go run ./gen`.
3. Redeploy softmod; use `ctx.Events().On…` in the plugin.

Details of the annotation language live in existing modules — copy `entity` or `playerattr` as a template.

---

## Repo map (only what you need)

| Path | Use for RPG |
|------|-------------|
| `service/` | Operator — leave alone unless deploy/runtime breaks |
| `softmod/factop/` | Event emitters (mostly generated) |
| `client/` | Event schemas (source of truth for codegen) |
| `gen/` | Code generator |
| `plugin/` | SDK for your story plugin |
| `pluginexample/` | Put `rpgdemo` here first |
| `cmd/main.go` | CLI (`watch`, `softmod`, `plugin`, …) |
| `docs/rpg.md` | **This file — product plan** |

---

## Explicitly not the plan

- Rebuilding a large softmod RCON command suite before the story loop is fun  
- Quest DSL / visual novel layer  
- Putting campaign logic in the softmod  

---

## Next action

**`pluginexample/rpgdemo` 0.2.0** is the First Night slice (welcome → explore → industry siege → kill/death resolve).

```bash
export FACTOP_HOST=factorio
CGO_ENABLED=0 go build -ldflags "-s -w" -o /tmp/rpgdemo ./pluginexample/rpgdemo/
go run cmd/main.go plugin deploy rpgdemo 0.2.0 /tmp/rpgdemo
go run cmd/main.go plugin restart rpgdemo
# optional clean arc:
# ssh factorio 'rm -f /opt/factorio/plugins/data/rpgdemo/campaign.json'
# go run cmd/main.go plugin restart rpgdemo
```

**Play:** join → walk ~40 tiles → build furnace/drill → kill biters (or die). State: `/opt/factorio/plugins/data/rpgdemo/campaign.json`.

**Then:** chat/debug commands, research gates, more scenarios as needed.
