# Reference ↔ Go name map

Go names describe what things ARE; the reference's names are kept here for
traceability instead of being mirrored into the port. Add a row whenever a
Go name deliberately diverges from its reference counterpart. Reference
names inside ported comments always refer to the Lua side.

Terminology (user-set, 2026-08-28): **conquering** = the mechanic (item
text: "Passives ... are Conquered by ..."); **historic** = the item
keyword for jewels that conquer; **timeless jewel** = the basetype family
(the 5 legion jewels + Heroic Tragedy, types 1-6); **abyss** = the eye-
jewel historic family (types 7-11); **alternate** = the game's own term
for the replacement passives (dat tables AlternatePassiveSkills etc.).

| reference (Lua) | Go |
|---|---|
| `Classes/SkillsTab.lua` (load/logic half) | package `skills` |
| `tree.legion` / `Data/TimelessJewelData/LegionPassives.lua` | `tree.ConqueredPassives` / `data/raw/conqueredpassives.json` |
| `legionNodes[i]` / `legionAdditions[i]` (1-based) | `ConqueredPassives.NodesOrdered[i-1]` / `AdditionsOrdered[i-1]` |
| `Export/Scripts/legionPassives.lua` | `export/script_conqueredpassives.go` (Script Name "legionPassives", OutName "conqueredpassives") |
| PassiveSpec conquering branch (L1191-1376) | `tree/conquer.go` |
| `timelessJewelTypeByConqueror` (covers types 1-11) | `jewelTypeByConqueror` |
| `data.readLUT` (DataLegionLookUpTableHelper) | `tree.TimelessPassive` (computes; no LUT ships) |
| `data.readAbyssJewelLUT` | `tree.AbyssPassive` (computes; no LUT ships) |
| the LUT bins' generation inputs (dat AlternatePassive* tables) | `data/raw/conquertables.json` (export.BuildConquerTables, from the GGPK dats; guarded in test/export_test.go) |
| `/gamedata`-shaped export documents | package `data/schema` |
| fix_ascendancy_positions.py + Common.lua jsonToLua + TreeData/tree.lua | `export.BuildTreeDoc` / `cmd/treegen` (source: GGG skilltree-export tag) |
| REGENERATE_MOD_CACHE startup + Main:SaveModCache → Data/ModCache.lua | `internal/modcachegen.Build` → `data/raw/modcache.jsonl` (guard: test/modcachegen_test.go) |
| `ItemClass:BuildRaw` / `:BuildAndParseRaw` / `:Craft` | `item.(*Item).BuildRaw/BuildAndParseRaw/Craft` (item/buildraw.go) |
| Generated.lua `buildTreeDependentUniques` (Forbidden Flame/Flesh, Skin of the Lords, Impossible Escape) | `data.BuildTreeDependentUniques` (called from tree.Load) |
| whole-source league update (manual multi-step) | `cmd/sourceupdate` (artifacts + tree fetch + modcache + verification) |
| tools/canon.lua encode (dump format) | `test/luacanon` (test-only; artifacts ship conventional JSON) |
| the reference's Data/*.lua text (for byte-compares) | `test/luarender` (test-only; dies with the archive) |
| LuaJIT number-key table pairs() order (mods.lua tradeHashes) | `test/luarender/luatab.go` (artifact carries stat order; render test replays the hash walk) |
| LuaJIT math.random stream (legionPassives.lua layout offsets) | `test/luarender/luaprng.go` (artifact has no oidx; render test redraws the stream) |
| Lua patterns (baseMatch specs, table-archive keys) | Go regex in export/templates; `test/luapat` converts test-side only |
| minions.lua / skills.lua mod-constructor text (mod()/skill()/flag() calls) | structured mods (modparser codec) built at export; runtime decodes (`modparser.DecodeMods`); render test reprints the text (test/luarender) and takes hand-written lines from the archive templates |
| Data/SkillStatMap.lua | `data/skillstatmap.go` (converted once, typed, Go-maintained; export builds minion mods from `data.StatMapTable()`) |
| ModParser tag tables (`{ type = "Condition", var = ... }` maps) | typed `modparser.Tag` kinds (`Kind()` switch); one upstream typo tag type kept as `RawTag` |
| `mod.value` (number/string/table/function) | `modparser.Value` (typed record kinds; `Actor` stays a string) |
| `modDB:Combine(...)` | `modstore` typed `Sum`/`More`/`Flag`/`Override`/`List` |
| tools/gen_skilldata.lua, tools/gen_datatables.lua (retired generators) | converted-once typed tables in `data/` (`skills_custom.go`, `mapmods.go`, `tables.go`, `timeless.go`); no regeneration path |
| `string.lower` (ASCII) | `strings.ToLower`; `(?i)` in scan patterns |
| Item.lua `sanitiseText` / SkillsTab's copy / data's copy | `util.FoldText` (`internal/util/text.go`, one copy; folds uppercase accents too, where the reference folds only lowercase) |
| Export dat `row:Get(col)` | typed column accessors on `*Row`, checked against the hand-maintained `export/spec.go`; typed row queries (`RowByStr`/`RowByRef`/`RowsByInt`/`RowsByBool`/`RowsByRef`/`GetRowListMatch`) replace the `any`-valued `GetRow`/`GetRowList` |
| Export template `#directive` lines | structured JSON directives under `export/templates/` |
| `data.TrimTreeDependentUniques` (test expedient) | `data.Sources.SkipTreeDependentUniques` |
| `Env.StubHandoff` (fixture-mode flag) | `calc.ReplayInput.StubHandoff` |
| `%.14g` / `tostring(number)` helpers (`luaNumStr`, `luaNum`, `luaTostring`…) | `util.FormatG14` / `util.FormatIntOrG14` (`internal/util`, with `Quantize14`, `RoundHalfUp`, `Tonumber`) |
| Export/spec.lua | `export/spec.go` (transformed once, Go-maintained) |
