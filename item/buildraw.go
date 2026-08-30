// Port of ItemClass:BuildRaw / :BuildAndParseRaw / :Craft — the item's
// raw-text serializer and the crafted-template rebuild. Landed for the mod
// cache generator (Main.lua's loadItemDBs Crafts every crafted rare
// template before SaveModCache); the UI-only callers stay unported.

package item

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
)

var influenceDisplay = []string{"Shaper", "Elder", "Warlord", "Hunter", "Crusader", "Redeemer", "Searing Exarch", "Eater of Worlds"}

var reRangeShell = regexp.MustCompile(`\(-?[0-9.]+--?[0-9.]+\)`)
var reInt = regexp.MustCompile(`[0-9]+`)

func makeIDSpec(idList map[int]bool) string {
	ids := make([]int, 0, len(idList))
	for id := range idList {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = util.FormatIntOrG14(float64(id))
	}
	return strings.Join(parts, ",")
}

func prependToAllLines(line, prefix string) string {
	return prefix + strings.ReplaceAll(line, "\n", "\n"+prefix)
}

func (it *Item) writeModLine(rawLines *[]string, modLine *ModLine) {
	line := modLine.Line
	if modLine.Range != nil && reRangeShell.MatchString(line) {
		line = "{range:" + util.FormatIntOrG14(util.RoundHalfUp(*modLine.Range, 6)) + "}" + line
	}
	if modLine.CorruptedRange != nil {
		line = "{corruptedRange:" + util.FormatIntOrG14(util.RoundHalfUp(*modLine.CorruptedRange, 2)) + "}" + line
	}
	for _, f := range []string{"disabled", "crafted", "enchant", "custom", "scourge", "crucible", "mutated"} {
		if modLine.flag(f) {
			line = "{" + f + "}" + line
		}
	}
	if modLine.ModGroup != "" {
		line = prependToAllLines(line, "{modGroup:"+modLine.ModGroup+"}")
	}
	for _, f := range []string{"fractured", "prefix", "suffix", "exarch", "eater", "synthesis", "unscalable", "vestigial"} {
		if modLine.flag(f) {
			line = "{" + f + "}" + line
		}
	}
	hasNewSelection := modLine.VersionList != nil || modLine.VariantGroupList != nil
	if hasNewSelection && len(modLine.ModTags) > 0 {
		line = "{tags:" + strings.Join(modLine.ModTags, ",") + "}" + line
	}
	if modLine.VariantGroupList != nil {
		line = prependToAllLines(line, "{group:"+makeIDSpec(modLine.VariantGroupList)+"}")
	}
	if modLine.VariantList != nil {
		line = prependToAllLines(line, "{variant:"+makeIDSpec(modLine.VariantList)+"}")
	}
	if modLine.VersionList != nil {
		line = prependToAllLines(line, "{version:"+makeIDSpec(modLine.VersionList)+"}")
	}
	if !hasNewSelection && len(modLine.ModTags) > 0 {
		line = "{tags:" + strings.Join(modLine.ModTags, ",") + "}" + line
	}
	*rawLines = append(*rawLines, line)
}

// affixSpecLine renders one "Prefix: ..."/"Suffix: ..." template line.
func affixSpecLine(kind string, affix *Affix) string {
	rangeSpec := ""
	if affix.Range.Single.Set {
		rangeSpec = "{range:" + util.FormatIntOrG14(util.RoundHalfUp(affix.Range.Single.V, 3)) + "}"
	} else if affix.Range.Multi != nil {
		parts := make([]string, len(affix.Range.Multi))
		for i, v := range affix.Range.Multi {
			parts[i] = util.FormatIntOrG14(v)
		}
		rangeSpec = "{range:" + strings.Join(parts, ",") + "}"
	}
	fractured := ""
	if affix.Fractured {
		fractured = "{fractured}"
	}
	return kind + ": " + fractured + rangeSpec + affix.ModID
}

