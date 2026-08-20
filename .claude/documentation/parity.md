# Module inventory

Every unit of Path of Building, as it exists complete in `E:\tools\missingPassivesTest`.
This file answers *what the application is made of*. It describes that codebase only — no
implementation state, no port progress, no ordering. Modules are independently claimable;
anything with its dependencies met can be picked up at any time.

All paths are relative to `missingPassivesTest/src/`. Line counts measured 2026-08-19.

**Kind** — how the module divides, which is the axis decoupling runs along:

- `logic` — calculation, parsing, game data, persistent state. No presentation.
- `view` — presentation, layout, input handling. No domain knowledge.
- `mixed` — both, interleaved. The seam is named per row; this is where decoupling work is.

---

## Logic — calculation

| module | files | lines |
|---|---|---|
| `calc-core` | `Modules/Calcs.lua` `CalcSetup.lua` `CalcPerform.lua` `CalcTools.lua` `CalcFormat.lua` | 7,048 |
| `calc-offence` | `Modules/CalcOffence.lua` `CalcActiveSkill.lua` `CalcMirages.lua` `CalcTriggers.lua` | 9,145 |
| `calc-defence` | `Modules/CalcDefence.lua` | 3,828 |
| `calc-breakdown` | `Modules/CalcBreakdown.lua` | 251 |
| `mod-parser` | `Modules/ModParser.lua` `ModTools.lua` | 7,193 |
| `mod-store` | `Classes/ModStore.lua` `ModDB.lua` `ModList.lua` | 1,530 |
| `stat-describer` | `Modules/StatDescriber.lua` | 292 |

`mod-parser` converts the game's English mod text into structured modifiers: ~4,200 Lua
patterns composed by form / name / flag / tag scanners, plus ~3,900 lines of per-item special
cases and 1,231 inline closures. Uses no `%b` or `%f` constructs.

`calc-breakdown` produces per-stat derivation payloads (simple, mod, slot, area, effMult, dot,
critDot, leech, multiChain) consumed by `calcs-view`.

## Logic — game data

| module | files | lines |
|---|---|---|
| `game-data` | `Modules/Data.lua` + `Data/` (134 files) | 798,584 |
| `tree-data` | `Classes/PassiveTree.lua` + `TreeData/` (61 files) | 4,113,782 |
| `timeless-jewel-data` | `Modules/DataAbyssJewelLookUpTableHelper.lua` `DataLegionLookUpTableHelper.lua` `DataJewelFileLoader.lua` | 632 |
| `item-model` | `Classes/Item.lua` `Modules/ItemTools.lua` | 3,093 |
| `pantheon` | `Modules/PantheonTools.lua` | 18 |

`Data/` is mostly declarative — `mod(...)` constructor calls, only 210 `function` occurrences
across 798k lines. `tree-data` also owns sprite sheets, orbit geometry, connector construction
and cluster-jewel subgraph building.

## Logic — build state

| module | files | lines |
|---|---|---|
| `build-core` | `Modules/Build.lua` | 2,091 |
| `passive-spec` | `Classes/PassiveSpec.lua` | 2,407 |
| `undo` | `Classes/UndoHandler.lua` | 51 |
| `display-stats` | `Modules/BuildDisplayStats.lua` | 250 |
| `common` | `Modules/Common.lua` `Utils.lua` | 1,209 |
| `headless-adapter` | `HeadlessWrapper.lua` | 227 |

`undo` is a snapshot stack; `AddUndoState` also sets `modFlag` on its owner. Granularity is
decided per interaction, not per state change — a same-class ascendancy click pushes two
states (`PassiveTreeView.lua:433,504`), a cross-class click pushes one covering class,
ascendancy and node (`:466`), and the confirm-prompt path pushes inside the callback (`:475`).

`build-core` drives recalculation by coalescing: `buildFlag` set by any edit, one
`BuildOutput` per frame (`Build.lua:613-624`), with `outputRevision` incremented per rebuild.

## Logic — networking

