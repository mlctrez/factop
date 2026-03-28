local player_mod = {}

-- Distance in tiles before recording a new position
local MOVEMENT_THRESHOLD = 2

-- Helper to ensure storage is setup
local player_mod_setup = function()
    if storage.player_movement == nil then
        storage.player_movement = {}
    end
end

-- Helper to send UDP logs
local function log_event(event_name, event, additional_data)
    local player_index = event.player_index
    local player = game.players[player_index]
    if not (player and player.valid) then return end

    local message = string.format("[%s] #%d %s", event_name, player_index, player.name)
    if additional_data then
        message = message .. " " .. additional_data
    end

    -- Enforce max UDP packet size of 512 bytes
    if #message > 512 then
        message = message:sub(1, 512)
    end

    -- Using port 4000 for outgoing logs as per UDPBridge bridge to udp.incoming
    helpers.send_udp(4000, message, 0)
end

player_mod.on_player_joined_game = function(event)
    player_mod_setup()
    log_event("join", event)
    -- Record initial position
    storage.player_movement[event.player_index] = { x = game.players[event.player_index].position.x, y = game.players[event.player_index].position.y }
end

player_mod.on_player_left_game = function(event)
    log_event("leave", event)
end

player_mod.on_player_died = function(event)
    local cause = "unknown"
    if event.cause and event.cause.valid then
        cause = event.cause.name
    end
    log_event("death", event, string.format("by %s", cause))
end

player_mod.on_player_respawned = function(event)
    player_mod_setup()
    log_event("respawn", event)
    -- Reset position on respawn
    local player = game.players[event.player_index]
    if player and player.valid then
        storage.player_movement[event.player_index] = { x = player.position.x, y = player.position.y }
    end
end

player_mod.on_player_changed_position = function(event)
    player_mod_setup()
    local player = game.players[event.player_index]
    if player and player.valid then
        local last_pos = storage.player_movement[event.player_index]
        local current_pos = player.position

        local should_update = false
        if last_pos == nil then
            should_update = true
        else
            local dx = current_pos.x - last_pos.x
            local dy = current_pos.y - last_pos.y
            local distance = math.sqrt(dx * dx + dy * dy)
            if distance >= MOVEMENT_THRESHOLD then
                should_update = true
            end
        end

        if should_update then
            storage.player_movement[event.player_index] = { x = current_pos.x, y = current_pos.y }
            log_event("move", event, string.format("to {x=%.1f, y=%.1f}", current_pos.x, current_pos.y))
        end
    end
end

player_mod.events = {
    [defines.events.on_player_joined_game] = player_mod.on_player_joined_game,
    [defines.events.on_player_left_game] = player_mod.on_player_left_game,
    [defines.events.on_player_died] = player_mod.on_player_died,
    [defines.events.on_player_respawned] = player_mod.on_player_respawned,
    [defines.events.on_player_changed_position] = player_mod.on_player_changed_position
}

return player_mod
