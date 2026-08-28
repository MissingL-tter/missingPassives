// Package skills ports Classes/SkillsTab.lua's logic half: loading the
// build XML's <Skills> element into socket groups and gem instances,
// resolving gems against the game data (ProcessSocketGroup), the imbued
// support map, and the socket-colour matching pass the calc reads
// (UpdateSocketGroups). No view state: gem colour codes, controls and
// display fields stay unported.
//
// Groups and gems are scalar bags plus resolution pointers — the same
// shape the reference's Lua tables have and the calc input consumes.
package skills

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/item"
)

// --- XML shapes (attribute presence matters: tostring(nil) saves as the
// string "nil", which the reference reads back as a truthy string) ---

type XMLGem struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

type XMLSkill struct {
	Attrs []xml.Attr `xml:",any,attr"`
	Gems  []XMLGem   `xml:"Gem"`
}

type XMLSkillSet struct {
	Attrs  []xml.Attr `xml:",any,attr"`
	Skills []XMLSkill `xml:"Skill"`
}

// XMLSkills is the <Skills> element. The reference iterates its children in
// order and accepts both the old flat <Skill> format and <SkillSet>
// containers; interleaving of the two forms is not preserved here (no save
// mixes them).
type XMLSkills struct {
	Attrs  []xml.Attr    `xml:",any,attr"`
	Sets   []XMLSkillSet `xml:"SkillSet"`
	Skills []XMLSkill    `xml:"Skill"`
}

