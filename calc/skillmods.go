// Port of CalcActiveSkill.lua buildActiveSkillModList (+ getWeaponFlags,
// mergeSkillInstanceMods/mergeLevelMod). Stat iteration is SORTED on both
// sides: the reference's pairs(stats) order is LuaJIT string-hash-random
// per process, so tools/dump_build.lua replaces mergeSkillInstanceMods with
// a sorted-stats replica — a documented divergence from the vanilla app.
// Minion creation panics (needs the minion stage + a minion corpus).
package calc

import (
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/item"
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
	Uses        map[string]bool
	ItemSet     *ItemSetInput
	LifeTable   []float64
	WeaponData1 *item.WeaponData
	WeaponData2 *item.WeaponData

	// perform-stage actor state
	DB              *modstore.DB
	Ms              *modstore.Actor
	Output          modstore.Output
	ActiveSkillList []*ActiveSkill
	MainSkill       *ActiveSkill
}

// Buff is one activeSkill.buffList entry. The Opt flags are plain Lua
// assignments from skillData, so absent, false and true are all distinct
// (the archive comparison sees the key set).
type Buff struct {
	Type, Name      string
	ActiveSkillBuff bool
	ApplyNotPlayer  util.Opt[bool]
	ApplyMinions    util.Opt[bool]
	ApplyAllies     util.Opt[bool]
	AllowTotemBuff  util.Opt[bool]
	Cond            string
	StackVar        string
	StackLimit      util.Opt[float64]
	ModList         []*modparser.Mod
}

// getWeaponFlags ports the local getWeaponFlags. The reference returns
// `flags, info`; the callers only ever consume info's Melee field and its
// non-nilness, so the port returns exactly those: nil flags means the
// weapon is unusable, known reports whether the weapon type was recognized
// at all (the reference's info ~= nil).
func (env *Env) getWeaponFlags(weaponData *item.WeaponData, weaponTypes []map[string]bool) (_ *modparser.ModFlag, melee, known bool) {
	info, ok := data.WeaponTypeInfo[weaponType(weaponData)]
	if !ok {
		return nil, false, false
	}
	for _, types := range weaponTypes {
		if !types[weaponData.Type] &&
			(!weaponData.CountsAsAll1H || !(types["Claw"] || types["Dagger"] || types["One Handed Axe"] || types["One Handed Mace"] || types["One Handed Sword"])) {
			return nil, info.Melee, true
		}
	}
	flags := modparser.ModFlagByName[info.Flag]
	if weaponData.CountsAsAll1H {
		flags = modparser.FlagAxe | modparser.FlagClaw | modparser.FlagDagger | modparser.FlagMace | modparser.FlagSword
	}
	if weaponData.Type != "None" {
		flags |= modparser.FlagWeapon
		if info.OneHand {
			flags |= modparser.FlagWeapon1H
		} else {
			flags |= modparser.FlagWeapon2H
		}
		if info.Melee {
			flags |= modparser.FlagWeaponMelee
		} else {
			flags |= modparser.FlagWeaponRanged
		}
	}
	return &flags, info.Melee, true
}

// mergeLevelMod ports the local mergeLevelMod, without the instance cache
// (the cache only dedups identical copies).
func mergeLevelMod(modList *modstore.List, mod *modparser.Mod, value *float64) {
	if value == nil {
		modList.AddMod(mod)
		return
	}
	newMod := cloneMod(mod)
	switch v := newMod.Value.(type) {
	case modparser.ModRef:
		innerCopy := cloneMod(v.Mod)
		innerCopy.Value = modparser.Num(*value)
		v.Mod = innerCopy
		newMod.Value = v
	case modparser.DataRef:
		v.Value = modparser.Num(*value)
		newMod.Value = v
	case modparser.GemPropertyRef:
		v.Value = opt(*value)
		newMod.Value = v
	default:
		newMod.Value = modparser.Num(*value)
	}
	modList.AddMod(newMod)
}

// statMapScale applies a statMap entry's or group's scale keys to the
// stat value (a present value replaces it; 0 is a value, as in Lua).
func statMapScale(value, mult, div, base util.Opt[float64], statValue float64) *float64 {
	if value.Set {
		return &value.V
	}
	n := statValue*mult.Or(1)/div.Or(1) + base.Or(0)
	return &n
}

