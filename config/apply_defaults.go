package config

import (
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// The options in this file are the ones every build carries, because each
// declares a default state or a default placeholder: their apply functions
// run whether or not the saved file mentions them.

// cond is the { type = "Condition", var = ... } tag.
func cond(v string) modparser.Tag { return &modparser.CondTag{Var: v} }

// debuff is the { type = "GlobalEffect", effectType = "Debuff" } tag the
// stacking-debuff options attach.
func debuff() modparser.Tag { return &modparser.GlobalEffectTag{EffectType: "Debuff"} }

// skillName matches an active skill by name.
func skillName(name string) modparser.Tag {
	return &modparser.SkillNameTag{SkillName: name}
}

// stackSource is the reference's `val.." <name> Stacks"` modifier source:
// the stack count is part of the source string.
func stackSource(val float64, name string) string {
	return util.FormatG14(val) + " " + name + " Stacks"
}

// ConfigOptions.lua L219: resource gain calculation mode.
func applyResourceGainMode(v Value, t *Tab) {
	switch s, _ := StrOf(v); s {
	case "AVERAGE":
		t.mod("Condition:AverageResourceGain", modparser.Flag, modparser.Bool(true))
	case "MAX":
		t.mod("Condition:MaxResourceGain", modparser.Flag, modparser.Bool(true))
	}
}

// ConfigOptions.lua L311 and L366: two stage counters that hard-cap at ten
// stages, counted from the first, so the modifier carries val-1 capped at 9.
func applyBloodsoakedBannerStages(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("Multiplier:BloodsoakedBannerStageAfterFirst", modparser.Base, modparser.Num(math.Min(n-1, 9)), cond("Effective"))
}

func applyCorruptingCryStages(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("Multiplier:CorruptingCryStageAfterFirst", modparser.Base, modparser.Num(math.Min(n-1, 9)), cond("Effective"))
}

// ConfigOptions.lua L327.
func applyTargetBrandedEnemy(_ Value, t *Tab) {
	t.mod("Condition:TargetingBrandedEnemy", modparser.Flag, modparser.Bool(true))
}

// ConfigOptions.lua L362, L448, L725: three skills whose cooldown the
// build is assumed to bypass.
func applyConcPathBypassCD(_ Value, t *Tab) {
	t.mod("CooldownRecovery", modparser.Override, modparser.Num(0), skillName("Consecrated Path of Endurance"))
}

func applyFlickerStrikeBypassCD(_ Value, t *Tab) {
	t.mod("CooldownRecovery", modparser.Override, modparser.Num(0),
		&modparser.SkillNameTag{SkillName: "Flicker Strike", IncludeTransfigured: true})
}

func applyVigilantStrikeBypassCD(_ Value, t *Tab) {
	t.mod("CooldownRecovery", modparser.Override, modparser.Num(0), skillName("Vigilant Strike"))
}

// ConfigOptions.lua L427: Glorious Madness' touch stacks, capped at ten.
func applyTouchedDebuffsCount(v Value, t *Tab) {
	n, _ := NumOf(v)
	n = math.Min(n, 10)
	affected := cond("AffectedByGloriousMadness")
	add := func(name string, value float64, stack string) {
		t.Mods.AddMod(modparser.NewModFull(name, modparser.Inc, modparser.Num(value),
			stackSource(n, stack), true, modparser.FlagNone, modparser.KeywordNone, debuff(), affected))
	}
	add("DamageTaken", n*6, "Eroding Touch")
	add("ActionSpeed", -n*6, "Paralysing Touch")
	add("FlaskChargesGained", -n*9, "Diluting Touch")
	add("FlaskEffect", -n*9, "Diluting Touch")
	add("LifeRecoveryRate", -n*9, "Wasting Touch")
	add("EnergyShieldRecoveryRate", -n*9, "Wasting Touch")
}

// ConfigOptions.lua L548: Plague Bearer's two states.
func applyPlagueBearerState(v Value, t *Tab) {
	switch s, _ := StrOf(v); s {
	case "INC":
		t.mod("Condition:PlagueBearerIncubating", modparser.Flag, modparser.Bool(true))
	case "INF":
		t.mod("Condition:PlagueBearerInfecting", modparser.Flag, modparser.Bool(true))
	}
}

// spectreSkill is the Raise Spectre summon-skill tag the spectre options
// attach to the skills they enable.
func spectreSkill() modparser.Tag {
	return &modparser.SkillNameTag{SkillName: "Raise Spectre", IncludeTransfigured: true, SummonSkill: true}
}

// enableSkillData is the { key = "enable", value = true } SkillData record
// the "let this summon use its X" options grant.
func enableSkillData() modparser.Value {
	return modparser.DataRef{Key: "enable", Value: modparser.Bool(true)}
}

// ConfigOptions.lua L578 and L581: which of a spectre's skills to enable.
func applyRaiseSpectreEnableBuffs(_ Value, t *Tab) {
	t.mod("SkillData", modparser.List, enableSkillData(),
		&modparser.SkillTypeTag{SkillType: modparser.SkillTypeBuff}, spectreSkill())
}

func applyRaiseSpectreEnableCurses(_ Value, t *Tab) {
	t.mod("SkillData", modparser.List, enableSkillData(),
		&modparser.SkillTypeTag{SkillType: modparser.SkillTypeHex}, spectreSkill())
	t.mod("SkillData", modparser.List, enableSkillData(),
		&modparser.SkillTypeTag{SkillType: modparser.SkillTypeMark}, spectreSkill())
}

// ConfigOptions.lua L672-678: the Elemental Relic's three auras.
func applyRelicAura(auraID string) func(Value, *Tab) {
	return func(_ Value, t *Tab) {
		t.mod("SkillData", modparser.List, enableSkillData(),
			&modparser.SkillIDTag{SkillID: auraID},
			&modparser.SkillNameTag{SkillName: "Summon Elemental Relic", SummonSkill: true})
	}
}

// ConfigOptions.lua L625.
func applySigilOfPowerStages(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("Multiplier:SigilOfPowerStage", modparser.Base, modparser.Num(n))
}

// ConfigOptions.lua L638: the configured tiger count, and the live count
// it feeds through the skill's own limit.
func applySpectralTigerCount(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("Multiplier:SpectralTigerConfig", modparser.Base, modparser.Num(n))
	t.mod("Multiplier:SpectralTigerCount", modparser.Base, modparser.Num(1),
		&modparser.MultiplierTag{Var: "SpectralTigerConfig", LimitStat: "ActiveTigerLimit"})
}

// ConfigOptions.lua L771: the enemy hitbox radius the shotgunning
// calculations use. The placeholder write is not a notify, so it only
// moves the widget - the modifier is what carries the value.
func applyEnemySizePreset(v Value, t *Tab) {
	radius := map[string]float64{"Small": 2, "Medium": 3, "Large": 5, "Huge": 11}
	s, _ := StrOf(v)
	n, ok := radius[s]
	if !ok {
		return
	}
	t.mod("EnemyRadius", modparser.Base, modparser.Num(n))
}

// minionMod is the { mod = ... } MinionModifier record.
func minionMod(m *modparser.Mod) modparser.Value { return modparser.ModRef{Mod: m} }

// ConfigOptions.lua L945: how a repeating skill's repeats are counted.
func applyRepeatMode(v Value, t *Tab) {
	var condVar string
	switch s, _ := StrOf(v); s {
	case "AVERAGE":
		condVar = "Condition:averageRepeat"
	case "FINAL", "FINAL_DPS":
		condVar = "Condition:alwaysFinalRepeat"
	default:
		return
	}
	t.mod(condVar, modparser.Flag, modparser.Bool(true))
	// The minion copy carries no source of its own: the reference's
	// NewMod call omits it.
	t.Mods.AddMod(modparser.NewMod("MinionModifier", modparser.List,
		minionMod(modparser.NewModFull(condVar, modparser.Flag, modparser.Bool(true), "Config", true, modparser.FlagNone, modparser.KeywordNone))))
}

// ConfigOptions.lua L1225.
func applyWitheredStackCountSelf(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("Multiplier:WitheredStack", modparser.Base, modparser.Num(n), cond("Effective"))
}

// ConfigOptions.lua L1480.
func applyChampionIntimidate(_ Value, t *Tab) {
	t.enemyMod("Condition:ChampionIntimidate", modparser.Flag, modparser.Bool(true), cond("Combat"))
}

// ConfigOptions.lua L1560.
func applySacrificeMinion(_ Value, t *Tab) {
	t.mod("Condition:SacrificeMinionOnAttack", modparser.Flag, modparser.Bool(true), cond("Combat"))
}

// ConfigOptions.lua L1678.
func applyGamblesprintMovementSpeed(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("MovementSpeed", modparser.Inc, modparser.Num(n), cond("Combat"), cond("HaveGamblesprint"))
}

// ConfigOptions.lua L1871, L1930, L1970: enemy ailment and flame stacks.
func applyEnemyMultiplier(name string) func(Value, *Tab) {
	return func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod("Multiplier:"+name, modparser.Base, modparser.Num(n), cond("Effective"))
	}
}

