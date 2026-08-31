// CalcDefence.lua L2379-2566: guard, aegis, the ally life pools (frost
// shield, minions, totems, soul link), Vaal Arctic Armour and total pools.
package calc

import (
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
	"math"
)

// allyLifePool mirrors one allyLifePoolList entry (CalcDefence.lua L32).
type allyLifePool struct {
	key        string
	life       string
	mitigation string
	redirect   string
	fallback   string
	meleeOnly  bool
}

var allyLifePoolList = []allyLifePool{
	{key: "minion", life: "TotalMinionLife", mitigation: "MinionAllyDamageMitigation", redirect: "takenFromMinionBeforeYou", fallback: "Multiplier:MinionLife"},
	{key: "radianceSentinel", life: "TotalRadianceSentinelLife", mitigation: "RadianceSentinelAllyDamageMitigation", redirect: "takenFromRadianceSentinelBeforeYou"},
	{key: "spectres", life: "TotalSpectreLife", mitigation: "SpectreAllyDamageMitigation", redirect: "takenFromSpectresBeforeYou"},
	{key: "totems", life: "TotalTotemLife", mitigation: "TotemAllyDamageMitigation", redirect: "takenFromTotemsBeforeYou"},
	{key: "vaalRejuvenationTotems", life: "TotalVaalRejuvenationTotemLife", mitigation: "VaalRejuvenationTotemAllyDamageMitigation", redirect: "takenFromVaalRejuvenationTotemsBeforeYou"},
	{key: "voidSpawn", life: "TotalVoidSpawnLife", mitigation: "VoidSpawnAllyDamageMitigation", redirect: "takenFromVoidSpawnBeforeYou"},
	{key: "stoneGolem", life: "TotalStoneGolemLife", mitigation: "StoneGolemAllyDamageMitigation", redirect: "takenFromStoneGolemBeforeYou", meleeOnly: true},
	{key: "soulLink", life: "AlliedEnergyShield", mitigation: "SoulLinkMitigation"},
	{key: "frostShield", life: "FrostShieldLife", mitigation: "FrostShieldDamageMitigation"},
}

// allyPoolState is one entry of buildAllyLifePools' result.
type allyPoolState struct {
	remaining float64
	percent   float64
}

// buildAllyLifePools ports the local of the same name.
func buildAllyLifePools(output modstore.Output) map[string]*allyPoolState {
	pools := map[string]*allyPoolState{}
	for _, ally := range allyLifePoolList {
		life, hasLife := output[ally.life]
		mitigation, hasMit := output[ally.mitigation]
		if hasLife && life.Num() > 0 && hasMit && mitigation.Num() > 0 {
			pools[ally.key] = &allyPoolState{remaining: life.Num(), percent: math.Min(mitigation.Num(), 100) / 100}
		}
	}
	return pools
}

