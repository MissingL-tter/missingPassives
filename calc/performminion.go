// CalcPerform.lua L1120-1252 (minion mod DB), CalcActiveSkill L868-924
// (createMinionSkills), CalcOffence calcSkillDuration, and CalcDefence
// defenceForConditionals — the perform-stage helpers around minions and
// durations.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// initMinionModDB ports CalcPerform's initMinionModDB.
func (env *Env) initMinionModDB(activeSkill *ActiveSkill, output modstore.Output) {
	modDB := env.ModDB
	minion := activeSkill.Minion
	if output == nil {
		output = modstore.Output{}
	}
	minion.Output = output
	minion.DB.Multipliers["Level"] = minion.Level
	env.initModDB(minion.DB)
	baseLife := minion.LifeTable[int(minion.Level)-1] * minion.MinionData.Life
	if minion.Hostile {
		mult, ok := data.MapLevelLifeMult[int64(env.EnemyLevel)]
		if !ok {
			mult = 1
		}
		baseLife = baseLife * mult
	}
	minion.DB.AddMod(newModS("Life", modparser.Base, modparser.Num(math.Floor(baseLife)), "Base"))
	if minion.MinionData.EnergyShield != nil {
		esTable := data.MonsterAllyLifeTable
		if minion.Hostile {
			esTable = minion.LifeTable
		}
		baseES := esTable[int(minion.Level)-1] * minion.MinionData.Life * *minion.MinionData.EnergyShield
		if minion.Hostile {
			mult, ok := data.MapLevelLifeMult[int64(env.EnemyLevel)]
			if !ok {
				mult = 1
			}
			baseES = baseES * mult
		}
		minion.DB.AddMod(newModS("EnergyShield", modparser.Base, modparser.Num(math.Floor(baseES)), "Base"))
	}
	armourMult := 1.0
	if minion.MinionData.Armour != nil {
		armourMult = *minion.MinionData.Armour
	}
	minion.DB.AddMod(newModS("Armour", modparser.Base, modparser.Num(util.RoundHalfUp(data.MonsterArmourTable[int(minion.Level)-1]*armourMult, 0)), "Base"))
	evasionMult := 1.0
	if minion.MinionData.Evasion != nil {
		evasionMult = *minion.MinionData.Evasion
	}
	minion.DB.AddMod(newModS("Evasion", modparser.Base, modparser.Num(util.RoundHalfUp(data.MonsterEvasionTable[int(minion.Level)-1]*evasionMult, 0)), "Base"))
	if modDB.Flag(nil, "MinionAccuracyEqualsAccuracy") {
		accPerDex := data.Misc.AccuracyPerDexBase
		if ov, ok := modDB.Override(nil, "DexAccBonusOverride"); ok {
			accPerDex = valueNum(ov)
		}
		minion.DB.AddMod(newModS("Accuracy", modparser.Base, modparser.Num(Val(modDB, "Accuracy", nil)+Val(modDB, "Dex", nil)*accPerDex), "Player"))
	} else {
		minion.DB.AddMod(newModS("CannotBeEvaded", modparser.Flag, modparser.Num(1.0), "Minion Attacks always hit"))
	}
	mc := data.MonsterConstants
	minion.DB.AddMod(newModS("CritMultiplier", modparser.Base, modparser.Num(mc["base_critical_strike_multiplier"]-100), "Base"))
	minion.DB.AddMod(newModS("DotMultiplier", modparser.Base, modparser.Num(mc["critical_ailment_dot_multiplier_+"]), "Base", &modparser.CondTag{Var: "CriticalStrike"}))
	minion.DB.AddMod(newModS("FireResist", modparser.Base, modparser.Num(minion.MinionData.FireResist), "Base"))
	minion.DB.AddMod(newModS("ColdResist", modparser.Base, modparser.Num(minion.MinionData.ColdResist), "Base"))
	minion.DB.AddMod(newModS("LightningResist", modparser.Base, modparser.Num(minion.MinionData.LightningResist), "Base"))
	minion.DB.AddMod(newModS("ChaosResist", modparser.Base, modparser.Num(minion.MinionData.ChaosResist), "Base"))
	minion.DB.AddMod(newModS("CritChance", modparser.Inc, modparser.Num(mc["critical_strike_chance_+%_per_power_charge"]), "Base", &modparser.MultiplierTag{Var: "PowerCharge"}))
	minion.DB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(mc["base_attack_speed_+%_per_frenzy_charge"]), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.MultiplierTag{Var: "FrenzyCharge"}))
	minion.DB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(mc["base_cast_speed_+%_per_frenzy_charge"]), "Base", modparser.FlagCast, modparser.KeywordNone, &modparser.MultiplierTag{Var: "FrenzyCharge"}))
	minion.DB.AddMod(newModS("Damage", modparser.More, modparser.Num(mc["object_inherent_damage_+%_final_per_frenzy_charge"]), "Base", &modparser.MultiplierTag{Var: "FrenzyCharge"}))
	minion.DB.AddMod(newModS("PhysicalDamageReduction", modparser.Base, modparser.Num(mc["physical_damage_reduction_%_per_endurance_charge"]), "Base", &modparser.MultiplierTag{Var: "EnduranceCharge"}))
	minion.DB.AddMod(newModS("ElementalDamageReduction", modparser.Base, modparser.Num(mc["elemental_damage_reduction_%_per_endurance_charge_if_player_minion"]), "Base", &modparser.MultiplierTag{Var: "EnduranceCharge"}))
	minion.DB.AddMod(newModS("ProjectileCount", modparser.Base, modparser.Num(1.0), "Base"))
	minion.DB.AddMod(newModS("MaximumFortification", modparser.Base, modparser.Num(mc["base_max_fortification"]), "Base"))
	minion.DB.AddMod(newModSF("Damage", modparser.More, modparser.Num(200.0), "Base", modparser.FlagNone, modparser.KeywordBleed, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "Moving"}))
	for _, mod := range minion.MinionData.ModList {
		minion.DB.AddMod(mod)
	}
	for _, mod := range activeSkill.ExtraSkillModList {
		minion.DB.AddMod(mod)
	}
	if env.AegisModList != nil {
		minion.ItemList["Weapon 3"] = env.aegisItem
		minion.DB.AddList(env.AegisModList.Mods)
	}
	if env.TheIronMass != nil && minion.Type == "RaisedSkeleton" {
		minion.DB.AddList(env.TheIronMass.Mods)
	}
	if activeSkill.SkillData.Flag("minionUseBowAndQuiver") {
		if weaponType(weaponOf(env.Player.WeaponData1)) == "Bow" {
			w1, _ := env.Player.ItemList["Weapon 1"].(*Item)
			minion.DB.AddList(w1.In.SlotModList[1])
		}
		if w2, _ := env.Player.ItemList["Weapon 2"].(*Item); w2 != nil && w2.In.Type == "Quiver" {
			minion.DB.ScaleAddList(w2.In.ModList, math.Max(modDB.Sum(modparser.Base, nil, "WidowHailMultiplier"), 1), false)
		}
		if modDB.Flag(nil, "BlinkAndMirrorUseGloves") {
			if gloves, _ := env.Player.ItemList["Gloves"].(*Item); gloves != nil {
				minion.DB.AddList(gloves.In.ModList)
			}
		}
	}
	if activeSkill.SkillData.Flag("minionUseMainHandWeapon") {
		w1, _ := env.Player.ItemList["Weapon 1"].(*Item)
		minion.DB.AddList(w1.In.SlotModList[1])
	}
	if minion.ItemSet != nil || minion.Uses != nil {
		// iterate build.itemsTab.slots (name-keyed in the reference; slot
		// list order here — the writes are per-distinct-slot, order-free)
		for _, slot := range env.Build.ItemsTab.Slots {
			slotName := slot.SlotName
			if !minion.Uses[slotName] {
				continue
			}
			var item *Item
			if minion.ItemSet != nil {
				setSlot := slotName
				if slot.WeaponSet != nil && *slot.WeaponSet == 1 && minion.ItemSet.UseSecondWeaponSet != nil && *minion.ItemSet.UseSecondWeaponSet {
					setSlot = slotName + " Swap"
				}
				item = env.ItemPool[int(minion.ItemSet.Slots[setSlot])]
			} else {
				item, _ = env.Player.ItemList[slotName].(*Item)
			}
			if item != nil {
				minion.ItemList[slotName] = item
				srcList := item.In.ModList
				if srcList == nil && item.In.SlotModList != nil && slot.SlotNum != nil {
					srcList = item.In.SlotModList[int(*slot.SlotNum)]
				}
				minion.DB.AddList(srcList)
			}
		}
	}
	if modDB.Sum(modparser.Base, nil, "StrengthAddedToMinions") > 0 {
		minion.DB.AddMod(newModS("Str", modparser.Base, modparser.Num(util.RoundHalfUp(Val(modDB, "Str", nil)*modDB.Sum(modparser.Base, nil, "StrengthAddedToMinions")/100, 0)), "Player"))
	}
}

