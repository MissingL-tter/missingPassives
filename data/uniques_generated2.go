// Generated.lua port, continued: Precursor's Emblem, The Balance of Terror,
// and the Watcher's Eye family.

package data

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type chargeModTier map[string]string

var enduranceChargeMods = []chargeModTier{
	{
		"Gain every second":  "Gain an Endurance Charge every second if you've been Hit Recently",
		"+1 Maximum":         "+1 to Maximum Endurance Charges",
		"Cannot be Stunned":  "You cannot be Stunned while at maximum Endurance Charges",
		"Vaal Pact":          "You have Vaal Pact while at maximum Endurance Charges",
		"Intimidate":         "Intimidate Enemies for 4 seconds on Hit with Attacks while at maximum Endurance Charges",
	},
	{
		"Block Attacks":               "1% Chance to Block Attack Damage per Endurance Charge",
		"Spell Suppression":           "1% chance to Suppress Spell Damage per Endurance Charge",
		"Chaos Res":                   "+4% to Chaos Resistance per Endurance Charge",
		"Fire as Chaos":               "Gain 1% of Fire Damage as Extra Chaos Damage per Endurance Charge",
		"Attack and Cast Speed":       "1% increased Attack and Cast Speed per Endurance Charge",
		"Regen. Life":                 "Regenerate 0.3% of Life per second per Endurance Charge",
		"Inc. Critical Strike Chance": "6% increased Critical Strike Chance per Endurance Charge",
	},
	{
		"Up to Max.":      "15% chance that if you would gain Endurance Charges, you instead gain up to your maximum number of Endurance Charges",
		"Duration":        "(20-40)% increased Endurance Charge Duration",
		"Movement Speed":  "1% increased Movement Speed per Endurance Charge",
		"Armour":          "6% increased Armour per Endurance Charge",
		"Add Fire Damage": "(7-9) to (13-14) Fire Damage per Endurance Charge",
		"Inc. Damage":     "5% increased Damage per Endurance Charge",
		"On Kill":         "10% chance to gain an Endurance Charge on Kill",
	},
}

var frenzyChargeMods = []chargeModTier{
	{
		"Gain on Hit":          "10% chance to gain a Frenzy Charge on Hit",
		"+1 Maximum":           "+1 to Maximum Frenzy Charges",
		"Flask Charge on Crit": "Gain a Flask Charge when you deal a Critical Strike while at maximum Frenzy Charges",
		"Iron Reflexes":        "You have Iron Reflexes while at maximum Frenzy Charges",
		"Onslaught":            "Gain Onslaught for 4 seconds on Hit while at maximum Frenzy Charges",
	},
	{
		"Block Attacks":               "1% Chance to Block Attack Damage per Frenzy Charge",
		"Spell Suppression":           "1% chance to Suppress Spell Damage per Frenzy Charge",
		"Accuracy Rating":             "10% increased Accuracy Rating per Frenzy Charge",
		"Cold as Chaos":               "Gain 1% of Cold Damage as Extra Chaos Damage per Frenzy Charge",
		"Attack and Cast Speed":       "1% increased Attack and Cast Speed per Frenzy Charge",
		"Regen. Life":                 "Regenerate 0.3% of Life per second per Frenzy Charge",
		"Inc. Critical Strike Chance": "6% increased Critical Strike Chance per Frenzy Charge",
	},
	{
		"Up to Max.":      "15% chance that if you would gain Frenzy Charges, you instead gain up to your maximum number of Frenzy Charges",
		"Duration":        "(20-40)% increased Frenzy Charge Duration",
		"Movement Speed":  "1% increased Movement Speed per Frenzy Charge",
		"Evasion":         "8% increased Evasion Rating per Frenzy Charge",
		"Add Cold Damage": "(6-8) to (12-13) Cold Damage per Frenzy Charge",
		"Inc. Damage":     "5% increased Damage per Frenzy Charge",
		"On Kill":         "10% chance to gain an Frenzy Charge on Kill",
	},
}

