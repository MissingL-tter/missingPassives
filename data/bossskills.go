// data.bossSkills and data.bossSkillsList, from the bossData document.

package data

import (
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/data/schema"
)

// BossSkillData is one data.bossSkills entry.
type BossSkillData struct {
	DamageType             string               `lua:"DamageType"`
	DamageMultipliers      map[string][]float64 `lua:"DamageMultipliers"`
	UberDamageMultiplier   *float64             `lua:"UberDamageMultiplier"`
	DamagePenetrations     map[string]any       `lua:"DamagePenetrations"`
	UberDamagePenetrations map[string]any       `lua:"UberDamagePenetrations"`
	Speed                  *float64             `lua:"speed"`
	UberSpeed              *float64             `lua:"UberSpeed"`
	CritChance             *float64             `lua:"critChance"`
	EarlierUber            bool                 `lua:"earlierUber,omitempty"`
	AdditionalStats        *BossAdditionalStats `lua:"additionalStats"`
	Tooltip                string               `lua:"tooltip"`
}

type BossAdditionalStats struct {
	Base map[string]any `lua:"base"`
	Uber map[string]any `lua:"uber"`
}

// ValLabel is one bossSkillsList entry.
type ValLabel struct {
	Val   string `lua:"val"`
	Label string `lua:"label"`
}

// penValue resolves a pre-rendered penetration value: a number, or the
// literal `""`.
func penValue(text string) any {
	if text == "\"\"" {
		return ""
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		panic("data: bad penetration value " + text)
	}
	return n
}

// statSetValues resolves a pre-rendered additional-stat set: numbers, or
// the quoted "flag" literal.
func statSetValues(vals map[string]string) map[string]any {
	out := map[string]any{}
	for k, text := range vals {
		if strings.HasPrefix(text, "\"") {
			out[k] = strings.Trim(text, "\"")
		} else {
			n, err := strconv.ParseFloat(text, 64)
			if err != nil {
				panic("data: bad additional stat value " + text)
			}
			out[k] = n
		}
	}
	return out
}

func loadBossSkills(src schema.BossData) (map[string]BossSkillData, []ValLabel) {
	skills := map[string]BossSkillData{}
	for _, bs := range src.Skills {
		e := BossSkillData{
			DamageType:        bs.DamageType,
			DamageMultipliers: map[string][]float64{},
			Tooltip:           luaStringLiteral(bs.Tooltip),
		}
		for _, dm := range bs.DamageMultipliers {
			e.DamageMultipliers[dm.Type] = []float64{dm.Min, dm.Spread}
		}
		e.UberDamageMultiplier = bs.UberDamageMultiplier
		if bs.HasPen {
			e.DamagePenetrations = map[string]any{}
			for _, p := range bs.Pens {
				e.DamagePenetrations[p.Name] = penValue(p.Text)
			}
			if bs.HasUberPen {
				e.UberDamagePenetrations = map[string]any{}
				for _, p := range bs.UberPens {
					e.UberDamagePenetrations[p.Name] = penValue(p.Text)
				}
			}
		}
		if bs.Speed != 700 {
			v := bs.Speed
			e.Speed = &v
		}
		if bs.UberSpeed != nil && *bs.UberSpeed != 700 {
			e.UberSpeed = bs.UberSpeed
		}
		if bs.CritChance != 5 {
			v := float64(bs.CritChance)
			e.CritChance = &v
		}
		e.EarlierUber = bs.EarlierUber
		if bs.HasAdditional {
			as := &BossAdditionalStats{}
			if bs.BaseCount > 0 {
				as.Base = statSetValues(bs.BaseVals)
			}
			if bs.UberCount > 0 {
				as.Uber = statSetValues(bs.UberVals)
			}
			e.AdditionalStats = as
		}
		skills[bs.Key] = e
	}

	// The file returns the skill table and then the first #skillList table.
	var list []ValLabel
	if len(src.SkillLists) > 0 {
		list = append(list, ValLabel{Val: "None", Label: "None"})
		for _, name := range src.SkillLists[0] {
			list = append(list, ValLabel{Val: name, Label: name})
		}
	}
	return skills, list
}
