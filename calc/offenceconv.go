// CalcOffence.lua L1851-2110: corpse/monster explosions, the damage-type
// disable cache, the per-cfg conversion tables, the main/off-hand damage
// passes and the combineStat helper that folds them back together.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// damagePass is one entry of the reference's passList.
type damagePass struct {
	label  string
	source map[string]any
	cfg    *modstore.Cfg
	output map[string]any
}

// convTable resolves `(cfg and cfg.conversionTable) or activeSkill.conversionTable`.
func (c *offenceCtx) convTable(cfg *modstore.Cfg) conversionTable {
	if cfg != nil {
		if ct, ok := c.conversionTables[cfg]; ok {
			return ct
		}
	}
	return c.conversionTbl
}

// offenceConversion ports L1851-1934 and hands the pass list on.
func (env *Env) offenceConversion(c *offenceCtx) {
	actor, skillModList, skillCfg, skillData := c.actor, c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, activeSkill := c.skillFlags, c.output, c.activeSkill
	d := env.Data

	if activeSkill.SkillTypes[modparser.SkillType.Brand] {
		// infoMessage is UI-only; the value it formats is already in output.
		_ = skillData["countsAttachedBrandsInDamage"]
	}

	// Handle corpse and enemy explosions
	monsterLife := 100.0
	if truthy(skillData["corpseLife"]) {
		monsterLife = anyNum(skillData["corpseLife"])
	} else if env.EnemyLevel != 0 {
		monsterLife = luaIndex("monsterLifeTable", d.MonsterLifeTable, int(env.EnemyLevel))
	}
	c.monsterLife = monsterLife
	if truthy(skillData["explodeCorpse"]) && (truthy(skillData["corpseLife"]) || env.EnemyLevel != 0) {
		damageType := "Fire"
		if v := str(skillData["corpseExplosionDamageType"]); v != "" {
			damageType = v
		}
		mult := skillData["corpseExplosionLifeMultiplier"]
		if !truthy(mult) {
			mult = skillData["selfFireExplosionLifeMultiplier"]
		}
		skillData[damageType+"BonusMin"] = monsterLife * anyNum(mult)
		skillData[damageType+"BonusMax"] = monsterLife * anyNum(mult)
	}
	if skillFlags["monsterExplode"] {
		for _, damageType := range dmgTypeList {
			percentage := anyNum(skillData[damageType+"EffectiveExplodePercentage"])
			base := percentage * monsterLife / 100
			skillData[damageType+"Min"] = base
			skillData[damageType+"Max"] = base
		}
	}

	// Cache global damage disabling flags
	c.canDeal = map[string]bool{}
	for _, damageType := range dmgTypeList {
		c.canDeal[damageType] = !skillModList.Flag(skillCfg, "DealNo"+damageType, "DealNoDamage")
	}

	// Calculate damage conversion percentages
	c.conversionTbl = env.buildConversionTable(activeSkill, skillCfg)
	c.conversionTables = map[*modstore.Cfg]conversionTable{}
	if activeSkill.Weapon1Cfg != nil {
		c.conversionTables[activeSkill.Weapon1Cfg] = env.buildConversionTable(activeSkill, activeSkill.Weapon1Cfg)
	}
	if activeSkill.Weapon2Cfg != nil {
		c.conversionTables[activeSkill.Weapon2Cfg] = env.buildConversionTable(activeSkill, activeSkill.Weapon2Cfg)
	}

	// Configure damage passes
	var passList []*damagePass
	if c.isAttack {
		mainHand := map[string]any{}
		offHand := map[string]any{}
		output["MainHand"] = mainHand
		output["OffHand"] = offHand
		// env.keystonesAdded[...] is true-or-nil, so a missing keystone
		// leaves the key absent rather than storing false.
		if env.Keystone.KeystonesAdded["Precise Technique"] {
			output["PreciseTechnique"] = true
		} else {
			delete(output, "PreciseTechnique")
		}
		critOverride := skillModList.Override(skillCfg, "WeaponBaseCritChance")
		if skillFlags["weapon1Attack"] {
			activeSkill.Weapon1Cfg.SkillStats = mainHand
			c.mainHandStats = mainHand
			source := copyWeaponData(actor.ms.WeaponData1)
			if truthy(critOverride) && truthy(source["type"]) && str(source["type"]) != "None" {
				source["CritChance"] = anyNum(critOverride)
			}
			passList = append(passList, &damagePass{
				label: "Main Hand", source: source, cfg: activeSkill.Weapon1Cfg, output: mainHand,
			})
		}
		if skillFlags["weapon2Attack"] {
			activeSkill.Weapon2Cfg.SkillStats = offHand
			c.offHandStats = offHand
			source := copyWeaponData(actor.ms.WeaponData2)
			if truthy(critOverride) && truthy(source["type"]) && str(source["type"]) != "None" {
				source["CritChance"] = anyNum(critOverride)
			}
			if truthy(skillData["CritChance"]) {
				source["CritChance"] = skillData["CritChance"]
			}
			if truthy(skillData["setOffHandPhysicalMin"]) && truthy(skillData["setOffHandPhysicalMax"]) {
				source["PhysicalMin"] = skillData["setOffHandPhysicalMin"]
				source["PhysicalMax"] = skillData["setOffHandPhysicalMax"]
			}
			if truthy(skillData["setOffHandFireMin"]) && truthy(skillData["setOffHandFireMax"]) {
				source["FireMin"] = skillData["setOffHandFireMin"]
				source["FireMax"] = skillData["setOffHandFireMax"]
			}
			if truthy(skillData["setOffHandColdMin"]) && truthy(skillData["setOffHandColdMax"]) {
				source["ColdMin"] = skillData["setOffHandColdMin"]
				source["ColdMax"] = skillData["setOffHandColdMax"]
			}
			if truthy(skillData["attackTime"]) {
				source["AttackRate"] = 1000 / anyNum(skillData["attackTime"])
			}
			passList = append(passList, &damagePass{
				label: "Off Hand", source: source, cfg: activeSkill.Weapon2Cfg, output: offHand,
			})
		}
	} else {
		passList = append(passList, &damagePass{
			label: "Skill", source: skillData, cfg: skillCfg, output: output,
		})
	}
	c.passList = passList

	env.offenceHitRate(c)
}

