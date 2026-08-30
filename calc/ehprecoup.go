// CalcDefence.lua L2933-3070: recoup recovery, the Petrified Blood degen
// and the net recovery over time from enemy hits.
package calc

import "github.com/MissingL-tter/missingPassives/modparser"

func (env *Env) ehpRecoup(actor *performActor, damageCategoryConfig DamageCategory) {
	modDB := actor.db
	output := actor.output

	// recoup
	if output.N("anyRecoup") > 0 && damageCategoryConfig != DamageOverTime {
		totalDamage := 0.0
		totalElementalDamage := 0.0
		totalPhysicalDamageMitigated := output.N("NumberOfMitigatedDamagingHits") *
			(output.N("PhysicalTakenDamage") - output.N("PhysicalTakenHit"))
		extraPseudoRecoup := map[string][2]float64{}
		for _, damageType := range dmgTypeList {
			totalDamage += output.N(damageType + "RecoupableDamageTaken")
			if isElementalRes[damageType] {
				totalElementalDamage += output.N(damageType + "RecoupableDamageTaken")
			}
		}
		for i, recoupType := range recoupTypeList {
			recoupTime := 4.0
			if modDB.Flag(nil, "3Second"+recoupType+"Recoup") || modDB.Flag(nil, "3SecondRecoup") {
				recoupTime = 3
			}
			output.SetN("Total"+recoupType+"RecoupRecovery", output.N(recoupType+"Recoup")/100*totalDamage)
			if output.N("Elemental"+recoupType+"Recoup") > 0 && totalElementalDamage > 0 {
				output.SetN("Total"+recoupType+"RecoupRecovery", output.N("Total"+recoupType+"RecoupRecovery")+
					output.N("Elemental"+recoupType+"Recoup")/100*totalElementalDamage)
			}
			for _, damageType := range dmgTypeList {
				if output.N(damageType+recoupType+"Recoup") > 0 && output.N(damageType+"RecoupableDamageTaken") > 0 {
					output.SetN("Total"+recoupType+"RecoupRecovery", output.N("Total"+recoupType+"RecoupRecovery")+
						output.N(damageType+recoupType+"Recoup")/100*output.N(damageType+"RecoupableDamageTaken"))
				}
			}
			output.SetN("Total"+recoupType+"PseudoRecoup", output.N("PhysicalDamageMitigated"+recoupType+"PseudoRecoup")/100*totalPhysicalDamageMitigated)
			pseudoRecoupDuration := 4.0
			if v, ok := output["PhysicalDamageMitigated"+recoupType+"PseudoRecoupDuration"]; ok && v.Truthy() {
				pseudoRecoupDuration = v.Num()
			}
			// Pious Path
			if output.N("Total"+recoupType+"PseudoRecoup") != 0 {
				for j := i + 1; j < len(recoupTypeList); j++ {
					other := recoupTypeList[j]
					if modDB.Flag(nil, recoupType+"RegenerationRecovers"+other) &&
						!modDB.Flag(nil, "UnaffectedBy"+other+"Regen") &&
						!modDB.Flag(nil, "No"+other+"Regen") &&
						!modDB.Flag(nil, "CannotGain"+other) {
						extraPseudoRecoup[other] = [2]float64{
							output.N("Total"+recoupType+"PseudoRecoup") * output.N(other+"RecoveryRateMod") / output.N(recoupType+"RecoveryRateMod"),
							pseudoRecoupDuration,
						}
					}
				}
			}
			if modDB.Flag(nil, "UnaffectedBy"+recoupType+"Regen") {
				output.SetN("Total"+recoupType+"PseudoRecoup", 0.0)
			}
			extraMax, extraAvg := 0.0, 0.0
			if extra, ok := extraPseudoRecoup[recoupType]; ok {
				extraMax = extra[0] / extra[1]
				extraAvg = extra[0] / (output.N("EHPSurvivalTime") + extra[1])
			}
			output.SetN(recoupType+"RecoupRecoveryMax", output.N("Total"+recoupType+"RecoupRecovery")/recoupTime+
				output.N("Total"+recoupType+"PseudoRecoup")/pseudoRecoupDuration+extraMax)
			output.SetN(recoupType+"RecoupRecoveryAvg", output.N("Total"+recoupType+"RecoupRecovery")/(output.N("EHPSurvivalTime")+recoupTime)+
				output.N("Total"+recoupType+"PseudoRecoup")/(output.N("EHPSurvivalTime")+pseudoRecoupDuration)+extraAvg)
		}
	}

	// petrified blood "degen"
	_, hasLost := output["LifeLossLostOverTime"]
	_, hasBelowHalf := output["LifeBelowHalfLossLostOverTime"]
	if output.N("preventedLifeLossTotal") > 0 && hasLost && hasBelowHalf {
		lifeLossBelowHalfLost := modDB.Sum(modparser.Base, nil, "LifeLossBelowHalfLost") / 100
		total := output.N("LifeLossLostOverTime") + output.N("LifeBelowHalfLossLostOverTime")*lifeLossBelowHalfLost
		output.SetN("LifeLossLostMax", total/4)
		output.SetN("LifeLossLostAvg", total/(output.N("EHPSurvivalTime")+4))
	}

	// net recovery over time from enemy hits
	if output.N("LifeRecoupRecoveryAvg") > 0 || output.N("preventedLifeLossTotal") > 0 {
		output.SetN("netLifeRecoupAndLossLostOverTimeMax", output.N("LifeRecoupRecoveryMax")-output.N("LifeLossLostMax"))
		output.SetN("netLifeRecoupAndLossLostOverTimeAvg", output.N("LifeRecoupRecoveryAvg")-output.N("LifeLossLostAvg"))
		if output.N("LifeRecoupRecoveryAvg") > 0 && output.N("preventedLifeLossTotal") > 0 {
			output.SetFlag("showNetRecoup", true)
		}
	}
}
