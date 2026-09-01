# knowledge.md — what this project knows

Durable reference for the Go rebuild of Path of Building: how the reference
application works, how the conversion is verified, and every trap that cost
real time. Not a plan and not a status board — those are `parity.md` (module
tracker, current status) and the finished stage documents under
`deprecated/`. Facts here should stay true as the port advances; anything
time-bound is marked *(as of 2026-08-31)*.

Companion documents, and what each owns:

| document | owns |
|---|---|
| `parity.md` | per-module status, the two-axis completion rule, differential inventory |
| `poe-data-model.md` | Path of Exile domain semantics (modifiers, pools, skills, jewels, bases) |
| `later.md` | Lua-derived code kept by decision + the `#EVAL` quirk inventory |
| `deprecated/go-remodel-plan.md`, `deprecated/lua-gtfo.md`, `deprecated/lua-residue.md` | the three Lua-eradication stages (2026-08-29/30/31) — finished work, kept for their reasoning and precedents; only lua-residue.md's opportunistic list is still live |
| `calc-core-plan.md` | calc porting notes (statuses drift; reasoning is sound) |
| `lua-go-map.md` | every deliberate reference→Go name divergence |
| `CLAUDE.md` (repo root) | the standing rules the stages produced |

---

## 1. The two programs

**Path of Building** is a Lua desktop application for planning Path of Exile
characters: passive tree, items, skill gems, configuration → a full damage and
defence calculation. It runs on SimpleGraphic, a minimal engine providing a
window, a draw API and no widget toolkit — which is why ~68 of the 74 files in
`Classes/` are hand-built UI controls. It is ~5.05M lines across 374 Lua
files, but almost all of that is data: `Data/` is ~798k lines with only ~210
`function` occurrences.

Its own split, worth knowing when hunting for a file: `src/Modules/` (33
files, 41.5k lines) holds the engine and singletons — the calculation stages,
`Data.lua`, the modifier parser, `Common.lua`, `Build.lua`, `Main.lua`.
`src/Classes/` (74 files, 46.2k lines) holds the instantiable object model
(`Item`, `ModDB`, `ModList`, `ModStore`, `PassiveTree`, `PassiveSpec`,
`UndoHandler`) plus the entire UI, one file per tab and per widget.

**This repository** is a fork of PathOfBuildingCommunity/PathOfBuilding (9,547
commits, `upstream` remote present). Branch `master` is upstream; `flashbang`
strips the Lua UI so another language can drive the Lua engine; **`go` carries
the Go rebuild**, started 2026-08-19. The Lua application was moved wholesale
into `.archive/` and the Go module lives at the repository root.

The rebuild is engine-first: calculation, parsing, game data and the data
pipeline are ported; no presentation code exists and there is no runnable Go
application. Since 2026-08-31 a saved build computes end to end from its
XML alone (`build.Load` -> `calc.BuildOutput`, config included), but
nothing drives that outside the tests. *(as of 2026-08-31)* ~65,800 lines of production Go, 10,700
of test Go, **zero third-party dependencies**, Go 1.26, no CI, no Makefile, no
`go:generate`.

---

## 2. The archive

`.archive/` is a git submodule (`missingPassivesArchive`, origin; `upstream`
= PathOfBuildingCommunity) pinned at **v2.67.2 + 59 fork commits**
(`v2.67.2-59-g40e64620d`). It is read-only: never edit, move or regenerate it.
Terminology, set by the user: it is **"the archive"**, never "the oracle";
recorded reference behaviour is an **"archive dump"**.

- Working tree ~70 GB, dominated by `src/Export/ggpk/Content.ggpk` (73 GB),
  which is excluded only by a **local, uncommitted** rule in
  `.git/modules/.archive/info/exclude`. A fresh submodule clone loses that
  exclusion.
- `.archive/src/` holds the application; `.archive/src/Builds/` is the user's
  **live** builds directory — it mutates while they play. Corpus builds must
  be frozen copies under `test/corpus/`, never referenced from there. A
  fixture once drifted 126→130 allocated nodes mid-session with no code change.
- The `/bump` skill rebases the submodule onto a newer PoB release, tagging
  the outgoing tip `archive-<BASE>` first so archive SHAs pinned by past `go`
  commits stay resolvable after the rebase orphans them. A bump must
  re-extract the GGPK before regenerating anything — the extract is not in git,
  so stale game data regenerates silently.

Booting it headlessly: `dofile("HeadlessWrapper.lua")` from `.archive/src/`
stubs the SimpleGraphic API, loads `Launch.lua`, runs `OnInit` and one
`OnFrame`. Afterwards `data`, `modLib`, `LoadModule`, `new`, `build`,
`newBuild()`, `loadBuildFromXML(xml, name)`, `runCallback(name)` and
`wipeGlobalCache()` are live. **The wrapper stubs `Inflate`/`Deflate` to the
empty string** — build tree specs are zlib-compressed URLs, so without an FFI
rebind to `runtime/zlib1.dll` the spec silently degrades to a default 1-node
Scion instead of failing. `dump_calc.lua` carries that binding.

Running the GUI on Windows: the executable is literally
`runtime\Path{space}of{space}Building.exe` (`{space}` is part of the
filename); launching from the repo runs from `.archive/src/` with dev mode on.
Syntax-check Lua with `luajit -e "assert(loadfile('<file>'))"`.

---

## 3. Repository map

| package | ports | lines |
|---|---|---|
| `modparser` | `ModParser.lua` + `ModTools.lua` — mod text → structured mods | 13.9k |
| `modstore` | `ModStore/ModDB/ModList.lua` — mod containers + tag evaluation | 2.1k |
| `calc` | `Calcs/CalcSetup/CalcPerform/CalcOffence/CalcDefence/CalcTriggers/CalcMirages/CalcActiveSkill/CalcTools.lua` | 21.5k |
| `data` | `Data.lua` + the `Data/` tables — the runtime game data | 9.2k |
| `data/schema` | the artifact document types (typed JSON) | 0.9k |
| `item` | `Item.lua` + `ItemTools.lua` — parse half + range machinery | 4.2k |
| `tree` | `PassiveTree.lua`, `PassiveSpec.lua`, timeless/abyss jewel generation | 3.9k |
| `skills` | `SkillsTab.lua` logic half | 0.7k |
| `build` | `Build.lua`'s load half + `ItemsTab.lua`'s slot table — build XML → `calc.BuildInput` | 0.5k |
| `config` | `ConfigTab.lua`'s load half + `ConfigOptions.lua`'s 580 options and 532 apply bodies | 3.4k |
| `export` | `src/Export/` — GGPK dat reader, stat-description engine, 21 script builders | 9.2k |
| `internal/util` | kept reference numeric/text semantics + `Opt[T]` | 0.2k |
| `internal/modcachegen` | regenerates `data/raw/modcache.jsonl` from the Go parser | 0.1k |
| `cmd/{pobexport,sourceupdate,treegen}` | CLIs | 0.3k |
| `test`, `test/luacanon`, `test/luapat`, `test/luarender` | differentials + all Lua-shape knowledge | 10.7k |

Import graph (production): `modparser → internal/util`; `modstore →
modparser`; `item → data, modparser`; `data → data/schema, modparser`;
`tree → data, item, modparser`; `skills → data, item`; `calc → data, item,
modparser, modstore, skills`; `config → data, internal/util, modparser, modstore`;
`build → calc, config, data, item, skills, tree`;
`export → data, data/schema, modparser`. **No production package imports
`test/…`.** `build` is the composition root: it is the only package that
imports `calc`, and nothing imports it but the tests.

Ownership rule: the package that produces or stores a value declares its type
and every consumer imports it as-is. `modparser` owns `Mod`/`Tag`/`Value`;
`modstore` owns `Output`; `data` owns `StatMapEntry`/`GrantedEffect`; `item`
owns `WeaponData`/`JewelData`/`GrantedSkill`; `export` owns the documents.
No contracts package.

---

## 4. The differential method

The whole project rests on one technique: **a Lua script boots the archive and
records its behaviour into JSONL; a Go test replays those records and fails on
any disagreement.** Plausibility arguments were explicitly rejected by the
user in favour of exact differential numbers.

### 4.1 Two axes

A module is finished only when both hold (`parity.md` scores every module):

- **code** — every branch of the reference exists in Go, nothing behind a "not
  ported" guard. The reference is the scope, *not* the corpus: if it is in
  `.archive` it must be written, bugs included.
- **archive** — a differential that fails on any disagreement passes at 100%
  over everything the corpus reaches.

Coverage is a property of the corpus, not a defect of the port: written but
unexercised (`code [x] / archive [~]`) is the normal state. Conflating the two
once let calc-offence sit "done" with 78 of 80 trigger configs unported.

**Panics are two different things and must not be conflated.** A panic
mirroring a reference error ("the Lua errors here too") is parity and counts
as written code. A panic standing in for unported behaviour is a code gap and
must be named in the module's row. Auditing panics for "unfinished work"
without this distinction removes error-parity behaviour.

### 4.2 The canonical encoding

`tools/canon.lua` and `test/luacanon` produce the same text from either side.
Comparison is byte equality first; when the text differs, `EqualWithin` walks
both parses leaf by leaf, numbers agreeing once quantized to
`luacanon.CompareDigits` significant figures and everything else exactly
(§4.7). Never structural re-serialisation.

- every table → JSON object, keys stringified and **sorted as strings** (so a
  10-element array emits `"1","10","2",…` on both sides)
- Lua's 1-based arrays → Go slice index `i` emits under key `i+1`
- whole numbers |v|<1e15 print via `%d`; other numbers via `%.17g`, so a
  double is emitted whole and nothing is discarded before comparison
