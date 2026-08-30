package test

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// The mod-store differential test: replays tools/dump_modstore.lua's world —
// the whole parsed corpus distributed over a store tree with fixture actors,
// multipliers, conditions, items and configs — and compares every recorded
// aggregation result. Fails on any disagreement.
//
// Regenerate from .archive/src/ with: luajit ../../tools/dump_modstore.lua

type msItem struct {
	name, itype, rarity      string
	corrupted, shaper, elder bool
	fms                      map[string]bool
}

func (i *msItem) Name() string     { return i.name }
func (i *msItem) ItemType() string { return i.itype }
func (i *msItem) Rarity() string   { return i.rarity }
func (i *msItem) Corrupted() bool  { return i.corrupted }
func (i *msItem) Shaper() bool     { return i.shaper }
func (i *msItem) Elder() bool      { return i.elder }
func (i *msItem) FindModifierSubstring(sub, slot string) bool {
	return i.fms[sub+"|"+slot]
}

type msGem struct{ tags map[string]bool }

func (g *msGem) IsType(keyword string) bool { return g.tags[toLower(keyword)] }

// msWeapon is the fixture weapon-data table (countsAsAll1H plus Added* keys).
type msWeapon struct {
	all1H bool
	added map[string]bool
}

func (w *msWeapon) CountsAsAll1H() bool { return w.all1H }
func (w *msWeapon) AddedCond(cond string) (added, present bool) {
	added, present = w.added[cond]
	return
}

// msResolver serves the archive dump's gem-name → game-id table.
type msResolver struct{ gameIds map[string]string }

func (r *msResolver) GetGameIdFromGemName(name string, includeTransfigured bool) (string, bool) {
	id, ok := r.gameIds[toLower(name)]
	return id, ok
}

// msMurmur is Common.lua's murmurHash2 (verified against the archive by
// the export archive dump's tradeHashes).
func msMurmur(key []byte, seed uint32) uint32 {
	const m = 0x5bd1e995
	h := seed ^ uint32(len(key))
	for ; len(key) >= 4; key = key[4:] {
		k := binary.LittleEndian.Uint32(key)
		k *= m
		k ^= k >> 24
		k *= m
		h *= m
		h ^= k
	}
	if len(key) > 0 {
		var buf [4]byte
		copy(buf[:], key)
		h ^= binary.LittleEndian.Uint32(buf[:])
		h *= m
	}
	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}

func msHash(s string) string {
	return fmt.Sprintf("%d.%d", msMurmur([]byte(s), 0x9747b28c), msMurmur([]byte(s), 0x2312233))
}

func f17(v float64) string {
	return strconv.FormatFloat(v, 'g', 17, 64)
}

// numEq compares a %.17g archive dump string against a Go float exactly.
func numEq(fromArchive string, got float64) bool {
	want, err := strconv.ParseFloat(fromArchive, 64)
	if err != nil {
		return false
	}
	return want == got
}

// tryNum runs an aggregation, capturing reference-matching panics as "!".
func tryNum(fn func() float64) (v float64, errSentinel bool) {
	defer func() {
		if recover() != nil {
			errSentinel = true
		}
	}()
	return fn(), false
}

func tryAny(fn func() any) (v any, errSentinel bool) {
	defer func() {
		if recover() != nil {
			errSentinel = true
		}
	}()
	return fn(), false
}

type msCfgRec struct {
	Flags            *modparser.ModFlag     `json:"flags"`
	KeywordFlags     *modparser.KeywordFlag `json:"keywordFlags"`
	Source           string                 `json:"source"`
	SkillName        string                 `json:"skillName"`
	SummonSkillName  string                 `json:"summonSkillName"`
	SkillDist        *float64               `json:"skillDist"`
	SkillPartNum     *float64               `json:"skillPartNum"`
	SkillPartStr     *string                `json:"skillPartStr"`
	SlotName         string                 `json:"slotName"`
	SocketColor      string                 `json:"socketColor"`
	SocketNum        *float64               `json:"socketNum"`
	StrengthGems     *float64               `json:"strengthGems"`
	DexterityGems    *float64               `json:"dexterityGems"`
	IntelligenceGems *float64               `json:"intelligenceGems"`
	Actor            string                 `json:"actor"`
	SkillCond        map[string]bool        `json:"skillCond"`
	SkillTypes       map[string]bool        `json:"skillTypes"`
	BaseFlags        map[string]bool        `json:"baseFlags"`
	GeId             *string                `json:"geId"`
	GeBaseFlags      map[string]bool        `json:"geBaseFlags"`
	Gem              string                 `json:"gem"`
	Item             string                 `json:"item"`
	SkillStats       map[string]float64     `json:"skillStats"`
}

