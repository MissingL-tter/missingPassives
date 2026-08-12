-- Refreshes data/legacy-bases.tsv: the item bases PoB still ships that the game no
-- longer has. Run from src/, MANUALLY and rarely - only after a PoB data update that
-- changes the base list. Needs network; nothing else in the skill does.
--   luajit ../.claude/skills/cook/tools/fetch-legacy-bases.lua
--
-- Why it exists: PoB has an obtainability flag on uniques but none on bases, and
-- Data/Bases/*.lua is every base GGG ever shipped. A removed base looks exactly like a
-- live one in the UI, the dropdown and the parser. poewiki's `drop_enabled` is the only
-- signal, so it is pulled once, here, and frozen into a checked-in file that
-- dump-bases.lua reads offline.
local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local data = pob.data()
local OUT = "../.claude/skills/cook/data/"

-- poewiki class ids covering every equipment class PoB has bases for. A class missing
-- here yields no wiki row, which is reported as unmatched and KEPT, never dropped.
local WIKI_CLASSES = {
	"Amulet", "Belt", "Body Armour", "Boots", "Bow", "Claw", "Dagger", "Rune Dagger",
	"FishingRod", "Gloves", "Helmet", "Jewel", "AbyssJewel", "AnimalCharm",
	"One Hand Axe", "One Hand Mace", "One Hand Sword", "Thrusting One Hand Sword",
	"Quiver", "Ring", "Sceptre", "Shield", "Staff", "Warstaff", "Two Hand Axe",
	"Two Hand Mace", "Two Hand Sword", "Wand",
	"LifeFlask", "ManaFlask", "HybridFlask", "UtilityFlask", "Tincture", "BrequelGraft",
}

local function urlencode(s)
	return (s:gsub("[^%w%-%._~]", function(c)
		return string.format("%%%02X", string.byte(c))
	end))
end

-- Cargo double-escapes: HTML entities inside JSON strings, and non-ASCII as \uXXXX
-- ("Maelström Staff"). Undo both, JSON first.
local function utf8char(cp)
	if cp < 0x80 then return string.char(cp) end
	if cp < 0x800 then
		return string.char(0xC0 + math.floor(cp / 0x40), 0x80 + cp % 0x40)
	end
	return string.char(0xE0 + math.floor(cp / 0x1000),
		0x80 + math.floor(cp / 0x40) % 0x40, 0x80 + cp % 0x40)
end

local function unescape(s)
	s = s:gsub("\\u(%x%x%x%x)", function(h) return utf8char(tonumber(h, 16)) end)
	s = s:gsub('\\([""\\/])', "%1")
	return (s:gsub("&#0?39;", "'"):gsub("&quot;", '"'):gsub("&amp;", "&"))
end

-- Cargo export of the `items` table, normal rarity, equipment classes only.
-- Returns name -> obtainable (bool).
local function fetchWiki()
	local quoted = {}
	for i, c in ipairs(WIKI_CLASSES) do quoted[i] = '"' .. c .. '"' end
	local where = 'items.rarity_id="Normal" AND items.class_id IN ('
		.. table.concat(quoted, ",") .. ")"
	local url = "https://www.poewiki.net/index.php?title=Special:CargoExport"
		.. "&tables=items&format=json&limit=500"
		.. "&fields=" .. urlencode("items.name,items.class_id,items.drop_enabled,items.removal_version")
		.. "&where=" .. urlencode(where)

	local tmp = OUT .. "wiki-fetch.tmp"
	local byName, rows, offset = {}, 0, 0
	for _ = 1, 40 do
		local rc = os.execute(('curl -sSf --max-time 60 -o "%s" "%s&offset=%d"')
			:format(tmp, url, offset))
		assert(rc == 0 or rc == true, "poewiki fetch failed (curl) - rerun with network")
		local h = assert(io.open(tmp, "r"), "poewiki fetch produced no file")
		local body = h:read("*a")
		h:close()
		local page = 0
		for obj in body:gmatch("%b{}") do
			local name = obj:match('"name":%s*"(.-)"')
			if name then
				page = page + 1
				name = unescape(name)
				local drop = tonumber(obj:match('"drop enabled":%s*(%d+)')) or 0
				-- one name can carry both a removed row and a re-released row (the 3.29
				-- talismans); a single live row makes the name live
				byName[name] = (byName[name] or drop > 0) and true or false
			end
		end
		rows = rows + page
		if page < 500 then break end
		offset = offset + 500
	end
	os.remove(tmp)
	assert(rows >= 900, ("poewiki returned only %d rows - the query or the schema moved. "
		.. "Refusing to overwrite the list from it."):format(rows))
	return byName, rows
end

local wikiByName, wikiRows = fetchWiki()

-- The wiki spells accented names the way the game does ("Maelström Staff"); PoB
-- strips the diacritics. Fold the UTF-8 Latin-1 letters down so the two agree.
local FOLD = {
	["\195\160"] = "a", ["\195\161"] = "a", ["\195\162"] = "a", ["\195\164"] = "a",
	["\195\168"] = "e", ["\195\169"] = "e", ["\195\170"] = "e", ["\195\171"] = "e",
	["\195\172"] = "i", ["\195\173"] = "i", ["\195\174"] = "i", ["\195\175"] = "i",
	["\195\178"] = "o", ["\195\179"] = "o", ["\195\180"] = "o", ["\195\182"] = "o",
	["\195\185"] = "u", ["\195\186"] = "u", ["\195\187"] = "u", ["\195\188"] = "u",
	["\195\167"] = "c", ["\195\177"] = "n",
}
local function fold(s)
	return (s:gsub("\195[\128-\191]", function(c) return FOLD[c] or c end))
end

local wikiByFolded = {}
for name, ok in pairs(wikiByName) do wikiByFolded[fold(name)] = ok end

-- PoB also disambiguates same-name bases with a parenthetical the wiki does not use:
-- "Two-Stone Ring (Fire/Cold)", "Two-Toned Boots (Armour/Evasion)". nil = no row at all,
-- which is distinct from false.
local function wikiEntry(name)
	local ok = wikiByName[name]
	if ok ~= nil then return ok end
	ok = wikiByFolded[fold(name)]
	if ok ~= nil then return ok end
	local plain = name:match("^(.-) %b()$")
	if not plain then return nil end
	ok = wikiByName[plain]
	if ok ~= nil then return ok end
	return wikiByFolded[fold(plain)]
end

-- poewiki has no row at all for these, and absence is evidence of nothing - so each is
-- a recorded call, not a rule. A PoB base that is neither on the wiki nor listed here
-- gets KEPT and reported, so new data shows up loudly instead of vanishing.
local NO_WIKI_ROW = {
	["Energy Blade One Handed"] = "weapon the Energy Blade skill puts in your hand, not an item",
	["Energy Blade Two Handed"] = "same, two-handed",
	["Ethereal Blade"] = "not in the game",
	["Ethereal Bow"] = "not in the game",
	["Random One Hand Sword"] = "statless placeholder, unreferenced anywhere in src/",
	["Random Two Hand Sword"] = "statless placeholder, unreferenced anywhere in src/",
}

------------------------------------------------------------------ join
local names = {}
for k in pairs(data.itemBases) do names[#names + 1] = k end
table.sort(names)

local dead, unmatched = {}, {}
for _, name in ipairs(names) do
	local ok = wikiEntry(name)
	if ok == nil and NO_WIKI_ROW[name] then
		ok = false
	end
	if ok == nil then
		unmatched[#unmatched + 1] = name
	elseif not ok then
		dead[#dead + 1] = { name = name, class = data.itemBases[name].type }
	end
end
table.sort(dead, function(a, b)
	if a.class ~= b.class then return a.class < b.class end
	return a.name < b.name
end)

------------------------------------------------------------------ legacy-bases.tsv
local f = assert(io.open(OUT .. "legacy-bases.tsv", "w+"))
f:write("# GENERATED by .claude/skills/cook/tools/fetch-legacy-bases.lua - do not hand-edit\n")
f:write("# Bases PoB still ships that the game does not have: poewiki drop_enabled=0, plus\n")
f:write("# the NO_WIKI_ROW calls in the fetch tool. dump-bases.lua reads this file offline;\n")
f:write("# rerun the fetch tool only after a PoB base-data update.\n")
f:write(("# fetched %s from poewiki, %d wiki rows, %d PoB names unaccounted for (kept as live)\n")
	:format(os.date("!%Y-%m-%d"), wikiRows, #unmatched))
f:write("# name\tclass\n")
for _, e in ipairs(dead) do
	f:write(("%s\t%s\n"):format(e.name, e.class))
end
f:close()

print(("wrote legacy-bases.tsv: %d removed bases from %d wiki rows; %d PoB names are on "
	.. "neither the wiki nor the NO_WIKI_ROW list and stay live: %s"):format(
	#dead, wikiRows, #unmatched,
	#unmatched > 0 and table.concat(unmatched, ", ") or "none"))
