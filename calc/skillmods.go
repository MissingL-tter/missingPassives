// Port of CalcActiveSkill.lua buildActiveSkillModList (+ getWeaponFlags,
// mergeSkillInstanceMods/mergeLevelMod). Stat iteration is SORTED on both
// sides: the reference's pairs(stats) order is LuaJIT string-hash-random
// per process, so tools/dump_calc.lua replaces mergeSkillInstanceMods with
// a sorted-stats replica — a documented divergence from the vanilla app.
// Minion creation panics (needs the minion stage + a minion corpus).
package calc

import (
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// Minion is the activeSkill.minion table (becomes a full actor at the
// perform stage).
type Minion struct {
	Type        string
	MinionData  *data.Minion
	Hostile     bool
	Parent      *modstore.Actor
	EnemyActor  *modstore.Actor // minion.enemy
	Level       float64
	ItemList    map[string]modstore.Item
	Uses        map[string]any
	ItemSet     *ItemSetInput
	LifeTable   []float64
	WeaponData1 map[string]any
	WeaponData2 map[string]any

	// perform-stage actor state
	DB              *modstore.DB
	Ms              *modstore.Actor
	Output          map[string]any
	ActiveSkillList []*ActiveSkill
	MainSkill       *ActiveSkill
}

// Buff is one activeSkill.buffList entry: scalar keys as a bag plus the
// separated modifiers.
type Buff struct {
	KV      map[string]any
	ModList []*modparser.Mod
}

// modFlagByName is ModFlag[name] (getWeaponFlags looks flags up by the
// weaponTypeInfo flag string).
var modFlagByName = func() map[string]int64 {
	out := map[string]int64{}
	rv := reflect.ValueOf(modparser.ModFlag)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		out[rt.Field(i).Name] = rv.Field(i).Int()
	}
	return out
}()

// getWeaponFlags ports the local getWeaponFlags: nil flags means the
// weapon is unusable; info is non-nil whenever the type is known.
func (env *Env) getWeaponFlags(weaponData map[string]any, weaponTypes []map[string]bool) (*int64, *data.WeaponTypeInfo) {
	info, ok := env.Data.WeaponTypeInfo[str(weaponData["type"])]
	if !ok {
		return nil, nil
	}
	for _, types := range weaponTypes {
		if !types[str(weaponData["type"])] &&
			(!truthy(weaponData["countsAsAll1H"]) || !(types["Claw"] || types["Dagger"] || types["One Handed Axe"] || types["One Handed Mace"] || types["One Handed Sword"])) {
			return nil, &info
		}
	}
	flags := modFlagByName[info.Flag]
	if truthy(weaponData["countsAsAll1H"]) {
		flags = modparser.ModFlag.Axe | modparser.ModFlag.Claw | modparser.ModFlag.Dagger | modparser.ModFlag.Mace | modparser.ModFlag.Sword
	}
	if str(weaponData["type"]) != "None" {
		flags |= modparser.ModFlag.Weapon
		if info.OneHand {
			flags |= modparser.ModFlag.Weapon1H
		} else {
			flags |= modparser.ModFlag.Weapon2H
		}
		if info.Melee {
			flags |= modparser.ModFlag.WeaponMelee
		} else {
			flags |= modparser.ModFlag.WeaponRanged
		}
	}
	return &flags, &info
}

// mergeLevelMod ports the local mergeLevelMod, without the instance cache
// (the cache only dedups identical copies).
func mergeLevelMod(modList *modstore.List, mod *modparser.Mod, value *float64) {
	if value == nil {
		modList.AddMod(mod)
		return
	}
	newMod := modparser.CopyMod(mod)
	switch v := newMod.Value.(type) {
	case modparser.Tag:
		if inner, ok := v["mod"].(*modparser.Mod); ok {
			innerCopy := modparser.CopyMod(inner)
			innerCopy.Value = *value
			v["mod"] = innerCopy
		} else {
			v["value"] = *value
		}
	default:
		newMod.Value = *value
	}
	modList.AddMod(newMod)
}

// statMapKV reads a statMap scale key from an entry or group KV.
func statMapScale(kv map[string]any, statValue float64) *float64 {
	if v, ok := kv["value"]; ok && truthy(v) {
		n := anyNum(v)
		return &n
	}
	mult, div, base := 1.0, 1.0, 0.0
	scaled := false
	if v, ok := kv["mult"]; ok && truthy(v) {
		mult = anyNum(v)
		scaled = true
	}
	if v, ok := kv["div"]; ok && truthy(v) {
		div = anyNum(v)
		scaled = true
	}
	if v, ok := kv["base"]; ok && truthy(v) {
		base = anyNum(v)
		scaled = true
	}
	_ = scaled
	n := statValue*mult/div + base
	return &n
}

