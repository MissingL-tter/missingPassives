# Authorization

- A question is not a task. When my message asks why, whether, what if, what would happen,
  or how something works — "why not X" included — answer it and change no code. Reads,
  `go test`, and the comment and documentation fixes under **Fix on sight** stay free while
  answering.
- Before your first change to code in a turn, quote the words from my message that told you
  to make it. Cannot quote them: you are not authorized, so ask. When your edits reach a
  file the quote does not describe, say so before making that edit.
- Changing code needs an explicit go in my immediately preceding message — every statement
  in a `.go` file and every line of a `.lua` harness in `tools/`. Comments and
  documentation are not code; the corrections under **Fix on sight** are the only free
  documentation edits — any other change to a document, this file included, needs a go.
- Never write to any path under `.archive/`. Builds you author for testing go in
  `test/corpus/` as `authored_*.xml`.
- Deleting or truncating a committed file needs an explicit go naming that file.
  Regenerating one in place does not.
- `go run ./cmd/pobexport`, `go run ./cmd/sourceupdate` and `go run ./cmd/treegen` each
  need an explicit go. Changing exporter or tree code is not a reason to run them: verify
  that with `MP_EXPORT=1 go test ./test -run TestExportAgainstReference`.
- Never edit `README.md`; I decide when it changes. When a change makes a count in it
  stale, say so in your report and leave the file alone.
- Git writes — `add`, `commit`, `push`, `checkout`, `reset`, `stash`, `submodule update` —
  need an explicit go naming that action. Never draft or offer a commit message, and never
  say "ready to commit", unless I ask for one.
- Never commit anything under `Builds/`, `*.cfg`, `Settings.xml`, `inspect.lua`, or
  `.claude/skills/cook/recipes/` other than `template.txt`. A regenerated
  `test/testdata/*.jsonl` or `data/raw/*` goes in the same commit as the change that caused
  it, never a separate "update fixtures" commit.

Free without asking: any read or search; any read-only `git` command; `go build ./...`,
`gofmt`, `go vet`, `go test ./...` and any `-run` or `MP_*` subset; regenerating
`test/testdata/*.jsonl` with a `tools/dump_*.lua` harness; files under the session
scratchpad.

Before your first change to code in a session, read `.claude/documentation/knowledge.md`
§6 (traps) and §11 (standing decisions).

# Evidence

- Never state a cause, a count, or an absence you have not measured. A grep, a read, or a
  plausible mechanism is not a measurement: run the check that would fail if you were
  wrong, paste its output, then state the conclusion. Not run: write "not measured" and
  name the check that would settle it.
- State a measurement as a count over a scope — `93,855 calls, 0 divergences over the
  48-build corpus`. A number without the scope it was taken over is not a measurement.
- Before applying a fix, write which layer the defect is in and the observation that places
  it there. A fix that makes the symptom disappear without that statement is not finished.
- "Green" means `go test ./...` completed in this turn with no failures. Report it by
  quoting the differential's own counts — `18,525 checks, 0 disagreements` — never the bare
  word "passing". After a `-run` or `MP_ONLY` subset, write "subset only:" and the exact
  command, and do not call it green.
- When you add a fixture record, or claim one covers a branch, print its value in every
  config and paste them. A record that is 0 or empty in every config proves nothing: feed
  its input until it produces a non-zero result, or state in chat that it is a negative
  control. "Both sides agree" is compatible with "both sides did nothing".
- Cite `file:line` only for a line you read in this session. When repeating a citation from
  `knowledge.md`, `parity.md`, `later.md` or a memory, re-locate the code by name first and
  cite where it is now; when it has moved, correct that document in the same turn.

# When a differential fails

- Write which of three is wrong — the Go port, the comparison, or the `tools/dump_*.lua`
  harness that recorded the reference — and the observation that places it there, before
  editing any of them.
- A failure only on ordering is a defect in the comparison: compare as a multiset. Impose
  an order only after showing the specific case where the operation does not commute.
- Never change a `tools/dump_*.lua` harness or a `test/testdata/*.jsonl` fixture to make a
  comparison pass. Change them only to record more of what the archive does.
- Never add a shape to product code because a harness wants it. `data/cluster.go`'s
  `sort.Strings(info.JewelTypes)` was added for that reason and survives only because it is
  now justified on its own.

# Fix on sight

- Correct any comment the code contradicts, and any line in `.claude/documentation/` the
  code contradicts, on any turn, as long as no statement changes. List every such edit as
  its own line in your report.
- On finding a defect in product code outside the task, report `file:line` and what you
  observed; do not fix it, because changing code needs a go.

# Typed Go, not Lua in Go

