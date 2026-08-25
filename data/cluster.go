// data.clusterJewels and the derived notable lookup, from the cluster
// document.

package data

import (
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

type clusterJewels struct {
	Jewels           map[string]ClusterJewelSize `lua:"jewels"`
	NotableSortOrder map[string]float64          `lua:"notableSortOrder"`
	Keystones        []string                    `lua:"keystones"`
	OrbitOffsets     map[int64]map[int]float64   `lua:"orbitOffsets"`
}

type ClusterJewelSize struct {
	Size            string                      `lua:"size"`
	SizeIndex       float64                     `lua:"sizeIndex"`
	MinNodes        float64                     `lua:"minNodes"`
	MaxNodes        float64                     `lua:"maxNodes"`
	SmallIndicies   []float64                   `lua:"smallIndicies"`
	NotableIndicies []float64                   `lua:"notableIndicies"`
	SocketIndicies  []float64                   `lua:"socketIndicies"`
	TotalIndicies   float64                     `lua:"totalIndicies"`
	Skills          map[string]ClusterSkillData `lua:"skills"`
}

type ClusterSkillData struct {
	Name        string   `lua:"name"`
	Icon        string   `lua:"icon"`
	MasteryIcon *string  `lua:"masteryIcon"`
	Tag         string   `lua:"tag"`
	Stats       []string `lua:"stats"`
	Enchant     []string `lua:"enchant"`
}

func loadClusterJewels(src gamedata.ClusterJewels) clusterJewels {
	out := clusterJewels{
		Jewels:           map[string]ClusterJewelSize{},
		NotableSortOrder: map[string]float64{},
		Keystones:        src.Keystones,
		OrbitOffsets:     map[int64]map[int]float64{},
	}
	for _, j := range src.Jewels {
		size := ClusterJewelSize{
			Size:            j.Size,
			SizeIndex:       float64(j.SizeIndex),
			MinNodes:        float64(j.MinNodes),
			MaxNodes:        float64(j.MaxNodes),
			SmallIndicies:   intsToFloats(j.SmallIndicies),
			NotableIndicies: intsToFloats(j.NotableIndicies),
			SocketIndicies:  intsToFloats(j.SocketIndicies),
			TotalIndicies:   float64(j.TotalIndicies),
			Skills:          map[string]ClusterSkillData{},
		}
		for _, s := range j.Skills {
			// The "stats = { "" }" join artifact would apply to stat-less
			// skills, mirrored here for parity with the loaded file.
			stats := unescapeAll(s.Stats)
			if len(stats) == 0 {
				stats = []string{""}
			}
			enchant := make([]string, 0, len(s.Stats))
			for _, line := range s.Stats {
				enchant = append(enchant, luaUnescape("Added Small Passive Skills grant: "+line))
			}
			size.Skills[s.Id] = ClusterSkillData{
				Name:        s.Name,
				Icon:        s.Icon,
				MasteryIcon: s.MasteryIcon,
				Tag:         s.Tag,
				Stats:       stats,
				Enchant:     enchant,
			}
		}
		out.Jewels[j.Name] = size
	}
	for _, n := range src.NotableSortOrder {
		out.NotableSortOrder[n.Name] = float64(n.Order)
	}
	for _, o := range src.OrbitOffsets {
		starts := map[int]float64{}
		for i, s := range o.Starts {
			if i > 2 {
				break
			}
			starts[i] = float64(s)
		}
		out.OrbitOffsets[o.NodeId] = starts
	}
	return out
}

// ClusterJewelInfo is one data.clusterJewelInfoForNotable entry. JewelTypes
// is kept sorted: the reference builds it in Lua hash-iteration order, which
// the archive comparison normalises the same way (a documented deliberate
// divergence).
type ClusterJewelInfo struct {
	JewelTypes []string        `lua:"jewelTypes"`
	Size       map[string]bool `lua:"size"`
}

func computeClusterJewelInfo(jewelClusterMods map[string]ItemModData, jewels clusterJewels) map[string]*ClusterJewelInfo {
	// cluster jewel skill -> the notables which use that skill
	clusterSkillToNotables := map[string][]string{}
	for _, notableInfo := range jewelClusterMods {
		if len(notableInfo.Lines) == 0 {
			continue
		}
		// Translate the notable key to its name.
		idx := strings.Index(notableInfo.Lines[0], "1 Added Passive Skill is ")
		if idx < 0 {
			continue
		}
		notableName := notableInfo.Lines[0][idx+len("1 Added Passive Skill is "):]
		for weightIndex, clusterSkill := range notableInfo.WeightKey {
			if weightIndex < len(notableInfo.WeightVal) && notableInfo.WeightVal[weightIndex] > 0 {
				clusterSkillToNotables[clusterSkill] = append(clusterSkillToNotables[clusterSkill], notableName)
			}
		}
	}
	out := map[string]*ClusterJewelInfo{}
	for size, jewel := range jewels.Jewels {
		for skill := range jewel.Skills {
			for _, notableKey := range clusterSkillToNotables[skill] {
				info := out[notableKey]
				if info == nil {
					info = &ClusterJewelInfo{Size: map[string]bool{}}
					out[notableKey] = info
				}
				info.Size[size] = true
				info.JewelTypes = append(info.JewelTypes, skill)
			}
		}
	}
	for _, info := range out {
		sort.Strings(info.JewelTypes)
	}
	return out
}