| module | files | lines |
|---|---|---|
| `poe-api` | `Classes/PoEAPI.lua` `LaunchServer.lua` | 447 |
| `build-sites` | `Modules/BuildSiteTools.lua` | 129 |
| `launcher` | `Launch.lua` `LaunchInstall.lua` `GameVersions.lua` | 719 |
| `updater` | `UpdateCheck.lua` `UpdateApply.lua` | 387 |

`launcher` owns subscript management and `DownloadPage` (a curl subscript); every other
module's network access goes through it. `poe-api` covers OAuth token fetch, validation,
character download with rate limiting, and the local redirect catcher.

## View

No domain knowledge; these are presentation and input only.

| module | files | lines |
|---|---|---|
| `control-kit` | `Control` `ControlHost` `Button` `CheckBox` `DropDown` `Edit` `Label` `Path` `PopupDialog` `ResizableEdit` `ScrollBar` `Section` `Slider` `TextList` `TooltipHost` `RectangleOutline` `Dragger` `SearchHost` `ListControl` | 3,498 |
| `tree-view` | `Classes/PassiveTreeView.lua` `PassiveMasteryControl.lua` `PassiveSpecListControl.lua` `PowerReportListControl.lua` | 2,153 |
| `items-view` | `ItemDBControl` `ItemListControl` `ItemSetListControl` `ItemSlotControl` `NotableDBControl` `SharedItemListControl` `SharedItemSetListControl` | 1,543 |
| `skills-view` | `SkillListControl` `SkillSetListControl` `GemSelectControl` | 1,223 |
| `calcs-view` | `CalcSectionControl` `CalcBreakdownControl` | 1,295 |
| `timeless-jewel-view` | `TimelessJewelListControl` `TimelessJewelSocketControl` | 359 |
| `minion-library` | `MinionListControl` `MinionSearchListControl` | 170 |
| `toast` | `Modules/ToastNotification.lua` | 257 |

`ScrollBar`, `Dragger`, `Path`, `ResizableEdit` and `TextList` exist because SimpleGraphic
provides no widget toolkit; their line counts describe what they compensate for, not what an
equivalent must cost.

`tree-view` also owns node-click semantics — allocation, ascendancy switching, undo
granularity — which are domain decisions living in a `Draw` handler. That is the single
largest logic-in-view concentration in the codebase.

`GemSelectControl` is type-ahead over the gem list with per-candidate DPS ranking; it calls
the calc engine once per candidate, so it is view code with a calculation dependency.

## Mixed

Domain logic and presentation interleaved. The seam column names where they separate.

