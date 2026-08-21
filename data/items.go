// Item-related runtime data: essences, pantheons, crucible, master crafts,
// flavour text and enchantments, assembled from their gamedata documents the
// way Data.lua loads the generated files.

package data

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

// Essence is one data.essences entry (keyed by base item id).
type Essence struct {
	Name string             `lua:"name"`
	Type float64            `lua:"type"`
	Tier float64            `lua:"tier"`
	Mods map[string]string  `lua:"mods"`
}

func loadEssences(src gamedata.Essences) map[string]Essence {
	out := map[string]Essence{}
	for _, e := range src {
		out[e.BaseId] = Essence{
			Name: e.Name,
			Type: float64(e.Type),
			Tier: float64(e.Tier),
			Mods: e.Mods,
		}
	}
	return out
}

// Pantheon is one data.pantheons entry (keyed by god id).
type Pantheon struct {
	IsMajorGod bool                  `lua:"isMajorGod"`
	Souls      map[int]PantheonSoul  `lua:"souls"`
}

type PantheonSoul struct {
	Name string         `lua:"name"`
	Mods []PantheonMod  `lua:"mods"`
}

type PantheonMod struct {
	Line  string    `lua:"line"`
	Value []float64 `lua:"value"`
}

func loadPantheons(src gamedata.Pantheons) map[string]Pantheon {
	out := map[string]Pantheon{}
	for _, p := range src {
		pan := Pantheon{IsMajorGod: p.IsMajorGod, Souls: map[int]PantheonSoul{}}
		for _, g := range p.Gods {
			soul := PantheonSoul{Name: g.Name}
			for _, m := range g.Mods {
				soul.Mods = append(soul.Mods, PantheonMod{
					Line:  luaUnescape(m.Line),
					Value: []float64{float64(m.Value)},
				})
			}
			pan.Souls[g.Index] = soul
		}
		out[p.Id] = pan
	}
	return out
}

// CrucibleNode is one data.crucible entry (keyed by mod id).
type CrucibleNode struct {
	Lines               []string  `lua:"@array"`
	Type                string    `lua:"type,omitempty"`
	Tier                float64   `lua:"tier"`
	StatOrder           []float64 `lua:"statOrder"`
	Level               float64   `lua:"level"`
	Group               string    `lua:"group"`
	NodeType            string    `lua:"nodeType"`
	NodeLocation        []float64 `lua:"nodeLocation"`
	WeightKey           []string  `lua:"weightKey"`
	WeightVal           []float64 `lua:"weightVal"`
	WeightMultiplierKey []string  `lua:"weightMultiplierKey"`
	WeightMultiplierVal []float64 `lua:"weightMultiplierVal"`
	Tags                []string  `lua:"tags"`
	ModTags             []string  `lua:"modTags"`
}

func loadCrucible(src gamedata.CrucibleNodes) map[string]CrucibleNode {
	out := map[string]CrucibleNode{}
	for _, n := range src {
		c := CrucibleNode{
			Lines:     unescapeAll(n.Lines),
			Type:      n.Type,
			Tier:      float64(n.Tier),
			StatOrder: n.StatOrders,
			Level:     float64(n.Level),
			Group:     n.Group,
			NodeType:  n.NodeType,
			NodeLocation: intsToFloats(n.NodeLocation),
			WeightKey:    emptyIfNil(n.WeightKey),
			WeightVal:    intsToFloats(n.WeightVal),
			ModTags:      splitModTags(n.ModTags),
		}
		if len(n.WeightMultiplierKey) > 0 {
			c.WeightMultiplierKey = n.WeightMultiplierKey
			c.WeightMultiplierVal = intsToFloats(n.WeightMultiplierVal)
			if len(n.Tags) > 0 {
				c.Tags = n.Tags
			}
		}
		out[n.ModId] = c
	}
	return out
}

