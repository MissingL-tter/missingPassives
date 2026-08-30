// Port of Classes/Item.lua (the equippable item model) and the pieces of
// Modules/ItemTools.lua it leans on. The scope is the parse half: raw item
// text -> parsed state -> mod lists, i.e. everything the calc consumes
// (dump_calc.lua's itemFixture projection). The crafting half (Craft,
// BuildRaw, MutateMod) and the UI helpers stay unported.
//
// Reference-dead or corpus-unreachable branches panic loudly instead of
// diverging silently, per the porting method.
package item

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// DefaultItemAffixQuality is main.defaultItemAffixQuality (Main.lua L102).
var DefaultItemAffixQuality = 0.5

var dmgTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

var catalystList = []string{"Abrasive", "Accelerating", "Dextral", "Fertile", "Imbued", "Intrinsic", "Noxious", "Prismatic", "Sinistral", "Tempering", "Turbulent", "Unstable"}

var catalystDescriptorList = []string{"Attack", "Speed", "Suffix", "Life and Mana", "Caster", "Attribute", "Physical and Chaos", "Resistance", "Prefix", "Defence", "Elemental", "Critical"}

var catalystTags = [][]string{
	{"attack"},
	{"speed"},
	{"suffix"},
	{"life", "mana", "resource"},
	{"caster"},
	{"jewellery_attribute", "attribute"},
	{"physical_damage", "chaos_damage"},
	{"jewellery_resistance", "resistance"},
	{"prefix"},
	{"jewellery_defense", "defences", "armour", "evasion", "energyshield"},
	{"jewellery_elemental", "elemental_damage"},
	{"critical"},
}

// influenceKeys is itemLib.influenceInfo.all's key column, in order.
var influenceKeys = []string{"shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie", "cleansing", "tangle"}

// influenceKeysDefault is itemLib.influenceInfo.default (no eldritch pair).
var influenceKeysDefault = []string{"shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie"}

// influenceItemMap maps the item-text line to the influence key.
var influenceItemMap = map[string]string{
	"Shaper Item":          "shaper",
	"Elder Item":           "elder",
	"Warlord Item":         "adjudicator",
	"Hunter Item":          "basilisk",
	"Crusader Item":        "crusader",
	"Redeemer Item":        "eyrie",
	"Searing Exarch Item":  "cleansing",
	"Eater of Worlds Item": "tangle",
}

var lineFlags = map[string]bool{
	"crafted": true, "crucible": true, "custom": true, "disabled": true,
	"eater": true, "enchant": true, "exarch": true, "fractured": true,
	"implicit": true, "scourge": true, "synthesis": true, "mutated": true,
	"unscalable": true, "prefix": true, "suffix": true, "unveiled": true,
	"vestigial": true,
}

// rarityColorCodes is the subset of colorCodes keys a "Rarity:" line can
// name. BuildRaw only ever writes the five item rarities, so the loader
// path never sees the wider colorCodes table.
var rarityColorCodes = map[string]bool{
	"NORMAL": true, "MAGIC": true, "RARE": true, "UNIQUE": true, "RELIC": true,
}

// ModLine is one parsed modifier line (the reference's modLine table).
// Flags carries the lineFlags-keyed booleans ({crafted}, " (enchant)", ...).
type ModLine struct {
	Line             string
	ModList          []*modparser.Mod
	Extra            string // "" = fully understood
	HasModList       bool   // Lua modList ~= nil (an empty list is distinct from nil)
	ModTags          []string
	Flags            map[string]bool
	VariantList      map[int]bool
	VersionList      map[int]bool
	VariantGroupList map[int]bool
	ModGroup         string
	Range            *float64
	CorruptedRange   *float64
	ValueScalar      util.Opt[float64]
	Order            *float64
	ModID            string
	NewModID         string
	ShowSlider       bool
}

func (m *ModLine) flag(name string) bool { return m.Flags != nil && m.Flags[name] }

// Flag is the exported read for projections outside the package.
func (m *ModLine) Flag(name string) bool { return m.flag(name) }

func (m *ModLine) setFlag(name string) {
	if m.Flags == nil {
		m.Flags = map[string]bool{}
	}
	m.Flags[name] = true
}

