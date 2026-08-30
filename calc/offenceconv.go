// CalcOffence.lua L1851-2110: corpse/monster explosions, the damage-type
// disable cache, the per-cfg conversion tables, the main/off-hand damage
// passes and the combineStat helper that folds them back together.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// damagePass is one entry of the reference's passList.
type damagePass struct {
	label  string
	source *SkillData
	cfg    *modstore.Cfg
	output modstore.Output
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

	// (The reference's Brand block here only builds activeSkill.infoMessage.)

	// Handle corpse and enemy explosions
	monsterLife := 100.0
	if skillData.Flag("corpseLife") {
		monsterLife = skillData.N("corpseLife")
	} else if env.EnemyLevel != 0 {
		monsterLife = monsterLifeAtLevel(int(env.EnemyLevel))
	}
	c.monsterLife = monsterLife
	if skillData.Flag("explodeCorpse") && (skillData.Flag("corpseLife") || env.EnemyLevel != 0) {
		damageType := "Fire"
		if v := skillData.Str("corpseExplosionDamageType"); v != "" {
			damageType = v
		}
		mult := skillData.Get("corpseExplosionLifeMultiplier")
		if !mult.Truthy() {
			mult = skillData.Get("selfFireExplosionLifeMultiplier")
		}
		skillData.SetN(damageType+"BonusMin", monsterLife*mult.Num())
		skillData.SetN(damageType+"BonusMax", monsterLife*mult.Num())
	}
	if skillFlags["monsterExplode"] {
		for _, damageType := range dmgTypeList {
			percentage := skillData.N(damageType + "EffectiveExplodePercentage")
			base := percentage * monsterLife / 100
			skillData.SetN(damageType+"Min", base)
			skillData.SetN(damageType+"Max", base)
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
		mainHand := modstore.Output{}
		offHand := modstore.Output{}
		actor.mainHand, actor.offHand = mainHand, offHand
		// env.keystonesAdded[...] is true-or-nil, so a missing keystone
		// leaves the key absent rather than storing false.
		if env.Keystone.KeystonesAdded["Precise Technique"] {
			output.SetFlag("PreciseTechnique", true)
		} else {
			output.Del("PreciseTechnique")
		}
		critOverride, _ := skillModList.Override(skillCfg, "WeaponBaseCritChance")
		if skillFlags["weapon1Attack"] {
			activeSkill.Weapon1Cfg.SkillStats = mainHand
			c.mainHandStats = mainHand
			source := weaponPassSource(weaponOf(actor.ms.WeaponData1))
			if modparser.Truthy(critOverride) && source.Has("type") && source.Str("type") != "None" {
				source.SetN("CritChance", valueNum(critOverride))
			}
			passList = append(passList, &damagePass{
				label: "Main Hand", source: source, cfg: activeSkill.Weapon1Cfg, output: mainHand,
			})
		}
		if skillFlags["weapon2Attack"] {
			activeSkill.Weapon2Cfg.SkillStats = offHand
			c.offHandStats = offHand
			source := weaponPassSource(weaponOf(actor.ms.WeaponData2))
			if modparser.Truthy(critOverride) && source.Has("type") && source.Str("type") != "None" {
				source.SetN("CritChance", valueNum(critOverride))
			}
			if skillData.Flag("CritChance") {
				source.Set("CritChance", skillData.Get("CritChance"))
			}
			if skillData.Flag("setOffHandPhysicalMin") && skillData.Flag("setOffHandPhysicalMax") {
				source.Set("PhysicalMin", skillData.Get("setOffHandPhysicalMin"))
				source.Set("PhysicalMax", skillData.Get("setOffHandPhysicalMax"))
			}
			if skillData.Flag("setOffHandFireMin") && skillData.Flag("setOffHandFireMax") {
				source.Set("FireMin", skillData.Get("setOffHandFireMin"))
				source.Set("FireMax", skillData.Get("setOffHandFireMax"))
			}
			if skillData.Flag("setOffHandColdMin") && skillData.Flag("setOffHandColdMax") {
				source.Set("ColdMin", skillData.Get("setOffHandColdMin"))
				source.Set("ColdMax", skillData.Get("setOffHandColdMax"))
			}
			if skillData.Has("attackTime") {
				source.SetN("AttackRate", 1000/skillData.N("attackTime"))
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

// weaponPassSource is copyTable(actor.weaponDataN): the damage pass's
// source bag, holding the weapon's stats under the pass keys.
func weaponPassSource(wd *item.WeaponData) *SkillData {
	out := newSkillData()
	if wd == nil {
		return out
	}
	set := func(key string, v float64, present bool) {
		if present {
			out.SetN(key, v)
		}
	}
	if wd.Type != "" {
		out.SetStr("type", wd.Type)
	}
	if wd.Name != "" {
		out.SetStr("name", wd.Name)
	}
	set("AttackRate", wd.AttackRate, wd.AttackRate != 0)
	set("range", wd.Range, wd.Range != 0)
	set("AttackSpeedInc", wd.AttackSpeedInc.V, wd.AttackSpeedInc.Set)
	set("rangeBonus", wd.RangeBonus.V, wd.RangeBonus.Set)
	set("CritChance", wd.CritChance.V, wd.CritChance.Set)
	set("TotalDPS", wd.TotalDPS.V, wd.TotalDPS.Set)
	set("ElementalDPS", wd.ElementalDPS, wd.ElementalDPS != 0)
	for _, dmgType := range dmgTypeList {
		r := wd.Damage(dmgType)
		set(dmgType+"Min", r.Min, r.Min != 0)
		set(dmgType+"Max", r.Max, r.Max != 0)
		set(dmgType+"DPS", r.DPS, r.DPS != 0)
	}
	for k, v := range wd.Extra {
		out.Set(k, outValueOf(v))
	}
	return out
}

// CombineMode selects how combineStat folds a main-hand and an off-hand
// value into one (CalcOffence.lua L2003-2050). Every mode comes from a calc
// literal at a combineStat call; none is read from build data or config.
type CombineMode string

const (
	CombineOr CombineMode = "OR"
	// CombineAdd is the reference's second branch (L2007). No call site in
	// the reference passes it, so the branch is unreached; kept because the
	// port carries the reference's branch set.
	CombineAdd           CombineMode = "ADD"
	CombineAverage       CombineMode = "AVERAGE"
	CombineHarmonicMean  CombineMode = "HARMONICMEAN"
	CombineChance        CombineMode = "CHANCE"
	CombineChanceAilment CombineMode = "CHANCE_AILMENT"
	CombineDPS           CombineMode = "DPS"
)

// combineStat ports the local of the same name (L2003): fold the main-hand
// and off-hand pass results into the actor output. `extra` is the reference's
// vararg, used by the CHANCE modes.
func (env *Env) combineStat(c *offenceCtx, stat string, mode CombineMode, extra string) {
	output, skillFlags, skillData := c.output, c.skillFlags, c.skillData
	main, off := c.mainHandStats, c.offHandStats
	orElse := func() {
		if v := main.Get(stat); v.Truthy() {
			output.Set(stat, v)
		} else if v := off.Get(stat); v.Truthy() {
			output.Set(stat, v)
		} else {
			output.Del(stat)
		}
	}
	switch {
	case mode == CombineOr || !skillFlags["bothWeaponAttack"]:
		orElse()
	case mode == CombineAdd:
		output.SetN(stat, main.N(stat)+off.N(stat))
	case mode == CombineAverage:
		output.SetN(stat, (main.N(stat)+off.N(stat))/2)
	case mode == CombineHarmonicMean:
		if main.N(stat) == 0 || off.N(stat) == 0 {
			output.SetN(stat, 0.0)
		} else {
			output.SetN(stat, 2/(1/main.N(stat)+1/off.N(stat)))
		}
	case mode == CombineChance:
		if main.Flag(stat) && off.Flag(stat) {
			mainChance := main.N(extra) * main.N("HitChance")
			offChance := off.N(extra) * off.N("HitChance")
			if skillData.Flag("doubleHitsWhenDualWielding") {
				mainChance /= 10000
				offChance /= 10000
				output.SetN(stat, main.N(stat)*mainChance+off.N(stat)*offChance)
			} else {
				mainPortion := mainChance / (mainChance + offChance)
				offPortion := offChance / (mainChance + offChance)
				output.SetN(stat, main.N(stat)*mainPortion+off.N(stat)*offPortion)
			}
		} else {
			orElse()
		}
	case mode == CombineChanceAilment:
		if main.Flag(stat) && off.Flag(stat) {
			// The reference computes mainPortion/offPortion here and then
			// never uses them; only the stack split below matters.
			maxInstance := math.Max(main.N(stat), off.N(stat))
			minInstance := math.Min(main.N(stat), off.N(stat))
			stackName := strings.Replace(stat, "DPS", "", -1) + "Stacks"
			stacks, stacksMax := 1.0, 1.0
			if output.Flag(stackName) {
				stacks = output.N(stackName)
			}
			if output.Flag(stackName + "Max") {
				stacksMax = output.N(stackName + "Max")
			}
			maxInstanceStacks := math.Min(1, stacks/stacksMax)
			output.SetN(stat, maxInstance*maxInstanceStacks+minInstance*(1-maxInstanceStacks))
		} else {
			orElse()
		}
	case mode == CombineDPS:
		v := main.N(stat) + off.N(stat)
		if !skillData.Flag("doubleHitsWhenDualWielding") {
			v /= 2
		}
		output.SetN(stat, v)
	}
}
