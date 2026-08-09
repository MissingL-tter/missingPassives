-- Legal affix pool for an item base. Run from src/:
--   luajit ../.claude/skills/cook/tools/affixes.lua "Sorcerer Gloves" [pattern] [flags]
--   luajit ../.claude/skills/cook/tools/affixes.lua "Hubris Circlet" "Energy Shield" --tiers
--
-- Legality is PoB's own ItemClass:GetModSpawnWeight (Item.lua:1654), never pool membership.
-- Armour bases have no dedicated pool, so Item.lua:1008 falls back to data.itemMods.Item -
-- every mod in the game. "In the pool" therefore means nothing: LocalIncreasedEnergyShield11
-- (+91-100 ES) is in the Gloves pool at weight 0 and cannot roll there. Weight is the filter.
--
-- Prints modIds; author the item with them (see SKILL.md) so PoB writes the values.
-- Default listing: top two tiers per group. Author the T2 at {range:1} (realistic; true T1
-- left as upgrade room). Use T1 only when the recipe cannot be met without it, capped at
-- {range:0.85}, and say so in the report.
--
-- flags: --tiers   every tier, not just the top two per group
--        --shaper --elder --adjudicator --basilisk --crusader --eyrie --cleansing --tangle
local INFLUENCE = { "shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie",
	"cleansing", "tangle" }

-- Parse args before loading PoB: HeadlessWrapper clears the global `arg`.
local baseName, pattern, flags, skillTag = nil, nil, {}, nil
for i = 1, #arg do
	local s = arg[i]:match("^%-%-skill=(.+)")
	local f = arg[i]:match("^%-%-(.+)")
	if s then skillTag = s
	elseif f then flags[f] = true
	elseif not baseName then baseName = arg[i]
	else pattern = arg[i] end
end
assert(baseName, 'usage: luajit affixes.lua "<base name>" [pattern] [--tiers] [--shaper ...]'
	.. ' [--skill=<cluster skill tag>]')

local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local data = pob.data()

