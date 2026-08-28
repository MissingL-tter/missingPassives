package test

// The native bridge for the calc differential: instead of feeding calc the
// fixture's spec and item pool, build them from the natively parsed build
// (item package + tree.Spec) and project into the BuildInput shapes. Slots,
// skills tab and config stay fixture-fed until their modules' stages. Mods
// are deep-copied at projection: calc stamps sources in place (the
// reference mutates its per-process tables the same way), and the test
// process shares one cached tree across all builds.

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/skills"
	"github.com/MissingL-tter/missingPassives/tree"
)

func copyMods(mods []*modparser.Mod) []*modparser.Mod {
	if mods == nil {
		return nil
	}
	out := make([]*modparser.Mod, len(mods))
	for i, m := range mods {
		if m != nil {
			out[i] = modparser.CopyMod(m)
		}
	}
	return out
}

// nodeInputOf mirrors specNodeFixtureOf into the calc input shape,
// including the reference-order permutation for might/legacy additions.
func nodeInputOf(n *tree.SpecNode) *calc.NodeInput {
	in := &calc.NodeInput{
		ID:                   float64(n.ID()),
		Type:                 n.Type(),
		Name:                 n.EffectiveName(),
		DN:                   strPtr(n.Dn),
		DistanceToClassStart: n.DistanceToClassStart,
		ModList:              copyMods(referenceOrderModList(n)),
	}
	if n.KeystoneMod != nil {
		in.KeystoneMod = modparser.CopyMod(n.KeystoneMod)
	}
	if n.IsTattoo {
		in.IsTattoo = truePtr(true)
	}
	if n.OverrideType != "" {
		ot := n.OverrideType
		in.OverrideType = &ot
	}
	if n.ConqueredBy != nil {
		in.ConqueredBy = truePtr(true)
	}
	return in
}

// nativeSpecInput projects a native spec (and the socketed radius jewels'
// node data) into the fixture's SpecInput shape.
func nativeSpecInput(spec *tree.Spec, items map[int]*item.Item) *calc.SpecInput {
	out := &calc.SpecInput{
		AllocNodes:                map[int]*calc.NodeInput{},
		KeystoneMap:               map[string][]*modparser.Mod{},
		RadiusNodeData:            map[int]*calc.NodeInput{},
		AllocatedNotableCount:     spec.AllocatedNotableCount,
		AllocatedKeystoneCount:    spec.AllocatedKeystoneCount,
		AllocatedMasteryCount:     spec.AllocatedMasteryCount,
		AllocatedMasteryTypeCount: spec.AllocatedMasteryTypeCount,
		AllocatedMasteryTypes:     map[string]float64{},
		AllocatedTattooTypes:      map[string]float64{},
	}
	for id, node := range spec.AllocNodes {
		out.AllocNodes[int(id)] = nodeInputOf(node)
	}
	for name, ksNode := range spec.Tree.KeystoneMap {
		out.KeystoneMap[name] = copyMods(ksNode.ModList)
	}
	for socketID, itemID := range spec.Jewels {
		it := items[itemID]
		if it == nil || it.JewelRadiusIndex == nil {
			continue
		}
		specSocket := spec.Nodes[socketID]
		if specSocket == nil || specSocket.T.NodesInRadius == nil {
			continue
		}
		for id := range specSocket.T.NodesInRadius[*it.JewelRadiusIndex-1] {
			if spec.AllocNodes[id] == nil && spec.Nodes[id] != nil {
				out.RadiusNodeData[int(id)] = nodeInputOf(spec.Nodes[id])
			}
		}
	}
	for k, v := range spec.AllocatedMasteryTypes {
		out.AllocatedMasteryTypes[k] = v
	}
	for k, v := range spec.AllocatedTattooTypes {
		out.AllocatedTattooTypes[k] = v
	}
	return out
}

// applyNativeBuild swaps the fixture's spec and item pool for natively
// built ones. Loads fresh per call: calc mutates its input in place.
func applyNativeBuild(t *testing.T, buildKey, variant string, in *calc.BuildInput) {
	t.Helper()
	manifest := readManifest(t)
	xmlRel := manifest[buildKey]
	if xmlRel == "" {
		return // the empty build has no XML
	}
	xmlPath := filepath.Clean(filepath.Join("..", ".archive", "src", xmlRel))
	items := loadCorpusItems(t, xmlPath)
	spec := loadCorpusSpec(t, xmlPath, items)
	in.Spec = nativeSpecInput(spec, items)
	if len(in.ItemsTab.Items) > 0 { // .treeonly fixtures carry a wiped pool
		native := map[int]*calc.ItemInput{}
		for id, it := range items {
			native[id] = itemInputOf(it)
		}
		if len(native) != len(in.ItemsTab.Items) {
			t.Fatalf("%s: native item pool %d vs fixture %d", buildKey, len(native), len(in.ItemsTab.Items))
		}
		for id := range in.ItemsTab.Items {
			if native[id] == nil {
				t.Fatalf("%s: native pool missing item %d", buildKey, id)
			}
		}
		in.ItemsTab.Items = native
	}
	// Native skills tab. The dump's reduced variants wiped the XML socket
	// groups in place — the calc's granted-skill update then recreated the
	// item/tree-granted ones — and the wipe leaves the imbued map stale, so
	// the reduced variants keep the full load's map over an empty list.
	if in.SkillsTab != nil {
		blob, err := os.ReadFile(xmlPath)
		if err != nil {
			t.Fatal(err)
		}
		var doc xmlSkillsDoc
		if err := xml.Unmarshal(blob, &doc); err != nil {
			t.Fatal(err)
		}
		tab := skills.Load(&doc.Skills, in.CharacterLevel)
		slotSel := map[string]*item.Item{}
		for _, slot := range in.ItemsTab.Slots {
			if slot.ItemID != nil {
				slotSel[slot.SlotName] = items[int(*slot.ItemID)]
			}
		}
		tab.UpdateSocketGroups(func(slotName string) *item.Item { return slotSel[slotName] })
		native := &calc.SkillsTabInput{ImbuedSupportBySlot: map[string]string{}}
		for _, group := range tab.SocketGroupList {
			g := &calc.SocketGroupInput{KV: group.KV}
			for _, gem := range group.GemList {
				g.GemList = append(g.GemList, &calc.SocketGemInput{
					KV:              gem.KV,
					GemDataID:       gemDataID(gem),
					GrantedEffectID: grantedEffectID(gem),
				})
			}
			native.SocketGroups = append(native.SocketGroups, g)
		}
		for slot, ge := range tab.ImbuedSupportBySlot {
			native.ImbuedSupportBySlot[slot] = ge.Id
		}
		if strings.HasSuffix(variant, ".noskills") || strings.HasSuffix(variant, ".treeonly") {
			native.SocketGroups = nil
		}
		in.SkillsTab = native
	}
}
