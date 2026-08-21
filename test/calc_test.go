package test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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
func decodeCalcArray(v any) []any {
	m := v.(map[string]any)
	out := make([]any, len(m))
	for i := 1; i <= len(m); i++ {
		out[i-1] = m[strconv.Itoa(i)]
	}
	return out
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
	if v, ok := m["base"]; ok {
		b := v.(map[string]any)
		item.Base = &calc.ItemBaseInput{SubType: optCalcString(b, "subType"), Type: optCalcString(b, "type")}
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
		ClassID:         c["classId"].(float64),
		CurClassName:    c["curClassName"].(string),
		TreeVersion:     c["treeVersion"].(string),
		MainSocketGroup: c["mainSocketGroup"].(float64),
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
			if got := luacanon.Encode(in); got != c {
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
	d := loadGameData(t)
	variants := map[string]string{ // variant -> dump file
		"empty":            "calc_empty.jsonl",
		"coc.treeonly":     "calc_coc.jsonl",
		"coc.noskills":     "calc_coc.jsonl",
		"coc.full":         "calc_coc.jsonl",
		"zombies.treeonly": "calc_zombies.jsonl",
		"zombies.noskills": "calc_zombies.jsonl",
		"zombies.full":     "calc_zombies.jsonl",
		"lowlife.treeonly": "calc_lowlife.jsonl",
		"lowlife.noskills": "calc_lowlife.jsonl",
		"lowlife.full":     "calc_lowlife.jsonl",
		"spectre.treeonly": "calc_spectre.jsonl",
		"spectre.noskills": "calc_spectre.jsonl",
		"spectre.full":     "calc_spectre.jsonl",
		"cyclone.treeonly": "calc_cyclone.jsonl",
		"cyclone.noskills": "calc_cyclone.jsonl",
		"cyclone.full":     "calc_cyclone.jsonl",
		"rf.treeonly":      "calc_rf.jsonl",
		"rf.noskills":      "calc_rf.jsonl",
		"rf.full":          "calc_rf.jsonl",
		"holyrelic.treeonly": "calc_holyrelic.jsonl",
		"holyrelic.noskills": "calc_holyrelic.jsonl",
		"holyrelic.full":     "calc_holyrelic.jsonl",
		"eblade.treeonly":    "calc_eblade.jsonl",
		"eblade.noskills":    "calc_eblade.jsonl",
		"eblade.full":        "calc_eblade.jsonl",
	}
	checked := 0
	for variant, file := range variants {
		path := filepath.Join("testdata", file)
		if _, err := os.Stat(path); err != nil {
			t.Skipf("archive dump not present: %v", err)
		}
		var fixture, allocOrders, nodeOrders, grantedNodes, grantedAsc, ebItems, dbs, skills, skillLists string
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
			case variant + ".dbs":
				dbs = c
			case variant + ".skills":
				skills = c
			case variant + ".skillLists":
				skillLists = c
			}
		})
		if fixture == "" || allocOrders == "" || nodeOrders == "" || grantedNodes == "" || grantedAsc == "" || ebItems == "" || dbs == "" || skills == "" || skillLists == "" {
			t.Fatalf("%s: missing records for %s", file, variant)
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
		}
		in := decodeCalcFixture(m)
		env := calc.InitEnv(d, in, "MAIN", replay)
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
		// Negative control: a corrupted input must stop matching.
		bad := decodeCalcFixture(m)
		bad.ConfigModList = bad.ConfigModList[1:]
		badEnv := calc.InitEnv(d, bad, "MAIN", &calc.ReplayInput{
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
	if checked < 25 {
		t.Fatalf("expected 25 variants checked, got %d", checked)
	}
	t.Logf("calc initEnv vs archive: %d variants byte-identical", checked)
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
		if luacanon.Encode(in) == fixture {
			t.Errorf("%s corruption not detected by the echo", name)
		}
	}
}
