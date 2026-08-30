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
	output.SetN("anyRecoup", 0.0)
	addRecoup := func(v float64) { output.SetN("anyRecoup", output.N("anyRecoup")+v) }
	for _, recoupType := range recoupTypeList {
		baseRecoup := modDB.Sum(modparser.Base, nil, recoupType+"Recoup")
		if recoupType == "Life" && modDB.Flag(nil, "EnergyShieldRecoupInsteadOfLife") {
			output.SetN("LifeRecoup", 0.0)
			lifeRecoup := modDB.Sum(modparser.Base, nil, "LifeRecoup")
			modDB.AddMod(newModS("EnergyShieldRecoup", modparser.Base, modparser.Num(lifeRecoup), "Life Recoup Conversion"))
		} else {
			output.SetN(recoupType+"Recoup", baseRecoup*output.N(recoupType+"RecoveryRateMod"))
			addRecoup(output.N(recoupType + "Recoup"))
		}
		addToEnergyShieldFlag := "Add" + recoupType + "RecoupToEnergyShieldRecoup"
		if modDB.Flag(nil, addToEnergyShieldFlag) {
			flagMod := modDB.Tabulate(modparser.Flag, nil, addToEnergyShieldFlag)[0].Mod
			modDB.ReplaceMod(newModS("EnergyShieldRecoup", modparser.Base, modparser.Num(baseRecoup), flagMod.Source))
		}
	}

	if modDB.Flag(nil, "UsePowerCharges") && modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		perAbsorption := modDB.Sum(modparser.Base, nil, "PerAbsorptionElementalEnergyShieldRecoup")
		for _, elem := range []string{"Cold", "Fire", "Lightning"} {
			modDB.AddMod(newModS(elem+"EnergyShieldRecoup", modparser.Base, modparser.Num(perAbsorption), "Absorption Charges", &modparser.MultiplierTag{Var: "AbsorptionCharge"}))
		}
	}

	for _, recoupType := range recoupTypeList {
		for _, damageType := range dmgTypeList {
			recoup := modDB.Sum(modparser.Base, nil, damageType+recoupType+"Recoup")
			if recoupType == "Life" && modDB.Flag(nil, "EnergyShieldRecoupInsteadOfLife") {
				output.SetN(damageType+"LifeRecoup", 0.0)
				modDB.AddMod(newModS(damageType+"EnergyShieldRecoup", modparser.Base, modparser.Num(recoup), "Life Recoup Conversion"))
			} else {
				output.SetN(damageType+recoupType+"Recoup", recoup*output.N(recoupType+"RecoveryRateMod"))
				addRecoup(output.N(damageType + recoupType + "Recoup"))
			}
			addToEnergyShieldFlag := "Add" + recoupType + "RecoupToEnergyShieldRecoup"
			if modDB.Flag(nil, addToEnergyShieldFlag) {
				flagMod := modDB.Tabulate(modparser.Flag, nil, addToEnergyShieldFlag)[0].Mod
				modDB.ReplaceMod(newModS(damageType+"EnergyShieldRecoup", modparser.Base, modparser.Num(recoup), flagMod.Source))
			}
		}
	}
	// pseudo recoup (eg %physical damage prevented from hits regenerated)
	for _, resource := range recoupTypeList {
		if !modDB.Flag(nil, "No"+resource+"Regen") && !modDB.Flag(nil, "CannotGain"+resource) {
			pseudo := modDB.Sum(modparser.Base, nil, "PhysicalDamageMitigated"+resource+"PseudoRecoup")
			if pseudo > 0 {
				dur := modDB.Sum(modparser.Base, nil, "PhysicalDamageMitigated"+resource+"PseudoRecoupDuration")
				if dur == 0 {
					dur = 4
				}
				output.SetN("PhysicalDamageMitigated"+resource+"PseudoRecoupDuration", dur)
				inc := modDB.Sum(modparser.Inc, nil, resource+"Regen")
				more := modDB.More(nil, resource+"Regen")
				output.SetN("PhysicalDamageMitigated"+resource+"PseudoRecoup", pseudo*(1+inc/100)*more*output.N(resource+"RecoveryRateMod"))
				addRecoup(output.N("PhysicalDamageMitigated" + resource + "PseudoRecoup"))
			}
		}
	}

	// Ward recharge
	output.SetN("WardRechargeDelay", data.Misc.WardRechargeDelay/(1+modDB.Sum(modparser.Inc, nil, "WardRechargeFaster")/100))

	// Damage Reduction
	if ov, ok := modDB.Override(nil, "DamageReductionMax"); ok {
		output.SetN("DamageReductionMax", valueNum(ov))
	} else {
		output.SetN("DamageReductionMax", data.Misc.DamageReductionCap)
	}
	modDB.AddMod(newMod("ArmourAppliesToPhysicalDamageTaken", modparser.Base, modparser.Num(100.0)))
	for _, damageType := range dmgTypeList {
		ov, _ := modDB.Override(nil, damageType+"DamageReduction")
		if modparser.Truthy(ov) {
			output.SetN("Base"+damageType+"DamageReduction", valueNum(ov))
			output.SetN("Base"+damageType+"DamageReductionWhenHit", valueNum(ov))
			continue
		}
		names := elemNames(damageType, damageType+"DamageReduction", "ElementalDamageReduction")
		base := math.Min(math.Max(0, modDB.Sum(modparser.Base, nil, names...)), output.N("DamageReductionMax"))
		output.SetN("Base"+damageType+"DamageReduction", base)
		output.SetN("Base"+damageType+"DamageReductionWhenHit", math.Min(math.Max(0, base+modDB.Sum(modparser.Base, nil, damageType+"DamageReductionWhenHit")), output.N("DamageReductionMax")))
	}

	// Miscellaneous: move speed, avoidance
	if ov, ok := modDB.Override(nil, "MovementSpeed"); ok {
		output.SetN("MovementSpeedMod", valueNum(ov))
	} else if modDB.Flag(nil, "MovementSpeedEqualHighestLinkedPlayers") {
		panic("defence: MovementSpeedEqualHighestLinkedPlayers needs the party tab")
	} else {
		output.SetN("MovementSpeedMod", Mod(modDB, nil, "MovementSpeed"))
	}
	if modDB.Flag(nil, "MovementSpeedCannotBeBelowBase") {
		output.SetN("MovementSpeedMod", math.Max(output.N("MovementSpeedMod"), 1))
	}
	output.SetN("EffectiveMovementSpeedMod", output.N("MovementSpeedMod")*output.N("ActionSpeedMod"))

	if enemyDB.Flag(nil, "Blind") {
		output.SetN("BlindEffectMod", Mod(enemyDB, nil, "BlindEffect", "BuffEffectOnSelf")*100)
	}

	// recovery on block, needs to be after primary defences
	output.SetN("LifeOnBlock", 0.0)
	output.SetN("LifeOnSuppress", 0.0)
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		output.SetN("LifeOnBlock", modDB.Sum(modparser.Base, nil, "LifeOnBlock"))
		output.SetN("LifeOnSuppress", modDB.Sum(modparser.Base, nil, "LifeOnSuppress"))
	}
	output.SetN("ManaOnBlock", modDB.Sum(modparser.Base, nil, "ManaOnBlock"))
	output.SetN("EnergyShieldOnBlock", modDB.Sum(modparser.Base, nil, "EnergyShieldOnBlock"))
	output.SetN("EnergyShieldOnSpellBlock", modDB.Sum(modparser.Base, nil, "EnergyShieldOnSpellBlock"))
	output.SetN("EnergyShieldOnSuppress", modDB.Sum(modparser.Base, nil, "EnergyShieldOnSuppress"))

	// damage avoidances
	output.SetFlag("specificTypeAvoidance", false)
	for _, damageType := range dmgTypeList {
		output.SetN("Avoid"+damageType+"DamageChance", math.Min(modDB.Sum(modparser.Base, nil, "Avoid"+damageType+"DamageChance"), data.Misc.AvoidChanceCap))
		if output.N("Avoid"+damageType+"DamageChance") > 0 {
			output.SetFlag("specificTypeAvoidance", true)
		}
	}
	output.SetN("AvoidProjectilesChance", math.Min(modDB.Sum(modparser.Base, nil, "AvoidProjectilesChance"), data.Misc.AvoidChanceCap))
	output.SetN("AvoidAllDamageFromHitsChance", math.Min(modDB.Sum(modparser.Base, nil, "AvoidAllDamageFromHitsChance"), data.Misc.AvoidChanceCap))
	if modDB.Flag(nil, "BlindImmune") {
		output.SetN("BlindAvoidChance", 100.0)
	} else {
		output.SetN("BlindAvoidChance", math.Min(modDB.Sum(modparser.Base, nil, "AvoidBlind"), 100))
	}
	if modDB.Flag(nil, "ImpaleImmune") {
		output.SetN("ImpaleAvoidChance", 100.0)
	} else {
		output.SetN("ImpaleAvoidChance", math.Min(modDB.Sum(modparser.Base, nil, "AvoidImpale"), 100))
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
			output.SetFlag(imm.out, true)
		}
	}

	spellSuppressionChance := modDB.Sum(modparser.Base, nil, "SpellSuppressionChance")
	if modDB.Flag(nil, "SpellSuppressionAppliesToAilmentAvoidance") {
		// Ancestral Vision
		pct := modDB.Sum(modparser.Base, nil, "SpellSuppressionAppliesToAilmentAvoidancePercent") / 100
		modDB.AddMod(newModS("AvoidElementalAilments", modparser.Base, modparser.Num(math.Floor(pct*spellSuppressionChance)), "Ancestral Vision"))
	}
	if modDB.Flag(nil, "SpellSuppressionAppliesToChanceToDefendWithArmour") {
		// Foulborn Ancestral Vision
		pct, _ := modDB.Max(nil, "SpellSuppressionAppliesToChanceToDefendWithArmourPercent")
		armourPct, _ := modDB.Max(nil, "SpellSuppressionAppliesToChanceToDefendWithArmourPercentArmour")
		modDB.AddMod(newModS("ArmourDefense", modparser.Max, modparser.Num(armourPct-100), "Chance to Defend from Spell Suppression: Max Calc", &modparser.CondTag{Var: "ArmourMax"}))
		modDB.AddMod(newModS("ArmourDefense", modparser.Max, modparser.Num(math.Min(pct*spellSuppressionChance/100, 1.0)*(armourPct-100)), "Chance to Defend from Spell Suppression: Average Calc", &modparser.CondTag{Var: "ArmourAvg"}))
		modDB.AddMod(newModS("ArmourDefense", modparser.Max, modparser.Num(math.Min(math.Floor(pct*spellSuppressionChance/100), 1.0)*(armourPct-100)), // #EVAL the reference reads a nil global `modSource` here, so the
			// fallback string is always the source
			"Chance to Defend from Spell Suppression: Min Calc", &modparser.CondTag{Var: "ArmourMax", Neg: true}, &modparser.CondTag{Var: "ArmourAvg", Neg: true}))
	}
	armourDefense, _ := modDB.Max(nil, "ArmourDefense")
	output.SetN("ArmourDefense", armourDefense/100)
	if output.N("ArmourDefense") > 0 {
		output.SetN("RawArmourDefense", (1+output.N("ArmourDefense"))*100)
	}

	if modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") {
		if base := modDB.Sum(modparser.Base, nil, "AvoidShock"); base != 0 {
			modDB.AddMod(newModS("AvoidShockAppliesToElementalAilments", modparser.Base, modparser.Num(base), "Stormshroud"))
		}
	}

	for _, ailment := range data.NonElementalAilmentTypeList {
		if modDB.Flag(nil, ailment+"Immune") {
			output.SetN(ailment+"AvoidChance", 100.0)
		} else {
			output.SetN(ailment+"AvoidChance", math.Floor(math.Min(modDB.Sum(modparser.Base, nil, "Avoid"+ailment, "AvoidAilments"), 100)))
		}
	}
	for _, ailment := range data.ElementalAilmentTypeList {
		shockAvoidAppliesToAll := modDB.Flag(nil, "ShockAvoidAppliesToElementalAilments") && ailment != "Shock"
		if modDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") {
			output.SetN(ailment+"AvoidChance", 100.0)
		} else {
			extra := 0.0
			if shockAvoidAppliesToAll {
				extra = modDB.Sum(modparser.Base, nil, "AvoidShock")
			}
			output.SetN(ailment+"AvoidChance", math.Floor(math.Min(modDB.Sum(modparser.Base, nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments")+extra, 100)))
		}
	}

	if modDB.Flag(nil, "CurseImmune") {
		output.SetN("CurseAvoidChance", 100.0)
	} else {
		output.SetN("CurseAvoidChance", math.Min(modDB.Sum(modparser.Base, nil, "AvoidCurse"), 100))
	}
	if modDB.Flag(nil, "SilenceImmune") {
		output.SetN("SilenceAvoidChance", 100.0)
	} else {
		output.SetN("SilenceAvoidChance", output.N("CurseAvoidChance"))
	}
	output.SetN("CritExtraDamageReduction", math.Min(modDB.Sum(modparser.Base, nil, "ReduceCritExtraDamage"), 100))
	output.SetN("LightRadiusMod", Mod(modDB, nil, "LightRadius"))
	output.SetN("LightRadiusInc", math.Max(modDB.Sum(modparser.Inc, nil, "LightRadius"), 0))
	output.SetN("CurseEffectOnSelf", math.Max(modDB.More(nil, "CurseEffectOnSelf")*(100+modDB.Sum(modparser.Inc, nil, "CurseEffectOnSelf")), 0))
	output.SetN("ExposureEffectOnSelf", modDB.More(nil, "ExposureEffectOnSelf")*(100+modDB.Sum(modparser.Inc, nil, "ExposureEffectOnSelf")))
	output.SetN("WitherEffectOnSelf", modDB.More(nil, "WitherEffectOnSelf")*(100+modDB.Sum(modparser.Inc, nil, "WitherEffectOnSelf")))

	// Ailment duration on self
	output.SetN("DebuffExpirationRate", modDB.Sum(modparser.Base, nil, "SelfDebuffExpirationRate"))
	output.SetN("DebuffExpirationModifier", 10000/(100+output.N("DebuffExpirationRate")))
	output.SetFlag("showDebuffExpirationModifier", output.N("DebuffExpirationModifier") != 100)
	output.SetN("SelfBlindDuration", modDB.More(nil, "SelfBlindDuration")*(100+modDB.Sum(modparser.Inc, nil, "SelfBlindDuration"))*output.N("DebuffExpirationModifier")/100)

	if modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") {
		inc := modDB.Sum(modparser.Inc, nil, "SelfIgniteDuration")
		more := modDB.More(nil, "SelfIgniteDuration")
		if inc != 0 {
			modDB.AddMod(newModS("SelfIgniteDurationToElementalAilments", modparser.Inc, modparser.Num(inc), "Firesong"))
		}
		if more != 1 {
			modDB.AddMod(newModS("SelfIgniteDurationToElementalAilments", modparser.More, modparser.Num(more), "Firesong"))
		}
	}

	for _, ailment := range data.NonElementalAilmentTypeList {
		more := modDB.More(nil, "Self"+ailment+"Duration", "SelfAilmentDuration")
		inc := (100 + modDB.Sum(modparser.Inc, nil, "Self"+ailment+"Duration", "SelfAilmentDuration")) * 100
		output.SetN("Self"+ailment+"Duration", inc*more/(100+output.N("DebuffExpirationRate")+modDB.Sum(modparser.Base, nil, "Self"+ailment+"DebuffExpirationRate")))
	}
	for _, ailment := range data.ElementalAilmentTypeList {
		igniteAppliesToAll := modDB.Flag(nil, "IgniteDurationAppliesToElementalAilments") && ailment != "Ignite"
		more := modDB.More(nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
		incExtra := 0.0
		if igniteAppliesToAll {
			more *= modDB.More(nil, "SelfIgniteDuration")
			incExtra = modDB.Sum(modparser.Inc, nil, "SelfIgniteDuration")
		}
		inc := (100 + modDB.Sum(modparser.Inc, nil, "Self"+ailment+"Duration", "SelfAilmentDuration", "SelfElementalAilmentDuration") + incExtra) * 100
		output.SetN("Self"+ailment+"Duration", more*inc/(100+output.N("DebuffExpirationRate")+modDB.Sum(modparser.Base, nil, "Self"+ailment+"DebuffExpirationRate")))
	}
	for _, ailment := range data.AilmentTypeList {
		selfEffect := Mod(modDB, nil, "Self"+ailment+"Effect")
		var enemyEffect float64
		if modDB.Flag(nil, "Condition:"+ailment+"edSelf") {
			enemyEffect = Mod(modDB, nil, "Enemy"+ailment+"Effect")
		} else {
			enemyEffect = Mod(enemyDB, nil, "Enemy"+ailment+"Effect")
		}
		output.SetN("Self"+ailment+"Effect", selfEffect*enemyEffect*100)
	}
}
