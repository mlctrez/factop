factop_player = {}

factop_player.storage = function(event)
    if storage.factop_player_storage == nil then
        storage.factop_player_storage = {}
    end
    if storage.factop_player_storage[event.player_index] == nil then
        storage.factop_player_storage[event.player_index] = {}
    end
    return storage.factop_player_storage[event.player_index]
end

factop_player.position_updated = function(event)
    if event.player_index == nil then
        return
    end
    local player = game.players[event.player_index]
    if player == nil then
        return
    end
    if player.character == nil then
        return
    end
    factop_player.storage(event)["position"] = player.position
    print("factop_player.position_updated")
end

factop_player.created = function(event)
    print("factop_player.created")
    factop_player.position_updated(event)
end

factop_player.joined_game = function(event)
    print("factop_player.joined_game")
    factop_player.position_updated(event)
end

factop_player.changed_position = function(event)
    local player = game.players[event.player_index]
    local oldPosition = factop_player.storage(event)["position"]
    if oldPosition == nil then
        factop_player.position_updated(event)
    else
        local movement_delta = 4
        if math.abs(oldPosition.x - player.position.x) > movement_delta or
                math.abs(oldPosition.y - player.position.y) > movement_delta then
            factop_player.position_updated(event)
        end
    end
end

factop_player.log_selected_area = function(name, event)
    local r = {
        item = event.item, player_index = event.player_index,
        surface = event.surface.name, area = event.area
    }
    print("factop_player." .. name .. " -> " .. helpers.table_to_json(r))
end

-- https://lua-api.factorio.com/latest/events.html#on_player_selected_area
factop_player.selected_area = function(event)
    factop_player.log_selected_area("selected_area", event)
end

-- https://lua-api.factorio.com/latest/events.html#on_player_alt_selected_area
factop_player.alt_selected_area = function(event)
    factop_player.log_selected_area("alt_selected_area", event)
end

-- https://lua-api.factorio.com/latest/events.html#on_player_used_spidertron_remote
factop_player.used_spidertron_remote = function(event)
    local r = {
        player_index = event.player_index, position = event.position
    }
    print("factop_player.used_spidertron_remote -> " .. helpers.table_to_json(r))
end

-- https://lua-api.factorio.com/latest/events.html#on_player_used_capsule
factop_player.used_capsule = function(event)
    local r = {
        player_index = event.player_index, item = event.item.name,
        position = event.position
    }
    print("factop_player.used_capsule -> " .. helpers.table_to_json(r))
end

factop_player.died = function(event)
    print("factop_player.died")
    factop_player.position_updated(event)
end

factop_player.respawned = function(event)
    print("factop_player.respawned")
    factop_player.position_updated(event)
end

factop_player.left_game = function(event)
    print("factop_player.left_game")
    factop_player.position_updated(event)
end

factop_player.total_connected = function()
    local count = 0
    for _, player in pairs(game.players) do
        if player.connected then
            count = count + 1
        end
    end
    return count
end

factop_player.events = {
    [defines.events.on_player_created] = factop_player.created,
    [defines.events.on_player_joined_game] = factop_player.joined_game,
    [defines.events.on_player_changed_position] = factop_player.changed_position,
    [defines.events.on_player_selected_area] = factop_player.selected_area,
    [defines.events.on_player_alt_selected_area] = factop_player.alt_selected_area,
    [defines.events.on_player_used_spidertron_remote] = factop_player.used_spidertron_remote,
    [defines.events.on_player_used_capsule] = factop_player.used_capsule,
    [defines.events.on_player_died] = factop_player.died,
    [defines.events.on_player_respawned] = factop_player.respawned,
    [defines.events.on_player_left_game] = factop_player.left_game,
}

return factop_player
