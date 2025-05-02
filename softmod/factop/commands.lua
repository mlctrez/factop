factop_commands = {}

factop_commands.character = function()
    if game.player and game.player.valid then
        game.player.print("You have been given a new character!")
    end
    if game.player.character then
        game.player.character.destroy()
    end
    game.player.character = game.player.surface.create_entity {
        name = "character", position = game.player.position, force = game.player.force
    }
end

factop_commands.add_commands = function()
    commands.add_command(
            "factop_character",
            "\n recreates your character. your inventory, armor, weapons, and ammo will be lost",
            factop_commands.character
    )
end

return factop_commands