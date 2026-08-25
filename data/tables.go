// The tables Modules/Data.lua defines inline, ported verbatim.

package data

import "strings"

// Colour codes from Data/Global.lua (colorCodes).
const (
	colorStrength     = "^xE05030" // MARAUDER
	colorDexterity    = "^x70FF70" // RANGER
	colorIntelligence = "^x7070FF" // WITCH
	colorNormal       = "^xC8C8C8"
)

// Misc is data.misc — the magic numbers. Fields marked (derived) are
// computed from the .ot constants at Load.
type misc struct {
	ServerTickTime                    float64 `lua:"ServerTickTime"`
	ServerTickRate                    float64 `lua:"ServerTickRate"`
	AccuracyPerDexBase                float64 `lua:"AccuracyPerDexBase"`
	LowPoolThreshold                  float64 `lua:"LowPoolThreshold"`
	TemporalChainsEffectCap           float64 `lua:"TemporalChainsEffectCap"`
	BuffExpirationSlowCap             float64 `lua:"BuffExpirationSlowCap"`
	DamageReductionCap                float64 `lua:"DamageReductionCap"`              // (derived)
	EnemyPhysicalDamageReductionCap   float64 `lua:"EnemyPhysicalDamageReductionCap"` // (derived)
	ResistFloor                       float64 `lua:"ResistFloor"`
	MaxResistCap                      float64 `lua:"MaxResistCap"`
	EvadeChanceCap                    float64 `lua:"EvadeChanceCap"`
	DodgeChanceCap                    float64 `lua:"DodgeChanceCap"`
	BlockChanceCap                    float64 `lua:"BlockChanceCap"`
	SuppressionChanceCap              float64 `lua:"SuppressionChanceCap"`
	SuppressionEffect                 float64 `lua:"SuppressionEffect"`
	AvoidChanceCap                    float64 `lua:"AvoidChanceCap"`
	FortifyBaseDuration               float64 `lua:"FortifyBaseDuration"`
	ManaRegenBase                     float64 `lua:"ManaRegenBase"` // (derived)
	EnergyShieldRechargeBase          float64 `lua:"EnergyShieldRechargeBase"`
	EnergyShieldRechargeDelay         float64 `lua:"EnergyShieldRechargeDelay"`
	WardRechargeDelay                 float64 `lua:"WardRechargeDelay"`
	Transfiguration                   float64 `lua:"Transfiguration"`
	EnemyMaxResist                    float64 `lua:"EnemyMaxResist"` // (derived)
	LeechRateBase                     float64 `lua:"LeechRateBase"`
	DotDpsCap                         float64 `lua:"DotDpsCap"`
	BleedPercentBase                  float64 `lua:"BleedPercentBase"`
	BleedDurationBase                 float64 `lua:"BleedDurationBase"`
	PoisonPercentBase                 float64 `lua:"PoisonPercentBase"`
	PoisonDurationBase                float64 `lua:"PoisonDurationBase"`
	IgnitePercentBase                 float64 `lua:"IgnitePercentBase"`
	IgniteDurationBase                float64 `lua:"IgniteDurationBase"`
	ImpaleStoredDamageBase            float64 `lua:"ImpaleStoredDamageBase"`
	TrapTriggerRadiusBase             float64 `lua:"TrapTriggerRadiusBase"`
	MineDetonationRadiusBase          float64 `lua:"MineDetonationRadiusBase"`
	MineAuraRadiusBase                float64 `lua:"MineAuraRadiusBase"`
	BrandAttachmentRangeBase          float64 `lua:"BrandAttachmentRangeBase"`
	ProjectileDistanceCap             float64 `lua:"ProjectileDistanceCap"`
	PlayerMovementSpeed               float64 `lua:"PlayerMovementSpeed"` // (derived)
	MinStunChanceNeeded               float64 `lua:"MinStunChanceNeeded"`
	StunBaseMult                      float64 `lua:"StunBaseMult"`
	StunBaseDuration                  float64 `lua:"StunBaseDuration"`
	StunNotMeleeDamageMult            float64 `lua:"StunNotMeleeDamageMult"`
	MaxEnemyLevel                     float64 `lua:"MaxEnemyLevel"`
	MaxExperiencePenaltyFreeAreaLevel float64 `lua:"maxExperiencePenaltyFreeAreaLevel"`
	ExperiencePenaltyMultiplier       float64 `lua:"experiencePenaltyMultiplier"`
	StdBossDPSMult                    float64 `lua:"stdBossDPSMult"`
	PinnacleBossDPSMult               float64 `lua:"pinnacleBossDPSMult"`
	PinnacleBossPen                   float64 `lua:"pinnacleBossPen"`
	UberBossDPSMult                   float64 `lua:"uberBossDPSMult"`
	UberBossPen                       float64 `lua:"uberBossPen"`
	EhpCalcSpeedUp                    float64 `lua:"ehpCalcSpeedUp"`
	EhpCalcMaxDamage                  float64 `lua:"ehpCalcMaxDamage"`
	EhpCalcMaxIterationsToCalc        float64 `lua:"ehpCalcMaxIterationsToCalc"`
	MaxHitSmoothingPasses             float64 `lua:"maxHitSmoothingPasses"`
	MaxStatIncrease                   float64 `lua:"maxStatIncrease"`
	PvpElemental1                     float64 `lua:"PvpElemental1"`
	PvpElemental2                     float64 `lua:"PvpElemental2"`
	PvpNonElemental1                  float64 `lua:"PvpNonElemental1"`
	PvpNonElemental2                  float64 `lua:"PvpNonElemental2"`
	MatchingSocketQualityBonus        float64 `lua:"MatchingSocketQualityBonus"`
}

