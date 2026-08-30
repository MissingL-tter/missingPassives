package modparser

// Tables exposes every pattern table in its finished form — after the
// construction loops and specialModList's anchoring — keyed by the reference's
// table names. Test scaffolding only: it exists for the modtables differential
// under test/, which verifies each entry against the reference's own tables.
//
// Values are the tables' typed entries in exported form: modForm (a
// Stringer), string, []string, *PatternEntry, []*Mod, FlagTypeMod, JewelFn,
// bool, and TableFn for every closure.
func Tables() map[string]map[string]any {
	box := func(v any) any {
		switch t := v.(type) {
		case name:
			return string(t)
		case nameList:
			return []string(t)
		case modList:
			return []*Mod(t)
		case modFn, entryFn:
			return TableFn{}
		case flagName:
			return string(t)
		}
		return v
	}
	boxed := func(m map[string]any) map[string]any {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = box(v)
		}
		return out
	}
	tables := map[string]map[string]any{}
	tables["formList"] = anyMap(formList)
	tables["modNameList"] = boxed(anyMap(modNameList))
	tables["modFlagList"] = anyMap(modFlagList)
	tables["preFlagList"] = boxed(anyMap(preFlagList))
	tables["modTagList"] = boxed(anyMap(modTagList))
	tables["specialModList"] = boxed(anyMap(specialT.m))
	tables["unsupportedModList"] = anyMap(unsupportedModList)
	tables["suffixTypes"] = anyMap(suffixTypes)
	tables["dmgTypes"] = anyMap(dmgTypes)
	tables["penTypes"] = boxed(anyMap(penTypes))
	tables["resourceTypes"] = boxed(anyMap(resourceTypes))
	tables["regenTypes"] = boxed(anyMap(regenTypes))
	tables["degenTypes"] = boxed(anyMap(degenTypes))
	tables["costTypes"] = boxed(anyMap(costTypes))
	tables["baseCostTypes"] = boxed(anyMap(baseCostTypes))
	tables["flagTypes"] = boxed(anyMap(flagTypes))
	tables["skillNameList"] = anyMap(skillNameList)
	tables["preSkillNameList"] = anyMap(preSkillNameList)
	tables["clusterJewelSkills"] = anyMap(clusterJewelSkills)
	jewels := make(map[string]any, len(jewelFuncList))
	for k, e := range jewelFuncList {
		// Both fields of the Lua entry: the function and the type label.
		jewels[k] = JewelFn{Type: e.typ, Func: func(JewelNodeRef, JewelStoreWriter, *JewelFuncTag) {}}
	}
	tables["jewelFuncList"] = jewels
	return tables
}

func anyMap[V any](m map[string]V) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
