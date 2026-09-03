// CalcOffence.lua L4065-5560: the ailment pass — affliction chances, bleed,
// poison, ignite, the non-damaging ailments, knockback, enemy stun and
// impale — plus the secondary-effect combine that follows.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// ailmentCfg builds one of the bleed/poison/ignite dot configurations
// (L4318-4334 and its two copies). The reference gives skillCond a metatable
// falling back to skillCfg/cfg; only "CriticalStrike" is ever mutated after
// construction and this table shadows it, so a snapshot copy is exact.
func ailmentCfg(skillCfg, cfg *modstore.Cfg, keywordFlags modparser.KeywordFlag) *modstore.Cfg {
	cfgFlags := modparser.FlagNone
	if cfg.Flags != nil {
		cfgFlags = *cfg.Flags
	}
	cfgKeywords := modparser.KeywordNone
	if cfg.KeywordFlags != nil {
		cfgKeywords = *cfg.KeywordFlags
	}
	flags := modparser.FlagDot | modparser.FlagAilment | (cfgFlags & modparser.FlagWeaponMask)
	if cfgFlags&modparser.FlagMelee != 0 {
		flags |= modparser.FlagMeleeHit
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
		Flags:           flagp(flags),
		KeywordFlags:    keywordp((cfgKeywords &^ modparser.KeywordHit) | keywordFlags),
		SkillCond:       cond,
		SkillDist:       skillCfg.SkillDist,
	}
}

// checkWeapon1HFlags ports the local of the same name (L4210).
func checkWeapon1HFlags(targetCfg, cfg *modstore.Cfg) {
	cfgFlags := modparser.FlagNone
	if cfg.Flags != nil {
		cfgFlags = *cfg.Flags
	}
	t := modparser.FlagNone
	if targetCfg.Flags != nil {
		t = *targetCfg.Flags
	}
	targetCfg.Flags = flagp(t | (cfgFlags & modparser.FlagWeapon1H))
}

