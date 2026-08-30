// CalcOffence.lua L520-1022: the skill-data section — the stat-conversion
// chain (Battlemage/Spellblade, minion-damage transfer, spell/cast/claw
// conversions), the repeat handling, the random-phys split and momentum.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// maxOr is `skillModList:Max(cfg, name) or fallback`.
func maxOr(l *modstore.List, cfg *modstore.Cfg, fallback float64, names ...string) float64 {
	if v, ok := l.Max(cfg, names...); ok {
		return v
	}
	return fallback
}

// offenceSkillData ports L520-1022.
func (env *Env) offenceSkillData(c *offenceCtx) {
	actor, skillModList, skillCfg, skillData := c.actor, c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, modDB := c.skillFlags, c.output, c.modDB
	activeSkill := c.activeSkill

	if modDB.Flag(nil, "Elusive") && skillModList.Flag(nil, "SupportedByNightblade") {
		elusiveEffect := output.N("ElusiveEffectMod") / 100
		nightbladeMulti := skillModList.Sum(modparser.Base, nil, "NightbladeElusiveCritMultiplier")
		skillModList.AddMod(newModS("CritMultiplier", modparser.Base, modparser.Num(math.Floor(nightbladeMulti*elusiveEffect)), "Nightblade"))
	}

	// set other limits
	output.SetN("ActiveTrapLimit", skillModList.Sum(modparser.Base, skillCfg, "ActiveTrapLimit"))
	output.SetN("ActiveMineLimit", skillModList.Sum(modparser.Base, skillCfg, "ActiveMineLimit"))

	// set flask scaling
	if v, ok := env.ItemModDB.Multipliers["LifeFlaskRecovery"]; ok {
		output.SetN("LifeFlaskRecovery", v)
	} else {
		output.Del("LifeFlaskRecovery")
	}
	if v, ok := env.ItemModDB.Multipliers["LifeFlaskCharges"]; ok {
		output.SetN("LifeFlaskCharges", v)
	} else {
		output.Del("LifeFlaskCharges")
	}

	if modDB.Conditions.Get("AffectedByEnergyBlade") {
		dmgMod := Mod(skillModList, skillCfg, "EnergyBladeDamage")
		speedMod := Mod(skillModList, skillCfg, "EnergyBladeAttackSpeed")
		// The reference iterates a two-key table with pairs(); the two
		// branches touch disjoint weapon slots, so the order is immaterial.
		for _, pair := range [][2]string{{"Weapon 1", "1"}, {"Weapon 2", "2"}} {
			slotName, side := pair[0], pair[1]
			wd := weaponOf(actor.ms.WeaponData1)
			if side == "2" {
				wd = weaponOf(actor.ms.WeaponData2)
			}
			it, _ := actor.ms.ItemList[slotName].(*Item)
			if it == nil || it.In.WeaponData == nil || it.In.WeaponData[1] == nil || wd == nil {
				continue
			}
			if wd.Name == "" {
				continue
			}
			base := data.ItemBases[wd.Name]
			if base == nil || base.Weapon == nil {
				continue
			}
			wb := base.Weapon
			wd.CritChance = util.Some(wb.CritChanceBase)
			wd.AttackRate = wb.AttackRateBase * speedMod
			// The reference writes "Range" (not the "range" key it reads).
			wd.Set("Range", modparser.Num(wb.Range))
			for _, damageType := range dmgTypeList {
				baseMin, baseMax := 0.0, 0.0
				if damageType == "Physical" {
					baseMin, baseMax = wb.PhysicalMin, wb.PhysicalMax
				}
				r := wd.Damage(damageType)
				r.Min = baseMin + math.Floor(skillModList.Sum(modparser.Base, skillCfg, "EnergyBladeMin"+damageType)*dmgMod)
				r.Max = baseMax + math.Floor(skillModList.Sum(modparser.Base, skillCfg, "EnergyBladeMax"+damageType)*dmgMod)
			}
		}
	}

	// account for Battlemage
	// Note: we check conditions of Main Hand weapon using actor.itemList as
	// actor.weaponData1 is populated with unarmed values when no weapon slotted.
	if w1, _ := actor.ms.ItemList["Weapon 1"].(*Item); skillModList.Flag(nil, "Battlemage") &&
		w1 != nil && w1.In.WeaponData != nil && w1.In.WeaponData[1] != nil {
		multiplier := maxOr(skillModList, skillCfg, 100, "MainHandWeaponDamageAppliesToSpells") / 100
		for _, damageType := range dmgTypeList {
			skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Min*multiplier)), "Battlemage", modparser.FlagSpell, modparser.KeywordNone))
			skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Max*multiplier)), "Battlemage", modparser.FlagSpell, modparser.KeywordNone))
		}
	}
	// weapon1info/weapon2info are locals in the reference too, and used only
	// by the Spellblade block just below.
	weapon1info, hasWeapon1info := data.WeaponTypeInfo[weaponType(weaponOf(actor.ms.WeaponData1))]
	_, hasWeapon2info := data.WeaponTypeInfo[weaponType(weaponOf(actor.ms.WeaponData2))]

	// account for Spellblade
	if spellbladeMulti, ok := skillModList.Max(skillCfg, "OneHandWeaponDamageAppliesToSpells"); ok {
		w1, _ := actor.ms.ItemList["Weapon 1"].(*Item)
		if w1 != nil && w1.In.WeaponData != nil && w1.In.WeaponData[1] != nil &&
			hasWeapon1info && weapon1info.Melee && weapon1info.OneHand {
			second := 1.0
			if hasWeapon2info {
				second = 0.6
			}
			multiplier := spellbladeMulti / 100 * second
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Min*multiplier)), "Spellblade Main Hand", modparser.FlagSpell, modparser.KeywordNone))
				skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Max*multiplier)), "Spellblade Main Hand", modparser.FlagSpell, modparser.KeywordNone))
			}
			if hasWeapon2info {
				for _, damageType := range dmgTypeList {
					skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Min*multiplier)), "Spellblade Off Hand", modparser.FlagSpell, modparser.KeywordNone))
					skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Max*multiplier)), "Spellblade Off Hand", modparser.FlagSpell, modparser.KeywordNone))
				}
			}
		}
	}
	if skillModList.Flag(nil, "MinionDamageAppliesToPlayer") || skillModList.Flag(skillCfg, "MinionDamageAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Aid and The Scourge
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionDamageAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			mod := modRefOf(value)
			if mod != nil && mod.Name == "Damage" && mod.Type == modparser.Inc {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				skillModList.AddMod(modparser.NewModFull("Damage", modparser.Inc, modparser.Num(valueNum(mod.Value)*multiplier), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "MinionAttackSpeedAppliesToPlayer") {
		// Minion Attack Speed conversion from Spiritual Command
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionAttackSpeedAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			mod := modRefOf(value)
			if mod != nil && mod.Name == "Speed" && mod.Type == modparser.Inc && (mod.Flags == 0 || mod.Flags&modparser.FlagAttack != 0) {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				skillModList.AddMod(modparser.NewModFull("Speed", modparser.Inc, modparser.Num(valueNum(mod.Value)*multiplier), mod.Source, mod.SourceSet, modparser.FlagAttack, mod.KeywordFlags, modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "MinionCastSpeedAppliesToPlayer") {
		// Minion Cast Speed conversion from Spinehail
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionCastSpeedAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			mod := modRefOf(value)
			if mod != nil && mod.Name == "Speed" && mod.Type == modparser.Inc && (mod.Flags == 0 || mod.Flags&modparser.FlagCast != 0) {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				skillModList.AddMod(modparser.NewModFull("Speed", modparser.Inc, modparser.Num(valueNum(mod.Value)*multiplier), mod.Source, mod.SourceSet, modparser.FlagCast, mod.KeywordFlags, modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "EvasionAppliesToSpellDamage") {
		// The Unblinking Eye evasion rating to spell damage conversion
		// Must run before SpellDamageAppliesToAttacks so the generated spell
		// mods can chain into attacks
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{}, "Evasion") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("Damage", mod.Type, mod.Value, mod.Source, mod.SourceSet, modparser.FlagSpell, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillModList.Flag(nil, "SpellDamageAppliesToAttacks") || skillModList.Flag(skillCfg, "SpellDamageAppliesToAttacks") {
		// Spell Damage conversion from Crown of Eyes, Kinetic Bolt, and the
		// Wandslinger notable
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedSpellDamageAppliesToAttacks") / 100
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{Flags: flagp(modparser.FlagSpell)}, "Damage") {
			mod := value.Mod
			if mod.Flags&modparser.FlagSpell != 0 {
				modifiers := GetConvertedModTags(mod, multiplier, false)
				skillModList.AddMod(modparser.NewModFull("Damage", modparser.Inc, modparser.Num(valueNum(mod.Value)*multiplier), mod.Source, mod.SourceSet, (mod.Flags&^modparser.FlagSpell)|modparser.FlagAttack, mod.KeywordFlags, modifiers...))
				if mod.Source == "Strength" { // Prevent double-dipping from converted strength's damage bonus
					skillModList.ReplaceMod(newModSF("PhysicalDamage", modparser.Inc, modparser.Num(0.0), "Strength", modparser.FlagMelee, modparser.KeywordNone))
				}
			}
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToAttacks") {
		// Get all increases for this; assumption is that multiple sources
		// would not stack, so find the max
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedCastSpeedAppliesToAttacks") / 100
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{Flags: flagp(modparser.FlagCast)}, "Speed") {
			mod := value.Mod
			// Add a new mod for all mods that are cast only
			if mod.Flags&modparser.FlagCast != 0 {
				modifiers := GetConvertedModTags(mod, multiplier, false)
				skillModList.AddMod(modparser.NewModFull("Speed", modparser.Inc, modparser.Num(valueNum(mod.Value)*multiplier), mod.Source, mod.SourceSet, (mod.Flags&^modparser.FlagCast)|modparser.FlagAttack, mod.KeywordFlags, modifiers...))
			}
		}
	}
	if skillModList.Flag(nil, "ProjectileSpeedAppliesToBowDamage") {
		// Bow mastery projectile speed to damage with bows conversion
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{}, "ProjectileSpeed") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("Damage", mod.Type, mod.Value, mod.Source, mod.SourceSet, modparser.FlagBow|modparser.FlagHit, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillModList.Flag(nil, "ClawDamageAppliesToUnarmed") {
		// Claw Damage conversion from Rigwald's Curse
		cfg := &modstore.Cfg{Flags: flagp(modparser.FlagClaw | modparser.FlagHit), KeywordFlags: keywordp(modparser.KeywordHit)}
		for _, value := range skillModList.Tabulate(modparser.Inc, cfg, "Damage") {
			mod := value.Mod
			if mod.Flags&modparser.FlagClaw != 0 {
				skillModList.AddMod(modparser.NewModFull("Damage", mod.Type, mod.Value, mod.Source, mod.SourceSet, (mod.Flags&^modparser.FlagClaw)|modparser.FlagUnarmed|modparser.FlagMelee, mod.KeywordFlags, mod.Tags...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawAttackSpeedAppliesToUnarmed") {
		// Claw Attack Speed conversion from Rigwald's Curse
		cfg := &modstore.Cfg{Flags: flagp(modparser.FlagClaw | modparser.FlagAttack | modparser.FlagHit)}
		for _, value := range skillModList.Tabulate(modparser.Inc, cfg, "Speed") {
			mod := value.Mod
			if mod.Flags&modparser.FlagClaw != 0 && mod.Flags&modparser.FlagAttack != 0 {
				skillModList.AddMod(modparser.NewModFull("Speed", mod.Type, mod.Value, mod.Source, mod.SourceSet, (mod.Flags&^modparser.FlagClaw)|modparser.FlagUnarmed, mod.KeywordFlags, mod.Tags...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToUnarmed") {
		// Claw Crit Chance conversion from Rigwald's Curse
		cfg := &modstore.Cfg{Flags: flagp(modparser.FlagClaw | modparser.FlagHit)}
		for _, value := range skillModList.Tabulate(modparser.Inc, cfg, "CritChance") {
			mod := value.Mod
			if mod.Flags&modparser.FlagClaw != 0 {
				skillModList.AddMod(modparser.NewModFull("CritChance", mod.Type, mod.Value, mod.Source, mod.SourceSet, (mod.Flags&^modparser.FlagClaw)|modparser.FlagUnarmed, mod.KeywordFlags, mod.Tags...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToMinions") {
		// Claw Crit Chance conversion from Law of the Wilds
		cfg := &modstore.Cfg{Flags: flagp(modparser.FlagClaw | modparser.FlagHit)}
		for _, value := range skillModList.Tabulate(modparser.Inc, cfg, "CritChance") {
			mod := value.Mod
			if mod.Flags&modparser.FlagClaw != 0 {
				env.Minion.DB.AddMod(newModS("CritChance", mod.Type, mod.Value, mod.Source))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritMultiplierAppliesToMinions") {
		// Claw Crit Multi conversion from Law of the Wilds
		cfg := &modstore.Cfg{Flags: flagp(modparser.FlagClaw | modparser.FlagHit)}
		for _, value := range skillModList.Tabulate(modparser.Base, cfg, "CritMultiplier") {
			mod := value.Mod
			if mod.Flags&modparser.FlagClaw != 0 {
				env.Minion.DB.AddMod(newModS("CritMultiplier", mod.Type, mod.Value, mod.Source))
			}
		}
	}
	// The four resistance-driven crit/pen transfers each take only the first
	// tabulated mod (the reference breaks out of the loop).
	firstFlagSource := func(name string) (string, bool) {
		for _, value := range modDB.Tabulate(modparser.Flag, nil, name) {
			return value.Mod.Source, true
		}
		return "", false
	}
	if skillModList.Flag(nil, "CritChanceIncreasedByUncappedLightningRes") {
		if src, ok := firstFlagSource("CritChanceIncreasedByUncappedLightningRes"); ok {
			skillModList.AddMod(newModS("CritChance", modparser.Inc, modparser.Num(output.N("LightningResistTotal")), src))
		}
	}
	if skillModList.Flag(nil, "CritChanceIncreasedByLightningRes") {
		if src, ok := firstFlagSource("CritChanceIncreasedByLightningRes"); ok {
			skillModList.AddMod(newModS("CritChance", modparser.Inc, modparser.Num(output.N("LightningResist")), src))
		}
	}
	if skillModList.Flag(nil, "CritChanceIncreasedByOvercappedLightningRes") {
		if src, ok := firstFlagSource("CritChanceIncreasedByOvercappedLightningRes"); ok {
			skillModList.AddMod(newModS("CritChance", modparser.Inc, modparser.Num(output.N("LightningResistOverCap")), src))
		}
	}
	if skillModList.Flag(nil, "CritChanceIncreasedBySpellSuppressChance") {
		if src, ok := firstFlagSource("CritChanceIncreasedBySpellSuppressChance"); ok {
			skillModList.AddMod(newModS("CritChance", modparser.Inc, modparser.Num(output.N("SpellSuppressionChance")), src))
		}
	}
	if skillModList.Flag(nil, "FirePenIncreasedByUncappedFireRes") {
		if src, ok := firstFlagSource("FirePenIncreasedByUncappedFireRes"); ok {
			skillModList.AddMod(newModS("FirePenetration", modparser.Base, modparser.Num(output.N("FireResistOverCap")), src))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAccuracy") {
		// Light Radius conversion from Corona Solaris
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{}, "LightRadius") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("Accuracy", modparser.Inc, mod.Value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAreaOfEffect") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{}, "LightRadius") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("AreaOfEffect", modparser.Inc, modparser.Num(math.Floor(valueNum(mod.Value)/2)), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToDamage") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{}, "LightRadius") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("Damage", modparser.Inc, mod.Value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToTrapThrowingSpeed") {
		// Cast Speed conversion from Slavedriver's Hand
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{Flags: flagp(modparser.FlagCast)}, "Speed") {
			mod := value.Mod
			if mod.Flags == 0 || mod.Flags&modparser.FlagCast != 0 {
				skillModList.AddMod(modparser.NewModFull("TrapThrowingSpeed", modparser.Inc, mod.Value, mod.Source, mod.SourceSet, mod.Flags&^modparser.FlagCast&^modparser.FlagAttack, mod.KeywordFlags, mod.Tags...))
			}
		}
	}
	if skillData.Flag("arrowSpeedAppliesToAreaOfEffect") {
		// Arrow Speed conversion for Galvanic Arrow
		for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{Flags: flagp(modparser.FlagBow)}, "ProjectileSpeed") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("AreaOfEffect", modparser.Inc, mod.Value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "SingleProjectile") &&
		!skillModList.Flag(nil, "TriggeredBySnipe") {
		// Applies DPS multiplier based on projectile count
		skillData.SetN("dpsMultiplier", skillModList.Sum(modparser.Base, skillCfg, "ProjectileCount"))
	}

	env.offenceRepeats(c)

	if skillModList.Flag(nil, "WeaponPhysAppliesToSpells") {
		// Phys from weapon to Spells from Runegraft
		// #EVAL: `Sum(...) or 100` — Sum always returns a number, so the
		// fallback is dead and an absent mod means a 0% multiplier.
		mult := skillModList.Sum(modparser.Base, skillCfg, "WeaponPhysAppliesToSpellsPercent") / 100
		if weaponOf(actor.ms.WeaponData1) != nil {
			skillModList.AddMod(newModSF("PhysicalMin", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), "Physical").Min*mult)), "Runegraft of the Spellbound", modparser.FlagSpell, modparser.KeywordNone))
			skillModList.AddMod(newModSF("PhysicalMax", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), "Physical").Max*mult)), "Runegraft of the Spellbound", modparser.FlagSpell, modparser.KeywordNone))
		}
	}
	if skillData.Flag("gainPercentBaseWandDamageToSpells") {
		mult := skillData.N("gainPercentBaseWandDamageToSpells") / 100
		w1Wand := weaponType(weaponOf(actor.ms.WeaponData1)) == "Wand"
		w2Wand := weaponType(weaponOf(actor.ms.WeaponData2)) == "Wand"
		switch {
		case w1Wand && w2Wand:
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor((dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Min+dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Min)/2*mult)), "Spellslinger", modparser.FlagSpell, modparser.KeywordNone))
				skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor((dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Max+dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Max)/2*mult)), "Spellslinger", modparser.FlagSpell, modparser.KeywordNone))
			}
		case w1Wand:
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Min*mult)), "Spellslinger", modparser.FlagSpell, modparser.KeywordNone))
				skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Max*mult)), "Spellslinger", modparser.FlagSpell, modparser.KeywordNone))
			}
		case w2Wand:
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Min*mult)), "Spellslinger", modparser.FlagSpell, modparser.KeywordNone))
				skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor(dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Max*mult)), "Spellslinger", modparser.FlagSpell, modparser.KeywordNone))
			}
		}
	}
	if skillData.Flag("gainPercentBaseDaggerDamageToSpells") {
		weapon1IsDagger := weaponAdded(weaponOf(actor.ms.WeaponData1), "Dagger") || weaponType(weaponOf(actor.ms.WeaponData1)) == "Dagger"
		weapon2IsDagger := weaponAdded(weaponOf(actor.ms.WeaponData2), "Dagger") || weaponType(weaponOf(actor.ms.WeaponData2)) == "Dagger"
		both := 1.0
		if weapon1IsDagger && weapon2IsDagger {
			both = 0.5
		}
		mult := skillData.N("gainPercentBaseDaggerDamageToSpells") / 100 * both
		for _, damageType := range dmgTypeList {
			baseMin, baseMax := 0.0, 0.0
			if weapon1IsDagger {
				baseMin += dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Min
				baseMax += dmgOf(weaponOf(actor.ms.WeaponData1), damageType).Max
			}
			if weapon2IsDagger {
				baseMin += dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Min
				baseMax += dmgOf(weaponOf(actor.ms.WeaponData2), damageType).Max
			}
			skillModList.AddMod(newModSF(damageType+"Min", modparser.Base, modparser.Num(math.Floor(baseMin*mult)), "Blade Blast of Dagger Detonation", modparser.FlagSpell, modparser.KeywordNone))
			skillModList.AddMod(newModSF(damageType+"Max", modparser.Base, modparser.Num(math.Floor(baseMax*mult)), "Blade Blast of Dagger Detonation", modparser.FlagSpell, modparser.KeywordNone))
		}
	}

	if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[modparser.SkillTypeCanRapidFire] && !skillModList.Flag(nil, "NoRepeatBonuses") {
		// Applies DPS multiplier based on seals count
		output.SetN("SealCooldown", skillModList.Sum(modparser.Base, skillCfg, "SealGainFrequency")/Mod(skillModList, skillCfg, "SealGainFrequency"))
		output.SetN("SealMax", skillModList.Sum(modparser.Base, skillCfg, "SealCount"))
		output.Set("AverageBurstHits", output.Get("SealMax"))
		output.SetN("TimeMaxSeals", output.N("SealCooldown")*output.N("SealMax"))

		if !skillData.Flag("hitTimeOverride") {
			castTime := 0.0
			if ct := activeSkill.ActiveEffect.GrantedEffect.CastTime; ct != nil {
				castTime = *ct
			}
			if skillModList.Flag(nil, "UseMaxUnleash") {
				for _, value := range skillModList.Tabulate(modparser.Inc, &modstore.Cfg{}, "MaxSealCrit") {
					mod := value.Mod
					skillModList.AddMod(modparser.NewModFull("CritChance", modparser.Inc, mod.Value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
				}
				for _, value := range skillModList.Tabulate(modparser.More, &modstore.Cfg{}, "MaxSealDamage") {
					mod := value.Mod
					skillModList.AddMod(modparser.NewModFull("Damage", modparser.More, modparser.Num(valueNum(mod.Value)*output.N("SealMax")), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
				}
				env.PlayerMainSkill.SkillData.SetN("dpsMultiplier", 1+output.N("SealMax")*Mod(skillModList, skillCfg, "SealRepeatPenalty"))
				env.PlayerMainSkill.SkillData.SetN("hitTimeOverride", math.Max(output.N("TimeMaxSeals"),
					1/castTime*1.1*Mod(skillModList, skillCfg, "Speed")*output.N("ActionSpeedMod")))
			} else {
				env.PlayerMainSkill.SkillData.SetN("dpsMultiplier", 1+1/output.N("SealCooldown")/
					(1/castTime*1.1*Mod(skillModList, skillCfg, "Speed")*output.N("ActionSpeedMod"))*
					Mod(skillModList, skillCfg, "SealRepeatPenalty"))
			}
		}
	}

	physMode := "AVERAGE"
	if v := env.ConfigInput.PhysMode; v != "" {
		physMode = v
	}
	processedRandomMods := map[*modparser.Mod]bool{}
	for _, cfg := range []*modstore.Cfg{skillCfg, activeSkill.Weapon1Cfg, activeSkill.Weapon2Cfg} {
		if cfg == nil || skillModList.Sum(modparser.Base, cfg, "PhysicalDamageGainAsRandom", "PhysicalDamageConvertToRandom", "PhysicalDamageGainAsColdOrLightning") <= 0 {
			continue
		}
		skillFlags["randomPhys"] = true
		for _, value := range skillModList.Tabulate(modparser.Base, cfg, "PhysicalDamageGainAsRandom") {
			mod := value.Mod
			if processedRandomMods[mod] {
				continue
			}
			processedRandomMods[mod] = true
			effVal := valueNum(mod.Value) / 3
			mk := func(name string, value modparser.Value) *modparser.Mod {
				return modparser.NewModFull(name, modparser.Base, value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...)
			}
			switch physMode {
			case "AVERAGE":
				skillModList.AddMod(mk("PhysicalDamageGainAsFire", modparser.Num(effVal)))
				skillModList.AddMod(mk("PhysicalDamageGainAsCold", modparser.Num(effVal)))
				skillModList.AddMod(mk("PhysicalDamageGainAsLightning", modparser.Num(effVal)))
			case "FIRE":
				skillModList.AddMod(mk("PhysicalDamageGainAsFire", mod.Value))
			case "COLD":
				skillModList.AddMod(mk("PhysicalDamageGainAsCold", mod.Value))
			case "LIGHTNING":
				skillModList.AddMod(mk("PhysicalDamageGainAsLightning", mod.Value))
			}
		}
		for _, value := range skillModList.Tabulate(modparser.Base, cfg, "PhysicalDamageConvertToRandom") {
			mod := value.Mod
			if processedRandomMods[mod] {
				continue
			}
			processedRandomMods[mod] = true
			effVal := valueNum(mod.Value) / 3
			mk := func(name string, value modparser.Value) *modparser.Mod {
				return modparser.NewModFull(name, modparser.Base, value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...)
			}
			switch physMode {
			case "AVERAGE":
				skillModList.AddMod(mk("PhysicalDamageConvertToFire", modparser.Num(effVal)))
				skillModList.AddMod(mk("PhysicalDamageConvertToCold", modparser.Num(effVal)))
				skillModList.AddMod(mk("PhysicalDamageConvertToLightning", modparser.Num(effVal)))
			case "FIRE":
				skillModList.AddMod(mk("PhysicalDamageConvertToFire", mod.Value))
			case "COLD":
				skillModList.AddMod(mk("PhysicalDamageConvertToCold", mod.Value))
			case "LIGHTNING":
				skillModList.AddMod(mk("PhysicalDamageConvertToLightning", mod.Value))
			}
		}
		for _, value := range skillModList.Tabulate(modparser.Base, cfg, "PhysicalDamageGainAsColdOrLightning") {
			mod := value.Mod
			if processedRandomMods[mod] {
				continue
			}
			processedRandomMods[mod] = true
			effVal := valueNum(mod.Value) / 2
			mk := func(name string, value modparser.Value) *modparser.Mod {
				return modparser.NewModFull(name, modparser.Base, value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...)
			}
			switch physMode {
			case "AVERAGE", "FIRE":
				skillModList.AddMod(mk("PhysicalDamageGainAsCold", modparser.Num(effVal)))
				skillModList.AddMod(mk("PhysicalDamageGainAsLightning", modparser.Num(effVal)))
			case "COLD":
				skillModList.AddMod(mk("PhysicalDamageGainAsCold", mod.Value))
			case "LIGHTNING":
				skillModList.AddMod(mk("PhysicalDamageGainAsLightning", mod.Value))
			}
		}
	}
	// momentum stacks
	if skillModList.Flag(nil, "SupportedByMomentum") {
		maxMomentumStacks := skillModList.Sum(modparser.Base, skillCfg, "MomentumStacksMax")
		extraMomentumStacks := skillModList.Sum(modparser.Base, skillCfg, "MomentumStacksExtra")
		combat := &modparser.CondTag{Var: "Combat"}
		if maxMomentumStacks > 0 {
			if !modDB.HasMod(modparser.Base, nil, "Multiplier:MomentumStacks") {
				modDB.AddMod(newModS("Multiplier:MomentumStacks", modparser.Base, modparser.Num(math.Min((maxMomentumStacks+extraMomentumStacks)/2, maxMomentumStacks)), "Config", combat))
			} else if modDB.Sum(modparser.Base, nil, "Multiplier:MomentumStacks") > maxMomentumStacks {
				modDB.ReplaceMod(newModS("Multiplier:MomentumStacks", modparser.Base, modparser.Num(maxMomentumStacks), "Config", combat))
			}
		} else if modDB.HasMod(modparser.Base, nil, "Multiplier:MomentumStacks") {
			modDB.ReplaceMod(newModS("Multiplier:MomentumStacks", modparser.Base, modparser.Num(0.0), "Config"))
		}
	}
}

// repeatSkillTypesCheck ports the local of the same name (L775).
func repeatSkillTypesCheck(skillModList *modstore.List, activeSkillTypes map[modparser.SkillTypeID]bool) bool {
	excludeSkillTypes := []modparser.SkillTypeID{
		modparser.SkillTypeInstant, modparser.SkillTypeChannel, modparser.SkillTypeTriggered,
		modparser.SkillTypeRetaliation, modparser.SkillTypeNonRepeatable,
	}
	for _, typ := range excludeSkillTypes {
		if activeSkillTypes[typ] {
			return false
		}
	}
	return !skillModList.Flag(nil, "CannotRepeat") &&
		(activeSkillTypes[modparser.SkillTypeAttack] || activeSkillTypes[modparser.SkillTypeSpell])
}

// offenceRepeats ports L783-869: output.Repeats and everything the repeat
// count multiplies.
func (env *Env) offenceRepeats(c *offenceCtx) {
	skillModList, skillCfg, output := c.skillModList, c.skillCfg, c.output
	skillFlags, activeSkill := c.skillFlags, c.activeSkill

	repeats := 1.0
	if repeatSkillTypesCheck(skillModList, activeSkill.SkillTypes) {
		repeats += skillModList.Sum(modparser.Base, skillCfg, "RepeatCount")
	}
	output.SetN("Repeats", repeats)
	if repeats <= 1 {
		return
	}
	output.SetN("RepeatCount", repeats)
	// handle all the multipliers from Repeats
	repeatMode := env.ConfigInput.RepeatMode
	if repeatMode == "NONE" {
		return
	}
	average := repeatMode == "AVERAGE"

	for _, value := range skillModList.Tabulate(modparser.Inc, skillCfg, "RepeatFinalAreaOfEffect") {
		mod := value.Mod
		modValue := valueNum(mod.Value)
		if average {
			modValue /= repeats
		}
		skillModList.AddMod(modparser.NewModFull("AreaOfEffect", modparser.Inc, modparser.Num(modValue), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
	}
	for _, value := range skillModList.Tabulate(modparser.Inc, skillCfg, "RepeatPerRepeatAreaOfEffect") {
		mod := value.Mod
		modValue := valueNum(mod.Value) * (repeats - 1)
		if average {
			modValue /= 2
		}
		skillModList.AddMod(modparser.NewModFull("AreaOfEffect", modparser.Inc, modparser.Num(modValue), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
	}
	for _, value := range skillModList.Tabulate(modparser.Base, skillCfg, "RepeatFinalDoubleDamageChance") {
		mod := value.Mod
		modValue := valueNum(mod.Value)
		if average {
			modValue /= repeats
		}
		skillModList.AddMod(modparser.NewModFull("DoubleDamageChance", modparser.Base, modparser.Num(modValue), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
	}
	damageFinalMoreValueTotal := 1.0
	damageMoreValueTotal := 0.0
	for _, value := range skillModList.Tabulate(modparser.More, skillCfg, "RepeatFinalDamage") {
		mod := value.Mod
		modValue := valueNum(mod.Value)
		damageFinalMoreValueTotal *= 1 + modValue/100
		damageMoreValueTotal += modValue
		if average && !skillModList.Flag(nil, "OnlyFinalRepeat") {
			modValue /= repeats
		}
		skillModList.AddMod(modparser.NewModFull("Damage", modparser.More, modparser.Num(modValue), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
	}
	for _, value := range skillModList.Tabulate(modparser.More, skillCfg, "RepeatPerRepeatDamage") {
		mod := value.Mod
		modValue := valueNum(mod.Value) * (repeats - 1)
		if average {
			if damageFinalMoreValueTotal != 1 {
				// sum from 0 to num Repeats the damage each one does,
				// multiplied by the other repeat multipliers, divide the
				// total by the average other repeat multipliers and divide
				// by number of repeats
				modValue = ((100+valueNum(mod.Value)*(repeats-2)/2)*(repeats-1)+
					(100+valueNum(mod.Value)*(repeats-1))*damageFinalMoreValueTotal)/
					(repeats+damageMoreValueTotal/100) - 100
			} else {
				modValue /= 2
			}
		}
		skillModList.AddMod(modparser.NewModFull("Damage", modparser.More, modparser.Num(modValue), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
	}

	var lastMod *modparser.Mod
	damageFinalMoreValueTotal = damageMoreValueTotal
	for _, repeatCount := range []struct {
		n    float64
		name string
	}{{2, "One"}, {3, "Two"}, {4, "Three"}} {
		if repeatCount.n > repeats {
			break
		} else if average {
			for _, value := range skillModList.Tabulate(modparser.More, skillCfg, "Repeat"+repeatCount.name+"Damage") {
				damageMoreValueTotal += valueNum(value.Mod.Value)
				lastMod = value.Mod
			}
		} else if repeatCount.n == repeats {
			for _, value := range skillModList.Tabulate(modparser.More, skillCfg, "Repeat"+repeatCount.name+"Damage") {
				mod := value.Mod
				skillModList.AddMod(modparser.NewModFull("Damage", modparser.More, mod.Value, mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
			}
		}
	}
	if average && lastMod != nil {
		skillModList.AddMod(modparser.NewModFull("Damage", modparser.More, modparser.Num((damageMoreValueTotal/repeats+100)/(1+damageFinalMoreValueTotal/repeats/100)-100), lastMod.Source, lastMod.SourceSet, lastMod.Flags, lastMod.KeywordFlags, lastMod.Tags...))
	}
	if skillModList.Flag(nil, "FinalRepeatSumsDamage") {
		for _, value := range skillModList.Tabulate(modparser.Flag, skillCfg, "FinalRepeatSumsDamage") {
			mod := value.Mod
			skillModList.AddMod(modparser.NewModFull("Damage", modparser.More, modparser.Num((100*repeats+damageFinalMoreValueTotal)/(1+damageFinalMoreValueTotal/100)-100), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
		}
	}
	if skillFlags["trap"] || skillFlags["mine"] {
		skillModList.AddMod(newModS("DPS", modparser.More, modparser.Num((repeats-1)*100), "Repeat Count"))
	}
}
