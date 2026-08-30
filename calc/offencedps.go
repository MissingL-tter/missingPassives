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
	output.SetN("EnemyLifeRegen", enemyDB.Sum(modparser.Inc, cfg, "LifeRegen"))
	output.SetN("EnemyManaRegen", enemyDB.Sum(modparser.Inc, cfg, "ManaRegen"))
	output.SetN("EnemyEnergyShieldRegen", enemyDB.Sum(modparser.Inc, cfg, "EnergyShieldRegen"))

	// Calculate average damage and final DPS
	critChance := output.N("CritChance")
	output.SetN("AverageHit", c.totalHitAvg*(1-critChance/100)+c.totalCritAvg*critChance/100)
	if skillFlags["monsterExplode"] {
		output.SetN("AverageHitToMonsterLifePercentage", output.N("AverageHit")/c.monsterLife*100)
		if skillData.Flag("hitChanceIsExplodeChance") {
			output.Set("HitChance", output.Get("ExplodeChance"))
		}
	}
	output.SetN("AverageDamage", output.N("AverageHit")*output.N("HitChance")/100)
	burstHits := 1.0
	if output.Flag("AverageBurstHits") {
		burstHits = output.N("AverageBurstHits")
	}
	globalOutput.SetN("AverageBurstHits", burstHits)
	repeatPenalty := 1.0
	if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[modparser.SkillTypeCanRapidFire] && !skillModList.Flag(nil, "NoRepeatBonuses") {
		repeatPenalty = Mod(skillModList, skillCfg, "SealRepeatPenalty")
	}
	globalOutput.SetN("AverageBurstDamage", output.N("AverageDamage")+output.N("AverageDamage")*(burstHits-1)*repeatPenalty)
	globalOutput.SetFlag("ShowBurst", burstHits > 1)
	speed := globalOutput.N("Speed")
	if globalOutput.Has("HitSpeed") {
		speed = globalOutput.N("HitSpeed")
	}
	output.SetN("TotalDPS", output.N("AverageDamage")*speed*skillData.N("dpsMultiplier")*c.quantityMultiplier)

	// Calculate PvP values — setup flags
	skillFlags["isPvP"] = false
	skillFlags["notAttackPvP"] = false
	skillFlags["attackPvP"] = false
	skillFlags["weapon1AttackPvP"] = false
	skillFlags["weapon2AttackPvP"] = false
	skillFlags["notAveragePvP"] = false

	if env.ConfigInput.PvpScaling {
		skillFlags["isPvP"] = true
		skillFlags["attackPvP"] = skillFlags["attack"]
		skillFlags["notAttackPvP"] = !skillFlags["attack"]
		skillFlags["weapon1AttackPvP"] = skillFlags["weapon1Attack"]
		skillFlags["weapon2AttackPvP"] = skillFlags["weapon2Attack"]
		skillFlags["notAveragePvP"] = skillFlags["notAverage"]
		var pvpTvalue float64
		if v := env.ConfigInput.MultiplierPvpTvalueOverride; v.Set {
			pvpTvalue = v.V / 1000
		} else {
			switch {
			case skillData.Has("cooldown"):
				pvpTvalue = skillData.N("cooldown")
			case skillFlags["mine"]:
				pvpTvalue = orOne(output.N("MineLayingTime")) / globalOutput.N("ActionSpeedMod")
			case skillFlags["trap"]:
				pvpTvalue = orOne(output.N("TrapThrowingTime")) / globalOutput.N("ActionSpeedMod")
			default:
				speed := globalOutput.N("Speed")
				if globalOutput.Has("HitSpeed") {
					speed = globalOutput.N("HitSpeed")
				}
				pvpTvalue = 1 / (speed / globalOutput.N("ActionSpeedMod")) * skillModList.More(cfg, "PvpTvalueMultiplier")
			}
			if pvpTvalue > 2147483647 {
				pvpTvalue = 1
			}
		}
		pvpMultiplier := skillModList.More(cfg, "PvpDamageMultiplier")

		pvpNonElemental1 := data.Misc.PvpNonElemental1
		pvpNonElemental2 := data.Misc.PvpNonElemental2
		pvpElemental1 := data.Misc.PvpElemental1
		pvpElemental2 := data.Misc.PvpElemental2

		percentageNonElemental := (output.N("PhysicalHitAverage") + output.N("ChaosHitAverage")) /
			(output.N("TotalMin") + output.N("TotalMax")) * 2
		percentageElemental := 1 - percentageNonElemental
		portionNonElemental := math.Pow(output.N("AverageHit")/pvpTvalue/pvpNonElemental2, pvpNonElemental1) *
			pvpTvalue * pvpNonElemental2 * percentageNonElemental
		portionElemental := math.Pow(output.N("AverageHit")/pvpTvalue/pvpElemental2, pvpElemental1) *
			pvpTvalue * pvpElemental2 * percentageElemental
		output.SetN("PvpAverageHit", (portionNonElemental+portionElemental)*pvpMultiplier)
		output.SetN("PvpAverageDamage", output.N("PvpAverageHit")*output.N("HitChance")/100)
		speed := globalOutput.N("Speed")
		if globalOutput.Has("HitSpeed") {
			speed = globalOutput.N("HitSpeed")
		}
		output.SetN("PvpTotalDPS", output.N("PvpAverageDamage")*speed*skillData.N("dpsMultiplier"))

		// fix for these being nan
		for _, k := range []string{"PvpAverageHit", "PvpAverageDamage", "PvpTotalDPS"} {
			if math.IsNaN(output.N(k)) {
				output.SetN(k, 0.0)
			}
		}
	}
}

