# calc-core port plan (working notes)

**Superseded in parts by the luagtfo work (`deprecated/`, 2026-08-29/30/31).**
The reasoning and the reference findings below stand; these mechanisms named
throughout the document do not, and `knowledge.md` carries the current form:

| this document says | current |
|---|---|
| the dump's `allocOrders`/`nodeOrders`/`mirage*Orders` are replayed because LuaJIT hash order is not derivable in Go | false premise: `dump_build.lua` installs `pairs = sortedPairs` before the Calc modules load, so every recorded order is ascending node ids. The whole replay machinery is deleted; production calls `sortedIntKeys` and the test asserts each recorded order is ascending |
| `modstore.Externals` (GemIsType, GetGameIdFromGemName) wired by InitEnv | `modstore.Resolver`, an interface on the actor |
| `modstore.Conditions` widened to `map[string]any` for class-name strings | typed `Conditions map[string]CondValue` (bool or class name) |
| `Cfg.SkillStats` widened to `map[string]any` | typed `modstore.Output` (absent / false / value are three states) |
| `internal/luapat` is build-time only | now `test/luapat`, test-only |
| GlobalCache is a dump fixture | computed by the ported `buildOutput` driver and compared |
| 147 variants | 145 |
| skill callbacks counted as `UnportedFn` markers (97 of 131) | the marker type is gone; the data declares `SkillCustom.Callbacks` and calc registers bodies by (effect id, kind) — 34 of 134 declared pairs ported |
| statMap `_grantedEffect` is settled in sorted id order | still true, and modelled as `GrantedEffect.StatMapOwner` |
| the granted-passive/ascendancy node maps and the Energy Blade weapons are dump fixtures | derived: `SpecInput.Passives` (`calc.PassiveLookup`, implemented by `build.Passives`) and `calc.energyBladeFor`. `ReplayInput` is `GlobalCache` + `StubHandoff` |
| slots and config remain fixture-fed until their modules | slots, item sets and the build header are native (package `build`, `test/build_test.go`); only the four config fields are still fixture-fed |

Target: `Modules/Calcs.lua` + `CalcSetup.lua` + `CalcPerform.lua` + `CalcTools.lua` +
`CalcFormat.lua`, PLUS `CalcActiveSkill.lua` (tracker files it under calc-offence, but
initEnv needs createActiveSkill/buildActiveSkillModList — ported with the calc spine).
`defenceForConditionals` (CalcDefence) is called mid-perform, pulled in early.

## State (2026-08-26)

- Corpus: 48 builds / 147 variants, all byte-identical on every checkpoint.
- Written: initEnv (pass loop), the full perform body, defence + EHP (calc-defence
  complete for the corpus), offence, triggers (all 80 configs), mirages (all paths +
  Tawhoa's Chosen + the copyActiveSkill minion branch), copyActiveSkill +
  CALCULATOR-mode initEnv, the buildOutput cache driver (GlobalCache computed, not a
  fixture), getCachedOutputValue + stage caches, PvP scaling (offence + EHP),
  FindSkillGem + nameSpec migration, Foulborn Choir resistances block, minion damage
  fixup.
- Remaining calc guards: party-tab crit (party module deferred), the per-skill
  callback gate (97 of 131 UnportedFn), three defensive assertions. Earlier-noted
  panics never since closed: calcTotemLife (no corpus totem+redirect build), non-empty
  AffectedByAuraMod (nil-global in the reference).
- Unwritten: buildOutput's display half (FullDPS roll-up, cost warnings,
  conditions/multipliers discovery), CalcFormat.lua, CALCS mode.
- Native bridge (2026-08-27): the calc differential feeds calc NATIVELY built
  spec + item pool (test/calc_native_test.go — tree.Spec/item projections,
  mods deep-copied at the seam because calc stamps sources in place and the
  test process shares one cached tree; MP_FIXTURE=1 reverts to pure fixture
  replay). Slots and config remain fixture-fed until their modules; the
  skills tab is NATIVE too (package skills via the bridge,
  variant-aware: reduced variants feed empty group lists and calc recreates
  the granted groups; the stale imbued-map semantics of the dump's wipe are
  kept). Negative controls: dropping one native node mod fails 1,104
  comparisons; bumping one native gem level fails 319.

## Setup stage (initEnv)

- `tools/dump_build.lua`: fixture + allocOrder + dbs + per-stage records per variant
  (variant names: `empty` | `<key>.full/.noskills/.treeonly`). Dumps must be
  byte-stable across processes — ALWAYS verify a new or re-dump twice in separate
  processes and cmp; re-running the Go test cannot see a dump that is
  stable-but-wrong.
- Corpus XMLs live ONLY in `test/corpus/` (manifest.tsv paths all point there,
  2026-08-27). `.archive/src/Builds/` is the user's LIVE PoB builds directory —
  it mutates while they play, invisibly to git (untracked). Never point the
  manifest at it and never dump a fixture from it; freeze a copy into corpus
  first (cost of learning this: cocuser silently drifted 126→130 alloc nodes
  mid-session and its differential went red with no code change).
- `calc/input.go`: BuildInput fixture boundary; `test/calc_test.go` TestCalcFixtureEcho
  decodes + re-canons fixtures byte-identical, with corruption negatives.
