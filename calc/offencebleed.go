// CalcOffence.lua L4316-4923: the three damaging ailments — bleed, poison
// and ignite.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// ailmentDamageFn is the closure calcs.offence builds per pass.
type ailmentDamageFn func(typ string, sourceCritChance, sourceHitDmg, sourceCritDmg float64) float64

// offenceBleed ports L4316-4585.
func (env *Env) offenceBleed(c *offenceCtx, pass *damagePass, calcAilmentDamage ailmentDamageFn, debuffDurationMult float64) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output

	if !c.canDeal["Physical"] || (output.N("BleedChanceOnHit")+output.N("BleedChanceOnCrit")) <= 0 {
		return
	}
	dotCfg := ailmentCfg(skillCfg, cfg, modparser.KeywordBleed|modparser.KeywordAilment|modparser.KeywordPhysicalDot)
	if pass.label != "Off Hand" {
		activeSkill.BleedCfg = dotCfg
	} else {
		activeSkill.OHBleedCfg = dotCfg
	}
	checkWeapon1HFlags(dotCfg, cfg)

	// For bleeds we will be using a weighted average calculation
	configStacks := enemyDB.Sum(modparser.Base, nil, "Multiplier:BleedStacks")
	maxStacks := skillModList.Sum(modparser.Base, cfg, "BleedStacksMax")
	if ov, ok := skillModList.Override(cfg, "BleedStacksMax"); ok {
		maxStacks = valueNum(ov)
	}
	overrideStackPotential, hasOverrideStackPotential := 0.0, false
	if ov, ok := skillModList.Override(nil, "BleedStackPotentialOverride"); ok {
		overrideStackPotential, hasOverrideStackPotential = valueNum(ov)/maxStacks, true
	}
	globalOutput.SetN("BleedStacksMax", maxStacks)
	durationBase := data.Misc.BleedDurationBase
	if ov, ok := skillModList.Override(dotCfg, "BleedDurationBase"); ok {
		durationBase = valueNum(ov)
	} else if skillData.Flag("bleedDurationIsSkillDuration") && skillData.Flag("duration") {
		durationBase = skillData.N("duration")
	}
	durNames := optName(skillData.Flag("bleedIsSkillEffect"),
		[]string{"EnemyBleedDuration", "EnemyAilmentDuration", "DamagingAilmentDuration"}, "Duration")
	durationMod := Mod(skillModList, dotCfg, durNames...) * Mod(enemyDB, nil, "SelfBleedDuration", "SelfAilmentDuration") /
		Mod(enemyDB, dotCfg, "BleedExpireRate")
	durationMod = math.Max(durationMod, 0)
	rateMod := Mod(skillModList, cfg, "BleedFaster") + enemyDB.Sum(modparser.Inc, nil, "SelfBleedFaster")/100
	globalOutput.SetN("BleedDuration", durationBase*durationMod/rateMod*debuffDurationMult)

	// The chance any given hit applies bleed
	bleedChance := output.N("BleedChanceOnHit")/100*(1-output.N("CritChance")/100) +
		output.N("BleedChanceOnCrit")/100*output.N("CritChance")/100
	// The average number of bleeds that will be active on the enemy at once
	bleedStacks := output.N("HitChance") / 100 * bleedChance * skillData.N("dpsMultiplier")
	speed := globalOutput.N("Speed")
	if globalOutput.Has("HitSpeed") {
		speed = globalOutput.N("HitSpeed")
	}
	if speed > 0 {
		// assume skills with no cast, attack, or cooldown time are single cast
		bleedStacks = bleedStacks * globalOutput.N("BleedDuration") * speed
	}
	activeTotems := skillModList.Sum(modparser.Base, skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
	if ov, ok := env.ModDB.Override(nil, "TotemsSummoned"); ok {
		activeTotems = valueNum(ov)
	}
	if skillFlags["totem"] {
		bleedStacks = bleedStacks * activeTotems
	}
	if configStacks > 0 {
		bleedStacks = configStacks
	}

	if bleedStacks < 1 && overrideStackPotential <= 1 {
		skillModList.AddMod(newModS("Condition:SingleBleed", modparser.Flag, modparser.Bool(true), "bleed"))
	}

	// ratio of bleeds applied : max effective bleeds
	if hasOverrideStackPotential {
		globalOutput.SetN("BleedStackPotential", overrideStackPotential)
	} else {
		globalOutput.SetN("BleedStackPotential", bleedStacks/maxStacks)
	}

	// the amount of damage each bleed does as % maximum
	bleedRollAverage := 50.0
	if globalOutput.N("BleedStackPotential") > 1 {
		// shift damage towards top of range as only top bleeds apply
		bleedRollAverage = (bleedStacks - (maxStacks-1)/2) / (bleedStacks + 1) * 100
	}
	globalOutput.SetN("BleedRollAverage", bleedRollAverage)

	var avgCritBleedDmg, sourceMaxCritDmg, avgHitBleedDmg, sourceMinHitDmg, sourceMaxHitDmg float64
	for subPass := 1; subPass <= 2; subPass++ {
		dotCfg.SkillCond["CriticalStrike"] = subPass != 1

		min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, "Physical", 0, c.convTable(dotCfg))
		output.SetN("BleedPhysicalMin", min)
		output.SetN("BleedPhysicalMax", max)
		if subPass == 2 {
			if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
				dotCfg.SkillCond["CriticalStrike"] = false // force config to non-crit for dotMulti calculation
				output.SetN("CritBleedDotMulti", dotMulti(skillModList, dotCfg, "Physical"))
				dotCfg.SkillCond["CriticalStrike"] = true // reset to true to avoid unintended side effects
			} else {
				output.SetN("CritBleedDotMulti", dotMulti(skillModList, dotCfg, "Physical"))
			}
			sourceMinCritDmg := min * output.N("CritBleedDotMulti")
			sourceMaxCritDmg = max * output.N("CritBleedDotMulti")
			avgCritBleedDmg = sourceMinCritDmg + (sourceMaxCritDmg-sourceMinCritDmg)*bleedRollAverage/100
		} else {
			output.SetN("BleedDotMulti", dotMulti(skillModList, dotCfg, "Physical"))
			sourceMinHitDmg = min * output.N("BleedDotMulti")
			sourceMaxHitDmg = max * output.N("BleedDotMulti")
			avgHitBleedDmg = sourceMinHitDmg + (sourceMaxHitDmg-sourceMinHitDmg)*bleedRollAverage/100
		}
	}

	basePercent := data.Misc.BleedPercentBase
	if skillData.Flag("bleedBasePercent") {
		basePercent = skillData.N("bleedBasePercent")
	}
	// over-stacking bleed stacks increases the chance a critical bleed is present
	ailmentCritChance := 100 * (1 - math.Pow(1-output.N("CritChance")/100, math.Max(globalOutput.N("BleedStackPotential"), 1)))

	// The reference's baseMinVal/baseMaxVal only reach its breakdown, but the
	// calls still matter: each one rewrites output.BleedChance, and the last
	// one wins.
	calcAilmentDamage("Bleed", ailmentCritChance, sourceMinHitDmg, 0)
	calcAilmentDamage("Bleed", 100, sourceMaxHitDmg, sourceMaxCritDmg)
	averageBaseBleedDps := calcAilmentDamage("Bleed", ailmentCritChance, avgHitBleedDmg, avgCritBleedDmg)
	baseBleedDps := averageBaseBleedDps * basePercent / 100 *
		output.N("RuthlessBlowAilmentEffect") * output.N("FistOfWarDamageEffect") * globalOutput.N("AilmentWarcryEffect")
	if baseBleedDps > 0 {
		skillFlags["bleed"] = true
		skillFlags["duration"] = true
		effMult := 1.0
		if env.ModeEffective {
			resist := math.Min(math.Max(0, enemyDB.Sum(modparser.Base, nil, "PhysicalDamageReduction")), data.Misc.EnemyPhysicalDamageReductionCap)
			takenInc := enemyDB.Sum(modparser.Inc, dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
			takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			globalOutput.SetN("BleedEffMult", effMult)
		}
		effectMod := Mod(skillModList, dotCfg, "AilmentEffect")
		activeBleeds := math.Min(bleedStacks, maxStacks)
		output.SetN("BaseBleedDPS", baseBleedDps*effectMod*rateMod*activeBleeds*effMult)
		output.SetN("BleedDPS", math.Min(output.N("BaseBleedDPS"), data.Misc.DotDpsCap))
		globalOutput.SetN("BleedStacks", bleedStacks)
		globalOutput.SetN("BleedDamage", output.N("BaseBleedDPS")*globalOutput.N("BleedDuration"))
	}
}

