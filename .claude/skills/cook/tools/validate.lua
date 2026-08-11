-- League-legality validator. Run from src/:
--   luajit ../.claude/skills/cook/tools/validate.lua "Builds/My Build.xml"
-- Exit status is 0 only when it reports 0 problems.
--
-- Checks, each with its source of truth:
--   gems        grantedEffect.legacy -> Standard-only (GemTooltip.lua); level cap is
--               naturalMaxLevel + 1 (one corruption level, no exceptions)
--   uniques     no legacy-named variant selected; every active mod line must exist on a
--               current variant, in range
--   rare/magic  every explicit mod must match a mod that can actually spawn on that base,
--               in range, by drop / bench / essence; no shared mod groups; bench craft
--               limits; no elevated mods; no Scourge mods (gone from the game, but still
--               in the pools at full weight); T1 rolls capped at 0.85; plus the silent
--               affix-authoring failures (unknown id dropped, affixes past the limit
--               ignored, authored-but-never-crafted items contributing nothing)
--   enchants    no labyrinth enchants (removed from the game; PoB keeps the tables)
--   tree        PointsUsed / AscUsed / SecondaryAscUsed within Build.lua:884's budgets
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

-- Split stat text into a literal skeleton + value specs so a rolled line can be matched
-- against a range: "Adds (29-39) to (49-61) Cold Damage"
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

-- Rings, amulets and belts can be catalysed: every mod carrying a matching tag is multiplied
-- by 1 + quality/100, so a game export shows values above the mod's printed range -
-- Mageblood's +(25-35) Strength reads +42 at Intrinsic 20. Mirrors the file-local
-- getCatalystScalar (Item.lua:31); catalystTags is its table verbatim.
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

------------------------------------------------------------------ gems
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

------------------------------------------------------------------ uniques
print("=== UNIQUES ===")

-- Limited-to enforcement: PoB silently sets jewelData.limitDisabled on any socketed jewel
-- over its "Limited to" cap (the Historic cross-item cap of 1 included) - the jewel then
-- contributes nothing, with no warning anywhere in the UI or calcs.
for nodeId, itemId in pairs(build.spec.jewels or {}) do
	local item = build.itemsTab.items[itemId]
	if item and item.jewelData and item.jewelData.limitDisabled then
		bad("Jewel@%d %s: ILLEGAL - exceeds its 'Limited to' cap (a character cannot equip it); PoB silently disables it",
			nodeId, tostring(item.name))
	end
end

-- A variant is not a version. On Watcher's Eye each is a different aura mod and several are
-- selected at once ("Discipline: ES Per Hit" is variant 24 of 106), so "must be the last
-- variant" is meaningless. PoB marks an actually-legacy roll in the variant NAME - "Clarity:
-- Mana Added As ES (Pre 3.12.0)" - so ask that instead, and only look at the mod lines the
-- selected variants really grant.
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

