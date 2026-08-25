// CalcDefence.lua L1942-2108: the incoming hit damage multipliers.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

func (env *Env) ehpIncomingHit(actor *performActor, damageCategoryConfig string) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	output["totalTakenHit"] = 0.0

	impaleFlags := int64(0)
	if damageCategoryConfig == "Melee" || damageCategoryConfig == "Projectile" || damageCategoryConfig == "Average" {
		impaleFlags = modparser.ModFlag.Attack
	}
	impaleMult := 1.0
	if damageCategoryConfig == "Average" {
		impaleMult = 0.5
	}
	enemyImpaleChance := enemyDB.Sum("BASE", &modstore.Cfg{Flags: i64p(impaleFlags), KeywordFlags: i64p(0)}, "ImpaleChance") *
		impaleMult * (1 - outNum(output, "ImpaleAvoidChance"))

	for _, damageType := range dmgTypeList {
		// Calculate incoming damage multiplier
		resist := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
			if v, ok := output[damageType+"ResistWhenHit"]; ok && truthy(v) {
				resist = anyNum(v)
			} else {
				resist = outNum(output, damageType+"Resist")
			}
		}
		reduction := 0.0
		if !modDB.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
			if v, ok := output["Base"+damageType+"DamageReductionWhenHit"]; ok && truthy(v) {
				reduction = anyNum(v)
			} else {
				reduction = outNum(output, "Base"+damageType+"DamageReduction")
			}
		}
		enemyPen := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance", "EnemyCannotPen"+damageType+"Resistance") {
			enemyPen = outNum(output, damageType+"EnemyPen")
		}
		enemyOverwhelm := 0.0
		if !modDB.Flag(nil, "SelfIgnore"+damageType+"DamageReduction") {
			enemyOverwhelm = outNum(output, damageType+"EnemyOverwhelm")
		}
		damage := outNum(output, damageType+"TakenDamage")
		impaleDamage := 0.0
		if enemyImpaleChance > 0 && damageType == "Physical" {
			impaleDamage = damage * data.Misc.ImpaleStoredDamageBase
		}
		armourReduct := 0.0
		impaleArmourReduct := 0.0
		percentOfArmourApplies := 0.0
		if !modDB.Flag(nil, "ArmourDoesNotApplyTo"+damageType+"DamageTaken") {
			percentOfArmourApplies = modDB.Sum("BASE", nil, "ArmourAppliesTo"+damageType+"DamageTaken")
		}
		percentOfArmourApplies = math.Min(percentOfArmourApplies, 100)
		effectiveAppliedArmour := (outNum(output, "Armour") * percentOfArmourApplies / 100) * (1 + outNum(output, "ArmourDefense"))
		physicalReductionBasedOnWard := damageType == "Physical" && modDB.Flag(nil, "PhysicalReductionBasedOnWard")
		if physicalReductionBasedOnWard {
			multiplier := anyNum(modDB.Override(nil, "PhysicalReductionBasedOnWardPercent")) / 100
			effectiveAppliedArmour = outNum(output, "Ward") * multiplier
		}
		resMult := 1 - (resist-enemyPen)/100
		takenFlat := modDB.Sum("BASE", nil, "DamageTaken", damageType+"DamageTaken", "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
		switch damageCategoryConfig {
		case "Melee", "Projectile":
			takenFlat += modDB.Sum("BASE", nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks",
				damageType+"DamageTakenFrom"+damageCategoryConfig+"Attacks")
		case "Spell", "SpellProjectile":
			takenFlat += modDB.Sum("BASE", nil, "DamageTakenFromSpells", damageType+"DamageTakenFromSpells",
				damageType+"DamageTakenFromSpellProjectiles")
		case "Average":
			takenFlat += modDB.Sum("BASE", nil, "DamageTakenFromAttacks", damageType+"DamageTakenFromAttacks")/2 +
				modDB.Sum("BASE", nil, damageType+"DamageTakenFromProjectileAttacks")/4 +
				modDB.Sum("BASE", nil, "DamageTakenFromSpells", damageType+"DamageTakenFromSpells")/2 +
				modDB.Sum("BASE", nil, "DamageTakenFromSpellProjectiles", damageType+"DamageTakenFromSpellProjectiles")/4
		}
		output[damageType+"takenFlat"] = takenFlat
		if percentOfArmourApplies > 0 || physicalReductionBasedOnWard {
			armourReduct = armourReduction(effectiveAppliedArmour, damage*resMult)
			armourReduct = math.Min(outNum(output, "DamageReductionMax"), armourReduct)
			if impaleDamage > 0 {
				impaleArmourReduct = math.Min(outNum(output, "DamageReductionMax"), armourReduction(effectiveAppliedArmour, impaleDamage*resMult))
			}
		}
		totalReduct := math.Min(outNum(output, "DamageReductionMax"), armourReduct+reduction)
		reductMult := 1 - math.Max(math.Min(outNum(output, "DamageReductionMax"), totalReduct-enemyOverwhelm), 0)/100
		output[damageType+"DamageReduction"] = 100 - reductMult*100
		if impaleDamage > 0 {
			impaleDamage = impaleDamage * resMult * (1 - math.Max(math.Min(outNum(output, "DamageReductionMax"),
				math.Min(outNum(output, "DamageReductionMax"), impaleArmourReduct+reduction)-enemyOverwhelm), 0)/100)
			impaleDamage = impaleDamage * enemyImpaleChance / 100 * 5 * outNum(output, damageType+"TakenReflect")
		}
		takenMult := outNum(output, damageType+"TakenHitMult")
		spellSuppressMult := 1.0
		switch damageCategoryConfig {
		case "Melee", "Projectile":
			takenMult = outNum(output, damageType+"AttackTakenHitMult")
		case "Spell", "SpellProjectile":
			takenMult = outNum(output, damageType+"SpellTakenHitMult")
			if outNum(output, "EffectiveSpellSuppressionChance") == 100 {
				spellSuppressMult = 1 - outNum(output, "SpellSuppressionEffect")/100
			}
		case "Average":
			takenMult = (outNum(output, damageType+"SpellTakenHitMult") + outNum(output, damageType+"AttackTakenHitMult")) / 2
			if outNum(output, "EffectiveSpellSuppressionChance") == 100 {
				spellSuppressMult = 1 - outNum(output, "SpellSuppressionEffect")/100/2
			}
		}
		output[damageType+"EffectiveAppliedArmour"] = effectiveAppliedArmour
		output[damageType+"ResistTakenHitMulti"] = resMult
		afterReductionMulti := takenMult * spellSuppressMult
		output[damageType+"AfterReductionTakenHitMulti"] = afterReductionMulti
		baseMult := resMult * reductMult
		output[damageType+"BaseTakenHitMult"] = baseMult * afterReductionMulti
		takenMultReflect := outNum(output, damageType+"TakenReflect")
		finalReflect := baseMult * takenMultReflect
		output[damageType+"TakenHit"] = math.Max(damage*baseMult+takenFlat, 0)*takenMult*spellSuppressMult + impaleDamage
		if damage > 0 {
			output[damageType+"TakenHitMult"] = outNum(output, damageType+"TakenHit") / damage
		} else {
			output[damageType+"TakenHitMult"] = 0.0
		}
		output["totalTakenHit"] = outNum(output, "totalTakenHit") + outNum(output, damageType+"TakenHit")
		// AnyTakenReflect is always false in the reference (see ehp.go)
		if truthy(output["AnyTakenReflect"]) {
			output[damageType+"TakenReflectMult"] = finalReflect
		}
	}
}