// offencePoison ports L4587-4900.
func (env *Env) offencePoison(c *offenceCtx, pass *damagePass, calcAilmentDamage ailmentDamageFn, debuffDurationMult float64) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output

	if !c.canDeal["Chaos"] ||
		(output.N("PoisonChanceOnHit")+output.N("PoisonChanceOnCrit")+output.N("ChaosPoisonChance")) <= 0 {
		return
	}
	dotCfg := ailmentCfg(skillCfg, cfg, modparser.KeywordPoison|modparser.KeywordAilment|modparser.KeywordChaosDot)
	if pass.label != "Off Hand" {
		activeSkill.PoisonCfg = dotCfg
	} else {
		activeSkill.OHPoisonCfg = dotCfg
	}
	checkWeapon1HFlags(dotCfg, cfg)

	rateMod := Mod(skillModList, cfg, "PoisonFaster") + enemyDB.Sum(modparser.Inc, nil, "SelfPoisonFaster")/100
	durationBase := data.Misc.PoisonDurationBase
	if ov, ok := skillModList.Override(dotCfg, "PoisonDurationBase"); ok {
		durationBase = valueNum(ov)
	} else if skillData.Flag("poisonDurationIsSkillDuration") && skillData.Flag("duration") {
		durationBase = skillData.N("duration")
	}
	durNames := optName(skillData.Flag("poisonIsSkillEffect"),
		[]string{"EnemyPoisonDuration", "EnemyAilmentDuration", "DamagingAilmentDuration"}, "Duration")
	durationMod := math.Max(Mod(skillModList, dotCfg, durNames...)*Mod(enemyDB, nil, "SelfPoisonDuration", "SelfAilmentDuration"), 0)
	globalOutput.SetN("PoisonDuration", durationBase*durationMod/rateMod*debuffDurationMult)

	// The chance any given hit applies poison
	chaosPoisonChance := 0.0
	if output.N("ChaosHitAverage") > 0 {
		chaosPoisonChance = output.N("ChaosPoisonChance")
	}
	poisonChanceOnHit := math.Min(100, output.N("PoisonChanceOnHit")+chaosPoisonChance)
	poisonChanceOnCrit := math.Min(100, output.N("PoisonChanceOnCrit")+chaosPoisonChance)
	poisonChance := poisonChanceOnHit/100*(1-output.N("CritChance")/100) +
		poisonChanceOnCrit/100*output.N("CritChance")/100

	// Handling of "inflict x additional poisons"
	additionalPoisonStacks := 1.0
	if !skillModList.Flag(nil, "CannotMultiplePoison") {
		additionalPoisonStacks = 1 + math.Min(skillModList.Sum(modparser.Base, cfg, "AdditionalPoisonChance")/100, 1) +
			skillModList.Sum(modparser.Base, cfg, "AdditionalPoisonStacks")
	}

	// Calculate average number of poisons that will be active on the enemy at once
	poisonStackLimit, hasStackLimit := skillModList.Min(cfg, "PoisonStackLimit")
	stackMultiplier := 1.0
	if skillData.Flag("stackMultiplier") {
		stackMultiplier = skillData.N("stackMultiplier")
	}
	poisonStacks := output.N("HitChance") / 100 * poisonChance * additionalPoisonStacks *
		skillData.N("dpsMultiplier") * stackMultiplier * c.quantityMultiplier
	speed := globalOutput.N("Speed")
	if globalOutput.Has("HitSpeed") {
		speed = globalOutput.N("HitSpeed")
	}
	if speed > 0 {
		// assume skills with no cast, attack, or cooldown time are single cast
		poisonStacks = poisonStacks * globalOutput.N("PoisonDuration") * speed

		// If stack limit exists, avg. poison stack is more complicated
		if hasStackLimit && poisonStackLimit > 0 && poisonStacks > poisonStackLimit {
			numPoisoningHits := math.Ceil(poisonStackLimit / additionalPoisonStacks)
			maxPoisonStacks := numPoisoningHits * additionalPoisonStacks
			poisonStacks = math.Min(poisonStacks, maxPoisonStacks)
		}
	}
	if poisonStacks < additionalPoisonStacks && env.ConfigInput.MultiplierPoisonOnEnemy == 0 {
		skillModList.AddMod(newModS("Condition:NonPoisonedOnly", modparser.Flag, modparser.Bool(true), "Calculation"))
	}

	var sourceHitDmg, sourceCritDmg, sourceMaxCritDmg, sourceMinHitDmg, sourceMaxHitDmg float64
	for subPass := 1; subPass <= 2; subPass++ {
		dotCfg.SkillCond["CriticalStrike"] = subPass != 1

		totalMin, totalMax := 0.0, 0.0
		{
			min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, "Chaos", 0, c.convTable(dotCfg))
			output.SetN("PoisonChaosMin", min)
			output.SetN("PoisonChaosMax", max)
			totalMin += min
			totalMax += max
		}
		nonChaosMult := 1.0
		if output.N("ChaosPoisonChance") > 0 && output.N("PoisonChaosMax") > 0 {
			// Additional chance for chaos
			chance := "PoisonChanceOnHit"
			if subPass == 2 {
				chance = "PoisonChanceOnCrit"
			}
			chaosChance := math.Min(100, output.N(chance)+output.N("ChaosPoisonChance"))
			nonChaosMult = output.N(chance) / chaosChance
			output.SetN(chance, chaosChance)
		}
		addType := func(typ string, gate bool) {
			if !gate {
				return
			}
			min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, typ, dmgTypeFlagBits["Chaos"], c.convTable(dotCfg))
			output.SetN("Poison"+typ+"Min", min)
			output.SetN("Poison"+typ+"Max", max)
			totalMin += min * nonChaosMult
			totalMax += max * nonChaosMult
		}
		addType("Lightning", c.canDeal["Lightning"] && skillModList.Flag(cfg, "LightningCanPoison"))
		addType("Cold", c.canDeal["Cold"] && skillModList.Flag(cfg, "ColdCanPoison"))
		addType("Fire", c.canDeal["Fire"] && skillModList.Flag(cfg, "FireCanPoison"))
		addType("Physical", c.canDeal["Physical"])
		if subPass == 2 {
			if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
				dotCfg.SkillCond["CriticalStrike"] = false
				output.SetN("CritPoisonDotMulti", dotMulti(skillModList, dotCfg, "Chaos"))
				dotCfg.SkillCond["CriticalStrike"] = true
			} else {
				output.SetN("CritPoisonDotMulti", dotMulti(skillModList, dotCfg, "Chaos"))
			}
			sourceCritDmg = (totalMin + totalMax) / 2 * output.N("CritPoisonDotMulti")
			sourceMaxCritDmg = totalMax * output.N("CritPoisonDotMulti")
		} else {
			output.SetN("PoisonDotMulti", dotMulti(skillModList, dotCfg, "Chaos"))
			sourceHitDmg = (totalMin + totalMax) / 2 * output.N("PoisonDotMulti")
			sourceMinHitDmg = totalMin * output.N("PoisonDotMulti")
			sourceMaxHitDmg = totalMax * output.N("PoisonDotMulti")
		}
	}
	// Breakdown-only in the reference, but each call rewrites
	// output.PoisonChance and the last one wins.
	calcAilmentDamage("Poison", output.N("CritChance"), sourceMinHitDmg, 0)
	calcAilmentDamage("Poison", 100, sourceMaxHitDmg, sourceMaxCritDmg)
	baseVal := calcAilmentDamage("Poison", output.N("CritChance"), sourceHitDmg, sourceCritDmg) * data.Misc.PoisonPercentBase *
		output.N("RuthlessBlowAilmentEffect") * output.N("FistOfWarDamageEffect") * globalOutput.N("AilmentWarcryEffect")
	if baseVal > 0 {
		skillFlags["poison"] = true
		skillFlags["duration"] = true
		effMult := 1.0
		if env.ModeEffective {
			resist := env.calcResistForType(c, "Chaos", dotCfg)
			takenInc := enemyDB.Sum(modparser.Inc, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			globalOutput.SetN("PoisonEffMult", effMult)
		}
		if skillModList.Flag(nil, "Condition:NonPoisonedOnly") {
			poisonStacks = math.Min(additionalPoisonStacks, poisonStacks)
		}
		globalOutput.SetN("PoisonStacks", poisonStacks)
		effectMod := Mod(skillModList, dotCfg, "AilmentEffect")
		singlePoisonDPSCapped := math.Min(math.Min(baseVal*effectMod*rateMod*effMult, data.Misc.DotDpsCap), data.Misc.DotDpsCap)
		output.SetN("PoisonDPS", singlePoisonDPSCapped)
		output.SetN("PoisonDamage", singlePoisonDPSCapped*globalOutput.N("PoisonDuration"))
		groundMult := math.Max(maxOr(skillModList, nil, 0, "PoisonDpsAsCausticGround"), dbMaxOr(enemyDB, nil, 0, "PoisonDpsAsCausticGround"))
		if groundMult > 0 {
			output.SetN("CausticGroundDPS", math.Min(baseVal*effectMod*rateMod*effMult*groundMult/100, data.Misc.DotDpsCap))
			globalOutput.SetFlag("CausticGroundFromPoison", true)
		}
		if skillData.Flag("showAverage") {
			output.Set("TotalPoisonAverageDamage", output.Get("PoisonDamage"))
			output.Set("TotalPoisonDPS", output.Get("PoisonDPS"))
		} else {
			output.SetN("TotalPoisonDPS", math.Min(singlePoisonDPSCapped*poisonStacks, data.Misc.DotDpsCap))
		}
	}
}