- `calc/tools.go`: CalcTools.lua. `calc/setup.go`: initModDB + initEnv through the
  tree merge.
- `calc/items.go`: items stage (L697-1283) — slot/jewel gating, jewel limits, disabler
  chains, flask/tincture tracking, special-case items (Aegis/Iron Mass/Dervish/
  Kalandra/Widowhail/Gloves/Boots), rarity/influence/socket multipliers, granted
  passives via dump-resolved nodes.
- `calc/skills.go` + `calc/activeskill.go`: skills stage (L1349-1871) — weapon data
  (unarmed fallback, Hollow Palm), mainSocketGroup clamp, support gathering
  (ExtraSupport/imbued/crossLinked, applyGemMods/applySocketMods/addBestSupport,
  per-group cfg with gem-colour counts and snapshot semantics), createActiveSkill
  (skill types, rejected-support re-passes with Lua array-hole semantics, addFlags),
  canGrantedEffectSupportActiveSkill.
- `calc/skillmods.go`: buildActiveSkillModList — weapon flags, skill mod/keyword flag
  sets, skillCfg construction, support-level merges, gem/quality mods, stage/mine
  multipliers, SkillData extraction, GlobalEffect→buffList separation — plus
  mergeSkillInstanceMods/mergeLevelMod with SORTED stat iteration (dump_build.lua
  replaces the Lua function with a sorted-stats replica because pairs(stats) is
  string-hash-random per process; documented divergence).
- statMap lookups go through env.statMapLookup: the skillStatMapMeta lazy copy via
  data.LazyStatMapCopy, memoized PER ENV, never into the shared skill tables (keeps
  the game-data canon pristine; same values).
- grantedEffect.fromItem runtime mark kept per-Env (env.geFromItemMark), not on the
  shared data tables like the reference — revisit if a dump leaks the mutation across
  variants.
- modstore surface grown for calc: Store exposes Sum/More/Flag/Override/List/Tabulate/
  TabulateAll/GetCondition/GetMultiplier; MergeKeystones takes Store (initEnv calls it
  on the tree ModList); Actor gained Level; modparser.NewMod exported (createMod); Mod
  gained SourceSlot (item slot mod lists carry mod.sourceSlot; canon emits it on both
  sides). modstore.Externals (GemIsType, GetGameIdFromGemName) wired by InitEnv with
  the real calcLib implementations.
- Radius jewels (`calc/radius.go`): funcList re-derived by parsing the item's mod
  lines through modparser.Parse (Item.lua joins continuation lines with a SPACE for
  parsing but stores \n — normalize), asserted against dump funcTypes; fixture carries
  per-socket radiusNodes (id→type) + radiusNodeData (unallocated in-radius
  nodeFixtures) + conqueredBy; per-call buildModListForNode sequences captured as
  `.nodeOrders` (the tail beyond allocOrders = extraRadiusNodeList pairs order).
  modparser exports JewelStoreWriter/JewelNodeRef/JewelNodeFn aliases.
- Granted-skill socket groups (skills.go): match-or-create groups from
  env.grantedSkills + the processSocketGroup port. #EVAL: the reference's
  gemForSkill[skillId-string] lookup into a GE-keyed table never matches, so
  item-granted skills never resolve a gem. ProcessSocketGroup's triggered-cost wipe
  (SkillsTab.lua L1161) assigns `cost = {}`, not nil, and mutates the SHARED
  granted-effect level; kept per-env in env.TriggeredCostWipes — perform/costs must
  consult it. Source-group support gathering + forceSourceWeapon ported.
- Explode groups, Forbidden Flame/Flesh, classRestriction, corrupted-jewel-effect
  scaling: modstore Conditions widened to map[string]any (class-name strings); fixture
  additions: gem explodeSource item/node refs, containJewelSocket slot flag,
  `.grantedAscendancyNodes`; EnemyExplode works via data.materializeFullCustom
  (FullCustom skills get typed SkillTypes/BaseFlags/Stats/Levels/BaseMods views
  sharing the Custom tables; canon unchanged).
- Energy Blade: InitEnv is a pass loop (initEnvPass) — buildSkillsStage returns a
  restart request on an enabled Energy Blade gem; the pass restarts with override
  conditions and the order cursor carried forward (4 buildModListForNodeList calls per
  re-entering initEnv). Synthesized weapons come from the dump (`.energyBladeItems`
  per slot) instead of porting Item construction; no fixture entry = the info-nil/Bow
  fallthrough.
- Minion creation block: calc/skillmods.go Minion + item sets (createMinionSkills
  itself is perform-stage). `.skillLists` compares per-skill skillModList/cfg/flags/
  data/buffs/weapon cfgs plus minionList + minion {type/level/hostile/weaponData}.
  Fixture gained spectreList, itemsTab.itemSets/itemSetOrderList.
- skillLists nuance: early-disabled skills (default Melee with no weapon) return
  before minionList is set — the key is absent, not empty.
- Gotchas: Lua Flag() returns nil (not false), so t_insert(explodeSources,
  flag-result) is a NO-OP for non-exploding nodes; skillFlags is a set-of-true (never
  store false — the `or` chains keep nil). Go typed-nil trap: Cfg.SkillGem (any)
  holding a nil *data.Gem defeats eval's nil check — assign only when non-nil.
