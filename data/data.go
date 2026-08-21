// Package data assembles the game data the application consumes at runtime:
// the Go port of .archive/src/Modules/Data.lua. Generated data arrives as
// gamedata documents (the export pipeline's output); the tables Data.lua
// defines inline are ported here. Verified against the archive by
// test/gamedata_test.go, subtree by subtree.
package data

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

// Sources carries the gamedata documents Load consumes. It grows as the
// port covers more of Data.lua.
type Sources struct {
	Misc           gamedata.MiscData
	Costs          gamedata.Costs
	Boss           gamedata.BossData
	ModScalability gamedata.ModScalability
	Mods           gamedata.ModsData
	Essences       gamedata.Essences
	Pantheons      gamedata.Pantheons
	Crucible       gamedata.CrucibleNodes
	Masters        gamedata.MasterCrafts
	FlavourText    gamedata.FlavourTexts
	Enchants       gamedata.Enchants
	Cluster        gamedata.ClusterJewels
	Bases          gamedata.BasesData
	Uniques        gamedata.Uniques
	MinionsDoc     gamedata.Minions
	Skills         gamedata.SkillsData
	// StatMapCopies lists, per skill, the statMap keys the booted archive's
	// lazy copies materialised (a replay fixture from the archive dump).
	StatMapCopies map[string][]string
	// FoulbornMapJSONC is Data/ModFoulbornMap.jsonc's content.
	FoulbornMapJSONC []byte
}

// Data mirrors the Lua `data` table (the logic-bearing parts).
type Data struct {
	// Data/Misc.lua tables.
	MonsterEvasionTable             []float64
	MonsterAccuracyTable            []float64
	MonsterLifeTable                []float64
	MonsterLifeTable2               []float64
	MonsterLifeTable3               []float64
	MonsterAllyLifeTable            []float64
	MonsterDamageTable              []float64
	MonsterAllyDamageTable          []float64
	MonsterArmourTable              []float64
	MonsterAilmentThresholdTable    []float64
	MonsterPhysConversionMultiTable []float64
	GameConstants                   map[string]float64
	CharacterConstants              map[string]float64
	MonsterConstants                map[string]float64
	TotemLifeMult                   map[int64]float64
	MonsterVarietyLifeMult          map[string]float64
	MapLevelLifeMult                map[int64]float64
	MapLevelBossLifeMult            map[int64]float64
	MapLevelBossAilmentMult         map[int64]float64
	GoldRespecPrices                []int64

	Misc                      Misc
	PowerStatList             []PowerStat
	SkillColorMap             []string
	MonsterExperienceLevelMap map[int]float64
	CursePriority             map[string]int
	Keystones                 []string

	AilmentTypeList             []string
	ElementalAilmentTypeList    []string
	NonDamagingAilmentTypeList  []string
	NonElementalAilmentTypeList []string
	NonDamagingAilment          map[string]Ailment

	DefaultHighPrecision int
	HighPrecisionMods    map[string]map[string]int
	ModScalability       map[string][]Scalability

	WeaponTypeInfo    map[string]WeaponTypeInfo
	UnarmedWeaponData map[int]UnarmedWeapon

	JewelRadii map[string][]*JewelRadius
	// JewelRadius is the selected version's table. Data.lua nils it at its
	// own load (it assigns setJewelRadiiGlobally's nil return), but the tree
	// load calls setJewelRadiiGlobally again during boot, so the running
	// application always has it set; the port sets it directly.
	JewelRadius    []*JewelRadius
	MaxJewelRadius float64

	EnchantmentSource []EnchantmentSource

	TimelessJewelTypes     map[int]string
	TimelessJewelSeedMin   map[int]float64
	TimelessJewelSeedMax   map[int]float64
	TimelessJewelAdditions int
	// One-time-converted timeless jewel tables (typed by the
	// timeless-jewel-data module later).
	NodeIDList            map[string]any
	AbyssNotableNames     map[string]any
	TimelessJewelTradeIDs map[string]any
	TimelessJewelLUTs     map[string]any

	// MapMods is Data/ModMap.lua's table; the apply closures are Unported
	// markers until the config module lands.
	MapMods map[string]any

	ItemTagSpecial                 map[string]map[string][]string
	ItemTagSpecialExclusionPattern map[string]map[string][]string
	CasterTagCrucibleUniques       map[string]bool
	MinionTagCrucibleUniques       map[string]bool

	Costs []Cost

	ItemMods       map[string]map[string]ItemModData
	bbdPool        map[string]ItemModData
	VeiledMods     map[string]ItemModData
	BeastCraft     map[string]ItemModData
	NecropolisMods map[string]ItemModData
	UniqueMods     map[string][]UniqueModEntry

	Essences     map[string]Essence
	Pantheons    map[string]Pantheon
	Crucible     map[string]CrucibleNode
	MasterMods   []MasterCraft
	FlavourText  []FlavourText
	Enchantments map[string]map[string][]string
	// HelmetEnchants is data.enchantments["Helmet"] (skill -> source -> mods).
	HelmetEnchants map[string]map[string][]string

	RareLikeUniques map[string]RareLikeUnique

	ItemBases        map[string]*ItemBase
	ItemBaseLists    map[string][]ItemBaseEntry
	ItemBaseTypeList []string
	Rares            []string

	ClusterJewels             ClusterJewels
	ClusterJewelInfoForNotable map[string]*ClusterJewelInfo

	Bosses             map[string]Boss
	BossStats          BossStats
	EnemyIsBossTooltip string
	BossSkills         map[string]BossSkillData
	BossSkillsList     []ValLabel

	FoulbornMap map[string]any

	Minions  map[string]*Minion
	Spectres map[string]*Minion

	Skills       map[string]*GrantedEffect
	SkillStatMap map[string]*StatMapEntry

	Gems                           map[string]*Gem
	GemForSkill                    map[*GrantedEffect]string
	GemForBaseName                 map[string]string
	GemsByGameId                   map[string]map[string]*Gem
	GemGrantedEffectIdForVaalGemId map[string]string
	GemVaalGemIdForBaseGemId       map[string]string

	// Uniques holds the unique item text database by type, plus "race" and
	// "new". The "generated" list (Uniques/Special/Generated.lua, real code)
	// is not ported yet.
	Uniques map[string][]string
}

