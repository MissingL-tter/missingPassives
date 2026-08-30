# lua-gtfo — recommended fixes for the residual Lua-isms

Follow-up to `go-remodel-plan.md`, whose 15 steps landed 2026-08-29. That plan
cured the core (`Tag`, `Value`, `Output`, the transcribed tables); this file
lists what a post-remodel review found still standing, with the concrete fix
for each. Written 2026-08-30 against `a71948fe6`.

Placed here beside `go-remodel-plan.md` and `later.md` because those are the
two documents it continues; nothing else about the location is intended.

## Baseline observed before the review

| command | result |
|---|---|
| `go test ./...` | ok, 0 failures |
| `go test ./test ./tree ./modparser ./test/luacanon -v` | 28 top-level PASS, 0 FAIL, 2 SKIP (both by design) |
| `MP_EXPORT=1 go test ./test -run TestExportAgainstReference` | PASS — 123 files checked, 0 disagreements, + 4 corruption-negative subtests |

`map[string]any` across the whole product is down to 31 sites. Every one was
adjudicated; the ones that are correct are listed in **§5 Do not touch**.

## Standing constraints

- `./.archive` is read-only. Never change it, move it, or regenerate it.
- Byte-identical parity with the archive differentials survives every fix here.
- Lua-shape understanding lives only in `test/luacanon`, `test/luapat`,
  `test/luarender`. No fix below moves any of it back into the product.
- Verification shorthands (`BUILD FULL PARSE STORE CACHE DATA ITEM SKILLS SPEC
  TIMELESS CALC EXPORT ARTIFACTS`) are defined in `go-remodel-plan.md` §7.0.

---

## 1. Tier A — still behaves like Lua

### A1. Delete the Lua `[[ ]]` stream builders in `export`

`export/script_bases.go:324-335,486` and `export/script_umodstotext.go:102-112,263`
flatten typed Go into Lua long-bracket source text, run a line-stream
transform, then re-parse it with `splitUniqueFile` (`export/script_umodstotext.go:26`).

```go
raresStream = append(raresStream, "[[")      // script_bases.go:332
raresStream = append(raresStream, lines...)
raresStream = append(raresStream, "]],")
f, err := splitUniqueFile(raresStream)        // :486 — parse it back
```

Same class as the `luaString`/`numKey` converters removed from
`export/treegen.go` on 2026-08-29: a production function whose output shape is
defined by what Lua would produce.

**Fix.** The block structure already exists as `tplDoc.Sections[].Items[][]string`.
Run the transform over it directly: `splitUniqueFile([]string) (schema.UniqueFile, error)`
becomes `buildUniqueFile(items [][]string) schema.UniqueFile`, and the item
boundary that `"]],["` encodes becomes the loop boundary. Delete both stream
builders and the bracket parser.

**Parity.** None at risk — the bracket tokens are internal; `raresStream` dies
at `script_bases.go:486` and `outLines` at `script_umodstotext.go:263`. No
artifact byte contains them.
**Verify.** `EXPORT`.

### A2. One fold table, for real

`data/skills.go:807 sanitiseText` is a third copy, called from `data/gems.go:44`
(gem names) and `data/skills.go:442` (granted-effect names). It folds only the
lowercase accents; `item.FoldText` (`item/tools.go:116-119`) also folds
`Ä→A`, `Ö→O`, `\xc4`, `\xd6`. A name carrying an uppercase accent folds to `?`
in `data` and to `A`/`O` in `item`/`skills`.

Plan step 2 says `item.FoldText` replaces both `sanitiseText`s; `later.md` §1.1
and `lua-go-map.md` both describe one fold table. There are two.

**Fix.** Delete `data/skills.go:806-840`; call `item.FoldText`. If the
`data → item` import edge is unwanted, move `FoldText` to `internal/util`
instead — that is where the plan put the rest of the shared reference
semantics — and have `item`, `skills` and `data` all call it there.