// addMinionModifiers ports CalcPerform's addMinionModifiers.
func addMinionModifiers(modList modstore.Store, skillCfg *modstore.Cfg, minion *Minion) {
	for _, v := range modList.List(skillCfg, "MinionModifier") {
		tag, ok := v.(modparser.ModRef)
		if !ok || tag.Mod == nil {
			continue
		}
		if tag.MinionType == "" || minion.Type == tag.MinionType {
			minion.DB.AddMod(tag.Mod)
		}
	}
}

// createMinionSkills ports calcs.createMinionSkills.
func (env *Env) createMinionSkills(activeSkill *ActiveSkill) {
	activeEffect := activeSkill.ActiveEffect
	minion := activeSkill.Minion
	minionData := minion.MinionData

	minion.ActiveSkillList = nil
	var skillIdList []string
	for _, skillId := range minionData.SkillList {
		if data.Skills[skillId] != nil {
			skillIdList = append(skillIdList, skillId)
		}
	}
	for _, v := range activeSkill.SkillModList.List(activeSkill.SkillCfg, "ExtraMinionSkill") {
		tag, ok := v.(modparser.SkillRef)
		if !ok {
			panic("calc: non-SkillRef value in ExtraMinionSkill list (the Lua errors)")
		}
		match := true
		if tag.MinionList != nil {
			match = false
			for _, mv := range tag.MinionList {
				if mv == minion.Type {
					match = true
					break
				}
			}
		}
		if match {
			skillIdList = append(skillIdList, tag.SkillID)
		}
	}
	if len(skillIdList) == 0 {
		// avoid horrible crashes if a spectre has no skills for some reason
		skillIdList = append(skillIdList, "Melee")
	}
	for _, skillId := range skillIdList {
		ge := data.Skills[skillId]
		minionEffect := &ActiveEffect{GrantedEffect: ge, Level: 1, Quality: 0}
		if len(ge.Levels) > 1 {
			// walk levels 1..n while levelRequirement <= minion level
			for level := 1; ; level++ {
				levelData := ge.Levels[level]
				if levelData == nil {
					break
				}
				req, _ := lvlExtra(levelData, "levelRequirement")
				if req > minion.Level {
					break
				}
				minionEffect.Level = float64(level)
			}
		}
		minionSkill := env.createActiveSkill(minionEffect, activeSkill.SupportList, minion.Ms, nil, activeSkill)
		env.buildActiveSkillModList(minionSkill)
		minionSkill.SkillFlags["minion"] = true
		minionSkill.SkillFlags["minionSkill"] = true
		minionSkill.SkillFlags["haveMinion"] = true
		setFlag(minionSkill.SkillFlags, "spectre", activeSkill.SkillFlags["spectre"])
		minionSkill.SkillData.SetN("damageEffectiveness", 1+activeSkill.SkillData.N("minionDamageEffectiveness")/100)
		minion.ActiveSkillList = append(minion.ActiveSkillList, minionSkill)
	}
	skillIndex := 1.0
	if v := activeEffect.SrcInstance.SkillMinionSkill; v.Set {
		skillIndex = v.V
	}
	skillIndex = math.Max(math.Min(skillIndex, float64(len(minion.ActiveSkillList))), 1)
	if env.Mode == ModeMain {
		activeEffect.SrcInstance.SkillMinionSkill = util.Some(skillIndex)
	}
	minion.MainSkill = minion.ActiveSkillList[int(skillIndex)-1]
}

