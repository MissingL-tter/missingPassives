-- Dumps the loaded `data` table, subtree by subtree, to
-- test/testdata/gamedata_archive.jsonl for the game-data port's archive
-- comparison.
--
-- Run from .archive/src/:  luajit ../../tools/dump_gamedata.lua
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")

local out = assert(io.open("../../test/testdata/gamedata_archive.jsonl", "w"))
local function emit(key, value)
	out:write('{"k":', canon.quote(key), ',"c":', canon.quote(canon.encode(value)), "}\n")
end
-- Big subtrees compare by double murmur hash of their canon instead.
local function emitHash(key, value)
	local s = canon.encode14(value)
	local h = string.format("%d.%d", murmurHash2(s, 0x9747b28c), murmurHash2(s, 0x2312233))
	out:write('{"k":', canon.quote(key), ',"h":', canon.quote(h), "}\n")
end

-- The subtrees ported so far; grows with the port. Order is the file order.
local keys = {
	-- Data/Misc.lua
	"monsterEvasionTable", "monsterAccuracyTable", "monsterLifeTable",
	"monsterLifeTable2", "monsterLifeTable3", "monsterAllyLifeTable",
	"monsterDamageTable", "monsterAllyDamageTable", "monsterArmourTable",
	"monsterAilmentThresholdTable", "monsterPhysConversionMultiTable",
	"gameConstants", "characterConstants", "monsterConstants",
	"totemLifeMult", "monsterVarietyLifeMult", "mapLevelLifeMult",
	"mapLevelBossLifeMult", "mapLevelBossAilmentMult", "goldRespecPrices",
	-- Data.lua's own tables
	"misc", "powerStatList", "skillColorMap", "monsterExperienceLevelMap",
	"cursePriority", "keystones",
	"ailmentTypeList", "elementalAilmentTypeList",
	"nonDamagingAilmentTypeList", "nonElementalAilmentTypeList",
	"nonDamagingAilment",
	"defaultHighPrecision", "highPrecisionMods", "modScalability",
	"weaponTypeInfo", "unarmedWeaponData",
	"jewelRadii", "jewelRadius", "maxJewelRadius",
	"enchantmentSource",
	"timelessJewelTypes", "timelessJewelSeedMin", "timelessJewelSeedMax",
	"timelessJewelAdditions",
	"itemTagSpecial", "itemTagSpecialExclusionPattern",
	"casterTagCrucibleUniques", "minionTagCrucibleUniques",
	"costs", "mapMods",
	"nodeIDList", "abyssNotableNames", "timelessJewelTradeIDs",
	"timelessJewelLUTs",
	"describeStats", "readLUT", "repairLUTs", "readAbyssJewelLUT",
	"resolveAbyssJewelComponent", "getAbyssJewelComponentRoll",
	"bosses", "bossStats", "enemyIsBossTooltip",
	"essences", "pantheons", "crucible", "masterMods", "flavourText",
	"enchantments",
}

for _, key in ipairs(keys) do
	emit(key, data[key])
end

