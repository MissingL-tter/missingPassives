# Later — deferred items and reference quirks

Two lists that outlive the 2026-08-29 Lua remodel
(`.claude/documentation/deprecated/go-remodel-plan.md`):

1. **Kept Lua-derived code** — production code that reproduces the
   reference's behaviour and survived the remodel by decision, with the
   reason each one stayed.
2. **`#EVAL` inventory** — every reference quirk reproduced deliberately.
   Per `README.md`, each is a candidate to fix or delete once the archive
   comparison stops being the contract.

Neither list is work queued. Both are things to revisit when the archive
is deleted, or when a behaviour here turns out to matter.

---

## 1. Kept Lua-derived code

Note (2026-08-30): commit c456df051's message claims "no Lua-named function,
truthiness helper, or tostring-style coercion survives in the product". Read
literally that is false — `util.Tonumber`, `FormatG14`/`FormatIntOrG14`,
`cap1`, `modparser.Truthy`, `OutValue.Truthy` survive. Every survivor is
deliberate and inventoried in this section; this list, not the commit
message, is the authoritative statement.

The remodel's rule: **a production function whose output shape is defined by
what Lua would produce belongs test-side, regardless of whether it changes a
compared byte.** These passed that rule. Their common property is that they
consume Lua-flavoured input or reproduce an operation — none of them *emits*
a Lua-shaped value.

### 1.1 Ingestion (Lua → Go)

Reading data the way the reference read it. This is the port, not a
violation.

| function | what it reproduces |
|---|---|
| `internal/util.Tonumber` | Lua `tonumber(string)`: whitespace-trimmed, decimal/exponent forms, `0x` hex integers. Documented divergences in its comment (Go also accepts `inf`/`nan` spellings, hex floats with `p` exponents, underscore digit separators; `TrimSpace` strips Unicode spaces). 13 call sites: game-data `.ot` constants, skill stat text, mod-pattern captures. |
| `util.FoldText` (`internal/util/text.go`) | `Item.lua sanitiseText`: the accent/hyphen/cp1252 fold applied to item text before matching. Extended 2026-08-29 to fold uppercase accents (`Ö→O`, `Ä→A`) on PoB's own no-two-names-differ-by-an-accent assumption. Note: the folded bytes `\xc4`/`\xd6` are also UTF-8 lead bytes, the same hazard the pre-existing `\xe4`/`\xf6` folds carry. Moved out of `item` 2026-08-30 and `data`'s divergent third copy deleted: `item` imports `data`, so the one fold table could only live below both. That unification extends the uppercase superset to gem and granted-effect names, where the reference (`Common.lua:251`) folds only lowercase — free on today's corpus (no `data/raw` file contains `Ä`/`Ö` in any byte spelling) but it is the port's invention that won, not the reference's behaviour. |
| `item.gsubLimitFunc` | `string.gsub` with a replacement-count limit and capture-array semantics — Lua's limit counts *replacements*, and the `(%d+)([^%.])` pattern consumes the suffix character. |
| `item.gsubNumberHash`, `util.StripBalanced` | the `#` number substitution and Lua's `%b<>` balanced-delimiter match. `StripBalanced` moved to `internal/util` with `FoldText`; it has a second caller (`item/parseraw.go`) that runs it over `{`/`}` rather than angles, which is why it is exported and takes its delimiters as parameters. |
| `modparser.firstToUpper` | `str:gsub("^%l", string.upper)`. |
| `modparser.literalWeight` | **not** a Lua function: a port invention. The `scan` tie-break stands in for the reference's Lua-pattern-length ordering, which the regex conversion destroyed. Any restructuring of the pattern tables must preserve it. |

### 1.2 Arithmetic

`internal/util.RoundHalfUp(v, dec)` = `floor(v*10^dec + 0.5) / 10^dec`, a port
of `Common.lua round` ([Common.lua:647](../../.archive/src/Modules/Common.lua)).

