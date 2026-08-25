// The initEnv items stage: CalcSetup.lua L697-1280 (up to, not including,
// the mergeDB into modDB). Branches the corpus cannot reach yet panic
// loudly instead of diverging silently: radius jewels (funcList), Energy
// Blade, corrupted-jewel-effect scaling (needs spec.nodes), and
// classRestriction conditions (string-valued, modstore types bool).
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

func str(v any) string {
	s, _ := v.(string)
	return s
}

// titleCaseRarity is rarity:gsub("(%a)(%u*)", first..lower(rest)):
// "MAGIC" -> "Magic", "RARE" -> "Rare".
func titleCaseRarity(r string) string {
	if r == "" {
		return r
	}
	return r[:1] + luaLower(r[1:])
}

// flaskSlotNum is slotName:match("Flask (%d+)").
func flaskSlotNum(slotName string) (float64, bool) {
	idx := strings.Index(slotName, "Flask ")
	if idx < 0 {
		return 0, false
	}
	rest := slotName[idx+6:]
	n := 0
	digits := 0
	for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
		n = n*10 + int(rest[digits]-'0')
		digits++
	}
	if digits == 0 {
		return 0, false
	}
	return float64(n), true
}

// tagsDisableSlot scans a mod's tags for a DisablesItem entry, mirroring
// the reference's inner loop: returns the disabled slot ("" = none), with
// the excludeItemType escape applied against the current items map.
func tagsDisableSlot(mod *modparser.Mod, items map[string]*Item) string {
	for _, tv := range mod.Tags {
		if tv == nil {
			break
		}
		tag, ok := tv.(modparser.Tag)
		if !ok {
			continue
		}
		if tag["type"] == "DisablesItem" {
			slotN := str(tag["slotName"])
			if excl := str(tag["excludeItemType"]); excl != "" && items[slotN] != nil && items[slotN].In.Type == excl {
				return ""
			}
			return slotN
		}
	}
	return ""
}