// ConfigOptions.lua L1973.
func applyHallowingFlameRemovedByAlly(v Value, t *Tab) {
	n, _ := NumOf(v)
	t.mod("Multiplier:HallowingFlameStacksRemovedByAlly", modparser.Base, modparser.Num(n))
}

// ConfigOptions.lua L2052: Mania stacks, capped at fifteen.
func applyManiaDebuffsCount(v Value, t *Tab) {
	n, _ := NumOf(v)
	n = math.Min(n, 15)
	afflicted := cond("AfflictedByMania")
	add := func(name string, value float64) {
		t.EnemyMods.AddMod(modparser.NewModFull(name, modparser.Inc, modparser.Num(value),
			stackSource(n, "Mania"), true, modparser.FlagNone, modparser.KeywordNone, debuff(), afflicted))
	}
	add("DamageTaken", n*4)
	add("ActionSpeed", -n*2)
	add("LifeRecoveryRate", -n*10)
	add("EnergyShieldRecoveryRate", -n*10)
}

// ConfigOptions.lua L2227-2248: the enemy's own defensive stats. The
// resistances carry an "EnemyConfig" source, the rest "Config".
func applyEnemyResist(name string) func(Value, *Tab) {
	return func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.EnemyMods.AddMod(modparser.NewModFull(name, modparser.Base, modparser.Num(n),
			"EnemyConfig", true, modparser.FlagNone, modparser.KeywordNone))
	}
}

func applyEnemyDefence(name string) func(Value, *Tab) {
	return func(v Value, t *Tab) {
		n, _ := NumOf(v)
		t.enemyMod(name, modparser.Base, modparser.Num(n))
	}
}
