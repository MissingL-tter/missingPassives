// Port of .archive/src/Export/Scripts/modScalability.lua.

package export

import "github.com/MissingL-tter/missingPassives/gamedata"

func init() {
	Scripts = append(Scripts, Script{Name: "modScalability", Build: buildModScalability})
}

func buildModScalability(x *Ctx) (any, error) {
	x.LoadStatFile("stat_descriptions.txt")

	sc := gamedata.ModScalability{}
	for line, vals := range x.DescribeScalability("stat_descriptions.txt") {
		list := make([]gamedata.Scalability, len(vals))
		for i, v := range vals {
			list[i] = gamedata.Scalability{IsScalable: v.isScalable, Formats: v.formats}
		}
		sc[line] = list
	}
	return sc, nil
}
