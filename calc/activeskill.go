// Port of CalcActiveSkill.lua's createActiveSkill and the calcLib support
// check it needs, plus CalcSetup's applyGemMods/applySocketMods/
// addBestSupport helpers. buildActiveSkillModList is the next stage.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// ActiveSkill mirrors the Lua active skill table (the slice the ported
// stages populate).
type ActiveSkill struct {
	ActiveEffect     *ActiveEffect
	SupportList      []*ActiveEffect
	Actor            *modstore.Actor
	SummonSkill      *ActiveSkill
	SocketGroup      *SocketGroupInput
	SkillData        *SkillData
	SkillTypes       map[modparser.SkillTypeID]bool
	MinionSkillTypes map[modparser.SkillTypeID]bool
	SkillFlags       map[string]bool
	EffectList       []*ActiveEffect
	SlotName         string

	// buildActiveSkillModList state
	SkillPart         util.Opt[float64] // set when multipart
	SkillPartName     string
	DisableReason     string
	Weapon1Flags      *modparser.ModFlag
	Weapon2Flags      *modparser.ModFlag
	SkillCfg          *modstore.Cfg
	Weapon1Cfg        *modstore.Cfg
	Weapon2Cfg        *modstore.Cfg
	SkillModList      *modstore.List
	BaseSkillModList  *modstore.List
	ExtraSkillModList []*modparser.Mod
	SkillTotemId      *float64
	TriggeredBy       *ActiveEffect
	ActiveMineCount   *float64
	ActiveStageCount  *float64
	BuffListTyped     []*Buff // buffList
	MinionList        []string
	Minion            *Minion

	// offence-stage ailment / damage-over-time configurations
	BleedCfg, OHBleedCfg   *modstore.Cfg
	PoisonCfg, OHPoisonCfg *modstore.Cfg
	IgniteCfg, OHIgniteCfg *modstore.Cfg
	DecayCfg               *modstore.Cfg
	DotCfg                 *modstore.Cfg
	DotTypeCfg             map[string]*modstore.Cfg

	// Mirage is what CalcMirages hangs off the main skill: the sub-skill it
	// built, how many of it, and the sub-environment's output.
	Mirage       *MirageResult
	InfoMessage  string
	InfoMessage2 string

	// perform-stage marks
	BuffSkill       bool
	MinionBuffSkill bool
	TotemBuffSkill  bool
	DebuffSkill     bool
}

// geFromItem reads grantedEffect.fromItem: the template key, or the
// runtime mark addExtraSupports sets. The reference mutates the shared
// skill table; the replay keeps the mark per-Env so variants stay
// independent (revisit if a dump leaks the mutation across variants).
func (env *Env) geFromItem(ge *data.GrantedEffect) bool {
	if env.geFromItemMark[ge] {
		return true
	}
	return ge.Custom.FromItem
}

