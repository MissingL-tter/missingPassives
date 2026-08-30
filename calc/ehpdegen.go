// CalcDefence.lua L3339-3828: build degens, enemy degens (config DoT or
// self-applied ailments), and the net / comprehensive-net regen totals.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

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
		takenFromMana := output.N(damageType+"MindOverMatter") + output.N("sharedMindOverMatter")
		bypass := output.N(damageType + "EnergyShieldBypass")
		if output.N("EnergyShieldRegenRecovery") > 0 {
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
		baseVal := modDB.Sum(modparser.Base, nil, damageType+"Degen")
		if baseVal > 0 {
			for _, damageConvertedType := range dmgTypeList {
				convertPercent := actor.damageOverTimeShiftTable[damageType][damageConvertedType]
				if convertPercent > 0 {
					total := baseVal * (convertPercent / 100) * output.N(damageConvertedType+"TakenDotMult")
					output.SetN(damageConvertedType+"BuildDegen", output.N(damageConvertedType+"BuildDegen")+total)
					totalBuildDegen += total
				}
			}
		}
	}
	if totalBuildDegen != 0 {
		output.SetN("TotalBuildDegen", totalBuildDegen)
		output.SetN("NetLifeRegen", output.N("LifeRegenRecovery"))
		output.SetN("NetManaRegen", output.N("ManaRegenRecovery"))
		output.SetN("NetEnergyShieldRegen", output.N("EnergyShieldRegenRecovery"))
		totalLifeDegen, totalManaDegen, totalEnergyShieldDegen := 0.0, 0.0, 0.0
		for _, damageType := range dmgTypeList {
			if v, ok := output[damageType+"BuildDegen"]; ok {
				l, m, e := splitDegen(damageType, v.Num())
				totalLifeDegen += l
				totalManaDegen += m
				totalEnergyShieldDegen += e
			}
		}
		output.SetN("NetLifeRegen", output.N("NetLifeRegen")-totalLifeDegen)
		output.SetN("NetManaRegen", output.N("NetManaRegen")-totalManaDegen)
		output.SetN("NetEnergyShieldRegen", output.N("NetEnergyShieldRegen")-totalEnergyShieldDegen)
		output.SetN("TotalNetRegen", output.N("NetLifeRegen")+output.N("NetManaRegen")+output.N("NetEnergyShieldRegen"))
	}

	enemyCritAilmentEffect := 1 + output.N("EnemyCritChance")/100*0.5*(1-output.N("CritExtraDamageReduction")/100)
	// this is just used so that ailments don't always show up if the enemy
	// has no other way of applying the ailment and they have a low crit chance
	const enemyCritThreshold = 10.1
	enemyBleedChance := 0.0
	enemyIgniteChance := 0.0
	if output.N("SelfIgniteEffect") != 0 && output.N("IgniteAvoidChance") < 100 &&
		output.N("SelfIgniteDuration") != 0 && damageCategoryConfig != "DamageOverTime" {
		enemyIgniteChance = enemyDB.Sum(modparser.Base, nil, "IgniteChance", "ElementalAilmentChance")
		enemyCritAilmentChance := 0.0
		if !modDB.Flag(nil, "CritsOnYouDontAlwaysApplyElementalAilments") &&
			(output.N("EnemyCritChance") > enemyCritThreshold || enemyIgniteChance > 0) {
			enemyCritAilmentChance = output.N("EnemyCritChance")
		}
		enemyIgniteChance = (enemyCritAilmentChance + (1-enemyCritAilmentChance/100)*enemyIgniteChance) *
			(1 - output.N("IgniteAvoidChance")/100)
	}
	enemyPoisonChance := 0.0
	if output.N("SelfPoisonEffect") != 0 && output.N("PoisonAvoidChance") < 100 &&
		output.N("SelfPoisonDuration") != 0 && damageCategoryConfig != "DamageOverTime" {
		enemyPoisonChance = enemyDB.Sum(modparser.Base, nil, "PoisonChance") * (1 - output.N("PoisonAvoidChance")/100)
	}

	if damageCategoryConfig == "DamageOverTime" || (enemyIgniteChance+enemyPoisonChance+enemyBleedChance) > 0 {
		totalDegen := totalBuildDegen
		if damageCategoryConfig == "DamageOverTime" {
			for _, damageType := range dmgTypeList {
				baseVal := env.configOrPlaceholder(damageType, func(c *ConfigInput) map[string]float64 { return c.EnemyDamage })
				if baseVal > 0 {
					for _, damageConvertedType := range dmgTypeList {
						convertPercent := actor.damageOverTimeShiftTable[damageType][damageConvertedType]
						if convertPercent > 0 {
							total := baseVal * (convertPercent / 100) * output.N(damageConvertedType+"TakenDotMult")
							output.SetN(damageConvertedType+"EnemyDegen", output.N(damageConvertedType+"EnemyDegen")+total)
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
					baseVal += output.N(damageType + "TakenDamage")
				}
				baseVal = baseVal * ailmentPercentBase(ailment.source) *
					(enemyCritAilmentEffect / output.N("EnemyCritEffect")) *
					output.N("Self"+ailment.source+"Effect") / 100
				if baseVal > 0 {
					for _, damageConvertedType := range dmgTypeList {
						convertPercent := actor.damageOverTimeShiftTable[ailment.damageType][damageConvertedType]
						if convertPercent > 0 {
							total := baseVal * (convertPercent / 100) * output.N(damageConvertedType+"TakenDotMult")
							output.SetN(damageConvertedType+"EnemyDegen", output.N(damageConvertedType+"EnemyDegen")+total)
							totalDegen += total
						}
					}
				}
			}
		}
		if totalDegen != totalBuildDegen {
			output.SetN("TotalDegen", totalDegen)
			output.SetN("ComprehensiveNetLifeRegen", output.N("LifeRegenRecovery"))
			output.SetN("ComprehensiveNetManaRegen", output.N("ManaRegenRecovery"))
			output.SetN("ComprehensiveNetEnergyShieldRegen", output.N("EnergyShieldRegenRecovery"))
			totalLifeDegen, totalManaDegen, totalEnergyShieldDegen := 0.0, 0.0, 0.0
			for _, damageType := range dmgTypeList {
				typeDegen := output.N(damageType+"BuildDegen") + output.N(damageType+"EnemyDegen")
				if typeDegen != 0 {
					l, m, e := splitDegen(damageType, typeDegen)
					totalLifeDegen += l
					totalManaDegen += m
					totalEnergyShieldDegen += e
				}
			}
			output.SetN("ComprehensiveNetLifeRegen", output.N("ComprehensiveNetLifeRegen")+
				output.N("LifeRecoupRecoveryAvg")-totalLifeDegen-output.N("LifeLossLostAvg"))
			output.SetN("ComprehensiveNetManaRegen", output.N("ComprehensiveNetManaRegen")+
				output.N("ManaRecoupRecoveryAvg")-totalManaDegen)
			output.SetN("ComprehensiveNetEnergyShieldRegen", output.N("ComprehensiveNetEnergyShieldRegen")+
				output.N("EnergyShieldRecoupRecoveryAvg")-totalEnergyShieldDegen)
			output.SetN("ComprehensiveTotalNetRegen", output.N("ComprehensiveNetLifeRegen")+
				output.N("ComprehensiveNetManaRegen")+output.N("ComprehensiveNetEnergyShieldRegen"))
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
