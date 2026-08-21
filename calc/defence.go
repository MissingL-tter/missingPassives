// Port of .archive/src/Modules/CalcDefence.lua, staged: resistances first,
// then calcs.defence. buildDefenceEstimations is a later stage.
package calc

import (
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
	d := env.Data

	output["PhysicalResist"] = 0.0

	// Process Resistance conversion mods
	for _, resFrom := range resistTypeList {
		maxRes := 0.0
		haveMaxRes := false
		for _, resTo := range resistTypeList {
			conversionRate := modDB.Sum("BASE", nil, resFrom+"MaxResConvertTo"+resTo) / 100
			if conversionRate != 0 {
				if !haveMaxRes {
					haveMaxRes = true
					for _, mod := range modDB.Tabulate("BASE", nil, resFrom+"ResistMax") {
						if mod.Mod.Source != "Base" {
							maxRes += anyNum(mod.Value)
						}
					}
				}
				if maxRes != 0 {
					modDB.AddMod(newMod(resTo+"ResistMax", "BASE", maxRes*conversionRate, resFrom+" To "+resTo+" Max Resistance Conversion"))
				}
			}
		}
	}

	for _, resFrom := range resistTypeList {
		res := 0.0
		haveRes := false
		for _, resTo := range resistTypeList {
			conversionRate := modDB.Sum("BASE", nil, resFrom+"ResConvertTo"+resTo) / 100
			if conversionRate != 0 {
				if !haveRes {
					haveRes = true
					for _, mod := range modDB.Tabulate("BASE", nil, resFrom+"Resist") {
						if mod.Mod.Source != "Base" {
							res += anyNum(mod.Value)
						}
					}
				}
				if res != 0 {
					modDB.AddMod(newMod(resTo+"Resist", "BASE", res*conversionRate, resFrom+" To "+resTo+" Resistance Conversion"))
				}
				for _, mod := range modDB.Tabulate("INC", nil, resFrom+"Resist") {
					modDB.AddMod(newMod(resTo+"Resist", "INC", anyNum(mod.Value)*conversionRate, mod.Mod.Source))
				}
				for _, mod := range modDB.Tabulate("MORE", nil, resFrom+"Resist") {
					modDB.AddMod(newMod(resTo+"Resist", "MORE", anyNum(mod.Value)*conversionRate, mod.Mod.Source))
				}
			}
		}
	}

	// resistMaxOf is the shared `Override or min(cap, Sum(...))` shape.
	resistMaxOf := func(prefix, elem, sharedName string) float64 {
		if ov := modDB.Override(nil, prefix+elem+"ResistMax"); truthy(ov) {
			return anyNum(ov)
		}
		return math.Min(d.Misc.MaxResistCap, modDB.Sum("BASE", nil, elemNames(elem, prefix+elem+"ResistMax", sharedName)...))
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
				modDB.AddMod(newMod(elem+"ResistMax", "OVERRIDE", highestResistMax, highestResistMaxType+" Melding of the Flesh"))
			}
		}
	}

	for _, elem := range resistTypeList {
		min := d.Misc.ResistFloor
		max := resistMaxOf("", elem, "ElementalResistMax")
		totemMax := resistMaxOf("Totem", elem, "TotemElementalResistMax")

		var total, dotTotal float64
		haveDot := false
		if ov := modDB.Override(nil, elem+"Resist"); truthy(ov) {
			total = anyNum(ov)
		} else {
			base := modDB.Sum("BASE", nil, elemNames(elem, elem+"Resist", "ElementalResist")...)
			inc := math.Max(Mod(modDB, nil, elemNames(elem, elem+"Resist", "ElementalResist")...), 0)
			total = base * inc
			dotCfg := &modstore.Cfg{Flags: i64p(modparser.ModFlag.Dot), KeywordFlags: i64p(0)}
			dotBase := modDB.Sum("BASE", dotCfg, elemNames(elem, elem+"Resist", "ElementalResist")...)
			dotTotal = dotBase * inc
			haveDot = true
		}
		var totemTotal float64
		if ov := modDB.Override(nil, "Totem"+elem+"Resist"); truthy(ov) {
			totemTotal = anyNum(ov)
		} else {
			base := modDB.Sum("BASE", nil, elemNames(elem, "Totem"+elem+"Resist", "TotemElementalResist")...)
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

		output[elem+"Resist"] = final
		output[elem+"ResistTotal"] = total
		output[elem+"ResistOverCap"] = math.Max(0, total-max)
		output[elem+"ResistOver75"] = math.Max(0, final-75)
		output["Missing"+elem+"Resist"] = math.Max(0, max-final)
		output[elem+"ResistOverTime"] = dotFinal
		output["Totem"+elem+"Resist"] = totemFinal
		output["Totem"+elem+"ResistTotal"] = totemTotal
		output["Totem"+elem+"ResistOverCap"] = math.Max(0, totemTotal-totemMax)
		output["MissingTotem"+elem+"Resist"] = math.Max(0, totemMax-totemFinal)
	}
}

func i64p(v int64) *int64 { return &v }
