factop_tiles = {}

factop_tiles.create = function(sx, ex, sy, ey, surface, name)
    local tiles = {}
    for x = sx, ex do
        for y = sy, ey do
            table.insert(tiles, { name = name, position = { x, y } })
        end
    end
    game.surfaces[surface].set_tiles(tiles)
end

factop_tiles.replace = function(surface, area, old_tile, new_tile)
    local s = game.surfaces[surface]
    local tiles = {}
    for _, tile in pairs(s.find_tiles_filtered({ area = area, name = old_tile })) do
        table.insert(tiles, { name = new_tile, position = tile.position })
    end
    s.set_tiles(tiles)
end

factop_tiles.set_tile = function(surface, x, y, name)
    local s = game.surfaces[surface]
    local tiles = {}
    table.insert(tiles, { name = name, position = { x, y } })
    s.set_tiles(tiles)
end

