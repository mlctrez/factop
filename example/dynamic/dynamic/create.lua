factop_dynamic = {}

-- an example for creating tiles as the player moves, replacing only out-of-map tiles
-- this was tested in the softmod and works, moved here to preserve history

factop_dynamic.changed_position = function(event)
    local player = game.players[event.player_index]
    if not player then
        return
    end
    if not player.valid then
        return
    end
    if player.render_mode ~= defines.render_mode.game then
        return
    end
    local s = player.surface
    local searchTiles = s.find_tiles_filtered {
        name = "out-of-map", position = player.position, radius = 5, limit = 25
    }
    local tiles = {}
    local count = 0
    for _, tile in pairs(searchTiles) do
        count = count + 1
        table.insert(tiles, { name = "volcanic-ash-flats", position = tile.position })
    end
    if count > 0 then
        s.set_tiles(tiles)
    end
end

factop_dynamic.events = {
    [defines.events.on_player_changed_position] = factop_dynamic.changed_position,
}

return factop_dynamic