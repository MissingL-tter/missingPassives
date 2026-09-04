// The initEnv skills stage: CalcSetup.lua L1349-1871 — weapon data,
// support gathering, active skill creation, and main skill selection.
// buildActiveSkillModList is the next stage (the .skills checkpoint
// compares effect summaries, which are complete before it runs).
// Granted-skill and explode-source socket groups panic loudly (they need
// SkillsTab:ProcessSocketGroup); no corpus build reaches them yet.
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
	"github.com/MissingL-tter/missingPassives/skills"
)

// groupCfg is the per-group config table: a modstore Cfg (slot name plus
// running gem-colour counts) and the cached GemProperty list.
type groupCfg struct {
	Cfg             modstore.Cfg
	SlotName        string // "" = no slot
	PropertyModList []modstore.TabEntry
}

// addCount is `groupCfg.xGems = (groupCfg.xGems or 0) + n`: the key exists
// (possibly 0) once touched.
func addCount(p **float64, n float64) {
	if *p == nil {
		z := 0.0
		*p = &z
	}
	**p += n
}

// snapshotCfg is copyTable(groupCfg, true): the gem-count values are
// numbers in Lua, so the copy freezes them.
func snapshotCfg(cfg *modstore.Cfg) *modstore.Cfg {
	cp := *cfg
	clone := func(p *float64) *float64 {
		if p == nil {
			return nil
		}
		v := *p
		return &v
	}
	cp.StrengthGems = clone(cfg.StrengthGems)
	cp.DexterityGems = clone(cfg.DexterityGems)
	cp.IntelligenceGems = clone(cfg.IntelligenceGems)
	return &cp
}

func (env *Env) unarmedWeaponData() *item.WeaponData {
	return unarmedWeapon(data.UnarmedWeaponData[int(env.ClassID)])
}

// unarmedWeapon is the data.unarmedWeaponData[classId] table as weapon data.
func unarmedWeapon(uw data.UnarmedWeapon) *item.WeaponData {
	return &item.WeaponData{
		Type:       uw.Type,
		AttackRate: uw.AttackRate,
		CritChance: util.Some(uw.CritChance),
		Physical:   item.DamageRange{Min: uw.PhysicalMin, Max: uw.PhysicalMax},
	}
}

// processSocketGroup ports SkillsTab:ProcessSocketGroup for the granted-
// skill groups initEnv (re)builds. UI-only effects (colours, error
// messages) are skipped.
func (env *Env) processSocketGroup(group *SocketGroupInput) {
	for _, gem := range group.GemList {
		var prevDefaultLevel *float64
		if gem.GemData != nil {
			v := gem.GemData.NaturalMaxLevel
			prevDefaultLevel = &v
		}
		gem.GemData, gem.GrantedEffect = nil, nil
		switch {
		case gem.GemID != "":
			// Specified by gem ID (skills granted by skill gems)
			gem.GemData = data.Gems[gem.GemID]
			if gem.GemData != nil {
				gem.NameSpec = gem.GemData.Name
				gem.SkillID = gem.GemData.GrantedEffectId
			}
		case gem.SkillID != "":
			// Specified by skill ID (skills granted by items).
			// Archive parity: the reference indexes gemForSkill
			// (keyed by granted-effect TABLE) with the skillId STRING, which
			// never matches, so item-granted skills never resolve a gem.
			gem.GrantedEffect = data.Skills[gem.SkillID]
			if gem.Triggered {
				if lvl := gem.GrantedEffect.LevelData(gem.Level); lvl != nil {
					// the reference wipes the shared level's cost table;
					// kept per-env so the game-data canon stays pristine
					if env.TriggeredCostWipes == nil {
						env.TriggeredCostWipes = map[*data.SkillLevel]bool{}
					}
					env.TriggeredCostWipes[lvl] = true
				}
			}
		case strings.TrimSpace(gem.NameSpec) != "":
			// Specified by gem/skill name: the pre-1.4.20 migration path
			// (SkillsTab L1166). OnFrame migrates resolvable names before any
			// dump is taken, so the replay only re-runs this for names
			// FindSkillGem cannot resolve -- but the resolution is ported
			// whole regardless.
			if gemData := skills.FindSkillGem(gem.NameSpec); gemData != nil {
				gem.GemData = gemData
				gem.GemID = gemData.Id
				gem.SkillID = gemData.GrantedEffectId
				gem.NameSpec = gemData.Name
			} else {
				gem.GemData = nil
				gem.SkillID = ""
			}
		}
		// The reference nils gemData for `grantedEffect.unsupported`; no
		// template sets that key, so nothing to do.
		if gem.GemData != nil || gem.GrantedEffect != nil {
			gem.New = false
			grantedEffect := gem.GrantedEffect
			if grantedEffect == nil {
				grantedEffect = gem.GemData.GrantedEffect
			}
			if prevDefaultLevel != nil && gem.GemData != nil && gem.GemData.NaturalMaxLevel != *prevDefaultLevel {
				gem.Level = gem.GemData.NaturalMaxLevel
				gem.NaturalMaxLevel = gem.GemData.NaturalMaxLevel
			}
			validate := &ActiveEffect{GrantedEffect: gem.GrantedEffect, GemData: gem.GemData, Level: gem.Level}
			ValidateGemLevel(validate)
			gem.Level = validate.Level
			if gem.GemData != nil {
				reqLevel, _ := lvlExtra(grantedEffect.LevelData(validate.Level), "levelRequirement")
				gem.ReqLevel = util.Some(reqLevel)
				gem.ReqStr = util.Some(skills.GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqStr))
				gem.ReqDex = util.Some(skills.GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqDex))
				gem.ReqInt = util.Some(skills.GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqInt))
			}
		}
	}
}

