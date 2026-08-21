// data.itemBases, the derived base lists, and the rare templates, from the
// bases document.

package data

import (
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

// ItemBase is one data.itemBases entry.
type ItemBase struct {
	Type             string            `lua:"type"`
	SubType          string            `lua:"subType,omitempty"`
	Hidden           bool              `lua:"hidden,omitempty"`
	SocketLimit      *float64          `lua:"socketLimit"`
	Tags             map[string]bool   `lua:"tags"`
	InfluenceTags    map[string]string `lua:"influenceTags"`
	Implicit         *string           `lua:"implicit"`
	ImplicitModTypes [][]string        `lua:"implicitModTypes"`
	ImplicitIds      []string          `lua:"implicitIds"`
	Enchant          *string           `lua:"enchant"`
	EnchantIds       []string          `lua:"enchantIds"`
	EnchantModTypes  [][]string        `lua:"enchantModTypes"`
	CannotBeAnointed bool              `lua:"cannotBeAnointed,omitempty"`
	Weapon           *WeaponData       `lua:"weapon"`
	Armour           *ArmourData       `lua:"armour"`
	Flask            *FlaskData        `lua:"flask"`
	Tincture         *TinctureData     `lua:"tincture"`
	Req              Req               `lua:"req"`
	FlavourText      []string          `lua:"flavourText"`
}

type WeaponData struct {
	PhysicalMin    float64 `lua:"PhysicalMin"`
	PhysicalMax    float64 `lua:"PhysicalMax"`
	CritChanceBase float64 `lua:"CritChanceBase"`
	AttackRateBase float64 `lua:"AttackRateBase"`
	Range          float64 `lua:"Range"`
}

type ArmourData struct {
	BlockChance         *float64 `lua:"BlockChance"`
	ArmourBaseMin       *float64 `lua:"ArmourBaseMin"`
	ArmourBaseMax       *float64 `lua:"ArmourBaseMax"`
	EvasionBaseMin      *float64 `lua:"EvasionBaseMin"`
	EvasionBaseMax      *float64 `lua:"EvasionBaseMax"`
	EnergyShieldBaseMin *float64 `lua:"EnergyShieldBaseMin"`
	EnergyShieldBaseMax *float64 `lua:"EnergyShieldBaseMax"`
	MovementPenalty     *float64 `lua:"MovementPenalty"`
	WardBaseMin         *float64 `lua:"WardBaseMin"`
	WardBaseMax         *float64 `lua:"WardBaseMax"`
}

type FlaskData struct {
	Life        *float64 `lua:"life"`
	Mana        *float64 `lua:"mana"`
	Duration    float64  `lua:"duration"`
	ChargesUsed float64  `lua:"chargesUsed"`
	ChargesMax  float64  `lua:"chargesMax"`
	Buff        []string `lua:"buff"`
}

type TinctureData struct {
	ManaBurn float64 `lua:"manaBurn"`
	Cooldown float64 `lua:"cooldown"`
}

type Req struct {
	Level *float64 `lua:"level"`
	Str   *float64 `lua:"str"`
	Dex   *float64 `lua:"dex"`
	Int   *float64 `lua:"int"`
}

// RareLikeUnique describes a unique using the rare item crafting controls.
type RareLikeUnique struct {
	ValidBases              []any                  `lua:"validBases"`
	Affixes                 map[string]ItemModData `lua:"affixes"`
	PrefixLimit             float64                `lua:"prefixLimit"`
	SuffixLimit             float64                `lua:"suffixLimit"`
	IgnoreModType           bool                   `lua:"ignoreModType,omitempty"`
	AllowDuplicateGroups    bool                   `lua:"allowDuplicateGroups,omitempty"`
	SupportsCustomModifiers map[string]bool        `lua:"supportsCustomModifiers"`
}

// baseOnlyEntry is a validBases entry carrying just a base.
type baseOnlyEntry struct {
	Base *ItemBase `lua:"base"`
}

func (d *Data) loadRareLikeUniques() {
	if d.ItemBases["Ghostflame Blade"] == nil {
		return // partial Sources (tests)
	}
	crimsonStormMods := map[string]ItemModData{}
	for modId, mod := range d.VeiledMods {
		if mod.Affix == "of the Order" {
			crimsonStormMods[modId] = mod
		}
	}

	ghost := *d.ItemBases["Ghostflame Blade"]
	tags := map[string]bool{}
	for k, v := range ghost.Tags {
		tags[k] = v
	}
	tags["deepwater_sword"] = true
	ghost.Tags = tags

	abyss := make([]any, 0, len(d.ItemBaseLists["Jewel: Abyss"]))
	for i := range d.ItemBaseLists["Jewel: Abyss"] {
		abyss = append(abyss, d.ItemBaseLists["Jewel: Abyss"][i])
	}

	d.RareLikeUniques = map[string]RareLikeUnique{
		"subsume the source": {
			ValidBases:           abyss,
			Affixes:              d.ItemMods["JewelAbyss"],
			PrefixLimit:          4,
			SuffixLimit:          0,
			IgnoreModType:        true,
			AllowDuplicateGroups: true,
		},
		"the crimson storm": {
			Affixes:     crimsonStormMods,
			PrefixLimit: 0,
			SuffixLimit: 1,
		},
		"dread captain's cutlass": {
			ValidBases:  []any{baseOnlyEntry{Base: &ghost}},
			Affixes:     d.ItemMods["Explicit"],
			PrefixLimit: 3,
			SuffixLimit: 3,
			SupportsCustomModifiers: map[string]bool{
				"ESSENCE": true,
				"VEILED":  true,
				"CUSTOM":  true,
			},
		},
	}
}

// baseItemTypes is Data.lua's load order for Data/Bases/<type>.
var baseItemTypes = []string{
	"axe", "bow", "claw", "dagger", "fishing", "mace", "staff", "sword", "wand",
	"helmet", "body", "gloves", "boots", "shield", "quiver",
	"amulet", "ring", "belt", "jewel", "flask", "tincture", "graft",
}

func intPtrToFloat(v *int64) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v)
	return &f
}