**Parity.** Provably free on the shipped corpus: `data/raw/skills.json`,
`bases.json` and `mods.json` contain zero `Ä`/`Ö`/`\xc4`/`\xd6` bytes.
**Verify.** `DATA ITEM SKILLS`.

### A3. Type the four radius-jewel tables

`modparser/jewels.go:170,379,424,462` are `map[string]any` over three
inhabitants (`jewelNodeFunc`, `jewelFactory`, `[]jewelNodeFunc`). The type is
recovered at init by a `switch` with **no default** (`:572-586`, `:595-614`) —
an unexpected type is silently dropped and its mod line stops being recognised
— plus two bare `v.(jewelNodeFunc)` (`:589`, `:592`) that panic at package init.

**Fix.** The package already has this idiom four times at
`modparser/pattern.go:207-243`. Copy it:

```go
type jewelValue interface{ isJewelValue() }
func (JewelNodeFn) isJewelValue()  {}   // jewelNodeFunc aliases this
func (jewelFactory) isJewelValue() {}
type jewelFuncSeq []JewelNodeFn
func (jewelFuncSeq) isJewelValue() {}
```

Declare all four tables `map[string]jewelValue`, wrap the `:541` entry as
`jewelFuncSeq{...}`, and keep the build-time switch — now total, with a
`default: panic` that can only fire on a new inhabitant.

**Parity.** Zero. Every entry is reduced at init to
`jewelFuncEntry{typ, nodeFn, factory, re}`; `JewelFn.ID` comes from the table
key (`modparser/parse.go:76-83`), not the map's value type.
**Verify.** `PARSE SPEC CALC`.

### A4. `getThreshold` takes `[]string`, not `any`

`modparser/jewels.go:133-155`. The doc comment states the Lua contract
outright — *"attrib is a string or a []string"* — and the body branches
`attrib.([]string)` / `attrib.(string)` into two arms identical apart from the
loop.

**Fix.** `func getThreshold(attribs []string, name string, modType ModType, value Value, tags ...Tag) jewelNodeFunc`,
same for `getThresholdF`. Single-attribute call sites become `[]string{"Dex"}`.
Delete the scalar arm.

**Parity.** None. All 67 call sites pass a `"Str"`/`"Dex"`/`"Int"` literal or a
`[]string{…}` literal. The list arm over a one-element slice is bit-identical
to the scalar arm.
**Verify.** `PARSE SPEC CALC`.

### A5. Type item sockets; delete `calc.str`

`calc/input.go:209 Sockets []map[string]any` is the last `map[string]any` in
`calc`. It is read for one key, `"color"`, at `calc/items.go:527`,
`calc/skills.go:460` and `:560`, through `calc/items.go:19`:

```go
func str(v any) string {
	switch t := v.(type) {
	case string:        return t
	case modparser.Str: return string(t)
	}
	return ""   // silent
}
```

`str`'s other two callers (`calc/perform.go:503`, `calc/performbuffs.go:1411`)
`tostring` a `modparser.Value` out of a `List(…, "Keystone")` query, where a
non-`Str` silently becomes `""` and the keystone is dropped.

**Fix.**

```go
type SocketInput struct {
	Color string `lua:"color"`
}
// calc/input.go:209
Sockets []SocketInput `lua:"sockets"`
```

The three colour reads become `socket.Color`. At the two keystone sites:
`if s, ok := v.(modparser.Str); ok { … }`. `str` deletes.
`test/luacanon/calccanon.go` builds the typed socket instead of the map.

**Parity.** The `lua:"color"` tag keeps the canon projection byte-identical.
**Verify.** `CALC SPEC`.

### A6. `evalSocketedIn`'s `match` map is a slice

`modstore/eval.go:923-973`: `match := map[string]bool{}` with four literal keys
that are never read by name — only `for _, v := range match` at `:971`.

**Fix.** `var match []bool` with `append`, or an `ok := true; ok = ok && …`
accumulator. Same treatment at the sibling loop, `modstore/eval.go:905`.

**Parity.** The loop is a pure conjunction with early return under both `Neg`
polarities, so iteration order cannot matter — this is provable from the loop
body, not an assumption.
**Verify.** `STORE CALC`.

