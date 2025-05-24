factop_wall = {}

factop_wall.create_barrier = function(s, x, y)
    local wall = game.surfaces[s].create_entity({
        name = "stone-wall", position = { x, y }, force = "enemy",
        create_build_effect_smoke = false, snap_to_grid = true, move_stuck_players = true
    })
    if wall ~= nil then
        wall.destructible = false
    end
end

-- https://lua-api.factorio.com/latest/concepts/BoundingBox.html
-- Explicit definition
-- {left_top = {x = -2, y = -3}, right_bottom = {x = 5, y = 8}}
factop_wall.create_walls = function(s, force, bounding_box)
    local left_x = bounding_box.left_top.x
    local right_x = bounding_box.right_bottom.x
    local top_y = bounding_box.left_top.y
    local bottom_y = bounding_box.right_bottom.y

    local surface = game.surfaces[s]
    if surface == nil then
        error("no such game surface " .. s)
    end
    for x = left_x, right_x do
        for y = top_y, bottom_y do
            local wall = surface.create_entity({
                name = "stone-wall", position = { x, y }, force = force,
                create_build_effect_smoke = false, snap_to_grid = true, move_stuck_players = true
            })
            if wall ~= nil then
                wall.destructible = false
                wall.minable = false
            end
        end
    end
end

factop_wall.wall_line = function(s, force, start_pos, end_pos, width)
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

    for w = -width, width, 0.2 do
        for t = 0, length, 0.2 do
            local x = 0.5 + start_pos.x + t * dx + w * px
            local y = 0.5 + start_pos.y + t * dy + w * py
            local wall = surface.create_entity({
                name = "stone-wall", position = { x, y }, force = force,
                create_build_effect_smoke = false,
                snap_to_grid = true,
                move_stuck_players = true
            })
            if wall ~= nil then
                wall.destructible = false
                wall.minable = false
            end
        end
    end
end