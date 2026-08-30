// Port of .archive/src/Export/Scripts/crucible.lua.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "crucible", Build: buildCrucible})
}

// rowIds collects the Id column of a row list.
func rowIds(rows []*Row) []string {
	var out []string // nil for an empty cell: the artifact carries null, not []
	for _, r := range rows {
		out = append(out, r.Str("Id"))
	}
	return out
}

func buildCrucible(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("stat_descriptions.txt")

	weaponPassives, err := x.Dat("WeaponPassiveSkills")
	if err != nil {
		return nil, err
	}
	var cn schema.CrucibleNodes
	for crucible := range weaponPassives.Rows() {
		mod := crucible.Ref("Mod")
		modId := mod.Str("Id")
		if strings.HasSuffix(modId, "HardMode") {
			continue
		}
		stats, err := x.DescribeMod(mod)
		if err != nil {
			return nil, err
		}
		if len(stats.Orders) == 0 {
			// The Lua prints "Mod '...' has no stats".
			continue
		}
		n := schema.CrucibleNode{ModId: modId}
		switch mod.Int("GenerationType") {
		case 31:
			n.Type = "Spawn"
		case 32:
			n.Type = "MergeOnly"
		}
		n.Tier = crucible.Int("ModTier")
		n.Lines = stats.Lines
		n.StatOrders = stats.Orders
		n.Level = mod.Int("Level")
		n.Group = mod.Ref("Type").Str("Id")
		n.NodeType = crucible.Ref("Type").Str("Id")
		n.NodeLocation = crucible.Ints("NodeSpawnLocation")
		n.WeightKey = rowIds(mod.Refs("SpawnTags"))
		n.WeightVal = mod.Ints("SpawnWeights")
		genTags := mod.Refs("GenerationWeightTags")
		if len(genTags) > 0 {
			n.WeightMultiplierKey = rowIds(genTags)
			n.WeightMultiplierVal = mod.Ints("GenerationWeightValues")
			n.Tags = rowIds(mod.Refs("Tags"))
		}
		n.ModTags = stats.ModTags
		cn = append(cn, n)
	}
	return cn, nil
}