// calcSkillDuration ports CalcOffence's calcSkillDuration.
func (env *Env) calcSkillDuration(skillModList modstore.Store, skillCfg *modstore.Cfg, skillData *SkillData, enemyDB *modstore.DB) float64 {
	durationNames := []string{"Duration", "PrimaryDuration"}
	if skillData.Flag("mineDurationAppliesToSkill") {
		durationNames = append(durationNames, "MineDuration")
	}
	durationMod := Mod(skillModList, skillCfg, durationNames...)
	durationMod = math.Max(durationMod, 0)
	durationBase := skillData.N("duration") + skillModList.Sum(modparser.Base, skillCfg, "Duration", "PrimaryDuration")
	duration := durationBase * durationMod
	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(data.Misc.BuffExpirationSlowCap, Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	if skillData.Flag("debuff") {
		duration = duration * debuffDurationMult
	}
	return duration
}

// defenceForConditionals ports CalcDefence's calcs.defenceForConditionals.
func (env *Env) defenceForConditionals(actor *performActor) {
	modDB := actor.db
	output := actor.output
	for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
		item, _ := actor.ms.ItemList[slot].(*Item)
		if item == nil || item.In.ArmourData == nil {
			continue
		}
		for _, def := range []string{"Ward", "EnergyShield", "Armour", "Evasion"} {
			base := 0.0
			if !modDB.Flag(nil, "GainNo"+def+"From"+slot) {
				base = armourDataOf(item, def)
			}
			if base > 0 {
				output.SetN(def+"On"+slot, base)
			}
		}
	}
}