### A7. Export the value/tag decoders; delete the fake mod document

`data/skills.go:671` synthesises a Lua-shaped whole-mod JSON document purely to
borrow decoders that exist but are unexported:

```go
carrier := map[string]any{"name": "", "type": "BASE", "flags": 0, "keywordFlags": 0, "tags": []json.RawMessage{}}
raw, _ := json.Marshal(carrier)
decoded := modparser.DecodeMod(raw)
```

**Fix.** Export narrow decoders over the existing unexported bodies
(`modparser/modcache.go:299,319`):

```go
func DecodeValue(raw json.RawMessage) (Value, error)
func DecodeTag(raw json.RawMessage) (Tag, error)
```

`decodeTypoMod` calls them directly. Its `error` return then means something —
today the panic inside `DecodeMod` bypasses it entirely.

**Parity.** Internal decode path; no artifact changes.
**Verify.** `DATA CACHE`.

### A8. Export mod pools stop being keyed by Lua file paths

`export/script_mods.go:295-320`: `modPoolOrder []string` of
`"../Data/ModExplicit.lua"` strings keying `modPoolConds map[string]func(*Row) bool`,
populated in an `init()`.

**Fix.**

```go
type modPool struct {
	Out  string            // "ModExplicit"
	Cond func(mod *Row) bool
}
var modPools = []modPool{{Out: "ModExplicit", Cond: func(mod *Row) bool { … }}, …}
```

Order is slice order; the map and the `init()` both disappear; the `.lua`
extension stops being a lookup key and becomes part of the output path where it
belongs.

**Parity.** Directive/emission order is preserved by the slice order — the
thing `EXPORT` checks.
**Verify.** `EXPORT`.

---

## 2. Tier B — structural

### B1. Reach the dead reservation path, or mark it unported

`modstore/eval.go:75,78` (`Actor.ActiveSkillList`, `Actor.HasReservation`) are
written by exactly one place in the repo — `test/modstore_test.go:201-205`.
`calc` never sets either (`calc/performminion.go:168,219` writes
`calc.Minion.ActiveSkillList`, a different type). So `getStat`'s
`reservedPercent` branch at `:158` cannot fire in production. The field comment
says so: *"the SkillType.HasReservation id the fixtures key on"*.

**Fix.** Two parts.
1. Delete `Actor.HasReservation`; compare against the constant that already
   exists — `skill.SkillTypes[modparser.SkillTypeHasReservation]`
   (`modparser/globals.go:111`); `modstore` already imports `modparser`.
2. Either wire `calc` to populate a `Skills []ReservationSkill` on
   `modstore.Actor` at the point `calc/perform.go:521` already reads the same
   skill type, or leave the branch and give it the same explicit
   "not reached by the ported path" guard the other unported branches carry.
   Silently-dead is the only unacceptable state.

**Parity.** The fixture drives the branch today and must keep driving it.
**Verify.** `STORE CALC`.

### B2. Four artifact-schema fields carry Lua text where a typed value belongs

Same class as the `BossStatValue` fix landed 2026-08-29 — one of these is 40
lines from it.

| site | ships | parsed back |
|---|---|---|
| `data/schema/bossdata.go:57` | `Text string` — a number, or the literal two-character `""` (`export/script_bossdata.go:684` writes `text := "\"\""`) | `data/bossskills.go:71 penValue` |
| `data/schema/skills.go:87` | `Interp []string` — exporter holds `int64` (`export/script_skills.go:129`), stringifies at `:508` | `data/skills.go:770-775`, `util.Tonumber` + an error path |
| `data/schema/minions.go:41` | `Hostile string // raw template text` | `data/minions.go:79-90`, with a Lua-truthiness comment |
| `data/schema/bases.go:14` | `RareBlobs` **and** `Rares` — the rare database twice | `RareBlobs` → `data/bases.go:314`; `Rares` → only `test/luarender/bases.go:181` |

