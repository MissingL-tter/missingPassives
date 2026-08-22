// CalcOffence.lua L4316-4923: the three damaging ailments — bleed, poison
// and ignite.
package calc

import (
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
	d := env.Data

	if !c.canDeal["Physical"] || (outNum(output, "BleedChanceOnHit")+outNum(output, "BleedChanceOnCrit")) <= 0 {
		return
	}
	dotCfg := ailmentCfg(skillCfg, cfg, modparser.KeywordFlag.Bleed|modparser.KeywordFlag.Ailment|modparser.KeywordFlag.PhysicalDot)
	if pass.label != "Off Hand" {
		activeSkill.BleedCfg = dotCfg
	} else {
		activeSkill.OHBleedCfg = dotCfg
	}
	checkWeapon1HFlags(dotCfg, cfg)

	// For bleeds we will be using a weighted average calculation
	configStacks := enemyDB.Sum("BASE", nil, "Multiplier:BleedStacks")
	maxStacks := skillModList.Sum("BASE", cfg, "BleedStacksMax")
	if ov := skillModList.Override(cfg, "BleedStacksMax"); truthy(ov) {
		maxStacks = anyNum(ov)
	}
	overrideStackPotential, hasOverrideStackPotential := 0.0, false
	if ov := skillModList.Override(nil, "BleedStackPotentialOverride"); truthy(ov) {
		overrideStackPotential, hasOverrideStackPotential = anyNum(ov)/maxStacks, true
	}
	globalOutput["BleedStacksMax"] = maxStacks
	durationBase := d.Misc.BleedDurationBase
	if ov := skillModList.Override(dotCfg, "BleedDurationBase"); truthy(ov) {
		durationBase = anyNum(ov)
	} else if truthy(skillData["bleedDurationIsSkillDuration"]) && truthy(skillData["duration"]) {
		durationBase = anyNum(skillData["duration"])
	}
	durNames := optName(truthy(skillData["bleedIsSkillEffect"]),
		[]string{"EnemyBleedDuration", "EnemyAilmentDuration", "DamagingAilmentDuration"}, "Duration")
	durationMod := Mod(skillModList, dotCfg, durNames...) * Mod(enemyDB, nil, "SelfBleedDuration", "SelfAilmentDuration") /
		Mod(enemyDB, dotCfg, "BleedExpireRate")
	durationMod = math.Max(durationMod, 0)
	rateMod := Mod(skillModList, cfg, "BleedFaster") + enemyDB.Sum("INC", nil, "SelfBleedFaster")/100
	globalOutput["BleedDuration"] = durationBase * durationMod / rateMod * debuffDurationMult

	// The chance any given hit applies bleed
	bleedChance := outNum(output, "BleedChanceOnHit")/100*(1-outNum(output, "CritChance")/100) +
		outNum(output, "BleedChanceOnCrit")/100*outNum(output, "CritChance")/100
	// The average number of bleeds that will be active on the enemy at once
	bleedStacks := outNum(output, "HitChance") / 100 * bleedChance * anyNum(skillData["dpsMultiplier"])
	speed := outNum(globalOutput, "Speed")
	if truthy(globalOutput["HitSpeed"]) {
		speed = outNum(globalOutput, "HitSpeed")
	}
	if speed > 0 {
		// assume skills with no cast, attack, or cooldown time are single cast
		bleedStacks = bleedStacks * outNum(globalOutput, "BleedDuration") * speed
	}
	activeTotems := skillModList.Sum("BASE", skillCfg, "ActiveTotemLimit", "ActiveBallistaLimit")
	if ov := env.ModDB.Override(nil, "TotemsSummoned"); truthy(ov) {
		activeTotems = anyNum(ov)
	}
	if skillFlags["totem"] {
		bleedStacks = bleedStacks * activeTotems
	}
	if configStacks > 0 {
		bleedStacks = configStacks
	}

	if bleedStacks < 1 && overrideStackPotential <= 1 {
		skillModList.AddMod(newMod("Condition:SingleBleed", "FLAG", true, "bleed"))
	}

	// ratio of bleeds applied : max effective bleeds
	if hasOverrideStackPotential {
		globalOutput["BleedStackPotential"] = overrideStackPotential
	} else {
		globalOutput["BleedStackPotential"] = bleedStacks / maxStacks
	}

	// the amount of damage each bleed does as % maximum
	bleedRollAverage := 50.0
	if outNum(globalOutput, "BleedStackPotential") > 1 {
		// shift damage towards top of range as only top bleeds apply
		bleedRollAverage = (bleedStacks - (maxStacks-1)/2) / (bleedStacks + 1) * 100
	}
	globalOutput["BleedRollAverage"] = bleedRollAverage

	var avgCritBleedDmg, sourceMaxCritDmg, avgHitBleedDmg, sourceMinHitDmg, sourceMaxHitDmg float64
	for subPass := 1; subPass <= 2; subPass++ {
		dotCfg.SkillCond["CriticalStrike"] = subPass != 1

		min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, "Physical", 0, c.convTable(dotCfg))
		output["BleedPhysicalMin"] = min
		output["BleedPhysicalMax"] = max
		if subPass == 2 {
			if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
				dotCfg.SkillCond["CriticalStrike"] = false // force config to non-crit for dotMulti calculation
				output["CritBleedDotMulti"] = dotMulti(skillModList, dotCfg, "Physical")
				dotCfg.SkillCond["CriticalStrike"] = true // reset to true to avoid unintended side effects
			} else {
				output["CritBleedDotMulti"] = dotMulti(skillModList, dotCfg, "Physical")
			}
			sourceMinCritDmg := min * outNum(output, "CritBleedDotMulti")
			sourceMaxCritDmg = max * outNum(output, "CritBleedDotMulti")
			avgCritBleedDmg = sourceMinCritDmg + (sourceMaxCritDmg-sourceMinCritDmg)*bleedRollAverage/100
		} else {
			output["BleedDotMulti"] = dotMulti(skillModList, dotCfg, "Physical")
			sourceMinHitDmg = min * outNum(output, "BleedDotMulti")
			sourceMaxHitDmg = max * outNum(output, "BleedDotMulti")
			avgHitBleedDmg = sourceMinHitDmg + (sourceMaxHitDmg-sourceMinHitDmg)*bleedRollAverage/100
		}
	}

	basePercent := d.Misc.BleedPercentBase
	if truthy(skillData["bleedBasePercent"]) {
		basePercent = anyNum(skillData["bleedBasePercent"])
	}
	// over-stacking bleed stacks increases the chance a critical bleed is present
	ailmentCritChance := 100 * (1 - math.Pow(1-outNum(output, "CritChance")/100, math.Max(outNum(globalOutput, "BleedStackPotential"), 1)))

	// The reference's baseMinVal/baseMaxVal only reach its breakdown, but the
	// calls still matter: each one rewrites output.BleedChance, and the last
	// one wins.
	calcAilmentDamage("Bleed", ailmentCritChance, sourceMinHitDmg, 0)
	calcAilmentDamage("Bleed", 100, sourceMaxHitDmg, sourceMaxCritDmg)
	averageBaseBleedDps := calcAilmentDamage("Bleed", ailmentCritChance, avgHitBleedDmg, avgCritBleedDmg)
	baseBleedDps := averageBaseBleedDps * basePercent / 100 *
		outNum(output, "RuthlessBlowAilmentEffect") * outNum(output, "FistOfWarDamageEffect") * outNum(globalOutput, "AilmentWarcryEffect")
	if baseBleedDps > 0 {
		skillFlags["bleed"] = true
		skillFlags["duration"] = true
		effMult := 1.0
		if env.ModeEffective {
			resist := math.Min(math.Max(0, enemyDB.Sum("BASE", nil, "PhysicalDamageReduction")), d.Misc.EnemyPhysicalDamageReductionCap)
			takenInc := enemyDB.Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
			takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "PhysicalDamageTaken", "PhysicalDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			globalOutput["BleedEffMult"] = effMult
		}
		effectMod := Mod(skillModList, dotCfg, "AilmentEffect")
		activeBleeds := math.Min(bleedStacks, maxStacks)
		output["BaseBleedDPS"] = baseBleedDps * effectMod * rateMod * activeBleeds * effMult
		output["BleedDPS"] = math.Min(outNum(output, "BaseBleedDPS"), d.Misc.DotDpsCap)
		globalOutput["BleedStacks"] = bleedStacks
		globalOutput["BleedDamage"] = outNum(output, "BaseBleedDPS") * outNum(globalOutput, "BleedDuration")
	}
}

