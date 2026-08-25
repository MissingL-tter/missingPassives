// CalcDefence.lua L3339-3828: build degens, enemy degens (config DoT or
// self-applied ailments), and the net / comprehensive-net regen totals.
package calc

import "github.com/MissingL-tter/missingPassives/data"

// ailmentDegen is one entry of the reference's ailmentList.
type ailmentDegen struct {
	source      string
	damageType  string
	sourceTypes []string
}

func (env *Env) ehpDegens(actor *performActor, damageCategoryConfig string) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	// splitDegen divides one degen amount over life / mana / energy shield.
	splitDegen := func(damageType string, amount float64) (lifeDegen, manaDegen, energyShieldDegen float64) {
		takenFromMana := outNum(output, damageType+"MindOverMatter") + outNum(output, "sharedMindOverMatter")
		bypass := outNum(output, damageType+"EnergyShieldBypass")
		if outNum(output, "EnergyShieldRegenRecovery") > 0 {
			if modDB.Flag(nil, "EnergyShieldProtectsMana") {
				lifeDegen = amount * (1 - takenFromMana/100)
				energyShieldDegen = amount * (1 - bypass/100) * (takenFromMana / 100)
			} else {
				lifeDegen = amount * (bypass / 100) * (1 - takenFromMana/100)
				energyShieldDegen = amount * (1 - bypass/100)
			}
			manaDegen = amount * (bypass / 100) * (takenFromMana / 100)
		} else {
			lifeDegen = amount * (1 - takenFromMana/100)
			manaDegen = amount * (takenFromMana / 100)
		}
		return
	}

	// build degens
	totalBuildDegen := 0.0
	for _, damageType := range dmgTypeList {
		baseVal := modDB.Sum("BASE", nil, damageType+"Degen")
		if baseVal > 0 {
			for _, damageConvertedType := range dmgTypeList {
				convertPercent := actor.damageOverTimeShiftTable[damageType][damageConvertedType]
				if convertPercent > 0 {
					total := baseVal * (convertPercent / 100) * outNum(output, damageConvertedType+"TakenDotMult")
					output[damageConvertedType+"BuildDegen"] = outNum(output, damageConvertedType+"BuildDegen") + total
					totalBuildDegen += total
				}
			}
		}
	}
	if totalBuildDegen != 0 {
		output["TotalBuildDegen"] = totalBuildDegen
		output["NetLifeRegen"] = outNum(output, "LifeRegenRecovery")
		output["NetManaRegen"] = outNum(output, "ManaRegenRecovery")
		output["NetEnergyShieldRegen"] = outNum(output, "EnergyShieldRegenRecovery")
		totalLifeDegen, totalManaDegen, totalEnergyShieldDegen := 0.0, 0.0, 0.0
		for _, damageType := range dmgTypeList {
			if v, ok := output[damageType+"BuildDegen"]; ok {
				l, m, e := splitDegen(damageType, anyNum(v))
				totalLifeDegen += l
				totalManaDegen += m
				totalEnergyShieldDegen += e
			}
		}
		output["NetLifeRegen"] = outNum(output, "NetLifeRegen") - totalLifeDegen
		output["NetManaRegen"] = outNum(output, "NetManaRegen") - totalManaDegen
		output["NetEnergyShieldRegen"] = outNum(output, "NetEnergyShieldRegen") - totalEnergyShieldDegen
		output["TotalNetRegen"] = outNum(output, "NetLifeRegen") + outNum(output, "NetManaRegen") + outNum(output, "NetEnergyShieldRegen")
	}

	enemyCritAilmentEffect := 1 + outNum(output, "EnemyCritChance")/100*0.5*(1-outNum(output, "CritExtraDamageReduction")/100)
	// this is just used so that ailments don't always show up if the enemy
	// has no other way of applying the ailment and they have a low crit chance
	const enemyCritThreshold = 10.1
	enemyBleedChance := 0.0
	enemyIgniteChance := 0.0
	if outNum(output, "SelfIgniteEffect") != 0 && outNum(output, "IgniteAvoidChance") < 100 &&
		outNum(output, "SelfIgniteDuration") != 0 && damageCategoryConfig != "DamageOverTime" {
		enemyIgniteChance = enemyDB.Sum("BASE", nil, "IgniteChance", "ElementalAilmentChance")
		enemyCritAilmentChance := 0.0
		if !modDB.Flag(nil, "CritsOnYouDontAlwaysApplyElementalAilments") &&
			(outNum(output, "EnemyCritChance") > enemyCritThreshold || enemyIgniteChance > 0) {
			enemyCritAilmentChance = outNum(output, "EnemyCritChance")
		}
		enemyIgniteChance = (enemyCritAilmentChance + (1-enemyCritAilmentChance/100)*enemyIgniteChance) *
			(1 - outNum(output, "IgniteAvoidChance")/100)
	}
	enemyPoisonChance := 0.0
	if outNum(output, "SelfPoisonEffect") != 0 && outNum(output, "PoisonAvoidChance") < 100 &&
		outNum(output, "SelfPoisonDuration") != 0 && damageCategoryConfig != "DamageOverTime" {
		enemyPoisonChance = enemyDB.Sum("BASE", nil, "PoisonChance") * (1 - outNum(output, "PoisonAvoidChance")/100)
	}

	if damageCategoryConfig == "DamageOverTime" || (enemyIgniteChance+enemyPoisonChance+enemyBleedChance) > 0 {
		totalDegen := totalBuildDegen
		if damageCategoryConfig == "DamageOverTime" {
			for _, damageType := range dmgTypeList {
				baseVal := 0.0
				if p := tonum(env.ConfigInput["enemy"+damageType+"Damage"]); p != nil {
					baseVal = *p
				} else if p := tonum(env.Build.ConfigPlaceholder["enemy"+damageType+"Damage"]); p != nil {
					baseVal = *p
				}
				if baseVal > 0 {
					for _, damageConvertedType := range dmgTypeList {
						convertPercent := actor.damageOverTimeShiftTable[damageType][damageConvertedType]
						if convertPercent > 0 {
							total := baseVal * (convertPercent / 100) * outNum(output, damageConvertedType+"TakenDotMult")
							output[damageConvertedType+"EnemyDegen"] = outNum(output, damageConvertedType+"EnemyDegen") + total
							totalDegen += total
						}
					}
				}
			}
		} else {
			// ailmentList is keyed by source name; sorted for determinism
			// (each entry writes to its own damage type, so order is
			// immaterial to the result).
			var ailmentList []ailmentDegen
			if enemyBleedChance > 0 {
				ailmentList = append(ailmentList, ailmentDegen{"Bleed", "Physical", []string{"Physical"}})
			}
			if enemyIgniteChance > 0 {
				sourceTypes := []string{"Fire"}
				if enemyDB.Flag(nil, "AllDamageIgnites") {
					sourceTypes = dmgTypeList
				}
				ailmentList = append(ailmentList, ailmentDegen{"Ignite", "Fire", sourceTypes})
			}
			if enemyPoisonChance > 0 {
				ailmentList = append(ailmentList, ailmentDegen{"Poison", "Chaos", []string{"Physical", "Chaos"}})
			}
			for _, ailment := range ailmentList {
				baseVal := 0.0
				for _, damageType := range ailment.sourceTypes {
					baseVal += outNum(output, damageType+"TakenDamage")
				}
				baseVal = baseVal * ailmentPercentBase(ailment.source) *
					(enemyCritAilmentEffect / outNum(output, "EnemyCritEffect")) *
					outNum(output, "Self"+ailment.source+"Effect") / 100
				if baseVal > 0 {
					for _, damageConvertedType := range dmgTypeList {
						convertPercent := actor.damageOverTimeShiftTable[ailment.damageType][damageConvertedType]
						if convertPercent > 0 {
							total := baseVal * (convertPercent / 100) * outNum(output, damageConvertedType+"TakenDotMult")
							output[damageConvertedType+"EnemyDegen"] = outNum(output, damageConvertedType+"EnemyDegen") + total
							totalDegen += total
						}
					}
				}
			}
		}
		if totalDegen != totalBuildDegen {
			output["TotalDegen"] = totalDegen
			output["ComprehensiveNetLifeRegen"] = outNum(output, "LifeRegenRecovery")
			output["ComprehensiveNetManaRegen"] = outNum(output, "ManaRegenRecovery")
			output["ComprehensiveNetEnergyShieldRegen"] = outNum(output, "EnergyShieldRegenRecovery")
			totalLifeDegen, totalManaDegen, totalEnergyShieldDegen := 0.0, 0.0, 0.0
			for _, damageType := range dmgTypeList {
				typeDegen := outNum(output, damageType+"BuildDegen") + outNum(output, damageType+"EnemyDegen")
				if typeDegen != 0 {
					l, m, e := splitDegen(damageType, typeDegen)
					totalLifeDegen += l
					totalManaDegen += m
					totalEnergyShieldDegen += e
				}
			}
			output["ComprehensiveNetLifeRegen"] = outNum(output, "ComprehensiveNetLifeRegen") +
				outNum(output, "LifeRecoupRecoveryAvg") - totalLifeDegen - outNum(output, "LifeLossLostAvg")
			output["ComprehensiveNetManaRegen"] = outNum(output, "ComprehensiveNetManaRegen") +
				outNum(output, "ManaRecoupRecoveryAvg") - totalManaDegen
			output["ComprehensiveNetEnergyShieldRegen"] = outNum(output, "ComprehensiveNetEnergyShieldRegen") +
				outNum(output, "EnergyShieldRecoupRecoveryAvg") - totalEnergyShieldDegen
			output["ComprehensiveTotalNetRegen"] = outNum(output, "ComprehensiveNetLifeRegen") +
				outNum(output, "ComprehensiveNetManaRegen") + outNum(output, "ComprehensiveNetEnergyShieldRegen")
		}
	}
}

// ailmentPercentBase reads data.misc["<source>PercentBase"].
func ailmentPercentBase(source string) float64 {
	switch source {
	case "Bleed":
		return data.Misc.BleedPercentBase
	case "Ignite":
		return data.Misc.IgnitePercentBase
	case "Poison":
		return data.Misc.PoisonPercentBase
	}
	panic("ehp: unknown ailment degen source " + source)
}
