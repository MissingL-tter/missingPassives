// BuildModList / BuildModListForSlotNum / calcLocal / getRangedModList:
// Item.lua L2148-2653.
package item

import (
	"math"
	"regexp"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

func newMod(name, typ string, value any, rest ...any) *modparser.Mod {
	return modparser.NewMod(name, typ, value, rest...)
}

// truthy is Lua truthiness for the and/or value chains.
func truthy(v any) bool { return v != nil && v != false }

// modTag1 is mod[1]: the first tag, nil when absent. Tags can be stored
// as *D (transformed pattern tables); ModTags views those as Tag.
func modTag1(m *modparser.Mod) modparser.Tag {
	tags := modparser.ModTags(m)
	if len(tags) == 0 {
		return nil
	}
	if tags[0] == nil {
		panic("item: first tag has a shape ModTags cannot view")
	}
	return tags[0]
}

func modTag2Absent(m *modparser.Mod) bool {
	return len(modparser.ModTags(m)) < 2
}

// tagNum reads tag.num as a number.
func tagNum(tag modparser.Tag) (float64, bool) {
	switch n := tag["num"].(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	}
	return 0, false
}

// calcLocal ports the file-local calcLocal for numeric types (BASE/INC sum,
// MORE times-multiplier), removing matched mods from the list.
func calcLocal(list *[]*modparser.Mod, name, typ string, flags int64) float64 {
	result := 0.0
	if typ == "MORE" {
		result = 1
	}
	i := 0
	for i < len(*list) {
		mod := (*list)[i]
		tag1 := modTag1(mod)
		if mod.Name == name && mod.Type == typ && mod.Flags == flags && mod.KeywordFlags == 0 &&
			(tag1 == nil || tag1["type"] == "InSlot") {
			value := numOf(mod.Value)
			if typ == "MORE" {
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
func calcLocalFlag(list *[]*modparser.Mod, name string, flags int64) bool {
	result := false
	i := 0
	for i < len(*list) {
		mod := (*list)[i]
		tag1 := modTag1(mod)
		if mod.Name == name && mod.Type == "FLAG" && mod.Flags == flags && mod.KeywordFlags == 0 &&
			(tag1 == nil || tag1["type"] == "InSlot") {
			if b, ok := mod.Value.(bool); ok {
				result = result || b
			} else if truthy(mod.Value) {
				result = true
			}
			*list = append((*list)[:i], (*list)[i+1:]...)
		} else {
			i++
		}
	}
	return result
}

func numOf(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	panic("item: calcLocal on a non-numeric mod value")
}

// listValues is modList:List(nil, name) for the tagless LIST mods items
// carry; a tagged match panics rather than silently skipping.
func listValues(list []*modparser.Mod, name string) []any {
	var out []any
	for _, mod := range list {
		if mod.Name == name && mod.Type == "LIST" {
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
			numStr = luaNumStr(float64(slotNum))
		}
		slotName = strings.ReplaceAll(slotName, "1", numStr)
	}
	modList := make([]*modparser.Mod, 0, len(*baseList)+8)
	for _, baseMod := range *baseList {
		mod := modparser.CopyMod(baseMod)
		add := true
		for _, tag := range modparser.ModTags(mod) {
			if tag == nil {
				panic("item: tag shape ModTags cannot view on mod " + mod.Name)
			}
			if tag["type"] == "SlotNumber" || tag["type"] == "InSlot" {
				if num, isNum := tagNum(tag); !isNum || int(num) != slotNum || slotNum == 0 {
					add = false
					break
				}
			}
			for k, v := range tag {
				if s, isStr := v.(string); isStr {
					s = strings.ReplaceAll(s, "{SlotName}", slotName)
					if slotNum == 1 {
						s = strings.ReplaceAll(s, "{Hand}", "MainHand")
						s = strings.ReplaceAll(s, "{OtherSlotNum}", "2")
					} else {
						s = strings.ReplaceAll(s, "{Hand}", "OffHand")
						s = strings.ReplaceAll(s, "{OtherSlotNum}", "1")
					}
					tag[k] = s
				}
			}
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
				modList = append(modList, newMod(name, "BASE", 1.0, "Item Sockets"))
			}
		}
		unlinkedSockets := 0
		for _, count := range groupCounts {
			if count == 1 {
				unlinkedSockets++
			}
		}
		modList = append(modList, newMod("Multiplier:UnlinkedSocketIn"+slotName, "BASE", float64(unlinkedSockets), "Unlinked Item Sockets"))
	}
	craftedQuality := calcLocal(&modList, "Quality", "BASE", 0)
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
		modList = append(modList, newMod("Multiplier:QualityOn"+slotName, "BASE", *it.Quality, "Quality"))
	}
	quality := 0.0
	if it.Quality != nil {
		quality = *it.Quality
	}
	if it.Base.Weapon != nil {
		weaponData := map[string]any{}
		if it.WeaponData == nil {
			it.WeaponData = map[int]map[string]any{}
		}
		it.WeaponData[slotNum] = weaponData
		weaponData["type"] = it.Base.Type
		if it.Base.SubType != "" {
			weaponData["subType"] = it.Base.SubType
		}
		weaponData["name"] = it.Name
		attackSpeedInc := calcLocal(&modList, "Speed", "INC", modparser.ModFlag.Attack) +
			math.Floor(quality/8*calcLocal(&modList, "AlternateQualityLocalAttackSpeedPer8Quality", "INC", 0))
		weaponData["AttackSpeedInc"] = attackSpeedInc
		weaponData["AttackRate"] = roundDec(it.Base.Weapon.AttackRateBase*(1+attackSpeedInc/100), 2)
		rangeBonus := calcLocal(&modList, "WeaponRange", "BASE", 0) +
			10*calcLocal(&modList, "WeaponRangeMetre", "BASE", 0) +
			math.Floor(quality/10*calcLocal(&modList, "AlternateQualityLocalWeaponRangePer10Quality", "BASE", 0))
		weaponData["rangeBonus"] = rangeBonus
		weaponData["range"] = it.Base.Weapon.Range + rangeBonus
		localIncEle := calcLocal(&modList, "LocalElementalDamage", "INC", 0)
		for _, dmgType := range dmgTypeList {
			min := weaponBase(it.Base.Weapon, dmgType, "Min") + calcLocal(&modList, dmgType+"Min", "BASE", 0)
			max := weaponBase(it.Base.Weapon, dmgType, "Max") + calcLocal(&modList, dmgType+"Max", "BASE", 0)
			if dmgType == "Physical" {
				physInc := calcLocal(&modList, "PhysicalDamage", "INC", 0)
				qualityScalar := quality
				if calcLocal(&modList, "AlternateQualityWeapon", "BASE", 0) > 0 {
					qualityScalar = 0
				}
				min = round(min * (1 + physInc/100) * (1 + qualityScalar/100))
				max = round(max * (1 + physInc/100) * (1 + qualityScalar/100))
			} else if dmgType != "Chaos" {
				localInc := calcLocal(&modList, "Local"+dmgType+"Damage", "INC", 0) + localIncEle
				min = round(min * (1 + localInc/100))
				max = round(max * (1 + localInc/100))
			}
			if min > 0 && max > 0 {
				weaponData[dmgType+"Min"] = min
				weaponData[dmgType+"Max"] = max
				dps := (min + max) / 2 * weaponData["AttackRate"].(float64)
				weaponData[dmgType+"DPS"] = dps
				if dmgType != "Physical" && dmgType != "Chaos" {
					ele, _ := weaponData["ElementalDPS"].(float64)
					weaponData["ElementalDPS"] = ele + dps
				}
			}
		}
		weaponData["CritChance"] = roundDec(
			(it.Base.Weapon.CritChanceBase+calcLocal(&modList, "CritChance", "BASE", 0))*
				(1+(calcLocal(&modList, "CritChance", "INC", 0)+math.Floor(quality/4*calcLocal(&modList, "AlternateQualityLocalCritChancePer4Quality", "INC", 0)))/100), 2)
		for _, v := range listValues(modList, "WeaponData") {
			kv := v.(modparser.Tag)
			weaponData[kv["key"].(string)] = kv["value"]
		}
		for _, mod := range modList {
			tag1 := modTag1(mod)
			condVar := "OffHandAttack"
			if slotNum == 1 {
				condVar = "MainHandAttack"
			}
			if ((mod.Name == "Accuracy" && mod.Flags == 0) || (mod.Name == "ImpaleChance" && mod.Flags != modparser.ModFlag.Spell) ||
				((mod.Name == "LifeOnHit" || mod.Name == "ManaOnHit") && mod.Flags == modparser.ModFlag.Attack) ||
				((mod.Name == "PhysicalDamageLifeLeech" || mod.Name == "PhysicalDamageManaLeech") && mod.Flags == modparser.ModFlag.Attack)) &&
				(mod.KeywordFlags == 0 || mod.KeywordFlags == modparser.KeywordFlag.Attack) && tag1 == nil {
				setFirstTag(mod, modparser.Tag{"type": "Condition", "var": condVar})
			} else if (mod.Name == "PoisonChance" || mod.Name == "BleedChance") && mod.Flags != modparser.ModFlag.Spell &&
				(tag1 == nil || (tag1["type"] == "Condition" && tag1["var"] == "CriticalStrike" && modTag2Absent(mod))) {
				appendTag(mod, modparser.Tag{"type": "Condition", "var": condVar})
			}
		}
		totalDPS := 0.0
		for _, dmgType := range dmgTypeList {
			if dps, ok := weaponData[dmgType+"DPS"].(float64); ok {
				totalDPS += dps
			}
		}
		weaponData["TotalDPS"] = totalDPS
	} else if it.Base.Armour != nil {
		armourData := it.ArmourData
		ab := it.Base.Armour
		armourBase := calcLocal(&modList, "Armour", "BASE", 0) + orZero(ab.ArmourBaseMin)
		armourVariance := orZero(ab.ArmourBaseMax) - orZero(ab.ArmourBaseMin)
		armourEvasionBase := calcLocal(&modList, "ArmourAndEvasion", "BASE", 0)
		evasionBase := calcLocal(&modList, "Evasion", "BASE", 0) + orZero(ab.EvasionBaseMin)
		evasionVariance := orZero(ab.EvasionBaseMax) - orZero(ab.EvasionBaseMin)
		evasionEnergyShieldBase := calcLocal(&modList, "EvasionAndEnergyShield", "BASE", 0)
		energyShieldBase := calcLocal(&modList, "EnergyShield", "BASE", 0) + orZero(ab.EnergyShieldBaseMin)
		energyShieldVariance := orZero(ab.EnergyShieldBaseMax) - orZero(ab.EnergyShieldBaseMin)
		armourEnergyShieldBase := calcLocal(&modList, "ArmourAndEnergyShield", "BASE", 0)
		wardBase := calcLocal(&modList, "Ward", "BASE", 0) + orZero(ab.WardBaseMin)
		wardVariance := orZero(ab.WardBaseMax) - orZero(ab.WardBaseMin)
		armourInc := calcLocal(&modList, "Armour", "INC", 0)
		armourEvasionInc := calcLocal(&modList, "ArmourAndEvasion", "INC", 0)
		evasionInc := calcLocal(&modList, "Evasion", "INC", 0)
		evasionEnergyShieldInc := calcLocal(&modList, "EvasionAndEnergyShield", "INC", 0)
		energyShieldInc := calcLocal(&modList, "EnergyShield", "INC", 0)
		wardInc := calcLocal(&modList, "Ward", "INC", 0)
		armourEnergyShieldInc := calcLocal(&modList, "ArmourAndEnergyShield", "INC", 0)
		defencesInc := calcLocal(&modList, "Defences", "INC", 0)
		qualityScalar := quality
		if calcLocal(&modList, "AlternateQualityArmour", "BASE", 0) > 0 {
			qualityScalar = 0
		}
		num := func(key string) (float64, bool) {
			v, ok := armourData[key].(float64)
			return v, ok
		}
		if v, ok := num("Armour"); ok && v > 0 {
			if _, has := num("ArmourBasePercentile"); !has {
				p := (v/((1+(armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100)) - armourBase) / armourVariance
				armourData["ArmourBasePercentile"] = roundDec(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		if v, ok := num("Evasion"); ok && v > 0 {
			if _, has := num("EvasionBasePercentile"); !has {
				p := (v/((1+(evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100)) - evasionBase) / evasionVariance
				armourData["EvasionBasePercentile"] = roundDec(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		if v, ok := num("EnergyShield"); ok && v > 0 {
			if _, has := num("EnergyShieldBasePercentile"); !has {
				p := (v/((1+(energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100)) - energyShieldBase) / energyShieldVariance
				armourData["EnergyShieldBasePercentile"] = roundDec(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		if v, ok := num("Ward"); ok && v > 0 {
			if _, has := num("WardBasePercentile"); !has {
				p := (v/((1+(wardInc+defencesInc)/100)*(1+qualityScalar/100)) - wardBase) / wardVariance
				armourData["WardBasePercentile"] = roundDec(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		pct := func(key string) float64 {
			if v, ok := num(key); ok {
				return v
			}
			return 1
		}
		armourData["Armour"] = round((armourBase + armourEvasionBase + armourEnergyShieldBase + armourVariance*pct("ArmourBasePercentile")) * (1 + (armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)/100) * (1 + qualityScalar/100))
		armourData["Evasion"] = round((evasionBase + armourEvasionBase + evasionEnergyShieldBase + evasionVariance*pct("EvasionBasePercentile")) * (1 + (evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100) * (1 + qualityScalar/100))
		armourData["EnergyShield"] = round((energyShieldBase + evasionEnergyShieldBase + armourEnergyShieldBase + energyShieldVariance*pct("EnergyShieldBasePercentile")) * (1 + (energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100) * (1 + qualityScalar/100))
		armourData["Ward"] = round((wardBase + wardVariance*pct("WardBasePercentile")) * (1 + (wardInc+defencesInc)/100) * (1 + qualityScalar/100))
		if _, has := num("ArmourBasePercentile"); !has {
			if v, _ := num("Armour"); v > 0 {
				armourData["ArmourBasePercentile"] = 1.0
			}
		}
		if _, has := num("EvasionBasePercentile"); !has {
			if v, _ := num("Evasion"); v > 0 {
				armourData["EvasionBasePercentile"] = 1.0
			}
		}
		if _, has := num("EnergyShieldBasePercentile"); !has {
			if v, _ := num("EnergyShield"); v > 0 {
				armourData["EnergyShieldBasePercentile"] = 1.0
			}
		}
		if _, has := num("WardBasePercentile"); !has {
			if v, _ := num("Ward"); v > 0 {
				armourData["WardBasePercentile"] = 1.0
			}
		}
		if ab.BlockChance != nil {
			armourData["BlockChance"] = math.Floor((*ab.BlockChance + calcLocal(&modList, "BlockChance", "BASE", 0)) * (1 + calcLocal(&modList, "BlockChance", "INC", 0)/100))
		}
		if ab.MovementPenalty != nil {
			modList = append(modList, newMod("MovementSpeed", "INC", -*ab.MovementPenalty, it.ModSource,
				modparser.Tag{"type": "Condition", "var": "IgnoreMovementPenalties", "neg": true}))
		}
		for _, v := range listValues(modList, "ArmourData") {
			kv := v.(modparser.Tag)
			armourData[kv["key"].(string)] = kv["value"]
		}
	} else if it.Base.Flask != nil {
		flaskData := it.FlaskData
		fb := it.Base.Flask
		durationInc := calcLocal(&modList, "Duration", "INC", 0)
		durationMore := calcLocal(&modList, "Duration", "MORE", 0)
		if fb.Life != nil || fb.Mana != nil {
			flaskData["instantPerc"] = calcLocal(&modList, "FlaskInstantRecovery", "BASE", 0)
			flaskData["instantLowLifePerc"] = calcLocal(&modList, "FlaskLowLifeInstantRecovery", "BASE", 0)
			recoveryMod := 1 + calcLocal(&modList, "FlaskRecovery", "INC", 0)/100
			rateMod := 1 + calcLocal(&modList, "FlaskRecoveryRate", "INC", 0)/100
			flaskData["duration"] = roundDec(fb.Duration*(1+durationInc/100)/rateMod*durationMore, 1)
			instantPerc := flaskData["instantPerc"].(float64)
			if fb.Life != nil {
				lifeBase := *fb.Life * (1 + quality/100) * recoveryMod
				flaskData["lifeBase"] = lifeBase
				flaskData["lifeInstant"] = lifeBase * instantPerc / 100
				flaskData["lifeGradual"] = lifeBase * (1 - instantPerc/100)
				flaskData["lifeTotal"] = flaskData["lifeInstant"].(float64) + flaskData["lifeGradual"].(float64)
				flaskData["lifeAdditional"] = calcLocal(&modList, "FlaskAdditionalLifeRecovery", "BASE", 0)
				flaskData["lifeEffectNotRemoved"] = calcLocalFlag(baseList, "LifeFlaskEffectNotRemoved", 0)
			}
			if fb.Mana != nil {
				manaBase := *fb.Mana * (1 + quality/100) * recoveryMod
				flaskData["manaBase"] = manaBase
				flaskData["manaInstant"] = manaBase * instantPerc / 100
				flaskData["manaGradual"] = manaBase * (1 - instantPerc/100)
				flaskData["manaTotal"] = flaskData["manaInstant"].(float64) + flaskData["manaGradual"].(float64)
				flaskData["manaEffectNotRemoved"] = calcLocalFlag(baseList, "ManaFlaskEffectNotRemoved", 0)
			}
		} else {
			flaskData["duration"] = roundDec(fb.Duration*(1+durationInc/100)*(1+quality/100)*durationMore, 1)
		}
		flaskData["chargesMax"] = fb.ChargesMax + calcLocal(&modList, "FlaskCharges", "BASE", 0)
		flaskData["chargesUsed"] = math.Floor(fb.ChargesUsed * (1 + calcLocal(&modList, "FlaskChargesUsed", "INC", 0)/100))
		flaskData["gainMod"] = 1 + calcLocal(&modList, "FlaskChargeRecovery", "INC", 0)/100
		flaskData["effectInc"] = calcLocal(&modList, "FlaskEffect", "INC", 0) + calcLocal(&modList, "LocalEffect", "INC", 0)
		for _, v := range listValues(modList, "FlaskData") {
			kv := v.(modparser.Tag)
			flaskData[kv["key"].(string)] = kv["value"]
		}
	} else if it.Base.Tincture != nil {
		tinctureData := it.TinctureData
		tb := it.Base.Tincture
		tinctureData["manaBurn"] = (tb.ManaBurn + 0.01) / (1 + calcLocal(&modList, "TinctureManaBurnRate", "INC", 0)/100) / (1 + calcLocal(&modList, "TinctureManaBurnRate", "MORE", 0)/100)
		cooldownInc := calcLocal(&modList, "TinctureCooldownRecovery", "INC", 0) + calcLocal(&modList, "CooldownRecovery", "INC", 0)
		tinctureData["cooldownInc"] = cooldownInc
		tinctureData["cooldown"] = tb.Cooldown / (1 + cooldownInc/100)
		tinctureData["effectInc"] = calcLocal(&modList, "TinctureEffect", "INC", 0) + calcLocal(&modList, "LocalEffect", "INC", 0)
		for _, v := range listValues(modList, "TinctureData") {
			kv := v.(modparser.Tag)
			tinctureData[kv["key"].(string)] = kv["value"]
		}
	} else if it.Type == "Jewel" {
		if strings.Contains(it.Name, "Grand Spectrum") {
			spectrumMod := newMod("Multiplier:GrandSpectrum", "BASE", 1.0, it.Name)
			modList = append(modList, spectrumMod)
			modList = append(modList, newMod("MinionModifier", "LIST", modparser.Tag{"mod": spectrumMod}, it.Name))
		}
		jewelData := it.JewelData
		for _, fn := range listValues(modList, "JewelFunc") {
			funcList, _ := jewelData["funcList"].([]any)
			jewelData["funcList"] = append(funcList, fn)
		}
		for _, v := range listValues(modList, "JewelData") {
			kv := v.(modparser.Tag)
			jewelData[kv["key"].(string)] = kv["value"]
		}
		if keystones := listValues(modList, "ImpossibleEscapeKeystones"); keystones != nil {
			m := map[string]any{}
			for _, v := range keystones {
				kv := v.(modparser.Tag)
				m[kv["key"].(string)] = kv["value"]
			}
			jewelData["impossibleEscapeKeystones"] = m
		}
		if it.ClusterJewel != nil {
			var notables []any
			for _, name := range listValues(modList, "ClusterJewelNotable") {
				notables = append(notables, name)
			}
			jewelData["clusterJewelNotables"] = notables
			var addedMods []any
			for _, line := range listValues(modList, "AddToClusterJewelNode") {
				addedMods = append(addedMods, line)
			}
			jewelData["clusterJewelAddedMods"] = addedMods
			if it.ClusterJewel.Size == "Small" && jewelData["clusterJewelSkill"] == "affliction_curse_effect" {
				jewelData["clusterJewelSkill"] = "affliction_curse_effect_small"
			} else if it.ClusterJewel.Size == "Medium" && jewelData["clusterJewelSkill"] == "affliction_curse_effect_small" {
				jewelData["clusterJewelSkill"] = "affliction_curse_effect"
			}
			if count, ok := jewelData["clusterJewelNodeCount"].(float64); ok {
				jewelData["clusterJewelNodeCount"] = fmin(fmax(count, it.ClusterJewel.MinNodes), it.ClusterJewel.MaxNodes)
			}
			if skill, ok := jewelData["clusterJewelSkill"].(string); ok {
				if _, valid := it.ClusterJewel.Skills[skill]; !valid {
					delete(jewelData, "clusterJewelSkill")
				}
			}
			if it.ClusterJewelSkill == "" {
				it.ClusterJewelSkill, _ = jewelData["clusterJewelSkill"].(string)
			}
			if it.ClusterJewelNodeCount == nil {
				if count, ok := jewelData["clusterJewelNodeCount"].(float64); ok {
					c := count
					it.ClusterJewelNodeCount = &c
				}
			}
			// clusterJewelValid keeps the reference's or-chain VALUE
			// semantics (scalars() can surface it in the fixture).
			valid := jewelData["clusterJewelKeystone"]
			if !truthy(valid) {
				gate := jewelData["clusterJewelSkill"]
				if !truthy(gate) {
					gate = jewelData["clusterJewelSmallsAreNothingness"]
				}
				if truthy(gate) {
					valid = jewelData["clusterJewelNodeCount"]
				} else {
					valid = gate
				}
				if !truthy(valid) {
					if truthy(jewelData["clusterJewelSocketCountOverride"]) {
						valid = jewelData["clusterJewelNothingnessCount"]
					}
				}
			}
			if valid != nil {
				jewelData["clusterJewelValid"] = valid
			} else {
				delete(jewelData, "clusterJewelValid")
			}
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
	vs := modLine.ValueScalar
	if vs == 0 {
		vs = 1
	}
	line := applyRange(strings.ReplaceAll(modLine.Line, "\n", " "), *modLine.Range, vs, corruptedOr1(modLine))
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
		it.WeaponData = map[int]map[string]any{}
	} else if it.Base.Armour != nil {
		if it.ArmourData == nil {
			it.ArmourData = map[string]any{}
		}
	} else if it.Base.Flask != nil {
		it.FlaskData = map[string]any{}
		it.BuffModList = nil
		it.BuffModListInit = true
	} else if it.Base.Tincture != nil {
		it.TinctureData = map[string]any{}
		it.BuffModList = nil
		it.BuffModListInit = true
	} else if it.Type == "Jewel" {
		it.JewelData = map[string]any{}
	}
	it.RangeLineList = nil
	id := it.ID
	if id == 0 {
		id = -1
	}
	it.ModSource = "Item:" + luaNumStr(float64(id)) + ":" + it.Name
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
		baseList = append(baseList, newMod("ArmourData", "LIST", modparser.Tag{"key": "EnergyShield", "value": 0.0}))
		it.Requirements["int"] = 0
	}
	if calcLocalFlag(&baseList, "NoAttributeRequirements", 0) {
		it.Requirements["strMod"] = 0
		it.Requirements["dexMod"] = 0
		it.Requirements["intMod"] = 0
	} else {
		it.Requirements["strMod"] = math.Floor((it.Requirements["str"] + calcLocal(&baseList, "StrRequirement", "BASE", 0)) * (1 + calcLocal(&baseList, "StrRequirement", "INC", 0)/100))
		it.Requirements["dexMod"] = math.Floor((it.Requirements["dex"] + calcLocal(&baseList, "DexRequirement", "BASE", 0)) * (1 + calcLocal(&baseList, "DexRequirement", "INC", 0)/100))
		it.Requirements["intMod"] = math.Floor((it.Requirements["int"] + calcLocal(&baseList, "IntRequirement", "BASE", 0)) * (1 + calcLocal(&baseList, "IntRequirement", "INC", 0)/100))
	}
	it.GrantedSkills = []map[string]any{}
	for _, v := range listValues(baseList, "ExtraSkill") {
		kv := v.(modparser.Tag)
		if kv["name"] == "Unknown" {
			continue
		}
		skill := map[string]any{
			"skillId": kv["skillId"],
			"source":  it.ModSource,
		}
		if lvl, ok := kv["level"]; ok {
			skill["level"] = lvl
		}
		if ns, ok := kv["noSupports"]; ok {
			skill["noSupports"] = ns
		}
		if tr, ok := kv["triggered"]; ok {
			skill["triggered"] = tr
		}
		if tc, ok := kv["triggerChance"]; ok {
			skill["triggerChance"] = tc
		}
		it.GrantedSkills = append(it.GrantedSkills, skill)
	}
	socketCount := calcLocal(&baseList, "SocketCount", "BASE", 0)
	it.AbyssalSocketCount = calcLocal(&baseList, "AbyssalSocketCount", "BASE", 0)
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
	it.SocketedJewelEffectModifier = 1 + calcLocal(&baseList, "SocketedJewelEffect", "INC", 0)/100
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
