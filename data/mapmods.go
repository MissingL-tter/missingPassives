// Converted once from the reference's Data/ModMap.lua; Go-maintained
// since: the map modifier configuration data. Apply is a marker until the
// config module ports the apply closures.

package data

// MapModData is data.mapMods: the affix records and the prefix/suffix
// selector lists.
type MapModData struct {
	AffixData      map[string]*MapMod
	Prefix, Suffix []ValLabel
}

// MapMod is one affix record; an affix without configuration data is an
// empty record.
type MapMod struct {
	Label        string
	Tooltip      string
	TooltipLines []string
	Type         string // "list", "count" or "check"
	Values       []MapModValue
	Apply        MapModApplyKind
}

// MapModValue is one per-tier value: a number, or a list of values (the
// count-type affixes carry ranges, some per component).
type MapModValue struct {
	Num  float64
	List []MapModValue // nil = the value is Num
}

// MapModApplyKind identifies the affix's apply function.
type MapModApplyKind uint8

const (
	MapModApplyNone     MapModApplyKind = iota
	MapModApplyUnported                 // the Lua closure is not ported yet
)

var mapModsTable = MapModData{
	AffixData: map[string]*MapMod{
		"Abhorrent":       {},
		"Antagonist's":    {},
		"Armoured":        {Label: "Enemy Physical Damage reduction:", TooltipLines: []string{"+%d%% Monster Physical Damage Reduction"}, Type: "list", Values: []MapModValue{{Num: 20}, {Num: 30}, {Num: 40}}, Apply: MapModApplyUnported},
		"Bipedal":         {},
		"Buffered":        {TooltipLines: []string{"Monsters gain (%d to %d)%% of Maximum Life as Extra Maximum Energy Shield"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 20}, {Num: 29}}}, {List: []MapModValue{{Num: 30}, {Num: 39}}}, {List: []MapModValue{{Num: 40}, {Num: 49}}}}, Apply: MapModApplyUnported},
		"Burning":         {TooltipLines: []string{"Monsters deal (%d to %d)%% extra Physical Damage as Fire"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 50}, {Num: 69}}}, {List: []MapModValue{{Num: 70}, {Num: 89}}}, {List: []MapModValue{{Num: 90}, {Num: 110}}}}, Apply: MapModApplyUnported},
		"Capricious":      {},
		"Ceremonial":      {},
		"Chaining":        {},
		"Conflagrating":   {TooltipLines: []string{"All Monster Damage from Hits always Ignites"}, Type: "check", Apply: MapModApplyUnported},
		"Demonic":         {},
		"Emanant":         {},
		"Empowered":       {TooltipLines: []string{"Monsters have a %d%% chance to Ignite, Freeze and Shock on Hit"}, Type: "list", Values: []MapModValue{{}, {Num: 15}, {Num: 20}}, Apply: MapModApplyUnported},
		"Enthralled":      {},
		"Feasting":        {},
		"Fecund":          {TooltipLines: []string{"(%d to %d)%% more Monster Life"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 20}, {Num: 29}}}, {List: []MapModValue{{Num: 30}, {Num: 39}}}, {List: []MapModValue{{Num: 40}, {Num: 49}}}}, Apply: MapModApplyUnported},
		"Feral":           {},
		"Fleet":           {TooltipLines: []string{"(%d to %d)%% increased Monster Movement Speed", "(%d to %d)%% increased Monster Attack Speed", "(%d to %d)%% increased Monster Cast Speed"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{List: []MapModValue{{Num: 15}, {Num: 20}}}, {List: []MapModValue{{Num: 20}, {Num: 25}}}, {List: []MapModValue{{Num: 20}, {Num: 25}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 20}, {Num: 25}}}, {List: []MapModValue{{Num: 25}, {Num: 35}}}, {List: []MapModValue{{Num: 25}, {Num: 35}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 25}, {Num: 30}}}, {List: []MapModValue{{Num: 35}, {Num: 45}}}, {List: []MapModValue{{Num: 35}, {Num: 45}}}}}}, Apply: MapModApplyUnported},
		"Freezing":        {TooltipLines: []string{"Monsters deal (%d to %d)%% extra Physical Damage as Cold"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 50}, {Num: 69}}}, {List: []MapModValue{{Num: 70}, {Num: 89}}}, {List: []MapModValue{{Num: 90}, {Num: 110}}}}, Apply: MapModApplyUnported},
		"Haunting":        {},
		"Hexproof":        {Label: "Enemy is Hexproof?", TooltipLines: []string{"Monsters are Hexproof"}, Type: "check", Apply: MapModApplyUnported},
		"Hexwarded":       {Label: "Less effect of Curses on enemy:", TooltipLines: []string{"%d%% less effect of Curses on Monsters"}, Type: "list", Values: []MapModValue{{Num: 25}, {Num: 40}, {Num: 60}}, Apply: MapModApplyUnported},
		"Impaling":        {TooltipLines: []string{"Monsters' Attacks have %d%% chance to Impale on Hit"}, Type: "list", Values: []MapModValue{{Num: 25}, {Num: 40}, {Num: 60}}, Apply: MapModApplyUnported},
		"Impervious":      {TooltipLines: []string{"Monsters have a %d%% chance to avoid Poison, Impale, and Bleeding"}, Type: "list", Values: []MapModValue{{Num: 20}, {Num: 35}, {Num: 50}}, Apply: MapModApplyUnported},
		"Lunar":           {},
		"Mirrored":        {TooltipLines: []string{"Monsters reflect %d%% of Elemental Damage"}, Type: "list", Values: []MapModValue{{Num: 13}, {Num: 15}, {Num: 18}}, Apply: MapModApplyUnported},
		"Multifarious":    {},
		"Oppressive":      {TooltipLines: []string{"Monsters have +%d%% chance to Suppress Spell Damage"}, Type: "list", Values: []MapModValue{{Num: 30}, {Num: 45}, {Num: 60}}, Apply: MapModApplyUnported},
		"Overlord's":      {TooltipLines: []string{"Unique Boss deals %d%% increased Damage", "Unique Boss has %d%% increased Attack and Cast Speed"}, Type: "list", Values: []MapModValue{{List: []MapModValue{{Num: 15}, {Num: 20}}}, {List: []MapModValue{{Num: 20}, {Num: 25}}}, {List: []MapModValue{{Num: 25}, {Num: 30}}}}, Apply: MapModApplyUnported},
		"Profane":         {TooltipLines: []string{"Monsters gain (%d to %d)%% of their Physical Damage as Extra Chaos Damage", "Monsters Inflict Withered for %d seconds on Hit"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{List: []MapModValue{{}, {}}}, {List: []MapModValue{{}, {}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 21}, {Num: 25}}}, {List: []MapModValue{{Num: 100}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 31}, {Num: 35}}}, {List: []MapModValue{{Num: 100}}}}}}, Apply: MapModApplyUnported},
		"Punishing":       {TooltipLines: []string{"Monsters reflect %d%% of Physical Damage"}, Type: "list", Values: []MapModValue{{Num: 13}, {Num: 15}, {Num: 18}}, Apply: MapModApplyUnported},
		"Resistant":       {Label: "Enemy has Elemental / ^xD02090Chaos ^7Resist:", TooltipLines: []string{"+%d%% Monster Elemental Resistances", "+%d%% Monster Chaos Resistance"}, Type: "list", Values: []MapModValue{{List: []MapModValue{{Num: 20}, {Num: 15}}}, {List: []MapModValue{{Num: 30}, {Num: 20}}}, {List: []MapModValue{{Num: 40}, {Num: 25}}}}, Apply: MapModApplyUnported},
		"Savage":          {TooltipLines: []string{"(%d to %d)%% increased Monster Damage"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 14}, {Num: 17}}}, {List: []MapModValue{{Num: 18}, {Num: 21}}}, {List: []MapModValue{{Num: 22}, {Num: 25}}}}, Apply: MapModApplyUnported},
		"Shocking":        {TooltipLines: []string{"Monsters deal (%d to %d)%% extra Physical Damage as Lightning"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 50}, {Num: 69}}}, {List: []MapModValue{{Num: 70}, {Num: 89}}}, {List: []MapModValue{{Num: 90}, {Num: 110}}}}, Apply: MapModApplyUnported},
		"Skeletal":        {},
		"Slithering":      {},
		"Solar":           {},
		"Splitting":       {},
		"Titan's":         {TooltipLines: []string{"Unique Boss has %d%% increased Life", "Unique Boss has %d%% increased Area of Effect"}, Type: "list", Values: []MapModValue{{List: []MapModValue{{Num: 25}, {Num: 45}}}, {List: []MapModValue{{Num: 30}, {Num: 55}}}, {List: []MapModValue{{Num: 35}, {Num: 70}}}}, Apply: MapModApplyUnported},
		"Twinned":         {},
		"Undead":          {},
		"Unstoppable":     {TooltipLines: []string{"Monsters cannot be Taunted", "Monsters' Action Speed cannot be modified to below Base Value", "Monsters' Movement Speed cannot be modified to below Base Value"}, Type: "check", Values: []MapModValue{{List: []MapModValue{{}}}}, Apply: MapModApplyUnported},
		"Unwavering":      {TooltipLines: []string{"(%d to %d)%% more Monster Life", "Monsters cannot be Stunned"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{List: []MapModValue{{Num: 15}, {Num: 19}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 20}, {Num: 24}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 25}, {Num: 30}}}}}}, Apply: MapModApplyUnported},
		"of Balance":      {Label: "Player has Elemental Equilibrium?", TooltipLines: []string{"Players cannot inflict Exposure"}, Type: "check", Apply: MapModApplyUnported},
		"of Blinding":     {TooltipLines: []string{"Monsters Blind on Hit"}, Type: "check", Apply: MapModApplyUnported},
		"of Bloodlines":   {},
		"of Carnage":      {},
		"of Congealment":  {Label: "Cannot Leech ^xE05030Life ^7/ ^x7070FFMana?", TooltipLines: []string{"Monsters cannot be Leeched from"}, Type: "check", Apply: MapModApplyUnported},
		"of Consecration": {},
		"of Deadliness":   {TooltipLines: []string{"Monsters have (%d to %d)%% increased Critical Strike Chance", "+(%d to %d)%% to Monster Critical Strike Multiplier"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{List: []MapModValue{{Num: 160}, {Num: 200}}}, {List: []MapModValue{{Num: 30}, {Num: 35}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 260}, {Num: 300}}}, {List: []MapModValue{{Num: 36}, {Num: 40}}}}}, {List: []MapModValue{{List: []MapModValue{{Num: 360}, {Num: 400}}}, {List: []MapModValue{{Num: 41}, {Num: 45}}}}}}, Apply: MapModApplyUnported},
		"of Desecration":  {},
		"of Doubt":        {TooltipLines: []string{"Players have %d%% reduced effect of Non-Curse Auras from Skills"}, Type: "list", Values: []MapModValue{{Num: 25}, {Num: 40}, {Num: 60}}, Apply: MapModApplyUnported},
		"of Drought":      {Label: "Gains reduced Flask Charges:", TooltipLines: []string{"Players gain %d%% reduced Flask Charges"}, Type: "list", Values: []MapModValue{{Num: 30}, {Num: 40}, {Num: 50}}, Apply: MapModApplyUnported},
		"of Endurance":    {},
		"of Enervation":   {},
		"of Exposure":     {Label: "-X% maximum Resistances:", Tooltip: "Mid tier: 5-8%\nHigh tier: 9-12%", TooltipLines: []string{"Players have minus (%d to %d)%% to all maximum Resistances"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{}, {}}}, {List: []MapModValue{{Num: 5}, {Num: 8}}}, {List: []MapModValue{{Num: 9}, {Num: 12}}}}, Apply: MapModApplyUnported},
		"of Fatigue":      {TooltipLines: []string{"Players have %d%% less Cooldown Recovery Rate"}, Type: "list", Values: []MapModValue{{Num: 20}, {Num: 30}, {Num: 40}}, Apply: MapModApplyUnported},
		"of Flames":       {},
		"of Frenzy":       {},
		"of Giants":       {},
		"of Ice":          {},
		"of Impedance":    {},
		"of Impotence":    {Label: "Less Area of Effect:", TooltipLines: []string{"Players have %d%% less Area of Effect"}, Type: "list", Values: []MapModValue{{Num: 15}, {Num: 20}, {Num: 25}}, Apply: MapModApplyUnported},
		"of Imprecision":  {TooltipLines: []string{"Players have %d%% less Accuracy Rating"}, Type: "list", Values: []MapModValue{{Num: 15}, {Num: 20}, {Num: 25}}, Apply: MapModApplyUnported},
		"of Insulation":   {Label: "Enemy avoid Elemental Ailments:", TooltipLines: []string{"Monsters have %d%% chance to Avoid Elemental Ailments"}, Type: "list", Values: []MapModValue{{Num: 30}, {Num: 50}, {Num: 70}}, Apply: MapModApplyUnported},
		"of Lightning":    {},
		"of Miring":       {Label: "Unlucky Dodge / Enemy has inc. Accuracy:", TooltipLines: []string{"Monsters have %d%% increased Accuracy Rating", "Players have minus %d%% to amount of Suppressed Spell Damage Prevented"}, Type: "list", Values: []MapModValue{{List: []MapModValue{{Num: 10}, {Num: 30}}}, {List: []MapModValue{{Num: 15}, {Num: 40}}}, {List: []MapModValue{{Num: 20}, {Num: 50}}}}, Apply: MapModApplyUnported},
		"of Power":        {},
		"of Rust":         {Label: "Reduced Block Chance / less Armour:", TooltipLines: []string{"Players have %d%% less Armour", "Players have %d%% reduced Chance to Block"}, Type: "list", Values: []MapModValue{{List: []MapModValue{{Num: 20}, {Num: 20}}}, {List: []MapModValue{{Num: 30}, {Num: 25}}}, {List: []MapModValue{{Num: 40}, {Num: 30}}}}, Apply: MapModApplyUnported},
		"of Smothering":   {Label: "Less Recovery Rate of ^xE05030Life ^7and ^x88FFFFEnergy Shield:", TooltipLines: []string{"Players have %d%% less Recovery Rate of Life and Energy Shield"}, Type: "list", Values: []MapModValue{{Num: 20}, {Num: 40}, {Num: 60}}, Apply: MapModApplyUnported},
		"of Stasis":       {Label: "Cannot Regen ^xE05030Life^7, ^x7070FFMana ^7or ^x88FFFFES?", TooltipLines: []string{"Players cannot Regenerate Life, Mana or Energy Shield"}, Type: "check", Apply: MapModApplyUnported},
		"of Toughness":    {TooltipLines: []string{"Monsters take (%d to %d)%% reduced Extra Damage from Critical Strikes"}, Type: "count", Values: []MapModValue{{List: []MapModValue{{Num: 25}, {Num: 30}}}, {List: []MapModValue{{Num: 31}, {Num: 35}}}, {List: []MapModValue{{Num: 36}, {Num: 40}}}}, Apply: MapModApplyUnported},
		"of Transience":   {TooltipLines: []string{"Buffs on Players expire %d%% faster"}, Type: "list", Values: []MapModValue{{Num: 30}, {Num: 50}, {Num: 70}}, Apply: MapModApplyUnported},
		"of Venom":        {TooltipLines: []string{"Monsters Poison on Hit"}, Type: "check", Apply: MapModApplyUnported},
	},
	Prefix: []ValLabel{
		{Val: "NONE", Label: "None"},
		{Val: "Armoured", Label: "Enemy Phys D R                                Physical Damage reductionArmoured"},
		{Val: "Hexproof", Label: "Enemy is Hexproof?                                Hexproof"},
		{Val: "Hexwarded", Label: "Less Curse effect                                of Curses on enemyHexwarded"},
		{Val: "Resistant", Label: "Enemy Resist                                has Elemental / ChaosResistant"},
		{Val: "Unstoppable", Label: "Enemy Cannot Be Slowed                                 Monsters Taunted Action Speed modified below base valueUnstoppable"},
		{Val: "Impervious", Label: "avoid Poison and Bleed:                                Enemy Impervious"},
		{Val: "Savage", Label: "Enemy Inc Damage                                has increased DamageSavage"},
		{Val: "Burning", Label: "Enemy Phys As Fire                                 Monsters deal to extra Physical DamageBurning"},
		{Val: "Freezing", Label: "Enemy Phys As Cold                                 Monsters deal to extra Physical DamageFreezing"},
		{Val: "Shocking", Label: "Enemy Phys As Lightning                                 Monsters deal to extra Physical DamageShocking"},
		{Val: "Profane", Label: "Enemy Phys As Chaos                                 Monsters deal to extra Physical Damage Inflict Withered for seconds on Hit Profane"},
		{Val: "Fleet", Label: "Enemy Inc Speed                                 to increased Monster Movement Attack CastFleet"},
		{Val: "Impaling", Label: "Enemy Impale                                 Monsters have chance to with Attacks Impaling"},
		{Val: "Conflagrating", Label: "Hits always Ignites                                 All Monster Damage from Conflagrating"},
		{Val: "Empowered", Label: "Elemental Ailments on Hit                                 Monsters have chance to cause Empowered"},
		{Val: "Overlord's", Label: "Boss Inc Damage / Speed                                 Unique deals increased has Attack and CastOverlord's"},
	},
	Suffix: []ValLabel{
		{Val: "NONE", Label: "None"},
		{Val: "of Congealment", Label: "Cannot Leech                                Life / Manaof Congealment"},
		{Val: "of Drought", Label: "reduced Flask Charges                                Gainsof Drought"},
		{Val: "of Exposure", Label: "-X% maximum Res                                Resistancesof Exposure"},
		{Val: "of Impotence", Label: "Less Area of Effect:                                of Impotence"},
		{Val: "of Insulation", Label: "avoid Elemental Ailments:                                Enemyof Impotence"},
		{Val: "of Miring", Label: "Enemy has inc. Accuracy: / Players have to amount of Suppressed Spell Damage Prevented                                of Miring"},
		{Val: "of Rust", Label: "Reduced Block Chance / less Armour:                                of Rust"},
		{Val: "of Smothering", Label: "Less Recovery Rate of ^xE05030Life ^7and ^x88FFFFEnergy Shield:                                of Smothering"},
		{Val: "of Stasis", Label: "Cannot Regen                                Life, Mana or ESof Stasis"},
		{Val: "of Toughness", Label: "Enemy takes red. Extra Crit Damage:                                of Toughness"},
		{Val: "of Fatigue", Label: "Less Cooldown Recovery                                 Players have Rateof Fatigue"},
		{Val: "of Doubt", Label: "Reduced Aura Effect                                 Players have Non-Curse Auras from Skillsof Doubt"},
		{Val: "of Imprecision", Label: "Less Accuracy                                 Players have Ratingof Imprecision"},
		{Val: "of Venom", Label: "Poison On Hit                                 Monsters of Venom"},
		{Val: "of Deadliness", Label: "Enemy Critical Strike                                 Monsters have to increased Chance Monster Multiplierof Deadliness"},
	},
}
