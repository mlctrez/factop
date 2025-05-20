game.surfaces["nauvis"].show_clouds = false
game.surfaces["nauvis"].always_day = true

function fillArea(surface_name, left_top_x, left_top_y, right_bottom_x, right_bottom_y, tile_name)
    local surface = game.surfaces[surface_name]
    if not surface then
        return false
    end

    local tiles = {}
    for x = left_top_x, right_bottom_x do
        for y = left_top_y, right_bottom_y do
            table.insert(tiles, { name = tile_name, position = { x = x, y = y } })
        end
    end

    surface.set_tiles(tiles)
    return true
end
local min = -32
local max = 31
--fillArea("nauvis", min, min, max, max, "refined-concrete")

min = -225
max = 224

fillArea("nauvis", min, min, max, max, "refined-concrete")