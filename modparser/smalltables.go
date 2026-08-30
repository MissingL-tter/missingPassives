package modparser

// Small lookup tables — ModParser.lua:5893-6038.

// Modifiers that are recognised but unsupported — ModParser.lua:5894.
var unsupportedModList = map[string]bool{
	"properties are doubled while in a breach": true,
	"mirrored": true,
	"split":    true,
}

// suffixTypes — ModParser.lua:5901.
var suffixTypes = map[string]string{
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
var dmgTypes = map[string]string{
	"physical":  "Physical",
	"lightning": "Lightning",
	"cold":      "Cold",
	"fire":      "Fire",
	"chaos":     "Chaos",
}

// penTypes — ModParser.lua:5945.
var penTypes = map[string]nameValue{
	"lightning resistance":  name("LightningPenetration"),
	"cold resistance":       name("ColdPenetration"),
	"fire resistance":       name("FirePenetration"),
	"elemental resistance":  name("ElementalPenetration"),
	"elemental resistances": name("ElementalPenetration"),
	"chaos resistance":      name("ChaosPenetration"),
}

// resourceTypes — ModParser.lua:5953, including the generated "maximum X"
// variants from the do-block at 5971.
var resourceTypes = buildResourceTypes()

func buildResourceTypes() map[string]nameValue {
	base := map[string]nameValue{
		"life":                         name("Life"),
		"mana":                         name("Mana"),
		"energy shield":                name("EnergyShield"),
		"life and mana":                nameList{"Life", "Mana"},
		"life and energy shield":       nameList{"Life", "EnergyShield"},
		"life, mana and energy shield": nameList{"Life", "Mana", "EnergyShield"},
		"life, energy shield and mana": nameList{"Life", "Mana", "EnergyShield"},
		"mana and life":                nameList{"Life", "Mana"},
		"mana and energy shield":       nameList{"Mana", "EnergyShield"},
		"mana, life and energy shield": nameList{"Life", "Mana", "EnergyShield"},
		"mana, energy shield and life": nameList{"Life", "Mana", "EnergyShield"},
		"energy shield and life":       nameList{"Life", "EnergyShield"},
		"energy shield and mana":       nameList{"Mana", "EnergyShield"},
		"energy shield, life and mana": nameList{"Life", "Mana", "EnergyShield"},
		"energy shield, mana and life": nameList{"Life", "Mana", "EnergyShield"},
		"rage":                         name("Rage"),
	}
	// Collected first: Go's map iteration must not see the keys it adds,
	// mirroring the reference's separate maximumResourceTypes table.
	maximums := make(map[string]nameValue, len(base))
	for resource, values := range base {
		maximums["maximum "+resource] = values
	}
	for resource, values := range maximums {
		base[resource] = values
	}
	return base
}

// appendMod — ModParser.lua:5980: suffix every name in a resource table.
func appendMod(inputTable map[string]nameValue, suffix string) map[string]nameValue {
	out := make(map[string]nameValue, len(inputTable))
	for subLine, names := range inputTable {
		switch v := names.(type) {
		case name:
			out[subLine] = name(string(v) + suffix)
		case nameList:
			nl := make(nameList, len(v))
			for i, m := range v {
				nl[i] = m + suffix
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
var flagTypes = map[string]flagTypeValue{
	"phasing":              flagName("Condition:Phasing"),
	"onslaught":            flagName("Condition:Onslaught"),
	"rampage":              flagName("Condition:Rampage"),
	"soul eater":           flagName("Condition:CanHaveSoulEater"),
	"adrenaline":           flagName("Condition:Adrenaline"),
	"elusive":              flagName("Condition:CanBeElusive"),
	"arcane surge":         flagName("Condition:ArcaneSurge"),
	"fortify":              flagName("Condition:Fortified"),
	"fortified":            flagName("Condition:Fortified"),
	"unholy might":         flagName("Condition:UnholyMight"),
	"chaotic might":        flagName("Condition:ChaoticMight"),
	"tailwind":             flagName("Condition:Tailwind"),
	"intimidated":          flagName("Condition:Intimidated"),
	"crushed":              flagName("Condition:Crushed"),
	"chilled":              flagName("Condition:Chilled"),
	"blinded":              flagName("Condition:Blinded"),
	"no life regeneration": flagName("NoLifeRegen"),
	"hexproof":             FlagTypeMod{Name: "CurseEffectOnSelf", Type: More, Value: Num(-100)},
	`hindered,? with ([0-9]+)% reduced movement speed`: flagName("Condition:Hindered"),
	"unnerved":                     flagName("Condition:Unnerved"),
	"malediction":                  flagName("HasMalediction"),
	"debilitated":                  flagName("Condition:Debilitated"),
	"lesser brutal shrine buff":    flagName("Condition:LesserBrutalShrine"),
	"lesser massive shrine buff":   flagName("Condition:LesserMassiveShrine"),
	"acceleration shrine buff":     flagName("Condition:AccelerationShrine"),
	"brutal shrine buff":           flagName("Condition:BrutalShrine"),
	"diamond shrine buff":          flagName("Condition:DiamondShrine"),
	"echoing shrine buff":          flagName("Condition:EchoingShrine"),
	"gloom shrine buff":            flagName("Condition:GloomShrine"),
	"greater freezing shrine buff": flagName("Condition:GreaterFreezingShrine"),
	"greater shocking shrine buff": flagName("Condition:GreaterShockingShrine"),
	"greater skeletal shrine buff": flagName("Condition:GreaterSkeletalShrine"),
	"impenetrable shrine buff":     flagName("Condition:ImpenetrableShrine"),
	"massive shrine buff":          flagName("Condition:MassiveShrine"),
	"replenishing shrine buff":     flagName("Condition:ReplenishingShrine"),
	"resistance shrine buff":       flagName("Condition:ResistanceShrine"),
	"resonating shrine buff":       flagName("Condition:ResonatingShrine"),
}
