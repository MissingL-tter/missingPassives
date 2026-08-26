# Module inventory & port tracker

Every unit of Path of Building, as it exists complete in `.archive/` (this
branch's frozen Lua application). Modules are independently claimable; anything
with its dependencies met can be picked up at any time.

**Two axes, tracked separately.** Full parity means both are `[x]`:

- **code** — is the module *written*? Every branch of the reference exists in
  Go. **The reference is the scope, not the corpus**: if it is in `.archive`
  it belongs here, including its bugs (reproduced faithfully and tagged
  `#EVAL`). `[x]` means nothing is left behind a "not ported" guard.
- **archive** — does it *behave* like `.archive`? A differential test that
  **fails on any disagreement** and passes at 100% over everything the test
  corpus reaches — the modparser standard (13,173/13,173 corpus lines,
  8,800/8,800 table entries). `[x]` means the verified surface is the whole
  module; `[~]` means verified wherever exercised, with the covered fraction
  stated in the row.

`[ ]` not started · `[~]` partial · `[x]` complete.

Two rules that follow from the split:

- **Coverage is a property of the test corpus, not of the port.** A branch
  that is written but no corpus build exercises is `code [x] / archive [~]`.
  That is the normal state, not a defect — and it is not a reason to leave
  the branch unwritten.
- **Panics are not all equal.** A panic that mirrors an error in the
  reference ("the Lua errors here too") is parity and counts as written. A
  panic standing in for behaviour nobody has ported yet is a code gap and
  must be listed in the row.

Reference branches that are *unreachable in the reference itself* (no caller,
a dead `or` arm) count as written when the row says so — they cannot affect
behaviour either way.

**Closing a gap moves both axes at once.** The working method for a guarded
branch is not "port it blind and hope a build turns up later":

1. Get an env that reaches it — `mb search` for a real ladder character with
   the mod/skill, or hand-author a throwaway build in `.archive/src/Builds/`
   with just enough on it to trip the guard. It does not need to be a good
   build, only a reaching one.
2. Dump it (`tools/dump_calc.lua <key> <xml>`), add it to the variant map.
3. Write the branch, delete the guard, and confirm byte-identical.

That is how the corpus grew from 9 builds to 12; it is cheap per gap and it
means `code [x]` and `archive [x]` land together instead of drifting apart.

A module without a runnable Lua archive dump (pure view code) states in its
row what "verified" meant. Keep the reference files/lines columns untouched —
they describe the reference, not the port.

**`archive [x]` has a precision floor, and it is not yet reviewed.** Both
sides serialise compared numbers at `%.14g`, so the differential cannot see a
disagreement below the 14th significant digit. That is not a rounding
nicety: such a difference is invisible until some later comparison bound
flips on it, and then it appears as a whole-percent output error. One already
did — a 15th-digit difference in a cached source rate ran the trigger
simulation 1001 times instead of 1000 (`calc-core-plan.md`, harness bug 2) —
and it was only caught because the amplified result diverged, not because the
input comparison failed.

So every `archive [x]` below currently means **"agrees to 14 significant
digits at each checkpoint"**, not "is the same double". Nothing about a green
run today rules out sub-ulp drift that a future branch amplifies. Three
things are outstanding, and they are a POST-PARITY REVIEW item — do them once
the modules are written, not per-gap:

1. Re-run the whole corpus with the compared canons at `%.17g` on BOTH sides
   and see what survives. This is the actual review; everything else here is
   a symptom of not having done it.
2. ~~Five dumps still carried `%.14g` fixtures~~ — closed: their XML lives
   in `.archive/src/Builds/`, `test/corpus/manifest.tsv` maps every dump
   key to its build, and all 38 are re-dumped with exact fixtures (the
   fixture echo round-trips at `%.17g` via `luacanon.EncodeExact`).
   Separately, `GlobalCache` is no longer a fixture at all — the ported
   `buildOutput` driver computes it and the differential compares it —
   which removes the input channel the trigger-sim bug came through.
3. The sweep for Go's arbitrary-precision constant folding (`1 / 0.033` as a
   constant is one ulp off the reference's runtime division) was a regex over
   `data/` and `calc/` that checked seven candidates. It is not a proof: a
   constant expression with non-representable operands can hide in any
   `const` block or literal table entry.

**Kind** — how the module divides, the axis decoupling runs along:

- `logic` — calculation, parsing, game data, persistent state. No presentation.
- `view` — presentation, layout, input handling. No domain knowledge.
- `mixed` — both, interleaved. The seam is named per row; port the logic half
  first, verify it against the archive, then build the view half on top.

All paths relative to `.archive/src/`. Line counts measured 2026-08-19.

**Deleting `.archive/` is gated on more than the rows below.** Two known
dependencies on the Lua tree survive a fully-ported checklist, both in the
data pipeline rather than the runtime, and both need a home in the repo
proper before the archive can go:

- `cmd/pobexport -tpl` defaults to `.archive/src` for the hand-maintained
  `Export/Uniques` and `Export/Skills` templates, so regenerating game data
  still needs the Lua tree on disk.
- `internal/luapat` (Lua-pattern → Go regex) is build-time tooling whose own
  header says it is "deleted together with the Lua". Its one non-test caller
  is `export/script_bases.go` (the `baseMatch` directive), which is correct —
  it interprets Lua-pattern-shaped export specs. **Runtime code must never
  call it**: convert once, ship Go regex in the data, and guard the shipped
  table with a metacharacter test (see `test/itemtag_test.go`).

## Progress

Counted on both axes; full parity needs both.

| | code written | archive-verified |
|---|---|---|
| complete | 4 — `mod-parser`, `mod-store`, `game-data`, `export-tooling` | 4 — same |
| partial | 3 — `calc-core`, `calc-offence`, `calc-defence` | 3 — same |
| not started | 24 | 24 |

(31 modules; the 7 asset families are tracked separately below.)

The three partial rows are where the two axes come apart: their archive
coverage is high (75 variants, byte-identical on every checkpoint) while
their code still has guarded branches. Open code gaps, largest first:

| gap | count |
|---|---|
| skill callbacks (`UnportedFn` → `calc/skillfuncs.go`) | 97 of 131 |
| trigger configs (`CalcTriggers.lua` configTable) | **0 of 80 — all written**; roughly half verified, the rest await an authored build |
| map-mod `apply` closures (`Data/ModMap.lua`) | 42 |
| guard panics across `calc/` | 1 real (the per-skill callback gate) + party-tab crit (**deferred — see `party`**) + 3 defensive assertions |

---

## Logic — calculation

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[~]` | `[~]` | `calc-core` | `Modules/Calcs.lua` `CalcSetup.lua` `CalcPerform.lua` `CalcTools.lua` `CalcFormat.lua` | 7,048 | **code**: `CalcTools.lua`, `CalcSetup.lua` (initEnv: tree merge, items, skills/supports, buildActiveSkillModList, radius/threshold/transform jewels, granted-skill and Explode groups, Energy Blade re-entry) and the whole `CalcPerform.lua` body are written. `Calcs.lua`'s cache half is written — `cacheData`, `buildActiveSkill`, the `buildOutput` cache fill (`calc/buildoutput.go`), `PerformFull` with the real handoff — so `GlobalCache` is computed and compared, not a fixture. Unwritten: `buildOutput`'s display half (FullDPS roll-up, cost warnings, conditions/multipliers discovery), `CalcFormat.lua`, CALCS mode, and the remaining guarded branches at the initEnv stage — `ProcessSocketGroup` nameSpec migration, `ExtraJewelFunc` re-entry, minion damage fixup — plus 5 in perform (`getCachedOutputValue` stage caches, `calcs.resistances` Choir path, non-empty `AffectedByAuraMod`). CALCULATOR mode is now written (copyActiveSkill's second initEnv); CALCS still is not. **archive**: `test/calc_test.go` — 147 variants across 48 corpus builds byte-identical post-initEnv (`.dbs`, `.skills`, `.skillLists`) and post-perform (`.performDbs`/`.performOutput`/`.performMinion*`), plus a fixture round-trip (`TestCalcFixtureEcho`) with corruption negatives |
| `[~]` | `[~]` | `calc-offence` | `Modules/CalcOffence.lua` `CalcActiveSkill.lua` `CalcMirages.lua` `CalcTriggers.lua` | 9,145 | **code**: `CalcOffence.lua` written end to end, `radiusTertiaryBaseMargin` and `explosiveArrowFunc` included; `CalcActiveSkill.lua` landed with calc-core, plus `copyActiveSkill`. `CalcTriggers.lua`: **all 80 configTable entries written**, plus `CWCHandler`, `helmetFocusHandler`, and the Arcanist Brand / Battlemage / Infernal Cry / Manaforged / Kitava mana-spent / Doom Blast-vixen handler branches. Doom Blast's expiration/hexblast sources and `stagesAreOverlaps` (a reference-dead hook no entry sets) are written too — the trigger module is complete. `CalcMirages.lua`: complete — all five paths plus the copyActiveSkill minion branch. PvP scaling (offence + EHP) written; party-tab crit stays guarded (**party is deferred by decision**, 2026-08-26 — no party tab planned). Skill callbacks: 34 of 131 `UnportedFn` markers ported (`calc/skillfuncs.go`). **archive**: `test/calc_test.go` — 147 of 147 variants byte-identical (38 ladder + 9 authored shells + the hand-made mjolner build; `test/corpus/authored_*.xml` exist purely to reach guards) on every stage checkpoint, no slice skipped. Negative controls: +1e-7 on `AverageHit` fails all; +1e-9 on the simulated trigger rate fails exactly the 7 checkpoints of the two trigger-driven skills; +1 on the mirage count fails `mirage.full`. Corpus exercises: dual wield, Unleash seals, bifurcated crit, returning projectiles, bleed/poison/ignite/impale, brands, chaining, projectiles, pierce, DoT, traps, mines, ballistas, mirages, cast-when-damage-taken, cast-on-crit, cast-while-channelling |
| `[~]` | `[~]` | `calc-defence` | `Modules/CalcDefence.lua` | 3,828 | **code**: `resistances`, `defence` and `buildDefenceEstimations` written in full — resistance conversion/caps, block, primary defences, evade, suppression, dodge, regen, recharge, recoup, damage reduction, avoidance, self-ailment duration, then the EHP half (pool drains, `numberOfHitsToDie`, max hit taken, EHP vs dots, degens). PvP scaling written. The 4 party-tab branches (block/max-block/max-life-leech equal-to-party, `MovementSpeedEqualHighestLinkedPlayers`) and `TakenFromPartyMemberESBeforeYou` stay guarded — **party is deferred by decision** (2026-08-26). **archive**: `test/calc_test.go` — 147 variants byte-identical on `.defence*` and `.ehp*` (mod/enemy/item DBs plus player and minion output, 506 output keys incl. `TotalEHP` to 13 significant digits). Negative controls: a 1e-7 perturbation inside `reducePoolsByDamage` and a one-value change in the defence tail each fail every variant |
| `[ ]` | `[ ]` | `calc-breakdown` | `Modules/CalcBreakdown.lua` | 251 | |
| `[x]` | `[x]` | `mod-parser` | `Modules/ModParser.lua` `ModTools.lua` | 7,193 | `test/parse_test.go` 13,173/13,173 · `test/tables_test.go` 8,800/8,800 · `test/modtools_test.go` 12,736 mods x 7 behaviours, 0 disagreements (format*/parseTags/parseFormattedSourceMod/compareModParams/setSource + the deep-copying parse cache). `mergeKeystones` reassigned to `mod-store` — it operates on a live ModDB and the tree keystone map. **code** `[x]`: its 3 panics are all error parity (the reference errors on the same input) |
| `[x]` | `[x]` | `mod-store` | `Classes/ModStore.lua` `ModDB.lua` `ModList.lua` | 1,530 | `test/modstore_test.go` 18,525 checks, 0 disagreements: the parsed corpus distributed over a store tree with fixture actors/configs, every aggregation (Sum/More/Flag/Override/List/Tabulate/HasMod/Max/Min/GetMultiplier/GetCondition), construction behaviours (ScaleAdd/Merge/Replace/Convert), `mergeKeystones`, plus 59 synthetic mods covering the tag branches the corpus never produces. Reference crashes are part of the contract (recorded as error sentinels, the port fails identically). **code** `[x]`: all 9 panics are error parity — every one is a shape the reference itself errors on |
| `[ ]` | `[ ]` | `stat-describer` | `Modules/StatDescriber.lua` | 292 | |

`mod-parser` converts the game's English mod text into structured modifiers:
~4,200 patterns composed by form / name / flag / tag scanners, plus ~3,900
lines of per-item special cases. The Go port keys every pattern table with
ordinary Go regex.

`calc-breakdown` produces per-stat derivation payloads (simple, mod, slot,
area, effMult, dot, critDot, leech, multiChain) consumed by `calcs-view`.

## Logic — game data

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[x]` | `[x]` | `game-data` | `Modules/Data.lua` + `Data/` (134 files) | 798,584 | `/data` package + `test/gamedata_test.go`: 136 subtrees of the loaded `data` table compared canonically against the booted archive (tools/dump_gamedata.lua), 0 disagreements — the full load surface: all inline tables, mod pools, enchantments, item bases+lists, uniques (incl. the generated ones), skills (1,535 granted effects with structured mods, statMaps, template fragments), skillStatMap, gems + lookups, minions/spectres, cluster jewels, bosses+bossSkills, mapMods, timeless tables. Residuals owned by other modules: `UnportedFn` bodies (139 skill funcs, 41 mapMods applies → calc/config), 4 tree-built generated uniques (→ tree-data), describeStats (→ stat-describer), LUT readers (→ timeless-jewel-data), vocab.go regeneration (small follow-up). **code** `[x]` for the tables. It also carries 177 Lua closures as `UnportedFn` markers (135 skill callbacks in `skills_custom_gen.go`, 42 map-mod `apply` closures in `mapmods_gen.go`); those bodies belong to the consuming module, not here — 8 of the skill callbacks now live in `calc/skillfuncs.go` |
| `[~]` | `[~]` | `tree-data` | `Classes/PassiveTree.lua` + `TreeData/` (61 files) | 4,113,782 | **code**: `tree.Load` (package `tree`) ports the constructor's logic half for 3_29 (the only version the corpus uses; others panic at load): class/ascendancy maps, legacy alternate-ascendancy filtering, node typing, orbit geometry (positions), linkedId, ProcessStats (multiline split + line combining + modKey), keystone/notable/ascendancy/clusterNode maps, mastery effects, per-socket and per-keystone nodesInRadius, ConnectedTo<class>Start flags, and the legion (timeless) passive pool with abyss notable renaming. View half (sprites, assets, connectors) skipped. Data: `data/raw/tree_3_29.json` via `tools/dump_tree.lua`. **archive**: `tools/dump_tree_archive.lua` dumps a freshly built PassiveTree (pre-calc — a calc stamps item sources onto shared keystone mod lists, which poisons the build fixtures as a tree reference); `test/tree_test.go` — 3,352 nodes + 381 legion passives + mastery effects + all four name maps + socket radius sets byte-identical. Tolerances, both grounded: func addresses in modKey masked (run-dependent in the reference itself), duplicate-named ungrouped notables accepted when content-identical (pairs-order pick), and one node's x coordinate differs at 1e-14 (libm sin one-ulp; coordinates are ~1e4, radius membership exact). Negative controls: +1 orbit radius fails 5; dropping mod sources fails >5. Next: PassiveSpec (allocation, cluster graphs, radius attribute sums, timeless conquering — the LUT packaging decision pends) |
| `[ ]` | `[ ]` | `timeless-jewel-data` | `Modules/DataAbyssJewelLookUpTableHelper.lua` `DataLegionLookUpTableHelper.lua` `DataJewelFileLoader.lua` | 632 | |
| `[~]` | `[~]` | `item-model` | `Classes/Item.lua` `Modules/ItemTools.lua` | 3,093 | **code**: the parse half is written — `ParseRaw` end to end (spec lines, variants/variant groups, influences, catalysts, foulborn matching, the affix-limit/crafted tail), `BuildModList`/`BuildModListForSlotNum`/`calcLocal`/`getRangedModList`, and ItemTools' `applyRange`/`formatValue`/`applyValueScalar` with the full modScalability combination search; `item.LoadSaved` replicates ItemsTab:Load's per-`<Item>` protocol (attrib pre-seed, ParseRaw, ModRange, the second BuildModList). Unported: the in-game advanced-copy `{ }` format and its affix matching (panics), and the crafting half (`Craft`, `BuildRaw`, `MutateMod`, spawn weights beyond what parse needs). **archive**: `test/item_test.go` — every corpus `<Item>` parsed natively from the build XML and byte-compared against the calc fixtures' `itemsTab.items` projection: 1,034 of 1,034 items across 47 builds. Negative controls: %.14g→%.15g in the mod-cache quantization fails 1 item; +0.5 in calcLocal's MORE multiplier fails 4. **Trap discovered**: PoB preloads `Data/ModCache.lua` — parse results for its 13,173 lines come from the shipped file, whose numbers round-tripped through %.14g text. The port now ships those entries too: `data/raw/modCache.jsonl` (`tools/dump_modcache.lua`), served by `modparser` instead of parsing (~450µs parse vs ~µs decode; `modparser/modcache.go`). `test/modcache_test.go` proves per entry that the decode re-encodes to the file's exact bytes AND that a fresh parse with %.14g rounding produces the same — the shipped file has no stale entries. Negative control: rounding at 13 digits fails. The parser-level differentials pin fresh mode (`SetModCache(nil)`) because their dumps wiped the cache. Calc still consumes fixture items via ReplayInput — swapping in natively parsed items needs the tree port first (jewel-socket slots pull spec radius data). Full precision (dropping PoB's %.14g rounding once exact-match ceases to matter) is a one-call switch: stop installing the cache (`modparser.SetModCache(nil)`); the byte-locked differentials would then flag 15th-digit diffs by design. |
| `[ ]` | `[ ]` | `pantheon` | `Modules/PantheonTools.lua` | 18 | |

