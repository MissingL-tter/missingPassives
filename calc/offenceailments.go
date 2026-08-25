// CalcOffence.lua L4065-5560: the ailment pass — affliction chances, bleed,
// poison, ignite, the non-damaging ailments, knockback, enemy stun and
// impale — plus the secondary-effect combine that follows.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// ailmentCfg builds one of the bleed/poison/ignite dot configurations
// (L4318-4334 and its two copies). The reference gives skillCond a metatable
// falling back to skillCfg/cfg; only "CriticalStrike" is ever mutated after
// construction and this table shadows it, so a snapshot copy is exact.
func ailmentCfg(skillCfg, cfg *modstore.Cfg, keywordFlags int64) *modstore.Cfg {
	cfgFlags := int64(0)
	if cfg.Flags != nil {
		cfgFlags = *cfg.Flags
	}
	cfgKeywords := int64(0)
	if cfg.KeywordFlags != nil {
		cfgKeywords = *cfg.KeywordFlags
	}
	flags := modparser.ModFlag.Dot | modparser.ModFlag.Ailment | (cfgFlags & modparser.ModFlag.WeaponMask)
	if cfgFlags&modparser.ModFlag.Melee != 0 {
		flags |= modparser.ModFlag.MeleeHit
	}
	cond := map[string]bool{}
	for k, v := range cfg.SkillCond {
		cond[k] = v
	}
	for k, v := range skillCfg.SkillCond {
		if v {
			cond[k] = v
		}
	}
	cond["CriticalStrike"] = true
	return &modstore.Cfg{
		SkillName:       skillCfg.SkillName,
		SkillPart:       skillCfg.SkillPart,
		SkillTypes:      skillCfg.SkillTypes,
		SummonSkillName: skillCfg.SummonSkillName,
		SlotName:        skillCfg.SlotName,
		Flags:           i64p(flags),
		KeywordFlags:    i64p((cfgKeywords &^ modparser.KeywordFlag.Hit) | keywordFlags),
		SkillCond:       cond,
		SkillDist:       skillCfg.SkillDist,
	}
}

// checkWeapon1HFlags ports the local of the same name (L4210).
func checkWeapon1HFlags(targetCfg, cfg *modstore.Cfg) {
	cfgFlags := int64(0)
	if cfg.Flags != nil {
		cfgFlags = *cfg.Flags
	}
	t := int64(0)
	if targetCfg.Flags != nil {
		t = *targetCfg.Flags
	}
	targetCfg.Flags = i64p(t | (cfgFlags & modparser.ModFlag.Weapon1H))
}

// dotMultiRaw is `Override(DotMultiplier) or Sum(DotMultiplier) +
// Sum(<Type>DotMultiplier)`; dotMulti wraps it as the 1 + x/100 factor the
// three damaging ailments use.
func dotMultiRaw(skillModList *modstore.List, dotCfg *modstore.Cfg, typeName string) float64 {
	if ov := skillModList.Override(dotCfg, "DotMultiplier"); truthy(ov) {
		return anyNum(ov)
	}
	return skillModList.Sum("BASE", dotCfg, "DotMultiplier") + skillModList.Sum("BASE", dotCfg, typeName+"DotMultiplier")
}

func dotMulti(skillModList *modstore.List, dotCfg *modstore.Cfg, typeName string) float64 {
	return 1 + dotMultiRaw(skillModList, dotCfg, typeName)/100
}

