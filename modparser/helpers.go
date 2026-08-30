package modparser

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/internal/util"
)

// Helpers ported from ModParser.lua:2011-2119. Each mirrors its Lua original
// line for line; the source line is noted per function.

var damageTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

// dealNoNonDamageType — ModParser.lua:2017.
func dealNoNonDamageType(dmgType string, forMinionArg ...bool) []*Mod {
	forMinion := len(forMinionArg) > 0 && forMinionArg[0]
	dmgType = firstToUpper(dmgType)
	found := false
	for _, t := range damageTypeList {
		if t == dmgType {
			found = true
		}
	}
	if !found {
		return nil
	}
	var mods []*Mod
	for _, damageType := range damageTypeList {
		if damageType != dmgType {
			dealNo := flag("DealNo" + damageType)
			if forMinion {
				mods = append(mods, mod("MinionModifier", List, ModRef{Mod: dealNo}))
			} else {
				mods = append(mods, dealNo)
			}
		}
	}
	return mods
}

// gemIdLookup — ModParser.lua:2032-2045: skill display names to granted-effect
// ids, filtered the same way.
var gemIdLookup = buildGemIdLookup()

func buildGemIdLookup() map[string]string {
	out := map[string]string{
		"power charge on critical strike": "SupportPowerChargeOnCritical",
	}
	for _, sk := range skillData {
		if !sk.hidden || sk.fromItem || sk.fromTree {
			out[strings.ToLower(sk.name)] = sk.id
		}
	}
	return out
}

// grantedExtraSkill — ModParser.lua:2046.
func grantedExtraSkill(name string, level float64, noSupports ...bool) []*Mod {
	name = strings.ReplaceAll(name, " skill", "")
	id, ok := gemIdLookup[name]
	if !ok {
		return nil
	}
	value := SkillRef{SkillID: id, Level: opt(level)}
	if len(noSupports) > 0 && noSupports[0] {
		value.NoSupports = true
	}
	return []*Mod{mod("ExtraSkill", List, value)}
}

// triggerOpts are triggerExtraSkill's option table.
type triggerOpts struct {
	NoSupports     bool
	IgnoreHexproof bool
	OnCrit         bool
	TriggerChance  util.Opt[float64]
	SourceSkill    string
}

// triggerExtraSkill — ModParser.lua:2054.
func triggerExtraSkill(name string, level float64, opts ...triggerOpts) []*Mod {
	var options triggerOpts
	if len(opts) > 0 {
		options = opts[0]
	}
	mods := []*Mod{}
	name = strings.ReplaceAll(name, " skill", "")
	sourceSkill := options.SourceSkill
	if sourceSkill != "" {
		sourceSkill = strings.ReplaceAll(sourceSkill, " skill", "")
	}
	id, ok := gemIdLookup[name]
	if ok {
		value := SkillRef{SkillID: id, Level: opt(level), Triggered: true, NoSupports: options.NoSupports, Source: sourceSkill, TriggerChance: options.TriggerChance}
		mods = append(mods, mod("ExtraSkill", List, value))
	}
	if options.IgnoreHexproof {
		mods = append(mods, mod("SkillData", List, DataRef{Key: "ignoreHexproof", Value: Bool(true)}, &SkillIDTag{SkillID: gemIdLookup[name]}))
	}
	if options.OnCrit {
		mods = append(mods, mod("ExtraSkillMod", List, ModRef{Mod: mod("SkillData", List, DataRef{Key: "triggerOnCrit", Value: Bool(true)})}, &SkillIDTag{SkillID: gemIdLookup[name]}))
	}
	return mods
}

// titleWords mirrors string.gsub(" "..s, "%W%l", string.upper):sub(2): every
// lowercase letter following a non-word character is uppercased, including the
// first character.
func titleWords(s string) string {
	b := []byte(" " + s)
	for i := 1; i < len(b); i++ {
		prev := b[i-1]
		isWord := prev == '_' || (prev >= '0' && prev <= '9') || (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z')
		if !isWord && b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b[1:])
}

// condenseName mirrors name:gsub("^%l", upper):gsub(" %l", upper):gsub(" ", ""):
// title-case each space-led word, then strip the spaces.
func condenseName(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' && (i == 0 || b[i-1] == ' ') {
			b[i] -= 32
		}
	}
	return strings.ReplaceAll(string(b), " ", "")
}

// extraSupport — ModParser.lua:2072.
func extraSupport(name string, level float64, slotArg ...string) []*Mod {
	slot := ""
	hasSlot := len(slotArg) > 0
	if hasSlot {
		slot = slotArg[0]
	}
	skillId, ok := gemIdLookup[name]
	if !ok {
		skillId, ok = gemIdLookup[strings.TrimPrefix(name, "increased ")]
	}
	if !ok {
		skillId, ok = gemIdLookup[strings.TrimSuffix(name, " support")]
	}

	if slot == "main hand" || slot == "main hand weapon" {
		slot = "Weapon 1"
	} else if slot == "off hand" || slot == "off hand weapon" {
		slot = "Weapon 2"
	} else if hasSlot {
		slot = titleWords(slot)
	} else {
		slot = "{SlotName}"
	}

	if !ok {
		return nil
	}
	socketed := &SlotTag{SlotKind: TagSocketedIn, SlotName: slot}
	if g, resolved := supportGemResolve[skillId]; resolved {
		mods := []*Mod{mod("ExtraSupport", List, SkillRef{SkillID: g.grantedEffectId, Level: opt(level)}, socketed)}
		if g.hasSecondary {
			if g.secondarySupport {
				mods = append(mods, mod("ExtraSupport", List, SkillRef{SkillID: g.secondaryGrantedEffectId, Level: opt(level)}, socketed.Clone()))
			} else {
				mods = append(mods, mod("ExtraSkill", List, SkillRef{SkillID: g.secondaryGrantedEffectId, Level: opt(level)}))
			}
		}
		return mods
	}
	return []*Mod{mod("ExtraSupport", List, SkillRef{SkillID: skillId, Level: opt(level)}, socketed)}
}

// explodeFunc — ModParser.lua:2106.
func explodeFunc(chance float64, amount, typ string, tags ...Tag) []*Mod {
	amountNumber, ok := util.Tonumber(amount)
	if !ok {
		if amount == "tenth" {
			amountNumber, ok = 10, true
		} else if amount == "quarter" {
			amountNumber, ok = 25, true
		}
	}
	if !ok {
		return nil
	}
	explode := mod("ExplodeMod", List, ExplodeRef{Type: firstToUpper(typ), Chance: chance / 100, Amount: amountNumber, KeyOfScaledMod: "chance"}, tags...)
	return []*Mod{explode, flag("CanExplode")}
}

// m_huge mirrors Lua's math.huge.
var m_huge = math.Inf(1)

// gemIdOrNil mirrors indexing gemIdLookup in Lua: "" (the absent key the
// serialisers omit) instead of a guessed id when the skill is unknown.
func gemIdOrNil(name string) string {
	return gemIdLookup[name]
}
