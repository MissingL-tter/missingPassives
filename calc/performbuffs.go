// CalcPerform.lua L2141-3718: buffs/curses/links, curse slots, guards,
// buff application, ailments, exposures, and the ally-life gate. The party
// tab is always empty for ladder replays, so ally buff/curse/warcry/link
// sections reduce to no-ops.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
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
// modf(util.RoundHalfUp(v*scale, 2, 0)) (the integer part), non-integers scale raw.
func totemScaleValue(v, scale float64) float64 {
	if math.Floor(v) == v {
		r := util.RoundHalfUp(v*scale, 2)
		return math.Trunc(r)
	}
	return v * scale
}

func scaleTotemMod(mod *modparser.Mod, scale float64) *modparser.Mod {
	totemMod := cloneMod(mod)
	totemMod.Name = "Totem" + totemMod.Name
	if scale != 1 {
		switch v := totemMod.Value.(type) {
		case modparser.Num:
			totemMod.Value = modparser.Num(totemScaleValue(float64(v), scale))
		case modparser.ModRef:
			if v.Mod != nil {
				innerCopy := cloneMod(v.Mod)
				innerCopy.Value = modparser.Num(totemScaleValue(valueNum(innerCopy.Value), scale))
				v.Mod = innerCopy
				totemMod.Value = v
			}
		}
	}
	return totemMod
}

// dedupExtraModList ports the compareModParams accumulation used for
// ExtraAuraEffect / ExtraAuraDebuffEffect / ExtraLinkEffect lists.
func dedupExtraModList(entries []modparser.Value) []*modparser.Mod {
	var out []*modparser.Mod
	for _, v := range entries {
		mod := modRefOf(v)
		if mod == nil {
			continue
		}
		add := true
		for _, existing := range out {
			if modparser.CompareModParams(existing, mod) {
				existing.Value = modparser.Num(valueNum(existing.Value) + valueNum(mod.Value))
				add = false
				break
			}
		}
		if add {
			out = append(out, cloneMod(mod))
		}
	}
	return out
}

