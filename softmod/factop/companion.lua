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

local function on_entity_died(event)
    ensure_storage()
    local entity = event.entity
    if not (entity and entity.valid) then return end

    local unit_number = entity.unit_number
    for player_index, companion in pairs(storage.companions) do
        if companion.unit_number == unit_number then
            local msg = string.format("[companion_died] merlin:%d:%s:%d:%.1f:%.1f:%s:%d",
                player_index, companion.player_name, companion.unit_number,
                companion.x, companion.y, companion.surface_name, companion.surface_index)
            helpers.send_udp(4000, msg, 0)
            storage.companions[player_index] = nil
            return
        end
    end
end

companion_mod.on_init = register_commands
companion_mod.on_load = register_commands
companion_mod.events = {
    [defines.events.on_entity_died] = on_entity_died,
}

return companion_mod
