-- Player event emitter for factop.
-- Emits [move] UDP messages when a player moves beyond the movement threshold.

local player_mod = {}
local c = require("factop.common")

local MOVEMENT_THRESHOLD = 2

local function setup()
    if storage.player_movement == nil then
        storage.player_movement = {}
    end
end

local function on_player_changed_position(event)
    setup()
    local p = game.players[event.player_index]
    if not (p and p.valid) then return end

    local last = storage.player_movement[event.player_index]
    local pos = p.position

    local should_emit = false
    if last == nil then
        should_emit = true
    else
        local dx = pos.x - last.x
        local dy = pos.y - last.y
        if math.sqrt(dx * dx + dy * dy) >= MOVEMENT_THRESHOLD then
            should_emit = true
        end
    end

    if should_emit then
        storage.player_movement[event.player_index] = { x = pos.x, y = pos.y }
        local surface = p.surface
        local msg = string.format("[move] %s:%d:%.1f:%.1f:%s:%d",
            p.name, p.index, pos.x, pos.y,
            surface.name, surface.index)
        helpers.send_udp(4000, msg, 0)
    end
end

player_mod.events = {
    [defines.events.on_player_changed_position] = on_player_changed_position,
}

player_mod.on_init = setup
player_mod.on_load = setup

return player_mod