// orOne is `x or 1` over an output read that stores no key when absent.
func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
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
		speed := output.N("Speed")
		if output.Has("HitSpeed") {
			speed = output.N("HitSpeed")
		}
		skillData.SetN("summonSpeed", output.N("SummonedMinionsPerCast")*speed*skillData.N("dpsMultiplier"))
	}

	// Calculate leech rates
	output.SetN("LifeLeechInstanceRate", output.N("Life")*data.Misc.LeechRateBase*Mod(skillModList, skillCfg, "LifeLeechRate"))
	output.SetN("LifeLeechRate", output.N("LifeLeechInstances")*output.N("LifeLeechInstanceRate"))
	output.Set("LifeLeechPerHit", output.Get("LifeLeechInstanceRate"))
	output.SetN("EnergyShieldLeechInstanceRate", output.N("EnergyShield")*data.Misc.LeechRateBase*Mod(skillModList, skillCfg, "EnergyShieldLeechRate"))
	output.SetN("EnergyShieldLeechRate", output.N("EnergyShieldLeechInstances")*output.N("EnergyShieldLeechInstanceRate"))
	output.Set("EnergyShieldLeechPerHit", output.Get("EnergyShieldLeechInstanceRate"))
	output.SetN("ManaLeechInstanceRate", output.N("Mana")*data.Misc.LeechRateBase*Mod(skillModList, skillCfg, "ManaLeechRate"))
	output.SetN("ManaLeechRate", output.N("ManaLeechInstances")*output.N("ManaLeechInstanceRate"))
	output.Set("ManaLeechPerHit", output.Get("ManaLeechInstanceRate"))
	// On full life, Immortal Ambition treats life leech as energy shield leech
	if skillModList.Flag(nil, "ImmortalAmbition") {
		output.SetN("EnergyShieldLeechRate", output.N("EnergyShieldLeechRate")+output.N("LifeLeechRate"))
		output.SetN("EnergyShieldLeechPerHit", output.N("EnergyShieldLeechPerHit")+output.N("LifeLeechPerHit"))
		// Clears output.LifeLeechRate to disable leechLife flag
		output.SetN("LifeLeechRate", 0.0)
		output.SetN("LifeLeechPerHit", 0.0)
	}
	// Disable non-instant life leech
	if skillModList.Flag(nil, "UnaffectedByNonInstantLifeLeech") {
		output.SetN("LifeLeechRate", 0.0)
		output.SetN("LifeLeechPerHit", 0.0)
		output.SetN("LifeLeechInstances", 0.0)
	}
	output.SetN("LifeLeechRate", output.N("LifeLeechInstantRate")+
		math.Min(output.N("LifeLeechRate"), output.N("MaxLifeLeechRate"))*output.N("LifeRecoveryRateMod"))
	output.SetN("LifeLeechPerHit", output.N("LifeLeechInstant")+
		math.Min(output.N("LifeLeechPerHit"), output.N("MaxLifeLeechRate"))*output.N("LifeLeechDuration")*output.N("LifeRecoveryRateMod"))
	output.SetN("EnergyShieldLeechRate", output.N("EnergyShieldLeechInstantRate")+
		math.Min(output.N("EnergyShieldLeechRate"), output.N("MaxEnergyShieldLeechRate"))*output.N("EnergyShieldRecoveryRateMod"))
	output.SetN("EnergyShieldLeechPerHit", output.N("EnergyShieldLeechInstant")+
		math.Min(output.N("EnergyShieldLeechPerHit"), output.N("MaxEnergyShieldLeechRate"))*output.N("EnergyShieldLeechDuration")*output.N("EnergyShieldRecoveryRateMod"))
	output.SetN("ManaLeechRate", output.N("ManaLeechInstantRate")+
		math.Min(output.N("ManaLeechRate"), output.N("MaxManaLeechRate"))*output.N("ManaRecoveryRateMod"))
	output.SetN("ManaLeechPerHit", output.N("ManaLeechInstant")+
		math.Min(output.N("ManaLeechPerHit"), output.N("MaxManaLeechRate"))*output.N("ManaLeechDuration")*output.N("ManaRecoveryRateMod"))
	skillFlags["leechLife"] = output.N("LifeLeechRate") > 0
	skillFlags["leechES"] = output.N("EnergyShieldLeechRate") > 0
	skillFlags["leechMana"] = output.N("ManaLeechRate") > 0
	if skillData.Flag("showAverage") {
		output.SetN("LifeLeechGainPerHit", output.N("LifeLeechPerHit")+output.N("LifeOnHit"))
		output.SetN("EnergyShieldLeechGainPerHit", output.N("EnergyShieldLeechPerHit")+output.N("EnergyShieldOnHit"))
		output.SetN("ManaLeechGainPerHit", output.N("ManaLeechPerHit")+output.N("ManaOnHit"))
	} else {
		output.SetN("LifeLeechGainRate", output.N("LifeLeechRate")+output.N("LifeOnHitRate"))
		output.SetN("EnergyShieldLeechGainRate", output.N("EnergyShieldLeechRate")+output.N("EnergyShieldOnHitRate"))
		output.SetN("ManaLeechGainRate", output.N("ManaLeechRate")+output.N("ManaOnHitRate"))
	}
}
