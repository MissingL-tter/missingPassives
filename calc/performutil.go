// CalcPerform.lua L30-612: the per-actor helper functions of the perform
// stage (mergeBuff, life/mana, attributes/conditions, reservations, curse
// priority, enemy modifier application). Breakdown output (CALCS-mode UI)
// is skipped throughout.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// performActor bundles the per-actor state perform manipulates (the Lua
// actor tables: env.player / env.minion / env.enemy).
type performActor struct {
	ms     *modstore.Actor
	db     *modstore.DB
	output map[string]any

	mainSkill *ActiveSkill
	skills    []*ActiveSkill
	minion    *Minion // set when this actor is the main skill's minion

	enemy  *performActor
	parent *performActor

	reservedLifeBase, reservedLifePercent float64
	reservedManaBase, reservedManaPercent float64
	uncancellableLife, uncancellableMana  float64
	strDmgBonus                           float64
	appliedEnemyModifiers                 map[*modparser.Mod]bool

	// offence-stage weapon reach (actor.weaponRange1/2, set in the
	// skill-type-stats section)
	weaponRange1, weaponRange2 float64

	// EHP-stage shift tables (actor.damageShiftTable in the reference)
	damageShiftTable         map[string]map[string]float64
	damageOverTimeShiftTable map[string]map[string]float64
}

// roundDec is Common.lua's round(val, dec).
func roundDec(v float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(v*p+0.5) / p
}

// floorDec is Common.lua's floor(val, dec) — note the 0.0001 it adds before
// flooring, which is there to stop float error from dropping a value a whole
// unit at the requested precision (a More() product of 1.331 can land at
// 1.3309999999999997, which would floor to 1.3309 without it).
func floorDec(v float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(v*p+0.0001) / p
}

func outNum(m map[string]any, k string) float64 {
	return anyNum(m[k])
}

// mergeBuff ports CalcPerform's mergeBuff: merge an instance of a buff,
// taking the highest value of each modifier.
func mergeBuff(src []*modparser.Mod, destTable map[string]*modstore.List, destKey string) {
	if destTable[destKey] == nil {
		destTable[destKey] = modstore.NewList(nil)
	}
	dest := destTable[destKey]
	for _, mod := range src {
		match := false
		if mod.Type != "LIST" {
			for index, destMod := range dest.Mods {
				if modparser.CompareModParams(mod, destMod) {
					if dv, ok := destMod.Value.(float64); ok && anyNum(mod.Value) > dv {
						dest.Mods[index] = mod
					}
					match = true
					break
				}
			}
		}
		if !match {
			dest.Mods = append(dest.Mods, mod)
		}
	}
}

// doActorLifeMana ports CalcPerform's doActorLifeMana.
func (env *Env) doActorLifeMana(actor *performActor) {
	modDB := actor.db
	output := actor.output
	condList := modDB.Conditions

	lowLifePerc := modDB.Sum("BASE", nil, "LowLifePercentage")
	if lowLifePerc > 0 {
		output["LowLifePercentage"] = 100.0 * lowLifePerc
	} else {
		output["LowLifePercentage"] = 100.0 * data.Misc.LowPoolThreshold
	}
	fullLifePerc := modDB.Sum("BASE", nil, "FullLifePercentage")
	if fullLifePerc > 0 {
		output["FullLifePercentage"] = 100.0 * fullLifePerc
	} else {
		output["FullLifePercentage"] = 100.0
	}

	ci := modDB.Flag(nil, "ChaosInoculation")
	// Lua Flag() yields nil when unset, so the key only exists when true
	if ci {
		output["ChaosInoculation"] = true
	}
	// Life/mana pools
	if ci {
		output["Life"] = 1.0
		condList["FullLife"] = true
	} else {
		base := modDB.Sum("BASE", nil, "Life")
		inc := modDB.Sum("INC", nil, "Life")
		more := modDB.More(nil, "Life")
		override := modDB.Override(nil, "Life")
		conv := modDB.Sum("BASE", nil, "LifeConvertToEnergyShield")
		if truthy(override) {
			output["Life"] = anyNum(override)
		} else {
			output["Life"] = math.Max(roundDec(base*(1+inc/100)*more*(1-conv/100), 0), 1)
		}
	}
	manaConv := modDB.Sum("BASE", nil, "ManaConvertToArmour")
	output["Mana"] = roundDec(Val(modDB, "Mana", nil)*(1-manaConv/100), 0)
	output["LowestOfMaximumLifeAndMaximumMana"] = math.Min(outNum(output, "Life"), outNum(output, "Mana"))
}