**Fix.**
- `PenEntry{Name string; Value util.Opt[float64]}` serialised as
  `{"name":…,"value":25}` with the value omitted when blank — the shape
  `BossStatValue` already uses in the same file. `penValue` deletes.
- `Interp []int64`; `export/script_skills.go:507-509` appends
  `level.interp.vals...`; `data/skills.go:770-776` becomes
  `float64(t)`, deleting the `util.Tonumber` call and the
  "bad statInterpolation value" error path. `luarender` needs no change.
- `MinionDef{Hostile bool; HostileScale *float64}` decoded directly;
  `data/minions.go:79-90` collapses to two field reads.
- Keep `Rares` (directive-aligned, what the renderer walks), add
  `ExtraRares [][]string` for the template's hand-written passthrough blocks,
  drop `RareBlobs`. `data/bases.go:314` iterates `Rares` then `ExtraRares`.
  This removes a full copy of the rare database from every shipped `bases.json`.

**Parity.** All four change `data/raw/*.json` bytes. The archive comparison is
on the *rendered* `.lua`, not the JSON, so `test/luarender/{bossdata,skills,minions,bases}.go`
re-spell the Lua form test-side — exactly the `BossStatValue` recipe.
**Verify.** `DATA EXPORT ARTIFACTS` per sub-item; the resulting `data/raw` diff
is the intended one and is committed with the fix.

### B3. `ActorType` enum; delete `Actor.Others`

`modstore/eval.go:89-98` switches on `"enemy"`/`"parent"` literals then falls
through to `a.Others[actorType]`. `Actor.Others` (`:69`, commented
*"any other actor types"*) is never written anywhere — it exists only to mirror
the reference's open table lookup.

**Fix.** Follow `ConquerorKind`/`NodeKind`:

```go
type ActorType uint8
const (ActorNone ActorType = iota; ActorPlayer; ActorEnemy; ActorParent)
func (a ActorType) String() string       // over [...]string{"", "player", "enemy", "parent"}
var ActorTypeByName map[string]ActorType
```

Type `CondTag.Actor`, `StatTag.Actor`, `MultiplierTag.Actor`/`LimitActor`/
`ThresholdActor` and `Cfg.Actor`. `byType` becomes a total switch with
`default: return nil`; `Others` deletes. `ActorNone`'s zero value reproduces
the `""`-means-absent tests at `eval.go:213,226,745,752,769` unchanged.

**Shared type — runs alone**, and lands in `modparser` before `modstore`.
**Verify.** `PARSE STORE CACHE ITEM CALC DATA EXPORT`.

### B4. Skill-type sets keyed by `SkillTypeID`

`modstore/eval.go:50,119` are `map[float64]bool` while `modparser.SkillTypeID`
has existed since `modparser/globals.go:97` and every other package uses it
(`calc/activeskill.go:22`, `modparser/tag.go:92,261`). The cost is a downcast at
each use (`eval.go:598,604`) and a full per-skill map rebuild in
`calc/skillmods.go:438-441`.

**Fix.** `map[modparser.SkillTypeID]bool` on both `Cfg` and `ActiveSkill`;
`eval.go:598` becomes `cfg.SkillTypes[t]` and `:604`
`cfg.SkillTypes[tag.SkillType]`; `calc/skillmods.go:438-441` collapses to
`SkillTypes: activeSkill.SkillTypes`, deleting an allocation and copy per skill.
`test/modstore_test.go` decodes its JSON key with `strconv.ParseInt`.

**Shared type — runs alone.**
**Verify.** `STORE CALC SPEC`.

### B5. Delete the four dead `Extra` bags; type `AddedUsing`

`item/itemdata.go:39` and siblings. Only `WeaponData.Extra` is live, read at
`calc/offenceconv.go:180` — the data-driven LIST extension the plan explicitly
kept. `ArmourData.Extra`, `FlaskData.Extra`, `TinctureData.Extra` and
`JewelData.Extra` are populated (`:183,:257,:261,:283,:366`) and read by nothing.

