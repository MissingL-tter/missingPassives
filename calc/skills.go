// The initEnv skills stage: CalcSetup.lua L1349-1871 — weapon data,
// support gathering, active skill creation, and main skill selection.
// buildActiveSkillModList is the next stage (the .skills checkpoint
// compares effect summaries, which are complete before it runs).
// Granted-skill and explode-source socket groups panic loudly (they need
// SkillsTab:ProcessSocketGroup); no corpus build reaches them yet.
package calc

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
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

func (env *Env) unarmedWeaponData() map[string]any {
	uw := data.UnarmedWeaponData[int(env.ClassID)]
	return map[string]any{
		"type":        uw.Type,
		"AttackRate":  uw.AttackRate,
		"CritChance":  uw.CritChance,
		"PhysicalMin": uw.PhysicalMin,
		"PhysicalMax": uw.PhysicalMax,
	}
}

// processSocketGroup ports SkillsTab:ProcessSocketGroup for the granted-
// skill groups initEnv (re)builds. UI-only effects (colours, error
// messages) are skipped.
func (env *Env) processSocketGroup(group *SocketGroupInput) {
	for _, gem := range group.GemList {
		if _, ok := gem.KV["nameSpec"]; !ok {
			gem.KV["nameSpec"] = ""
		}
		var prevDefaultLevel *float64
		if gem.GemData != nil {
			v := gem.GemData.NaturalMaxLevel
			prevDefaultLevel = &v
		}
		gem.GemData, gem.GrantedEffect = nil, nil
		if id := str(gem.KV["gemId"]); id != "" {
			// Specified by gem ID (skills granted by skill gems)
			gem.GemData = data.Gems[id]
			if gem.GemData != nil {
				gem.KV["nameSpec"] = gem.GemData.Name
				gem.KV["skillId"] = gem.GemData.GrantedEffectId
			}
		} else if skillId := str(gem.KV["skillId"]); skillId != "" {
			// Specified by skill ID (skills granted by items).
			// #EVAL: archive parity — the reference indexes gemForSkill
			// (keyed by granted-effect TABLE) with the skillId STRING, which
			// never matches, so item-granted skills never resolve a gem.
			gem.GrantedEffect = data.Skills[skillId]
			if truthy(gem.KV["triggered"]) {
				if lvl := gem.GrantedEffect.Levels[anyNum(gem.KV["level"])]; lvl != nil {
					// the reference wipes the shared level's cost table;
					// kept per-env so the game-data canon stays pristine
					if env.TriggeredCostWipes == nil {
						env.TriggeredCostWipes = map[*data.SkillLevel]bool{}
					}
					env.TriggeredCostWipes[lvl] = true
				}
			}
		} else if strings.TrimSpace(str(gem.KV["nameSpec"])) != "" {
			// Specified by gem/skill name: the pre-1.4.20 migration path
			// (SkillsTab L1166). OnFrame migrates resolvable names before any
			// dump is taken, so the replay only re-runs this for names
			// FindSkillGem cannot resolve -- but the resolution is ported
			// whole regardless.
			if gemData := findSkillGem(str(gem.KV["nameSpec"])); gemData != nil {
				gem.GemData = gemData
				gem.KV["gemId"] = gemData.Id
				gem.KV["skillId"] = gemData.GrantedEffectId
				gem.KV["nameSpec"] = gemData.Name
			} else {
				gem.GemData = nil
				delete(gem.KV, "skillId")
			}
		}
		if gem.GemData != nil && geUnsupported(gem.GemData.GrantedEffect) {
			gem.GemData = nil
		}
		if gem.GemData != nil || gem.GrantedEffect != nil {
			delete(gem.KV, "new")
			grantedEffect := gem.GrantedEffect
			if grantedEffect == nil {
				grantedEffect = gem.GemData.GrantedEffect
			}
			if prevDefaultLevel != nil && gem.GemData != nil && gem.GemData.NaturalMaxLevel != *prevDefaultLevel {
				gem.KV["level"] = gem.GemData.NaturalMaxLevel
				gem.KV["naturalMaxLevel"] = gem.GemData.NaturalMaxLevel
			}
			validate := &ActiveEffect{GrantedEffect: gem.GrantedEffect, GemData: gem.GemData, Level: anyNum(gem.KV["level"])}
			ValidateGemLevel(validate)
			gem.KV["level"] = validate.Level
			if gem.GemData != nil {
				reqLevel, _ := lvlExtra(grantedEffect.Levels[validate.Level], "levelRequirement")
				gem.KV["reqLevel"] = reqLevel
				gem.KV["reqStr"] = GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqStr)
				gem.KV["reqDex"] = GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqDex)
				gem.KV["reqInt"] = GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqInt)
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

	if env.Mode == "MAIN" {
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
				if str(sg.KV["source"]) == gs.Source && gs.Source != "" && str(sg.KV["slot"]) == gs.SlotName {
					if len(sg.GemList) > 0 {
						g1 := sg.GemList[0]
						if str(g1.KV["skillId"]) == gs.SkillID &&
							(anyNum(g1.KV["level"]) == gs.Level || anyNum(g1.KV["level"]) == getNormalizedSkillLevel(gs)) {
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
					KV:      map[string]any{"label": "", "enabled": true, "source": gs.Source},
					GemList: []*SocketGemInput{},
				}
				if gs.SlotName != "" {
					group.KV["slot"] = gs.SlotName
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
				activeGem = &SocketGemInput{KV: map[string]any{
					"skillId": gs.SkillID,
					"quality": 0.0,
					"enabled": true,
				}}
				if gs.NameSpec != "" {
					activeGem.KV["nameSpec"] = gs.NameSpec
				}
			}
			activeGem.KV["fromItem"] = gs.SourceItem != nil
			delete(activeGem.KV, "gemId")
			activeGem.GemDataID = nil
			activeGem.KV["level"] = gs.Level
			activeGem.KV["enableGlobal1"] = true
			setKV(activeGem.KV, "noSupports", gs.NoSupports)
			group.KV["noSupports"] = gs.NoSupports
			setKV(activeGem.KV, "triggered", gs.Raw["triggered"])
			setKV(activeGem.KV, "triggerChance", gs.Raw["triggerChance"])
			group.GemList = []*SocketGemInput{activeGem}
			env.processSocketGroup(group)
		}

		if len(env.ExplodeSources) != 0 {
			// Check if a matching group already exists
			var group *SocketGroupInput
			for _, sg := range in.SkillsTab.SocketGroups {
				if str(sg.KV["source"]) == "Explode" {
					group = sg
					break
				}
			}
			if group == nil {
				group = &SocketGroupInput{
					KV: map[string]any{
						"label": "On Kill Monster Explosion", "enabled": true,
						"source": "Explode", "noSupports": true,
					},
					GemList: []*SocketGemInput{},
				}
				in.SkillsTab.SocketGroups = append(in.SkillsTab.SocketGroups, group)
			}
			// Update the group
			group.ExplodeSources = env.ExplodeSources
			// gemsBySource keys on explodeSource.modSource or .id
			explodeKey := func(src any) string {
				switch t := src.(type) {
				case *Item:
					if t.In.ModSource == nil {
						panic("calc: explode-source item without modSource")
					}
					return *t.In.ModSource
				case *NodeInput:
					return fmt.Sprintf("#%d", int64(t.ID))
				}
				panic("calc: unknown explode source type")
			}
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
					gemsBySource[explodeKey(gem.ExplodeSource)] = gem
				}
			}
			newList := []*SocketGemInput{}
			for _, src := range env.ExplodeSources {
				gem := gemsBySource[explodeKey(src)]
				if gem == nil {
					gem = &SocketGemInput{KV: map[string]any{
						"skillId": "EnemyExplode", "quality": 0.0, "enabled": true,
						"level": 1.0, "triggered": true,
					}}
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
			if group.KV["source"] == nil || markList[group] {
				kept = append(kept, group)
			}
		}
		in.SkillsTab.SocketGroups = kept
	}
	groups := in.SkillsTab.SocketGroups

	// Get the weapon data tables for the equipped weapons
	w1Item, _ := env.Player.ItemList["Weapon 1"].(*Item)
	if w1Item != nil && w1Item.In.WeaponData != nil && w1Item.In.WeaponData[1] != nil {
		env.Player.WeaponData1 = w1Item.In.WeaponData[1]
	} else {
		env.Player.WeaponData1 = env.unarmedWeaponData()
	}
	if truthy(env.Player.WeaponData1["countsAsDualWielding"]) {
		env.Player.WeaponData2 = w1Item.In.WeaponData[2]
	} else if env.Player.ItemList["Weapon 2"] == nil {
		// Hollow Palm Technique
		if w1Item == nil && env.Player.ItemList["Gloves"] == nil && env.ModDB.Mods["Keystone"] != nil {
			for _, keystone := range env.ModDB.Mods["Keystone"] {
				if keystone.Value == "Hollow Palm Technique" {
					env.Player.WeaponData2 = env.unarmedWeaponData()
					break
				}
			}
		}
		if env.Player.WeaponData2 == nil {
			env.Player.WeaponData2 = map[string]any{}
		}
	} else {
		w2Item := env.Player.ItemList["Weapon 2"].(*Item)
		if w2Item.In.WeaponData != nil && w2Item.In.WeaponData[2] != nil {
			env.Player.WeaponData2 = w2Item.In.WeaponData[2]
		} else {
			env.Player.WeaponData2 = map[string]any{}
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
	for _, entry := range env.ModDB.Tabulate("LIST", nil, "LinkedSupport") {
		v, _ := entry.Value.(modparser.Tag)
		slot := entry.Mod.SourceSlot
		env.CrossLinkedSupportGroups[slot] = append(env.CrossLinkedSupportGroups[slot], str(v["targetSlotName"]))
	}

	supportsBySlot := map[string]map[*SocketGroupInput]*[]*ActiveEffect{}
	supportsByGroup := map[*SocketGroupInput]*[]*ActiveEffect{}
	groupCfgs := map[*SocketGroupInput]*groupCfg{}
	processedSockets := map[*SocketGemInput]bool{}

	groupSlotName := func(group *SocketGroupInput) string {
		if s, ok := group.KV["slot"].(string); ok {
			return strings.ReplaceAll(s, " Swap", "")
		}
		return ""
	}
	getGroupCfg := func(group *SocketGroupInput) *groupCfg {
		if gc := groupCfgs[group]; gc != nil {
			return gc
		}
		slotName := groupSlotName(group)
		gc := &groupCfg{SlotName: slotName}
		gc.Cfg.SlotName = slotName
		gc.PropertyModList = env.ModDB.Tabulate("LIST", &modstore.Cfg{SlotName: slotName}, "GemProperty")
		groupCfgs[group] = gc
		return gc
	}

	// Process support gems adding them to applicable support lists
	for index, group := range groups {
		var slot *SlotInput
		if s, ok := group.KV["slot"].(string); ok {
			slot = env.slotsByName[s]
		}
		activeSet := 1.0
		if in.ItemsTab.UseSecondWeaponSet != nil && *in.ItemsTab.UseSecondWeaponSet {
			activeSet = 2
		}
		slotEnabled := slot == nil || slot.WeaponSet == nil || *slot.WeaponSet == activeSet
		group.KV["slotEnabled"] = slotEnabled
		if !(index+1 == env.MainSocketGroup || (truthy(group.KV["enabled"]) && slotEnabled)) {
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

		addExtraSupports := func(value modparser.Tag, grantedEffect *data.GrantedEffect, level *float64) {
			if grantedEffect == nil && value != nil {
				grantedEffect = data.Skills[str(value["skillId"])]
			}
			if value != nil && grantedEffect != nil {
				// Only item ExtraSupport gems are flagged as fromItem
				env.geFromItemMark[grantedEffect] = true
			}
			if value != nil && grantedEffect != nil && !grantedEffect.Support {
				// Skill gems sharing a support gem's name (e.g. Barrage)
				grantedEffect = data.Skills["Support"+str(value["skillId"])]
				env.geFromItemMark[grantedEffect] = true
			}
			if grantedEffect != nil {
				gemId := data.GemForBaseName[luaLower(grantedEffect.Name)]
				if gemId == "" {
					gemId = data.GemForBaseName[luaLower(grantedEffect.Name+" Support")]
				}
				lvl := 0.0
				if level != nil {
					lvl = *level
				} else if value != nil {
					lvl = anyNum(value["level"])
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
		if group.KV["source"] == nil {
			// Add extra supports from the item this group is socketed in
			for _, v := range env.ModDB.List(&gc.Cfg, "ExtraSupport") {
				tag, _ := v.(modparser.Tag)
				addExtraSupports(tag, nil, nil)
			}
		}
		// if the slot has an imbued support, add it as an ExtraSupport
		if geId := in.SkillsTab.ImbuedSupportBySlot[gc.SlotName]; geId != "" && truthy(group.KV["imbuedSupport"]) {
			imbued := data.Skills[geId]
			one := 1.0
			addExtraSupports(nil, imbued, &one)
			if gemData := data.Gems[data.GemForSkill[imbued]]; gemData != nil && gemData.SecondaryGrantedEffect != nil && gemData.SecondaryGrantedEffect.Support {
				addExtraSupports(nil, gemData.SecondaryGrantedEffect, &one)
			}
		}

		for gemIndex, gem := range group.GemList {
			if !truthy(gem.KV["enabled"]) {
				continue
			}
			processGrantedEffect := func(grantedEffect *data.GrantedEffect) {
				if grantedEffect == nil || !grantedEffect.Support {
					return
				}
				socketBonus := 0.0
				if truthy(gem.KV["matchesSocket"]) {
					socketBonus = data.Misc.MatchingSocketQualityBonus
				}
				supportEffect := &ActiveEffect{
					GrantedEffect: grantedEffect,
					Level:         anyNum(gem.KV["level"]),
					Quality:       anyNum(gem.KV["quality"]) + socketBonus,
					SocketQuality: socketBonus,
					SrcInstance:   gem,
					GemData:       gem.GemData,
					IsSupporting:  map[*SocketGemInput]bool{},
				}
				if gem.GemData != nil {
					socketedItem, _ := env.Player.ItemList[gc.SlotName].(*Item)
					var socketedIn map[string]any
					if socketedItem != nil && gemIndex < len(socketedItem.In.Sockets) {
						socketedIn = socketedItem.In.Sockets[gemIndex]
					}
					supportEffect.GemCfg = snapshotCfg(&gc.Cfg)
					if socketedIn != nil {
						supportEffect.GemCfg.SocketColor = str(socketedIn["color"])
					}
					sn := float64(gemIndex + 1)
					supportEffect.GemCfg.SocketNum = &sn
					if socketedIn != nil {
						env.applyGemMods(supportEffect, env.ModDB.Tabulate("LIST", supportEffect.GemCfg, "GemProperty"))
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
		if !(index+1 == env.MainSocketGroup || (truthy(group.KV["enabled"]) && truthy(group.KV["slotEnabled"]))) {
			continue
		}
		gc := getGroupCfg(group)
		slotName := gc.SlotName

		// Create active skills
		for gemIndex, gem := range group.GemList {
			if !truthy(gem.KV["enabled"]) || (gem.GemData == nil && gem.GrantedEffect == nil) {
				continue
			}
			var grantedEffectList []*data.GrantedEffect
			if gem.GemData != nil {
				grantedEffectList = gem.GemData.GrantedEffectList
			} else {
				grantedEffectList = []*data.GrantedEffect{gem.GrantedEffect}
			}
			for geIndex, grantedEffect := range grantedEffectList {
				enableGlobal := truthy(gem.KV["enableGlobal1"])
				if geIndex == 1 {
					enableGlobal = truthy(gem.KV["enableGlobal2"])
				}
				if grantedEffect.Support || geUnsupported(grantedEffect) || (env.geHasGlobalEffect(grantedEffect) && !enableGlobal) {
					continue
				}
				socketBonus := 0.0
				if truthy(gem.KV["matchesSocket"]) {
					socketBonus = data.Misc.MatchingSocketQualityBonus
				}
				activeEffect := &ActiveEffect{
					GrantedEffect: grantedEffect,
					Level:         anyNum(gem.KV["level"]),
					Quality:       anyNum(gem.KV["quality"]) + socketBonus,
					SocketQuality: socketBonus,
					SrcInstance:   gem,
					GemData:       gem.GemData,
				}
				if gem.GemData != nil {
					socketedItem, _ := env.Player.ItemList[slotName].(*Item)
					var socketedIn map[string]any
					if socketedItem != nil && gemIndex < len(socketedItem.In.Sockets) {
						socketedIn = socketedItem.In.Sockets[gemIndex]
					}
					activeEffect.GemCfg = snapshotCfg(&gc.Cfg)
					if socketedIn != nil {
						activeEffect.GemCfg.SocketColor = str(socketedIn["color"])
					}
					sn := float64(gemIndex + 1)
					activeEffect.GemCfg.SocketNum = &sn
					if socketedIn != nil {
						env.applyGemMods(activeEffect, env.ModDB.Tabulate("LIST", activeEffect.GemCfg, "GemProperty"))
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
				if !truthy(group.KV["noSupports"]) {
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
					if group.KV["source"] != nil && slotName != "" && supportsBySlot[slotName] != nil {
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
			}
			if gem.GemData != nil {
				req := func(key string) float64 {
					if gem.ReqOverride != nil {
						return *gem.ReqOverride
					}
					return anyNum(gem.KV[key])
				}
				env.RequirementsTableGems = append(env.RequirementsTableGems, GemRequirement{
					SourceGem: gem,
					Str:       req("reqStr"),
					Dex:       req("reqDex"),
					Int:       req("reqInt"),
				})
			}
		}
	}

	// Process calculated active skill lists
	for index, group := range groups {
		socketGroupSkillList := skillListsByGroup[group]
		if index+1 == env.MainSocketGroup || (truthy(group.KV["enabled"]) && truthy(group.KV["slotEnabled"])) {
			gc := getGroupCfg(group)
			for _, v := range env.ModDB.List(&gc.Cfg, "GroupProperty") {
				tag, _ := v.(modparser.Tag)
				if mod, ok := tag["value"].(*modparser.Mod); ok {
					env.ModDB.AddMod(modparser.SetSource(mod, gc.SlotName))
				}
			}

			if index+1 == env.MainSocketGroup && len(socketGroupSkillList) > 0 {
				// Select the main skill from this socket group
				cur := anyNum(group.KV["mainActiveSkill"])
				if cur == 0 {
					cur = 1
				}
				activeSkillIndex := int(math.Min(float64(len(socketGroupSkillList)), cur))
				if env.Mode == "MAIN" {
					group.KV["mainActiveSkill"] = float64(activeSkillIndex)
				}
				env.PlayerMainSkill = socketGroupSkillList[activeSkillIndex-1]
			}
		}

		// (displayLabel / displaySkillList are UI-only and skipped)

		// Check for enabled energy blade to see if we need to regenerate
		if !truthy(env.ModDB.Conditions["AffectedByEnergyBlade"]) && truthy(group.KV["enabled"]) && truthy(group.KV["slotEnabled"]) {
			for _, gem := range group.GemList {
				ge := gem.GrantedEffect
				if gem.GemData != nil {
					ge = gem.GemData.GrantedEffect
				}
				if ge != nil && !ge.Support && truthy(gem.KV["enabled"]) && ge.Name == "Energy Blade" {
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
	}

	// Build skill modifier lists
	for _, activeSkill := range env.PlayerActiveSkills {
		env.buildActiveSkillModList(activeSkill)
	}
	return false
}

// findSkillGem ports SkillsTab:FindSkillGem (L1076): match a gem by name
// through increasingly broad abbreviation patterns. Returns nil when the
// name is unrecognised or ambiguous (the reference reports an error message
// either way; nothing compared carries it). The reference iterates
// pairs(data.gems) -- hash order -- which only affects which two names an
// ambiguity error cites; the match result is order-independent, and this
// port scans in sorted id order.
// FindSkillGem exposes the lookup for the skills-tab load (its
// pre-1.4.20 nameSpec migration path resolves gems the same way).
func FindSkillGem(nameSpec string) *data.Gem { return findSkillGem(nameSpec) }

func findSkillGem(nameSpec string) *data.Gem {
	type matcher func(name string) bool
	isAlpha := func(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
	lowerRun := func(s string, i int) int { // len of [a-z]+ run at i
		n := 0
		for i+n < len(s) && s[i+n] >= 'a' && s[i+n] <= 'z' {
			n++
		}
		return n
	}
	noSpaces := strings.ReplaceAll(nameSpec, " ", "")

	// 1. Exact match, case-insensitive.
	exact := func(name string) bool { return strings.EqualFold(name, nameSpec) }

	// 2. Simple abbreviation ("CtF" -> "Cold to Fire"): each spec letter
	// starts a word and is followed by one or more lowercase letters;
	// non-letters in the spec match literally. Subject is " "+name, anchored
	// both ends.
	simpleAbbrev := func(name string) bool {
		s := " " + name
		i := 0
		for k := 0; k < len(nameSpec); k++ {
			c := nameSpec[k]
			if isAlpha(c) {
				if i >= len(s) || s[i] != ' ' {
					return false
				}
				i++
				if i >= len(s) || s[i] != c {
					return false
				}
				i++
				run := lowerRun(s, i)
				if run == 0 {
					return false
				}
				i += run
			} else {
				if i >= len(s) || s[i] != c {
					return false
				}
				i++
			}
		}
		return i == len(s)
	}

	// 3. Abbreviated words ("CldFr" -> "Cold to Fire"): lowercase spec
	// letters may be preceded by lowercase runs; anchored, must end in a
	// lowercase run. Greedy with backtracking over the optional runs is
	// needed for faithfulness; implement recursively.
	var wordAbbrevAt func(s string, i, k int) bool
	wordAbbrevAt = func(s string, i, k int) bool {
		if k == len(noSpaces) {
			run := lowerRun(s, i)
			return run > 0 && i+run == len(s)
		}
		c := noSpaces[k]
		if c >= 'a' && c <= 'z' {
			// "%l*c": try every split of a lowercase run before c.
			for skip := 0; ; skip++ {
				j := i + skip
				if j < len(s) && s[j] == c && wordAbbrevAt(s, j+1, k+1) {
					return true
				}
				if j >= len(s) || !(s[j] >= 'a' && s[j] <= 'z') {
					return false
				}
			}
		}
		if i < len(s) && s[i] == c {
			return wordAbbrevAt(s, i+1, k+1)
		}
		return false
	}
	wordAbbrev := func(name string) bool { return wordAbbrevAt(" "+name, 1, 0) }

	// 4. Global abbreviation ("CtoF" -> "Cold to Fire"): spec letters appear
	// in order anywhere (case-sensitive); unanchored tail.
	globalAbbrev := func(name string) bool {
		s := " " + name
		i := 0
		for k := 0; k < len(noSpaces); k++ {
			c := noSpaces[k]
			j := strings.IndexByte(s[i:], c)
			if j < 0 {
				return false
			}
			i += j + 1
		}
		return true
	}

	// 5. The same, case-insensitive.
	globalAbbrevFold := func(name string) bool {
		s := strings.ToLower(" " + name)
		spec := strings.ToLower(noSpaces)
		i := 0
		for k := 0; k < len(spec); k++ {
			j := strings.IndexByte(s[i:], spec[k])
			if j < 0 {
				return false
			}
			i += j + 1
		}
		return true
	}

	gemIds := make([]string, 0, len(data.Gems))
	for id := range data.Gems {
		gemIds = append(gemIds, id)
	}
	sort.Strings(gemIds)
	for _, match := range []matcher{exact, simpleAbbrev, wordAbbrev, globalAbbrev, globalAbbrevFold} {
		var found *data.Gem
		for _, id := range gemIds {
			g := data.Gems[id]
			if match(g.Name) {
				if found != nil {
					return nil // ambiguous
				}
				found = g
			}
		}
		if found != nil {
			return found
		}
	}
	return nil
}
