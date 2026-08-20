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
  keys via `internal/luapat`. This covers the entries no corpus line reaches.
  **8,800 / 8,800 agree.**

```sh
go test ./...        # test/ holds the differential harnesses
```

Both oracles are frozen snapshots of the reference and stay valid after the Lua
is deleted. While it still exists they can be regenerated from `.archive/src/`:

```sh
luajit ../../tools/dump_oracle.lua    # -> test/testdata/oracle.jsonl
luajit ../../tools/dump_tables.lua    # -> test/testdata/tables_oracle.jsonl
```

## Layout

| file | role |
|---|---|
| `modparser/mod.go` | `Mod`/`Tag` types, `mod()`/`flag()` (createMod semantics incl. nil-hole tags), the `scan` driver over compiled regex tables |
| `modparser/parse.go` | `parseMod` — the scan pipeline, per-form value resolution, tag combination, wrapper mods — and the public `Parse` |
| `modparser/globals.go` | ModFlag / KeywordFlag / SkillType from `.archive/src/Data/Global.lua` |
| `modparser/forms.go` `names.go` `modflags.go` `preflags.go` `tags.go` `smalltables.go` | the pattern tables — regex keys; `names.go`/`modflags.go` and the cost/suffix tables are literal-substring tables, exactly as the reference scans them |
| `modparser/special.go` | specialModList, converted mechanically and verified by both oracles |
| `modparser/special_hand.go` | the ~70 specialModList closures needing real statements, hand-ported with source line citations |
| `modparser/helpers.go` | grantedExtraSkill / triggerExtraSkill / extraSupport / explodeFunc / dealNoNonDamageType |
| `modparser/jewels.go` | radius-jewel node functions (getSimpleConv / getPerStat / getThreshold families) against narrow `modStoreWriter` interfaces, for the future calc-engine port. Keys are exact mod text; parametric entries are keyed by regex and wrapped in `jewelFactory` |
| `modparser/tables_build.go` | the data-driven construction loops (skill names, gem specials, keystones, cluster jewel skills) |
| `modparser/vocab_gen.go` | GENERATED from `.archive/src/Data` by `tools/gen_vocab.lua` — the game vocabulary the loops consume. Regenerate each league |
| `modparser/canon.go` | canonical serialiser, byte-compatible with `tools/canon.lua` |
| `internal/luapat` | Lua-pattern → regex converter, used only by the table-oracle test to map the reference's keys; deleted with the Lua |

## Public API

```go
mods, extra := modparser.Parse(line)
```

`mods` is nil when the line is not understood, an empty slice when it is
recognised but grants nothing, else a list of `*Mod` (and wrapper mods whose
values embed further mods). `extra` is the unconsumed remainder, "" when the
whole line parsed. Two-pass skill-name resolution and all reference semantics
included.

## Scan semantics

`scan` finds the best entry of a table within a line: earliest match wins, then
the longest match. A full tie (identical span) prefers the entry with more
pattern text **outside capture groups** — the more literal, more specific
variant — then the longer pattern. The reference broke full ties by Lua
pattern length; regex class syntax inflates lengths differently, and the
outside-groups weight preserves the intended specific-over-generic ordering
(both oracles pass under it).

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
  greatest skill id in `tools/gen_vocab.lua` (the reference's winner was
  whichever `pairs()` visited last); the table test flags any disagreement.