func miscTable(characterConstants, monsterConstants map[string]float64) misc {
	return misc{
		ServerTickTime: 0.033,
		// float64(...) forces the division to round through the double, as
		// the reference's runtime `1 / 0.033` does: Go would otherwise fold
		// the untyped constant expression at arbitrary precision and land
		// one ulp away, which a tick-rounding ceil can amplify.
		ServerTickRate:                  1 / float64(0.033),
		AccuracyPerDexBase:              2,
		LowPoolThreshold:                0.5,
		TemporalChainsEffectCap:         75,
		BuffExpirationSlowCap:           0.25,
		DamageReductionCap:              characterConstants["maximum_physical_damage_reduction_%"],
		EnemyPhysicalDamageReductionCap: monsterConstants["maximum_physical_damage_reduction_%"],
		ResistFloor:                     -200,
		MaxResistCap:                    90,
		EvadeChanceCap:                  95,
		DodgeChanceCap:                  75,
		BlockChanceCap:                  90,
		SuppressionChanceCap:            100,
		SuppressionEffect:               40,
		AvoidChanceCap:                  75,
		FortifyBaseDuration:             6,
		ManaRegenBase:                   characterConstants["mana_regeneration_rate_per_minute_%"] / 60 / 100,
		// #EVAL: archive parity — Data.lua writes this key twice in one table
		// constructor (the derived value, then 0.33); under LuaJIT the
		// derived value is the one that survives.
		EnergyShieldRechargeBase:          characterConstants["energy_shield_recharge_rate_per_minute_%"] / 60 / 100,
		EnergyShieldRechargeDelay:         2,
		WardRechargeDelay:                 2,
		Transfiguration:                   0.3,
		EnemyMaxResist:                    monsterConstants["base_maximum_all_resistances_%"],
		LeechRateBase:                     0.02,
		DotDpsCap:                         35791394, // (2 ^ 31 - 1) / 60 (int max / 60 seconds)
		BleedPercentBase:                  70,
		BleedDurationBase:                 5,
		PoisonPercentBase:                 0.30,
		PoisonDurationBase:                2,
		IgnitePercentBase:                 0.9,
		IgniteDurationBase:                4,
		ImpaleStoredDamageBase:            0.1,
		TrapTriggerRadiusBase:             10,
		MineDetonationRadiusBase:          60,
		MineAuraRadiusBase:                35,
		BrandAttachmentRangeBase:          30,
		ProjectileDistanceCap:             150,
		PlayerMovementSpeed:               characterConstants["base_speed"],
		MinStunChanceNeeded:               20,
		StunBaseMult:                      200,
		StunBaseDuration:                  0.35,
		StunNotMeleeDamageMult:            0.75,
		MaxEnemyLevel:                     85,
		MaxExperiencePenaltyFreeAreaLevel: 70,
		ExperiencePenaltyMultiplier:       0.06,
		// Expected values to calculate EHP
		StdBossDPSMult:      4 / 4.40,
		PinnacleBossDPSMult: 8 / 4.40,
		PinnacleBossPen:     15.0 / 5,
		UberBossDPSMult:     10 / 4.25,
		UberBossPen:         40.0 / 5,
		// ehp helper function magic numbers
		EhpCalcSpeedUp:             8,
		EhpCalcMaxDamage:           100000000,
		EhpCalcMaxIterationsToCalc: 50,
		MaxHitSmoothingPasses:      8,
		MaxStatIncrease:            2, // 100% increased
		// PvP scaling used for hogm
		PvpElemental1:              0.55,
		PvpElemental2:              150,
		PvpNonElemental1:           0.57,
		PvpNonElemental2:           90,
		MatchingSocketQualityBonus: 10,
	}
}

