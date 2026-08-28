package test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// diffWindow returns s around the first byte where s and other diverge.
func diffWindow(s, other string) string {
	i := 0
	for i < len(s) && i < len(other) && s[i] == other[i] {
		i++
	}
	start := i - 80
	if start < 0 {
		start = 0
	}
	end := i + 120
	if end > len(s) {
		end = len(s)
	}
	oend := i + 120
	if oend > len(other) {
		oend = len(other)
	}
	ostart := start
	if ostart > len(other) {
		ostart = len(other)
	}
	return "[@" + strconv.Itoa(i) + "] got  ..." + s[start:end] + "...\n" +
		"          want ..." + other[ostart:oend] + "..."
}

// Structured mods inside the data tree canonicalise via their plain-table
// shadow.
var _ = func() bool {
	luacanon.RegisterAdapter(func(v any) (any, bool) {
		switch t := v.(type) {
		case *modparser.Mod:
			return data.ModCanon(t), true
		case *modparser.D:
			return data.DCanon(t), true
		case *data.GrantedEffect:
			return data.GrantedEffectCanon(t), true
		case *data.StatMapEntry:
			return data.StatMapEntryCanon(t), true
		case *data.SkillLevel:
			return data.SkillLevelCanon(t), true
		case data.UnportedFn:
			return luacanon.Fn{}, true
		case *data.Gem:
			return data.GemCanon(t), true
		}
		return nil, false
	})
	return true
}()

