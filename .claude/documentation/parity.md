# Module inventory & port tracker

Every unit of Path of Building, as it exists complete in `.archive/` (this
branch's frozen Lua application). Modules are independently claimable; anything
with its dependencies met can be picked up at any time.

**Checking a module off.** `[ ]` not started · `[~]` in progress / partial ·
`[x]` done. Done means: ported to Go, with a differential test against the
`.archive` implementation that **fails on any disagreement** and passes at
100% — the modparser standard (its archive dumps: 13,173/13,173 corpus lines,
8,800/8,800 table entries). A module without a runnable Lua archive dump (pure
view code) instead states in its row what "verified" meant. Update the row's status,
add the verifying test's name, and keep the reference files/lines column
untouched — it describes the reference, not the port.

**Kind** — how the module divides, the axis decoupling runs along:

- `logic` — calculation, parsing, game data, persistent state. No presentation.
- `view` — presentation, layout, input handling. No domain knowledge.
- `mixed` — both, interleaved. The seam is named per row; port the logic half
  first, verify it against the archive, then build the view half on top.

All paths relative to `.archive/src/`. Line counts measured 2026-08-19.

## Progress

| | count |
|---|---|
| done | 3 (`mod-parser`, `export-tooling`, `mod-store`) |
| in progress | 0 |
| not started | 37 |

---

## Logic — calculation

| status | module | files | lines | verified by |
|---|---|---|---|---|
| `[ ]` | `calc-core` | `Modules/Calcs.lua` `CalcSetup.lua` `CalcPerform.lua` `CalcTools.lua` `CalcFormat.lua` | 7,048 | |
| `[ ]` | `calc-offence` | `Modules/CalcOffence.lua` `CalcActiveSkill.lua` `CalcMirages.lua` `CalcTriggers.lua` | 9,145 | |
| `[ ]` | `calc-defence` | `Modules/CalcDefence.lua` | 3,828 | |
| `[ ]` | `calc-breakdown` | `Modules/CalcBreakdown.lua` | 251 | |
| `[x]` | `mod-parser` | `Modules/ModParser.lua` `ModTools.lua` | 7,193 | `test/parse_test.go` 13,173/13,173 · `test/tables_test.go` 8,800/8,800 · `test/modtools_test.go` 12,736 mods x 7 behaviours, 0 disagreements (format*/parseTags/parseFormattedSourceMod/compareModParams/setSource + the deep-copying parse cache). `mergeKeystones` reassigned to `mod-store` — it operates on a live ModDB and the tree keystone map |
| `[x]` | `mod-store` | `Classes/ModStore.lua` `ModDB.lua` `ModList.lua` | 1,530 | `test/modstore_test.go` 18,525 checks, 0 disagreements: the parsed corpus distributed over a store tree with fixture actors/configs, every aggregation (Sum/More/Flag/Override/List/Tabulate/HasMod/Max/Min/GetMultiplier/GetCondition), construction behaviours (ScaleAdd/Merge/Replace/Convert), `mergeKeystones`, plus 59 synthetic mods covering the tag branches the corpus never produces. Reference crashes are part of the contract (recorded as error sentinels, the port fails identically) |
| `[ ]` | `stat-describer` | `Modules/StatDescriber.lua` | 292 | |

`mod-parser` converts the game's English mod text into structured modifiers:
~4,200 patterns composed by form / name / flag / tag scanners, plus ~3,900
lines of per-item special cases. The Go port keys every pattern table with
ordinary Go regex.

`calc-breakdown` produces per-stat derivation payloads (simple, mod, slot,
area, effMult, dot, critDot, leech, multiChain) consumed by `calcs-view`.

## Logic — game data

| status | module | files | lines | verified by |
|---|---|---|---|---|
| `[ ]` | `game-data` | `Modules/Data.lua` + `Data/` (134 files) | 798,584 | partial: `/gamedata` + the export builders now produce the generated data as typed JSON documents (no Lua); `/modparser/vocab.go` carries the parser's vocabulary, extracted one-time and Go-maintained. Remaining here: loading the documents into the app, the hand-maintained `Data/*.lua` parts, and vocab regeneration |
| `[ ]` | `tree-data` | `Classes/PassiveTree.lua` + `TreeData/` (61 files) | 4,113,782 | |
| `[ ]` | `timeless-jewel-data` | `Modules/DataAbyssJewelLookUpTableHelper.lua` `DataLegionLookUpTableHelper.lua` `DataJewelFileLoader.lua` | 632 | |
| `[ ]` | `item-model` | `Classes/Item.lua` `Modules/ItemTools.lua` | 3,093 | |
| `[ ]` | `pantheon` | `Modules/PantheonTools.lua` | 18 | |

`Data/` is mostly declarative — `mod(...)` constructor calls, only 210
`function` occurrences across 798k lines. `tree-data` also owns sprite sheets,
orbit geometry, connector construction and cluster-jewel subgraph building.

## Logic — build state

| status | module | files | lines | verified by |
|---|---|---|---|---|
| `[ ]` | `build-core` | `Modules/Build.lua` | 2,091 | |
| `[ ]` | `passive-spec` | `Classes/PassiveSpec.lua` | 2,407 | |
| `[ ]` | `undo` | `Classes/UndoHandler.lua` | 51 | |
| `[ ]` | `display-stats` | `Modules/BuildDisplayStats.lua` | 250 | output byte-locked (memory `statbox-byte-lock`) — the archive comparison should be byte-level |
| `[ ]` | `common` | `Modules/Common.lua` `Utils.lua` | 1,209 | port pieces on demand; note what landed here |
| `[ ]` | `headless-adapter` | `HeadlessWrapper.lua` | 227 | the host-environment contract; its Go analogue emerges from whatever the ported modules need |

`undo` is a snapshot stack; `AddUndoState` also sets `modFlag` on its owner.
Granularity is decided per interaction, not per state change — a same-class
ascendancy click pushes two states (`PassiveTreeView.lua:433,504`), a
cross-class click pushes one covering class, ascendancy and node (`:466`), and
the confirm-prompt path pushes inside the callback (`:475`).

`build-core` drives recalculation by coalescing: `buildFlag` set by any edit,
one `BuildOutput` per frame (`Build.lua:613-624`), with `outputRevision`
incremented per rebuild.

## Logic — networking

| status | module | files | lines | verified by |
|---|---|---|---|---|
| `[ ]` | `poe-api` | `Classes/PoEAPI.lua` `LaunchServer.lua` | 447 | |
| `[ ]` | `build-sites` | `Modules/BuildSiteTools.lua` | 129 | |
| `[ ]` | `launcher` | `Launch.lua` `LaunchInstall.lua` `GameVersions.lua` | 719 | |
| `[ ]` | `updater` | `UpdateCheck.lua` `UpdateApply.lua` | 387 | |

`launcher` owns subscript management and `DownloadPage` (a curl subscript);
every other module's network access goes through it. `poe-api` covers OAuth
token fetch, validation, character download with rate limiting, and the local
redirect catcher.

## View

No domain knowledge; presentation and input only. No Lua archive dump is possible —
each row must state its own verification when checked off (golden screenshots,
DOM/state assertions, or manual sign-off noted here).

| status | module | files | lines | verified by |
|---|---|---|---|---|
| `[ ]` | `control-kit` | `Control` `ControlHost` `Button` `CheckBox` `DropDown` `Edit` `Label` `Path` `PopupDialog` `ResizableEdit` `ScrollBar` `Section` `Slider` `TextList` `TooltipHost` `RectangleOutline` `Dragger` `SearchHost` `ListControl` | 3,498 | |
| `[ ]` | `tree-view` | `Classes/PassiveTreeView.lua` `PassiveMasteryControl.lua` `PassiveSpecListControl.lua` `PowerReportListControl.lua` | 2,153 | |
| `[ ]` | `items-view` | `ItemDBControl` `ItemListControl` `ItemSetListControl` `ItemSlotControl` `NotableDBControl` `SharedItemListControl` `SharedItemSetListControl` | 1,543 | |
| `[ ]` | `skills-view` | `SkillListControl` `SkillSetListControl` `GemSelectControl` | 1,223 | |
| `[ ]` | `calcs-view` | `CalcSectionControl` `CalcBreakdownControl` | 1,295 | |
| `[ ]` | `timeless-jewel-view` | `TimelessJewelListControl` `TimelessJewelSocketControl` | 359 | |
| `[ ]` | `minion-library` | `MinionListControl` `MinionSearchListControl` | 170 | |
| `[ ]` | `toast` | `Modules/ToastNotification.lua` | 257 | |

`ScrollBar`, `Dragger`, `Path`, `ResizableEdit` and `TextList` exist because
SimpleGraphic provides no widget toolkit; their line counts describe what they
compensate for, not what an equivalent must cost.

`tree-view` also owns node-click semantics — allocation, ascendancy switching,
undo granularity — which are domain decisions living in a `Draw` handler. That
is the single largest logic-in-view concentration in the codebase.

`GemSelectControl` is type-ahead over the gem list with per-candidate DPS
ranking; it calls the calc engine once per candidate, so it is view code with a
calculation dependency.

## Mixed

Domain logic and presentation interleaved. The seam column names where they
separate; check the halves off independently (`logic:[ ] view:[ ]` in the
status cell).

| status | module | files | lines | seam |
|---|---|---|---|---|
| `logic:[ ] view:[ ]` | `tree` | `Classes/TreeTab.lua` | 2,873 | spec management, mastery eligibility, tattoos, power calc and timeless search are logic; the popup constructors are view |
| `logic:[ ] view:[ ]` | `items` | `Classes/ItemsTab.lua` `Modules/ItemSlotHelper.lua` | 5,243 | each crafting popup fuses an option-list builder, a DPS-sorted ranking and an apply step — those three are logic, the dialog is view |
| `logic:[ ] view:[ ]` | `skills` | `Classes/SkillsTab.lua` | 1,475 | socket-group processing, gem matching and `OptimiseSockets` are logic; gem slot rows are view |
| `logic:[ ] view:[ ]` | `calcs` | `Classes/CalcsTab.lua` `Modules/CalcSections.lua` | 3,366 | `CalcSections.lua` is 2,591 lines of section schema — data, not code. Power calculation and the modKey cache are logic; section layout and pinning are view |
| `logic:[ ] view:[ ]` | `config` | `Classes/ConfigTab.lua` `Modules/ConfigOptions.lua` `ConfigVisibility.lua` `Classes/ConfigSetListControl.lua` | 4,179 | 580 options (check 331, count 163, list 41, integer 11, float 1) carrying 524 embedded apply closures. `ConfigVisibility.lua` computes per-option relevance from mainEnv usage maps — logic. Widget rendering is view |
| `logic:[ ] view:[ ]` | `notes` | `Classes/NotesTab.lua` | 107 | text persistence is logic; colour-code buttons are view |
| `logic:[ ] view:[ ]` | `party` | `Classes/PartyTab.lua` | 1,037 | `ParseBuffs`, `setBuffExports`, `exportBuffs` are logic; the simple/advanced editors are view |
| `logic:[ ] view:[ ]` | `import` | `Classes/ImportTab.lua` | 1,931 | code generation, character download, re-import state preservation are logic; status display and option toggles are view |
| `logic:[ ] view:[ ]` | `compare` | `Classes/CompareTab.lua` `CompareEntry.lua` `CompareCalcsHelpers.lua` `ComparePowerReportListControl.lua` | 6,171 | `CompareEntry` builds a second in-process build with its own tabs and a stubbed party actor — logic. The diff sub-views reuse the other modules' views |
| `logic:[ ] view:[ ]` | `tooltips` | `Classes/Tooltip.lua` `TooltipHost.lua` `GemTooltip.lua` | 978 | `Tooltip` splits cleanly along its own method list: `Clear`/`AddLine`/`AddSeparator`/`SetRecipe` accumulate content, `GetSize`/`GetDynamicSize`/`CalculateColumns`/`Draw` render it. Content composition — `AddItemTooltip`, `AddStatComparesToTooltip`, `GemTooltip` — is logic and is roughly 3k lines living in `items` and `skills` |
| `logic:[ ] view:[ ]` | `trade` | `TradeQuery` `TradeQueryGenerator` `TradeQueryRequests` `TradeQueryRateLimiter` `TradeHelpers` `CompareBuySimilar` `TradeStatWeightMultiplierListControl` | 4,839 | weight generation, query building, rate-limit policy parsing are logic; `TradeQuery.lua` (1,376) is the view. The generator runs as a coroutine pumped by the frame loop with a progress popup |
| `logic:[ ] view:[ ]` | `app-shell` | `Modules/Main.lua` | 1,816 | mode management, settings load/save, user paths are logic; the popup registry, key handling and chrome are view |
| `logic:[ ] view:[ ]` | `build-list` | `Modules/BuildList.lua` `BuildListHelpers.lua` `Classes/BuildListControl.lua` `FolderListControl.lua` | 796 | scan, filter, sort, header parsing and folder moves are logic (`BuildListHelpers`); the screen is view |
| `logic:[ ] view:[ ]` | `ext-build-lists` | `ExtBuildListControl` `ExtBuildListProvider` `PoBArchivesProvider` | 646 | providers are logic; the listing is view |

## Assets

`.archive/src/Assets/` — 87 images plus `ascendants/` (class portrait jpegs).
Not code; each family checks off when the Go UI actually renders it.

| status | family | files | bound to |
|---|---|---|---|
| `[ ]` | item slot icons (`icon_amulet` … `icon_weapon_2_swap`) | 16 | `items-view` |
| `[ ]` | tooltip headers, 9-sliced left/middle/right per rarity (white, magic, rare, unique, gem, foil) | 18 + 6 separators | `tooltips` |
| `[ ]` | passive headers, 9-sliced per node type (normal, notable, keystone, jewel, mastery allocated/unallocated, ascendancy) | 15 | `tooltips` |
| `[ ]` | influence icons (shaper, elder, crusader, redeemer, hunter, warlord, exarch, eater, synthesis, fractured, veiled, breach, memory, experimented, vestigial) | 15 | `items-view`, `tooltips` |
| `[ ]` | radius rings: `ring.png` (jewel socket hover), `ShadedInnerRing`/`ShadedOuterRing` + `Flipped` variants (Thread of Hope), `small_ring.png` (search highlight) | 6 | `tree-view` |
| `[ ]` | `range_guide.png` + `game_ui_small.png` overlay | 2 | `calcs-view` area-of-effect scale |
| `[ ]` | `intangibilitybg.png`, `memorybg.png`, `ascendants/` portraits | — | `tree-view` |

Tree art is separate and lives in `TreeData/` under `tree-data` (sprite sheets,
mastery, frame, group and connector images, `jewel-radius.png`).

## Tooling

| status | module | files | lines | verified by |
|---|---|---|---|---|
| `[x]` | `export-tooling` | `src/Export/` (65 files) | 51,545 | `test/export_test.go` 123/123 files byte-identical |

Reads the GGPK's dat files and produces the game data the application
consumes — as **structured JSON documents** typed by `/gamedata` (one per
script), no Lua in the pipeline. Ported to `/export` + `cmd/pobexport`: dat64
reader (`dat.go`), column schemas (`spec_gen.go`, one-time transform of
`spec.lua`, Go-maintained), statdesc engine (`statdesc.go`), LuaJIT PRNG
(`luaprng.go`) and hash-table iteration (`luatab.go`) replicas (both baked
into *data*: LegionPassives offsets, tradeHashes order), and 23 of 24
`Scripts/*.lua` as document builders. Extraction stays with
`bun_extract_file.exe` (Export/ggpk/README.md); `WriteEnumFiles` replaces
`enums.lua`'s synthetic dat writes. Verification: build every document over
the extracted GGPK, round-trip through JSON, render back to the reference's
`Data/*.lua` byte format with `internal/luarender` (test-only; holds all the
Lua serialisation quirks and dies with the archive) and byte-compare all 123
files against the checked-in copies — `TestExportAgainstReference`, 123/123.
Excluded by decision: `legionSprites.lua` (GIMP sprite-sheet asset pipeline;
the checked-in PNGs/tree-legion.lua stay as-is).

## Dependencies

Only real edges; everything else is independent.

```
game-data ──> mod-parser ──> mod-store ──> calc-core ──┬─> calc-offence
                                                       ├─> calc-defence
                                                       └─> calc-breakdown ──> calcs-view
game-data ──> item-model ──> items
tree-data ──> passive-spec ──> tree ──> tree-view
timeless-jewel-data ──> tree ──> timeless-jewel-view
calc-core ──> calcs, config, skills-view (gem ranking), trade (weight generation)
control-kit ──> every view module
tooltips ──> items, skills, tree-view, config
items + skills ──> import                    (character import writes both)
build-core ──> every mixed module
compare ──> tree-view, items, skills, config, calcs
launcher ──> poe-api, build-sites, updater, trade
build-list-helpers ──> build-list
assets ──> tooltips, items-view, tree-view, calcs-view
export-tooling ──> game-data          (done; generates its input from the GGPK)
```

Claimable now (dependencies met or absent): `calc-core` (mod-store done), `stat-describer`, `pantheon`, `undo`,
`common`, `notes` (logic), `party` (logic), `toast`, `minion-library`,
`ext-build-lists`, `display-stats`, `build-list` (logic), `game-data`
(export-tooling now feeds it).

## Cross-cutting behaviours

Properties of the application that no single module owns, and that anything
reproducing a module must preserve.

- **Recalculation is coalesced, never per-edit.** Edits set `buildFlag`; one
  `BuildOutput` runs per frame. `outputRevision` marks each rebuild.
- **Long operations are coroutines pumped by the frame loop**, with a blocking
  progress popup and no cancel: `trade`'s weight generation
  (`TradeQueryGenerator.lua:618`), build-list scanning. Timeless-jewel search
  blocks outright.
- **Undo is per-tab, not global.** Each mixed module is its own `UndoHandler`
  and handles Ctrl+Z itself while active (`CalcsTab:317`, `ConfigTab:982`,
  `ItemsTab:1466`, `SkillsTab:623`, `TreeTab:370`). `NotesTab` and `PartyTab`
  delegate to their edit controls.
- **Unsaved state is an OR across every module's `modFlag`** plus
  `spec.modFlag` and `treeTab.searchFlag` (`Build.lua:626`).
- **Statbox, warning and tooltip text is engine-composed**, carrying `^7` /
  `^xRRGGBB` colour codes and a per-line font size.
- **Nine view modes**: TREE SKILLS ITEMS CALCS CONFIG NOTES PARTY IMPORT
  COMPARE, persisted as a `viewMode` attribute on the build XML
  (`Build.lua:78,960`).
- **Errors and prompts are engine-raised**: `Launch:ShowErrMsg`,
  `main:ShowPrompt`, `main:OpenConfirmPopup`, update-available notices
  (`Main.lua:265-276`).
