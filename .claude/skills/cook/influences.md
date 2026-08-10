# Influences

Hand-maintained legality rules binding every build. Judge **end states only**: an item is
legal if it can exist; currencies, crafting routes and cost are never your concern - the
affix-tier limits and the conventions below are the only cost proxies, and the user owns
them. `validate.lua` does NOT enforce anything on this page.

## Influence types

- Six influences: **Shaper, Elder, Crusader, Hunter, Redeemer, Warlord**.
- Legal end states: an item has none, one, or two DIFFERENT influences (doubles exist via
  Awakener's Orb) and may then carry mods from each influence's pool.
- Influences only open prefix/suffix pools - they never add, replace or interact with
  implicits.
- **Fractured items can never be influenced.**
- `affixes.lua` flags / PoB `influenceTags`: `shaper`, `elder`, `crusader`,
  `adjudicator`=Warlord, `basilisk`=Hunter, `eyrie`=Redeemer (inferred mapping),
  `cleansing`=Searing Exarch, `tangle`=Eater of Worlds.

## Eldritch implicits (Searing Exarch / Eater of Worlds)

- **Slots: Body Armour, Helmet, Gloves, Boots only.** PoB's `ModEldritch.lua` also carries
  amulet spawn weights - NOT true in game; never author eldritch implicits anywhere else.
- Eldritch implicits and influence **never coexist on one item**, in either direction.
  Items with eldritch implicits are not "influenced".
- At most **1 Exarch + 1 Eater implicit** - never two from the same side. The side is
  `type = "Exarch"/"Eater"` in `src/Data/ModEldritch.lua`; check per id before pairing.
- **No base implicit can coexist** with an eldritch implicit (Two-Toned Boots lose their
  res, Gripped Gloves their projectile damage). Enchants are not implicits and coexist
  fine.
- **Uniques cannot have eldritch implicits** (a few ship with them built in, e.g. The
  Eternal Struggle).
- Tiers t1-t6 (`...EldritchImplicit1-6` ids, Lesser through Perfect). Legal pairs: up to
  **t4/t4**; stretch states **t5/t4** and **t6/t4**. Both sides t5+ do not exist. A lone
  eldritch implicit is legal but never authored - fill both sides, up to t4, like any free
  slot.
- Convention (mirrors the affix rule): **t4/t4 standard**; a t5/t6 dominant side is the
  stretch case - disclose it in the report like a T1 affix.
