// Port of .archive/src/Modules/CalcOffence.lua, staged. This file holds the
// module-level helpers calcs.offence builds on: the damage-type flag sets,
// the recursive conversion-aware damage calculation, and the radius maths.
package calc

import (
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// dmgTypeFlagOrder and dmgTypeFlagBits mirror the reference's dmgTypeFlags:
// Elemental (0x0E) deliberately overlaps Lightning|Cold|Fire.
var dmgTypeFlagOrder = []string{"Physical", "Lightning", "Cold", "Fire", "Elemental", "Chaos"}

var dmgTypeFlagBits = map[string]int64{
	"Physical":  0x01,
	"Lightning": 0x02,
	"Cold":      0x04,
	"Fire":      0x08,
	"Elemental": 0x0E,
	"Chaos":     0x10,
}

// damageStatsForTypes memoizes the modifier-name sets calcDamage sums over,
// replacing the reference's __index metatable cache.
var damageStatsForTypes = map[int64][]string{}

func damageStatsFor(flags int64) []string {
	if names, ok := damageStatsForTypes[flags]; ok {
		return names
	}
	names := []string{"Damage"}
	for _, typ := range dmgTypeFlagOrder {
		if flags&dmgTypeFlagBits[typ] != 0 {
			names = append(names, typ+"Damage")
		}
	}
	damageStatsForTypes[flags] = names
	return names
}

// conversionEntry is one row of activeSkill.conversionTable: how much of
// this damage type converts or is gained as each other type, plus the
// multiplier left for the type itself.
type conversionEntry struct {
	Conversion map[string]float64
	Gain       map[string]float64
	Mult       float64
	// combined conversion+gain per destination type, which is what
	// calcDamage reads through conversionTable[from][to]
	Total map[string]float64
}

type conversionTable map[string]*conversionEntry

// to reports the combined convert+gain multiplier from one type to another.
func (ct conversionTable) to(from, to string) float64 {
	e := ct[from]
	if e == nil {
		return 0
	}
	return e.Total[to]
}

// calcDamage ports the local calcDamage: the min/max damage for one type,
// recursing into the types that precede it in the conversion sequence.
func (env *Env) calcDamage(activeSkill *ActiveSkill, output modstore.Output, cfg *modstore.Cfg,
	damageType string, typeFlags int64, ct conversionTable) (float64, float64) {
	skillModList := activeSkill.SkillModList
	typeFlags |= dmgTypeFlagBits[damageType]

	// Calculate conversions
	addMin, addMax := 0.0, 0.0
	for _, otherType := range dmgTypeList {
		if otherType == damageType {
			// Damage can only be converted from damage types that precede
			// this one in the conversion sequence, so stop here
			break
		}
		convMult := ct.to(otherType, damageType)
		if convMult > 0 {
			// Damage is being converted/gained from the other damage type
			min, max := env.calcDamage(activeSkill, output, cfg, otherType, typeFlags, ct)
			addMin += min * convMult
			addMax += max * convMult
		}
	}
	if addMin != 0 && addMax != 0 {
		addMin = util.RoundHalfUp(addMin, 0)
		addMax = util.RoundHalfUp(addMax, 0)
	}

	baseMin := output.N(damageType + "MinBase")
	baseMax := output.N(damageType + "MaxBase")
	if baseMin == 0 && baseMax == 0 {
		// No base damage for this type, don't need to calculate modifiers
		return addMin, addMax
	}

	// Combine modifiers
	modNames := damageStatsFor(typeFlags)
	inc := 1 + skillModList.Sum(modparser.Inc, cfg, modNames...)/100
	more := skillModList.More(cfg, modNames...)
	genericMoreMinDamage := skillModList.More(cfg, "MinDamage")
	genericMoreMaxDamage := skillModList.More(cfg, "MaxDamage")
	moreMinDamage := skillModList.More(cfg, "Min"+damageType+"Damage")
	moreMaxDamage := skillModList.More(cfg, "Max"+damageType+"Damage")
	incMinDamage := 1 + skillModList.Sum(modparser.Inc, cfg, "Min"+damageType+"Damage")/100
	incMaxDamage := 1 + skillModList.Sum(modparser.Inc, cfg, "Max"+damageType+"Damage")/100

	return util.RoundHalfUp(((baseMin*inc*more)*genericMoreMinDamage+addMin)*moreMinDamage*incMinDamage, 0),
		util.RoundHalfUp(((baseMax*inc*more)*genericMoreMaxDamage+addMax)*moreMaxDamage*incMaxDamage, 0)
}

// calcAilmentSourceDamage ports the local of the same name: the damage left
// on this type after conversion away from it.
func (env *Env) calcAilmentSourceDamage(activeSkill *ActiveSkill, output modstore.Output, cfg *modstore.Cfg,
	damageType string, typeFlags int64, ct conversionTable) (float64, float64) {
	min, max := env.calcDamage(activeSkill, output, cfg, damageType, typeFlags, ct)
	convMult := 1.0
	if e := ct[damageType]; e != nil {
		convMult = e.Mult
	}
	return min * convMult, max * convMult
}

// calcRadius ports the local calcRadius.
func calcRadius(baseRadius, areaMod float64) float64 {
	return math.Floor(baseRadius * math.Floor(100*math.Sqrt(areaMod)) / 100)
}

// calcMoltenStrikeTertiaryRadius ports the local of the same name: PoE is
// assumed to round only at the end.
func calcMoltenStrikeTertiaryRadius(baseRadius, deadzoneRadius, areaMod, speedMod float64) float64 {
	maxDistIgnoringSpeed := math.Sqrt(baseRadius*baseRadius*areaMod - deadzoneRadius*deadzoneRadius*(areaMod-1))
	return math.Floor((maxDistIgnoringSpeed-deadzoneRadius)*speedMod + deadzoneRadius)
}

// buildConversionTable ports the local buildConversionTable: per damage
// type, how much converts (global + skill) and is gained as each later type
// in the sequence, with the scaling the reference applies when conversion
// exceeds 100%.
func (env *Env) buildConversionTable(activeSkill *ActiveSkill, cfg *modstore.Cfg) conversionTable {
	skillModList := activeSkill.SkillModList
	ct := conversionTable{}
	for damageTypeIndex := 0; damageTypeIndex < 4; damageTypeIndex++ {
		damageType := dmgTypeList[damageTypeIndex]
		globalConv := map[string]float64{}
		skillConv := map[string]float64{}
		add := map[string]float64{}
		globalTotal, skillTotal := 0.0, 0.0
		for otherTypeIndex := damageTypeIndex + 1; otherTypeIndex < 5; otherTypeIndex++ {
			// For all possible destination types, check for global and skill conversions
			otherType := dmgTypeList[otherTypeIndex]
			convNames := []string{damageType + "DamageConvertTo" + otherType}
			gainNames := []string{damageType + "DamageGainAs" + otherType}
			if isElementalRes[damageType] {
				convNames = append(convNames, "ElementalDamageConvertTo"+otherType)
				gainNames = append(gainNames, "ElementalDamageGainAs"+otherType)
			}
			if damageType != "Chaos" {
				convNames = append(convNames, "NonChaosDamageConvertTo"+otherType)
				gainNames = append(gainNames, "NonChaosDamageGainAs"+otherType)
			}
			globalConv[otherType] = math.Max(skillModList.Sum(modparser.Base, cfg, convNames...), 0)
			globalTotal += globalConv[otherType]
			skillConv[otherType] = math.Max(skillModList.Sum(modparser.Base, cfg, "Skill"+damageType+"DamageConvertTo"+otherType), 0)
			skillTotal += skillConv[otherType]
			add[otherType] = math.Max(skillModList.Sum(modparser.Base, cfg, gainNames...), 0)
		}
		if skillTotal > 100 {
			// Skill conversion exceeds 100%, scale it down and remove
			// non-skill conversions
			factor := 100 / skillTotal
			for typ := range skillConv {
				skillConv[typ] *= factor
			}
			for typ := range globalConv {
				globalConv[typ] = 0
			}
		} else if globalTotal+skillTotal > 100 {
			// Conversion exceeds 100%, scale down non-skill conversions
			factor := (100 - skillTotal) / globalTotal
			for typ := range globalConv {
				globalConv[typ] *= factor
			}
			globalTotal *= factor
		}
		entry := &conversionEntry{
			Conversion: map[string]float64{},
			Gain:       map[string]float64{},
			Total:      map[string]float64{},
		}
		for typ := range globalConv {
			entry.Conversion[typ] = (globalConv[typ] + skillConv[typ]) / 100
			entry.Gain[typ] = add[typ] / 100
			entry.Total[typ] = (globalConv[typ] + skillConv[typ] + add[typ]) / 100
		}
		entry.Mult = 1 - math.Min((globalTotal+skillTotal)/100, 1)
		ct[damageType] = entry
	}
	ct["Chaos"] = &conversionEntry{
		Conversion: map[string]float64{},
		Gain:       map[string]float64{},
		Total:      map[string]float64{},
		Mult:       1,
	}
	return ct
}
