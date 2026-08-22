// Ports of the hand-written skill callbacks that Data/Skills/*.lua attaches
// to granted effects (initialFunc, preSkillTypeFunc, preDamageFunc,
// postCritFunc, preDotFunc). The generated data tables carry an UnportedFn
// marker for each one; runSkillFunc consults this registry first and panics
// on anything still unported, so a corpus build can never silently skip a
// callback.
package calc

import (
	"math"
	"strconv"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// skillFunc is one ported callback. It gets the same reach the Lua closure
// has: the active skill, the pass-independent output, and the environment.
type skillFunc func(env *Env, c *offenceCtx)

// skillFuncs is keyed "<grantedEffectId>:<callbackName>".
var skillFuncs = map[string]skillFunc{
	"Cyclone:initialFunc":              cycloneInitialFunc("Skill:Cyclone"),
	"CycloneAltX:initialFunc":          cycloneInitialFunc("Skill:CycloneAltX"),
	"VaalCyclone:initialFunc":          cycloneInitialFunc("Skill:Cyclone"),
	"BloodSacramentUnique:initialFunc": bloodSacramentInitialFunc,
	"EnemyExplode:preDamageFunc":       enemyExplodePreDamageFunc,
	"StormBrand:preDamageFunc":         brandHitTimeOverride,
	"RighteousFire:preDamageFunc":      righteousFirePreDamageFunc,
	"BlazingSalvo:preDamageFunc":       blazingSalvoPreDamageFunc,
	"ShrapnelBallista:preDamageFunc":   shrapnelBallistaPreDamageFunc,
}

// cycloneInitialFunc ports the Cyclone family's initialFunc: the melee range
// the skill's area scales with. The three copies differ only in mod source.
func cycloneInitialFunc(source string) skillFunc {
	return func(env *Env, c *offenceCtx) {
		activeSkill := c.activeSkill
		skillModList := activeSkill.SkillModList
		actor := c.actor
		rng := 0.0
		if activeSkill.SkillFlags["weapon1Attack"] && truthy(actor.ms.WeaponData1["range"]) {
			weapon1RangeBonus := skillModList.Sum("BASE", activeSkill.Weapon1Cfg, "MeleeWeaponRange") +
				10*skillModList.Sum("BASE", activeSkill.Weapon1Cfg, "MeleeWeaponRangeMetre") +
				anyNum(actor.ms.WeaponData1["rangeBonus"])
			if activeSkill.SkillFlags["weapon2Attack"] && truthy(actor.ms.WeaponData2["range"]) {
				// dual wield average
				rng = (weapon1RangeBonus + skillModList.Sum("BASE", activeSkill.Weapon2Cfg, "MeleeWeaponRange") +
					10*skillModList.Sum("BASE", activeSkill.Weapon2Cfg, "MeleeWeaponRangeMetre") +
					anyNum(actor.ms.WeaponData2["rangeBonus"])) / 2
			} else {
				// primary hand attack
				rng = weapon1RangeBonus
			}
		} else {
			// unarmed
			rng = skillModList.Sum("BASE", activeSkill.SkillCfg, "UnarmedRange") +
				10*skillModList.Sum("BASE", activeSkill.SkillCfg, "UnarmedRangeMetre")
		}
		skillModList.AddMod(newMod("Multiplier:AdditionalMeleeRange", "BASE", rng, source))
	}
}

// bloodSacramentInitialFunc ports the Blood Sacrament (Sanguimancy) callback.
func bloodSacramentInitialFunc(env *Env, c *offenceCtx) {
	if outNum(c.output, "LifeReservedPercent") >= 100 {
		return
	}
	skillData := c.skillData
	lifeReservedPercent := 3.0
	if truthy(skillData["LifeReservedPercent"]) {
		lifeReservedPercent = anyNum(skillData["LifeReservedPercent"])
	}
	// `skillData.LifeReservedBase or math.huge`
	lifeReserved := mathHuge
	if truthy(skillData["LifeReservedBase"]) {
		lifeReserved = anyNum(skillData["LifeReservedBase"])
	}
	c.skillModList.AddMod(newMod("Multiplier:ChannelledLifeReservedPercentPerStage", "BASE", lifeReservedPercent, "Blood Sacrament"))
	c.skillModList.AddMod(newMod("Multiplier:ChannelledLifeReservedPerStage", "BASE", lifeReserved, "Blood Sacrament"))
}

// mathHuge is Lua's math.huge.
var mathHuge = math.Inf(1)

// explodeSourceKey ports `explodeSource.modSource or "Tree:"..explodeSource.id`.
func explodeSourceKey(src any) string {
	switch t := src.(type) {
	case *Item:
		if t.In.ModSource != nil {
			return *t.In.ModSource
		}
	case *NodeInput:
		return "Tree:" + strconv.FormatInt(int64(t.ID), 10)
	}
	panic("offence: explode source without a modSource")
}

// enemyExplodePreDamageFunc ports the EnemyExplode preDamageFunc
// (Data/Skills/other.lua L6076): which damage types the corpse explosion
// deals and the chance it happens.
func enemyExplodePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	explodeChance := 0.0
	part := anyNum(activeSkill.SkillPart)
	if part != 3 {
		src := activeSkill.ActiveEffect.SrcInstance.ExplodeSource
		activeEffectSource := explodeSourceKey(src)
		for _, entry := range skillModList.Tabulate("LIST", skillCfg, "ExplodeMod") {
			if entry.Mod.Source != activeEffectSource {
				continue
			}
			tag, _ := entry.Value.(modparser.Tag)
			typ := str(tag["type"])
			amount := anyNum(tag["amount"])
			if typ == "RandomElement" {
				skillData["FireEffectiveExplodePercentage"] = amount / 3
				skillData["ColdEffectiveExplodePercentage"] = amount / 3
				skillData["LightningEffectiveExplodePercentage"] = amount / 3
			} else {
				skillData[typ+"EffectiveExplodePercentage"] = amount
			}
			if part == 2 {
				explodeChance = 1
			} else {
				explodeChance = anyNum(tag["chance"])
			}
		}
	} else {
		// Every loop below is a commutative accumulation, so the reference's
		// pairs() order does not reach the result.
		type amountChance map[float64]float64
		typeAmountChances := map[string]amountChance{}
		for _, value := range skillModList.List(skillCfg, "ExplodeMod") {
			tag, _ := value.(modparser.Tag)
			typ := str(tag["type"])
			ac := typeAmountChances[typ]
			if ac == nil {
				ac = amountChance{}
				typeAmountChances[typ] = ac
			}
			ac[anyNum(tag["amount"])] += anyNum(tag["chance"])
		}
		for typ, ac := range typeAmountChances {
			physExplodeChance := 0.0
			for amount, chance := range ac {
				amountXChance := amount * chance
				if typ == "RandomElement" {
					for _, ele := range []string{"Fire", "Cold", "Lightning"} {
						skillData[ele+"EffectiveExplodePercentage"] = anyNum(skillData[ele+"EffectiveExplodePercentage"]) + amountXChance/3
					}
				} else {
					skillData[typ+"EffectiveExplodePercentage"] = anyNum(skillData[typ+"EffectiveExplodePercentage"]) + amountXChance
				}
				if typ == "Physical" {
					physExplodeChance = 1 - ((1 - physExplodeChance) * (1 - chance))
				}
				explodeChance = 1 - ((1 - explodeChance) * (1 - chance))
			}
			if typ == "Physical" && physExplodeChance != 0 {
				skillModList.AddMod(newMod("CalcArmourAsThoughDealing", "MORE", 100/math.Min(physExplodeChance, 1)-100))
			}
		}
	}
	output["ExplodeChance"] = math.Min(explodeChance*100, 100)
}

