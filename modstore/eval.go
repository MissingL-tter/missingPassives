// Port of ModStore.lua's EvalMod/GetStat and the surrounding actor/config
// model. Tags are modparser's typed tags; in-place tag mutations (tag.div
// from divVar) are preserved on the shared tag objects.

package modstore

import (
	"math"
	"regexp"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// Resolver is the game-data lookup mod evaluation reaches into
// (calcLib.getGameIdFromGemName); calc installs it on every actor, tests
// inject fixtures.
type Resolver interface {
	GetGameIdFromGemName(name string, includeTransfigured bool) (string, bool)
}

// GemRef is what the SocketedIn keyword check asks of cfg.skillGem.
type GemRef interface {
	IsType(keyword string) bool
}

// WeaponData is what the weapon-type condition tags read off
// actor.weaponData1/2; calc's live weapon-data table implements it.
type WeaponData interface {
	CountsAsAll1H() bool
	// AddedCond reads the "Added"+condition entry Varunastra handling writes:
	// present reports the key, added its truthiness.
	AddedCond(cond string) (added, present bool)
}

// Item is what ItemCondition reads off an item.
type Item interface {
	Name() string
	ItemType() string
	Rarity() string
	Corrupted() bool
	Shaper() bool
	Elder() bool
	FindModifierSubstring(substring, itemSlotName string) bool
}

// ActiveSkill carries what GetStat's reservation branches read.
type ActiveSkill struct {
	SkillTypes map[float64]bool
	Disable    bool // skillFlags.disable
	SkillData  map[string]float64
	BuffNames  []string // buffList[i].name
}

// MinionData carries the minion-table fields mod evaluation and the skill
// builder read (the reference hands the whole minion data table around).
type MinionData struct {
	MonsterTags []string
	DamageFixup *float64
}

// Actor is the slice of the calc actor that mod stores touch.
type Actor struct {
	Player          *Actor
	Enemy           *Actor
	ParentActor     *Actor
	Level           float64
	Others          map[string]*Actor // any other actor types
	DB              Store             // actor.modDB
	Output          Output            // Lua actor.output
	ItemList        map[string]Item
	WeaponData1     WeaponData
	WeaponData2     WeaponData
	ActiveSkillList []*ActiveSkill
	MinionData      *MinionData
	ManaEfficiency  float64
	HasReservation  float64 // the SkillType.HasReservation id the fixtures key on
	Resolver        Resolver
}

func (a *Actor) resolver() Resolver {
	if a.Resolver == nil {
		panic("modstore: no Resolver on the store's actor")
	}
	return a.Resolver
}

func (a *Actor) byType(actorType string) *Actor {
	switch actorType {
	case "enemy":
		return a.Enemy
	case "parent":
		return a.ParentActor
	}
	if a.Others != nil {
		return a.Others[actorType]
	}
	return nil
}

// GrantedEffectRef is cfg.skillGrantedEffect.
type GrantedEffectRef struct {
	Id        string
	BaseFlags map[string]bool
}

// Cfg is the query configuration. Pointer fields distinguish absent from
// zero where the Lua nil-checks (`not cfg.keywordFlags`, `cfg.skillDist`).
type Cfg struct {
	Flags        *modparser.ModFlag
	KeywordFlags *modparser.KeywordFlag
	Source       string

	SkillName          string
	SummonSkillName    string
	SkillDist          *float64
	SkillCond          map[string]bool
	SkillTypes         map[float64]bool
	SkillPart          util.Opt[float64]
	SlotName           string
	SocketColor        string
	SocketNum          *float64
	StrengthGems       *float64
	DexterityGems      *float64
	IntelligenceGems   *float64
	SkillGem           GemRef
	SkillGrantedEffect *GrantedEffectRef
	BaseFlags          map[string]bool
	Item               Item
	Actor              string
	SkillStats         Output // aliases the pass output table (MainHand/OffHand)
}

func pow10(n int) float64      { return math.Pow(10, float64(n)) }
func floorf(v float64) float64 { return math.Floor(v) }

// getStat ports ModStore:GetStat over an explicit target store.
func getStat(s Store, stat string, cfg *Cfg) float64 {
	actor := s.base().Actor
	isNameInBuffList := func(skill *ActiveSkill, names ...string) bool {
		for _, buff := range skill.BuffNames {
			for _, name := range names {
				if name != "" && buff == name {
					return true
				}
			}
		}
		return false
	}
	reservedPercent := func(totalStat, baseKey string) float64 {
		reserved := 0.0
		total := actor.Output.Get(totalStat).Num()
		if total == 0 {
			return 0
		}
		for _, skill := range actor.ActiveSkillList {
			if skill.SkillTypes[actor.HasReservation] && !skill.Disable && len(skill.BuffNames) > 0 && cfg != nil &&
				isNameInBuffList(skill, cfg.SkillName, cfg.SummonSkillName) {
				reserved = math.Floor(skill.SkillData[baseKey] / total * 100)
				break
			}
		}
		return math.Min(reserved, 100)
	}
	switch stat {
	case "ManaReservedPercent":
		return reservedPercent("Mana", "ManaReservedBase")
	case "LifeReservedPercent":
		return reservedPercent("Life", "LifeReservedBase")
	}
	out := func(key string) OutValue { return actor.Output.Get(key) }
	if v := out(stat); stat == "ManaUnreserved" && v.Kind != OutAbsent {
		n := v.Num()
		if math.IsNaN(n) {
			return out("Mana").Num()
		}
		if n < 0 {
			reservedPercentBeforeEfficiency := (math.Abs(out("ManaUnreservedPercent").Num()) + 100) * ((100 + actor.ManaEfficiency) / 100)
			return out("Mana").Num() * (math.Ceil(reservedPercentBeforeEfficiency) / 100)
		}
	}
	// `actor.output[stat] or (cfg and cfg.skillStats[stat]) or 0`: a stored
	// false falls through to the skill stats, a stored 0 does not.
	if v := out(stat); v.Truthy() {
		return v.Num()
	}
	if cfg != nil {
		if v := cfg.SkillStats.Get(stat); v.Truthy() {
			return v.Num()
		}
	}
	return 0
}

var reCapWords = regexp.MustCompile(`([a-z])([0-9A-Za-z]*)`)

// valueNum is Lua arithmetic over a mod value: numbers or numeric strings;
// anything else is the Lua arithmetic error.
func valueNum(v modparser.Value) float64 {
	n, ok := modparser.NumOf(v)
	if !ok {
		panic("modstore: arithmetic on non-numeric value (the Lua errors)")
	}
	return n
}

// evalMod ports ModStore:EvalMod. Returns nil when a tag rejects the mod.
func evalMod(ctx Store, mod *modparser.Mod, cfg *Cfg, globalLimits map[string]float64) modparser.Value {
	base := ctx.base()
	value := mod.Value
	tags := modparser.ModTags(mod)
	for _, tag := range tags {
		switch tag := tag.(type) {
		case *modparser.MultiplierTag:
			if tag.IsThreshold {
				if !evalMultiplierThreshold(ctx, tag, cfg) {
					return nil
				}
				continue
			}
			target := ctx
			limitTarget := ctx
			if tag.LimitActor != "" {
				limitActor := base.getActor(tag.LimitActor)
				if limitActor == nil {
					return nil
				}
				limitTarget = limitActor.DB
			}
			if tag.Actor != "" {
				actor := base.getActor(tag.Actor)
				if actor == nil {
					return nil
				}
				target = actor.DB
			}
			b := 0.0
			if tag.VarList != nil {
				for _, v := range tag.VarList {
					b += getMultiplier(target, v, cfg, false)
				}
			} else {
				b = getMultiplier(target, tag.Var, cfg, false)
			}
			if tag.DivVar != "" {
				// #EVAL: archive parity — writes the computed div back into the
				// SHARED tag, visible to every later evaluation of this mod.
				tag.Div = optOf(getMultiplier(ctx, tag.DivVar, cfg, false))
			}
			div := tag.Div.Or(1)
			mult := math.Floor(b/div + 0.0001)
			if tag.NoFloor {
				mult = b / div
			}
			var limitTotal, limitNegTotal *float64
			if tag.Limit.Set || tag.LimitVar != "" || tag.LimitStat != "" {
				var limit float64
				if tag.Limit.Set {
					limit = tag.Limit.V
				} else if tag.LimitVar != "" {
					limit = getMultiplier(limitTarget, tag.LimitVar, cfg, false)
				} else {
					limit = getStat(limitTarget, tag.LimitStat, cfg)
				}
				if tag.LimitTotal {
					limitTotal = &limit
				} else if tag.LimitNegTotal {
					limitNegTotal = &limit
				} else {
					mult = math.Min(mult, limit)
				}
			}
			if tag.Invert && mult != 0 {
				mult = 1 / mult
			}
			value = scaleValue(value, mult, tag.Base.Or(0), limitTotal, limitNegTotal, false)
		case *modparser.StatTag:
			switch tag.StatKind {
			case modparser.TagPerStat:
				target := ctx
				if actor := base.getActor(tag.Actor); actor != nil {
					target = actor.DB
				}
				b := 0.0
				if tag.StatList != nil {
					for _, st := range tag.StatList {
						b += getStat(target, st, cfg)
					}
				} else {
					b = getStat(target, tag.Stat, cfg)
				}
				if tag.DivVar != "" {
					// #EVAL: archive parity — writes the computed div back into the
					// SHARED tag, visible to every later evaluation of this mod.
					tag.Div = optOf(getMultiplier(ctx, tag.DivVar, cfg, false))
				}
				div := tag.Div.Or(1)
				mult := math.Floor(b/div + 0.0001)
				var limitTotal *float64
				if tag.Limit.Set || tag.LimitVar != "" {
					var limit float64
					if tag.Limit.Set {
						limit = tag.Limit.V
					} else {
						limit = getMultiplier(ctx, tag.LimitVar, cfg, false)
					}
					if tag.LimitTotal {
						limitTotal = &limit
					} else {
						mult = math.Min(mult, limit)
					}
				}
				value = scaleValue(value, mult, tag.Base.Or(0), limitTotal, nil, false)
			case modparser.TagPercentStat:
				target := ctx
				if actor := base.getActor(tag.Actor); actor != nil {
					target = actor.DB
				}
				b := 0.0
				if tag.StatList != nil {
					for _, st := range tag.StatList {
						b += getStat(target, st, cfg)
					}
				} else {
					b = getStat(target, tag.Stat, cfg)
				}
				var percent float64
				hasPercent := false
				if tag.Percent.Set {
					percent, hasPercent = tag.Percent.V, true
				} else if tag.PercentVar != "" {
					percent, hasPercent = getMultiplier(ctx, tag.PercentVar, cfg, false), true
				}
				mult := b
				if hasPercent {
					mult = b * percent / 100
				}
				if tag.Floor {
					mult = math.Floor(mult)
				}
				var limitTotal *float64
				if tag.Limit.Set || tag.LimitVar != "" {
					var limit float64
					if tag.Limit.Set {
						limit = tag.Limit.V
					} else {
						limit = getMultiplier(ctx, tag.LimitVar, cfg, false)
					}
					if tag.LimitTotal {
						limitTotal = &limit
					} else {
						mult = math.Min(mult, limit)
					}
				}
				value = scaleValue(value, mult, tag.Base.Or(0), limitTotal, nil, true)
			case modparser.TagStatThreshold:
				var stat float64
				if tag.StatList != nil {
					// #EVAL: archive parity — the reference shadows its accumulator
					// with the loop variable and adds a stat NAME to a number, which
					// errors.
					panic("modstore: StatThreshold statList arithmetic on stat name (the Lua errors)")
				} else {
					stat = getStat(ctx, tag.Stat, cfg)
				}
				var threshold float64
				if tag.Threshold.Set {
					threshold = tag.Threshold.V
				} else {
					threshold = getStat(ctx, tag.ThresholdStat, cfg)
				}
				if tag.ThresholdPercent.Set {
					threshold = threshold * tag.ThresholdPercent.V / 100
				} else if tag.ThresholdPercentVar != "" {
					threshold = threshold * getMultiplier(ctx, tag.ThresholdPercentVar, cfg, false) / 100
				}
				if (tag.Upper && stat > threshold) || (!tag.Upper && stat < threshold) {
					return nil
				}
			}
		case *modparser.DistanceRampTag:
			if cfg == nil || cfg.SkillDist == nil {
				return nil
			}
			ramp := tag.Ramp
			dist := *cfg.SkillDist
			first := ramp[0]
			last := ramp[len(ramp)-1]
			if dist <= first[0] {
				value = modparser.Num(valueNum(value) * first[1])
			} else if dist >= last[0] {
				value = modparser.Num(valueNum(value) * last[1])
			} else {
				for i := 0; i < len(ramp)-1; i++ {
					dat := ramp[i]
					next := ramp[i+1]
					if dist <= next[0] {
						d0, v0 := dat[0], dat[1]
						d1, v1 := next[0], next[1]
						value = modparser.Num(valueNum(value) * (v0 + (v1-v0)*(dist-d0)/(d1-d0)))
						break
					}
				}
			}
		case *modparser.MeleeProximityTag:
			if cfg == nil || cfg.SkillDist == nil {
				return nil
			}
			ramp := tag.Ramp
			dist := *cfg.SkillDist
			if dist <= 15 {
				value = modparser.Num(valueNum(value) * ramp[0])
			} else if dist >= 16 && dist <= 39 {
				r := ramp[0]
				value = modparser.Num(valueNum(value) * (r - (r/25)*(dist-15)))
			} else if dist >= 40 {
				value = modparser.Num(0)
			}
		case *modparser.LimitTag:
			var limit float64
			if tag.Limit.Set {
				limit = tag.Limit.V
			} else {
				limit = getMultiplier(ctx, tag.LimitVar, cfg, false)
			}
			value = modparser.Num(math.Min(valueNum(value), limit))
		case *modparser.CondTag:
			if tag.IsActor {
				if !evalActorCondition(ctx, tag, cfg) {
					return nil
				}
				continue
			}
			match := false
			var allOneH WeaponData
			if wd := base.Actor.WeaponData1; wd != nil && wd.CountsAsAll1H() {
				allOneH = wd
			} else if wd := base.Actor.WeaponData2; wd != nil && wd.CountsAsAll1H() {
				allOneH = wd
			}
			checkVar := func(v string) (matched, rejected bool) {
				if tag.Neg && allOneH != nil {
					if added, present := allOneH.AddedCond(v); present {
						// Varunastra adds all weapon-type conditions; ignore
						// unless the condition was not added by it.
						if !added {
							return false, true
						}
						return false, false
					}
				}
				if getCondition(ctx, v, cfg, false) || (cfg != nil && cfg.SkillCond != nil && cfg.SkillCond[v]) {
					return true, false
				}
				return false, false
			}
			if tag.VarList != nil {
				for _, v := range tag.VarList {
					m, rejected := checkVar(v)
					if rejected {
						return nil
					}
					if m {
						match = true
						break
					}
				}
			} else {
				m, rejected := checkVar(tag.Var)
				if rejected {
					return nil
				}
				match = m
			}
			if tag.Neg {
				match = !match
			}
			if !match {
				return nil
			}
		case *modparser.ItemCondTag:
			if !evalItemCondition(ctx, tag, cfg) {
				return nil
			}
		case *modparser.SlotTag:
			switch tag.SlotKind {
			case modparser.TagSocketedIn:
				if !evalSocketedIn(tag, cfg) {
					return nil
				}
			case modparser.TagSlotName:
				if cfg == nil {
					return nil
				}
				match := false
				if tag.SlotNameList != nil {
					for _, slot := range tag.SlotNameList {
						if slot == cfg.SlotName {
							match = true
							break
						}
					}
				} else {
					match = tag.SlotName == cfg.SlotName
				}
				if tag.Neg {
					match = !match
				}
				if !match {
					return nil
				}
			}
		case *modparser.SkillNameTag:
			match := false
			if tag.IncludeTransfigured {
				// The cfg-side lookup falls back to "" (a string); the tag-side
				gameIds := base.Actor.resolver()
				matchGameId := ""
				if tag.SummonSkill {
					if cfg != nil {
						matchGameId, _ = gameIds.GetGameIdFromGemName(cfg.SummonSkillName, true)
					}
				} else if cfg != nil && cfg.SkillName != "" {
					matchGameId, _ = gameIds.GetGameIdFromGemName(cfg.SkillName, true)
				}
				// lookup yields nil for unknown names, which never equals it.
				if tag.SkillNameList != nil {
					for _, name := range tag.SkillNameList {
						if name != "" {
							if id, found := gameIds.GetGameIdFromGemName(name, true); found && id == matchGameId {
								match = true
								break
							}
						}
					}
				} else if tag.SkillName != "" {
					if id, found := gameIds.GetGameIdFromGemName(tag.SkillName, true); found && id == matchGameId {
						match = true
					}
				}
			} else {
				matchName := ""
				if tag.SummonSkill {
					if cfg != nil {
						matchName = cfg.SummonSkillName
					}
				} else if cfg != nil {
					matchName = cfg.SkillName
				}
				matchName = strings.ToLower(matchName)
				if tag.SkillNameList != nil {
					for _, n := range tag.SkillNameList {
						if strings.ToLower(n) == matchName {
							match = true
							break
						}
					}
				} else {
					match = tag.SkillName != "" && strings.ToLower(tag.SkillName) == matchName
				}
			}
			if tag.Neg {
				match = !match
			}
			if !match {
				return nil
			}
		case *modparser.SkillIDTag:
			if cfg == nil || cfg.SkillGrantedEffect == nil || cfg.SkillGrantedEffect.Id != tag.SkillID {
				return nil
			}
		case *modparser.SkillPartTag:
			if cfg == nil {
				return nil
			}
			match := false
			if tag.PartList != nil {
				for _, part := range tag.PartList {
					if partEq(part, true, cfg.SkillPart) {
						match = true
						break
					}
				}
			} else {
				match = partEq(tag.Part.V, tag.Part.Set, cfg.SkillPart)
			}
			if tag.Neg {
				match = !match
			}
			if !match {
				return nil
			}
		case *modparser.SkillTypeTag:
			match := false
			if tag.SkillTypeList != nil {
				for _, t := range tag.SkillTypeList {
					if cfg != nil && cfg.SkillTypes != nil && cfg.SkillTypes[float64(t)] {
						match = true
						break
					}
				}
			} else if tag.SkillType != 0 {
				match = cfg != nil && cfg.SkillTypes != nil && cfg.SkillTypes[float64(tag.SkillType)]
			}
			if tag.Neg {
				match = !match
			}
			if !match {
				return nil
			}
		case *modparser.BaseFlagTag:
			var baseFlags map[string]bool
			if cfg != nil {
				if cfg.SkillGrantedEffect != nil && cfg.SkillGrantedEffect.BaseFlags != nil {
					baseFlags = cfg.SkillGrantedEffect.BaseFlags
				} else {
					baseFlags = cfg.BaseFlags
				}
			}
			match := baseFlags != nil && baseFlags[tag.BaseFlag]
			if tag.Neg {
				match = !match
			}
			if !match {
				return nil
			}
		case *modparser.ModFlagOrTag:
			if cfg == nil || cfg.Flags == nil {
				return nil
			}
			if *cfg.Flags&tag.ModFlags == 0 {
				return nil
			}
		case *modparser.KeywordAndTag:
			if cfg == nil || cfg.KeywordFlags == nil {
				return nil
			}
			if *cfg.KeywordFlags&tag.KeywordFlags != tag.KeywordFlags {
				return nil
			}
		case *modparser.MonsterTag:
			if base.Actor == nil || base.Actor.MinionData == nil || base.Actor.MinionData.MonsterTags == nil {
				return nil
			}
			match := false
			for _, tagName := range base.Actor.MinionData.MonsterTags {
				matchName := strings.ToLower(tagName)
				if tag.NameList != nil {
					for _, n := range tag.NameList {
						if strings.ToLower(n) == matchName {
							match = true
							break
						}
					}
				} else {
					match = tag.Name != "" && strings.ToLower(tag.Name) == matchName
				}
				if match {
					break
				}
			}
			if tag.Neg {
				match = !match
			}
			if !match {
				return nil
			}
		}
	}

	// Apply global limits
	for _, tag := range tags {
		var gl modparser.Value
		key := ""
		switch t := tag.(type) {
		case *modparser.MultiplierTag:
			if t.GlobalLimit.Set {
				gl, key = modparser.Num(t.GlobalLimit.V), t.GlobalLimitKey
			}
		case *modparser.StatTag:
			if t.GlobalLimit.Set {
				gl, key = modparser.Num(t.GlobalLimit.V), t.GlobalLimitKey
			}
		}
		if globalLimits != nil && gl != nil && key != "" {
			limit := float64(gl.(modparser.Num))
			v := 0.0
			if value != nil {
				v = valueNum(value)
			}
			if globalLimits[key]+v > limit {
				v = limit - globalLimits[key]
			}
			globalLimits[key] += v
			value = modparser.Num(v)
		}
	}
	return value
}

func evalMultiplierThreshold(ctx Store, tag *modparser.MultiplierTag, cfg *Cfg) bool {
	base := ctx.base()
	target := ctx
	thresholdTarget := ctx
	hasThresholdActor := tag.ThresholdActor != ""
	if hasThresholdActor {
		thresholdActor := base.getActor(tag.ThresholdActor)
		if thresholdActor == nil {
			return false
		}
		thresholdTarget = thresholdActor.DB
	}
	if tag.Actor != "" {
		actor := base.getActor(tag.Actor)
		if actor == nil {
			return false
		}
		target = actor.DB
	}
	mult := 0.0
	if tag.VarList != nil {
		for _, v := range tag.VarList {
			mult += getMultiplier(target, v, cfg, false)
		}
	} else {
		mult = getMultiplier(target, tag.Var, cfg, false)
	}
	var threshold float64
	if tag.Threshold.Set {
		threshold = tag.Threshold.V
	} else {
		thTarget := target
		if hasThresholdActor {
			thTarget = thresholdTarget
		}
		threshold = getMultiplier(thTarget, tag.ThresholdVar, cfg, false)
	}
	if (tag.Upper && mult > threshold) ||
		(tag.Equals && mult != threshold) ||
		(!tag.Upper && mult < threshold) {
		return false
	}
	return true
}

func evalActorCondition(ctx Store, tag *modparser.CondTag, cfg *Cfg) bool {
	base := ctx.base()
	match := false
	target := ctx
	hasTarget := true
	if tag.Actor != "" {
		actor := base.getActor(tag.Actor)
		if actor != nil {
			target = actor.DB
		} else {
			hasTarget = false
		}
	}
	if hasTarget && (tag.Var != "" || tag.VarList != nil) {
		if tag.VarList != nil {
			for _, v := range tag.VarList {
				if getCondition(target, v, cfg, false) {
					match = true
					break
				}
			}
		} else {
			match = getCondition(target, tag.Var, cfg, false)
		}
	} else if tag.Actor != "" && cfg != nil && tag.Actor == cfg.Actor {
		match = true
	}
	if tag.Neg {
		match = !match
	}
	return match
}

// partEq is Lua == between a tag's skill part (absent = nil) and the
// query's (a number or nil).
func partEq(part float64, set bool, cfgPart util.Opt[float64]) bool {
	if !set {
		return !cfgPart.Set
	}
	return cfgPart.Set && part == cfgPart.V
}

// scaleValue applies `value * mult + base` with the limit clamps, copying
// record values like the Lua does. ceil selects PercentStat's m_ceil form.
func scaleValue(value modparser.Value, mult, tagBase float64, limitTotal, limitNegTotal *float64, ceil bool) modparser.Value {
	apply := func(v float64) float64 {
		v = v*mult + tagBase
		if ceil {
			v = math.Ceil(v)
		}
		if limitTotal != nil {
			v = math.Min(v, *limitTotal)
		}
		if limitNegTotal != nil {
			v = math.Max(v, *limitNegTotal)
		}
		return v
	}
	switch t := value.(type) {
	case modparser.ModRef:
		cp := modparser.CloneValue(t).(modparser.ModRef)
		cp.Mod.Value = modparser.Num(apply(valueNum(cp.Mod.Value)))
		return cp
	case modparser.DataRef:
		cp := modparser.CloneValue(t).(modparser.DataRef)
		cp.Value = modparser.Num(apply(valueNum(cp.Value)))
		return cp
	case modparser.GemPropertyRef:
		cp := modparser.CloneValue(t).(modparser.GemPropertyRef)
		cp.Value = optOf(apply(cp.Value.Or(0)))
		return cp
	}
	return modparser.Num(apply(valueNum(value)))
}

// evalItemCondition ports the ItemCondition tag.
func evalItemCondition(ctx Store, tag *modparser.ItemCondTag, cfg *Cfg) bool {
	base := ctx.base()
	itemSlot := strings.ToLower(tag.ItemSlot)
	itemSlot = reCapWords.ReplaceAllStringFunc(itemSlot, func(m string) string {
		return strings.ToUpper(m[:1]) + m[1:]
	})
	itemSlot = strings.TrimSpace(itemSlot)
	items := map[string]Item{}
	if tag.AllSlots {
		items = base.Actor.ItemList
	} else if base.Actor.ItemList != nil {
		if tag.BothSlots {
			itemSlot1 := base.Actor.ItemList[itemSlot+" 1"]
			itemSlot2 := base.Actor.ItemList[itemSlot+" 2"]
			if itemSlot1 != nil && strings.Contains(itemSlot1.Name(), "Kalandra's Touch") {
				itemSlot1 = itemSlot2
			}
			if itemSlot2 != nil && strings.Contains(itemSlot2.Name(), "Kalandra's Touch") {
				itemSlot2 = itemSlot1
			}
			if itemSlot1 != nil && itemSlot2 != nil {
				items = map[string]Item{itemSlot + " 1": itemSlot1, itemSlot + " 2": itemSlot2}
			}
		} else {
			item := base.Actor.ItemList[itemSlot]
			if item == nil && cfg != nil {
				item = cfg.Item
			}
			if item != nil && strings.Contains(item.Name(), "Kalandra's Touch") {
				swapped := itemSlot
				if strings.HasSuffix(itemSlot, "1") {
					swapped = itemSlot[:len(itemSlot)-1] + "2"
				} else if strings.HasSuffix(itemSlot, "2") {
					swapped = itemSlot[:len(itemSlot)-1] + "1"
				}
				item = base.Actor.ItemList[swapped]
			}
			if item == nil && cfg != nil {
				item = cfg.Item
			}
			if item != nil {
				items = map[string]Item{itemSlot: item}
			}
		}
	}
	neg := tag.Neg
	var matches []bool
	if tag.SearchCond != "" {
		for slot, item := range items {
			include := (!tag.AllSlots || (item.ItemType() != "Jewel" && item.ItemType() != "Graft")) && slot != itemSlot
			if include || !tag.ExcludeSelf {
				matches = append(matches, item.FindModifierSubstring(strings.ToLower(tag.SearchCond), strings.ToLower(slot)))
			}
		}
	}
	if tag.RarityCond != "" {
		for _, item := range items {
			matches = append(matches, item.Rarity() == tag.RarityCond)
		}
	}
	if tag.CorruptedCond.Set {
		for _, item := range items {
			matches = append(matches, item.Corrupted() == tag.CorruptedCond.V)
		}
	}
	if tag.ShaperCond.Set {
		for _, item := range items {
			matches = append(matches, item.Shaper() == tag.ShaperCond.V)
		}
	}
	if tag.ElderCond.Set {
		for _, item := range items {
			matches = append(matches, item.Elder() == tag.ElderCond.V)
		}
	}
	if tag.NameCond != "" {
		for _, item := range items {
			matches = append(matches, strings.EqualFold(strings.ToLower(item.Name()), strings.ToLower(tag.NameCond)))
		}
	}
	hasItems := len(items) > 0
	match := true
	for _, b := range matches {
		if b == neg {
			match = false
			break
		}
	}
	if !match || (!hasItems && !neg) {
		return false
	}
	return true
}

// evalSocketedIn ports the SocketedIn tag; returns false to reject the mod.
func evalSocketedIn(tag *modparser.SlotTag, cfg *Cfg) bool {
	if cfg == nil || (tag.SlotName == "" && tag.Keyword == "" && tag.SocketColor == "") {
		return false
	}
	socketsIsAll := tag.SocketsAll
	match := map[string]bool{}
	if tag.SlotName != "" {
		match["slotName"] = tag.SlotName == cfg.SlotName
	}
	if tag.Keyword != "" {
		match["keyword"] = cfg.SkillGem != nil && cfg.SkillGem.IsType(tag.Keyword)
	} else if tag.SocketColor != "" && !socketsIsAll {
		match["socketColor"] = tag.SocketColor == cfg.SocketColor
	}
	if socketsIsAll || tag.Sockets != nil || tag.SocketCount.Set {
		var count float64
		switch tag.SocketColor {
		case "R":
			if cfg.StrengthGems != nil {
				count = *cfg.StrengthGems
			}
		case "G":
			if cfg.DexterityGems != nil {
				count = *cfg.DexterityGems
			}
		case "B":
			if cfg.IntelligenceGems != nil {
				count = *cfg.IntelligenceGems
			}
		}
		if socketsIsAll {
			total := 0.0
			for _, p := range []*float64{cfg.IntelligenceGems, cfg.DexterityGems, cfg.StrengthGems} {
				if p != nil {
					total += *p
				}
			}
			match["sockets"] = total == count && total > 0
		} else if tag.Sockets != nil {
			if cfg.SocketNum == nil {
				return false
			}
			found := false
			for _, s := range tag.Sockets {
				if s == *cfg.SocketNum {
					found = true
					break
				}
			}
			match["sockets"] = found
		} else {
			match["sockets"] = count < tag.SocketCount.V
		}
	}
	for _, v := range match {
		if (!tag.Neg && !v) || (tag.Neg && v) {
			return false
		}
	}
	return true
}