Separately, `item/itemdata.go:37 AddedUsing map[string]bool` holds five closed
flags reached by string surgery in both directions:
`strings.HasPrefix(key, "AddedUsing")` writing (`:118-124`) and
`strings.TrimPrefix(cond, "Using")` reading (`calc/tools.go:373`).

**Fix.** Delete the four dead bags and their `setExtra` calls — a LIST key that
would have landed there now reaches the `default` and can panic, which is
correct, since nothing produces one. Replace `AddedUsing` with
`struct{ Axe, Sword, Dagger, Mace, Claw bool }` plus a small `weaponType` enum
for the two name→slot lookups (`calc/tools.go:373,409`); the real writers at
`calc/performutil.go:181-190` become direct field assignments, and the dead
`HasPrefix` arm at `item/itemdata.go:118-124` goes.

**Shared type (`item.WeaponData`) — runs alone**, and before B11.
**Verify.** `ITEM CALC SPEC SKILLS`.

### B6. Stop discarding `ok` on mod-value assertions

`calc/skillmods.go:713-720` is the clearest: `tag, _ := v.(modparser.DataRef)`
then `SkillData.Set(tag.Key, …)` — a value of any other shape writes key `""`.
Also `calc/skillmods.go:141`, `:572`, `:595`, where a zero `GemPropertyRef`
produces a mod literally named `"GemSupport"`. 18 sites total; several *others*
do guard afterwards (`calc/performbuffs.go:1030`, `calc/activeskill.go:331`), so
the two intentions are indistinguishable in the source today.

**Fix.** At each unguarded site, take the `ok` and panic in the package's
existing idiom:

```go
tag, ok := v.(modparser.DataRef)
if !ok { panic("calc: non-DataRef value in SkillData list (the Lua errors)") }
```

Where nil-tolerance *is* intended, spell it —
`ref, ok := v.(modparser.PropertyModRef); if !ok || ref.Mod == nil { continue }`.

**Parity.** None of the panics can fire on the corpus; `CALC` proves it.
**Verify.** `CALC SPEC`.

### B7. Typed row queries in `export`; delete `numeric`

`export/dat.go:158`: `GetRow(key string, value any)` / `GetRowList` sit beside
nine correctly typed read accessors, and compare through `cellEquals` and
`numeric(v any) (float64, bool)` — a cross-type numeric coercion, exactly the
Lua-semantics helper the plan set out to remove.

**Fix.** Four typed queries mirroring the typed accessors, sharing the existing
`getRowList` predicate plumbing:

```go
func (d *DatFile) RowByStr(key, v string) *Row
func (d *DatFile) RowByInt(key string, v int64) *Row
func (d *DatFile) RowByRef(key string, v *Row) *Row   // handles v == nil explicitly
func (d *DatFile) RowsByInt(key string, v int64) []*Row
func (d *DatFile) RowsByBool(key string, v bool) []*Row
func (d *DatFile) RowsByRef(key string, v *Row) []*Row
```

`numeric` and `cellEquals` delete.
**Verify.** `EXPORT`.

### B8. Named constants in the converted data tables

`data/skillstatmap.go:18` and throughout `data/skills_custom.go`, the one-off
converter typed the containers and transcribed the machine integers:
`modparser.KeywordFlag(524288)` is `KeywordAilment`; `65536` (63 sites) is
`KeywordAttack`; `786432` (36 sites) is `KeywordHit|KeywordAilment`;
`83886080` (11 sites) is `FlagUnarmed|FlagWeaponMelee`; `SkillTypeID(32)`
(30 sites) has a name. Step 3 generated `ModFlagByName`/`KeywordFlagByName` and
rewrote the *pattern* tables; these two data tables were missed.

**Fix.** Mechanical substitution to the named constants. Better, since 1,437 of
1,474 `skills_custom.go` calls and all 751 non-flagged `skillstatmap.go` calls
pass `FlagNone, KeywordNone, ""`: give `genMod` a struct-literal form so the
three empty arguments stop being spelled at all.

**Parity.** Constant substitution is value-identical; `DATA` proves it.
**Verify.** `DATA CALC SPEC`.

