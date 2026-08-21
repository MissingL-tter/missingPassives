// Port of .archive/src/Export/Scripts/miscdata.lua (and the pieces of
// Modules/Utils.lua it uses: stringify/saveTableToFile).

package export

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/gamedata"
)

func init() {
	Scripts = append(Scripts, Script{Name: "miscdata", Build: buildMiscdata})
}

// numCell converts a numeric dat cell to float64.
func numCell(v any) float64 {
	switch n := v.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	}
	panic("miscdata: non-numeric cell")
}

// otConstants parses an .ot file's wanted blocks the way miscdata.lua does.
func otConstants(x *Ctx, file string, alsoPathfinding bool) []gamedata.KV {
	raw := x.GetFile(file)
	if raw == "" {
		return nil
	}
	text := convertUTF16to8([]byte(raw), 0)
	ws := strings.NewReplacer(" ", "", "\t", "", "\v", "", "\f", "")
	inWantedBlock := false
	var out []gamedata.KV
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
				out = append(out, gamedata.KV{Key: key, Value: value})
			}
		}
	}
	return out
}

func buildMiscdata(x *Ctx) (any, error) {
	var d gamedata.MiscData
	m := &d.Misc

	x.Dat("DefaultMonsterStats").Rows(func(stats *Row) bool {
		m.MonsterEvasion = append(m.MonsterEvasion, numCell(stats.Get("Evasion")))
		m.MonsterAccuracy = append(m.MonsterAccuracy, numCell(stats.Get("Accuracy")))
		m.MonsterLife = append(m.MonsterLife, numCell(stats.Get("MonsterLife")))
		m.MonsterLife2 = append(m.MonsterLife2, numCell(stats.Get("AltLife1")))
		m.MonsterLife3 = append(m.MonsterLife3, numCell(stats.Get("AltLife2")))
		m.MonsterAllyLife = append(m.MonsterAllyLife, numCell(stats.Get("MinionLife")))
		m.MonsterDamage = append(m.MonsterDamage, numCell(stats.Get("Damage")))
		m.MonsterAllyDamage = append(m.MonsterAllyDamage, numCell(stats.Get("MinionDamage")))
		m.MonsterAilmentThreshold = append(m.MonsterAilmentThreshold, numCell(stats.Get("AilmentThreshold")))
		m.MonsterPhysConversionMulti = append(m.MonsterPhysConversionMulti, numCell(stats.Get("MonsterPhysConversionMulti")))
		return true
	})
	mdri := x.Dat("GameConstants").GetRow("Id", "MonsterDamageReductionImprovement")
	mdriRatio := float64(mdri.Get("Value").(int64)) / float64(mdri.Get("Divisor").(int64))
	for i := 1; i <= 100; i++ {
		m.MonsterArmour = append(m.MonsterArmour, math.Floor((10+2*float64(i))*math.Pow(1+mdriRatio/100, float64(i))))
	}

	x.Dat("GameConstants").Rows(func(row *Row) bool {
		m.GameConstants = append(m.GameConstants, gamedata.IdValue{
			Id:    luaStr(row.Get("Id")),
			Value: float64(row.Get("Value").(int64)) / float64(row.Get("Divisor").(int64)),
		})
		return true
	})

	m.CharacterConstants = otConstants(x, "Metadata/Characters/Character.ot", true)
	m.MonsterConstants = otConstants(x, "Metadata/Monsters/Monster.ot", false)

	totemKeys := map[int64]bool{}
	x.Dat("SkillTotemVariations").Rows(func(vr *Row) bool {
		st := vr.Get("SkillTotem").(int64)
		if !totemKeys[st] {
			totemKeys[st] = true
			m.TotemLifeMult = append(m.TotemLifeMult, gamedata.IntMult{
				Id:   st,
				Mult: float64(vr.Get("MonsterVariety").(*Row).Get("LifeMultiplier").(int64)) / 100,
			})
		}
		return true
	})

	cachedEntry := map[string]bool{}
	x.Dat("MonsterVarieties").Rows(func(row *Row) bool {
		for _, mm := range row.Get("Mods").([]any) {
			mod := mm.(*Row)
			name := luaStr(row.Get("Name"))
			if luaStr(mod.Get("Id")) == "MonsterNecromancerRaisable" && !cachedEntry[name] {
				m.MonsterVarietyLifeMult = append(m.MonsterVarietyLifeMult, gamedata.NameMult{
					Name: name,
					Mult: float64(row.Get("LifeMultiplier").(int64)) / 100,
				})
				cachedEntry[name] = true
				break
			}
		}
		return true
	})

	x.Dat("MonsterMapDifficulty").Rows(func(row *Row) bool {
		m.MapLevelLifeMult = append(m.MapLevelLifeMult, gamedata.LevelMult{
			Level: row.Get("AreaLevel").(int64),
			Mult:  1 + float64(row.Get("LifePercentIncrease").(int64))/100,
		})
		return true
	})
	x.Dat("MonsterMapBossDifficulty").Rows(func(vr *Row) bool {
		lvl := vr.Get("AreaLevel").(int64)
		m.MapLevelBossLifeMult = append(m.MapLevelBossLifeMult, gamedata.LevelMult{
			Level: lvl, Mult: 1 + float64(vr.Get("BossLifePercentIncrease").(int64))/100,
		})
		m.MapLevelBossAilmentMult = append(m.MapLevelBossAilmentMult, gamedata.LevelMult{
			Level: lvl, Mult: (100 + float64(vr.Get("BossAilmentPercentDecrease").(int64))) / 100,
		})
		return true
	})
	x.Dat("VillageBalancePerLevelShared").Rows(func(row *Row) bool {
		m.GoldRespecPrices = append(m.GoldRespecPrices, row.Get("GoldRespec").(int64))
		return true
	})

	d.CurrencyNames = gamedata.CurrencyNames{}
	nameClean := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")
	x.Dat("CurrencyExchange").Rows(func(row *Row) bool {
		base := row.Get("BaseItemType").(*Row)
		name := luaStr(base.Get("Name"))
		if luaStr(base.Get("ItemClass").(*Row).Get("Id")) == "StackableCurrency" &&
			name != "" && !strings.Contains(name, "DNT") {
			d.CurrencyNames[luaStr(base.Get("Id"))] = nameClean.Replace(name)
		}
		return true
	})
	return d, nil
}
