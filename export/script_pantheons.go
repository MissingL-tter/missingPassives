// Port of .archive/src/Export/Scripts/pantheons.lua.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "pantheons", Build: buildPantheons})
}

func buildPantheons(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("stat_descriptions.txt")

	layout, err := x.Dat("PantheonPanelLayout")
	if err != nil {
		return nil, err
	}
	var ps schema.Pantheons
	for _, p := range layout.GetRowList("IsDisabled", false) {
		pan := schema.Pantheon{
			Id:         p.Str("Id"),
			IsMajorGod: p.Bool("IsMajorGod"),
		}
		type god struct {
			name     string
			statKeys []*Row
			values   []int64
		}
		gods := []god{
			{p.Str("GodName1"), p.Refs("Effect1StatsKey"), p.Ints("Effect1Values")},
			{p.Str("GodName2"), p.Refs("Effect2StatsKey"), p.Ints("Effect2Values")},
			{p.Str("GodName3"), p.Refs("Effect3StatsKey"), p.Ints("Effect3Values")},
			{p.Str("GodName4"), p.Refs("Effect4StatsKey"), p.Ints("Effect4Values")},
		}
		for gi, g := range gods {
			if len(g.statKeys) == 0 {
				continue
			}
			pg := schema.PantheonGod{Index: gi + 1, Name: g.name}
			for si, keyRow := range g.statKeys {
				value := g.values[si]
				id := keyRow.Str("Id")
				stats := map[string]*statVal{
					id: {min: float64(value), max: float64(value)},
				}
				lines, err := x.DescribeStats(stats)
				if err != nil {
					return nil, err
				}
				pg.Mods = append(pg.Mods, schema.PantheonMod{
					StatId: id,
					Line:   strings.Join(lines.Lines, " "),
					Value:  value,
				})
			}
			pan.Gods = append(pan.Gods, pg)
		}
		ps = append(ps, pan)
	}
	return ps, nil
}
