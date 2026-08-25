// CalcOffence.lua L3547-4064: the pass driver, the per-pass average damage
// and DPS, the main/off-hand combine, and the leech rates.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// offenceDamage drives the second pass loop and the work that follows it.
func (env *Env) offenceDamage(c *offenceCtx) {
	for _, pass := range c.passList {
		env.offenceExerts(c, pass)
		env.offenceRuthless(c, pass)
		env.offenceCrit(c, pass)
		env.offenceDamageTypes(c, pass)
		env.offencePassDPS(c, pass)
	}
	env.offenceCombine(c)
	env.offenceLeechRates(c)
	env.offenceAilments(c)
}

// offencePassDPS ports L3723-3895 for one pass.
func (env *Env) offencePassDPS(c *offenceCtx, pass *damagePass) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, activeSkill := c.skillFlags, c.enemyDB, c.activeSkill
	cfg, output := pass.cfg, pass.output
	globalOutput := c.output

	// Enemy Regeneration Rate
	output["EnemyLifeRegen"] = enemyDB.Sum("INC", cfg, "LifeRegen")
	output["EnemyManaRegen"] = enemyDB.Sum("INC", cfg, "ManaRegen")
	output["EnemyEnergyShieldRegen"] = enemyDB.Sum("INC", cfg, "EnergyShieldRegen")

	// Calculate average damage and final DPS
	critChance := outNum(output, "CritChance")
	output["AverageHit"] = c.totalHitAvg*(1-critChance/100) + c.totalCritAvg*critChance/100
	if skillFlags["monsterExplode"] {
		output["AverageHitToMonsterLifePercentage"] = outNum(output, "AverageHit") / c.monsterLife * 100
		if truthy(skillData["hitChanceIsExplodeChance"]) {
			output["HitChance"] = output["ExplodeChance"]
		}
	}
	output["AverageDamage"] = outNum(output, "AverageHit") * outNum(output, "HitChance") / 100
	burstHits := 1.0
	if truthy(output["AverageBurstHits"]) {
		burstHits = anyNum(output["AverageBurstHits"])
	}
	globalOutput["AverageBurstHits"] = burstHits
	repeatPenalty := 1.0
	if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[modparser.SkillType.CanRapidFire] && !skillModList.Flag(nil, "NoRepeatBonuses") {
		repeatPenalty = Mod(skillModList, skillCfg, "SealRepeatPenalty")
	}
	globalOutput["AverageBurstDamage"] = outNum(output, "AverageDamage") + outNum(output, "AverageDamage")*(burstHits-1)*repeatPenalty
	globalOutput["ShowBurst"] = burstHits > 1
	speed := outNum(globalOutput, "Speed")
	if truthy(globalOutput["HitSpeed"]) {
		speed = outNum(globalOutput, "HitSpeed")
	}
	output["TotalDPS"] = outNum(output, "AverageDamage") * speed * anyNum(skillData["dpsMultiplier"]) * c.quantityMultiplier

	// Calculate PvP values — setup flags
	skillFlags["isPvP"] = false
	skillFlags["notAttackPvP"] = false
	skillFlags["attackPvP"] = false
	skillFlags["weapon1AttackPvP"] = false
	skillFlags["weapon2AttackPvP"] = false
	skillFlags["notAveragePvP"] = false

	if truthy(env.ConfigInput["PvpScaling"]) {
		panic("offence: PvP scaling unported (no corpus build sets PvpScaling)")
	}
}

// offenceCombine ports L3897-3956: fold the two weapon passes together.
func (env *Env) offenceCombine(c *offenceCtx) {
	if !c.isAttack {
		return
	}
	for _, stat := range []string{"PreEffectiveCritChance", "CritChance", "CritMultiplier"} {
		env.combineStat(c, stat, "AVERAGE", "")
	}
	for _, stat := range []string{
		"AverageDamage", "PvpAverageDamage", "TotalDPS", "PvpTotalDPS",
		"LifeLeechDuration", "LifeLeechInstances", "LifeLeechInstant", "LifeLeechInstantRate", "LifeLeechInstantProportion",
		"EnergyShieldLeechDuration", "EnergyShieldLeechInstances", "EnergyShieldLeechInstant", "EnergyShieldLeechInstantRate", "EnergyShieldLeechInstantProportion",
		"ManaLeechDuration", "ManaLeechInstances", "ManaLeechInstant", "ManaLeechInstantRate", "ManaLeechInstantProportion",
		"LifeOnHit", "LifeOnHitRate", "LifeOnKill",
		"EnergyShieldOnHit", "EnergyShieldOnHitRate", "EnergyShieldOnKill",
		"ManaOnHit", "ManaOnHitRate", "ManaOnKill",
		"impaleStoredHitAvg",
	} {
		env.combineStat(c, stat, "DPS", "")
	}
}

