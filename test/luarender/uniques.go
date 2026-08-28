// Renders schema.Uniques as the Data/Uniques/<type>.lua files
// (Scripts/uModsToText.lua's outputs). The document carries only the item
// text; the Lua wrapper (section preambles, closers, trailing lines) is
// reconstructed from the archive's template file, whose passthrough text
// the generated file preserves verbatim.
//
// uTextToMods.lua is the inverse tool; its itemTypes list is fully commented
// out in the reference, so it does nothing and has no port.

package luarender

import (
	"fmt"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() { register("uModsToText", renderUniques) }

// wrapSection is one [[..]] run of the archive template with its wrapper.
type wrapSection struct {
	pre    []string
	items  int
	closer string
}

// splitWrapper parses the archive template's line stream for the wrapper
// text and per-section item counts.
func splitWrapper(raw string) (sections []wrapSection, post []string) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var pending []string
	inItem := false
	for _, line := range lines {
		switch {
		case !inItem && line == "[[":
			sections = append(sections, wrapSection{pre: pending, items: 1})
			pending = nil
			inItem = true
		case inItem && line == "]],[[":
			sections[len(sections)-1].items++
		case inItem && (line == "]]," || line == "]]" || line == "]],}"):
			sections[len(sections)-1].closer = line
			inItem = false
		case !inItem:
			pending = append(pending, line)
		}
	}
	return sections, pending
}

func renderUniques(d schema.Uniques, tpl Templates) (map[string]string, error) {
	files := map[string]string{}
	for name, f := range d {
		raw, err := tpl.Read("Export/Uniques/" + name + ".lua")
		if err != nil {
			return nil, err
		}
		wrap, post := splitWrapper(raw)
		if len(wrap) != len(f.Sections) {
			return nil, fmt.Errorf("uniques %s: %d template sections vs %d document sections", name, len(wrap), len(f.Sections))
		}
		var b B
		for si, sec := range f.Sections {
			if wrap[si].items != len(sec.Items) {
				return nil, fmt.Errorf("uniques %s section %d: %d template items vs %d document items", name, si, wrap[si].items, len(sec.Items))
			}
			for _, line := range wrap[si].pre {
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
			b.W(wrap[si].closer, "\n")
		}
		for _, line := range post {
			b.W(line, "\n")
		}
		files["Data/Uniques/"+name+".lua"] = b.String()
	}
	return files, nil
}
