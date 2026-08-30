// Renders schema.BossData as Data/BossSkills.lua and Data/Bosses.lua
// (Scripts/bossData.lua's outputs).

package luarender

import (
	"sort"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() { register("bossData", renderBossData) }

// penText spells one penetration value the way the reference exporter did:
// the number, or a Lua empty-string literal for the blank placeholder.
func penText(p schema.PenEntry) string {
	if p.Value == nil {
		return "\"\""
	}
	return luaNum(*p.Value)
}

func renderBossData(d schema.BossData, tpl Templates) (map[string]string, error) {
	nextSkill, nextList := 0, 0
	// Numbers spell as %.14g, flags as the quoted "flag" literal.
	writeStatSet := func(b *B, vals map[string]schema.BossStatValue) {
		keys := make([]string, 0, len(vals))
		for k := range vals {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				b.W(",")
			}
			b.W("\n\t\t\t\t", k, " = ")
			if v := vals[k]; v.Flag {
				b.W("\"flag\"")
			} else {
				b.W(v.Value)
			}
		}
	}
	skillsDirectives := map[string]func(string, *B){
		"skill": func(_ string, b *B) {
			if nextSkill >= len(d.Skills) {
				panic("bossData: more #skill directives than data")
			}
			bs := d.Skills[nextSkill]
			nextSkill++
			b.W("	[\"", bs.Key, "\"] = {\n")
			b.W("		DamageType = \"", bs.DamageType, "\",\n")
			b.W("		DamageMultipliers = {\n")
			for i, dm := range bs.DamageMultipliers {
				if i > 0 {
					b.W(",\n")
				}
				b.W("			", dm.Type, " = { ", dm.Min, ", ", dm.Spread, " }")
			}
			b.W("\n		}")
			if bs.UberDamageMultiplier != nil {
				b.W(",\n		UberDamageMultiplier = ", *bs.UberDamageMultiplier)
			}
			if bs.HasPen {
				b.W(",\n		DamagePenetrations = {\n")
				for i, p := range bs.Pens {
					if i > 0 {
						b.W(",\n")
					}
					b.W("			", p.Name, " = ", penText(p))
				}
				b.W("\n		}")
				if bs.HasUberPen {
					b.W(",\n		UberDamagePenetrations = {\n")
					for i, p := range bs.UberPens {
						if i > 0 {
							b.W(",\n")
						}
						b.W("			", p.Name, " = ", penText(p))
					}
					b.W("\n		}")
				}
			}
			if bs.Speed != 700 {
				b.W(",\n		speed = ", bs.Speed)
			}
			if bs.UberSpeed != nil && *bs.UberSpeed != 700 {
				b.W(",\n		UberSpeed = ", *bs.UberSpeed)
			}
			if bs.CritChance != 5 {
				b.W(",\n		critChance = ", bs.CritChance)
			}
			if bs.EarlierUber {
				b.W(",\n		earlierUber = true")
			}
			if bs.HasAdditional {
				b.W(",\n		additionalStats = {")
				if bs.BaseCount > 0 {
					b.W("\n			base = {")
					writeStatSet(b, bs.BaseVals)
					b.W("\n			}")
				}
				if bs.UberCount > 0 {
					if bs.BaseCount > 0 {
						b.W(",")
					}
					b.W("\n			uber = {")
					writeStatSet(b, bs.UberVals)
					b.W("\n			}")
				}
				b.W("\n		}")
			}
		},
		"tooltip": func(_ string, b *B) {
			b.W(",\n		tooltip = ", "\"", luaEsc(d.Skills[nextSkill-1].Tooltip), "\"\n")
			b.W("	},\n")
		},
		"skillList": func(_ string, b *B) {
			if nextList >= len(d.SkillLists) {
				panic("bossData: more #skillList directives than data")
			}
			b.W("},{\n")
			b.W("    { val = \"None\", label = \"None\" }")
			for _, skillName := range d.SkillLists[nextList] {
				b.W(",\n    { val = \"", skillName, "\", label = \"", skillName, "\" }")
			}
			b.W("\n}")
			nextList++
		},
	}
	var sb B
	if err := processTemplate(tpl, "BossSkills", "Enemies/", &sb, skillsDirectives); err != nil {
		return nil, err
	}

	nextBoss := 0
	monstersDirectives := map[string]func(string, *B){
		"boss": func(_ string, b *B) {
			if nextBoss >= len(d.Bosses) {
				panic("bossData: more #boss directives than data")
			}
			bm := d.Bosses[nextBoss]
			nextBoss++
			if bm == nil {
				return
			}
			b.W("bosses[\"", bm.DisplayName, "\"] = {\n")
			b.W("\tarmourMult = ", bm.ArmourMult, ",\n")
			b.W("\tevasionMult = ", bm.EvasionMult, ",\n")
			if bm.IsUber {
				b.W("\tisUber = true,\n")
			} else {
				b.W("\tisUber = false,\n")
			}
			b.W("}\n")
		},
	}
	var mb B
	if err := processTemplate(tpl, "Bosses", "Enemies/", &mb, monstersDirectives); err != nil {
		return nil, err
	}
	return map[string]string{
		"Data/BossSkills.lua": sb.String(),
		"Data/Bosses.lua":     mb.String(),
	}, nil
}
