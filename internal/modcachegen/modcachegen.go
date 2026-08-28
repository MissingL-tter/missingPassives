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
// entries, each value re-encoded with %.14g quantization (the file the
// reference wrote had round-tripped its numbers through %.14g text).
//
// test/modcachegen_test.go proves the output byte-identical to the
// committed artifact. Requires data.Load to have run.
package modcachegen

import (
	"bytes"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/luacanon"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/tree"
)

// Structured mods canonicalise via their plain-table shadow (the same
// adapter the differential tests register; duplicates are harmless).
func init() {
	luacanon.RegisterAdapter(func(v any) (any, bool) {
		switch t := v.(type) {
		case *modparser.Mod:
			return data.ModCanon(t), true
		case *modparser.D:
			return data.DCanon(t), true
		}
		return nil, false
	})
}

// jewelFuncSkipped mirrors SaveModCache's skip: entries whose first mod is
// the unserializable JewelFunc/ExtraJewelFunc closure.
func jewelFuncSkipped(mods []any) bool {
	if len(mods) == 0 {
		return false
	}
	m, ok := mods[0].(*modparser.Mod)
	return ok && (m.Name == "JewelFunc" || m.Name == "ExtraJewelFunc")
}

// Build regenerates the mod cache document for the given tree version.
func Build(treeVersion string) []byte {
	modparser.SetModCache(nil) // record fresh parses
	defer modparser.SetModCache(data.LoadedModCache)

	tree.Load(treeVersion)

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
		mods, extra := modparser.Parse(line)
		if jewelFuncSkipped(mods) {
			continue
		}
		m := "null"
		if mods != nil {
			m = luacanon.EncodeExact(modparser.Quantize14(mods))
		}
		e := "null"
		if extra != "" {
			e = luacanon.Quote(extra)
		}
		buf.WriteString(`{"k":` + luacanon.Quote(line) + `,"m":` + m + `,"e":` + e + "}\n")
	}
	return buf.Bytes()
}
