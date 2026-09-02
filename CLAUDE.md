# First action, every prompt

Before anything else, read CLAUDE.md — this file and the global one. Every
prompt, no exceptions, including one-word replies and mid-task continuations.
Already having the rules in context is not the point; re-reading is the
checkpoint.

# Required reading

Read `.claude/documentation/knowledge.md` once per session, before the first
substantive change to code, artifacts or docs. It is the consolidated record
of how Path of Building works, how the differential method verifies the port,
and every trap that has already cost time — the traps are cross-cutting (one
float or iteration-order rule bites in calc and export alike), so scoping the
read per task means re-reading it per task.

Re-read the relevant section when entering a subsystem for the first time in a
session (§8 per package, §6 for the trap classes, §4 before touching a
harness). It is a reference to consult, not a checklist to recite.

Closing is a documentation event. What parity.md or knowledge.md records as
outstanding gets rewritten in the same turn it stops being outstanding — a
status written at discovery and never revised is read back later as fact,
including by you. later.md is an inventory of deliberate decisions, not a
queue: it goes stale when an entry is deleted, not when one completes.
calc-core-plan.md's statuses drift by design; parity.md and the tests are
current truth.

# No Lua-shaped Go

The disease three remodel passes cured (`.claude/documentation/deprecated/`:
go-remodel-plan.md, lua-gtfo.md, lua-residue.md — adjudication vocabulary and
precedents live there). Any new code, port work included, follows these; a
deliberate exception cites its guarding differential at the site.

- Port behaviour, never shape. `map[string]any` / `[]any` / `any` exist only
  at a true I/O decode edge (external JSON before it becomes typed). Closed
  key sets are structs or enums, not string-keyed maps; value unions are
  sealed interfaces (`modparser.Value`/`Tag` are the model); `util.Opt[T]`
  only where the reference distinguishes absent from zero AND an archive
  diff sees it — plain types everywhere else. Renaming, wrapping, or hiding
  a bag behind typed accessors is laundering, not typing.
- Lua semantics — truthiness, nil-vs-absent, string→number coercion,
  1-based indexing, `#`/ipairs length rules, pairs() order, shared-table
  mutation, tostring number spelling — enter production only when a named
  differential proves them load-bearing, evidence cited at the site.
  Otherwise parse/coerce at the edge and delete the helper. Before replaying
  anything a dump recorded, check whether the dump derived it (dump_build ran
  under sortedPairs; "LuaJIT order" is usually just sorted) — a fact about
  that file, not a requirement on the port and not a convention to extend.
- Never change a dump to make a comparison pass; the archive is the referee
  and an edited referee tests nothing. A comparison failing only on order is
  a defect in the comparison: compare as a multiset. Order changes a result
  only where the operation does not commute — increased sums, more
  multiplies; show the specific case before imposing an order (§4.6).
- Lua-shape understanding lives in test/luacanon, test/luapat,
  test/luarender only. Artifacts (data/raw, export/templates) carry typed
  conventional JSON — no Lua source fragments, pre-joined list text,
  directive strings, or raw text re-parsed at load; the exporter types at
  its edge, luarender re-spells the Lua form test-side.
- The producing package declares the type; consumers import it as-is — no
  narrow interface that every consumer downcasts back out of.
- Closed string domains get named types/consts. A type switch with a silent
  default or a discarded `ok` on an assertion is a bug: handle it or panic
  loudly. Panics are only reference-error mirrors ("(the Lua errors)") or
  unported-branch guards; load/decode paths return error.
- Production exports nothing that exists only for a test; fixture-only
  channels are named as such.
- A reference quirk reproduced deliberately gets `#EVAL` plus the test that
  guards it (README.md has the convention).