func (env *Env) ehpGuard(actor *performActor, damageCategoryConfig DamageCategory) {
	modDB := actor.db
	output := actor.output

	// Guard
	output.SetFlag("AnyGuard", false)
	output.SetN("sharedGuardAbsorbRate", math.Min(modDB.Sum(modparser.Base, nil, "GuardAbsorbRate"), 100))
	if output.N("sharedGuardAbsorbRate") > 0 {
		output.SetFlag("OnlySharedGuard", true)
		output.SetN("sharedGuardAbsorb", Val(modDB, "GuardAbsorbLimit", nil))
	}
	for _, damageType := range dmgTypeList {
		output.SetN(damageType+"GuardAbsorbRate", math.Min(modDB.Sum(modparser.Base, nil, damageType+"GuardAbsorbRate"), 100))
		if output.N(damageType+"GuardAbsorbRate") > 0 {
			output.SetFlag("ehpSectionAnySpecificTypes", true)
			output.SetFlag("AnyGuard", true)
			output.SetFlag("OnlySharedGuard", false)
			output.SetN(damageType+"GuardAbsorb", Val(modDB, damageType+"GuardAbsorbLimit", nil))
		}
	}

	// aegis
	output.SetFlag("AnyAegis", false)
	sharedAegis, _ := modDB.Max(nil, "AegisValue")
	sharedElementalAegis, _ := modDB.Max(nil, "ElementalAegisValue")
	output.SetN("sharedAegis", sharedAegis)
	output.SetN("sharedElementalAegis", sharedElementalAegis)
	if sharedAegis > 0 {
		output.SetFlag("AnyAegis", true)
	}
	if sharedElementalAegis > 0 {
		output.SetFlag("ehpSectionAnySpecificTypes", true)
		output.SetFlag("AnyAegis", true)
	}
	for _, damageType := range dmgTypeList {
		aegisValue, _ := modDB.Max(nil, damageType+"AegisValue")
		if aegisValue > 0 {
			output.SetFlag("ehpSectionAnySpecificTypes", true)
			output.SetFlag("AnyAegis", true)
			output.SetN(damageType+"Aegis", aegisValue)
		} else {
			output.SetN(damageType+"Aegis", 0.0)
		}
		if isElementalRes[damageType] {
			output.SetN(damageType+"AegisDisplay", output.N(damageType+"Aegis")+sharedElementalAegis)
		}
	}

	// taken from allies before you, eg. frost shield
	output.SetN("FrostShieldLife", modDB.Sum(modparser.Base, nil, "FrostGlobeHealth"))
	output.SetN("FrostShieldDamageMitigation", modDB.Sum(modparser.Base, nil, "FrostGlobeDamageMitigation"))

	// Every ally redirect uses the same pool rules; the Safeguarding Golem
	// is the only melee-only case.
	for _, ally := range allyLifePoolList {
		if ally.redirect == "" {
			continue
		}
		mitigation := modDB.Sum(modparser.Base, nil, ally.redirect)
		if ally.meleeOnly {
			switch damageCategoryConfig {
			case DamageMelee:
				// unchanged
			case DamageAverage:
				mitigation = mitigation / 4
			default:
				mitigation = 0
			}
		}
		output.SetN(ally.mitigation, mitigation)
		if mitigation != 0 {
			life := modDB.Sum(modparser.Base, nil, ally.life)
			if life == 0 && ally.fallback != "" {
				life = modDB.Sum(modparser.Base, nil, ally.fallback)
			}
			if ov, ok := modDB.Override(nil, ally.life); ok {
				output.SetN(ally.life, valueNum(ov))
			} else {
				output.SetN(ally.life, life)
			}
		}
	}

	// Companionship and an ally-specific redirect can both spend the same
	// minion's Life.
	for _, ally := range allyLifePoolList {
		if modDB.Flag(nil, "MinionLifeShares"+ally.life) {
			specificOverride, ok := modDB.Override(nil, ally.life)
			if ok && !hasOverride(modDB, nil, "TotalMinionLife") {
				output.SetN("TotalMinionLife", valueNum(specificOverride))
			} else if output.N("TotalMinionLife") == 0 {
				// `not x or x == 0`: absent and present-zero both take the
				// ally pool, and assigning an absent value deletes the key,
				// as the Lua nil assignment does (CalcDefence.lua:2484-2485).
				output.Set("TotalMinionLife", output.Get(ally.life))
			}
			output.SetN("MinionAllyDamageMitigation", output.N("MinionAllyDamageMitigation")+output.N(ally.mitigation))
			output.SetN(ally.mitigation, 0.0)
			output.Del(ally.life)
		}
	}

	// When Vaal Rejuvenation is treated as the nearest Totem, both
	// redirects use its Life.
	if output.N("TotemAllyDamageMitigation") > 0 && output.N("TotalTotemLife") == 0 &&
		output.N("TotalVaalRejuvenationTotemLife") > 0 {
		output.SetN("VaalRejuvenationTotemAllyDamageMitigation", output.N("VaalRejuvenationTotemAllyDamageMitigation")+output.N("TotemAllyDamageMitigation"))
		output.SetN("TotemAllyDamageMitigation", 0.0)
		output.Del("TotalTotemLife")
	}

	// from Allied Energy Shield
	output.SetN("SoulLinkMitigation", modDB.Sum(modparser.Base, nil, "TakenFromParentESBeforeYou"))
	if output.N("SoulLinkMitigation") != 0 {
		output.SetN("AlliedEnergyShield", actor.parent.output.N("EnergyShieldRecoveryCap"))
	} else {
		output.SetN("SoulLinkMitigation", modDB.Sum(modparser.Base, nil, "TakenFromPartyMemberESBeforeYou"))
		if output.N("SoulLinkMitigation") != 0 {
			panic("ehp: TakenFromPartyMemberESBeforeYou needs the party tab")
		}
	}

	// Vaal Arctic Armour
	output.SetN("VaalArcticArmourLife", modDB.Sum(modparser.Base, nil, "VaalArcticArmourMaxHits"))
	output.SetN("VaalArcticArmourMitigation", math.Min(-modDB.Sum(modparser.More, nil, "VaalArcticArmourMitigation")/100, 1))

	// total pool
	for _, damageType := range dmgTypeList {
		output.SetN(damageType+"TotalPool", output.N(damageType+"ManaEffectiveLife"))
		output.SetN(damageType+"TotalHitPool", output.N(damageType+"MoMHitPool"))
		if output.N(damageType+"EnergyShieldBypass") < 100 && !modDB.Flag(nil, "EnergyShieldProtectsMana") {
			bypass := output.N(damageType + "EnergyShieldBypass")
			if bypass > 0 {
				poolProtected := output.N("EnergyShieldRecoveryCap") / (1 - bypass/100) * (bypass / 100)
				output.SetN(damageType+"TotalPool", math.Max(output.N(damageType+"TotalPool")-poolProtected, 0)+
					math.Min(output.N(damageType+"TotalPool"), poolProtected)/(bypass/100))
				output.SetN(damageType+"TotalHitPool", math.Max(output.N(damageType+"TotalHitPool")-poolProtected, 0)+
					math.Min(output.N(damageType+"TotalHitPool"), poolProtected)/(bypass/100))
			} else {
				output.SetN(damageType+"TotalPool", output.N(damageType+"TotalPool")+output.N("EnergyShieldRecoveryCap"))
				output.SetN(damageType+"TotalHitPool", output.N(damageType+"TotalHitPool")+output.N("EnergyShieldRecoveryCap"))
			}
		}
	}
}
