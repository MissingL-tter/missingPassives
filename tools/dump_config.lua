-- Dumps one config-tab modifier build per option value, so the port's
-- apply functions can be compared against the reference for options no
-- corpus build sets. The corpus reaches about 32 of the 580; this reaches
-- all of them.
--
--   cd .archive/src && luajit ../../tools/dump_config.lua
--
-- Writes test/testdata/config_options.jsonl:
--   <case>.value       the value the option was set to (replay input)
--   <case>.mods        configTab.modList after BuildModList
--   <case>.enemyMods   configTab.enemyModList
--   <case>.input       the input table it left behind
--   <case>.placeholder the placeholder table it left behind
--
-- A case is "<var>#<n>". Each starts from a freshly defaulted config set,
-- so only the named option differs from a new build.
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

-- The option appliers walk a few data tables with pairs() -- the boss
-- preset's damage multipliers, penetrations and additional stats -- and
-- the order reaches the emitted mod lists. tools/dump_build.lua settles
-- the same question by installing a sorted pairs before the modules load;
-- do the same here so one convention covers both dumps and the port can
-- iterate in sorted order.
local rawpairs = pairs
local function sortedPairs(t)
	local num, str, other = {}, {}, {}
	for k in rawpairs(t) do
		if type(k) == "number" then
			num[#num + 1] = k
		elseif type(k) == "string" then
			str[#str + 1] = k
		else
			other[#other + 1] = k
		end
	end
	table.sort(num)
	table.sort(str)
	local keys = {}
	for _, k in ipairs(num) do keys[#keys + 1] = k end
	for _, k in ipairs(str) do keys[#keys + 1] = k end
	for _, k in ipairs(other) do keys[#keys + 1] = k end
	local i = 0
	return function()
		i = i + 1
		if keys[i] ~= nil then
			return keys[i], t[keys[i]]
		end
	end
end
pairs = sortedPairs

local canon = dofile("../../tools/canon.lua")
local varList = LoadModule("Modules/ConfigOptions")

local out = assert(io.open("../../test/testdata/config_options.jsonl", "w"))
local function emit(name, value)
	out:write('{"k":', canon.quote(name), ',"c":', canon.quote(canon.encode(value)), "}\n")
end
local function emitExact(name, value)
	out:write('{"k":', canon.quote(name), ',"c":', canon.quote(canon.encodeExact(value)), "}\n")
end
local function modArray(list)
	local o = {}
	for i, mod in ipairs(list or {}) do
		o[i] = mod
	end
	return o
end

newBuild()
local configTab = build.configTab

-- The values each option type is exercised at. Numeric options get a
-- small value, one past the caps several appliers impose, and (where zero
-- is meaningful) zero.
local function valuesFor(varData)
	if varData.type == "check" then
		return { true }
	elseif varData.type == "list" then
		local vals = {}
		for i, entry in ipairs(varData.list or {}) do
			vals[i] = entry.val
		end
		return vals
	elseif varData.type == "countAllowZero" then
		return { 0, 1, 7, 25 }
	else -- count, integer, float
		return { 1, 7, 25 }
	end
end

local cases, failures = 0, 0
for _, varData in ipairs(varList) do
	if varData.var then
		for i, value in ipairs(valuesFor(varData)) do
			local case = varData.var .. "#" .. i
			-- Fresh config set: only this option differs from a new build.
			configTab:NewConfigSet(1, "Default")
			configTab:SetActiveConfigSet(1, true)
			-- Control state lives on the widgets and survives a rebuild in
			-- the application; reset the one an option writes so each case
			-- starts where a freshly loaded build does.
			if configTab.varControls['enemyDamageType'] then
				configTab.varControls['enemyDamageType'].enabled = true
			end
			configTab.input[varData.var] = value
			local ok, err = pcall(function() configTab:BuildModList() end)
			if not ok then
				emit(case .. ".error", tostring(err))
				failures = failures + 1
			else
				emitExact(case .. ".value", { var = varData.var, value = value })
				emit(case .. ".mods", modArray(configTab.modList))
				emit(case .. ".enemyMods", modArray(configTab.enemyModList))
				emit(case .. ".input", configTab.input)
				emit(case .. ".placeholder", configTab.placeholder)
				cases = cases + 1
			end
		end
	end
end
out:close()
io.stderr:write(string.format("config option dump: %d cases, %d errored\n", cases, failures))