func attrOf(attrs []xml.Attr, name string) (string, bool) {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// --- model ---

// GemInstance is one gem: the reference's scalar fields as a bag plus the
// resolved data references.
type GemInstance struct {
	KV            map[string]any
	GemData       *data.Gem
	GrantedEffect *data.GrantedEffect
}

// SocketGroup is one socket group (one <Skill> element).
type SocketGroup struct {
	KV      map[string]any
	GemList []*GemInstance
}

type SkillSet struct {
	ID              int
	Title           string
	SocketGroupList []*SocketGroup
}

// Tab is the loaded skills tab (logic half).
type Tab struct {
	SkillSets         map[int]*SkillSet
	SkillSetOrderList []int
	ActiveSkillSetID  int
	SocketGroupList   []*SocketGroup // the active set's list

	DefaultGemLevel     string
	DefaultGemQuality   float64
	ImbuedSupportBySlot map[string]*data.GrantedEffect

	characterLevel float64
}

// Load ports SkillsTabClass:Load. characterLevel feeds the
// "characterLevel" default-gem-level policy.
func Load(x *XMLSkills, characterLevel float64) *Tab {
	t := &Tab{
		SkillSets:           map[int]*SkillSet{},
		ImbuedSupportBySlot: map[string]*data.GrantedEffect{},
		characterLevel:      characterLevel,
	}
	// Legacy default-gem-level settings.
	mgl, _ := attrOf(x.Attrs, "matchGemLevelToCharacterLevel")
	dgl, hasDgl := attrOf(x.Attrs, "defaultGemLevel")
	switch {
	case mgl == "true":
		t.DefaultGemLevel = "characterLevel"
	case hasDgl && !isNumber(dgl):
		// SelByValue: an unknown key leaves the control on its default.
		switch dgl {
		case "normalMaximum", "corruptedMaximum", "awakenedMaximum", "characterLevel", "levelOne":
			t.DefaultGemLevel = dgl
		default:
			t.DefaultGemLevel = "normalMaximum"
		}
	default:
		t.DefaultGemLevel = "normalMaximum"
	}
	if q, ok := attrOf(x.Attrs, "defaultGemQuality"); ok {
		if n, err := strconv.ParseFloat(q, 64); err == nil {
			t.DefaultGemQuality = math.Max(math.Min(n, 23), 0)
		}
	}

	for _, node := range x.Skills {
		// Old format: initialize skill set 1 on first use.
		if len(t.SkillSetOrderList) == 0 {
			t.SkillSetOrderList = []int{1}
			t.newSkillSet(1)
		}
		t.loadSkill(&node, 1)
	}
	for _, setNode := range x.Sets {
		idStr, _ := attrOf(setNode.Attrs, "id")
		id := int(num(idStr))
		set := t.newSkillSet(id)
		set.Title, _ = attrOf(setNode.Attrs, "title")
		t.SkillSetOrderList = append(t.SkillSetOrderList, id)
		for i := range setNode.Skills {
			t.loadSkill(&setNode.Skills[i], id)
		}
	}
	active := 1
	if a, ok := attrOf(x.Attrs, "activeSkillSet"); ok {
		if n, err := strconv.ParseFloat(a, 64); err == nil {
			active = int(n)
		}
	}
	t.SetActiveSkillSet(active)
	return t
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func num(s string) float64 {
	n, _ := strconv.ParseFloat(s, 64)
	return n
}

func (t *Tab) newSkillSet(id int) *SkillSet {
	set := &SkillSet{ID: id}
	t.SkillSets[id] = set
	return set
}

// loadSkill ports SkillsTabClass:LoadSkill for one <Skill> element.
func (t *Tab) loadSkill(node *XMLSkill, skillSetID int) {
	kv := map[string]any{}
	group := &SocketGroup{KV: kv}
	attr := func(name string) (string, bool) { return attrOf(node.Attrs, name) }

	active, _ := attr("active")
	enabled, _ := attr("enabled")
	kv["enabled"] = active == "true" || enabled == "true"
	if v, ok := attr("includeInFullDPS"); ok {
		kv["includeInFullDPS"] = v == "true"
	}
	if v, ok := attr("groupCount"); ok && isNumber(v) {
		kv["groupCount"] = num(v)
	}
	if v, ok := attr("label"); ok {
		kv["label"] = v
	}
	slot, hasSlot := attr("slot")
	if hasSlot {
		kv["slot"] = slot
	}
	if v, ok := attr("source"); ok {
		kv["source"] = v
	}
	kv["mainActiveSkill"] = numOr(attr("mainActiveSkill"))(1)
	kv["mainActiveSkillCalcs"] = numOr(attr("mainActiveSkillCalcs"))(1)
	if v, ok := attr("imbuedSupport"); ok && hasSlot {
		kv["imbuedSupport"] = v
	}

	for i := range node.Gems {
		group.GemList = append(group.GemList, t.loadGem(&node.Gems[i]))
	}
	if v, ok := attr("skillPart"); ok && isNumber(v) && len(group.GemList) > 0 {
		group.GemList[0].KV["skillPart"] = num(v)
	}
	t.ProcessSocketGroup(group)
	set := t.SkillSets[skillSetID]
	set.SocketGroupList = append(set.SocketGroupList, group)
}

func numOr(s string, ok bool) func(def float64) float64 {
	return func(def float64) float64 {
		if ok {
			if n, err := strconv.ParseFloat(s, 64); err == nil {
				return n
			}
		}
		return def
	}
}

func (t *Tab) loadGem(child *XMLGem) *GemInstance {
	kv := map[string]any{}
	gem := &GemInstance{KV: kv}
	attr := func(name string) (string, bool) { return attrOf(child.Attrs, name) }

	nameSpec, _ := attr("nameSpec")
	kv["nameSpec"] = sanitiseText(nameSpec)
	if gameID, ok := attr("gemId"); ok {
		var gemData *data.Gem
		possibleVariants := data.GemsByGameId[gameID]
		if possibleVariants != nil {
			if variantID, ok := attr("variantId"); ok {
				gemData = possibleVariants[variantID]
			} else if skillID, ok := attr("skillId"); ok {
				// Old format relying on granted-effect id uniqueness; the
				// reference's pairs() pick is unique when it matches at all.
				for _, vid := range sortedKeys(possibleVariants) {
					if possibleVariants[vid].GrantedEffectId == skillID {
						gemData = possibleVariants[vid]
						break
					}
				}
			}
		}
		if gemData != nil {
			kv["gemId"] = gemData.Id
			kv["skillId"] = gemData.GrantedEffectId
			kv["nameSpec"] = gemData.Name
		}
	} else if skillID, ok := attr("skillId"); ok {
		if grantedEffect := data.Skills[skillID]; grantedEffect != nil {
			if gemID, ok := data.GemForSkill[grantedEffect]; ok {
				kv["gemId"] = gemID
			}
			kv["skillId"] = grantedEffect.Id
			kv["nameSpec"] = grantedEffect.Name
		}
	}
	if v, ok := attr("level"); ok && isNumber(v) {
		kv["level"] = num(v)
	}
	if v, ok := attr("quality"); ok && isNumber(v) {
		kv["quality"] = num(v)
	}
	kv["nameSpec"] = sanitiseText(kv["nameSpec"].(string))
	en, hasEn := attr("enabled")
	kv["enabled"] = !hasEn || en == "true"
	eg1, hasEg1 := attr("enableGlobal1")
	kv["enableGlobal1"] = !hasEg1 || eg1 == "true"
	eg2, _ := attr("enableGlobal2")
	kv["enableGlobal2"] = eg2 == "true"
	kv["count"] = numOr(attr("count"))(1)
	for _, key := range []string{
		"skillPart", "skillPartCalcs", "skillStageCount", "skillStageCountCalcs",
		"skillMineCount", "skillMineCountCalcs", "skillMinionItemSet",
		"skillMinionItemSetCalcs", "skillMinionSkill", "skillMinionSkillCalcs",
	} {
		if v, ok := attr(key); ok && isNumber(v) {
			kv[key] = num(v)
		}
	}
	for _, key := range []string{"skillMinion", "skillMinionCalcs"} {
		if v, ok := attr(key); ok {
			kv[key] = v
		}
	}
	return gem
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// small maps; insertion sort keeps this dependency-free
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// sanitiseText ports Common.lua sanitiseText: strip balanced <> markup and
// normalise unicode hyphens, but only when a byte >= 128 or '<' occurs.
func sanitiseText(text string) string {
	needs := false
	for i := 0; i < len(text); i++ {
		if text[i] >= 128 || text[i] == '<' {
			needs = true
			break
		}
	}
	if !needs {
		return text
	}
	// %b<> — balanced-delimiter strip.
	var b strings.Builder
	depth := 0
	start := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '<':
			if depth == 0 {
				b.WriteString(text[start:i])
			}
			depth++
		case '>':
			if depth > 0 {
				depth--
				if depth == 0 {
					start = i + 1
				}
			}
		}
	}
	if depth == 0 {
		b.WriteString(text[start:])
	} else {
		// unbalanced '<': Lua %b<> fails to match, leaves the rest as-is
		b.WriteString(text[start:])
	}
	s := b.String()
	for _, hy := range []string{"‐", "‑", "‒"} {
		s = strings.ReplaceAll(s, hy, "-")
	}
	return s
}

// ProcessSocketGroup ports SkillsTabClass:ProcessSocketGroup (gem colour
// codes skipped: view-only).
func (t *Tab) ProcessSocketGroup(group *SocketGroup) {
	for _, gem := range group.GemList {
		kv := gem.KV
		if _, ok := kv["nameSpec"]; !ok {
			kv["nameSpec"] = ""
		}
		var prevDefaultLevel float64
		hasPrev := false
		if gem.GemData != nil {
			prevDefaultLevel, hasPrev = gem.GemData.NaturalMaxLevel, true
		} else if truthy(kv["new"]) {
			prevDefaultLevel, hasPrev = 20, true
		}
		gem.GemData, gem.GrantedEffect = nil, nil
		if gemID, ok := kv["gemId"].(string); ok {
			// Specified by gem ID (skills granted by skill gems).
			delete(kv, "errMsg")
			gem.GemData = data.Gems[gemID]
			if gem.GemData != nil {
				kv["nameSpec"] = gem.GemData.Name
				kv["skillId"] = gem.GemData.GrantedEffectId
			}
		} else if skillID, ok := kv["skillId"].(string); ok {
			// Specified by skill ID (skills granted by items).
			delete(kv, "errMsg")
			if grantedEffect := data.Skills[skillID]; grantedEffect != nil {
				if gemID, ok := data.GemForSkill[grantedEffect]; ok {
					gem.GemData = data.Gems[gemID]
				} else {
					gem.GrantedEffect = grantedEffect
				}
			}
			if truthy(kv["triggered"]) && gem.GrantedEffect != nil {
				if lvl, ok := kv["level"].(float64); ok {
					if sl := gem.GrantedEffect.Levels[lvl]; sl != nil {
						// The reference wipes the SHARED level's cost table;
						// the calc port keeps that per-env
						// (TriggeredCostWipes) — nothing to do at load.
						_ = sl
					}
				}
			}
		} else if strings.ContainsFunc(str(kv["nameSpec"]), notSpace) {
			// Pre-1.4.20 migration by gem name (FindSkillGem). The ported
			// lookup collapses "ambiguous" into nil; the reference's
			// ambiguity message cites two hash-order names, but no corpus
			// build carries an ambiguous spec.
			delete(kv, "errMsg")
			gem.GemData = calc.FindSkillGem(str(kv["nameSpec"]))
			if gem.GemData != nil {
				kv["gemId"] = gem.GemData.Id
				kv["skillId"] = gem.GemData.GrantedEffectId
				kv["nameSpec"] = gem.GemData.Name
			} else {
				kv["errMsg"] = "Unrecognised gem name '" + str(kv["nameSpec"]) + "'"
				delete(kv, "gemId")
				delete(kv, "skillId")
			}
		} else {
			delete(kv, "errMsg")
			delete(kv, "skillId")
		}
		// grantedEffect.unsupported gate: no 3.29 PoE1 skill carries the
		// flag; the data tables have no such field.
		if gem.GemData != nil || gem.GrantedEffect != nil {
			delete(kv, "new")
			grantedEffect := gem.GrantedEffect
			if grantedEffect == nil {
				grantedEffect = gem.GemData.GrantedEffect
			}
			if hasPrev && gem.GemData != nil && gem.GemData.NaturalMaxLevel != prevDefaultLevel {
				kv["level"] = gem.GemData.NaturalMaxLevel
				kv["naturalMaxLevel"] = kv["level"]
			}
			validateGemLevel(gem)
			if gem.GemData != nil {
				lvl := kv["level"].(float64)
				reqLevel := float64(0)
				if sl := grantedEffect.Levels[lvl]; sl != nil {
					reqLevel = sl.Extra["levelRequirement"]
				}
				kv["reqLevel"] = reqLevel
				kv["reqStr"] = calc.GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqStr)
				kv["reqDex"] = calc.GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqDex)
				kv["reqInt"] = calc.GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqInt)
			}
		}
	}
}

