// Renders schema.ModsData as the twenty Data/Mod*.lua pool files plus
// Export/Uniques/ModTextMap.lua (Scripts/mods.lua's outputs).

package luarender

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() { register("mods", renderMods) }

// modPoolPaths maps pool ids to their output paths.
var modPoolPaths = map[string]string{
	"ModExplicit": "Data/ModExplicit.lua", "ModCorrupted": "Data/ModCorrupted.lua",
	"ModDelve": "Data/ModDelve.lua", "ModSynthesis": "Data/ModSynthesis.lua",
	"ModScourge": "Data/ModScourge.lua", "ModEldritch": "Data/ModEldritch.lua",
	"ModFlask": "Data/ModFlask.lua", "ModTincture": "Data/ModTincture.lua",
	"ModJewel": "Data/ModJewel.lua", "ModJewelAbyss": "Data/ModJewelAbyss.lua",
	"ModJewelCluster": "Data/ModJewelCluster.lua", "ModJewelCharm": "Data/ModJewelCharm.lua",
	"WatchersEye":    "Data/Uniques/Special/WatchersEye.lua",
	"BoundByDestiny": "Data/Uniques/Special/BoundByDestiny.lua",
	"ModVeiled":      "Data/ModVeiled.lua", "ModNecropolis": "Data/ModNecropolis.lua",
	"ModItemExclusive": "Data/ModItemExclusive.lua", "ModGraft": "Data/ModGraft.lua",
	"BeastCraft": "Data/BeastCraft.lua", "ModFoulborn": "Data/ModFoulborn.lua",
}

func renderMods(d schema.ModsData, _ Templates) (map[string]string, error) {
	files := map[string]string{}
	for id, pool := range d.Pools {
		path, ok := modPoolPaths[id]
		if !ok {
			return nil, fmt.Errorf("mods: unknown pool %q", id)
		}
		var b B
		b.itemHeader()
		for _, m := range pool {
			b.W("\t[\"", m.Id, "\"] = { ")
			if m.Type != "" {
				b.W("type = \"", m.Type, "\", ")
			}
			b.W("affix = \"", m.Affix, "\", ")
			b.W("\"", strings.Join(m.Lines, "\", \""), "\", ")
			b.W("statOrder = { ", nums(m.StatOrders, ", "), " }, ")
			b.W("level = ", m.Level, ", group = \"", m.Group, "\", ")
			b.W("weightKey = { ")
			for _, tag := range m.WeightKey {
				b.W("\"", tag, "\", ")
			}
			b.W("}, ")
			b.W("weightVal = { ", ints(m.WeightVal, ", "), " }, ")
			if len(m.WeightMultiplierKey) > 0 {
				b.W("weightMultiplierKey = { ")
				for _, tag := range m.WeightMultiplierKey {
					b.W("\"", tag, "\", ")
				}
				b.W("}, ")
				b.W("weightMultiplierVal = { ", ints(m.WeightMultiplierVal, ", "), " }, ")
				if len(m.Tags) > 0 {
					b.W("tags = { ")
					for _, tag := range m.Tags {
						b.W("\"", tag, "\", ")
					}
					b.W("}, ")
				}
			}
			b.W("modTags = { ", m.ModTags, " }, ")
			// The document carries the mod's stat order; the reference file
			// wrote pairs() over a hash table — replay the insertions and
			// walk LuaJIT's iteration order.
			th := &ljTab{}
			for _, e := range m.TradeHashes {
				th.Set(float64(e.Hash), e.Lines)
			}
			b.W("tradeHashes = { ")
			th.Pairs(func(hash float64, v any) {
				b.W(fmt.Sprintf("[%d] = { %s }, ", int64(hash), "\""+strings.Join(v.([]string), "\", \"")+"\""))
			})
			b.W("} ")
			b.W("},\n")
		}
		b.W("}")
		files[path] = b.String()
	}

	var tb B
	tb.itemHeader()
	keys := make([]string, 0, len(d.TextMap))
	for k := range d.TextMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		tb.W("\t[\"", key, "\"] = { ")
		for _, modName := range d.TextMap[key] {
			tb.W("\"", modName, "\", ")
		}
		tb.W("},\n")
	}
	tb.W("\n}")
	files["Export/Uniques/ModTextMap.lua"] = tb.String()
	return files, nil
}
