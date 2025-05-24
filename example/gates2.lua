local surface = game.surfaces["nauvis"]

local wallLeft = surface.find_entity("stone-wall", { -2.5, -10.5 })
local wallRight = surface.find_entity("stone-wall", { 2.5, -10.5 })

if wallLeft == nil or wallRight == nil then
    error("unable to find walls")
end

local a = wallLeft.get_wire_connector(defines.wire_connector_id.circuit_red, true)
local b = wallRight.get_wire_connector(defines.wire_connector_id.circuit_red, true)
if a == nil and b == nil then
    error("unable to find wire connectors")
end
if not a.is_connected_to(b, defines.wire_origin.script) then
    print("connecting wire")
    a.connect_to(b, false, defines.wire_origin.script)
else
    print("wire already connected")
end

-- https://lua-api.factorio.com/latest/classes/LuaWallControlBehavior.html
local wallControl = wallLeft.get_or_create_control_behavior()
if wallControl == nil then
    error("unable to get control behavior")
end

-- how to toggle
local condition = { first_signal = { type = "virtual", name = "signal-unlock" }, comparator = ">", constant = 0 }
if wallControl.circuit_condition.fulfilled then
    condition.comparator = ">"
    wallControl.circuit_condition = condition
else
    condition.comparator = "="
    wallControl.circuit_condition = condition
end

--local condition = { first_signal = { type = "item", name = "raw-fish" }, comparator = "=", constant = 0 }
--
--wallLeft.get_or_create_control_behavior().circuit_condition = condition
--wallRight.get_or_create_control_behavior().circuit_condition = condition

