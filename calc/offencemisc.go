// CalcOffence.lua L2422-2540: the misc DPS multipliers (skill DPS mods,
// brands, returning projectiles), the main/off-hand speed combine and the
// quantity multiplier.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// offenceMiscDPS ports L2422-2540.
func (env *Env) offenceMiscDPS(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, modDB := c.skillFlags, c.output, c.modDB
	activeSkill, isAttack := c.activeSkill, c.isAttack
	d := env.Data

	// Other Misc DPS multipliers (like custom source)
	dpsMultiplier := 1.0
	if truthy(skillData["dpsMultiplier"]) {
		dpsMultiplier = anyNum(skillData["dpsMultiplier"])
	}
	dpsMultiplier = dpsMultiplier * (1 + skillModList.Sum("INC", skillCfg, "DPS")/100) * skillModList.More(skillCfg, "DPS")
	skillData["dpsMultiplier"] = dpsMultiplier
	if activeSkill.SkillTypes[modparser.SkillType.Brand] && !truthy(skillData["countsAttachedBrandsInDamage"]) {
		dpsMultiplier *= outNum(output, "AttachedBrandCount")
		skillData["dpsMultiplier"] = dpsMultiplier
		skillDPSMult := 1.0
		if truthy(output["SkillDPSMultiplier"]) {
			skillDPSMult = anyNum(output["SkillDPSMultiplier"])
		}
		output["SkillDPSMultiplier"] = skillDPSMult * outNum(output, "AttachedBrandCount")
	}
	if str(env.ConfigInput["repeatMode"]) == "FINAL" || skillModList.Flag(nil, "OnlyFinalRepeat") {
		repeats := 1.0
		if truthy(output["Repeats"]) {
			repeats = anyNum(output["Repeats"])
		}
		dpsMultiplier /= repeats
		skillData["dpsMultiplier"] = dpsMultiplier
	}
	// Returning Projectiles hit the enemy again on the way back, at reduced
	// damage for some sources of Return. Skipped while viewing a skill part
	// that already represents the returning Projectile, as that would count
	// it twice.
	if skillFlags["projectile"] && skillModList.Flag(skillCfg, "ProjectilesReturn") &&
		!activeSkill.SkillTypes[modparser.SkillType.ProjectileCannotReturn] &&
		!skillModList.Flag(skillCfg, "Condition:ReturningProjectile") {
		returnHits := skillModList.Sum("BASE", skillCfg, "Multiplier:ReturningProjectileHits")
		if returnHits > 0 {
			output["ReturningProjectileHits"] = returnHits
			// calcLib.mod so that "increased/reduced" sources that apply only
			// while Returning are picked up alongside the "more/less" ones,
			// rather than silently ignored
			output["ReturningProjectileDamageMod"] = Mod(skillModList, skillCfg, "ReturningProjectileDamage")
			returnMultiplier := 1 + returnHits*outNum(output, "ReturningProjectileDamageMod")
			dpsMultiplier *= returnMultiplier
			skillData["dpsMultiplier"] = dpsMultiplier
			skillDPSMult := 1.0
			if truthy(output["SkillDPSMultiplier"]) {
				skillDPSMult = anyNum(output["SkillDPSMultiplier"])
			}
			output["SkillDPSMultiplier"] = skillDPSMult * returnMultiplier
		}
	}
	if skillModList.Flag(nil, "TriggeredBySnipe") {
		skillFlags["channelRelease"] = true
	}
	// `Flag(...) and Sum(...)` — Flag yields nil when unset, so the key is
	// absent rather than false.
	if skillModList.Flag(nil, "HasTrauma") {
		output["SustainableTrauma"] = skillModList.Sum("BASE", skillCfg, "Multiplier:SustainableTraumaStacks")
	} else {
		delete(output, "SustainableTrauma")
	}
	// Mantra of Flames buff count.
	// #EVAL: `cfg` here is not the pass cfg (that local died with the loop
	// above) but an undeclared global, i.e. nil.
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum("BASE", nil, "Multiplier:TraumaStacks")
	modDB.Multipliers["BuffOnSelf"] += skillModList.Sum("BASE", nil, "Multiplier:VoltaxicWaitingStages")

	if isAttack {
		// Combine hit chance and attack speed
		env.combineStat(c, "AccuracyHitChance", "AVERAGE", "")
		env.combineStat(c, "HitChance", "AVERAGE", "")
		env.combineStat(c, "Speed", "HARMONICMEAN", "")
		env.combineStat(c, "HitSpeed", "OR", "")
		env.combineStat(c, "HitTime", "OR", "")
		if outNum(output, "Speed") == 0 {
			output["Time"] = 0.0
		} else {
			output["Time"] = 1 / outNum(output, "Speed")
		}

		if outNum(output, "Time") > 1 {
			modDB.AddMod(newMod("Condition:OneSecondAttackTime", "FLAG", true))
		}
		if skillModList.Flag(nil, "UseOffhandAttackSpeed") && !skillFlags["forceMainHand"] {
			output["Speed"] = c.offHandStats["Speed"]
			output["Time"] = c.offHandStats["Time"]
		}
		if truthy(skillData["hitTimeOverride"]) && !truthy(skillData["triggeredOnDeath"]) {
			output["HitTime"] = skillData["hitTimeOverride"]
			output["HitSpeed"] = 1 / outNum(output, "HitTime")
		} else if truthy(skillData["hitTimeMultiplier"]) && truthy(output["Time"]) && !truthy(skillData["triggeredOnDeath"]) {
			output["HitTime"] = outNum(output, "Time") * anyNum(skillData["hitTimeMultiplier"])
			if truthy(output["Cooldown"]) && truthy(skillData["triggered"]) {
				output["HitSpeed"] = 1 / math.Max(outNum(output, "HitTime"), outNum(output, "Cooldown"))
			} else if truthy(output["Cooldown"]) {
				output["HitSpeed"] = 1 / (outNum(output, "HitTime") + outNum(output, "Cooldown"))
			} else {
				output["HitSpeed"] = math.Min(1/outNum(output, "HitTime"), d.Misc.ServerTickRate)
			}
		}
	}

	// Grab quantity multiplier
	quantityMultiplier := math.Max(activeSkill.SkillModList.Sum("BASE", activeSkill.SkillCfg, "QuantityMultiplier"), 1)
	c.quantityMultiplier = quantityMultiplier
	if quantityMultiplier > 1 {
		output["QuantityMultiplier"] = quantityMultiplier
	}

	env.offenceDamage(c)
}
