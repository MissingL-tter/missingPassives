// Package skills ports Classes/SkillsTab.lua's logic half: loading the
// build XML's <Skills> element into socket groups and gem instances,
// resolving gems against the game data (ProcessSocketGroup), the imbued
// support map, and the socket-colour matching pass the calc reads
// (UpdateSocketGroups). No view state: gem colour codes, controls and
// display fields stay unported.
//
// Groups and gems are typed records plus resolution pointers; calc embeds
// them in its socket-group input.
package skills

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
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

// Gem is one gem instance: the reference's scalar fields, typed, plus the
// resolved data references. Strings hold "" for a key the reference never
// set (it never stores an empty id, message, or minion name); Opt fields
// are keys whose absence the reference distinguishes from a stored
// zero/false (calc deletes them, or saves false/0 explicitly).
type Gem struct {
	NameSpec      string
	GemID         string
	SkillID       string
	ErrMsg        string
	Level         float64
	Quality       float64
	Count         util.Opt[float64] // absent on calc-created granted gems
	Enabled       bool
	EnableGlobal1 bool
	EnableGlobal2 bool
	// MatchesSocket: the gem sits in a socket of its own colour (or in an
	// item whose sockets always match).
	MatchesSocket bool
	New           bool
	Triggered     bool
	NoSupports    bool
	// FromItem is calc-stamped on granted gems.
	FromItem bool
	// Requirements are (re)computed only for gem-data gems and stay stale
	// when a gem later resolves by skill id, so presence is independent of
	// GemData.
	ReqLevel        util.Opt[float64]
	ReqStr          util.Opt[float64]
	ReqDex          util.Opt[float64]
	ReqInt          util.Opt[float64]
	NaturalMaxLevel float64 // 0 = never recorded
	TriggerChance   util.Opt[float64]

	// Persisted UI selections the calc reads back and clears when the
	// skill has no parts/stages/mines/minions. A present 0 is truthy to
	// the reference, hence Opt.
	SkillPart               util.Opt[float64]
	SkillPartCalcs          util.Opt[float64]
	SkillStageCount         util.Opt[float64]
	SkillStageCountCalcs    util.Opt[float64]
	SkillMineCount          util.Opt[float64]
	SkillMineCountCalcs     util.Opt[float64]
	SkillMinion             string
	SkillMinionCalcs        string
	SkillMinionSkill        util.Opt[float64]
	SkillMinionSkillCalcs   util.Opt[float64]
	SkillMinionItemSet      util.Opt[float64]
	SkillMinionItemSetCalcs util.Opt[float64]

	GemData       *data.Gem
	GrantedEffect *data.GrantedEffect
}

// SocketGroup is one socket group (one <Skill> element).
type SocketGroup struct {
	Enabled          bool
	IncludeInFullDPS bool
	GroupCount       float64 // 0 = not saved
	Label            string
	Slot             string // "" = no slot
	Source           string // "" = socketed group; else the granting item/node
	ImbuedSupport    string
	// MainActiveSkill(Calcs) default to 1 on load; calc-created groups have
	// none until the main-skill selection writes one.
	MainActiveSkill      util.Opt[float64]
	MainActiveSkillCalcs util.Opt[float64]
	// NoSupports is calc-stamped on granted groups (true or absent).
	NoSupports bool
	// SlotEnabled is calc-stamped by the skills stage.
	SlotEnabled bool
	GemList     []*Gem
}

