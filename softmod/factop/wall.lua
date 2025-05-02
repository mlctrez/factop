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