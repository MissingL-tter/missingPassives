// CalcDefence.lua L2568-2733: numberOfHitsToDie, the iterative solver that
// drains the pools hit by hit (recursing with an accelerated multiplier)
// until life reaches zero.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"
)

// damageIn models the DamageIn table: per-type damage plus the control
// fields. Optional numeric fields are pointers because the reference tests
// them for presence, not value.
type damageIn struct {
	dmg map[string]float64

	cycles     float64
	iterations float64

	TrackRecoupable       bool
	TrackLifeLossOverTime bool

	wardBypass    float64
	wardBypassSet bool

	GainWhenHit         bool
	LifeWhenHit         float64
	ManaWhenHit         float64
	EnergyShieldWhenHit float64

	MissingLifeBeforeEnemyHit *float64
	MissingManaBeforeEnemyHit *float64

	LimitEHPSpeedup bool
	cyclesRan       bool
}

func newDamageIn() *damageIn {
	return &damageIn{dmg: map[string]float64{}, cycles: 1}
}

// numberOfHitsToDie ports the local of the same name. It mutates in.cycles,
// in.iterations, in.cyclesRan and in.wardBypass exactly as the reference
// mutates its table, because the caller reads them back.
func (env *Env) numberOfHitsToDie(in *damageIn, actor *performActor) float64 {
	modDB := actor.db
	output := actor.output

	numHits := 0.0
	if in.cycles == 0 {
		in.cycles = 1
	}

	// check damage in isn't 0 and that ward doesn't mitigate all damage
	for _, damageType := range dmgTypeList {
		numHits += in.dmg[damageType]
	}
	if numHits == 0 {
		return math.Inf(1)
	}

	wardAvoidBreakChance := math.Min(modDB.Sum(modparser.Base, nil, "WardAvoidBreakChance")/100, 1)
	if modDB.Flag(nil, "Condition:WardNotBreak") {
		wardAvoidBreakChance = 1
	}
	if wardAvoidBreakChance == 1 && output.N("Ward") > 0 && numHits < output.N("Ward") {
		return math.Inf(1)
	}
	numHits = 0

	ward := output.N("Ward")
	// don't apply non-perma ward for speed up calcs as it won't zero it
	// correctly per hit
	if wardAvoidBreakChance < 1 && in.cycles > 1 {
		ward = 0
	}
	aegis := map[string]float64{
		"shared":          output.N("sharedAegis"),
		"sharedElemental": output.N("sharedElementalAegis"),
	}
	guard := map[string]float64{"shared": output.N("sharedGuardAbsorb")}
	for _, damageType := range dmgTypeList {
		aegis[damageType] = output.N(damageType + "Aegis")
		guard[damageType] = output.N(damageType + "GuardAbsorb")
	}
	alliesTakenBeforeYou := buildAllyLifePools(output)

	esCap := output.N("EnergyShieldRecoveryCap")
	manaStart := output.N("ManaUnreserved")
	lifeStart := output.N("LifeRecoverable")
	poolTable := &poolSet{
		AlliesTakenBeforeYou:          alliesTakenBeforeYou,
		Aegis:                         aegis,
		Guard:                         guard,
		Ward:                          &ward,
		EnergyShield:                  &esCap,
		Mana:                          &manaStart,
		Life:                          &lifeStart,
		LifeLossLostOverTime:          output.N("LifeLossLostOverTime"),
		LifeBelowHalfLossLostOverTime: output.N("LifeBelowHalfLossLostOverTime"),
		damageTakenThatCanBeRecouped:  map[string]float64{},
	}
	// live pool values threaded through the loop
	poolLife := lifeStart
	poolMana := manaStart
	poolES := esCap
	poolWardActiveChance := 0.0
	if ward > 0 {
		poolWardActiveChance = 1
	}
	var lastResult *poolResult

	if in.cycles != 1 {
		in.TrackRecoupable = false
		in.TrackLifeLossOverTime = false
	}
	if !in.wardBypassSet {
		in.wardBypass = modDB.Sum(modparser.Base, nil, "WardBypass")
		in.wardBypassSet = true
	}

	vaalArcticArmourHitsLeft := output.N("VaalArcticArmourLife")
	if in.cycles > 1 {
		vaalArcticArmourHitsLeft = 0
	}

	iterationMultiplier := 1.0
	damageTotal := 0.0
	maxDamage := data.Misc.EhpCalcMaxDamage
	maxIterations := data.Misc.EhpCalcMaxIterationsToCalc
	for poolLife > 0 && in.iterations < maxIterations {
		in.iterations++
		damage := map[string]float64{}
		damageTotal = 0
		vaalArcticArmourMultiplier := 1.0
		if vaalArcticArmourHitsLeft > 0 {
			vaalArcticArmourMultiplier = 1 - output.N("VaalArcticArmourMitigation")*math.Min(vaalArcticArmourHitsLeft/iterationMultiplier, 1)
		}
		vaalArcticArmourHitsLeft -= iterationMultiplier
		for _, damageType := range dmgTypeList {
			dmg := in.dmg[damageType]
			if dmg > 0 {
				damage[damageType] = dmg * iterationMultiplier * vaalArcticArmourMultiplier
			}
			damageTotal += dmg
		}
		if (in.GainWhenHit || in.MissingLifeBeforeEnemyHit != nil || in.MissingManaBeforeEnemyHit != nil) &&
			(iterationMultiplier > 1 || in.cycles > 1) {
			gainMult := iterationMultiplier * in.cycles
			if in.MissingLifeBeforeEnemyHit != nil {
				poolLife = math.Min(poolLife+*in.MissingLifeBeforeEnemyHit*(output.N("LifeUnreserved")-poolLife)*(gainMult-1)/100,
					gainMult*output.N("LifeRecoverable"))
			}
			if in.MissingManaBeforeEnemyHit != nil {
				poolMana = math.Min(poolMana+*in.MissingManaBeforeEnemyHit*(output.N("ManaUnreserved")-poolMana)*(gainMult-1)/100,
					gainMult*output.N("ManaUnreserved"))
			}
			if in.GainWhenHit {
				poolLife = math.Min(poolLife+in.LifeWhenHit*(gainMult-1), gainMult*output.N("LifeRecoverable"))
				poolMana = math.Min(poolMana+in.ManaWhenHit*(gainMult-1), gainMult*output.N("ManaUnreserved"))
				poolES = math.Min(poolES+in.EnergyShieldWhenHit*(gainMult-1), gainMult*output.N("EnergyShieldRecoveryCap"))
			}
		}
		if in.MissingLifeBeforeEnemyHit != nil && poolLife > 0 {
			poolLife = math.Min(poolLife+*in.MissingLifeBeforeEnemyHit*(output.N("LifeUnreserved")-poolLife)/100,
				output.N("LifeRecoverable"))
		}
		if in.MissingManaBeforeEnemyHit != nil && poolMana > 0 {
			poolMana = math.Min(poolMana+*in.MissingManaBeforeEnemyHit*(output.N("ManaUnreserved")-poolMana)/100,
				output.N("ManaUnreserved"))
		}
		poolTable.Life = &poolLife
		poolTable.Mana = &poolMana
		poolTable.EnergyShield = &poolES
		poolTable.Ward = &ward
		wac := poolWardActiveChance
		poolTable.WardActiveChance = &wac
		lastResult = env.reducePoolsByDamage(poolTable, damage, actor)
		poolLife = lastResult.Life
		poolMana = lastResult.Mana
		poolES = lastResult.EnergyShield
		ward = lastResult.Ward
		poolWardActiveChance = lastResult.WardActiveChance
		poolTable = &poolSet{
			AlliesTakenBeforeYou:          lastResult.AlliesTakenBeforeYou,
			Aegis:                         lastResult.Aegis,
			Guard:                         lastResult.Guard,
			LifeLossLostOverTime:          lastResult.LifeLossLostOverTime,
			LifeBelowHalfLossLostOverTime: lastResult.LifeBelowHalfLossLostOverTime,
			damageTakenThatCanBeRecouped:  lastResult.damageTakenThatCanBeRecouped,
		}

		// If still living and the amount of damage exceeds maximum
		// threshold we survived infinite number of hits.
		if poolLife > 0 && damageTotal >= maxDamage {
			return math.Inf(1)
		}
		if in.GainWhenHit && poolLife > 0 {
			poolLife = math.Min(poolLife+in.LifeWhenHit, output.N("LifeRecoverable"))
			poolMana = math.Min(poolMana+in.ManaWhenHit, output.N("ManaUnreserved"))
			poolES = math.Min(poolES+in.EnergyShieldWhenHit, output.N("EnergyShieldRecoveryCap"))
		}
		iterationMultiplier = 1
		// to speed it up, run recursively but accelerated.
		// MoM/life-loss-prevention mechanics can collapse too many hits into
		// one resulting in eHP jumps so we slow the acceleration.
		speedUp := data.Misc.EhpCalcSpeedUp
		if in.LimitEHPSpeedup {
			speedUp = 4
		}
		wardAvoidBreakActive := wardAvoidBreakChance < 1 && poolWardActiveChance > 0.01
		if !in.cyclesRan && !wardAvoidBreakActive && poolLife > 0 && in.iterations < maxIterations {
			inner := newDamageIn()
			for _, damageType := range dmgTypeList {
				inner.dmg[damageType] = in.dmg[damageType] * speedUp
			}
			inner.LimitEHPSpeedup = in.LimitEHPSpeedup
			if in.GainWhenHit {
				inner.GainWhenHit = true
				inner.LifeWhenHit = in.LifeWhenHit
				inner.ManaWhenHit = in.ManaWhenHit
				inner.EnergyShieldWhenHit = in.EnergyShieldWhenHit
			}
			inner.cycles = in.cycles * speedUp
			inner.iterations = in.iterations
			hits := env.numberOfHitsToDie(inner, actor)
			if math.IsInf(hits, 1) {
				// avoid unnecessary calculations if we know we survive
				// infinite hits
				return math.Inf(1)
			}
			iterationMultiplier = math.Max((hits-1)*speedUp-1, 1)
			in.iterations = inner.iterations
			in.cyclesRan = true
		}
		numHits += iterationMultiplier
	}
	if lastResult != nil {
		if in.TrackRecoupable {
			for damageType, recoupable := range lastResult.damageTakenThatCanBeRecouped {
				output.SetN(damageType+"RecoupableDamageTaken", output.N(damageType+"RecoupableDamageTaken")+recoupable)
			}
		}
		if in.TrackLifeLossOverTime {
			output.SetN("LifeLossLostOverTime", output.N("LifeLossLostOverTime")+lastResult.LifeLossLostOverTime)
			output.SetN("LifeBelowHalfLossLostOverTime", output.N("LifeBelowHalfLossLostOverTime")+lastResult.LifeBelowHalfLossLostOverTime)
		}
		// Don't count overkill damage and only on final pass as to not
		// break speedup.
		if poolLife == 0 && in.cycles == 1 {
			numHits -= lastResult.OverkillDamage / damageTotal
		}
	}
	// Recalculate total hit damage
	damageTotal = 0
	for _, damageType := range dmgTypeList {
		damageTotal += in.dmg[damageType] * numHits
	}
	if poolLife >= 0 && damageTotal >= maxDamage {
		return math.Inf(1)
	}
	if math.IsNaN(numHits) {
		return 0
	}
	return math.Max(numHits, 0)
}
