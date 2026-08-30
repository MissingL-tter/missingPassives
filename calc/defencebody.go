// CalcDefence.lua L636-1634: calcs.defence. Split from defence.go (which
// holds resistances and the shared helpers) to keep the files readable.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"
)

// hitChance ports calcs.hitChance.
func hitChance(evasion, accuracy float64) float64 {
	if accuracy < 0 {
		return 5
	}
	rawChance := accuracy / (accuracy + math.Pow(evasion/5, 0.9)) * 125
	return math.Max(math.Min(util.RoundHalfUp(rawChance, 0), 100), 5)
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
	return util.RoundHalfUp(armourReductionF(armour, raw), 0)
}

// armourDataOf reads one defence stat of a slot's armourData, or 0.
func armourDataOf(it *Item, stat string) float64 {
	if it == nil || it.In == nil || it.In.ArmourData == nil {
		return 0
	}
	return it.In.ArmourData.Defence(stat).Value.Or(0)
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

	// Action Speed
	output.SetN("ActionSpeedMod", env.actionSpeedMod(actor))

	env.Resistances(actor)
	if env.Minion != nil && modDB.Sum(modparser.Base, nil, "ResistanceAddedToMinions") > 0 {
		for _, elem := range resistTypeList {
			final := output.N(elem + "Resist")
			env.Minion.DB.AddMod(newModS(elem+"Resist", modparser.Base, modparser.Num(math.Floor(final*modDB.Sum(modparser.Base, nil, "ResistanceAddedToMinions")/100)), "Player"))
		}
	}
	// Formless Inferno
	if actor == env.minionPA {
		env.doActorLifeMana(actor)
		env.doActorLifeManaReservation(actor, false)
	}

	// Block
	output.SetN("BlockChanceMax", math.Min(modDB.Sum(modparser.Base, nil, "BlockChanceMax"), data.Misc.BlockChanceCap))
	if modDB.Flag(nil, "MaximumBlockAttackChanceIsEqualToParent") {
		output.SetN("BlockChanceMax", actor.parent.output.N("BlockChanceMax"))
	} else if modDB.Flag(nil, "MaximumBlockAttackChanceIsEqualToPartyMember") {
		panic("defence: MaximumBlockAttackChanceIsEqualToPartyMember needs the party tab")
	}
	output.SetN("BlockChanceOverCap", 0.0)
	output.SetN("SpellBlockChanceOverCap", 0.0)
	baseBlockChance := 0.0
	for _, slot := range []string{"Weapon 2", "Weapon 3"} {
		it, _ := actor.ms.ItemList[slot].(*Item)
		if it != nil && it.In.ArmourData != nil {
			baseBlockChance += it.In.ArmourData.BlockChance.Or(0)
		}
	}
	output.SetN("ShieldBlockChance", baseBlockChance)
	if !env.Keystone.KeystonesAdded["Necromantic Aegis"] {
		if ov, ok := modDB.Override(nil, "ReplaceShieldBlock"); ok {
			baseBlockChance = valueNum(ov)
		}
	}
	// Apply player block overrides if Necromantic Aegis allocated
	if actor == env.minionPA && env.Keystone.KeystonesAdded["Necromantic Aegis"] {
		if ov, ok := env.ModDB.Override(nil, "ReplaceShieldBlock"); ok {
			baseBlockChance = valueNum(ov)
		}
	}

	if modDB.Flag(nil, "BlockAttackChanceIsEqualToParent") {
		output.SetN("BlockChance", math.Min(actor.parent.output.N("BlockChance"), output.N("BlockChanceMax")))
	} else if modDB.Flag(nil, "BlockAttackChanceIsEqualToPartyMember") {
		panic("defence: BlockAttackChanceIsEqualToPartyMember needs the party tab")
	} else if modDB.Flag(nil, "MaxBlockIfNotBlockedRecently") {
		output.SetN("BlockChance", output.N("BlockChanceMax"))
	} else {
		inc := modDB.Sum(modparser.Inc, nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalBlockChance := util.RoundHalfUp((baseBlockChance+modDB.Sum(modparser.Base, nil, "BlockChance"))*(1+inc/100)*more, 0)
		output.SetN("BlockChance", math.Min(totalBlockChance, output.N("BlockChanceMax")))
		output.SetN("BlockChanceOverCap", math.Max(0, totalBlockChance-output.N("BlockChanceMax")))
	}

	output.SetN("ProjectileBlockChance", math.Min(output.N("BlockChance")+modDB.Sum(modparser.Base, nil, "ProjectileBlockChance")*Mod(modDB, nil, "BlockChance"), output.N("BlockChanceMax")))
	if modDB.Flag(nil, "SpellBlockChanceMaxIsBlockChanceMax") {
		output.SetN("SpellBlockChanceMax", output.N("BlockChanceMax"))
	} else {
		output.SetN("SpellBlockChanceMax", math.Min(modDB.Sum(modparser.Base, nil, "SpellBlockChanceMax"), data.Misc.BlockChanceCap))
	}
	if modDB.Flag(nil, "MaxSpellBlockIfNotBlockedRecently") {
		output.SetN("SpellBlockChance", output.N("SpellBlockChanceMax"))
		output.SetN("SpellProjectileBlockChance", output.N("SpellBlockChanceMax"))
	} else if modDB.Flag(nil, "SpellBlockChanceIsBlockChance") {
		output.SetN("SpellBlockChance", output.N("BlockChance"))
		output.SetN("SpellProjectileBlockChance", output.N("ProjectileBlockChance"))
		output.SetN("SpellBlockChanceOverCap", output.N("BlockChanceOverCap"))
	} else {
		inc := modDB.Sum(modparser.Inc, nil, "BlockChance")
		more := modDB.More(nil, "BlockChance")
		totalSpellBlockChance := util.RoundHalfUp(modDB.Sum(modparser.Base, nil, "SpellBlockChance")*(1+inc/100)*more, 0)
		output.SetN("SpellBlockChance", math.Min(totalSpellBlockChance, output.N("SpellBlockChanceMax")))
		output.SetN("SpellBlockChanceOverCap", math.Max(0, totalSpellBlockChance-output.N("SpellBlockChanceMax")))
		output.SetN("SpellProjectileBlockChance", math.Max(math.Min(output.N("SpellBlockChance")+modDB.Sum(modparser.Base, nil, "ProjectileSpellBlockChance")*Mod(modDB, nil, "SpellBlockChance"), output.N("SpellBlockChanceMax")), 0))
	}
	if modDB.Flag(nil, "CannotBlockAttacks") {
		output.SetN("BlockChance", 0.0)
		output.SetN("ProjectileBlockChance", 0.0)
	}
	if modDB.Flag(nil, "CannotBlockSpells") {
		output.SetN("SpellBlockChance", 0.0)
		output.SetN("SpellProjectileBlockChance", 0.0)
	}
	for _, blockType := range []string{"BlockChance", "ProjectileBlockChance", "SpellBlockChance", "SpellProjectileBlockChance"} {
		if env.ModeEffective {
			output.SetN("Effective"+blockType, math.Max(output.N(blockType)-enemyDB.Sum(modparser.Base, nil, "reduceEnemyBlock"), 0))
		} else {
			output.SetN("Effective"+blockType, output.N(blockType))
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
		if worstOf := env.ConfigInput.EHPUnluckyWorstOf; worstOf.Set && worstOf.V != 1 {
			blockRolls = -worstOf.V / 2
		}
		if blockRolls != 0 {
			blockChance := output.N("Effective"+blockType) / 100
			if modDB.Flag(nil, "Unexciting") {
				// Unexciting rolls three times and keeps the median result
				output.SetN("Effective"+blockType, (3*math.Pow(blockChance, 2)-2*math.Pow(blockChance, 3))*100)
			} else if blockRolls > 0 {
				output.SetN("Effective"+blockType, (1-math.Pow(1-blockChance, blockRolls+1))*100)
			} else {
				output.SetN("Effective"+blockType, math.Pow(blockChance, math.Abs(blockRolls))*output.N("Effective"+blockType))
			}
		}
	}
	output.SetN("EffectiveAverageBlockChance", (output.N("EffectiveBlockChance")+output.N("EffectiveProjectileBlockChance")+
		output.N("EffectiveSpellBlockChance")+output.N("EffectiveSpellProjectileBlockChance"))/4)
	output.SetN("BlockEffect", 100-modDB.Sum(modparser.Base, nil, "BlockEffect"))
	if output.N("BlockEffect") != 0 {
		output.SetFlag("ShowBlockEffect", true)
		output.SetN("DamageTakenOnBlock", 100-output.N("BlockEffect"))
	}

	if modDB.Flag(nil, "ArmourAppliesToEnergyShieldRecharge") {
		// Armour to ES Recharge conversion from Armour and Energy Shield Mastery
		multiplier := 100.0
		if v, ok := modDB.Max(nil, "ImprovedArmourAppliesToEnergyShieldRecharge"); ok {
			multiplier = v
		}
		multiplier = multiplier / 100
		for _, value := range modDB.Tabulate(modparser.Inc, nil, "Armour", "ArmourAndEvasion", "Defences") {
			mod := value.Mod
			modifiers := GetConvertedModTags(mod, multiplier, false)
			modDB.AddMod(modparser.NewModFull("EnergyShieldRecharge", modparser.Inc, modparser.Num(math.Floor(valueNum(mod.Value)*multiplier)), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, modifiers...))
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
			for _, value := range modDB.Tabulate(modparser.Flag, nil, conv.flag) {
				modDB.AddMod(newModS(conv.target, modparser.Inc, modparser.Num(output.N(conv.from)), value.Mod.Source))
				break
			}
		}
	}

	env.defencePrimary(actor)
}
