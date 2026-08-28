// Port of .archive/src/Export/Scripts/pantheons.lua.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "pantheons", Build: buildPantheons})
}

func buildPantheons(x *Ctx) (any, error) {
	x.LoadStatFile("stat_descriptions.txt")

	var ps schema.Pantheons
	for _, p := range x.Dat("PantheonPanelLayout").GetRowList("IsDisabled", false) {
		pan := schema.Pantheon{
			Id:         luaStr(p.Get("Id")),
			IsMajorGod: p.Get("IsMajorGod").(bool),
		}
		type god struct {
			name     string
			statKeys []any
			values   []any
		}
		gods := []god{
			{luaStr(p.Get("GodName1")), p.Get("Effect1StatsKey").([]any), p.Get("Effect1Values").([]any)},
			{luaStr(p.Get("GodName2")), p.Get("Effect2StatsKey").([]any), p.Get("Effect2Values").([]any)},
			{luaStr(p.Get("GodName3")), p.Get("Effect3StatsKey").([]any), p.Get("Effect3Values").([]any)},
			{luaStr(p.Get("GodName4")), p.Get("Effect4StatsKey").([]any), p.Get("Effect4Values").([]any)},
		}
		for gi, g := range gods {
			if len(g.statKeys) == 0 {
				continue
			}
			pg := schema.PantheonGod{Index: gi + 1, Name: g.name}
			for si, key := range g.statKeys {
				keyRow := key.(*Row)
				value := g.values[si].(int64)
				id := luaStr(keyRow.Get("Id"))
				stats := map[string]*statVal{
					id: {min: float64(value), max: float64(value)},
				}
				pg.Mods = append(pg.Mods, schema.PantheonMod{
					StatId: id,
					Line:   strings.Join(x.DescribeStats(stats).Lines, " "),
					Value:  value,
				})
			}
			pan.Gods = append(pan.Gods, pg)
		}
		ps = append(ps, pan)
	}
	return ps, nil
}
