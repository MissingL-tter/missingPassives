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
	"errors"
	"fmt"
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
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
//
// It returns an error rather than panicking because the bytes are not
// always the embedded artifact: data.Sources.ModCacheJSONL is a caller-set
// field, and data.RawSourcesFromDir fills it from data/raw on disk — the
// same file cmd/sourceupdate overwrites, so an interrupted regeneration
// leaves a truncated document that the next run reads back. data.Load
// promises an error for malformed input and now keeps that promise here.
// On error the installed cache is left untouched: the index is built whole
// before it is swapped in, so a bad document cannot half-load.
//
// Only the document's record framing is checked here. An entry's mods
// decode lazily on first Parse of its line, and a fault there stays a
// panic — Parse's signature carries no error channel (see DecodeMods).
func SetModCache(jsonl []byte) error {
	parseCacheMu.Lock()
	defer parseCacheMu.Unlock()
	if jsonl == nil {
		if modCache == nil {
			return nil
		}
		modCache = nil
		parseCache = map[string]parseResult{}
		return nil
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
			return fmt.Errorf("modparser: bad modCache.jsonl: %w", err)
		}
		index[rec.K] = modCacheEntry{Mods: rec.M, Extra: rec.E}
	}
	if modCache != nil && len(modCache) == len(index) {
		return nil
	}
	modCache = index
	parseCache = map[string]parseResult{}
	return nil
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
		res.mods = DecodeMods(entry.Mods)
		res.recognised = true
	}
	if entry.Extra != nil {
		res.extra = *entry.Extra
	}
	return res, true
}

// The cache entry value format: conventional JSON with explicit kinds.
// A mod is {"name","type","flags","keywordFlags","value","source"?,"tags"}
// (a null tag is a hole); a tag is {"kind":<reference type>, <params>};
// a scalar value is its JSON scalar, an infinite one {"kind":"inf"|"-inf"},
// a record value {"kind":<record kind>, <params>}. Object keys sort under
// json.Marshal, so the encoding is deterministic.

// EncodeMods serialises a mod list in the cache entry format.
func EncodeMods(mods []*Mod) []byte {
	list := make([]any, len(mods))
	for i, m := range mods {
		list[i] = encodeMod(m)
	}
	out, err := json.Marshal(list)
	if err != nil {
		panic("modparser: encoding mods: " + err.Error())
	}
	return out
}

func encodeMod(m *Mod) map[string]any {
	out := map[string]any{
		"name":         m.Name,
		"type":         m.Type.String(),
		"flags":        uint64(m.Flags),
		"keywordFlags": uint64(m.KeywordFlags),
		"value":        encodeValue(m.Value),
	}
	if m.SourceSet {
		out["source"] = m.Source
	}
	tags := make([]any, len(m.Tags))
	for i, tag := range m.Tags {
		if tag != nil {
			tags[i] = encodeTag(tag)
		}
	}
	out["tags"] = tags
	return out
}

func encodeTag(t Tag) map[string]any {
	out := map[string]any{"kind": TagTypeName(t)}
	for _, p := range t.Params() {
		out[p.Name] = encodeParam(p.Value)
	}
	return out
}

func encodeValue(v Value) any {
	switch t := v.(type) {
	case nil:
		return nil
	case Num:
		return encodeNumber(float64(t))
	case Bool:
		return bool(t)
	case Str:
		return string(t)
	}
	kind, params, ok := ValueParams(v)
	if !ok {
		panic("modparser: unencodable value")
	}
	out := map[string]any{"kind": kind.String()}
	for _, p := range params {
		if inner, isValue := p.Value.(Value); isValue && kind == ValueKindData && p.Name == "value" {
			out[p.Name] = encodeValue(inner) // the nested value, as decodeValue reads it back
			continue
		}
		out[p.Name] = encodeParam(p.Value)
	}
	return out
}

// encodeNumber writes an infinity as a tagged object — JSON has none.
func encodeNumber(f float64) any {
	if math.IsInf(f, 1) {
		return map[string]any{"kind": "inf"}
	}
	if math.IsInf(f, -1) {
		return map[string]any{"kind": "-inf"}
	}
	return f
}

