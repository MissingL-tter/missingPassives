package gamedata

// ModsData is the mods script's document: the twenty mod pools plus the
// unique mod text map (Export/Uniques/ModTextMap.lua).
type ModsData struct {
	// Pools is keyed by pool id (the output basename: "ModExplicit",
	// "WatchersEye", "BeastCraft", ...); each pool is in Mods row order.
	Pools   map[string][]ItemMod `json:"pools"`
	TextMap map[string][]string  `json:"textMap"` // lower-cased line -> mod ids
}

type ItemMod struct {
	Id                  string      `json:"id"`
	Type                string      `json:"type,omitempty"`
	Affix               string      `json:"affix"`
	Lines               []string    `json:"lines"`
	StatOrders          []float64   `json:"statOrders"`
	Level               int64       `json:"level"`
	Group               string      `json:"group"`
	WeightKey           []string    `json:"weightKey"`
	WeightVal           []int64     `json:"weightVal"`
	WeightMultiplierKey []string    `json:"weightMultiplierKey,omitempty"`
	WeightMultiplierVal []int64     `json:"weightMultiplierVal,omitempty"`
	Tags                []string    `json:"tags,omitempty"`
	ModTags             string      `json:"modTags"`
	TradeHashes         []TradeHash `json:"tradeHashes"`
}

// TradeHash carries one trade-site stat id hash and its described lines.
// #EVAL: the order of these entries is a LuaJIT hash-table artifact preserved
// for archive parity; sort by hash once the format is Go-owned.
type TradeHash struct {
	Hash  int64    `json:"hash"`
	Lines []string `json:"lines"`
}
