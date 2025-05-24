-- an example script that creates entities for any blueprinted ghost on a surface
local surface = game.get_surface(1)
local ghosts = surface.find_entities_filtered { type = "entity-ghost" }
for _, ghost in pairs(ghosts) do
    ghost.revive()
end
