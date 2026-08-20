package modparser

// Small lookup tables — ModParser.lua:5893-6038.

// Modifiers that are recognised but unsupported — ModParser.lua:5894.
var unsupportedModList = map[string]bool{
	"properties are doubled while in a breach": true,
	"mirrored": true,
	"split":    true,
}

// suffixTypes — ModParser.lua:5901.
var suffixTypes = map[string]any{
	"as extra lightning damage":        "GainAsLightning",
	"added as lightning damage":        "GainAsLightning",
	"gained as extra lightning damage": "GainAsLightning",
	"as extra cold damage":             "GainAsCold",
	"added as cold damage":             "GainAsCold",
	"gained as extra cold damage":      "GainAsCold",
	"as extra fire damage":             "GainAsFire",
	"added as fire damage":             "GainAsFire",
	"gained as extra fire damage":      "GainAsFire",
	"as extra chaos damage":            "GainAsChaos",
	"added as chaos damage":            "GainAsChaos",
	"gained as extra chaos damage":     "GainAsChaos",
	"converted to lightning":           "ConvertToLightning",
	"converted to lightning damage":    "ConvertToLightning",
	"converted to cold damage":         "ConvertToCold",
	"converted to fire damage":         "ConvertToFire",
	"converted to fire":                "ConvertToFire",
	"converted to chaos damage":        "ConvertToChaos",
	"added as energy shield":           "GainAsEnergyShield",
	"as extra maximum energy shield":   "GainAsEnergyShield",
	"converted to energy shield":       "ConvertToEnergyShield",
	"as extra armour":                  "GainAsArmour",
	"as physical damage":               "AsPhysical",
	"as lightning damage":              "AsLightning",
	"as cold damage":                   "AsCold",
	"as fire damage":                   "AsFire",
	"as fire":                          "AsFire",
	"as chaos damage":                  "AsChaos",
	"leeched as life and mana":         "Leech",
	"leeched as life":                  "LifeLeech",
	"is leeched as life":               "LifeLeech",
	"leeched as mana":                  "ManaLeech",
	"is leeched as mana":               "ManaLeech",
	"leeched as energy shield":         "EnergyShieldLeech",
	"is leeched as energy shield":      "EnergyShieldLeech",
}

// dmgTypes — ModParser.lua:5938.
var dmgTypes = map[string]any{
	"physical":  "Physical",
	"lightning": "Lightning",
	"cold":      "Cold",
	"fire":      "Fire",
	"chaos":     "Chaos",
}

// penTypes — ModParser.lua:5945.
var penTypes = map[string]any{
	"lightning resistance":  "LightningPenetration",
	"cold resistance":       "ColdPenetration",
	"fire resistance":       "FirePenetration",
	"elemental resistance":  "ElementalPenetration",
	"elemental resistances": "ElementalPenetration",
	"chaos resistance":      "ChaosPenetration",
}

// resourceTypes — ModParser.lua:5953, including the generated "maximum X"
// variants from the do-block at 5971.
var resourceTypes = buildResourceTypes()

func buildResourceTypes() map[string]any {
	base := map[string]any{
		"life":                         "Life",
		"mana":                         "Mana",
		"energy shield":                "EnergyShield",
		"life and mana":                []any{"Life", "Mana"},
		"life and energy shield":       []any{"Life", "EnergyShield"},
		"life, mana and energy shield": []any{"Life", "Mana", "EnergyShield"},
		"life, energy shield and mana": []any{"Life", "Mana", "EnergyShield"},
		"mana and life":                []any{"Life", "Mana"},
		"mana and energy shield":       []any{"Mana", "EnergyShield"},
		"mana, life and energy shield": []any{"Life", "Mana", "EnergyShield"},
		"mana, energy shield and life": []any{"Life", "Mana", "EnergyShield"},
		"energy shield and life":       []any{"Life", "EnergyShield"},
		"energy shield and mana":       []any{"Mana", "EnergyShield"},
		"energy shield, life and mana": []any{"Life", "Mana", "EnergyShield"},
		"energy shield, mana and life": []any{"Life", "Mana", "EnergyShield"},
		"rage":                         "Rage",
	}
	// Collected first: Go's map iteration must not see the keys it adds,
	// mirroring the reference's separate maximumResourceTypes table.
	maximums := make(map[string]any, len(base))
	for resource, values := range base {
		maximums["maximum "+resource] = values
	}
	for resource, values := range maximums {
		base[resource] = values
	}
	return base
}

// appendMod — ModParser.lua:5980: suffix every name in a resource table.
func appendMod(inputTable map[string]any, suffix string) map[string]any {
	out := make(map[string]any, len(inputTable))
	for subLine, mods := range inputTable {
		if s, ok := mods.(string); ok {
			out[subLine] = s + suffix
		} else {
			list := mods.([]any)
			nl := make([]any, len(list))
			for i, m := range list {
				nl[i] = m.(string) + suffix
			}
			out[subLine] = nl
		}
	}
	return out
}

var regenTypes = appendMod(resourceTypes, "Regen")
var degenTypes = appendMod(resourceTypes, "Degen")
var costTypes = appendMod(resourceTypes, "Cost")
var baseCostTypes = appendMod(resourceTypes, "CostNoMult")

// flagTypes — ModParser.lua:5998. The reference repeats two shrine keys with
// identical values; they appear once here.
var flagTypes = map[string]any{
	"phasing":              "Condition:Phasing",
	"onslaught":            "Condition:Onslaught",
	"rampage":              "Condition:Rampage",
	"soul eater":           "Condition:CanHaveSoulEater",
	"adrenaline":           "Condition:Adrenaline",
	"elusive":              "Condition:CanBeElusive",
	"arcane surge":         "Condition:ArcaneSurge",
	"fortify":              "Condition:Fortified",
	"fortified":            "Condition:Fortified",
	"unholy might":         "Condition:UnholyMight",
	"chaotic might":        "Condition:ChaoticMight",
	"tailwind":             "Condition:Tailwind",
	"intimidated":          "Condition:Intimidated",
	"crushed":              "Condition:Crushed",
	"chilled":              "Condition:Chilled",
	"blinded":              "Condition:Blinded",
	"no life regeneration": "NoLifeRegen",
	"hexproof":             Tag{"name": "CurseEffectOnSelf", "value": -100, "type": "MORE"},
	`hindered,? with ([0-9]+)% reduced movement speed`: "Condition:Hindered",
	"unnerved":                     "Condition:Unnerved",
	"malediction":                  "HasMalediction",
	"debilitated":                  "Condition:Debilitated",
	"lesser brutal shrine buff":    "Condition:LesserBrutalShrine",
	"lesser massive shrine buff":   "Condition:LesserMassiveShrine",
	"acceleration shrine buff":     "Condition:AccelerationShrine",
	"brutal shrine buff":           "Condition:BrutalShrine",
	"diamond shrine buff":          "Condition:DiamondShrine",
	"echoing shrine buff":          "Condition:EchoingShrine",
	"gloom shrine buff":            "Condition:GloomShrine",
	"greater freezing shrine buff": "Condition:GreaterFreezingShrine",
	"greater shocking shrine buff": "Condition:GreaterShockingShrine",
	"greater skeletal shrine buff": "Condition:GreaterSkeletalShrine",
	"impenetrable shrine buff":     "Condition:ImpenetrableShrine",
	"massive shrine buff":          "Condition:MassiveShrine",
	"replenishing shrine buff":     "Condition:ReplenishingShrine",
	"resistance shrine buff":       "Condition:ResistanceShrine",
	"resonating shrine buff":       "Condition:ResonatingShrine",
}
