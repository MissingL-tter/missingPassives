// ParseRaw: Item.lua L406-1606. Raw item text -> parsed state, ending in
// BuildModList. Corpus-unreachable branches (the in-game advanced copy
// format and its affix matching) panic loudly.
package item

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

var (
	rarityLineRe    = regexp.MustCompile(`^Rarity: ([A-Za-z]+)`)
	specColonRe     = regexp.MustCompile(`^([A-Za-z ()]+:?): (.+)$`)
	specRequiresRe  = regexp.MustCompile(`^(Requires [A-Za-z]+) (.+)$`)
	qualityCatRe    = regexp.MustCompile(`Quality \(([A-Za-z ]+) Modifiers\)`)
	qualityPctRe    = regexp.MustCompile(`(\d+)%`)
	leadAlphaRe     = regexp.MustCompile(`^[A-Za-z ]+`)
	reminderRe      = regexp.MustCompile(`^\([A-Za-z]+`)
	reminderOfRe    = regexp.MustCompile(`^\(\d+%? of `)
	flaskLastsRe    = regexp.MustCompile(`^Lasts .+ Seconds$`)
	flaskConsumesRe = regexp.MustCompile(`^Consumes \d+ of \d+ Charges on use$`)
	flaskChargesRe  = regexp.MustCompile(`^Currently has \d+ Charges$`)
	tagRe           = regexp.MustCompile(`\{([A-Za-z]*):?([^}]*)\}`)
	parenFlagRe     = regexp.MustCompile(` \(([a-z]+)\)`)
	digitsRe        = regexp.MustCompile(`\d+`)
	rangeFindRe     = regexp.MustCompile(`\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	curBaseRe       = regexp.MustCompile(`(-?\d+\.?\d*)\((-?\d+\.?\d*)\)`)
	nameParenRe     = regexp.MustCompile(` \(.+\)`)
	variantVerRe    = regexp.MustCompile(`\{([0-9A-Za-z_]+)\}(.+)`)
	rangeSpecRe     = regexp.MustCompile(`\{range:([^}]+)\}(.+)`)
	modGroupRe      = regexp.MustCompile(`\{modGroup:([^}]+)\}`)
	prefixAllowRe   = regexp.MustCompile(` prefix modifiers? allowed`)
	suffixAllowRe   = regexp.MustCompile(` suffix modifiers? allowed`)
	prefixPlusRe    = regexp.MustCompile(`\+(\d+) prefix modifiers? allowed`)
	prefixMinusRe   = regexp.MustCompile(`-(\d+) prefix modifiers? allowed`)
	suffixPlusRe    = regexp.MustCompile(`\+(\d+) suffix modifiers? allowed`)
	suffixMinusRe   = regexp.MustCompile(`-(\d+) suffix modifiers? allowed`)
)

// modMagnitudePatterns are the three Lua patterns (as Go regexps) that
// detect modifier-magnitude lines; the third is the "are doubled" form.
var modMagnitudePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(\d+)% ([ir][ne][cd][ru][ec][ae][sd]e?d?) ?([a-zA-Z \t\n\v\f\r]*) modifier magnitudes`),
	regexp.MustCompile(`(\d+)% ([ir][ne][cd][ru][ec][ae][sd]e?d?) effect of ([sp][ur][fe]fix)es`),
	regexp.MustCompile(`([a-zA-Z \t\n\v\f\r]*) modifier magnitudes are doubled`),
}

var modMagnitudeTagMap = map[string]string{
	"defence":     "defences",
	"enchantment": "enchant",
}

// uniqueModStatOrder caches the ItemExclusive stat-order lookup the way the
// reference's file-local upvalue does.
var uniqueModStatOrder struct {
	once       sync.Once
	exact      map[string]float64
	normalised map[string]float64
}

func uniqueStatOrder() (map[string]float64, map[string]float64) {
	uniqueModStatOrder.once.Do(func() {
		exact := map[string]float64{}
		normalised := map[string]float64{}
		for _, mod := range data.ItemMods["ItemExclusive"] {
			for index, line := range mod.Lines {
				if index >= len(mod.StatOrder) {
					break
				}
				exactLine := strings.ReplaceAll(luaLower(line), "\n", " ")
				statLine := normaliseModLine(line)
				order := mod.StatOrder[index]
				if cur, ok := exact[exactLine]; !ok || order < cur {
					exact[exactLine] = order
				}
				if cur, ok := normalised[statLine]; !ok || order < cur {
					normalised[statLine] = order
				}
			}
		}
		uniqueModStatOrder.exact = exact
		uniqueModStatOrder.normalised = normalised
	})
	return uniqueModStatOrder.exact, uniqueModStatOrder.normalised
}

// sortCraftedModLines ports the file-local sortCraftedModLines (stable
// three-group ordering with statOrder inside the non-crafted groups).
func sortCraftedModLines(modLines []*ModLine) {
	sourceOrder := map[*ModLine]int{}
	for index, modLine := range modLines {
		sourceOrder[modLine] = index
	}
	group := func(m *ModLine) int {
		if m.flag("crafted") || m.flag("custom") {
			return 3
		}
		if m.flag("fractured") {
			return 1
		}
		return 2
	}
	sort.SliceStable(modLines, func(x, y int) bool {
		a, b := modLines[x], modLines[y]
		aGroup, bGroup := group(a), group(b)
		if aGroup != bGroup {
			return aGroup < bGroup
		}
		if aGroup < 3 && !ptrEqual(a.Order, b.Order) {
			return ordOrHuge(a.Order) < ordOrHuge(b.Order)
		}
		return sourceOrder[a] < sourceOrder[b]
	})
}

func ptrEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func ordOrHuge(p *float64) float64 {
	if p == nil {
		return mathHuge
	}
	return *p
}

var mathHuge = math.Inf(1)

// New ports the Item constructor: sanitiseText then ParseRaw. Direct
// ParseRaw callers (the XML load path, BuildAndParseRaw) skip the
// sanitise, exactly as the reference does.
func New(raw string, rarity string, highQuality bool) *Item {
	it := &Item{}
	it.ParseRaw(sanitiseText(raw), rarity, highQuality)
	return it
}

// ParseRaw ports ItemClass:ParseRaw. rarity == "" means the nil argument
// (the loader path); highQuality mirrors the third argument.
func (it *Item) ParseRaw(raw string, rarity string, highQuality bool) {
	it.Raw = raw
	it.Name = "?"
	it.NamePrefix = ""
	it.NameSuffix = ""
	it.Base = nil
	if rarity != "" {
		it.Rarity = rarity
	} else {
		it.Rarity = "UNIQUE"
	}
	it.Quality = nil
	rawLines := trimRawLines(raw)
	for i, line := range rawLines {
		rawLines[i] = escapeGGGString(line)
	}
	mode := "WIKI"
	if rarity != "" {
		mode = "GAME"
	}
	l := 0 // 0-based cursor over rawLines
	itemClass := ""
	if l < len(rawLines) {
		if strings.HasPrefix(rawLines[l], "Item Class:") {
			// The reference's gsub here is a no-op for the single-space
			// format every real line uses; itemClass keeps the whole line.
			itemClass = rawLines[l]
			l++
		}
		if l < len(rawLines) {
			if m := rarityLineRe.FindStringSubmatch(rawLines[l]); m != nil {
				mode = "GAME"
				upper := strings.ToUpper(m[1])
				if rarityColorCodes[upper] {
					it.Rarity = upper
				}
				if it.Rarity == "UNIQUE" || it.Rarity == "RELIC" {
					for _, line := range rawLines {
						if strings.Contains(line, "Foil Unique") {
							it.Rarity = "RELIC"
							if cm := regexp.MustCompile(`\((.*?)\)`).FindStringSubmatch(line); cm != nil {
								it.FoilType = cm[1]
							} else {
								it.FoilType = "Rainbow"
							}
							break
						}
					}
				}
				l++
			}
		}
	}
	if l < len(rawLines) {
		if rawLines[l] == "--------" {
			l++
		}
		if l < len(rawLines) {
			it.Name = rawLines[l]
		}
		unidentified := false
		for _, line := range rawLines {
			if line == "Unidentified" {
				unidentified = true
				break
			}
		}
		if !(it.Rarity == "NORMAL" || it.Rarity == "MAGIC" || unidentified) {
			l++
		}
	}
	checkSection := false
	it.Sockets = nil
	it.SocketColourAlwaysMatches = false
	it.ClassRequirementModLines = nil
	it.BuffModLines = nil
	it.EnchantModLines = nil
	it.ScourgeModLines = nil
	it.ImplicitModLines = nil
	it.ExplicitModLines = nil
	it.CrucibleModLines = nil
	it.AdvancedCopy = false
	it.modMagnitudeMods = nil
	implicitLines := 0
	it.VariantList = nil
	it.VersionList = nil
	it.AllowDuplicateVariants = false
	it.VariantGroups = map[int]map[int]map[int]bool{}
	if it.VariantGroupSelections == nil {
		it.VariantGroupSelections = map[int]int{}
	}
	it.UsesVariantGroups = false
	it.Prefixes = AffixList{}
	it.Suffixes = AffixList{}
	it.Requirements = map[string]float64{"str": 0, "dex": 0, "int": 0}
	it.BaseLines = map[string]*baseLine{}
	it.Foulborn = false
	it.Vestigial = false
	it.mutatedLines = nil
	it.RareLikeUnique = nil
	if it.Influence == nil {
		it.Influence = map[string]bool{}
	}
	var importedLevelReq *float64
	var flaskBuffLines map[string]bool
	deferJewelRadiusIndexAssignment := false
	gameModeStage := "FINDIMPLICIT"
	foundExplicit, foundImplicit := false, false
	linePrefix := ""
	linePostfix := ""

lineLoop:
	for l < len(rawLines) {
		line := rawLines[l]
		if line == "Veiled Prefix" || line == "Veiled Suffix" {
			it.Veiled = true
		}
		switch {
		case flaskBuffLines != nil && flaskBuffLines[line]:
			delete(flaskBuffLines, line)
		case line == "--------":
			linePrefix = ""
			linePostfix = ""
			checkSection = true
		case line == "Split":
			it.Split = true
		case line == "Mirrored":
			it.Mirrored = true
		case line == "Corrupted":
			it.Corrupted = true
		case line == "Fractured Item":
			it.Fractured = true
		case line == "Synthesised Item":
			it.Synthesised = true
		case strings.Contains(line, "Foil Unique"):
			if cm := regexp.MustCompile(`\((.*?)\)`).FindStringSubmatch(line); cm != nil {
				it.FoilType = cm[1]
			} else {
				it.FoilType = "Rainbow"
			}
		case influenceItemMap[line] != "":
			it.Influence[influenceItemMap[line]] = true
		case line == "Requirements:":
			// nothing to do
		case reminderRe.MatchString(line) || reminderOfRe.MatchString(line):
			// Reminder text: skip to the line ending with ")".
			for l < len(rawLines) && !strings.HasSuffix(rawLines[l], ")") {
				l++
			}
		case it.Base != nil && it.Base.Flask != nil &&
			(flaskLastsRe.MatchString(line) || flaskConsumesRe.MatchString(line) || flaskChargesRe.MatchString(line)):
			// In-game flask state; not modifier lines.
		case strings.HasPrefix(line, "{ "):
			panic("item: in-game advanced copy format unported (line: " + line + ")")
		default:
			line = linePrefix + line + linePostfix
			if checkSection {
				if gameModeStage == "IMPLICIT" {
					if foundImplicit {
						gameModeStage = "EXPLICIT"
						foundExplicit = true
					} else {
						gameModeStage = "FINDEXPLICIT"
					}
				} else if gameModeStage == "EXPLICIT" {
					gameModeStage = "DONE"
				} else if gameModeStage == "FINDIMPLICIT" && it.ItemLevel != nil &&
					!strings.Contains(line, " (implicit)") && !strings.Contains(line, " (enchant)") &&
					!strings.Contains(line, "Talisman Tier") {
					gameModeStage = "EXPLICIT"
					foundExplicit = true
				}
				checkSection = false
			}
			specName, specVal := "", ""
			if m := specColonRe.FindStringSubmatch(line); m != nil {
				specName, specVal = m[1], m[2]
				if specName == "Class:" {
					specName = "Requires Class"
				}
			} else if m := specRequiresRe.FindStringSubmatch(line); m != nil {
				specName, specVal = m[1], m[2]
			}
			if specName != "" {
				switch {
				case specName == "Unique ID":
					it.UniqueID = specVal
				case specName == "Item Level":
					it.ItemLevel = specToNumber(specVal)
				case specName == "Memory Strands":
					it.MemoryStrands = specToNumber(specVal)
				case specName == "Requires Class":
					it.ClassRestriction = specVal
				case qualityCatRe.MatchString(specName):
					if pm := qualityPctRe.FindStringSubmatch(specVal); pm != nil {
						it.CatalystQuality = specToNumber(pm[1])
					}
					descriptor := qualityCatRe.FindStringSubmatch(specName)[1]
					for i, d := range catalystDescriptorList {
						if descriptor == d {
							cat := i + 1
							it.Catalyst = &cat
						}
					}
				case specName == "Quality":
					it.Quality = specToNumber(specVal)
				case specName == "Sockets":
					group := 0.0
					for _, c := range []byte(specVal) {
						switch c {
						case 'R', 'G', 'B', 'W', 'A':
							it.Sockets = append(it.Sockets, &Socket{Color: string(c), Group: group})
						case ' ':
							group++
						}
					}
				case specName == "Radius" && it.Type == "Jewel":
					label := leadAlphaRe.FindString(specVal)
					it.JewelRadiusLabel = label
					if label == "Variable" {
						deferJewelRadiusIndexAssignment = true
					} else {
						for index, rad := range data.JewelRadius {
							if label == rad.Label {
								idx := index + 1 // 1-based, as the reference stores it
								it.JewelRadiusIndex = &idx
								break
							}
						}
					}
				case specName == "Limited to" && it.Type == "Jewel":
					it.Limit = specToNumber(specVal)
				case specName == "Version":
					it.VersionList = append(it.VersionList, specVal)
					it.UsesVariantGroups = true
				case specName == "Variant":
					if m := variantVerRe.FindStringSubmatch(specVal); m != nil {
						it.VariantList = append(it.VariantList, m[2])
					} else {
						it.VariantList = append(it.VariantList, specVal)
					}
				case specName == "Talisman Tier":
					it.TalismanTier = specToNumber(specVal)
				case specName == "Armour" || specName == "Evasion Rating" || specName == "Evasion" ||
					specName == "Energy Shield" || specName == "Ward":
					if specName == "Evasion Rating" {
						specName = "Evasion"
						if it.BaseName == "Two-Toned Boots (Armour/Energy Shield)" {
							it.BaseName = "Two-Toned Boots (Armour/Evasion)"
							it.Base = data.ItemBases[it.BaseName]
						}
					} else if specName == "Energy Shield" {
						specName = "EnergyShield"
						if it.BaseName == "Two-Toned Boots (Armour/Evasion)" {
							it.BaseName = "Two-Toned Boots (Evasion/Energy Shield)"
							it.Base = data.ItemBases[it.BaseName]
						}
					}
					if it.ArmourData == nil {
						it.ArmourData = map[string]any{}
					}
					if n := specToNumber(specVal); n != nil {
						it.ArmourData[specName] = *n
					} else {
						delete(it.ArmourData, specName)
					}
				case strings.Contains(specName, "BasePercentile"):
					if it.ArmourData == nil {
						it.ArmourData = map[string]any{}
					}
					if n := specToNumber(specVal); n != nil {
						it.ArmourData[specName] = *n
					} else {
						it.ArmourData[specName] = 0.0
					}
				case specName == "Requires Level":
					setReq(it.Requirements, "level", specToNumber(specVal))
				case specName == "Level":
					importedLevelReq = specToNumber(specVal)
				case specName == "LevelReq":
					setReq(it.Requirements, "level", specToNumber(specVal))
				case specName == "Has Alt Variant":
					it.HasAltVariant = true
				case specName == "Has Alt Variant Two":
					it.HasAltVariant2 = true
				case specName == "Has Alt Variant Three":
					it.HasAltVariant3 = true
				case specName == "Has Alt Variant Four":
					it.HasAltVariant4 = true
				case specName == "Has Alt Variant Five":
					it.HasAltVariant5 = true
				case specName == "Selected Version":
					it.SelectedVersion = specToInt(specVal)
					it.UsesVariantGroups = true
				case specName == "Selected Variant Group":
					if m := regexp.MustCompile(`^(\d+)\s*=\s*(\d+)$`).FindStringSubmatch(specVal); m != nil {
						groupID, _ := strconv.Atoi(m[1])
						variantID, _ := strconv.Atoi(m[2])
						it.VariantGroupSelections[groupID] = variantID
						it.UsesVariantGroups = true
					}
				case specName == "Selected Variant":
					it.Variant = specToInt(specVal)
				case specName == "Selected Alt Variant":
					it.VariantAlt = specToInt(specVal)
				case specName == "Selected Alt Variant Two":
					it.VariantAlt2 = specToInt(specVal)
				case specName == "Selected Alt Variant Three":
					it.VariantAlt3 = specToInt(specVal)
				case specName == "Selected Alt Variant Four":
					it.VariantAlt4 = specToInt(specVal)
				case specName == "Selected Alt Variant Five":
					it.VariantAlt5 = specToInt(specVal)
				case specName == "Allow Duplicate Variants":
					it.AllowDuplicateVariants = specVal == "true"
				case specName == "Has Variants" || specName == "Selected Variants":
					l++ // legacy Watcher's Eye storage: skip the next line
				case specName == "League":
					it.League = specVal
				case specName == "Crafted":
					it.Crafted = true
				case specName == "Scourge":
					it.Scourge = true
				case specName == "Crucible":
					it.Crucible = true
				case specName == "Implicit":
					it.Implicit = true
				case specName == "Prefix" || specName == "Suffix":
					affixes := &it.Suffixes
					if specName == "Prefix" {
						affixes = &it.Prefixes
					}
					fractured := strings.HasPrefix(specVal, "{fractured}")
					specVal = strings.TrimPrefix(specVal, "{fractured}")
					var rng any
					affix := specVal
					if m := rangeSpecRe.FindStringSubmatch(specVal); m != nil {
						rangeStr, aff := m[1], m[2]
						affix = aff
						if strings.Contains(rangeStr, ",") {
							var ranges []float64
							for _, value := range strings.Split(rangeStr, ",") {
								if n, err := strconv.ParseFloat(value, 64); err == nil {
									ranges = append(ranges, n)
								}
							}
							rng = ranges
						} else if n, err := strconv.ParseFloat(rangeStr, 64); err == nil {
							rng = n
						}
					}
					if rng == nil && affix != "None" {
						rng = DefaultItemAffixQuality
					}
					affixes.List = append(affixes.List, &Affix{ModID: affix, Range: rng, Fractured: fractured})
				case specName == "Implicits":
					implicitLines = 0
					if n := specToNumber(specVal); n != nil {
						implicitLines = int(*n)
					}
					gameModeStage = "EXPLICIT"
				case specName == "Unreleased":
					it.Unreleased = specVal == "true"
				case specName == "Upgrade":
					it.UpgradePaths = append(it.UpgradePaths, specVal)
				case specName == "Source":
					it.Source = specVal
				case specName == "Cluster Jewel Skill":
					if it.ClusterJewel != nil {
						if _, ok := it.ClusterJewel.Skills[specVal]; ok {
							it.ClusterJewelSkill = specVal
						}
					}
				case specName == "Cluster Jewel Node Count":
					if it.ClusterJewel != nil {
						num := it.ClusterJewel.MaxNodes
						if n := specToNumber(specVal); n != nil {
							num = *n
						}
						v := fmin(fmax(num, it.ClusterJewel.MinNodes), it.ClusterJewel.MaxNodes)
						it.ClusterJewelNodeCount = &v
					}
				case specName == "Catalyst":
					for i, name := range catalystList {
						if specVal == name {
							cat := i + 1
							it.Catalyst = &cat
						}
					}
				case specName == "CatalystQuality":
					it.CatalystQuality = specToNumber(specVal)
				case specName == "Intangibility":
					it.Intangibility = specToNumber(specVal)
				case specName == "Note":
					it.Note = specVal
				case specName == "Str" || specName == "Strength" || specName == "Dex" || specName == "Dexterity" ||
					specName == "Int" || specName == "Intelligence":
					setReq(it.Requirements, luaLower(specName[:3]), specToNumber(specVal))
				case specName == "Critical Strike Range" || specName == "Attacks per Second" || specName == "Weapon Range" ||
					specName == "Critical Strike Chance" || specName == "Physical Damage" || specName == "Elemental Damage" ||
					specName == "Chaos Damage" || specName == "Chance to Block" || specName == "Block chance":
					it.HiddenSpecs = true
				// The reference writes this containment check with :match(),
				// so item text with pattern metacharacters is searched with
				// them interpreted ("20% reduced" looks for "20 reduced").
				// Deliberate divergence: plain containment ships here. No
				// shipped or corpus item distinguishes the two — the halves
				// of an unknown "Name: Value" line either sit verbatim in
				// the item's name (colon-named bases: "Contract: Repository",
				// "Maven's Invitation: X" — all metacharacter-free) or are
				// absent under both readings.
				case !(strings.Contains(it.Name, specName) && strings.Contains(it.Name, specVal)):
					foundExplicit = true
					gameModeStage = "EXPLICIT"
				}
			}
			if line == "Prefixes:" {
				foundExplicit = true
				gameModeStage = "EXPLICIT"
			}
			if specName == "" || foundExplicit || foundImplicit {
				modLine := &ModLine{ModTags: []string{}}
				line = gsubLimitFunc(line, tagRe.String(), -1, func(caps []string) string {
					k, val := caps[1], caps[2]
					switch {
					case k == "variant":
						modLine.VariantList = map[int]bool{}
						for _, varID := range digitsRe.FindAllString(val, -1) {
							n, _ := strconv.Atoi(varID)
							modLine.VariantList[n] = true
						}
					case k == "version":
						modLine.VersionList = map[int]bool{}
						for _, verID := range digitsRe.FindAllString(val, -1) {
							n, _ := strconv.Atoi(verID)
							modLine.VersionList[n] = true
						}
						it.UsesVariantGroups = true
					case k == "group":
						modLine.VariantGroupList = map[int]bool{}
						for _, groupID := range digitsRe.FindAllString(val, -1) {
							n, _ := strconv.Atoi(groupID)
							if n > 0 {
								modLine.VariantGroupList[n] = true
							}
						}
						it.UsesVariantGroups = true
					case k == "tags":
						for _, tag := range regexp.MustCompile(`[A-Za-z_]+`).FindAllString(val, -1) {
							modLine.ModTags = append(modLine.ModTags, tag)
						}
					case k == "modGroup":
						modLine.ModGroup = val
					case k == "range":
						it.AdvancedCopy = true
						if n, err := strconv.ParseFloat(val, 64); err == nil {
							modLine.Range = &n
						}
					case k == "corruptedRange":
						if n, err := strconv.ParseFloat(val, 64); err == nil {
							modLine.CorruptedRange = &n
						}
					case lineFlags[k]:
						modLine.setFlag(k)
					}
					return ""
				})
				if modLine.VariantGroupList != nil && modLine.VariantList != nil {
					for _, groupID := range boolKeys(modLine.VariantGroupList) {
						group := it.VariantGroups[groupID]
						if group == nil {
							group = map[int]map[int]bool{}
							it.VariantGroups[groupID] = group
						}
						for _, variantID := range boolKeys(modLine.VariantList) {
							if variantID >= 1 && variantID <= len(it.VariantList) {
								versions := group[variantID]
								if versions == nil {
									versions = map[int]bool{}
									group[variantID] = versions
								}
								if modLine.VersionList != nil {
									for _, versionID := range boolKeys(modLine.VersionList) {
										if versionID >= 1 && versionID <= len(it.VersionList) {
											versions[versionID] = true
										}
									}
								} else {
									versions[0] = true
								}
							}
						}
					}
				}
				line = gsubLimitFunc(line, parenFlagRe.String(), -1, func(caps []string) string {
					if lineFlags[caps[1]] {
						modLine.setFlag(caps[1])
					}
					return ""
				})
				if modLine.flag("enchant") {
					modLine.setFlag("crafted")
					modLine.setFlag("implicit")
				}
				if modLine.flag("vestigial") {
					it.Vestigial = true
				}
				baseName := ""
				if rangeFindRe.MatchString(line) {
					it.AdvancedCopy = true
				}
				if it.Base == nil && (it.Rarity == "NORMAL" || it.Rarity == "MAGIC") {
					if strings.Contains(it.Name, "Energy Blade") && itemClass != "" {
						if strings.Contains(itemClass, "One Hand") {
							it.Name = "Energy Blade One Handed"
						} else {
							it.Name = "Energy Blade Two Handed"
						}
					}
					if data.ItemBases[it.Name] != nil {
						baseName = it.Name
					} else {
						bestLength := -1
						bestS, bestE := 0, 0
						// pairs(data.itemBases) with a strictly-greater
						// keep: ties fall to iteration order. Scan sorted;
						// the differential owns any real tie.
						for _, itemBaseName := range sortedBaseNames() {
							s := strings.Index(it.Name, itemBaseName)
							if s >= 0 {
								e := s + len(itemBaseName) - 1
								if e-s > bestLength {
									baseName = itemBaseName
									bestLength = e - s
									bestS, bestE = s, e
								}
							}
						}
						if baseName != "" {
							it.NamePrefix = it.Name[:bestS]
							it.NameSuffix = it.Name[bestE+1:]
						}
					}
					if baseName == "" {
						if s := strings.Index(it.Name, "Two-Toned Boots"); s >= 0 {
							baseName = "Two-Toned Boots"
							it.NamePrefix = it.Name[:s]
							it.NameSuffix = it.Name[s+len("Two-Toned Boots"):]
						}
					}
					it.Name = nameParenRe.ReplaceAllString(it.Name, "")
				}
				if baseName == "" {
					baseName = strings.TrimPrefix(line, "Superior ")
					baseName = strings.TrimPrefix(baseName, "Synthesised ")
					baseName = strings.TrimPrefix(baseName, "Vestigial ")
				}
				if baseName == "Two-Toned Boots" {
					baseName = "Two-Toned Boots (Armour/Energy Shield)"
				}
				if base := data.ItemBases[baseName]; base != nil {
					it.BaseLines[baseName] = &baseLine{
						line:             baseName,
						variantList:      modLine.VariantList,
						versionList:      modLine.VersionList,
						variantGroupList: modLine.VariantGroupList,
					}
					if it.UsesVariantGroups && it.VersionList != nil && it.SelectedVersion == nil {
						sel := len(it.VersionList)
						it.SelectedVersion = &sel
					}
					baseMatches := false
					if it.UsesVariantGroups {
						baseMatches = it.CheckModLineVariant(modLine)
					} else {
						baseMatches = it.Variant == nil || modLine.VariantList == nil || modLine.VariantList[*it.Variant]
					}
					if baseMatches {
						it.BaseName = baseName
						if !(it.Rarity == "NORMAL" || it.Rarity == "MAGIC") {
							it.Title = it.Name
						}
						if it.Rarity == "UNIQUE" && it.Title != "" &&
							!strings.Contains(luaLower(it.Title), "might of the meek") {
							key := regexp.MustCompile(`^[Ff]oulborn `).ReplaceAllString(it.Title, "")
							it.mutatedLines = foulbornEntry(key)
						}
						it.Type = base.Type
						it.Base = base
						it.Affixes = affixPool(base)
						if it.Title != "" {
							if rlu, ok := data.RareLikeUniques[luaLower(it.Title)]; ok {
								it.RareLikeUnique = &rlu
								it.Affixes = rlu.Affixes
							}
						}
						it.Corruptible = base.Type != "Flask"
						it.CanBeInfluenced = base.InfluenceTags != nil
						it.ClusterJewel = clusterJewelFor(it.BaseName)
						it.Requirements["str"] = reqOrZero(base.Req.Str)
						it.Requirements["dex"] = reqOrZero(base.Req.Dex)
						it.Requirements["int"] = reqOrZero(base.Req.Int)
						maxReq := fmax(it.Requirements["str"], fmax(it.Requirements["dex"], it.Requirements["int"]))
						if maxReq == it.Requirements["dex"] {
							it.DefaultSocketColor = "G"
						} else if maxReq == it.Requirements["int"] {
							it.DefaultSocketColor = "B"
						} else {
							it.DefaultSocketColor = "R"
						}
						if base.Flask != nil && base.Flask.Buff != nil && flaskBuffLines == nil {
							flaskBuffLines = map[string]bool{}
							for _, buffLine := range base.Flask.Buff {
								flaskBuffLines[buffLine] = true
								modList, extra := parseModLine(buffLine)
								it.BuffModLines = append(it.BuffModLines, &ModLine{
									Line: buffLine, Extra: extra, ModList: modList, HasModList: true,
								})
							}
						}
						// base.tincture.buff: no tincture base carries buff
						// data (the Go TinctureData has no such field).
					}
					// Base lines don't need mod parsing.
					l++
					continue lineLoop
				}
				if modLine.flag("implicit") {
					foundImplicit = true
					gameModeStage = "IMPLICIT"
				}
				catalystScalar := 1.0
				if strings.HasSuffix(line, " - Unscalable Value") {
					line = strings.TrimSuffix(line, " - Unscalable Value")
					modLine.setFlag("unscalable")
				} else {
					catalystScalar = getCatalystScalar(it.Catalyst, modLine, it.CatalystQuality)
				}
				// Advanced copy current(base) fixed values -> keep current.
				line = gsubLimitFunc(line, curBaseRe.String(), -1, func(caps []string) string {
					return caps[1]
				})
				// The pendingAffixList branch only arises from the in-game
				// advanced copy format, which panics above. The plain path:
				{
					// Advanced copy enum ranges: strip "(a-b)" groups with
					// no digits.
					line = stripEnumRanges(line)
					// value(min-max) roll extraction: absent from
					// PoB-serialised text, so the loop below never matches
					// for the corpus; ported for completeness.
					bestPrecisionDelta := -1.0
					bestPrecisionRange := -1.0
					var firstRollRange *float64
					hasIndependentRolls := false
					advancedCopyLine := line
					for {
						m := regexp.MustCompile(`(-?\d+\.?\d*)\((-?\d+\.?\d*--?\d+\.?\d*)\)`).FindStringSubmatch(line)
						if m == nil {
							break
						}
						value, rangeStr := m[1], m[2]
						rm := regexp.MustCompile(`(-?\d+\.?\d*)-(-?\d+\.?\d*)`).FindStringSubmatch(rangeStr)
						minV, _ := strconv.ParseFloat(rm[1], 64)
						maxV, _ := strconv.ParseFloat(rm[2], 64)
						if minV > maxV {
							minV, maxV = maxV, minV
						}
						valueN, _ := strconv.ParseFloat(value, 64)
						delta := maxV - minV
						rollRange := 0.5
						if delta > 0 {
							rollRange = roundDec((valueN-minV)/delta, 6)
						}
						if firstRollRange != nil && *firstRollRange != rollRange {
							hasIndependentRolls = true
						}
						if firstRollRange == nil {
							firstRollRange = &rollRange
						}
						if delta > bestPrecisionDelta {
							bestPrecisionRange = roundDec((valueN-minV)/delta, 6)
							bestPrecisionDelta = delta
						}
						whole := value + "(" + rangeStr + ")"
						if bestPrecisionRange > 1 || bestPrecisionRange < 0 {
							line = strings.Replace(line, whole, value, 1)
						} else {
							sign := ""
							if valueN < 0 {
								sign = "+"
							}
							line = strings.Replace(line, whole, sign+"("+rm[1]+"-"+rm[2]+")", 1)
						}
					}
					if hasIndependentRolls {
						line = regexp.MustCompile(`(-?\d+\.?\d*)\(-?\d+\.?\d*--?\d+\.?\d*\)`).ReplaceAllString(advancedCopyLine, "$1")
					} else if bestPrecisionRange <= 1 && bestPrecisionRange >= 0 {
						modLine.Range = &bestPrecisionRange
					}
				}
				rangedLine := applyRange(line, 1, catalystScalar, corruptedOr1(modLine))
				modList, extra, parsed := parseModLine3(rangedLine)
				nextModGroup := ""
				if l+1 < len(rawLines) {
					if m := modGroupRe.FindStringSubmatch(rawLines[l+1]); m != nil {
						nextModGroup = m[1]
					}
				}
				if (!parsed || extra != "") && l+1 < len(rawLines) && modLine.ModGroup == nextModGroup {
					nextLine := stripBalanced(rawLines[l+1], '{', '}')
					nextLine = gsubLimitFunc(nextLine, ` ?\(([a-z]+)\)`, -1, func(caps []string) string { return "" })
					combLine := line + " " + nextLine
					rangedLine = applyRange(combLine, 1, catalystScalar, corruptedOr1(modLine))
					modList, extra, parsed = parseModLine3(rangedLine)
					if parsed && extra == "" {
						line = line + "\n" + nextLine
						l++
					} else {
						modList, extra, parsed = parseModLine3(rangedLine)
					}
				}
				lineLower := luaLower(line)
				if modLine.flag("disabled") {
					lineLower = ""
				}
				switch {
				case lineLower == "implicit modifiers cannot be changed":
					it.ImplicitsCannotBeChanged = true
				case prefixAllowRe.MatchString(lineLower):
					it.Prefixes.Limit = adjustLimit(it.Prefixes.Limit, lineLower, prefixPlusRe, prefixMinusRe)
				case suffixAllowRe.MatchString(lineLower):
					it.Suffixes.Limit = adjustLimit(it.Suffixes.Limit, lineLower, suffixPlusRe, suffixMinusRe)
				case strings.Contains(lineLower, "can be anointed"):
					it.CanBeAnointed = true
				case lineLower == "can have a second enchantment modifier",
					lineLower == "can have 1 additional enchantment modifiers":
					it.CanHaveTwoEnchants = true
				case lineLower == "can have 2 additional enchantment modifiers":
					it.CanHaveTwoEnchants = true
					it.CanHaveThreeEnchants = true
				case lineLower == "can have 3 additional enchantment modifiers":
					it.CanHaveTwoEnchants = true
					it.CanHaveThreeEnchants = true
					it.CanHaveFourEnchants = true
				case lineLower == "has elder, shaper and all conqueror influences":
					it.HasElderShaperAndAllConquerorInfluences = true
					for _, key := range influenceKeysDefault {
						it.Influence[key] = true
					}
				case strings.Contains(lineLower, "if the eater of worlds is dominant"):
					it.CanHaveEldritchInfluence = true
				case lineLower == "has a crucible passive skill tree with only support passive skills":
					it.CanHaveOnlySupportSkillsCrucibleTree = true
				case lineLower == "has a crucible passive skill tree":
					it.CanHaveShieldCrucibleTree = true
				case lineLower == "has a two handed sword crucible passive skill tree":
					it.CanHaveTwoHandedSwordCrucibleTree = true
				case lineLower == "cannot roll caster modifiers":
					it.RestrictTag = true
					it.NoCaster = true
				case lineLower == "cannot roll attack modifiers":
					it.RestrictTag = true
					it.NoAttack = true
				case lineLower == "cannot roll modifiers of non-cold damage types":
					it.RestrictDamageType = true
					it.OnlyColdDamage = true
				case lineLower == "cannot roll modifiers of non-fire damage types":
					it.RestrictDamageType = true
					it.OnlyFireDamage = true
				case lineLower == "cannot roll modifiers of non-lightning damage types":
					it.RestrictDamageType = true
					it.OnlyLightningDamage = true
				case lineLower == "cannot roll modifiers of non-chaos damage types":
					it.RestrictDamageType = true
					it.OnlyChaosDamage = true
				case lineLower == "cannot roll modifiers of non-physical damage types":
					it.RestrictDamageType = true
					it.OnlyPhysicalDamage = true
				}
				if !modLine.flag("disabled") {
					it.detectModMagnitude(line, rangedLine, modLine, catalystScalar)
				}
				var modLines *[]*ModLine
				switch {
				case modLine.flag("enchant") || (modLine.flag("crafted") && len(it.EnchantModLines)+len(it.ImplicitModLines) < implicitLines):
					modLines = &it.EnchantModLines
				case modLine.flag("scourge"):
					modLines = &it.ScourgeModLines
				case strings.Contains(line, "Requires Class"):
					modLines = &it.ClassRequirementModLines
				case modLine.flag("implicit") || (!modLine.flag("crafted") && len(it.EnchantModLines)+len(it.ScourgeModLines)+len(it.ImplicitModLines) < implicitLines):
					modLines = &it.ImplicitModLines
				case modLine.flag("crucible"):
					modLines = &it.CrucibleModLines
				default:
					modLines = &it.ExplicitModLines
				}
				modLine.Line = line
				if parsed {
					modLine.ModList = modList
					modLine.HasModList = true
					modLine.Extra = extra
					modLine.ValueScalar = catalystScalar
					if modLine.Range == nil {
						r := DefaultItemAffixQuality
						modLine.Range = &r
					}
					*modLines = append(*modLines, modLine)
					if mode == "GAME" {
						if gameModeStage == "FINDIMPLICIT" {
							gameModeStage = "IMPLICIT"
						} else if gameModeStage == "FINDEXPLICIT" {
							foundExplicit = true
							gameModeStage = "EXPLICIT"
						} else if gameModeStage == "EXPLICIT" {
							foundExplicit = true
						}
					} else {
						foundExplicit = true
					}
				} else if mode == "GAME" {
					if gameModeStage == "IMPLICIT" || gameModeStage == "EXPLICIT" ||
						(gameModeStage == "FINDIMPLICIT" && data.ItemBases[line] == nil && it.Name != line &&
							!strings.Contains(line, "Two-Toned") &&
							!(it.Base != nil && (line == it.Base.Type || (it.Base.SubType != "" && line == it.Base.SubType+" "+it.Base.Type)))) {
						modLine.ModList = nil
						modLine.HasModList = true
						modLine.Extra = line
						r := DefaultItemAffixQuality
						modLine.Range = &r
						*modLines = append(*modLines, modLine)
					} else if gameModeStage == "FINDEXPLICIT" {
						gameModeStage = "DONE"
					}
				} else if foundExplicit || gameModeStage == "EXPLICIT" {
					modLine.ModList = nil
					modLine.HasModList = true
					modLine.Extra = line
					*modLines = append(*modLines, modLine)
				}
			}
		}
		l++
	}
	it.parseRawTail(implicitLines, importedLevelReq, highQuality, deferJewelRadiusIndexAssignment)
}

