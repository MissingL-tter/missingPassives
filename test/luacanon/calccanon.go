// Calc input/state records: the flat build.configInput table decoded into
// calc.ConfigInput and rendered back for the fixture echo, and buffList
// entries rendered for the skill-list checkpoint.
package luacanon

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

func init() {
	RegisterAdapter(func(v any) (any, bool) {
		switch t := v.(type) {
		case *calc.ConfigInput:
			return withExtras(t, ConfigInputTable(t)), true
		case *calc.SkillData:
			return SkillDataTable(t), true
		}
		return nil, false
	})
}

// SkillDataTable is the flat Lua skillData table: the three typed maps
// merged (a nil SkillData is an empty table).
func SkillDataTable(d *calc.SkillData) map[string]any {
	m := map[string]any{}
	if d == nil {
		return m
	}
	for k, v := range d.Nums {
		m[k] = v
	}
	for k, v := range d.Flags {
		m[k] = v
	}
	for k, v := range d.Strs {
		m[k] = v
	}
	return m
}

var damageTypes = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

// configNum is tonumber() over a config value: numbers pass, numeric text
// parses, anything else is absent.
func configNum(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		return util.Tonumber(t)
	}
	return 0, false
}

// configTruthy is Lua truthiness over a config value.
func configTruthy(v any) bool {
	b, isBool := v.(bool)
	return v != nil && (!isBool || b)
}

// strMode is the str reader for the config fields the calc gives a defined
// string type; the table carries the same bytes either way.
func strMode[T ~string](m map[string]any, used map[string]bool, key string, dst *T) {
	if v, ok := m[key].(string); ok {
		*dst = T(v)
		used[key] = true
	}
}

// ConfigInputFromTable decodes build.configInput (or, with placeholder set,
// build.configPlaceholder, whose enemy<Type>Overwhelm default the reference
// reads under the doubled key "enemy<Type>enemyOverwhelm"). Keys the calc
// never reads go to Extras so the echo re-emits them.
func ConfigInputFromTable(m map[string]any, placeholder bool) *calc.ConfigInput {
	c := &calc.ConfigInput{
		EnemyDamage:             map[string]float64{},
		EnemyPen:                map[string]float64{},
		EnemyOverwhelm:          map[string]float64{},
		EnemyResist:             map[string]float64{},
		BalanceOfTerrorSelfCast: map[string]bool{},
	}
	used := map[string]bool{}
	str := func(key string, dst *string) {
		if v, ok := m[key].(string); ok {
			*dst = v
			used[key] = true
		}
	}
	flag := func(key string, dst *bool) {
		if v, ok := m[key]; ok {
			*dst = configTruthy(v)
			used[key] = true
		}
	}
	opt := func(key string, dst *util.Opt[float64]) {
		if v, ok := configNum(m[key]); ok {
			*dst = util.Some(v)
			used[key] = true
		}
	}
	num := func(key string, dst *float64) {
		if v, ok := configNum(m[key]); ok {
			*dst = v
			used[key] = true
		}
	}
	perType := func(suffix string, dst map[string]float64) {
		for _, dt := range damageTypes {
			key := "enemy" + dt + suffix
			if v, ok := configNum(m[key]); ok {
				dst[dt] = v
				used[key] = true
			}
		}
	}
	str("bandit", &c.Bandit)
	str("pantheonMajorGod", &c.PantheonMajorGod)
	str("pantheonMinorGod", &c.PantheonMinorGod)
	strMode(m, used, "enemyDamageType", &c.EnemyDamageType)
	strMode(m, used, "ailmentMode", &c.AilmentMode)
	strMode(m, used, "repeatMode", &c.RepeatMode)
	strMode(m, used, "physMode", &c.PhysMode)
	str("ruthlessSupportMode", &c.RuthlessSupportMode)
	str("ChanceToIgnoreEnemyPhysicalDamageReductionMode", &c.ChanceToIgnoreEnemyPhysicalDamageReductionMode)
	str("doomBlastSource", &c.DoomBlastSource)
	flag("PvpScaling", &c.PvpScaling)
	flag("DisableEHPGainOnBlock", &c.DisableEHPGainOnBlock)
	flag("conditionLowLife", &c.ConditionLowLife)
	flag("excludeCullingDPS", &c.ExcludeCullingDPS)
	flag("EEIgnoreHitDamage", &c.EEIgnoreHitDamage)
	flag("ignoreJewelLimits", &c.IgnoreJewelLimits)
	flag("ignoreItemDisablers", &c.IgnoreItemDisablers)
	if v, ok := m["conditionLowEnergyShield"]; ok {
		c.ConditionLowEnergyShield = util.Some(configTruthy(v))
		used["conditionLowEnergyShield"] = true
	}
	opt("EHPUnluckyWorstOf", &c.EHPUnluckyWorstOf)
	opt("enemyCritChance", &c.EnemyCritChance)
	opt("enemyCritDamage", &c.EnemyCritDamage)
	opt("enemySpeed", &c.EnemySpeed)
	opt("enemyMultiplierPvpDamage", &c.EnemyMultiplierPvpDamage)
	opt("multiplierPvpTvalueOverride", &c.MultiplierPvpTvalueOverride)
	opt("resistancePenalty", &c.ResistancePenalty)
	opt("meleeDistance", &c.MeleeDistance)
	opt("projectileDistance", &c.ProjectileDistance)
	opt("overrideEmptyRedSockets", &c.OverrideEmptyRedSockets)
	opt("overrideEmptyGreenSockets", &c.OverrideEmptyGreenSockets)
	opt("overrideEmptyBlueSockets", &c.OverrideEmptyBlueSockets)
	opt("overrideEmptyWhiteSockets", &c.OverrideEmptyWhiteSockets)
	num("multiplierPoisonOnEnemy", &c.MultiplierPoisonOnEnemy)
	num("multiplierSummonedMinion", &c.MultiplierSummonedMinion)
	num("multiplierManaBurnStacks", &c.MultiplierManaBurnStacks)
	perType("Damage", c.EnemyDamage)
	perType("Pen", c.EnemyPen)
	perType("Resist", c.EnemyResist)
	if placeholder {
		perType("enemyOverwhelm", c.EnemyOverwhelm)
	} else {
		perType("Overwhelm", c.EnemyOverwhelm)
	}
	for k, v := range m {
		if curse, ok := strings.CutPrefix(k, "balanceOfTerrorSelfCast"); ok && configTruthy(v) {
			c.BalanceOfTerrorSelfCast[curse] = true
			used[k] = true
		}
	}
	extras := map[string]any{}
	for k, v := range m {
		if !used[k] {
			extras[k] = v
		}
	}
	if len(extras) > 0 {
		Extras[c] = extras
	}
	return c
}

