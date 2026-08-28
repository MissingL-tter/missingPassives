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

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "statdesc", Build: buildStatdesc})
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

func sdNum(s string) *schema.NumOrStr {
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return &schema.NumOrStr{Num: &n}
	}
	return &schema.NumOrStr{Str: &s}
}

func sdStr(s string) *schema.NumOrStr {
	return &schema.NumOrStr{Str: &s}
}

func buildStatdesc(x *Ctx) (any, error) {
	docs := schema.StatDescs{}
	for _, name := range statFileList {
		docs[name] = parseStatFile(x, name)
	}
	return docs, nil
}

func parseStatFile(x *Ctx, name string) *schema.StatDescFile {
	f := &schema.StatDescFile{}
	var descs []*schema.StatDescriptor
	// curDescriptor starts detached, as in the Lua: lines before the first
	// description mutate a table never added to the file.
	curDescriptor := &schema.StatDescriptor{Lang: []schema.DescLine{}}
	inEnglish := false
	prepend := ""

	processLine := func(line string) {
		line = prepend + line
		prepend = ""
		if m := reParentInclude.FindStringSubmatch(line); m != nil {
			f.Parent = m[1]
			return
		}
		if m := reNoDesc.FindStringSubmatch(line); m != nil {
			descs = append(descs, &schema.StatDescriptor{
				NoDesc: true,
				Stats:  []string{m[1]},
			})
			return
		}
		if strings.Contains(line, "handed_description") ||
			(strings.Contains(line, "description") && !strings.Contains(line, "_description")) {
			curDescriptor = &schema.StatDescriptor{Lang: []schema.DescLine{}}
			inEnglish = true
			if m := reDescName.FindStringSubmatch(line); m != nil {
				curDescriptor.Name = m[1]
			}
			descs = append(descs, curDescriptor)
			return
		}
		if curDescriptor.Stats == nil {
			if m := reStatsLine.FindStringSubmatch(line); m != nil {
				stats := []string{}
				stats = append(stats, reStatWord.FindAllString(m[1], -1)...)
				curDescriptor.Stats = stats
			} else {
				// Try to combine it with the next line.
				prepend = line
			}
			return
		}
		if reLangLine.MatchString(line) {
			inEnglish = false
			return
		}
		if !inEnglish || strings.Contains(line, "table_only") {
			return
		}
		m := reDescLine.FindStringSubmatch(line)
		if m == nil {
			return
		}
		statLimits, quality, text, special := m[1], m[2], m[3], m[4]
		desc := schema.DescLine{Text: escapeGGGString(text), Limits: []schema.DescLimit{}}
		for _, statLimit := range reLimitTok.FindAllString(statLimits, -1) {
			var limit schema.DescLimit
			if statLimit == "#" {
				limit = schema.DescLimit{Min: sdStr("#"), Max: sdStr("#")}
			} else if reLimitNum.MatchString(statLimit) {
				n, _ := strconv.ParseFloat(statLimit, 64)
				limit = schema.DescLimit{Min: &schema.NumOrStr{Num: &n}, Max: &schema.NumOrStr{Num: &n}}
			} else if neg := reLimitNeg.FindStringSubmatch(statLimit); neg != nil {
				n, _ := strconv.ParseFloat(neg[1], 64)
				limit = schema.DescLimit{Min: sdStr("!"), Max: &schema.NumOrStr{Num: &n}}
			} else if r := reLimitRange.FindStringSubmatch(statLimit); r != nil {
				limit = schema.DescLimit{Min: sdNum(r[1]), Max: sdNum(r[2])}
			}
			desc.Limits = append(desc.Limits, limit)
		}
		tokens := reSpecialTok.FindAllString(special, -1)
		for ti := 0; ti < len(tokens); {
			token := tokens[ti]
			if token == "canonical_line" {
				v := true
				desc.Specials = append(desc.Specials, schema.DescSpecial{K: "canonical_line", VBool: &v})
				ti++
			} else if ti+1 < len(tokens) {
				sp := schema.DescSpecial{K: token}
				if ns := sdNum(tokens[ti+1]); ns.Num != nil {
					sp.VNum = ns.Num
				} else {
					sp.VStr = ns.Str
				}
				desc.Specials = append(desc.Specials, sp)
				ti += 2
			} else {
				ti++
			}
		}
		if strings.Contains(quality, "gem_quality") {
			desc.Quality = quality
		}
		curDescriptor.Lang = append(curDescriptor.Lang, desc)
	}

	text := convertUTF16to8([]byte(x.GetFile("Metadata/StatDescriptions/"+name+".txt")), 0)
	for _, l := range reLine.FindAllString(text, -1) {
		processLine(l)
	}
	for _, d := range descs {
		f.Descriptors = append(f.Descriptors, *d)
	}
	return f
}
