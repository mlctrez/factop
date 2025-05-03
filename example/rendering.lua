for _, ren in ipairs(rendering.get_all_objects()) do
    ren.destroy()
end

-- example script for placing renderings on a surface

local surface = game.surfaces.nauvis
local start_y = 2.45
local draw_text_line = function(message)
    if message == "" then
        start_y = start_y + .5
        return
    end
    if message == "skip" then
        start_y = start_y + 3.5
        return
    end
    rendering.draw_text {
        text = message,
        surface = surface,
        target = { x = 0.05, y = start_y },
        color = { r = .7, g = .5 },
        scale = 3,
        alignment = "center",
        vertical_alignment = "middle",
        draw_on_ground = true,
    }
    start_y = start_y + 1.2
end

draw_text_line("/factop")

rendering.draw_sprite {
    surface = surface,
    target = { x = 0.22, y = -4 },
    sprite = "file/img/factorio.png",
    render_layer = 3,
}

rendering.draw_light {
    surface = surface,
    target = { x = 0, y = 0 },
    sprite = "utility/light_medium",
    color = { r = 1, g = 1, b = 1 },
    intensity = .8,
    scale = 10,
}

