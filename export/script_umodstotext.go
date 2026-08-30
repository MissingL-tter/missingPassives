// Port of .archive/src/Export/Scripts/uModsToText.lua: resolves the mod-id
// templates in Export/Uniques/<type>.lua into the unique item text database.
//
// uTextToMods.lua is the inverse tool; its itemTypes list is fully commented
// out in the reference, so it does nothing and has no port.

package export

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "uModsToText", Build: buildUModsToText})
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

func buildUModsToText(x *Ctx) (schema.Document, error) {
	if err := x.EnsureMods(); err != nil {
		return nil, err
	}
	// The Lua script inherits whatever stat file happens to be loaded; the
	// checked-in Data/Uniques were generated with mods.lua's state (tincture,
	// which includes the item stat descriptions), so pin that here instead of
	// depending on script order.
	x.LoadStatFile("tincture_stat_descriptions.txt")
	mods, err := x.Dat("Mods")
	if err != nil {
		return nil, err
	}

	doc := schema.Uniques{}
	for _, name := range uniqueItemTypes {
		tplDoc, err := readUniqueTemplate(name)
		if err != nil {
			return nil, err
		}
		var f schema.UniqueFile
		var cur []string
		emit := func(line string) { cur = append(cur, line) }

		statOrder := map[float64][]string{}
		modLines := 0
		nextOrder := float64(100000)
		// The per-item reset below clears the stat order but not this
		// countdown: an "Implicits: n" whose n outruns its item stays live
		// into the next one, as in the reference.
		var implicits *int

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

		for _, tplSec := range tplDoc.Sections {
			if len(tplSec.Items) == 0 {
				continue // a section is a run of items; no items, no section
			}
			f.Sections = append(f.Sections, schema.UniqueSection{})
			si := len(f.Sections) - 1
			for _, item := range tplSec.Items {
				// One item is one unique; the transform's per-item state
				// starts over at each item boundary.
				cur = []string{}
				statOrder = map[float64][]string{}
				modLines = 0
				nextOrder = 100000
				for _, line := range item {
					if implicits != nil {
						*implicits--
					}
					specMatch := reSpecLine.FindStringSubmatch(line)
					if specMatch == nil {
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
								if modRow := mods.RowByStr("Id", modName); modRow != nil {
									stats := map[string]*statVal{}
									for i := 1; i <= 6; i++ {
										if sr := modRow.Ref("Stat" + strconv.Itoa(i)); sr != nil && i-1 < len(values) {
											stats[sr.Str("Id")] = &statVal{min: values[i-1].min, max: values[i-1].max}
										}
									}
									lines, err := x.DescribeStats(stats)
									if err != nil {
										return nil, err
									}
									legacyMod = lines.Lines
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
				f.Sections[si].Items = append(f.Sections[si].Items, cur)
			}
		}
		doc[name] = f
	}
	return doc, nil
}
