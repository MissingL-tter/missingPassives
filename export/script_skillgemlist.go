// Port of .archive/src/Export/Scripts/skillGemList.lua.
//
// The Lua's `export` flag is false, so only the plain gem lists are written;
// the grantedEffectString/temp1 side is dead code there (and its ipairs over
// the single-Key GrantedEffectStatSets cell yields nothing under LuaJIT's raw
// ipairs anyway), so it is not ported.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() {
	Scripts = append(Scripts, Script{Name: "skillGemList", Build: buildSkillGemList})
}

func buildSkillGemList(x *Ctx) (any, error) {
	var doc gamedata.SkillGemList

	types := []string{"Strength", "Dexterity", "Intelligence", "Other"}

	for i := 1; i <= len(types); i++ {
		var active, support []gamedata.SkillGemEntry
		x.Dat("SkillGems").Rows(func(skillGem *Row) bool {
			for _, ge := range skillGem.Get("GemVariants").([]any) {
				gemEffect := ge.(*Row)
				var colour string
				if skillGem.Get("Str").(int64) >= 50 {
					colour = "Strength"
				} else if skillGem.Get("Int").(int64) >= 50 {
					colour = "Intelligence"
				} else if skillGem.Get("Dex").(int64) >= 50 {
					colour = "Dexterity"
				} else {
					colour = "Other"
				}
				gemId := luaStr(gemEffect.Get("Id"))
				desc := luaStr(gemEffect.Get("Description"))
				if skillGem.Get("IsSupport").(bool) {
					gemName := luaStr(skillGem.Get("BaseItemType").(*Row).Get("Name"))
					if skillGem.Get("GemColour").(int64) == int64(i) &&
						!strings.Contains(gemId, "Unknown") && !strings.Contains(gemId, "Playtest") &&
						!strings.Contains(gemId, "Royale") && !strings.Contains(gemName, "DNT") &&
						!strings.Contains(gemName, "UNUSED") && !strings.Contains(gemName, "NOT CURRENTLY USED") &&
						!strings.Contains(gemName, "WIP") && !strings.Contains(gemName, "Unnamed") &&
						!strings.Contains(desc, "DNT") {
						e := gamedata.SkillGemEntry{Name: gemName, Effects: []string{luaStr(gemEffect.Get("GrantedEffect").(*Row).Get("Id"))}}
						if ge2, ok := gemEffect.Get("GrantedEffect2").(*Row); ok {
							e.Effects = append(e.Effects, luaStr(ge2.Get("Id")))
						}
						support = append(support, e)
					}
				} else {
					grantedEffect := gemEffect.Get("GrantedEffect").(*Row)
					gemName := luaStr(grantedEffect.Get("ActiveSkill").(*Row).Get("DisplayName"))
					if gemName != "" && types[i-1] == colour &&
						!strings.Contains(gemId, "Unknown") && !strings.Contains(gemId, "Playtest") &&
						!strings.Contains(gemId, "Royale") && !strings.Contains(gemName, "...") &&
						!strings.Contains(gemName, "DNT") && !strings.Contains(gemName, "UNUSED") &&
						!strings.Contains(gemName, "NOT CURRENTLY USED") && !strings.Contains(gemName, "Unnamed") &&
						!strings.Contains(luaStr(grantedEffect.Get("Id")), "HardMode") &&
						!strings.Contains(luaStr(skillGem.Get("BaseItemType").(*Row).Get("Name")), "DNT") &&
						!strings.Contains(desc, "DNT") &&
						!(skillGem.Get("IsVaalGem").(bool) && gemEffect.Get("Variant").(int64) != 5) {
						e := gamedata.SkillGemEntry{Name: gemName, Effects: []string{luaStr(grantedEffect.Get("Id"))}}
						if ge2, ok := gemEffect.Get("GrantedEffect2").(*Row); ok && !skillGem.Get("IsVaalGem").(bool) {
							e.Effects = append(e.Effects, luaStr(ge2.Get("Id")))
						}
						active = append(active, e)
					}
				}
			}
			return true
		})
		doc.Groups = append(doc.Groups, gamedata.SkillGemGroup{
			Type:    types[i-1],
			Active:  active,
			Support: support,
		})
	}
	return doc, nil
}