// Scalability is one data.modScalability value entry.
type Scalability struct {
	IsScalable bool     `lua:"isScalable"`
	Formats    []string `lua:"formats"` // nil = absent
}

type Cost struct {
	Resource       string  `lua:"Resource"`
	Stat           *string `lua:"Stat"`
	ResourceString string  `lua:"ResourceString"`
	Divisor        float64 `lua:"Divisor"`
}

type Boss struct {
	ArmourMult  float64 `lua:"armourMult"`
	EvasionMult float64 `lua:"evasionMult"`
	IsUber      bool    `lua:"isUber"`
}

type BossStats struct {
	PinnacleArmourMean  float64 `lua:"PinnacleArmourMean"`
	PinnacleEvasionMean float64 `lua:"PinnacleEvasionMean"`
	UberArmourMean      float64 `lua:"UberArmourMean"`
	UberEvasionMean     float64 `lua:"UberEvasionMean"`
}

// Load assembles the runtime data set.
func Load(src Sources) *Data {
	d := &Data{}
	d.loadMisc(src.Misc)
	d.Misc = miscTable(d.CharacterConstants, d.MonsterConstants)
	d.PowerStatList = buildPowerStatList()
	d.SkillColorMap = []string{colorStrength, colorDexterity, colorIntelligence, colorNormal}
	d.MonsterExperienceLevelMap = buildMonsterExperienceLevelMap(d.Misc)
	d.CursePriority = cursePriority
	d.Keystones = keystones
	d.AilmentTypeList = ailmentTypeList
	d.ElementalAilmentTypeList = elementalAilmentTypeList
	d.NonDamagingAilmentTypeList = nonDamagingAilmentTypeList
	d.NonElementalAilmentTypeList = nonElementalAilmentTypeList
	d.NonDamagingAilment = nonDamagingAilment
	d.DefaultHighPrecision = 1
	d.HighPrecisionMods = highPrecisionMods
	d.ModScalability = map[string][]Scalability{}
	for line, vals := range src.ModScalability {
		list := make([]Scalability, len(vals))
		for i, v := range vals {
			list[i] = Scalability{IsScalable: v.IsScalable, Formats: v.Formats}
		}
		// The reference loads these keys from a Lua string literal, so \n
		// sequences in the described text become real newlines.
		d.ModScalability[luaUnescape(line)] = list
	}
	d.WeaponTypeInfo = weaponTypeInfo
	d.UnarmedWeaponData = unarmedWeaponData
	d.JewelRadii, d.MaxJewelRadius = buildJewelRadii()
	d.JewelRadius = d.JewelRadii["3_16"]
	d.EnchantmentSource = enchantmentSource
	d.TimelessJewelTypes = timelessJewelTypes
	d.TimelessJewelSeedMin = timelessJewelSeedMin
	d.TimelessJewelSeedMax = timelessJewelSeedMax
	d.TimelessJewelAdditions = 337 // #legionAdditions
	d.NodeIDList = nodeIDListTable
	d.AbyssNotableNames = abyssNotableNamesTable
	d.TimelessJewelTradeIDs = timelessJewelTradeIDsTable
	d.TimelessJewelLUTs = map[string]any{}
	d.MapMods = mapModsTable
	d.ItemTagSpecial = itemTagSpecial
	d.ItemTagSpecialExclusionPattern = itemTagSpecialExclusionPattern
	d.CasterTagCrucibleUniques = casterTagCrucibleUniques
	d.MinionTagCrucibleUniques = minionTagCrucibleUniques

	for _, c := range src.Costs {
		cost := Cost{Resource: c.Resource, ResourceString: c.ResourceString, Divisor: float64(c.Divisor)}
		cost.Stat = c.Stat
		d.Costs = append(d.Costs, cost)
	}

	d.loadItemMods(src.Mods)
	d.Essences = loadEssences(src.Essences)
	d.Pantheons = loadPantheons(src.Pantheons)
	d.Crucible = loadCrucible(src.Crucible)
	d.MasterMods = loadMasterMods(src.Masters)
	d.FlavourText = loadFlavourText(src.FlavourText)
	d.Enchantments = loadEnchantments(src.Enchants)
	d.HelmetEnchants = loadHelmetEnchants(src.Enchants)

	d.loadBases(src.Bases)
	d.loadRareLikeUniques()
	d.ClusterJewels = loadClusterJewels(src.Cluster)
	d.ClusterJewelInfoForNotable = computeClusterJewelInfo(d.ItemMods["JewelCluster"], d.ClusterJewels)

	d.loadBosses(src.Boss)
	d.BossSkills, d.BossSkillsList = loadBossSkills(src.Boss)

	d.SkillStatMap = skillStatMap
	d.loadSkills(src.Skills, src.StatMapCopies)
	d.loadGems(src.Skills)
	d.loadMinions(src.MinionsDoc)

	d.Uniques = map[string][]string{}
	for typ, f := range src.Uniques {
		var blobs []string
		for _, sec := range f.Sections {
			for _, item := range sec.Items {
				blobs = append(blobs, strings.Join(item, "\n")+"\n")
			}
		}
		d.Uniques[typ] = blobs
	}
	d.Uniques["race"] = uniquesRace
	d.Uniques["new"] = []string{}
	// Data/Uniques/graft.lua is a hand-maintained empty placeholder (the
	// exporter doesn't generate graft uniques yet).
	d.Uniques["graft"] = []string{}
	d.buildGeneratedUniques()

	if len(src.FoulbornMapJSONC) > 0 {
		d.FoulbornMap = map[string]any{}
		if err := json.Unmarshal(stripJSONCComments(src.FoulbornMapJSONC), &d.FoulbornMap); err != nil {
			panic("data: foulborn map: " + err.Error())
		}
	}
	return d
}

