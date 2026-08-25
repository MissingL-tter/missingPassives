package test

import (
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
)

// metaChars are the regex metacharacters that must not appear inside an
// item-tag pattern (^ and $ are stripped as anchors before the scan).
const metaChars = `^$()%.[]*+-?\|{}`

// interiorMeta returns the first metacharacter inside pat, ignoring a
// leading ^ / trailing $, or "" when the pattern is literal text.
func interiorMeta(pat string) string {
	core := strings.TrimSuffix(strings.TrimPrefix(pat, "^"), "$")
	if i := strings.IndexAny(core, metaChars); i >= 0 {
		return core[i : i+1]
	}
	return ""
}

// TestItemTagPatternsAreLiteral guards the assumption calc.patFind relies
// on: the item-tag tables use no regex syntax beyond a leading ^ / trailing
// $. Anything richer needs a dialect decision — the reference matches these
// with Lua string.find, whose metacharacters differ from Go's — so an edit
// that introduces one must fail here rather than silently change matching.
func TestItemTagPatternsAreLiteral(t *testing.T) {
	loadGameData(t)
	checked := 0
	for table, tbl := range map[string]map[string]map[string][]string{
		"itemTagSpecial":                 data.ItemTagSpecial,
		"itemTagSpecialExclusionPattern": data.ItemTagSpecialExclusionPattern,
	} {
		for key, slots := range tbl {
			if strings.IndexAny(key, metaChars) >= 0 {
				t.Errorf("%s: key %q contains regex metacharacters", table, key)
			}
			for slot, pats := range slots {
				if strings.IndexAny(slot, metaChars) >= 0 {
					t.Errorf("%s: slot %q contains regex metacharacters", table, slot)
				}
				for _, pat := range pats {
					checked++
					if m := interiorMeta(pat); m != "" {
						t.Errorf("%s[%q][%q]: %q has interior metacharacter %q; "+
							"calc.patFind assumes literal text plus ^/$ anchors",
							table, key, slot, pat, m)
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no item-tag patterns checked")
	}
	t.Logf("item-tag patterns checked: %d", checked)
}

// TestInteriorMetaDetectsPatterns is the negative control: the scan above
// only means something if it rejects real pattern syntax.
func TestInteriorMetaDetectsPatterns(t *testing.T) {
	for _, pat := range []string{
		"Life %d+ regenerated", "Adds (%d+) to", "Cannot Leech.", "a-z", "x|y",
	} {
		if interiorMeta(pat) == "" {
			t.Errorf("%q accepted as literal text", pat)
		}
	}
	for _, pat := range []string{
		"Zealot's Oath", "^Cannot Leech$", "^Allocates", "chance to Evade",
	} {
		if m := interiorMeta(pat); m != "" {
			t.Errorf("%q rejected over %q, but it is literal text", pat, m)
		}
	}
}
