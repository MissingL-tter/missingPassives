package test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/calc"
	"github.com/MissingL-tter/missingPassives/internal/luacanon"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// calcDumpFiles lists the per-corpus archive dumps tools/dump_calc.lua
// writes (one process per source build).
var calcDumpFiles = []string{"calc_empty.jsonl", "calc_coc.jsonl", "calc_zombies.jsonl", "calc_lowlife.jsonl", "calc_spectre.jsonl", "calc_cyclone.jsonl", "calc_rf.jsonl", "calc_holyrelic.jsonl", "calc_eblade.jsonl"}

func decodeCalcModList(v any) []*modparser.Mod {
	m := v.(map[string]any)
	out := make([]*modparser.Mod, len(m))
	for i := 1; i <= len(m); i++ {
		out[i-1] = decodeCanonMod(m[strconv.Itoa(i)].(map[string]any))
	}
	return out
}

func optCalcString(m map[string]any, k string) *string {
	if v, ok := m[k]; ok {
		s := v.(string)
		return &s
	}
	return nil
}

func optCalcBool(m map[string]any, k string) *bool {
	if v, ok := m[k]; ok {
		b := v.(bool)
		return &b
	}
	return nil
}

func decodeCalcCounts(v any) map[string]float64 {
	out := map[string]float64{}
	for k, e := range v.(map[string]any) {
		out[k] = e.(float64)
	}
	return out
}

func optCalcFloat(m map[string]any, k string) *float64 {
	if v, ok := m[k]; ok {
		n := v.(float64)
		return &n
	}
	return nil
}

// decodeCalcArray turns a canon array object {"1":..,"2":..} into a slice.
// Only the contiguous run from 1 counts, as Lua's array part does: a table
// can carry named keys alongside it (item.sockets.colourAlwaysMatches on
// Dialla's Malefaction).
func decodeCalcArray(v any) []any {
	m := v.(map[string]any)
	var out []any
	for i := 1; ; i++ {
		e, ok := m[strconv.Itoa(i)]
		if !ok {
			return out
		}
		out = append(out, e)
	}
}

func decodeCalcStrings(v any) []string {
	arr := decodeCalcArray(v)
	out := make([]string, len(arr))
	for i, e := range arr {
		out[i] = e.(string)
	}
	return out
}

func decodeCalcScalarMaps(v any) []map[string]any {
	arr := decodeCalcArray(v)
	out := make([]map[string]any, len(arr))
	for i, e := range arr {
		out[i] = e.(map[string]any)
	}
	return out
}

func decodeCalcItem(m map[string]any) *calc.ItemInput {
	item := &calc.ItemInput{
		Name:                        m["name"].(string),
		ModSource:                   optCalcString(m, "modSource"),
		Title:                       optCalcString(m, "title"),
		BaseName:                    optCalcString(m, "baseName"),
		Type:                        m["type"].(string),
		Rarity:                      m["rarity"].(string),
		Corrupted:                   optCalcBool(m, "corrupted"),
		Shaper:                      optCalcBool(m, "shaper"),
		Elder:                       optCalcBool(m, "elder"),
		Adjudicator:                 optCalcBool(m, "adjudicator"),
		Basilisk:                    optCalcBool(m, "basilisk"),
		Crusader:                    optCalcBool(m, "crusader"),
		Eyrie:                       optCalcBool(m, "eyrie"),
		Foulborn:                    optCalcBool(m, "foulborn"),
		ClassRestriction:            optCalcString(m, "classRestriction"),
		Limit:                       optCalcFloat(m, "limit"),
		AbyssalSocketCount:          optCalcFloat(m, "abyssalSocketCount"),
		SocketedJewelEffectModifier: optCalcFloat(m, "socketedJewelEffectModifier"),
		JewelRadiusIndex:            optCalcFloat(m, "jewelRadiusIndex"),
		GrantedSkills:               decodeCalcScalarMaps(m["grantedSkills"]),
		ExplicitLines:               decodeCalcStrings(m["explicitLines"]),
		OtherLines:                  decodeCalcStrings(m["otherLines"]),
	}
	item.Quality = optCalcFloat(m, "quality")
	if v, ok := m["base"]; ok {
		b := v.(map[string]any)
		item.Base = &calc.ItemBaseInput{SubType: optCalcString(b, "subType"), Type: optCalcString(b, "type")}
		if fv, ok := b["flask"]; ok {
			f := fv.(map[string]any)
			item.Base.Flask = &calc.FlaskBaseInput{Life: f["life"], Mana: f["mana"]}
		}
	}
	if v, ok := m["modList"]; ok {
		item.ModList = decodeCalcModList(v)
	}
	if v, ok := m["slotModList"]; ok {
		item.SlotModList = map[int][]*modparser.Mod{}
		for idx, l := range v.(map[string]any) {
			n, err := strconv.Atoi(idx)
			if err != nil {
				panic("bad slotModList index " + idx)
			}
			item.SlotModList[n] = decodeCalcModList(l)
		}
	}
	if v, ok := m["baseModList"]; ok {
		item.BaseModList = decodeCalcModList(v)
	}
	if v, ok := m["buffModList"]; ok {
		item.BuffModList = decodeCalcModList(v)
	}
	if v, ok := m["requirements"]; ok {
		item.Requirements = decodeCalcCounts(v)
	}
	if v, ok := m["sockets"]; ok {
		item.Sockets = decodeCalcScalarMaps(v)
	}
	for key, dst := range map[string]*map[string]any{
		"jewelData": &item.JewelData, "flaskData": &item.FlaskData,
		"tinctureData": &item.TinctureData, "armourData": &item.ArmourData,
	} {
		if v, ok := m[key]; ok {
			*dst = v.(map[string]any)
		}
	}
	if v, ok := m["funcTypes"]; ok {
		item.FuncTypes = decodeCalcStrings(v)
	}
	if v, ok := m["weaponData"]; ok {
		item.WeaponData = map[int]map[string]any{}
		for idx, side := range v.(map[string]any) {
			n, err := strconv.Atoi(idx)
			if err != nil {
				panic("bad weaponData index " + idx)
			}
			item.WeaponData[n] = side.(map[string]any)
		}
	}
	return item
}