// offencePoison ports L4587-4900.
func (env *Env) offencePoison(c *offenceCtx, pass *damagePass, calcAilmentDamage ailmentDamageFn, debuffDurationMult float64) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output
	d := env.Data

	if !c.canDeal["Chaos"] ||
		(outNum(output, "PoisonChanceOnHit")+outNum(output, "PoisonChanceOnCrit")+outNum(output, "ChaosPoisonChance")) <= 0 {
		return
	}
	dotCfg := ailmentCfg(skillCfg, cfg, modparser.KeywordFlag.Poison|modparser.KeywordFlag.Ailment|modparser.KeywordFlag.ChaosDot)
	if pass.label != "Off Hand" {
		activeSkill.PoisonCfg = dotCfg
	} else {
		activeSkill.OHPoisonCfg = dotCfg
	}
	checkWeapon1HFlags(dotCfg, cfg)

	rateMod := Mod(skillModList, cfg, "PoisonFaster") + enemyDB.Sum("INC", nil, "SelfPoisonFaster")/100
	durationBase := d.Misc.PoisonDurationBase
	if ov := skillModList.Override(dotCfg, "PoisonDurationBase"); truthy(ov) {
		durationBase = anyNum(ov)
	} else if truthy(skillData["poisonDurationIsSkillDuration"]) && truthy(skillData["duration"]) {
		durationBase = anyNum(skillData["duration"])
	}
	durNames := optName(truthy(skillData["poisonIsSkillEffect"]),
		[]string{"EnemyPoisonDuration", "EnemyAilmentDuration", "DamagingAilmentDuration"}, "Duration")
	durationMod := math.Max(Mod(skillModList, dotCfg, durNames...)*Mod(enemyDB, nil, "SelfPoisonDuration", "SelfAilmentDuration"), 0)
	globalOutput["PoisonDuration"] = durationBase * durationMod / rateMod * debuffDurationMult

	// The chance any given hit applies poison
	chaosPoisonChance := 0.0
	if outNum(output, "ChaosHitAverage") > 0 {
		chaosPoisonChance = outNum(output, "ChaosPoisonChance")
	}
	poisonChanceOnHit := math.Min(100, outNum(output, "PoisonChanceOnHit")+chaosPoisonChance)
	poisonChanceOnCrit := math.Min(100, outNum(output, "PoisonChanceOnCrit")+chaosPoisonChance)
	poisonChance := poisonChanceOnHit/100*(1-outNum(output, "CritChance")/100) +
		poisonChanceOnCrit/100*outNum(output, "CritChance")/100

	// Handling of "inflict x additional poisons"
	additionalPoisonStacks := 1.0
	if !skillModList.Flag(nil, "CannotMultiplePoison") {
		additionalPoisonStacks = 1 + math.Min(skillModList.Sum("BASE", cfg, "AdditionalPoisonChance")/100, 1) +
			skillModList.Sum("BASE", cfg, "AdditionalPoisonStacks")
	}

	// Calculate average number of poisons that will be active on the enemy at once
	poisonStackLimit, hasStackLimit := skillModList.Min(cfg, "PoisonStackLimit")
	stackMultiplier := 1.0
	if truthy(skillData["stackMultiplier"]) {
		stackMultiplier = anyNum(skillData["stackMultiplier"])
	}
	poisonStacks := outNum(output, "HitChance") / 100 * poisonChance * additionalPoisonStacks *
		anyNum(skillData["dpsMultiplier"]) * stackMultiplier * c.quantityMultiplier
	speed := outNum(globalOutput, "Speed")
	if truthy(globalOutput["HitSpeed"]) {
		speed = outNum(globalOutput, "HitSpeed")
	}
	if speed > 0 {
		// assume skills with no cast, attack, or cooldown time are single cast
		poisonStacks = poisonStacks * outNum(globalOutput, "PoisonDuration") * speed

		// If stack limit exists, avg. poison stack is more complicated
		if hasStackLimit && poisonStackLimit > 0 && poisonStacks > poisonStackLimit {
			numPoisoningHits := math.Ceil(poisonStackLimit / additionalPoisonStacks)
			maxPoisonStacks := numPoisoningHits * additionalPoisonStacks
			poisonStacks = math.Min(poisonStacks, maxPoisonStacks)
		}
	}
	if poisonStacks < additionalPoisonStacks && anyNum(env.ConfigInput["multiplierPoisonOnEnemy"]) == 0 {
		skillModList.AddMod(newMod("Condition:NonPoisonedOnly", "FLAG", true, "Calculation"))
	}

	var sourceHitDmg, sourceCritDmg, sourceMaxCritDmg, sourceMinHitDmg, sourceMaxHitDmg float64
	for subPass := 1; subPass <= 2; subPass++ {
		dotCfg.SkillCond["CriticalStrike"] = subPass != 1

		totalMin, totalMax := 0.0, 0.0
		{
			min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, "Chaos", 0, c.convTable(dotCfg))
			output["PoisonChaosMin"] = min
			output["PoisonChaosMax"] = max
			totalMin += min
			totalMax += max
		}
		nonChaosMult := 1.0
		if outNum(output, "ChaosPoisonChance") > 0 && outNum(output, "PoisonChaosMax") > 0 {
			// Additional chance for chaos
			chance := "PoisonChanceOnHit"
			if subPass == 2 {
				chance = "PoisonChanceOnCrit"
			}
			chaosChance := math.Min(100, outNum(output, chance)+outNum(output, "ChaosPoisonChance"))
			nonChaosMult = outNum(output, chance) / chaosChance
			output[chance] = chaosChance
		}
		addType := func(typ string, gate bool) {
			if !gate {
				return
			}
			min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, typ, dmgTypeFlagBits["Chaos"], c.convTable(dotCfg))
			output["Poison"+typ+"Min"] = min
			output["Poison"+typ+"Max"] = max
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
				output["CritPoisonDotMulti"] = dotMulti(skillModList, dotCfg, "Chaos")
				dotCfg.SkillCond["CriticalStrike"] = true
			} else {
				output["CritPoisonDotMulti"] = dotMulti(skillModList, dotCfg, "Chaos")
			}
			sourceCritDmg = (totalMin + totalMax) / 2 * outNum(output, "CritPoisonDotMulti")
			sourceMaxCritDmg = totalMax * outNum(output, "CritPoisonDotMulti")
		} else {
			output["PoisonDotMulti"] = dotMulti(skillModList, dotCfg, "Chaos")
			sourceHitDmg = (totalMin + totalMax) / 2 * outNum(output, "PoisonDotMulti")
			sourceMinHitDmg = totalMin * outNum(output, "PoisonDotMulti")
			sourceMaxHitDmg = totalMax * outNum(output, "PoisonDotMulti")
		}
	}
	// Breakdown-only in the reference, but each call rewrites
	// output.PoisonChance and the last one wins.
	calcAilmentDamage("Poison", outNum(output, "CritChance"), sourceMinHitDmg, 0)
	calcAilmentDamage("Poison", 100, sourceMaxHitDmg, sourceMaxCritDmg)
	baseVal := calcAilmentDamage("Poison", outNum(output, "CritChance"), sourceHitDmg, sourceCritDmg) * d.Misc.PoisonPercentBase *
		outNum(output, "RuthlessBlowAilmentEffect") * outNum(output, "FistOfWarDamageEffect") * outNum(globalOutput, "AilmentWarcryEffect")
	if baseVal > 0 {
		skillFlags["poison"] = true
		skillFlags["duration"] = true
		effMult := 1.0
		if env.ModeEffective {
			resist := env.calcResistForType(c, "Chaos", dotCfg)
			takenInc := enemyDB.Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			globalOutput["PoisonEffMult"] = effMult
		}
		if skillModList.Flag(nil, "Condition:NonPoisonedOnly") {
			poisonStacks = math.Min(additionalPoisonStacks, poisonStacks)
		}
		globalOutput["PoisonStacks"] = poisonStacks
		effectMod := Mod(skillModList, dotCfg, "AilmentEffect")
		singlePoisonDPSCapped := math.Min(math.Min(baseVal*effectMod*rateMod*effMult, d.Misc.DotDpsCap), d.Misc.DotDpsCap)
		output["PoisonDPS"] = singlePoisonDPSCapped
		output["PoisonDamage"] = singlePoisonDPSCapped * outNum(globalOutput, "PoisonDuration")
		groundMult := math.Max(maxOr(skillModList, nil, 0, "PoisonDpsAsCausticGround"), dbMaxOr(enemyDB, nil, 0, "PoisonDpsAsCausticGround"))
		if groundMult > 0 {
			output["CausticGroundDPS"] = math.Min(baseVal*effectMod*rateMod*effMult*groundMult/100, d.Misc.DotDpsCap)
			globalOutput["CausticGroundFromPoison"] = true
		}
		if truthy(skillData["showAverage"]) {
			output["TotalPoisonAverageDamage"] = output["PoisonDamage"]
			output["TotalPoisonDPS"] = output["PoisonDPS"]
		} else {
			output["TotalPoisonDPS"] = math.Min(singlePoisonDPSCapped*poisonStacks, d.Misc.DotDpsCap)
		}
	}
}

