# calc-core port plan (working notes)

Target: `Modules/Calcs.lua` + `CalcSetup.lua` + `CalcPerform.lua` +
`CalcTools.lua` + `CalcFormat.lua`, PLUS `CalcActiveSkill.lua` (tracker
files it under calc-offence, but initEnv needs createActiveSkill/
buildActiveSkillModList — port it as part of the calc spine).
`defenceForConditionals` (CalcDefence) is called mid-perform and gets
pulled in early too.

## Status (2026-08-21)

DONE:
- `tools/dump_calc.lua`: fixture + allocOrder + dbs + skills records per
  variant (empty | coc.full/.noskills/.treeonly). Byte-stable across
  processes (verified 2x for both corpus keys).
- `calc/input.go`: BuildInput fixture boundary; `test/calc_test.go`
  TestCalcFixtureEcho — 4 fixtures decode + re-canon byte-identical, plus
  corruption negatives.
- `calc/tools.go`: CalcTools.lua ported EXCEPT
  canGrantedEffectSupportActiveSkill (needs ActiveSkill — skills stage).
- `calc/setup.go`: initModDB + initEnv through the tree merge.
- `calc/items.go`: the full items stage (L697-1283) — slot/jewel gating,
  jewel limits, disabler chains, flask/tincture tracking, special-case
  items (Aegis/Iron Mass/Dervish/Kalandra/Widowhail/Gloves/Boots),
  rarity/influence/socket multipliers, granted passives via dump-resolved
  nodes. Unreachable-in-corpus branches panic loudly: radius jewels,
  Energy Blade, corrupted-jewel-effect scaling, classRestriction (string
  condition — modstore Conditions is bool, widen when a corpus needs it),
  Item.FindModifierSubstring (needs itemTagSpecial; match with Go regex
  — luapat is build-time only).
  **TestCalcInitEnvAgainstReference: empty + coc.treeonly + coc.noskills
  post-initEnv modDB/enemyDB/itemModDB byte-identical, corrupted-input
  negative control.** Full suite green.
- `calc/skills.go` + `calc/activeskill.go`: the skills stage (L1349-1871)
  — weapon data (unarmed fallback, Hollow Palm), mainSocketGroup clamp,
  support gathering (ExtraSupport/imbued/crossLinked, applyGemMods/
  applySocketMods/addBestSupport, per-group cfg with gem-colour counts and
  snapshot semantics), createActiveSkill (skill types, rejected-support
  re-passes with Lua array-hole semantics, addFlags) +
  canGrantedEffectSupportActiveSkill. buildActiveSkillModList is a
  documented no-op (next stage; .skills summaries are final before it).
  Corpus-unreachable panics: granted-skill/explode socket groups
  (ProcessSocketGroup), Energy Blade re-entry, source-group supports.
  grantedEffect.fromItem runtime mark kept per-Env (env.geFromItemMark),
  not on the shared data tables like the reference — revisit if a dump
  leaks the mutation across variants.
  **TestCalcInitEnvAgainstReference: all 4 variants (empty, coc.treeonly,
  coc.noskills, coc.full) — dbs AND .skills summaries byte-identical.**
- `calc/skillmods.go`: buildActiveSkillModList ported (weapon flags,
  skill mod/keyword flag sets, skillCfg construction, support-level merges,
  gem/quality mods, stage/mine multipliers, SkillData extraction,
  GlobalEffect→buffList separation) + mergeSkillInstanceMods/mergeLevelMod
  (SORTED stat iteration — dump_calc.lua replaces the Lua function with a
  sorted-stats replica because pairs(stats) is string-hash-random per
  process; documented divergence). statMap lookups go through
  env.statMapLookup: the skillStatMapMeta lazy copy via
  data.LazyStatMapCopy, memoized PER ENV (not into the shared skill
  tables — keeps the game-data canon pristine; same values).
  Minion creation panics (needs minion stage + corpus).
  **`.skillLists` checkpoint (per-skill skillModList/cfg/flags/data/buffs/
  weapon cfgs) byte-identical for all 4 variants.**
  modstore.Externals (GemIsType, GetGameIdFromGemName) wired by InitEnv
  with the real calcLib implementations.
  Gotcha found: Lua Flag() returns nil (not false), so
  t_insert(explodeSources, flag-result) is a NO-OP for non-exploding
  nodes; and skillFlags is a set-of-true (never store false — the `or`
  chains keep nil).
