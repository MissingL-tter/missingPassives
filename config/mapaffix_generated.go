package config

// Converted once from Data/ModMap.lua: the map-modifier appliers the eight
// map affix dropdowns dispatch into. The reference keeps them beside the
// affix data; here they live with the tab that calls them, and the data
// package's MapMod.Apply stays the marker saying one exists.

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// val, v, v1 and v2 read the affix's value table at the selected tier: a
// plain number, one of a pair, or one of a nested pair.
func (a mapAffixArgs) val() data.MapModValue {
	if a.Tier < 1 || a.Tier > len(a.Values) {
		return data.MapModValue{}
	}
	return a.Values[a.Tier-1]
}

func (a mapAffixArgs) v() float64 { return a.val().Num }

func (a mapAffixArgs) v1(i int) float64 {
	l := a.val().List
	if i < 1 || i > len(l) {
		return 0
	}
	return l[i-1].Num
}

func (a mapAffixArgs) v2(i, j int) float64 {
	l := a.val().List
	if i < 1 || i > len(l) {
		return 0
	}
	l = l[i-1].List
	if j < 1 || j > len(l) {
		return 0
	}
	return l[j-1].Num
}

var mapAffixApplies = map[string]func(a mapAffixArgs, t *Tab){
	// Data/ModMap.lua L12
	"Armoured": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("PhysicalDamageReduction", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Armoured", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L20
	"Hexproof": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("Hexproof", modparser.Flag, modparser.Bool(true), "Map mod Hexproof", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L29
	"Hexwarded": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("CurseEffectOnSelf", modparser.More, modparser.Num(-a.v()*a.Effect), "Map mod Hexwarded", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L38
	"Resistant": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("ElementalResist", modparser.Base, modparser.Num(a.v1(1)*a.Effect), "Map mod Resistant", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("ChaosResist", modparser.Base, modparser.Num(a.v1(2)*a.Effect), "Map mod Resistant", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L47
	"Unwavering": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("AvoidStun", modparser.Base, modparser.Num(100), "Map mod Unwavering", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("Life", modparser.More, modparser.Num((a.v2(1, 1)+(a.v2(1, 2)-a.v2(1, 1))*a.RollRange/100)*a.Effect), "Map mod Unwavering", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L56
	"Fecund": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("Life", modparser.More, modparser.Num((a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod Fecund", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L64
	"Unstoppable": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("MinimumActionSpeed", modparser.Max, modparser.Num(100), "Map mod Unstoppable", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L73
	"Impervious": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("AvoidPoison", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Impervious", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("AvoidImpale", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Impervious", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("AvoidBleed", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Impervious", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L83
	"Oppressive": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("SpellSuppressionChance", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Oppressive", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L91
	"Buffered": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("LifeGainAsEnergyShield", modparser.Base, modparser.Num((a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod Buffered", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L99
	"Titan's": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("Life", modparser.More, modparser.Num(a.v1(1)*a.Effect), "Map mod Titan's", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "RareOrUnique"}))
		t.EnemyMods.AddMod(modparser.NewModFull("AreaOfEffect", modparser.Inc, modparser.Num(a.v1(2)*a.Effect), "Map mod Titan's", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "RareOrUnique"}))
	},
	// Data/ModMap.lua L109
	"Savage": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("Damage", modparser.Inc, modparser.Num((a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod Savage", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L117
	"Burning": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("PhysicalDamageGainAsFire", modparser.Base, modparser.Num((a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod Burning", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L125
	"Freezing": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("PhysicalDamageGainAsCold", modparser.Base, modparser.Num((a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod Freezing", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L133
	"Shocking": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("PhysicalDamageGainAsLightning", modparser.Base, modparser.Num((a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod Shocking", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L141
	"Profane": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("PhysicalDamageGainAsChaos", modparser.Base, modparser.Num((a.v2(1, 1)+(a.v2(1, 2)-a.v2(1, 1))*a.RollRange/100)*a.Effect), "Map mod Profane", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("Condition:CanBeWithered", modparser.Flag, modparser.Bool(true), "Map mod Profane", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L150
	"Fleet": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("Speed", modparser.Inc, modparser.Num((a.v2(2, 1)+(a.v2(2, 2)-a.v2(2, 1))*a.RollRange/100)*a.Effect), "Map mod Fleet", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L158
	"Conflagrating": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("IgniteChance", modparser.Base, modparser.Num(100), "Map mod Conflagrating", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("AllDamageIgnites", modparser.Flag, modparser.Bool(true), "Map mod Conflagrating", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L167
	"Impaling": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("ImpaleChance", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Impaling", true, modparser.FlagAttack, modparser.KeywordNone))
	},
	// Data/ModMap.lua L175
	"Empowered": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("ElementalAilmentChance", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod Empowered", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L183
	"Overlord's": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("Damage", modparser.Inc, modparser.Num(a.v1(1)*a.Effect), "Map mod Overlord's", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "RareOrUnique"}))
		t.EnemyMods.AddMod(modparser.NewModFull("Speed", modparser.Inc, modparser.Num(a.v1(2)*a.Effect), "Map mod Overlord's", true, modparser.FlagNone, modparser.KeywordNone, &modparser.CondTag{Var: "RareOrUnique"}))
	},
	// Data/ModMap.lua L208
	"of Balance": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("Keystone", modparser.List, modparser.Str("Elemental Equilibrium"), "Map mod of Balance", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L217
	"of Congealment": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("CannotLeechLifeFromSelf", modparser.Flag, modparser.Bool(true), "Map mod of Congealment", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("CannotLeechManaFromSelf", modparser.Flag, modparser.Bool(true), "Map mod of Congealment", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("CannotLeechEnergyShieldFromSelf", modparser.Flag, modparser.Bool(true), "Map mod of Congealment", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L228
	"of Drought": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("FlaskChargesGained", modparser.Inc, modparser.Num(-a.v()*a.Effect), "Map mod of Drought", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L238
	"of Exposure": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("FireResistMax", modparser.Base, modparser.Num(-((a.v1(1) + (a.v1(2)-a.v1(1))*a.RollRange/100) * a.Effect)), "Map mod of Exposure", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("ColdResistMax", modparser.Base, modparser.Num(-((a.v1(1) + (a.v1(2)-a.v1(1))*a.RollRange/100) * a.Effect)), "Map mod of Exposure", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("LightningResistMax", modparser.Base, modparser.Num(-((a.v1(1) + (a.v1(2)-a.v1(1))*a.RollRange/100) * a.Effect)), "Map mod of Exposure", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("ChaosResistMax", modparser.Base, modparser.Num(-((a.v1(1) + (a.v1(2)-a.v1(1))*a.RollRange/100) * a.Effect)), "Map mod of Exposure", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L253
	"of Impotence": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("AreaOfEffect", modparser.More, modparser.Num(-a.v()*a.Effect), "Map mod of Impotence", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L262
	"of Insulation": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("AvoidElementalAilments", modparser.Base, modparser.Num(a.v()*a.Effect), "Map mod of Insulation", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L271
	"of Miring": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("DodgeChanceIsUnlucky", modparser.Flag, modparser.Bool(true), "Map mod of Miring", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("SpellSuppressionEffect", modparser.Base, modparser.Num(-a.v1(1)*a.Effect), "Map mod of Miring", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("Accuracy", modparser.Inc, modparser.Num(a.v1(2)*a.Effect), "Map mod of Miring", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L282
	"of Rust": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("BlockChance", modparser.Inc, modparser.Num(-a.v1(1)*a.Effect), "Map mod of Rust", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("Armour", modparser.More, modparser.Num(-a.v1(2)*a.Effect), "Map mod of Rust", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L292
	"of Smothering": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("LifeRecoveryRate", modparser.More, modparser.Num(-a.v()*a.Effect), "Map mod of Smothering", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("EnergyShieldRecoveryRate", modparser.More, modparser.Num(-a.v()*a.Effect), "Map mod of Smothering", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L301
	"of Stasis": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("NoLifeRegen", modparser.Flag, modparser.Bool(true), "Map mod of Stasis", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("NoEnergyShieldRegen", modparser.Flag, modparser.Bool(true), "Map mod of Stasis", true, modparser.FlagNone, modparser.KeywordNone))
		t.Mods.AddMod(modparser.NewModFull("NoManaRegen", modparser.Flag, modparser.Bool(true), "Map mod of Stasis", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L311
	"of Toughness": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("SelfCritMultiplier", modparser.Inc, modparser.Num(-(a.v1(1)+(a.v1(2)-a.v1(1))*a.RollRange/100)*a.Effect), "Map mod of Toughness", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L319
	"of Fatigue": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("CooldownRecovery", modparser.More, modparser.Num(-a.v()*a.Effect), "Map mod of Fatigue", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L334
	"of Doubt": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("AuraEffect", modparser.Inc, modparser.Num(-a.v()*a.Effect), "Map mod of Doubt", true, modparser.FlagNone, modparser.KeywordNone, &modparser.SkillTypeTag{SkillType: modparser.SkillTypeAura}, &modparser.SkillTypeTag{SkillType: modparser.SkillTypeAppliesCurse}))
	},
	// Data/ModMap.lua L342
	"of Imprecision": func(a mapAffixArgs, t *Tab) {
		t.Mods.AddMod(modparser.NewModFull("Accuracy", modparser.More, modparser.Num(-a.v()*a.Effect), "Map mod of Imprecision", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L356
	"of Venom": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("PoisonChance", modparser.Base, modparser.Num(100), "Map mod of Venom", true, modparser.FlagNone, modparser.KeywordNone))
	},
	// Data/ModMap.lua L364
	"of Deadliness": func(a mapAffixArgs, t *Tab) {
		t.EnemyMods.AddMod(modparser.NewModFull("CritChance", modparser.Inc, modparser.Num((a.v2(1, 1)+(a.v2(1, 2)-a.v2(1, 1))*a.RollRange/100)*a.Effect), "Map mod of Deadliness", true, modparser.FlagNone, modparser.KeywordNone))
		t.EnemyMods.AddMod(modparser.NewModFull("CritMultiplier", modparser.Base, modparser.Num((a.v2(2, 1)+(a.v2(2, 2)-a.v2(2, 1))*a.RollRange/100)*a.Effect), "Map mod of Deadliness", true, modparser.FlagNone, modparser.KeywordNone))
	},
}
