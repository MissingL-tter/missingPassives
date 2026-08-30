// parseRawTail: Item.lua L1337-1606 (everything after the line loop, up to
// and including the trailing BuildModList).
package item

import (
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

var foulbornPrefixRe = regexp.MustCompile(`^[Ff]oulborn `)
var foulbornAnyRe = regexp.MustCompile(`[Ff]oulborn `)

// foulbornEntry reads data.foulbornMap[title] as orig-mod-id -> foul-mod-id.
func foulbornEntry(title string) map[string]string {
	return data.FoulbornMap[title]
}

func clusterJewelFor(baseName string) *data.ClusterJewelSize {
	if jewel, ok := data.ClusterJewels.Jewels[baseName]; ok {
		j := jewel
		return &j
	}
	return nil
}

func (it *Item) parseRawTail(implicitLines int, importedLevelReq *float64, highQuality bool, deferJewelRadiusIndexAssignment bool) {
	if it.AdvancedCopy && (it.Rarity == "UNIQUE" || it.Rarity == "RELIC") {
		exact, normalised := uniqueStatOrder()
		for _, modLine := range it.ExplicitModLines {
			exactLine := strings.ReplaceAll(strings.ToLower(modLine.Line), "\n", " ")
			if order, ok := exact[exactLine]; ok {
				o := order
				modLine.Order = &o
			} else if order, ok := normalised[normaliseModLine(modLine.Line)]; ok {
				o := order
				modLine.Order = &o
			} else {
				modLine.Order = nil
			}
		}
	}
	if it.Base != nil && len(it.Sockets) > 0 {
		// In-game totals include socketed gems; rebuild from the base.
		it.Requirements.Str = util.Some(reqOrZero(it.Base.Req.Str))
		it.Requirements.Dex = util.Some(reqOrZero(it.Base.Req.Dex))
		it.Requirements.Int = util.Some(reqOrZero(it.Base.Req.Int))
	}
	if it.Base != nil && !it.Requirements.Level.Set {
		if importedLevelReq != nil && len(it.Sockets) == 0 {
			it.Requirements.Level = util.Some(*importedLevelReq)
		} else if it.Base.Req.Level != nil {
			it.Requirements.Level = util.Some(*it.Base.Req.Level)
		}
	}
	// Cane of Kulemak hack.
	if strings.ToLower(it.Title) == "cane of kulemak" && (it.Rarity == "UNIQUE" || it.Rarity == "RELIC") && it.AdvancedCopy {
		for _, mod := range it.ExplicitModLines {
			if !strings.Contains(mod.Line, "magnitude") {
				mod.setFlag("unveiled")
			}
		}
	}
	if it.AdvancedCopy || it.Crafted {
		for _, magnitudeMod := range it.modMagnitudeMods {
			var modLists [][]*ModLine
			if magnitudeMod.modType != "" {
				switch magnitudeMod.modType {
				case "implicit":
					modLists = [][]*ModLine{it.ImplicitModLines}
				case "explicit":
					modLists = [][]*ModLine{it.ExplicitModLines}
				case "enchant":
					modLists = [][]*ModLine{it.EnchantModLines}
				default:
					panic("item: unexpected modMagnitude modType " + magnitudeMod.modType)
				}
			} else {
				modLists = [][]*ModLine{it.ImplicitModLines, it.ExplicitModLines, it.EnchantModLines}
			}
			for _, mods := range modLists {
				for _, mod := range mods {
					if mod.VariantList != nil && it.GetModLineVariantCount(mod) == 0 {
						continue
					}
					tagLookup := map[string]bool{}
					for _, curTag := range mod.ModTags {
						tagLookup[curTag] = true
					}
					for _, lineFlag := range []string{"unveiled", "prefix", "suffix"} {
						if mod.flag(lineFlag) {
							tagLookup[lineFlag] = true
						}
					}
					match := true
					for _, magnitudeTag := range magnitudeMod.tags {
						if !tagLookup[magnitudeTag] {
							match = false
						}
					}
					if magnitudeMod.anyTags != nil && !(tagLookup[magnitudeMod.anyTags[0]] || tagLookup[magnitudeMod.anyTags[1]]) {
						match = false
					}
					if match && !mod.flag("unscalable") {
						vs := mod.ValueScalar.Or(1)
						if magnitudeMod.multiplier != nil {
							mod.ValueScalar = util.Some(vs * *magnitudeMod.multiplier)
						} else {
							mod.ValueScalar = util.Some(vs + magnitudeMod.quality/100)
						}
					}
					if mod.ValueScalar.Set && mod.ValueScalar.V != 1 {
						r := 1.0
						if mod.Range != nil {
							r = *mod.Range
						}
						rangedLine := applyRange(mod.Line, r, mod.ValueScalar.V, 1)
						modList, extra, parsed := parseModLine3(rangedLine)
						mod.ModList = modList
						mod.HasModList = parsed
						mod.Extra = extra
					}
				}
			}
		}
	}
	if it.AdvancedCopy && len(it.ExplicitModLines) > 1 {
		sortCraftedModLines(it.ExplicitModLines)
	}
	it.AffixLimit = 0
	if it.Crafted {
		if it.Affixes == nil {
			it.Crafted = false
		} else if it.Rarity == "MAGIC" {
			if it.Prefixes.Limit != nil || it.Suffixes.Limit != nil {
				p := fmax(fmin(limOrZero(it.Prefixes.Limit)+1, 2), 0)
				s := fmax(fmin(limOrZero(it.Suffixes.Limit)+1, 2), 0)
				it.Prefixes.Limit = &p
				it.Suffixes.Limit = &s
				it.AffixLimit = p + s
			} else {
				it.AffixLimit = 2
			}
		} else if it.Rarity == "RARE" {
			if (it.Type == "Jewel" && !(it.Base.SubType == "Abyss" && it.Corrupted)) || it.Type == "Graft" {
				it.AffixLimit = 4
			} else {
				it.AffixLimit = 6
			}
			if it.Prefixes.Limit != nil || it.Suffixes.Limit != nil {
				p := fmax(fmin(limOrZero(it.Prefixes.Limit)+it.AffixLimit/2, it.AffixLimit), 0)
				s := fmax(fmin(limOrZero(it.Suffixes.Limit)+it.AffixLimit/2, it.AffixLimit), 0)
				it.Prefixes.Limit = &p
				it.Suffixes.Limit = &s
				it.AffixLimit = p + s
			}
		} else if it.RareLikeUnique != nil {
			p := it.RareLikeUnique.PrefixLimit
			s := it.RareLikeUnique.SuffixLimit
			it.Prefixes.Limit = &p
			it.Suffixes.Limit = &s
			it.AffixLimit = p + s
		} else {
			it.Crafted = false
		}
		if it.Crafted {
			for _, list := range []*AffixList{&it.Prefixes, &it.Suffixes} {
				limit := it.AffixLimit / 2
				if list.Limit != nil {
					limit = *list.Limit
				}
				for i := 1; i <= int(limit); i++ {
					if i > len(list.List) {
						for len(list.List) < i {
							list.List = append(list.List, &Affix{ModID: "None"})
						}
					} else if list.List[i-1].ModID != "None" {
						if _, known := it.Affixes[list.List[i-1].ModID]; !known {
							for _, modID := range sortedAffixIDs(it.Affixes) {
								if list.List[i-1].ModID == it.Affixes[modID].Affix {
									list.List[i-1].ModID = modID
									break
								}
							}
							if _, known := it.Affixes[list.List[i-1].ModID]; !known {
								list.List[i-1].ModID = "None"
							}
						}
					}
				}
			}
		}
	}
	if it.Base != nil && it.Base.SocketLimit != nil && len(it.Sockets) == 0 {
		for i := 0; i < int(*it.Base.SocketLimit); i++ {
			it.Sockets = append(it.Sockets, &Socket{Color: it.DefaultSocketColor, Group: 0})
		}
	}
	it.AbyssalSocketCount = 0
	if it.UsesVariantGroups {
		it.NormaliseVariantSelections()
	} else if it.VariantList != nil {
		clampVariant := func(p *int) *int {
			v := len(it.VariantList)
			if p != nil {
				v = imin(len(it.VariantList), *p)
			}
			return &v
		}
		it.Variant = clampVariant(it.Variant)
		if it.HasAltVariant {
			it.VariantAlt = clampVariant(it.VariantAlt)
		}
		if it.HasAltVariant2 {
			it.VariantAlt2 = clampVariant(it.VariantAlt2)
		}
		if it.HasAltVariant3 {
			it.VariantAlt3 = clampVariant(it.VariantAlt3)
		}
		if it.HasAltVariant4 {
			it.VariantAlt4 = clampVariant(it.VariantAlt4)
		}
		if it.HasAltVariant5 {
			it.VariantAlt5 = clampVariant(it.VariantAlt5)
		}
	}
	if it.mutatedLines != nil {
		checkMod := func(modID, newModID string, mutated bool) {
			var originalMod data.ItemModData
			var ok bool
			if mutated {
				originalMod, ok = data.ItemMods["Foulborn"][modID]
			} else {
				originalMod, ok = data.ItemMods["ItemExclusive"][modID]
			}
			if !ok {
				// The reference just logs and returns.
				return
			}
			findMatchingLines := func(lines []string) []*ModLine {
				var matchingLines []*ModLine
				matchedLines := map[*ModLine]bool{}
				for _, line := range lines {
					statLine := normaliseModLine(line)
					for _, modLine := range it.ExplicitModLines {
						if !matchedLines[modLine] && normaliseModLine(modLine.Line) == statLine && it.CheckModLineVariant(modLine) {
							matchedLines[modLine] = true
							matchingLines = append(matchingLines, modLine)
							break
						}
					}
				}
				if len(lines) == len(matchingLines) {
					return matchingLines
				}
				return nil
			}
			matchingLines := findMatchingLines([]string{strings.Join(originalMod.Lines, " ")})
			if matchingLines == nil {
				matchingLines = findMatchingLines(originalMod.Lines)
			}
			for _, modLine := range matchingLines {
				modLine.ModID = modID
				modLine.NewModID = newModID
				if mutated {
					modLine.setFlag("mutated")
				}
			}
		}
		origIDs := make([]string, 0, len(it.mutatedLines))
		for k := range it.mutatedLines {
			origIDs = append(origIDs, k)
		}
		sort.Strings(origIDs)
		for _, origModID := range origIDs {
			foulModID := it.mutatedLines[origModID]
			checkMod(origModID, foulModID, false)
			checkMod(foulModID, origModID, true)
		}
	}
	for _, v := range it.ExplicitModLines {
		if v.flag("mutated") {
			it.Foulborn = true
		}
	}
	hasFoulbornPrefix := it.Title != "" && foulbornPrefixRe.MatchString(strings.ToLower(it.Title))
	if it.Foulborn && !hasFoulbornPrefix {
		it.Title = "Foulborn " + it.Title
	} else if !it.Foulborn && hasFoulbornPrefix {
		it.Title = foulbornAnyRe.ReplaceAllString(it.Title, "")
	}
	if it.BaseName != "" && it.Title != "" {
		it.Name = it.Title + ", " + nameParenRe.ReplaceAllString(it.BaseName, "")
	}
	if it.Quality == nil {
		it.NormaliseQuality()
		if highQuality {
			it.NormaliseQuality()
		}
	}
	if it.Base != nil && it.Base.Enchant != nil && len(it.EnchantModLines) == 0 {
		enchantIndex := 0
		for _, line := range strings.Split(*it.Base.Enchant, "\n") {
			if line == "" {
				continue
			}
			modList, extra := parseModLine(line)
			ml := &ModLine{
				Line: line, Extra: extra, ModList: modList, HasModList: true,
				ModTags: []string{},
			}
			ml.setFlag("crafted")
			ml.setFlag("implicit")
			ml.setFlag("enchant")
			if enchantIndex < len(it.Base.EnchantModTypes) {
				ml.ModTags = it.Base.EnchantModTypes[enchantIndex]
			}
			it.EnchantModLines = append(it.EnchantModLines, ml)
			enchantIndex++
		}
	}
	it.BuildModList()
	if deferJewelRadiusIndexAssignment {
		it.JewelRadiusIndex = nil
		if it.JewelData != nil && it.JewelData.RadiusIndex != 0 {
			idx := it.JewelData.RadiusIndex
			it.JewelRadiusIndex = &idx
		}
	}
}

func limOrZero(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}

func sortedAffixIDs(m map[string]data.ItemModData) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
