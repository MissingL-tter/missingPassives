// Port of .archive/src/Export/Scripts/modScalability.lua.

package export

import (
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "modScalability", Build: buildModScalability})
}

func buildModScalability(x *Ctx) (schema.Document, error) {
	x.LoadStatFile("stat_descriptions.txt")

	scal, err := x.DescribeScalability("stat_descriptions.txt")
	if err != nil {
		return nil, err
	}
	sc := schema.ModScalability{}
	for line, vals := range scal {
		list := make([]schema.Scalability, len(vals))
		for i, v := range vals {
			list[i] = schema.Scalability{IsScalable: v.isScalable, Formats: v.formats}
		}
		// The stat description text spells line breaks as \n.
		sc[strings.ReplaceAll(line, `\n`, "\n")] = list
	}
	return sc, nil
}