// performBuffs continues Perform at CalcPerform.lua L2141.
// ~1,000-line function, a straight transliteration of the reference body; left unsplit by decision (2026-08-29).
func (env *Env) performBuffs(hasGuaranteedBonechill bool, nonUniqueFlasksApplyToMinion bool) {
	modDB := env.ModDB
	enemyDB := env.EnemyDB
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
		if !activeSkill.SkillFlags["disable"] && !env.LimitedSkills[env.cacheSkillUUID(activeSkill)] {
			geName := activeSkill.ActiveEffect.GrantedEffect.Name
			part2 := activeSkill.SkillPart.V == 2
			if (geName == "Blight" || geName == "Blight of Contagion" || geName == "Blight of Atrophy") && part2 {
				vals := env.cachedOutputValues(activeSkill)
				rate, duration := vals.Speed.V, vals.Duration.V
				baseMaxStages := activeSkill.SkillModList.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, "BlightBaseMaxStages")
				maximum := math.Min(math.Floor(rate*duration)-1, baseMaxStages-1)
				activeSkill.SkillModList.AddMod(newModS("Multiplier:"+noSpace(geName)+"MaxStages", modparser.Base, modparser.Num(maximum), "Base"))
				activeSkill.SkillModList.AddMod(newModS("Multiplier:"+noSpace(geName)+"StageAfterFirst", modparser.Base, modparser.Num(maximum), "Base"))
			}
			if geName == "Penance Brand of Dissipation" && part2 {
				// HitSpeed is the brand activation frequency
				vals := env.cachedOutputValues(activeSkill)
				activationFrequency, duration := vals.HitSpeed.V, vals.Duration.V
				ticks := math.Max(math.Min(math.Floor(activationFrequency*duration)-1, 19), 0)
				activeSkill.SkillModList.AddMod(newModS("Multiplier:PenanceBrandofDissipationMaxStages", modparser.Base, modparser.Num(ticks), "Base"))
				activeSkill.SkillModList.AddMod(newModS("Multiplier:PenanceBrandofDissipationStageAfterFirst", modparser.Base, modparser.Num(ticks), "Base"))
			}
			if (geName == "Scorching Ray" || geName == "Scorching Ray of Immolation") && part2 {
				maximum := 7.0
				activeSkill.SkillModList.AddMod(newModS("Multiplier:"+noSpace(geName)+"MaxStages", modparser.Base, modparser.Num(maximum), "Base"))
				activeSkill.SkillModList.AddMod(newModS("Multiplier:"+noSpace(geName)+"StageAfterFirst", modparser.Base, modparser.Num(maximum), "Base"))
			}
			if geName == "Earthquake of Amplification" && part2 {
				duration := env.cachedOutputValues(activeSkill).Duration.V
				durationMulti := math.Floor(duration * 10)
				activeSkill.SkillModList.AddMod(newModS("Multiplier:100msEarthquakeDuration", modparser.Base, modparser.Num(durationMulti), "Skill:EarthquakeAltX"))
			}
		}
	}

	appliedCombustion := false
	warcryList := map[string]bool{}
	for _, activeSkill := range env.PlayerActiveSkills {
		skillModList := activeSkill.SkillModList
		skillCfg := activeSkill.SkillCfg
		for _, buff := range activeSkill.BuffListTyped {
			buffName := buff.Name
			buffType := buff.Type
			if buff.Cond != "" && !skillModList.GetCondition(buff.Cond, skillCfg) {
				// Nothing!
			} else if buffType == "GlobalDB" {
				modDB.AddList(buff.ModList)
			} else if buffType == "Buff" {
				if env.ModeBuffs && (!activeSkill.SkillFlags["totem"] || buff.AllowTotemBuff.Or(false)) {
					var buffCfg *modstore.Cfg
					var modStore modstore.Store = modDB
					if buff.ActiveSkillBuff {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if !buff.ApplyNotPlayer.Or(false) {
						activeSkill.BuffSkill = true
						modDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
						srcList := modstore.NewList(nil)
						inc := modStore.Sum(modparser.Inc, buffCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer") + skillModList.Sum(modparser.Inc, buffCfg, noSpace(buffName)+"Effect")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnSelf")
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods, buffs, buffName)
						if activeSkill.SkillData.Flag("thisIsNotABuff") {
							notBuff[buffName] = true
						}
					}
					if env.Minion != nil && !env.Minion.Hostile && (buff.ApplyMinions.Or(false) || buff.ApplyAllies.Or(false) || skillModList.Flag(nil, "BuffAppliesToAllies")) {
						activeSkill.MinionBuffSkill = true
						env.Minion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
						srcList := modstore.NewList(nil)
						inc := modStore.Sum(modparser.Inc, buffCfg, "BuffEffect") + env.Minion.DB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")
						more := modStore.More(buffCfg, "BuffEffect") * env.Minion.DB.More(nil, "BuffEffectOnSelf")
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods, minionBuffs, buffName)
					}
				}
			} else if buffType == "Guard" {
				if env.ModeBuffs && (!activeSkill.SkillFlags["totem"] || buff.AllowTotemBuff.Or(false)) {
					var buffCfg *modstore.Cfg
					var modStore modstore.Store = modDB
					if buff.ActiveSkillBuff {
						buffCfg = skillCfg
						modStore = skillModList
					}
					if !buff.ApplyNotPlayer.Or(false) {
						activeSkill.BuffSkill = true
						srcList := modstore.NewList(nil)
						inc := modStore.Sum(modparser.Inc, buffCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
						more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnSelf")
						srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
						mergeBuff(srcList.Mods, guards, buffName)
					}
				}
			} else if buffType == "Warcry" {
				if env.ModeBuffs {
					var modStore modstore.Store = skillModList
					warcryName := noSpace(strings.ReplaceAll(strings.ReplaceAll(buffName, " Cry", ""), "'s", ""))
					baseExerts := modStore.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, warcryName+"ExertedAttacks")
					if baseExerts > 0 {
						extraExertions := modStore.Sum(modparser.Base, nil, "ExtraExertedAttacks")
						exertMultiplier := modStore.More(nil, "ExtraExertedAttacks")
						modDB.AddMod(newMod("Num"+warcryName+"Exerts", modparser.Base, modparser.Num(math.Floor((baseExerts+extraExertions)*exertMultiplier))))
						if !warcryList[buffName] {
							modDB.AddMod(newModS("Multiplier:ExertingWarcryCount", modparser.Base, modparser.Num(1.0), buffName))
							warcryList[buffName] = true
						}
					}
					if !skillModList.Flag(nil, "CannotShareWarcryBuffs") {
						warcryPower := 0.0
						if ov, ok := modDB.Override(nil, "WarcryPower"); ok {
							warcryPower = valueNum(ov)
						} else {
							warcryPower = math.Max(modDB.Sum(modparser.Base, nil, "WarcryPower")*(1+modDB.Sum(modparser.Inc, nil, "WarcryPower")/100), modDB.Sum(modparser.Base, nil, "MinimumWarcryPower"))
						}
						for i, warcryBuff := range buff.ModList {
							if len(warcryBuff.Tags) > 0 {
								if tag, ok := warcryBuff.Tags[0].(*modparser.GlobalEffectTag); ok && tag.EffectType == "Warcry" && tag.Div.Set {
									power := warcryPower
									if tag.Limit.Set {
										power = math.Min(warcryPower, tag.Limit.V)
									}
									// The reference writes this straight into the
									// shared mod's tag (and the dump scrubs it off
									// between variants). Copy-on-write keeps the
									// loaded data immutable: the buff list is
									// env-owned, so the copy is visible everywhere
									// the reference's in-place write would be.
									cow := copyModForTagWrite(warcryBuff)
									cow.Tags[0].(*modparser.GlobalEffectTag).WarcryPowerBonus = opt(math.Floor(power / tag.Div.V))
									buff.ModList[i] = cow
								}
							}
						}
						fullDuration := env.calcSkillDuration(modStore, skillCfg, activeSkill.SkillData, enemyDB)
						actualCooldown := 0.0
						if ov, ok := modStore.Override(skillCfg, "CooldownRecovery"); ok {
							actualCooldown = valueNum(ov)
						} else {
							actualCooldown = (activeSkill.SkillData.N("cooldown") + modStore.Sum(modparser.Base, skillCfg, "CooldownRecovery")) / Mod(modStore, skillCfg, "CooldownRecovery")
						}
						uptime := math.Min(fullDuration/actualCooldown, 1)
						if modDB.Flag(nil, "Condition:WarcryMaxHit") {
							uptime = 1
						}
						var extraWarcryModList []*modparser.Mod
						warcryBuffBonus := func(m *modparser.Mod) float64 {
							if len(m.Tags) > 0 {
								if tag, ok := m.Tags[0].(*modparser.GlobalEffectTag); ok && tag.WarcryPowerBonus.Set {
									return tag.WarcryPowerBonus.V
								}
							}
							return 1
						}
						if !modDB.Flag(nil, "CannotGainWarcryBuffs") {
							if !buff.ApplyNotPlayer.Or(false) {
								activeSkill.BuffSkill = true
								modDB.Conditions.Set("AffectedBy"+warcryName, true)
								srcList := modstore.NewList(nil)
								inc := modStore.Sum(modparser.Inc, skillCfg, "BuffEffect", "BuffEffectOnSelf", "BuffEffectOnPlayer")
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
							env.Minion.DB.Conditions.Set("AffectedBy"+warcryName, true)
							srcList := modstore.NewList(nil)
							inc := skillModList.Sum(modparser.Inc, skillCfg, "BuffEffect") + env.Minion.DB.Sum(modparser.Inc, skillCfg, "BuffEffectOnSelf")
							more := skillModList.More(skillCfg, "BuffEffect") * env.Minion.DB.More(skillCfg, "BuffEffectOnSelf")
							for _, warcryBuff := range buff.ModList {
								mult := (1 + inc/100) * more * warcryBuffBonus(warcryBuff) * uptime
								srcList.ScaleAddList([]*modparser.Mod{warcryBuff}, mult, false)
							}
							// Special handling for the minion side to add the flat damage bonus
							if activeSkill.ActiveEffect.GrantedEffect.Name == "Rallying Cry" {
								warcryPowerBonus := math.Floor(math.Min(warcryPower, 30) / 5)
								rallyingWeaponEffect := math.Floor(activeSkill.SkillModList.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, "RallyingCryAllyDamageBonusPer5Power") * warcryPowerBonus)
								rallyInc := modStore.Sum(modparser.Inc, skillCfg, "BuffEffect") + env.Minion.DB.Sum(modparser.Inc, skillCfg, "BuffEffectOnSelf")
								rallyingBonusMoreMultiplier := 1 + activeSkill.SkillModList.Sum(modparser.Base, env.PlayerMainSkill.SkillCfg, "RallyingCryMinionDamageBonusMultiplier")
								for _, damageType := range []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"} {
									dmg := dmgOf(weaponOf(env.Player.WeaponData1), damageType)
									for _, side := range []struct {
										suffix string
										value  float64
									}{{"Min", dmg.Min}, {"Max", dmg.Max}} {
										if side.value != 0 {
											extraWarcryModList = append(extraWarcryModList, newModSF(damageType+side.suffix, modparser.Base, modparser.Num(side.value*rallyingWeaponEffect/100), "Rallying Cry", modparser.FlagNone, modparser.KeywordAttack, &modparser.GlobalEffectTag{EffectType: "Warcry", Div: opt(5.0), Limit: opt(30.0)}))
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
					if !activeSkill.SkillData.Flag("auraCannotAffectSelf") {
						inc := skillModList.Sum(modparser.Inc, skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnSelf", "AuraEffectOnSelf", "AuraBuffEffect", "SkillAuraEffectOnSelf")
						mult := (1 + inc/100) * more
						// allyBuffs is empty: the ally-effect comparison passes
						activeSkill.BuffSkill = true
						modDB.Conditions.Set("AffectedByAura", true)
						if strings.HasPrefix(buffName, "Vaal") && len(buffName) > 5 {
							modDB.Conditions.Set("AffectedBy"+noSpace(buffName[5:]), true)
						}
						modDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
						srcList := modstore.NewList(nil)
						srcList.ScaleAddList(buff.ModList, mult, false)
						srcList.ScaleAddList(extraAuraModList, mult, false)
						mergeBuff(srcList.Mods, buffs, buffName)
					}
					if !(modDB.Flag(nil, "SelfAurasCannotAffectAllies") || modDB.Flag(nil, "SelfAurasOnlyAffectYou") || modDB.Flag(nil, "SelfAuraSkillsCannotAffectAllies")) {
						if env.Minion != nil && (!env.Minion.Hostile || modDB.Flag(nil, "AurasAffectEnemies")) {
							inc := skillModList.Sum(modparser.Inc, skillCfg, "AuraEffect", "BuffEffect") + env.Minion.DB.Sum(modparser.Inc, skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect") * env.Minion.DB.More(skillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
							mult := (1 + inc/100) * more
							activeSkill.MinionBuffSkill = true
							env.Minion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
							env.Minion.DB.Conditions.Set("AffectedByAura", true)
							srcList := modstore.NewList(nil)
							srcList.ScaleAddList(buff.ModList, mult, false)
							srcList.ScaleAddList(extraAuraModList, mult, false)
							mergeBuff(srcList.Mods, minionBuffs, buffName)
						}
						if modDB.Flag(nil, "AurasAffectEnemies") && !skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget") {
							inc := skillModList.Sum(modparser.Inc, skillCfg, "AuraEffect", "BuffEffect")
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
						env.PlayerMainSkill.SkillModList.Conditions.Set("AffectedBy"+noSpace(buffName), true)
						env.PlayerMainSkill.SkillModList.Conditions.Set("AffectedByAura", true)
						srcList := modstore.NewList(nil)
						inc := skillModList.Sum(modparser.Inc, skillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
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
							if other.Type == "AuraDebuff" {
								auraDebuffFound = true
								break
							}
						}
						if !auraDebuffFound {
							activeSkill.DebuffSkill = true
							extraDebuffModList := dedupExtraModList(modDB.List(skillCfg, "ExtraAuraDebuffEffect"))
							// The reference merges the UNSCALED list here
							mergeBuff(extraDebuffModList, debuffs, buffName)
						}
					}
				}
			} else if buffType == "Debuff" || buffType == "AuraDebuff" {
				var stackCount float64
				if stackVar := buff.StackVar; stackVar != "" {
					stackCount = skillModList.Sum(modparser.Base, skillCfg, "Multiplier:"+stackVar)
					if buff.StackLimit.Set {
						stackCount = math.Min(stackCount, buff.StackLimit.V)
					}
				} else if activeSkill.SkillData.Flag("stackCount") {
					stackCount = activeSkill.SkillData.N("stackCount")
				} else {
					stackCount = 1
				}
				if env.ModeEffective && stackCount > 0 {
					activeSkill.DebuffSkill = true
					enemyDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
					modDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
					srcList := modstore.NewList(nil)
					mult := 1.0
					var extraAuraModList []*modparser.Mod
					if buffType == "AuraDebuff" {
						extraAuraModList = dedupExtraModList(modDB.List(skillCfg, "ExtraAuraDebuffEffect"))
						mult = 0
						if !modDB.Flag(nil, "SelfAurasOnlyAffectYou") {
							inc := skillModList.Sum(modparser.Inc, skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							more := skillModList.More(skillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
							mult = (1 + inc/100) * more
						}
					}
					if buffType == "Debuff" {
						inc := skillModList.Sum(modparser.Inc, skillCfg, "DebuffEffect")
						more := skillModList.More(skillCfg, "DebuffEffect")
						mult = (1 + inc/100) * more
					}
					srcList.ScaleAddList(buff.ModList, mult*stackCount, false)
					srcList.ScaleAddList(extraAuraModList, mult*stackCount, false)
					if activeSkill.SkillData.Flag("stackCount") || buff.StackVar != "" {
						srcList.AddMod(newModS("Multiplier:"+buffName+"Stack", modparser.Base, modparser.Num(stackCount), buffName))
					}
					mergeBuff(srcList.Mods, debuffs, buffName)
				}
			} else if buffType == "Curse" || buffType == "CurseBuff" {
				mark := activeSkill.SkillTypes[modparser.SkillTypeMark]
				modDB.Conditions.Set("SelfCast"+noSpace(buffName), !(activeSkill.SkillTypes[modparser.SkillTypeTriggered] || activeSkill.SkillTypes[modparser.SkillTypeAura]))
				skipCurse := false
				if env.ConfigInput.BalanceOfTerrorSelfCast[noSpace(buffName)] && !mark {
					skipCurse = true
				}
				if !skipCurse && (env.ModeEffective && (!enemyDB.Flag(nil, "Hexproof") || modDB.Flag(nil, "CursesIgnoreHexproof") || activeSkill.SkillData.Flag("ignoreHexLimit") || activeSkill.SkillData.Flag("ignoreHexproof")) || mark) {
					curse := &curseEntry{
						name:                   buffName,
						fromPlayer:             true,
						priority:               env.determineCursePriority(buffName, activeSkill),
						isMark:                 mark,
						ignoreHexLimit:         (modDB.Flag(activeSkill.SkillCfg, "CursesIgnoreHexLimit") || activeSkill.SkillData.Flag("ignoreHexLimit")) && !mark,
						socketedCursesHexLimit: modDB.Flag(activeSkill.SkillCfg, "SocketedCursesAdditionalLimit"),
					}
					inc := skillModList.Sum(modparser.Inc, skillCfg, "CurseEffect") + enemyDB.Sum(modparser.Inc, nil, "CurseEffectOnSelf")
					if activeSkill.SkillTypes[modparser.SkillTypeAura] && !activeSkill.SkillTypes[modparser.SkillTypeRemoteMined] {
						inc += skillModList.Sum(modparser.Inc, skillCfg, "AuraEffect")
					}
					more := skillModList.More(skillCfg, "CurseEffect")
					if !curse.isMark {
						more *= enemyDB.More(nil, "CurseEffectOnSelf")
					}
					mult := 0.0
					if !((modDB.Flag(nil, "SelfAurasOnlyAffectYou") || skillModList.Flag(skillCfg, "SelfAurasAffectYouAndLinkedTarget")) && activeSkill.SkillTypes[modparser.SkillTypeAura]) {
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
						buffInc := modDB.Sum(modparser.Inc, skillCfg, "BuffEffectOnSelf")
						buffMore := modDB.More(skillCfg, "BuffEffectOnSelf")
						curse.buffModList.ScaleAddList(temp.Mods, (1+buffInc/100)*buffMore, false)
						if env.Minion != nil {
							curse.minionBuffModList = modstore.NewList(nil)
							minionInc := env.Minion.DB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")
							minionMore := env.Minion.DB.More(nil, "BuffEffectOnSelf")
							curse.minionBuffModList.ScaleAddList(temp.Mods, (1+minionInc/100)*minionMore, false)
						}
					}
					curses = append(curses, curse)
				}
			} else if buffType == "Link" {
				linksApplyToMinions := env.Minion != nil && modDB.Flag(nil, "Condition:CanLinkToMinions") && modDB.Flag(nil, "Condition:LinkedToMinion") &&
					!env.Minion.DB.Flag(nil, "Condition:CannotBeDamaged") &&
					!(env.Minion.MainSkill.SummonSkill != nil && env.Minion.MainSkill.SummonSkill.SkillTypes[modparser.SkillTypeMinionsAreUndamagable])
				var linkApplied bool
				if env.ModeBuffs && !linkApplied && linksApplyToMinions {
					var extraLinkModList []*modparser.Mod
					for _, v := range modDB.List(skillCfg, "ExtraLinkEffect") {
						mod := modRefOf(v)
						if mod == nil {
							continue
						}
						add := true
						for _, existing := range extraLinkModList {
							if modparser.CompareModParams(existing, mod) {
								existing.Value = modparser.Num(valueNum(existing.Value) + valueNum(mod.Value))
								add = false
								break
							}
						}
						if add {
							extraLinkModList = append(extraLinkModList, cloneMod(mod))
							if mod.Name == "ParentNonUniqueFlasksAppliedToYou" {
								nonUniqueFlasksApplyToMinion = true
							}
						}
					}
					inc := skillModList.Sum(modparser.Inc, skillCfg, "LinkEffect", "BuffEffect")
					more := skillModList.More(skillCfg, "LinkEffect", "BuffEffect")
					activeSkill.MinionBuffSkill = true
					env.Minion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
					env.Minion.DB.Conditions.Set("AffectedByLink", true)
					srcList := modstore.NewList(nil)
					inc += env.Minion.DB.Sum(modparser.Inc, nil, "BuffEffectOnSelf", "LinkEffectOnSelf")
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
				effect = math.Floor(6 * (1 + modDB.Sum(modparser.Inc, nil, "MinionWitherEffect")/100))
			} else {
				effect = math.Floor(6 * (1 + modDB.Sum(modparser.Inc, nil, "WitherEffect")/100))
			}
			modDB.AddMod(newMod("WitherEffectStack", modparser.Max, modparser.Num(effect)))
		}
		// Handle combustion
		if enemyDB.Flag(nil, "Condition:Ignited") && (activeSkill.SkillTypes[modparser.SkillTypeDamage] || activeSkill.SkillTypes[modparser.SkillTypeAttack]) && !appliedCombustion {
			for _, support := range activeSkill.SupportList {
				if support.GrantedEffect.Name == "Combustion" {
					if !skillModList.Flag(activeSkill.SkillCfg, "CannotIgnite") {
						value := skillModList.Sum(modparser.Base, activeSkill.SkillCfg, "CombustionFireResist")
						enemyDB.AddMod(newModS("FireResist", modparser.Base, modparser.Num(value), "Combustion", &modparser.GlobalEffectTag{EffectType: "Debuff", EffectName: "Combustion"}, &modparser.CondTag{Var: "Ignited"}))
						appliedCombustion = true
					}
					break
				}
			}
		}
		if activeSkill.Minion != nil && activeSkill.Minion.ActiveSkillList != nil {
			castingMinion := activeSkill.Minion
			for _, activeMinionSkill := range castingMinion.ActiveSkillList {
				setSpectreSource := func(modList []*modparser.Mod, sourceSkill string) []*modparser.Mod {
					if !activeSkill.SkillFlags["spectre"] {
						return modList
					}
					source := "Spectre:"
					if sourceSkill != "" {
						source = source + sourceSkill + " - " + castingMinion.MinionData.Name
					} else {
						source = source + castingMinion.MinionData.Name
					}
					// The reference stamps the shared tables in place
					// (CalcPerform.lua L2724-2736), which reaches mods aliased
					// into the loaded game data; stamped clones carry the same
					// bytes into every merge without mutating data.Skills
					// (lua-residue.md T2).
					out := make([]*modparser.Mod, len(modList))
					for i, m := range modList {
						c := m.Clone()
						c.Source = source
						c.SourceSet = true
						out[i] = c
					}
					return out
				}
				minionSkillModList := activeMinionSkill.SkillModList
				minionSkillCfg := activeMinionSkill.SkillCfg
				for _, buff := range activeMinionSkill.BuffListTyped {
					buffName := buff.Name
					buffType := buff.Type
					if buffType == "Buff" {
						if env.ModeBuffs && activeMinionSkill.SkillData.Flag("enable") {
							var buffCfg *modstore.Cfg
							var modStore modstore.Store = castingMinion.DB
							if buff.ActiveSkillBuff {
								buffCfg = minionSkillCfg
								modStore = minionSkillModList
							}
							if buff.ApplyAllies.Or(false) {
								activeMinionSkill.BuffSkill = true
								modDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
								srcList := modstore.NewList(nil)
								inc := modStore.Sum(modparser.Inc, buffCfg, "BuffEffect", "BuffEffectOnPlayer") + modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf")
								more := modStore.More(buffCfg, "BuffEffect", "BuffEffectOnPlayer") * modDB.More(nil, "BuffEffectOnSelf")
								srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
								mergeBuff(srcList.Mods, buffs, buffName)
								mergeBuff(buff.ModList, buffs, buffName)
								if activeMinionSkill.SkillData.Flag("thisIsNotABuff") {
									notBuff[buffName] = true
								}
							}
							envMinionCheck := env.Minion != nil && (env.Minion == castingMinion || buff.ApplyAllies.Or(false))
							if buff.ApplyMinions.Or(false) || envMinionCheck {
								activeMinionSkill.MinionBuffSkill = true
								if envMinionCheck {
									env.Minion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
								} else {
									castingMinion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
								}
								srcList := modstore.NewList(nil)
								names := []string{"BuffEffect"}
								if env.Minion == castingMinion {
									names = append(names, "BuffEffectOnSelf")
								}
								inc := modStore.Sum(modparser.Inc, buffCfg, names...)
								more := modStore.More(buffCfg, names...)
								srcList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
								mergeBuff(srcList.Mods, minionBuffs, buffName)
								mergeBuff(buff.ModList, minionBuffs, buffName)
								if activeMinionSkill.SkillData.Flag("thisIsNotABuff") {
									notBuff[buffName] = true
								}
							}
						}
					} else if buffType == "Aura" {
						if env.ModeBuffs && activeMinionSkill.SkillData.Flag("enable") {
							extraAuraModList := dedupExtraModList(castingMinion.DB.List(minionSkillCfg, "ExtraAuraEffect"))
							if !(castingMinion.DB.Flag(nil, "SelfAurasCannotAffectAllies") || castingMinion.DB.Flag(nil, "SelfAurasOnlyAffectYou") || castingMinion.DB.Flag(nil, "SelfAuraSkillsCannotAffectAllies") || minionSkillModList.Flag(minionSkillCfg, "SelfAurasAffectYouAndLinkedTarget")) {
								if !modDB.Flag(nil, "AlliesAurasCannotAffectSelf") && !modDB.Conditions.Get("AffectedBy"+noSpace(buffName)) {
									inc := minionSkillModList.Sum(modparser.Inc, minionSkillCfg, "AuraEffect", "BuffEffect", "BuffEffectOnPlayer", "AuraBuffEffect") + modDB.Sum(modparser.Inc, minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect") * modDB.More(minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									mult := (1 + inc/100) * more
									activeMinionSkill.BuffSkill = true
									modDB.Conditions.Set("AffectedByAura", true)
									if strings.HasPrefix(buffName, "Vaal") && len(buffName) > 5 {
										modDB.Conditions.Set("AffectedBy"+noSpace(buffName[5:]), true)
									}
									modDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
									srcList := modstore.NewList(nil)
									srcList.ScaleAddList(buff.ModList, mult, false)
									srcList.ScaleAddList(extraAuraModList, mult, false)
									srcList.Mods = setSpectreSource(srcList.Mods, buffName)
									mergeBuff(srcList.Mods, buffs, buffName)
								}
								if env.Minion != nil && !env.Minion.DB.Conditions.Get("AffectedBy"+noSpace(buffName)) && (env.Minion != castingMinion || !activeSkill.SkillData.Flag("auraCannotAffectSelf")) {
									inc := minionSkillModList.Sum(modparser.Inc, minionSkillCfg, "AuraEffect", "BuffEffect") + env.Minion.DB.Sum(modparser.Inc, minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect") * env.Minion.DB.More(minionSkillCfg, "BuffEffectOnSelf", "AuraEffectOnSelf")
									mult := (1 + inc/100) * more
									activeMinionSkill.MinionBuffSkill = true
									env.Minion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
									env.Minion.DB.Conditions.Set("AffectedByAura", true)
									srcList := modstore.NewList(nil)
									srcList.ScaleAddList(buff.ModList, mult, false)
									srcList.ScaleAddList(extraAuraModList, mult, false)
									srcList.Mods = setSpectreSource(srcList.Mods, buffName)
									mergeBuff(srcList.Mods, minionBuffs, buffName)
								}
								// The reference additionally stamps the SHARED buff/extra
								// mod tables here ("export list mutation",
								// CalcPerform.lua L2733-2736) — a write into loaded game
								// data whose only effect is cross-build; deliberately not
								// reproduced (lua-residue.md T2). The merges above carry
								// the stamped clones.
								if env.PlayerMainSkill.SkillFlags["totem"] && !env.PlayerMainSkill.SkillModList.Conditions.Get("AffectedBy"+noSpace(buffName)) {
									activeMinionSkill.TotemBuffSkill = true
									env.PlayerMainSkill.SkillModList.Conditions.Set("AffectedBy"+noSpace(buffName), true)
									env.PlayerMainSkill.SkillModList.Conditions.Set("AffectedByAura", true)
									srcList := modstore.NewList(nil)
									inc := minionSkillModList.Sum(modparser.Inc, minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "AuraBuffEffect")
									scale := math.Max((1+inc/100)*more, 0)
									for _, modList := range [][]*modparser.Mod{extraAuraModList, buff.ModList} {
										for _, mod := range modList {
											if mod.Name == "EnergyShield" || mod.Name == "Armour" || mod.Name == "Evasion" || totemAuraModRe.MatchString(mod.Name) {
												srcList.AddMod(scaleTotemMod(mod, scale))
											}
										}
									}
									srcList.Mods = setSpectreSource(srcList.Mods, "")
									mergeBuff(srcList.Mods, buffs, "Totem "+buffName)
								}
							}
						}
					} else if buffType == "Curse" {
						if env.ModeEffective && activeMinionSkill.SkillData.Flag("enable") && (!enemyDB.Flag(nil, "Hexproof") || activeMinionSkill.SkillTypes[modparser.SkillTypeMark]) {
							curse := &curseEntry{
								name:     buffName,
								priority: env.determineCursePriority(buffName, activeMinionSkill),
							}
							inc := minionSkillModList.Sum(modparser.Inc, minionSkillCfg, "CurseEffect") + enemyDB.Sum(modparser.Inc, nil, "CurseEffectOnSelf")
							more := minionSkillModList.More(minionSkillCfg, "CurseEffect") * enemyDB.More(nil, "CurseEffectOnSelf")
							curse.modList = modstore.NewList(nil)
							curse.modList.ScaleAddList(buff.ModList, (1+inc/100)*more, false)
							minionCurses = append(minionCurses, curse)
						}
					} else if buffType == "Debuff" || buffType == "AuraDebuff" {
						var stackCount float64
						if stackVar := buff.StackVar; stackVar != "" {
							stackCount = minionSkillModList.Sum(modparser.Base, minionSkillCfg, "Multiplier:"+stackVar)
							if buff.StackLimit.Set {
								stackCount = math.Min(stackCount, buff.StackLimit.V)
							}
						} else if activeMinionSkill.SkillData.Flag("stackCount") {
							stackCount = activeMinionSkill.SkillData.N("stackCount")
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
									inc := minionSkillModList.Sum(modparser.Inc, minionSkillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
									more := minionSkillModList.More(minionSkillCfg, "AuraEffect", "BuffEffect", "DebuffEffect")
									mult = (1 + inc/100) * more
									if enemyDB.Conditions.Get("AffectedBy" + noSpace(buffName)) {
										mult = 0
									}
								}
							}
							enemyDB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
							if env.Minion != nil && env.Minion == castingMinion {
								env.Minion.DB.Conditions.Set("AffectedBy"+noSpace(buffName), true)
							}
							if buffType == "Debuff" {
								inc := minionSkillModList.Sum(modparser.Inc, minionSkillCfg, "DebuffEffect")
								more := minionSkillModList.More(minionSkillCfg, "DebuffEffect")
								mult = (1 + inc/100) * more
							}
							srcList.ScaleAddList(buff.ModList, mult*stackCount, false)
							if activeMinionSkill.SkillData.Flag("stackCount") || buff.StackVar != "" {
								srcList.AddMod(newModS("Multiplier:"+buffName+"Stack", modparser.Base, modparser.Num(activeMinionSkill.SkillData.N("stackCount")), buffName))
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
			tripleDmgChancePerEndurance := modDB.Sum(modparser.Base, nil, "PerBrutalTripleDamageChance")
			modDB.AddMod(newMod("TripleDamageChance", modparser.Base, modparser.Num(tripleDmgChancePerEndurance), &modparser.MultiplierTag{Var: "BrutalCharge"}))
		}
		if modDB.Flag(nil, "UseFrenzyCharges") && modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
			dmgPerAffliction := modDB.Sum(modparser.Base, nil, "PerAfflictionAilmentDamage")
			effectPerAffliction := modDB.Sum(modparser.Base, nil, "PerAfflictionNonDamageEffect")
			modDB.AddMod(newModSF("Damage", modparser.More, modparser.Num(dmgPerAffliction), "Affliction Charges", modparser.FlagNone, modparser.KeywordAilment, &modparser.MultiplierTag{Var: "AfflictionCharge"}))
			for _, name := range []string{"EnemyChillEffect", "EnemyShockEffect", "EnemyFreezeEffect", "EnemyScorchEffect", "EnemyBrittleEffect", "EnemySapEffect"} {
				modDB.AddMod(newModS(name, modparser.More, modparser.Num(effectPerAffliction), "Affliction Charges", &modparser.MultiplierTag{Var: "AfflictionCharge"}))
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
			tag, ok := v.(modparser.SkillRef)
			if !ok {
				continue
			}
			grantedEffect := data.Skills[tag.SkillID]
			if grantedEffect == nil {
				continue
			}
			gemModList := modstore.NewList(nil)
			env.mergeSkillInstanceMods(gemModList, &ActiveEffect{
				GrantedEffect: grantedEffect,
				Level:         tag.Level.Or(0),
				Quality:       0,
			}, nil)
			var curseModList []*modparser.Mod
			for _, mod := range gemModList.Mods {
				for _, tv := range mod.Tags {
					if mt, ok := tv.(*modparser.GlobalEffectTag); ok && mt.EffectType == "Curse" {
						curseModList = append(curseModList, mod)
						break
					}
				}
			}
			if tag.ApplyToPlayer {
				if curseDB.Sum(modparser.Base, nil, "AvoidCurse") < 100 {
					curseDB.Conditions.Set("Cursed", true)
					curseDB.Multipliers["CurseOnSelf"] = curseDB.Multipliers["CurseOnSelf"] + 1
					curseDB.Conditions.Set("AffectedBy"+noSpace(grantedEffect.Name), true)
					cfg := &modstore.Cfg{SkillName: grantedEffect.Name}
					inc := curseDB.Sum(modparser.Inc, cfg, "CurseEffectOnSelf") + gemModList.Sum(modparser.Inc, nil, "CurseEffectAgainstPlayer")
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
				curse.modList.ScaleAddList(curseModList, (1+enemyDB.Sum(modparser.Inc, nil, "CurseEffectOnSelf")/100)*enemyDB.More(nil, "CurseEffectOnSelf"), false)
				*ecd.dest = append(*ecd.dest, curse)
			}
		}
	}
	// ally curses: party tab empty

	// Set curse limit
	if modDB.Flag(nil, "CurseLimitIsMaximumPowerCharges") {
		output.SetN("EnemyCurseLimit", output.N("PowerChargesMax"))
	} else {
		output.SetN("EnemyCurseLimit", modDB.Sum(modparser.Base, nil, "EnemyCurseLimit"))
	}
	cursesLimit := output.N("EnemyCurseLimit")

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
					if len(activeSkill.BuffListTyped) > 0 && curse.name == activeSkill.BuffListTyped[0].Name && activeSkill.SkillTypes[modparser.SkillTypeAura] {
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
				socketedCursesHexLimitValue := modDB.Sum(modparser.Base, nil, "SocketedCursesHexLimitValue")
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
		modDB.Conditions.Set("AffectedByNonVaalGuardSkill", true)
	}
	for _, guard := range guardSlots {
		modDB.Conditions.Set("AffectedByGuardSkill", true)
		modDB.Conditions.Set("AffectedBy"+noSpace(guard.name), true)
		mergeBuff(guard.modList.Mods, buffs, guard.name)
	}
	output.SetFlag("GuardSkillActive", modDB.Conditions.Get("AffectedByGuardSkill"))

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
		enemyDB.Conditions.Set("Cursed", true)
		if slot.isMark {
			enemyDB.Conditions.Set("Marked", true)
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
	maxImpaleStacks := modDB.Sum(modparser.Base, nil, "ImpaleStacksMax") * (1 + modDB.Sum(modparser.Base, nil, "ImpaleAdditionalDurationChance")/100)
	if !enemyDB.HasMod(modparser.Base, nil, "Multiplier:ImpaleStacks") {
		enemyDB.AddMod(newModS("Multiplier:ImpaleStacks", modparser.Base, modparser.Num(maxImpaleStacks), "Config", &modparser.CondTag{Var: "Combat"}))
	} else if enemyDB.Sum(modparser.Base, nil, "Multiplier:ImpaleStacks") > maxImpaleStacks {
		enemyDB.ReplaceMod(newModS("Multiplier:ImpaleStacks", modparser.Base, modparser.Num(maxImpaleStacks), "Config", &modparser.CondTag{Var: "Combat"}))
	}

	// Foulborn Choir of the Storm: needs to run after the main auras (in
	// case of Purity of Lightning / Elements) but before extra auras
	// (Radiant Faith). Resistances run on a CHILD ModDB so the conversion
	// pass cannot mutate the player's own mods; the output map is shared, so
	// the resist numbers land where the mana re-derivation reads them.
	if modDB.Flag(nil, "ManaIncreasedByOvercappedLightningRes") {
		tempDB := modstore.NewDB(modDB)
		tempActor := &performActor{
			db:        tempDB,
			output:    env.playerPA.output,
			mainSkill: env.playerPA.mainSkill,
			skills:    env.playerPA.skills,
			enemy:     env.playerPA.enemy,
			ms:        env.Player,
		}
		tempDB.Actor = env.Player
		env.Resistances(tempActor)
		// Re-derive life/mana and the reservations now that overcapped
		// lightning resistance feeds increased mana.
		env.doActorLifeMana(env.playerPA)
		env.doActorLifeManaReservation(env.playerPA, true)
	}

	if modDB.Flag(env.PlayerMainSkill.SkillCfg, "Condition:CanInflictHallowingFlame") {
		magnitude := 0.0
		if ov, ok := modDB.Override(nil, "HallowingFlameMagnitude"); ok {
			magnitude = valueNum(ov)
		} else {
			magnitude = modDB.Sum(modparser.Inc, nil, "HallowingFlameMagnitude")
		}
		val := math.Floor(25 * (1 + magnitude/100))
		modDB.AddMod(newMod("ExtraAura", modparser.List, modparser.ModRef{Mod: newModS("PhysicalDamageGainAsFire", modparser.Base, modparser.Num(val), "Hallowing Flame", &modparser.GlobalEffectTag{EffectType: "Global", Unscalable: true}, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "HallowingFlame"}, &modparser.MultiplierTag{Var: "HallowingFlame", Actor: "enemy"})}))
	}

	// Check for extra auras
	for _, v := range modDB.List(nil, "ExtraAura") {
		tag, ok := v.(modparser.ModRef)
		if !ok || tag.Mod == nil {
			continue
		}
		mod := tag.Mod
		modList := []*modparser.Mod{mod}
		if !tag.OnlyAllies && !(tag.FromAllies && modDB.Flag(nil, "AlliesAurasCannotAffectSelf")) {
			inc := modDB.Sum(modparser.Inc, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			more := modDB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
			modDB.ScaleAddList(modList, (1+inc/100)*more, false)
			// notBuff is never set on an ExtraAura record.
			modDB.Multipliers["BuffOnSelf"] = modDB.Multipliers["BuffOnSelf"] + 1
		}
		if tag.FromAllies || !modDB.Flag(nil, "SelfAurasCannotAffectAllies") {
			if env.Minion != nil {
				inc := env.Minion.DB.Sum(modparser.Inc, nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				more := env.Minion.DB.More(nil, "BuffEffectOnSelf", "AuraEffectOnSelf")
				env.Minion.DB.ScaleAddList(modList, (1+inc/100)*more, false)
			}
			totemModBlacklist := mod.Name == "Speed" || mod.Name == "CritMultiplier" || mod.Name == "CritChance"
			if env.PlayerMainSkill.SkillFlags["totem"] && !totemModBlacklist {
				totemMod := cloneMod(mod)
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
		env.Player.WeaponData1 = weaponRef(unarmedWeapon(data.UnarmedWeaponData[int(env.Build.ClassID)]))
		modDB.Conditions.Set("Unarmed", true)
		// env.player.Gloves is never set in the reference, so this
		// branch always marks Unencumbered
		modDB.Conditions.Set("Unencumbered", true)
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
	output := env.playerPA.output

	type ailmentDef struct {
		condition string
		mods      func(num float64) []*modparser.Mod
	}
	shockStacksMax := func() float64 {
		if ov, ok := modDB.Override(nil, "ShockStacksMax"); ok {
			return valueNum(ov)
		}
		return modDB.Sum(modparser.Base, nil, "ShockStacksMax")
	}
	scorchStacksMax := func() float64 {
		if ov, ok := modDB.Override(nil, "ScorchStacksMax"); ok {
			return valueNum(ov)
		}
		return modDB.Sum(modparser.Base, nil, "ScorchStacksMax")
	}
	ailments := map[string]ailmentDef{
		"Chill": {"Chilled", func(num float64) []*modparser.Mod {
			mods := []*modparser.Mod{newModS("ActionSpeed", modparser.Inc, modparser.Num(-num), "Chill", &modparser.CondTag{Var: "Chilled"})}
			if modDB.Flag(nil, "ChillEffectIncDamageTaken") {
				mods = append(mods, newModS("DamageTaken", modparser.Inc, modparser.Num(num), "Ahuana's Bite", &modparser.CondTag{Var: "Chilled"}))
			} else if modDB.Flag(nil, "ChillEffectIncColdDamageTaken") {
				mods = append(mods, newModS("ColdDamageTaken", modparser.Inc, modparser.Num(num), "Chilled by Hits", &modparser.CondTag{Var: "Chilled"}))
			} else if modDB.Flag(nil, "ChillingAreaIncColdDamageTaken") {
				mods = append(mods, newModS("ColdDamageTaken", modparser.Inc, modparser.Num(num), "Chilling Area", &modparser.CondTag{Var: "Chilled"}))
			} else if output.Flag("HasBonechill") && (hasGuaranteedBonechill || enemyDB.Sum(modparser.Base, nil, "ChillVal") > 0) {
				mods = append(mods, newModS("ColdDamageTaken", modparser.Inc, modparser.Num(num), "Bonechill", &modparser.CondTag{Var: "Chilled"}))
			}
			if modDB.Flag(nil, "ChillEffectLessDamageDealt") {
				mods = append(mods, newModS("Damage", modparser.More, modparser.Num(-num/2), "Shaper of Winter", &modparser.CondTag{Var: "Chilled"}))
			}
			return mods
		}},
		"Shock": {"Shocked", func(num float64) []*modparser.Mod {
			var mods []*modparser.Mod
			if modDB.Flag(nil, "ShockCanStack") {
				mods = append(mods, newModS("DamageTaken", modparser.Inc, modparser.Num(num), "Shock", &modparser.CondTag{Var: "Shocked"}, &modparser.MultiplierTag{Var: "ShockStacks", Limit: opt(shockStacksMax())}))
				output.SetN("CurrentShock", num*math.Min(enemyDB.Sum(modparser.Base, nil, "Multiplier:ShockStacks"), shockStacksMax()))
			} else {
				mods = append(mods, newModS("DamageTaken", modparser.Inc, modparser.Num(num), "Shock", &modparser.CondTag{Var: "Shocked"}))
			}
			return mods
		}},
		"Scorch": {"Scorched", func(num float64) []*modparser.Mod {
			var mods []*modparser.Mod
			if modDB.Flag(nil, "ScorchCanStack") {
				mods = append(mods, newModS("ElementalResist", modparser.Base, modparser.Num(-num), "Scorch", &modparser.CondTag{Var: "Scorched"}, &modparser.MultiplierTag{Var: "ScorchStacks", Limit: opt(scorchStacksMax())}))
				output.SetN("CurrentScorch", num*math.Min(enemyDB.Sum(modparser.Base, nil, "Multiplier:ScorchStacks"), scorchStacksMax()))
			} else {
				mods = append(mods, newModS("ElementalResist", modparser.Base, modparser.Num(-num), "Scorch", &modparser.CondTag{Var: "Scorched"}))
			}
			return mods
		}},
		"Brittle": {"Brittle", func(num float64) []*modparser.Mod {
			return []*modparser.Mod{newModS("SelfCritChance", modparser.Base, modparser.Num(num), "Brittle", &modparser.CondTag{Var: "Brittle"})}
		}},
		"Sap": {"Sapped", func(num float64) []*modparser.Mod {
			return []*modparser.Mod{newModS("Damage", modparser.More, modparser.Num(-num), "Sap", &modparser.CondTag{Var: "Sapped"})}
		}},
	}

	names := make([]string, 0, len(ailments))
	for name := range ailments {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, ailment := range names {
		val := ailments[ailment]
		// The reference's first clause (`Val > 0 or Sum(...)`) is
		// always truthy: the Sum returns a number and 0 is truthy in Lua
		if !(enemyDB.Flag(nil, "Condition:Already"+val.condition) || enemyDB.Flag(nil, ailment+"Immune", "ElementalAilmentImmune") || enemyDB.Sum(modparser.Base, nil, "Avoid"+ailment, "AvoidAilments", "AvoidElementalAilments") >= 100) {
			override := 0.0
			minimum := 0.0
			for _, value := range modDB.Tabulate(modparser.Base, nil, ailment+"Base", ailment+"Override", ailment+"Minimum") {
				mod := value.Mod
				effect := valueNum(mod.Value)
				scalesWithSource := mod.Name == ailment+"Base" || mod.Name == ailment+"Minimum"
				if mod.Name == ailment+"Override" {
					enemyDB.AddMod(newModS("Condition:"+val.condition, modparser.Flag, modparser.Bool(true), mod.Source))
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
					modDB.AddMod(modparser.NewModFull(ailment+"Override", modparser.Base, modparser.Num(effect), mod.Source, mod.SourceSet, mod.Flags, mod.KeywordFlags, mod.Tags...))
					if mod.Name == ailment+"Minimum" {
						minimum += effect
					}
				}
				override = math.Max(math.Max(override, effect), minimum)
			}
			maxAilment := 0.0
			if ov, ok := modDB.Override(nil, ailment+"Max"); ok {
				maxAilment = valueNum(ov)
			} else {
				for _, skill := range env.PlayerActiveSkills {
					skillMax := data.NonDamagingAilment[ailment].Max + skill.BaseSkillModList.Sum(modparser.Base, nil, ailment+"Max")
					if skillMax > maxAilment {
						maxAilment = skillMax
					}
				}
			}
			output.SetN("Maximum"+ailment, maxAilment)
			prec := math.Pow(10, data.NonDamagingAilment[ailment].Precision)
			output.SetN("Current"+ailment, math.Floor(math.Min(math.Max(override, enemyDB.Sum(modparser.Base, nil, ailment+"Val")), maxAilment)*prec)/prec)
			for _, mod := range val.mods(output.N("Current" + ailment)) {
				enemyDB.AddMod(mod)
			}
			enemyDB.AddMod(newMod("Condition:Already"+val.condition, modparser.Flag, modparser.Bool(true), &modparser.CondTag{Var: val.condition}))
		}
	}

	// Update chill and shock multipliers
	chillEffectMultiplier := enemyDB.Sum(modparser.Base, nil, "Multiplier:ChillEffect")
	if _, ok := output["CurrentChill"]; ok && chillEffectMultiplier < output.N("CurrentChill") {
		enemyDB.AddMod(newModS("Multiplier:ChillEffect", modparser.Base, modparser.Num(output.N("CurrentChill")-chillEffectMultiplier), ""))
	}
	shockEffectMultiplier := enemyDB.Sum(modparser.Base, nil, "Multiplier:ShockEffect")
	if _, ok := output["CurrentShock"]; ok && shockEffectMultiplier < output.N("CurrentShock") {
		enemyDB.AddMod(newModS("Multiplier:ShockEffect", modparser.Base, modparser.Num(output.N("CurrentShock")-shockEffectMultiplier), ""))
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
			for _, entry := range enemyDB.Tabulate(modparser.Base, nil, element+"Exposure") {
				if valueNum(entry.Value) < min {
					min = valueNum(entry.Value)
					source = entry.Mod.Source
				}
			}
			if !math.IsInf(min, 1) {
				// Modify the magnitude of all exposures
				for _, entry := range modDB.Tabulate(modparser.Base, nil, "ExtraExposure", "Extra"+element+"Exposure") {
					min += valueNum(entry.Value)
				}
				// `m_min(min, modDB:Override(nil, "ExposureMin"))`: Override
				// returns NO VALUES when nothing matches (ModDB.lua:219 falls
				// off the end), not nil — so the call collapses to the
				// one-argument m_min(min), which is just min.
				resist := min
				if exposureMin, ok := modDB.Override(nil, "ExposureMin"); ok {
					resist = math.Min(min, valueNum(exposureMin))
				}
				enemyDB.AddMod(newModS("Condition:Has"+element+"Exposure", modparser.Flag, modparser.Bool(true), ""))
				enemyDB.AddMod(newModS(element+"Resist", modparser.Base, modparser.Num(resist), source))
				modDB.AddMod(newModS("Condition:AppliedExposureRecently", modparser.Flag, modparser.Bool(true), ""))
			}
		}
	}

	// Handle consecrated ground effects on enemies
	if enemyDB.Flag(nil, "Condition:OnConsecratedGround") {
		effect := 1 + modDB.Sum(modparser.Inc, nil, "ConsecratedGroundEffect")/100
		enemyDB.AddMod(newModS("DamageTaken", modparser.Inc, modparser.Num(math.Floor(enemyDB.Sum(modparser.Inc, nil, "DamageTakenConsecratedGround")*effect)), "Consecrated Ground"))
	}

	// Full DPS and builds without a supported redirect do not need ally Life.
	allyLifeRedirects := []string{
		"takenFromSpectresBeforeYou", "takenFromMinionBeforeYou",
		"takenFromRadianceSentinelBeforeYou", "takenFromVoidSpawnBeforeYou", "takenFromStoneGolemBeforeYou",
		"takenFromTotemsBeforeYou", "takenFromVaalRejuvenationTotemsBeforeYou",
	}
	if modDB.Sum(modparser.Base, nil, allyLifeRedirects...) != 0 {
		env.performAllyLife()
	}
}

// performGemLevel ports the main skill's gem level/quality block
// (CalcPerform.lua L3862-3916). It sits AFTER the defence/offence handoff
// in the reference, which is invisible while the dump stubs that handoff
// out but matters to PerformFull, so it is its own step.
func (env *Env) performGemLevel() {
	output := env.playerPA.output
	mainSkill := env.PlayerMainSkill
	if mainSkill != nil && mainSkill.ActiveEffect != nil && mainSkill.ActiveEffect.SrcInstance != nil {
		baseLevel := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemLevel")
		totalItemLevel := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemItemLevel")
		totalSupportLevel := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemSupportLevel")
		output.SetFlag("GemHasLevel", true)
		output.SetN("GemLevel", baseLevel+totalSupportLevel+totalItemLevel)

		baseQuality := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemQuality")
		totalItemQuality := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemItemQuality")
		totalSupportQuality := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemSupportQuality")
		socketQuality := mainSkill.SkillModList.Sum(modparser.Base, mainSkill.SkillCfg, "GemSocketQuality")
		output.SetFlag("GemHasQuality", true)
		output.SetN("GemQuality", baseQuality+totalSupportQuality+totalItemQuality+socketQuality)
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
// L3599-3718).
func (env *Env) performAllyLife() {
	modDB := env.ModDB
	output := env.playerPA.output

	calculatedLifePool := map[string]bool{}
	spectreCount := 0.0
	firstSpectreLife := 0.0
	haveFirstSpectre := false
	spectreLimit := output.N("ActiveSpectreLimit")
	if ov := env.ConfigInput.MultiplierSummonedMinion; ov != 0 {
		spectreLimit = math.Min(spectreLimit, ov)
	}
	for _, activeSkill := range env.PlayerActiveSkills {
		minion := activeSkill.Minion
		if minion != nil && !activeSkill.SkillFlags["disable"] && !activeSkill.SkillTypes[modparser.SkillTypeMinionsAreUndamagable] {
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
				if skillPool != nil && modDB.Sum(modparser.Base, nil, skillPool.redirect) != 0 && modDB.Sum(modparser.Base, nil, companionshipLifePool.redirect) != 0 {
					// Both redirects spend the same minion's Life, so they must share one pool.
					pools = []*lifePool{companionshipLifePool}
					modDB.AddMod(newModS("MinionLifeShares"+skillPool.life, modparser.Flag, modparser.Bool(true), "Companionship"))
				}
			}

			minionLife := 0.0
			haveMinionLife := false
			for _, pool := range pools {
				canAdd := ((pool == spectreLifePool && spectreCount < spectreLimit) ||
					(pool != spectreLifePool && !calculatedLifePool[pool.life])) &&
					modDB.Sum(modparser.Base, nil, pool.redirect) != 0 && !hasOverride(modDB, nil, pool.life)
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
								name, ok := v.(modparser.Str)
								if !ok {
									continue
								}
								if mods, ok := env.Build.Spec.KeystoneMap[string(name)]; ok {
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
						minionLife = minion.Output.N("Life")
						haveMinionLife = true
					}

					// Void Spawn redirects scale from their summon limit, so
					// their Life pool uses that same limit.
					count := 1.0
					if pool.limit != "" {
						if output.Flag(pool.limit) {
							count = output.N(pool.limit)
						} else {
							count = Val(activeSkill.SkillModList, pool.limit, nil)
						}
					}
					if pool == spectreLifePool {
						count = 1
						spectreCount++
					}
					life := minionLife * count
					modDB.AddMod(newModS(pool.life, modparser.Base, modparser.Num(life), pool.source))
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
		modDB.AddMod(newModS(spectreLifePool.life, modparser.Base, modparser.Num(firstSpectreLife*(spectreLimit-spectreCount)), spectreLifePool.source))
	}

	var normalTotem, vaalRejuvenationTotem *allyTotem
	for _, activeSkill := range env.PlayerActiveSkills {
		if activeSkill.SkillFlags["totem"] && !activeSkill.SkillFlags["disable"] && activeSkill.SkillData.Has("totemLevel") && activeSkill.SkillTotemId != nil {
			life, _ := env.calcTotemLife(activeSkill)
			totem := &allyTotem{name: activeSkill.ActiveEffect.GrantedEffect.Name, life: life}
			if totem.name == "Vaal Rejuvenation Totem" {
				vaalRejuvenationTotem = totem
			} else if normalTotem == nil || life > normalTotem.life {
				normalTotem = totem
			}
		}
	}

	// PoB cannot know which Totem is nearest, so use the eligible type with
	// the most Life.
	totemLifeOverride, _ := modDB.Override(nil, "TotalTotemLife")
	totemRedirect := modDB.Sum(modparser.Base, nil, "takenFromTotemsBeforeYou")
	nearestIsVaal := vaalRejuvenationTotem != nil &&
		(normalTotem == nil || vaalRejuvenationTotem.life > normalTotem.life) && !modparser.Truthy(totemLifeOverride)
	nearestTotem := normalTotem
	if nearestIsVaal {
		nearestTotem = vaalRejuvenationTotem
	}
	if nearestTotem != nil && !nearestIsVaal && totemRedirect != 0 && !modparser.Truthy(totemLifeOverride) {
		modDB.AddMod(newModS("TotalTotemLife", modparser.Base, modparser.Num(nearestTotem.life), "Totem"))
	}
	if vaalRejuvenationTotem != nil &&
		(modDB.Sum(modparser.Base, nil, "takenFromVaalRejuvenationTotemsBeforeYou") != 0 || (nearestIsVaal && totemRedirect != 0)) &&
		!hasOverride(modDB, nil, "TotalVaalRejuvenationTotemLife") {
		modDB.AddMod(newModS("TotalVaalRejuvenationTotemLife", modparser.Base, modparser.Num(vaalRejuvenationTotem.life), "Vaal Rejuvenation Totem"))
	}
}

// allyTotem is one candidate totem for the ally-life pools.
type allyTotem struct {
	name string
	life float64
}

// calcTotemLife ports calcs.calcTotemLife (CalcOffence.lua L316).
func (env *Env) calcTotemLife(activeSkill *ActiveSkill) (life, lifeMod float64) {
	lifeMod = Mod(activeSkill.SkillModList, activeSkill.SkillCfg, "TotemLife")
	totemLevel := int(activeSkill.SkillData.N("totemLevel"))
	mult := data.TotemLifeMult[int64(*activeSkill.SkillTotemId)]
	life = util.RoundHalfUp(math.Floor(data.MonsterAllyLifeTable[totemLevel-1]*mult)*lifeMod, 0)
	return life, lifeMod
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

// copyModForTagWrite shallow-copies a mod with a fresh Tags slice and a
// fresh first tag map, so a tag write cannot reach the loaded data the mod
// came from. Everything else stays shared — the write is the only mutation.
func copyModForTagWrite(m *modparser.Mod) *modparser.Mod {
	c := *m
	c.Tags = append([]modparser.Tag(nil), m.Tags...)
	if c.Tags[0] != nil {
		c.Tags[0] = c.Tags[0].Clone()
	}
	return &c
}

// cachedOutputs is what getCachedOutputValue (CalcPerform L30) hands the
// stage-count skills: values from the skill's own cached solo build.
type cachedOutputs struct {
	Speed, HitSpeed, Duration util.Opt[float64]
}

// cachedOutputValues builds the skill's solo cache on a miss (the {uuid}
// limited flag stops the nested build's stage loop from recursing back
// into this skill) and reads the stage-count outputs from it.
func (env *Env) cachedOutputValues(activeSkill *ActiveSkill) cachedOutputs {
	uuid := env.cacheSkillUUID(activeSkill)
	if env.GlobalCache[uuid] == nil || env.Mode == ModeCalculator {
		env.BuildActiveSkill(env.Mode, activeSkill, uuid, uuid)
	}
	c := env.GlobalCache[uuid]
	read := func(k string) util.Opt[float64] {
		if v := c.out(k); v.Kind != modstore.OutAbsent {
			return util.Some(v.Num())
		}
		return util.Opt[float64]{}
	}
	return cachedOutputs{Speed: read("Speed"), HitSpeed: read("HitSpeed"), Duration: read("Duration")}
}
