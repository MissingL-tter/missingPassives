-- Dumps ModStore/ModDB/ModList behaviour to test/testdata/modstore_archive.jsonl.
--
-- Run from .archive/src/:  luajit ../../tools/dump_modstore.lua
--
-- Every corpus line is parsed and distributed across a store tree
-- (rootDB <- midList <- leafDB) plus enemy/parent actor DBs; multipliers,
-- conditions, actor outputs, items and configs are fixtures EMITTED INTO the
-- archive dump so the Go replay builds the identical world from data. calcLib's
-- gemIsType/getGameIdFromGemName are replaced with fixture-backed stubs
-- (their real implementations belong to other modules); item stubs implement
-- FindModifierSubstring over a fixture lookup.
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")

local function f17(x)
	return string.format("%.17g", x)
end

local out = assert(io.open("../../test/testdata/modstore_archive.jsonl", "w"))
local function emit(parts)
	out:write(table.concat(parts), "\n")
end

-- ---------------------------------------------------------------- corpus --
local corpus = {}
LoadModule("Data/ModCache", corpus)
local lines = {}
for line in pairs(corpus) do
	lines[#lines + 1] = line
end
table.sort(lines)

-- Parse fresh: HeadlessWrapper preloads modLib.parseModCache from
-- Data/ModCache, whose values round-tripped through %.14g serialisation;
-- the Go side always parses fresh, so compare parser semantics, not the
-- cache serialisation.
wipeTable(modLib.parseModCache)

local parsed = {}
for _, line in ipairs(lines) do
	local modList = modLib.parseMod(line)
	if modList and #modList > 0 then
		parsed[#parsed + 1] = { line = line, mods = modList }
	end
end

-- ------------------------------------------------ tag vocabulary harvest --
local varNames, statNames, condNames = {}, {}, {}
local skillTypes, skillParts, slotNames, skillNames, skillIds, baseFlagNames = {}, {}, {}, {}, {}, {}
local function note(set, v)
	if v ~= nil and set[v] == nil then
		set[v] = true
	end
end
local function harvestTag(tag)
	local t = tag.type
	if t == "Multiplier" or t == "MultiplierThreshold" then
		note(varNames, tag.var)
		for _, v in ipairs(tag.varList or {}) do note(varNames, v) end
		note(varNames, tag.divVar)
		note(varNames, tag.limitVar)
		note(varNames, tag.thresholdVar)
		note(statNames, tag.limitStat)
	elseif t == "PerStat" or t == "PercentStat" or t == "StatThreshold" then
		note(statNames, tag.stat)
		for _, v in ipairs(tag.statList or {}) do note(statNames, v) end
		note(statNames, tag.thresholdStat)
		note(varNames, tag.divVar)
		note(varNames, tag.limitVar)
		note(varNames, tag.percentVar)
		note(varNames, tag.thresholdPercentVar)
	elseif t == "Condition" or t == "ActorCondition" then
		note(condNames, tag.var)
		for _, v in ipairs(tag.varList or {}) do note(condNames, v) end
	elseif t == "SkillType" then
		note(skillTypes, tag.skillType)
		for _, v in ipairs(tag.skillTypeList or {}) do note(skillTypes, v) end
	elseif t == "SkillPart" then
		note(skillParts, tag.skillPart)
		for _, v in ipairs(tag.skillPartList or {}) do note(skillParts, v) end
	elseif t == "SlotName" then
		note(slotNames, tag.slotName)
		for _, v in ipairs(tag.slotNameList or {}) do note(slotNames, v) end
	elseif t == "SkillName" then
		note(skillNames, tag.skillName)
		for _, v in ipairs(tag.skillNameList or {}) do note(skillNames, v) end
	elseif t == "SkillId" then
		note(skillIds, tag.skillId)
	elseif t == "BaseFlag" then
		note(baseFlagNames, tag.baseFlag)
	elseif t == "Limit" then
		note(varNames, tag.limitVar)
	end
end
for _, rec in ipairs(parsed) do
	for _, mod in ipairs(rec.mods) do
		for _, tag in ipairs(mod) do
			harvestTag(tag)
		end
	end
end
local function sortedKeys(set)
	local keys = {}
	for k in pairs(set) do keys[#keys + 1] = k end
	table.sort(keys, function(a, b)
		if type(a) == type(b) then return a < b end
		return tostring(a) < tostring(b)
	end)
	return keys
end

-- ------------------------------------------------------------- fixtures --
-- Deterministic values keyed by sorted index; emitted below as data.
local storeNames = { "root", "mid", "leaf", "enemy", "parentDB" }
local multFix, condFix = {}, {}
for _, s in ipairs(storeNames) do multFix[s], condFix[s] = {}, {} end
for i, v in ipairs(sortedKeys(varNames)) do
	if i % 4 ~= 0 then multFix.leaf[v] = (i % 11) + (i % 3) * 0.5 end
	if i % 3 == 0 then multFix.root[v] = (i % 7) * 2 end
	if i % 5 == 0 then multFix.enemy[v] = (i % 6) + 1 end
	if i % 6 == 0 then multFix.parentDB[v] = (i % 4) * 3 end
end
-- Cross-actor multipliers the synthetic Multiplier/MultiplierThreshold
-- records read (SynthMult2 off the enemy; SynthMult3's limit and SynthMT5's
-- threshold off the parent). They go through multFix because that is the
-- one channel both sides see: the stores take it below and the "mult"
-- record hands it to the Go test. A direct enemyDB:AddMod here is invisible
-- to the port and fails the differential (2026-09-02, later.md 5).
multFix.enemy.AbyssJewelType = 2.5
multFix.parentDB.ActiveHolyStrikeMinionLimitDoubled = 4
multFix.parentDB.SynthMTThr = 2
for i, v in ipairs(sortedKeys(condNames)) do
	if i % 3 == 0 then condFix.leaf[v] = true end
	if i % 5 == 0 then condFix.root[v] = true end
	if i % 4 == 0 then condFix.enemy[v] = true end
end
local outputFix = { player = { SynthPercentStatBase = 100 }, enemy = {}, parentA = {} }
for i, s in ipairs(sortedKeys(statNames)) do
	if i % 3 ~= 0 then outputFix.player[s] = (i % 13) * 7 + (i % 2) * 0.25 end
	if i % 4 == 0 then outputFix.enemy[s] = (i % 9) * 3 end
end
outputFix.player.Mana = 1200
outputFix.player.Life = 4800
outputFix.player.ManaUnreserved = -60
outputFix.player.ManaUnreservedPercent = -5
outputFix.enemy.Mana = 300

local itemsFix = {
	item1 = { name = "Voidheart", type = "Ring", rarity = "UNIQUE", corrupted = true, shaper = false, elder = false,
		fms = { ["increased fire damage|ring 1"] = true, ["life|ring 1"] = true, ["intelligence|ring 1"] = true } },
	item2 = { name = "Kalandra's Touch", type = "Ring", rarity = "UNIQUE", corrupted = false, shaper = false, elder = true, fms = { ["intelligence|ring 2"] = true } },
	item3 = { name = "Hubris Circlet", type = "Helmet", rarity = "RARE", corrupted = false, shaper = true, elder = false,
		fms = { ["intelligence|helmet"] = true } },
	item4 = { name = "Cobalt Jewel", type = "Jewel", rarity = "RARE", corrupted = false, shaper = false, elder = false, fms = {} },
}
local function makeItem(fix)
	return {
		name = fix.name, type = fix.type, rarity = fix.rarity,
		corrupted = fix.corrupted, shaper = fix.shaper, elder = fix.elder,
		FindModifierSubstring = function(self, sub, slot)
			return fix.fms[sub .. "|" .. slot] == true
		end,
	}
end

local gemsFix = {
	gem1 = { tags = { support = true, spell = true } },
	gem2 = { tags = { attack = true, projectile = true } },
}
local gameIdsFix = {}
do
	local ids = sortedKeys(skillNames)
	for i, name in ipairs(ids) do
		if i % 2 == 0 then
			gameIdsFix[name:lower()] = "GameId" .. tostring(i % 5)
		end
	end
end

-- calcLib stubs (the real implementations belong to game-data/calc modules).
calcLib = calcLib or {}
calcLib.gemIsType = function(gem, keyword)
	return gem.tags[keyword:lower()] == true
end
calcLib.getGameIdFromGemName = function(name, includeTransfigured)
	return name and gameIdsFix[name:lower()] or nil
end

-- ------------------------------------------------------------ store tree --
local rootDB = new("ModDB")
local midList = new("ModList", rootDB)
local leafDB = new("ModDB", midList)
local enemyDB = new("ModDB")
local parentDB = new("ModDB")

local playerActor = {
	output = outputFix.player,
	itemList = {
		["Ring 1"] = makeItem(itemsFix.item1),
		["Ring 2"] = makeItem(itemsFix.item2),
		["Helmet"] = makeItem(itemsFix.item3),
		["Jewel 3"] = makeItem(itemsFix.item4),
	},
	weaponData1 = { countsAsAll1H = true, AddedSword = true, AddedAxe = false },
	weaponData2 = { },
	activeSkillList = {
		{ skillTypes = { [SkillType.HasReservation] = true }, skillFlags = { }, skillData = { ManaReservedBase = 300, LifeReservedBase = 960 },
			buffList = { { name = "Hatred" } } },
		{ skillTypes = { [SkillType.HasReservation] = true }, skillFlags = { disable = true }, skillData = { ManaReservedBase = 500 },
			buffList = { { name = "Wrath" } } },
	},
	minionData = { monsterTags = { "demon", "humanoid", "beast" } },
	ManaEfficiency = 20,
}
local enemyActor = { output = outputFix.enemy, modDB = enemyDB }
local parentActor = { output = outputFix.parentA, modDB = parentDB }
playerActor.modDB = leafDB
playerActor.enemy = enemyActor
playerActor.parent = parentActor
enemyActor.player = playerActor

for _, store in ipairs({ rootDB, midList, leafDB }) do
	store.actor = playerActor
end
enemyDB.actor = enemyActor
parentDB.actor = parentActor
local storeByName = { root = rootDB, mid = midList, leaf = leafDB, enemy = enemyDB, parentDB = parentDB }
for name, store in pairs(storeByName) do
	store.multipliers = multFix[name] or {}
	store.conditions = condFix[name] or {}
end

-- Distribute the corpus: per parsed line, each mod gets a source and a store
-- cyclically; both are emitted so the replay applies the same placement.
local sources = { "Item:3:Abyssal Sceptre", "Tree:1234", "Skill:Fireball", "Config", "PastLife:Tree" }
local placement = { "leaf", "leaf", "mid", "root", "leaf", "enemy", "mid", "leaf", "root", "parentDB" }
local counter = 0
local modNames = {}
for _, rec in ipairs(parsed) do
	local stores, srcs = {}, {}
	for i, mod in ipairs(rec.mods) do
		counter = counter + 1
		local storeName = placement[(counter % #placement) + 1]
		local src = sources[(counter % #sources) + 1]
		stores[i] = storeName
		srcs[i] = src
		local copy = copyTable(mod)
		modLib.setSource(copy, src)
		storeByName[storeName]:AddMod(copy)
		if storeName ~= "enemy" and storeName ~= "parentDB" then
			modNames[copy.name] = true
		end
	end
	local sparts = {}
	local qparts = {}
	for i = 1, #stores do
		sparts[i] = canon.quote(stores[i])
		qparts[i] = canon.quote(srcs[i])
	end
	emit({ '{"k":"add","line":', canon.quote(rec.line), ',"stores":[', table.concat(sparts, ","), '],"sources":[', table.concat(qparts, ","), ']}' })
end

-- ------------------------------------------------------ fixture emission --
local function jsonMapNum(t)
	local keys = sortedKeys(t)
	local parts = {}
	for i, k in ipairs(keys) do
		parts[i] = canon.quote(tostring(k)) .. ":" .. f17(t[k])
	end
	return "{" .. table.concat(parts, ",") .. "}"
end
local function jsonMapBool(t)
	local keys = sortedKeys(t)
	local parts = {}
	for i, k in ipairs(keys) do
		parts[i] = canon.quote(tostring(k)) .. ":" .. tostring(t[k])
	end
	return "{" .. table.concat(parts, ",") .. "}"
end
for _, s in ipairs(storeNames) do
	emit({ '{"k":"mult","store":', canon.quote(s), ',"vals":', jsonMapNum(multFix[s]), "}" })
	emit({ '{"k":"cond","store":', canon.quote(s), ',"vals":', jsonMapBool(condFix[s]), "}" })
end
emit({ '{"k":"output","actor":"player","vals":', jsonMapNum(outputFix.player), "}" })
emit({ '{"k":"output","actor":"enemy","vals":', jsonMapNum(outputFix.enemy), "}" })
emit({ '{"k":"output","actor":"parent","vals":', jsonMapNum(outputFix.parentA), "}" })
do
	local parts = {}
	local names = sortedKeys(gameIdsFix)
	for i, n in ipairs(names) do
		parts[i] = canon.quote(n) .. ":" .. canon.quote(gameIdsFix[n])
	end
	emit({ '{"k":"gameIds","vals":{', table.concat(parts, ","), "}}" })
end
do
	local parts = {}
	for _, id in ipairs({ "item1", "item2", "item3", "item4" }) do
		local it = itemsFix[id]
		local fparts = {}
		for _, key in ipairs(sortedKeys(it.fms)) do
			fparts[#fparts + 1] = canon.quote(key) .. ":true"
		end
		parts[#parts + 1] = table.concat({ canon.quote(id), ':{"name":', canon.quote(it.name),
			',"type":', canon.quote(it.type), ',"rarity":', canon.quote(it.rarity),
			',"corrupted":', tostring(it.corrupted), ',"shaper":', tostring(it.shaper),
			',"elder":', tostring(it.elder), ',"fms":{', table.concat(fparts, ","), "}}" })
	end
	emit({ '{"k":"items","vals":{', table.concat(parts, ","), "}}" })
	local gparts = {}
	for _, id in ipairs({ "gem1", "gem2" }) do
		local tparts = {}
		for _, t in ipairs(sortedKeys(gemsFix[id].tags)) do
			tparts[#tparts + 1] = canon.quote(t) .. ":true"
		end
		gparts[#gparts + 1] = canon.quote(id) .. ':{"tags":{' .. table.concat(tparts, ",") .. "}}"
	end
	emit({ '{"k":"gems","vals":{', table.concat(gparts, ","), "}}" })
end

-- Tag-type census on stderr, to judge archive-dump coverage.
do
	local census = {}
	for _, rec in ipairs(parsed) do
		for _, mod in ipairs(rec.mods) do
			for _, tag in ipairs(mod) do
				census[tag.type or "?"] = (census[tag.type or "?"] or 0) + 1
			end
		end
	end
	for _, t in ipairs(sortedKeys(census)) do
		io.stderr:write(string.format("tag %-22s %d\n", t, census[t]))
	end
end

-- --------------------------------------------------------------- configs --
local stList = sortedKeys(skillTypes)
local spList = sortedKeys(skillParts)
local snList = sortedKeys(slotNames)
local sknList = sortedKeys(skillNames)
local sidList = sortedKeys(skillIds)
local bfList = sortedKeys(baseFlagNames)
local condList = sortedKeys(condNames)

local function pickTypes(step)
	local t = {}
	for i, v in ipairs(stList) do
		if i % step == 0 then t[v] = true end
	end
	return t
end
local function pickConds(step)
	local t = {}
	for i, v in ipairs(condList) do
		if i % step == 0 then t[v] = true end
	end
	return t
end

local cfgs = {
	nil, -- cfg 1: no config at all
	{ }, -- cfg 2: empty
	{ flags = ModFlag.Attack + ModFlag.Melee, keywordFlags = KeywordFlag.Attack },
	{ flags = ModFlag.Spell + ModFlag.Area + ModFlag.Hit, keywordFlags = KeywordFlag.Spell + KeywordFlag.Hit },
	{ keywordFlags = KeywordFlag.Aura + KeywordFlag.MatchAll },
	{ source = "Item" },
	{ source = "Tree", flags = ModFlag.Projectile },
	{ skillName = sknList[1] or "Fireball", summonSkillName = "SynthSummonName", skillTypes = pickTypes(2), skillPart = spList[1],
	  slotName = snList[1], skillDist = 22, skillCond = pickConds(2),
	  skillGrantedEffect = { id = sidList[1] or "None", baseFlags = { [bfList[1] or "none"] = true } } },
	{ skillName = sknList[3] or "", skillTypes = pickTypes(3), skillPart = 2, slotName = snList[2],
	  skillDist = 8, actor = "enemy", skillCond = pickConds(3), item = makeItem(itemsFix.item3) },
	{ skillDist = 55, skillTypes = pickTypes(5), baseFlags = { [bfList[2] or "none"] = true },
	  skillGem = gemsFix.gem1, slotName = "Weapon 1", socketColor = "R", socketNum = 2,
	  strengthGems = 2, dexterityGems = 0, intelligenceGems = 0 },
	{ flags = ModFlag.Claw + ModFlag.Hit + ModFlag.Attack + ModFlag.Weapon1H + ModFlag.WeaponMelee,
	  keywordFlags = KeywordFlag.Hit + KeywordFlag.Attack, skillName = sknList[4],
	  skillGem = gemsFix.gem2, socketColor = "B", socketNum = 1, strengthGems = 0, dexterityGems = 0, intelligenceGems = 3,
	  skillStats = { UsedSkillStat = 77 } },
	{ flags = ModFlag.Minion, keywordFlags = KeywordFlag.Minion, actor = "parent", summonSkillName = sknList[6],
	  skillCond = pickConds(4), skillPart = spList[3], skillDist = 40 },
}

-- Emit configs as data: only the fields the replay needs to rebuild them.
local function cfgJson(cfg)
	if cfg == nil then
		return "null"
	end
	local parts = {}
	local function add(k, v)
		parts[#parts + 1] = canon.quote(k) .. ":" .. v
	end
	if cfg.flags then add("flags", string.format("%d", cfg.flags)) end
	if cfg.keywordFlags then add("keywordFlags", string.format("%d", cfg.keywordFlags)) end
	if cfg.source then add("source", canon.quote(cfg.source)) end
	if cfg.skillName then add("skillName", canon.quote(cfg.skillName)) end
	if cfg.summonSkillName then add("summonSkillName", canon.quote(cfg.summonSkillName)) end
	if cfg.skillDist then add("skillDist", f17(cfg.skillDist)) end
	if cfg.skillPart ~= nil then
		if type(cfg.skillPart) == "number" then add("skillPartNum", f17(cfg.skillPart)) else add("skillPartStr", canon.quote(tostring(cfg.skillPart))) end
	end
	if cfg.slotName then add("slotName", canon.quote(cfg.slotName)) end
	if cfg.socketColor then add("socketColor", canon.quote(cfg.socketColor)) end
	if cfg.socketNum then add("socketNum", f17(cfg.socketNum)) end
	if cfg.strengthGems then add("strengthGems", f17(cfg.strengthGems)) end
	if cfg.dexterityGems then add("dexterityGems", f17(cfg.dexterityGems)) end
	if cfg.intelligenceGems then add("intelligenceGems", f17(cfg.intelligenceGems)) end
	if cfg.actor then add("actor", canon.quote(cfg.actor)) end
	if cfg.skillCond then add("skillCond", jsonMapBool(cfg.skillCond)) end
	if cfg.skillTypes then
		local m = {}
		for k, v in pairs(cfg.skillTypes) do m[tostring(k)] = v end
		add("skillTypes", jsonMapBool(m))
	end
	if cfg.baseFlags then add("baseFlags", jsonMapBool(cfg.baseFlags)) end
	if cfg.skillGrantedEffect then
		add("geId", canon.quote(cfg.skillGrantedEffect.id))
		add("geBaseFlags", jsonMapBool(cfg.skillGrantedEffect.baseFlags))
	end
	if cfg.skillGem then
		add("gem", cfg.skillGem == gemsFix.gem1 and canon.quote("gem1") or canon.quote("gem2"))
	end
	if cfg.item then add("item", canon.quote("item3")) end
	if cfg.skillStats then add("skillStats", jsonMapNum(cfg.skillStats)) end
	return "{" .. table.concat(parts, ",") .. "}"
end
do
	local parts = {}
	for i = 1, 12 do
		parts[i] = cfgJson(cfgs[i])
	end
	emit({ '{"k":"cfgs","list":[', table.concat(parts, ","), "]}" })
end

-- --------------------------------------------------------------- queries --
local nameList = sortedKeys(modNames)
local records = 0
for ni, name in ipairs(nameList) do
	local names = { name }
	if ni % 10 == 0 and nameList[ni + 1] then
		names[2] = nameList[ni + 1]
	end
	local parts = { '{"k":"q","names":[' }
	for i, n in ipairs(names) do
		if i > 1 then parts[#parts + 1] = "," end
		parts[#parts + 1] = canon.quote(n)
	end
	parts[#parts + 1] = '],"res":['
	-- Some corpus mods are out of contract for some aggregations (e.g.
	-- Tabulate(nil) hitting a boolean-valued mod with a Multiplier tag
	-- errors in the reference); those results are recorded as "!" and the
	-- port must fail the same way.
	local function tryNum(fn)
		local ok, v = pcall(fn)
		if not ok then return '"!"' end
		return '"' .. f17(v) .. '"'
	end
	for ci = 1, 12 do
		local cfg = cfgs[ci]
		if ci > 1 then parts[#parts + 1] = "," end
		local sumB = tryNum(function() return leafDB:Sum("BASE", cfg, unpack(names)) end)
		local sumI = tryNum(function() return leafDB:Sum("INC", cfg, unpack(names)) end)
		local more = tryNum(function() return leafDB:More(cfg, unpack(names)) end)
		local okF, flag = pcall(function() return leafDB:Flag(cfg, unpack(names)) end)
		local okO, ovr = pcall(function() return leafDB:Override(cfg, unpack(names)) end)
		local okL, listC = pcall(function() return canon.encode(leafDB:List(cfg, unpack(names))) end)
		local okT, tabC = pcall(function() return canon.encode(leafDB:Tabulate(nil, cfg, unpack(names))) end)
		local hasB = rootDB:HasMod("BASE", cfg, unpack(names)) -- through-List HasMod errors in the reference
		local okX, maxV = pcall(function() return leafDB:Max(cfg, unpack(names)) end)
		local okN, minV = pcall(function() return leafDB:Min(cfg, unpack(names)) end)
		if not okL then listC = "!" end
		if not okT then tabC = "!" end
		parts[#parts + 1] = table.concat({
			'{"sb":', sumB, ',"si":', sumI, ',"mo":', more,
			',"fl":', okF and tostring(flag == true) or '"!"',
			',"ov":', okO and canon.encode(ovr) or '"!"',
			',"li":', canon.quote(listC),
			',"ta":', canon.quote(tabC),
			',"ha":', tostring(hasB),
			',"mx":', (not okX) and '"!"' or maxV and ('"' .. f17(maxV) .. '"') or "null",
			',"mn":', (not okN) and '"!"' or minV and ('"' .. f17(minV) .. '"') or "null",
			"}",
		})
	end
	parts[#parts + 1] = "]}"
	emit(parts)
	records = records + 1
end

-- Multiplier/condition lookups across configs.
for i, v in ipairs(sortedKeys(varNames)) do
	if i % 2 == 0 then
		local parts = { '{"k":"gm","var":', canon.quote(v), ',"res":[' }
		for ci = 1, 12 do
			if ci > 1 then parts[#parts + 1] = "," end
			local ok, r = pcall(function() return leafDB:GetMultiplier(v, cfgs[ci]) end)
			parts[#parts + 1] = ok and ('"' .. f17(r) .. '"') or '"!"'
		end
		parts[#parts + 1] = "]}"
		emit(parts)
	end
end
for i, v in ipairs(sortedKeys(condNames)) do
	if i % 2 == 0 then
		local parts = { '{"k":"gc","var":', canon.quote(v), ',"res":[' }
		for ci = 1, 12 do
			if ci > 1 then parts[#parts + 1] = "," end
			local ok, r = pcall(function() return leafDB:GetCondition(v, cfgs[ci]) end)
			parts[#parts + 1] = ok and tostring(r == true) or '"!"'
		end
		parts[#parts + 1] = "]}"
		emit(parts)
	end
end

-- Queries on the enemy store (player-actor resolution from the enemy side).
for ni, name in ipairs(nameList) do
	if ni % 25 == 0 then
		local okS, sumB = pcall(function() return enemyDB:Sum("BASE", cfgs[8], name) end)
		local okM, more = pcall(function() return enemyDB:More(cfgs[9], name) end)
		local okT, tab = pcall(function() return canon.encode(enemyDB:Tabulate(nil, cfgs[2], name)) end)
		if not okT then tab = "!" end
		emit({ '{"k":"eq","name":', canon.quote(name), ',"sb":', okS and ('"' .. f17(sumB) .. '"') or '"!"', ',"mo":', okM and ('"' .. f17(more) .. '"') or '"!"', ',"ta":', canon.quote(tab), '}' })
	end
end

-- ---------------------------------------------- construction behaviours --
local function listArr(list)
	local arr = {}
	for i = 1, #list do arr[i] = list[i] end
	return arr
end
local scales = { 0.5, 1.2, 2, -0.4, 2.76, 0 }
for pi, rec in ipairs(parsed) do
	if pi % 7 == 0 then
		local scale = scales[(pi % #scales) + 1]
		local list = new("ModList")
		for _, mod in ipairs(rec.mods) do
			list:ScaleAddMod(copyTable(mod), scale)
		end
		local db = new("ModDB")
		for _, mod in ipairs(rec.mods) do
			db:ScaleAddMod(copyTable(mod), scale)
		end
		emit({ '{"k":"scale","line":', canon.quote(rec.line), ',"scale":"', f17(scale),
			'","list":', canon.quote(canon.encode(listArr(list))), ',"db":', canon.quote(canon.encode(db.mods)), "}" })
	end
	if pi % 11 == 0 then
		local list = new("ModList")
		for _, mod in ipairs(rec.mods) do
			list:MergeMod(copyTable(mod))
		end
		for _, mod in ipairs(rec.mods) do
			list:MergeMod(copyTable(mod))
		end
		emit({ '{"k":"mergeMod","line":', canon.quote(rec.line), ',"list":', canon.quote(canon.encode(listArr(list))), "}" })
	end
	if pi % 13 == 0 then
		-- Replace/Convert flows on a DB with a List parent.
		local base = new("ModList")
		local db = new("ModDB", base)
		for _, mod in ipairs(rec.mods) do
			base:AddMod(copyTable(mod))
		end
		for i, mod in ipairs(rec.mods) do
			local repl = copyTable(mod)
			repl.value = (type(repl.value) == "number") and (repl.value + 100) or repl.value
			if not db:ReplaceModInternal(repl) then db:AddMod(repl) end
			local conv = copyTable(mod)
			conv.name = mod.name .. "X"
			if not db:ConvertModInternal(mod.name, conv) then db:AddMod(conv) end
		end
		emit({ '{"k":"replace","line":', canon.quote(rec.line),
			',"base":', canon.quote(canon.encode(listArr(base))), ',"db":', canon.quote(canon.encode(db.mods)), "}" })
	end
end

-- ------------------------------------------------------- mergeKeystones --
do
	-- Build a keystone map out of parsed corpus lines and feed a DB holding
	-- the corpus' "Keystone" LIST mods (plus synthetic granters).
	local keystoneMap = {}
	local mapNames = {}
	local ki = 0
	for pi, rec in ipairs(parsed) do
		if pi % 97 == 0 and ki < 24 then
			ki = ki + 1
			local ksName = "Keystone" .. ki
			keystoneMap[ksName] = { modList = {} }
			for _, mod in ipairs(rec.mods) do
				table.insert(keystoneMap[ksName].modList, copyTable(mod))
			end
			mapNames[#mapNames + 1] = ksName
		end
	end
	local db = new("ModDB")
	for i, ksName in ipairs(mapNames) do
		local src = (i % 2 == 0) and "Tree:node" or "Item:5:Sceptre"
		db:AddMod(modLib.createMod("Keystone", "LIST", ksName, src))
		if i % 3 == 0 then
			-- duplicate granter: keystonesAdded must dedupe
			db:AddMod(modLib.createMod("Keystone", "LIST", ksName, "Item:6:Ring"))
		end
	end
	db:AddMod(modLib.createMod("Keystone", "LIST", "UnknownKeystone", "Tree:x"))
	local mapC = {}
	for i, ksName in ipairs(mapNames) do
		mapC[i] = canon.quote(ksName) .. ":" .. canon.quote(canon.encode(keystoneMap[ksName].modList))
	end
	local env = { spec = { tree = { keystoneMap = keystoneMap } } }
	modLib.mergeKeystones(env, db)
	local addedNames = {}
	for name in pairs(env.keystonesAdded) do addedNames[#addedNames + 1] = name end
	table.sort(addedNames)
	local addedC = {}
	for i, n in ipairs(addedNames) do addedC[i] = canon.quote(n) end
	emit({ '{"k":"keystones","map":{', table.concat(mapC, ","), '},"added":[', table.concat(addedC, ","),
		'],"db":', canon.quote(canon.encode(db.mods)), "}" })
end

-- ------------------------------------------------------- synthetic mods --
-- The corpus never produces some tag types/branches (MonsterTag, BaseFlag,
-- SkillPart, Limit, the Varunastra weapon-condition path, Kalandra slot
-- swaps, reservation stats, keyOfScaledMod scaling ...). Cover them with
-- hand-built mods; each is emitted as canon JSON for the replay to rebuild.
do
	local nanValue = 0 / 0
	outputFix.parentA.ManaUnreserved = nanValue
	outputFix.parentA.Mana = 777
	emit({ '{"k":"outputNan","actor":"parent","stat":"ManaUnreserved"}' })
	emit({ '{"k":"outputSet","actor":"parent","stat":"Mana","val":"777"}' })

	local vn = sortedKeys(varNames)
	local sn = sortedKeys(statNames)
	local mkm = modLib.createMod
	local synthList = {
		-- Feeds the Multiplier variable the synthetic Multiplier/GlobalLimit mods
		-- read. Without it every plain var = vn[2] reader summed to 0 and the
		-- shared global limit was never reached (found 2026-09-02 when the
		-- SynthGlobalLimitPair skip came out). 1.5 so floor (1) and noFloor
		-- (1.5) differ.
		mkm("Multiplier:" .. vn[2], "BASE", 1.5, "Synth"),
		-- feed the other Multiplier vars the still-inert synthetic records read,
		-- so their tag branch actually evaluates (2026-09-02, later.md 5).
		mkm("Multiplier:ActiveGolemLimitDoubled", "BASE", 3, "Synth"),
		mkm("Multiplier:SynthMTMult", "BASE", 5, "Synth"),
		mkm("SynthMonsterTag1", "INC", 10, "Synth", { type = "MonsterTag", monsterTag = "Demon" }),
		mkm("SynthMonsterTag2", "INC", 10, "Synth", { type = "MonsterTag", monsterTag = "Beast" }),
		mkm("SynthMonsterTag3", "INC", 10, "Synth", { type = "MonsterTag", monsterTagList = { "beast", "HUMANOID" } }),
		mkm("SynthMonsterTag4", "INC", 10, "Synth", { type = "MonsterTag", monsterTag = "Demon", neg = true }),
		mkm("SynthBaseFlag1", "INC", 10, "Synth", { type = "BaseFlag", baseFlag = bfList[2] or "none" }),
		mkm("SynthBaseFlag2", "INC", 10, "Synth", { type = "BaseFlag", baseFlag = "absent", neg = true }),
		mkm("SynthSkillPart1", "INC", 10, "Synth", { type = "SkillPart", skillPart = 2 }),
		mkm("SynthSkillPart2", "INC", 10, "Synth", { type = "SkillPart", skillPartList = { 2, 3 } }),
		mkm("SynthSkillPart3", "INC", 10, "Synth", { type = "SkillPart", skillPart = 2, neg = true }),
		mkm("SynthLimit1", "BASE", 500, "Synth", { type = "Limit", limit = 42 }),
		mkm("SynthLimit2", "BASE", 500, "Synth", { type = "Limit", limitVar = vn[3] }),
		mkm("SynthMult1", "BASE", 2, "Synth", { type = "Multiplier", var = vn[2], div = 2, base = 5, invert = true }),
		mkm("SynthMult2", "BASE", 2, "Synth", { type = "Multiplier", var = vn[2], noFloor = true, limit = 3, actor = "enemy" }),
		mkm("SynthMult3", "BASE", 2, "Synth", { type = "Multiplier", var = vn[5], limitVar = vn[7], limitActor = "parent", limitTotal = true }),
		mkm("SynthMult4", "BASE", 2, "Synth", { type = "Multiplier", var = vn[5], limitNegTotal = true, limit = -3 }),
		mkm("SynthMult5", "BASE", 2, "Synth", { type = "Multiplier", var = vn[5], limitStat = sn[2], limitTotal = true }),
		mkm("SynthMult6", "BASE", 2, "Synth", { type = "Multiplier", varList = { vn[2], vn[5] }, divVar = vn[9] }),
		mkm("SynthMult7", "BASE", 2, "Synth", { type = "Multiplier", var = vn[2], actor = "nonexistent" }),
		mkm("SynthMultList", "LIST", { mod = mkm("Nested", "BASE", 3, "SynthNested") }, "Synth", { type = "Multiplier", var = vn[2], base = 1, limitTotal = true, limit = 7 }),
		mkm("SynthMultKey", "LIST", { keyOfScaledMod = "value", value = 10, other = "x" }, "Synth", { type = "Multiplier", var = vn[2] }),
		mkm("SynthMT1", "BASE", 4, "Synth", { type = "MultiplierThreshold", var = vn[2], threshold = 2 }),
		mkm("SynthMT2", "BASE", 4, "Synth", { type = "MultiplierThreshold", var = vn[2], threshold = 2, upper = true }),
		mkm("SynthMT3", "BASE", 4, "Synth", { type = "MultiplierThreshold", var = vn[2], thresholdVar = vn[5] }),
		mkm("SynthMT4", "BASE", 4, "Synth", { type = "MultiplierThreshold", var = vn[2], threshold = 99, equals = true }),
		mkm("SynthMT5", "BASE", 4, "Synth", { type = "MultiplierThreshold", var = "SynthMTMult", thresholdActor = "parent", thresholdVar = "SynthMTThr" }),
		mkm("SynthPS1", "BASE", 1, "Synth", { type = "PerStat", stat = "ManaReservedPercent" }),
		mkm("SynthPS2", "BASE", 1, "Synth", { type = "PerStat", stat = "LifeReservedPercent", div = 5 }),
		mkm("SynthPS3", "BASE", 1, "Synth", { type = "PerStat", stat = "ManaUnreserved" }),
		mkm("SynthPS4", "BASE", 1, "Synth", { type = "PerStat", stat = "ManaUnreserved", actor = "parent" }),
		mkm("SynthPS5", "BASE", 1, "Synth", { type = "PerStat", statList = { sn[2], sn[4] }, limitVar = vn[3], limitTotal = true }),
		mkm("SynthPS6", "BASE", 1, "Synth", { type = "PerStat", stat = sn[2], divVar = vn[9], base = 2 }),
		mkm("SynthPCS1", "BASE", 3, "Synth", { type = "PercentStat", stat = "SynthPercentStatBase", percentVar = vn[2], floor = true }),
		mkm("SynthPCS2", "BASE", 3, "Synth", { type = "PercentStat", statList = { sn[2], sn[4] }, percent = 50, limit = 9, actor = "enemy" }),
		mkm("SynthST1", "BASE", 6, "Synth", { type = "StatThreshold", stat = sn[2], thresholdStat = sn[4] }),
		mkm("SynthST2", "BASE", 6, "Synth", { type = "StatThreshold", statList = { sn[2] }, threshold = -1 }),
		mkm("SynthST3", "BASE", 6, "Synth", { type = "StatThreshold", stat = "SynthUnsetStat", threshold = 5, thresholdPercentVar = vn[2], upper = true }),
		mkm("SynthCondNeg1", "INC", 10, "Synth", { type = "Condition", var = "Sword", neg = true }),
		mkm("SynthCondNeg2", "INC", 10, "Synth", { type = "Condition", var = "Axe", neg = true }),
		mkm("SynthCondNeg3", "INC", 10, "Synth", { type = "Condition", varList = { "Sword", condList[3] }, neg = true }),
		mkm("SynthActorCond1", "INC", 10, "Synth", { type = "ActorCondition", actor = "enemy", var = condList[4] }),
		mkm("SynthActorCond2", "INC", 10, "Synth", { type = "ActorCondition", actor = "enemy" }),
		mkm("SynthActorCond3", "INC", 10, "Synth", { type = "ActorCondition", actor = "nonexistent", var = condList[4], neg = true }),
		mkm("SynthItem1", "INC", 10, "Synth", { type = "ItemCondition", itemSlot = "ring 1", searchCond = "Life" }),
		mkm("SynthItem2", "INC", 10, "Synth", { type = "ItemCondition", itemSlot = "Ring", bothSlots = true, corruptedCond = true }),
		mkm("SynthItem3", "INC", 10, "Synth", { type = "ItemCondition", allSlots = true, searchCond = "intelligence", excludeSelf = true, itemSlot = "Helmet" }),
		mkm("SynthItem4", "INC", 10, "Synth", { type = "ItemCondition", itemSlot = "Helmet", rarityCond = "RARE", shaperCond = true, elderCond = false }),
		mkm("SynthItem5", "INC", 10, "Synth", { type = "ItemCondition", itemSlot = "Ring 2", nameCond = "voidheart" }),
		mkm("SynthItem6", "INC", 10, "Synth", { type = "ItemCondition", itemSlot = "Weapon 1", searchCond = "Life", neg = true }),
		mkm("SynthSock1", "INC", 10, "Synth", { type = "SocketedIn", slotName = "Weapon 1" }),
		mkm("SynthSock2", "INC", 10, "Synth", { type = "SocketedIn", slotName = "Weapon 1", keyword = "spell" }),
		mkm("SynthSock3", "INC", 10, "Synth", { type = "SocketedIn", socketColor = "R", sockets = "all" }),
		mkm("SynthSock4", "INC", 10, "Synth", { type = "SocketedIn", socketColor = "B", sockets = { 1, 3 } }),
		mkm("SynthSock5", "INC", 10, "Synth", { type = "SocketedIn", socketColor = "R", sockets = 3 }),
		mkm("SynthSkillName1", "INC", 10, "Synth", { type = "SkillName", skillName = "SynthSummonName", summonSkill = true }),
		mkm("SynthSkillName2", "INC", 10, "Synth", { type = "SkillName", skillNameList = { sknList[2], sknList[4] }, includeTransfigured = true }),
		mkm("SynthSkillId1", "INC", 10, "Synth", { type = "SkillId", skillId = sidList[1] }),
		mkm("SynthGlobalLimit1", "BASE", 30, "Synth", { type = "Multiplier", var = vn[2], globalLimit = 50, globalLimitKey = "SynthGL" }),
		mkm("SynthGlobalLimit2", "BASE", 30, "Synth", { type = "Multiplier", var = vn[2], globalLimit = 50, globalLimitKey = "SynthGL" }),
		mkm("SynthUnscalable", "BASE", 7, "Synth", { type = "Condition", var = condList[3], unscalable = true }),
	}
	local synthDB = new("ModDB", midList)
	synthDB.actor = playerActor
	synthDB.multipliers = {}
	synthDB.conditions = {}
	for _, m in ipairs(synthList) do
		synthDB:AddMod(m)
		emit({ '{"k":"synth","spec":', canon.quote(canon.encode(m)), "}" })
	end
	local function tryNum2(fn)
		local ok, v = pcall(fn)
		if not ok then return '"!"' end
		return '"' .. f17(v) .. '"'
	end
	for _, m in ipairs(synthList) do
		local parts = { '{"k":"sq","name":', canon.quote(m.name), ',"res":[' }
		for ci = 1, 12 do
			local cfg = cfgs[ci]
			if ci > 1 then parts[#parts + 1] = "," end
			local sb = tryNum2(function() return synthDB:Sum("BASE", cfg, m.name) end)
			local si = tryNum2(function() return synthDB:Sum("INC", cfg, m.name) end)
			local okT, tabC = pcall(function() return canon.encode(synthDB:Tabulate(nil, cfg, m.name)) end)
			if not okT then tabC = "!" end
			parts[#parts + 1] = table.concat({ '{"sb":', sb, ',"si":', si, ',"ta":', canon.quote(tabC), "}" })
		end
		parts[#parts + 1] = "]}"
		emit(parts)
	end
	-- shared global limit across both mods in one Sum: the same three
	-- checks as every other synthetic record. Until 2026-09-02 this block
	-- hardcoded si to 0 and ta to "skip" with no reason recorded; the
	-- port's two-name Tabulate under a shared limit was unverified.
	do
		local parts = { '{"k":"sq","name":"SynthGlobalLimitPair","res":[' }
		for ci = 1, 12 do
			local cfg = cfgs[ci]
			if ci > 1 then parts[#parts + 1] = "," end
			local sb = tryNum2(function() return synthDB:Sum("BASE", cfg, "SynthGlobalLimit1", "SynthGlobalLimit2") end)
			local si = tryNum2(function() return synthDB:Sum("INC", cfg, "SynthGlobalLimit1", "SynthGlobalLimit2") end)
			local okT, tabC = pcall(function() return canon.encode(synthDB:Tabulate(nil, cfg, "SynthGlobalLimit1", "SynthGlobalLimit2")) end)
			if not okT then tabC = "!" end
			parts[#parts + 1] = table.concat({ '{"sb":', sb, ',"si":', si, ',"ta":', canon.quote(tabC), "}" })
		end
		parts[#parts + 1] = "]}"
		emit(parts)
	end
	-- ScaleAddMod over the synthetic value shapes
	do
		local list = new("ModList")
		list:ScaleAddMod(copyTable(synthList[#synthList]), 2.5)
		list:ScaleAddMod(copyTable(synthList[20]), 2.5) -- keyOfScaledMod
		list:ScaleAddMod(copyTable(synthList[19]), 2.5) -- nested value.mod
		emit({ '{"k":"synthScale","list":', canon.quote(canon.encode(listArr(list))), "}" })
	end
end
out:close()
io.stderr:write(string.format("modstore archive dump: %d query records over %d names, %d parsed lines\n",
	records, #nameList, #parsed))
