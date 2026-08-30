// CalcOffence.lua L5605-6168: decay, dropped burning ground, the generic
// damage-over-time section, cost per second, self-hit damage and the
// combined DPS estimate.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// copyCfg is copyTable(cfg, true): a shallow copy of the query config.
func copyCfg(cfg *modstore.Cfg) *modstore.Cfg {
	out := *cfg
	return &out
}

// dotMultiDB is the DotMultiplier `or` chain against a ModDB.
func dotMultiDB(db *modstore.DB, cfg *modstore.Cfg, typeName string) float64 {
	if ov, ok := db.Override(cfg, "DotMultiplier"); ok {
		return valueNum(ov)
	}
	return db.Sum(modparser.Base, cfg, "DotMultiplier") + db.Sum(modparser.Base, cfg, typeName+"DotMultiplier")
}

// offenceDot ports L5605-6168.
func (env *Env) offenceDot(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, output := c.activeSkill, c.output

	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(data.Misc.BuffExpirationSlowCap, Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}

	if skillFlags["hit"] && skillData.Flag("decay") && c.canDeal["Chaos"] {
		// Calculate DPS for Essence of Delirium's Decay effect
		skillFlags["decay"] = true
		skillCfgKeywords := modparser.KeywordNone
		if skillCfg.KeywordFlags != nil {
			skillCfgKeywords = *skillCfg.KeywordFlags
		}
		dotCfg := &modstore.Cfg{
			SkillName:       skillCfg.SkillName,
			SkillPart:       skillCfg.SkillPart,
			SkillTypes:      skillCfg.SkillTypes,
			SummonSkillName: skillCfg.SummonSkillName,
			SlotName:        skillCfg.SlotName,
			Flags:           flagp(modparser.FlagDot),
			KeywordFlags:    keywordp((skillCfgKeywords &^ modparser.KeywordHit) | modparser.KeywordChaosDot),
		}
		activeSkill.DecayCfg = dotCfg
		effMult := 1.0
		if env.ModeEffective {
			resist := env.calcResistForType(c, "Chaos", dotCfg)
			takenInc := enemyDB.Sum(modparser.Inc, nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			takenMore := enemyDB.More(nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			output.SetN("DecayEffMult", effMult)
		}
		inc := skillModList.Sum(modparser.Inc, dotCfg, "Damage", "ChaosDamage")
		more := skillModList.More(dotCfg, "Damage", "ChaosDamage")
		mult := dotMultiRaw(skillModList, dotCfg, "Chaos")
		output.SetN("DecayDPS", skillData.N("decay")*(1+inc/100)*more*(1+mult/100)*effMult)
		output.SetN("DecayDuration", 8*debuffDurationMult)
	}

	baseDropsBurningGround := modDB.Sum(modparser.Base, nil, "DropsBurningGround")
	if baseDropsBurningGround > 0 && c.canDeal["Fire"] {
		dotTakenCfg := &modstore.Cfg{Flags: flagp(modparser.FlagDot), KeywordFlags: keywordp(0)}
		dotTypeCfg := &modstore.Cfg{Flags: flagp(modparser.FlagDot), KeywordFlags: keywordp(modparser.KeywordFireDot)}
		effMult := 1.0
		if env.ModeEffective {
			resist := env.calcResistForType(c, "Fire", dotTypeCfg)
			takenInc := enemyDB.Sum(modparser.Inc, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
		}
		inc := modDB.Sum(modparser.Inc, dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
		more := modDB.More(dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
		mult := dotMultiDB(modDB, dotTypeCfg, "Fire")
		total := baseDropsBurningGround * (1 + inc/100) * more * (1 + mult/100) * effMult
		if !output.Has("BurningGroundDPS") || output.N("BurningGroundDPS") < total {
			output.SetN("BurningGroundDPS", total)
			output.SetFlag("BurningGroundFromIgnite", false)
		}
	}

	// Calculate skill DOT components
	skillCfgFlags := modparser.FlagNone
	if skillCfg.Flags != nil {
		skillCfgFlags = *skillCfg.Flags
	}
	skillCfgKeywords := modparser.KeywordNone
	if skillCfg.KeywordFlags != nil {
		skillCfgKeywords = *skillCfg.KeywordFlags
	}
	dotFlags := modparser.FlagDot | skillCfgFlags
	clearFlag := func(flag modparser.ModFlag, keep bool) {
		if dotFlags|flag == dotFlags && !keep {
			dotFlags &^= flag
		}
	}
	clearFlag(modparser.FlagArea, skillData.Flag("dotIsArea"))
	clearFlag(modparser.FlagProjectile, skillData.Flag("dotIsProjectile"))
	clearFlag(modparser.FlagSpell, skillData.Flag("dotIsSpell"))
	clearFlag(modparser.FlagAttack, skillData.Flag("dotIsAttack"))
	clearFlag(modparser.FlagHit, skillData.Flag("dotIsHit"))
	dotCfg := &modstore.Cfg{
		SkillName:       skillCfg.SkillName,
		SkillPart:       skillCfg.SkillPart,
		SkillTypes:      skillCfg.SkillTypes,
		SummonSkillName: skillCfg.SummonSkillName,
		SlotName:        skillCfg.SlotName,
		Flags:           flagp(dotFlags),
		KeywordFlags:    keywordp(skillCfgKeywords &^ modparser.KeywordHit),
	}

	// spell_damage_modifiers_apply_to_skill_dot does not apply to enemy damage taken
	dotTakenCfg := copyCfg(dotCfg)
	if skillData.Flag("dotIsSpell") {
		dotTakenCfg.Flags = flagp(dotFlags &^ modparser.FlagSpell)
	}

	activeSkill.DotCfg = dotCfg
	activeSkill.DotTypeCfg = map[string]*modstore.Cfg{}
	output.SetN("TotalDotInstance", 0.0)

	env.runSkillFunc(c, data.CallbackPreDot)

	// Section handles generic damage over time
	for _, damageType := range dmgTypeList {
		dotTypeCfg := copyCfg(dotCfg)
		dotTypeCfg.KeywordFlags = keywordp(*dotCfg.KeywordFlags | keywordDotFlag(damageType))
		activeSkill.DotTypeCfg[damageType] = dotTypeCfg
		baseVal := 0.0
		if c.canDeal[damageType] {
			baseVal = skillData.N(damageType + "Dot")
		}
		if baseVal > 0 || output.N(damageType+"Dot") > 0 {
			skillFlags["dot"] = true
			effMult := 1.0
			// Section handles Enemy Damage Taken based on Configs
			if env.ModeEffective {
				resist := 0.0
				takenInc := enemyDB.Sum(modparser.Inc, dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
				takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
				if isElementalRes[damageType] {
					takenInc += enemyDB.Sum(modparser.Inc, dotTakenCfg, "ElementalDamageTaken")
					takenMore *= enemyDB.More(dotTakenCfg, "ElementalDamageTaken")
				}
				if damageType == "Physical" {
					resist = math.Max(0, math.Min(enemyDB.Sum(modparser.Base, nil, "PhysicalDamageReduction"), data.Misc.EnemyPhysicalDamageReductionCap))
				} else {
					resist = env.calcResistForType(c, damageType, dotTypeCfg)
				}
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
				output.SetN(damageType+"DotEffMult", effMult)
			}
			names := optName(isElementalRes[damageType], []string{"Damage", damageType + "Damage"}, "ElementalDamage")
			inc := skillModList.Sum(modparser.Inc, dotTypeCfg, names...)
			if skillModList.Flag(nil, "dotIsHeraldOfAsh") {
				inc = math.Max(inc-skillModList.Sum(modparser.Inc, skillCfg, names...), 0)
			}
			more := skillModList.More(dotTypeCfg, names...)
			mult := dotMultiRaw(skillModList, dotTypeCfg, damageType)
			aura := 1.0
			if activeSkill.SkillTypes[modparser.SkillTypeAura] && !activeSkill.SkillTypes[modparser.SkillTypeRemoteMined] &&
				!activeSkill.SkillTypes[modparser.SkillTypeBanner] {
				aura = Mod(skillModList, dotTypeCfg, "AuraEffect")
			}
			total := baseVal * (1 + inc/100) * more * (1 + mult/100) * aura * effMult
			if !output.Flag(damageType+"Dot") || output.N(damageType+"Dot") == 0 {
				output.SetN(damageType+"Dot", total)
				output.SetN("TotalDotInstance", math.Min(output.N("TotalDotInstance")+total, data.Misc.DotDpsCap))
			} else {
				output.SetN("TotalDotInstance", math.Min(output.N("TotalDotInstance")+total+output.N(damageType+"Dot"), data.Misc.DotDpsCap))
			}
		}
	}

	switch {
	case skillModList.Flag(nil, "DotCanStack"):
		skillFlags["DotCanStack"] = true
		speed := output.N("Speed")
		// Check if skill is being triggered via Mine or Trap
		if *dotCfg.KeywordFlags&modparser.KeywordMine != 0 {
			speed = output.N("MineLayingSpeed")
		} else if *dotCfg.KeywordFlags&modparser.KeywordTrap != 0 {
			speed = output.N("TrapThrowingSpeed")
		}
		output.SetN("TotalDot", math.Min(output.N("TotalDotInstance")*speed*output.N("Duration")*
			skillData.N("dpsMultiplier")*c.quantityMultiplier, data.Misc.DotDpsCap))
		output.Set("TotalDotCalcSection", output.Get("TotalDot"))
	case skillModList.Flag(nil, "dotIsBurningGround"):
		output.SetN("TotalDot", 0.0)
		output.Set("TotalDotCalcSection", output.Get("TotalDotInstance"))
		if !output.Has("BurningGroundDPS") || output.N("BurningGroundDPS") < output.N("TotalDotInstance") {
			output.SetN("BurningGroundDPS", math.Max(output.N("BurningGroundDPS"), output.N("TotalDotInstance")))
			output.SetFlag("BurningGroundFromIgnite", false)
		}
	case skillModList.Flag(nil, "dotIsCausticGround"):
		output.SetN("TotalDot", 0.0)
		output.Set("TotalDotCalcSection", output.Get("TotalDotInstance"))
		if !output.Has("CausticGroundDPS") || output.N("CausticGroundDPS") < output.N("TotalDotInstance") {
			output.SetN("CausticGroundDPS", math.Max(output.N("CausticGroundDPS"), output.N("TotalDotInstance")))
			output.SetFlag("CausticGroundFromPoison", false)
		}
	case skillModList.Flag(nil, "dotIsCorruptingBlood"):
		output.SetN("TotalDot", 0.0)
		output.Set("TotalDotCalcSection", output.Get("TotalDotInstance"))
		if !output.Has("CorruptingBloodDPS") || output.N("CorruptingBloodDPS") < output.N("TotalDotInstance") {
			output.SetN("CorruptingBloodDPS", math.Max(output.N("CorruptingBloodDPS"), output.N("TotalDotInstance")))
		}
	default:
		if skillModList.Flag(nil, "DotCanStackAsTotems") && skillFlags["totem"] {
			skillFlags["DotCanStack"] = true
		}
		attachedBrandCount := 1.0
		if activeSkill.SkillTypes[modparser.SkillTypeBrand] && !skillData.Flag("countsAttachedBrandsInDamage") &&
			output.Flag("AttachedBrandCount") {
			attachedBrandCount = output.N("AttachedBrandCount")
		}
		if attachedBrandCount > 1 {
			output.SetN("TotalDot", math.Min(output.N("TotalDotInstance")*attachedBrandCount, data.Misc.DotDpsCap))
		} else {
			output.Set("TotalDot", output.Get("TotalDotInstance"))
		}
		output.Set("TotalDotCalcSection", output.Get("TotalDot"))
	}

	env.offenceCostPerSecond(c)
	env.offenceSelfHit(c)
	env.offenceCombinedDPS(c)
}

// keywordDotFlag maps a damage type to its KeywordFlag.<Type>Dot bit.
func keywordDotFlag(damageType string) modparser.KeywordFlag {
	switch damageType {
	case "Physical":
		return modparser.KeywordPhysicalDot
	case "Lightning":
		return modparser.KeywordLightningDot
	case "Cold":
		return modparser.KeywordColdDot
	case "Fire":
		return modparser.KeywordFireDot
	case "Chaos":
		return modparser.KeywordChaosDot
	}
	panic("offence: no dot keyword flag for " + damageType)
}

// offenceCostPerSecond ports L5806-5862.
func (env *Env) offenceCostPerSecond(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, output := c.skillFlags, c.output

	for _, resource := range costOrder {
		val := c.costs[resource]
		eb := env.ModDB.Flag(nil, "EnergyShieldProtectsMana")
		if !(val.upfront && output.Flag(resource+"HasCost") && output.N(resource+"Cost") > 0 &&
			!(output.Flag(resource+"PerSecondHasCost") && !(eb && skillModList.Sum(modparser.Base, skillCfg, "ManaCostAsEnergyShieldCost") > 0)) &&
			(output.N("Speed") > 0 || output.Flag("Cooldown"))) {
			continue
		}
		usedResource := resource
		if eb && resource == "Mana" {
			usedResource = "ES"
		}

		repeats := 1.0
		if output.Has("Repeats") {
			repeats = output.N("Repeats")
		}
		useSpeed := 1.0
		switch {
		case skillFlags["trap"] || skillFlags["mine"]:
			preSpeed := output.N("MineLayingSpeed")
			if output.Has("TrapThrowingSpeed") {
				preSpeed = output.N("TrapThrowingSpeed")
			}
			cooldown, hasCooldown := 0.0, false
			if output.Has("TrapCooldown") {
				cooldown, hasCooldown = output.N("TrapCooldown"), true
			} else if output.Flag("Cooldown") {
				cooldown, hasCooldown = output.N("Cooldown"), true
			}
			if hasCooldown && cooldown > 0 {
				useSpeed = 1 / cooldown
			} else {
				useSpeed = preSpeed
			}
		case skillFlags["totem"]:
			if output.Flag("Cooldown") && output.N("Cooldown") > 0 {
				if output.N("TotemPlacementSpeed") > 0 {
					useSpeed = output.N("TotemPlacementSpeed")
				} else {
					useSpeed = 1 / output.N("Cooldown")
				}
			} else {
				useSpeed = output.N("TotemPlacementSpeed")
			}
			useSpeed /= repeats
		case skillModList.Flag(nil, "HasSeals") && skillModList.Flag(nil, "UseMaxUnleash") &&
			env.PlayerMainSkill.SkillData.Flag("hitTimeOverride"):
			useSpeed = env.PlayerMainSkill.SkillData.N("hitTimeOverride") / repeats
		default:
			if output.Flag("Cooldown") && output.N("Cooldown") > 0 {
				if output.N("Speed") > 0 {
					useSpeed = output.N("Speed")
				} else {
					useSpeed = 1 / output.N("Cooldown")
				}
			} else {
				useSpeed = output.N("Speed")
			}
			useSpeed /= repeats
		}
		_ = skillData

		output.SetFlag(usedResource+"PerSecondHasCost", true)
		output.SetN(usedResource+"PerSecondCost", output.N(usedResource+"PerSecondCost")+output.N(resource+"Cost")*useSpeed)
	}
}

// offenceCombinedDPS ports L5984-6168.
func (env *Env) offenceCombinedDPS(c *offenceCtx) {
	skillData, skillFlags, output := c.skillData, c.skillFlags, c.output

	baseKey := "TotalDPS"
	if skillData.Flag("showAverage") {
		baseKey = "AverageDamage"
	}
	baseDPS := output.N(baseKey)
	output.SetN("CombinedDPS", baseDPS)
	combinedAvg := baseDPS
	if skillFlags["dot"] {
		output.SetN("WithDotDPS", baseDPS+output.N("TotalDot"))
	}
	if c.quantityMultiplier > 1 && output.Flag("TotalPoisonDPS") {
		output.SetN("TotalPoisonDPS", math.Min(output.N("TotalPoisonDPS")*c.quantityMultiplier, data.Misc.DotDpsCap))
	}
	if skillData.Flag("showAverage") {
		combinedAvg += output.N("TotalPoisonAverageDamage")
		output.SetN("WithPoisonDPS", baseDPS+output.N("TotalPoisonAverageDamage"))
	} else {
		output.SetN("WithPoisonDPS", baseDPS+output.N("TotalPoisonDPS"))
	}
	if skillFlags["ignite"] {
		if skillFlags["igniteCanStack"] {
			if skillData.Flag("showAverage") {
				combinedAvg = output.N("CombinedDPS") + output.N("IgniteDamage")
			} else {
				output.SetN("WithIgniteDPS", baseDPS+output.N("TotalIgniteDPS"))
			}
		} else if skillData.Flag("showAverage") {
			output.SetN("WithIgniteDPS", baseDPS+output.N("IgniteDamage"))
			combinedAvg += output.N("IgniteDamage")
		} else {
			output.SetN("WithIgniteDPS", baseDPS+output.N("IgniteDPS"))
		}
	} else {
		output.SetN("WithIgniteDPS", baseDPS)
	}
	if skillFlags["monsterExplode"] {
		output.SetN("CombinedAvgToMonsterLife", combinedAvg/c.monsterLife*100)
	}
	if skillFlags["bleed"] {
		if skillData.Flag("showAverage") {
			output.SetN("WithBleedDPS", baseDPS+output.N("BleedDamage"))
			combinedAvg += output.N("BleedDamage")
		} else {
			output.SetN("WithBleedDPS", baseDPS+output.N("BleedDPS"))
		}
	} else {
		output.SetN("WithBleedDPS", baseDPS)
	}
	if skillFlags["impale"] {
		var impaleDPS float64
		if skillFlags["attack"] && skillData.Flag("doubleHitsWhenDualWielding") && skillFlags["bothWeaponAttack"] {
			// separately combine
			mainMod, offMod := 1.0, 1.0
			if c.mainHandStats.Has("ImpaleModifier") {
				mainMod = c.mainHandStats.N("ImpaleModifier")
			}
			if c.offHandStats.Has("ImpaleModifier") {
				offMod = c.offHandStats.N("ImpaleModifier")
			}
			mainHandImpaleDPS := c.mainHandStats.N("impaleStoredHitAvg") * (mainMod - 1) *
				c.mainHandStats.N("HitChance") / 100 * skillData.N("dpsMultiplier")
			offHandImpaleDPS := c.offHandStats.N("impaleStoredHitAvg") * (offMod - 1) *
				c.offHandStats.N("HitChance") / 100 * skillData.N("dpsMultiplier")
			impaleDPS = mainHandImpaleDPS + offHandImpaleDPS
		} else {
			mod := 1.0
			if output.Has("ImpaleModifier") {
				mod = output.N("ImpaleModifier")
			}
			impaleDPS = output.N("impaleStoredHitAvg") * (mod - 1) * output.N("HitChance") / 100 * skillData.N("dpsMultiplier")
		}
		if output.N("ImpaleDuration") <= 0 {
			impaleDPS = 0
		}
		if skillData.Flag("showAverage") {
			output.SetN("WithImpaleDPS", output.N("AverageDamage")+impaleDPS)
			combinedAvg += impaleDPS
		} else {
			skillFlags["notAverage"] = true
			speed := output.N("Speed")
			if output.Has("HitSpeed") {
				speed = output.N("HitSpeed")
			}
			impaleDPS = impaleDPS * speed
			output.SetN("WithImpaleDPS", output.N("TotalDPS")+impaleDPS)
		}
		if c.quantityMultiplier > 1 {
			impaleDPS = impaleDPS * c.quantityMultiplier
		}
		output.SetN("ImpaleDPS", impaleDPS)
		output.SetN("CombinedDPS", output.N("CombinedDPS")+impaleDPS)
	}
	output.SetN("CombinedAvg", combinedAvg)

	bestCull := 1.0
	if m := c.activeSkill.Mirage; m != nil && m.Output != nil && m.Output.Has("TotalDPS") {
		mo := m.Output
		mirageCount := m.Count
		output.SetN("MirageDPS", mo.N("TotalDPS")*mirageCount)
		output.SetN("CombinedDPS", output.N("CombinedDPS")+mo.N("TotalDPS")*mirageCount)
		// Plain assignments: absent on the mirage side stays absent here.
		output.Set("MirageBurningGroundDPS", mo.Get("BurningGroundDPS"))
		output.Set("MirageCausticGroundDPS", mo.Get("CausticGroundDPS"))

		if mo.Has("IgniteDPS") && mo.N("IgniteDPS") > output.N("IgniteDPS") {
			output.SetN("MirageDPS", output.N("MirageDPS")+mo.N("IgniteDPS"))
			output.SetN("IgniteDPS", 0.0)
		}
		if mo.Has("BleedDPS") && mo.N("BleedDPS") > output.N("BleedDPS") {
			output.SetN("MirageDPS", output.N("MirageDPS")+mo.N("BleedDPS"))
			output.SetN("BleedDPS", 0.0)
		}
		if mo.Has("PoisonDPS") {
			v := mo.N("PoisonDPS")
			output.SetN("MirageDPS", output.N("MirageDPS")+v*mirageCount)
			output.SetN("CombinedDPS", output.N("CombinedDPS")+v*mirageCount)
		}
		if mo.Has("ImpaleDPS") {
			v := mo.N("ImpaleDPS")
			output.SetN("MirageDPS", output.N("MirageDPS")+v*mirageCount)
			output.SetN("CombinedDPS", output.N("CombinedDPS")+v*mirageCount)
		}
		if mo.Has("DecayDPS") {
			v := mo.N("DecayDPS")
			output.SetN("MirageDPS", output.N("MirageDPS")+v)
			output.SetN("CombinedDPS", output.N("CombinedDPS")+v)
		}
		if mo.Flag("TotalDot") && (skillFlags["DotCanStack"] || !output.Flag("TotalDot")) {
			n := 1.0
			if skillFlags["DotCanStack"] {
				n = mirageCount
			}
			output.SetN("MirageDPS", output.N("MirageDPS")+mo.N("TotalDot")*n)
			output.SetN("CombinedDPS", output.N("CombinedDPS")+mo.N("TotalDot")*n)
		}
		if mo.N("CullMultiplier") > 1 {
			bestCull = mo.N("CullMultiplier")
		}
	}

	totalDotDPS := output.N("TotalDot") + output.N("TotalPoisonDPS") +
		math.Max(output.N("CausticGroundDPS"), output.N("MirageCausticGroundDPS"))
	if output.Flag("TotalIgniteDPS") {
		totalDotDPS += output.N("TotalIgniteDPS")
	} else {
		totalDotDPS += output.N("IgniteDPS")
	}
	totalDotDPS += math.Max(output.N("BurningGroundDPS"), output.N("MirageBurningGroundDPS")) +
		output.N("BleedDPS") + output.N("CorruptingBloodDPS") + output.N("DecayDPS")
	output.SetN("TotalDotDPS", math.Min(totalDotDPS, data.Misc.DotDpsCap))
	if output.N("TotalDotDPS") != totalDotDPS {
		output.SetFlag("showTotalDotDPS", true)
	}
	if !skillData.Flag("showAverage") {
		output.SetN("CombinedDPS", output.N("CombinedDPS")+output.N("TotalDotDPS"))
	}

	bestCull = math.Max(bestCull, output.N("CullMultiplier"))
	output.SetN("CullingDPS", output.N("CombinedDPS")*(bestCull-1))
	output.SetN("ReservationDPS", output.N("CombinedDPS")*(output.N("ReservationDpsMultiplier")-1))
	output.SetN("CombinedDPS", output.N("CombinedDPS")*bestCull*output.N("ReservationDpsMultiplier"))
}

var _ = strings.Contains
