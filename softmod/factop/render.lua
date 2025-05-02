factop_render = {}

factop_render.all_objects = function()
    local all = {}
    for _, ren in ipairs(rendering.get_all_objects()) do
        table.insert(all, ren.id)
    end
    rcon.print(helpers.table_to_json({ all_objects = all }))
end

factop_render.sprite = function(surface, x, y, sprite)
    local ren = rendering.draw_sprite {
        surface = surface,
        target = { x, y },
        sprite = sprite,
        render_layer = 3,
    }
    rcon.print(helpers.table_to_json({ id = ren.id }))
end

factop_render.move = function(id, x, y)
    local ren = rendering.get_object_by_id(id)
    if ren ~= nil then
        ren.target = { x, y }
        rcon.print(helpers.table_to_json({ target = { x, y } }))
    else
        rcon.print(helpers.table_to_json({ error = "rendering object not found" }))
    end
end