// mergeSkillInstanceMods ports calcs.mergeSkillInstanceMods with SORTED
// stat iteration (matching the dump-side replacement).
func (env *Env) mergeSkillInstanceMods(modList *modstore.List, skillEffect *ActiveEffect, extraStats []any) {
	ValidateGemLevel(skillEffect)
	grantedEffect := skillEffect.GrantedEffect
	stats := BuildSkillInstanceStats(env.Data, skillEffect, grantedEffect)
	if len(extraStats) > 0 {
		for _, sv := range extraStats {
			tag, _ := sv.(modparser.Tag)
			stats[str(tag["key"])] += anyNum(tag["value"])
		}
	}
	statKeys := make([]string, 0, len(stats))
	for stat := range stats {
		statKeys = append(statKeys, stat)
	}
	sort.Strings(statKeys)
	for _, stat := range statKeys {
		statValue := stats[stat]
		mapEntry := env.statMapLookup(grantedEffect, stat)
		if mapEntry == nil {
			continue
		}
		for _, modOrGroup := range mapEntry.Mods {
			switch m := modOrGroup.(type) {
			case *modparser.Mod:
				mergeLevelMod(modList, m, statMapScale(mapEntry.KV, statValue))
			case *modparser.D:
				if m.KV["name"] != nil {
					panic("calc: D-shaped statMap mod reached mergeLevelMod (flags-slot-tag artifact) for stat " + stat)
				}
				// a group: its own scale over its member mods
				for _, gv := range m.Arr {
					gm, ok := gv.(*modparser.Mod)
					if !ok {
						panic("calc: unexpected statMap group member for stat " + stat)
					}
					mergeLevelMod(modList, gm, statMapScale(m.KV, statValue))
				}
			default:
				panic("calc: unexpected statMap entry shape for stat " + stat)
			}
		}
	}
	for _, bm := range grantedEffect.BaseMods {
		mod, ok := bm.(*modparser.Mod)
		if !ok {
			panic("calc: non-mod baseMods entry (unported skill callback?)")
		}
		modList.AddMod(mod)
	}
}

// setFlag keeps the skillFlags set-of-true invariant (Lua stores true or
// removes the key; false is never stored by these paths).
func setFlag(flags map[string]bool, k string, v bool) {
	if v {
		flags[k] = true
	}
}

func lvlExtra(level *data.SkillLevel, key string) (float64, bool) {
	if level == nil {
		return 0, false
	}
	v, ok := level.Extra[key]
	return v, ok
}

