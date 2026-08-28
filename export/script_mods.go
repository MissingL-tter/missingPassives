// Port of .archive/src/Export/Scripts/mods.lua.

package export

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "mods", Build: buildMods})
}

// Modifier domains and generation types (poewiki).
const (
	domainItem         = 1
	domainFlask        = 2
	domainJewel        = 10
	domainAbyssJewel   = 13
	domainDelveFossil  = 16
	domainClusterJewel = 21
	domainUnveiled     = 28
	domainTincture     = 34
	domainCharm        = 35
	domainGraft        = 38

	genPrefix          = 1
	genSuffix          = 2
	genIntrinsic       = 3
	genCorrupted       = 5
	genEssence         = 11
	genScourgeBenefit  = 24
	genScourgeDownside = 25
	genSearingExarch   = 28
	genEaterOfWorlds   = 29
)

func intToBytes(v uint32) []byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	return b[:]
}

// murmurHash2 matches Common.lua's murmurHash2 (canonical MurmurHash2, with
// a zero-padded 4-byte tail read the way bytesToInt pads).
func murmurHash2(key []byte, seed uint32) uint32 {
	const m = 0x5bd1e995
	h := seed ^ uint32(len(key))
	for ; len(key) >= 4; key = key[4:] {
		k := binary.LittleEndian.Uint32(key)
		k *= m
		k ^= k >> 24
		k *= m
		h *= m
		h ^= k
	}
	if len(key) > 0 {
		var buf [4]byte
		copy(buf[:], key)
		h ^= binary.LittleEndian.Uint32(buf[:])
		h *= m
	}
	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}

var (
	reDigits    = regexp.MustCompile(`[0-9]+`)
	reHellscape = regexp.MustCompile(`^Hellscape[UpDown]+sideMap`)
	reNumberish = regexp.MustCompile(`-*[0-9]*\.*[0-9]+-*-*[0-9]*\.*[0-9]*`)
)

// rowIdList collects tag ids the way writeTagIds did (ipairs semantics).
func rowIdList(cell any) []string {
	var out []string
	for _, tag := range listRows(cell) {
		out = append(out, luaStr(tag.Get("Id")))
	}
	return out
}

// listLen is Lua # on a list cell whose nil elements never became keys.
func listLen(cell any) int {
	return len(listRows(cell))
}

