local s = game.surfaces["nauvis"]

-- an example script to equip a player with some armor, weapons, ammo, and destroyer bots

local player = game.players[1]
if player.connected then
    player.force.research_all_technologies()

    if player.character == nil then
        player.character = s.create_entity { name = "character", position = { 0, 0 }, force = "player" }
    end

    local char = player.character
    local guns = char.get_inventory(defines.inventory.character_guns)
    local ammo = char.get_inventory(defines.inventory.character_ammo)

    guns[1].set_stack { name = "submachine-gun", quality = "legendary" }
    ammo[1].set_stack { name = "uranium-rounds-magazine", quality = "legendary", count = 100 }

    guns[2].set_stack { name = "rocket-launcher", quality = "legendary" }
    ammo[2].set_stack { name = "explosive-rocket", quality = "legendary", count = 100 }

    guns[3].set_stack { name = "flamethrower", quality = "legendary" }
    ammo[3].set_stack { name = "flamethrower-ammo", quality = "legendary", count = 100 }

    local main_inventory = char.get_inventory(defines.inventory.character_main)

    local equip_inventory = function(name, count, q_pos)
        local insert_count = count - main_inventory.get_item_count({ name = name, quality = "legendary" })
        if insert_count > 0 then
            main_inventory.insert { name = name, count = insert_count, quality = "legendary" }
        end
        player.set_quick_bar_slot(q_pos, { name = name, quality = "legendary" })
    end
    equip_inventory("poison-capsule", 200, 1)
    equip_inventory("slowdown-capsule", 200, 2)
    equip_inventory("defender-capsule", 200, 3)
    equip_inventory("destroyer-capsule", 200, 4)
    equip_inventory("distractor-capsule", 200, 5)
    equip_inventory("explosive-rocket", 800, 6)
    equip_inventory("repair-pack", 100, 7)

    local ch_armor = char.get_inventory(defines.inventory.character_armor)
    if ch_armor ~= nil then
        local stack = ch_armor[1]
        --stack.set_stack { name = "power-armor-mk2", quality = "legendary" }
        stack.set_stack { name = "mech-armor", quality = "legendary" }
        stack.grid.put({ name = "fusion-reactor-equipment", quality = "legendary" })
        stack.grid.put({ name = "fusion-reactor-equipment", quality = "legendary" })
        stack.grid.put({ name = "fusion-reactor-equipment", quality = "legendary" })
        for _ = 1, 6 do
            stack.grid.put({ name = "battery-mk3-equipment", quality = "legendary" })
        end
        --for _ = 1, 2 do
        --    stack.grid.put({ name = "personal-roboport-mk2-equipment", quality = "legendary" })
        --end
        --stack.grid.put({ name = "night-vision-equipment", quality = "legendary" })
        --stack.grid.put({ name = "belt-immunity-equipment", quality = "legendary" })
        --stack.grid.put({ name = "belt-immunity-equipment", quality = "legendary" })

        for _ = 1, 7 do
            stack.grid.put({ name = "exoskeleton-equipment", quality = "legendary" })
        end
        for _ = 1, 16 do
            stack.grid.put({ name = "energy-shield-mk2-equipment", quality = "legendary" })
            stack.grid.put({ name = "personal-laser-defense-equipment", quality = "legendary" })
        end
        for _, eq in pairs(stack.grid.equipment) do
            if eq.max_shield > 0 then
                eq.shield = eq.max_shield
            end
            eq.energy = eq.max_energy
        end
    end
end