func decodeCalcItemsTab(v any) *calc.ItemsTabInput {
	m := v.(map[string]any)
	tab := &calc.ItemsTabInput{
		UseSecondWeaponSet: optCalcBool(m, "useSecondWeaponSet"),
		Items:              map[int]*calc.ItemInput{},
	}
	for _, sv := range decodeCalcArray(m["slots"]) {
		s := sv.(map[string]any)
		slot := &calc.SlotInput{
			SlotName:       s["slotName"].(string),
			Label:          s["label"].(string),
			SlotNum:        optCalcFloat(s, "slotNum"),
			WeaponSet:      optCalcFloat(s, "weaponSet"),
			NodeID:         optCalcFloat(s, "nodeId"),
			Active:         optCalcBool(s, "active"),
			ParentSlotName: optCalcString(s, "parentSlotName"),
			ItemID:         optCalcFloat(s, "itemId"),
		}
		slot.ContainJewelSocket = optCalcBool(s, "containJewelSocket")
		if rv, ok := s["radiusNodes"]; ok {
			slot.RadiusNodes = map[int]string{}
			for idStr, tv := range rv.(map[string]any) {
				id, err := strconv.Atoi(idStr)
				if err != nil {
					panic("bad radiusNodes id " + idStr)
				}
				slot.RadiusNodes[id] = tv.(string)
			}
		}
		if av, ok := s["radiusAttributes"]; ok {
			slot.RadiusAttributes = av.(map[string]any)
		}
		tab.Slots = append(tab.Slots, slot)
	}
	for idStr, iv := range m["items"].(map[string]any) {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			panic("bad item id " + idStr)
		}
		tab.Items[id] = decodeCalcItem(iv.(map[string]any))
	}
	tab.ItemSets = map[int]*calc.ItemSetInput{}
	for idStr, sv := range m["itemSets"].(map[string]any) {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			panic("bad item set id " + idStr)
		}
		s := sv.(map[string]any)
		set := &calc.ItemSetInput{
			UseSecondWeaponSet: optCalcBool(s, "useSecondWeaponSet"),
			Slots:              decodeCalcCounts(s["slots"]),
		}
		tab.ItemSets[id] = set
	}
	tab.ItemSetOrderList = []float64{}
	for _, v := range decodeCalcArray(m["itemSetOrderList"]) {
		tab.ItemSetOrderList = append(tab.ItemSetOrderList, v.(float64))
	}
	return tab
}

func decodeCalcNode(n map[string]any) *calc.NodeInput {
	node := &calc.NodeInput{
		ID:           n["id"].(float64),
		Type:         n["type"].(string),
		Name:         optCalcString(n, "name"),
		DN:           optCalcString(n, "dn"),
		IsTattoo:     optCalcBool(n, "isTattoo"),
		OverrideType: optCalcString(n, "overrideType"),
		ConqueredBy:  optCalcBool(n, "conqueredBy"),
		ModList:      decodeCalcModList(n["modList"]),
	}
	if v, ok := n["distanceToClassStart"].(float64); ok {
		node.DistanceToClassStart = &v
	}
	if kv, ok := n["keystoneMod"]; ok {
		node.KeystoneMod = decodeCanonMod(kv.(map[string]any))
	}
	return node
}

func decodeCalcSkillsTab(v any) *calc.SkillsTabInput {
	m := v.(map[string]any)
	tab := &calc.SkillsTabInput{SocketGroups: []*calc.SocketGroupInput{}}
	if iv, ok := m["imbuedSupportBySlot"]; ok {
		tab.ImbuedSupportBySlot = map[string]string{}
		for k, e := range iv.(map[string]any) {
			tab.ImbuedSupportBySlot[k] = e.(string)
		}
	}
	for _, gv := range decodeCalcArray(m["socketGroupList"]) {
		g := gv.(map[string]any)
		sg := &calc.SocketGroupInput{
			KV:      g["kv"].(map[string]any),
			GemList: []*calc.SocketGemInput{},
		}
		for _, gemv := range decodeCalcArray(g["gemList"]) {
			gm := gemv.(map[string]any)
			sg.GemList = append(sg.GemList, &calc.SocketGemInput{
				KV:                  gm["kv"].(map[string]any),
				GemDataID:           optCalcString(gm, "gemDataId"),
				GrantedEffectID:     optCalcString(gm, "grantedEffectId"),
				ExplodeSourceItemID: optCalcFloat(gm, "explodeSourceItemId"),
				ExplodeSourceNodeID: optCalcFloat(gm, "explodeSourceNodeId"),
			})
		}
		tab.SocketGroups = append(tab.SocketGroups, sg)
	}
	return tab
}

func decodeCalcFixture(c map[string]any) *calc.BuildInput {
	spec := c["spec"].(map[string]any)
	allocNodes := map[int]*calc.NodeInput{}
	for idStr, nv := range spec["allocNodes"].(map[string]any) {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			panic("non-integer allocNodes key " + idStr)
		}
		allocNodes[id] = decodeCalcNode(nv.(map[string]any))
	}
	keystoneMap := map[string][]*modparser.Mod{}
	for name, mods := range spec["keystoneMap"].(map[string]any) {
		keystoneMap[name] = decodeCalcModList(mods)
	}
	radiusNodeData := map[int]*calc.NodeInput{}
	for idStr, nv := range spec["radiusNodeData"].(map[string]any) {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			panic("bad radiusNodeData id " + idStr)
		}
		radiusNodeData[id] = decodeCalcNode(nv.(map[string]any))
	}
	stats := c["classStats"].(map[string]any)
	var configEnemyLevel *float64
	if v, ok := c["configEnemyLevel"]; ok {
		n := v.(float64)
		configEnemyLevel = &n
	}
	return &calc.BuildInput{
		CharacterLevel:   c["characterLevel"].(float64),
		ConfigEnemyLevel: configEnemyLevel,
		ClassID:          c["classId"].(float64),
		CurClassName:     c["curClassName"].(string),
		TreeVersion:      c["treeVersion"].(string),
		MainSocketGroup:  c["mainSocketGroup"].(float64),
		ClassStats: calc.ClassStats{
			BaseStr: stats["base_str"].(float64),
			BaseDex: stats["base_dex"].(float64),
			BaseInt: stats["base_int"].(float64),
		},
		ItemsTab:           decodeCalcItemsTab(c["itemsTab"]),
		SkillsTab:          decodeCalcSkillsTab(c["skillsTab"]),
		SpectreList:        decodeCalcStrings(c["spectreList"]),
		ConfigInput:        c["configInput"].(map[string]any),
		ConfigPlaceholder:  c["configPlaceholder"].(map[string]any),
		ConfigModList:      decodeCalcModList(c["configModList"]),
		ConfigEnemyModList: decodeCalcModList(c["configEnemyModList"]),
		PartyEnemyModList:  decodeCalcModList(c["partyEnemyModList"]),
		Spec: &calc.SpecInput{
			AllocNodes:                allocNodes,
			KeystoneMap:               keystoneMap,
			RadiusNodeData:            radiusNodeData,
			AllocatedNotableCount:     spec["allocatedNotableCount"].(float64),
			AllocatedKeystoneCount:    spec["allocatedKeystoneCount"].(float64),
			AllocatedMasteryCount:     spec["allocatedMasteryCount"].(float64),
			AllocatedMasteryTypeCount: spec["allocatedMasteryTypeCount"].(float64),
			AllocatedMasteryTypes:     decodeCalcCounts(spec["allocatedMasteryTypes"]),
			AllocatedTattooTypes:      decodeCalcCounts(spec["allocatedTattooTypes"]),
		},
	}
}

