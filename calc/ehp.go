// Port of calcs.buildDefenceEstimations (CalcDefence.lua L1635-3828),
// staged. This file covers L1645-1940: chance to not be hit, the enemy
// damage input, damage-taken-as shifting, and the taken multipliers.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

var hitSourceList = []string{"Attack", "Spell"}

// configOrPlaceholder reads a per-damage-type config value, falling back
// to the placeholder default (`tonumber(configInput[k]) or
// tonumber(configPlaceholder[k]) or 0`).
func (env *Env) configOrPlaceholder(damageType string, field func(*ConfigInput) map[string]float64) float64 {
	if v, ok := field(env.ConfigInput)[damageType]; ok {
		return v
	}
	return field(env.Build.ConfigPlaceholder)[damageType]
}

// RunEHP runs the EHP stage the way the reference reaches it, and the way
// the dump records it: the player, then the minion when there is one.
func (env *Env) RunEHP() {
	env.buildDefenceEstimations(env.playerPA)
	if env.Minion != nil {
		env.buildDefenceEstimations(env.minionPA)
	}
}

func (env *Env) buildDefenceEstimations(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	damageCategoryConfig := DamageAverage
	if v := env.ConfigInput.EnemyDamageType; v != "" {
		damageCategoryConfig = v
	}

	// chance to not be hit calculations
	if damageCategoryConfig != DamageOverTime {
		worstOf := 1.0
		if v := env.ConfigInput.EHPUnluckyWorstOf; v.Set {
			worstOf = v.V
		}
		avoidAll := output.N("AvoidAllDamageFromHitsChance")
		avoidProjectiles := output.N("AvoidProjectilesChance")
		if output.Flag("specificTypeAvoidance") {
			avoidProjectiles = 0
		}
		output.SetN("MeleeNotHitChance", 100-(1-output.N("MeleeEvadeChance")/100)*(1-output.N("EffectiveAttackDodgeChance")/100)*(1-avoidAll/100)*100)
		output.SetN("ProjectileNotHitChance", 100-(1-output.N("ProjectileEvadeChance")/100)*(1-output.N("EffectiveAttackDodgeChance")/100)*(1-avoidAll/100)*(1-avoidProjectiles/100)*100)
		output.SetN("SpellNotHitChance", 100-(1-output.N("EffectiveSpellDodgeChance")/100)*(1-avoidAll/100)*100)
		output.SetN("SpellProjectileNotHitChance", 100-(1-output.N("EffectiveSpellDodgeChance")/100)*(1-avoidAll/100)*(1-avoidProjectiles/100)*100)
		output.SetN("UntypedNotHitChance", 100-(1-avoidAll/100)*100)
		output.SetN("AverageNotHitChance", (output.N("MeleeNotHitChance")+output.N("ProjectileNotHitChance")+
			output.N("SpellNotHitChance")+output.N("SpellProjectileNotHitChance"))/4)
		output.SetN("AverageEvadeChance", (output.N("MeleeEvadeChance")+output.N("ProjectileEvadeChance"))/4)
		output.SetN("ConfiguredNotHitChance", output.N(string(damageCategoryConfig)+"NotHitChance"))
		output.SetN("ConfiguredEvadeChance", output.N(string(damageCategoryConfig)+"EvadeChance"))
		// unlucky config to lower the value of block, dodge, evade etc for ehp
		if worstOf > 1 {
			output.SetN("ConfiguredNotHitChance", output.N("ConfiguredNotHitChance")/100*output.N("ConfiguredNotHitChance"))
			output.SetN("ConfiguredEvadeChance", output.N("ConfiguredEvadeChance")/100*output.N("ConfiguredEvadeChance"))
			if worstOf == 4 {
				output.SetN("ConfiguredNotHitChance", output.N("ConfiguredNotHitChance")/100*output.N("ConfiguredNotHitChance"))
				output.SetN("ConfiguredEvadeChance", output.N("ConfiguredEvadeChance")/100*output.N("ConfiguredEvadeChance"))
			}
		}
	}

	// Enemy damage input and modifications
	output.SetN("totalEnemyDamage", 0.0)
	output.SetN("totalEnemyDamageIn", 0.0)
	if damageCategoryConfig == DamageOverTime {
		for _, damageType := range dmgTypeList {
			output.SetN(damageType+"EnemyPen", 0.0)
			output.SetN(damageType+"EnemyDamageMult", Mod(enemyDB, nil, enemyDamageNames(damageType)...))
			output.SetN(damageType+"EnemyOverwhelm", 0.0)
			output.SetN(damageType+"EnemyDamage", 0.0)
		}
	} else {
		enemyCritChance := 0.0
		switch {
		case enemyDB.Flag(nil, "NeverCrit"):
			enemyCritChance = 0
		case enemyDB.Flag(nil, "AlwaysCrit"):
			enemyCritChance = 100
		default:
			base := 0.0
			if ov, ok := modDB.Override(nil, "enemyCritChance"); ok {
				base = valueNum(ov)
			} else if v := env.ConfigInput.EnemyCritChance; v.Set {
				base = v.V
			} else if v := env.Build.ConfigPlaceholder.EnemyCritChance; v.Set {
				base = v.V
			}
			scaled := base * (1 + modDB.Sum(modparser.Inc, nil, "EnemyCritChance")/100 + enemyDB.Sum(modparser.Inc, nil, "CritChance")/100) *
				(1 - output.N("ConfiguredEvadeChance")/100)
			enemyCritChance = math.Max(math.Min(scaled, 100), 0)
		}
		output.SetN("EnemyCritChance", enemyCritChance)
		enemyCritDamageBase := 0.0
		if v := env.ConfigInput.EnemyCritDamage; v.Set {
			enemyCritDamageBase = v.V
		} else if v := env.Build.ConfigPlaceholder.EnemyCritDamage; v.Set {
			enemyCritDamageBase = v.V
		}
		enemyCritDamage := math.Max(enemyCritDamageBase+enemyDB.Sum(modparser.Base, nil, "CritMultiplier"), 0)
		output.SetN("EnemyCritEffect", 1+enemyCritChance/100*(enemyCritDamage/100)*(1-output.N("CritExtraDamageReduction")/100))
		// Match all keywordFlags parameter for enemy min-max damage mods
		enemyCfg := &modstore.Cfg{KeywordFlags: keywordp(^modparser.KeywordMatchAll)}
		enemyDamageConversion := map[string]map[string]float64{}

		for _, damageType := range dmgTypeList {
			enemyDamageMult := Mod(enemyDB, nil, enemyDamageNames(damageType)...)
			enemyDamage := env.configOrPlaceholder(damageType, func(c *ConfigInput) map[string]float64 { return c.EnemyDamage })
			enemyPen := env.configOrPlaceholder(damageType, func(c *ConfigInput) map[string]float64 { return c.EnemyPen })
			enemyOverwhelm := env.configOrPlaceholder(damageType, func(c *ConfigInput) map[string]float64 { return c.EnemyOverwhelm })

			// Add min-max enemy damage from mods
			enemyDamage += (enemyDB.Sum(modparser.Base, enemyCfg, damageType+"Min") + enemyDB.Sum(modparser.Base, enemyCfg, damageType+"Max")) / 2

			// Conversion and Gain As Mods
			conversionTotal := 0.0
			if damageType == "Physical" {
				conv := map[string]float64{}
				convSkill := map[string]float64{}
				total, totalSkill := 0.0, 0.0
				for _, damageTypeTo := range dmgTypeList {
					convSkill[damageTypeTo] = enemyDB.Sum(modparser.Base, enemyCfg, damageType+"DamageSkillConvertTo"+damageTypeTo)
					conv[damageTypeTo] = enemyDB.Sum(modparser.Base, enemyCfg, damageType+"DamageConvertTo"+damageTypeTo)
					totalSkill += convSkill[damageTypeTo]
					total += conv[damageTypeTo]
				}
				// Cap the amount of conversion to 100%
				if totalSkill > 100 {
					mult := 100 / totalSkill
					totalSkill *= mult
					total = 0
					for _, damageTypeTo := range dmgTypeList {
						convSkill[damageTypeTo] *= mult
						conv[damageTypeTo] = 0
					}
				} else if total+totalSkill > 100 {
					mult := (100 - totalSkill) / total
					total *= mult
					for _, damageTypeTo := range dmgTypeList {
						conv[damageTypeTo] *= mult
					}
				}
				conversionTotal = total + totalSkill
				// Calculate the amount converted/gained as
				for _, damageTypeTo := range dmgTypeList {
					gainAsPercent := enemyDB.Sum(modparser.Base, enemyCfg, damageType+"DamageGainAs"+damageTypeTo) / 100
					conversionPercent := conv[damageTypeTo] / 100
					skillConversionPercent := convSkill[damageTypeTo] / 100
					if skillConversionPercent > 0 && damageTypeTo != "Chaos" {
						physBonus := 1 + data.MonsterPhysConversionMultiTable[int(env.EnemyLevel)-1]/100
						conversionPercent += skillConversionPercent * physBonus
					}
					if gainAsPercent > 0 || conversionPercent > 0 {
						if enemyDamageConversion[damageTypeTo] == nil {
							enemyDamageConversion[damageTypeTo] = map[string]float64{}
						}
						enemyDamageConversion[damageTypeTo][damageType] = enemyDamage*gainAsPercent + enemyDamage*conversionPercent
					}
				}
			}

			enemyOverwhelm += enemyDB.Sum(modparser.Base, nil, "PhysicalOverwhelm") + modDB.Sum(modparser.Base, nil, "EnemyPhysicalOverwhelm")

			output.SetN(damageType+"EnemyPen", enemyPen)
			output.SetN(damageType+"EnemyDamageMult", enemyDamageMult)
			output.SetN(damageType+"EnemyOverwhelm", enemyOverwhelm)
			output.SetN("totalEnemyDamageIn", output.N("totalEnemyDamageIn")+enemyDamage)
			output.SetN(damageType+"EnemyDamage", enemyDamage*(1-conversionTotal/100)*enemyDamageMult*output.N("EnemyCritEffect"))
			if conv := enemyDamageConversion[damageType]; conv != nil {
				// sorted: the reference iterates pairs() over damage-type keys
				for _, damageTypeFrom := range dmgTypeList {
					dmg, ok := conv[damageTypeFrom]
					if !ok {
						continue
					}
					mult := Mod(enemyDB, nil, enemyConvertedDamageNames(damageType, damageTypeFrom)...)
					output.SetN(damageType+"EnemyDamage", output.N(damageType+"EnemyDamage")+dmg*mult*output.N("EnemyCritEffect"))
				}
			}
			output.SetN("totalEnemyDamage", output.N("totalEnemyDamage")+output.N(damageType+"EnemyDamage"))
		}
	}

	env.ehpDamageTakenAs(actor)
	env.ehpIncomingHit(actor, damageCategoryConfig)
	env.ehpStun(actor, damageCategoryConfig)
	env.ehpPools(actor)
	env.ehpGuard(actor, damageCategoryConfig)
	env.ehpHitCounts(actor, damageCategoryConfig)
	env.ehpRecoup(actor, damageCategoryConfig)
	env.ehpMaxHit(actor)
	env.ehpDegens(actor, damageCategoryConfig)
	if env.ConfigInput.PvpScaling {
		output := actor.output
		pvpTvalue := output.N("enemySkillTime")
		pvpMultiplier := 1.0
		if v := env.ConfigInput.EnemyMultiplierPvpDamage; v.Set {
			pvpMultiplier = v.V / 100
		}
		pvpNonElemental1 := data.Misc.PvpNonElemental1
		pvpNonElemental2 := data.Misc.PvpNonElemental2
		pvpElemental1 := data.Misc.PvpElemental1
		pvpElemental2 := data.Misc.PvpElemental2

		percentageNonElemental := (output.N("PhysicalTakenHit") + output.N("ChaosTakenHit")) / output.N("totalTakenHit")
		percentageElemental := 1 - percentageNonElemental
		portionNonElemental := math.Pow(output.N("totalTakenHit")/pvpTvalue/pvpNonElemental2, pvpNonElemental1) *
			pvpTvalue * pvpNonElemental2 * percentageNonElemental
		portionElemental := math.Pow(output.N("totalTakenHit")/pvpTvalue/pvpElemental2, pvpElemental1) *
			pvpTvalue * pvpElemental2 * percentageElemental
		output.SetN("PvPTotalTakenHit", (portionNonElemental+portionElemental)*pvpMultiplier)
	}
}

