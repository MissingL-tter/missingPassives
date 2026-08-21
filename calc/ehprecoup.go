// CalcDefence.lua L2933-3070: recoup recovery, the Petrified Blood degen
// and the net recovery over time from enemy hits.
package calc

func (env *Env) ehpRecoup(actor *performActor, damageCategoryConfig string) {
	modDB := actor.db
	output := actor.output

	// recoup
	if outNum(output, "anyRecoup") > 0 && damageCategoryConfig != "DamageOverTime" {
		totalDamage := 0.0
		totalElementalDamage := 0.0
		totalPhysicalDamageMitigated := outNum(output, "NumberOfMitigatedDamagingHits") *
			(outNum(output, "PhysicalTakenDamage") - outNum(output, "PhysicalTakenHit"))
		extraPseudoRecoup := map[string][2]float64{}
		for _, damageType := range dmgTypeList {
			totalDamage += outNum(output, damageType+"RecoupableDamageTaken")
			if isElementalRes[damageType] {
				totalElementalDamage += outNum(output, damageType+"RecoupableDamageTaken")
			}
		}
		for i, recoupType := range recoupTypeList {
			recoupTime := 4.0
			if modDB.Flag(nil, "3Second"+recoupType+"Recoup") || modDB.Flag(nil, "3SecondRecoup") {
				recoupTime = 3
			}
			output["Total"+recoupType+"RecoupRecovery"] = outNum(output, recoupType+"Recoup") / 100 * totalDamage
			if outNum(output, "Elemental"+recoupType+"Recoup") > 0 && totalElementalDamage > 0 {
				output["Total"+recoupType+"RecoupRecovery"] = outNum(output, "Total"+recoupType+"RecoupRecovery") +
					outNum(output, "Elemental"+recoupType+"Recoup")/100*totalElementalDamage
			}
			for _, damageType := range dmgTypeList {
				if outNum(output, damageType+recoupType+"Recoup") > 0 && outNum(output, damageType+"RecoupableDamageTaken") > 0 {
					output["Total"+recoupType+"RecoupRecovery"] = outNum(output, "Total"+recoupType+"RecoupRecovery") +
						outNum(output, damageType+recoupType+"Recoup")/100*outNum(output, damageType+"RecoupableDamageTaken")
				}
			}
			output["Total"+recoupType+"PseudoRecoup"] = outNum(output, "PhysicalDamageMitigated"+recoupType+"PseudoRecoup") / 100 * totalPhysicalDamageMitigated
			pseudoRecoupDuration := 4.0
			if v, ok := output["PhysicalDamageMitigated"+recoupType+"PseudoRecoupDuration"]; ok && truthy(v) {
				pseudoRecoupDuration = anyNum(v)
			}
			// Pious Path
			if outNum(output, "Total"+recoupType+"PseudoRecoup") != 0 {
				for j := i + 1; j < len(recoupTypeList); j++ {
					other := recoupTypeList[j]
					if modDB.Flag(nil, recoupType+"RegenerationRecovers"+other) &&
						!modDB.Flag(nil, "UnaffectedBy"+other+"Regen") &&
						!modDB.Flag(nil, "No"+other+"Regen") &&
						!modDB.Flag(nil, "CannotGain"+other) {
						extraPseudoRecoup[other] = [2]float64{
							outNum(output, "Total"+recoupType+"PseudoRecoup") * outNum(output, other+"RecoveryRateMod") / outNum(output, recoupType+"RecoveryRateMod"),
							pseudoRecoupDuration,
						}
					}
				}
			}
			if modDB.Flag(nil, "UnaffectedBy"+recoupType+"Regen") {
				output["Total"+recoupType+"PseudoRecoup"] = 0.0
			}
			extraMax, extraAvg := 0.0, 0.0
			if extra, ok := extraPseudoRecoup[recoupType]; ok {
				extraMax = extra[0] / extra[1]
				extraAvg = extra[0] / (outNum(output, "EHPSurvivalTime") + extra[1])
			}
			output[recoupType+"RecoupRecoveryMax"] = outNum(output, "Total"+recoupType+"RecoupRecovery")/recoupTime +
				outNum(output, "Total"+recoupType+"PseudoRecoup")/pseudoRecoupDuration + extraMax
			output[recoupType+"RecoupRecoveryAvg"] = outNum(output, "Total"+recoupType+"RecoupRecovery")/(outNum(output, "EHPSurvivalTime")+recoupTime) +
				outNum(output, "Total"+recoupType+"PseudoRecoup")/(outNum(output, "EHPSurvivalTime")+pseudoRecoupDuration) + extraAvg
		}
	}

	// petrified blood "degen"
	_, hasLost := output["LifeLossLostOverTime"]
	_, hasBelowHalf := output["LifeBelowHalfLossLostOverTime"]
	if outNum(output, "preventedLifeLossTotal") > 0 && hasLost && hasBelowHalf {
		lifeLossBelowHalfLost := modDB.Sum("BASE", nil, "LifeLossBelowHalfLost") / 100
		total := outNum(output, "LifeLossLostOverTime") + outNum(output, "LifeBelowHalfLossLostOverTime")*lifeLossBelowHalfLost
		output["LifeLossLostMax"] = total / 4
		output["LifeLossLostAvg"] = total / (outNum(output, "EHPSurvivalTime") + 4)
	}

	// net recovery over time from enemy hits
	if outNum(output, "LifeRecoupRecoveryAvg") > 0 || outNum(output, "preventedLifeLossTotal") > 0 {
		output["netLifeRecoupAndLossLostOverTimeMax"] = outNum(output, "LifeRecoupRecoveryMax") - outNum(output, "LifeLossLostMax")
		output["netLifeRecoupAndLossLostOverTimeAvg"] = outNum(output, "LifeRecoupRecoveryAvg") - outNum(output, "LifeLossLostAvg")
		if outNum(output, "LifeRecoupRecoveryAvg") > 0 && outNum(output, "preventedLifeLossTotal") > 0 {
			output["showNetRecoup"] = true
		}
	}
}
