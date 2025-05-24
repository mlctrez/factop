function findTiles(surfaceName, position, radius)
    local s = game.surfaces[surfaceName]
    local tableResult = {}
    if s ~= nil and s.valid then
        local tiles = {}
        local count = 0
        local searchTiles = s.find_tiles_filtered { position = position, radius = radius, limit = 500 }
        for _, v in pairs(searchTiles) do
            if v ~= nil and v.valid then
                count = count + 1
                table.insert(tiles, { name = v.name, position = v.position })
            end
        end
        -- helpers.table_to_json treats an empty table as an object
        -- so only set the tiles property if there were some tiles found
        if count > 0 then
            tableResult = { tiles = tiles, count = count }
        else
            tableResult = { count = count }
        end
    else
        tableResult = { error = "surface not found" }
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
