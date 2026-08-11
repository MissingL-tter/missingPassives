Standing user preferences, every build, no exceptions. A recipe overrides one only by saying
so explicitly; your own judgement never does.

## Character level 95

Every build is authored at character level 95 - a build that only works at 100 is
unreasonable. Life, mana, accuracy and the 117+extras passive budget all follow from the
level; `validate.lua` enforces both the level and the budget.

## Cost is not your concern

Judge items purely by legal obtainability of the END state - never by currency, crafting
route or market price. The T2 `{range:1}` / T1 `{range:0.85}` affix rule and the tier
conventions in `influences.md` are the only cost proxies; apply them and move on.

## Mageblood flasks

Every Mageblood build includes at least one movement-speed flask among the constantly-applied
slots - no one wants to be slow on a Mageblood character. Silver Flask by default; when the
build deliberately constrains attack or cast rate (trigger cooldowns, spend-throttled
casting), Onslaught's rate portion is wasted there - use a Quicksilver instead.

## Gear mods

Never author a mod gated on an enemy or an arena the character is not always in - implicits,
affixes and bench crafts alike, mostly Eldritch (Searing Exarch / Eater of Worlds) implicits.
DPS is measured against a level 84 Pinnacle boss, so these count in every figure reported and
in almost no map - the number stops describing play.

- `while a Unique Enemy is Nearby`
- `while a Rare or Unique Enemy is Nearby`
- `While a Pinnacle Atlas Boss is in your Presence`
- any `while in <boss>'s arena` / pinnacle-arena variant
