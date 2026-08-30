// Port of .archive/src/Modules/CalcDefence.lua, staged: resistances first,
// then calcs.defence. buildDefenceEstimations is a later stage.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

var resistTypeList = []string{"Fire", "Cold", "Lightning", "Chaos"}

var isElementalRes = map[string]bool{"Fire": true, "Cold": true, "Lightning": true}

// elemNames appends the shared Elemental* name when elem is elemental,
// mirroring the reference's `isElemental[elem] and "…"` vararg (a false
// there passes nil, which Sum skips).
func elemNames(elem, own, shared string) []string {
	if isElementalRes[elem] {
		return []string{own, shared}
	}
	return []string{own}
}

// Resistances ports calcs.resistances.
func (env *Env) Resistances(actor *performActor) {
	modDB := actor.db
	output := actor.output

	output.SetN("PhysicalResist", 0.0)

	// Process Resistance conversion mods
	for _, resFrom := range resistTypeList {
		maxRes := 0.0
		haveMaxRes := false
		for _, resTo := range resistTypeList {
			conversionRate := modDB.Sum(modparser.Base, nil, resFrom+"MaxResConvertTo"+resTo) / 100
			if conversionRate != 0 {
				if !haveMaxRes {
					haveMaxRes = true
					for _, mod := range modDB.Tabulate(modparser.Base, nil, resFrom+"ResistMax") {
						if mod.Mod.Source != "Base" {
							maxRes += valueNum(mod.Value)
						}
					}
				}
				if maxRes != 0 {
					modDB.AddMod(newModS(resTo+"ResistMax", modparser.Base, modparser.Num(maxRes*conversionRate), resFrom+" To "+resTo+" Max Resistance Conversion"))
				}
			}
		}
	}

	for _, resFrom := range resistTypeList {
		res := 0.0
		haveRes := false
		for _, resTo := range resistTypeList {
			conversionRate := modDB.Sum(modparser.Base, nil, resFrom+"ResConvertTo"+resTo) / 100
			if conversionRate != 0 {
				if !haveRes {
					haveRes = true
					for _, mod := range modDB.Tabulate(modparser.Base, nil, resFrom+"Resist") {
						if mod.Mod.Source != "Base" {
							res += valueNum(mod.Value)
						}
					}
				}
				if res != 0 {
					modDB.AddMod(newModS(resTo+"Resist", modparser.Base, modparser.Num(res*conversionRate), resFrom+" To "+resTo+" Resistance Conversion"))
				}
				for _, mod := range modDB.Tabulate(modparser.Inc, nil, resFrom+"Resist") {
					modDB.AddMod(newModS(resTo+"Resist", modparser.Inc, modparser.Num(valueNum(mod.Value)*conversionRate), mod.Mod.Source))
				}
				for _, mod := range modDB.Tabulate(modparser.More, nil, resFrom+"Resist") {
					modDB.AddMod(newModS(resTo+"Resist", modparser.More, modparser.Num(valueNum(mod.Value)*conversionRate), mod.Mod.Source))
				}
			}
		}
	}

	// resistMaxOf is the shared `Override or min(cap, Sum(...))` shape.
	resistMaxOf := func(prefix, elem, sharedName string) float64 {
		if ov, ok := modDB.Override(nil, prefix+elem+"ResistMax"); ok {
			return valueNum(ov)
		}
		return math.Min(data.Misc.MaxResistCap, modDB.Sum(modparser.Base, nil, elemNames(elem, prefix+elem+"ResistMax", sharedName)...))
	}

	// Highest Maximum Elemental Resistance for Melding of the Flesh
	if modDB.Flag(nil, "ElementalResistMaxIsHighestResistMax") {
		highestResistMax := 0.0
		highestResistMaxType := ""
		for _, elem := range resistTypeList {
			resistMax := resistMaxOf("", elem, "ElementalResistMax")
			if resistMax > highestResistMax && isElementalRes[elem] {
				highestResistMax = resistMax
				highestResistMaxType = elem
			}
		}
		for _, elem := range resistTypeList {
			if isElementalRes[elem] {
				modDB.AddMod(newModS(elem+"ResistMax", modparser.Override, modparser.Num(highestResistMax), highestResistMaxType+" Melding of the Flesh"))
			}
		}
	}

	for _, elem := range resistTypeList {
		min := data.Misc.ResistFloor
		max := resistMaxOf("", elem, "ElementalResistMax")
		totemMax := resistMaxOf("Totem", elem, "TotemElementalResistMax")

		var total, dotTotal float64
		haveDot := false
		if ov, ok := modDB.Override(nil, elem+"Resist"); ok {
			total = valueNum(ov)
		} else {
			base := modDB.Sum(modparser.Base, nil, elemNames(elem, elem+"Resist", "ElementalResist")...)
			inc := math.Max(Mod(modDB, nil, elemNames(elem, elem+"Resist", "ElementalResist")...), 0)
			total = base * inc
			dotCfg := &modstore.Cfg{Flags: flagp(modparser.FlagDot), KeywordFlags: keywordp(0)}
			dotBase := modDB.Sum(modparser.Base, dotCfg, elemNames(elem, elem+"Resist", "ElementalResist")...)
			dotTotal = dotBase * inc
			haveDot = true
		}
		var totemTotal float64
		if ov, ok := modDB.Override(nil, "Totem"+elem+"Resist"); ok {
			totemTotal = valueNum(ov)
		} else {
			base := modDB.Sum(modparser.Base, nil, elemNames(elem, "Totem"+elem+"Resist", "TotemElementalResist")...)
			totemTotal = base * math.Max(Mod(modDB, nil, elemNames(elem, "Totem"+elem+"Resist", "TotemElementalResist")...), 0)
		}

		// Fractional resistances are truncated
		total = math.Trunc(total)
		if haveDot {
			dotTotal = math.Trunc(dotTotal)
		} else {
			dotTotal = total
		}
		totemTotal = math.Trunc(totemTotal)
		min = math.Trunc(min)
		max = math.Trunc(max)
		totemMax = math.Trunc(totemMax)

		final := math.Max(math.Min(total, max), min)
		dotFinal := math.Max(math.Min(dotTotal, max), min)
		totemFinal := math.Max(math.Min(totemTotal, totemMax), min)

		output.SetN(elem+"Resist", final)
		output.SetN(elem+"ResistTotal", total)
		output.SetN(elem+"ResistOverCap", math.Max(0, total-max))
		output.SetN(elem+"ResistOver75", math.Max(0, final-75))
		output.SetN("Missing"+elem+"Resist", math.Max(0, max-final))
		output.SetN(elem+"ResistOverTime", dotFinal)
		output.SetN("Totem"+elem+"Resist", totemFinal)
		output.SetN("Totem"+elem+"ResistTotal", totemTotal)
		output.SetN("Totem"+elem+"ResistOverCap", math.Max(0, totemTotal-totemMax))
		output.SetN("MissingTotem"+elem+"Resist", math.Max(0, totemMax-totemFinal))
	}
}

func flagp(v modparser.ModFlag) *modparser.ModFlag            { return &v }
func keywordp(v modparser.KeywordFlag) *modparser.KeywordFlag { return &v }
