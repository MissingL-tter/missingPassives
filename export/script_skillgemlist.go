// Port of .archive/src/Export/Scripts/skillGemList.lua.
//
// The Lua's `export` flag is false, so only the plain gem lists are written;
// the grantedEffectString/temp1 side is dead code there (and its ipairs over
// the single-Key GrantedEffectStatSets cell yields nothing under LuaJIT's raw
// ipairs anyway), so it is not ported.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "skillGemList", Build: buildSkillGemList})
}

func buildSkillGemList(x *Ctx) (schema.Document, error) {
	skillGems, err := x.Dat("SkillGems")
	if err != nil {
		return nil, err
	}
	var doc schema.SkillGemList

	types := []string{"Strength", "Dexterity", "Intelligence", "Other"}

	for i := 1; i <= len(types); i++ {
		var active, support []schema.SkillGemEntry
		for skillGem := range skillGems.Rows() {
			for _, gemEffect := range skillGem.Refs("GemVariants") {
				var colour string
				if skillGem.Int("Str") >= 50 {
					colour = "Strength"
				} else if skillGem.Int("Int") >= 50 {
					colour = "Intelligence"
				} else if skillGem.Int("Dex") >= 50 {
					colour = "Dexterity"
				} else {
					colour = "Other"
				}
				gemId := gemEffect.Str("Id")
				desc := gemEffect.Str("Description")
				if skillGem.Bool("IsSupport") {
					gemName := skillGem.Ref("BaseItemType").Str("Name")
					if skillGem.Int("GemColour") == int64(i) &&
						!strings.Contains(gemId, "Unknown") && !strings.Contains(gemId, "Playtest") &&
						!strings.Contains(gemId, "Royale") && !strings.Contains(gemName, "DNT") &&
						!strings.Contains(gemName, "UNUSED") && !strings.Contains(gemName, "NOT CURRENTLY USED") &&
						!strings.Contains(gemName, "WIP") && !strings.Contains(gemName, "Unnamed") &&
						!strings.Contains(desc, "DNT") {
						e := schema.SkillGemEntry{Name: gemName, Effects: []string{gemEffect.Ref("GrantedEffect").Str("Id")}}
						if ge2 := gemEffect.Ref("GrantedEffect2"); ge2 != nil {
							e.Effects = append(e.Effects, ge2.Str("Id"))
						}
						support = append(support, e)
					}
				} else {
					grantedEffect := gemEffect.Ref("GrantedEffect")
					gemName := grantedEffect.Ref("ActiveSkill").Str("DisplayName")
					if gemName != "" && types[i-1] == colour &&
						!strings.Contains(gemId, "Unknown") && !strings.Contains(gemId, "Playtest") &&
						!strings.Contains(gemId, "Royale") && !strings.Contains(gemName, "...") &&
						!strings.Contains(gemName, "DNT") && !strings.Contains(gemName, "UNUSED") &&
						!strings.Contains(gemName, "NOT CURRENTLY USED") && !strings.Contains(gemName, "Unnamed") &&
						!strings.Contains(grantedEffect.Str("Id"), "HardMode") &&
						!strings.Contains(skillGem.Ref("BaseItemType").Str("Name"), "DNT") &&
						!strings.Contains(desc, "DNT") &&
						!(skillGem.Bool("IsVaalGem") && gemEffect.Int("Variant") != 5) {
						e := schema.SkillGemEntry{Name: gemName, Effects: []string{grantedEffect.Str("Id")}}
						if ge2 := gemEffect.Ref("GrantedEffect2"); ge2 != nil && !skillGem.Bool("IsVaalGem") {
							e.Effects = append(e.Effects, ge2.Str("Id"))
						}
						active = append(active, e)
					}
				}
			}
		}
		doc.Groups = append(doc.Groups, schema.SkillGemGroup{
			Type:    types[i-1],
			Active:  active,
			Support: support,
		})
	}
	return doc, nil
}
