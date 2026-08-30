# Module inventory & port tracker

Every unit of Path of Building as it exists complete in `.archive/` (this
branch's frozen Lua application). Modules are independently claimable once
their dependencies are met.

Two axes, tracked separately; full parity = both `[x]`:

- **code** — the module is *written*: every branch of the reference exists in
  Go. The reference is the scope, not the corpus: if it is in `.archive` it
  belongs here, including its bugs (reproduced faithfully, tagged `#EVAL`).
  `[x]` = nothing left behind a "not ported" guard.
- **archive** — it *behaves* like `.archive`: a differential test that fails
  on any disagreement and passes at 100% over everything the test corpus
  reaches — the modparser standard (13,173/13,173 corpus lines, 8,800/8,800
  table entries). `[x]` = the verified surface is the whole module; `[~]` =
  verified wherever exercised, covered fraction stated in the row.

`[ ]` not started · `[~]` partial · `[x]` complete.

- Coverage is a property of the test corpus, not of the port: written but
  corpus-unexercised = `code [x] / archive [~]` — the normal state, not a
  defect, and not a reason to leave a branch unwritten.
- Panics are not all equal: one mirroring a reference error ("the Lua errors
  here too") is parity and counts as written; one standing in for unported
  behaviour is a code gap and must be listed in the row.
- Reference branches *unreachable in the reference itself* (no caller, a dead
  `or` arm) count as written when the row says so.

Closing a guarded branch moves both axes at once — never "port it blind and
hope a build turns up later" (this method grew the corpus 9 → 12 builds,
cheap per gap):

1. Get an env that reaches it — `mb search` for a real ladder character with
   the mod/skill, or hand-author a throwaway build in `.archive/src/Builds/`
   with just enough on it to trip the guard.
2. Dump it (`tools/dump_calc.lua <key> <xml>`), add it to the variant map.
3. Write the branch, delete the guard, confirm byte-identical.

A module without a runnable Lua archive dump (pure view code) states in its
row what "verified" meant. Keep the reference files/lines columns untouched —
they describe the reference, not the port.

**`archive [x]` has a precision floor, not yet reviewed.** Both sides
serialise compared numbers at `%.14g`, so the differential cannot see a
disagreement below the 14th significant digit. Such a difference is invisible
until some later comparison bound flips on it, then appears as a whole-percent
output error — one already did: a 15th-digit difference in a cached source
rate ran the trigger simulation 1001 times instead of 1000
(`calc-core-plan.md`, harness bug 2), caught only because the amplified
result diverged. So every `archive [x]` below means **"agrees to 14
significant digits at each checkpoint"**, not "is the same double"; a green
run does not rule out sub-ulp drift a future branch amplifies. Outstanding —
a POST-PARITY REVIEW item, done once the modules are written, not per-gap:

1. Re-run the whole corpus with the compared canons at `%.17g` on BOTH sides
   and see what survives — the actual review.
2. ~~Five dumps still carried `%.14g` fixtures~~ — closed: their XML lives
   in `.archive/src/Builds/`, `test/corpus/manifest.tsv` maps every dump
   key to its build, and all 38 are re-dumped with exact fixtures (the
   fixture echo round-trips at `%.17g` via `luacanon.EncodeExact`).
   Separately, `GlobalCache` is no longer a fixture at all — the ported
   `buildOutput` driver computes it and the differential compares it —
   which removes the input channel the trigger-sim bug came through.
3. The sweep for Go's arbitrary-precision constant folding (`1 / 0.033` as a
   constant is one ulp off the reference's runtime division) was a regex over
   `data/` and `calc/` that checked seven candidates — not a proof: a
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
data pipeline rather than the runtime; both need a home in the repo proper
first:

- ~~`cmd/pobexport -tpl`~~ (CLOSED 2026-08-28: the hand-maintained
  templates — Export/Bases directive lists (with `#baseMatch` patterns
  rewritten as Go regex), the Export/Uniques item-text database, Skills/
  Minions/Enemies directive files, and ModFoulbornMap — live in-repo as
  conventional JSON under `export/templates/`, embedded into the exporter;
  the archive copies serve only the render test, which reconstructs the
  generated files' passthrough/wrapper text from them. The `-tpl` flag and
  `Ctx.TplDir` are gone.)
- ~~`internal/luapat`~~ (CLOSED 2026-08-28: now `test/luapat`, test-only —
  the table-archive differential maps the reference's Lua-pattern keys onto
  the shipped Go-regex ones. Every product conversion happened once at the
  source; **runtime code must never convert Lua patterns** — the shipped
  tables are plain Go regex, guarded by the metacharacter test in
  `test/itemtag_test.go`.)
- **Three shipped raw artifacts are regenerable only by Lua-dumping the
  archive** (each was an expedient during its stage, none was added to this
  gate at the time — corrected 2026-08-28): ~~tattoo pool~~ (CLOSED
  2026-08-28: tree reads data/raw/tattoopassives.json — the export document
  — directly; the Lua dump and tools/dump_tattoo.lua are deleted, and
  test/tattoo_test.go byte-guards the embedded document against the
  archive's TattooPassives.lua so an archive update fails loudly instead of
  drifting), ~~`data/raw/tree_3_29.json`~~ (CLOSED 2026-08-28: `cmd/treegen` /
  `export.BuildTreeDoc` reproduces PoB's whole tree ingestion — the
  fix_ascendancy_positions.py port, the jsonToLua regex pipeline's
  semantics, canon — from GGG's published skilltree-export JSON (tag
  3.29.1), byte-identical to the retired luajit dump modulo its CRLF tail;
  tools/dump_tree.lua deleted; test/treegen_test.go pins the artifact to
  the gzipped GGG source in testdata), and ~~`data/raw/modcache.jsonl`~~
  (CLOSED 2026-08-28: `internal/modcachegen` regenerates it from the Go
  parser — the REGENERATE_MOD_CACHE environment ported: empty parse cache,
  tree.Load ProcessStats parses, the unique+rare-template walk with Craft
  at affix quality 0.5, SaveModCache's sorted JewelFunc-skipping write with
  %.14g quantization; key set 13,173/13,173 exact, artifact regenerated LF;
  tools/dump_modcache.lua deleted; test/modcachegen_test.go is the guard;
  `cmd/sourceupdate -modcache-only` regenerates. Both this artifact's mod
  values and tree_3_29.json are CONVENTIONAL JSON (standing rule: artifacts
  never ship in Lua-table shape; luacanon/luarender live under test/ and
  the Lua-format conversion happens only in tests). Closing it surfaced two
  real parity bugs: item sanitiseText (now `item.FoldText`) was missing the ä/ö/cp1252/catch-all
  folds — Maelström-named bases never resolved — and tree tattoo loading
  parsed shadowed duplicate-name rows the reference collapses first).
  Test fixtures may keep Lua/luajit dependencies (they die with
  the archive); SHIPPED data must not.
