// Port of .archive/src/Modules/CalcTriggers.lua: the trigger-rate stage that
// runs between the EHP stage and calcs.offence. Its one external input is
// GlobalCache (see calc/globalcache.go).
//
// The reference dispatches through a ~90-entry configTable of unique items,
// support gems and skills. Every key is present here so the dispatch itself
// is faithful; the entries no corpus build reaches panic rather than guess,
// and the same goes for the branches of defaultTriggerHandler that need
// inputs the corpus never produces.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// triggerConfig is the config table defaultTriggerHandler consumes.
type triggerConfig struct {
	actor       *performActor
	source      *ActiveSkill
	triggerName string
	sourceName  string
	// customTriggerName replaces the whole infoMessage prefix.
	customTriggerName string
	trigRate          *float64
	// uuid pre-resolved by a config that ran findTriggerSkill itself.
	uuid          string
	triggerChance *float64

	triggeredSkills []*simSkill

	triggerSkillCond   func(env *Env, skill *ActiveSkill) bool
	triggeredSkillCond func(env *Env, skill *ActiveSkill) bool
	// comparer overrides how findTriggerSkill picks between candidates.
	comparer func(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool
	// customHandler replaces defaultTriggerHandler outright (CWC, Focus).
	customHandler func(env *Env)

	useCastRate           bool
	triggerOnUse          bool
	allowTriggered        bool
	ignoreSourceRate      bool
	assumingEveryHitKills bool
	sourceWeapon          bool
	overlaps              *float64
	stagesAreOverlaps     *float64
}

// simSkill is one packaged skill for calcMultiSpellRotationImpact.
type simSkill struct {
	uuid          string
	cd            *float64
	cdOverride    *float64
	addsCastTime  *float64
	icdr          *float64
	addedCooldown *float64

	// simulation state
	simCD    float64
	nextTrig float64
	count    float64
}

// addTriggerIncMoreMods ports the local of the same name (L23).
func addTriggerIncMoreMods(activeSkill, sourceSkill *ActiveSkill) {
	for _, value := range activeSkill.SkillModList.Tabulate(modparser.Inc, sourceSkill.SkillCfg, "TriggeredDamage") {
		m := value.Mod
		activeSkill.SkillModList.AddMod(modparser.NewModFull("Damage", modparser.Inc, m.Value, m.Source, m.SourceSet, m.Flags, m.KeywordFlags, m.Tags...))
	}
	for _, value := range activeSkill.SkillModList.Tabulate(modparser.More, sourceSkill.SkillCfg, "TriggeredDamage") {
		m := value.Mod
		activeSkill.SkillModList.AddMod(modparser.NewModFull("Damage", modparser.More, m.Value, m.Source, m.SourceSet, m.Flags, m.KeywordFlags, m.Tags...))
	}
}

// slotMatch ports the local of the same name (L32).
func (env *Env) slotMatch(skill *ActiveSkill) bool {
	main := env.PlayerMainSkill
	fromItem := env.geFromItem(main.ActiveEffect.GrantedEffect) || env.geFromItem(skill.ActiveEffect.GrantedEffect)
	if !fromItem {
		if main.ActiveEffect.SrcInstance != nil && main.ActiveEffect.SrcInstance.FromItem {
			fromItem = true
		}
		if skill.ActiveEffect.SrcInstance != nil && skill.ActiveEffect.SrcInstance.FromItem {
			fromItem = true
		}
	}
	match1 := fromItem && skill.SocketGroup != nil && main.SocketGroup != nil &&
		skill.SocketGroup.Slot == main.SocketGroup.Slot
	match2 := !env.geFromItem(main.ActiveEffect.GrantedEffect) && skill.SocketGroup == main.SocketGroup
	return match1 || match2
}

// isTriggered ports the global of the same name (L40).
func isTriggered(skill *ActiveSkill) bool {
	if skill.SkillData.Flag("triggeredByUnique") || skill.SkillData.Flag("triggered") {
		return true
	}
	if skill.SkillTypes[modparser.SkillTypeInbuiltTrigger] || skill.SkillTypes[modparser.SkillTypeTriggered] {
		return true
	}
	// (`grantedEffect.triggered` is vestigial: no template sets it)
	return skill.ActiveEffect.SrcInstance != nil && skill.ActiveEffect.SrcInstance.Triggered
}

// processAddedCastTime ports the local of the same name (L44). Returns nil
// when the skill does not add its cast time to the trigger cooldown.
func processAddedCastTime(skill *ActiveSkill) *float64 {
	if !skill.SkillModList.Flag(skill.SkillCfg, "SpellCastTimeAddedToCooldownIfTriggered") {
		return nil
	}
	baseCastTime := 1.0
	if skill.SkillData.Flag("castTimeOverride") {
		baseCastTime = skill.SkillData.N("castTimeOverride")
	} else if ct := skill.ActiveEffect.GrantedEffect.CastTime; ct != nil {
		baseCastTime = *ct
	}
	inc := skill.SkillModList.Sum(modparser.Inc, skill.SkillCfg, "Speed")
	more := skill.SkillModList.More(skill.SkillCfg, "Speed")
	csi := util.RoundHalfUp((1+inc/100)*more, 2)
	addsCastTime := baseCastTime / csi
	skill.SkillFlags["addsCastTime"] = true
	return &addsCastTime
}

// packageSkillDataForSimulation ports the local of the same name (L64).
func (env *Env) packageSkillDataForSimulation(skill *ActiveSkill) *simSkill {
	s := &simSkill{
		uuid:         env.cacheSkillUUID(skill),
		addsCastTime: processAddedCastTime(skill),
	}
	if skill.SkillData.Flag("cooldown") {
		n := skill.SkillData.N("cooldown")
		s.cd = &n
	}
	if ov, ok := skill.SkillModList.Override(skill.SkillCfg, "CooldownRecovery"); ok {
		n := valueNum(ov)
		s.cdOverride = &n
	}
	icdr := Mod(skill.SkillModList, skill.SkillCfg, "CooldownRecovery")
	s.icdr = &icdr
	added := skill.SkillModList.Sum(modparser.Base, skill.SkillCfg, "CooldownRecovery")
	s.addedCooldown = &added
	return s
}

// defaultComparer ports the local of the same name (L68).
func defaultComparer(env *Env, uuid string, source *ActiveSkill, triggerRate *float64) bool {
	cached := env.GlobalCache[uuid]
	cachedSpeed, ok := cached.speedOrHitSpeed()
	if source == nil && ok {
		return true
	}
	rate := 0.0
	if triggerRate != nil {
		rate = *triggerRate
	}
	return ok && cachedSpeed > rate
}

// findTriggerSkill ports the local of the same name (L74): pick the trigger
// source with the highest cached rate.
func (env *Env) findTriggerSkill(skill, source *ActiveSkill, triggerRate *float64,
	comparer func(*Env, string, *ActiveSkill, *float64) bool) (*ActiveSkill, *float64, string) {
	if comparer == nil {
		comparer = defaultComparer
	}
	uuid := env.cacheSkillUUID(skill)
	// A miss is filled on the spot with a whole nested perform, and a
	// CALCULATOR env rebuilds even on a hit -- its cache bucket belongs to
	// the sub-environment, not to the build.
	if env.GlobalCache[uuid] == nil || env.Mode == ModeCalculator {
		env.BuildActiveSkill(env.Mode, skill, uuid)
	}
	cached := env.GlobalCache[uuid]
	usedByMirage := skill.SkillCfg != nil && skill.SkillCfg.SkillCond != nil && skill.SkillCfg.SkillCond["usedByMirage"]
	if cached != nil && comparer(env, uuid, source, triggerRate) && !skill.SkillFlags["disable"] && skill.SkillCfg != nil &&
		!usedByMirage && !skill.SkillTypes[modparser.SkillTypeOtherThingUsesSkill] {
		speed, _ := cached.speedOrHitSpeed()
		return skill, &speed, uuid
	}
	if source != nil {
		return source, triggerRate, env.cacheSkillUUID(source)
	}
	return source, triggerRate, ""
}

// calcMultiSpellRotationImpact ports the global of the same name (L90): the
// 1000-attack simulation that turns a source rate into a per-skill trigger
// rate, accounting for cooldown alignment.
func (env *Env) calcMultiSpellRotationImpact(skillRotation []*simSkill, sourceRate float64,
	triggerCD *float64, chance float64, actor *performActor) float64 {
	rotationIndex := 0
	nextTrigger := 0.0
	triggerIncrement := 1 / sourceRate
	simTime := triggerIncrement * 1000 // Simulate 1000 attacks
	skillCount := len(skillRotation)

	numOr := func(p *float64, def float64) float64 {
		if p == nil {
			return def
		}
		return *p
	}
	for _, skill := range skillRotation {
		base := (numOr(skill.cd, 0) + numOr(skill.addedCooldown, 0)) / numOr(skill.icdr, 1)
		if skill.cdOverride != nil {
			base = *skill.cdOverride
		}
		skill.simCD = math.Max(base, (numOr(triggerCD, 0)+numOr(skill.addsCastTime, 0))/numOr(skill.icdr, 1))
		skill.nextTrig = 0
		skill.count = 0
	}

	for nextTrigger < simTime {
		currentIndex := rotationIndex
		for {
			if skillRotation[currentIndex].nextTrig <= nextTrigger {
				// Skill at current index off cooldown, trigger it. The
				// cooldown starts at the beginning of the current tick and
				// ends at the next tick after expiration.
				skillRotation[currentIndex].count++
				skillRotation[currentIndex].nextTrig = ceilB(floorB(nextTrigger, data.Misc.ServerTickTime)+
					skillRotation[currentIndex].simCD, data.Misc.ServerTickTime)
				break
			}
			currentIndex = (currentIndex + 1) % skillCount // on cooldown, try the next
			if currentIndex == rotationIndex {             // all checked, trigger wasted
				break
			}
		}
		rotationIndex = (rotationIndex + 1) % skillCount
		nextTrigger += triggerIncrement
	}

	mainRate := 0.0
	mainUUID := env.cacheSkillUUID(actor.mainSkill)
	for _, sd := range skillRotation {
		// Account for trigger chance: the expected value of a geometric
		// distribution where p = chance, times triggerIncrement.
		rate := 1 / (simTime/sd.count + (triggerIncrement / chance * 100) - triggerIncrement)
		if mainUUID == sd.uuid {
			mainRate = rate
		}
	}
	return mainRate
}

func ceilB(x, base float64) float64  { return base * math.Ceil(x/base) }
func floorB(x, base float64) float64 { return base * math.Floor(x/base) }

// RunTriggers ports calcs.triggers (L1603) for one actor.
func (env *Env) RunTriggers(actor *performActor) {
	if actor == nil || actor.mainSkill == nil || actor.mainSkill.SkillFlags["disable"] {
		return
	}
	// The limitedSkills check is the RECURSION BREAKER for the cache
	// driver: Manaforged Arrows builds its own skill to learn its mana
	// cost, marking that uuid limited, so the nested perform's trigger
	// stage must skip itself (L1604). Without this the build recurses
	// unboundedly, allocating a full environment per level.
	if env.LimitedSkills[env.cacheSkillUUID(actor.mainSkill)] {
		return
	}
	main := actor.mainSkill
	skillName := main.ActiveEffect.GrantedEffect.Name
	triggerName := ""
	if main.TriggeredBy != nil {
		triggerName = main.TriggeredBy.GrantedEffect.Name
	}
	uniqueName := ""
	if isTriggered(main) {
		uniqueName = env.uniqueItemTriggerName(main)
	}
	skillID := main.ActiveEffect.GrantedEffect.Id

	keys := []string{skillID, strings.ToLower(skillName), strings.ToLower(triggerName)}
	keys = append(keys, strings.TrimPrefix(strings.ToLower(triggerName), "awakened "))
	keys = append(keys, strings.ToLower(uniqueName))

	var config *triggerConfig
	for _, k := range keys {
		if k == "" {
			continue
		}
		build, ok := triggerConfigTable[k]
		if !ok {
			continue
		}
		if build == nil {
			panic("triggers: configTable entry \"" + k + "\" is unported")
		}
		if config = build(env, actor); config != nil {
			break
		}
	}
	if config == nil {
		main.SkillData.Del("triggered")
		return
	}
	if config.actor == nil {
		config.actor = actor
	}
	if config.triggerName == "" {
		if triggerName != "" {
			config.triggerName = triggerName
		} else if skillName != "" {
			config.triggerName = skillName
		} else {
			config.triggerName = uniqueName
		}
	}
	if config.triggerChance == nil && main.ActiveEffect.SrcInstance != nil {
		if v := main.ActiveEffect.SrcInstance.TriggerChance; v.Set {
			n := v.V
			config.triggerChance = &n
		}
	}
	if config.customHandler != nil {
		config.customHandler(env)
		return
	}
	env.defaultTriggerHandler(config)
}

// uniqueItemTriggerName ports getUniqueItemTriggerName (L1586): the name of
// the unique item that grants the trigger.
func (env *Env) uniqueItemTriggerName(skill *ActiveSkill) string {
	if v := skill.SkillData.Str("triggerSource"); v != "" {
		return v
	}
	if len(skill.SupportList) >= 1 {
		for _, gemInstance := range skill.SupportList {
			ge := gemInstance.GrantedEffect
			if ge != nil && env.geFromItem(ge) && !ge.Support {
				return ge.Name
			}
		}
	}
	if skill.SocketGroup != nil {
		// source:find(".*:.*:(.*),.*") — the item name between the second
		// colon and the last comma of e.g. "Item:14:Miracle Nail, Ruby Ring".
		if src := skill.SocketGroup.Source; src != "" {
			if i := strings.Index(src, ":"); i >= 0 {
				if j := strings.Index(src[i+1:], ":"); j >= 0 {
					rest := src[i+1+j+1:]
					if k := strings.LastIndex(rest, ","); k >= 0 {
						return rest[:k]
					}
				}
			}
		}
	}
	return ""
}

var _ = modstore.Cfg{}
