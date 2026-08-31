// Package build assembles a calc.BuildInput from a saved Path of Building
// build file: the item pool and its slot table, the passive spec, and the
// skills tab, each loaded through the package that owns it.
//
// What it does not cover is the config tab (Classes/ConfigTab.lua and
// Modules/ConfigOptions.lua: 580 options carrying 524 apply closures,
// none of them ported). Load leaves ConfigInput, ConfigPlaceholder,
// ConfigModList and ConfigEnemyModList unset, and the calc falls back to
// its own documented defaults. Every build in the corpus draws 31 to 48
// modifiers from those closures - option defaults as much as user
// selections - so a build assembled here computes without them, not
// around them. Until the config tab is ported, use this to drive the
// engine, not to reproduce the application's numbers.
package build

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/skills"
	"github.com/MissingL-tter/missingPassives/tree"
)

// Build is one loaded build file: the calc's input plus the models it was
// projected from, so a caller that wants the items or the spec does not
// have to parse the file twice.
type Build struct {
	Input  *calc.BuildInput
	Spec   *tree.Spec
	Items  map[int]*item.Item
	Skills *skills.Tab
}

// Load assembles a build file against an already-loaded tree. The tree's
// version must match the build's active spec: a spec saved on another
// tree needs that tree's node ids and mod text.
func Load(blob []byte, tr *tree.Tree) (*Build, error) {
	doc, err := Decode(blob)
	if err != nil {
		return nil, fmt.Errorf("decode build: %w", err)
	}
	// The spec reads the item pool for its socketed jewels, so the items
	// load first.
	items := loadItems(doc)
	spec, err := loadSpec(doc, tr, items)
	if err != nil {
		return nil, err
	}
	specElem := activeSpec(doc)

	level := attrNum(doc.Build.Level, 1)
	sets, active := loadItemSets(doc)
	slots := slotTable(spec)
	fillSlots(slots, spec, items, active)

	tab := skills.Load(&doc.Skills, level)
	slotItem := map[string]*item.Item{}
	for _, s := range slots {
		if s.ItemID != nil {
			slotItem[s.SlotName] = items[int(*s.ItemID)]
		}
	}
	tab.UpdateSocketGroups(func(slotName string) *item.Item { return slotItem[slotName] })

	in := &calc.BuildInput{
		CharacterLevel:  level,
		ClassID:         float64(spec.CurClassID),
		CurClassName:    spec.CurClassName,
		TreeVersion:     specElem.TreeVersion,
		MainSocketGroup: mainSocketGroup(doc),
		SpectreList:     spectreList(doc),
		Spec:            SpecInput(spec, items),
		SkillsTab:       SkillsTabInput(tab),
		ItemsTab: &calc.ItemsTabInput{
			Slots:            slots,
			Items:            itemInputs(items),
			ItemSets:         sets,
			ItemSetOrderList: setOrder(doc),
		},
	}
	if active != nil && active.useSecondWeaponSet {
		in.ItemsTab.UseSecondWeaponSet = truePtr()
	}
	if class := tr.Classes[spec.CurClassID]; class != nil {
		in.ClassStats = calc.ClassStats{
			BaseStr: class.BaseStr,
			BaseDex: class.BaseDex,
			BaseInt: class.BaseInt,
		}
	}
	return &Build{Input: in, Spec: spec, Items: items, Skills: tab}, nil
}

// activeSpec is the <Spec> the build has selected, defaulting to the
// first the way the reference's spec list does.
func activeSpec(doc *Doc) specElem {
	idx := doc.Tree.ActiveSpec
	if idx < 1 || idx > len(doc.Tree.Specs) {
		idx = 1
	}
	if len(doc.Tree.Specs) == 0 {
		return specElem{}
	}
	return doc.Tree.Specs[idx-1]
}

func loadSpec(doc *Doc, tr *tree.Tree, items map[int]*item.Item) (*tree.Spec, error) {
	if len(doc.Tree.Specs) == 0 {
		return nil, fmt.Errorf("build has no <Spec> element")
	}
	x := activeSpec(doc)
	if x.TreeVersion != tr.Version {
		return nil, fmt.Errorf("build is on tree %q, loaded tree is %q", x.TreeVersion, tr.Version)
	}
	// An explicit version attribute wins; a spec carrying a nodes list
	// without one predates the cluster-hash change.
	version := 2
	if n, err := strconv.Atoi(x.ClusterHashFormatVersion); err == nil {
		version = n
	} else if x.Nodes != "" {
		version = 1
	}
	saved := &tree.SavedSpec{
		ClassID:                  attrInt64(x.ClassID),
		AscendClassID:            attrInt64(x.AscendClassID),
		SecondaryAscendClassID:   attrInt64(x.SecondaryAscendClassID),
		Nodes:                    x.Nodes,
		MasteryEffects:           x.MasteryEffects,
		Sockets:                  map[int64]int{},
		ClusterHashFormatVersion: version,
	}
	for _, o := range x.Overrides.Overrides {
		saved.Overrides = append(saved.Overrides, tree.SavedOverride{
			NodeID: o.NodeID, Dn: o.Dn, Icon: o.Icon, ActiveEffectImage: o.ActiveEffectImage,
		})
	}
	for _, s := range x.Sockets.Sockets {
		saved.Sockets[s.NodeID] = s.ItemID
	}
	spec := tree.NewSpec(tr, items)
	spec.LoadSaved(saved)
	spec.PostLoad()
	return spec, nil
}

