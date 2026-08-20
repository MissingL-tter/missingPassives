package modparser

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// The rest of modLib — .archive/src/Modules/ModTools.lua. createMod is mod()
// in mod.go; parseMod is Parse in parse.go. Everything here is the pure
// remainder: tag parsing, the canonical mod formatting family, comparison and
// source stamping. mergeKeystones is not here — it operates on a live ModDB
// and the tree's keystone map, so it belongs to the mod-store port.

// ParseTags — ModTools.lua:50: "type=Condition/var=X,type=..." into tags.
// Comma separates tags, slash separates params; only "threshold" values are
// numeric, and the literal "true" becomes a boolean.
func ParseTags(line string) []any {
	if line == "" || line == "-" {
		return []any{}
	}
	tags := []any{}
	for _, tagGroup := range strings.Split(line, ",") {
		if tagGroup == "" {
			continue
		}
		tagSet := Tag{}
		for _, tag := range strings.Split(tagGroup, "/") {
			if tag == "" {
				continue
			}
			name, value, ok := matchTagParam(tag)
			if !ok {
				continue // the reference logs "Error tag invalid" and skips
			}
			if name == "threshold" {
				if n, isNum := tonumber(value); isNum {
					tagSet[name] = n
				} else {
					tagSet[name] = nil
				}
				continue
			}
			if value == "true" {
				tagSet[name] = true
			} else {
				tagSet[name] = value
			}
		}
		tags = append(tags, tagSet)
	}
	return tags
}

// matchTagParam mirrors tag:match("^(%a+)=(.+)").
func matchTagParam(tag string) (string, string, bool) {
	eq := strings.IndexByte(tag, '=')
	if eq <= 0 || eq == len(tag)-1 {
		return "", "", false
	}
	name := tag[:eq]
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return "", "", false
		}
	}
	return name, tag[eq+1:], true
}

// ParseFormattedSourceMod — ModTools.lua:78: the inverse of FormatSourceMod,
// over "value|source|name|type|flags|keywordFlags|tags".
func ParseFormattedSourceMod(line string) *Mod {
	modStrings := strings.Split(line, "|")
	if len(modStrings) < 4 {
		return nil
	}
	var value any
	if modStrings[0] == "true" {
		value = true
	} else if n, ok := tonumber(modStrings[0]); ok {
		value = n
	} else {
		value = float64(0)
	}
	m := &Mod{
		Value:     value,
		Source:    modStrings[1],
		SourceSet: true,
		Name:      modStrings[2],
		Type:      modStrings[3],
	}
	if len(modStrings) >= 5 {
		m.Flags = modFlagNames[modStrings[4]] // or 0, as ModFlag[s] or 0
	}
	if len(modStrings) >= 6 {
		m.KeywordFlags = keywordFlagNames[modStrings[5]]
	}
	if len(modStrings) >= 7 {
		m.Tags = append(m.Tags, ParseTags(modStrings[6])...)
	}
	return m
}

// CompareModParams — ModTools.lua:99: same name, type, flags, keyword flags,
// and identical tags by formatted equality.
func CompareModParams(modA, modB *Mod) bool {
	// #mod in the reference counts the constructor-allocated array slots, so
	// a nil hole still counts toward the length even though ipairs stops there.
	if modA.Name != modB.Name || modA.Type != modB.Type ||
		modA.Flags != modB.Flags || modA.KeywordFlags != modB.KeywordFlags ||
		len(modA.Tags) != len(modB.Tags) {
		return false
	}
	for i := 0; i < tagArrayLen(modA); i++ {
		ta, tb := asTag(modA.Tags[i]), asTag(modB.Tags[i])
		if ta["type"] != tb["type"] {
			return false
		}
		if FormatTag(ta) != FormatTag(tb) {
			return false
		}
	}
	return true
}

// tagArrayLen mirrors Lua's ipairs/# over the mod's array part: a nil tag is a
// hole that ends iteration.
func tagArrayLen(m *Mod) int {
	for i, t := range m.Tags {
		if t == nil {
			return i
		}
	}
	return len(m.Tags)
}

