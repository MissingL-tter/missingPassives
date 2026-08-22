// CalcOffence.lua L520-1022: the skill-data section — the stat-conversion
// chain (Battlemage/Spellblade, minion-damage transfer, spell/cast/claw
// conversions), the repeat handling, the random-phys split and momentum.
package calc

import (
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// modArgs builds the trailing NewMod arguments the reference writes as
// `mod.source, mod.flags, mod.keywordFlags, unpack(mod)`.
func modArgs(source string, flags, keywordFlags int64, tags []any) []any {
	args := make([]any, 0, 3+len(tags))
	args = append(args, source, flags, keywordFlags)
	return append(args, tags...)
}

// maxOr is `skillModList:Max(cfg, name) or fallback`.
func maxOr(l *modstore.List, cfg *modstore.Cfg, fallback float64, names ...string) float64 {
	if v, ok := l.Max(cfg, names...); ok {
		return v
	}
	return fallback
}

// wdNum reads a weaponData field (`weaponData[key] or 0`).
func wdNum(wd map[string]any, key string) float64 {
	if wd == nil {
		return 0
	}
	return anyNum(wd[key])
}

// offenceSkillData ports L520-1022.
func (env *Env) offenceSkillData(c *offenceCtx) {
	actor, skillModList, skillCfg, skillData := c.actor, c.skillModList, c.skillCfg, c.skillData
	skillFlags, output, modDB := c.skillFlags, c.output, c.modDB
	activeSkill := c.activeSkill
	d := env.Data

	if modDB.Flag(nil, "Elusive") && skillModList.Flag(nil, "SupportedByNightblade") {
		elusiveEffect := outNum(output, "ElusiveEffectMod") / 100
		nightbladeMulti := skillModList.Sum("BASE", nil, "NightbladeElusiveCritMultiplier")
		skillModList.AddMod(newMod("CritMultiplier", "BASE", math.Floor(nightbladeMulti*elusiveEffect), "Nightblade"))
	}

	// set other limits
	output["ActiveTrapLimit"] = skillModList.Sum("BASE", skillCfg, "ActiveTrapLimit")
	output["ActiveMineLimit"] = skillModList.Sum("BASE", skillCfg, "ActiveMineLimit")

	// set flask scaling
	if v, ok := env.ItemModDB.Multipliers["LifeFlaskRecovery"]; ok {
		output["LifeFlaskRecovery"] = v
	} else {
		delete(output, "LifeFlaskRecovery")
	}
	if v, ok := env.ItemModDB.Multipliers["LifeFlaskCharges"]; ok {
		output["LifeFlaskCharges"] = v
	} else {
		delete(output, "LifeFlaskCharges")
	}

	if truthy(modDB.Conditions["AffectedByEnergyBlade"]) {
		dmgMod := Mod(skillModList, skillCfg, "EnergyBladeDamage")
		speedMod := Mod(skillModList, skillCfg, "EnergyBladeAttackSpeed")
		// The reference iterates a two-key table with pairs(); the two
		// branches touch disjoint weapon slots, so the order is immaterial.
		for _, pair := range [][2]string{{"Weapon 1", "1"}, {"Weapon 2", "2"}} {
			slotName, side := pair[0], pair[1]
			wd := actor.ms.WeaponData1
			if side == "2" {
				wd = actor.ms.WeaponData2
			}
			it, _ := actor.ms.ItemList[slotName].(*Item)
			if it == nil || it.In.WeaponData == nil || it.In.WeaponData[1] == nil || wd == nil {
				continue
			}
			name := str(wd["name"])
			if name == "" {
				continue
			}
			base := d.ItemBases[name]
			if base == nil || base.Weapon == nil {
				continue
			}
			wb := base.Weapon
			wd["CritChance"] = wb.CritChanceBase
			wd["AttackRate"] = wb.AttackRateBase * speedMod
			wd["Range"] = wb.Range
			for _, damageType := range dmgTypeList {
				baseMin, baseMax := 0.0, 0.0
				if damageType == "Physical" {
					baseMin, baseMax = wb.PhysicalMin, wb.PhysicalMax
				}
				wd[damageType+"Min"] = baseMin + math.Floor(skillModList.Sum("BASE", skillCfg, "EnergyBladeMin"+damageType)*dmgMod)
				wd[damageType+"Max"] = baseMax + math.Floor(skillModList.Sum("BASE", skillCfg, "EnergyBladeMax"+damageType)*dmgMod)
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
			skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, damageType+"Min")*multiplier), "Battlemage", modparser.ModFlag.Spell))
			skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, damageType+"Max")*multiplier), "Battlemage", modparser.ModFlag.Spell))
		}
	}
	if wi, ok := d.WeaponTypeInfo[str(actor.ms.WeaponData1["type"])]; ok {
		c.weapon1info = &wi
	}
	if wi, ok := d.WeaponTypeInfo[str(actor.ms.WeaponData2["type"])]; ok {
		c.weapon2info = &wi
	}
	weapon1info, hasWeapon1info := c.weapon1info, c.weapon1info != nil
	hasWeapon2info := c.weapon2info != nil

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
				skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, damageType+"Min")*multiplier), "Spellblade Main Hand", modparser.ModFlag.Spell))
				skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, damageType+"Max")*multiplier), "Spellblade Main Hand", modparser.ModFlag.Spell))
			}
			if hasWeapon2info {
				for _, damageType := range dmgTypeList {
					skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor(wdNum(actor.ms.WeaponData2, damageType+"Min")*multiplier), "Spellblade Off Hand", modparser.ModFlag.Spell))
					skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor(wdNum(actor.ms.WeaponData2, damageType+"Max")*multiplier), "Spellblade Off Hand", modparser.ModFlag.Spell))
				}
			}
		}
	}
	if skillModList.Flag(nil, "MinionDamageAppliesToPlayer") || skillModList.Flag(skillCfg, "MinionDamageAppliesToPlayer") {
		// Minion Damage conversion from Spiritual Aid and The Scourge
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionDamageAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			tag, _ := value.(modparser.Tag)
			mod, _ := tag["mod"].(*modparser.Mod)
			if mod != nil && mod.Name == "Damage" && mod.Type == "INC" {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				skillModList.AddMod(newMod("Damage", "INC", anyNum(mod.Value)*multiplier, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, modifiers)...))
			}
		}
	}
	if skillModList.Flag(nil, "MinionAttackSpeedAppliesToPlayer") {
		// Minion Attack Speed conversion from Spiritual Command
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionAttackSpeedAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			tag, _ := value.(modparser.Tag)
			mod, _ := tag["mod"].(*modparser.Mod)
			if mod != nil && mod.Name == "Speed" && mod.Type == "INC" && (mod.Flags == 0 || mod.Flags&modparser.ModFlag.Attack != 0) {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				skillModList.AddMod(newMod("Speed", "INC", anyNum(mod.Value)*multiplier, modArgs(mod.Source, modparser.ModFlag.Attack, mod.KeywordFlags, modifiers)...))
			}
		}
	}
	if skillModList.Flag(nil, "MinionCastSpeedAppliesToPlayer") {
		// Minion Cast Speed conversion from Spinehail
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedMinionCastSpeedAppliesToPlayer") / 100
		for _, value := range skillModList.List(skillCfg, "MinionModifier") {
			tag, _ := value.(modparser.Tag)
			mod, _ := tag["mod"].(*modparser.Mod)
			if mod != nil && mod.Name == "Speed" && mod.Type == "INC" && (mod.Flags == 0 || mod.Flags&modparser.ModFlag.Cast != 0) {
				modifiers := GetConvertedModTags(mod, multiplier, true)
				skillModList.AddMod(newMod("Speed", "INC", anyNum(mod.Value)*multiplier, modArgs(mod.Source, modparser.ModFlag.Cast, mod.KeywordFlags, modifiers)...))
			}
		}
	}
	if skillModList.Flag(nil, "EvasionAppliesToSpellDamage") {
		// The Unblinking Eye evasion rating to spell damage conversion
		// Must run before SpellDamageAppliesToAttacks so the generated spell
		// mods can chain into attacks
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{}, "Evasion") {
			mod := value.Mod
			skillModList.AddMod(newMod("Damage", mod.Type, mod.Value, modArgs(mod.Source, modparser.ModFlag.Spell, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillModList.Flag(nil, "SpellDamageAppliesToAttacks") || skillModList.Flag(skillCfg, "SpellDamageAppliesToAttacks") {
		// Spell Damage conversion from Crown of Eyes, Kinetic Bolt, and the
		// Wandslinger notable
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedSpellDamageAppliesToAttacks") / 100
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{Flags: i64p(modparser.ModFlag.Spell)}, "Damage") {
			mod := value.Mod
			if mod.Flags&modparser.ModFlag.Spell != 0 {
				modifiers := GetConvertedModTags(mod, multiplier, false)
				skillModList.AddMod(newMod("Damage", "INC", anyNum(mod.Value)*multiplier,
					modArgs(mod.Source, (mod.Flags&^modparser.ModFlag.Spell)|modparser.ModFlag.Attack, mod.KeywordFlags, modifiers)...))
				if mod.Source == "Strength" { // Prevent double-dipping from converted strength's damage bonus
					skillModList.ReplaceMod(newMod("PhysicalDamage", "INC", 0.0, "Strength", modparser.ModFlag.Melee))
				}
			}
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToAttacks") {
		// Get all increases for this; assumption is that multiple sources
		// would not stack, so find the max
		multiplier := maxOr(skillModList, skillCfg, 100, "ImprovedCastSpeedAppliesToAttacks") / 100
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{Flags: i64p(modparser.ModFlag.Cast)}, "Speed") {
			mod := value.Mod
			// Add a new mod for all mods that are cast only
			if mod.Flags&modparser.ModFlag.Cast != 0 {
				modifiers := GetConvertedModTags(mod, multiplier, false)
				skillModList.AddMod(newMod("Speed", "INC", anyNum(mod.Value)*multiplier,
					modArgs(mod.Source, (mod.Flags&^modparser.ModFlag.Cast)|modparser.ModFlag.Attack, mod.KeywordFlags, modifiers)...))
			}
		}
	}
	if skillModList.Flag(nil, "ProjectileSpeedAppliesToBowDamage") {
		// Bow mastery projectile speed to damage with bows conversion
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{}, "ProjectileSpeed") {
			mod := value.Mod
			skillModList.AddMod(newMod("Damage", mod.Type, mod.Value,
				modArgs(mod.Source, modparser.ModFlag.Bow|modparser.ModFlag.Hit, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillModList.Flag(nil, "ClawDamageAppliesToUnarmed") {
		// Claw Damage conversion from Rigwald's Curse
		cfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Claw | modparser.ModFlag.Hit), KeywordFlags: i64p(modparser.KeywordFlag.Hit)}
		for _, value := range skillModList.Tabulate("INC", cfg, "Damage") {
			mod := value.Mod
			if mod.Flags&modparser.ModFlag.Claw != 0 {
				skillModList.AddMod(newMod("Damage", mod.Type, mod.Value,
					modArgs(mod.Source, (mod.Flags&^modparser.ModFlag.Claw)|modparser.ModFlag.Unarmed|modparser.ModFlag.Melee, mod.KeywordFlags, mod.Tags)...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawAttackSpeedAppliesToUnarmed") {
		// Claw Attack Speed conversion from Rigwald's Curse
		cfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Claw | modparser.ModFlag.Attack | modparser.ModFlag.Hit)}
		for _, value := range skillModList.Tabulate("INC", cfg, "Speed") {
			mod := value.Mod
			if mod.Flags&modparser.ModFlag.Claw != 0 && mod.Flags&modparser.ModFlag.Attack != 0 {
				skillModList.AddMod(newMod("Speed", mod.Type, mod.Value,
					modArgs(mod.Source, (mod.Flags&^modparser.ModFlag.Claw)|modparser.ModFlag.Unarmed, mod.KeywordFlags, mod.Tags)...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToUnarmed") {
		// Claw Crit Chance conversion from Rigwald's Curse
		cfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Claw | modparser.ModFlag.Hit)}
		for _, value := range skillModList.Tabulate("INC", cfg, "CritChance") {
			mod := value.Mod
			if mod.Flags&modparser.ModFlag.Claw != 0 {
				skillModList.AddMod(newMod("CritChance", mod.Type, mod.Value,
					modArgs(mod.Source, (mod.Flags&^modparser.ModFlag.Claw)|modparser.ModFlag.Unarmed, mod.KeywordFlags, mod.Tags)...))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritChanceAppliesToMinions") {
		// Claw Crit Chance conversion from Law of the Wilds
		cfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Claw | modparser.ModFlag.Hit)}
		for _, value := range skillModList.Tabulate("INC", cfg, "CritChance") {
			mod := value.Mod
			if mod.Flags&modparser.ModFlag.Claw != 0 {
				env.Minion.DB.AddMod(newMod("CritChance", mod.Type, mod.Value, mod.Source))
			}
		}
	}
	if skillModList.Flag(nil, "ClawCritMultiplierAppliesToMinions") {
		// Claw Crit Multi conversion from Law of the Wilds
		cfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Claw | modparser.ModFlag.Hit)}
		for _, value := range skillModList.Tabulate("BASE", cfg, "CritMultiplier") {
			mod := value.Mod
			if mod.Flags&modparser.ModFlag.Claw != 0 {
				env.Minion.DB.AddMod(newMod("CritMultiplier", mod.Type, mod.Value, mod.Source))
			}
		}
	}
	// The four resistance-driven crit/pen transfers each take only the first
	// tabulated mod (the reference breaks out of the loop).
	firstFlagSource := func(name string) (string, bool) {
		for _, value := range modDB.Tabulate("FLAG", nil, name) {
			return value.Mod.Source, true
		}
		return "", false
	}
	if skillModList.Flag(nil, "CritChanceIncreasedByUncappedLightningRes") {
		if src, ok := firstFlagSource("CritChanceIncreasedByUncappedLightningRes"); ok {
			skillModList.AddMod(newMod("CritChance", "INC", outNum(output, "LightningResistTotal"), src))
		}
	}
	if skillModList.Flag(nil, "CritChanceIncreasedByLightningRes") {
		if src, ok := firstFlagSource("CritChanceIncreasedByLightningRes"); ok {
			skillModList.AddMod(newMod("CritChance", "INC", outNum(output, "LightningResist"), src))
		}
	}
	if skillModList.Flag(nil, "CritChanceIncreasedByOvercappedLightningRes") {
		if src, ok := firstFlagSource("CritChanceIncreasedByOvercappedLightningRes"); ok {
			skillModList.AddMod(newMod("CritChance", "INC", outNum(output, "LightningResistOverCap"), src))
		}
	}
	if skillModList.Flag(nil, "CritChanceIncreasedBySpellSuppressChance") {
		if src, ok := firstFlagSource("CritChanceIncreasedBySpellSuppressChance"); ok {
			skillModList.AddMod(newMod("CritChance", "INC", outNum(output, "SpellSuppressionChance"), src))
		}
	}
	if skillModList.Flag(nil, "FirePenIncreasedByUncappedFireRes") {
		if src, ok := firstFlagSource("FirePenIncreasedByUncappedFireRes"); ok {
			skillModList.AddMod(newMod("FirePenetration", "BASE", outNum(output, "FireResistOverCap"), src))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAccuracy") {
		// Light Radius conversion from Corona Solaris
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{}, "LightRadius") {
			mod := value.Mod
			skillModList.AddMod(newMod("Accuracy", "INC", mod.Value, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToAreaOfEffect") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{}, "LightRadius") {
			mod := value.Mod
			skillModList.AddMod(newMod("AreaOfEffect", "INC", math.Floor(anyNum(mod.Value)/2), modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillModList.Flag(nil, "LightRadiusAppliesToDamage") {
		// Light Radius conversion from Wreath of Phrecia
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{}, "LightRadius") {
			mod := value.Mod
			skillModList.AddMod(newMod("Damage", "INC", mod.Value, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillModList.Flag(nil, "CastSpeedAppliesToTrapThrowingSpeed") {
		// Cast Speed conversion from Slavedriver's Hand
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{Flags: i64p(modparser.ModFlag.Cast)}, "Speed") {
			mod := value.Mod
			if mod.Flags == 0 || mod.Flags&modparser.ModFlag.Cast != 0 {
				skillModList.AddMod(newMod("TrapThrowingSpeed", "INC", mod.Value,
					modArgs(mod.Source, mod.Flags&^modparser.ModFlag.Cast&^modparser.ModFlag.Attack, mod.KeywordFlags, mod.Tags)...))
			}
		}
	}
	if truthy(skillData["arrowSpeedAppliesToAreaOfEffect"]) {
		// Arrow Speed conversion for Galvanic Arrow
		for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{Flags: i64p(modparser.ModFlag.Bow)}, "ProjectileSpeed") {
			mod := value.Mod
			skillModList.AddMod(newMod("AreaOfEffect", "INC", mod.Value, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillModList.Flag(nil, "SequentialProjectiles") && !skillModList.Flag(nil, "OneShotProj") &&
		!skillModList.Flag(nil, "NoAdditionalProjectiles") && !skillModList.Flag(nil, "SingleProjectile") &&
		!skillModList.Flag(nil, "TriggeredBySnipe") {
		// Applies DPS multiplier based on projectile count
		skillData["dpsMultiplier"] = skillModList.Sum("BASE", skillCfg, "ProjectileCount")
	}

	env.offenceRepeats(c)

	if skillModList.Flag(nil, "WeaponPhysAppliesToSpells") {
		// Phys from weapon to Spells from Runegraft
		// #EVAL: `Sum(...) or 100` — Sum always returns a number, so the
		// fallback is dead and an absent mod means a 0% multiplier.
		mult := skillModList.Sum("BASE", skillCfg, "WeaponPhysAppliesToSpellsPercent") / 100
		if actor.ms.WeaponData1 != nil {
			skillModList.AddMod(newMod("PhysicalMin", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, "PhysicalMin")*mult), "Runegraft of the Spellbound", modparser.ModFlag.Spell))
			skillModList.AddMod(newMod("PhysicalMax", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, "PhysicalMax")*mult), "Runegraft of the Spellbound", modparser.ModFlag.Spell))
		}
	}
	if truthy(skillData["gainPercentBaseWandDamageToSpells"]) {
		mult := anyNum(skillData["gainPercentBaseWandDamageToSpells"]) / 100
		w1Wand := str(actor.ms.WeaponData1["type"]) == "Wand"
		w2Wand := str(actor.ms.WeaponData2["type"]) == "Wand"
		switch {
		case w1Wand && w2Wand:
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor((wdNum(actor.ms.WeaponData1, damageType+"Min")+wdNum(actor.ms.WeaponData2, damageType+"Min"))/2*mult), "Spellslinger", modparser.ModFlag.Spell))
				skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor((wdNum(actor.ms.WeaponData1, damageType+"Max")+wdNum(actor.ms.WeaponData2, damageType+"Max"))/2*mult), "Spellslinger", modparser.ModFlag.Spell))
			}
		case w1Wand:
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, damageType+"Min")*mult), "Spellslinger", modparser.ModFlag.Spell))
				skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor(wdNum(actor.ms.WeaponData1, damageType+"Max")*mult), "Spellslinger", modparser.ModFlag.Spell))
			}
		case w2Wand:
			for _, damageType := range dmgTypeList {
				skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor(wdNum(actor.ms.WeaponData2, damageType+"Min")*mult), "Spellslinger", modparser.ModFlag.Spell))
				skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor(wdNum(actor.ms.WeaponData2, damageType+"Max")*mult), "Spellslinger", modparser.ModFlag.Spell))
			}
		}
	}
	if truthy(skillData["gainPercentBaseDaggerDamageToSpells"]) {
		weapon1IsDagger := truthy(actor.ms.WeaponData1["AddedUsingDagger"]) || str(actor.ms.WeaponData1["type"]) == "Dagger"
		weapon2IsDagger := truthy(actor.ms.WeaponData2["AddedUsingDagger"]) || str(actor.ms.WeaponData2["type"]) == "Dagger"
		both := 1.0
		if weapon1IsDagger && weapon2IsDagger {
			both = 0.5
		}
		mult := anyNum(skillData["gainPercentBaseDaggerDamageToSpells"]) / 100 * both
		for _, damageType := range dmgTypeList {
			baseMin, baseMax := 0.0, 0.0
			if weapon1IsDagger {
				baseMin += wdNum(actor.ms.WeaponData1, damageType+"Min")
				baseMax += wdNum(actor.ms.WeaponData1, damageType+"Max")
			}
			if weapon2IsDagger {
				baseMin += wdNum(actor.ms.WeaponData2, damageType+"Min")
				baseMax += wdNum(actor.ms.WeaponData2, damageType+"Max")
			}
			skillModList.AddMod(newMod(damageType+"Min", "BASE", math.Floor(baseMin*mult), "Blade Blast of Dagger Detonation", modparser.ModFlag.Spell))
			skillModList.AddMod(newMod(damageType+"Max", "BASE", math.Floor(baseMax*mult), "Blade Blast of Dagger Detonation", modparser.ModFlag.Spell))
		}
	}

	if skillModList.Flag(nil, "HasSeals") && activeSkill.SkillTypes[modparser.SkillType.CanRapidFire] && !skillModList.Flag(nil, "NoRepeatBonuses") {
		// Applies DPS multiplier based on seals count
		output["SealCooldown"] = skillModList.Sum("BASE", skillCfg, "SealGainFrequency") / Mod(skillModList, skillCfg, "SealGainFrequency")
		output["SealMax"] = skillModList.Sum("BASE", skillCfg, "SealCount")
		output["AverageBurstHits"] = output["SealMax"]
		output["TimeMaxSeals"] = outNum(output, "SealCooldown") * outNum(output, "SealMax")

		if !truthy(skillData["hitTimeOverride"]) {
			castTime := 0.0
			if ct := activeSkill.ActiveEffect.GrantedEffect.CastTime; ct != nil {
				castTime = *ct
			}
			if skillModList.Flag(nil, "UseMaxUnleash") {
				for _, value := range skillModList.Tabulate("INC", &modstore.Cfg{}, "MaxSealCrit") {
					mod := value.Mod
					skillModList.AddMod(newMod("CritChance", "INC", mod.Value, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
				}
				for _, value := range skillModList.Tabulate("MORE", &modstore.Cfg{}, "MaxSealDamage") {
					mod := value.Mod
					skillModList.AddMod(newMod("Damage", "MORE", anyNum(mod.Value)*outNum(output, "SealMax"), modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
				}
				env.PlayerMainSkill.SkillData["dpsMultiplier"] = 1 + outNum(output, "SealMax")*Mod(skillModList, skillCfg, "SealRepeatPenalty")
				env.PlayerMainSkill.SkillData["hitTimeOverride"] = math.Max(outNum(output, "TimeMaxSeals"),
					1/castTime*1.1*Mod(skillModList, skillCfg, "Speed")*outNum(output, "ActionSpeedMod"))
			} else {
				env.PlayerMainSkill.SkillData["dpsMultiplier"] = 1 + 1/outNum(output, "SealCooldown")/
					(1/castTime*1.1*Mod(skillModList, skillCfg, "Speed")*outNum(output, "ActionSpeedMod"))*
					Mod(skillModList, skillCfg, "SealRepeatPenalty")
			}
		}
	}

	physMode := "AVERAGE"
	if v := str(env.ConfigInput["physMode"]); v != "" {
		physMode = v
	}
	processedRandomMods := map[*modparser.Mod]bool{}
	for _, cfg := range []*modstore.Cfg{skillCfg, activeSkill.Weapon1Cfg, activeSkill.Weapon2Cfg} {
		if cfg == nil || skillModList.Sum("BASE", cfg, "PhysicalDamageGainAsRandom", "PhysicalDamageConvertToRandom", "PhysicalDamageGainAsColdOrLightning") <= 0 {
			continue
		}
		skillFlags["randomPhys"] = true
		for _, value := range skillModList.Tabulate("BASE", cfg, "PhysicalDamageGainAsRandom") {
			mod := value.Mod
			if processedRandomMods[mod] {
				continue
			}
			processedRandomMods[mod] = true
			effVal := anyNum(mod.Value) / 3
			args := modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)
			switch physMode {
			case "AVERAGE":
				skillModList.AddMod(newMod("PhysicalDamageGainAsFire", "BASE", effVal, args...))
				skillModList.AddMod(newMod("PhysicalDamageGainAsCold", "BASE", effVal, args...))
				skillModList.AddMod(newMod("PhysicalDamageGainAsLightning", "BASE", effVal, args...))
			case "FIRE":
				skillModList.AddMod(newMod("PhysicalDamageGainAsFire", "BASE", mod.Value, args...))
			case "COLD":
				skillModList.AddMod(newMod("PhysicalDamageGainAsCold", "BASE", mod.Value, args...))
			case "LIGHTNING":
				skillModList.AddMod(newMod("PhysicalDamageGainAsLightning", "BASE", mod.Value, args...))
			}
		}
		for _, value := range skillModList.Tabulate("BASE", cfg, "PhysicalDamageConvertToRandom") {
			mod := value.Mod
			if processedRandomMods[mod] {
				continue
			}
			processedRandomMods[mod] = true
			effVal := anyNum(mod.Value) / 3
			args := modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)
			switch physMode {
			case "AVERAGE":
				skillModList.AddMod(newMod("PhysicalDamageConvertToFire", "BASE", effVal, args...))
				skillModList.AddMod(newMod("PhysicalDamageConvertToCold", "BASE", effVal, args...))
				skillModList.AddMod(newMod("PhysicalDamageConvertToLightning", "BASE", effVal, args...))
			case "FIRE":
				skillModList.AddMod(newMod("PhysicalDamageConvertToFire", "BASE", mod.Value, args...))
			case "COLD":
				skillModList.AddMod(newMod("PhysicalDamageConvertToCold", "BASE", mod.Value, args...))
			case "LIGHTNING":
				skillModList.AddMod(newMod("PhysicalDamageConvertToLightning", "BASE", mod.Value, args...))
			}
		}
		for _, value := range skillModList.Tabulate("BASE", cfg, "PhysicalDamageGainAsColdOrLightning") {
			mod := value.Mod
			if processedRandomMods[mod] {
				continue
			}
			processedRandomMods[mod] = true
			effVal := anyNum(mod.Value) / 2
			args := modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)
			switch physMode {
			case "AVERAGE", "FIRE":
				skillModList.AddMod(newMod("PhysicalDamageGainAsCold", "BASE", effVal, args...))
				skillModList.AddMod(newMod("PhysicalDamageGainAsLightning", "BASE", effVal, args...))
			case "COLD":
				skillModList.AddMod(newMod("PhysicalDamageGainAsCold", "BASE", mod.Value, args...))
			case "LIGHTNING":
				skillModList.AddMod(newMod("PhysicalDamageGainAsLightning", "BASE", mod.Value, args...))
			}
		}
	}
	// momentum stacks
	if skillModList.Flag(nil, "SupportedByMomentum") {
		maxMomentumStacks := skillModList.Sum("BASE", skillCfg, "MomentumStacksMax")
		extraMomentumStacks := skillModList.Sum("BASE", skillCfg, "MomentumStacksExtra")
		combat := modparser.Tag{"type": "Condition", "var": "Combat"}
		if maxMomentumStacks > 0 {
			if !modDB.HasMod("BASE", nil, "Multiplier:MomentumStacks") {
				modDB.AddMod(newMod("Multiplier:MomentumStacks", "BASE",
					math.Min((maxMomentumStacks+extraMomentumStacks)/2, maxMomentumStacks), "Config", combat))
			} else if modDB.Sum("BASE", nil, "Multiplier:MomentumStacks") > maxMomentumStacks {
				modDB.ReplaceMod(newMod("Multiplier:MomentumStacks", "BASE", maxMomentumStacks, "Config", combat))
			}
		} else if modDB.HasMod("BASE", nil, "Multiplier:MomentumStacks") {
			modDB.ReplaceMod(newMod("Multiplier:MomentumStacks", "BASE", 0.0, "Config"))
		}
	}
}