// PowerStat is one data.powerStatList entry. Transform ports the Lua
// transforms (present entries invert values or strip a leading "The ").
type PowerStat struct {
	Stat           *string       `lua:"stat"`
	Label          string        `lua:"label"`
	CombinedOffDef bool          `lua:"combinedOffDef,omitempty"`
	IgnoreForItems bool          `lua:"ignoreForItems,omitempty"`
	IgnoreForNodes bool          `lua:"ignoreForNodes,omitempty"`
	ReverseSort    bool          `lua:"reverseSort,omitempty"`
	ItemField      *string       `lua:"itemField"`
	Transform      func(any) any `lua:"transform"`
}

func str(s string) *string { return &s }

func negate(v any) any {
	if n, ok := v.(float64); ok {
		return -n
	}
	return v
}

func buildPowerStatList() []PowerStat {
	nameField := "Name"
	list := []PowerStat{
		{Stat: nil, Label: "Offence/Defence", CombinedOffDef: true, IgnoreForItems: true},
		{Stat: nil, Label: "Name", ItemField: &nameField, IgnoreForNodes: true, ReverseSort: true, Transform: func(v any) any {
			if s, ok := v.(string); ok && len(s) >= 4 && s[:4] == "The " {
				return s[4:]
			}
			return v
		}},
		{Stat: str("FullDPS"), Label: "Full DPS"},
		{Stat: str("CombinedDPS"), Label: "Combined DPS"},
		{Stat: str("TotalDPS"), Label: "Hit DPS"},
		{Stat: str("WithImpaleDPS"), Label: "Impale + Hit DPS"},
		{Stat: str("AverageDamage"), Label: "Average Hit"},
		{Stat: str("Speed"), Label: "Attack/Cast Speed"},
		{Stat: str("TotalDot"), Label: "DoT DPS"},
		{Stat: str("BleedDPS"), Label: "Bleed DPS"},
		{Stat: str("IgniteDPS"), Label: "Ignite DPS"},
		{Stat: str("PoisonDPS"), Label: "Poison DPS"},
		{Stat: str("Life"), Label: "Life"},
		{Stat: str("LifeRegen"), Label: "Life regen"},
		{Stat: str("LifeLeechRate"), Label: "Life leech"},
		{Stat: str("Armour"), Label: "Armour"},
		{Stat: str("Evasion"), Label: "Evasion"},
		{Stat: str("EnergyShield"), Label: "Energy Shield"},
		{Stat: str("EnergyShieldRecoveryCap"), Label: "Recoverable ES"},
		{Stat: str("EnergyShieldRegen"), Label: "Energy Shield regen"},
		{Stat: str("EnergyShieldLeechRate"), Label: "Energy Shield leech"},
		{Stat: str("Mana"), Label: "Mana"},
		{Stat: str("ManaRegen"), Label: "Mana regen"},
		{Stat: str("ManaLeechRate"), Label: "Mana leech"},
		{Stat: str("Ward"), Label: "Ward"},
		{Stat: str("Str"), Label: "Strength"},
		{Stat: str("Dex"), Label: "Dexterity"},
		{Stat: str("Int"), Label: "Intelligence"},
		{Stat: str("TotalAttr"), Label: "Total Attributes"},
		{Stat: str("MeleeAvoidChance"), Label: "Melee avoid chance"},
		{Stat: str("SpellAvoidChance"), Label: "Spell avoid chance"},
		{Stat: str("ProjectileAvoidChance"), Label: "Projectile avoid chance"},
		{Stat: str("TotalEHP"), Label: "Effective Hit Pool"},
		{Stat: str("SecondMinimalMaximumHitTaken"), Label: "Eff. Maximum Hit Taken"},
		{Stat: str("PhysicalTakenHit"), Label: "Taken Phys dmg", Transform: negate},
		{Stat: str("LightningTakenHit"), Label: "Taken Lightning dmg", Transform: negate},
		{Stat: str("ColdTakenHit"), Label: "Taken Cold dmg", Transform: negate},
		{Stat: str("FireTakenHit"), Label: "Taken Fire dmg", Transform: negate},
		{Stat: str("ChaosTakenHit"), Label: "Taken Chaos dmg", Transform: negate},
		{Stat: str("CritChance"), Label: "Crit Chance"},
		{Stat: str("CritMultiplier"), Label: "Crit Multiplier"},
		{Stat: str("BleedChance"), Label: "Bleed Chance"},
		{Stat: str("FreezeChance"), Label: "Freeze Chance"},
		{Stat: str("IgniteChance"), Label: "Ignite Chance"},
		{Stat: str("ShockChance"), Label: "Shock Chance"},
		{Stat: str("EffectiveMovementSpeedMod"), Label: "Move speed"},
		{Stat: str("LightRadiusMod"), Label: "Light Radius"},
		{Stat: str("BlockChance"), Label: "Block Chance"},
		{Stat: str("SpellBlockChance"), Label: "Spell Block Chance"},
		{Stat: str("SpellSuppressionChance"), Label: "Spell Suppression Chance"},
	}
	// these stats don't exist on minions or generally don't exist on both
	// player and minion
	minionNonApplicableStats := map[string]bool{
		"AverageDamage": true, "TotalDot": true, "Str": true, "Dex": true,
		"Int": true, "Spirit": true, "EffectiveLootRarityMod": true,
		"LightRadiusMod": true,
	}
	n := len(list)
	for i := 0; i < n; i++ {
		e := list[i]
		if e.Stat == nil || strings.Contains(*e.Stat, "DPS") || minionNonApplicableStats[*e.Stat] {
			continue
		}
		clone := e
		clone.Stat = str("Minion" + *e.Stat)
		clone.Label = "Minion " + e.Label
		list = append(list, clone)
	}
	return list
}

