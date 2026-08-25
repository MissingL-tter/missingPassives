// CalcOffence.lua L2107-2421: the per-pass hit-chance and attack/cast speed
// calculation, including the sustainable-trauma solve.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

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

		if truthy(skillData["averageBurstHits"]) {
			output["AverageBurstHits"] = skillData["averageBurstHits"]
		} else if truthy(output["Repeats"]) && anyNum(output["Repeats"]) > 1 {
			output["AverageBurstHits"] = output["Repeats"]
		}

		// Calculate hit chance
		base := skillModList.Sum("BASE", cfg, "Accuracy")
		baseVsEnemy := skillModList.Sum("BASE", cfg, "Accuracy", "AccuracyVsEnemy")
		inc := skillModList.Sum("INC", cfg, "Accuracy")
		incVsEnemy := skillModList.Sum("INC", cfg, "Accuracy", "AccuracyVsEnemy")
		// #EVAL: the reference calls More("MORE", cfg, "Accuracy") — the
		// "MORE" string lands in the cfg slot and the real cfg becomes a
		// (never-matching) modifier name, so this is a cfg-less More.
		more := skillModList.More(&modstore.Cfg{}, "Accuracy")
		moreVsEnemy := skillModList.More(&modstore.Cfg{}, "Accuracy", "AccuracyVsEnemy")

		output["Accuracy"] = math.Max(0, math.Floor(base*(1+inc/100)*more))
		accuracyVsEnemy := math.Max(0, math.Floor(baseVsEnemy*(1+incVsEnemy/100)*moreVsEnemy))
		if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.label == "Main Hand" {
			storedMainHandAccuracy = outNum(output, "Accuracy")
			storedMainHandAccuracyVsEnemy = accuracyVsEnemy
			haveStoredMainHandAccuracy = true
		} else if skillModList.Flag(nil, "Condition:OffHandAccuracyIsMainHandAccuracy") && pass.label == "Off Hand" && haveStoredMainHandAccuracy {
			output["Accuracy"] = storedMainHandAccuracy
			accuracyVsEnemy = storedMainHandAccuracyVsEnemy
		}
		if !isAttack || skillModList.Flag(cfg, "CannotBeEvaded") || truthy(skillData["cannotBeEvaded"]) ||
			(env.ModeEffective && enemyDB.Flag(nil, "CannotEvade")) {
			output["AccuracyHitChance"] = 100.0
		} else {
			enemyEvasion := math.Max(roundDec(Val(enemyDB, "Evasion", nil), 0), 0)
			output["AccuracyHitChance"] = hitChance(enemyEvasion, accuracyVsEnemy) * Mod(skillModList, cfg, "HitChance")
		}
		// enemy block chance
		output["enemyBlockChance"] = math.Max(math.Min(enemyDB.Sum("BASE", cfg, "BlockChance"), 100)-skillModList.Sum("BASE", cfg, "reduceEnemyBlock"), 0)
		if enemyDB.Flag(nil, "CannotBlockAttacks") && isAttack {
			output["enemyBlockChance"] = 0.0
		}

		output["HitChance"] = outNum(output, "AccuracyHitChance") * (1 - outNum(output, "enemyBlockChance")/100)
		if outNum(output, "enemyBlockChance") > 0 && !isAttack {
			globalOutput["enemyHasSpellBlock"] = true
		}

		// Check Precise Technique Keystone condition per pass as MH/OH might
		// have different values
		condName := strings.Replace(pass.label, " ", "", -1) + "AccRatingHigherThanMaxLife"
		skillModList.Conditions[condName] = outNum(output, "Accuracy") > outNum(env.Player.Output, "Life")

		// Calculate attack/cast speed
		castTime := 0.0
		hasCastTime := false
		if ct := activeSkill.ActiveEffect.GrantedEffect.CastTime; ct != nil {
			castTime, hasCastTime = *ct, true
		}
		switch {
		case hasCastTime && castTime == 0 && !truthy(skillData["castTimeOverride"]) && !truthy(skillData["triggered"]):
			output["Time"] = 0.0
			output["Speed"] = 0.0
		case truthy(skillData["timeOverride"]):
			output["Time"] = skillData["timeOverride"]
			output["Speed"] = 1 / outNum(output, "Time")
		case truthy(skillData["fixedCastTime"]):
			output["Time"] = castTime
			output["Speed"] = 1 / outNum(output, "Time")
		case truthy(skillData["triggerTime"]) && truthy(skillData["triggered"]):
			activeSkillsLinked := skillModList.Sum("BASE", cfg, "ActiveSkillsLinkedToTrigger")
			t := anyNum(skillData["triggerTime"]) / (1 + skillModList.Sum("INC", cfg, "CooldownRecovery")/100)
			if activeSkillsLinked > 0 {
				t *= activeSkillsLinked
			}
			output["Time"] = t
			output["TriggerTime"] = t
			output["Speed"] = 1 / t
		case truthy(skillData["triggerRate"]) && truthy(skillData["triggered"]):
			output["Time"] = 1 / anyNum(skillData["triggerRate"])
			output["TriggerTime"] = output["Time"]
			output["Speed"] = anyNum(skillData["triggerRate"])
			skillData["showAverage"] = false
		default:
			var baseTime float64
			if isAttack {
				if truthy(skillData["attackSpeedMultiplier"]) && truthy(source["AttackRate"]) {
					source["AttackRate"] = anyNum(source["AttackRate"]) * (1 + anyNum(skillData["attackSpeedMultiplier"])/100)
				}
				attackRate := 1.0
				if truthy(source["AttackRate"]) {
					attackRate = anyNum(source["AttackRate"])
				}
				switch {
				case truthy(skillData["castTimeOverridesAttackTime"]):
					// Skill is overriding weapon attack speed
					baseTime = castTime / (1 + anyNum(source["AttackSpeedInc"])/100)
				case Mod(skillModList, skillCfg, "SkillAttackTime") > 0:
					baseTime = (1/attackRate + skillModList.Sum("BASE", cfg, "Speed")) * Mod(skillModList, skillCfg, "SkillAttackTime")
				default:
					baseTime = 1/attackRate + skillModList.Sum("BASE", cfg, "Speed")
				}
			} else {
				switch {
				case truthy(skillData["castTimeOverride"]):
					baseTime = anyNum(skillData["castTimeOverride"])
				case hasCastTime:
					// a present 0 wins the `or` chain, as in Lua
					baseTime = castTime
				default:
					baseTime = 1
				}
			}
			more := skillModList.More(cfg, "Speed")
			repeats := 1.0
			if truthy(globalOutput["Repeats"]) {
				repeats = anyNum(globalOutput["Repeats"])
			}
			output["Repeats"] = repeats

			// Calculates the max number of trauma stacks you can sustain
			if skillModList.Flag(nil, "HasTrauma") {
				effectiveAttackRateCap := data.Misc.ServerTickRate * repeats
				duration := skillModList.Sum("BASE", cfg, "TraumaDuration") * Mod(skillModList, skillCfg, "Duration")
				traumaPerAttack := 1 + math.Min(skillModList.Sum("BASE", cfg, "ExtraTrauma"), 100)/100
				incAttackSpeedPerTrauma := skillModList.Sum("INC", skillCfg, "SpeedPerTrauma")
				// compute trauma using an exact form.
				configTrauma := skillModList.Sum("BASE", skillCfg, "Multiplier:TraumaStacks")
				// remove trauma attack speed added by config.
				incTrauma := skillModList.Sum("INC", cfg, "Speed") - incAttackSpeedPerTrauma*configTrauma
				attackSpeedBeforeInc := 1 / baseTime * outNum(globalOutput, "ActionSpeedMod") * more
				incAttackSpeedPerTraumaCap := (effectiveAttackRateCap - attackSpeedBeforeInc*(1+incTrauma/100)) / attackSpeedBeforeInc * 100
				traumaRateBeforeInc := traumaPerAttack * (outNum(output, "HitChance") / 100) * attackSpeedBeforeInc / repeats
				trauma := traumaRateBeforeInc * (1 + incTrauma/100) / (1/duration - traumaRateBeforeInc*incAttackSpeedPerTrauma/100)
				if trauma < 0 || incAttackSpeedPerTrauma*trauma > incAttackSpeedPerTraumaCap {
					// invalid long term trauma generation as maximum attack
					// rate is once per tick.
					trauma = traumaPerAttack * (outNum(output, "HitChance") / 100) * effectiveAttackRateCap / repeats * duration
				}
				if skillFlags["bothWeaponAttack"] {
					// halve trauma rate when dual wielding so pass 2 doesn't
					// double your trauma rate
					trauma = trauma / 2
				}
				skillModList.AddMod(newMod("Multiplier:SustainableTraumaStacks", "BASE", trauma, "Maximum Sustainable Trauma Stacks"))
			}
			if skillModList.Sum("BASE", skillCfg, "Multiplier:TraumaStacks") == 0 {
				skillModList.AddMod(newMod("Multiplier:TraumaStacks", "BASE",
					skillModList.Sum("BASE", skillCfg, "Multiplier:SustainableTraumaStacks"), "Maximum Sustainable Trauma Stacks"))
			}
			incSpeed := skillModList.Sum("INC", cfg, "Speed")

			if skillFlags["warcry"] {
				output["Speed"] = 1 / outNum(globalOutput, "WarcryCastTime")
			} else {
				output["Speed"] = 1 / (baseTime/roundDec((1+incSpeed/100)*more, 2) +
					skillModList.Sum("BASE", cfg, "TotalAttackTime") + skillModList.Sum("BASE", cfg, "TotalCastTime"))
			}
			output["CastRate"] = output["Speed"]
			if skillFlags["selfCast"] {
				// Self-cast skill; apply action speed
				output["Speed"] = outNum(output, "Speed") * outNum(globalOutput, "ActionSpeedMod")
				output["CastRate"] = output["Speed"]
			}
			if skillFlags["totem"] {
				// Totem skill. Apply action speed
				totemActionSpeed := 1 + modDB.Sum("INC", nil, "TotemActionSpeed")/100
				output["TotemActionSpeed"] = totemActionSpeed
				output["Speed"] = outNum(output, "Speed") * totemActionSpeed
				output["CastRate"] = output["Speed"]
				if truthy(skillData["totemFireOnce"]) {
					output["HitTime"] = 1/outNum(output, "Speed") + outNum(globalOutput, "TotemPlacementTime")
					output["HitSpeed"] = 1 / outNum(output, "HitTime")
				}
			}
			if truthy(globalOutput["Cooldown"]) {
				output["Cooldown"] = globalOutput["Cooldown"]
				output["Speed"] = math.Min(outNum(output, "Speed"), 1/outNum(output, "Cooldown")*repeats)
			}
			if truthy(output["Cooldown"]) && skillFlags["selfCast"] {
				skillFlags["notAverage"] = true
				skillFlags["showAverage"] = false
				skillData["showAverage"] = false
			}
			if !activeSkill.SkillTypes[modparser.SkillType.Channel] {
				output["Speed"] = math.Min(outNum(output, "Speed"), data.Misc.ServerTickRate*repeats)
			}
			if outNum(output, "Speed") == 0 {
				output["Time"] = 0.0
			} else {
				output["Time"] = 1 / outNum(output, "Speed")
			}
		}
		if truthy(skillData["hitTimeOverride"]) && !truthy(skillData["triggeredOnDeath"]) {
			output["HitTime"] = skillData["hitTimeOverride"]
			output["HitSpeed"] = 1 / outNum(output, "HitTime")
			// Brands always have hitTimeOverride
			if activeSkill.SkillTypes[modparser.SkillType.Brand] && !skillModList.Flag(nil, "UnlimitedBrandDuration") {
				output["BrandTicks"] = math.Floor(outNum(output, "Duration") * outNum(output, "HitSpeed"))
			}
		} else if truthy(skillData["hitTimeMultiplier"]) && truthy(output["Time"]) && !truthy(skillData["triggeredOnDeath"]) {
			output["HitTime"] = outNum(output, "Time") * anyNum(skillData["hitTimeMultiplier"])
			if truthy(output["Cooldown"]) && truthy(skillData["triggered"]) {
				output["HitSpeed"] = 1 / math.Max(outNum(output, "HitTime"), outNum(output, "Cooldown"))
			} else if truthy(output["Cooldown"]) {
				output["HitSpeed"] = 1 / (outNum(output, "HitTime") + outNum(output, "Cooldown"))
			} else {
				output["HitSpeed"] = 1 / outNum(output, "HitTime")
			}
		}
	}

	env.offenceMiscDPS(c)
}
