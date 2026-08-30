// CalcOffence.lua L1457-1658: skill duration (primary/secondary/tertiary,
// aura, reserve, soul-gain prevention, totem, impale) and skill uptime.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// optName renders the reference's `cond and "Name" or nil` argument: the
// name when the condition holds, dropped otherwise.
func optName(cond bool, names []string, name string) []string {
	if cond {
		return append(names, name)
	}
	return names
}

// offenceDuration ports L1457-1658.
func (env *Env) offenceDuration(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	output, enemyDB, activeSkill := c.output, c.enemyDB, c.activeSkill

	// Skill duration
	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(data.Misc.BuffExpirationSlowCap, Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	mineDur := skillData.Flag("mineDurationAppliesToSkill")

	durationMod := Mod(skillModList, skillCfg, optName(mineDur, []string{"Duration", "PrimaryDuration"}, "MineDuration")...)
	durationMod = math.Max(durationMod, 0)
	output.SetN("DurationMod", durationMod)

	durationBase := skillData.N("duration") + skillModList.Sum(modparser.Base, skillCfg, "Duration", "PrimaryDuration")
	permanent := activeSkill.Minion != nil && skillModList.Flag(skillCfg, activeSkill.Minion.Type+"PermanentDuration")
	if durationBase > 0 && !permanent {
		duration := durationBase * durationMod
		if skillData.Flag("debuff") {
			duration *= debuffDurationMult
		}
		output.SetN("Duration", math.Ceil(duration*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
	}
	durationBase = skillData.N("durationSecondary") + skillModList.Sum(modparser.Base, skillCfg, "Duration", "SecondaryDuration")
	if durationBase > 0 {
		dm := math.Max(Mod(skillModList, skillCfg, optName(mineDur, []string{"Duration", "SecondaryDuration"}, "MineDuration")...), 0)
		duration := durationBase * dm
		if skillData.Flag("debuffSecondary") {
			duration *= debuffDurationMult
		}
		output.SetN("DurationSecondary", math.Ceil(duration*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
	}
	durationBase = skillData.N("durationTertiary") + skillModList.Sum(modparser.Base, skillCfg, "Duration", "TertiaryDuration")
	if durationBase > 0 {
		dm := math.Max(Mod(skillModList, skillCfg, optName(mineDur, []string{"Duration", "TertiaryDuration"}, "MineDuration")...), 0)
		duration := durationBase * dm
		if skillData.Flag("debuffTertiary") {
			duration *= debuffDurationMult
		}
		output.SetN("DurationTertiary", math.Ceil(duration*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
	}
	if durationBase = skillData.N("auraDuration"); durationBase > 0 {
		dm := math.Max(Mod(skillModList, skillCfg, "Duration"), 0)
		output.SetN("AuraDuration", math.Ceil(durationBase*dm*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
	}
	if durationBase = skillData.N("reserveDuration"); durationBase > 0 {
		dm := math.Max(Mod(skillModList, skillCfg, "Duration"), 0)
		output.SetN("ReserveDuration", math.Ceil(durationBase*dm*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)
	}
	if durationBase = skillData.N("soulPreventionDuration"); durationBase > 0 {
		names := []string{"SoulGainPreventionDuration"}
		names = optName(skillData.Flag("skillEffectAppliesToSoulGainPrevention"), names, "Duration")
		names = optName(mineDur, names, "MineDuration")
		dm := math.Max(Mod(skillModList, skillCfg, names...), 0)
		output.SetN("SoulGainPreventionDuration", math.Max(math.Ceil(durationBase*dm*data.Misc.ServerTickRate), 1)/data.Misc.ServerTickRate)
	}
	totemDurationMod := math.Max(Mod(skillModList, skillCfg, "TotemDuration"), 0)
	output.SetN("TotemDurationMod", totemDurationMod)
	totemDurationBase := skillModList.Sum(modparser.Base, skillCfg, "TotemDuration")
	output.SetN("TotemDuration", math.Ceil(totemDurationBase*totemDurationMod*data.Misc.ServerTickRate)/data.Misc.ServerTickRate)

	impaleDurationMod := math.Max(Mod(skillModList, skillCfg, "ImpaleDuration"), 0)
	output.SetN("ImpaleDurationMod", impaleDurationMod)
	impaleDurationBase := data.CharacterConstants["impaled_debuff_base_duration_ms"] / 1000
	output.SetN("ImpaleDuration", math.Ceil(impaleDurationBase*impaleDurationMod*data.Misc.ServerTickRate*debuffDurationMult)/data.Misc.ServerTickRate)

	// Skill uptime
	// exclude vaal skills as we currently don't support soul generation or
	// gain prevention.
	if !activeSkill.SkillTypes[modparser.SkillTypeVaal] {
		cooldown := output.N("Cooldown")
		// #EVAL: "reserveDuration" is lowercase in the reference's list
		// while the output key is "ReserveDuration", so that entry never
		// finds a duration and never writes reserveDurationUptime.
		for _, durationType := range []string{"Duration", "DurationSecondary", "DurationTertiary", "AuraDuration", "reserveDuration"} {
			duration := output.N(durationType)
			if duration != 0 && cooldown != 0 {
				var uptime float64
				if skillModList.Flag(skillCfg, "NoCooldownRecoveryInDuration") {
					uptime = duration / (cooldown + duration)
				} else {
					uptime = duration / cooldown
				}
				uptime = math.Min(uptime, 1)
				output.SetN(durationType+"Uptime", uptime*100)
			}
		}
	}

	env.offenceCosts(c)
}
