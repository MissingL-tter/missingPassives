package modparser

// Tables exposes every pattern table in its finished form — after the
// construction loops and specialModList's anchoring — keyed by the reference's
// table names. It exists for the differential test suite under go/test, which
// verifies each entry against the reference's own tables.
func Tables() map[string]map[string]any {
	jewels := make(map[string]any, len(jewelFuncList))
	for k, e := range jewelFuncList {
		// Both fields of the Lua entry: the function and the type label.
		if e.factory != nil {
			jewels[k] = Tag{"func": e.factory, "type": e.typ}
		} else {
			jewels[k] = Tag{"func": e.nodeFn, "type": e.typ}
		}
	}
	unsupported := make(map[string]any, len(unsupportedModList))
	for k, v := range unsupportedModList {
		unsupported[k] = v
	}
	return map[string]map[string]any{
		"formList":           formList,
		"modNameList":        modNameList,
		"modFlagList":        modFlagList,
		"preFlagList":        preFlagList,
		"modTagList":         modTagList,
		"specialModList":     specialT.m,
		"unsupportedModList": unsupported,
		"suffixTypes":        suffixTypes,
		"dmgTypes":           dmgTypes,
		"penTypes":           penTypes,
		"resourceTypes":      resourceTypes,
		"regenTypes":         regenTypes,
		"degenTypes":         degenTypes,
		"costTypes":          costTypes,
		"baseCostTypes":      baseCostTypes,
		"flagTypes":          flagTypes,
		"skillNameList":      skillNameList,
		"preSkillNameList":   preSkillNameList,
		"jewelFuncList":      jewels,
		"clusterJewelSkills": clusterJewelSkills,
	}
}
