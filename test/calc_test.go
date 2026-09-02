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
	"github.com/MissingL-tter/missingPassives/internal/util"
	calcitem "github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// buildDumpFiles lists the per-corpus archive dumps tools/dump_build.lua
// writes (one process per source build).
var buildDumpFiles = []string{"build_empty.jsonl", "build_coc.jsonl", "build_zombies.jsonl", "build_lowlife.jsonl", "build_spectre.jsonl", "build_cyclone.jsonl", "build_rf.jsonl", "build_holyrelic.jsonl", "build_eblade.jsonl"}

func decodeCalcModList(v any) []*modparser.Mod {
	m := v.(map[string]any)
	out := make([]*modparser.Mod, len(m))
	for i := 1; i <= len(m); i++ {
		out[i-1] = luacanon.ModFromTable(m[strconv.Itoa(i)].(map[string]any))
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

func optCalcNum(m map[string]any, k string) util.Opt[float64] {
	if v, ok := m[k]; ok {
		return util.Some(v.(float64))
	}
	return util.Opt[float64]{}
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

func decodeCalcGrantedSkills(v any) []calcitem.GrantedSkill {
	arr := decodeCalcArray(v)
	out := make([]calcitem.GrantedSkill, len(arr))
	for i, e := range arr {
		out[i] = luacanon.GrantedSkillFromTable(e.(map[string]any))
	}
	return out
}

func decodeCalcSockets(v any) []calc.SocketInput {
	arr := decodeCalcArray(v)
	out := make([]calc.SocketInput, len(arr))
	for i, e := range arr {
		m := e.(map[string]any)
		out[i] = calc.SocketInput{Color: m["color"].(string), Group: m["group"].(float64)}
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
		GrantedSkills:               decodeCalcGrantedSkills(m["grantedSkills"]),
		ExplicitLines:               decodeCalcStrings(m["explicitLines"]),
		OtherLines:                  decodeCalcStrings(m["otherLines"]),
	}
	item.Quality = optCalcFloat(m, "quality")
	if v, ok := m["base"]; ok {
		b := v.(map[string]any)
		item.Base = &calc.ItemBaseInput{SubType: optCalcString(b, "subType"), Type: optCalcString(b, "type")}
		if fv, ok := b["flask"]; ok {
			f := fv.(map[string]any)
			item.Base.Flask = &calc.FlaskBaseInput{Life: optCalcNum(f, "life"), Mana: optCalcNum(f, "mana")}
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
		item.Requirements = luacanon.RequirementsFromTable(v.(map[string]any))
	}
	if v, ok := m["sockets"]; ok {
		item.Sockets = decodeCalcSockets(v)
	}
	if v, ok := m["jewelData"]; ok {
		item.JewelData = luacanon.JewelDataFromTable(v.(map[string]any))
	}
	if v, ok := m["flaskData"]; ok {
		item.FlaskData = luacanon.FlaskDataFromTable(v.(map[string]any))
	}
	if v, ok := m["tinctureData"]; ok {
		item.TinctureData = luacanon.TinctureDataFromTable(v.(map[string]any))
	}
	if v, ok := m["armourData"]; ok {
		item.ArmourData = luacanon.ArmourDataFromTable(v.(map[string]any))
	}
	if v, ok := m["funcTypes"]; ok {
		item.FuncTypes = decodeCalcStrings(v)
	}
	if v, ok := m["weaponData"]; ok {
		item.WeaponData = map[int]*calcitem.WeaponData{}
		for idx, side := range v.(map[string]any) {
			n, err := strconv.Atoi(idx)
			if err != nil {
				panic("bad weaponData index " + idx)
			}
			item.WeaponData[n] = luacanon.WeaponDataFromTable(side.(map[string]any))
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
			slot.RadiusAttributes = decodeCalcCounts(av)
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
		node.KeystoneMod = luacanon.ModFromTable(kv.(map[string]any))
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
			SocketGroup: luacanon.SocketGroupFromTable(g["kv"].(map[string]any)),
			GemList:     []*calc.SocketGemInput{},
		}
		for _, gemv := range decodeCalcArray(g["gemList"]) {
			gm := gemv.(map[string]any)
			sg.GemList = append(sg.GemList, &calc.SocketGemInput{
				Gem:                 luacanon.GemInstanceFromTable(gm["kv"].(map[string]any)),
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
		ConfigInput:        luacanon.ConfigInputFromTable(c["configInput"].(map[string]any), false),
		ConfigPlaceholder:  luacanon.ConfigInputFromTable(c["configPlaceholder"].(map[string]any), true),
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
		// Mod lists in the archive carry the reference's coerced numeric tag
		// text; this is the one reference-side normalisation point for
		// every fixture read through here (calc, item, tree tests).
		fn(rec.K, luacanon.NormalizeArchiveMods(rec.C))
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
	for _, name := range buildDumpFiles {
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
			if got := luacanon.EncodeExact(in); !luacanon.SameCanon(got, c) {
				t.Errorf("%s %s: echo diverged\n%s", name, k, diffWindow(got, c))
			}
		})
	}
	if fixtures < 4 {
		t.Fatalf("expected at least 4 fixture records, saw %d", fixtures)
	}
	t.Logf("calc fixture echo: %d fixtures byte-identical", fixtures)
}

// dbShadow re-canonicalises one DB the way dump_build.lua's dbState does.
type dbShadow struct {
	Mods        map[string][]*modparser.Mod `lua:"mods"`
	Conditions  modstore.Conditions         `lua:"conditions"`
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

// cfgScalars mirrors dump_build.lua's scalars() over a skill Cfg.
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
	if cfg.SkillPart.Set {
		m["skillPart"] = cfg.SkillPart.V
	}
	return m
}

// weaponShadow is a minion weapon table's scalar projection ({} when the
// minion has none).
func weaponShadow(wd *calcitem.WeaponData) map[string]any {
	if wd == nil {
		return map[string]any{}
	}
	return luacanon.WeaponDataTable(wd)
}

func nonNilMods(mods []*modparser.Mod) []*modparser.Mod {
	if mods == nil {
		return []*modparser.Mod{}
	}
	return mods
}

type skillListShadow struct {
	ModList           []*modparser.Mod   `lua:"modList"`
	Cfg               map[string]any     `lua:"cfg"`
	Flags             map[string]bool    `lua:"flags"`
	Data              *calc.SkillData    `lua:"data"`
	Buffs             []map[string]any   `lua:"buffs"`
	Weapon1Flags      *modparser.ModFlag `lua:"weapon1Flags"`
	Weapon2Flags      *modparser.ModFlag `lua:"weapon2Flags"`
	Weapon1Cfg        map[string]any     `lua:"weapon1Cfg"`
	Weapon1Cond       map[string]bool    `lua:"weapon1Cond"`
	Weapon2Cfg        map[string]any     `lua:"weapon2Cfg"`
	Weapon2Cond       map[string]bool    `lua:"weapon2Cond"`
	DisableReason     string             `lua:"disableReason,omitempty"`
	SkillPartName     string             `lua:"skillPartName,omitempty"`
	SkillTotemId      *float64           `lua:"skillTotemId"`
	ExtraSkillModList []*modparser.Mod   `lua:"extraSkillModList"`
	MinionList        []string           `lua:"minionList"`
	Minion            map[string]any     `lua:"minion"`
}

func skillListShadowOf(env *calc.Env, as *calc.ActiveSkill) skillListShadow {
	sh := skillListShadow{
		ModList:           nonNilMods(as.SkillModList.Mods),
		Cfg:               cfgScalars(as.SkillCfg),
		Flags:             as.SkillFlags,
		Data:              as.SkillData,
		Buffs:             []map[string]any{},
		Weapon1Flags:      as.Weapon1Flags,
		Weapon2Flags:      as.Weapon2Flags,
		DisableReason:     as.DisableReason,
		SkillPartName:     as.SkillPartName,
		SkillTotemId:      as.SkillTotemId,
		ExtraSkillModList: nonNilMods(as.ExtraSkillModList),
	}
	for _, buff := range as.BuffListTyped {
		b := luacanon.BuffTable(buff)
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
			"weaponData1": weaponShadow(as.Minion.WeaponData1),
			"weaponData2": weaponShadow(as.Minion.WeaponData2),
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

// assertOrdersConsistent checks the shape of the dump's recorded pairs()
// orders: each nodeOrders sequence begins with its allocOrders walk and
// continues with the extra-radius nodes. It says nothing about the order
// WITHIN a walk. The dump records the reference's real numeric-key order
// (LuaJIT's, which is stable per process and not ascending - it used to be
// sorted by the harness, tools/dump_build.lua:49); production walks ascending
// because some fixed order is needed and none observable depends on which
// (knowledge.md 4.6). The two orders differ and are compared as multisets.
func assertOrdersConsistent(t *testing.T, label string, allocOrders, nodeOrders [][]int) {
	t.Helper()
	for i, seq := range nodeOrders {
		if i >= len(allocOrders) {
			break
		}
		alloc := allocOrders[i]
		if len(seq) < len(alloc) {
			t.Fatalf("%s: nodeOrders[%d] shorter than its alloc order", label, i)
		}
		for j, id := range alloc {
			if seq[j] != id {
				t.Fatalf("%s: nodeOrders[%d][%d]=%d != allocOrders' %d", label, i, j, seq[j], id)
			}
		}
	}
}

// fixturePassives resolves granted passives from the dumped name->node
// maps, for a replay that is not reading a native spec.
type fixturePassives struct {
	passive    map[string]*calc.NodeInput
	ascendancy map[string]*calc.NodeInput
}

func (f *fixturePassives) GrantedPassive(name string) *calc.NodeInput {
	return f.passive[name]
}

func (f *fixturePassives) GrantedAscendancyNode(name string) *calc.NodeInput {
	return f.ascendancy[name]
}

// assertEnergyBlades compares the weapons calc synthesized for an Energy
// Blade build against the ones the archive recorded. The dump captured
// them because the port used to take them from the fixture; calc builds
// them through the item machinery now, so this is the check that the
// construction agrees.
func assertEnergyBlades(t *testing.T, variant string, env *calc.Env, want map[string]*calc.ItemInput) {
	t.Helper()
	for _, slot := range []string{"Weapon 1", "Weapon 2"} {
		refItem := want[slot]
		var gotItem *calc.ItemInput
		if it, ok := env.Player.ItemList[slot].(*calc.Item); ok && it != nil &&
			strings.HasPrefix(it.In.Name, "Energy Blade") {
			gotItem = it.In
		}
		switch {
		case refItem == nil && gotItem == nil:
		case refItem == nil:
			t.Errorf("%s %s: synthesized an Energy Blade the archive has none of", variant, slot)
		case gotItem == nil:
			t.Errorf("%s %s: archive has an Energy Blade, calc synthesized none", variant, slot)
		default:
			if got, wantCanon := luacanon.EncodeExact(gotItem), luacanon.EncodeExact(refItem); !luacanon.SameCanon(got, wantCanon) {
				t.Errorf("%s %s Energy Blade diverged\n%s", variant, slot, diffWindow(got, wantCanon))
			}
		}
	}
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
	loadData(t)
	variants := map[string]string{ // variant -> dump file
		"empty":                 "build_empty.jsonl",
		"coc.treeonly":          "build_coc.jsonl",
		"coc.noskills":          "build_coc.jsonl",
		"coc.full":              "build_coc.jsonl",
		"zombies.treeonly":      "build_zombies.jsonl",
		"zombies.noskills":      "build_zombies.jsonl",
		"zombies.full":          "build_zombies.jsonl",
		"lowlife.treeonly":      "build_lowlife.jsonl",
		"lowlife.noskills":      "build_lowlife.jsonl",
		"lowlife.full":          "build_lowlife.jsonl",
		"spectre.treeonly":      "build_spectre.jsonl",
		"spectre.noskills":      "build_spectre.jsonl",
		"spectre.full":          "build_spectre.jsonl",
		"cyclone.treeonly":      "build_cyclone.jsonl",
		"cyclone.noskills":      "build_cyclone.jsonl",
		"cyclone.full":          "build_cyclone.jsonl",
		"rf.treeonly":           "build_rf.jsonl",
		"rf.noskills":           "build_rf.jsonl",
		"rf.full":               "build_rf.jsonl",
		"holyrelic.treeonly":    "build_holyrelic.jsonl",
		"holyrelic.noskills":    "build_holyrelic.jsonl",
		"holyrelic.full":        "build_holyrelic.jsonl",
		"eblade.treeonly":       "build_eblade.jsonl",
		"eblade.noskills":       "build_eblade.jsonl",
		"eblade.full":           "build_eblade.jsonl",
		"cocuser.treeonly":      "build_cocuser.jsonl",
		"cocuser.noskills":      "build_cocuser.jsonl",
		"cocuser.full":          "build_cocuser.jsonl",
		"dualstrike.treeonly":   "build_dualstrike.jsonl",
		"dualstrike.noskills":   "build_dualstrike.jsonl",
		"dualstrike.full":       "build_dualstrike.jsonl",
		"bfbb.treeonly":         "build_bfbb.jsonl",
		"bfbb.noskills":         "build_bfbb.jsonl",
		"bfbb.full":             "build_bfbb.jsonl",
		"ballista.treeonly":     "build_ballista.jsonl",
		"ballista.noskills":     "build_ballista.jsonl",
		"ballista.full":         "build_ballista.jsonl",
		"trap.treeonly":         "build_trap.jsonl",
		"trap.noskills":         "build_trap.jsonl",
		"trap.full":             "build_trap.jsonl",
		"mirage.treeonly":       "build_mirage.jsonl",
		"mirage.noskills":       "build_mirage.jsonl",
		"mirage.full":           "build_mirage.jsonl",
		"exparrow.treeonly":     "build_exparrow.jsonl",
		"exparrow.noskills":     "build_exparrow.jsonl",
		"exparrow.full":         "build_exparrow.jsonl",
		"blight.treeonly":       "build_blight.jsonl",
		"blight.noskills":       "build_blight.jsonl",
		"blight.full":           "build_blight.jsonl",
		"cwc.treeonly":          "build_cwc.jsonl",
		"cwc.noskills":          "build_cwc.jsonl",
		"cwc.full":              "build_cwc.jsonl",
		"poetpen.treeonly":      "build_poetpen.jsonl",
		"poetpen.noskills":      "build_poetpen.jsonl",
		"poetpen.full":          "build_poetpen.jsonl",
		"cospri.treeonly":       "build_cospri.jsonl",
		"cospri.noskills":       "build_cospri.jsonl",
		"cospri.full":           "build_cospri.jsonl",
		"saviour.treeonly":      "build_saviour.jsonl",
		"saviour.noskills":      "build_saviour.jsonl",
		"saviour.full":          "build_saviour.jsonl",
		"slinger.treeonly":      "build_slinger.jsonl",
		"slinger.noskills":      "build_slinger.jsonl",
		"slinger.full":          "build_slinger.jsonl",
		"mine.treeonly":         "build_mine.jsonl",
		"mine.noskills":         "build_mine.jsonl",
		"mine.full":             "build_mine.jsonl",
		"brand.treeonly":        "build_brand.jsonl",
		"brand.noskills":        "build_brand.jsonl",
		"brand.full":            "build_brand.jsonl",
		"absolution.treeonly":   "build_absolution.jsonl",
		"absolution.noskills":   "build_absolution.jsonl",
		"absolution.full":       "build_absolution.jsonl",
		"corrupting.treeonly":   "build_corrupting.jsonl",
		"corrupting.noskills":   "build_corrupting.jsonl",
		"corrupting.full":       "build_corrupting.jsonl",
		"fissure.treeonly":      "build_fissure.jsonl",
		"fissure.noskills":      "build_fissure.jsonl",
		"fissure.full":          "build_fissure.jsonl",
		"tornado.treeonly":      "build_tornado.jsonl",
		"tornado.noskills":      "build_tornado.jsonl",
		"tornado.full":          "build_tornado.jsonl",
		"toxicrain.treeonly":    "build_toxicrain.jsonl",
		"toxicrain.noskills":    "build_toxicrain.jsonl",
		"toxicrain.full":        "build_toxicrain.jsonl",
		"moltenstrike.treeonly": "build_moltenstrike.jsonl",
		"moltenstrike.noskills": "build_moltenstrike.jsonl",
		"moltenstrike.full":     "build_moltenstrike.jsonl",
		"earthquake.treeonly":   "build_earthquake.jsonl",
		"earthquake.noskills":   "build_earthquake.jsonl",
		"earthquake.full":       "build_earthquake.jsonl",
		"arctotem.treeonly":     "build_arctotem.jsonl",
		"arctotem.noskills":     "build_arctotem.jsonl",
		"arctotem.full":         "build_arctotem.jsonl",
		"doomblast.treeonly":    "build_doomblast.jsonl",
		"doomblast.noskills":    "build_doomblast.jsonl",
		"doomblast.full":        "build_doomblast.jsonl",
		"callpyre.treeonly":     "build_callpyre.jsonl",
		"callpyre.noskills":     "build_callpyre.jsonl",
		"callpyre.full":         "build_callpyre.jsonl",
		"tempest.treeonly":      "build_tempest.jsonl",
		"tempest.noskills":      "build_tempest.jsonl",
		"tempest.full":          "build_tempest.jsonl",
		"voidstorm.treeonly":    "build_voidstorm.jsonl",
		"voidstorm.noskills":    "build_voidstorm.jsonl",
		"voidstorm.full":        "build_voidstorm.jsonl",
		"shockwave.treeonly":    "build_shockwave.jsonl",
		"shockwave.noskills":    "build_shockwave.jsonl",
		"shockwave.full":        "build_shockwave.jsonl",
		"toad.treeonly":         "build_toad.jsonl",
		"toad.noskills":         "build_toad.jsonl",
		"toad.full":             "build_toad.jsonl",
		"trig1.treeonly":        "build_trig1.jsonl",
		"trig1.noskills":        "build_trig1.jsonl",
		"trig1.full":            "build_trig1.jsonl",
		"trig2.treeonly":        "build_trig2.jsonl",
		"trig2.noskills":        "build_trig2.jsonl",
		"trig2.full":            "build_trig2.jsonl",
		"trig3.treeonly":        "build_trig3.jsonl",
		"trig3.noskills":        "build_trig3.jsonl",
		"trig3.full":            "build_trig3.jsonl",
		"trig4.treeonly":        "build_trig4.jsonl",
		"trig4.noskills":        "build_trig4.jsonl",
		"trig4.full":            "build_trig4.jsonl",
		"mjolner.treeonly":      "build_mjolner.jsonl",
		"mjolner.noskills":      "build_mjolner.jsonl",
		"mjolner.full":          "build_mjolner.jsonl",
		"stages.treeonly":       "build_stages.jsonl",
		"stages.noskills":       "build_stages.jsonl",
		"stages.full":           "build_stages.jsonl",
		"doomexp.treeonly":      "build_doomexp.jsonl",
		"doomexp.noskills":      "build_doomexp.jsonl",
		"doomexp.full":          "build_doomexp.jsonl",
		"doomhex.treeonly":      "build_doomhex.jsonl",
		"doomhex.noskills":      "build_doomhex.jsonl",
		"doomhex.full":          "build_doomhex.jsonl",
		"misc1.treeonly":        "build_misc1.jsonl",
		"misc1.noskills":        "build_misc1.jsonl",
		"misc1.full":            "build_misc1.jsonl",
		"misc2.treeonly":        "build_misc2.jsonl",
		"misc2.noskills":        "build_misc2.jsonl",
		"misc2.full":            "build_misc2.jsonl",
		"misc3.treeonly":        "build_misc3.jsonl",
		"misc3.noskills":        "build_misc3.jsonl",
		"misc3.full":            "build_misc3.jsonl",
	}
	checked := 0
	// MP_ONLY=<prefix> narrows the run to one build while diagnosing a
	// divergence, so an unrelated variant's guard panic cannot pre-empt it.
	toleratedValues = 0
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
	if toleratedValues > 0 {
		t.Logf("calc initEnv vs archive: %d variants agree, %d values only once quantized to %d significant figures (see 4.7)",
			checked, toleratedValues, luacanon.CompareDigits)
	} else {
		t.Logf("calc initEnv vs archive: %d variants byte-identical", checked)
	}
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
		// The recorded orders are not replayed: production walks ascending
		// ids, the dump holds the reference's own order, and the state they
		// produce is compared as a multiset. Only the recording's shape is
		// checked here.
		assertOrdersConsistent(t, variant, decodeAllocOrders(allocOrders), decodeAllocOrders(nodeOrders))
		assertOrdersConsistent(t, variant+" (mirage)", decodeAllocOrders(mirageAllocOrders), decodeAllocOrders(mirageNodeOrders))
		replay := &calc.ReplayInput{}
		in := decodeCalcFixture(m)
		// The dumped name->node maps back the lookup for a pure fixture
		// replay; the native bridge below installs the derived one.
		in.Spec.Passives = &fixturePassives{
			passive:    decodeGrantedPassiveNodes(grantedNodes),
			ascendancy: decodeGrantedPassiveNodes(grantedAsc),
		}
		// The native bridge: spec and item pool come from the natively
		// parsed build (MP_FIXTURE=1 reverts to the pure fixture replay
		// while diagnosing whether a divergence is native- or calc-side).
		if os.Getenv("MP_FIXTURE") == "" {
			buildKey := strings.TrimSuffix(strings.TrimPrefix(file, "build_"), ".jsonl")
			applyNativeBuild(t, buildKey, variant, in)
		}
		// GlobalCache is computed, not fed: calcs.buildOutput runs a whole
		// perform per active skill, and the reference's own snapshot is the
		// state that leaves behind. The staged replay below then overwrites
		// the main skill's entry with a pre-offence one, exactly as the
		// dump's manual perform does.
		if os.Getenv("MP_NODRIVER") == "" {
			replay.GlobalCache = calc.BuildOutput(in, "MAIN", replay).GlobalCache
		}
		// The checkpoint phase mirrors the dump's stubbed defence/offence
		// handoff, so nested performs (mirage sub-environments) stay
		// body-only exactly as the dump's did.
		replay.StubHandoff = true
		env := calc.InitEnv(in, "MAIN", replay)
		assertEnergyBlades(t, variant, env, decodeEnergyBladeItems(ebItems))
		got := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		checked++
		canonDiverged(t, variant+" dbs", got, dbs)
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
		canonDiverged(t, variant+" skills", luacanon.Encode(summaries), skills)
		shadows := make([]skillListShadow, len(env.PlayerActiveSkills))
		for i, as := range env.PlayerActiveSkills {
			shadows[i] = skillListShadowOf(env, as)
		}
		canonDiverged(t, variant+" skillLists", luacanon.Encode(shadows), skillLists)
		// Perform runs on the same env (mirroring the dump) and the
		// post-perform-body state is compared against the archive.
		env.Perform()
		gotPerform := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		canonDiverged(t, variant+" performDbs", gotPerform, performDbs)
		canonDiverged(t, variant+" performOutput", luacanon.Encode(env.Player.Output), performOutput)
		if env.Minion != nil {
			if performMinionDb == "" || performMinionOutput == "" {
				t.Errorf("%s: Go perform produced a minion but the archive has no minion records", variant)
			} else {
				canonDiverged(t, variant+" performMinionDb", luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}), performMinionDb)
				canonDiverged(t, variant+" performMinionOutput", luacanon.Encode(env.Minion.Output), performMinionOutput)
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
		canonDiverged(t, variant+" defenceDbs", gotDefence, defenceDbs)
		canonDiverged(t, variant+" defenceOutput", luacanon.Encode(env.Player.Output), defenceOutput)
		if env.Minion != nil {
			canonDiverged(t, variant+" defenceMinionDb", luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}), defenceMinionDb)
			canonDiverged(t, variant+" defenceMinionOutput", luacanon.Encode(env.Minion.Output), defenceMinionOutput)
		}
		// EHP stage, on the post-defence state, player then minion.
		env.RunEHP()
		gotEHP := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		canonDiverged(t, variant+" ehpDbs", gotEHP, ehpDbs)
		canonDiverged(t, variant+" ehpOutput", luacanon.Encode(env.Player.Output), ehpOutput)
		if env.Minion != nil {
			canonDiverged(t, variant+" ehpMinionDb", luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}), ehpMinionDb)
			canonDiverged(t, variant+" ehpMinionOutput", luacanon.Encode(env.Minion.Output), ehpMinionOutput)
		}
		// The cache the driver built, at the point the dump snapshots it.
		if out := os.Getenv("MP_DUMPGC"); out != "" {
			os.WriteFile(out, []byte(luacanon.Encode(cacheShadowOf(env.GlobalCache))), 0644)
			os.WriteFile(out+".want", []byte(globalCache), 0644)
		}
		canonDiverged(t, variant+" globalCache", luacanon.Encode(cacheShadowOf(env.GlobalCache)), globalCache)
		// Trigger stage, then the mirage gate and offence — the same
		// sequence and the same interleaving of checkpoints the dump uses
		// (CalcPerform L3726-3729).
		env.RunTriggersPlayer()
		gotTrig := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		canonDiverged(t, variant+" triggersDbs", gotTrig, triggersDbs)
		canonDiverged(t, variant+" triggersOutput", luacanon.Encode(env.Player.Output), triggersOutput)
		canonDiverged(t, variant+" triggersSkillData", luacanon.Encode(env.PlayerMainSkill.SkillData), triggersSkillData)
		env.RunOffencePlayer()
		gotOffence := luacanon.Encode(dbsShadow{
			Mod:   shadowOf(env.ModDB),
			Enemy: shadowOf(env.EnemyDB),
			Item:  shadowOf(env.ItemModDB),
		})
		canonDiverged(t, variant+" offenceDbs", gotOffence, offenceDbs)
		canonDiverged(t, variant+" offenceOutput", luacanon.Encode(env.Player.Output), offenceOutput)
		canonDiverged(t, variant+" offenceSkillOutput", luacanon.Encode(env.PlayerMainSkill.SkillData), offenceSkillOutput)
		if env.Minion != nil {
			env.RunTriggersMinion()
			canonDiverged(t, variant+" triggersMinionOutput", luacanon.Encode(env.Minion.Output), triggersMinionOutput)
			canonDiverged(t, variant+" triggersMinionSkillData", luacanon.Encode(env.Minion.MainSkill.SkillData), triggersMinionSkillData)
			env.RunOffenceMinion()
			canonDiverged(t, variant+" offenceMinionDb", luacanon.Encode(dbShadow{Mods: env.Minion.DB.Mods, Conditions: env.Minion.DB.Conditions, Multipliers: env.Minion.DB.Multipliers}), offenceMinionDb)
			canonDiverged(t, variant+" offenceMinionOutput", luacanon.Encode(env.Minion.Output), offenceMinionOutput)
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
				if m.SkillPart.Set {
					shadow["skillPart"] = m.SkillPart.V
				}
				if m.SkillPartName != "" {
					shadow["skillPartName"] = m.SkillPartName
				}
				canonDiverged(t, variant+" mirage", luacanon.Encode(shadow), mirage)
				canonDiverged(t, variant+" mirageOutput", luacanon.Encode(m.Output), mirageOutput)
			}
		} else if env.PlayerMainSkill.Mirage != nil {
			t.Errorf("%s built a mirage the archive has none of", variant)
		}
		// Negative control: a corrupted input must stop matching.
		bad := decodeCalcFixture(m)
		bad.ConfigModList = bad.ConfigModList[1:]
		bad.Spec.Passives = &fixturePassives{
			passive:    decodeGrantedPassiveNodes(grantedNodes),
			ascendancy: decodeGrantedPassiveNodes(grantedAsc),
		}
		badEnv := calc.InitEnv(bad, "MAIN", &calc.ReplayInput{})
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
	path := filepath.Join("testdata", "build_coc.jsonl")
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
					if v, ok := mod.Value.(modparser.Num); ok {
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

// cacheShadow mirrors dump_build.lua's cacheState: the scalar headline fields
// cacheData stored, plus scalar slices taken THROUGH the entry's live env at
// snapshot time -- which is why they show stages that ran after the entry was
// cached.
type cacheShadow struct {
	Name                   string          `lua:"Name"`
	Speed                  *float64        `lua:"Speed"`
	HitSpeed               *float64        `lua:"HitSpeed"`
	ManaCost               *float64        `lua:"ManaCost"`
	LifeCost               *float64        `lua:"LifeCost"`
	ESCost                 *float64        `lua:"ESCost"`
	RageCost               *float64        `lua:"RageCost"`
	HitChance              *float64        `lua:"HitChance"`
	AccuracyHitChance      *float64        `lua:"AccuracyHitChance"`
	PreEffectiveCritChance *float64        `lua:"PreEffectiveCritChance"`
	CritChance             *float64        `lua:"CritChance"`
	TotalDPS               *float64        `lua:"TotalDPS"`
	Output                 modstore.Output `lua:"output"`
	OutputMainHand         modstore.Output `lua:"outputMainHand"`
	OutputOffHand          modstore.Output `lua:"outputOffHand"`
	MainSkillData          *calc.SkillData `lua:"mainSkillData"`
	ActiveSkillData        *calc.SkillData `lua:"activeSkillData"`
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
			sh.Output = env.Player.Output
			// absent weapon-pass tables render as {} (dump-side scalars(nil))
			sh.OutputMainHand, sh.OutputOffHand = env.PlayerWeaponOutputs()
			if sh.OutputMainHand == nil {
				sh.OutputMainHand = modstore.Output{}
			}
			if sh.OutputOffHand == nil {
				sh.OutputOffHand = modstore.Output{}
			}
			if env.PlayerMainSkill != nil {
				sh.MainSkillData = env.PlayerMainSkill.SkillData
			}
		}
		if e.ActiveSkill != nil {
			sh.ActiveSkillData = e.ActiveSkill.SkillData
		}
		out[uuid] = sh
	}
	return out
}

// toleratedValues counts numeric leaves that matched only once quantized
// to luacanon.CompareDigits, so a pass reports how much last-digit drift it
// absorbed rather than hiding it.
var toleratedValues int

// canonDiverged compares one checkpoint's canonical encoding against the
// archive's. Identical text passes outright; otherwise the two are walked
// leaf by leaf, numbers agreeing to luacanon.CompareDigits significant
// figures and everything else exactly. Reports true when it logged a
// failure.
func canonDiverged(t *testing.T, label, got, want string) bool {
	t.Helper()
	if got == want {
		return false
	}
	diffs, tolerated, err := luacanon.EqualWithin(got, want)
	if err != nil {
		t.Errorf("%s diverged, and the canon would not parse (%v):\n%s", label, err, diffWindow(got, want))
		return true
	}
	if len(diffs) > 0 {
		t.Errorf("%s diverged:%s", label, luacanon.FormatDiffs(diffs, 8))
		return true
	}
	toleratedValues += tolerated
	return false
}