| module | files | lines | seam |
|---|---|---|---|
| `tree` | `Classes/TreeTab.lua` | 2,873 | spec management, mastery eligibility, tattoos, power calc and timeless search are logic; the popup constructors are view. `ImportTree`/`ExportTree` show the separated shape |
| `items` | `Classes/ItemsTab.lua` `Modules/ItemSlotHelper.lua` | 5,243 | each crafting popup fuses an option-list builder, a DPS-sorted ranking and an apply step — those three are logic, the dialog is view |
| `skills` | `Classes/SkillsTab.lua` | 1,475 | socket-group processing, gem matching and `OptimiseSockets` are logic; gem slot rows are view |
| `calcs` | `Classes/CalcsTab.lua` `Modules/CalcSections.lua` | 3,366 | `CalcSections.lua` is 2,591 lines of section schema — data, not code. Power calculation and the modKey cache are logic; section layout and pinning are view |
| `config` | `Classes/ConfigTab.lua` `Modules/ConfigOptions.lua` `ConfigVisibility.lua` `Classes/ConfigSetListControl.lua` | 4,179 | 580 options (check 331, count 163, list 41, integer 11, float 1) carrying 524 embedded apply closures. `ConfigVisibility.lua` computes per-option relevance from mainEnv usage maps — logic. Widget rendering is view |
| `notes` | `Classes/NotesTab.lua` | 107 | text persistence is logic; colour-code buttons are view |
| `party` | `Classes/PartyTab.lua` | 1,037 | `ParseBuffs`, `setBuffExports`, `exportBuffs` are logic; the simple/advanced editors are view |
| `import` | `Classes/ImportTab.lua` | 1,931 | code generation, character download, re-import state preservation are logic; status display and option toggles are view |
| `compare` | `Classes/CompareTab.lua` `CompareEntry.lua` `CompareCalcsHelpers.lua` `ComparePowerReportListControl.lua` | 6,171 | `CompareEntry` builds a second in-process build with its own tabs and a stubbed party actor — logic. The diff sub-views reuse the other modules' views |
| `tooltips` | `Classes/Tooltip.lua` `TooltipHost.lua` `GemTooltip.lua` | 978 | `Tooltip` splits cleanly along its own method list: `Clear`/`AddLine`/`AddSeparator`/`SetRecipe` accumulate content, `GetSize`/`GetDynamicSize`/`CalculateColumns`/`Draw` render it. Content composition — `AddItemTooltip`, `AddStatComparesToTooltip`, `GemTooltip` — is logic and is roughly 3k lines living in `items` and `skills` |
| `trade` | `TradeQuery` `TradeQueryGenerator` `TradeQueryRequests` `TradeQueryRateLimiter` `TradeHelpers` `CompareBuySimilar` `TradeStatWeightMultiplierListControl` | 4,839 | weight generation, query building, rate-limit policy parsing are logic; `TradeQuery.lua` (1,376) is the view. The generator runs as a coroutine pumped by the frame loop with a progress popup |
| `app-shell` | `Modules/Main.lua` | 1,816 | mode management, settings load/save, user paths are logic; the popup registry, key handling and chrome are view |
| `build-list` | `Modules/BuildList.lua` `BuildListHelpers.lua` `Classes/BuildListControl.lua` `FolderListControl.lua` | 796 | scan, filter, sort, header parsing and folder moves are logic (`BuildListHelpers`); the screen is view |
| `ext-build-lists` | `ExtBuildListControl` `ExtBuildListProvider` `PoBArchivesProvider` | 646 | providers are logic; the listing is view |

## Tooling

| module | files | lines |
|---|---|---|
| `export-tooling` | `src/Export/` (65 files) | 51,545 |

Development tool for regenerating `Data/` from game files. Separate application with its own
`Main.lua` and `Launch.lua`.

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
```

Nothing depends on: `notes`, `party`, `toast`, `minion-library`, `ext-build-lists`,
`export-tooling`, `stat-describer`, `pantheon`.

## Cross-cutting behaviours

Properties of the application that no single module owns, and that anything reproducing a
module must preserve.

- **Recalculation is coalesced, never per-edit.** Edits set `buildFlag`; one `BuildOutput`
  runs per frame. `outputRevision` marks each rebuild.
- **Long operations are coroutines pumped by the frame loop**, with a blocking progress popup
  and no cancel: `trade`'s weight generation (`TradeQueryGenerator.lua:618`), build-list
  scanning. Timeless-jewel search blocks outright.
- **Undo is per-tab, not global.** Each mixed module is its own `UndoHandler` and handles
  Ctrl+Z itself while active (`CalcsTab:317`, `ConfigTab:982`, `ItemsTab:1466`,
  `SkillsTab:623`, `TreeTab:370`). `NotesTab` and `PartyTab` delegate to their edit controls.
- **Unsaved state is an OR across every module's `modFlag`** plus `spec.modFlag` and
  `treeTab.searchFlag` (`Build.lua:626`).
- **Statbox, warning and tooltip text is engine-composed**, carrying `^7` / `^xRRGGBB` colour
  codes and a per-line font size.
- **Nine view modes**: TREE SKILLS ITEMS CALCS CONFIG NOTES PARTY IMPORT COMPARE, persisted as
  a `viewMode` attribute on the build XML (`Build.lua:78,960`).
- **Errors and prompts are engine-raised**: `Launch:ShowErrMsg`, `main:ShowPrompt`,
  `main:OpenConfirmPopup`, update-available notices (`Main.lua:265-276`).
