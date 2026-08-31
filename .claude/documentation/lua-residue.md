# lua-residue — stage 3: what the 2026-08-30 adversarial review found

Successor to `go-remodel-plan.md` (stage 1) and `lua-gtfo.md` (stage 2). Written
against `311fd7d75`; baseline `go test -count=1 ./...` green, `MP_EXPORT=1`
123/0. Every claim below carries its evidence pointer — the lesson of stage 2
was that verdicts without citations get inherited instead of re-checked.

Line numbers are hints; locate by name.

## T1 — the replay machinery reproduces sorted order

`calc/setup.go` `nextAllocOrder` panics without a dump-captured
`pairs()` order; `ReplayInput` carries AllocOrders/NodeOrders/Mirage* and only
`test/calc_test.go:912` constructs it. Plan L10/S12 called the order "LuaJIT
hash order, not derivable". **Evidence it is derivable:** `tools/dump_calc.lua`
installs `pairs = sortedPairs` before the Calc modules load (`:131`, comment at
`:49-55`) and the recording wrapper iterates `for id in sortedPairs(nodeList)`
(`:290`); fixture orders ascend (calc_cyclone allocOrders: 476, 5233, 54127…).

Fix: iterate sorted keys of `env.AllocNodes` / `env.ExtraRadiusNodeList`;
delete `Env.AllocOrders/ExtraOrders/allocOrderIdx`, the `orderStart` threading,
`ReplayInput.{AllocOrders,NodeOrders,MirageAllocOrders,MirageNodeOrders}`, the
mirage order swap. Test-side: assert every recorded fixture order is ascending
(proof the derivation is faithful), then feed nothing.

Follow-on (not in this stage, direction decision): the rest of `ReplayInput` —
GrantedPassiveNodes/GrantedAscendancyNodes resolvable from `tree.Spec`,
`ItemInput.FuncTypes` + `deriveFuncList` replaced by `item.JewelData.FuncList`
(built at `item/buildmodlist.go:385`; the re-parse-and-assert in
`calc/radius.go` is a test oracle running in production), EnergyBladeItems —
would let `BuildInput` be built natively in production, not only in
`test/calc_native_test.go`.

## T2 — correctness fixes (reference divergences the corpus never reaches)

