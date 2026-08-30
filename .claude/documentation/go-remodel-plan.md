# Go remodel plan — curing Lua-shaped Go

Audit date 2026-08-28, HEAD `dfee58f04` + the uncommitted working tree
(structured-mods conversion, calc renames, `maxLevel`). Plan only; no code
changed. Filename proposed here (see Open Questions §8.1).

Verdict vocabulary used throughout:

- **Infection** — Lua-ism with no behavioural justification; replace with typed Go.
- **Load-bearing** — encodes reference behaviour the archive differentials see;
  keep the behaviour, re-express behind a typed API.
- **Sanctioned** — test-side Lua understanding at the `.archive` boundary.
- **Generated** — fix the generator / its emitted type design.

## 1. Executive summary

The runtime model is Lua-shaped at its core and the disease is concentrated,
not diffuse. Production Go is 63,099 lines; 13,844 of those are transcribed
or generated tables. The single root is `modparser.Tag = map[string]any`
(`modparser/mod.go:27`) plus `Mod.Value any` / `Mod.Tags []any`
(`mod.go:15,23`): every consumer of a modifier — modstore, item, calc, tree,
data, export — reads it through Lua-semantics helpers. `truthy` is defined
six times (calc, item, modparser, modstore, skills, tree) and called 632
times (calc 535, modstore 76). `%.14g` number formatting is defined four
times, ASCII `string.lower` four times, `sanitiseText` twice with different
behaviour.

Counts of untyped containers in hand-written production code (excluding
`*_gen.go`, `special.go`, `vocab.go`): calc 82 `map[string]any` / 38 `[]any`
/ 180 `any`; modstore 7/12/46; item 19/6/34; tree 36/4/64; export 24/66/162;
skills 4/0/11; data (hand-written) ~50. In generated data: `data/*_gen.go`
carry 5,240 `map[string]any` and 2,777 `[]any` because the two Lua
generators (`tools/gen_skilldata.lua`, `tools/gen_datatables.lua`) had no
type information to emit. `modparser/special.go` (448 KB) carries 2,021
`[]any` because every pattern closure is `func(caps) any`.

