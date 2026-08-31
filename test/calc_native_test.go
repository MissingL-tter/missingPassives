package test

// The native bridge for the calc differential: instead of feeding calc the
// dumped fixture, assemble the BuildInput from the build XML through
// package build and substitute everything the assembler covers - spec,
// item pool, slot table, item sets and skills tab. The config tab is not
// ported, so its four fields stay fixture-fed. Mods are deep-copied at
// projection: calc stamps sources in place (the reference mutates its
// per-process tables the same way), and the test process shares one
// cached tree across all builds.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/build"
	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/tree"
)

func copyMods(mods []*modparser.Mod) []*modparser.Mod {
	if mods == nil {
		return nil
	}
	out := make([]*modparser.Mod, len(mods))
	for i, m := range mods {
		if m != nil {
			out[i] = m.Clone()
		}
	}
	return out
}

// referenceOrder re-permutes a projected node's mod list into the
// reference's pairs() order. Production merges timeless additions in
// first-seen order; the archive's order is a LuaJIT hash walk, which only
// the differential needs (see referenceOrderModList).
func referenceOrder(in *calc.NodeInput, n *tree.SpecNode) {
	in.ModList = copyMods(referenceOrderModList(n))
}

// nativeSpecInput restores the archive's mod order on a projected spec.
func nativeSpecInput(spec *tree.Spec, in *calc.SpecInput) *calc.SpecInput {
	for id, node := range in.AllocNodes {
		referenceOrder(node, spec.AllocNodes[int64(id)])
	}
	for id, node := range in.RadiusNodeData {
		referenceOrder(node, spec.Nodes[int64(id)])
	}
	return in
}

// applyNativeBuild swaps the fixture's build state for a natively
// assembled one. Loads fresh per call: calc mutates its input in place.
func applyNativeBuild(t *testing.T, buildKey, variant string, in *calc.BuildInput) {
	t.Helper()
	manifest := readManifest(t)
	xmlRel := manifest[buildKey]
	if xmlRel == "" {
		return // the empty build has no XML
	}
	xmlPath := filepath.Clean(filepath.Join("..", ".archive", "src", xmlRel))
	blob, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatal(err)
	}
	native, err := build.Load(blob, loadTree329(t))
	if err != nil {
		t.Fatalf("%s: %v", buildKey, err)
	}
	in.Spec = nativeSpecInput(native.Spec, native.Input.Spec)
	// The dump's reduced variants wiped the item pool or the socket groups
	// in place before recording, so those variants keep the fixture's.
	if len(in.ItemsTab.Items) > 0 { // .treeonly fixtures carry a wiped pool
		got := native.Input.ItemsTab
		if len(got.Items) != len(in.ItemsTab.Items) {
			t.Fatalf("%s: native item pool %d vs fixture %d", buildKey, len(got.Items), len(in.ItemsTab.Items))
		}
		for id := range in.ItemsTab.Items {
			if got.Items[id] == nil {
				t.Fatalf("%s: native pool missing item %d", buildKey, id)
			}
		}
		in.ItemsTab = got
	}
	// The wipe leaves the imbued map stale, so the reduced variants keep
	// the full load's map over an empty group list.
	if in.SkillsTab != nil {
		tab := native.Input.SkillsTab
		if strings.HasSuffix(variant, ".noskills") || strings.HasSuffix(variant, ".treeonly") {
			tab.SocketGroups = nil
		}
		in.SkillsTab = tab
	}
}