| site | divergence | fix |
|---|---|---|
| `calc/ehpguard.go:139-148` | `!Has("TotalMinionLife")` for the reference's `not x or x == 0` (CalcDefence.lua:2484); `SetN(…, N(ally.life))` stores 0 where the reference assigns a possibly-nil; `specificOverride != nil` instead of the ok bit | `if v, ok := Override; ok && …` / `else if output.N(k) == 0 { output.Set(k, output.Get(ally.life)) }` |
| `modparser/jewels.go:139` | `data.Stat(stat) != 0` for `data[stat] ~= 0` (ModParser.lua:6318) — `nil ~= 0` is true, so an empty radius emits a zero-valued mod in the reference and nothing here | `!data.HasStat(stat) \|\| data.Stat(stat) != 0` |
| `modstore/eval.go:154-168` (lua-gtfo B1, half-applied) | reservation branch reads `Actor.ActiveSkillList`, which no production path fills; PerStat tags on Mana/LifeReservedPercent (modparser/tags.go:192,195) evaluate 0 where the reference computes `floor(base/total*100)` | make `modstore.ActiveSkill` an interface over live calc state; calc wires `env.Player.ActiveSkillList` |
| `calc/performbuffs.go:553,641` | `setSpectreSource` writes `Source` in place on mods aliasing `data.Skills` (skillmods.go:174 appends BaseMods pointers) — cross-build contamination of loaded data; also `performutil.go` applyEnemyModifiers | stamp clones; merged DB bytes identical |
| `modparser/parse.go:383-389` | DOUBLED writes `Names[1]` into the shared `*PatternEntry` (pattern.go nameEntry returns the table's pointer) — reference bug (ModParser.lua:6795) with cross-line contamination; every fixture DOUBLED line hits a `name(...)` entry so nothing observes it | copy the entry before writing |
| `calc/perform.go:107` vs `calc/performminion.go:92,135` | `Ms.ItemList` copied from `Minion.ItemList` before initMinionModDB writes Weapon 3 / item-set slots; every reader uses `Ms.ItemList` → minions never get Using* item conditions (fixtures show only OffHandIsEmpty) | initMinionModDB writes both maps (minimal); full actor unification stays opportunistic |
| `item/itemdata.go:425-441` | `num`/`str` discard ok: a mis-kinded LIST value silently writes 0/"" (the shape lua-gtfo A5 deleted from calc) | nil stays 0/"" (Lua nil-assignment); any other wrong kind panics |
| `calc/triggerconfig.go` | header claims every reference configTable key present; `"avenging flame"` (CalcTriggers.lua:1486) is missing → silent no-trigger instead of the documented panic | add `"avenging flame": nil` |

None of these change any current fixture byte (each was verified unexercised);
the suite is the guard that the fix itself is inert on the corpus.

## T3 — the last live Lua coercion

`NumOf`'s Str arm (`modparser/value.go:203`) is live: five closures emit
numeric captures as `Str` — `special.go:1100` (LightningMin), `:1778`
(BuffEffect), `:2090` (PierceCount), `:2361` (PseudoRecoupDuration), `:3321`
(ExtraSkillStat ×2). Shipped `modcache.jsonl` carries `"value":"2"` /
`"value":"10"`; `parse_archive.jsonl` the same two. Stage 1's S3 fix covered 7
closures and stopped counting.

Fix, one wave: `c.str`→`c.v` at the five sites; extend the test-side
normalisation (`test/luacanon/modcanon.go NormalizeArchiveMods` + the decoder
in `test/luacanon/decode.go`) to numeric-text scalar values; fix the codec's
infinity spelling for tag params (`modparser/modcache.go encodeParam` Num case
emits the text `"inf"` where values get `{"kind":"inf"}` — the only reason
`paramReader`'s Str arm has a producer besides the test decoder); regenerate
`modcache.jsonl`. Then delete: `NumOf` Str arm, `calc/tools.go valueNum` Str
arm, `modstore/eval.go valueNum` (via NumOf), `calc/skilldata.go SkillData.N`
Strs fallback (settles lua-gtfo §5's twin-N question by deletion — no `SetStr`
site stores numeric text), `paramReader` numOfParam Str arm and `nums`
StrList arm (keep the empty-list case).

`GrantedPassive` (`special.go:3032`) and `impossibleEscapeKeystone`
(`:2684`) stay Str — genuinely names.

## T4 — Lua directive/source text still in artifacts (B2 class)

One sub-item at a time; each regenerates `data/raw` and the diff is committed
with the fix. The archive comparison is on the rendered `.lua`
(`test/luarender`), which re-spells the Lua form.

1. `schema.SkillTail.ModsArgs string` — exporter joins typed flags
   (`export/script_skills.go:488`), data re-parses by `strings.Contains`
   (`data/skills.go:735`). → `Flags []string`; luarender's `#mods` line comes
   from the archive template, unaffected.
2. `schema.KV.Value string` (raw `.ot` text) — evaluated at load with
   `util.Tonumber` (`data/data.go otConstantMap`). → parse at the export edge,
   `Value float64` (census: all values plain decimal integers); luarender
   re-renders the number.
3. `DescribeModTags` builds `"a", "b"` Lua list text
   (`export/statdesc.go:620`); `data/items.go splitModTags` un-quotes it, for
   mods/bases/masters/crucible. → `ModTags []string` throughout; luarender
   quotes-and-joins.
4. The `{"kind":"mixed","arr":…,"kv":…}` Lua-table record in
   `export/templates/skills/sup_str.json:26`, decoded by renaming `type`→`kind`
   (`data/skills.go:648-686`). → a typed `{"kind":"typo",…}` record in the
   template; decoder reads it directly; skills.json regenerates.

## Opportunistic (do when touching the area; not scheduled)

Store `self`/`base()` + duplicated aggregation → `modsNamed` iterator, and the
two `*Internal` exports; `weaponOf`/`*Item` downcasts (57+21 sites) →
concrete types on `modstore.Actor`; itemdata per-kind key enums + dead
`WeaponData.Extra`; `TimelessPassive` positional `[]int` → typed record
(0-based pools, slots by id) with the bin test flattening test-side, and the
`jewelType` magic ints → `ConquerorKind`; `ProcessSocketGroup`/
`ValidateGemLevel` de-duplication (skills vs calc, bodies already diverge);
trigger item-title regex → `SourceItem.In.Title`; `RawTag`+`ParseTags`/
`ParseFormattedSourceMod`/`FormatSourceMod` to test-side; gem-count
`**float64` → plain fields; `export/enums.go` deletion (nothing reads the
synthetic tables — no script reads an Enum column referencing them; dat.go
only gates on presence); Explosive Arrow via the callback registry;
test-only surface relocations (`Tables()`/`PatternEntry`,
`SkipTreeDependentUniques` — dead even for its test, `LoadedModCache`,
`StubHandoff`, `TimelessAdditions`, `Stats.ModKey`, minion artifact
provenance rows).

## Not doing, by decision

float64→int sweep (churn, no behaviour); remaining closed-set bags
(`SkillLevel.Extra`, `ModLine.Flags`, `Influence`, `DataRef.Merge`, actor
string constants); naming cosmetics (`m_huge`, `cap1`); everything stage 1/2
kept by recorded decision (data globals, `lua:` tags, tag nil holes, `%.14g`
family, `RoundHalfUp`, `Row.cells`).

## Status

Execution 2026-08-30, this session. Filled in per wave below.

| wave | outcome |
|---|---|
| docs | landed — plan L10/S12 corrected, lua-go-map row fixed, later.md notes the commit-claim overstatement |
| T1 | landed — sorted iteration; Env.AllocOrders/ExtraOrders/allocOrderIdx, ReplayInput's four order fields, orderStart threading and mirageReplay deleted; test asserts every recorded order ascending (assertOrdersSorted); full suite green |
| T2 | landed — all eight fixes; B1 via a live modstore.ActiveSkill interface implemented by calc.ActiveSkill and wired at both PlayerActiveSkills appends; the spectre "export list mutation" of shared game data is deliberately not reproduced (comment at the site); full suite green |
| T3 | landed — five closures `c.str`→`c.v`; NumOf/valueNum×2/SkillData.N/paramReader coercion arms deleted (SkillData.N and Output.N now agree); codec infinities tagged objects both ways; fixture decoding + NormalizeArchiveMods convert the archive's numeric text (closed field set + value); modcache.jsonl regenerated (2 lines); full suite green |
| T4 | landed — all four: ModsFlags []string (schema/exporter/loader), .ot KV.Value float64 (parsed at the export edge; otConstantMap is a plain index), DescribeModTags []string end to end (splitModTags deleted; luarender joinModTags re-spells the Lua list text), typo record replaces the mixed arr/kv table (template + decoder). Artifacts regenerated (bases/crucible/masters/miscdata/mods/skills + modcache); full suite green; MP_EXPORT 123/0 |
