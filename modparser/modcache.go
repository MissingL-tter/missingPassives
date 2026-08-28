// The shipped mod cache. PoB preloads Data/ModCache.lua into parseMod's
// cache (Main.lua L125), so for its 13,173 lines the parser serves the
// file's pre-parsed entries — whose numbers round-tripped through %.14g
// text when PoB wrote the file — and never parses fresh. This port does
// the same: data/raw/modcache.jsonl (internal/modcachegen) carries the
// file's entries, SetModCache installs them, and Parse serves a cached
// line by decoding its entry instead of parsing (~450µs parse vs ~µs
// decode). Lines outside the file — including the JewelFunc lines
// Main:SaveModCache skips — parse fresh at full precision, as in PoB.
//
// test/modcache_test.go proves both halves: every decoded entry re-encodes
// to the file's exact bytes, and equals a fresh parse with every number
// %.14g-rounded — so the shipped file holds no stale entries and serving
// it changes nothing but speed and the 15th digit.
package modparser

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
)

type modCacheEntry struct {
	Mods  json.RawMessage `json:"m"`
	Extra *string         `json:"e"`
}

// modCache maps a mod line to its undecoded shipped entry; entries decode
// lazily on first Parse of the line.
var modCache map[string]modCacheEntry

// SetModCache installs (or, with nil, removes) the shipped cache. The
// parse cache is dropped whenever the policy changes so entries computed
// under the other policy cannot leak; the only inputs in play are nil and
// the one embedded document, so same entry count means same document.
func SetModCache(jsonl []byte) {
	parseCacheMu.Lock()
	defer parseCacheMu.Unlock()
	if jsonl == nil {
		if modCache == nil {
			return
		}
		modCache = nil
		parseCache = map[string]parseResult{}
		return
	}
	index := make(map[string]modCacheEntry, 16384)
	dec := json.NewDecoder(bytes.NewReader(jsonl))
	for dec.More() {
		var rec struct {
			K string          `json:"k"`
			M json.RawMessage `json:"m"`
			E *string         `json:"e"`
		}
		if err := dec.Decode(&rec); err != nil {
			panic("modparser: bad modCache.jsonl: " + err.Error())
		}
		index[rec.K] = modCacheEntry{Mods: rec.M, Extra: rec.E}
	}
	if modCache != nil && len(modCache) == len(index) {
		return
	}
	modCache = index
	parseCache = map[string]parseResult{}
}

// ParsedLines returns every line Parse has answered since the parse cache
// was last reset (SetModCache resets it). With the shipped cache removed
// (SetModCache(nil)) this is the record of what a walk parsed — the mod
// cache generator's key source.
func ParsedLines() []string {
	parseCacheMu.Lock()
	defer parseCacheMu.Unlock()
	lines := make([]string, 0, len(parseCache))
	for line := range parseCache {
		lines = append(lines, line)
	}
	return lines
}

// cacheLookup decodes the shipped entry for line, if one exists. Called
// with parseCacheMu held.
func cacheLookup(line string) (parseResult, bool) {
	entry, ok := modCache[line]
	if !ok {
		return parseResult{}, false
	}
	var res parseResult
	if string(entry.Mods) != "null" {
		var m map[string]any
		if err := json.Unmarshal(entry.Mods, &m); err != nil {
			panic("modparser: bad cached mods for " + line + ": " + err.Error())
		}
		res.mods = make([]any, len(m))
		for i := 1; i <= len(m); i++ {
			mv, ok := m[strconv.Itoa(i)]
			if !ok {
				panic("modparser: cached mod list with a hole for " + line)
			}
			res.mods[i-1] = decodeCachedVal(mv)
		}
	}
	if entry.Extra != nil {
		res.extra = *entry.Extra
	}
	return res, true
}

// decodeCachedVal maps one canon-JSON value back to the parse-result
// shape: mods by their name/type/flags signature, numeric-keyed objects to
// arrays, other objects to tags.
func decodeCachedVal(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	if _, hasName := m["name"]; hasName {
		if _, hasType := m["type"]; hasType {
			if _, hasFlags := m["flags"]; hasFlags {
				return decodeCachedMod(m)
			}
		}
	}
	numeric := len(m) > 0
	for i := 1; i <= len(m); i++ {
		if _, ok := m[strconv.Itoa(i)]; !ok {
			numeric = false
			break
		}
	}
	if numeric {
		out := make([]any, len(m))
		for i := 1; i <= len(m); i++ {
			out[i-1] = decodeCachedVal(m[strconv.Itoa(i)])
		}
		return out
	}
	out := Tag{}
	for k, e := range m {
		out[k] = decodeCachedVal(e)
	}
	return out
}

func decodeCachedMod(m map[string]any) *Mod {
	mod := &Mod{
		Name:         m["name"].(string),
		Type:         m["type"].(string),
		Flags:        int64(m["flags"].(float64)),
		KeywordFlags: int64(m["keywordFlags"].(float64)),
		Value:        decodeCachedVal(m["value"]),
	}
	if src, ok := m["source"].(string); ok {
		mod.Source = src
		mod.SourceSet = true
	}
	// Tag arrays can have holes (a second tag with no first); keep them.
	maxIdx := 0
	for k := range m {
		if i, err := strconv.Atoi(k); err == nil && i > maxIdx {
			maxIdx = i
		}
	}
	for i := 1; i <= maxIdx; i++ {
		if tv, ok := m[strconv.Itoa(i)]; ok {
			mod.Tags = append(mod.Tags, decodeCachedVal(tv))
		} else {
			mod.Tags = append(mod.Tags, nil)
		}
	}
	for k := range m {
		switch k {
		case "name", "type", "flags", "keywordFlags", "value", "source":
		default:
			if _, err := strconv.Atoi(k); err != nil {
				panic("modparser: unexpected cached mod field " + k)
			}
		}
	}
	return mod
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

// Quantize14 deep-copies a parse result and rounds every number the way
// writing and reloading Data/ModCache.lua does. Used by the modcache
// differential to prove the shipped entries equal a fresh parse.
func Quantize14(mods []any) []any {
	out := copyAnyList(mods)
	for i, m := range out {
		out[i] = quantizeDeep(m)
	}
	return out
}

// quantizeDeep applies quant14 to every number reachable in a parse
// result. The caller deep-copies first: parseMod's result can share
// tables with the global pattern lists.
func quantizeDeep(v any) any {
	switch t := v.(type) {
	case float64:
		return quant14(t)
	case int:
		return int(quant14(float64(t)))
	case int64:
		return int64(quant14(float64(t)))
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
