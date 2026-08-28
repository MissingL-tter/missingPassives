// Port of Generated.lua's buildTreeDependentUniques half: the uniques that
// need tree data (Forbidden Flame/Flesh from the class notables, Skin of
// the Lords and Impossible Escape from the keystones). The reference calls
// it at the end of the PassiveTree constructor; tree.Load mirrors that,
// passing plain name sets so data stays tree-independent.

package data

import (
	"sort"
	"strings"
)

// generatedBaseLen is where the load-time half of Uniques["generated"]
// ends; BuildTreeDependentUniques truncates back to it so repeated tree
// loads cannot duplicate the tree-dependent items.
var generatedBaseLen int

// skinExcludedKeystones is excludedPassiveKeystones (infinite-loop guards).
var skinExcludedKeystones = map[string]bool{
	"Chaos Inoculation": true,
	"Necromantic Aegis": true,
}

// TrimTreeDependentUniques restores Uniques["generated"] to its load-time
// (pre-tree) state — the state the archive game-data dump captured.
func TrimTreeDependentUniques() {
	Uniques["generated"] = Uniques["generated"][:generatedBaseLen]
}

// BuildTreeDependentUniques appends the tree-dependent generated uniques.
// classNotables is tree.ClassNotables; nativeKeystones the deduplicated
// names of keystones that are on the tree proper (not blighted, positioned).
func BuildTreeDependentUniques(classNotables map[string][]string, nativeKeystones []string) {
	Uniques["generated"] = Uniques["generated"][:generatedBaseLen]
	add := func(lines []string) {
		Uniques["generated"] = append(Uniques["generated"], strings.Join(lines, "\n"))
	}

	// ------------------------------------------- Forbidden Flame/Flesh --
	classList := make([]string, 0, len(classNotables))
	for className := range classNotables {
		if className != "alternate_ascendancies" {
			classList = append(classList, className)
		}
	}
	sort.Strings(classList)
	for _, name := range []string{"Flame", "Flesh"} {
		var forbidden []string
		forbidden = append(forbidden, "Rarity: UNIQUE")
		forbidden = append(forbidden, "Forbidden "+name)
		if name == "Flame" {
			forbidden = append(forbidden, "Crimson Jewel")
		} else {
			forbidden = append(forbidden, "Cobalt Jewel")
		}
		for _, className := range classList {
			notableTable := classNotables[className]
			sort.Strings(notableTable)
			for _, notableName := range notableTable {
				forbidden = append(forbidden, "Variant: ("+className+") "+notableName)
			}
		}
		if name == "Flame" {
			forbidden = append(forbidden, "Source: Drops from unique{The Searing Exarch}")
		} else {
			forbidden = append(forbidden, "Source: Drops from unique{The Eater of Worlds}")
		}
		forbidden = append(forbidden, "Limited to: 1")
		forbidden = append(forbidden, "Item Level: 83")
		index := 1
		other := "Flesh"
		if name == "Flesh" {
			other = "Flame"
		}
		for _, className := range classList {
			for _, notableName := range classNotables[className] {
				v := "{variant:" + luaNumString(float64(index)) + "}"
				forbidden = append(forbidden, v+"Requires Class "+className)
				forbidden = append(forbidden, v+"Allocates "+notableName+" if you have the matching modifier on Forbidden "+other)
				index++
			}
		}
		forbidden = append(forbidden, "Corrupted")
		add(forbidden)
	}

	// --------------------------------------------- Skin of the Lords --
	skinKeystones := make([]string, 0, len(nativeKeystones))
	for _, name := range nativeKeystones {
		if !skinExcludedKeystones[name] {
			skinKeystones = append(skinKeystones, name)
		}
	}
	sort.Strings(skinKeystones)
	skin := []string{
		"Skin of the Lords",
		"Simple Robe",
		"League: Breach",
		"Source: Upgraded from unique{Skin of the Loyal} using currency{Blessing of Chayula}",
	}
	for _, name := range skinKeystones {
		skin = append(skin, "Variant: "+name)
	}
	skin = append(skin, "Implicits: 0")
	skin = append(skin, "Sockets cannot be modified")
	skin = append(skin, "+2 to Level of Socketed Gems")
	skin = append(skin, "100% increased Global Defences")
	skin = append(skin, "You can only Socket Corrupted Gems in this item")
	for index, name := range skinKeystones {
		skin = append(skin, "{variant:"+luaNumString(float64(index+1))+"}"+name)
	}
	skin = append(skin, "Corrupted")
	add(skin)

	// --------------------------------------------- Impossible Escape --
	impossibleKeystones := append([]string{}, nativeKeystones...)
	sort.Strings(impossibleKeystones)
	impossible := []string{
		"Impossible Escape",
		"Viridian Jewel",
		"League: Sentinel",
		"Source: Drops from unique{The Maven} (Uber)",
		"Limited to: 1",
		"Radius: Small",
	}
	for _, name := range impossibleKeystones {
		impossible = append(impossible, "Variant: "+name)
	}
	impossible = append(impossible, "Variant: Everything (QoL Test Variant)")
	variantCount := len(impossibleKeystones) + 1
	for index, name := range impossibleKeystones {
		impossible = append(impossible, "{variant:"+luaNumString(float64(index+1))+","+luaNumString(float64(variantCount))+"}Passive Skills in radius of "+name+" can be allocated without being connected to your tree")
	}
	impossible = append(impossible, "Corrupted")
	add(impossible)
}