// offenceAilments ports L4065-5560.
func (env *Env) offenceAilments(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, actor := c.activeSkill, c.actor
	globalOutput := c.output

	ailmentData := data.NonDamagingAilment
	for _, ailment := range data.AilmentTypeList {
		skillFlags[strings.ToLower(ailment)] = false
	}
	skillFlags["igniteCanStack"] = skillModList.Flag(skillCfg, "IgniteCanStack")
	skillFlags["igniteToChaos"] = skillModList.Flag(skillCfg, "IgniteToChaos")
	skillFlags["impale"] = false

	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(data.Misc.BuffExpirationSlowCap, Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}

	// Calculate ailments and debuffs (poison, bleed, ignite, impale, exposure, etc)
	for _, pass := range c.passList {
		cfg, output := pass.cfg, pass.output
		if cfg.SkillCond == nil {
			cfg.SkillCond = map[string]bool{}
		}

		// Perfect Agony
		var handCondition modparser.Tag
		if pass.label == "Off Hand" {
			handCondition = modparser.Tag{"type": "Condition", "var": "OffHandAttack"}
		} else if pass.label == "Main Hand" {
			handCondition = modparser.Tag{"type": "Condition", "var": "MainHandAttack"}
		}
		handArg := any(nil)
		if handCondition != nil {
			handArg = handCondition
		}
		// Note: This section is the legacy implementation of 'Perfect Agony'
		if skillModList.Sum("BASE", nil, "CritMultiplierAppliesToDegen") > 0 {
			for _, value := range skillModList.Tabulate("BASE", cfg, "CritMultiplier") {
				mod := value.Mod
				if mod.Source != "Base" { // The global base Crit Multi doesn't apply to ailments with Perfect Agony
					args := []any{mod.Source, modparser.ModFlag.Ailment, int64(0),
						modparser.Tag{"type": "Condition", "var": "CriticalStrike"}, handArg}
					args = append(args, mod.Tags...)
					skillModList.AddMod(newMod("DotMultiplier", "BASE", math.Floor(anyNum(mod.Value)/2), args...))
				}
			}
		}
		if skillModList.Flag(nil, "DotMultiplierIsCritMultiplier") {
			// On enemy crit multiplier effects also apply for Perfect Agony
			var base float64
			if multiOverride := skillModList.Override(cfg, "CritMultiplier"); truthy(multiOverride) {
				base = anyNum(multiOverride) - 100
			} else {
				base = skillModList.Sum("BASE", cfg, "CritMultiplier")
			}
			skillModList.AddMod(newMod("DotMultiplier", "OVERRIDE", base+enemyDB.Sum("BASE", cfg, "SelfCritMultiplier"),
				"Perfect Agony", modparser.ModFlag.Ailment, int64(0),
				modparser.Tag{"type": "Condition", "var": "CriticalStrike"}, handArg))
		}

		// Calculate chance to inflict secondary dots/status effects
		cfg.SkillCond["CriticalStrike"] = true
		if !skillFlags["attack"] || skillModList.Flag(cfg, "CannotBleed") {
			output["BleedChanceOnCrit"] = 0.0
		} else {
			output["BleedChanceOnCrit"] = math.Min(100, skillModList.Sum("BASE", cfg, "BleedChance")+enemyDB.Sum("BASE", nil, "SelfBleedChance"))
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotPoison") {
			output["PoisonChanceOnCrit"] = 0.0
		} else {
			output["PoisonChanceOnCrit"] = math.Min(100, skillModList.Sum("BASE", cfg, "PoisonChance")+enemyDB.Sum("BASE", nil, "SelfPoisonChance"))
		}
		if !skillFlags["hit"] {
			output["ImpaleChanceOnCrit"] = 0.0
		} else if env.ModeEffective {
			output["ImpaleChanceOnCrit"] = math.Min(100, skillModList.Sum("BASE", cfg, "ImpaleChance"))
		} else {
			output["ImpaleChanceOnCrit"] = 0.0
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotKnockback") {
			output["KnockbackChanceOnCrit"] = 0.0
		} else {
			output["KnockbackChanceOnCrit"] = skillModList.Sum("BASE", cfg, "EnemyKnockbackChance")
		}
		cfg.SkillCond["CriticalStrike"] = false
		if !skillFlags["attack"] || skillModList.Flag(cfg, "CannotBleed") {
			output["BleedChanceOnHit"] = 0.0
		} else {
			output["BleedChanceOnHit"] = math.Min(100, skillModList.Sum("BASE", cfg, "BleedChance")+enemyDB.Sum("BASE", nil, "SelfBleedChance"))
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotPoison") {
			output["PoisonChanceOnHit"] = 0.0
			output["ChaosPoisonChance"] = 0.0
		} else {
			output["PoisonChanceOnHit"] = math.Min(100, skillModList.Sum("BASE", cfg, "PoisonChance")+enemyDB.Sum("BASE", nil, "SelfPoisonChance"))
			output["ChaosPoisonChance"] = math.Min(100, skillModList.Sum("BASE", cfg, "ChaosPoisonChance"))
		}
		// Elemental Ailment Affliction Chance | Elemental Ailment Additionals
		for _, ailment := range data.ElementalAilmentTypeList {
			chance := skillModList.Sum("BASE", cfg, "Enemy"+ailment+"Chance") + enemyDB.Sum("BASE", nil, "Self"+ailment+"Chance")
			// #EVAL: `(Flag and 100 or 0) + Sum(...)` parses as
			// `Flag and 100 or (0 + Sum(...))`, so an immune enemy yields
			// exactly 100 and the avoid sum is dropped.
			var avoidRaw float64
			if enemyDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") {
				avoidRaw = 100
			} else {
				avoidRaw = enemyDB.Sum("BASE", nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments")
			}
			avoid := 1 - math.Min(avoidRaw, 100)/100
			if ailment == "Chill" {
				chance = 100
			}
			chance = chance * avoid
			// Warden's Oath of Summer Scorch Chance
			if ailment == "Ignite" && env.ModDB.Flag(nil, "IgniteCanScorch") {
				output["ScorchChance"] = math.Min(100, chance)
				skillModList.AddMod(newMod("EnemyScorchChance", "BASE", chance, "Ignite Chance"))
			}
			if skillFlags["hit"] && !(skillModList.Flag(cfg, "Cannot"+ailment) || avoid == 0) {
				output[ailment+"ChanceOnHit"] = math.Min(100, chance)
				ad, hasAd := ailmentData[ailment]
				if skillModList.Flag(cfg, "CritsDontAlways"+ailment) || // e.g. Painseeker
					(hasAd && ad.Alt && !skillModList.Flag(cfg, "CritAlwaysAltAilments")) { // e.g. Secrets of Suffering
					output[ailment+"ChanceOnCrit"] = output[ailment+"ChanceOnHit"]
				} else {
					output[ailment+"ChanceOnCrit"] = 100.0
				}
			} else {
				output[ailment+"ChanceOnHit"] = 0.0
				output[ailment+"ChanceOnCrit"] = 0.0
			}
			// Warden's Oath of Summer Scorch on Crit Chance
			if ailment == "Scorch" && env.ModDB.Flag(nil, "IgniteCanScorch") && avoid != 0 {
				output["ScorchChanceOnCrit"] = 100.0
			}
			critPart := outNum(output, ailment+"ChanceOnCrit")
			if skillModList.Flag(cfg, "NeverCrit") {
				critPart = 0
			}
			if outNum(output, ailment+"ChanceOnHit")+critPart > 0 {
				skillFlags["inflict"+ailment] = true
			}
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotKnockback") {
			output["KnockbackChanceOnHit"] = 0.0
		} else {
			output["KnockbackChanceOnHit"] = skillModList.Sum("BASE", cfg, "EnemyKnockbackChance")
		}
		if env.ModeEffective {
			output["ImpaleChance"] = math.Min(100, skillModList.Sum("BASE", cfg, "ImpaleChance"))
		} else {
			output["ImpaleChance"] = 0.0
		}
		if skillModList.Sum("BASE", cfg, "FireExposureChance") > 0 {
			skillFlags["applyFireExposure"] = true
		}
		if skillModList.Sum("BASE", cfg, "ColdExposureChance") > 0 {
			skillFlags["applyColdExposure"] = true
		}
		if skillModList.Sum("BASE", cfg, "LightningExposureChance") > 0 {
			skillFlags["applyLightningExposure"] = true
		}
		if env.ModeEffective {
			for _, ailment := range data.AilmentTypeList {
				mult := 1 - enemyDB.Sum("BASE", nil, "Avoid"+ailment)/100
				if enemyDB.Flag(nil, ailment+"Immune") {
					mult = 0
				}
				output[ailment+"ChanceOnHit"] = outNum(output, ailment+"ChanceOnHit") * mult
				output[ailment+"ChanceOnCrit"] = outNum(output, ailment+"ChanceOnCrit") * mult
				if ailment == "Poison" {
					output["ChaosPoisonChance"] = outNum(output, "ChaosPoisonChance") * mult
				}
			}
		}

		ailmentMode := "AVERAGE"
		if v := str(env.ConfigInput["ailmentMode"]); v != "" {
			ailmentMode = v
		}
		if ailmentMode == "CRIT" || modDB.Flag(nil, "AilmentsOnlyFromCrit") {
			for _, ailment := range data.AilmentTypeList {
				output[ailment+"ChanceOnHit"] = 0.0
			}
		}

		// Perfect Agony + Elemental Overload completely disables ailment
		// application
		if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
			for _, ailment := range data.AilmentTypeList {
				if modDB.Flag(nil, "AilmentsOnlyFromCrit") {
					output[ailment+"ChanceOnCrit"] = 0.0
				} else {
					output[ailment+"ChanceOnCrit"] = output[ailment+"ChanceOnHit"]
				}
			}
		}

		// Calculates normal and crit damage to be used in non-damaging
		// ailment calculations
		calcAverageSourceDamage := func(ailment string) (float64, float64) {
			sourceHitDmg, sourceCritDmg := 0.0, 0.0
			for _, typ := range dmgTypeList {
				if !c.canDeal[typ] {
					continue
				}
				var applies bool
				if typ == ailmentData[ailment].AssociatedType {
					applies = !skillModList.Flag(cfg, typ+"Cannot"+ailment)
				} else {
					applies = skillModList.Flag(cfg, typ+"Can"+ailment)
				}
				if applies {
					sourceHitDmg += outNum(output, typ+"HitAverage")
					sourceCritDmg += outNum(output, typ+"CritAverage")
				}
			}
			return sourceHitDmg, sourceCritDmg
		}

		// Calculate the inflict chance and base damage of a secondary effect
		calcAilmentDamage := func(typ string, sourceCritChance, sourceHitDmg, sourceCritDmg float64) float64 {
			chanceOnHit, chanceOnCrit := outNum(output, typ+"ChanceOnHit"), outNum(output, typ+"ChanceOnCrit")
			// Use sourceCritChance to factor in chance a critical ailment is present
			chanceFromHit := chanceOnHit * (1 - sourceCritChance/100)
			chanceFromCrit := chanceOnCrit * sourceCritChance / 100
			output[typ+"Chance"] = chanceFromHit + chanceFromCrit
			baseFromHit := sourceHitDmg * chanceFromHit / (chanceFromHit + chanceFromCrit)
			baseFromCrit := sourceCritDmg * chanceFromCrit / (chanceFromHit + chanceFromCrit)
			return baseFromHit + baseFromCrit
		}

		env.offenceBleed(c, pass, calcAilmentDamage, debuffDurationMult)
		env.offencePoison(c, pass, calcAilmentDamage, debuffDurationMult)
		env.offenceIgnite(c, pass, calcAilmentDamage, debuffDurationMult)

		// Calculate non-damaging ailments effect and duration modifiers.
		// (enemyThreshold and the per-ailment effect/thresh closures feed the
		// breakdown only, so they are not computed here.)
		if activeSkill.SkillTypes[modparser.SkillType.ChillingArea] || activeSkill.SkillTypes[modparser.SkillType.NonHitChill] {
			skillFlags["chill"] = true
			incChill := skillModList.Sum("INC", cfg, "EnemyChillEffect")
			moreChill := skillModList.More(cfg, "EnemyChillEffect")
			output["ChillEffectMod"] = (1 + incChill/100) * moreChill * Mod(enemyDB, nil, "SelfChillEffect")
			incChillDuration := skillModList.Sum("INC", cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration") +
				enemyDB.Sum("INC", nil, "SelfChillDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
			moreChillDuration := skillModList.More(cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration") *
				enemyDB.More(nil, "SelfChillDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
			output["ChillDurationMod"] = (1 + incChillDuration/100) * moreChillDuration
			chillMax := ailmentData["Chill"].Max
			if ov := skillModList.Override(nil, "ChillMax"); truthy(ov) {
				chillMax = anyNum(ov)
			}
			output["ChillSourceEffect"] = math.Min(chillMax, math.Floor(*ailmentData["Chill"].Default*outNum(output, "ChillEffectMod")))
		}
		// Crit ailments are done differently for Freeze
		if outNum(output, "FreezeChanceOnHit")+outNum(output, "FreezeChanceOnCrit") > 0 {
			hit, crit := calcAverageSourceDamage("Freeze")
			baseVal := calcAilmentDamage("Freeze", outNum(output, "CritChance"), hit, crit) * skillModList.More(cfg, "FreezeAsThoughDealing")
			if baseVal > 0 {
				skillFlags["freeze"] = true
				output["FreezeDurationMod"] = 1 +
					skillModList.Sum("INC", cfg, "EnemyFreezeDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration")/100 +
					enemyDB.Sum("INC", nil, "SelfFreezeDuration", "SelfElementalAilmentDuration", "SelfAilmentDuration", "HoarfrostFreezeDuration")/100
			}
		}
		// Cycle through all non-damage ailments here, modifying them if needed.
		// The reference iterates a 5-key table with pairs(); each iteration
		// touches only its own keys, so the order is immaterial.
		for _, ailment := range []string{"Chill", "Shock", "Scorch", "Brittle", "Sap"} {
			if outNum(output, ailment+"ChanceOnHit")+outNum(output, ailment+"ChanceOnCrit") <= 0 {
				continue
			}
			// Sets the crit strike condition to match ailment mode.
			cfg.SkillCond["CriticalStrike"] = ailmentMode == "CRIT" && !modDB.Flag(nil, "AilmentsAreNeverFromCrit")

			hit, crit := calcAverageSourceDamage(ailment)
			damage := calcAilmentDamage(ailment, outNum(output, "CritChance"), hit, crit) * skillModList.More(cfg, ailment+"AsThoughDealing")
			// We check if there is a damage instance above 0 since if you deal
			// 0 damage, you don't apply anything.
			if damage > 0 {
				skillFlags[strings.ToLower(ailment)] = true
				incDur := skillModList.Sum("INC", cfg, "Enemy"+ailment+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") +
					enemyDB.Sum("INC", nil, "Self"+ailment+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
				moreDur := skillModList.More(cfg, "Enemy"+ailment+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") *
					enemyDB.More(nil, "Self"+ailment+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
				output[ailment+"Duration"] = *ailmentData[ailment].Duration * (1 + incDur/100) * moreDur * debuffDurationMult
				// Line Controls Crit Conditional for Crit Mastery
				output[ailment+"EffectMod"] = Mod(skillModList, cfg, "Enemy"+ailment+"Effect") * Mod(enemyDB, nil, "Self"+ailment+"Effect")
			}
		}

		// Calculate knockback chance/distance
		output["KnockbackChance"] = math.Min(100, outNum(output, "KnockbackChanceOnHit")*(1-outNum(output, "CritChance")/100)+
			outNum(output, "KnockbackChanceOnCrit")*outNum(output, "CritChance")/100+enemyDB.Sum("BASE", nil, "SelfKnockbackChance"))
		if outNum(output, "KnockbackChance") > 0 {
			output["KnockbackDistance"] = roundDec(4*Mod(skillModList, cfg, "EnemyKnockbackDistance"), 0)
		}

		// Calculate enemy stun modifiers
		enemyStunThresholdRed := -skillModList.Sum("INC", cfg, "EnemyStunThreshold")
		if enemyStunThresholdRed > 75 {
			output["EnemyStunThresholdMod"] = 1 - (75+(enemyStunThresholdRed-75)*25/(enemyStunThresholdRed-50))/100
		} else {
			output["EnemyStunThresholdMod"] = 1 - enemyStunThresholdRed/100
		}
		base := 0.35
		if truthy(skillData["baseStunDuration"]) {
			base = anyNum(skillData["baseStunDuration"])
		}
		incDur := skillModList.Sum("INC", cfg, "EnemyStunDuration")
		incDurCrit := skillModList.Sum("INC", cfg, "EnemyStunDurationOnCrit")
		moreDur := skillModList.More(cfg, "EnemyStunDuration")
		chanceToDouble := math.Min(skillModList.Sum("BASE", cfg, "DoubleEnemyStunDurationChance")+
			enemyDB.Sum("BASE", cfg, "SelfDoubleStunDurationChance"), 100)
		incRecov := enemyDB.Sum("INC", nil, "StunRecovery")
		minimumStunDuration := base * moreDur / (1 + incRecov/100)
		output["EnemyStunDuration"] = minimumStunDuration
		if incDurCrit != 0 && outNum(output, "CritChance") != 0 {
			if outNum(output, "CritChance") == 100 {
				minimumStunDuration = minimumStunDuration * (1 + (incDur+incDurCrit)/100)
				output["EnemyStunDuration"] = minimumStunDuration
			} else {
				output["EnemyStunDuration"] = outNum(output, "EnemyStunDuration") * (1 + (incDur+incDurCrit*outNum(output, "CritChance")/100)/100)
			}
		} else {
			minimumStunDuration = minimumStunDuration * (1 + incDur/100)
			output["EnemyStunDuration"] = minimumStunDuration
		}
		if chanceToDouble != 0 {
			output["EnemyStunDuration"] = outNum(output, "EnemyStunDuration") * (1 + chanceToDouble/100)
		}

		// Calculate impale chance and modifiers
		if c.canDeal["Physical"] && (outNum(output, "ImpaleChance")+outNum(output, "ImpaleChanceOnCrit")) > 0 {
			skillFlags["impale"] = true
			critChance := outNum(output, "CritChance") / 100
			impaleChance := math.Min(outNum(output, "ImpaleChance")/100, 1)*(1-critChance) +
				math.Min(outNum(output, "ImpaleChanceOnCrit")/100, 1)*critChance
			maxStacks := skillModList.Sum("BASE", cfg, "ImpaleStacksMax") * (1 + skillModList.Sum("BASE", cfg, "ImpaleAdditionalDurationChance")/100)
			configStacks := enemyDB.Sum("BASE", cfg, "Multiplier:ImpaleStacks")
			impaleStacks := math.Min(maxStacks, configStacks)

			baseStoredDamage := data.Misc.ImpaleStoredDamageBase
			storedExpectedDamageIncOnBleed := skillModList.Sum("INC", cfg, "ImpaleEffectOnBleed") *
				math.Min(skillModList.Sum("BASE", cfg, "BleedChance")/100, 1)
			storedExpectedDamageInc := (skillModList.Sum("INC", cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed) / 100
			storedExpectedDamageMore := roundDec(skillModList.More(cfg, "ImpaleEffect"), 2)
			storedExpectedDamageModifier := (1 + storedExpectedDamageInc) * storedExpectedDamageMore
			impaleStoredDamage := baseStoredDamage * storedExpectedDamageModifier
			impaleHitDamageMod := impaleStoredDamage * impaleStacks

			enemyArmour := math.Max(Val(enemyDB, "Armour", nil), 0)
			impaleArmourReduction := armourReductionF(enemyArmour, impaleHitDamageMod*outNum(output, "impaleStoredHitAvg"))
			impaleResist := math.Min(math.Max(0, enemyDB.Sum("BASE", nil, "PhysicalDamageReduction")+
				skillModList.Sum("BASE", cfg, "EnemyImpalePhysicalDamageReduction")+impaleArmourReduction), data.Misc.EnemyPhysicalDamageReductionCap)
			if skillModList.Flag(cfg, "IgnoreEnemyImpalePhysicalDamageReduction") {
				impaleResist = 0
			}
			impaleTakenCfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Hit)}
			impaleTaken := (1 + enemyDB.Sum("INC", impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")/100) *
				enemyDB.More(impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")
			impaleDMGModifier := impaleHitDamageMod * (1 - impaleResist/100) * impaleChance * impaleTaken

			globalOutput["ImpaleStacksMax"] = maxStacks
			globalOutput["ImpaleStacks"] = impaleStacks
			output["ImpaleStoredDamage"] = impaleStoredDamage * 100
			output["ImpaleModifier"] = 1 + impaleDMGModifier
		}
	}
	_ = actor

	// Combine secondary effect stats
	if c.isAttack {
		env.combineStat(c, "BleedChance", "AVERAGE", "")
		env.combineStat(c, "BleedDPS", "CHANCE_AILMENT", "BleedChance")
		env.combineStat(c, "PoisonChance", "AVERAGE", "")
		env.combineStat(c, "PoisonDPS", "CHANCE", "PoisonChance")
		env.combineStat(c, "CausticGroundDPS", "CHANCE", "PoisonChance")
		env.combineStat(c, "TotalPoisonDPS", "DPS", "")
		env.combineStat(c, "PoisonDamage", "CHANCE", "PoisonChance")
		if truthy(skillData["showAverage"]) {
			env.combineStat(c, "TotalPoisonAverageDamage", "DPS", "")
		} else {
			env.combineStat(c, "TotalPoisonStacks", "DPS", "")
		}
		env.combineStat(c, "IgniteChance", "AVERAGE", "")
		env.combineStat(c, "IgniteDPS", "CHANCE_AILMENT", "IgniteChance")
		if skillFlags["igniteCanStack"] {
			env.combineStat(c, "IgniteDamage", "CHANCE", "IgniteChance")
			if truthy(skillData["showAverage"]) {
				env.combineStat(c, "TotalIgniteAverageDamage", "DPS", "")
			}
			env.combineStat(c, "IgniteStacksMax", "DPS", "")
			env.combineStat(c, "TotalIgniteDPS", "DPS", "")
		}
		for _, stat := range []string{
			"ChillEffectMod", "ChillDuration", "ShockChance", "ShockDuration", "ShockEffectMod",
			"FreezeChance", "FreezeDurationMod", "ScorchChance", "ScorchEffectMod", "ScorchDuration",
			"BrittleChance", "BrittleEffectMod", "BrittleDuration", "SapChance", "SapEffectMod", "SapDuration",
			"ImpaleChance", "ImpaleStoredDamage",
		} {
			env.combineStat(c, stat, "AVERAGE", "")
		}
		env.combineStat(c, "ImpaleModifier", "CHANCE", "ImpaleChance")
	}

	env.offenceDot(c)
}
