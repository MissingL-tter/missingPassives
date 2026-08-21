-- One-time generator (Go-maintained afterwards): emits
--   data/mapmods_gen.go   from Data/ModMap.lua (apply closures become
--                         UnportedFn markers; port with the config module)
--   data/timeless_gen.go  from Data/TimelessJewelData/{NodeIndexMapping,
--                         AbyssNotableNames,LegionTradeIds}.lua
--
-- Run from .archive/src/:  luajit ../../tools/gen_datatables.lua
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")
local quote = canon.quote

local function goNum(n)
	if n == math.floor(n) and n < 1e15 and n > -1e15 then
		return string.format("%d", n)
	end
	return string.format("%.17g", n)
end

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
	return keys
end

-- goExpr: generic value emitter (no mods in these tables).
local function goExpr(v)
	local t = type(v)
	if v == nil then
		return "nil"
	elseif t == "boolean" then
		return tostring(v)
	elseif t == "number" then
		return "float64(" .. goNum(v) .. ")"
	elseif t == "string" then
		return quote(v)
	elseif t == "function" then
		return "UnportedFn{}"
	elseif t ~= "table" then
		error("unsupported type " .. t)
	end
	local nArr = 0
	for _ in ipairs(v) do
		nArr = nArr + 1
	end
	local hashKeys = {}
	for k in pairs(v) do
		if not (type(k) == "number" and k == math.floor(k) and k >= 1 and k <= nArr) then
			hashKeys[#hashKeys + 1] = k
		end
	end
	table.sort(hashKeys, function(a, b) return tostring(a) < tostring(b) end)
	if #hashKeys == 0 then
		local parts = {}
		for _, e in ipairs(v) do
			parts[#parts + 1] = goExpr(e)
		end
		return "[]any{" .. table.concat(parts, ", ") .. "}"
	end
	if nArr == 0 then
		local parts = {}
		for _, k in ipairs(hashKeys) do
			parts[#parts + 1] = quote(tostring(k)) .. ": " .. goExpr(v[k])
		end
		return "map[string]any{" .. table.concat(parts, ", ") .. "}"
	end
	local ap, hp = {}, {}
	for _, e in ipairs(v) do
		ap[#ap + 1] = goExpr(e)
	end
	for _, k in ipairs(hashKeys) do
		hp[#hp + 1] = quote(tostring(k)) .. ": " .. goExpr(v[k])
	end
	return "&modparser.D{Arr: []any{" .. table.concat(ap, ", ") .. "}, KV: map[string]any{" .. table.concat(hp, ", ") .. "}}"
end

local function emitVar(f, name, v)
	f:write("var ", name, " = ")
	if type(v) ~= "table" then
		error("expected table for " .. name)
	end
	-- top level as map[string]any (mixed and numeric keys stringified)
	f:write("map[string]any{\n")
	for _, k in ipairs(sortedKeys(v)) do
		f:write("\t", quote(tostring(k)), ": ", goExpr(v[k]), ",\n")
	end
	f:write("}\n\n")
end

do
	local f = assert(io.open("../../data/mapmods_gen.go", "w"))
	f:write("// Code generated from Data/ModMap.lua (one-time transform, Go-\n")
	f:write("// maintained): the map modifier configuration data. The apply\n")
	f:write("// closures are UnportedFn markers, ported with the config module.\n\npackage data\n\n")
	emitVar(f, "mapModsTable", data.mapMods)
	f:close()
end

do
	local f = assert(io.open("../../data/timeless_gen.go", "w"))
	f:write("// Code generated from Data/TimelessJewelData/{NodeIndexMapping,\n")
	f:write("// AbyssNotableNames,LegionTradeIds}.lua (one-time transform,\n")
	f:write("// Go-maintained).\n\npackage data\n\n")
	f:write("import \"github.com/MissingL-tter/missingPassives/modparser\"\n\nvar _ = modparser.D{}\n\n")
	emitVar(f, "nodeIDListTable", data.nodeIDList)
	emitVar(f, "abyssNotableNamesTable", data.abyssNotableNames)
	emitVar(f, "timelessJewelTradeIDsTable", data.timelessJewelTradeIDs)
	f:close()
end

print("generated mapmods_gen.go and timeless_gen.go")
