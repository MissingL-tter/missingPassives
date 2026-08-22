// CalcOffence.lua L2971-3300: crit chance/multiplier, double/triple damage,
// culling, the Cryogenesis added-damage redirect and the base hit damage.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// offenceFistOfWar ports L2986-3020 (the Fist of War branch of the Ruthless
// section), kept separate so the enclosing switch stays readable.
func (env *Env) offenceFistOfWar(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, output := c.skillModList, c.skillCfg, pass.output
	activeSkill, globalOutput := c.activeSkill, c.output

	globalOutput["FistOfWarDamageMultiplier"] = skillModList.Sum("BASE", nil, "FistOfWarDamageMultiplier") / 100
	globalOutput["FistOfWarUptimeRatio"] = math.Min((1/outNum(globalOutput, "Speed"))/outNum(globalOutput, "FistOfWarCooldown"), 1) * 100
	globalOutput["AvgFistOfWarDamage"] = globalOutput["FistOfWarDamageMultiplier"]
	globalOutput["AvgFistOfWarDamageEffect"] = 1 + outNum(globalOutput, "FistOfWarDamageMultiplier")*(outNum(globalOutput, "FistOfWarUptimeRatio")/100)
	globalOutput["MaxFistOfWarDamageEffect"] = 1 + outNum(globalOutput, "FistOfWarDamageMultiplier")
	if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		output["FistOfWarDamageEffect"] = globalOutput["MaxFistOfWarDamageEffect"]
		skillModList.AddMod(newMod("AreaOfEffect", "INC", skillModList.Sum("BASE", nil, "FistOfWarIncAoE"), "Max Fist of War Boosted AoE"))
	} else {
		output["FistOfWarDamageEffect"] = globalOutput["AvgFistOfWarDamageEffect"]
		skillModList.AddMod(newMod("AreaOfEffect", "INC",
			math.Floor(skillModList.Sum("BASE", nil, "FistOfWarIncAoE")/100*outNum(globalOutput, "FistOfWarUptimeRatio")), "Avg Fist Of War Boosted AoE"))
	}
	_ = skillCfg
	env.calcAreaOfEffect(c)
	globalOutput["TheoreticalOffensiveWarcryEffect"] = outNum(globalOutput, "TheoreticalOffensiveWarcryEffect") * outNum(globalOutput, "AvgFistOfWarDamageEffect")
	globalOutput["TheoreticalMaxOffensiveWarcryEffect"] = outNum(globalOutput, "TheoreticalMaxOffensiveWarcryEffect") * outNum(globalOutput, "MaxFistOfWarDamageEffect")
}

