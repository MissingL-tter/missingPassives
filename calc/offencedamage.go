// CalcOffence.lua L3326-3720: the two-pass (crit / non-crit) damage loop —
// per damage type hit damage, enemy resistances and damage taken, and the
// leech / gain-on-hit / gain-on-kill totals.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modstore"
)

// offenceDamageTypes ports L3326-3720 for one pass of passList.
func (env *Env) offenceDamageTypes(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output
	d := env.Data

	totalHitMin, totalHitMax, totalHitAvg := 0.0, 0.0, 0.0
	totalCritMin, totalCritMax, totalCritAvg := 0.0, 0.0, 0.0
	ghostReaver := skillModList.Flag(nil, "GhostReaver")
	ghostReaverLifeLeech := 0.0
	output["LifeLeech"] = 0.0
	output["LifeLeechInstant"] = 0.0
	output["EnergyShieldLeech"] = 0.0
	output["EnergyShieldLeechInstant"] = 0.0
	output["ManaLeech"] = 0.0
	output["ManaLeechInstant"] = 0.0
	output["impaleStoredHitAvg"] = 0.0

	if cfg.SkillCond == nil {
		cfg.SkillCond = map[string]bool{}
	}
	for p := 1; p <= 2; p++ {
		// Pass 1 is critical strike damage, pass 2 is non-critical strike
		cfg.SkillCond["CriticalStrike"] = p == 1
		lifeLeechTotal, energyShieldLeechTotal, manaLeechTotal := 0.0, 0.0, 0.0
		noLifeLeech := skillModList.Flag(cfg, "CannotLeechLife") || enemyDB.Flag(nil, "CannotLeechLifeFromSelf") || skillModList.Flag(cfg, "CannotGainLife")
		noEnergyShieldLeech := skillModList.Flag(cfg, "CannotLeechEnergyShield") || enemyDB.Flag(nil, "CannotLeechEnergyShieldFromSelf") || skillModList.Flag(cfg, "CannotGainEnergyShield")
		noManaLeech := skillModList.Flag(cfg, "CannotLeechMana") || enemyDB.Flag(nil, "CannotLeechManaFromSelf") || skillModList.Flag(cfg, "CannotGainMana")
		for _, damageType := range dmgTypeList {
			damageTypeHitMin, damageTypeHitMax, damageTypeHitAvg := 0.0, 0.0, 0.0
			if skillFlags["hit"] && c.canDeal[damageType] {
				damageTypeHitMin, damageTypeHitMax = env.calcDamage(activeSkill, output, cfg, damageType, 0, c.convTable(cfg))
				convMult := 1.0
				if e := c.convTable(cfg)[damageType]; e != nil {
					convMult = e.Mult
				}
				var allMult float64
				if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
					allMult = convMult * outNum(output, "ScaledDamageEffect") * outNum(output, "RuthlessBlowHitEffect") *
						outNum(output, "FistOfWarDamageEffect") * outNum(globalOutput, "MaxOffensiveWarcryEffect")
				} else {
					allMult = convMult * outNum(output, "ScaledDamageEffect") * outNum(output, "RuthlessBlowHitEffect") *
						outNum(output, "FistOfWarDamageEffect") * outNum(globalOutput, "OffensiveWarcryEffect")
				}
				output["allMult"] = allMult
				if p == 1 {
					// Apply crit multiplier
					allMult = allMult * outNum(output, "CritMultiplier")
				}
				damageTypeHitMin *= allMult
				damageTypeHitMax *= allMult

				var damageTypeLuckyChance float64
				if skillModList.Flag(skillCfg, "LuckyHits") ||
					(p == 2 && damageType == "Lightning" && skillModList.Flag(skillCfg, "LightningNoCritLucky")) ||
					(p == 1 && skillModList.Flag(skillCfg, "CritLucky")) ||
					(damageType == "Lightning" && modDB.Flag(nil, "LightningLuckHits")) ||
					(damageType == "Chaos" && modDB.Flag(nil, "ChaosLuckyHits")) ||
					(damageType == "Fire" && modDB.Flag(nil, "FireLuckyHits")) ||
					(damageType == "Cold" && modDB.Flag(nil, "ColdLuckyHits")) ||
					((damageType == "Lightning" || damageType == "Cold" || damageType == "Fire") && skillModList.Flag(skillCfg, "ElementalLuckHits")) {
					damageTypeLuckyChance = 1
				} else {
					damageTypeLuckyChance = math.Min(skillModList.Sum("BASE", skillCfg, "LuckyHitsChance"), 100) / 100
				}
				if skillModList.Flag(skillCfg, "UnluckyHits") {
					damageTypeLuckyChance = damageTypeLuckyChance - 1
				}
				rolls := damageTypeLuckyChance
				if skillModList.Flag(skillCfg, "ExtremeLuck") {
					rolls *= 2
				}
				if skillModList.Flag(skillCfg, "Unexciting") {
					rolls = 0
				}
				output["DamageRolls"] = rolls
				if p == 1 {
					output["CritDamageRolls"] = rolls
				}
				rolls = math.Abs(rolls) + 2
				damageTypeHitAvgNotLucky := damageTypeHitMin/rolls + damageTypeHitMax/rolls
				damageTypeHitAvgLucky := damageTypeHitMin/rolls + (rolls-1)*damageTypeHitMax/rolls
				damageTypeHitAvgUnlucky := (rolls-1)*damageTypeHitMin/rolls + damageTypeHitMax/rolls
				if damageTypeLuckyChance >= 0 {
					damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-damageTypeLuckyChance) + damageTypeHitAvgLucky*damageTypeLuckyChance
				} else {
					damageTypeHitAvg = damageTypeHitAvgNotLucky*(1-math.Abs(damageTypeLuckyChance)) + damageTypeHitAvgUnlucky*math.Abs(damageTypeLuckyChance)
				}

				if (damageTypeHitMin != 0 || damageTypeHitMax != 0) && env.ModeEffective {
					// Apply enemy resistances and damage taken modifiers
					resist, pen := 0.0, 0.0
					takenInc := enemyDB.Sum("INC", cfg, "DamageTaken", damageType+"DamageTaken")
					takenMore := enemyDB.More(cfg, "DamageTaken", damageType+"DamageTaken")
					// Check if player is supposed to ignore a damage type, or
					// if it's ignored on enemy side
					useThisResist := func(dt string) float64 {
						ignoreNames := optName(isElementalRes[dt], []string{"Ignore" + dt + "Resistance"}, "IgnoreElementalResistances")
						if skillModList.Flag(cfg, ignoreNames...) || enemyDB.Flag(nil, "SelfIgnore"+dt+"Resistance") {
							return 0
						}
						chanceNames := optName(isElementalRes[dt], []string{"ChanceToIgnore" + dt + "Resistance"}, "ChanceToIgnoreElementalResistances")
						chanceToIgnore := skillModList.Sum("BASE", cfg, chanceNames...)
						chanceToIgnore = math.Min(math.Max(chanceToIgnore, 0), 100)
						return 1 - chanceToIgnore/100
					}
					if damageType == "Physical" {
						// store pre-armour physical damage from attacks for
						// impale calculations
						if p == 1 {
							output["impaleStoredHitAvg"] = outNum(output, "impaleStoredHitAvg") + damageTypeHitAvg*(outNum(output, "CritChance")/100)
						} else {
							output["impaleStoredHitAvg"] = outNum(output, "impaleStoredHitAvg") + damageTypeHitAvg*(1-outNum(output, "CritChance")/100)
						}
						enemyArmour := math.Max(Val(enemyDB, "Armour", nil), 0)
						armourRed := armourReductionF(enemyArmour, damageTypeHitAvg*skillModList.More(cfg, "CalcArmourAsThoughDealing"))
						chanceToIgnoreDR := math.Min(skillModList.Sum("BASE", cfg, "ChanceToIgnoreEnemyPhysicalDamageReduction"), 100)
						if chanceToIgnoreDR > 0 && chanceToIgnoreDR < 100 {
							switch str(env.ConfigInput["ChanceToIgnoreEnemyPhysicalDamageReductionMode"]) {
							case "MAX":
								chanceToIgnoreDR = 100
							case "MIN":
								chanceToIgnoreDR = 0
							}
						}
						if skillModList.Flag(cfg, "IgnoreEnemyPhysicalDamageReduction") || chanceToIgnoreDR >= 100 {
							resist = 0
						} else {
							resist = math.Min(math.Max(0, enemyDB.Sum("BASE", nil, "PhysicalDamageReduction")+
								skillModList.Sum("BASE", cfg, "EnemyPhysicalDamageReduction")+armourRed), d.Misc.EnemyPhysicalDamageReductionCap)
							if resist > 0 {
								resist = resist * (1 - (skillModList.Sum("BASE", nil, "PartialIgnoreEnemyPhysicalDamageReduction")/100 + chanceToIgnoreDR/100))
							}
						}
					} else {
						// #EVAL: dotCfg is an undeclared global here (the
						// ailment sections declare their own local), so the
						// hit resist is looked up with a nil cfg.
						resist = env.calcResistForType(c, damageType, nil)
						elementUsed := damageType
						if ((skillModList.Flag(cfg, "ChaosDamageUsesLowestResistance") || skillModList.Flag(cfg, "ChaosDamageUsesHighestResistance")) && damageType == "Chaos") ||
							(skillModList.Flag(cfg, "ElementalDamageUsesLowestResistance") && isElementalRes[damageType]) {
							if isElementalRes[damageType] {
								takenInc += enemyDB.Sum("INC", cfg, "ElementalDamageTaken")
							}
							// Find the lowest resist of all the elements and
							// use that if it's lower
							for _, eleDamageType := range dmgTypeList {
								if isElementalRes[eleDamageType] && useThisResist(eleDamageType) > 0 && damageType != eleDamageType {
									currentElementResist := env.calcResistForType(c, eleDamageType, nil)
									if skillModList.Flag(cfg, "ChaosDamageUsesHighestResistance") {
										if resist < currentElementResist {
											resist = currentElementResist
											elementUsed = eleDamageType
										}
									} else {
										if resist > currentElementResist {
											resist = currentElementResist
											elementUsed = eleDamageType
										}
									}
								}
							}
							// Update the penetration based on the element used
							if isElementalRes[elementUsed] {
								pen = skillModList.Sum("BASE", cfg, elementUsed+"Penetration", "ElementalPenetration")
							} else if elementUsed == "Chaos" {
								pen = skillModList.Sum("BASE", cfg, "ChaosPenetration")
							}
						} else if isElementalRes[damageType] {
							pen = skillModList.Sum("BASE", cfg, damageType+"Penetration", "ElementalPenetration")
							takenInc += enemyDB.Sum("INC", cfg, "ElementalDamageTaken")
						} else if damageType == "Chaos" {
							pen = skillModList.Sum("BASE", cfg, "ChaosPenetration")
						}
					}
					invertChanceEle := math.Max(math.Min(skillModList.Sum("CHANCE", cfg, "HitsInvertEleResChance"), 1), 0)
					invertChanceChaos := math.Max(math.Min(skillModList.Sum("CHANCE", cfg, "HitsInvertChaosResChance"), 1), 0)
					invertChance := 0.0
					if isElementalRes[damageType] {
						invertChance = invertChanceEle
					} else if damageType == "Chaos" {
						invertChance = invertChanceChaos
					}
					// (the reference also rewrites sourceRes here; it only
					// feeds the breakdown text.)
					if skillFlags["projectile"] {
						takenInc += enemyDB.Sum("INC", nil, "ProjectileDamageTaken")
					}
					if skillFlags["projectile"] && skillFlags["attack"] {
						takenInc += enemyDB.Sum("INC", nil, "ProjectileAttackDamageTaken")
					}
					if skillFlags["trap"] || skillFlags["mine"] {
						takenInc += enemyDB.Sum("INC", nil, "TrapMineDamageTaken")
					}
					effMult := (1 + takenInc/100) * takenMore
					useResChance := useThisResist(damageType)
					useRes := useResChance > 0
					cannotElePenIgnore := isElementalRes[damageType] && skillModList.Flag(cfg, "CannotElePenIgnore")
					if cannotElePenIgnore {
						effectiveResist := resist
						if invertChance > 0 {
							effectiveResist = resist - 2*invertChance*resist
						}
						effMult = effMult * (1 - effectiveResist/100)
					} else if useRes {
						var effectiveResist float64
						if invertChance > 0 {
							effectiveResist = ((resist-pen)*(1-invertChance) + (-resist-pen)*invertChance) * useResChance
						} else {
							effectiveResist = (resist - pen) * useResChance
						}
						effMult = effMult * (1 - effectiveResist/100)
					}
					damageTypeHitMin *= effMult
					damageTypeHitMax *= effMult
					damageTypeHitAvg *= effMult
				}

				// Beginning of Leech Calculation for this DamageType
				lifeLeech, energyShieldLeech, manaLeech := 0.0, 0.0, 0.0
				if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
					lifeLeech = skillModList.Sum("BASE", cfg, "DamageLifeLeechToPlayer")
				} else {
					if skillModList.Flag(nil, "LifeLeechBasedOnChaosDamage") {
						if damageType == "Chaos" {
							lifeLeech = skillModList.Sum("BASE", cfg, "DamageLeech", "DamageLifeLeech", "PhysicalDamageLifeLeech",
								"LightningDamageLifeLeech", "ColdDamageLifeLeech", "FireDamageLifeLeech", "ChaosDamageLifeLeech",
								"ElementalDamageLifeLeech") + enemyDB.Sum("BASE", cfg, "SelfDamageLifeLeech")/100
						}
					} else {
						names := optName(isElementalRes[damageType], []string{"DamageLeech", "DamageLifeLeech", damageType + "DamageLifeLeech"}, "ElementalDamageLifeLeech")
						lifeLeech = skillModList.Sum("BASE", cfg, names...) + enemyDB.Sum("BASE", cfg, "SelfDamageLifeLeech")/100
					}
					esNames := optName(isElementalRes[damageType], []string{"DamageEnergyShieldLeech", damageType + "DamageEnergyShieldLeech"}, "ElementalDamageEnergyShieldLeech")
					energyShieldLeech = skillModList.Sum("BASE", cfg, esNames...) + enemyDB.Sum("BASE", cfg, "SelfDamageEnergyShieldLeech")/100
					manaNames := optName(isElementalRes[damageType], []string{"DamageLeech", "DamageManaLeech", damageType + "DamageManaLeech"}, "ElementalDamageManaLeech")
					manaLeech = skillModList.Sum("BASE", cfg, manaNames...) + enemyDB.Sum("BASE", cfg, "SelfDamageManaLeech")/100
				}

				if lifeLeech > 0 && !noLifeLeech {
					lifeLeechTotal += damageTypeHitAvg * lifeLeech / 100
				}
				if manaLeech > 0 && !noManaLeech {
					manaLeechTotal += damageTypeHitAvg * manaLeech / 100
				}
				if energyShieldLeech > 0 && !noEnergyShieldLeech {
					energyShieldLeechTotal += damageTypeHitAvg * energyShieldLeech / 100
				}
			}
			if p == 1 {
				output[damageType+"CritAverage"] = damageTypeHitAvg
				totalCritAvg += damageTypeHitAvg
				totalCritMin += damageTypeHitMin
				totalCritMax += damageTypeHitMax
			} else {
				output[damageType+"HitAverage"] = damageTypeHitAvg
				totalHitAvg += damageTypeHitAvg
				totalHitMin += damageTypeHitMin
				totalHitMax += damageTypeHitMax
			}
		}
		if truthy(skillData["lifeLeechPerUse"]) && !noLifeLeech {
			lifeLeechTotal += anyNum(skillData["lifeLeechPerUse"])
		}
		if truthy(skillData["manaLeechPerUse"]) {
			manaLeechTotal += anyNum(skillData["manaLeechPerUse"])
		}

		// leech caps per instance
		if ghostReaver {
			lifeLeechTotal = math.Min(lifeLeechTotal, outNum(globalOutput, "MaxEnergyShieldLeechInstance"))
		} else {
			lifeLeechTotal = math.Min(lifeLeechTotal, outNum(globalOutput, "MaxLifeLeechInstance"))
		}
		energyShieldLeechTotal = math.Min(energyShieldLeechTotal, outNum(globalOutput, "MaxEnergyShieldLeechInstance"))
		manaLeechTotal = math.Min(manaLeechTotal, outNum(globalOutput, "MaxManaLeechInstance"))
		if ghostReaver && noEnergyShieldLeech {
			lifeLeechTotal = 0
		}

		portion := 1 - outNum(output, "CritChance")/100
		if p == 1 {
			portion = outNum(output, "CritChance") / 100
		}
		if ghostReaver {
			ghostReaverLifeLeech += lifeLeechTotal * portion
		} else {
			output["LifeLeech"] = outNum(output, "LifeLeech") + lifeLeechTotal*portion
		}
		output["EnergyShieldLeech"] = outNum(output, "EnergyShieldLeech") + energyShieldLeechTotal*portion
		output["ManaLeech"] = outNum(output, "ManaLeech") + manaLeechTotal*portion
	}
	output["TotalMin"] = totalHitMin
	output["TotalMax"] = totalHitMax
	c.totalCritMin, c.totalCritMax, c.totalCritAvg = totalCritMin, totalCritMax, totalCritAvg
	c.totalHitAvg = totalHitAvg

	if skillModList.Flag(skillCfg, "ElementalEquilibrium") && !truthy(env.ConfigInput["EEIgnoreHitDamage"]) &&
		(outNum(output, "FireHitAverage")+outNum(output, "ColdHitAverage")+outNum(output, "LightningHitAverage") > 0) {
		// Update enemy hit-by-damage-type conditions
		enemyDB.Conditions["HitByFireDamage"] = outNum(output, "FireHitAverage") > 0
		enemyDB.Conditions["HitByColdDamage"] = outNum(output, "ColdHitAverage") > 0
		enemyDB.Conditions["HitByLightningDamage"] = outNum(output, "LightningHitAverage") > 0
	}

	highestType := "Physical"

	// For each damage type, calculate percentage of total damage. Also tracks
	// the highest damage type and outputs a Condition:TypeIsHighestDamageType
	// flag for whichever the highest type is
	for _, damageType := range dmgTypeList {
		if outNum(output, damageType+"HitAverage") > 0 {
			skillModList.AddMod(newMod("Condition:"+damageType+"HasDamage", "FLAG", true, "Config"))
			if outNum(output, damageType+"HitAverage") > outNum(output, highestType+"HitAverage") {
				highestType = damageType
			}
		}
	}
	if !skillModList.Flag(nil, "IsHighestDamageTypeOVERRIDE") {
		skillModList.AddMod(newMod("Condition:"+highestType+"IsHighestDamageType", "FLAG", true, "Config"))
	}

	speed := outNum(globalOutput, "Speed")
	if truthy(globalOutput["HitSpeed"]) {
		speed = outNum(globalOutput, "HitSpeed")
	}
	hitRate := outNum(output, "HitChance") / 100 * speed * anyNum(skillData["dpsMultiplier"])
	c.hitRate = hitRate

	// Calculate leech
	getLeechInstances := func(amount, total float64) (float64, float64) {
		if total == 0 {
			return 0, 0
		}
		duration := amount / total / d.Misc.LeechRateBase
		return duration, duration * hitRate
	}

	// Instant Leech
	output["LifeLeechInstantProportion"] = math.Max(math.Min(skillModList.Sum("BASE", cfg, "InstantLifeLeech"), 100), 0) / 100
	if outNum(output, "LifeLeechInstantProportion") > 0 {
		output["LifeLeechInstant"] = outNum(output, "LifeLeech") * outNum(output, "LifeLeechInstantProportion")
		output["LifeLeech"] = outNum(output, "LifeLeech") * (1 - outNum(output, "LifeLeechInstantProportion"))
	}
	if ghostReaver && ghostReaverLifeLeech > 0 {
		output["EnergyShieldLeech"] = outNum(output, "EnergyShieldLeech") + ghostReaverLifeLeech
		output["LifeLeech"] = 0.0
		output["LifeLeechInstant"] = 0.0
	}
	output["EnergyShieldLeechInstantProportion"] = math.Max(math.Min(skillModList.Sum("BASE", cfg, "InstantEnergyShieldLeech"), 100), 0) / 100
	if outNum(output, "EnergyShieldLeechInstantProportion") > 0 {
		output["EnergyShieldLeechInstant"] = outNum(output, "EnergyShieldLeech") * outNum(output, "EnergyShieldLeechInstantProportion")
		output["EnergyShieldLeech"] = outNum(output, "EnergyShieldLeech") * (1 - outNum(output, "EnergyShieldLeechInstantProportion"))
	}
	output["ManaLeechInstantProportion"] = math.Max(math.Min(skillModList.Sum("BASE", cfg, "InstantManaLeech"), 100), 0) / 100
	if outNum(output, "ManaLeechInstantProportion") > 0 {
		output["ManaLeechInstant"] = outNum(output, "ManaLeech") * outNum(output, "ManaLeechInstantProportion")
		output["ManaLeech"] = outNum(output, "ManaLeech") * (1 - outNum(output, "ManaLeechInstantProportion"))
	}

	output["LifeLeechDuration"], output["LifeLeechInstances"] = getLeechInstances(outNum(output, "LifeLeech"), outNum(globalOutput, "Life"))
	output["LifeLeechInstantRate"] = outNum(output, "LifeLeechInstant") * hitRate
	output["EnergyShieldLeechDuration"], output["EnergyShieldLeechInstances"] = getLeechInstances(outNum(output, "EnergyShieldLeech"), outNum(globalOutput, "EnergyShield"))
	output["EnergyShieldLeechInstantRate"] = outNum(output, "EnergyShieldLeechInstant") * hitRate
	output["ManaLeechDuration"], output["ManaLeechInstances"] = getLeechInstances(outNum(output, "ManaLeech"), outNum(globalOutput, "Mana"))
	output["ManaLeechInstantRate"] = outNum(output, "ManaLeechInstant") * hitRate

	// Calculate gain on hit
	if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
		output["LifeOnHit"] = 0.0
		output["EnergyShieldOnHit"] = 0.0
		output["ManaOnHit"] = 0.0
	} else {
		output["LifeOnHit"] = 0.0
		if !skillModList.Flag(cfg, "CannotGainLife") && !skillModList.Flag(cfg, "CannotRecoverLifeOutsideLeech") {
			output["LifeOnHit"] = skillModList.Sum("BASE", cfg, "LifeOnHit") + enemyDB.Sum("BASE", cfg, "SelfLifeOnHit")
		}
		output["EnergyShieldOnHit"] = 0.0
		if !skillModList.Flag(cfg, "CannotGainEnergyShield") {
			output["EnergyShieldOnHit"] = skillModList.Sum("BASE", cfg, "EnergyShieldOnHit") + enemyDB.Sum("BASE", cfg, "SelfEnergyShieldOnHit")
		}
		output["ManaOnHit"] = 0.0
		if !skillModList.Flag(cfg, "CannotGainMana") {
			output["ManaOnHit"] = skillModList.Sum("BASE", cfg, "ManaOnHit") + enemyDB.Sum("BASE", cfg, "SelfManaOnHit")
		}
	}
	output["LifeOnHitRate"] = outNum(output, "LifeOnHit") * hitRate
	output["EnergyShieldOnHitRate"] = outNum(output, "EnergyShieldOnHit") * hitRate
	output["ManaOnHitRate"] = outNum(output, "ManaOnHit") * hitRate

	// Calculate gain on kill
	if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
		output["LifeOnKill"] = 0.0
		output["EnergyShieldOnKill"] = 0.0
		output["ManaOnKill"] = 0.0
	} else {
		output["LifeOnKill"] = 0.0
		if !skillModList.Flag(cfg, "CannotGainLife") && !skillModList.Flag(cfg, "CannotRecoverLifeOutsideLeech") {
			output["LifeOnKill"] = math.Floor(skillModList.Sum("BASE", cfg, "LifeOnKill"))
		}
		output["EnergyShieldOnKill"] = 0.0
		if !skillModList.Flag(cfg, "CannotGainEnergyShield") {
			output["EnergyShieldOnKill"] = math.Floor(skillModList.Sum("BASE", cfg, "EnergyShieldOnKill"))
		}
		output["ManaOnKill"] = 0.0
		if !skillModList.Flag(cfg, "CannotGainMana") {
			output["ManaOnKill"] = math.Floor(skillModList.Sum("BASE", cfg, "ManaOnKill"))
		}
	}
}

var _ = modstore.Cfg{}
