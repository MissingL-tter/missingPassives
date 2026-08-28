// Port of .archive/src/Export/Scripts/masters.lua.

package export

import "github.com/MissingL-tter/missingPassives/data/schema"

func init() {
	Scripts = append(Scripts, Script{Name: "masters", Build: buildMasters})
}

func buildMasters(x *Ctx) (any, error) {
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

	var mc schema.MasterCrafts
	for _, craft := range x.Dat("CraftingBenchOptions").GetRowList("IsDisabled", false) {
		mod, ok := craft.Get("Mod").(*Row)
		if !ok {
			continue
		}
		c := schema.MasterCraft{Affix: luaStr(mod.Get("Name"))}
		switch mod.Get("GenerationType").(int64) {
		case 1:
			c.Type = "Prefix"
		case 2:
			c.Type = "Suffix"
		}
		stats := x.DescribeMod(mod)
		c.ModTags = stats.ModTags
		c.Lines = stats.Lines
		c.StatOrders = stats.Orders
		c.Level = mod.Get("Level").(int64)
		c.Group = luaStr(mod.Get("Type").(*Row).Get("Id"))
		uniqueTypes := map[string]bool{}
		for _, category := range craft.Get("ItemCategories").([]any) {
			for _, itemClass := range category.(*Row).Get("ItemClasses").([]any) {
				mapped, found := itemClassMap[luaStr(itemClass.(*Row).Get("Id"))]
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