// repeatSkillTypesCheck ports the local of the same name (L775).
func repeatSkillTypesCheck(skillModList *modstore.List, activeSkillTypes map[int64]bool) bool {
	excludeSkillTypes := []int64{
		modparser.SkillType.Instant, modparser.SkillType.Channel, modparser.SkillType.Triggered,
		modparser.SkillType.Retaliation, modparser.SkillType.NonRepeatable,
	}
	for _, typ := range excludeSkillTypes {
		if activeSkillTypes[typ] {
			return false
		}
	}
	return !skillModList.Flag(nil, "CannotRepeat") &&
		(activeSkillTypes[modparser.SkillType.Attack] || activeSkillTypes[modparser.SkillType.Spell])
}

// offenceRepeats ports L783-869: output.Repeats and everything the repeat
// count multiplies.
func (env *Env) offenceRepeats(c *offenceCtx) {
	skillModList, skillCfg, output := c.skillModList, c.skillCfg, c.output
	skillFlags, activeSkill := c.skillFlags, c.activeSkill

	repeats := 1.0
	if repeatSkillTypesCheck(skillModList, activeSkill.SkillTypes) {
		repeats += skillModList.Sum("BASE", skillCfg, "RepeatCount")
	}
	output["Repeats"] = repeats
	if repeats <= 1 {
		return
	}
	output["RepeatCount"] = repeats
	// handle all the multipliers from Repeats
	repeatMode := str(env.ConfigInput["repeatMode"])
	if repeatMode == "NONE" {
		return
	}
	average := repeatMode == "AVERAGE"

	for _, value := range skillModList.Tabulate("INC", skillCfg, "RepeatFinalAreaOfEffect") {
		mod := value.Mod
		modValue := anyNum(mod.Value)
		if average {
			modValue /= repeats
		}
		skillModList.AddMod(newMod("AreaOfEffect", "INC", modValue, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
	}
	for _, value := range skillModList.Tabulate("INC", skillCfg, "RepeatPerRepeatAreaOfEffect") {
		mod := value.Mod
		modValue := anyNum(mod.Value) * (repeats - 1)
		if average {
			modValue /= 2
		}
		skillModList.AddMod(newMod("AreaOfEffect", "INC", modValue, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
	}
	for _, value := range skillModList.Tabulate("BASE", skillCfg, "RepeatFinalDoubleDamageChance") {
		mod := value.Mod
		modValue := anyNum(mod.Value)
		if average {
			modValue /= repeats
		}
		skillModList.AddMod(newMod("DoubleDamageChance", "BASE", modValue, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
	}
	damageFinalMoreValueTotal := 1.0
	damageMoreValueTotal := 0.0
	for _, value := range skillModList.Tabulate("MORE", skillCfg, "RepeatFinalDamage") {
		mod := value.Mod
		modValue := anyNum(mod.Value)
		damageFinalMoreValueTotal *= 1 + modValue/100
		damageMoreValueTotal += modValue
		if average && !skillModList.Flag(nil, "OnlyFinalRepeat") {
			modValue /= repeats
		}
		skillModList.AddMod(newMod("Damage", "MORE", modValue, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
	}
	for _, value := range skillModList.Tabulate("MORE", skillCfg, "RepeatPerRepeatDamage") {
		mod := value.Mod
		modValue := anyNum(mod.Value) * (repeats - 1)
		if average {
			if damageFinalMoreValueTotal != 1 {
				// sum from 0 to num Repeats the damage each one does,
				// multiplied by the other repeat multipliers, divide the
				// total by the average other repeat multipliers and divide
				// by number of repeats
				modValue = ((100+anyNum(mod.Value)*(repeats-2)/2)*(repeats-1)+
					(100+anyNum(mod.Value)*(repeats-1))*damageFinalMoreValueTotal)/
					(repeats+damageMoreValueTotal/100) - 100
			} else {
				modValue /= 2
			}
		}
		skillModList.AddMod(newMod("Damage", "MORE", modValue, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
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
			for _, value := range skillModList.Tabulate("MORE", skillCfg, "Repeat"+repeatCount.name+"Damage") {
				damageMoreValueTotal += anyNum(value.Mod.Value)
				lastMod = value.Mod
			}
		} else if repeatCount.n == repeats {
			for _, value := range skillModList.Tabulate("MORE", skillCfg, "Repeat"+repeatCount.name+"Damage") {
				mod := value.Mod
				skillModList.AddMod(newMod("Damage", "MORE", mod.Value, modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
			}
		}
	}
	if average && lastMod != nil {
		skillModList.AddMod(newMod("Damage", "MORE",
			(damageMoreValueTotal/repeats+100)/(1+damageFinalMoreValueTotal/repeats/100)-100,
			modArgs(lastMod.Source, lastMod.Flags, lastMod.KeywordFlags, lastMod.Tags)...))
	}
	if skillModList.Flag(nil, "FinalRepeatSumsDamage") {
		for _, value := range skillModList.Tabulate("FLAG", skillCfg, "FinalRepeatSumsDamage") {
			mod := value.Mod
			skillModList.AddMod(newMod("Damage", "MORE",
				(100*repeats+damageFinalMoreValueTotal)/(1+damageFinalMoreValueTotal/100)-100,
				modArgs(mod.Source, mod.Flags, mod.KeywordFlags, mod.Tags)...))
		}
	}
	if skillFlags["trap"] || skillFlags["mine"] {
		skillModList.AddMod(newMod("DPS", "MORE", (repeats-1)*100, "Repeat Count"))
	}
}