`Data/` is mostly declarative — `mod(...)` constructor calls, only 210
`function` occurrences across 798k lines. `tree-data` also owns sprite sheets,
orbit geometry, connector construction and cluster-jewel subgraph building.

## Logic — build state

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[ ]` | `[ ]` | `build-core` | `Modules/Build.lua` | 2,091 | |
| `[~]` | `[~]` | `passive-spec` | `Classes/PassiveSpec.lua` | 2,407 | **code**: `tree.Spec` (tree/spec.go, specdeps.go) — spec nodes shadowing tree nodes (replaceNode identity short-circuit for the reference's metatable scheme), XML load (nodes/masteryEffects attrs, sockets), ImportFromNodeList, class/ascendancy selection, ResetNodes, the full BuildAllDependsAndPaths dependency/pruning analysis (FindStartFromNode with the Ascendant un-visit rule, orphan pruning, mastery effect application + counts, intuitive-leap-like and Impossible Escape radius rules, potentialDeps/intuitiveLeaps resolution), BuildPathFromNode, SetNodeDistanceToClassStart, GetShortestPathToClassStart, Split Personality path, PostLoad. Guarded (panic): timeless conquering (see the plan below). Cluster jewel subgraphs (BuildSubgraph + BuildClusterJewelGraphs with the legacy v1 hash conversion) and tattoo overrides (the committed `Data/TattooPassives.lua`, dumped to `data/raw/tattooOverrides.json` by `tools/dump_tattoo.lua` — the GGPK-export tattooPassives.json differs in shape and is NOT the runtime file) are ported, plus granted-passive resolution (anoints and Forbidden Flame/Flesh through notableMap/ascendancyMap). **Timeless plan (decided 2026-08-26)**: do NOT ship the LUT bins — port the reverse-engineered generation algorithm instead (openly published in several GitHub reimplementations, e.g. the C# TimelessEmulator that generated PoB's tables and a Go seed-search implementation; user ruled provenance a non-issue). compute(seed, node) replaces every table read, including seed search (~8,000 seeds × ~50 in-radius nodes ≈ 400k evaluations, milliseconds; the table is just memoization). Differential: run the algorithm over every (node, seed) cell and byte-compare against the shipped bins in `.archive/src/Data/TimelessJewelData/` (~60M cells, exhaustive) — the bins stay in the repo as the verification reference and nothing ships in the binary; any cell the algorithm cannot reproduce becomes an explicit override list, since the bins are PoB's behavior contract. The algorithm's own weight/pool inputs become one small raw artifact. Abyss jewel types (7-11) have their own LUTs/readers — scope them when a corpus build needs them. **archive**: `test/spec_test.go` — for every non-timeless corpus build: allocNodes projections (cluster subgraph nodes included), mastery/notable/keystone/tattoo counts, radius-jewel node data, and granted passive/ascendancy node resolution byte-identical vs the calc fixtures: 3,463 nodes across 24 builds; 23 timeless builds wait on the algorithm stage. Negative controls: truncating mastery stats fails 5; reversing cluster notable sort order fails 77; dropping tattoo overrides fails 187. Traps: 32 tattoo overrides carry their own `name` field (shadows the node's original; the rest fall through, the reference's metatable nil-unshadowing); tattooed masteries reparse with the numeric node id as source while other tattoos keep the tattoo's string id. |
| `[ ]` | `[ ]` | `undo` | `Classes/UndoHandler.lua` | 51 | |
| `[ ]` | `[ ]` | `display-stats` | `Modules/BuildDisplayStats.lua` | 250 | output byte-locked (memory `statbox-byte-lock`) — the archive comparison should be byte-level |
| `[ ]` | `[ ]` | `common` | `Modules/Common.lua` `Utils.lua` | 1,209 | port pieces on demand; note what landed here |
| `[ ]` | `[ ]` | `headless-adapter` | `HeadlessWrapper.lua` | 227 | the host-environment contract; its Go analogue emerges from whatever the ported modules need |

`undo` is a snapshot stack; `AddUndoState` also sets `modFlag` on its owner.
Granularity is decided per interaction, not per state change — a same-class
ascendancy click pushes two states (`PassiveTreeView.lua:433,504`), a
cross-class click pushes one covering class, ascendancy and node (`:466`), and
the confirm-prompt path pushes inside the callback (`:475`).

`build-core` drives recalculation by coalescing: `buildFlag` set by any edit,
one `BuildOutput` per frame (`Build.lua:613-624`), with `outputRevision`
incremented per rebuild.

## Logic — networking

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[ ]` | `[ ]` | `poe-api` | `Classes/PoEAPI.lua` `LaunchServer.lua` | 447 | |
| `[ ]` | `[ ]` | `build-sites` | `Modules/BuildSiteTools.lua` | 129 | |
| `[ ]` | `[ ]` | `launcher` | `Launch.lua` `LaunchInstall.lua` `GameVersions.lua` | 719 | |
| `[ ]` | `[ ]` | `updater` | `UpdateCheck.lua` `UpdateApply.lua` | 387 | |

