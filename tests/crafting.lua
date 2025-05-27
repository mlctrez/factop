-- crafting.lua
local testBase = function()
    rcon.print("testBase")
end

local testError = function()
    error("testError")
end

local testCleanup = function(surface)
    local bounds = 200
    local area = { { -bounds, -bounds }, { bounds, bounds } }
    for _, f in pairs(surface.find_entities_filtered { area = area }) do
        if f.name ~= "character" then
            f.destroy()
        end
    end
end

local testGhost = function()
    local s = game.surfaces[1]
    testCleanup(s)

    local chest = s.create_entity {
        position = { 1.5, 1.5 }, name = "steel-chest",
        force = "player", create_build_effect_smoke = false }
    if chest == nil then
        error("unable to create conflict chest")
    end

    local m = s.create_entity {
        position = { 0.0, 0.0 }, name = "entity-ghost", force = "player",
        inner_name = "assembling-machine-3" }
    if m == nil then
        error("assembling machine not created")
    end
    -- verify that snap to grid worked
    if m.position.x ~= 0.5 or m.position.y ~= 0.5 then
        error("assembling machine unexpected position : " .. serpent.line(m.position))
    end
    if not m.set_recipe("steel-chest", "legendary") then
        error("set ghost recipe failed")
    end
    if m.get_recipe() == nil then
        error("get ghost recipe failed")
    end

    local collides, ne
    collides, ne = m.silent_revive({ return_item_request_proxy = true })
    if collides ~= nil then
        error("revive one collides was nil")
    end
    if ne ~= nil then
        error("expected nil proxy")
    end
    if not chest.destroy() then
        error("unable to delete conflict chest")
    end

    collides, ne = m.silent_revive({ return_item_request_proxy = true })
    if collides == nil then
        error("expected collides to be nil")
    end
    if ne == nil then
        error("proxy was nil")
    end
    if ne.name ~= "assembling-machine-3" then
        error("ghost not revived")
    end
    if not ne.die() then
        error("unable to die revived machine")
    end
    local found = s.find_entities_filtered { name = "entity-ghost", position = { 0.5, 0.5 } }
    if #found ~= 1 then
        error("unable to find ghost after kill")
    end
    _, ne = found[1].revive({ return_item_request_proxy = true })
    if ne == nil then
        error("unable to revive ghost")
    end
    if not ne.destroy() then
        error("unable to destroy revived machine ")
    end


end
