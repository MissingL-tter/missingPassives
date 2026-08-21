// Port of .archive/src/Export/Scripts/crucible.lua.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() {
	Scripts = append(Scripts, Script{Name: "crucible", Build: buildCrucible})
}

// intCells converts a numeric list cell to int64s.
func intCells(list []any) []int64 {
	out := make([]int64, len(list))
	for i, v := range list {
		out[i] = v.(int64)
	}
	return out
}

// rowIds collects the Id column of a row-list cell.
func rowIds(list []any) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, luaStr(v.(*Row).Get("Id")))
	}
	return out
}

func buildCrucible(x *Ctx) (any, error) {
	x.LoadStatFile("stat_descriptions.txt")

	var cn gamedata.CrucibleNodes
	x.Dat("WeaponPassiveSkills").Rows(func(crucible *Row) bool {
		mod := crucible.Get("Mod").(*Row)
		modId := luaStr(mod.Get("Id"))
		if strings.HasSuffix(modId, "HardMode") {
			return true
		}
		stats := x.DescribeMod(mod)
		if len(stats.Orders) == 0 {
			// The Lua prints "Mod '...' has no stats".
			return true
		}
		n := gamedata.CrucibleNode{ModId: modId}
		switch mod.Get("GenerationType").(int64) {
		case 31:
			n.Type = "Spawn"
		case 32:
			n.Type = "MergeOnly"
		}
		n.Tier = crucible.Get("ModTier").(int64)
		n.Lines = stats.Lines
		n.StatOrders = stats.Orders
		n.Level = mod.Get("Level").(int64)
		n.Group = luaStr(mod.Get("Type").(*Row).Get("Id"))
		n.NodeType = luaStr(crucible.Get("Type").(*Row).Get("Id"))
		n.NodeLocation = intCells(crucible.Get("NodeSpawnLocation").([]any))
		n.WeightKey = rowIds(mod.Get("SpawnTags").([]any))
		n.WeightVal = intCells(mod.Get("SpawnWeights").([]any))
		genTags := mod.Get("GenerationWeightTags").([]any)
		if len(genTags) > 0 {
			n.WeightMultiplierKey = rowIds(genTags)
			n.WeightMultiplierVal = intCells(mod.Get("GenerationWeightValues").([]any))
			n.Tags = rowIds(mod.Get("Tags").([]any))
		}
		n.ModTags = stats.ModTags
		cn = append(cn, n)
		return true
	})
	return cn, nil
}
