factop_storage = {}

factop_storage.keys = function(item)
    if item == nil then
        item = storage
    end
    local all_keys = {}
    for k, _ in pairs(item) do
        table.insert(all_keys, k)
    end
    return all_keys
end

factop_storage.get = function(tableKey, key)
    if tableKey == nil or key == nil then
        error("factop_storage.get - tableKey and key required")
    end
    if storage[tableKey] == nil then
        storage[tableKey] = {}
    end
    return storage[tableKey][key]
end

factop_storage.put = function(tableKey, key, value)
    if tableKey == nil or key == nil then
        error("factop_storage.put - tableKey and key required")
    end
    if storage[tableKey] == nil then
        storage[tableKey] = {}
    end
    storage[tableKey][key] = value
end