// offenceLeechRates ports L4004-4064.
func (env *Env) offenceLeechRates(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, activeSkill := c.skillFlags, c.output, c.activeSkill

	if activeSkill.Minion != nil {
		speed := outNum(output, "Speed")
		if truthy(output["HitSpeed"]) {
			speed = outNum(output, "HitSpeed")
		}
		skillData["summonSpeed"] = outNum(output, "SummonedMinionsPerCast") * speed * anyNum(skillData["dpsMultiplier"])
	}

	// Calculate leech rates
	output["LifeLeechInstanceRate"] = outNum(output, "Life") * data.Misc.LeechRateBase * Mod(skillModList, skillCfg, "LifeLeechRate")
	output["LifeLeechRate"] = outNum(output, "LifeLeechInstances") * outNum(output, "LifeLeechInstanceRate")
	output["LifeLeechPerHit"] = output["LifeLeechInstanceRate"]
	output["EnergyShieldLeechInstanceRate"] = outNum(output, "EnergyShield") * data.Misc.LeechRateBase * Mod(skillModList, skillCfg, "EnergyShieldLeechRate")
	output["EnergyShieldLeechRate"] = outNum(output, "EnergyShieldLeechInstances") * outNum(output, "EnergyShieldLeechInstanceRate")
	output["EnergyShieldLeechPerHit"] = output["EnergyShieldLeechInstanceRate"]
	output["ManaLeechInstanceRate"] = outNum(output, "Mana") * data.Misc.LeechRateBase * Mod(skillModList, skillCfg, "ManaLeechRate")
	output["ManaLeechRate"] = outNum(output, "ManaLeechInstances") * outNum(output, "ManaLeechInstanceRate")
	output["ManaLeechPerHit"] = output["ManaLeechInstanceRate"]
	// On full life, Immortal Ambition treats life leech as energy shield leech
	if skillModList.Flag(nil, "ImmortalAmbition") {
		output["EnergyShieldLeechRate"] = outNum(output, "EnergyShieldLeechRate") + outNum(output, "LifeLeechRate")
		output["EnergyShieldLeechPerHit"] = outNum(output, "EnergyShieldLeechPerHit") + outNum(output, "LifeLeechPerHit")
		// Clears output.LifeLeechRate to disable leechLife flag
		output["LifeLeechRate"] = 0.0
		output["LifeLeechPerHit"] = 0.0
	}
	// Disable non-instant life leech
	if skillModList.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
		output["LifeLeechRate"] = 0.0
		output["LifeLeechPerHit"] = 0.0
		output["LifeLeechInstances"] = 0.0
	}
	output["LifeLeechRate"] = outNum(output, "LifeLeechInstantRate") +
		math.Min(outNum(output, "LifeLeechRate"), outNum(output, "MaxLifeLeechRate"))*outNum(output, "LifeRecoveryRateMod")
	output["LifeLeechPerHit"] = outNum(output, "LifeLeechInstant") +
		math.Min(outNum(output, "LifeLeechPerHit"), outNum(output, "MaxLifeLeechRate"))*outNum(output, "LifeLeechDuration")*outNum(output, "LifeRecoveryRateMod")
	output["EnergyShieldLeechRate"] = outNum(output, "EnergyShieldLeechInstantRate") +
		math.Min(outNum(output, "EnergyShieldLeechRate"), outNum(output, "MaxEnergyShieldLeechRate"))*outNum(output, "EnergyShieldRecoveryRateMod")
	output["EnergyShieldLeechPerHit"] = outNum(output, "EnergyShieldLeechInstant") +
		math.Min(outNum(output, "EnergyShieldLeechPerHit"), outNum(output, "MaxEnergyShieldLeechRate"))*outNum(output, "EnergyShieldLeechDuration")*outNum(output, "EnergyShieldRecoveryRateMod")
	output["ManaLeechRate"] = outNum(output, "ManaLeechInstantRate") +
		math.Min(outNum(output, "ManaLeechRate"), outNum(output, "MaxManaLeechRate"))*outNum(output, "ManaRecoveryRateMod")
	output["ManaLeechPerHit"] = outNum(output, "ManaLeechInstant") +
		math.Min(outNum(output, "ManaLeechPerHit"), outNum(output, "MaxManaLeechRate"))*outNum(output, "ManaLeechDuration")*outNum(output, "ManaRecoveryRateMod")
	skillFlags["leechLife"] = outNum(output, "LifeLeechRate") > 0
	skillFlags["leechES"] = outNum(output, "EnergyShieldLeechRate") > 0
	skillFlags["leechMana"] = outNum(output, "ManaLeechRate") > 0
	if truthy(skillData["showAverage"]) {
		output["LifeLeechGainPerHit"] = outNum(output, "LifeLeechPerHit") + outNum(output, "LifeOnHit")
		output["EnergyShieldLeechGainPerHit"] = outNum(output, "EnergyShieldLeechPerHit") + outNum(output, "EnergyShieldOnHit")
		output["ManaLeechGainPerHit"] = outNum(output, "ManaLeechPerHit") + outNum(output, "ManaOnHit")
	} else {
		output["LifeLeechGainRate"] = outNum(output, "LifeLeechRate") + outNum(output, "LifeOnHitRate")
		output["EnergyShieldLeechGainRate"] = outNum(output, "EnergyShieldLeechRate") + outNum(output, "EnergyShieldOnHitRate")
		output["ManaLeechGainRate"] = outNum(output, "ManaLeechRate") + outNum(output, "ManaOnHitRate")
	}
}
