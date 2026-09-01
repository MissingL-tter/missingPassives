package test

// The item-model differential: parse every corpus build's <Item> nodes
// natively (item.LoadSaved over the raw text) and byte-compare each parsed
// item's fixture projection against the archive dump's itemsTab.items
// (tools/dump_calc.lua itemFixture). The dump is pure reference here: the
// input is the build XML itself.

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

type xmlSavedModRange struct {
	ID    int     `xml:"id,attr"`
	Range float64 `xml:"range,attr"`
}

type xmlSavedItem struct {
	ID          int                `xml:"id,attr"`
	Variant     string             `xml:"variant,attr"`
	VariantAlt  string             `xml:"variantAlt,attr"`
	VariantAlt2 string             `xml:"variantAlt2,attr"`
	VariantAlt3 string             `xml:"variantAlt3,attr"`
	VariantAlt4 string             `xml:"variantAlt4,attr"`
	VariantAlt5 string             `xml:"variantAlt5,attr"`
	Raw         string             `xml:",chardata"`
	ModRanges   []xmlSavedModRange `xml:"ModRange"`
}

type xmlBuildItems struct {
	Items struct {
		Items []xmlSavedItem `xml:"Item"`
	} `xml:"Items"`
}

func attrInt(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

// loadCorpusItems parses the build XML's <Item> nodes through the ported
// loader, keyed by item id (base-less items dropped, as in ItemsTab:Load).
func loadCorpusItems(t *testing.T, xmlPath string) map[int]*item.Item {
	t.Helper()
	blob, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatalf("read %s: %v", xmlPath, err)
	}
	var doc xmlBuildItems
	if err := xml.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("decode %s: %v", xmlPath, err)
	}
	out := map[int]*item.Item{}
	for _, x := range doc.Items.Items {
		saved := &item.SavedItem{
			ID:          x.ID,
			Variant:     attrInt(x.Variant),
			VariantAlt:  attrInt(x.VariantAlt),
			VariantAlt2: attrInt(x.VariantAlt2),
			VariantAlt3: attrInt(x.VariantAlt3),
			VariantAlt4: attrInt(x.VariantAlt4),
			VariantAlt5: attrInt(x.VariantAlt5),
			Raw:         x.Raw,
			ModRanges:   nil,
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

// TestItemParseAgainstReference is the item-model differential.
func TestItemParseAgainstReference(t *testing.T) {
	loadData(t)
	manifest := readManifest(t)
	only := os.Getenv("MP_ONLY_ITEM")
	compared, builds := 0, 0
	dumpPaths, err := filepath.Glob(filepath.Join("testdata", "calc_*.jsonl"))
	if err != nil || len(dumpPaths) == 0 {
		t.Skipf("archive dumps not present")
	}
	sort.Strings(dumpPaths)
	for _, path := range dumpPaths {
		name := filepath.Base(path)
		buildKey := strings.TrimSuffix(strings.TrimPrefix(name, "calc_"), ".jsonl")
		if only != "" && buildKey != only {
			continue
		}
		xmlRel, ok := manifest[buildKey]
		if !ok || xmlRel == "" {
			continue // the empty build has no XML and no items
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
			// single-variant dumps (authored shells) use the sole fixture
			if len(fixtures) != 1 {
				t.Fatalf("%s: no %s.full fixture and %d candidates", name, buildKey, len(fixtures))
			}
			for _, c := range fixtures {
				content = c
			}
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(content), &m); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ref := decodeCalcFixture(m)
		if ref.ItemsTab == nil {
			t.Fatalf("%s: fixture has no itemsTab", name)
		}
		got := loadCorpusItems(t, xmlPath)
		var refIDs, gotIDs []int
		for id := range ref.ItemsTab.Items {
			refIDs = append(refIDs, id)
		}
		for id := range got {
			gotIDs = append(gotIDs, id)
		}
		sort.Ints(refIDs)
		sort.Ints(gotIDs)
		if len(refIDs) != len(gotIDs) {
			t.Errorf("%s: item id sets differ: ref %v vs got %v", buildKey, refIDs, gotIDs)
			continue
		}
		for i := range refIDs {
			if refIDs[i] != gotIDs[i] {
				t.Fatalf("%s: item id sets differ: ref %v vs got %v", buildKey, refIDs, gotIDs)
			}
		}
		for _, id := range refIDs {
			want := luacanon.EncodeExact(ref.ItemsTab.Items[id])
			gotCanon := luacanon.EncodeExact(calc.ItemInputOf(got[id]))
			if !luacanon.SameCanon(gotCanon, want) {
				t.Errorf("%s item %d (%s): parse diverged\n%s", buildKey, id, got[id].Name, diffWindow(gotCanon, want))
			}
			compared++
		}
		builds++
	}
	if only == "" && compared < 100 {
		t.Fatalf("expected hundreds of items, compared %d", compared)
	}
	t.Logf("item differential: %d items byte-identical across %d builds", compared, builds)
}

// readManifest parses test/corpus/manifest.tsv (dump key -> XML path
// relative to .archive/src).
func readManifest(t *testing.T) map[string]string {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("corpus", "manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(blob), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			out[parts[0]] = strings.TrimSpace(parts[1])
		} else {
			out[parts[0]] = ""
		}
	}
	return out
}
