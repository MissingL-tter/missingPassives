// Port of .archive/src/Export/Scripts/statdesc.lua: exports each stat
// description file as Data/StatDescriptions/<name>.lua.
//
// This script has its own parser, subtly different from statdesc.lua's:
// includes record a parent instead of recursing, `lang` blocks are dropped
// entirely, descriptors carry no order, and unparsed stat lines are prepended
// to the next line.

package export

import (
	"regexp"
	"strconv"
	"strings"
)

func init() {
	var outs []string
	for _, name := range statFileList {
		outs = append(outs, "Data/StatDescriptions/"+name+".lua")
	}
	Scripts = append(Scripts, Script{Name: "statdesc", Outs: outs, Run: scriptStatdesc})
}

var statFileList = []string{
	"active_skill_gem_stat_descriptions",
	"aura_skill_stat_descriptions",
	"banner_aura_skill_stat_descriptions",
	"beam_skill_stat_descriptions",
	"brand_skill_stat_descriptions",
	"curse_skill_stat_descriptions",
	"debuff_skill_stat_descriptions",
	"secondary_debuff_skill_stat_descriptions",
	"gem_stat_descriptions",
	"minion_attack_skill_stat_descriptions",
	"minion_skill_stat_descriptions",
	"minion_spell_skill_stat_descriptions",
	"minion_spell_damage_skill_stat_descriptions",
	"single_minion_spell_skill_stat_descriptions",
	"monster_stat_descriptions",
	"offering_skill_stat_descriptions",
	"skill_stat_descriptions",
	"stat_descriptions",
	"variable_duration_skill_stat_descriptions",
	"buff_skill_stat_descriptions",
	"tincture_stat_descriptions",
	"graft_stat_descriptions",
}

var reParentInclude = regexp.MustCompile(`include "Metadata/StatDescriptions/(.+)\.txt"$`)

// limitEntryTable builds the {[1]=,[2]=} limit table with number or string
// values as the Lua stores them.
func limitEntry(a, b any) luaTable {
	t := luaTable{}
	if a != nil {
		t[1] = a
	}
	if b != nil {
		t[2] = b
	}
	return t
}

func numOrStr(s string) any {
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n
	}
	return s
}

func scriptStatdesc(x *Ctx) error {
	for _, name := range statFileList {
		if err := processStatFile(x, name); err != nil {
			return err
		}
	}
	return nil
}

func processStatFile(x *Ctx, name string) error {
	statDescriptor := luaTable{}
	arrayLen := 0
	var curLang *luaTable // nil while in a non-English lang block
	var curLangLen int
	curDescriptor := luaTable{}
	prepend := ""

	processLine := func(line string) {
		line = prepend + line
		prepend = ""
		if m := reParentInclude.FindStringSubmatch(line); m != nil {
			statDescriptor["parent"] = m[1]
			return
		}
		if m := reNoDesc.FindStringSubmatch(line); m != nil {
			arrayLen++
			statDescriptor[arrayLen] = luaTable{"stats": luaTable{1: m[1]}}
			statDescriptor[m[1]] = float64(arrayLen)
			return
		}
		if strings.Contains(line, "handed_description") ||
			(strings.Contains(line, "description") && !strings.Contains(line, "_description")) {
			lang := luaTable{}
			curLang = &lang
			curLangLen = 0
			curDescriptor = luaTable{1: lang}
			if m := reDescName.FindStringSubmatch(line); m != nil {
				curDescriptor["name"] = m[1]
			}
			arrayLen++
			statDescriptor[arrayLen] = curDescriptor
			return
		}
		if curDescriptor["stats"] == nil {
			if m := reStatsLine.FindStringSubmatch(line); m != nil {
				stats := luaTable{}
				si := 0
				for _, stat := range reStatWord.FindAllString(m[1], -1) {
					si++
					stats[si] = stat
					statDescriptor[stat] = float64(arrayLen)
				}
				curDescriptor["stats"] = stats
			} else {
				// Try to combine it with the next line.
				prepend = line
			}
			return
		}
		if reLangLine.MatchString(line) {
			curLang = nil
			return
		}
		if curLang == nil || strings.Contains(line, "table_only") {
			return
		}
		m := reDescLine.FindStringSubmatch(line)
		if m == nil {
			return
		}
		statLimits, quality, text, special := m[1], m[2], m[3], m[4]
		desc := luaTable{"text": escapeGGGString(text), "limit": luaTable{}}
		descLen := 0
		limits := desc["limit"].(luaTable)
		limitLen := 0
		for _, statLimit := range reLimitTok.FindAllString(statLimits, -1) {
			var limit luaTable
			if statLimit == "#" {
				limit = limitEntry("#", "#")
			} else if reLimitNum.MatchString(statLimit) {
				n, _ := strconv.ParseFloat(statLimit, 64)
				limit = limitEntry(n, n)
			} else if neg := reLimitNeg.FindStringSubmatch(statLimit); neg != nil {
				n, _ := strconv.ParseFloat(neg[1], 64)
				limit = limitEntry("!", n)
			} else if r := reLimitRange.FindStringSubmatch(statLimit); r != nil {
				limit = limitEntry(numOrStr(r[1]), numOrStr(r[2]))
			} else {
				limit = luaTable{}
			}
			limitLen++
			limits[limitLen] = limit
		}
		tokens := reSpecialTok.FindAllString(special, -1)
		for ti := 0; ti < len(tokens); {
			token := tokens[ti]
			if token == "canonical_line" {
				descLen++
				desc[descLen] = luaTable{"k": "canonical_line", "v": true}
				ti++
			} else if ti+1 < len(tokens) {
				descLen++
				desc[descLen] = luaTable{"k": token, "v": numOrStr(tokens[ti+1])}
				ti += 2
			} else {
				ti++
			}
		}
		if strings.Contains(quality, "gem_quality") {
			desc[quality] = true
		}
		curLangLen++
		(*curLang)[curLangLen] = desc
	}

	text := convertUTF16to8([]byte(x.GetFile("Metadata/StatDescriptions/"+name+".txt")), 0)
	for _, l := range reLine.FindAllString(text, -1) {
		processLine(l)
	}

	out := x.Out("Data/StatDescriptions/" + name + ".lua")
	out.W("-- This file is automatically generated, do not edit!\n")
	out.W("-- Item data (c) Grinding Gear Games\n\nreturn ")
	writeLuaTable(out, statDescriptor, 1)
	return out.Close()
}
