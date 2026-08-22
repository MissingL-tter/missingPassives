// CalcOffence.lua L323-560: the offence entry, area of effect, the enemy
// resistance helper, and the early skill-data / stat-bonus updates.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// offenceCtx bundles the locals calcs.offence threads through its sections.
type offenceCtx struct {
	actor       *performActor
	activeSkill *ActiveSkill

	modDB   *modstore.DB
	enemyDB *modstore.DB
	output  map[string]any

	skillModList *modstore.List
	skillData    map[string]any
	skillFlags   map[string]bool
	skillCfg     *modstore.Cfg

	// weapon type info for the wielded weapons, nil when the slot holds
	// nothing the table knows (the reference's weapon1info/weapon2info)
	weapon1info, weapon2info *data.WeaponTypeInfo

	// the offence-local state later sections read back
	isAttack         bool
	canDeal          map[string]bool
	conversionTbl    conversionTable
	conversionTables map[*modstore.Cfg]conversionTable
	passList         []*damagePass
	mainHandStats    map[string]any
	offHandStats     map[string]any

	monsterLife        float64
	quantityMultiplier float64
	hitRate            float64
	totalHitAvg        float64
	totalCritMin       float64
	totalCritMax       float64
	totalCritAvg       float64
	costs              map[string]*costEntry
}

// The trigger/mirage/offence handoff, split so a caller can interleave the
// archive checkpoints exactly where the reference emits them
// (CalcPerform L3726-3729).

// RunTriggersPlayer runs calcs.triggers for the player actor.
func (env *Env) RunTriggersPlayer() { env.RunTriggers(env.playerPA) }

// RunOffencePlayer runs calcs.offence for the player's main skill, unless
// calcs.mirages took the calculation over.
func (env *Env) RunOffencePlayer() {
	if !env.RunMirages() {
		env.offence(env.playerPA, env.PlayerMainSkill)
	}
}

// RunTriggersMinion runs calcs.triggers for the minion actor.
func (env *Env) RunTriggersMinion() { env.RunTriggers(env.minionPA) }

// RunOffenceMinion runs calcs.offence for the minion's main skill.
func (env *Env) RunOffenceMinion() { env.offence(env.minionPA, env.Minion.MainSkill) }

// RunOffence runs the whole handoff in the reference's order.
func (env *Env) RunOffence() {
	env.RunTriggersPlayer()
	env.RunOffencePlayer()
	if env.Minion != nil {
		env.RunTriggersMinion()
		env.RunOffenceMinion()
	}
}

func (env *Env) offence(actor *performActor, activeSkill *ActiveSkill) {
	c := &offenceCtx{
		actor:        actor,
		activeSkill:  activeSkill,
		modDB:        actor.db,
		enemyDB:      actor.enemy.db,
		output:       actor.output,
		skillModList: activeSkill.SkillModList,
		skillData:    activeSkill.SkillData,
		skillFlags:   activeSkill.SkillFlags,
		skillCfg:     activeSkill.SkillCfg,
	}

	if truthy(c.skillData["showAverage"]) {
		c.skillFlags["showAverage"] = true
	} else {
		c.skillFlags["notAverage"] = true
	}

	if c.skillFlags["disable"] {
		// Skill is disabled
		c.output["CombinedDPS"] = 0.0
		return
	}

	env.offencePrologue(c)
}

