-- League-legality validator. Run from src/:
--   luajit ../.claude/skills/cook/tools/validate.lua "Builds/My Build.xml"
-- Exit status is 0 only when it reports 0 problems.
--
-- Five checks, each with a stated source of truth:
--   1. gems      grantedEffect.legacy       -> Standard-only (GemTooltip.lua)
--   2. gem level naturalMaxLevel + 1 (every gem can be corrupted one level past max)
--   3. uniques   selected variant must be the last one, and not "no longer obtainable"
--   4. uniques   each hand-written mod line must exist on a current variant, in range
--   5. rare/magic each explicit mod must match a mod that can actually spawn on that base,
--                in range, by drop / bench / essence - plus the two silent affix failures
local BUILD = arg[1] or os.getenv("BUILDXML")
assert(BUILD, "usage: luajit validate.lua <path to build .xml>")
local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local build = pob.load(BUILD)
local data = pob.data()

local problems = 0
local function bad(fmt, ...)
	problems = problems + 1
	print("  !! " .. string.format(fmt, ...))
end

-- Split stat text into a literal skeleton + value specs, so a rolled line can be
-- matched against a range: "Adds (29-39) to (49-61) Cold Damage"
--   -> "Adds # to # Cold Damage", {{29,39},{49,61}}
local function parse(text)
	local specs = {}
	local skel = text:gsub("%(%-?[%d%.]+%-%-?[%d%.]+%)", function(r)
		local lo, hi = r:match("%((%-?[%d%.]+)%-(%-?[%d%.]+)%)")
		specs[#specs + 1] = { tonumber(lo), tonumber(hi) }
		return "#"
	end):gsub("%-?[%d%.]+", function(n)
		specs[#specs + 1] = { tonumber(n), tonumber(n) }
		return "#"
	end)
	return skel, specs
end
-- scalar widens the top of the range for a catalysed mod; see catalystScalar
local function inRange(specs, vals, scalar)
	if #specs ~= #vals then return false end
	for i, s in ipairs(specs) do
		local x = vals[i][1]
		local lo, hi = math.min(s[1], s[2]), math.max(s[1], s[2]) * (scalar or 1)
		if x < lo - 1e-9 or x > hi + 1e-9 then return false end
	end
	return true
end

-- Rings, amulets and belts can be catalysed, which multiplies every mod carrying a
-- matching tag by 1 + quality/100. A game export therefore shows values above the mod's
-- printed range: Mageblood's +(25-35) Strength reads +42 at Intrinsic 20. Mirrors the
-- file-local getCatalystScalar (Item.lua:31); catalystTags is its table verbatim.
local catalystTags = {
	{ "attack" }, { "speed" }, { "suffix" }, { "life", "mana", "resource" }, { "caster" },
	{ "jewellery_attribute", "attribute" }, { "physical_damage", "chaos_damage" },
	{ "jewellery_resistance", "resistance" }, { "prefix" },
	{ "jewellery_defense", "defences", "armour", "evasion", "energyshield" },
	{ "jewellery_elemental", "elemental_damage" }, { "critical" },
}
local function catalystScalar(item, modTags, modType)
	local tags = catalystTags[item.catalyst or 0]
	if not tags or not modTags then return 1 end
	local has = {}
	for _, t in ipairs(modTags) do has[t] = true end
	if modType then has[modType:lower()] = true end -- sinistral/dextral key off prefix/suffix
	for _, t in ipairs(tags) do
		if has[t] then return (100 + (item.catalystQuality or 20)) / 100 end
	end
	return 1
end
local function clean(line)
	return (line:gsub("{[^}]*}", ""):gsub("^%s+", ""):gsub("%s+$", ""))
end

------------------------------------------------------------------ 1+2. gems
print("=== GEMS ===")
for _, sg in ipairs(build.skillsTab.socketGroupList) do
	for _, gem in ipairs(sg.gemList) do
		local gd = gem.gemData
		if not gd then
			-- skills granted by an item or ascendancy are not socketed gems
			if gem.skillId or sg.source then
				print(string.format("  --  %-34s (granted by %s)", tostring(gem.nameSpec),
					tostring(sg.source or "ascendancy/item")))
			else
				bad("unrecognised gem: %s", tostring(gem.nameSpec))
			end
		else
			local ge, ge2 = gd.grantedEffect, gd.secondaryGrantedEffect
			if (ge and ge.legacy) or (ge2 and ge2.legacy) then
				bad("LEGACY GEM (Standard-only): %s   [group: %s]", gd.name, sg.label or "?")
			else
				local maxLvl = gd.naturalMaxLevel or 20
				local cap = maxLvl + 1 -- every gem can be corrupted one level past its natural max
				if gem.level > cap then
					bad("gem level %d exceeds max %d (natural %d, +1 corrupt): %s",
						gem.level, cap, maxLvl, gd.name)
				else
					print(string.format("  ok  %-34s lvl %2d/%2d  (natural max %d)", gd.name,
						gem.level, gem.quality, maxLvl))
				end
			end
		end
	end
end

------------------------------------------------------------------ 3+4. uniques
print("=== UNIQUES ===")

-- A variant is not a version. On Watcher's Eye each one is a different aura mod and
-- several are selected at once ("Discipline: ES Per Hit" is variant 24 of 106), so
-- "must be the last variant" is meaningless. PoB marks an actually-legacy roll in the
-- variant NAME - "Clarity: Mana Added As ES (Pre 3.12.0)" - so ask that instead, and
-- only look at the mod lines the selected variants really grant.
local uniqueIndex = {}
for _, list in pairs(data.uniques) do
	if type(list) == "table" then
		for _, raw in ipairs(list) do
			if type(raw) == "string" then
				local title = raw:match("^Rarity: UNIQUE\n([^\n]+)") or raw:match("^([^\n]+)")
				if title then
					local lines = uniqueIndex[title] or {}
					for line in raw:gmatch("[^\n]+") do
						local text = line:match("^{variant:[%d,]+}(.+)$")
						if not text and not line:match("^%a[%w ]*:") then text = line end
						if text then
							-- unique mod lines carry their own tags: {tags:attribute}
							local tags = {}
							for t in (line:match("{tags:([^}]+)}") or ""):gmatch("[^,]+") do
								tags[#tags + 1] = t
							end
							lines[#lines + 1] = { text = (text:gsub("{[^}]*}", "")), tags = tags }
						end
					end
					uniqueIndex[title] = lines
				end
			end
		end
	end
end

local function coveredBy(lines, myText, item)
	local mySkel, myVals = parse(myText)
	for _, cand in ipairs(lines) do
		local skel, specs = parse(cand.text)
		if skel == mySkel and inRange(specs, myVals, catalystScalar(item, cand.tags)) then
			return true
		end
	end
	return false
end
local function looksLegacy(name)
	return (name:match("[Pp]re %d") or name:match("[Ll]egacy")) ~= nil
end
local function checkUnique(label, item)
	if not item or item.rarity ~= "UNIQUE" then return end
	local vlist = item.variantList
	if vlist and #vlist > 0 then
		-- the primary pick plus every alternate the item actually uses (Item.lua:2117)
		local picked, names = { item.variant }, {}
		for i = 1, 5 do
			local sfx = i == 1 and "" or i
			if item["hasAltVariant" .. sfx] then picked[#picked + 1] = item["variantAlt" .. sfx] end
		end
		for _, v in ipairs(picked) do
			local name = v and vlist[v]
			if name then
				names[#names + 1] = name
				if looksLegacy(name) then
					bad("%s (%s): LEGACY variant selected: %q", label, item.title or "?", name)
				end
			end
		end
		print(string.format("  ok  %-12s %-22s %s", label, item.title or "?",
			table.concat(names, " + ")))
	else
		print(string.format("  ok  %-12s %-22s (no variants)", label, item.title or "?"))
	end
	local lines = uniqueIndex[item.title or ""]
	if not lines then
		print(string.format("  ??  %-12s %s: not in unique DB, mods unverifiable", label,
			item.title or "?"))
	else
		-- A rare-like unique rolls affixes on top of its fixed text, so its mod lines are
		-- not all in the unique DB. checkMods owns those.
		for _, v in ipairs(item.explicitModLines or {}) do
			local text = clean(v.line)
			if not item.rareLikeUnique and item:CheckModLineVariant(v)
				and not coveredBy(lines, text, item) then
				bad("%s (%s): mod not found on this unique: %q", label, item.title, text)
			end
		end
	end
	for _, line in ipairs(item.rawLines or {}) do
		if line:lower():match("no longer obtainable") or line:lower():match("removed from") then
			bad("%s (%s): %s", label, item.title or "?", line)
		end
	end
end

------------------------------------------------------------------ 5. rare/magic mods
-- Legality is ItemClass:GetModSpawnWeight (Item.lua:1654), NOT pool membership. Armour
-- bases have no dedicated pool, so Item.lua:1008 falls back to data.itemMods.Item - every
-- mod in the game. A mod can sit in the base's pool at weight 0 and be impossible to roll
-- there: that is how body-armour-only "+(91-100) to maximum Energy Shield" reached gloves.
--
-- Three legal routes onto a rare, each with its own gate:
--   drop     item.affixes and GetModSpawnWeight > 0    (influence-aware)
--   bench    data.masterMods, types[item.type]         (empty weightKey, so weight is 0)
--   essence  essence.mods[item.type]                   (weight 0 everywhere by design)
--
-- This checks generated text, so it covers both authoring styles: an affix-id item has
-- its lines written by Craft() from the real pool, and a weight-0 affix still fails here.
-- Rare-like uniques are craftable, so they come through here too. PoB already merges the
-- right pool into item.affixes - the full explicit pool for Dread Captain's Cutlass, the
-- veiled pool for The Crimson Storm, abyss jewel mods for Subsume the Source - and
-- CanHaveMod weighs a mod against rareLikeUnique.validBases (Item.lua:2668) rather than
-- the unique's own base, which is what lets a Cutlass roll One Handed Sword mods.
local function checkMods(label, item)
	if not item or not item.base then return end
	if item.rarity == "UNIQUE" and not item.rareLikeUnique then return end

	-- Affix-id authoring fails silently in two ways, so catch both here.
	-- An unknown modId is reset to "None" (Item.lua:1467) and vanishes from the item.
	for _, line in ipairs(item.rawLines or {}) do
		local spec = line:match("^Prefix: (.+)$") or line:match("^Suffix: (.+)$")
		if spec then
			local id = spec:gsub("^{fractured}", ""):gsub("^{range:[^}]*}", "")
			if id ~= "None" and not (item.affixes or {})[id] then
				bad("%s: unknown affix id %q - PoB drops it silently", label, id)
			end
		end
	end
	-- Craft() only reads up to the limit (Item.lua:2031), so extra affixes are ignored.
	if item.crafted then
		for _, ty in ipairs({ "prefixes", "suffixes" }) do
			local list, n = item[ty] or {}, 0
			for _, a in ipairs(list) do if a.modId and a.modId ~= "None" then n = n + 1 end end
			local lim = list.limit or (item.affixLimit and item.affixLimit / 2) or 3
			if n > lim then
				bad("%s: %d %s exceed the limit of %d - PoB ignores the extras", label, n, ty, lim)
			end
		end
	end

	local legal, blocked = {}, {}
	local function add(into, mod, via)
		for _, st in ipairs(mod) do
			if type(st) == "string" then
				local skel, specs = parse(st)
				into[skel] = into[skel] or {}
				table.insert(into[skel], { specs = specs, text = st, via = via,
					tags = mod.modTags, ty = mod.type, group = mod.group })
			end
		end
	end
	local rlu = item.rareLikeUnique ~= nil
	for _, mod in pairs(item.affixes or {}) do
		if type(mod) == "table" then
			-- The veiled and abyss pools carry no weightKey: they are already specific to
			-- the item, so membership is the whole test. Everything else must weigh in.
			local ok = mod.weightKey and item:CanHaveMod(mod) or (not mod.weightKey and rlu)
			add(ok and legal or blocked, mod, rlu and "unique pool" or "drop")
		end
	end
	-- a rare-like unique's own fixed lines are in the unique DB, not the affix pool
	if rlu then
		for _, cand in ipairs(uniqueIndex[item.title or ""] or {}) do
			local skel, specs = parse(cand.text)
			legal[skel] = legal[skel] or {}
			table.insert(legal[skel], { specs = specs, text = cand.text, via = "unique",
				tags = cand.tags })
		end
	end
	-- bench mods are indexed separately as well: a {crafted} line has to be matched against
	-- this pool specifically, since its text often also exists as a drop mod
	local benchIndex = {}
	for _, craft in ipairs(data.masterMods or {}) do
		if craft.types and craft.types[item.type] then
			add(legal, craft, "bench")
			add(benchIndex, craft, "bench")
		end
	end
	for _, e in pairs(data.essences or {}) do
		local mod = e.mods and e.mods[item.type] and (item.affixes or {})[e.mods[item.type]]
		if mod then add(legal, mod, "essence") end
	end

	-- of the candidates sharing a skeleton, the one reaching the highest value
	local function bestOf(list)
		local best
		for _, c in ipairs(list or {}) do
			if not best or (c.specs[1] and best.specs[1] and c.specs[1][2] > best.specs[1][2]) then
				best = c
			end
		end
		return best and best.text
	end
	-- One item cannot carry two affixes from the same group; PoB's craft UI filters the
	-- list by group (ItemsTab.lua:3411) and blocks a bench craft whose group is taken.
	-- Subsume the Source is the only thing allowed to repeat one.
	local groups = {}
	local function claimGroup(group, who)
		if not group or (item.rareLikeUnique and item.rareLikeUnique.allowDuplicateGroups) then
			return
		end
		if groups[group] then
			bad("%s: two affixes share group %q: %s and %s", label, group, groups[group], who)
		else
			groups[group] = who
		end
	end
	local function checkLine(text)
		local skel, vals = parse(text)
		local hit
		for _, c in ipairs(legal[skel] or {}) do
			if inRange(c.specs, vals, catalystScalar(item, c.tags, c.ty)) then hit = c break end
		end
		if hit then
			print(string.format("  ok  %-12s %-56s [%s]", label, text, hit.via))
			return hit
		elseif legal[skel] then
			bad("%s: %-56s out of range, best on this base: %s", label, text, bestOf(legal[skel]))
		elseif blocked[skel] then
			bad("%s: %-56s CANNOT ROLL on %s (spawn weight 0): %s", label, text,
				item.baseName or "?", bestOf(blocked[skel]))
		else
			bad("%s: %-56s no such mod on this base", label, text)
		end
	end

	if item.crafted then
		-- Affix-authored: check the ids, not the text. Craft() sums two mods that share a
		-- statOrder into one line (Item.lua:2046), so Incorporeal + Hummingbird's renders as
		-- a single "152% increased Evasion and Energy Shield" that matches no single mod.
		-- The ids are unambiguous, so text matching is not needed and would misfire.
		for _, ty in ipairs({ "prefixes", "suffixes" }) do
			for _, affix in ipairs(item[ty] or {}) do
				local mod = affix.modId ~= "None" and (item.affixes or {})[affix.modId]
				if mod then
					local lines = {}
					for _, s in ipairs(mod) do if type(s) == "string" then lines[#lines + 1] = s end end
					local stat = table.concat(lines, " / ")
					if mod.weightKey and not item:CanHaveMod(mod) then
						bad("%s: %s CANNOT ROLL on %s (spawn weight 0): %s", label, affix.modId,
							item.baseName or "?", stat)
					else
						print(string.format("  ok  %-12s %-44s %s", label, affix.modId, stat))
					end
					claimGroup(mod.group, affix.modId)
				end
			end
		end
		-- bench crafts and hand-added lines survive Craft() and are still only text
		local benches, multicraft = 0, false
		for _, v in ipairs(item.explicitModLines or {}) do
			if v.crafted or v.custom then
				local text = clean(v.line)
				checkLine(text)
				if text == "Can have up to 3 Crafted Modifiers" then
					multicraft = true
				else
					local skel, vals = parse(text)
					for _, c in ipairs(benchIndex[skel] or {}) do
						if inRange(c.specs, vals) then
							benches = benches + 1
							claimGroup(c.group, "bench craft: " .. text)
							break
						end
					end
				end
			end
		end
		-- one bench craft unless the "of Crafting" suffix is on the item, then three
		local benchLimit = multicraft and 3 or 1
		if benches > benchLimit then
			bad("%s: %d bench crafts, limit is %d%s", label, benches, benchLimit,
				multicraft and "" or " without \"Can have up to 3 Crafted Modifiers\"")
		end
	else
		for _, v in ipairs(item.explicitModLines or {}) do checkLine(clean(v.line)) end
	end
end

------------------------------------------------------------------ walk every slot
local order = { "Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet",
	"Ring 1", "Ring 2", "Belt", "Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5" }
for _, s in ipairs(order) do
	local slot = build.itemsTab.slots[s]
	if slot then checkUnique(s, build.itemsTab.items[slot.selItemId or 0]) end
end
for nodeId, sock in pairs(build.itemsTab.sockets or {}) do
	checkUnique("Jewel@" .. nodeId, build.itemsTab.items[sock.selItemId or 0])
end
print("=== RARE / MAGIC MODS ===")
for _, s in ipairs(order) do
	local slot = build.itemsTab.slots[s]
	if slot then checkMods(s, build.itemsTab.items[slot.selItemId or 0]) end
end
for nodeId, sock in pairs(build.itemsTab.sockets or {}) do
	checkMods("Jewel@" .. nodeId, build.itemsTab.items[sock.selItemId or 0])
end

print(string.format("\n=== TOTAL PROBLEMS: %d ===", problems))
os.exit(problems == 0 and 0 or 1)
