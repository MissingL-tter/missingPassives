// CalcOffence.lua L1659-1845: the skill cost table — base costs per
// resource, the conversion chains (Blood Magic, Petrified Blood, Whispers
// of Infinity, Hateforge) and the increased/more/efficiency application.
package calc

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// costEntry is one row of the reference's `costs` table.
type costEntry struct {
	typ                          string
	upfront                      bool
	percent                      bool
	unaffectedByGenericCostMults bool
	text                         string

	baseCost       float64
	baseCostRaw    float64
	hasBaseCostRaw bool // only the Mana row initialises baseCostRaw
	totalCost      float64
	baseCostNoMult float64
	finalBaseCost  float64
}

// setBaseCostRaw mirrors `costs[x].baseCostRaw = 0` in the conversion
// branches: on a row that never had the field, the assignment CREATES it,
// and 0 is truthy, so the second pass then emits that row's ...CostRaw.
func (e *costEntry) setBaseCostRaw(v float64) {
	e.baseCostRaw = v
	e.hasBaseCostRaw = true
}

var costOrder = []string{
	"Mana", "Life", "ES", "Soul", "Rage", "ManaPercent", "LifePercent",
	"ManaPerMinute", "LifePerMinute", "ManaPercentPerMinute",
	"LifePercentPerMinute", "ESPerMinute", "ESPercentPerMinute",
}

func newCostTable() map[string]*costEntry {
	return map[string]*costEntry{
		"Mana":                 {typ: "Mana", upfront: true, text: "mana", hasBaseCostRaw: true},
		"Life":                 {typ: "Life", upfront: true, text: "life"},
		"ES":                   {typ: "ES", upfront: true, text: "ES"},
		"Soul":                 {typ: "Soul", upfront: true, unaffectedByGenericCostMults: true, text: "soul"},
		"Rage":                 {typ: "Rage", upfront: true, text: "rage"},
		"ManaPercent":          {typ: "Mana", upfront: true, percent: true, text: "mana"},
		"LifePercent":          {typ: "Life", upfront: true, percent: true, text: "life"},
		"ManaPerMinute":        {typ: "Mana", text: "mana/s"},
		"LifePerMinute":        {typ: "Life", text: "life/s"},
		"ManaPercentPerMinute": {typ: "Mana", percent: true, text: "mana/s"},
		"LifePercentPerMinute": {typ: "Life", percent: true, text: "life/s"},
		"ESPerMinute":          {typ: "ES", text: "ES/s"},
		"ESPercentPerMinute":   {typ: "ES", percent: true, text: "ES/s"},
	}
}

// costDivisors indexes data.costs by resource name, mirroring the __index
// metatable Data.lua installs on the array.
var costDivisors map[string]float64

func costDivisor(resource string) float64 {
	if costDivisors == nil {
		costDivisors = map[string]float64{}
		for i := range data.Costs {
			costDivisors[data.Costs[i].Resource] = data.Costs[i].Divisor
		}
	}
	v, ok := costDivisors[resource]
	if !ok {
		panic("offence: no cost divisor for " + resource)
	}
	return v
}

