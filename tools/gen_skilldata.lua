-- One-time generator (Go-maintained afterwards): emits
--   data/skillstatmap_gen.go   from the loaded Data/SkillStatMap.lua
--   data/skills_custom_gen.go  from the hand-written per-skill template
--                              fragments (custom keys + raw statMap)
--
-- Run from .archive/src/:  luajit ../../tools/gen_skilldata.lua
--
-- Functions become data.UnportedFn markers; their bodies live in the
-- Export/Skills templates and are ported by the calc modules as needed.
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path

-- Capture the pristine skillStatMap and the raw statMap key sets BEFORE the
-- boot's lazy copies land: hook is not possible, so instead load only the
-- data environment? The boot is required for LoadModule; instead we snapshot
-- template statMap keys by loading SkillStatMap and the skill files
-- ourselves after boot but reading the generator's own fresh copies.
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")
local quote = canon.quote

local function isMod(v)
	return type(v) == "table" and v.name ~= nil and v.type ~= nil and v.flags ~= nil
end

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t) do
		keys[#keys + 1] = k
	end
	table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
	return keys
end

local function goNum(n)
	if n == math.floor(n) and n < 1e15 and n > -1e15 then
		return string.format("%d", n)
	end
	return string.format("%.17g", n)
end

local goExpr

-- goModArgs emits the makeSkillMod-compatible argument list of a mod table.
local function goMod(v, indent)
	if type(v.flags) ~= "number" or type(v.keywordFlags) ~= "number" then
		-- upstream typo (a tag sits in the flags slot); keep the raw table
		local kv = { quote("name") .. ": " .. quote(v.name), quote("type") .. ": " .. quote(v.type) }
		if v.value ~= nil then
			kv[#kv + 1] = quote("value") .. ": " .. goExpr(v.value, indent)
		end
		kv[#kv + 1] = quote("flags") .. ": " .. goExpr(v.flags, indent)
		kv[#kv + 1] = quote("keywordFlags") .. ": " .. goExpr(v.keywordFlags, indent)
		local arr = {}
		for _, tag in ipairs(v) do
			arr[#arr + 1] = goExpr(tag, indent)
		end
		table.sort(kv)
		local s = "&modparser.D{"
		if #arr > 0 then
			s = s .. "Arr: []any{" .. table.concat(arr, ", ") .. "}, "
		end
		return s .. "KV: map[string]any{" .. table.concat(kv, ", ") .. "}}"
	end
	local parts = { quote(v.name), quote(v.type) }
	if v.value == nil then
		parts[#parts + 1] = "nil"
	else
		parts[#parts + 1] = goExpr(v.value, indent)
	end
	parts[#parts + 1] = "int64(" .. goNum(v.flags) .. ")"
	parts[#parts + 1] = "int64(" .. goNum(v.keywordFlags) .. ")"
	local fn = "genMod"
	local src = v.source
	parts[#parts + 1] = src and quote(src) or quote("")
	for _, tag in ipairs(v) do
		parts[#parts + 1] = goExpr(tag, indent)
	end
	return fn .. "(" .. table.concat(parts, ", ") .. ")"
end

goExpr = function(v, indent)
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
	if isMod(v) then
		return goMod(v, indent)
	end
	-- split array and hash parts
	local arr, kv = {}, {}
	local nArr = 0
	for _, e in ipairs(v) do
		nArr = nArr + 1
		arr[nArr] = e
	end
	local hashKeys = {}
	for k in pairs(v) do
		if not (type(k) == "number" and k == math.floor(k) and k >= 1 and k <= nArr) then
			hashKeys[#hashKeys + 1] = k
		end
	end
	table.sort(hashKeys, function(a, b) return tostring(a) < tostring(b) end)
	local parts = {}
	if #hashKeys == 0 then
		for _, e in ipairs(arr) do
			parts[#parts + 1] = goExpr(e, indent)
		end
		return "[]any{" .. table.concat(parts, ", ") .. "}"
	end
	if nArr == 0 then
		-- numeric hash keys (e.g. hand-written levels with gaps) emit as
		-- their tostring: the canon stringifies keys anyway
		for _, k in ipairs(hashKeys) do
			parts[#parts + 1] = quote(tostring(k)) .. ": " .. goExpr(v[k], indent)
		end
		return "map[string]any{" .. table.concat(parts, ", ") .. "}"
	end
	-- mixed: a group that acquired hash keys (metatable processMod quirk)
	local ap = {}
	for _, e in ipairs(arr) do
		ap[#ap + 1] = goExpr(e, indent)
	end
	local hp = {}
	for _, k in ipairs(hashKeys) do
		hp[#hp + 1] = quote(tostring(k)) .. ": " .. goExpr(v[k], indent)
	end
	return "&modparser.D{Arr: []any{" .. table.concat(ap, ", ") .. "}, KV: map[string]any{" .. table.concat(hp, ", ") .. "}}"
end

-- statMap entry -> Go &StatMapEntry{...}
local function goStatMapEntry(entry)
	local mods = {}
	for _, e in ipairs(entry) do
		if isMod(e) then
			mods[#mods + 1] = goMod(e)
		elseif type(e) == "table" then
			-- a group of mods
			mods[#mods + 1] = goExpr(e)
		else
			error("unexpected statMap array element")
		end
	end
	local kv = {}
	local nArr = #entry
	for _, k in ipairs(sortedKeys(entry)) do
		if not (type(k) == "number" and k >= 1 and k <= nArr) then
			kv[#kv + 1] = quote(k) .. ": " .. goExpr(entry[k])
		end
	end
	local s = "{"
	if #mods > 0 then
		s = s .. "Mods: []any{" .. table.concat(mods, ", ") .. "}"
	end
	if #kv > 0 then
		if #mods > 0 then
			s = s .. ", "
		end
		s = s .. "KV: map[string]any{" .. table.concat(kv, ", ") .. "}"
	end
	return s .. "}"
end

-- ------------------------------------------------- skillstatmap_gen.go --
do
	local mkMod = function(modName, modType, modVal, flags, keywordFlags, ...)
		return { name = modName, type = modType, value = modVal, flags = flags or 0, keywordFlags = keywordFlags or 0, ... }
	end
	local mkFlag = function(modName, ...)
		return mkMod(modName, "FLAG", true, 0, 0, ...)
	end
	local mkSkill = function(dataKey, dataValue, ...)
		return mkMod("SkillData", "LIST", { key = dataKey, value = dataValue }, 0, 0, ...)
	end
	local statMap = LoadModule("Data/SkillStatMap", mkMod, mkFlag, mkSkill)

	local f = assert(io.open("../../data/skillstatmap_gen.go", "w"))
	f:write("// Code generated from Data/SkillStatMap.lua (one-time transform,\n")
	f:write("// Go-maintained): stat id -> internal modifier mapping for skills.\n\npackage data\n\n")
	f:write("var skillStatMap = map[string]*StatMapEntry{\n")
	for _, k in ipairs(sortedKeys(statMap)) do
		f:write("\t", quote(k), ": ", goStatMapEntry(statMap[k]), ",\n")
	end
	f:write("}\n")
	f:close()
	print("skillstatmap entries:", #sortedKeys(statMap))
end

-- ------------------------------------------------ skills_custom_gen.go --
-- The hand-written fragments must be pre-boot state, so reload the skill
-- files fresh (the same constructors Data.lua passes) and keep only the
-- keys the exporter does not generate.
do
	local generated = {
		name = true, hidden = true, description = true, color = true, support = true,
		baseTypeName = true, flavourText = true, baseEffectiveness = true,
		incrementalEffectiveness = true, requireSkillTypes = true, addSkillTypes = true,
		excludeSkillTypes = true, isTrigger = true, supportGemsOnly = true,
		ignoreMinionTypes = true, plusVersionOf = true, weaponTypes = true,
		statDescriptionScope = true, skillTypes = true, minionSkillTypes = true,
		skillTotemId = true, castTime = true, cannotBeSupported = true,
		baseFlags = true, baseMods = true, qualityStats = true, constantStats = true,
		stats = true, notMinionStat = true, levels = true,
	}
	local mkMod = function(modName, modType, modVal, flags, keywordFlags, ...)
		return { name = modName, type = modType, value = modVal, flags = flags or 0, keywordFlags = keywordFlags or 0, ... }
	end
	local mkFlag = function(modName, ...)
		return mkMod(modName, "FLAG", true, 0, 0, ...)
	end
	local mkSkill = function(dataKey, dataValue, ...)
		return mkMod("SkillData", "LIST", { key = dataKey, value = dataValue }, 0, 0, ...)
	end
	local fresh = {}
	for _, name in ipairs({ "act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int" }) do
		LoadModule("Data/Skills/" .. name, fresh, mkMod, mkFlag, mkSkill)
	end

	-- Hand-touched keys: template passthrough between #skill and #mods
	-- writes table keys directly (statMap, skillTypes overrides, levels
	-- when #mods gates them out, ...). Whatever the passthrough touched is
	-- captured from the loaded file: the loaded value is the truth
	-- regardless of which duplicate constructor key won.
	local handKeys = {}        -- skill id -> { key = true }
	local directiveSkills = {} -- ids that came from a #skill directive
	for _, name in ipairs({ "act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int" }) do
		local cur
		local depth = 0
		for line in io.lines("Export/Skills/" .. name .. ".txt") do
			local dname, args = line:match("^#([A-Za-z]+) ?(.*)")
			if dname == "skill" then
				cur = args:match("^([0-9A-Za-z_]+) .+") or args
				directiveSkills[cur] = true
				depth = 1
			elseif dname == "mods" then
				depth = 0
			elseif dname == nil and cur then
				local skillsOpen = line:match('^skills%["([0-9A-Za-z_]+)"%]')
				if skillsOpen then
					-- a fully hand-written block; tracked separately
					depth = 0
				elseif depth == 1 then
					local key = line:match("^\t?([A-Za-z_][A-Za-z0-9_]*) *=")
					if key then
						handKeys[cur] = handKeys[cur] or {}
						handKeys[cur][key] = true
					end
				end
				if depth >= 1 then
					local _, opens = line:gsub("{", "")
					local _, closes = line:gsub("}", "")
					depth = depth + opens - closes
					if depth < 1 then
						depth = 0
					end
				end
			end
		end
	end

	-- Identity registry: templates alias other skills' tables
	-- (baseMods = skills.ExplosiveTrap.baseMods). First owner in sorted
	-- order wins; later holders emit SkillAlias markers so the Go side
	-- shares the same objects.
	local reg = {}
	for _, id in ipairs(sortedKeys(fresh)) do
		local ge = fresh[id]
		for _, k in ipairs(sortedKeys(ge)) do
			local v = ge[k]
			if type(v) == "table" and reg[v] == nil then
				reg[v] = { id = id, key = k }
			end
		end
	end
	local function aliasOf(id, k, v)
		if type(v) ~= "table" then
			return nil
		end
		local owner = reg[v]
		if owner and not (owner.id == id and owner.key == k) then
			return owner
		end
		return nil
	end

	local f = assert(io.open("../../data/skills_custom_gen.go", "w"))
	f:write("// Code generated from the hand-written fragments of the Export/Skills\n")
	f:write("// templates (one-time transform, Go-maintained): per-skill custom keys\n")
	f:write("// and template statMaps. UnportedFn marks Lua functions whose bodies\n")
	f:write("// are ported by the calc modules; the source lives in the templates.\n\npackage data\n\n")
	f:write("import \"github.com/MissingL-tter/missingPassives/modparser\"\n\n")
	f:write("var _ = modparser.D{}\n\n")
	f:write("var skillCustom = map[string]*SkillCustom{\n")
	local nCustom = 0
	for _, id in ipairs(sortedKeys(fresh)) do
		local ge = fresh[id]
		local hand = handKeys[id] or {}
		local full = not directiveSkills[id]
		local customKeys = {}
		for _, k in ipairs(sortedKeys(ge)) do
			if (full or not generated[k] or hand[k]) and k ~= "statMap" then
				customKeys[#customKeys + 1] = k
			end
		end
		if #customKeys > 0 or ge.statMap or full then
			nCustom = nCustom + 1
			f:write("\t", quote(id), ": {")
			if full then
				f:write("Full: true, ")
			end
			if ge.statMap then
				local owner = aliasOf(id, "statMap", ge.statMap)
				if owner then
					f:write("StatMapAlias: ", quote(owner.id))
				else
					f:write("StatMap: map[string]*StatMapEntry{")
					local first = true
					for _, k in ipairs(sortedKeys(ge.statMap)) do
						if not first then
							f:write(", ")
						end
						first = false
						f:write(quote(k), ": ", goStatMapEntry(ge.statMap[k]))
					end
					f:write("}")
				end
				if #customKeys > 0 then
					f:write(", ")
				end
			end
			if #customKeys > 0 then
				f:write("Keys: map[string]any{")
				for i, k in ipairs(customKeys) do
					if i > 1 then
						f:write(", ")
					end
					local owner = aliasOf(id, k, ge[k])
					if owner then
						f:write(quote(k), ": SkillAlias{Skill: ", quote(owner.id), ", Key: ", quote(owner.key), "}")
					else
						f:write(quote(k), ": ", goExpr(ge[k]))
					end
				end
				f:write("}")
			end
			f:write("},\n")
		end
	end
	f:write("}\n")
	f:close()
	print("skills with custom fragments:", nCustom)
end
