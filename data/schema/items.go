package schema

// Essences holds essence crafting data, in Essences row order (tier > 0
// only).
type Essences []Essence

func (Essences) isDocument() {}

type Essence struct {
	BaseId string            `json:"baseId"`
	Name   string            `json:"name"`
	Type   int               `json:"type"`
	Tier   int64             `json:"tier"`
	Mods   map[string]string `json:"mods"` // item type -> mod id
}

// ModScalability maps a described mod line to the scalability of each of its
// values.
type ModScalability map[string][]Scalability

func (ModScalability) isDocument() {}

type Scalability struct {
	IsScalable bool     `json:"isScalable"`
	Formats    []string `json:"formats"` // nil when the value carries no formats
}

// MasterCrafts holds the crafting bench options, in CraftingBenchOptions row
// order (enabled rows with a mod only).
type MasterCrafts []MasterCraft

func (MasterCrafts) isDocument() {}

type MasterCraft struct {
	Type       string    `json:"type"` // "Prefix", "Suffix" or ""
	Affix      string    `json:"affix"`
	ModTags    string    `json:"modTags"` // described tag list, pre-joined
	Lines      []string  `json:"lines"`
	StatOrders []float64 `json:"statOrders"`
	Level      int64     `json:"level"`
	Group      string    `json:"group"`
	Types      []string  `json:"types"` // craftable item types, first-seen order
}

// CrucibleNodes holds the crucible passive data, in WeaponPassiveSkills row
// order (HardMode and stat-less mods excluded).
type CrucibleNodes []CrucibleNode

func (CrucibleNodes) isDocument() {}

type CrucibleNode struct {
	ModId               string    `json:"modId"`
	Type                string    `json:"type"` // "Spawn", "MergeOnly" or ""
	Tier                int64     `json:"tier"`
	Lines               []string  `json:"lines"`
	StatOrders          []float64 `json:"statOrders"`
	Level               int64     `json:"level"`
	Group               string    `json:"group"`
	NodeType            string    `json:"nodeType"`
	NodeLocation        []int64   `json:"nodeLocation"`
	WeightKey           []string  `json:"weightKey"`
	WeightVal           []int64   `json:"weightVal"`
	WeightMultiplierKey []string  `json:"weightMultiplierKey"` // nil when absent
	WeightMultiplierVal []int64   `json:"weightMultiplierVal"`
	Tags                []string  `json:"tags"`
	ModTags             string    `json:"modTags"`
}