**Not a Lua-ism.** Lua has no `round` — that is why `Common.lua` defines one;
`floor(x+0.5)` is the textbook round-half-up, a language-neutral choice.
float64 in, float64 out, nothing Lua-defined in the representation; its result
*is* the product's computed answer, so changing it computes a different life
total, not fewer Lua-isms.

Measured differences from `math.Round(v*p)/p` — three classes, not one:

| v (dec=0) | `RoundHalfUp` | `math.Round` | why |
|---|---|---|---|
| −2.5 | **−2** | −3 | ties break toward +∞, not away from zero |
| −3.5 | **−3** | −4 | same |
| −0.5 | **0** | −1 | same |
| 0.49999999999999994 | **1** | 0 | `x + 0.5` rounds up to exactly 1.0 in the *addition*, before `floor` sees it |
| 4503599627370497 (2⁵²+1) | **4503599627370498** | unchanged | adding 0.5 past 2⁵² lands on the next representable value |

Consequence: this cannot be re-expressed as "`math.Round` plus a negative-tie
fix" — that reintroduces rows 4 and 5 as divergences.

**Measured 2026-08-29**: instrumenting `RoundHalfUp` to compute
`math.Round(v*p)/p` alongside its own result and running the corpus
(145 calc variants, 1,034 items, 9,050 spec nodes, tree load, game-data load)
gave **93,855 calls, 0 divergences**. So on everything the corpus exercises
the two are identical — the three classes are latent, not active. That is
"no measured difference", not "safe to replace": the corpus is 47 builds and
a negative tie is plausible in a build it does not contain.

Call-site distribution (~90): ~60 at `dec=0` (life/mana/ES/armour/evasion
totals, block and evade chances, added damage, accuracy, reservations);
16 at `dec=2` (area-of-effect mods, attack speed, weapon attack rate, skill
cost, impale effect, crit extra damage, reserved percent, trigger rate,
mirage spawn time, buff scaling); 5 at `dec=1` (regen rate and percent, flask
duration ×2, the conquered `per_minute`→`per_second` roll conversion); 6 at
`dec=10` (never alone — always the inner half of `RoundHalfUp(RoundHalfUp(x,
10), 2)`); 5 at `dec=3/4/6` (item `{range:…}` specs, `BasePercentile`, the
roll-range precision search).

**Open**: the name. `RoundHalfUp` reads as away-from-zero to most people,
which is what its sibling `roundSymmetric` (`item/tools.go:14,21`) actually
does. `RoundHalfToPositive` or `RoundHalfCeil` would say what the table shows.
Also unexplained: why the area/speed mods round at `dec=10` before `dec=2`.
It does not fix the `0.145` case (measured — still `0.14`), so it is
presumably scrubbing accumulated multiply-chain noise; that is inference, not
something the archive or the tests establish.

The other three rounders stayed local to `item` because that is their only
caller: `roundSymmetric`/`roundSymmetricDec` (half away from zero),
`floorSymmetric` (truncate), `alwaysPositiveRound` (`trunc(v+0.5)`).

### 1.3 Domain text our own parser reads back

`internal/util.FormatIntOrG14` renders numbers into PoE item and stat text
which `modparser` then parses. The format must agree with our parser;
provenance is Lua `tostring`, but the requirement is real.

Sites: `item/buildraw.go` (the whole `{range:…}` / `Selected Variant` /
`Item Level` / stat-line reconstruction), `item/applyrange.go` (range
application and the precision search), `data/uniques_watcherseye.go` and
`data/uniques_treedep.go` (generated unique text blobs and `{variant:N}`
prefixes), `data/data.go:484,493` (boss penetration description),
`tree/conquer.go:28,31` (substituting rolls into conquered stat text).

Naming question only — nothing to move.

### 1.4 Display text

`internal/util.FormatG14` in `calc/mirages.go:173,234,318` builds
`InfoMessage` (`"3 Mirage Archers using …"`). Nothing parses it. Keeping
Lua's number spelling is a free choice, not a constraint.

(The codec's Lua-borrowed `"inf"` string spelling for infinite tag params
was retired 2026-08-30: every infinity now encodes as the tagged
`{"kind":"inf"}` object — lua-residue.md T3.)

