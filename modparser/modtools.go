package modparser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
)

// The rest of modLib — .archive/src/Modules/ModTools.lua. createMod is mod()
// in mod.go; parseMod is Parse in parse.go. Everything here is the pure
// remainder: tag parsing, the canonical mod formatting family, comparison and
// source stamping. mergeKeystones is not here — it operates on a live ModDB
// and the tree's keystone map, so it belongs to the mod-store port.

// ParseTags — ModTools.lua:50: "type=Condition/var=X,type=..." into raw
// tags. Comma separates tags, slash separates params; only "threshold"
// values are numeric (a non-numeric one is dropped), and the literal "true"
// becomes a boolean.
func ParseTags(line string) []Tag {
	if line == "" || line == "-" {
		return []Tag{}
	}
	tags := []Tag{}
	for _, tagGroup := range strings.Split(line, ",") {
		if tagGroup == "" {
			continue
		}
		tagSet := &RawTag{}
		for _, tag := range strings.Split(tagGroup, "/") {
			if tag == "" {
				continue
			}
			name, value, ok := matchTagParam(tag)
			if !ok {
				continue // the reference logs "Error tag invalid" and skips
			}
			switch {
			case name == "type":
				tagSet.Type = value
			case name == "threshold":
				if n, isNum := util.Tonumber(value); isNum {
					tagSet.Fields = append(tagSet.Fields, Param{name, Num(n)})
				}
			case value == "true":
				tagSet.Fields = append(tagSet.Fields, Param{name, Bool(true)})
			default:
				tagSet.Fields = append(tagSet.Fields, Param{name, Str(value)})
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

// FormattedSourceMod is what ParseFormattedSourceMod reads back: the
// reference stores the type text verbatim (any string, not only a mod type),
// so the record keeps it as text rather than forcing it into ModType.
type FormattedSourceMod struct {
	Value        Value
	Source, Name string
	Type         string
	Flags        ModFlag
	KeywordFlags KeywordFlag
	Tags         []Tag
}

// ParseFormattedSourceMod — ModTools.lua:78: the inverse of FormatSourceMod,
// over "value|source|name|type|flags|keywordFlags|tags".
func ParseFormattedSourceMod(line string) *FormattedSourceMod {
	modStrings := strings.Split(line, "|")
	if len(modStrings) < 4 {
		return nil
	}
	var value Value
	if modStrings[0] == "true" {
		value = Bool(true)
	} else if n, ok := util.Tonumber(modStrings[0]); ok {
		value = Num(n)
	} else {
		value = Num(0)
	}
	m := &FormattedSourceMod{
		Value:  value,
		Source: modStrings[1],
		Name:   modStrings[2],
		Type:   modStrings[3],
	}
	if len(modStrings) >= 5 {
		m.Flags = ModFlagByName[modStrings[4]] // or 0, as ModFlag[s] or 0
	}
	if len(modStrings) >= 6 {
		m.KeywordFlags = KeywordFlagByName[modStrings[5]]
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
		ta, tb := modA.Tags[i], modB.Tags[i]
		if tb == nil || TagTypeName(ta) != TagTypeName(tb) {
			return false
		}
		if FormatTag(ta) != FormatTag(tb) {
			return false
		}
	}
	return true
}

// FormatFlags — ModTools.lua:114: the sorted names of every flag whose bits
// are wholly contained in flags, comma-joined; "-" when none.
func FormatFlags[F ModFlag | KeywordFlag](flags F, src map[string]F) string {
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

// formatParamValue spells one scalar mod or tag param. The whole Format*
// family here is PoB's own canonical mod serialisation (ModTools.lua
// formatMod/formatTag/formatValue), kept deliberately: the product consumes
// this text (flask cache keys, tree node ModKey) and the differentials
// compare it byte for byte, so the spelling is fixed — numbers at %.14g
// through util.FormatG14, flags and skill types as decimal integers.
func formatParamValue(v ParamValue) string {
	switch t := v.(type) {
	case Str:
		return string(t)
	case Num:
		return util.FormatG14(float64(t))
	case Bool:
		return strconv.FormatBool(bool(t))
	case ModFlag:
		return strconv.FormatUint(uint64(t), 10)
	case KeywordFlag:
		return strconv.FormatUint(uint64(t), 10)
	case SkillTypeID:
		return strconv.FormatInt(int64(t), 10)
	}
	panic(fmt.Sprintf("modparser: %T has no scalar mod-text spelling", v))
}

// tagParams returns a tag's params with "type" forced first — the ordering
// formatTag and formatValue share.
func tagParams(t Tag) []Param {
	params := t.Params()
	if typ := TagTypeName(t); typ != "" {
		return append([]Param{{"type", Str(typ)}}, params...)
	}
	return params
}

// FormatTag — ModTools.lua:129.
func FormatTag(tag Tag) string {
	var sb strings.Builder
	for i, p := range tagParams(tag) {
		if i > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(p.Name + "=" + formatTagParam(p.Value))
	}
	return sb.String()
}

// formatTagParam spells one tag param: a list becomes {a,b} — formatTag
// comma-joins a list of scalars and numeric-keys a list of pairs — and
// anything else takes its scalar spelling.
func formatTagParam(v ParamValue) string {
	switch t := v.(type) {
	case StrList:
		return "{" + strings.Join(t, ",") + "}"
	case NumList:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = util.FormatG14(e)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case SkillTypeList:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = strconv.FormatInt(int64(e), 10)
		}
		return "{" + strings.Join(parts, ",") + "}"
	case Pairs:
		parts := make([]string, len(t))
		for i, pair := range t {
			parts[i] = "1=" + util.FormatG14(pair[0]) + "/2=" + util.FormatG14(pair[1])
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
	return formatParamValue(v)
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

// FormatValue — ModTools.lua:174: scalars via tostring; records as
// {k=v/k=v...} with "type" first, and a "mod" param embedding FormatMod.
func FormatValue(value Value) string {
	switch t := value.(type) {
	case nil:
		return "nil"
	case Num:
		return util.FormatG14(float64(t))
	case Bool:
		return strconv.FormatBool(bool(t))
	case Str:
		return string(t)
	}
	_, params, ok := ValueParams(value)
	if !ok {
		panic(fmt.Sprintf("modparser: %T is not a listed value kind", value))
	}
	names := make([]string, 0, len(params))
	byName := make(map[string]ParamValue, len(params))
	haveType := false
	for _, p := range params {
		if p.Name == "type" {
			haveType = true
		} else {
			names = append(names, p.Name)
		}
		byName[p.Name] = p.Value
	}
	sort.Strings(names)
	if haveType {
		names = append([]string{"type"}, names...)
	}
	var sb strings.Builder
	for i, name := range names {
		if i > 0 {
			sb.WriteByte('/')
		}
		switch pv := byName[name].(type) {
		case *Mod:
			if name == "mod" {
				sb.WriteString(name + "=[" + FormatMod(pv) + "]")
			} else {
				sb.WriteString(name + "=" + formatPlainMod(pv))
			}
		case Conqueror:
			sb.WriteString(name + "=" + formatConqueror(pv))
		case StrList:
			// formatValue recurses into a list as a numeric-keyed table.
			parts := make([]string, len(pv))
			for i, e := range pv {
				parts[i] = strconv.Itoa(i+1) + "=" + e
			}
			sb.WriteString(name + "={" + strings.Join(parts, "/") + "}")
		case Value:
			// Scalars included: FormatValue spells those itself.
			sb.WriteString(name + "=" + FormatValue(pv))
		case JewelFnRef:
			// Lua prints "function: <address>" here. An address is not stable
			// across runs, which defeats what the text is for (modKey is a
			// cache key), so the function's own id is spelled instead; the
			// tree differential normalises both spellings
			// (test/tree_test.go funcAddrRe).
			sb.WriteString(name + "=" + pv.ID)
		default:
			panic(fmt.Sprintf("modparser: %T has no mod-text spelling", pv))
		}
	}
	return "{" + sb.String() + "}"
}

// formatPlainMod formats a mod under any key other than "mod": its plain
// table via pairs, named fields sorted with "type" first. Tags would add
// numeric keys, which Lua's t_sort cannot order against strings — the
// reference would error there, so this port refuses the same shape loudly.
func formatPlainMod(m *Mod) string {
	if tagArrayLen(m) > 0 {
		panic("FormatValue: nested mod with tags has no defined ordering (the reference errors here too)")
	}
	parts := []string{"type=" + m.Type.String(), "flags=" + formatParamValue(m.Flags), "keywordFlags=" + formatParamValue(m.KeywordFlags), "name=" + m.Name}
	if m.SourceSet {
		parts = append(parts, "source="+m.Source)
	}
	parts = append(parts, "value="+FormatValue(m.Value))
	sort.Strings(parts[1:])
	return "{" + strings.Join(parts, "/") + "}"
}

func formatConqueror(c Conqueror) string {
	return "{type=" + c.Kind.String() + "/id=" + c.IDText() + "}"
}

// FormatModParams — ModTools.lua:205: "name|type|flags|keywordFlags|tags".
func FormatModParams(m *Mod) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", m.Name, m.Type,
		FormatFlags(m.Flags, ModFlagByName),
		FormatFlags(m.KeywordFlags, KeywordFlagByName),
		FormatTags(ModTags(m)))
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
	if ref, ok := m.Value.(ModRef); ok && ref.Mod != nil {
		ref.Mod.Source = source
		ref.Mod.SourceSet = true
	}
	return m
}
