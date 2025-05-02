local handler = require("event_handler")

local add_lib = function(lib_name)
    print("softmod loading " .. lib_name)
    local l = require(lib_name)
    if type(l) == "table" then
        handler.add_lib(l)
    end
end

-- softmod.go will add one line for each lua file