type msQRes struct {
	Sb  string          `json:"sb"`
	Si  string          `json:"si"`
	Mo  string          `json:"mo"`
	Fl  json.RawMessage `json:"fl"`
	Ov  json.RawMessage `json:"ov"`
	Li  string          `json:"li"`
	Ta  string          `json:"ta"`
	Ha  bool            `json:"ha"`
	Mx  json.RawMessage `json:"mx"`
	Mn  json.RawMessage `json:"mn"`
	LiC *string         `json:"liC"`
	TaC *string         `json:"taC"`
}

func TestModStoreAgainstReference(t *testing.T) {
	// The reference dump for this differential parsed fresh (its tool wipes
	// the preloaded ModCache); run the parser in the same mode.
	modparser.SetModCache(nil)

	f, err := os.Open("testdata/modstore_archive.jsonl")
	if err != nil {
		t.Fatalf("modstore archive dump not generated (run luajit ../../tools/dump_modstore.lua from .archive/src/): %v", err)
	}
	defer f.Close()

	// The fixture world.
	rootDB := modstore.NewDB(nil)
	midList := modstore.NewList(rootDB)
	leafDB := modstore.NewDB(midList)
	enemyDB := modstore.NewDB(nil)
	parentDB := modstore.NewDB(nil)
	playerActor := rootDB.Actor
	midList.Actor = playerActor
	leafDB.Actor = playerActor
	enemyActor := enemyDB.Actor
	parentActor := parentDB.Actor
	playerActor.DB = leafDB
	playerActor.Enemy = enemyActor
	playerActor.ParentActor = parentActor
	enemyActor.DB = enemyDB
	enemyActor.Player = playerActor
	parentActor.DB = parentDB
	playerActor.WeaponData1 = &msWeapon{all1H: true, added: map[string]bool{"Sword": true, "Axe": false}}
	playerActor.WeaponData2 = &msWeapon{}
	playerActor.MinionData = &modstore.MinionData{MonsterTags: []string{"demon", "humanoid"}}
	playerActor.ManaEfficiency = 20
	hasRes := float64(modparser.SkillTypeHasReservation)
	playerActor.HasReservation = hasRes
	enemyActor.HasReservation = hasRes
	parentActor.HasReservation = hasRes
	playerActor.ActiveSkillList = []*modstore.ActiveSkill{
		{SkillTypes: map[float64]bool{hasRes: true}, SkillData: map[string]float64{"ManaReservedBase": 300, "LifeReservedBase": 960}, BuffNames: []string{"Hatred"}},
		{SkillTypes: map[float64]bool{hasRes: true}, Disable: true, SkillData: map[string]float64{"ManaReservedBase": 500}, BuffNames: []string{"Wrath"}},
	}
	stores := map[string]modstore.Store{
		"root": rootDB, "mid": midList, "leaf": leafDB, "enemy": enemyDB, "parentDB": parentDB,
	}
	baseOf := map[string]*modstore.ModStore{
		"root": &rootDB.ModStore, "mid": &midList.ModStore, "leaf": &leafDB.ModStore,
		"enemy": &enemyDB.ModStore, "parentDB": &parentDB.ModStore,
	}
	actorsByName := map[string]*modstore.Actor{"player": playerActor, "enemy": enemyActor, "parent": parentActor}
	outputs := map[string]modstore.Output{}
	outputOf := func(actor string) modstore.Output {
		if outputs[actor] == nil {
			outputs[actor] = modstore.Output{}
			actorsByName[actor].Output = outputs[actor]
		}
		return outputs[actor]
	}

	items := map[string]*msItem{}
	gems := map[string]*msGem{}
	resolver := &msResolver{}
	for _, actor := range actorsByName {
		actor.Resolver = resolver
	}

	var cfgs []*modstore.Cfg
	buildCfg := func(rec *msCfgRec) *modstore.Cfg {
		if rec == nil {
			return nil
		}
		cfg := &modstore.Cfg{
			Flags:            rec.Flags,
			KeywordFlags:     rec.KeywordFlags,
			Source:           rec.Source,
			SkillName:        rec.SkillName,
			SummonSkillName:  rec.SummonSkillName,
			SkillDist:        rec.SkillDist,
			SlotName:         rec.SlotName,
			SocketColor:      rec.SocketColor,
			SocketNum:        rec.SocketNum,
			StrengthGems:     rec.StrengthGems,
			DexterityGems:    rec.DexterityGems,
			IntelligenceGems: rec.IntelligenceGems,
			Actor:            rec.Actor,
			SkillCond:        rec.SkillCond,
			BaseFlags:        rec.BaseFlags,
		}
		if rec.SkillStats != nil {
			stats := modstore.Output{}
			for k, v := range rec.SkillStats {
				stats.SetN(k, v)
			}
			cfg.SkillStats = stats
		}
		if rec.SkillPartNum != nil {
			cfg.SkillPart = util.Some(*rec.SkillPartNum)
		} else if rec.SkillPartStr != nil {
			t.Fatalf("string skill part %q: the config models numeric parts only", *rec.SkillPartStr)
		}
		if rec.SkillTypes != nil {
			cfg.SkillTypes = map[float64]bool{}
			for k, v := range rec.SkillTypes {
				n, err := strconv.ParseFloat(k, 64)
				if err != nil {
					t.Fatalf("bad skillType key %q", k)
				}
				cfg.SkillTypes[n] = v
			}
		}
		if rec.GeId != nil {
			cfg.SkillGrantedEffect = &modstore.GrantedEffectRef{Id: *rec.GeId, BaseFlags: rec.GeBaseFlags}
		}
		if rec.Gem != "" {
			cfg.SkillGem = gems[rec.Gem]
		}
		if rec.Item != "" {
			cfg.Item = items[rec.Item]
		}
		return cfg
	}

	parseLine := func(line string) []*modparser.Mod {
		mods, _, _ := modparser.Parse(line)
		return mods
	}

	tabCanon := func(entries []modstore.TabEntry) string {
		arr := make([]any, len(entries))
		for i, e := range entries {
			arr[i] = map[string]any{"value": e.Value, "mod": e.Mod}
		}
		return luacanon.CanonMods(arr)
	}
	listCanon := func(vals []modparser.Value) string {
		return luacanon.CanonMods(vals)
	}
	modsCanon := func(mods []*modparser.Mod) string {
		arr := make([]any, len(mods))
		for i, m := range mods {
			arr[i] = m
		}
		return luacanon.CanonMods(arr)
	}
	dbCanon := func(db *modstore.DB) string {
		m := map[string]any{}
		for name, list := range db.Mods {
			arr := make([]any, len(list))
			for i, mod := range list {
				arr[i] = mod
			}
			m[name] = arr
		}
		return luacanon.CanonMods(m)
	}

	var corpusLines []string
	// q records whose reference canon carries coerced numeric-string tag
	// fields (cfg1's full canon reveals it; the other cfgs only hash).
	coercedRecords := map[string]bool{}
	recKey := func(names []string) string { return strings.Join(names, ",") }
	var synthDB *modstore.DB
	var synthMods []*modparser.Mod
	var checked, disagree, shown int
	fail := func(what string, detail string) {
		disagree++
		if shown < 12 {
			shown++
			t.Errorf("%s: %s", what, detail)
		}
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<26)
	for sc.Scan() {
		lineBytes := sc.Bytes()
		if len(lineBytes) == 0 {
			continue
		}
		var head struct {
			K string `json:"k"`
		}
		if err := json.Unmarshal(lineBytes, &head); err != nil {
			t.Fatalf("decoding archive dump line: %v", err)
		}
		switch head.K {
		case "add":
			var rec struct {
				Line    string   `json:"line"`
				Stores  []string `json:"stores"`
				Sources []string `json:"sources"`
			}
			json.Unmarshal(lineBytes, &rec)
			mods := parseLine(rec.Line)
			if len(mods) != len(rec.Stores) {
				t.Fatalf("%s: parsed %d mods, archive dump placed %d", rec.Line, len(mods), len(rec.Stores))
			}
			for i, mod := range mods {
				modparser.SetSource(mod, rec.Sources[i])
				stores[rec.Stores[i]].AddMod(mod)
			}
			corpusLines = append(corpusLines, rec.Line)
		case "mult":
			var rec struct {
				Store string             `json:"store"`
				Vals  map[string]float64 `json:"vals"`
			}
			json.Unmarshal(lineBytes, &rec)
			baseOf[rec.Store].Multipliers = rec.Vals
		case "cond":
			var rec struct {
				Store string          `json:"store"`
				Vals  map[string]bool `json:"vals"`
			}
			json.Unmarshal(lineBytes, &rec)
			conds := modstore.Conditions{}
			for k, v := range rec.Vals {
				conds.Set(k, v)
			}
			baseOf[rec.Store].Conditions = conds
		case "output":
			var rec struct {
				Actor string             `json:"actor"`
				Vals  map[string]float64 `json:"vals"`
			}
			json.Unmarshal(lineBytes, &rec)
			out := modstore.Output{}
			for k, v := range rec.Vals {
				out.SetN(k, v)
			}
			outputs[rec.Actor] = out
			actorsByName[rec.Actor].Output = out
		case "gameIds":
			var rec struct {
				Vals map[string]string `json:"vals"`
			}
			json.Unmarshal(lineBytes, &rec)
			resolver.gameIds = rec.Vals
		case "items":
			var rec struct {
				Vals map[string]struct {
					Name      string          `json:"name"`
					Type      string          `json:"type"`
					Rarity    string          `json:"rarity"`
					Corrupted bool            `json:"corrupted"`
					Shaper    bool            `json:"shaper"`
					Elder     bool            `json:"elder"`
					Fms       map[string]bool `json:"fms"`
				} `json:"vals"`
			}
			json.Unmarshal(lineBytes, &rec)
			for id, it := range rec.Vals {
				items[id] = &msItem{name: it.Name, itype: it.Type, rarity: it.Rarity,
					corrupted: it.Corrupted, shaper: it.Shaper, elder: it.Elder, fms: it.Fms}
			}
			playerActor.ItemList = map[string]modstore.Item{
				"Ring 1": items["item1"], "Ring 2": items["item2"],
				"Helmet": items["item3"], "Jewel 3": items["item4"],
			}
		case "gems":
			var rec struct {
				Vals map[string]struct {
					Tags map[string]bool `json:"tags"`
				} `json:"vals"`
			}
			json.Unmarshal(lineBytes, &rec)
			for id, g := range rec.Vals {
				gems[id] = &msGem{tags: g.Tags}
			}
		case "cfgs":
			var rec struct {
				List []*msCfgRec `json:"list"`
			}
			json.Unmarshal(lineBytes, &rec)
			for _, c := range rec.List {
				cfgs = append(cfgs, buildCfg(c))
			}
		case "q":
			var rec struct {
				Names []string `json:"names"`
				Res   []msQRes `json:"res"`
			}
			json.Unmarshal(lineBytes, &rec)
			for ci, want := range rec.Res {
				cfg := cfgs[ci]
				checked++
				checkNum := func(field, fromArchive string, fn func() float64) {
					got, panicked := tryNum(fn)
					if fromArchive == "!" {
						if !panicked {
							fail("q "+field, fmt.Sprintf("%v cfg%d: reference errored, port returned %v", rec.Names, ci+1, got))
						}
						return
					}
					if panicked {
						fail("q "+field, fmt.Sprintf("%v cfg%d: port panicked, reference got %s", rec.Names, ci+1, fromArchive))
						return
					}
					if !numEq(fromArchive, got) {
						fail("q "+field, fmt.Sprintf("%v cfg%d: want %s got %s", rec.Names, ci+1, fromArchive, f17(got)))
					}
				}
				checkNum("sum BASE", want.Sb, func() float64 { return leafDB.Sum(modparser.Base, cfg, rec.Names...) })
				checkNum("sum INC", want.Si, func() float64 { return leafDB.Sum(modparser.Inc, cfg, rec.Names...) })
				checkNum("more", want.Mo, func() float64 { return leafDB.More(cfg, rec.Names...) })

				gotFlag, flagPanic := tryAny(func() any { return leafDB.Flag(cfg, rec.Names...) })
				if string(want.Fl) == `"!"` {
					if !flagPanic {
						fail("q flag", fmt.Sprintf("%v cfg%d: reference errored", rec.Names, ci+1))
					}
				} else if flagPanic {
					fail("q flag", fmt.Sprintf("%v cfg%d: port panicked", rec.Names, ci+1))
				} else if fmt.Sprintf("%v", gotFlag) != string(want.Fl) {
					fail("q flag", fmt.Sprintf("%v cfg%d: want %s got %v", rec.Names, ci+1, want.Fl, gotFlag))
				}

				gotOvr, ovrPanic := tryAny(func() any {
					v, _ := leafDB.Override(cfg, rec.Names...)
					return v
				})
				if string(want.Ov) == `"!"` {
					if !ovrPanic {
						fail("q override", fmt.Sprintf("%v cfg%d: reference errored", rec.Names, ci+1))
					}
				} else if ovrPanic {
					fail("q override", fmt.Sprintf("%v cfg%d: port panicked", rec.Names, ci+1))
				} else if luacanon.CanonMods(gotOvr) != string(want.Ov) {
					fail("q override", fmt.Sprintf("%v cfg%d: want %s got %s", rec.Names, ci+1, want.Ov, luacanon.CanonMods(gotOvr)))
				}

				gotList, listPanic := tryAny(func() any { return listCanon(leafDB.List(cfg, rec.Names...)) })
				listC := "!"
				if !listPanic {
					listC = gotList.(string)
				}
				// The archive hashed its own canon text; where cfg1's full
				// canon shows the reference kept a numeric tag field as
				// text (luacanon.NormalizeArchiveMods), every cfg's hash is
				// over that text and cannot be normalised, so cfg1's full
				// canon comparison is the check for the record.
				if want.LiC != nil {
					wantLiC := luacanon.NormalizeArchiveMods(*want.LiC)
					if wantLiC != *want.LiC {
						coercedRecords[recKey(rec.Names)+"|li"] = true
					}
					if listC != wantLiC {
						fail("q listC", fmt.Sprintf("%v cfg%d:\n  want %s\n  got  %s", rec.Names, ci+1, wantLiC, listC))
					}
				}
				if msHash(listC) != want.Li && !coercedRecords[recKey(rec.Names)+"|li"] {
					fail("q list", fmt.Sprintf("%v cfg%d: hash mismatch (got %s)", rec.Names, ci+1, listC))
				}

				gotTab, tabPanic := tryAny(func() any { return tabCanon(leafDB.TabulateAll(cfg, rec.Names...)) })
				tabC := "!"
				if !tabPanic {
					tabC = gotTab.(string)
				}
				if want.TaC != nil {
					wantTaC := luacanon.NormalizeArchiveMods(*want.TaC)
					if wantTaC != *want.TaC {
						coercedRecords[recKey(rec.Names)+"|ta"] = true
					}
					if tabC != wantTaC {
						fail("q tabC", fmt.Sprintf("%v cfg%d:\n  want %s\n  got  %s", rec.Names, ci+1, wantTaC, tabC))
					}
				}
				if msHash(tabC) != want.Ta && !coercedRecords[recKey(rec.Names)+"|ta"] {
					fail("q tab", fmt.Sprintf("%v cfg%d: hash mismatch (got %s)", rec.Names, ci+1, tabC))
				}

				if got := rootDB.HasMod(modparser.Base, cfg, rec.Names...); got != want.Ha {
					fail("q hasMod", fmt.Sprintf("%v cfg%d: want %v got %v", rec.Names, ci+1, want.Ha, got))
				}

				checkOpt := func(field string, fromArchive json.RawMessage, fn func() (float64, bool)) {
					got, panicked := tryAny(func() any {
						v, ok := fn()
						if !ok {
							return nil
						}
						return v
					})
					if string(fromArchive) == `"!"` {
						if !panicked {
							fail("q "+field, fmt.Sprintf("%v cfg%d: reference errored", rec.Names, ci+1))
						}
						return
					}
					if panicked {
						fail("q "+field, fmt.Sprintf("%v cfg%d: port panicked", rec.Names, ci+1))
						return
					}
					if string(fromArchive) == "null" {
						if got != nil {
							fail("q "+field, fmt.Sprintf("%v cfg%d: want nil got %v", rec.Names, ci+1, got))
						}
						return
					}
					var s string
					json.Unmarshal(fromArchive, &s)
					if got == nil || !numEq(s, got.(float64)) {
						fail("q "+field, fmt.Sprintf("%v cfg%d: want %s got %v", rec.Names, ci+1, s, got))
					}
				}
				checkOpt("max", want.Mx, func() (float64, bool) { return leafDB.Max(cfg, rec.Names...) })
				checkOpt("min", want.Mn, func() (float64, bool) { return leafDB.Min(cfg, rec.Names...) })
			}
		case "gm":
			var rec struct {
				Var string   `json:"var"`
				Res []string `json:"res"`
			}
			json.Unmarshal(lineBytes, &rec)
			for ci, fromArchive := range rec.Res {
				checked++
				got, panicked := tryNum(func() float64 { return leafDB.GetMultiplier(rec.Var, cfgs[ci]) })
				if fromArchive == "!" {
					if !panicked {
						fail("gm", fmt.Sprintf("%s cfg%d: reference errored", rec.Var, ci+1))
					}
				} else if panicked || !numEq(fromArchive, got) {
					fail("gm", fmt.Sprintf("%s cfg%d: want %s got %s (panic=%v)", rec.Var, ci+1, fromArchive, f17(got), panicked))
				}
			}
		case "gc":
			var rec struct {
				Var string            `json:"var"`
				Res []json.RawMessage `json:"res"`
			}
			json.Unmarshal(lineBytes, &rec)
			for ci, fromArchive := range rec.Res {
				checked++
				got, panicked := tryAny(func() any { return leafDB.GetCondition(rec.Var, cfgs[ci]) })
				if string(fromArchive) == `"!"` {
					if !panicked {
						fail("gc", fmt.Sprintf("%s cfg%d: reference errored", rec.Var, ci+1))
					}
				} else if panicked || fmt.Sprintf("%v", got) != string(fromArchive) {
					fail("gc", fmt.Sprintf("%s cfg%d: want %s got %v (panic=%v)", rec.Var, ci+1, fromArchive, got, panicked))
				}
			}
		case "eq":
			var rec struct {
				Name string `json:"name"`
				Sb   string `json:"sb"`
				Mo   string `json:"mo"`
				Ta   string `json:"ta"`
			}
			json.Unmarshal(lineBytes, &rec)
			checked++
			checkNumE := func(field, fromArchive string, fn func() float64) {
				got, panicked := tryNum(fn)
				if fromArchive == "!" {
					if !panicked {
						fail("eq "+field, fmt.Sprintf("%s: reference errored", rec.Name))
					}
				} else if panicked || !numEq(fromArchive, got) {
					fail("eq "+field, fmt.Sprintf("%s: want %s got %s (panic=%v)", rec.Name, fromArchive, f17(got), panicked))
				}
			}
			checkNumE("sum", rec.Sb, func() float64 { return enemyDB.Sum(modparser.Base, cfgs[7], rec.Name) })
			checkNumE("more", rec.Mo, func() float64 { return enemyDB.More(cfgs[8], rec.Name) })
			gotTab, tabPanic := tryAny(func() any { return tabCanon(enemyDB.TabulateAll(cfgs[1], rec.Name)) })
			tabC := "!"
			if !tabPanic {
				tabC = gotTab.(string)
			}
			if msHash(tabC) != rec.Ta {
				fail("eq tab", fmt.Sprintf("%s: hash mismatch (got %s)", rec.Name, tabC))
			}
		case "scale":
			var rec struct {
				Line  string `json:"line"`
				Scale string `json:"scale"`
				List  string `json:"list"`
				DB    string `json:"db"`
			}
			json.Unmarshal(lineBytes, &rec)
			checked++
			rec.List, rec.DB = luacanon.NormalizeArchiveMods(rec.List), luacanon.NormalizeArchiveMods(rec.DB)
			scale, _ := strconv.ParseFloat(rec.Scale, 64)
			list := modstore.NewList(nil)
			for _, mod := range parseLine(rec.Line) {
				list.ScaleAddMod(mod, scale, false)
			}
			db := modstore.NewDB(nil)
			for _, mod := range parseLine(rec.Line) {
				db.ScaleAddMod(mod, scale, false)
			}
			if got := modsCanon(list.Mods); got != rec.List {
				fail("scale list", fmt.Sprintf("%s x%s:\n  want %s\n  got  %s", rec.Line, rec.Scale, rec.List, got))
			}
			if got := dbCanon(db); got != rec.DB {
				fail("scale db", fmt.Sprintf("%s x%s:\n  want %s\n  got  %s", rec.Line, rec.Scale, rec.DB, got))
			}
		case "mergeMod":
			var rec struct {
				Line string `json:"line"`
				List string `json:"list"`
			}
			json.Unmarshal(lineBytes, &rec)
			checked++
			rec.List = luacanon.NormalizeArchiveMods(rec.List)
			list := modstore.NewList(nil)
			for _, mod := range parseLine(rec.Line) {
				list.MergeMod(mod, false)
			}
			for _, mod := range parseLine(rec.Line) {
				list.MergeMod(mod, false)
			}
			if got := modsCanon(list.Mods); got != rec.List {
				fail("mergeMod", fmt.Sprintf("%s:\n  want %s\n  got  %s", rec.Line, rec.List, got))
			}
		case "replace":
			var rec struct {
				Line string `json:"line"`
				Base string `json:"base"`
				DB   string `json:"db"`
			}
			json.Unmarshal(lineBytes, &rec)
			checked++
			rec.Base, rec.DB = luacanon.NormalizeArchiveMods(rec.Base), luacanon.NormalizeArchiveMods(rec.DB)
			base := modstore.NewList(nil)
			db := modstore.NewDB(base)
			for _, mod := range parseLine(rec.Line) {
				base.AddMod(mod)
			}
			for _, mod := range parseLine(rec.Line) {
				repl := mod.Clone()
				if v, ok := repl.Value.(modparser.Num); ok {
					repl.Value = v + 100
				}
				if !db.ReplaceModInternal(repl) {
					db.AddMod(repl)
				}
				conv := mod.Clone()
				conv.Name = mod.Name + "X"
				if !db.ConvertModInternal(mod.Name, conv) {
					db.AddMod(conv)
				}
			}
			if got := modsCanon(base.Mods); got != rec.Base {
				fail("replace base", fmt.Sprintf("%s:\n  want %s\n  got  %s", rec.Line, rec.Base, got))
			}
			if got := dbCanon(db); got != rec.DB {
				fail("replace db", fmt.Sprintf("%s:\n  want %s\n  got  %s", rec.Line, rec.DB, got))
			}
		case "outputNan":
			var rec struct {
				Actor string `json:"actor"`
				Stat  string `json:"stat"`
			}
			json.Unmarshal(lineBytes, &rec)
			outputOf(rec.Actor).SetN(rec.Stat, math.NaN())
		case "outputSet":
			var rec struct {
				Actor string `json:"actor"`
				Stat  string `json:"stat"`
				Val   string `json:"val"`
			}
			json.Unmarshal(lineBytes, &rec)
			n, _ := strconv.ParseFloat(rec.Val, 64)
			outputOf(rec.Actor).SetN(rec.Stat, n)
		case "synth":
			var rec struct {
				Spec string `json:"spec"`
			}
			json.Unmarshal(lineBytes, &rec)
			// One synthetic value carries an arbitrary extra key ("other")
			// beside keyOfScaledMod; the typed record has no slot for it and
			// the scaling under test does not read it.
			rec.Spec = strings.Replace(rec.Spec, `"other":"x",`, "", 1)
			var m map[string]any
			if err := json.Unmarshal([]byte(rec.Spec), &m); err != nil {
				t.Fatalf("bad synth spec: %v", err)
			}
			mod := luacanon.ModFromTable(m)
			if synthDB == nil {
				synthDB = modstore.NewDB(midList)
				synthDB.Actor = playerActor
			}
			synthDB.AddMod(mod)
			synthMods = append(synthMods, mod)
			if got := luacanon.CanonMods(mod); got != luacanon.NormalizeArchiveMods(rec.Spec) {
				fail("synth spec", fmt.Sprintf("decode round-trip:\n  want %s\n  got  %s", rec.Spec, got))
			}
		case "sq":
			var rec struct {
				Name string `json:"name"`
				Res  []struct {
					Sb string `json:"sb"`
					Si string `json:"si"`
					Ta string `json:"ta"`
				} `json:"res"`
			}
			json.Unmarshal(lineBytes, &rec)
			names := []string{rec.Name}
			if rec.Name == "SynthGlobalLimitPair" {
				names = []string{"SynthGlobalLimit1", "SynthGlobalLimit2"}
			}
			for ci, want := range rec.Res {
				checked++
				cfg := cfgs[ci]
				checkSyn := func(field, fromArchive string, fn func() float64) {
					got, panicked := tryNum(fn)
					if fromArchive == "!" {
						if !panicked {
							fail("sq "+field, fmt.Sprintf("%s cfg%d: reference errored, port got %v", rec.Name, ci+1, got))
						}
						return
					}
					if panicked || !numEq(fromArchive, got) {
						fail("sq "+field, fmt.Sprintf("%s cfg%d: want %s got %s (panic=%v)", rec.Name, ci+1, fromArchive, f17(got), panicked))
					}
				}
				checkSyn("sum BASE", want.Sb, func() float64 { return synthDB.Sum(modparser.Base, cfg, names...) })
				checkSyn("sum INC", want.Si, func() float64 { return synthDB.Sum(modparser.Inc, cfg, names...) })
				want.Ta = strings.ReplaceAll(want.Ta, `"other":"x",`, "") // see the synth case
				if want.Ta != "skip" && want.Ta != "\"skip\"" {
					gotTab, tabPanic := tryAny(func() any { return tabCanon(synthDB.TabulateAll(cfg, names...)) })
					tabC := "!"
					if !tabPanic {
						tabC = gotTab.(string)
					}
					if tabC != want.Ta {
						fail("sq tab", fmt.Sprintf("%s cfg%d:\n  want %s\n  got  %s", rec.Name, ci+1, want.Ta, tabC))
					}
				}
			}
		case "synthScale":
			var rec struct {
				List string `json:"list"`
			}
			json.Unmarshal(lineBytes, &rec)
			checked++
			list := modstore.NewList(nil)
			list.ScaleAddMod(synthMods[len(synthMods)-1].Clone(), 2.5, false)
			list.ScaleAddMod(synthMods[19].Clone(), 2.5, false)
			list.ScaleAddMod(synthMods[18].Clone(), 2.5, false)
			rec.List = strings.Replace(rec.List, `"other":"x",`, "", 1)
			if got := modsCanon(list.Mods); got != rec.List {
				fail("synthScale", fmt.Sprintf("\n  want %s\n  got  %s", rec.List, got))
			}
		case "keystones":
			var rec struct {
				Map   map[string]string `json:"map"`
				Added []string          `json:"added"`
				DB    string            `json:"db"`
			}
			json.Unmarshal(lineBytes, &rec)
			checked++
			// Rebuild the keystone map with the dump's selection rule and
			// verify it matches the recorded canon before merging.
			keystoneMods := map[string][]*modparser.Mod{}
			var mapNames []string
			ki := 0
			pi := 0
			// Re-derive parsed order from the adds seen: we replay the same
			// sorted corpus.
			for _, line := range corpusLines {
				pi++
				if pi%97 == 0 && ki < 24 {
					ki++
					ksName := fmt.Sprintf("Keystone%d", ki)
					keystoneMods[ksName] = parseLine(line)
					mapNames = append(mapNames, ksName)
				}
			}
			for name, wantCanon := range rec.Map {
				if got := modsCanon(keystoneMods[name]); got != luacanon.NormalizeArchiveMods(wantCanon) {
					fail("keystones map", fmt.Sprintf("%s:\n  want %s\n  got  %s", name, wantCanon, got))
				}
			}
			db := modstore.NewDB(nil)
			for i, ksName := range mapNames {
				src := "Item:5:Sceptre"
				if (i+1)%2 == 0 {
					src = "Tree:node"
				}
				db.AddMod(&modparser.Mod{Name: "Keystone", Type: modparser.List, Value: modparser.Str(ksName), Source: src, SourceSet: true})
				if (i+1)%3 == 0 {
					db.AddMod(&modparser.Mod{Name: "Keystone", Type: modparser.List, Value: modparser.Str(ksName), Source: "Item:6:Ring", SourceSet: true})
				}
			}
			db.AddMod(&modparser.Mod{Name: "Keystone", Type: modparser.List, Value: modparser.Str("UnknownKeystone"), Source: "Tree:x", SourceSet: true})
			env := &modstore.KeystoneEnv{KeystoneMods: keystoneMods}
			modstore.MergeKeystones(env, db)
			var added []string
			for name := range env.KeystonesAdded {
				added = append(added, name)
			}
			sortStrings(added)
			if fmt.Sprintf("%v", added) != fmt.Sprintf("%v", rec.Added) {
				fail("keystones added", fmt.Sprintf("want %v got %v", rec.Added, added))
			}
			if got := dbCanon(db); got != luacanon.NormalizeArchiveMods(rec.DB) {
				fail("keystones db", fmt.Sprintf("db:\n  want %s\n  got  %s", rec.DB, got))
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	t.Logf("modstore vs archive: %d checks, %d disagreements", checked, disagree)
	if disagree > 0 {
		t.Fatalf("%d disagreements with the archive", disagree)
	}
}

func toLower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
