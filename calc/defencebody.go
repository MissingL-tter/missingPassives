// CalcDefence.lua L636-1634: calcs.defence. Split from defence.go (which
// holds resistances and the shared helpers) to keep the files readable.
package calc

import (
	"math"
)

// hitChance ports calcs.hitChance.
func hitChance(evasion, accuracy float64) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := accuracy / (accuracy + math.Pow(evasion/5, 0.9)) * 125
	return math.Max(math.Min(roundDec(rawChance, 0), 100), 5)
}

// armourReductionF ports calcs.armourReductionF.
func armourReductionF(armour, raw float64) float64 {
	if armour == 0 && raw == 0 {
		return 0
	}
	return armour / (armour + raw*5) * 100
}

// armourReduction ports calcs.armourReduction.
func armourReduction(armour, raw float64) float64 {
	return roundDec(armourReductionF(armour, raw), 0)
}

// armourDataOf reads one numeric field of a slot's armourData, or 0.
func armourDataOf(it *Item, key string) float64 {
	if it == nil || it.In == nil || it.In.ArmourData == nil {
		return 0
	}
	return anyNum(it.In.ArmourData[key])
}

// RunDefence runs the defence stage the way the reference reaches it after
// the perform body: the player first, then the minion when there is one.
func (env *Env) RunDefence() {
	env.defence(env.playerPA)
	if env.Minion != nil {
		env.defence(env.minionPA)
	}
}