// Socket is one entry of item.sockets.
type Socket struct {
	Color string
	Group float64
}

// Affix is one prefixes/suffixes entry.
type Affix struct {
	ModID     string
	Range     AffixRange
	Fractured bool
}

// AffixList is the reference's prefixes/suffixes table: an array plus a
// `limit` hash field.
type AffixList struct {
	List  []*Affix
	Limit *float64
}

type modMagnitudeMod struct {
	tags       []string
	anyTags    []string
	quality    float64
	multiplier *float64
	modType    string
}

type baseLine struct {
	line             string
	variantList      map[int]bool
	versionList      map[int]bool
	variantGroupList map[int]bool
}

// Item mirrors the reference Item class's parse-relevant state.
type Item struct {
	ID  int // items tab id; 0 = unset (modSource then uses -1... see BuildModList)
	Raw string

	Name       string
	NamePrefix string
	NameSuffix string
	Title      string
	BaseName   string
	Base       *data.ItemBase
	Type       string
	Rarity     string
	FoilType   string

	// Influences, keyed by influenceKeys entries.
	Influence map[string]bool

	UniqueID         string
	ItemLevel        *float64
	MemoryStrands    *float64
	ClassRestriction string
	Quality          *float64
	CatalystQuality  *float64
	Catalyst         *int // 1-based index into catalystList
	Intangibility    *float64
	League           string
	Note             string
	Source           string
	UpgradePaths     []string
	Unreleased       bool

	Corrupted   bool
	Split       bool
	Mirrored    bool
	Fractured   bool
	Synthesised bool
	Veiled      bool
	Crafted     bool
	Scourge     bool
	Crucible    bool
	Implicit    bool
	Foulborn    bool
	Vestigial   bool
	HiddenSpecs bool

	AdvancedCopy bool

	ImplicitsCannotBeChanged bool
	CanBeAnointed            bool
	CanHaveTwoEnchants       bool
	CanHaveThreeEnchants     bool
	CanHaveFourEnchants      bool
	CanHaveEldritchInfluence bool

	HasElderShaperAndAllConquerorInfluences bool
	CanHaveOnlySupportSkillsCrucibleTree    bool
	CanHaveShieldCrucibleTree               bool
	CanHaveTwoHandedSwordCrucibleTree       bool

	RestrictTag         bool
	NoCaster, NoAttack  bool
	RestrictDamageType  bool
	OnlyColdDamage      bool
	OnlyFireDamage      bool
	OnlyLightningDamage bool
	OnlyChaosDamage     bool
	OnlyPhysicalDamage  bool

	TalismanTier *float64
	Limit        *float64

	Sockets                   []*Socket
	SocketColourAlwaysMatches bool
	DefaultSocketColor        string
	AbyssalSocketCount        float64
	SelectableSocketCount     float64

	Requirements Requirements

	ClassRequirementModLines []*ModLine
	BuffModLines             []*ModLine
	EnchantModLines          []*ModLine
	ScourgeModLines          []*ModLine
	ImplicitModLines         []*ModLine
	ExplicitModLines         []*ModLine
	CrucibleModLines         []*ModLine

	VariantList            []string
	VersionList            []string
	Variant                *int
	VariantAlt             *int
	VariantAlt2            *int
	VariantAlt3            *int
	VariantAlt4            *int
	VariantAlt5            *int
	HasAltVariant          bool
	HasAltVariant2         bool
	HasAltVariant3         bool
	HasAltVariant4         bool
	HasAltVariant5         bool
	AllowDuplicateVariants bool
	UsesVariantGroups      bool
	SelectedVersion        *int
	VariantGroups          map[int]map[int]map[int]bool // group -> variant -> version set (0 = versionless)
	VariantGroupSelections map[int]int

	Prefixes AffixList
	Suffixes AffixList

	Affixes        map[string]data.ItemModData
	RareLikeUnique *data.RareLikeUnique

	BaseLines map[string]*baseLine

	Corruptible           bool
	CanBeInfluenced       bool
	ClusterJewel          *data.ClusterJewelSize
	ClusterJewelSkill     string
	ClusterJewelNodeCount *float64

	JewelRadiusLabel string
	JewelRadiusIndex *int

	AffixLimit     float64
	CraftedQuality *float64

	modMagnitudeMods []*modMagnitudeMod
	mutatedLines     map[string]string

	// Built by BuildModList / BuildModListForSlotNum.
	ModSource   string
	BaseModList []*modparser.Mod
	ModList     []*modparser.Mod
	SlotModList map[int][]*modparser.Mod
	BuffModList []*modparser.Mod
	// BuffModListInit: the reference sets buffModList = {} (a table) for
	// flask/tincture bases; nil-vs-empty is fixture-visible.
	BuffModListInit bool
	RangeLineList   []*ModLine
	HasModTags      bool
	GrantedSkills   []GrantedSkill

	WeaponData   map[int]*WeaponData
	ArmourData   *ArmourData
	FlaskData    *FlaskData
	TinctureData *TinctureData
	JewelData    *JewelData

	SocketedJewelEffectModifier float64
}

