// Port of ModStore.lua's EvalMod/GetStat and the surrounding actor/config
// model. Tag tables are modparser.Tag maps; field reads follow Lua
// truthiness, and in-place tag mutations (tag.div from divVar) are preserved.

package modstore

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// Externals are the calcLib functions EvalMod reaches into; the game-data /
// calc modules provide them, tests inject fixtures.
var Externals struct {
	GemIsType            func(gem any, keyword string) bool
	GetGameIdFromGemName func(name string, includeTransfigured bool) (string, bool)
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

// MinionData carries what the MonsterTag tag reads.
type MinionData struct {
	MonsterTags []string
}

// Actor is the slice of the calc actor that mod stores touch.
type Actor struct {
	Player          *Actor
	Enemy           *Actor
	ParentActor     *Actor
	Level           float64
	Others          map[string]*Actor // any other actor types
	DB              Store             // actor.modDB
	Output          map[string]any // numeric stats plus perform-owned tables/flags (Lua actor.output)
	ItemList        map[string]Item
	WeaponData1     map[string]any
	WeaponData2     map[string]any
	ActiveSkillList []*ActiveSkill
	MinionData      *MinionData
	ManaEfficiency  float64
	HasReservation  float64 // the SkillType.HasReservation id the fixtures key on
}

// outNum reads a numeric output entry (non-numbers act as 0, as Lua
// arithmetic on them would error before reaching here in practice).
func outNum(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
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
	Flags        *int64
	KeywordFlags *int64
	Source       string

	SkillName          string
	SummonSkillName    string
	SkillDist          *float64
	SkillCond          map[string]bool
	SkillTypes         map[float64]bool
	SkillPart          any
	SlotName           string
	SocketColor        string
	SocketNum          *float64
	StrengthGems       *float64
	DexterityGems      *float64
	IntelligenceGems   *float64
	SkillGem           any
	SkillGrantedEffect *GrantedEffectRef
	BaseFlags          map[string]bool
	Item               Item
	Actor              string
	SkillStats         map[string]any // aliases the pass output table (MainHand/OffHand)
}

func pow10(n int) float64      { return math.Pow(10, float64(n)) }
func floorf(v float64) float64 { return math.Floor(v) }

func tstr(tag modparser.Tag, k string) string {
	if s, ok := tag[k].(string); ok {
		return s
	}
	return ""
}

func tnum(tag modparser.Tag, k string) (float64, bool) {
	return numValue(tag[k])
}

// tlist returns a list-shaped tag value: parser tables store them as *D
// (array part), hand closures as []any.
func tlist(tag modparser.Tag, k string) []any {
	return asList(tag[k])
}

func asList(v any) []any {
	switch l := v.(type) {
	case []any:
		return l
	case *modparser.D:
		if l.Arr == nil {
			return []any{}
		}
		return l.Arr
	case modparser.Tag:
		// an empty table in a list position is a truthy empty list
		if len(l) == 0 {
			return []any{}
		}
	}
	return nil
}

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
		total := outNum(actor.Output[totalStat])
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
	if raw, present := actor.Output[stat]; stat == "ManaUnreserved" && present {
		v := outNum(raw)
		if math.IsNaN(v) {
			return outNum(actor.Output["Mana"])
		}
		if v < 0 {
			reservedPercentBeforeEfficiency := (math.Abs(outNum(actor.Output["ManaUnreservedPercent"])) + 100) * ((100 + actor.ManaEfficiency) / 100)
			return outNum(actor.Output["Mana"]) * (math.Ceil(reservedPercentBeforeEfficiency) / 100)
		}
	}
	// `actor.output[stat] or (cfg and cfg.skillStats[stat]) or 0`: a stored
	// false falls through to the skill stats, a stored 0 does not.
	if v, present := actor.Output[stat]; present && truthy(v) {
		return outNum(v)
	}
	if cfg != nil && cfg.SkillStats != nil {
		if v, present := cfg.SkillStats[stat]; present && truthy(v) {
			return outNum(v)
		}
	}
	return 0
}

var reCapWords = regexp.MustCompile(`([a-z])([0-9A-Za-z]*)`)

