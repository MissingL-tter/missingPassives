// Port of .archive/src/Export/Scripts/flavourText.lua.

package export

import (
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

func init() {
	Scripts = append(Scripts, Script{Name: "flavourText", Build: buildFlavourText})
}

var (
	reTrailUnderscore = regexp.MustCompile(`_+$`)
	reBraces          = regexp.MustCompile(`\{(.*?)\}`)
	reAngles          = regexp.MustCompile(`<<(.*?)>>`)
)

func normalizeId(id string) string {
	return reTrailUnderscore.ReplaceAllString(id, "")
}

func luaTrim(s string) string {
	return strings.Trim(s, " \t\n\v\f\r")
}

func cleanAndSplit(str string) []string {
	str = strings.ReplaceAll(str, "\r\n", "\n")
	str = strings.ReplaceAll(str, "<default>", "\n^8")

	var lines []string
	for _, line := range strings.Split(str, "\n") {
		if line == "" {
			continue
		}
		line = luaTrim(line)
		if line == "" {
			continue
		}
		line = reBraces.ReplaceAllString(line, "$1")
		line = reAngles.ReplaceAllString(line, "")
		line = luaTrim(line)
		line = strings.ReplaceAll(line, "\"", "\\\"")
		if strings.HasPrefix(line, "^8") && (len(lines) == 0 || lines[len(lines)-1] != "") {
			lines = append(lines, "")
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

type forcedName struct{ id, name string }

var forcedNameList = []forcedName{
	{"UniqueOneHandAxe5", "Jack, the Axe"},
	{"UniqueBootsDIY", "Doryani's Delusion"},
	{"UniqueGlovesStrDex9", "Tombfist"},
	{"UniqueAmulet45", "Impresence"},
	{"UniqueAmuletVictor1", "Talisman of the Victor"},
	{"UniqueStaff8", "Agnerod South"},
	{"UniqueStaff8", "Agnerod North"},
	{"UniqueStaff8", "Agnerod West"},
	{"Ring11", "Gifts from Above"},
	{"Ring12", "Death Rush"},
	{"Ring13", "Shavronne's Revelation"},
	{"Belt5", "Auxium"},
	{"Amulet13", "Daresso's Salute"},
	{"Amulet14", "Voll's Devotion"},
	{"Amulet15", "Victario's Acuity"},
	{"UniqueAmulet27", "Star of Wraeclast"},
	{"UniqueAmulet29x", "Replica Winterheart"},
	{"UniqueJewel82x", "Replica Primordial Might"},
	{"UniqueBootsDex10", "Abberath's Hooves"},
	{"UniqueRing54", "Precursor's Emblem"},
	{"UniqueHelmetStrInt11", "Lightpoacher"},
	{"UniqueBootsDexInt6", "Bubonic Trail"},
	{"UniqueBodyDexInt9", "Shroud of the Lightless"},
	{"UniqueGlovesStrInt6", "Volkuur's Guidance"},
	{"UniqueBootsAtlas1", "Beacon of Madness"},
	{"UniqueBootsAtlas1", "Demigod's Eye"},
	{"UniqueShieldDemigods", "Demigod's Beacon"},
	{"UniqueBootsDemigods1", "Demigod's Stride"},
	{"UniqueBeltDemigods1", "Demigod's Bounty"},
	{"UniqueBodyDemigods", "Demigod's Dominance"},
	{"UniqueHelmetDemigods1", "Demigod's Immortality"},
	{"UniqueQuiver1", "Blackgleam"},
	{"FatedUnique8", "The Signal Fire"},
	{"UniqueBootsStr1", "Windscream"},
	{"FatedUnique35", "Windshriek"},
	{"UniqueBodyDex7", "Briskwrap"},
	{"FatedUnique31", "Wildwrap"},
	{"UniqueShieldInt2", "Matua Tupuna"},
	{"FatedUnique61", "Whakatutuki o Matua"},
	{"UniqueBow8", "Storm Cloud"},
	{"FatedUnique21", "The Tempest"},
	{"UniqueBelt2", "The Magnate"},
	{"FatedUnique46", "The Tactician"},
	{"FatedUnique47", "The Nomad"},
	{"UniqueStaff14", "The Stormheart"},
	{"FatedUnique29", "The Stormwall"},
	{"UniqueTwoHandAxe3", "Limbsplit"},
	{"FatedUnique18", "The Cauteriser"},
	{"UniqueTwoHandSword4", "Queen's Decree"},
	{"FatedUnique25", "Queen's Escape"},
	{"UniqueHelmetDexInt1", "Malachai's Simula"},
	{"FatedUnique43", "Malachai's Awakening"},
	{"UniqueRing2", "Kaom's Sign"},
	{"FatedUnique1", "Kaom's Way"},
	{"UniqueQuiver6", "Hyrri's Bite"},
	{"FatedUnique49", "Hyrri's Demise"},
	{"UniqueGlovesDex1", "Hrimsorrow"},
	{"FatedUnique10", "Hrimburn"},
	{"UniqueTwoHandMace2", "Geofri's Baptism"},
	{"FatedUnique52", "Geofri's Devotion"},
	{"UniqueDexHelmet2", "Heatshiver"},
	{"FatedUnique36", "Frostferno"},
	{"UniqueBootsStrDex3", "Dusktoe"},
	{"FatedUnique26", "Duskblight"},
	{"UniqueOneHandSword1", "Redbeak"},
	{"FatedUnique54", "Dreadbeak"},
	{"UniqueBow11", "Doomfletch"},
	{"FatedUnique19", "Doomfletch's Prism"},
	{"UniqueGlovesInt2", "Doedre's Tenure"},
	{"FatedUnique28", "Doedre's Malevolence"},
	{"UniqueBow3", "Death's Harp"},
	{"FatedUnique5", "Death's Opus"},
	{"UniqueTwoHandMace5", "Chober Chaber"},
	{"FatedUnique58", "Chaber Cairn"},
	{"UniqueOneHandMace4", "Cameria's Maul"},
	{"FatedUnique53", "Cameria's Avarice"},
	{"UniqueIntHelmet2", "Asenath's Mark"},
	{"FatedUnique41", "Asenath's Chant"},
	{"UniqueWand8", "Reverberation Rod"},
	{"FatedUnique23", "Amplification Rod"},
	{"UniqueOneHandAxe7", "Dreadarc"},
	{"FatedUnique50", "Dreadsurge"},
	{"UniqueJewel16", "Apparitions"},
	{"UniqueDescentOneHandSword1", "Blood of Summer"},
	{"UniqueDescentOneHandAxe1", "Rust of Winter"},
	{"UniqueDescentOneHandMace1", "Ashes of the Sun"},
	{"UniqueDescentWand1", "Splinter of the Moon"},
	{"UniqueDescentTwoHandSword1", "Thunder of the Dawn"},
	{"UniqueDescentStaff1", "Vestige of Divinity"},
	{"UniqueDescentDagger1", "Fragment of Eternity"},
	{"UniqueDescentBow1", "Relic of the Cycle"},
	{"UniqueDescentClaw1", "Scar of Fate"},
	{"UniqueDescentHelmet1", "Tears of Entropy"},
	{"UniqueDescentShield1", "Remnant of Empires"},
	{"UniqueDescentBelt1", "Chains of Time"},
	{"UniqueDescentQuiver1", "Slivers of Providence"},
	{"FatedUnique57", "The Iron Fortress"},
	{"UniqueBodyStr9", "Iron Heart"},
	{"UniqueOneHandSword10", "The Goddess Unleashed"},
	{"UniqueRapier1", "The Goddess Bound"},
	{"FatedUnique48", "Winterweave"},
	{"UniqueRing28", "Bloodboil"},
	{"FatedUnique27", "Atziri's Reflection"},
	{"UniqueShieldDex3", "Atziri's Mirror"},
	{"Ring11x", "Replica Gifts from Above"},
}

func buildFlavourText(x *Ctx) (any, error) {
	uniqueNameLookup := map[string]string{}
	x.Dat("UniqueStashLayout").Rows(func(row *Row) bool {
		name := luaStr(row.Get("WordsKey").(*Row).Get("Text2"))
		id := normalizeId(luaStr(row.Get("ItemVisualIdentity").(*Row).Get("Id")))
		if strings.Contains(id, "Map") || strings.Contains(id, "AlternateArt") ||
			strings.Contains(id, "AtlasUpgrade") || strings.Contains(id, "HeistQuest") {
			return true
		}
		uniqueNameLookup[id] = name
		return true
	})

	flavourTextById := map[string][]string{}
	x.Dat("FlavourText").Rows(func(c *Row) bool {
		flavourTextById[normalizeId(luaStr(c.Get("Id")))] = cleanAndSplit(luaStr(c.Get("Text")))
		return true
	})

	var fts schema.FlavourTexts
	x.flavourEntries = nil
	addEntry := func(id, name string, textLines []string) {
		x.flavourEntries = append(x.flavourEntries, flavourEntry{id: id, name: name})
		fts = append(fts, schema.FlavourText{Id: id, Name: name, Text: textLines})
	}

	for _, entry := range forcedNameList {
		if textLines, ok := flavourTextById[entry.id]; ok {
			addEntry(entry.id, entry.name, textLines)
		}
	}

	ids := make([]string, 0, len(uniqueNameLookup))
	for id := range uniqueNameLookup {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if lines, ok := flavourTextById[id]; ok {
			addEntry(id, uniqueNameLookup[id], lines)
		}
	}
	return fts, nil
}
