// CalcPerform.lua L2141-3718: buffs/curses/links, curse slots, guards,
// buff application, ailments, exposures, and the ally-life gate. The party
// tab is always empty for ladder replays, so ally buff/curse/warcry/link
// sections reduce to no-ops.
package calc

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

type curseEntry struct {
	name                   string
	fromPlayer             bool
	priority               float64
	isMark                 bool
	ignoreHexLimit         bool
	socketedCursesHexLimit bool
	modList                *modstore.List
	buffModList            *modstore.List
	minionBuffModList      *modstore.List
}

func noSpace(s string) string { return strings.ReplaceAll(s, " ", "") }

var totemAuraModRe = regexp.MustCompile(`Resist?M?a?x?$`)

// totemScaleValue mirrors the Lua totem-aura scaling: integers get
// modf(round(v*scale, 2)) (the integer part), non-integers scale raw.
func totemScaleValue(v, scale float64) float64 {
	if math.Floor(v) == v {
		r := roundDec(v*scale, 2)
		return math.Trunc(r)
	}
	return v * scale
}

func scaleTotemMod(mod *modparser.Mod, scale float64) *modparser.Mod {
	totemMod := modparser.CopyMod(mod)
	totemMod.Name = "Totem" + totemMod.Name
	if scale != 1 {
		switch v := totemMod.Value.(type) {
		case float64:
			totemMod.Value = totemScaleValue(v, scale)
		case modparser.Tag:
			if inner, ok := v["mod"].(*modparser.Mod); ok {
				innerCopy := modparser.CopyMod(inner)
				innerCopy.Value = totemScaleValue(anyNum(innerCopy.Value), scale)
				v["mod"] = innerCopy
			}
		}
	}
	return totemMod
}

// dedupExtraModList ports the compareModParams accumulation used for
// ExtraAuraEffect / ExtraAuraDebuffEffect / ExtraLinkEffect lists.
func dedupExtraModList(entries []any) []*modparser.Mod {
	var out []*modparser.Mod
	for _, v := range entries {
		tag, _ := v.(modparser.Tag)
		mod, _ := tag["mod"].(*modparser.Mod)
		if mod == nil {
			continue
		}
		add := true
		for _, existing := range out {
			if modparser.CompareModParams(existing, mod) {
				existing.Value = anyNum(existing.Value) + anyNum(mod.Value)
				add = false
				break
			}
		}
		if add {
			out = append(out, modparser.CopyMod(mod))
		}
	}
	return out
}

func bkv(b *Buff, k string) any     { return b.KV[k] }
func bstr(b *Buff, k string) string { return str(b.KV[k]) }