var powerChargeMods = []chargeModTier{
	{
		"Gain on Crit":             "20% chance to gain a Power Charge on Critical Strike",
		"+1 Maximum":               "+1 to Maximum Power Charges",
		"Arcane Surge with Spells": "Gain Arcane Surge on Hit with Spells while at maximum Power Charges",
		"Mind over Matter":         "You have Mind over Matter while at maximum Power Charges",
		"Additional Curse":         "You can apply an additional Curse while at maximum Power Charges",
	},
	{
		"Block Attacks":         "1% Chance to Block Attack Damage per Power Charge",
		"Spell Suppression":     "1% chance to Suppress Spell Damage per Power Charge",
		"Phys. Damage Red.":     "1% additional Physical Damage Reduction per Power Charge",
		"Lightning as Chaos":    "Gain 1% of Lightning Damage as Extra Chaos Damage per Power Charge",
		"Attack and Cast Speed": "1% increased Attack and Cast Speed per Power Charge",
		"Regen. Life":           "Regenerate 0.3% of Life per second per Power Charge",
		"Crit Strike Multi":     "+3% to Critical Strike Multiplier per Power Charge",
	},
	{
		"Up to Max.":           "15% chance that if you would gain Power Charges, you instead gain up to your maximum number of Power Charges",
		"Duration":             "(20-40)% increased Power Charge Duration",
		"Movement Speed":       "1% increased Movement Speed per Power Charge",
		"Energy Shield":        "3% increased Energy Shield per Power Charge",
		"Add Lightning Damage": "(1-2) to (18-20) Lightning Damage per Power Charge",
		"Inc. Damage":          "5% increased Damage per Power Charge",
		"On Kill":              "10% chance to gain an Power Charge on Kill",
	},
}

func sortedTierKeys(m chargeModTier) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (d *Data) buildPrecursorsEmblem() []string {
	out := []string{
		`Precursor's Emblem
{variant:1}Topaz Ring
{variant:2}Sapphire Ring
{variant:3}Ruby Ring
{variant:4}Two-Stone Ring (Cold/Lightning)
{variant:5}Two-Stone Ring (Fire/Lightning)
{variant:6}Two-Stone Ring (Fire/Cold)
{variant:7}Prismatic Ring
League: Delve
Source: Vendor Recipe
Variant: Topaz Ring
Variant: Sapphire Ring
Variant: Ruby Ring
Variant: Two-Stone Ring (Cold/Lightning)
Variant: Two-Stone Ring (Fire/Lightning)
Variant: Two-Stone Ring (Fire/Cold)
Variant: Prismatic Ring`,
	}
	types := []struct {
		prefix string
		mods   []chargeModTier
	}{
		{"Endurance - ", enduranceChargeMods},
		{"Frenzy - ", frenzyChargeMods},
		{"Power - ", powerChargeMods},
	}
	for _, t := range types {
		for _, mods := range t.mods {
			for _, desc := range sortedTierKeys(mods) {
				out = append(out, "Variant: "+t.prefix+desc)
			}
		}
	}
	out = append(out, `Selected Variant: 1
Has Alt Variant: true
Has Alt Variant Two: true
Has Alt Variant Three: true
LevelReq: 49
Implicits: 7
{tags:jewellery_resistance}{variant:1}+(20-30)% to Lightning Resistance
{tags:jewellery_resistance}{variant:2}+(20-30)% to Cold Resistance
{tags:jewellery_resistance}{variant:3}+(20-30)% to Fire Resistance
{tags:jewellery_resistance}{variant:4}+(12-16)% to Cold and Lightning Resistances
{tags:jewellery_resistance}{variant:5}+(12-16)% to Fire and Lightning Resistances
{tags:jewellery_resistance}{variant:6}+(12-16)% to Fire and Cold Resistances
{tags:jewellery_resistance}{variant:7}+(8-10)% to all Elemental Resistances
{tags:jewellery_attribute}{variant:1}+20 to Intelligence
{tags:jewellery_attribute}{variant:2}+20 to Dexterity
{tags:jewellery_attribute}{variant:3}+20 to Strength
{tags:jewellery_attribute}{variant:4}+20 to Strength and Intelligence
{tags:jewellery_attribute}{variant:5}+20 to Dexterity and Intelligence
{tags:jewellery_attribute}{variant:6}+20 to Strength and Dexterity
{tags:jewellery_attribute}{variant:7}+20 to all Attributes
{tags:jewellery_defense}5% increased maximum Energy Shield
{tags:life}5% increased maximum Life`)

	index := 8
	for _, t := range types {
		for tierIdx, mods := range t.mods {
			tier := float64(tierIdx + 1)
			for _, desc := range sortedTierKeys(mods) {
				mod := mods[desc]
				if rePctNumber.MatchString(mod) {
					mod = reAnyNumber.ReplaceAllStringFunc(mod, func(num string) string {
						n, _ := strconv.ParseFloat(num, 64)
						return "(" + num + "-" + luaNumString(n*tier) + ")"
					})
				} else if reRangePct.MatchString(mod) {
					mod = reRangeHigher.ReplaceAllStringFunc(mod, func(m string) string {
						parts := reRangeHigher.FindStringSubmatch(m)
						n, _ := strconv.ParseFloat(parts[2], 64)
						return parts[1] + luaNumString(n*tier) + ")"
					})
				} else if reAddedRange.MatchString(mod) {
					mod = reAddedParts.ReplaceAllStringFunc(mod, func(m string) string {
						parts := reAddedParts.FindStringSubmatch(m)
						h1, _ := strconv.ParseFloat(parts[2], 64)
						h2, _ := strconv.ParseFloat(parts[4], 64)
						return parts[1] + luaNumString(h1*tier) + parts[3] + luaNumString(h2*tier) + ")"
					})
				}
				out = append(out, "{variant:"+itoa(index)+"}{range:0}"+mod)
				index++
			}
		}
	}
	return out
}