// forEachCalcRecord streams every record of one dump file.
func forEachCalcRecord(t *testing.T, path string, fn func(k, c string)) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<28)
	for sc.Scan() {
		var rec struct {
			K string `json:"k"`
			C string `json:"c"`
		}
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatal(err)
		}
		fn(rec.K, rec.C)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
}

// TestCalcFixtureEcho verifies the fixture decode is lossless: every
// .fixture record decodes into calc.BuildInput and re-canonicalises to the
// dump's exact bytes. A decode that drops or reshapes a field the later
// initEnv stages depend on fails here, before any calc logic runs.
func TestCalcFixtureEcho(t *testing.T) {
	fixtures := 0
	for _, name := range calcDumpFiles {
		path := filepath.Join("testdata", name)
		if _, err := os.Stat(path); err != nil {
			t.Skipf("archive dump not present: %v", err)
		}
		forEachCalcRecord(t, path, func(k, c string) {
			if len(k) < 8 || k[len(k)-8:] != ".fixture" {
				return
			}
			fixtures++
			var m map[string]any
			if err := json.Unmarshal([]byte(c), &m); err != nil {
				t.Fatalf("%s %s: %v", name, k, err)
			}
			in := decodeCalcFixture(m)
			// Fixtures are emitted round-trippably (%.17g), so the echo
			// re-encodes at the same precision.
			if got := luacanon.EncodeExact(in); got != c {
				t.Errorf("%s %s: echo diverged\n%s", name, k, diffWindow(got, c))
			}
		})
	}
	if fixtures < 4 {
		t.Fatalf("expected at least 4 fixture records, saw %d", fixtures)
	}
	t.Logf("calc fixture echo: %d fixtures byte-identical", fixtures)
}

// dbShadow re-canonicalises one DB the way dump_calc.lua's dbState does.
type dbShadow struct {
	Mods        map[string][]*modparser.Mod `lua:"mods"`
	Conditions  map[string]any              `lua:"conditions"`
	Multipliers map[string]float64          `lua:"multipliers"`
}

type dbsShadow struct {
	Mod   dbShadow `lua:"mod"`
	Enemy dbShadow `lua:"enemy"`
	Item  dbShadow `lua:"item"`
}

func shadowOf(db *modstore.DB) dbShadow {
	return dbShadow{Mods: db.Mods, Conditions: db.Conditions, Multipliers: db.Multipliers}
}

// cfgScalars mirrors dump_calc.lua's scalars() over a skill Cfg.
func cfgScalars(cfg *modstore.Cfg) map[string]any {
	m := map[string]any{}
	if cfg.Flags != nil {
		m["flags"] = *cfg.Flags
	}
	if cfg.KeywordFlags != nil {
		m["keywordFlags"] = *cfg.KeywordFlags
	}
	if cfg.SkillName != "" {
		m["skillName"] = cfg.SkillName
	}
	if cfg.SummonSkillName != "" {
		m["summonSkillName"] = cfg.SummonSkillName
	}
	if cfg.SkillDist != nil {
		m["skillDist"] = *cfg.SkillDist
	}
	if cfg.SlotName != "" {
		m["slotName"] = cfg.SlotName
	}
	if cfg.SocketColor != "" {
		m["socketColor"] = cfg.SocketColor
	}
	if cfg.SocketNum != nil {
		m["socketNum"] = *cfg.SocketNum
	}
	if cfg.SkillPart != nil {
		m["skillPart"] = cfg.SkillPart
	}
	return m
}

// scalarsOnly keeps string/number/bool values (dump-side scalars()).
func scalarsOnly(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		switch v.(type) {
		case string, float64, int64, int, bool:
			out[k] = v
		}
	}
	return out
}

func nonNilMods(mods []*modparser.Mod) []*modparser.Mod {
	if mods == nil {
		return []*modparser.Mod{}
	}
	return mods
}

type skillListShadow struct {
	ModList           []*modparser.Mod `lua:"modList"`
	Cfg               map[string]any   `lua:"cfg"`
	Flags             map[string]bool  `lua:"flags"`
	Data              map[string]any   `lua:"data"`
	Buffs             []map[string]any `lua:"buffs"`
	Weapon1Flags      *int64           `lua:"weapon1Flags"`
	Weapon2Flags      *int64           `lua:"weapon2Flags"`
	Weapon1Cfg        map[string]any   `lua:"weapon1Cfg"`
	Weapon1Cond       map[string]bool  `lua:"weapon1Cond"`
	Weapon2Cfg        map[string]any   `lua:"weapon2Cfg"`
	Weapon2Cond       map[string]bool  `lua:"weapon2Cond"`
	DisableReason     string           `lua:"disableReason,omitempty"`
	SkillPartName     string           `lua:"skillPartName,omitempty"`
	SkillTotemId      *float64         `lua:"skillTotemId"`
	ExtraSkillModList []*modparser.Mod `lua:"extraSkillModList"`
	MinionList        []string         `lua:"minionList"`
	Minion            map[string]any   `lua:"minion"`
}

func skillListShadowOf(env *calc.Env, as *calc.ActiveSkill) skillListShadow {
	sh := skillListShadow{
		ModList:           nonNilMods(as.SkillModList.Mods),
		Cfg:               cfgScalars(as.SkillCfg),
		Flags:             as.SkillFlags,
		Data:              scalarsOnly(as.SkillData),
		Buffs:             []map[string]any{},
		Weapon1Flags:      as.Weapon1Flags,
		Weapon2Flags:      as.Weapon2Flags,
		DisableReason:     as.DisableReason,
		SkillPartName:     as.SkillPartName,
		SkillTotemId:      as.SkillTotemId,
		ExtraSkillModList: nonNilMods(as.ExtraSkillModList),
	}
	for _, buff := range as.BuffListTyped {
		b := scalarsOnly(buff.KV)
		b["modList"] = nonNilMods(buff.ModList)
		sh.Buffs = append(sh.Buffs, b)
	}
	if as.Weapon1Cfg != nil {
		sh.Weapon1Cfg = cfgScalars(as.Weapon1Cfg)
		sh.Weapon1Cond = as.Weapon1Cfg.SkillCond
	}
	if as.Weapon2Cfg != nil {
		sh.Weapon2Cfg = cfgScalars(as.Weapon2Cfg)
		sh.Weapon2Cond = as.Weapon2Cfg.SkillCond
	}
	// nil = the early-disable return ran before minionList was set
	sh.MinionList = as.MinionList
	if as.Minion != nil {
		sh.Minion = map[string]any{
			"type":        as.Minion.Type,
			"level":       as.Minion.Level,
			"hostile":     as.Minion.Hostile,
			"weaponData1": scalarsOnly(as.Minion.WeaponData1),
			"weaponData2": scalarsOnly(as.Minion.WeaponData2),
		}
	}
	return sh
}