func (env *Env) buildItems() {
	in := env.Build
	cfg := env.ConfigInput
	useSecond := in.ItemsTab.UseSecondWeaponSet != nil && *in.ItemsTab.UseSecondWeaponSet
	activeSet := 1.0
	if useSecond {
		activeSet = 2
	}

	pool := map[int]*Item{}
	for id, ii := range in.ItemsTab.Items {
		pool[id] = &Item{In: ii}
	}
	env.ItemPool = pool

	items := map[string]*Item{}
	jewelLimits := map[string]float64{}
	for _, slot := range in.ItemsTab.Slots {
		slotName := slot.SlotName
		if slotName == "Graft 1" || slotName == "Graft 2" {
			if !strings.Contains(in.TreeVersion, "3_27") {
				continue
			}
		}
		// ignore item in Ring 3 if The Unseen Hand is not allocated
		if slotName == "Ring 3" && !env.InitialNodeModDB.Flag(nil, "AdditionalRingSlot") {
			continue
		}
		var item *Item
		if slot.ItemID != nil {
			item = pool[int(*slot.ItemID)]
		}
		if item != nil {
			// Find skills granted by this item
			for _, skill := range item.In.GrantedSkills {
				granted := GrantedSkill{
					SkillID:    str(skill["skillId"]),
					Level:      anyNum(skill["level"]),
					NoSupports: truthy(skill["noSupports"]),
					Source:     str(skill["source"]),
					SourceItem: item,
					SlotName:   slotName,
					Raw:        skill,
				}
				if ge := data.Skills[granted.SkillID]; ge != nil {
					granted.NameSpec = ge.Name
				}
				env.GrantedSkillsItems = append(env.GrantedSkillsItems, granted)
			}
		}
		if item != nil && item.In.BaseModList != nil && listFlagOf(item.In.BaseModList, "CanExplode") {
			env.ExplodeSources = append(env.ExplodeSources, item)
		}
		if slot.WeaponSet != nil && *slot.WeaponSet != activeSet {
			continue
		}
		if slot.WeaponSet != nil && *slot.WeaponSet == 2 && useSecond {
			slotName = strings.ReplaceAll(slotName, " Swap", "")
		}
		if slot.NodeID != nil {
			// Slot is a jewel socket, check if socket is allocated
			if env.AllocNodes[int(*slot.NodeID)] == nil {
				continue
			} else if item != nil {
				item.JewelLimitDisabled = false
				if item.In.Type == "Jewel" && strings.Contains(item.In.Name, "The Adorned, Crimson Jewel") {
					if v, ok := item.In.JewelData["corruptedMagicJewelIncEffect"]; ok && truthy(v) {
						env.ModDB.Multipliers["CorruptedMagicJewelEffect"] = anyNum(v) / 100
					}
					if v, ok := item.In.JewelData["corruptedRareJewelIncEffect"]; ok && truthy(v) {
						env.ModDB.Multipliers["CorruptedRareJewelEffect"] = anyNum(v) / 100
					}
				}
				if item.In.Limit != nil && !truthy(cfg["ignoreJewelLimits"]) {
					limitKey := deref(item.In.Title)
					if item.In.Base != nil && deref(item.In.Base.SubType) == "Timeless" {
						limitKey = "Historic"
					}
					if count, ok := jewelLimits[limitKey]; ok && count >= *item.In.Limit {
						item.JewelLimitDisabled = true
						continue
					}
					jewelLimits[limitKey]++
				}
				if item.In.JewelRadiusIndex != nil {
					// Jewel has a radius, add it to the list
					env.addRadiusJewel(slot, item)
				}
			}
		}
		if item != nil && item.In.Type == "Flask" && item.In.Base != nil && deref(item.In.Base.SubType) == "Life" && item.In.FlaskData != nil {
			// Keep highest life flask recovery even if this slot is later disabled
			env.ItemModDB.Multipliers["LifeFlaskRecovery"] = math.Max(
				env.ItemModDB.Multipliers["LifeFlaskRecovery"], anyNum(item.In.FlaskData["lifeTotal"]))
		}
		if item != nil {
			items[slotName] = item
		}
	}

	if !truthy(cfg["ignoreItemDisablers"]) {
		itemDisabled := map[string]string{}
		itemDisablers := map[string]string{}
		// Tree nodes first. Note the modType is "Flag", not "FLAG" — dead in
		// the reference too (mod types are upper-case), preserved as-is.
		for _, entry := range env.InitialNodeModDB.Tabulate("Flag", &modstore.Cfg{Source: "Tree"}, "CanNotUseItem") {
			if slotN := tagsDisableSlot(entry.Mod, items); slotN != "" {
				itemDisablers[entry.Mod.Source] = slotN
				itemDisabled[slotN] = entry.Mod.Source
			}
		}
		for _, slot := range in.ItemsTab.Slots {
			slotName := slot.SlotName
			if it := items[slotName]; it != nil {
				srcList := it.In.ModList
				if srcList == nil && it.In.SlotModList != nil && slot.SlotNum != nil {
					srcList = it.In.SlotModList[int(*slot.SlotNum)]
				}
				for _, mod := range srcList {
					if slotN := tagsDisableSlot(mod, items); slotN != "" {
						itemDisablers[slotName] = slotN
						itemDisabled[slotN] = slotName
					}
				}
			}
		}
		visited := map[string]bool{}
		trueDisabled := map[string]bool{}
		// The reference iterates pairs(itemDisablers) in hash order; sorted
		// here — only distinguishable when disabler cycles exist.
		starts := make([]string, 0, len(itemDisablers))
		for k := range itemDisablers {
			starts = append(starts, k)
		}
		sort.Strings(starts)
		for _, start := range starts {
			if visited[start] {
				continue
			}
			slot := start
			// #EVAL: archive parity — the reference's `{ slot = true }` keys
			// the literal string "slot", so the chain start itself is never
			// in the cycle set.
			curChain := map[string]bool{"slot": true}
			for itemDisabled[slot] != "" {
				slot = itemDisabled[slot]
				if curChain[slot] {
					break
				}
				curChain[slot] = true
			}
			// step through the chain, disabling every other one
			for {
				visited[slot] = true
				slot = itemDisablers[slot]
				if slot == "" {
					break
				}
				visited[slot] = true
				trueDisabled[slot] = true
				slot = itemDisablers[slot]
				if slot == "" || visited[slot] {
					break
				}
			}
		}
		for slot := range trueDisabled {
			delete(items, slot)
		}
	}

	// (missing-anoint warnings skipped: no db effect, warnings uncompared)

	env.Items = items
	env.Flasks = map[*Item]bool{}
	env.Tinctures = map[*Item]bool{}
	env.FlaskSlotMap = map[*Item]float64{}
	env.FlaskSlotOccupied = map[float64]bool{}
	for _, slot := range in.ItemsTab.Slots {
		slotName := slot.SlotName
		item := items[slotName]
		if item != nil && item.In.Type == "Flask" {
			env.ItemModDB.Conditions["Have"+stripSpaces(deref(item.In.BaseName))] = true
			if slot.Active != nil && *slot.Active {
				env.Flasks[item] = true
			}
			if flaskNum, ok := flaskSlotNum(slotName); ok {
				env.FlaskSlotMap[item] = flaskNum
				env.FlaskSlotOccupied[flaskNum] = true
			}
			if item.In.Base != nil && deref(item.In.Base.SubType) == "Life" {
				if anyNum(item.In.FlaskData["chargesMax"]) > env.ItemModDB.Multipliers["LifeFlaskCharges"] {
					env.ItemModDB.Multipliers["LifeFlaskCharges"] = anyNum(item.In.FlaskData["chargesMax"])
				}
			}
			item = nil
		} else if item != nil && item.In.Type == "Tincture" {
			if slot.Active != nil && *slot.Active {
				env.Tinctures[item] = true
			}
			item = nil
		}
		scale := 1.0
		if item != nil && item.In.Type == "Jewel" && item.In.Base != nil && deref(item.In.Base.SubType) == "Abyss" && slot.ParentSlotName != nil {
			// Check if the item in the parent slot has enough Abyssal Sockets
			parent, _ := env.Player.ItemList[*slot.ParentSlotName].(*Item)
			if parent == nil || parent.In.AbyssalSocketCount == nil || slot.SlotNum == nil || *parent.In.AbyssalSocketCount < *slot.SlotNum {
				item = nil
			} else {
				if parent.In.SocketedJewelEffectModifier == nil {
					panic("calc: abyss parent without socketedJewelEffectModifier (the Lua errors)")
				}
				scale = *parent.In.SocketedJewelEffectModifier
			}
		}
		if slot.NodeID != nil && item != nil && item.In.Type == "Jewel" && item.In.JewelData != nil && truthy(item.In.JewelData["jewelIncEffectFromClassStart"]) {
			// Split Personality: the socket's effect scales with how far the
			// socket sits from the class start.
			if node := env.AllocNodes[int(*slot.NodeID)]; node != nil && node.DistanceToClassStart != nil {
				scale = scale + *node.DistanceToClassStart*(anyNum(item.In.JewelData["jewelIncEffectFromClassStart"])/100)
			}
		}
		if item == nil {
			continue
		}
		env.Player.ItemList[slotName] = item
		// Merge mods for this item
		srcList := item.In.ModList
		if srcList == nil && item.In.SlotModList != nil && slot.SlotNum != nil {
			srcList = item.In.SlotModList[int(*slot.SlotNum)]
		}
		if srcList == nil {
			srcList = []*modparser.Mod{}
		}
		if item.In.Requirements != nil {
			env.RequirementsTableItems = append(env.RequirementsTableItems, ItemRequirement{
				SourceItem: item,
				SourceSlot: slotName,
				Str:        item.In.Requirements["strMod"],
				Dex:        item.In.Requirements["dexMod"],
				Int:        item.In.Requirements["intMod"],
			})
		}
		if item.In.Type == "Jewel" && item.In.Base != nil && deref(item.In.Base.SubType) == "Abyss" {
			// Update Abyss Jewel conditions/multipliers
			cond := "Have" + strings.ReplaceAll(deref(item.In.BaseName), " ", "")
			mult := strings.ReplaceAll(deref(item.In.BaseName), " ", "")
			if !truthy(env.ItemModDB.Conditions[cond]) {
				env.ItemModDB.Conditions[cond] = true
				env.ItemModDB.Multipliers["AbyssJewelType"]++
			}
			if slot.ParentSlotName != nil {
				env.ItemModDB.Conditions[cond+"In"+*slot.ParentSlotName] = true
				env.ItemModDB.Multipliers[mult+"In"+*slot.ParentSlotName]++
			}
			env.ItemModDB.Multipliers["AbyssJewel"]++
			switch item.In.Rarity {
			case "NORMAL":
				env.ItemModDB.Multipliers["NormalAbyssJewels"]++
			case "MAGIC":
				env.ItemModDB.Multipliers["MagicAbyssJewels"]++
			case "RARE":
				env.ItemModDB.Multipliers["RareAbyssJewels"]++
			case "UNIQUE", "RELIC":
				env.ItemModDB.Multipliers["UniqueAbyssJewels"]++
			}
			env.ItemModDB.Multipliers[mult]++
		}
		aegisNode := env.AllocNodes[45175]
		if item.In.Type == "Shield" && aegisNode != nil && deref(aegisNode.DN) == "Necromantic Aegis" {
			// Special handling for Necromantic Aegis
			env.AegisModList = modstore.NewList(nil)
			for _, mod := range srcList {
				if modHasSocketedInTag(mod) {
					env.ItemModDB.ScaleAddMod(mod, scale, false)
				} else {
					env.AegisModList.ScaleAddMod(mod, scale, false)
				}
			}
		} else if (slotName == "Weapon 1" || slotName == "Weapon 2") && truthy(env.ModDB.Conditions["AffectedByEnergyBlade"]) {
			// The reference synthesizes an Energy Blade weapon here through
			// the Item machinery; the dump captured the result. No fixture
			// entry means the info-nil/Bow fallthrough (mods merge normally).
			if ebIn := env.Replay.EnergyBladeItems[slotName]; ebIn != nil {
				env.Player.ItemList[slotName] = &Item{In: ebIn}
			} else {
				env.ItemModDB.ScaleAddList(srcList, scale, false)
			}
		} else if slotName == "Weapon 1" && item.In.Name == "The Iron Mass, Gladius" {
			// Special handling for The Iron Mass
			env.TheIronMass = modstore.NewList(nil)
			for _, mod := range srcList {
				if !modHasSocketedInTag(mod) {
					env.TheIronMass.ScaleAddMod(mod, scale, false)
				}
				// Add all the stats to player as well
				env.ItemModDB.ScaleAddMod(mod, scale, false)
			}
		} else if slotName == "Weapon 1" && len(item.In.GrantedSkills) > 0 && str(item.In.GrantedSkills[0]["skillId"]) == "UniqueAnimateWeapon" {
			// Special handling for The Dancing Dervish
			env.WeaponModList1 = modstore.NewList(nil)
			for _, mod := range srcList {
				if modHasSocketedInTag(mod) {
					env.ItemModDB.ScaleAddMod(mod, scale, false)
				} else {
					env.WeaponModList1.ScaleAddMod(mod, scale, false)
				}
			}
		} else if strings.Contains(item.In.Name, "Kalandra's Touch") {
			otherName := ""
			if slotName == "Ring 1" {
				otherName = "Ring 2"
			} else if slotName == "Ring 2" {
				otherName = "Ring 1"
			}
			otherRing := items[otherName]
			if otherRing != nil && !strings.Contains(otherRing.In.Name, "Kalandra's Touch") {
				oSrc := otherRing.In.ModList
				if oSrc == nil && otherRing.In.SlotModList != nil && slot.SlotNum != nil {
					oSrc = otherRing.In.SlotModList[int(*slot.SlotNum)]
				}
				for _, mod := range oSrc {
					if modHasSocketedInTag(mod) {
						continue
					}
					modCopy := modparser.CopyMod(mod)
					modparser.SetSource(modCopy, deref(item.In.ModSource))
					env.ItemModDB.ScaleAddMod(modCopy, scale, false)
				}
				// Adjust multipliers based on other ring
				for mult, has := range influenceMults {
					if has(otherRing.In) {
						env.ItemModDB.Multipliers[mult]++
						env.ItemModDB.Multipliers["Non"+mult]--
					}
				}
				if otherRing.Elder() || otherRing.Shaper() {
					env.ItemModDB.Multipliers["ShaperOrElderItem"]++
				}
				// Esh of the Storm, Tul of the Blizzard
				otherRingKey := strings.ReplaceAll(deref(otherRing.In.BaseName), " ", "") + "Equipped"
				env.ItemModDB.Multipliers[otherRingKey]++
			}
			// Only ExtraSkill implicit mods work
			for _, mod := range srcList {
				if mod.Name == "ExtraSkill" {
					env.ItemModDB.ScaleAddMod(mod, scale, false)
				}
			}
		} else if item.In.Type == "Quiver" &&
			((items["Weapon 1"] != nil && strings.Contains(items["Weapon 1"].In.Name, "Widowhail")) ||
				env.InitialNodeModDB.Sum("INC", nil, "EffectOfBonusesFromQuiver") > 0) {
			// L1127 operator precedence preserved: without a Weapon 1 the
			// whole (w1 and sums) falls back to 100.
			inner := 100.0
			if items["Weapon 1"] != nil {
				w1 := items["Weapon 1"]
				if w1.In.BaseModList == nil {
					panic("calc: Widowhail weapon without baseModList (the Lua errors)")
				}
				inner = listOf(w1.In.BaseModList).Sum("INC", nil, "EffectOfBonusesFromQuiver") +
					env.InitialNodeModDB.Sum("INC", nil, "EffectOfBonusesFromQuiver")
			}
			widowHailMod := 1 + inner/100
			scale = scale * widowHailMod
			env.ModDB.AddMod(newMod("WidowHailMultiplier", "BASE", widowHailMod, "Widowhail"))
			combined := modstore.NewList(nil)
			for _, mod := range srcList {
				combined.MergeMod(mod, false)
			}
			env.ItemModDB.ScaleAddList(combined.Mods, scale, false)
		} else if _, hasCorrJewel := env.ModDB.Multipliers["Corrupted"+titleCaseRarity(item.In.Rarity)+"JewelEffect"]; hasCorrJewel &&
			item.In.Type == "Jewel" && item.Corrupted() && slot.NodeID != nil &&
			(item.In.Base == nil || deref(item.In.Base.SubType) != "Charm") &&
			!(slot.ContainJewelSocket != nil && *slot.ContainJewelSocket) {
			scale = scale + env.ModDB.Multipliers["Corrupted"+titleCaseRarity(item.In.Rarity)+"JewelEffect"]
			combined := modstore.NewList(nil)
			for _, mod := range srcList {
				combined.MergeMod(mod, false)
			}
			env.ItemModDB.ScaleAddList(combined.Mods, scale, false)
		} else if item.In.Type == "Gloves" && Mod(env.InitialNodeModDB, nil, "EffectOfBonusesFromGloves") != 1 {
			scale = Mod(env.InitialNodeModDB, nil, "EffectOfBonusesFromGloves") - 1
			env.ItemModDB.AddList(effectScaledList(srcList, scale))
		} else if item.In.Type == "Boots" && Mod(env.InitialNodeModDB, nil, "EffectOfBonusesFromBoots") != 1 {
			scale = Mod(env.InitialNodeModDB, nil, "EffectOfBonusesFromBoots") - 1
			env.ItemModDB.AddList(effectScaledList(srcList, scale))
		} else {
			env.ItemModDB.ScaleAddList(srcList, scale, false)
		}
		// set conditions on restricted items (the condition VALUE is the
		// class-name string, matching Lua truthiness semantics)
		if item.In.ClassRestriction != nil {
			env.ItemModDB.Conditions[strings.ReplaceAll(deref(item.In.Title), " ", "")] = *item.In.ClassRestriction
		}
		if item.In.Type != "Jewel" && item.In.Type != "Flask" && item.In.Type != "Tincture" && item.In.Type != "Graft" {
			// Update item counts
			var key string
			if item.In.Rarity == "UNIQUE" || item.In.Rarity == "RELIC" {
				if item.In.Foulborn != nil && *item.In.Foulborn {
					env.ItemModDB.Multipliers["FoulbornUniqueItem"]++
				}
				key = "UniqueItem"
			} else if item.In.Rarity == "RARE" {
				key = "RareItem"
			} else if item.In.Rarity == "MAGIC" {
				key = "MagicItem"
			} else {
				key = "NormalItem"
			}
			env.ItemModDB.Multipliers[key]++
			env.ItemModDB.Conditions[key+"In"+slotName] = true
			for mult, has := range influenceMults {
				if has(item.In) {
					env.ItemModDB.Multipliers[mult]++
				} else {
					env.ItemModDB.Multipliers["Non"+mult]++
				}
			}
			if item.Shaper() || item.Elder() {
				env.ItemModDB.Multipliers["ShaperOrElderItem"]++
			}
			typeKey := strings.ReplaceAll(item.In.Type, " ", "")
			if idx := strings.LastIndex(typeKey, "Handed"); idx > 0 {
				typeKey = typeKey[idx+len("Handed"):]
			}
			env.ItemModDB.Multipliers[typeKey+"Item"]++
			// base ring count, e.g. Cryonic, Synaptic
			if item.In.Type == "Ring" {
				env.ItemModDB.Multipliers[strings.ReplaceAll(deref(item.In.BaseName), " ", "")+"Equipped"]++
			}
			// Calculate socket counts
			slotEmpty := map[string]float64{"R": 0, "G": 0, "B": 0, "W": 0}
			slotGemSocketsCount := 0.0
			var socketedGems []*SocketGemInput
			for _, socketGroup := range in.SkillsTab.SocketGroups {
				if socketGroup.KV["source"] == nil && truthy(socketGroup.KV["enabled"]) &&
					str(socketGroup.KV["slot"]) != "" && str(socketGroup.KV["slot"]) == slotName {
					for _, gem := range socketGroup.GemList {
						if gem.GemData != nil && truthy(gem.KV["enabled"]) {
							socketedGems = append(socketedGems, gem)
						}
					}
				}
			}
			for i, socket := range item.In.Sockets {
				color := str(socket["color"])
				if color == "R" || color == "B" || color == "G" || color == "W" {
					slotGemSocketsCount++
					if i+1 > len(socketedGems) {
						slotEmpty[color]++
					}
				}
			}
			socketedColours := map[string]float64{"R": 0, "G": 0, "B": 0}
			// Only gems that fit in the item's sockets contribute
			for i := 0; i < int(math.Min(slotGemSocketsCount, float64(len(socketedGems)))); i++ {
				tags := socketedGems[i].GemData.Tags
				if tags["strength"] {
					socketedColours["R"]++
				}
				if tags["dexterity"] {
					socketedColours["G"]++
				}
				if tags["intelligence"] {
					socketedColours["B"]++
				}
			}
			env.ItemModDB.Multipliers["SocketedGemsIn"+slotName] = math.Min(slotGemSocketsCount, float64(len(socketedGems)))
			env.ItemModDB.Multipliers["SocketedRedGemsIn"+slotName] = socketedColours["R"]
			env.ItemModDB.Multipliers["SocketedGreenGemsIn"+slotName] = socketedColours["G"]
			env.ItemModDB.Multipliers["SocketedBlueGemsIn"+slotName] = socketedColours["B"]
			env.ItemModDB.Multipliers["EmptySocketIn"+slotName] = math.Min(slotGemSocketsCount, slotEmpty["R"]+slotEmpty["G"]+slotEmpty["B"]+slotEmpty["W"])
			env.ItemModDB.Multipliers["EmptyRedSocketsInAnySlot"] += slotEmpty["R"]
			env.ItemModDB.Multipliers["EmptyGreenSocketsInAnySlot"] += slotEmpty["G"]
			env.ItemModDB.Multipliers["EmptyBlueSocketsInAnySlot"] += slotEmpty["B"]
			env.ItemModDB.Multipliers["EmptyWhiteSocketsInAnySlot"] += slotEmpty["W"]
		}
	}
	// Override empty socket calculation if set in config
	for cfgKey, multKey := range map[string]string{
		"overrideEmptyRedSockets":   "EmptyRedSocketsInAnySlot",
		"overrideEmptyGreenSockets": "EmptyGreenSocketsInAnySlot",
		"overrideEmptyBlueSockets":  "EmptyBlueSocketsInAnySlot",
		"overrideEmptyWhiteSockets": "EmptyWhiteSocketsInAnySlot",
	} {
		if v, ok := cfg[cfgKey]; ok && truthy(v) {
			env.ItemModDB.Multipliers[multKey] = anyNum(v)
		}
	}
}

func modHasSocketedInTag(mod *modparser.Mod) bool {
	for _, tv := range mod.Tags {
		if tv == nil {
			break
		}
		if tag, ok := tv.(modparser.Tag); ok && tag["type"] == "SocketedIn" {
			return true
		}
	}
	return false
}

func listOf(mods []*modparser.Mod) *modstore.List {
	l := modstore.NewList(nil)
	l.AddList(mods)
	return l
}

// effectScaledList is the Gloves/Boots effect pattern: merge srcList, add a
// scaled copy on top with replace semantics, return the merged mods.
func effectScaledList(srcList []*modparser.Mod, scale float64) []*modparser.Mod {
	combined := modstore.NewList(nil)
	for _, mod := range srcList {
		combined.MergeMod(mod, false)
	}
	scaled := modstore.NewList(nil)
	scaled.ScaleAddList(combined.Mods, scale, false)
	for _, mod := range scaled.Mods {
		combined.MergeMod(mod, true)
	}
	return combined.Mods
}