func encodeParam(v ParamValue) any {
	switch t := v.(type) {
	case *Mod:
		return encodeMod(t)
	case Pairs:
		out := make([][2]any, len(t))
		for i, pair := range t {
			out[i] = [2]any{encodeParam(Num(pair[0])), encodeParam(Num(pair[1]))}
		}
		return out
	case Conqueror:
		return map[string]any{"kind": t.Kind.String(), "index": t.Index, "v2": t.V2}
	case JewelNodeFn, JewelFnRef:
		panic("modparser: a jewel function cannot be encoded")
	case Num:
		if math.IsInf(float64(t), 0) {
			return util.FormatG14(float64(t)) // "inf"/"-inf"; numeric fields coerce it back
		}
		return float64(t)
	case NumList:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = encodeParam(Num(e))
		}
		return out
	case Value:
		return encodeValue(t)
	}
	return v
}

// DecodeMods is EncodeMods' inverse. A malformed entry is a build artifact
// fault and panics rather than returning an error: the caller that matters
// is cacheLookup, decoding an installed cache entry lazily inside Parse,
// and Parse's result is (mods, extra, recognised) with no error channel —
// an error return here would only move the panic into cacheLookup. The two
// document callers (data's minion and skill records) are in the same class:
// an artifact the build produced, malformed. Callers that must report
// instead of fault use DecodeValue/DecodeTag.
func DecodeMods(raw []byte) []*Mod {
	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		panic("modparser: bad structured mods: " + err.Error())
	}
	out := make([]*Mod, len(list))
	for i, e := range list {
		out[i] = decodeMod(e)
	}
	return out
}

// DecodeMod decodes one mod object of the cache entry format. It panics on
// a malformed object for the reason DecodeMods gives.
func DecodeMod(raw []byte) *Mod { return decodeMod(raw) }

// DecodeValue and DecodeTag decode one value, or one tag, of the cache
// entry format on its own. They exist for callers outside the embedded
// artifact — data's mixed baseMods records — where a malformed record is
// a record to report on, not the build fault DecodeMods treats it as.
// They wrap the very decoders the artifact path uses, and that path is
// deliberately panicking (a bad artifact must not be half-loaded), so the
// fault is turned into an error here at the boundary rather than threaded
// back through the recursive decoders.
func DecodeValue(raw json.RawMessage) (v Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			v, err = nil, decodeFault(r)
		}
	}()
	return decodeValue(raw), nil
}

func DecodeTag(raw json.RawMessage) (t Tag, err error) {
	defer func() {
		if r := recover(); r != nil {
			t, err = nil, decodeFault(r)
		}
	}()
	return decodeTag(raw), nil
}

// decodeFault turns a recovered decoder panic into an error. The decoders
// panic with their own already-prefixed strings; anything else is a
// runtime fault and gets the prefix here.
func decodeFault(r any) error {
	if s, ok := r.(string); ok {
		return errors.New(s)
	}
	return fmt.Errorf("modparser: decode: %v", r)
}

func decodeObject(raw json.RawMessage) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		panic("modparser: bad structured mods: " + err.Error())
	}
	return m
}

func decodeMod(raw json.RawMessage) *Mod {
	m := decodeObject(raw)
	mod := &Mod{}
	mustUnmarshal(m["name"], &mod.Name)
	var typ string
	mustUnmarshal(m["type"], &typ)
	mod.Type = modTypeOf(typ)
	var flags, kw uint64
	mustUnmarshal(m["flags"], &flags)
	mustUnmarshal(m["keywordFlags"], &kw)
	mod.Flags, mod.KeywordFlags = ModFlag(flags), KeywordFlag(kw)
	mod.Value = decodeValue(m["value"])
	if src, ok := m["source"]; ok {
		mustUnmarshal(src, &mod.Source)
		mod.SourceSet = true
	}
	var tags []json.RawMessage
	mustUnmarshal(m["tags"], &tags)
	for _, t := range tags {
		if string(t) == "null" {
			mod.Tags = append(mod.Tags, nil)
		} else {
			mod.Tags = append(mod.Tags, decodeTag(t))
		}
	}
	return mod
}

func mustUnmarshal(raw json.RawMessage, out any) {
	if raw == nil {
		panic("modparser: bad structured mods: missing field")
	}
	if err := json.Unmarshal(raw, out); err != nil {
		panic("modparser: bad structured mods: " + err.Error())
	}
}

func modTypeOf(name string) ModType {
	t, ok := ModTypeByName[name]
	if !ok {
		panic("modparser: unknown mod type " + name)
	}
	return t
}

func decodeTag(raw json.RawMessage) Tag {
	m := decodeObject(raw)
	var kind string
	mustUnmarshal(m["kind"], &kind)
	delete(m, "kind")
	params := make([]Param, 0, len(m))
	for k, v := range m {
		params = append(params, Param{k, decodeParam(v)})
	}
	t, ok := TagFromParams(kind, params)
	if !ok {
		panic("modparser: bad structured mods: tag " + kind)
	}
	return t
}

