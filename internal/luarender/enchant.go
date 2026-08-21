// Renders gamedata.Enchants as the seven Data/Enchantment*.lua files
// (Scripts/enchant.lua's outputs).

package luarender

import (
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() { register("enchant", renderEnchant) }

var enchantSourceOrder = []string{"NORMAL", "CRUEL", "MERCILESS", "ENDGAME", "DEDICATION", "ENKINDLING", "INSTILLING", "HARVEST", "HEIST"}

func renderEnchant(d gamedata.Enchants, _ Templates) (map[string]string, error) {
	byDiff := func(m map[string][][]string) string {
		var b B
		b.itemHeader()
		for _, diff := range enchantSourceOrder {
			if list, ok := m[diff]; ok {
				b.W("\t[\"" + diff + "\"] = {\n")
				for _, stats := range list {
					b.W("\t\t\"" + strings.Join(stats, "/") + "\",\n")
				}
				b.W("\t},\n")
			}
		}
		b.W("}")
		return b.String()
	}

	var hb B
	hb.itemHeader()
	skills := make([]string, 0, len(d.Helmet))
	for s := range d.Helmet {
		skills = append(skills, s)
	}
	sort.Strings(skills)
	for _, skill := range skills {
		hb.W("\t[\"" + skill + "\"] = {\n")
		for _, diff := range enchantSourceOrder {
			if list, ok := d.Helmet[skill][diff]; ok {
				hb.W("\t\t[\"" + diff + "\"] = {\n")
				for _, stats := range list {
					hb.W("\t\t\t\"" + strings.Join(stats, "/") + "\",\n")
				}
				hb.W("\t\t},\n")
			}
		}
		hb.W("\t},\n")
	}
	hb.W("}")

	return map[string]string{
		"Data/EnchantmentBoots.lua":  byDiff(d.Boots),
		"Data/EnchantmentGloves.lua": byDiff(d.Gloves),
		"Data/EnchantmentBelt.lua":   byDiff(d.Belt),
		"Data/EnchantmentFlask.lua":  byDiff(d.Flask),
		"Data/EnchantmentBody.lua":   byDiff(d.Body),
		"Data/EnchantmentWeapon.lua": byDiff(d.Weapon),
		"Data/EnchantmentHelmet.lua": hb.String(),
	}, nil
}
