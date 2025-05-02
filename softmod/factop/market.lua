factop_market = {}

factop_market.summon = function(player)

    for _, c in pairs(player.surface.find_entities_filtered { name = "market", player.force }) do
        player.print("market here -> " .. c.gps_tag)
        return
    end

    -- find a non colliding place for market to appear
    local appear_at = player.surface.find_non_colliding_position("market", player.position, 10, 1, false)
    if appear_at == nil then
        player.print("unable to place market, move to a more open area")
        return
    end
    local market = player.surface.create_entity {
        name = "market", force = player.force, position = appear_at
    }
    market.color = { 255, 255, 255, 255 } -- cannot color the market?
    market.operable = true -- false for when the market is closed?
    market.destructible = false -- enemies and players cannot destroy
    -- todo: right now this is just a fixed market that gives nothing for 1 iron-ore
    -- todo: the gameplay could modify the market offers or close/open the market
    -- https://lua-api.factorio.com/latest/concepts/Offer.html
    -- https://lua-api.factorio.com/latest/classes/LuaEntity.html#add_market_item
    -- https://lua-api.factorio.com/latest/classes/LuaEntity.html#remove_market_item
    -- https://lua-api.factorio.com/latest/classes/LuaEntity.html#get_market_items
    -- https://lua-api.factorio.com/latest/classes/LuaEntity.html#get_market_items
    -- https://lua-api.factorio.com/latest/classes/LuaEntity.html#clear_market_items

    market.add_market_item {
        price = { { name = "iron-ore", count = 1 } },
        offer = { type = "nothing" } }
end

-- handles console commands for the market, right now it just summons one
-- The structure of ccd defined at
-- https://lua-api.factorio.com/latest/concepts/CustomCommandData.html
factop_market.command = function(ccd)
    factop_market.summon(game.players[ccd.player_index])
end

factop_market.add_commands = function()
    commands.add_command("market", "creates or shows location of the market", factop_market.command)
end

-- https://lua-api.factorio.com/latest/events.html#on_market_item_purchased
factop_market.item_purchased = function(event)
    -- the market has already provided the item, this would be a way to log
    -- for statistics or modify other game play
    game.players[event.player_index].print("thanks for your purchase")
end

factop_market.events = {
    [defines.events.on_market_item_purchased] = factop_market.item_purchased,
}

return factop_market