- `canon.encode14` / `luacanon.Encode14` re-render at `%.14g` for the
  **hashed** subtrees only: a hash cannot absorb a last-digit difference, so
  both sides quantize to the reference's own data precision before hashing
- functions → `{"__fn":true}`; NaN/±inf → quoted `tostring`
- `canon.encode` reads values with `rawget`, so `__index` metamethods cannot
  materialise values the table does not hold

Go structs encode through a **`lua:"key"` tag protocol**: untagged fields are
skipped, `,omitempty` drops zero values, and `lua:"@array"` splices a slice in
as the enclosing table's array part. The tag, not the Go field name, is the
contract. Typed domain values reach the encoder through a global adapter
registry (`luacanon.RegisterAdapter`); a new typed shape without an adapter
encodes by raw reflection and diverges.

**One precision now.** Everything encodes at `%.17g` on both sides, so no
bit of a double is discarded before anything looks at it, and comparison
absorbs last-digit drift numerically (`luacanon.EqualWithin` quantizes to
`CompareDigits = 14`) rather than by truncating the text. `canon.encodeExact`
/ `luacanon.EncodeExact` remain as the explicit spelling for replay *input*.
The old split — compared values at `%.14g`, input at `%.17g` — caused a real
bug (§6.1) and existed only because hashed records could not tolerate a
last-digit difference.

Nothing is recorded as a hash any more. The game-data dump used to store its
13 largest subtrees as `{"k":…,"h":"<h1>.<h2>"}` (double `murmurHash2`, seeds
`0x9747b28c`/`0x2312233`), which kept the file at 4.2 MB but meant the dump
never held the reference's data — only a fingerprint of it. A mismatch could
report "differs" and never where, arrays inside those records could not be
compared as multisets, and both sides were forced to `%.14g`. All 136 records
now store canon text (37.9 MB). Recording the evidence beats saving the
bytes; if size ever forces the question again, compress the file rather than
digest the data.

The one sanctioned semantic normalisation of the reference side is
`luacanon.NormalizeArchiveMods`: the archive's parser stored numeric captures
as text (`"div":"5"`, `"value":"2"`) where the port parses them at parse time,
so a closed field set (div, limit, percent, base, threshold, thresholdPercent,
globalLimit, value) is rewritten to bare numbers before comparison. Every
fixture read routes through `forEachCalcRecord`, which applies it. Any other
normalisation of the reference side would be a way to hide a divergence.

### 4.3 Fixtures as data

The reusable dump design (built for mod-store, reused for calc): **the Lua
dump emits the world as fixtures** — stores, actors, multipliers, conditions,
items, configs — alongside expected results, and the Go test rebuilds that
world from the records. No fixture code is maintained on both sides.

- Fixture *values* are derived from the corpus, not hand-picked:
  `dump_modstore.lua` harvests the tag vocabulary the parsed corpus actually
  uses, sorts each set and assigns values by index modulo, so a newly parsed
  tag variable gets a fixture automatically.
- **Reference errors are part of the contract**: every query is wrapped in
  `pcall` and a failure records the sentinel `"!"`; the Go port must panic on
  exactly those inputs. `test/modstore_test.go` requires the two sides to line
  up in both directions.
- Where the corpus cannot reach a branch, the dump adds **synthetic** records
  (59 hand-built mods covering MonsterTag, BaseFlag, SkillPart, Limit, the
  Multiplier options, ItemCondition shapes, …). That list is where a future
  mod-store gap gets closed.