### 1.5 Mod-cache precision — a PoB-ism, kept by decision (2026-08-29)

`modparser.Quantize14`, applied by `internal/modcachegen.BuildFrom` when it
writes `data/raw/modcache.jsonl`, sends every number in a parsed mod through
`%.14g` text and back (`internal/util.Quantize14` is the scalar half;
`modparser/modcache.go:445` walks the typed mod tree).

**Why it exists**: PoB cached parsed mod lines in `Data/ModCache.lua`, a Lua
*source* file. Writing it formatted every number as text and loading it
parsed the text back, so the values PoB actually fed to its calcs were the
round-tripped ones. Our regenerated artifact reproduces that, and the runtime
decodes those numbers rather than parsing the 13,173 lines fresh.

**PoB-ism, not Lua-ism**: the precision loss comes from PoB caching parsed
data in a serialized file, not from Lua semantics — any language's text cache
drops the same digits; `%.14g` is PoB's format choice. Direction also differs
from the removed Go→Lua converters: it reshapes nothing into Lua form, it
reproduces a value the reference's pipeline produced.

**Why kept for now**: those quantized values *are* the product's numbers.
Dropping the quantization would give full-precision parse results, shifting
inputs to every calc comparison; `parity.md` records that the differentials
compare at `%.14g` and so cannot see below the 14th significant digit
anyway — the precision floor is exactly this. The honest removal is to ship
full precision and quantize **test-side** when comparing, which is a
deliberate parity-review task, not a cleanup. `parity.md` describes the
switch (stop installing the cache with `modparser.SetModCache(nil)`; the
byte-locked differentials then flag 15th-digit diffs by design).

Guards: `TestModCacheAgainstShippedFile` (per-entry decode-echo plus a fresh
parse quantized to the same bytes) and `TestModCacheGeneration` (whole-file
regeneration). Rounding at 13 digits or at `%.15g` both fail them.

### 1.6 Truthiness — the most load-bearing Lua semantics left

Two functions reproduce Lua's `if x then`, where **only `nil` and `false` are
falsy** — `0` and `""` are true. Both are load-bearing and both were verified,
not assumed.

| function | what it reproduces |
|---|---|
| `modparser.Truthy(Value)` (`modparser/value.go`) | guards the OVERRIDE/FLAG/LIST queries in `modstore/db.go` and `modstore/list.go` where the reference tests a mod value for truthiness. A Go `!= 0` check would break an OVERRIDE of `0`. On today's tables it happens to equal `v != nil` — the only `Bool(false)` in the pattern tables is nested inside a `DataRef` — but the equality is a property of the data, not of the code. |
| `modstore.OutValue.Truthy()` (`modstore/output.go`) | `Output.Flag(key)`. **Not** the same as `Has(key)`: the calc genuinely stores `false` at 13+ sites (`calc/ehpguard.go`, `calc/ehppools.go`, `calc/ehp.go`, `calc/defencetail.go`), so absent / present-false / present-truthy are three distinct states the reference distinguishes. |

The open string key set on `modstore.Output` is load-bearing for the same
reason it looks lazy: `calc` composes ~220 read keys and ~189 write keys at
runtime (`"Base"+damageType+"DamageReductionWhenHit"`,
`ailment+"ChanceOnCrit"`) against ~1,168/897 literal ones. A struct cannot
carry that without generating a key enum.

---

## 2. `#EVAL` inventory

Reference quirks reproduced deliberately. `grep -rn '#EVAL'` is the live
list; this is a snapshot with the reason each one exists.

### calc — Lua parse-precedence and dead-branch artifacts

