package test

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/build"
	"github.com/MissingL-tter/missingPassives/config"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// TestConfigStateAgainstReference is the config tab's load-side
// differential: every corpus build's <Config> element loaded natively,
// with the resulting input and placeholder tables compared key for key
// against the ones the archive dumped.
//
// This checks the option table's data half - every variable's name, type,
// list values and defaults - plus the load path's two stored-value
// migrations, independently of the apply functions.
func TestConfigStateAgainstReference(t *testing.T) {
	loadData(t)
	manifest := readManifest(t)
	dumpPaths, err := filepath.Glob(filepath.Join("testdata", "calc_*.jsonl"))
	if err != nil || len(dumpPaths) == 0 {
		t.Skipf("archive dumps not present")
	}
	sort.Strings(dumpPaths)
	only := os.Getenv("MP_ONLY_CONFIG")
	builds, compared := 0, 0
	for _, path := range dumpPaths {
		buildKey := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "calc_"), ".jsonl")
		if only != "" && buildKey != only {
			continue
		}
		xmlRel := manifest[buildKey]
		if xmlRel == "" {
			continue // the empty build has no XML
		}
		fixture := ""
		forEachCalcRecord(t, path, func(k, c string) {
			if strings.HasSuffix(k, ".full.fixture") || (fixture == "" && strings.HasSuffix(k, ".fixture")) {
				fixture = c
			}
		})
		var m map[string]any
		if err := json.Unmarshal([]byte(fixture), &m); err != nil {
			t.Fatalf("%s: %v", buildKey, err)
		}
		blob, err := os.ReadFile(filepath.Clean(filepath.Join("..", ".archive", "src", xmlRel)))
		if err != nil {
			t.Fatal(err)
		}
		var doc build.Doc
		if err := xml.Unmarshal(blob, &doc); err != nil {
			t.Fatalf("%s: %v", buildKey, err)
		}
		level, _ := m["characterLevel"].(float64)
		tab := config.Load(&doc.Config, level)
		// The dumped tables are post-BuildModList: the enemy and boss
		// options write placeholders as they apply.
		tab.BuildModList()
		compared += compareConfigTable(t, buildKey, "input", tab.Input, m["configInput"])
		compared += compareConfigTable(t, buildKey, "placeholder", tab.Placeholder, m["configPlaceholder"])
		builds++
	}
	if only == "" && builds < 40 {
		t.Fatalf("expected the corpus, loaded %d builds", builds)
	}
	t.Logf("config state differential: %d values agree across %d builds", compared, builds)
}

// compareConfigTable diffs one loaded table against the dumped one, and
// reports how many values agreed.
func compareConfigTable(t *testing.T, buildKey, which string, got map[config.Var]config.Value, refAny any) int {
	t.Helper()
	ref, _ := refAny.(map[string]any)
	agreed := 0
	var names []string
	seen := map[string]bool{}
	for name := range ref {
		names = append(names, name)
		seen[name] = true
	}
	for name := range got {
		if !seen[string(name)] {
			names = append(names, string(name))
		}
	}
	sort.Strings(names)
	for _, name := range names {
		gotVal, has := got[config.Var(name)]
		refVal, inRef := ref[name]
		switch {
		case !inRef && has:
			t.Errorf("%s %s[%s]: %v, absent from the archive", buildKey, which, name, gotVal)
		case inRef && !has:
			t.Errorf("%s %s[%s]: absent, archive has %v", buildKey, which, name, refVal)
		case !configValueEqual(gotVal, refVal):
			t.Errorf("%s %s[%s]: %#v vs archive %#v", buildKey, which, name, gotVal, refVal)
		default:
			agreed++
		}
	}
	return agreed
}

func configValueEqual(got config.Value, ref any) bool {
	switch v := got.(type) {
	case config.Bool:
		b, ok := ref.(bool)
		return ok && bool(v) == b
	case config.Num:
		n, ok := ref.(float64)
		return ok && float64(v) == n
	case config.Str:
		s, ok := ref.(string)
		return ok && string(v) == s
	}
	return false
}

// TestConfigModListCoverage reports how much of the archive's config
// modifier output the ported apply functions reproduce. It is the
// worklist for the rest of the option table: the apply bodies land in
// batches, and this names what is still missing.
func TestConfigModListCoverage(t *testing.T) {
	loadData(t)
	manifest := readManifest(t)
	dumpPaths, err := filepath.Glob(filepath.Join("testdata", "calc_*.jsonl"))
	if err != nil || len(dumpPaths) == 0 {
		t.Skipf("archive dumps not present")
	}
	sort.Strings(dumpPaths)
	missing := map[string]int{}
	extra := map[string]int{}
	var want, got int
	for _, path := range dumpPaths {
		buildKey := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "calc_"), ".jsonl")
		xmlRel := manifest[buildKey]
		if xmlRel == "" {
			continue
		}
		fixture := ""
		forEachCalcRecord(t, path, func(k, c string) {
			if strings.HasSuffix(k, ".full.fixture") || (fixture == "" && strings.HasSuffix(k, ".fixture")) {
				fixture = c
			}
		})
		var m map[string]any
		if err := json.Unmarshal([]byte(fixture), &m); err != nil {
			t.Fatalf("%s: %v", buildKey, err)
		}
		blob, err := os.ReadFile(filepath.Clean(filepath.Join("..", ".archive", "src", xmlRel)))
		if err != nil {
			t.Fatal(err)
		}
		var doc build.Doc
		if err := xml.Unmarshal(blob, &doc); err != nil {
			t.Fatalf("%s: %v", buildKey, err)
		}
		level, _ := m["characterLevel"].(float64)
		tab := config.Load(&doc.Config, level)
		tab.BuildModList()
		for _, pair := range []struct {
			refKey string
			mods   []*modparser.Mod
		}{
			{"configModList", tab.Mods.Mods},
			{"configEnemyModList", tab.EnemyMods.Mods},
		} {
			refMods := decodeCalcModList(m[pair.refKey])
			want += len(refMods)
			got += len(pair.mods)
			have := map[string]int{}
			for _, mod := range pair.mods {
				have[luacanon.EncodeExact(mod)]++
			}
			for _, mod := range refMods {
				canon := luacanon.EncodeExact(mod)
				if have[canon] > 0 {
					have[canon]--
					continue
				}
				missing[mod.Name]++
			}
			for canon, n := range have {
				if n > 0 {
					extra[canon] += n
				}
			}
		}
	}
	var names []string
	for name := range missing {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if missing[names[i]] != missing[names[j]] {
			return missing[names[i]] > missing[names[j]]
		}
		return names[i] < names[j]
	})
	t.Logf("config mods: %d of %d reproduced; %d distinct names missing", got, want, len(names))
	for _, name := range names {
		t.Logf("  missing x%-3d %s", missing[name], name)
	}
	for canon, n := range extra {
		t.Errorf("produced x%d a modifier the archive has none of: %s", n, canon)
	}
}