// buildSkillsStage returns true when an enabled Energy Blade gem demands
// the initEnv re-entry.
func (env *Env) buildSkillsStage() bool {
	in := env.Build
	env.geFromItemMark = map[*data.GrantedEffect]bool{}
	env.slotsByName = map[string]*SlotInput{}
	for _, slot := range in.ItemsTab.Slots {
		env.slotsByName[slot.SlotName] = slot
	}

	if env.Mode == ModeMain {
		markList := map[*SocketGroupInput]bool{}
		getNormalizedSkillLevel := func(gs *GrantedSkill) float64 {
			// Levels in socketGroup.gemList[1].level are normalized
			norm := &ActiveEffect{GrantedEffect: data.Skills[gs.SkillID], Level: gs.Level}
			ValidateGemLevel(norm)
			return norm.Level
		}
		// Process extra skills granted by items or tree nodes
		for i := range env.GrantedSkills {
			gs := &env.GrantedSkills[i]
			var group *SocketGroupInput
			for _, sg := range in.SkillsTab.SocketGroups {
				if sg.Source == gs.Source && gs.Source != "" && sg.Slot == gs.SlotName {
					if len(sg.GemList) > 0 {
						g1 := sg.GemList[0]
						if g1.SkillID == gs.SkillID &&
							(g1.Level == gs.Level || g1.Level == getNormalizedSkillLevel(gs)) {
							group = sg
							markList[sg] = true
							break
						}
					}
				}
			}
			if group == nil {
				// Create a new group for this skill
				group = &SocketGroupInput{
					SocketGroup: &skills.SocketGroup{Enabled: true, Source: gs.Source, Slot: gs.SlotName},
					GemList:     []*SocketGemInput{},
				}
				in.SkillsTab.SocketGroups = append(in.SkillsTab.SocketGroups, group)
				markList[group] = true
			}

			// Update the group
			group.SourceItem = gs.SourceItem
			group.SourceNode = gs.SourceNode
			var activeGem *SocketGemInput
			if len(group.GemList) > 0 {
				activeGem = group.GemList[0]
			} else {
				activeGem = &SocketGemInput{Gem: &skills.Gem{SkillID: gs.SkillID, NameSpec: gs.NameSpec, Enabled: true}}
			}
			activeGem.FromItem = gs.SourceItem != nil
			activeGem.GemID = ""
			activeGem.GemDataID = nil
			activeGem.Level = gs.Level
			activeGem.EnableGlobal1 = true
			activeGem.NoSupports = gs.NoSupports
			group.NoSupports = gs.NoSupports
			if gs.Triggered {
				activeGem.Triggered = true
			}
			activeGem.TriggerChance = gs.TriggerChance
			group.GemList = []*SocketGemInput{activeGem}
			env.processSocketGroup(group)
		}

		if len(env.ExplodeSources) != 0 {
			// Check if a matching group already exists
			var group *SocketGroupInput
			for _, sg := range in.SkillsTab.SocketGroups {
				if sg.Source == "Explode" {
					group = sg
					break
				}
			}
			if group == nil {
				group = &SocketGroupInput{
					SocketGroup: &skills.SocketGroup{Label: "On Kill Monster Explosion", Enabled: true, Source: "Explode", NoSupports: true},
					GemList:     []*SocketGemInput{},
				}
				in.SkillsTab.SocketGroups = append(in.SkillsTab.SocketGroups, group)
			}
			// Update the group
			group.ExplodeSources = env.ExplodeSources
			// gemsBySource keys on explodeSource.modSource or .id
			gemsBySource := map[string]*SocketGemInput{}
			for _, gem := range group.GemList {
				// resolve the fixture's explode source references
				if gem.ExplodeSourceItemID != nil {
					gem.ExplodeSource = env.ItemPool[int(*gem.ExplodeSourceItemID)]
				} else if gem.ExplodeSourceNodeID != nil {
					id := int(*gem.ExplodeSourceNodeID)
					node := env.AllocNodes[id]
					if node == nil {
						node = env.Build.Spec.RadiusNodeData[id]
					}
					gem.ExplodeSource = node
				}
				if gem.ExplodeSource != nil {
					gemsBySource[gem.ExplodeSource.ExplodeKey()] = gem
				}
			}
			newList := []*SocketGemInput{}
			for _, src := range env.ExplodeSources {
				gem := gemsBySource[src.ExplodeKey()]
				if gem == nil {
					gem = &SocketGemInput{Gem: &skills.Gem{SkillID: "EnemyExplode", Enabled: true, Level: 1, Triggered: true}}
					gem.ExplodeSource = src
				}
				newList = append(newList, gem)
			}
			group.GemList = newList
			markList[group] = true
			env.processSocketGroup(group)
		}
		// Remove any socket groups that no longer have a matching item
		kept := in.SkillsTab.SocketGroups[:0]
		for _, group := range in.SkillsTab.SocketGroups {
			if !group.Granted() || markList[group] {
				kept = append(kept, group)
			}
		}
		in.SkillsTab.SocketGroups = kept
	}
	groups := in.SkillsTab.SocketGroups

	// Get the weapon data tables for the equipped weapons
	w1Item, _ := env.Player.ItemList["Weapon 1"].(*Item)
	if w1Item != nil && w1Item.In.WeaponData != nil && w1Item.In.WeaponData[1] != nil {
		env.Player.WeaponData1 = weaponRef(w1Item.In.WeaponData[1])
	} else {
		env.Player.WeaponData1 = weaponRef(env.unarmedWeaponData())
	}
	if weaponOf(env.Player.WeaponData1).CountsAsDualWielding {
		env.Player.WeaponData2 = weaponRef(w1Item.In.WeaponData[2])
	} else if env.Player.ItemList["Weapon 2"] == nil {
		// Hollow Palm Technique
		if w1Item == nil && env.Player.ItemList["Gloves"] == nil && env.ModDB.Mods["Keystone"] != nil {
			for _, keystone := range env.ModDB.Mods["Keystone"] {
				if keystone.Value == modparser.Str("Hollow Palm Technique") {
					env.Player.WeaponData2 = weaponRef(env.unarmedWeaponData())
					break
				}
			}
		}
		if weaponOf(env.Player.WeaponData2) == nil {
			env.Player.WeaponData2 = weaponRef(&item.WeaponData{})
		}
	} else {
		w2Item := env.Player.ItemList["Weapon 2"].(*Item)
		if w2Item.In.WeaponData != nil && w2Item.In.WeaponData[2] != nil {
			env.Player.WeaponData2 = weaponRef(w2Item.In.WeaponData[2])
		} else {
			env.Player.WeaponData2 = weaponRef(&item.WeaponData{})
		}
	}

	// Determine main skill group
	msg := in.MainSocketGroup
	if msg == 0 {
		msg = 1
	}
	msg = math.Min(math.Max(float64(len(groups)), 1), msg)
	in.MainSocketGroup = msg
	env.MainSocketGroup = int(msg)

	// Process supports and put them into the correct buckets
	env.CrossLinkedSupportGroups = map[string][]string{}
	for _, entry := range env.ModDB.Tabulate(modparser.List, nil, "LinkedSupport") {
		v, ok := entry.Value.(modparser.LinkedSupportRef)
		if !ok {
			panic("calc: non-LinkedSupportRef value in LinkedSupport list (the Lua errors)")
		}
		slot := entry.Mod.SourceSlot
		env.CrossLinkedSupportGroups[slot] = append(env.CrossLinkedSupportGroups[slot], v.TargetSlotName)
	}

	supportsBySlot := map[string]map[*SocketGroupInput]*[]*ActiveEffect{}
	supportsByGroup := map[*SocketGroupInput]*[]*ActiveEffect{}
	groupCfgs := map[*SocketGroupInput]*groupCfg{}
	processedSockets := map[*SocketGemInput]bool{}

	groupSlotName := func(group *SocketGroupInput) string {
		return strings.ReplaceAll(group.Slot, " Swap", "")
	}
	getGroupCfg := func(group *SocketGroupInput) *groupCfg {
		if gc := groupCfgs[group]; gc != nil {
			return gc
		}
		slotName := groupSlotName(group)
		gc := &groupCfg{SlotName: slotName}
		gc.Cfg.SlotName = slotName
		gc.PropertyModList = env.ModDB.Tabulate(modparser.List, &modstore.Cfg{SlotName: slotName}, "GemProperty")
		groupCfgs[group] = gc
		return gc
	}

	// Process support gems adding them to applicable support lists
	for index, group := range groups {
		var slot *SlotInput
		if group.Slot != "" {
			slot = env.slotsByName[group.Slot]
		}
		activeSet := 1.0
		if in.ItemsTab.UseSecondWeaponSet != nil && *in.ItemsTab.UseSecondWeaponSet {
			activeSet = 2
		}
		slotEnabled := slot == nil || slot.WeaponSet == nil || *slot.WeaponSet == activeSet
		group.SlotEnabled = slotEnabled
		if !(index+1 == env.MainSocketGroup || (group.Enabled && slotEnabled)) {
			continue
		}
		gc := getGroupCfg(group)
		var targetListList []*[]*ActiveEffect
		if gc.SlotName != "" {
			if supportsBySlot[gc.SlotName] == nil {
				supportsBySlot[gc.SlotName] = map[*SocketGroupInput]*[]*ActiveEffect{}
			}
			if supportsBySlot[gc.SlotName][group] == nil {
				supportsBySlot[gc.SlotName][group] = &[]*ActiveEffect{}
			}
			targetListList = append(targetListList, supportsBySlot[gc.SlotName][group])
		} else {
			if supportsByGroup[group] == nil {
				supportsByGroup[group] = &[]*ActiveEffect{}
			}
			targetListList = append(targetListList, supportsByGroup[group])
		}

		addExtraSupports := func(value *modparser.SkillRef, grantedEffect *data.GrantedEffect, level *float64) {
			if grantedEffect == nil && value != nil {
				grantedEffect = data.Skills[value.SkillID]
			}
			if value != nil && grantedEffect != nil {
				// Only item ExtraSupport gems are flagged as fromItem
				env.geFromItemMark[grantedEffect] = true
			}
			if value != nil && grantedEffect != nil && !grantedEffect.Support {
				// Skill gems sharing a support gem's name (e.g. Barrage)
				grantedEffect = data.Skills["Support"+value.SkillID]
				env.geFromItemMark[grantedEffect] = true
			}
			if grantedEffect != nil {
				gemId := data.GemForBaseName[strings.ToLower(grantedEffect.Name)]
				if gemId == "" {
					gemId = data.GemForBaseName[strings.ToLower(grantedEffect.Name+" Support")]
				}
				lvl := 0.0
				if level != nil {
					lvl = *level
				} else if value != nil {
					lvl = value.Level.Or(0)
				}
				for _, targetList := range targetListList {
					*targetList = append(*targetList, &ActiveEffect{
						GrantedEffect: grantedEffect,
						GemData:       data.Gems[gemId],
						Level:         lvl,
						Quality:       0,
					})
				}
			}
		}

		// if not unique item that provides skills
		if !group.Granted() {
			// Add extra supports from the item this group is socketed in
			for _, v := range env.ModDB.List(&gc.Cfg, "ExtraSupport") {
				if ref, ok := v.(modparser.SkillRef); ok {
					addExtraSupports(&ref, nil, nil)
				}
			}
		}
		// if the slot has an imbued support, add it as an ExtraSupport
		if geId := in.SkillsTab.ImbuedSupportBySlot[gc.SlotName]; geId != "" && group.ImbuedSupport != "" {
			imbued := data.Skills[geId]
			one := 1.0
			addExtraSupports(nil, imbued, &one)
			if gemData := data.Gems[data.GemForSkill[imbued]]; gemData != nil && gemData.SecondaryGrantedEffect != nil && gemData.SecondaryGrantedEffect.Support {
				addExtraSupports(nil, gemData.SecondaryGrantedEffect, &one)
			}
		}

		for gemIndex, gem := range group.GemList {
			if !gem.Enabled {
				continue
			}
			processGrantedEffect := func(grantedEffect *data.GrantedEffect) {
				if grantedEffect == nil || !grantedEffect.Support {
					return
				}
				socketBonus := 0.0
				if gem.MatchesSocket {
					socketBonus = data.Misc.MatchingSocketQualityBonus
				}
				supportEffect := &ActiveEffect{
					GrantedEffect: grantedEffect,
					Level:         gem.Level,
					Quality:       gem.Quality + socketBonus,
					SocketQuality: socketBonus,
					SrcInstance:   gem,
					GemData:       gem.GemData,
					IsSupporting:  map[*SocketGemInput]bool{},
				}
				if gem.GemData != nil {
					socketedItem, _ := env.Player.ItemList[gc.SlotName].(*Item)
					var socketedIn *SocketInput
					if socketedItem != nil && gemIndex < len(socketedItem.In.Sockets) {
						socketedIn = &socketedItem.In.Sockets[gemIndex]
					}
					supportEffect.GemCfg = snapshotCfg(&gc.Cfg)
					if socketedIn != nil {
						supportEffect.GemCfg.SocketColor = socketedIn.Color
					}
					sn := float64(gemIndex + 1)
					supportEffect.GemCfg.SocketNum = &sn
					if socketedIn != nil {
						env.applyGemMods(supportEffect, env.ModDB.Tabulate(modparser.List, supportEffect.GemCfg, "GemProperty"))
					} else {
						env.applyGemMods(supportEffect, gc.PropertyModList)
					}
					gem.ReqOverride = supportEffect.Req
					if !processedSockets[gem] {
						processedSockets[gem] = true
						modSource := ""
						if socketedItem != nil {
							modSource = socketedItem.In.Name
						}
						env.applySocketMods(gem.GemData, &gc.Cfg, gemIndex+1, modSource)
						// Keep track of the gem count for each colour
						if gem.GemData.Tags["intelligence"] {
							addCount(&gc.Cfg.IntelligenceGems, 1)
						} else {
							addCount(&gc.Cfg.IntelligenceGems, 0)
						}
						if gem.GemData.Tags["dexterity"] {
							addCount(&gc.Cfg.DexterityGems, 1)
						} else {
							addCount(&gc.Cfg.DexterityGems, 0)
						}
						if gem.GemData.Tags["strength"] {
							addCount(&gc.Cfg.StrengthGems, 1)
						} else {
							addCount(&gc.Cfg.StrengthGems, 0)
						}
					}
				}
				// Validate support gem level in case there is no active skill
				ValidateGemLevel(supportEffect)

				for _, targetList := range targetListList {
					addBestSupport(supportEffect, targetList, env.Mode)
				}
			}
			if gem.GemData != nil {
				processGrantedEffect(gem.GemData.GrantedEffect)
				processGrantedEffect(gem.GemData.SecondaryGrantedEffect)
			} else {
				processGrantedEffect(gem.GrantedEffect)
			}
		}
	}

	// Process active skills adding the applicable supports
	skillListsByGroup := map[*SocketGroupInput][]*ActiveSkill{}
	for index, group := range groups {
		if !(index+1 == env.MainSocketGroup || (group.Enabled && group.SlotEnabled)) {
			continue
		}
		gc := getGroupCfg(group)
		slotName := gc.SlotName

		// Create active skills
		for gemIndex, gem := range group.GemList {
			if !gem.Enabled || (gem.GemData == nil && gem.GrantedEffect == nil) {
				continue
			}
			var grantedEffectList []*data.GrantedEffect
			if gem.GemData != nil {
				grantedEffectList = gem.GemData.GrantedEffectList
			} else {
				grantedEffectList = []*data.GrantedEffect{gem.GrantedEffect}
			}
			for geIndex, grantedEffect := range grantedEffectList {
				enableGlobal := gem.EnableGlobal1
				if geIndex == 1 {
					enableGlobal = gem.EnableGlobal2
				}
				// (`grantedEffect.unsupported` is vestigial: no template sets it)
				if grantedEffect.Support || (env.geHasGlobalEffect(grantedEffect) && !enableGlobal) {
					continue
				}
				socketBonus := 0.0
				if gem.MatchesSocket {
					socketBonus = data.Misc.MatchingSocketQualityBonus
				}
				activeEffect := &ActiveEffect{
					GrantedEffect: grantedEffect,
					Level:         gem.Level,
					Quality:       gem.Quality + socketBonus,
					SocketQuality: socketBonus,
					SrcInstance:   gem,
					GemData:       gem.GemData,
				}
				if gem.GemData != nil {
					socketedItem, _ := env.Player.ItemList[slotName].(*Item)
					var socketedIn *SocketInput
					if socketedItem != nil && gemIndex < len(socketedItem.In.Sockets) {
						socketedIn = &socketedItem.In.Sockets[gemIndex]
					}
					activeEffect.GemCfg = snapshotCfg(&gc.Cfg)
					if socketedIn != nil {
						activeEffect.GemCfg.SocketColor = socketedIn.Color
					}
					sn := float64(gemIndex + 1)
					activeEffect.GemCfg.SocketNum = &sn
					if socketedIn != nil {
						env.applyGemMods(activeEffect, env.ModDB.Tabulate(modparser.List, activeEffect.GemCfg, "GemProperty"))
					} else {
						env.applyGemMods(activeEffect, gc.PropertyModList)
					}
					gem.ReqOverride = activeEffect.Req
					if !processedSockets[gem] {
						processedSockets[gem] = true
						modSource := ""
						if socketedItem != nil {
							modSource = socketedItem.In.Name
						}
						env.applySocketMods(gem.GemData, &gc.Cfg, gemIndex+1, modSource)
						if gem.GemData.Tags["intelligence"] {
							addCount(&gc.Cfg.IntelligenceGems, 1)
						} else {
							addCount(&gc.Cfg.IntelligenceGems, 0)
						}
						if gem.GemData.Tags["dexterity"] {
							addCount(&gc.Cfg.DexterityGems, 1)
						} else {
							addCount(&gc.Cfg.DexterityGems, 0)
						}
						if gem.GemData.Tags["strength"] {
							addCount(&gc.Cfg.StrengthGems, 1)
						} else {
							addCount(&gc.Cfg.StrengthGems, 0)
						}
					}
				}
				var appliedSupportList []*ActiveEffect
				if !group.NoSupports {
					var src *[]*ActiveEffect
					if supportsByGroup[group] != nil {
						src = supportsByGroup[group]
					} else if supportsBySlot[slotName] != nil {
						src = supportsBySlot[slotName][group]
					}
					if src != nil {
						appliedSupportList = append(appliedSupportList, *src...)
					}
					// if skill granted by unique item, add socketed supports
					// from other socketGroups in the slot (the reference
					// iterates pairs(supportLists[slotName]) in hash order;
					// group-list order here — matters only when competing
					// supports share the slot)
					if group.Granted() && slotName != "" && supportsBySlot[slotName] != nil {
						// (displayGemList tooltip bookkeeping skipped)
						for _, otherGroup := range groups {
							if sl := supportsBySlot[slotName][otherGroup]; sl != nil {
								for _, supportEffect := range *sl {
									addBestSupport(supportEffect, &appliedSupportList, env.Mode)
								}
							}
						}
					}
					// then add supports from crossLinked socketGroups
					// (the reference iterates the map in hash order; sorted
					// here — order only matters with competing supports)
					linkSlots := make([]string, 0, len(env.CrossLinkedSupportGroups))
					for k := range env.CrossLinkedSupportGroups {
						linkSlots = append(linkSlots, k)
					}
					sort.Strings(linkSlots)
					for _, crossSlot := range linkSlots {
						for _, supportedSlot := range env.CrossLinkedSupportGroups[crossSlot] {
							if supportedSlot == slotName && supportsBySlot[crossSlot] != nil {
								for gi, otherGroup := range groups {
									_ = gi
									if sl := supportsBySlot[crossSlot][otherGroup]; sl != nil {
										for _, supportEffect := range *sl {
											addBestSupport(supportEffect, &appliedSupportList, env.Mode)
										}
									}
								}
							}
						}
					}
				}
				activeSkill := env.createActiveSkill(activeEffect, appliedSupportList, env.Player, group, nil)
				if gem.GemData != nil {
					activeSkill.SlotName = slotName
				}
				skillListsByGroup[group] = append(skillListsByGroup[group], activeSkill)
				env.PlayerActiveSkills = append(env.PlayerActiveSkills, activeSkill)
				env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, activeSkill)
			}
			if gem.GemData != nil {
				req := func(v util.Opt[float64]) float64 {
					if gem.ReqOverride != nil {
						return *gem.ReqOverride
					}
					return v.V
				}
				env.RequirementsTableGems = append(env.RequirementsTableGems, GemRequirement{
					SourceGem: gem,
					Str:       req(gem.ReqStr),
					Dex:       req(gem.ReqDex),
					Int:       req(gem.ReqInt),
				})
			}
		}
	}

	// Process calculated active skill lists
	for index, group := range groups {
		socketGroupSkillList := skillListsByGroup[group]
		if index+1 == env.MainSocketGroup || (group.Enabled && group.SlotEnabled) {
			gc := getGroupCfg(group)
			for _, v := range env.ModDB.List(&gc.Cfg, "GroupProperty") {
				if ref, ok := v.(modparser.PropertyModRef); ok && ref.Mod != nil {
					env.ModDB.AddMod(modparser.SetSource(ref.Mod, gc.SlotName))
				}
			}

			if index+1 == env.MainSocketGroup && len(socketGroupSkillList) > 0 {
				// Select the main skill from this socket group
				cur := group.MainActiveSkill.V
				if cur == 0 {
					cur = 1
				}
				activeSkillIndex := int(math.Min(float64(len(socketGroupSkillList)), cur))
				if env.Mode == ModeMain {
					group.MainActiveSkill = util.Some(float64(activeSkillIndex))
				}
				env.PlayerMainSkill = socketGroupSkillList[activeSkillIndex-1]
			}
		}

		// (displayLabel / displaySkillList are UI-only and skipped)

		// Check for enabled energy blade to see if we need to regenerate
		if !env.ModDB.Conditions.Get("AffectedByEnergyBlade") && group.Enabled && group.SlotEnabled {
			for _, gem := range group.GemList {
				ge := gem.GrantedEffect
				if gem.GemData != nil {
					ge = gem.GemData.GrantedEffect
				}
				if ge != nil && !ge.Support && gem.Enabled && ge.Name == "Energy Blade" {
					return true
				}
			}
		}
	}

	if env.PlayerMainSkill == nil {
		// Add a default main skill if none are specified
		defaultEffect := &ActiveEffect{
			GrantedEffect: data.Skills["Melee"],
			Level:         1,
			Quality:       0,
		}
		env.PlayerMainSkill = env.createActiveSkill(defaultEffect, nil, env.Player, nil, nil)
		env.PlayerActiveSkills = append(env.PlayerActiveSkills, env.PlayerMainSkill)
		env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, env.PlayerMainSkill)
	}

	// Build skill modifier lists
	for _, activeSkill := range env.PlayerActiveSkills {
		env.buildActiveSkillModList(activeSkill)
	}
	return false
}