// evalMod ports ModStore:EvalMod. Returns nil when a tag rejects the mod.
func evalMod(ctx Store, mod *modparser.Mod, cfg *Cfg, globalLimits map[string]float64) any {
	base := ctx.base()
	value := mod.Value
	tags := modparser.ModTags(mod)
	for _, tag := range tags {
		switch tstr(tag, "type") {
		case "Multiplier":
			target := ctx
			limitTarget := ctx
			if la := tstr(tag, "limitActor"); la != "" {
				limitActor := base.getActor(la)
				if limitActor == nil {
					return nil
				}
				limitTarget = limitActor.DB
			}
			if act := tstr(tag, "actor"); act != "" {
				actor := base.getActor(act)
				if actor == nil {
					return nil
				}
				target = actor.DB
			}
			b := 0.0
			if varList := tlist(tag, "varList"); varList != nil {
				for _, v := range varList {
					b += getMultiplier(target, v.(string), cfg, false)
				}
			} else {
				b = getMultiplier(target, tstr(tag, "var"), cfg, false)
			}
			if dv := tstr(tag, "divVar"); dv != "" {
				// #EVAL: archive parity — writes the computed div back into the
				// SHARED tag table, visible to every later evaluation of this mod.
				tag["div"] = getMultiplier(ctx, dv, cfg, false)
			}
			div := tArith(tag, "div", 1)
			mult := math.Floor(b/div + 0.0001)
			if truthy(tag["noFloor"]) {
				mult = b / div
			}
			var limitTotal, limitNegTotal *float64
			if truthy(tag["limit"]) || truthy(tag["limitVar"]) || truthy(tag["limitStat"]) {
				var limit float64
				if l, ok := luaArith(tag["limit"]); ok && truthy(tag["limit"]) {
					limit = l
				} else if lv := tstr(tag, "limitVar"); lv != "" {
					limit = getMultiplier(limitTarget, lv, cfg, false)
				} else {
					limit = getStat(limitTarget, tstr(tag, "limitStat"), cfg)
				}
				if truthy(tag["limitTotal"]) {
					limitTotal = &limit
				} else if truthy(tag["limitNegTotal"]) {
					limitNegTotal = &limit
				} else {
					mult = math.Min(mult, limit)
				}
			}
			if truthy(tag["invert"]) && mult != 0 {
				mult = 1 / mult
			}
			tagBase := tArith(tag, "base", 0)
			value = scaleValue(value, mult, tagBase, limitTotal, limitNegTotal, false)
		case "MultiplierThreshold":
			target := ctx
			thresholdTarget := ctx
			hasThresholdActor := tstr(tag, "thresholdActor") != ""
			if hasThresholdActor {
				thresholdActor := base.getActor(tstr(tag, "thresholdActor"))
				if thresholdActor == nil {
					return nil
				}
				thresholdTarget = thresholdActor.DB
			}
			if act := tstr(tag, "actor"); act != "" {
				actor := base.getActor(act)
				if actor == nil {
					return nil
				}
				target = actor.DB
			}
			mult := 0.0
			if varList := tlist(tag, "varList"); varList != nil {
				for _, v := range varList {
					mult += getMultiplier(target, v.(string), cfg, false)
				}
			} else {
				mult = getMultiplier(target, tstr(tag, "var"), cfg, false)
			}
			var threshold float64
			if truthy(tag["threshold"]) {
				t, ok := tnum(tag, "threshold")
				if !ok {
					panic("modstore: non-numeric threshold (the Lua errors)")
				}
				threshold = t
			} else {
				thTarget := target
				if hasThresholdActor {
					thTarget = thresholdTarget
				}
				threshold = getMultiplier(thTarget, tstr(tag, "thresholdVar"), cfg, false)
			}
			if (truthy(tag["upper"]) && mult > threshold) ||
				(truthy(tag["equals"]) && mult != threshold) ||
				(!truthy(tag["upper"]) && mult < threshold) {
				return nil
			}
		case "PerStat":
			target := ctx
			if actor := base.getActor(tstr(tag, "actor")); actor != nil {
				target = actor.DB
			}
			b := 0.0
			if statList := tlist(tag, "statList"); statList != nil {
				for _, st := range statList {
					b += getStat(target, st.(string), cfg)
				}
			} else {
				b = getStat(target, tstr(tag, "stat"), cfg)
			}
			if dv := tstr(tag, "divVar"); dv != "" {
				// #EVAL: archive parity — writes the computed div back into the
				// SHARED tag table, visible to every later evaluation of this mod.
				tag["div"] = getMultiplier(ctx, dv, cfg, false)
			}
			div := tArith(tag, "div", 1)
			mult := math.Floor(b/div + 0.0001)
			var limitTotal *float64
			if truthy(tag["limit"]) || truthy(tag["limitVar"]) {
				var limit float64
				if l, ok := luaArith(tag["limit"]); ok && truthy(tag["limit"]) {
					limit = l
				} else {
					limit = getMultiplier(ctx, tstr(tag, "limitVar"), cfg, false)
				}
				if truthy(tag["limitTotal"]) {
					limitTotal = &limit
				} else {
					mult = math.Min(mult, limit)
				}
			}
			tagBase := tArith(tag, "base", 0)
			value = scaleValue(value, mult, tagBase, limitTotal, nil, false)
		case "PercentStat":
			target := ctx
			if actor := base.getActor(tstr(tag, "actor")); actor != nil {
				target = actor.DB
			}
			b := 0.0
			if statList := tlist(tag, "statList"); statList != nil {
				for _, st := range statList {
					b += getStat(target, st.(string), cfg)
				}
			} else {
				b = getStat(target, tstr(tag, "stat"), cfg)
			}
			var percent float64
			hasPercent := false
			if truthy(tag["percent"]) {
				percent, hasPercent = tArith(tag, "percent", 0), true
			} else if pv := tstr(tag, "percentVar"); pv != "" {
				percent, hasPercent = getMultiplier(ctx, pv, cfg, false), true
			}
			mult := b
			if hasPercent {
				mult = b * percent / 100
			}
			if truthy(tag["floor"]) {
				mult = math.Floor(mult)
			}
			var limitTotal *float64
			if truthy(tag["limit"]) || truthy(tag["limitVar"]) {
				var limit float64
				if l, ok := luaArith(tag["limit"]); ok && truthy(tag["limit"]) {
					limit = l
				} else {
					limit = getMultiplier(ctx, tstr(tag, "limitVar"), cfg, false)
				}
				if truthy(tag["limitTotal"]) {
					limitTotal = &limit
				} else {
					mult = math.Min(mult, limit)
				}
			}
			tagBase := tArith(tag, "base", 0)
			value = scaleValue(value, mult, tagBase, limitTotal, nil, true)
		case "StatThreshold":
			var stat float64
			if tlist(tag, "statList") != nil {
				// #EVAL: archive parity — the reference shadows its accumulator
				// with the loop variable and adds a stat NAME to a number, which
				// errors.
				panic("modstore: StatThreshold statList arithmetic on stat name (the Lua errors)")
			} else {
				stat = getStat(ctx, tstr(tag, "stat"), cfg)
			}
			var threshold float64
			if truthy(tag["threshold"]) {
				t, ok := tnum(tag, "threshold")
				if !ok {
					panic("modstore: non-numeric threshold (the Lua errors)")
				}
				threshold = t
			} else {
				threshold = getStat(ctx, tstr(tag, "thresholdStat"), cfg)
			}
			if truthy(tag["thresholdPercent"]) || truthy(tag["thresholdPercentVar"]) {
				var tp float64
				hasTP := false
				if truthy(tag["thresholdPercent"]) {
					tp, hasTP = tArith(tag, "thresholdPercent", 0), true
				} else if pv := tstr(tag, "thresholdPercentVar"); pv != "" {
					tp, hasTP = getMultiplier(ctx, pv, cfg, false), true
				}
				if hasTP {
					threshold = threshold * tp / 100
				}
			}
			if (truthy(tag["upper"]) && stat > threshold) || (!truthy(tag["upper"]) && stat < threshold) {
				return nil
			}
		case "DistanceRamp":
			if cfg == nil || cfg.SkillDist == nil {
				return nil
			}
			ramp := tlist(tag, "ramp")
			dist := *cfg.SkillDist
			first := asList(ramp[0])
			last := asList(ramp[len(ramp)-1])
			if dist <= toNum(first[0]) {
				value = arithNum(value) * arithNum(first[1])
			} else if dist >= toNum(last[0]) {
				value = arithNum(value) * arithNum(last[1])
			} else {
				for i := 0; i < len(ramp)-1; i++ {
					dat := asList(ramp[i])
					next := asList(ramp[i+1])
					if dist <= toNum(next[0]) {
						d0, v0 := arithNum(dat[0]), arithNum(dat[1])
						d1, v1 := arithNum(next[0]), arithNum(next[1])
						value = arithNum(value) * (v0 + (v1-v0)*(dist-d0)/(d1-d0))
						break
					}
				}
			}
		case "MeleeProximity":
			if cfg == nil || cfg.SkillDist == nil {
				return nil
			}
			ramp := tlist(tag, "ramp")
			dist := *cfg.SkillDist
			if dist <= 15 {
				value = arithNum(value) * arithNum(ramp[0])
			} else if dist >= 16 && dist <= 39 {
				r := arithNum(ramp[0])
				value = arithNum(value) * (r - (r/25)*(dist-15))
			} else if dist >= 40 {
				value = 0.0
			}
		case "Limit":
			var limit float64
			if truthy(tag["limit"]) {
				limit = tArith(tag, "limit", 0)
			} else {
				limit = getMultiplier(ctx, tstr(tag, "limitVar"), cfg, false)
			}
			value = math.Min(arithNum(value), limit)
		case "Condition":
			match := false
			var allOneH map[string]any
			if wd := base.Actor.WeaponData1; wd != nil && truthy(wd["countsAsAll1H"]) {
				allOneH = wd
			} else if wd := base.Actor.WeaponData2; wd != nil && truthy(wd["countsAsAll1H"]) {
				allOneH = wd
			}
			neg := truthy(tag["neg"])
			checkVar := func(v string) (matched, rejected bool) {
				if neg && allOneH != nil {
					if added, present := allOneH["Added"+v]; present {
						// Varunastra adds all weapon-type conditions; ignore
						// unless the condition was not added by it.
						if !truthy(added) {
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
			if varList := tlist(tag, "varList"); varList != nil {
				for _, v := range varList {
					m, rejected := checkVar(v.(string))
					if rejected {
						return nil
					}
					if m {
						match = true
						break
					}
				}
			} else {
				m, rejected := checkVar(tstr(tag, "var"))
				if rejected {
					return nil
				}
				match = m
			}
			if neg {
				match = !match
			}
			if !match {
				return nil
			}
		case "ActorCondition":
			match := false
			target := ctx
			hasTarget := true
			if act := tstr(tag, "actor"); act != "" {
				actor := base.getActor(act)
				if actor != nil {
					target = actor.DB
				} else {
					hasTarget = false
				}
			}
			if hasTarget && (truthy(tag["var"]) || tlist(tag, "varList") != nil) {
				if varList := tlist(tag, "varList"); varList != nil {
					for _, v := range varList {
						if getCondition(target, v.(string), cfg, false) {
							match = true
							break
						}
					}
				} else {
					match = getCondition(target, tstr(tag, "var"), cfg, false)
				}
			} else if act := tstr(tag, "actor"); act != "" && cfg != nil && act == cfg.Actor {
				match = true
			}
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		case "ItemCondition":
			if !evalItemCondition(ctx, tag, cfg) {
				return nil
			}
		case "SocketedIn":
			if !evalSocketedIn(tag, cfg) {
				return nil
			}
		case "SkillName":
			match := false
			if truthy(tag["includeTransfigured"]) {
				// The cfg-side lookup falls back to "" (a string); the tag-side
				matchGameId := ""
				if truthy(tag["summonSkill"]) {
					if cfg != nil {
						matchGameId, _ = Externals.GetGameIdFromGemName(cfg.SummonSkillName, true)
					}
				} else if cfg != nil && cfg.SkillName != "" {
					matchGameId, _ = Externals.GetGameIdFromGemName(cfg.SkillName, true)
				}
				// lookup yields nil for unknown names, which never equals it.
				if nameList := tlist(tag, "skillNameList"); nameList != nil {
					for _, n := range nameList {
						name, ok := n.(string)
						if ok && name != "" {
							if id, found := Externals.GetGameIdFromGemName(name, true); found && id == matchGameId {
								match = true
								break
							}
						}
					}
				} else if sn := tstr(tag, "skillName"); sn != "" {
					if id, found := Externals.GetGameIdFromGemName(sn, true); found && id == matchGameId {
						match = true
					}
				}
			} else {
				matchName := ""
				if truthy(tag["summonSkill"]) {
					if cfg != nil {
						matchName = cfg.SummonSkillName
					}
				} else if cfg != nil {
					matchName = cfg.SkillName
				}
				matchName = strings.ToLower(matchName)
				if nameList := tlist(tag, "skillNameList"); nameList != nil {
					for _, n := range nameList {
						if strings.ToLower(n.(string)) == matchName {
							match = true
							break
						}
					}
				} else {
					match = tstr(tag, "skillName") != "" && strings.ToLower(tstr(tag, "skillName")) == matchName
				}
			}
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		case "SkillId":
			if cfg == nil || cfg.SkillGrantedEffect == nil || cfg.SkillGrantedEffect.Id != tstr(tag, "skillId") {
				return nil
			}
		case "SkillPart":
			if cfg == nil {
				return nil
			}
			match := false
			if partList := tlist(tag, "skillPartList"); partList != nil {
				for _, part := range partList {
					if luaEq(part, cfg.SkillPart) {
						match = true
						break
					}
				}
			} else {
				match = luaEq(tag["skillPart"], cfg.SkillPart)
			}
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		case "SkillType":
			match := false
			if typeList := tlist(tag, "skillTypeList"); typeList != nil {
				for _, t := range typeList {
					if cfg != nil && cfg.SkillTypes != nil && cfg.SkillTypes[toNum(t)] {
						match = true
						break
					}
				}
			} else if st, ok := tnum(tag, "skillType"); ok {
				match = cfg != nil && cfg.SkillTypes != nil && cfg.SkillTypes[st]
			}
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		case "BaseFlag":
			var baseFlags map[string]bool
			if cfg != nil {
				if cfg.SkillGrantedEffect != nil && cfg.SkillGrantedEffect.BaseFlags != nil {
					baseFlags = cfg.SkillGrantedEffect.BaseFlags
				} else {
					baseFlags = cfg.BaseFlags
				}
			}
			match := baseFlags != nil && baseFlags[tstr(tag, "baseFlag")]
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		case "SlotName":
			if cfg == nil {
				return nil
			}
			match := false
			if slotList := tlist(tag, "slotNameList"); slotList != nil {
				for _, slot := range slotList {
					if slot.(string) == cfg.SlotName {
						match = true
						break
					}
				}
			} else {
				match = tstr(tag, "slotName") == cfg.SlotName
			}
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		case "ModFlagOr":
			if cfg == nil || cfg.Flags == nil {
				return nil
			}
			mf := tArith(tag, "modFlags", 0)
			if *cfg.Flags&int64(mf) == 0 {
				return nil
			}
		case "KeywordFlagAnd":
			if cfg == nil || cfg.KeywordFlags == nil {
				return nil
			}
			kf := tArith(tag, "keywordFlags", 0)
			if *cfg.KeywordFlags&int64(kf) != int64(kf) {
				return nil
			}
		case "MonsterTag":
			if base.Actor == nil || base.Actor.MinionData == nil || base.Actor.MinionData.MonsterTags == nil {
				return nil
			}
			match := false
			for _, tagName := range base.Actor.MinionData.MonsterTags {
				matchName := strings.ToLower(tagName)
				if tagList := tlist(tag, "monsterTagList"); tagList != nil {
					for _, n := range tagList {
						if strings.ToLower(n.(string)) == matchName {
							match = true
							break
						}
					}
				} else {
					match = tstr(tag, "monsterTag") != "" && strings.ToLower(tstr(tag, "monsterTag")) == matchName
				}
				if match {
					break
				}
			}
			if truthy(tag["neg"]) {
				match = !match
			}
			if !match {
				return nil
			}
		}
	}

	// Apply global limits
	for _, tag := range tags {
		gl, hasGL := tnum(tag, "globalLimit")
		key := tstr(tag, "globalLimitKey")
		if globalLimits != nil && hasGL && key != "" {
			v := 0.0
			if value != nil {
				v = arithNum(value)
			}
			if globalLimits[key]+v > gl {
				v = gl - globalLimits[key]
			}
			globalLimits[key] += v
			value = v
		}
	}
	return value
}

// luaEq is Lua == over the value kinds SkillPart carries.
func luaEq(a, b any) bool {
	if an, ok := numValue(a); ok {
		if bn, ok2 := numValue(b); ok2 {
			return an == bn
		}
		return false
	}
	return a == b
}

// scaleValue applies `value * mult + base` with the limit clamps, copying
// table values like the Lua does. ceil selects PercentStat's m_ceil form.
func scaleValue(value any, mult, tagBase float64, limitTotal, limitNegTotal *float64, ceil bool) any {
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
	if vt, ok := value.(modparser.Tag); ok {
		cp := modparser.CopyValue(vt).(modparser.Tag)
		if vm, ok := cp["mod"].(*modparser.Mod); ok {
			vm.Value = apply(arithNum(vm.Value))
		} else {
			cp["value"] = apply(arithNum(cp["value"]))
		}
		return cp
	}
	return apply(arithNum(value))
}

// evalItemCondition ports the ItemCondition tag.
func evalItemCondition(ctx Store, tag modparser.Tag, cfg *Cfg) bool {
	base := ctx.base()
	itemSlot := strings.ToLower(tstr(tag, "itemSlot"))
	itemSlot = reCapWords.ReplaceAllStringFunc(itemSlot, func(m string) string {
		return strings.ToUpper(m[:1]) + m[1:]
	})
	itemSlot = luaTrim(itemSlot)
	items := map[string]Item{}
	if truthy(tag["allSlots"]) {
		items = base.Actor.ItemList
	} else if base.Actor.ItemList != nil {
		if truthy(tag["bothSlots"]) {
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
	neg := truthy(tag["neg"])
	var matches []bool
	if sc := tstr(tag, "searchCond"); sc != "" {
		allSlots := truthy(tag["allSlots"])
		for slot, item := range items {
			include := (!allSlots || (item.ItemType() != "Jewel" && item.ItemType() != "Graft")) && slot != itemSlot
			if include || !truthy(tag["excludeSelf"]) {
				matches = append(matches, item.FindModifierSubstring(strings.ToLower(sc), strings.ToLower(slot)))
			}
		}
	}
	if rc := tstr(tag, "rarityCond"); rc != "" {
		for _, item := range items {
			matches = append(matches, item.Rarity() == rc)
		}
	}
	if cc, present := tag["corruptedCond"]; present {
		for _, item := range items {
			matches = append(matches, item.Corrupted() == truthy(cc))
		}
	}
	if sc, present := tag["shaperCond"]; present {
		for _, item := range items {
			matches = append(matches, item.Shaper() == truthy(sc))
		}
	}
	if ec, present := tag["elderCond"]; present {
		for _, item := range items {
			matches = append(matches, item.Elder() == truthy(ec))
		}
	}
	if nc := tstr(tag, "nameCond"); nc != "" {
		for _, item := range items {
			matches = append(matches, strings.EqualFold(strings.ToLower(item.Name()), strings.ToLower(nc)))
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
func evalSocketedIn(tag modparser.Tag, cfg *Cfg) bool {
	if cfg == nil || (!truthy(tag["slotName"]) && !truthy(tag["keyword"]) && !truthy(tag["socketColor"])) {
		return false
	}
	socketsIsAll := false
	if s, ok := tag["sockets"].(string); ok && s == "all" {
		socketsIsAll = true
	}
	match := map[string]bool{}
	if sn := tstr(tag, "slotName"); sn != "" {
		match["slotName"] = sn == cfg.SlotName
	}
	if kw := tstr(tag, "keyword"); kw != "" {
		match["keyword"] = cfg.SkillGem != nil && Externals.GemIsType(cfg.SkillGem, kw)
	} else if sc := tstr(tag, "socketColor"); sc != "" && !socketsIsAll {
		match["socketColor"] = sc == cfg.SocketColor
	}
	if truthy(tag["sockets"]) {
		var count float64
		switch tstr(tag, "socketColor") {
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
		} else if socketList := asList(tag["sockets"]); socketList != nil {
			if cfg.SocketNum == nil {
				return false
			}
			found := false
			for _, s := range socketList {
				if toNum(s) == *cfg.SocketNum {
					found = true
					break
				}
			}
			match["sockets"] = found
		} else if n, ok := tnum(tag, "sockets"); ok {
			match["sockets"] = count < n
		} else {
			return false
		}
	}
	neg := truthy(tag["neg"])
	for _, v := range match {
		if (!neg && !v) || (neg && v) {
			return false
		}
	}
	return true
}

// luaTrim trims the whitespace class Lua's %s covers.
func luaTrim(s string) string {
	return strings.Trim(s, " \t\n\v\f\r")
}

// luaArith coerces like Lua arithmetic: numbers pass through, numeric
// strings convert (tonumber), anything else fails.
func luaArith(v any) (float64, bool) {
	if n, ok := numValue(v); ok {
		return n, true
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

// tArith reads a tag field for an arithmetic context: absent/false gives the
// default; non-coercible truthy values panic (Lua's arithmetic error).
func tArith(tag modparser.Tag, k string, def float64) float64 {
	v := tag[k]
	if !truthy(v) {
		return def
	}
	n, ok := luaArith(v)
	if !ok {
		panic("modstore: arithmetic on non-numeric tag field " + k + " (the Lua errors)")
	}
	return n
}

// arithNum is Lua arithmetic over a mod value: numbers or numeric strings;
// anything else is the Lua arithmetic error.
func arithNum(v any) float64 {
	if n, ok := luaArith(v); ok {
		return n
	}
	panic("modstore: arithmetic on non-numeric value (the Lua errors)")
}
