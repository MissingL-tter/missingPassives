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

func buildMods(x *Ctx) (schema.Document, error) {
	if x.modsDoc != nil {
		return x.modsDoc, nil
	}
	mods, err := x.Dat("Mods")
	if err != nil {
		return nil, err
	}
	x.LoadStatFile("tincture_stat_descriptions.txt", "graft_stat_descriptions.txt")

	// In-memory captures of the generated files for the scripts that reload
	// them (ModTextMap below, uModsToText, mapUniqueToFoulborn).
	x.modItemExclusive = map[string]*modEntry{}
	x.modFoulborn = map[string]*modEntry{}

	buildPool := func(pool modPool) ([]schema.ItemMod, error) {
		var out []schema.ItemMod
		for mod := range mods.Rows() {
			if !pool.Cond(mod) {
				continue
			}
			domain := mod.Int("Domain")
			modId := mod.Str("Id")
			if domain == domainDelveFossil && strings.Contains(pool.Out, "Item") {
				spawnTags := mod.Refs("SpawnTags")
				if spawnTags[0].Str("Id") == "abyss_jewel" && spawnTags[1].Str("Id") == "jewel" && len(spawnTags) == 3 {
					continue
				}
			} else if domain == domainDelveFossil && strings.Contains(pool.Out, "JewelAbyss") {
				if !tagListContainsId(mod.Refs("SpawnTags"), "abyss_jewel") {
					continue
				}
			} else if domain == domainDelveFossil && strings.Contains(pool.Out, "Jewel") {
				if !tagListContainsId(mod.Refs("SpawnTags"), "jewel") {
					continue
				}
			}
			// game data has 0 and 0, which means no description is generated
			if modId == "JewelExpansionPassiveNodes" {
				mod.SetCell("Stat2Value", Interval{2, 12})
			}
			stats, err := x.DescribeMod(mod)
			if err != nil {
				return nil, err
			}
			if len(stats.Orders) == 0 || strings.HasPrefix(stats.Lines[0], "DNT") {
				continue
			}
			genType := mod.Int("GenerationType")
			family := mod.Refs("Family")
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
					} else if len(family) > 1 && strings.Contains(family[1].Str("Id"), "MatchedInfluencesTier") {
						f2 := family[1].Str("Id")
						f1 := family[0].Str("Id")
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
			m.Affix = mod.Str("Name")
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
			m.Level = mod.Int("Level")
			m.Group = mod.Ref("Type").Str("Id")
			m.WeightKey = rowIds(mod.Refs("SpawnTags"))
			m.WeightVal = mod.Ints("SpawnWeights")
			genTags := mod.Refs("GenerationWeightTags")
			if len(genTags) > 0 {
				modTagRows := mod.Refs("Tags")
				if genType == genSuffix && len(modTagRows) > 0 && pool.Out == "ModJewelCluster" &&
					modTagRows[0].Str("Id") == "has_affliction_notable" {
					// make large clusters only have 1 notable suffix
					m.WeightMultiplierKey = append([]string{"has_affliction_notable2"}, rowIds(genTags)...)
					m.WeightMultiplierVal = append([]int64{0}, mod.Ints("GenerationWeightValues")...)
					m.Tags = append([]string{"has_affliction_notable2"}, rowIds(modTagRows)...)
				} else {
					m.WeightMultiplierKey = rowIds(genTags)
					m.WeightMultiplierVal = mod.Ints("GenerationWeightValues")
					if len(modTagRows) > 0 {
						m.Tags = rowIds(modTagRows)
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
				stat := mod.Ref(fmt.Sprintf("Stat%d", modIdx))
				if stat == nil {
					break
				}
				currentStats := map[string]*statVal{}
				iv := mod.Ivl(fmt.Sprintf("Stat%dValue", modIdx))
				currentStats[stat.Str("Id")] = &statVal{min: float64(iv[0]), max: float64(iv[1])}
				if modIdx == 6 {
					break
				}
				bytes := intToBytes(uint32(stat.Int("Hash")))
				// # to # stats consist of two different stats as the min and
				// max have different ranges
				if strings.Contains(stat.Str("Id"), "minimum") {
					if nextStat := mod.Ref(fmt.Sprintf("Stat%d", modIdx+1)); nextStat != nil &&
						strings.Contains(nextStat.Str("Id"), "maximum") {
						modIdx++
						bytes = append(bytes, intToBytes(uint32(nextStat.Int("Hash")))...)
						iv2 := mod.Ivl(fmt.Sprintf("Stat%dValue", modIdx))
						currentStats[nextStat.Str("Id")] = &statVal{min: float64(iv2[0]), max: float64(iv2[1])}
					}
				}
				desc, err := x.DescribeStats(currentStats)
				if err != nil {
					return nil, err
				}
				m.TradeHashes = append(m.TradeHashes, schema.TradeHash{Hash: int64(murmurHash2(bytes, 0x02312233)), Lines: desc.Lines})
				modIdx++
			}
			out = append(out, m)

			if pool.Out == "ModItemExclusive" || pool.Out == "ModFoulborn" {
				var tagIds []string
				for _, t := range mod.Refs("ImplicitTags") {
					tagIds = append(tagIds, t.Str("Id"))
				}
				entry := &modEntry{lines: append([]string(nil), stats.Lines...), orders: stats.Orders, tags: tagIds}
				if pool.Out == "ModItemExclusive" {
					x.modItemExclusive[modId] = entry
				} else {
					x.modFoulborn[modId] = entry
				}
			}
		}
		return out, nil
	}

	doc := &schema.ModsData{Pools: map[string][]schema.ItemMod{}}
	for _, pool := range modPools {
		if doc.Pools[pool.Out], err = buildPool(pool); err != nil {
			return nil, fmt.Errorf("%s: %w", pool.Out, err)
		}
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

// domainOf, genTypeOf and idOf are the three Mods columns the pool conditions
// select on.
func domainOf(mod *Row) int64  { return mod.Int("Domain") }
func genTypeOf(mod *Row) int64 { return mod.Int("GenerationType") }
func idOf(mod *Row) string     { return mod.Str("Id") }

// modPool is one generated pool: the id it is emitted under (the output
// file's basename) and the Mods filter that fills it.
type modPool struct {
	Out  string
	Cond func(mod *Row) bool
}

// modPools is in mods.lua's generation order (the dat mutation and the cache
// captures are order-sensitive).
var modPools = []modPool{
	{Out: "ModExplicit", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainItem &&
			(genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix) &&
			!strings.Contains(idOf(mod), "Royale") &&
			!strings.Contains(idOf(mod), "Necropolis") &&
			!strings.HasPrefix(idOf(mod), "Synthesis") &&
			!(genTypeOf(mod) == genSearingExarch || genTypeOf(mod) == genEaterOfWorlds) &&
			len(mod.Refs("AuraFlags")) == 0
	}},
	{Out: "ModCorrupted", Cond: func(mod *Row) bool {
		return genTypeOf(mod) == genCorrupted && domainOf(mod) == domainItem
	}},
	{Out: "ModDelve", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainDelveFossil
	}},
	{Out: "ModSynthesis", Cond: func(mod *Row) bool {
		return genTypeOf(mod) == genIntrinsic && domainOf(mod) == domainItem && strings.HasPrefix(idOf(mod), "Synthesis")
	}},
	{Out: "ModScourge", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainItem &&
			(genTypeOf(mod) == genScourgeBenefit || genTypeOf(mod) == genScourgeDownside) &&
			!reHellscape.MatchString(idOf(mod))
	}},
	{Out: "ModEldritch", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainItem && (genTypeOf(mod) == genSearingExarch || genTypeOf(mod) == genEaterOfWorlds)
	}},
	{Out: "ModFlask", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainFlask && (genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix)
	}},
	{Out: "ModTincture", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainTincture && (genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix)
	}},
	{Out: "ModJewel", Cond: func(mod *Row) bool {
		return (domainOf(mod) == domainJewel || domainOf(mod) == domainDelveFossil) &&
			(genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix || genTypeOf(mod) == genCorrupted)
	}},
	{Out: "ModJewelAbyss", Cond: func(mod *Row) bool {
		return (domainOf(mod) == domainAbyssJewel || domainOf(mod) == domainDelveFossil) &&
			(genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix || genTypeOf(mod) == genCorrupted)
	}},
	{Out: "ModJewelCluster", Cond: func(mod *Row) bool {
		return (domainOf(mod) == domainClusterJewel && (genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix)) ||
			(domainOf(mod) == domainJewel && genTypeOf(mod) == genCorrupted)
	}},
	{Out: "ModJewelCharm", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainCharm && (genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix)
	}},
	{Out: "WatchersEye", Cond: func(mod *Row) bool {
		family := mod.Refs("Family")
		return len(family) > 0 && (family[0].Str("Id") == "AuraBonus" || family[0].Str("Id") == "ArbalestBonus") &&
			genTypeOf(mod) == genIntrinsic && !strings.HasPrefix(idOf(mod), "Synthesis")
	}},
	{Out: "BoundByDestiny", Cond: func(mod *Row) bool {
		family := mod.Refs("Family")
		return len(family) > 1 && strings.Contains(family[1].Str("Id"), "MatchedInfluencesTier")
	}},
	{Out: "ModVeiled", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainUnveiled && (genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix)
	}},
	{Out: "ModNecropolis", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainItem && strings.HasPrefix(idOf(mod), "NecropolisCrafting")
	}},
	{Out: "ModItemExclusive", Cond: func(mod *Row) bool {
		family := mod.Refs("Family")
		return (domainOf(mod) == domainItem || domainOf(mod) == domainFlask || domainOf(mod) == domainJewel || domainOf(mod) == domainClusterJewel || domainOf(mod) == domainTincture) &&
			genTypeOf(mod) == genIntrinsic &&
			(len(family) > 0 && family[0].Str("Id") != "AuraBonus") &&
			!strings.HasPrefix(idOf(mod), "Synthesis") && !strings.Contains(idOf(mod), "Royale") &&
			!strings.Contains(idOf(mod), "Cowards") && !strings.Contains(idOf(mod), "Map") &&
			!strings.Contains(idOf(mod), "Ultimatum") && !strings.HasPrefix(idOf(mod), "MutatedUnique") &&
			!strings.Contains(idOf(mod), "UNUSED")
	}},
	{Out: "ModGraft", Cond: func(mod *Row) bool {
		return domainOf(mod) == domainGraft && (genTypeOf(mod) == genPrefix || genTypeOf(mod) == genSuffix || genTypeOf(mod) == genCorrupted)
	}},
	{Out: "BeastCraft", Cond: func(mod *Row) bool {
		return strings.Contains(idOf(mod), "Aspect") && genTypeOf(mod) == genSuffix
	}},
	{Out: "ModFoulborn", Cond: func(mod *Row) bool {
		return (domainOf(mod) == domainItem || domainOf(mod) == domainJewel) && genTypeOf(mod) == genIntrinsic &&
			strings.HasPrefix(idOf(mod), "MutatedUnique")
	}},
}

func tagListContainsId(rows []*Row, id string) bool {
	for _, r := range rows {
		if r.Str("Id") == id {
			return true
		}
	}
	return false
}
