-- Generates modparser/vocab_gen.go from src/Data.
-- Run from .archive/src/:  luajit ../../tools/gen_vocab.lua
--
-- ModParser.lua builds several of its tables by looping over the exported game
-- data (skill names, gem tags, cluster jewel notables). The Go parser runs the
-- same loops; this file gives it the same vocabulary. Everything here reads
-- src/Data only — nothing is taken from ModParser.lua.
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local function q(s)
	return string.format("%q", s)
end

local out = assert(io.open("../../modparser/vocab_gen.go", "w"))
out:write([[
// Code generated from .archive/src/Data by tools/gen_vocab.lua. DO NOT EDIT.
// Regenerate from .archive/src/ with: luajit ../../tools/gen_vocab.lua

package modparser

// skillInfo mirrors what ModParser reads from data.skills when building
// gemIdLookup: every skill's display name mapped to its id, filtered on
// hidden/fromItem/fromTree.
type skillInfo struct {
	id, name                  string
	hidden, fromItem, fromTree bool
}

// gemSkillInfo mirrors what ModParser reads from data.gems when building
// skillNameList, preSkillNameList and the per-skill specialModList entries.
type gemSkillInfo struct {
	name                                    string
	totem, buff, auraOrHerald, curse        bool
	mine, trap, chaining, bow, projectile   bool
}

// supportGemInfo carries what extraSupport resolves through
// data.gemForBaseName and data.gems.
type supportGemInfo struct {
	grantedEffectId          string
	secondaryGrantedEffectId string
	hasSecondary             bool
	secondarySupport         bool
}
]])

-- data.skills, sorted for stable output
local skillIds = {}
for id in pairs(data.skills) do
	skillIds[#skillIds + 1] = id
end
table.sort(skillIds)

out:write("\nvar skillData = []skillInfo{\n")
for _, id in ipairs(skillIds) do
	local sk = data.skills[id]
	out:write(string.format("\t{id: %s, name: %s, hidden: %s, fromItem: %s, fromTree: %s},\n",
		q(sk.id), q(sk.name or ""), tostring(sk.hidden or false),
		tostring(sk.fromItem or false), tostring(sk.fromTree or false)))
end
out:write("}\n")

-- data.gems: non-hidden, non-support granted effects with the tags the
-- construction loops test
local gemIds = {}
for id in pairs(data.gems) do
	gemIds[#gemIds + 1] = id
end
table.sort(gemIds)

out:write("\nvar gemSkills = []gemSkillInfo{\n")
local seen = {}
for _, gemId in ipairs(gemIds) do
	local gemData = data.gems[gemId]
	local grantedEffect = gemData.grantedEffect
	if not grantedEffect.hidden and not grantedEffect.support and not seen[grantedEffect.name] then
		seen[grantedEffect.name] = true
		local buff = (grantedEffect.skillTypes[SkillType.Buff] or grantedEffect.baseFlags.buff) and true or false
		out:write(string.format(
			"\t{name: %s, totem: %s, buff: %s, auraOrHerald: %s, curse: %s, mine: %s, trap: %s, chaining: %s, bow: %s, projectile: %s},\n",
			q(grantedEffect.name), tostring(gemData.tags.totem or false), tostring(buff),
			tostring((gemData.tags.aura or gemData.tags.herald) and true or false),
			tostring(gemData.tags.curse or false), tostring(gemData.tags.mine or false),
			tostring(gemData.tags.trap or false), tostring(gemData.tags.chaining or false),
			tostring(gemData.tags.bow or false), tostring(gemData.tags.projectile or false)))
	end
end
out:write("}\n")

-- extraSupport's gem resolution, precomputed per skill id
out:write("\nvar supportGemResolve = map[string]supportGemInfo{\n")
for _, id in ipairs(skillIds) do
	local sk = data.skills[id]
	local gemId = data.gemForBaseName[(sk.name .. " Support"):lower()]
	if gemId and data.gems[gemId] then
		local g = data.gems[gemId]
		out:write(string.format(
			"\t%s: {grantedEffectId: %s, secondaryGrantedEffectId: %s, hasSecondary: %s, secondarySupport: %s},\n",
			q(sk.id), q(g.grantedEffectId or ""), q(g.secondaryGrantedEffectId or ""),
			tostring(g.secondaryGrantedEffect ~= nil),
			tostring((g.secondaryGrantedEffect and g.secondaryGrantedEffect.support) and true or false)))
	end
end
out:write("}\n")

-- data.keystones (tree keystone names)
out:write("\nvar keystoneNames = []string{\n")
for _, name in ipairs(data.keystones) do
	out:write("\t", q(name), ",\n")
end
out:write("}\n")

-- data.clusterJewels
out:write("\nvar clusterJewelKeystones = []string{\n")
for _, name in ipairs(data.clusterJewels.keystones) do
	out:write("\t", q(name), ",\n")
end
out:write("}\n")

local notables = {}
for notable in pairs(data.clusterJewels.notableSortOrder) do
	notables[#notables + 1] = notable
end
table.sort(notables)
out:write("\nvar clusterJewelNotables = []string{\n")
for _, notable in ipairs(notables) do
	out:write("\t", q(notable), ",\n")
end
out:write("}\n")

-- cluster jewel skill enchant lines -> skill id. A few enchant lines appear on
-- more than one jewel size with different skill ids; the reference keeps
-- whichever pairs() visits last, which is arbitrary. The lexicographically
-- greatest id is chosen here so the output is deterministic; the table
-- comparison test flags it if the runtime table ever disagrees.
local enchants = {}
for baseName, jewel in pairs(data.clusterJewels.jewels) do
	for skillId, skill in pairs(jewel.skills) do
		local key = table.concat(skill.enchant, " "):lower()
		if enchants[key] == nil or skillId > enchants[key] then
			enchants[key] = skillId
		end
	end
end
local enchantLines = {}
for line in pairs(enchants) do
	enchantLines[#enchantLines + 1] = line
end
table.sort(enchantLines)
out:write("\nvar clusterJewelSkillEnchants = map[string]string{\n")
for _, line in ipairs(enchantLines) do
	out:write("\t", q(line), ": ", q(enchants[line]), ",\n")
end
out:write("}\n")

-- data.nonDamagingAilmentTypeList
out:write("\nvar nonDamagingAilmentTypeList = []string{")
for i, ailment in ipairs(data.nonDamagingAilmentTypeList) do
	if i > 1 then out:write(", ") end
	out:write(q(ailment))
end
out:write("}\n")

out:close()
io.stderr:write("vocab_gen.go written\n")

-- Reopen the file to append data.nonDamagingAilment defaults.
local out2 = assert(io.open("../../modparser/vocab_gen.go", "a"))
out2:write("\nvar nonDamagingAilmentDefault = map[string]float64{\n")
for _, ailment in ipairs(data.nonDamagingAilmentTypeList) do
	local info = data.nonDamagingAilment[ailment]
	out2:write(string.format("\t%q: %s,\n", ailment, tostring(info.default or 0)))
end
out2:write("}\n")
out2:close()