func notSpace(r rune) bool {
	return r != ' ' && r != '\t' && r != '\n' && r != '\r' && r != '\f' && r != '\v'
}

func truthy(v any) bool { return v != nil && v != false }

func str(v any) string {
	s, _ := v.(string)
	return s
}

// validateGemLevel ports calcLib.validateGemLevel over the KV bag.
func validateGemLevel(gem *GemInstance) {
	kv := gem.KV
	grantedEffect := gem.GrantedEffect
	if grantedEffect == nil {
		grantedEffect = gem.GemData.GrantedEffect
	}
	level, _ := kv["level"].(float64)
	if grantedEffect.Levels[level] == nil {
		level = math.Max(1, level)
		if n := luaLevelsLen(grantedEffect.Levels); n > 0 {
			level = math.Min(n, level)
		}
	}
	if grantedEffect.Levels[level] == nil && gem.GemData != nil {
		level = gem.GemData.NaturalMaxLevel
	}
	if grantedEffect.Levels[level] == nil {
		// The reference grabs next() — hash-arbitrary; lowest keeps it
		// deterministic (same choice as calc.ValidateGemLevel).
		first, found := 0.0, false
		for lvl := range grantedEffect.Levels {
			if !found || lvl < first {
				first, found = lvl, true
			}
		}
		if found {
			level = first
		}
	}
	kv["level"] = level
}