func buildMods(x *Ctx) (any, error) {
	if x.modsDoc != nil {
		return x.modsDoc, nil
	}
	x.LoadStatFile("tincture_stat_descriptions.txt", "graft_stat_descriptions.txt")

	// In-memory captures of the generated files for the scripts that reload
	// them (ModTextMap below, uModsToText, mapUniqueToFoulborn).
	x.modItemExclusive = map[string]*modEntry{}
	x.modFoulborn = map[string]*modEntry{}

	buildPool := func(outName string) []schema.ItemMod {
		var pool []schema.ItemMod
		condFunc := modPoolConds[outName]
		x.Dat("Mods").Rows(func(mod *Row) bool {
			if !condFunc(mod) {
				return true
			}
			domain := mod.Get("Domain").(int64)
			modId := luaStr(mod.Get("Id"))
			if domain == domainDelveFossil && strings.Contains(outName, "Item") {
				spawnTags := listRows(mod.Get("SpawnTags"))
				if luaStr(spawnTags[0].Get("Id")) == "abyss_jewel" && luaStr(spawnTags[1].Get("Id")) == "jewel" && len(spawnTags) == 3 {
					return true
				}
			} else if domain == domainDelveFossil && strings.Contains(outName, "JewelAbyss") {
				if !tagListContainsId(mod.Get("SpawnTags"), "abyss_jewel") {
					return true
				}
			} else if domain == domainDelveFossil && strings.Contains(outName, "Jewel") {
				if !tagListContainsId(mod.Get("SpawnTags"), "jewel") {
					return true
				}
			}
			// game data has 0 and 0, which means no description is generated
			if modId == "JewelExpansionPassiveNodes" {
				mod.SetCell("Stat2Value", Interval{2, 12})
			}
			stats := x.DescribeMod(mod)
			if len(stats.Orders) == 0 || strings.HasPrefix(stats.Lines[0], "DNT") {
				return true
			}
			genType := mod.Get("GenerationType").(int64)
			family := listRows(mod.Get("Family"))
			m := schema.ItemMod{Id: modId}
			switch genType {
			case genPrefix:
				m.Type = "Prefix"
			case genSuffix:
				m.Type = "Suffix"
			case genIntrinsic:
				if domain == domainItem {
					if strings.HasPrefix(modId, "Synthesis") {
						m.Type = "Synthesis"
					} else if len(family) > 1 && strings.Contains(luaStr(family[1].Get("Id")), "MatchedInfluencesTier") {
						f2 := luaStr(family[1].Get("Id"))
						f1 := luaStr(family[0].Get("Id"))
						prefix := f1
						if idx := strings.Index(f1, "Influence"); idx >= 0 {
							prefix = f1[:idx]
						}
						m.Type = reDigits.FindString(f2) + prefix
					}
				} else if domain == domainDelveFossil {
					m.Type = "DelveImplicit"
				}
			case genCorrupted:
				m.Type = "Corrupted"
			case genScourgeBenefit:
				m.Type = "ScourgeUpside"
			case genScourgeDownside:
				m.Type = "ScourgeDownside"
			case genSearingExarch:
				m.Type = "Exarch"
			case genEaterOfWorlds:
				m.Type = "Eater"
			}
			m.Affix = luaStr(mod.Get("Name"))
			if strings.Contains(modId, "EldritchImplicitUniquePresence") && len(stats.Lines) > 0 {
				for i, stat := range stats.Lines {
					stats.Lines[i] = "While a Unique Enemy is in your Presence, " + stat
				}
			}
			if strings.Contains(modId, "EldritchImplicitPinnaclePresence") && len(stats.Lines) > 0 {
				for i, stat := range stats.Lines {
					stats.Lines[i] = "While a Pinnacle Atlas Boss is in your Presence, " + stat
				}
			}
			m.Lines = stats.Lines
			m.StatOrders = stats.Orders
			m.Level = mod.Get("Level").(int64)
			m.Group = luaStr(mod.Get("Type").(*Row).Get("Id"))
			m.WeightKey = rowIdList(mod.Get("SpawnTags"))
			m.WeightVal = intCells(mod.Get("SpawnWeights").([]any))
			genTags := listRows(mod.Get("GenerationWeightTags"))
			if len(genTags) > 0 {
				modTagRows := listRows(mod.Get("Tags"))
				if genType == genSuffix && len(modTagRows) > 0 && outName == "../Data/ModJewelCluster.lua" &&
					luaStr(modTagRows[0].Get("Id")) == "has_affliction_notable" {
					// make large clusters only have 1 notable suffix
					m.WeightMultiplierKey = append([]string{"has_affliction_notable2"}, rowIdList(mod.Get("GenerationWeightTags"))...)
					m.WeightMultiplierVal = append([]int64{0}, intCells(mod.Get("GenerationWeightValues").([]any))...)
					m.Tags = append([]string{"has_affliction_notable2"}, rowIdList(mod.Get("Tags"))...)
				} else {
					m.WeightMultiplierKey = rowIdList(mod.Get("GenerationWeightTags"))
					m.WeightMultiplierVal = intCells(mod.Get("GenerationWeightValues").([]any))
					if len(modTagRows) > 0 {
						m.Tags = rowIdList(mod.Get("Tags"))
					}
				}
			}
			m.ModTags = stats.ModTags

			// Timeless jewels have special trade ids; see
			// https://www.pathofexile.com/api/trade/data/stats
			// Entries carry the mod's stat order; the reference file's
			// LuaJIT hash-iteration order is reproduced in the render test.
			modIdx := 1
			for {
				stat, ok := mod.Get(fmt.Sprintf("Stat%d", modIdx)).(*Row)
				if !ok {
					break
				}
				currentStats := map[string]*statVal{}
				iv := mod.Get(fmt.Sprintf("Stat%dValue", modIdx)).(Interval)
				currentStats[luaStr(stat.Get("Id"))] = &statVal{min: float64(iv[0]), max: float64(iv[1])}
				if modIdx == 6 {
					break
				}
				bytes := intToBytes(uint32(stat.Get("Hash").(int64)))
				// # to # stats consist of two different stats as the min and
				// max have different ranges
				if strings.Contains(luaStr(stat.Get("Id")), "minimum") {
					if nextStat, ok := mod.Get(fmt.Sprintf("Stat%d", modIdx+1)).(*Row); ok &&
						strings.Contains(luaStr(nextStat.Get("Id")), "maximum") {
						modIdx++
						bytes = append(bytes, intToBytes(uint32(nextStat.Get("Hash").(int64)))...)
						iv2 := mod.Get(fmt.Sprintf("Stat%dValue", modIdx)).(Interval)
						currentStats[luaStr(nextStat.Get("Id"))] = &statVal{min: float64(iv2[0]), max: float64(iv2[1])}
					}
				}
				desc := x.DescribeStats(currentStats)
				m.TradeHashes = append(m.TradeHashes, schema.TradeHash{Hash: int64(murmurHash2(bytes, 0x02312233)), Lines: desc.Lines})
				modIdx++
			}
			pool = append(pool, m)

			if outName == "../Data/ModItemExclusive.lua" || outName == "../Data/ModFoulborn.lua" {
				var tagIds []string
				for _, t := range listRows(mod.Get("ImplicitTags")) {
					tagIds = append(tagIds, luaStr(t.Get("Id")))
				}
				entry := &modEntry{lines: append([]string(nil), stats.Lines...), orders: stats.Orders, tags: tagIds}
				if outName == "../Data/ModItemExclusive.lua" {
					x.modItemExclusive[modId] = entry
				} else {
					x.modFoulborn[modId] = entry
				}
			}
			return true
		})
		return pool
	}

	doc := schema.ModsData{Pools: map[string][]schema.ItemMod{}}
	for _, outName := range modPoolOrder {
		doc.Pools[poolId(outName)] = buildPool(outName)
	}

	// Generate unique mod mappings from text to mod.
	doc.TextMap = map[string][]string{}
	modNames := make([]string, 0, len(x.modItemExclusive))
	for name := range x.modItemExclusive {
		modNames = append(modNames, name)
	}
	sort.Strings(modNames)
	for _, modName := range modNames {
		first := x.modItemExclusive[modName].lines[0]
		if strings.HasPrefix(first, "DNT") {
			continue
		}
		lower := strings.ToLower(first)
		doc.TextMap[lower] = append(doc.TextMap[lower], modName)
		// Add generic mod for matching legacy values
		if strings.ContainsAny(first, "0123456789") {
			genericText := reNumberish.ReplaceAllString(first, "#")
			if genericText != first {
				lower := strings.ToLower(genericText)
				doc.TextMap[lower] = append(doc.TextMap[lower], modName)
			}
		}
	}
	x.modsDoc = doc
	return doc, nil
}

