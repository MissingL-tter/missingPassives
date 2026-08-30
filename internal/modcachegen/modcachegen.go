// Package modcachegen regenerates data/raw/modcache.jsonl — the pre-parsed
// mod-line cache — from the Go parser, replacing the retired Lua dump of
// PoB's shipped Data/ModCache.lua.
//
// It reproduces the reference's generation environment (Main.lua with
// REGENERATE_MOD_CACHE: an empty parse cache, then LoadTree — whose
// ProcessStats parses every tree/tattoo/conquered stat line — then
// loadItemDBs, which parses every unique and rare template, Crafting the
// crafted templates at affix quality 0.5) and then SaveModCache's walk:
// every parsed line sorted by key, minus the JewelFunc/ExtraJewelFunc
// entries, each value %.14g-quantized (the reference's file round-tripped
// its numbers through %.14g text) and serialised in the conventional-JSON
// entry format modparser.EncodeMods defines.
//
// test/modcachegen_test.go proves the output byte-identical to the
// committed artifact. Requires data.Load to have run.
package modcachegen

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/tree"
)

func mustJSON(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

// jewelFuncSkipped mirrors SaveModCache's skip: entries whose first mod is
// the unserializable JewelFunc/ExtraJewelFunc closure.
func jewelFuncSkipped(mods []*modparser.Mod) bool {
	if len(mods) == 0 {
		return false
	}
	m := mods[0]
	return m.Name == "JewelFunc" || m.Name == "ExtraJewelFunc"
}

// Build regenerates the mod cache document for the given tree version
// from the embedded data set.
func Build(treeVersion string) []byte {
	src, err := data.RawSources()
	if err != nil {
		panic(err)
	}
	return BuildFrom(src, treeVersion)
}

// BuildFrom regenerates the mod cache document over src (cmd/sourceupdate
// passes the directory it has just written). It runs data.Load itself —
// the unique walk needs the tree-dependent uniques — so prior Load state
// is irrelevant.
func BuildFrom(src data.Sources, treeVersion string) []byte {
	if err := data.Load(src); err != nil {
		panic(err)
	}
	modparser.SetModCache(nil) // record fresh parses
	defer modparser.SetModCache(data.LoadedModCache)

	if _, err := tree.Load(treeVersion); err != nil {
		panic(err)
	}

	for _, typeList := range data.Uniques {
		for _, raw := range typeList {
			item.New(raw, "UNIQUE", true)
		}
	}
	for _, raw := range data.Rares {
		it := item.New(raw, "RARE", true)
		if it.Base == nil || !it.Crafted {
			continue
		}
		// loadItemDBs auto-adds the base implicits before crafting.
		if it.Base.Implicit != nil && len(it.ImplicitModLines) == 0 {
			i := 0
			for _, line := range strings.Split(*it.Base.Implicit, "\n") {
				if line == "" {
					continue
				}
				ml := &item.ModLine{Line: line}
				if i < len(it.Base.ImplicitModTypes) {
					ml.ModTags = it.Base.ImplicitModTypes[i]
				}
				it.ImplicitModLines = append(it.ImplicitModLines, ml)
				i++
			}
		}
		it.Craft()
	}

	lines := modparser.ParsedLines()
	sort.Strings(lines)
	var buf bytes.Buffer
	for _, line := range lines {
		mods, extra, recognised := modparser.Parse(line)
		if jewelFuncSkipped(mods) {
			continue
		}
		m := "null"
		if recognised {
			m = string(modparser.EncodeMods(modparser.Quantize14(mods)))
		}
		e := "null"
		if extra != "" {
			e = string(mustJSON(extra))
		}
		buf.WriteString(`{"k":` + string(mustJSON(line)) + `,"m":` + m + `,"e":` + e + "}\n")
	}
	return buf.Bytes()
}
