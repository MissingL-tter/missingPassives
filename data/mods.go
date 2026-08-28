// data.itemMods and friends: the crafting mod pools, assembled from the
// mods document the way Data.lua loads the generated Mod*.lua files.

package data

import (
	"sort"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func sortStrings(s []string) { sort.Strings(s) }

// ItemModData is one mod pool entry.
type ItemModData struct {
	Lines               []string           `lua:"@array"`
	Type                string             `lua:"type,omitempty"`
	Affix               string             `lua:"affix"`
	StatOrder           []float64          `lua:"statOrder"`
	Level               float64            `lua:"level"`
	Group               string             `lua:"group"`
	WeightKey           []string           `lua:"weightKey"`
	WeightVal           []float64          `lua:"weightVal"`
	WeightMultiplierKey []string           `lua:"weightMultiplierKey"`
	WeightMultiplierVal []float64          `lua:"weightMultiplierVal"`
	Tags                []string           `lua:"tags"`
	ModTags             []string           `lua:"modTags"`
	TradeHashes         map[int64][]string `lua:"tradeHashes"`
}

// itemModPools maps data.itemMods keys to mods-document pool ids.
var itemModPools = map[string]string{
	"Explicit":      "ModExplicit",
	"ItemExclusive": "ModItemExclusive",
	"Corrupted":     "ModCorrupted",
	"Delve":         "ModDelve",
	"Synthesis":     "ModSynthesis",
	"Scourge":       "ModScourge",
	"Eldritch":      "ModEldritch",
	"Flask":         "ModFlask",
	"Tincture":      "ModTincture",
	"Graft":         "ModGraft",
	"Jewel":         "ModJewel",
	"JewelAbyss":    "ModJewelAbyss",
	"JewelCluster":  "ModJewelCluster",
	"JewelCharm":    "ModJewelCharm",
	"Foulborn":      "ModFoulborn",
	"WatchersEye":   "WatchersEye",
}

// itemModsItemKeys lists the pools merged into data.itemMods.Item, in the
// Lua's merge order (later pools overwrite shared ids).
var itemModsItemKeys = []string{"Explicit", "ItemExclusive", "Corrupted", "Delve", "Synthesis", "Scourge", "Eldritch"}

func loadModPool(pool []schema.ItemMod) map[string]ItemModData {
	out := map[string]ItemModData{}
	for _, m := range pool {
		e := ItemModData{
			Lines:     unescapeAll(m.Lines),
			Type:      m.Type,
			Affix:     m.Affix,
			StatOrder: m.StatOrders,
			Level:     float64(m.Level),
			Group:     m.Group,
			WeightKey: emptyIfNil(m.WeightKey),
			WeightVal: intsToFloats(m.WeightVal),
			ModTags:   splitModTags(m.ModTags),
		}
		if len(m.WeightMultiplierKey) > 0 {
			e.WeightMultiplierKey = m.WeightMultiplierKey
			e.WeightMultiplierVal = intsToFloats(m.WeightMultiplierVal)
			if len(m.Tags) > 0 {
				e.Tags = m.Tags
			}
		}
		e.TradeHashes = map[int64][]string{}
		for _, th := range m.TradeHashes {
			// #EVAL: archive parity — the exporter wraps the joined lines in
			// quotes, so zero described lines load as { "" }, not { }.
			if len(th.Lines) == 0 {
				e.TradeHashes[th.Hash] = []string{""}
			} else {
				e.TradeHashes[th.Hash] = unescapeAll(th.Lines)
			}
		}
		out[m.Id] = e
	}
	return out
}

func loadItemMods(src schema.ModsData) {
	ItemMods = map[string]map[string]ItemModData{}
	for key, poolId := range itemModPools {
		ItemMods[key] = loadModPool(src.Pools[poolId])
	}
	VeiledMods = loadModPool(src.Pools["ModVeiled"])
	BeastCraft = loadModPool(src.Pools["BeastCraft"])
	NecropolisMods = loadModPool(src.Pools["ModNecropolis"])
	// kept aside for the generated Bound by Destiny unique
	bbdPool = loadModPool(src.Pools["BoundByDestiny"])

	// combined table of many mod categories
	item := map[string]ItemModData{}
	for _, key := range itemModsItemKeys {
		for id, e := range ItemMods[key] {
			item[id] = e
		}
	}
	ItemMods["Item"] = item

	// data.uniqueMods["Watcher's Eye"]: the WatchersEye pool as a sorted list.
	watchers := ItemMods["WatchersEye"]
	ids := make([]string, 0, len(watchers))
	for id := range watchers {
		ids = append(ids, id)
	}
	sortStrings(ids)
	var list []UniqueModEntry
	for _, id := range ids {
		list = append(list, UniqueModEntry{Id: id, Mod: watchers[id]})
	}
	UniqueMods = map[string][]UniqueModEntry{"Watcher's Eye": list}
}

// UniqueModEntry is one data.uniqueMods list entry.
type UniqueModEntry struct {
	Id  string      `lua:"Id"`
	Mod ItemModData `lua:"mod"`
}
