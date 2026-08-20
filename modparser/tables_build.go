package modparser

import (
	"regexp"
	"strings"
)

// Construction of the data-driven tables — ModParser.lua:6040-6094 (skill name
// lookups plus per-skill specialModList entries), 5881-5886 (keystones) and
// 6540-6553 (cluster jewel skills). The loops are ported as written; the data
// they iterate comes from vocab_gen.go.

// skillLists bundles the three tables the gem loop fills, so plain var
// initialisers (which Go orders by dependency, unlike init functions) build
// them before the scan tables that read them.
type skillListsT struct {
	skillNames, preSkillNames, gemSpecials map[string]any
}

var skillLists = buildSkillLists()

var skillNameList = skillLists.skillNames
var preSkillNameList = skillLists.preSkillNames

// gemSpecialMods holds the specialModList entries the gem loop generates; they
// are merged with the literal entries when the scan table is built.
var gemSpecialMods = skillLists.gemSpecials

func buildSkillLists() skillListsT {
	skillNameList := map[string]any{}
	preSkillNameList := map[string]any{}
	gemSpecialMods := map[string]any{}
	// ModParser.lua:6042 — the one literal entry. Sigh.
	skillNameList[" corpse cremation "] = d(p("tag", Tag{"type": "SkillName", "skillName": "Cremation", "includeTransfigured": true}))

	for _, g := range gemSkills {
		skillName := g.name
		// Keys built from names go into regex tables; QuoteMeta keeps a name
		// with regex metacharacters literal (none exist today).
		lower := regexp.QuoteMeta(asciiLower(skillName))
		nameTag := Tag{"type": "SkillName", "skillName": skillName, "includeTransfigured": true}

		skillNameList[" "+lower+" "] = d(p("tag", nameTag))
		preSkillNameList["^"+lower+" "] = d(p("tag", nameTag))
		preSkillNameList["^"+lower+" has ?a? "] = d(p("tag", nameTag))
		preSkillNameList["^"+lower+" deals "] = d(p("tag", nameTag))
		preSkillNameList["^"+lower+" damage "] = d(p("tag", nameTag))
		if g.totem {
			preSkillNameList["^"+lower+" totem deals "] = d(p("tag", nameTag))
			preSkillNameList["^"+lower+" totem grants "] = d(p("addToSkill", nameTag), p("tag", Tag{"type": "GlobalEffect", "effectType": "Buff"}))
		}
		if g.buff {
			preSkillNameList["^"+lower+" grants "] = d(p("addToSkill", nameTag), p("tag", Tag{"type": "GlobalEffect", "effectType": "Buff"}))
			preSkillNameList["^"+lower+" grants a?n? ?additional "] = d(p("addToSkill", nameTag), p("tag", Tag{"type": "GlobalEffect", "effectType": "Buff"}))
		}
		if g.auraOrHerald {
			affected := Tag{"type": "Condition", "var": "AffectedBy" + strings.ReplaceAll(skillName, " ", "")}
			skillNameList["while affected by "+lower] = d(p("tag", affected))
			skillNameList["while using "+lower] = d(p("tag", affected))
		}
		if g.curse {
			skillNameList["if you've cast "+lower+" in the past ([0-9]+) seconds"] = d(p("tag", Tag{"type": "Condition", "var": "SelfCast" + condenseName(skillName)}))
		}
		if g.mine {
			gemSpecialMods["^"+lower+" has ([0-9]+)% increased throwing speed"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("MineLayingSpeed", "INC", c.n(1))}, nameTag)}
			})
		}
		if g.trap {
			gemSpecialMods["([0-9]+)% increased "+lower+" throwing speed"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("TrapThrowingSpeed", "INC", c.n(1))}, nameTag)}
			})
		}
		if g.chaining {
			gemSpecialMods["^"+lower+" chains an additional time"] = []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ChainCountMax", "BASE", 1)}, nameTag)}
			gemSpecialMods["^"+lower+" chains an additional ([0-9]+) times"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ChainCountMax", "BASE", c.n(1))}, nameTag)}
			})
			gemSpecialMods["^"+lower+" chains ([0-9]+) additional times"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ChainCountMax", "BASE", c.n(1))}, nameTag)}
			})
		}
		if g.bow {
			gemSpecialMods["^"+lower+" fires an additional arrow"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", 1, nil, 0, KeywordFlag.Arrow)}, nameTag)}
			})
			gemSpecialMods["^"+lower+" fires ([0-9]+) additional arrows?"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", c.n(1), nil, 0, KeywordFlag.Arrow)}, nameTag)}
			})
		}
		if g.projectile {
			gemSpecialMods["^"+lower+" pierces an additional target"] = []any{mod("PierceCount", "BASE", 1, nameTag)}
			gemSpecialMods["^"+lower+" pierces ([0-9]+) additional targets?"] = fnGem(func(c caps) any {
				return []any{mod("PierceCount", "BASE", c.n(1), nameTag)}
			})
		}
		if g.bow || g.projectile {
			gemSpecialMods["^"+lower+" fires an additional projectile"] = []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", 1)}, nameTag)}
			gemSpecialMods["^"+lower+" fires ([0-9]+) additional projectiles"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", c.n(1))}, nameTag)}
			})
			gemSpecialMods["^"+lower+" fires ([0-9]+) additional shard projectiles"] = fnGem(func(c caps) any {
				return []any{mod("ExtraSkillMod", "LIST", Tag{"mod": mod("ProjectileCount", "BASE", c.n(1))}, nameTag)}
			})
		}
	}
	return skillListsT{skillNames: skillNameList, preSkillNames: preSkillNameList, gemSpecials: gemSpecialMods}
}

// fnGem wraps a gem-loop closure; it exists so the loop above reads like the
// reference while still producing the shared fn type.
func fnGem(f func(c caps) any) fn { return fn(f) }

// keystoneSpecialMods — ModParser.lua:5881-5886.
func keystoneSpecialMods() map[string]any {
	out := map[string]any{}
	for _, name := range keystoneNames {
		out[asciiLower(name)] = []any{
			mod("Keystone", "LIST", name),
			flag("Condition:Have" + condenseName(firstToUpper(name))),
		}
	}
	for _, name := range clusterJewelKeystones {
		out[asciiLower(name)] = []any{mod("Keystone", "LIST", name)}
	}
	return out
}

// clusterJewelSkills — ModParser.lua:6540-6553: whole-line lookups for cluster
// jewel enchants, notables and keystones.
var clusterJewelSkills = buildClusterJewelSkills()

func buildClusterJewelSkills() map[string]any {
	out := map[string]any{}
	for line, skillId := range clusterJewelSkillEnchants {
		out[line] = []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelSkill", "value": skillId})}
	}
	for _, notable := range clusterJewelNotables {
		out["1 added passive skill is "+asciiLower(notable)] = []any{mod("ClusterJewelNotable", "LIST", notable)}
	}
	for _, keystone := range clusterJewelKeystones {
		out["adds "+asciiLower(keystone)] = []any{mod("JewelData", "LIST", Tag{"key": "clusterJewelKeystone", "value": keystone})}
	}
	return out
}
