// Port of .archive/src/Export/Scripts/miscdata.lua (and the pieces of
// Modules/Utils.lua it uses: stringify/saveTableToFile).

package export

import (
	"fmt"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

func init() {
	Scripts = append(Scripts, Script{Name: "miscdata", Build: buildMiscdata})
}

// otConstants parses an .ot file's wanted blocks the way miscdata.lua
// does, evaluating each value to a number at this edge (the reference
// shipped raw text and re-ran tonumber at load — lua-residue.md T4).
func otConstants(x *Ctx, file string, alsoPathfinding bool) ([]schema.KV, error) {
	raw := x.GetFile(file)
	if raw == "" {
		return nil, nil
	}
	text := convertUTF16to8([]byte(raw), 0)
	ws := strings.NewReplacer(" ", "", "\t", "", "\v", "", "\f", "")
	inWantedBlock := false
	var out []schema.KV
	for _, line := range reLine.FindAllString(text, -1) {
		if strings.HasPrefix(line, "Stats") || (alsoPathfinding && strings.HasPrefix(line, "Pathfinding")) {
			inWantedBlock = true
		} else if inWantedBlock && strings.HasPrefix(line, "}") {
			inWantedBlock = false
		} else if inWantedBlock && strings.Contains(line, "=") {
			stripped := ws.Replace(line)
			eq := strings.Index(stripped, "=")
			key, value := stripped[:eq], stripped[eq+1:]
			if value != "" {
				n, ok := util.Tonumber(value)
				if !ok {
					return nil, fmt.Errorf("%s: non-numeric constant %s = %q", file, key, value)
				}
				out = append(out, schema.KV{Key: key, Value: n})
			}
		}
	}
	return out, nil
}

func buildMiscdata(x *Ctx) (schema.Document, error) {
	var d schema.MiscData
	m := &d.Misc

	var (
		defaultMonsterStats, gameConstants, totemVariations, monsterVarieties *DatFile
		mapDifficulty, mapBossDifficulty, villageBalance, currencyExchange    *DatFile
	)
	for name, dst := range map[string]**DatFile{
		"DefaultMonsterStats":          &defaultMonsterStats,
		"GameConstants":                &gameConstants,
		"SkillTotemVariations":         &totemVariations,
		"MonsterVarieties":             &monsterVarieties,
		"MonsterMapDifficulty":         &mapDifficulty,
		"MonsterMapBossDifficulty":     &mapBossDifficulty,
		"VillageBalancePerLevelShared": &villageBalance,
		"CurrencyExchange":             &currencyExchange,
	} {
		var err error
		if *dst, err = x.Dat(name); err != nil {
			return nil, err
		}
	}

	for stats := range defaultMonsterStats.Rows() {
		m.MonsterEvasion = append(m.MonsterEvasion, float64(stats.Int("Evasion")))
		m.MonsterAccuracy = append(m.MonsterAccuracy, float64(stats.Int("Accuracy")))
		m.MonsterLife = append(m.MonsterLife, float64(stats.Int("MonsterLife")))
		m.MonsterLife2 = append(m.MonsterLife2, float64(stats.Int("AltLife1")))
		m.MonsterLife3 = append(m.MonsterLife3, float64(stats.Int("AltLife2")))
		m.MonsterAllyLife = append(m.MonsterAllyLife, float64(stats.Int("MinionLife")))
		m.MonsterDamage = append(m.MonsterDamage, stats.Float("Damage"))
		m.MonsterAllyDamage = append(m.MonsterAllyDamage, stats.Float("MinionDamage"))
		m.MonsterAilmentThreshold = append(m.MonsterAilmentThreshold, float64(stats.Int("AilmentThreshold")))
		m.MonsterPhysConversionMulti = append(m.MonsterPhysConversionMulti, float64(stats.Int("MonsterPhysConversionMulti")))
	}
	mdri := gameConstants.RowByStr("Id", "MonsterDamageReductionImprovement")
	mdriRatio := float64(mdri.Int("Value")) / float64(mdri.Int("Divisor"))
	for i := 1; i <= 100; i++ {
		m.MonsterArmour = append(m.MonsterArmour, math.Floor((10+2*float64(i))*math.Pow(1+mdriRatio/100, float64(i))))
	}

	for row := range gameConstants.Rows() {
		m.GameConstants = append(m.GameConstants, schema.IdValue{
			Id:    row.Str("Id"),
			Value: float64(row.Int("Value")) / float64(row.Int("Divisor")),
		})
	}

	var err error
	if m.CharacterConstants, err = otConstants(x, "Metadata/Characters/Character.ot", true); err != nil {
		return nil, err
	}
	if m.MonsterConstants, err = otConstants(x, "Metadata/Monsters/Monster.ot", false); err != nil {
		return nil, err
	}

	totemKeys := map[int64]bool{}
	for vr := range totemVariations.Rows() {
		st := vr.Int("SkillTotem")
		if !totemKeys[st] {
			totemKeys[st] = true
			m.TotemLifeMult = append(m.TotemLifeMult, schema.IntMult{
				Id:   st,
				Mult: float64(vr.Ref("MonsterVariety").Int("LifeMultiplier")) / 100,
			})
		}
	}

	cachedEntry := map[string]bool{}
	for row := range monsterVarieties.Rows() {
		for _, mod := range row.Refs("Mods") {
			name := row.Str("Name")
			if mod.Str("Id") == "MonsterNecromancerRaisable" && !cachedEntry[name] {
				m.MonsterVarietyLifeMult = append(m.MonsterVarietyLifeMult, schema.NameMult{
					Name: name,
					Mult: float64(row.Int("LifeMultiplier")) / 100,
				})
				cachedEntry[name] = true
				break
			}
		}
	}

	for row := range mapDifficulty.Rows() {
		m.MapLevelLifeMult = append(m.MapLevelLifeMult, schema.LevelMult{
			Level: row.Int("AreaLevel"),
			Mult:  1 + float64(row.Int("LifePercentIncrease"))/100,
		})
	}
	for vr := range mapBossDifficulty.Rows() {
		lvl := vr.Int("AreaLevel")
		m.MapLevelBossLifeMult = append(m.MapLevelBossLifeMult, schema.LevelMult{
			Level: lvl, Mult: 1 + float64(vr.Int("BossLifePercentIncrease"))/100,
		})
		m.MapLevelBossAilmentMult = append(m.MapLevelBossAilmentMult, schema.LevelMult{
			Level: lvl, Mult: (100 + float64(vr.Int("BossAilmentPercentDecrease"))) / 100,
		})
	}
	for row := range villageBalance.Rows() {
		m.GoldRespecPrices = append(m.GoldRespecPrices, row.Int("GoldRespec"))
	}

	d.CurrencyNames = schema.CurrencyNames{}
	nameClean := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")
	for row := range currencyExchange.Rows() {
		base := row.Ref("BaseItemType")
		name := base.Str("Name")
		if base.Ref("ItemClass").Str("Id") == "StackableCurrency" &&
			name != "" && !strings.Contains(name, "DNT") {
			d.CurrencyNames[base.Str("Id")] = nameClean.Replace(name)
		}
	}
	return d, nil
}