| site | quirk |
|---|---|
| `calc/defencetail.go:180` | the reference reads a nil global `modSource`, so the fallback string is always the source |
| `calc/ehp.go:333` | assigns `false` in both branches ("this needs a rework as well"), so `AnyTakenReflect` never becomes true |
| `calc/ehpmaxhit.go:84` | `a or 0 + b or 0` parses as `a or (0+b) or 0`, so only the shared value is read when non-nil |
| `calc/ehpstun.go:58` | the second branch is `elseif ~= "Melee"`, which only runs when the category IS "Average", so the Melee multiplier can never apply to a non-melee category |
| `calc/items.go:211` | `{ slot = true }` keys the literal string `"slot"`, so the chain start is never in the cycle set |
| `calc/mirages.go:279` | also checks `SkillType.Totem`, which `Global.lua` never defines — the nil index reads nil, so that arm is dead |
| `calc/offenceailments.go:179` | `(Flag and 100 or 0) + Sum(...)` parses as `Flag and 100 or (0 + Sum(...))`, so an immune enemy yields exactly 100 and the avoid sum is dropped |
| `calc/offencecrit.go:231` | `Sum(...) or 0 + X + Y` parses as `Sum(...) or (0+X+Y)`; `Sum` always returns a number, so the enemy and on-crit terms are dead |
| `calc/offencedamage.go:150` | `dotCfg` is an undeclared global here, so the hit resist is looked up with a nil cfg |
| `calc/offenceduration.go:94` | `"reserveDuration"` is lowercase in the reference's list while the output key is `"ReserveDuration"`, so that entry never finds a duration |
| `calc/offencehitrate.go:40` | `More("MORE", cfg, "Accuracy")` — the `"MORE"` string lands in the cfg slot and the real cfg becomes a never-matching modifier name, making this a cfg-less `More` |
| `calc/offencemisc.go:78` | `cfg` is not the pass cfg (that local died with the loop above) but an undeclared global, i.e. nil |
| `calc/offenceselfhit.go:188` | the Forbidden Rite block iterates `ipairs({["FRDamageTaken"]=...})` — only a string key, so `ipairs` yields nothing and the block never runs |
| `calc/offenceskilldata.go:336` | `Sum(...) or 100` — `Sum` always returns a number, so the fallback is dead and an absent mod means a 0% multiplier |
| `calc/perform.go:217` | the trailing `and Sum(...,"MaxDoom")` is a bare number, always truthy, gating nothing |
| `calc/performbuffs.go:384` | merges the UNSCALED list here |
| `calc/performbuffs.go:1091` | `env.player.Gloves` is never set, so this branch always marks Unencumbered |
| `calc/performbuffs.go:1195` | the first clause `Val > 0 or Sum(...)` is always truthy: `Sum` returns a number and 0 is truthy in Lua |
| `calc/performmisc.go:282` | `m_max(Sum, Override) or default` — `Override` returns NO VALUES when unset, collapsing to one-argument `m_max`, and the `or default` tail is dead because 0 is truthy |
| `calc/performmisc.go:325` | same shape for Shock |
| `calc/skills.go:92` | indexes `gemForSkill` (keyed by granted-effect TABLE) with the skillId STRING, which never matches, so item-granted skills never resolve a gem |
| `calc/tools.go:99` | for a levels table like `{6, 7}` this ignores `levelRequirement`; shipped data only has single-entry cases |
| `calc/triggerhandler.go:186` | the guard is on `actor.mainSkill.triggeredBy` but the read is on `env.player.mainSkill.triggeredBy` — the same table for the player actor |
| `calc/triggerhandler.go:225` | an earlier note claimed the field was dead; that held only for the paths ported at the time |
| `calc/triggerhandler.go:355` | the second arm compares two freshly built tables, never true in Lua, so it reduces to `ignoresTickRate and not config.triggeredSkillCond` |
| `calc/triggerhandler.go:548` | no configTable entry ever SETS `stagesAreOverlaps` — a reference-dead hook, kept so the expression is the reference's rather than a guess |

Two `#EVAL`s in calc are **not** reference quirks but decisions recorded
2026-08-29: `calc/performbuffs.go:89` (`performBuffs`, ~1,000 lines) and
`calc/skillmods.go:200` (`buildActiveSkillModList`, ~800 lines) are straight
transliterations of the reference bodies, left unsplit.

