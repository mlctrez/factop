local surface = game.get_surface(1)
local canDie = false

-- this creates a stop sign shaped train rail around {0,0}
-- don't ask how long this took to figure out

local lock_entity = function(entity)
    if entity ~= nil then
        entity.minable = false
        entity.destructible = canDie
    end
end

local ent = nil
local pos = { x = -32, y = 10 }
for i = 1, 9 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false
    })
    lock_entity(ent)
    pos.y = pos.y - 2
end

ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northeast
})
lock_entity(ent)

pos.y = pos.y - 6
pos.x = pos.x + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northeast
})
lock_entity(ent)

pos.y = pos.y - 2
pos.x = pos.x + 4

for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.northeast
    })
    lock_entity(ent)
    pos.y = pos.y - 2
    pos.x = pos.x + 2
end

pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.west
})
lock_entity(ent)
pos.x = pos.x + 6
pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.west
})
lock_entity(ent)

pos.x = pos.x + 2
for i = 1, 10 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.west
    })
    lock_entity(ent)
    pos.x = pos.x + 2
end
pos.x = pos.x + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southeast
})
lock_entity(ent)
pos.x = pos.x + 4
pos.y = pos.y + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southeast
})
lock_entity(ent)

pos.x = pos.x + 4
pos.y = pos.y + 4
for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.northwest
    })
    lock_entity(ent)
    pos.x = pos.x + 2
    pos.y = pos.y + 2
end

ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.north
})
lock_entity(ent)

pos.x = pos.x + 2
pos.y = pos.y + 6
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.north
})
lock_entity(ent)

pos.y = pos.y + 2
for i = 1, 9 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.north
    })
    lock_entity(ent)
    pos.y = pos.y + 2
end

pos.y = pos.y + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southwest
})
lock_entity(ent)
pos.x = pos.x - 2
pos.y = pos.y + 4
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.southwest
})
lock_entity(ent)

pos.x = pos.x - 8
pos.y = pos.y + 10
for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.southwest
    })
    lock_entity(ent)
    pos.y = pos.y - 2
    pos.x = pos.x + 2
end

pos.x = pos.x - 12
pos.y = pos.y + 10
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.east
})
lock_entity(ent)

pos.x = pos.x - 4
pos.y = pos.y + 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.east
})
lock_entity(ent)

pos.x = pos.x - 4
for i = 1, 10 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.west
    })
    lock_entity(ent)
    pos.x = pos.x - 2
end

ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northwest
})
lock_entity(ent)

pos.x = pos.x - 6
pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.northwest
})
lock_entity(ent)

pos.x = pos.x - 2
pos.y = pos.y - 2
for i = 1, 4 do
    ent = surface.create_entity({
        force = "player", name = "elevated-straight-rail", position = pos,
        create_build_effect_smoke = false, direction = defines.direction.northwest
    })
    lock_entity(ent)
    pos.x = pos.x - 2
    pos.y = pos.y - 2
end

pos.x = pos.x - 2
pos.y = pos.y - 2
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-b", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.south
})
lock_entity(ent)
pos.x = pos.x - 2
pos.y = pos.y - 4
ent = surface.create_entity({
    force = "player", name = "elevated-curved-rail-a", position = pos,
    create_build_effect_smoke = false, direction = defines.direction.south
})
lock_entity(ent)