local item = new("Item", "Rarity: RARE\nAffix Probe\n" .. baseName .. "\n")
assert(item.base, "unknown base: " .. baseName)
local influences = {}
for _, k in ipairs(INFLUENCE) do
	if flags[k] then item[k] = true; influences[#influences + 1] = k end
end

-- Cluster jewel notables are gated on the jewel's enchant skill tag, so the pool is
-- meaningless without one. No --skill: list the base's skills, stop.
local extraTags = {}
if item.clusterJewel then
	if not skillTag or not item.clusterJewel.skills[skillTag] then
		print(string.format("# %s needs --skill=<tag>; its skills:", baseName))
		local tags = {}
		for id in pairs(item.clusterJewel.skills) do tags[#tags + 1] = id end
		table.sort(tags)
		for _, id in ipairs(tags) do
			local sk = item.clusterJewel.skills[id]
			print(string.format("  %-42s %s", id, table.concat(sk.stats or {}, " / ")))
		end
		os.exit(skillTag and 1 or 0)
	end
	item.clusterJewelSkill = skillTag
	extraTags[item.clusterJewel.skills[skillTag].tag] = true
end

local function matches(mod, id)
	if not pattern then return true end
	local p = pattern:lower()
	if id:lower():find(p, 1, true) or (mod.affix or ""):lower():find(p, 1, true) then return true end
	for _, s in ipairs(mod) do
		if type(s) == "string" and s:lower():find(p, 1, true) then return true end
	end
	return false
end
local function stats(mod)
	local out = {}
	for _, s in ipairs(mod) do if type(s) == "string" then out[#out + 1] = s end end
	return table.concat(out, " / ")
end

------------------------------------------------------------------ header
local tags = {}
for k in pairs(item.base.tags or {}) do tags[#tags + 1] = k end
table.sort(tags)
print(string.format("# %s  (type %s)", baseName, tostring(item.type)))
print(string.format("  tags: %s", table.concat(tags, ", ")))
if #influences > 0 then print(string.format("  influence: %s", table.concat(influences, ", "))) end
print(string.format("  %s per group%s - author the T2 at {range:1}\n",
	flags.tiers and "every tier" or "top two tiers",
	pattern and (", matching " .. string.format("%q", pattern)) or ""))

------------------------------------------------------------------ drop pool
-- Elevated (Maven-crafted) mods are omitted entirely: legal in-game, but policy is never to
-- use them (too expensive to assume), and validate.lua rejects them anyway.
local byType = { Prefix = {}, Suffix = {} }
for id, mod in pairs(item.affixes or {}) do
	if type(mod) == "table" and byType[mod.type] and item:GetModSpawnWeight(mod, extraTags) > 0
		and not (mod.affix or ""):match("Elevated") and matches(mod, id) then
		table.insert(byType[mod.type], { id = id, mod = mod })
	end
end
for _, ty in ipairs({ "Prefix", "Suffix" }) do
	local list = byType[ty]
	if not flags.tiers then
		-- Top two tiers per group; the T2/{range:1} convention is in the header comment.
		local byGroup = {}
		for _, e in ipairs(list) do
			local g = e.mod.group or e.id
			byGroup[g] = byGroup[g] or {}
			table.insert(byGroup[g], e)
		end
		list = {}
		for _, tiers in pairs(byGroup) do
			table.sort(tiers, function(a, b) return (a.mod.level or 0) > (b.mod.level or 0) end)
			for i = 1, math.min(2, #tiers) do
				tiers[i].tier = "T" .. i
				list[#list + 1] = tiers[i]
			end
		end
	end
	table.sort(list, function(a, b)
		local ga, gb = a.mod.group or a.id, b.mod.group or b.id
		if ga ~= gb then return ga < gb end
		return (a.mod.level or 0) > (b.mod.level or 0)
	end)
	print(string.format("## %ses (%d)", ty, #list))
	for _, e in ipairs(list) do
		print(string.format("  %-3s ilvl %-3d  %-44s %-18s %s", e.tier or "", e.mod.level or 0,
			e.id, e.mod.affix or "", stats(e.mod)))
	end
	print("")
end

------------------------------------------------------------------ bench crafts
-- ItemsTab.lua:3421 gates these on craft.types[item.type], and blocks any craft whose group
-- already exists on the item. Bench mods have an empty weightKey, so their spawn weight is
-- always 0 - legal by a different route, so they must be listed here.
local bench = {}
for _, craft in ipairs(data.masterMods) do
	if craft.types and craft.types[item.type] and matches(craft, craft.affix or "") then
		bench[#bench + 1] = craft
	end
end
table.sort(bench, function(a, b)
	if a.type ~= b.type then return a.type == "Prefix" end
	return stats(a) < stats(b)
end)
print(string.format("## Bench crafts (%d)   - write as {crafted} text lines", #bench))
for _, c in ipairs(bench) do
	print(string.format("  %-7s ilvl %-3d  %s", c.type or "", c.level or 0, stats(c)))
end
print("")

------------------------------------------------------------------ essences
-- essence.mods maps item type -> modId. Essence mods sit at weight 0 everywhere: they never
-- roll, so they are only obtainable by using the essence.
local ess = {}
for _, e in pairs(data.essences or {}) do
	local id = e.mods and e.mods[item.type]
	local mod = id and item.affixes and item.affixes[id]
	if mod and matches(mod, id) then
		ess[#ess + 1] = { name = e.name, id = id, mod = mod }
	end
end
table.sort(ess, function(a, b) return a.name < b.name end)
print(string.format("## Essences (%d)", #ess))
for _, e in ipairs(ess) do
	print(string.format("  %-30s %-7s %-40s %s", e.name, e.mod.type or "", e.id, stats(e.mod)))
end
