# Handoff plan: Merlin companion (summon + follow)

**Status:** Ready for implementation by another agent/LLM.  
**Repo:** `github.com/mlctrez/factop` (branch `master` as of handoff).  
**Related live stack:** Factorio host `factorio` / `10.0.0.181`, factop service, plugins under `/opt/factorio/plugins/`.

---

## 0. Update (v2): follow moved into the softmod

v1 (plugin `OnMove` + RCON, sections 2.3/3/6 below) made Merlin lurch every
~2 tiles. v2 implements **approach A** from 2.2:

- `softmod/factop/companion.lua` registers `on_nth_tick[3]` (20x/sec). For each
  companion it compares Merlin to the owner's `physical_position` and sets
  `walking_state` with an 8-way direction, so he actually walks (animated).
- Hysteresis: a walking Merlin stops within **2** tiles; an idle Merlin only
  starts again when the owner is more than **4** tiles away.
- Teleport fallback (via `find_non_colliding_position`): owner more than **20**
  tiles away on the same surface, or Merlin stuck (barely moving) for **60**
  ticks while trying to walk.
- Owner offline, dead/respawning (physical controller not `character`), or on
  another surface: Merlin stops and waits (characters can't teleport across
  surfaces); following resumes when the owner is back.
- Entity gone without `on_entity_died`, or owner removed → cleanup +
  `companion_dismissed`. Death still emits `companion_died`.
- The merlin plugin no longer subscribes to `OnMove` or sends RCON; it only
  tracks the player → unit_number mapping and logs lifecycle events.
- Pure follow math (`direction_for`, `follow_decision`) is unit-tested in
  `softmod/testdata/companion_test.lua` (run by `go test ./softmod/`).

The rest of this document is the original v1 handoff, kept for history.

---

## 1. Product goal

Add a **new factop plugin** so a player can **summon an NPC named Merlin** who **follows them** around the map.

- Summon is player-initiated (v1: softmod chat command `/merlin`).
- Merlin is a free-standing Factorio **`character`** entity (not the player’s controlled character).
- Follow behavior must feel usable in multiplayer.
- Prefer existing factop patterns: softmod events over UDP → NATS → plugin `EventRouter` / RCON for mutations.

---

## 2. Architecture re-evaluation (softmod vs Go)

### 2.1 Is it true that follow via Go adds no extra messaging?

**Mostly true for UDP/NATS move traffic. Incomplete if you only consider that.**

| Channel | Already exists? | If follow is driven by plugin `OnMove` |
|---------|-----------------|----------------------------------------|
| **UDP/NATS `[move]`** | **Yes** — `softmod/factop/player.lua` emits when player moves ≥ ~2 tiles | **No extra move messages.** Plugin reuses the same stream as `pluginone` (lab tiles) and `rpgdemo`. |
| **RCON** | Used by plugins for `/sc …` | **Extra RCON per move event** to teleport Merlin (same class of load as pluginone painting a tile each move). |
| **New companion events** | No | Summon/dismiss/died are **few** messages (not per-tick). |

So the earlier plan’s claim that “plugin OnMove follow adds messaging” was **overstated for UDP**. What it adds is **RCON work**, not a second move pipeline.

`pluginone` already proves the pattern:

```go
ctx.Events().OnMove(func(mv player.Move) {
    ctx.Rcon(fmt.Sprintf("/sc game.surfaces['%s'].set_tiles(...)", ...))
})
```

Merlin follow can mirror that: on move → teleport companion near player.

### 2.2 Softmod nth_tick follow vs Go OnMove follow

| Approach | Pros | Cons |
|----------|------|------|
| **A. Softmod `on_nth_tick` follow** | Smoother (every N ticks, not every 2 tiles); no RCON; works even if plugin is down | More Lua; “brain” split; companion logic in two places |
| **B. Plugin `OnMove` + RCON teleport (recommended v1)** | Matches pluginone/rpgdemo; **reuses existing move UDP**; companion policy stays in Go; less softmod | Follow snaps every ~2 tiles; RCON latency/load; Merlin stops if plugin stops |
| **C. Softmod create only + plugin follow** | Good split: softmod spawns/emits once; Go owns follow policy | Still RCON on each move |

### 2.3 Recommendation for v1 (revised)

