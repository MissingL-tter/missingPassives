package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/build"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// TestBuildLoadAgainstReference is the differential for the build
// assembler: every corpus build loaded from its XML alone, with the
// ItemsTab state it produces - the slot table, the item sets, the header
// scalars - byte-compared against the archive's fixture.
//
// The config tab is not ported, so the assembler leaves configInput,
// configPlaceholder and the two config mod lists unset; those fields are
// not compared here.
func TestBuildLoadAgainstReference(t *testing.T) {
	tr := loadTree329(t)
	manifest := readManifest(t)
	dumpPaths, err := filepath.Glob(filepath.Join("testdata", "build_*.jsonl"))
	if err != nil || len(dumpPaths) == 0 {
		t.Skipf("archive dumps not present")
	}
	sort.Strings(dumpPaths)
	only := os.Getenv("MP_ONLY_BUILD")
	builds, slotsCompared := 0, 0
	for _, path := range dumpPaths {
		buildKey := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "build_"), ".jsonl")
		if only != "" && buildKey != only {
			continue
		}
		xmlRel := manifest[buildKey]
		if xmlRel == "" {
			continue // the empty build has no XML
		}
		xmlPath := filepath.Clean(filepath.Join("..", ".archive", "src", xmlRel))
		fixtures := map[string]string{}
		forEachCalcRecord(t, path, func(k, c string) {
			if strings.HasSuffix(k, ".fixture") {
				fixtures[strings.TrimSuffix(k, ".fixture")] = c
			}
		})
		content, ok := fixtures[buildKey+".full"]
		if !ok {
			if len(fixtures) != 1 {
				t.Fatalf("%s: no %s.full fixture and %d candidates", buildKey, buildKey, len(fixtures))
			}
			for _, c := range fixtures {
				content = c
			}
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(content), &m); err != nil {
			t.Fatalf("%s: %v", buildKey, err)
		}
		ref := decodeCalcFixture(m)
		blob, err := os.ReadFile(xmlPath)
		if err != nil {
			t.Fatal(err)
		}
		got, err := build.Load(blob, tr)
		if err != nil {
			t.Errorf("%s: load: %v", buildKey, err)
			continue
		}
		in := got.Input
		if in.CharacterLevel != ref.CharacterLevel || in.ClassID != ref.ClassID ||
			in.CurClassName != ref.CurClassName || in.TreeVersion != ref.TreeVersion ||
			in.MainSocketGroup != ref.MainSocketGroup {
			t.Errorf("%s: header diverged: level %v/%v class %v/%v %q/%q tree %q/%q mainGroup %v/%v",
				buildKey, in.CharacterLevel, ref.CharacterLevel, in.ClassID, ref.ClassID,
				in.CurClassName, ref.CurClassName, in.TreeVersion, ref.TreeVersion,
				in.MainSocketGroup, ref.MainSocketGroup)
		}
		if in.ClassStats != ref.ClassStats {
			t.Errorf("%s: classStats diverged: %+v vs %+v", buildKey, in.ClassStats, ref.ClassStats)
		}
		if a, b := luacanon.Encode(in.SpectreList), luacanon.Encode(ref.SpectreList); a != b {
			t.Errorf("%s: spectreList diverged: %s vs %s", buildKey, a, b)
		}
		if len(in.ItemsTab.Slots) != len(ref.ItemsTab.Slots) {
			t.Errorf("%s: slot count %d vs reference %d", buildKey, len(in.ItemsTab.Slots), len(ref.ItemsTab.Slots))
			continue
		}
		for i := range ref.ItemsTab.Slots {
			gotCanon := luacanon.EncodeExact(in.ItemsTab.Slots[i])
			want := luacanon.EncodeExact(ref.ItemsTab.Slots[i])
			if !luacanon.SameCanon(gotCanon, want) {
				t.Errorf("%s slot %d (%s) diverged\n%s", buildKey, i+1,
					ref.ItemsTab.Slots[i].SlotName, diffWindow(gotCanon, want))
			}
			slotsCompared++
		}
		if a, b := luacanon.EncodeExact(in.ItemsTab.UseSecondWeaponSet), luacanon.EncodeExact(ref.ItemsTab.UseSecondWeaponSet); !luacanon.SameCanon(a, b) {
			t.Errorf("%s: useSecondWeaponSet diverged: %s vs %s", buildKey, a, b)
		}
		if a, b := luacanon.Encode(in.ItemsTab.ItemSetOrderList), luacanon.Encode(ref.ItemsTab.ItemSetOrderList); a != b {
			t.Errorf("%s: itemSetOrderList diverged: %s vs %s", buildKey, a, b)
		}
		var setIDs []int
		for id := range ref.ItemsTab.ItemSets {
			setIDs = append(setIDs, id)
		}
		sort.Ints(setIDs)
		for _, id := range setIDs {
			gotSet := in.ItemsTab.ItemSets[id]
			if gotSet == nil {
				t.Errorf("%s: item set %d missing", buildKey, id)
				continue
			}
			if a, b := luacanon.EncodeExact(gotSet), luacanon.EncodeExact(ref.ItemsTab.ItemSets[id]); !luacanon.SameCanon(a, b) {
				t.Errorf("%s item set %d diverged\n%s", buildKey, id, diffWindow(a, b))
			}
		}
		for id := range in.ItemsTab.ItemSets {
			if ref.ItemsTab.ItemSets[id] == nil {
				t.Errorf("%s: extra item set %s", buildKey, strconv.Itoa(id))
			}
		}
		builds++
	}
	if only == "" && builds < 40 {
		t.Fatalf("expected the corpus, loaded %d builds", builds)
	}
	t.Logf("build differential: %d slots byte-identical across %d builds", slotsCompared, builds)
}