// stripJSONCComments removes full-line // comments (the only kind the file
// uses).
func stripJSONCComments(b []byte) []byte {
	lines := strings.Split(string(b), "\n")
	out := lines[:0]
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "//") {
			continue
		}
		out = append(out, l)
	}
	return []byte(strings.Join(out, "\n"))
}

// loadMisc unpacks the Data/Misc.lua document.
func (d *Data) loadMisc(m gamedata.MiscData) {
	d.MonsterEvasionTable = m.Misc.MonsterEvasion
	d.MonsterAccuracyTable = m.Misc.MonsterAccuracy
	d.MonsterLifeTable = m.Misc.MonsterLife
	d.MonsterLifeTable2 = m.Misc.MonsterLife2
	d.MonsterLifeTable3 = m.Misc.MonsterLife3
	d.MonsterAllyLifeTable = m.Misc.MonsterAllyLife
	d.MonsterDamageTable = m.Misc.MonsterDamage
	d.MonsterAllyDamageTable = m.Misc.MonsterAllyDamage
	d.MonsterArmourTable = m.Misc.MonsterArmour
	d.MonsterAilmentThresholdTable = m.Misc.MonsterAilmentThreshold
	d.MonsterPhysConversionMultiTable = m.Misc.MonsterPhysConversionMulti
	d.GameConstants = map[string]float64{}
	for _, c := range m.Misc.GameConstants {
		d.GameConstants[c.Id] = c.Value
	}
	d.CharacterConstants = otConstantMap(m.Misc.CharacterConstants)
	d.MonsterConstants = otConstantMap(m.Misc.MonsterConstants)
	d.TotemLifeMult = map[int64]float64{}
	for _, t := range m.Misc.TotemLifeMult {
		d.TotemLifeMult[t.Id] = t.Mult
	}
	d.MonsterVarietyLifeMult = map[string]float64{}
	for _, v := range m.Misc.MonsterVarietyLifeMult {
		d.MonsterVarietyLifeMult[v.Name] = v.Mult
	}
	levelMap := func(list []gamedata.LevelMult) map[int64]float64 {
		out := map[int64]float64{}
		for _, v := range list {
			out[v.Level] = v.Mult
		}
		return out
	}
	d.MapLevelLifeMult = levelMap(m.Misc.MapLevelLifeMult)
	d.MapLevelBossLifeMult = levelMap(m.Misc.MapLevelBossLifeMult)
	d.MapLevelBossAilmentMult = levelMap(m.Misc.MapLevelBossAilmentMult)
	d.GoldRespecPrices = m.Misc.GoldRespecPrices
}

