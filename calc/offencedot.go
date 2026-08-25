// CalcOffence.lua L5605-6168: decay, dropped burning ground, the generic
// damage-over-time section, cost per second, self-hit damage and the
// combined DPS estimate.
package calc

import (
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
	if ov := db.Override(cfg, "DotMultiplier"); truthy(ov) {
		return anyNum(ov)
	}
	return db.Sum("BASE", cfg, "DotMultiplier") + db.Sum("BASE", cfg, typeName+"DotMultiplier")
}

// offenceDot ports L5605-6168.
func (env *Env) offenceDot(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	skillFlags, enemyDB, modDB := c.skillFlags, c.enemyDB, c.modDB
	activeSkill, output := c.activeSkill, c.output
	d := env.Data

	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(d.Misc.BuffExpirationSlowCap, Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}

	if skillFlags["hit"] && truthy(skillData["decay"]) && c.canDeal["Chaos"] {
		// Calculate DPS for Essence of Delirium's Decay effect
		skillFlags["decay"] = true
		skillCfgKeywords := int64(0)
		if skillCfg.KeywordFlags != nil {
			skillCfgKeywords = *skillCfg.KeywordFlags
		}
		dotCfg := &modstore.Cfg{
			SkillName:       skillCfg.SkillName,
			SkillPart:       skillCfg.SkillPart,
			SkillTypes:      skillCfg.SkillTypes,
			SummonSkillName: skillCfg.SummonSkillName,
			SlotName:        skillCfg.SlotName,
			Flags:           i64p(modparser.ModFlag.Dot),
			KeywordFlags:    i64p((skillCfgKeywords &^ modparser.KeywordFlag.Hit) | modparser.KeywordFlag.ChaosDot),
		}
		activeSkill.DecayCfg = dotCfg
		effMult := 1.0
		if env.ModeEffective {
			resist := env.calcResistForType(c, "Chaos", dotCfg)
			takenInc := enemyDB.Sum("INC", nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			takenMore := enemyDB.More(nil, "DamageTaken", "DamageTakenOverTime", "ChaosDamageTaken", "ChaosDamageTakenOverTime")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
			output["DecayEffMult"] = effMult
		}
		inc := skillModList.Sum("INC", dotCfg, "Damage", "ChaosDamage")
		more := skillModList.More(dotCfg, "Damage", "ChaosDamage")
		mult := dotMultiRaw(skillModList, dotCfg, "Chaos")
		output["DecayDPS"] = anyNum(skillData["decay"]) * (1 + inc/100) * more * (1 + mult/100) * effMult
		output["DecayDuration"] = 8 * debuffDurationMult
	}

	baseDropsBurningGround := modDB.Sum("BASE", nil, "DropsBurningGround")
	if baseDropsBurningGround > 0 && c.canDeal["Fire"] {
		dotTakenCfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Dot), KeywordFlags: i64p(0)}
		dotTypeCfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Dot), KeywordFlags: i64p(modparser.KeywordFlag.FireDot)}
		effMult := 1.0
		if env.ModeEffective {
			resist := env.calcResistForType(c, "Fire", dotTypeCfg)
			takenInc := enemyDB.Sum("INC", dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", "FireDamageTaken", "FireDamageTakenOverTime", "ElementalDamageTaken")
			effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
		}
		inc := modDB.Sum("INC", dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
		more := modDB.More(dotTypeCfg, "Damage", "FireDamage", "ElementalDamage")
		mult := dotMultiDB(modDB, dotTypeCfg, "Fire")
		total := baseDropsBurningGround * (1 + inc/100) * more * (1 + mult/100) * effMult
		if !truthy(output["BurningGroundDPS"]) || outNum(output, "BurningGroundDPS") < total {
			output["BurningGroundDPS"] = total
			output["BurningGroundFromIgnite"] = false
		}
	}

	// Calculate skill DOT components
	skillCfgFlags := int64(0)
	if skillCfg.Flags != nil {
		skillCfgFlags = *skillCfg.Flags
	}
	skillCfgKeywords := int64(0)
	if skillCfg.KeywordFlags != nil {
		skillCfgKeywords = *skillCfg.KeywordFlags
	}
	dotFlags := modparser.ModFlag.Dot | skillCfgFlags
	clearFlag := func(flag int64, keep bool) {
		if dotFlags|flag == dotFlags && !keep {
			dotFlags &^= flag
		}
	}
	clearFlag(modparser.ModFlag.Area, truthy(skillData["dotIsArea"]))
	clearFlag(modparser.ModFlag.Projectile, truthy(skillData["dotIsProjectile"]))
	clearFlag(modparser.ModFlag.Spell, truthy(skillData["dotIsSpell"]))
	clearFlag(modparser.ModFlag.Attack, truthy(skillData["dotIsAttack"]))
	clearFlag(modparser.ModFlag.Hit, truthy(skillData["dotIsHit"]))
	dotCfg := &modstore.Cfg{
		SkillName:       skillCfg.SkillName,
		SkillPart:       skillCfg.SkillPart,
		SkillTypes:      skillCfg.SkillTypes,
		SummonSkillName: skillCfg.SummonSkillName,
		SlotName:        skillCfg.SlotName,
		Flags:           i64p(dotFlags),
		KeywordFlags:    i64p(skillCfgKeywords &^ modparser.KeywordFlag.Hit),
	}

	// spell_damage_modifiers_apply_to_skill_dot does not apply to enemy damage taken
	dotTakenCfg := copyCfg(dotCfg)
	if truthy(skillData["dotIsSpell"]) {
		dotTakenCfg.Flags = i64p(dotFlags &^ modparser.ModFlag.Spell)
	}

	activeSkill.DotCfg = dotCfg
	activeSkill.DotTypeCfg = map[string]*modstore.Cfg{}
	output["TotalDotInstance"] = 0.0

	env.runSkillFunc(c, "preDotFunc")

	// Section handles generic damage over time
	for _, damageType := range dmgTypeList {
		dotTypeCfg := copyCfg(dotCfg)
		dotTypeCfg.KeywordFlags = i64p(*dotCfg.KeywordFlags | keywordDotFlag(damageType))
		activeSkill.DotTypeCfg[damageType] = dotTypeCfg
		baseVal := 0.0
		if c.canDeal[damageType] {
			baseVal = anyNum(skillData[damageType+"Dot"])
		}
		if baseVal > 0 || outNum(output, damageType+"Dot") > 0 {
			skillFlags["dot"] = true
			effMult := 1.0
			// Section handles Enemy Damage Taken based on Configs
			if env.ModeEffective {
				resist := 0.0
				takenInc := enemyDB.Sum("INC", dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
				takenMore := enemyDB.More(dotTakenCfg, "DamageTaken", "DamageTakenOverTime", damageType+"DamageTaken", damageType+"DamageTakenOverTime")
				if isElementalRes[damageType] {
					takenInc += enemyDB.Sum("INC", dotTakenCfg, "ElementalDamageTaken")
					takenMore *= enemyDB.More(dotTakenCfg, "ElementalDamageTaken")
				}
				if damageType == "Physical" {
					resist = math.Max(0, math.Min(enemyDB.Sum("BASE", nil, "PhysicalDamageReduction"), d.Misc.EnemyPhysicalDamageReductionCap))
				} else {
					resist = env.calcResistForType(c, damageType, dotTypeCfg)
				}
				effMult = (1 - resist/100) * (1 + takenInc/100) * takenMore
				output[damageType+"DotEffMult"] = effMult
			}
			names := optName(isElementalRes[damageType], []string{"Damage", damageType + "Damage"}, "ElementalDamage")
			inc := skillModList.Sum("INC", dotTypeCfg, names...)
			if skillModList.Flag(nil, "dotIsHeraldOfAsh") {
				inc = math.Max(inc-skillModList.Sum("INC", skillCfg, names...), 0)
			}
			more := skillModList.More(dotTypeCfg, names...)
			mult := dotMultiRaw(skillModList, dotTypeCfg, damageType)
			aura := 1.0
			if activeSkill.SkillTypes[modparser.SkillType.Aura] && !activeSkill.SkillTypes[modparser.SkillType.RemoteMined] &&
				!activeSkill.SkillTypes[modparser.SkillType.Banner] {
				aura = Mod(skillModList, dotTypeCfg, "AuraEffect")
			}
			total := baseVal * (1 + inc/100) * more * (1 + mult/100) * aura * effMult
			if !truthy(output[damageType+"Dot"]) || outNum(output, damageType+"Dot") == 0 {
				output[damageType+"Dot"] = total
				output["TotalDotInstance"] = math.Min(outNum(output, "TotalDotInstance")+total, d.Misc.DotDpsCap)
			} else {
				output["TotalDotInstance"] = math.Min(outNum(output, "TotalDotInstance")+total+outNum(output, damageType+"Dot"), d.Misc.DotDpsCap)
			}
		}
	}

	switch {
	case skillModList.Flag(nil, "DotCanStack"):
		skillFlags["DotCanStack"] = true
		speed := outNum(output, "Speed")
		// Check if skill is being triggered via Mine or Trap
		if *dotCfg.KeywordFlags&modparser.KeywordFlag.Mine != 0 {
			speed = outNum(output, "MineLayingSpeed")
		} else if *dotCfg.KeywordFlags&modparser.KeywordFlag.Trap != 0 {
			speed = outNum(output, "TrapThrowingSpeed")
		}
		output["TotalDot"] = math.Min(outNum(output, "TotalDotInstance")*speed*outNum(output, "Duration")*
			anyNum(skillData["dpsMultiplier"])*c.quantityMultiplier, d.Misc.DotDpsCap)
		output["TotalDotCalcSection"] = output["TotalDot"]
	case skillModList.Flag(nil, "dotIsBurningGround"):
		output["TotalDot"] = 0.0
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
		if !truthy(output["BurningGroundDPS"]) || outNum(output, "BurningGroundDPS") < outNum(output, "TotalDotInstance") {
			output["BurningGroundDPS"] = math.Max(outNum(output, "BurningGroundDPS"), outNum(output, "TotalDotInstance"))
			output["BurningGroundFromIgnite"] = false
		}
	case skillModList.Flag(nil, "dotIsCausticGround"):
		output["TotalDot"] = 0.0
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
		if !truthy(output["CausticGroundDPS"]) || outNum(output, "CausticGroundDPS") < outNum(output, "TotalDotInstance") {
			output["CausticGroundDPS"] = math.Max(outNum(output, "CausticGroundDPS"), outNum(output, "TotalDotInstance"))
			output["CausticGroundFromPoison"] = false
		}
	case skillModList.Flag(nil, "dotIsCorruptingBlood"):
		output["TotalDot"] = 0.0
		output["TotalDotCalcSection"] = output["TotalDotInstance"]
		if !truthy(output["CorruptingBloodDPS"]) || outNum(output, "CorruptingBloodDPS") < outNum(output, "TotalDotInstance") {
			output["CorruptingBloodDPS"] = math.Max(outNum(output, "CorruptingBloodDPS"), outNum(output, "TotalDotInstance"))
		}
	default:
		if skillModList.Flag(nil, "DotCanStackAsTotems") && skillFlags["totem"] {
			skillFlags["DotCanStack"] = true
		}
		attachedBrandCount := 1.0
		if activeSkill.SkillTypes[modparser.SkillType.Brand] && !truthy(skillData["countsAttachedBrandsInDamage"]) &&
			truthy(output["AttachedBrandCount"]) {
			attachedBrandCount = outNum(output, "AttachedBrandCount")
		}
		if attachedBrandCount > 1 {
			output["TotalDot"] = math.Min(outNum(output, "TotalDotInstance")*attachedBrandCount, d.Misc.DotDpsCap)
		} else {
			output["TotalDot"] = output["TotalDotInstance"]
		}
		output["TotalDotCalcSection"] = output["TotalDot"]
	}

	env.offenceCostPerSecond(c)
	env.offenceSelfHit(c)
	env.offenceCombinedDPS(c)
}

// keywordDotFlag maps a damage type to its KeywordFlag.<Type>Dot bit.
func keywordDotFlag(damageType string) int64 {
	switch damageType {
	case "Physical":
		return modparser.KeywordFlag.PhysicalDot
	case "Lightning":
		return modparser.KeywordFlag.LightningDot
	case "Cold":
		return modparser.KeywordFlag.ColdDot
	case "Fire":
		return modparser.KeywordFlag.FireDot
	case "Chaos":
		return modparser.KeywordFlag.ChaosDot
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
		if !(val.upfront && truthy(output[resource+"HasCost"]) && outNum(output, resource+"Cost") > 0 &&
			!(truthy(output[resource+"PerSecondHasCost"]) && !(eb && skillModList.Sum("BASE", skillCfg, "ManaCostAsEnergyShieldCost") > 0)) &&
			(outNum(output, "Speed") > 0 || truthy(output["Cooldown"]))) {
			continue
		}
		usedResource := resource
		if eb && resource == "Mana" {
			usedResource = "ES"
		}

		repeats := 1.0
		if truthy(output["Repeats"]) {
			repeats = anyNum(output["Repeats"])
		}
		useSpeed := 1.0
		switch {
		case skillFlags["trap"] || skillFlags["mine"]:
			preSpeed := outNum(output, "MineLayingSpeed")
			if truthy(output["TrapThrowingSpeed"]) {
				preSpeed = outNum(output, "TrapThrowingSpeed")
			}
			cooldown, hasCooldown := 0.0, false
			if truthy(output["TrapCooldown"]) {
				cooldown, hasCooldown = outNum(output, "TrapCooldown"), true
			} else if truthy(output["Cooldown"]) {
				cooldown, hasCooldown = outNum(output, "Cooldown"), true
			}
			if hasCooldown && cooldown > 0 {
				useSpeed = 1 / cooldown
			} else {
				useSpeed = preSpeed
			}
		case skillFlags["totem"]:
			if truthy(output["Cooldown"]) && outNum(output, "Cooldown") > 0 {
				if outNum(output, "TotemPlacementSpeed") > 0 {
					useSpeed = outNum(output, "TotemPlacementSpeed")
				} else {
					useSpeed = 1 / outNum(output, "Cooldown")
				}
			} else {
				useSpeed = outNum(output, "TotemPlacementSpeed")
			}
			useSpeed /= repeats
		case skillModList.Flag(nil, "HasSeals") && skillModList.Flag(nil, "UseMaxUnleash") &&
			truthy(env.PlayerMainSkill.SkillData["hitTimeOverride"]):
			useSpeed = anyNum(env.PlayerMainSkill.SkillData["hitTimeOverride"]) / repeats
		default:
			if truthy(output["Cooldown"]) && outNum(output, "Cooldown") > 0 {
				if outNum(output, "Speed") > 0 {
					useSpeed = outNum(output, "Speed")
				} else {
					useSpeed = 1 / outNum(output, "Cooldown")
				}
			} else {
				useSpeed = outNum(output, "Speed")
			}
			useSpeed /= repeats
		}
		_ = skillData

		output[usedResource+"PerSecondHasCost"] = true
		output[usedResource+"PerSecondCost"] = outNum(output, usedResource+"PerSecondCost") + outNum(output, resource+"Cost")*useSpeed
	}
}

// offenceCombinedDPS ports L5984-6168.
func (env *Env) offenceCombinedDPS(c *offenceCtx) {
	skillData, skillFlags, output := c.skillData, c.skillFlags, c.output
	d := env.Data

	baseKey := "TotalDPS"
	if truthy(skillData["showAverage"]) {
		baseKey = "AverageDamage"
	}
	baseDPS := outNum(output, baseKey)
	output["CombinedDPS"] = baseDPS
	combinedAvg := baseDPS
	if skillFlags["dot"] {
		output["WithDotDPS"] = baseDPS + outNum(output, "TotalDot")
	}
	if c.quantityMultiplier > 1 && truthy(output["TotalPoisonDPS"]) {
		output["TotalPoisonDPS"] = math.Min(outNum(output, "TotalPoisonDPS")*c.quantityMultiplier, d.Misc.DotDpsCap)
	}
	if truthy(skillData["showAverage"]) {
		combinedAvg += outNum(output, "TotalPoisonAverageDamage")
		output["WithPoisonDPS"] = baseDPS + outNum(output, "TotalPoisonAverageDamage")
	} else {
		output["WithPoisonDPS"] = baseDPS + outNum(output, "TotalPoisonDPS")
	}
	if skillFlags["ignite"] {
		if skillFlags["igniteCanStack"] {
			if truthy(skillData["showAverage"]) {
				combinedAvg = outNum(output, "CombinedDPS") + outNum(output, "IgniteDamage")
			} else {
				output["WithIgniteDPS"] = baseDPS + outNum(output, "TotalIgniteDPS")
			}
		} else if truthy(skillData["showAverage"]) {
			output["WithIgniteDPS"] = baseDPS + outNum(output, "IgniteDamage")
			combinedAvg += outNum(output, "IgniteDamage")
		} else {
			output["WithIgniteDPS"] = baseDPS + outNum(output, "IgniteDPS")
		}
	} else {
		output["WithIgniteDPS"] = baseDPS
	}
	if skillFlags["monsterExplode"] {
		output["CombinedAvgToMonsterLife"] = combinedAvg / c.monsterLife * 100
	}
	if skillFlags["bleed"] {
		if truthy(skillData["showAverage"]) {
			output["WithBleedDPS"] = baseDPS + outNum(output, "BleedDamage")
			combinedAvg += outNum(output, "BleedDamage")
		} else {
			output["WithBleedDPS"] = baseDPS + outNum(output, "BleedDPS")
		}
	} else {
		output["WithBleedDPS"] = baseDPS
	}
	if skillFlags["impale"] {
		var impaleDPS float64
		if skillFlags["attack"] && truthy(skillData["doubleHitsWhenDualWielding"]) && skillFlags["bothWeaponAttack"] {
			// separately combine
			mainMod, offMod := 1.0, 1.0
			if truthy(c.mainHandStats["ImpaleModifier"]) {
				mainMod = anyNum(c.mainHandStats["ImpaleModifier"])
			}
			if truthy(c.offHandStats["ImpaleModifier"]) {
				offMod = anyNum(c.offHandStats["ImpaleModifier"])
			}
			mainHandImpaleDPS := anyNum(c.mainHandStats["impaleStoredHitAvg"]) * (mainMod - 1) *
				anyNum(c.mainHandStats["HitChance"]) / 100 * anyNum(skillData["dpsMultiplier"])
			offHandImpaleDPS := anyNum(c.offHandStats["impaleStoredHitAvg"]) * (offMod - 1) *
				anyNum(c.offHandStats["HitChance"]) / 100 * anyNum(skillData["dpsMultiplier"])
			impaleDPS = mainHandImpaleDPS + offHandImpaleDPS
		} else {
			mod := 1.0
			if truthy(output["ImpaleModifier"]) {
				mod = outNum(output, "ImpaleModifier")
			}
			impaleDPS = outNum(output, "impaleStoredHitAvg") * (mod - 1) * outNum(output, "HitChance") / 100 * anyNum(skillData["dpsMultiplier"])
		}
		if outNum(output, "ImpaleDuration") <= 0 {
			impaleDPS = 0
		}
		if truthy(skillData["showAverage"]) {
			output["WithImpaleDPS"] = outNum(output, "AverageDamage") + impaleDPS
			combinedAvg += impaleDPS
		} else {
			skillFlags["notAverage"] = true
			speed := outNum(output, "Speed")
			if truthy(output["HitSpeed"]) {
				speed = outNum(output, "HitSpeed")
			}
			impaleDPS = impaleDPS * speed
			output["WithImpaleDPS"] = outNum(output, "TotalDPS") + impaleDPS
		}
		if c.quantityMultiplier > 1 {
			impaleDPS = impaleDPS * c.quantityMultiplier
		}
		output["ImpaleDPS"] = impaleDPS
		output["CombinedDPS"] = outNum(output, "CombinedDPS") + impaleDPS
	}
	output["CombinedAvg"] = combinedAvg

	bestCull := 1.0
	if m := c.activeSkill.Mirage; m != nil && m.Output != nil && truthy(m.Output["TotalDPS"]) {
		mo := m.Output
		mirageCount := m.Count
		output["MirageDPS"] = anyNum(mo["TotalDPS"]) * mirageCount
		output["CombinedDPS"] = outNum(output, "CombinedDPS") + anyNum(mo["TotalDPS"])*mirageCount
		// Plain assignments: absent on the mirage side stays absent here.
		assignKV(output, "MirageBurningGroundDPS", mo["BurningGroundDPS"])
		assignKV(output, "MirageCausticGroundDPS", mo["CausticGroundDPS"])

		if truthy(mo["IgniteDPS"]) && anyNum(mo["IgniteDPS"]) > outNum(output, "IgniteDPS") {
			output["MirageDPS"] = outNum(output, "MirageDPS") + anyNum(mo["IgniteDPS"])
			output["IgniteDPS"] = 0.0
		}
		if truthy(mo["BleedDPS"]) && anyNum(mo["BleedDPS"]) > outNum(output, "BleedDPS") {
			output["MirageDPS"] = outNum(output, "MirageDPS") + anyNum(mo["BleedDPS"])
			output["BleedDPS"] = 0.0
		}
		if v, ok := mo["PoisonDPS"]; ok {
			output["MirageDPS"] = outNum(output, "MirageDPS") + anyNum(v)*mirageCount
			output["CombinedDPS"] = outNum(output, "CombinedDPS") + anyNum(v)*mirageCount
		}
		if v, ok := mo["ImpaleDPS"]; ok {
			output["MirageDPS"] = outNum(output, "MirageDPS") + anyNum(v)*mirageCount
			output["CombinedDPS"] = outNum(output, "CombinedDPS") + anyNum(v)*mirageCount
		}
		if v, ok := mo["DecayDPS"]; ok {
			output["MirageDPS"] = outNum(output, "MirageDPS") + anyNum(v)
			output["CombinedDPS"] = outNum(output, "CombinedDPS") + anyNum(v)
		}
		if truthy(mo["TotalDot"]) && (skillFlags["DotCanStack"] || !truthy(output["TotalDot"])) {
			n := 1.0
			if skillFlags["DotCanStack"] {
				n = mirageCount
			}
			output["MirageDPS"] = outNum(output, "MirageDPS") + anyNum(mo["TotalDot"])*n
			output["CombinedDPS"] = outNum(output, "CombinedDPS") + anyNum(mo["TotalDot"])*n
		}
		if anyNum(mo["CullMultiplier"]) > 1 {
			bestCull = anyNum(mo["CullMultiplier"])
		}
	}

	totalDotDPS := outNum(output, "TotalDot") + outNum(output, "TotalPoisonDPS") +
		math.Max(outNum(output, "CausticGroundDPS"), outNum(output, "MirageCausticGroundDPS"))
	if truthy(output["TotalIgniteDPS"]) {
		totalDotDPS += outNum(output, "TotalIgniteDPS")
	} else {
		totalDotDPS += outNum(output, "IgniteDPS")
	}
	totalDotDPS += math.Max(outNum(output, "BurningGroundDPS"), outNum(output, "MirageBurningGroundDPS")) +
		outNum(output, "BleedDPS") + outNum(output, "CorruptingBloodDPS") + outNum(output, "DecayDPS")
	output["TotalDotDPS"] = math.Min(totalDotDPS, d.Misc.DotDpsCap)
	if outNum(output, "TotalDotDPS") != totalDotDPS {
		output["showTotalDotDPS"] = true
	}
	if !truthy(skillData["showAverage"]) {
		output["CombinedDPS"] = outNum(output, "CombinedDPS") + outNum(output, "TotalDotDPS")
	}

	bestCull = math.Max(bestCull, outNum(output, "CullMultiplier"))
	output["CullingDPS"] = outNum(output, "CombinedDPS") * (bestCull - 1)
	output["ReservationDPS"] = outNum(output, "CombinedDPS") * (outNum(output, "ReservationDpsMultiplier") - 1)
	output["CombinedDPS"] = outNum(output, "CombinedDPS") * bestCull * outNum(output, "ReservationDpsMultiplier")
}

var _ = strings.Contains