func decodeEnergyBladeItems(c string) map[string]*calc.ItemInput {
	var m map[string]map[string]any
	if err := json.Unmarshal([]byte(c), &m); err != nil {
		panic(err)
	}
	out := map[string]*calc.ItemInput{}
	for slot, it := range m {
		out[slot] = decodeCalcItem(it)
	}
	return out
}

func decodeAllocOrders(c string) [][]int {
	// Absent from dumps taken before the key existed; a build with no
	// mirages consumes no orders either way.
	if c == "" {
		return nil
	}
	var m map[string]map[string]any
	if err := json.Unmarshal([]byte(c), &m); err != nil {
		panic(err)
	}
	out := make([][]int, len(m))
	for i := 1; i <= len(m); i++ {
		call := m[strconv.Itoa(i)]
		order := make([]int, len(call))
		for j := 1; j <= len(call); j++ {
			order[j-1] = int(call[strconv.Itoa(j)].(float64))
		}
		out[i-1] = order
	}
	return out
}

func decodeGrantedPassiveNodes(c string) map[string]*calc.NodeInput {
	var m map[string]map[string]any
	if err := json.Unmarshal([]byte(c), &m); err != nil {
		panic(err)
	}
	out := map[string]*calc.NodeInput{}
	for name, n := range m {
		out[name] = decodeCalcNode(n)
	}
	return out
}