var cursePriority = map[string]int{
	"Temporal Chains":    1, // Despair and Elemental Weakness override Temporal Chains.
	"Enfeeble":           2, // Elemental Weakness and Vulnerability override Enfeeble.
	"Vulnerability":      3, // Despair and Elemental Weakness override Vulnerability. Vulnerability was reworked in 3.1.0.
	"Elemental Weakness": 4, // Despair and Flammability override Elemental Weakness.
	"Flammability":       5, // Frostbite overrides Flammability.
	"Frostbite":          6, // Conductivity overrides Frostbite.
	"Conductivity":       7,
	"Despair":            8, // Despair was created in 3.1.0.
	"Punishment":         9, // Punishment was reworked in 3.12.0.
	"Warlord's Mark":     10,
	"Assassin's Mark":    11,
	"Sniper's Mark":      12,
	"Poacher's Mark":     13,
	"SocketPriorityBase": 100,
	"Weapon 1":           1000,
	"Amulet":             2000,
	"Helmet":             3000,
	"Weapon 2":           4000,
	"Body Armour":        5000,
	"Gloves":             6000,
	"Boots":              7000,
	"Ring 1":             8000,
	"Ring 2":             9000,
	"Ring 3":             10000,
	"CurseFromEquipment": 11000,
	"CurseFromAura":      20000,
}

