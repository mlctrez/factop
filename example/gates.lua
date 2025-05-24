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

function createWalls(surface_name, left_top_x, left_top_y, right_bottom_x, right_bottom_y)
    local surface = game.surfaces[surface_name]
    if not surface then
        return false
    end

    for x = left_top_x, right_bottom_x do
        for y = left_top_y, right_bottom_y do
            if x == left_top_x or x == right_bottom_x or y == left_top_y or y == right_bottom_y then

                local dir = nil
                local name = "stone-wall"
                if y == left_top_y and math.abs(x) < 2 then
                    name = "gate"
                    dir = defines.direction.east
                end

                local ent = surface.create_entity {
                    name = name,
                    position = { x = x, y = y },
                    force = "enemy",
                    raise_built = false,
                    move_stuck_players = true,
                    create_build_effect_smoke = false,
                    direction = dir,
                }
                if ent ~= nil then
                    ent.minable = false
                    ent.destructible = false
                end
            end
        end
    end
    return true
end

local min = -10.5
local max = 10.5

removeWalls("nauvis", min, min, max, max)
createWalls("nauvis", min, min, max, max)

-- wiring and control of the gate must be performed in a follow up lua script
-- since some delay must occur between when the walls are placed and
-- when the connections and control can be applied.  see gates2.lua

