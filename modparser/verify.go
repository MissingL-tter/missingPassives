package modparser

// Tables exposes every pattern table in its finished form — after the
// construction loops and specialModList's anchoring — keyed by the reference's
// table names. Test scaffolding only: it exists for the modtables differential
// under test/, which verifies each entry against the reference's own tables.
//
// Values are the tables' typed entries as the TableEntry union: the form enum
// (a Stringer), TableStr, TableStrs, *PatternEntry, TableMods, FlagTypeMod,
// JewelFn, TableBool, and TableFn for every closure.
func Tables() map[string]map[string]TableEntry {
	tables := map[string]map[string]TableEntry{}
	tables["formList"] = sealedTable(formList)
	tables["modNameList"] = boxedTable(modNameList, boxName)
	tables["modFlagList"] = sealedTable(modFlagList)
	tables["preFlagList"] = boxedTable(preFlagList, boxEntry)
	tables["modTagList"] = boxedTable(modTagList, boxEntry)
	tables["specialModList"] = boxedTable(specialT.m, boxMods)
	tables["unsupportedModList"] = boxedTable(unsupportedModList, boxBool)
	tables["suffixTypes"] = boxedTable(suffixTypes, boxStr)
	tables["dmgTypes"] = boxedTable(dmgTypes, boxStr)
	tables["penTypes"] = boxedTable(penTypes, boxName)
	tables["resourceTypes"] = boxedTable(resourceTypes, boxName)
	tables["regenTypes"] = boxedTable(regenTypes, boxName)
	tables["degenTypes"] = boxedTable(degenTypes, boxName)
	tables["costTypes"] = boxedTable(costTypes, boxName)
	tables["baseCostTypes"] = boxedTable(baseCostTypes, boxName)
	tables["flagTypes"] = boxedTable(flagTypes, boxFlagType)
	tables["skillNameList"] = sealedTable(skillNameList)
	tables["preSkillNameList"] = sealedTable(preSkillNameList)
	tables["clusterJewelSkills"] = boxedTable(clusterJewelSkills, boxModList)
	jewels := make(map[string]TableEntry, len(jewelFuncList))
	for k, e := range jewelFuncList {
		// Both fields of the Lua entry: the function and the type label.
		jewels[k] = JewelFn{Type: e.typ, Func: func(JewelNodeRef, JewelStoreWriter, *JewelFuncTag) {}}
	}
	tables["jewelFuncList"] = jewels
	return tables
}

// sealedTable renders a table whose values already satisfy TableEntry.
func sealedTable[V TableEntry](m map[string]V) map[string]TableEntry {
	out := make(map[string]TableEntry, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// boxedTable renders a table whose values need converting first.
func boxedTable[V any](m map[string]V, box func(V) TableEntry) map[string]TableEntry {
	out := make(map[string]TableEntry, len(m))
	for k, v := range m {
		out[k] = box(v)
	}
	return out
}

func boxStr(s string) TableEntry     { return TableStr(s) }
func boxBool(b bool) TableEntry      { return TableBool(b) }
func boxModList(l []*Mod) TableEntry { return TableMods(l) }

func boxName(v nameValue) TableEntry {
	switch t := v.(type) {
	case name:
		return TableStr(t)
	case nameList:
		return TableStrs(t)
	case *PatternEntry:
		return t
	}
	panic("modparser: unhandled nameValue in Tables()")
}

func boxEntry(v entryValue) TableEntry {
	switch t := v.(type) {
	case *PatternEntry:
		return t
	case entryFn:
		return TableFn{}
	}
	panic("modparser: unhandled entryValue in Tables()")
}

func boxMods(v modsValue) TableEntry {
	switch t := v.(type) {
	case modList:
		return TableMods(t)
	case modFn:
		return TableFn{}
	}
	panic("modparser: unhandled modsValue in Tables()")
}

func boxFlagType(v flagTypeValue) TableEntry {
	switch t := v.(type) {
	case flagName:
		return TableStr(t)
	case FlagTypeMod:
		return t
	}
	panic("modparser: unhandled flagTypeValue in Tables()")
}
