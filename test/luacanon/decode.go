package luacanon

import (
	"fmt"
	"math"
	"strconv"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// Decoding of the archive's plain-table mod shape back into typed mods, for
// fixtures that carry whole mod stores (calc, modstore synth records).

// ModFromTable rebuilds a mod from its canon table (as JSON decodes it).
func ModFromTable(m map[string]any) *modparser.Mod {
	mod := &modparser.Mod{
		Name:         m["name"].(string),
		Type:         modparser.ModTypeByName[m["type"].(string)],
		Flags:        modparser.ModFlag(m["flags"].(float64)),
		KeywordFlags: modparser.KeywordFlag(m["keywordFlags"].(float64)),
	}
	if s, ok := m["source"].(string); ok {
		mod.Source = s
		mod.SourceSet = true
	}
	if s, ok := m["sourceSlot"].(string); ok {
		mod.SourceSlot = s
	}
	if v, ok := m["value"]; ok {
		mod.Value = ValueFromTable(v)
	}
	// Numbered keys up to the highest present; a missing index is a hole.
	last := 0
	for k := range m {
		if n, err := strconv.Atoi(k); err == nil && n > last {
			last = n
		}
	}
	for i := 1; i <= last; i++ {
		if tv, ok := m[strconv.Itoa(i)]; ok {
			mod.Tags = append(mod.Tags, TagFromTable(tv.(map[string]any)))
		} else {
			mod.Tags = append(mod.Tags, nil)
		}
	}
	return mod
}

// TagFromTable rebuilds a tag from its canon table.
func TagFromTable(m map[string]any) modparser.Tag {
	typ, _ := m["type"].(string)
	params := make([]modparser.Param, 0, len(m))
	for k, v := range m {
		if k == "type" {
			continue
		}
		params = append(params, modparser.Param{Name: k, Value: modparser.ParamOf(canonList(v))})
	}
	t, ok := modparser.TagFromParams(typ, params)
	if !ok {
		panic(fmt.Sprintf("luacanon: cannot decode tag %v", m))
	}
	return t
}

// canonList turns a numbered-key table into []any (nested), leaving other
// values as they are; "inf"/"-inf" strings are the quoted infinities.
func canonList(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			return []any{} // an empty Lua table in a list position
		}
		list, ok := numberedList(t)
		if !ok {
			return t
		}
		for i, e := range list {
			list[i] = canonList(e)
		}
		return list
	case string:
		switch t {
		case "inf":
			return math.Inf(1)
		case "-inf":
			return math.Inf(-1)
		}
	}
	return v
}

// numberedList reads a table whose keys are 1..n.
func numberedList(m map[string]any) ([]any, bool) {
	if len(m) == 0 {
		return nil, false
	}
	out := make([]any, len(m))
	for i := 1; i <= len(m); i++ {
		e, ok := m[strconv.Itoa(i)]
		if !ok {
			return nil, false
		}
		out[i-1] = e
	}
	return out, true
}

// ValueFromTable rebuilds a value from its canon shape, choosing the record
// kind by key set.
func ValueFromTable(v any) modparser.Value {
	switch t := v.(type) {
	case nil:
		return nil
	case float64:
		return modparser.Num(t)
	case bool:
		return modparser.Bool(t)
	case string:
		switch t {
		case "inf":
			return modparser.Num(math.Inf(1))
		case "-inf":
			return modparser.Num(math.Inf(-1))
		}
		return modparser.Str(t)
	case map[string]any:
		if list, ok := numberedList(t); ok {
			pairs := make(modparser.Pairs, len(list))
			for i, e := range list {
				pair, _ := numberedList(e.(map[string]any))
				pairs[i] = [2]float64{pair[0].(float64), pair[1].(float64)}
			}
			return pairs
		}
		kind, params := recordParams(t)
		val, ok := modparser.ValueFromParams(kind, params)
		if !ok {
			panic(fmt.Sprintf("luacanon: cannot decode value %v", t))
		}
		return val
	}
	panic(fmt.Sprintf("luacanon: cannot decode value %T", v))
}

func has(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// recordParams classifies a record table by its keys and converts nested
// values.
func recordParams(m map[string]any) (string, []modparser.Param) {
	var kind string
	switch {
	case has(m, "mod"):
		kind = "mod"
	case len(m) == 1 && has(m, "value"):
		kind = "propertyMod"
	case has(m, "keyword", "keywordList", "keyOfScaledMod") && !has(m, "chance"):
		kind = "gemProperty"
	case has(m, "key"):
		kind = "data"
	case has(m, "chance", "amount"):
		kind = "explode"
	case has(m, "func"):
		kind = "jewelFn"
	case has(m, "conqueror") || (len(m) == 1 && has(m, "id")):
		kind = "conqueredBy"
	case has(m, "baseDamage", "dmgMult"):
		kind = "selfDamage"
	case has(m, "side"):
		kind = "ascendancyNode"
	case has(m, "targetSlotName"):
		kind = "linkedSupport"
	default:
		kind = "skill"
	}
	params := make([]modparser.Param, 0, len(m))
	for k, v := range m {
		var pv any
		switch {
		case k == "mod" || k == "value" && kind == "propertyMod":
			pv = ModFromTable(v.(map[string]any))
		case k == "value" && kind == "data":
			pv = ValueFromTable(v)
		case k == "conqueror":
			c := v.(map[string]any)
			cq := modparser.Conqueror{Kind: modparser.ConquerorKindByName[c["type"].(string)]}
			switch id := c["id"].(type) {
			case float64:
				cq.Index = int(id)
			case string:
				n, _ := strconv.Atoi(id[:len(id)-3])
				cq.Index, cq.V2 = n, true
			}
			pv = cq
		case k == "func":
			// The canon marks a function's presence only; a placeholder keeps
			// "func" in the value so a re-encode reproduces the fixture.
			pv = modparser.JewelFnRef{Fn: func(modparser.JewelNodeRef, modparser.JewelStoreWriter, *modparser.JewelFuncTag) {}}
		default:
			pv = canonList(v)
		}
		params = append(params, modparser.Param{Name: k, Value: modparser.ParamOf(pv)})
	}
	return kind, params
}