// Granted reports whether the group is a granted-skill group (has a source).
func (g *SocketGroup) Granted() bool { return g.Source != "" }

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
	group := &SocketGroup{}
	attr := func(name string) (string, bool) { return attrOf(node.Attrs, name) }

	active, _ := attr("active")
	enabled, _ := attr("enabled")
	group.Enabled = active == "true" || enabled == "true"
	fullDPS, _ := attr("includeInFullDPS")
	group.IncludeInFullDPS = fullDPS == "true"
	if v, ok := attr("groupCount"); ok && isNumber(v) {
		group.GroupCount = num(v)
	}
	group.Label, _ = attr("label")
	group.Slot, _ = attr("slot")
	group.Source, _ = attr("source")
	group.MainActiveSkill = util.Some(numOr(attr("mainActiveSkill"))(1))
	group.MainActiveSkillCalcs = util.Some(numOr(attr("mainActiveSkillCalcs"))(1))
	if v, ok := attr("imbuedSupport"); ok && group.Slot != "" {
		group.ImbuedSupport = v
	}

	for i := range node.Gems {
		group.GemList = append(group.GemList, t.loadGem(&node.Gems[i]))
	}
	if v, ok := attr("skillPart"); ok && isNumber(v) && len(group.GemList) > 0 {
		group.GemList[0].SkillPart = util.Some(num(v))
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

func (t *Tab) loadGem(child *XMLGem) *Gem {
	gem := &Gem{}
	attr := func(name string) (string, bool) { return attrOf(child.Attrs, name) }

	nameSpec, _ := attr("nameSpec")
	gem.NameSpec = util.FoldText(nameSpec)
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
			gem.GemID = gemData.Id
			gem.SkillID = gemData.GrantedEffectId
			gem.NameSpec = gemData.Name
		}
	} else if skillID, ok := attr("skillId"); ok {
		if grantedEffect := data.Skills[skillID]; grantedEffect != nil {
			if gemID, ok := data.GemForSkill[grantedEffect]; ok {
				gem.GemID = gemID
			}
			gem.SkillID = grantedEffect.Id
			gem.NameSpec = grantedEffect.Name
		}
	}
	if v, ok := attr("level"); ok && isNumber(v) {
		gem.Level = num(v)
	}
	if v, ok := attr("quality"); ok && isNumber(v) {
		gem.Quality = num(v)
	}
	gem.NameSpec = util.FoldText(gem.NameSpec)
	en, hasEn := attr("enabled")
	gem.Enabled = !hasEn || en == "true"
	eg1, hasEg1 := attr("enableGlobal1")
	gem.EnableGlobal1 = !hasEg1 || eg1 == "true"
	eg2, _ := attr("enableGlobal2")
	gem.EnableGlobal2 = eg2 == "true"
	gem.Count = util.Some(numOr(attr("count"))(1))
	numAttr := func(key string) util.Opt[float64] {
		if v, ok := attr(key); ok && isNumber(v) {
			return util.Some(num(v))
		}
		return util.Opt[float64]{}
	}
	gem.SkillPart = numAttr("skillPart")
	gem.SkillPartCalcs = numAttr("skillPartCalcs")
	gem.SkillStageCount = numAttr("skillStageCount")
	gem.SkillStageCountCalcs = numAttr("skillStageCountCalcs")
	gem.SkillMineCount = numAttr("skillMineCount")
	gem.SkillMineCountCalcs = numAttr("skillMineCountCalcs")
	gem.SkillMinionItemSet = numAttr("skillMinionItemSet")
	gem.SkillMinionItemSetCalcs = numAttr("skillMinionItemSetCalcs")
	gem.SkillMinionSkill = numAttr("skillMinionSkill")
	gem.SkillMinionSkillCalcs = numAttr("skillMinionSkillCalcs")
	gem.SkillMinion, _ = attr("skillMinion")
	gem.SkillMinionCalcs, _ = attr("skillMinionCalcs")
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

