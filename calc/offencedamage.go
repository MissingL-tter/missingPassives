// CalcOffence.lua L3326-3720: the two-pass (crit / non-crit) damage loop —
// per damage type hit damage, enemy resistances and damage taken, and the
// leech / gain-on-hit / gain-on-kill totals.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"

	"github.com/MissingL-tter/missingPassives/modstore"
)

// offenceDamageTypes ports L3326-3720 for one pass of passList.
func (env *Env) offenceDamageTypes(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output

	totalHitMin, totalHitMax, totalHitAvg := 0.0, 0.0, 0.0
	totalCritMin, totalCritMax, totalCritAvg := 0.0, 0.0, 0.0
	ghostReaver := skillModList.Flag(nil, "GhostReaver")
	ghostReaverLifeLeech := 0.0
	output.SetN("LifeLeech", 0.0)
	output.SetN("LifeLeechInstant", 0.0)
	output.SetN("EnergyShieldLeech", 0.0)
	output.SetN("EnergyShieldLeechInstant", 0.0)
	output.SetN("ManaLeech", 0.0)
	output.SetN("ManaLeechInstant", 0.0)
	output.SetN("impaleStoredHitAvg", 0.0)

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
					allMult = convMult * output.N("ScaledDamageEffect") * output.N("RuthlessBlowHitEffect") *
						output.N("FistOfWarDamageEffect") * globalOutput.N("MaxOffensiveWarcryEffect")
				} else {
					allMult = convMult * output.N("ScaledDamageEffect") * output.N("RuthlessBlowHitEffect") *
						output.N("FistOfWarDamageEffect") * globalOutput.N("OffensiveWarcryEffect")
				}
				output.SetN("allMult", allMult)
				if p == 1 {
					// Apply crit multiplier
					allMult = allMult * output.N("CritMultiplier")
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
					damageTypeLuckyChance = math.Min(skillModList.Sum(modparser.Base, skillCfg, "LuckyHitsChance"), 100) / 100
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
				output.SetN("DamageRolls", rolls)
				if p == 1 {
					output.SetN("CritDamageRolls", rolls)
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
					takenInc := enemyDB.Sum(modparser.Inc, cfg, "DamageTaken", damageType+"DamageTaken")
					takenMore := enemyDB.More(cfg, "DamageTaken", damageType+"DamageTaken")
					// Check if player is supposed to ignore a damage type, or
					// if it's ignored on enemy side
					useThisResist := func(dt string) float64 {
						ignoreNames := optName(isElementalRes[dt], []string{"Ignore" + dt + "Resistance"}, "IgnoreElementalResistances")
						if skillModList.Flag(cfg, ignoreNames...) || enemyDB.Flag(nil, "SelfIgnore"+dt+"Resistance") {
							return 0
						}
						chanceNames := optName(isElementalRes[dt], []string{"ChanceToIgnore" + dt + "Resistance"}, "ChanceToIgnoreElementalResistances")
						chanceToIgnore := skillModList.Sum(modparser.Base, cfg, chanceNames...)
						chanceToIgnore = math.Min(math.Max(chanceToIgnore, 0), 100)
						return 1 - chanceToIgnore/100
					}
					if damageType == "Physical" {
						// store pre-armour physical damage from attacks for
						// impale calculations
						if p == 1 {
							output.SetN("impaleStoredHitAvg", output.N("impaleStoredHitAvg")+damageTypeHitAvg*(output.N("CritChance")/100))
						} else {
							output.SetN("impaleStoredHitAvg", output.N("impaleStoredHitAvg")+damageTypeHitAvg*(1-output.N("CritChance")/100))
						}
						enemyArmour := math.Max(Val(enemyDB, "Armour", nil), 0)
						armourRed := armourReductionF(enemyArmour, damageTypeHitAvg*skillModList.More(cfg, "CalcArmourAsThoughDealing"))
						chanceToIgnoreDR := math.Min(skillModList.Sum(modparser.Base, cfg, "ChanceToIgnoreEnemyPhysicalDamageReduction"), 100)
						if chanceToIgnoreDR > 0 && chanceToIgnoreDR < 100 {
							switch env.ConfigInput.ChanceToIgnoreEnemyPhysicalDamageReductionMode {
							case "MAX":
								chanceToIgnoreDR = 100
							case "MIN":
								chanceToIgnoreDR = 0
							}
						}
						if skillModList.Flag(cfg, "IgnoreEnemyPhysicalDamageReduction") || chanceToIgnoreDR >= 100 {
							resist = 0
						} else {
							resist = math.Min(math.Max(0, enemyDB.Sum(modparser.Base, nil, "PhysicalDamageReduction")+
								skillModList.Sum(modparser.Base, cfg, "EnemyPhysicalDamageReduction")+armourRed), data.Misc.EnemyPhysicalDamageReductionCap)
							if resist > 0 {
								resist = resist * (1 - (skillModList.Sum(modparser.Base, nil, "PartialIgnoreEnemyPhysicalDamageReduction")/100 + chanceToIgnoreDR/100))
							}
						}
					} else {
						// dotCfg is an undeclared global here (the
						// ailment sections declare their own local), so the
						// hit resist is looked up with a nil cfg.
						resist = env.calcResistForType(c, damageType, nil)
						elementUsed := damageType
						if ((skillModList.Flag(cfg, "ChaosDamageUsesLowestResistance") || skillModList.Flag(cfg, "ChaosDamageUsesHighestResistance")) && damageType == "Chaos") ||
							(skillModList.Flag(cfg, "ElementalDamageUsesLowestResistance") && isElementalRes[damageType]) {
							if isElementalRes[damageType] {
								takenInc += enemyDB.Sum(modparser.Inc, cfg, "ElementalDamageTaken")
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
								pen = skillModList.Sum(modparser.Base, cfg, elementUsed+"Penetration", "ElementalPenetration")
							} else if elementUsed == "Chaos" {
								pen = skillModList.Sum(modparser.Base, cfg, "ChaosPenetration")
							}
						} else if isElementalRes[damageType] {
							pen = skillModList.Sum(modparser.Base, cfg, damageType+"Penetration", "ElementalPenetration")
							takenInc += enemyDB.Sum(modparser.Inc, cfg, "ElementalDamageTaken")
						} else if damageType == "Chaos" {
							pen = skillModList.Sum(modparser.Base, cfg, "ChaosPenetration")
						}
					}
					invertChanceEle := math.Max(math.Min(skillModList.Sum(modparser.Chance, cfg, "HitsInvertEleResChance"), 1), 0)
					invertChanceChaos := math.Max(math.Min(skillModList.Sum(modparser.Chance, cfg, "HitsInvertChaosResChance"), 1), 0)
					invertChance := 0.0
					if isElementalRes[damageType] {
						invertChance = invertChanceEle
					} else if damageType == "Chaos" {
						invertChance = invertChanceChaos
					}
					// (the reference also rewrites sourceRes here; it only
					// feeds the breakdown text.)
					if skillFlags["projectile"] {
						takenInc += enemyDB.Sum(modparser.Inc, nil, "ProjectileDamageTaken")
					}
					if skillFlags["projectile"] && skillFlags["attack"] {
						takenInc += enemyDB.Sum(modparser.Inc, nil, "ProjectileAttackDamageTaken")
					}
					if skillFlags["trap"] || skillFlags["mine"] {
						takenInc += enemyDB.Sum(modparser.Inc, nil, "TrapMineDamageTaken")
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
					lifeLeech = skillModList.Sum(modparser.Base, cfg, "DamageLifeLeechToPlayer")
				} else {
					if skillModList.Flag(nil, "LifeLeechBasedOnChaosDamage") {
						if damageType == "Chaos" {
							lifeLeech = skillModList.Sum(modparser.Base, cfg, "DamageLeech", "DamageLifeLeech", "PhysicalDamageLifeLeech",
								"LightningDamageLifeLeech", "ColdDamageLifeLeech", "FireDamageLifeLeech", "ChaosDamageLifeLeech",
								"ElementalDamageLifeLeech") + enemyDB.Sum(modparser.Base, cfg, "SelfDamageLifeLeech")/100
						}
					} else {
						names := optName(isElementalRes[damageType], []string{"DamageLeech", "DamageLifeLeech", damageType + "DamageLifeLeech"}, "ElementalDamageLifeLeech")
						lifeLeech = skillModList.Sum(modparser.Base, cfg, names...) + enemyDB.Sum(modparser.Base, cfg, "SelfDamageLifeLeech")/100
					}
					esNames := optName(isElementalRes[damageType], []string{"DamageEnergyShieldLeech", damageType + "DamageEnergyShieldLeech"}, "ElementalDamageEnergyShieldLeech")
					energyShieldLeech = skillModList.Sum(modparser.Base, cfg, esNames...) + enemyDB.Sum(modparser.Base, cfg, "SelfDamageEnergyShieldLeech")/100
					manaNames := optName(isElementalRes[damageType], []string{"DamageLeech", "DamageManaLeech", damageType + "DamageManaLeech"}, "ElementalDamageManaLeech")
					manaLeech = skillModList.Sum(modparser.Base, cfg, manaNames...) + enemyDB.Sum(modparser.Base, cfg, "SelfDamageManaLeech")/100
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
				output.SetN(damageType+"CritAverage", damageTypeHitAvg)
				totalCritAvg += damageTypeHitAvg
				totalCritMin += damageTypeHitMin
				totalCritMax += damageTypeHitMax
			} else {
				output.SetN(damageType+"HitAverage", damageTypeHitAvg)
				totalHitAvg += damageTypeHitAvg
				totalHitMin += damageTypeHitMin
				totalHitMax += damageTypeHitMax
			}
		}
		if skillData.Flag("lifeLeechPerUse") && !noLifeLeech {
			lifeLeechTotal += skillData.N("lifeLeechPerUse")
		}
		if skillData.Flag("manaLeechPerUse") {
			manaLeechTotal += skillData.N("manaLeechPerUse")
		}

		// leech caps per instance
		if ghostReaver {
			lifeLeechTotal = math.Min(lifeLeechTotal, globalOutput.N("MaxEnergyShieldLeechInstance"))
		} else {
			lifeLeechTotal = math.Min(lifeLeechTotal, globalOutput.N("MaxLifeLeechInstance"))
		}
		energyShieldLeechTotal = math.Min(energyShieldLeechTotal, globalOutput.N("MaxEnergyShieldLeechInstance"))
		manaLeechTotal = math.Min(manaLeechTotal, globalOutput.N("MaxManaLeechInstance"))
		if ghostReaver && noEnergyShieldLeech {
			lifeLeechTotal = 0
		}

		portion := 1 - output.N("CritChance")/100
		if p == 1 {
			portion = output.N("CritChance") / 100
		}
		if ghostReaver {
			ghostReaverLifeLeech += lifeLeechTotal * portion
		} else {
			output.SetN("LifeLeech", output.N("LifeLeech")+lifeLeechTotal*portion)
		}
		output.SetN("EnergyShieldLeech", output.N("EnergyShieldLeech")+energyShieldLeechTotal*portion)
		output.SetN("ManaLeech", output.N("ManaLeech")+manaLeechTotal*portion)
	}
	output.SetN("TotalMin", totalHitMin)
	output.SetN("TotalMax", totalHitMax)
	// totalCritMin/Max feed only the reference breakdown; totalCritAvg is
	// what the average-hit maths needs.
	_, _ = totalCritMin, totalCritMax
	c.totalCritAvg = totalCritAvg
	c.totalHitAvg = totalHitAvg

	if skillModList.Flag(skillCfg, "ElementalEquilibrium") && !env.ConfigInput.EEIgnoreHitDamage &&
		(output.N("FireHitAverage")+output.N("ColdHitAverage")+output.N("LightningHitAverage") > 0) {
		// Update enemy hit-by-damage-type conditions
		enemyDB.Conditions.Set("HitByFireDamage", output.N("FireHitAverage") > 0)
		enemyDB.Conditions.Set("HitByColdDamage", output.N("ColdHitAverage") > 0)
		enemyDB.Conditions.Set("HitByLightningDamage", output.N("LightningHitAverage") > 0)
	}

	highestType := "Physical"

	// For each damage type, calculate percentage of total damage. Also tracks
	// the highest damage type and outputs a Condition:TypeIsHighestDamageType
	// flag for whichever the highest type is
	for _, damageType := range dmgTypeList {
		if output.N(damageType+"HitAverage") > 0 {
			skillModList.AddMod(newModS("Condition:"+damageType+"HasDamage", modparser.Flag, modparser.Bool(true), "Config"))
			if output.N(damageType+"HitAverage") > output.N(highestType+"HitAverage") {
				highestType = damageType
			}
		}
	}
	if !skillModList.Flag(nil, "IsHighestDamageTypeOVERRIDE") {
		skillModList.AddMod(newModS("Condition:"+highestType+"IsHighestDamageType", modparser.Flag, modparser.Bool(true), "Config"))
	}

	speed := globalOutput.N("Speed")
	if globalOutput.Has("HitSpeed") {
		speed = globalOutput.N("HitSpeed")
	}
	hitRate := output.N("HitChance") / 100 * speed * skillData.N("dpsMultiplier")

	// Calculate leech
	getLeechInstances := func(amount, total float64) (float64, float64) {
		if total == 0 {
			return 0, 0
		}
		duration := amount / total / data.Misc.LeechRateBase
		return duration, duration * hitRate
	}

	// Instant Leech
	output.SetN("LifeLeechInstantProportion", math.Max(math.Min(skillModList.Sum(modparser.Base, cfg, "InstantLifeLeech"), 100), 0)/100)
	if output.N("LifeLeechInstantProportion") > 0 {
		output.SetN("LifeLeechInstant", output.N("LifeLeech")*output.N("LifeLeechInstantProportion"))
		output.SetN("LifeLeech", output.N("LifeLeech")*(1-output.N("LifeLeechInstantProportion")))
	}
	if ghostReaver && ghostReaverLifeLeech > 0 {
		output.SetN("EnergyShieldLeech", output.N("EnergyShieldLeech")+ghostReaverLifeLeech)
		output.SetN("LifeLeech", 0.0)
		output.SetN("LifeLeechInstant", 0.0)
	}
	output.SetN("EnergyShieldLeechInstantProportion", math.Max(math.Min(skillModList.Sum(modparser.Base, cfg, "InstantEnergyShieldLeech"), 100), 0)/100)
	if output.N("EnergyShieldLeechInstantProportion") > 0 {
		output.SetN("EnergyShieldLeechInstant", output.N("EnergyShieldLeech")*output.N("EnergyShieldLeechInstantProportion"))
		output.SetN("EnergyShieldLeech", output.N("EnergyShieldLeech")*(1-output.N("EnergyShieldLeechInstantProportion")))
	}
	output.SetN("ManaLeechInstantProportion", math.Max(math.Min(skillModList.Sum(modparser.Base, cfg, "InstantManaLeech"), 100), 0)/100)
	if output.N("ManaLeechInstantProportion") > 0 {
		output.SetN("ManaLeechInstant", output.N("ManaLeech")*output.N("ManaLeechInstantProportion"))
		output.SetN("ManaLeech", output.N("ManaLeech")*(1-output.N("ManaLeechInstantProportion")))
	}

	setLeech := func(resource string, pool float64) {
		duration, instances := getLeechInstances(output.N(resource+"Leech"), pool)
		output.SetN(resource+"LeechDuration", duration)
		output.SetN(resource+"LeechInstances", instances)
	}
	setLeech("Life", globalOutput.N("Life"))
	output.SetN("LifeLeechInstantRate", output.N("LifeLeechInstant")*hitRate)
	setLeech("EnergyShield", globalOutput.N("EnergyShield"))
	output.SetN("EnergyShieldLeechInstantRate", output.N("EnergyShieldLeechInstant")*hitRate)
	setLeech("Mana", globalOutput.N("Mana"))
	output.SetN("ManaLeechInstantRate", output.N("ManaLeechInstant")*hitRate)

	// Calculate gain on hit
	if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
		output.SetN("LifeOnHit", 0.0)
		output.SetN("EnergyShieldOnHit", 0.0)
		output.SetN("ManaOnHit", 0.0)
	} else {
		output.SetN("LifeOnHit", 0.0)
		if !skillModList.Flag(cfg, "CannotGainLife") && !skillModList.Flag(cfg, "CannotRecoverLifeOutsideLeech") {
			output.SetN("LifeOnHit", skillModList.Sum(modparser.Base, cfg, "LifeOnHit")+enemyDB.Sum(modparser.Base, cfg, "SelfLifeOnHit"))
		}
		output.SetN("EnergyShieldOnHit", 0.0)
		if !skillModList.Flag(cfg, "CannotGainEnergyShield") {
			output.SetN("EnergyShieldOnHit", skillModList.Sum(modparser.Base, cfg, "EnergyShieldOnHit")+enemyDB.Sum(modparser.Base, cfg, "SelfEnergyShieldOnHit"))
		}
		output.SetN("ManaOnHit", 0.0)
		if !skillModList.Flag(cfg, "CannotGainMana") {
			output.SetN("ManaOnHit", skillModList.Sum(modparser.Base, cfg, "ManaOnHit")+enemyDB.Sum(modparser.Base, cfg, "SelfManaOnHit"))
		}
	}
	output.SetN("LifeOnHitRate", output.N("LifeOnHit")*hitRate)
	output.SetN("EnergyShieldOnHitRate", output.N("EnergyShieldOnHit")*hitRate)
	output.SetN("ManaOnHitRate", output.N("ManaOnHit")*hitRate)

	// Calculate gain on kill
	if skillFlags["mine"] || skillFlags["trap"] || skillFlags["totem"] {
		output.SetN("LifeOnKill", 0.0)
		output.SetN("EnergyShieldOnKill", 0.0)
		output.SetN("ManaOnKill", 0.0)
	} else {
		output.SetN("LifeOnKill", 0.0)
		if !skillModList.Flag(cfg, "CannotGainLife") && !skillModList.Flag(cfg, "CannotRecoverLifeOutsideLeech") {
			output.SetN("LifeOnKill", math.Floor(skillModList.Sum(modparser.Base, cfg, "LifeOnKill")))
		}
		output.SetN("EnergyShieldOnKill", 0.0)
		if !skillModList.Flag(cfg, "CannotGainEnergyShield") {
			output.SetN("EnergyShieldOnKill", math.Floor(skillModList.Sum(modparser.Base, cfg, "EnergyShieldOnKill")))
		}
		output.SetN("ManaOnKill", 0.0)
		if !skillModList.Flag(cfg, "CannotGainMana") {
			output.SetN("ManaOnKill", math.Floor(skillModList.Sum(modparser.Base, cfg, "ManaOnKill")))
		}
	}
}

var _ = modstore.Cfg{}
