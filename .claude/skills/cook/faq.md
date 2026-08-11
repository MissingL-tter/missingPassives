# FAQ - the did-you-miss-it checklist

Run against every build BEFORE the final craft/validate/export. Each entry was actually
missed once.

## Free value left on the table

- Flask enchants: Enkindling `70% increased effect / Gains no Charges during Effect` on
  every Mageblood flask - the downside is dead when flasks cannot be Used.
- Amulet anoint (`{enchant}Allocates <notable>`) - a free notable, sweep it.
- Eldritch implicits on ALL FOUR of body/helmet/gloves/boots, BOTH sides each, t4/t4
  (`influences.md`). A lone implicit or empty slot is never correct.
- Every prefix/suffix slot filled or its emptiness justified in the report - no `None`
  padding on delivered items.
- One bench craft per item where a prefix/suffix is free (`{crafted}` line).
- Quality 20 on gems, armour, flasks; corrupt-level 21 on actives and auras that gain from
  it - but check `naturalMaxLevel` first (Frostmage and Greater Spell Echo cap at 3+1, not
  20+1).
- All tree jewel sockets and cluster proxy sockets filled - verify MECHANICALLY (iterate
  allocated Socket nodes against `spec.jewels`); eyeballing shipped an empty one.

## Reading and reasoning errors

- An item or gem name from memory is a HYPOTHESIS. Confirm against a whitelist before
  designing around it: uniques via the Obtainable section of `data/uniques.md`, gems via
  `data/skills.md` / `data/supports.md`. One grep - hit means exists and legal, miss means
  no, whatever the reason. Never reason from absence in a trap-list.
- Defensive architecture is a search space, same as supports: when a defence line is hard
  to reach, enumerate and measure competing FRAMES (which aura set, which conversion or
  cap mechanic, which unique enabler) before grinding increments inside the frame you
  started with. Reasoned-not-swept is how a dominant frame gets missed.
- Mechanics that scale on RESERVED amounts invert normal aura logic (reservation
  efficiency lowers reserved; "wasted" reservation is fuel). When a mechanic keys off an
  unusual quantity, re-derive what helps from scratch instead of importing habits.
- "Spend/spent <resource>" modifiers require spending EXACTLY that resource - anything
  that redirects payment silently disables them. Eldritch Battery pays from ES (ES paid
  is not mana spent: kills Indigon, Arcane Surge support, "per X mana spent recently"
  nodes); Blood Magic / Lifetap pay life, same effect; the litany is long. Before
  crediting any spend-scaling mod, trace which pool each cost actually leaves.

## Silent breakage

- After ANY gem-list edit: re-check `skillPart` and `mainActiveSkill` - they silently reset
  (Ball Lightning part 1 / wrong main skill regression).
- Conditional mods must actually fire: probe with `customMods` (Intolerance of Sin sat dead
  below 150 devotion; PoB never warned).
- Attribute requirements: `ReqStr/ReqDex/ReqInt` vs actual - PoB computes DPS even when
  gems could not level.
- `ManaUnreserved >= 0`, and for trigger builds the TRIGGERED skill's cost is real
  (post-3.15) - check sustain, not just the cyclone/attack row.
- Attacks stop when their cost is unpayable even with "insufficient mana doesn't prevent
  spells" tech - the attack's cost must be literally 0.

## Report-time checks

- All four resistances (chaos too) and all four max-hit lines measured, not just the ones
  the recipe names.
- Trigger builds: trigger rate vs cap, and attack-speed ALIGNMENT (more AS can lower DPS).
- Charges assumed in config have a generation source that works against the configured
  enemy (kills don't happen vs a pinnacle boss).
- With Mageblood equipped, the leftmost N magic utility flasks are ALWAYS active - the
  flask "active" checkbox is irrelevant to them, in game and in PoB. Don't set it, don't
  read anything into it in someone else's save.