// CanGrantedEffectSupportActiveSkill ports the calcLib check (deferred
// from tools.go until ActiveSkill existed). The reference's first test,
// `grantedEffect.unsupported`, is vestigial: no template sets the key.
func (env *Env) canGrantedEffectSupportActiveSkill(grantedEffect *data.GrantedEffect, activeSkill *ActiveSkill, imbuedSupport bool) bool {
	if activeSkill.ActiveEffect.GrantedEffect.CannotBeSupported {
		return false
	}
	if grantedEffect.SupportGemsOnly && activeSkill.ActiveEffect.GemData == nil {
		return false
	}

	// Forbidden Shako / Hungry Loop style: item-granted supports cannot
	// support item-granted skills
	if env.geFromItem(grantedEffect) && grantedEffect.Support {
		ae := activeSkill.ActiveEffect
		srcFromItem := ae.SrcInstance != nil && ae.SrcInstance.FromItem
		if env.geFromItem(ae.GrantedEffect) || len(ae.GrantedEffect.ModSource) >= 4 && ae.GrantedEffect.ModSource[:4] == "Item" || srcFromItem {
			return false
		}
	}

	var effectiveSkillTypes map[modparser.SkillTypeID]bool
	var effectiveMinionTypes map[modparser.SkillTypeID]bool
	if imbuedSupport {
		// Use the skillTypes from the gem so it ignores any support added types
		if activeSkill.SummonSkill != nil {
			effectiveSkillTypes = activeSkill.SummonSkill.ActiveEffect.GrantedEffect.SkillTypes
		} else {
			effectiveSkillTypes = activeSkill.ActiveEffect.GrantedEffect.SkillTypes
		}
		if !grantedEffect.IgnoreMinionTypes {
			if activeSkill.SummonSkill != nil {
				effectiveMinionTypes = activeSkill.SummonSkill.ActiveEffect.GrantedEffect.MinionSkillTypes
			} else {
				effectiveMinionTypes = activeSkill.ActiveEffect.GrantedEffect.MinionSkillTypes
			}
		}
	} else {
		if activeSkill.SummonSkill != nil {
			effectiveSkillTypes = activeSkill.SummonSkill.SkillTypes
		} else {
			effectiveSkillTypes = activeSkill.SkillTypes
		}
		if !grantedEffect.IgnoreMinionTypes {
			if activeSkill.SummonSkill != nil {
				effectiveMinionTypes = activeSkill.SummonSkill.MinionSkillTypes
			} else {
				effectiveMinionTypes = activeSkill.MinionSkillTypes
			}
		}
	}

	if len(grantedEffect.ExcludeSkillTypes) > 0 && grantedEffect.ExcludeSkillTypes[0] != 0 &&
		DoesTypeExpressionMatch(grantedEffect.ExcludeSkillTypes, effectiveSkillTypes, nil) {
		return false
	}
	if grantedEffect.IsTrigger && activeSkill.Actor.Enemy.Player != activeSkill.Actor {
		return false
	}
	// Sacred Wisps / Varunastra weapon type matching
	actorHasAllOneHand := (activeSkill.Actor.WeaponData1 != nil && activeSkill.Actor.WeaponData1.CountsAsAll1H()) ||
		(activeSkill.Actor.WeaponData2 != nil && activeSkill.Actor.WeaponData2.CountsAsAll1H())
	if grantedEffect.WeaponTypes != nil {
		activeTypeLookup := map[string]bool{}
		for activeType := range activeSkill.ActiveEffect.GrantedEffect.WeaponTypes {
			activeTypeLookup[activeType] = true
		}
		if len(activeTypeLookup) == 0 {
			return false
		}
		if actorHasAllOneHand {
			activeTypeLookup["Claw"] = true
			activeTypeLookup["Dagger"] = true
			activeTypeLookup["One Handed Axe"] = true
			activeTypeLookup["One Handed Mace"] = true
			activeTypeLookup["One Handed Sword"] = true
		}
		typeMatch := false
		for grantedType := range grantedEffect.WeaponTypes {
			if activeTypeLookup[grantedType] {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}
	if len(grantedEffect.RequireSkillTypes) > 0 && grantedEffect.RequireSkillTypes[0] != 0 {
		return DoesTypeExpressionMatch(grantedEffect.RequireSkillTypes, effectiveSkillTypes, effectiveMinionTypes)
	}
	return true
}

// createActiveSkill ports calcs.createActiveSkill.
func (env *Env) createActiveSkill(activeEffect *ActiveEffect, supportList []*ActiveEffect, actor *modstore.Actor, socketGroup *SocketGroupInput, summonSkill *ActiveSkill) *ActiveSkill {
	activeSkill := &ActiveSkill{
		ActiveEffect: activeEffect,
		SupportList:  supportList,
		Actor:        actor,
		SummonSkill:  summonSkill,
		SocketGroup:  socketGroup,
		SkillData:    newSkillData(),
	}

	activeGrantedEffect := activeEffect.GrantedEffect

	// Initialise skill types
	activeSkill.SkillTypes = map[modparser.SkillTypeID]bool{}
	for k, v := range activeGrantedEffect.SkillTypes {
		activeSkill.SkillTypes[k] = v
	}
	if activeGrantedEffect.MinionSkillTypes != nil {
		activeSkill.MinionSkillTypes = map[modparser.SkillTypeID]bool{}
		for k, v := range activeGrantedEffect.MinionSkillTypes {
			activeSkill.MinionSkillTypes[k] = v
		}
	}

	// Initialise skill flag set ('attack', 'projectile', etc)
	skillFlags := map[string]bool{}
	for k, v := range activeGrantedEffect.BaseFlags {
		skillFlags[k] = v
	}
	activeSkill.SkillFlags = skillFlags
	// hit = hit or Attack or Damage or Projectile: all-nil stays absent
	if !skillFlags["hit"] && (activeSkill.SkillTypes[modparser.SkillTypeAttack] ||
		activeSkill.SkillTypes[modparser.SkillTypeDamage] ||
		activeSkill.SkillTypes[modparser.SkillTypeProjectile]) {
		skillFlags["hit"] = true
	}

	// Process support skills
	activeSkill.EffectList = []*ActiveEffect{activeEffect}
	// rejectedSupportsIndices with Lua array-hole semantics: removing an
	// entry leaves a nil hole, and the next repeat pass's ipairs stops there.
	var rejected []*ActiveEffect

	for _, supportEffect := range supportList {
		// Pass 1: Add skill types from compatible supports
		if supportEffect.GrantedEffect.Support {
			if env.canGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill, false) {
				for _, id := range supportEffect.GrantedEffect.AddSkillTypes {
					if id != 0 {
						activeSkill.SkillTypes[id] = true
					}
				}
			}
		} else {
			rejected = append(rejected, supportEffect)
		}
	}

	for {
		notAddedNewSupport := true
		for i := 0; i < len(rejected) && rejected[i] != nil; i++ {
			supportEffect := rejected[i]
			if supportEffect.GrantedEffect.Support {
				if env.canGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill, false) {
					notAddedNewSupport = false
					rejected[i] = nil
					for _, id := range supportEffect.GrantedEffect.AddSkillTypes {
						if id != 0 {
							activeSkill.SkillTypes[id] = true
						}
					}
				}
			}
		}
		if notAddedNewSupport {
			break
		}
	}

	for _, supportEffect := range supportList {
		// Pass 2: Add all compatible supports
		if supportEffect.GrantedEffect.Support {
			if env.canGrantedEffectSupportActiveSkill(supportEffect.GrantedEffect, activeSkill, false) {
				activeSkill.EffectList = append(activeSkill.EffectList, supportEffect)
				if supportEffect.IsSupporting != nil && activeEffect.SrcInstance != nil {
					supportEffect.IsSupporting[activeEffect.SrcInstance] = true
				}
				if summonSkill == nil {
					for k := range supportEffect.GrantedEffect.Custom.AddFlags {
						skillFlags[k] = true
					}
				}
			}
		}
	}

	return activeSkill
}