// MasterCraft is one data.masterMods entry.
type MasterCraft struct {
	Lines     []string        `lua:"@array"`
	Type      string          `lua:"type,omitempty"`
	Affix     string          `lua:"affix"`
	ModTags   []string        `lua:"modTags"`
	StatOrder []float64       `lua:"statOrder"`
	Level     float64         `lua:"level"`
	Group     string          `lua:"group"`
	Types     map[string]bool `lua:"types"`
}

func loadMasterMods(src gamedata.MasterCrafts) []MasterCraft {
	out := make([]MasterCraft, 0, len(src))
	for _, c := range src {
		types := map[string]bool{}
		for _, t := range c.Types {
			types[t] = true
		}
		out = append(out, MasterCraft{
			Lines:     unescapeAll(c.Lines),
			Type:      c.Type,
			Affix:     c.Affix,
			ModTags:   splitModTags(c.ModTags),
			StatOrder: c.StatOrders,
			Level:     float64(c.Level),
			Group:     c.Group,
			Types:     types,
		})
	}
	return out
}

// FlavourText is one data.flavourText entry.
type FlavourText struct {
	Id   string   `lua:"id"`
	Name string   `lua:"name"`
	Text []string `lua:"text"`
}

func loadFlavourText(src gamedata.FlavourTexts) []FlavourText {
	out := make([]FlavourText, 0, len(src))
	for _, ft := range src {
		out = append(out, FlavourText{Id: ft.Id, Name: ft.Name, Text: unescapeAll(ft.Text)})
	}
	return out
}

// loadEnchantments builds data.enchantments: the seven generated pools plus
// the Flask alias and the per-weapon-type expansion Data.lua performs.
func loadEnchantments(src gamedata.Enchants) map[string]map[string][]string {
	pool := func(m map[string][][]string) map[string][]string {
		out := map[string][]string{}
		for source, mods := range m {
			list := make([]string, 0, len(mods))
			for _, lines := range mods {
				list = append(list, luaUnescape(joinSlash(lines)))
			}
			out[source] = list
		}
		return out
	}
	out := map[string]map[string][]string{
		"Boots":        pool(src.Boots),
		"Gloves":       pool(src.Gloves),
		"Belt":         pool(src.Belt),
		"Body Armour":  pool(src.Body),
		"Weapon":       pool(src.Weapon),
		"UtilityFlask": pool(src.Flask),
	}
	out["Flask"] = out["UtilityFlask"]
	// Weapon lists are plain strings, so every weapon base type gets the
	// Weapon table's lists (the table-typed branch of the Lua loop is dead
	// with the current data).
	for baseType := range weaponTypeInfo {
		out[baseType] = map[string][]string{}
		for source, list := range out["Weapon"] {
			out[baseType][source] = list
		}
	}
	return out
}

// HelmetEnchants is data.enchantments["Helmet"]: skill -> source -> mods.
func loadHelmetEnchants(src gamedata.Enchants) map[string]map[string][]string {
	out := map[string]map[string][]string{}
	for skill, bySource := range src.Helmet {
		m := map[string][]string{}
		for source, mods := range bySource {
			list := make([]string, 0, len(mods))
			for _, lines := range mods {
				list = append(list, luaUnescape(joinSlash(lines)))
			}
			m[source] = list
		}
		out[skill] = m
	}
	return out
}

func joinSlash(lines []string) string {
	s := ""
	for i, l := range lines {
		if i > 0 {
			s += "/"
		}
		s += l
	}
	return s
}

func unescapeAll(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = luaUnescape(l)
	}
	return out
}

func intsToFloats(vals []int64) []float64 {
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i] = float64(v)
	}
	return out
}

func emptyIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// splitModTags parses a describeModTags string (`"a", "b"` or empty) back
// into the tag list Lua sees after loading it inside `{ ... }`.
func splitModTags(s string) []string {
	if s == "" {
		return []string{}
	}
	var out []string
	for _, part := range strings.Split(s, ", ") {
		out = append(out, strings.Trim(part, "\""))
	}
	return out
}
