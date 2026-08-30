// CalcDefence.lua L1119-1400: spell suppression, dodge, recovery rate
// modifiers, leech caps, regeneration and energy shield recharge.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"
	"strings"
)

// regenResources is the reference's resource list; the chain-breaker and
// Pious Path loops only ever look FORWARD in it, so order is load-bearing.
var regenResources = []string{"Mana", "Life", "Energy Shield", "Rage"}

func (env *Env) defenceRecovery(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	spellSuppressionChance := modDB.Sum(modparser.Base, nil, "SpellSuppressionChance")
	totalSpellSuppressionChance := spellSuppressionChance
	if ov, ok := modDB.Override(nil, "SpellSuppressionChance"); ok {
		totalSpellSuppressionChance = valueNum(ov)
	}

	// Acrobatics Spell Suppression to Spell Dodge Chance conversion
	if modDB.Flag(nil, "ConvertSpellSuppressionToSpellDodge") {
		modDB.AddMod(newModS("SpellDodgeChance", modparser.Base, modparser.Num(spellSuppressionChance/2), "Acrobatics"))
	}

	output.SetN("SpellSuppressionChance", math.Min(totalSpellSuppressionChance, data.Misc.SuppressionChanceCap))
	output.SetN("SpellSuppressionEffect", math.Max(data.Misc.SuppressionEffect+modDB.Sum(modparser.Base, nil, "SpellSuppressionEffect"), 0))

	if enemyDB.Flag(nil, "CannotBeSuppressed") {
		output.SetN("EffectiveSpellSuppressionChance", 0.0)
	} else {
		output.SetN("EffectiveSpellSuppressionChance", output.N("SpellSuppressionChance"))
	}
	suppressRolls := 0.0
	if env.ModeEffective {
		if modDB.Flag(nil, "SpellSuppressionChanceIsLucky") {
			suppressRolls++
		}
		if modDB.Flag(nil, "SpellSuppressionChanceIsUnlucky") {
			suppressRolls--
		}
		if modDB.Flag(nil, "ExtremeLuck") {
			suppressRolls *= 2
		}
	}
	// unlucky config to lower the value of block, dodge, evade etc for ehp
	if worstOf := env.ConfigInput.EHPUnluckyWorstOf; worstOf.Set && worstOf.V != 1 {
		suppressRolls = -worstOf.V / 2
	}
	if suppressRolls != 0 {
		suppressChance := output.N("EffectiveSpellSuppressionChance") / 100
		if modDB.Flag(nil, "Unexciting") {
			output.SetN("EffectiveSpellSuppressionChance", (3*math.Pow(suppressChance, 2)-2*math.Pow(suppressChance, 3))*100)
		} else if suppressRolls > 0 {
			output.SetN("EffectiveSpellSuppressionChance", (1-math.Pow(1-suppressChance, suppressRolls+1))*100)
		} else {
			output.SetN("EffectiveSpellSuppressionChance", math.Pow(suppressChance, math.Abs(suppressRolls))*output.N("EffectiveSpellSuppressionChance"))
		}
	}
	output.SetN("SpellSuppressionChanceOverCap", math.Max(0, totalSpellSuppressionChance-data.Misc.SuppressionChanceCap))

	// Dodge
	totalAttackDodgeChance := modDB.Sum(modparser.Base, nil, "AttackDodgeChance")
	totalSpellDodgeChance := modDB.Sum(modparser.Base, nil, "SpellDodgeChance")
	attackDodgeChanceMax := data.Misc.DodgeChanceCap
	spellDodgeChanceMax := modDB.Sum(modparser.Base, nil, "SpellDodgeChanceMax")
	if ov, ok := modDB.Override(nil, "SpellDodgeChanceMax"); ok {
		spellDodgeChanceMax = valueNum(ov)
	}
	enemyReduceDodgeChance := enemyDB.Sum(modparser.Base, nil, "reduceEnemyDodge")

	output.SetN("AttackDodgeChance", math.Min(totalAttackDodgeChance, attackDodgeChanceMax))
	output.SetN("SpellDodgeChance", math.Min(totalSpellDodgeChance, spellDodgeChanceMax))
	if enemyDB.Flag(nil, "CannotBeDodged") {
		output.SetN("EffectiveAttackDodgeChance", 0.0)
		output.SetN("EffectiveSpellDodgeChance", 0.0)
	} else {
		output.SetN("EffectiveAttackDodgeChance", math.Min(math.Max(totalAttackDodgeChance-enemyReduceDodgeChance, 0), attackDodgeChanceMax))
		output.SetN("EffectiveSpellDodgeChance", math.Min(math.Max(totalSpellDodgeChance-enemyReduceDodgeChance, 0), spellDodgeChanceMax))
	}
	if env.ModeEffective && modDB.Flag(nil, "DodgeChanceIsUnlucky") {
		output.SetN("EffectiveAttackDodgeChance", output.N("EffectiveAttackDodgeChance")/100*output.N("EffectiveAttackDodgeChance"))
		output.SetN("EffectiveSpellDodgeChance", output.N("EffectiveSpellDodgeChance")/100*output.N("EffectiveSpellDodgeChance"))
	}
	output.SetN("AttackDodgeChanceOverCap", math.Max(0, totalAttackDodgeChance-attackDodgeChanceMax))
	output.SetN("SpellDodgeChanceOverCap", math.Max(0, totalSpellDodgeChance-spellDodgeChanceMax))

	// Recovery modifiers
	output.SetN("LifeRecoveryRateMod", 1.0)
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		output.SetN("LifeRecoveryRateMod", Mod(modDB, nil, "LifeRecoveryRate"))
	}
	output.SetN("ManaRecoveryRateMod", Mod(modDB, nil, "ManaRecoveryRate"))
	output.SetN("EnergyShieldRecoveryRateMod", Mod(modDB, nil, "EnergyShieldRecoveryRate"))

	// Leech caps
	output.SetN("MaxLifeLeechInstance", output.N("Life")*Val(modDB, "MaxLifeLeechInstance", nil)/100)
	output.SetN("MaxLifeLeechRatePercent", Val(modDB, "MaxLifeLeechRate", nil))
	if modDB.Flag(nil, "MaximumLifeLeechIsEqualToParent") {
		output.SetN("MaxLifeLeechRatePercent", actor.parent.output.N("MaxLifeLeechRatePercent"))
	} else if modDB.Flag(nil, "MaximumLifeLeechIsEqualToPartyMember") {
		panic("defence: MaximumLifeLeechIsEqualToPartyMember needs the party tab")
	}
	output.SetN("MaxLifeLeechRate", output.N("Life")*output.N("MaxLifeLeechRatePercent")/100)
	output.SetN("MaxEnergyShieldLeechInstance", output.N("EnergyShield")*Val(modDB, "MaxEnergyShieldLeechInstance", nil)/100)
	output.SetN("MaxEnergyShieldLeechRate", output.N("EnergyShield")*Val(modDB, "MaxEnergyShieldLeechRate", nil)/100)
	output.SetN("MaxManaLeechInstance", output.N("Mana")*Val(modDB, "MaxManaLeechInstance", nil)/100)
	output.SetN("MaxManaLeechRate", output.N("Mana")*Val(modDB, "MaxManaLeechRate", nil)/100)

	// Regeneration
	for i, resourceName := range regenResources {
		resource := strings.ReplaceAll(resourceName, " ", "")
		pool := output.N(resource)
		baseRegen := 0.0
		inc := modDB.Sum(modparser.Inc, nil, resource+"Regen")
		more := modDB.More(nil, resource+"Regen")
		regen := 0.0
		regenRate := 0.0
		recoveryRateMod := 1.0
		if v, ok := output[resource+"RecoveryRateMod"]; ok && v.Truthy() {
			recoveryRateMod = v.Num()
		}
		if modDB.Flag(nil, "No"+resource+"Regen") || modDB.Flag(nil, "CannotGain"+resource) {
			output.SetN(resource+"Regen", 0.0)
		} else if resource == "Life" && modDB.Flag(nil, "ZealotsOath") {
			output.SetN("LifeRegen", 0.0)
			if lifeBase := modDB.Sum(modparser.Base, nil, "LifeRegen"); lifeBase > 0 {
				modDB.AddMod(newModS("EnergyShieldRegen", modparser.Base, modparser.Num(lifeBase), "Zealot's Oath"))
			}
			if lifePercent := modDB.Sum(modparser.Base, nil, "LifeRegenPercent"); lifePercent > 0 {
				modDB.AddMod(newModS("EnergyShieldRegenPercent", modparser.Base, modparser.Num(lifePercent), "Zealot's Oath"))
			}
		} else {
			if inc != 0 {
				// legacy chain breaker: redirect the increase to a later resource
				for j := i + 1; j < len(regenResources); j++ {
					other := strings.ReplaceAll(regenResources[j], " ", "")
					if modDB.Flag(nil, resource+"RegenTo"+other+"Regen") {
						modDB.AddMod(newModS(other+"Regen", modparser.Inc, modparser.Num(inc), resourceName+" instead applies to "+regenResources[j]))
						inc = 0
					}
				}
			}
			if resource == "Life" && modDB.Sum(modparser.Base, nil, "LifeRegenAppliesToEnergyShield") > 0 {
				conversion := math.Min(modDB.Sum(modparser.Base, nil, "LifeRegenAppliesToEnergyShield"), 100) / 100
				lifeBase := modDB.Sum(modparser.Base, nil, "LifeRegen")
				lifePercent := modDB.Sum(modparser.Base, nil, "LifeRegenPercent")
				modDB.AddMod(newModS("EnergyShieldRegen", modparser.Base, modparser.Num(floorDec(lifeBase*conversion, 2)), "Life Regen to ES Regen"))
				modDB.AddMod(newModS("EnergyShieldRegenPercent", modparser.Base, modparser.Num(floorDec(lifePercent*conversion, 2)), "Life Regen to ES Regen"))
			}
			baseRegen = modDB.Sum(modparser.Base, nil, resource+"Regen") + pool*modDB.Sum(modparser.Base, nil, resource+"RegenPercent")/100
			regen = baseRegen * (1 + inc/100) * more
			if regen != 0 {
				// Pious Path
				for j := i + 1; j < len(regenResources); j++ {
					other := strings.ReplaceAll(regenResources[j], " ", "")
					if modDB.Flag(nil, resource+"RegenerationRecovers"+other) {
						modDB.AddMod(newModS(other+"Recovery", modparser.Base, modparser.Num(regen), resourceName+" Regeneration Recovers "+regenResources[j]))
					}
				}
			}
			regenRate = util.RoundHalfUp(regen*recoveryRateMod, 1)
			output.SetN(resource+"Regen", regenRate)
		}
		output.SetN(resource+"RegenInc", inc)
		baseDegen := modDB.Sum(modparser.Base, nil, resource+"Degen") + pool*modDB.Sum(modparser.Base, nil, resource+"DegenPercent")/100
		tinctureDegenPercent := modDB.Sum(modparser.Base, nil, resource+"DegenPercentTincture")
		// tincture minimum 1 degen per stack
		baseDegen += math.Max(pool*tinctureDegenPercent/100, tinctureDegenPercent)
		degenRate := 0.0
		if baseDegen > 0 {
			degenRate = baseDegen * Mod(modDB, nil, resource+"Degen")
		}
		output.SetN(resource+"Degen", degenRate)
		recoveryRate := modDB.Sum(modparser.Base, nil, resource+"Recovery") * recoveryRateMod
		output.SetN(resource+"Recovery", recoveryRate)
		effectiveRegen := regenRate
		if modDB.Flag(nil, "UnaffectedBy"+resource+"Regen") {
			effectiveRegen = 0
		}
		output.SetN(resource+"RegenRecovery", effectiveRegen-degenRate+recoveryRate)
		if output.N(resource+"RegenRecovery") > 0 {
			modDB.AddMod(newModS("Condition:CanGain"+resource, modparser.Flag, modparser.Bool(true), resourceName+"Regen"))
		}
		if pool > 0 {
			output.SetN(resource+"RegenPercent", util.RoundHalfUp(output.N(resource+"RegenRecovery")/pool*100, 1))
		} else {
			output.SetN(resource+"RegenPercent", 0.0)
		}
	}

	// Energy Shield Recharge
	// `Flag(A) and not Flag(B)`: nil (no key) when A is unset, else a bool
	if modDB.Flag(nil, "EnergyShieldRechargeAppliesToLife") {
		output.SetFlag("EnergyShieldRechargeAppliesToLife", !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech"))
	}
	output.SetFlag("EnergyShieldRechargeAppliesToEnergyShield", !(modDB.Flag(nil, "NoEnergyShieldRecharge") ||
		modDB.Flag(nil, "CannotGainEnergyShield") || output.Flag("EnergyShieldRechargeAppliesToLife")))

	if output.Flag("EnergyShieldRechargeAppliesToLife") || output.Flag("EnergyShieldRechargeAppliesToEnergyShield") {
		inc := modDB.Sum(modparser.Inc, nil, "EnergyShieldRecharge")
		more := modDB.More(nil, "EnergyShieldRecharge")
		base := data.Misc.EnergyShieldRechargeBase
		if ov, ok := modDB.Override(nil, "EnergyShieldRecharge"); ok {
			base = valueNum(ov)
		}
		if output.Flag("EnergyShieldRechargeAppliesToLife") {
			recharge := output.N("Life") * base * (1 + inc/100) * more
			output.SetN("LifeRecharge", util.RoundHalfUp(recharge*output.N("LifeRecoveryRateMod"), 0))
		} else {
			recharge := output.N("EnergyShield") * base * (1 + inc/100) * more
			output.SetN("EnergyShieldRecharge", util.RoundHalfUp(recharge*output.N("EnergyShieldRecoveryRateMod"), 0))
		}
		output.SetN("EnergyShieldRechargeDelay", data.Misc.EnergyShieldRechargeDelay/(1+modDB.Sum(modparser.Inc, nil, "EnergyShieldRechargeFaster")/100))
	} else {
		output.SetN("EnergyShieldRecharge", 0.0)
	}

	env.defenceRecoup(actor)
}
