// Ported from Modules/Data.lua: data.highPrecisionMods and
// data.defaultHighPrecision, consulted by ScaleAddMod and the MORE
// aggregation. Moves to the game-data module when that lands.

package modstore

import "github.com/MissingL-tter/missingPassives/modparser"

const defaultHighPrecision = 1

var highPrecisionMods = map[string]map[modparser.ModType]int{
	"CritChance":                       {modparser.Base: 2},
	"SelfCritChance":                   {modparser.Base: 2},
	"LifeRegenPercent":                 {modparser.Base: 2},
	"ManaRegenPercent":                 {modparser.Base: 2},
	"EnergyShieldRegenPercent":         {modparser.Base: 2},
	"LifeRegen":                        {modparser.Base: 1},
	"ManaRegen":                        {modparser.Base: 1},
	"EnergyShieldRegen":                {modparser.Base: 1},
	"RageRegen":                        {modparser.Base: 1},
	"LifeDegenPercent":                 {modparser.Base: 2},
	"LifeDegenPercentTincture":         {modparser.Base: 2},
	"ManaDegenPercent":                 {modparser.Base: 2},
	"ManaDegenPercentTincture":         {modparser.Base: 2},
	"EnergyShieldDegenPercent":         {modparser.Base: 2},
	"LifeDegen":                        {modparser.Base: 1},
	"ManaDegen":                        {modparser.Base: 1},
	"EnergyShieldDegen":                {modparser.Base: 1},
	"DamageLifeLeech":                  {modparser.Base: 2},
	"PhysicalDamageLifeLeech":          {modparser.Base: 2},
	"ElementalDamageLifeLeech":         {modparser.Base: 2},
	"FireDamageLifeLeech":              {modparser.Base: 2},
	"ColdDamageLifeLeech":              {modparser.Base: 2},
	"LightningDamageLifeLeech":         {modparser.Base: 2},
	"ChaosDamageLifeLeech":             {modparser.Base: 2},
	"DamageManaLeech":                  {modparser.Base: 2},
	"PhysicalDamageManaLeech":          {modparser.Base: 2},
	"ElementalDamageManaLeech":         {modparser.Base: 2},
	"FireDamageManaLeech":              {modparser.Base: 2},
	"ColdDamageManaLeech":              {modparser.Base: 2},
	"LightningDamageManaLeech":         {modparser.Base: 2},
	"ChaosDamageManaLeech":             {modparser.Base: 2},
	"DamageEnergyShieldLeech":          {modparser.Base: 2},
	"PhysicalDamageEnergyShieldLeech":  {modparser.Base: 2},
	"ElementalDamageEnergyShieldLeech": {modparser.Base: 2},
	"FireDamageEnergyShieldLeech":      {modparser.Base: 2},
	"ColdDamageEnergyShieldLeech":      {modparser.Base: 2},
	"LightningDamageEnergyShieldLeech": {modparser.Base: 2},
	"ChaosDamageEnergyShieldLeech":     {modparser.Base: 2},
	"SupportManaMultiplier":            {modparser.More: 4},
}