// offenceCrit ports L2972-3300 for one pass.
func (env *Env) offenceCrit(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	activeSkill, actor, enemyDB, modDB := c.activeSkill, c.actor, c.enemyDB, c.modDB
	cfg, output, source := pass.cfg, pass.output, pass.source
	globalOutput := c.output

	// Calculate maximum sustainable fuses and explosion rate for Explosive
	// Arrow. Does not take into account mines or traps.
	if activeSkill.ActiveEffect.GrantedEffect.Name == "Explosive Arrow" {
		panic("offence: Explosive Arrow's explosiveArrowFunc callback is unported")
	}

	// Calculate crit chance, crit multiplier, and their combined effect
	switch {
	case skillModList.Flag(cfg, "NeverCrit"):
		output["PreEffectiveCritChance"] = 0.0
		output["CritChance"] = 0.0
		output["CritMultiplier"] = 0.0
		output["BonusCritDotMultiplier"] = 0.0
		output["CritEffect"] = 1.0
	case skillModList.Flag(cfg, "SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat"):
		repeats := 1.0
		if truthy(output["Repeats"]) {
			repeats = anyNum(output["Repeats"])
		}
		if repeats == 1 {
			output["PreEffectiveCritChance"] = 0.0
			output["CritChance"] = 0.0
			output["CritMultiplier"] = 0.0
			output["BonusCritDotMultiplier"] = 0.0
			output["CritEffect"] = 1.0
		} else if skillModList.Flag(cfg, "SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat") {
			switch str(env.ConfigInput["repeatMode"]) {
			case "None":
				output["PreEffectiveCritChance"] = 0.0
				output["CritChance"] = 0.0
			case "AVERAGE":
				output["PreEffectiveCritChance"] = 100 / repeats
				output["CritChance"] = 100 / repeats
			default:
				output["PreEffectiveCritChance"] = 100.0
				output["CritChance"] = 100.0
			}
		}
	default:
		critOverride := skillModList.Override(cfg, "CritChance")
		// destructive link
		if skillModList.Flag(cfg, "MainHandCritIsEqualToParent") {
			if actor.parent == nil {
				panic("offence: MainHandCritIsEqualToParent without a parent actor")
			}
			if mh, ok := actor.parent.output["MainHand"].(map[string]any); ok && truthy(mh) {
				critOverride = mh["CritChance"]
			} else {
				critOverride = actor.parent.ms.WeaponData1["CritChance"]
			}
		} else if skillModList.Flag(cfg, "MainHandCritIsEqualToPartyMember") {
			panic("offence: MainHandCritIsEqualToPartyMember (party tab) unported")
		}
		baseCrit := 0.0
		if truthy(critOverride) {
			baseCrit = anyNum(critOverride)
		} else if truthy(source["CritChance"]) {
			baseCrit = anyNum(source["CritChance"])
		}

		baseCritFromMainHand := skillModList.Flag(cfg, "BaseCritFromMainHand")
		baseCritFromParentMainHand := skillModList.Flag(cfg, "AttackCritIsEqualToParentMainHand")
		if baseCritFromMainHand {
			baseCrit = anyNum(actor.ms.WeaponData1["CritChance"])
		} else if baseCritFromParentMainHand {
			if actor.parent != nil && actor.parent.ms.WeaponData1 != nil {
				baseCrit = anyNum(actor.parent.ms.WeaponData1["CritChance"])
			}
		}

		if truthy(critOverride) && anyNum(critOverride) == 100 {
			output["PreEffectiveCritChance"] = 100.0
			output["PreBifurcateCritChance"] = 100.0
			output["CritChance"] = 100.0
		} else {
			base, inc, more := 0.0, 0.0, 1.0
			if !truthy(critOverride) {
				selfBase, selfInc := 0.0, 0.0
				if env.ModeEffective {
					selfBase = enemyDB.Sum("BASE", nil, "SelfCritChance")
					selfInc = enemyDB.Sum("INC", nil, "SelfCritChance")
				}
				base = skillModList.Sum("BASE", cfg, "CritChance") + selfBase
				inc = skillModList.Sum("INC", cfg, "CritChance") + selfInc
				more = skillModList.More(cfg, "CritChance")
			}
			critChance := (baseCrit + base) * (1 + inc/100) * more
			cap := skillModList.Sum("BASE", cfg, "CritChanceCap")
			if ov := skillModList.Override(nil, "CritChanceCap"); truthy(ov) {
				cap = anyNum(ov)
			}
			critChance = math.Min(critChance, cap)
			if (baseCrit + base) > 0 {
				critChance = math.Max(critChance, 0)
			}
			output["PreEffectiveCritChance"] = critChance
			critRolls := 0.0
			if env.ModeEffective && skillModList.Flag(cfg, "CritChanceLucky") {
				critRolls++
			}
			if skillModList.Flag(skillCfg, "ExtremeLuck") {
				critRolls *= 2
			}
			if critRolls != 0 {
				if modDB.Flag(nil, "Unexciting") {
					// Unexciting rolls three times and keeps the median
					// result -> 3p^2 - 2p^3
					p := critChance / 100
					critChance = (3*p*p - 2*p*p*p) * 100
				} else {
					critChance = (1 - math.Pow(1-critChance/100, critRolls+1)) * 100
				}
			}
			output["PreBifurcateCritChance"] = critChance
			if env.ModeEffective && skillModList.Flag(cfg, "BifurcateCrit") {
				critChance = (1 - math.Pow(1-critChance/100, 2)) * 100
			}
			if env.ModeEffective {
				if skillModList.Flag(skillCfg, "Every3UseCrit") {
					critChance = (2*critChance + 100) / 3
				}
				if skillModList.Flag(skillCfg, "Every5UseCrit") {
					critChance = (4*critChance + 100) / 5
				}
				critChance = critChance * outNum(output, "AccuracyHitChance") / 100
			}
			output["CritChance"] = critChance
		}
	}
	if !truthy(output["CritEffect"]) {
		if skillModList.Flag(cfg, "NoCritMultiplier") {
			output["CritMultiplier"] = 1.0
		} else {
			extraDamage := skillModList.Sum("BASE", cfg, "CritMultiplier") / 100
			if multiOverride := skillModList.Override(skillCfg, "CritMultiplier"); truthy(multiOverride) {
				extraDamage = (anyNum(multiOverride) - 100) / 100
			}
			if env.ModeEffective {
				enemyInc := 1 + enemyDB.Sum("INC", nil, "SelfCritMultiplier")/100
				extraDamage += enemyDB.Sum("BASE", nil, "SelfCritMultiplier") / 100
				extraDamage = roundDec(extraDamage*enemyInc, 2)
			}
			// if crit bifurcates are enabled, roll for crit twice and add
			// multiplier for each
			critOverride := skillModList.Override(cfg, "CritChance")
			if env.ModeEffective && skillModList.Flag(cfg, "BifurcateCrit") && truthy(output["PreBifurcateCritChance"]) &&
				!(truthy(critOverride) && anyNum(critOverride) == 100) {
				critChancePercentage := outNum(output, "PreBifurcateCritChance")
				bifurcateMultiChance := critChancePercentage * critChancePercentage / 100
				effectiveCritChance := outNum(output, "CritChance")
				bifurcateUseChance := 1.0
				// Guaranteed crit uses do not roll crit chance and therefore
				// cannot bifurcate
				if skillModList.Flag(skillCfg, "Every3UseCrit") {
					bifurcateUseChance = bifurcateUseChance * 2 / 3
				}
				if skillModList.Flag(skillCfg, "Every5UseCrit") {
					bifurcateUseChance = bifurcateUseChance * 4 / 5
				}
				bifurcateMultiChance *= bifurcateUseChance
				conditionalBifurcateChance := 0.0
				if effectiveCritChance > 0 {
					conditionalBifurcateChance = bifurcateMultiChance / effectiveCritChance
				}
				output["CritBifurcates"] = 1 + conditionalBifurcateChance
				extraDamage = extraDamage + conditionalBifurcateChance*extraDamage
				// mod doesn't affect output and is purely descriptive
				skillModList.AddMod(newMod("CritMultiplier", "MORE", floorDec(conditionalBifurcateChance*100, 2),
					"Bifurcated Crit Damage Bonus", modparser.ModFlag.Hit))
			}
			output["CritMultiplier"] = 1 + math.Max(0, extraDamage)
		}
		critChancePercentage := outNum(output, "CritChance") / 100
		output["CritEffect"] = 1 - critChancePercentage + critChancePercentage*outNum(output, "CritMultiplier")
		output["BonusCritDotMultiplier"] = (skillModList.Sum("BASE", cfg, "CritMultiplier") - 50) *
			skillModList.Sum("BASE", cfg, "CritMultiplierAppliesToDegen") / 10000
	}
	if outNum(output, "CritChance") != 0 {
		skillModList.Conditions["CritInPast8Sec"] = true
	}

	output["ScaledDamageEffect"] = 1.0

	// Calculate chance and multiplier for dealing triple damage on Normal and Crit
	output["TripleDamageChanceOnCrit"] = math.Min(skillModList.Sum("BASE", cfg, "TripleDamageChanceOnCrit"), 100)
	// #EVAL: `Sum(...) or 0 + X + Y` parses as `Sum(...) or (0+X+Y)`, and Sum
	// always returns a number, so the enemy and on-crit terms are dead here.
	output["TripleDamageChance"] = math.Min(skillModList.Sum("BASE", cfg, "TripleDamageChance"), 100)
	output["TripleDamageEffect"] = 2 * outNum(output, "TripleDamageChance") / 100

	// Calculate chance and multiplier for dealing double damage on Normal and Crit
	output["DoubleDamageChanceOnCrit"] = math.Min(skillModList.Sum("BASE", cfg, "DoubleDamageChanceOnCrit"), 100)
	selfDouble := 0.0
	if env.ModeEffective {
		selfDouble = enemyDB.Sum("BASE", cfg, "SelfDoubleDamageChance")
	}
	output["DoubleDamageChance"] = math.Min(skillModList.Sum("BASE", cfg, "DoubleDamageChance")+selfDouble+
		outNum(output, "DoubleDamageChanceOnCrit")*outNum(output, "CritChance")/100, 100)
	if truthy(globalOutput["IntimidatingUpTimeRatio"]) && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		output["DoubleDamageChance"] = 100.0
	} else if truthy(globalOutput["IntimidatingUpTimeRatio"]) {
		output["DoubleDamageChance"] = math.Min(outNum(output, "DoubleDamageChance")+outNum(globalOutput, "IntimidatingUpTimeRatio"), 100)
	}
	// Triple Damage overrides Double Damage. If you have both, it's the same
	// as just having Triple. We need to subtract the probability of both
	// happening in favor of Triple Damage
	if outNum(output, "TripleDamageChance") > 0 {
		output["DoubleDamageChance"] = math.Max(outNum(output, "DoubleDamageChance")-
			outNum(output, "TripleDamageChance")*outNum(output, "DoubleDamageChance")/100, 0)
	}
	output["DoubleDamageEffect"] = outNum(output, "DoubleDamageChance") / 100
	output["ScaledDamageEffect"] = outNum(output, "ScaledDamageEffect") * (1 + outNum(output, "DoubleDamageEffect") + outNum(output, "TripleDamageEffect"))

	speed := outNum(globalOutput, "Speed")
	if truthy(globalOutput["HitSpeed"]) {
		speed = outNum(globalOutput, "HitSpeed")
	}
	hitRate := outNum(output, "HitChance") / 100 * speed * anyNum(skillData["dpsMultiplier"])

	// Calculate culling DPS
	if truthy(env.ConfigInput["excludeCullingDPS"]) {
		globalOutput["CullPercent"] = 0.0
		globalOutput["CullMultiplier"] = 1.0
	} else {
		criticalCull := maxOr(skillModList, cfg, 0, "CriticalCullPercent")
		if criticalCull > 0 {
			criticalCull = math.Min(criticalCull, criticalCull*(1-math.Pow(1-outNum(output, "CritChance")/100, hitRate)))
		}
		regularCull := maxOr(skillModList, cfg, 0, "CullPercent")
		maxCullPercent := math.Max(criticalCull, regularCull)
		globalOutput["CullPercent"] = maxCullPercent
		globalOutput["CullMultiplier"] = 100 / (100 - maxCullPercent)
	}

	// Calculate reservation DPS
	globalOutput["ReservationDpsMultiplier"] = 100 / (100 - enemyDB.Sum("BASE", nil, "LifeReservationPercent"))

	env.runSkillFunc(c, "postCritFunc")

	// Added damage redirection (Cryogenesis). Convert all added damage mods
	// to the target type before the damage loop. Base Elemental Hit is
	// excluded per the node text.
	addedDamageRedirectType := ""
	if skillModList.Flag(cfg, "AllAddedDamageAsLightning") {
		addedDamageRedirectType = "Lightning"
	} else if skillModList.Flag(cfg, "AllAddedDamageAsCold") {
		addedDamageRedirectType = "Cold"
	}
	if addedDamageRedirectType != "" {
		for _, damageType := range dmgTypeList {
			if damageType == addedDamageRedirectType {
				continue
			}
			for _, suffix := range []string{"Min", "Max"} {
				for _, value := range skillModList.Tabulate("BASE", cfg, damageType+suffix) {
					mod := value.Mod
					if strings.Contains(mod.Source, "ElementalHit") {
						continue
					}
					args := []any{mod.Source, mod.Flags, mod.KeywordFlags, modparser.Tag{"type": "Cryogenesis Added Damage"}}
					args = append(args, mod.Tags...)
					skillModList.ConvertMod(damageType+suffix, newMod(addedDamageRedirectType+suffix, "BASE", mod.Value, args...))
				}
			}
		}
	}

	// Calculate base hit damage
	gel := activeSkill.ActiveEffect.GrantedEffectLevel
	for _, damageType := range dmgTypeList {
		damageTypeMin := damageType + "Min"
		damageTypeMax := damageType + "Max"
		baseMultiplier := 1.0
		if v, ok := lvlExtra(gel, "baseMultiplier"); ok {
			baseMultiplier = v
		} else if truthy(skillData["baseMultiplier"]) {
			baseMultiplier = anyNum(skillData["baseMultiplier"])
		}
		damageEffectiveness := 1.0
		if v, ok := lvlExtra(gel, "damageEffectiveness"); ok {
			damageEffectiveness = v
		} else if truthy(skillData["damageEffectiveness"]) {
			damageEffectiveness = anyNum(skillData["damageEffectiveness"])
		}
		addedMin := skillModList.Sum("BASE", cfg, damageTypeMin) + enemyDB.Sum("BASE", cfg, "Self"+damageTypeMin)
		addedMax := skillModList.Sum("BASE", cfg, damageTypeMax) + enemyDB.Sum("BASE", cfg, "Self"+damageTypeMax)
		addedMult := Mod(skillModList, cfg, "Added"+damageType+"Damage", "AddedDamage")
		baseMin := (anyNum(source[damageTypeMin])+anyNum(source[damageType+"BonusMin"]))*baseMultiplier + addedMin*damageEffectiveness*addedMult
		baseMax := (anyNum(source[damageTypeMax])+anyNum(source[damageType+"BonusMax"]))*baseMultiplier + addedMax*damageEffectiveness*addedMult
		output[damageTypeMin+"Base"] = baseMin
		output[damageTypeMax+"Base"] = baseMax
	}
}
