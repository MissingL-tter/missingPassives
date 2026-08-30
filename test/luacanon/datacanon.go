package luacanon

import (
	"strconv"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

// Plain-table shadows of the data package's structured values, in the shape
// the archive canon (tools/dump_gamedata.lua) uses. The game-data test
// registers them as Encode adapters.

// GrantedEffectTable builds the plain-table shape of a skill
// (statMap._grantedEffect is elided on both sides). A full-custom skill
// carries only the keys its template block wrote.
func GrantedEffectTable(ge *data.GrantedEffect) map[string]any {
	m := map[string]any{
		"name":      ge.Name,
		"id":        ge.Id,
		"modSource": ge.ModSource,
		"statMap":   ge.StatMap,
	}
	if ge.Hidden {
		m["hidden"] = true
	}
	if ge.Description != nil {
		m["description"] = *ge.Description
	}
	if ge.BaseEffectiveness != nil {
		m["baseEffectiveness"] = *ge.BaseEffectiveness
	}
	if ge.FullCustom {
		if ge.Color != 0 {
			m["color"] = ge.Color
		}
		if ge.StatDescriptionScope != "" {
			m["statDescriptionScope"] = ge.StatDescriptionScope
		}
		if ge.SkillTypes != nil {
			m["skillTypes"] = ge.SkillTypes
		}
		if ge.CastTime != nil {
			m["castTime"] = *ge.CastTime
		}
	} else {
		m["color"] = ge.Color
		m["statDescriptionScope"] = ge.StatDescriptionScope
		if ge.BaseTypeName != nil {
			m["baseTypeName"] = *ge.BaseTypeName
		}
		if ge.FlavourText != nil {
			m["flavourText"] = ge.FlavourText
		}
		if ge.IncrementalEffectiveness != nil {
			m["incrementalEffectiveness"] = *ge.IncrementalEffectiveness
		}
		if ge.Support {
			m["support"] = true
			m["requireSkillTypes"] = ge.RequireSkillTypes
			m["addSkillTypes"] = ge.AddSkillTypes
			m["excludeSkillTypes"] = ge.ExcludeSkillTypes
			if ge.IsTrigger {
				m["isTrigger"] = true
			}
			if ge.SupportGemsOnly {
				m["supportGemsOnly"] = true
			}
			if ge.IgnoreMinionTypes {
				m["ignoreMinionTypes"] = true
			}
			if ge.PlusVersionOf != nil {
				m["plusVersionOf"] = *ge.PlusVersionOf
			}
		} else {
			m["skillTypes"] = ge.SkillTypes
			if ge.MinionSkillTypes != nil {
				m["minionSkillTypes"] = ge.MinionSkillTypes
			}
			if ge.SkillTotemId != nil {
				m["skillTotemId"] = *ge.SkillTotemId
			}
			if ge.CastTime != nil {
				m["castTime"] = *ge.CastTime
			}
			if ge.CannotBeSupported {
				m["cannotBeSupported"] = true
			}
		}
		if ge.WeaponTypes != nil {
			m["weaponTypes"] = ge.WeaponTypes
		}
	}
	if ge.BaseFlags != nil {
		m["baseFlags"] = ge.BaseFlags
	}
	if ge.BaseMods != nil {
		m["baseMods"] = ge.BaseMods
	}
	if ge.LevelMods != nil {
		m["levelMods"] = ge.LevelMods
	}
	if ge.QualityStats != nil {
		m["qualityStats"] = statValueTables(ge.QualityStats)
	}
	if ge.ConstantStats != nil {
		m["constantStats"] = statValueTables(ge.ConstantStats)
	}
	if ge.Stats != nil {
		m["stats"] = ge.Stats
		if len(ge.NotMinionStat) > 0 {
			m["notMinionStat"] = ge.NotMinionStat
		}
	}
	if ge.Levels != nil {
		m["levels"] = ge.Levels
	}
	if ge.HasGlobalEffect {
		m["hasGlobalEffect"] = true
	}
	skillCustomInto(m, ge.Custom)
	return m
}

// statValueTables renders {id, value} pairs as the reference's 2-tuples.
func statValueTables(list []schema.StatValue) [][]any {
	out := make([][]any, len(list))
	for i, s := range list {
		out[i] = []any{s.Id, s.Value}
	}
	return out
}

// skillCustomInto merges the template's custom keys into the skill table.
func skillCustomInto(m map[string]any, c data.SkillCustom) {
	for k, set := range map[string]bool{"fromItem": c.FromItem, "fromTree": c.FromTree, "legacy": c.Legacy, "minionHasItemSet": c.MinionHasItemSet, "hideFromGemList": c.HideFromGemList} {
		if set {
			m[k] = true
		}
	}
	if c.Parts != nil {
		parts := make([]map[string]any, len(c.Parts))
		for i, p := range c.Parts {
			pm := map[string]any{"name": p.Name}
			for k, v := range p.Flags {
				pm[k] = v
			}
			parts[i] = pm
		}
		m["parts"] = parts
	}
	if c.MinionList != nil {
		m["minionList"] = c.MinionList
	}
	if c.AddMinionList != nil {
		m["addMinionList"] = c.AddMinionList
	}
	if c.AddFlags != nil {
		m["addFlags"] = c.AddFlags
	}
	if c.MinionUses != nil {
		m["minionUses"] = c.MinionUses
	}
	for kind := range c.Callbacks {
		m[kind.String()] = Fn{}
	}
}

// StatMapEntryTable merges mods and scaling keys into one table shadow.
func StatMapEntryTable(e *data.StatMapEntry) map[string]any {
	m := map[string]any{}
	for i, mod := range e.Mods {
		m[strconv.Itoa(i+1)] = mod
	}
	optInto(m, "div", e.Div)
	optInto(m, "mult", e.Mult)
	optInto(m, "base", e.Base)
	optInto(m, "value", e.Value)
	if e.SkillFlag != "" {
		m["skillFlag"] = e.SkillFlag
	}
	return m
}

func optInto(m map[string]any, key string, v util.Opt[float64]) {
	if v.Set {
		m[key] = v.V
	}
}

// SkillModTable renders one statMap element: the mod itself, a group's
// members plus scale/source/tags, or a typo record's raw table.
func SkillModTable(m data.SkillMod) any {
	switch {
	case m.Mod != nil:
		return m.Mod
	case m.Group != nil:
		g := m.Group
		t := map[string]any{}
		for i, mod := range g.Mods {
			t[strconv.Itoa(i+1)] = mod
		}
		for i, tag := range g.Tags {
			t[strconv.Itoa(len(g.Mods)+i+1)] = tag
		}
		if g.Div.Set {
			t["div"] = g.Div.V
		}
		if g.Mult.Set {
			t["mult"] = g.Mult.V
		}
		if g.Source != "" {
			t["source"] = g.Source
		}
		return t
	case m.Typo != nil:
		t := m.Typo
		r := map[string]any{"name": t.Name, "keywordFlags": t.KeywordFlags}
		if t.Type != "" {
			r["type"] = t.Type
		}
		if t.Value != nil {
			r["value"] = t.Value
		}
		if t.FlagsTag != nil {
			r["flags"] = t.FlagsTag
		} else {
			r["flags"] = t.Flags
		}
		for i, n := range t.StrayNums {
			r[strconv.Itoa(i+1)] = n
		}
		for i, tag := range t.StrayTags {
			r[strconv.Itoa(len(t.StrayNums)+i+1)] = tag
		}
		if t.Source != "" {
			r["source"] = t.Source
		}
		return r
	}
	return nil
}

// SkillLevelTable merges a level's values, extras, interpolation and cost.
func SkillLevelTable(l *data.SkillLevel) map[string]any {
	m := map[string]any{}
	for i, v := range l.Values {
		m[strconv.Itoa(i+1)] = v
	}
	for k, v := range l.Extra {
		m[k] = v
	}
	if len(l.StatInterpolation) > 0 {
		m["statInterpolation"] = l.StatInterpolation
	}
	if l.Cost != nil {
		m["cost"] = l.Cost
	}
	return m
}

// GemTable builds a gem's plain-table shadow; granted effects appear as
// their skill ids (the dump normalises the same way to avoid inlining
// entire skills).
func GemTable(g *data.Gem) map[string]any {
	m := map[string]any{
		"id":              g.Id,
		"name":            g.Name,
		"gameId":          g.GameId,
		"variantId":       g.VariantId,
		"grantedEffectId": g.GrantedEffectId,
		"tags":            g.Tags,
		"tagString":       g.TagString,
		"reqStr":          g.ReqStr,
		"reqDex":          g.ReqDex,
		"reqInt":          g.ReqInt,
		"naturalMaxLevel": g.NaturalMaxLevel,
	}
	if g.BaseTypeName != nil {
		m["baseTypeName"] = *g.BaseTypeName
	}
	if g.SecondaryGrantedEffectId != nil {
		m["secondaryGrantedEffectId"] = *g.SecondaryGrantedEffectId
	}
	if g.SecondaryEffectName != nil {
		m["secondaryEffectName"] = *g.SecondaryEffectName
	}
	if g.VaalGem {
		m["vaalGem"] = true
	}
	if g.GrantedEffect != nil {
		m["grantedEffect"] = "\x1bskill:" + g.GrantedEffect.Id
	}
	if g.SecondaryGrantedEffect != nil {
		m["secondaryGrantedEffect"] = "\x1bskill:" + g.SecondaryGrantedEffect.Id
	}
	list := map[string]any{}
	for i, ge := range g.GrantedEffectList {
		if ge != nil {
			list[strconv.Itoa(i+1)] = "\x1bskill:" + ge.Id
		}
	}
	m["grantedEffectList"] = list
	return m
}

// GemForSkillTable keys data.GemForSkill by skill id.
func GemForSkillTable() map[string]string {
	out := map[string]string{}
	for ge, gemId := range data.GemForSkill {
		out[ge.Id] = gemId
	}
	return out
}

// NodeIDListTable rebuilds the reference's nodeIDList: node rows keyed by
// graph id beside the localIdToGlobalId/size/sizeNotable metadata.
func NodeIDListTable() map[string]any {
	m := map[string]any{
		"size":        data.NodeIDListSize,
		"sizeNotable": data.NodeIDListSizeNotable,
	}
	for id, e := range data.NodeIDList {
		m[strconv.FormatInt(id, 10)] = map[string]any{"index": e.Index, "size": e.Size}
	}
	l2g := make([]map[string]any, len(data.LocalIDToGlobalID))
	for i, t := range data.LocalIDToGlobalID {
		row := map[string]any{"size": t.Size}
		for local, global := range t.Global {
			row[strconv.Itoa(local)] = global
		}
		l2g[i] = row
	}
	m["localIdToGlobalId"] = l2g
	return m
}

// TradeIDsTable renders one jewel type's trade ids.
func TradeIDsTable(t data.TradeIDs) map[string]any {
	m := map[string]any{}
	if t.Keystone != nil {
		m["keystone"] = t.Keystone
	}
	if t.Devotion != nil {
		m["devotion"] = t.Devotion
	}
	return m
}

// MapModTable renders one affix record (an empty record is an empty table).
func MapModTable(m *data.MapMod) map[string]any {
	t := map[string]any{}
	if m.Label != "" {
		t["label"] = m.Label
	}
	if m.Tooltip != "" {
		t["tooltip"] = m.Tooltip
	}
	if m.TooltipLines != nil {
		t["tooltipLines"] = m.TooltipLines
	}
	if m.Type != "" {
		t["type"] = m.Type
	}
	if m.Values != nil {
		t["values"] = m.Values
	}
	if m.Apply != data.MapModApplyNone {
		t["apply"] = Fn{}
	}
	return t
}

// MapModValueTable renders a per-tier value: a number, or its list.
func MapModValueTable(v data.MapModValue) any {
	if v.List == nil {
		return v.Num
	}
	return v.List
}

// MapModsTable renders data.mapMods.
func MapModsTable(mm data.MapModData) map[string]any {
	return map[string]any{"AffixData": mm.AffixData, "Prefix": mm.Prefix, "Suffix": mm.Suffix}
}

// PowerStatTable renders a powerStatList entry; the transform is a function.
func PowerStatTable(p data.PowerStat) map[string]any {
	m := map[string]any{"label": p.Label}
	if p.Stat != nil {
		m["stat"] = *p.Stat
	}
	if p.ItemField != nil {
		m["itemField"] = *p.ItemField
	}
	for k, set := range map[string]bool{"combinedOffDef": p.CombinedOffDef, "ignoreForItems": p.IgnoreForItems, "ignoreForNodes": p.IgnoreForNodes, "reverseSort": p.ReverseSort} {
		if set {
			m[k] = true
		}
	}
	if p.Transform != data.TransformNone {
		m["transform"] = Fn{}
	}
	return m
}

// BossPenTable renders a penetration set: numbers, or the "" placeholder.
func BossPenTable(pens map[string]util.Opt[float64]) map[string]any {
	m := map[string]any{}
	for k, v := range pens {
		if v.Set {
			m[k] = v.V
		} else {
			m[k] = ""
		}
	}
	return m
}

// BossSkillTable renders a boss skill; the penetration sets go through
// BossPenTable (the struct's lua tags cover the other keys).
func BossSkillTable(s data.BossSkillData) map[string]any {
	m := map[string]any{
		"DamageType":        s.DamageType,
		"DamageMultipliers": s.DamageMultipliers,
		"tooltip":           s.Tooltip,
	}
	if s.UberDamageMultiplier != nil {
		m["UberDamageMultiplier"] = *s.UberDamageMultiplier
	}
	if s.DamagePenetrations != nil {
		m["DamagePenetrations"] = BossPenTable(s.DamagePenetrations)
	}
	if s.UberDamagePenetrations != nil {
		m["UberDamagePenetrations"] = BossPenTable(s.UberDamagePenetrations)
	}
	if s.Speed != nil {
		m["speed"] = *s.Speed
	}
	if s.UberSpeed != nil {
		m["UberSpeed"] = *s.UberSpeed
	}
	if s.CritChance != nil {
		m["critChance"] = *s.CritChance
	}
	if s.EarlierUber {
		m["earlierUber"] = true
	}
	if s.AdditionalStats != nil {
		m["additionalStats"] = s.AdditionalStats
	}
	return m
}

// BossStatTable renders one additional stat: a number or "flag".
func BossStatTable(s data.BossStat) any {
	if s.Flag {
		return "flag"
	}
	return s.Value
}