**Prefer Go-centric follow (approach B/C), not softmod nth_tick**, because:

1. Move events **already** flow for every player; follow does **not** require new continuous UDP.
2. The project direction is **plugins as brains** (`docs/rpg.md`).
3. Summon is rare; create via softmod command **or** plugin RCON both work — softmod command is better for player UX (`/merlin` in-game without plugin round-trip for intent).

**v1 recommended split:**

| Piece | Where |
|-------|--------|
| `/merlin` and `/merlin-dismiss` (player commands) | **Softmod** `companion.lua` |
| Create/destroy character entity | **Softmod** (on command) — or plugin RCON if you want zero softmod entity code; softmod create is cleaner |
| Emit `companion_summoned` / `dismissed` / `died` | **Softmod** (hand `helpers.send_udp`) |
| Parse events + `OnCompanion*` | **Codegen** from `client/companion/` |
| Remember active Merlin per player; **follow on `OnMove`** | **Plugin** `pluginexample/merlin` via RCON teleport |
| Flavor text | **Plugin** |

Optional later upgrade: move follow into softmod nth_tick if snap/RCON becomes annoying — plugin keeps summon policy/dialogue only.

### 2.4 What still must be softmod (or new events)

Even with Go follow, you still need **something** for:

1. **Player summon intent** without a chat event yet → softmod `commands.add_command("merlin", ...)`.
2. **Entity create** next to the player (softmod or plugin RCON — either OK).
3. **Lifecycle events** so the plugin knows `unit_number` → softmod UDP after create/destroy, plus death detection.

You do **not** need softmod follow for v1 if plugin OnMove is accepted.

---

## 3. Target design (v1)

```
Player runs /merlin
    → softmod creates character near player
    → softmod stores owner (storage, not on_load)
    → softmod UDP [companion_summoned] merlin:playerIndex:...:unitNumber:x:y:surface...
    → plugin OnCompanionSummoned: record player→unitNumber; print flavor
Player walks (existing [move] events)
    → plugin OnMove: if player has Merlin, RCON teleport entity unit_number near mv.X/Y
Player /merlin-dismiss or companion dies
    → softmod cleanup + UDP dismissed/died
    → plugin drops mapping
```

### Factorio entity notes

- `surface.create_entity{ name = "character", position = ..., force = player.force }`.
- Do **not** call `associate_character` on the human player.
- Track logical name `"merlin"` in events; prototype is still `"character"`.
- One companion per player in v1.
- **Never write `storage` in `on_load`** (see `softmod/factop/player.lua` fix — multiplayer CRC crash).

### Follow RCON sketch (plugin)

```lua
-- via /sc from plugin
local e = game.get_entity_by_unit_number(UNIT)
if e and e.valid then
  e.teleport({x = PX + 2, y = PY}, game.surfaces[SURFACE])
end
```

(Exact API: prefer `game.get_entity_by_unit_number` if available in 2.0.77; else store and find by unit_number on surface — verify against `apidoc/factorio-api.md` at implement time.)

### Move event already available

- Softmod: `softmod/factop/player.lua` → `[move] name:index:x:y:surface:surfaceIndex`
- Go: `client/player.Move`, `plugin.OnMove`
- Threshold: `MOVEMENT_THRESHOLD = 2` tiles

---

## 4. New events (codegen)

**Package:** `client/companion/companion.go`

| Tag | Fields (suggested wire order) |
|-----|-------------------------------|
| `companion_summoned` | Name, PlayerIndex, PlayerName, UnitNumber, X, Y, SurfaceName, SurfaceIndex |
| `companion_dismissed` | same |
| `companion_died` | same |

- Use `// gen:event tag=...` for each; **hand-emit from softmod** (command-driven, not a Factorio define event).
- `go run ./gen` → `*_gen.go`, `plugin/events_companion_gen.go`.
- Do **not** generate `player.lua`; do **not** put follow into gen templates.

Example wire:

```text
[companion_summoned] merlin:1:mlctrez:4242:10.0:20.0:nauvis:1
```

---

## 5. Softmod module

**File:** `softmod/factop/companion.lua` (auto `add_lib` via embed walk).

