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
	"github.com/MissingL-tter/missingPassives/internal/luacanon"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
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

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func truePtrOrNil(b bool) *bool {
	if !b {
		return nil
	}
	return &b
}

func f64Ptr(v float64) *float64 { return &v }

// itemInputOf projects a parsed item into the fixture's ItemInput shape,
// mirroring dump_calc.lua's itemFixture().
func itemInputOf(it *item.Item) *calc.ItemInput {
	in := &calc.ItemInput{
		Name:             it.Name,
		ModSource:        strPtrOrNil(it.ModSource),
		Title:            strPtrOrNil(it.Title),
		BaseName:         strPtrOrNil(it.BaseName),
		Type:             it.Type,
		Rarity:           it.Rarity,
		Corrupted:        truePtrOrNil(it.Corrupted),
		Shaper:           truePtrOrNil(it.Influence["shaper"]),
		Elder:            truePtrOrNil(it.Influence["elder"]),
		Adjudicator:      truePtrOrNil(it.Influence["adjudicator"]),
		Basilisk:         truePtrOrNil(it.Influence["basilisk"]),
		Crusader:         truePtrOrNil(it.Influence["crusader"]),
		Eyrie:            truePtrOrNil(it.Influence["eyrie"]),
		Foulborn:         &it.Foulborn,
		ClassRestriction: strPtrOrNil(it.ClassRestriction),
		Limit:            it.Limit,
		Quality:          it.Quality,
	}
	if it.Base != nil {
		base := &calc.ItemBaseInput{Type: strPtrOrNil(it.Base.Type), SubType: strPtrOrNil(it.Base.SubType)}
		if it.Base.Flask != nil {
			fb := &calc.FlaskBaseInput{}
			if it.Base.Flask.Life != nil {
				fb.Life = *it.Base.Flask.Life
			}
			if it.Base.Flask.Mana != nil {
				fb.Mana = *it.Base.Flask.Mana
			}
			base.Flask = fb
		}
		in.Base = base
	}
	if it.ModList != nil {
		in.ModList = it.ModList
	}
	if it.SlotModList != nil {
		in.SlotModList = it.SlotModList
	}
	if it.BaseModList != nil {
		in.BaseModList = it.BaseModList
	}
	if it.BuffModList != nil {
		in.BuffModList = it.BuffModList
	} else if it.BuffModListInit {
		in.BuffModList = []*modparser.Mod{}
	}
	in.GrantedSkills = it.GrantedSkills
	if it.Requirements != nil {
		in.Requirements = it.Requirements
	}
	sockets := make([]map[string]any, 0, len(it.Sockets))
	for _, s := range it.Sockets {
		sockets = append(sockets, map[string]any{"color": s.Color, "group": s.Group})
	}
	in.Sockets = sockets
	in.AbyssalSocketCount = f64Ptr(it.AbyssalSocketCount)
	in.SocketedJewelEffectModifier = f64Ptr(it.SocketedJewelEffectModifier)
	if it.JewelRadiusIndex != nil {
		in.JewelRadiusIndex = f64Ptr(float64(*it.JewelRadiusIndex))
	}
	if it.JewelData != nil {
		if funcList, ok := it.JewelData["funcList"].([]any); ok {
			for _, fv := range funcList {
				tag, _ := fv.(modparser.Tag)
				typ, _ := tag["type"].(string)
				in.FuncTypes = append(in.FuncTypes, typ)
			}
		}
		in.JewelData = scalarsOnly(it.JewelData)
	}
	if it.FlaskData != nil {
		in.FlaskData = scalarsOnly(it.FlaskData)
	}
	if it.TinctureData != nil {
		in.TinctureData = scalarsOnly(it.TinctureData)
	}
	if it.ArmourData != nil {
		in.ArmourData = scalarsOnly(it.ArmourData)
	}
	if it.WeaponData != nil {
		wd := map[int]map[string]any{}
		for i := 1; i <= 2; i++ {
			if side, ok := it.WeaponData[i]; ok {
				wd[i] = scalarsOnly(side)
			}
		}
		in.WeaponData = wd
	}
	expl, other := []string{}, []string{}
	collect := func(lines []*item.ModLine, dst *[]string) {
		for _, v := range lines {
			if !v.Flag("disabled") && it.CheckModLineVariant(v) {
				*dst = append(*dst, v.Line)
			}
		}
	}
	collect(it.ExplicitModLines, &expl)
	collect(it.EnchantModLines, &other)
	collect(it.ScourgeModLines, &other)
	collect(it.ImplicitModLines, &other)
	collect(it.CrucibleModLines, &other)
	in.ExplicitLines = expl
	in.OtherLines = other
	return in
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
			gotCanon := luacanon.EncodeExact(itemInputOf(got[id]))
			if want != gotCanon {
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