// variantPtr returns the alt-variant slot i (0 = variant, 1..5 = alts).
func (it *Item) variantSlot(i int) (*int, bool) {
	switch i {
	case 0:
		return it.Variant, true
	case 1:
		return it.VariantAlt, it.HasAltVariant
	case 2:
		return it.VariantAlt2, it.HasAltVariant2
	case 3:
		return it.VariantAlt3, it.HasAltVariant3
	case 4:
		return it.VariantAlt4, it.HasAltVariant4
	case 5:
		return it.VariantAlt5, it.HasAltVariant5
	}
	return nil, false
}

// CheckModLineVariant ports ItemClass:CheckModLineVariant.
func (it *Item) CheckModLineVariant(modLine *ModLine) bool {
	if it.UsesVariantGroups {
		if modLine.VersionList != nil && (it.SelectedVersion == nil || !modLine.VersionList[*it.SelectedVersion]) {
			return false
		}
		if modLine.VariantGroupList != nil {
			if modLine.VariantList == nil {
				return false
			}
			for groupID := range modLine.VariantGroupList {
				if selected, ok := it.VariantGroupSelections[groupID]; ok && modLine.VariantList[selected] {
					return true
				}
			}
			return false
		}
		return modLine.VariantList == nil
	}
	if modLine.VariantList == nil {
		return true
	}
	if it.Variant != nil && modLine.VariantList[*it.Variant] {
		return true
	}
	for i := 1; i <= 5; i++ {
		if v, has := it.variantSlot(i); has && v != nil && modLine.VariantList[*v] {
			return true
		}
	}
	return false
}

// GetModLineVariantCount ports ItemClass:GetModLineVariantCount.
func (it *Item) GetModLineVariantCount(modLine *ModLine) int {
	if !it.AllowDuplicateVariants || modLine.VariantList == nil {
		if it.CheckModLineVariant(modLine) {
			return 1
		}
		return 0
	}
	count := 0
	if it.Variant != nil && modLine.VariantList[*it.Variant] {
		count = 1
	}
	for i := 1; i <= 5; i++ {
		if v, has := it.variantSlot(i); has && v != nil && modLine.VariantList[*v] {
			count++
		}
	}
	return count
}

// GetPrimarySlot ports ItemClass:GetPrimarySlot.
func (it *Item) GetPrimarySlot() string {
	if it.Base.Weapon != nil {
		return "Weapon 1"
	}
	switch it.Type {
	case "Quiver", "Shield":
		return "Weapon 2"
	case "Ring":
		return "Ring 1"
	case "Flask":
		return "Flask 1"
	case "Tincture":
		return "Flask 1"
	case "Graft":
		return "Graft 1"
	}
	return it.Type
}

// NormaliseQuality ports ItemClass:NormaliseQuality.
func (it *Item) NormaliseQuality() {
	if it.Base != nil && (it.Base.Armour != nil || it.Base.Weapon != nil || it.Base.Flask != nil || it.Base.Tincture != nil) {
		if it.Quality == nil {
			q := 0.0
			it.Quality = &q
		} else if it.UniqueID == "" && !it.Corrupted && !it.Split && !it.Mirrored && *it.Quality < 20 {
			q := 20.0
			it.Quality = &q
		}
	}
}

