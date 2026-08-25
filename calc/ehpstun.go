// CalcDefence.lua L2109-2212: stun threshold, avoidance, duration and
// self-stun chance.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
)

func (env *Env) ehpStun(actor *performActor, damageCategoryConfig string) {
	modDB := actor.db
	output := actor.output

	stunThresholdBase := 0.0
	switch {
	case modDB.Flag(nil, "StunThresholdBasedOnEnergyShieldInsteadOfLife"):
		stunThresholdBase = outNum(output, "EnergyShield") * modDB.Sum("BASE", nil, "StunThresholdEnergyShieldPercent") / 100
	case modDB.Flag(nil, "StunThresholdBasedOnManaInsteadOfLife"):
		stunThresholdBase = outNum(output, "Mana") * modDB.Sum("BASE", nil, "StunThresholdManaPercent") / 100
	case modDB.Flag(nil, "ChaosInoculation"):
		stunThresholdBase = modDB.Sum("BASE", nil, "Life")
	default:
		stunThresholdBase = outNum(output, "Life")
	}
	if modDB.Flag(nil, "AddESToStunThreshold") {
		esMult := modDB.Sum("BASE", nil, "ESToStunThresholdPercent")
		stunThresholdBase += outNum(output, "EnergyShield") * esMult / 100
	}
	stunThresholdMod := 1 + modDB.Sum("INC", nil, "StunThreshold")/100
	output["StunThreshold"] = stunThresholdBase * stunThresholdMod

	notAvoidChance := 0.0
	if !modDB.Flag(nil, "StunImmune") {
		notAvoidChance = 100 - math.Min(modDB.Sum("BASE", nil, "AvoidStun"), 100)
	}
	// Having any energy shield when the hit occurs grants 50% chance to
	// avoid stun; PoB applies it only when ES exceeds incoming damage.
	if outNum(output, "EnergyShield") > outNum(output, "totalTakenHit") && !env.ModDB.Flag(nil, "EnergyShieldProtectsMana") {
		notAvoidChance = notAvoidChance * 0.5
	}
	output["StunAvoidChance"] = 100 - notAvoidChance

	if outNum(output, "StunAvoidChance") >= 100 {
		output["StunDuration"] = 0.0
		output["BlockDuration"] = 0.0
	} else {
		stunDuration := 1 + modDB.Sum("INC", nil, "StunDuration")/100
		baseStunDuration := data.Misc.StunBaseDuration
		stunRecovery := 1 + modDB.Sum("INC", nil, "StunRecovery")/100
		stunAndBlockRecovery := 1 + modDB.Sum("INC", nil, "StunRecovery", "BlockRecovery")/100
		output["StunDuration"] = math.Ceil(baseStunDuration*stunDuration/stunRecovery*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
		output["BlockDuration"] = math.Ceil(baseStunDuration*stunDuration/stunAndBlockRecovery*data.Misc.ServerTickRate) / data.Misc.ServerTickRate
	}
	output["InterruptStunAvoidChance"] = math.Min(modDB.Sum("BASE", nil, "AvoidInterruptStun"), 100)

	effectiveEnemyDamage := outNum(output, "totalTakenHit") + outNum(output, "PhysicalTakenHit")*0.25
	// #EVAL the reference's second branch is `elseif ~= "Melee"`, which only
	// runs when the category IS "Average", so the Melee multiplier below can
	// never apply to a non-melee category.
	if damageCategoryConfig != "Average" {
		effectiveEnemyDamage = effectiveEnemyDamage * (1 + data.Misc.StunNotMeleeDamageMult*3) / 4
	} else if damageCategoryConfig != "Melee" {
		effectiveEnemyDamage = effectiveEnemyDamage * data.Misc.StunNotMeleeDamageMult
	}
	baseStunChance := math.Min(data.Misc.StunBaseMult*effectiveEnemyDamage/outNum(output, "StunThreshold"), 100)
	chance := 0.0
	if baseStunChance > data.Misc.MinStunChanceNeeded {
		chance = baseStunChance
	}
	output["SelfStunChance"] = chance * notAvoidChance / 100
}
