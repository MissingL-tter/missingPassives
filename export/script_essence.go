// Port of .archive/src/Export/Scripts/essence.lua.

package export

import "github.com/MissingL-tter/missingPassives/gamedata"

func init() {
	Scripts = append(Scripts, Script{Name: "essence", Build: buildEssence})
}

func buildEssence(x *Ctx) (any, error) {
	colMap := map[string]string{
		"Amulet":                     "AmuletMod",
		"Ring":                       "RingMod",
		"Belt":                       "BeltMod",
		"Quiver":                     "QuiverMod",
		"Helmet":                     "HelmetMod",
		"Body Armour":                "BodyArmourMod",
		"Boots":                      "BootsMod",
		"Gloves":                     "GlovesMod",
		"Shield":                     "ShieldMod",
		"Bow":                        "BowMod",
		"Claw":                       "ClawMod",
		"Dagger":                     "DaggerMod",
		"Staff":                      "StaffMod",
		"Wand":                       "WandMod",
		"One Handed Axe":             "OneHandAxeMod",
		"One Handed Mace":            "OneHandMaceMod",
		"One Handed Sword":           "OneHandSwordMod",
		"Sceptre":                    "SceptreMod",
		"Thrusting One Handed Sword": "ThrustingOneHandSwordMod",
		"Two Handed Axe":             "TwoHandAxeMod",
		"Two Handed Mace":            "TwoHandMaceMod",
		"Two Handed Sword":           "TwoHandSwordMod",
	}

	var es gamedata.Essences
	x.Dat("Essences").Rows(func(essence *Row) bool {
		if essence.Get("Tier").(int64) > 0 {
			base := essence.Get("BaseItemType").(*Row)
			e := gamedata.Essence{
				BaseId: luaStr(base.Get("Id")),
				Name:   luaStr(base.Get("Name")),
				Type:   essence.Get("Type").(*Row).Index - 1,
				Tier:   essence.Get("Tier").(int64),
				Mods:   map[string]string{},
			}
			for typ, col := range colMap {
				e.Mods[typ] = luaStr(essence.Get(col).(*Row).Get("Id"))
			}
			es = append(es, e)
		}
		return true
	})
	return es, nil
}
