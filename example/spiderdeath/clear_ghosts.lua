example_spider = {}

example_spider.clear = function()
    local surface = game.surfaces["nauvis"]
    local ghosts = surface.find_entities_filtered { type = "entity-ghost" }
    for _, ghost in pairs(ghosts) do
        if ghost.ghost_name == "spidertron" then
            ghost.destroy()
        end
    end
end