// keystones lists all keystones not exclusive to timeless or cluster jewels.
var keystones = []string{
	"Acrobatics", "Ancestral Bond", "Arrow Dancing", "Arsenal of Vengeance",
	"Avatar of Fire", "Bitter Frost", "Blood Magic", "Bloodsoaked Blade",
	"Call to Arms", "Chaos Inoculation", "Conduit", "Corrupted Soul",
	"Crimson Dance", "Divine Flesh", "Divine Shield", "Doomsday",
	"Eldritch Battery", "Elemental Equilibrium", "Elemental Overload",
	"Eternal Youth", "Ghost Dance", "Ghost Reaver", "Glancing Blows",
	"Hex Master", "Imbalanced Guard", "Immortal Ambition", "Inner Conviction",
	"Iron Grip", "Iron Reflexes", "Iron Will", "Lethe Shade", "Magebane",
	"Mind Over Matter", "Minion Instability", "Mortal Conviction",
	"Necromantic Aegis", "Pain Attunement", "Perfect Agony",
	"Phase Acrobatics", "Point Blank", "Power of Purpose",
	"Precise Technique", "Resolute Technique", "Roiling Tempest",
	"Runebinder", "Solipsism", "Supreme Decadence", "Supreme Ego",
	"The Agnostic", "The Impaler", "Transcendence", "Unwavering Stance",
	"Vaal Pact", "Versatile Combatant", "Voracious Flame", "Wicked Ward",
	"Wind Dancer", "Zealot's Oath",
}

var (
	ailmentTypeList             = []string{"Bleed", "Poison", "Ignite", "Chill", "Freeze", "Shock", "Scorch", "Brittle", "Sap"}
	elementalAilmentTypeList    = []string{"Ignite", "Chill", "Freeze", "Shock", "Scorch", "Brittle", "Sap"}
	nonDamagingAilmentTypeList  = []string{"Chill", "Freeze", "Shock", "Scorch", "Brittle", "Sap"}
	nonElementalAilmentTypeList = []string{"Bleed", "Poison"}
)

// Ailment is one data.nonDamagingAilment entry.
type Ailment struct {
	AssociatedType string   `lua:"associatedType"`
	Alt            bool     `lua:"alt"`
	Default        *float64 `lua:"default"`
	Min            float64  `lua:"min"`
	Max            float64  `lua:"max"`
	Precision      float64  `lua:"precision"`
	Duration       *float64 `lua:"duration"`
}

func num(v float64) *float64 { return &v }

var nonDamagingAilment = map[string]Ailment{
	"Chill":   {AssociatedType: "Cold", Alt: false, Default: num(10), Min: 5, Max: 30, Precision: 0, Duration: num(2)},
	"Freeze":  {AssociatedType: "Cold", Alt: false, Default: nil, Min: 0.3, Max: 3, Precision: 2, Duration: nil},
	"Shock":   {AssociatedType: "Lightning", Alt: false, Default: num(15), Min: 5, Max: 50, Precision: 0, Duration: num(2)},
	"Scorch":  {AssociatedType: "Fire", Alt: true, Default: num(10), Min: 0, Max: 30, Precision: 0, Duration: num(4)},
	"Brittle": {AssociatedType: "Cold", Alt: true, Default: num(2), Min: 0, Max: 6, Precision: 2, Duration: num(4)},
	"Sap":     {AssociatedType: "Lightning", Alt: true, Default: num(6), Min: 0, Max: 20, Precision: 0, Duration: num(4)},
}

