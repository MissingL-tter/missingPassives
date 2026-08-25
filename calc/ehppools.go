// CalcDefence.lua L2214-2377: the recoverable life pool, prevented life
// loss (Petrified Blood), energy shield bypass and Mind over Matter.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
)

// calcLifeHitPoolWithLossPrevention ports the local of the same name.
func calcLifeHitPoolWithLossPrevention(life, maxLife, lifeLossPrevented, lifeLossBelowHalfPrevented float64) float64 {
	halfLife := maxLife * 0.5
	aboveLow := math.Max(life-halfLife, 0)
	return aboveLow/(1-lifeLossPrevented/100) +
		math.Min(life, halfLife)/(1-lifeLossBelowHalfPrevented/100)/(1-lifeLossPrevented/100)
}

func (env *Env) ehpPools(actor *performActor) {
	modDB := actor.db
	output := actor.output

	// Life Recoverable
	output["LifeRecoverable"] = outNum(output, "LifeUnreserved")
	if truthy(env.ConfigInput["conditionLowLife"]) {
		lowPerc := data.Misc.LowPoolThreshold
		if v, ok := output["LowLifePercentage"]; ok && truthy(v) {
			lowPerc = anyNum(v)
		}
		output["LifeRecoverable"] = math.Min(outNum(output, "Life")*lowPerc/100, outNum(output, "LifeUnreserved"))
		if outNum(output, "LifeRecoverable") < outNum(output, "LifeUnreserved") {
			output["CappingLife"] = true
		}
	}

	// Dissolution of the flesh life pool change
	if modDB.Flag(nil, "DamageInsteadReservesLife") {
		output["LifeRecoverable"] = (outNum(output, "LifeCancellableReservation") / 100) * outNum(output, "Life")
	}

	output["LifeRecoverable"] = math.Max(outNum(output, "LifeRecoverable"), 1)

	// Prevented life loss taken over 4 seconds (and Petrified Blood)
	{
		recoverable := outNum(output, "LifeRecoverable")
		preventedLifeLoss := math.Min(modDB.Sum("BASE", nil, "LifeLossPrevented"), 100)
		output["preventedLifeLoss"] = preventedLifeLoss
		initialLifeLossBelowHalfPrevented := modDB.Sum("BASE", nil, "LifeLossBelowHalfPrevented")
		output["preventedLifeLossBelowHalf"] = (1 - preventedLifeLoss/100) * initialLifeLossBelowHalfPrevented
		if !truthy(env.ConfigInput["conditionLowLife"]) {
			// portion of life that is lowlife
			portionLife := math.Min(outNum(output, "Life")*0.5/recoverable, 1)
			output["preventedLifeLossTotal"] = preventedLifeLoss + outNum(output, "preventedLifeLossBelowHalf")*portionLife
		} else {
			output["preventedLifeLossTotal"] = preventedLifeLoss + outNum(output, "preventedLifeLossBelowHalf")
		}
		output["LifeHitPool"] = calcLifeHitPoolWithLossPrevention(recoverable, outNum(output, "Life"),
			preventedLifeLoss, initialLifeLossBelowHalfPrevented)
	}

	// Energy Shield bypass
	output["AnyBypass"] = false
	output["MinimumBypass"] = 100.0
	for _, damageType := range dmgTypeList {
		if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
			output[damageType+"EnergyShieldBypass"] = 100.0
			output["AnyBypass"] = true
		} else {
			if ov := modDB.Override(nil, damageType+"EnergyShieldBypass"); truthy(ov) {
				output[damageType+"EnergyShieldBypass"] = anyNum(ov)
			} else {
				output[damageType+"EnergyShieldBypass"] = modDB.Sum("BASE", nil, damageType+"EnergyShieldBypass")
			}
			if outNum(output, damageType+"EnergyShieldBypass") != 0 {
				output["AnyBypass"] = true
			}
			if damageType == "Chaos" {
				if !modDB.Flag(nil, "ChaosNotBypassEnergyShield") {
					output[damageType+"EnergyShieldBypass"] = outNum(output, damageType+"EnergyShieldBypass") + 100
				} else {
					output["AnyBypass"] = true
				}
			}
		}
		output[damageType+"EnergyShieldBypass"] = math.Max(math.Min(outNum(output, damageType+"EnergyShieldBypass"), 100), 0)
		output["MinimumBypass"] = math.Min(outNum(output, "MinimumBypass"), outNum(output, damageType+"EnergyShieldBypass"))
	}

	output["ehpSectionAnySpecificTypes"] = false
	// Mind over Matter
	output["OnlySharedMindOverMatter"] = false
	output["AnySpecificMindOverMatter"] = false
	output["sharedMindOverMatter"] = math.Min(modDB.Sum("BASE", nil, "DamageTakenFromManaBeforeLife"), 100)

	// calcMoMEBPool returns the combined pool plus the mana/ES breakdown.
	calcMoMEBPool := func(lifePool, momEffect, esBypass float64) (pool, maxManaUsable, manaUsed, maxESUsable float64) {
		mana := math.Max(outNum(output, "ManaUnreserved"), 0)
		maxMoMPool := math.Inf(1)
		if momEffect < 1 {
			maxMoMPool = lifePool/(1-momEffect) - lifePool
		}
		maxManaUsable = math.Floor(math.Min(mana, maxMoMPool))
		maxESUsable = 0
		if modDB.Flag(nil, "EnergyShieldProtectsMana") && esBypass < 1 {
			maxESUsable = math.Floor(math.Min(math.Min(
				outNum(output, "EnergyShieldRecoveryCap"),
				maxMoMPool*(1-esBypass)),
				(lifePool+maxManaUsable)/(1-(1-esBypass)*momEffect)-(lifePool+maxManaUsable)))
		}
		manaUsed = math.Floor(math.Min(maxMoMPool-maxESUsable, maxManaUsable))
		return lifePool + manaUsed + maxESUsable, maxManaUsable, manaUsed, maxESUsable
	}

	if outNum(output, "sharedMindOverMatter") > 0 {
		mindOverMatter := outNum(output, "sharedMindOverMatter") / 100
		esBypass := outNum(output, "MinimumBypass") / 100
		sharedMoMPool, _, _, _ := calcMoMEBPool(outNum(output, "LifeRecoverable"), mindOverMatter, esBypass)
		output["sharedManaEffectiveLife"] = sharedMoMPool
		hitPool, _, _, _ := calcMoMEBPool(outNum(output, "LifeHitPool"), mindOverMatter, esBypass)
		output["sharedMoMHitPool"] = hitPool
	} else {
		output["sharedManaEffectiveLife"] = outNum(output, "LifeRecoverable")
		output["sharedMoMHitPool"] = outNum(output, "LifeHitPool")
	}
	for _, damageType := range dmgTypeList {
		output[damageType+"MindOverMatter"] = math.Min(modDB.Sum("BASE", nil, damageType+"DamageTakenFromManaBeforeLife"),
			100-outNum(output, "sharedMindOverMatter"))
		if outNum(output, damageType+"MindOverMatter") > 0 ||
			(outNum(output, damageType+"EnergyShieldBypass") > outNum(output, "MinimumBypass") && outNum(output, "sharedMindOverMatter") > 0) {
			output["ehpSectionAnySpecificTypes"] = true
			output["AnySpecificMindOverMatter"] = true
			output["OnlySharedMindOverMatter"] = false
			mindOverMatter := (outNum(output, damageType+"MindOverMatter") + outNum(output, "sharedMindOverMatter")) / 100
			esBypass := outNum(output, damageType+"EnergyShieldBypass") / 100
			typedMoMPool, _, _, _ := calcMoMEBPool(outNum(output, "LifeRecoverable"), mindOverMatter, esBypass)
			output[damageType+"ManaEffectiveLife"] = typedMoMPool
			hitPool, _, _, _ := calcMoMEBPool(outNum(output, "LifeHitPool"), mindOverMatter, esBypass)
			output[damageType+"MoMHitPool"] = hitPool
		} else {
			output[damageType+"ManaEffectiveLife"] = outNum(output, "sharedManaEffectiveLife")
			output[damageType+"MoMHitPool"] = outNum(output, "sharedMoMHitPool")
		}
	}
}