func loadItems(doc *Doc) map[int]*item.Item {
	out := map[int]*item.Item{}
	for _, x := range doc.Items.Items {
		saved := &item.SavedItem{
			ID:          x.ID,
			Variant:     attrIntPtr(x.Variant),
			VariantAlt:  attrIntPtr(x.VariantAlt),
			VariantAlt2: attrIntPtr(x.VariantAlt2),
			VariantAlt3: attrIntPtr(x.VariantAlt3),
			VariantAlt4: attrIntPtr(x.VariantAlt4),
			VariantAlt5: attrIntPtr(x.VariantAlt5),
			Raw:         x.Raw,
		}
		for _, mr := range x.ModRanges {
			saved.ModRanges = append(saved.ModRanges, item.SavedModRange{ID: mr.ID, Range: mr.Range})
		}
		if it := item.LoadSaved(saved); it != nil {
			out[x.ID] = it
		}
	}
	return out
}

func itemInputs(items map[int]*item.Item) map[int]*calc.ItemInput {
	out := map[int]*calc.ItemInput{}
	for id, it := range items {
		out[id] = calc.ItemInputOf(it)
	}
	return out
}

// loadItemSets decodes the <ItemSet> elements and reports which one the
// build has active. A legacy build saved its slots outside any set; those
// become the single active set.
func loadItemSets(doc *Doc) (map[int]*calc.ItemSetInput, *itemSet) {
	out := map[int]*calc.ItemSetInput{}
	sets := map[int]*itemSet{}
	for i, x := range doc.Items.ItemSets {
		id := int(attrNum(x.ID, float64(i+1)))
		set := &itemSet{
			id:                 id,
			useSecondWeaponSet: x.UseSecondWeaponSet == "true",
			selItemID:          map[string]int{},
			active:             map[string]bool{},
		}
		for _, s := range x.Slots {
			if n := int(attrNum(s.ItemID, 0)); n != 0 {
				set.selItemID[s.Name] = n
			}
			if s.Active == "true" {
				set.active[s.Name] = true
			}
		}
		sets[id] = set
	}
	if len(sets) == 0 && len(doc.Items.Slots) > 0 {
		set := &itemSet{id: 1, selItemID: map[string]int{}, active: map[string]bool{}}
		for _, s := range doc.Items.Slots {
			if n := int(attrNum(s.ItemID, 0)); n != 0 {
				set.selItemID[s.Name] = n
			}
			if s.Active == "true" {
				set.active[s.Name] = true
			}
		}
		sets[1] = set
	}
	for id, set := range sets {
		in := &calc.ItemSetInput{Slots: map[string]float64{}}
		if set.useSecondWeaponSet {
			in.UseSecondWeaponSet = truePtr()
		}
		for name, itemID := range set.selItemID {
			in.Slots[name] = float64(itemID)
		}
		out[id] = in
	}
	active := sets[int(attrNum(doc.Items.ActiveItemSet, 0))]
	if active == nil {
		// SetActiveItemSet falls back to the first set in save order.
		for _, x := range doc.Items.ItemSets {
			if s := sets[int(attrNum(x.ID, 0))]; s != nil {
				active = s
				break
			}
		}
	}
	if active == nil {
		active = sets[1]
	}
	return out, active
}

func setOrder(doc *Doc) []float64 {
	var out []float64
	for i, x := range doc.Items.ItemSets {
		out = append(out, attrNum(x.ID, float64(i+1)))
	}
	if out == nil && len(doc.Items.Slots) > 0 {
		out = []float64{1}
	}
	return out
}

// mainSocketGroup is Build.lua's `mainSkillIndex or mainSocketGroup or 1`.
func mainSocketGroup(doc *Doc) float64 {
	if n, err := strconv.ParseFloat(strings.TrimSpace(doc.Build.MainSkillIndex), 64); err == nil {
		return n
	}
	return attrNum(doc.Build.MainSocketGroup, 1)
}

// spectreList keeps the saved spectre ids the minion data knows, the way
// Build:Load filters them.
func spectreList(doc *Doc) []string {
	var out []string
	for _, s := range doc.Build.Spectres {
		if s.ID != "" && data.Minions[s.ID] != nil {
			out = append(out, s.ID)
		}
	}
	return out
}

func attrNum(s string, def float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return n
}

func attrInt64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0 // an absent or non-numeric attribute reads as 0
	}
	return n
}

func attrIntPtr(s string) *int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &n
}
