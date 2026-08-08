---
name: cook
description: Build a Path of Exile 1 character in Path of Building from a build file and verify it against hard constraints. Use ONLY when the user explicitly invokes it by name (/cook) - never trigger it from a passing mention of builds, gems, items or Path of Building.
---

Build a Path of Exile 1 character meeting the request named by the argument, verified in
headless Path of Building, and save it as an `.xml` in `src/Builds/` so it opens in PoB.

Use any resources available - headless PoB for anything measurable, `data/` for what is
obtainable, the web for mechanics and interactions. Ask any questions as they arise.

## The request

The argument names a file in this skill's `recipes/` directory, without the extension:
`/cook cocnova` reads `recipes/cocnova.txt`. Read it first, before anything else.

- No argument: list the `recipes/*.txt` files, minus `template.txt`, and ask which to cook.
- No such file: say so, list what is there, and offer to make one from `template.txt`.
- `template.txt` is the blank form and is never itself a build. Copy it to start a new one.

Each file holds `CLASS`, `ASCENDANCY`, `SKILL`, `SKILL TYPE`, `DPS`, `REQUIRED ITEMS`,
`DEFENCE MINIMUMS`, `NOTES`. Text after a `#` is a comment.

A blank field is yours to decide - choose well and say what you chose and why.
`SKILL TYPE` only applies when `SKILL` is blank. If the file is entirely blank, point at
it rather than inventing a request.

Echo the request back as a constraint checklist before building, then report a measured
figure against every line at the end, including any that fell short.

## Measuring

DPS is always measured against a level 84 Guardian/Pinnacle boss.

```
config: enemyLevel = 84, enemyIsBoss = "Pinnacle"
stat:   mainOutput.WithDotDPS      -- "Total DPS inc. DoT"
```

Every figure, offensive or defensive, comes from `mainOutput` rather than from adding
up mods by hand. Sweep permutations in headless PoB instead of reasoning about them -
trigger builds in particular do not behave the way the mod text reads.

## Items

Never hand-write a mod line on a rare. Author it as affix ids and let PoB write the text:

```
Rarity: RARE
Frost Grip
Sorcerer Gloves
Crafted: true
Prefix: {range:1}LocalIncreasedEnergyShield7
Suffix: {range:1}ColdResist8
```

`{range:1}` rolls the top of the range, `0.5` the middle. Find ids with `affixes.lua`.
Typing values by hand is how a body-armour mod ended up on four other slots.

Legality is spawn weight, not pool membership: armour bases fall back to a pool holding
every mod in the game, so a mod can be listed for the base and still be impossible there.
Bench crafts have no spawn weight - add those as `{crafted}` text lines.

Rules PoB will not enforce while you type: no two affixes may share a mod `group`, and an
item gets one bench craft unless it carries "Can have up to 3 Crafted Modifiers". Two
failures are also silent - an unknown affix id is dropped, and affixes past 3 prefixes or
3 suffixes are ignored. Run `validate.lua` before delivering; it catches all of these.

## Data

Generated from PoB's own database; regenerate with the matching `tools/dump-*.lua`.
Check ingredients here before designing around them - a gem or unique that turns out to
be unobtainable after the build is tuned costs a rebuild.

- `data/gems.md` - unobtainable gems, upgraded support tiers, level caps. Read every build.
- `data/skills.md` - obtainable active gems, with tags.
- `data/supports.md` - obtainable supports, with tags and descriptions.
- `data/uniques.md` - unobtainable uniques, variant traps, Foulborn.
- `data/ascendancies.md` - every ascendancy and Bloodline node, and the shared 8-point rule.

## Tools

Run from `src/`:

```sh
luajit ../.claude/skills/cook/tools/validate.lua "Builds/My Build.xml"
```

- `pob.lua` - headless bootstrap, required by the others.
- `affixes.lua "<base>" [pattern]` - legal affix ids for a base, with tiers and ranges.
- `validate.lua` - obtainability check; exits non-zero on any problem.
- `dump-gems.lua`, `dump-uniques.lua`, `dump-ascendancies.lua` - regenerate `data/`.

Builds save to `src/Builds/`, settings to `src/`. Existing builds there are real work -
do not overwrite without asking.
