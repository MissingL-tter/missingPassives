// CalcPerform.lua L1583-1900: flask and tincture merging. Iteration over
// the flask/tincture sets follows the item-id order (the reference's
// table-keyed pairs order is process-random; the effects are order-free —
// mergeBuff keys on content — so any fixed order reproduces the state).
package calc

import (
	"math"
	"sort"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// sortedFlasks returns the set's items ordered by item id.
func (env *Env) sortedFlasks(set map[*Item]bool) []*Item {
	idOf := map[*Item]int{}
	for id, item := range env.ItemPool {
		idOf[item] = id
	}
	var out []*Item
	for item, on := range set {
		if on {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(a, b int) bool { return idOf[out[a]] < idOf[out[b]] })
	return out
}

func flaskBaseLife(item *Item) bool {
	return item.In.Base != nil && item.In.Base.Flask != nil && item.In.Base.Flask.Life.Set
}

func flaskBaseMana(item *Item) bool {
	return item.In.Base != nil && item.In.Base.Flask != nil && item.In.Base.Flask.Mana.Set
}

// getFlaskInstantRecovery ports the local getFlaskInstantRecovery.
func (env *Env) getFlaskInstantRecovery(item *Item) float64 {
	modDB := env.ModDB
	instantPerc := item.In.FlaskData.InstantPerc.Or(0)
	if modDB.Flag(nil, "Condition:LowLife") {
		instantPerc += item.In.FlaskData.InstantLowLifePerc.Or(0)
	}
	if flaskBaseLife(item) {
		instantPerc += modDB.Sum(modparser.Base, nil, "LifeFlaskInstantRecovery")
	}
	if flaskBaseMana(item) {
		instantPerc += modDB.Sum(modparser.Base, nil, "ManaFlaskInstantRecovery")
	}
	return math.Min(100, instantPerc)
}

// calcFlaskRecovery ports the local calcFlaskRecovery.
func (env *Env) calcFlaskRecovery(typ string, item *Item, effectInc, flaskTotalRateInc, flaskDurInc float64) []*modparser.Mod {
	modDB := env.ModDB
	var out []*modparser.Mod
	pool := item.In.FlaskData.Pool(typ)

	if !(pool != nil && pool.EffectNotRemoved) && !modDB.Flag(nil, typ+"FlaskEffectNotRemoved") {
		return out
	}

	name := item.In.Name
	base := 0.0
	if pool != nil {
		base = pool.Base
	}
	dur := item.In.FlaskData.Duration
	instPerc := env.getFlaskInstantRecovery(item)
	flaskRecInc := modDB.Sum(modparser.Inc, nil, "Flask"+typ+"Recovery")
	flaskRecMore := modDB.More(nil, "Flask"+typ+"Recovery")
	flaskRateInc := modDB.Sum(modparser.Inc, nil, "Flask"+typ+"RecoveryRate")
	flaskTotal := base * (1 - instPerc/100) * (1 + flaskRecInc/100) * flaskRecMore * (1 + flaskDurInc/100)
	flaskDur := dur * (1 + flaskDurInc/100) / (1 + flaskTotalRateInc/100) / (1 + flaskRateInc/100)

	// low-life more recovery is not affected by flask effect (verified
	// ingame); counteract by removing flask effect beforehand
	lowLifeFlaskRecMore := modDB.More(nil, "FlaskLifeRecoveryLowLife")
	if lowLifeFlaskRecMore > 1 {
		if typ == "Life" {
			flaskTotal = flaskTotal * ((lowLifeFlaskRecMore-1)/(1+effectInc/100) + 1)
		}
	}

	out = append(out, newModS(typ+"Recovery", modparser.Base, modparser.Num(flaskTotal/flaskDur), name))
	if modDB.Flag(nil, typ+"FlaskAppliesToEnergyShield") {
		out = append(out, newModS("EnergyShieldRecovery", modparser.Base, modparser.Num(flaskTotal/flaskDur), name))
	}
	if modDB.Flag(nil, typ+"FlaskAppliesToLife") {
		out = append(out, newModS("LifeRecovery", modparser.Base, modparser.Num(flaskTotal/flaskDur), name))
	}
	return out
}

// mergeFlasks ports the local mergeFlasks.
func (env *Env) mergeFlasks(flasks map[*Item]bool, onlyRecovery, checkNonRecoveryFlasksForMinions bool, nonUniqueFlasksApplyToMinion bool) {
	modDB := env.ModDB
	playerCfg := &modstore.Cfg{Actor: "player"}
	effectInc := modDB.Sum(modparser.Inc, playerCfg, "FlaskEffect")
	effectIncMagic := modDB.Sum(modparser.Inc, playerCfg, "MagicUtilityFlaskEffect")
	effectIncMagicNoAdjacent := modDB.Sum(modparser.Inc, playerCfg, "MagicFlaskNoAdjacentEffect")
	effectIncNonPlayer := modDB.Sum(modparser.Inc, nil, "FlaskEffect")
	effectIncMagicNonPlayer := modDB.Sum(modparser.Inc, nil, "MagicUtilityFlaskEffect")
	flasksApplyToMinion := env.Minion != nil && modDB.Flag(env.PlayerMainSkill.SkillCfg, "FlasksApplyToMinion")
	quickSilverAppliesToAllies := env.Minion != nil && modDB.Flag(env.PlayerMainSkill.SkillCfg, "QuickSilverAppliesToAllies")
	flaskTotalRateInc := modDB.Sum(modparser.Inc, nil, "FlaskRecoveryRate")
	flaskDurInc := modDB.Sum(modparser.Inc, nil, "FlaskDuration")

	flaskBuffs := map[string]*modstore.List{}
	flaskConditions := map[string]bool{}
	flaskConditionsNonUtility := map[string]bool{}
	flaskBuffsPerBase := map[string]map[string]*modstore.List{}
	flaskBuffsNonPlayer := map[string]*modstore.List{}
	flaskBuffsPerBaseNonPlayer := map[string]map[string]*modstore.List{}
	flaskBuffsNonUtility := map[string]*modstore.List{}

	calcFlaskMods := func(item *Item, baseName string, buffModList, modList []*modparser.Mod, onlyMinion bool) {
		flaskEffectInc := effectInc + item.In.FlaskData.EffectInc
		flaskEffectIncNonPlayer := effectIncNonPlayer + item.In.FlaskData.EffectInc
		if item.In.Rarity == "MAGIC" && !(flaskBaseLife(item) || flaskBaseMana(item)) {
			flaskEffectInc += effectIncMagic
			flaskEffectIncNonPlayer += effectIncMagicNonPlayer
		}
		// Essence of Desolation belt mod: bonus for magic flasks with no
		// flask in an adjacent slot
		if item.In.Rarity == "MAGIC" && effectIncMagicNoAdjacent != 0 {
			if flaskSlotNum, ok := env.FlaskSlotMap[item]; ok {
				hasAdjacent := env.FlaskSlotOccupied[flaskSlotNum-1] || env.FlaskSlotOccupied[flaskSlotNum+1]
				if !hasAdjacent {
					flaskEffectInc += effectIncMagicNoAdjacent
				}
			}
		}
		effectMod := 1 + flaskEffectInc/100
		effectModNonPlayer := 1 + flaskEffectIncNonPlayer/100

		// utility flasks group by base, uniques by name, magic flasks by
		// their modifiers
		if len(buffModList) > 0 {
			if !onlyMinion {
				srcList := modstore.NewList(nil)
				srcList.ScaleAddList(buffModList, effectMod, false)
				mergeBuff(srcList.Mods, flaskBuffs, baseName)
				mergeBuff(srcList.Mods, flaskBuffsPerBase[deref(item.In.BaseName)], baseName)
			}
			if (!onlyRecovery || checkNonRecoveryFlasksForMinions) &&
				(flasksApplyToMinion || quickSilverAppliesToAllies || (nonUniqueFlasksApplyToMinion && item.In.Rarity != "UNIQUE" && item.In.Rarity != "RELIC")) {
				srcList := modstore.NewList(nil)
				srcList.ScaleAddList(buffModList, effectModNonPlayer, false)
				mergeBuff(srcList.Mods, flaskBuffsNonPlayer, baseName)
				mergeBuff(srcList.Mods, flaskBuffsPerBaseNonPlayer[deref(item.In.BaseName)], baseName)
			}
		}

		if len(modList) > 0 {
			srcList := modstore.NewList(nil)
			srcList.ScaleAddList(modList, effectMod, false)
			var key string
			if item.In.Rarity == "UNIQUE" || item.In.Rarity == "RELIC" {
				key = deref(item.In.Title)
			} else {
				key = ""
				for _, mod := range modList {
					key = key + modparser.FormatModParams(mod) + "&"
				}
			}
			if !onlyRecovery {
				mergeBuff(srcList.Mods, flaskBuffs, key)
				mergeBuff(srcList.Mods, flaskBuffsPerBase[deref(item.In.BaseName)], key)
				mergeBuff(srcList.Mods, flaskBuffsNonUtility, key)
			}
			if (!onlyRecovery || checkNonRecoveryFlasksForMinions) &&
				(flasksApplyToMinion || quickSilverAppliesToAllies || (nonUniqueFlasksApplyToMinion && item.In.Rarity != "UNIQUE" && item.In.Rarity != "RELIC")) {
				srcList := modstore.NewList(nil)
				srcList.ScaleAddList(modList, effectModNonPlayer, false)
				mergeBuff(srcList.Mods, flaskBuffsNonPlayer, key)
				mergeBuff(srcList.Mods, flaskBuffsPerBaseNonPlayer[deref(item.In.BaseName)], key)
			}
		}
	}

	for _, item := range env.sortedFlasks(flasks) {
		baseName := deref(item.In.BaseName)
		if flaskBuffsPerBase[baseName] == nil {
			flaskBuffsPerBase[baseName] = map[string]*modstore.List{}
		}
		if flaskBuffsPerBaseNonPlayer[baseName] == nil {
			flaskBuffsPerBaseNonPlayer[baseName] = map[string]*modstore.List{}
		}
		instantPerc := env.getFlaskInstantRecovery(item)
		if instantPerc < 100 {
			flaskConditions["UsingFlask"] = true
			flaskConditions["Using"+stripSpaces(baseName)] = true
			if flaskBaseLife(item) || flaskBaseMana(item) {
				flaskConditionsNonUtility["UsingFlask"] = true
				flaskConditionsNonUtility["Using"+stripSpaces(baseName)] = true
			}
			if flaskBaseLife(item) && !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
				flaskConditions["UsingLifeFlask"] = true
				flaskConditionsNonUtility["UsingLifeFlask"] = true
			}
			if flaskBaseMana(item) {
				flaskConditions["UsingManaFlask"] = true
				flaskConditionsNonUtility["UsingManaFlask"] = true
			}
		}
		if baseName == "Iron Flask" {
			chargesGeneratedOnWardBreak := modDB.Sum(modparser.Base, nil, "IronFlaskChargesGeneratedOnWardBreak")
			if chargesGeneratedOnWardBreak > 0 {
				gainMod := item.In.FlaskData.GainMod * (1 + modDB.Sum(modparser.Inc, nil, "FlaskChargesGained")/100)
				chargesUsed := item.In.FlaskData.ChargesUsed * (1 + modDB.Sum(modparser.Inc, nil, "FlaskChargesUsed")/100)
				if chargesGeneratedOnWardBreak*gainMod > chargesUsed {
					flaskConditions["UnbrokenWard"] = true
				}
			}
		}

		if onlyRecovery {
			if flaskBaseLife(item) && !modDB.Flag(nil, "CannotRecoverLifeOutsideLeech") {
				calcFlaskMods(item, "LifeFlask", env.calcFlaskRecovery("Life", item, effectInc, flaskTotalRateInc, flaskDurInc), nil, false)
			}
			if flaskBaseMana(item) {
				calcFlaskMods(item, "ManaFlask", env.calcFlaskRecovery("Mana", item, effectInc, flaskTotalRateInc, flaskDurInc), nil, false)
			}
			if checkNonRecoveryFlasksForMinions {
				calcFlaskMods(item, baseName, item.In.BuffModList, item.In.ModList, true)
			}
		} else {
			calcFlaskMods(item, baseName, item.In.BuffModList, item.In.ModList, false)
		}
	}
	sortedListKeys := func(m map[string]*modstore.List) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	sortedBoolKeys := func(m map[string]bool) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	if modDB.Flag(nil, "UtilityFlasksDoNotApplyToPlayer") {
		for _, cond := range sortedBoolKeys(flaskConditionsNonUtility) {
			modDB.Conditions.Set(cond, flaskConditionsNonUtility[cond])
		}
		for _, k := range sortedListKeys(flaskBuffsNonUtility) {
			modDB.AddList(flaskBuffsNonUtility[k].Mods)
		}
	} else if !modDB.Flag(nil, "FlasksDoNotApplyToPlayer") {
		for _, cond := range sortedBoolKeys(flaskConditions) {
			modDB.Conditions.Set(cond, flaskConditions[cond])
		}
		for _, k := range sortedListKeys(flaskBuffs) {
			modDB.AddList(flaskBuffs[k].Mods)
		}
	}
	if env.Minion != nil {
		if flasksApplyToMinion || nonUniqueFlasksApplyToMinion {
			minionModDB := env.Minion.DB
			for _, cond := range sortedBoolKeys(flaskConditions) {
				minionModDB.Conditions.Set(cond, flaskConditions[cond])
			}
			for _, k := range sortedListKeys(flaskBuffsNonPlayer) {
				minionModDB.AddList(flaskBuffsNonPlayer[k].Mods)
			}
		} else if quickSilverAppliesToAllies && flaskBuffsPerBaseNonPlayer["Quicksilver Flask"] != nil {
			minionModDB := env.Minion.DB
			minionModDB.Conditions.Set("UsingQuicksilverFlask", flaskConditions["UsingQuicksilverFlask"])
			minionModDB.Conditions.Set("UsingFlask", flaskConditions["UsingFlask"])
			for _, k := range sortedListKeys(flaskBuffsPerBaseNonPlayer["Quicksilver Flask"]) {
				minionModDB.AddList(flaskBuffsPerBaseNonPlayer["Quicksilver Flask"][k].Mods)
			}
		}
	}
}

// mergeTinctures ports the local mergeTinctures.
func (env *Env) mergeTinctures(tinctures map[*Item]bool) {
	modDB := env.ModDB
	playerCfg := &modstore.Cfg{Actor: "player"}
	effectInc := modDB.Sum(modparser.Inc, playerCfg, "TinctureEffect")
	effectIncMagic := modDB.Sum(modparser.Inc, playerCfg, "MagicTinctureEffect")
	tinctureLimit := modDB.Sum(modparser.Base, nil, "TinctureLimit")

	tincturesNotInflictManaBurn := math.Min(modDB.Sum(modparser.Base, nil, "TincturesNotInflictManaBurn"), 100)
	canGainRequiredBurn := modDB.Flag(nil, "Condition:WeepingWoundsInsteadOfManaBurn") ||
		(tincturesNotInflictManaBurn < 100 && env.playerPA.output.N("ManaUnreserved") > 0)
	if !canGainRequiredBurn {
		return
	}
	tinctureBuffs := map[string]*modstore.List{}
	tinctureConditions := map[string]bool{}
	tinctureBuffsPerBase := map[string]map[string]*modstore.List{}

	calcTinctureMods := func(item *Item, baseName string, buffModList, modList []*modparser.Mod) {
		tinctureEffectInc := effectInc + item.In.TinctureData.EffectInc
		if item.In.Rarity == "MAGIC" {
			tinctureEffectInc += effectIncMagic
		}
		// effect multiplier rounded to 2 decimal places before applying
		quality := 0.0
		if item.In.Quality != nil {
			quality = *item.In.Quality
		}
		effectMod := math.Floor((1+tinctureEffectInc/100)*(1+quality/100)*100) / 100

		if len(buffModList) > 0 {
			srcList := modstore.NewList(nil)
			srcList.ScaleAddList(buffModList, effectMod, false)
			mergeBuff(srcList.Mods, tinctureBuffs, baseName)
			mergeBuff(srcList.Mods, tinctureBuffsPerBase[deref(item.In.BaseName)], baseName)
		}
		if len(modList) > 0 {
			srcList := modstore.NewList(nil)
			srcList.ScaleAddList(modList, effectMod, false)
			var key string
			if item.In.Rarity == "UNIQUE" || item.In.Rarity == "RELIC" {
				key = deref(item.In.Title)
			} else {
				key = ""
				for _, mod := range modList {
					key = key + modparser.FormatModParams(mod) + "&"
				}
			}
			mergeBuff(srcList.Mods, tinctureBuffs, key)
			mergeBuff(srcList.Mods, tinctureBuffsPerBase[deref(item.In.BaseName)], key)
		}
	}
	for _, item := range env.sortedFlasks(tinctures) {
		if tinctureLimit <= 0 {
			break
		}
		tinctureLimit--
		baseName := deref(item.In.BaseName)
		if tinctureBuffsPerBase[baseName] == nil {
			tinctureBuffsPerBase[baseName] = map[string]*modstore.List{}
		}
		tinctureConditions["UsingTincture"] = true
		tinctureConditions["Using"+stripSpaces(baseName)] = true
		calcTinctureMods(item, baseName, item.In.BuffModList, item.In.ModList)
	}
	condKeys := make([]string, 0, len(tinctureConditions))
	for k := range tinctureConditions {
		condKeys = append(condKeys, k)
	}
	sort.Strings(condKeys)
	for _, cond := range condKeys {
		modDB.Conditions.Set(cond, tinctureConditions[cond])
	}
	buffKeys := make([]string, 0, len(tinctureBuffs))
	for k := range tinctureBuffs {
		buffKeys = append(buffKeys, k)
	}
	sort.Strings(buffKeys)
	for _, k := range buffKeys {
		modDB.AddList(tinctureBuffs[k].Mods)
	}
	if modDB.Flag(nil, "TinctureRangedWeapons") {
		for _, k := range buffKeys {
			for _, buff := range tinctureBuffs[k].Mods {
				if buff.Flags&modparser.FlagWeaponMelee == modparser.FlagWeaponMelee {
					nm := cloneMod(buff)
					nm.Flags = (nm.Flags &^ modparser.FlagWeaponMelee) | modparser.FlagWeaponRanged
					modDB.AddList([]*modparser.Mod{nm})
				}
			}
		}
	}
}