// matchAfterSpace is baseName:match("%s(%S+)"): the word after the first
// whitespace.
func matchAfterSpace(s string) string {
	idx := strings.IndexAny(s, " \t")
	if idx < 0 {
		return ""
	}
	rest := s[idx+1:]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		rest = rest[1:]
	}
	end := strings.IndexAny(rest, " \t")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// doActorAttribsConditions ports CalcPerform's doActorAttribsConditions.
func (env *Env) doActorAttribsConditions(actor *performActor) {
	modDB := actor.db
	output := actor.output
	condList := modDB.Conditions
	itemList := actor.ms.ItemList
	weaponData1 := actor.ms.WeaponData1
	weaponData2 := actor.ms.WeaponData2

	// Set conditions
	if (itemList["Weapon 2"] != nil && itemList["Weapon 2"].ItemType() == "Shield") ||
		(actor == env.playerPA && env.AegisModList != nil) {
		condList["UsingShield"] = true
	} else if itemList["Weapon 2"] == nil {
		condList["OffHandIsEmpty"] = true
	}
	applyWeaponConds := func(weaponData map[string]any) {
		info := data.WeaponTypeInfo[str(weaponData["type"])]
		condList["Using"+info.Flag] = true
		if truthy(weaponData["countsAsAll1H"]) {
			weaponData["AddedUsingAxe"] = !truthy(condList["UsingAxe"])
			condList["UsingAxe"] = true
			// Varunastra is a sword
			weaponData["AddedUsingSword"] = strings.Contains(str(weaponData["name"]), "Varunastra") || !truthy(condList["UsingSword"])
			condList["UsingSword"] = true
			weaponData["AddedUsingDagger"] = !truthy(condList["UsingDagger"])
			condList["UsingDagger"] = true
			weaponData["AddedUsingMace"] = !truthy(condList["UsingMace"])
			condList["UsingMace"] = true
			weaponData["AddedUsingClaw"] = !truthy(condList["UsingClaw"])
			condList["UsingClaw"] = true
			condList["WieldingDifferentWeaponTypes"] = true
		}
		if info.Melee {
			condList["UsingMeleeWeapon"] = true
		}
		if info.OneHand {
			condList["UsingOneHandedWeapon"] = true
		} else {
			condList["UsingTwoHandedWeapon"] = true
		}
	}
	if str(weaponData1["type"]) == "None" {
		condList["Unarmed"] = true
		if itemList["Weapon 2"] == nil && itemList["Gloves"] == nil {
			condList["Unencumbered"] = true
		}
	} else {
		applyWeaponConds(weaponData1)
	}
	for _, slotName := range []string{"Helmet", "Body Armour", "Gloves", "Boots"} {
		if itemList[slotName] != nil {
			condList["Using"+slotName] = true
		}
	}
	for _, graftSlot := range []string{"Graft 1", "Graft 2"} {
		if it, _ := itemList[graftSlot].(*Item); it != nil && it.In.BaseName != nil {
			condList["Using"+matchAfterSpace(*it.In.BaseName)] = true
		}
	}
	if _, hasType := weaponData2["type"]; hasType {
		applyWeaponConds(weaponData2)
	}
	if truthy(weaponData1["type"]) && truthy(weaponData2["type"]) {
		condList["DualWielding"] = true
		if (str(weaponData1["type"]) == "Claw" || truthy(weaponData1["countsAsAll1H"])) &&
			(str(weaponData2["type"]) == "Claw" || truthy(weaponData2["countsAsAll1H"])) {
			condList["DualWieldingClaws"] = true
		}
		if (str(weaponData1["type"]) == "Dagger" || truthy(weaponData1["countsAsAll1H"])) &&
			(str(weaponData2["type"]) == "Dagger" || truthy(weaponData2["countsAsAll1H"])) {
			condList["DualWieldingDaggers"] = true
		}
		getWeaponType := func(weaponData map[string]any) string {
			// GGG treats thrusting 1H swords as 1H swords
			if str(weaponData["type"]) == "One Handed Sword" && str(weaponData["subType"]) == "Thrusting" {
				return "One Handed Sword"
			}
			info := data.WeaponTypeInfo[str(weaponData["type"])]
			if info.Label != nil {
				return *info.Label
			}
			if s := str(weaponData["subType"]); s != "" {
				return s
			}
			return str(weaponData["type"])
		}
		if getWeaponType(weaponData1) != getWeaponType(weaponData2) {
			info1 := data.WeaponTypeInfo[str(weaponData1["type"])]
			info2 := data.WeaponTypeInfo[str(weaponData2["type"])]
			if info1.OneHand && info2.OneHand {
				condList["WieldingDifferentWeaponTypes"] = true
			}
		}
	}
	mainSkill := actor.mainSkill
	if env.ModeCombat {
		if !truthy(mainSkill.SkillData["triggered"]) && !mainSkill.SkillFlags["trap"] && !mainSkill.SkillFlags["mine"] && !mainSkill.SkillFlags["totem"] {
			if mainSkill.SkillFlags["attack"] {
				condList["AttackedRecently"] = true
			} else if mainSkill.SkillFlags["spell"] {
				condList["CastSpellRecently"] = true
			}
			if mainSkill.SkillTypes[modparser.SkillType.Movement] {
				condList["UsedMovementSkillRecently"] = true
			}
			if mainSkill.SkillFlags["minion"] && !mainSkill.SkillFlags["permanentMinion"] {
				condList["UsedMinionSkillRecently"] = true
			}
			if mainSkill.SkillTypes[modparser.SkillType.Vaal] {
				condList["UsedVaalSkillRecently"] = true
			}
			if mainSkill.SkillTypes[modparser.SkillType.Channel] {
				condList["Channelling"] = true
			}
		}
		if mainSkill.SkillFlags["hit"] && !mainSkill.SkillFlags["trap"] && !mainSkill.SkillFlags["mine"] && !mainSkill.SkillFlags["totem"] {
			condList["HitRecently"] = true
			if mainSkill.SkillFlags["spell"] {
				condList["HitSpellRecently"] = true
			}
		}
		if mainSkill.SkillFlags["totem"] {
			condList["HaveTotem"] = true
			condList["SummonedTotemRecently"] = true
			if mainSkill.SkillFlags["hit"] {
				condList["TotemsHitRecently"] = true
				if mainSkill.SkillFlags["spell"] {
					condList["TotemsSpellHitRecently"] = true
				}
			}
		}
		if mainSkill.SkillFlags["mine"] {
			condList["DetonatedMinesRecently"] = true
		}
		if mainSkill.SkillFlags["trap"] {
			condList["TriggeredTrapsRecently"] = true
		}
		playerMainCfg := env.PlayerMainSkill.SkillCfg
		if modDB.Sum("BASE", nil, "EnemyScorchChance") > 0 ||
			modDB.Flag(nil, "CritAlwaysAltAilments") && !modDB.Flag(playerMainCfg, "NeverCrit") ||
			modDB.Flag(nil, "IgniteCanScorch") {
			condList["CanInflictScorch"] = true
		}
		if modDB.Sum("BASE", nil, "EnemyBrittleChance") > 0 ||
			modDB.Flag(nil, "CritAlwaysAltAilments") && !modDB.Flag(playerMainCfg, "NeverCrit") {
			condList["CanInflictBrittle"] = true
		}
		if modDB.Sum("BASE", nil, "EnemySapChance") > 0 ||
			modDB.Flag(nil, "CritAlwaysAltAilments") && !modDB.Flag(playerMainCfg, "NeverCrit") {
			condList["CanInflictSap"] = true
		}
		// Shrine buffs: before life pool for massive shrine
		shrineEffectMod := 1 + modDB.Sum("INC", nil, "BuffEffectOnSelf", "ShrineBuffEffect")/100
		flr := func(v float64) float64 { return math.Floor(v) }
		if modDB.Flag(nil, "AccelerationShrine") {
			modDB.AddMod(newMod("ActionSpeed", "INC", flr(50*shrineEffectMod), "Acceleration Shrine"))
			modDB.AddMod(newMod("ProjectileSpeed", "INC", flr(80*shrineEffectMod), "Acceleration Shrine"))
		}
		if modDB.Flag(nil, "BrutalShrine") {
			modDB.AddMod(newMod("Damage", "INC", flr(50*shrineEffectMod), "Brutal Shrine"))
			modDB.AddMod(newMod("EnemyStunDuration", "INC", flr(30*shrineEffectMod), "Brutal Shrine"))
			modDB.AddMod(newMod("EnemyKnockbackChance", "INC", 100.0, "Brutal Shrine"))
		}
		if modDB.Flag(nil, "DiamondShrine") {
			modDB.AddMod(newMod("CritChance", "OVERRIDE", 100.0, "Diamond Shrine"))
		}
		if modDB.Flag(nil, "DivineShrine") {
			modDB.AddMod(newMod("DamageTaken", "MORE", -100.0, "Divine Shrine"))
		}
		if modDB.Flag(nil, "EchoingShrine") {
			modDB.AddMod(newMod("Speed", "MORE", flr(100*shrineEffectMod), "Echoing Shrine", modparser.ModFlag.Attack))
			modDB.AddMod(newMod("Speed", "MORE", flr(100*shrineEffectMod), "Echoing Shrine", modparser.ModFlag.Cast))
			modDB.AddMod(newMod("RepeatCount", "BASE", flr(1*shrineEffectMod), "Echoing Shrine"))
		}
		if modDB.Flag(nil, "GloomShrine") {
			modDB.AddMod(newMod("NonChaosDamageGainAsChaos", "BASE", flr(10*shrineEffectMod), "Gloom Shrine"))
		}
		if modDB.Flag(nil, "GreaterFreezingShrine") {
			modDB.AddMod(newMod("PhysicalDamageGainAsCold", "BASE", flr(30*shrineEffectMod), "Greater Freezing Shrine"))
		}
		if modDB.Flag(nil, "GreaterShockingShrine") {
			modDB.AddMod(newMod("PhysicalDamageGainAsLightning", "BASE", flr(30*shrineEffectMod), "Greater Shocking Shrine"))
		}
		if modDB.Flag(nil, "GreaterSkeletalShrine") {
			modDB.AddMod(newMod("PhysicalDamageGainAsChaos", "BASE", flr(30*shrineEffectMod), "Greater Skeletal Shrine"))
		}
		if modDB.Flag(nil, "ImpenetrableShrine") {
			modDB.AddMod(newMod("Armour", "INC", flr(100*shrineEffectMod), "Impenetrable Shrine"))
			modDB.AddMod(newMod("Evasion", "INC", flr(100*shrineEffectMod), "Impenetrable Shrine"))
			modDB.AddMod(newMod("EnergyShield", "INC", flr(100*shrineEffectMod), "Impenetrable Shrine"))
		}
		if modDB.Flag(nil, "MassiveShrine") {
			modDB.AddMod(newMod("Life", "INC", flr(40*shrineEffectMod), "Massive Shrine"))
			modDB.AddMod(newMod("AreaOfEffect", "INC", flr(40*shrineEffectMod), "Massive Shrine"))
		}
		if modDB.Flag(nil, "ReplenishingShrine") {
			modDB.AddMod(newMod("ManaRegen", "INC", 200*shrineEffectMod, "Replenishing Shrine"))
			modDB.AddMod(newMod("LifeRegenPercent", "BASE", 6.7*shrineEffectMod, "Replenishing Shrine"))
		}
		if modDB.Flag(nil, "ResistanceShrine") {
			modDB.AddMod(newMod("ElementalResist", "BASE", flr(50*shrineEffectMod), "Resistance Shrine"))
			modDB.AddMod(newMod("ElementalResistMax", "BASE", flr(10*shrineEffectMod), "Resistance Shrine"))
			modDB.AddMod(newMod("ChaosResistMax", "BASE", flr(10*shrineEffectMod), "Resistance Shrine"))
		}
		if modDB.Flag(nil, "ResonatingShrine") {
			modDB.AddMod(newMod("PowerChargesMax", "BASE", flr(1*shrineEffectMod), "Resonating Shrine"))
			modDB.AddMod(newMod("FrenzyChargesMax", "BASE", flr(1*shrineEffectMod), "Resonating Shrine"))
			modDB.AddMod(newMod("EnduranceChargesMax", "BASE", flr(1*shrineEffectMod), "Resonating Shrine"))
		}
		if modDB.Flag(nil, "LesserAccelerationShrine") && !modDB.Flag(nil, "AccelerationShrine") {
			modDB.AddMod(newMod("ActionSpeed", "INC", flr(10*shrineEffectMod), "Lesser Acceleration Shrine"))
			modDB.AddMod(newMod("ProjectileSpeed", "INC", flr(30*shrineEffectMod), "Lesser Acceleration Shrine"))
		}
		if modDB.Flag(nil, "LesserBrutalShrine") {
			modDB.AddMod(newMod("Damage", "INC", flr(20*shrineEffectMod), "Lesser Brutal Shrine"))
			modDB.AddMod(newMod("EnemyStunDuration", "INC", flr(20*shrineEffectMod), "Lesser Brutal Shrine"))
			modDB.AddMod(newMod("EnemyKnockbackChance", "INC", 100.0, "Lesser Brutal Shrine"))
		}
		if modDB.Flag(nil, "LesserImpenetrableShrine") {
			modDB.AddMod(newMod("Armour", "INC", flr(50*shrineEffectMod), "Lesser Impenetrable Shrine"))
			modDB.AddMod(newMod("Evasion", "INC", flr(50*shrineEffectMod), "Lesser Impenetrable Shrine"))
			modDB.AddMod(newMod("EnergyShield", "INC", flr(50*shrineEffectMod), "Lesser Impenetrable Shrine"))
		}
		if modDB.Flag(nil, "LesserMassiveShrine") {
			modDB.AddMod(newMod("Life", "INC", flr(20*shrineEffectMod), "Lesser Massive Shrine"))
			modDB.AddMod(newMod("AreaOfEffect", "INC", flr(20*shrineEffectMod), "Lesser Massive Shrine"))
		}
		if modDB.Flag(nil, "LesserReplenishingShrine") {
			modDB.AddMod(newMod("ManaRegen", "INC", 100*shrineEffectMod, "Lesser Replenishing Shrine"))
			modDB.AddMod(newMod("LifeRegenPercent", "BASE", 3.3*shrineEffectMod, "Lesser Replenishing Shrine"))
		}
		if modDB.Flag(nil, "LesserResistanceShrine") {
			modDB.AddMod(newMod("ElementalResist", "BASE", flr(25*shrineEffectMod), "Lesser Resistance Shrine"))
			modDB.AddMod(newMod("ElementalResistMax", "BASE", flr(2*shrineEffectMod), "Lesser Resistance Shrine"))
			modDB.AddMod(newMod("ChaosResistMax", "BASE", flr(2*shrineEffectMod), "Lesser Resistance Shrine"))
		}
	}
	if env.ModeEffective {
		pm := env.PlayerMainSkill
		for _, el := range []string{"Fire", "Cold", "Lightning"} {
			if pm.SkillModList.Sum("BASE", pm.SkillCfg, el+"ExposureChance") > 0 || modDB.Sum("BASE", nil, el+"ExposureChance") > 0 {
				condList["CanApply"+el+"Exposure"] = true
			}
		}
	}

	setAttribConds := func() {
		stats := []float64{outNum(output, "Str"), outNum(output, "Dex"), outNum(output, "Int")}
		sort.Float64s(stats)
		output["LowestAttribute"] = stats[0]
		condList["TwoHighestAttributesEqual"] = stats[1] == stats[2]

		oStr, oDex, oInt := outNum(output, "Str"), outNum(output, "Dex"), outNum(output, "Int")
		condList["DexHigherThanInt"] = oDex > oInt
		condList["StrHigherThanInt"] = oStr > oInt
		condList["IntHigherThanDex"] = oInt > oDex
		condList["StrHigherThanDex"] = oStr > oDex
		condList["IntHigherThanStr"] = oInt > oStr
		condList["DexHigherThanStr"] = oDex > oStr

		condList["StrHighestAttribute"] = oStr >= oDex && oStr >= oInt
		condList["IntHighestAttribute"] = oInt >= oStr && oInt >= oDex
		condList["DexHighestAttribute"] = oDex >= oStr && oDex >= oInt
		condList["IntSingleHighestAttribute"] = oInt > oStr && oInt > oDex
		condList["DexSingleHighestAttribute"] = oDex > oStr && oDex > oInt
	}

	calculateAttributes := func() {
		for pass := 1; pass <= 2; pass++ {
			for _, stat := range []string{"Str", "Dex", "Int"} {
				output[stat] = math.Max(roundDec(Val(modDB, stat, nil), 0), 0)
			}
			setAttribConds()
		}
	}

	calculateOmniscience := func() {
		classStats := env.Build.ClassStats
		baseOf := map[string]float64{"Str": classStats.BaseStr, "Dex": classStats.BaseDex, "Int": classStats.BaseInt}
		for pass := 1; pass <= 2; pass++ {
			if pass != 1 {
				for _, stat := range []string{"Str", "Dex", "Int"} {
					base := baseOf[stat]
					output[stat] = math.Min(roundDec(Val(modDB, stat, nil), 0), base)
					modDB.AddMod(newMod("Omni", "BASE", modDB.Sum("BASE", nil, stat)-base, stat+" conversion Omniscience"))
					modDB.AddMod(newMod("Omni", "INC", modDB.Sum("INC", nil, stat), "Omniscience"))
					modDB.AddMod(newMod("Omni", "MORE", modDB.Sum("MORE", nil, stat), "Omniscience"))
				}
			}
			if pass != 2 {
				// Subtract out double and triple dips
				reduction := map[string]float64{}
				for _, typ := range []string{"BASE", "INC", "MORE"} {
					conv := map[string]float64{}
					for _, stat := range []string{"StrDex", "StrInt", "DexInt", "All"} {
						conv[stat] = modDB.Sum(typ, nil, stat)
					}
					reduction[typ] = conv["StrDex"] + conv["StrInt"] + conv["DexInt"] + 2*conv["All"]
				}
				modDB.AddMod(newMod("Omni", "BASE", -reduction["BASE"], "Reduction from Double/Triple Dipped attributes to Omniscience"))
				modDB.AddMod(newMod("Omni", "INC", -reduction["INC"], "Reduction from Double/Triple Dipped attributes to Omniscience"))
				modDB.AddMod(newMod("Omni", "MORE", -reduction["MORE"], "Reduction from Double/Triple Dipped attributes to Omniscience"))
			}
			for _, stat := range []string{"Str", "Dex", "Int"} {
				output[stat] = baseOf[stat]
			}
			output["Omni"] = math.Max(roundDec(Val(modDB, "Omni", nil), 0), 0)
			setAttribConds()
		}
	}

	if modDB.Flag(nil, "Omniscience") {
		calculateOmniscience()
	} else {
		calculateAttributes()
	}

	// Calculate total attributes
	output["TotalAttr"] = outNum(output, "Str") + outNum(output, "Dex") + outNum(output, "Int")

	// Special case for Devotion
	output["Devotion"] = modDB.Sum("BASE", nil, "Devotion")

	// Add attribute bonuses
	if !modDB.Flag(nil, "NoAttributeBonuses") {
		if !modDB.Flag(nil, "NoStrengthAttributeBonuses") {
			if !modDB.Flag(nil, "NoStrBonusToLife") {
				modDB.AddMod(newMod("Life", "BASE", math.Floor(outNum(output, "Str")/2), "Strength"))
			}
			strDmgBonusRatioOverride := modDB.Sum("BASE", nil, "StrDmgBonusRatioOverride")
			if strDmgBonusRatioOverride > 0 {
				actor.strDmgBonus = math.Floor((outNum(output, "Str") + modDB.Sum("BASE", nil, "DexIntToMeleeBonus")) * strDmgBonusRatioOverride)
			} else {
				actor.strDmgBonus = math.Floor((outNum(output, "Str") + modDB.Sum("BASE", nil, "DexIntToMeleeBonus")) / 5)
			}
			modDB.AddMod(newMod("PhysicalDamage", "INC", actor.strDmgBonus, "Strength", modparser.ModFlag.Melee))
		}
		if !modDB.Flag(nil, "NoDexterityAttributeBonuses") {
			accPerDex := data.Misc.AccuracyPerDexBase
			if ov := modDB.Override(nil, "DexAccBonusOverride"); truthy(ov) {
				accPerDex = anyNum(ov)
			}
			modDB.AddMod(newMod("Accuracy", "BASE", outNum(output, "Dex")*accPerDex, "Dexterity"))
			if !modDB.Flag(nil, "NoDexBonusToEvasion") {
				modDB.AddMod(newMod("Evasion", "INC", math.Floor(outNum(output, "Dex")/5), "Dexterity"))
			}
		}
		if !modDB.Flag(nil, "NoIntelligenceAttributeBonuses") {
			if !modDB.Flag(nil, "NoIntBonusToMana") {
				modDB.AddMod(newMod("Mana", "BASE", math.Floor(outNum(output, "Int")/2), "Intelligence"))
			}
			if !modDB.Flag(nil, "NoIntBonusToES") {
				modDB.AddMod(newMod("EnergyShield", "INC", math.Floor(outNum(output, "Int")/10), "Intelligence"))
			}
		}
	}

	env.doActorLifeMana(actor)
}