// performBuffs continues Perform at CalcPerform.lua L2141.
func (env *Env) performBuffs(hasGuaranteedBonechill bool, nonUniqueFlasksApplyToMinion bool) {
	modDB := env.ModDB
	enemyDB := env.EnemyDB
	d := env.Data
	output := env.playerPA.output

	// Combine buffs/debuffs
	buffs := map[string]*modstore.List{}
	env.Buffs = buffs
	notBuff := map[string]bool{}
	guards := map[string]*modstore.List{}
	minionBuffs := map[string]*modstore.List{}
	env.MinionBuffsOut = minionBuffs
	debuffs := map[string]*modstore.List{}
	env.Debuffs = debuffs
	var curses []*curseEntry
	var minionCurses []*curseEntry
	minionCursesLimit := 1.0
	// allyBuffs (env.partyMembers["Aura"]) is always empty for ladder
	// replays, as are the Warcry/Link/Curse party groups.

	// Spectre ally/enemy-limit mods come from build.spectreList, which
	// ladder imports never carry; hasActiveSpectreSkill alone is inert.

	// Sustainable-stage skills need cached output values
	for _, activeSkill := range env.PlayerActiveSkills {
		if !activeSkill.SkillFlags["disable"] {
			geName := activeSkill.ActiveEffect.GrantedEffect.Name
			part2 := anyNum(activeSkill.SkillPart) == 2
			if (geName == "Blight" || geName == "Blight of Contagion" || geName == "Blight of Atrophy") && part2 {
				panic("perform: Blight part 2 needs getCachedOutputValue (unported stage cache)")
			}
			if geName == "Penance Brand of Dissipation" && part2 {
				panic("perform: Penance Brand of Dissipation part 2 needs getCachedOutputValue")
			}
			if (geName == "Scorching Ray" || geName == "Scorching Ray of Immolation") && part2 {
				maximum := 7.0
				activeSkill.SkillModList.AddMod(newMod("Multiplier:"+noSpace(geName)+"MaxStages", "BASE", maximum, "Base"))
				activeSkill.SkillModList.AddMod(newMod("Multiplier:"+noSpace(geName)+"StageAfterFirst", "BASE", maximum, "Base"))
			}
			if geName == "Earthquake of Amplification" && part2 {
				panic("perform: Earthquake of Amplification part 2 needs getCachedOutputValue")
			}
		}
	}

	appliedCombustion := false
	warcryList := map[string]bool{}
	for _, activeSkill := range env.PlayerActiveSkills {
		skillModList := activeSkill.SkillModList
		skillCfg := activeSkill.SkillCfg
		for _, buff := range activeSkill.BuffListTyped {
			buffName := bstr(buff, "name")
			buffType := bstr(buff, "type")
			if cond := bstr(buff, "cond"); cond != "" && !skillModList.GetCondition(cond, skillCfg) {
				// Nothing!
			} else if enemyCond := bstr(buff, "enemyCond"); enemyCond != "" && !enemyDB.GetCondition(enemyCond, nil) {
				// Also nothing :/
			} else if buffType == "GlobalDB" {
				modDB.AddList(buff.ModList)
			} else if buffType == "Buff" {
				if env.ModeBuffs && (!activeSkill.SkillFlags["totem"] || truthy(bkv(buff, "allowTotemBuff"))) {
					var buffCfg *modstore.Cfg
					var modStore modstore.Store = modDB
					if truthy(bkv(buff, "activeSkillBuff")) {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if !truthy(bkv(buff, "applyNotPlayer")) {
						activeSkill.BuffSkill = true
						modDB.Conditions["AffectedBy"+noSpace(buffName)] = true
						srcList := modstore.NewList(nil)
						inc := modStore.Sum("INC", buffCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer") + skillModList.Sum("INC", buffCfg, noSpace(buffName)+"Effect")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnSelf")
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods, buffs, buffName)
						if truthy(activeSkill.SkillData["thisIsNotABuff"]) {
							notBuff[buffName] = true
						}
					}
					if env.Minion != nil && !env.Minion.Hostile && (truthy(bkv(buff, "applyMinions")) || truthy(bkv(buff, "applyAllies")) || skillModList.Flag(nil, "BuffAppliesToAllies")) {
						activeSkill.MinionBuffSkill = true
						env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
						srcList := modstore.NewList(nil)
						inc := modStore.Sum("INC", buffCfg, "BuffEffect") + env.Minion.DB.Sum("INC", nil, "BuffEffectOnSelf")
						more := modStore.More(buffCfg, "BuffEffect") * env.Minion.DB.More(nil, "BuffEffectOnSelf")
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods, minionBuffs, buffName)
					}
				}
			} else if buffType == "Guard" {
				if env.ModeBuffs && (!activeSkill.SkillFlags["totem"] || truthy(bkv(buff, "allowTotemBuff"))) {
					var buffCfg *modstore.Cfg
					var modStore modstore.Store = modDB
					if truthy(bkv(buff, "activeSkillBuff")) {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if !truthy(bkv(buff, "applyNotPlayer")) {
						activeSkill.BuffSkill = true
						srcList := modstore.NewList(nil)
						inc := modStore.Sum("INC", buffCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnSelf")
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods, guards, buffName)
					}
				}
			} else if buffType == "Warcry" {
				if env.ModeBuffs {
					var modStore modstore.Store = skillModList
					warcryName := noSpace(strings.ReplaceAll(strings.ReplaceAll(buffName, " Cry", ""), "'s", ""))
					baseExerts := modStore.Sum("BASE", env.PlayerMainSkill.SkillCfg, warcryName+"ExertedAttacks")
					if baseExerts > 0 {
						extraExertions := modStore.Sum("BASE", nil, "ExtraExertedAttacks")
						exertMultiplier := modStore.More(nil, "ExtraExertedAttacks")
						modDB.AddMod(newMod("Num"+warcryName+"Exerts", "BASE", math.Floor((baseExerts+extraExertions)*exertMultiplier)))
						if !warcryList[buffName] {
							modDB.AddMod(newMod("Multiplier:ExertingWarcryCount", "BASE", 1.0, buffName))
							warcryList[buffName] = true
						}
					}
					if !skillModList.Flag(nil, "CannotShareWarcryBuffs") {
						warcryPower := 0.0
						if ov := modDB.Override(nil, "WarcryPower"); truthy(ov) {
							warcryPower = anyNum(ov)
						} else {
							warcryPower = math.Max(modDB.Sum("BASE", nil, "WarcryPower")*(1+modDB.Sum("INC", nil, "WarcryPower")/100), modDB.Sum("BASE", nil, "MinimumWarcryPower"))
						}
						for _, warcryBuff := range buff.ModList {
							if len(warcryBuff.Tags) > 0 {
								if tag, ok := warcryBuff.Tags[0].(modparser.Tag); ok && tag["effectType"] == "Warcry" && truthy(tag["div"]) {
									power := warcryPower
									if truthy(tag["limit"]) {
										power = math.Min(warcryPower, anyNum(tag["limit"]))
									}
									tag["warcryPowerBonus"] = math.Floor(power / anyNum(tag["div"]))
								}
							}
						}
						fullDuration := env.calcSkillDuration(modStore, skillCfg, activeSkill.SkillData, enemyDB)
						actualCooldown := 0.0
						if ov := modStore.Override(skillCfg, "CooldownRecovery"); truthy(ov) {
							actualCooldown = anyNum(ov)
						} else {
							actualCooldown = (anyNum(activeSkill.SkillData["cooldown"]) + modStore.Sum("BASE", skillCfg, "CooldownRecovery")) / Mod(modStore, skillCfg, "CooldownRecovery")
						}
						uptime := math.Min(fullDuration/actualCooldown, 1)
						if modDB.Flag(nil, "Condition:WarcryMaxHit") {
							uptime = 1
						}
						var extraWarcryModList []*modparser.Mod
						warcryBuffBonus := func(m *modparser.Mod) float64 {
							if len(m.Tags) > 0 {
								if tag, ok := m.Tags[0].(modparser.Tag); ok && truthy(tag["warcryPowerBonus"]) {
									return anyNum(tag["warcryPowerBonus"])
								}
							}
							return 1
						}
						if !modDB.Flag(nil, "CannotGainWarcryBuffs") {
							if !truthy(bkv(buff, "applyNotPlayer")) {
								activeSkill.BuffSkill = true
								modDB.Conditions["AffectedBy"+warcryName] = true
								srcList := modstore.NewList(nil)
								inc := modStore.Sum("INC", skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
								more := modStore.More(skillCfg, "BuffEffect", "BuffEffectOnSelf")
								for _, warcryBuff := range buff.ModList {
									mult := (1 + inc/100) * more * warcryBuffBonus(warcryBuff) * uptime
									srcList.ScaleAddList([]*modparser.Mod{warcryBuff}, mult, false)
								}
								mergeBuff(srcList.Mods, buffs, buffName)
							}
						}
						if env.Minion != nil {
							activeSkill.MinionBuffSkill = true
							env.Minion.DB.Conditions["AffectedBy"+warcryName] = true
							srcList := modstore.NewList(nil)
							inc := skillModList.Sum("INC", skillCfg, "BuffEffect") + env.Minion.DB.Sum("INC", skillCfg, "BuffEffectOnSelf")
							more := skillModList.More(skillCfg, "BuffEffect") * env.Minion.DB.More(skillCfg, "BuffEffectOnSelf")
							for _, warcryBuff := range buff.ModList {
								mult := (1 + inc/100) * more * warcryBuffBonus(warcryBuff) * uptime
								srcList.ScaleAddList([]*modparser.Mod{warcryBuff}, mult, false)
							}
							// Special handling for the minion side to add the flat damage bonus
							if activeSkill.ActiveEffect.GrantedEffect.Name == "Rallying Cry" {
								warcryPowerBonus := math.Floor(math.Min(warcryPower, 30) / 5)
								rallyingWeaponEffect := math.Floor(activeSkill.SkillModList.Sum("BASE", env.PlayerMainSkill.SkillCfg, "RallyingCryAllyDamageBonusPer5Power") * warcryPowerBonus)
								rallyInc := modStore.Sum("INC", skillCfg, "BuffEffect") + env.Minion.DB.Sum("INC", skillCfg, "BuffEffectOnSelf")
								rallyingBonusMoreMultiplier := 1 + activeSkill.SkillModList.Sum("BASE", env.PlayerMainSkill.SkillCfg, "RallyingCryMinionDamageBonusMultiplier")
								for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
									for _, suffix := range []string{"Min", "Max"} {
										if truthy(env.Player.WeaponData1[damageType+suffix]) {
											extraWarcryModList = append(extraWarcryModList, newMod(damageType+suffix, "BASE",
												anyNum(env.Player.WeaponData1[damageType+suffix])*rallyingWeaponEffect/100,
												"Rallying Cry", int64(0), modparser.KeywordFlag.Attack,
												modparser.Tag{"type": "GlobalEffect", "effectType": "Warcry", "div": 5.0, "limit": 30.0}))
										}
									}
								}
								srcList.ScaleAddList(extraWarcryModList, (1+rallyInc/100)*rallyingBonusMoreMultiplier*uptime, false)
							}
							mergeBuff(srcList.Mods, minionBuffs, buffName)
						}
					}
				}
			} else if buffType == "Aura" {
				if env.ModeBuffs {
					extraAuraModList := dedupExtraModList(modDB.List(skillCfg, "ExtraAuraEffect"))
					if !truthy(activeSkill.SkillData["auraCannotAffectSelf"]) {
						inc := skillModList.Sum("INC", skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						mult := (1 + inc/100) * more
						// allyBuffs is empty: the ally-effect comparison passes
						activeSkill.BuffSkill = true
						modDB.Conditions["AffectedByAura"] = true
						if strings.HasPrefix(buffName, "Vaal") && len(buffName) > 5 {
							modDB.Conditions["AffectedBy"+noSpace(buffName[5:])] = true
						}
						modDB.Conditions["AffectedBy"+noSpace(buffName)] = true
						srcList := modstore.NewList(nil)
						srcList.ScaleAddList(buff.ModList, mult, false)
						srcList.ScaleAddList(extraAuraModList, mult, false)
						mergeBuff(srcList.Mods, buffs, buffName)
					}
					if !(modDB.Flag(nil, "SelfAurasCannotAffectAllies") || modDB.Flag(nil, "SelfAurasOnlyAffectYou") || modDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						if env.Minion != nil && (!env.Minion.Hostile || modDB.Flag(nil, "AurasAffectEnemies")) {
							inc := skillModList.Sum("INC", skillCfg, "AuraEffect", "BuffEffect") + env.Minion.DB.Sum("INC", skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect") * env.Minion.DB.More(skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
							mult := (1 + inc/100) * more
							activeSkill.MinionBuffSkill = true
							env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
							env.Minion.DB.Conditions["AffectedByAura"] = true
							srcList := modstore.NewList(nil)
							srcList.ScaleAddList(buff.ModList, mult, false)
							srcList.ScaleAddList(extraAuraModList, mult, false)
							mergeBuff(srcList.Mods, minionBuffs, buffName)
						}
						if modDB.Flag(nil, "AurasAffectEnemies") && !skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget") {
							inc := skillModList.Sum("INC", skillCfg, "AuraEffect", "BuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect")
							mult := (1 + inc/100) * more
							srcList := modstore.NewList(nil)
							srcList.ScaleAddList(buff.ModList, mult, false)
							srcList.ScaleAddList(extraAuraModList, mult, false)
							mergeBuff(srcList.Mods, debuffs, buffName)
						}
					}
					if env.PlayerMainSkill.SkillFlags["totem"] && !(modDB.Flag(nil, "SelfAurasCannotAffectAllies") || modDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						activeSkill.TotemBuffSkill = true
						env.PlayerMainSkill.SkillModList.Conditions["AffectedBy"+noSpace(buffName)] = true
						env.PlayerMainSkill.SkillModList.Conditions["AffectedByAura"] = true
						srcList := modstore.NewList(nil)
						inc := skillModList.Sum("INC", skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
						scale := math.Max((1+inc/100)*more, 0)
						for _, modList := range [][]*modparser.Mod{extraAuraModList, buff.ModList} {
							for _, mod := range modList {
								if mod.Name == "EnergyShield" || mod.Name == "Armour" || mod.Name == "Evasion" || totemAuraModRe.MatchString(mod.Name) {
									srcList.AddMod(scaleTotemMod(mod, scale))
								}
							}
						}
						mergeBuff(srcList.Mods, buffs, "Totem "+buffName)
					}
					// aura with an added debuff but no AuraDebuff entry
					if env.ModeEffective && len(modDB.List(skillCfg, "ExtraAuraDebuffEffect")) > 0 && !(modDB.Flag(nil, "SelfAurasOnlyAffectYou") || skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget")) {
						auraDebuffFound := false
						for _, other := range activeSkill.BuffListTyped {
							if bstr(other, "type") == "AuraDebuff" {
								auraDebuffFound = true
								break
							}
						}
						if !auraDebuffFound {
							activeSkill.DebuffSkill = true
							extraDebuffModList := dedupExtraModList(modDB.List(skillCfg, "ExtraAuraDebuffEffect"))
							// #EVAL the reference merges the UNSCALED list here
							mergeBuff(extraDebuffModList, debuffs, buffName)
						}
					}
				}
			} else if buffType == "Debuff" || buffType == "AuraDebuff" {
				var stackCount float64
				if stackVar := bstr(buff, "stackVar"); stackVar != "" {
					stackCount = skillModList.Sum("BASE", skillCfg, "Multiplier:"+stackVar)
					if truthy(bkv(buff, "stackLimit")) {
						stackCount = math.Min(stackCount, anyNum(bkv(buff, "stackLimit")))
					} else if limitVar := bstr(buff, "stackLimitVar"); limitVar != "" {
						stackCount = math.Min(stackCount, skillModList.Sum("BASE", skillCfg, "Multiplier:"+limitVar))
					}
				} else if truthy(activeSkill.SkillData["stackCount"]) {
					stackCount = anyNum(activeSkill.SkillData["stackCount"])
				} else {
					stackCount = 1
				}
				if env.ModeEffective && stackCount > 0 {
					activeSkill.DebuffSkill = true
					enemyDB.Conditions["AffectedBy"+noSpace(buffName)] = true
					modDB.Conditions["AffectedBy"+noSpace(buffName)] = true
					srcList := modstore.NewList(nil)
					mult := 1.0
					var extraAuraModList []*modparser.Mod
					if buffType == "AuraDebuff" {
						extraAuraModList = dedupExtraModList(modDB.List(skillCfg, "ExtraAuraDebuffEffect"))
						mult = 0
						if !modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
							inc := skillModList.Sum("INC", skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							mult = (1 + inc/100) * more
						}
					}
					if buffType == "Debuff" {
						inc := skillModList.Sum("INC", skillCfg, "DebuffEffect")
						more := skillModList.More(skillCfg, "DebuffEffect")
						mult = (1 + inc/100) * more
					}
					srcList.ScaleAddList(buff.ModList, mult*stackCount, false)
					srcList.ScaleAddList(extraAuraModList, mult*stackCount, false)
					if truthy(activeSkill.SkillData["stackCount"]) || bstr(buff, "stackVar") != "" {
						srcList.AddMod(newMod("Multiplier:"+buffName+"Stack", "BASE", stackCount, buffName))
					}
					mergeBuff(srcList.Mods, debuffs, buffName)
				}
			} else if buffType == "Curse" || buffType == "CurseBuff" {
				mark := activeSkill.SkillTypes[modparser.SkillType.Mark]
				modDB.Conditions["SelfCast"+noSpace(buffName)] = !(activeSkill.SkillTypes[modparser.SkillType.Triggered] || activeSkill.SkillTypes[modparser.SkillType.Aura])
				skipCurse := false
				if truthy(env.ConfigInput["balanceOfTerrorSelfCast"+noSpace(buffName)]) && !mark {
					skipCurse = true
				}
				if !skipCurse && (env.ModeEffective && (!enemyDB.Flag(nil, "Hexproof") || modDB.Flag(nil, "CursesIgnoreHexproof") || truthy(activeSkill.SkillData["ignoreHexLimit"]) || truthy(activeSkill.SkillData["ignoreHexproof"])) || mark) {
					curse := &curseEntry{
						name:                   buffName,
						fromPlayer:             true,
						priority:               env.determineCursePriority(buffName, activeSkill),
						isMark:                 mark,
						ignoreHexLimit:         (modDB.Flag(activeSkill.SkillCfg, "CursesIgnoreHexLimit") || truthy(activeSkill.SkillData["ignoreHexLimit"])) && !mark,
						socketedCursesHexLimit: modDB.Flag(activeSkill.SkillCfg, "SocketedCursesAdditionalLimit"),
					}
					inc := skillModList.Sum("INC", skillCfg, "CurseEffect") + enemyDB.Sum("INC", nil, "CurseEffectOnSelf")
					if activeSkill.SkillTypes[modparser.SkillType.Aura] && !activeSkill.SkillTypes[modparser.SkillType.RemoteMined] {
						inc += skillModList.Sum("INC", skillCfg, "AuraEffect")
					}
					more := skillModList.More(skillCfg, "CurseEffect")
					if !curse.isMark {
						more *= enemyDB.More(nil, "CurseEffectOnSelf")
					}
					mult := 0.0
					if !((modDB.Flag(nil, "SelfAurasOnlyAffectYou") || skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget")) && activeSkill.SkillTypes[modparser.SkillType.Aura]) {
						mult = (1 + inc/100) * more
					}
					if buffType == "Curse" {
						curse.modList = modstore.NewList(nil)
						curse.modList.ScaleAddList(buff.ModList, mult, false)
					} else {
						// Curse applies a buff; scale by curse effect, then buff effect
						temp := modstore.NewList(nil)
						temp.ScaleAddList(buff.ModList, mult, false)
						curse.buffModList = modstore.NewList(nil)
						buffInc := modDB.Sum("INC", skillCfg, "BuffEffectOnSelf")
						buffMore := modDB.More(skillCfg, "BuffEffectOnSelf")
						curse.buffModList.ScaleAddList(temp.Mods, (1+buffInc/100)*buffMore, false)
						if env.Minion != nil {
							curse.minionBuffModList = modstore.NewList(nil)
							minionInc := env.Minion.DB.Sum("INC", nil, "BuffEffectOnSelf")
							minionMore := env.Minion.DB.More(nil, "BuffEffectOnSelf")
							curse.minionBuffModList.ScaleAddList(temp.Mods, (1+minionInc/100)*minionMore, false)
						}
					}
					curses = append(curses, curse)
				}
			} else if buffType == "Link" {
				linksApplyToMinions := env.Minion != nil && modDB.Flag(nil, "Condition:CanLinkToMinions") && modDB.Flag(nil, "Condition:LinkedToMinion") &&
					!env.Minion.DB.Flag(nil, "Condition:CannotBeDamaged") &&
					!(env.Minion.MainSkill.SummonSkill != nil && env.Minion.MainSkill.SummonSkill.SkillTypes[modparser.SkillType.MinionsAreUndamagable])
				var linkApplied bool
				if env.ModeBuffs && !linkApplied && linksApplyToMinions {
					var extraLinkModList []*modparser.Mod
					for _, v := range modDB.List(skillCfg, "ExtraLinkEffect") {
						tag, _ := v.(modparser.Tag)
						mod, _ := tag["mod"].(*modparser.Mod)
						if mod == nil {
							continue
						}
						add := true
						for _, existing := range extraLinkModList {
							if modparser.CompareModParams(existing, mod) {
								existing.Value = anyNum(existing.Value) + anyNum(mod.Value)
								add = false
								break
							}
						}
						if add {
							extraLinkModList = append(extraLinkModList, modparser.CopyMod(mod))
							if mod.Name == "ParentNonUniqueFlasksAppliedToYou" {
								nonUniqueFlasksApplyToMinion = true
							}
						}
					}
					inc := skillModList.Sum("INC", skillCfg, "LinkEffect", "BuffEffect")
					more := skillModList.More(skillCfg, "LinkEffect", "BuffEffect")
					activeSkill.MinionBuffSkill = true
					env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
					env.Minion.DB.Conditions["AffectedByLink"] = true
					srcList := modstore.NewList(nil)
					inc += env.Minion.DB.Sum("INC", nil, "BuffEffectOnSelf", "LinkEffectOnSelf")
					more *= env.Minion.DB.More(nil, "BuffEffectOnSelf", "LinkEffectOnSelf")
					mult := (1 + inc/100) * more
					srcList.ScaleAddList(buff.ModList, mult, false)
					srcList.ScaleAddList(extraLinkModList, mult, false)
					mergeBuff(srcList.Mods, minionBuffs, buffName)
					linkApplied = true
				}
			}
		}
		if skillModList.Flag(nil, "Condition:CanWither") || (activeSkill.Minion != nil && env.Minion != nil && env.Minion.DB.Flag(nil, "Condition:CanWither")) {
			var effect float64
			if activeSkill.Minion != nil {
				effect = math.Floor(6 * (1 + modDB.Sum("INC", nil, "MinionWitherEffect")/100))
			} else {
				effect = math.Floor(6 * (1 + modDB.Sum("INC", nil, "WitherEffect")/100))
			}
			modDB.AddMod(newMod("WitherEffectStack", "MAX", effect))
		}
		// Handle combustion
		if enemyDB.Flag(nil, "Condition:Ignited") && (activeSkill.SkillTypes[modparser.SkillType.Damage] || activeSkill.SkillTypes[modparser.SkillType.Attack]) && !appliedCombustion {
			for _, support := range activeSkill.SupportList {
				if support.GrantedEffect.Name == "Combustion" {
					if !skillModList.Flag(activeSkill.SkillCfg, "CannotIgnite") {
						value := skillModList.Sum("BASE", activeSkill.SkillCfg, "CombustionFireResist")
						enemyDB.AddMod(newMod("FireResist", "BASE", value, "Combustion",
							modparser.Tag{"type": "GlobalEffect", "effectType": "Debuff", "effectName": "Combustion"},
							modparser.Tag{"type": "Condition", "var": "Ignited"}))
						appliedCombustion = true
					}
					break
				}
			}
		}
		if activeSkill.Minion != nil && activeSkill.Minion.ActiveSkillList != nil {
			castingMinion := activeSkill.Minion
			for _, activeMinionSkill := range castingMinion.ActiveSkillList {
				setSpectreSource := func(modList []*modparser.Mod, sourceSkill string) {
					if activeSkill.SkillFlags["spectre"] {
						source := "Spectre:"
						if sourceSkill != "" {
							source = source + sourceSkill + " - " + castingMinion.MinionData.Name
						} else {
							source = source + castingMinion.MinionData.Name
						}
						for _, m := range modList {
							m.Source = source
							m.SourceSet = true
						}
					}
				}
				minionSkillModList := activeMinionSkill.SkillModList
				minionSkillCfg := activeMinionSkill.SkillCfg
				for _, buff := range activeMinionSkill.BuffListTyped {
					buffName := bstr(buff, "name")
					buffType := bstr(buff, "type")
					if buffType == "Buff" {
						if env.ModeBuffs && truthy(activeMinionSkill.SkillData["enable"]) {
							var buffCfg *modstore.Cfg
							var modStore modstore.Store = castingMinion.DB
							if truthy(bkv(buff, "activeSkillBuff")) {
								buffCfg = minionSkillCfg
								modStore = minionSkillModList
							}
							if truthy(bkv(buff, "applyAllies")) {
								activeMinionSkill.BuffSkill = true
								modDB.Conditions["AffectedBy"+noSpace(buffName)] = true
								srcList := modstore.NewList(nil)
								inc := modStore.Sum("INC", buffCfg, "BuffEffect", "BuffEffectOnPlayer") + modDB.Sum("INC", nil, "BuffEffectOnSelf")
								more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnPlayer") * modDB.More(nil, "BuffEffectOnSelf")
								srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
								mergeBuff(srcList.Mods, buffs, buffName)
								mergeBuff(buff.ModList, buffs, buffName)
								if truthy(activeMinionSkill.SkillData["thisIsNotABuff"]) {
									notBuff[buffName] = true
								}
							}
							envMinionCheck := env.Minion != nil && (env.Minion == castingMinion || truthy(bkv(buff, "applyAllies")))
							if truthy(bkv(buff, "applyMinions")) || envMinionCheck {
								activeMinionSkill.MinionBuffSkill = true
								if envMinionCheck {
									env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
								} else {
									castingMinion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
								}
								srcList := modstore.NewList(nil)
								names := []string{"BuffEffect"}
								if env.Minion == castingMinion {
									names = append(names, "BuffEffectOnSelf")
								}
								inc := modStore.Sum("INC", buffCfg, names...)
								more := modStore.More(buffCfg, names...)
								srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
								mergeBuff(srcList.Mods, minionBuffs, buffName)
								mergeBuff(buff.ModList, minionBuffs, buffName)
								if truthy(activeMinionSkill.SkillData["thisIsNotABuff"]) {
									notBuff[buffName] = true
								}
							}
						}
					} else if buffType == "Aura" {
						if env.ModeBuffs && truthy(activeMinionSkill.SkillData["enable"]) {
							extraAuraModList := dedupExtraModList(castingMinion.DB.List(minionSkillCfg, "ExtraAuraEffect"))
							if !(castingMinion.DB.Flag(nil, "SelfAurasCannotAffectAllies") || castingMinion.DB.Flag(nil, "SelfAurasOnlyAffectYou") || castingMinion.DB.Flag(nil, "SelfAuraSkillsCannotAffectAllies") || minionSkillModList.Flag(minionSkillCfg, "SelfAurasAffectYouAndLinkedTarget")) {
								if !modDB.Flag(nil, "AlliesAurasCannotAffectSelf") && !truthy(modDB.Conditions["AffectedBy"+noSpace(buffName)]) {
									inc := minionSkillModList.Sum("INC", minionSkillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnPlayer", "AuraBuffEffect") + modDB.Sum("INC", minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect") * modDB.More(minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									mult := (1 + inc/100) * more
									activeMinionSkill.BuffSkill = true
									modDB.Conditions["AffectedByAura"] = true
									if strings.HasPrefix(buffName, "Vaal") && len(buffName) > 5 {
										modDB.Conditions["AffectedBy"+noSpace(buffName[5:])] = true
									}
									modDB.Conditions["AffectedBy"+noSpace(buffName)] = true
									srcList := modstore.NewList(nil)
									srcList.ScaleAddList(buff.ModList, mult, false)
									srcList.ScaleAddList(extraAuraModList, mult, false)
									setSpectreSource(srcList.Mods, buffName)
									mergeBuff(srcList.Mods, buffs, buffName)
								}
								if env.Minion != nil && !truthy(env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)]) && (env.Minion != castingMinion || !truthy(activeSkill.SkillData["auraCannotAffectSelf"])) {
									inc := minionSkillModList.Sum("INC", minionSkillCfg, "AuraEffect", "BuffEffect") + env.Minion.DB.Sum("INC", minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect") * env.Minion.DB.More(minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									mult := (1 + inc/100) * more
									activeMinionSkill.MinionBuffSkill = true
									env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
									env.Minion.DB.Conditions["AffectedByAura"] = true
									srcList := modstore.NewList(nil)
									srcList.ScaleAddList(buff.ModList, mult, false)
									srcList.ScaleAddList(extraAuraModList, mult, false)
									setSpectreSource(srcList.Mods, buffName)
									mergeBuff(srcList.Mods, minionBuffs, buffName)
								}
								// export list mutation: setSpectreSource runs over the
								// SHARED buff/extra mods (AddList aliases them)
								newModList := append(append([]*modparser.Mod{}, buff.ModList...), extraAuraModList...)
								setSpectreSource(newModList, buffName)
								if env.PlayerMainSkill.SkillFlags["totem"] && !truthy(env.PlayerMainSkill.SkillModList.Conditions["AffectedBy"+noSpace(buffName)]) {
									activeMinionSkill.TotemBuffSkill = true
									env.PlayerMainSkill.SkillModList.Conditions["AffectedBy"+noSpace(buffName)] = true
									env.PlayerMainSkill.SkillModList.Conditions["AffectedByAura"] = true
									srcList := modstore.NewList(nil)
									inc := minionSkillModList.Sum("INC", minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
									scale := math.Max((1+inc/100)*more, 0)
									for _, modList := range [][]*modparser.Mod{extraAuraModList, buff.ModList} {
										for _, mod := range modList {
											if mod.Name == "EnergyShield" || mod.Name == "Armour" || mod.Name == "Evasion" || totemAuraModRe.MatchString(mod.Name) {
												srcList.AddMod(scaleTotemMod(mod, scale))
											}
										}
									}
									setSpectreSource(srcList.Mods, "")
									mergeBuff(srcList.Mods, buffs, "Totem "+buffName)
								}
							}
						}
					} else if buffType == "Curse" {
						if env.ModeEffective && truthy(activeMinionSkill.SkillData["enable"]) && (!enemyDB.Flag(nil, "Hexproof") || activeMinionSkill.SkillTypes[modparser.SkillType.Mark]) {
							curse := &curseEntry{
								name:     buffName,
								priority: env.determineCursePriority(buffName, activeMinionSkill),
							}
							inc := minionSkillModList.Sum("INC", minionSkillCfg, "CurseEffect") + enemyDB.Sum("INC", nil, "CurseEffectOnSelf")
							more := minionSkillModList.More(minionSkillCfg, "CurseEffect") * enemyDB.More(nil, "CurseEffectOnSelf")
							curse.modList = modstore.NewList(nil)
							curse.modList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
							minionCurses = append(minionCurses, curse)
						}
					} else if buffType == "Debuff" || buffType == "AuraDebuff" {
						var stackCount float64
						if stackVar := bstr(buff, "stackVar"); stackVar != "" {
							stackCount = minionSkillModList.Sum("BASE", minionSkillCfg, "Multiplier:"+stackVar)
							if truthy(bkv(buff, "stackLimit")) {
								stackCount = math.Min(stackCount, anyNum(bkv(buff, "stackLimit")))
							} else if limitVar := bstr(buff, "stackLimitVar"); limitVar != "" {
								stackCount = math.Min(stackCount, minionSkillModList.Sum("BASE", minionSkillCfg, "Multiplier:"+limitVar))
							}
						} else if truthy(activeMinionSkill.SkillData["stackCount"]) {
							stackCount = anyNum(activeMinionSkill.SkillData["stackCount"])
						} else {
							stackCount = 1
						}
						if env.ModeEffective && stackCount > 0 {
							activeMinionSkill.DebuffSkill = true
							srcList := modstore.NewList(nil)
							mult := 1.0
							if buffType == "AuraDebuff" {
								mult = 0
								if !minionSkillModList.Flag(nil, "SelfAurasOnlyAffectYou") || minionSkillModList.Flag(minionSkillCfg, "SelfAurasAffectYouAndLinkedTarget") {
									inc := minionSkillModList.Sum("INC", minionSkillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
									mult = (1 + inc/100) * more
									if truthy(enemyDB.Conditions["AffectedBy"+noSpace(buffName)]) {
										mult = 0
									}
								}
							}
							enemyDB.Conditions["AffectedBy"+noSpace(buffName)] = true
							if env.Minion != nil && env.Minion == castingMinion {
								env.Minion.DB.Conditions["AffectedBy"+noSpace(buffName)] = true
							}
							if buffType == "Debuff" {
								inc := minionSkillModList.Sum("INC", minionSkillCfg, "DebuffEffect")
								more := minionSkillModList.More(minionSkillCfg, "DebuffEffect")
								mult = (1 + inc/100) * more
							}
							srcList.ScaleAddList(buff.ModList, mult*stackCount, false)
							if truthy(activeMinionSkill.SkillData["stackCount"]) || bstr(buff, "stackVar") != "" {
								srcList.AddMod(newMod("Multiplier:"+buffName+"Stack", "BASE", activeMinionSkill.SkillData["stackCount"], buffName))
							}
							mergeBuff(srcList.Mods, debuffs, buffName)
						}
					}
				}
			}
		}
	}
	// ally otherEffects / Aura / AuraDebuff / party Warcry / party Link:
	// party tab is empty for ladder replays

	if env.ModeCombat {
		// 2 steps to account for effects affecting life recovery from flasks
		env.mergeFlasks(env.Flasks, true, nonUniqueFlasksApplyToMinion, nonUniqueFlasksApplyToMinion)
	}

	// Process stats from alternate charges
	if env.ModeCombat {
		if modDB.Flag(nil, "UseEnduranceCharges") && modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
			tripleDmgChancePerEndurance := modDB.Sum("BASE", nil, "PerBrutalTripleDamageChance")
			modDB.AddMod(newMod("TripleDamageChance", "BASE", tripleDmgChancePerEndurance, modparser.Tag{"type": "Multiplier", "var": "BrutalCharge"}))
		}
		if modDB.Flag(nil, "UseFrenzyCharges") && modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
			dmgPerAffliction := modDB.Sum("BASE", nil, "PerAfflictionAilmentDamage")
			effectPerAffliction := modDB.Sum("BASE", nil, "PerAfflictionNonDamageEffect")
			modDB.AddMod(newMod("Damage", "MORE", dmgPerAffliction, "Affliction Charges", int64(0), modparser.KeywordFlag.Ailment, modparser.Tag{"type": "Multiplier", "var": "AfflictionCharge"}))
			for _, name := range []string{"EnemyChillEffect", "EnemyShockEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"} {
				modDB.AddMod(newMod(name, "MORE", effectPerAffliction, "Affliction Charges", modparser.Tag{"type": "Multiplier", "var": "AfflictionCharge"}))
			}
		}
	}

	// Check for extra curses
	type extraCurseDest struct {
		dest *[]*curseEntry
		db   *modstore.DB
	}
	extraCurseDests := []extraCurseDest{{&curses, modDB}}
	if env.Minion != nil {
		extraCurseDests = append(extraCurseDests, extraCurseDest{&minionCurses, env.Minion.DB})
	}
	for _, ecd := range extraCurseDests {
		curseDB := ecd.db
		for _, v := range curseDB.List(nil, "ExtraCurse") {
			tag, _ := v.(modparser.Tag)
			grantedEffect := d.Skills[str(tag["skillId"])]
			if grantedEffect == nil {
				continue
			}
			gemModList := modstore.NewList(nil)
			env.mergeSkillInstanceMods(gemModList, &ActiveEffect{
				GrantedEffect: grantedEffect,
				Level:         anyNum(tag["level"]),
				Quality:       0,
			}, nil)
			var curseModList []*modparser.Mod
			for _, mod := range gemModList.Mods {
				for _, tv := range mod.Tags {
					if mt, ok := tv.(modparser.Tag); ok && mt["type"] == "GlobalEffect" && mt["effectType"] == "Curse" {
						curseModList = append(curseModList, mod)
						break
					}
				}
			}
			if truthy(tag["applyToPlayer"]) {
				if curseDB.Sum("BASE", nil, "AvoidCurse") < 100 {
					curseDB.Conditions["Cursed"] = true
					curseDB.Multipliers["CurseOnSelf"] = curseDB.Multipliers["CurseOnSelf"] + 1
					curseDB.Conditions["AffectedBy"+noSpace(grantedEffect.Name)] = true
					cfg := &modstore.Cfg{SkillName: grantedEffect.Name}
					inc := curseDB.Sum("INC", cfg, "CurseEffectOnSelf") + gemModList.Sum("INC", nil, "CurseEffectAgainstPlayer")
					more := curseDB.More(cfg, "CurseEffectOnSelf") * gemModList.More(nil, "CurseEffectAgainstPlayer")
					curseDB.ScaleAddList(curseModList, math.Max((1+inc/100)*more, 0), false)
				}
			} else if !enemyDB.Flag(nil, "Hexproof") || curseDB.Flag(nil, "CursesIgnoreHexproof") {
				curse := &curseEntry{
					name:       grantedEffect.Name,
					fromPlayer: ecd.dest == &curses,
					priority:   env.determineCursePriority(grantedEffect.Name, nil),
				}
				curse.modList = modstore.NewList(nil)
				curse.modList.ScaleAddList(curseModList, (1+enemyDB.Sum("INC", nil, "CurseEffectOnSelf")/100)*enemyDB.More(nil, "CurseEffectOnSelf"), false)
				*ecd.dest = append(*ecd.dest, curse)
			}
		}
	}
	// ally curses: party tab empty

	// Set curse limit
	if modDB.Flag(nil, "CurseLimitIsMaximumPowerCharges") {
		output["EnemyCurseLimit"] = outNum(output, "PowerChargesMax")
	} else {
		output["EnemyCurseLimit"] = modDB.Sum("BASE", nil, "EnemyCurseLimit")
	}
	cursesLimit := outNum(output, "EnemyCurseLimit")

	// Assign curses to slots
	var curseSlots []*curseEntry
	markSlotted := false
	type curseSource struct {
		entries []*curseEntry
		limit   float64
	}
	for _, source := range []curseSource{{curses, cursesLimit}, {minionCurses, minionCursesLimit}} {
		for _, curse := range source.entries {
			// Calculate curses that ignore hex limit after
			if !curse.ignoreHexLimit && !curse.socketedCursesHexLimit {
				slot := 0
				skipAddingCurse := false
				// Check if we need to disable a certain curse aura.
				for _, activeSkill := range env.PlayerActiveSkills {
					if len(activeSkill.BuffListTyped) > 0 && curse.name == bstr(activeSkill.BuffListTyped[0], "name") && activeSkill.SkillTypes[modparser.SkillType.Aura] {
						if modDB.Flag(nil, "SelfAurasOnlyAffectYou") || activeSkill.SkillModList.Flag(env.PlayerMainSkill.SkillCfg, "SelfAurasAffectYouAndLinkedTarget") {
							skipAddingCurse = true
						}
						break
					}
				}
				for i := 1; i <= int(source.limit); i++ {
					if curse.isMark && markSlotted {
						slot = 0
						break
					}
					var slotted *curseEntry
					if i <= len(curseSlots) {
						slotted = curseSlots[i-1]
					}
					if slotted == nil {
						slot = i
						break
					} else if slotted.name == curse.name {
						if slotted.priority < curse.priority {
							slot = i
						} else {
							slot = 0
						}
						break
					} else if slotted.priority < curse.priority {
						slot = i
					}
				}
				if slot != 0 {
					if slot <= len(curseSlots) && curseSlots[slot-1] != nil && curseSlots[slot-1].isMark {
						markSlotted = false
					}
					if !skipAddingCurse {
						for len(curseSlots) < slot {
							curseSlots = append(curseSlots, nil)
						}
						curseSlots[slot-1] = curse
					}
					if curse.isMark {
						markSlotted = true
					}
				}
			}
		}
	}

	for _, source := range [][]*curseEntry{curses, minionCurses} {
		for _, curse := range source {
			if curse.ignoreHexLimit {
				skipAddingCurse := false
				for i := range curseSlots {
					if curseSlots[i] != nil && curseSlots[i].name == curse.name {
						if curseSlots[i].priority < curse.priority {
							curseSlots[i] = curse
						}
						skipAddingCurse = true
						break
					}
				}
				if !skipAddingCurse {
					curseSlots = append(curseSlots, curse)
				}
			}
			if curse.socketedCursesHexLimit {
				socketedCursesHexLimitValue := modDB.Sum("BASE", nil, "SocketedCursesHexLimitValue")
				skipAddingCurse := false
				for i := range curseSlots {
					if curseSlots[i] != nil && curseSlots[i].name == curse.name {
						if curseSlots[i].priority < curse.priority {
							curseSlots[i] = curse
						}
						skipAddingCurse = true
						break
					}
					if float64(i+1) >= socketedCursesHexLimitValue {
						skipAddingCurse = true
					}
				}
				if !skipAddingCurse {
					curseSlots = append(curseSlots, curse)
				}
			}
		}
	}
	env.CurseSlots = env.CurseSlots[:0]
	for _, s := range curseSlots {
		env.CurseSlots = append(env.CurseSlots, s)
	}

	// Process guard buffs
	type guardSlot struct {
		name    string
		modList *modstore.List
	}
	var guardSlots []guardSlot
	nonVaal := false
	for _, name := range sortedListKeysOf(guards) {
		if name == "Vaal Molten Shell" {
			guardSlots = []guardSlot{{name, guards[name]}}
			nonVaal = false
			break
		} else if strings.HasPrefix(name, "Vaal") {
			guardSlots = append(guardSlots, guardSlot{name, guards[name]})
		} else if !nonVaal {
			guardSlots = append(guardSlots, guardSlot{name, guards[name]})
			nonVaal = true
		}
	}
	if nonVaal {
		modDB.Conditions["AffectedByNonVaalGuardSkill"] = true
	}
	for _, guard := range guardSlots {
		modDB.Conditions["AffectedByGuardSkill"] = true
		modDB.Conditions["AffectedBy"+noSpace(guard.name)] = true
		mergeBuff(guard.modList.Mods, buffs, guard.name)
	}
	output["GuardSkillActive"] = truthy(modDB.Conditions["AffectedByGuardSkill"])

	// Apply buff/debuff modifiers
	for _, name := range sortedListKeysOf(buffs) {
		modList := buffs[name]
		modDB.AddList(modList.Mods)
		if !notBuff[name] {
			modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + 1
		}
		if env.Minion != nil {
			addMinionModifiers(modList, env.PlayerMainSkill.SkillCfg, env.Minion)
		}
	}
	if env.Minion != nil {
		for _, name := range sortedListKeysOf(minionBuffs) {
			env.Minion.DB.AddList(minionBuffs[name].Mods)
		}
	}
	for _, name := range sortedListKeysOf(debuffs) {
		enemyDB.AddList(debuffs[name].Mods)
	}
	modDB.Multipliers["CurseOnEnemy"] = float64(len(curseSlots))
	for _, slot := range curseSlots {
		if slot == nil {
			continue
		}
		enemyDB.Conditions["Cursed"] = true
		if slot.isMark {
			enemyDB.Conditions["Marked"] = true
		}
		if slot.modList != nil {
			enemyDB.AddList(slot.modList.Mods)
		}
		if slot.buffModList != nil {
			modDB.AddList(slot.buffModList.Mods)
		}
		if slot.minionBuffModList != nil {
			env.Minion.DB.AddList(slot.minionBuffModList.Mods)
		}
	}

	// Fix the configured impale stacks on the enemy
	maxImpaleStacks := modDB.Sum("BASE", nil, "ImpaleStacksMax") * (1 + modDB.Sum("BASE", nil, "ImpaleAdditionalDurationChance")/100)
	if !enemyDB.HasMod("BASE", nil, "Multiplier:ImpaleStacks") {
		enemyDB.AddMod(newMod("Multiplier:ImpaleStacks", "BASE", maxImpaleStacks, "Config", modparser.Tag{"type": "Condition", "var": "Combat"}))
	} else if enemyDB.Sum("BASE", nil, "Multiplier:ImpaleStacks") > maxImpaleStacks {
		enemyDB.ReplaceMod(newMod("Multiplier:ImpaleStacks", "BASE", maxImpaleStacks, "Config", modparser.Tag{"type": "Condition", "var": "Combat"}))
	}

	if modDB.Flag(nil, "ManaIncreasedByOvercappedLightningRes") {
		panic("perform: ManaIncreasedByOvercappedLightningRes needs calcs.resistances (unported)")
	}

	if modDB.Flag(env.PlayerMainSkill.SkillCfg, "Condition:CanInflictHallowingFlame") {
		magnitude := 0.0
		if ov := modDB.Override(nil, "HallowingFlameMagnitude"); truthy(ov) {
			magnitude = anyNum(ov)
		} else {
			magnitude = modDB.Sum("INC", nil, "HallowingFlameMagnitude")
		}
		val := math.Floor(25 * (1 + magnitude/100))
		modDB.AddMod(newMod("ExtraAura", "LIST", modparser.Tag{"mod": newMod("PhysicalDamageGainAsFire", "BASE", val, "Hallowing Flame",
			modparser.Tag{"type": "GlobalEffect", "effectType": "Global", "unscalable": true},
			modparser.Tag{"type": "ActorCondition", "actor": "enemy", "var": "HallowingFlame"},
			modparser.Tag{"type": "Multiplier", "var": "HallowingFlame", "actor": "enemy"})}))
	}

	// Check for extra auras
	for _, v := range modDB.List(nil, "ExtraAura") {
		tag, _ := v.(modparser.Tag)
		mod, _ := tag["mod"].(*modparser.Mod)
		if mod == nil {
			continue
		}
		modList := []*modparser.Mod{mod}
		if !truthy(tag["onlyAllies"]) && !(truthy(tag["fromAllies"]) && modDB.Flag(nil, "AlliesAurasCannotAffectSelf")) {
			inc := modDB.Sum("INC", nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			modDB.ScaleAddList(modList, (1+inc/100)*more, false)
			if !truthy(tag["notBuff"]) {
				modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + 1
			}
		}
		if truthy(tag["fromAllies"]) || !modDB.Flag(nil, "SelfAurasCannotAffectAllies") {
			if env.Minion != nil {
				inc := env.Minion.DB.Sum("INC", nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.Minion.DB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				env.Minion.DB.ScaleAddList(modList, (1+inc/100)*more, false)
			}
			totemModBlacklist := mod.Name == "Speed" || mod.Name == "CritMultiplier" || mod.Name == "CritChance"
			if env.PlayerMainSkill.SkillFlags["totem"] && !totemModBlacklist {
				totemMod := modparser.CopyMod(mod)
				if strings.Contains(totemMod.Name, "Condition:") {
					totemMod.Name = strings.ReplaceAll(totemMod.Name, "Condition:", "Condition:Totem")
				} else {
					totemMod.Name = "Totem" + totemMod.Name
				}
				modDB.AddMod(totemMod)
			}
		}
	}
	// allyBuffs extraAura: party tab empty

	// Check for modifiers to apply to actors affected by player auras or curses
	if len(modDB.List(nil, "AffectedByAuraMod")) > 0 {
		// the reference indexes a nil global (affectedByAura) here and errors
		panic("perform: AffectedByAuraMod list non-empty (reference errors on nil affectedByAura)")
	}

	// Merge keystones again to catch any that were added by buffs
	modstore.MergeKeystones(&env.Keystone, modDB)

	// Special handling for Dancing Dervish
	if modDB.Flag(nil, "DisableWeapons") {
		uw := d.UnarmedWeaponData[int(env.Build.ClassID)]
		env.Player.WeaponData1 = map[string]any{
			"type": uw.Type, "AttackRate": uw.AttackRate, "CritChance": uw.CritChance,
			"PhysicalMin": uw.PhysicalMin, "PhysicalMax": uw.PhysicalMax,
		}
		modDB.Conditions["Unarmed"] = true
		// #EVAL env.player.Gloves is never set in the reference, so this
		// branch always marks Unencumbered
		modDB.Conditions["Unencumbered"] = true
	} else if env.WeaponModList1 != nil {
		modDB.AddList(env.WeaponModList1.Mods)
	}

	// Process prerequisites for conditionals
	env.defenceForConditionals(env.playerPA)
	if env.Minion != nil {
		env.defenceForConditionals(env.minionPA)
	}

	// Process misc buffs/modifiers
	env.doActorCharges(env.playerPA)
	env.doActorMisc(env.playerPA)
	if env.Minion != nil {
		env.doActorCharges(env.minionPA)
		env.doActorMisc(env.minionPA)
	}

	env.performAilments(hasGuaranteedBonechill)

	env.doActorCharges(env.enemyPA)
	env.doActorMisc(env.enemyPA)

	env.performTail()
}

// performAilments ports the non-damaging ailment maximum/strongest block
// (CalcPerform.lua L3431-3556).
func (env *Env) performAilments(hasGuaranteedBonechill bool) {
	modDB := env.ModDB
	enemyDB := env.EnemyDB
	d := env.Data
	output := env.playerPA.output

	type ailmentDef struct {
		condition string
		mods      func(num float64) []*modparser.Mod
	}
	shockStacksMax := func() float64 {
		if ov := modDB.Override(nil, "ShockStacksMax"); truthy(ov) {
			return anyNum(ov)
		}
		return modDB.Sum("BASE", nil, "ShockStacksMax")
	}
	scorchStacksMax := func() float64 {
		if ov := modDB.Override(nil, "ScorchStacksMax"); truthy(ov) {
			return anyNum(ov)
		}
		return modDB.Sum("BASE", nil, "ScorchStacksMax")
	}
	ailments := map[string]ailmentDef{
		"Chill": {"Chilled", func(num float64) []*modparser.Mod {
			mods := []*modparser.Mod{newMod("ActionSpeed", "INC", -num, "Chill", modparser.Tag{"type": "Condition", "var": "Chilled"})}
			if modDB.Flag(nil, "ChillEffectIncDamageTaken") {
				mods = append(mods, newMod("DamageTaken", "INC", num, "Ahuana's Bite", modparser.Tag{"type": "Condition", "var": "Chilled"}))
			} else if modDB.Flag(nil, "ChillEffectIncColdDamageTaken") {
				mods = append(mods, newMod("ColdDamageTaken", "INC", num, "Chilled by Hits", modparser.Tag{"type": "Condition", "var": "Chilled"}))
			} else if modDB.Flag(nil, "ChillingAreaIncColdDamageTaken") {
				mods = append(mods, newMod("ColdDamageTaken", "INC", num, "Chilling Area", modparser.Tag{"type": "Condition", "var": "Chilled"}))
			} else if truthy(output["HasBonechill"]) && (hasGuaranteedBonechill || enemyDB.Sum("BASE", nil, "ChillVal") > 0) {
				mods = append(mods, newMod("ColdDamageTaken", "INC", num, "Bonechill", modparser.Tag{"type": "Condition", "var": "Chilled"}))
			}
			if modDB.Flag(nil, "ChillEffectLessDamageDealt") {
				mods = append(mods, newMod("Damage", "MORE", -num/2, "Shaper of Winter", modparser.Tag{"type": "Condition", "var": "Chilled"}))
			}
			return mods
		}},
		"Shock": {"Shocked", func(num float64) []*modparser.Mod {
			var mods []*modparser.Mod
			if modDB.Flag(nil, "ShockCanStack") {
				mods = append(mods, newMod("DamageTaken", "INC", num, "Shock", modparser.Tag{"type": "Condition", "var": "Shocked"}, modparser.Tag{"type": "Multiplier", "var": "ShockStacks", "limit": shockStacksMax()}))
				output["CurrentShock"] = num * math.Min(enemyDB.Sum("BASE", nil, "Multiplier:ShockStacks"), shockStacksMax())
			} else {
				mods = append(mods, newMod("DamageTaken", "INC", num, "Shock", modparser.Tag{"type": "Condition", "var": "Shocked"}))
			}
			return mods
		}},
		"Scorch": {"Scorched", func(num float64) []*modparser.Mod {
			var mods []*modparser.Mod
			if modDB.Flag(nil, "ScorchCanStack") {
				mods = append(mods, newMod("ElementalResist", "BASE", -num, "Scorch", modparser.Tag{"type": "Condition", "var": "Scorched"}, modparser.Tag{"type": "Multiplier", "var": "ScorchStacks", "limit": scorchStacksMax()}))
				output["CurrentScorch"] = num * math.Min(enemyDB.Sum("BASE", nil, "Multiplier:ScorchStacks"), scorchStacksMax())
			} else {
				mods = append(mods, newMod("ElementalResist", "BASE", -num, "Scorch", modparser.Tag{"type": "Condition", "var": "Scorched"}))
			}
			return mods
		}},
		"Brittle": {"Brittle", func(num float64) []*modparser.Mod {
			return []*modparser.Mod{newMod("SelfCritChance", "BASE", num, "Brittle", modparser.Tag{"type": "Condition", "var": "Brittle"})}
		}},
		"Sap": {"Sapped", func(num float64) []*modparser.Mod {
			return []*modparser.Mod{newMod("Damage", "MORE", -num, "Sap", modparser.Tag{"type": "Condition", "var": "Sapped"})}
		}},
	}

	names := make([]string, 0, len(ailments))
	for name := range ailments {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, ailment := range names {
		val := ailments[ailment]
		// #EVAL the reference's first clause (`Val > 0 or Sum(...)`) is
		// always truthy: the Sum returns a number and 0 is truthy in Lua
		if !(enemyDB.Flag(nil, "Condition:Already"+val.condition) || enemyDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") || enemyDB.Sum("BASE", nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments") >= 100) {
			override := 0.0
			minimum := 0.0
			for _, value := range modDB.Tabulate("BASE", nil, ailment+"Base", ailment+"Override", ailment+"Minimum") {
				mod := value.Mod
				effect := anyNum(mod.Value)
				scalesWithSource := mod.Name == ailment+"Base" || mod.Name == ailment+"Minimum"
				if mod.Name == ailment+"Override" {
					enemyDB.AddMod(newMod("Condition:"+val.condition, "FLAG", true, mod.Source))
				}
				if scalesWithSource {
					if ailment != "Scorch" && ailment != "Sap" && ailment != "Brittle" &&
						!env.PlayerMainSkill.SkillModList.Flag(nil, "Cannot"+ailment) &&
						env.PlayerMainSkill.SkillFlags["hit"] && modDB.Flag(nil, "ChecksHighestDamage") {
						effect *= Mod(env.PlayerMainSkill.SkillModList, nil, "Enemy"+ailment+"Effect")
					} else {
						effect *= Mod(modDB, nil, "Enemy"+ailment+"Effect")
					}
				}
				effect *= Mod(enemyDB, nil, "Self"+ailment+"Effect")
				if scalesWithSource {
					args := []any{mod.Source, mod.Flags, mod.KeywordFlags}
					args = append(args, mod.Tags...)
					modDB.AddMod(newMod(ailment+"Override", "BASE", effect, args...))
					if mod.Name == ailment+"Minimum" {
						minimum += effect
					}
				}
				override = math.Max(math.Max(override, effect), minimum)
			}
			maxAilment := 0.0
			if ov := modDB.Override(nil, ailment+"Max"); truthy(ov) {
				maxAilment = anyNum(ov)
			} else {
				for _, skill := range env.PlayerActiveSkills {
					skillMax := d.NonDamagingAilment[ailment].Max + skill.BaseSkillModList.Sum("BASE", nil, ailment+"Max")
					if skillMax > maxAilment {
						maxAilment = skillMax
					}
				}
			}
			output["Maximum"+ailment] = maxAilment
			prec := math.Pow(10, d.NonDamagingAilment[ailment].Precision)
			output["Current"+ailment] = math.Floor(math.Min(math.Max(override, enemyDB.Sum("BASE", nil, ailment+"Val")), maxAilment)*prec) / prec
			for _, mod := range val.mods(outNum(output, "Current"+ailment)) {
				enemyDB.AddMod(mod)
			}
			enemyDB.AddMod(newMod("Condition:Already"+val.condition, "FLAG", true, modparser.Tag{"type": "Condition", "var": val.condition}))
		}
	}

	// Update chill and shock multipliers
	chillEffectMultiplier := enemyDB.Sum("BASE", nil, "Multiplier:ChillEffect")
	if _, ok := output["CurrentChill"]; ok && chillEffectMultiplier < outNum(output, "CurrentChill") {
		enemyDB.AddMod(newMod("Multiplier:ChillEffect", "BASE", outNum(output, "CurrentChill")-chillEffectMultiplier, ""))
	}
	shockEffectMultiplier := enemyDB.Sum("BASE", nil, "Multiplier:ShockEffect")
	if _, ok := output["CurrentShock"]; ok && shockEffectMultiplier < outNum(output, "CurrentShock") {
		enemyDB.AddMod(newMod("Multiplier:ShockEffect", "BASE", outNum(output, "CurrentShock")-shockEffectMultiplier, ""))
	}
}

// performTail ports exposures, consecrated ground, and the ally-life gate
// (CalcPerform.lua L3561-3718). Defence/offence are the dump's stubs.
func (env *Env) performTail() {
	modDB := env.ModDB
	enemyDB := env.EnemyDB

	major, minor := 0, 0
	if m := regexp.MustCompile(`(\d+)_(\d+)`).FindStringSubmatch(env.Build.TreeVersion); m != nil {
		major = atoiSafe(m[1])
		minor = atoiSafe(m[2])
	}

	// Apply exposures
	for _, element := range []string{"Fire", "Cold", "Lightning"} {
		if (major <= 3 && minor <= 15) ||
			!modDB.Flag(nil, "ElementalEquilibrium") ||
			(element == "Fire" && !enemyDB.Flag(nil, "Condition:HitByFireDamage")) ||
			(element == "Cold" && !enemyDB.Flag(nil, "Condition:HitByColdDamage")) ||
			(element == "Lightning" && !enemyDB.Flag(nil, "Condition:HitByLightningDamage")) {
			min := math.Inf(1)
			source := ""
			for _, entry := range enemyDB.Tabulate("BASE", nil, element+"Exposure") {
				if anyNum(entry.Value) < min {
					min = anyNum(entry.Value)
					source = entry.Mod.Source
				}
			}
			if !math.IsInf(min, 1) {
				// Modify the magnitude of all exposures
				for _, entry := range modDB.Tabulate("BASE", nil, "ExtraExposure", "Extra"+element+"Exposure") {
					min += anyNum(entry.Value)
				}
				exposureMin := modDB.Override(nil, "ExposureMin")
				if exposureMin == nil {
					panic("perform: ExposureMin override missing (reference would error in m_min)")
				}
				enemyDB.AddMod(newMod("Condition:Has"+element+"Exposure", "FLAG", true, ""))
				enemyDB.AddMod(newMod(element+"Resist", "BASE", math.Min(min, anyNum(exposureMin)), source))
				modDB.AddMod(newMod("Condition:AppliedExposureRecently", "FLAG", true, ""))
			}
		}
	}

	// Handle consecrated ground effects on enemies
	if enemyDB.Flag(nil, "Condition:OnConsecratedGround") {
		effect := 1 + modDB.Sum("INC", nil, "ConsecratedGroundEffect")/100
		enemyDB.AddMod(newMod("DamageTaken", "INC", math.Floor(enemyDB.Sum("INC", nil, "DamageTakenConsecratedGround")*effect), "Consecrated Ground"))
	}

	// Full DPS and builds without a supported redirect do not need ally Life.
	allyLifeRedirects := []string{
		"takenFromSpectresBeforeYou", "takenFromMinionBeforeYou",
		"takenFromRadianceSentinelBeforeYou", "takenFromVoidSpawnBeforeYou", "takenFromStoneGolemBeforeYou",
		"takenFromTotemsBeforeYou", "takenFromVaalRejuvenationTotemsBeforeYou",
	}
	if modDB.Sum("BASE", nil, allyLifeRedirects...) != 0 {
		env.performAllyLife()
	}

	// Gem level/quality of the main skill (CalcPerform.lua L3862-3916;
	// runs after the defence/offence handoff the dump stubs out)
	output := env.playerPA.output
	mainSkill := env.PlayerMainSkill
	if mainSkill != nil && mainSkill.ActiveEffect != nil && mainSkill.ActiveEffect.SrcInstance != nil {
		baseLevel := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemLevel")
		totalItemLevel := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemItemLevel")
		totalSupportLevel := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemSupportLevel")
		output["GemHasLevel"] = true
		output["GemLevel"] = baseLevel + totalSupportLevel + totalItemLevel

		baseQuality := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemQuality")
		totalItemQuality := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemItemQuality")
		totalSupportQuality := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemSupportQuality")
		socketQuality := mainSkill.SkillModList.Sum("BASE", mainSkill.SkillCfg, "GemSocketQuality")
		output["GemHasQuality"] = true
		output["GemQuality"] = baseQuality + totalSupportQuality + totalItemQuality + socketQuality
	}
}

// lifePool mirrors CalcPerform's ally life pool descriptors (L1222-1232).
type lifePool struct {
	life, redirect, source, limit string
}

var spectreLifePool = &lifePool{"TotalSpectreLife", "takenFromSpectresBeforeYou", "Spectres", "ActiveSpectreLimit"}
var companionshipLifePool = &lifePool{"TotalMinionLife", "takenFromMinionBeforeYou", "Minion", ""}
var minionLifePoolBySkill = map[string]*lifePool{
	"Summon Sentinel of Radiance":        {"TotalRadianceSentinelLife", "takenFromRadianceSentinelBeforeYou", "Sentinel of Radiance", ""},
	"Summon Void Spawn":                  {"TotalVoidSpawnLife", "takenFromVoidSpawnBeforeYou", "Void Spawns", "ActiveVoidSpawnLimit"},
	"Summon Stone Golem of Safeguarding": {"TotalStoneGolemLife", "takenFromStoneGolemBeforeYou", "Stone Golem", ""},
}

// performAllyLife ports the needsAllyLife block (CalcPerform.lua
// L3599-3718). Totem life needs calcTotemLife, which is unported.
func (env *Env) performAllyLife() {
	modDB := env.ModDB
	output := env.playerPA.output

	calculatedLifePool := map[string]bool{}
	spectreCount := 0.0
	firstSpectreLife := 0.0
	haveFirstSpectre := false
	spectreLimit := outNum(output, "ActiveSpectreLimit")
	if ov := anyNum(env.ConfigInput["multiplierSummonedMinion"]); ov != 0 {
		spectreLimit = math.Min(spectreLimit, ov)
	}
	for _, activeSkill := range env.PlayerActiveSkills {
		minion := activeSkill.Minion
		if minion != nil && !activeSkill.SkillFlags["disable"] && !activeSkill.SkillTypes[modparser.SkillType.MinionsAreUndamagable] {
			var skillPool *lifePool
			if activeSkill.SkillFlags["spectre"] {
				skillPool = spectreLifePool
			} else {
				skillPool = minionLifePoolBySkill[activeSkill.ActiveEffect.GrantedEffect.Name]
			}
			var pools []*lifePool
			if skillPool != nil {
				pools = append(pools, skillPool)
			}

			// Companionship is stored on the supported skill, which
			// identifies the one minion it affects.
			hasCompanionship := false
			for _, buff := range activeSkill.BuffListTyped {
				for _, mod := range buff.ModList {
					if mod.Name == companionshipLifePool.redirect {
						hasCompanionship = true
						break
					}
				}
				if hasCompanionship {
					break
				}
			}
			if hasCompanionship {
				pools = append(pools, companionshipLifePool)
				if skillPool != nil && modDB.Sum("BASE", nil, skillPool.redirect) != 0 && modDB.Sum("BASE", nil, companionshipLifePool.redirect) != 0 {
					// Both redirects spend the same minion's Life, so they must share one pool.
					pools = []*lifePool{companionshipLifePool}
					modDB.AddMod(newMod("MinionLifeShares"+skillPool.life, "FLAG", true, "Companionship"))
				}
			}

			minionLife := 0.0
			haveMinionLife := false
			for _, pool := range pools {
				canAdd := ((pool == spectreLifePool && spectreCount < spectreLimit) ||
					(pool != spectreLifePool && !calculatedLifePool[pool.life])) &&
					modDB.Sum("BASE", nil, pool.redirect) != 0 && !truthy(modDB.Override(nil, pool.life))
				if canAdd {
					if !haveMinionLife {
						if minion != env.Minion {
							env.initMinionModDB(activeSkill, nil)
							addMinionModifiers(activeSkill.SkillModList, activeSkill.SkillCfg, minion)
							for _, name := range sortedListKeysOf(env.Buffs) {
								addMinionModifiers(env.Buffs[name], activeSkill.SkillCfg, minion)
							}
							for _, name := range sortedListKeysOf(env.MinionBuffsOut) {
								minion.DB.AddList(env.MinionBuffsOut[name].Mods)
							}
							for _, v := range minion.DB.List(nil, "Keystone") {
								if mods, ok := env.Build.Spec.KeystoneMap[str(v)]; ok {
									minion.DB.AddList(mods)
								}
							}
							pa := &performActor{
								ms: minion.Ms, db: minion.DB, output: minion.Output,
								mainSkill: minion.MainSkill, skills: minion.ActiveSkillList,
								minion: minion, enemy: env.enemyPA, parent: env.playerPA,
							}
							env.doActorAttribsConditions(pa)
						} else {
							env.doActorLifeMana(env.minionPA)
						}
						minionLife = outNum(minion.Output, "Life")
						haveMinionLife = true
					}

					// Void Spawn redirects scale from their summon limit, so
					// their Life pool uses that same limit.
					count := 1.0
					if pool.limit != "" {
						if truthy(output[pool.limit]) {
							count = outNum(output, pool.limit)
						} else {
							count = Val(activeSkill.SkillModList, pool.limit, nil)
						}
					}
					if pool == spectreLifePool {
						count = 1
						spectreCount++
					}
					life := minionLife * count
					modDB.AddMod(newMod(pool.life, "BASE", life, pool.source))
					if pool == spectreLifePool {
						if !haveFirstSpectre {
							haveFirstSpectre = true
							firstSpectreLife = minionLife
						}
					} else {
						calculatedLifePool[pool.life] = true
					}
				}
			}
		}
	}
	if haveFirstSpectre && spectreCount < spectreLimit {
		// Each Raise Spectre group represents one Spectre; the first group fills any slots left over.
		modDB.AddMod(newMod(spectreLifePool.life, "BASE", firstSpectreLife*(spectreLimit-spectreCount), spectreLifePool.source))
	}

	for _, activeSkill := range env.PlayerActiveSkills {
		if activeSkill.SkillFlags["totem"] && !activeSkill.SkillFlags["disable"] && truthy(activeSkill.SkillData["totemLevel"]) && activeSkill.SkillTotemId != nil {
			panic("perform: totem ally-life needs calcTotemLife (unported)")
		}
	}
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