// FormatFlags — ModTools.lua:114: the sorted names of every flag whose bits
// are wholly contained in flags, comma-joined; "-" when none.
func FormatFlags(flags int64, src map[string]int64) string {
	var names []string
	for name, val := range src {
		if flags&val == val {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "-"
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// luaTostring matches Lua's tostring for the value types mods carry.
func luaTostring(v any) string {
	switch t := v.(type) {
	case nil:
		return "nil"
	case bool:
		return strconv.FormatBool(t)
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		if math.IsInf(t, 1) {
			return "inf"
		}
		if math.IsInf(t, -1) {
			return "-inf"
		}
		if math.IsNaN(t) {
			return "nan"
		}
		return fmt.Sprintf("%.14g", t)
	}
	return fmt.Sprintf("%v", v)
}

// tagKeys returns a tag's param names sorted, with "type" forced first when
// present — the ordering formatTag and formatValue share.
func tagKeys(t Tag) ([]string, bool) {
	names := make([]string, 0, len(t))
	haveType := false
	for name, val := range t {
		if val == nil {
			continue // a nil value is an absent key in Lua
		}
		if name == "type" {
			haveType = true
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if haveType {
		names = append([]string{"type"}, names...)
	}
	return names, haveType
}

// FormatTag — ModTools.lua:129.
func FormatTag(tag Tag) string {
	names, _ := tagKeys(tag)
	var sb strings.Builder
	for i, name := range names {
		if i > 0 {
			sb.WriteByte('/')
		}
		val := tag[name]
		if s, isTable := formatTagTableValue(val); isTable {
			sb.WriteString(fmt.Sprintf("%s={%s}", name, s))
		} else {
			sb.WriteString(fmt.Sprintf("%s=%s", name, luaTostring(val)))
		}
	}
	return sb.String()
}

// formatTagTableValue handles a table-valued tag param: a list of tags becomes
// FormatTags, a list of scalars is comma-joined, a plain table recurses.
func formatTagTableValue(val any) (string, bool) {
	switch t := val.(type) {
	case []any:
		if len(t) == 0 {
			// An empty Lua table has no [1]; the reference recurses formatTag
			// on it, yielding "".
			return "", true
		}
		if _, isTag := t[0].(Tag); isTag {
			return FormatTags(anyToTags(t)), true
		}
		if _, isList := t[0].([]any); isList {
			// list of lists (e.g. DistanceRamp ramps) — formatTags formats each
			// with formatTag over numeric keys
			return FormatTags(anyToTags(t)), true
		}
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = luaTostring(e)
		}
		return strings.Join(parts, ","), true
	case Tag:
		return FormatTag(t), true
	case *D:
		// Transformed tables carry their array part in a *D.
		if len(t.KV) == 0 {
			s, _ := formatTagTableValue(t.Arr)
			return s, true
		}
		if len(t.Arr) == 0 {
			return FormatTag(Tag(t.KV)), true
		}
		panic("formatTagTableValue: mixed array/hash tag value")
	}
	return "", false
}

// anyToTags views a list's elements as tags; a scalar-list element becomes a
// numeric-keyed tag exactly as Lua's pairs would see it.
func anyToTags(list []any) []Tag {
	out := make([]Tag, 0, len(list))
	for _, e := range list {
		switch t := e.(type) {
		case Tag:
			out = append(out, t)
		case []any:
			tt := Tag{}
			for i, v := range t {
				tt[strconv.Itoa(i+1)] = v
			}
			out = append(out, tt)
		}
	}
	return out
}

// FormatTags — ModTools.lua:166.
func FormatTags(tagList []Tag) string {
	if len(tagList) == 0 {
		return "-"
	}
	parts := make([]string, len(tagList))
	for i, tag := range tagList {
		parts[i] = FormatTag(tag)
	}
	return strings.Join(parts, ",")
}

// FormatValue — ModTools.lua:174: scalars via tostring; tables as
// {k=v/k=v...} with "type" first, and a "mod" param embedding FormatMod.
func FormatValue(value any) string {
	switch t := value.(type) {
	case Tag:
		names, _ := tagKeys(t)
		var sb strings.Builder
		for i, name := range names {
			if i > 0 {
				sb.WriteByte('/')
			}
			if name == "mod" {
				sb.WriteString(fmt.Sprintf("%s=[%s]", name, FormatMod(t[name].(*Mod))))
			} else {
				sb.WriteString(fmt.Sprintf("%s=%s", name, FormatValue(t[name])))
			}
		}
		return "{" + sb.String() + "}"
	case []any:
		// A Lua array formats through the same path: numeric keys, sorted.
		var sb strings.Builder
		for i, e := range t {
			if i > 0 {
				sb.WriteByte('/')
			}
			sb.WriteString(fmt.Sprintf("%d=%s", i+1, FormatValue(e)))
		}
		return "{" + sb.String() + "}"
	case *Mod:
		// A mod under any key other than "mod" formats as its plain table via
		// pairs: named fields sorted with "type" first. Tags would add numeric
		// keys, which Lua's t_sort cannot order against strings — the reference
		// would error there, so this port refuses the same shape loudly.
		if tagArrayLen(t) > 0 {
			panic("FormatValue: nested mod with tags has no defined ordering (the reference errors here too)")
		}
		kv := Tag{"name": t.Name, "type": t.Type, "value": t.Value, "flags": t.Flags, "keywordFlags": t.KeywordFlags}
		if t.SourceSet {
			kv["source"] = t.Source
		}
		return FormatValue(kv)
	}
	return luaTostring(value)
}

// FormatModParams — ModTools.lua:205: "name|type|flags|keywordFlags|tags".
func FormatModParams(m *Mod) string {
	tags := ModTags(m)
	return fmt.Sprintf("%s|%s|%s|%s|%s", m.Name, m.Type,
		FormatFlags(m.Flags, modFlagNames),
		FormatFlags(m.KeywordFlags, keywordFlagNames),
		FormatTags(tags))
}

// FormatMod — ModTools.lua:209.
func FormatMod(m *Mod) string {
	return FormatValue(m.Value) + " = " + FormatModParams(m)
}

// FormatSourceMod — ModTools.lua:213.
func FormatSourceMod(m *Mod) string {
	return fmt.Sprintf("%s|%s|%s", FormatValue(m.Value), m.Source, FormatModParams(m))
}

// SetSource — ModTools.lua:217: stamps the mod, and a nested value.mod too.
func SetSource(m *Mod, source string) *Mod {
	m.Source = source
	m.SourceSet = true
	if vt, ok := m.Value.(Tag); ok {
		if inner, ok := vt["mod"].(*Mod); ok {
			inner.Source = source
			inner.SourceSet = true
		}
	}
	return m
}

// CopyMod deep-copies a mod, its tags and its value, the way the reference's
// copyTable does before handing out cached or table-stored mods.
func CopyMod(m *Mod) *Mod {
	cp := *m
	cp.Tags = copyAnyList(m.Tags)
	cp.Value = copyAny(m.Value)
	return &cp
}

func copyAny(v any) any {
	switch t := v.(type) {
	case Tag:
		out := make(Tag, len(t))
		for k, e := range t {
			out[k] = copyAny(e)
		}
		return out
	case []any:
		return copyAnyList(t)
	case *Mod:
		return CopyMod(t)
	case *D:
		return &D{Arr: copyAnyList(t.Arr), KV: copyAny(Tag(t.KV)).(Tag)}
	}
	return v
}

func copyAnyList(list []any) []any {
	if list == nil {
		return nil
	}
	out := make([]any, len(list))
	for i, e := range list {
		out[i] = copyAny(e)
	}
	return out
}

// ModTags returns a mod's tag array up to the first hole, each viewed as a
// Tag (transformed tables may store tags as *D).
func ModTags(m *Mod) []Tag {
	var out []Tag
	for _, t := range m.Tags {
		if t == nil {
			break
		}
		out = append(out, asTag(t))
	}
	return out
}