### B9. Move the two test-only exports behind `export_test.go`

- `modparser/verify.go:11 Tables() map[string]map[string]any` — the product's
  only remaining `map[string]map[string]any`, rebuilding a Lua
  table-of-anything. Sole caller `test/modtables_test.go:35`.
- `data/data.go:41 Sources.StatMapCopies` — written only by
  `test/gamedata_test.go:107` and `test/loaddata_test.go:83`. `data/raw.go:82`
  already documents it as *"the archive-dump replay fixture"*.

**Fix.** Go's own mechanism for this: unexport both and re-export them from an
`export_test.go` in their own package, which the differential test then reaches
through the same-package test binary. If `test/` must stay a separate package,
the smaller fix is to type `Tables()`'s return as
`map[string]map[string]TableEntry` over the `nameValue`/`entryValue` interfaces
that already exist in `modparser/pattern.go:207-243`, so at least no `any`
crosses the boundary.

**Verify.** `PARSE DATA`.

### B10. Type the stringly domains; delete the dead transform

`calc/ehp.go:40` — `damageCategoryConfig` is a bare `string` through 7
signatures and ~20 literal comparisons (`calc/ehpehp.go:50,79`,
`calc/ehpguard.go:112`, `calc/ehphit.go:76,107`,
`calc/ehpdegen.go:84,96,100,102`). A typo compiles and silently selects the
fallback branch. Same shape on `ConfigInput.EnemyDamageType`/`AilmentMode`/
`RepeatMode`/`PhysMode` (`calc/input.go:47-50`).

`data/tables.go:187 TransformKind.Apply(v any) any` is a transcribed Lua closure
with **zero callers anywhere in the tree**.

**Fix.**

```go
type DamageCategory string
const (
	DamageAverage         DamageCategory = "Average"
	DamageUntyped         DamageCategory = "Untyped"
	DamageOverTime        DamageCategory = "DamageOverTime"
	DamageMelee           DamageCategory = "Melee"
	DamageProjectile      DamageCategory = "Projectile"
	DamageSpell           DamageCategory = "Spell"
	DamageSpellProjectile DamageCategory = "SpellProjectile"
)
```

`ConfigInput.EnemyDamageType DamageCategory`; the six `ehp*` helpers take it;
concatenation sites become `output.N(string(cat)+"NotHitChance")`. Same for
`AilmentMode`, `RepeatMode`, `PhysMode`, and `Mode` (MAIN/CALCULATOR). Delete
`TransformKind.Apply` outright.

**Parity.** A defined string type has the same underlying bytes; nothing moves.
**Verify.** `CALC DATA`.

### B11. `weaponPassSource` reads `.Set`, not `!= 0`

`calc/offenceconv.go:167-179` reproduces which keys the Lua table held. For the
four `util.Opt` fields it asks the right question
(`set("AttackSpeedInc", wd.AttackSpeedInc.V, wd.AttackSpeedInc.Set)`, `:169`);
for `AttackRate`, `Range`, `ElementalDPS` and each damage type's `Min`/`Max`/
`DPS` it guesses from the value (`wd.AttackRate != 0`). A weapon whose
`PhysicalMin` is genuinely 0 omits a key the reference's `copyTable` carries.

**Fix.** Promote the remaining `item.WeaponData` numerics to the option type the
struct already uses: `AttackRate, Range, ElementalDPS util.Opt[float64]` and
`type DamageRange struct{ Min, Max, DPS util.Opt[float64] }`
(`item/itemdata.go:16-41`). `item/itemdata.go:84-88,100,102,112` write
`optNum(v)` like their siblings; `calc/offenceconv.go:167-179` becomes the same
shape as `:169` throughout.

**Do the measurement first**, using the technique `later.md` §1.2 records for
`RoundHalfUp`: instrument every site where the `!= 0` test and `.Set` disagree,
run the full corpus, and report the count. If it is zero, this is a
readability fix and can be dropped; if it is not, it is a live bug. Revert the
instrumentation completely before landing.

