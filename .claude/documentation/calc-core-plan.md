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

DONE: calcs.offence (CalcOffence.lua L323-6168) — 24 of 25 variants
byte-identical on .offenceDbs/.offenceOutput/.offenceSkillOutput/
.offenceMinionDb/.offenceMinionOutput. Files: calc/offence.go (damage-type
flag sets, calcDamage/calcAilmentSourceDamage, radius maths, conversion
table), offencebody.go (entry, calcAreaOfEffect, calcResistForType,
runSkillFunc, prologue), offenceskilldata.go (stat-conversion chain,
repeats, random phys, momentum), offencetypestats.go (minion limits,
chain/projectile/pierce, melee range, aura/curse/warcry/link, reservation
mults, trap/mine/totem/brand/corpse), offenceduration.go, offencecosts.go,
offenceconv.go (explosions, canDeal, per-cfg conversion tables, passList,
combineStat), offencehitrate.go (accuracy, speed, trauma), offencemisc.go,
offencewarcry.go (exerts, pacts, ruthless), offencecrit.go (crit, double/
triple damage, culling, Cryogenesis redirect, base hit damage),
offencedamage.go (the two-pass damage loop, resists, leech), offencedps.go
(per-pass DPS, combine, leech rates), offenceailments.go + offencebleed.go
(affliction chances, bleed/poison/ignite, non-damaging ailments, knockback,
stun, impale), offencedot.go (decay, burning ground, generic DoT, cost per
second, combined DPS), offenceselfhit.go (+ applyDmgTakenConversion).
Entry point: env.RunOffence() = player then minion, matching the dump.
Negative control: +1e-7 on AverageHit fails all 24.

DONE: calcs.triggers + calcs.mirages, closing CALC-OFFENCE at 25/25 with
NO slice skipped. Files: calc/triggers.go (helpers, findTriggerSkill, the
1000-attack calcMultiSpellRotationImpact rotation sim, the RunTriggers
entry), triggerhandler.go (defaultTriggerHandler), triggerconfig.go (the
80-key configTable dispatch), mirages.go, globalcache.go.
Checkpoints: .triggersDbs/.triggersOutput/.triggersSkillData and the new
.triggersMinionOutput/.triggersMinionSkillData; the test now runs the exact
dump sequence RunTriggersPlayer -> RunOffencePlayer (mirage-gated) ->
RunTriggersMinion -> RunOffenceMinion.
Negative control: +1e-9 on the simulated trigger rate fails exactly 7
checkpoints -- coc.full triggersOutput/triggersSkillData/offenceOutput/
offenceSkillOutput and holyrelic.full triggersMinionOutput/
triggersMinionSkillData/offenceMinionOutput -- proving both that the
trigger checkpoints are load-bearing and that offence consumes the rate.

GLOBALCACHE IS A DUMP FIXTURE (.globalCache), not something the ported
stages compute. Earlier plan text had the ordering wrong: it assumed the
cache had to be filled by porting Calcs.lua buildOutput first. In the dump
the cache is already populated by the app's own OnFrame buildOutput, and
the dump's manual calcs.perform then OVERWRITES the main skill's entry with
a pre-offence one (Speed nil, because offence is stubbed). Triggers reads a
bounded field set -- enumerate it with
  grep -oE "GlobalCache\.cachedData\[env\.mode\]\[[a-zA-Z]+\]\.[A-Za-z.]+" CalcTriggers.lua
-- so the dump snapshots exactly those, immediately before realTriggers.
A cache miss in the Go replay panics rather than inventing a value.

Trigger traps:
- `ignoresTickRate = ignoresTickRate or (storedUses and storedUses > 1)`
  writes a REAL false when storedUses is present but 1; only an absent
  storedUses leaves the key absent. Cost two variants until fixed.