// detectModMagnitude ports the modMagnitudePattern loop.
func (it *Item) detectModMagnitude(line, rangedLine string, modLine *ModLine, catalystScalar float64) {
	for pi, pattern := range modMagnitudePatterns {
		if !pattern.MatchString(luaLower(rangedLine)) {
			continue
		}
		r := DefaultItemAffixQuality
		if modLine.Range != nil {
			r = *modLine.Range
		}
		reRanged := applyRange(line, r, catalystScalar, corruptedOr1(modLine))
		m := pattern.FindStringSubmatch(luaLower(reRanged))
		if m == nil {
			break
		}
		var amount, increaseOrDecrease, modTagsString string
		var multiplier *float64
		if pi == 2 {
			// "are doubled": one capture -> swap variables.
			modTagsString = m[1]
			amount = "100"
			increaseOrDecrease = "increased"
			two := 2.0
			multiplier = &two
		} else {
			amount, increaseOrDecrease, modTagsString = m[1], m[2], m[3]
		}
		if amount != "" && modTagsString != "" && (increaseOrDecrease == "increased" || increaseOrDecrease == "reduced") {
			amountN, _ := strconv.ParseFloat(amount, 64)
			quality := amountN
			if increaseOrDecrease == "reduced" {
				quality = -amountN
			}
			if modTagsString == "explicit physical and chaos damage" {
				it.modMagnitudeMods = append(it.modMagnitudeMods, &modMagnitudeMod{
					tags: []string{"damage"}, anyTags: []string{"physical", "chaos"},
					quality: quality, modType: "explicit",
				})
			} else {
				var modTags []string
				modType := ""
				for _, word := range strings.Fields(modTagsString) {
					word = luaLower(word)
					if mapped, ok := modMagnitudeTagMap[word]; ok {
						word = mapped
					}
					if word == "implicit" || word == "explicit" || word == "enchant" {
						modType = word
					} else {
						modTags = append(modTags, word)
					}
				}
				it.modMagnitudeMods = append(it.modMagnitudeMods, &modMagnitudeMod{
					tags: modTags, quality: quality, multiplier: multiplier, modType: modType,
				})
			}
			break
		}
	}
}