- Function-valued fields cannot be serialised: the dump emits a *fingerprint*
  (a jewel's `funcList` becomes its `type` strings) and the replay re-derives
  the functions from the same input, asserting against the fingerprint.

### 4.4 Guarantees the harnesses carry

- **Negative controls are mandatory.** Each module records a perturbation and
  its blast radius: +1e-7 on `AverageHit` fails all calc variants; +1e-9 on
  the simulated trigger rate fails exactly 7 checkpoints; `%.14g`→`%.15g` in
  the mod-cache quantization fails 1 item; +1 orbit radius fails 5 tree nodes;
  reversing cluster-notable sort fails 77; dropping tattoo overrides fails
  187; one gem level +1 fails 319 skill comparisons; +1 seed skew fails the
  jewel tables wholesale. A differential that cannot be made to fail proves
  nothing.
- **Work floors** stop a truncated fixture passing vacuously: calc ≥25
  variants, item ≥100 items, spec/skills ≥5 builds, tree ≥3,000 records, pairs
  calibration ≥250,000 sequences, export exactly 123 files.
- **Missing input is two cases.** Committed fixtures are mandatory —
  `t.Fatalf` with the exact regeneration command. Inputs derived from the
  submodule or the GGPK are optional — `t.Skipf`, so a checkout without them
  still runs green.
- A narrowed run that matches nothing fails explicitly (`MP_ONLY=%q matched no
  variants`).
- The game-data differential fails on any dumped subtree the port does not
  check, so coverage cannot silently shrink.

### 4.5 Staged checkpoints and the native bridge

The calc dump stubs out the tail of `calcs.perform` (`defence`,
`buildDefenceEstimations`, `triggers`, `mirages`→true, `offence`) and then
runs each stage explicitly, emitting `dbs`/`skills`/`skillLists`, then
`performDbs`/`performOutput`, `defence*`, `ehp*`, `globalCache`, `triggers*`,
`mirage*`, `offence*`, with parallel `*Minion*` records. This let a
1,600-line module land with its own checkpoint instead of waiting for 5,800
lines of offence.

Each build dumps three progressively stripped variants — `full`, `noskills`
(socket groups wiped), `treeonly` (items also wiped) — so a failure localises:
if `treeonly` agrees and `full` does not, the problem is items or skills.

The **native bridge** replaces fixture-fed inputs with natively built ones as
each upstream module lands. It now calls `build.Load` on the corpus build's
XML and substitutes everything package `build` assembles: spec, item pool,
slot table, item sets, skills tab, config and the header scalars. Nothing
in the calc differential's input comes from the dump any more. One test
therefore exercises six ports transitively. `MP_FIXTURE=1`
reverts to pure fixture replay — the switch that separates "native parser
bug" from "calc bug". Mods are deep-copied at the seam because the calc
stamps sources in place and the test process shares one cached tree.

The bridge keeps one test-side correction: `referenceOrderModList` permutes a
Glorious Vanity node's mod list into the reference's `pairs()` order.
Production merges timeless additions in first-seen order; the archive's order
is a LuaJIT hash walk, and the emulation of it stays test-side (§6.4).

### 4.6 Ordering, sorting, and never editing the referee

This is the mistake this port keeps making, in several disguises. It always
starts the same way: a comparison serialises a collection into an ordered
array and byte-compares it, the two sides disagree only on order, and the
repair goes into the wrong place.

**A comparison that fails only on order is a defect in the comparison.**
Ask whether the *behaviour* depends on order or only the *check* does.
Byte-comparing a serialised array makes every check order-sensitive
whatever the programs do. When only membership and counts carry meaning,
compare as a multiset and the question disappears. Reach for an imposed
order only after the multiset comparison has failed.

**Order changes a result only where the operation does not commute.**
Increased sums, more multiplies; both commute, so for most modifier work
the order is irrelevant to every number. The places worth suspecting are
winner-takes-one modifiers (an override, a max) and float summation showing
in the last digits. If order matters somewhere, demonstrate that case -
do not sort everything on the suspicion.

**Worked example, and the worst one so far.** `tools/dump_gamedata.lua`
sorted `clusterJewelInfoForNotable`'s `jewelTypes` arrays before recording
them, with a comment calling it "a documented deliberate divergence", and
`data/cluster.go:141` sorted the same list in PRODUCTION so the two would
agree. Every layer of that was wrong except the last: the reference builds
the list in Lua hash order and the dump had never once recorded it; the
"documented divergence" documented a convenience; and shipped code carried a
shape that existed to satisfy a harness. The repair is the shape every case
of this takes - the harness records what the reference does (55 notables
turn out to have a non-sorted order), the comparison stops caring about
order, and the production sort stays but is now labelled as what it is: a
fix for Go's randomised map iteration, not parity with the reference. If a
sort in production cannot say which of those two it is, that is the bug.

**Never install semantics into a dump to make a comparison pass.** The
archive is the referee for every judgement here: whether a behaviour is
load-bearing, whether a quirk must be reproduced, whether a module is done.
That only holds while the reference side is untouched. A dump script that
changes what the application does turns the differential from a test into a
mirror - it reports agreement on precisely the thing that was in question,
and once the dump is editable any claim can be made to come out either way.

**`tools/dump_calc.lua` runs under a sorted `pairs`. That is a fact about
that file, not a requirement on the port and not a convention to extend.**
It means the orders that dump recorded are sorted, so replaying them was
pointless. It does not mean the Go side must sort, and it does not license
a new dump to sort. Both of those have been assumed here and both are
wrong.

**Measured 2026-09-01, per site.** Every one of the 57 non-algorithmic
sorts was reversed INDIVIDUALLY and the full suite re-run, with the export,
timeless and tattoo differentials enabled. The experiment ran on a copy of
the tree in the scratchpad, so production was never edited. Run it that way
again: it needs no permission and cannot leave the repo dirty.

| category | count | effect of reversing it |
|---|---|---|
| **Algorithmic** - the order IS the computation | 18 | changes the answer, by construction |
| **Eligible for removal** - nothing any test observes | 34 | nothing at all |
| **Load-bearing** - a differential fails, for a reason | 20 | see below |
| **Order enforced, necessity unproven** | 3 | `later.md` section 3 |

The 20 that fail do so for demonstrable reasons: 9 `data/uniques_*` sorts
assign VARIANT NUMBERS (`"{variant:"+itoa(i+1)+"}"`), so reversing renames
which notable is variant 1 and saved builds reference variants by number;
8 export sorts set the byte order of generated `Data/*.lua` (78 files
differ); `modparser/modtools.go:153,272` join sorted names into formatted
mod text; `tree/cluster.go:367` moves computed outputs.

**Two methodological traps this run walked into, both worth remembering.**
First, a static reading of the consuming loop is weaker evidence than the
experiment: `calc/items.go:205` (a `visited` guard) and `calc/perform.go:349`
(a `+=`) were both called not-eligible from reading the code, and both are
in the eligible 34. Second, and worse - `TestExportAgainstReference` skips
unless `MP_EXPORT=1`, so 12 export sorts came back CLEAN from a suite that
never ran them. A CLEAN result from a skipped test is not evidence. Check
what actually ran (`go test ./... -v | grep SKIP`) before believing a
negative result.

The split is what makes the question answerable. An algorithmic sort ranks
or picks: `calc/performutil.go` does `sort.Float64s(stats)` then reads
`stats[0]` as `LowestAttribute`, so reversing it returns the highest -
`LowestAttribute: 167 vs archive 74`. Those 18 are load-bearing by
definition and were never the question.

A determinism sort only replaces Go's arbitrary map-iteration order with a
fixed one. Reversing all 34 eligible ones at once passes the entire suite.
Note what that does NOT license: reversal is still a fixed order, while
DELETION gives Go's randomised order, so a regenerated dump would differ run
to run even where nothing is wrong. Eligible means "the order is arbitrary",
not "the determinism is free".

The flask sort looked like a counter-example and is not one. `mergeBuff`
(`calc/performutil.go:69`, `CalcPerform.lua:44`) keeps the HIGHER of two
mods with the same parameters - the game's rule, strictly-greater on both
sides - so a bigger flask wins whatever the order. Order reaches only an
exact TIE, and a tie means the values are equal, so the sole difference is
which item's name lands in the winner's `source` string. Two "of the
Pangolin" flasks both granting `Armour INC 105` is exactly that. Nothing
reads that string as data: the calc's only decisions on `source` are
`!= "Base"` (`calc/defence.go:44,65`, `calc/offenceailments.go:121`),
`Contains("ElementalHit")` (`calc/offencecrit.go:302`) and the
`itemDisablers` key (`calc/items.go:178`, which walks item and tree mod
lists, never a buff list); every other use copies it onto a derived mod.
Two flask sources are indistinguishable to all of them.

So: outside the 18 that rank and the 1 that decides cluster notables,
sorting in this port buys determinism and nothing else. The determinism is
real - without a sort the credited item changes run to run - but it is a
property of the recorded dump, not of the program's answers. That does NOT mean
the 34 eligible ones can be deleted - reversal is still a deterministic
order, while deletion gives Go's randomised order.

The comparison side of that is done: `EqualWithin` detects a Lua array (an
object keyed "1".."n") and compares it as a multiset, so a reordering is no
longer a failure, and `SameCanon` is what the differentials call. Applying
it took the calc differential from **797 structural divergences to 10**
under blanket reversal. It is applied where order cannot reach a number and
deliberately NOT to mod-list sequences (see 4.6's boundary note and
`test/luacanon/equal_test.go`). Those 10 are not a residue to chase. They are the flask tie above,
and it is not an ordering difference at all - the two runs record different
TEXT (`Item:11: Dabbler's Sapphire Flask` vs `Item:14: Dabbler's Quicksilver
Flask`), so a multiset of the same elements cannot absorb them and should
not. Suppressing them would mean excluding `source` from comparison, which
is a decision about which fields are evidence, not about order. With the
sorts in place - the shipping state - the differential is clean.
`test/luacanon/equal_test.go` pins both directions: reorderings pass, a
changed element, a duplicate standing in for a distinct value, a differing
length and a keyed object with swapped values all still fail.

Not yet converted: the tree, build, game-data, config-option and mod-cache
differentials still compare strings positionally. One of them should stay
that way - `modtools` checks formatted mod TEXT, where term order is part
of the thing under test.

Cost, for the record: `sortedIntKeys` is 2.05% of a recalc in the CPU
profile (8.3ms cold, 7.7ms warm), most of it building the key slice rather
than `sort.Ints` at 0.68%. No other helper reaches the sampling floor.
Sorting can only cost, never gain - but computing one ordering per pass
instead of three would recover nearly the same time without touching
determinism.

### 4.7 Known blind spots

- **The precision floor is gone; what it was hiding is now visible and
  reported.** Compared canons are `%.17g` on both sides, so the dumps record
  every double whole and nothing is discarded before anything looks at it.
  When the text differs, `luacanon.EqualWithin` walks both parses leaf by
  leaf: numbers must agree once each is rendered to `CompareDigits` (14)
  significant figures, everything else exactly. That is the same question
  the old `%.14g` string comparison asked - "the same number, as precisely
  as the reference knows it" - but asked of the values rather than of their
  rendering, and it reports what it absorbed instead of hiding it. Fourteen
  is set by the reference, not chosen: PoB writes its data files and its
  ModCache as `%.14g` text and reads them back, so a number arriving that
  way carries 14 significant digits and no more. The calc differential
  reports **145 variants agree, 105 values only once quantized**. Negative
  control: a 1e-9 perturbation fails 10 checkpoints.
- **The drift has a named cause.** Go's `math.Pow` is not correctly rounded
  where the C library `pow` behind LuaJIT's `^` is. For x=0.65 LuaJIT's
  `x^3` is 0.27462500000000001; `x*x*x` and `math.Pow(x,3)` are both
  0.27462500000000006. It reaches `EffectiveSpellBlockChance`, the
  suppression family, the PvP damage chain and the ignite chances, across 19
  `math.Pow` sites in calc. `math/big` closes the integer-exponent half
  exactly (verified against the reference); the fractional half - evade
  chance's `^0.9`, ignite stacks, hit rate, the PvP exponents - has no
  `math/big` answer, since it offers no power for a real exponent.
- **The number models are identical, so a 17-digit comparison is
  meaningful.** Established 2026-09-01: the dumps run under LuaJIT 2.1 x64,
  which is SSE2 - no x87 80-bit intermediates. `(1e16 + 1) - 1e16` is 0 on
  both sides, and a battery of six cases agrees digit for digit at 17
  between LuaJIT's interpreter, its JIT traces, `jit.off()`, and Go. A
  failure at digits 15-17 is therefore a real difference in what was
  computed, never a difference in how numbers are represented.
- **Three tooling defects found while measuring, all fixed.**
  `canon.encodeExact` restored `floatFormat` to a *hardcoded* `"%.14g"`
  instead of to its previous value, so every compared record emitted after
  the first fixture was silently forced back to 14 - the floor could not be
  raised at all until that was repaired, and the first attempt to raise it
  looked like a clean pass. And `authored_triggers4.xml` had no line in
  `test/corpus/manifest.tsv`, so `calc_trig4.jsonl` could not be regenerated
  by the documented command and went stale. Third: `dump_gamedata.lua`
  listed two subtrees as `modscalability` and `flavourtext` where the
  reference's table spells them `modScalability` and `flavourText`, so
  `data[key]` read nil and both dumped as `null` - the port had checks for
  them under the right names all along, and they had never once been
  compared. All three share a shape: a check that looked like it was
  running and was not.
- **Constant folding: swept, clean.** Go evaluates an expression made only
  of untyped constants exactly and rounds once at the end; the reference
  rounds each literal to a double first and rounds again at every
  operation. Where a literal is not exactly representable the two part
  company - `1 / 0.033` folds to 30.303030303030305 where the reference
  computes 30.303030303030301. The fix at a site is to give the literal a
  type (`const d float64 = 0.033`), which rounds it before the operation.
  `TestNoUnsafeConstantFolding` replaces the old seven-candidate regex: it
  evaluates every fractional constant expression in production source both
  ways, using `go/constant` - the same exact arithmetic the compiler uses -
  and resolves untyped named constants as well as literals. **35
  expressions examined, 0 unsafe.** Both paths carry a positive control
  (an injected `1 / 0.033`, and one behind a named constant, are each
  caught), and the test fails outright if it examines nothing, because a
  sweep that reaches no source passes for the wrong reason.
- **The `math.Pow` divergence is settled, not pending.** It is a known,
  quantified, tolerated difference: 105 values across the corpus, all at the
  16th digit, absorbed because the comparison asks for agreement to 14
  significant figures - which is all the precision the reference itself
  carries. Nothing needs fixing for the differential to hold, and the count
  is printed on every run, so the day it grows past 14 digits the run fails.
  Writing a correctly-rounded power would only close the integer-exponent
  half anyway; the fractional sites (evade chance, ignite stacks, hit rate,
  the PvP exponents) have no clean answer in the standard library.
- **A trap the folding sweep does NOT cover:** Go's `1/3` over untyped
  INTEGER constants is 0, where Lua gives 0.333 - a semantic difference,
  not a rounding one. Telling a deliberate integer division from an
  accidental one needs type context the sweep does not carry.
- **Shared-path bugs are invisible.** A defect in code both sides pass through
  compares equal to itself. `quantizeTag` once dropped a tag and agreed with
  itself; the mitigation is to make shared normalisers *loud* (panic) rather
  than lossy.
- **A partial canon is a blind canon.** `data.ModCanon` drove the whole calc
  port while omitting `mod.replaced`/`mod.converted`; a divergence there would
  have compared equal at every stage. Audit what a canon emits, not just that
  it passes.
- **The reference is non-deterministic in known places** — see §6.2. Those are
  recorded as accepted answer sets, not chased.

---

## 5. The porting recipe

1. **Mechanically transform pure-data tables** Lua→Go, converting Lua patterns
   to Go regex **once, at the source**. Hand-port closures containing
   statements, citing reference line numbers.
2. **Write the Lua dump** using `tools/canon.lua`; make the Go test fail on any
   diff. Prove the dump reproducible: dump the same source **twice in two
   processes and byte-compare the files** — re-running the Go test cannot see a
   dump that is stable-but-wrong.
3. **Land at 100%**, with a negative control.

Closing a guarded (unported) branch moves both axes at once — never port blind
and hope a build turns up:

1. get an environment that reaches it — a real ladder character (`mb search`
   in the sibling tool `E:/tools/missingBuild`, not part of this repo) or a
   hand-authored throwaway build (`test/corpus/authored_*.xml`, 10 so far);
2. dump it (`luajit ../../tools/dump_calc.lua <key> <xml>`), add it to the
   variant map **and to `test/corpus/manifest.tsv`** — a dump key absent from
   the manifest silently skips the item, spec and skills differentials;
3. write the branch, delete the guard, confirm byte-identical.

This grew the corpus 9 → 12 → 25 → 34 → 38 → 42 → 48 builds.

**Settling "is this dead / is this wrong on real data":** instrument the site
to compute both candidate answers, run the full corpus, count divergences,
then revert the instrumentation completely. Zero divergences downgrades an item
to readability; non-zero promotes it to a bug. Applied to `RoundHalfUp`: 93,855
calls, 0 divergences — which means "no measured difference", not "safe to
replace".

**Multi-item change sequencing:** items with no shared type or signature go in
one wave and run in parallel; anything changing a type that crosses package
boundaries runs alone. Every wave ends with build + full suite; waves touching
`export/`, `data/schema` or the modparser codec also run the export
differential.

---

## 6. Traps

### 6.1 Floating point and determinism

- **Go folds untyped constant expressions at arbitrary precision.** `1 / 0.033`
  as a Go constant is `30.303030303030305`; Lua's runtime division is
  `30.303030303030301`. That fed `ceil(x * ServerTickRate)` and moved a skill
  duration a whole server tick. Fix: force typed-constant rounding —
  `1 / float64(0.033)` (`data/tables.go`). Only expressions whose operands are
  not exactly representable can differ. The sweep for other instances was a
  regex over `data/` and `calc/` checking seven candidates — explicitly *not*
  a proof.
- **Replay input serialised at `%.14g` is lossy and amplifies.** A cached
  source rate dumped as `10.063177748344` versus the true
  `10.063177748344373` — same 14 digits, opposite sides of a simulation loop's
  `<` bound — made a trigger simulation count 1000 instead of 1001 and the
  trigger rate came out 0.1% low. Hence `%.17g` for input, `%.14g` for
  comparison.
- **Float addition is not associative.** The tree generator computes an offset
  then adds it, because the upstream Python computes the difference first;
  sums over numeric-keyed tables must walk the reference's key order
  (`sortedNumKeys`). Rewriting `x + (t - s)` as `(x - s) + t` breaks byte
  agreement.
- **`RoundHalfUp` ≠ `math.Round(v*p)/p` in three independent classes**:
  negative ties break toward +∞ (−2.5→−2); `0.49999999999999994`→1 because the
  *addition* rounds to exactly 1.0 before `floor` sees it; at 2⁵²+1 adding 0.5
  lands on the next representable double. It cannot be re-expressed as
  "`math.Round` plus a negative-tie fix".
- **`Common.lua floor(val, dec)` adds 0.0001 before flooring** to stop
  accumulated error dropping a value a whole unit (a MORE product of 1.331
  lands at 1.3309999999999997). The same epsilon appears in the evaluator's
  per-multiplier division. Adding or removing it changes results.
- Every `dec=10` use of `RoundHalfUp` is the inner half of a nested
  `RoundHalfUp(RoundHalfUp(x, 10), 2)` — six area-of-effect/speed sites. Why is
  unexplained; collapsing it is not safe.

### 6.2 Iteration order

- **LuaJIT `pairs()` over integer-keyed tables is hash-slot order**, not
  insertion order. Emulated exactly test-side in `test/luapairs_test.go`
  (hash rotation on the double's bits, `lj_fls(k-1)` array bins, main-position
  eviction, freetop scan, rehash reinsertion), calibrated against the real
  interpreter over **252,060 sequences** — every insertion sequence of length
  1–3 over keys 0..37 plus 200,000 random length-4 sequences. The emulation is
  only trustworthy inside that domain.
- **By decision (2026-08-27) that emulation stays out of production.** Glorious
  Vanity's addition merge runs in first-seen order and records the blocks
  (`SpecNode.TimelessAdditions`); the differential permutes into reference
  order before comparing (disabling it fails 4 nodes). The difference was
  judged display-only. **That judgement is now testable and untested**
  (2026-09-01): reordering a sum moves it around the 16th digit, which the
  comparison can see since it went to full precision, so whether this order
  reaches a computed output can be answered instead of assumed. Failing a
  differential does not answer it - a positional comparison fails on any
  reordering whatever the numbers do (4.6). Compare the computed outputs.
- **A last-writer-wins assignment inside a `pairs()` loop over a shared table
  is process-random.** `Data.lua:1039` stamps
  `grantedEffect.statMap._grantedEffect` while iterating `data.skills`, and two
  skills can share one statMap table (`ExplosiveTrapAltX` aliases
  `ExplosiveTrap`'s). Settled by re-assigning in sorted skill-id order on both
  sides; modelled in Go as `GrantedEffect.StatMapOwner`. **General rule: any
  last-writer-wins assignment over `pairs` needs a sorted normalisation applied
  identically on both sides.**
- **Sets keyed by tables defeat sorting by key type.** `env.flasks`/
  `env.tinctures` are keyed by item tables, so `pairs()` walked memory
  addresses — random per process. `sortedPairs` orders them by the key's `id`
  when every key has one, else by the group's position in the socket-group
  list.
- **Before replaying anything a dump recorded, check whether the dump derived
  it.** `dump_calc.lua` installs `pairs = sortedPairs` *before* the Calc
  modules load (they localise `pairs` at load time), so every recorded node
  order is simply ascending ids. The belief that these were captured LuaJIT
  hash order survived two remodel stages and cost a whole replay machinery,
  deleted 2026-08-31 in favour of `sortedIntKeys`. The test now asserts every
  recorded order is ascending.
- **Genuine LuaJIT order does leak into shipped data** in a few places where
  sorting is not an option: `tradeHashes` entry order in the mods export, and
  `LegionPassives.lua`'s per-node `oidx` (drawn from an unseeded `math.random`
  stream — it is layout noise, not data). Both are replayed by test-side
  replicas (`test/luarender/luatab.go`, `luaprng.go`) that die with the archive.
- **The reference is genuinely non-deterministic in two measured places** —
  two cluster-jewel sizes sharing one enchant text (three runs gave both
  answers; the port keeps the lexicographically greatest and the test carries
  an allow-list), and an abyss node where two stats share identical
  `(min-max)` text (four dump processes gave rolls 2,3,2,2; the port's
  dat-order pick is one of the reference's own outcomes).
- **Mutating a Go map while ranging over it** does not reproduce a Lua loop
  that reads one table and writes another — `buildResourceTypes` collects the
  "maximum X" keys into a separate map first, or the port produces
  "maximum maximum life".

### 6.3 Lua value semantics

- **Only `nil` and `false` are falsy — `0` and `""` are true.** Skill-data `or`
  chains treat a present zero as a winning value; "no reservation" writes
  explicit zeros, so `== 0` as a stand-in for absence is wrong.
- **Absent / present-false / present-zero are three distinct states.**
  `ModStore:GetStat` is `output[stat] or cfg.skillStats[stat] or 0`, so a
  stored `false` falls through and a stored `0` does not. Modelled with an
  explicit kind tag (`modstore.OutValue`), and assigning an absent value
  *deletes* the key. `Flag()` returning nil means `output.X = Flag(...)` stores
  no key at all when false — every divergence in the first defence run was
  this one rule.
- **`nil ~= 0` is TRUE.** A guard meant as "is this stat non-zero" also passes
  when the stat was never accumulated (`getPerStat`'s radius case emits a
  zero-valued mod where a Go zero-value map emits nothing).
- **`nil` vs empty is fixture-visible.** In the item model modList/baseList/
  grantedSkills/sockets/otherLines are always tables; `buffModList` is `{}`
  only for flask and tincture bases. A Lua table cannot hold nil, so nil is an
  absent key, not a present empty one.
- **Join artifact:** `"" .. table.concat(lines)` with zero lines loads as
  `{ "" }`, not `{}` (tradeHashes, flask buffs, cluster stats).
- **Duplicate table keys: LuaJIT keeps the FIRST.** `Data.lua`'s misc table
  defines `EnergyShieldRechargeBase` twice and the derived `1/3` survives, not
  the literal `0.33`.
- **Lua coerces numeric strings in arithmetic but not in comparisons** (`2/"3"`
  works, `1<"4"` errors). Seven parser closures passed raw captures where
  numbers were expected; a later census found five more. All now parse at parse
  time and every string arm is deleted from the numeric readers.
- **Reading an undeclared global yields nil**, and PoB does it in live paths:
  `modSource`, `dotCfg`, a dead `cfg`, `nullValue` (so failed LIST evaluations
  drop silently), `ItemClasses`.
- **`or` binds looser than arithmetic.** `Sum(...) or 0 + X + Y` parses as
  `Sum(...) or (0+X+Y)`, so TripleDamageChance drops its enemy and on-crit
  terms; `(Flag and 100 or 0) + Sum(Avoid)` parses as
  `Flag and 100 or (0+Sum)`. These read as ordinary arithmetic in the Lua.
- **`ModDB:OverrideInternal` falls off the end and returns ZERO values**, not
  nil — `m_min(min, modDB:Override(...))` becomes `m_min(min)`. Go has no
  zero-value return; each site must be hand-translated.
- **Tag arrays carry holes.** A nil among a mod's tags leaves a numbering gap
  (key `"2"` with no `"1"`); trailing nils vanish. `len(Tags)` is `#mod`;
  `ModTags()` is the `ipairs` length that stops at the first hole. Both are
  used.
- **`tostring(nil)` is `"nil"`** — flasks and grafts genuinely get the slot
  name `"Flask nil"`, which lands in mod sources the differentials compare.

### 6.4 Aliasing and shared mutation

The reference mutates shared tables constantly, and the port reproduces most
of it deliberately:

- **Tags are pointer structs on purpose.** The evaluator writes a computed
  `div` (from `divVar`) back into the shared tag, so every later evaluation of
  that mod sees it. Value-type tags would silently discard cross-evaluation
  state; evaluation is not pure.
- **`ModList:MergeMod` copies shallowly**, so the merged copy shares tag tables
  — and their mutations — with the original. Deep-copying "for safety" changes
  results.
- **`mergeKeystones` stamps the tree's own keystone mod list in place** when
  the granter's source is not tree-flavoured (the reference's local is named
  `fromTree` and is true when it is *not*). This is why the tree dump must come
  from a freshly built tree, before any calc runs.
- **Where the reference memoizes into shared game data at calc time**, the port
  uses per-environment overlays instead so loaded data stays pristine and still
  matches the game-data canon: `statMapOverlay`, `globalEffectOverlay`,
  `TriggeredCostWipes`.
- **Two cross-build contaminations were fixed against the reference** (both
  commented at the site): spectre source stamping wrote into mods aliased into
  `data.Skills`, and the DOUBLED mod form wrote a name into the shared pattern
  entry. Both now stamp clones.
- **Spec nodes initially share their source node's stat-line slice**; `cloneSd`
  and clearing the source pointer after an in-place rewrite are what stop a
  conquering addition reaching back into the shared tree.
- **Cluster rebuilds replace node objects wholesale**, so `Linked` lists hold
  stale pointers; `buildPathFromNode` normalises each link to the canonical
  object as it walks.

### 6.5 Caching and identity

- **PoB preloads `Data/ModCache.lua` (13,173 lines) into the parser's cache**
  at startup (`Main.lua:118-126`), so for those lines it never parses — and
  those numbers round-tripped through `%.14g` text when the file was written.
  Live PoB modifier values are therefore *quantized*: `1/3` is
  `0.33333333333332998`, not the IEEE third. The port ships the same entries
  (`data/raw/modcache.jsonl`, 13,173 records) and serves them (~µs decode vs
  ~450µs parse). Parser-level differentials pin fresh mode
  (`modparser.SetModCache(nil)`) because their dumps wipe the cache;
  `loadData` re-arms it. Full precision later is a one-call switch.
- **The parse cache deep-copies on every return**, because callers routinely
  mutate what they are handed (`SetSource` is the common case). Returning
  cached objects directly corrupts the cache.
- **Cache entries store the environment and skill themselves, not snapshots**,
  so every read through a `GlobalCache` entry is LIVE — a later stage changes
  what an already-cached entry appears to hold. Snapshotting diverges.
- **The cache map must be materialised in the requesting environment before any
  nested perform**, or nested `cacheData` writes into its own lazily created
  map, siblings re-miss, and a skill picks up a stage multiplier it should not
  have.
- **The recursion breaker is a single check** at the top of the trigger stage:
  skip when the main skill's uuid is in `env.limitedSkills`. Manaforged Arrows
  builds its own skill to learn its mana cost; without the guard the port
  allocated an environment per level and consumed ~100 GB. `BuildActiveSkill`
  carries a defensive depth-20 panic with no reference counterpart.
- **Dat rows have stable identity** (same id → same object, so ref cells
  compare by pointer) and cells cache lazily; `SetCell` mutates the shared
  cache, so **export scripts are not independent** — running a subset can
  produce different output than a full run.

### 6.6 Text and encoding

- **UTF-16→UTF-8 keeps two reference quirks**: surrogate-range or >0x10FFFF
  code points emit a literal `?`, and a pending high surrogate survives
  intervening non-surrogate units (so it can pair with a later low surrogate).
- **32-bit floats are decoded by hand**: denormals collapse to signed zero, and
  the all-ones exponent decodes as an enormous finite number.
  `math.Float32frombits` would be correct IEEE and wrong for parity.
- **`FoldText`** (the reference's `sanitiseText`) strips balanced `<…>` spans,
  folds the Unicode hyphen family and a/o-umlauts (UTF-8 *and* cp1252
  spellings), then replaces any remaining byte >127 with `?` — but only runs at
  all when the string contains a byte >127 or a `<`. Hazard: `\xc4`/`\xd6` are
  also UTF-8 lead bytes, so a genuine multi-byte sequence starting with one is
  mangled. The port folds uppercase accents where the reference folds only
  lowercase — a deliberate extension, free on today's corpus.
- **Captures come back lowercased** because the reference matched against a
  lowercased copy of the line; the port matches case-insensitively against the
  original and lowercases captures explicitly.
- **`.gitattributes` sets `* text=auto eol=lf`.** A tool writing CRLF into
  `data/raw` fails the byte-comparison guards for reasons unrelated to content.
- **Generated Lua data does not load as its bytes**: text passes through Lua
  string literals so `\n` must be unescaped — but `[[long brackets]]` are RAW
  and must never be, and `[[` followed by a newline skips only that first
  newline.

### 6.7 Harness traps

- One process per corpus source: reloading a build in-process leaves stale
  state.
- `arg` is clobbered by the HeadlessWrapper boot — read `arg[1]`/`arg[2]`
  before `dofile`.
- ModList/ModDB objects carry actor/parent backrefs — canon the array part only
  or the encoder recurses forever.
- `GlobalCache` is a global and the progressive strip mutates one loaded build,
  so each variant calls `wipeGlobalCache()` and temporarily restores the real
  stage functions for the cache fill, re-stubbing immediately.
- The archive leaves perform residue on shared skill tables
  (`warcryBuff[1].warcryPowerBonus`); the dump scrubs it before and after each
  variant, or variant N's dump depends on variant N−1 having run.
- `dump_calc.lua` replaces `mergeSkillInstanceMods` wholesale with a
  sorted-stats replica, because the original iterates `pairs(stats)` over
  string keys (randomised per process). That replica must be re-synced by hand
  if the archive's body changes.
- `dump_modtables.lua` reaches module-local tables by reading `ModParser.lua`
  as text, rewriting its trailing `return` with a gsub, and `loadstring`ing the
  result — a reusable way to observe module-locals without editing the
  read-only submodule.

### 6.8 Data-format traps

- **A `.datc64` table** is a 4-byte little-endian row count, fixed-width rows,
  an 8-byte `0xBB` marker, then variable data. Column widths: Bool 1, UInt16 2,
  Int/UInt/Float/Enum 4, Interval 8, String 8, ShortKey 8, Key 16, and every
  list cell is 16 regardless of element type. Strings are UTF-16LE at an offset
  into the variable section.
- **Null references** are `0xFEFEFEFE` / `0xFEFEFEFEFEFEFEFE` and decode to a
  nil cell. A nil `*Row` boxed into `any` is *not* a nil interface, so
  "find rows whose ref column is empty" needs an explicit nil predicate or it
  silently returns nothing.
- **Enum columns resolve conditionally**: if the referenced table differs from
  the one being read, the cell decodes to a raw `int64`; only a self-reference
  yields a row; an unloaded target yields nil. So the correct typed accessor
  for an Enum column is not obvious and the wrong one panics.
- **Out-of-range reads return the reference exporter's sentinels**, not errors:
  `-1337` (Int/Float), `1337` (UInt/UInt16/Interval/ref raw), `"<no offset>"`,
  `"<bad offset>"`. Seeing 1337 in an artifact means a truncated file or a
  wrong schema, not a game value.
- **List cells truncate at 1000 elements**, matching the reference.
- **Nothing validates a column schema against the file**: row width comes from
  the file's own header arithmetic and is never cross-checked against the
  summed column widths. A schema wrong by one column reads plausible garbage
  from every later column; only the 123-file export differential catches it.
- **`AlternatePassiveAdditions` names two columns `SpawnWeight`.** The real one
  is the FIRST and must be read by position; the port calls the second
  `SpawnWeight2`. A sibling column's name has a trailing space
  (`"NotableReplacementSpawnWeight "`).
- **The extracted GGPK dirs are lowercase** (`data/`, `metadata/`) while the
  loader joins `"Data"` — this works only on a case-insensitive filesystem.

---

## 7. Lua-isms: the three purges

Three explicit stages removed Lua-shaped Go from the product, each planned in
its own document and executed with the whole suite green at every boundary:
`deprecated/go-remodel-plan.md` (15 steps, 2026-08-29),
`deprecated/lua-gtfo.md` (7 waves, 2026-08-30) and
`deprecated/lua-residue.md` (4 tiers, 2026-08-31). They are finished work;
what they still carry is the reasoning, the precedents and — in
lua-residue.md — an opportunistic list nobody has scheduled.

**Adjudication vocabulary**, reused across all three — use it instead of a
binary "is this ugly" judgement:

| verdict | meaning |
|---|---|
| **infection** | a Lua-ism with no behavioural justification → replace with typed Go |
| **load-bearing** | encodes reference behaviour a differential observes → keep the behaviour, re-express behind a typed API |
| **sanctioned** | test-side Lua understanding at the archive boundary |
| **generated** | fix the generator or the type design of what it emits |

**What landed:** `modparser.Value` and `Tag` are sealed sums of typed kinds;
`Mod.Type/Flags/KeywordFlags` are named types; `modstore.Output` is a typed
three-state map; calc's bags became structs; `internal/util` holds the kept
reference semantics; `map[string]any` survives in exactly two production files,
both genuine JSON edges (`modparser/modcache.go`, `export/treegen.go`).

**The standing rules are in the repo's `CLAUDE.md`.** In one line: *port
behaviour, never shape.*

**Meta-lessons the third stage recorded** (the reason it found what two passes
missed):

- A verdict in a document is a claim, not a fact — including one you wrote.
  Check it against evidence *outside* production code; production comments only
  confirm themselves. Every verdict should carry its evidence pointer.
- Citations decay fast here. Both `lua-gtfo.md` and `lua-residue.md` instruct
  readers to treat every line number as a hint and locate code by name.
  `later.md`'s `#EVAL` line numbers have already drifted (the *set* of sites is
  still exact).
- A green suite proves the corpus only. Eight correctness fixes in the third
  stage were reference divergences no fixture reaches, found by reading the
  reference beside the port; the suite's role there was to prove the fix inert.
- "Nothing reads it" does not license deleting a field — the fixture echo
  compares the whole projection. The refined test is whether the *emitted key
  set* is empty, which needs a producer census plus a fixture scan.
- Two proposals were withdrawn on evidence: an actor-name enum (the fixtures
  carry a fourth value, `"nonexistent"`, whose text must round-trip) and a
  weapon-data retype (instrumentation measured zero divergences).

**Kept by decision, inventoried in `later.md`** — do not "clean up": Lua
truthiness (`modparser.Truthy`, `OutValue.Truthy`), `%.14g` number spelling
(`FormatG14`/`FormatIntOrG14`, product cache keys depend on it), `Quantize14`,
`RoundHalfUp`, `Tonumber`, `FoldText`, `StripBalanced`, `gsubLimitFunc`,
`literalWeight`, ~392 `lua:"…"` struct tags, the `data` package's 70+ globals.
Commit `c456df051`'s claim that no such helper survives is literally false;
`later.md` §1 is the authoritative statement.

**`#EVAL`** marks behaviour that exists only to match the archive — reproduced
bugs, undefined globals read as nil, precedence accidents, hash-order
artifacts, LuaJIT internals. `grep -rn '#EVAL'` is the live list (57 sites in
Go source, 34 prose references *as of 2026-08-31*). Each is a candidate to
delete once the archive stops being the contract.

---

## 8. PoB internals worth knowing

### 8.1 Modifier parsing

`ModParser.lua` turns one line of English mod text into structured mods by
scanning a fixed table sequence: jewel radius functions → whole-line cluster
lookups → the unsupported list → `specialModList` → `preFlagList` → leading
skill name → `formList` → up to two `modTagList` tags → `modNameList` →
trailing skill name → `modFlagList`. Each scan cuts its span out of the line;
the remainder is returned as `extra`.

- **Two passes per line.** Pass 1 looks for the skill name *after* the mod
  name; if it produced mods but left a remainder, the whole line is re-parsed
  as pass 2 (skill name *before*), and only pass 2's result is kept.
- **`scan` ranking:** earliest match wins, then longest, then — on an exact
  span tie — pattern specificity. The reference used Lua pattern length; the
  port cannot (regex class syntax inflates lengths and inverts the order), so
  `literalWeight` counts pattern text **outside capture groups**. A port
  invention, load-bearing, and the one place the matching rule deliberately
  differs.
- `specialModList` assembly order matters: literal and keystone entries are
  collected and anchored `^…$` first, and only afterwards does the per-gem loop
  add its own partially-anchored entries.
- All patterns ship as **ordinary Go regex** (backtick literals), compiled
  once. `test/luapat` converts Lua patterns to regex **test-side only**; a
  regression once called it from calc and would have pinned a transition tool
  into the shipped engine. `TestItemTagPatternsAreLiteral` guards the shipped
  item-tag tables against interior metacharacters.
- Mod types: BASE, INC, MORE, FLAG, OVERRIDE, LIST, MAX, MIN, plus CHANCE,
  DUMMY and a mixed-case `"Flag"` that is **not** `"FLAG"` — an upstream typo
  that is self-consistent (the calc queries with the same string), so
  normalising it would make item-disabling mods unreachable.
- PoB's own canonical mod text (`formatMod`/`formatTag`/`formatValue`) is
  consumed by the product itself — flask cache keys, tree node `modKey` — so
  its spelling is frozen by two forces at once. Where the reference prints
  `function: <address>` the port substitutes a stable id built from the table
  key plus captures, and the tree differential normalises both spellings.

### 8.2 The modifier store

Three types: `ModStore` (abstract base: parent link, actor, direct multiplier
and condition tables), `ModDB` (mods bucketed by name; most queries run
against it), `ModList` (flat slice, supports `MergeMod`). Both embed the base
and chain to a parent; every aggregation recurses and combines.

- **Tag evaluation** rejects a mod or scales its value. Actor resolution:
  `"player"` falls back through this actor's player → parent's player →
  enemy's player; any other name is a plain lookup; an unresolvable actor
  named by a Multiplier tag rejects the mod entirely.
- **Global limits** (cap the total contribution of a mod group) are applied in
  a second loop and only when the aggregation passed a limits table — only
  `ModDB`'s Sum, More and Tabulate create one. In Flag/Override/List a global
  limit silently does nothing.
- **MORE precision:** mods in the high-precision table (leech/regen 2dp, crit
  2dp, support mana multiplier 4dp) make the per-name product *floor* at that
  precision; otherwise it rounds to 2dp.
- **Asymmetries that look like bugs and are load-bearing:** `Max` uses
  `val > (max or 0)` so an all-negative candidate set registers nothing;
  `HasMod` exists only on `ModDB` so calling it through a `ModList` errors;
  only `ModDB.Sum` guards source-less mods, so every other aggregation errors
  on them under a source filter.

### 8.3 The calculation

Pipeline: `initEnv` → `buildActiveSkillModList` → `calcs.perform` →
(defence → EHP → triggers → mirages → offence) → `cacheData`, then the same
four stages for the minion. `PerformFull` reproduces exactly this order; stage
order is load-bearing because later stages read what earlier ones wrote.

- **`initEnv` is a restart loop**, not one pass: finding an enabled Energy
  Blade gem restarts the whole build with `AffectedByEnergyBlade` appended to
  the override conditions. Anything counting work during construction must
  expect it to run twice.
- Within one pass: character constants → bandit/pantheon → enemy DB →
  config/party mod lists → override conditions → tree nodes → items →
  `mergeDB` → granted passives and ascendancy nodes → second tree merge →
  tree-granted skills → skills/supports.
- **Keystones merge into `env.initialNodeModDB` at setup**, and resolved
  keystone mods reach the main DB only in perform, which first resets the
  added-keystone set.
- **Actor model:** three parallel `performActor` bundles (player, enemy, and
  the main skill's minion — at most one, `env.minion =
  env.player.mainSkill.minion`), each with its DB, output, main skill and skill
  list, plus `mainHand`/`offHand` output tables that stay nil until an attack's
  offence stage fills them.
- **`ActiveSkill`** = one active `ActiveEffect` (granted effect, gem data,
  level, quality) + support effects + socket group + actor + open `SkillData` +
  skill-type set + string skill flags + its mod list and query configs.
  Support compatibility is `canGrantedEffectSupportActiveSkill`; duplicate
  supports resolve by level then quality, and a "plus version of" supersedes
  its base.
- **`GlobalCache.cachedData[mode][uuid]`** is written as the very last
  statement of every perform. The uuid is
  `<name sans spaces>_<UPPER SLOT sans spaces or NO_SLOT>_<gem index>_<group
  index>`, and the gem index compares gem-instance *identity*, not name. The
  port keeps only the MAIN bucket.
- **Triggers** dispatch through a table keyed by five strings tried in order
  (granted-effect id, lowercased skill name, lowercased trigger name, that name
  minus an `"awakened "` prefix, lowercased granting unique's name). All 81
  reference keys are present so lookup order is faithful; an unported one is an
  explicit `nil` that panics on match.
  `calcMultiSpellRotationImpact` simulates 1000 attacks — which is why sub-digit
  differences in the source rate shift the result ~0.1%.
- **Mirages** cover five cases and *report whether they took over*: Mirage
  Archer and Sacred Wisps run alongside the player's own offence; The
  Saviour's Reflection and Tawhoa's Chosen replace the main skill outright;
  General's Cry rewrites it in place.
- **Skill callbacks:** the data declares which callback kinds a granted effect
  has; the bodies live in `calc/skillfuncs.go` keyed (effect id, kind).
  `runSkillFunc` panics on anything unported — the most common reason a new
  corpus build fails to run, and the fix is always one registry entry.
- CALCULATOR mode differs from MAIN only in skipping UI write-backs.

### 8.4 Items

`ParseRaw` sorts every line into six ordered lists (class-requirement, enchant,
scourge, implicit, explicit, crucible — plus buff lines for flask/tincture
bases); `BuildModList` turns those into the flat mod lists the calc consumes.

- Two parse modes: GAME (a `Rarity:` line or an explicit rarity) runs a state
  machine over `--------` separators to find where implicits stop; WIKI is
  PoB's own saved format.
- Line annotations are curly-brace directives stripped before parsing —
  `{variant:…}`, `{version:N}`, `{group:N}`, `{tags:…}`, `{range:0.5}`,
  `{fractured}` — plus 17 boolean flags; in-game text uses ` (enchant)`
  suffixes instead. `enchant` implies `crafted`+`implicit`.
- **`calcLocal`** separates local item mods from character mods by *removing*
  matching mods from the list as it sums or multiplies them; weapon rates,
  armour defences, flask durations are chains of these calls. Ordering matters.
- Weapons and Rings build a **mod list per slot number** (1, 2, plus 3 for
  rings); everything else builds one with slot nil. `{SlotName}`, `{Hand}`,
  `{OtherSlotNum}` are substituted per slot.
- **`applyRange`** resolves `(min-max)` shells through `data.ModScalability`
  (searching substitution combinations largest-first) with per-entry precision
  handlers, falling back to `HighPrecisionMods` parsing only when no
  scalability key matches.
- Armour stores both a value and a `BasePercentile`; when text supplies a value
  but no percentile, the percentile is backed out and the value recomputed.
- **An item authored by affix identifier contributes nothing until `Craft()`
  runs.** `BuildRaw`/`Craft` were ported because the mod-cache generator needs
  them, not for a UI.

### 8.5 Passive tree

`tree.Load` decodes the document, migrates classes 1-based→0-based, drops the
root node and legacy alternate ascendancies, classifies nodes through a fixed
chain (class start → ascendancy start → mastery → socket → keystone → notable →
normal), builds the name maps, computes orbital positions, derives linked ids,
precomputes per-socket radius sets and stamps `ConnectedTo<Class>Start`.

- **`ProcessStats`** splits a newline-containing stat line *in place* into
  several lines, and retries an unparsed line combined with one, two, three …
  following lines, leaving consumed lines as empty placeholders. Both change
  the index alignment between stat lines and per-line mods.
- **A spec shadows the tree**: in Lua via `__index`, in Go via explicit copies
  reset by `replaceNode`. Fields the source does not set fall through — most
  importantly `name`, which conquered and tattoo nodes lack. Reading a
  shadowable field directly instead of through `EffectiveName()` wrongly pruned
  four Impossible Escape nodes.
- **`BuildAllDependsAndPaths` is a four-pass rebuild** after every allocation
  change: (1) reset + radius-jewel flags; (2) tattoo overrides then conquering;
  (3) mastery effects and recounts; (4) dependencies, prune unreachable
  allocations, rebuild path distances. The pass order is the contract.
- **Timeless and abyss jewels are computed, not looked up.** The port ships no
  LUT bins: `tree/historic.go` implements the reverse-engineered generation
  (a TinyMT32 variant seeded per (node graph id, jewel seed), weighted
  replacement/addition pools from `data/raw/conquertables.json`) and the abyss
  half ports the open C# generator (weighted random walk from the socket,
  per-node rolls, Zorath's ascendancy picks). Proven against all 33,148,707
  legion cells and every abyss record. **The two generators draw differently —
  legion by plain modulo, abyss by rejection sampling — and the difference is
  per-bit observable.**
- The alternate-passive pool must keep **file array order**: the reference
  indexes it positionally (slots 77/78 Might/Legacy of the Vaal, 91 templar
  devotion, 110 eternal blank; an id ≥337 means "replace with pool node
  id+1−337", below means "add addition id").
- Legacy cluster hashes (saves before the current scheme) are remapped at load
  through a one-shot legacy-id → current-id conversion.

### 8.6 Skills tab

Skill sets → socket groups (one `<Skill>` element: enabled, label, slot,
optional `source` marking a granted group, imbued support, main-skill
selection) → gem instances. A gem resolves in three ways in order: gem id +
variant id (or granted-effect id in the old format), then skill id (item-granted
skills, possibly with no gem), then name text through `FindSkillGem`'s five
increasingly permissive passes (each must match exactly one gem or the name is
ambiguous).

The main socket group is `min(max(numberOfGroups,1), configuredIndex)` with 0
meaning 1, and is **always processed even when disabled**.

### 8.7 Assembling a build

`Modules/Build.lua` loads a saved build by handing each `<...>` element to
the tab that owns it, then feeds the tabs' state to `calcs.buildOutput`.
Package `build` ports the load half of that: `build.Load(xml, tree)` returns
a `calc.BuildInput` plus the models it came from (`*tree.Spec`, the item
pool, `*skills.Tab`).

What the slot table costs to reproduce is worth knowing, because it is not
in any tab's data — `ItemsTab` *constructs* it: the 18 base slots in a fixed
order, a `" Swap"` twin after each weapon, six abyssal-socket sub-slots under
each weapon-swap twin and again under each of the seven slots that host them,
then one `"Jewel <id>"` slot per tree socket node in ascending id. 131 slots
on a 3.29 tree. A slot's `slotNum` is `tonumber(name:match("%d+$") or
name:match("%d+"))` — trailing digits win, so `"Weapon 1 Swap"` is 1 and
`"Weapon 1 Abyssal Socket 3"` is 3. Equipped items come from the active
`<ItemSet>` for ordinary slots and from `spec.jewels` for socket slots.
`test/build_test.go` compares all of it against the archive: 6,157 slots
byte-identical across 47 builds.

Two things in that construction are **not** reproduced, deliberately:

- `UpdateSockets` relabels allocated socket slots `"Socket #1"`, `"Socket
  #2"`… That is a layout pass, not load state — the archive's dumps all
  carry the bare `"Socket"`, so the port never numbers them.
- `slot.containJewelSocket` is read by `CalcSetup` (and two trade modules)
  and **assigned nowhere in the reference**. It is always nil, so the
  corrupted-jewel-effect branch that tests it always takes the true arm.

Three traps in the config port, all from the option table reaching through
the tab's widgets: `SetPlaceholder(v, true)` hands the value to the
control, which stringifies it and parses it back, so a stored placeholder
is quantized to `%.14g`; `SetPlaceholder("", true)` parses to nil and
therefore DELETES the key; and a string-valued `<Placeholder>` element is
stored as an *input*, not a placeholder.

**The config tab.** `Classes/ConfigTab.lua` + `Modules/ConfigOptions.lua`
(4,179 lines, 580 options, 532 apply closures) produce `configInput`,
`configPlaceholder`, `configModList` and `configEnemyModList`. Package
`config` ports the load half, `BuildModList` and all 532 apply bodies —
487 converted mechanically from the reference text, the rest hand-ported —
plus `Data/ModMap.lua`'s 41 map-affix appliers, which the eight map
dropdowns dispatch into (4 of those are empty in the reference).

Why the defaults dominate: an option's apply closure runs on its *default*
as much as on a user selection, so a build whose XML sets two options still
draws ~32 Config-sourced modifiers (corpus range 31–48). The placeholder
half is computed as well — enemy armour, evasion, resistances and damage
scaled to the enemy level, all written by the `enemyIsBoss` preset as it
applies, and rewritten by `presetBossSkills` when a boss skill is named.

With config in, `build.Load` fills every field of `calc.BuildInput`, and
the calc differential runs on it: 145 variants agree with the archive from
a build XML alone.

The build corpus only sets about 32 of the 580 options, which would have
left the rest unverified. `tools/dump_config.lua` closes that: it drives
the archive itself, setting every option in turn on an otherwise-default
build — each value of a list option, four of a numeric one — and records
what `BuildModList` produced. 1,254 cases, 73,824 comparisons, all 580
options, zero disagreements. **The corpus is not the limit of what the
differential can reach; it is only the inputs that happened to be lying
around.** The same move is available anywhere a port's inputs are
enumerable (`modstore_test.go`'s 59 synthetic mods were the first).

### 8.8 Export and data

Pipeline: `Content.ggpk` → (`bun_extract_file.exe`, not ported) → `.datc64`
tables → `export/` builders → typed JSON documents (`data/schema`) →
`data.Load` → package-level game data → consumed by calc/item/tree.

- 24 reference scripts: 21 are Go document builders registered into
  `export.Scripts` by package `init`; `enums.lua` becomes
  `export.WriteEnumFiles`; `legionSprites.lua` (a GIMP sprite pipeline) is
  excluded by decision; `uTextToMods.lua` has no port because its item-type
  list is entirely commented out in the reference.
- **Two tables do not exist in the game archive at all** —
  `influenceTypes.datc64` and `passiveSkillTypes.datc64` are synthesised.
  `WriteEnumFiles` must run **before** `LoadDats`.
- The reference's `Data/*.lua` interleave generated data with hand-maintained
  templates whose `#directive` lines mark insertion points. In the port those
  templates are structured JSON under `export/templates/` and **directive order
  is load-bearing** — documents pair outputs positionally (one skill header per
  `#skill`, one tail per `#mods`), and the builders carry running state.
- **`data.Load` has real ordering dependencies**: mod cache into the parser
  first, then misc tables, mod pools, item bases, the cluster-notable index
  (computed from the cluster pool), skills/gems/minions, uniques. Tree-dependent
  uniques are appended later by `tree.Load`, truncating back to a recorded
  length first so repeated loads cannot duplicate them.
- Loaded data is **package-level and immutable after Load**.
- Only documents something reads are embedded; `data/raw.go` embeds by explicit
  filename, so a new tree version needs the `//go:embed` list edited by hand.
- The stat-description engine parses GGG's `Metadata/StatDescriptions/*.txt`
  (blocks of limits + quoted text + named transforms) and is where nearly every
  human-readable mod line comes from. Two deliberately different parsers exist
  (engine vs export script); sharing them would break byte agreement. Which
  file is loaded is **context state persisting across scripts in one run**, so
  a script that does not pin its file inherits the previous one's.

---

## 9. Path of Exile domain

`poe-data-model.md` is the archive-verified domain reference: the modifier
model (name/type/value/two flag bitmasks/source/tags), the 29 tag kinds, the
15 value shapes, crafting pools, item bases, skills and gems, minions and
bosses, cluster and timeless jewels, node overrides, the text pipeline. Read it
before any PoE-mechanics work and extend it as ports teach more. Two
supplements learned since:

- `notMinionStat` appends its negated parent-actor tag **only** when the
  granted effect is a support or has the Buff skill type — `poe-data-model.md`
  states it unconditionally.
- **Tattoos and runegrafts are one override system.** 32 of the shipped
  overrides carry their own `name`; those are runegrafts (override type
  `AlternateMastery`) which replace masteries, not tattoos, though PoB ships
  them in the same pool. Terminology set 2026-08-28: *conquering* is the
  mechanic, *historic* the item keyword, *timeless jewel* the basetype family
  (types 1–6), *abyss* the eye-jewel family (7–11), *alternate* the game's own
  term for the replacement passives.

---

## 10. Tooling and operations

### Dump harnesses (`tools/`, nine Lua files)

Run **from `.archive/src/`** with `luajit ../../tools/<name>.lua` — every one
hard-codes that relative path.

| harness | output | contents |
|---|---|---|
| `dump_parse.lua` | `parse_archive.jsonl` | 13,173 records, one per corpus line |
| `dump_modtables.lua` | `tables_archive.jsonl` | 8,800 pattern-table entries across 20 tables |
| `dump_modtools.lua` | `modtools_archive.jsonl` | 10,752 records × 7 modLib behaviours |
| `dump_modstore.lua` | `modstore_archive.jsonl` | the fixture world + 18,525 checks |
| `dump_gamedata.lua` | `gamedata_archive.jsonl` | 136 subtrees of the booted `data` table |
| `dump_tree.lua` | `tree_archive.jsonl` | a freshly built PassiveTree (before any calc) |
| `dump_calc.lua` | `calc_<key>.jsonl` | per build, three variants, every stage checkpoint |
| `gen_pairs_orders.lua` | `pairs_orders.txt` | 252,060 LuaJIT iteration orders |
| `canon.lua` | — | the shared canonical serialiser |

Re-dump the whole calc corpus with the loop in `test/corpus/manifest.tsv`'s
header: `cd .archive/src && luajit ../../tools/dump_calc.lua "$k" "$f"` per row.

### Go commands

```sh
go test ./...                                            # every committed-fixture differential (~90s)
MP_EXPORT=1 go test ./test -run TestExportAgainstReference # 123 files, needs the GGPK (~97-112s)
go run ./cmd/pobexport -src .archive/src/Export/ggpk -out data/raw [script ...]
go run ./cmd/sourceupdate [-treetag 3.29.1]              # full league/data update
go run ./cmd/sourceupdate -modcache-only                 # just modcache.jsonl
luajit ../../tools/dump_config.lua                        # from .archive/src: the per-option config dump
go run ./cmd/treegen                                     # tree_<version>.json from GGG's published JSON
```

`sourceupdate` regenerates every artifact, fetches the tree JSON by release
tag, rebuilds the mod cache from the freshly written documents
(`RawSourcesFromDir`, because `go run` embeds the *old* `data/raw`), reports
what it cannot regenerate, and runs the suite. **Caveat:** the four tree-side
documents are read through `RawDoc` (always embedded), so after a tree bump one
run is not enough to converge. **A partial `pobexport` run does not produce a
complete `data/raw`** — `conquertables.json` and `modfoulbornmap.jsonc` are
only written on a full run.

### Test switches

| variable | effect |
|---|---|
| `MP_EXPORT=1` | enables the export differential (off by default: needs the GGPK, ~97s) |
| `MP_ONLY`, `MP_ONLY_ITEM`, `MP_ONLY_SKILLS`, `MP_ONLY_SPEC`, `MP_ONLY_BUILD`, `MP_ONLY_CONFIG`, `MP_ONLY_OPTION` | narrow to one build/option/prefix |
| `MP_FIXTURE=1` | revert the calc to pure fixture replay (bypass the native bridge) |
| `MP_NODRIVER=1` | skip filling the global cache via the `BuildOutput` driver |
| `MP_GUARDS` | turn an unported-branch panic into a reported failure and carry on, so one run enumerates the whole guard surface |
| `MP_DUMPGC=<path>` | write the computed global cache and the expected one (`<path>.want`) for diffing |
| `MP_PROFILE=<key>` | run the timing harness on one build |

Parse-test flags: `-diffs=N` (how many disagreeing lines to print, default 10),
`-diffgrep=<substring>`.

### Artifacts *(as of 2026-08-31)*

`data/raw/` is 25 files / ~46 MB, all committed and embedded. Largest:
`mods.json` 15.9 MB, `skills.json` 12.3 MB, `statdesc.json` 8.1 MB,
`modcache.jsonl` 2.8 MB (13,173 lines), `modscalability.json` 1.6 MB,
`tree_3_29.json` 1.5 MB. Three carry cheap provenance guards that run without
the GGPK: `TestModCacheGeneration`, `TestTreeArtifactMatchesGGGSource`,
`TestTattooArtifactMatchesArchive`.

`test/testdata/` is 57 files / ~180 MB, committed directly (no LFS), of which
49 are calc dumps. `test/corpus/` is 97 files / 4.6 MB — 48 build XMLs (10 of
them `authored_*`) plus `manifest.tsv` (48 rows) and some `.json`/`.pobcode`
import originals. The calc differential runs **145 variants over those 49 dump
files**.

### Repo skills (`.claude/skills/`)

`bump` (rebase `.archive` onto a newer PoB release), `cook` (build a PoE
character to a written recipe inside the headless reference — required to read
`poe-data-model.md` first), `dev` (toggle PoB dev mode), plus the `claudify`
command/agent for compressing `.claude/` instruction files.

---

## 11. Standing decisions

| decision | date | note |
|---|---|---|
| Say **"the archive"**, never "oracle" | 2026-08-20 | applies to conversation, docs, comments, filenames |
| **Solo project** | — | one developer, one consumer; never frame risk around hypothetical collaborators |
| **No Lua in the product** | 2026-08-28/29 | conventional Go + JSON, in shape as well as format; Lua-shape knowledge only under `test/` |
| **Regex, not a Lua-pattern engine** | 2026-08-20 | a pattern engine was built first and rejected: "i wanted regex, I wanted to be able to maintain it" |
| **Party tab deferred** | 2026-08-26 | its calc guards are not gaps; do not propose party work |
| **`legionSprites.lua` excluded** | — | GIMP sprite pipeline; checked-in PNGs stay |
| **Timeless/abyss LUTs computed, not shipped** | 2026-08-27 | the bins live only in `.archive` |
| **LuaJIT pairs emulation stays test-side** | 2026-08-27 | the revisit condition (calc output surfacing mod order) is now testable and untested — see 6.2 |
| **`%.14g` mod-cache quantization kept** | — | match PoB now, true precision later; one-call switch |
| **Lua data generators deleted** | 2026-08-29 | emitted Go converted once to typed form, no regeneration path |
| **`lua:"…"` tags kept** | 2026-08-29 | cost recorded; the alternative is a test-side field-name table |

---

## 12. Stale names, and where knowledge lives

Historical names that appear in memories, commit messages and the README but
**no longer exist** — all verified gone:

| old | current |
|---|---|
| `internal/luacanon`, `internal/luarender`, `internal/luapat` | `test/luacanon`, `test/luarender`, `test/luapat` |
| `gamedata/` | `data/schema` (package `schema`) |
| `export/luaprng.go`, `export/luatab.go` | `test/luarender/luaprng.go`, `luatab.go` |
| `modparser/canon.go`, `modparser.Canon` | `test/luacanon` |
| `data/*_gen.go`, `modparser/vocab_gen.go` | `data/skills_custom.go`, `mapmods.go`, `timeless.go`, `skillstatmap.go`, `modparser/vocab.go` |
| `data/modexpr.go`, `data/luanum.go` | gone (structured mods via the codec; `internal/util`) |
| `tools/gen_vocab.lua`, `gen_skilldata.lua`, `gen_datatables.lua`, `dump_modcache.lua`, `dump_tattoo.lua`, `dump_tables.lua` | deleted |
| `tree/timeless.go`, `tree/abyss.go` | `tree/historic.go` |
| `modstore.Externals` | `modstore.Resolver` |
| `Env.AllocOrders` / `ReplayInput` order fields | deleted 2026-08-31; `sortedIntKeys` |
| `ReplayInput.GrantedPassiveNodes` / `.GrantedAscendancyNodes` | deleted 2026-08-31; `SpecInput.Passives` (`calc.PassiveLookup`, implemented by `build.Passives`) |
| `ReplayInput.EnergyBladeItems` | deleted 2026-08-31; `calc.energyBladeFor` builds the weapon through the item package |
| `test.itemInputOf` / `test.nodeInputOf` / `test.nativeSpecInput`'s projection | `calc.ItemInputOf`, `build.SpecNodeInput`, `build.SpecInput` |
| `test/tables_test.go` | `test/modtables_test.go` |

Drift corrected 2026-08-31 (recorded so the fix is not re-litigated):
`README.md` (Parse's three-value signature, `data/schema`, `test/luarender`,
`test/luapat`, 21 of 24 export scripts), `parity.md` (145 calc variants not
147, 145 not 75, the retired `UnportedFn` marker, the native bridge, and the
mod-cache generator), `poe-data-model.md` and `lua-go-map.md` (`data/schema`),
`calc-core-plan.md` (a superseded-by-luagtfo table at its head),
`later.md` (every `#EVAL` and helper citation re-pointed at its current
line), `data/schema/gamedata.go` (the package comment said "Package
gamedata"), and `cmd/treegen/main.go` (its header claimed to reproduce PoB's
`jsonToLua` mangling; `export/treegen.go` is authoritative and says that stage
is deliberately **not** reproduced). Citations decay fast here — locate code by
name, and treat any line number older than a few commits as a hint.

Knowledge lives in three places and they have different lifetimes: this
document and its companions (durable, in-repo, versioned with the code); the
session memory store at
`E:/env/claude/projects/e--tools-missingPassives/memory/` (cross-session
recall — accurate on method and traps, drifted on counts and file locations);
and code comments plus `#EVAL` markers (authoritative at the site, and the only
form that travels with a refactor).
