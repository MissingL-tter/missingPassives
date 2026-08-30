// CalcDefence.lua L149-175 and L3120-3325: takenHitFromDamage and the
// maximum-hit-taken solve (a quadratic per converted part, with an
// iterative smoothing pass when conversion splits the hit).
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"
)

// takenHitFromDamage ports calcs.takenHitFromDamage.
func (env *Env) takenHitFromDamage(rawDamage float64, damageType string, actor *performActor) (float64, map[string]float64) {
	output := actor.output
	modDB := actor.db

	damageMitigationMultiplierForType := func(damage float64, typ string) float64 {
		totalResistMult := output.N(typ + "ResistTakenHitMulti")
		effectiveAppliedArmour := output.N(typ + "EffectiveAppliedArmour")
		armourDRPercent := armourReductionF(effectiveAppliedArmour, damage*totalResistMult)
		flatDRPercent := 0.0
		if !modDB.Flag(nil, "SelfIgnoreBase"+typ+"DamageReduction") {
			if v, ok := output["Base"+typ+"DamageReductionWhenHit"]; ok && v.Truthy() {
				flatDRPercent = v.Num()
			} else {
				flatDRPercent = output.N("Base" + typ + "DamageReduction")
			}
		}
		totalDRPercent := math.Min(output.N("DamageReductionMax"), armourDRPercent+flatDRPercent)
		enemyOverwhelmPercent := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+typ+"DamageReduction") {
			enemyOverwhelmPercent = output.N(typ + "EnemyOverwhelm")
		}
		totalDRMulti := 1 - math.Max(math.Min(output.N("DamageReductionMax"), totalDRPercent-enemyOverwhelmPercent), 0)/100
		return totalResistMult * totalDRMulti
	}

	receivedDamageSum := 0.0
	damages := map[string]float64{}
	for _, damageConvertedType := range dmgTypeList {
		convertPercent, ok := actor.damageShiftTable[damageType][damageConvertedType]
		if !ok {
			continue
		}
		takenFlat := output.N(damageConvertedType + "takenFlat")
		if convertPercent > 0 || takenFlat != 0 {
			convertedDamage := rawDamage * convertPercent / 100
			vaalArctic := math.Min(-modDB.Sum(modparser.More, nil, "VaalArcticArmourMitigation")/100, 1)
			reducedDamage := util.RoundHalfUp(math.Max(convertedDamage*damageMitigationMultiplierForType(convertedDamage, damageConvertedType)+takenFlat, 0)*
				output.N(damageConvertedType+"AfterReductionTakenHitMulti"), 0) * (1 - vaalArctic)
			receivedDamageSum += reducedDamage
			if reducedDamage > 0 || convertPercent > 0 {
				damages[damageConvertedType] = reducedDamage
			}
		}
	}
	return receivedDamageSum, damages
}

