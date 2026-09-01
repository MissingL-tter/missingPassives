# missingPassives — Go rebuild

A ground-up Go rebuild of Path of Building, module by module, verified against
the Lua application under `.archive/`. Modules land only at 100% differential
agreement with the reference.

## modparser

A Go rewrite of `.archive/src/Modules/ModParser.lua` — the component that turns Path of
Exile's English modifier text into structured modifiers. Self-contained: it
takes nothing from ModParser.lua at runtime and keeps working when that file is
deleted. **All patterns are ordinary Go regular expressions**, written as
backtick literals in the table files and compiled once at start-up.

## modstore

A Go rewrite of `.archive/src/Classes/ModStore.lua`/`ModDB.lua`/`ModList.lua`
— the modifier containers and the EvalMod tag interpreter the calculation
engine queries. External reaches (calcLib gem lookups, item modifier search)
are narrow injected interfaces until their own modules land; `mergeKeystones`
(deferred from mod-parser) lives here.

**`TestModStoreAgainstReference`** rebuilds the reference's fixture world from
data the dump emits — the whole parsed corpus distributed over a store tree
with actors, multipliers, conditions, items and twelve configs — and compares
every aggregation, construction behaviour and `mergeKeystones` result, plus 59
synthetic mods covering the tag branches the corpus never produces. Scalars
compare exactly (%.17g), structures canonically. Where the reference errors on
out-of-contract input (untyped Tabulate over boolean values, StatThreshold
stat lists, source filters on source-less mods), the archive dump records an error
sentinel and the port must fail identically. **18,525 checks, 0
disagreements.**

## export

A Go rewrite of `.archive/src/Export/` — the tool that reads the game's
`.dat64` tables out of the GGPK and produces the game data the application
consumes. Its output is **structured JSON documents** (one per script, typed
by the `data/schema` package) — no Lua anywhere in the pipeline. `export/`
holds the dat64 reader, column schemas, the stat-description engine and 21 of
the 24 export scripts as document builders (`enums.lua` becomes
`export.WriteEnumFiles`; `legionSprites`, a GIMP sprite-sheet pipeline, is
excluded by decision; `uTextToMods` is a no-op in the reference, its itemTypes
list fully commented out); `cmd/pobexport` is the CLI:

```sh
# extraction stays with bun_extract_file.exe — see .archive/src/Export/ggpk/README.md
go run ./cmd/pobexport -src .archive/src/Export/ggpk -out <dir> [script ...]
```

**`data/schema/`** defines the documents: typed, JSON-tagged structs (costs,
mods, bases, skills, minions, uniques, stat descriptions, ...). This is the
data model the calculation engine will consume.

**`test/luarender/`** turns those documents back into the byte-exact
`Data/*.lua` files the reference Lua exporter produced. It exists only for
the differential test and is imported by nothing else; the serialisation
quirks of the reference (`%.14g` text, `pairs()` layouts, template
interleaving) live here, and the package is deleted whole when the archive
comparison stops being the contract.

**`TestExportAgainstReference`** builds every document over the extracted
GGPK, round-trips it through JSON, renders it back to Lua with test/luarender and
byte-compares all 123 files against the checked-in copies the Lua exporter
produced from the same game version. **123 / 123 agree.** Where the reference
leaks LuaJIT internals into its *data*, the builders replicate them exactly:
the default-seeded `math.random` stream (`test/luarender/luaprng.go`, baked into
LegionPassives layout offsets) and number-keyed `pairs()` iteration order
(`test/luarender/luatab.go`, baked into tradeHashes entry order; verified against
LuaJIT over 3,000 randomized tables).

## data

The runtime data set — the Go port of `.archive/src/Modules/Data.lua`.
`data.Load` assembles the package-level game data from the schema documents plus
the tables Data.lua defines inline, including everything the Lua computes at
load: combined mod pools, per-weapon-type enchant expansion, cluster-notable
lookups, boss stat means, item base lists, the full skill database (granted
effects with structured mods, template statMaps and hand fragments), gems
with their lookups and synthesized Vaal Alt gems, minions/spectres, and the
programmatically generated uniques. The hand-written `mod(...)` fragments in
templates are structured mods built at export and decoded by the modparser
codec — no Lua anywhere.