func loadItemBase(b gamedata.ItemBase) *ItemBase {
	e := &ItemBase{
		Type:        b.Type,
		SubType:     b.SubType,
		Hidden:      b.Hidden,
		SocketLimit: b.SocketLimit,
		Tags:        map[string]bool{},
	}
	for _, t := range b.Tags {
		e.Tags[t] = true
	}
	if b.InfluenceBaseTag != "" {
		e.InfluenceTags = map[string]string{}
		for _, suffix := range []string{"shaper", "elder", "adjudicator", "basilisk", "crusader", "eyrie", "cleansing", "tangle"} {
			e.InfluenceTags[suffix] = b.InfluenceBaseTag + "_" + suffix
		}
	}
	e.ImplicitModTypes = [][]string{}
	for _, t := range b.ImplicitModTypes {
		e.ImplicitModTypes = append(e.ImplicitModTypes, splitModTags(t))
	}
	if len(b.Implicit) > 0 {
		s := luaUnescape(strings.Join(b.Implicit, "\\n"))
		e.Implicit = &s
		e.ImplicitIds = b.ImplicitIds
	}
	if len(b.Enchant) > 0 {
		s := luaUnescape(strings.Join(b.Enchant, "\\n"))
		e.Enchant = &s
		e.EnchantIds = b.EnchantIds
		e.EnchantModTypes = [][]string{}
		for _, t := range b.EnchantModTypes {
			e.EnchantModTypes = append(e.EnchantModTypes, splitModTags(t))
		}
	}
	e.CannotBeAnointed = b.CannotBeAnointed
	if w := b.Weapon; w != nil {
		e.Weapon = &WeaponData{
			PhysicalMin:    float64(w.PhysicalMin),
			PhysicalMax:    float64(w.PhysicalMax),
			CritChanceBase: w.CritChanceBase,
			AttackRateBase: w.AttackRateBase,
			Range:          float64(w.Range),
		}
	}
	if a := b.Armour; a != nil {
		e.Armour = &ArmourData{
			BlockChance:         intPtrToFloat(a.BlockChance),
			ArmourBaseMin:       intPtrToFloat(a.ArmourMin),
			ArmourBaseMax:       intPtrToFloat(a.ArmourMax),
			EvasionBaseMin:      intPtrToFloat(a.EvasionMin),
			EvasionBaseMax:      intPtrToFloat(a.EvasionMax),
			EnergyShieldBaseMin: intPtrToFloat(a.EnergyShieldMin),
			EnergyShieldBaseMax: intPtrToFloat(a.EnergyShieldMax),
			MovementPenalty:     intPtrToFloat(a.MovementPenalty),
			WardBaseMin:         intPtrToFloat(a.WardMin),
			WardBaseMax:         intPtrToFloat(a.WardMax),
		}
	}
	if f := b.Flask; f != nil {
		fd := &FlaskData{
			Life:        intPtrToFloat(f.Life),
			Mana:        intPtrToFloat(f.Mana),
			Duration:    f.Duration,
			ChargesUsed: float64(f.ChargesUsed),
			ChargesMax:  float64(f.ChargesMax),
		}
		if f.HasBuff {
			if len(f.Buff) == 0 {
				fd.Buff = []string{""}
			} else {
				fd.Buff = unescapeAll(f.Buff)
			}
		}
		e.Flask = fd
	}
	if tn := b.Tincture; tn != nil {
		e.Tincture = &TinctureData{ManaBurn: tn.ManaBurn, Cooldown: tn.Cooldown}
	}
	e.Req = Req{
		Level: intPtrToFloat(b.ReqLevel),
		Str:   intPtrToFloat(b.ReqStr),
		Dex:   intPtrToFloat(b.ReqDex),
		Int:   intPtrToFloat(b.ReqInt),
	}
	if len(b.FlavourText) > 0 {
		e.FlavourText = unescapeAll(b.FlavourText)
	}
	return e
}

