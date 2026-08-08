-- Legal affix pool for an item base. Run from src/:
--   luajit ../.claude/skills/cook/tools/affixes.lua "Sorcerer Gloves" [pattern] [flags]
--   luajit ../.claude/skills/cook/tools/affixes.lua "Hubris Circlet" "Energy Shield" --tiers
--
-- Legality is PoB's own ItemClass:GetModSpawnWeight (Item.lua:1654), never pool membership.
-- Armour bases have no dedicated pool, so Item.lua:1008 falls back to data.itemMods.Item -
-- every mod in the game. "In the pool" therefore means nothing: LocalIncreasedEnergyShield11
-- (+91-100 ES) is in the Gloves pool at weight 0 and cannot roll there. Weight is the filter.
--
-- Print modIds, then author the item with them (see SKILL.md) so PoB writes the values.
--
-- flags: --tiers   every tier, not just the highest per group
--        --shaper --elder --adjudicator --basilisk --crusader --eyrie --cleansing --tangle
local INFLUENCE = { "shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie",
	"cleansing", "tangle" }

-- Parse args before loading PoB: HeadlessWrapper clears the global `arg`.
local baseName, pattern, flags = nil, nil, {}
for i = 1, #arg do
	local f = arg[i]:match("^%-%-(.+)")
	if f then flags[f] = true
	elseif not baseName then baseName = arg[i]
	else pattern = arg[i] end
end
assert(baseName, 'usage: luajit affixes.lua "<base name>" [pattern] [--tiers] [--shaper ...]')

local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local data = pob.data()

local item = new("Item", "Rarity: RARE\nAffix Probe\n" .. baseName .. "\n")
assert(item.base, "unknown base: " .. baseName)
local influences = {}
for _, k in ipairs(INFLUENCE) do
	if flags[k] then item[k] = true; influences[#influences + 1] = k end
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
print(string.format("  %s tier per group%s\n", flags.tiers and "every" or "highest",
	pattern and (", matching " .. string.format("%q", pattern)) or ""))

------------------------------------------------------------------ drop pool
local byType = { Prefix = {}, Suffix = {} }
for id, mod in pairs(item.affixes or {}) do
	if type(mod) == "table" and byType[mod.type] and item:GetModSpawnWeight(mod) > 0
		and matches(mod, id) then
		table.insert(byType[mod.type], { id = id, mod = mod })
	end
end
for _, ty in ipairs({ "Prefix", "Suffix" }) do
	local list = byType[ty]
	if not flags.tiers then -- one entry per group, the highest ilvl tier
		local best = {}
		for _, e in ipairs(list) do
			local g = e.mod.group or e.id
			if not best[g] or (e.mod.level or 0) > (best[g].mod.level or 0) then best[g] = e end
		end
		list = {}
		for _, e in pairs(best) do list[#list + 1] = e end
	end
	table.sort(list, function(a, b)
		local ga, gb = a.mod.group or a.id, b.mod.group or b.id
		if ga ~= gb then return ga < gb end
		return (a.mod.level or 0) < (b.mod.level or 0)
	end)
	print(string.format("## %ses (%d)", ty, #list))
	for _, e in ipairs(list) do
		print(string.format("  ilvl %-3d  %-44s %-18s %s", e.mod.level or 0, e.id,
			e.mod.affix or "", stats(e.mod)))
	end
	print("")
end

------------------------------------------------------------------ bench crafts
-- ItemsTab.lua:3421 gates these on craft.types[item.type], and blocks any craft whose
-- group already exists on the item. Bench mods have an empty weightKey, so spawn weight
-- is always 0 for them - they are legal by a different route and must be checked here.
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
-- essence.mods maps item type -> modId. Essence mods sit at weight 0 everywhere: they
-- never roll, so they are only obtainable by using the essence.
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
