function playerPositions()
    local tableResult = {}
    for _, player in pairs(game.players) do
        if player and player.valid and player.connected then
            table.insert(tableResult, { name = player.name, position = player.position })
        end
    end
    rcon.print(helpers.table_to_json(tableResult))
end

function labTiles()
    local s = game.surfaces[1]
    local tiles = {}
    for x = -32, 32 do
        for y = -32, 32 do
            local name = "lab-dark-1"
            if (x + y) % 2 == 1 then
                name = "lab-dark-2"
            end
            table.insert(tiles, { name = name, position = { x, y } })
        end
    end
    s.set_tiles(tiles)
end