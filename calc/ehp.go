// Port of calcs.buildDefenceEstimations (CalcDefence.lua L1635-3828),
// staged. This file covers L1645-1940: chance to not be hit, the enemy
// damage input, damage-taken-as shifting, and the taken multipliers.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strconv"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

var hitSourceList = []string{"Attack", "Spell"}

// tonum ports Lua tonumber() over the config bags: strings are parsed,
// numbers pass through, anything else (including absent) yields nil.
func tonum(v any) *float64 {
	switch t := v.(type) {
	case float64:
		n := t
		return &n
	case int64:
		n := float64(t)
		return &n
	case string:
		if n, err := strconv.ParseFloat(t, 64); err == nil {
			return &n
		}
	}
	return nil
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

	damageCategoryConfig := "Average"
	if v := env.ConfigInput["enemyDamageType"]; truthy(v) {
		damageCategoryConfig = str(v)
	}

	// chance to not be hit calculations
	if damageCategoryConfig != "DamageOverTime" {
		worstOf := 1.0
		if v := env.ConfigInput["EHPUnluckyWorstOf"]; truthy(v) {
			worstOf = anyNum(v)
		}
		avoidAll := outNum(output, "AvoidAllDamageFromHitsChance")
		avoidProjectiles := outNum(output, "AvoidProjectilesChance")
		if truthy(output["specificTypeAvoidance"]) {
			avoidProjectiles = 0
		}
		output["MeleeNotHitChance"] = 100 - (1-outNum(output, "MeleeEvadeChance")/100)*(1-outNum(output, "EffectiveAttackDodgeChance")/100)*(1-avoidAll/100)*100
		output["ProjectileNotHitChance"] = 100 - (1-outNum(output, "ProjectileEvadeChance")/100)*(1-outNum(output, "EffectiveAttackDodgeChance")/100)*(1-avoidAll/100)*(1-avoidProjectiles/100)*100
		output["SpellNotHitChance"] = 100 - (1-outNum(output, "EffectiveSpellDodgeChance")/100)*(1-avoidAll/100)*100
		output["SpellProjectileNotHitChance"] = 100 - (1-outNum(output, "EffectiveSpellDodgeChance")/100)*(1-avoidAll/100)*(1-avoidProjectiles/100)*100
		output["UntypedNotHitChance"] = 100 - (1-avoidAll/100)*100
		output["AverageNotHitChance"] = (outNum(output, "MeleeNotHitChance") + outNum(output, "ProjectileNotHitChance") +
			outNum(output, "SpellNotHitChance") + outNum(output, "SpellProjectileNotHitChance")) / 4
		output["AverageEvadeChance"] = (outNum(output, "MeleeEvadeChance") + outNum(output, "ProjectileEvadeChance")) / 4
		output["ConfiguredNotHitChance"] = outNum(output, damageCategoryConfig+"NotHitChance")
		output["ConfiguredEvadeChance"] = outNum(output, damageCategoryConfig+"EvadeChance")
		// unlucky config to lower the value of block, dodge, evade etc for ehp
		if worstOf > 1 {
			output["ConfiguredNotHitChance"] = outNum(output, "ConfiguredNotHitChance") / 100 * outNum(output, "ConfiguredNotHitChance")
			output["ConfiguredEvadeChance"] = outNum(output, "ConfiguredEvadeChance") / 100 * outNum(output, "ConfiguredEvadeChance")
			if worstOf == 4 {
				output["ConfiguredNotHitChance"] = outNum(output, "ConfiguredNotHitChance") / 100 * outNum(output, "ConfiguredNotHitChance")
				output["ConfiguredEvadeChance"] = outNum(output, "ConfiguredEvadeChance") / 100 * outNum(output, "ConfiguredEvadeChance")
			}
		}
	}

	// Enemy damage input and modifications
	output["totalEnemyDamage"] = 0.0
	output["totalEnemyDamageIn"] = 0.0
	if damageCategoryConfig == "DamageOverTime" {
		for _, damageType := range dmgTypeList {
			output[damageType+"EnemyPen"] = 0.0
			output[damageType+"EnemyDamageMult"] = Mod(enemyDB, nil, enemyDamageNames(damageType)...)
			output[damageType+"EnemyOverwhelm"] = 0.0
			output[damageType+"EnemyDamage"] = 0.0
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
			if ov := modDB.Override(nil, "enemyCritChance"); truthy(ov) {
				base = anyNum(ov)
			} else if v := env.ConfigInput["enemyCritChance"]; truthy(v) {
				base = anyNum(v)
			} else if v := env.Build.ConfigPlaceholder["enemyCritChance"]; truthy(v) {
				base = anyNum(v)
			}
			scaled := base * (1 + modDB.Sum("INC", nil, "EnemyCritChance")/100 + enemyDB.Sum("INC", nil, "CritChance")/100) *
				(1 - outNum(output, "ConfiguredEvadeChance")/100)
			enemyCritChance = math.Max(math.Min(scaled, 100), 0)
		}
		output["EnemyCritChance"] = enemyCritChance
		enemyCritDamageBase := 0.0
		if v := env.ConfigInput["enemyCritDamage"]; truthy(v) {
			enemyCritDamageBase = anyNum(v)
		} else if v := env.Build.ConfigPlaceholder["enemyCritDamage"]; truthy(v) {
			enemyCritDamageBase = anyNum(v)
		}
		enemyCritDamage := math.Max(enemyCritDamageBase+enemyDB.Sum("BASE", nil, "CritMultiplier"), 0)
		output["EnemyCritEffect"] = 1 + enemyCritChance/100*(enemyCritDamage/100)*(1-outNum(output, "CritExtraDamageReduction")/100)
		// Match all keywordFlags parameter for enemy min-max damage mods
		enemyCfg := &modstore.Cfg{KeywordFlags: i64p(^modparser.KeywordFlag.MatchAll)}
		enemyDamageConversion := map[string]map[string]float64{}

		for _, damageType := range dmgTypeList {
			enemyDamageMult := Mod(enemyDB, nil, enemyDamageNames(damageType)...)
			var enemyDamage float64
			if p := tonum(env.ConfigInput["enemy"+damageType+"Damage"]); p != nil {
				enemyDamage = *p
			} else if p := tonum(env.Build.ConfigPlaceholder["enemy"+damageType+"Damage"]); p != nil {
				enemyDamage = *p
			}
			var enemyPen float64
			if p := tonum(env.ConfigInput["enemy"+damageType+"Pen"]); p != nil {
				enemyPen = *p
			} else if p := tonum(env.Build.ConfigPlaceholder["enemy"+damageType+"Pen"]); p != nil {
				enemyPen = *p
			}
			var enemyOverwhelm float64
			if p := tonum(env.ConfigInput["enemy"+damageType+"Overwhelm"]); p != nil {
				enemyOverwhelm = *p
			} else if p := tonum(env.Build.ConfigPlaceholder["enemy"+damageType+"enemyOverwhelm"]); p != nil {
				// #EVAL the reference's placeholder key really is
				// "enemy<Type>enemyOverwhelm" (doubled prefix), so it never
				// matches a real placeholder
				enemyOverwhelm = *p
			}

			// Add min-max enemy damage from mods
			enemyDamage += (enemyDB.Sum("BASE", enemyCfg, damageType+"Min") + enemyDB.Sum("BASE", enemyCfg, damageType+"Max")) / 2

			// Conversion and Gain As Mods
			conversionTotal := 0.0
			if damageType == "Physical" {
				conv := map[string]float64{}
				convSkill := map[string]float64{}
				total, totalSkill := 0.0, 0.0
				for _, damageTypeTo := range dmgTypeList {
					convSkill[damageTypeTo] = enemyDB.Sum("BASE", enemyCfg, damageType+"DamageSkillConvertTo"+damageTypeTo)
					conv[damageTypeTo] = enemyDB.Sum("BASE", enemyCfg, damageType+"DamageConvertTo"+damageTypeTo)
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
					gainAsPercent := enemyDB.Sum("BASE", enemyCfg, damageType+"DamageGainAs"+damageTypeTo) / 100
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

			enemyOverwhelm += enemyDB.Sum("BASE", nil, "PhysicalOverwhelm") + modDB.Sum("BASE", nil, "EnemyPhysicalOverwhelm")

			output[damageType+"EnemyPen"] = enemyPen
			output[damageType+"EnemyDamageMult"] = enemyDamageMult
			output[damageType+"EnemyOverwhelm"] = enemyOverwhelm
			output["totalEnemyDamageIn"] = outNum(output, "totalEnemyDamageIn") + enemyDamage
			output[damageType+"EnemyDamage"] = enemyDamage * (1 - conversionTotal/100) * enemyDamageMult * outNum(output, "EnemyCritEffect")
			if conv := enemyDamageConversion[damageType]; conv != nil {
				// sorted: the reference iterates pairs() over damage-type keys
				for _, damageTypeFrom := range dmgTypeList {
					dmg, ok := conv[damageTypeFrom]
					if !ok {
						continue
					}
					mult := Mod(enemyDB, nil, enemyConvertedDamageNames(damageType, damageTypeFrom)...)
					output[damageType+"EnemyDamage"] = outNum(output, damageType+"EnemyDamage") + dmg*mult*outNum(output, "EnemyCritEffect")
				}
			}
			output["totalEnemyDamage"] = outNum(output, "totalEnemyDamage") + outNum(output, damageType+"EnemyDamage")
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
	if truthy(env.ConfigInput["PvpScaling"]) {
		panic("ehp: PvP scaling unported (no corpus build enables it)")
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
				dotShiftTable[destType] = modDB.Sum("BASE", nil, dotNames...)
				dotDestinationTotal += dotShiftTable[destType]
				shiftTable[destType] = dotShiftTable[destType] + modDB.Sum("BASE", nil, hitNames...)
				destTotal += shiftTable[destType]
			}
		}
		dotShiftTable[damageType] = math.Max(100-dotDestinationTotal, 0)
		actor.damageOverTimeShiftTable[damageType] = dotShiftTable
		shiftTable[damageType] = math.Max(100-destTotal, 0)
		actor.damageShiftTable[damageType] = shiftTable

		// add same type damage
		output[damageType+"TakenDamage"] = outNum(output, damageType+"EnemyDamage") * shiftTable[damageType] / 100
	}
	// converted damage types
	for _, damageType := range dmgTypeList {
		for _, damageConvertedType := range dmgTypeList {
			if damageType != damageConvertedType {
				damage := outNum(output, damageType+"EnemyDamage") * actor.damageShiftTable[damageType][damageConvertedType] / 100
				output[damageConvertedType+"TakenDamage"] = outNum(output, damageConvertedType+"TakenDamage") + damage
			}
		}
	}
	// total
	output["totalTakenDamage"] = 0.0
	for _, damageType := range dmgTypeList {
		output["totalTakenDamage"] = outNum(output, "totalTakenDamage") + outNum(output, damageType+"TakenDamage")
	}

	// Damage taken multipliers/Degen calculations
	output["AnyTakenReflect"] = false
	for _, damageType := range dmgTypeList {
		baseTakenInc := modDB.Sum("INC", nil, "DamageTaken", damageType+"DamageTaken")
		baseTakenMore := modDB.More(nil, "DamageTaken", damageType+"DamageTaken")
		if isElementalRes[damageType] {
			baseTakenInc += modDB.Sum("INC", nil, "ElementalDamageTaken")
			baseTakenMore *= modDB.More(nil, "ElementalDamageTaken")
		}
		{ // Hit
			takenInc := baseTakenInc + modDB.Sum("INC", nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenWhenHit", damageType+"DamageTakenWhenHit")
			if isElementalRes[damageType] {
				takenInc += modDB.Sum("INC", nil, "ElementalDamageTakenWhenHit")
				takenMore *= modDB.More(nil, "ElementalDamageTakenWhenHit")
			}
			output[damageType+"TakenHitMult"] = math.Max((1+takenInc/100)*takenMore, 0)

			for _, hitType := range hitSourceList {
				baseTakenIncType := takenInc + modDB.Sum("INC", nil, hitType+"DamageTaken")
				baseTakenMoreType := takenMore * modDB.More(nil, hitType+"DamageTaken")
				output[hitType+"TakenHitMult"] = math.Max((1+baseTakenIncType/100)*baseTakenMoreType, 0)
				output[damageType+hitType+"TakenHitMult"] = outNum(output, hitType+"TakenHitMult")
			}
			// Reflect
			takenInc += modDB.Sum("INC", nil, "ReflectedDamageTaken", damageType+"ReflectedDamageTaken")
			takenMore *= modDB.More(nil, "ReflectedDamageTaken", damageType+"ReflectedDamageTaken")
			if isElementalRes[damageType] {
				takenInc += modDB.Sum("INC", nil, "ElementalReflectedDamageTaken")
				takenMore *= modDB.More(nil, "ElementalReflectedDamageTaken")
			}
			output[damageType+"TakenReflect"] = math.Max((1+takenInc/100)*takenMore, 0)
			// #EVAL the reference assigns false in both branches here
			// ("this needs a rework as well"), so AnyTakenReflect never
			// becomes true
			if outNum(output, damageType+"TakenReflect") != outNum(output, damageType+"TakenHitMult") {
				output["AnyTakenReflect"] = false
			}
		}
		{ // Dot
			takenInc := baseTakenInc + modDB.Sum("INC", nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
			takenMore := baseTakenMore * modDB.More(nil, "DamageTakenOverTime", damageType+"DamageTakenOverTime")
			if isElementalRes[damageType] {
				takenInc += modDB.Sum("INC", nil, "ElementalDamageTakenOverTime")
				takenMore *= modDB.More(nil, "ElementalDamageTakenOverTime")
			}
			resist := 0.0
			if !modDB.Flag(nil, "SelfIgnore"+damageType+"Resistance") {
				if v, ok := output[damageType+"ResistOverTime"]; ok && truthy(v) {
					resist = anyNum(v)
				} else {
					resist = outNum(output, damageType+"Resist")
				}
			}
			reduction := 0.0
			if !modDB.Flag(nil, "SelfIgnoreBase"+damageType+"DamageReduction") {
				reduction = outNum(output, "Base"+damageType+"DamageReduction")
			}
			output[damageType+"TakenDotMult"] = math.Max((1-resist/100)*(1-reduction/100)*(1+takenInc/100)*takenMore, 0)
		}
	}
}
