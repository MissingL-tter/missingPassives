// Renders gamedata.SkillsData as the Data/Skills/<name>.lua files and
// Data/Gems.lua (Scripts/skills.lua's outputs).

package luarender

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() { register("skills", renderSkills) }

var skillTemplateFiles = []string{"act_str", "act_dex", "act_int", "other", "glove", "minion", "spectre", "sup_str", "sup_dex", "sup_int"}

// num renders a float the way luaStrAny rendered the mixed int/float cells.
func renderSkillHeader(b *B, h gamedata.SkillHeader) {
	b.W("skills[\"", h.GrantedId, "\"] = {\n")
	if h.Invalid {
		return
	}
	b.W("\tname = \"", h.Name, "\",\n")
	if h.Hidden {
		b.W("\thidden = true,\n")
	} else if h.Support {
		if h.Description != nil {
			b.W("\tdescription = \"", *h.Description, "\",\n")
		}
	} else if h.BaseTypeName != nil {
		b.W("\tbaseTypeName = \"", *h.BaseTypeName, "\",\n")
	}
	if h.HasFlavour {
		b.W("\tflavourText = {")
		for _, line := range h.FlavourText {
			b.W("\"", line, "\", ")
		}
		b.W("},\n")
	}
	b.W("\tcolor = ", h.Color, ",\n")
	if h.BaseEffectiveness != nil {
		b.W("\tbaseEffectiveness = ", *h.BaseEffectiveness, ",\n")
	}
	if h.IncrementalEffectiveness != nil {
		b.W("\tincrementalEffectiveness = ", *h.IncrementalEffectiveness, ",\n")
	}
	writeWeaponTypes := func() {
		if len(h.WeaponTypes) > 0 {
			b.W("\tweaponTypes = {\n")
			for _, typ := range h.WeaponTypes {
				b.W("\t\t[\"", typ, "\"] = true,\n")
			}
			b.W("\t},\n")
		}
	}
	if h.Support {
		b.W("\tsupport = true,\n")
		b.W("\trequireSkillTypes = { ")
		for _, typ := range h.RequireSkillTypes {
			b.W(typ, ", ")
		}
		b.W("},\n")
		b.W("\taddSkillTypes = { ")
		for _, typ := range h.AddSkillTypes {
			b.W(typ, ", ")
		}
		b.W("},\n")
		b.W("\texcludeSkillTypes = { ")
		for _, typ := range h.ExcludeSkillTypes {
			b.W(typ, ", ")
		}
		b.W("},\n")
		if h.IsTrigger {
			b.W("\tisTrigger = true,\n")
		}
		if h.SupportGemsOnly {
			b.W("\tsupportGemsOnly = true,\n")
		}
		if h.IgnoreMinionTypes {
			b.W("\tignoreMinionTypes = true,\n")
		}
		if h.PlusVersionOf != nil {
			b.W("\tplusVersionOf = \"", *h.PlusVersionOf, "\",\n")
		}
		writeWeaponTypes()
		b.W("\tstatDescriptionScope = \"gem_stat_descriptions\",\n")
	} else {
		if h.Description != nil {
			b.W("\tdescription = \"", *h.Description, "\",\n")
		}
		b.W("\tskillTypes = { ")
		for _, typ := range h.SkillTypes {
			b.W("[", typ, "] = true, ")
		}
		b.W("},\n")
		if len(h.MinionSkillTypes) > 0 {
			b.W("\tminionSkillTypes = { ")
			for _, typ := range h.MinionSkillTypes {
				b.W("[", typ, "] = true, ")
			}
			b.W("},\n")
		}
		writeWeaponTypes()
		b.W("\tstatDescriptionScope = \"", h.StatDescriptionScope, "\",\n")
		if h.SkillTotemId != nil {
			b.W("\tskillTotemId = ", *h.SkillTotemId, ",\n")
		}
		b.W("\tcastTime = ", *h.CastTime, ",\n")
		if h.CannotBeSupported {
			b.W("\tcannotBeSupported = true,\n")
		}
	}
}