// calcAreaOfEffect ports the local of the same name (L345-459).
func (env *Env) calcAreaOfEffect(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillFlags
	_ = skillData
	output := c.output

	incArea, moreArea := Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectPrimary")
	output["AreaOfEffectMod"] = roundDec(roundDec(incArea*moreArea, 10), 2)
	if truthy(c.skillData["radiusIsWeaponRange"]) {
		rng := 0.0
		if c.skillFlags["weapon1Attack"] {
			rng = math.Max(rng, c.actor.weaponRange1)
		}
		if c.skillFlags["weapon2Attack"] {
			rng = math.Max(rng, c.actor.weaponRange2)
		}
		c.skillData["radius"] = rng + 2
	}
	if truthy(c.skillData["radius"]) {
		c.skillFlags["area"] = true
		baseRadius := anyNum(c.skillData["radius"]) + anyNum(c.skillData["radiusExtra"]) +
			skillModList.Sum("BASE", skillCfg, "AreaOfEffect")
		output["AreaOfEffectRadius"] = calcRadius(baseRadius, outNum(output, "AreaOfEffectMod"))
		output["AreaOfEffectRadiusMetres"] = outNum(output, "AreaOfEffectRadius") / 10
		if truthy(c.skillData["radiusSecondary"]) {
			incAreaSecondary, moreAreaSecondary := Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
			output["AreaOfEffectModSecondary"] = roundDec(roundDec(incAreaSecondary*moreAreaSecondary, 10), 2)
			baseRadius = anyNum(c.skillData["radiusSecondary"]) + anyNum(c.skillData["radiusExtra"])
			output["AreaOfEffectRadiusSecondary"] = calcRadius(baseRadius, outNum(output, "AreaOfEffectModSecondary"))
			output["AreaOfEffectRadiusSecondaryMetres"] = outNum(output, "AreaOfEffectRadiusSecondary") / 10
		}
		if truthy(c.skillData["radiusTertiary"]) {
			incAreaTertiary, moreAreaTertiary := Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
			output["AreaOfEffectModTertiary"] = roundDec(roundDec(incAreaTertiary*moreAreaTertiary, 10), 2)
			baseRadius = anyNum(c.skillData["radiusTertiary"]) + anyNum(c.skillData["radiusExtra"])
			if truthy(c.skillData["projectileSpeedAppliesToMSAreaOfEffect"]) {
				incSpeedTertiary, moreSpeedTertiary := Mods(skillModList, skillCfg, "ProjectileSpeed")
				output["SpeedModTertiary"] = roundDec(roundDec(incSpeedTertiary*moreSpeedTertiary, 10), 2)
				output["AreaOfEffectRadiusTertiary"] = calcMoltenStrikeTertiaryRadius(baseRadius,
					anyNum(c.skillData["radiusSecondary"]), outNum(output, "AreaOfEffectModTertiary"), outNum(output, "SpeedModTertiary"))
			} else if truthy(c.skillData["radiusTertiaryBaseMargin"]) {
				panic("offence: radiusTertiaryBaseMargin (Explosive Trap random radius) unported")
			} else {
				output["AreaOfEffectRadiusTertiary"] = calcRadius(baseRadius, outNum(output, "AreaOfEffectModTertiary"))
			}
			output["AreaOfEffectRadiusTertiaryMetres"] = outNum(output, "AreaOfEffectRadiusTertiary") / 10
		}
	}
}

// calcResistForType ports the local of the same name (L461-474).
func (env *Env) calcResistForType(c *offenceCtx, damageType string, cfg *modstore.Cfg) float64 {
	d := env.Data
	enemyDB := c.enemyDB

	var resist float64
	haveResist := false
	if ov := enemyDB.Override(cfg, damageType+"Resist"); truthy(ov) {
		resist = anyNum(ov)
		haveResist = true
	}
	maxResist := d.Misc.EnemyMaxResist
	if !enemyDB.Flag(nil, "DoNotChangeMaxResFromConfig") {
		configured := d.Misc.EnemyMaxResist
		if v := env.ConfigInput["enemy"+damageType+"Resist"]; truthy(v) {
			configured = anyNum(v)
		}
		maxResist = math.Min(math.Max(configured, d.Misc.EnemyMaxResist), d.Misc.MaxResistCap)
	}
	if !haveResist {
		if env.ModDB.Flag(nil, "Enemy"+damageType+"ResistEqualToYours") {
			resist = outNum(env.Player.Output, damageType+"Resist")
		} else {
			names := elemNames(damageType, damageType+"Resist", "ElementalResist")
			resist = enemyDB.Sum("BASE", cfg, names...) * math.Max(Mod(enemyDB, cfg, names...), 0)
		}
	}
	return math.Max(math.Min(resist, maxResist), d.Misc.ResistFloor)
}

