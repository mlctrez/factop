local udp_mod = {}

udp_mod.periodic = function()
    helpers.recv_udp(0)
end

udp_mod.on_nth_tick = {
    [20] = udp_mod.periodic
}

return udp_mod