**Depends on B5** (same struct). **Shared type — runs alone.**
**Verify.** `ITEM CALC SPEC`.

---

## 3. Tier C — cosmetic

| site | fix |
|---|---|
| `data/uniques_generated.go`, `_generated2.go`, `_generated3.go` | Hand-written ports named `_generated`, split with meaningless numeric suffixes because a Lua file was split. Plan N5 said drop the suffix. Rename to their contents: `uniques_pools.go`, `uniques_watcherseye.go`, `uniques_thatwhichwastaken.go` |
| `calc/performmisc.go:14` | Doc comment still names the deleted `luaOr` |
| `internal/util/num.go:36` | `FormatInt(float64)` takes a float because Lua had one number type; `data/tables.go:174 itoa(int)` round-trips `int → float64 → int64`. Add `FormatIntN(int) string`, or change `itoa` to `strconv.Itoa` |
| `modstore/store.go:36` | Eight `*Internal` methods exported on an interface already unimplementable outside the package. Unexport |
| `modparser/valueparams.go:6-19` | Record value kinds are bare string constants where `TagKind` got a real enum, so `ValueParams`/`ValueFromParams` are string dispatch. `type ValueKind uint8` with `String()`, matching `TagKind` |
| `calc/input.go`, `data/tables.go` (390 `lua:"…"` tags) | **Accepted by decision** (plan §8, B3: "do nothing"). Recorded so the cost is on the page: production carries Lua field names read by one line, `test/luacanon/luacanon.go:176`. The alternative remains a test-side field-name table |

---

## 4. Plan drift to record

Steps 1–12 and 14 landed as designed — verified directly: `Parse` signature
exact, `CopyMod`→`Clone()`, `D` internalised, `Combine` gone,
`Externals`→`Resolver`, `ListInternal` returning, `ColType`/`iter.Seq`/
`Script.Build(*Ctx) (schema.Document, error)`, typed template directives
replacing the regex DSL, `tools/` deleted, both `#EVAL` size tags present.
Undocumented drift, all of which should be added to `go-remodel-plan.md` §7.1's
Deviations list whether or not the fix is taken:

1. **Step 2 unfinished** — the third `sanitiseText` (A2). Three documents claim
   one fold table.
2. **Step 13's `SpawnWeight2` never landed.** `export/conquertables.go:155` uses
   `W: r.IntAt(2)`, a magic column index, with a comment explaining that the
   spec names two columns `SpawnWeight` and last-wins name lookup picks the
   wrong one. **Fix:** rename the duplicate in `export/spec.go:103,109,123` to
   `SpawnWeight2` as the plan specified, and use `r.Int("SpawnWeight")`.
   Verify `EXPORT SPEC`.
3. **Step 4 codec drift undocumented.** §6.1 specified `json.Marshaler` on
   `Mod`/`Value`/`Tag`; what landed is `map[string]any` assembly
   (`modparser/modcache.go:121-215`) plus the stringly params codec. Defensible
   at a serialization edge, but it is what produced the bare string value kinds
   (Tier C) and `quantizeTag` (`modparser/modcache.go:389-411`), which passes a
   typed tag through a string-keyed param bag and **drops the `ok`**.
   **Fix:** at minimum, propagate that `ok` — a failed round-trip currently
   silently keeps the unquantized tag.
4. **Error policy half-applied.** `Load` paths return `error`, but
   `SetModCache`/`DecodeMods`/`DecodeMod` panic in 13 places
   (`modparser/modcache.go:58-411`) inside a `data.Load` that returns `error`.
   Acceptable for the embedded artifact, not for `RawSourcesFromDir`.
   **Fix:** give the three entry points `error` returns; keep the internal
   panics as the assertions they are. Of `tree`'s 10 panics only
   `tree/spec.go:367` carries the required "(the Lua errors)" comment — add it
   to the nine in `tree/cluster.go`/`tree/conquer.go`, or reclassify them.
5. **`lua-go-map.md:45`** claims spec-generated row types that do not exist. The
   accessors landed; the generated types did not. **Fix:** correct the row.
