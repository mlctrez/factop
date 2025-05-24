local surface = game.surfaces[1]

-- this does not charge collectors or do any damage
-- target is where the lightning strikes the ground
-- position is where the lighting starts in the sky
-- if looks best when position is up and to the right
-- left as in this example for target and position
local target = { x = 0, y = 0 }
local position = { x = target.x + 5, y = target.y - 20 }
surface.create_entity {
    name = "lightning", position = position, target = target
}

-- there is a surface property that creates a lightning generator
-- so this call only works on fulgora
-- there might be a way to set that property on other surfaces
local fulgora = game.surfaces["fulgora"]
if fulgora ~= nil then
    fulgora.execute_lightning { name = "lightning", position = { 5, 5 } }
end

rcon.print(fulgora.get_property("magnetic-field"))

