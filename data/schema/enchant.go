package schema

// Enchants holds the enchantment pools. Each inner map is keyed by source
// ("NORMAL", "CRUEL", ..., "ENKINDLING", "HARVEST", ...); each mod is its
// described stat lines.
type Enchants struct {
	Boots  map[string][][]string `json:"boots"`
	Gloves map[string][][]string `json:"gloves"`
	Belt   map[string][][]string `json:"belt"`
	Flask  map[string][][]string `json:"flask"`
	Body   map[string][][]string `json:"body"`
	Weapon map[string][][]string `json:"weapon"`
	// Helmet is keyed by skill name, then source.
	Helmet map[string]map[string][][]string `json:"helmet"`
}