func decodeValue(raw json.RawMessage) Value {
	if raw == nil || string(raw) == "null" {
		return nil
	}
	var scalar any
	if raw[0] != '{' {
		mustUnmarshal(raw, &scalar)
		switch t := scalar.(type) {
		case float64:
			return Num(t)
		case bool:
			return Bool(t)
		case string:
			return Str(t)
		}
		panic("modparser: bad structured mods: value")
	}
	m := decodeObject(raw)
	var kindName string
	mustUnmarshal(m["kind"], &kindName)
	switch kindName {
	case "inf":
		return Num(math.Inf(1))
	case "-inf":
		return Num(math.Inf(-1))
	}
	delete(m, "kind")
	kind := ValueKindByName[kindName]
	params := make([]Param, 0, len(m))
	for k, v := range m {
		var pv ParamValue
		switch {
		case k == "mod" || k == "value" && kind == ValueKindPropertyMod:
			pv = decodeMod(v)
		case k == "value" && kind == ValueKindData:
			pv = decodeValue(v)
		case k == "conqueror":
			c := decodeObject(v)
			var cq Conqueror
			var ckind string
			mustUnmarshal(c["kind"], &ckind)
			cq.Kind = ConquerorKindByName[ckind]
			mustUnmarshal(c["index"], &cq.Index)
			mustUnmarshal(c["v2"], &cq.V2)
			pv = cq
		default:
			pv = decodeParam(v)
		}
		params = append(params, Param{k, pv})
	}
	v, ok := valueFromParams(kind, params)
	if !ok {
		panic("modparser: bad structured mods: value " + kindName)
	}
	return v
}

// decodeParam reads a tag/record field as its JSON scalar or list;
// TagFromParams/ValueFromParams coerce it to the field's type.
func decodeParam(raw json.RawMessage) ParamValue {
	var v any
	mustUnmarshal(raw, &v)
	return ParamOf(v)
}

// Quantize14 deep-copies a mod list and rounds every number the way
// writing and reloading Data/ModCache.lua does. Used by the modcache
// differential to prove the shipped entries equal a fresh parse.
func Quantize14(mods []*Mod) []*Mod {
	out := make([]*Mod, len(mods))
	for i, m := range mods {
		out[i] = quantizeMod(m.Clone())
	}
	return out
}

func quantizeMod(m *Mod) *Mod {
	m.Value = quantizeValue(m.Value)
	for i, tag := range m.Tags {
		if tag != nil {
			m.Tags[i] = quantizeTag(tag)
		}
	}
	return m
}

// quantizeTag faults on a tag that will not rebuild rather than returning
// the nil TagFromParams hands back: a nil tag is a hole in Mod.Tags, and a
// hole encodes as null on both sides of the mod-cache differential — the
// shipped file (internal/modcachegen) and the fresh-parse comparison
// (test/modcache_test.go) both run through Quantize14 — so the dropped tag
// would agree with itself and never surface as a disagreement. Same policy
// as quantizeValue below.
func quantizeTag(t Tag) Tag {
	params := t.Params()
	for i := range params {
		params[i].Value = quantizeParam(params[i].Value)
	}
	q, ok := TagFromParams(TagTypeName(t), params)
	if !ok {
		panic("modparser: quantize: " + TagTypeName(t))
	}
	return q
}

func quantizeValue(v Value) Value {
	if n, ok := v.(Num); ok {
		return Num(util.Quantize14(float64(n)))
	}
	kind, params, ok := ValueParams(v)
	if !ok {
		return v
	}
	for i := range params {
		params[i].Value = quantizeParam(params[i].Value)
	}
	q, ok := valueFromParams(kind, params)
	if !ok {
		panic("modparser: quantize: " + kind.String())
	}
	return q
}

func quantizeParam(v ParamValue) ParamValue {
	switch t := v.(type) {
	case Num:
		return Num(util.Quantize14(float64(t)))
	case *Mod:
		return quantizeMod(t)
	case Pairs:
		out := make(Pairs, len(t))
		for i, pair := range t {
			out[i] = [2]float64{util.Quantize14(pair[0]), util.Quantize14(pair[1])}
		}
		return out
	case Value:
		return quantizeValue(t)
	case NumList:
		out := make(NumList, len(t))
		for i, e := range t {
			out[i] = util.Quantize14(e)
		}
		return out
	}
	return v
}
