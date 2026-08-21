// Ported from Modules/Data.lua: data.highPrecisionMods and
// data.defaultHighPrecision, consulted by ScaleAddMod and the MORE
// aggregation. Moves to the game-data module when that lands.

package modstore

const defaultHighPrecision = 1

var highPrecisionMods = map[string]map[string]int{
	"CritChance":                       {"BASE": 2},
	"SelfCritChance":                   {"BASE": 2},
	"LifeRegenPercent":                 {"BASE": 2},
	"ManaRegenPercent":                 {"BASE": 2},
	"EnergyShieldRegenPercent":         {"BASE": 2},
	"LifeRegen":                        {"BASE": 1},
	"ManaRegen":                        {"BASE": 1},
	"EnergyShieldRegen":                {"BASE": 1},
	"RageRegen":                        {"BASE": 1},
	"LifeDegenPercent":                 {"BASE": 2},
	"LifeDegenPercentTincture":         {"BASE": 2},
	"ManaDegenPercent":                 {"BASE": 2},
	"ManaDegenPercentTincture":         {"BASE": 2},
	"EnergyShieldDegenPercent":         {"BASE": 2},
	"LifeDegen":                        {"BASE": 1},
	"ManaDegen":                        {"BASE": 1},
	"EnergyShieldDegen":                {"BASE": 1},
	"DamageLifeLeech":                  {"BASE": 2},
	"PhysicalDamageLifeLeech":          {"BASE": 2},
	"ElementalDamageLifeLeech":         {"BASE": 2},
	"FireDamageLifeLeech":              {"BASE": 2},
	"ColdDamageLifeLeech":              {"BASE": 2},
	"LightningDamageLifeLeech":         {"BASE": 2},
	"ChaosDamageLifeLeech":             {"BASE": 2},
	"DamageManaLeech":                  {"BASE": 2},
	"PhysicalDamageManaLeech":          {"BASE": 2},
	"ElementalDamageManaLeech":         {"BASE": 2},
	"FireDamageManaLeech":              {"BASE": 2},
	"ColdDamageManaLeech":              {"BASE": 2},
	"LightningDamageManaLeech":         {"BASE": 2},
	"ChaosDamageManaLeech":             {"BASE": 2},
	"DamageEnergyShieldLeech":          {"BASE": 2},
	"PhysicalDamageEnergyShieldLeech":  {"BASE": 2},
	"ElementalDamageEnergyShieldLeech": {"BASE": 2},
	"FireDamageEnergyShieldLeech":      {"BASE": 2},
	"ColdDamageEnergyShieldLeech":      {"BASE": 2},
	"LightningDamageEnergyShieldLeech": {"BASE": 2},
	"ChaosDamageEnergyShieldLeech":     {"BASE": 2},
	"SupportManaMultiplier":            {"MORE": 4},
}
