package calc

import (
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// ItemInputOf projects a parsed item into the slice of its state the calc
// stages read. It is the bridge between the item package (which owns
// parsing) and BuildInput: whoever assembles a build - the native loader
// or the differential's fixture replay - projects through here, so the
// two agree by construction.
func ItemInputOf(it *item.Item) *ItemInput {
	in := &ItemInput{
		Name:             it.Name,
		ModSource:        strOrNil(it.ModSource),
		Title:            strOrNil(it.Title),
		BaseName:         strOrNil(it.BaseName),
		Type:             it.Type,
		Rarity:           it.Rarity,
		Corrupted:        trueOrNil(it.Corrupted),
		Shaper:           trueOrNil(it.Influence["shaper"]),
		Elder:            trueOrNil(it.Influence["elder"]),
		Adjudicator:      trueOrNil(it.Influence["adjudicator"]),
		Basilisk:         trueOrNil(it.Influence["basilisk"]),
		Crusader:         trueOrNil(it.Influence["crusader"]),
		Eyrie:            trueOrNil(it.Influence["eyrie"]),
		Foulborn:         &it.Foulborn,
		ClassRestriction: strOrNil(it.ClassRestriction),
		Limit:            it.Limit,
		Quality:          it.Quality,
	}
	if it.Base != nil {
		base := &ItemBaseInput{Type: strOrNil(it.Base.Type), SubType: strOrNil(it.Base.SubType)}
		if it.Base.Flask != nil {
			fb := &FlaskBaseInput{}
			if it.Base.Flask.Life != nil {
				fb.Life = util.Some(*it.Base.Flask.Life)
			}
			if it.Base.Flask.Mana != nil {
				fb.Mana = util.Some(*it.Base.Flask.Mana)
			}
			base.Flask = fb
		}
		in.Base = base
	}
	in.ModList = it.ModList
	in.SlotModList = it.SlotModList
	in.BaseModList = it.BaseModList
	if it.BuffModList != nil {
		in.BuffModList = it.BuffModList
	} else if it.BuffModListInit {
		// Item:BuildModList leaves buffModList an empty table on items
		// that reach the buff branch; empty and absent are distinct to
		// the stages that test it for presence.
		in.BuffModList = []*modparser.Mod{}
	}
	in.GrantedSkills = it.GrantedSkills
	in.Requirements = &it.Requirements
	sockets := make([]SocketInput, 0, len(it.Sockets))
	for _, s := range it.Sockets {
		sockets = append(sockets, SocketInput{Color: s.Color, Group: s.Group})
	}
	in.Sockets = sockets
	in.AbyssalSocketCount = numPtr(it.AbyssalSocketCount)
	in.SocketedJewelEffectModifier = numPtr(it.SocketedJewelEffectModifier)
	if it.JewelRadiusIndex != nil {
		in.JewelRadiusIndex = numPtr(float64(*it.JewelRadiusIndex))
	}
	if it.JewelData != nil {
		for _, fn := range it.JewelData.FuncList {
			in.FuncTypes = append(in.FuncTypes, fn.Type)
		}
	}
	in.JewelData = it.JewelData
	in.FlaskData = it.FlaskData
	in.TinctureData = it.TinctureData
	in.ArmourData = it.ArmourData
	if it.WeaponData != nil {
		wd := map[int]*item.WeaponData{}
		for i := 1; i <= 2; i++ {
			if side, ok := it.WeaponData[i]; ok {
				wd[i] = side
			}
		}
		in.WeaponData = wd
	}
	expl, other := []string{}, []string{}
	collect := func(lines []*item.ModLine, dst *[]string) {
		for _, v := range lines {
			if !v.Flag("disabled") && it.CheckModLineVariant(v) {
				*dst = append(*dst, v.Line)
			}
		}
	}
	collect(it.ExplicitModLines, &expl)
	collect(it.EnchantModLines, &other)
	collect(it.ScourgeModLines, &other)
	collect(it.ImplicitModLines, &other)
	collect(it.CrucibleModLines, &other)
	in.ExplicitLines = expl
	in.OtherLines = other
	return in
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func trueOrNil(b bool) *bool {
	if !b {
		return nil
	}
	return &b
}

func numPtr(v float64) *float64 { return &v }
