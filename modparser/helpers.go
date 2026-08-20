package modparser

import (
	"math"
	"strings"
)

// Helpers ported from ModParser.lua:2011-2119. Each mirrors its Lua original
// line for line; the source line is noted per function.

var damageTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

// dealNoNonDamageType — ModParser.lua:2017.
func dealNoNonDamageType(dmgType string, forMinionArg ...bool) any {
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
	var mods []any
	for _, damageType := range damageTypeList {
		if damageType != dmgType {
			dealNo := flag("DealNo" + damageType)
			if forMinion {
				mods = append(mods, mod("MinionModifier", "LIST", Tag{"mod": dealNo}))
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
			out[asciiLower(sk.name)] = sk.id
		}
	}
	return out
}

// grantedExtraSkill — ModParser.lua:2046.
func grantedExtraSkill(name string, level any, noSupports ...bool) any {
	name = strings.ReplaceAll(name, " skill", "")
	id, ok := gemIdLookup[name]
	if !ok {
		return nil
	}
	value := Tag{"skillId": id, "level": toNum(level)}
	if len(noSupports) > 0 && noSupports[0] {
		value["noSupports"] = true
	}
	return []any{mod("ExtraSkill", "LIST", value)}
}

// triggerExtraSkill — ModParser.lua:2054.
func triggerExtraSkill(name string, level any, opts ...Tag) any {
	var options Tag
	if len(opts) > 0 {
		options = opts[0]
	}
	lvl := toNum(level)
	mods := []any{}
	name = strings.ReplaceAll(name, " skill", "")
	sourceSkill, _ := options["sourceSkill"].(string)
	if sourceSkill != "" {
		sourceSkill = strings.ReplaceAll(sourceSkill, " skill", "")
	}
	id, ok := gemIdLookup[name]
	if ok {
		value := Tag{"skillId": id, "level": lvl, "triggered": true}
		if truthy(options["noSupports"]) {
			value["noSupports"] = true
		}
		if sourceSkill != "" {
			value["source"] = sourceSkill
		}
		if tc, has := options["triggerChance"]; has && tc != nil {
			value["triggerChance"] = toNum(tc)
		}
		mods = append(mods, mod("ExtraSkill", "LIST", value))
	}
	if truthy(options["ignoreHexproof"]) {
		mods = append(mods, mod("SkillData", "LIST", Tag{"key": "ignoreHexproof", "value": true}, Tag{"type": "SkillId", "skillId": gemIdLookup[name]}))
	}
	if truthy(options["onCrit"]) {
		mods = append(mods, mod("ExtraSkillMod", "LIST", Tag{"mod": mod("SkillData", "LIST", Tag{"key": "triggerOnCrit", "value": true})}, Tag{"type": "SkillId", "skillId": gemIdLookup[name]}))
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
func extraSupport(name string, level any, slotArg ...string) any {
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
	if g, resolved := supportGemResolve[skillId]; resolved {
		mods := []any{mod("ExtraSupport", "LIST", Tag{"skillId": g.grantedEffectId, "level": toNum(level)}, Tag{"type": "SocketedIn", "slotName": slot})}
		if g.hasSecondary {
			if g.secondarySupport {
				mods = append(mods, mod("ExtraSupport", "LIST", Tag{"skillId": g.secondaryGrantedEffectId, "level": toNum(level)}, Tag{"type": "SocketedIn", "slotName": slot}))
			} else {
				mods = append(mods, mod("ExtraSkill", "LIST", Tag{"skillId": g.secondaryGrantedEffectId, "level": toNum(level)}))
			}
		}
		return mods
	}
	return []any{
		mod("ExtraSupport", "LIST", Tag{"skillId": skillId, "level": toNum(level)}, Tag{"type": "SocketedIn", "slotName": slot}),
	}
}

// explodeFunc — ModParser.lua:2106.
func explodeFunc(chance float64, amount, typ string, tags ...any) any {
	amountNumber, ok := tonumber(amount)
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
	args := append([]any{}, tags...)
	explode := mod("ExplodeMod", "LIST", Tag{
		"type": firstToUpper(typ), "chance": chance / 100,
		"amount": amountNumber, "keyOfScaledMod": "chance",
	}, args...)
	return []any{explode, flag("CanExplode")}
}

// m_huge mirrors Lua's math.huge.
var m_huge = math.Inf(1)

// truthy mirrors Lua truthiness: nil and false are false, all else true.
func truthy(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

// toNum mirrors Lua's implicit tonumber on values that may arrive as capture
// strings or as numbers.
func toNum(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case string:
		f, _ := tonumber(t)
		return f
	}
	return 0
}

// gemIdOrNil mirrors indexing gemIdLookup in Lua: nil (an absent key in the
// built table) instead of Go's zero string when the skill is unknown.
func gemIdOrNil(name string) any {
	if id, ok := gemIdLookup[name]; ok {
		return id
	}
	return nil
}
