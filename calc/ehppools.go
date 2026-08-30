// CalcDefence.lua L2214-2377: the recoverable life pool, prevented life
// loss (Petrified Blood), energy shield bypass and Mind over Matter.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
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
	output.SetN("LifeRecoverable", output.N("LifeUnreserved"))
	if env.ConfigInput.ConditionLowLife {
		lowPerc := data.Misc.LowPoolThreshold
		if v, ok := output["LowLifePercentage"]; ok && v.Truthy() {
			lowPerc = v.Num()
		}
		output.SetN("LifeRecoverable", math.Min(output.N("Life")*lowPerc/100, output.N("LifeUnreserved")))
		if output.N("LifeRecoverable") < output.N("LifeUnreserved") {
			output.SetFlag("CappingLife", true)
		}
	}

	// Dissolution of the flesh life pool change
	if modDB.Flag(nil, "DamageInsteadReservesLife") {
		output.SetN("LifeRecoverable", (output.N("LifeCancellableReservation")/100)*output.N("Life"))
	}

	output.SetN("LifeRecoverable", math.Max(output.N("LifeRecoverable"), 1))

	// Prevented life loss taken over 4 seconds (and Petrified Blood)
	{
		recoverable := output.N("LifeRecoverable")
		preventedLifeLoss := math.Min(modDB.Sum(modparser.Base, nil, "LifeLossPrevented"), 100)
		output.SetN("preventedLifeLoss", preventedLifeLoss)
		initialLifeLossBelowHalfPrevented := modDB.Sum(modparser.Base, nil, "LifeLossBelowHalfPrevented")
		output.SetN("preventedLifeLossBelowHalf", (1-preventedLifeLoss/100)*initialLifeLossBelowHalfPrevented)
		if !env.ConfigInput.ConditionLowLife {
			// portion of life that is lowlife
			portionLife := math.Min(output.N("Life")*0.5/recoverable, 1)
			output.SetN("preventedLifeLossTotal", preventedLifeLoss+output.N("preventedLifeLossBelowHalf")*portionLife)
		} else {
			output.SetN("preventedLifeLossTotal", preventedLifeLoss+output.N("preventedLifeLossBelowHalf"))
		}
		output.SetN("LifeHitPool", calcLifeHitPoolWithLossPrevention(recoverable, output.N("Life"),
			preventedLifeLoss, initialLifeLossBelowHalfPrevented))
	}

	// Energy Shield bypass
	output.SetFlag("AnyBypass", false)
	output.SetN("MinimumBypass", 100.0)
	for _, damageType := range dmgTypeList {
		if modDB.Flag(nil, "UnblockedDamageDoesBypassES") {
			output.SetN(damageType+"EnergyShieldBypass", 100.0)
			output.SetFlag("AnyBypass", true)
		} else {
			if ov, ok := modDB.Override(nil, damageType+"EnergyShieldBypass"); ok {
				output.SetN(damageType+"EnergyShieldBypass", valueNum(ov))
			} else {
				output.SetN(damageType+"EnergyShieldBypass", modDB.Sum(modparser.Base, nil, damageType+"EnergyShieldBypass"))
			}
			if output.N(damageType+"EnergyShieldBypass") != 0 {
				output.SetFlag("AnyBypass", true)
			}
			if damageType == "Chaos" {
				if !modDB.Flag(nil, "ChaosNotBypassEnergyShield") {
					output.SetN(damageType+"EnergyShieldBypass", output.N(damageType+"EnergyShieldBypass")+100)
				} else {
					output.SetFlag("AnyBypass", true)
				}
			}
		}
		output.SetN(damageType+"EnergyShieldBypass", math.Max(math.Min(output.N(damageType+"EnergyShieldBypass"), 100), 0))
		output.SetN("MinimumBypass", math.Min(output.N("MinimumBypass"), output.N(damageType+"EnergyShieldBypass")))
	}

	output.SetFlag("ehpSectionAnySpecificTypes", false)
	// Mind over Matter
	output.SetFlag("OnlySharedMindOverMatter", false)
	output.SetFlag("AnySpecificMindOverMatter", false)
	output.SetN("sharedMindOverMatter", math.Min(modDB.Sum(modparser.Base, nil, "DamageTakenFromManaBeforeLife"), 100))

	// calcMoMEBPool returns the combined pool plus the mana/ES breakdown.
	calcMoMEBPool := func(lifePool, momEffect, esBypass float64) (pool, maxManaUsable, manaUsed, maxESUsable float64) {
		mana := math.Max(output.N("ManaUnreserved"), 0)
		maxMoMPool := math.Inf(1)
		if momEffect < 1 {
			maxMoMPool = lifePool/(1-momEffect) - lifePool
		}
		maxManaUsable = math.Floor(math.Min(mana, maxMoMPool))
		maxESUsable = 0
		if modDB.Flag(nil, "EnergyShieldProtectsMana") && esBypass < 1 {
			maxESUsable = math.Floor(math.Min(math.Min(
				output.N("EnergyShieldRecoveryCap"),
				maxMoMPool*(1-esBypass)),
				(lifePool+maxManaUsable)/(1-(1-esBypass)*momEffect)-(lifePool+maxManaUsable)))
		}
		manaUsed = math.Floor(math.Min(maxMoMPool-maxESUsable, maxManaUsable))
		return lifePool + manaUsed + maxESUsable, maxManaUsable, manaUsed, maxESUsable
	}

	if output.N("sharedMindOverMatter") > 0 {
		mindOverMatter := output.N("sharedMindOverMatter") / 100
		esBypass := output.N("MinimumBypass") / 100
		sharedMoMPool, _, _, _ := calcMoMEBPool(output.N("LifeRecoverable"), mindOverMatter, esBypass)
		output.SetN("sharedManaEffectiveLife", sharedMoMPool)
		hitPool, _, _, _ := calcMoMEBPool(output.N("LifeHitPool"), mindOverMatter, esBypass)
		output.SetN("sharedMoMHitPool", hitPool)
	} else {
		output.SetN("sharedManaEffectiveLife", output.N("LifeRecoverable"))
		output.SetN("sharedMoMHitPool", output.N("LifeHitPool"))
	}
	for _, damageType := range dmgTypeList {
		output.SetN(damageType+"MindOverMatter", math.Min(modDB.Sum(modparser.Base, nil, damageType+"DamageTakenFromManaBeforeLife"),
			100-output.N("sharedMindOverMatter")))
		if output.N(damageType+"MindOverMatter") > 0 ||
			(output.N(damageType+"EnergyShieldBypass") > output.N("MinimumBypass") && output.N("sharedMindOverMatter") > 0) {
			output.SetFlag("ehpSectionAnySpecificTypes", true)
			output.SetFlag("AnySpecificMindOverMatter", true)
			output.SetFlag("OnlySharedMindOverMatter", false)
			mindOverMatter := (output.N(damageType+"MindOverMatter") + output.N("sharedMindOverMatter")) / 100
			esBypass := output.N(damageType+"EnergyShieldBypass") / 100
			typedMoMPool, _, _, _ := calcMoMEBPool(output.N("LifeRecoverable"), mindOverMatter, esBypass)
			output.SetN(damageType+"ManaEffectiveLife", typedMoMPool)
			hitPool, _, _, _ := calcMoMEBPool(output.N("LifeHitPool"), mindOverMatter, esBypass)
			output.SetN(damageType+"MoMHitPool", hitPool)
		} else {
			output.SetN(damageType+"ManaEffectiveLife", output.N("sharedManaEffectiveLife"))
			output.SetN(damageType+"MoMHitPool", output.N("sharedMoMHitPool"))
		}
	}
}