// offenceIgnite ports L4902-5235.
func (env *Env) offenceIgnite(c *offenceCtx, pass *damagePass, calcAilmentDamage ailmentDamageFn, debuffDurationMult float64) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output

	if !c.canDeal["Fire"] || (output.N("IgniteChanceOnHit")+output.N("IgniteChanceOnCrit")) <= 0 {
		return
	}
	dotCfg := ailmentCfg(skillCfg, cfg, modparser.KeywordIgnite|modparser.KeywordAilment|modparser.KeywordFireDot)
	if pass.label != "Off Hand" {
		activeSkill.IgniteCfg = dotCfg
	} else {
		activeSkill.OHIgniteCfg = dotCfg
	}
	checkWeapon1HFlags(dotCfg, cfg)

	globalOutput.SetN("IgniteChancePerHit", output.N("IgniteChanceOnHit")*(1-output.N("CritChance")/100)+
		output.N("IgniteChanceOnCrit")*output.N("CritChance")/100)

	// For ignites we will be using a weighted average calculation
	maxStacks := 1.0
	if skillFlags["igniteCanStack"] {
		if ov, ok := skillModList.Override(cfg, "IgniteStacks"); ok {
			maxStacks = valueNum(ov)
		} else {
			maxStacks = maxStacks + skillModList.Sum(modparser.Base, cfg, "IgniteStacks")
		}
	}
	overrideStackPotential, hasOverrideStackPotential := 0.0, false
	if ov, ok := skillModList.Override(nil, "IgniteStackPotentialOverride"); ok {
		overrideStackPotential, hasOverrideStackPotential = valueNum(ov)/maxStacks, true
	}
	globalOutput.SetN("IgniteStacksMax", maxStacks)

	rateMod := (Mod(skillModList, cfg, "IgniteBurnFaster") + enemyDB.Sum(modparser.Inc, nil, "SelfIgniteBurnFaster")/100) /
		Mod(skillModList, cfg, "IgniteBurnSlower")
	durationBase := data.Misc.IgniteDurationBase
	if ov, ok := skillModList.Override(dotCfg, "IgniteDurationBase"); ok {
		durationBase = valueNum(ov)
	}
	durationMod := math.Max(Mod(skillModList, dotCfg, "EnemyIgniteDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration", "DamagingAilmentDuration")*
		Mod(enemyDB, nil, "SelfIgniteDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration"), 0)
	globalOutput.SetN("IgniteDuration", durationBase*durationMod/rateMod*debuffDurationMult)

	// The chance any given hit applies ignite
	igniteChance := output.N("IgniteChanceOnHit")/100*(1-output.N("CritChance")/100) +
		output.N("IgniteChanceOnCrit")/100*output.N("CritChance")/100
	// The average number of ignites that will be active on the enemy at once
	igniteStacks := output.N("HitChance") / 100 * igniteChance * skillData.N("dpsMultiplier")
	speed := globalOutput.N("Speed")
	if globalOutput.Has("HitSpeed") {
		speed = globalOutput.N("HitSpeed")
	}
	if !skillData.Flag("triggeredOnDeath") {
		if output.Flag("Cooldown") {
			hitTime := output.N("Time")
			if output.Flag("HitTime") {
				hitTime = output.N("HitTime")
			}
			igniteStacks = igniteStacks * globalOutput.N("IgniteDuration") / math.Max(output.N("Cooldown"), hitTime)
		} else if speed > 0 {
			igniteStacks = igniteStacks * globalOutput.N("IgniteDuration") * speed
		}
	}
	// ratio of ignites applied : max effective ignites
	if hasOverrideStackPotential {
		globalOutput.SetN("IgniteStackPotential", overrideStackPotential)
	} else {
		globalOutput.SetN("IgniteStackPotential", igniteStacks/maxStacks)
	}

	// the amount of damage each ignite does as % maximum
	igniteRollAverage := 50.0
	if globalOutput.N("IgniteStackPotential") > 1 {
		igniteRollAverage = (igniteStacks - (maxStacks-1)/2) / (igniteStacks + 1) * 100
	}
	globalOutput.SetN("IgniteRollAverage", igniteRollAverage)

	var sourceHitDmg, sourceCritDmg, sourceMaxCritDmg, sourceMinHitDmg, sourceMaxHitDmg float64
	for subPass := 1; subPass <= 2; subPass++ {
		dotCfg.SkillCond["CriticalStrike"] = subPass != 1
		totalMin, totalMax := 0.0, 0.0
		addType := func(typ string, gate bool) {
			if !gate {
				return
			}
			min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, typ, dmgTypeFlagBits["Fire"], c.convTable(dotCfg))
			output.SetN("Ignite"+typ+"Min", min)
			output.SetN("Ignite"+typ+"Max", max)
			totalMin += min
			totalMax += max
		}
		addType("Physical", c.canDeal["Physical"] && skillModList.Flag(cfg, "PhysicalCanIgnite"))
		addType("Lightning", c.canDeal["Lightning"] && skillModList.Flag(cfg, "LightningCanIgnite"))
		addType("Cold", c.canDeal["Cold"] && skillModList.Flag(cfg, "ColdCanIgnite"))
		addType("Fire", c.canDeal["Fire"] && !skillModList.Flag(cfg, "FireCannotIgnite"))
		addType("Chaos", c.canDeal["Chaos"] && skillModList.Flag(cfg, "ChaosCanIgnite"))
		if subPass == 2 {
			if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
				dotCfg.SkillCond["CriticalStrike"] = false
				output.SetN("CritIgniteDotMulti", dotMulti(skillModList, dotCfg, "Fire"))
				dotCfg.SkillCond["CriticalStrike"] = true
			} else {
				output.SetN("CritIgniteDotMulti", dotMulti(skillModList, dotCfg, "Fire"))
			}
			sourceCritDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * output.N("CritIgniteDotMulti")
			sourceMaxCritDmg = totalMax * output.N("CritIgniteDotMulti")
		} else {
			output.SetN("IgniteDotMulti", dotMulti(skillModList, dotCfg, "Fire"))
			sourceHitDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * output.N("IgniteDotMulti")
			sourceMinHitDmg = totalMin * output.N("IgniteDotMulti")
			sourceMaxHitDmg = totalMax * output.N("IgniteDotMulti")
		}
		output.SetN("IgniteTotalMin", totalMin)
		output.SetN("IgniteTotalMax", totalMax)
	}
	// over-stacking ignite stacks increases the chance a critical ignite is present
	ailmentCritChance := 100 * (1 - math.Pow(1-output.N("CritChance")/100, math.Max(1, igniteStacks)))
	// Breakdown-only in the reference, but each call rewrites
	// output.IgniteChance and the last one wins.
	calcAilmentDamage("Ignite", ailmentCritChance, sourceMinHitDmg, 0)
	calcAilmentDamage("Ignite", 100, sourceMaxHitDmg, sourceMaxCritDmg)
	baseVal := calcAilmentDamage("Ignite", ailmentCritChance, sourceHitDmg, sourceCritDmg) * data.Misc.IgnitePercentBase *
		output.N("RuthlessBlowAilmentEffect") * output.N("FistOfWarDamageEffect") * globalOutput.N("AilmentWarcryEffect")
	if baseVal > 0 {
		skillFlags["ignite"] = true
		effMult := 1.0
		if env.ModeEffective {
			if skillModList.Flag(cfg, "IgniteToChaos") {
				resist := env.calcResistForType(c, "Chaos", dotCfg)
				takenInc := enemyDB.Sum(modparser.Inc, dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			} else {
				resist := env.calcResistForType(c, "Fire", dotCfg)
				takenInc := enemyDB.Sum(modparser.Inc, dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
				takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			}
			globalOutput.SetN("IgniteEffMult", effMult)
		}
		effectMod := Mod(skillModList, dotCfg, "AilmentEffect")
		activeIgnites := math.Min(igniteStacks, maxStacks)
		output.SetN("IgniteDPS", math.Min(baseVal*effectMod*rateMod*activeIgnites*effMult, data.Misc.DotDpsCap))
		groundMult := math.Max(maxOr(skillModList, nil, 0, "IgniteDpsAsBurningGround"), dbMaxOr(enemyDB, nil, 0, "IgniteDpsAsBurningGround"))
		if groundMult > 0 {
			// Always use fire eff multi
			resist := env.calcResistForType(c, "Fire", dotCfg)
			takenInc := enemyDB.Sum(modparser.Inc, dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			fireEffMult := (1 - resist/100) * (1 + takenInc/100) * takenMore
			globalOutput.SetN("BurningGroundDPS", math.Min(baseVal*effectMod*rateMod*fireEffMult*groundMult/100, data.Misc.DotDpsCap))
			globalOutput.SetFlag("BurningGroundFromIgnite", true)
		}
		globalOutput.SetN("IgniteDamage", output.N("IgniteDPS")*globalOutput.N("IgniteDuration"))
		if skillFlags["igniteCanStack"] {
			output.SetN("IgniteDamage", output.N("IgniteDPS")*globalOutput.N("IgniteDuration"))
			output.SetN("IgniteStacksMax", maxStacks)
			output.Set("TotalIgniteDPS", output.Get("IgniteDPS"))
		}
	}
}

// dbMaxOr is `store:Max(cfg, name) or fallback` for a ModDB.
func dbMaxOr(db *modstore.DB, cfg *modstore.Cfg, fallback float64, names ...string) float64 {
	if v, ok := db.Max(cfg, names...); ok {
		return v
	}
	return fallback
}
