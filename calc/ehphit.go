// CalcDefence.lua L1942-2108: the incoming hit damage multipliers.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

func (env *Env) ehpIncomingHit(actor *performActor, damageCategoryConfig DamageCategory) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	output.SetN("totalTakenHit", 0.0)

	impaleFlags := modparser.FlagNone
	if damageCategoryConfig == DamageMelee || damageCategoryConfig == DamageProjectile || damageCategoryConfig == DamageAverage {
		impaleFlags = modparser.FlagAttack
	}
	impaleMult := 1.0
	if damageCategoryConfig == DamageAverage {
		impaleMult = 0.5
	}
	enemyImpaleChance := enemyDB.Sum(modparser.Base, &modstore.Cfg{Flags: flagp(impaleFlags), KeywordFlags: keywordp(0)}, "ImpaleChance") *
		impaleMult * (1 - output.N("ImpaleAvoidChance"))

	for _, damageType := range dmgTypeList {
		// Calculate incoming damage multiplier
		resist := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
			if v, ok := output[damageType+"ResistWhenHit"]; ok && v.Truthy() {
				resist = v.Num()
			} else {
				resist = output.N(damageType + "Resist")
			}
		}
		reduction := 0.0
		if !modDB.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
			if v, ok := output["Base"+damageType+"DamageReductionWhenHit"]; ok && v.Truthy() {
				reduction = v.Num()
			} else {
				reduction = output.N("Base" + damageType + "DamageReduction")
			}
		}
		enemyPen := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance", "EnemyCannotPen"+damageType+"Resistance") {
			enemyPen = output.N(damageType + "EnemyPen")
		}
		enemyOverwhelm := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"DamageReduction") {
			enemyOverwhelm = output.N(damageType + "EnemyOverwhelm")
		}
		damage := output.N(damageType + "TakenDamage")
		impaleDamage := 0.0
		if enemyImpaleChance > 0 && damageType == "Physical" {
			impaleDamage = damage * data.Misc.ImpaleStoredDamageBase
		}
		armourReduct := 0.0
		impaleArmourReduct := 0.0
		percentOfArmourApplies := 0.0
		if !modDB.Flag(nil, "ArmourDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfArmourApplies = modDB.Sum(modparser.Base, nil, "ArmourAppliesTo"+damageType+"DamageTaken")
		}
		percentOfArmourApplies = math.Min(percentOfArmourApplies, 100)
		effectiveAppliedArmour := (output.N("Armour") * percentOfArmourApplies / 100) * (1 + output.N("ArmourDefense"))
		physicalReductionBasedOnWard := damageType == "Physical" && modDB.Flag(nil, "PhysicalReductionBasedOnWard")
		if physicalReductionBasedOnWard {
			multiplier := overrideNum(modDB, nil, "PhysicalReductionBasedOnWardPercent") / 100
			effectiveAppliedArmour = output.N("Ward") * multiplier
		}
		resMult := 1 - (resist-enemyPen)/100
		takenFlat := modDB.Sum(modparser.Base, nil, "DamageTaken", damageType+"DamageTaken", "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
		switch damageCategoryConfig {
		case DamageMelee, DamageProjectile:
			takenFlat += modDB.Sum(modparser.Base, nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks",
				damageType+"DamageTakenFrom"+string(damageCategoryConfig)+"Attacks")
		case DamageSpell, DamageSpellProjectile:
			takenFlat += modDB.Sum(modparser.Base, nil, "DamageTakenFromSpells", damageType+"DamageTakenFromSpells",
				damageType+"DamageTakenFromSpellProjectiles")
		case DamageAverage:
			takenFlat += modDB.Sum(modparser.Base, nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks")/2 +
				modDB.Sum(modparser.Base, nil, damageType+"DamageTakenFromProjectileAttacks")/4 +
				modDB.Sum(modparser.Base, nil, "DamageTakenFromSpells", damageType+"DamageTakenFromSpells")/2 +
				modDB.Sum(modparser.Base, nil, "DamageTakenFromSpellProjectiles", damageType+"DamageTakenFromSpellProjectiles")/4
		}
		output.SetN(damageType+"takenFlat", takenFlat)
		if percentOfArmourApplies > 0 || physicalReductionBasedOnWard {
			armourReduct = armourReduction(effectiveAppliedArmour, damage*resMult)
			armourReduct = math.Min(output.N("DamageReductionMax"), armourReduct)
			if impaleDamage > 0 {
				impaleArmourReduct = math.Min(output.N("DamageReductionMax"), armourReduction(effectiveAppliedArmour, impaleDamage*resMult))
			}
		}
		totalReduct := math.Min(output.N("DamageReductionMax"), armourReduct+reduction)
		reductMult := 1 - math.Max(math.Min(output.N("DamageReductionMax"), totalReduct-enemyOverwhelm), 0)/100
		output.SetN(damageType+"DamageReduction", 100-reductMult*100)
		if impaleDamage > 0 {
			impaleDamage = impaleDamage * resMult * (1 - math.Max(math.Min(output.N("DamageReductionMax"),
				math.Min(output.N("DamageReductionMax"), impaleArmourReduct+reduction)-enemyOverwhelm), 0)/100)
			impaleDamage = impaleDamage * enemyImpaleChance / 100 * 5 * output.N(damageType+"TakenReflect")
		}
		takenMult := output.N(damageType + "TakenHitMult")
		spellSuppressMult := 1.0
		switch damageCategoryConfig {
		case DamageMelee, DamageProjectile:
			takenMult = output.N(damageType + "AttackTakenHitMult")
		case DamageSpell, DamageSpellProjectile:
			takenMult = output.N(damageType + "SpellTakenHitMult")
			if output.N("EffectiveSpellSuppressionChance") == 100 {
				spellSuppressMult = 1 - output.N("SpellSuppressionEffect")/100
			}
		case DamageAverage:
			takenMult = (output.N(damageType+"SpellTakenHitMult") + output.N(damageType+"AttackTakenHitMult")) / 2
			if output.N("EffectiveSpellSuppressionChance") == 100 {
				spellSuppressMult = 1 - output.N("SpellSuppressionEffect")/100/2
			}
		}
		output.SetN(damageType+"EffectiveAppliedArmour", effectiveAppliedArmour)
		output.SetN(damageType+"ResistTakenHitMulti", resMult)
		afterReductionMulti := takenMult * spellSuppressMult
		output.SetN(damageType+"AfterReductionTakenHitMulti", afterReductionMulti)
		baseMult := resMult * reductMult
		output.SetN(damageType+"BaseTakenHitMult", baseMult*afterReductionMulti)
		takenMultReflect := output.N(damageType + "TakenReflect")
		finalReflect := baseMult * takenMultReflect
		output.SetN(damageType+"TakenHit", math.Max(damage*baseMult+takenFlat, 0)*takenMult*spellSuppressMult+impaleDamage)
		if damage > 0 {
			output.SetN(damageType+"TakenHitMult", output.N(damageType+"TakenHit")/damage)
		} else {
			output.SetN(damageType+"TakenHitMult", 0.0)
		}
		output.SetN("totalTakenHit", output.N("totalTakenHit")+output.N(damageType+"TakenHit"))
		// AnyTakenReflect is always false in the reference (see ehp.go)
		if output.Flag("AnyTakenReflect") {
			output.SetN(damageType+"TakenReflectMult", finalReflect)
		}
	}
}
