package config

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

// damageTypes is the order the per-type enemy keys are read in.
var damageTypes = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

// Input projects the tab's chosen values into the typed slice of them the
// calc reads directly. The options the calc does not read are absent by
// design: they reach it as modifiers instead.
func (t *Tab) ConfigInput() *calc.ConfigInput {
	return projectConfig(t.Input, false)
}

// Placeholders projects the standing-in values the same way. The calc
// consults them where an input is unset.
func (t *Tab) ConfigPlaceholder() *calc.ConfigInput {
	return projectConfig(t.Placeholder, true)
}

func projectConfig(src map[Var]Value, placeholder bool) *calc.ConfigInput {
	c := &calc.ConfigInput{
		EnemyDamage:             map[string]float64{},
		EnemyPen:                map[string]float64{},
		EnemyOverwhelm:          map[string]float64{},
		EnemyResist:             map[string]float64{},
		BalanceOfTerrorSelfCast: map[string]bool{},
	}
	str := func(key Var, dst *string) {
		if s, ok := StrOf(src[key]); ok {
			*dst = s
		}
	}
	flag := func(key Var, dst *bool) {
		if v, ok := src[key]; ok {
			*dst = Truthy(v)
		}
	}
	opt := func(key Var, dst *util.Opt[float64]) {
		if n, ok := NumOf(src[key]); ok {
			*dst = util.Some(n)
		}
	}
	num := func(key Var, dst *float64) {
		if n, ok := NumOf(src[key]); ok {
			*dst = n
		}
	}
	perType := func(suffix string, dst map[string]float64) {
		for _, dt := range damageTypes {
			if n, ok := NumOf(src[Var("enemy"+dt+suffix)]); ok {
				dst[dt] = n
			}
		}
	}
	str("bandit", &c.Bandit)
	str("pantheonMajorGod", &c.PantheonMajorGod)
	str("pantheonMinorGod", &c.PantheonMinorGod)
	if s, ok := StrOf(src["enemyDamageType"]); ok {
		c.EnemyDamageType = calc.DamageCategory(s)
	}
	if s, ok := StrOf(src["ailmentMode"]); ok {
		c.AilmentMode = calc.AilmentMode(s)
	}
	if s, ok := StrOf(src["repeatMode"]); ok {
		c.RepeatMode = calc.RepeatMode(s)
	}
	if s, ok := StrOf(src["physMode"]); ok {
		c.PhysMode = calc.PhysMode(s)
	}
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
	flag("conditionLowEnergyShield", &c.ConditionLowEnergyShield)
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
		// The placeholder's overwhelm keys carry the doubled name the
		// option table gives them.
		perType("enemyOverwhelm", c.EnemyOverwhelm)
	} else {
		perType("Overwhelm", c.EnemyOverwhelm)
	}
	for k, v := range src {
		if curse, ok := strings.CutPrefix(string(k), "balanceOfTerrorSelfCast"); ok && Truthy(v) {
			c.BalanceOfTerrorSelfCast[curse] = true
		}
	}
	return c
}
