// CalcDefence.lua L2109-2212: stun threshold, avoidance, duration and
// self-stun chance.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"
)

func (env *Env) ehpStun(actor *performActor, damageCategoryConfig DamageCategory) {
	modDB := actor.db
	output := actor.output

	stunThresholdBase := 0.0
	switch {
	case modDB.Flag(nil, "StunThresholdBasedOnEnergyShieldInsteadOfLife"):
		stunThresholdBase = output.N("EnergyShield") * modDB.Sum(modparser.Base, nil, "StunThresholdEnergyShieldPercent") / 100
	case modDB.Flag(nil, "StunThresholdBasedOnManaInsteadOfLife"):
		stunThresholdBase = output.N("Mana") * modDB.Sum(modparser.Base, nil, "StunThresholdManaPercent") / 100
	case modDB.Flag(nil, "ChaosInoculation"):
		stunThresholdBase = modDB.Sum(modparser.Base, nil, "Life")
	default:
		stunThresholdBase = output.N("Life")
	}
	if modDB.Flag(nil, "AddESToStunThreshold") {
		esMult := modDB.Sum(modparser.Base, nil, "ESToStunThresholdPercent")
		stunThresholdBase += output.N("EnergyShield") * esMult / 100
	}
	stunThresholdMod := 1 + modDB.Sum(modparser.Inc, nil, "StunThreshold")/100
	output.SetN("StunThreshold", stunThresholdBase*stunThresholdMod)

	notAvoidChance := 0.0
	if !modDB.Flag(nil, "StunImmune") {
		notAvoidChance = 100 - math.Min(modDB.Sum(modparser.Base, nil, "AvoidStun"), 100)
	}
	// Having any energy shield when the hit occurs grants 50% chance to
	// avoid stun; PoB applies it only when ES exceeds incoming damage.
	if output.N("EnergyShield") > output.N("totalTakenHit") && !env.ModDB.Flag(nil, "EnergyShieldProtectsMana") {
		notAvoidChance = notAvoidChance * 0.5
	}
	output.SetN("StunAvoidChance", 100-notAvoidChance)

	if output.N("StunAvoidChance") >= 100 {
		output.SetN("StunDuration", 0.0)
		output.SetN("BlockDuration", 0.0)
	} else {
		stunDuration := 1 + modDB.Sum(modparser.Inc, nil, "StunDuration")/100
		baseStunDuration := data.Misc.StunBaseDuration
		stunRecovery := 1 + modDB.Sum(modparser.Inc, nil, "StunRecovery")/100
		stunAndBlockRecovery := 1 + modDB.Sum(modparser.Inc, nil, "StunRecovery", "BlockRecovery")/100
		output.SetN("StunDuration", math.Ceil(baseStunDuration*stunDuration/stunRecovery*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
		output.SetN("BlockDuration", math.Ceil(baseStunDuration*stunDuration/stunAndBlockRecovery*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
	}
	output.SetN("InterruptStunAvoidChance", math.Min(modDB.Sum(modparser.Base, nil, "AvoidInterruptStun"), 100))

	effectiveEnemyDamage := output.N("totalTakenHit") + output.N("PhysicalTakenHit")*0.25
	// #EVAL the reference's second branch is `elseif ~= "Melee"`, which only
	// runs when the category IS "Average", so the Melee multiplier below can
	// never apply to a non-melee category.
	if damageCategoryConfig != DamageAverage {
		effectiveEnemyDamage = effectiveEnemyDamage * (1 + data.Misc.StunNotMeleeDamageMult*3) / 4
	} else if damageCategoryConfig != DamageMelee {
		effectiveEnemyDamage = effectiveEnemyDamage * data.Misc.StunNotMeleeDamageMult
	}
	baseStunChance := math.Min(data.Misc.StunBaseMult*effectiveEnemyDamage/output.N("StunThreshold"), 100)
	chance := 0.0
	if baseStunChance > data.Misc.MinStunChanceNeeded {
		chance = baseStunChance
	}
	output.SetN("SelfStunChance", chance*notAvoidChance/100)
}