var reBaseLabel = regexp.MustCompile(` \(.+\)`)

// ItemBaseEntry is one data.itemBaseLists list entry.
type ItemBaseEntry struct {
	Label string    `lua:"label"`
	Name  string    `lua:"name"`
	Base  *ItemBase `lua:"base"`
}

func (d *Data) loadBases(src gamedata.BasesData) {
	d.ItemBases = map[string]*ItemBase{}
	for _, typ := range baseItemTypes {
		for _, event := range src.Types[typ] {
			for _, b := range event {
				d.ItemBases[b.DisplayName] = loadItemBase(b)
			}
		}
	}

	// Build lists of item bases, separated by type.
	d.ItemBaseLists = map[string][]ItemBaseEntry{}
	for name, base := range d.ItemBases {
		if base.Hidden {
			continue
		}
		typ := base.Type
		if base.SubType != "" {
			typ = typ + ": " + base.SubType
		}
		d.ItemBaseLists[typ] = append(d.ItemBaseLists[typ], ItemBaseEntry{
			Label: reBaseLabel.ReplaceAllString(name, ""),
			Name:  name,
			Base:  base,
		})
	}
	d.ItemBaseTypeList = nil
	for typ, list := range d.ItemBaseLists {
		d.ItemBaseTypeList = append(d.ItemBaseTypeList, typ)
		sort.Slice(list, func(a, b int) bool {
			al, bl := list[a].Base.Req.Level, list[b].Base.Req.Level
			if (al == nil && bl == nil) || (al != nil && bl != nil && *al == *bl) {
				return list[a].Name < list[b].Name
			}
			av, bv := 1.0, 1.0
			if al != nil {
				av = *al
			}
			if bl != nil {
				bv = *bl
			}
			return av > bv
		})
	}
	sort.Strings(d.ItemBaseTypeList)

	// Rare templates: the generated and hand-written [[...]] blobs, in file
	// order. Long-bracket strings are raw — no escape processing.
	d.Rares = nil
	for _, lines := range src.RareBlobs {
		d.Rares = append(d.Rares, strings.Join(lines, "\n")+"\n")
	}
}
