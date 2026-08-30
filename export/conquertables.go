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

// treeVersionID is the row id of the AlternateTreeVersions ref.
func treeVersionID(r *Row) (int64, error) {
	tv := r.Ref("AlternateTreeVersionsKey")
	if tv == nil {
		return 0, fmt.Errorf("row %d: null AlternateTreeVersionsKey", r.ID)
	}
	return int64(tv.ID), nil
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

	tvDat, err := x.Dat("AlternateTreeVersions")
	if err != nil {
		return nil, err
	}
	altSkills, err := x.Dat("AlternatePassiveSkills")
	if err != nil {
		return nil, err
	}
	altAdditions, err := x.Dat("AlternatePassiveAdditions")
	if err != nil {
		return nil, err
	}
	passiveSkills, err := x.Dat("PassiveSkills")
	if err != nil {
		return nil, err
	}

	for r := range tvDat.Rows() {
		k := r.ID
		if k < 1 || k > 11 {
			continue
		}
		v := ctVersion{
			K:             k,
			SmallAttr:     r.Bool("SmallAttributeReplaced"),
			SmallNorm:     r.Bool("SmallNormalPassiveReplaced "),
			MinAdd:        r.Ivl("NotableAdditions")[0],
			MaxAdd:        r.Ivl("NotableAdditions")[1],
			NotableWeight: r.Int("NotableReplacementSpawnWeight "),
		}
		// The abyss generator (TimelessJewelData AlternateTreeVersion.cs)
		// overrides the dat's 0/0 addition counts to 1/1 for types 7-11;
		// the shipped bins agree with the override.
		if k >= 7 {
			v.MinAdd, v.MaxAdd = 1, 1
		}
		doc.Versions = append(doc.Versions, v)
	}

	for r := range altSkills.Rows() {
		tv, err := treeVersionID(r)
		if err != nil {
			return nil, fmt.Errorf("AlternatePassiveSkills %w", err)
		}
		s := ctSkill{
			ID:    r.Str("Id"),
			TV:    tv,
			Types: r.Ints("PassiveType"),
			Stats: len(r.Refs("StatsKeys")),
			W:     r.Int("SpawnWeight"),
			RMin:  r.Ivl("Random")[0],
			RMax:  r.Ivl("Random")[1],
		}
		for i, col := range []string{"Stat1", "Stat2", "Stat3", "Stat4"} {
			iv := r.Ivl(col)
			s.Min[i], s.Max[i] = iv[0], iv[1]
		}
		doc.Skills = append(doc.Skills, s)
	}

	for r := range altAdditions.Rows() {
		tv, err := treeVersionID(r)
		if err != nil {
			return nil, fmt.Errorf("AlternatePassiveAdditions %w", err)
		}
		a := ctAddition{
			ID:    r.Str("Id"),
			TV:    tv,
			Types: r.Ints("PassiveType"),
			Stats: len(r.Refs("StatsKeys")),
			// AlternatePassiveAdditions carries two weight columns, both
			// named "SpawnWeight" by the reference spec.lua; the shipped LUT
			// bins agree with the first, so the spec here keeps that name and
			// calls the second SpawnWeight2.
			W: r.Int("SpawnWeight"),
		}
		for i, col := range []string{"Stat1", "Stat2"} {
			iv := r.Ivl(col)
			a.Min[i], a.Max[i] = iv[0], iv[1]
		}
		doc.Additions = append(doc.Additions, a)
	}

	// Passive classification for the modifiable-node set.
	byGraph := map[int64]*Row{}
	for r := range passiveSkills.Rows() {
		if _, seen := byGraph[r.Int("PassiveSkillNodeId")]; !seen {
			byGraph[r.Int("PassiveSkillNodeId")] = r
		}
	}
	sorted := append([]int64{}, nodeIDs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	for _, gid := range sorted {
		r := byGraph[gid]
		if r == nil {
			return nil, fmt.Errorf("no PassiveSkills row for graph id %d", gid)
		}
		if r.Bool("Keystone") || r.Bool("JewelSocket") {
			return nil, fmt.Errorf("graph id %d is a keystone/socket — not modifiable", gid)
		}
		t := 2 // SmallNormal
		stats := r.Refs("Stats")
		if r.Bool("Notable") {
			t = 3
		} else if len(stats) == 1 {
			if smallAttributeStat(int64(stats[0].ID)) {
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
