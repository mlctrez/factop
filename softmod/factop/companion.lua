-- Companion entity manager for factop.
-- Provides /merlin and /merlin-dismiss player commands.
-- Emits [companion_summoned], [companion_dismissed], [companion_died] UDP events.

local companion_mod = {}

local function ensure_storage()
    if storage.companions == nil then
        storage.companions = {}
    end
end

local function register_commands()
    pcall(function()
        commands.add_command("merlin", "Summon Merlin companion", function(cmd)
            if not cmd.player_index or cmd.player_index == 0 then return end
            ensure_storage()

            local player_index = cmd.player_index
            local player = game.players[player_index]
            if not (player and player.valid) then return end

            if storage.companions[player_index] then
                player.print("You already have an active companion.")
                return
            end

            local surface = player.surface
            local pos = player.position
            local entity = surface.create_entity{
                name = "character",
                position = { pos.x + 2, pos.y },
                force = player.force,
            }
            if not entity then
                player.print("Failed to create companion entity.")
                return
            end

            entity.color = { r = 0, g = 0, b = 0, a = 1 }

            storage.companions[player_index] = {
                name = "merlin",
                unit_number = entity.unit_number,
                x = entity.position.x,
                y = entity.position.y,
                surface_name = surface.name,
                surface_index = surface.index,
                player_name = player.name,
            }

            local msg = string.format("[companion_summoned] merlin:%d:%s:%d:%.1f:%.1f:%s:%d",
                player_index, player.name, entity.unit_number,
                entity.position.x, entity.position.y,
                surface.name, surface.index)
            helpers.send_udp(4000, msg, 0)
        end)
    end)

    pcall(function()
        commands.add_command("merlin-dismiss", "Dismiss Merlin companion", function(cmd)
            if not cmd.player_index or cmd.player_index == 0 then return end
            ensure_storage()

            local player_index = cmd.player_index
            local player = game.players[player_index]
            if not (player and player.valid) then return end

            local companion = storage.companions[player_index]
            if not companion then
                player.print("You have no active companion.")
                return
            end

            -- Capture position data from entity if valid, otherwise use stored data
            local x = companion.x
            local y = companion.y
            local surface_name = companion.surface_name
            local surface_index = companion.surface_index

            local entity = game.get_entity_by_unit_number(companion.unit_number)
            if entity and entity.valid then
                x = entity.position.x
                y = entity.position.y
                entity.destroy()
            end

            local msg = string.format("[companion_dismissed] merlin:%d:%s:%d:%.1f:%.1f:%s:%d",
                player_index, companion.player_name, companion.unit_number,
                x, y, surface_name, surface_index)
            helpers.send_udp(4000, msg, 0)

            storage.companions[player_index] = nil
            player.print("Companion dismissed.")
        end)
    end)
end

local function emit(tag, player_index, companion, x, y)
    local msg = string.format("[%s] merlin:%d:%s:%d:%.1f:%.1f:%s:%d",
        tag, player_index, companion.player_name, companion.unit_number,
        x, y, companion.surface_name, companion.surface_index)
    helpers.send_udp(4000, msg, 0)
end

local function on_entity_died(event)
    if not storage.companions then return end
    local entity = event.entity
    if not (entity and entity.valid) then return end

    local unit_number = entity.unit_number
    for player_index, companion in pairs(storage.companions) do
        if companion.unit_number == unit_number then
            local pos = entity.position
            emit("companion_died", player_index, companion, pos.x, pos.y)
            storage.companions[player_index] = nil
            return
        end
    end
end

-- A removed player can no longer own a companion: destroy it and report it
-- as dismissed.
local function on_player_removed(event)
    if not storage.companions then return end
    local companion = storage.companions[event.player_index]
    if not companion then return end
    local x, y = companion.x, companion.y
    local entity = game.get_entity_by_unit_number(companion.unit_number)
    if entity and entity.valid then
        x, y = entity.position.x, entity.position.y
        entity.destroy()
    end
    emit("companion_dismissed", event.player_index, companion, x, y)
    storage.companions[event.player_index] = nil
end

---------------------------------------------------------------------------
-- Follow: every FOLLOW_INTERVAL ticks each companion walks toward its owner.
---------------------------------------------------------------------------

-- How often the follow handler runs. A character runs ~0.15 tiles/tick, so
-- re-aiming every 3 ticks (20x/sec) moves Merlin at most ~0.45 tiles between
-- decisions: well inside the stop band, so he never overshoots far and the
-- 8-way heading updates smoothly. Cost is one entity lookup per companion.
local FOLLOW_INTERVAL = 3
-- Hysteresis band (tiles from the owner). A walking companion stops once
-- within FOLLOW_STOP_DIST; an idle one only starts walking again once the
-- owner is farther than FOLLOW_START_DIST, so small moves don't cause jitter.
local FOLLOW_STOP_DIST = 2
local FOLLOW_START_DIST = 4
-- Farther than this (same surface) Merlin is teleported next to the owner
-- instead of walking (fast vehicles, trains, respawn at spawn, ...).
local FOLLOW_TELEPORT_DIST = 20
-- If Merlin tries to walk but barely moves for this many ticks he is
-- considered stuck (water, buildings) and teleported next to the owner.
local FOLLOW_STUCK_TICKS = 60
local FOLLOW_STUCK_EPSILON = 0.05 -- tiles moved per check that count as "stuck"
-- Radius searched for a free spot when teleporting.
local TELEPORT_SEARCH_RADIUS = 8

-- tan(22.5 degrees): boundary between a straight and a diagonal heading.
local TAN_22_5 = 0.41421356237

