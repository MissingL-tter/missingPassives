// CalcDefence.lua L149-175 and L3120-3325: takenHitFromDamage and the
// maximum-hit-taken solve (a quadratic per converted part, with an
// iterative smoothing pass when conversion splits the hit).
package calc

import "math"

// takenHitFromDamage ports calcs.takenHitFromDamage.
func (env *Env) takenHitFromDamage(rawDamage float64, damageType string, actor *performActor) (float64, map[string]float64) {
	output := actor.output
	modDB := actor.db

	damageMitigationMultiplierForType := func(damage float64, typ string) float64 {
		totalResistMult := outNum(output, typ+"ResistTakenHitMulti")
		effectiveAppliedArmour := outNum(output, typ+"EffectiveAppliedArmour")
		armourDRPercent := armourReductionF(effectiveAppliedArmour, damage*totalResistMult)
		flatDRPercent := 0.0
		if !modDB.Flag(nil, "SelfIgnoreBase"+typ+"DamageReduction") {
			if v, ok := output["Base"+typ+"DamageReductionWhenHit"]; ok && truthy(v) {
				flatDRPercent = anyNum(v)
			} else {
				flatDRPercent = outNum(output, "Base"+typ+"DamageReduction")
			}
		}
		totalDRPercent := math.Min(outNum(output, "DamageReductionMax"), armourDRPercent+flatDRPercent)
		enemyOverwhelmPercent := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+typ+"DamageReduction") {
			enemyOverwhelmPercent = outNum(output, typ+"EnemyOverwhelm")
		}
		totalDRMulti := 1 - math.Max(math.Min(outNum(output, "DamageReductionMax"), totalDRPercent-enemyOverwhelmPercent), 0)/100
		return totalResistMult * totalDRMulti
	}

	receivedDamageSum := 0.0
	damages := map[string]float64{}
	for _, damageConvertedType := range dmgTypeList {
		convertPercent, ok := actor.damageShiftTable[damageType][damageConvertedType]
		if !ok {
			continue
		}
		takenFlat := outNum(output, damageConvertedType+"takenFlat")
		if convertPercent > 0 || takenFlat != 0 {
			convertedDamage := rawDamage * convertPercent / 100
			vaalArctic := math.Min(-modDB.Sum("MORE", nil, "VaalArcticArmourMitigation")/100, 1)
			reducedDamage := roundDec(math.Max(convertedDamage*damageMitigationMultiplierForType(convertedDamage, damageConvertedType)+takenFlat, 0)*
				outNum(output, damageConvertedType+"AfterReductionTakenHitMulti"), 0) * (1 - vaalArctic)
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
	d := env.Data

	// fix total pools, as they aren't used anymore
	for _, damageType := range dmgTypeList {
		// ward
		wardBypass := modDB.Sum("BASE", nil, "WardBypass")
		if wardBypass > 0 {
			poolProtected := outNum(output, "Ward") / (1 - wardBypass/100) * (wardBypass / 100)
			sourcePool := outNum(output, damageType+"TotalHitPool")
			sourcePool = math.Max(sourcePool-poolProtected, 0) + math.Min(sourcePool, poolProtected)/(wardBypass/100)
			output[damageType+"TotalHitPool"] = sourcePool
		} else {
			output[damageType+"TotalHitPool"] = outNum(output, damageType+"TotalHitPool") + outNum(output, "Ward")
		}
		// aegis
		elementalAegis := 0.0
		if isElementalRes[damageType] {
			elementalAegis = outNum(output, damageType+"AegisDisplay")
		}
		output[damageType+"TotalHitPool"] = outNum(output, damageType+"TotalHitPool") +
			math.Max(math.Max(outNum(output, damageType+"Aegis"), outNum(output, "sharedAegis")), elementalAegis)
		// guard skill. #EVAL Lua's `a or 0 + b or 0` parses as `a or (0+b) or
		// 0`, so only the shared value is read here when it is non-nil.
		guardAbsorbRate := outNum(output, "sharedGuardAbsorbRate")
		if guardAbsorbRate == 0 {
			guardAbsorbRate = outNum(output, damageType+"GuardAbsorbRate")
		}
		if guardAbsorbRate > 0 {
			guardAbsorb := outNum(output, "sharedGuardAbsorb")
			if guardAbsorb == 0 {
				guardAbsorb = outNum(output, damageType+"GuardAbsorb")
			}
			if guardAbsorbRate >= 100 {
				output[damageType+"TotalHitPool"] = outNum(output, damageType+"TotalHitPool") + guardAbsorb
			} else {
				poolProtected := guardAbsorb / (guardAbsorbRate / 100) * (1 - guardAbsorbRate/100)
				output[damageType+"TotalHitPool"] = math.Max(outNum(output, damageType+"TotalHitPool")-poolProtected, 0) +
					math.Min(outNum(output, damageType+"TotalHitPool"), poolProtected)/(1-guardAbsorbRate/100)
			}
		}
		// Undo the ally pool drains in reverse order to recover the incoming hit.
		for index := len(allyLifePoolList) - 1; index >= 0; index-- {
			ally := allyLifePoolList[index]
			life, hasLife := output[ally.life]
			mitigation, hasMit := output[ally.mitigation]
			if hasLife && anyNum(life) > 0 && hasMit && anyNum(mitigation) > 0 {
				mit := math.Min(anyNum(mitigation), 100)
				if mit == 100 {
					output[damageType+"TotalHitPool"] = outNum(output, damageType+"TotalHitPool") + anyNum(life)
				} else {
					poolProtected := anyNum(life) / (mit / 100) * (1 - mit/100)
					output[damageType+"TotalHitPool"] = math.Max(outNum(output, damageType+"TotalHitPool")-poolProtected, 0) +
						math.Min(outNum(output, damageType+"TotalHitPool"), poolProtected)/(1-mit/100)
				}
			}
		}
	}

	for _, damageType := range dmgTypeList {
		partMin := math.Inf(1)
		useConversionSmoothing := false
		for _, damageConvertedType := range dmgTypeList {
			convertPercent := actor.damageShiftTable[damageType][damageConvertedType]
			takenFlat := outNum(output, damageConvertedType+"takenFlat")
			if convertPercent > 0 || takenFlat != 0 {
				hitTaken := 0.0
				effectiveAppliedArmour := outNum(output, damageConvertedType+"EffectiveAppliedArmour")
				damageConvertedMulti := convertPercent / 100
				totalHitPool := outNum(output, damageConvertedType+"TotalHitPool")
				totalTakenMulti := outNum(output, damageConvertedType+"AfterReductionTakenHitMulti") * (1 - outNum(output, "VaalArcticArmourMitigation"))
				if damageConvertedMulti <= 0 {
					takenWithoutIncoming := math.Max(takenFlat, 0) * totalTakenMulti
					if takenWithoutIncoming >= totalHitPool {
						hitTaken = 0
					} else {
						hitTaken = math.Inf(1)
					}
				} else if effectiveAppliedArmour == 0 && convertPercent == 100 {
					// simpler calculation for no armour DR
					totalResistMult := outNum(output, damageConvertedType+"ResistTakenHitMulti")
					drMulti := totalResistMult * (1 - outNum(output, damageConvertedType+"DamageReduction")/100)
					hitTaken = math.Max(totalHitPool/damageConvertedMulti/drMulti-takenFlat, 0) / totalTakenMulti
				} else {
					// Solve the damage chain backwards for the raw hit: see
					// the reference's derivation, a quadratic in RAW.
					totalResistMult := outNum(output, damageConvertedType+"ResistTakenHitMulti")
					reductionPercent := 0.0
					if !modDB.Flag(nil, "SelfIgnoreBase"+damageConvertedType+"DamageReduction") {
						if v, ok := output["Base"+damageConvertedType+"DamageReductionWhenHit"]; ok && truthy(v) {
							reductionPercent = anyNum(v)
						} else {
							reductionPercent = outNum(output, "Base"+damageConvertedType+"DamageReduction")
						}
					}
					flatDR := reductionPercent / 100
					enemyOverwhelmPercent := 0.0
					if !modDB.Flag(nil, "SelfIgnore"+damageConvertedType+"DamageReduction") {
						enemyOverwhelmPercent = outNum(output, damageConvertedType+"EnemyOverwhelm")
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
					maxDRMaxHit := noDRMaxHit / (1 - (outNum(output, "DamageReductionMax")-enemyOverwhelmPercent)/100)
					hitTaken = math.Floor(math.Max(math.Min(raw, maxDRMaxHit), noDRMaxHit))
					useConversionSmoothing = useConversionSmoothing || convertPercent != 100
				}
				partMin = math.Min(partMin, hitTaken)
			}
		}

		enemyDamageMult := outNum(output, damageType+"EnemyDamageMult")

		var finalMaxHit float64
		switch {
		case math.IsInf(partMin, 1):
			finalMaxHit = math.Inf(1)
		case useConversionSmoothing:
			passIncomingDamage := partMin
			previousOverkill := 0.0
			havePrevious := false
			for n := 1; float64(n) <= d.Misc.MaxHitSmoothingPasses; n++ {
				_, passDamages := env.takenHitFromDamage(passIncomingDamage, damageType, actor)
				passPools := env.reducePoolsByDamage(nil, passDamages, actor)
				passOverkill := passPools.OverkillDamage - passPools.hitPoolRemaining
				passRatio := 0.0
				for partType := range passDamages {
					partPool := outNum(output, partType+"TotalHitPool")
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
			finalMaxHit = roundDec(passIncomingDamage/enemyDamageMult, 0)
		default:
			finalMaxHit = roundDec(partMin/enemyDamageMult, 0)
		}

		output[damageType+"MaximumHitTaken"] = finalMaxHit
	}

	// second minimum used for power calcs, as there are issues using
	// average or minimum
	minimum := math.Inf(1)
	secondMinimum := math.Inf(1)
	for _, damageType := range dmgTypeList {
		v := outNum(output, damageType+"MaximumHitTaken")
		if v < minimum {
			secondMinimum = minimum
			minimum = v
		} else if v < secondMinimum {
			secondMinimum = v
		}
	}
	output["SecondMinimalMaximumHitTaken"] = secondMinimum

	// effective health pool vs dots
	for _, damageType := range dmgTypeList {
		output[damageType+"DotEHP"] = outNum(output, damageType+"TotalPool") / outNum(output, damageType+"TakenDotMult")
	}
}
