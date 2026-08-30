// data.bossSkills and data.bossSkillsList, from the bossData document.

package data

import (
	"fmt"
	"strconv"

	"github.com/MissingL-tter/missingPassives/data/schema"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

// BossSkillData is one data.bossSkills entry. A penetration left unset is
// the reference's `""` placeholder.
type BossSkillData struct {
	DamageType             string                       `lua:"DamageType"`
	DamageMultipliers      map[string][]float64         `lua:"DamageMultipliers"`
	UberDamageMultiplier   *float64                     `lua:"UberDamageMultiplier"`
	DamagePenetrations     map[string]util.Opt[float64] `lua:"DamagePenetrations"`
	UberDamagePenetrations map[string]util.Opt[float64] `lua:"UberDamagePenetrations"`
	Speed                  *float64                     `lua:"speed"`
	UberSpeed              *float64                     `lua:"UberSpeed"`
	CritChance             *float64                     `lua:"critChance"`
	EarlierUber            bool                         `lua:"earlierUber,omitempty"`
	AdditionalStats        *BossAdditionalStats         `lua:"additionalStats"`
	Tooltip                string                       `lua:"tooltip"`
}

type BossAdditionalStats struct {
	Base map[string]BossStat `lua:"base"`
	Uber map[string]BossStat `lua:"uber"`
}

// BossStat is one additional stat: a number, or the "flag" marker.
type BossStat struct {
	Value float64
	Flag  bool
}

// ValLabel is one bossSkillsList entry.
type ValLabel struct {
	Val   string `lua:"val"`
	Label string `lua:"label"`
}

// penValue resolves a pre-rendered penetration value: a number, or the
// literal `""` placeholder (absent).
func penValue(text string) (util.Opt[float64], error) {
	if text == "\"\"" {
		return util.Opt[float64]{}, nil
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return util.Opt[float64]{}, fmt.Errorf("data: bad penetration value %q", text)
	}
	return util.Some(n), nil
}

// statSetValues copies an additional-stat set into the runtime table.
func statSetValues(vals map[string]schema.BossStatValue) map[string]BossStat {
	out := map[string]BossStat{}
	for k, v := range vals {
		out[k] = BossStat{Value: v.Value, Flag: v.Flag}
	}
	return out
}

func penSet(pens []schema.PenEntry) (map[string]util.Opt[float64], error) {
	out := map[string]util.Opt[float64]{}
	for _, p := range pens {
		v, err := penValue(p.Text)
		if err != nil {
			return nil, err
		}
		out[p.Name] = v
	}
	return out, nil
}

func loadBossSkills(src schema.BossData) (map[string]BossSkillData, []ValLabel, error) {
	skills := map[string]BossSkillData{}
	for _, bs := range src.Skills {
		e := BossSkillData{
			DamageType:        bs.DamageType,
			DamageMultipliers: map[string][]float64{},
			Tooltip:           bs.Tooltip,
		}
		for _, dm := range bs.DamageMultipliers {
			e.DamageMultipliers[dm.Type] = []float64{dm.Min, dm.Spread}
		}
		e.UberDamageMultiplier = bs.UberDamageMultiplier
		var err error
		if bs.HasPen {
			if e.DamagePenetrations, err = penSet(bs.Pens); err != nil {
				return nil, nil, err
			}
			if bs.HasUberPen {
				if e.UberDamagePenetrations, err = penSet(bs.UberPens); err != nil {
					return nil, nil, err
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
	return skills, list, nil
}