var balanceOfTerrorMods = map[string]string{
	"Vulnerability: Double Damage":                            "(6-10)% chance to deal Double Damage if you've cast Vulnerability in the past 10 seconds",
	"Vulnerability: Unaffected by Bleeding":                   "You are Unaffected by Bleeding if you've cast Vulnerability in the past 10 seconds",
	"Enfeeble: Critical Strike Multiplier":                    "+(30-40)% to Critical Strike Multiplier if you've cast Enfeeble in the past 10 seconds",
	"Enfeeble: Take no Extra Crit Damage":                     "Take no Extra Damage from Critical Strikes if you've cast Enfeeble in the past 10 seconds",
	"Despair: Immune to Curses":                               "Immune to Curses if you've cast Despair in the past 10 seconds",
	"Despair: Inflict Withered":                               "Inflict Withered for 2 seconds on Hit if you've cast Despair in the past 10 seconds",
	"Punishment: Immune to Reflected Damage":                  "Immune to Reflected Damage if you've cast Punishment in the past 10 seconds",
	"Punishment: Intimidate":                                  "Intimidate Enemies on Hit if you've cast Punishment in the past 10 seconds",
	"Frostbite: Cold Exposure":                                "Cold Exposure on Hit if you've cast Frostbite in the past 10 seconds",
	"Frostbite: Unaffected by Freeze":                         "You are Unaffected by Freeze if you've cast Frostbite in the past 10 seconds",
	"Flammability: Fire Exposure":                             "Inflict Fire Exposure on Hit if you've cast Flammability in the past 10 seconds",
	"Flammability: Unaffected by Ignite":                      "You are Unaffected by Ignite if you've cast Flammability in the past 10 seconds",
	"Conductivity: Lightning Exposure":                        "Inflict Lightning Exposure on Hit if you've cast Conductivity in the past 10 seconds",
	"Conductivity: Unaffected by Shock":                       "You are Unaffected by Shock if you've cast Conductivity in the past 10 seconds",
	"Elemental Weakness: Immune to Exposure":                  "Immune to Exposure if you've cast Elemental Weakness in the past 10 seconds",
	"Elemental Weakness: Physical Damage as a Random Element": "Gain (30-40)% of Physical Damage as a Random Element if you've cast Elemental Weakness in the past 10 seconds",
	"Temporal Chains: Cooldown Recovery Rate":                 "(20-25)% increased Cooldown Recovery Rate if you've cast Temporal Chains in the past 10 seconds",
	"Temporal Chains: Action Speed":                           "Action Speed cannot be Slowed below Base Value if you've cast Temporal Chains in the past 10 seconds",
}