// otConstantMap evaluates the .ot constants' raw value text as Lua numbers.
func otConstantMap(kvs []gamedata.KV) map[string]float64 {
	out := map[string]float64{}
	for _, kv := range kvs {
		out[kv.Key] = luaTonumber(kv.Value)
	}
	return out
}

func luaTonumber(s string) float64 {
	s = strings.TrimSpace(s)
	if n, err := parseLuaNumber(s); err == nil {
		return n
	}
	panic("data: non-numeric .ot constant " + s)
}

func buildMonsterExperienceLevelMap(misc Misc) map[int]float64 {
	effective := func(areaLevel float64) float64 {
		if areaLevel <= misc.MaxExperiencePenaltyFreeAreaLevel {
			return areaLevel
		}
		n := areaLevel - misc.MaxExperiencePenaltyFreeAreaLevel
		return areaLevel - n*(n+1)/2*misc.ExperiencePenaltyMultiplier
	}
	out := map[int]float64{}
	for i := 1; i <= int(misc.MaxEnemyLevel)+2; i++ {
		out[i] = effective(float64(i))
	}
	return out
}

func (d *Data) loadBosses(b gamedata.BossData) {
	d.Bosses = map[string]Boss{}
	var count, uberCount float64
	var armourTotal, evasionTotal float64
	var uberArmourTotal, uberEvasionTotal float64
	for _, bm := range b.Bosses {
		if bm == nil {
			continue
		}
		d.Bosses[bm.DisplayName] = Boss{
			ArmourMult:  float64(bm.ArmourMult),
			EvasionMult: float64(bm.EvasionMult),
			IsUber:      bm.IsUber,
		}
		if bm.IsUber {
			uberCount++
			uberArmourTotal += float64(bm.ArmourMult)
			uberEvasionTotal += float64(bm.EvasionMult)
		}
		count++
		armourTotal += float64(bm.ArmourMult)
		evasionTotal += float64(bm.EvasionMult)
	}
	d.BossStats = BossStats{
		PinnacleArmourMean:  100 + armourTotal/count,
		PinnacleEvasionMean: 100 + evasionTotal/count,
		UberArmourMean:      100 + uberArmourTotal/uberCount,
		UberEvasionMean:     100 + uberEvasionTotal/uberCount,
	}
	d.EnemyIsBossTooltip = bossTooltip(d.Misc, d.BossStats)
}

