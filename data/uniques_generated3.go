// Generated.lua port, continued: That Which Was Taken and Replica
// Dragonfang's Flight.

package data

import (
	"regexp"
	"sort"
	"strings"
)

var (
	reCharmDigits   = regexp.MustCompile(`[0-9]+`)
	reCharmUpper    = regexp.MustCompile(`[A-Z]`)
	reTrailingTwo   = regexp.MustCompile(`2$`)
	reTrailingOne   = regexp.MustCompile(`1$`)
	reAltXYSuffix   = regexp.MustCompile(`Alt[XY]$`)
)

func (d *Data) buildThatWhichWasTaken() []string {
	out := []string{`
Item Class: Jewels
Rarity: Unique
That Which Was Taken
Crimson Jewel
League: Affliction
Has Alt Variant: true
Has Alt Variant Two: true
Has Alt Variant Three: true
Selected Variant: 82
Selected Alt Variant: 104
Selected Alt Variant Two: 106
Selected Alt Variant Three: 125
Variant: None
`[1:]}

	charms := d.ItemMods["JewelCharm"]
	var modIds []string
	for modId := range charms {
		if !reTrailingOne.MatchString(modId) {
			modIds = append(modIds, modId)
		}
	}
	sort.Strings(modIds)
	for _, modId := range modIds {
		v := abbreviateModId(modId)
		v = strings.ReplaceAll(v, "AnimalCharm", "")
		v = strings.ReplaceAll(v, "LIfe", "Life")
		v = strings.ReplaceAll(v, "OnHIt", "OnHit")
		v = reTrailingTwo.ReplaceAllString(v, "")
		v = strings.ReplaceAll(v, "New", "")
		v = reCharmUpper.ReplaceAllString(v, " $0")
		v = reCharmDigits.ReplaceAllString(v, " $0")
		v = strings.ReplaceAll(v, "_", "")
		v = strings.ReplaceAll(v, "E S", "ES")
		out = append(out, "Variant:"+v)
	}
	out = append(out, `Limited to: 1
Requirements:
Level: 48
Item Level: 86
`)
	index := 2
	for _, modId := range modIds {
		for _, line := range charms[modId].Lines {
			out = append(out, "{variant:"+itoa(index)+"}"+line)
		}
		index++
	}
	return out
}

// gemIsType covers the two calcLib.gemIsType queries the generated uniques
// make ("active skill" and "non-vaal").
func gemIsType(g *Gem, typ string) bool {
	switch typ {
	case "active skill":
		return g.Tags["grants_active_skill"] && !g.Tags["support"]
	case "non-vaal":
		return !g.Tags["vaal"]
	}
	panic("data: unhandled gemIsType " + typ)
}

func (d *Data) buildReplicaDragonfangsFlight() []string {
	mods := map[string]string{}
	for _, gem := range d.Gems {
		if !reAltXYSuffix.MatchString(gem.GrantedEffectId) && gemIsType(gem, "active skill") && gemIsType(gem, "non-vaal") {
			mods[gem.Name] = "+3 to Level of all " + gem.Name + " Gems"
		}
	}
	out := []string{
		"Replica Dragonfang's Flight\n\tOnyx Amulet\n\tSelected Variant: 2\n\tHas Alt Variant: true\n\tSelected Alt Variant: 3\n\tLevelReq: 56\n\t",
		"Variant: Pre 3.23.0\nVariant: Current\n",
	}
	type pair struct{ line, name string }
	var sorted []pair
	for name, line := range mods {
		sorted = append(sorted, pair{line, name})
	}
	sort.Slice(sorted, func(a, b int) bool {
		if sorted[a].line == sorted[b].line {
			return sorted[a].name < sorted[b].name
		}
		return sorted[a].line < sorted[b].line
	})
	for _, m := range sorted {
		out = append(out, "Variant: "+m.name)
	}
	out = append(out, `Implicits: 1
{tags:jewellery_attribute}+(10-16) to all Attributes
{tags:jewellery_resistance}{variant:1}+(10-15)% to all Elemental Resistances
{tags:jewellery_resistance}{variant:2}+(5-10)% to all Elemental Resistances
`)
	index := 3
	for _, m := range sorted {
		out = append(out, "{variant:"+itoa(index)+"}"+m.line)
		index++
	}
	out = append(out, `
{variant:1}(10-15)% increased Reservation Efficiency of Skills
{variant:2}(5-10)% increased Reservation Efficiency of Skills
{variant:1}Items and Gems have (10-15)% reduced Attribute Requirements
{variant:2}Items and Gems have (5-10)% reduced Attribute Requirements
`[1:])
	return out
}
