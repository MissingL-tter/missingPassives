// CalcOffence.lua L2422-2540: the misc DPS multipliers (skill DPS mods,
// brands, returning projectiles), the main/off-hand speed combine and the
// quantity multiplier.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// offenceMiscDPS ports L2422-2540.
func (env *Env) offenceMiscDPS(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, modDB := c.skillFlags, c.output, c.modDB
	activeSkill, isAttack := c.activeSkill, c.isAttack

	// Other Misc DPS multipliers (like custom source)
	dpsMultiplier := 1.0
	if skillData.Has("dpsMultiplier") {
		dpsMultiplier = skillData.N("dpsMultiplier")
	}
	dpsMultiplier = dpsMultiplier * (1 + skillModList.Sum(modparser.Inc, skillCfg, "DPS")/100) * skillModList.More(skillCfg, "DPS")
	skillData.SetN("dpsMultiplier", dpsMultiplier)
	if activeSkill.SkillTypes[modparser.SkillTypeBrand] && !skillData.Flag("countsAttachedBrandsInDamage") {
		dpsMultiplier *= output.N("AttachedBrandCount")
		skillData.SetN("dpsMultiplier", dpsMultiplier)
		skillDPSMult := 1.0
		if output.Has("SkillDPSMultiplier") {
			skillDPSMult = output.N("SkillDPSMultiplier")
		}
		output.SetN("SkillDPSMultiplier", skillDPSMult*output.N("AttachedBrandCount"))
	}
	if env.ConfigInput.RepeatMode == "FINAL" || skillModList.Flag(nil, "OnlyFinalRepeat") {
		repeats := 1.0
		if output.Has("Repeats") {
			repeats = output.N("Repeats")
		}
		dpsMultiplier /= repeats
		skillData.SetN("dpsMultiplier", dpsMultiplier)
	}
	// Returning Projectiles hit the enemy again on the way back, at reduced
	// damage for some sources of Return. Skipped while viewing a skill part
	// that already represents the returning Projectile, as that would count
	// it twice.
	if skillFlags["projectile"] && skillModList.Flag(skillCfg, "ProjectilesReturn") &&
		!activeSkill.SkillTypes[modparser.SkillTypeProjectileCannotReturn] &&
		!skillModList.Flag(skillCfg, "Condition:ReturningProjectile") {
		returnHits := skillModList.Sum(modparser.Base, skillCfg, "Multiplier:ReturningProjectileHits")
		if returnHits > 0 {
			output.SetN("ReturningProjectileHits", returnHits)
			// calcLib.mod so that "increased/reduced" sources that apply only
			// while Returning are picked up alongside the "more/less" ones,
			// rather than silently ignored
			output.SetN("ReturningProjectileDamageMod", Mod(skillModList, skillCfg, "ReturningProjectileDamage"))
			returnMultiplier := 1 + returnHits*output.N("ReturningProjectileDamageMod")
			dpsMultiplier *= returnMultiplier
			skillData.SetN("dpsMultiplier", dpsMultiplier)
			skillDPSMult := 1.0
			if output.Has("SkillDPSMultiplier") {
				skillDPSMult = output.N("SkillDPSMultiplier")
			}
			output.SetN("SkillDPSMultiplier", skillDPSMult*returnMultiplier)
		}
	}
	if skillModList.Flag(nil, "TriggeredBySnipe") {
		skillFlags["channelRelease"] = true
	}
	// `Flag(...) and Sum(...)` — Flag yields nil when unset, so the key is
	// absent rather than false.
	if skillModList.Flag(nil, "HasTrauma") {
		output.SetN("SustainableTrauma", skillModList.Sum(modparser.Base, skillCfg, "Multiplier:SustainableTraumaStacks"))
	} else {
		output.Del("SustainableTrauma")
	}
	// Mantra of Flames buff count.
	// #EVAL: `cfg` here is not the pass cfg (that local died with the loop
	// above) but an undeclared global, i.e. nil.
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(modparser.Base, nil, "Multiplier:TraumaStacks")
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum(modparser.Base, nil, "Multiplier:VoltaxicWaitingStages")

	if isAttack {
		// Combine hit chance and attack speed
		env.combineStat(c, "AccuracyHitChance", "AVERAGE", "")
		env.combineStat(c, "HitChance", "AVERAGE", "")
		env.combineStat(c, "Speed", "HARMONICMEAN", "")
		env.combineStat(c, "HitSpeed", "OR", "")
		env.combineStat(c, "HitTime", "OR", "")
		if output.N("Speed") == 0 {
			output.SetN("Time", 0.0)
		} else {
			output.SetN("Time", 1/output.N("Speed"))
		}

		if output.N("Time") > 1 {
			modDB.AddMod(newMod("Condition:OneSecondAttackTime", modparser.Flag, modparser.Bool(true)))
		}
		if skillModList.Flag(nil, "UseOffhandAttackSpeed") && !skillFlags["forceMainHand"] {
			output.Set("Speed", c.offHandStats.Get("Speed"))
			output.Set("Time", c.offHandStats.Get("Time"))
		}
		if skillData.Flag("hitTimeOverride") && !skillData.Flag("triggeredOnDeath") {
			output.Set("HitTime", skillData.Get("hitTimeOverride"))
			output.SetN("HitSpeed", 1/output.N("HitTime"))
		} else if skillData.Has("hitTimeMultiplier") && output.Flag("Time") && !skillData.Flag("triggeredOnDeath") {
			output.SetN("HitTime", output.N("Time")*skillData.N("hitTimeMultiplier"))
			if output.Flag("Cooldown") && skillData.Flag("triggered") {
				output.SetN("HitSpeed", 1/math.Max(output.N("HitTime"), output.N("Cooldown")))
			} else if output.Flag("Cooldown") {
				output.SetN("HitSpeed", 1/(output.N("HitTime")+output.N("Cooldown")))
			} else {
				output.SetN("HitSpeed", math.Min(1/output.N("HitTime"), data.Misc.ServerTickRate))
			}
		}
	}

	// Grab quantity multiplier
	quantityMultiplier := math.Max(activeSkill.SkillModList.Sum(modparser.Base, activeSkill.SkillCfg, "QuantityMultiplier"), 1)
	c.quantityMultiplier = quantityMultiplier
	if quantityMultiplier > 1 {
		output.SetN("QuantityMultiplier", quantityMultiplier)
	}

	env.offenceDamage(c)
}
