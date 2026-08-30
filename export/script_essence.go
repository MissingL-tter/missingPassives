// Port of .archive/src/Export/Scripts/essence.lua.

package export

import "github.com/MissingL-tter/missingPassives/data/schema"

func init() {
	Scripts = append(Scripts, Script{Name: "essence", Build: buildEssence})
}

func buildEssence(x *Ctx) (schema.Document, error) {
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

	essences, err := x.Dat("Essences")
	if err != nil {
		return nil, err
	}
	var es schema.Essences
	for essence := range essences.Rows() {
		if essence.Int("Tier") > 0 {
			base := essence.Ref("BaseItemType")
			e := schema.Essence{
				BaseId: base.Str("Id"),
				Name:   base.Str("Name"),
				Type:   essence.Ref("Type").ID,
				Tier:   essence.Int("Tier"),
				Mods:   map[string]string{},
			}
			for typ, col := range colMap {
				e.Mods[typ] = essence.Ref(col).Str("Id")
			}
			es = append(es, e)
		}
	}
	return es, nil
}
