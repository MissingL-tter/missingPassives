package config

import (
	"github.com/MissingL-tter/missingPassives/modparser"
)

// applyFuncs binds each option's apply function to its variable.
// options.go carries the table's data half, converted once from the
// reference; these are the bodies, hand-ported with their source cited.
//
// An option missing from here has no body ported yet: it contributes no
// modifiers, and the calc differential is what reports the shortfall.
var applyFuncs = map[Var]func(Value, *Tab){
	"conditionEnemyRareOrUnique": applyEnemyRareOrUnique,
	"enemyIsBoss":                applyEnemyIsBoss,

	"resourceGainMode":                            applyResourceGainMode,
	"bloodsoakedBannerStages":                     applyBloodsoakedBannerStages,
	"conditionCorruptingCryStages":                applyCorruptingCryStages,
	"targetBrandedEnemy":                          applyTargetBrandedEnemy,
	"ConcPathBypassCD":                            applyConcPathBypassCD,
	"FlickerStrikeBypassCD":                       applyFlickerStrikeBypassCD,
	"VigilantStrikeBypassCD":                      applyVigilantStrikeBypassCD,
	"touchedDebuffsCount":                         applyTouchedDebuffsCount,
	"plagueBearerState":                           applyPlagueBearerState,
	"raiseSpectreEnableBuffs":                     applyRaiseSpectreEnableBuffs,
	"raiseSpectreEnableCurses":                    applyRaiseSpectreEnableCurses,
	"summonElementalRelicEnableAngerAura":         applyRelicAura("Anger"),
	"summonElementalRelicEnableHatredAura":        applyRelicAura("Hatred"),
	"summonElementalRelicEnableWrathAura":         applyRelicAura("Wrath"),
	"sigilOfPowerStages":                          applySigilOfPowerStages,
	"configSpectralTigerCount":                    applySpectralTigerCount,
	"enemySizePreset":                             applyEnemySizePreset,
	"repeatMode":                                  applyRepeatMode,
	"multiplierWitheredStackCountSelf":            applyWitheredStackCountSelf,
	"conditionChampionIntimidate":                 applyChampionIntimidate,
	"conditionSacrificeMinion":                    applySacrificeMinion,
	"GamblesprintMovementSpeed":                   applyGamblesprintMovementSpeed,
	"ScorchStacks":                                applyEnemyMultiplier("ScorchStacks"),
	"ShockStacks":                                 applyEnemyMultiplier("ShockStacks"),
	"multiplierEnemyHallowingFlame":               applyEnemyMultiplier("HallowingFlame"),
	"multiplierHallowingFlameStacksRemovedByAlly": applyHallowingFlameRemovedByAlly,
	"maniaDebuffsCount":                           applyManiaDebuffsCount,
	"enemyLightningResist":                        applyEnemyResist("LightningResist"),
	"enemyColdResist":                             applyEnemyResist("ColdResist"),
	"enemyFireResist":                             applyEnemyResist("FireResist"),
	"enemyChaosResist":                            applyEnemyResist("ChaosResist"),
	"enemyEvasion":                                applyEnemyDefence("Evasion"),
	"enemyArmour":                                 applyEnemyDefence("Armour"),
}

func init() {
	for v, fn := range applyFuncs {
		opt := byVar[v]
		if opt == nil {
			panic("config: apply function for unknown option " + string(v))
		}
		opt.Apply = fn
	}
}

// mod adds one Config-sourced modifier to the player's list.
func (t *Tab) mod(name string, typ modparser.ModType, v modparser.Value, tags ...modparser.Tag) {
	t.Mods.AddMod(modparser.NewModFull(name, typ, v, "Config", true, modparser.FlagNone, modparser.KeywordNone, tags...))
}

// enemyMod adds one Config-sourced modifier to the enemy's list.
func (t *Tab) enemyMod(name string, typ modparser.ModType, v modparser.Value, tags ...modparser.Tag) {
	t.EnemyMods.AddMod(modparser.NewModFull(name, typ, v, "Config", true, modparser.FlagNone, modparser.KeywordNone, tags...))
}

// bossMod and bossEnemyMod are the same with the "Boss" source the enemy
// preset stamps on the modifiers it grants.
func (t *Tab) bossMod(name string, typ modparser.ModType, v modparser.Value, tags ...modparser.Tag) {
	t.Mods.AddMod(modparser.NewModFull(name, typ, v, "Boss", true, modparser.FlagNone, modparser.KeywordNone, tags...))
}

func (t *Tab) bossEnemyMod(name string, typ modparser.ModType, v modparser.Value, tags ...modparser.Tag) {
	t.EnemyMods.AddMod(modparser.NewModFull(name, typ, v, "Boss", true, modparser.FlagNone, modparser.KeywordNone, tags...))
}

// effective is the { type = "Condition", var = "Effective" } tag the enemy
// presets attach, so their modifiers apply only to effective-DPS passes.
func effective() modparser.Tag {
	return &modparser.CondTag{Var: "Effective"}
}
