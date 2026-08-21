// CalcDefence.lua L826-1115: the primary defences block (ward, energy
// shield, armour, evasion) and evade chance.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modstore"
)

// defenceSlots is the reference's slot list for gear defences. The Lua
// iterates it with pairs over an array, so the order is the array order.
var defenceSlots = []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"}

func (env *Env) defencePrimary(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output
	d := env.Data

	ironReflexes := modDB.Flag(nil, "IronReflexes")
	ward, energyShield, armour, evasion := 0.0, 0.0, 0.0, 0.0
	gearWard, gearEnergyShield, gearArmour, gearEvasion := 0.0, 0.0, 0.0, 0.0

	for _, slot := range defenceSlots {
		it, _ := actor.ms.ItemList[slot].(*Item)
		if it == nil || it.In == nil || it.In.ArmourData == nil {
			continue
		}
		slotCfg := &modstore.Cfg{SlotName: slot}
		energyShieldBase := 0.0
		if !modDB.Flag(nil, "GainNoEnergyShieldFrom"+slot) {
			energyShieldBase = armourDataOf(it, "EnergyShield")
		}
		armourBase := 0.0
		if !modDB.Flag(nil, "GainNoArmourFrom"+slot) {
			armourBase = armourDataOf(it, "Armour")
		}
		evasionBase := 0.0
		if !(modDB.Flag(nil, "GainNoEvasionFrom"+slot) || (modDB.Flag(nil, "GainNoArmourFrom"+slot) && ironReflexes)) {
			evasionBase = armourDataOf(it, "Evasion")
		}
		wardBase := 0.0
		if !modDB.Flag(nil, "GainNoWardFrom"+slot) {
			wardBase = armourDataOf(it, "Ward")
		}
		if slot == "Body Armour" && modDB.Flag(nil, "ConvertBodyArmourArmourEvasionToWard") {
			conversion := math.Min(modDB.Sum("BASE", nil, "BodyArmourArmourEvasionToWardPercent")/100, 1)
			convertedArmour := armourBase * conversion
			convertedEvasion := evasionBase * conversion
			armourBase -= convertedArmour
			evasionBase -= convertedEvasion
			wardBase += convertedEvasion + convertedArmour
		}
		if wardBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				inc := modDB.Sum("INC", slotCfg, "Ward", "Defences", "EnergyShield")
				more := modDB.More(slotCfg, "Ward", "Defences")
				ward += wardBase * (1 + inc/100) * more
			} else {
				ward += wardBase * Mod(modDB, slotCfg, "Ward", "Defences")
			}
			gearWard += wardBase
		}
		if energyShieldBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				energyShield += energyShieldBase * modDB.More(slotCfg, "EnergyShield", "Defences")
				gearEnergyShield += energyShieldBase
			} else if !modDB.Flag(nil, "ConvertArmourESToLife") {
				energyShield += energyShieldBase * Mod(modDB, slotCfg, "EnergyShield", "Defences", slot+"ESAndArmour")
				gearEnergyShield += energyShieldBase
			}
		}
		if armourBase > 0 {
			armour += armourBase * Mod(modDB, slotCfg, "Armour", "ArmourAndEvasion", "Defences", slot+"ESAndArmour")
			gearArmour += armourBase
		}
		if evasionBase > 0 {
			gearEvasion += evasionBase
			if ironReflexes {
				armour += evasionBase * Mod(modDB, slotCfg, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
			} else {
				evasion += evasionBase * Mod(modDB, slotCfg, "Evasion", "ArmourAndEvasion", "Defences")
			}
		}
	}

	if wardBase := modDB.Sum("BASE", nil, "Ward"); wardBase > 0 {
		if modDB.Flag(nil, "EnergyShieldToWard") {
			inc := modDB.Sum("INC", nil, "Ward", "Defences", "EnergyShield")
			more := modDB.More(nil, "Ward", "Defences")
			ward += wardBase * (1 + inc/100) * more
		} else {
			ward += wardBase * Mod(modDB, nil, "Ward", "Defences")
		}
	}
	if energyShieldBase := modDB.Sum("BASE", nil, "EnergyShield"); energyShieldBase > 0 {
		if modDB.Flag(nil, "EnergyShieldToWard") {
			energyShield += energyShieldBase * modDB.More(nil, "EnergyShield", "Defences")
		} else {
			energyShield += energyShieldBase * Mod(modDB, nil, "EnergyShield", "Defences")
		}
	}
	if armourBase := modDB.Sum("BASE", nil, "Armour", "ArmourAndEvasion"); armourBase > 0 {
		armour += armourBase * Mod(modDB, nil, "Armour", "ArmourAndEvasion", "Defences")
	}
	if evasionBase := modDB.Sum("BASE", nil, "Evasion", "ArmourAndEvasion"); evasionBase > 0 {
		if ironReflexes {
			armour += evasionBase * Mod(modDB, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
		} else {
			evasion += evasionBase * Mod(modDB, nil, "Evasion", "ArmourAndEvasion", "Defences")
		}
	}
	if convManaToArmour := modDB.Sum("BASE", nil, "ManaConvertToArmour"); convManaToArmour > 0 {
		armourBase := 2 * modDB.Sum("BASE", nil, "Mana") * convManaToArmour / 100
		armour += armourBase * Mod(modDB, nil, "Mana", "Armour", "ArmourAndEvasion", "Defences")
	}
	if convManaToES := modDB.Sum("BASE", nil, "ManaGainAsEnergyShield"); convManaToES > 0 {
		energyShieldBase := modDB.Sum("BASE", nil, "Mana") * convManaToES / 100
		energyShield += energyShieldBase * Mod(modDB, nil, "Mana", "EnergyShield", "Defences")
	}
	if convLifeToArmour := modDB.Sum("BASE", nil, "LifeGainAsArmour"); convLifeToArmour > 0 {
		armourBase := modDB.Sum("BASE", nil, "Life") * convLifeToArmour / 100
		if modDB.Flag(nil, "ChaosInoculation") {
			armour += 1
		} else {
			armour += armourBase * Mod(modDB, nil, "Life", "Armour", "ArmourAndEvasion", "Defences")
		}
	}
	if convLifeToES := modDB.Sum("BASE", nil, "LifeConvertToEnergyShield", "LifeGainAsEnergyShield"); convLifeToES > 0 {
		energyShieldBase := modDB.Sum("BASE", nil, "Life") * convLifeToES / 100
		if modDB.Flag(nil, "ChaosInoculation") {
			energyShield += 1
		} else {
			energyShield += energyShieldBase * Mod(modDB, nil, "Life", "EnergyShield", "Defences")
		}
	}
	if convEvasionToArmour := modDB.Sum("BASE", nil, "EvasionGainAsArmour"); convEvasionToArmour > 0 {
		armourBase := (modDB.Sum("BASE", nil, "Evasion", "ArmourAndEvasion") + gearEvasion) * convEvasionToArmour / 100
		armour += armourBase * Mod(modDB, nil, "Evasion", "Armour", "ArmourAndEvasion", "Defences")
	}

	if ov := modDB.Override(nil, "EnergyShield"); truthy(ov) {
		output["EnergyShield"] = anyNum(ov)
	} else {
		output["EnergyShield"] = math.Max(roundDec(energyShield, 0), 0)
	}
	output["Armour"] = math.Max(roundDec(armour, 0), 0)
	output["Evasion"] = math.Max(roundDec(evasion, 0), 0)
	output["MeleeEvasion"] = math.Max(roundDec(evasion*Mod(modDB, nil, "MeleeEvasion"), 0), 0)
	output["ProjectileEvasion"] = math.Max(roundDec(evasion*Mod(modDB, nil, "ProjectileEvasion"), 0), 0)
	output["LowestOfArmourAndEvasion"] = math.Min(outNum(output, "Armour"), outNum(output, "Evasion"))
	output["Ward"] = math.Max(math.Floor(ward), 0)
	output["Gear:Ward"] = gearWard
	output["Gear:EnergyShield"] = gearEnergyShield
	output["Gear:Armour"] = gearArmour
	output["Gear:Evasion"] = gearEvasion

	armourCap := modDB.Flag(nil, "ArmourESRecoveryCap")
	evasionCap := modDB.Flag(nil, "EvasionESRecoveryCap")
	lowESConfig := truthy(env.ConfigInput["conditionLowEnergyShield"])
	// Lua `A and B or C and D or configInput[...]`: when both guards fail the
	// chain yields the raw config value, which is nil (no key) when unset.
	var cappingESValue any
	switch {
	case armourCap && outNum(output, "Armour") < outNum(output, "EnergyShield"):
		cappingESValue = true
	case evasionCap && outNum(output, "Evasion") < outNum(output, "EnergyShield"):
		cappingESValue = true
	default:
		cappingESValue = env.ConfigInput["conditionLowEnergyShield"]
	}
	if cappingESValue != nil {
		output["CappingES"] = cappingESValue
	}
	if cappingES := truthy(cappingESValue); cappingES {
		var cap float64
		switch {
		case armourCap && evasionCap:
			cap = math.Min(outNum(output, "Armour"), outNum(output, "Evasion"))
		case armourCap:
			cap = outNum(output, "Armour")
		case evasionCap:
			cap = outNum(output, "Evasion")
		default:
			cap = outNum(output, "EnergyShield")
		}
		if lowESConfig {
			cap = math.Min(outNum(output, "EnergyShield")*d.Misc.LowPoolThreshold, cap)
		}
		output["EnergyShieldRecoveryCap"] = cap
	} else {
		output["EnergyShieldRecoveryCap"] = outNum(output, "EnergyShield")
	}

	if modDB.Flag(nil, "CannotEvade") || enemyDB.Flag(nil, "CannotBeEvaded") {
		output["EvadeChance"] = 0.0
		output["MeleeEvadeChance"] = 0.0
		output["ProjectileEvadeChance"] = 0.0
	} else if modDB.Flag(nil, "AlwaysEvade") {
		output["EvadeChance"] = 100.0
		output["MeleeEvadeChance"] = 100.0
		output["ProjectileEvadeChance"] = 100.0
	} else {
		enemyAccuracy := roundDec(Val(enemyDB, "Accuracy", nil), 0)
		evadeChance := modDB.Sum("BASE", nil, "EvadeChance")
		hitCh := Mod(enemyDB, nil, "HitChance")
		evadeStat := outNum(output, "Evasion")
		meleeEvadeStat := outNum(output, "MeleeEvasion")
		projectileEvadeStat := outNum(output, "ProjectileEvasion")
		if modDB.Flag(nil, "EvadeChanceBasedOnWard") {
			multiplier := anyNum(modDB.Override(nil, "EvadeChanceBasedOnWardPercent")) / 100
			evadeStat = outNum(output, "Ward") * multiplier
			meleeEvadeStat = evadeStat
			projectileEvadeStat = evadeStat
		}
		output["EvadeChance"] = 100 - (hitChance(evadeStat, enemyAccuracy)-evadeChance)*hitCh
		output["MeleeEvadeChance"] = math.Max(0, math.Min(d.Misc.EvadeChanceCap,
			(100-(hitChance(meleeEvadeStat, enemyAccuracy)-evadeChance)*hitCh)*Mod(modDB, nil, "EvadeChance", "MeleeEvadeChance")))
		output["ProjectileEvadeChance"] = math.Max(0, math.Min(d.Misc.EvadeChanceCap,
			(100-(hitChance(projectileEvadeStat, enemyAccuracy)-evadeChance)*hitCh)*Mod(modDB, nil, "EvadeChance", "ProjectileEvadeChance")))
		// Evade chance is only shown merged when melee and projectile agree
		if outNum(output, "MeleeEvadeChance") != outNum(output, "ProjectileEvadeChance") {
			output["splitEvade"] = true
		} else {
			output["EvadeChance"] = outNum(output, "MeleeEvadeChance")
			output["noSplitEvade"] = true
		}
	}

	env.defenceRecovery(actor)
}
