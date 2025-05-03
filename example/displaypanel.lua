local s = game.surfaces[1]

-- display-panel does not require power, so it might be a useful way to show
-- the player a message.. this example shows how to create two display panels
-- with text that are indestructible

local panels = s.find_entities_filtered({
    area = { { -10, -10 }, { 10, 10 } }, name = "display-panel", force = "player"
})
for _, panel in ipairs(panels) do
    panel.destroy()
end

local lock_entity = function(entity)
    entity.minable = false
    entity.destructible = false
    entity.operable = false
end

local horizontal = 3
local vertical = 2.65

-- https://lua-api.factorio.com/latest/prototypes/DisplayPanelPrototype.html

local condition = { first_signal = { type = "item", name = "raw-fish" }, comparator = "=", constant = 0 }

local panelLeft = s.create_entity({
    name = "display-panel",
    quality = "legendary",
    position = { -horizontal, vertical },
    force = "player",
    snap_to_grid = false,
    create_build_effect_smoke = false,
    move_stuck_players = true,
})

lock_entity(panelLeft)
panelLeft.get_or_create_control_behavior().set_message(1, {
    icon = { type = "virtual", name = "right-arrow" },
    text = "This is a puzzle you",
    condition = condition
})
--panelLeft.always_show_in_alt_mode = true
local panelRight = s.create_entity({
    name = "display-panel",
    quality = "legendary",
    position = { horizontal, vertical },
    force = "player",
    snap_to_grid = false,
    create_build_effect_smoke = false,
    move_stuck_players = true,
})
lock_entity(panelRight)
panelRight.get_or_create_control_behavior().set_message(1, {
    icon = { type = "virtual", name = "left-arrow" },
    text = "will need to solve.",
    condition = condition
})

--- workaround: since the display panel doesn't show anything,
--- connect it to a circuit network, and disconnect it immediately after,
--- to give it an update
local a = panelLeft.get_wire_connector(defines.wire_connector_id.circuit_red, true)
local b = panelRight.get_wire_connector(defines.wire_connector_id.circuit_red, false)
a.connect_to(b, false, defines.wire_origin.script)
a.disconnect_from(b)

