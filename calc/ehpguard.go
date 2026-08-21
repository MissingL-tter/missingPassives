// CalcDefence.lua L2379-2566: guard, aegis, the ally life pools (frost
// shield, minions, totems, soul link), Vaal Arctic Armour and total pools.
package calc

import "math"

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
func buildAllyLifePools(output map[string]any) map[string]*allyPoolState {
	pools := map[string]*allyPoolState{}
	for _, ally := range allyLifePoolList {
		life, hasLife := output[ally.life]
		mitigation, hasMit := output[ally.mitigation]
		if hasLife && anyNum(life) > 0 && hasMit && anyNum(mitigation) > 0 {
			pools[ally.key] = &allyPoolState{remaining: anyNum(life), percent: math.Min(anyNum(mitigation), 100) / 100}
		}
	}
	return pools
}

func (env *Env) ehpGuard(actor *performActor, damageCategoryConfig string) {
	modDB := actor.db
	output := actor.output

	// Guard
	output["AnyGuard"] = false
	output["sharedGuardAbsorbRate"] = math.Min(modDB.Sum("BASE", nil, "GuardAbsorbRate"), 100)
	if outNum(output, "sharedGuardAbsorbRate") > 0 {
		output["OnlySharedGuard"] = true
		output["sharedGuardAbsorb"] = Val(modDB, "GuardAbsorbLimit", nil)
	}
	for _, damageType := range dmgTypeList {
		output[damageType+"GuardAbsorbRate"] = math.Min(modDB.Sum("BASE", nil, damageType+"GuardAbsorbRate"), 100)
		if outNum(output, damageType+"GuardAbsorbRate") > 0 {
			output["ehpSectionAnySpecificTypes"] = true
			output["AnyGuard"] = true
			output["OnlySharedGuard"] = false
			output[damageType+"GuardAbsorb"] = Val(modDB, damageType+"GuardAbsorbLimit", nil)
		}
	}

	// aegis
	output["AnyAegis"] = false
	sharedAegis, _ := modDB.Max(nil, "AegisValue")
	sharedElementalAegis, _ := modDB.Max(nil, "ElementalAegisValue")
	output["sharedAegis"] = sharedAegis
	output["sharedElementalAegis"] = sharedElementalAegis
	if sharedAegis > 0 {
		output["AnyAegis"] = true
	}
	if sharedElementalAegis > 0 {
		output["ehpSectionAnySpecificTypes"] = true
		output["AnyAegis"] = true
	}
	for _, damageType := range dmgTypeList {
		aegisValue, _ := modDB.Max(nil, damageType+"AegisValue")
		if aegisValue > 0 {
			output["ehpSectionAnySpecificTypes"] = true
			output["AnyAegis"] = true
			output[damageType+"Aegis"] = aegisValue
		} else {
			output[damageType+"Aegis"] = 0.0
		}
		if isElementalRes[damageType] {
			output[damageType+"AegisDisplay"] = outNum(output, damageType+"Aegis") + sharedElementalAegis
		}
	}

	// taken from allies before you, eg. frost shield
	output["FrostShieldLife"] = modDB.Sum("BASE", nil, "FrostGlobeHealth")
	output["FrostShieldDamageMitigation"] = modDB.Sum("BASE", nil, "FrostGlobeDamageMitigation")

	// Every ally redirect uses the same pool rules; the Safeguarding Golem
	// is the only melee-only case.
	for _, ally := range allyLifePoolList {
		if ally.redirect == "" {
			continue
		}
		mitigation := modDB.Sum("BASE", nil, ally.redirect)
		if ally.meleeOnly {
			switch damageCategoryConfig {
			case "Melee":
				// unchanged
			case "Average":
				mitigation = mitigation / 4
			default:
				mitigation = 0
			}
		}
		output[ally.mitigation] = mitigation
		if mitigation != 0 {
			life := modDB.Sum("BASE", nil, ally.life)
			if life == 0 && ally.fallback != "" {
				life = modDB.Sum("BASE", nil, ally.fallback)
			}
			if ov := modDB.Override(nil, ally.life); truthy(ov) {
				output[ally.life] = anyNum(ov)
			} else {
				output[ally.life] = life
			}
		}
	}

	// Companionship and an ally-specific redirect can both spend the same
	// minion's Life.
	for _, ally := range allyLifePoolList {
		if modDB.Flag(nil, "MinionLifeShares"+ally.life) {
			specificOverride := modDB.Override(nil, ally.life)
			if !truthy(modDB.Override(nil, "TotalMinionLife")) && specificOverride != nil {
				output["TotalMinionLife"] = anyNum(specificOverride)
			} else if !truthy(output["TotalMinionLife"]) {
				output["TotalMinionLife"] = outNum(output, ally.life)
			}
			output["MinionAllyDamageMitigation"] = outNum(output, "MinionAllyDamageMitigation") + outNum(output, ally.mitigation)
			output[ally.mitigation] = 0.0
			delete(output, ally.life)
		}
	}

	// When Vaal Rejuvenation is treated as the nearest Totem, both
	// redirects use its Life.
	if outNum(output, "TotemAllyDamageMitigation") > 0 && outNum(output, "TotalTotemLife") == 0 &&
		outNum(output, "TotalVaalRejuvenationTotemLife") > 0 {
		output["VaalRejuvenationTotemAllyDamageMitigation"] = outNum(output, "VaalRejuvenationTotemAllyDamageMitigation") + outNum(output, "TotemAllyDamageMitigation")
		output["TotemAllyDamageMitigation"] = 0.0
		delete(output, "TotalTotemLife")
	}

	// from Allied Energy Shield
	output["SoulLinkMitigation"] = modDB.Sum("BASE", nil, "TakenFromParentESBeforeYou")
	if outNum(output, "SoulLinkMitigation") != 0 {
		output["AlliedEnergyShield"] = outNum(actor.parent.output, "EnergyShieldRecoveryCap")
	} else {
		output["SoulLinkMitigation"] = modDB.Sum("BASE", nil, "TakenFromPartyMemberESBeforeYou")
		if outNum(output, "SoulLinkMitigation") != 0 {
			panic("ehp: TakenFromPartyMemberESBeforeYou needs the party tab")
		}
	}

	// Vaal Arctic Armour
	output["VaalArcticArmourLife"] = modDB.Sum("BASE", nil, "VaalArcticArmourMaxHits")
	output["VaalArcticArmourMitigation"] = math.Min(-modDB.Sum("MORE", nil, "VaalArcticArmourMitigation")/100, 1)

	// total pool
	for _, damageType := range dmgTypeList {
		output[damageType+"TotalPool"] = outNum(output, damageType+"ManaEffectiveLife")
		output[damageType+"TotalHitPool"] = outNum(output, damageType+"MoMHitPool")
		if outNum(output, damageType+"EnergyShieldBypass") < 100 && !modDB.Flag(nil, "EnergyShieldProtectsMana") {
			bypass := outNum(output, damageType+"EnergyShieldBypass")
			if bypass > 0 {
				poolProtected := outNum(output, "EnergyShieldRecoveryCap") / (1 - bypass/100) * (bypass / 100)
				output[damageType+"TotalPool"] = math.Max(outNum(output, damageType+"TotalPool")-poolProtected, 0) +
					math.Min(outNum(output, damageType+"TotalPool"), poolProtected)/(bypass/100)
				output[damageType+"TotalHitPool"] = math.Max(outNum(output, damageType+"TotalHitPool")-poolProtected, 0) +
					math.Min(outNum(output, damageType+"TotalHitPool"), poolProtected)/(bypass/100)
			} else {
				output[damageType+"TotalPool"] = outNum(output, damageType+"TotalPool") + outNum(output, "EnergyShieldRecoveryCap")
				output[damageType+"TotalHitPool"] = outNum(output, damageType+"TotalHitPool") + outNum(output, "EnergyShieldRecoveryCap")
			}
		}
	}
}