- Dump stability: the sorted-pairs injection around LoadModule "Modules/Calcs" fixed
  numeric+string key order but left TABLE keys raw — env.flasks/env.tinctures are sets
  keyed by item tables, so pairs() walked hash order over addresses, random per
  process (coc carries two flasks granting an identical Armour mod; the dedup kept
  whichever merged first and `.performDbs` recorded a different mod `source` per run;
  the Go side sorts by item id and passed only by luck). Fixed by ordering table keys
  by item id when every key has one, matching calc.sortedFlasks.
- The corpus caught a REAL modparser port bug: getThreshold set inner value.mod Source
  without SourceSet, so the canon dropped it.
- Perform-residue scrub: the app's load-time calc mutates shared skill tag tables in
  place (warcryPowerBonus, CalcPerform L2330); the dump scrubs data.skills before
  emitting (perform recomputes it; extend the scrub list if more keys surface). The
  Go-side scrub/restore equivalents were deleted when data became immutable (see data
  boot below).

## Perform

CalcPerform body (L1252-3718): calc/perform.go (prologue through charges +
reservations), performbuffs.go (buff/curse/link loop, curse slots, guards, buff
application, ailments, exposures, ally-life, GemLevel tail), performutil.go /
performmisc.go / performminion.go / performflasks.go (doActor* + flask/tincture
merges). Checkpoints `.performDbs`/`.performOutput`/`.performMinionDb`/
`.performMinionOutput`. Ally-life (needsAllyLife) is corpus-reachable via
Companionship support (lowlife).

FindModifierSubstring: plain Go regex (calc.patFind, cached); Item.D carries the data
tables and the fixture's ExplicitLines/OtherLines are pre-filtered. It must NOT use
internal/luapat — that package is build-time only ("deleted together with the Lua"),
and calling it from the calc engine would pin it into the shipped runtime. Safe
because the tables use no syntax beyond a leading ^ / trailing $, and searchCond comes
from a letters-and-whitespace-only capture (ModParser.lua ItemCondition tags), so no
metacharacter can reach the matcher; test/itemtag_test.go guards that.

Perform traps found by the differential:
- skillData `or` chains: a PRESENT 0 wins (0 truthy in Lua) — never use `==0` as
  absence for skillData reads (reservation percents/Forced keys; "no reservation"
  effects write explicit 0s).
- Lua Flag() yields nil: `output.X = Flag(...)` stores no key when false
  (ChaosInoculation) — only write the key when true.
