# No Lua-shaped Go

The disease three remodel passes cured (go-remodel-plan.md, lua-gtfo.md,
lua-residue.md — adjudication vocabulary and precedents live there). Any new
code, port work included, follows these; a deliberate exception cites its
guarding differential at the site.

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
  anything a dump recorded, check whether the dump derived it (dump_calc ran
  under sortedPairs; "LuaJIT order" is usually just sorted).
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
