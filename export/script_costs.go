// Port of .archive/src/Export/Scripts/costs.lua.

package export

import "github.com/MissingL-tter/missingPassives/data/schema"

func init() {
	Scripts = append(Scripts, Script{Name: "costs", Build: buildCosts})
}

func buildCosts(x *Ctx) (schema.Document, error) {
	costTypes, err := x.Dat("CostTypes")
	if err != nil {
		return nil, err
	}
	var costs schema.Costs
	for c := range costTypes.Rows() {
		ct := schema.CostType{
			Resource:       c.Str("Resource"),
			ResourceString: c.Str("ResourceString"),
			Divisor:        c.Int("Divisor"),
		}
		if stat := c.Ref("Stat"); stat != nil {
			id := stat.Str("Id")
			ct.Stat = &id
		}
		costs = append(costs, ct)
	}
	// special case for soul cost
	soulStat := " "
	costs = append(costs, schema.CostType{
		Resource:       "Soul",
		Stat:           &soulStat,
		ResourceString: "{0} Souls Per Use",
		Divisor:        1,
	})
	return costs, nil
}