// BuildRaw ports ItemClass:BuildRaw.
func (it *Item) BuildRaw() string {
	var rawLines []string
	add := func(s string) { rawLines = append(rawLines, s) }
	add("Rarity: " + it.Rarity)
	if it.Title != "" {
		add(it.Title)
		add(it.BaseName)
	} else {
		add(it.NamePrefix + it.BaseName + it.NameSuffix)
	}
	if it.ArmourData != nil {
		for _, typ := range []string{"Armour", "Evasion", "EnergyShield", "Ward"} {
			stat := it.ArmourData.Defence(typ)
			if stat.Value.Set && stat.Value.V > 0 {
				add(strings.ReplaceAll(typ, "EnergyShield", "Energy Shield") + ": " + util.FormatIntOrG14(stat.Value.V))
				if stat.BasePercentile.Set {
					add(typ + "BasePercentile: " + util.FormatIntOrG14(stat.BasePercentile.V))
				}
			}
		}
	}
	if it.Intangibility != nil {
		add(fmt.Sprintf("Intangibility: %d%%", int(*it.Intangibility)))
	}
	if it.UniqueID != "" {
		add("Unique ID: " + it.UniqueID)
	}
	if it.League != "" {
		add("League: " + it.League)
	}
	if it.Unreleased {
		add("Unreleased: true")
	}
	for i, key := range influenceKeys {
		if it.Influence[key] {
			add(influenceDisplay[i] + " Item")
		}
	}
	if it.Crafted {
		add("Crafted: true")
		for _, affix := range it.Prefixes.List {
			add(affixSpecLine("Prefix", affix))
		}
		for _, affix := range it.Suffixes.List {
			add(affixSpecLine("Suffix", affix))
		}
	}
	if it.Catalyst != nil && *it.Catalyst > 0 {
		add("Catalyst: " + catalystList[*it.Catalyst-1])
	}
	if it.CatalystQuality != nil {
		add("CatalystQuality: " + util.FormatIntOrG14(*it.CatalystQuality))
	}
	if it.ClusterJewel != nil {
		if it.ClusterJewelSkill != "" {
			add("Cluster Jewel Skill: " + it.ClusterJewelSkill)
		}
		if it.ClusterJewelNodeCount != nil {
			add("Cluster Jewel Node Count: " + util.FormatIntOrG14(*it.ClusterJewelNodeCount))
		}
	}
	if it.TalismanTier != nil {
		add("Talisman Tier: " + util.FormatIntOrG14(*it.TalismanTier))
	}
	if it.ItemLevel != nil {
		add("Item Level: " + util.FormatIntOrG14(*it.ItemLevel))
	}
	if it.MemoryStrands != nil {
		add("Memory Strands: " + util.FormatIntOrG14(*it.MemoryStrands))
	}
	if it.VersionList != nil {
		for _, versionName := range it.VersionList {
			add("Version: " + versionName)
		}
		if it.SelectedVersion != nil {
			add("Selected Version: " + util.FormatIntOrG14(float64(*it.SelectedVersion)))
		}
	}
	if it.VariantList != nil {
		for _, variantName := range it.VariantList {
			add("Variant: " + variantName)
		}
		if it.UsesVariantGroups {
			groupIDs := make([]int, 0, len(it.VariantGroups))
			for id := range it.VariantGroups {
				groupIDs = append(groupIDs, id)
			}
			sort.Ints(groupIDs)
			for _, groupID := range groupIDs {
				if variantID, ok := it.VariantGroupSelections[groupID]; ok {
					add("Selected Variant Group: " + util.FormatIntOrG14(float64(groupID)) + "=" + util.FormatIntOrG14(float64(variantID)))
				}
			}
		} else {
			add("Selected Variant: " + util.FormatIntOrG14(float64(*it.Variant)))
		}
		for _, bl := range it.BaseLines {
			if bl.variantList != nil || bl.versionList != nil || bl.variantGroupList != nil {
				it.writeModLine(&rawLines, &ModLine{Line: bl.line, VariantList: bl.variantList, VersionList: bl.versionList, VariantGroupList: bl.variantGroupList})
			}
		}
		if !it.UsesVariantGroups {
			for _, alt := range []struct {
				has bool
				sel *int
				lbl string
			}{
				{it.HasAltVariant, it.VariantAlt, ""},
				{it.HasAltVariant2, it.VariantAlt2, " Two"},
				{it.HasAltVariant3, it.VariantAlt3, " Three"},
				{it.HasAltVariant4, it.VariantAlt4, " Four"},
				{it.HasAltVariant5, it.VariantAlt5, " Five"},
			} {
				if alt.has {
					add("Has Alt Variant" + alt.lbl + ": true")
					add("Selected Alt Variant" + alt.lbl + ": " + util.FormatIntOrG14(float64(*alt.sel)))
				}
			}
		}
		if it.AllowDuplicateVariants {
			add("Allow Duplicate Variants: true")
		}
	}
	if it.VariantList == nil {
		for _, bl := range it.BaseLines {
			if bl.versionList != nil || bl.variantGroupList != nil {
				it.writeModLine(&rawLines, &ModLine{Line: bl.line, VariantList: bl.variantList, VersionList: bl.versionList, VariantGroupList: bl.variantGroupList})
			}
		}
	}
	if it.Quality != nil {
		add("Quality: " + util.FormatIntOrG14(*it.Quality))
	}
	if len(it.Sockets) > 0 {
		line := "Sockets: "
		for i, socket := range it.Sockets {
			line += socket.Color
			if i+1 < len(it.Sockets) {
				if socket.Group == it.Sockets[i+1].Group {
					line += "-"
				} else {
					line += " "
				}
			}
		}
		add(line)
	}
	if it.Requirements.Level.Set {
		add("LevelReq: " + util.FormatIntOrG14(it.Requirements.Level.V))
	}
	if it.JewelRadiusLabel != "" {
		add("Radius: " + it.JewelRadiusLabel)
	}
	if it.Limit != nil {
		add("Limited to: " + util.FormatIntOrG14(*it.Limit))
	}
	if it.ClassRestriction != "" {
		add("Requires Class " + it.ClassRestriction)
	}
	add("Implicits: " + util.FormatIntOrG14(float64(len(it.EnchantModLines)+len(it.ImplicitModLines)+len(it.ScourgeModLines))))
	for _, group := range [][]*ModLine{it.EnchantModLines, it.ScourgeModLines, it.ClassRequirementModLines, it.ImplicitModLines, it.ExplicitModLines, it.CrucibleModLines} {
		for _, modLine := range group {
			it.writeModLine(&rawLines, modLine)
		}
	}
	if it.Split {
		add("Split")
	}
	if it.Mirrored {
		add("Mirrored")
	}
	if it.Fractured {
		add("Fractured Item")
	}
	if it.Corrupted || it.Scourge {
		add("Corrupted")
	}
	if it.FoilType != "" {
		add("Foil Unique (" + it.FoilType + ")")
	}
	return strings.Join(rawLines, "\n")
}

