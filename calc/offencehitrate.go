// CalcOffence.lua L2107-2421: the per-pass hit-chance and attack/cast speed
// calculation, including the sustainable-trauma solve.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// offenceHitRate ports L2107-2421.
func (env *Env) offenceHitRate(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, isAttack := c.activeSkill, c.isAttack
	globalOutput := c.output

	var storedMainHandAccuracy, storedMainHandAccuracyVsEnemy float64
	haveStoredMainHandAccuracy := false

	// Calculate how often you hit (speed, accuracy, block, etc)
	for _, pass := range c.passList {
		source, output, cfg := pass.source, pass.output, pass.cfg

		if skillData.Flag("averageBurstHits") {
			output.Set("AverageBurstHits", skillData.Get("averageBurstHits"))
		} else if output.Has("Repeats") && output.N("Repeats") > 1 {
			output.Set("AverageBurstHits", output.Get("Repeats"))
		}

		// Calculate hit chance
		base := skillModList.Sum(modparser.Base, cfg, "Accuracy")
		baseVsEnemy := skillModList.Sum(modparser.Base, cfg, "Accuracy", "AccuracyVsEnemy")
		inc := skillModList.Sum(modparser.Inc, cfg, "Accuracy")
		incVsEnemy := skillModList.Sum(modparser.Inc, cfg, "Accuracy", "AccuracyVsEnemy")
		// #EVAL: the reference calls More("MORE", cfg, "Accuracy") — the
		// "MORE" string lands in the cfg slot and the real cfg becomes a
		// (never-matching) modifier name, so this is a cfg-less More.
		more := skillModList.More(&modstore.Cfg{}, "Accuracy")
		moreVsEnemy := skillModList.More(&modstore.Cfg{}, "Accuracy", "AccuracyVsEnemy")

		output.SetN("Accuracy", math.Max(0, math.Floor(base*(1+inc/100)*more)))
		accuracyVsEnemy := math.Max(0, math.Floor(baseVsEnemy*(1+incVsEnemy/100)*moreVsEnemy))
		if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.label == "Main Hand" {
			storedMainHandAccuracy = output.N("Accuracy")
			storedMainHandAccuracyVsEnemy = accuracyVsEnemy
			haveStoredMainHandAccuracy = true
		} else if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.label == "Off Hand" && haveStoredMainHandAccuracy {
			output.SetN("Accuracy", storedMainHandAccuracy)
			accuracyVsEnemy = storedMainHandAccuracyVsEnemy
		}
		if !isAttack || skillModList.Flag(cfg, "CannotBeEvaded") || skillData.Flag("cannotBeEvaded") ||
			(env.ModeEffective && enemyDB.Flag(nil, "CannotEvade")) {
			output.SetN("AccuracyHitChance", 100.0)
		} else {
			enemyEvasion := math.Max(util.RoundHalfUp(Val(enemyDB, "Evasion", nil), 0), 0)
			output.SetN("AccuracyHitChance", hitChance(enemyEvasion, accuracyVsEnemy)*Mod(skillModList, cfg, "HitChance"))
		}
		// enemy block chance
		output.SetN("enemyBlockChance", math.Max(math.Min(enemyDB.Sum(modparser.Base, cfg, "BlockChance"), 100)-skillModList.Sum(modparser.Base, cfg, "reduceEnemyBlock"), 0))
		if enemyDB.Flag(nil, "CannotBlockAttacks") && isAttack {
			output.SetN("enemyBlockChance", 0.0)
		}

		output.SetN("HitChance", output.N("AccuracyHitChance")*(1-output.N("enemyBlockChance")/100))
		if output.N("enemyBlockChance") > 0 && !isAttack {
			globalOutput.SetFlag("enemyHasSpellBlock", true)
		}

		// Check Precise Technique Keystone condition per pass as MH/OH might
		// have different values
		condName := strings.Replace(pass.label, " ", "", -1) + "AccRatingHigherThanMaxLife"
		skillModList.Conditions.Set(condName, output.N("Accuracy") > env.Player.Output.N("Life"))

		// Calculate attack/cast speed
		castTime := 0.0
		hasCastTime := false
		if ct := activeSkill.ActiveEffect.GrantedEffect.CastTime; ct != nil {
			castTime, hasCastTime = *ct, true
		}
		switch {
		case hasCastTime && castTime == 0 && !skillData.Flag("castTimeOverride") && !skillData.Flag("triggered"):
			output.SetN("Time", 0.0)
			output.SetN("Speed", 0.0)
		case skillData.Has("timeOverride"):
			output.Set("Time", skillData.Get("timeOverride"))
			output.SetN("Speed", 1/output.N("Time"))
		case skillData.Flag("fixedCastTime"):
			output.SetN("Time", castTime)
			output.SetN("Speed", 1/output.N("Time"))
		case skillData.Has("triggerTime") && skillData.Flag("triggered"):
			activeSkillsLinked := skillModList.Sum(modparser.Base, cfg, "ActiveSkillsLinkedToTrigger")
			t := skillData.N("triggerTime") / (1 + skillModList.Sum(modparser.Inc, cfg, "CooldownRecovery")/100)
			if activeSkillsLinked > 0 {
				t *= activeSkillsLinked
			}
			output.SetN("Time", t)
			output.SetN("TriggerTime", t)
			output.SetN("Speed", 1/t)
		case skillData.Flag("triggerRate") && skillData.Flag("triggered"):
			output.SetN("Time", 1/skillData.N("triggerRate"))
			output.Set("TriggerTime", output.Get("Time"))
			output.SetN("Speed", skillData.N("triggerRate"))
			skillData.SetFlag("showAverage", false)
		default:
			var baseTime float64
			if isAttack {
				if skillData.Has("attackSpeedMultiplier") && source.Has("AttackRate") {
					source.SetN("AttackRate", source.N("AttackRate")*(1+skillData.N("attackSpeedMultiplier")/100))
				}
				attackRate := 1.0
				if source.Has("AttackRate") {
					attackRate = source.N("AttackRate")
				}
				switch {
				case skillData.Flag("castTimeOverridesAttackTime"):
					// Skill is overriding weapon attack speed
					baseTime = castTime / (1 + source.N("AttackSpeedInc")/100)
				case Mod(skillModList, skillCfg, "SkillAttackTime") > 0:
					baseTime = (1/attackRate + skillModList.Sum(modparser.Base, cfg, "Speed")) * Mod(skillModList, skillCfg, "SkillAttackTime")
				default:
					baseTime = 1/attackRate + skillModList.Sum(modparser.Base, cfg, "Speed")
				}
			} else {
				switch {
				case skillData.Flag("castTimeOverride"):
					baseTime = skillData.N("castTimeOverride")
				case hasCastTime:
					// a present 0 wins the `or` chain, as in Lua
					baseTime = castTime
				default:
					baseTime = 1
				}
			}
			more := skillModList.More(cfg, "Speed")
			repeats := 1.0
			if globalOutput.Has("Repeats") {
				repeats = globalOutput.N("Repeats")
			}
			output.SetN("Repeats", repeats)

			// Calculates the max number of trauma stacks you can sustain
			if skillModList.Flag(nil, "HasTrauma") {
				effectiveAttackRateCap := data.Misc.ServerTickRate * repeats
				duration := skillModList.Sum(modparser.Base, cfg, "TraumaDuration") * Mod(skillModList, skillCfg, "Duration")
				traumaPerAttack := 1 + math.Min(skillModList.Sum(modparser.Base, cfg, "ExtraTrauma"), 100)/100
				incAttackSpeedPerTrauma := skillModList.Sum(modparser.Inc, skillCfg, "SpeedPerTrauma")
				// compute trauma using an exact form.
				configTrauma := skillModList.Sum(modparser.Base, skillCfg, "Multiplier:TraumaStacks")
				// remove trauma attack speed added by config.
				incTrauma := skillModList.Sum(modparser.Inc, cfg, "Speed") - incAttackSpeedPerTrauma*configTrauma
				attackSpeedBeforeInc := 1 / baseTime * globalOutput.N("ActionSpeedMod") * more
				incAttackSpeedPerTraumaCap := (effectiveAttackRateCap - attackSpeedBeforeInc*(1+incTrauma/100)) / attackSpeedBeforeInc * 100
				traumaRateBeforeInc := traumaPerAttack * (output.N("HitChance") / 100) * attackSpeedBeforeInc / repeats
				trauma := traumaRateBeforeInc * (1 + incTrauma/100) / (1/duration - traumaRateBeforeInc*incAttackSpeedPerTrauma/100)
				if trauma < 0 || incAttackSpeedPerTrauma*trauma > incAttackSpeedPerTraumaCap {
					// invalid long term trauma generation as maximum attack
					// rate is once per tick.
					trauma = traumaPerAttack * (output.N("HitChance") / 100) * effectiveAttackRateCap / repeats * duration
				}
				if skillFlags["bothWeaponAttack"] {
					// halve trauma rate when dual wielding so pass 2 doesn't
					// double your trauma rate
					trauma = trauma / 2
				}
				skillModList.AddMod(newModS("Multiplier:SustainableTraumaStacks", modparser.Base, modparser.Num(trauma), "Maximum Sustainable Trauma Stacks"))
			}
			if skillModList.Sum(modparser.Base, skillCfg, "Multiplier:TraumaStacks") == 0 {
				skillModList.AddMod(newModS("Multiplier:TraumaStacks", modparser.Base, modparser.Num(skillModList.Sum(modparser.Base, skillCfg, "Multiplier:SustainableTraumaStacks")), "Maximum Sustainable Trauma Stacks"))
			}
			incSpeed := skillModList.Sum(modparser.Inc, cfg, "Speed")

			if skillFlags["warcry"] {
				output.SetN("Speed", 1/globalOutput.N("WarcryCastTime"))
			} else {
				output.SetN("Speed", 1/(baseTime/util.RoundHalfUp((1+incSpeed/100)*more, 2)+
					skillModList.Sum(modparser.Base, cfg, "TotalAttackTime")+skillModList.Sum(modparser.Base, cfg, "TotalCastTime")))
			}
			output.Set("CastRate", output.Get("Speed"))
			if skillFlags["selfCast"] {
				// Self-cast skill; apply action speed
				output.SetN("Speed", output.N("Speed")*globalOutput.N("ActionSpeedMod"))
				output.Set("CastRate", output.Get("Speed"))
			}
			if skillFlags["totem"] {
				// Totem skill. Apply action speed
				totemActionSpeed := 1 + modDB.Sum(modparser.Inc, nil, "TotemActionSpeed")/100
				output.SetN("TotemActionSpeed", totemActionSpeed)
				output.SetN("Speed", output.N("Speed")*totemActionSpeed)
				output.Set("CastRate", output.Get("Speed"))
				if skillData.Flag("totemFireOnce") {
					output.SetN("HitTime", 1/output.N("Speed")+globalOutput.N("TotemPlacementTime"))
					output.SetN("HitSpeed", 1/output.N("HitTime"))
				}
			}
			if globalOutput.Flag("Cooldown") {
				output.Set("Cooldown", globalOutput.Get("Cooldown"))
				output.SetN("Speed", math.Min(output.N("Speed"), 1/output.N("Cooldown")*repeats))
			}
			if output.Flag("Cooldown") && skillFlags["selfCast"] {
				skillFlags["notAverage"] = true
				skillFlags["showAverage"] = false
				skillData.SetFlag("showAverage", false)
			}
			if !activeSkill.SkillTypes[modparser.SkillTypeChannel] {
				output.SetN("Speed", math.Min(output.N("Speed"), data.Misc.ServerTickRate*repeats))
			}
			if output.N("Speed") == 0 {
				output.SetN("Time", 0.0)
			} else {
				output.SetN("Time", 1/output.N("Speed"))
			}
		}
		if skillData.Flag("hitTimeOverride") && !skillData.Flag("triggeredOnDeath") {
			output.Set("HitTime", skillData.Get("hitTimeOverride"))
			output.SetN("HitSpeed", 1/output.N("HitTime"))
			// Brands always have hitTimeOverride
			if activeSkill.SkillTypes[modparser.SkillTypeBrand] && !skillModList.Flag(nil, "UnlimitedBrandDuration") {
				output.SetN("BrandTicks", math.Floor(output.N("Duration")*output.N("HitSpeed")))
			}
		} else if skillData.Has("hitTimeMultiplier") && output.Flag("Time") && !skillData.Flag("triggeredOnDeath") {
			output.SetN("HitTime", output.N("Time")*skillData.N("hitTimeMultiplier"))
			if output.Flag("Cooldown") && skillData.Flag("triggered") {
				output.SetN("HitSpeed", 1/math.Max(output.N("HitTime"), output.N("Cooldown")))
			} else if output.Flag("Cooldown") {
				output.SetN("HitSpeed", 1/(output.N("HitTime")+output.N("Cooldown")))
			} else {
				output.SetN("HitSpeed", 1/output.N("HitTime"))
			}
		}
	}

	env.offenceMiscDPS(c)
}