// brandHitTimeOverride ports the brand family's preDamageFunc: the brand's
// activation frequency becomes the skill's hit time.
func brandHitTimeOverride(env *Env, c *offenceCtx) {
	activeSkill := c.activeSkill
	skillModList, skillCfg, skillData := activeSkill.SkillModList, activeSkill.SkillCfg, activeSkill.SkillData
	skillData["hitTimeOverride"] = anyNum(skillData["repeatFrequency"]) /
		(1 + skillModList.Sum("INC", skillCfg, "Speed", "BrandActivationFrequency")/100) /
		skillModList.More(skillCfg, "BrandActivationFrequency")
}

// righteousFirePreDamageFunc ports Righteous Fire's preDamageFunc: the burn
// scales off the totem's or the character's own life and energy shield.
func righteousFirePreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillData := activeSkill.SkillData
	if activeSkill.SkillFlags["totem"] && outNum(output, "TotemLife") > 1 {
		skillData["FireDot"] = outNum(output, "TotemLife")*anyNum(skillData["RFLifeMultiplier"]) +
			outNum(output, "TotemEnergyShield")*anyNum(skillData["RFESMultiplier"])
	} else if outNum(output, "LifeUnreserved") > 1 {
		skillData["FireDot"] = outNum(output, "Life")*anyNum(skillData["RFLifeMultiplier"]) +
			outNum(output, "EnergyShield")*anyNum(skillData["RFESMultiplier"])
	}
}

// blazingSalvoPreDamageFunc ports Blazing Salvo's preDamageFunc: the
// "All Projectiles" skill part multiplies DPS by the projectile count.
func blazingSalvoPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	if anyNum(activeSkill.SkillPart) != 2 {
		return
	}
	mult := 1.0
	if truthy(activeSkill.SkillData["dpsMultiplier"]) {
		mult = anyNum(activeSkill.SkillData["dpsMultiplier"])
	}
	activeSkill.SkillData["dpsMultiplier"] = mult * outNum(output, "ProjectileCount")
}

// shrapnelBallistaPreDamageFunc ports Shrapnel Ballista's preDamageFunc: the
// shotgunning overlap multiplies DPS, and splits that return add a
// conditional more-multiplier.
func shrapnelBallistaPreDamageFunc(env *Env, c *offenceCtx) {
	activeSkill, output := c.activeSkill, c.output
	skillModList, skillData := activeSkill.SkillModList, activeSkill.SkillData
	if !skillModList.Flag(nil, "SequentialProjectiles") {
		mult := 1.0
		if truthy(skillData["dpsMultiplier"]) {
			mult = anyNum(skillData["dpsMultiplier"])
		}
		// `overlap or (Rain and ProjectileCount or 1)`
		overlap := 1.0
		if truthy(skillData["ShrapnelBallistaProjectileOverlap"]) {
			overlap = anyNum(skillData["ShrapnelBallistaProjectileOverlap"])
		} else if activeSkill.SkillTypes[modparser.SkillType.Rain] {
			overlap = outNum(output, "ProjectileCount")
		}
		skillData["dpsMultiplier"] = mult * math.Min(overlap, outNum(output, "ProjectileCount"))
	}
	if splitCount := outNum(output, "SplitCount"); splitCount > 0 {
		skillModList.AddMod(newMod("DPS", "MORE", splitCount*100, "Split Return", int64(0),
			modparser.Tag{"type": "Condition", "var": "ReturningProjectile"}))
	}
}