// doActorLifeManaReservation ports CalcPerform's doActorLifeManaReservation.
func (env *Env) doActorLifeManaReservation(actor *performActor, addAura bool) {
	modDB := actor.db
	output := actor.output
	condList := modDB.Conditions

	for _, pool := range []string{"Life", "Mana"} {
		max := outNum(output, pool)
		reserved := 0.0
		if max > 0 {
			lowPerc := modDB.Sum("BASE", nil, "Low"+pool+"Percentage")
			resBase, resPercent, uncancellable := actor.reservedLifeBase, actor.reservedLifePercent, actor.uncancellableLife
			if pool == "Mana" {
				resBase, resPercent, uncancellable = actor.reservedManaBase, actor.reservedManaPercent, actor.uncancellableMana
			}
			reserved = resBase + math.Ceil(max*resPercent/100)
			output[pool+"Reserved"] = math.Min(reserved, max)
			output[pool+"ReservedPercent"] = math.Min(reserved/max*100, 100)
			output[pool+"Unreserved"] = max - reserved
			output[pool+"UnreservedPercent"] = (max - reserved) / max * 100
			output[pool+"UncancellableReservation"] = math.Min(uncancellable, 0)
			output[pool+"CancellableReservation"] = 100 - uncancellable
			threshold := data.Misc.LowPoolThreshold
			if lowPerc > 0 {
				threshold = lowPerc
			}
			if (max-reserved)/max <= threshold {
				condList["Low"+pool] = true
			}
		}
		if addAura {
			for _, v := range modDB.List(nil, "GrantReserved"+pool+"AsAura") {
				tag, _ := v.(modparser.Tag)
				mod, _ := tag["mod"].(*modparser.Mod)
				if mod == nil {
					continue
				}
				auraMod := modparser.CopyMod(mod)
				auraMod.Value = math.Floor(anyNum(auraMod.Value) * math.Min(reserved, outNum(output, pool)))
				modDB.AddMod(newMod("ExtraAura", "LIST", modparser.Tag{"mod": auraMod}))
			}
		}
	}
}

