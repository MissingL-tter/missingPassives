// CalcDefence.lua L188-421: calcs.reducePoolsByDamage. Drains the ally,
// aegis, guard, ward, energy shield, mana and life pools by an incoming
// damage table, in that order, and reports what is left.
//
// Every pairs(damageTable) loop in the reference is order-independent (a
// sum, a per-key init, a subtract-from-all, or a min), so the Go maps need
// no ordering; only the outer dmgTypeList walk is sequenced.
package calc

import (
	"github.com/MissingL-tter/missingPassives/modparser"
	"math"
)

// poolSet is the poolTable argument: nil fields fall back to the actor's
// output, matching the reference's `poolTbl.X or output.Y or 0` chains.
type poolSet struct {
	AlliesTakenBeforeYou          map[string]*allyPoolState
	Aegis                         map[string]float64
	Guard                         map[string]float64
	Ward                          *float64
	WardActiveChance              *float64
	EnergyShield                  *float64
	Mana                          *float64
	Life                          *float64
	LifeLossLostOverTime          float64
	LifeBelowHalfLossLostOverTime float64
	damageTakenThatCanBeRecouped  map[string]float64
}

// poolResult is reducePoolsByDamage's return table.
type poolResult struct {
	AlliesTakenBeforeYou          map[string]*allyPoolState
	Aegis                         map[string]float64
	Guard                         map[string]float64
	Ward                          float64
	WardActiveChance              float64
	EnergyShield                  float64
	Mana                          float64
	Life                          float64
	damageTakenThatCanBeRecouped  map[string]float64
	LifeLossLostOverTime          float64
	LifeBelowHalfLossLostOverTime float64
	OverkillDamage                float64
	hitPoolRemaining              float64
	resourcesLostToTypeDamage     map[string]map[string]float64
}

func fOr(p *float64, fallback float64) float64 {
	if p != nil {
		return *p
	}
	return fallback
}