// stripEnumRanges is line:gsub("(%s*)(%b())", ...): drop parenthesised
// groups that contain "-" but no digit.
func stripEnumRanges(line string) string {
	var sb strings.Builder
	i := 0
	for i < len(line) {
		// %s* then %b()
		j := i
		for j < len(line) && isLuaSpace(line[j]) {
			j++
		}
		if j < len(line) && line[j] == '(' {
			depth := 1
			k := j + 1
			for k < len(line) && depth > 0 {
				if line[k] == '(' {
					depth++
				} else if line[k] == ')' {
					depth--
				}
				k++
			}
			if depth == 0 {
				group := line[j:k]
				if strings.Contains(group, "-") && !strings.ContainsAny(group, "0123456789") {
					// drop space + group
					i = k
					continue
				}
				sb.WriteString(line[i:k])
				i = k
				continue
			}
		}
		sb.WriteByte(line[i])
		i++
	}
	return sb.String()
}

func isLuaSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

func adjustLimit(limit *float64, lineLower string, plusRe, minusRe *regexp.Regexp) *float64 {
	cur := 0.0
	if limit != nil {
		cur = *limit
	}
	if m := plusRe.FindStringSubmatch(lineLower); m != nil {
		n, _ := strconv.ParseFloat(m[1], 64)
		cur += n
	}
	if m := minusRe.FindStringSubmatch(lineLower); m != nil {
		n, _ := strconv.ParseFloat(m[1], 64)
		cur -= n
	}
	return &cur
}

