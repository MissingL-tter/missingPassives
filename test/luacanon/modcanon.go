package luacanon

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// CanonMods serialises a modparser value (mods, tags, values, pattern-table
// entries) the same way tools/canon.lua serialises the reference's, so the
// two can be compared as strings. Every table becomes a JSON object with
// sorted keys; array indices are stringified ("1", "2", …); whole numbers
// print without a decimal point; functions print as {"__fn":true} since
// behaviour cannot be compared bytewise.
func CanonMods(v any) string {
	var sb strings.Builder
	writeCanon(&sb, Shadow(v))
	return sb.String()
}

// Shadow converts a typed modparser value into the plain-table shape the
// archive canon uses (maps with the reference's keys, []any arrays,
// scalars); anything else passes through.
func Shadow(v any) any {
	switch t := v.(type) {
	case *modparser.Mod:
		if t == nil {
			return nil
		}
		return ModTable(t)
	case []*modparser.Mod:
		if t == nil {
			return nil
		}
		out := make([]any, len(t))
		for i, m := range t {
			out[i] = Shadow(m)
		}
		return out
	case modparser.Tag:
		return TagTable(t)
	case []modparser.Tag:
		out := make([]any, len(t))
		for i, tag := range t {
			out[i] = Shadow(tag)
		}
		return out
	case []modparser.Value:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = ValueTable(e)
		}
		return out
	case modparser.Value:
		return ValueTable(t)
	case modparser.Param:
		return paramValue(t.Value)
	case *modparser.PatternEntry:
		return EntryTable(t)
	case modparser.FlagTypeMod:
		return map[string]any{"name": t.Name, "type": t.Type.String(), "value": ValueTable(t.Value)}
	case modparser.TableFn:
		return Fn{}
	case *modparser.FormattedSourceMod:
		return formattedSourceModTable(t)
	case []string:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = e
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = Shadow(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = Shadow(e)
		}
		return out
	}
	return v
}

// ModTable is a mod's Lua table shape: numbered tags (nil = a hole) plus
// the named fields; source omitted when unset, as a nil key vanishes.
func ModTable(m *modparser.Mod) map[string]any {
	out := map[string]any{
		"name":         m.Name,
		"type":         m.Type.String(),
		"flags":        uint64(m.Flags),
		"keywordFlags": uint64(m.KeywordFlags),
	}
	if m.Value != nil {
		out["value"] = ValueTable(m.Value)
	}
	if m.SourceSet {
		out["source"] = m.Source
	}
	if m.SourceSlot != "" {
		out["sourceSlot"] = m.SourceSlot
	}
	// ReplaceMod/ConvertMod stamp these on the mod table, so the archive
	// canon carries them; without them the calc differential is blind to
	// that bookkeeping.
	if m.Replaced {
		out["replaced"] = true
	}
	if m.Converted {
		out["converted"] = true
	}
	for i, tag := range m.Tags {
		if tag != nil {
			out[strconv.Itoa(i+1)] = TagTable(tag)
		}
	}
	return out
}

func formattedSourceModTable(m *modparser.FormattedSourceMod) map[string]any {
	out := map[string]any{
		"name":         m.Name,
		"type":         m.Type,
		"flags":        uint64(m.Flags),
		"keywordFlags": uint64(m.KeywordFlags),
		"source":       m.Source,
	}
	if m.Value != nil {
		out["value"] = ValueTable(m.Value)
	}
	for i, tag := range m.Tags {
		if tag != nil {
			out[strconv.Itoa(i+1)] = TagTable(tag)
		}
	}
	return out
}

// TagTable is a tag's Lua table shape: its type plus set params.
func TagTable(t modparser.Tag) map[string]any {
	out := map[string]any{}
	if typ := modparser.TagTypeName(t); typ != "" {
		out["type"] = typ
	}
	for _, p := range t.Params() {
		out[p.Name] = paramValue(p.Value)
	}
	return out
}

// ValueTable is a value's Lua shape: scalars as themselves, records as
// tables under their reference keys.
func ValueTable(v modparser.Value) any {
	switch t := v.(type) {
	case nil:
		return nil
	case modparser.Num:
		return float64(t)
	case modparser.Bool:
		return bool(t)
	case modparser.Str:
		return string(t)
	case modparser.Pairs:
		return pairsTable(t)
	}
	_, params, ok := modparser.ValueParams(v)
	if !ok {
		return "?unsupported?"
	}
	out := map[string]any{}
	for _, p := range params {
		out[p.Name] = paramValue(p.Value)
	}
	return out
}

// paramValue renders one tag/record param.
func paramValue(v modparser.ParamValue) any {
	switch t := v.(type) {
	case *modparser.Mod:
		return ModTable(t)
	case modparser.Pairs:
		return pairsTable(t)
	case modparser.Value:
		return ValueTable(t)
	case modparser.Conqueror:
		id := any(float64(t.Index))
		if t.V2 {
			id = t.IDText()
		}
		return map[string]any{"type": t.Kind.String(), "id": id}
	case modparser.JewelFnRef:
		return Fn{}
	case modparser.StrList:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out
	case modparser.NumList:
		out := make([]any, len(t))
		for i, n := range t {
			out[i] = n
		}
		return out
	case modparser.SkillTypeList:
		// A zero id is the reference's nil hole in the list.
		out := make([]any, len(t))
		for i, n := range t {
			if n != 0 {
				out[i] = float64(n)
			}
		}
		return out
	case modparser.SkillTypeID:
		return float64(t)
	case modparser.ModFlag:
		return uint64(t)
	case modparser.KeywordFlag:
		return uint64(t)
	}
	return v
}

func pairsTable(t modparser.Pairs) []any {
	out := make([]any, len(t))
	for i, pair := range t {
		out[i] = []any{pair[0], pair[1]}
	}
	return out
}

