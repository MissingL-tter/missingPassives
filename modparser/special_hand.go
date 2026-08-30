package modparser

import (
	"math"
	"strconv"
	"strings"
)

// specialModList entries whose Lua closures need real statements — branches,
// loops, helper calls with option tables. Each is a direct transcription of
// the cited ModParser.lua lines. They are merged over the transformed entries
// when the scan table is built; keys are identical to the reference's.

// conquerorList — ModParser.lua:41.
var conquerorList = map[string]Conqueror{
	`xibaqua`:  {Kind: ConquerorVaal, Index: 1, V2: false},
	`zerphi`:   {Kind: ConquerorVaal, Index: 2, V2: false},
	`doryani`:  {Kind: ConquerorVaal, Index: 3, V2: false},
	`ahuana`:   {Kind: ConquerorVaal, Index: 2, V2: true},
	`deshret`:  {Kind: ConquerorMaraketh, Index: 1, V2: false},
	`asenath`:  {Kind: ConquerorMaraketh, Index: 2, V2: false},
	`nasima`:   {Kind: ConquerorMaraketh, Index: 3, V2: false},
	`balbala`:  {Kind: ConquerorMaraketh, Index: 1, V2: true},
	`cadiro`:   {Kind: ConquerorEternal, Index: 1, V2: false},
	`victario`: {Kind: ConquerorEternal, Index: 2, V2: false},
	`chitus`:   {Kind: ConquerorEternal, Index: 3, V2: false},
	`caspiro`:  {Kind: ConquerorEternal, Index: 3, V2: true},
	`kaom`:     {Kind: ConquerorKarui, Index: 1, V2: false},
	`rakiata`:  {Kind: ConquerorKarui, Index: 2, V2: false},
	`kiloava`:  {Kind: ConquerorKarui, Index: 3, V2: false},
	`akoya`:    {Kind: ConquerorKarui, Index: 3, V2: true},
	`venarius`: {Kind: ConquerorTemplar, Index: 1, V2: false},
	`dominus`:  {Kind: ConquerorTemplar, Index: 2, V2: false},
	`avarius`:  {Kind: ConquerorTemplar, Index: 3, V2: false},
	`maxarius`: {Kind: ConquerorTemplar, Index: 1, V2: true},
	`vorana`:   {Kind: ConquerorKalguur, Index: 1, V2: false},
	`uhtred`:   {Kind: ConquerorKalguur, Index: 2, V2: false},
	`medved`:   {Kind: ConquerorKalguur, Index: 3, V2: false},
	`tecrod`:   {Kind: ConquerorAbyssMurderous, Index: 1, V2: false},
	`ulaman`:   {Kind: ConquerorAbyssSearching, Index: 1, V2: false},
	`kurgal`:   {Kind: ConquerorAbyssHypnotic, Index: 1, V2: false},
	`amanamu`:  {Kind: ConquerorAbyssGhastly, Index: 1, V2: false},
	`zorath`:   {Kind: ConquerorAbyssSpecial, Index: 1, V2: false},
}

// conqueredBy — the shared body of the eight timeless jewel entries at
// ModParser.lua:5776-5799.
func conqueredBy(c caps) []*Mod {
	value := ConqueredBy{Seed: c.n(1)}
	if conq, ok := conquerorList[strings.ToLower(c.s(2))]; ok {
		value.Conqueror = &conq
	}
	return []*Mod{mod("JewelData", List, DataRef{Key: "conqueredBy", Value: value})}
}

// selfCastVar builds the "SelfCast<Curse>" condition var used by the
// "if you've cast X in the past N seconds" family.
func selfCastVar(curse string) string {
	return "SelfCast" + condenseName(curse)
}