// applyGemMods ports CalcSetup's applyGemMods over a GemProperty Tabulate
// result. (effect.gemPropertyInfo is tooltip-only and skipped.)
func (env *Env) applyGemMods(effect *ActiveEffect, modList []modstore.TabEntry) {
	for _, entry := range modList {
		value, ok := entry.Value.(modparser.GemPropertyRef)
		if !ok {
			panic("calc: non-GemPropertyRef value in GemProperty list (the Lua errors)")
		}
		match := true
		if value.KeywordList != nil {
			for _, kw := range value.KeywordList {
				if !GemIsType(effect.GemData, kw, true) {
					match = false
					break
				}
			}
		} else if !GemIsType(effect.GemData, value.Keyword, true) {
			match = false
		}
		if match {
			key := value.Key
			v := value.Value.Or(0)
			if key == "quality" {
				if modHasSocketedInTag(entry.Mod) {
					effect.ItemQuality += v
				} else {
					effect.GlobalQuality += v
				}
			}
			switch key {
			case "level":
				effect.Level += v
			case "quality":
				effect.Quality += v
			case "req":
				cur := 0.0
				if effect.Req != nil {
					cur = *effect.Req
				}
				n := cur + v
				effect.Req = &n
			default:
				if effect.Extra == nil {
					effect.Extra = map[string]float64{}
				}
				effect.Extra[key] += v
			}
			// save for buildActiveSkillModList's GemItem mods
			effect.GemPropertyInfo = append(effect.GemPropertyInfo, entry)
		}
	}
}

// applySocketMods ports CalcSetup's applySocketMods.
func (env *Env) applySocketMods(gem *data.Gem, groupCfg *modstore.Cfg, socketNum int, modSource string) {
	socketCfg := *groupCfg
	socketCfg.SkillGem = gemRef{gem}
	sn := float64(socketNum)
	socketCfg.SocketNum = &sn
	for _, v := range env.ModDB.List(&socketCfg, "SocketProperty") {
		ref, ok := v.(modparser.PropertyModRef)
		if !ok || ref.Mod == nil {
			continue
		}
		mod := ref.Mod
		src := modSource
		if src == "" {
			src = groupCfg.SlotName
		}
		env.ModDB.AddMod(modparser.SetSource(mod, src))
	}
}

// addBestSupport ports CalcSetup's addBestSupport.
func addBestSupport(supportEffect *ActiveEffect, appliedSupportList *[]*ActiveEffect, mode CalcMode) {
	add := true
	for index, otherSupport := range *appliedSupportList {
		// Check if there's another better support already present
		if supportEffect.GrantedEffect == otherSupport.GrantedEffect {
			add = false
			if supportEffect.Level > otherSupport.Level || (supportEffect.Level == otherSupport.Level && supportEffect.Quality > otherSupport.Quality) {
				if mode == ModeMain {
					otherSupport.Superseded = true
				}
				(*appliedSupportList)[index] = supportEffect
			} else {
				supportEffect.Superseded = true
			}
			break
		} else if supportEffect.GrantedEffect.PlusVersionOf != nil && *supportEffect.GrantedEffect.PlusVersionOf == otherSupport.GrantedEffect.Id {
			add = false
			if mode == ModeMain {
				otherSupport.Superseded = true
			}
			(*appliedSupportList)[index] = supportEffect
		} else if otherSupport.GrantedEffect.PlusVersionOf != nil && *otherSupport.GrantedEffect.PlusVersionOf == supportEffect.GrantedEffect.Id {
			add = false
			supportEffect.Superseded = true
		}
	}
	if add {
		*appliedSupportList = append(*appliedSupportList, supportEffect)
	}
}
