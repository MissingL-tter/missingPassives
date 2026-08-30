// CalcOffence.lua L323-560: the offence entry, area of effect, the enemy
// resistance helper, and the early skill-data / stat-bonus updates.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// offenceCtx bundles the locals calcs.offence threads through its sections.
type offenceCtx struct {
	actor       *performActor
	activeSkill *ActiveSkill

	modDB   *modstore.DB
	enemyDB *modstore.DB
	output  modstore.Output

	skillModList *modstore.List
	skillData    *SkillData
	skillFlags   map[string]bool
	skillCfg     *modstore.Cfg

	// the offence-local state later sections read back
	isAttack         bool
	canDeal          map[string]bool
	conversionTbl    conversionTable
	conversionTables map[*modstore.Cfg]conversionTable
	passList         []*damagePass
	mainHandStats    modstore.Output
	offHandStats     modstore.Output

	// output.AreaOfEffectRadiusTertiaryOccurrences: a radius -> count
	// table, read back by Explosive Trap's preDamageFunc (not diffed)
	radiusTertiaryOccurrences map[float64]float64

	monsterLife        float64
	quantityMultiplier float64
	totalHitAvg        float64
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
	env.MirageHandled = env.RunMirages()
	if !env.MirageHandled {
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

	if c.skillData.Flag("showAverage") {
		c.skillFlags["showAverage"] = true
	} else {
		c.skillFlags["notAverage"] = true
	}

	if c.skillFlags["disable"] {
		// Skill is disabled
		c.output.SetN("CombinedDPS", 0.0)
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
	output.SetN("AreaOfEffectMod", util.RoundHalfUp(util.RoundHalfUp(incArea*moreArea, 10), 2))
	if c.skillData.Flag("radiusIsWeaponRange") {
		rng := 0.0
		if c.skillFlags["weapon1Attack"] {
			rng = math.Max(rng, c.actor.weaponRange1)
		}
		if c.skillFlags["weapon2Attack"] {
			rng = math.Max(rng, c.actor.weaponRange2)
		}
		c.skillData.SetN("radius", rng+2)
	}
	if c.skillData.Has("radius") {
		c.skillFlags["area"] = true
		baseRadius := c.skillData.N("radius") + c.skillData.N("radiusExtra") +
			skillModList.Sum(modparser.Base, skillCfg, "AreaOfEffect")
		output.SetN("AreaOfEffectRadius", calcRadius(baseRadius, output.N("AreaOfEffectMod")))
		output.SetN("AreaOfEffectRadiusMetres", output.N("AreaOfEffectRadius")/10)
		if c.skillData.Flag("radiusSecondary") {
			incAreaSecondary, moreAreaSecondary := Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectSecondary")
			output.SetN("AreaOfEffectModSecondary", util.RoundHalfUp(util.RoundHalfUp(incAreaSecondary*moreAreaSecondary, 10), 2))
			baseRadius = c.skillData.N("radiusSecondary") + c.skillData.N("radiusExtra")
			output.SetN("AreaOfEffectRadiusSecondary", calcRadius(baseRadius, output.N("AreaOfEffectModSecondary")))
			output.SetN("AreaOfEffectRadiusSecondaryMetres", output.N("AreaOfEffectRadiusSecondary")/10)
		}
		if c.skillData.Flag("radiusTertiary") {
			incAreaTertiary, moreAreaTertiary := Mods(skillModList, skillCfg, "AreaOfEffect", "AreaOfEffectTertiary")
			output.SetN("AreaOfEffectModTertiary", util.RoundHalfUp(util.RoundHalfUp(incAreaTertiary*moreAreaTertiary, 10), 2))
			baseRadius = c.skillData.N("radiusTertiary") + c.skillData.N("radiusExtra")
			if c.skillData.Flag("projectileSpeedAppliesToMSAreaOfEffect") {
				incSpeedTertiary, moreSpeedTertiary := Mods(skillModList, skillCfg, "ProjectileSpeed")
				output.SetN("SpeedModTertiary", util.RoundHalfUp(util.RoundHalfUp(incSpeedTertiary*moreSpeedTertiary, 10), 2))
				output.SetN("AreaOfEffectRadiusTertiary", calcMoltenStrikeTertiaryRadius(baseRadius,
					c.skillData.N("radiusSecondary"), output.N("AreaOfEffectModTertiary"), output.N("SpeedModTertiary")))
			} else if c.skillData.Flag("radiusTertiaryBaseMargin") {
				// "Smaller explosions have between 30% reduced and 30%
				// increased base radius at random" (Explosive Trap only).
				// Each 1% step of the deviation is one equally likely
				// outcome, so the reported radius is their mean -- but note
				// the loop runs one step past marginWidth outcomes, which
				// the divisor does not account for.
				margin := c.skillData.N("radiusTertiaryBaseMargin") / 100
				marginWidth := c.skillData.N("radiusTertiaryBaseMargin")*2 + 1
				baseRadiiOccurrences := map[float64]float64{}
				// Accumulating the step, as the Lua numeric for does.
				for deviation := 1 - margin; deviation <= 1+margin+0.01; deviation += 0.01 {
					baseRadiiOccurrences[math.Floor(baseRadius*deviation)]++
				}
				sumOfRandomRadii := 0.0
				radiiOccurrences := map[float64]float64{}
				for _, adjustedBaseRadius := range sortedNumKeys(baseRadiiOccurrences) {
					occurrenceCount := baseRadiiOccurrences[adjustedBaseRadius]
					radiusForDeviation := calcRadius(adjustedBaseRadius, output.N("AreaOfEffectModTertiary"))
					sumOfRandomRadii += radiusForDeviation * occurrenceCount
					radiiOccurrences[radiusForDeviation] += occurrenceCount
				}
				output.SetN("AreaOfEffectRadiusTertiary", sumOfRandomRadii/marginWidth)
				// Read back by Explosive Trap's preDamageFunc; scalarsOnly
				// keeps it out of the output canon, as in the reference dump.
				c.radiusTertiaryOccurrences = radiiOccurrences
			} else {
				output.SetN("AreaOfEffectRadiusTertiary", calcRadius(baseRadius, output.N("AreaOfEffectModTertiary")))
			}
			output.SetN("AreaOfEffectRadiusTertiaryMetres", output.N("AreaOfEffectRadiusTertiary")/10)
		}
	}
}

// calcResistForType ports the local of the same name (L461-474).
func (env *Env) calcResistForType(c *offenceCtx, damageType string, cfg *modstore.Cfg) float64 {
	enemyDB := c.enemyDB

	var resist float64
	haveResist := false
	if ov, ok := enemyDB.Override(cfg, damageType+"Resist"); ok {
		resist = valueNum(ov)
		haveResist = true
	}
	maxResist := data.Misc.EnemyMaxResist
	if !enemyDB.Flag(nil, "DoNotChangeMaxResFromConfig") {
		configured := data.Misc.EnemyMaxResist
		if v, ok := env.ConfigInput.EnemyResist[damageType]; ok {
			configured = v
		}
		maxResist = math.Min(math.Max(configured, data.Misc.EnemyMaxResist), data.Misc.MaxResistCap)
	}
	if !haveResist {
		if env.ModDB.Flag(nil, "Enemy"+damageType+"ResistEqualToYours") {
			resist = env.Player.Output.N(damageType + "Resist")
		} else {
			names := elemNames(damageType, damageType+"Resist", "ElementalResist")
			resist = enemyDB.Sum(modparser.Base, cfg, names...) * math.Max(Mod(enemyDB, cfg, names...), 0)
		}
	}
	return math.Max(math.Min(resist, maxResist), data.Misc.ResistFloor)
}

// runSkillFunc ports the local of the same name: the granted effect's
// hand-written Lua callbacks. Any that a corpus build reaches must be
// ported into Go before it can be exact.
func (env *Env) runSkillFunc(c *offenceCtx, kind data.CallbackKind) {
	ge := c.activeSkill.ActiveEffect.GrantedEffect
	if !ge.Custom.Callbacks[kind] {
		return
	}
	if ported, ok := skillFuncs[skillFuncKey{ge.Id, kind}]; ok {
		ported(env, c)
		return
	}
	panic("offence: granted effect " + ge.Id + " has an unported " + kind.String() + " callback")
}

// offencePrologue covers L483-560: the initial skill func, the triggered /
// focused conditions, the SkillData merges and the attribute bonuses.
func (env *Env) offencePrologue(c *offenceCtx) {
	skillModList, skillCfg, skillData := c.skillModList, c.skillCfg, c.skillData

	// NOTE: calcAreaOfEffect is defined early in the reference but first
	// called at L1152, in the skill-type-stats section, after weaponRange
	// is set. Do not hoist it here.
	env.runSkillFunc(c, data.CallbackInitial)

	if skillCfg.SkillCond == nil {
		skillCfg.SkillCond = map[string]bool{}
	}
	skillCfg.SkillCond["SkillIsTriggered"] = skillData.Flag("triggered")
	if skillCfg.SkillCond["SkillIsTriggered"] {
		c.skillFlags["triggered"] = true
	}
	skillCfg.SkillCond["SkillIsFocused"] = skillData.Flag("chanceToTriggerOnFocus")
	if skillCfg.SkillCond["SkillIsFocused"] {
		c.skillFlags["focused"] = true
	}

	// Update skill data
	for _, v := range skillModList.List(skillCfg, "SkillData") {
		tag, _ := v.(modparser.DataRef)
		key := tag.Key
		if tag.Merge == "MAX" {
			skillData.SetN(key, math.Max(valueNum(tag.Value), skillData.N(key)))
		} else {
			skillData.Set(key, outValueOf(tag.Value))
		}
	}

	// Add addition stat bonuses
	if skillModList.Flag(nil, "IronGrip") {
		skillModList.AddMod(newModSF("PhysicalDamage", modparser.Inc, modparser.Num(c.actor.strDmgBonus), "Strength", modparser.FlagAttack|modparser.FlagProjectile, modparser.KeywordNone))
	}
	if skillModList.Flag(nil, "IronWill") {
		skillModList.AddMod(newModSF("Damage", modparser.Inc, modparser.Num(c.actor.strDmgBonus), "Strength", modparser.FlagSpell, modparser.KeywordNone))
	}
	if skillModList.Flag(nil, "TransfigurationOfBody") {
		skillModList.AddMod(newModSF("Damage", modparser.Inc, modparser.Num(math.Floor(skillModList.Sum(modparser.Inc, nil, "Life")*data.Misc.Transfiguration)), "Transfiguration of Body", modparser.FlagAttack, modparser.KeywordNone))
	}
	if skillModList.Flag(nil, "TransfigurationOfMind") {
		skillModList.AddMod(newModS("Damage", modparser.Inc, modparser.Num(math.Floor(skillModList.Sum(modparser.Inc, nil, "Mana")*data.Misc.Transfiguration)), "Transfiguration of Mind"))
	}
	if skillModList.Flag(nil, "TransfigurationOfSoul") {
		skillModList.AddMod(newModSF("Damage", modparser.Inc, modparser.Num(math.Floor(skillModList.Sum(modparser.Inc, nil, "EnergyShield")*data.Misc.Transfiguration)), "Transfiguration of Soul", modparser.FlagSpell, modparser.KeywordNone))
	}

	env.offenceSkillData(c)

	c.isAttack = c.skillFlags["attack"]

	env.runSkillFunc(c, data.CallbackPreSkillType)

	env.offenceSkillTypeStats(c)
}