------------------------------------------------------------------ rare/magic mods
-- Legality is ItemClass:GetModSpawnWeight (Item.lua:1654), NOT pool membership. Armour bases
-- have no dedicated pool, so Item.lua:1008 falls back to data.itemMods.Item - every mod in
-- the game. A mod can sit in the base's pool at weight 0 and be impossible to roll there:
-- that is how body-armour-only "+(91-100) to maximum Energy Shield" reached gloves.
--
-- Three legal routes onto a rare, each with its own gate:
--   drop     item.affixes and GetModSpawnWeight > 0    (influence-aware)
--   bench    data.masterMods, types[item.type]         (empty weightKey, so weight is 0)
--   essence  essence.mods[item.type]                   (weight 0 everywhere by design)
--
-- Checks generated text, so it covers both authoring styles: an affix-id item has its lines
-- written by Craft() from the real pool, and a weight-0 affix still fails here. Rare-like
-- uniques are craftable, so they come through too. PoB already merges the right pool into
-- item.affixes - full explicit pool for Dread Captain's Cutlass, veiled pool for The Crimson
-- Storm, abyss jewel mods for Subsume the Source - and CanHaveMod weighs a mod against
-- rareLikeUnique.validBases (Item.lua:2668) rather than the unique's own base, which is what
-- lets a Cutlass roll One Handed Sword mods.
local function checkMods(label, item)
	if not item or not item.base then return end
	if item.rarity == "UNIQUE" and not item.rareLikeUnique then return end

	-- Cluster jewel notables are gated on the jewel's enchant skill tag, which is not a base
	-- tag - the craft UI passes it as an extra include tag (ItemsTab.lua:2150). Without this
	-- every legal notable reads as spawn weight 0.
	local extraTags = {}
	if item.clusterJewel and item.clusterJewelSkill then
		local skill = item.clusterJewel.skills[item.clusterJewelSkill]
		if skill then extraTags[skill.tag] = true end
	end

	-- Affix-id authoring fails silently in two ways, so catch both here.
	-- An unknown modId is reset to "None" (Item.lua:1467) and vanishes from the item.
	for _, line in ipairs(item.rawLines or {}) do
		local spec = line:match("^Prefix: (.+)$") or line:match("^Suffix: (.+)$")
		if spec then
			local id = spec:gsub("^{fractured}", ""):gsub("^{range:[^}]*}", "")
			local mod = (item.affixes or {})[id]
			if id ~= "None" and not mod then
				bad("%s: unknown affix id %q - PoB drops it silently", label, id)
			end
			-- Elevated (Maven-crafted) influence mods are legal but so expensive that a
			-- build resting on one is not a build anyone can assemble.
			if mod and (mod.affix or ""):match("Elevated") then
				bad("%s: %s is an elevated mod (%q) - legal but too expensive, use the "
					.. "unelevated tier", label, id, mod.affix)
			end
			-- Scourge mods sit in the pools at full weight but are gone from the game.
			if mod and data.itemMods.Scourge[id] then
				bad("%s: %s is a Scourge mod - Scourge is gone from the game", label, id)
			end
			-- Realism policy: author the T2 tier at {range:1}. T1 is allowed when the recipe
			-- demands it, but never above {range:0.85} - a perfectly rolled top tier is not
			-- an item anyone assembles. Only drop mods have tiers, so bench and essence mods
			-- (empty weightKey) are exempt, as are fixed-value mods ("+1 to Level of all
			-- Skill Gems") where there is nothing to roll.
			local rolls = false
			for _, st in ipairs(mod or {}) do
				if type(st) == "string" and st:match("%(%-?[%d%.]+%-%-?[%d%.]+%)") then
					rolls = true
					break
				end
			end
			if mod and rolls and mod.weightKey and mod.group then
				-- Same type only: Scourge "HellscapeUpside" mods share drop-mod groups at
				-- weight 1000 but are neither Prefix nor Suffix, and would mask the real
				-- top tier (they hid Seething on gloves).
				local topLevel = mod.level or 0
				for _, other in pairs(item.affixes) do
					if type(other) == "table" and other.group == mod.group
						and other.type == mod.type
						and other.weightKey and item:GetModSpawnWeight(other, extraTags) > 0 then
						topLevel = math.max(topLevel, other.level or 0)
					end
				end
				if (mod.level or 0) >= topLevel then
					local worst = 0
					for r in (spec:match("{range:([^}]+)}") or "0.5"):gmatch("[^,]+") do
						worst = math.max(worst, tonumber(r) or 0.5)
					end
					if worst > 0.85 then
						bad("%s: %s is the top tier at {range:%.2f} - T1 rolls cap at 0.85",
							label, id, worst)
					end
				end
			end
		end
	end
	-- Craft() only reads up to the limit (Item.lua:2031), so extra affixes are ignored.
	if item.crafted then
		local named = 0
		for _, ty in ipairs({ "prefixes", "suffixes" }) do
			local list, n = item[ty] or {}, 0
			for _, a in ipairs(list) do if a.modId and a.modId ~= "None" then n = n + 1 end end
			named = named + n
			local lim = list.limit or (item.affixLimit and item.affixLimit / 2) or 3
			if n > lim then
				bad("%s: %d %s exceed the limit of %d - PoB ignores the extras", label, n, ty, lim)
			end
		end
		-- Craft() is UI-only (ItemsTab.lua:808), never run on load. An authored item with
		-- affix ids but no generated mod text parses and equips fine while adding NOTHING to
		-- the calcs, so every measurement lies. tools/craft.lua bakes the text in.
		if named > 0 then
			local generated = 0
			for _, v in ipairs(item.explicitModLines or {}) do
				if v.prefix or v.suffix then generated = generated + 1 end
			end
			if generated == 0 then
				bad("%s: %d affixes but no generated mod text - item is INERT in calcs, "
					.. "run tools/craft.lua", label, named)
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
	-- Scourge mods are gone from the game but still sit in the pools at full spawn weight
	-- (data.itemMods.Scourge is merged into itemMods.Item), so weight alone would accept
	-- them. Divert to their own bucket for a precise error.
	local removed = {}
	local rlu = item.rareLikeUnique ~= nil
	for id, mod in pairs(item.affixes or {}) do
		if type(mod) == "table" then
			if data.itemMods.Scourge[id] then
				add(removed, mod, "scourge")
			else
				-- The veiled and abyss pools carry no weightKey: they are already specific to
				-- the item, so membership is the whole test. Everything else must weigh in.
				local ok = mod.weightKey and item:CanHaveMod(mod, extraTags)
					or (not mod.weightKey and rlu)
				add(ok and legal or blocked, mod, rlu and "unique pool" or "drop")
			end
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
	-- bench mods are also indexed separately: a {crafted} line must be matched against this
	-- pool specifically, since its text often also exists as a drop mod
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
	-- One item cannot carry two affixes from the same group; PoB's craft UI filters the list
	-- by group (ItemsTab.lua:3411) and blocks a bench craft whose group is taken. Subsume the
	-- Source is the only thing allowed to repeat one.
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
		elseif removed[skel] then
			bad("%s: %-56s only matches a Scourge mod - Scourge is gone from the game",
				label, text)
		elseif blocked[skel] then
			bad("%s: %-56s CANNOT ROLL on %s (spawn weight 0): %s", label, text,
				item.baseName or "?", bestOf(blocked[skel]))
		else
			bad("%s: %-56s no such mod on this base", label, text)
		end
	end

	if item.crafted then
		-- Affix-authored: check the ids, not the text. Craft() sums two mods sharing a
		-- statOrder into one line (Item.lua:2046), so Incorporeal + Hummingbird's renders as a
		-- single "152% increased Evasion and Energy Shield" matching no single mod. The ids
		-- are unambiguous, so text matching is not needed and would misfire.
		for _, ty in ipairs({ "prefixes", "suffixes" }) do
			for _, affix in ipairs(item[ty] or {}) do
				local mod = affix.modId ~= "None" and (item.affixes or {})[affix.modId]
				if mod then
					local lines = {}
					for _, s in ipairs(mod) do if type(s) == "string" then lines[#lines + 1] = s end end
					local stat = table.concat(lines, " / ")
					if mod.weightKey and not item:CanHaveMod(mod, extraTags) then
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

-- Labyrinth enchants are gone from the game, but PoB still ships the tables for old builds,
-- so nothing else rejects them. Slot is the wrong test: the legal sources (Harvest, Heist,
-- Dedication, Instilling / Enkindling, amulet anoints) use the same {enchant} tag.
-- data.enchantments (Data.lua:665) is keyed by source, so ask it instead. The lab keys are
-- its four difficulties, NORMAL included - on gloves that tier is "Trigger Word of ...",
-- climbing to Edict / Decree / Commandment, all of it lab. Non-lab sources pass untouched.
local LAB_KEYS = { NORMAL = true, CRUEL = true, MERCILESS = true, ENDGAME = true }
local labEnchants = {}
for _, byType in pairs(data.enchantments or {}) do
	for key, list in pairs(byType) do
		if LAB_KEYS[key] then
			for _, text in ipairs(list) do labEnchants[text] = true end
		end
	end
end

local function checkEnchants(label, item)
	if not item then return end
	for _, line in ipairs(item.rawLines or {}) do
		if line:match("^{enchant}") then
			local text = line:gsub("^{enchant}", ""):gsub("^{crafted}", "")
			if labEnchants[text] then
				bad("%s: labyrinth enchant %q - no longer in the game", label, text)
			end
		end
	end
end

------------------------------------------------------------------ eldritch implicits
-- influences.md, enforced: an ELDRITCH ITEM (cleansing/tangle mark or any eldritch implicit)
-- and an influenced item are mutually exclusive categories; at most one Exarch + one Eater
-- implicit per item, armour slots only, no coexistence with base implicits, both sides t5+
-- impossible. A line is treated as eldritch when it text-matches a ModEldritch entry after
-- the base's own implicit gets first claim. Uniques are skipped: the few with eldritch
-- lines ship that way.
local ELDRITCH_SLOTS = { ["Body Armour"] = true, ["Helmet"] = true, ["Gloves"] = true,
	["Boots"] = true }
local INFLUENCES = { "shaper", "elder", "crusader", "basilisk", "eyrie", "adjudicator" }
local eldritchIndex = {}
for id, mod in pairs(data.itemMods.Eldritch or {}) do
	if type(mod) == "table" and (mod.type == "Exarch" or mod.type == "Eater") then
		local tier = tonumber(tostring(id):match("(%d+)_*$")) or 0
		for _, st in ipairs(mod) do
			if type(st) == "string" then
				local skel, specs = parse(st)
				eldritchIndex[skel] = eldritchIndex[skel] or {}
				table.insert(eldritchIndex[skel],
					{ specs = specs, side = mod.type, tier = tier, id = id })
			end
		end
	end
end
-- Influence end-states (influences.md): at most two different influences per item (two only
-- exists via Awakener's Orb); checkEldritch owns the eldritch/influence exclusion.
-- Spawn-weight legality of the influenced MODS is already GetModSpawnWeight's job in
-- checkMods; uniques are skipped - a few (Disintegrator et al) carry influence flags by
-- design. Fracturing is deliberately NOT checked: it is a crafting route, the user's domain.
local ELDRITCH_FLAGS = { "cleansing", "tangle" }
local function checkInfluences(label, item)
	if not item or not item.base or item.rarity == "UNIQUE" then return end
	local flags = {}
	for _, inf in ipairs(INFLUENCES) do
		if item[inf] then flags[#flags + 1] = inf end
	end
	if #flags > 2 then
		bad("%s: %d influences (%s) - the cap is two, and two only via Awakener's Orb",
			label, #flags, table.concat(flags, ", "))
	end
end
local function checkEldritch(label, item)
	if not item or not item.base or item.rarity == "UNIQUE" then return end
	-- the base's own implicit takes precedence: "15% increased Spell Damage" on an Imbued
	-- Wand is the base implicit even though the text also matches a helmet eldritch mod
	local baseImp = {}
	for line in tostring(item.base.implicit or ""):gmatch("[^\n]+") do
		local skel, specs = parse(line)
		baseImp[#baseImp + 1] = { skel = skel, specs = specs }
	end
	local count, tier = {}, {}
	local plainImplicits = 0
	for _, v in ipairs(item.implicitModLines or {}) do
		local text = clean(v.line)
		local skel, vals = parse(text)
		local isBase = false
		for _, b in ipairs(baseImp) do
			if b.skel == skel and inRange(b.specs, vals) then isBase = true break end
		end
		local best -- lowest tier whose range fits: the charitable reading
		if not isBase then
			for _, c in ipairs(eldritchIndex[skel] or {}) do
				if inRange(c.specs, vals) and (not best or c.tier < best.tier) then best = c end
			end
		end
		if best then
			count[best.side] = (count[best.side] or 0) + 1
			tier[best.side] = math.max(tier[best.side] or 0, best.tier)
		else
			plainImplicits = plainImplicits + 1
		end
	end
	local marks = {}
	for _, m in ipairs(ELDRITCH_FLAGS) do
		if item[m] then marks[#marks + 1] = m end
	end
	local hasLines = (count.Exarch or count.Eater) ~= nil
	if not hasLines and #marks == 0 then return end
	-- an ELDRITCH ITEM (marked cleansing/tangle or carrying eldritch implicits) and an
	-- influenced item are mutually exclusive categories, in both directions
	for _, inf in ipairs(INFLUENCES) do
		if item[inf] then
			bad("%s: eldritch item carries %s influence - influenced items cannot be eldritch and vice versa",
				label, inf)
		end
	end
	if not hasLines then return end
	for _, side in ipairs({ "Exarch", "Eater" }) do
		if (count[side] or 0) > 1 then
			bad("%s: %d %s implicits - an item holds at most one per side", label, count[side], side)
		end
	end
	if not ELDRITCH_SLOTS[item.base.type or ""] then
		bad("%s: eldritch implicit on a %s - only Body Armour/Helmet/Gloves/Boots can have them",
			label, item.base.type or "?")
	end
	if plainImplicits > 0 then
		bad("%s: eldritch implicit alongside %d other implicit(s) - eldritch REPLACES the base implicit",
			label, plainImplicits)
	end
	if (tier.Exarch or 0) >= 5 and (tier.Eater or 0) >= 5 then
		bad("%s: both eldritch implicits t5+ (t%d/t%d) - IMPOSSIBLE, cap is t6/t4 or t5/t4",
			label, tier.Exarch, tier.Eater)
	else
		print(string.format("  ok  %-12s eldritch %s%s", label,
			count.Exarch and ("Exarch t" .. tier.Exarch) or "-",
			count.Eater and (" + Eater t" .. tier.Eater) or ""))
	end
end

------------------------------------------------------------------ walk every slot
local order = { "Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet",
	"Ring 1", "Ring 2", "Belt", "Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5" }
for _, s in ipairs(order) do
	local slot = build.itemsTab.slots[s]
	if slot then
		local item = build.itemsTab.items[slot.selItemId or 0]
		checkUnique(s, item)
		checkEnchants(s, item)
	end
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
print("=== ELDRITCH / INFLUENCE ===")
for _, s in ipairs(order) do
	local slot = build.itemsTab.slots[s]
	if slot then
		local item = build.itemsTab.items[slot.selItemId or 0]
		checkEldritch(s, item)
		checkInfluences(s, item)
	end
end

------------------------------------------------------------------ build state
-- Impossible-character checks: PoB computes happily through all of these and reports them
-- only as quiet numbers, so each must be an error here.
print("=== BUILD STATE ===")
do
	local out = build.calcsTab.mainOutput or {}
	local before = problems
	for _, a in ipairs({ "Str", "Dex", "Int" }) do
		local have, need = out[a] or 0, out["Req" .. a] or 0
		if need > have then
			bad("attributes: %d %s required, only %d - gems could not level or function",
				need, a, have)
		end
	end
	if (out.ManaUnreserved or 0) < 0 then
		bad("reservations exceed mana pool by %d - this aura set cannot all be active",
			-out.ManaUnreserved)
	end
	if (out.LifeUnreserved or 1) < 0 then
		bad("reservations exceed life pool by %d", -out.LifeUnreserved)
	end
	for id, node in pairs(build.spec.allocNodes or {}) do
		if node.type == "Socket" then
			local itemId = build.spec.jewels and build.spec.jewels[id]
			if not itemId or itemId == 0 then
				bad("Jewel@%d: allocated jewel socket is EMPTY - a passive point buying nothing", id)
			end
		end
	end
	if problems == before then
		print(string.format("  ok  attributes %d/%d/%d vs req %d/%d/%d, unreserved mana %d, all sockets filled",
			out.Str or 0, out.Dex or 0, out.Int or 0, out.ReqStr or 0, out.ReqDex or 0,
			out.ReqInt or 0, out.ManaUnreserved or 0))
	end
end

------------------------------------------------------------------ tree budget
-- Level doctrine (preferences.md): every build is authored at character level 95 - one
-- that only works at 100 is unreasonable. The budget mirrors Build.lua:884 at that
-- level: usedMax = 94 levels + 23 quest + ExtraPoints (bandit / quest extras land in
-- ExtraPoints); ascendancy and Bloodline each cap at 8 and DRAW ON THE SAME 8 - ascUsed
-- already includes every Bloodline node. A 9th point loads and calcs fine, PoB only puts a
-- warning in a UI corner, so it must be an error here.
print("=== TREE ===")
do
	local used, ascUsed, secondaryAscUsed = build.spec:CountAllocNodes()
	local extra = (build.calcsTab.mainOutput or {}).ExtraPoints or 0
	local usedMax = 94 + 23 + extra
	local before = problems
	if (build.characterLevel or 0) ~= 95 then
		bad("character level %d - every build is authored at 95 (preferences.md)",
			build.characterLevel or 0)
	end
	if used > usedMax then
		bad("tree: %d passive points, budget is %d (94 levels + 23 quest + %d extra)",
			used, usedMax, extra)
	end
	if ascUsed > 8 then
		bad("tree: %d ascendancy points (Bloodline included), budget is 8", ascUsed)
	end
	if secondaryAscUsed > 8 then
		bad("tree: %d Bloodline points, budget is 8", secondaryAscUsed)
	end
	if problems == before then
		print(string.format("  ok  %d/%d points, %d/8 ascendancy (%d Bloodline)",
			used, usedMax, ascUsed, secondaryAscUsed))
	end
end

print(string.format("\n=== TOTAL PROBLEMS: %d ===", problems))
os.exit(problems == 0 and 0 or 1)
