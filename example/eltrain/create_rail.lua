local surface = game.get_surface(1)
local canDie = false

-- this creates a stop sign shaped train rail around {0,0}
-- don't ask how long this took to figure out

local ent = nil
local pos = { x = -30, y = 6 }
for i = 1, 8 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false
    })
    ent.destructible = canDie
    pos.y = pos.y - 2
end

ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northeast
})
ent.destructible = canDie

pos.y = pos.y - 6
pos.x = pos.x + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northeast
})
ent.destructible = canDie

pos.y = pos.y - 2
pos.x = pos.x + 4

for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.northeast
    })
    ent.destructible = canDie
    pos.y = pos.y - 2
    pos.x = pos.x + 2
end

pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.west
})
ent.destructible = canDie
pos.x = pos.x + 6
pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.west
})
ent.destructible = canDie

pos.x = pos.x + 2
for i = 1, 8 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.west
    })
    ent.destructible = canDie
    pos.x = pos.x + 2
end
pos.x = pos.x + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southeast
})
ent.destructible = canDie
pos.x = pos.x + 4
pos.y = pos.y + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southeast
})
ent.destructible = canDie

pos.x = pos.x + 4
pos.y = pos.y + 4
for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.northwest
    })
    ent.destructible = canDie
    pos.x = pos.x + 2
    pos.y = pos.y + 2
end

ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.north
})
ent.destructible = canDie

pos.x = pos.x + 2
pos.y = pos.y + 6
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.north
})
ent.destructible = canDie

pos.y = pos.y + 2
for i = 1, 8 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.north
    })
    ent.destructible = canDie
    pos.y = pos.y + 2
end

pos.y = pos.y + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southwest
})
ent.destructible = canDie
pos.x = pos.x - 2
pos.y = pos.y + 4
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southwest
})
ent.destructible = canDie

pos.x = pos.x - 8
pos.y = pos.y + 10
for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.southwest
    })
    ent.destructible = canDie
    pos.y = pos.y - 2
    pos.x = pos.x + 2
end

pos.x = pos.x - 12
pos.y = pos.y + 10
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.east
})
ent.destructible = canDie

pos.x = pos.x - 4
pos.y = pos.y + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.east
})
ent.destructible = canDie

pos.x = pos.x - 4
for i = 1, 8 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.west
    })
    ent.destructible = canDie
    pos.x = pos.x - 2
end

ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northwest
})
ent.destructible = canDie

pos.x = pos.x - 6
pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northwest
})
ent.destructible = canDie

pos.x = pos.x - 2
pos.y = pos.y - 2
for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.northwest
    })
    ent.destructible = canDie
    pos.x = pos.x - 2
    pos.y = pos.y - 2
end

pos.x = pos.x - 2
pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.south
})
ent.destructible = canDie
pos.x = pos.x - 2
pos.y = pos.y - 4
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.south
})
ent.destructible = canDie