// ConfigInputTable renders the decoded config back to its flat table. The
// placeholder's Overwhelm defaults are never present, so the doubled key is
// not re-emitted.
func ConfigInputTable(c *calc.ConfigInput) map[string]any {
	t := table{}
	t.str("bandit", c.Bandit)
	t.str("pantheonMajorGod", c.PantheonMajorGod)
	t.str("pantheonMinorGod", c.PantheonMinorGod)
	t.str("enemyDamageType", string(c.EnemyDamageType))
	t.str("ailmentMode", string(c.AilmentMode))
	t.str("repeatMode", string(c.RepeatMode))
	t.str("physMode", string(c.PhysMode))
	t.str("ruthlessSupportMode", c.RuthlessSupportMode)
	t.str("ChanceToIgnoreEnemyPhysicalDamageReductionMode", c.ChanceToIgnoreEnemyPhysicalDamageReductionMode)
	t.str("doomBlastSource", c.DoomBlastSource)
	t.flag("PvpScaling", c.PvpScaling)
	t.flag("DisableEHPGainOnBlock", c.DisableEHPGainOnBlock)
	t.flag("conditionLowLife", c.ConditionLowLife)
	t.flag("excludeCullingDPS", c.ExcludeCullingDPS)
	t.flag("EEIgnoreHitDamage", c.EEIgnoreHitDamage)
	t.flag("ignoreJewelLimits", c.IgnoreJewelLimits)
	t.flag("ignoreItemDisablers", c.IgnoreItemDisablers)
	t.optBool("conditionLowEnergyShield", c.ConditionLowEnergyShield)
	t.opt("EHPUnluckyWorstOf", c.EHPUnluckyWorstOf)
	t.opt("enemyCritChance", c.EnemyCritChance)
	t.opt("enemyCritDamage", c.EnemyCritDamage)
	t.opt("enemySpeed", c.EnemySpeed)
	t.opt("enemyMultiplierPvpDamage", c.EnemyMultiplierPvpDamage)
	t.opt("multiplierPvpTvalueOverride", c.MultiplierPvpTvalueOverride)
	t.opt("resistancePenalty", c.ResistancePenalty)
	t.opt("meleeDistance", c.MeleeDistance)
	t.opt("projectileDistance", c.ProjectileDistance)
	t.opt("overrideEmptyRedSockets", c.OverrideEmptyRedSockets)
	t.opt("overrideEmptyGreenSockets", c.OverrideEmptyGreenSockets)
	t.opt("overrideEmptyBlueSockets", c.OverrideEmptyBlueSockets)
	t.opt("overrideEmptyWhiteSockets", c.OverrideEmptyWhiteSockets)
	t.num("multiplierPoisonOnEnemy", c.MultiplierPoisonOnEnemy)
	t.num("multiplierSummonedMinion", c.MultiplierSummonedMinion)
	t.num("multiplierManaBurnStacks", c.MultiplierManaBurnStacks)
	for dt, v := range c.EnemyDamage {
		t["enemy"+dt+"Damage"] = v
	}
	for dt, v := range c.EnemyPen {
		t["enemy"+dt+"Pen"] = v
	}
	for dt, v := range c.EnemyOverwhelm {
		t["enemy"+dt+"Overwhelm"] = v
	}
	for dt, v := range c.EnemyResist {
		t["enemy"+dt+"Resist"] = v
	}
	for curse := range c.BalanceOfTerrorSelfCast {
		t["balanceOfTerrorSelfCast"+curse] = true
	}
	return t
}

// BuffTable renders one activeSkill.buffList entry's scalar keys (the
// dump's scalars() projection; the caller adds modList).
func BuffTable(b *calc.Buff) map[string]any {
	t := table{"type": b.Type, "name": b.Name}
	t.flag("activeSkillBuff", b.ActiveSkillBuff)
	t.optBool("applyNotPlayer", b.ApplyNotPlayer)
	t.optBool("applyMinions", b.ApplyMinions)
	t.optBool("applyAllies", b.ApplyAllies)
	t.optBool("allowTotemBuff", b.AllowTotemBuff)
	t.str("cond", b.Cond)
	t.str("stackVar", b.StackVar)
	t.opt("stackLimit", b.StackLimit)
	return t
}
