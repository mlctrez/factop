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

factop_tiles.tile_line = function(s, name, start_pos, end_pos, width)
    local surface = game.surfaces[s]
    if surface == nil then
        error("no such game surface " .. s)
    end
    local dx = end_pos.x - start_pos.x
    local dy = end_pos.y - start_pos.y
    local length = (dx * dx + dy * dy) ^ 0.5
    if length == 0 then
        return
    end
    if width == nil then
        width = 1
    end

    dx = dx / length
    dy = dy / length
    local px = -dy
    local py = dx

    local tiles = {}
    for w = -width, width, 1 do
        for t = 0, length, 1 do
            local x = start_pos.x + t * dx + w * px
            local y = start_pos.y + t * dy + w * py
            table.insert(tiles, { name = name, position = { x, y } })
        end
    end
    surface.set_tiles(tiles)
end