// luaLevelsLen mirrors Lua # on the levels table: the length of the
// contiguous run from 1.
func luaLevelsLen(levels map[float64]*data.SkillLevel) float64 {
	n := float64(0)
	for levels[n+1] != nil {
		n++
	}
	return n
}

// SetActiveSkillSet ports the logic half of SkillsTabClass:SetActiveSkillSet.
func (t *Tab) SetActiveSkillSet(skillSetID int) {
	if len(t.SkillSetOrderList) == 0 {
		t.SkillSetOrderList = []int{1}
		t.newSkillSet(1)
	}
	if t.SkillSets[skillSetID] == nil {
		skillSetID = t.SkillSetOrderList[0]
	}
	t.SocketGroupList = t.SkillSets[skillSetID].SocketGroupList
	t.RebuildImbuedSupportBySlot()
	t.ActiveSkillSetID = skillSetID
}

// RebuildImbuedSupportBySlot ports the same-named method.
func (t *Tab) RebuildImbuedSupportBySlot() {
	t.ImbuedSupportBySlot = map[string]*data.GrantedEffect{}
	for _, group := range t.SocketGroupList {
		slot, hasSlot := group.KV["slot"].(string)
		imbued, hasImbued := group.KV["imbuedSupport"].(string)
		if hasSlot && hasImbued {
			gemID, ok := data.GemForBaseName[strings.ToLower(imbued)+" support"]
			if !ok {
				continue
			}
			if gem := data.Gems[gemID]; gem != nil && gem.GrantedEffect != nil {
				t.ImbuedSupportBySlot[slot] = gem.GrantedEffect
			}
		}
	}
}