- `commands.add_command("merlin", ...)` — summon for `cmd.player_index`.
- `commands.add_command("merlin-dismiss", ...)` — destroy if present.
- `on_init` / `on_load`: **register_commands only** (idempotent; no storage init on load).
- Lazy `ensure_storage()` in command handlers and death handler only.
- `storage.companions[player_index] = { name="merlin", unit_number=... }`.
- On summon: create entity, fill storage, `helpers.send_udp(4000, "[companion_summoned] ...", 0)`.
- On dismiss / invalid entity: destroy, emit dismissed/died, clear storage.
- Optional `on_entity_died` or periodic validity check.
- **v1: no nth_tick follow** if using plugin OnMove follow.

---

## 6. Plugin

**Path:** `pluginexample/merlin/` (name flexible).

- `Setup`: load optional JSON map from `DataDir()` (`companions.json`: player index → unit number + name).
- `Run`:
  - `OnCompanionSummoned` → save mapping, flavor print.
  - `OnCompanionDismissed` / `OnCompanionDied` → clear mapping, print.
  - `OnMove` → if mapping exists for `mv.Index`, RCON teleport that unit to offset near position (same surface as move).
- Persist mapping on change (same pattern as `rpgdemo` store: write on change, not only on shutdown).
- Coexist with `rpgdemo` and `pluginone` (all can subscribe to `OnMove`).

---

## 7. Implementation order

1. Softmod `companion.lua`: summon/dismiss + storage + UDP emit (manual string format matching schema).
2. `client/companion` schemas + `go run ./gen`.
3. Plugin merlin: event handlers + OnMove follow + DataDir persistence.
4. Deploy softmod + plugin to host; playtest `/merlin`, walk, dismiss.
5. Short note in `docs/rpg.md` under companions / plugins.

---

## 8. Testing checklist

- [ ] `/merlin` spawns character; `focmd watch` shows `udp.incoming.companion_summoned`.
- [ ] Walk ≥2 tiles → Merlin teleports near player (plugin RCON).
- [ ] `/merlin-dismiss` removes entity + event.
- [ ] Softmod redeploy does not crash on_load (no storage write in on_load).
- [ ] Plugin restart: re-summon or rehydrate from events; document if mapping is lost until re-summon.
- [ ] Two players: each can have own Merlin (unit_number isolation).
- [ ] `go test` for companion parse / plugin package if tests added.

---

## 9. Out of scope (v1)

- Custom Merlin graphics / non-`character` prototype.
- Softmod nth_tick smooth follow (done in v2, see section 0; still no real pathfinding).
- Chat NLP summon; combat AI; dialogue trees.
- Fire/UDP outgoing path.
- Changing global move threshold for everyone.

---

## 10. Key repo references (for implementer)

| Path | Relevance |
|------|-----------|
| `docs/rpg.md` | Product architecture (plugins = brains) |
| `softmod/factop/player.lua` | Move emit + **storage/on_load rules** |
| `pluginexample/pluginone/main.go` | OnMove + RCON pattern |
| `pluginexample/rpgdemo/` | Plugin DataDir state pattern |
| `client/entity/entity.go` | Multi-tag event annotation example |
| `client/player/player.go` | Move schema (`gen:event` only, no gen:lua) |
| `gen/` | `go run ./gen` pipeline |
| `softmod/softmod.go` | Auto-discovers `factop/*.lua` except `common.lua` |
| `plugin/events.go` | EventRouter |
| `apidoc/factorio-api.md` | `create_entity`, character, player APIs |

---

## 11. Success criteria

- Player can `/merlin` and see a companion character follow them as they walk.
- Follow uses **existing** `[move]` stream (no new per-step UDP).
- Plugin is the place for Merlin behavior policy; softmod handles in-world entity + commands + lifecycle events.
- No multiplayer storage CRC crash on softmod load.

---

## 12. Decision log (for the next LLM)

1. **User requested a plugin** for NPC Merlin summon + follow; suspected new events needed.
2. First plan put follow in softmod nth_tick to avoid “extra messaging.”
3. **User challenge:** move messages already exist — follow would not add move UDP.
4. **Revised:** that challenge is **correct for UDP/NATS**. Residual cost is **RCON per move**, same class as pluginone. v1 should **prefer plugin OnMove follow** for alignment with project structure; softmod still does summon command + entity create + lifecycle UDP events.
5. Handoff written for continuation under weekly token limits.