// TestCalcInitEnvAgainstReference replays each fixture through calc.InitEnv
// and compares the three mod databases byte-for-byte against the archive's
// post-initEnv state. Variants with items or skills wait on those stages;
// the variant list below only ever grows.
func TestCalcInitEnvAgainstReference(t *testing.T) {
	loadGameData(t)
	variants := map[string]string{ // variant -> dump file
		"empty":                 "calc_empty.jsonl",
		"coc.treeonly":          "calc_coc.jsonl",
		"coc.noskills":          "calc_coc.jsonl",
		"coc.full":              "calc_coc.jsonl",
		"zombies.treeonly":      "calc_zombies.jsonl",
		"zombies.noskills":      "calc_zombies.jsonl",
		"zombies.full":          "calc_zombies.jsonl",
		"lowlife.treeonly":      "calc_lowlife.jsonl",
		"lowlife.noskills":      "calc_lowlife.jsonl",
		"lowlife.full":          "calc_lowlife.jsonl",
		"spectre.treeonly":      "calc_spectre.jsonl",
		"spectre.noskills":      "calc_spectre.jsonl",
		"spectre.full":          "calc_spectre.jsonl",
		"cyclone.treeonly":      "calc_cyclone.jsonl",
		"cyclone.noskills":      "calc_cyclone.jsonl",
		"cyclone.full":          "calc_cyclone.jsonl",
		"rf.treeonly":           "calc_rf.jsonl",
		"rf.noskills":           "calc_rf.jsonl",
		"rf.full":               "calc_rf.jsonl",
		"holyrelic.treeonly":    "calc_holyrelic.jsonl",
		"holyrelic.noskills":    "calc_holyrelic.jsonl",
		"holyrelic.full":        "calc_holyrelic.jsonl",
		"eblade.treeonly":       "calc_eblade.jsonl",
		"eblade.noskills":       "calc_eblade.jsonl",
		"eblade.full":           "calc_eblade.jsonl",
		"cocuser.treeonly":      "calc_cocuser.jsonl",
		"cocuser.noskills":      "calc_cocuser.jsonl",
		"cocuser.full":          "calc_cocuser.jsonl",
		"dualstrike.treeonly":   "calc_dualstrike.jsonl",
		"dualstrike.noskills":   "calc_dualstrike.jsonl",
		"dualstrike.full":       "calc_dualstrike.jsonl",
		"bfbb.treeonly":         "calc_bfbb.jsonl",
		"bfbb.noskills":         "calc_bfbb.jsonl",
		"bfbb.full":             "calc_bfbb.jsonl",
		"ballista.treeonly":     "calc_ballista.jsonl",
		"ballista.noskills":     "calc_ballista.jsonl",
		"ballista.full":         "calc_ballista.jsonl",
		"trap.treeonly":         "calc_trap.jsonl",
		"trap.noskills":         "calc_trap.jsonl",
		"trap.full":             "calc_trap.jsonl",
		"mirage.treeonly":       "calc_mirage.jsonl",
		"mirage.noskills":       "calc_mirage.jsonl",
		"mirage.full":           "calc_mirage.jsonl",
		"exparrow.treeonly":     "calc_exparrow.jsonl",
		"exparrow.noskills":     "calc_exparrow.jsonl",
		"exparrow.full":         "calc_exparrow.jsonl",
		"blight.treeonly":       "calc_blight.jsonl",
		"blight.noskills":       "calc_blight.jsonl",
		"blight.full":           "calc_blight.jsonl",
		"cwc.treeonly":          "calc_cwc.jsonl",
		"cwc.noskills":          "calc_cwc.jsonl",
		"cwc.full":              "calc_cwc.jsonl",
		"poetpen.treeonly":      "calc_poetpen.jsonl",
		"poetpen.noskills":      "calc_poetpen.jsonl",
		"poetpen.full":          "calc_poetpen.jsonl",
		"cospri.treeonly":       "calc_cospri.jsonl",
		"cospri.noskills":       "calc_cospri.jsonl",
		"cospri.full":           "calc_cospri.jsonl",
		"saviour.treeonly":      "calc_saviour.jsonl",
		"saviour.noskills":      "calc_saviour.jsonl",
		"saviour.full":          "calc_saviour.jsonl",
		"slinger.treeonly":      "calc_slinger.jsonl",
		"slinger.noskills":      "calc_slinger.jsonl",
		"slinger.full":          "calc_slinger.jsonl",
		"mine.treeonly":         "calc_mine.jsonl",
		"mine.noskills":         "calc_mine.jsonl",
		"mine.full":             "calc_mine.jsonl",
		"brand.treeonly":        "calc_brand.jsonl",
		"brand.noskills":        "calc_brand.jsonl",
		"brand.full":            "calc_brand.jsonl",
		"absolution.treeonly":   "calc_absolution.jsonl",
		"absolution.noskills":   "calc_absolution.jsonl",
		"absolution.full":       "calc_absolution.jsonl",
		"corrupting.treeonly":   "calc_corrupting.jsonl",
		"corrupting.noskills":   "calc_corrupting.jsonl",
		"corrupting.full":       "calc_corrupting.jsonl",
		"fissure.treeonly":      "calc_fissure.jsonl",
		"fissure.noskills":      "calc_fissure.jsonl",
		"fissure.full":          "calc_fissure.jsonl",
		"tornado.treeonly":      "calc_tornado.jsonl",
		"tornado.noskills":      "calc_tornado.jsonl",
		"tornado.full":          "calc_tornado.jsonl",
		"toxicrain.treeonly":    "calc_toxicrain.jsonl",
		"toxicrain.noskills":    "calc_toxicrain.jsonl",
		"toxicrain.full":        "calc_toxicrain.jsonl",
		"moltenstrike.treeonly": "calc_moltenstrike.jsonl",
		"moltenstrike.noskills": "calc_moltenstrike.jsonl",
		"moltenstrike.full":     "calc_moltenstrike.jsonl",
		"earthquake.treeonly":   "calc_earthquake.jsonl",
		"earthquake.noskills":   "calc_earthquake.jsonl",
		"earthquake.full":       "calc_earthquake.jsonl",
		"arctotem.treeonly":     "calc_arctotem.jsonl",
		"arctotem.noskills":     "calc_arctotem.jsonl",
		"arctotem.full":         "calc_arctotem.jsonl",
		"doomblast.treeonly":    "calc_doomblast.jsonl",
		"doomblast.noskills":    "calc_doomblast.jsonl",
		"doomblast.full":        "calc_doomblast.jsonl",
		"callpyre.treeonly":     "calc_callpyre.jsonl",
		"callpyre.noskills":     "calc_callpyre.jsonl",
		"callpyre.full":         "calc_callpyre.jsonl",
		"tempest.treeonly":      "calc_tempest.jsonl",
		"tempest.noskills":      "calc_tempest.jsonl",
		"tempest.full":          "calc_tempest.jsonl",
		"voidstorm.treeonly":    "calc_voidstorm.jsonl",
		"voidstorm.noskills":    "calc_voidstorm.jsonl",
		"voidstorm.full":        "calc_voidstorm.jsonl",
		"shockwave.treeonly":    "calc_shockwave.jsonl",
		"shockwave.noskills":    "calc_shockwave.jsonl",
		"shockwave.full":        "calc_shockwave.jsonl",
		"toad.treeonly":         "calc_toad.jsonl",
		"toad.noskills":         "calc_toad.jsonl",
		"toad.full":             "calc_toad.jsonl",
		"trig1.treeonly":        "calc_trig1.jsonl",
		"trig1.noskills":        "calc_trig1.jsonl",
		"trig1.full":            "calc_trig1.jsonl",
		"trig2.treeonly":        "calc_trig2.jsonl",
		"trig2.noskills":        "calc_trig2.jsonl",
		"trig2.full":            "calc_trig2.jsonl",
		"trig3.treeonly":        "calc_trig3.jsonl",
		"trig3.noskills":        "calc_trig3.jsonl",
		"trig3.full":            "calc_trig3.jsonl",
		"trig4.treeonly":        "calc_trig4.jsonl",
		"trig4.noskills":        "calc_trig4.jsonl",
		"trig4.full":            "calc_trig4.jsonl",
		"mjolner.treeonly":      "calc_mjolner.jsonl",
		"mjolner.noskills":      "calc_mjolner.jsonl",
		"mjolner.full":          "calc_mjolner.jsonl",
		"stages.treeonly":       "calc_stages.jsonl",
		"stages.noskills":       "calc_stages.jsonl",
		"stages.full":           "calc_stages.jsonl",
		"doomexp.treeonly":      "calc_doomexp.jsonl",
		"doomexp.noskills":      "calc_doomexp.jsonl",
		"doomexp.full":          "calc_doomexp.jsonl",
		"doomhex.treeonly":      "calc_doomhex.jsonl",
		"doomhex.noskills":      "calc_doomhex.jsonl",
		"doomhex.full":          "calc_doomhex.jsonl",
		"misc1.treeonly":        "calc_misc1.jsonl",
		"misc1.noskills":        "calc_misc1.jsonl",
		"misc1.full":            "calc_misc1.jsonl",
		"misc2.treeonly":        "calc_misc2.jsonl",
		"misc2.noskills":        "calc_misc2.jsonl",
		"misc2.full":            "calc_misc2.jsonl",
		"misc3.treeonly":        "calc_misc3.jsonl",
		"misc3.noskills":        "calc_misc3.jsonl",
		"misc3.full":            "calc_misc3.jsonl",
	}
	checked := 0
	// MP_ONLY=<prefix> narrows the run to one build while diagnosing a
	// divergence, so an unrelated variant's guard panic cannot pre-empt it.
	only := os.Getenv("MP_ONLY")
	// MP_GUARDS=1 turns an unported-branch panic into a reported failure and
	// carries on to the next variant, so one run enumerates the whole guard
	// surface instead of stopping at the first one.
	collectGuards := os.Getenv("MP_GUARDS") != ""
	for variant, file := range variants {
		if only != "" && !strings.HasPrefix(variant, only) {
			continue
		}
		func() {
			if collectGuards {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s GUARD: %v", variant, r)
					}
				}()
			}
			checkCalcVariant(t, variant, file, &checked)
		}()
	}
	if checked < 25 && only == "" {
		t.Fatalf("expected 25 variants checked, got %d", checked)
	}
	// A narrowed run that matches nothing passes vacuously and reads as
	// green; fail it instead.
	if only != "" && checked == 0 {
		t.Fatalf("MP_ONLY=%q matched no variants", only)
	}
	t.Logf("calc initEnv vs archive: %d variants byte-identical", checked)
}