func renderSkillTail(b *B, t gamedata.SkillTail, args string) {
	if !strings.Contains(args, "noBaseFlags") && !t.Support {
		b.W("\tbaseFlags = {\n")
		for _, flag := range t.BaseFlags {
			b.W("\t\t", flag, " = true,\n")
		}
		b.W("\t},\n")
	}
	if !strings.Contains(args, "noBaseMods") && len(t.BaseMods) > 0 {
		b.W("\tbaseMods = {\n")
		for _, mod := range t.BaseMods {
			b.W("\t\t", mod, ",\n")
		}
		b.W("\t},\n")
	}
	if !strings.Contains(args, "noQualityStats") && len(t.QualityStats) > 0 {
		b.W("\tqualityStats = {\n")
		for _, stat := range t.QualityStats {
			b.W("\t\t{ \"", stat.Id, "\", ", stat.Value, " },\n")
		}
		b.W("\t},\n")
	}
	if !strings.Contains(args, "noStats") {
		if len(t.ConstantStats) > 0 {
			b.W("\tconstantStats = {\n")
			for _, stat := range t.ConstantStats {
				b.W("\t\t{ \"", stat.Id, "\", ", stat.Value, " },\n")
			}
			b.W("\t},\n")
		}
		b.W("\tstats = {\n")
		for _, stat := range t.Stats {
			b.W("\t\t\"", stat, "\",\n")
		}
		b.W("\t},\n")
		if len(t.NotMinionStat) > 0 {
			b.W("\tnotMinionStat = {\n")
			for _, stat := range t.NotMinionStat {
				b.W("\t\t\"", stat, "\",\n")
			}
			b.W("\t},\n")
		}
	}
	if !strings.Contains(args, "noLevels") {
		b.W("\tlevels = {\n")
		for _, level := range t.Levels {
			b.W("\t\t[", level.Level, "] = { ")
			for _, v := range level.Values {
				b.W(v, ", ")
			}
			extraKeys := make([]string, 0, len(level.Extra))
			for k := range level.Extra {
				extraKeys = append(extraKeys, k)
			}
			sort.Strings(extraKeys)
			for _, k := range extraKeys {
				b.W(k, " = ", level.Extra[k], ", ")
			}
			if len(level.Interp) > 0 {
				b.W("statInterpolation = { ")
				for _, t := range level.Interp {
					b.W(t, ", ")
				}
				b.W("}, ")
			}
			if len(level.Cost) > 0 {
				b.W("cost = { ")
				costKeys := make([]string, 0, len(level.Cost))
				for k := range level.Cost {
					costKeys = append(costKeys, k)
				}
				sort.Strings(costKeys)
				for _, k := range costKeys {
					b.W(k, " = ", level.Cost[k], ", ")
				}
				b.W("}, ")
			}
			b.W("},\n")
		}
		b.W("\t},\n")
	}
	b.W("}")
}

func renderSkills(d gamedata.SkillsData, tpl Templates) (map[string]string, error) {
	files := map[string]string{}
	for _, name := range skillTemplateFiles {
		f := d.Files[name]
		nextSkill, nextTail := 0, 0
		directives := map[string]func(string, *B){
			"skill": func(_ string, b *B) {
				if nextSkill >= len(f.Skills) {
					panic(fmt.Sprintf("skills: template %s has more #skill directives than data", name))
				}
				renderSkillHeader(b, f.Skills[nextSkill])
				nextSkill++
			},
			"mods": func(args string, b *B) {
				if nextTail >= len(f.Tails) {
					panic(fmt.Sprintf("skills: template %s has more #mods directives than data", name))
				}
				renderSkillTail(b, f.Tails[nextTail], args)
				nextTail++
			},
		}
		var b B
		if err := processTemplate(tpl, name, "Skills/", &b, directives); err != nil {
			return nil, err
		}
		files["Data/Skills/"+name+".lua"] = b.String()
	}

	var gb B
	gb.W("-- This file is automatically generated, do not edit!\n")
	gb.W("-- Gem data (c) Grinding Gear Games\n\nreturn {\n")
	for _, g := range d.Gems {
		gb.W("\t[\"", "Metadata/Items/Gems/SkillGem"+g.VariantId, "\"] = {\n")
		gb.W("\t\tname = \"", g.Name, "\",\n")
		if g.BaseTypeName != nil {
			gb.W("\t\tbaseTypeName = \"", *g.BaseTypeName, "\",\n")
		}
		gb.W("\t\tgameId = \"", g.GameId, "\",\n")
		gb.W("\t\tvariantId = \"", g.VariantId, "\",\n")
		gb.W("\t\tgrantedEffectId = \"", g.GrantedEffectId, "\",\n")
		if g.SecondaryGrantedEffectId != nil {
			gb.W("\t\tsecondaryGrantedEffectId = \"", *g.SecondaryGrantedEffectId, "\",\n")
		}
		if g.SecondaryEffectName != nil {
			gb.W("\t\tsecondaryEffectName = \"", *g.SecondaryEffectName, "\",\n")
		}
		if g.VaalGem {
			gb.W("\t\tvaalGem = true,\n")
		}
		gb.W("\t\ttags = {\n")
		for _, tag := range g.Tags {
			gb.W("\t\t\t", tag, " = true,\n")
		}
		gb.W("\t\t},\n")
		gb.W("\t\ttagString = \"", g.TagString, "\",\n")
		gb.W("\t\treqStr = ", g.ReqStr, ",\n")
		gb.W("\t\treqDex = ", g.ReqDex, ",\n")
		gb.W("\t\treqInt = ", g.ReqInt, ",\n")
		gb.W("\t\tnaturalMaxLevel = ", g.NaturalMaxLevel, ",\n")
		gb.W("\t},\n")
	}
	gb.W("}")
	files["Data/Gems.lua"] = gb.String()
	return files, nil
}
