// Port of .archive/src/Export/Scripts/uModsToText.lua: resolves the mod-id
// templates in Export/Uniques/<type>.lua into the unique item text database.
//
// uTextToMods.lua is the inverse tool; its itemTypes list is fully commented
// out in the reference, so it does nothing and has no port.

package export

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "uModsToText", Build: buildUModsToText})
}

// splitUniqueFile parses the transform's output lines into sections of item
// blobs; renderUniques inverts it exactly.
func splitUniqueFile(lines []string) schema.UniqueFile {
	var f schema.UniqueFile
	var pending []string
	var cur []string
	inItem := false
	sec := -1
	for _, line := range lines {
		switch {
		case !inItem && line == "[[":
			f.Sections = append(f.Sections, schema.UniqueSection{Pre: pending})
			pending = nil
			sec = len(f.Sections) - 1
			cur = []string{}
			inItem = true
		case inItem && line == "]],[[":
			f.Sections[sec].Items = append(f.Sections[sec].Items, cur)
			cur = []string{}
		case inItem && (line == "]]," || line == "]]" || line == "]],}"):
			f.Sections[sec].Items = append(f.Sections[sec].Items, cur)
			f.Sections[sec].Closer = line
			cur = nil
			inItem = false
		case inItem:
			cur = append(cur, line)
		default:
			pending = append(pending, line)
		}
	}
	if inItem {
		panic("uModsToText: unterminated item block")
	}
	f.Post = pending
	return f
}

var uniqueItemTypes = []string{
	"axe", "bow", "claw", "dagger", "fishing", "mace", "staff", "sword", "wand",
	"helmet", "body", "gloves", "boots", "shield", "quiver",
	"amulet", "ring", "belt", "jewel", "flask", "tincture",
}

var catalystTags = map[string]bool{
	"elemental_damage": true, "caster": true, "attack": true, "defences": true,
	"resource": true, "resistance": true, "attribute": true, "physical_damage": true,
	"chaos_damage": true, "speed": true, "critical": true,
}

var (
	reSpecLine     = regexp.MustCompile(`^([A-Za-z ]+): (.+)$`)
	reVersionTag   = regexp.MustCompile(`(\{version:[0-9,]+\})`)
	reVariantTag   = regexp.MustCompile(`(\{variant:[0-9,]+\})`)
	reGroupTag     = regexp.MustCompile(`(\{group:[0-9,]+\})`)
	reFracturedTag = regexp.MustCompile(`(\{fractured\})`)
	reAnyTag       = regexp.MustCompile(`\{.*?\}`)
	reModIdBrack   = regexp.MustCompile(`^([0-9A-Za-z_ ]+)\[`)
	reModIdOnly    = regexp.MustCompile(`^([0-9A-Za-z_ ]+)$`)
	reBrackRange   = regexp.MustCompile(`\[[^\[\]]*\]`)
	reRangePair    = regexp.MustCompile(`\[([0-9.\-]+),([0-9.\-]+)\]`)
)

