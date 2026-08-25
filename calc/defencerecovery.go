// CalcDefence.lua L1119-1400: spell suppression, dodge, recovery rate
// modifiers, leech caps, regeneration and energy shield recharge.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
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

	spellSuppressionChance := modDB.Sum("BASE", nil, "SpellSuppressionChance")
	totalSpellSuppressionChance := spellSuppressionChance
	if ov := modDB.Override(nil, "SpellSuppressionChance"); truthy(ov) {
		totalSpellSuppressionChance = anyNum(ov)
	}

	// Acrobatics Spell Suppression to Spell Dodge Chance conversion
	if modDB.Flag(nil, "ConvertSpellSuppressionToSpellDodge") {
		modDB.AddMod(newMod("SpellDodgeChance", "BASE", spellSuppressionChance/2, "Acrobatics"))
	}

	output["SpellSuppressionChance"] = math.Min(totalSpellSuppressionChance, data.Misc.SuppressionChanceCap)
	output["SpellSuppressionEffect"] = math.Max(data.Misc.SuppressionEffect+modDB.Sum("BASE", nil, "SpellSuppressionEffect"), 0)

	if enemyDB.Flag(nil, "CannotBeSuppressed") {
		output["EffectiveSpellSuppressionChance"] = 0.0
	} else {
		output["EffectiveSpellSuppressionChance"] = outNum(output, "SpellSuppressionChance")
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
	if worstOf := env.ConfigInput["EHPUnluckyWorstOf"]; truthy(worstOf) && anyNum(worstOf) != 1 {
		suppressRolls = -anyNum(worstOf) / 2
	}
	if suppressRolls != 0 {
		suppressChance := outNum(output, "EffectiveSpellSuppressionChance") / 100
		if modDB.Flag(nil, "Unexciting") {
			output["EffectiveSpellSuppressionChance"] = (3*math.Pow(suppressChance, 2) - 2*math.Pow(suppressChance, 3)) * 100
		} else if suppressRolls > 0 {
			output["EffectiveSpellSuppressionChance"] = (1 - math.Pow(1-suppressChance, suppressRolls+1)) * 100
		} else {
			output["EffectiveSpellSuppressionChance"] = math.Pow(suppressChance, math.Abs(suppressRolls)) * outNum(output, "EffectiveSpellSuppressionChance")
		}
	}
	output["SpellSuppressionChanceOverCap"] = math.Max(0, totalSpellSuppressionChance-data.Misc.SuppressionChanceCap)

	// Dodge
	totalAttackDodgeChance := modDB.Sum("BASE", nil, "AttackDodgeChance")
	totalSpellDodgeChance := modDB.Sum("BASE", nil, "SpellDodgeChance")
	attackDodgeChanceMax := data.Misc.DodgeChanceCap
	spellDodgeChanceMax := modDB.Sum("BASE", nil, "SpellDodgeChanceMax")
	if ov := modDB.Override(nil, "SpellDodgeChanceMax"); truthy(ov) {
		spellDodgeChanceMax = anyNum(ov)
	}
	enemyReduceDodgeChance := enemyDB.Sum("BASE", nil, "reduceEnemyDodge")

	output["AttackDodgeChance"] = math.Min(totalAttackDodgeChance, attackDodgeChanceMax)
	output["SpellDodgeChance"] = math.Min(totalSpellDodgeChance, spellDodgeChanceMax)
	if enemyDB.Flag(nil, "CannotBeDodged") {
		output["EffectiveAttackDodgeChance"] = 0.0
		output["EffectiveSpellDodgeChance"] = 0.0
	} else {
		output["EffectiveAttackDodgeChance"] = math.Min(math.Max(totalAttackDodgeChance-enemyReduceDodgeChance, 0), attackDodgeChanceMax)
		output["EffectiveSpellDodgeChance"] = math.Min(math.Max(totalSpellDodgeChance-enemyReduceDodgeChance, 0), spellDodgeChanceMax)
	}
	if env.ModeEffective && modDB.Flag(nil, "DodgeChanceIsUnlucky") {
		output["EffectiveAttackDodgeChance"] = outNum(output, "EffectiveAttackDodgeChance") / 100 * outNum(output, "EffectiveAttackDodgeChance")
		output["EffectiveSpellDodgeChance"] = outNum(output, "EffectiveSpellDodgeChance") / 100 * outNum(output, "EffectiveSpellDodgeChance")
	}
	output["AttackDodgeChanceOverCap"] = math.Max(0, totalAttackDodgeChance-attackDodgeChanceMax)
	output["SpellDodgeChanceOverCap"] = math.Max(0, totalSpellDodgeChance-spellDodgeChanceMax)

	// Recovery modifiers
	output["LifeRecoveryRateMod"] = 1.0
	if !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
		output["LifeRecoveryRateMod"] = Mod(modDB, nil, "LifeRecoveryRate")
	}
	output["ManaRecoveryRateMod"] = Mod(modDB, nil, "ManaRecoveryRate")
	output["EnergyShieldRecoveryRateMod"] = Mod(modDB, nil, "EnergyShieldRecoveryRate")

	// Leech caps
	output["MaxLifeLeechInstance"] = outNum(output, "Life") * Val(modDB, "MaxLifeLeechInstance", nil) / 100
	output["MaxLifeLeechRatePercent"] = Val(modDB, "MaxLifeLeechRate", nil)
	if modDB.Flag(nil, "MaximumLifeLeechIsEqualToParent") {
		output["MaxLifeLeechRatePercent"] = outNum(actor.parent.output, "MaxLifeLeechRatePercent")
	} else if modDB.Flag(nil, "MaximumLifeLeechIsEqualToPartyMember") {
		panic("defence: MaximumLifeLeechIsEqualToPartyMember needs the party tab")
	}
	output["MaxLifeLeechRate"] = outNum(output, "Life") * outNum(output, "MaxLifeLeechRatePercent") / 100
	output["MaxEnergyShieldLeechInstance"] = outNum(output, "EnergyShield") * Val(modDB, "MaxEnergyShieldLeechInstance", nil) / 100
	output["MaxEnergyShieldLeechRate"] = outNum(output, "EnergyShield") * Val(modDB, "MaxEnergyShieldLeechRate", nil) / 100
	output["MaxManaLeechInstance"] = outNum(output, "Mana") * Val(modDB, "MaxManaLeechInstance", nil) / 100
	output["MaxManaLeechRate"] = outNum(output, "Mana") * Val(modDB, "MaxManaLeechRate", nil) / 100

	// Regeneration
	for i, resourceName := range regenResources {
		resource := strings.ReplaceAll(resourceName, " ", "")
		pool := outNum(output, resource)
		baseRegen := 0.0
		inc := modDB.Sum("INC", nil, resource+"Regen")
		more := modDB.More(nil, resource+"Regen")
		regen := 0.0
		regenRate := 0.0
		recoveryRateMod := 1.0
		if v, ok := output[resource+"RecoveryRateMod"]; ok && truthy(v) {
			recoveryRateMod = anyNum(v)
		}
		if modDB.Flag(nil, "No"+resource+"Regen") || modDB.Flag(nil, "CannotGain"+resource) {
			output[resource+"Regen"] = 0.0
		} else if resource == "Life" && modDB.Flag(nil, "ZealotsOath") {
			output["LifeRegen"] = 0.0
			if lifeBase := modDB.Sum("BASE", nil, "LifeRegen"); lifeBase > 0 {
				modDB.AddMod(newMod("EnergyShieldRegen", "BASE", lifeBase, "Zealot's Oath"))
			}
			if lifePercent := modDB.Sum("BASE", nil, "LifeRegenPercent"); lifePercent > 0 {
				modDB.AddMod(newMod("EnergyShieldRegenPercent", "BASE", lifePercent, "Zealot's Oath"))
			}
		} else {
			if inc != 0 {
				// legacy chain breaker: redirect the increase to a later resource
				for j := i + 1; j < len(regenResources); j++ {
					other := strings.ReplaceAll(regenResources[j], " ", "")
					if modDB.Flag(nil, resource+"RegenTo"+other+"Regen") {
						modDB.AddMod(newMod(other+"Regen", "INC", inc, resourceName+" instead applies to "+regenResources[j]))
						inc = 0
					}
				}
			}
			if resource == "Life" && modDB.Sum("BASE", nil, "LifeRegenAppliesToEnergyShield") > 0 {
				conversion := math.Min(modDB.Sum("BASE", nil, "LifeRegenAppliesToEnergyShield"), 100) / 100
				lifeBase := modDB.Sum("BASE", nil, "LifeRegen")
				lifePercent := modDB.Sum("BASE", nil, "LifeRegenPercent")
				modDB.AddMod(newMod("EnergyShieldRegen", "BASE", floorDec(lifeBase*conversion, 2), "Life Regen to ES Regen"))
				modDB.AddMod(newMod("EnergyShieldRegenPercent", "BASE", floorDec(lifePercent*conversion, 2), "Life Regen to ES Regen"))
			}
			baseRegen = modDB.Sum("BASE", nil, resource+"Regen") + pool*modDB.Sum("BASE", nil, resource+"RegenPercent")/100
			regen = baseRegen * (1 + inc/100) * more
			if regen != 0 {
				// Pious Path
				for j := i + 1; j < len(regenResources); j++ {
					other := strings.ReplaceAll(regenResources[j], " ", "")
					if modDB.Flag(nil, resource+"RegenerationRecovers"+other) {
						modDB.AddMod(newMod(other+"Recovery", "BASE", regen, resourceName+" Regeneration Recovers "+regenResources[j]))
					}
				}
			}
			regenRate = roundDec(regen*recoveryRateMod, 1)
			output[resource+"Regen"] = regenRate
		}
		output[resource+"RegenInc"] = inc
		baseDegen := modDB.Sum("BASE", nil, resource+"Degen") + pool*modDB.Sum("BASE", nil, resource+"DegenPercent")/100
		tinctureDegenPercent := modDB.Sum("BASE", nil, resource+"DegenPercentTincture")
		// tincture minimum 1 degen per stack
		baseDegen += math.Max(pool*tinctureDegenPercent/100, tinctureDegenPercent)
		degenRate := 0.0
		if baseDegen > 0 {
			degenRate = baseDegen * Mod(modDB, nil, resource+"Degen")
		}
		output[resource+"Degen"] = degenRate
		recoveryRate := modDB.Sum("BASE", nil, resource+"Recovery") * recoveryRateMod
		output[resource+"Recovery"] = recoveryRate
		effectiveRegen := regenRate
		if modDB.Flag(nil, "UnaffectedBy"+resource+"Regen") {
			effectiveRegen = 0
		}
		output[resource+"RegenRecovery"] = effectiveRegen - degenRate + recoveryRate
		if outNum(output, resource+"RegenRecovery") > 0 {
			modDB.AddMod(newMod("Condition:CanGain"+resource, "FLAG", true, resourceName+"Regen"))
		}
		if pool > 0 {
			output[resource+"RegenPercent"] = roundDec(outNum(output, resource+"RegenRecovery")/pool*100, 1)
		} else {
			output[resource+"RegenPercent"] = 0.0
		}
	}

	// Energy Shield Recharge
	// `Flag(A) and not Flag(B)`: nil (no key) when A is unset, else a bool
	if modDB.Flag(nil, "EnergyShieldRechargeAppliesToLife") {
		output["EnergyShieldRechargeAppliesToLife"] = !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech")
	}
	output["EnergyShieldRechargeAppliesToEnergyShield"] = !(modDB.Flag(nil, "NoEnergyShieldRecharge") ||
		modDB.Flag(nil, "CannotGainEnergyShield") || truthy(output["EnergyShieldRechargeAppliesToLife"]))

	if truthy(output["EnergyShieldRechargeAppliesToLife"]) || truthy(output["EnergyShieldRechargeAppliesToEnergyShield"]) {
		inc := modDB.Sum("INC", nil, "EnergyShieldRecharge")
		more := modDB.More(nil, "EnergyShieldRecharge")
		base := data.Misc.EnergyShieldRechargeBase
		if ov := modDB.Override(nil, "EnergyShieldRecharge"); truthy(ov) {
			base = anyNum(ov)
		}
		if truthy(output["EnergyShieldRechargeAppliesToLife"]) {
			recharge := outNum(output, "Life") * base * (1 + inc/100) * more
			output["LifeRecharge"] = roundDec(recharge*outNum(output, "LifeRecoveryRateMod"), 0)
		} else {
			recharge := outNum(output, "EnergyShield") * base * (1 + inc/100) * more
			output["EnergyShieldRecharge"] = roundDec(recharge*outNum(output, "EnergyShieldRecoveryRateMod"), 0)
		}
		output["EnergyShieldRechargeDelay"] = data.Misc.EnergyShieldRechargeDelay / (1 + modDB.Sum("INC", nil, "EnergyShieldRechargeFaster")/100)
	} else {
		output["EnergyShieldRecharge"] = 0.0
	}

	env.defenceRecoup(actor)
}