Do not write Go that thinks in Lua tables.

- Decode external bytes into structs. `map[string]any` and `[]any` stay inside the decode
  function — today `export/dat.go`, `export/treegen.go`, `modparser/modcache.go`,
  `modparser/tag.go`, nowhere else. Adding a decode site to a file outside that list needs
  an explicit go.
- Closed key sets are structs or enums; value unions are sealed interfaces
  (`modparser.Value`, `modparser.Tag` are the model). Renaming a bag, or hiding one behind
  typed accessors, does not satisfy this.
- A type switch with a silent default, or an assertion whose `ok` is discarded, is a bug:
  handle the case or panic.
- `util.Opt` wraps a number only where the input itself has an empty state (an empty
  config box, an item line without `{range:}`, no item in the slot) and a reader answers
  differently for empty than for zero; never a bool — a missing bool key compares as
  false (`luacanon.FalseIsAbsent`). A differential seeing a missing key is not by itself
  a reason.
- Reproduce a Lua semantic — truthiness, nil-vs-absent, string→number coercion, 1-based
  indexing, `pairs()` order, `tostring` number spelling — in product code only with the
  differential that proves it load-bearing cited in a comment at that site.
- Lua-shape code lives in `test/luacanon`, `test/luapat`, `test/luarender`, nowhere else.
  Artifacts under `data/raw/` and `export/templates/` carry conventional JSON: no Lua
  fragments, no pre-joined list text, no strings re-parsed at load.
- The producing package declares the type and consumers import it; do not add a narrow
  interface that every consumer downcasts back out of.
- Panics mirror a reference error or guard an unported branch; load and decode paths return
  an error.
- Export nothing from a product package that exists only for a test.

# Documents

- When my message states a rule this file does not cover, or one that contradicts a line in
  it, say so in that turn: name the line it collides with, or say the file is silent,
  propose the exact replacement text, and wait for a go before writing it.
- When a change makes a `parity.md` row or a `knowledge.md` statement false, rewrite it in
  the same turn.
- A `parity.md` row you change states the count that justifies its mark. Do not downgrade a
  row to `[~]` in place of correcting its number.
- `.claude/documentation/later.md` is my inventory of deliberate decisions. Correct an
  entry that has become false; never add your own unfinished or deferred work to it, and
  never file a defect there — a defect goes in your report as `file:line`, not in a
  document.
- `calc-core-plan.md` statuses drift by design. Where a document and the tests disagree,
  the tests and `parity.md` are right.

# Scope

Continue while the extra work is plainly part of what I asked for. The moment it is not,
stop and ask, naming the file and the change. This one is judgment: unsure whether it is
still plainly included means it is not — ask.

# Reporting a change

- The verbatim command that verified it, and the differential's own counts.
- What you did not verify — the export differential when it did not run, packages with no
  tests, a branch no corpus build reaches — and nothing at all here when the full suite ran
  and covers the change.
- Each comment or documentation line you corrected on sight, one per line.

# Conflicts

Fix on sight covers comments and `.claude/documentation/`; where it appears to reach a
statement in a `.go` or `.lua` file, the code gate wins. Those two exceptions override the
global rule that production-source edits need a go — nothing else here loosens it.

# Reference

```sh
go build ./...
go test ./...                                              # ~90s, the committed-fixture differentials
MP_EXPORT=1 go test ./test -run TestExportAgainstReference # 123 files, needs the GGPK, ~100s
cd .archive/src && luajit ../../tools/dump_<name>.lua      # every harness hard-codes ../../tools/
go run ./cmd/pobexport -src .archive/src/Export/ggpk -out data/raw   # gated; a partial run leaves data/raw incomplete
go run ./cmd/sourceupdate                                  # gated; -modcache-only for just the cache
```

`MP_ONLY`, `MP_ONLY_ITEM`, `MP_ONLY_SKILLS`, `MP_ONLY_SPEC`, `MP_ONLY_BUILD`,
`MP_ONLY_CONFIG`, `MP_ONLY_OPTION` narrow to one build or option. `MP_GUARDS` reports
unported-branch panics instead of raising them, so one run enumerates the whole guard
surface. `MP_FIXTURE=1` bypasses the native bridge. `MP_DUMPGC=<path>` writes the computed
and expected global caches for diffing. `MP_NODRIVER=1` skips the cache-filling driver.

`.claude/documentation/`: `parity.md` per-module port state · `knowledge.md` the
differential method, the traps, the standing decisions · `later.md` my inventory ·
`poe-data-model.md` PoE domain semantics, required before `.claude/skills/cook` work ·
`lua-go-map.md` reference-file to package map · `deprecated/` the three remodel passes.