// getCatalystScalar ports the file-local getCatalystScalar.
func getCatalystScalar(catalystID *int, modLine *ModLine, quality *float64) float64 {
	if modLine.flag("unscalable") {
		return 1
	}
	tags := modLine.ModTags
	if catalystID == nil || *catalystID < 1 || *catalystID > len(catalystTags) || len(tags) == 0 {
		return 1
	}
	q := 20.0
	if quality != nil {
		q = *quality
	}
	tagLookup := map[string]bool{}
	for _, t := range tags {
		tagLookup[t] = true
	}
	for _, lf := range []string{"prefix", "suffix"} {
		if modLine.flag(lf) {
			tagLookup[lf] = true
		}
	}
	for _, catalystTag := range catalystTags[*catalystID-1] {
		if tagLookup[catalystTag] {
			return (100 + q) / 100
		}
	}
	return 1
}

// normaliseModLine ports the file-local normaliseModLine.
func normaliseModLine(line string) string {
	s := gsubNumberHash(line)
	// %(%-?#%-#%) matches both the signed and unsigned range shells.
	s = strings.ReplaceAll(s, "(-#-#)", "#")
	s = strings.ReplaceAll(s, "(#-#)", "#")
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "\n", " ")
}

// IsVariantGroupOptionEligible ports the same-named method.
func (it *Item) IsVariantGroupOptionEligible(groupID, variantID int) bool {
	group := it.VariantGroups[groupID]
	if group == nil {
		return false
	}
	versions := group[variantID]
	if versions == nil {
		return false
	}
	return versions[0] || (it.SelectedVersion != nil && versions[*it.SelectedVersion])
}

// GetVariantGroupOptions ports the same-named method (excludeSelected is
// always false on the parse path).
func (it *Item) GetVariantGroupOptions(groupID int, excludeSelected bool) []int {
	var options []int
	if it.VariantGroups == nil || it.VariantGroups[groupID] == nil {
		return options
	}
	used := map[int]bool{}
	if excludeSelected {
		for _, otherGroupID := range sortedKeys(it.VariantGroups) {
			if otherGroupID != groupID {
				if variantID, ok := it.VariantGroupSelections[otherGroupID]; ok && it.IsVariantGroupOptionEligible(otherGroupID, variantID) {
					used[variantID] = true
				}
			}
		}
	}
	for variantID := 1; variantID <= len(it.VariantList); variantID++ {
		if it.IsVariantGroupOptionEligible(groupID, variantID) && !used[variantID] {
			options = append(options, variantID)
		}
	}
	return options
}

// NormaliseVariantSelections ports the same-named method.
func (it *Item) NormaliseVariantSelections() {
	if len(it.VersionList) > 0 {
		sel := len(it.VersionList)
		if it.SelectedVersion != nil {
			sel = *it.SelectedVersion
		}
		sel = imax(1, imin(len(it.VersionList), sel))
		it.SelectedVersion = &sel
	} else {
		it.SelectedVersion = nil
	}
	if it.VariantGroupSelections == nil {
		it.VariantGroupSelections = map[int]int{}
	}
	for groupID := range it.VariantGroupSelections {
		if it.VariantGroups[groupID] == nil {
			delete(it.VariantGroupSelections, groupID)
		}
	}
	used := map[int]bool{}
	var needsSelection []int
	for _, groupID := range sortedKeys(it.VariantGroups) {
		if len(it.GetVariantGroupOptions(groupID, false)) > 0 {
			selected, ok := it.VariantGroupSelections[groupID]
			if ok && it.IsVariantGroupOptionEligible(groupID, selected) && !used[selected] {
				used[selected] = true
			} else {
				needsSelection = append(needsSelection, groupID)
			}
		}
	}
	for _, groupID := range needsSelection {
		selected := 0
		for _, variantID := range it.GetVariantGroupOptions(groupID, false) {
			if !used[variantID] {
				selected = variantID
				break
			}
		}
		if selected != 0 {
			it.VariantGroupSelections[groupID] = selected
			used[selected] = true
		} else {
			delete(it.VariantGroupSelections, groupID)
		}
	}
}