- modstore: Store interface now exposes Sum/More/Flag/Override/List/
  Tabulate/TabulateAll/GetCondition/GetMultiplier; MergeKeystones takes
  Store (initEnv calls it on the tree ModList); Actor gained Level.
  modparser.NewMod exported (createMod); Mod gained SourceSlot (item slot
  mod lists carry mod.sourceSlot; canon emits it on both sides).

- **Corpus now comes from ../missingBuild** (mb: 10k real ladder chars,
  5.7k with stored PoB builds; `mb serve --detach` first). Pull with:
  `mb pob-run --char <name> --reprocess --pob <repo>/.archive --dump test/corpus`
  then `luajit tools/dump_calc.lua <key> ../../test/corpus/<name>.xml`.
  The dump's own asserts triage which branch a build needs. NOTE: ladder
  imports never carry spectreList/UI config — spectre builds contribute
  gear but no live minion; use fixed-minionList mains (zombies, Absolution,
  Holy Relic, Animate Guardian) for minion coverage.
- Corpus keys live: zombies (EssohNecro: Raise Zombie + Animate Guardian
  — exercises minion creation incl. item sets), lowlife
  (Higger_Bebrafication: low-life ES). Minion creation block PORTED
  (calc/skillmods.go Minion + item sets; createMinionSkills itself is
  perform-stage). .skillLists now also compares minionList + minion
  {type/level/hostile/weaponData}. Fixture gained spectreList,
  itemsTab.itemSets/itemSetOrderList.
- Go typed-nil trap hit once: Cfg.SkillGem (any) holding a nil *data.Gem
  defeats eval's nil check — assign only when non-nil.

- **Radius jewels PORTED** (calc/radius.go): funcList re-derived by
  parsing the item's mod lines through modparser.Parse (Item.lua joins
  continuation lines with a SPACE for parsing but stores \n — normalize),
  asserted against dump funcTypes; fixture carries per-socket radiusNodes
  (id→type) + radiusNodeData (unallocated in-radius nodeFixtures) +
  conqueredBy; per-call buildModListForNode sequences captured as
  `.nodeOrders` (the tail beyond allocOrders = extraRadiusNodeList pairs
  order). That order WAS process-random, making radius-build dumps unstable
  across runs; the sorted-pairs injection around LoadModule "Modules/Calcs"
  fixed it. A SECOND source survived until the EHP stage exposed it: the
  shim sorted numeric and string keys but left TABLE keys in raw order, and
  env.flasks/env.tinctures are sets keyed by item tables, so pairs() walked
  them in hash order over addresses — random per process. coc carries two
  flasks granting an identical Armour mod, so the dedup kept whichever
  merged first and `.performDbs` recorded a different mod `source` per run.
  It passed only by luck (the Go side sorts by item id and the dump happened
  to agree). Fixed by ordering table keys by item id when every key has one,
  matching calc.sortedFlasks. VERIFY REPRODUCIBILITY by dumping one source
  twice in separate processes and cmp-ing — not by re-running the Go test,
  which cannot see a dump that is stable-but-wrong. modparser exports
  JewelStoreWriter/JewelNodeRef/JewelNodeFn aliases.
- **Granted-skill socket groups PORTED** (skills.go): match-or-create
  groups from env.grantedSkills + processSocketGroup port (#EVAL: the
  reference's gemForSkill[skillId-string] lookup into a GE-keyed table
  never matches, so item-granted skills never resolve a gem; triggered
  cost wipe kept per-env in env.TriggeredCostWipes — perform must consult
  it). Source-group support gathering + forceSourceWeapon ported.
- Corpus now 6 sources / 16 variants: empty, coc, zombies, lowlife,
  spectre (gktiiS: Voidfletcher granted VoidShot, abyss eye jewels,
  threshold+transform radius jewels, Elegant Hubris), cyclone
  (COTA_Halbae: Light of Meaning, curse-on-hit ring).
- The corpus caught a REAL modparser port bug: getThreshold set
  inner value.mod Source without SourceSet, so the canon dropped it.