func buildBalanceOfTerror() []string {
	out := []string{
		"The Balance of Terror",
		"Cobalt Jewel",
		"League: Sanctum",
		"Source: Drops from unique{Lycia, Herald of the Scourge} in normal{The Beyond}",
		"Has Alt Variant: true",
		"Has Alt Variant Two: true",
		"Selected Alt Variant Two: 1",
		"Limited to: 1",
		"LevelReq: 56",
		"Variant: None",
	}
	names := make([]string, 0, len(balanceOfTerrorMods))
	for name := range balanceOfTerrorMods {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		out = append(out, "Variant: "+name)
	}
	out = append(out, "+(10-15)% to all Elemental Resistances")
	for i, name := range names {
		out = append(out, "{variant:"+itoa(i+2)+"}"+balanceOfTerrorMods[name])
	}
	return out
}

// watchersEyeLegacyMods carries the legacy variant table; LegacyRange is
// the replacement text of the Lua legacyMod gsub("%(.*%)", ...).
type watchersLegacy struct {
	Version     string
	LegacyRange string
	Rename      string
}

var watchersEyeLegacyMods = map[string]watchersLegacy{
	"ClarityManaAddedAsEnergyShield":              {Version: "3.12.0", LegacyRange: "(12-18)"},
	"ClarityReducedManaCost":                      {Version: "3.8.0"},
	"ClarityManaRecoveryRate":                     {Version: "3.12.0", LegacyRange: "(20-30)"},
	"DisciplineEnergyShieldRecoveryRate":          {Version: "3.12.0", LegacyRange: "(20-30)"},
	"MalevolenceDamageOverTimeMultiplier":         {Version: "3.8.0", LegacyRange: "(36-44)"},
	"MalevolenceLifeAndEnergyShieldRecoveryRate":  {Version: "3.12.0", LegacyRange: "(15-20)"},
	"PrecisionIncreasedCriticalStrikeMultiplier":  {Version: "3.12.0", LegacyRange: "(30-50)"},
	"VitalityDamageLifeLeech":                     {Version: "3.12.0", LegacyRange: "(1-1.5)"},
	"VitalityFlatLifeRegen":                       {Version: "3.12.0"},
	"VitalityLifeRecoveryRate":                    {Version: "3.12.0", LegacyRange: "(20-30)"},
	"WrathLightningDamageManaLeech":               {Version: "3.8.0"},
	"GraceChanceToDodge":                          {Rename: "Grace: Chance to Suppress Spells"},
	"HasteChanceToDodgeSpells":                    {Rename: "Haste: Chance to Suppress Spells"},
	"PurityOfFireTakePhysicalAsFire":              {Version: "3.25.0"},
	"PurityOfIceTakePhysicalAsIce":                {Version: "3.25.0"},
	"PurityOfLightningTakePhysicalAsLightning":    {Version: "3.25.0"},
	"PurityOfElementsTakePhysicalAsFire_":         {Version: "3.25.0"},
	"PurityOfElementsTakePhysicalAsCold":          {Version: "3.25.0"},
	"PurityOfElementsTakePhysicalAsLightning":     {Version: "3.25.0"},
	"PurityOfFireReducedReflectedFireDamage":      {},
	"PurityOfIceReducedReflectedColdDamage":       {},
	"PurityOfLightningReducedReflectedLightningDamage":                {},
	"MalevolenceSkillEffectDuration":                                  {},
	"ZealotryMaximumEnergyShieldPerSecondToMaximumEnergyShieldLeechRate": {},
	"MalevolenceColdDamageOverTimeMultiplier":                         {},
	"MalevolenceChaosNonAilmentDamageOverTimeMultiplier":              {},
}

