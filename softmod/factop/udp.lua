factop_udp = {}

-- https://lua-api.factorio.com/latest/events.html#on_udp_packet_received
factop_udp.packet_received = function(event)
    --print("factop_udp.packet_received " .. event.payload)
end

factop_udp.event_handler = function(event)
    --print("factop_udp.event_handler " .. event.name)
    --helpers.send_udp(4000, "Hello world", 0)
end

factop_udp.periodic = function()
    -- https://lua-api.factorio.com/latest/classes/LuaHelpers.html#recv_udp
    helpers.recv_udp(0)
end

factop_udp.events = {
    [defines.events.on_udp_packet_received] = factop_udp.packet_received,
    [defines.events.on_player_joined_game] = factop_udp.event_handler,
    [defines.events.on_player_changed_position] = factop_udp.event_handler,
}

factop_udp.on_nth_tick = {
    [20] = factop_udp.periodic
}

return factop_udp