// buildActiveSkillModList ports calcs.buildActiveSkillModList.
func (env *Env) buildActiveSkillModList(activeSkill *ActiveSkill) {
	d := env.Data
	skillTypes := activeSkill.SkillTypes
	skillFlags := activeSkill.SkillFlags
	activeEffect := activeSkill.ActiveEffect
	activeGrantedEffect := activeEffect.GrantedEffect

	// Set mode flags
	setFlag(skillFlags, "buffs", env.ModeBuffs)
	setFlag(skillFlags, "combat", env.ModeCombat)
	setFlag(skillFlags, "effective", env.ModeEffective)

	// Handle multipart skills
	activeGemParts, _ := activeGrantedEffect.Custom["parts"].([]any)
	if len(activeGemParts) > 1 {
		cur := anyNum(activeEffect.SrcInstance.KV["skillPart"])
		if cur == 0 {
			cur = 1
		}
		if cur > float64(len(activeGemParts)) {
			cur = float64(len(activeGemParts))
		}
		activeEffect.SrcInstance.KV["skillPart"] = cur
		activeSkill.SkillPart = cur
		part, _ := activeGemParts[int(cur)-1].(map[string]any)
		partKeys := make([]string, 0, len(part))
		for k := range part {
			partKeys = append(partKeys, k)
		}
		sort.Strings(partKeys) // pairs order; only true/false writes, order-free
		for _, k := range partKeys {
			if part[k] == true {
				skillFlags[k] = true
			} else if part[k] == false {
				delete(skillFlags, k)
			}
		}
		activeSkill.SkillPartName = str(part["name"])
		skillFlags["multiPart"] = true
	} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		delete(activeEffect.SrcInstance.KV, "skillPart")
		delete(activeEffect.SrcInstance.KV, "skillPartCalcs")
	}

	w2Item := activeSkill.Actor.ItemList["Weapon 2"]
	if (skillTypes[modparser.SkillType.RequiresShield] || skillFlags["shieldAttack"]) && activeSkill.SummonSkill == nil &&
		(w2Item == nil || w2Item.ItemType() != "Shield") {
		// Skill requires a shield to be equipped
		skillFlags["disable"] = true
		activeSkill.DisableReason = "This skill requires a Shield"
	}

	if skillFlags["shieldAttack"] {
		// Special handling for Spectral Shield Throw
		skillFlags["weapon2Attack"] = true
		zero := int64(0)
		activeSkill.Weapon2Flags = &zero
	} else {
		// Set weapon flags
		if skillFlags["forceSourceWeapon"] && activeSkill.SocketGroup != nil && activeSkill.SocketGroup.SourceItem != nil {
			// Some item-granted attacks must use the weapon that grants them.
			// The reference assigns match(...)~=nil — false is stored.
			sourceSlot := str(activeSkill.SocketGroup.KV["slot"])
			skillFlags["forceMainHand"] = strings.HasPrefix(sourceSlot, "Weapon 1")
			skillFlags["forceOffHand"] = strings.HasPrefix(sourceSlot, "Weapon 2")
		}
		var weaponTypes []map[string]bool
		if activeGrantedEffect.WeaponTypes != nil {
			weaponTypes = append(weaponTypes, activeGrantedEffect.WeaponTypes)
		}
		for _, skillEffect := range activeSkill.EffectList {
			if skillEffect.GrantedEffect.Support && skillEffect.GrantedEffect.WeaponTypes != nil {
				weaponTypes = append(weaponTypes, skillEffect.GrantedEffect.WeaponTypes)
			}
		}
		var weapon1Flags *int64
		var weapon1Info *data.WeaponTypeInfo
		if !skillFlags["forceOffHand"] {
			weapon1Flags, weapon1Info = env.getWeaponFlags(activeSkill.Actor.WeaponData1, weaponTypes)
		}
		if weapon1Flags == nil && activeSkill.SummonSkill != nil {
			// Minion skills seem to ignore weapon types
			f := modFlagByName[d.WeaponTypeInfo["None"].Flag]
			info := d.WeaponTypeInfo["None"]
			weapon1Flags, weapon1Info = &f, &info
		}
		if weapon1Flags != nil {
			if skillFlags["attack"] || skillFlags["dotFromAttack"] {
				activeSkill.Weapon1Flags = weapon1Flags
				skillFlags["weapon1Attack"] = true
				if weapon1Info.Melee && skillFlags["melee"] {
					delete(skillFlags, "projectile")
				} else if !weapon1Info.Melee && skillFlags["projectile"] {
					delete(skillFlags, "melee")
				}
			}
		} else if (skillTypes[modparser.SkillType.DualWieldOnly] || skillFlags["forceMainHand"] || weapon1Info != nil) && activeSkill.SummonSkill == nil {
			// Skill requires a compatible main hand weapon
			skillFlags["disable"] = true
			activeSkill.DisableReason = "Main Hand weapon is not usable with this skill"
		}
		if !skillFlags["forceMainHand"] {
			weapon2Flags, weapon2Info := env.getWeaponFlags(activeSkill.Actor.WeaponData2, weaponTypes)
			if weapon2Flags != nil {
				if skillTypes[modparser.SkillType.DualWieldRequiresDifferentTypes] &&
					str(activeSkill.Actor.WeaponData1["type"]) == str(activeSkill.Actor.WeaponData2["type"]) &&
					!(truthy(activeSkill.Actor.WeaponData2["countsAsAll1H"]) || truthy(activeSkill.Actor.WeaponData1["countsAsAll1H"])) {
					skillFlags["disable"] = true
					if activeSkill.DisableReason == "" {
						activeSkill.DisableReason = "Weapon Types Need to be Different"
					}
				} else if skillFlags["attack"] || skillFlags["dotFromAttack"] {
					activeSkill.Weapon2Flags = weapon2Flags
					skillFlags["weapon2Attack"] = true
				}
			} else if (skillTypes[modparser.SkillType.DualWieldOnly] || weapon2Info != nil) && activeSkill.SummonSkill == nil {
				skillFlags["disable"] = true
				if activeSkill.DisableReason == "" {
					activeSkill.DisableReason = "Off Hand weapon is not usable with this skill"
				}
			} else if skillFlags["disable"] {
				activeSkill.DisableReason = "No usable weapon equipped"
			}
		}
		if skillFlags["attack"] {
			setFlag(skillFlags, "bothWeaponAttack", skillFlags["weapon1Attack"] && skillFlags["weapon2Attack"])
		}
	}

	// Apply stat-map flagged skill flags
	for stat, statValue := range BuildSkillInstanceStats(d, activeEffect, activeGrantedEffect) {
		mapEntry := env.statMapLookup(activeGrantedEffect, stat)
		if statValue != 0 && mapEntry != nil {
			if sf := str(mapEntry.KV["skillFlag"]); sf != "" {
				skillFlags[sf] = true
			}
		}
	}
	// Build skill mod flag set
	var skillModFlags int64
	if skillFlags["hit"] {
		skillModFlags |= modparser.ModFlag.Hit
	}
	if skillFlags["attack"] {
		skillModFlags |= modparser.ModFlag.Attack
	} else {
		skillModFlags |= modparser.ModFlag.Cast
		if skillFlags["spell"] {
			skillModFlags |= modparser.ModFlag.Spell
		}
	}
	if skillFlags["melee"] {
		skillModFlags |= modparser.ModFlag.Melee
	} else if skillFlags["projectile"] {
		skillModFlags |= modparser.ModFlag.Projectile
		skillFlags["chaining"] = true
	}
	if skillFlags["area"] {
		skillModFlags |= modparser.ModFlag.Area
	}

	// Build skill keyword flag set
	var skillKeywordFlags int64
	if skillFlags["hit"] {
		skillKeywordFlags |= modparser.KeywordFlag.Hit
	}
	for _, e := range []struct {
		st int64
		kw int64
	}{
		{modparser.SkillType.Aura, modparser.KeywordFlag.Aura},
		{modparser.SkillType.AppliesCurse, modparser.KeywordFlag.Curse},
		{modparser.SkillType.Warcry, modparser.KeywordFlag.Warcry},
		{modparser.SkillType.Movement, modparser.KeywordFlag.Movement},
		{modparser.SkillType.Vaal, modparser.KeywordFlag.Vaal},
		{modparser.SkillType.Lightning, modparser.KeywordFlag.Lightning},
		{modparser.SkillType.Cold, modparser.KeywordFlag.Cold},
		{modparser.SkillType.Fire, modparser.KeywordFlag.Fire},
		{modparser.SkillType.Chaos, modparser.KeywordFlag.Chaos},
		{modparser.SkillType.Physical, modparser.KeywordFlag.Physical},
	} {
		if skillTypes[e.st] {
			skillKeywordFlags |= e.kw
		}
	}
	if skillFlags["weapon1Attack"] && activeSkill.Weapon1Flags != nil && *activeSkill.Weapon1Flags&modparser.ModFlag.Bow != 0 {
		skillKeywordFlags |= modparser.KeywordFlag.Bow
	}
	if skillFlags["brand"] {
		skillKeywordFlags |= modparser.KeywordFlag.Brand
	}
	if skillFlags["arrow"] {
		skillKeywordFlags |= modparser.KeywordFlag.Arrow
	}
	if skillFlags["totem"] {
		skillKeywordFlags |= modparser.KeywordFlag.Totem
	} else if skillFlags["trap"] {
		skillKeywordFlags |= modparser.KeywordFlag.Trap
	} else if skillFlags["mine"] {
		skillKeywordFlags |= modparser.KeywordFlag.Mine
	} else if !skillTypes[modparser.SkillType.Triggered] {
		skillFlags["selfCast"] = true
	}
	if skillTypes[modparser.SkillType.Attack] {
		skillKeywordFlags |= modparser.KeywordFlag.Attack
	}
	if skillTypes[modparser.SkillType.Spell] && !skillFlags["cast"] {
		skillKeywordFlags |= modparser.KeywordFlag.Spell
	}

	// Get skill totem ID for totem skills
	if skillFlags["totem"] {
		if activeGrantedEffect.SkillTotemId != nil {
			activeSkill.SkillTotemId = activeGrantedEffect.SkillTotemId
		} else {
			id := 1.0
			if activeGrantedEffect.Color == 2 {
				id = 2
			} else if activeGrantedEffect.Color == 3 {
				id = 3
			}
			activeSkill.SkillTotemId = &id
		}
	}

	// Calculate melee/projectile distance
	distKey := "projectileDistance"
	if skillFlags["melee"] {
		distKey = "meleeDistance"
	}
	var effectiveRange *float64
	if v, ok := env.ConfigInput[distKey]; ok && truthy(v) {
		n := anyNum(v)
		effectiveRange = &n
	} else if v, ok := env.Build.ConfigPlaceholder[distKey]; ok && truthy(v) {
		n := anyNum(v)
		effectiveRange = &n
	}

	// Build config structure for modifier searches
	cfgFlags := skillModFlags
	if activeSkill.Weapon1Flags != nil {
		cfgFlags = skillModFlags | *activeSkill.Weapon1Flags
	} else if activeSkill.Weapon2Flags != nil {
		cfgFlags = skillModFlags | *activeSkill.Weapon2Flags
	}
	cfgSkillTypes := map[float64]bool{}
	for k, v := range activeSkill.SkillTypes {
		cfgSkillTypes[float64(k)] = v
	}
	kf := skillKeywordFlags
	activeSkill.SkillCfg = &modstore.Cfg{
		Flags:              &cfgFlags,
		KeywordFlags:       &kf,
		SkillName:          strings.TrimPrefix(activeGrantedEffect.Name, "Vaal "),
		SkillGrantedEffect: &modstore.GrantedEffectRef{Id: activeGrantedEffect.Id, BaseFlags: activeGrantedEffect.BaseFlags},
		SkillPart:          activeSkill.SkillPart,
		SkillTypes:         cfgSkillTypes,
		SkillCond:          map[string]bool{},
	}
	if activeEffect.GemData != nil {
		// typed-nil guard: a nil *data.Gem in the any field would defeat
		// the eval's skillGem nil check
		activeSkill.SkillCfg.SkillGem = activeEffect.GemData
	}
	if activeSkill.SummonSkill != nil {
		activeSkill.SkillCfg.SummonSkillName = activeSkill.SummonSkill.ActiveEffect.GrantedEffect.Name
	}
	if env.ModeEffective {
		activeSkill.SkillCfg.SkillDist = effectiveRange
	}
	if activeSkill.SlotName != "" {
		activeSkill.SkillCfg.SlotName = activeSkill.SlotName
	} else if activeEffect.GemCfg != nil {
		activeSkill.SkillCfg.SlotName = activeEffect.GemCfg.SlotName
	}
	if activeEffect.GemCfg != nil {
		activeSkill.SkillCfg.SocketColor = activeEffect.GemCfg.SocketColor
		activeSkill.SkillCfg.SocketNum = activeEffect.GemCfg.SocketNum
	}

	if skillFlags["weapon1Attack"] {
		// copyTable(skillCfg, true) + skillCond overlay (the reference's
		// metatable __index; the base skillCond is empty here and later
		// writes go through perform-time cfgs)
		cfg := *activeSkill.SkillCfg
		cfg.SkillCond = map[string]bool{"MainHandAttack": true}
		f := skillModFlags | *activeSkill.Weapon1Flags
		cfg.Flags = &f
		activeSkill.Weapon1Cfg = &cfg
	}
	if skillFlags["weapon2Attack"] {
		cfg := *activeSkill.SkillCfg
		cfg.SkillCond = map[string]bool{"OffHandAttack": true}
		f := skillModFlags | *activeSkill.Weapon2Flags
		cfg.Flags = &f
		activeSkill.Weapon2Cfg = &cfg
	}

	// Initialise skill modifier list
	skillModList := modstore.NewList(activeSkill.Actor.DB)
	activeSkill.SkillModList = skillModList
	activeSkill.BaseSkillModList = skillModList

	if activeSkill.Actor.MinionData != nil {
		panic("calc: minion damage fixup unported")
	}

	// Mods which apply curses are not disabled by Gruthkul's Pelt
	curseApplicationSkill := activeSkill.SocketGroup != nil && activeSkill.SocketGroup.SourceItem != nil &&
		skillFlags["curse"] && activeEffect.SrcInstance != nil &&
		truthy(activeEffect.SrcInstance.KV["noSupports"]) && truthy(activeEffect.SrcInstance.KV["triggered"])
	if skillModList.Flag(activeSkill.SkillCfg, "DisableSkill") &&
		!(skillModList.Flag(activeSkill.SkillCfg, "EnableSkill") || (curseApplicationSkill && skillModList.Flag(nil, "ForceEnableCurseApplication"))) {
		skillFlags["disable"] = true
		activeSkill.DisableReason = "Skills of this type are disabled"
	}

	if skillFlags["disable"] {
		for k := range skillFlags {
			delete(skillFlags, k)
		}
		skillFlags["disable"] = true
		ValidateGemLevel(activeEffect)
		activeEffect.GrantedEffectLevel = activeGrantedEffect.Levels[activeEffect.Level]
		return
	}

	// Add support gem modifiers to skill mod list
	for _, skillEffect := range activeSkill.EffectList {
		if !skillEffect.GrantedEffect.Support {
			continue
		}
		env.mergeSkillInstanceMods(skillModList, skillEffect, nil)
		level := skillEffect.GrantedEffect.Levels[skillEffect.Level]
		if v, ok := lvlExtra(level, "manaMultiplier"); ok {
			skillModList.AddMod(newMod("SupportManaMultiplier", "MORE", v, skillEffect.GrantedEffect.ModSource))
		}
		if v, ok := lvlExtra(level, "manaReservationPercent"); ok {
			activeSkill.SkillData["manaReservationPercent"] = v
		}
		if skillEffect.GrantedEffect.AddSkillTypes != nil && !skillFlags["disable"] && skillEffect.GrantedEffect.IsTrigger {
			if activeSkill.TriggeredBy != nil {
				skillFlags["disable"] = true
				activeSkill.DisableReason = "This skill is supported by more than one trigger"
			} else {
				activeSkill.TriggeredBy = skillEffect
			}
		}
		if v, ok := lvlExtra(level, "PvPDamageMultiplier"); ok {
			skillModList.AddMod(newMod("PvpDamageMultiplier", "MORE", v, skillEffect.GrantedEffect.ModSource))
		}
		if v, ok := lvlExtra(level, "storedUses"); ok {
			activeSkill.SkillData["storedUses"] = v
		}
		if v, ok := lvlExtra(level, "vaalStoredUses"); ok {
			// reference precedence: a or (0 + b)
			if _, has := activeSkill.SkillData["storedUses"]; !has {
				activeSkill.SkillData["storedUses"] = 0 + v
			}
		}
	}

	// Apply gem/quality modifiers from support gems
	gemLevel := activeEffect.Level
	gemQuality := activeEffect.Quality
	if activeEffect.SrcInstance != nil {
		gemLevel = anyNum(activeEffect.SrcInstance.KV["level"])
		gemQuality = anyNum(activeEffect.SrcInstance.KV["quality"])
	}
	skillModList.AddMod(newMod("GemLevel", "BASE", gemLevel, "Max Level"))
	skillModList.AddMod(newMod("GemQuality", "BASE", gemQuality, "Max Quality"))
	socketMatches := activeEffect.SrcInstance != nil && truthy(activeEffect.SrcInstance.KV["matchesSocket"])
	if socketMatches {
		skillModList.AddMod(newMod("GemSocketQuality", "BASE", d.Misc.MatchingSocketQualityBonus, "Socket Quality"))
	}
	for _, supportProperty := range skillModList.Tabulate("LIST", activeSkill.SkillCfg, "SupportedGemProperty") {
		value, _ := supportProperty.Value.(modparser.Tag)
		if str(value["keyword"]) == "grants_active_skill" && activeEffect.GemData != nil && !activeEffect.GemData.Tags["support"] {
			key := str(value["key"])
			v := anyNum(value["value"])
			if key == "quality" {
				activeEffect.SupportQuality += v
			}
			switch key {
			case "level":
				activeEffect.Level += v
			case "quality":
				activeEffect.Quality += v
			default:
				if activeEffect.Extra == nil {
					activeEffect.Extra = map[string]float64{}
				}
				activeEffect.Extra[key] += v
			}
			args := []any{supportProperty.Mod.Source}
			if len(supportProperty.Mod.Tags) > 0 && supportProperty.Mod.Tags[0] != nil {
				args = append(args, supportProperty.Mod.Tags[0])
			}
			skillModList.AddMod(newMod("GemSupport"+firstUpper(key), "BASE", v, args...))
		}
	}

	for _, gemProperty := range activeEffect.GemPropertyInfo {
		value, _ := gemProperty.Value.(modparser.Tag)
		args := []any{gemProperty.Mod.Source}
		if len(gemProperty.Mod.Tags) > 0 && gemProperty.Mod.Tags[0] != nil {
			args = append(args, gemProperty.Mod.Tags[0])
		}
		skillModList.AddMod(newMod("GemItem"+firstUpper(str(value["key"])), "BASE", anyNum(value["value"]), args...))
	}

	// Add active gem modifiers
	activeEffect.ActorLevel = nil // actor.minionData and actor.level
	env.mergeSkillInstanceMods(skillModList, activeEffect, skillModList.List(activeSkill.SkillCfg, "ExtraSkillStat"))
	activeEffect.GrantedEffectLevel = activeGrantedEffect.Levels[activeEffect.Level]

	// Add extra modifiers from granted effect level
	level := activeEffect.GrantedEffectLevel
	if v, ok := lvlExtra(level, "critChance"); ok {
		activeSkill.SkillData["CritChance"] = v
	}
	if v, ok := lvlExtra(level, "damageMultiplier"); ok {
		skillModList.AddMod(newMod("Damage", "MORE", v, activeGrantedEffect.ModSource, modparser.ModFlag.Attack))
	}
	if v, ok := lvlExtra(level, "attackTime"); ok {
		activeSkill.SkillData["attackTime"] = v
	}
	if v, ok := lvlExtra(level, "attackSpeedMultiplier"); ok {
		activeSkill.SkillData["attackSpeedMultiplier"] = v
	}
	if v, ok := lvlExtra(level, "cooldown"); ok {
		activeSkill.SkillData["cooldown"] = v
	}
	if v, ok := lvlExtra(level, "storedUses"); ok {
		activeSkill.SkillData["storedUses"] = v
	}
	if v, ok := lvlExtra(level, "vaalStoredUses"); ok {
		if _, has := activeSkill.SkillData["storedUses"]; !has {
			activeSkill.SkillData["storedUses"] = 0 + v
		}
	}
	if v, ok := lvlExtra(level, "soulPreventionDuration"); ok {
		activeSkill.SkillData["soulPreventionDuration"] = v
	}
	if v, ok := lvlExtra(level, "PvPDamageMultiplier"); ok {
		skillModList.AddMod(newMod("PvpDamageMultiplier", "MORE", v, activeGrantedEffect.ModSource))
	}

	// Add extra modifiers from other sources
	activeSkill.ExtraSkillModList = nil
	for _, v := range skillModList.List(activeSkill.SkillCfg, "ExtraSkillMod") {
		tag, _ := v.(modparser.Tag)
		mod, _ := tag["mod"].(*modparser.Mod)
		if mod == nil {
			continue
		}
		skillModList.AddMod(mod)
		activeSkill.ExtraSkillModList = append(activeSkill.ExtraSkillModList, mod)
	}

	// Find totem level
	if skillFlags["totem"] {
		if v, ok := lvlExtra(activeEffect.GrantedEffectLevel, "levelRequirement"); ok {
			activeSkill.SkillData["totemLevel"] = v
		}
	}

	// Add active mine multiplier
	if skillFlags["mine"] {
		if activeEffect.SrcInstance != nil {
			if v, ok := activeEffect.SrcInstance.KV["skillMineCount"]; ok && truthy(v) {
				count := anyNum(v)
				activeSkill.ActiveMineCount = &count
				if count > 0 {
					skillModList.AddMod(newMod("Multiplier:ActiveMineCount", "BASE", count, "Base"))
					existing := env.EnemyDB.Multipliers["ActiveMineCount"]
					if count > existing {
						existing = count
					}
					env.EnemyDB.Multipliers["ActiveMineCount"] = existing
				}
			}
		}
	} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		delete(activeEffect.SrcInstance.KV, "skillMineCountCalcs")
		delete(activeEffect.SrcInstance.KV, "skillMineCount")
	}

	// Stages
	noPotentialStage := true
	for _, pv := range activeGemParts {
		if part, ok := pv.(map[string]any); ok && truthy(part["stages"]) {
			noPotentialStage = false
			break
		}
	}
	strippedName := stripSpaces(activeGrantedEffect.Name)
	limit := skillModList.Sum("BASE", activeSkill.SkillCfg, "Multiplier:"+strippedName+"MaxStages")
	if limit > 0 {
		activeSkill.SkillData["stagesMax"] = limit
		skillFlags["multiStage"] = true
		// srcInstance.skillStageCount or stagesMax or 1 (Lua truthiness:
		// a present 0 wins over the fallback)
		cur := limit
		if activeEffect.SrcInstance != nil {
			if v, ok := activeEffect.SrcInstance.KV["skillStageCount"]; ok && truthy(v) {
				cur = anyNum(v)
			}
		}
		minStage := 1 + skillModList.Sum("BASE", activeSkill.SkillCfg, "Multiplier:"+strippedName+"MinimumStage")
		if cur < minStage {
			cur = minStage
		}
		activeSkill.ActiveStageCount = &cur
		if cur > 0 {
			skillModList.AddMod(newMod("Multiplier:"+strippedName+"Stage", "BASE", minFloat(limit, cur), "Base"))
			cur = cur - 1
			activeSkill.ActiveStageCount = &cur
			skillModList.AddMod(newMod("Multiplier:"+strippedName+"StageAfterFirst", "BASE", minFloat(limit-1, cur), "Base"))
		}
	} else if noPotentialStage && activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		delete(activeEffect.SrcInstance.KV, "skillStageCountCalcs")
		delete(activeEffect.SrcInstance.KV, "skillStageCount")
	}

	// Extract skill data
	for _, v := range env.ModDB.List(activeSkill.SkillCfg, "SkillData") {
		tag, _ := v.(modparser.Tag)
		activeSkill.SkillData[str(tag["key"])] = tag["value"]
	}
	for _, v := range skillModList.List(activeSkill.SkillCfg, "SkillData") {
		tag, _ := v.(modparser.Tag)
		activeSkill.SkillData[str(tag["key"])] = tag["value"]
	}

	// Create minion
	var minionList []string
	isSpectre := false
	minionSupportLevel := map[string]float64{}
	if mlv, ok := activeGrantedEffect.Custom["minionList"]; ok {
		ml, _ := mlv.([]any)
		if len(ml) > 0 {
			for _, m := range ml {
				minionList = append(minionList, str(m))
			}
		} else {
			minionList = append([]string{}, env.Build.SpectreList...)
			isSpectre = true
		}
	}
	for _, skillEffect := range activeSkill.EffectList {
		if !skillEffect.GrantedEffect.Support {
			continue
		}
		if amv, ok := skillEffect.GrantedEffect.Custom["addMinionList"]; ok {
			am, _ := amv.([]any)
			for _, mv := range am {
				minionType := str(mv)
				found := false
				for _, existing := range minionList {
					if existing == minionType {
						found = true
						break
					}
				}
				if !found {
					lvl := skillEffect.GrantedEffect.Levels[skillEffect.Level]
					req, _ := lvlExtra(lvl, "levelRequirement")
					minionSupportLevel[minionType] = req
					minionList = append(minionList, minionType)
				}
			}
		}
	}
	if minionList == nil {
		minionList = []string{}
	}
	activeSkill.MinionList = minionList
	if len(minionList) > 0 && activeSkill.Actor.MinionData == nil {
		// select minion type from srcInstance state
		minionType := minionList[0]
		if activeEffect.SrcInstance != nil {
			if sel := str(activeEffect.SrcInstance.KV["skillMinion"]); sel != "" {
				for _, m := range minionList {
					if m == sel {
						minionType = sel
						break
					}
				}
			}
			activeEffect.SrcInstance.KV["skillMinion"] = minionType
		}
		minionData := d.Minions[minionType]
		if minionData == nil {
			panic("calc: unknown minion type " + minionType)
		}
		minion := &Minion{Type: minionType, MinionData: minionData, Hostile: truthy(minionData.Hostile)}
		activeSkill.Minion = minion
		skillFlags["haveMinion"] = true
		if minion.Hostile {
			minion.Parent = env.Enemy
			minion.EnemyActor = env.Player
		} else {
			minion.Parent = env.Player
			minion.EnemyActor = env.Enemy
		}
		sd := func(key string) any { return activeSkill.SkillData[key] }
		lvlReq, _ := lvlExtra(activeEffect.GrantedEffectLevel, "levelRequirement")
		if truthy(sd("minionLevelIsEnemyLevel")) {
			minion.Level = env.EnemyLevel
		} else if truthy(sd("minionLevelIsPlayerLevel")) {
			// min(build.characterLevel [or skillData.minionLevel or lvlReq —
			// env.build is always set here], minionLevelIsPlayerLevel)
			minion.Level = math.Min(env.Build.CharacterLevel, anyNum(sd("minionLevelIsPlayerLevel")))
		} else if v, ok := minionSupportLevel[minionType]; ok {
			minion.Level = v
		} else if truthy(sd("minionLevel")) {
			minion.Level = anyNum(sd("minionLevel"))
		} else {
			minion.Level = lvlReq
		}
		minion.Level = math.Min(math.Max(minion.Level, 1), 100)
		minion.ItemList = map[string]modstore.Item{}
		if uses, ok := activeGrantedEffect.Custom["minionUses"].(map[string]any); ok {
			minion.Uses = uses
		}
		if minion.Hostile {
			minion.LifeTable = d.MonsterLifeTable
		} else if minionData.LifeScaling == "AltLife1" {
			minion.LifeTable = d.MonsterLifeTable2
		} else if minionData.LifeScaling == "AltLife2" {
			minion.LifeTable = d.MonsterLifeTable3
		} else if isSpectre {
			minion.LifeTable = d.MonsterLifeTable
		} else {
			minion.LifeTable = d.MonsterAllyLifeTable
		}
		attackTime := minionData.AttackTime
		damageTable := d.MonsterAllyDamageTable
		if isSpectre || minion.Hostile {
			damageTable = d.MonsterDamageTable
		}
		damage := damageTable[int(minion.Level)-1] * minionData.Damage
		if !minionData.BaseDamageIgnoresAttackSpeed {
			damage = damage * attackTime
		}
		if _, ok := activeGrantedEffect.Custom["minionHasItemSet"]; ok {
			cur := 0
			if activeEffect.SrcInstance != nil {
				cur = int(anyNum(activeEffect.SrcInstance.KV["skillMinionItemSet"]))
			}
			if env.Build.ItemsTab.ItemSets[cur] == nil {
				cur = int(env.Build.ItemsTab.ItemSetOrderList[0])
				if activeEffect.SrcInstance != nil {
					activeEffect.SrcInstance.KV["skillMinionItemSet"] = float64(cur)
				}
			}
			minion.ItemSet = env.Build.ItemsTab.ItemSets[cur]
		} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
			delete(activeEffect.SrcInstance.KV, "skillMinionItemSetCalcs")
			delete(activeEffect.SrcInstance.KV, "skillMinionItemSet")
		}
		if (truthy(sd("minionUseBowAndQuiver")) && str(env.Player.WeaponData1["type"]) == "Bow") || truthy(sd("minionUseMainHandWeapon")) {
			minion.WeaponData1 = env.Player.WeaponData1
		} else if env.TheIronMass != nil && minionType == "RaisedSkeleton" {
			minion.WeaponData1 = env.Player.WeaponData1
		} else {
			wtype := "None"
			if minionData.WeaponType1 != nil {
				wtype = *minionData.WeaponType1
			}
			minion.WeaponData1 = map[string]any{
				"type":        wtype,
				"AttackRate":  1 / attackTime,
				"CritChance":  5.0,
				"PhysicalMin": round(damage * (1 - minionData.DamageSpread)),
				"PhysicalMax": round(damage * (1 + minionData.DamageSpread)),
				"range":       minionData.AttackRange,
			}
		}
		minion.WeaponData2 = map[string]any{}
		if minion.Uses != nil {
			setSlot := func(base string) string {
				if minion.ItemSet != nil && minion.ItemSet.UseSecondWeaponSet != nil && *minion.ItemSet.UseSecondWeaponSet {
					return base + " Swap"
				}
				return base
			}
			if truthy(minion.Uses["Weapon 1"]) {
				if minion.ItemSet != nil {
					item := env.Build.ItemsTab.Items[int(minion.ItemSet.Slots[setSlot("Weapon 1")])]
					if item != nil && item.WeaponData != nil {
						minion.WeaponData1 = item.WeaponData[1]
					}
				} else {
					minion.WeaponData1 = env.Player.WeaponData1
				}
			}
			if truthy(minion.Uses["Weapon 2"]) {
				if minion.ItemSet != nil {
					item := env.Build.ItemsTab.Items[int(minion.ItemSet.Slots[setSlot("Weapon 2")])]
					if item != nil && item.WeaponData != nil {
						minion.WeaponData2 = item.WeaponData[2]
					}
				} else {
					minion.WeaponData2 = env.Player.WeaponData2
				}
			}
		}
		if !isSpectre && skillModList.Flag(activeSkill.SkillCfg, "NonSpectreMinionsUseParentMainHandAttackTime") &&
			truthy(env.Player.WeaponData1["AttackRate"]) && anyNum(env.Player.WeaponData1["AttackRate"]) > 0 {
			// copy before replacing only the base attack rate
			cp := map[string]any{}
			for k, v := range minion.WeaponData1 {
				cp[k] = v
			}
			cp["AttackRate"] = env.Player.WeaponData1["AttackRate"]
			minion.WeaponData1 = cp
		}
	} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		for _, k := range []string{"skillMinionCalcs", "skillMinion", "skillMinionItemSetCalcs", "skillMinionItemSet", "skillMinionSkill", "skillMinionSkillCalcs"} {
			delete(activeEffect.SrcInstance.KV, k)
		}
	}

	// Separate global effect modifiers
	i := 0
	for i < len(skillModList.Mods) {
		mod := skillModList.Mods[i]
		var effectType, effectName string
		var effectTag modparser.Tag
		for _, tv := range mod.Tags {
			if tv == nil {
				break
			}
			if tag, ok := tv.(modparser.Tag); ok && tag["type"] == "GlobalEffect" {
				effectType = str(tag["effectType"])
				effectName = str(tag["effectName"])
				if effectName == "" {
					effectName = activeGrantedEffect.Name
				}
				effectTag = tag
				break
			}
		}
		if effectTag != nil && truthy(effectTag["modCond"]) && !skillModList.GetCondition(str(effectTag["modCond"]), activeSkill.SkillCfg) {
			skillModList.Mods = append(skillModList.Mods[:i], skillModList.Mods[i+1:]...)
		} else if effectType != "" {
			var buff *Buff
			for _, skillBuff := range activeSkill.BuffListTyped {
				if str(skillBuff.KV["type"]) == effectType && str(skillBuff.KV["name"]) == effectName {
					buff = skillBuff
					break
				}
			}
			if buff == nil {
				buff = &Buff{KV: map[string]any{"type": effectType, "name": effectName}}
				for src, dst := range map[string]string{
					"allowTotemBuff":      "allowTotemBuff",
					"effectCond":          "cond",
					"effectEnemyCond":     "enemyCond",
					"effectStackVar":      "stackVar",
					"effectStackLimit":    "stackLimit",
					"effectStackLimitVar": "stackLimitVar",
					"applyNotPlayer":      "applyNotPlayer",
					"applyMinions":        "applyMinions",
				} {
					if v, ok := effectTag[src]; ok && v != nil {
						buff.KV[dst] = v
					}
				}
				if mod.Source == activeGrantedEffect.ModSource {
					// Inherit buff configuration from the active skill
					buff.KV["activeSkillBuff"] = true
					if !truthy(buff.KV["applyNotPlayer"]) {
						setKV(buff.KV, "applyNotPlayer", activeSkill.SkillData["buffNotPlayer"])
					}
					if !truthy(buff.KV["applyMinions"]) {
						setKV(buff.KV, "applyMinions", activeSkill.SkillData["buffMinions"])
					}
					setKV(buff.KV, "applyAllies", activeSkill.SkillData["buffAllies"])
					setKV(buff.KV, "allowTotemBuff", activeSkill.SkillData["allowTotemBuff"])
				}
				activeSkill.BuffListTyped = append(activeSkill.BuffListTyped, buff)
			}
			match := false
			for di, destMod := range buff.ModList {
				if modparser.CompareModParams(mod, destMod) && (destMod.Type == "BASE" || destMod.Type == "INC") {
					cp := modparser.CopyMod(destMod)
					cp.Value = anyNum(cp.Value) + anyNum(mod.Value)
					buff.ModList[di] = cp
					match = true
					break
				}
			}
			if !match {
				buff.ModList = append(buff.ModList, mod)
			}
			skillModList.Mods = append(skillModList.Mods[:i], skillModList.Mods[i+1:]...)
		} else {
			i++
		}
	}

	if len(activeSkill.BuffListTyped) > 0 {
		env.AuxSkillList = append(env.AuxSkillList, activeSkill)
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// firstUpper is str:gsub("^%l", string.upper).
func firstUpper(s string) string {
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

// setKV assigns only non-nil values (Lua nil assignment leaves the key
// absent; false is stored).
func setKV(kv map[string]any, k string, v any) {
	if v != nil {
		kv[k] = v
	}
}

// keywordFlagByName is KeywordFlag[name] for dynamic lookups.
var keywordFlagByName = func() map[string]int64 {
	out := map[string]int64{}
	rv := reflect.ValueOf(modparser.KeywordFlag)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		out[rt.Field(i).Name] = rv.Field(i).Int()
	}
	return out
}()
