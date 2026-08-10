# Influences

Hand-maintained legality rules, every build. Judge END STATES only: legal = can exist;
currency, crafting route and cost are never your concern - the affix-tier limits and the
conventions below are the only cost proxies, and the user owns them. `validate.lua` enforces
the end states; read this BEFORE authoring so items are legal in one pass.

## Influence types

- Six: Shaper, Elder, Crusader, Hunter, Redeemer, Warlord.
- Legal end states: none, one, or two DIFFERENT influences (doubles exist via Awakener's
  Orb); the item may then carry mods from each influence's pool.
- Influences only open prefix/suffix pools - never add, replace or interact with implicits.
- `affixes.lua` flags / PoB `influenceTags`: `shaper`, `elder`, `crusader`,
  `adjudicator`=Warlord, `basilisk`=Hunter, `eyrie`=Redeemer (inferred mapping).

## Eldritch items (Searing Exarch / Eater of Worlds)

An item that takes an eldritch implicit (ichor/ember) becomes an ELDRITCH ITEM - its own
category, not an influenced item. The two categories are mutually exclusive in both
directions: an influenced item can never become eldritch, an eldritch item can never take
influence. PoB flags: `cleansing`=Exarch, `tangle`=Eater.

### Eldritch implicits

- Slots: Body Armour, Helmet, Gloves, Boots ONLY. PoB's `ModEldritch.lua` also carries
  amulet spawn weights - NOT true in game; never author eldritch implicits anywhere else.
- At most 1 Exarch + 1 Eater implicit - never two from the same side. Side is
  `type = "Exarch"/"Eater"` in `src/Data/ModEldritch.lua`; check per id before pairing.
- No base implicit can coexist (Two-Toned Boots lose their res, Gripped Gloves their
  projectile damage). Enchants are not implicits and coexist fine.
- Uniques cannot have eldritch implicits (a few ship with them built in, e.g. The Eternal
  Struggle).
- Tiers t1-t6 (`...EldritchImplicit1-6` ids, Lesser through Perfect). Legal pairs: up to
  t4/t4; stretch states t5/t4 and t6/t4. Both sides t5+ do not exist. A lone eldritch
  implicit is legal but never authored - fill both sides, up to t4, like any free slot.
- Convention (mirrors the affix rule): t4/t4 standard; a t5/t6 dominant side is the stretch
  case - disclose it in the report like a T1 affix.