// The game-data differential test: builds the schema documents over the
// extracted GGPK, assembles the runtime data set with data.Load, and
// compares each ported subtree canonically against the archive dump
// (tools/dump_gamedata.lua run over the loaded Lua application). Fails on
// any disagreement.
func TestGameDataAgainstReference(t *testing.T) {
	dumpPath := filepath.Join("testdata", "gamedata_archive.jsonl")
	// The loaded data set is immutable after Load (calc runs keep their
	// mutations per-env), so it can be compared regardless of what ran
	// before this test — except Uniques["generated"], which a tree load
	// extends with the tree-dependent uniques; the archive dump captured
	// the load-time (pre-tree) state.
	loadData(t)
	data.TrimTreeDependentUniques()

	checks := map[string]func() any{
		"monsterEvasionTable":             func() any { return data.MonsterEvasionTable },
		"monsterAccuracyTable":            func() any { return data.MonsterAccuracyTable },
		"monsterLifeTable":                func() any { return data.MonsterLifeTable },
		"monsterLifeTable2":               func() any { return data.MonsterLifeTable2 },
		"monsterLifeTable3":               func() any { return data.MonsterLifeTable3 },
		"monsterAllyLifeTable":            func() any { return data.MonsterAllyLifeTable },
		"monsterDamageTable":              func() any { return data.MonsterDamageTable },
		"monsterAllyDamageTable":          func() any { return data.MonsterAllyDamageTable },
		"monsterArmourTable":              func() any { return data.MonsterArmourTable },
		"monsterAilmentThresholdTable":    func() any { return data.MonsterAilmentThresholdTable },
		"monsterPhysConversionMultiTable": func() any { return data.MonsterPhysConversionMultiTable },
		"gameConstants":                   func() any { return data.GameConstants },
		"characterConstants":              func() any { return data.CharacterConstants },
		"monsterConstants":                func() any { return data.MonsterConstants },
		"totemLifeMult":                   func() any { return data.TotemLifeMult },
		"monsterVarietyLifeMult":          func() any { return data.MonsterVarietyLifeMult },
		"mapLevelLifeMult":                func() any { return data.MapLevelLifeMult },
		"mapLevelBossLifeMult":            func() any { return data.MapLevelBossLifeMult },
		"mapLevelBossAilmentMult":         func() any { return data.MapLevelBossAilmentMult },
		"goldRespecPrices":                func() any { return data.GoldRespecPrices },
		"misc":                            func() any { return data.Misc },
		"powerStatList": func() any {
			m := map[string]any{"GetFromOutput": luacanon.Fn{}}
			for i, e := range data.PowerStatList {
				m[strconv.Itoa(i+1)] = e
			}
			return m
		},
		"skillColorMap":                  func() any { return data.SkillColorMap },
		"monsterExperienceLevelMap":      func() any { return data.MonsterExperienceLevelMap },
		"cursePriority":                  func() any { return data.CursePriority },
		"keystones":                      func() any { return data.Keystones },
		"ailmentTypeList":                func() any { return data.AilmentTypeList },
		"elementalAilmentTypeList":       func() any { return data.ElementalAilmentTypeList },
		"nonDamagingAilmentTypeList":     func() any { return data.NonDamagingAilmentTypeList },
		"nonElementalAilmentTypeList":    func() any { return data.NonElementalAilmentTypeList },
		"nonDamagingAilment":             func() any { return data.NonDamagingAilment },
		"defaultHighPrecision":           func() any { return data.DefaultHighPrecision },
		"highPrecisionMods":              func() any { return data.HighPrecisionMods },
		"modScalability":                 func() any { return data.ModScalability },
		"weaponTypeInfo":                 func() any { return data.WeaponTypeInfo },
		"unarmedWeaponData":              func() any { return data.UnarmedWeaponData },
		"jewelRadii":                     func() any { return data.JewelRadii },
		"jewelRadius":                    func() any { return data.JewelRadius },
		"maxJewelRadius":                 func() any { return data.MaxJewelRadius },
		"enchantmentSource":              func() any { return data.EnchantmentSource },
		"timelessJewelTypes":             func() any { return data.TimelessJewelTypes },
		"timelessJewelSeedMin":           func() any { return data.TimelessJewelSeedMin },
		"timelessJewelSeedMax":           func() any { return data.TimelessJewelSeedMax },
		"timelessJewelAdditions":         func() any { return data.TimelessJewelAdditions },
		"itemTagSpecial":                 func() any { return data.ItemTagSpecial },
		"itemTagSpecialExclusionPattern": func() any { return data.ItemTagSpecialExclusionPattern },
		"casterTagCrucibleUniques":       func() any { return data.CasterTagCrucibleUniques },
		"minionTagCrucibleUniques":       func() any { return data.MinionTagCrucibleUniques },
		"costs":                          func() any { return data.Costs },
		"mapMods":                        func() any { return data.MapMods },
		"nodeIDList":                     func() any { return data.NodeIDList },
		"abyssNotableNames":              func() any { return data.AbyssNotableNames },
		"timelessJewelTradeIDs":          func() any { return data.TimelessJewelTradeIDs },
		"timelessJewelLUTs":              func() any { return data.TimelessJewelLUTs },
		// function-valued members: ported by the stat-describer and
		// timeless-jewel-data modules
		"describeStats":              func() any { return luacanon.Fn{} },
		"readLUT":                    func() any { return luacanon.Fn{} },
		"repairLUTs":                 func() any { return luacanon.Fn{} },
		"readAbyssJewelLUT":          func() any { return luacanon.Fn{} },
		"resolveAbyssJewelComponent": func() any { return luacanon.Fn{} },
		"getAbyssJewelComponentRoll": func() any { return luacanon.Fn{} },
		"bosses":                     func() any { return data.Bosses },
		"bossStats":                  func() any { return data.BossStats },
		"enemyIsBossTooltip":         func() any { return data.EnemyIsBossTooltip },
		"essences":                   func() any { return data.Essences },
		"pantheons":                  func() any { return data.Pantheons },
		"crucible":                   func() any { return data.Crucible },
		"masterMods":                 func() any { return data.MasterMods },
		"flavourText":                func() any { return data.FlavourText },
		"enchantments": func() any {
			m := map[string]any{"Helmet": data.HelmetEnchants}
			for k, v := range data.Enchantments {
				m[k] = v
			}
			return m
		},
		"beastCraft":                 func() any { return data.BeastCraft },
		"necropolisMods":             func() any { return data.NecropolisMods },
		"uniqueMods":                 func() any { return data.UniqueMods },
		"veiledMods":                 func() any { return data.VeiledMods },
		"clusterJewels":              func() any { return data.ClusterJewels },
		"clusterJewelInfoForNotable": func() any { return data.ClusterJewelInfoForNotable },
		"bossSkills":                 func() any { return data.BossSkills },
		"bossSkillsList":             func() any { return data.BossSkillsList },
		"foulbornMap":                func() any { return data.FoulbornMap },
		"itemBases":                  func() any { return data.ItemBases },
		"itemBaseLists":              func() any { return data.ItemBaseLists },
		"itemBaseTypeList":           func() any { return data.ItemBaseTypeList },
		"rares":                      func() any { return data.Rares },
		"rareLikeUniques":            func() any { return data.RareLikeUniques },
		"minions":                    func() any { return data.Minions },
		"spectres":                   func() any { return data.Spectres },
		"skills":                     func() any { return data.Skills },
		"skillStatMap":               func() any { return data.SkillStatMap },
		"gems":                       func() any { return data.Gems },
		"gemForSkill":                func() any { return data.GemForSkillCanon() },
		"gemForBaseName":             func() any { return data.GemForBaseName },
		"gemsByGameId": func() any {
			out := map[string]map[string]string{}
			for gameId, m := range data.GemsByGameId {
				out[gameId] = map[string]string{}
				for variantId, gem := range m {
					out[gameId][variantId] = gem.Id
				}
			}
			return out
		},
		"gemGrantedEffectIdForVaalGemId": func() any { return data.GemGrantedEffectIdForVaalGemId },
		"gemVaalGemIdForBaseGemId":       func() any { return data.GemVaalGemIdForBaseGemId },
		"skills.statMapKeys": func() any {
			out := map[string][]string{}
			for id, keys := range readStatMapCopies(dumpPath) {
				out[id] = keys
			}
			return out
		},
	}
	for typ := range data.Uniques {
		typ := typ
		checks["uniques."+typ] = func() any { return data.Uniques[typ] }
	}
	_ = checks["uniques.generated"] // built by data.buildGeneratedUniques
	for key := range data.ItemMods {
		key := key
		checks["itemMods."+key] = func() any { return data.ItemMods[key] }
	}

	f, err := os.Open(dumpPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<28)
	var checked, disagree int
	for sc.Scan() {
		var rec struct {
			K string `json:"k"`
			C string `json:"c"`
			H string `json:"h"`
		}
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("bad dump line: %v", err)
		}
		check, ok := checks[rec.K]
		if !ok {
			disagree++
			t.Errorf("%s: dumped but not ported/checked", rec.K)
			continue
		}
		checked++
		got := luacanon.Encode(check())
		if rec.H != "" {
			if msHash(got) != rec.H {
				disagree++
				t.Errorf("%s differs from the archive (canon hash mismatch, %d bytes)", rec.K, len(got))
			}
			continue
		}
		if got != rec.C {
			disagree++
			t.Errorf("%s differs from the archive\n  got:  %s\n  want: %s", rec.K, diffWindow(got, rec.C), diffWindow(rec.C, got))
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("game-data vs archive: %d subtrees checked, %d disagreements", checked, disagree)
	if disagree > 0 {
		t.Fatalf("%d disagreements with the archive", disagree)
	}
}
