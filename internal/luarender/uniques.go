// Renders schema.Uniques as the Data/Uniques/<type>.lua files
// (Scripts/uModsToText.lua's outputs).
//
// uTextToMods.lua is the inverse tool; its itemTypes list is fully commented
// out in the reference, so it does nothing and has no port.

package luarender

import "github.com/MissingL-tter/missingPassives/data/schema"

func init() { register("uModsToText", renderUniques) }

func renderUniques(d schema.Uniques, _ Templates) (map[string]string, error) {
	files := map[string]string{}
	for name, f := range d {
		var b B
		for _, sec := range f.Sections {
			for _, line := range sec.Pre {
				b.W(line, "\n")
			}
			b.W("[[\n")
			for i, item := range sec.Items {
				if i > 0 {
					b.W("]],[[\n")
				}
				for _, line := range item {
					b.W(line, "\n")
				}
			}
			b.W(sec.Closer, "\n")
		}
		for _, line := range f.Post {
			b.W(line, "\n")
		}
		files["Data/Uniques/"+name+".lua"] = b.String()
	}
	return files, nil
}
