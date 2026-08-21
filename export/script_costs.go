// Port of .archive/src/Export/Scripts/costs.lua.

package export

import "github.com/MissingL-tter/missingPassives/gamedata"

func init() {
	Scripts = append(Scripts, Script{Name: "costs", Build: buildCosts})
}

func buildCosts(x *Ctx) (any, error) {
	var costs gamedata.Costs
	x.Dat("CostTypes").Rows(func(c *Row) bool {
		ct := gamedata.CostType{
			Resource:       luaStr(c.Get("Resource")),
			ResourceString: luaStr(c.Get("ResourceString")),
			Divisor:        c.Get("Divisor").(int64),
		}
		if stat, ok := c.Get("Stat").(*Row); ok {
			id := luaStr(stat.Get("Id"))
			ct.Stat = &id
		}
		costs = append(costs, ct)
		return true
	})
	// special case for soul cost
	soulStat := " "
	costs = append(costs, gamedata.CostType{
		Resource:       "Soul",
		Stat:           &soulStat,
		ResourceString: "{0} Souls Per Use",
		Divisor:        1,
	})
	return costs, nil
}
