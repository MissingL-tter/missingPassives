// CalcDefence.lua L2735-2931: the number of hits needed to kill, the
// mitigated-hits pass (block, suppression, avoidance, gain-when-hit), the
// total EHP and the survival time.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
)

func (env *Env) ehpHitCounts(actor *performActor, damageCategoryConfig string) {
	modDB := actor.db
	enemyDB := actor.enemy.db
	output := actor.output

	if damageCategoryConfig == "DamageOverTime" {
		return
	}

	// number of damaging hits needed to be taken to die
	{
		in := newDamageIn()
		for _, damageType := range dmgTypeList {
			in.dmg[damageType] = outNum(output, damageType+"TakenHit")
		}
		in.LimitEHPSpeedup = outNum(output, "preventedLifeLossTotal") > 0
		output["NumberOfDamagingHits"] = env.numberOfHitsToDie(in, actor)
	}

	{
		in := newDamageIn()
		blockChance := outNum(output, "EffectiveBlockChance") / 100
		if damageCategoryConfig != "Melee" && damageCategoryConfig != "Untyped" {
			blockChance = outNum(output, "Effective"+damageCategoryConfig+"BlockChance") / 100
		}
		if enemyDB.Flag(nil, "CannotBeBlocked") {
			blockChance = 0
		}
		blockEffect := 1 - blockChance*outNum(output, "BlockEffect")/100
		suppressChance := 0.0
		suppressionEffect := 1.0
		extraAvoidChance := 0.0
		averageAvoidChance := 0.0
		gainOnBlockEnabled := !truthy(env.ConfigInput["DisableEHPGainOnBlock"]) && outNum(output, "NumberOfDamagingHits") > 1
		if gainOnBlockEnabled {
			in.LifeWhenHit = outNum(output, "LifeOnBlock") * blockChance
			in.ManaWhenHit = outNum(output, "ManaOnBlock") * blockChance
			in.EnergyShieldWhenHit = outNum(output, "EnergyShieldOnBlock") * blockChance
			switch damageCategoryConfig {
			case "Spell", "SpellProjectile":
				in.EnergyShieldWhenHit += outNum(output, "EnergyShieldOnSpellBlock") * blockChance
			case "Average":
				in.EnergyShieldWhenHit += outNum(output, "EnergyShieldOnSpellBlock") / 2 * blockChance
			}
		}
		// suppression
		if damageCategoryConfig == "Spell" || damageCategoryConfig == "SpellProjectile" || damageCategoryConfig == "Average" {
			suppressChance = outNum(output, "EffectiveSpellSuppressionChance") / 100
		}
		// We include suppression in damage reduction if it is 100%,
		// otherwise we handle it here.
		if suppressChance < 1 {
			if damageCategoryConfig == "Average" {
				suppressChance = suppressChance / 2
			}
			in.EnergyShieldWhenHit += outNum(output, "EnergyShieldOnSuppress") * suppressChance
			in.LifeWhenHit += outNum(output, "LifeOnSuppress") * suppressChance
			suppressionEffect = 1 - suppressChance*outNum(output, "SpellSuppressionEffect")/100
		} else {
			half := 1.0
			if damageCategoryConfig == "Average" {
				half = 0.5
			}
			in.EnergyShieldWhenHit += outNum(output, "EnergyShieldOnSuppress") * half
			in.LifeWhenHit += outNum(output, "LifeOnSuppress") * half
		}
		// extra avoid chance
		switch damageCategoryConfig {
		case "Projectile", "SpellProjectile":
			extraAvoidChance += outNum(output, "AvoidProjectilesChance")
		case "Average":
			extraAvoidChance += outNum(output, "AvoidProjectilesChance") / 2
		}
		// gain when hit (currently just gain on block/suppress, and
		// Defiance of Destiny)
		if gainOnBlockEnabled {
			missingLife := modDB.Sum("BASE", nil, "MissingLifeBeforeEnemyHit")
			missingMana := modDB.Sum("BASE", nil, "MissingManaBeforeEnemyHit")
			in.MissingLifeBeforeEnemyHit = &missingLife
			in.MissingManaBeforeEnemyHit = &missingMana
			if in.LifeWhenHit != 0 || in.ManaWhenHit != 0 || in.EnergyShieldWhenHit != 0 ||
				missingLife != 0 || missingMana != 0 {
				in.GainWhenHit = true
			}
		} else {
			in.LifeWhenHit = 0
			in.ManaWhenHit = 0
			in.EnergyShieldWhenHit = 0
		}
		for _, damageType := range dmgTypeList {
			// Emperor's Vigilance (this needs to fail with divine flesh as
			// it can't override it, hence the check for high bypass). The
			// per-type bypass written here is never read back by the
			// solver, matching the reference.
			avoidChance := 0.0
			if truthy(output["specificTypeAvoidance"]) {
				avoidChance = math.Min(outNum(output, "Avoid"+damageType+"DamageChance")+extraAvoidChance, data.Misc.AvoidChanceCap)
				// unlucky config to lower the value of block, dodge, evade etc for ehp
				worstOf := 1.0
				if v := env.ConfigInput["EHPUnluckyWorstOf"]; truthy(v) {
					worstOf = anyNum(v)
				}
				if worstOf > 1 {
					avoidChance = avoidChance / 100 * avoidChance
					if worstOf == 4 {
						avoidChance = avoidChance / 100 * avoidChance
					}
				}
				averageAvoidChance += avoidChance
			}
			in.dmg[damageType] = outNum(output, damageType+"TakenHit") * (blockEffect * suppressionEffect * (1 - avoidChance/100))
		}
		// recoup initialisation
		if outNum(output, "anyRecoup") > 0 {
			in.TrackRecoupable = true
			for _, damageType := range dmgTypeList {
				output[damageType+"RecoupableDamageTaken"] = 0.0
			}
		}
		// taken over time degen initialisation
		if outNum(output, "preventedLifeLossTotal") > 0 {
			in.TrackLifeLossOverTime = true
			output["LifeLossLostOverTime"] = 0.0
			output["LifeBelowHalfLossLostOverTime"] = 0.0
		}
		in.LimitEHPSpeedup = in.TrackRecoupable || in.TrackLifeLossOverTime || in.GainWhenHit
		averageAvoidChance = averageAvoidChance / 5
		output["ConfiguredDamageChance"] = 100 * (blockEffect * suppressionEffect * (1 - averageAvoidChance/100))
		if outNum(output, "ConfiguredDamageChance") != 100 || in.TrackRecoupable || in.TrackLifeLossOverTime || in.GainWhenHit {
			output["NumberOfMitigatedDamagingHits"] = env.numberOfHitsToDie(in, actor)
		} else {
			output["NumberOfMitigatedDamagingHits"] = outNum(output, "NumberOfDamagingHits")
		}
	}

	// chance to not be hit
	output["TotalNumberOfHits"] = outNum(output, "NumberOfMitigatedDamagingHits") / (1 - outNum(output, "ConfiguredNotHitChance")/100)

	// effective hit pool
	output["TotalEHP"] = outNum(output, "TotalNumberOfHits") * outNum(output, "totalEnemyDamageIn")

	// survival time
	enemySpeed := 700.0
	if v := env.ConfigInput["enemySpeed"]; truthy(v) {
		enemySpeed = anyNum(v)
	} else if v := env.Build.ConfigPlaceholder["enemySpeed"]; truthy(v) {
		enemySpeed = anyNum(v)
	}
	enemySkillTime := enemySpeed / (1 + enemyDB.Sum("INC", nil, "Speed")/100)
	enemyActionSpeed := env.actionSpeedMod(actor.enemy)
	enemySkillTime = enemySkillTime / 1000 / enemyActionSpeed
	output["enemySkillTime"] = enemySkillTime
	output["EHPSurvivalTime"] = outNum(output, "TotalNumberOfHits") * enemySkillTime
}
