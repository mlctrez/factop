-- Pure-Lua unit tests for softmod/factop/companion.lua follow logic.
-- Stubs the Factorio globals the module touches. Factorio API methods are
-- called with "." (not ":"), so stubs take no self argument. Run from the repo root:
--   lua5.2 softmod/testdata/companion_test.lua
-- (also run by TestCompanionLua in softmod/softmod_test.go).

package.path = "softmod/factop/?.lua;" .. package.path

defines = {
    events = { on_entity_died = 1, on_player_removed = 2 },
    controllers = { character = 1, ghost = 2, remote = 3 },
    direction = {
        north = 0, northeast = 2, east = 4, southeast = 6,
        south = 8, southwest = 10, west = 12, northwest = 14,
    },
}
storage = {}
local sent = {}
helpers = { send_udp = function(_, msg) sent[#sent + 1] = msg end }
commands = { add_command = function() end }

local entities, players = {}, {}
game = {
    get_entity_by_unit_number = function(n) return entities[n] end,
    get_player = function(i) return players[i] end,
}

local c = require("companion")

local failures = 0
local function check(cond, msg)
    if not cond then
        failures = failures + 1
        print("FAIL: " .. msg)
    end
end
local function eq(got, want, msg)
    check(got == want, string.format("%s: got %s want %s", msg, tostring(got), tostring(want)))
end

-- direction_for: 8-way heading, north is -y.
local dirs = {
    { 0, -5, "north" }, { 0, 5, "south" }, { 5, 0, "east" }, { -5, 0, "west" },
    { 3, -3, "northeast" }, { -3, -3, "northwest" }, { 3, 3, "southeast" }, { -3, 3, "southwest" },
    { 1, -10, "north" }, { 10, 1, "east" }, { -10, -1, "west" }, { -1, 10, "south" },
    { 4, -9, "northeast" }, { 9, 4, "southeast" }, -- just past 22.5 degrees off axis
    { 3.9, -10, "north" }, { 10, 3.9, "east" },  -- just inside 22.5 degrees
}
for _, d in ipairs(dirs) do
    eq(c.direction_for(d[1], d[2]), d[3], string.format("direction_for(%g,%g)", d[1], d[2]))
end

-- follow_decision: hysteresis band and teleport fallback.
local S, A, T = c.FOLLOW_STOP_DIST, c.FOLLOW_START_DIST, c.FOLLOW_TELEPORT_DIST
check(S < A and A < T, "thresholds ordered stop < start < teleport")
local cases = {
    { 0, false, "stop" }, { 0, true, "stop" },
    { S, true, "stop" }, { S, false, "stop" },
    { S + 0.1, false, "hold" }, { A, false, "hold" },
    { A + 0.1, false, "walk" }, { S + 0.1, true, "walk" }, { A, true, "walk" },
    { T, false, "walk" }, { T, true, "walk" },
    { T + 0.1, false, "teleport" }, { T + 0.1, true, "teleport" }, { 500, true, "teleport" },
}
for _, k in ipairs(cases) do
    eq(c.follow_decision(k[1] * k[1], k[2]), k[3],
        string.format("follow_decision(dist=%g, walking=%s)", k[1], tostring(k[2])))
end

-- Simulation helpers for on_follow_tick.
local function new_entity(unit, x, y)
    local e = { valid = true, unit_number = unit, position = { x = x, y = y },
        surface_index = 1, direction = 0, walking_state = { walking = false, direction = 0 } }
    e.surface = {
        find_non_colliding_position = function(name, pos) return { x = pos.x + 1, y = pos.y } end,
    }
    e.teleport = function(pos) e.position = { x = pos.x, y = pos.y }; return true end
    e.destroy = function() e.valid = false; entities[unit] = nil end
    entities[unit] = e
    return e
end
local function new_player(i, x, y)
    local p = { valid = true, connected = true, name = "p" .. i,
        physical_controller_type = defines.controllers.character,
        physical_surface_index = 1, physical_position = { x = x, y = y } }
    players[i] = p
    return p
end
local function summon(i, e)
    storage.companions = storage.companions or {}
    storage.companions[i] = { name = "merlin", unit_number = e.unit_number, x = e.position.x,
        y = e.position.y, surface_name = "nauvis", surface_index = 1, player_name = "p" .. i }
end

-- Guard: no companions -> no work, no storage writes.
storage = {}
c.on_follow_tick()
eq(storage.companions, nil, "no companions table created by tick")

-- Far-ish owner: Merlin starts walking east toward them.
local e = new_entity(10, 0, 0)
local p = new_player(1, 8, 0)
summon(1, e)
c.on_follow_tick()
eq(e.walking_state.walking, true, "walks when owner beyond start dist")
eq(e.walking_state.direction, defines.direction.east, "walks east")

-- Close enough: stops.
e.position = { x = 6.5, y = 0 }
c.on_follow_tick()
eq(e.walking_state.walking, false, "stops within stop dist")

-- Owner steps a little (inside start dist): stays idle (no jitter).
p.physical_position = { x = 9.5, y = 0 }
c.on_follow_tick()
eq(e.walking_state.walking, false, "holds when idle inside start dist")

-- Owner far away: teleported next to them, not walking.
p.physical_position = { x = 100, y = 50 }
c.on_follow_tick()
eq(e.position.x, 101, "teleported x"); eq(e.position.y, 50, "teleported y")
eq(e.walking_state.walking, false, "not walking after teleport")

-- Stuck: walking but not moving for FOLLOW_STUCK_TICKS -> teleport.
e.position = { x = 90, y = 50 }
local ticks = 0
local teleported = false
for _ = 1, 100 do
    c.on_follow_tick()
    ticks = ticks + c.FOLLOW_INTERVAL
    if e.position.x == 101 then teleported = true; break end
end
check(teleported, "stuck companion teleported")
check(ticks >= c.FOLLOW_STUCK_TICKS and ticks <= c.FOLLOW_STUCK_TICKS + 2 * c.FOLLOW_INTERVAL,
    "stuck teleport after ~FOLLOW_STUCK_TICKS, took " .. ticks)

-- Owner on another surface / dead / offline: Merlin waits, mapping kept.
for _, mut in ipairs({
    function() p.physical_surface_index = 2 end,
    function() p.physical_controller_type = defines.controllers.ghost end,
    function() p.connected = false end,
}) do
    e.position = { x = 0, y = 0 }
    e.walking_state = { walking = true, direction = 4 }
    p.physical_surface_index, p.physical_controller_type, p.connected = 1, defines.controllers.character, true
    mut()
    c.on_follow_tick()
    eq(e.walking_state.walking, false, "waits while owner unavailable")
    check(storage.companions[1] ~= nil, "mapping kept while owner unavailable")
end
p.physical_surface_index, p.physical_controller_type, p.connected = 1, defines.controllers.character, true

-- Entity destroyed outside on_entity_died: dismissed event + cleanup.
sent = {}
e.destroy()
c.on_follow_tick()
eq(storage.companions[1], nil, "mapping removed when entity gone")
eq(#sent, 1, "one event sent")
check(sent[1] and sent[1]:find("^%[companion_dismissed%] merlin:1:p1:10:") ~= nil,
    "dismissed event emitted: " .. tostring(sent[1]))

-- Owner gone (removed player): entity destroyed + dismissed.
sent = {}
local e2 = new_entity(20, 0, 0)
summon(2, e2)
players[2] = nil
c.on_follow_tick()
eq(e2.valid, false, "orphan companion destroyed")
eq(storage.companions[2], nil, "orphan mapping removed")
check(sent[1] and sent[1]:find("^%[companion_dismissed%] merlin:2:") ~= nil, "orphan dismissed event")

-- on_entity_died emits died with the entity position.
sent = {}
local e3 = new_entity(30, 3, 4)
new_player(3, 3, 4)
summon(3, e3)
c.events[defines.events.on_entity_died]({ entity = e3 })
eq(storage.companions[3], nil, "died mapping removed")
check(sent[1] == "[companion_died] merlin:3:p3:30:3.0:4.0:nauvis:1", "died event: " .. tostring(sent[1]))

-- on_player_removed destroys + dismisses.
sent = {}
local e4 = new_entity(40, 1, 1)
summon(4, e4)
c.events[defines.events.on_player_removed]({ player_index = 4 })
eq(e4.valid, false, "removed player's companion destroyed")
eq(storage.companions[4], nil, "removed player's mapping cleared")
check(sent[1] and sent[1]:find("^%[companion_dismissed%] merlin:4:") ~= nil, "removed player dismissed event")

-- Registered on the follow interval.
check(c.on_nth_tick[c.FOLLOW_INTERVAL] == c.on_follow_tick, "on_nth_tick registered")

if failures > 0 then
    print(failures .. " failure(s)")
    os.exit(1)
end
print("ok")