// highPrecisionMods duplicates modstore's copy for now; modstore consumes
// this package's once game-data is wired through.
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

// WeaponTypeInfo is one data.weaponTypeInfo entry.
type weaponTypeDef struct {
	OneHand bool    `lua:"oneHand"`
	Melee   bool    `lua:"melee"`
	Flag    string  `lua:"flag"`
	Label   *string `lua:"label"`
}

var weaponTypeInfo = map[string]weaponTypeDef{
	"None":                       {OneHand: true, Melee: true, Flag: "Unarmed"},
	"Bow":                        {OneHand: false, Melee: false, Flag: "Bow"},
	"Claw":                       {OneHand: true, Melee: true, Flag: "Claw"},
	"Dagger":                     {OneHand: true, Melee: true, Flag: "Dagger"},
	"Staff":                      {OneHand: false, Melee: true, Flag: "Staff"},
	"Wand":                       {OneHand: true, Melee: false, Flag: "Wand"},
	"One Handed Axe":             {OneHand: true, Melee: true, Flag: "Axe"},
	"One Handed Mace":            {OneHand: true, Melee: true, Flag: "Mace"},
	"One Handed Sword":           {OneHand: true, Melee: true, Flag: "Sword"},
	"Sceptre":                    {OneHand: true, Melee: true, Flag: "Mace", Label: str("Sceptre")},
	"Thrusting One Handed Sword": {OneHand: true, Melee: true, Flag: "Sword", Label: str("One Handed Sword")},
	"Fishing Rod":                {OneHand: false, Melee: true, Flag: "Fishing"},
	"Two Handed Axe":             {OneHand: false, Melee: true, Flag: "Axe"},
	"Two Handed Mace":            {OneHand: false, Melee: true, Flag: "Mace"},
	"Two Handed Sword":           {OneHand: false, Melee: true, Flag: "Sword"},
}

// UnarmedWeapon is one data.unarmedWeaponData entry (keyed by class id).
type UnarmedWeapon struct {
	Type        string  `lua:"type"`
	AttackRate  float64 `lua:"AttackRate"`
	CritChance  float64 `lua:"CritChance"`
	PhysicalMin float64 `lua:"PhysicalMin"`
	PhysicalMax float64 `lua:"PhysicalMax"`
}

var unarmedWeaponData = map[int]UnarmedWeapon{
	0: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 6}, // Scion
	1: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 8}, // Marauder
	2: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 5}, // Ranger
	3: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 5}, // Witch
	4: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 6}, // Duelist
	5: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 6}, // Templar
	6: {Type: "None", AttackRate: 1.2, CritChance: 0, PhysicalMin: 2, PhysicalMax: 5}, // Shadow
}

// JewelRadius is one data.jewelRadii entry; the squared fields exist only on
// the tree version setJewelRadiiGlobally selected.
type jewelRadius struct {
	Inner        float64  `lua:"inner"`
	Outer        float64  `lua:"outer"`
	Col          string   `lua:"col"`
	Label        string   `lua:"label"`
	InnerSquared *float64 `lua:"innerSquared"`
	OuterSquared *float64 `lua:"outerSquared"`
}

