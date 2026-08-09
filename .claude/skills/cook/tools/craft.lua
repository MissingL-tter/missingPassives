-- Bakes affix-authored items into real mods. Run from src/ after authoring or after
-- editing any Prefix:/Suffix: line, BEFORE measuring anything:
--   luajit ../.claude/skills/cook/tools/craft.lua "Builds/My Build.xml"
--
-- Item:Craft() is only ever called by the ItemsTab UI (ItemsTab.lua:808), never on load. A
-- freshly authored item carries only Prefix:/Suffix: property lines: it parses and validates
-- fine but contributes NOTHING to the calcs - no mod text exists yet, so every stat measured
-- before crafting is measured against naked bases. This runs Craft() on every crafted item
-- and saves, so the file carries generated text.
--
-- Changing an item later: edit its Prefix:/Suffix: lines in the XML and re-run this. Do NOT
-- script edits by rebuilding items from a crafted file's raw text - each new("Item", raw)
-- cycle on text that already holds generated lines re-parses those lines back INTO
-- item.prefixes/item.suffixes, on top of the property lines (three cycles left 9 entries on a
-- 3-prefix item; Craft() still renders only the first 3, so the corruption is invisible until
-- validate flags it). Editing the property lines and re-crafting is idempotent; that is the
-- whole loop.
local BUILD = arg[1] or os.getenv("BUILDXML")
assert(BUILD, 'usage: luajit craft.lua "Builds/My Build.xml"')
local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local build = pob.load(BUILD)

local n = 0
for id, item in pairs(build.itemsTab.items) do
	if item.crafted then
		item:Craft()
		n = n + 1
		print(string.format("  crafted #%-3d %s", id, tostring(item.name)))
	end
end
build.itemsTab:PopulateSlots()
pob.refresh()
pob.save(BUILD)
print(string.format("saved %s (%d items crafted)", BUILD, n))