// determineCursePriority ports CalcPerform's determineCursePriority.
func (env *Env) determineCursePriority(curseName string, activeSkill *ActiveSkill) float64 {
	source := ""
	slot := ""
	socket := 1
	if activeSkill != nil && activeSkill.SocketGroup != nil {
		source = str(activeSkill.SocketGroup.KV["source"])
		slot = str(activeSkill.SocketGroup.KV["slot"])
		for k, v := range activeSkill.SocketGroup.GemList {
			if v.GemData != nil && v.GemData.Name == curseName {
				// limit of 8 to avoid collision with CurseFromEquipment
				socket = k + 1
				if socket > 8 {
					socket = 8
				}
				break
			}
		}
	}
	basePriority := float64(data.CursePriority[curseName])
	socketPriority := float64(socket) * float64(data.CursePriority["SocketPriorityBase"])
	// slot:gsub(" (Swap)", "") — the parens are a Lua capture: strips " Swap"
	slotPriority := float64(data.CursePriority[strings.ReplaceAll(slot, " Swap", "")])
	sourcePriority := 0.0
	if activeSkill != nil && activeSkill.SkillTypes[modparser.SkillType.Aura] {
		sourcePriority = float64(data.CursePriority["CurseFromAura"])
	} else if source != "" {
		sourcePriority = float64(data.CursePriority["CurseFromEquipment"])
	}
	if source != "" && (slotPriority == float64(data.CursePriority["Ring 2"]) || slotPriority == float64(data.CursePriority["Ring 3"])) {
		slotPriority = float64(data.CursePriority["Ring 1"])
	}
	return basePriority + socketPriority + slotPriority + sourcePriority
}

// applyEnemyModifiers ports CalcPerform's applyEnemyModifiers.
func applyEnemyModifiers(actor *performActor, clearCache bool) {
	if clearCache || actor.appliedEnemyModifiers == nil {
		actor.appliedEnemyModifiers = map[*modparser.Mod]bool{}
	}
	cache := actor.appliedEnemyModifiers
	enemyDB := actor.enemy.db
	for _, value := range actor.db.TabulateAll(nil, "EnemyModifier") {
		tag, _ := value.Value.(modparser.Tag)
		mod, _ := tag["mod"].(*modparser.Mod)
		if mod != nil && !cache[mod] {
			source := mod.Source
			if !mod.SourceSet {
				source = value.Mod.Source
			}
			enemyDB.AddMod(modparser.SetSource(mod, source))
			cache[mod] = true
		}
	}
}
