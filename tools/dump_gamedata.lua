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
	local s = canon.encode(value)
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
	"costs",
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
-- data.uniques minus "generated" (Uniques/Special/Generated.lua is real
-- code, not yet ported), one record per type.
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
emitHash("rareLikeUniques", data.rareLikeUniques)
emitHash("itemBases", data.itemBases)
emitHash("itemBaseLists", data.itemBaseLists)
emit("itemBaseTypeList", data.itemBaseTypeList)
emit("rares", data.rares)

out:close()
print("dumped subtrees")