// enemyDamageNames is the reference's `"Damage", <type>.."Damage",
// isElemental[type] and "ElementalDamage" or nil` vararg.
func enemyDamageNames(damageType string) []string {
	names := []string{"Damage", damageType + "Damage"}
	if isElementalRes[damageType] {
		names = append(names, "ElementalDamage")
	}
	return names
}

// enemyConvertedDamageNames adds the source type's names for converted
// enemy damage.
func enemyConvertedDamageNames(damageType, damageTypeFrom string) []string {
	names := []string{"Damage", damageType + "Damage", damageTypeFrom + "Damage"}
	if isElementalRes[damageType] {
		names = append(names, "ElementalDamage")
	}
	if isElementalRes[damageTypeFrom] {
		names = append(names, "ElementalDamage")
	}
	return names
}

// ehpDamageTakenAs ports the damage-shift tables and taken multipliers
// (L1802-1940).
func (env *Env) ehpDamageTakenAs(actor *performActor) {
	modDB := actor.db
	output := actor.output

	actor.damageShiftTable = map[string]map[string]float64{}
	actor.damageOverTimeShiftTable = map[string]map[string]float64{}
	for _, damageType := range dmgTypeList {
		shiftTable := map[string]float64{}
		dotShiftTable := map[string]float64{}
		destTotal, dotDestinationTotal := 0.0, 0.0
		for _, destType := range dmgTypeList {
			if destType != damageType {
				dotNames := []string{damageType + "DamageTakenAs" + destType}
				hitNames := []string{damageType + "DamageFromHitsTakenAs" + destType}
				if isElementalRes[damageType] {
					dotNames = append(dotNames, "ElementalDamageTakenAs"+destType)
					hitNames = append(hitNames, "ElementalDamageFromHitsTakenAs"+destType)
				}
				dotShiftTable[destType] = modDB.Sum(modparser.Base, nil, dotNames...)
				dotDestinationTotal += dotShiftTable[destType]
				shiftTable[destType] = dotShiftTable[destType] + modDB.Sum(modparser.Base, nil, hitNames...)
				destTotal += shiftTable[destType]
			}
		}
		dotShiftTable[damageType] = math.Max(100-dotDestinationTotal, 0)
		actor.damageOverTimeShiftTable[damageType] = dotShiftTable
		shiftTable[damageType] = math.Max(100-destTotal, 0)
		actor.damageShiftTable[damageType] = shiftTable

		// add same type damage
		output.SetN(damageType+"TakenDamage", output.N(damageType+"EnemyDamage")*shiftTable[damageType]/100)
	}
	// converted damage types
	for _, damageType := range dmgTypeList {
		for _, damageConvertedType := range dmgTypeList {
			if damageType != damageConvertedType {
				damage := output.N(damageType+"EnemyDamage") * actor.damageShiftTable[damageType][damageConvertedType] / 100
				output.SetN(damageConvertedType+"TakenDamage", output.N(damageConvertedType+"TakenDamage")+damage)
			}
		}
	}
	// total
	output.SetN("totalTakenDamage", 0.0)
	for _, damageType := range dmgTypeList {
		output.SetN("totalTakenDamage", output.N("totalTakenDamage")+output.N(damageType+"TakenDamage"))
	}

	// Damage taken multipliers/Degen calculations
	output.SetFlag("AnyTakenReflect", false)
	for _, damageType := range dmgTypeList {
		baseTakenInc := modDB.Sum(modparser.Inc, nil, "DamageTaken", damageType+"DamageTaken")
		baseTakenMore := modDB.More(nil, "DamageTaken", damageType+"DamageTaken")
		if isElementalRes[damageType] {
			baseTakenInc += modDB.Sum(modparser.Inc, nil, "ElementalDamageTaken")
			baseTakenMore *= modDB.More(nil, "ElementalDamageTaken")
		}
		{ // Hit
			takenInc := baseTakenInc + modDB.Sum(modparser.Inc, nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			if isElementalRes[damageType] {
				takenInc += modDB.Sum(modparser.Inc, nil, "ElementalDamageTakenWhenHit")
				takenMore *= modDB.More(nil, "ElementalDamageTakenWhenHit")
			}
			output.SetN(damageType+"TakenHitMult", math.Max((1+takenInc/100)*takenMore, 0))

			for _, hitType := range hitSourceList {
				baseTakenIncType := takenInc + modDB.Sum(modparser.Inc, nil, hitType+"DamageTaken")
				baseTakenMoreType := takenMore * modDB.More(nil, hitType+"DamageTaken")
				output.SetN(hitType+"TakenHitMult", math.Max((1+baseTakenIncType/100)*baseTakenMoreType, 0))
				output.SetN(damageType+hitType+"TakenHitMult", output.N(hitType+"TakenHitMult"))
			}
			// Reflect
			takenInc += modDB.Sum(modparser.Inc, nil, "ReflectedDamageTaken", damageType+"ReflectedDamageTaken")
			takenMore *= modDB.More(nil, "ReflectedDamageTaken", damageType+"ReflectedDamageTaken")
			if isElementalRes[damageType] {
				takenInc += modDB.Sum(modparser.Inc, nil, "ElementalReflectedDamageTaken")
				takenMore *= modDB.More(nil, "ElementalReflectedDamageTaken")
			}
			output.SetN(damageType+"TakenReflect", math.Max((1+takenInc/100)*takenMore, 0))
			// The reference assigns false in both branches here
			// ("this needs a rework as well"), so AnyTakenReflect never
			// becomes true
			if output.N(damageType+"TakenReflect") != output.N(damageType+"TakenHitMult") {
				output.SetFlag("AnyTakenReflect", false)
			}
		}
		{ // Dot
			takenInc := baseTakenInc + modDB.Sum(modparser.Inc, nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
			if isElementalRes[damageType] {
				takenInc += modDB.Sum(modparser.Inc, nil, "ElementalDamageTakenOverTime")
				takenMore *= modDB.More(nil, "ElementalDamageTakenOverTime")
			}
			resist := 0.0
			if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
				if v, ok := output[damageType+"ResistOverTime"]; ok && v.Truthy() {
					resist = v.Num()
				} else {
					resist = output.N(damageType + "Resist")
				}
			}
			reduction := 0.0
			if !modDB.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
				reduction = output.N("Base" + damageType + "DamageReduction")
			}
			output.SetN(damageType+"TakenDotMult", math.Max((1-resist/100)*(1-reduction/100)*(1+takenInc/100)*takenMore, 0))
		}
	}
}