func setReq(req map[string]float64, key string, v *float64) {
	if v != nil {
		req[key] = *v
	} else {
		delete(req, key)
	}
}

func specToInt(s string) *int {
	if n := specToNumber(s); n != nil {
		v := int(*n)
		return &v
	}
	return nil
}

func reqOrZero(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}

func fmin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func fmax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func boolKeys(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func corruptedOr1(modLine *ModLine) float64 {
	if modLine.CorruptedRange != nil {
		return *modLine.CorruptedRange
	}
	return 1
}

// parseModLine3 wraps modparser.Parse into ([]*Mod, extra, parsed):
// parsed=false is Lua's nil modList.
func parseModLine3(line string) ([]*modparser.Mod, string, bool) {
	mods, extra := modparser.Parse(line)
	if mods == nil {
		return nil, extra, false
	}
	out := make([]*modparser.Mod, 0, len(mods))
	for _, mv := range mods {
		if m, ok := mv.(*modparser.Mod); ok {
			out = append(out, m)
		} else {
			panic("item: non-Mod entry in parse result for line: " + line)
		}
	}
	return out, extra, true
}

// parseModLine is the two-value form used for flask buff lines
// (modList or {}).
func parseModLine(line string) ([]*modparser.Mod, string) {
	mods, extra, parsed := parseModLine3(line)
	if !parsed {
		return nil, extra
	}
	return mods, extra
}

func affixPool(base *data.ItemBase) map[string]data.ItemModData {
	if base.SubType != "" {
		if pool, ok := data.ItemMods[base.Type+base.SubType]; ok {
			return pool
		}
	}
	if pool, ok := data.ItemMods[base.Type]; ok {
		return pool
	}
	return data.ItemMods["Item"]
}

var sortedBaseNamesCache struct {
	once  sync.Once
	names []string
}

func sortedBaseNames() []string {
	sortedBaseNamesCache.once.Do(func() {
		names := make([]string, 0, len(data.ItemBases))
		for name := range data.ItemBases {
			names = append(names, name)
		}
		sort.Strings(names)
		sortedBaseNamesCache.names = names
	})
	return sortedBaseNamesCache.names
}
