package config

import (
	"math"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// ConfigOptions.lua L2062: the enemy is rare or unique.
func applyEnemyRareOrUnique(_ Value, t *Tab) {
	t.enemyMod("Condition:RareOrUnique", modparser.Flag, modparser.Bool(true), effective())
}

// applyEnemyIsBoss ports the enemyIsBoss preset (ConfigOptions.lua
// L2065-2201). Besides the modifiers it grants, it resets the whole enemy
// stat block's placeholders, so it is the option that decides what an
// unfilled enemy field defaults to.
func applyEnemyIsBoss(v Value, t *Tab) {
	preset, _ := StrOf(v)

	// Set first so every branch starts from the same speed and crit.
	t.SetPlaceholder("enemySpeed", 700)
	t.SetPlaceholder("enemyCritChance", 5)
	t.SetPlaceholder("enemyCritDamage", data.MonsterConstants["base_critical_strike_multiplier"]-100)

	switch preset {
	case "None":
		t.ClearPlaceholder("enemyLightningResist")
		t.ClearPlaceholder("enemyColdResist")
		t.ClearPlaceholder("enemyFireResist")
		t.ClearPlaceholder("enemyChaosResist")

		t.ClearPlaceholder("enemyLevel")
		defaultLevel := t.presetLevel(83, false)

		t.SetPlaceholder("enemyPhysicalDamage", round(monsterDamage(defaultLevel)*1.5))
		t.ClearPlaceholder("enemyLightningDamage")
		t.ClearPlaceholder("enemyColdDamage")
		t.ClearPlaceholder("enemyFireDamage")
		t.ClearPlaceholder("enemyChaosDamage")

		t.ClearPlaceholder("enemyPhysicalOverwhelm")
		t.ClearPlaceholder("enemyLightningPen")
		t.ClearPlaceholder("enemyColdPen")
		t.ClearPlaceholder("enemyFirePen")

		t.SetPlaceholder("enemyArmour", monsterArmour(defaultLevel))
		t.SetPlaceholder("enemyEvasion", monsterEvasion(defaultLevel))

	case "Boss":
		t.enemyMod("Condition:RareOrUnique", modparser.Flag, modparser.Bool(true), effective())
		t.bossEnemyMod("AilmentThreshold", modparser.More, modparser.Num(488))
		t.bossMod("WarcryPower", modparser.Base, modparser.Num(20))
		t.bossMod("Multiplier:EnemyPower", modparser.Base, modparser.Num(20))

		t.SetPlaceholder("enemyLightningResist", 40)
		t.SetPlaceholder("enemyColdResist", 40)
		t.SetPlaceholder("enemyFireResist", 40)
		t.SetPlaceholder("enemyChaosResist", 25)

		t.ClearPlaceholder("enemyLevel")
		defaultLevel := t.presetLevel(83, false)

		defaultDamage := round(monsterDamage(defaultLevel) * 1.5 * data.Misc.StdBossDPSMult)
		t.SetPlaceholder("enemyPhysicalDamage", defaultDamage)
		t.SetPlaceholder("enemyLightningDamage", defaultDamage)
		t.SetPlaceholder("enemyColdDamage", defaultDamage)
		t.SetPlaceholder("enemyFireDamage", defaultDamage)
		t.SetPlaceholder("enemyChaosDamage", round(defaultDamage/2.5))

		t.ClearPlaceholder("enemyPhysicalOverwhelm")
		t.ClearPlaceholder("enemyLightningPen")
		t.ClearPlaceholder("enemyColdPen")
		t.ClearPlaceholder("enemyFirePen")

		t.SetPlaceholder("enemyArmour", monsterArmour(defaultLevel))
		t.SetPlaceholder("enemyEvasion", monsterEvasion(defaultLevel))

	case "Pinnacle", "Uber":
		uber := preset == "Uber"
		t.enemyMod("Condition:RareOrUnique", modparser.Flag, modparser.Bool(true), effective())
		t.enemyMod("Condition:PinnacleBoss", modparser.Flag, modparser.Bool(true), effective())
		if uber {
			t.bossEnemyMod("DamageTaken", modparser.More, modparser.Num(-70))
		}
		t.bossEnemyMod("AilmentThreshold", modparser.More, modparser.Num(404))
		t.bossMod("WarcryPower", modparser.Base, modparser.Num(20))
		t.bossMod("Multiplier:EnemyPower", modparser.Base, modparser.Num(20))

		t.SetPlaceholder("enemyLightningResist", 50)
		t.SetPlaceholder("enemyColdResist", 50)
		t.SetPlaceholder("enemyFireResist", 50)
		t.SetPlaceholder("enemyChaosResist", 30)

		base, dpsMult, pen := 84.0, data.Misc.PinnacleBossDPSMult, data.Misc.PinnacleBossPen
		armourMean, evasionMean := data.BossStats.PinnacleArmourMean, data.BossStats.PinnacleEvasionMean
		chaosDivisor := 2.5
		if uber {
			base, dpsMult, pen = 85, data.Misc.UberBossDPSMult, data.Misc.UberBossPen
			armourMean, evasionMean = data.BossStats.UberArmourMean, data.BossStats.UberEvasionMean
			chaosDivisor = 4
		}
		t.SetPlaceholder("enemyLevel", base)
		// A pinnacle boss's level is not capped by the character's: the
		// preset takes whichever of the two is higher.
		defaultLevel := t.presetLevel(base, true)

		defaultDamage := round(monsterDamage(defaultLevel) * 1.5 * dpsMult)
		t.SetPlaceholder("enemyPhysicalDamage", defaultDamage)
		t.SetPlaceholder("enemyLightningDamage", defaultDamage)
		t.SetPlaceholder("enemyColdDamage", defaultDamage)
		t.SetPlaceholder("enemyFireDamage", defaultDamage)
		t.SetPlaceholder("enemyChaosDamage", round(defaultDamage/chaosDivisor))

		t.SetPlaceholder("enemyLightningPen", pen)
		t.SetPlaceholder("enemyColdPen", pen)
		t.SetPlaceholder("enemyFirePen", pen)

		t.SetPlaceholder("enemyArmour", round(monsterArmour(defaultLevel)*(armourMean/100)))
		t.SetPlaceholder("enemyEvasion", round(monsterEvasion(defaultLevel)*(evasionMean/100)))
	}
}

// presetLevel re-settles the enemy level after the preset has written its
// own placeholder, and reports the level the preset's stat tables are read
// at. atLeast keeps the preset's own floor, for the bosses whose level the
// character's does not cap.
func (t *Tab) presetLevel(fallback float64, atLeast bool) float64 {
	t.UpdateLevel()
	if t.EnemyLevel == 0 {
		return fallback
	}
	if atLeast {
		return math.Max(t.EnemyLevel, fallback)
	}
	return t.EnemyLevel
}

// The monster stat tables are 1-based in the reference.
func monsterDamage(level float64) float64 { return data.MonsterDamageTable[int(level)-1] }
func monsterArmour(level float64) float64 { return data.MonsterArmourTable[int(level)-1] }
func monsterEvasion(level float64) float64 {
	return data.MonsterEvasionTable[int(level)-1]
}

// round is the reference's global round() at zero decimals.
func round(v float64) float64 { return util.RoundHalfUp(v, 0) }
