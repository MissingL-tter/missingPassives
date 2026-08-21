// CalcDefence.lua L2568-2733: numberOfHitsToDie, the iterative solver that
// drains the pools hit by hit (recursing with an accelerated multiplier)
// until life reaches zero.
package calc

import "math"

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
	d := env.Data

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

	wardAvoidBreakChance := math.Min(modDB.Sum("BASE", nil, "WardAvoidBreakChance")/100, 1)
	if modDB.Flag(nil, "Condition:WardNotBreak") {
		wardAvoidBreakChance = 1
	}
	if wardAvoidBreakChance == 1 && outNum(output, "Ward") > 0 && numHits < outNum(output, "Ward") {
		return math.Inf(1)
	}
	numHits = 0

	ward := outNum(output, "Ward")
	// don't apply non-perma ward for speed up calcs as it won't zero it
	// correctly per hit
	if wardAvoidBreakChance < 1 && in.cycles > 1 {
		ward = 0
	}
	aegis := map[string]float64{
		"shared":          outNum(output, "sharedAegis"),
		"sharedElemental": outNum(output, "sharedElementalAegis"),
	}
	guard := map[string]float64{"shared": outNum(output, "sharedGuardAbsorb")}
	for _, damageType := range dmgTypeList {
		aegis[damageType] = outNum(output, damageType+"Aegis")
		guard[damageType] = outNum(output, damageType+"GuardAbsorb")
	}
	alliesTakenBeforeYou := buildAllyLifePools(output)

	esCap := outNum(output, "EnergyShieldRecoveryCap")
	manaStart := outNum(output, "ManaUnreserved")
	lifeStart := outNum(output, "LifeRecoverable")
	poolTable := &poolSet{
		AlliesTakenBeforeYou:          alliesTakenBeforeYou,
		Aegis:                         aegis,
		Guard:                         guard,
		Ward:                          &ward,
		EnergyShield:                  &esCap,
		Mana:                          &manaStart,
		Life:                          &lifeStart,
		LifeLossLostOverTime:          outNum(output, "LifeLossLostOverTime"),
		LifeBelowHalfLossLostOverTime: outNum(output, "LifeBelowHalfLossLostOverTime"),
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
		in.wardBypass = modDB.Sum("BASE", nil, "WardBypass")
		in.wardBypassSet = true
	}

	vaalArcticArmourHitsLeft := outNum(output, "VaalArcticArmourLife")
	if in.cycles > 1 {
		vaalArcticArmourHitsLeft = 0
	}

	iterationMultiplier := 1.0
	damageTotal := 0.0
	maxDamage := d.Misc.EhpCalcMaxDamage
	maxIterations := d.Misc.EhpCalcMaxIterationsToCalc
	for poolLife > 0 && in.iterations < maxIterations {
		in.iterations++
		damage := map[string]float64{}
		damageTotal = 0
		vaalArcticArmourMultiplier := 1.0
		if vaalArcticArmourHitsLeft > 0 {
			vaalArcticArmourMultiplier = 1 - outNum(output, "VaalArcticArmourMitigation")*math.Min(vaalArcticArmourHitsLeft/iterationMultiplier, 1)
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
				poolLife = math.Min(poolLife+*in.MissingLifeBeforeEnemyHit*(outNum(output, "LifeUnreserved")-poolLife)*(gainMult-1)/100,
					gainMult*outNum(output, "LifeRecoverable"))
			}
			if in.MissingManaBeforeEnemyHit != nil {
				poolMana = math.Min(poolMana+*in.MissingManaBeforeEnemyHit*(outNum(output, "ManaUnreserved")-poolMana)*(gainMult-1)/100,
					gainMult*outNum(output, "ManaUnreserved"))
			}
			if in.GainWhenHit {
				poolLife = math.Min(poolLife+in.LifeWhenHit*(gainMult-1), gainMult*outNum(output, "LifeRecoverable"))
				poolMana = math.Min(poolMana+in.ManaWhenHit*(gainMult-1), gainMult*outNum(output, "ManaUnreserved"))
				poolES = math.Min(poolES+in.EnergyShieldWhenHit*(gainMult-1), gainMult*outNum(output, "EnergyShieldRecoveryCap"))
			}
		}
		if in.MissingLifeBeforeEnemyHit != nil && poolLife > 0 {
			poolLife = math.Min(poolLife+*in.MissingLifeBeforeEnemyHit*(outNum(output, "LifeUnreserved")-poolLife)/100,
				outNum(output, "LifeRecoverable"))
		}
		if in.MissingManaBeforeEnemyHit != nil && poolMana > 0 {
			poolMana = math.Min(poolMana+*in.MissingManaBeforeEnemyHit*(outNum(output, "ManaUnreserved")-poolMana)/100,
				outNum(output, "ManaUnreserved"))
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
			poolLife = math.Min(poolLife+in.LifeWhenHit, outNum(output, "LifeRecoverable"))
			poolMana = math.Min(poolMana+in.ManaWhenHit, outNum(output, "ManaUnreserved"))
			poolES = math.Min(poolES+in.EnergyShieldWhenHit, outNum(output, "EnergyShieldRecoveryCap"))
		}
		iterationMultiplier = 1
		// to speed it up, run recursively but accelerated.
		// MoM/life-loss-prevention mechanics can collapse too many hits into
		// one resulting in eHP jumps so we slow the acceleration.
		speedUp := d.Misc.EhpCalcSpeedUp
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
				output[damageType+"RecoupableDamageTaken"] = outNum(output, damageType+"RecoupableDamageTaken") + recoupable
			}
		}
		if in.TrackLifeLossOverTime {
			output["LifeLossLostOverTime"] = outNum(output, "LifeLossLostOverTime") + lastResult.LifeLossLostOverTime
			output["LifeBelowHalfLossLostOverTime"] = outNum(output, "LifeBelowHalfLossLostOverTime") + lastResult.LifeBelowHalfLossLostOverTime
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
