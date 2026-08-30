// Port of .archive/src/Export/Scripts/masters.lua.

package export

import "github.com/MissingL-tter/missingPassives/data/schema"

func init() {
	Scripts = append(Scripts, Script{Name: "masters", Build: buildMasters})
}

func buildMasters(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("stat_descriptions.txt")

	itemClassMap := map[string]string{
		"LifeFlask":                "Flask",
		"ManaFlask":                "Flask",
		"HybridFlask":              "Flask",
		"Amulet":                   "Amulet",
		"Ring":                     "Ring",
		"Claw":                     "Claw",
		"Dagger":                   "Dagger",
		"Rune Dagger":              "Dagger",
		"Wand":                     "Wand",
		"One Hand Sword":           "One Handed Sword",
		"Thrusting One Hand Sword": "Thrusting One Handed Sword",
		"One Hand Axe":             "One Handed Axe",
		"One Hand Mace":            "One Handed Mace",
		"Bow":                      "Bow",
		"Fishing Rod":              "Fishing Rod",
		"Staff":                    "Staff",
		"Warstaff":                 "Staff",
		"Two Hand Sword":           "Two Handed Sword",
		"Two Hand Axe":             "Two Handed Axe",
		"Two Hand Mace":            "Two Handed Mace",
		"Quiver":                   "Quiver",
		"Belt":                     "Belt",
		"Gloves":                   "Gloves",
		"Boots":                    "Boots",
		"Body Armour":              "Body Armour",
		"Helmet":                   "Helmet",
		"Shield":                   "Shield",
		"Sceptre":                  "Sceptre",
		"UtilityFlask":             "Flask",
		"UtilityFlaskCritical":     "Flask",
		"Map":                      "Map",
		"Jewel":                    "Jewel",
	}

	benchOptions, err := x.Dat("CraftingBenchOptions")
	if err != nil {
		return nil, err
	}
	var mc schema.MasterCrafts
	for _, craft := range benchOptions.GetRowList("IsDisabled", false) {
		mod := craft.Ref("Mod")
		if mod == nil {
			continue
		}
		c := schema.MasterCraft{Affix: mod.Str("Name")}
		switch mod.Int("GenerationType") {
		case 1:
			c.Type = "Prefix"
		case 2:
			c.Type = "Suffix"
		}
		stats, err := x.DescribeMod(mod)
		if err != nil {
			return nil, err
		}
		c.ModTags = stats.ModTags
		c.Lines = stats.Lines
		c.StatOrders = stats.Orders
		c.Level = mod.Int("Level")
		c.Group = mod.Ref("Type").Str("Id")
		uniqueTypes := map[string]bool{}
		for _, category := range craft.Refs("ItemCategories") {
			for _, itemClass := range category.Refs("ItemClasses") {
				mapped, found := itemClassMap[itemClass.Str("Id")]
				if found && !uniqueTypes[mapped] {
					uniqueTypes[mapped] = true
					c.Types = append(c.Types, mapped)
				}
			}
		}
		mc = append(mc, c)
	}
	return mc, nil
}