// copyWeaponData is copyTable(actor.weaponDataN): a shallow copy.
func copyWeaponData(wd map[string]any) map[string]any {
	out := make(map[string]any, len(wd))
	for k, v := range wd {
		out[k] = v
	}
	return out
}

// combineStat ports the local of the same name (L2003): fold the main-hand
// and off-hand pass results into the actor output. `extra` is the reference's
// vararg, used by the CHANCE modes.
func (env *Env) combineStat(c *offenceCtx, stat, mode string, extra string) {
	output, skillFlags, skillData := c.output, c.skillFlags, c.skillData
	main, off := c.mainHandStats, c.offHandStats
	orElse := func() {
		if v := main[stat]; truthy(v) {
			output[stat] = v
		} else if v := off[stat]; truthy(v) {
			output[stat] = v
		} else {
			delete(output, stat)
		}
	}
	switch {
	case mode == "OR" || !skillFlags["bothWeaponAttack"]:
		orElse()
	case mode == "ADD":
		output[stat] = anyNum(main[stat]) + anyNum(off[stat])
	case mode == "AVERAGE":
		output[stat] = (anyNum(main[stat]) + anyNum(off[stat])) / 2
	case mode == "HARMONICMEAN":
		if anyNum(main[stat]) == 0 || anyNum(off[stat]) == 0 {
			output[stat] = 0.0
		} else {
			output[stat] = 2 / (1/anyNum(main[stat]) + 1/anyNum(off[stat]))
		}
	case mode == "CHANCE":
		if truthy(main[stat]) && truthy(off[stat]) {
			mainChance := anyNum(main[extra]) * anyNum(main["HitChance"])
			offChance := anyNum(off[extra]) * anyNum(off["HitChance"])
			if truthy(skillData["doubleHitsWhenDualWielding"]) {
				mainChance /= 10000
				offChance /= 10000
				output[stat] = anyNum(main[stat])*mainChance + anyNum(off[stat])*offChance
			} else {
				mainPortion := mainChance / (mainChance + offChance)
				offPortion := offChance / (mainChance + offChance)
				output[stat] = anyNum(main[stat])*mainPortion + anyNum(off[stat])*offPortion
			}
		} else {
			orElse()
		}
	case mode == "CHANCE_AILMENT":
		if truthy(main[stat]) && truthy(off[stat]) {
			mainChance := anyNum(main[extra]) * anyNum(main["HitChance"])
			offChance := anyNum(off[extra]) * anyNum(off["HitChance"])
			_, _ = mainChance, offChance
			maxInstance := math.Max(anyNum(main[stat]), anyNum(off[stat]))
			minInstance := math.Min(anyNum(main[stat]), anyNum(off[stat]))
			stackName := strings.Replace(stat, "DPS", "", -1) + "Stacks"
			stacks, stacksMax := 1.0, 1.0
			if truthy(output[stackName]) {
				stacks = anyNum(output[stackName])
			}
			if truthy(output[stackName+"Max"]) {
				stacksMax = anyNum(output[stackName+"Max"])
			}
			maxInstanceStacks := math.Min(1, stacks/stacksMax)
			output[stat] = maxInstance*maxInstanceStacks + minInstance*(1-maxInstanceStacks)
		} else {
			orElse()
		}
	case mode == "DPS":
		v := anyNum(main[stat]) + anyNum(off[stat])
		if !truthy(skillData["doubleHitsWhenDualWielding"]) {
			v /= 2
		}
		output[stat] = v
	}
}
