local player_mod = {}
local c = require("factop.common")

-- Player event tracking and manipulation module for factop softmod.
-- Handles UDP event logging for player lifecycle events, movement tracking,
-- and RCON commands for player management (list, respawn, teleport).

-- Distance in tiles before recording a new position
local MOVEMENT_THRESHOLD = 2

local function player_mod_setup()
    if storage.player_movement == nil then
        storage.player_movement = {}
    end
end

local function log_event(event_name, event, additional_data)
    local player_index = event.player_index
    local player = game.players[player_index]
    if not (player and player.valid) then return end

    local message = string.format("[%s] #%d %s", event_name, player_index, player.name)
    if additional_data then
        message = message .. " " .. additional_data
    end

    if #message > 512 then
        message = message:sub(1, 512)
    end

    helpers.send_udp(4000, message, 0)
end

-- ---------------------------------------------------------------------------
-- Event handlers
-- ---------------------------------------------------------------------------

local function on_player_joined_game(event)
    player_mod_setup()
    log_event("join", event)

    local player = game.players[event.player_index]
    if player and player.valid then
        -- Record initial position
        storage.player_movement[event.player_index] = {
            x = player.position.x, y = player.position.y
        }
        -- Auto-create character if missing (e.g. after script-based entity clearing)
        if not player.character or not player.character.valid then
            player.create_character()
        end
    end
end

local function on_player_left_game(event)
    log_event("leave", event)
end

local function on_player_died(event)
    local cause = "unknown"
    if event.cause and event.cause.valid then
        cause = event.cause.name
    end
    log_event("death", event, string.format("by %s", cause))
end

local function on_player_respawned(event)
    player_mod_setup()
    log_event("respawn", event)
    local player = game.players[event.player_index]
    if player and player.valid then
        storage.player_movement[event.player_index] = {
            x = player.position.x, y = player.position.y
        }
    end
end

local function on_player_changed_position(event)
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
            if math.sqrt(dx * dx + dy * dy) >= MOVEMENT_THRESHOLD then
                should_update = true
            end
        end

        if should_update then
            storage.player_movement[event.player_index] = {
                x = current_pos.x, y = current_pos.y
            }
            log_event("move", event, string.format("to {x=%.1f, y=%.1f}",
                current_pos.x, current_pos.y))
        end
    end
end

-- ---------------------------------------------------------------------------
-- Console commands
-- ---------------------------------------------------------------------------

local function register_commands()

    commands.add_command("player-respawn", "Respawn player character. Usage: /player-respawn [player]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)

        if #args >= 1 then
            local player = game.get_player(args[1])
            if not player then
                local idx = tonumber(args[1])
                if idx then player = game.players[idx] end
            end
            if not player then c.reply("Player not found: " .. args[1]) return end
            if player.character and player.character.valid then
                c.reply(player.name .. " already has a character")
                return
            end
            local ok = player.create_character()
            if ok then
                c.reply("Created character for " .. player.name)
            else
                c.reply("Failed to create character for " .. player.name)
            end
        else
            local count = 0
            for _, player in pairs(game.connected_players) do
                if not player.character or not player.character.valid then
                    if player.create_character() then
                        count = count + 1
                    end
                end
            end
            c.reply("Respawned " .. count .. " players")
        end
    end)

    commands.add_command("player-teleport", "Teleport player. Usage: /player-teleport player x,y [surface]", function(cmd)
        if not c.rcon_only(cmd) then return end
        local args = c.parse_args(cmd)
        if #args < 2 then c.reply("Usage: /player-teleport player x,y [surface]") return end

        local player = game.get_player(args[1])
        if not player then
            local idx = tonumber(args[1])
            if idx then player = game.players[idx] end
        end
        if not player then c.reply("Player not found: " .. args[1]) return end

        local pos = c.parse_position(args[2])
        if not pos then c.reply("Invalid position. Use: x,y") return end

        local surface = c.get_surface(args[3])
        if not surface then c.reply("Surface not found") return end

        player.teleport(pos, surface)
        c.reply(string.format("Teleported %s to {%.1f,%.1f}", player.name, pos.x, pos.y))
    end)

end

-- ---------------------------------------------------------------------------
-- Module registration
-- ---------------------------------------------------------------------------

player_mod.events = {
    [defines.events.on_player_joined_game] = on_player_joined_game,
    [defines.events.on_player_left_game] = on_player_left_game,
    [defines.events.on_player_died] = on_player_died,
    [defines.events.on_player_respawned] = on_player_respawned,
    [defines.events.on_player_changed_position] = on_player_changed_position,
}

player_mod.on_init = register_commands
player_mod.on_load = register_commands

return player_mod
