package modparser

import "math"

// specialModList entries whose Lua closures need real statements — branches,
// loops, helper calls with option tables. Each is a direct transcription of
// the cited ModParser.lua lines. They are merged over the transformed entries
// when the scan table is built; keys are identical to the reference's.

// conquerorList — ModParser.lua:41.
var conquerorList = map[string]Tag{
	`xibaqua`:  {"id": 1, "type": "vaal"},
	`zerphi`:   {"id": 2, "type": "vaal"},
	`doryani`:  {"id": 3, "type": "vaal"},
	`ahuana`:   {"id": "2_v2", "type": "vaal"},
	`deshret`:  {"id": 1, "type": "maraketh"},
	`asenath`:  {"id": 2, "type": "maraketh"},
	`nasima`:   {"id": 3, "type": "maraketh"},
	`balbala`:  {"id": "1_v2", "type": "maraketh"},
	`cadiro`:   {"id": 1, "type": "eternal"},
	`victario`: {"id": 2, "type": "eternal"},
	`chitus`:   {"id": 3, "type": "eternal"},
	`caspiro`:  {"id": "3_v2", "type": "eternal"},
	`kaom`:     {"id": 1, "type": "karui"},
	`rakiata`:  {"id": 2, "type": "karui"},
	`kiloava`:  {"id": 3, "type": "karui"},
	`akoya`:    {"id": "3_v2", "type": "karui"},
	`venarius`: {"id": 1, "type": "templar"},
	`dominus`:  {"id": 2, "type": "templar"},
	`avarius`:  {"id": 3, "type": "templar"},
	`maxarius`: {"id": "1_v2", "type": "templar"},
	`vorana`:   {"id": 1, "type": "kalguur"},
	`uhtred`:   {"id": 2, "type": "kalguur"},
	`medved`:   {"id": 3, "type": "kalguur"},
	`tecrod`:   {"id": 1, "type": "abyss_murderous"},
	`ulaman`:   {"id": 1, "type": "abyss_searching"},
	`kurgal`:   {"id": 1, "type": "abyss_hypnotic"},
	`amanamu`:  {"id": 1, "type": "abyss_ghastly"},
	`zorath`:   {"id": 1, "type": "abyss_special"},
}

// conqueredBy — the shared body of the eight timeless jewel entries at
// ModParser.lua:5776-5799.
func conqueredBy(c caps) any {
	value := Tag{"id": c.n(1)}
	if conq, ok := conquerorList[asciiLower(c.s(2))]; ok {
		value["conqueror"] = conq
	}
	return []any{mod("JewelData", "LIST", Tag{"key": "conqueredBy", "value": value})}
}

// selfCastVar builds the "SelfCast<Curse>" condition var used by the
// "if you've cast X in the past N seconds" family.
func selfCastVar(curse string) string {
	return "SelfCast" + condenseName(curse)
}

