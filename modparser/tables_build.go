package modparser

import (
	"regexp"
	"strings"
)

// Construction of the data-driven tables — ModParser.lua:6040-6094 (skill name
// lookups plus per-skill specialModList entries), 5881-5886 (keystones) and
// 6540-6553 (cluster jewel skills). The loops are ported as written; the data
// they iterate comes from vocab.go.

// skillLists bundles the three tables the gem loop fills, so plain var
// initialisers (which Go orders by dependency, unlike init functions) build
// them before the scan tables that read them.
type skillListsT struct {
	skillNames, preSkillNames map[string]*PatternEntry
	gemSpecials               map[string]modsValue
}

var skillLists = buildSkillLists()

var skillNameList = skillLists.skillNames
var preSkillNameList = skillLists.preSkillNames

// gemSpecialMods holds the specialModList entries the gem loop generates; they
// are merged with the literal entries when the scan table is built.
var gemSpecialMods = skillLists.gemSpecials

func buildSkillLists() skillListsT {
	skillNameList := map[string]*PatternEntry{}
	preSkillNameList := map[string]*PatternEntry{}
	gemSpecialMods := map[string]modsValue{}
	// ModParser.lua:6042 — the one literal entry. Sigh.
	skillNameList[" corpse cremation "] = &PatternEntry{Tag: &SkillNameTag{SkillName: "Cremation", IncludeTransfigured: true}}

	for _, g := range gemSkills {
		skillName := g.name
		// Keys built from names go into regex tables; QuoteMeta keeps a name
		// with regex metacharacters literal (none exist today).
		lower := regexp.QuoteMeta(strings.ToLower(skillName))
		// One shared tag table per skill, as the reference's local nameTag.
		nameTag := &SkillNameTag{SkillName: skillName, IncludeTransfigured: true}
		buffEffect := func() *PatternEntry {
			return &PatternEntry{AddToSkill: nameTag, Tag: &GlobalEffectTag{EffectType: "Buff"}}
		}

		skillNameList[" "+lower+" "] = &PatternEntry{Tag: nameTag}
		preSkillNameList["^"+lower+" "] = &PatternEntry{Tag: nameTag}
		preSkillNameList["^"+lower+" has ?a? "] = &PatternEntry{Tag: nameTag}
		preSkillNameList["^"+lower+" deals "] = &PatternEntry{Tag: nameTag}
		preSkillNameList["^"+lower+" damage "] = &PatternEntry{Tag: nameTag}
		if g.totem {
			preSkillNameList["^"+lower+" totem deals "] = &PatternEntry{Tag: nameTag}
			preSkillNameList["^"+lower+" totem grants "] = buffEffect()
		}
		if g.buff {
			preSkillNameList["^"+lower+" grants "] = buffEffect()
			preSkillNameList["^"+lower+" grants a?n? ?additional "] = buffEffect()
		}
		if g.auraOrHerald {
			affected := &CondTag{Var: "AffectedBy" + strings.ReplaceAll(skillName, " ", "")}
			skillNameList["while affected by "+lower] = &PatternEntry{Tag: affected}
			skillNameList["while using "+lower] = &PatternEntry{Tag: affected}
		}
		if g.curse {
			skillNameList["if you've cast "+lower+" in the past ([0-9]+) seconds"] = &PatternEntry{Tag: &CondTag{Var: "SelfCast" + condenseName(skillName)}}
		}
		if g.mine {
			gemSpecialMods["^"+lower+" has ([0-9]+)% increased throwing speed"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("MineLayingSpeed", Inc, c.v(1))}, nameTag)}
			})
		}
		if g.trap {
			gemSpecialMods["([0-9]+)% increased "+lower+" throwing speed"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("TrapThrowingSpeed", Inc, c.v(1))}, nameTag)}
			})
		}
		if g.chaining {
			gemSpecialMods["^"+lower+" chains an additional time"] = modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("ChainCountMax", Base, Num(1))}, nameTag)}
			gemSpecialMods["^"+lower+" chains an additional ([0-9]+) times"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ChainCountMax", Base, c.v(1))}, nameTag)}
			})
			gemSpecialMods["^"+lower+" chains ([0-9]+) additional times"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ChainCountMax", Base, c.v(1))}, nameTag)}
			})
		}
		if g.bow {
			gemSpecialMods["^"+lower+" fires an additional arrow"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: modf("ProjectileCount", Base, Num(1), FlagNone, KeywordArrow)}, nameTag)}
			})
			gemSpecialMods["^"+lower+" fires ([0-9]+) additional arrows?"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: modf("ProjectileCount", Base, c.v(1), FlagNone, KeywordArrow)}, nameTag)}
			})
		}
		if g.projectile {
			gemSpecialMods["^"+lower+" pierces an additional target"] = modList{mod("PierceCount", Base, Num(1), nameTag)}
			gemSpecialMods["^"+lower+" pierces ([0-9]+) additional targets?"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("PierceCount", Base, c.v(1), nameTag)}
			})
		}
		if g.bow || g.projectile {
			gemSpecialMods["^"+lower+" fires an additional projectile"] = modList{mod("ExtraSkillMod", List, ModRef{Mod: mod("ProjectileCount", Base, Num(1))}, nameTag)}
			gemSpecialMods["^"+lower+" fires ([0-9]+) additional projectiles"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ProjectileCount", Base, c.v(1))}, nameTag)}
			})
			gemSpecialMods["^"+lower+" fires ([0-9]+) additional shard projectiles"] = modFn(func(c caps) []*Mod {
				return []*Mod{mod("ExtraSkillMod", List, ModRef{Mod: mod("ProjectileCount", Base, c.v(1))}, nameTag)}
			})
		}
	}
	return skillListsT{skillNames: skillNameList, preSkillNames: preSkillNameList, gemSpecials: gemSpecialMods}
}

// keystoneSpecialMods — ModParser.lua:5881-5886.
func keystoneSpecialMods() map[string]modsValue {
	out := map[string]modsValue{}
	for _, name := range keystoneNames {
		out[strings.ToLower(name)] = modList{
			mod("Keystone", List, Str(name)),
			flag("Condition:Have" + condenseName(firstToUpper(name))),
		}
	}
	for _, name := range clusterJewelKeystones {
		out[strings.ToLower(name)] = modList{mod("Keystone", List, Str(name))}
	}
	return out
}

// clusterJewelSkills — ModParser.lua:6540-6553: whole-line lookups for cluster
// jewel enchants, notables and keystones.
var clusterJewelSkills = buildClusterJewelSkills()

func buildClusterJewelSkills() map[string][]*Mod {
	out := map[string][]*Mod{}
	for line, skillId := range clusterJewelSkillEnchants {
		out[line] = []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelSkill", Value: Str(skillId)})}
	}
	for _, notable := range clusterJewelNotables {
		out["1 added passive skill is "+strings.ToLower(notable)] = []*Mod{mod("ClusterJewelNotable", List, Str(notable))}
	}
	for _, keystone := range clusterJewelKeystones {
		out["adds "+strings.ToLower(keystone)] = []*Mod{mod("JewelData", List, DataRef{Key: "clusterJewelKeystone", Value: Str(keystone)})}
	}
	return out
}