// offenceCosts ports L1659-1845.
func (env *Env) offenceCosts(c *offenceCtx) {
	skillModList, skillCfg, output := c.skillModList, c.skillCfg, c.output
	activeSkill := c.activeSkill

	costs := newCostTable()
	c.costs = costs

	if !skillModList.Flag(skillCfg, "HasNoCost") {
		// Support cost multipliers are calculated first and rounded down
		// after 4 digits
		mult := floorDec(skillModList.More(skillCfg, "SupportManaMultiplier"), 4)
		// First pass to calculate base costs. Used for cost conversion
		// (e.g. Petrified Blood)
		additionalLifeCost := skillModList.Sum(modparser.Base, skillCfg, "ManaCostAsLifeCost") / 100
		additionalESCost := skillModList.Sum(modparser.Base, skillCfg, "ManaCostAsEnergyShieldCost") / 100
		hybridLifeCost := skillModList.Sum(modparser.Base, skillCfg, "HybridManaAndLifeCost_Life") / 100
		gel := activeSkill.ActiveEffect.GrantedEffectLevel

		for _, resource := range costOrder {
			val := costs[resource]
			skillCost, hasSkillCost := 0.0, false
			if ov, ok := skillModList.Override(skillCfg, "Base"+resource+"CostOverride"); ok {
				skillCost, hasSkillCost = valueNum(ov), true
			} else if gel != nil && gel.Cost != nil && !env.TriggeredCostWipes[gel] {
				// ProcessSocketGroup wipes the level's cost table for a
				// triggered item-granted skill; the replay records that mark
				// per-env instead of mutating the shared game data.
				if v, ok := gel.Cost[resource]; ok {
					skillCost, hasSkillCost = v, true
				}
			}
			baseCost := 0.0
			if hasSkillCost {
				baseCost = util.RoundHalfUp(skillCost/costDivisor(resource), 2)
			}
			// Flat cost from gem e.g. Divine Blessing
			baseCostNoMult := skillModList.Sum(modparser.Base, skillCfg, resource+"CostNoMult")
			divineBlessingCorrection := 0.0
			if val.upfront {
				baseCost += skillModList.Sum(modparser.Base, skillCfg, resource+"CostBase") // Rage Cost
				val.totalCost = skillModList.Sum(modparser.Base, skillCfg, resource+"Cost", "Cost")
				if resource == "Mana" && activeSkill.SkillTypes[modparser.SkillTypeReservationBecomesCost] && !val.percent &&
					!skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") && !skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") {
					// Divine Blessing / Totem auras
					reservedFlat := skillDataOrLevel(activeSkill.SkillData, gel, val.text+"ReservationFlat")
					baseCost += reservedFlat
					reservedPercent := skillDataOrLevel(activeSkill.SkillData, gel, val.text+"ReservationPercent")
					baseCost += math.Floor(output.N(resource) * reservedPercent / 100)
					// Divine Blessing / Totem aura skills that have a percent
					// reservation, round instead of floor the value. This
					// corrects the final result if it would round up
					divineBlessingCorrection = util.RoundHalfUp(output.N(resource)*reservedPercent/100*mult, 0) -
						math.Floor(output.N(resource)*reservedPercent/100*mult)
				}
			}
			val.baseCost += baseCost
			val.baseCostNoMult += baseCostNoMult
			val.finalBaseCost = (math.Floor(val.baseCost*mult) + val.baseCostNoMult) + divineBlessingCorrection
			if val.hasBaseCostRaw {
				val.baseCostRaw = val.baseCost*mult + val.baseCostNoMult + divineBlessingCorrection
			}
			switch val.typ {
			case "Life":
				manaType := costs[strings.Replace(resource, "Life", "Mana", -1)]
				if skillModList.Flag(skillCfg, "CostLifeInsteadOfMana") { // Blood Magic / Lifetap
					val.baseCost += manaType.baseCost
					val.baseCostNoMult += manaType.baseCostNoMult
					val.finalBaseCost += manaType.finalBaseCost
					manaType.baseCost = 0
					manaType.setBaseCostRaw(0)
					manaType.finalBaseCost = 0
					manaType.baseCostNoMult = 0
				} else if (additionalLifeCost > 0 || hybridLifeCost > 0) && !skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") {
					val.baseCost = manaType.baseCost
					val.finalBaseCost += util.RoundHalfUp(manaType.finalBaseCost*(hybridLifeCost+additionalLifeCost), 0)
				}
			case "ES":
				manaType := costs[strings.Replace(resource, "ES", "Mana", -1)]
				lifeType := costs[strings.Replace(resource, "ES", "Life", -1)]
				if skillModList.Flag(skillCfg, "CostESInsteadOfManaOrLife") { // Whispers of Infinity
					val.baseCost += manaType.baseCost + lifeType.baseCost
					val.baseCostNoMult += manaType.baseCostNoMult + lifeType.baseCostNoMult
					val.finalBaseCost += manaType.finalBaseCost + lifeType.finalBaseCost
					manaType.baseCost = 0
					manaType.setBaseCostRaw(0)
					manaType.finalBaseCost = 0
					manaType.baseCostNoMult = 0
					lifeType.baseCost = 0
					lifeType.setBaseCostRaw(0)
					lifeType.finalBaseCost = 0
					lifeType.baseCostNoMult = 0
				} else if additionalESCost > 0 {
					val.baseCost = manaType.baseCost
					val.finalBaseCost += util.RoundHalfUp(manaType.finalBaseCost*additionalESCost, 0)
				}
			case "Rage":
				if skillModList.Flag(skillCfg, "CostRageInsteadOfSouls") { // Hateforge
					soul := costs["Soul"]
					val.baseCost = soul.baseCost
					val.baseCostNoMult += soul.baseCostNoMult
					val.finalBaseCost = soul.baseCost
					mult = 1
					soul.baseCost = 0
					soul.baseCostNoMult = 0
					soul.finalBaseCost = 0
				}
			}
		}
		for _, key := range costOrder {
			val := costs[key]
			resource := key
			if !val.upfront {
				resource = strings.Replace(resource, "Minute", "Second", -1)
			}
			hasCost := val.baseCost > 0 || val.totalCost > 0 || val.baseCostNoMult > 0 || val.finalBaseCost > 0
			output.SetFlag(resource+"HasCost", hasCost)
			costName := resource + "Cost"
			costNameRaw := costName + "Raw"
			moreType := 1.0
			moreCost := 1.0
			inc := 0.0
			costEfficiency := Mod(skillModList, skillCfg, val.typ+"CostEfficiency", "CostEfficiency")
			if !val.unaffectedByGenericCostMults {
				cost := val.finalBaseCost
				moreType = skillModList.More(skillCfg, val.typ+"Cost")
				moreCost = skillModList.More(skillCfg, "Cost")
				inc = skillModList.Sum(modparser.Inc, skillCfg, val.typ+"Cost", "Cost")
				if val.hasBaseCostRaw {
					output.SetN(costNameRaw, math.Max(0, math.Max(0, (1+inc/100)*val.baseCostRaw*moreType*moreCost/costEfficiency)+val.totalCost))
				} else {
					output.Del(costNameRaw)
				}
				if inc < 0 {
					cost = math.Max(0, math.Ceil((1+inc/100)*cost))
				} else {
					cost = math.Max(0, math.Floor((1+inc/100)*cost))
				}
				if moreType < 1 {
					cost = math.Max(0, math.Ceil(moreType*cost))
				} else {
					cost = math.Max(0, math.Floor(moreType*cost))
				}
				if moreCost < 1 {
					cost = math.Max(0, math.Ceil(moreCost*cost))
				} else {
					cost = math.Max(0, math.Floor(moreCost*cost))
				}
				// Apply cost efficiency (similar to reservation efficiency)
				cost = math.Max(0, cost/costEfficiency)
				cost = math.Max(0, cost+val.totalCost)
				if val.typ == "Mana" && hybridLifeCost > 0 { // Life/Mana Mastery
					cost = math.Max(0, math.Floor((1-hybridLifeCost)*cost))
					if val.hasBaseCostRaw {
						output.SetN(costNameRaw, math.Max(0, (1-hybridLifeCost)*output.N(costNameRaw)))
					}
				}
				output.SetN(costName, cost)
			} else {
				moreType = skillModList.More(skillCfg, val.typ+"Cost")
				inc = skillModList.Sum(modparser.Inc, skillCfg, val.typ+"Cost")
				cost := math.Floor(val.baseCost + val.baseCostNoMult)
				cost = math.Max(0, (1+inc/100)*cost)
				cost = math.Max(0, moreType*cost)
				// Apply cost efficiency for unaffected costs too
				cost = math.Max(0, cost/costEfficiency)
				cost = math.Max(0, cost+val.totalCost)
				output.SetN(costName, cost)
				if val.hasBaseCostRaw {
					output.SetN(costNameRaw, math.Max(0, math.Max(0, (1+inc/100)*(val.baseCostRaw+val.baseCostNoMult)*moreType/costEfficiency)+val.totalCost))
				} else {
					output.Del(costNameRaw)
				}
			}
		}
	}

	// account for Sacrificial Zeal
	// Note: Sacrificial Zeal grants Added Spell Physical Damage equal to 25%
	// of the Skill's Mana Cost, and causes you to take Physical Damage over
	// Time, for 4 seconds
	if skillModList.Flag(nil, "Condition:SacrificialZeal") && output.Flag("ManaHasCost") {
		multiplier := 0.25
		skillModList.AddMod(newModSF("PhysicalMin", modparser.Base, modparser.Num(math.Floor(output.N("ManaCost")*multiplier)), "Sacrificial Zeal", modparser.FlagSpell, modparser.KeywordNone))
		skillModList.AddMod(newModSF("PhysicalMax", modparser.Base, modparser.Num(math.Floor(output.N("ManaCost")*multiplier)), "Sacrificial Zeal", modparser.FlagSpell, modparser.KeywordNone))
	}

	env.runSkillFunc(c, data.CallbackPreDamage)

	env.offenceConversion(c)
}

// skillDataOrLevel is `skillData[key] or grantedEffectLevel[key] or 0`,
// honouring a present 0 in either (0 is truthy in Lua).
func skillDataOrLevel(sd *SkillData, gel *data.SkillLevel, key string) float64 {
	if v := sd.Get(key); v.Truthy() {
		return v.Num()
	}
	if v, ok := lvlExtra(gel, key); ok {
		return v
	}
	return 0
}
