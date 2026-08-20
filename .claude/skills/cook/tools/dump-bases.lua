-- Regenerates data/bases.md AND data/bases.db from one collection pass.
-- Run from .archive/src/ (sqlite3 must be on PATH):
--   luajit ../../.claude/skills/cook/tools/dump-bases.lua
--
-- Source is data.itemBases (Data/Bases/*.lua, GGG's BaseItemTypes verbatim). Three
-- exclusions, none of them judgement calls:
--   * every name in data/legacy-bases.tsv - bases the game removed (talismans, grafts,
--     charms, quietly retired weapon bases). PoB ships every base GGG ever shipped and,
--     unlike uniques, has no obtainability flag, so that list comes from poewiki via
--     tools/fetch-legacy-bases.lua. Read offline here; this script never uses network.
--   * base.hidden - Modules/Data.lua:1225 keeps these out of the base-type selection
--     lists (Energy Blade forms, Golden Mantle, the pre-rework quivers).
--   * a weapon-class base with no `weapon` stat block - "Random One Hand Sword" and
--     "Random Two Hand Sword", unreferenced anywhere in src/, no damage, no class tags.
-- The db answers "which base", never "which mod": affix legality is spawn weight, and
-- only tools/affixes.lua resolves that.
local pob = dofile("../../.claude/skills/cook/tools/pob.lua")
local data = pob.data()
local OUT = "../.claude/skills/cook/data/"
local q, qn, num = pob.q, pob.qn, pob.num
local templateLine = pob.templateLine

------------------------------------------------------------------ legacy list
-- data/legacy-bases.tsv, produced by tools/fetch-legacy-bases.lua. Offline: this script
-- never touches the network. Refresh the tsv only after a PoB base-data update.
local function loadLegacy()
	local path = OUT .. "legacy-bases.tsv"
	local h = assert(io.open(path, "r"),
		"missing " .. path .. " - run tools/fetch-legacy-bases.lua first")
	local byName, n = {}, 0
	for line in h:lines() do
		if line ~= "" and line:sub(1, 1) ~= "#" then
			local name, class = line:match("^([^\t]*)\t([^\t]*)$")
			assert(name and name ~= "", "malformed legacy-bases.tsv line: " .. line)
			byName[name] = class
			n = n + 1
		end
	end
	h:close()
	assert(n > 0, path .. " is empty - rerun tools/fetch-legacy-bases.lua")
	return byName, n
end

local legacyByName = loadLegacy()

-- Item.lua:2127 GetPrimarySlot, on a base rather than a built item.
local function primarySlot(base)
	if base.weapon then return "Weapon 1" end
	local t = base.type
	if t == "Quiver" or t == "Shield" then return "Weapon 2" end
	if t == "Ring" then return "Ring 1" end
	if t == "Flask" or t == "Tincture" then return "Flask 1" end
	if t == "Graft" then return "Graft 1" end
	return t
end

local function sortedKeys(t)
	local keys = {}
	for k in pairs(t or {}) do keys[#keys + 1] = k end
	table.sort(keys)
	return keys
end

-- implicit/enchant hold newline-separated lines, one entry per line in the parallel
-- *Ids / *ModTypes lists.
local function splitLines(text)
	if not text or text == "" then return {} end
	local out = {}
	for line in (text .. "\n"):gmatch("([^\n]*)\n") do
		if line ~= "" then out[#out + 1] = line end
	end
	return out
end

------------------------------------------------------------------ collect
local bases, legacy = {}, {}
local nHidden, nStatless = 0, 0
for _, name in ipairs(sortedKeys(data.itemBases)) do
	local base = data.itemBases[name]
	local gone = legacyByName[name]
	if base.hidden then
		nHidden = nHidden + 1
	elseif data.weaponTypeInfo[base.type] and not base.weapon then
		nStatless = nStatless + 1
	elseif gone then
		legacy[#legacy + 1] = { name = name, class = base.type }
	else
		bases[#bases + 1] = { name = name, base = base }
	end
end
table.sort(legacy, function(a, b)
	if a.class ~= b.class then return a.class < b.class end
	return a.name < b.name
end)
table.sort(bases, function(a, b)
	if a.base.type ~= b.base.type then return a.base.type < b.base.type end
	return a.name < b.name
end)

-- class summary for the md
local classes, classOrder = {}, {}
for _, e in ipairs(bases) do
	local t = e.base.type
	local c = classes[t]
	if not c then
		c = { class = t, n = 0, slot = primarySlot(e.base), influenced = 0,
			socketMin = nil, socketMax = nil, subs = {} }
		classes[t] = c
		classOrder[#classOrder + 1] = c
	end
	c.n = c.n + 1
	if e.base.influenceTags then c.influenced = c.influenced + 1 end
	local sl = e.base.socketLimit
	if sl then
		c.socketMin = c.socketMin and math.min(c.socketMin, sl) or sl
		c.socketMax = c.socketMax and math.max(c.socketMax, sl) or sl
	end
	if e.base.subType then c.subs[e.base.subType] = true end
end
table.sort(classOrder, function(a, b) return a.class < b.class end)

-- anoint split, for the trap section: the flag rides on individual bases, not on the
-- Talisman subtype.
local nNoAnoint, nTalisman, nAmulet = 0, 0, 0
for _, e in ipairs(bases) do
	if e.base.type == "Amulet" then
		nAmulet = nAmulet + 1
		if e.base.subType == "Talisman" then nTalisman = nTalisman + 1 end
		if e.base.cannotBeAnointed then nNoAnoint = nNoAnoint + 1 end
	end
end

local treeVersion = tostring(_G.build.spec.treeVersion):gsub("_", ".")

------------------------------------------------------------------ bases.md
local f = assert(io.open(OUT .. "bases.md", "w+"))
local function w(fmt, ...)
	f:write(select("#", ...) > 0 and string.format(fmt, ...) or fmt, "\n")
end

w("<!-- GENERATED by .claude/skills/cook/tools/dump-bases.lua - do not hand-edit -->")
w("# Bases\n")
w("_Generated from PoB tree data `%s`. Regenerate after any data update - this script",
	treeVersion)
w("rebuilds `bases.db` too._\n")
w("Every equippable base in the game, with the numbers that decide which one is best")
w("for a slot: %d rows, one per base. A base may be built on IF AND ONLY IF it has a", #bases)
w("row - %d bases removed from the game, %d hidden bases and %d statless weapon",
	#legacy, nHidden, nStatless)
w("placeholders are excluded at generation. Query")
w("before naming a base; a remembered name is a hypothesis, and PoB's own base list")
w("is not the answer - it still offers every base GGG has ever shipped.\n")
w("**Scope: bases, not mods.** A row says the base exists, what it is worth, and which")
w("influences it accepts. It says nothing about which affixes can roll on it - that is")
w("spawn weight, and `tools/affixes.lua \"<base>\"` is the only answer. Never infer a")
w("mod pool from `base_tags`.\n")

w("## Querying bases.db\n")
w("Run `sqlite3` with paths from the repo root (from `src/`, prefix `../`). Names are")
w("exactly what goes in an item's base line, parentheses included: `Two-Stone Ring")
w("(Fire/Cold)`. Double any apostrophe inside SQL strings.\n")
w("- `bases(id, name, class, subtype, slot, req_level, req_str, req_dex, req_int,")
w("  socket_limit, one_hand, melee, implicit, can_be_anointed, armour_min, armour_max,")
w("  evasion_min, evasion_max, es_min, es_max, ward_min, ward_max, block_chance,")
w("  movement_penalty, phys_min, phys_max, crit_chance, attack_rate, phys_dps, range,")
w("  flask_life, flask_mana, flask_duration, flask_charges_used, flask_charges_max,")
w("  tincture_mana_burn, tincture_cooldown)` - `class` is the item class")
w("  (`Body Armour`, `Bow`, `Jewel`); `slot` is the primary equipment slot PoB puts it")
w("  in. Requirements are 0, never NULL, when the base has none. Every stat column is")
w("  NULL when the column does not apply to that class - filter on `IS NOT NULL`, not")
w("  on `> 0`. `implicit` is the joined implicit text for eyeballing; query")
w("  `base_mods` for structure.")
w("- `base_tags(base_id, tag)` - GGG's base tags (`str_armour`, `two_hand_weapon`,")
w("  `talisman`). Identity only. NOT a mod pool.")
w("- `base_influences(base_id, influence, mod_tag)` - one row per influence the base")
w("  accepts (`shaper`, `elder`, `adjudicator`, `basilisk`, `crusader`, `eyrie`,")
w("  `cleansing`, `tangle`); `mod_tag` is what `affixes.lua --<influence>` looks up. No")
w("  rows = the base cannot be influenced at all.")
w("- `base_mods(id, base_id, scope, ord, line, template, affix_id, types)` - scope")
w("  `implicit|enchant|flask_buff`; `line` keeps ranges (`+(20-30) to Strength`);")
w("  `template` is the line with every number/range as `#` - LIKE-match on it, not on")
w("  `line`; `affix_id` is GGG's mod id; `types` is the space-joined mod-type tags.")
w("- `base_mod_values(mod_id, ord, min, max)` - a line's numbers in reading order;")
w("  fixed values have min = max. Numeric criteria go here, never regex on `line`.")
w("- `legacy_bases(name, class)` - the bases excluded as unobtainable, so a remembered")
w("  name resolves to \"removed from the game\" instead of silence. NOT usable.")
w("- `meta(key, value)` - `tree_version`, `generated_at`, `legacy_bases`.\n")
w("Canonical queries:\n")
w("```sh")
w("DB=.claude/skills/cook/data/bases.db")
w("# existence + slot - zero rows means DO NOT USE, whatever the reason")
w("sqlite3 $DB \"SELECT class, subtype, slot, req_level FROM bases WHERE name='Vaal Regalia';\"")
w("# best base of a type: highest ES body armour wearable at 95")
w("sqlite3 $DB \"SELECT name, es_max, req_int FROM bases WHERE class='Body Armour'")
w("  AND es_max IS NOT NULL AND req_level<=95 ORDER BY es_max DESC LIMIT 5;\"")
w("# best weapon: top physical DPS one-handers that are not Sceptres")
w("sqlite3 $DB \"SELECT name, class, phys_dps, crit_chance, attack_rate FROM bases")
w("  WHERE one_hand=1 AND class<>'Sceptre' ORDER BY phys_dps DESC LIMIT 10;\"")
w("# fastest bases for a trigger build, crit-viable")
w("sqlite3 $DB \"SELECT name, class, attack_rate, crit_chance FROM bases")
w("  WHERE attack_rate IS NOT NULL AND crit_chance>=6.5 ORDER BY attack_rate DESC LIMIT 10;\"")
w("# implicit search: bases whose implicit gives +1 to a gem level")
w("sqlite3 $DB \"SELECT b.name, b.class, m.line FROM base_mods m JOIN bases b ON b.id=m.base_id")
w("  WHERE m.scope='implicit' AND m.template LIKE '%to Level of all % Skill Gems%';\"")
w("# influence check before authoring a shaper mod")
w("sqlite3 $DB \"SELECT i.influence, i.mod_tag FROM base_influences i")
w("  JOIN bases b ON b.id=i.base_id WHERE b.name='Hubris Circlet';\"")
w("# a name that returns nothing: ask whether it USED to exist")
w("sqlite3 $DB \"SELECT class FROM legacy_bases WHERE name='Imbued Wand';\"")
w("```\n")

w("## What min and max mean\n")
w("The two families of range column are NOT the same measurement.\n")
w("- **Weapons** - `phys_min`/`phys_max` are the damage roll of a hit, both real. The")
w("  ranking number is `phys_dps` = `(phys_min+phys_max)/2 * attack_rate`, stored")
w("  rounded to 2dp so it can be sorted directly. It is base physical only: no quality,")
w("  no affixes, no added elemental.")
w("- **Armour** - `armour_min`..`armour_max` (and evasion / ES / ward) is a variance")
w("  band on the base defence stat, not two separate stats. `Item.lua:2340` applies")
w("  `min + variance * (percentile or 1)`, so an item you author with no percentile set")
w("  gets **`*_max`** - the 100th-percentile base. Compare bases on `*_max` and the")
w("  build measures what it compares.\n")
w("Quality multiplies both families (20% quality = 20% more base defence, or 20% more")
w("base physical on a weapon) and is applied after the base value, so it never changes")
w("which base wins within a class.\n")

w("## Classes\n")
w("`slot` is the primary slot; a second copy goes in the paired slot where one exists")
w("(`Ring 2`/`Ring 3`, `Weapon 2`, `Flask 2..5`). `influenced` counts bases in the")
w("class that accept influence mods at all.\n")
w("| class | slot | bases | influenced | sockets | subtypes |")
w("|---|---|---|---|---|---|")
for _, c in ipairs(classOrder) do
	local sockets = "-"
	if c.socketMin then
		sockets = c.socketMin == c.socketMax and tostring(c.socketMin)
			or (c.socketMin .. "-" .. c.socketMax)
	end
	local subs = sortedKeys(c.subs)
	w("| %s | %s | %d | %d | %s | %s |", c.class, c.slot, c.n, c.influenced, sockets,
		#subs > 0 and table.concat(subs, ", ") or "-")
end

w("\n## Removed bases\n")
w("PoB has no obtainability flag on bases the way it does on uniques - `Data/Bases/*.lua`")
w("is every base GGG ever shipped, and a removed one looks exactly like a live one in the")
w("UI, the base dropdown and the parser. The only signal is poewiki's `drop_enabled`, so")
w("it is frozen into `data/legacy-bases.tsv` and this script reads that file offline;")
w("%d PoB bases are excluded by it. Bases the wiki has no row for at all are named", #legacy)
w("one by one in the fetch tool's `NO_WIKI_ROW` table - absence from a wiki is evidence")
w("of nothing, so each is a recorded call. Anything in neither place is KEPT and")
w("reported, so new PoB data surfaces loudly instead of vanishing.\n")
w("Refresh the tsv with `tools/fetch-legacy-bases.lua` (the only tool here that needs")
w("network) after a PoB base-data update. Nothing else re-queries the wiki.\n")
w("| class | n | bases |")
w("|---|---|---|")
do
	local groups, order = {}, {}
	for _, e in ipairs(legacy) do
		if not groups[e.class] then
			groups[e.class] = {}
			order[#order + 1] = e.class
		end
		table.insert(groups[e.class], e.name)
	end
	for _, k in ipairs(order) do
		w("| %s | %d | %s |", k, #groups[k], table.concat(groups[k], ", "))
	end
end

w("\n## Traps\n")
if nTalisman == nNoAnoint then
	w("- **A Talisman costs you the anoint.** All %d surviving Talisman bases ship a fixed",
		nTalisman)
	w("  `enchant`-scope mod instead and carry `can_be_anointed = 0`; the %d plain amulets",
		nAmulet - nTalisman)
	w("  anoint normally. The Talismans that did anoint are the ones 3.29 removed, so a")
	w("  half-remembered \"talisman with an implicit\" is a legacy base - check")
	w("  `legacy_bases`. Read the flag rather than the subtype; it is NULL off amulets,")
	w("  where anointing does not exist.")
else
	w("- **Anoint legality is per base, not per subtype.** %d of the %d Talisman amulets",
		nNoAnoint, nTalisman)
	w("  ship with a fixed `enchant`-scope mod and carry `can_be_anointed = 0`; the other")
	w("  %d Talismans and all %d plain amulets have a normal implicit and anoint fine.",
		nTalisman - nNoAnoint, nAmulet - nTalisman)
	w("  Read the flag - \"it is a Talisman\" decides nothing. `can_be_anointed` is NULL")
	w("  off amulets, where anointing does not exist.")
end
w("- **Thrusting swords are One Handed Swords.** `class` says `One Handed Sword` and")
w("  `subtype` says `Thrusting`; GGG treats the class as identical for weapon")
w("  restrictions (`CalcPerform.lua:226`), so a skill that wants a sword takes one. The")
w("  subtype only matters for the dual-wield same-class check.")
w("- **Grafts are not in the game** - a 3.27 league mechanic, removed in 3.28. They are")
w("  in `legacy_bases` like any other removed base, but they get their own note because")
w("  PoB carries a full Graft mod pool and two Graft slots as well: it hides the slots")
w("  (`ItemsTab.lua:166`) and drops graft items from calcs (`CalcSetup.lua:703`) on any")
w("  tree but 3.27, so on the %s tree a graft parses fine and measures as nothing.",
	treeVersion)
w("- **No `base_influences` rows means no influence, ever** - jewels, flasks and")
w("  tinctures have none. One Exarch + one Eater implicit remains a separate axis with")
w("  its own rules: `influences.md`.")
w("- **`socket_limit` is the ceiling, not a promise** - a 6-link needs a base with")
w("  `socket_limit = 6`, which is body armours and two-handed weapons only. NULL means")
w("  the class takes no sockets.")
w("- **Jewel bases live here too**, but what a jewel DOES to the tree - cluster")
w("  mechanics, timeless conquests, radius transformers - is `jewels.md`. This db only")
w("  says which jewel bases exist and what they socket into.")
f:close()

------------------------------------------------------------------ bases.db
local nTags, nInfluences, nMods = 0, 0, 0
pob.buildDb(OUT .. "bases.db", function(w)
	w([[
PRAGMA journal_mode=OFF;
BEGIN;
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT);
CREATE TABLE bases (
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	class TEXT NOT NULL,
	subtype TEXT,
	slot TEXT NOT NULL,
	req_level INTEGER NOT NULL,
	req_str INTEGER NOT NULL,
	req_dex INTEGER NOT NULL,
	req_int INTEGER NOT NULL,
	socket_limit INTEGER,
	one_hand INTEGER,
	melee INTEGER,
	implicit TEXT,
	can_be_anointed INTEGER,
	armour_min REAL, armour_max REAL,
	evasion_min REAL, evasion_max REAL,
	es_min REAL, es_max REAL,
	ward_min REAL, ward_max REAL,
	block_chance REAL,
	movement_penalty REAL,
	phys_min REAL, phys_max REAL,
	crit_chance REAL,
	attack_rate REAL,
	phys_dps REAL,
	range REAL,
	flask_life REAL, flask_mana REAL,
	flask_duration REAL,
	flask_charges_used INTEGER, flask_charges_max INTEGER,
	tincture_mana_burn REAL, tincture_cooldown REAL
);
CREATE TABLE base_tags (
	base_id INTEGER NOT NULL REFERENCES bases(id),
	tag TEXT NOT NULL,
	PRIMARY KEY (base_id, tag)
);
CREATE TABLE base_influences (
	base_id INTEGER NOT NULL REFERENCES bases(id),
	influence TEXT NOT NULL,
	mod_tag TEXT NOT NULL,
	PRIMARY KEY (base_id, influence)
);
CREATE TABLE base_mods (
	id INTEGER PRIMARY KEY,
	base_id INTEGER NOT NULL REFERENCES bases(id),
	scope TEXT NOT NULL,
	ord INTEGER NOT NULL,
	line TEXT NOT NULL,
	template TEXT NOT NULL,
	affix_id TEXT,
	types TEXT
);
CREATE TABLE base_mod_values (
	mod_id INTEGER NOT NULL REFERENCES base_mods(id),
	ord INTEGER NOT NULL,
	min REAL NOT NULL,
	max REAL NOT NULL,
	PRIMARY KEY (mod_id, ord)
);
CREATE TABLE legacy_bases (
	name TEXT PRIMARY KEY,
	class TEXT NOT NULL
);]])

	for _, e in ipairs(legacy) do
		w(("INSERT INTO legacy_bases VALUES (%s, %s);"):format(q(e.name), q(e.class)))
	end

	local modId = 0
	for id, entry in ipairs(bases) do
		local name, base = entry.name, entry.base
		local req, wpn = base.req or {}, base.weapon
		local arm, flask, tinc = base.armour or {}, base.flask, base.tincture
		local wti = data.weaponTypeInfo[base.type]

		local physDps
		if wpn and wpn.AttackRateBase then
			physDps = ((wpn.PhysicalMin or 0) + (wpn.PhysicalMax or 0)) / 2 * wpn.AttackRateBase
			physDps = math.floor(physDps * 100 + 0.5) / 100
		end
		-- only amulets can be anointed at all; the flag is meaningless elsewhere
		local anoint = base.type == "Amulet" and (base.cannotBeAnointed and 0 or 1) or nil

		w(("INSERT INTO bases VALUES (%d, %s, %s, %s, %s, %d, %d, %d, %d, %s, %s, %s, %s, %s, "
			.. "%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, "
			.. "%s, %s, %s, %s, %s, %s, %s);"):format(
			id, q(name), q(base.type), qn(base.subType), q(primarySlot(base)),
			req.level or 0, req.str or 0, req.dex or 0, req.int or 0,
			num(base.socketLimit),
			wti and (wti.oneHand and 1 or 0) or "NULL",
			wti and (wti.melee and 1 or 0) or "NULL",
			qn(base.implicit), num(anoint),
			num(arm.ArmourBaseMin), num(arm.ArmourBaseMax),
			num(arm.EvasionBaseMin), num(arm.EvasionBaseMax),
			num(arm.EnergyShieldBaseMin), num(arm.EnergyShieldBaseMax),
			num(arm.WardBaseMin), num(arm.WardBaseMax),
			num(arm.BlockChance), num(arm.MovementPenalty),
			num(wpn and wpn.PhysicalMin), num(wpn and wpn.PhysicalMax),
			num(wpn and wpn.CritChanceBase), num(wpn and wpn.AttackRateBase),
			num(physDps), num(wpn and wpn.Range),
			num(flask and flask.life), num(flask and flask.mana),
			num(flask and flask.duration),
			num(flask and flask.chargesUsed), num(flask and flask.chargesMax),
			num(tinc and tinc.manaBurn), num(tinc and tinc.cooldown)))

		for _, tag in ipairs(sortedKeys(base.tags)) do
			w(("INSERT INTO base_tags VALUES (%d, %s);"):format(id, q(tag)))
			nTags = nTags + 1
		end

		for _, inf in ipairs(sortedKeys(base.influenceTags)) do
			w(("INSERT INTO base_influences VALUES (%d, %s, %s);"):format(
				id, q(inf), q(base.influenceTags[inf])))
			nInfluences = nInfluences + 1
		end

		local scopes = {
			{ "implicit", splitLines(base.implicit), base.implicitIds, base.implicitModTypes },
			{ "enchant", splitLines(base.enchant), base.enchantIds, base.enchantModTypes },
			{ "flask_buff", flask and flask.buff or {}, nil, nil },
		}
		for _, sc in ipairs(scopes) do
			local scope, lines, ids, types = sc[1], sc[2], sc[3], sc[4]
			for ord, line in ipairs(lines) do
				modId = modId + 1
				local template, vals = templateLine(line)
				local tl = types and types[ord]
				w(("INSERT INTO base_mods VALUES (%d, %d, %s, %d, %s, %s, %s, %s);"):format(
					modId, id, q(scope), ord, q(line), q(template),
					qn(ids and ids[ord]), qn(tl and #tl > 0 and table.concat(tl, " ") or nil)))
				for vord, v in ipairs(vals) do
					w(("INSERT INTO base_mod_values VALUES (%d, %d, %s, %s);"):format(
						modId, vord, tostring(v[1]), tostring(v[2])))
				end
			end
		end
	end
	nMods = modId

	w(("INSERT INTO meta VALUES ('tree_version', %s);"):format(q(treeVersion)))
	w(("INSERT INTO meta VALUES ('generated_at', %s);"):format(q(os.date("!%Y-%m-%d"))))
	w("INSERT INTO meta VALUES ('generator', 'dump-bases.lua');")
	w(("INSERT INTO meta VALUES ('legacy_bases', '%d');"):format(#legacy))
	w([[
CREATE INDEX idx_bases_name ON bases(name);
CREATE INDEX idx_bases_class ON bases(class);
CREATE INDEX idx_bases_slot ON bases(slot);
CREATE INDEX idx_base_tags_tag ON base_tags(tag);
CREATE INDEX idx_base_mods_base ON base_mods(base_id);
CREATE INDEX idx_base_mods_template ON base_mods(template);
COMMIT;]])
end)

print(string.format(
	"wrote bases.md + bases.db: %d bases in %d classes (excluded %d removed, %d hidden, %d statless), %d tags, %d influence rows, %d mod lines, tree %s",
	#bases, #classOrder, #legacy, nHidden, nStatless,
	nTags, nInfluences, nMods, treeVersion))