- Hex doom guard's trailing `and Sum(...,"MaxDoom")` is a bare number, always truthy
  (#EVAL) — HexDoomLimit/HexDoom set for every hex curse.
- Ailment gate first clause (`Val>0 or Sum(...)`) is always truthy (#EVAL) — only the
  immune/avoid clause gates.
- GemLevel/GemQuality output block (L3862+) runs AFTER the stubbed defence/offence
  handoff — part of the perform checkpoint.

## Defence

calcs.defence (CalcDefence.lua L636-1632) + calcs.resistances: calc/defence.go
(resistances + shared helpers), defencebody.go (action speed, block, res-driven
conversions), defenceprimary.go (ward/ES/armour/evasion, evade chance),
defencerecovery.go (suppression, dodge, leech caps, regen, ES recharge),
defencetail.go (recoup, damage reduction, avoidance, self-ailment duration/effect).
Entry: env.RunDefence() = player then minion, back to back, matching the dump.
Checkpoints `.defenceDbs`/`.defenceOutput`/`.defenceMinionDb`/`.defenceMinionOutput`;
negative control: perturbing one output value fails every variant.

The dump reaches post-perform stages by keeping the real stage functions in locals
before stubbing the handoff, then calling them explicitly AFTER the earlier records —
each checkpoint stays independent. Triggers gets its own records ahead of the offence
ones, so it could be ported and verified alone. No corpus build makes the reference
error in these stages (all sources dump cleanly with the real offence running).

Defence traps (same family as perform's):
- Flag() nil: CappingES (an or-chain ending in a configInput read),
  Corrupted-Blood/Maim/Hinder/Knockback immunity, and
  EnergyShieldRechargeAppliesToLife (`Flag(A) and not Flag(B)` — absent when A is
  unset, present-and-false when A is set and B is too). By contrast `not (...)` and
  `x ~= y` DO store real booleans.
- Party-tab branches (BlockChance/MaxBlock/MaxLifeLeech equal-to-party,
  MovementSpeedEqualHighestLinkedPlayers) panic: ladder builds never populate
  partyTab.

## EHP

calcs.buildDefenceEstimations (CalcDefence.lua L1635-3828): calc/ehp.go (not-hit
chances, enemy damage input, taken-as shifts, taken multipliers), ehphit.go (incoming
hit mitigation), ehpstun.go, ehppools.go (life recoverable, Petrified Blood, ES
bypass, MoM), ehpguard.go (guard, aegis, ally pools, Vaal Arctic Armour, total pools),
ehpreduce.go (reducePoolsByDamage), ehphits.go (numberOfHitsToDie), ehpehp.go (hit
counts, total EHP, survival time), ehprecoup.go, ehpmaxhit.go (takenHitFromDamage +
the quadratic max-hit solve with smoothing), ehpdegen.go. Entry: env.RunEHP().
Checkpoints `.ehpDbs`/`.ehpOutput`/`.ehpMinionDb`/`.ehpMinionOutput` — green on the
first run, verified real by perturbing overkillDamage by 1e-7 inside
reducePoolsByDamage (fails every variant).

#EVAL quirks in the EHP body:
- Max-hit pool fixup reads `shared or 0 + typed or 0`, which Lua parses as
  `shared or (0+typed) or 0` — the typed guard rate/absorb is dead whenever a shared
  value exists.
- AnyTakenReflect is assigned false in BOTH branches ("this needs a rework as well"),
  so the reflect multiplier block never runs.
- Every pairs(damageTable) loop inside reducePoolsByDamage is order-independent (sum /
  per-key init / subtract-from-all / min), so the Go maps need no ordering; only the
  dmgTypeList walks are sequenced.

## Offence

calcs.offence (CalcOffence.lua L323-6168): calc/offence.go (damage-type flag sets,
calcDamage/calcAilmentSourceDamage, radius maths, conversion table), offencebody.go
(entry, calcAreaOfEffect, calcResistForType, runSkillFunc, prologue),
offenceskilldata.go (stat-conversion chain, repeats, random phys, momentum),
offencetypestats.go (minion limits, chain/projectile/pierce, melee range,
aura/curse/warcry/link, reservation mults, trap/mine/totem/brand/corpse),
offenceduration.go, offencecosts.go, offenceconv.go (explosions, canDeal, per-cfg
conversion tables, passList, combineStat), offencehitrate.go (accuracy, speed,
trauma), offencemisc.go, offencewarcry.go (exerts, pacts, ruthless), offencecrit.go
(crit, double/triple damage, culling, Cryogenesis redirect, base hit damage),
offencedamage.go (the two-pass damage loop, resists, leech), offencedps.go (per-pass
DPS, combine, leech rates), offenceailments.go + offencebleed.go (affliction chances,
bleed/poison/ignite, non-damaging ailments, knockback, stun, impale), offencedot.go
(decay, burning ground, generic DoT, cost per second, combined DPS), offenceselfhit.go
(+ applyDmgTakenConversion). Entry: env.RunOffence() = player then minion, matching
the dump. Checkpoints `.offenceDbs`/`.offenceOutput`/`.offenceSkillOutput`/
`.offenceMinionDb`/`.offenceMinionOutput`; negative control: +1e-7 on AverageHit
fails every variant.

Section map (L323-6168): prologue/AoE 323-493, skill data 494-1022, skill type stats
1023-1456, duration 1457-1625, uptime 1626-1658, costs 1659-1841, conversion
1878-1933, damage passes 1934-2110, hit rate 2111-2421, misc DPS 2422-2540, MAIN
DAMAGE 2541-4009, leech 4010-4064, AILMENTS 4065-5554, secondary 5555-5675, DoT
5676-5863, self hit 5864-6010, combined DPS 6011-6168. calcAreaOfEffect is DEFINED at
L345 but first CALLED at L1152, after the skill-type-stats section sets
actor.weaponRange1.

Offence traps found by the differential:
- `Sum(...) or 0 + X + Y` parses as `Sum(...) or (0+X+Y)`, and Sum always returns a
  number, so TripleDamageChance drops the enemy and on-crit terms entirely (#EVAL).
- `(Flag and 100 or 0) + Sum(Avoid...)` parses as `Flag and 100 or (0 + Sum(...))`:
  an immune enemy yields exactly 100 and the avoid sum is dead (#EVAL).
- `skillModList:More("MORE", cfg, "Accuracy")` puts "MORE" in the cfg slot, so the
  real cfg becomes a never-matching modifier name — that Accuracy More is cfg-less.
  Ported as More(&Cfg{}, "Accuracy"): a non-nil EMPTY cfg, because ModStore:EvalMod
  distinguishes `not cfg` from a cfg whose fields are all nil (#EVAL).
- `cfg` in the Mantra of Flames BuffOnSelf lines is an undeclared global (the pass
  local died with the loop above), i.e. nil (#EVAL).
- `dotCfg` in the hit-damage resistance lookup is likewise a global; only the ailment
  sections declare a local of that name (#EVAL).
- `ipairs({["FRDamageTaken"] = ...})` iterates zero times, so the Forbidden Rite
  self-hit block never runs (#EVAL).
- ModStore:GetStat is an `or` chain: a stored FALSE in actor.output falls through to
  cfg.skillStats, a stored 0 does not. Cfg.SkillStats widened to map[string]any so the
  weapon passes can alias output.MainHand / output.OffHand live.
- The bleed/poison/ignite dot cfgs give skillCond a metatable falling back to
  skillCfg/cfg. Only "CriticalStrike" is mutated after construction and the dot table
  shadows it, so a snapshot copy is exact.
- The cost section must not read a wiped level's cost — env.TriggeredCostWipes'
  first consumer.

## Triggers, mirages, GlobalCache

Files: calc/triggers.go (helpers, findTriggerSkill, the 1000-attack
calcMultiSpellRotationImpact rotation sim, the RunTriggers entry), triggerhandler.go
(defaultTriggerHandler), triggerconfig.go (the 80-key configTable dispatch),
mirages.go, globalcache.go. Checkpoints: `.triggersDbs`/`.triggersOutput`/
`.triggersSkillData` + `.triggersMinionOutput`/`.triggersMinionSkillData`; the test
runs the exact dump sequence RunTriggersPlayer -> RunOffencePlayer (mirage-gated) ->
RunTriggersMinion -> RunOffenceMinion. Negative control: +1e-9 on the simulated
trigger rate fails exactly the 7 checkpoints of the two trigger-driven skills
(coc.full triggersOutput/triggersSkillData/offenceOutput/offenceSkillOutput,
holyrelic.full triggersMinionOutput/triggersMinionSkillData/offenceMinionOutput) —
the trigger checkpoints are load-bearing and offence consumes the rate.

GlobalCache anatomy: Data/Global.lua L355 declares `cachedData = {MAIN={}, CALCS={},
CALCULATOR={}}`; Common.lua cacheData() stores ~12 output values (Speed, HitSpeed, the
costs, crit, TotalDPS) per uuid; cacheSkillUUID is name_SLOT_gemIdx_groupIdx.
CalcPerform L3918 calls cacheData at the very END of perform; Calcs.lua buildOutput
(L202) drives the fill: for every skill with includeInFullDPS it sets
fullEnv.player.mainSkill and runs a whole calcs.perform, so each skill caches its own
final (offence-fed) output — the cache cannot be populated before offence exists;
port order was offence -> driver+cacheData -> triggers. Triggers reads a bounded
field set — enumerate with
  grep -oE "GlobalCache\.cachedData\[env\.mode\]\[[a-zA-Z]+\]\.[A-Za-z.]+" CalcTriggers.lua
In the dump the cache is populated by the app's own OnFrame buildOutput, and the
dump's manual calcs.perform then OVERWRITES the main skill's entry with a pre-offence
one (Speed nil, offence stubbed). Triage: calcs.triggers is a NO-OP for every corpus
variant except coc.full, whose live trigger path writes four output keys
(EffectiveSourceRate, SkillTriggerRate, Speed, TriggerRateCap) plus
skillData.triggerRate/triggerSourceUUID; its trigger rate 7.56756 is Cyclone's cached
speed.

Trigger traps:
- `ignoresTickRate = ignoresTickRate or (storedUses and storedUses > 1)` writes a REAL
  false when storedUses is present but 1; only an absent storedUses leaves the key
  absent. Cost two variants until fixed.
- `triggeredSkills[1] == packageSkillDataForSimulation(...)` compares two freshly
  built tables, never true, so that arm reduces to
  `ignoresTickRate and not config.triggeredSkillCond` (#EVAL).
- `actor.mainSkill.triggeredBy.ignoresTickRate` is dead: nothing in the archive sets
  that field (#EVAL).
- The triggerCD read guards on actor.mainSkill.triggeredBy but reads
  env.player.mainSkill.triggeredBy (#EVAL; same table for the player).

Skill callbacks live in calc/skillfuncs.go, keyed "<grantedEffectId>:<callbackName>",
consulted by runSkillFunc before the UnportedFn panic. Ported include:
Cyclone/CycloneAltX/VaalCyclone + BloodSacramentUnique initialFunc, EnemyExplode /
StormBrand / RighteousFire preDamageFunc, Explosive Trap random radius +
preDamageFunc, Explosive Arrow explosiveArrowFunc, Blade Blast, Ice Spear of
Splitting, Lightning Tendrils of Eccentricity (preDamage + postCrit), Herald of the
Breach, Penance Brand of Dissipation, the jewel-func AddList ModList argument.

Mirage sub-environments (Mirage Archer, Sacred Wisps, Tawhoa's Chosen, ...):
- copyActiveSkill runs createActiveSkill FIRST, then a whole second initEnv (mode
  CALCULATOR, inheriting env.override), then buildActiveSkillModList. CALCULATOR
  differs from MAIN only in skipping UI write-backs (node.finalModList, displayEffect,
  displayLabel, jewelRadiusData, superseded, group.mainActiveSkill); `superseded` is
  read only by SkillsTab, so nothing behavioural.
- That second initEnv consumes its own buildModListForNodeList orders — the dump
  records them separately (`.mirageAllocOrders`/`.mirageNodeOrders`) and ReplayInput
  carries them.
- The dump's stubbed handoff applies to the nested perform too, so the sub-env's
  output is the perform-body state — exactly what Go's env.Perform() produces.
  Compared as `.mirage` (name/count/skillPart) and `.mirageOutput`.
- `mainSkill.ModFlags` / `.KeywordFlags`, which every mirage path passes to NewMod,
  are never assigned anywhere in the archive: both nil.

## The buildOutput driver (Calcs.lua cache half)

cacheData, buildActiveSkill, the buildOutput cache fill (calc/buildoutput.go),
PerformFull with the real per-actor handoff. GlobalCache is COMPUTED: the test builds
it with the driver and compares against the archive's `.globalCache` snapshot
(dumpVariant runs wipeGlobalCache + calcs.buildOutput with the real stage functions
restored, exactly as Build.lua:675 does). What the driver forced out (each verified by
the cache canon):

- The recursion breaker at the TOP of calcs.triggers (L1604): skip the stage when
  env.limitedSkills[mainSkill uuid]. It stops Manaforged Arrows (which builds its own
  skill to learn its mana cost) from recursing; without it the port recursed
  unboundedly — a full Env per level, ~100GB commit before it was killed.
  BuildActiveSkill also carries a DEFENSIVE depth-20 panic with no reference
  counterpart.
- CachedSkill accessors are LIVE through the stored Env/ActiveSkill, as in the
  reference: a stage that runs after cacheData changes what the entry appears to hold.
  The first port snapshotted them and diverged.
- Nested performs inherit the handoff state: `calcs.perform(newEnv)` inside a mirage
  runs whatever calcs.defence/offence currently are — stubbed no-ops during the dump's
  checkpoint phase, the real stages inside the driver. Env.StubHandoff models that;
  the test sets it on checkpoint envs. Missing it cost voidstorm's Sacred-Wisps mirage
  its TotalDPS (and the cache canon caught the missing MirageDPS — the offence
  mirage-DPS tail is ported).
- A THIRD pairs() nondeterminism (same shape as the flasks and the statMap backref):
  the granted-skill support gather iterates supportLists[slotName] keyed by GROUP
  TABLES, and shared-statMap mod SOURCES are stamped last-writer-wins under
  pairs(data.skills). Settled: the sortedPairs shim orders group keys by
  socketGroupList position (dumpBuild global), and dump_build re-stamps shared mod
  sources in sorted id order, mirroring dump_gamedata's reassign.
- Every initEnv of one build yields identical alloc orders (dump-verified), so
  sub-environments replay the top env's orders, and mirageReplay falls back to them
  when the mirage only runs inside a driver build.
- test/corpus/manifest.tsv maps every dump key to its build XML (the five
  archive-Builds/ ones included). Env knobs: MP_ONLY=<variant prefix> narrows
  TestCalcInitEnvAgainstReference to one build (an unrelated variant's guard panic
  cannot pre-empt the one being diagnosed); MP_GUARDS turns guard panics into
  collected failures; MP_DUMPGC dumps the cache canons.
- Fixtures are exact end to end: luacanon.EncodeExact mirrors canon.encodeExact and
  the fixture echo round-trips at %.17g.

getCachedOutputValue found a REAL driver bug: BuildOutput started with a NIL cache
map, so nested cacheData installed into per-env private maps — siblings re-missed,
FillGlobalCache rebuilt skills without the limited flag, and Penance's own solo build
got its stage multiplier. Materialize the shared map in the PARENT before any perform.
The {uuid} limited flag is what stops self-recursion.

## Corpus

From ../missingBuild (mb: 10k real ladder chars, 5.7k with stored PoB builds;
`mb serve --detach` first). Pull:
`mb pob-run --char <name> --reprocess --pob <repo>/.archive --dump test/corpus`
then `luajit tools/dump_build.lua <key> ../../test/corpus/<name>.xml`. The dump's own
asserts triage which branch a build needs. Guard-driven growth: pull a build for a
panic (`mb search`), dump it, port until green. Synthetic overrides (config
permutations) multiply coverage cheaply.

- Ladder imports never carry spectreList/UI config — spectre builds contribute gear
  but no live minion; use fixed-minionList mains (zombies, Absolution, Holy Relic,
  Animate Guardian) for minion coverage.
- Key → coverage: zombies (EssohNecro: Raise Zombie + Animate Guardian — minion
  creation incl. item sets), lowlife (Higger_Bebrafication: low-life ES; ally-life via
  Companionship), spectre (gktiiS: Voidfletcher granted VoidShot, abyss eye jewels,
  threshold+transform radius jewels, Elegant Hubris), cyclone (COTA_Halbae: Light of
  Meaning, curse-on-hit ring), rf (TwoGuysOneMirror: Chieftain RF, Glorious Vanity,
  The Adorned + corrupted jewels, Forbidden pair, explode sources), holyrelic
  (Waffle_Idol: Holy Relic minions, Fortress Covenant, Explode group), eblade
  (DEE_FOUR_LOH_BAD: Inquisitor Storm Brand + spectres + Stone Golem + Animate
  Guardian + Brutal Restraint). Earlier closures also came via Explosive Trap /
  Explosive Arrow / Blade Blast / Ice Spear of Splitting / Lightning Tendrils of
  Eccentricity / Herald of the Breach / Penance Brand of Dissipation builds, the
  "cast when damage taken" and "TriggeredMoltenStrike" trigger configs, and the two
  mirage paths that forced copyActiveSkill/CALCULATOR-mode initEnv/nested perform.
- Authored shells (test/corpus/authored_*.xml + manifest rows) exist purely to reach
  guards: trig1 (gem triggers: melee-kill/stunned/ward-break/death/prismatic-burst/
  snipe/intuitive-link), trig2 (Mjolner, Oskarm, Kitava's Thirst, both Mark rings),
  trig3 (Asenath's Chant, Maloney's Mechanism, Lioneye's Paws), trig4 (Ngamahu's
  Flame, Shroud of the Lightless) — these closed the 80 trigger configs +
  helmetFocusHandler + the Kitava mana-spent handler; a stages build
  (Blight/Penance/EQ-of-Amplification part 2; Scorching Ray verified with them);
  config-flip copies of the corpus doomblast build (Doom Blast expiration + hexblast
  sources; config DEFAULTS reach the fixture — doomBlastSource=vixen came from the
  default, the XML carries nothing); misc1 (PvpScaling config); Tawhoa's Felling +
  Earthquake (Tawhoa's Chosen mirage); Saviour + Dominating Blow (copyActiveSkill
  minion branch, verified through the Reflection sub-build's cache canon).
- Authoring notes: copy a corpus XML's shape; socket groups need no items except where
  the trigger checks the slot's item (Maloney's reads the quiver's modSource); unique
  item text comes verbatim from Data/Uniques/* minus variant markers, one line per mod
  (the two-line Kitava's trigger text is ONE mod line in an item); only the ACTIVE
  ItemSet's slots exist (a second ItemSet is invisible to the calc). Verify a new
  authored build loads by checking the dump's .skills list before porting against it.
- Roughly half the trigger config table is corpus-exercised; the rest is
  code[x]/archive-unverified awaiting more shells.

## Guard-tail notes (2026-08-26)

- FindSkillGem + the nameSpec migration: five Lua abbreviation patterns implemented
  natively; OnFrame pre-migrates resolvable names before any dump, so only
  UNRESOLVABLE names exercise the replay branch.
- stagesAreOverlaps: reference-dead (nothing sets it) — implemented as the real
  expression, panics removed.
- Foulborn Choir overcapped-lightning-res block: temp child-ModDB resistances +
  life/mana re-derivation; a plain rare amulet line reaches it.
- Minion damage fixup: the Go guard was BROADER than the reference — L488 acts only on
  minionData.damageFixup (Flame/Lightning Golem); minion actors carry MinionData
  (MonsterTags + DamageFixup).

## Data boot (2026-08-26)

- `data` is PACKAGE-LEVEL: `type Data` is gone, the 90 fields are vars, Load fills
  them in place, call sites read `data.Skills` / `data.Misc.ServerTickRate`. All
  `env.Data` / `d :=` plumbing deleted; ten signatures dropped their data param.
  Prerequisite: data is IMMUTABLE after Load — the two remaining calc-time writers
  relocated per-env (hasGlobalEffect -> env.globalEffectOverlay via LazyStatMapCopy's
  second return; warcryPowerBonus -> copy-on-write into the env-owned buff list),
  which deleted the test restore dance and the warcry scrub.
- The seven var/type name collisions resolved to internal lowercase types; the last
  exported shape (WeaponTypeDef) vanished by reshaping getWeaponFlags to return
  (flags, melee, known) — callers only ever consumed info.Melee and nil-ness. Zero
  exported shape names.
- `data/raw/` holds pobexport's complete output (21 documents + the hand-maintained
  modfoulbornmap.jsonc, ~42MB), COMMITTED and EMBEDDED (//go:embed raw in data/raw.go;
  RawSources() builds Load's input). A built binary is self-contained. Regeneration is
  explicit, part of the GGPK-update workflow: `pobexport -src <ggpk>` (default -out
  data/raw). The GGPK is needed ONLY for regeneration and the export differential;
  every other test runs from the embedded raw. The fingerprint doc-cache (test/.cache)
  is deleted.
- Timing: calc differential 5.7s cold for all variants (was ~140s); full suite ~110s,
  dominated by the export differential (97s, the one GGPK-dependent test).

## Three harness bugs (each hidden behind a formatting or folding step)

1. `_grantedEffect` is process-random for a SHARED statMap. Data.lua:1039 sets
   `grantedEffect.statMap._grantedEffect = grantedEffect` inside a
   `pairs(data.skills)` loop; ExplosiveTrapAltX aliases ExplosiveTrap's statMap table,
   so the backref is last-writer-wins over a random order — and the lazy statMap
   copies then stamp the winner as the mod source even for the other skill. Two dumps
   of the trap build differed in exactly one byte range. Fixed by settling it in
   sorted id order in dump_build.lua (matching dump_gamedata's reassign pass) and
   modelling it in Go as GrantedEffect.StatMapOwner, which LazyStatMapCopy stamps
   with.
2. Fixtures were serialised at %.14g, which is lossy. The cospri build's trigger sim
   ran 1000 attacks off the cached source rate; the fixture's 10.063177748344 and the
   reference's true 10.063177748344373 are the same 14 digits but land on either side
   of the loop's `<` bound, so the sim counted 1000 triggers instead of 1001 and the
   trigger rate came out 0.1% low. canon.encodeExact (%.17g) now serialises replay
   INPUT; the compared canons stay at 14 digits on both sides. Only re-dumped builds
   carry exact fixtures; re-dump when a build shows an unexplained tail-digit
   divergence.
3. Go folds untyped constant expressions at arbitrary precision. `1 / 0.033` as a Go
   constant is 30.303030303030305; Lua's runtime division is 30.303030303030301.
   data.Misc.ServerTickRate fed a `m_ceil(x * rate)` tick rounding, so Absolution's
   3.3s duration ceilinged to the next tick in Go and not in the reference. Fixed with
   `1 / float64(0.033)`, which forces the typed-constant rounding. Swept the rest of
   data/tables.go — no other constant expression differs from its runtime form.

## OPEN, POST-PARITY

The compared canons are `%.14g` on both sides, so the differential is blind below the
15th significant digit — exactly the band harness bug 2 lived in. A green run does not
rule out sub-ulp drift that a later comparison bound amplifies. The review: re-run the
corpus with the canons at `%.17g` on both sides. Until then treat every `archive [x]`
as "agrees to 14 digits". Two loose ends feed it: `coc`/`cocuser`/`dualstrike`/
`bfbb`/`empty` still carry `%.14g` FIXTURES (their XML is outside test/corpus; the
other 33 were re-dumped exact and no compared canon moved), and the constant-folding
sweep was a regex over data/ and calc/, not a proof.

Test-isolation note: processMod sets `ge.HasGlobalEffect` on the granted effect it is
passed, so a calc run mutates the SHARED data set (faithfully — the reference mutates
its own global tables the same way; calc/skills.go:551 reads the flag). The game-data
canon is the post-load state, so gamedata_helper_test records hasGlobalEffect at load
and TestGameDataAgainstReference restores it before comparing.

## Dump gotchas (hard-won)

- HeadlessWrapper stubs `Inflate` to "" — without binding runtime/zlib1.dll the tree
  spec SILENTLY stays a default 1-node Scion (build codes are zlib). dump_build.lua
  carries the same ffi binding as cook's pob.lua.
- `arg` is clobbered by the HeadlessWrapper boot — read arg[1]/arg[2] BEFORE dofile.
- ModList/ModDB objects carry actor/parent backrefs — canon their array part only
  (modArray) or canon.encode recurses forever.
- `pairs(env.allocNodes)` order is LuaJIT numeric-key hash order: deterministic per
  table STATE, NOT derivable in Go — and the table grows mid-initEnv (granted
  passives), so the two buildModListForNodeList calls (L672, L1319) can see different
  orders. The dump wraps calcs.buildModListForNodeList (dynamic lookup, so wrapping
  works) and records the order per call as `<variant>.allocOrders`; the replay pops
  one per call. Granted-passive nodes are dump-resolved from the tree maps into
  `<variant>.grantedPassiveNodes` (GrantedPassive values are lowercase names).

## Replay semantics found (CalcSetup)

- initEnv one-shot MAIN: specCopy is a pure cache copy (non-mutating) — replay skips
  it entirely.
- mergeKeystones at L673 targets env.initialNodeModDB, NOT modDB; keystone RESOLVED
  mods only reach modDB in perform (CalcPerform L1257 resets env.keystonesAdded = {}
  first, else the L673 marks would starve it). keystoneMap comes from the fixture
  (spec.keystoneMap: name → modList).
- The skill/support stages write NOTHING into modDB/enemyDB/itemModDB (verified by
  grep over CalcSetup L1335+ and CalcActiveSkill; the only write is enemy
  ActiveMineCount for mine skills) — dbs parity for item-less variants doesn't need
  the skill spine.
- Items stage db writes are all item-guarded except the L1262-65
  overrideEmpty*Sockets config assignments (nil-assign no-ops without the config
  keys) and mergeDB(modDB, itemModDB) at L1283.
- `env.enemyLevel = build.configTab.enemyLevel or min(MaxEnemyLevel, charLevel)`
  reads configTab.enemyLevel (PROCESSED field, not input) — dumped as fixture
  `configEnemyLevel`.

## Architecture

Go package `calc`. Builds on: `modstore` (ModDB/ModList; NewMod = modparser.NewMod),
`data` (constants, skills, gems, minions), `modparser` (Parse for pantheon/config
lines, jewels.go radius funcs keyed by mod line).

**Fixture boundary** (`calc.BuildInput`): everything initEnv reads from `build`,
dumped per build by `tools/dump_build.lua` from the loaded XML: characterLevel,
classId, configEnemyLevel, curClassName, treeVersion, mainSocketGroup, classStats,
configInput scalars, config/party mod list canons, spec (allocNodes with
modList/keystoneMod, keystoneMap, counts, masteryTypes/tattooTypes), itemsTab (slots
+ per-item payloads incl. mod-list canons, scalar bags for flask/jewel/weapon/armour
data, active mod-line texts for FindModifierSubstring, modSource; item mods carry
`sourceSlot` — a first-class modparser.Mod field), plus the later additions noted
above (spectreList, itemSets, radius/energy-blade/explode/granted-node records,
alloc/node/mirage orders, `.globalCache`). Socket counting in items.go assumes zero
socketed gems (true for every compared variant) — wire it to the socket-group fixture
when that lands.

The variant map in the differential only grows. Reading state: Calcs.lua /
CalcTools.lua / CalcSetup.lua fully read; CalcPerform.lua / CalcActiveSkill.lua were
read structure-first and ported against the differential.

NEXT (calc-core): buildOutput's display half, CalcFormat.lua, CALCS mode.
