// CalcDefence.lua L1385-1632: recoup, ward recharge, damage reduction,
// movement, avoidance/immunities and self-ailment duration and effect.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

var dmgTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

var recoupTypeList = []string{"Life", "Mana", "EnergyShield"}

func (env *Env) defenceRecoup(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	// recoup
	output["anyRecoup"] = 0.0
	addRecoup := func(v float64) { output["anyRecoup"] = outNum(output, "anyRecoup") + v }
	for _, recoupType := range recoupTypeList {
		baseRecoup := modDB.Sum("BASE", nil, recoupType+"Recoup")
		if recoupType == "Life" && modDB.Flag(nil, "EnergyShieldRecoupInsteadOfLife") {
			output["LifeRecoup"] = 0.0
			lifeRecoup := modDB.Sum("BASE", nil, "LifeRecoup")
			modDB.AddMod(newMod("EnergyShieldRecoup", "BASE", lifeRecoup, "Life Recoup Conversion"))
		} else {
			output[recoupType+"Recoup"] = baseRecoup * outNum(output, recoupType+"RecoveryRateMod")
			addRecoup(outNum(output, recoupType+"Recoup"))
		}
		addToEnergyShieldFlag := "Add" + recoupType + "RecoupToEnergyShieldRecoup"
		if modDB.Flag(nil, addToEnergyShieldFlag) {
			flagMod := modDB.Tabulate("FLAG", nil, addToEnergyShieldFlag)[0].Mod
			modDB.ReplaceMod(newMod("EnergyShieldRecoup", "BASE", baseRecoup, flagMod.Source))
		}
	}

	if modDB.Flag(nil, "UsePowerCharges") && modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		perAbsorption := modDB.Sum("BASE", nil, "PerAbsorptionElementalEnergyShieldRecoup")
		for _, elem := range []string{"Cold", "Fire", "Lightning"} {
			modDB.AddMod(newMod(elem+"EnergyShieldRecoup", "BASE", perAbsorption, "Absorption Charges",
				modparser.Tag{"type": "Multiplier", "var": "AbsorptionCharge"}))
		}
	}

	for _, recoupType := range recoupTypeList {
		for _, damageType := range dmgTypeList {
			recoup := modDB.Sum("BASE", nil, damageType+recoupType+"Recoup")
			if recoupType == "Life" && modDB.Flag(nil, "EnergyShieldRecoupInsteadOfLife") {
				output[damageType+"LifeRecoup"] = 0.0
				modDB.AddMod(newMod(damageType+"EnergyShieldRecoup", "BASE", recoup, "Life Recoup Conversion"))
			} else {
				output[damageType+recoupType+"Recoup"] = recoup * outNum(output, recoupType+"RecoveryRateMod")
				addRecoup(outNum(output, damageType+recoupType+"Recoup"))
			}
			addToEnergyShieldFlag := "Add" + recoupType + "RecoupToEnergyShieldRecoup"
			if modDB.Flag(nil, addToEnergyShieldFlag) {
				flagMod := modDB.Tabulate("FLAG", nil, addToEnergyShieldFlag)[0].Mod
				modDB.ReplaceMod(newMod(damageType+"EnergyShieldRecoup", "BASE", recoup, flagMod.Source))
			}
		}
	}
	// pseudo recoup (eg %physical damage prevented from hits regenerated)
	for _, resource := range recoupTypeList {
		if !modDB.Flag(nil, "No"+resource+"Regen") && !modDB.Flag(nil, "CannotGain"+resource) {
			pseudo := modDB.Sum("BASE", nil, "PhysicalDamageMitigated"+resource+"PseudoRecoup")
			if pseudo > 0 {
				dur := modDB.Sum("BASE", nil, "PhysicalDamageMitigated"+resource+"PseudoRecoupDuration")
				if dur == 0 {
					dur = 4
				}
				output["PhysicalDamageMitigated"+resource+"PseudoRecoupDuration"] = dur
				inc := modDB.Sum("INC", nil, resource+"Regen")
				more := modDB.More(nil, resource+"Regen")
				output["PhysicalDamageMitigated"+resource+"PseudoRecoup"] = pseudo * (1 + inc/100) * more * outNum(output, resource+"RecoveryRateMod")
				addRecoup(outNum(output, "PhysicalDamageMitigated"+resource+"PseudoRecoup"))
			}
		}
	}

	// Ward recharge
	output["WardRechargeDelay"] = data.Misc.WardRechargeDelay / (1 + modDB.Sum("INC", nil, "WardRechargeFaster")/100)

	// Damage Reduction
	if ov := modDB.Override(nil, "DamageReductionMax"); truthy(ov) {
		output["DamageReductionMax"] = anyNum(ov)
	} else {
		output["DamageReductionMax"] = data.Misc.DamageReductionCap
	}
	modDB.AddMod(newMod("ArmourAppliesToPhysicalDamageTaken", "BASE", 100.0))
	for _, damageType := range dmgTypeList {
		ov := modDB.Override(nil, damageType+"DamageReduction")
		if truthy(ov) {
			output["Base"+damageType+"DamageReduction"] = anyNum(ov)
			output["Base"+damageType+"DamageReductionWhenHit"] = anyNum(ov)
			continue
		}
		names := elemNames(damageType, damageType+"DamageReduction", "ElementalDamageReduction")
		base := math.Min(math.Max(0, modDB.Sum("BASE", nil, names...)), outNum(output, "DamageReductionMax"))
		output["Base"+damageType+"DamageReduction"] = base
		output["Base"+damageType+"DamageReductionWhenHit"] = math.Min(math.Max(0, base+modDB.Sum("BASE", nil, damageType+"DamageReductionWhenHit")), outNum(output, "DamageReductionMax"))
	}

	// Miscellaneous: move speed, avoidance
	if ov := modDB.Override(nil, "MovementSpeed"); truthy(ov) {
		output["MovementSpeedMod"] = anyNum(ov)
	} else if modDB.Flag(nil, "MovementSpeedEqualHighestLinkedPlayers") {
		panic("defence: MovementSpeedEqualHighestLinkedPlayers needs the party tab")
	} else {
		output["MovementSpeedMod"] = Mod(modDB, nil, "MovementSpeed")
	}
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		output["MovementSpeedMod"] = math.Max(outNum(output, "MovementSpeedMod"), 1)
	}
	output["EffectiveMovementSpeedMod"] = outNum(output, "MovementSpeedMod") * outNum(output, "ActionSpeedMod")

	if enemyDB.Flag(nil, "Blind") {
		output["BlindEffectMod"] = Mod(enemyDB, nil, "BlindEffect", "BuffEffectOnSelf") * 100
	}

	// recovery on block, needs to be after primary defences
	output["LifeOnBlock"] = 0.0
	output["LifeOnSuppress"] = 0.0
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		output["LifeOnBlock"] = modDB.Sum("BASE", nil, "LifeOnBlock")
		output["LifeOnSuppress"] = modDB.Sum("BASE", nil, "LifeOnSuppress")
	}
	output["ManaOnBlock"] = modDB.Sum("BASE", nil, "ManaOnBlock")
	output["EnergyShieldOnBlock"] = modDB.Sum("BASE", nil, "EnergyShieldOnBlock")
	output["EnergyShieldOnSpellBlock"] = modDB.Sum("BASE", nil, "EnergyShieldOnSpellBlock")
	output["EnergyShieldOnSuppress"] = modDB.Sum("BASE", nil, "EnergyShieldOnSuppress")

	// damage avoidances
	output["specificTypeAvoidance"] = false
	for _, damageType := range dmgTypeList {
		output["Avoid"+damageType+"DamageChance"] = math.Min(modDB.Sum("BASE", nil, "Avoid"+damageType+"DamageChance"), data.Misc.AvoidChanceCap)
		if outNum(output, "Avoid"+damageType+"DamageChance") > 0 {
			output["specificTypeAvoidance"] = true
		}
	}
	output["AvoidProjectilesChance"] = math.Min(modDB.Sum("BASE", nil, "AvoidProjectilesChance"), data.Misc.AvoidChanceCap)
	output["AvoidAllDamageFromHitsChance"] = math.Min(modDB.Sum("BASE", nil, "AvoidAllDamageFromHitsChance"), data.Misc.AvoidChanceCap)
	if modDB.Flag(nil, "BlindImmune") {
		output["BlindAvoidChance"] = 100.0
	} else {
		output["BlindAvoidChance"] = math.Min(modDB.Sum("BASE", nil, "AvoidBlind"), 100)
	}
	if modDB.Flag(nil, "ImpaleImmune") {
		output["ImpaleAvoidChance"] = 100.0
	} else {
		output["ImpaleAvoidChance"] = math.Min(modDB.Sum("BASE", nil, "AvoidImpale"), 100)
	}
	// Status effect / ailment immunities. Lua Flag() yields nil when unset,
	// so a false immunity stores no key at all.
	for _, imm := range []struct{ out, flag string }{
		{"CorruptedBloodImmunity", "CorruptedBloodImmune"},
		{"MaimImmunity", "MaimImmune"},
		{"HinderImmunity", "HinderImmune"},
		{"KnockbackImmunity", "KnockbackImmune"},
	} {
		if modDB.Flag(nil, imm.flag) {
			output[imm.out] = true
		}
	}

	spellSuppressionChance := modDB.Sum("BASE", nil, "SpellSuppressionChance")
	if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
		// Ancestral Vision
		pct := modDB.Sum("BASE", nil, "SpellSuppressionAppliesToAilmentAvoidancePercent") / 100
		modDB.AddMod(newMod("AvoidElementalAilments", "BASE", math.Floor(pct*spellSuppressionChance), "Ancestral Vision"))
	}
	if modDB.Flag(nil, "SpellSuppressionAppliesToChanceToDefendWithArmour") {
		// Foulborn Ancestral Vision
		pct, _ := modDB.Max(nil, "SpellSuppressionAppliesToChanceToDefendWithArmourPercent")
		armourPct, _ := modDB.Max(nil, "SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour")
		modDB.AddMod(newMod("ArmourDefense", "MAX", armourPct-100,
			"Chance to Defend from Spell Suppression: Max Calc", modparser.Tag{"type": "Condition", "var": "ArmourMax"}))
		modDB.AddMod(newMod("ArmourDefense", "MAX", math.Min(pct*spellSuppressionChance/100, 1.0)*(armourPct-100),
			"Chance to Defend from Spell Suppression: Average Calc", modparser.Tag{"type": "Condition", "var": "ArmourAvg"}))
		modDB.AddMod(newMod("ArmourDefense", "MAX", math.Min(math.Floor(pct*spellSuppressionChance/100), 1.0)*(armourPct-100),
			// #EVAL the reference reads a nil global `modSource` here, so the
			// fallback string is always the source
			"Chance to Defend from Spell Suppression: Min Calc",
			modparser.Tag{"type": "Condition", "var": "ArmourMax", "neg": true},
			modparser.Tag{"type": "Condition", "var": "ArmourAvg", "neg": true}))
	}
	armourDefense, _ := modDB.Max(nil, "ArmourDefense")
	output["ArmourDefense"] = armourDefense / 100
	if outNum(output, "ArmourDefense") > 0 {
		output["RawArmourDefense"] = (1 + outNum(output, "ArmourDefense")) * 100
	}

	if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
		if base := modDB.Sum("BASE", nil, "AvoidShock"); base != 0 {
			modDB.AddMod(newMod("AvoidShockAppliesToElementalAilments", "BASE", base, "Stormshroud"))
		}
	}

	for _, ailment := range data.NonElementalAilmentTypeList {
		if modDB.Flag(nil, ailment+"Immune") {
			output[ailment+"AvoidChance"] = 100.0
		} else {
			output[ailment+"AvoidChance"] = math.Floor(math.Min(modDB.Sum("BASE", nil, "Avoid"+ailment, "AvoidAilments"), 100))
		}
	}
	for _, ailment := range data.ElementalAilmentTypeList {
		shockAvoidAppliesToAll := modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") && ailment != "Shock"
		if modDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") {
			output[ailment+"AvoidChance"] = 100.0
		} else {
			extra := 0.0
			if shockAvoidAppliesToAll {
				extra = modDB.Sum("BASE", nil, "AvoidShock")
			}
			output[ailment+"AvoidChance"] = math.Floor(math.Min(modDB.Sum("BASE", nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments")+extra, 100))
		}
	}

	if modDB.Flag(nil, "CurseImmune") {
		output["CurseAvoidChance"] = 100.0
	} else {
		output["CurseAvoidChance"] = math.Min(modDB.Sum("BASE", nil, "AvoidCurse"), 100)
	}
	if modDB.Flag(nil, "SilenceImmune") {
		output["SilenceAvoidChance"] = 100.0
	} else {
		output["SilenceAvoidChance"] = outNum(output, "CurseAvoidChance")
	}
	output["CritExtraDamageReduction"] = math.Min(modDB.Sum("BASE", nil, "ReduceCritExtraDamage"), 100)
	output["LightRadiusMod"] = Mod(modDB, nil, "LightRadius")
	output["LightRadiusInc"] = math.Max(modDB.Sum("INC", nil, "LightRadius"), 0)
	output["CurseEffectOnSelf"] = math.Max(modDB.More(nil, "CurseEffectOnSelf")*(100+modDB.Sum("INC", nil, "CurseEffectOnSelf")), 0)
	output["ExposureEffectOnSelf"] = modDB.More(nil, "ExposureEffectOnSelf") * (100 + modDB.Sum("INC", nil, "ExposureEffectOnSelf"))
	output["WitherEffectOnSelf"] = modDB.More(nil, "WitherEffectOnSelf") * (100 + modDB.Sum("INC", nil, "WitherEffectOnSelf"))

	// Ailment duration on self
	output["DebuffExpirationRate"] = modDB.Sum("BASE", nil, "SelfDebuffExpirationRate")
	output["DebuffExpirationModifier"] = 10000 / (100 + outNum(output, "DebuffExpirationRate"))
	output["showDebuffExpirationModifier"] = outNum(output, "DebuffExpirationModifier") != 100
	output["SelfBlindDuration"] = modDB.More(nil, "SelfBlindDuration") * (100 + modDB.Sum("INC", nil, "SelfBlindDuration")) * outNum(output, "DebuffExpirationModifier") / 100

	if modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") {
		inc := modDB.Sum("INC", nil, "SelfIgniteDuration")
		more := modDB.More(nil, "SelfIgniteDuration")
		if inc != 0 {
			modDB.AddMod(newMod("SelfIgniteDurationToElementalAilments", "INC", inc, "Firesong"))
		}
		if more != 1 {
			modDB.AddMod(newMod("SelfIgniteDurationToElementalAilments", "MORE", more, "Firesong"))
		}
	}

	for _, ailment := range data.NonElementalAilmentTypeList {
		more := modDB.More(nil, "Self"+ailment+"Duration", "SelfAilmentDuration")
		inc := (100 + modDB.Sum("INC", nil, "Self"+ailment+"Duration", "SelfAilmentDuration")) * 100
		output["Self"+ailment+"Duration"] = inc * more / (100 + outNum(output, "DebuffExpirationRate") + modDB.Sum("BASE", nil, "Self"+ailment+"DebuffExpirationRate"))
	}
	for _, ailment := range data.ElementalAilmentTypeList {
		igniteAppliesToAll := modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") && ailment != "Ignite"
		more := modDB.More(nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
		incExtra := 0.0
		if igniteAppliesToAll {
			more *= modDB.More(nil, "SelfIgniteDuration")
			incExtra = modDB.Sum("INC", nil, "SelfIgniteDuration")
		}
		inc := (100 + modDB.Sum("INC", nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "SelfElementalAilmentDuration") + incExtra) * 100
		output["Self"+ailment+"Duration"] = more * inc / (100 + outNum(output, "DebuffExpirationRate") + modDB.Sum("BASE", nil, "Self"+ailment+"DebuffExpirationRate"))
	}
	for _, ailment := range data.AilmentTypeList {
		selfEffect := Mod(modDB, nil, "Self"+ailment+"Effect")
		var enemyEffect float64
		if modDB.Flag(nil, "Condition:"+ailment+"edSelf") {
			enemyEffect = Mod(modDB, nil, "Enemy"+ailment+"Effect")
		} else {
			enemyEffect = Mod(enemyDB, nil, "Enemy"+ailment+"Effect")
		}
		output["Self"+ailment+"Effect"] = selfEffect * enemyEffect * 100
	}
}