// runSkillFunc ports the local of the same name: the granted effect's
// hand-written Lua callbacks. Any that a corpus build reaches must be
// ported into Go before it can be exact.
func (env *Env) runSkillFunc(c *offenceCtx, name string) {
	fn, ok := c.activeSkill.ActiveEffect.GrantedEffect.Custom[name]
	if !ok || fn == nil {
		return
	}
	id := c.activeSkill.ActiveEffect.GrantedEffect.Id
	if ported, ok := skillFuncs[id+":"+name]; ok {
		ported(env, c)
		return
	}
	if _, unported := fn.(data.UnportedFn); unported {
		panic("offence: granted effect " + id + " has an unported " + name + " callback")
	}
	panic("offence: unexpected " + name + " callback shape")
}

// offencePrologue covers L483-560: the initial skill func, the triggered /
// focused conditions, the SkillData merges and the attribute bonuses.
func (env *Env) offencePrologue(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData
	d := env.Data

	// NOTE: calcAreaOfEffect is defined early in the reference but first
	// called at L1152, in the skill-type-stats section, after weaponRange
	// is set. Do not hoist it here.
	env.runSkillFunc(c, "initialFunc")

	if skillCfg.SkillCond == nil {
		skillCfg.SkillCond = map[string]bool{}
	}
	skillCfg.SkillCond["SkillIsTriggered"] = truthy(skillData["triggered"])
	if skillCfg.SkillCond["SkillIsTriggered"] {
		c.skillFlags["triggered"] = true
	}
	skillCfg.SkillCond["SkillIsFocused"] = truthy(skillData["chanceToTriggerOnFocus"])
	if skillCfg.SkillCond["SkillIsFocused"] {
		c.skillFlags["focused"] = true
	}

	// Update skill data
	for _, v := range skillModList.List(skillCfg, "SkillData") {
		tag, _ := v.(modparser.Tag)
		key := str(tag["key"])
		if str(tag["merge"]) == "MAX" {
			skillData[key] = math.Max(anyNum(tag["value"]), anyNum(skillData[key]))
		} else {
			skillData[key] = tag["value"]
		}
	}

	// Add addition stat bonuses
	if skillModList.Flag(nil, "IronGrip") {
		skillModList.AddMod(newMod("PhysicalDamage", "INC", c.actor.strDmgBonus, "Strength",
			modparser.ModFlag.Attack|modparser.ModFlag.Projectile))
	}
	if skillModList.Flag(nil, "IronWill") {
		skillModList.AddMod(newMod("Damage", "INC", c.actor.strDmgBonus, "Strength", modparser.ModFlag.Spell))
	}
	if skillModList.Flag(nil, "TransfigurationOfBody") {
		skillModList.AddMod(newMod("Damage", "INC",
			math.Floor(skillModList.Sum("INC", nil, "Life")*d.Misc.Transfiguration),
			"Transfiguration of Body", modparser.ModFlag.Attack))
	}
	if skillModList.Flag(nil, "TransfigurationOfMind") {
		skillModList.AddMod(newMod("Damage", "INC",
			math.Floor(skillModList.Sum("INC", nil, "Mana")*d.Misc.Transfiguration),
			"Transfiguration of Mind"))
	}
	if skillModList.Flag(nil, "TransfigurationOfSoul") {
		skillModList.AddMod(newMod("Damage", "INC",
			math.Floor(skillModList.Sum("INC", nil, "EnergyShield")*d.Misc.Transfiguration),
			"Transfiguration of Soul", modparser.ModFlag.Spell))
	}

	env.offenceSkillData(c)

	c.isAttack = c.skillFlags["attack"]

	env.runSkillFunc(c, "preSkillTypeFunc")

	env.offenceSkillTypeStats(c)
}
