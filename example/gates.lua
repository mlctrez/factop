function createWalls(surface_name, left_top_x, left_top_y, right_bottom_x, right_bottom_y)
    local surface = game.surfaces[surface_name]
    if not surface then
        return false
    end

    for x = left_top_x, right_bottom_x do
        for y = left_top_y, right_bottom_y do
            if x == left_top_x or x == right_bottom_x or y == left_top_y or y == right_bottom_y then
                local wall = surface.create_entity {
                    name = "stone-wall",
                    position = { x = x, y = y },
                    force = "enemy",
                    raise_built = false,
                    move_stuck_players = true,
                    create_build_effect_smoke = false,
                }
                if wall then
                    wall.destructible = false
                end
            end
        end
    end
    return true
end

function removeWalls(surface_name, left_top_x, left_top_y, right_bottom_x, right_bottom_y)
    local surface = game.surfaces[surface_name]
    if not surface then
        return false
    end

    local area = { { left_top_x, left_top_y }, { right_bottom_x, right_bottom_y } }
    local walls = surface.find_entities_filtered { area = area, name = { "stone-wall", "gate" } }
    for _, wall in pairs(walls) do
        wall.destroy()
    end
    return true
end

local min = -10.5
local max = 10.5

removeWalls("nauvis", min, min, max, max)
--createWalls("nauvis", min, min, max, max)