6. **`later.md` omits `modparser.Truthy` (`modparser/value.go:215`) and
   `modstore.OutValue.Truthy` (`modstore/output.go:26`)** from its kept-Lua
   inventory, though they are the most load-bearing Lua semantics left in the
   product. **Fix:** add both to §1 with the evidence — `Output` genuinely
   stores `false` at 13+ sites, so `Truthy()` is not `Has()`.

Correctly recorded, therefore not drift: G9 (`WriteEnumFiles`), `RawTag`,
`Actor` staying a string (superseded by B3), `GrantedSkill` plain bools,
`StubHandoff` as a `ReplayInput` field.

---

## 5. Do not touch

Verified correct; changing any of these makes the code worse or breaks parity.

- **`modstore.Output` + `OutValue.Truthy()`** (`modstore/output.go:18-81`). The
  open string key set is load-bearing: `calc` has **220 computed-key reads and
  189 computed-key writes** (`"Base"+damageType+"DamageReductionWhenHit"`,
  `ailment+"ChanceOnCrit"`) against 1,168/897 literal ones. And `Truthy()` is
  not `Has()`: `Output` stores `false` at 13+ sites
  (`calc/ehpguard.go:57,68,74`, `calc/ehppools.go:62,89,91,92`, …), and Lua's
  `if output.X` is false only for nil and false. Three states, honestly encoded.
- **`Tag.Params() []Param` / `ParamValue`** (`modparser/tag.go:81-116`).
  `Param.Value` is a sealed union, not `any`; `Params()` is a per-kind method,
  not reflection; and the sorted reference key names *are* the bytes
  `FormatTag` emits into `ModKey`, a cache key (`tree/stats.go:72`).
- **`export/treegen.go:102-310`**. A JSON-passthrough transformer, documented as
  *"unknown keys must round-trip verbatim"*. Typed decoding would silently drop
  GGG keys.
- **`export/dat.go Row{cells map[int]any}`**. The cell type is unknown until the
  spec is read; the map is unexported and every read goes through a
  spec-checked typed accessor. (Its *query* side is B7.)
- **`export/template_directives.go`**. Sum types with a JSON discriminator and a
  constructor registry — idiomatic Go for polymorphic JSON, not string dispatch.
- **`calc.SkillData`** (`calc/skilldata.go`). The three-map shape is what plan
  §8 decided, not drift. The one real issue is that the twins' `N()` differ —
  `SkillData.N` coerces a numeric string via `modparser.NumOf`, `Output.N` does
  not. Decide which is right and make them agree; do not delete either type.
- **`modparser.NumOf`'s `Str` arm** (`modparser/value.go:203`). It is Lua's
  arithmetic string coercion and it has four live callers. Whether it is dead on
  today's data is unproven; instrument before removing, like B11.

---

## 6. Suggested sequencing

Wave rules from the remodel execution apply: steps with no shared type or
signature impact go in parallel; shared-type steps run alone; the parity suite
runs on the merged tree per wave.

| wave | items | note |
|---|---|---|
| 1 | A2, A3, A4, A6, A8, B7, B8, B10, C-cosmetics | disjoint packages, no shared types |
| 2 | A1, A5, A7 (+ the `modparser` decoder export), B9, drift 2/5/6 | A7 needs the wave-1 tree; A5 touches the canon projection |
| 3 | **B3 alone** | `ActorType` crosses `modparser` → `modstore` → `calc` |
| 4 | **B4 alone** | `SkillTypeID` crosses `modstore` → `calc` → test |
| 5 | **B5 alone** | `item.WeaponData` crosses `item` → `calc` |
| 6 | B11 (measure first), B6, B1 | B11 depends on B5 |
| 7 | B2 (four sub-items, one at a time), drift 3/4 | each regenerates an artifact; `ARTIFACTS` per sub-item |

Every wave ends with `BUILD` + `FULL`; waves touching `export/`, `data/schema`,
the `modparser` codec or `data.StatMapTable` also run `EXPORT`.