- ~~Lua-shaped Go in production~~ (CLOSED 2026-08-29, the go-remodel plan:
  `modparser.Canon` and every `*Canon` adapter → `test/luacanon`;
  `tools/gen_skilldata.lua`/`gen_datatables.lua` deleted, their tables
  converted once to typed Go in `data/` with no regeneration path; the
  `Tag`/`D` maps → typed `Tag` kinds and `Value`; `Row.Get` → typed dat
  accessors; `luaStr`/`luaNum*`/`luaLower`/`sanitiseText` → `util.FormatG14`,
  `strings.ToLower`, `item.FoldText`; template `#directives` → structured
  JSON; `data/raw` embeds only what is read. `.archive/` remains the
  differential reference; nothing in the product reads it or emulates Lua.)

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
| `[x]` | `[x]` | `game-data` | `Modules/Data.lua` + `Data/` (134 files) | 798,584 | `/data` package + `test/gamedata_test.go`: 136 subtrees of the loaded `data` table compared canonically against the booted archive (tools/dump_gamedata.lua), 0 disagreements — the full load surface: all inline tables, mod pools, enchantments, item bases+lists, uniques (incl. the generated ones), skills (1,535 granted effects with structured mods, statMaps, template fragments), skillStatMap, gems + lookups, minions/spectres, cluster jewels, bosses+bossSkills, mapMods, timeless tables. Residuals owned by other modules: `UnportedFn` bodies (139 skill funcs, 41 mapMods applies → calc/config), 4 tree-built generated uniques (→ tree-data), describeStats (→ stat-describer), LUT readers (→ timeless-jewel-data), vocab.go regeneration (small follow-up). **code** `[x]` for the tables. It also carries 177 Lua closures as `UnportedFn` markers (135 skill callbacks in `skills_custom.go`, 42 map-mod `apply` closures in `mapmods.go`); those bodies belong to the consuming module, not here — 8 of the skill callbacks now live in `calc/skillfuncs.go` |
| `[~]` | `[~]` | `tree-data` | `Classes/PassiveTree.lua` + `TreeData/` (61 files) | 4,113,782 | **code**: `tree.Load` (package `tree`) ports the constructor's logic half for 3_29 (the only version the corpus uses; others panic at load): class/ascendancy maps, legacy alternate-ascendancy filtering, node typing, orbit geometry (positions), linkedId, ProcessStats (multiline split + line combining + modKey), keystone/notable/ascendancy/clusterNode maps, mastery effects, per-socket and per-keystone nodesInRadius, ConnectedTo<class>Start flags, and the legion (timeless) passive pool with abyss notable renaming. View half (sprites, assets, connectors) skipped. Data: `data/raw/tree_3_29.json` via `tools/dump_tree.lua`. **archive**: `tools/dump_tree.lua` dumps a freshly built PassiveTree (pre-calc — a calc stamps item sources onto shared keystone mod lists, which poisons the build fixtures as a tree reference); `test/tree_test.go` — 3,352 nodes + 381 legion passives + mastery effects + all four name maps + socket radius sets byte-identical. Tolerances, both grounded: func addresses in modKey masked (run-dependent in the reference itself), duplicate-named ungrouped notables accepted when content-identical (pairs-order pick), and one node's x coordinate differs at 1e-14 (libm sin one-ulp; coordinates are ~1e4, radius membership exact). Negative controls: +1 orbit radius fails 5; dropping mod sources fails >5. View half remains (sprites, assets, connectors); radius attribute sums pend a consumer |
| `[x]` | `[x]` | `timeless-jewel-data` | `Modules/DataAbyssJewelLookUpTableHelper.lua` `DataLegionLookUpTableHelper.lua` `DataJewelFileLoader.lua` | 632 | **code**: `tree/historic.go` replaces readLUT with computation — the reverse-engineered generation algorithm (TinyMT32-variant RNG seeded per (node graph id, jewel seed); weighted replacement/addition pools). Pool inputs are one raw artifact `data/raw/conquertables.json` (117KB, generated from the GGPK dat tables by `export.BuildConquerTables` — the MP_EXPORT-gated `test/export_test.go` regenerates and byte-compares it; the additions table's spec names two columns "SpawnWeight" and the real weight is the FIRST, read by index. 6 rows carry renamed ids vs the shipped LegionPassives — position is identity, bins agree). `TimelessPassive` reproduces readLUT's returned table (global ids + rolls; EH seed/20; notable-row gate as `index < sizeNotable` — the reference's `<=` reads a missing row and yields empty). JewelFileLoader/zip splitting not ported (nothing to load). Abyss types (7-11): `tree/historic.go` ports the open C# DatafileGenerator (LocalIdentity/TimelessJewelData, Generator branch) — weighted random walk from the socket over the tree graph (weights cluster 5 / notable 25 / attribute 1 / else 5, abyssSize 60, masteries and class starts gated), per-node modification rolls, Zorath's per-node blocks plus one-notable-per-ascendancy selection (single-seed TinyMT). The abyss generator draws with REJECTION SAMPLING while the legion bins are plain modulo — both proven per-bit; abyss node typing is TREE-based (attr regex on stat text, AscendancyNotable type 5) vs the legion dat-based typing; tree versions 7-11 hardcode minAdd/maxAdd 1/1 overriding the dat's 0/0; orphaned legacy notables (empty in+out, kept by PoB but absent from the generator's tree) are excluded from Zorath's node set. `AbyssPassive` mirrors readAbyssJewelLUT (types 7-10 socket walks; 11 path + ascendancy picks, empty-modification entries included). **archive**: `test/timeless_test.go` — every (node, seed) cell of all 6 legion types vs the shipped bins (inflated from the committed .zip/.zip.partN, the .bins beside them are PoB's inflate cache): 33,148,707 cells byte-identical; `tree/abyss_test.go` — every record of all 5 abyss LUTs: 663,684 socket records (4 types × 21 sockets × 7,901 seeds), 2,143 Zorath node blocks and 165,921 ascendancy selections byte-identical (headers, socket/node id lists and the rebuilt local-id mapping included). Abyss negative control: +1 seed skew fails wholesale (GV full variable-length records — replacement id + stat rolls, or addition ids + rolls; other types one local id per notable×seed; localIdToGlobalId decode from NodeIndexMapping). Negative control: +1 seed skew fails en masse. ~3s wall. |
| `[~]` | `[~]` | `item-model` | `Classes/Item.lua` `Modules/ItemTools.lua` | 3,093 | **code**: the parse half is written — `ParseRaw` end to end (spec lines, variants/variant groups, influences, catalysts, foulborn matching, the affix-limit/crafted tail), `BuildModList`/`BuildModListForSlotNum`/`calcLocal`/`getRangedModList`, and ItemTools' `applyRange`/`formatValue`/`applyValueScalar` with the full modScalability combination search; `item.LoadSaved` replicates ItemsTab:Load's per-`<Item>` protocol (attrib pre-seed, ParseRaw, ModRange, the second BuildModList). Unported: the in-game advanced-copy `{ }` format and its affix matching (panics), and the crafting half (`Craft`, `BuildRaw`, `MutateMod`, spawn weights beyond what parse needs). **archive**: `test/item_test.go` — every corpus `<Item>` parsed natively from the build XML and byte-compared against the calc fixtures' `itemsTab.items` projection: 1,034 of 1,034 items across 47 builds. Negative controls: %.14g→%.15g in the mod-cache quantization fails 1 item; +0.5 in calcLocal's MORE multiplier fails 4. **Trap discovered**: PoB preloads `Data/ModCache.lua` — parse results for its 13,173 lines come from the shipped file, whose numbers round-tripped through %.14g text. The port now ships those entries too: `data/raw/modcache.jsonl` (`tools/dump_modcache.lua`), served by `modparser` instead of parsing (~450µs parse vs ~µs decode; `modparser/modcache.go`). `test/modcache_test.go` proves per entry that the decode re-encodes to the file's exact bytes AND that a fresh parse with %.14g rounding produces the same — the shipped file has no stale entries. Negative control: rounding at 13 digits fails. The parser-level differentials pin fresh mode (`SetModCache(nil)`) because their dumps wiped the cache. Calc still consumes fixture items via ReplayInput — swapping in natively parsed items needs the tree port first (jewel-socket slots pull spec radius data). Full precision (dropping PoB's %.14g rounding once exact-match ceases to matter) is a one-call switch: stop installing the cache (`modparser.SetModCache(nil)`); the byte-locked differentials would then flag 15th-digit diffs by design. |
| `[ ]` | `[ ]` | `pantheon` | `Modules/PantheonTools.lua` | 18 | |