var specialModListHand = map[string]any{
	// --- Explode mods — ModParser.lua:2121-2186 ---
	`enemies you kill have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: fn(func(c caps) any { // Obliteration, Unspeakable Gifts, synth implicit, crusader body, Ngamahu Warmonger tattoo
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies you kill have ?a? ?([0-9]+)% chance to explode, dealing ([0-9]+)% of their maximum life as (.+) damage`: fn(func(c caps) any { // Hinekora, Death's Fury 3.22
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies you or your totems kill have ([0-9]+)% chance to explode, dealing ([0-9]+)% of their maximum life as (.+) damage`: fn(func(c caps) any { // Hinekora, Death's Fury 3.23
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies you kill while using pride have ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: fn(func(c caps) any { // Sublime Vision
		return explodeFunc(c.n(1), c.s(2), c.s(3), Tag{"type": "Condition", "var": "AffectedByPride"})
	}),
	`enemies you kill during effect have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as damage of a random element`: fn(func(c caps) any { // Oriath's End
		return explodeFunc(c.n(1), c.s(2), "randomElement", Tag{"type": "Condition", "var": "UsingFlask"})
	}),
	`enemies you kill while affected by glorious madness have a ([0-9]+)% chance to explode, dealing a (.+) of their life as (.+) damage`: fn(func(c caps) any { // Beacon of Madness
		return explodeFunc(c.n(1), c.s(2), c.s(3), Tag{"type": "Condition", "var": "AffectedByGloriousMadness"})
	}),
	`enemies killed with attack hits have a ([0-9]+)% chance to explode, dealing a (.+) of their life as (.+) damage`: fn(func(c caps) any { // Devastator
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies killed with wand hits have a ([0-9]+)% chance to explode, dealing a (.+) of their life as (.+) damage`: fn(func(c caps) any { // Explosive Force
		return explodeFunc(c.n(1), c.s(2), c.s(3), Tag{"type": "Condition", "var": "UsingWand"})
	}),
	`cursed enemies you or your minions kill have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: fn(func(c caps) any { // Profane Bloom
		return explodeFunc(c.n(1), c.s(2), c.s(3), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})
	}),
	`enemies you kill explode, dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // legacy synth, legacy crusader
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`enemies killed explode dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // Quecholli
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`enemies on fungal ground you kill explode, dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // Sporeguard
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "ActorCondition", "actor": "enemy", "var": "OnFungalGround"})
	}),
	`enemies killed with attack or spell hits explode, dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // Shaper 2H mace mod
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`shocked enemies you kill explode, dealing ([0-9]+)% of their life as (.+) damage which cannot shock`: fn(func(c caps) any { // Inpulsa's Broken Heart
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Shocked"})
	}),
	`ignited enemies you kill explode, dealing ([0-9]+)% of their life as (.+) damage which cannot ignite`: fn(func(c caps) any { // Inpulsa's Broken Heart
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Ignited"})
	}),
	`bleeding enemies you kill explode, dealing ([0-9]+)% of their maximum life as (.+) damage`: fn(func(c caps) any { // Haemophilia
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Bleeding"})
	}),
	`burning enemies you kill have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: fn(func(c caps) any { // Haemophilia
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Burning"})
	}),
	`enemies killed near corpses affected by your curses explode, dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // Asenath's Gentle Touch
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "MultiplierThreshold", "var": "NearbyCorpse", "threshold": 1}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Cursed"})
	}),
	`enemies taunted by your warcries explode on death, dealing ([0-9]+)% of their maximum life as (.+) damage`: fn(func(c caps) any { // Al Dhih
		return explodeFunc(100, c.s(1), c.s(2), Tag{"type": "ActorCondition", "actor": "enemy", "var": "Taunted"}, Tag{"type": "Condition", "var": "UsedWarcryRecently"})
	}),
	`totems explode on death, dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // Crucible weapon mod
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`nearby corpses explode when you warcry, dealing ([0-9]+)% of their life as (.+) damage`: fn(func(c caps) any { // Ruthless Berserker node
		return explodeFunc(100, c.s(1), c.s(2))
	}),

	// --- Elemental Equilibrium — ModParser.lua:2415-2430 ---
	`enemies you hit with elemental damage temporarily get (\+[0-9]+)% resistance to those elements and (-[0-9]+)% resistance to other elements`: fn(func(c caps) any {
		plus, minus := c.n(1), c.n(2)
		return []any{
			flag("ElementalEquilibrium"),
			flag("ElementalEquilibriumLegacy"),
			mod("EnemyModifier", "LIST", Tag{"mod": mod("FireResist", "BASE", plus, Tag{"type": "Condition", "var": "HitByFireDamage"})}),
			mod("EnemyModifier", "LIST", Tag{"mod": mod("FireResist", "BASE", minus, Tag{"type": "Condition", "var": "HitByFireDamage", "neg": true}, Tag{"type": "Condition", "varList": []any{"HitByColdDamage", "HitByLightningDamage"}})}),
			mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdResist", "BASE", plus, Tag{"type": "Condition", "var": "HitByColdDamage"})}),
			mod("EnemyModifier", "LIST", Tag{"mod": mod("ColdResist", "BASE", minus, Tag{"type": "Condition", "var": "HitByColdDamage", "neg": true}, Tag{"type": "Condition", "varList": []any{"HitByFireDamage", "HitByLightningDamage"}})}),
			mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningResist", "BASE", plus, Tag{"type": "Condition", "var": "HitByLightningDamage"})}),
			mod("EnemyModifier", "LIST", Tag{"mod": mod("LightningResist", "BASE", minus, Tag{"type": "Condition", "var": "HitByLightningDamage", "neg": true}, Tag{"type": "Condition", "varList": []any{"HitByFireDamage", "HitByColdDamage"}})}),
		}
	}),

	// --- ModParser.lua:2370: gain no X from equipped Y ---
	`gain no (.+) from equipped (.+)`: fn(func(c caps) any {
		stat, slot := c.s(1), c.s(2)
		if slot == "shield" {
			slot = "Weapon 2"
		}
		return []any{flag("GainNo" + stripSpaces(titleWords(stat)) + "From" + titleWords(slot))}
	}),

	// --- ModParser.lua:2500 (approx): all damage bypasses energy shield ---
	// The Chaos entry allows overriding "chaos damage does not bypass energy shield".
	`all damage taken bypasses energy shield`: []any{
		mod("PhysicalEnergyShieldBypass", "OVERRIDE", 100),
		mod("LightningEnergyShieldBypass", "OVERRIDE", 100),
		mod("ColdEnergyShieldBypass", "OVERRIDE", 100),
		mod("FireEnergyShieldBypass", "OVERRIDE", 100),
		mod("ChaosEnergyShieldBypass", "OVERRIDE", 100),
	},

	// --- Armour Mastery — ModParser.lua:2445-2475 ---
	`([0-9]+)% chance to defend with double your armour for each time you've been hit by an enemy recently, up to ([0-9]+)%`: fn(func(c caps) any {
		numChance, cap := c.n(1), c.n(2)
		return []any{
			mod("ArmourDefense", "MAX", 100, "Armour Mastery: Max Calc", Tag{"type": "Condition", "var": "ArmourMax"}),
			mod("ArmourDefense", "MAX", math.Min(numChance/100, 1.0)*100, "Armour Mastery: Average Calc", Tag{"type": "Condition", "var": "ArmourAvg"}, Tag{"type": "Multiplier", "var": "BeenHitRecently", "limit": cap / numChance}),
			mod("ArmourDefense", "MAX", math.Min(math.Floor(numChance/100), 1.0)*100, "Armour Mastery: Min Calc", Tag{"type": "Condition", "var": "ArmourMax", "neg": true}, Tag{"type": "Condition", "var": "ArmourAvg", "neg": true}, Tag{"type": "Multiplier", "var": "BeenHitRecently", "limit": cap / numChance}),
		}
	}),
	`([0-9]+)% chance to defend with ([0-9]+)% of armour for each time you've been hit by an enemy recently, up to ([0-9]+)%`: fn(func(c caps) any {
		numChance, numArmourMultiplier, cap := c.n(1), c.n(2), c.n(3)
		return []any{
			mod("ArmourDefense", "MAX", numArmourMultiplier-100, "Armour Mastery: Max Calc", Tag{"type": "Condition", "var": "ArmourMax"}),
			mod("ArmourDefense", "MAX", math.Min(numChance/100, 1.0)*(numArmourMultiplier-100), "Armour Mastery: Average Calc", Tag{"type": "Condition", "var": "ArmourAvg"}, Tag{"type": "Multiplier", "var": "BeenHitRecently", "limit": cap / numChance}),
			mod("ArmourDefense", "MAX", math.Min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100), "Armour Mastery: Min Calc", Tag{"type": "Condition", "var": "ArmourMax", "neg": true}, Tag{"type": "Condition", "var": "ArmourAvg", "neg": true}, Tag{"type": "Multiplier", "var": "BeenHitRecently", "limit": cap / numChance}),
		}
	}),
	`([0-9]+)% chance to defend with double armour`: fn(func(c caps) any {
		numChance := c.n(1)
		return []any{
			mod("ArmourDefense", "MAX", 100, "Armour Mastery: Max Calc", Tag{"type": "Condition", "var": "ArmourMax"}),
			mod("ArmourDefense", "MAX", math.Min(numChance/100, 1.0)*100, "Armour Mastery: Average Calc", Tag{"type": "Condition", "var": "ArmourAvg"}),
			mod("ArmourDefense", "MAX", math.Min(math.Floor(numChance/100), 1.0)*100, "Armour Mastery: Min Calc", Tag{"type": "Condition", "var": "ArmourMax", "neg": true}, Tag{"type": "Condition", "var": "ArmourAvg", "neg": true}),
		}
	}),
	`([0-9]+)% chance to defend with ([0-9]+)% of armour`: fn(func(c caps) any {
		numChance, numArmourMultiplier := c.n(1), c.n(2)
		return []any{
			mod("ArmourDefense", "MAX", numArmourMultiplier-100, "Armour Mastery: Max Calc", Tag{"type": "Condition", "var": "ArmourMax"}),
			mod("ArmourDefense", "MAX", math.Min(numChance/100, 1.0)*(numArmourMultiplier-100), "Armour Mastery: Average Calc", Tag{"type": "Condition", "var": "ArmourAvg"}),
			mod("ArmourDefense", "MAX", math.Min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100), "Armour Mastery: Min Calc", Tag{"type": "Condition", "var": "ArmourMax", "neg": true}, Tag{"type": "Condition", "var": "ArmourAvg", "neg": true}),
		}
	}),

	// --- Radius jewel transforms with parametric radius — ModParser.lua:2611,2619 ---
	`non-unique jewels cause increases and reductions to other damage types in a ([a-zA-Z]+) radius to be transformed to apply to ([a-zA-Z]+) damage`: fn(func(c caps) any {
		dmgType := c.s(2)
		inner := getSimpleConv([]string{"PhysicalDamage", "FireDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"}, firstToUpper(dmgType)+"Damage", "INC", true, 0, "")
		wrapped := jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
			// Ignore Timeless Jewels
			if n, ok := node.(jewelNode); ok && !n.ConqueredBy() {
				inner(node, out, data)
			}
		})
		return []any{mod("ExtraJewelFunc", "LIST",
			Tag{"radius": firstToUpper(c.s(1)), "type": "Other", "func": wrapped},
			Tag{"type": "ItemCondition", "itemSlot": "{SlotName}", "rarityCond": "UNIQUE", "neg": true},
			Tag{"type": "ItemCondition", "itemSlot": "{SlotName}", "rarityCond": "RELIC", "neg": true})}
	}),
	`non-unique jewels cause small and notable passive skills in a ([a-zA-Z]+) radius to also grant \+([0-9]+) to ([a-zA-Z]+)`: fn(func(c caps) any {
		val, attr := c.n(2), c.s(3)
		attrShort := firstToUpper(attr)
		if len(attrShort) > 3 {
			attrShort = attrShort[:3] // firstToUpper(attr):match("^%a%l%l")
		}
		wrapped := jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
			if n, ok := node.(jewelNode); ok && !n.ConqueredBy() && (n.Type() == "Notable" || n.Type() == "Normal") {
				out.NewMod(attrShort, "BASE", val, data["modSource"])
			}
		})
		return []any{mod("ExtraJewelFunc", "LIST",
			Tag{"radius": firstToUpper(c.s(1)), "type": "Other", "func": wrapped},
			Tag{"type": "ItemCondition", "itemSlot": "{SlotName}", "rarityCond": "UNIQUE", "neg": true},
			Tag{"type": "ItemCondition", "itemSlot": "{SlotName}", "rarityCond": "RELIC", "neg": true})}
	}),

	// --- ModParser.lua:2890: arcane surge ailment effect ---
	`non-damaging ailments have ([0-9]+)% reduced effect on you while you have arcane surge`: fn(func(c caps) any {
		var mods []any
		for _, ailment := range nonDamagingAilmentTypeList {
			mods = append(mods, mod("Self"+ailment+"Effect", "INC", -c.n(1), Tag{"type": "Condition", "var": "AffectedByArcaneSurge"}))
		}
		return mods
	}),

	// --- ModParser.lua:2931,4992: capped multipliers dividing two captures ---
	`([0-9]+)% increased attack and cast speed for each corpse consumed recently, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Speed", "INC", c.n(1), Tag{"type": "Multiplier", "var": "CorpseConsumedRecently", "limit": c.n(2) / c.n(1)})}
	}),
	`([0-9]+)% increased armour per second you've been stationary, up to a maximum of ([0-9]+)%`: fn(func(c caps) any {
		return []any{mod("Armour", "INC", c.n(1), Tag{"type": "Multiplier", "var": "StationarySeconds", "limit": c.n(2) / c.n(1)}, Tag{"type": "Condition", "var": "Stationary"})}
	}),

	// --- ModParser.lua:3097: display-only ---
	`can be allflame crafted as if rare`: []any{}, // Display only. For Dread Captain's Cutlass and supported by crafting UI

	// --- Gem property entries with branches — ModParser.lua:3220-3235 ---
	`([+\-][0-9]+)%? to ([a-zA-Z]+) of socketed ?([a-zA-Z\- ]*) gems`: fn(func(c caps) any {
		typ := c.s(3)
		if typ == "" {
			typ = "all"
		}
		return []any{mod("GemProperty", "LIST", Tag{"keyword": typ, "key": c.s(2), "value": c.n(1)}, Tag{"type": "SocketedIn", "slotName": "{SlotName}"})}
	}),
	`([+\-][0-9]+)%? to ([a-zA-Z]+) of all (.+) gems`: fn(func(c caps) any {
		skill := c.s(3)
		if _, ok := gemIdLookup[skill]; ok {
			return []any{mod("GemProperty", "LIST", Tag{"keyword": skill, "key": "level", "value": c.n(1), "keyOfScaledMod": "value"})}
		}
		var wordList []any
		for _, word := range splitWords(skill) {
			wordList = append(wordList, word)
		}
		return []any{mod("GemProperty", "LIST", Tag{"keywordList": wordList, "key": c.s(2), "value": c.n(1), "keyOfScaledMod": "value"})}
	}),

	// --- ModParser.lua:3295: phantasm special case ---
	`trigger level ([0-9]+) (.+) when you consume a corpse`: fn(func(c caps) any {
		if c.s(2) == "summon phantasm skill" {
			return triggerExtraSkill("triggered summon phantasm skill", c.n(1))
		}
		return triggerExtraSkill(c.s(2), c.n(1))
	}),

	// --- Item-granted curses — ModParser.lua:3340-3355,4083 ---
	`curse enemies with ([^0-9]+) on [a-zA-Z]+, with ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{
			mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(1)), "level": 1, "noSupports": true, "triggered": true}),
			mod("CurseEffect", "INC", c.n(2), Tag{"type": "SkillName", "skillName": titleWords(c.s(1))}),
		}
	}),
	`curse enemies with ([^0-9]+) on [a-zA-Z]+, with ([0-9]+)% reduced effect`: fn(func(c caps) any {
		return []any{
			mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(1)), "level": 1, "noSupports": true, "triggered": true}),
			mod("CurseEffect", "INC", -c.n(2), Tag{"type": "SkillName", "skillName": titleWords(c.s(1))}),
		}
	}),
	`[0-9]+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with ([^0-9]+) on [a-zA-Z]+, with ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{
			mod("ExtraSkill", "LIST", Tag{"skillId": gemIdOrNil(c.s(1)), "level": 1, "noSupports": true, "triggered": true}),
			mod("CurseEffect", "INC", c.n(2), Tag{"type": "SkillName", "skillName": titleWords(c.s(1))}),
		}
	}),
	`you are cursed with ([^0-9]+), with ([0-9]+)% increased effect`: fn(func(c caps) any {
		return []any{
			mod("ExtraCurse", "LIST", Tag{"skillId": gemIdOrNil(c.s(1)), "level": 1, "applyToPlayer": true}),
			mod("CurseEffectOnSelf", "INC", c.n(2), Tag{"type": "SkillName", "skillName": titleWords(c.s(1))}),
		}
	}),
	`grants level ([0-9]+) (.+) curse aura during f?l?a?s?k? ?effect`: fn(func(c caps) any {
		skillId, ok := gemIdLookup[replaceAll(c.s(2), " skill", "")]
		if !ok {
			skillId = "Unknown"
		}
		return []any{mod("ExtraCurse", "LIST", Tag{"skillId": skillId, "level": c.n(1)}, Tag{"type": "Condition", "var": "UsingFlask"})}
	}),

	// --- ModParser.lua:3375: linked supports ---
	`socketed support gems can also support skills from y?o?u?r? ?e?q?u?i?p?p?e?d? ?([a-zA-Z \t\n\v\f\r]+)`: fn(func(c caps) any {
		targetItemSlotName := "Body Armour"
		if c.s(1) == "main hand" {
			targetItemSlotName = "Weapon 1"
		}
		return []any{mod("LinkedSupport", "LIST", Tag{"targetSlotName": targetItemSlotName})}
	}),

	// --- Self-cast curse family — ModParser.lua:3480,4152,4235-4238,4891-4894,4951-4954,5028,5850 ---
	`gain ([0-9]+)% of physical damage as a random element if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("PhysicalDamageGainAsRandom", "BASE", c.n(1), Tag{"type": "Condition", "var": selfCastVar(c.s(2))})}
	}),
	`inflict withered for ([0-9]+) seconds on hit if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{flag("Condition:CanWither", Tag{"type": "Condition", "var": selfCastVar(c.s(2))})}
	}),
	`([a-zA-Z]+) exposure on hit if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(1))+"Exposure", "BASE", -10)}, Tag{"type": "Condition", "var": selfCastVar(c.s(2))}, Tag{"type": "Condition", "var": "Effective"})}
	}),
	`inflict ([a-zA-Z]+) exposure on hit if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": mod(firstToUpper(c.s(1))+"Exposure", "BASE", -10)}, Tag{"type": "Condition", "var": selfCastVar(c.s(2))}, Tag{"type": "Condition", "var": "Effective"})}
	}),
	`you are unaffected by (.+) if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("Self"+firstToUpper(c.s(1))+"Effect", "MORE", -100, Tag{"type": "Condition", "var": selfCastVar(c.s(2))}, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true})}
	}),
	`immun[ei]t?y? to ([a-zA-Z]*?)s? if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{flag(firstToUpper(c.s(1))+"Immune", Tag{"type": "Condition", "var": selfCastVar(c.s(2))})}
	}),
	`action speed cannot be slowed below base value if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "Condition", "var": selfCastVar(c.s(1))})}
	}),
	`action speed cannot be modified to below base value if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("MinimumActionSpeed", "MAX", 100, Tag{"type": "Condition", "var": selfCastVar(c.s(1))})}
	}),
	`intimidate enemies on hit if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, nil, ModFlag.Hit, Tag{"type": "Condition", "var": selfCastVar(c.s(1))})}
	}),
	`take no extra damage from critical strikes if you've cast (.*?) in the past ([0-9]+) seconds`: fn(func(c caps) any {
		return []any{mod("ReduceCritExtraDamage", "BASE", 100, Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true}, Tag{"type": "Condition", "var": selfCastVar(c.s(1))})}
	}),

	// --- ModParser.lua (misc branches) ---
	`treats enemy monster elemental resistance values as inverted`: []any{
		mod("HitsInvertEleResChance", "CHANCE", 100, Tag{"type": "Condition", "var": "{Hand}Attack"}),
	},
	`base ([a-zA-Z]+) duration is ([0-9.]+) seconds?`: fn(func(c caps) any {
		ailment := c.s(1)
		var ailmentName string
		switch ailment {
		case "bleeding", "bleed":
			ailmentName = "Bleed"
		case "ignite", "poison":
			ailmentName = firstToUpper(ailment)
		default:
			return nil
		}
		return []any{mod(ailmentName+"DurationBase", "OVERRIDE", c.n(2))}
	}),
	`minions convert ([0-9]+)% of (.+) damage to (.+) damage per socketed ([a-zA-Z]+) gem`: fn(func(c caps) any {
		colour := c.s(4)
		if colour != "red" && colour != "green" && colour != "blue" {
			return nil
		}
		return []any{mod("MinionModifier", "LIST",
			Tag{"mod": mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), "BASE", c.n(1))},
			Tag{"type": "Multiplier", "var": "Socketed" + firstToUpper(colour) + "GemsIn{SlotName}"})}
	}),
	`([0-9]+)% reduced bonuses gained from equipped (.+)`: fn(func(c caps) any {
		return []any{mod("EffectOfBonusesFrom"+firstToUpper(c.s(2)), "INC", -c.n(1))}
	}),
	`right ring slot: you cannot regenerate mana`: []any{flag("NoManaRegen", Tag{"type": "SlotNumber", "num": 2})},
	`you cannot regenerate energy shield`:         []any{flag("NoEnergyShieldRegen")},
	// MultiplierThreshold is on RageStacks because Rage is only set in
	// CalcPerform if Condition:CanGainRage is true; Bear's Girdle does not flag
	// CanGainRage.
	`nearby enemies are intimidated while you have rage`: []any{
		mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "MultiplierThreshold", "var": "RageStack", "threshold": 1}),
	},
	`nearby enemies are crushed while you have ?a?t? least ([0-9]+) rage`: fn(func(c caps) any {
		return []any{mod("EnemyModifier", "LIST", Tag{"mod": flag("Condition:Crushed")}, Tag{"type": "MultiplierThreshold", "var": "RageStack", "threshold": c.n(1)})}
	}),
	`regenerate ([0-9]+) rage per second for every ([0-9]+) life recovery per second from regeneration`: fn(func(c caps) any {
		return []any{
			mod("RageRegen", "BASE", c.n(1), Tag{"type": "PercentStat", "stat": "LifeRegen", "percent": c.n(1) / c.n(2) * 100}),
			flag("Condition:CanGainRage"),
		}
	}),

	// --- Timeless jewel conquerors — ModParser.lua:5776-5799 ---
	`bathed in the blood of ([0-9]+) sacrificed in the name of (.+)`:         fn(conqueredBy),
	`carved to glorify ([0-9]+) new faithful converted by high templar (.+)`: fn(conqueredBy),
	`commanded leadership over ([0-9]+) warriors under (.+)`:                 fn(conqueredBy),
	`commissioned ([0-9]+) coins to commemorate (.+)`:                        fn(conqueredBy),
	`denoted service of ([0-9]+) dekhara in the akhara of (.+)`:              fn(conqueredBy),
	`remembrancing ([0-9]+) songworthy deeds by the line of (.+)`:            fn(conqueredBy),
	`subjugating ([0-9]+) souls in the thrall of (.+)`:                       fn(conqueredBy),
	`binding ([0-9]+) souls to phylacteries to sustain (.+)`:                 fn(conqueredBy),
}
