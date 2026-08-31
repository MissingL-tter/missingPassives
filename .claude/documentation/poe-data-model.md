# PoE/PoB data model — verified reference

What the data actually means, learned porting it. Every claim here survived a
differential test against the archive (see parity.md for which test). Go
packages named are the authoritative, typed form; `.archive/src` paths are
the Lua originals.

Pipeline: GGPK dats → `export/` builders → `data/schema` JSON documents →
`data.Load` → the runtime `data.Data` (= Lua's global `data`). Calc consumes
`data` + `modstore` + `modparser`.

## The modifier

`modparser.Mod{Name, Type, Value, Flags, KeywordFlags, Source, Tags}` — one
shape everywhere: parsed item text, skill statMaps, minion modLists.

Types and their aggregation (modstore):
- `BASE` — additive flat. `Sum` adds them.
- `INC` — additive percentage. Sum, then ×(1+total/100).
- `MORE` — multiplicative. `More` folds ×(1+v/100) each; per-name product is
  rounded to 2dp unless the name is in highPrecisionMods (Data:
  `HighPrecisionMods`, e.g. leech/regen at 2dp, SupportManaMultiplier 4dp) —
  then floor at that precision.
- `FLAG` — boolean OR. Value true; first truthy wins.
- `OVERRIDE` — first truthy value replaces the computed stat entirely.
- `LIST` — collected values (tables); `SkillData` LIST mods carry
  `{key=..., value=...}` payloads (e.g. radius, duration).
- `MAX`/`MIN` via Max/Min aggregation (Max floors at 0).

`Flags` (ModFlag) and `KeywordFlags` (KeywordFlag) are bitmasks —
`modparser/globals.go`. A mod applies when `queryFlags & mod.Flags ==
mod.Flags` (mod's flags must be a subset of the context). KeywordFlags OR
by default; the `MatchAll` bit (0x40000000) switches to subset matching.
Flag examples: Attack 0x1, Spell 0x2, Hit 0x4, Dot 0x8, Melee/Area/
Projectile...; keyword: Aura, Curse, Bleed, Poison, per-skill classes.

Tags gate a mod's value (evalMod, modstore/eval.go). The important ones:
- `Condition {var|varList, neg}` — player condition flags.
- `ActorCondition {actor="enemy"|"parent", var}` — another actor's condition.
  Skill statMap mods marked notMinionStat get `{ActorCondition, actor=
  "parent", neg=true}` appended so they don't leak to minions.
- `Multiplier {var, div, limit, limitTotal, actor}` — scales by a counted
  stat (charges, stacks); div divides, limit caps (limitTotal caps the
  scaled result).
- `MultiplierThreshold {var, threshold}` — active only at/above a count.
- `PerStat {stat|statList, div}` / `PercentStat` (ceil) /
  `StatThreshold {stat, threshold}` — scale by/gate on computed stats.
- `SkillName/SkillId/SkillType/SkillPart` — restrict to a skill context.
- `SlotName/SocketedIn/ItemCondition` — item/slot context (Kalandra ring
  swaps handled in ItemCondition).
- `ModFlagOr/KeywordFlagAnd/MonsterTag` — flag/tag gates.
- `GlobalEffect {effectType="Buff"|"Debuff"|"Aura"|"Guard", effectName}` —
  the mod applies through a named effect; its presence sets the granted
  effect's hasGlobalEffect.
- `DistanceRamp/MeleeProximity/Limit` — positional scaling.

## Crafting mod pools (`data.ItemMods` + Veiled/BeastCraft/Necropolis)

Pools: Explicit, ItemExclusive (implicits + unique explicits), Corrupted,
Delve, Synthesis, Scourge, Eldritch, Flask, Tincture, Graft, Jewel,
JewelAbyss, JewelCluster, JewelCharm, Foulborn, WatchersEye; `Item` is
Explicit∪ItemExclusive∪Corrupted∪Delve∪Synthesis∪Scourge∪Eldritch (later
overwrites earlier on shared ids).

Entry semantics (`data.ItemModData`):
- `Lines` — the display text (ranges as `(a-b)`).
- `Type` — Prefix/Suffix/Corrupted/Synthesis/DelveImplicit/ScourgeUpside/
  ScourgeDownside/Exarch/Eater, or `<tier><influence>` (e.g. "1Shaper") for
  matched-influence implicits.
- `Group` — mod family: two affixes sharing a group cannot coexist on one
  item. PoB does not enforce this while typing (cook's validate.lua does).
- `WeightKey/WeightVal` — spawn legality: paired tag→weight lists, first
  matching tag of the ITEM's tag set decides; weight 0 = cannot roll there.
  `default` is the fallback key. Pool membership alone is NOT legality —
  armour bases fall back to a pool holding everything.
- `WeightMultiplierKey/Val` — generation weight multipliers (same pairing).
- `ModTags` — describe-tags (catalysts target these; the catalyst-relevant
  set: elemental_damage, caster, attack, defences, resource, resistance,
  attribute, physical_damage, chaos_damage, speed, critical).
- `TradeHashes` — trade-site stat ids per stat: murmur2(GGG stat hash bytes,
  seed 0x02312233); min+max stat pairs hash both hashes concatenated.
  Trade's `explicit.stat_N` ids are exactly these numbers.
- `Level` — mod ilvl requirement. Item req level from an implicit is
  floor(modLevel × 0.8).

Veiled mods (`data.VeiledMods`): affixes named Chosen (prefix), of the Order
(suffix), Catarina's/other master signatures; master mods weight on pool
keys like `catarina_veiled_prefix` and list only excluded slots. The
generated uniques (Paradoxica, Cane of Kulemak, Queen's Hunger) enumerate
these pools as variants.

## Item bases (`data.ItemBases`)

Per base: `Type`/`SubType` (UI grouping), sorted `Tags` set (spawn-weight
matching keys), `InfluenceTags` (influence → tag: `<baseTag>_shaper`,
`_elder`, `_adjudicator`(warlord), `_basilisk`(hunter), `_crusader`,
`_eyrie`(redeemer), `_cleansing`(exarch), `_tangle`(eater)), implicit and
enchant text (newline-joined) with per-line mod-type tags and GGG ids,
`CannotBeAnointed` when the base has innate enchant rows, `Req` (level from
max(dropLevel if >4, implicit levels ×0.8)), and per-kind blocks:
- weapon: PhysicalMin/Max, CritChanceBase (=dat/100), AttackRateBase
  (=round(1000/speed, 2)), Range.
- armour: Armour/Evasion/EnergyShield/Ward BaseMin/Max, BlockChance
  (shields), MovementPenalty (negated).
- flask: life/mana per use, duration (=dat/10), charges, buff lines.
- tincture: manaBurn and cooldown (both dat/1000).

`ItemBaseLists` groups visible bases by "Type" or "Type: SubType", sorted
req-level DESC then name. Essences map base id → per-slot mod id at each
tier. `RareLikeUniques` = uniques crafted with rare controls (Subsume the
Source: 4 abyss prefixes; Crimson Storm: "of the Order" suffixes; Dread
Captain's Cutlass: Explicit pool on a deepwater_sword-tagged base).

## Skills and gems (`data.Skills`, `data.Gems`)

GrantedEffect: id = granted-effect id ("FrostBolt"); `modSource` =
"Skill:<id>"; support skills carry require/add/excludeSkillTypes (SkillType
numbers), isTrigger when they add SkillType.Triggered, supportGemsOnly,
plusVersionOf (Empower-style + versions); active skills carry skillTypes
set, castTime (ms/1000), weaponTypes restriction, statDescriptionScope (the
stat-description file describing its tooltip), skillTotemId.

`Levels[level]`: array values (one per stat in statMapOrder; float stats
divided by their interpolation base), plus levelRequirement, cost
({resource=amount}; Vaal souls as cost.Soul), manaMultiplier (costMult-100),
cooldown (ms/1000), damageEffectiveness/baseMultiplier (dat/10000+1),
critChance (dat/100), statInterpolation codes per value (1=linear, 2=level
interpolated, 3=effectiveness-scaled — consumed by calc).

`StatMap` (per-skill overrides) and `data.SkillStatMap` (global fallback,
717 stats): stat id → mods + `div`/`mult` (per-minute stats div 60, ms div
1000, permyriad div 100), `value` override; entries can be groups (several
mods sharing one value). The fallback copy is lazy in the reference; a
skill's own statMap wins.

Gems: key "Metadata/Items/Gems/SkillGem<variantId>"; variantId ≠ gameId —
transfigured gems are variants of one gameId ("of Shrapnel" etc, AltX/AltY
granted-effect suffixes). Vaal gems: grantedEffect = the base skill,
secondaryGrantedEffect = the Vaal skill; PoB synthesizes `<gemId>AltX` gems
for Vaal versions of transfigured skills ("Vaal X of Y"). naturalMaxLevel
from ItemExperiencePerLevel (usually 20). Support-gem display names drop
" Support" except SupportBarrage (name collision with the Barrage skill).
Tag SET (gem.Tags) is what gemIsType queries: "active skill" =
grants_active_skill && !support; "non-vaal" = !vaal.

## Minions, bosses, misc

Minion (`data.Minions`, spectres merged in with limit ActiveSpectreLimit):
life/damage are multipliers (dat/100) applied to the per-level monster
tables (`MonsterLifeTable` etc, levels 1..100); attackTime ms/1000;
energyShield mult = 0.4×dat/100; resists are the Merciless columns;
damageFixup 0.11/0.22/0.33 from the SpeedAndDamageFixup mods; modList mods
get source "Minion:<name>".

Bosses: `Bosses[name] = {armourMult, evasionMult, isUber}`;
`BossStats` = 100 + mean over all/uber. `BossSkills["<Boss> <Skill>"]`:
DamageMultipliers[type] = {avgMult, spread} (damage as multiplier of
monsterBaseDamage at level 84; spread = damage range /100), uber variants
(UberDamageMultiplier /100), DamagePenetrations (+Uber set), speed (cast
ms, 700 = default, omitted), critChance default 5. The "enemy is Boss"
config numbers: standard = 4/4.40 of monster damage per type, pinnacle =
8/4.40 + 3% pen, uber = 10/4.25 + 8% pen — the 4.40 = 4 damage types + 40%
chaos.

Misc caps worth knowing (data.Misc, all archive-verified): MaxResistCap 90,
ResistFloor -200, EvadeChanceCap 95, SuppressionChanceCap 100 (effect 40%),
BlockChanceCap 90, AvoidChanceCap 75, DotDpsCap 35791394 (int32/60),
LowPoolThreshold 0.5, bleed 70%×5s, poison 30%×2s, ignite 90%×4s, server
tick 0.033s (trigger cooldowns snap to it — cook's sweep warning). Ailment
thresholds per non-damaging ailment in NonDamagingAilment (Chill default
10% cap 30, Shock 15% cap 50, Scorch/Brittle/Sap alt-ailments). Curse
priority: data.CursePriority (lower wins slots).

## Cluster and timeless jewels

Cluster (`data.ClusterJewels`): 3 sizes (Small 2-3 nodes, Medium 4-5,
Large 8-12); per size, `Skills` = the base-jewel skill types ("Added Small
Passive Skills grant: X"); notable legality = JewelCluster pool entries
whose weightKey contains the skill with weight > 0.
`ClusterJewelInfoForNotable[name]` inverts that: which sizes/skill types can
roll a given notable (this is Megalomaniac's variant list source).

Timeless: 11 types (5 legion + Heroic Tragedy + 5 abyss), seed ranges in
TimelessJewelSeedMin/Max (Elegant Hubris seeds are ÷20 internally).
LegionPassives node `oidx` values are PRNG layout junk, not data. Node
lookups go through binary LUTs (timeless-jewel-data module, unported).

## Node overrides: tattoos and runegrafts

One override system (`spec.hashOverrides`, pool = the committed
Data/TattooPassives.lua — NOT the GGPK-export document of similar name)
covers two game mechanics: tattoos (replace small/notable/keystone
passives) and runegrafts (overrideType `AlternateMastery`, replace
masteries). Saved as `<Override nodeId dn icon activeEffectImage>`; the
node keeps its numeric id, only content is replaced. Runegraft entries
carry their own `name` field; tattoo entries do not, so the original
node's name shows through (metatable nil-unshadowing). Overridden
masteries reparse under the numeric node id; other overrides keep the
pool entry's string id (e.g. Tree:Ramako3) as mod source.

## Text pipeline

Stat descriptions (Data/StatDescriptions/*.lua ← gamedata.StatDescs): per
stat-set, language entries with value-range limits (`#`=any, `!n`=not n,
`n|m` ranges) and format specials (canonical_line, reminderstring,
per-minute-to-per-second conversions, negate/divide handlers). A skill's
statDescriptionScope picks the file; `skillpopup_stat_filters.txt` maps
skills to scopes. Item mod text uses stat_descriptions; the describer engine
lives in export/statdesc.go (module stat-describer re-homes it).

Mod lines with `\n` inside them are real newlines at runtime (multi-line
mods). Unique items are text blobs (Data/Uniques) that the item parser
reads: `Variant:` lines enumerate versions, `{variant:N}` prefixes bind
lines to them, `{crafted}`/`{fractured}`/`{tags:...}`/`{range:X}` annotate
lines, `Implicits: N` splits implicits from explicits.