// poolId is the output basename: "../Data/ModExplicit.lua" -> "ModExplicit".
func poolId(outName string) string {
	base := outName[strings.LastIndex(outName, "/")+1:]
	return strings.TrimSuffix(base, ".lua")
}

// modPoolOrder preserves mods.lua's generation order (the dat mutation and
// cache captures are order-sensitive).
var modPoolOrder = []string{
	"../Data/ModExplicit.lua", "../Data/ModCorrupted.lua", "../Data/ModDelve.lua", "../Data/ModSynthesis.lua",
	"../Data/ModScourge.lua", "../Data/ModEldritch.lua", "../Data/ModFlask.lua", "../Data/ModTincture.lua",
	"../Data/ModJewel.lua", "../Data/ModJewelAbyss.lua", "../Data/ModJewelCluster.lua", "../Data/ModJewelCharm.lua",
	"../Data/Uniques/Special/WatchersEye.lua", "../Data/Uniques/Special/BoundByDestiny.lua",
	"../Data/ModVeiled.lua", "../Data/ModNecropolis.lua", "../Data/ModItemExclusive.lua", "../Data/ModGraft.lua",
	"../Data/BeastCraft.lua", "../Data/ModFoulborn.lua",
}

var modPoolConds = map[string]func(mod *Row) bool{}