func bossTooltip(misc Misc, stats BossStats) string {
	fs := func(v float64) string { return luaIntString(math.Floor(v)) }
	return "Bosses' damage is monster damage scaled to an average damage of their attacks\n" +
		"This is divided by 4.40 to represent 4 damage types + some (40% as much) ^xD02090chaos\n" +
		"^7Fill in the exact damage numbers if more precision is needed\n\n" +
		"Bosses' armour and evasion multiplier are calculated using the average of the boss type\n\n" +
		"Standard Boss adds the following modifiers:\n" +
		"\t+40% to enemy Elemental Resistances\n" +
		"\t+25% to enemy ^xD02090Chaos Resistance\n" +
		"\t^7" + fs(misc.StdBossDPSMult*100) + "% of monster Damage of each type\n" +
		"\t" + fs(misc.StdBossDPSMult*4.4*100) + "% of monster Damage total\n\n" +
		"Guardian / Pinnacle Boss adds the following modifiers:\n" +
		"\t+50% to enemy Elemental Resistances\n" +
		"\t+30% to enemy ^xD02090Chaos Resistance\n" +
		"\t^7" + fs(stats.PinnacleArmourMean) + "% of monster Armour\n" +
		"\t" + fs(stats.PinnacleEvasionMean) + "% of monster ^x33FF77Evasion\n" +
		"\t^7" + fs(misc.PinnacleBossDPSMult*100) + "% of monster Damage of each type\n" +
		"\t" + fs(misc.PinnacleBossDPSMult*4.4*100) + "% of monster Damage total\n" +
		"\t" + luaNumString(misc.PinnacleBossPen) + "% penetration\n\n" +
		"Uber Pinnacle Boss adds the following modifiers:\n" +
		"\t+50% to enemy Elemental Resistances\n" +
		"\t+30% to enemy ^xD02090Chaos Resistance\n" +
		"\t^7" + fs(stats.UberArmourMean) + "% of monster Armour\n" +
		"\t" + fs(stats.UberEvasionMean) + "% of monster ^x33FF77Evasion\n" +
		"\t^770% less to enemy Damage taken\n" +
		"\t" + fs(misc.UberBossDPSMult*100) + "% of monster Damage of each type\n" +
		"\t" + fs(misc.UberBossDPSMult*4.25*100) + "% of monster Damage total\n" +
		"\t" + luaNumString(misc.UberBossPen) + "% penetration"
}