func (env *Env) ehpMaxHit(actor *performActor) {
	modDB := actor.db
	output := actor.output

	// fix total pools, as they aren't used anymore
	for _, damageType := range dmgTypeList {
		// ward
		wardBypass := modDB.Sum(modparser.Base, nil, "WardBypass")
		if wardBypass > 0 {
			poolProtected := output.N("Ward") / (1 - wardBypass/100) * (wardBypass / 100)
			sourcePool := output.N(damageType + "TotalHitPool")
			sourcePool = math.Max(sourcePool-poolProtected, 0) + math.Min(sourcePool, poolProtected)/(wardBypass/100)
			output.SetN(damageType+"TotalHitPool", sourcePool)
		} else {
			output.SetN(damageType+"TotalHitPool", output.N(damageType+"TotalHitPool")+output.N("Ward"))
		}
		// aegis
		elementalAegis := 0.0
		if isElementalRes[damageType] {
			elementalAegis = output.N(damageType + "AegisDisplay")
		}
		output.SetN(damageType+"TotalHitPool", output.N(damageType+"TotalHitPool")+
			math.Max(math.Max(output.N(damageType+"Aegis"), output.N("sharedAegis")), elementalAegis))
		// guard skill. #EVAL Lua's `a or 0 + b or 0` parses as `a or (0+b) or
		// 0`, so only the shared value is read here when it is non-nil.
		guardAbsorbRate := output.N("sharedGuardAbsorbRate")
		if guardAbsorbRate == 0 {
			guardAbsorbRate = output.N(damageType + "GuardAbsorbRate")
		}
		if guardAbsorbRate > 0 {
			guardAbsorb := output.N("sharedGuardAbsorb")
			if guardAbsorb == 0 {
				guardAbsorb = output.N(damageType + "GuardAbsorb")
			}
			if guardAbsorbRate >= 100 {
				output.SetN(damageType+"TotalHitPool", output.N(damageType+"TotalHitPool")+guardAbsorb)
			} else {
				poolProtected := guardAbsorb / (guardAbsorbRate / 100) * (1 - guardAbsorbRate/100)
				output.SetN(damageType+"TotalHitPool", math.Max(output.N(damageType+"TotalHitPool")-poolProtected, 0)+
					math.Min(output.N(damageType+"TotalHitPool"), poolProtected)/(1-guardAbsorbRate/100))
			}
		}
		// Undo the ally pool drains in reverse order to recover the incoming hit.
		for index := len(allyLifePoolList) - 1; index >= 0; index-- {
			ally := allyLifePoolList[index]
			life, hasLife := output[ally.life]
			mitigation, hasMit := output[ally.mitigation]
			if hasLife && life.Num() > 0 && hasMit && mitigation.Num() > 0 {
				mit := math.Min(mitigation.Num(), 100)
				if mit == 100 {
					output.SetN(damageType+"TotalHitPool", output.N(damageType+"TotalHitPool")+life.Num())
				} else {
					poolProtected := life.Num() / (mit / 100) * (1 - mit/100)
					output.SetN(damageType+"TotalHitPool", math.Max(output.N(damageType+"TotalHitPool")-poolProtected, 0)+
						math.Min(output.N(damageType+"TotalHitPool"), poolProtected)/(1-mit/100))
				}
			}
		}
	}

	for _, damageType := range dmgTypeList {
		partMin := math.Inf(1)
		useConversionSmoothing := false
		for _, damageConvertedType := range dmgTypeList {
			convertPercent := actor.damageShiftTable[damageType][damageConvertedType]
			takenFlat := output.N(damageConvertedType + "takenFlat")
			if convertPercent > 0 || takenFlat != 0 {
				hitTaken := 0.0
				effectiveAppliedArmour := output.N(damageConvertedType + "EffectiveAppliedArmour")
				damageConvertedMulti := convertPercent / 100
				totalHitPool := output.N(damageConvertedType + "TotalHitPool")
				totalTakenMulti := output.N(damageConvertedType+"AfterReductionTakenHitMulti") * (1 - output.N("VaalArcticArmourMitigation"))
				if damageConvertedMulti <= 0 {
					takenWithoutIncoming := math.Max(takenFlat, 0) * totalTakenMulti
					if takenWithoutIncoming >= totalHitPool {
						hitTaken = 0
					} else {
						hitTaken = math.Inf(1)
					}
				} else if effectiveAppliedArmour == 0 && convertPercent == 100 {
					// simpler calculation for no armour DR
					totalResistMult := output.N(damageConvertedType + "ResistTakenHitMulti")
					drMulti := totalResistMult * (1 - output.N(damageConvertedType+"DamageReduction")/100)
					hitTaken = math.Max(totalHitPool/damageConvertedMulti/drMulti-takenFlat, 0) / totalTakenMulti
				} else {
					// Solve the damage chain backwards for the raw hit: see
					// the reference's derivation, a quadratic in RAW.
					totalResistMult := output.N(damageConvertedType + "ResistTakenHitMulti")
					reductionPercent := 0.0
					if !modDB.Flag(nil, "SelfIgnoreBase"+damageConvertedType+"DamageReduction") {
						if v, ok := output["Base"+damageConvertedType+"DamageReductionWhenHit"]; ok && v.Truthy() {
							reductionPercent = v.Num()
						} else {
							reductionPercent = output.N("Base" + damageConvertedType + "DamageReduction")
						}
					}
					flatDR := reductionPercent / 100
					enemyOverwhelmPercent := 0.0
					if !modDB.Flag(nil, "SelfIgnore"+damageConvertedType+"DamageReduction") {
						enemyOverwhelmPercent = output.N(damageConvertedType + "EnemyOverwhelm")
					}

					resistXConvert := totalResistMult * damageConvertedMulti
					a := 5 * (1 - flatDR + enemyOverwhelmPercent/100) * totalTakenMulti * resistXConvert * resistXConvert
					b := ((enemyOverwhelmPercent/100-flatDR)*effectiveAppliedArmour*totalTakenMulti - 5*(totalHitPool-takenFlat*totalTakenMulti)) * resistXConvert
					c := -effectiveAppliedArmour * (totalHitPool - takenFlat*totalTakenMulti)

					raw := math.Inf(1)
					if a != 0 {
						raw = (math.Sqrt(math.Max(b*b-4*a*c, 0)) - b) / (2 * a)
					}

					// tack on some caps
					noDRMaxHit := totalHitPool / damageConvertedMulti / totalResistMult / totalTakenMulti * (1 - takenFlat*totalTakenMulti/totalHitPool)
					maxDRMaxHit := noDRMaxHit / (1 - (output.N("DamageReductionMax")-enemyOverwhelmPercent)/100)
					hitTaken = math.Floor(math.Max(math.Min(raw, maxDRMaxHit), noDRMaxHit))
					useConversionSmoothing = useConversionSmoothing || convertPercent != 100
				}
				partMin = math.Min(partMin, hitTaken)
			}
		}

		enemyDamageMult := output.N(damageType + "EnemyDamageMult")

		var finalMaxHit float64
		switch {
		case math.IsInf(partMin, 1):
			finalMaxHit = math.Inf(1)
		case useConversionSmoothing:
			passIncomingDamage := partMin
			previousOverkill := 0.0
			havePrevious := false
			for n := 1; float64(n) <= data.Misc.MaxHitSmoothingPasses; n++ {
				_, passDamages := env.takenHitFromDamage(passIncomingDamage, damageType, actor)
				passPools := env.reducePoolsByDamage(nil, passDamages, actor)
				passOverkill := passPools.OverkillDamage - passPools.hitPoolRemaining
				passRatio := 0.0
				for partType := range passDamages {
					partPool := output.N(partType + "TotalHitPool")
					if partPool > 0 {
						passRatio = math.Max(passRatio, (passOverkill+partPool)/partPool)
					}
				}
				if passRatio <= 0 {
					passRatio = 1
				}
				stepSize := 1.0
				if n > 1 && havePrevious && previousOverkill != 0 && !math.IsNaN(previousOverkill) {
					stepSize = math.Min(math.Abs((passOverkill-previousOverkill)/previousOverkill), 2)
				}
				stepAdjust := 0.0
				if stepSize > 1 {
					stepAdjust = -passOverkill / stepSize
				} else if n > 1 {
					stepAdjust = -passOverkill * stepSize
				}
				previousOverkill = passOverkill
				havePrevious = true
				passIncomingDamage = (passIncomingDamage + stepAdjust) / math.Sqrt(passRatio)
				if passOverkill < 1 && passOverkill > -1 {
					break
				}
			}
			finalMaxHit = util.RoundHalfUp(passIncomingDamage/enemyDamageMult, 0)
		default:
			finalMaxHit = util.RoundHalfUp(partMin/enemyDamageMult, 0)
		}

		output.SetN(damageType+"MaximumHitTaken", finalMaxHit)
	}

	// second minimum used for power calcs, as there are issues using
	// average or minimum
	minimum := math.Inf(1)
	secondMinimum := math.Inf(1)
	for _, damageType := range dmgTypeList {
		v := output.N(damageType + "MaximumHitTaken")
		if v < minimum {
			secondMinimum = minimum
			minimum = v
		} else if v < secondMinimum {
			secondMinimum = v
		}
	}
	output.SetN("SecondMinimalMaximumHitTaken", secondMinimum)

	// effective health pool vs dots
	for _, damageType := range dmgTypeList {
		output.SetN(damageType+"DotEHP", output.N(damageType+"TotalPool")/output.N(damageType+"TakenDotMult"))
	}
}