// offenceIgnite ports L4902-5235.
func (env *Env) offenceIgnite(c *offenceCtx, pass *damagePass, calcAilmentDamage ailmentDamageFn, debuffDurationMult float64) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, cfg, output := c.activeSkill, pass.cfg, pass.output
	globalOutput := c.output
	d := env.Data

	if !c.canDeal["Fire"] || (outNum(output, "IgniteChanceOnHit")+outNum(output, "IgniteChanceOnCrit")) <= 0 {
		return
	}
	dotCfg := ailmentCfg(skillCfg, cfg, modparser.KeywordFlag.Ignite|modparser.KeywordFlag.Ailment|modparser.KeywordFlag.FireDot)
	if pass.label != "Off Hand" {
		activeSkill.IgniteCfg = dotCfg
	} else {
		activeSkill.OHIgniteCfg = dotCfg
	}
	checkWeapon1HFlags(dotCfg, cfg)

	globalOutput["IgniteChancePerHit"] = outNum(output, "IgniteChanceOnHit")*(1-outNum(output, "CritChance")/100) +
		outNum(output, "IgniteChanceOnCrit")*outNum(output, "CritChance")/100

	// For ignites we will be using a weighted average calculation
	maxStacks := 1.0
	if skillFlags["igniteCanStack"] {
		if ov := skillModList.Override(cfg, "IgniteStacks"); truthy(ov) {
			maxStacks = anyNum(ov)
		} else {
			maxStacks = maxStacks + skillModList.Sum("BASE", cfg, "IgniteStacks")
		}
	}
	overrideStackPotential, hasOverrideStackPotential := 0.0, false
	if ov := skillModList.Override(nil, "IgniteStackPotentialOverride"); truthy(ov) {
		overrideStackPotential, hasOverrideStackPotential = anyNum(ov)/maxStacks, true
	}
	globalOutput["IgniteStacksMax"] = maxStacks

	rateMod := (Mod(skillModList, cfg, "IgniteBurnFaster") + enemyDB.Sum("INC", nil, "SelfIgniteBurnFaster")/100) /
		Mod(skillModList, cfg, "IgniteBurnSlower")
	durationBase := d.Misc.IgniteDurationBase
	if ov := skillModList.Override(dotCfg, "IgniteDurationBase"); truthy(ov) {
		durationBase = anyNum(ov)
	}
	durationMod := math.Max(Mod(skillModList, dotCfg, "EnemyIgniteDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration", "DamagingAilmentDuration")*
		Mod(enemyDB, nil, "SelfIgniteDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration"), 0)
	globalOutput["IgniteDuration"] = durationBase * durationMod / rateMod * debuffDurationMult

	// The chance any given hit applies ignite
	igniteChance := outNum(output, "IgniteChanceOnHit")/100*(1-outNum(output, "CritChance")/100) +
		outNum(output, "IgniteChanceOnCrit")/100*outNum(output, "CritChance")/100
	// The average number of ignites that will be active on the enemy at once
	igniteStacks := outNum(output, "HitChance") / 100 * igniteChance * anyNum(skillData["dpsMultiplier"])
	speed := outNum(globalOutput, "Speed")
	if truthy(globalOutput["HitSpeed"]) {
		speed = outNum(globalOutput, "HitSpeed")
	}
	if !truthy(skillData["triggeredOnDeath"]) {
		if truthy(output["Cooldown"]) {
			hitTime := outNum(output, "Time")
			if truthy(output["HitTime"]) {
				hitTime = outNum(output, "HitTime")
			}
			igniteStacks = igniteStacks * outNum(globalOutput, "IgniteDuration") / math.Max(outNum(output, "Cooldown"), hitTime)
		} else if speed > 0 {
			igniteStacks = igniteStacks * outNum(globalOutput, "IgniteDuration") * speed
		}
	}
	// ratio of ignites applied : max effective ignites
	if hasOverrideStackPotential {
		globalOutput["IgniteStackPotential"] = overrideStackPotential
	} else {
		globalOutput["IgniteStackPotential"] = igniteStacks / maxStacks
	}

	// the amount of damage each ignite does as % maximum
	igniteRollAverage := 50.0
	if outNum(globalOutput, "IgniteStackPotential") > 1 {
		igniteRollAverage = (igniteStacks - (maxStacks-1)/2) / (igniteStacks + 1) * 100
	}
	globalOutput["IgniteRollAverage"] = igniteRollAverage

	var sourceHitDmg, sourceCritDmg, sourceMaxCritDmg, sourceMinHitDmg, sourceMaxHitDmg float64
	for subPass := 1; subPass <= 2; subPass++ {
		dotCfg.SkillCond["CriticalStrike"] = subPass != 1
		totalMin, totalMax := 0.0, 0.0
		addType := func(typ string, gate bool) {
			if !gate {
				return
			}
			min, max := env.calcAilmentSourceDamage(activeSkill, output, dotCfg, typ, dmgTypeFlagBits["Fire"], c.convTable(dotCfg))
			output["Ignite"+typ+"Min"] = min
			output["Ignite"+typ+"Max"] = max
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
				output["CritIgniteDotMulti"] = dotMulti(skillModList, dotCfg, "Fire")
				dotCfg.SkillCond["CriticalStrike"] = true
			} else {
				output["CritIgniteDotMulti"] = dotMulti(skillModList, dotCfg, "Fire")
			}
			sourceCritDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * outNum(output, "CritIgniteDotMulti")
			sourceMaxCritDmg = totalMax * outNum(output, "CritIgniteDotMulti")
		} else {
			output["IgniteDotMulti"] = dotMulti(skillModList, dotCfg, "Fire")
			sourceHitDmg = (totalMin + (totalMax-totalMin)*igniteRollAverage/100) * outNum(output, "IgniteDotMulti")
			sourceMinHitDmg = totalMin * outNum(output, "IgniteDotMulti")
			sourceMaxHitDmg = totalMax * outNum(output, "IgniteDotMulti")
		}
		output["IgniteTotalMin"] = totalMin
		output["IgniteTotalMax"] = totalMax
	}
	// over-stacking ignite stacks increases the chance a critical ignite is present
	ailmentCritChance := 100 * (1 - math.Pow(1-outNum(output, "CritChance")/100, math.Max(1, igniteStacks)))
	// Breakdown-only in the reference, but each call rewrites
	// output.IgniteChance and the last one wins.
	calcAilmentDamage("Ignite", ailmentCritChance, sourceMinHitDmg, 0)
	calcAilmentDamage("Ignite", 100, sourceMaxHitDmg, sourceMaxCritDmg)
	baseVal := calcAilmentDamage("Ignite", ailmentCritChance, sourceHitDmg, sourceCritDmg) * d.Misc.IgnitePercentBase *
		outNum(output, "RuthlessBlowAilmentEffect") * outNum(output, "FistOfWarDamageEffect") * outNum(globalOutput, "AilmentWarcryEffect")
	if baseVal > 0 {
		skillFlags["ignite"] = true
		effMult := 1.0
		if env.ModeEffective {
			if skillModList.Flag(cfg, "IgniteToChaos") {
				resist := env.calcResistForType(c, "Chaos", dotCfg)
				takenInc := enemyDB.Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			} else {
				resist := env.calcResistForType(c, "Fire", dotCfg)
				takenInc := enemyDB.Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
				takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			}
			globalOutput["IgniteEffMult"] = effMult
		}
		effectMod := Mod(skillModList, dotCfg, "AilmentEffect")
		activeIgnites := math.Min(igniteStacks, maxStacks)
		output["IgniteDPS"] = math.Min(baseVal*effectMod*rateMod*activeIgnites*effMult, d.Misc.DotDpsCap)
		groundMult := math.Max(maxOr(skillModList, nil, 0, "IgniteDpsAsBurningGround"), dbMaxOr(enemyDB, nil, 0, "IgniteDpsAsBurningGround"))
		if groundMult > 0 {
			// Always use fire eff multi
			resist := env.calcResistForType(c, "Fire", dotCfg)
			takenInc := enemyDB.Sum("INC", dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			takenMore := enemyDB.More(dotCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			fireEffMult := (1 - resist/100) * (1 + takenInc/100) * takenMore
			globalOutput["BurningGroundDPS"] = math.Min(baseVal*effectMod*rateMod*fireEffMult*groundMult/100, d.Misc.DotDpsCap)
			globalOutput["BurningGroundFromIgnite"] = true
		}
		globalOutput["IgniteDamage"] = outNum(output, "IgniteDPS") * outNum(globalOutput, "IgniteDuration")
		if skillFlags["igniteCanStack"] {
			output["IgniteDamage"] = outNum(output, "IgniteDPS") * outNum(globalOutput, "IgniteDuration")
			output["IgniteStacksMax"] = maxStacks
			output["TotalIgniteDPS"] = output["IgniteDPS"]
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