func checkCalcVariant(t *testing.T, variant, file string, checkedTotal *int) {
	{
		checked := 0
		defer func() { *checkedTotal += checked }()
		path := filepath.Join("testdata", file)
		if _, err := os.Stat(path); err != nil {
			t.Skipf("archive dump not present: %v", err)
		}
		var fixture, allocOrders, nodeOrders, grantedNodes, grantedAsc, ebItems, dbs, skills, skillLists string
		var performDbs, performOutput, performMinionDb, performMinionOutput string
		var defenceDbs, defenceOutput, defenceMinionDb, defenceMinionOutput string
		var ehpDbs, ehpOutput, ehpMinionDb, ehpMinionOutput string
		var offenceDbs, offenceOutput, offenceSkillOutput, offenceMinionDb, offenceMinionOutput string
		var globalCache, triggersDbs, triggersOutput, triggersSkillData string
		var triggersMinionOutput, triggersMinionSkillData string
		var mirageAllocOrders, mirageNodeOrders, mirage, mirageOutput string
		forEachCalcRecord(t, path, func(k, c string) {
			switch k {
			case variant + ".fixture":
				fixture = c
			case variant + ".allocOrders":
				allocOrders = c
			case variant + ".nodeOrders":
				nodeOrders = c
			case variant + ".grantedPassiveNodes":
				grantedNodes = c
			case variant + ".grantedAscendancyNodes":
				grantedAsc = c
			case variant + ".energyBladeItems":
				ebItems = c
			case variant + ".mirageAllocOrders":
				mirageAllocOrders = c
			case variant + ".mirageNodeOrders":
				mirageNodeOrders = c
			case variant + ".mirage":
				mirage = c
			case variant + ".mirageOutput":
				mirageOutput = c
			case variant + ".dbs":
				dbs = c
			case variant + ".skills":
				skills = c
			case variant + ".skillLists":
				skillLists = c
			case variant + ".performDbs":
				performDbs = c
			case variant + ".performOutput":
				performOutput = c
			case variant + ".performMinionDb":
				performMinionDb = c
			case variant + ".performMinionOutput":
				performMinionOutput = c
			case variant + ".defenceDbs":
				defenceDbs = c
			case variant + ".defenceOutput":
				defenceOutput = c
			case variant + ".defenceMinionDb":
				defenceMinionDb = c
			case variant + ".defenceMinionOutput":
				defenceMinionOutput = c
			case variant + ".ehpDbs":
				ehpDbs = c
			case variant + ".ehpOutput":
				ehpOutput = c
			case variant + ".ehpMinionDb":
				ehpMinionDb = c
			case variant + ".ehpMinionOutput":
				ehpMinionOutput = c
			case variant + ".offenceDbs":
				offenceDbs = c
			case variant + ".offenceOutput":
				offenceOutput = c
			case variant + ".offenceSkillOutput":
				offenceSkillOutput = c
			case variant + ".offenceMinionDb":
				offenceMinionDb = c
			case variant + ".offenceMinionOutput":
				offenceMinionOutput = c
			case variant + ".globalCache":
				globalCache = c
			case variant + ".triggersDbs":
				triggersDbs = c
			case variant + ".triggersOutput":
				triggersOutput = c
			case variant + ".triggersSkillData":
				triggersSkillData = c
			case variant + ".triggersMinionOutput":
				triggersMinionOutput = c
			case variant + ".triggersMinionSkillData":
				triggersMinionSkillData = c
			}
		})
		if fixture == "" || allocOrders == "" || nodeOrders == "" || grantedNodes == "" || grantedAsc == "" || ebItems == "" || dbs == "" || skills == "" || skillLists == "" {
			t.Fatalf("%s: missing records for %s", file, variant)
		}
		if performDbs == "" || performOutput == "" {
			t.Fatalf("%s: missing perform records for %s", file, variant)
		}
		if defenceDbs == "" || defenceOutput == "" {
			t.Fatalf("%s: missing defence records for %s", file, variant)
		}
		if ehpDbs == "" || ehpOutput == "" {
			t.Fatalf("%s: missing ehp records for %s", file, variant)
		}
		if offenceDbs == "" || offenceOutput == "" {
			t.Fatalf("%s: missing offence records for %s", file, variant)
		}
		if globalCache == "" || triggersDbs == "" || triggersOutput == "" {
			t.Fatalf("%s: missing trigger records for %s", file, variant)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(fixture), &m); err != nil {
			t.Fatal(err)
		}
		replay := &calc.ReplayInput{
			AllocOrders:            decodeAllocOrders(allocOrders),
			NodeOrders:             decodeAllocOrders(nodeOrders),
			GrantedPassiveNodes:    decodeGrantedPassiveNodes(grantedNodes),
			GrantedAscendancyNodes: decodeGrantedPassiveNodes(grantedAsc),
			EnergyBladeItems:       decodeEnergyBladeItems(ebItems),
			MirageAllocOrders:      decodeAllocOrders(mirageAllocOrders),
			MirageNodeOrders:       decodeAllocOrders(mirageNodeOrders),
		}
		in := decodeCalcFixture(m)
		// The native bridge: spec and item pool come from the natively
		// parsed build (MP_FIXTURE=1 reverts to the pure fixture replay
		// while diagnosing whether a divergence is native- or calc-side).
		if os.Getenv("MP_FIXTURE") == "" {
			buildKey := strings.TrimSuffix(strings.TrimPrefix(file, "calc_"), ".jsonl")
			applyNativeBuild(t, buildKey, in)
		}
		// GlobalCache is computed, not fed: calcs.buildOutput runs a whole
		// perform per active skill, and the reference's own snapshot is the
		// state that leaves behind. The staged replay below then overwrites
		// the main skill's entry with a pre-offence one, exactly as the
		// dump's manual perform does.
		if os.Getenv("MP_NODRIVER") == "" {
			replay.GlobalCache = calc.BuildOutput(in, "MAIN", replay).GlobalCache
		}
		env := calc.InitEnv(in, "MAIN", replay)
		// The checkpoint phase mirrors the dump's stubbed defence/offence
		// handoff, so nested performs (mirage sub-environments) stay
		// body-only exactly as the dump's did.
		env.StubHandoff = true
		got := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		checked++
		if got != dbs {
			t.Errorf("%s dbs diverged:\n%s", variant, diffWindow(got, dbs))
		}
		type skillShadow struct {
			Name    string  `lua:"name"`
			Id      string  `lua:"id"`
			Level   float64 `lua:"level"`
			Quality float64 `lua:"quality"`
			IsMain  bool    `lua:"isMain,omitempty"`
		}
		summaries := make([]skillShadow, len(env.PlayerActiveSkills))
		for i, as := range env.PlayerActiveSkills {
			summaries[i] = skillShadow{
				Name:    as.ActiveEffect.GrantedEffect.Name,
				Id:      as.ActiveEffect.GrantedEffect.Id,
				Level:   as.ActiveEffect.Level,
				Quality: as.ActiveEffect.Quality,
				IsMain:  env.PlayerMainSkill == as,
			}
		}
		if got := luacanon.Encode(summaries); got != skills {
			t.Errorf("%s skills diverged:\n%s", variant, diffWindow(got, skills))
		}
		shadows := make([]skillListShadow, len(env.PlayerActiveSkills))
		for i, as := range env.PlayerActiveSkills {
			shadows[i] = skillListShadowOf(env, as)
		}
		if got := luacanon.Encode(shadows); got != skillLists {
			t.Errorf("%s skillLists diverged:\n%s", variant, diffWindow(got, skillLists))
		}
		// Perform runs on the same env (mirroring the dump) and the
		// post-perform-body state is compared against the archive.
		env.Perform()
		gotPerform := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		if gotPerform != performDbs {
			t.Errorf("%s performDbs diverged:\n%s", variant, diffWindow(gotPerform, performDbs))
		}
		if got := luacanon.Encode(scalarsOnly(env.Player.Output)); got != performOutput {
			t.Errorf("%s performOutput diverged:\n%s", variant, diffWindow(got, performOutput))
		}
		if env.Minion != nil {
			if performMinionDb == "" || performMinionOutput == "" {
				t.Errorf("%s: Go perform produced a minion but the archive has no minion records", variant)
			} else {
				if got := luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}); got != performMinionDb {
					t.Errorf("%s performMinionDb diverged:\n%s", variant, diffWindow(got, performMinionDb))
				}
				if got := luacanon.Encode(scalarsOnly(env.Minion.Output)); got != performMinionOutput {
					t.Errorf("%s performMinionOutput diverged:\n%s", variant, diffWindow(got, performMinionOutput))
				}
			}
		} else if performMinionDb != "" {
			t.Errorf("%s: archive has minion perform records but Go produced no minion", variant)
		}
		// Perform mutates shared skill tag tables (warcryPowerBonus,
		// CalcPerform L2330) like the reference; the dump scrubs that
		// residue at each variant start, so scrub before the next variant
		// reuses the shared game data.
		// Defence stage, run on the post-perform-body state exactly as the
		// dump does (player then minion, back to back).
		env.RunDefence()
		gotDefence := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		if gotDefence != defenceDbs {
			t.Errorf("%s defenceDbs diverged:\n%s", variant, diffWindow(gotDefence, defenceDbs))
		}
		if got := luacanon.Encode(scalarsOnly(env.Player.Output)); got != defenceOutput {
			t.Errorf("%s defenceOutput diverged:\n%s", variant, diffWindow(got, defenceOutput))
		}
		if env.Minion != nil {
			if got := luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}); got != defenceMinionDb {
				t.Errorf("%s defenceMinionDb diverged:\n%s", variant, diffWindow(got, defenceMinionDb))
			}
			if got := luacanon.Encode(scalarsOnly(env.Minion.Output)); got != defenceMinionOutput {
				t.Errorf("%s defenceMinionOutput diverged:\n%s", variant, diffWindow(got, defenceMinionOutput))
			}
		}
		// EHP stage, on the post-defence state, player then minion.
		env.RunEHP()
		gotEHP := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		if gotEHP != ehpDbs {
			t.Errorf("%s ehpDbs diverged:\n%s", variant, diffWindow(gotEHP, ehpDbs))
		}
		if got := luacanon.Encode(scalarsOnly(env.Player.Output)); got != ehpOutput {
			t.Errorf("%s ehpOutput diverged:\n%s", variant, diffWindow(got, ehpOutput))
		}
		if env.Minion != nil {
			if got := luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}); got != ehpMinionDb {
				t.Errorf("%s ehpMinionDb diverged:\n%s", variant, diffWindow(got, ehpMinionDb))
			}
			if got := luacanon.Encode(scalarsOnly(env.Minion.Output)); got != ehpMinionOutput {
				t.Errorf("%s ehpMinionOutput diverged:\n%s", variant, diffWindow(got, ehpMinionOutput))
			}
		}
		// The cache the driver built, at the point the dump snapshots it.
		if out := os.Getenv("MP_DUMPGC"); out != "" {
			os.WriteFile(out, []byte(luacanon.Encode(cacheShadowOf(env.GlobalCache))), 0644)
			os.WriteFile(out+".want", []byte(globalCache), 0644)
		}
		if got := luacanon.Encode(cacheShadowOf(env.GlobalCache)); got != globalCache {
			t.Errorf("%s globalCache diverged:\n%s", variant, diffWindow(got, globalCache))
		}
		// Trigger stage, then the mirage gate and offence — the same
		// sequence and the same interleaving of checkpoints the dump uses
		// (CalcPerform L3726-3729).
		env.RunTriggersPlayer()
		gotTrig := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		if gotTrig != triggersDbs {
			t.Errorf("%s triggersDbs diverged:\n%s", variant, diffWindow(gotTrig, triggersDbs))
		}
		if got := luacanon.Encode(scalarsOnly(env.Player.Output)); got != triggersOutput {
			t.Errorf("%s triggersOutput diverged:\n%s", variant, diffWindow(got, triggersOutput))
		}
		if got := luacanon.Encode(scalarsOnly(env.PlayerMainSkill.SkillData)); got != triggersSkillData {
			t.Errorf("%s triggersSkillData diverged:\n%s", variant, diffWindow(got, triggersSkillData))
		}
		env.RunOffencePlayer()
		gotOffence := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		if gotOffence != offenceDbs {
			t.Errorf("%s offenceDbs diverged:\n%s", variant, diffWindow(gotOffence, offenceDbs))
		}
		if got := luacanon.Encode(scalarsOnly(env.Player.Output)); got != offenceOutput {
			t.Errorf("%s offenceOutput diverged:\n%s", variant, diffWindow(got, offenceOutput))
		}
		if got := luacanon.Encode(scalarsOnly(env.PlayerMainSkill.SkillData)); got != offenceSkillOutput {
			t.Errorf("%s offenceSkillOutput diverged:\n%s", variant, diffWindow(got, offenceSkillOutput))
		}
		if env.Minion != nil {
			env.RunTriggersMinion()
			if got := luacanon.Encode(scalarsOnly(env.Minion.Output)); got != triggersMinionOutput {
				t.Errorf("%s triggersMinionOutput diverged:\n%s", variant, diffWindow(got, triggersMinionOutput))
			}
			if got := luacanon.Encode(scalarsOnly(env.Minion.MainSkill.SkillData)); got != triggersMinionSkillData {
				t.Errorf("%s triggersMinionSkillData diverged:\n%s", variant, diffWindow(got, triggersMinionSkillData))
			}
			env.RunOffenceMinion()
			if got := luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}); got != offenceMinionDb {
				t.Errorf("%s offenceMinionDb diverged:\n%s", variant, diffWindow(got, offenceMinionDb))
			}
			if got := luacanon.Encode(scalarsOnly(env.Minion.Output)); got != offenceMinionOutput {
				t.Errorf("%s offenceMinionOutput diverged:\n%s", variant, diffWindow(got, offenceMinionOutput))
			}
		}
		// Mirage stage: the sub-environment CalcMirages builds. RunMirages
		// runs inside RunOffencePlayer, at the same point the dump emits
		// these.
		if mirage != "" {
			m := env.PlayerMainSkill.Mirage
			if m == nil {
				t.Errorf("%s mirage missing: archive has %s", variant, mirage)
			} else {
				shadow := map[string]any{
					"name":    m.Name,
					"count":   m.Count,
					"handled": env.MirageHandled,
				}
				if m.SkillPart != nil {
					shadow["skillPart"] = m.SkillPart
				}
				if m.SkillPartName != "" {
					shadow["skillPartName"] = m.SkillPartName
				}
				if got := luacanon.Encode(shadow); got != mirage {
					t.Errorf("%s mirage diverged:\n%s", variant, diffWindow(got, mirage))
				}
				if got := luacanon.Encode(scalarsOnly(m.Output)); got != mirageOutput {
					t.Errorf("%s mirageOutput diverged:\n%s", variant, diffWindow(got, mirageOutput))
				}
			}
		} else if env.PlayerMainSkill.Mirage != nil {
			t.Errorf("%s built a mirage the archive has none of", variant)
		}
		// Negative control: a corrupted input must stop matching.
		bad := decodeCalcFixture(m)
		bad.ConfigModList = bad.ConfigModList[1:]
		badEnv := calc.InitEnv(bad, "MAIN", &calc.ReplayInput{
			AllocOrders:            decodeAllocOrders(allocOrders),
			NodeOrders:             decodeAllocOrders(nodeOrders),
			GrantedPassiveNodes:    decodeGrantedPassiveNodes(grantedNodes),
			GrantedAscendancyNodes: decodeGrantedPassiveNodes(grantedAsc),
			EnergyBladeItems:       decodeEnergyBladeItems(ebItems),
		})
		badGot := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(badEnv.ModDB),
			Enemy: shadowOf(badEnv.EnemyDB),
			Item:  shadowOf(badEnv.ItemModDB),
		})
		if badGot == dbs {
			t.Errorf("%s: corrupted input still matched the archive dbs", variant)
		}
	}
}

