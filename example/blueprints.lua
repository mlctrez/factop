local s = game.surfaces[1]

local destroy_items = function()
    local panels = s.find_entities_filtered({
        area = { { -1000, -1000 }, { 1000, 1000 } }, name = { "steel-chest", "stone-wall", "gate" }, force = { "player", "enemy" }
    })
    for _, p in pairs(panels) do
        p.destroy()
    end
end
destroy_items()

-- example of how to place a blueprint into a player's cursor,
-- this could be considered rude depending on what the player is currently doing
if false then
    game.players[1].cursor_stack.import_stack([[
0eNptj92Kg0AMhV8l5HoqdK2C8xy9W0oZa9oGxihm3K2I777R7XZvCoGQv3O+zFjHkfqBJaGfkS+dKPrPGZVvEuLak9ASemxY+ximXR+EIi4OWRp6oN8vJ4ckiRPT7+lWTGcZ25oGW3DvJRz2ndpVJ6uLKe3yfVY4nNCXWWEGiR4Ghcc7K1gEaEk13AjSPSSYuhG+OUa4GgiwKDfrhODpApsLvN7LcAXlRK1p/j/tMIbaaF505z+6Lxp0gyvKj+pQVZbyKi/zZfkB+XZoLQ==
]])
end

local lock_entity = function(entity)
    entity.minable = false
    entity.destructible = false
    entity.operable = false
end

local chest = s.create_entity({
    name = "steel-chest", position = { 0, 0 }, force = "player",
    move_stuck_players = true, create_build_effect_smoke = false,
})
lock_entity(chest)

local some_walls = "0eNqVnNFuGkkQRf9lnmE1XT3dPcOvrKIVjkcWEh4swNmNIv59AUci2fVR1X1yrMDxJXM9PlS186N72r/Pb8fdcu42P7rTsn1bnw/rl+Pu+fb5P90m26r7fvtwWXXbp9Nh/36e17fHve2Wl25zPr7Pq+7tcNqdd4dlfZz32/Pu2/wfSKp3SKpXyO7rYTl1mz+vX233smz3t4cs29e523Sn82GZ139v9/vu9sDleb499/Jl1c3L+foF5o/n3T/5/tfy/vo0H68PWH3y/EemnxHW/R/lHuL2h8tl9T+MhTDJoeQQxcsyhCjZoZQQxRxKDVGKQ2khyuBQxhClOZQpRKkOJfUhzORhYvUdPUywvl5/U6zAyWtwilU4eR1OsRInr8XpUeOX7Xn+hPBbg1fd8+44f/34++EzXqzQyWt0Gr1cVcsVK3fyvkcs1u7k1dti9U5evy3Wb/P6bbF+m/vTItZv8/ptwZu012+L3abNu09brNfm9dpit2pzexjrs3l36xzrs3l9zrE+m9fnHOtz9nqYY33OrrcElcPrYS6ajSXAVA1DaZqkYxRmlCiUZZIEk8ywl9SQKElSQ6KYpIZEyZIaEmWQ1JAoRVJDolTJDInSJDEkyqh5IWEmTQvpHUmvWSFhkiaFhLGoE4bUq2RNCSnVEDXCWKqiCSGlqpoPEqZpOkiYUbNBwkyaDNI75F5zQcIkTQUJY5oJEiZrIkiYQfNAwhRNAwlTNQskTNMkkDCj5m6gJ3XSVJImNL2WhjBJE0l4Uc00DKXJmkcaYAYNkwFTJJGkMFWiUJamXW4KI3aY0kza5YY0Y69hIM0ojpQHwJiGKYDRZsoUZpAolKVol5vCVA1DaZp2uSnNqGEozaRd7grT6V7DNMAk6XpTGJMolEUcUlCYQcNQmqJdbkpTNQyladrlHgEzapgJMJN0vUdajfQSZiJM0i44xjGNg3mydskxjzhswzzqsA1XWeK4LSUCiQM3DKSN3DjPJCo2bulEO8ZESfVjTCQaMicSHTkZgURLTplAmidzIM2UOY/oyhxIfceHiURfxkQmGjMm+mXR9/nY6nHZCyFEX04DgbKTxY+i2TInEX2ZQd7++oHA1yTaMmfxdtYPBGaZwm2pdFBB9OTUCJSibcEomiVzEtGTGTSE24KvSbRkzlLDbcEsoiMnEq8sWnIi88qaJ2OgQRNlzDOIpsyBRFXmRKIrcyJRljmRaMuGR5tEWzYyr0GzZQ6k2TLnEW0ZAxXRljFREW2ZE4m2zInUiTKZVxFt2ci8ijhVxkCaLXMedbKMgURb5kTqdJkSVdGWMVEVJ8xGLlZFZzYysqpNmTmQZs6cRzRnDiTOmjmR6M+cSJw3cyJx4mwkWE10aSPZa9rUmQNpRs15RKPmQOLsmROJXs2JxPkzJxLt2sjUmmjXRqbWNLvGQKNm15hnFO2aA4l2zYlEu+ZEol1zItGuMx6IF+06k6kFt4PJ5Wh2zRyt1fgPFFwQmpcnuCHMLsekA5zMydJxUuYM0iFO5hTpSClzqnSQkzlNOlbKnFE7y8mgSTtbSiALrguTuaCknS9lUPho6B3hnsK0XjwcysnCx0OjycQDopysamc7GdS0s6YMGrXznQyatPOmCAquEc1tenCNaG7Tg2tEc4sZXCNacUGDdtaTQUU7e8qgqp33ZFDTzp8ySD12hyBx5Jfx9/N67RQqJjJx5McgceTHL00c+WUjkDYbYU7RzBY5VTNJ5DTNJJEzaiaJnEkzSeIEV4zV5STNJJFjmkkiJ2smiZxBNEkEFdEkEVRFk0RQE00SQWPYJC3ka9HfI3S/S37ZPXomGUsWXEImt+fBJWRyix5cQia36cElpLlNDy4hzW16cAlpbtODS0hzmx7cQppfzEk0SQIF15Dm3sCDa0hzmx1cQ5rb7OAa0txmB9eQ2W12ERc2DBIH2x+gL6tud55fr096/I9Gq+7bfDzdn1KqTcM0XT/kKdd8ufwLMZAuSg=="

-- using a steel-chest to import a blueprint and use it then destroy the chest quickly afterwards
local inv = chest.get_inventory(defines.inventory.item_main)
inv.find_empty_stack().import_stack(some_walls)
local blueprint = inv.find_item_stack("blueprint")
for x = -1, 1 do
    for y = -1, 1 do
        local items = blueprint.build_blueprint {
            surface = "nauvis", position = { x * 32, y * 32 },
            force = "player", build_mode = defines.build_mode.superforced
        }
        for _, item in pairs(items) do
            item.revive()
        end
    end
end

chest.destroy()