// mergeSkillInstanceMods ports calcs.mergeSkillInstanceMods with SORTED
// stat iteration (matching the dump-side replacement).
func (env *Env) mergeSkillInstanceMods(modList *modstore.List, skillEffect *ActiveEffect, extraStats []modparser.Value) {
	ValidateGemLevel(skillEffect)
	grantedEffect := skillEffect.GrantedEffect
	stats := BuildSkillInstanceStats(skillEffect, grantedEffect)
	if len(extraStats) > 0 {
		for _, sv := range extraStats {
			tag, ok := sv.(modparser.DataRef)
			if !ok {
				panic("calc: non-DataRef value in ExtraSkillStat list (the Lua errors)")
			}
			stats[tag.Key] += valueNum(tag.Value)
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
		for _, m := range mapEntry.Mods {
			switch {
			case m.Mod != nil:
				mergeLevelMod(modList, m.Mod, statMapScale(mapEntry.Value, mapEntry.Mult, mapEntry.Div, mapEntry.Base, statValue))
			case m.Group != nil:
				// a group: its own scale over its member mods
				for _, gm := range m.Group.Mods {
					mergeLevelMod(modList, gm, statMapScale(util.Opt[float64]{}, m.Group.Mult, m.Group.Div, util.Opt[float64]{}, statValue))
				}
			case m.Typo != nil:
				// the reference would error on the malformed table
				panic("calc: typo statMap record reached mergeLevelMod for stat " + stat)
			}
		}
	}
	for _, m := range grantedEffect.BaseMods {
		if m.Mod == nil {
			// the reference would error on the malformed table
			panic("calc: typo baseMods record reached the skill mod list of " + grantedEffect.Id)
		}
		modList.AddMod(m.Mod)
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
// #EVAL: ~800-line function, a straight transliteration of the reference body; left unsplit by decision (2026-08-29).
func (env *Env) buildActiveSkillModList(activeSkill *ActiveSkill) {
	skillTypes := activeSkill.SkillTypes
	skillFlags := activeSkill.SkillFlags
	activeEffect := activeSkill.ActiveEffect
	activeGrantedEffect := activeEffect.GrantedEffect

	// Set mode flags
	setFlag(skillFlags, "buffs", env.ModeBuffs)
	setFlag(skillFlags, "combat", env.ModeCombat)
	setFlag(skillFlags, "effective", env.ModeEffective)

	// Handle multipart skills
	activeGemParts := activeGrantedEffect.Custom.Parts
	if len(activeGemParts) > 1 {
		cur := activeEffect.SrcInstance.SkillPart.V
		if cur == 0 {
			cur = 1
		}
		if cur > float64(len(activeGemParts)) {
			cur = float64(len(activeGemParts))
		}
		activeEffect.SrcInstance.SkillPart = util.Some(cur)
		activeSkill.SkillPart = util.Some(cur)
		part := activeGemParts[int(cur)-1]
		for k, set := range part.Flags { // pairs order; only true/false writes, order-free
			if set {
				skillFlags[k] = true
			} else {
				delete(skillFlags, k)
			}
		}
		activeSkill.SkillPartName = part.Name
		skillFlags["multiPart"] = true
	} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		activeEffect.SrcInstance.SkillPart = util.Opt[float64]{}
		activeEffect.SrcInstance.SkillPartCalcs = util.Opt[float64]{}
	}

	w2Item := activeSkill.Actor.ItemList["Weapon 2"]
	if (skillTypes[modparser.SkillTypeRequiresShield] || skillFlags["shieldAttack"]) && activeSkill.SummonSkill == nil &&
		(w2Item == nil || w2Item.ItemType() != "Shield") {
		// Skill requires a shield to be equipped
		skillFlags["disable"] = true
		activeSkill.DisableReason = "This skill requires a Shield"
	}

	if skillFlags["shieldAttack"] {
		// Special handling for Spectral Shield Throw
		skillFlags["weapon2Attack"] = true
		zero := modparser.FlagNone
		activeSkill.Weapon2Flags = &zero
	} else {
		// Set weapon flags
		if skillFlags["forceSourceWeapon"] && activeSkill.SocketGroup != nil && activeSkill.SocketGroup.SourceItem != nil {
			// Some item-granted attacks must use the weapon that grants them.
			// The reference assigns match(...)~=nil — false is stored.
			sourceSlot := activeSkill.SocketGroup.Slot
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
		var weapon1Flags *modparser.ModFlag
		var weapon1Melee, weapon1Known bool
		if !skillFlags["forceOffHand"] {
			weapon1Flags, weapon1Melee, weapon1Known = env.getWeaponFlags(weaponOf(activeSkill.Actor.WeaponData1), weaponTypes)
		}
		if weapon1Flags == nil && activeSkill.SummonSkill != nil {
			// Minion skills seem to ignore weapon types
			f := modparser.ModFlagByName[data.WeaponTypeInfo["None"].Flag]
			weapon1Flags, weapon1Melee, weapon1Known = &f, data.WeaponTypeInfo["None"].Melee, true
		}
		if weapon1Flags != nil {
			if skillFlags["attack"] || skillFlags["dotFromAttack"] {
				activeSkill.Weapon1Flags = weapon1Flags
				skillFlags["weapon1Attack"] = true
				if weapon1Melee && skillFlags["melee"] {
					delete(skillFlags, "projectile")
				} else if !weapon1Melee && skillFlags["projectile"] {
					delete(skillFlags, "melee")
				}
			}
		} else if (skillTypes[modparser.SkillTypeDualWieldOnly] || skillFlags["forceMainHand"] || weapon1Known) && activeSkill.SummonSkill == nil {
			// Skill requires a compatible main hand weapon
			skillFlags["disable"] = true
			activeSkill.DisableReason = "Main Hand weapon is not usable with this skill"
		}
		if !skillFlags["forceMainHand"] {
			weapon2Flags, _, weapon2Known := env.getWeaponFlags(weaponOf(activeSkill.Actor.WeaponData2), weaponTypes)
			if weapon2Flags != nil {
				if skillTypes[modparser.SkillTypeDualWieldRequiresDifferentTypes] &&
					weaponType(weaponOf(activeSkill.Actor.WeaponData1)) == weaponType(weaponOf(activeSkill.Actor.WeaponData2)) &&
					!(weaponOf(activeSkill.Actor.WeaponData2) != nil && weaponOf(activeSkill.Actor.WeaponData2).CountsAsAll1H ||
						weaponOf(activeSkill.Actor.WeaponData1) != nil && weaponOf(activeSkill.Actor.WeaponData1).CountsAsAll1H) {
					skillFlags["disable"] = true
					if activeSkill.DisableReason == "" {
						activeSkill.DisableReason = "Weapon Types Need to be Different"
					}
				} else if skillFlags["attack"] || skillFlags["dotFromAttack"] {
					activeSkill.Weapon2Flags = weapon2Flags
					skillFlags["weapon2Attack"] = true
				}
			} else if (skillTypes[modparser.SkillTypeDualWieldOnly] || weapon2Known) && activeSkill.SummonSkill == nil {
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
	for stat, statValue := range BuildSkillInstanceStats(activeEffect, activeGrantedEffect) {
		mapEntry := env.statMapLookup(activeGrantedEffect, stat)
		if statValue != 0 && mapEntry != nil {
			if sf := mapEntry.SkillFlag; sf != "" {
				skillFlags[sf] = true
			}
		}
	}
	// Build skill mod flag set
	var skillModFlags modparser.ModFlag
	if skillFlags["hit"] {
		skillModFlags |= modparser.FlagHit
	}
	if skillFlags["attack"] {
		skillModFlags |= modparser.FlagAttack
	} else {
		skillModFlags |= modparser.FlagCast
		if skillFlags["spell"] {
			skillModFlags |= modparser.FlagSpell
		}
	}
	if skillFlags["melee"] {
		skillModFlags |= modparser.FlagMelee
	} else if skillFlags["projectile"] {
		skillModFlags |= modparser.FlagProjectile
		skillFlags["chaining"] = true
	}
	if skillFlags["area"] {
		skillModFlags |= modparser.FlagArea
	}

	// Build skill keyword flag set
	var skillKeywordFlags modparser.KeywordFlag
	if skillFlags["hit"] {
		skillKeywordFlags |= modparser.KeywordHit
	}
	for _, e := range []struct {
		st modparser.SkillTypeID
		kw modparser.KeywordFlag
	}{
		{modparser.SkillTypeAura, modparser.KeywordAura},
		{modparser.SkillTypeAppliesCurse, modparser.KeywordCurse},
		{modparser.SkillTypeWarcry, modparser.KeywordWarcry},
		{modparser.SkillTypeMovement, modparser.KeywordMovement},
		{modparser.SkillTypeVaal, modparser.KeywordVaal},
		{modparser.SkillTypeLightning, modparser.KeywordLightning},
		{modparser.SkillTypeCold, modparser.KeywordCold},
		{modparser.SkillTypeFire, modparser.KeywordFire},
		{modparser.SkillTypeChaos, modparser.KeywordChaos},
		{modparser.SkillTypePhysical, modparser.KeywordPhysical},
	} {
		if skillTypes[e.st] {
			skillKeywordFlags |= e.kw
		}
	}
	if skillFlags["weapon1Attack"] && activeSkill.Weapon1Flags != nil && *activeSkill.Weapon1Flags&modparser.FlagBow != 0 {
		skillKeywordFlags |= modparser.KeywordBow
	}
	if skillFlags["brand"] {
		skillKeywordFlags |= modparser.KeywordBrand
	}
	if skillFlags["arrow"] {
		skillKeywordFlags |= modparser.KeywordArrow
	}
	if skillFlags["totem"] {
		skillKeywordFlags |= modparser.KeywordTotem
	} else if skillFlags["trap"] {
		skillKeywordFlags |= modparser.KeywordTrap
	} else if skillFlags["mine"] {
		skillKeywordFlags |= modparser.KeywordMine
	} else if !skillTypes[modparser.SkillTypeTriggered] {
		skillFlags["selfCast"] = true
	}
	if skillTypes[modparser.SkillTypeAttack] {
		skillKeywordFlags |= modparser.KeywordAttack
	}
	if skillTypes[modparser.SkillTypeSpell] && !skillFlags["cast"] {
		skillKeywordFlags |= modparser.KeywordSpell
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
	distance := func(c *ConfigInput) util.Opt[float64] {
		if skillFlags["melee"] {
			return c.MeleeDistance
		}
		return c.ProjectileDistance
	}
	var effectiveRange *float64
	if v := distance(env.ConfigInput); v.Set {
		effectiveRange = &v.V
	} else if v := distance(env.Build.ConfigPlaceholder); v.Set {
		effectiveRange = &v.V
	}

	// Build config structure for modifier searches
	cfgFlags := skillModFlags
	if activeSkill.Weapon1Flags != nil {
		cfgFlags = skillModFlags | *activeSkill.Weapon1Flags
	} else if activeSkill.Weapon2Flags != nil {
		cfgFlags = skillModFlags | *activeSkill.Weapon2Flags
	}
	kf := skillKeywordFlags
	activeSkill.SkillCfg = &modstore.Cfg{
		Flags:              &cfgFlags,
		KeywordFlags:       &kf,
		SkillName:          strings.TrimPrefix(activeGrantedEffect.Name, "Vaal "),
		SkillGrantedEffect: &modstore.GrantedEffectRef{Id: activeGrantedEffect.Id, BaseFlags: activeGrantedEffect.BaseFlags},
		SkillPart:          activeSkill.SkillPart,
		// aliases the skill's own set, as the reference does: later writers
		// (the mirage skill types) are meant to be visible through the config
		SkillTypes: activeSkill.SkillTypes,
		SkillCond:  map[string]bool{},
	}
	if activeEffect.GemData != nil {
		// typed-nil guard: a nil *data.Gem in the any field would defeat
		// the eval's skillGem nil check
		activeSkill.SkillCfg.SkillGem = gemRef{activeEffect.GemData}
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

	if md := activeSkill.Actor.MinionData; md != nil && md.DamageFixup != nil {
		// Spell damage fixup for a few minions whose attack time was
		// rebalanced: less damage, more speed, net-neutral DPS.
		skillModList.AddMod(newModSF("Damage", modparser.More, modparser.Num(-100**md.DamageFixup), "Damage Fixup", modparser.FlagAttack, modparser.KeywordNone))
		skillModList.AddMod(newModSF("Speed", modparser.More, modparser.Num(100**md.DamageFixup), "Damage Fixup", modparser.FlagAttack, modparser.KeywordNone))
	}

	// Mods which apply curses are not disabled by Gruthkul's Pelt
	curseApplicationSkill := activeSkill.SocketGroup != nil && activeSkill.SocketGroup.SourceItem != nil &&
		skillFlags["curse"] && activeEffect.SrcInstance != nil &&
		activeEffect.SrcInstance.NoSupports && activeEffect.SrcInstance.Triggered
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
		activeEffect.GrantedEffectLevel = activeGrantedEffect.LevelData(activeEffect.Level)
		return
	}

	// Add support gem modifiers to skill mod list
	for _, skillEffect := range activeSkill.EffectList {
		if !skillEffect.GrantedEffect.Support {
			continue
		}
		env.mergeSkillInstanceMods(skillModList, skillEffect, nil)
		level := skillEffect.GrantedEffect.LevelData(skillEffect.Level)
		if v, ok := lvlExtra(level, "manaMultiplier"); ok {
			skillModList.AddMod(newModS("SupportManaMultiplier", modparser.More, modparser.Num(v), skillEffect.GrantedEffect.ModSource))
		}
		if v, ok := lvlExtra(level, "manaReservationPercent"); ok {
			activeSkill.SkillData.SetN("manaReservationPercent", v)
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
			skillModList.AddMod(newModS("PvpDamageMultiplier", modparser.More, modparser.Num(v), skillEffect.GrantedEffect.ModSource))
		}
		if v, ok := lvlExtra(level, "storedUses"); ok {
			activeSkill.SkillData.SetN("storedUses", v)
		}
		if v, ok := lvlExtra(level, "vaalStoredUses"); ok {
			// reference precedence: a or (0 + b)
			if !activeSkill.SkillData.Has("storedUses") {
				activeSkill.SkillData.SetN("storedUses", 0+v)
			}
		}
	}

	// Apply gem/quality modifiers from support gems
	gemLevel := activeEffect.Level
	gemQuality := activeEffect.Quality
	if activeEffect.SrcInstance != nil {
		gemLevel = activeEffect.SrcInstance.Level
		gemQuality = activeEffect.SrcInstance.Quality
	}
	skillModList.AddMod(newModS("GemLevel", modparser.Base, modparser.Num(gemLevel), "Max Level"))
	skillModList.AddMod(newModS("GemQuality", modparser.Base, modparser.Num(gemQuality), "Max Quality"))
	socketMatches := activeEffect.SrcInstance != nil && activeEffect.SrcInstance.MatchesSocket.V
	if socketMatches {
		skillModList.AddMod(newModS("GemSocketQuality", modparser.Base, modparser.Num(data.Misc.MatchingSocketQualityBonus), "Socket Quality"))
	}
	for _, supportProperty := range skillModList.Tabulate(modparser.List, activeSkill.SkillCfg, "SupportedGemProperty") {
		value, ok := supportProperty.Value.(modparser.GemPropertyRef)
		if !ok {
			panic("calc: non-GemPropertyRef value in SupportedGemProperty list (the Lua errors)")
		}
		if value.Keyword == "grants_active_skill" && activeEffect.GemData != nil && !activeEffect.GemData.Tags["support"] {
			key := value.Key
			v := value.Value.Or(0)
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
			skillModList.AddMod(newModS("GemSupport"+firstUpper(key), modparser.Base, modparser.Num(v), supportProperty.Mod.Source, firstTag(supportProperty.Mod)...))
		}
	}

	for _, gemProperty := range activeEffect.GemPropertyInfo {
		value, ok := gemProperty.Value.(modparser.GemPropertyRef)
		if !ok {
			panic("calc: non-GemPropertyRef value in GemPropertyInfo (the Lua errors)")
		}
		skillModList.AddMod(newModS("GemItem"+firstUpper(value.Key), modparser.Base, modparser.Num(value.Value.Or(0)), gemProperty.Mod.Source, firstTag(gemProperty.Mod)...))
	}

	// Add active gem modifiers
	activeEffect.ActorLevel = nil // actor.minionData and actor.level
	env.mergeSkillInstanceMods(skillModList, activeEffect, skillModList.List(activeSkill.SkillCfg, "ExtraSkillStat"))
	activeEffect.GrantedEffectLevel = activeGrantedEffect.LevelData(activeEffect.Level)

	// Add extra modifiers from granted effect level
	level := activeEffect.GrantedEffectLevel
	if v, ok := lvlExtra(level, "critChance"); ok {
		activeSkill.SkillData.SetN("CritChance", v)
	}
	if v, ok := lvlExtra(level, "damageMultiplier"); ok {
		skillModList.AddMod(newModF("Damage", modparser.More, modparser.Num(v), modparser.FlagAttack, modparser.KeywordNone))
	}
	if v, ok := lvlExtra(level, "attackTime"); ok {
		activeSkill.SkillData.SetN("attackTime", v)
	}
	if v, ok := lvlExtra(level, "attackSpeedMultiplier"); ok {
		activeSkill.SkillData.SetN("attackSpeedMultiplier", v)
	}
	if v, ok := lvlExtra(level, "cooldown"); ok {
		activeSkill.SkillData.SetN("cooldown", v)
	}
	if v, ok := lvlExtra(level, "storedUses"); ok {
		activeSkill.SkillData.SetN("storedUses", v)
	}
	if v, ok := lvlExtra(level, "vaalStoredUses"); ok {
		if !activeSkill.SkillData.Has("storedUses") {
			activeSkill.SkillData.SetN("storedUses", 0+v)
		}
	}
	if v, ok := lvlExtra(level, "soulPreventionDuration"); ok {
		activeSkill.SkillData.SetN("soulPreventionDuration", v)
	}
	if v, ok := lvlExtra(level, "PvPDamageMultiplier"); ok {
		skillModList.AddMod(newModS("PvpDamageMultiplier", modparser.More, modparser.Num(v), activeGrantedEffect.ModSource))
	}

	// Add extra modifiers from other sources
	activeSkill.ExtraSkillModList = nil
	for _, v := range skillModList.List(activeSkill.SkillCfg, "ExtraSkillMod") {
		mod := modRefOf(v)
		if mod == nil {
			continue
		}
		skillModList.AddMod(mod)
		activeSkill.ExtraSkillModList = append(activeSkill.ExtraSkillModList, mod)
	}

	// Find totem level
	if skillFlags["totem"] {
		if v, ok := lvlExtra(activeEffect.GrantedEffectLevel, "levelRequirement"); ok {
			activeSkill.SkillData.SetN("totemLevel", v)
		}
	}

	// Add active mine multiplier
	if skillFlags["mine"] {
		if activeEffect.SrcInstance != nil {
			if v := activeEffect.SrcInstance.SkillMineCount; v.Set {
				count := v.V
				activeSkill.ActiveMineCount = &count
				if count > 0 {
					skillModList.AddMod(newModS("Multiplier:ActiveMineCount", modparser.Base, modparser.Num(count), "Base"))
					existing := env.EnemyDB.Multipliers["ActiveMineCount"]
					if count > existing {
						existing = count
					}
					env.EnemyDB.Multipliers["ActiveMineCount"] = existing
				}
			}
		}
	} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		activeEffect.SrcInstance.SkillMineCountCalcs = util.Opt[float64]{}
		activeEffect.SrcInstance.SkillMineCount = util.Opt[float64]{}
	}

	// Stages
	noPotentialStage := true
	for _, pv := range activeGemParts {
		if pv.Flags["stages"] {
			noPotentialStage = false
			break
		}
	}
	strippedName := stripSpaces(activeGrantedEffect.Name)
	limit := skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "Multiplier:"+strippedName+"MaxStages")
	if limit > 0 {
		activeSkill.SkillData.SetN("stagesMax", limit)
		skillFlags["multiStage"] = true
		// srcInstance.skillStageCount or stagesMax or 1 (Lua truthiness:
		// a present 0 wins over the fallback)
		cur := limit
		if activeEffect.SrcInstance != nil {
			if v := activeEffect.SrcInstance.SkillStageCount; v.Set {
				cur = v.V
			}
		}
		minStage := 1 + skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "Multiplier:"+strippedName+"MinimumStage")
		if cur < minStage {
			cur = minStage
		}
		activeSkill.ActiveStageCount = &cur
		if cur > 0 {
			skillModList.AddMod(newModS("Multiplier:"+strippedName+"Stage", modparser.Base, modparser.Num(minFloat(limit, cur)), "Base"))
			cur = cur - 1
			activeSkill.ActiveStageCount = &cur
			skillModList.AddMod(newModS("Multiplier:"+strippedName+"StageAfterFirst", modparser.Base, modparser.Num(minFloat(limit-1, cur)), "Base"))
		}
	} else if noPotentialStage && activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		activeEffect.SrcInstance.SkillStageCountCalcs = util.Opt[float64]{}
		activeEffect.SrcInstance.SkillStageCount = util.Opt[float64]{}
	}

	// Extract skill data
	for _, v := range env.ModDB.List(activeSkill.SkillCfg, "SkillData") {
		tag, ok := v.(modparser.DataRef)
		if !ok {
			panic("calc: non-DataRef value in SkillData list (the Lua errors)")
		}
		activeSkill.SkillData.Set(tag.Key, outValueOf(tag.Value))
	}
	for _, v := range skillModList.List(activeSkill.SkillCfg, "SkillData") {
		tag, ok := v.(modparser.DataRef)
		if !ok {
			panic("calc: non-DataRef value in SkillData list (the Lua errors)")
		}
		activeSkill.SkillData.Set(tag.Key, outValueOf(tag.Value))
	}

	// Create minion
	var minionList []string
	isSpectre := false
	minionSupportLevel := map[string]float64{}
	if ml := activeGrantedEffect.Custom.MinionList; ml != nil {
		if len(ml) > 0 {
			minionList = append(minionList, ml...)
		} else {
			minionList = append([]string{}, env.Build.SpectreList...)
			isSpectre = true
		}
	}
	for _, skillEffect := range activeSkill.EffectList {
		if !skillEffect.GrantedEffect.Support {
			continue
		}
		{
			for _, minionType := range skillEffect.GrantedEffect.Custom.AddMinionList {
				found := false
				for _, existing := range minionList {
					if existing == minionType {
						found = true
						break
					}
				}
				if !found {
					lvl := skillEffect.GrantedEffect.LevelData(skillEffect.Level)
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
			if sel := activeEffect.SrcInstance.SkillMinion; sel != "" {
				for _, m := range minionList {
					if m == sel {
						minionType = sel
						break
					}
				}
			}
			activeEffect.SrcInstance.SkillMinion = minionType
		}
		minionData := data.Minions[minionType]
		if minionData == nil {
			panic("calc: unknown minion type " + minionType)
		}
		minion := &Minion{Type: minionType, MinionData: minionData, Hostile: minionData.Hostile}
		activeSkill.Minion = minion
		skillFlags["haveMinion"] = true
		if minion.Hostile {
			minion.Parent = env.Enemy
			minion.EnemyActor = env.Player
		} else {
			minion.Parent = env.Player
			minion.EnemyActor = env.Enemy
		}
		sd := activeSkill.SkillData
		lvlReq, _ := lvlExtra(activeEffect.GrantedEffectLevel, "levelRequirement")
		if sd.Flag("minionLevelIsEnemyLevel") {
			minion.Level = env.EnemyLevel
		} else if sd.Flag("minionLevelIsPlayerLevel") {
			// min(build.characterLevel [or skillData.minionLevel or lvlReq —
			// env.build is always set here], minionLevelIsPlayerLevel)
			minion.Level = math.Min(env.Build.CharacterLevel, sd.N("minionLevelIsPlayerLevel"))
		} else if v, ok := minionSupportLevel[minionType]; ok {
			minion.Level = v
		} else if sd.Flag("minionLevel") {
			minion.Level = sd.N("minionLevel")
		} else {
			minion.Level = lvlReq
		}
		minion.Level = math.Min(math.Max(minion.Level, 1), 100)
		minion.ItemList = map[string]modstore.Item{}
		if uses := activeGrantedEffect.Custom.MinionUses; uses != nil {
			minion.Uses = uses
		}
		if minion.Hostile {
			minion.LifeTable = data.MonsterLifeTable
		} else if minionData.LifeScaling == "AltLife1" {
			minion.LifeTable = data.MonsterLifeTable2
		} else if minionData.LifeScaling == "AltLife2" {
			minion.LifeTable = data.MonsterLifeTable3
		} else if isSpectre {
			minion.LifeTable = data.MonsterLifeTable
		} else {
			minion.LifeTable = data.MonsterAllyLifeTable
		}
		attackTime := minionData.AttackTime
		damageTable := data.MonsterAllyDamageTable
		if isSpectre || minion.Hostile {
			damageTable = data.MonsterDamageTable
		}
		damage := damageTable[int(minion.Level)-1] * minionData.Damage
		if !minionData.BaseDamageIgnoresAttackSpeed {
			damage = damage * attackTime
		}
		if activeGrantedEffect.Custom.MinionHasItemSet {
			cur := 0
			if activeEffect.SrcInstance != nil {
				cur = int(activeEffect.SrcInstance.SkillMinionItemSet.V)
			}
			if env.Build.ItemsTab.ItemSets[cur] == nil {
				cur = int(env.Build.ItemsTab.ItemSetOrderList[0])
				if activeEffect.SrcInstance != nil {
					activeEffect.SrcInstance.SkillMinionItemSet = util.Some(float64(cur))
				}
			}
			minion.ItemSet = env.Build.ItemsTab.ItemSets[cur]
		} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
			activeEffect.SrcInstance.SkillMinionItemSetCalcs = util.Opt[float64]{}
			activeEffect.SrcInstance.SkillMinionItemSet = util.Opt[float64]{}
		}
		if (sd.Flag("minionUseBowAndQuiver") && weaponType(weaponOf(env.Player.WeaponData1)) == "Bow") || sd.Flag("minionUseMainHandWeapon") {
			minion.WeaponData1 = weaponOf(env.Player.WeaponData1)
		} else if env.TheIronMass != nil && minionType == "RaisedSkeleton" {
			minion.WeaponData1 = weaponOf(env.Player.WeaponData1)
		} else {
			wtype := "None"
			if minionData.WeaponType1 != nil {
				wtype = *minionData.WeaponType1
			}
			minion.WeaponData1 = &item.WeaponData{
				Type:       wtype,
				AttackRate: 1 / attackTime,
				CritChance: util.Some(5.0),
				Physical: item.DamageRange{
					Min: util.RoundHalfUp(damage*(1-minionData.DamageSpread), 0),
					Max: util.RoundHalfUp(damage*(1+minionData.DamageSpread), 0),
				},
				Range: minionData.AttackRange,
			}
		}
		minion.WeaponData2 = &item.WeaponData{}
		if minion.Uses != nil {
			setSlot := func(base string) string {
				if minion.ItemSet != nil && minion.ItemSet.UseSecondWeaponSet != nil && *minion.ItemSet.UseSecondWeaponSet {
					return base + " Swap"
				}
				return base
			}
			if minion.Uses["Weapon 1"] {
				if minion.ItemSet != nil {
					item := env.Build.ItemsTab.Items[int(minion.ItemSet.Slots[setSlot("Weapon 1")])]
					if item != nil && item.WeaponData != nil {
						minion.WeaponData1 = item.WeaponData[1]
					}
				} else {
					minion.WeaponData1 = weaponOf(env.Player.WeaponData1)
				}
			}
			if minion.Uses["Weapon 2"] {
				if minion.ItemSet != nil {
					item := env.Build.ItemsTab.Items[int(minion.ItemSet.Slots[setSlot("Weapon 2")])]
					if item != nil && item.WeaponData != nil {
						minion.WeaponData2 = item.WeaponData[2]
					}
				} else {
					minion.WeaponData2 = weaponOf(env.Player.WeaponData2)
				}
			}
		}
		if !isSpectre && skillModList.Flag(activeSkill.SkillCfg, "NonSpectreMinionsUseParentMainHandAttackTime") &&
			weaponOf(env.Player.WeaponData1).AttackRate > 0 {
			// copy before replacing only the base attack rate
			cp := minion.WeaponData1.Clone()
			cp.AttackRate = weaponOf(env.Player.WeaponData1).AttackRate
			minion.WeaponData1 = cp
		}
	} else if activeEffect.SrcInstance != nil && !(activeEffect.GemData != nil && activeEffect.GemData.SecondaryGrantedEffect != nil) {
		src := activeEffect.SrcInstance
		src.SkillMinionCalcs, src.SkillMinion = "", ""
		src.SkillMinionItemSetCalcs, src.SkillMinionItemSet = util.Opt[float64]{}, util.Opt[float64]{}
		src.SkillMinionSkill, src.SkillMinionSkillCalcs = util.Opt[float64]{}, util.Opt[float64]{}
	}

	// Separate global effect modifiers
	i := 0
	for i < len(skillModList.Mods) {
		mod := skillModList.Mods[i]
		var effectType, effectName string
		var effectTag *modparser.GlobalEffectTag
		for _, tv := range mod.Tags {
			if tv == nil {
				break
			}
			if tag, ok := tv.(*modparser.GlobalEffectTag); ok {
				effectType = tag.EffectType
				effectName = tag.EffectName
				if effectName == "" {
					effectName = activeGrantedEffect.Name
				}
				effectTag = tag
				break
			}
		}
		if effectTag != nil && effectTag.ModCond != "" && !skillModList.GetCondition(effectTag.ModCond, activeSkill.SkillCfg) {
			skillModList.Mods = append(skillModList.Mods[:i], skillModList.Mods[i+1:]...)
		} else if effectType != "" {
			var buff *Buff
			for _, skillBuff := range activeSkill.BuffListTyped {
				if skillBuff.Type == effectType && skillBuff.Name == effectName {
					buff = skillBuff
					break
				}
			}
			if buff == nil {
				buff = &Buff{
					Type:       effectType,
					Name:       effectName,
					Cond:       effectTag.EffectCond,
					StackVar:   effectTag.EffectStackVar,
					StackLimit: effectTag.EffectStackLimit,
				}
				if effectTag.AllowTotemBuff {
					buff.AllowTotemBuff = util.Some(true)
				}
				if effectTag.ApplyNotPlayer {
					buff.ApplyNotPlayer = util.Some(true)
				}
				if effectTag.ApplyMinions {
					buff.ApplyMinions = util.Some(true)
				}
				if mod.Source == activeGrantedEffect.ModSource {
					// Inherit buff configuration from the active skill.
					// These are plain assignments in the reference, so a nil
					// on the right clears whatever the effect tag put there
					// -- notably allowTotemBuff.
					sd := func(key string) util.Opt[bool] {
						if !activeSkill.SkillData.Has(key) {
							return util.Opt[bool]{}
						}
						return util.Some(activeSkill.SkillData.Flag(key))
					}
					buff.ActiveSkillBuff = true
					if !buff.ApplyNotPlayer.Or(false) {
						buff.ApplyNotPlayer = sd("buffNotPlayer")
					}
					if !buff.ApplyMinions.Or(false) {
						buff.ApplyMinions = sd("buffMinions")
					}
					buff.ApplyAllies = sd("buffAllies")
					buff.AllowTotemBuff = sd("allowTotemBuff")
				}
				activeSkill.BuffListTyped = append(activeSkill.BuffListTyped, buff)
			}
			match := false
			for di, destMod := range buff.ModList {
				if modparser.CompareModParams(mod, destMod) && (destMod.Type == modparser.Base || destMod.Type == modparser.Inc) {
					cp := cloneMod(destMod)
					cp.Value = modparser.Num(valueNum(cp.Value) + valueNum(mod.Value))
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
