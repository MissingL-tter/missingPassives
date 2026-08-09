---
name: cook
description: Build a Path of Exile 1 character in Path of Building from a recipe file and verify it against hard constraints.
disable-model-invocation: true
---

Build a PoE1 character meeting the request named by the argument, verified in headless PoB,
saved as an `.xml` in `src/Builds/`. Filename = tree version, skill, ascendancy, defence:
`3.29 CoC Blazing Salvo Assassin CI.xml`, never `salvo.xml`. Ask before overwriting an
existing file in `src/Builds/`.

Sources: headless PoB for anything measurable, `data/` for what is obtainable, the web for
mechanics. Ask questions as they arise.

Every build starts from an empty file; never skeleton off another `src/Builds/` build. Path
the tree fresh; derive gear and links from the recipe.

## The request

The argument names a file in this skill's `recipes/`, without extension: `/cook cocnova`
reads `recipes/cocnova.txt`. Read it before anything else.

- No argument: list `recipes/*.txt` minus `template.txt`, ask which to cook.
- No such file: say so, list what is there, offer to make one from `template.txt`.
- `template.txt` is the blank form, never itself a build.
- Fields: `CLASS`, `ASCENDANCY`, `SKILL`, `SKILL TYPE`, `DPS`, `REQUIRED ITEMS`,
  `DEFENCE MINIMUMS`, `NOTES`. Text after `#` is a comment.
- Blank field = yours to decide - choose well and say what you chose and why.
- `SKILL TYPE` applies only when `SKILL` is blank.
- Entirely blank file: point at it rather than inventing a request.

Echo the request back as a constraint checklist before building; at the end report a
measured figure against every line, shortfalls included.

## Measuring

DPS is always against a level 84 Pinnacle boss:

```
config: enemyLevel = 84, enemyIsBoss = "Pinnacle"
stat:   mainOutput.WithDotDPS or mainOutput.TotalDPS   -- WithDotDPS is nil without a DoT
```

Never report `CombinedDPS` (it adds culling). Every figure comes from `mainOutput`, never
from adding up mods by hand. Sweep permutations instead of reasoning about them - trigger
builds do not behave as the mod text reads. Probe hypothetical stats via
`configTab.input.customMods = "..."` + `configTab:BuildModList()` before hunting them on
gear: trigger cooldowns snap to server ticks, so a stat can be +24% DPS at one number and
zero just below it.

## Items

Never hand-write a mod line on a rare. Author affix ids and let PoB write the text:

```
Rarity: RARE
Frost Grip
Sorcerer Gloves
Crafted: true
Item Level: 85
Prefix: {range:1}LocalIncreasedEnergyShield6
Suffix: {range:1}ColdResist7
Implicits: 1
While a Pinnacle Atlas Boss is in your Presence, Inflict Fire Exposure on Hit, applying -22% to Fire Resistance
```

- `{range:N}` is the roll position, 0..1 across the mod's value range. Find ids with
  `affixes.lua`; it marks T1/T2 per group.
- Author the **T2 tier at `{range:1}`** - realistic, with T1 left as upgrade room. Use T1
  only when the recipe cannot be met without it, capped at `{range:0.85}`, and say so in
  the report.
- `Prefix:`/`Suffix:` lines are header properties - they MUST come before `Implicits: N`,
  which switches the parser to mod-text mode and turns later property lines into dead
  literal text.
- Authoring is not crafting: `Item:Craft()` is UI-only, so an authored item contributes
  ZERO to calcs until `craft.lua` runs. To change an item, edit its `Prefix:`/`Suffix:`
  lines in the XML and re-run `craft.lua`. Never rebuild items from a crafted file's raw
  text in a script - each cycle re-parses generated lines back into the affix lists.
- Legality is spawn weight, not pool membership: armour bases fall back to a pool holding
  every mod in the game, so a mod can be listed for a base and still be impossible there.
- Bench crafts have no spawn weight - add those as `{crafted}` text lines.

In PoB's data but not in the game (`validate.lua` rejects all of it):

- Labyrinth enchants, all four difficulties - on gloves every tier from "Word of" to
  "Commandment". Harvest / Heist / Dedication / Instilling / Enkindling enchants and
  amulet anoints share the `{enchant}` tag and stay legal.
- Scourge (Hellscape) mods - still in the pools at full spawn weight.
- Elevated (Maven) influence mods - technically obtainable, too expensive to assume.

PoB does not enforce, while you type: no two affixes may share a mod `group`; one
bench craft unless the item has "Can have up to 3 Crafted Modifiers"; an unknown affix id
is silently dropped; affixes past 3 prefixes / 3 suffixes are silently ignored.
`validate.lua` catches all of these, plus inert never-crafted items, T1 rolls over 0.85,
and tree budgets (122+extras points; 8 ascendancy, Bloodline included).

## Headless mutation

For sweeps, mutate the loaded build and `pob.refresh()`; `pob.save()` only when a change is
accepted. Gotchas:

- Gem swap: set `skillId`/`variantId`/`nameSpec`/`level` AND nil `gemId`, `gemData`,
  `grantedEffect` - a stale `gemId` silently wins re-resolution and the swap reverts.
  Then `skillsTab:ProcessSocketGroup(group)`.
- `spec:AllocNode(node)` allocates the whole path; `spec:DeallocNode(node)` drops
  dependents. Check `spec:CountAllocNodes()` after both.
- A `secondaryAscendClassId` attribute re-allocates that Bloodline's start node on every
  load. Removing the node is not enough; remove the attribute.
- Runtime items via `new("Item", raw)` need `item:Craft()` to act, and the authored raw,
  not a crafted file's. Added items also need `t_insert(itemsTab.itemOrderList, id)` or
  `SaveDB` drops them; slot assignments save from item sets, not `slot.selItemId`.

## Data

Generated from PoB's database; regenerate with the matching `tools/dump-*.lua`. Check
ingredients here *before* designing around them.

- `data/gems.md` - unobtainable gems, upgraded support tiers, level caps. Read every build.
- `data/skills.md` - obtainable active gems, with tags.
- `data/supports.md` - obtainable supports, with tags and descriptions.
- `data/uniques.md` - unobtainable uniques, variant traps, Foulborn.
- `data/ascendancies.md` - every ascendancy and Bloodline node, and the shared 8-point rule.
- `data/cluster-jewels.md` - sizes, skills, notables with stats, the authoring recipe,
  measured point economics. Read before pathing any tree - a Large cluster whose enchant
  stat is live competes with the tree's own wheels.

## Tools

Run from `src/`:

```sh
luajit ../.claude/skills/cook/tools/validate.lua "Builds/My Build.xml"
```

- `pob.lua` - headless bootstrap (`load`, `refresh`, `save`), required by the others.
- `affixes.lua "<base>" [pattern] [--tiers] [--shaper ...] [--skill=<tag>]` - legal affix
  ids for a base; cluster jewel bases need `--skill` (bare, it lists the base's skills).
- `craft.lua` - bakes authored affix ids into real mods.
- `validate.lua` - legality check; exits non-zero on any problem.
- `export.lua` - uploads the build to pobb.in, prints the link.
- `dump-gems.lua`, `dump-uniques.lua`, `dump-ascendancies.lua`, `dump-cluster-jewels.lua` -
  regenerate `data/`.

The loop: author -> `craft.lua` -> `validate.lua` -> measure -> edit affix lines ->
`craft.lua` again. Never measure between authoring and crafting.

Deliver every finished build as BOTH the `.xml` path and a pobb.in link from `export.lua`,
run after the final craft + validate so the paste matches the file.
