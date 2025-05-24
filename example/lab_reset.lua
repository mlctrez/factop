-- clears all chunks on the map except for {0,0}
local surface = game.surfaces["nauvis"];
surface.daytime = 0
surface.freeze_daytime = true

-- move player to the center of the {0,0} chunk
game.players[1].teleport({ 16, 16 })
game.players[1].force.set_spawn_position({ 16, 16 }, surface)
--game.players[1].force.set_spawn_position(game.players[1].position, game.players[1].surface)
game.players[1].force.clear_chart()
game.forces["player"].cancel_charting(surface);
for chunk in surface.get_chunks() do
    if math.abs(chunk.x) > 0 or math.abs(chunk.y) > 0 then
        surface.delete_chunk(chunk)
    else
        local to_delete = surface.find_entities_filtered { area = chunk.area }
        for _, ent in pairs(to_delete) do
            local isPlayerCharacter = false
            for _, p in pairs(game.players) do
                if ent.name == "character" and p.character == ent then
                    isPlayerCharacter = true
                end
            end
            if not isPlayerCharacter then
                ent.destroy()
            end
        end
    end
end

local tiles = {}
for x = 0, 31 do
    for y = 0, 31 do
        local name = "lab-dark-1"
        if (x + y) % 2 == 1 then
            name = "lab-dark-2"
        end
        --if x == 0 or y == 0 or x == 31 or y == 31 then
        --    name = "lab-dark-1"
        --end
        table.insert(tiles, { name = name, position = { x, y } })
    end
end
surface.set_tiles(tiles)

--surface.request_to_generate_chunks({ 0, 0 }, 15)
--surface.force_generate_chunk_requests()