`Data/` is mostly declarative — `mod(...)` constructor calls, only 210
`function` occurrences across 798k lines. `tree-data` also owns sprite sheets,
orbit geometry, connector construction and cluster-jewel subgraph building.

## Logic — build state

| code | archive | module | files | lines | evidence & gaps |
|---|---|---|---|---|---|
| `[ ]` | `[ ]` | `build-core` | `Modules/Build.lua` | 2,091 | |
| `[~]` | `[~]` | `passive-spec` | `Classes/PassiveSpec.lua` | 2,407 | **code**: `tree.Spec` (tree/spec.go, specdeps.go) — spec nodes shadowing tree nodes (replaceNode identity short-circuit for the reference's metatable scheme), XML load (nodes/masteryEffects attrs, sockets), ImportFromNodeList, class/ascendancy selection, ResetNodes, the full BuildAllDependsAndPaths dependency/pruning analysis (FindStartFromNode with the Ascendant un-visit rule, orphan pruning, mastery effect application + counts, intuitive-leap-like and Impossible Escape radius rules, potentialDeps/intuitiveLeaps resolution), BuildPathFromNode, SetNodeDistanceToClassStart, GetShortestPathToClassStart, Split Personality path, PostLoad. Guarded (panic): timeless conquering (see the plan below). Cluster jewel subgraphs (BuildSubgraph + BuildClusterJewelGraphs with the legacy v1 hash conversion) and tattoo overrides (the committed `Data/TattooPassives.lua`, read directly from the export document `data/raw/tattoopassives.json` since 2026-08-28) are ported, plus granted-passive resolution (anoints and Forbidden Flame/Flesh through notableMap/ascendancyMap). **Timeless conquering (shipped 2026-08-27, per the 08-26 plan)**: `tree/conquer.go` ports the conquering branch of BuildAllDependsAndPaths — notable/keystone/normal replacement and augmentation per conqueror, replaceHelperFunc's roll substitution (per_minute round(v/60,1), permyriad /100, _ms /1000; `(min-max)` then bare-min gsub), NodeAdditionOrReplacementFromString (addition modList PREPENDS, sd/mods/modKey append; sd copied on write — spec nodes share the legion node's slices), ReconnectNodeToClassStart (Pure Talent flags). Legion pool got its array order back (`NodesOrdered`/`AdditionsOrdered` — the reference indexes legionNodes[77] positionally) and legion keystones their keystoneMod (processNodeStats, source `Tree<stringid>`). GV might/legacy additions merge iterates a Lua table with pairs() (LuaJIT int-key hash-slot order, deterministic): production merges in FIRST-SEEN order and records the blocks (SpecNode.TimelessAdditions) — the difference is display-only (which rolled line sits where); `test/luapairs_test.go` emulates the shipped LuaJIT's int-key table exactly (hashrot on double bits, bestasize bins `lj_fls(k-1)`, main-position eviction, freetop scan, rehash reinsertion order; verified over 252,060 sequences, tools/gen_pairs_orders.lua → test/testdata/pairs_orders.txt) and the differential permutes our mod list into reference order before byte-comparing (disabling the permutation fails 4 nodes). If the future calc-native integration surfaces mod ORDER in its checkpoints, revisit (user decision 2026-08-27: keep the emulation out of production until calc proves it matters). Traps hit: conqueredBy.id decodes as Go int (coerce, don't assert float64); conquered keystones nil-unshadow `name` (NodeInKeystoneRadius must read the effective name or Impossible Escape nodes get pruned — 4 nodes in the EH corpus build); the fixture's radius node data reads nodesInRadius off the SPEC socket node, nil when a cluster subgraph replaced it. Abyss conquering (specdeps collectAbyssConquests + the component branch of applyConquered — type 1 replaces via legionNodes, type 2 adds via legionAdditions, rolls substituted per statDesc index) is live; the 2 Reclaimed Malevolence (Zorath) builds compare clean, and the four walk-based uniques (Festering Vengeance/Extinguishing Grasp/Baleful Dominion/Destructive Aspiration, types 7-10) were verified by swapping each into the cocuser build against fresh dumps: 3 byte-exact, the 4th exact except one node where the REFERENCE ITSELF is per-process random — pairs(addition.stats) order under LuaJIT string-hash randomization decides which roll lands when two stats share identical '(min-max)' text (proven: 4 dump processes gave rolls 2,3,2,2). The port's dat-order pick is one of the reference's own outcomes; if a committed corpus build ever hits such a node, accept the swap-equivalent value. **archive**: `test/spec_test.go` — for every non-timeless corpus build: allocNodes projections (cluster subgraph nodes included), mastery/notable/keystone/tattoo counts, radius-jewel node data, and granted passive/ascendancy node resolution byte-identical vs the calc fixtures: 9,046 nodes across all 47 builds (legion-timeless and abyss-jewel builds included). Negative controls: truncating mastery stats fails 5; reversing cluster notable sort order fails 77; dropping tattoo overrides fails 187. Traps: 32 overrides carry their own `name` field — these are the runegrafts (overrideType `AlternateMastery`), which replace masteries and are not technically tattoos though PoB ships them in the same TattooPassives override pool (shadows the node's original name; the rest fall through, the reference's metatable nil-unshadowing); overridden masteries reparse with the numeric node id as source while other overrides keep the pool entry's string id. |
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
| `logic:[~] view:[ ]` | `skills` | `Classes/SkillsTab.lua` | 1,475 | socket-group processing, gem matching and `OptimiseSockets` are logic; gem slot rows are view. **logic**: package `skills` ports the load half — XML → socket groups/gems as scalar bags (attribute presence mirrors Lua: a saved "nil" string is present-and-falsey-compared), gem resolution (gemId/variantId/skillId branches, the object-keyed gemForSkill lookup, FindSkillGem nameSpec migration via calc's port + its errMsg), ProcessSocketGroup (validateGemLevel, req fields; colour codes skipped as view), default-gem-level legacy mapping, RebuildImbuedSupportBySlot, UpdateSocketGroups (socket-colour matching; a missing socket index DELETES matchesSocket — Lua nil assignment). `test/skills_test.go`: 369 groups / 1,001 gems byte-identical across 47 builds; masked with owners stated: calc-stamped keys (skillPart/skillMinion/stage/mine families, triggered), view keys (color), granted (`source`) groups compare XML-owned keys only (the calc's granted update rewrites their gems — ported in calc, proven by the bridge). The calc native bridge feeds calc the native tab (reduced variants: groups wiped, calc recreates granted ones; stale imbued map semantics kept): 145 variants byte-identical; negative control: one gem level +1 fails 319 comparisons. Unported: OptimiseSockets, ProcessGemLevel (new-gem defaulting; nothing on the load path calls it). |
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
consumes — as **structured JSON documents** typed by `data/schema` (one per
script), no Lua in the pipeline. Ported to `/export` + `cmd/pobexport`: dat64
reader (`dat.go`), column schemas (`spec.go`, one-time transform of
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