func init() {
	dom := func(mod *Row) int64 { return mod.Get("Domain").(int64) }
	gen := func(mod *Row) int64 { return mod.Get("GenerationType").(int64) }
	id := func(mod *Row) string { return luaStr(mod.Get("Id")) }
	modPoolConds["../Data/ModExplicit.lua"] = func(mod *Row) bool {
		return dom(mod) == domainItem &&
			(gen(mod) == genPrefix || gen(mod) == genSuffix) &&
			!strings.Contains(id(mod), "Royale") &&
			!strings.Contains(id(mod), "Necropolis") &&
			!strings.HasPrefix(id(mod), "Synthesis") &&
			!(gen(mod) == genSearingExarch || gen(mod) == genEaterOfWorlds) &&
			listLen(mod.Get("AuraFlags")) == 0
	}
	modPoolConds["../Data/ModCorrupted.lua"] = func(mod *Row) bool {
		return gen(mod) == genCorrupted && dom(mod) == domainItem
	}
	modPoolConds["../Data/ModDelve.lua"] = func(mod *Row) bool {
		return dom(mod) == domainDelveFossil
	}
	modPoolConds["../Data/ModSynthesis.lua"] = func(mod *Row) bool {
		return gen(mod) == genIntrinsic && dom(mod) == domainItem && strings.HasPrefix(id(mod), "Synthesis")
	}
	modPoolConds["../Data/ModScourge.lua"] = func(mod *Row) bool {
		return dom(mod) == domainItem &&
			(gen(mod) == genScourgeBenefit || gen(mod) == genScourgeDownside) &&
			!reHellscape.MatchString(id(mod))
	}
	modPoolConds["../Data/ModEldritch.lua"] = func(mod *Row) bool {
		return dom(mod) == domainItem && (gen(mod) == genSearingExarch || gen(mod) == genEaterOfWorlds)
	}
	modPoolConds["../Data/ModFlask.lua"] = func(mod *Row) bool {
		return dom(mod) == domainFlask && (gen(mod) == genPrefix || gen(mod) == genSuffix)
	}
	modPoolConds["../Data/ModTincture.lua"] = func(mod *Row) bool {
		return dom(mod) == domainTincture && (gen(mod) == genPrefix || gen(mod) == genSuffix)
	}
	modPoolConds["../Data/ModJewel.lua"] = func(mod *Row) bool {
		return (dom(mod) == domainJewel || dom(mod) == domainDelveFossil) &&
			(gen(mod) == genPrefix || gen(mod) == genSuffix || gen(mod) == genCorrupted)
	}
	modPoolConds["../Data/ModJewelAbyss.lua"] = func(mod *Row) bool {
		return (dom(mod) == domainAbyssJewel || dom(mod) == domainDelveFossil) &&
			(gen(mod) == genPrefix || gen(mod) == genSuffix || gen(mod) == genCorrupted)
	}
	modPoolConds["../Data/ModJewelCluster.lua"] = func(mod *Row) bool {
		return (dom(mod) == domainClusterJewel && (gen(mod) == genPrefix || gen(mod) == genSuffix)) ||
			(dom(mod) == domainJewel && gen(mod) == genCorrupted)
	}
	modPoolConds["../Data/ModJewelCharm.lua"] = func(mod *Row) bool {
		return dom(mod) == domainCharm && (gen(mod) == genPrefix || gen(mod) == genSuffix)
	}
	modPoolConds["../Data/Uniques/Special/WatchersEye.lua"] = func(mod *Row) bool {
		family := listRows(mod.Get("Family"))
		return len(family) > 0 && (luaStr(family[0].Get("Id")) == "AuraBonus" || luaStr(family[0].Get("Id")) == "ArbalestBonus") &&
			gen(mod) == genIntrinsic && !strings.HasPrefix(id(mod), "Synthesis")
	}
	modPoolConds["../Data/Uniques/Special/BoundByDestiny.lua"] = func(mod *Row) bool {
		family := listRows(mod.Get("Family"))
		return len(family) > 1 && strings.Contains(luaStr(family[1].Get("Id")), "MatchedInfluencesTier")
	}
	modPoolConds["../Data/ModVeiled.lua"] = func(mod *Row) bool {
		return dom(mod) == domainUnveiled && (gen(mod) == genPrefix || gen(mod) == genSuffix)
	}
	modPoolConds["../Data/ModNecropolis.lua"] = func(mod *Row) bool {
		return dom(mod) == domainItem && strings.HasPrefix(id(mod), "NecropolisCrafting")
	}
	modPoolConds["../Data/ModItemExclusive.lua"] = func(mod *Row) bool {
		family := listRows(mod.Get("Family"))
		return (dom(mod) == domainItem || dom(mod) == domainFlask || dom(mod) == domainJewel || dom(mod) == domainClusterJewel || dom(mod) == domainTincture) &&
			gen(mod) == genIntrinsic &&
			(len(family) > 0 && luaStr(family[0].Get("Id")) != "AuraBonus") &&
			!strings.HasPrefix(id(mod), "Synthesis") && !strings.Contains(id(mod), "Royale") &&
			!strings.Contains(id(mod), "Cowards") && !strings.Contains(id(mod), "Map") &&
			!strings.Contains(id(mod), "Ultimatum") && !strings.HasPrefix(id(mod), "MutatedUnique") &&
			!strings.Contains(id(mod), "UNUSED")
	}
	modPoolConds["../Data/ModGraft.lua"] = func(mod *Row) bool {
		return dom(mod) == domainGraft && (gen(mod) == genPrefix || gen(mod) == genSuffix || gen(mod) == genCorrupted)
	}
	modPoolConds["../Data/BeastCraft.lua"] = func(mod *Row) bool {
		return strings.Contains(id(mod), "Aspect") && gen(mod) == genSuffix
	}
	modPoolConds["../Data/ModFoulborn.lua"] = func(mod *Row) bool {
		return (dom(mod) == domainItem || dom(mod) == domainJewel) && gen(mod) == genIntrinsic &&
			strings.HasPrefix(id(mod), "MutatedUnique")
	}
}

func tagListContainsId(cell any, id string) bool {
	for _, r := range listRows(cell) {
		if luaStr(r.Get("Id")) == id {
			return true
		}
	}
	return false
}