**`TestGameDataAgainstReference`** boots the archive application, dumps the
loaded `data` table subtree by subtree (`tools/dump_gamedata.lua` →
canonical serialisation, murmur-hashed for the large trees) and compares the
Go assembly against it: **136 subtrees, 0 disagreements.** Values that
round-trip through Lua string literals are unescaped at load; `[[...]]` long
brackets are raw. Where the reference's pairs() order decides a winner
(shared skill tables' mod sources, gem lookups), both sides re-derive in
sorted order — documented divergences. Lua functions carried by the data
(139 skill callbacks, 41 map-mod appliers) land with the module that consumes
them — the map-mod appliers are ported, in package `config`; the four tree-dependent generated
uniques land with tree-data.

## build

Assembles a `calc.BuildInput` from a saved Path of Building build file, and
closes the last gap between a saved file and a computed build.
`build.Load(xml, tree)` loads the item pool, the active passive spec, the
item sets and the skills tab through the packages that own them, constructs
`ItemsTab`'s 131-slot table, and resolves the passives item text names
rather than references by id (anoints, Forbidden Flame/Flesh).

**`TestBuildLoadAgainstReference`** loads every corpus build from its XML
alone and byte-compares the result against the archive's fixture: **6,157
slots across 47 builds**, plus header scalars, class stats, spectre list and
item sets. The calc differential runs on the same assembler.

The configuration tab comes through package `config`, which ports
`ConfigOptions.lua`'s 580 options and all 532 of their apply bodies, plus
`ModMap.lua`'s 41 map-affix appliers. **`TestConfigStateAgainstReference`**
compares the loaded option state against the archive — 2,667 values across
47 builds — and **`TestConfigModListCoverage`** compares the modifiers it
produces: **2,347 of 2,347 byte-identical, none produced that the archive
lacks**. The build corpus only sets about 32 of the 580 options, so
**`TestConfigOptionsAgainstReference`** drives the archive directly instead:
`tools/dump_config.lua` sets every option in turn on an otherwise-default
build and records what it produced. **1,254 cases, 73,824 comparisons, all
580 options, 0 disagreements.**

## Verification

Two differential tests, both of which **fail on any disagreement**:

- **`TestAgainstReference`** — every modifier line the application has ever
  parsed (the 13,173-line key set of `.archive/src/Data/ModCache.lua`) replayed through
  this port and compared byte for byte against what ModParser.lua produced.
  **13,173 / 13,173 agree.**
- **`TestTablesAgainstReference`** — every entry of every pattern table (8,800
  across 20 tables) compared canonically against the reference's own tables:
  data entries byte for byte, closure entries as agreeing that both sides hold
  a function there. Reference keys (Lua patterns) are mapped onto the regex
  keys via `test/luapat`. This covers the entries no corpus line reaches.
  **8,800 / 8,800 agree.**
- **`TestModToolsAgainstReference`** — the rest of modLib (the formatting
  family, `parseTags`, `parseFormattedSourceMod`, `compareModParams`,
  `setSource`, the deep-copying parse cache) replayed over every parsed
  modifier of the corpus: 12,736 mods x 7 behaviours.
  **0 disagreements.** `mergeKeystones` is deferred to the mod-store port,
  where its dependencies (live ModDB, tree keystone map) exist.

```sh
go test ./...        # test/ holds the differential harnesses
```

The archive dumps are frozen snapshots of the archive and stay valid after the Lua
is deleted. While it still exists they can be regenerated from `.archive/src/`:

```sh
luajit ../../tools/dump_parse.lua    # -> test/testdata/parse_archive.jsonl
luajit ../../tools/dump_modtables.lua   # -> test/testdata/tables_archive.jsonl
```

## Layout

| file | role |
|---|---|
| `modparser/mod.go` | `Mod`/`Tag` types, `mod()`/`flag()` (createMod semantics incl. nil-hole tags), the `scan` driver over compiled regex tables |
| `modparser/parse.go` | `parseMod` — the scan pipeline, per-form value resolution, tag combination, wrapper mods — and the public `Parse` |
| `modparser/globals.go` | ModFlag / KeywordFlag / SkillType from `.archive/src/Data/Global.lua` |
| `modparser/forms.go` `names.go` `modflags.go` `preflags.go` `tags.go` `smalltables.go` | the pattern tables — regex keys; `names.go`/`modflags.go` and the cost/suffix tables are literal-substring tables, exactly as the reference scans them |
| `modparser/special.go` | specialModList, converted mechanically and verified against the archive by both differential tests |
| `modparser/special_hand.go` | the ~70 specialModList closures needing real statements, hand-ported with source line citations |
| `modparser/modtools.go` | the rest of modLib: ParseTags / ParseFormattedSourceMod / CompareModParams / Format* / SetSource / CopyMod |
| `modparser/helpers.go` | grantedExtraSkill / triggerExtraSkill / extraSupport / explodeFunc / dealNoNonDamageType |
| `modparser/jewels.go` | radius-jewel node functions (getSimpleConv / getPerStat / getThreshold families) against narrow `modStoreWriter` interfaces, for the future calc-engine port. Keys are exact mod text; parametric entries are keyed by regex and wrapped in `jewelFactory` |
| `modparser/tables_build.go` | the data-driven construction loops (skill names, gem specials, keystones, cluster jewel skills) |
| `modparser/vocab.go` | the game vocabulary the loops consume (skills, gems, cluster notables, ailment defaults). Extracted one-time from the reference, Go-maintained; regeneration lands with the game-data module |
| `test/luacanon` | canonical serialiser, byte-compatible with `tools/canon.lua`; test-only |
| `test/luapat` | Lua-pattern → regex converter, used only by the table-archive test to map the archive's keys; deleted with the Lua |

## Public API

```go
mods, extra, recognised := modparser.Parse(line)
```

`recognised` is false when the line is not understood (an expected state for
garbage item text); `mods` is then nil. A recognised line that grants nothing
yields an empty slice, else a list of `*Mod` (and wrapper mods whose values
embed further mods). `extra` is the unconsumed remainder, "" when the whole
line parsed. Two-pass skill-name resolution and all reference semantics
included.

## Scan semantics

`scan` finds the best entry of a table within a line: earliest match wins, then
the longest match. A full tie (identical span) prefers the entry with more
pattern text **outside capture groups** — the more literal, more specific
variant — then the longer pattern. The reference broke full ties by Lua
pattern length; regex class syntax inflates lengths differently, and the
outside-groups weight preserves the intended specific-over-generic ordering
(the archive dumps pass under it).

## Semantics deliberately preserved

- `SkillType.Totem` does not exist in Global.lua; the one entry referencing it
  carries nil, exactly like the reference.
- A nil among a mod's tags leaves a numbering hole (`"2"` with no `"1"`),
  matching Lua table constructors; trailing nils vanish.
- Lua-side data oddities (a duplicated `"when you warcry"` key, `math.huge`
  durations, empty-string mod sources) are reproduced, not cleaned up.
- The reference anchors specialModList's literal entries with `^...$` and then
  adds the per-gem entries with their own partial anchoring; assembly order is
  identical here.

## Deliberate divergences

- Pattern tables iterate in sorted key order (the reference uses `pairs()`,
  which is arbitrary), so residual ties resolve deterministically.
- Duplicate cluster-jewel enchant texts resolve to the lexicographically
  greatest skill id in `modparser/vocab.go` (the reference's winner was
  whichever `pairs()` visited last); the table test flags any disagreement.

## #EVAL markers

Behavior that exists only to match the archive — reproduced bugs, undefined
globals read as nil, hash-order artifacts, LuaJIT internals — is tagged
`#EVAL` at the site that preserves it. Each is a candidate to fix or delete
once proven non-load-bearing (i.e. once its consumers are ported and the
archive comparison is no longer the contract). `grep -rn '#EVAL'` lists them
all.