// ProcessSocketGroup ports SkillsTabClass:ProcessSocketGroup (gem colour
// codes skipped: view-only).
func (t *Tab) ProcessSocketGroup(group *SocketGroup) {
	for _, gem := range group.GemList {
		var prevDefaultLevel float64
		hasPrev := false
		if gem.GemData != nil {
			prevDefaultLevel, hasPrev = gem.GemData.NaturalMaxLevel, true
		} else if gem.New {
			prevDefaultLevel, hasPrev = 20, true
		}
		gem.GemData, gem.GrantedEffect = nil, nil
		switch {
		case gem.GemID != "":
			// Specified by gem ID (skills granted by skill gems).
			gem.ErrMsg = ""
			gem.GemData = data.Gems[gem.GemID]
			if gem.GemData != nil {
				gem.NameSpec = gem.GemData.Name
				gem.SkillID = gem.GemData.GrantedEffectId
			}
		case gem.SkillID != "":
			// Specified by skill ID (skills granted by items).
			gem.ErrMsg = ""
			if grantedEffect := data.Skills[gem.SkillID]; grantedEffect != nil {
				if gemID, ok := data.GemForSkill[grantedEffect]; ok {
					gem.GemData = data.Gems[gemID]
				} else {
					gem.GrantedEffect = grantedEffect
				}
			}
			// The reference wipes the SHARED level's cost table for a
			// triggered granted effect; the calc port keeps that per-env
			// (TriggeredCostWipes) -- nothing to do at load.
		case strings.ContainsFunc(gem.NameSpec, notSpace):
			// Pre-1.4.20 migration by gem name (FindSkillGem). The ported
			// lookup collapses "ambiguous" into nil; the reference's
			// ambiguity message cites two hash-order names, but no corpus
			// build carries an ambiguous spec.
			gem.ErrMsg = ""
			gem.GemData = FindSkillGem(gem.NameSpec)
			if gem.GemData != nil {
				gem.GemID = gem.GemData.Id
				gem.SkillID = gem.GemData.GrantedEffectId
				gem.NameSpec = gem.GemData.Name
			} else {
				gem.ErrMsg = "Unrecognised gem name '" + gem.NameSpec + "'"
				gem.GemID, gem.SkillID = "", ""
			}
		default:
			gem.ErrMsg, gem.SkillID = "", ""
		}
		// grantedEffect.unsupported gate: no 3.29 PoE1 skill carries the
		// flag; the data tables have no such field.
		if gem.GemData != nil || gem.GrantedEffect != nil {
			gem.New = false
			grantedEffect := gem.GrantedEffect
			if grantedEffect == nil {
				grantedEffect = gem.GemData.GrantedEffect
			}
			if hasPrev && gem.GemData != nil && gem.GemData.NaturalMaxLevel != prevDefaultLevel {
				gem.Level = gem.GemData.NaturalMaxLevel
				gem.NaturalMaxLevel = gem.Level
			}
			validateGemLevel(gem)
			if gem.GemData != nil {
				reqLevel := float64(0)
				if sl := grantedEffect.LevelData(gem.Level); sl != nil {
					reqLevel = sl.Extra["levelRequirement"]
				}
				gem.ReqLevel = util.Some(reqLevel)
				gem.ReqStr = util.Some(GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqStr))
				gem.ReqDex = util.Some(GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqDex))
				gem.ReqInt = util.Some(GetGemStatRequirement(reqLevel, grantedEffect.Support, gem.GemData.ReqInt))
			}
		}
	}
}

func notSpace(r rune) bool {
	return r != ' ' && r != '\t' && r != '\n' && r != '\r' && r != '\f' && r != '\v'
}

// validateGemLevel ports calcLib.validateGemLevel.
func validateGemLevel(gem *Gem) {
	grantedEffect := gem.GrantedEffect
	if grantedEffect == nil {
		grantedEffect = gem.GemData.GrantedEffect
	}
	level := gem.Level
	if grantedEffect.LevelData(level) == nil {
		level = math.Max(1, level)
		if n := grantedEffect.LevelCount(); n > 0 {
			level = math.Min(float64(n), level)
		}
	}
	if grantedEffect.LevelData(level) == nil && gem.GemData != nil {
		level = gem.GemData.NaturalMaxLevel
	}
	if grantedEffect.LevelData(level) == nil {
		// The reference grabs next() -- hash-arbitrary; lowest keeps it
		// deterministic (same choice as calc.ValidateGemLevel).
		first, found := 0, false
		for lvl := range grantedEffect.Levels {
			if !found || lvl < first {
				first, found = lvl, true
			}
		}
		if found {
			level = float64(first)
		}
	}
	gem.Level = level
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
		if group.Slot != "" && group.ImbuedSupport != "" {
			gemID, ok := data.GemForBaseName[strings.ToLower(group.ImbuedSupport)+" support"]
			if !ok {
				continue
			}
			if gem := data.Gems[gemID]; gem != nil && gem.GrantedEffect != nil {
				t.ImbuedSupportBySlot[group.Slot] = gem.GrantedEffect
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
			gem.MatchesSocket = false
		}
		if group.Slot == "" || group.Granted() {
			continue
		}
		slot := group.Slot
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
			if it == nil {
				continue
			}
			var gemColour string
			if c := int(grantedEffect.Color); c >= 1 && c <= 3 {
				gemColour = colours[c]
			}
			gem.MatchesSocket = it.SocketColourAlwaysMatches ||
				(gemIdx < len(it.Sockets) && it.Sockets[gemIdx].Color == gemColour)
		}
		slotSocketedCounts[slot] = gemOffset + len(group.GemList)
	}
}
