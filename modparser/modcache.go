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
		var arr []any
		if err := json.Unmarshal(entry.Mods, &arr); err != nil {
			panic("modparser: bad cached mods for " + line + ": " + err.Error())
		}
		res.mods = make([]any, len(arr))
		for i, v := range arr {
			res.mods[i] = decodeCachedVal(v)
		}
	}
	if entry.Extra != nil {
		res.extra = *entry.Extra
	}
	return res, true
}

// The cache entry value format: conventional JSON. Mods and D tables are
// discriminated objects ({"_":"mod",...}, {"_":"d",...}); tags are plain
// objects, lists are arrays. EncodeMods and decodeCachedVal are the two
// halves; the generator (internal/modcachegen) writes with EncodeMods.

func encodeCachedVal(v any) any {
	switch t := v.(type) {
	case float64:
		// JSON has no Infinity; math.huge appears in tag limits.
		if math.IsInf(t, 1) {
			return map[string]any{"_": "inf"}
		}
		if math.IsInf(t, -1) {
			return map[string]any{"_": "-inf"}
		}
		return t
	case *Mod:
		m := map[string]any{
			"_":            "mod",
			"name":         t.Name,
			"type":         t.Type,
			"flags":        t.Flags,
			"keywordFlags": t.KeywordFlags,
			"value":        encodeCachedVal(t.Value),
		}
		if t.SourceSet {
			m["source"] = t.Source
		}
		tags := make([]any, len(t.Tags))
		for i, tag := range t.Tags {
			tags[i] = encodeCachedVal(tag)
		}
		m["tags"] = tags
		return m
	case *D:
		arr := make([]any, len(t.Arr))
		for i, e := range t.Arr {
			arr[i] = encodeCachedVal(e)
		}
		kv := make(map[string]any, len(t.KV))
		for k, e := range t.KV {
			kv[k] = encodeCachedVal(e)
		}
		return map[string]any{"_": "d", "arr": arr, "kv": kv}
	case Tag:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = encodeCachedVal(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = encodeCachedVal(e)
		}
		return out
	}
	return v
}

// EncodeMods serialises a parse result's mod list in the cache entry
// format (deterministic: object keys sort under json.Marshal).
func EncodeMods(mods []any) []byte {
	out, err := json.Marshal(encodeCachedVal(any(append([]any{}, mods...))))
	if err != nil {
		panic("modparser: encoding mods: " + err.Error())
	}
	return out
}

func decodeCachedVal(v any) any {
	switch t := v.(type) {
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = decodeCachedVal(e)
		}
		return out
	case map[string]any:
		switch t["_"] {
		case "inf":
			return math.Inf(1)
		case "-inf":
			return math.Inf(-1)
		case "mod":
			mod := &Mod{
				Name:         t["name"].(string),
				Type:         t["type"].(string),
				Flags:        int64(t["flags"].(float64)),
				KeywordFlags: int64(t["keywordFlags"].(float64)),
				Value:        decodeCachedVal(t["value"]),
			}
			if src, ok := t["source"].(string); ok {
				mod.Source = src
				mod.SourceSet = true
			}
			for _, tag := range t["tags"].([]any) {
				mod.Tags = append(mod.Tags, decodeCachedVal(tag))
			}
			return mod
		case "d":
			d := &D{KV: map[string]any{}}
			for _, e := range t["arr"].([]any) {
				d.Arr = append(d.Arr, decodeCachedVal(e))
			}
			for k, e := range t["kv"].(map[string]any) {
				d.KV[k] = decodeCachedVal(e)
			}
			return d
		default:
			out := Tag{}
			for k, e := range t {
				out[k] = decodeCachedVal(e)
			}
			return out
		}
	}
	return v
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