var specialModListHand = map[string]modsValue{
	// --- Explode mods — ModParser.lua:2121-2186 ---
	`enemies you kill have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Obliteration, Unspeakable Gifts, synth implicit, crusader body, Ngamahu Warmonger tattoo
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies you kill have ?a? ?([0-9]+)% chance to explode, dealing ([0-9]+)% of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Hinekora, Death's Fury 3.22
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies you or your totems kill have ([0-9]+)% chance to explode, dealing ([0-9]+)% of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Hinekora, Death's Fury 3.23
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies you kill while using pride have ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Sublime Vision
		return explodeFunc(c.n(1), c.s(2), c.s(3), &CondTag{Var: "AffectedByPride"})
	}),
	`enemies you kill during effect have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as damage of a random element`: modFn(func(c caps) []*Mod { // Oriath's End
		return explodeFunc(c.n(1), c.s(2), "randomElement", &CondTag{Var: "UsingFlask"})
	}),
	`enemies you kill while affected by glorious madness have a ([0-9]+)% chance to explode, dealing a (.+) of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Beacon of Madness
		return explodeFunc(c.n(1), c.s(2), c.s(3), &CondTag{Var: "AffectedByGloriousMadness"})
	}),
	`enemies killed with attack hits have a ([0-9]+)% chance to explode, dealing a (.+) of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Devastator
		return explodeFunc(c.n(1), c.s(2), c.s(3))
	}),
	`enemies killed with wand hits have a ([0-9]+)% chance to explode, dealing a (.+) of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Explosive Force
		return explodeFunc(c.n(1), c.s(2), c.s(3), &CondTag{Var: "UsingWand"})
	}),
	`cursed enemies you or your minions kill have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Profane Bloom
		return explodeFunc(c.n(1), c.s(2), c.s(3), &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})
	}),
	`enemies you kill explode, dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // legacy synth, legacy crusader
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`enemies killed explode dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Quecholli
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`enemies on fungal ground you kill explode, dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Sporeguard
		return explodeFunc(100, c.s(1), c.s(2), &CondTag{IsActor: true, Actor: "enemy", Var: "OnFungalGround"})
	}),
	`enemies killed with attack or spell hits explode, dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Shaper 2H mace mod
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`shocked enemies you kill explode, dealing ([0-9]+)% of their life as (.+) damage which cannot shock`: modFn(func(c caps) []*Mod { // Inpulsa's Broken Heart
		return explodeFunc(100, c.s(1), c.s(2), &CondTag{IsActor: true, Actor: "enemy", Var: "Shocked"})
	}),
	`ignited enemies you kill explode, dealing ([0-9]+)% of their life as (.+) damage which cannot ignite`: modFn(func(c caps) []*Mod { // Inpulsa's Broken Heart
		return explodeFunc(100, c.s(1), c.s(2), &CondTag{IsActor: true, Actor: "enemy", Var: "Ignited"})
	}),
	`bleeding enemies you kill explode, dealing ([0-9]+)% of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Haemophilia
		return explodeFunc(100, c.s(1), c.s(2), &CondTag{IsActor: true, Actor: "enemy", Var: "Bleeding"})
	}),
	`burning enemies you kill have a ([0-9]+)% chance to explode, dealing a (.+) of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Haemophilia
		return explodeFunc(100, c.s(1), c.s(2), &CondTag{IsActor: true, Actor: "enemy", Var: "Burning"})
	}),
	`enemies killed near corpses affected by your curses explode, dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Asenath's Gentle Touch
		return explodeFunc(100, c.s(1), c.s(2), &MultiplierTag{IsThreshold: true, Var: "NearbyCorpse", Threshold: opt(1)}, &CondTag{IsActor: true, Actor: "enemy", Var: "Cursed"})
	}),
	`enemies taunted by your warcries explode on death, dealing ([0-9]+)% of their maximum life as (.+) damage`: modFn(func(c caps) []*Mod { // Al Dhih
		return explodeFunc(100, c.s(1), c.s(2), &CondTag{IsActor: true, Actor: "enemy", Var: "Taunted"}, &CondTag{Var: "UsedWarcryRecently"})
	}),
	`totems explode on death, dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Crucible weapon mod
		return explodeFunc(100, c.s(1), c.s(2))
	}),
	`nearby corpses explode when you warcry, dealing ([0-9]+)% of their life as (.+) damage`: modFn(func(c caps) []*Mod { // Ruthless Berserker node
		return explodeFunc(100, c.s(1), c.s(2))
	}),

	// --- Elemental Equilibrium — ModParser.lua:2415-2430 ---
	`enemies you hit with elemental damage temporarily get (\+[0-9]+)% resistance to those elements and (-[0-9]+)% resistance to other elements`: modFn(func(c caps) []*Mod {
		plus, minus := c.n(1), c.n(2)
		return []*Mod{flag("ElementalEquilibrium"), flag("ElementalEquilibriumLegacy"), mod("EnemyModifier", List, ModRef{Mod: mod("FireResist", Base, Num(plus), &CondTag{Var: "HitByFireDamage"})}), mod("EnemyModifier", List, ModRef{Mod: mod("FireResist", Base, Num(minus), &CondTag{Var: "HitByFireDamage", Neg: true}, &CondTag{VarList: []string{"HitByColdDamage", "HitByLightningDamage"}})}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdResist", Base, Num(plus), &CondTag{Var: "HitByColdDamage"})}), mod("EnemyModifier", List, ModRef{Mod: mod("ColdResist", Base, Num(minus), &CondTag{Var: "HitByColdDamage", Neg: true}, &CondTag{VarList: []string{"HitByFireDamage", "HitByLightningDamage"}})}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningResist", Base, Num(plus), &CondTag{Var: "HitByLightningDamage"})}), mod("EnemyModifier", List, ModRef{Mod: mod("LightningResist", Base, Num(minus), &CondTag{Var: "HitByLightningDamage", Neg: true}, &CondTag{VarList: []string{"HitByFireDamage", "HitByColdDamage"}})})}
	}),

	// --- ModParser.lua:2370: gain no X from equipped Y ---
	`gain no (.+) from equipped (.+)`: modFn(func(c caps) []*Mod {
		stat, slot := c.s(1), c.s(2)
		if slot == "shield" {
			slot = "Weapon 2"
		}
		return []*Mod{flag("GainNo" + stripSpaces(titleWords(stat)) + "From" + titleWords(slot))}
	}),

	// --- ModParser.lua:2500 (approx): all damage bypasses energy shield ---
	// The Chaos entry allows overriding "chaos damage does not bypass energy shield".
	`all damage taken bypasses energy shield`: modList{mod("PhysicalEnergyShieldBypass", Override, Num(100)), mod("LightningEnergyShieldBypass", Override, Num(100)), mod("ColdEnergyShieldBypass", Override, Num(100)), mod("FireEnergyShieldBypass", Override, Num(100)), mod("ChaosEnergyShieldBypass", Override, Num(100))},

	// --- Armour Mastery — ModParser.lua:2445-2475 ---
	`([0-9]+)% chance to defend with double your armour for each time you've been hit by an enemy recently, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		numChance, cap := c.n(1), c.n(2)
		return []*Mod{mods("ArmourDefense", Max, Num(100), "Armour Mastery: Max Calc", &CondTag{Var: "ArmourMax"}), mods("ArmourDefense", Max, Num(math.Min(numChance/100, 1.0)*100), "Armour Mastery: Average Calc", &CondTag{Var: "ArmourAvg"}, &MultiplierTag{Var: "BeenHitRecently", Limit: opt(cap / numChance)}), mods("ArmourDefense", Max, Num(math.Min(math.Floor(numChance/100), 1.0)*100), "Armour Mastery: Min Calc", &CondTag{Var: "ArmourMax", Neg: true}, &CondTag{Var: "ArmourAvg", Neg: true}, &MultiplierTag{Var: "BeenHitRecently", Limit: opt(cap / numChance)})}
	}),
	`([0-9]+)% chance to defend with ([0-9]+)% of armour for each time you've been hit by an enemy recently, up to ([0-9]+)%`: modFn(func(c caps) []*Mod {
		numChance, numArmourMultiplier, cap := c.n(1), c.n(2), c.n(3)
		return []*Mod{mods("ArmourDefense", Max, Num(numArmourMultiplier-100), "Armour Mastery: Max Calc", &CondTag{Var: "ArmourMax"}), mods("ArmourDefense", Max, Num(math.Min(numChance/100, 1.0)*(numArmourMultiplier-100)), "Armour Mastery: Average Calc", &CondTag{Var: "ArmourAvg"}, &MultiplierTag{Var: "BeenHitRecently", Limit: opt(cap / numChance)}), mods("ArmourDefense", Max, Num(math.Min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100)), "Armour Mastery: Min Calc", &CondTag{Var: "ArmourMax", Neg: true}, &CondTag{Var: "ArmourAvg", Neg: true}, &MultiplierTag{Var: "BeenHitRecently", Limit: opt(cap / numChance)})}
	}),
	`([0-9]+)% chance to defend with double armour`: modFn(func(c caps) []*Mod {
		numChance := c.n(1)
		return []*Mod{mods("ArmourDefense", Max, Num(100), "Armour Mastery: Max Calc", &CondTag{Var: "ArmourMax"}), mods("ArmourDefense", Max, Num(math.Min(numChance/100, 1.0)*100), "Armour Mastery: Average Calc", &CondTag{Var: "ArmourAvg"}), mods("ArmourDefense", Max, Num(math.Min(math.Floor(numChance/100), 1.0)*100), "Armour Mastery: Min Calc", &CondTag{Var: "ArmourMax", Neg: true}, &CondTag{Var: "ArmourAvg", Neg: true})}
	}),
	`([0-9]+)% chance to defend with ([0-9]+)% of armour`: modFn(func(c caps) []*Mod {
		numChance, numArmourMultiplier := c.n(1), c.n(2)
		return []*Mod{mods("ArmourDefense", Max, Num(numArmourMultiplier-100), "Armour Mastery: Max Calc", &CondTag{Var: "ArmourMax"}), mods("ArmourDefense", Max, Num(math.Min(numChance/100, 1.0)*(numArmourMultiplier-100)), "Armour Mastery: Average Calc", &CondTag{Var: "ArmourAvg"}), mods("ArmourDefense", Max, Num(math.Min(math.Floor(numChance/100), 1.0)*(numArmourMultiplier-100)), "Armour Mastery: Min Calc", &CondTag{Var: "ArmourMax", Neg: true}, &CondTag{Var: "ArmourAvg", Neg: true})}
	}),

	// --- Radius jewel transforms with parametric radius — ModParser.lua:2611,2619 ---
	`non-unique jewels cause increases and reductions to other damage types in a ([a-zA-Z]+) radius to be transformed to apply to ([a-zA-Z]+) damage`: modFn(func(c caps) []*Mod {
		dmgType := c.s(2)
		inner := getSimpleConv([]string{"PhysicalDamage", "FireDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"}, firstToUpper(dmgType)+"Damage", Inc, true, 0, 0)
		wrapped := jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			// Ignore Timeless Jewels
			if node != nil && !node.ConqueredBy() {
				inner(node, out, data)
			}
		})
		return []*Mod{mod("ExtraJewelFunc", List, JewelFn{Type: "Other", Func: wrapped, Radius: firstToUpper(c.s(1)), ID: JewelFnID("nonUniqueTransformDamage", dmgType)}, &ItemCondTag{ItemSlot: "{SlotName}", RarityCond: "UNIQUE", Neg: true}, &ItemCondTag{ItemSlot: "{SlotName}", RarityCond: "RELIC", Neg: true})}
	}),
	`non-unique jewels cause small and notable passive skills in a ([a-zA-Z]+) radius to also grant \+([0-9]+) to ([a-zA-Z]+)`: modFn(func(c caps) []*Mod {
		val, attr := c.n(2), c.s(3)
		attrShort := firstToUpper(attr)
		if len(attrShort) > 3 {
			attrShort = attrShort[:3] // firstToUpper(attr):match("^%a%l%l")
		}
		wrapped := jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if node != nil && !node.ConqueredBy() && (node.Type() == "Notable" || node.Type() == "Normal") {
				out.AddMod(mods(attrShort, Base, Num(val), data.ModSource))
			}
		})
		return []*Mod{mod("ExtraJewelFunc", List, JewelFn{Type: "Other", Func: wrapped, Radius: firstToUpper(c.s(1)), ID: JewelFnID("nonUniqueGrantAttribute", attrShort, strconv.FormatFloat(val, 'g', -1, 64))}, &ItemCondTag{ItemSlot: "{SlotName}", RarityCond: "UNIQUE", Neg: true}, &ItemCondTag{ItemSlot: "{SlotName}", RarityCond: "RELIC", Neg: true})}
	}),

	// --- ModParser.lua:2890: arcane surge ailment effect ---
	`non-damaging ailments have ([0-9]+)% reduced effect on you while you have arcane surge`: modFn(func(c caps) []*Mod {
		var mods []*Mod
		for _, ailment := range nonDamagingAilmentTypeList {
			mods = append(mods, mod("Self"+ailment+"Effect", Inc, Num(-c.n(1)), &CondTag{Var: "AffectedByArcaneSurge"}))
		}
		return mods
	}),

	// --- ModParser.lua:2931,4992: capped multipliers dividing two captures ---
	`([0-9]+)% increased attack and cast speed for each corpse consumed recently, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Speed", Inc, Num(c.n(1)), &MultiplierTag{Var: "CorpseConsumedRecently", Limit: opt(c.n(2) / c.n(1))})}
	}),
	`([0-9]+)% increased armour per second you've been stationary, up to a maximum of ([0-9]+)%`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Armour", Inc, Num(c.n(1)), &MultiplierTag{Var: "StationarySeconds", Limit: opt(c.n(2) / c.n(1))}, &CondTag{Var: "Stationary"})}
	}),

	// --- ModParser.lua:3097: display-only ---
	`can be allflame crafted as if rare`: modList{}, // Display only. For Dread Captain's Cutlass and supported by crafting UI

	// --- Gem property entries with branches — ModParser.lua:3220-3235 ---
	`([+\-][0-9]+)%? to ([a-zA-Z]+) of socketed ?([a-zA-Z\- ]*) gems`: modFn(func(c caps) []*Mod {
		typ := c.s(3)
		if typ == "" {
			typ = "all"
		}
		return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: typ, Key: c.s(2), Value: opt(c.n(1))}, &SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}"})}
	}),
	`([+\-][0-9]+)%? to ([a-zA-Z]+) of all (.+) gems`: modFn(func(c caps) []*Mod {
		skill := c.s(3)
		if _, ok := gemIdLookup[skill]; ok {
			return []*Mod{mod("GemProperty", List, GemPropertyRef{Keyword: skill, Key: "level", Value: opt(c.n(1)), KeyOfScaledMod: "value"})}
		}
		var wordList []string
		for _, word := range splitWords(skill) {
			wordList = append(wordList, word)
		}
		return []*Mod{mod("GemProperty", List, GemPropertyRef{KeywordList: wordList, Key: c.s(2), Value: opt(c.n(1)), KeyOfScaledMod: "value"})}
	}),

	// --- ModParser.lua:3295: phantasm special case ---
	`trigger level ([0-9]+) (.+) when you consume a corpse`: modFn(func(c caps) []*Mod {
		if c.s(2) == "summon phantasm skill" {
			return triggerExtraSkill("triggered summon phantasm skill", c.n(1))
		}
		return triggerExtraSkill(c.s(2), c.n(1))
	}),

	// --- Item-granted curses — ModParser.lua:3340-3355,4083 ---
	`curse enemies with ([^0-9]+) on [a-zA-Z]+, with ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(1)), Level: opt(1), NoSupports: true, Triggered: true}), mod("CurseEffect", Inc, Num(c.n(2)), &SkillNameTag{SkillName: titleWords(c.s(1))})}
	}),
	`curse enemies with ([^0-9]+) on [a-zA-Z]+, with ([0-9]+)% reduced effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(1)), Level: opt(1), NoSupports: true, Triggered: true}), mod("CurseEffect", Inc, Num(-c.n(2)), &SkillNameTag{SkillName: titleWords(c.s(1))})}
	}),
	`[0-9]+% chance to curse n?o?n?-?c?u?r?s?e?d? ?enemies with ([^0-9]+) on [a-zA-Z]+, with ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraSkill", List, SkillRef{SkillID: gemIdOrNil(c.s(1)), Level: opt(1), NoSupports: true, Triggered: true}), mod("CurseEffect", Inc, Num(c.n(2)), &SkillNameTag{SkillName: titleWords(c.s(1))})}
	}),
	`you are cursed with ([^0-9]+), with ([0-9]+)% increased effect`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ExtraCurse", List, SkillRef{SkillID: gemIdOrNil(c.s(1)), Level: opt(1), ApplyToPlayer: true}), mod("CurseEffectOnSelf", Inc, Num(c.n(2)), &SkillNameTag{SkillName: titleWords(c.s(1))})}
	}),
	`grants level ([0-9]+) (.+) curse aura during f?l?a?s?k? ?effect`: modFn(func(c caps) []*Mod {
		skillId, ok := gemIdLookup[replaceAll(c.s(2), " skill", "")]
		if !ok {
			skillId = "Unknown"
		}
		return []*Mod{mod("ExtraCurse", List, SkillRef{SkillID: skillId, Level: opt(c.n(1))}, &CondTag{Var: "UsingFlask"})}
	}),

	// --- ModParser.lua:3375: linked supports ---
	`socketed support gems can also support skills from y?o?u?r? ?e?q?u?i?p?p?e?d? ?([a-zA-Z \t\n\v\f\r]+)`: modFn(func(c caps) []*Mod {
		targetItemSlotName := "Body Armour"
		if c.s(1) == "main hand" {
			targetItemSlotName = "Weapon 1"
		}
		return []*Mod{mod("LinkedSupport", List, LinkedSupportRef{TargetSlotName: targetItemSlotName})}
	}),

	// --- Self-cast curse family — ModParser.lua:3480,4152,4235-4238,4891-4894,4951-4954,5028,5850 ---
	`gain ([0-9]+)% of physical damage as a random element if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("PhysicalDamageGainAsRandom", Base, Num(c.n(1)), &CondTag{Var: selfCastVar(c.s(2))})}
	}),
	`inflict withered for ([0-9]+) seconds on hit if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{flag("Condition:CanWither", &CondTag{Var: selfCastVar(c.s(2))})}
	}),
	`([a-zA-Z]+) exposure on hit if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(1))+"Exposure", Base, Num(-10))}, &CondTag{Var: selfCastVar(c.s(2))}, &CondTag{Var: "Effective"})}
	}),
	`inflict ([a-zA-Z]+) exposure on hit if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: mod(firstToUpper(c.s(1))+"Exposure", Base, Num(-10))}, &CondTag{Var: selfCastVar(c.s(2))}, &CondTag{Var: "Effective"})}
	}),
	`you are unaffected by (.+) if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("Self"+firstToUpper(c.s(1))+"Effect", More, Num(-100), &CondTag{Var: selfCastVar(c.s(2))}, &GlobalEffectTag{EffectType: "Global", Unscalable: true})}
	}),
	`immun[ei]t?y? to ([a-zA-Z]*?)s? if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{flag(firstToUpper(c.s(1))+"Immune", &CondTag{Var: selfCastVar(c.s(2))})}
	}),
	`action speed cannot be slowed below base value if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinimumActionSpeed", Max, Num(100), &CondTag{Var: selfCastVar(c.s(1))})}
	}),
	`action speed cannot be modified to below base value if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("MinimumActionSpeed", Max, Num(100), &CondTag{Var: selfCastVar(c.s(1))})}
	}),
	`intimidate enemies on hit if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{modf("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, FlagHit, KeywordNone, &CondTag{Var: selfCastVar(c.s(1))})}
	}),
	`take no extra damage from critical strikes if you've cast (.*?) in the past ([0-9]+) seconds`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("ReduceCritExtraDamage", Base, Num(100), &GlobalEffectTag{EffectType: "Global", Unscalable: true}, &CondTag{Var: selfCastVar(c.s(1))})}
	}),

	// --- ModParser.lua (misc branches) ---
	`treats enemy monster elemental resistance values as inverted`: modList{mod("HitsInvertEleResChance", Chance, Num(100), &CondTag{Var: "{Hand}Attack"})},
	`base ([a-zA-Z]+) duration is ([0-9.]+) seconds?`: modFn(func(c caps) []*Mod {
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
		return []*Mod{mod(ailmentName+"DurationBase", Override, Num(c.n(2)))}
	}),
	`minions convert ([0-9]+)% of (.+) damage to (.+) damage per socketed ([a-zA-Z]+) gem`: modFn(func(c caps) []*Mod {
		colour := c.s(4)
		if colour != "red" && colour != "green" && colour != "blue" {
			return nil
		}
		return []*Mod{mod("MinionModifier", List, ModRef{Mod: mod(firstToUpper(c.s(2))+"DamageConvertTo"+firstToUpper(c.s(3)), Base, Num(c.n(1)))}, &MultiplierTag{Var: "Socketed" + firstToUpper(colour) + "GemsIn{SlotName}"})}
	}),
	`([0-9]+)% reduced bonuses gained from equipped (.+)`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EffectOfBonusesFrom"+firstToUpper(c.s(2)), Inc, Num(-c.n(1)))}
	}),
	`right ring slot: you cannot regenerate mana`: modList{flag("NoManaRegen", &SlotTag{SlotKind: TagSlotNumber, Num: 2})},
	`you cannot regenerate energy shield`:         modList{flag("NoEnergyShieldRegen")},
	// MultiplierThreshold is on RageStacks because Rage is only set in
	// CalcPerform if Condition:CanGainRage is true; Bear's Girdle does not flag
	// CanGainRage.
	`nearby enemies are intimidated while you have rage`: modList{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &MultiplierTag{IsThreshold: true, Var: "RageStack", Threshold: opt(1)})},
	`nearby enemies are crushed while you have ?a?t? least ([0-9]+) rage`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Crushed")}, &MultiplierTag{IsThreshold: true, Var: "RageStack", Threshold: opt(c.n(1))})}
	}),
	`regenerate ([0-9]+) rage per second for every ([0-9]+) life recovery per second from regeneration`: modFn(func(c caps) []*Mod {
		return []*Mod{mod("RageRegen", Base, Num(c.n(1)), &StatTag{StatKind: TagPercentStat, Stat: "LifeRegen", Percent: opt(c.n(1) / c.n(2) * 100)}), flag("Condition:CanGainRage")}
	}),

	// --- Timeless jewel conquerors — ModParser.lua:5776-5799 ---
	`bathed in the blood of ([0-9]+) sacrificed in the name of (.+)`:         modFn(conqueredBy),
	`carved to glorify ([0-9]+) new faithful converted by high templar (.+)`: modFn(conqueredBy),
	`commanded leadership over ([0-9]+) warriors under (.+)`:                 modFn(conqueredBy),
	`commissioned ([0-9]+) coins to commemorate (.+)`:                        modFn(conqueredBy),
	`denoted service of ([0-9]+) dekhara in the akhara of (.+)`:              modFn(conqueredBy),
	`remembrancing ([0-9]+) songworthy deeds by the line of (.+)`:            modFn(conqueredBy),
	`subjugating ([0-9]+) souls in the thrall of (.+)`:                       modFn(conqueredBy),
	`binding ([0-9]+) souls to phylacteries to sustain (.+)`:                 modFn(conqueredBy),
}