// defence ports calcs.defence (CalcDefence.lua L636).
func (env *Env) defence(actor *performActor) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output
	d := env.Data

	// Action Speed
	output["ActionSpeedMod"] = env.actionSpeedMod(actor)

	env.Resistances(actor)
	if env.Minion != nil && modDB.Sum("BASE", nil, "ResistanceAddedToMinions") > 0 {
		for _, elem := range resistTypeList {
			final := outNum(output, elem+"Resist")
			env.Minion.DB.AddMod(newMod(elem+"Resist", "BASE",
				math.Floor(final*modDB.Sum("BASE", nil, "ResistanceAddedToMinions")/100), "Player"))
		}
	}
	// Formless Inferno
	if actor == env.minionPA {
		env.doActorLifeMana(actor)
		env.doActorLifeManaReservation(actor, false)
	}

	// Block
	output["BlockChanceMax"] = math.Min(modDB.Sum("BASE", nil, "BlockChanceMax"), d.Misc.BlockChanceCap)
	if modDB.Flag(nil, "MaximumBlockAttackChanceIsEqualToParent") {
		output["BlockChanceMax"] = outNum(actor.parent.output, "BlockChanceMax")
	} else if modDB.Flag(nil, "MaximumBlockAttackChanceIsEqualToPartyMember") {
		panic("defence: MaximumBlockAttackChanceIsEqualToPartyMember needs the party tab")
	}
	output["BlockChanceOverCap"] = 0.0
	output["SpellBlockChanceOverCap"] = 0.0
	baseBlockChance := 0.0
	for _, slot := range []string{"Weapon 2", "Weapon 3"} {
		it, _ := actor.ms.ItemList[slot].(*Item)
		baseBlockChance += armourDataOf(it, "BlockChance")
	}
	output["ShieldBlockChance"] = baseBlockChance
	if !env.Keystone.KeystonesAdded["Necromantic Aegis"] {
		if ov := modDB.Override(nil, "ReplaceShieldBlock"); truthy(ov) {
			baseBlockChance = anyNum(ov)
		}
	}
	// Apply player block overrides if Necromantic Aegis allocated
	if actor == env.minionPA && env.Keystone.KeystonesAdded["Necromantic Aegis"] {
		if ov := env.ModDB.Override(nil, "ReplaceShieldBlock"); truthy(ov) {
			baseBlockChance = anyNum(ov)
		}
	}

	if modDB.Flag(nil, "BlockAttackChanceIsEqualToParent") {
		output["BlockChance"] = math.Min(outNum(actor.parent.output, "BlockChance"), outNum(output, "BlockChanceMax"))
	} else if modDB.Flag(nil, "BlockAttackChanceIsEqualToPartyMember") {
		panic("defence: BlockAttackChanceIsEqualToPartyMember needs the party tab")
	} else if modDB.Flag(nil, "MaxBlockIfNotBlockedRecently") {
		output["BlockChance"] = outNum(output, "BlockChanceMax")
	} else {
		inc := modDB.Sum("INC", nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalBlockChance := roundDec((baseBlockChance+modDB.Sum("BASE", nil, "BlockChance"))*(1+inc/100)*more, 0)
		output["BlockChance"] = math.Min(totalBlockChance, outNum(output, "BlockChanceMax"))
		output["BlockChanceOverCap"] = math.Max(0, totalBlockChance-outNum(output, "BlockChanceMax"))
	}

	output["ProjectileBlockChance"] = math.Min(outNum(output, "BlockChance")+modDB.Sum("BASE", nil, "ProjectileBlockChance")*Mod(modDB, nil, "BlockChance"), outNum(output, "BlockChanceMax"))
	if modDB.Flag(nil, "SpellBlockChanceMaxIsBlockChanceMax") {
		output["SpellBlockChanceMax"] = outNum(output, "BlockChanceMax")
	} else {
		output["SpellBlockChanceMax"] = math.Min(modDB.Sum("BASE", nil, "SpellBlockChanceMax"), d.Misc.BlockChanceCap)
	}
	if modDB.Flag(nil, "MaxSpellBlockIfNotBlockedRecently") {
		output["SpellBlockChance"] = outNum(output, "SpellBlockChanceMax")
		output["SpellProjectileBlockChance"] = outNum(output, "SpellBlockChanceMax")
	} else if modDB.Flag(nil, "SpellBlockChanceIsBlockChance") {
		output["SpellBlockChance"] = outNum(output, "BlockChance")
		output["SpellProjectileBlockChance"] = outNum(output, "ProjectileBlockChance")
		output["SpellBlockChanceOverCap"] = outNum(output, "BlockChanceOverCap")
	} else {
		inc := modDB.Sum("INC", nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalSpellBlockChance := roundDec(modDB.Sum("BASE", nil, "SpellBlockChance")*(1+inc/100)*more, 0)
		output["SpellBlockChance"] = math.Min(totalSpellBlockChance, outNum(output, "SpellBlockChanceMax"))
		output["SpellBlockChanceOverCap"] = math.Max(0, totalSpellBlockChance-outNum(output, "SpellBlockChanceMax"))
		output["SpellProjectileBlockChance"] = math.Max(math.Min(outNum(output, "SpellBlockChance")+modDB.Sum("BASE", nil, "ProjectileSpellBlockChance")*Mod(modDB, nil, "SpellBlockChance"), outNum(output, "SpellBlockChanceMax")), 0)
	}
	if modDB.Flag(nil, "CannotBlockAttacks") {
		output["BlockChance"] = 0.0
		output["ProjectileBlockChance"] = 0.0
	}
	if modDB.Flag(nil, "CannotBlockSpells") {
		output["SpellBlockChance"] = 0.0
		output["SpellProjectileBlockChance"] = 0.0
	}
	for _, blockType := range []string{"BlockChance", "ProjectileBlockChance", "SpellBlockChance", "SpellProjectileBlockChance"} {
		if env.ModeEffective {
			output["Effective"+blockType] = math.Max(outNum(output, blockType)-enemyDB.Sum("BASE", nil, "reduceEnemyBlock"), 0)
		} else {
			output["Effective"+blockType] = outNum(output, blockType)
		}
		blockRolls := 0.0
		if env.ModeEffective {
			if modDB.Flag(nil, blockType+"IsLucky") {
				blockRolls++
			}
			if modDB.Flag(nil, blockType+"IsUnlucky") {
				blockRolls--
			}
			if modDB.Flag(nil, "ExtremeLuck") {
				blockRolls *= 2
			}
		}
		// unlucky config to lower the value of block, dodge, evade etc for ehp
		if worstOf := env.ConfigInput["EHPUnluckyWorstOf"]; truthy(worstOf) && anyNum(worstOf) != 1 {
			blockRolls = -anyNum(worstOf) / 2
		}
		if blockRolls != 0 {
			blockChance := outNum(output, "Effective"+blockType) / 100
			if modDB.Flag(nil, "Unexciting") {
				// Unexciting rolls three times and keeps the median result
				output["Effective"+blockType] = (3*math.Pow(blockChance, 2) - 2*math.Pow(blockChance, 3)) * 100
			} else if blockRolls > 0 {
				output["Effective"+blockType] = (1 - math.Pow(1-blockChance, blockRolls+1)) * 100
			} else {
				output["Effective"+blockType] = math.Pow(blockChance, math.Abs(blockRolls)) * outNum(output, "Effective"+blockType)
			}
		}
	}
	output["EffectiveAverageBlockChance"] = (outNum(output, "EffectiveBlockChance") + outNum(output, "EffectiveProjectileBlockChance") +
		outNum(output, "EffectiveSpellBlockChance") + outNum(output, "EffectiveSpellProjectileBlockChance")) / 4
	output["BlockEffect"] = 100 - modDB.Sum("BASE", nil, "BlockEffect")
	if outNum(output, "BlockEffect") != 0 {
		output["ShowBlockEffect"] = true
		output["DamageTakenOnBlock"] = 100 - outNum(output, "BlockEffect")
	}

	if modDB.Flag(nil, "ArmourAppliesToEnergyShieldRecharge") {
		// Armour to ES Recharge conversion from Armour and Energy Shield Mastery
		multiplier := 100.0
		if v, ok := modDB.Max(nil, "ImprovedArmourAppliesToEnergyShieldRecharge"); ok {
			multiplier = v
		}
		multiplier = multiplier / 100
		for _, value := range modDB.Tabulate("INC", nil, "Armour", "ArmourAndEvasion", "Defences") {
			mod := value.Mod
			modifiers := GetConvertedModTags(mod, multiplier, false)
			args := []any{mod.Source, mod.Flags, mod.KeywordFlags}
			args = append(args, modifiers...)
			modDB.AddMod(newMod("EnergyShieldRecharge", "INC", math.Floor(anyNum(mod.Value)*multiplier), args...))
		}
	}

	// Flag-driven defence conversions: each takes the first tabulated
	// mod's source and breaks (the reference loops only to reach [1]).
	for _, conv := range []struct{ flag, target, from string }{
		{"ArmourIncreasedByUncappedFireRes", "Armour", "FireResistTotal"},
		{"ArmourIncreasedByOvercappedFireRes", "Armour", "FireResistOverCap"},
		{"EvasionRatingIncreasedByUncappedColdRes", "Evasion", "ColdResistTotal"},
		{"EvasionRatingIncreasedByOvercappedColdRes", "Evasion", "ColdResistOverCap"},
		{"EnergyShieldIncreasedByChanceToBlockSpellDamage", "EnergyShield", "SpellBlockChance"},
		{"EnergyShieldIncreasedByChaosResistance", "EnergyShield", "ChaosResist"},
	} {
		if modDB.Flag(nil, conv.flag) {
			for _, value := range modDB.Tabulate("FLAG", nil, conv.flag) {
				modDB.AddMod(newMod(conv.target, "INC", outNum(output, conv.from), value.Mod.Source))
				break
			}
		}
	}

	env.defencePrimary(actor)
}