// buildJewelRadii constructs data.jewelRadii and applies
// setJewelRadiiGlobally(latestTreeVersion) — the latest version is past
// 3.15, so 3_16 is selected and mutated.
func buildJewelRadii() (map[string][]*jewelRadius, float64) {
	radii := map[string][]*jewelRadius{
		"3_15": {
			{Inner: 0, Outer: 800, Col: "^xBB6600", Label: "Small"},
			{Inner: 0, Outer: 1200, Col: "^x66FFCC", Label: "Medium"},
			{Inner: 0, Outer: 1500, Col: "^x2222CC", Label: "Large"},
			{Inner: 850, Outer: 1100, Col: "^xD35400", Label: "Variable"},
			{Inner: 1150, Outer: 1400, Col: "^x66FFCC", Label: "Variable"},
			{Inner: 1450, Outer: 1700, Col: "^x2222CC", Label: "Variable"},
			{Inner: 1750, Outer: 2000, Col: "^xC100FF", Label: "Variable"},
			{Inner: 1750, Outer: 2000, Col: "^xC100FF", Label: "Variable"},
		},
		"3_16": {
			{Inner: 0, Outer: 960, Col: "^xBB6600", Label: "Small"},
			{Inner: 0, Outer: 1440, Col: "^x66FFCC", Label: "Medium"},
			{Inner: 0, Outer: 1800, Col: "^x2222CC", Label: "Large"},
			{Inner: 0, Outer: 2400, Col: "^xC100FF", Label: "Very Large"},
			{Inner: 0, Outer: 2880, Col: "^x0B9300", Label: "Massive"},
			{Inner: 960, Outer: 1320, Col: "^xD35400", Label: "Variable"},
			{Inner: 1320, Outer: 1680, Col: "^x66FFCC", Label: "Variable"},
			{Inner: 1680, Outer: 2040, Col: "^x2222CC", Label: "Variable"},
			{Inner: 2040, Outer: 2400, Col: "^xC100FF", Label: "Variable"},
			{Inner: 2400, Outer: 2880, Col: "^x0B9300", Label: "Variable"},
		},
	}
	maxRadius := 0.0
	for _, r := range radii["3_16"] {
		r.OuterSquared = num(r.Outer * r.Outer)
		r.InnerSquared = num(r.Inner * r.Inner)
		if r.Outer > maxRadius {
			maxRadius = r.Outer
		}
	}
	return radii, maxRadius
}

// EnchantmentSource is one data.enchantmentSource entry.
type enchantmentSource struct {
	Name  string `lua:"name"`
	Label string `lua:"label"`
}

var EnchantmentSource = []enchantmentSource{
	{Name: "ENKINDLING", Label: "Enkindling Orb"},
	{Name: "INSTILLING", Label: "Instilling Orb"},
	{Name: "RUNESMITH", Label: "Runecraft Bench"},
	{Name: "HEIST", Label: "Heist"},
	{Name: "HARVEST", Label: "Harvest"},
	{Name: "DEDICATION", Label: "Dedication to the Goddess"},
	{Name: "ENDGAME", Label: "Eternal Labyrinth"},
	{Name: "MERCILESS", Label: "Merciless Labyrinth"},
	{Name: "CRUEL", Label: "Cruel Labyrinth"},
	{Name: "NORMAL", Label: "Normal Labyrinth"},
}

var timelessJewelTypes = map[int]string{
	1: "Glorious Vanity", 2: "Lethal Pride", 3: "Brutal Restraint",
	4: "Militant Faith", 5: "Elegant Hubris", 6: "Heroic Tragedy",
	7: "Abyss Tecrod", 8: "Abyss Ulaman", 9: "Abyss Kurgal",
	10: "Abyss Amanamu", 11: "Abyss Zorath",
}

var timelessJewelSeedMin = map[int]float64{
	1: 100, 2: 10000, 3: 500, 4: 2000, 5: 2000.0 / 20,
	6: 100, 7: 100, 8: 100, 9: 100, 10: 100, 11: 100,
}

var timelessJewelSeedMax = map[int]float64{
	1: 8000, 2: 18000, 3: 8000, 4: 10000, 5: 160000.0 / 20,
	6: 8000, 7: 8000, 8: 8000, 9: 8000, 10: 8000, 11: 8000,
}

// itemTagSpecial: manually seeded modifier tag against item slot table for
// Mastery Item Condition based modifiers.
var itemTagSpecial = map[string]map[string][]string{
	"life": {
		"body armour": {
			// Keystone
			"Blood Magic", "Eternal Youth", "Ghost Reaver", "Mind Over Matter",
			"The Agnostic", "Vaal Pact", "Zealot's Oath",
			// Special Cases
			"^Cannot Leech$",
		},
	},
	"evasion": {
		"ring": {
			// Delve
			"chance to Evade",
			// Unique
			"Cannot Evade",
		},
	},
	"defence": {},
}