- skillLists nuance: early-disabled skills (default Melee with no weapon)
  return before minionList is set — the key is absent, not empty.

- **Explode groups, Forbidden Flame/Flesh, classRestriction,
  corrupted-jewel-effect scaling PORTED.** modstore Conditions widened to
  map[string]any (class-name strings). Fixture additions: gem
  explodeSource item/node refs, containJewelSocket slot flag,
  `.grantedAscendancyNodes` record; EnemyExplode works via
  data.materializeFullCustom (FullCustom skills get typed
  SkillTypes/BaseFlags/Stats/Levels/BaseMods views sharing the Custom
  tables; canon unchanged).
- **Perform-residue scrub**: the app's load-time calc mutates shared
  skill tag tables in place (warcryPowerBonus, CalcPerform L2330); the
  dump scrubs data.skills before emitting (documented divergence —
  perform recomputes it; extend the scrub list if more keys surface).
- Corpus now 8 sources / 22 variants: + rf (TwoGuysOneMirror: Chieftain
  RF, Glorious Vanity, The Adorned + corrupted jewels, Forbidden pair,
  explode sources), holyrelic (Waffle_Idol: Holy Relic minions, Fortress
  Covenant, Explode group).

- **Energy Blade PORTED**: InitEnv is now a pass loop (initEnvPass) —
  buildSkillsStage returns a restart request when an enabled Energy Blade
  gem is found, the pass restarts with override conditions and the order
  cursor carried forward (4 buildModListForNodeList calls per re-entering
  initEnv). The synthesized weapons come from the dump
  (`.energyBladeItems` per slot) instead of porting Item construction;
  no fixture entry = the info-nil/Bow fallthrough. Corpus key: eblade
  (DEE_FOUR_LOH_BAD, Inquisitor Storm Brand + spectres + Stone Golem +
  Animate Guardian + Brutal Restraint).

DONE: CalcPerform body (L1252-3718) 25/25 byte-identical on
.performDbs/.performOutput/.performMinionDb/.performMinionOutput.
Files: calc/perform.go (prologue through charges + reservations),
calc/performbuffs.go (buff/curse/link loop, curse slots, guards, buff
application, ailments, exposures, ally-life, GemLevel tail),
calc/performutil.go / performmisc.go / performminion.go /
performflasks.go (doActor* + flask/tincture merges).
FindModifierSubstring ported with plain Go regex (calc.patFind, cached);
Item.D carries the data tables and the fixture's ExplicitLines/OtherLines
are pre-filtered. It must NOT use internal/luapat: that package is
build-time only ("deleted together with the Lua"), and calling it from the
calc engine would pin it into the shipped runtime. Safe because the tables
use no syntax beyond a leading ^ / trailing $, and searchCond comes from a
letters-and-whitespace-only capture (ModParser.lua ItemCondition tags), so
no metacharacter can reach the matcher; test/itemtag_test.go guards that.
Perform traps found by the differential:
- skillData `or` chains: a PRESENT 0 wins (0 truthy in Lua) — never use
  `==0` as absence for skillData reads (reservation percents/Forced keys;
  "no reservation" effects write explicit 0s).
- Lua Flag() yields nil: `output.X = Flag(...)` stores no key when false
  (ChaosInoculation) — only write the key when true.
