local s = game.surfaces["nauvis"]

-- demonstrates how character entities can be created and then have them launch
-- robot capsules. this demo creates 2 each of enemy and player forces and then
-- they launch each of the three types of robot capsules

-- this must be run at least once, otherwise follower robot count limits the show
--game.forces["player"].research_all_technologies()
--game.forces["enemy"].research_all_technologies()

local to_cleanup = s.find_entities_filtered {
    name = {
        "character-corpse",
        "medium-scorchmark-tintable",
        "destroyer-remnants",
        "defender-remnants",
        "distractor-remnants",
    }
}
for _, v in pairs(to_cleanup) do
    v.destroy()
end

-- cleanup any active entities on the board of force enemy
to_cleanup = s.find_entities_filtered {
    name = {
        "character",
        "destroyer",
        "defender",
        "distractor" }
}
for _, v in pairs(to_cleanup) do
    if v.name == "character" then
        local isPlayer = false
        for _, p in pairs(game.players) do
            if p.character == v then
                isPlayer = true
            end
        end
        if not isPlayer then
            v.destroy()
        end
    else
        v.destroy()
    end
end


local ironMan = function(position, force, capsule_type, amount)
    local the_man = s.create_entity {
        name = "character", force = force, position = position
    }
    if the_man ~= nil then
        the_man.destructible = true
    end
    if force == "enemy" then
        the_man.color = { 0, 0, 0 }
    end
    local range = 10
    for i = 1, amount do
        local rad = (i / amount + (game.tick % 360) / 360) * 2 * math.pi
        local x_offset = range * math.cos(rad)
        local y_offset = range * math.sin(rad)
        s.create_entity {
            name = capsule_type,
            --quality = "legendary",
            position = the_man.position,
            force = force,
            raise_built = true,
            create_build_effect_smoke = false,
            source = the_man,
            target = { the_man.position.x + x_offset, the_man.position.y + y_offset },
        }
    end
end

-- stores the last robot type in game storage so subsequent runs
-- can use a different type
local robot_type = storage["iron_man_robot_type"]
if robot_type == nil then
    robot_type = "distractor-capsule"
    storage["iron_man_robot_type"] = robot_type
else
    if robot_type == "distractor-capsule" then
        robot_type = "destroyer-capsule"
        storage["iron_man_robot_type"] = robot_type
    else
        if robot_type == "destroyer-capsule" then
            robot_type = "defender-capsule"
            storage["iron_man_robot_type"] = robot_type
        else
            if robot_type == "defender-capsule" then
                robot_type = "distractor-capsule"
                storage["iron_man_robot_type"] = robot_type
            end
        end
    end
end

local yPos = -50
local xPos = 14
local offset = 10
local player_count = 12
local enemy_count = player_count - 2

ironMan({ xPos, yPos }, "player", robot_type, player_count)
ironMan({ xPos, yPos + offset }, "enemy", robot_type, enemy_count)
ironMan({ xPos + offset, yPos }, "enemy", robot_type, enemy_count)
ironMan({ xPos + offset, yPos + offset }, "player", robot_type, player_count)