-- The mod pools, hashed per pool.
local poolKeys = {}
for key in pairs(data.itemMods) do
	poolKeys[#poolKeys + 1] = key
end
table.sort(poolKeys)
for _, key in ipairs(poolKeys) do
	emitHash("itemMods." .. key, data.itemMods[key])
end
emitHash("veiledMods", data.veiledMods)
emit("beastCraft", data.beastCraft)
emit("necropolisMods", data.necropolisMods)
emit("uniqueMods", data.uniqueMods)
emit("clusterJewels", data.clusterJewels)
-- clusterJewelInfoForNotable's jewelTypes arrays are built in Lua
-- hash-iteration order; both sides compare them sorted (a documented
-- deliberate divergence).
local normInfo = {}
for name, info in pairs(data.clusterJewelInfoForNotable) do
	local types = {}
	for i, v in ipairs(info.jewelTypes) do
		types[i] = v
	end
	table.sort(types)
	normInfo[name] = { jewelTypes = types, size = info.size }
end
emit("clusterJewelInfoForNotable", normInfo)
emit("bossSkills", data.bossSkills)
emit("bossSkillsList", data.bossSkillsList)
emit("foulbornMap", data.foulbornMap)
-- data.uniques, one record per type. The tree-dependent generated items
-- (buildTreeDependentUniques) land with the tree-data module and are
-- excluded here.
local uqKeys = {}
for k in pairs(data.uniques) do
	if k ~= "generated" then
		uqKeys[#uqKeys + 1] = k
	end
end
table.sort(uqKeys)
for _, k in ipairs(uqKeys) do
	emitHash("uniques." .. k, data.uniques[k])
end
do
	local generated = {}
	for _, blob in ipairs(data.uniques.generated) do
		local treeBuilt = blob:find("Forbidden Flame", 1, true) or blob:find("Forbidden Flesh", 1, true)
			or blob:find("Skin of the Lords", 1, true) or blob:find("Impossible Escape", 1, true)
		if not treeBuilt then
			generated[#generated + 1] = blob
		end
	end
	emitHash("uniques.generated", generated)
end
-- skills: the templates alias tables across skills, so shared mods' final
-- source is last-writer-wins under pairs() — whose order varies per process
-- (LuaJIT string-hash randomisation). Re-assign existing source fields in
-- sorted skill order so both sides are deterministic.
do
	local ids = {}
	for id in pairs(data.skills) do
		ids[#ids + 1] = id
	end
	table.sort(ids)
	local function reassign(el, modSource, id)
		if type(el) ~= "table" then
			return
		end
		if rawget(el, "source") ~= nil then
			el.source = modSource
		end
		local v = rawget(el, "value")
		if type(v) == "table" and type(rawget(v, "mod")) == "table" and rawget(rawget(v, "mod"), "source") ~= nil then
			rawget(v, "mod").source = "Skill:" .. id
		end
		for _, inner in ipairs(el) do
			reassign(inner, modSource, id)
		end
	end
	for _, id in ipairs(ids) do
		local ge = data.skills[id]
		local modSource = "Skill:" .. id
		for _, list in ipairs({ ge.baseMods, ge.qualityMods, ge.levelMods }) do
			if type(list) == "table" then
				for _, el in ipairs(list) do
					reassign(el, modSource, id)
				end
			end
		end
		for sk, entry in pairs(ge.statMap) do
			if sk ~= "_grantedEffect" then
				for _, el in ipairs(entry) do
					reassign(el, modSource, id)
				end
			end
		end
	end
end

-- skills: strip the statMap._grantedEffect backref, and record each raw
-- statMap's key list as a replay fixture (the boot's lazy statMap copies
-- land in the raw tables; the Go side re-materialises the same keys).
local skillsNorm = {}
local statMapKeys = {}
for id, ge in pairs(data.skills) do
	local copy = {}
	for k, v in pairs(ge) do
		if k == "statMap" then
			local sm = {}
			local keys = {}
			for sk, sv in pairs(v) do
				if sk ~= "_grantedEffect" then
					sm[sk] = sv
					keys[#keys + 1] = sk
				end
			end
			table.sort(keys)
			copy.statMap = sm
			statMapKeys[id] = keys
		else
			copy[k] = v
		end
	end
	skillsNorm[id] = copy
end
emit("skills.statMapKeys", statMapKeys)
emitHash("skills", skillsNorm)
emitHash("skillStatMap", data.skillStatMap)
-- gems: skill tables appear as "\27skill:<id>" markers (both sides), and
-- the pairs-dependent lookups are rebuilt in sorted gem-id order (a
-- documented deliberate divergence for collision/first-match cases).
do
	local function sortedKeys(t)
		local keys = {}
		for k in pairs(t) do
			keys[#keys + 1] = k
		end
		table.sort(keys)
		return keys
	end
	local gemIds = sortedKeys(data.gems)
	local gemsNorm = {}
	for id, gem in pairs(data.gems) do
		local copy = {}
		for k, v in pairs(gem) do
			if k == "grantedEffect" or k == "secondaryGrantedEffect" then
				copy[k] = "\27skill:" .. v.id
			elseif k == "grantedEffectList" then
				local l = {}
				for i, ge in ipairs(v) do
					l[i] = "\27skill:" .. ge.id
				end
				copy[k] = l
			else
				copy[k] = v
			end
		end
		gemsNorm[id] = copy
	end
	emitHash("gems", gemsNorm)

	local gemForSkill, gemForBaseName = {}, {}
	for _, id in ipairs(gemIds) do
		local gem = data.gems[id]
		gemForSkill[gem.grantedEffect.id] = id
		local baseName = gem.name
		if gem.grantedEffect.support and gem.grantedEffectId ~= "SupportBarrage" then
			baseName = baseName .. " Support"
		end
		gemForBaseName[baseName:lower()] = id
		if gem.baseTypeName and gem.baseTypeName ~= baseName then
			gemForBaseName[gem.baseTypeName:lower()] = id
		end
	end
	emit("gemForSkill", gemForSkill)
	emit("gemForBaseName", gemForBaseName)

	local byGameId = {}
	for id, gem in pairs(data.gems) do
		byGameId[gem.gameId] = byGameId[gem.gameId] or {}
		byGameId[gem.gameId][gem.variantId] = gem.id
	end
	emit("gemsByGameId", byGameId)

	local originals = {}
	for _, id in ipairs(gemIds) do
		local base, alt = id:match("^(.+)(Alt[XY])$")
		if not (base and data.gems[base]) then
			originals[#originals + 1] = id
		end
	end
	local gemGranted, gemVaal = {}, {}
	for _, id in ipairs(originals) do
		local gem = data.gems[id]
		if gem.vaalGem and gem.secondaryGrantedEffectId then
			local sec = gem.secondaryGrantedEffectId
			gemGranted[sec] = id
			for _, otherId in ipairs(originals) do
				if data.gems[otherId].grantedEffectId == sec then
					gemVaal[id] = otherId
					break
				end
			end
			for _, alt in ipairs({ "AltX", "AltY" }) do
				if data.skills[sec .. alt] then
					gemGranted[sec .. alt] = id .. alt
					gemVaal[id .. alt] = gemVaal[id] .. alt
				end
			end
		end
	end
	emit("gemGrantedEffectIdForVaalGemId", gemGranted)
	emit("gemVaalGemIdForBaseGemId", gemVaal)
end
emitHash("minions", data.minions)
emitHash("spectres", data.spectres)
emitHash("rareLikeUniques", data.rareLikeUniques)
emitHash("itemBases", data.itemBases)
emitHash("itemBaseLists", data.itemBaseLists)
emit("itemBaseTypeList", data.itemBaseTypeList)
emit("rares", data.rares)

out:close()
print("dumped subtrees")