// dotMultiRaw is `Override(DotMultiplier) or Sum(DotMultiplier) +
// Sum(<Type>DotMultiplier)`; dotMulti wraps it as the 1 + x/100 factor the
// three damaging ailments use.
func dotMultiRaw(skillModList *modstore.List, dotCfg *modstore.Cfg, typeName string) float64 {
	if ov, ok := skillModList.Override(dotCfg, "DotMultiplier"); ok {
		return valueNum(ov)
	}
	return skillModList.Sum(modparser.Base, dotCfg, "DotMultiplier") + skillModList.Sum(modparser.Base, dotCfg, typeName+"DotMultiplier")
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
			handCondition = &modparser.CondTag{Var: "OffHandAttack"}
		} else if pass.label == "Main Hand" {
			handCondition = &modparser.CondTag{Var: "MainHandAttack"}
		}
		// Note: This section is the legacy implementation of 'Perfect Agony'
		if skillModList.Sum(modparser.Base, nil, "CritMultiplierAppliesToDegen") > 0 {
			for _, value := range skillModList.Tabulate(modparser.Base, cfg, "CritMultiplier") {
				mod := value.Mod
				if mod.Source != "Base" { // The global base Crit Multi doesn't apply to ailments with Perfect Agony
					args := []modparser.Tag{&modparser.CondTag{Var: "CriticalStrike"}, handCondition}
					args = append(args, mod.Tags...)
					skillModList.AddMod(modparser.NewModFull("DotMultiplier", modparser.Base, modparser.Num(math.Floor(valueNum(mod.Value)/2)), mod.Source, mod.SourceSet, modparser.FlagAilment, modparser.KeywordNone, args...))
				}
			}
		}
		if skillModList.Flag(nil, "DotMultiplierIsCritMultiplier") {
			// On enemy crit multiplier effects also apply for Perfect Agony
			var base float64
			if multiOverride, ok := skillModList.Override(cfg, "CritMultiplier"); ok {
				base = valueNum(multiOverride) - 100
			} else {
				base = skillModList.Sum(modparser.Base, cfg, "CritMultiplier")
			}
			skillModList.AddMod(newModSF("DotMultiplier", modparser.Override, modparser.Num(base+enemyDB.Sum(modparser.Base, cfg, "SelfCritMultiplier")), "Perfect Agony", modparser.FlagAilment, modparser.KeywordNone, &modparser.CondTag{Var: "CriticalStrike"}, handCondition))
		}

		// Calculate chance to inflict secondary dots/status effects
		cfg.SkillCond["CriticalStrike"] = true
		if !skillFlags["attack"] || skillModList.Flag(cfg, "CannotBleed") {
			output.SetN("BleedChanceOnCrit", 0.0)
		} else {
			output.SetN("BleedChanceOnCrit", math.Min(100, skillModList.Sum(modparser.Base, cfg, "BleedChance")+enemyDB.Sum(modparser.Base, nil, "SelfBleedChance")))
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotPoison") {
			output.SetN("PoisonChanceOnCrit", 0.0)
		} else {
			output.SetN("PoisonChanceOnCrit", math.Min(100, skillModList.Sum(modparser.Base, cfg, "PoisonChance")+enemyDB.Sum(modparser.Base, nil, "SelfPoisonChance")))
		}
		if !skillFlags["hit"] {
			output.SetN("ImpaleChanceOnCrit", 0.0)
		} else if env.ModeEffective {
			output.SetN("ImpaleChanceOnCrit", math.Min(100, skillModList.Sum(modparser.Base, cfg, "ImpaleChance")))
		} else {
			output.SetN("ImpaleChanceOnCrit", 0.0)
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotKnockback") {
			output.SetN("KnockbackChanceOnCrit", 0.0)
		} else {
			output.SetN("KnockbackChanceOnCrit", skillModList.Sum(modparser.Base, cfg, "EnemyKnockbackChance"))
		}
		cfg.SkillCond["CriticalStrike"] = false
		if !skillFlags["attack"] || skillModList.Flag(cfg, "CannotBleed") {
			output.SetN("BleedChanceOnHit", 0.0)
		} else {
			output.SetN("BleedChanceOnHit", math.Min(100, skillModList.Sum(modparser.Base, cfg, "BleedChance")+enemyDB.Sum(modparser.Base, nil, "SelfBleedChance")))
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotPoison") {
			output.SetN("PoisonChanceOnHit", 0.0)
			output.SetN("ChaosPoisonChance", 0.0)
		} else {
			output.SetN("PoisonChanceOnHit", math.Min(100, skillModList.Sum(modparser.Base, cfg, "PoisonChance")+enemyDB.Sum(modparser.Base, nil, "SelfPoisonChance")))
			output.SetN("ChaosPoisonChance", math.Min(100, skillModList.Sum(modparser.Base, cfg, "ChaosPoisonChance")))
		}
		// Elemental Ailment Affliction Chance | Elemental Ailment Additionals
		for _, ailment := range data.ElementalAilmentTypeList {
			chance := skillModList.Sum(modparser.Base, cfg, "Enemy"+ailment+"Chance") + enemyDB.Sum(modparser.Base, nil, "Self"+ailment+"Chance")
			// `(Flag and 100 or 0) + Sum(...)` parses as
			// `Flag and 100 or (0 + Sum(...))`, so an immune enemy yields
			// exactly 100 and the avoid sum is dropped.
			var avoidRaw float64
			if enemyDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") {
				avoidRaw = 100
			} else {
				avoidRaw = enemyDB.Sum(modparser.Base, nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments")
			}
			avoid := 1 - math.Min(avoidRaw, 100)/100
			if ailment == "Chill" {
				chance = 100
			}
			chance = chance * avoid
			// Warden's Oath of Summer Scorch Chance
			if ailment == "Ignite" && env.ModDB.Flag(nil, "IgniteCanScorch") {
				output.SetN("ScorchChance", math.Min(100, chance))
				skillModList.AddMod(newModS("EnemyScorchChance", modparser.Base, modparser.Num(chance), "Ignite Chance"))
			}
			if skillFlags["hit"] && !(skillModList.Flag(cfg, "Cannot"+ailment) || avoid == 0) {
				output.SetN(ailment+"ChanceOnHit", math.Min(100, chance))
				ad, hasAd := ailmentData[ailment]
				if skillModList.Flag(cfg, "CritsDontAlways"+ailment) || // e.g. Painseeker
					(hasAd && ad.Alt && !skillModList.Flag(cfg, "CritAlwaysAltAilments")) { // e.g. Secrets of Suffering
					output.Set(ailment+"ChanceOnCrit", output.Get(ailment+"ChanceOnHit"))
				} else {
					output.SetN(ailment+"ChanceOnCrit", 100.0)
				}
			} else {
				output.SetN(ailment+"ChanceOnHit", 0.0)
				output.SetN(ailment+"ChanceOnCrit", 0.0)
			}
			// Warden's Oath of Summer Scorch on Crit Chance
			if ailment == "Scorch" && env.ModDB.Flag(nil, "IgniteCanScorch") && avoid != 0 {
				output.SetN("ScorchChanceOnCrit", 100.0)
			}
			critPart := output.N(ailment + "ChanceOnCrit")
			if skillModList.Flag(cfg, "NeverCrit") {
				critPart = 0
			}
			if output.N(ailment+"ChanceOnHit")+critPart > 0 {
				skillFlags["inflict"+ailment] = true
			}
		}
		if !skillFlags["hit"] || skillModList.Flag(cfg, "CannotKnockback") {
			output.SetN("KnockbackChanceOnHit", 0.0)
		} else {
			output.SetN("KnockbackChanceOnHit", skillModList.Sum(modparser.Base, cfg, "EnemyKnockbackChance"))
		}
		if env.ModeEffective {
			output.SetN("ImpaleChance", math.Min(100, skillModList.Sum(modparser.Base, cfg, "ImpaleChance")))
		} else {
			output.SetN("ImpaleChance", 0.0)
		}
		if skillModList.Sum(modparser.Base, cfg, "FireExposureChance") > 0 {
			skillFlags["applyFireExposure"] = true
		}
		if skillModList.Sum(modparser.Base, cfg, "ColdExposureChance") > 0 {
			skillFlags["applyColdExposure"] = true
		}
		if skillModList.Sum(modparser.Base, cfg, "LightningExposureChance") > 0 {
			skillFlags["applyLightningExposure"] = true
		}
		if env.ModeEffective {
			for _, ailment := range data.AilmentTypeList {
				mult := 1 - enemyDB.Sum(modparser.Base, nil, "Avoid"+ailment)/100
				if enemyDB.Flag(nil, ailment+"Immune") {
					mult = 0
				}
				output.SetN(ailment+"ChanceOnHit", output.N(ailment+"ChanceOnHit")*mult)
				output.SetN(ailment+"ChanceOnCrit", output.N(ailment+"ChanceOnCrit")*mult)
				if ailment == "Poison" {
					output.SetN("ChaosPoisonChance", output.N("ChaosPoisonChance")*mult)
				}
			}
		}

		ailmentMode := AilmentAverage
		if v := env.ConfigInput.AilmentMode; v != "" {
			ailmentMode = v
		}
		if ailmentMode == AilmentCrit || modDB.Flag(nil, "AilmentsOnlyFromCrit") {
			for _, ailment := range data.AilmentTypeList {
				output.SetN(ailment+"ChanceOnHit", 0.0)
			}
		}

		// Perfect Agony + Elemental Overload completely disables ailment
		// application
		if modDB.Flag(nil, "AilmentsAreNeverFromCrit") {
			for _, ailment := range data.AilmentTypeList {
				if modDB.Flag(nil, "AilmentsOnlyFromCrit") {
					output.SetN(ailment+"ChanceOnCrit", 0.0)
				} else {
					output.Set(ailment+"ChanceOnCrit", output.Get(ailment+"ChanceOnHit"))
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
					sourceHitDmg += output.N(typ + "HitAverage")
					sourceCritDmg += output.N(typ + "CritAverage")
				}
			}
			return sourceHitDmg, sourceCritDmg
		}

		// Calculate the inflict chance and base damage of a secondary effect
		calcAilmentDamage := func(typ string, sourceCritChance, sourceHitDmg, sourceCritDmg float64) float64 {
			chanceOnHit, chanceOnCrit := output.N(typ+"ChanceOnHit"), output.N(typ+"ChanceOnCrit")
			// Use sourceCritChance to factor in chance a critical ailment is present
			chanceFromHit := chanceOnHit * (1 - sourceCritChance/100)
			chanceFromCrit := chanceOnCrit * sourceCritChance / 100
			output.SetN(typ+"Chance", chanceFromHit+chanceFromCrit)
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
		if activeSkill.SkillTypes[modparser.SkillTypeChillingArea] || activeSkill.SkillTypes[modparser.SkillTypeNonHitChill] {
			skillFlags["chill"] = true
			incChill := skillModList.Sum(modparser.Inc, cfg, "EnemyChillEffect")
			moreChill := skillModList.More(cfg, "EnemyChillEffect")
			output.SetN("ChillEffectMod", (1+incChill/100)*moreChill*Mod(enemyDB, nil, "SelfChillEffect"))
			incChillDuration := skillModList.Sum(modparser.Inc, cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration") +
				enemyDB.Sum(modparser.Inc, nil, "SelfChillDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
			moreChillDuration := skillModList.More(cfg, "EnemyChillDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration") *
				enemyDB.More(nil, "SelfChillDuration", "SelfAilmentDuration", "SelfElementalAilmentDuration")
			output.SetN("ChillDurationMod", (1+incChillDuration/100)*moreChillDuration)
			chillMax := ailmentData["Chill"].Max
			if ov, ok := skillModList.Override(nil, "ChillMax"); ok {
				chillMax = valueNum(ov)
			}
			output.SetN("ChillSourceEffect", math.Min(chillMax, math.Floor(*ailmentData["Chill"].Default*output.N("ChillEffectMod"))))
		}
		// Crit ailments are done differently for Freeze
		if output.N("FreezeChanceOnHit")+output.N("FreezeChanceOnCrit") > 0 {
			hit, crit := calcAverageSourceDamage("Freeze")
			baseVal := calcAilmentDamage("Freeze", output.N("CritChance"), hit, crit) * skillModList.More(cfg, "FreezeAsThoughDealing")
			if baseVal > 0 {
				skillFlags["freeze"] = true
				output.SetN("FreezeDurationMod", 1+
					skillModList.Sum(modparser.Inc, cfg, "EnemyFreezeDuration", "EnemyAilmentDuration", "EnemyElementalAilmentDuration")/100+
					enemyDB.Sum(modparser.Inc, nil, "SelfFreezeDuration", "SelfElementalAilmentDuration", "SelfAilmentDuration", "HoarfrostFreezeDuration")/100)
			}
		}
		// Cycle through all non-damage ailments here, modifying them if needed.
		// The reference iterates a 5-key table with pairs(); each iteration
		// touches only its own keys, so the order is immaterial.
		for _, ailment := range []string{"Chill", "Shock", "Scorch", "Brittle", "Sap"} {
			if output.N(ailment+"ChanceOnHit")+output.N(ailment+"ChanceOnCrit") <= 0 {
				continue
			}
			// Sets the crit strike condition to match ailment mode.
			cfg.SkillCond["CriticalStrike"] = ailmentMode == AilmentCrit && !modDB.Flag(nil, "AilmentsAreNeverFromCrit")

			hit, crit := calcAverageSourceDamage(ailment)
			damage := calcAilmentDamage(ailment, output.N("CritChance"), hit, crit) * skillModList.More(cfg, ailment+"AsThoughDealing")
			// We check if there is a damage instance above 0 since if you deal
			// 0 damage, you don't apply anything.
			if damage > 0 {
				skillFlags[strings.ToLower(ailment)] = true
				incDur := skillModList.Sum(modparser.Inc, cfg, "Enemy"+ailment+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") +
					enemyDB.Sum(modparser.Inc, nil, "Self"+ailment+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
				moreDur := skillModList.More(cfg, "Enemy"+ailment+"Duration", "EnemyElementalAilmentDuration", "EnemyAilmentDuration") *
					enemyDB.More(nil, "Self"+ailment+"Duration", "SelfElementalAilmentDuration", "SelfAilmentDuration")
				output.SetN(ailment+"Duration", *ailmentData[ailment].Duration*(1+incDur/100)*moreDur*debuffDurationMult)
				// Line Controls Crit Conditional for Crit Mastery
				output.SetN(ailment+"EffectMod", Mod(skillModList, cfg, "Enemy"+ailment+"Effect")*Mod(enemyDB, nil, "Self"+ailment+"Effect"))
			}
		}

		// Calculate knockback chance/distance
		output.SetN("KnockbackChance", math.Min(100, output.N("KnockbackChanceOnHit")*(1-output.N("CritChance")/100)+
			output.N("KnockbackChanceOnCrit")*output.N("CritChance")/100+enemyDB.Sum(modparser.Base, nil, "SelfKnockbackChance")))
		if output.N("KnockbackChance") > 0 {
			output.SetN("KnockbackDistance", util.RoundHalfUp(4*Mod(skillModList, cfg, "EnemyKnockbackDistance"), 0))
		}

		// Calculate enemy stun modifiers
		enemyStunThresholdRed := -skillModList.Sum(modparser.Inc, cfg, "EnemyStunThreshold")
		if enemyStunThresholdRed > 75 {
			output.SetN("EnemyStunThresholdMod", 1-(75+(enemyStunThresholdRed-75)*25/(enemyStunThresholdRed-50))/100)
		} else {
			output.SetN("EnemyStunThresholdMod", 1-enemyStunThresholdRed/100)
		}
		base := 0.35
		if skillData.Flag("baseStunDuration") {
			base = skillData.N("baseStunDuration")
		}
		incDur := skillModList.Sum(modparser.Inc, cfg, "EnemyStunDuration")
		incDurCrit := skillModList.Sum(modparser.Inc, cfg, "EnemyStunDurationOnCrit")
		moreDur := skillModList.More(cfg, "EnemyStunDuration")
		chanceToDouble := math.Min(skillModList.Sum(modparser.Base, cfg, "DoubleEnemyStunDurationChance")+
			enemyDB.Sum(modparser.Base, cfg, "SelfDoubleStunDurationChance"), 100)
		incRecov := enemyDB.Sum(modparser.Inc, nil, "StunRecovery")
		minimumStunDuration := base * moreDur / (1 + incRecov/100)
		output.SetN("EnemyStunDuration", minimumStunDuration)
		if incDurCrit != 0 && output.N("CritChance") != 0 {
			if output.N("CritChance") == 100 {
				minimumStunDuration = minimumStunDuration * (1 + (incDur+incDurCrit)/100)
				output.SetN("EnemyStunDuration", minimumStunDuration)
			} else {
				output.SetN("EnemyStunDuration", output.N("EnemyStunDuration")*(1+(incDur+incDurCrit*output.N("CritChance")/100)/100))
			}
		} else {
			minimumStunDuration = minimumStunDuration * (1 + incDur/100)
			output.SetN("EnemyStunDuration", minimumStunDuration)
		}
		if chanceToDouble != 0 {
			output.SetN("EnemyStunDuration", output.N("EnemyStunDuration")*(1+chanceToDouble/100))
		}

		// Calculate impale chance and modifiers
		if c.canDeal["Physical"] && (output.N("ImpaleChance")+output.N("ImpaleChanceOnCrit")) > 0 {
			skillFlags["impale"] = true
			critChance := output.N("CritChance") / 100
			impaleChance := math.Min(output.N("ImpaleChance")/100, 1)*(1-critChance) +
				math.Min(output.N("ImpaleChanceOnCrit")/100, 1)*critChance
			maxStacks := skillModList.Sum(modparser.Base, cfg, "ImpaleStacksMax") * (1 + skillModList.Sum(modparser.Base, cfg, "ImpaleAdditionalDurationChance")/100)
			configStacks := enemyDB.Sum(modparser.Base, cfg, "Multiplier:ImpaleStacks")
			impaleStacks := math.Min(maxStacks, configStacks)

			baseStoredDamage := data.Misc.ImpaleStoredDamageBase
			storedExpectedDamageIncOnBleed := skillModList.Sum(modparser.Inc, cfg, "ImpaleEffectOnBleed") *
				math.Min(skillModList.Sum(modparser.Base, cfg, "BleedChance")/100, 1)
			storedExpectedDamageInc := (skillModList.Sum(modparser.Inc, cfg, "ImpaleEffect") + storedExpectedDamageIncOnBleed) / 100
			storedExpectedDamageMore := util.RoundHalfUp(skillModList.More(cfg, "ImpaleEffect"), 2)
			storedExpectedDamageModifier := (1 + storedExpectedDamageInc) * storedExpectedDamageMore
			impaleStoredDamage := baseStoredDamage * storedExpectedDamageModifier
			impaleHitDamageMod := impaleStoredDamage * impaleStacks

			enemyArmour := math.Max(Val(enemyDB, "Armour", nil), 0)
			impaleArmourReduction := armourReductionF(enemyArmour, impaleHitDamageMod*output.N("impaleStoredHitAvg"))
			impaleResist := math.Min(math.Max(0, enemyDB.Sum(modparser.Base, nil, "PhysicalDamageReduction")+
				skillModList.Sum(modparser.Base, cfg, "EnemyImpalePhysicalDamageReduction")+impaleArmourReduction), data.Misc.EnemyPhysicalDamageReductionCap)
			if skillModList.Flag(cfg, "IgnoreEnemyImpalePhysicalDamageReduction") {
				impaleResist = 0
			}
			impaleTakenCfg := &modstore.Cfg{Flags: flagp(modparser.FlagHit)}
			impaleTaken := (1 + enemyDB.Sum(modparser.Inc, impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")/100) *
				enemyDB.More(impaleTakenCfg, "DamageTaken", "PhysicalDamageTaken", "ReflectedDamageTaken")
			impaleDMGModifier := impaleHitDamageMod * (1 - impaleResist/100) * impaleChance * impaleTaken

			globalOutput.SetN("ImpaleStacksMax", maxStacks)
			globalOutput.SetN("ImpaleStacks", impaleStacks)
			output.SetN("ImpaleStoredDamage", impaleStoredDamage*100)
			output.SetN("ImpaleModifier", 1+impaleDMGModifier)
		}
	}
	_ = actor

	// Combine secondary effect stats
	if c.isAttack {
		env.combineStat(c, "BleedChance", CombineAverage, "")
		env.combineStat(c, "BleedDPS", CombineChanceAilment, "BleedChance")
		env.combineStat(c, "PoisonChance", CombineAverage, "")
		env.combineStat(c, "PoisonDPS", CombineChance, "PoisonChance")
		env.combineStat(c, "CausticGroundDPS", CombineChance, "PoisonChance")
		env.combineStat(c, "TotalPoisonDPS", CombineDPS, "")
		env.combineStat(c, "PoisonDamage", CombineChance, "PoisonChance")
		if skillData.Flag("showAverage") {
			env.combineStat(c, "TotalPoisonAverageDamage", CombineDPS, "")
		} else {
			env.combineStat(c, "TotalPoisonStacks", CombineDPS, "")
		}
		env.combineStat(c, "IgniteChance", CombineAverage, "")
		env.combineStat(c, "IgniteDPS", CombineChanceAilment, "IgniteChance")
		if skillFlags["igniteCanStack"] {
			env.combineStat(c, "IgniteDamage", CombineChance, "IgniteChance")
			if skillData.Flag("showAverage") {
				env.combineStat(c, "TotalIgniteAverageDamage", CombineDPS, "")
			}
			env.combineStat(c, "IgniteStacksMax", CombineDPS, "")
			env.combineStat(c, "TotalIgniteDPS", CombineDPS, "")
		}
		for _, stat := range []string{
			"ChillEffectMod", "ChillDuration", "ShockChance", "ShockDuration", "ShockEffectMod",
			"FreezeChance", "FreezeDurationMod", "ScorchChance", "ScorchEffectMod", "ScorchDuration",
			"BrittleChance", "BrittleEffectMod", "BrittleDuration", "SapChance", "SapEffectMod", "SapDuration",
			"ImpaleChance", "ImpaleStoredDamage",
		} {
			env.combineStat(c, stat, CombineAverage, "")
		}
		env.combineStat(c, "ImpaleModifier", CombineChance, "ImpaleChance")
	}

	env.offenceDot(c)
}