// TestCalcFixtureEchoDetectsCorruption is the negative control: a mutated
// input must stop matching the archive canon.
func TestCalcFixtureEchoDetectsCorruption(t *testing.T) {
	path := filepath.Join("testdata", "calc_coc.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("archive dump not present: %v", err)
	}
	var fixture string
	forEachCalcRecord(t, path, func(k, c string) {
		if k == "coc.full.fixture" {
			fixture = c
		}
	})
	if fixture == "" {
		t.Fatal("coc.full.fixture record missing")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(fixture), &m); err != nil {
		t.Fatal(err)
	}
	corrupt := map[string]func(*calc.BuildInput){
		"characterLevel": func(in *calc.BuildInput) { in.CharacterLevel++ },
		"nodeModValue": func(in *calc.BuildInput) {
			for _, node := range in.Spec.AllocNodes {
				for _, mod := range node.ModList {
					if v, ok := mod.Value.(float64); ok {
						mod.Value = v + 1
						return
					}
				}
			}
			t.Fatal("no numeric node mod to corrupt")
		},
		"droppedConfigMod": func(in *calc.BuildInput) {
			in.ConfigModList = in.ConfigModList[1:]
		},
	}
	for name, mutate := range corrupt {
		in := decodeCalcFixture(m)
		mutate(in)
		if luacanon.EncodeExact(in) == fixture {
			t.Errorf("%s corruption not detected by the echo", name)
		}
	}
}

