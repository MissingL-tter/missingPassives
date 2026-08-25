// CalcPerform.lua L1120-1252 (minion mod DB), CalcActiveSkill L868-924
// (createMinionSkills), CalcOffence calcSkillDuration, and CalcDefence
// defenceForConditionals — the perform-stage helpers around minions and
// durations.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// initMinionModDB ports CalcPerform's initMinionModDB.
func (env *Env) initMinionModDB(activeSkill *ActiveSkill, output map[string]any) {
	modDB := env.ModDB
	minion := activeSkill.Minion
	if output == nil {
		output = map[string]any{}
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
	minion.DB.AddMod(newMod("Life", "BASE", math.Floor(baseLife), "Base"))
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
		minion.DB.AddMod(newMod("EnergyShield", "BASE", math.Floor(baseES), "Base"))
	}
	armourMult := 1.0
	if minion.MinionData.Armour != nil {
		armourMult = *minion.MinionData.Armour
	}
	minion.DB.AddMod(newMod("Armour", "BASE", roundDec(data.MonsterArmourTable[int(minion.Level)-1]*armourMult, 0), "Base"))
	evasionMult := 1.0
	if minion.MinionData.Evasion != nil {
		evasionMult = *minion.MinionData.Evasion
	}
	minion.DB.AddMod(newMod("Evasion", "BASE", roundDec(data.MonsterEvasionTable[int(minion.Level)-1]*evasionMult, 0), "Base"))
	if modDB.Flag(nil, "MinionAccuracyEqualsAccuracy") {
		accPerDex := data.Misc.AccuracyPerDexBase
		if ov := modDB.Override(nil, "DexAccBonusOverride"); truthy(ov) {
			accPerDex = anyNum(ov)
		}
		minion.DB.AddMod(newMod("Accuracy", "BASE", Val(modDB, "Accuracy", nil)+Val(modDB, "Dex", nil)*accPerDex, "Player"))
	} else {
		minion.DB.AddMod(newMod("CannotBeEvaded", "FLAG", 1.0, "Minion Attacks always hit"))
	}
	mc := data.MonsterConstants
	minion.DB.AddMod(newMod("CritMultiplier", "BASE", mc["base_critical_strike_multiplier"]-100, "Base"))
	minion.DB.AddMod(newMod("DotMultiplier", "BASE", mc["critical_ailment_dot_multiplier_+"], "Base", modparser.Tag{"type": "Condition", "var": "CriticalStrike"}))
	minion.DB.AddMod(newMod("FireResist", "BASE", minion.MinionData.FireResist, "Base"))
	minion.DB.AddMod(newMod("ColdResist", "BASE", minion.MinionData.ColdResist, "Base"))
	minion.DB.AddMod(newMod("LightningResist", "BASE", minion.MinionData.LightningResist, "Base"))
	minion.DB.AddMod(newMod("ChaosResist", "BASE", minion.MinionData.ChaosResist, "Base"))
	minion.DB.AddMod(newMod("CritChance", "INC", mc["critical_strike_chance_+%_per_power_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "PowerCharge"}))
	minion.DB.AddMod(newMod("Speed", "INC", mc["base_attack_speed_+%_per_frenzy_charge"], "Base", modparser.ModFlag.Attack, modparser.Tag{"type": "Multiplier", "var": "FrenzyCharge"}))
	minion.DB.AddMod(newMod("Speed", "INC", mc["base_cast_speed_+%_per_frenzy_charge"], "Base", modparser.ModFlag.Cast, modparser.Tag{"type": "Multiplier", "var": "FrenzyCharge"}))
	minion.DB.AddMod(newMod("Damage", "MORE", mc["object_inherent_damage_+%_final_per_frenzy_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "FrenzyCharge"}))
	minion.DB.AddMod(newMod("PhysicalDamageReduction", "BASE", mc["physical_damage_reduction_%_per_endurance_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "EnduranceCharge"}))
	minion.DB.AddMod(newMod("ElementalDamageReduction", "BASE", mc["elemental_damage_reduction_%_per_endurance_charge_if_player_minion"], "Base", modparser.Tag{"type": "Multiplier", "var": "EnduranceCharge"}))
	minion.DB.AddMod(newMod("ProjectileCount", "BASE", 1.0, "Base"))
	minion.DB.AddMod(newMod("MaximumFortification", "BASE", mc["base_max_fortification"], "Base"))
	minion.DB.AddMod(newMod("Damage", "MORE", 200.0, "Base", int64(0), modparser.KeywordFlag.Bleed, modparser.Tag{"type": "ActorCondition", "actor": "enemy", "var": "Moving"}))
	for _, mv := range minion.MinionData.ModList {
		if mod, ok := mv.(*modparser.Mod); ok {
			minion.DB.AddMod(mod)
		} else {
			panic("calc: non-mod minion modList entry (flags-slot artifact)")
		}
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
	if truthy(activeSkill.SkillData["minionUseBowAndQuiver"]) {
		if str(env.Player.WeaponData1["type"]) == "Bow" {
			w1, _ := env.Player.ItemList["Weapon 1"].(*Item)
			minion.DB.AddList(w1.In.SlotModList[1])
		}
		if w2, _ := env.Player.ItemList["Weapon 2"].(*Item); w2 != nil && w2.In.Type == "Quiver" {
			minion.DB.ScaleAddList(w2.In.ModList, math.Max(modDB.Sum("BASE", nil, "WidowHailMultiplier"), 1), false)
		}
		if modDB.Flag(nil, "BlinkAndMirrorUseGloves") {
			if gloves, _ := env.Player.ItemList["Gloves"].(*Item); gloves != nil {
				minion.DB.AddList(gloves.In.ModList)
			}
		}
	}
	if truthy(activeSkill.SkillData["minionUseMainHandWeapon"]) {
		w1, _ := env.Player.ItemList["Weapon 1"].(*Item)
		minion.DB.AddList(w1.In.SlotModList[1])
	}
	if minion.ItemSet != nil || minion.Uses != nil {
		// iterate build.itemsTab.slots (name-keyed in the reference; slot
		// list order here — the writes are per-distinct-slot, order-free)
		for _, slot := range env.Build.ItemsTab.Slots {
			slotName := slot.SlotName
			if !truthy(minion.Uses[slotName]) {
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
	if modDB.Sum("BASE", nil, "StrengthAddedToMinions") > 0 {
		minion.DB.AddMod(newMod("Str", "BASE", roundDec(Val(modDB, "Str", nil)*modDB.Sum("BASE", nil, "StrengthAddedToMinions")/100, 0), "Player"))
	}
}

// addMinionModifiers ports CalcPerform's addMinionModifiers.
func addMinionModifiers(modList modstore.Store, skillCfg *modstore.Cfg, minion *Minion) {
	for _, v := range modList.List(skillCfg, "MinionModifier") {
		tag, _ := v.(modparser.Tag)
		mod, _ := tag["mod"].(*modparser.Mod)
		if mod == nil {
			continue
		}
		if !truthy(tag["type"]) || minion.Type == str(tag["type"]) {
			minion.DB.AddMod(mod)
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
		tag, _ := v.(modparser.Tag)
		minionList := tag["minionList"]
		match := true
		if truthy(minionList) {
			match = false
			for _, mv := range asAnyList(minionList) {
				if str(mv) == minion.Type {
					match = true
					break
				}
			}
		}
		if match {
			skillIdList = append(skillIdList, str(tag["skillId"]))
		}
	}
	if len(skillIdList) == 0 {
		// avoid horrible crashes if a spectre has no skills for some reason
		skillIdList = append(skillIdList, "Melee")
	}
	for _, skillId := range skillIdList {
		ge := data.Skills[skillId]
		minionEffect := &ActiveEffect{GrantedEffect: ge, Level: 1, Quality: 0}
		if luaLevelsLen(ge.Levels) > 1 {
			// walk levels 1..n while levelRequirement <= minion level
			for level := 1.0; ; level++ {
				levelData := ge.Levels[level]
				if levelData == nil {
					break
				}
				req, _ := lvlExtra(levelData, "levelRequirement")
				if req > minion.Level {
					break
				}
				minionEffect.Level = level
			}
		}
		minionSkill := env.createActiveSkill(minionEffect, activeSkill.SupportList, minion.Ms, nil, activeSkill)
		env.buildActiveSkillModList(minionSkill)
		minionSkill.SkillFlags["minion"] = true
		minionSkill.SkillFlags["minionSkill"] = true
		minionSkill.SkillFlags["haveMinion"] = true
		setFlag(minionSkill.SkillFlags, "spectre", activeSkill.SkillFlags["spectre"])
		minionSkill.SkillData["damageEffectiveness"] = 1 + anyNum(activeSkill.SkillData["minionDamageEffectiveness"])/100
		minion.ActiveSkillList = append(minion.ActiveSkillList, minionSkill)
	}
	skillIndex := 1.0
	if v, ok := activeEffect.SrcInstance.KV["skillMinionSkill"]; ok && truthy(v) {
		skillIndex = anyNum(v)
	}
	skillIndex = math.Max(math.Min(skillIndex, float64(len(minion.ActiveSkillList))), 1)
	if env.Mode == "MAIN" {
		activeEffect.SrcInstance.KV["skillMinionSkill"] = skillIndex
	}
	minion.MainSkill = minion.ActiveSkillList[int(skillIndex)-1]
}

func asAnyList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case *modparser.D:
		return t.Arr
	}
	return nil
}

// calcSkillDuration ports CalcOffence's calcSkillDuration.
func (env *Env) calcSkillDuration(skillModList modstore.Store, skillCfg *modstore.Cfg, skillData map[string]any, enemyDB *modstore.DB) float64 {
	durationNames := []string{"Duration", "PrimaryDuration"}
	if truthy(skillData["mineDurationAppliesToSkill"]) {
		durationNames = append(durationNames, "MineDuration")
	}
	durationMod := Mod(skillModList, skillCfg, durationNames...)
	durationMod = math.Max(durationMod, 0)
	durationBase := anyNum(skillData["duration"]) + skillModList.Sum("BASE", skillCfg, "Duration", "PrimaryDuration")
	duration := durationBase * durationMod
	debuffDurationMult := 1.0
	if env.ModeEffective {
		debuffDurationMult = 1 / math.Max(data.Misc.BuffExpirationSlowCap, Mod(enemyDB, skillCfg, "BuffExpireFaster"))
	}
	if truthy(skillData["debuff"]) {
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
		armourData := item.In.ArmourData
		for _, def := range []string{"Ward", "EnergyShield", "Armour", "Evasion"} {
			base := 0.0
			if !modDB.Flag(nil, "GainNo"+def+"From"+slot) {
				base = anyNum(armourData[def])
			}
			if base > 0 {
				output[def+"On"+slot] = base
			}
		}
	}
}