var (
	reSublime  = regexp.MustCompile(`^SublimeVision`)
	reArbalist = regexp.MustCompile(`^SummonArbalist`)
	reTypeSpec = regexp.MustCompile(`([0-9]+)([A-Za-z]+)`)
)

func watchersVariantName(id string) string {
	v := abbreviateModId(id)
	v = rePurityHead.ReplaceAllString(v, "$0:")
	v = strings.ReplaceAll(v, "New", "")
	v = spaceCaps(v)
	v = strings.ReplaceAll(v, "_", "")
	v = strings.ReplaceAll(v, "E S", "ES")
	return v
}

func (d *Data) buildEyeFamily() (watchersEye, sublimeVision, voranasMarch, boundByDestiny []string) {
	watchersEye = []string{`
Watcher's Eye
Prismatic Jewel
Source: Drops from unique{The Elder} or unique{The Elder} (Uber)
Has Alt Variant: true
Has Alt Variant Two: true
Selected Variant: 5
Selected Alt Variant: 30
Selected Alt Variant Two: 1
`[1:]}
	sublimeVision = []string{`
Sublime Vision
Prismatic Jewel
Shaper Item
Source: Drops from unique{The Elder} (Uber Uber) or unique{The Shaper} (Uber)
Limited to: 1
`[1:]}
	voranasMarch = []string{`
Vorana's March
Runic Sabatons
League: Expedition
Source: Drops from unique{Olroth, Origin of the Fall} in normal{Expedition Logbook}
Has Alt Variant: true
Has Alt Variant Two: true
Has Alt Variant Three: true
Selected Variant: 24
Selected Alt Variant: 10
Selected Alt Variant Two: 11
Selected Alt Variant Three: 13
`[1:]}
	boundByDestiny = []string{`
Bound by Destiny
Prismatic Jewel
Source: Drops from unique{Incarnation of Neglect} or unique{Incarnation of Fear} or unique{Incarnation of Dread}
Limited to: 1
Has Alt Variant: true
Has Alt Variant Two: true
Selected Variant: 1
Selected Alt Variant: 19
Selected Alt Variant Two: 37
`[1:]}

	voranasMarch = append(voranasMarch, "Variant: None")
	watchersEye = append(watchersEye, "Variant: None")

	for _, mod := range d.UniqueMods["Watcher's Eye"] {
		if !reSublime.MatchString(mod.Id) && !reArbalist.MatchString(mod.Id) {
			variantName := watchersVariantName(mod.Id)
			if legacy, ok := watchersEyeLegacyMods[mod.Id]; ok {
				if legacy.Version != "" {
					watchersEye = append(watchersEye, "Variant:"+variantName+" (Pre "+legacy.Version+")")
				}
				if legacy.LegacyRange != "" {
					watchersEye = append(watchersEye, "Variant:"+variantName)
				}
				if legacy.Rename != "" {
					watchersEye = append(watchersEye, "Variant: "+legacy.Rename)
				}
			} else {
				watchersEye = append(watchersEye, "Variant:"+variantName)
			}
		} else if !reArbalist.MatchString(mod.Id) {
			variantName := spaceCaps(strings.Replace(mod.Id, "SublimeVision", "", -1))
			sublimeVision = append(sublimeVision, "Variant:"+variantName)
		} else {
			v := abbreviateModId(mod.Id)
			v = strings.ReplaceAll(v, "SummonArbalist", "")
			v = spaceCaps(v)
			v = strings.ReplaceAll(v, "_", "")
			v = strings.ReplaceAll(v, "Percent To ", "")
			v = strings.ReplaceAll(v, "Chance To ", "")
			v = strings.ReplaceAll(v, "Targets To ", "")
			v = reFor4Seconds.ReplaceAllString(v, "")
			v = strings.ReplaceAll(v, " Percent", "")
			v = strings.ReplaceAll(v, "Number Of ", "")
			voranasMarch = append(voranasMarch, "Variant:"+v)
		}
	}

	// Bound by Destiny, from its own pool sorted by (type, id).
	bbdPool := d.bbdPool
	type bbdEntry struct {
		id  string
		mod ItemModData
	}
	var bbdMods []bbdEntry
	for id := range bbdPool {
		bbdMods = append(bbdMods, bbdEntry{id: id})
	}
	sort.Slice(bbdMods, func(a, b int) bool {
		ta, tb := bbdPool[bbdMods[a].id].Type, bbdPool[bbdMods[b].id].Type
		if ta == tb {
			return bbdMods[a].id < bbdMods[b].id
		}
		return ta < tb
	})
	for i := range bbdMods {
		bbdMods[i].mod = bbdPool[bbdMods[i].id]
	}
	for _, e := range bbdMods {
		typePart := reTypeSpec.ReplaceAllString(e.mod.Type, "$2 $1:")
		idPart := abbreviateModId(e.id)
		if idx := strings.Index(idPart, e.mod.Type); idx >= 0 {
			idPart = idPart[:idx]
		}
		idPart = strings.ReplaceAll(idPart, "New", "")
		idPart = spaceCaps(idPart)
		idPart = strings.ReplaceAll(idPart, "_", "")
		idPart = strings.ReplaceAll(idPart, "E S", "ES")
		idPart = strings.ReplaceAll(idPart, "Velocity", "Speed")
		idPart = strings.ReplaceAll(idPart, "Permyriad", "")
		boundByDestiny = append(boundByDestiny, "Variant: "+typePart+idPart)
	}
	for i, e := range bbdMods {
		for _, line := range e.mod.Lines {
			boundByDestiny = append(boundByDestiny, "{variant:"+itoa(i+1)+"}"+line)
		}
	}

	watchersEye = append(watchersEye, `Limited to: 1
(4-6)% increased maximum Energy Shield
(4-6)% increased maximum Life
(4-6)% increased maximum Mana`)
	voranasMarch = append(voranasMarch, `Requires Level 69, 46 Str, 46 Dex, 46 Int
Has no Sockets
Triggers Level 20 Summon Arbalists when Equipped
25% increased Movement Speed`)

	iWE, iSV, iVM := 2, 1, 2
	for _, mod := range d.UniqueMods["Watcher's Eye"] {
		switch {
		case !reSublime.MatchString(mod.Id) && !reArbalist.MatchString(mod.Id):
			if legacy, ok := watchersEyeLegacyMods[mod.Id]; ok {
				if legacy.LegacyRange != "" {
					for _, line := range mod.Mod.Lines {
						watchersEye = append(watchersEye, "{variant:"+itoa(iWE)+"}"+reParenGroup.ReplaceAllString(line, legacy.LegacyRange))
					}
					iWE++
				}
				if legacy.Version != "" || legacy.Rename != "" {
					for _, line := range mod.Mod.Lines {
						watchersEye = append(watchersEye, "{variant:"+itoa(iWE)+"}"+line)
					}
					iWE++
				}
			} else {
				for _, line := range mod.Mod.Lines {
					watchersEye = append(watchersEye, "{variant:"+itoa(iWE)+"}"+line)
				}
				iWE++
			}
		case !reArbalist.MatchString(mod.Id):
			for _, line := range mod.Mod.Lines {
				sublimeVision = append(sublimeVision, "{variant:"+itoa(iSV)+"}"+line)
			}
			iSV++
		default:
			for _, line := range mod.Mod.Lines {
				voranasMarch = append(voranasMarch, "{variant:"+itoa(iVM)+"}"+line)
			}
			iVM++
		}
	}
	return watchersEye, sublimeVision, voranasMarch, boundByDestiny
}
