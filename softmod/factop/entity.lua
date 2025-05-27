factop_entity = {}

factop_entity.destroy_ghosts = function(params)
    if params == nil then
        error("no parameters provided")
    end
    if params.area == "surface" then
        params.area = nil
    else
        if params.area == nil then
            local m = 500
            params.area = { left_top = { x = -m, y = -m },
                            right_bottom = { x = m, y = m } }
        end
    end

    local surface = nil
    if params.surface ~= nil then
        surface = game.surfaces[params.surface]
        if surface == nil then
            error("surface \"" .. params.surface .. "\" not found")
        end
    end
    if surface == nil then
        surface = game.surfaces[1]
    end
    if params.name == nil then
        error("name parameter required")
    end

    local ghosts = surface.find_entities_filtered {
        type = "entity-ghost",
        area = params.area,
    }
    local destroy_count = 0
    for _, ghost in pairs(ghosts) do
        if ghost.ghost_name == params.name then
            ghost.destroy()
            destroy_count = destroy_count + 1
        end
    end

    if params.debug ~= nil then
        params.surface = surface.name
        params.destroy_count = destroy_count
        rcon.print(helpers.table_to_json(params))
    end
end