- Hex doom guard's trailing `and Sum(...,"MaxDoom")` is a bare number,
  always truthy (#EVAL) — HexDoomLimit/HexDoom set for every hex curse.
- Ailment gate first clause (`Val>0 or Sum(...)`) is always truthy
  (#EVAL) — only the immune/avoid clause gates.
- GemLevel/GemQuality output block (L3862+) runs AFTER the stubbed
  defence/offence handoff — part of the perform checkpoint.
- Ally-life (needsAllyLife) is corpus-reachable via Companionship support
  (lowlife); calcTotemLife still panics (no corpus totem+redirect build).
- Go test scrubs warcryPowerBonus tags after each Perform (mirror of the
  dump's per-variant scrub) so shared gamedata stays clean across variants.

Still panicking (no corpus build reaches them): Blight/Penance/EQ part-2
stage caches (getCachedOutputValue), calcs.resistances
(ManaIncreasedByOvercappedLightningRes), calcTotemLife, non-empty
AffectedByAuraMod (nil-global in reference), ProcessSocketGroup nameSpec
migration path, CALCS mode. Corpus: 9 sources / 25 variants.

DONE: calcs.defence (CalcDefence.lua L636-1632) + calcs.resistances,
25/25 byte-identical on .defenceDbs/.defenceOutput/.defenceMinionDb/
.defenceMinionOutput. Files: calc/defence.go (resistances + shared
helpers), defencebody.go (action speed, block, res-driven conversions),
defenceprimary.go (ward/ES/armour/evasion, evade chance),
defencerecovery.go (suppression, dodge, leech caps, regen, ES recharge),
defencetail.go (recoup, damage reduction, avoidance, self-ailment
duration/effect). Entry point: env.RunDefence() = player then minion,
back to back, matching the dump.

The dump reaches the stage by keeping the real calcs.defence in a local
before stubbing the handoff, then calling it explicitly AFTER the perform
records are emitted — so both checkpoints stay independent and can be
compared separately. Same trick extends to the offence stages.

Defence traps (same family as perform's, worth expecting again):
- Lua Flag() yields nil, so `output.X = Flag(...)` writes NO key when
  false: CappingES (an or-chain ending in a configInput read),
  Corrupted-Blood/Maim/Hinder/Knockback immunity, and
  EnergyShieldRechargeAppliesToLife (`Flag(A) and not Flag(B)` — absent
  when A is unset, but present-and-false when A is set and B is too).
  By contrast `not (...)` and `x ~= y` DO store real booleans.
- Party-tab branches (BlockChance/MaxBlock/MaxLifeLeech equal-to-party,
  MovementSpeedEqualHighestLinkedPlayers) panic: ladder builds never
  populate partyTab.
- Negative control verified: perturbing one output value fails all 25.

DONE: calcs.buildDefenceEstimations (CalcDefence.lua L1635-3828), 25/25
byte-identical on .ehpDbs/.ehpOutput/.ehpMinionDb/.ehpMinionOutput --
GREEN ON THE FIRST RUN, verified real by perturbing overkillDamage by 1e-7
inside reducePoolsByDamage (fails all 25). Files: calc/ehp.go (not-hit
chances, enemy damage input, taken-as shifts, taken multipliers),
ehphit.go (incoming hit mitigation), ehpstun.go, ehppools.go (life
recoverable, Petrified Blood, ES bypass, MoM), ehpguard.go (guard, aegis,
ally pools, Vaal Arctic Armour, total pools), ehpreduce.go
(reducePoolsByDamage), ehphits.go (numberOfHitsToDie), ehpehp.go (hit
counts, total EHP, survival time), ehprecoup.go, ehpmaxhit.go
(takenHitFromDamage + the quadratic max-hit solve with smoothing),
ehpdegen.go. Entry point: env.RunEHP().

That makes CALC-DEFENCE COMPLETE for the corpus (calcs.defence +
resistances + buildDefenceEstimations). PvP scaling panics (no corpus
build sets PvpScaling).

More #EVAL quirks in the EHP body:
- Max-hit pool fixup reads `shared or 0 + typed or 0`, which Lua parses as
  `shared or (0+typed) or 0` -- the typed guard rate/absorb is dead
  whenever a shared value exists.
- AnyTakenReflect is assigned false in BOTH branches ("this needs a rework
  as well"), so the reflect multiplier block never runs.
- Every pairs(damageTable) loop inside reducePoolsByDamage is
  order-independent (sum / per-key init / subtract-from-all / min), so the
  Go maps need no ordering; only the dmgTypeList walks are sequenced.

NEXT: calc-offence (CalcOffence + CalcActiveSkill + CalcMirages +
CalcTriggers, ~9.1k lines). Same staging: keep the real calcs.triggers /
mirages / offence in locals before stubbing, then call them explicitly
after the EHP records so each stage stays independently comparable.

## Dump gotchas (hard-won)

- HeadlessWrapper stubs `Inflate` to "" — without binding runtime/zlib1.dll
  the tree spec SILENTLY stays a default 1-node Scion (build codes are
  zlib). dump_calc.lua carries the same ffi binding as cook's pob.lua.
- `arg` is clobbered by the HeadlessWrapper boot — read arg[1]/arg[2]
  BEFORE dofile.
- ModList/ModDB objects carry actor/parent backrefs — canon their array
  part only (modArray) or canon.encode recurses forever.
- `pairs(env.allocNodes)` order is LuaJIT numeric-key hash order:
  deterministic per table STATE, NOT derivable in Go — and the table grows
  mid-initEnv (granted passives), so the two buildModListForNodeList calls
  (L672, L1319) can see different orders. The dump wraps
  calcs.buildModListForNodeList (dynamic lookup, so wrapping works) and
  records the order per call as `<variant>.allocOrders`; the replay pops
  one per call. Granted-passive nodes are dump-resolved from the tree maps
  into `<variant>.grantedPassiveNodes` (GrantedPassive values are
  lowercase names).

## Replay semantics found (CalcSetup)

- initEnv one-shot MAIN: specCopy is a pure cache copy (non-mutating) —
  replay skips it entirely.
- mergeKeystones at L673 targets env.initialNodeModDB, NOT modDB; keystone
  RESOLVED mods only reach modDB in perform (CalcPerform L1257 resets
  env.keystonesAdded = {} first, else the L673 marks would starve it).
  keystoneMap comes from the fixture (spec.keystoneMap: name → modList).
- The skill/support stages write NOTHING into modDB/enemyDB/itemModDB
  (verified by grep over CalcSetup L1335+ and CalcActiveSkill; the only
  write is enemy ActiveMineCount for mine skills) — so dbs parity for
  item-less variants doesn't need the skill spine.
- Items stage db writes are all item-guarded except the L1262-65
  overrideEmpty*Sockets config assignments (nil-assign no-ops without the
  config keys) and mergeDB(modDB, itemModDB) at L1283.
- `env.enemyLevel = build.configTab.enemyLevel or min(MaxEnemyLevel, charLevel)`
  reads configTab.enemyLevel (PROCESSED field, not input) — dumped as
  fixture `configEnemyLevel`.
- initEnv reads to port next: items loop details L697-1280 (read once, but
  port needs a line-level re-read); skills stage L1335-1871;
  CalcActiveSkill.lua createActiveSkill/buildActiveSkillModList.

## Architecture

Go package `calc`. Builds on: `modstore` (ModDB/ModList; NewMod =
modparser.NewMod), `data` (constants, skills, gems, minions), `modparser`
(Parse for pantheon/config lines, jewels.go radius funcs keyed by mod line).

**Fixture boundary** (`calc.BuildInput`): everything initEnv reads from
`build`, dumped per build by `tools/dump_calc.lua` from loaded
`.archive/src/Builds/*.xml`. Currently: characterLevel, classId,
configEnemyLevel, curClassName, treeVersion, mainSocketGroup, classStats,
configInput scalars, config/party mod list canons, spec (allocNodes with
modList/keystoneMod, keystoneMap, counts, masteryTypes/tattooTypes).
Items are in (`itemsTab`: slots + per-item payloads incl. mod-list canons,
scalar bags for flask/jewel/weapon/armour data, active mod-line texts for
FindModifierSubstring, modSource). Item mods carry `sourceSlot` — now a
first-class modparser.Mod field. STILL TO ADD for the skills stage:
socketGroupList + gem instances, imbuedSupportBySlot; socket counting in
items.go currently assumes zero socketed gems (true for every compared
variant) — wire it to the socket-group fixture when that lands.

**Comparison checkpoints**: post-initEnv dbs (live), post-initEnv skills
summaries (pending skills stage), post-perform dbs + output allowlist
(pending). The variant map in TestCalcInitEnvAgainstReference only grows:
empty, coc.treeonly → +coc.noskills (items) → +coc.full (skills).

## Corpus

Only 2 builds exist (.archive/src/Builds, CoC Assassin variants). Need
diversity (minion build, DoT, totem, aura stacker, low-life, MoM...) —
author via /cook or import before trusting coverage. Synthetic overrides
(config permutations) multiply coverage cheaply once the harness runs.

## Reading state (resume points)
- Calcs.lua / CalcTools.lua / CalcSetup.lua: fully read (CalcSetup items
  loop needs a line-level re-read at porting time).
- CalcPerform.lua / CalcActiveSkill.lua: structure only.