// cacheShadow mirrors dump_calc.lua's cacheState: the scalar headline fields
// cacheData stored, plus scalar slices taken THROUGH the entry's live env at
// snapshot time -- which is why they show stages that ran after the entry was
// cached.
type cacheShadow struct {
	Name                   string         `lua:"Name"`
	Speed                  *float64       `lua:"Speed"`
	HitSpeed               *float64       `lua:"HitSpeed"`
	ManaCost               *float64       `lua:"ManaCost"`
	LifeCost               *float64       `lua:"LifeCost"`
	ESCost                 *float64       `lua:"ESCost"`
	RageCost               *float64       `lua:"RageCost"`
	HitChance              *float64       `lua:"HitChance"`
	AccuracyHitChance      *float64       `lua:"AccuracyHitChance"`
	PreEffectiveCritChance *float64       `lua:"PreEffectiveCritChance"`
	CritChance             *float64       `lua:"CritChance"`
	TotalDPS               *float64       `lua:"TotalDPS"`
	Output                 map[string]any `lua:"output"`
	OutputMainHand         map[string]any `lua:"outputMainHand"`
	OutputOffHand          map[string]any `lua:"outputOffHand"`
	MainSkillData          map[string]any `lua:"mainSkillData"`
	ActiveSkillData        map[string]any `lua:"activeSkillData"`
}

func cacheShadowOf(cache map[string]*calc.CachedSkill) map[string]*cacheShadow {
	out := map[string]*cacheShadow{}
	for uuid, e := range cache {
		sh := &cacheShadow{
			Name: e.Name, Speed: e.Speed, HitSpeed: e.HitSpeed,
			ManaCost: e.ManaCost, LifeCost: e.LifeCost, ESCost: e.ESCost,
			RageCost: e.RageCost, HitChance: e.HitChance,
			AccuracyHitChance:      e.AccuracyHitChance,
			PreEffectiveCritChance: e.PreEffectiveCritChance,
			CritChance:             e.CritChance, TotalDPS: e.TotalDPS,
		}
		if env := e.Env; env != nil && env.Player != nil {
			po := env.Player.Output
			sub := func(k string) map[string]any {
				t, _ := po[k].(map[string]any)
				return scalarsOnly(t)
			}
			sh.Output = scalarsOnly(po)
			sh.OutputMainHand = sub("MainHand")
			sh.OutputOffHand = sub("OffHand")
			if env.PlayerMainSkill != nil {
				sh.MainSkillData = scalarsOnly(env.PlayerMainSkill.SkillData)
			}
		}
		if e.ActiveSkill != nil {
			sh.ActiveSkillData = scalarsOnly(e.ActiveSkill.SkillData)
		}
		out[uuid] = sh
	}
	return out
}
