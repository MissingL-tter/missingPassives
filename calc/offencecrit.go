// CalcOffence.lua L2971-3300: crit chance/multiplier, double/triple damage,
// culling, the Cryogenesis added-damage redirect and the base hit damage.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// offenceFistOfWar ports L2986-3020 (the Fist of War branch of the Ruthless
// section), kept separate so the enclosing switch stays readable.
func (env *Env) offenceFistOfWar(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, output := c.skillModList, c.skillCfg, pass.output
	activeSkill, globalOutput := c.activeSkill, c.output

	globalOutput.SetN("FistOfWarDamageMultiplier", skillModList.Sum(modparser.Base, nil, "FistOfWarDamageMultiplier")/100)
	globalOutput.SetN("FistOfWarUptimeRatio", math.Min((1/globalOutput.N("Speed"))/globalOutput.N("FistOfWarCooldown"), 1)*100)
	globalOutput.Set("AvgFistOfWarDamage", globalOutput.Get("FistOfWarDamageMultiplier"))
	globalOutput.SetN("AvgFistOfWarDamageEffect", 1+globalOutput.N("FistOfWarDamageMultiplier")*(globalOutput.N("FistOfWarUptimeRatio")/100))
	globalOutput.SetN("MaxFistOfWarDamageEffect", 1+globalOutput.N("FistOfWarDamageMultiplier"))
	if activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		output.Set("FistOfWarDamageEffect", globalOutput.Get("MaxFistOfWarDamageEffect"))
		skillModList.AddMod(newModS("AreaOfEffect", modparser.Inc, modparser.Num(skillModList.Sum(modparser.Base, nil, "FistOfWarIncAoE")), "Max Fist of War Boosted AoE"))
	} else {
		output.Set("FistOfWarDamageEffect", globalOutput.Get("AvgFistOfWarDamageEffect"))
		skillModList.AddMod(newModS("AreaOfEffect", modparser.Inc, modparser.Num(math.Floor(skillModList.Sum(modparser.Base, nil, "FistOfWarIncAoE")/100*globalOutput.N("FistOfWarUptimeRatio"))), "Avg Fist Of War Boosted AoE"))
	}
	_ = skillCfg
	env.calcAreaOfEffect(c)
	globalOutput.SetN("TheoreticalOffensiveWarcryEffect", globalOutput.N("TheoreticalOffensiveWarcryEffect")*globalOutput.N("AvgFistOfWarDamageEffect"))
	globalOutput.SetN("TheoreticalMaxOffensiveWarcryEffect", globalOutput.N("TheoreticalMaxOffensiveWarcryEffect")*globalOutput.N("MaxFistOfWarDamageEffect"))
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
		env.explosiveArrowFunc(c, output)
	}

	// Calculate crit chance, crit multiplier, and their combined effect
	switch {
	case skillModList.Flag(cfg, "NeverCrit"):
		output.SetN("PreEffectiveCritChance", 0.0)
		output.SetN("CritChance", 0.0)
		output.SetN("CritMultiplier", 0.0)
		output.SetN("BonusCritDotMultiplier", 0.0)
		output.SetN("CritEffect", 1.0)
	case skillModList.Flag(cfg, "SpellSkillsCannotDealCriticalStrikesExceptOnFinalRepeat"):
		repeats := 1.0
		if output.Has("Repeats") {
			repeats = output.N("Repeats")
		}
		if repeats == 1 {
			output.SetN("PreEffectiveCritChance", 0.0)
			output.SetN("CritChance", 0.0)
			output.SetN("CritMultiplier", 0.0)
			output.SetN("BonusCritDotMultiplier", 0.0)
			output.SetN("CritEffect", 1.0)
		} else if skillModList.Flag(cfg, "SpellSkillsAlwaysDealCriticalStrikesOnFinalRepeat") {
			switch env.ConfigInput.RepeatMode {
			case "None":
				output.SetN("PreEffectiveCritChance", 0.0)
				output.SetN("CritChance", 0.0)
			case "AVERAGE":
				output.SetN("PreEffectiveCritChance", 100/repeats)
				output.SetN("CritChance", 100/repeats)
			default:
				output.SetN("PreEffectiveCritChance", 100.0)
				output.SetN("CritChance", 100.0)
			}
		}
	default:
		critOverride, _ := skillModList.Override(cfg, "CritChance")
		// destructive link
		if skillModList.Flag(cfg, "MainHandCritIsEqualToParent") {
			if actor.parent == nil {
				panic("offence: MainHandCritIsEqualToParent without a parent actor")
			}
			if mh := actor.parent.mainHand; mh != nil {
				critOverride = valueOfOut(mh.Get("CritChance"))
			} else {
				if wd := weaponOf(actor.parent.ms.WeaponData1); wd != nil && wd.CritChance.Set {
					critOverride = modparser.Num(wd.CritChance.V)
				}
			}
		} else if skillModList.Flag(cfg, "MainHandCritIsEqualToPartyMember") {
			panic("offence: MainHandCritIsEqualToPartyMember (party tab) unported")
		}
		baseCrit := 0.0
		if modparser.Truthy(critOverride) {
			baseCrit = valueNum(critOverride)
		} else if source.Flag("CritChance") {
			baseCrit = source.N("CritChance")
		}

		baseCritFromMainHand := skillModList.Flag(cfg, "BaseCritFromMainHand")
		baseCritFromParentMainHand := skillModList.Flag(cfg, "AttackCritIsEqualToParentMainHand")
		if baseCritFromMainHand {
			baseCrit = weaponCrit(weaponOf(actor.ms.WeaponData1))
		} else if baseCritFromParentMainHand {
			if actor.parent != nil && weaponOf(actor.parent.ms.WeaponData1) != nil {
				baseCrit = weaponCrit(weaponOf(actor.parent.ms.WeaponData1))
			}
		}

		if modparser.Truthy(critOverride) && valueNum(critOverride) == 100 {
			output.SetN("PreEffectiveCritChance", 100.0)
			output.SetN("PreBifurcateCritChance", 100.0)
			output.SetN("CritChance", 100.0)
		} else {
			base, inc, more := 0.0, 0.0, 1.0
			if !modparser.Truthy(critOverride) {
				selfBase, selfInc := 0.0, 0.0
				if env.ModeEffective {
					selfBase = enemyDB.Sum(modparser.Base, nil, "SelfCritChance")
					selfInc = enemyDB.Sum(modparser.Inc, nil, "SelfCritChance")
				}
				base = skillModList.Sum(modparser.Base, cfg, "CritChance") + selfBase
				inc = skillModList.Sum(modparser.Inc, cfg, "CritChance") + selfInc
				more = skillModList.More(cfg, "CritChance")
			}
			critChance := (baseCrit + base) * (1 + inc/100) * more
			cap := skillModList.Sum(modparser.Base, cfg, "CritChanceCap")
			if ov, ok := skillModList.Override(nil, "CritChanceCap"); ok {
				cap = valueNum(ov)
			}
			critChance = math.Min(critChance, cap)
			if (baseCrit + base) > 0 {
				critChance = math.Max(critChance, 0)
			}
			output.SetN("PreEffectiveCritChance", critChance)
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
			output.SetN("PreBifurcateCritChance", critChance)
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
				critChance = critChance * output.N("AccuracyHitChance") / 100
			}
			output.SetN("CritChance", critChance)
		}
	}
	if !output.Has("CritEffect") {
		if skillModList.Flag(cfg, "NoCritMultiplier") {
			output.SetN("CritMultiplier", 1.0)
		} else {
			extraDamage := skillModList.Sum(modparser.Base, cfg, "CritMultiplier") / 100
			if multiOverride, ok := skillModList.Override(skillCfg, "CritMultiplier"); ok {
				extraDamage = (valueNum(multiOverride) - 100) / 100
			}
			if env.ModeEffective {
				enemyInc := 1 + enemyDB.Sum(modparser.Inc, nil, "SelfCritMultiplier")/100
				extraDamage += enemyDB.Sum(modparser.Base, nil, "SelfCritMultiplier") / 100
				extraDamage = util.RoundHalfUp(extraDamage*enemyInc, 2)
			}
			// if crit bifurcates are enabled, roll for crit twice and add
			// multiplier for each
			critOverride, _ := skillModList.Override(cfg, "CritChance")
			if env.ModeEffective && skillModList.Flag(cfg, "BifurcateCrit") && output.Has("PreBifurcateCritChance") &&
				!(modparser.Truthy(critOverride) && valueNum(critOverride) == 100) {
				critChancePercentage := output.N("PreBifurcateCritChance")
				bifurcateMultiChance := critChancePercentage * critChancePercentage / 100
				effectiveCritChance := output.N("CritChance")
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
				output.SetN("CritBifurcates", 1+conditionalBifurcateChance)
				extraDamage = extraDamage + conditionalBifurcateChance*extraDamage
				// mod doesn't affect output and is purely descriptive
				skillModList.AddMod(newModSF("CritMultiplier", modparser.More, modparser.Num(floorDec(conditionalBifurcateChance*100, 2)), "Bifurcated Crit Damage Bonus", modparser.FlagHit, modparser.KeywordNone))
			}
			output.SetN("CritMultiplier", 1+math.Max(0, extraDamage))
		}
		critChancePercentage := output.N("CritChance") / 100
		output.SetN("CritEffect", 1-critChancePercentage+critChancePercentage*output.N("CritMultiplier"))
		output.SetN("BonusCritDotMultiplier", (skillModList.Sum(modparser.Base, cfg, "CritMultiplier")-50)*
			skillModList.Sum(modparser.Base, cfg, "CritMultiplierAppliesToDegen")/10000)
	}
	if output.N("CritChance") != 0 {
		skillModList.Conditions.Set("CritInPast8Sec", true)
	}

	output.SetN("ScaledDamageEffect", 1.0)

	// Calculate chance and multiplier for dealing triple damage on Normal and Crit
	output.SetN("TripleDamageChanceOnCrit", math.Min(skillModList.Sum(modparser.Base, cfg, "TripleDamageChanceOnCrit"), 100))
	// #EVAL: `Sum(...) or 0 + X + Y` parses as `Sum(...) or (0+X+Y)`, and Sum
	// always returns a number, so the enemy and on-crit terms are dead here.
	output.SetN("TripleDamageChance", math.Min(skillModList.Sum(modparser.Base, cfg, "TripleDamageChance"), 100))
	output.SetN("TripleDamageEffect", 2*output.N("TripleDamageChance")/100)

	// Calculate chance and multiplier for dealing double damage on Normal and Crit
	output.SetN("DoubleDamageChanceOnCrit", math.Min(skillModList.Sum(modparser.Base, cfg, "DoubleDamageChanceOnCrit"), 100))
	selfDouble := 0.0
	if env.ModeEffective {
		selfDouble = enemyDB.Sum(modparser.Base, cfg, "SelfDoubleDamageChance")
	}
	output.SetN("DoubleDamageChance", math.Min(skillModList.Sum(modparser.Base, cfg, "DoubleDamageChance")+selfDouble+
		output.N("DoubleDamageChanceOnCrit")*output.N("CritChance")/100, 100))
	if globalOutput.Has("IntimidatingUpTimeRatio") && activeSkill.SkillModList.Flag(nil, "Condition:WarcryMaxHit") {
		output.SetN("DoubleDamageChance", 100.0)
	} else if globalOutput.Has("IntimidatingUpTimeRatio") {
		output.SetN("DoubleDamageChance", math.Min(output.N("DoubleDamageChance")+globalOutput.N("IntimidatingUpTimeRatio"), 100))
	}
	// Triple Damage overrides Double Damage. If you have both, it's the same
	// as just having Triple. We need to subtract the probability of both
	// happening in favor of Triple Damage
	if output.N("TripleDamageChance") > 0 {
		output.SetN("DoubleDamageChance", math.Max(output.N("DoubleDamageChance")-
			output.N("TripleDamageChance")*output.N("DoubleDamageChance")/100, 0))
	}
	output.SetN("DoubleDamageEffect", output.N("DoubleDamageChance")/100)
	output.SetN("ScaledDamageEffect", output.N("ScaledDamageEffect")*(1+output.N("DoubleDamageEffect")+output.N("TripleDamageEffect")))

	speed := globalOutput.N("Speed")
	if globalOutput.Has("HitSpeed") {
		speed = globalOutput.N("HitSpeed")
	}
	hitRate := output.N("HitChance") / 100 * speed * skillData.N("dpsMultiplier")

	// Calculate culling DPS
	if env.ConfigInput.ExcludeCullingDPS {
		globalOutput.SetN("CullPercent", 0.0)
		globalOutput.SetN("CullMultiplier", 1.0)
	} else {
		criticalCull := maxOr(skillModList, cfg, 0, "CriticalCullPercent")
		if criticalCull > 0 {
			criticalCull = math.Min(criticalCull, criticalCull*(1-math.Pow(1-output.N("CritChance")/100, hitRate)))
		}
		regularCull := maxOr(skillModList, cfg, 0, "CullPercent")
		maxCullPercent := math.Max(criticalCull, regularCull)
		globalOutput.SetN("CullPercent", maxCullPercent)
		globalOutput.SetN("CullMultiplier", 100/(100-maxCullPercent))
	}

	// Calculate reservation DPS
	globalOutput.SetN("ReservationDpsMultiplier", 100/(100-enemyDB.Sum(modparser.Base, nil, "LifeReservationPercent")))

	env.runSkillFunc(c, data.CallbackPostCrit)

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
				for _, value := range skillModList.Tabulate(modparser.Base, cfg, damageType+suffix) {
					mod := value.Mod
					if strings.Contains(mod.Source, "ElementalHit") {
						continue
					}
					args := []modparser.Tag{&modparser.MarkerTag{Marker: modparser.TagCryogenesisAddedDamage}}
					args = append(args, mod.Tags...)
					skillModList.ConvertMod(damageType+suffix, modparser.NewModFull(addedDamageRedirectType+suffix, modparser.Base, mod.Value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, args...))
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
		} else if skillData.Flag("baseMultiplier") {
			baseMultiplier = skillData.N("baseMultiplier")
		}
		damageEffectiveness := 1.0
		if v, ok := lvlExtra(gel, "damageEffectiveness"); ok {
			damageEffectiveness = v
		} else if skillData.Has("damageEffectiveness") {
			damageEffectiveness = skillData.N("damageEffectiveness")
		}
		addedMin := skillModList.Sum(modparser.Base, cfg, damageTypeMin) + enemyDB.Sum(modparser.Base, cfg, "Self"+damageTypeMin)
		addedMax := skillModList.Sum(modparser.Base, cfg, damageTypeMax) + enemyDB.Sum(modparser.Base, cfg, "Self"+damageTypeMax)
		addedMult := Mod(skillModList, cfg, "Added"+damageType+"Damage", "AddedDamage")
		baseMin := (source.N(damageTypeMin)+source.N(damageType+"BonusMin"))*baseMultiplier + addedMin*damageEffectiveness*addedMult
		baseMax := (source.N(damageTypeMax)+source.N(damageType+"BonusMax"))*baseMultiplier + addedMax*damageEffectiveness*addedMult
		output.SetN(damageTypeMin+"Base", baseMin)
		output.SetN(damageTypeMax+"Base", baseMax)
	}
}