// reducePoolsByDamage ports calcs.reducePoolsByDamage. damageTable is
// mutated exactly as the reference mutates it (non-positive entries are
// dropped, the rest ceiled).
func (env *Env) reducePoolsByDamage(poolTable *poolSet, damageTable map[string]float64, actor *performActor) *poolResult {
	output := actor.output
	modDB := actor.db
	poolTbl := poolTable
	if poolTbl == nil {
		poolTbl = &poolSet{}
	}

	damageTotal := 0.0
	for damageType, damage := range damageTable {
		if damage > 0 {
			damageTable[damageType] = math.Ceil(damage)
			damageTotal += damageTable[damageType]
		} else {
			delete(damageTable, damageType)
		}
	}

	alliesTakenBeforeYou := poolTbl.AlliesTakenBeforeYou
	if alliesTakenBeforeYou == nil {
		alliesTakenBeforeYou = buildAllyLifePools(output)
	}

	damageTakenThatCanBeRecouped := poolTbl.damageTakenThatCanBeRecouped
	if damageTakenThatCanBeRecouped == nil {
		damageTakenThatCanBeRecouped = map[string]float64{}
	}
	aegis := poolTbl.Aegis
	if aegis == nil {
		aegis = map[string]float64{
			"shared":          output.N("sharedAegis"),
			"sharedElemental": output.N("sharedElementalAegis"),
		}
		for damageType := range damageTable {
			aegis[damageType] = output.N(damageType + "Aegis")
		}
	}
	guard := poolTbl.Guard
	if guard == nil {
		guard = map[string]float64{"shared": output.N("sharedGuardAbsorb")}
		for damageType := range damageTable {
			guard[damageType] = output.N(damageType + "GuardAbsorb")
		}
	}

	ward := fOr(poolTbl.Ward, output.N("Ward"))
	wardActiveChance := 0.0
	if poolTbl.WardActiveChance != nil {
		wardActiveChance = *poolTbl.WardActiveChance
	} else if ward > 0 {
		wardActiveChance = 1
	}
	wardAvoidBreakChance := math.Min(modDB.Sum(modparser.Base, nil, "WardAvoidBreakChance")/100, 1)
	if modDB.Flag(nil, "Condition:WardNotBreak") {
		wardAvoidBreakChance = 1
	}
	wardBypassBelow := modDB.Sum(modparser.Base, nil, "WardBypassBelowPercent") / 100

	energyShield := fOr(poolTbl.EnergyShield, output.N("EnergyShieldRecoveryCap"))
	mana := fOr(poolTbl.Mana, output.N("ManaUnreserved"))
	life := fOr(poolTbl.Life, output.N("LifeRecoverable"))
	lifeLossBelowHalfPrevented := modDB.Sum(modparser.Base, nil, "LifeLossBelowHalfPrevented")
	lifeLossLostOverTime := poolTbl.LifeLossLostOverTime
	lifeBelowHalfLossLostOverTime := poolTbl.LifeBelowHalfLossLostOverTime
	overkillDamage := 0.0

	ward = math.Max(ward, 0)
	wardBeforeHit := ward
	energyShield = math.Max(energyShield, 0)
	mana = math.Max(mana, 0)
	life = math.Max(life, 0)

	// Initializing MoM(+EB(+Bypass)) pools here saves logic later to avoid
	// overusing pools for every damageType
	resourcesLostToTypeDamage := map[string]map[string]float64{}
	for _, damageType := range dmgTypeList {
		resourcesLostToTypeDamage[damageType] = map[string]float64{}
	}
	lifeHitPoolInitial := calcLifeHitPoolWithLossPrevention(life, output.N("Life"),
		output.N("preventedLifeLoss"), lifeLossBelowHalfPrevented)
	momPoolsRemaining := map[string]float64{}
	esPoolsRemaining := map[string]float64{}
	for damageType := range damageTable {
		momEffect := math.Min(output.N("sharedMindOverMatter")+output.N(damageType+"MindOverMatter"), 100) / 100
		maxMoM := lifeHitPoolInitial/(1-momEffect) - lifeHitPoolInitial
		momPool := mana
		if momEffect < 1 {
			momPool = math.Min(maxMoM, mana)
		}
		lifePlusMoMHitPool := lifeHitPoolInitial + momPool
		esBypass := output.N(damageType+"EnergyShieldBypass") / 100
		esPool := energyShield
		if esBypass > 0 {
			esPool = math.Min(lifePlusMoMHitPool/esBypass-lifePlusMoMHitPool, energyShield)
		}
		if modDB.Flag(nil, "EnergyShieldProtectsMana") && esBypass < 1 {
			esPoolsRemaining[damageType] = math.Floor(math.Min(math.Min(
				maxMoM*(1-esBypass), energyShield),
				lifePlusMoMHitPool/(1-(1-esBypass)*momEffect)-lifePlusMoMHitPool))
			momPoolsRemaining[damageType] = math.Floor(math.Min(maxMoM-esPoolsRemaining[damageType], momPool))
		} else {
			esPoolsRemaining[damageType] = math.Floor(esPool)
			momPoolsRemaining[damageType] = math.Floor(momPool)
		}
	}

	setLost := func(damageType, key string, v float64) {
		if v >= 1 {
			resourcesLostToTypeDamage[damageType][key] = v
		}
	}

	for _, damageType := range dmgTypeList {
		damageRemainder, present := damageTable[damageType]
		if !present {
			continue
		}
		for _, ally := range allyLifePoolList {
			allyValues := alliesTakenBeforeYou[ally.key]
			if allyValues != nil && allyValues.remaining > 0 {
				tempDamage := math.Min(damageRemainder*allyValues.percent, allyValues.remaining)
				allyValues.remaining = math.Floor(allyValues.remaining - tempDamage)
				damageRemainder -= tempDamage
				setLost(damageType, ally.key, tempDamage)
			}
		}
		// frost shield / soul link / other taken before you does not count
		// as you taking damage
		damageTakenThatCanBeRecouped[damageType] += damageRemainder
		if aegis[damageType] > 0 {
			tempDamage := math.Min(damageRemainder, aegis[damageType])
			aegis[damageType] -= tempDamage
			damageRemainder -= tempDamage
			setLost(damageType, "aegis", tempDamage)
		}
		if isElementalRes[damageType] && aegis["sharedElemental"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["sharedElemental"])
			aegis["sharedElemental"] -= tempDamage
			damageRemainder -= tempDamage
			setLost(damageType, "sharedElementalAegis", tempDamage)
		}
		if aegis["shared"] > 0 {
			tempDamage := math.Min(damageRemainder, aegis["shared"])
			aegis["shared"] -= tempDamage
			damageRemainder -= tempDamage
			setLost(damageType, "sharedAegis", tempDamage)
		}
		if guard[damageType] > 0 {
			tempDamage := math.Min(damageRemainder*output.N(damageType+"GuardAbsorbRate")/100, guard[damageType])
			guard[damageType] = math.Floor(guard[damageType] - tempDamage)
			damageRemainder -= tempDamage
			setLost(damageType, "guard", tempDamage)
		}
		if guard["shared"] > 0 {
			tempDamage := math.Min(damageRemainder*output.N("sharedGuardAbsorbRate")/100, guard["shared"])
			guard["shared"] = math.Floor(guard["shared"] - tempDamage)
			damageRemainder -= tempDamage
			setLost(damageType, "sharedGuard", tempDamage)
		}
		if ward > 0 && (wardBypassBelow == 0 || damageTotal >= wardBeforeHit*wardBypassBelow) {
			tempDamage := math.Min(damageRemainder*(1-modDB.Sum(modparser.Base, nil, "WardBypass")/100), ward)
			ward -= tempDamage
			tempDamage = tempDamage * wardActiveChance
			damageRemainder -= tempDamage
			setLost(damageType, "ward", tempDamage)
		}
		momEffect := math.Min(output.N("sharedMindOverMatter")+output.N(damageType+"MindOverMatter"), 100) / 100
		esBypass := output.N(damageType+"EnergyShieldBypass") / 100
		esPool := esPoolsRemaining[damageType]
		if energyShield > 0 && !modDB.Flag(nil, "EnergyShieldProtectsMana") && esBypass < 1 {
			tempDamage := math.Min(damageRemainder*(1-esBypass), esPool)
			for damageType2 := range damageTable {
				esPoolsRemaining[damageType2] = math.Max(esPoolsRemaining[damageType2]-tempDamage, 0)
			}
			energyShield -= tempDamage
			damageRemainder -= tempDamage
			setLost(damageType, "energyShield", tempDamage)
		}
		if momEffect > 0 {
			momDamage := math.Ceil(damageRemainder * momEffect)
			if modDB.Flag(nil, "EnergyShieldProtectsMana") && energyShield > 0 && esBypass < 1 {
				tempDamage := math.Ceil(math.Min(momDamage*(1-esBypass), esPool))
				for damageType2 := range damageTable {
					esPoolsRemaining[damageType2] = math.Floor(math.Max(esPoolsRemaining[damageType2]-tempDamage, 0))
				}
				energyShield -= tempDamage
				damageRemainder -= tempDamage
				momDamage -= tempDamage
				setLost(damageType, "energyShield", tempDamage)
			}
			tempDamage := math.Ceil(math.Min(momDamage, momPoolsRemaining[damageType]))
			for damageType2 := range damageTable {
				momPoolsRemaining[damageType2] = math.Floor(math.Max(momPoolsRemaining[damageType2]-tempDamage, 0))
			}
			mana -= tempDamage
			damageRemainder -= tempDamage
			setLost(damageType, "mana", tempDamage)
		}
		if output.N("preventedLifeLossTotal") > 0 {
			halfLife := output.N("Life") * 0.5
			lifeOverHalfLife := math.Max(life-halfLife, 0)
			preventPercent := output.N("preventedLifeLoss") / 100
			poolAboveLow := lifeOverHalfLife / (1 - preventPercent)
			preventBelowHalfPercent := lifeLossBelowHalfPrevented / 100
			damageThatLifeCanStillTake := poolAboveLow +
				math.Max(math.Min(life, halfLife), 0)/(1-preventBelowHalfPercent)/(1-preventPercent)
			if damageThatLifeCanStillTake < damageRemainder {
				overkillDamage += damageRemainder - damageThatLifeCanStillTake
				damageRemainder = damageThatLifeCanStillTake
			}
			if output.N("preventedLifeLossBelowHalf") != 0 {
				damageToSplit := math.Min(damageRemainder, poolAboveLow)
				lostLife := damageToSplit * (1 - preventPercent)
				preventedLoss := damageToSplit * preventPercent
				damageRemainder -= damageToSplit
				lifeLossLostOverTime += preventedLoss
				life -= lostLife
				resourcesLostToTypeDamage[damageType]["life"] = lostLife
				prevented := preventedLoss
				if life <= halfLife {
					unspecificallyLowLifePreventedDamage := damageRemainder * preventPercent
					lifeLossLostOverTime += unspecificallyLowLifePreventedDamage
					damageRemainder -= unspecificallyLowLifePreventedDamage
					specificallyLowLifePreventedDamage := damageRemainder * preventBelowHalfPercent
					lifeBelowHalfLossLostOverTime += specificallyLowLifePreventedDamage
					damageRemainder -= specificallyLowLifePreventedDamage
					prevented += unspecificallyLowLifePreventedDamage + specificallyLowLifePreventedDamage
				}
				if prevented >= 1 {
					resourcesLostToTypeDamage[damageType]["lifeLossPrevented"] = prevented
				} else {
					delete(resourcesLostToTypeDamage[damageType], "lifeLossPrevented")
				}
			} else {
				tempDamage := damageRemainder * output.N("preventedLifeLoss") / 100
				lifeLossLostOverTime += tempDamage
				damageRemainder -= tempDamage
				setLost(damageType, "lifeLossPrevented", tempDamage)
			}
		}
		if life > 0 {
			tempDamage := math.Min(damageRemainder, life)
			life -= tempDamage
			damageRemainder -= tempDamage
			if tempDamage > 0 {
				resourcesLostToTypeDamage[damageType]["life"] = resourcesLostToTypeDamage[damageType]["life"] + tempDamage
			}
		}
		overkillDamage += damageRemainder
		setLost(damageType, "overkill", damageRemainder)
	}

	momPoolRemaining := math.Inf(1)
	esPoolRemaining := math.Inf(1)
	for damageType := range damageTable {
		momPoolRemaining = math.Min(momPoolRemaining, momPoolsRemaining[damageType])
		esPoolRemaining = math.Min(esPoolRemaining, esPoolsRemaining[damageType])
	}
	hitPoolRemaining := 0.0
	if life >= 1 {
		hitPoolRemaining = calcLifeHitPoolWithLossPrevention(life, output.N("Life"),
			output.N("preventedLifeLoss"), lifeLossBelowHalfPrevented)
	}
	if !math.IsInf(momPoolRemaining, 1) {
		hitPoolRemaining += momPoolRemaining
	}
	if !math.IsInf(esPoolRemaining, 1) {
		hitPoolRemaining += esPoolRemaining
	}

	resultWard := ward
	resultWardActiveChance := wardActiveChance
	if ward < wardBeforeHit {
		resultWard = 0
		if wardAvoidBreakChance > 0 {
			resultWard = wardBeforeHit
		}
		resultWardActiveChance = wardActiveChance * wardAvoidBreakChance
	}

	return &poolResult{
		AlliesTakenBeforeYou:          alliesTakenBeforeYou,
		Aegis:                         aegis,
		Guard:                         guard,
		Ward:                          math.Floor(resultWard),
		WardActiveChance:              resultWardActiveChance,
		EnergyShield:                  math.Floor(energyShield),
		Mana:                          math.Floor(mana),
		Life:                          math.Floor(life),
		damageTakenThatCanBeRecouped:  damageTakenThatCanBeRecouped,
		LifeLossLostOverTime:          math.Ceil(lifeLossLostOverTime),
		LifeBelowHalfLossLostOverTime: math.Ceil(lifeBelowHalfLossLostOverTime),
		OverkillDamage:                math.Ceil(overkillDamage),
		hitPoolRemaining:              math.Floor(hitPoolRemaining),
		resourcesLostToTypeDamage:     resourcesLostToTypeDamage,
	}
}