- `triggeredSkills[1] == packageSkillDataForSimulation(...)` compares two
  freshly built tables, never true, so that arm reduces to
  `ignoresTickRate and not config.triggeredSkillCond` (#EVAL).
- `actor.mainSkill.triggeredBy.ignoresTickRate` is dead: nothing in the
  whole archive sets that field (#EVAL).
- The triggerCD read guards on actor.mainSkill.triggeredBy but reads
  env.player.mainSkill.triggeredBy (#EVAL; same table for the player).
- ProcessSocketGroup's triggered-cost wipe (SkillsTab.lua L1161) assigns
  `cost = {}`, not nil, and mutates the SHARED granted-effect level.

Guarded, not ported (loud panic, no corpus build reaches): 78 of the 80
trigger configs, helmetFocusHandler, CWCHandler, the Arcanist Brand /
Kitava's Thirst / Battlemage's Cry / Infernal Cry / Manaforged / Doom Blast
/ stagesAreOverlaps branches of defaultTriggerHandler, and all 5 mirage
paths. Widening the corpus is what would shrink that surface.

Hand-written skill callbacks now live in calc/skillfuncs.go, keyed
"<grantedEffectId>:<callbackName>", consulted by runSkillFunc before the
UnportedFn panic. Ported so far: Cyclone/CycloneAltX/VaalCyclone
initialFunc, BloodSacramentUnique initialFunc, EnemyExplode preDamageFunc,
StormBrand preDamageFunc, RighteousFire preDamageFunc.

Offence traps found by the differential:
- `Sum(...) or 0 + X + Y` parses as `Sum(...) or (0+X+Y)`, and Sum always
  returns a number, so TripleDamageChance drops the enemy and on-crit
  terms entirely (#EVAL).
- `(Flag and 100 or 0) + Sum(Avoid...)` parses as
  `Flag and 100 or (0 + Sum(...))`: an immune enemy yields exactly 100 and
  the avoid sum is dead (#EVAL).
- `skillModList:More("MORE", cfg, "Accuracy")` puts "MORE" in the cfg slot,
  so the real cfg becomes a never-matching modifier name — that Accuracy
  More is cfg-less. Ported as More(&Cfg{}, "Accuracy"): a non-nil EMPTY
  cfg, because ModStore:EvalMod distinguishes `not cfg` from a cfg whose
  fields are all nil (#EVAL).
- `cfg` in the Mantra of Flames BuffOnSelf lines is an undeclared global
  (the pass local died with the loop above), i.e. nil (#EVAL).
- `dotCfg` in the hit-damage resistance lookup is likewise a global; only
  the ailment sections declare a local of that name (#EVAL).
- `ipairs({["FRDamageTaken"] = ...})` iterates zero times, so the Forbidden
  Rite self-hit block never runs (#EVAL).
- ModStore:GetStat is an `or` chain: a stored FALSE in actor.output falls
  through to cfg.skillStats, a stored 0 does not. Cfg.SkillStats had to
  widen to map[string]any so the weapon passes can alias output.MainHand /
  output.OffHand live.
- The bleed/poison/ignite dot cfgs give skillCond a metatable falling back
  to skillCfg/cfg. Only "CriticalStrike" is mutated after construction and
  the dot table shadows it, so a snapshot copy is exact.
- ProcessSocketGroup's triggered-cost wipe (env.TriggeredCostWipes) has its
  first consumer here: the cost section must not read a wiped level's cost.
- calcAreaOfEffect is DEFINED at L345 but first CALLED at L1152, after the
  skill-type-stats section sets actor.weaponRange1.

- The dump keeps calcs.triggers / mirages / offence in locals before
  stubbing and calls them explicitly after the EHP records. Triggers gets
  its OWN records (.triggersDbs/.triggersOutput/.triggersSkillData) ahead
  of the offence ones, so it can be ported and verified on its own instead
  of only becoming testable once all ~5.8k lines of offence exist.
  All 9 sources dump cleanly with the real offence running, so no corpus
  build makes the reference error in this stage.
- TRIAGE, calcs.triggers: it is a NO-OP for 24 of the 25 variants (DBs and
  output both unchanged). Only coc.full — the actual Cast on Critical
  Strike build — has a live trigger path, and it writes just four output
  keys (EffectiveSourceRate, SkillTriggerRate, Speed, TriggerRateCap) plus
  skillData.triggerRate / triggerSourceUUID. So the corpus needs the entry
  gate, the configTable dispatch, and the CoC branch of
  defaultTriggerHandler — not the whole module.
- BLOCKER for that one path, and the ORDER IT IMPLIES: findTriggerSkill
  and defaultTriggerHandler read GlobalCache.cachedData[mode][uuid]
  .HitSpeed/.Speed — the SOURCE skill's speed. coc.full's trigger rate of
  7.56756 is Cyclone's cached speed.
  GlobalCache itself is trivial: Data/Global.lua L355 declares
  `cachedData = {MAIN={}, CALCS={}, CALCULATOR={}}`, and Common.lua
  cacheData() stores ~12 output values (Speed, HitSpeed, the costs, crit,
  TotalDPS) per uuid. cacheSkillUUID is name_SLOT_gemIdx_groupIdx.
  What is NOT trivial is FILLING it. CalcPerform L3918 calls cacheData at
  the very END of perform, and Calcs.lua buildOutput (L202) drives it: for
  every skill with includeInFullDPS it sets fullEnv.player.mainSkill and
  runs a whole calcs.perform, so each skill caches its own final output.
  Those values come from calcs.offence.
  => The cache cannot be populated before offence exists. Correct order is
  (1) calcs.offence, (2) the buildOutput driver loop + cacheData, then
  (3) calcs.triggers. Porting GlobalCache "first" is not possible; it is a
  consequence of offence, not a prerequisite.

- calcs.offence section map (L323-6168): prologue/AoE 323-493, skill data
  494-1022, skill type stats 1023-1456, duration 1457-1625, uptime
  1626-1658, costs 1659-1841, conversion 1878-1933, damage passes
  1934-2110, hit rate 2111-2421, misc DPS 2422-2540, MAIN DAMAGE 2541-4009,
  leech 4010-4064, AILMENTS 4065-5554, secondary 5555-5675, DoT 5676-5863,
  self hit 5864-6010, combined DPS 6011-6168.

NEXT (calc-core, not calc-offence): the Calcs.lua buildOutput driver +
cacheData. That is what fills GlobalCache for real; until it exists the
cache stays a dump fixture. Porting it also removes the last reason
calc/globalcache.go has to be a fixture boundary, and is a prerequisite for
CALCS mode and getCachedOutputValue. Then CalcFormat.lua.

DONE (2026-08-22): corpus 12 -> 25 builds / 75 variants, all byte-identical.
Guard-driven: pull a build for a panic (`mb search`), dump it, port until
green. Closed this pass: Explosive Trap random radius + preDamageFunc,
Explosive Arrow explosiveArrowFunc, Blade Blast / Ice Spear of Splitting /
Lightning Tendrils of Eccentricity (preDamage + postCrit) / Herald of the
Breach / Penance Brand of Dissipation callbacks, the "cast when damage
taken" and "TriggeredMoltenStrike" trigger configs, the jewel-func AddList
ModList argument, and TWO WHOLE MIRAGE PATHS (Mirage Archer, Sacred Wisps)
-- which meant porting calcs.copyActiveSkill, CALCULATOR-mode initEnv and
the nested calcs.perform.

Mirage sub-environments:
- copyActiveSkill runs createActiveSkill FIRST, then a whole second
  initEnv (mode CALCULATOR, inheriting env.override), then
  buildActiveSkillModList. CALCULATOR differs from MAIN only in skipping
  write-backs that exist for the UI (node.finalModList, displayEffect,
  displayLabel, jewelRadiusData, superseded, group.mainActiveSkill);
  `superseded` is read only by SkillsTab, so nothing behavioural.
- That second initEnv consumes its own buildModListForNodeList orders, so
  the dump records them separately (`.mirageAllocOrders`/`.mirageNodeOrders`)
  and ReplayInput carries them.
- The dump's stubbed handoff applies to the nested perform too, so the
  sub-env's output is the perform-body state -- exactly what Go's
  env.Perform() produces. Compared as `.mirage` (name/count/skillPart) and
  `.mirageOutput`.
- `mainSkill.ModFlags` / `.KeywordFlags`, which every mirage path passes to
  NewMod, are never assigned anywhere in the archive: both are nil.

THREE HARNESS BUGS FOUND, each hidden behind a formatting or folding step:

1. `_grantedEffect` is process-random for a SHARED statMap. Data.lua:1039
   sets `grantedEffect.statMap._grantedEffect = grantedEffect` inside a
   `pairs(data.skills)` loop; ExplosiveTrapAltX aliases ExplosiveTrap's
   statMap table, so the backref is last-writer-wins over a random order --
   and the lazy statMap copies then stamp the winner as the mod source even
   for the other skill. Two dumps of the trap build differed in exactly one
   byte range. Fixed by settling it in sorted id order in dump_calc.lua
   (matching dump_gamedata's reassign pass) and modelling it in Go as
   GrantedEffect.StatMapOwner, which LazyStatMapCopy stamps with.
   ALWAYS re-verify a new dump twice with cmp.

2. Fixtures were serialised at %.14g, which is lossy. The cospri build's
   trigger sim ran 1000 attacks off the cached source rate; the fixture's
   10.063177748344 and the reference's true 10.063177748344373 are the same
   14 digits but land on either side of the loop's `<` bound, so the sim
   counted 1000 triggers instead of 1001 and the trigger rate came out
   0.1% low. canon.encodeExact (%.17g) now serialises replay INPUT; the
   compared canons stay at 14 digits on both sides. Only re-dumped builds
   carry exact fixtures; re-dump when a build shows an unexplained tail-digit
   divergence.

3. Go folds untyped constant expressions at arbitrary precision. `1 / 0.033`
   as a Go constant is 30.303030303030305; Lua's runtime division is
   30.303030303030301. data.Misc.ServerTickRate fed a `m_ceil(x * rate)`
   tick rounding, so Absolution's 3.3s duration ceilinged to the next tick
   in Go and not in the reference. Fixed with `1 / float64(0.033)`, which
   forces the typed-constant rounding. Swept the rest of data/tables.go --
   no other constant expression differs from its runtime form.

DONE (2026-08-25): Calcs.lua's cache half -- cacheData, buildActiveSkill,
the buildOutput cache fill (calc/buildoutput.go), PerformFull with the real
per-actor handoff. **GlobalCache is COMPUTED now, not a fixture**: the test
builds it with the driver and compares it against the archive's `.globalCache`
snapshot (dumpVariant runs wipeGlobalCache + calcs.buildOutput with the real
stage functions restored, exactly as Build.lua:675 does). 114/114 variants
across 38 builds byte-identical, zero guards reachable from the corpus.

What the driver forced out (each verified by the cache canon):
- The recursion breaker at the TOP of calcs.triggers (L1604): skip the stage
  when env.limitedSkills[mainSkill uuid]. It is what stops Manaforged Arrows
  (which builds its own skill to learn its mana cost) from recursing. Its
  absence in the port recursed unboundedly -- a full Env per level, ~100GB
  commit before it was killed. BuildActiveSkill now also carries a
  DEFENSIVE depth-20 panic with no reference counterpart.
- CachedSkill accessors are LIVE through the stored Env/ActiveSkill, as in
  the reference: a stage that runs after cacheData changes what the entry
  appears to hold. The first port snapshotted them and diverged.
- Nested performs inherit the handoff state: `calcs.perform(newEnv)` inside
  a mirage runs whatever calcs.defence/offence currently are -- stubbed
  no-ops during the dump's checkpoint phase, the real stages inside the
  driver. Env.StubHandoff models that; the test sets it on checkpoint envs.
  Missing it cost voidstorm's Sacred-Wisps mirage its TotalDPS (and the
  cache canon caught the missing MirageDPS -- the offence mirage-DPS tail
  is ported now too).
- A THIRD pairs() nondeterminism, same shape as the flasks and the statMap
  backref: the granted-skill support gather iterates supportLists[slotName]
  keyed by GROUP TABLES; and the shared-statMap mod SOURCES are stamped
  last-writer-wins under pairs(data.skills). Both settled: the sortedPairs
  shim orders group keys by socketGroupList position (dumpBuild global), and
  dump_calc re-stamps shared mod sources in sorted id order, mirroring
  dump_gamedata's reassign. ALWAYS cmp a re-dump twice.
- Every initEnv of one build yields identical alloc orders (dump-verified),
  so sub-environments replay the top env's orders, and mirageReplay falls
  back to them when the mirage only runs inside a driver build.
- test/corpus/manifest.tsv maps every dump key to its build XML (the five
  archive-Builds/ ones included); MP_ONLY narrows the test, MP_GUARDS turns
  guard panics into collected failures, MP_DUMPGC dumps the cache canons.
- Fixtures are exact end to end now: luacanon.EncodeExact mirrors
  canon.encodeExact and the fixture echo round-trips at %.17g.

Now unblocked: getCachedOutputValue (Blight part 2 guard), CALCS mode,
buildOutput's display half (FullDPS roll-up, cost warnings, config
discovery).

OPEN, POST-PARITY: the compared canons are `%.14g` on both sides, so the
differential is blind below the 15th significant digit -- exactly the band
harness bug 2 lived in. A green run does not rule out sub-ulp drift that a
later comparison bound amplifies. The review is: re-run the corpus with the
canons at `%.17g` on both sides. Until then treat every `archive [x]` as
"agrees to 14 digits". Two loose ends feed it: `coc`/`cocuser`/`dualstrike`/
`bfbb`/`empty` still carry `%.14g` FIXTURES (their XML is outside
test/corpus; the other 33 were re-dumped exact and no compared canon moved),
and the constant-folding sweep was a regex over data/ and calc/, not a proof.

Test-isolation note: processMod sets `ge.HasGlobalEffect` on the granted
effect it is passed, so a calc run mutates the SHARED data set (faithfully
-- the reference mutates its own global tables the same way, and
calc/skills.go:551 reads the flag). The game-data canon is the post-load
state, so gamedata_helper_test records hasGlobalEffect at load and
TestGameDataAgainstReference restores it before comparing.

MP_ONLY=<variant prefix> narrows TestCalcInitEnvAgainstReference to one
build, so an unrelated variant's guard panic cannot pre-empt the one being
diagnosed.

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
