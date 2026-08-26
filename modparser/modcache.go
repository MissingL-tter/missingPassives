// The shipped mod cache. PoB preloads Data/ModCache.lua into parseMod's
// cache (Main.lua L125), so every cached line resolves to the file's
// entry, whose numbers round-tripped through %.14g text when the file was
// written (Main:SaveModCache -> writeLuaTable -> tostring). The parse
// differential (test/parse_test.go, whose corpus IS this key set, compared
// at %.14g) proves the shipped entries match this parser's fresh output,
// so cache semantics reduce to: for exactly these keys, quantize every
// number in the fresh result to %.14g. Lines outside the set — including
// the JewelFunc lines SaveModCache skips — parse fresh at full precision,
// as they do in PoB.
package modparser

import (
	"math"
	"strconv"
)

var modCacheKeys map[string]struct{}

// SetModCacheKeys installs (or, with nil, removes) the shipped cache's key
// set. The parse cache is dropped whenever the policy changes, so entries
// computed under the other policy cannot leak. The only sets in play are
// nil and the one embedded key list, so same-length is taken as same-set.
func SetModCacheKeys(keys []string) {
	parseCacheMu.Lock()
	defer parseCacheMu.Unlock()
	if keys == nil {
		if modCacheKeys == nil {
			return
		}
		modCacheKeys = nil
	} else {
		if modCacheKeys != nil && len(modCacheKeys) == len(keys) {
			return
		}
		set := make(map[string]struct{}, len(keys))
		for _, k := range keys {
			set[k] = struct{}{}
		}
		modCacheKeys = set
	}
	parseCache = map[string]parseResult{}
}

// quant14 is one number's trip through the cache file: tostring (%.14g)
// then tonumber. math.huge survives writeLuaTable verbatim.
func quant14(v float64) float64 {
	if math.IsInf(v, 0) || v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return v
	}
	s := strconv.FormatFloat(v, 'g', 14, 64)
	n, _ := strconv.ParseFloat(s, 64)
	return n
}

// quantizeDeep applies quant14 to every number reachable in a parse
// result. The caller deep-copies the entry first: parseMod's result can
// share tables with the global pattern lists (clusterJewelSkills hands its
// entry back directly), and mutating those would corrupt every later
// parse.
func quantizeDeep(v any) any {
	switch t := v.(type) {
	case float64:
		return quant14(t)
	case *Mod:
		t.Value = quantizeDeep(t.Value)
		for i, tag := range t.Tags {
			t.Tags[i] = quantizeDeep(tag)
		}
		return t
	case Tag:
		for k, e := range t {
			t[k] = quantizeDeep(e)
		}
		return t
	case []any:
		for i, e := range t {
			t[i] = quantizeDeep(e)
		}
		return t
	case *D:
		for i, e := range t.Arr {
			t.Arr[i] = quantizeDeep(e)
		}
		for k, e := range t.KV {
			t.KV[k] = quantizeDeep(e)
		}
		return t
	}
	return v
}