### modstore — aggregation and aliasing quirks

| site | quirk |
|---|---|
| `modstore/db.go:23` | only `ModDB.SumInternal` guards source-less mods (its extra `mod.source and` check); every other aggregation errors on them, so `guardNil=false` panics |
| `modstore/db.go:223`, `list.go:174` | `or nullValue` reads an undefined global (nil), so failed evaluations are dropped silently |
| `modstore/db.go:288` | see `HasMod` |
| `modstore/eval.go:256,303` | writes the computed `div` back into the SHARED tag, visible to every later evaluation of that mod |
| `modstore/eval.go:369` | the reference shadows its accumulator with the loop variable and adds a stat NAME to a number, which errors |
| `modstore/keystones.go:20` | mutates the keystone map's own mods through `setSource`, so the tree's shared modList carries the last granter's source |
| `modstore/list.go:55` | `copyTable(self[i], true)` is SHALLOW, so the merged copy shares tag tables (and their mutations) with the original |
| `modstore/store.go:346` | `val > (max or 0)` means all-negative candidates never register (`Max` of {−5,−2} is nil, not −2) |
| `modstore/store.go:380` | only `ModDB` implements `HasModInternal`; calling it on a `ModList` errors in the reference, so this panics |

### data / modparser

| site | quirk |
|---|---|
| `data/minions.go:66` | see `Misc.EnergyShieldRechargeBase` |
| `data/mods.go:78` | the exporter wraps joined lines in quotes, so zero described lines load as `{ "" }`, not `{ }` |
| `data/schema/mods.go:32` | entry order is a LuaJIT hash-table artifact preserved for archive parity — sort by hash once the format is Go-owned |
| `data/tables.go:108` | `Data.lua` writes this key twice in one table constructor (the derived value, then 0.33); under LuaJIT the derived value survives |
| `modparser/special.go:2993` | the reference writes `SkillTypeTotem`, which does not exist in `Global.lua`, so the list carries nil |
| `modparser/tags.go:740` | `"when you warcry"` appears twice in the reference with the same value (duplicate table key) |

### export

| site | quirk |
|---|---|
| `export/script_bases.go:83` | a `remove_tag` for an absent tag is `table.remove(tags, nil)`, which pops the LAST element |
| `export/script_bossdata.go:541` | the Lua adds to `base.count` instead of `uber.count` (a bug it keeps) |
| `export/script_enchant.go:269` | compares `SkillTypes` ROWS against the number 39, always false; only the id substring check can set `isVaal` |
| `export/script_skills.go:120` | `statInterpolation` cells are aliased and mutated across levels sharing a stat row |
| `export/statdesc.go:270` | adjusts the cached copy's order a second time, leaving the per-file cache with skewed orders |
| `export/statdesc.go:539` | `ItemClasses` is never defined in the Lua either; reaching this errored there too |

### test-side (dies with the archive)

| site | quirk |
|---|---|
| `test/luarender/luaprng.go:6` | LuaJIT `math.random` replica — once the generated data format is Go-owned, the random layout offsets deserve replacing and this file goes away |
| `test/luarender/luatab.go:5` | LuaJIT hash-table order replica — the `tradeHashes` ordering is an artifact, not data; sort the keys and delete this file once the format is Go-owned |
| `test/luarender/minions.go:19` | the reference's comma logic skips nested-table entries, so a nested table abuts its neighbour without a separator |
| `test/modtables_test.go:113` | `referenceNondeterminism` lists entries where the archive itself differs run to run (two cluster-jewel sizes sharing enchant text with different values) |

### Prose `#EVAL` references (not sites)

`README.md:188-194` documents the convention; `.claude/documentation/parity.md:11`
states that reproduced bugs get tagged; `.claude/documentation/calc-core-plan.md`
lines 88, 156-157, 202, 243-255, 303-307 discuss specific quirks in narrative
form (that document's statuses drift — the tests and `parity.md` are current
truth); `modstore/store.go:5` is the package-level note that its quirks are
tagged.