// BuildAndParseRaw ports ItemClass:BuildAndParseRaw.
func (it *Item) BuildAndParseRaw() {
	it.ParseRaw(it.BuildRaw(), "", false)
}

// combineStats is Craft's stat-combining gsub: each number in the existing
// line gains the corresponding number of the incoming line.
func combineStats(existing, incoming string) string {
	nums := reInt.FindAllString(incoming, -1)
	i := 0
	return reInt.ReplaceAllStringFunc(existing, func(num string) string {
		var a, b float64
		fmt.Sscanf(num, "%g", &a)
		fmt.Sscanf(nums[i], "%g", &b)
		i++
		return util.FormatIntOrG14(a + b)
	})
}

// Craft ports ItemClass:Craft — rebuild explicit modifiers from the item's
// affixes (at their ranges, 0.5 by default) and re-parse.
func (it *Item) Craft() {
	var savedMods []*ModLine
	for _, mod := range it.ExplicitModLines {
		if mod.flag("crafted") || mod.flag("custom") || (it.RareLikeUnique != nil && !(mod.flag("prefix") || mod.flag("suffix"))) {
			savedMods = append(savedMods, mod)
		}
	}

	it.ExplicitModLines = nil
	it.NamePrefix = ""
	it.NameSuffix = ""
	it.Requirements.Level = optFloat(it.Base.Req.Level)
	statOrder := map[float64]*ModLine{}
	for _, list := range []*AffixList{&it.Prefixes, &it.Suffixes} {
		limit := it.AffixLimit / 2
		if list.Limit != nil {
			limit = *list.Limit
		}
		for i := 1; i <= int(limit); i++ {
			for len(list.List) < i {
				list.List = append(list.List, &Affix{ModID: "None"})
			}
			affix := list.List[i-1]
			mod, ok := it.Affixes[affix.ModID]
			if !ok {
				continue
			}
			if mod.Type == "Prefix" {
				it.NamePrefix = mod.Affix + " " + it.NamePrefix
			} else if mod.Type == "Suffix" {
				it.NameSuffix = it.NameSuffix + " " + mod.Affix
			}
			lvl := math.Floor(mod.Level * 0.8)
			if !it.Requirements.Level.Set || it.Requirements.Level.V < lvl {
				it.Requirements.Level = util.Some(lvl)
			}
			for j, line := range mod.Lines {
				rng := affix.Range.Single.Or(0.5)
				line = applyRange(line, rng, 1, 1)
				order := mod.StatOrder[j]
				if existing := statOrder[order]; existing != nil {
					existing.Line = combineStats(existing.Line, line)
				} else {
					modLine := &ModLine{Line: line, Order: &order, ModTags: mod.ModTags}
					if modLine.ModTags == nil {
						modLine.ModTags = []string{}
					}
					if affix.Fractured {
						modLine.setFlag("fractured")
					}
					if mod.Type == "Prefix" {
						modLine.setFlag("prefix")
					} else if mod.Type == "Suffix" {
						modLine.setFlag("suffix")
					}
					it.ExplicitModLines = append(it.ExplicitModLines, modLine)
					statOrder[order] = modLine
				}
			}
		}
	}

	it.ExplicitModLines = append(it.ExplicitModLines, savedMods...)
	if len(it.ExplicitModLines) > 1 {
		sortCraftedModLines(it.ExplicitModLines)
	}

	it.BuildAndParseRaw()
}