// EntryTable is a pattern-table entry's Lua shape: names under "1".., the
// control keys by name.
func EntryTable(e *modparser.PatternEntry) map[string]any {
	out := map[string]any{}
	for i, n := range e.Names {
		out[strconv.Itoa(i+1)] = n
	}
	for i, tags := range e.PerModTags {
		out[strconv.Itoa(i+1)] = map[string]any{"tag": Shadow(tags)}
	}
	if e.Tag != nil {
		out["tag"] = TagTable(e.Tag)
	}
	if e.TagList != nil {
		out["tagList"] = Shadow(e.TagList)
	}
	if e.Flags != 0 {
		out["flags"] = uint64(e.Flags)
	}
	if e.KeywordFlags != 0 {
		out["keywordFlags"] = uint64(e.KeywordFlags)
	}
	flags := map[string]bool{"addToMinion": e.AddToMinion, "addToAura": e.AddToAura, "onlyAddToBanners": e.OnlyAddToBanners,
		"newAura": e.NewAura, "newAuraOnlyAllies": e.NewAuraOnlyAllies, "applyToEnemy": e.ApplyToEnemy, "actorEnemy": e.ActorEnemy}
	for k, v := range flags {
		if v {
			out[k] = true
		}
	}
	if e.AddToMinionTag != nil {
		out["addToMinionTag"] = TagTable(e.AddToMinionTag)
	}
	if e.AddToSkill != nil {
		out["addToSkill"] = TagTable(e.AddToSkill)
	}
	if e.PlayerTag != nil {
		out["playerTag"] = TagTable(e.PlayerTag)
	}
	if e.PlayerTagList != nil {
		out["playerTagList"] = Shadow(e.PlayerTagList)
	}
	if e.ModSuffix != "" {
		out["modSuffix"] = e.ModSuffix
	}
	return out
}

func writeCanon(sb *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		sb.WriteString("null")
	case bool:
		sb.WriteString(strconv.FormatBool(t))
	case int:
		sb.WriteString(strconv.Itoa(t))
	case int64:
		sb.WriteString(strconv.FormatInt(t, 10))
	case uint64:
		sb.WriteString(strconv.FormatUint(t, 10))
	case fmt.Stringer: // the parser's form enum prints as its reference text
		writeCanonString(sb, t.String())
	case float64:
		sb.WriteString(canonNumber(t))
	case string:
		writeCanonString(sb, t)
	case Fn:
		sb.WriteString(`{"__fn":true}`)
	case []any:
		// A pure array: numbered keys starting at 1, emitted in STRING-sorted
		// order (Lua-side canon sorts stringified keys, so "10" precedes "2").
		// A nil element vanishes, as a nil in a Lua constructor leaves a hole.
		keys := make([]string, 0, len(t))
		for i, e := range t {
			if e != nil {
				keys = append(keys, strconv.Itoa(i+1))
			}
		}
		sort.Strings(keys)
		sb.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeCanonString(sb, k)
			sb.WriteByte(':')
			idx, _ := strconv.Atoi(k)
			writeCanon(sb, t[idx-1])
		}
		sb.WriteByte('}')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			if t[k] == nil {
				continue // Lua tables have no nil-valued keys
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		sb.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeCanonString(sb, k)
			sb.WriteByte(':')
			writeCanon(sb, t[k])
		}
		sb.WriteByte('}')
	default:
		if reflect.ValueOf(v).Kind() == reflect.Func {
			sb.WriteString(`{"__fn":true}`)
			return
		}
		writeCanonString(sb, "?unsupported?")
	}
}

// canonNumber matches canon.lua: whole numbers print with no decimal point,
// everything else with %.14g; infinities and NaN print as Lua's tostring
// renders them, quoted.
func canonNumber(f float64) string {
	if math.IsInf(f, 1) {
		return `"inf"`
	}
	if math.IsInf(f, -1) {
		return `"-inf"`
	}
	if math.IsNaN(f) {
		return `"nan"`
	}
	if f == float64(int64(f)) && f < 1e15 && f > -1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return formatG14(f)
}

// formatG14 mimics C's %.14g, which Lua's string.format uses. Go writes
// exponents as e+06, as C does, for the magnitudes mod values take.
func formatG14(f float64) string {
	return strconv.FormatFloat(f, 'g', 17, 64)
}

var canonEscapes = map[byte]string{
	'"': `\"`, '\\': `\\`, '\b': `\b`, '\f': `\f`, '\n': `\n`, '\r': `\r`, '\t': `\t`,
}

func writeCanonString(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if esc, ok := canonEscapes[c]; ok {
			sb.WriteString(esc)
			continue
		}
		if c < 0x20 || c == 0x7f {
			const hex = "0123456789abcdef"
			sb.WriteString(`\u00`)
			sb.WriteByte(hex[c>>4])
			sb.WriteByte(hex[c&0xf])
			continue
		}
		sb.WriteByte(c)
	}
	sb.WriteByte('"')
}

// numericStringField matches the closed set of numeric tag fields — plus
// the mod value itself — that the reference's parser filled from raw text
// captures ("div":"5", "value":"2"); the port parses them at parse time,
// so the archive side is normalised to numbers before comparison
// (lua-residue.md T3).
var numericStringField = regexp.MustCompile(`"(div|limit|percent|base|threshold|thresholdPercent|globalLimit|value)":"(-?[0-9]+(?:\.[0-9]+)?)"`)

// NormalizeArchiveMods rewrites a fixture's canon text so numeric strings
// in the closed numeric tag fields read as numbers.
func NormalizeArchiveMods(canon string) string {
	return numericStringField.ReplaceAllString(canon, `"$1":$2`)
}