// UpdateSocketGroups ports the same-named method: socket-colour matching
// for gems in item-linked groups. slotItem maps a slot name to the item
// selected in it (nil when empty) — supplied by the caller so the tab
// stays decoupled from the items tab.
func (t *Tab) UpdateSocketGroups(slotItem func(slotName string) *item.Item) {
	colours := [4]string{1: "R", 2: "G", 3: "B"}
	slotSocketedCounts := map[string]int{}
	for _, group := range t.SocketGroupList {
		for _, gem := range group.GemList {
			gem.KV["matchesSocket"] = false
		}
		slot, hasSlot := group.KV["slot"].(string)
		_, hasSource := group.KV["source"]
		if !hasSlot || hasSource {
			continue
		}
		gemOffset := slotSocketedCounts[slot]
		for i, gem := range group.GemList {
			if gem.GemData == nil && gem.GrantedEffect == nil {
				continue
			}
			grantedEffect := gem.GrantedEffect
			if grantedEffect == nil {
				grantedEffect = gem.GemData.GrantedEffect
			}
			gemIdx := gemOffset + i
			it := slotItem(slot)
			if it == nil || it.Sockets == nil {
				continue
			}
			if it.SocketColourAlwaysMatches {
				gem.KV["matchesSocket"] = true
			} else if gemIdx >= len(it.Sockets) {
				// Lua: sockets[gemIdx] is nil, `nil and (...)` assigns nil —
				// which REMOVES the key.
				delete(gem.KV, "matchesSocket")
			} else {
				var gemColour string
				if c := int(grantedEffect.Color); c >= 1 && c <= 3 {
					gemColour = colours[c]
				}
				gem.KV["matchesSocket"] = it.Sockets[gemIdx].Color == gemColour
			}
		}
		slotSocketedCounts[slot] = gemOffset + len(group.GemList)
	}
}