-- direction_for returns the 8-way defines.direction name pointing along
-- (dx, dy) in map coordinates (north is -y, east is +x). Uses only
-- comparisons, so it is deterministic and needs no trig.
local function direction_for(dx, dy)
    local ax, ay = math.abs(dx), math.abs(dy)
    if ax <= ay * TAN_22_5 then
        if dy < 0 then return "north" end
        return "south"
    end
    if ay <= ax * TAN_22_5 then
        if dx > 0 then return "east" end
        return "west"
    end
    if dy < 0 then
        if dx > 0 then return "northeast" end
        return "northwest"
    end
    if dx > 0 then return "southeast" end
    return "southwest"
end

-- follow_decision is the pure follow policy. dist_sq is the squared distance
-- from Merlin to his owner and walking says whether he is walking now.
-- Returns "teleport", "stop", "hold" (leave idle) or "walk".
local function follow_decision(dist_sq, walking)
    if dist_sq > FOLLOW_TELEPORT_DIST * FOLLOW_TELEPORT_DIST then
        return "teleport"
    end
    if dist_sq <= FOLLOW_STOP_DIST * FOLLOW_STOP_DIST then
        return "stop"
    end
    if not walking and dist_sq <= FOLLOW_START_DIST * FOLLOW_START_DIST then
        return "hold"
    end
    return "walk"
end

local function stop_walking(entity)
    if entity.walking_state.walking then
        entity.walking_state = { walking = false, direction = entity.direction }
    end
end

local function teleport_near(entity, surface, pos)
    local dest = surface.find_non_colliding_position("character", pos,
        TELEPORT_SEARCH_RADIUS, 0.5) or pos
    stop_walking(entity)
    entity.teleport(dest)
end

-- follow_one advances one companion. Returns false if the companion should
-- be dropped from storage (entity gone or owner gone).
local function follow_one(player_index, companion)
    local entity = game.get_entity_by_unit_number(companion.unit_number)
    if not (entity and entity.valid) then
        -- Destroyed without on_entity_died (script/editor): report dismissed.
        emit("companion_dismissed", player_index, companion, companion.x, companion.y)
        return false
    end

    local epos = entity.position
    companion.x, companion.y = epos.x, epos.y

    local player = game.get_player(player_index)
    if not (player and player.valid) then
        entity.destroy()
        emit("companion_dismissed", player_index, companion, epos.x, epos.y)
        return false
    end

    -- Wait in place while the owner is offline, dead/respawning (no physical
    -- character), or on another surface (characters can't be teleported
    -- across surfaces). Following resumes when the owner is back; if they are
    -- then far away Merlin teleports to them.
    if not player.connected
        or player.physical_controller_type ~= defines.controllers.character
        or player.physical_surface_index ~= entity.surface_index then
        companion.stuck_ticks = nil
        stop_walking(entity)
        return true
    end

    -- physical_position is the character (or the vehicle it sits in), not the
    -- remote-view camera.
    local ppos = player.physical_position
    local dx = ppos.x - epos.x
    local dy = ppos.y - epos.y
    local walking = entity.walking_state.walking
    local action = follow_decision(dx * dx + dy * dy, walking)

    if action == "walk" and walking and companion.last_x then
        local mx = epos.x - companion.last_x
        local my = epos.y - companion.last_y
        if mx * mx + my * my < FOLLOW_STUCK_EPSILON * FOLLOW_STUCK_EPSILON then
            companion.stuck_ticks = (companion.stuck_ticks or 0) + FOLLOW_INTERVAL
            if companion.stuck_ticks >= FOLLOW_STUCK_TICKS then
                action = "teleport"
            end
        else
            companion.stuck_ticks = nil
        end
    elseif action ~= "walk" then
        companion.stuck_ticks = nil
    end
    companion.last_x, companion.last_y = epos.x, epos.y

    if action == "teleport" then
        teleport_near(entity, entity.surface, ppos)
        companion.stuck_ticks = nil
        companion.last_x, companion.last_y = nil, nil
        companion.x, companion.y = entity.position.x, entity.position.y
    elseif action == "stop" then
        stop_walking(entity)
    elseif action == "walk" then
        local dir = defines.direction[direction_for(dx, dy)]
        local ws = entity.walking_state
        if not (ws.walking and ws.direction == dir) then
            entity.walking_state = { walking = true, direction = dir }
        end
    end
    return true
end

local function on_follow_tick()
    -- Cheap guard: nothing to do (and no storage writes) without companions.
    local companions = storage.companions
    if not companions or next(companions) == nil then return end
    local remove
    for player_index, companion in pairs(companions) do
        if not follow_one(player_index, companion) then
            remove = remove or {}
            remove[#remove + 1] = player_index
        end
    end
    if remove then
        for _, player_index in ipairs(remove) do
            companions[player_index] = nil
        end
    end
end

companion_mod.on_init = register_commands
companion_mod.on_load = register_commands -- commands only; never touch storage here
companion_mod.events = {
    [defines.events.on_entity_died] = on_entity_died,
    [defines.events.on_player_removed] = on_player_removed,
}
companion_mod.on_nth_tick = {
    [FOLLOW_INTERVAL] = on_follow_tick,
}

-- Exposed for unit tests (softmod/companion_test.lua); ignored by event_handler.
companion_mod.direction_for = direction_for
companion_mod.follow_decision = follow_decision
companion_mod.on_follow_tick = on_follow_tick
companion_mod.FOLLOW_INTERVAL = FOLLOW_INTERVAL
companion_mod.FOLLOW_STOP_DIST = FOLLOW_STOP_DIST
companion_mod.FOLLOW_START_DIST = FOLLOW_START_DIST
companion_mod.FOLLOW_TELEPORT_DIST = FOLLOW_TELEPORT_DIST
companion_mod.FOLLOW_STUCK_TICKS = FOLLOW_STUCK_TICKS

return companion_mod