`launcher` owns subscript management and `DownloadPage` (a curl subscript);
every other module's network access goes through it. `poe-api` covers OAuth
token fetch, validation, character download with rate limiting, and the local
redirect catcher.

## View

No domain knowledge; presentation and input only. No Lua archive dump is possible —
each row must state its own verification when checked off (golden screenshots,
DOM/state assertions, or manual sign-off noted here).

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[ ]` | `[ ]` | `control-kit` | `Control` `ControlHost` `Button` `CheckBox` `DropDown` `Edit` `Label` `Path` `PopupDialog` `ResizableEdit` `ScrollBar` `Section` `Slider` `TextList` `TooltipHost` `RectangleOutline` `Dragger` `SearchHost` `ListControl` | 3,498 | |
| `[ ]` | `[ ]` | `tree-view` | `Classes/PassiveTreeView.lua` `PassiveMasteryControl.lua` `PassiveSpecListControl.lua` `PowerReportListControl.lua` | 2,153 | |
| `[ ]` | `[ ]` | `items-view` | `ItemDBControl` `ItemListControl` `ItemSetListControl` `ItemSlotControl` `NotableDBControl` `SharedItemListControl` `SharedItemSetListControl` | 1,543 | |
| `[ ]` | `[ ]` | `skills-view` | `SkillListControl` `SkillSetListControl` `GemSelectControl` | 1,223 | |
| `[ ]` | `[ ]` | `calcs-view` | `CalcSectionControl` `CalcBreakdownControl` | 1,295 | |
| `[ ]` | `[ ]` | `timeless-jewel-view` | `TimelessJewelListControl` `TimelessJewelSocketControl` | 359 | |
| `[ ]` | `[ ]` | `minion-library` | `MinionListControl` `MinionSearchListControl` | 170 | |
| `[ ]` | `[ ]` | `toast` | `Modules/ToastNotification.lua` | 257 | |

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

| code | archive | module | files | lines | seam |
|---|---|---|---|---|---|
| `logic:[ ] view:[ ]` | `tree` | `Classes/TreeTab.lua` | 2,873 | spec management, mastery eligibility, tattoos, power calc and timeless search are logic; the popup constructors are view |
| `logic:[ ] view:[ ]` | `items` | `Classes/ItemsTab.lua` `Modules/ItemSlotHelper.lua` | 5,243 | each crafting popup fuses an option-list builder, a DPS-sorted ranking and an apply step — those three are logic, the dialog is view |
| `logic:[ ] view:[ ]` | `skills` | `Classes/SkillsTab.lua` | 1,475 | socket-group processing, gem matching and `OptimiseSockets` are logic; gem slot rows are view |
| `logic:[ ] view:[ ]` | `calcs` | `Classes/CalcsTab.lua` `Modules/CalcSections.lua` | 3,366 | `CalcSections.lua` is 2,591 lines of section schema — data, not code. Power calculation and the modKey cache are logic; section layout and pinning are view |
| `logic:[ ] view:[ ]` | `config` | `Classes/ConfigTab.lua` `Modules/ConfigOptions.lua` `ConfigVisibility.lua` `Classes/ConfigSetListControl.lua` | 4,179 | 580 options (check 331, count 163, list 41, integer 11, float 1) carrying 524 embedded apply closures. `ConfigVisibility.lua` computes per-option relevance from mainEnv usage maps — logic. Widget rendering is view |
| `logic:[ ] view:[ ]` | `notes` | `Classes/NotesTab.lua` | 107 | text persistence is logic; colour-code buttons are view |
| `logic:[ ] view:[ ]` | `party` **(DEFERRED by decision 2026-08-26: no party tab planned; its calc guards stay in place and are not gaps)** | `Classes/PartyTab.lua` | 1,037 | `ParseBuffs`, `setBuffExports`, `exportBuffs` are logic; the simple/advanced editors are view |
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

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[x]` | `[x]` | `export-tooling` | `src/Export/` (65 files) | 51,545 | `test/export_test.go` 123/123 files byte-identical. **code** `[x]`: two reference branches are deliberately absent because they are unreachable in the reference — enchant.lua's "Craft" generation arm (no caller passes a Craft key) and skillgemlist.lua's `grantedEffectString` side (its `export` flag is false, and its `ipairs` over a single-Key cell yields nothing under LuaJIT anyway) |

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
`common`, `notes` (logic), `toast`, `minion-library`,
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