var itemTagSpecialExclusionPattern = map[string]map[string][]string{
	"life": {
		"amulet": {
			"lower Life on Hit", "your Spectres' Life",
			"when on Full Life", "when on Low Life", "^Allocates",
		},
		"body armour": {
			"Life as Physical Damage", "Life as Extra Maximum Energy Shield",
			"maximum Life as Fire Damage", "while on Full Life",
			"while you are on Full Life", "when on Full Life",
			"when on Low Life",
			"Gain Maximum Life instead of Maximum Energy Shield",
			"^Socketed Gems are Supported by Level", "^Allocates",
			"Void Spawns' Life",
		},
		"boots": {
			"Enemy's Life", "^Enemies Cannot Leech Life",
			"their Life as Chaos Damage", "when on Full Life",
			"when on Low Life", "^Allocates",
		},
		"belt": {
			"Life as Extra Maximum Energy Shield",
			"Life Recovery from Flasks is applied to nearby Allies",
			"Life Flasks gain", "when on Full Life", "when on Low Life",
			"^Allocates",
		},
		"gloves": {
			"maximum Life as Physical Damage", "Traps Cost Life",
			"when on Full Life", "when on Low Life", "^Allocates",
		},
		"helmet": {
			"Recouped as Life", "Life when you Suppress",
			"Leech when on Low Life", "while no Life is Reserved",
			"^Socketed Gems are Supported by Level", "when on Full Life",
			"when on Low Life", "^Allocates",
		},
		"ring 1": {
			"Energy Shield instead of Life",
			"increased Damage while Leeching Life", "when on Full Life",
			"when on Low Life", "^Allocates",
		},
		"ring 2": {
			"Energy Shield instead of Life",
			"increased Damage while Leeching Life", "when on Full Life",
			"when on Low Life", "^Allocates",
		},
		"weapon 1": {
			"^Socketed Gems are Supported by Level",
			"maximum Life as Chaos Damage",
			"total Maximum Life and Energy Shield as Fire Damage",
			"Life as Physical Damage", "maximum Life as Fire Damage",
			"when on Full Life", "when on Low Life", "^Allocates",
		},
		"weapon 2": {
			"maximum Life as Chaos Damage",
			"^Socketed Gems Cost and Reserve Life",
			"increased Damage while Leeching Life",
			"Life as Physical Damage", "maximum Life as Fire Damage",
			"when on Full Life", "when on Low Life", "^Allocates",
		},
	},
	"evasion": {
		"ring": {},
	},
	"defence": {},
}

var casterTagCrucibleUniques = map[string]bool{
	"Atziri's Rule": true, "Cane of Kulemak": true, "Cane of Unravelling": true,
	"Cospri's Malice": true, "Cybil's Paw": true, "Disintegrator": true,
	"Duskdawn": true, "Geofri's Devotion": true, "Mjolner": true,
	"Pledge of Hands": true, "Soulwrest": true, "Taryn's Shiver": true,
	"The Rippling Thoughts": true, "The Surging Thoughts": true,
	"The Whispering Ice": true, "Tremor Rod": true, "Xirgil's Crank": true,
}

var minionTagCrucibleUniques = map[string]bool{
	"Arakaali's Fang": true, "Ashcaller": true, "Chaber Cairn": true,
	"Chober Chaber": true, "Clayshaper": true, "Earendel's Embrace": true,
	"Femurs of the Saints": true, "Jorrhast's Blacksteel": true,
	"Law of the Wilds": true, "Midnight Bargain": true,
	"Mon'tregul's Grasp": true, "Null's Inclination": true,
	"Queen's Decree": true, "Queen's Escape": true,
	"Replica Earendel's Embrace": true, "Replica Midnight Bargain": true,
	"Severed in Sleep": true, "Soulwrest": true, "The Black Cane": true,
	"The Iron Mass": true, "The Scourge": true, "United in Dream": true,
}