func buildUModsToText(x *Ctx) (any, error) {
	if err := x.EnsureMods(); err != nil {
		return nil, err
	}
	// The Lua script inherits whatever stat file happens to be loaded; the
	// checked-in Data/Uniques were generated with mods.lua's state (tincture,
	// which includes the item stat descriptions), so pin that here instead of
	// depending on script order.
	x.LoadStatFile("tincture_stat_descriptions.txt")
	mods := x.Dat("Mods")

	doc := schema.Uniques{}
	for _, name := range uniqueItemTypes {
		raw, err := os.ReadFile(filepath.Join(x.TplDir, "Export", "Uniques", name+".lua"))
		if err != nil {
			return nil, err
		}
		var outLines []string
		emit := func(line string) { outLines = append(outLines, line) }

		statOrder := map[float64][]string{}
		modLines := 0
		var implicits *int
		nextOrder := float64(100000)

		writeStatOrder := func() {
			orders := make([]float64, 0, len(statOrder))
			for o := range statOrder {
				orders = append(orders, o)
			}
			sort.Float64s(orders)
			for _, o := range orders {
				for _, line := range statOrder[o] {
					emit(line)
				}
			}
		}

		// io.lines: split on newlines, no trailing empty line.
		lines := strings.Split(string(raw), "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for _, line := range lines {
			line = strings.TrimSuffix(line, "\r")
			if implicits != nil {
				*implicits--
			}
			specMatch := reSpecLine.FindStringSubmatch(line)
			if strings.Contains(line, "]],") { // start new unique
				writeStatOrder()
				emit(line)
				statOrder = map[float64][]string{}
				modLines = 0
				nextOrder = 100000
			} else if specMatch == nil {
				prefix := ""
				versionString := reVersionTag.FindString(line)
				variantString := reVariantTag.FindString(line)
				groupString := reGroupTag.FindString(line)
				fractured := reFracturedTag.FindString(line)
				cleanLine := reAnyTag.ReplaceAllString(line, "")
				var modName string
				if m := reModIdBrack.FindStringSubmatch(cleanLine); m != nil {
					modName = m[1]
				} else if m := reModIdOnly.FindStringSubmatch(cleanLine); m != nil {
					modName = m[1]
				}
				legacy := ""
				if modName != "" {
					legacy = cleanLine[len(modName):]
				}
				// Legacy ranges must contain actual brackets
				if legacy != "" && !strings.Contains(legacy, "[") {
					legacy = ""
					modName = ""
				}
				var mod *modEntry
				if modName != "" {
					mod = x.modItemExclusive[modName]
				}
				if mod != nil || (modName != "" && legacy != "") {
					modLines++
					prefix += versionString + variantString + groupString
					var tags []string
					if mod != nil && (name == "amulet" || name == "ring" || name == "belt") {
						for _, tag := range mod.tags {
							if catalystTags[tag] {
								tags = append(tags, tag)
							}
						}
					}
					if len(tags) > 0 {
						prefix += "{tags:" + strings.Join(tags, ",") + "}"
					}
					prefix += fractured
					var legacyMod []string
					legacyFound := false // the Lua's legacyMod is a (possibly empty) table when the mod row exists
					if legacy != "" {
						type mm struct{ min, max float64 }
						var values []mm
						for _, rng := range reBrackRange.FindAllString(legacy, -1) {
							if m := reRangePair.FindStringSubmatch(rng); m != nil {
								mn, _ := strconv.ParseFloat(m[1], 64)
								mx, _ := strconv.ParseFloat(m[2], 64)
								values = append(values, mm{mn, mx})
							}
						}
						if modRow := mods.GetRow("Id", modName); modRow != nil {
							stats := map[string]*statVal{}
							for i := 1; i <= 6; i++ {
								if sr, ok := modRow.Get("Stat" + luaStr(i)).(*Row); ok && i-1 < len(values) {
									stats[luaStr(sr.Get("Id"))] = &statVal{min: values[i-1].min, max: values[i-1].max}
								}
							}
							legacyMod = x.DescribeStats(stats).Lines
							legacyFound = true
						}
					}
					var modText []string
					if legacyFound {
						modText = legacyMod
					} else if mod != nil {
						modText = mod.lines
					}
					var order float64
					for i, l := range modText {
						if i == 0 {
							if mod != nil && len(mod.orders) > 0 {
								order = mod.orders[0]
							} else {
								order = nextOrder
							}
						}
						nextOrder++
						statOrder[order] = append(statOrder[order], prefix+l)
					}
				} else {
					if modLines > 0 || implicits != nil {
						// Unresolved text lines get a sequential order to
						// preserve position among mods
						statOrder[nextOrder] = append(statOrder[nextOrder], line)
						nextOrder++
					} else {
						emit(line)
					}
				}
			} else { // spec line
				if specMatch[1] == "Implicits" {
					n, _ := strconv.Atoi(specMatch[2])
					implicits = &n
				} else {
					emit(line)
				}
			}
			if implicits != nil && *implicits == 0 {
				count := 0
				for _, l := range statOrder {
					count += len(l)
				}
				emit("Implicits: " + strconv.Itoa(count))
				writeStatOrder()
				implicits = nil
				statOrder = map[float64][]string{}
				modLines = 0
			}
		}
		writeStatOrder()
		doc[name] = splitUniqueFile(outLines)
	}
	return doc, nil
}
