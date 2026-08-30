// Port of .archive/src/Export/Scripts/mapUniqueToFoulborn.lua.

package export

import (
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "mapUniqueToFoulborn", Build: buildMapUniqueToFoulborn})
}

var foulbornIdFixer = strings.NewReplacer(
	"Amluet", "Amulet",
	"Botts", "Boots",
	"HelmetDex2", "DexHelmet2",
	"HelmetInt2", "IntHelmet2",
)

func buildMapUniqueToFoulborn(x *Ctx) (schema.Document, error) {
	if err := x.EnsureMods(); err != nil {
		return nil, err
	}
	// The Lua dofiles Scripts/flavourText.lua unconditionally.
	if _, err := buildFlavourText(x); err != nil {
		return nil, err
	}

	foulbornIds := make([]string, 0, len(x.modFoulborn))
	for id := range x.modFoulborn {
		foulbornIds = append(foulbornIds, id)
	}
	sort.Strings(foulbornIds)

	fm := schema.FoulbornMap{}
	for _, unique := range x.flavourEntries {
		re := regexp.MustCompile(regexp.QuoteMeta(unique.id) + "[A-Za-z]+")
		for _, modId := range foulbornIds {
			sanitized := foulbornIdFixer.Replace(modId)
			if re.MatchString(sanitized) {
				fm[unique.name] = append(fm[unique.name], strings.Join(x.modFoulborn[modId].lines, " "))
			}
		}
	}
	return fm, nil
}
