// BuildConquerTables generates data/raw/conquertables.json — the historic
// jewel (conquering) generation inputs — straight from the GGPK dat tables
// (AlternatePassiveSkills / AlternatePassiveAdditions /
// AlternateTreeVersions / PassiveSkills), replacing the retired python
// generator that trimmed third-party dat-table JSON exports. Field names
// mirror the dat-table vocabulary the generation algorithm was reverse-
// engineered against; the historic differentials (33.1M timeless cells +
// every abyss record) verify the content.
package export

import (
	"encoding/json"
	"fmt"
	"sort"
)

type ctVersion struct {
	K             int   `json:"k"`
	SmallAttr     bool  `json:"smallAttr"`
	SmallNorm     bool  `json:"smallNorm"`
	MinAdd        int64 `json:"minAdd"`
	MaxAdd        int64 `json:"maxAdd"`
	NotableWeight int64 `json:"notableWeight"`
}

type ctSkill struct {
	ID    string   `json:"id"`
	TV    int64    `json:"tv"`
	Types []int64  `json:"types"`
	Stats int      `json:"stats"`
	Min   [4]int64 `json:"min"`
	Max   [4]int64 `json:"max"`
	W     int64    `json:"w"`
	RMin  int64    `json:"rmin"`
	RMax  int64    `json:"rmax"`
}

type ctAddition struct {
	ID    string   `json:"id"`
	TV    int64    `json:"tv"`
	Types []int64  `json:"types"`
	Stats int      `json:"stats"`
	Min   [2]int64 `json:"min"`
	Max   [2]int64 `json:"max"`
	W     int64    `json:"w"`
}

type ctPassive struct {
	G int64 `json:"g"`
	T int   `json:"t"`
}

type ctDoc struct {
	Versions  []ctVersion  `json:"versions"`
	Skills    []ctSkill    `json:"skills"`
	Additions []ctAddition `json:"additions"`
	Passives  []ctPassive  `json:"passives"`
}

func ctListLen(v any) int {
	list, _ := v.([]any)
	return len(list)
}

func intList(v any) []int64 {
	list, _ := v.([]any)
	out := make([]int64, 0, len(list))
	for _, e := range list {
		out = append(out, e.(int64))
	}
	return out
}

func keyIndex(v any) (int64, error) {
	r, ok := v.(*Row)
	if !ok {
		return 0, fmt.Errorf("expected key row, got %T", v)
	}
	return int64(r.Index - 1), nil // dat row ids are 0-based; Row.Index is 1-based
}

// smallAttributeStat mirrors the client's small-attribute stat window
// (stat row ids 573..579, bitmask 0x49 — the same test the generation
// algorithm and the skills-tab classification use).
func smallAttributeStat(statIdx int64) bool {
	bit := (statIdx + 1) - 574
	return bit >= 0 && bit <= 6 && (0x49>>uint(bit))&1 != 0
}

// BuildConquerTables builds the document. nodeIDs is the modifiable-node
// graph-id set (data.NodeIDList — the shipped NodeIndexMapping).
func BuildConquerTables(x *Ctx, nodeIDs []int64) ([]byte, error) {
	doc := ctDoc{}

	tvDat := x.Dat("AlternateTreeVersions")
	tvDat.Rows(func(r *Row) bool {
		k := r.Index - 1
		if k < 1 || k > 11 {
			return true
		}
		v := ctVersion{
			K:             k,
			SmallAttr:     r.Get("SmallAttributeReplaced").(bool),
			SmallNorm:     r.Get("SmallNormalPassiveReplaced ").(bool),
			MinAdd:        r.Get("NotableAdditions").(Interval)[0],
			MaxAdd:        r.Get("NotableAdditions").(Interval)[1],
			NotableWeight: r.Get("NotableReplacementSpawnWeight ").(int64),
		}
		// The abyss generator (TimelessJewelData AlternateTreeVersion.cs)
		// overrides the dat's 0/0 addition counts to 1/1 for types 7-11;
		// the shipped bins agree with the override.
		if k >= 7 {
			v.MinAdd, v.MaxAdd = 1, 1
		}
		doc.Versions = append(doc.Versions, v)
		return true
	})

	var buildErr error
	x.Dat("AlternatePassiveSkills").Rows(func(r *Row) bool {
		tv, err := keyIndex(r.Get("AlternateTreeVersionsKey"))
		if err != nil {
			buildErr = fmt.Errorf("AlternatePassiveSkills row %d: %w", r.Index, err)
			return false
		}
		s := ctSkill{
			ID:    r.Get("Id").(string),
			TV:    tv,
			Types: intList(r.Get("PassiveType")),
			Stats: ctListLen(r.Get("StatsKeys")),
			W:     r.Get("SpawnWeight").(int64),
			RMin:  r.Get("Random").(Interval)[0],
			RMax:  r.Get("Random").(Interval)[1],
		}
		for i, col := range []string{"Stat1", "Stat2", "Stat3", "Stat4"} {
			iv := r.Get(col).(Interval)
			s.Min[i], s.Max[i] = iv[0], iv[1]
		}
		doc.Skills = append(doc.Skills, s)
		return true
	})
	if buildErr != nil {
		return nil, buildErr
	}

	x.Dat("AlternatePassiveAdditions").Rows(func(r *Row) bool {
		tv, err := keyIndex(r.Get("AlternateTreeVersionsKey"))
		if err != nil {
			buildErr = fmt.Errorf("AlternatePassiveAdditions row %d: %w", r.Index, err)
			return false
		}
		a := ctAddition{
			ID:    r.Get("Id").(string),
			TV:    tv,
			Types: intList(r.Get("PassiveType")),
			Stats: ctListLen(r.Get("StatsKeys")),
			// The spec names two columns "SpawnWeight" (as the reference
			// spec.lua does); the weight the shipped LUT bins agree with is
			// the FIRST (column 2), which name lookup (last-wins) misses.
			W: r.GetAt(2).(int64),
		}
		for i, col := range []string{"Stat1", "Stat2"} {
			iv := r.Get(col).(Interval)
			a.Min[i], a.Max[i] = iv[0], iv[1]
		}
		doc.Additions = append(doc.Additions, a)
		return true
	})
	if buildErr != nil {
		return nil, buildErr
	}

	// Passive classification for the modifiable-node set.
	byGraph := map[int64]*Row{}
	x.Dat("PassiveSkills").Rows(func(r *Row) bool {
		if _, seen := byGraph[r.Get("PassiveSkillNodeId").(int64)]; !seen {
			byGraph[r.Get("PassiveSkillNodeId").(int64)] = r
		}
		return true
	})
	sorted := append([]int64{}, nodeIDs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for _, gid := range sorted {
		r := byGraph[gid]
		if r == nil {
			return nil, fmt.Errorf("no PassiveSkills row for graph id %d", gid)
		}
		if r.Get("Keystone").(bool) || r.Get("JewelSocket").(bool) {
			return nil, fmt.Errorf("graph id %d is a keystone/socket — not modifiable", gid)
		}
		t := 2 // SmallNormal
		stats := r.Get("Stats").([]any)
		if r.Get("Notable").(bool) {
			t = 3
		} else if len(stats) == 1 {
			idx, err := keyIndex(stats[0])
			if err != nil {
				return nil, fmt.Errorf("graph id %d stat: %w", gid, err)
			}
			if smallAttributeStat(idx) {
				t = 1
			}
		}
		doc.Passives = append(doc.Passives, ctPassive{G: gid, T: t})
	}

	out, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
