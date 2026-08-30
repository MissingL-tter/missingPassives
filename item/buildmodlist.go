// BuildModList / BuildModListForSlotNum / calcLocal / getRangedModList:
// Item.lua L2148-2653.
package item

import (
	"math"
	"regexp"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

func newMod(name string, typ modparser.ModType, value modparser.Value, tags ...modparser.Tag) *modparser.Mod {
	return modparser.NewMod(name, typ, value, tags...)
}

// newModS is newMod with a source.
func newModS(name string, typ modparser.ModType, value modparser.Value, source string, tags ...modparser.Tag) *modparser.Mod {
	return modparser.NewModFull(name, typ, value, source, true, 0, 0, tags...)
}

// modTag1 is mod[1]: the first tag, nil when absent.
func modTag1(m *modparser.Mod) modparser.Tag {
	tags := modparser.ModTags(m)
	if len(tags) == 0 {
		return nil
	}
	return tags[0]
}

func modTag2Absent(m *modparser.Mod) bool {
	return len(modparser.ModTags(m)) < 2
}

// isInSlot reports an InSlot tag.
func isInSlot(tag modparser.Tag) bool {
	st, ok := tag.(*modparser.SlotTag)
	return ok && st.SlotKind == modparser.TagInSlot
}

// calcLocal ports the file-local calcLocal for numeric types (BASE/INC sum,
// MORE times-multiplier), removing matched mods from the list.
func calcLocal(list *[]*modparser.Mod, name string, typ modparser.ModType, flags modparser.ModFlag) float64 {
	result := 0.0
	if typ == modparser.More {
		result = 1
	}
	i := 0
	for i < len(*list) {
		mod := (*list)[i]
		tag1 := modTag1(mod)
		if mod.Name == name && mod.Type == typ && mod.Flags == flags && mod.KeywordFlags == 0 &&
			(tag1 == nil || isInSlot(tag1)) {
			value := num(mod.Value)
			if typ == modparser.More {
				result = result * ((100 + value) / 100)
			} else {
				result = result + value
			}
			*list = append((*list)[:i], (*list)[i+1:]...)
		} else {
			i++
		}
	}
	return result
}

// calcLocalFlag is calcLocal with type FLAG: boolean or.
func calcLocalFlag(list *[]*modparser.Mod, name string, flags modparser.ModFlag) bool {
	result := false
	i := 0
	for i < len(*list) {
		mod := (*list)[i]
		tag1 := modTag1(mod)
		if mod.Name == name && mod.Type == modparser.Flag && mod.Flags == flags && mod.KeywordFlags == 0 &&
			(tag1 == nil || isInSlot(tag1)) {
			if b, ok := mod.Value.(modparser.Bool); ok {
				result = result || bool(b)
			} else if modparser.Truthy(mod.Value) {
				result = true
			}
			*list = append((*list)[:i], (*list)[i+1:]...)
		} else {
			i++
		}
	}
	return result
}

// listValues is modList:List(nil, name) for the tagless LIST mods items
// carry; a tagged match panics rather than silently skipping.
func listValues(list []*modparser.Mod, name string) []modparser.Value {
	var out []modparser.Value
	for _, mod := range list {
		if mod.Name == name && mod.Type == modparser.List {
			if modTag1(mod) != nil {
				panic("item: tagged " + name + " LIST mod needs the full evaluator")
			}
			out = append(out, mod.Value)
		}
	}
	return out
}

// weaponBase reads self.base.weapon[key]; only physical base damage exists
// in the data.
func weaponBase(w *data.WeaponData, dmgType, side string) float64 {
	if dmgType == "Physical" {
		if side == "Min" {
			return w.PhysicalMin
		}
		return w.PhysicalMax
	}
	return 0
}

// BuildModListForSlotNum ports ItemClass:BuildModListForSlotNum. slotNum 0
// is the Lua nil call (non-weapon, non-ring items).
func (it *Item) BuildModListForSlotNum(baseList *[]*modparser.Mod, slotNum int) []*modparser.Mod {
	slotName := it.GetPrimarySlot()
	if slotNum != 1 {
		// tostring(nil) == "nil": flasks and grafts genuinely get
		// "Flask nil" here in the reference; keep it byte-faithful.
		numStr := "nil"
		if slotNum != 0 {
			numStr = util.FormatIntOrG14(float64(slotNum))
		}
		slotName = strings.ReplaceAll(slotName, "1", numStr)
	}
	modList := make([]*modparser.Mod, 0, len(*baseList)+8)
	for _, baseMod := range *baseList {
		mod := baseMod.Clone()
		add := true
		for _, tag := range modparser.ModTags(mod) {
			if st, ok := tag.(*modparser.SlotTag); ok && (st.SlotKind == modparser.TagSlotNumber || st.SlotKind == modparser.TagInSlot) {
				if int(st.Num) != slotNum || slotNum == 0 {
					add = false
					break
				}
			}
			tag.ReplaceStrings(func(s string) string {
				s = strings.ReplaceAll(s, "{SlotName}", slotName)
				if slotNum == 1 {
					s = strings.ReplaceAll(s, "{Hand}", "MainHand")
					return strings.ReplaceAll(s, "{OtherSlotNum}", "2")
				}
				s = strings.ReplaceAll(s, "{Hand}", "OffHand")
				return strings.ReplaceAll(s, "{OtherSlotNum}", "1")
			})
		}
		if add {
			mod.SourceSlot = slotName
			modList = append(modList, mod)
		}
	}
	if len(it.Sockets) > 0 {
		multiName := map[string]string{
			"R": "Multiplier:RedSocketIn" + slotName,
			"G": "Multiplier:GreenSocketIn" + slotName,
			"B": "Multiplier:BlueSocketIn" + slotName,
			"W": "Multiplier:WhiteSocketIn" + slotName,
		}
		groupCounts := map[float64]int{}
		for _, socket := range it.Sockets {
			groupCounts[socket.Group]++
			if name, ok := multiName[socket.Color]; ok {
				modList = append(modList, newModS(name, modparser.Base, modparser.Num(1), "Item Sockets"))
			}
		}
		unlinkedSockets := 0
		for _, count := range groupCounts {
			if count == 1 {
				unlinkedSockets++
			}
		}
		modList = append(modList, newModS("Multiplier:UnlinkedSocketIn"+slotName, modparser.Base, modparser.Num(unlinkedSockets), "Unlinked Item Sockets"))
	}
	craftedQuality := calcLocal(&modList, "Quality", modparser.Base, 0)
	if it.CraftedQuality == nil || craftedQuality != *it.CraftedQuality {
		if it.CraftedQuality != nil {
			q := 0.0
			if it.Quality != nil {
				q = *it.Quality
			}
			q = q - *it.CraftedQuality + craftedQuality
			it.Quality = &q
		}
		cq := craftedQuality
		it.CraftedQuality = &cq
	}
	if it.Quality != nil {
		modList = append(modList, newModS("Multiplier:QualityOn"+slotName, modparser.Base, modparser.Num(*it.Quality), "Quality"))
	}
	quality := 0.0
	if it.Quality != nil {
		quality = *it.Quality
	}
	if it.Base.Weapon != nil {
		weaponData := &WeaponData{Type: it.Base.Type, SubType: it.Base.SubType, Name: it.Name}
		if it.WeaponData == nil {
			it.WeaponData = map[int]*WeaponData{}
		}
		it.WeaponData[slotNum] = weaponData
		attackSpeedInc := calcLocal(&modList, "Speed", modparser.Inc, modparser.FlagAttack) +
			math.Floor(quality/8*calcLocal(&modList, "AlternateQualityLocalAttackSpeedPer8Quality", modparser.Inc, 0))
		weaponData.AttackSpeedInc = util.Some(attackSpeedInc)
		weaponData.AttackRate = util.RoundHalfUp(it.Base.Weapon.AttackRateBase*(1+attackSpeedInc/100), 2)
		rangeBonus := calcLocal(&modList, "WeaponRange", modparser.Base, 0) +
			10*calcLocal(&modList, "WeaponRangeMetre", modparser.Base, 0) +
			math.Floor(quality/10*calcLocal(&modList, "AlternateQualityLocalWeaponRangePer10Quality", modparser.Base, 0))
		weaponData.RangeBonus = util.Some(rangeBonus)
		weaponData.Range = it.Base.Weapon.Range + rangeBonus
		localIncEle := calcLocal(&modList, "LocalElementalDamage", modparser.Inc, 0)
		for _, dmgType := range dmgTypeList {
			min := weaponBase(it.Base.Weapon, dmgType, "Min") + calcLocal(&modList, dmgType+"Min", modparser.Base, 0)
			max := weaponBase(it.Base.Weapon, dmgType, "Max") + calcLocal(&modList, dmgType+"Max", modparser.Base, 0)
			if dmgType == "Physical" {
				physInc := calcLocal(&modList, "PhysicalDamage", modparser.Inc, 0)
				qualityScalar := quality
				if calcLocal(&modList, "AlternateQualityWeapon", modparser.Base, 0) > 0 {
					qualityScalar = 0
				}
				min = util.RoundHalfUp(min*(1+physInc/100)*(1+qualityScalar/100), 0)
				max = util.RoundHalfUp(max*(1+physInc/100)*(1+qualityScalar/100), 0)
			} else if dmgType != "Chaos" {
				localInc := calcLocal(&modList, "Local"+dmgType+"Damage", modparser.Inc, 0) + localIncEle
				min = util.RoundHalfUp(min*(1+localInc/100), 0)
				max = util.RoundHalfUp(max*(1+localInc/100), 0)
			}
			if min > 0 && max > 0 {
				dps := (min + max) / 2 * weaponData.AttackRate
				*weaponData.Damage(dmgType) = DamageRange{Min: min, Max: max, DPS: dps}
				if dmgType != "Physical" && dmgType != "Chaos" {
					weaponData.ElementalDPS += dps
				}
			}
		}
		weaponData.CritChance = util.Some(util.RoundHalfUp(
			(it.Base.Weapon.CritChanceBase+calcLocal(&modList, "CritChance", modparser.Base, 0))*
				(1+(calcLocal(&modList, "CritChance", modparser.Inc, 0)+math.Floor(quality/4*calcLocal(&modList, "AlternateQualityLocalCritChancePer4Quality", modparser.Inc, 0)))/100), 2))
		for _, v := range listValues(modList, "WeaponData") {
			kv := v.(modparser.DataRef)
			weaponData.Set(kv.Key, kv.Value)
		}
		for _, mod := range modList {
			tag1 := modTag1(mod)
			condVar := "OffHandAttack"
			if slotNum == 1 {
				condVar = "MainHandAttack"
			}
			if ((mod.Name == "Accuracy" && mod.Flags == 0) || (mod.Name == "ImpaleChance" && mod.Flags != modparser.FlagSpell) ||
				((mod.Name == "LifeOnHit" || mod.Name == "ManaOnHit") && mod.Flags == modparser.FlagAttack) ||
				((mod.Name == "PhysicalDamageLifeLeech" || mod.Name == "PhysicalDamageManaLeech") && mod.Flags == modparser.FlagAttack)) &&
				(mod.KeywordFlags == 0 || mod.KeywordFlags == modparser.KeywordAttack) && tag1 == nil {
				setFirstTag(mod, &modparser.CondTag{Var: condVar})
			} else if (mod.Name == "PoisonChance" || mod.Name == "BleedChance") && mod.Flags != modparser.FlagSpell &&
				(tag1 == nil || (isCritCond(tag1) && modTag2Absent(mod))) {
				appendTag(mod, &modparser.CondTag{Var: condVar})
			}
		}
		totalDPS := 0.0
		for _, dmgType := range dmgTypeList {
			totalDPS += weaponData.Damage(dmgType).DPS
		}
		weaponData.TotalDPS = util.Some(totalDPS)
	} else if it.Base.Armour != nil {
		armourData := it.ArmourData
		ab := it.Base.Armour
		armourBase := calcLocal(&modList, "Armour", modparser.Base, 0) + orZero(ab.ArmourBaseMin)
		armourVariance := orZero(ab.ArmourBaseMax) - orZero(ab.ArmourBaseMin)
		armourEvasionBase := calcLocal(&modList, "ArmourAndEvasion", modparser.Base, 0)
		evasionBase := calcLocal(&modList, "Evasion", modparser.Base, 0) + orZero(ab.EvasionBaseMin)
		evasionVariance := orZero(ab.EvasionBaseMax) - orZero(ab.EvasionBaseMin)
		evasionEnergyShieldBase := calcLocal(&modList, "EvasionAndEnergyShield", modparser.Base, 0)
		energyShieldBase := calcLocal(&modList, "EnergyShield", modparser.Base, 0) + orZero(ab.EnergyShieldBaseMin)
		energyShieldVariance := orZero(ab.EnergyShieldBaseMax) - orZero(ab.EnergyShieldBaseMin)
		armourEnergyShieldBase := calcLocal(&modList, "ArmourAndEnergyShield", modparser.Base, 0)
		wardBase := calcLocal(&modList, "Ward", modparser.Base, 0) + orZero(ab.WardBaseMin)
		wardVariance := orZero(ab.WardBaseMax) - orZero(ab.WardBaseMin)
		armourInc := calcLocal(&modList, "Armour", modparser.Inc, 0)
		armourEvasionInc := calcLocal(&modList, "ArmourAndEvasion", modparser.Inc, 0)
		evasionInc := calcLocal(&modList, "Evasion", modparser.Inc, 0)
		evasionEnergyShieldInc := calcLocal(&modList, "EvasionAndEnergyShield", modparser.Inc, 0)
		energyShieldInc := calcLocal(&modList, "EnergyShield", modparser.Inc, 0)
		wardInc := calcLocal(&modList, "Ward", modparser.Inc, 0)
		armourEnergyShieldInc := calcLocal(&modList, "ArmourAndEnergyShield", modparser.Inc, 0)
		defencesInc := calcLocal(&modList, "Defences", modparser.Inc, 0)
		qualityScalar := quality
		if calcLocal(&modList, "AlternateQualityArmour", modparser.Base, 0) > 0 {
			qualityScalar = 0
		}
		// A parsed stat backs out its base-roll percentile before the
		// stat is recomputed from the base.
		percentile := func(stat *DefenceStat, base, variance, inc float64) {
			if stat.Value.Set && stat.Value.V > 0 && !stat.BasePercentile.Set {
				p := (stat.Value.V/((1+inc/100)*(1+qualityScalar/100)) - base) / variance
				stat.BasePercentile = util.Some(util.RoundHalfUp(math.Max(math.Min(p, 1), 0), 4))
			}
		}
		percentile(&armourData.Armour, armourBase, armourVariance, armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)
		percentile(&armourData.Evasion, evasionBase, evasionVariance, evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)
		percentile(&armourData.EnergyShield, energyShieldBase, energyShieldVariance, energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)
		percentile(&armourData.Ward, wardBase, wardVariance, wardInc+defencesInc)
		pct := func(stat *DefenceStat) float64 { return stat.BasePercentile.Or(1) }
		armourData.Armour.Value = util.Some(util.RoundHalfUp((armourBase+armourEvasionBase+armourEnergyShieldBase+armourVariance*pct(&armourData.Armour))*(1+(armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100), 0))
		armourData.Evasion.Value = util.Some(util.RoundHalfUp((evasionBase+armourEvasionBase+evasionEnergyShieldBase+evasionVariance*pct(&armourData.Evasion))*(1+(evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100), 0))
		armourData.EnergyShield.Value = util.Some(util.RoundHalfUp((energyShieldBase+evasionEnergyShieldBase+armourEnergyShieldBase+energyShieldVariance*pct(&armourData.EnergyShield))*(1+(energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100), 0))
		armourData.Ward.Value = util.Some(util.RoundHalfUp((wardBase+wardVariance*pct(&armourData.Ward))*(1+(wardInc+defencesInc)/100)*(1+qualityScalar/100), 0))
		for _, stat := range []*DefenceStat{&armourData.Armour, &armourData.Evasion, &armourData.EnergyShield, &armourData.Ward} {
			if !stat.BasePercentile.Set && stat.Value.V > 0 {
				stat.BasePercentile = util.Some(1.0)
			}
		}
		if ab.BlockChance != nil {
			armourData.BlockChance = util.Some(math.Floor((*ab.BlockChance + calcLocal(&modList, "BlockChance", modparser.Base, 0)) * (1 + calcLocal(&modList, "BlockChance", modparser.Inc, 0)/100)))
		}
		if ab.MovementPenalty != nil {
			modList = append(modList, newModS("MovementSpeed", modparser.Inc, modparser.Num(-*ab.MovementPenalty), it.ModSource,
				&modparser.CondTag{Var: "IgnoreMovementPenalties", Neg: true}))
		}
		for _, v := range listValues(modList, "ArmourData") {
			kv := v.(modparser.DataRef)
			armourData.Set(kv.Key, kv.Value)
		}
	} else if it.Base.Flask != nil {
		flaskData := it.FlaskData
		fb := it.Base.Flask
		durationInc := calcLocal(&modList, "Duration", modparser.Inc, 0)
		durationMore := calcLocal(&modList, "Duration", modparser.More, 0)
		if fb.Life != nil || fb.Mana != nil {
			instantPerc := calcLocal(&modList, "FlaskInstantRecovery", modparser.Base, 0)
			flaskData.InstantPerc = util.Some(instantPerc)
			flaskData.InstantLowLifePerc = util.Some(calcLocal(&modList, "FlaskLowLifeInstantRecovery", modparser.Base, 0))
			recoveryMod := 1 + calcLocal(&modList, "FlaskRecovery", modparser.Inc, 0)/100
			rateMod := 1 + calcLocal(&modList, "FlaskRecoveryRate", modparser.Inc, 0)/100
			flaskData.Duration = util.RoundHalfUp(fb.Duration*(1+durationInc/100)/rateMod*durationMore, 1)
			recovery := func(amount float64) *FlaskRecovery {
				base := amount * (1 + quality/100) * recoveryMod
				rec := &FlaskRecovery{Base: base, Instant: base * instantPerc / 100, Gradual: base * (1 - instantPerc/100)}
				rec.Total = rec.Instant + rec.Gradual
				return rec
			}
			if fb.Life != nil {
				flaskData.Life = recovery(*fb.Life)
				flaskData.Life.Additional = util.Some(calcLocal(&modList, "FlaskAdditionalLifeRecovery", modparser.Base, 0))
				flaskData.Life.EffectNotRemoved = calcLocalFlag(baseList, "LifeFlaskEffectNotRemoved", 0)
			}
			if fb.Mana != nil {
				flaskData.Mana = recovery(*fb.Mana)
				flaskData.Mana.EffectNotRemoved = calcLocalFlag(baseList, "ManaFlaskEffectNotRemoved", 0)
			}
		} else {
			flaskData.Duration = util.RoundHalfUp(fb.Duration*(1+durationInc/100)*(1+quality/100)*durationMore, 1)
		}
		flaskData.ChargesMax = fb.ChargesMax + calcLocal(&modList, "FlaskCharges", modparser.Base, 0)
		flaskData.ChargesUsed = math.Floor(fb.ChargesUsed * (1 + calcLocal(&modList, "FlaskChargesUsed", modparser.Inc, 0)/100))
		flaskData.GainMod = 1 + calcLocal(&modList, "FlaskChargeRecovery", modparser.Inc, 0)/100
		flaskData.EffectInc = calcLocal(&modList, "FlaskEffect", modparser.Inc, 0) + calcLocal(&modList, "LocalEffect", modparser.Inc, 0)
		for _, v := range listValues(modList, "FlaskData") {
			kv := v.(modparser.DataRef)
			flaskData.Set(kv.Key, kv.Value)
		}
	} else if it.Base.Tincture != nil {
		tinctureData := it.TinctureData
		tb := it.Base.Tincture
		tinctureData.ManaBurn = (tb.ManaBurn + 0.01) / (1 + calcLocal(&modList, "TinctureManaBurnRate", modparser.Inc, 0)/100) / (1 + calcLocal(&modList, "TinctureManaBurnRate", modparser.More, 0)/100)
		cooldownInc := calcLocal(&modList, "TinctureCooldownRecovery", modparser.Inc, 0) + calcLocal(&modList, "CooldownRecovery", modparser.Inc, 0)
		tinctureData.CooldownInc = cooldownInc
		tinctureData.Cooldown = tb.Cooldown / (1 + cooldownInc/100)
		tinctureData.EffectInc = calcLocal(&modList, "TinctureEffect", modparser.Inc, 0) + calcLocal(&modList, "LocalEffect", modparser.Inc, 0)
		for _, v := range listValues(modList, "TinctureData") {
			kv := v.(modparser.DataRef)
			tinctureData.Set(kv.Key, kv.Value)
		}
	} else if it.Type == "Jewel" {
		if strings.Contains(it.Name, "Grand Spectrum") {
			spectrumMod := newModS("Multiplier:GrandSpectrum", modparser.Base, modparser.Num(1), it.Name)
			modList = append(modList, spectrumMod)
			modList = append(modList, newModS("MinionModifier", modparser.List, modparser.ModRef{Mod: spectrumMod}, it.Name))
		}
		jewelData := it.JewelData
		for _, fn := range listValues(modList, "JewelFunc") {
			jewelData.FuncList = append(jewelData.FuncList, fn.(modparser.JewelFn))
		}
		for _, v := range listValues(modList, "JewelData") {
			kv := v.(modparser.DataRef)
			jewelData.Set(kv.Key, kv.Value)
		}
		if keystones := listValues(modList, "ImpossibleEscapeKeystones"); keystones != nil {
			jewelData.ImpossibleEscapeKeystones = map[string]bool{}
			for _, v := range keystones {
				kv := v.(modparser.DataRef)
				jewelData.ImpossibleEscapeKeystones[kv.Key] = modparser.Truthy(kv.Value)
			}
		}
		if it.ClusterJewel != nil {
			jewelData.ClusterJewelNotables = []string{}
			for _, name := range listValues(modList, "ClusterJewelNotable") {
				jewelData.ClusterJewelNotables = append(jewelData.ClusterJewelNotables, string(name.(modparser.Str)))
			}
			jewelData.ClusterJewelAddedMods = []string{}
			for _, line := range listValues(modList, "AddToClusterJewelNode") {
				jewelData.ClusterJewelAddedMods = append(jewelData.ClusterJewelAddedMods, string(line.(modparser.Str)))
			}
			if it.ClusterJewel.Size == "Small" && jewelData.ClusterJewelSkill == "affliction_curse_effect" {
				jewelData.ClusterJewelSkill = "affliction_curse_effect_small"
			} else if it.ClusterJewel.Size == "Medium" && jewelData.ClusterJewelSkill == "affliction_curse_effect_small" {
				jewelData.ClusterJewelSkill = "affliction_curse_effect"
			}
			if jewelData.ClusterJewelNodeCount != 0 {
				jewelData.ClusterJewelNodeCount = int(fmin(fmax(float64(jewelData.ClusterJewelNodeCount), it.ClusterJewel.MinNodes), it.ClusterJewel.MaxNodes))
			}
			if _, valid := it.ClusterJewel.Skills[jewelData.ClusterJewelSkill]; !valid {
				jewelData.ClusterJewelSkill = ""
			}
			if it.ClusterJewelSkill == "" {
				it.ClusterJewelSkill = jewelData.ClusterJewelSkill
			}
			if it.ClusterJewelNodeCount == nil && jewelData.ClusterJewelNodeCount != 0 {
				c := float64(jewelData.ClusterJewelNodeCount)
				it.ClusterJewelNodeCount = &c
			}
			jewelData.ClusterJewelValid = jewelData.ClusterJewelKeystone != "" ||
				((jewelData.ClusterJewelSkill != "" || jewelData.ClusterJewelSmallsAreNothingness) && jewelData.ClusterJewelNodeCount != 0) ||
				(jewelData.ClusterJewelSocketCountOverride != 0 && jewelData.ClusterJewelNothingnessCount != 0)
		}
	}
	return modList
}

func orZero(p *float64) float64 {
	if p != nil {
		return *p
	}
	return 0
}

// isCritCond reports a Condition tag on CriticalStrike.
func isCritCond(tag modparser.Tag) bool {
	ct, ok := tag.(*modparser.CondTag)
	return ok && !ct.IsActor && ct.Var == "CriticalStrike"
}

func setFirstTag(m *modparser.Mod, tag modparser.Tag) {
	if len(m.Tags) == 0 {
		m.Tags = append(m.Tags, tag)
	} else {
		m.Tags[0] = tag
	}
}

func appendTag(m *modparser.Mod, tag modparser.Tag) {
	// t_insert appends after the contiguous prefix.
	for i, tv := range m.Tags {
		if tv == nil {
			m.Tags[i] = tag
			return
		}
	}
	m.Tags = append(m.Tags, tag)
}

var rangeAnyRe = regexp.MustCompile(`\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)

// getRangedModList ports the file-local getRangedModList. Returns
// (list, isEmpty, ok): ok=false is the Lua nil/false return.
func (it *Item) getRangedModList(modLine *ModLine) ([]*modparser.Mod, bool, bool) {
	if modLine.Range == nil || !rangeAnyRe.MatchString(modLine.Line) {
		return nil, false, false
	}
	line := applyRange(strings.ReplaceAll(modLine.Line, "\n", " "), *modLine.Range, modLine.ValueScalar.Or(1), corruptedOr1(modLine))
	list, extra, parsed := parseModLine3(line)
	if isZeroValueLine(line) {
		return nil, true, true
	}
	if extra != "" || !parsed {
		return nil, false, false
	}
	return list, false, true
}

var requiresClassRe = regexp.MustCompile(`Requires Class (.+)`)
var variantTagStripRe = regexp.MustCompile(`\{variant:([\d,]+)\}`)

// BuildModList ports ItemClass:BuildModList.
func (it *Item) BuildModList() {
	if it.Base == nil {
		return
	}
	baseList := make([]*modparser.Mod, 0, 16)
	if it.Base.Weapon != nil {
		it.WeaponData = map[int]*WeaponData{}
	} else if it.Base.Armour != nil {
		if it.ArmourData == nil {
			it.ArmourData = &ArmourData{}
		}
	} else if it.Base.Flask != nil {
		it.FlaskData = &FlaskData{}
		it.BuffModList = nil
		it.BuffModListInit = true
	} else if it.Base.Tincture != nil {
		it.TinctureData = &TinctureData{}
		it.BuffModList = nil
		it.BuffModListInit = true
	} else if it.Type == "Jewel" {
		it.JewelData = &JewelData{}
	}
	it.RangeLineList = nil
	id := it.ID
	if id == 0 {
		id = -1
	}
	it.ModSource = "Item:" + util.FormatIntOrG14(float64(id)) + ":" + it.Name
	for _, modLine := range it.BuffModLines {
		if modLine.Extra == "" && it.CheckModLineVariant(modLine) {
			for _, mod := range modLine.ModList {
				mod.Source = it.ModSource
				mod.SourceSet = true
				it.BuffModList = append(it.BuffModList, mod)
			}
		}
	}
	processModLine := func(modLine *ModLine) {
		if modLine.flag("disabled") {
			return
		}
		variantCount := it.GetModLineVariantCount(modLine)
		if variantCount > 0 {
			if strings.Contains(modLine.Line, "Requires Class") {
				stripped := variantTagStripRe.ReplaceAllString(modLine.Line, "")
				if m := requiresClassRe.FindStringSubmatch(stripped); m != nil {
					it.ClassRestriction = m[1]
				}
			}
			rangedApplied := false
			if modLine.Extra == "" {
				if list, isEmpty, ok := it.getRangedModList(modLine); ok {
					if isEmpty {
						modLine.ModList = nil
					} else {
						modLine.ModList = list
					}
					modLine.HasModList = true
					modLine.ShowSlider = true
					it.RangeLineList = append(it.RangeLineList, modLine)
					rangedApplied = true
				}
			}
			if !rangedApplied && modLine.ModID != "" && modLine.NewModID != "" {
				it.RangeLineList = append(it.RangeLineList, modLine)
			}
			if modLine.Extra == "" {
				for _, mod := range modLine.ModList {
					for range variantCount {
						baseList = append(baseList, modparser.SetSource(mod, it.ModSource))
					}
				}
				if len(modLine.ModTags) > 0 {
					it.HasModTags = true
				}
			}
		}
	}
	for _, modLine := range it.EnchantModLines {
		processModLine(modLine)
	}
	for _, modLine := range it.ScourgeModLines {
		processModLine(modLine)
	}
	for _, modLine := range it.ClassRequirementModLines {
		processModLine(modLine)
	}
	for _, modLine := range it.ImplicitModLines {
		processModLine(modLine)
	}
	for _, modLine := range it.ExplicitModLines {
		processModLine(modLine)
	}
	for _, modLine := range it.CrucibleModLines {
		processModLine(modLine)
	}
	if it.Name == "Tabula Rasa, Simple Robe" || it.Name == "Skin of the Loyal, Simple Robe" ||
		it.Name == "Skin of the Lords, Simple Robe" || it.Name == "The Apostate, Cabalist Regalia" {
		baseList = append(baseList, newMod("ArmourData", modparser.List, modparser.DataRef{Key: "EnergyShield", Value: modparser.Num(0)}))
		it.Requirements.Int = util.Some(0.0)
	}
	req := &it.Requirements
	if calcLocalFlag(&baseList, "NoAttributeRequirements", 0) {
		req.StrMod, req.DexMod, req.IntMod = util.Some(0.0), util.Some(0.0), util.Some(0.0)
	} else {
		req.StrMod = util.Some(math.Floor((req.Str.Or(0) + calcLocal(&baseList, "StrRequirement", modparser.Base, 0)) * (1 + calcLocal(&baseList, "StrRequirement", modparser.Inc, 0)/100)))
		req.DexMod = util.Some(math.Floor((req.Dex.Or(0) + calcLocal(&baseList, "DexRequirement", modparser.Base, 0)) * (1 + calcLocal(&baseList, "DexRequirement", modparser.Inc, 0)/100)))
		req.IntMod = util.Some(math.Floor((req.Int.Or(0) + calcLocal(&baseList, "IntRequirement", modparser.Base, 0)) * (1 + calcLocal(&baseList, "IntRequirement", modparser.Inc, 0)/100)))
	}
	it.GrantedSkills = []GrantedSkill{}
	for _, v := range listValues(baseList, "ExtraSkill") {
		kv := v.(modparser.SkillRef)
		it.GrantedSkills = append(it.GrantedSkills, GrantedSkill{
			SkillID: kv.SkillID, Source: it.ModSource, Level: kv.Level,
			NoSupports: kv.NoSupports, Triggered: kv.Triggered, TriggerChance: kv.TriggerChance,
		})
	}
	socketCount := calcLocal(&baseList, "SocketCount", modparser.Base, 0)
	it.AbyssalSocketCount = calcLocal(&baseList, "AbyssalSocketCount", modparser.Base, 0)
	socketLimit := 0.0
	if it.Base.SocketLimit != nil {
		socketLimit = *it.Base.SocketLimit
	}
	it.SelectableSocketCount = fmax(socketLimit, float64(len(it.Sockets))) - it.AbyssalSocketCount
	if calcLocalFlag(&baseList, "NoSockets", 0) {
		it.Sockets = nil
		it.SelectableSocketCount = 0
		it.AbyssalSocketCount = 0
	} else if socketCount > 0 {
		it.SelectableSocketCount = socketCount
		group := 0.0
		n := imax(int(socketCount), len(it.Sockets))
		newSockets := make([]*Socket, 0, n)
		for i := 1; i <= n; i++ {
			if i > int(socketCount) {
				break
			}
			if i <= len(it.Sockets) && it.Sockets[i-1] != nil {
				group = it.Sockets[i-1].Group
				newSockets = append(newSockets, it.Sockets[i-1])
			} else {
				newSockets = append(newSockets, &Socket{Color: it.DefaultSocketColor, Group: group})
			}
		}
		it.Sockets = newSockets
	} else if it.AbyssalSocketCount > 0 {
		var newSockets []*Socket
		group := 0.0
		for _, socket := range it.Sockets {
			if socket.Color != "A" {
				if float64(len(newSockets)) >= it.SelectableSocketCount {
					break
				}
				newSockets = append(newSockets, socket)
				group = socket.Group
			}
		}
		for i := 0; i < int(it.AbyssalSocketCount); i++ {
			group++
			newSockets = append(newSockets, &Socket{Color: "A", Group: group})
		}
		it.Sockets = newSockets
	}
	if it.Sockets != nil && calcLocalFlag(&baseList, "SocketAlwaysMatches", 0) {
		it.SocketColourAlwaysMatches = true
	}
	it.SocketedJewelEffectModifier = 1 + calcLocal(&baseList, "SocketedJewelEffect", modparser.Inc, 0)/100
	if it.Base.Weapon != nil || it.Type == "Ring" {
		it.SlotModList = map[int][]*modparser.Mod{}
		it.ModList = nil
		for i := 1; i <= 2; i++ {
			it.SlotModList[i] = it.BuildModListForSlotNum(&baseList, i)
		}
		if it.Type == "Ring" {
			it.SlotModList[3] = it.BuildModListForSlotNum(&baseList, 3)
		}
	} else {
		it.SlotModList = nil
		it.ModList = it.BuildModListForSlotNum(&baseList, 0)
	}
	it.BaseModList = baseList
}
