// CalcDefence.lua L826-1115: the primary defences block (ward, energy
// shield, armour, evasion) and evade chance.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// defenceSlots is the reference's slot list for gear defences. The Lua
// iterates it with pairs over an array, so the order is the array order.
var defenceSlots = []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"}

func (env *Env) defencePrimary(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

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
			conversion := math.Min(modDB.Sum(modparser.Base, nil, "BodyArmourArmourEvasionToWardPercent")/100, 1)
			convertedArmour := armourBase * conversion
			convertedEvasion := evasionBase * conversion
			armourBase -= convertedArmour
			evasionBase -= convertedEvasion
			wardBase += convertedEvasion + convertedArmour
		}
		if wardBase > 0 {
			if modDB.Flag(nil, "EnergyShieldToWard") {
				inc := modDB.Sum(modparser.Inc, slotCfg, "Ward", "Defences", "EnergyShield")
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

	if wardBase := modDB.Sum(modparser.Base, nil, "Ward"); wardBase > 0 {
		if modDB.Flag(nil, "EnergyShieldToWard") {
			inc := modDB.Sum(modparser.Inc, nil, "Ward", "Defences", "EnergyShield")
			more := modDB.More(nil, "Ward", "Defences")
			ward += wardBase * (1 + inc/100) * more
		} else {
			ward += wardBase * Mod(modDB, nil, "Ward", "Defences")
		}
	}
	if energyShieldBase := modDB.Sum(modparser.Base, nil, "EnergyShield"); energyShieldBase > 0 {
		if modDB.Flag(nil, "EnergyShieldToWard") {
			energyShield += energyShieldBase * modDB.More(nil, "EnergyShield", "Defences")
		} else {
			energyShield += energyShieldBase * Mod(modDB, nil, "EnergyShield", "Defences")
		}
	}
	if armourBase := modDB.Sum(modparser.Base, nil, "Armour", "ArmourAndEvasion"); armourBase > 0 {
		armour += armourBase * Mod(modDB, nil, "Armour", "ArmourAndEvasion", "Defences")
	}
	if evasionBase := modDB.Sum(modparser.Base, nil, "Evasion", "ArmourAndEvasion"); evasionBase > 0 {
		if ironReflexes {
			armour += evasionBase * Mod(modDB, nil, "Armour", "Evasion", "ArmourAndEvasion", "Defences")
		} else {
			evasion += evasionBase * Mod(modDB, nil, "Evasion", "ArmourAndEvasion", "Defences")
		}
	}
	if convManaToArmour := modDB.Sum(modparser.Base, nil, "ManaConvertToArmour"); convManaToArmour > 0 {
		armourBase := 2 * modDB.Sum(modparser.Base, nil, "Mana") * convManaToArmour / 100
		armour += armourBase * Mod(modDB, nil, "Mana", "Armour", "ArmourAndEvasion", "Defences")
	}
	if convManaToES := modDB.Sum(modparser.Base, nil, "ManaGainAsEnergyShield"); convManaToES > 0 {
		energyShieldBase := modDB.Sum(modparser.Base, nil, "Mana") * convManaToES / 100
		energyShield += energyShieldBase * Mod(modDB, nil, "Mana", "EnergyShield", "Defences")
	}
	if convLifeToArmour := modDB.Sum(modparser.Base, nil, "LifeGainAsArmour"); convLifeToArmour > 0 {
		armourBase := modDB.Sum(modparser.Base, nil, "Life") * convLifeToArmour / 100
		if modDB.Flag(nil, "ChaosInoculation") {
			armour += 1
		} else {
			armour += armourBase * Mod(modDB, nil, "Life", "Armour", "ArmourAndEvasion", "Defences")
		}
	}
	if convLifeToES := modDB.Sum(modparser.Base, nil, "LifeConvertToEnergyShield", "LifeGainAsEnergyShield"); convLifeToES > 0 {
		energyShieldBase := modDB.Sum(modparser.Base, nil, "Life") * convLifeToES / 100
		if modDB.Flag(nil, "ChaosInoculation") {
			energyShield += 1
		} else {
			energyShield += energyShieldBase * Mod(modDB, nil, "Life", "EnergyShield", "Defences")
		}
	}
	if convEvasionToArmour := modDB.Sum(modparser.Base, nil, "EvasionGainAsArmour"); convEvasionToArmour > 0 {
		armourBase := (modDB.Sum(modparser.Base, nil, "Evasion", "ArmourAndEvasion") + gearEvasion) * convEvasionToArmour / 100
		armour += armourBase * Mod(modDB, nil, "Evasion", "Armour", "ArmourAndEvasion", "Defences")
	}

	if ov, ok := modDB.Override(nil, "EnergyShield"); ok {
		output.SetN("EnergyShield", valueNum(ov))
	} else {
		output.SetN("EnergyShield", math.Max(util.RoundHalfUp(energyShield, 0), 0))
	}
	output.SetN("Armour", math.Max(util.RoundHalfUp(armour, 0), 0))
	output.SetN("Evasion", math.Max(util.RoundHalfUp(evasion, 0), 0))
	output.SetN("MeleeEvasion", math.Max(util.RoundHalfUp(evasion*Mod(modDB, nil, "MeleeEvasion"), 0), 0))
	output.SetN("ProjectileEvasion", math.Max(util.RoundHalfUp(evasion*Mod(modDB, nil, "ProjectileEvasion"), 0), 0))
	output.SetN("LowestOfArmourAndEvasion", math.Min(output.N("Armour"), output.N("Evasion")))
	output.SetN("Ward", math.Max(math.Floor(ward), 0))
	output.SetN("Gear:Ward", gearWard)
	output.SetN("Gear:EnergyShield", gearEnergyShield)
	output.SetN("Gear:Armour", gearArmour)
	output.SetN("Gear:Evasion", gearEvasion)

	armourCap := modDB.Flag(nil, "ArmourESRecoveryCap")
	evasionCap := modDB.Flag(nil, "EvasionESRecoveryCap")
	lowESConfig := env.ConfigInput.ConditionLowEnergyShield.Or(false)
	// Lua `A and B or C and D or configInput[...]`: when both guards fail the
	// chain yields the raw config value, which is nil (no key) when unset.
	cappingESValue := env.ConfigInput.ConditionLowEnergyShield
	switch {
	case armourCap && output.N("Armour") < output.N("EnergyShield"):
		cappingESValue = util.Some(true)
	case evasionCap && output.N("Evasion") < output.N("EnergyShield"):
		cappingESValue = util.Some(true)
	}
	if cappingESValue.Set {
		output.SetFlag("CappingES", cappingESValue.V)
	}
	if cappingES := cappingESValue.Or(false); cappingES {
		var cap float64
		switch {
		case armourCap && evasionCap:
			cap = math.Min(output.N("Armour"), output.N("Evasion"))
		case armourCap:
			cap = output.N("Armour")
		case evasionCap:
			cap = output.N("Evasion")
		default:
			cap = output.N("EnergyShield")
		}
		if lowESConfig {
			cap = math.Min(output.N("EnergyShield")*data.Misc.LowPoolThreshold, cap)
		}
		output.SetN("EnergyShieldRecoveryCap", cap)
	} else {
		output.SetN("EnergyShieldRecoveryCap", output.N("EnergyShield"))
	}

	if modDB.Flag(nil, "CannotEvade") || enemyDB.Flag(nil, "CannotBeEvaded") {
		output.SetN("EvadeChance", 0.0)
		output.SetN("MeleeEvadeChance", 0.0)
		output.SetN("ProjectileEvadeChance", 0.0)
	} else if modDB.Flag(nil, "AlwaysEvade") {
		output.SetN("EvadeChance", 100.0)
		output.SetN("MeleeEvadeChance", 100.0)
		output.SetN("ProjectileEvadeChance", 100.0)
	} else {
		enemyAccuracy := util.RoundHalfUp(Val(enemyDB, "Accuracy", nil), 0)
		evadeChance := modDB.Sum(modparser.Base, nil, "EvadeChance")
		hitCh := Mod(enemyDB, nil, "HitChance")
		evadeStat := output.N("Evasion")
		meleeEvadeStat := output.N("MeleeEvasion")
		projectileEvadeStat := output.N("ProjectileEvasion")
		if modDB.Flag(nil, "EvadeChanceBasedOnWard") {
			multiplier := overrideNum(modDB, nil, "EvadeChanceBasedOnWardPercent") / 100
			evadeStat = output.N("Ward") * multiplier
			meleeEvadeStat = evadeStat
			projectileEvadeStat = evadeStat
		}
		output.SetN("EvadeChance", 100-(hitChance(evadeStat, enemyAccuracy)-evadeChance)*hitCh)
		output.SetN("MeleeEvadeChance", math.Max(0, math.Min(data.Misc.EvadeChanceCap,
			(100-(hitChance(meleeEvadeStat, enemyAccuracy)-evadeChance)*hitCh)*Mod(modDB, nil, "EvadeChance", "MeleeEvadeChance"))))
		output.SetN("ProjectileEvadeChance", math.Max(0, math.Min(data.Misc.EvadeChanceCap,
			(100-(hitChance(projectileEvadeStat, enemyAccuracy)-evadeChance)*hitCh)*Mod(modDB, nil, "EvadeChance", "ProjectileEvadeChance"))))
		// Evade chance is only shown merged when melee and projectile agree
		if output.N("MeleeEvadeChance") != output.N("ProjectileEvadeChance") {
			output.SetFlag("splitEvade", true)
		} else {
			output.SetN("EvadeChance", output.N("MeleeEvadeChance"))
			output.SetFlag("noSplitEvade", true)
		}
	}

	env.defenceRecovery(actor)
}