Two areas are already at target: `data/schema/` (16 files, zero `any`) and
`export/conquertables.go` + `internal/modcachegen` (typed docs, error
returns). `tree/historic.go` (a C# port, not Lua) is the package's best file.
The package boundary to `test/` holds (`go list -deps` shows no production
package imports `test/...`); the *knowledge* boundary leaks in five places
(§4.7).

Everything is fixable incrementally with the suite green at each step; the
one genuinely risky step is typing `Tag` (§7 step 4) because seven parser
closures emit numeric tag fields as strings and the archive dumps carry
those strings verbatim.

## 2. Baseline

Run on the working tree (HEAD `dfee58f04` + uncommitted changes listed by
`git status`), Go 1.26.5 windows/amd64.

| command | result | wall |
|---|---|---|
| `go test ./...` | all ok (modparser, test, test/luapat, tree) | test 47.1s, tree 6.5s |
| `go test ./test/... ./tree/... ./modparser/... -v` | 27 top-level PASS, 0 FAIL, 2 SKIP | 49.2s |
| `MP_EXPORT=1 go test ./test -run TestExportAgainstReference -v` | PASS — "123 files checked, 0 disagreements" + 4 corruption-negative subtests | 112.3s |

The two skips, both by design and not findings:

- `TestExportAgainstReference` — skipped unless `MP_EXPORT=1`
  (`test/export_test.go:39-40`); run separately above, green.
- `TestProfileRecalc` — profiling harness, skipped unless `MP_PROFILE=<key>`
  (`test/profile_test.go:24-26`).

All other skips in the suite are "archive input not present" guards
(`test/calc_test.go:387,781,1142`, `test/item_test.go:228`,
`test/spec_test.go:211`, `test/skills_test.go:75`, `test/tattoo_test.go:27`,
`test/timeless_test.go:43`, `test/treegen_test.go:24`, `tree/abyss_test.go:35`,
`test/luapairs_calib_test.go:20`); none fired — all 57 fixtures in
`test/testdata` and 97 corpus files are committed, the `.archive` bins and
GGPK extract are present.

No failing test. No `os.Getenv` in production code. Every archive
differential and the commands that exercise it are listed in §7.0.

## 3. Per-package assessment

| package | lines | API shape | severity | headline |
|---|---|---|---|---|
| `modparser` | 12,258 | Lua | **critical** | `Tag = map[string]any` alias, `Value any`, `Tags []any`, `D{Arr,KV}` mixed table, `Parse` returns `[]any`, `fn func(caps) any` for 982 closures, positional-polymorphic `mod()` constructor, four `tonumber` bodies, bare `int64` flags |
| `modstore` | 2,046 | Lua | **critical** | `luaEq`/`luaArith`/`tArith`/`tstr`/`tnum` readers over tag maps, `Combine(...) any` with value-dependent return, `Output map[string]any`, raw mod-type strings, `Externals` global |
| `calc` | 21,414 | mixed | **high** (volume) | `Output`/`SkillData` `map[string]any` (511 + 165 literal keys), 493 `truthy`, 6 closed-key bags held open (`ConfigInput`, `Buff.KV`, `WeaponData`, `Minion.Uses`, `GrantedSkill.Raw`, `Flask Life/Mana any`), `[]any` for 2-variant unions |
| `data` | 8,561 + 6,206 gen | mixed | **high** | `StatMapEntry{Mods []any; KV map[string]any}` × 1,667 entries, `GrantedEffect.Custom map[string]any`, 4 untyped globals (`NodeIDList`, `MapMods`, `FoulbornMap`…), `Levels map[float64]`, reflection `init`, 22 panics no `error`, six `*Canon` exports for tests |
| `item` | 4,020 | mostly Go | medium | `WeaponData/ArmourData/FlaskData/TinctureData/JewelData map[string]any`, `GrantedSkills []map[string]any`, `Affix.Range any`, `Requirements map[string]float64`, `ValueScalar` 0-sentinel beside `*float64` siblings |
| `tree` | 4,329 | mostly Go | medium | `Node.Raw map[string]any` (serves 2 keys), `decode.go` `any`→T shims, `ConqueredBy any`, `sdIdentity any`, string node/conqueror kinds, `jd(it,key)` stringly jewel-data accessor |
| `skills` | 576 | Lua | medium | `GemInstance.KV`/`SocketGroup.KV map[string]any` bags with ~37 enumerable keys; third divergent `sanitiseText` |
| `export` | 8,800 | mixed | medium | `Row.Get(col) any` + 328 assertions + 118 `luaStr` string-column wrappers, `Build func(*Ctx)(any,error)` erasing 21 typed docs, `[]any` `*Row|*otMod` union, template DSL text re-parsed by regex, 12 data-driven panics behind an `error` signature |
| `data/schema` | 721 | Go | none | zero `any`; the model to copy |
| `internal/modcachegen`, `cmd/*` | 374 | Go | low | inherits `[]any` from modparser; `go run` subprocess in `sourceupdate` avoidable |
| `test/luacanon`, `test/luapat`, `test/luarender` | 3,066 | sanctioned | — | correct direction; adapters registered by the test, not production |

## 4. Finding inventory

Each finding: `id` · `file:line` · what · verdict. Ranges cited are from the
survey; line numbers are as of the audited tree.

### 4.1 Untyped table mimicry

| id | site | what | verdict |
|---|---|---|---|
| U1 | `modparser/mod.go:27` | `type Tag = map[string]any` — an *alias*, so `Tag`, `D.KV` and any `map[string]any` are indistinguishable to the compiler; `asTable`/`asTag` (`parse.go:536-544,622-632`) exist to disambiguate at runtime | Infection |
| U2 | `modparser/mod.go:15,23` | `Mod.Value any`, `Mod.Tags []any`; value kinds observed: `bool`, `float64`, `int` (untyped table constants — `any(1) != any(1.0)`), `int64`, `string`, record `Tag`, `[]any` (transient), `nil`, `+Inf` | Infection |
| U3 | `modparser/mod.go:32-60` | `D{Arr []any; KV map[string]any}` built by `d(items ...any)` + `p(k,v)` with a runtime `it.(pair)` check; `D.KV` control keys (`tag`, `tagList`, `flags`, `addToMinion`, `modSuffix`, … ~30) copied wholesale into `misc map[string]any` at `parse.go:382,407-409` | Infection |
| U4 | `modparser/mod.go:74-101` | `mod(name, typ, value, rest ...any)` decides argument *meaning* by argument *type* (`rest[0].(string)` = source, `asInt64(rest[1])` = flags); trailing-nil stripping reproduces Lua constructor holes | Load-bearing semantics, Infection encoding |
| U5 | `modparser/mod.go:325`, `tables_build.go:112` | `type fn func(c caps) any` for every closure; returns `[]any`, `Tag`, or `nil` — three contracts, one signature, disambiguated by assertion at `parse.go:157,177,189,193,199,259,336` | Infection |
| U6 | `modparser/parse.go:81` | `Parse(line) ([]any, string)`; non-`*Mod` elements reachable via `asModList` default branch (`parse.go:645`); `nil` vs `[]any{}` signals unrecognised vs partial (`parse.go:104,151,213,375`) | Infection |
| U7 | `modparser/modcache.go:111-164,190-238` | hand-rolled discriminated union over `map[string]any` keyed by `"_"`; anything without `"_"` decodes to `Tag` (`:229-234`); unchecked assertions | Infection |
| U8 | `modparser/globals.go:8-162` | `ModFlag`/`KeywordFlag`/`SkillType` are anonymous struct values with `int64` fields, not named types — `Mod.Flags`/`KeywordFlags` are bare `int64`; comment `:3-5` says kept "so hand-ported parser code reads exactly like the reference" | Infection |
| U9 | `modparser/special.go` (448 KB), `tags.go`, `names.go`, `preflags.go`, `modflags.go`, `jewels.go` | transcribed pattern tables: `map[string]any` pattern → `fn` / `[]any` / `Tag` / `*D` / `string`; 3,009 `any` tokens in `special.go` | Generated (transcription; blast radius of U1-U5) |
| U10 | `modstore/eval.go:56` | `Actor.Output map[string]any`; `getStat` (`eval.go:199-219`) needs absent / present-falsey / numeric (Lua `output[stat] or cfg.skillStats[stat] or 0`) — three states, not "any type" | Load-bearing semantics, Infection type |
| U11 | `modstore/eval.go:58-59`, `:115,119,124`, `:45` | `WeaponData1/2 map[string]any` (read only for `countsAsAll1H` and `"Added"+v` *presence*, `eval.go:497-511`), `Cfg.SkillPart any`, `Cfg.SkillGem any`, `Cfg.SkillStats map[string]any` (aliases calc's output map), `TabEntry.Value any` | Infection |
| U12 | `modstore/store.go:57` | `Conditions map[string]any` — inhabited set is `bool \| string` (Forbidden-jewel class names tested for truthiness only) | Load-bearing semantics, Infection type |
| U13 | `modstore/eval.go:143-163` | `tlist`/`asList` accept `[]any`, `*D` array part, and an empty `Tag` as "truthy empty list" | Infection |
| U14 | `calc/performutil.go:22`, `calc/offencebody.go:20,23,33-34`, `calc/mirages.go:24-25`, `calc/skillmods.go:39` | the stat `output map[string]any`: 511 distinct literal keys, 920 literal sites, plus 161 composite-key sites (`output[damageType+recoupType+"Recoup"]`, `calc/defencetail.go:56`; `calc/ehp.go:350`); key set and nil-vs-absent are byte-compared by `scalarsOnly` (`test/calc_test.go:463-472`) at six checkpoints; nested sub-tables `output["MainHand"]` (`calc/globalcache.go:55`) are *dropped* by the diff | Load-bearing key-set; Infection value type |
| U15 | `calc/activeskill.go:20` | `SkillData map[string]any`: 165 literal keys, open at runtime — filled from `SkillData` LIST mods `activeSkill.SkillData[str(tag["key"])] = tag["value"]` (`calc/skillmods.go:745-751`) and `SkillData[name+"ReservedBase"]` (`calc/perform.go:602,610`) | Load-bearing key-set; Infection value type |
| U16 | `calc/setup.go:152`, `calc/input.go:27-28` | `ConfigInput map[string]any` — 26 distinct keys, values coerced by `tonum` (`calc/ehp.go:17-33`, "strings are parsed") | Infection |
| U17 | `calc/skillmods.go:46` | `Buff.KV map[string]any` — closed 7-key set (`type name activeSkillBuff applyNotPlayer applyMinions applyAllies allowTotemBuff`), read via `bkv` (`calc/performbuffs.go:87`), written via `assignKV`/`luaOr` (`calc/skillmods.go:998-1001`) | Infection |
| U18 | `calc/skillmods.go:33-34`, `calc/skills.go:55-57`, `item/item.go:287` | `WeaponData map[string]any` — 15 fixed keys (`type subType name range rangeBonus AttackRate CritChance countsAsAll1H countsAsDualWielding AddedUsing{Axe,Claw,Dagger,Mace,Sword}`) + data-driven LIST extension (`item/buildmodlist.go:278-281`); read via `wdNum` (`calc/offenceskilldata.go:31`), copied by `copyWeaponData` (`calc/offenceconv.go:149`) | Infection (fixed part), Load-bearing (open extension) |
| U19 | `calc/skillmods.go:30` | `Minion.Uses map[string]any` — only ever `truthy(minion.Uses[slot])` (`calc/performminion.go:124`, `calc/skillmods.go:907,917`): it is a set | Infection |
| U20 | `calc/setup.go:196`, `calc/input.go:53` | `ExplodeSources []any` holding `*Item \| *NodeInput`, switched at `calc/skillfuncs.go:111-121` and `calc/skills.go:246-255` (both with panics) | Infection |
| U21 | `calc/setup.go:247` | `Env.CurseSlots []any` — every reader assumes `curseEntry` (`calc/performbuffs.go:850,867,874`) | Infection |
| U22 | `calc/activeskill.go:28`, `calc/mirages.go:21` | `SkillPart any` — always a number or nil (`anyNum(activeSkill.SkillPart)`, `calc/skillfuncs.go:130`) | Infection |
| U23 | `calc/setup.go:30` | `GrantedSkill.Raw map[string]any` — two keys read (`triggered`, `triggerChance`, `calc/skills.go:217-218`) | Infection |
| U24 | `calc/input.go:114-116` | `FlaskBaseInput.Life/Mana any` — comment: "numbers whose presence the calc reads as truthiness" | Infection |
| U25 | `calc/radius.go:24,47,52`, `modparser/jewels.go:19-37` | `RadiusJewel.Attributes map[string]any`; `JewelStoreWriter.Sum(typ, cfg any, …)` / `AddList(list any)` / `JewelNodeFn(node any, …)` — `any` params written to mirror modstore's signature and dodge an import cycle. **The cycle is fake**: every jewel closure calls `out.Sum("BASE", nil, stat)` (`jewels.go:110,125,131`) — no caller supplies a `Cfg`; `node` is always a `JewelNodeRef`; `AddList`'s `any` is the `[]any` parse shape (dies in step 4) | Infection — typed interface stays in `modparser` with no `modstore` reference |
| U26 | `calc/activeskill.go:220-256` | `var rejected []any` with explicit nil holes to reproduce `ipairs` stopping at the first hole; element is always `*ActiveEffect` | Load-bearing semantics; `[]*ActiveEffect` with nils preserves it |
| U27 | `data/skills.go:20-23` | `StatMapEntry{Mods []any; KV map[string]any}` × 717 global + ~950 per-skill entries; KV key set closed and counted: `div` 216, `value` 17, `cost` 16, `mult` 15, `cooldown` 13, `baseMultiplier` 12, `base` 9, `statInterpolation` 4, `levelRequirement` 4, `skillFlag` 2 (+3 mod-shaped one-offs) | Generated (fix `tools/gen_skilldata.lua:143-173`) |
| U28 | `data/skills.go:113` | `GrantedEffect.Custom map[string]any` — 16 keys, closed: `fromItem` 177, `parts` 159, `preDamageFunc` 50, `minionList` 49, `legacy` 36, `fromTree` 27, `baseFlags` 11, `addFlags` 10, `minionHasItemSet` 5, `levels` 5, `initialFunc` 3, `baseEffectiveness` 3, `preSkillTypeFunc` 2, `addMinionList` 2, `explosiveArrowFunc` 1, `baseMods` 1 | Generated (fix `gen_skilldata.lua:342-356`) |
| U29 | `data/skills.go:74-76,94-95` | `RequireSkillTypes/AddSkillTypes/ExcludeSkillTypes []any` (nil holes for unknown `SkillType.X`), `QualityStats/ConstantStats [][]any` (2-tuples `(string,float64)`) | Infection |
| U30 | `data/data.go:110-117,158` | `NodeIDList`, `AbyssNotableNames`, `TimelessJewelTradeIDs`, `TimelessJewelLUTs`, `MapMods`, `FoulbornMap` as `map[string]any`; `NodeIDList` mixes numeric-string node keys with 3 metadata keys, filtered by `strconv.ParseInt` success (`data.go:213-223`); `FoulbornMap` is JSON-decoded into `map[string]any` (`data.go:320-323`) then re-derived with two panics at `item/tail.go:17-35` | Generated (`gen_datatables.lua`) / Infection (`FoulbornMap`) |
| U31 | `data/minions.go:37`, `data/bossskills.go:17-18,28-29`, `data/bases.go:81`, `data/tables.go:165` | `Hostile any` (`true\|false\|float64`, `minions.go:73-86`), `DamagePenetrations map[string]any` (`float64` or sentinel `""`/`"flag"`), `ValidBases []any` (single concrete element type), `Transform func(any) any` (two implementations) | Infection |
| U32 | `data/skills.go:100` | `Levels map[float64]*SkillLevel` — float-keyed because Lua array indices are numbers; `schema.SkillLevel.Level` is already `int64` (`data/schema/skills.go:78`) | Infection |
| U33 | `item/item.go:286-291` | `GrantedSkills []map[string]any` (6 known keys, `buildmodlist.go:700-723`), `ArmourData/FlaskData/TinctureData/JewelData map[string]any`; `JewelData` read by 20+ literal keys through `tree.jd(it,key)` (`tree/spec.go:442`) | Infection (fixed part), Load-bearing (LIST extension) |
| U34 | `item/item.go:118` | `Affix.Range any // number, list of numbers, or nil`; switch at `buildraw.go:81-96` handles `float64`, `[]any` *and* `[]float64` because two producers disagree (`parseraw.go:496`) | Infection |
| U35 | `item/item.go:222`, `parseraw.go:1119` | `Requirements map[string]float64` with closed keys (`str dex int level strMod dexMod intMod`); `setReq` exists to emulate assign-nil-deletes | Infection |
| U36 | `item/item.go:90`, `buildmodlist.go:571-574` | `ModLine.ValueScalar float64 // 0 = unset (Lua nil)` beside `*float64` siblings; fix-up code compensates | Infection |
| U37 | `skills/skills.go:63-74` | `GemInstance.KV`, `SocketGroup.KV map[string]any` — header `:8-9` "scalar bags … the same shape the reference's Lua tables have"; writes are enumerable (`loadSkill` 9 keys `:176-199`, `loadGem` ~20 `:228-288`, `ProcessSocketGroup` 8 `:357-439`); unchecked `kv["nameSpec"].(string)` `:267`, `kv["level"].(float64)` `:431` | Infection (the "nil" string quirk is an XML-value fact, `*string` holding `"nil"` reproduces it) |
| U38 | `tree/tree.go:92` | `Node.Raw map[string]any` retained for every node to serve **two** keys: `expansionJewel.{size,index,proxy,parent}` (`cluster.go:26`, `spec.go:161`, `specdeps.go:322,331`) and `activeEffectImage` (`spec.go:227`); `"expansionSkill"` is written (`cluster.go:381,546,589`) and read nowhere; `tattoo.go:36` re-boxes a typed field back into `Raw` | Infection |
| U39 | `tree/decode.go` (67 lines) | `canonArray`/`str`/`num`/`numPtr`/`boolean`/`strList`/`idList` `any`→T shims; `numPtr` has no caller; `idList` panics on non string/float64 (`:59`); 9 of tree's 17 `.(map[` assertions are in `tree.go` decode (`:207-433`) though `tattoo.go:20` and `historic.go:460` already unmarshal into typed structs | Infection |
| U40 | `tree/spec.go:28,58` | `SpecNode.ConqueredBy any` (item `JewelData["conqueredBy"]` map or the synthesized map at `specdeps.go:32`); `sdIdentity any` (compares `*Node` \| `*MasteryEffect` \| nil) | Infection |
| U41 | `tree/specdeps.go:14-40`, `conquer.go:161` | `collectAbyssConquests` packs typed `[]AbyssComponent` into `map[string]any{"id","conqueror","modification"}` which `applyConquered` unpacks with `conqueredByMap(cq)["modification"].([]AbyssComponent)` | Infection |
| U42 | `tree/historic.go:459-529` | `buildAbyssWorld` re-decodes `tree_3_29.json` into `map[string]map[string]any` — a second parse of data `*Tree` holds — because `Node` lacks `IsJustIcon`/`IsMultipleChoice`/`IsAscendancyStart` | Infection |
| U43 | `export/dat.go:47-51,220-242,267-344` | `Row.Get(col) any` though `spec_gen.go` declares every column's type; 328 assertions across `export/` (`.(int64)` 132, `.(*Row)` 75, `.([]any)` 48, `.(bool)` 25, `.(float64)` 25, `.(Interval)` 13, `.(string)` 10); e.g. 15 in 38 lines at `script_minions.go:161-198` | Infection |
| U44 | `export/script.go:34-44`, `write.go:26-30`, `script.go:27` | `Script.Build func(*Ctx) (any, error)` — all 21 builders return a concrete `schema.*` doc, erased at `return`, marshalled immediately; `Ctx.modsDoc any` | Infection |
| U45 | `export/script_minions.go:37-73,226-258` | `getOTStats(…, modList []any) []any` — `*Row \| *otMod` union already de-ducked into `otMod` but kept in `[]any` with a type switch | Infection |
| U46 | `export/script.go:89-108` | `luaStr(v any) string` — 183 calls, 118 of them `luaStr(row.Get("<String column>"))`, i.e. `v.(string)` spelled as `tostring`; panics at `:107` | Infection |
| U47 | `export/templates.go:21-25,49` | `templateDirective{Line, Name, Mods}` sum-type-by-sentinel, discriminated by sniffing `e[0] == '"'`; template payload is the Lua exporter's text DSL inside JSON strings (`"#monster … "`, `"#baseMatch …"`, `"#forceShow true"`), re-parsed by `reDirective` (`script.go:136`) + `strings.Fields` (`script_minions.go:130`) | Infection |
| U48 | `export/treegen.go:106,134,143,156-157` | `BuildTreeDoc` walks GGG JSON as `map[string]any` — correct (unknown keys must round-trip verbatim) — but `fixTree`'s unchecked assertions panic on GGG drift though it returns `error` (`:150,164`) | Load-bearing DOM; Infection in `fixTree` |

### 4.2 Lua semantics reimplemented

| id | site | what | verdict |
|---|---|---|---|
| S1 | `calc/tools.go:353`, `item/buildmodlist.go:19`, `modparser/helpers.go:202`, `modstore/store.go:96`, `skills/skills.go:448`, `tree/spec.go:451` | `truthy(v any)` six times; 632 calls. ~145 calc calls are `truthy(map[key])` (present-and-not-false, unavoidable while U14/U15 stand); ~350 are on locals of known type | Infection (each disappears with the typed container it reads) |
| S2 | `modstore/eval.go:784-792` | `luaEq(a, b any)` — `==` fallback on two `any` values **panics** on uncomparable types; `tag["skillPart"]` can hold a `Tag` map → Lua would return false | Infection **with a latent defect** |
| S3 | `modstore/eval.go:994-1028` | `luaArith` string→number coercion (Lua arithmetic metamethod) + `tArith`/`arithNum` panics "the Lua errors"; **is exercised by the corpus**: `data/raw/modcache.jsonl` carries `"div":"5"`, `"div":"4"`, `"div":"2"`, `"limit":"30"`, six `"percent":"1"`; the archive dump `test/testdata/parse_archive.jsonl` has e.g. `{"div":"5","limit":-30,…}` for "5% less Damage taken per 5 Rage…"; sources are seven closures passing `c.s(n)` where the reference passed the raw capture: `modparser/special.go:108,266,422,1086,3150,3153,3162` (`div`), `:493` (`limit`), `:1683,1686` (`percent`) | Load-bearing today; becomes Infection once numbers are parsed at parse time and the archive side is normalized in the test (§6.1, §7 step 4) |
| S4 | `modstore/eval.go:130-141,259,350` | three coercion policies for one map (`tstr` non-string→`""`, `tnum` numbers only, `tArith` string-coercing); `tag["div"] = getMultiplier(…)` **written back into the shared tag** — `#EVAL`-documented, visible to later evaluations | Load-bearing (write-back); Infection (three readers) |
| S5 | `calc/tools.go:361-374` | `anyNum(v any)` — `nil→0` is the Lua `or 0` idiom; `int64`/`int` arms exist only because three producers emit three Go number types | Load-bearing (`nil→0`), Infection (number-kind sprawl) |
| S6 | `calc/skillmods.go:1044-1067` | `setKV` (skip nil), `assignKV` (delete on nil), `luaOr` (`a or b`) — 8 sites; `assignKV` is the nil-vs-absent distinction the byte-diff detects | Load-bearing semantics; `luaOr` Infection (3 calls on `Buff.KV` bools) |
| S7 | `calc/mirages.go:224`, `item/tools.go:44`, `tree/conquer.go:78`, `data/luanum.go:22-27`, `export/script.go:71`, `modparser/modtools.go:157-182`, `modparser/canon.go:191`, `modparser/modcache.go:242` | `%.14g` `tostring(number)` defined 8 times; reaches artifacts/fixtures (mirage `InfoMessage`, conquered stat text substitution, unique item blobs, mod cache quantization) | Load-bearing behaviour; Infection duplication |
| S8 | `calc/tools.go:334`, `data/gems.go:193`, `item/tools.go:52`, `tree/tree.go:596`, `modparser/mod.go:130` | ASCII-only `string.lower` ×5 | Infection — **decided 2026-08-29**: all five deleted; `strings.ToLower` / `(?i)` regex (see ~~L2~~) |
| S9 | `item/tools.go:133-157` vs `skills/skills.go:308-350` | two `sanitiseText` with different fold sets (seven hyphens + umlauts + high-byte `?` vs three hyphens) | Infection — **decided**: one `item.FoldText`, extended to uppercase accents (Ö→O, Ä→A) on PoB's own no-two-names-differ-by-an-accent assumption; skills calls it |
| S10 | `modparser/mod.go:257`, `helpers.go:214`, `mod.go:338`, `modstore/eval.go:994` | four `tonumber` bodies; `strconv.ParseFloat` accepts `1e5`/`Inf`/hex-float and rejects Lua `0x1f` | Infection (one body; document divergence) |
| S11 | `calc/tools.go:376-407`, `calc/offenceconv.go:149`, `calc/performbuffs.go:1542`, `modparser/modtools.go:373-407`, `data/skills.go:658-694` | deep-copy helpers over `any` trees (`copyTagValue`, `copyWeaponData`, `copyModForTagWrite`, `CopyMod`/`copyAny`, `deepCopyAny`); `parse.go:613-617 copyModList` is *shallow* while `Common.lua:418 copyTable` recurses — masked by `Parse`'s deep copy, exposed on the `parse.go:164` path | Load-bearing (copies precede mutation); Infection (reflection-shaped switches; typed `Clone()` methods replace them) |
| S12 | `calc/setup.go:164-172`, `calc/tools.go:409-419` | `AllocOrders/ExtraOrders` — `pairs()` order captured from the reference dump and replayed; panics at `calc/setup.go:517,542,556,561` when misaligned | Load-bearing (float sums are not associative; LuaJIT hash order not derivable) — do not touch |
| S13 | `modparser/modtools.go:114` vs `:129-138` | `len(m.Tags)` (`#mod`) vs `tagArrayLen` (`ipairs` stops at first nil) — two lengths for one slice, both used by `CompareModParams` | Load-bearing; typed `[]Tag` without holes must carry the hole count |
| S14 | `modparser/modtools.go:38-43` | `ParseTags` writes `nil` (not 0) for a non-numeric `threshold` | Load-bearing |
| S15 | `data/luanum.go:36-67`, 12 `luaUnescape` sites (`bases.go:184,189`, `cluster.go:69`, `data.go:253`, `gems.go:39-52`, `items.go:58-229`, `skills.go:698-790`), `bossskills.go:75` | `luaUnescape`/`luaStringLiteral` exist because export documents ship **Lua source fragments** (`schema.SkillHeader.Description` "final escaped text", `schema.BossSkill.Tooltip` a quoted Lua literal) | Infection at the exporter (emit decoded values) |
| S16 | `data/skills.go:592-600`, `:465-520` | 1-based `t_insert` emulation (`for i := 1; ; i++ { key := itoa(i) …}`), `LazyStatMapCopy` porting a `__index` metatable, `anyList` "nil stays nil" | Infection |
| S17 | `tree/spec.go:182-214` | `resetToSource`/`EffectiveName`/`replaceNode` — nil-unshadowing (`*string`), table-identity short-circuit (`sdIdentity == any(src)`), shallow-shared `Sd` with deep-copied `ModList`; `stats.go:55` splices `Sd` in place → latent aliasing | Load-bearing (`test/spec_test.go:78` compares `Name *string`; the identity short-circuit prevents the 4-pass rebuild wiping conquered mods); Infection encoding |
| S18 | `tree/specdeps.go:137-143` | `if m[k]==0 { prev := m[k]; m[k] = prev+1; count++ } else { m[k]++ }` — Lua `if not t[k]` transliterated with a vestigial read | Infection (cosmetic) |
| S19 | `tree/conquer.go:78-83,86`, 1-based `conqueredNode1`, `+1-337` offset (`:168,219,283,313`), `desc.Index-1`, `NodesInRadius[radiusIndex-1]` | `luaNumStr`/`luaRound`; indices that mirror the reference's LUT layout and serialized ids | Load-bearing |
| S20 | `export/common.go:39-101`, `dat.go:274-305`, `treegen.go:212-264` | `bytesToFloat` denormal/exponent-128 bug-compat, UTF-16 pending-surrogate bug-compat, `-1337/1337/"<bad offset>"` sentinels, `numKey`/`luaString` | Load-bearing — do not touch |
| S21 | `export/dat.go:49,116-135,339`, `conquertables.go:74-80,98` | `Row.Index` 1-based "matching the Lua rowIndex"; every consumer subtracts 1; only the ref-resolve `+1` at `:339` is format-mandated | Infection |
| S22 | `modparser/mod.go:189-193,289-314` | `scan`'s `literalWeight` tie-break — a port invention replacing Lua-pattern-length ordering | Load-bearing (documented divergence) |
| S23 | `skills/skills.go:488-496` | `luaLevelsLen` (`#` over a hash table) — needed only because of U32 | Infection (becomes `len` on a slice / `maxLevel` on an int map) |

### 4.3 Lua data shapes as inputs/outputs

| id | site | what | verdict |
|---|---|---|---|
| I1 | `modstore/store.go:275-316`, `db.go:197,217`, `list.go:148,168` | `Combine(modType string, …) any` returns `float64\|bool\|any\|[]any` depending on a string argument; `Override(…) any`; `List(…) []any`; `ListInternal(result *[]any, …)` out-parameter — the untyped return has propagated `truthy`/`anyNum` wrappers into 48 calc files | Infection, high leverage |
| I2 | `modstore/eval.go:18-21` | `var Externals struct{ GemIsType func(gem any, …); GetGameIdFromGemName … }` — mutable global function registry standing in for Lua `calcLib` | Infection |
| I3 | `calc/offencetypestats.go:411-419` | `rampTable([][2]float64) []any` — typed input deliberately downgraded to `{{x,y},…}` nested `[]any` "the shape the modifier evaluator reads" | Infection |
| I4 | `calc/performbuffs.go:1559-1569` | `getCachedOutputValue(…keys string) []any` positional heterogeneous return, unpacked by `anyNum` | Infection |
| I5 | `calc/globalcache.go:38-76` | `out/outputSub/mainSkillData/activeSkillData` get-by-string-path into a live env; `outputSub` asserts nested `map[string]any` | Load-bearing key access; nested tables are not diffed → liftable |
| I6 | `calc/ehpguard.go:36`, `offence.go:68,118`, `offenceselfhit.go:33`, `offencetypestats.go:18,37`, `performminion.go:16,248`, `skillfuncs.go:312`, `offencecosts.go:274` | ~10 functions taking `output map[string]any` / `skillData map[string]any` parameters | consequence of U14/U15 |
| I7 | `data/raw.go:62`, `skills.go:844,952,964,982`, `gems.go:206`, `minions.go:162` | `RawDoc(name string, out any)`; six exported `*Canon(...) map[string]any` shadow builders whose only caller is `test/gamedata_test.go:54-70` | Infection (test knowledge in production; §4.7) |
| I8 | `tree/spec.go:442-449` | `jd(it *item.Item, key string) any` / `jdTrue` — stringly accessor over `item.JewelData` with 20+ literal keys | Infection (rooted in U33) |
| I9 | `tree/historic.go:335`, `conquer.go:218-227` | `TimelessPassive(…) []int` positional return, dispatched on `headerSize == 2\|3\|6\|8` | Load-bearing shape (cell-for-cell compared) — name the header sizes |
| I10 | `export/script_minions.go:79-98`, `script_skills.go:519-529`, `script_bases.go` | `WalkTemplate(name, inDir, directives map[string]func(string), modDirectives map[string]func([]byte))` — dictionary-dispatched closures mutating captured state (`state`, `defs` re-pointed at `:307`, `spectre` calling `monster` then `emit`) | Infection |
| I11 | `data/schema/minions.go:13-15`, `export/script_minions.go:156-159` | `MinionDef{Skip: true}` placeholder keeps emit-sequence alignment where the Lua prints "Invalid Variety" | Load-bearing; document as parity artifact |

### 4.4 Dynamic dispatch on strings

| id | site | what | verdict |
|---|---|---|---|
| D1 | `modstore/eval.go:230` (20 cases), `store.go:276-292,344,363,422`, `db.go:118-275`, `list.go:52`, `item/buildmodlist.go:53-117,160`, `tail.go:86-97` | `switch tstr(tag,"type")` with a silent no-op default; mod-type literals `"BASE" "INC" "MORE" "FLAG" "OVERRIDE" "LIST" "MAX" "MIN"` everywhere, no constants — a typo is a silent zero | Infection |
| D2 | `modparser/parse.go:265-372` | 17-arm `switch modForm` over form-code strings from `forms.go` | Infection (closed enum) |
| D3 | `modparser/special_hand.go:11-30` | `conquerorList` records `{"type":"vaal", "id": int\|string}` stored as `Tag` — not a tag; also `flagTypes` records `{"type":"BASE",…}` (`parse.go:336-339`) share the `Tag` type | Infection (three record kinds in one type) |
| D4 | `tree/tree.go:447-492`, `conqueredpassives.go:105-115`, `tattoo.go:47-53`, `cluster.go:373-581`, ~30 comparison sites | `Node.Type string` — seven closed values; `conquerorType(cq) string` → `jewelTypeByConqueror map[string]int` (`conquer.go:17-29`) with magic fallback `5` (`:159`), thresholds `>= 7`, `== 1`, `== 11` (`:160,215`, `specdeps.go:23,28,82`) and a *second* string switch on the same datum (`conquer.go:306-348`); magic pool indices `77 78 91 110` as comments (`conquer.go:240-347`); `AbyssComponent.Type int` 1\|2 | Infection |
| D5 | `calc/skillfuncs.go:20-56`, `offencebody.go:215` | `skillFuncs map[string]skillFunc` keyed `id+":"+callback` — composite string key | Infection (mild); registry itself Sanctioned |
| D6 | `calc/skillmods.go:53-61,1070-1078` | `modFlagByName`/`keywordFlagByName` built by **reflection** over `modparser.ModFlag`; `data/skills.go:130-144` `modConstants` `init()` reflection over the same structs because `schema.SkillHeader.SkillTypes` ships `"SkillType.X"` strings (`data/schema/skills.go:39-49`) | Infection (emit numbers, or a generated name→bit map in modparser) |
| D7 | `calc/triggerconfig.go:23-26`, `triggers.go:283-302`; `calc/setup.go:104-112`; `calc/mirages.go:47-60` | trigger config table keyed by lowercased item/skill names with a five-key fallback order; influence-multiplier map; mirage variant selection on `SkillData` keys | Load-bearing / Sanctioned (typed values; key strings are the reference's own) |
| D8 | `export/script.go:44,47`, `dat.go:375-381`, `write.go:38-44` | `Scripts` registry by name (Sanctioned); `x.Dat("MonsterVarieties")` + `Get("Column")` two levels of stringly access (Infection with U43); `want["conquerTables"]` camelCase selection key vs lowercased names | mixed |
| D9 | `export/dat.go:59-73,267-327` | `Col.Type string` switched on `"Bool"/"Int"/…` with two `panic("unknown dat column type")` | Infection (iota enum makes them unreachable) |

### 4.5 Naming and layout from Lua

| id | site | what | verdict |
|---|---|---|---|
| N1 | every package | file headers cite Lua files and line ranges (`calc/defenceprimary.go:1` "CalcDefence.lua L826-1115"; `item/tail.go` "Item.lua L1337-1606"; `export/script_*.go` ↔ `Scripts/*.lua`; `modparser/*.go` ↔ ModParser.lua tables) | Sanctioned while the archive is the contract — this is what makes the differentials reviewable |
| N2 | `calc/performbuffs.go:88` (1,033-line function), `calc/skillmods.go` `buildActiveSkillModList` (812 lines) | single functions transliterated from Lua bodies | Infection — **out of scope for this plan** (no reformatting passes; §8.9) |
| N3 | `lua*` identifiers: `luaEq luaArith luaTrim luaNumStr luaLower luaLevelsLen luaOr luaTostring luaTostringAny luaStr luaNum luaTonumber luaUnescape luaStringLiteral luaIntString luaNumString luaRound luaString`; `m_huge` (`modparser/helpers.go:199`), `mathHuge` (`calc/skillfuncs.go:107`), `cap1`, `tonumber`, `d`/`p`, `firstToUpper`, `Stats.Sd`/`SpecNode.Dn`/`Node.SortedStats`/`Group.Orbits // oo`, `conqueredNodeDoc.{Ks,Not,M}`, `NodeMod.Nil` | Infection for helpers that die with typing (`luaOr luaEq luaArith luaStr luaTostringAny luaLevelsLen luaTonumber`); Sanctioned for the survivors that *specify* Lua semantics (`%.14g`, ASCII lower, `luaTrim`, `luaString`) once consolidated (§6.9) |
| N4 | `tree/conquer.go:323` `str := "4"` shadows package func `str` (`decode.go:16`); `tree/spec.go:258` `close :=` shadows builtin; `tree/specdeps.go:578` `sortedStringKeys2` duplicates the generic | Infection (cosmetic) |
| N5 | `export/spec_gen.go:1-3` "Transformed one-time … maintained here in Go" but named `_gen`; `data/uniques_generated*.go` are hand-ported despite the name; `modparser/special.go` has no "generated" header though it is a transcription | Infection (naming misleads about regeneration) |
| N6 | `data/data.go:53-177` | 124-line `var (…)` block of 70+ mutable package globals — the Lua global `data` table; read-only by convention (`:49-52`) | Infection (§8.6) |

### 4.6 Missing Go idiom

| id | site | what | verdict |
|---|---|---|---|
| G1 | `tree.Load(version) *Tree` (`tree/tree.go:195`), `data.Load(src Sources)` (`data/data.go:226`), `data.RawDoc` (`raw.go:62`), `modparser.Parse`, `modstore`, `item`, `skills` | **no function in calc, data, item, modparser, modstore, skills, tree returns `error`**; 22 panics in hand-written `data/` (`bossskills.go:46,61`, `data.go:322,395`, `gems.go:74,110,176`, `luanum.go:39,67`, `minions.go:82`, `raw.go:19,22,49,54`, `skills.go:160-812`, `uniques_generated3.go:82`) | Infection for load/decode paths; Sanctioned for reference-error mirrors and unported-branch guards |
| G2 | panics that are **shape assertions over `any`** — `calc/skillmods.go:176,182,187,194`, `performminion.go:88`, `radius.go:67`, `skills.go:255`, `skillfuncs.go:120`; `item/buildmodlist.go:29,158`, `tail.go:24,30`; `tree/decode.go:59`, `tree.go:294`, `cluster.go:36`, `stats.go:21`; `modparser/modcache.go:57,93,171,185`; `export/script.go:107` | each is unreachable once the container is typed | Infection |
| G3 | `export/` 12 data-driven panics behind `Build(...) (any, error)`: `script_bossdata.go:206,364,368,479`, `script_minions.go:66`, `script_miscdata.go:25`, `script_umodstotext.go:49`, `statdesc.go:343,525`, `treegen.go:233,251`, `dat.go:378` | abort a 97s build on GGG drift through a signature that already carries `error` | Infection |
| G4 | `modstore/store.go:115` `panic("modstore: number expected")` | the one of nine modstore panics without a "(the Lua errors)" comment; `parity.md:163` claims all nine are error parity | doc gap — verify and annotate |
| G5 | `export/dat.go:134-140` `Rows(yield func(*Row) bool)` | range-over-func shape, 0 range callers, 27 callback callers, `var buildErr error` smuggling (`conquertables.go:119-168`) | Infection (`iter.Seq[*Row]`) |
| G6 | `data/raw.go:13` `//go:embed raw` | embeds all 26 artifacts (~46 MB); `skillgemlist.json`, `mapuniquetofoulborn.json` have no reader; `statdesc.json` (8.1 MB) read only by `test/luarender/statdesc.go:14` | Infection (§8.7) |
| G7 | `data/skills.go:130-144` | `init()` + `reflect` to build `modConstants` | Infection (see D6) |
| G8 | `cmd/sourceupdate/main.go:130-139` | re-execs `go run ./cmd/sourceupdate -modcache-only` because embedded `data/raw` is stale in-process; `data.Load(data.RawSources())` (`:64`) already takes sources as a parameter | Infection (add `RawSourcesFromDir`) |
| G9 | `export/enums.go` `WriteEnumFiles` | writes two synthetic `.datc64` files into the read-only GGPK extract as a side effect of loading | Load-bearing behaviour (enums.lua does the same); Infection location (inject into the in-memory `DatSet`) |
| G10 | `calc/radius.go:31-46`, `calc/setup.go:41-46` | 16 exported methods without doc comments (interface satisfaction); otherwise doc coverage across the repo is unusually good | minor |

### 4.7 Test-boundary leaks (Lua-shape knowledge resident in production for the tests' sake)

The package boundary holds — `go list -deps` over every production package
contains no `missingPassives/test` import; `test/luacanon` and
`test/luapat` import no production package; `test/luarender` imports only
`data/schema` and `modparser`; the single `luacanon.RegisterAdapter` call
is in `test/gamedata_test.go:54`. Knowledge leaks:

| id | site | what |
|---|---|---|
| B1 | `data/minions.go:162`, `data/skills.go:844,952,964,982`, `data/gems.go:206` | six exported `*Canon` plain-table shadow builders; only caller `test/gamedata_test.go:57-69` |
| B2 | `modparser/canon.go` (221 lines, imports `reflect`) | `modparser.Canon` — zero production callers (`scan_test.go`, `test/{parse,modtables,modtools,modstore}_test.go`); duplicates `luacanon` for `*Mod`/`*D` |
| B3 | ~272 `lua:"…"` struct tags on production types (`data/tables.go` 92, `bases.go` 57, `items.go` 35, `minions.go` 24, `cluster.go` 21, `bossskills.go` 15, `mods.go` 15, `data.go` 13; `calc/input.go` 60+, `calc/globalcache.go` 12) incl. the `lua:"@array"` sentinel (`data/mods.go:16`, `items.go:71,117`) | read by exactly one line in the repo: `test/luacanon/luacanon.go:154` |
| B4 | `data/uniques_treedep.go:27` `TrimTreeDependentUniques` | rewinds production state to the dump point; sole caller `test/gamedata_test.go:89` |
| B5 | `calc/setup.go:177-181` `Env.StubHandoff` | branched on in production (`calc/buildoutput.go:55-56`, `calc/mirages.go:99`), written only by `test/calc_test.go:918` |

Not leaks (checked): `modparser.CopyMod` (18 production callers),
`EncodeMods`/`Quantize14` (`internal/modcachegen`, `export/script_minions.go:290`),
`data.UnportedFn` (`calc/offencebody.go:219`), `calc.ReplayInput` (a parameter
of production `InitEnv`/`BuildOutput`), `data/luanum.go` (unexported).

`tools/*.lua`: eight dump scripts whose outputs are all committed (`git
ls-files test/testdata` = 57, `tools/dump_*.lua:1-3` headers name each
target); two one-time generators (`gen_datatables.lua`, `gen_skilldata.lua`)
whose emitted `.go` is committed and "Go-maintained afterwards". No Lua
tool is on the product's build or run path. Caveat: re-deriving
`mapmods_gen.go`/`timeless_gen.go`/`skillstatmap_gen.go`/`skills_custom_gen.go`
for a new game version needs luajit + the archive (§8.5).

## 5. Load-bearing behaviours to preserve

Each survives every step, re-expressed behind a typed API. The archive test
that would catch its loss is named.

| # | behaviour | why it exists | guarded by |
|---|---|---|---|
| L1 | `%.14g` number→text (S7) and `Quantize14` round-trip (`modparser/modcache.go:242-299`) | PoB's `Data/ModCache.lua` round-tripped numbers through `%.14g`; the port ships those exact values; mirage `InfoMessage`, conquered stat text, unique blobs are user-visible strings compared byte-for-byte | `TestModCacheAgainstShippedFile` (13 and 15 digits both fail — `parity.md:181`), `TestItemParseAgainstReference`, `TestCalcInitEnvAgainstReference`, `TestTimelessAgainstBins` |
| ~~L2~~ | ~~ASCII-only `string.lower`~~ — **dropped by decision 2026-08-29**: every `luaLower`/`asciiLower` is deleted for `strings.ToLower`; `scan` compiles `(?i)`+pattern and matches the original line (no lowercased copy, no offset alignment). Verified safe: no pattern key carries an uppercase literal outside a character class (`grep` over all tables, 2026-08-29), so no dead pattern comes alive. `PARSE`/`CALC` are the proof. | — | — |
| L3 | key presence vs absence vs `false` in `output`/`SkillData`/`Buff`/gem attributes (S6, U14, U15, U17, U37) | `scalarsOnly` diffs the key set; Lua `t.k = nil` deletes; `false` is stored | `TestCalcInitEnvAgainstReference`, `TestSkillsTabAgainstReference` (the `delete(matchesSocket)` case, `parity.md:262`) |
| L4 | `getStat`'s present-but-falsey fall-through (`modstore/eval.go:199-219`) | `output[stat] or cfg.skillStats[stat] or 0` — a stored `false` falls through, a stored `0` does not | `TestModStoreAgainstReference`, calc |
| L5 | the `tag.div` write-back into the shared tag (`modstore/eval.go:259,350`) | reference mutates the tag table; later evaluations of the same mod see the computed div | `TestModStoreAgainstReference` (synthetic tag cases), calc |
| L6 | string-typed numeric tag fields coerce in arithmetic (S3) | seven closures pass raw captures; Lua arithmetic coerces | after step 4 the *behaviour* (numeric result) is preserved by parsing at parse time; the *representation* moves to the test-side normalizer (§6.1) — `TestAgainstReference`, `TestModStoreAgainstReference`, `TestItemParseAgainstReference`, calc |
| L7 | `#mod` vs `ipairs` tag length (S13); `ParseTags` nil threshold (S14) | `CompareModParams`/`FormatTag` semantics | `TestModToolsAgainstReference` |
| L8 | positional-polymorphic `createMod` args and trailing-nil holes (U4) | tags can be nil in the reference tables | `TestAgainstReference`, `TestTablesAgainstReference` |
| L9 | `rejected` support list with nil holes stopping iteration (U26) | which supports get added | `TestCalcInitEnvAgainstReference` |
| L10 | captured `pairs()` order replay `AllocOrders/ExtraOrders` (S12) | float sums over LuaJIT hash order | calc; do not touch |
| L11 | deep copy before mutation (S11), shallow `MergeMod` sharing tag tables (`modstore/list.go:56-63`), `mergeKeystones` mutating the tree's shared modList (`keystones.go:20-24`) | reference aliasing is observable | `TestModStoreAgainstReference`, `TestSpecAgainstReference` |
| L12 | `Max` never registers all-negative candidates (`store.go:341`); `or nullValue` drops failed LIST evals (`db.go:222`, `list.go:173`); `sourceOK(guardNil)` asymmetry (`db.go:26`); `tabValueKept` (`db.go:261`) | `#EVAL`-documented reference quirks | `TestModStoreAgainstReference` |
| L13 | `"Flask nil"` slot text (`item/buildmodlist.go:145-150`), `HasModList`/`BuffModListInit` nil-vs-empty (`item/item.go:81,280-282`), `New` sanitises / `ParseRaw` does not (`parseraw.go:139-150`), `gsubLimitFunc` replacement-count limit (`applyrange.go:66-113`), `strings.Contains` divergence note (`parseraw.go:552-561`) | reference behaviour incl. `tostring(nil)` | `TestItemParseAgainstReference` |
| L14 | `EffectiveName` nil-fallthrough, `sdIdentity` short-circuit, 1-based LUT indices and `+1-337` offset, `luaRound`, `TimelessAdditions` first-seen merge record (S17, S19) | metatable nil-unshadowing; 4-pass rebuild; LUT layout | `TestSpecAgainstReference`, `TestTimelessAgainstBins`, `tree` abyss tests |
| L15 | `historic.go:396-399` strict `<` for `sizeNotable`; `tattoo.go:55,59` collapse-then-process; `tree.go:479` notableMap group rule | reference missing-field reads / order-sensitive rule | `TestTimelessAgainstBins`, `TestTattooArtifactMatchesArchive`, `TestTreeAgainstReference` |
| L16 | export: `bytesToFloat`/UTF-16/codepoint bug-compat, `readValue` sentinels, `Row` pointer identity + `SetCell`, positional `SpawnWeight` (`conquertables.go:158-160`), `MinionDef.Skip`, template directive order, `Scripts` order, `numKey`/`luaString`, `luaNum` at `script_bossdata.go:577,695` and `script_skills.go:505` (S20, I11) | reference exporter behaviour | `TestExportAgainstReference` (123/123), `TestTreeArtifactMatchesGGGSource`, conquertables guard in `TestExportAgainstReference` |
| L17 | `scan` `literalWeight` tie-break (S22) | stands in for Lua-pattern-length ordering | `TestAgainstReference` |
| L18 | string-sorted numeric keys (`"10" < "2"`) in canon (`modparser/canon.go:48`, `luacanon`) | `tools/canon.lua` serialization | every canon-based differential — this moves *entirely* test-side in step 1 |

## 6. Target design

### 6.1 `modparser` — the modifier

```go
type ModType uint8   // Base, Inc, More, Flag, Override, List, Max, Min; String() gives "BASE"…
type ModFlag uint64  // named consts replacing the ModFlag struct fields; same bit values
type KeywordFlag uint64
type SkillTypeID int64 // SkillType.X

type Mod struct {
    Name         string
    Type         ModType
    Value        Value
    Flags        ModFlag
    KeywordFlags KeywordFlag
    Source       string
    SourceSet    bool
    SourceSlot   string
    Replaced, Converted bool
    Tags         []Tag      // may contain nil (L8/L9); TagCount() = ipairs length (L7)
}
```

`Value` is a sealed sum type — the observed kinds are exhaustively:

```go
type Value interface{ isValue() }
type Num  float64        // int/int64/float64 all normalise here (U2 hazard closed); +Inf allowed
type Flag bool
type Str  string
type ModRef   struct{ Mod *Mod; OnlyAllies bool }                       // {mod=…} records
type SkillRef struct{ SkillID string; Level, TriggerChance Num; Triggered, NoSupports, OnlyAllies, OnCrit, IgnoreHexproof bool; Source string }
type DataRef  struct{ Key string; Value Value }                          // SkillData / JewelData
type ExplodeRef struct{ Type string; Chance, Amount Num; KeyOfScaledMod string }
type JewelFn  struct{ Type string; Func JewelNodeFn; Radius string }
type Conqueror struct{ ID string; Kind ConquerorKind }                   // conquerorList (D3)
type Pairs    [][2]float64                                               // ramp / dmg pairs (I3)
```

`Tag` is a sealed interface over per-kind pointer structs (pointer so L5's
in-place `div` write-back stays an ordinary field assignment on a shared
object):

```go
type TagKind uint8
type Tag interface{ Kind() TagKind }
type Actor uint8 // ActorNone, ActorEnemy, ActorParent, ActorPlayer

type CondTag        struct{ Actor Actor; Var string; VarList []string; Neg bool }           // Condition + ActorCondition (1,349 uses)
type MultiplierTag  struct{ Actor Actor; Var string; VarList []string; Div, Limit, GlobalLimit, Threshold Opt[float64]; DivVar, LimitVar, LimitStat, GlobalLimitKey string; LimitTotal, LimitNegTotal, Neg, Upper bool; Threshold-kind bool /*MultiplierThreshold*/ }
type StatTag        struct{ Kind TagKind /*PerStat|PercentStat|StatThreshold*/; Actor Actor; Stat, ThresholdStat string; StatList []string; Div, Percent, Threshold, ThresholdPercent Opt[float64]; Upper, Neg bool }
type SkillNameTag   struct{ SkillName string; SkillNameList []string; IncludeTransfigured bool }
type SkillTypeTag   struct{ SkillType SkillTypeID; SkillTypeList []SkillTypeID; Neg bool }
type SkillIDTag     struct{ SkillID string }
type SkillPartTag   struct{ Part SkillPart; PartList []SkillPart }       // SkillPart = Num | Str (S2 closed)
type SlotTag        struct{ Kind TagKind /*SocketedIn|SlotName|SlotNumber|InSlot|DisablesItem*/; SlotName string; SlotNameList []string; Num int; Keyword string }
type GlobalEffectTag struct{ EffectType, EffectName, Effect string; Unscalable, FromAllies bool }
type ItemCondTag    struct{ ItemSlot, RarityCond, SearchCond, NameCond, ExcludeItemType, Sockets, SocketColor string; ShaperCond, ElderCond, CorruptedCond, Neg bool }
type DistanceRampTag struct{ Ramp Pairs }
type MeleeProximityTag, LimitTag …                                       // per the 66-key inventory
type ModFlagOrTag   struct{ ModFlags ModFlag }
type KeywordAndTag  struct{ Keyword string; KeywordList []string }
type GlobalTag, OtherTag struct{ … }
type RawTag         map[string]string                                    // ParseTags round-trip only (FormatTag)
type Opt[T any]     struct{ V T; Set bool }                              // absent ≠ zero (S14, L3); lives in the internal/ helper package (§6.9)
```

`Opt[T]` is not a whole-struct convention (decision 2026-08-29): a field is
plain `bool`/`float64`/`string` wherever it is only ever read as a value;
`Opt[T]` only where the reference distinguishes absent from zero/false *and*
an archive diff sees it — here the tag numerics `Div`/`Limit`/`Threshold`/
`Percent`/`Base`/`GlobalLimit`. Slices use `nil` vs `[]T{}` directly for
nil-vs-empty. Tags are pointer-held so L5's `Div` write-back is visible to
later evaluations; `Opt` is the field type on that pointer-held struct.

`scan` (`mod.go:194-245`): `newScanTable` compiles `"(?i)" + k` and
`FindStringSubmatchIndex` runs on the original line; plain tables either
`strings.ToLower` both sides or go through `regexp.QuoteMeta` on the same
path (implementer's pick). `asciiLower` deleted.

Template strings (`"{Hand}Attack"`, `"{SlotName}"`, `"Ring {OtherSlotNum}"`)
stay as strings; `item/buildmodlist.go:166-177`'s substitution walks typed
string fields instead of every map value.

Constructors: the transcribed tables keep short builders but typed:
`mod(name, ModType, Value, ...Tag)`, `modf(name, ModType, Value, ModFlag, KeywordFlag, ...Tag)`,
`mods(source string, …)`; closures become `func(caps) Result` where
`type Result struct{ Mods []*Mod; Recognised bool }` (U5, U6). `D` shrinks to
an unexported pattern-table record (`arr []Value; ctl controlKeys` — a struct
with the ~30 control keys as typed fields) and never leaves the package.

`Parse(line string) (mods []*Mod, extra string, recognised bool)`.

Parse-time numeric coercion (S3): the seven closures change `c.s(n)` →
`c.n(n)` for `div`/`limit`/`percent`. Behaviour is unchanged (Lua coerced
in arithmetic); representation changes. The archive dumps keep the string.
**Test-side normalizer** (`test/luacanon`): `NormalizeArchiveMods(json)`
parses the fixture's canon JSON and, for the closed set of numeric tag
fields (`div limit percent base threshold thresholdPercent limitTotal? no —
booleans excluded`), converts a numeric string to a number before
comparison. Applies to parse/modtables/modtools/modstore/item/calc/gamedata
fixtures (all carry mods). This is the sanctioned "normalize both sides".

Mod cache codec (U7): `EncodeMods([]*Mod)`/`DecodeMods` become a typed
`json.Marshaler` on `Mod`/`Value`/`Tag` with an explicit `"kind"`
discriminator; `data/raw/modcache.jsonl`, `skills.json`, `minions.json`
regenerate (numbers where strings were; `kind` field). `Quantize14` stays.

`modparser/canon.go` moves to `test/luacanon` (B2).

### 6.2 `modstore`

```go
type OutValue struct{ Kind outKind /*Absent|Num|Bool|Str*/; N float64; B bool; S string }
type Output map[string]OutValue     // absent key ≠ present-false (L3, L4)
type Actor struct{ …; Output Output; WeaponData1, WeaponData2 *WeaponData; … }
type Conditions map[string]CondValue // CondValue = Bool | Str (U12)
type Cfg struct{ …; SkillPart Opt[SkillPart]; SkillGem *data.Gem; SkillStats Output }

func (s Store) Sum(t ModType, cfg *Cfg, names ...string) float64
func (s Store) More(cfg *Cfg, …) float64
func (s Store) Flag(cfg *Cfg, …) bool
func (s Store) Override(cfg *Cfg, …) (Value, bool)
func (s Store) List(cfg *Cfg, …) []Value
func (s Store) Tabulate(t ModType, cfg *Cfg, …) []TabEntry   // TabEntry{Value Value; Mod *Mod}
```

`Combine` is deleted (I1); `ListInternal` returns instead of taking `*[]any`.
`evalMod` switches on `tag.Kind()` with typed field access; `tArith`/`tstr`/
`tnum`/`tlist`/`luaArith`/`arithNum`/`luaEq` are deleted; the L5 write-back
becomes `mt.Div = Opt[float64]{v, true}` on the shared `*MultiplierTag`.
`Externals` becomes a `Resolver` interface passed at store construction (I2).
The nine panics keep their "(the Lua errors)" comments; `store.go:115` gets
one or becomes unreachable.

### 6.3 `calc`

`Output` keeps its map shape (U14 is load-bearing on key set) but not its
value type. The type is `modstore.Output` (declared in step 5 beside
`getStat`'s absent/present-false/numeric semantics; calc adopts it in step
12 with no new import edge):

```go
type Output struct {
    Num  map[string]float64
    Flag map[string]bool
    Str  map[string]string
    MainHand, OffHand *Output   // lifted from output["MainHand"] (I5; not diffed)
}
func (o *Output) N(k string) float64      // absent → 0 (S5's nil→0)
func (o *Output) Has(k string) bool
func (o *Output) SetN / SetFlag / SetStr / Del
```

`SkillData` same three-map shape (U15's open key set; ~90 `triggeredBy*`
flags may later fold into a `TriggerSource` bitset — §8.4). Closed bags
become structs: `ConfigInput` (26 keys, `Opt[float64]`/`Opt[bool]`/`Opt[string]`),
`Buff{Type, Name string; ActiveSkillBuff bool; ApplyNotPlayer, ApplyMinions, ApplyAllies, AllowTotemBuff Opt[bool]; ModList}`
(pointers/Opt keep `assignKV` delete semantics, L3), `WeaponData` (15 fixed
fields + `Extra map[string]Value` for the LIST extension), `Minion.Uses map[string]bool`,
`type ExplodeSource interface{ ExplodeKey() string }` on `*Item`/`*NodeInput`,
`CurseSlots []*curseEntry`, `SkillPart Opt[float64]`,
`GrantedSkill{Triggered Opt[bool]; TriggerChance Opt[float64]}`,
`FlaskBaseInput.Life/Mana Opt[float64]`, `rejected []*ActiveEffect` (nil holes kept).
`truthy` survives nowhere in calc: map reads become `Has`/`Flag` reads,
locals become their real types. `anyNum` dies with `Output.N`. `luaOr` dies
with `Buff`. `rampTable` returns `modparser.Pairs`. `getCachedOutputValue`
returns a small struct of `Opt[float64]`. `modFlagByName` uses a generated
`modparser.ModFlagByName map[string]ModFlag` (D6).

Not touched: file layout (N1), `AllocOrders` (L10), `triggerConfigTable`
(D7), unported-branch and reference-error panics.

### 6.4 `data`

```go
type StatMapEntry struct {
    Mods              []StatMapMod        // StatMapMod = Mod *modparser.Mod | Group []StatMapMod
    Div, Mult, Base, Value, BaseMultiplier, Cooldown, LevelRequirement Opt[float64]
    Cost              map[string]float64
    StatInterpolation []float64
    SkillFlag         string
}
type GrantedEffect struct { …; Levels map[int]*SkillLevel; RequireSkillTypes []SkillTypeID /*0 = unknown*/; QualityStats, ConstantStats []schema.StatValue; Custom SkillCustom }
type SkillCustom struct { FromItem, FromTree, Legacy, MinionHasItemSet bool; Parts []SkillPart; MinionList, AddMinionList []string; AddFlags, BaseFlags map[string]bool; BaseEffectiveness Opt[float64]; Levels []*SkillLevel; BaseMods []*modparser.Mod; Callbacks map[CallbackKind]bool }
type CallbackKind uint8 // InitialFunc, PreSkillTypeFunc, PreDamageFunc, PostCritFunc, ExplosiveArrowFunc
```

`UnportedFn` stays as the marker for a callback whose body is not yet in
`calc/skillfuncs.go`; `skillFuncs` keys become `struct{ID string; Kind CallbackKind}` (D5).
Timeless tables: `NodeIDList map[int64]NodeIndex` + `LocalIDToGlobalID []int32`
+ `Size, SizeNotable int`; `AbyssNotableNames map[string]string`;
`TimelessJewelTradeIDs map[int][]TradeID`; `MapMods{AffixData map[string]*MapMod; Prefix, Suffix []ValLabel}` with `MapMod.Apply MapModApplyKind`.
`FoulbornMap map[string]map[string]string` decoded directly. `Minion.Hostile`
→ `HostileScale Opt[float64]` + `Hostile bool`; `DamagePenetrations map[string]Opt[float64]`;
`ValidBases []baseOnlyEntry`; `Transform TransformKind`. `Load(src Sources) error`.
`modConstants` replaced by exporter-emitted numeric ids (or `modparser.SkillTypeByName`).
The six `*Canon` builders move to `test/luacanon` adapters (B1). Exporter
emits decoded text so `luaUnescape`/`luaStringLiteral` die (S15).

Generators: the emitted shape changes are in `tools/gen_skilldata.lua`
(`goStatMapEntry :143-173`, custom keys `:342-356`, `goMod :46-81`) and
`tools/gen_datatables.lua` (`goExpr :32-80`). Both are one-shot; §8.5 decides
whether to edit-and-rerun or convert once in Go and delete them.

### 6.5 `item`

`WeaponData{Type, SubType, Name string; AttackRate, AttackSpeedInc, Range, RangeBonus, CritChance, TotalDPS, ElementalDPS float64; Damage [5]MinMax; CountsAsAll1H, CountsAsDualWielding bool; AddedUsing map[string]bool; Extra map[string]modparser.Value}`
and the same pattern for `ArmourData`, `FlaskData`, `TinctureData`,
`JewelData` (every key `tree.jd` reads becomes a field; `Extra` keeps the
data-driven LIST part). `GrantedSkill{SkillID, Source string; Level, TriggerChance Opt[float64]; NoSupports, Triggered Opt[bool]}`.
`Affix.Range AffixRange{Single Opt[float64]; Multi []float64}`.
`Requirements struct{ Str, Dex, Int, Level, StrMod, DexMod, IntMod Opt[float64] }`.
`ModLine.ValueScalar Opt[float64]`. `item.FoldText` (S9) is the one fold
table, used by `skills` too. `item/tail.go:17-35` deletes with U30.

### 6.6 `tree`

`Node` gains `ExpansionJewel *ExpansionJewel{Size, Index int64; Proxy NodeID; Parent *NodeID}`,
`ActiveEffectImage string`, `IsJustIcon, IsMultipleChoice bool` (U42); `Raw`
and `decode.go` are deleted; `Load` unmarshals into a `schema.PassiveTree`
doc (as `tattoo.go:20` does). `type NodeKind uint8` (seven values),
`type ConquerorKind uint8` (1-11, `IsAbyss()`), `type OverrideKind string`
(open; const `AlternateMastery`), `type AbyssComponentKind uint8`, named
constants for pool indices 77/78/91/110.
`SpecNode.ConqueredBy *Conquest{Seed float64; Conqueror ConquerorKind; ConqID string; Abyss []AbyssComponent}`
built directly from typed `JewelData.ConqueredBy` and from
`collectAbyssConquests` (U41 gone). Shadowing: `nodeOverride{src *Node; srcME *MasteryEffect; Dn string; Name *string; Stats; KeystoneMod; IsTattoo bool; OverrideType OverrideKind}`
with `sameSource(*Node) bool` replacing `sdIdentity any`, plus `Stats.cloneSd()`
in `resetToSource` to close the splice aliasing. `Load(version) (*Tree, error)`.

### 6.7 `skills`

```go
type Gem struct { NameSpec, GemID, SkillID, ErrMsg Opt[string]; Level, Quality, Count, ReqLevel, ReqStr, ReqDex, ReqInt Opt[float64]; Enabled, EnableGlobal1, EnableGlobal2, New, Triggered, MatchesSocket Opt[bool]; CalcOwned map[string]float64; GemData *data.Gem; GrantedEffect *data.GrantedEffect }
type SocketGroup struct { Enabled bool; Slot, Label, Source, ImbuedSupport Opt[string]; GroupCount, MainActiveSkill /*…*/ Opt[float64]; GemList []*Gem }
```

`Opt[string]{"nil", true}` reproduces the saved-`"nil"` quirk; `Set=false`
reproduces `delete(matchesSocket)`. `luaLevelsLen` dies with U32.

### 6.8 `export`

Spec-generated typed accessors on `Row` — `Int(col) int64`, `Str(col) string`,
`Bool(col) bool`, `Ref(col) *Row`, `Refs(col) []*Row`, `Ints(col) []int64`,
`Ivl(col) Interval`, `At(i)` for duplicate names — panicking only on spec
mismatch (generate `SpawnWeight2` for the duplicate). Keep `rowCache`
interning, `SetCell`, sentinels. `ColType` iota. `Row.Index` 0-based
(rename `ID`). `Rows() iter.Seq[*Row]`.
`type Script struct{ Name, OutName string; Build func(*Ctx) (Document, error) }`
with `Document interface{ isDocument() }` implemented by every `schema.*` doc
(or a generic `register[T]` mirroring `test/luarender/luarender.go:32`).
`statEntry{ID string; Stats [6]struct{ID string; Val float64}}` with
`fromModsRow`/`fromOT` replacing the `[]any` union. Templates become
structured JSON (`{"directive":"monster","variety":…,"name":…,"skills":[…]}`)
with typed per-template documents and a typed visitor; `reDirective` and
`strings.Fields` positional parsing die; `WalkTemplate` moves to
`templates.go`. Twelve data-driven panics become `error` returns. `luaStr`
dies; `luaNum` survives only at its three artifact-bound sites under the
consolidated name. Directive order and `#emit` sequence are preserved (L16).

### 6.9 Cross-cutting

- `internal/util` holding `Opt[T]` and the reference
  numeric semantics that stay (L1): `FormatG14(float64) string` /
  `Quantize14`, `RoundHalfUp(v float64, dec int) float64`, and one
  `Tonumber(string) (float64, bool)` (documented divergences from Lua
  `tonumber`, S10). Eight `%.14g` bodies and four tonumbers collapse into
  it. No lower-casing helper (S8 dropped); `luaTrim` sites use
  `strings.TrimSpace` unless a differential shows Lua `%s` differing (it
  covers `\v\f` which `TrimSpace` also covers).
- `sanitiseText` → `item.FoldText` (S9), the single fold table.
- **Ownership principle (decided 2026-08-29)**: the package that produces or
  stores a value declares its type and every consumer imports it as-is —
  `modparser` → `Mod`/`Tag`/`Value`; `modstore` → `Output`; `data` →
  `StatMapEntry`/`GrantedEffect`; `item` → `WeaponData`/`JewelData`/
  `GrantedSkill`; `export` → typed `Document` returns and dat accessors. No
  contracts package: that would only be warranted for a type two services
  must agree on that neither produces, and nothing in the plan is in that
  position.
- Errors: `Load` functions return `error`; panics remain only for (a)
  reference-error mirrors, each with a "(the Lua errors)" comment, and (b)
  unported-branch guards.
- `lua:"…"` tags: replaced by test-side field-name tables in
  `test/luacanon` (B3) — see §8.8 for the alternative.

## 7. Ordered step plan

### 7.0 Verification vocabulary

| shorthand | command | proves |
|---|---|---|
| `BUILD` | `go build ./... && go vet ./...` | compiles, vets |
| `FULL` | `go test ./...` | every committed-fixture differential (≈55s) |
| `PARSE` | `go test ./test -run 'TestAgainstReference$\|TestTablesAgainstReference\|TestModToolsAgainstReference'` | parser vs `parse_archive.jsonl` (13,173 lines), tables vs `tables_archive.jsonl` (8,800), modtools vs `modtools_archive.jsonl` |
| `STORE` | `go test ./test -run TestModStoreAgainstReference` | 18,525 modstore checks vs `modstore_archive.jsonl` |
| `CACHE` | `go test ./test -run 'TestModCacheAgainstShippedFile\|TestModCacheGeneration'` | mod cache decode/re-encode + regeneration byte-equal to `data/raw/modcache.jsonl` |
| `DATA` | `go test ./test -run 'TestGameDataAgainstReference\|TestTreeAgainstReference\|TestTattooArtifactMatchesArchive\|TestTreeArtifactMatchesGGGSource'` | 136 data subtrees, tree load, tattoo doc, tree artifact |
| `ITEM` | `go test ./test -run TestItemParseAgainstReference` | 1,034 items |
| `SKILLS` | `go test ./test -run TestSkillsTabAgainstReference` | 369 groups / 1,001 gems |
| `SPEC` | `go test ./test -run TestSpecAgainstReference && go test ./tree` | 9,046 spec nodes; abyss bins |
| `TIMELESS` | `go test ./test -run TestTimelessAgainstBins` | 33.1M legion cells |
| `CALC` | `go test ./test -run 'TestCalcInitEnvAgainstReference\|TestCalcFixtureEcho'` | 147 variants × 6 checkpoints + fixture echo |
| `EXPORT` | `MP_EXPORT=1 go test ./test -run TestExportAgainstReference` | 123 generated files byte-identical (≈112s; needs the GGPK extract) |
| `ARTIFACTS` | `go run ./cmd/sourceupdate -skiptests` then `git status --porcelain data/raw` | artifacts regenerate to the committed bytes (run when a step changes an artifact's encoding; the resulting diff is the intended one and is committed with the step) |

Every step ends with `BUILD` + `FULL`; the step-specific commands name the
differential most likely to move. `EXPORT` runs at every step that touches
`export/`, `data/schema`, `modparser`'s codec, or `data.StatMapTable`.

### 7.1 Steps

Each step lands independently with the suite green. M = mechanical, J =
judgement. Dependencies reference step numbers.

**Step 1 — move Lua-shape knowledge to `test/`** (M; deps none) — landed 2026-08-29
Scope: `modparser/canon.go` → `test/luacanon/modcanon.go`; `data.{ModCanon,DCanon,GrantedEffectCanon,StatMapEntryCanon,SkillLevelCanon,GemCanon}` → adapters in `test/luacanon` (registered from `test/gamedata_test.go` as today); `data.TrimTreeDependentUniques` → a test helper that re-runs the load-time trim through an exported *state* accessor only if one already exists (else keep, documented as test-only — §8.10); `modparser.Tables()`/`verify.go` stay (modtables test needs them) but get a "test scaffolding" doc line. `modparser/scan_test.go` uses the moved canon.
Verify: `PARSE STORE DATA CACHE`.

**Step 2 — one home for reference numeric semantics; lower-casing dropped** (M; deps none) — landed 2026-08-29
Scope: new `internal/util` with `Opt[T]`, `FormatG14`/`Quantize14`, `RoundHalfUp`, `Tonumber`; replace the 8 `%.14g` and 4 `tonumber` bodies (S7, S10). Delete all five `luaLower`/`asciiLower` (S8) for `strings.ToLower`; `newScanTable` compiles `(?i)`+pattern and `scan` matches the original line. `item.FoldText` replaces both `sanitiseText`s (S9), extended to uppercase accents; `skills/skills.go:308-350` deleted. The `luaTrim` sites move to `strings.TrimSpace`.
Risk: `FoldText`'s uppercase-accent extension and `(?i)` change behaviour only if the corpus contains such text — `ITEM`/`PARSE` are the proof; a failure there means a real base/mod name differs by an accent and the fold must be narrowed, not the test.
Verify: `PARSE STORE ITEM SKILLS SPEC CALC EXPORT DATA CACHE`.

**Step 3 — typed enums and flags in `modparser`** (M, wide; deps none) — landed 2026-08-29
Scope: `ModType`, `ModFlag`, `KeywordFlag`, `SkillTypeID`, `modForm` enum (D1, D2, U8); `Mod.Type/Flags/KeywordFlags` retyped; the transcribed tables rewritten by script (`"BASE"`→`Base`, `ModFlag.Attack`→`FlagAttack`…); `modparser.ModFlagByName`/`KeywordFlagByName`/`SkillTypeByName` generated maps replace both reflection sites (D6, G7). Callers in modstore/item/calc/data/export updated (literal strings → constants). `data/schema.SkillHeader.SkillTypes` keeps its strings for now (exporter unchanged); `data/skills.go` resolves through the new map.
Verify: `PARSE STORE CACHE ITEM CALC DATA EXPORT`.

**Step 4 — typed `Tag` and `Value`** (J, the risky one; deps 1, 3) — landed 2026-08-29
Scope: §6.1 types; `mod()`/`flag()`/`d()`/`p()` replaced by typed builders; the seven `c.s`→`c.n` coercions (S3); `special.go`/`tags.go`/`preflags.go`/`modflags.go`/`names.go`/`jewels.go`/`special_hand.go` rewritten by a one-off conversion program (AST-level, run once, deleted) with hand fix-ups; `D` internalised; `Parse` signature per §6.1; codec (U7) retyped; `CopyMod` → `Clone()`; `tagArrayLen` semantics kept via nil entries. Consumers: modstore (`evalMod` switch on `Kind()`), item (`buildmodlist` tag reads/template substitution, `tail.go`), calc (`modArgs`, `asAnyList`, `copyTagValue`, LIST value reads), tree (`stats.go:parse3`, `conquer.go` conqueror), data (`skills.go` statMap mods, `minions.go`), export (`script_minions.go:266-290`, `script_skills.go` baseMods), `internal/modcachegen`.
Test-side: `test/luacanon` numeric-string normalizer for archive mod fixtures (§6.1) applied in `PARSE STORE ITEM CALC DATA`.
Artifacts: `ARTIFACTS` regenerates `modcache.jsonl`, `skills.json`, `minions.json` (numbers replace the 14 string-typed fields; codec `kind` field). `test/luarender/minions.go:tableToString` reads typed tags for the `mod(...)` reprint.
Risk: the coercion changes representation in every fixture that carries one of the 14 lines; `FormatTag`/`CompareModParams` outputs for those lines must be re-checked in `TestModToolsAgainstReference`; `luaEq`'s map-panic path disappears (S2). Sub-land in three commits if needed: (a) `Value`, (b) `Tag`, (c) `Parse` signature + codec.
Verify: `PARSE STORE CACHE ITEM SKILLS SPEC CALC DATA EXPORT ARTIFACTS`.

**Step 5 — `modstore` typed surface** (J; deps 4) — landed 2026-08-29
Scope: §6.2 — `Output`/`OutValue`, `Conditions`, `Cfg` fields, `Combine` split, `ListInternal` return, `Externals` → `Resolver`, `WeaponData` pointer type shared with item (or a narrow interface), `tlist` gone, mod-type constants (from step 3). calc's 48 files of `Override(...)`/`List(...)` callers updated (`truthy(ov)` → `ok`). `store.go:115` panic annotated or removed (G4). U25: `modparser.JewelStoreWriter` retyped in place — `AddMod(*Mod)`, `MergeMod(*Mod)`, `AddList([]*Mod)`, `Sum(ModType, names ...string) float64` (no cfg), `Mods() []*Mod`; `JewelNodeFn func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag)`; the 56 functions stay in `modparser/jewels.go`; `calc/radius.go`'s `listWriter` keeps adapting `*modstore.List` (`l.Sum(t, nil, names...)`) and loses its type switch and panic. No new package, no move.
Verify: `STORE CALC ITEM SPEC`.

**Step 6 — `data` typed model; Lua generators deleted** (M; deps 3, 4) — landed 2026-08-29
Scope: §6.4 — `StatMapEntry`, `SkillCustom`/`CallbackKind`, `Levels map[int]`, `*SkillTypes []SkillTypeID`, `Quality/ConstantStats []schema.StatValue`, timeless tables, `MapMods`, `FoulbornMap`, `Hostile`, `DamagePenetrations`, `ValidBases`, `Transform`; the four `*_gen.go` rewritten typed by a one-off Go conversion (or by hand) and `tools/gen_datatables.lua` + `tools/gen_skilldata.lua` deleted (decided 2026-08-29: no regeneration path is kept — the source tables are upstream hand-maintained Lua and a future version is ported by hand regardless; re-running the generators would overwrite typed Go with `any`-bags); drop the `_gen` suffix (N5); `deepCopyAny`/`anyList`/`t_insert` emulation/`LazyStatMapCopy` retyped; `Load` returns `error` (callers: `test/loaddata_test.go`, `cmd/*`, `internal/modcachegen`). `item/tail.go:17-35` deleted. `tree/historic.go:384-400`, `conqueredpassives.go:77`, `test/timeless_test.go:71,83` read typed tables. `calc/skillmods.go:130-134,348`, `calc/offencebody.go:210-224`, `calc/skillfuncs.go` keys updated.
Verify: `DATA CALC TIMELESS SPEC EXPORT` (`EXPORT` because `script_minions.go` reads `StatMapTable`).

**Step 7 — exporter emits decoded values; `data` drops Lua-literal decoding** (J; deps 6) — landed 2026-08-29
Scope: `export/script_skills.go`/`script_bossdata.go` and their `data/schema` fields ship decoded text and numeric `SkillType` ids; `data/luanum.go` `luaUnescape`/`luaStringLiteral`/`luaTonumber` deleted (S15); `modConstants` deleted (D6); `data/raw/skills.json`, `bossdata.json` regenerate; `test/luarender/skills.go`/`bossdata.go` re-escape on render (test-side).
Verify: `DATA CALC EXPORT ARTIFACTS`.

**Step 8 — `item` sub-structs** (M; deps 4, 5) — landed 2026-08-29
Scope: §6.5. Consumers: calc (`items.go`, `skillmods.go` weapon data, `skills.go` granted skills, `setup.go` flask/jewel data), tree (`jd`/`jdTrue` → field reads; U40's item-side `ConqueredBy` map), skills (sanitiser already shared from step 2), modstore `ItemCondition` reads.
Verify: `ITEM SKILLS SPEC CALC`.

**Step 9 — `tree` typed nodes and kinds** (J; deps 8) — landed 2026-08-29
Scope: §6.6 — typed tree doc decode (delete `decode.go`), `Node.Raw` removed, `NodeKind`/`ConquerorKind`/`OverrideKind`/`AbyssComponentKind`, `Conquest`, `nodeOverride` with `sameSource`, `cloneSd`, `buildAbyssWorld` reading `*Tree`, named pool indices, `Load` error, N4 shadows fixed.
Verify: `SPEC TIMELESS DATA CALC`.

**Step 10 — `skills` typed gems and groups** (M; deps 4, 8) — landed 2026-08-29
Scope: §6.7; calc's skills-tab input bridge (`calc/input.go` `SocketGroupInput.KV`/`SocketGemInput.KV`, `test/calc_native_test.go`) reads the typed fields; `luaLevelsLen` deleted; the two unchecked assertions gone.
Verify: `SKILLS CALC`.

**Step 11 — `calc` closed bags → structs** (M; deps 4, 5, 8, 10) — landed 2026-08-29
Scope: `ConfigInput`, `Buff`, `WeaponData` (shared type from step 8), `Minion.Uses`, `ExplodeSource`, `CurseSlots`, `SkillPart`, `GrantedSkill`, `FlaskBaseInput`, `RadiusJewel.Attributes`, `rejected []*ActiveEffect`, `rampTable`, `getCachedOutputValue`, `luaOr`/`setKV`/`assignKV` on those bags, `tonum` (`calc/ehp.go:17-33`), `listWriter`'s remaining `any` (U25, already retyped in step 5), the nine shape-assertion panics (G2).
Verify: `CALC SPEC SKILLS`.

**Step 12 — `calc` `Output`/`SkillData` typed maps; `truthy`/`anyNum` retired** (M but large; deps 5, 11) — landed 2026-08-29
Scope: §6.3 `Output`/`SkillData` three-map types across all 56 files (1,100+ sites, scriptable: `output["K"]` reads in numeric context → `.N("K")`, `truthy(output["K"])` → `.Flag["K"]`/`.Has("K")`, writes by value kind); `MainHand/OffHand` lifted; modstore `Actor.Output`/`Cfg.SkillStats` share the type (step 5 defined `OutValue`; reconcile to one type here — §8.12); `performActor.output`, `offenceCtx`, `MirageResult`, `Minion.Output` all use `modstore.Output` (decided: one type, owned by modstore); `test/calc_test.go:scalarsOnly` reads the three maps (test-side).
Verify: `CALC SPEC SKILLS STORE`.

**Step 13 — `export` input side** (M for accessors/Script/statEntry/iter, J for templates; deps 6 for the `StatMapTable` read, otherwise independent — can run in parallel with steps 5-12) — landed 2026-08-29
Scope: §6.8 — spec-generated accessors + `ColType` + 0-based `ID` + `iter.Seq` (13a); `Script.Build` typed + `Ctx.modsDoc` (13b); `statEntry` (13c); structured templates + typed visitor (13d — converts `export/templates/**/*.json` once; `test/luarender/template.go` and the archive-template wrapper reconstruction keep reading the *archive* `.txt` templates, unaffected); 12 panics → errors (13e); `luaStr`/`luaTostringAny` deleted; `fixTree` checked assertions (U48); `RawSourcesFromDir` and no subprocess in `sourceupdate` (G8); `WriteEnumFiles` into the in-memory `DatSet` (G9 — verify `EXPORT` still finds the enum tables).
Verify: `EXPORT` after each sub-step; `go run ./cmd/sourceupdate -skiptests` then `git status --porcelain data/raw` clean.

**Step 14 — `StubHandoff`; `TrimTreeDependentUniques`** (J; deps 12) — landed 2026-08-29
Scope: B3 is dropped by decision (the `lua:"…"` tags stay). B4 — `data.TrimTreeDependentUniques` deleted; `data.Sources` gains an option that skips tree-dependent unique generation and `gamedata_test.go` loads with it. B5 — `StubHandoff` becomes a parameter of the test's replay entry (`ReplayInput` field) rather than an `Env` flag, if `calc/buildoutput.go:55` can read it from the replay input; otherwise document it as the fixture-mode flag it is.
Verify: `DATA CALC`.

**Step 15 — regeneration and docs** (M; deps all) — landed 2026-08-29
Scope: `parity.md` rows and `lua-go-map.md` rows for every renamed type/helper; `go-modparser-port`/`go-calccore-port`-style notes only where a trap changed; `tools/gen_*.lua` already deleted in step 6; `_gen` suffixes corrected (N5); `data/raw` embed pruned to per-document embeds without `skillgemlist.json`/`mapuniquetofoulborn.json`/`statdesc.json` (G6); `#EVAL` size tags on `performBuffs` (`calc/performbuffs.go:88`) and `buildActiveSkillModList` (`calc/skillmods.go`) (N2).
Verify: `FULL EXPORT`.

Parallelisable: step 13 with 5-12; step 2 with 3; steps 8/9/10 are sequential
on 4-5 but independent of each other except 9 on 8.

**Deviations from the plan as landed** (all steps green 2026-08-29):
- 2: `FormatG14` and `FormatIntOrG14` kept distinct (integers in [1e14,1e15) and `-0` differ); `Tonumber` accepts `0x` hex ints.
- 3: `ModType` carries `Chance`, `Dummy`, `FlagTypo` extras; `ParseFormattedSourceMod` returns `*FormattedSourceMod`.
- 4: `Value` has extra record kinds (`PropertyModRef`, `GemPropertyRef`, `SelfDamage`, `AscendancyNodeRef`, `LinkedSupportRef`); `Actor` stays a string; `PatternEntry` exported for the tables differential; `luacanon.NormalizeArchiveMods` is the test-side normalizer; `RawTag` keeps one upstream typo tag type.
- 5: `GemIsType` moved onto `GemRef`.
- 6: statMap key set is exactly `div mult base value skillFlag` (the plan's extras were level-row keys); `SkillMod` sum includes `TypoMod`; `TimelessJewelLUTs` deleted.
- 8: `GrantedSkill` flags are plain bools.
- 9: `buildAbyssWorld` reads the typed tree doc kept on `Tree` (`Load` drops legacy nodes); `NodeID` is a schema type only.
- 10: `FindSkillGem`/`GetGemStatRequirement` moved to `skills/gemlookup.go`.
- 12: `StatSource` removed; `output["Minion"]` dropped; sub-tables lifted to `performActor.mainHand/offHand`.
- 13: G9 (`WriteEnumFiles` into the in-memory `DatSet`) not done — it still writes into the GGPK extract.
- 14: the skip flag is package state set by `Load`; `modcachegen.BuildFrom` loads its own full state and the gamedata test its own skipped state; `data.Load` made idempotent (`Costs` reset).
- 15: statdesc.json needs no on-disk reader — the export differential renders the freshly built document in memory; `RawSourcesFromDir` + `modcachegen.BuildFrom` replace the `sourceupdate` subprocess.

### 7.2 Mechanical vs judgement, and what is risky

- Mechanical (scriptable, review by diff): 1, 2, 3, 8, 10, 11, 12, 13a-c/e, 15.
- Judgement: 4 (type design already fixed above; the judgement is in the
  one-off table conversion and the fixture normalizer), 5 (`OutValue`
  shape), 7 (which schema fields change), 9 (override struct), 13d
  (template document shape), 14.
- Risky: **4** — 448 KB of transcribed tables rewritten in one go, fixture
  normalization added to five differentials, three artifacts regenerated.
  Mitigation: land `Value` first (no fixture change), then `Tag` with the
  normalizer, then the `Parse` signature; run `PARSE` after each table file
  is converted (the parser is exercised per line, so a single mistranscribed
  closure fails exactly its lines). **12** — sheer site count; mitigation:
  convert per file with `CALC` under `MP_ONLY` narrowing. **13d** — template
  directive order is load-bearing; mitigation: the converter must preserve
  array order and `EXPORT` checks all 123 files.

## 8. Open questions (human decisions)

Decided 2026-08-29 (recorded in §6.1, §6.9, §7 steps 2/6/12):

- **Opt[T] vs pointers** → `Opt[T]` in an `internal/` package, used only
  where the reference distinguishes absent from zero/false and the archive
  diff sees it: the `assignKV`-deleted buff flags, `matchesSocket` and the
  `"nil"`-string gem attributes, tag numerics (`div`/`limit`/`threshold`…),
  `ConfigInput` values, item `Requirements`, `ValueScalar`, `SkillPart`,
  flask `Life`/`Mana`, `TriggerChance`, `Affix.Range`, `getStat`'s
  present-but-false fall-through. Plain value types everywhere else; nil vs
  `[]T{}` for slices.
- **Shared helpers** → ASCII-lower dropped entirely (`strings.ToLower`,
  `(?i)` regex in `scan`); `sanitiseText` → `item.FoldText` with uppercase
  accents folded, skills' copy deleted; the helper set is
  `Quantize14`/`FormatG14`, `RoundHalfUp`, one `tonumber`, beside `Opt[T]`.
- **Generated tables** → convert once in Go, delete both Lua generators, no
  regeneration path.
- **`Output` ownership** → one type, owned by `modstore`; calc adopts it.
  General rule: the producing/storing package declares the type.

Decided 2026-08-29, second round:

- **Plan filename** → this file.
- **Helper package** → `internal/util` (`Opt[T]`, `Quantize14`/`FormatG14`,
  `RoundHalfUp`, `Tonumber`).
- **`SkillData` shape** → three maps (`Num`/`Flag`/`Str`); no
  `TriggerSource` bitset.
- **`data` package globals** (N6) → deferred; the 70-global block stays.
- **`data/raw` embedding** (G6) → prune: per-document embeds, drop
  `skillgemlist.json`, `mapuniquetofoulborn.json`, `statdesc.json` from the
  binary (statdesc stays on disk for `test/luarender/statdesc.go`).
- **`lua:"…"` struct tags** (B3) → do nothing; the tags stay as they are and
  step 14 loses that half.
- **Giant functions** (N2) → no split; `performBuffs`
  (`calc/performbuffs.go:88`) and `buildActiveSkillModList`
  (`calc/skillmods.go`) get an `#EVAL` tag naming their size, in step 15.
- **`TrimTreeDependentUniques`** (B4) → deleted; `gamedata_test.go` loads
  through a `data.Sources` option that skips the tree-dependent unique
  generation.
- **`modparser`↔`calc` cycle** (U25) → no cycle exists once the interface
  is typed honestly: no jewel closure passes a `Cfg` (`jewels.go:110,125,131`
  all pass `nil`), so `JewelStoreWriter.Sum` drops the cfg parameter and
  the interface references nothing in `modstore`. It stays in `modparser`,
  the 56 functions stay in `modparser/jewels.go`, `calc/radius.go` keeps
  its thin adapters. No `internal/contracts`. Recorded trigger: if a future
  jewel func genuinely needs a query config, `Cfg` moves to
  `internal/contracts` and both packages import it — that is the case the
  ownership principle (§6.9) reserves a contracts package for. Lands with
  step 5.
- **Error-return policy** → `Load`/decode paths return `error`;
  reference-error mirrors and unported guards keep `panic` with their
  comments; `Parse` returns `(mods []*Mod, extra string, recognised bool)`
  — an unrecognised line is an expected state (garbage item text), not an
  error.

Nothing remains open. Step 1 is ready to start on your word.
