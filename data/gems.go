// data.gems and the gem lookups, ported from Data.lua's gem setup
// (including the Vaal AltX/AltY gem synthesis).
//
// The reference builds gemForSkill / gemForBaseName / the Vaal id lookups
// under pairs(), whose order varies per process; both sides of the archive
// comparison rebuild them in sorted gem-id order instead (a documented
// deliberate divergence for the collision cases).

package data

import "github.com/MissingL-tter/missingPassives/data/schema"

type Gem struct {
	Id                       string
	Name                     string
	BaseTypeName             *string
	GameId                   string
	VariantId                string
	GrantedEffectId          string
	SecondaryGrantedEffectId *string
	SecondaryEffectName      *string
	VaalGem                  bool
	Tags                     map[string]bool
	TagString                string
	ReqStr                   float64
	ReqDex                   float64
	ReqInt                   float64
	NaturalMaxLevel          float64

	GrantedEffect          *GrantedEffect
	SecondaryGrantedEffect *GrantedEffect
	GrantedEffectList      []*GrantedEffect
}

func loadGems(src schema.SkillsData) {
	Gems = map[string]*Gem{}
	for _, g := range src.Gems {
		gem := &Gem{
			Name:            sanitiseText(luaUnescape(g.Name)),
			GameId:          g.GameId,
			VariantId:       g.VariantId,
			GrantedEffectId: g.GrantedEffectId,
			VaalGem:         g.VaalGem,
			Tags:            map[string]bool{},
			TagString:       luaUnescape(g.TagString),
			ReqStr:          float64(g.ReqStr),
			ReqDex:          float64(g.ReqDex),
			ReqInt:          float64(g.ReqInt),
			NaturalMaxLevel: float64(g.NaturalMaxLevel),
		}
		if g.BaseTypeName != nil {
			s := luaUnescape(*g.BaseTypeName)
			gem.BaseTypeName = &s
		}
		gem.SecondaryGrantedEffectId = g.SecondaryGrantedEffectId
		gem.SecondaryEffectName = g.SecondaryEffectName
		for _, t := range g.Tags {
			gem.Tags[t] = true
		}
		Gems["Metadata/Items/Gems/SkillGem"+g.VariantId] = gem
	}

	setupGem := func(gem *Gem, gemId string) {
		gem.Id = gemId
		gem.GrantedEffect = Skills[gem.GrantedEffectId]
		if gem.SecondaryGrantedEffectId != nil {
			gem.SecondaryGrantedEffect = Skills[*gem.SecondaryGrantedEffectId]
		}
		gem.GrantedEffectList = []*GrantedEffect{gem.GrantedEffect}
		if gem.SecondaryGrantedEffect != nil {
			gem.GrantedEffectList = append(gem.GrantedEffectList, gem.SecondaryGrantedEffect)
		}
		if gem.NaturalMaxLevel == 0 {
			panic("data: gem without naturalMaxLevel " + gemId)
		}
	}
	for gemId, gem := range Gems {
		setupGem(gem, gemId)
	}

	// Vaal AltX/AltY gems: synthesised for vaal gems whose secondary effect
	// has an Alt variant skill.
	toAdd := map[string]*Gem{}
	for _, gemId := range sortedGemIds(Gems) {
		gem := Gems[gemId]
		if !gem.VaalGem || gem.SecondaryGrantedEffectId == nil {
			continue
		}
		for _, alt := range []string{"AltX", "AltY"} {
			altSkill := Skills[*gem.SecondaryGrantedEffectId+alt]
			if altSkill == nil {
				continue
			}
			secondary := *gem.SecondaryGrantedEffectId + alt
			newGem := &Gem{
				GameId:                   gem.GameId,
				VariantId:                gem.VariantId + alt,
				GrantedEffectId:          gem.GrantedEffectId,
				SecondaryGrantedEffectId: &secondary,
				VaalGem:                  gem.VaalGem,
				Tags:                     map[string]bool{},
				TagString:                gem.TagString,
				ReqStr:                   gem.ReqStr,
				ReqDex:                   gem.ReqDex,
				ReqInt:                   gem.ReqInt,
				NaturalMaxLevel:          gem.NaturalMaxLevel,
			}
			// Hybrid gems use the display name of the active skill
			if altSkill.BaseTypeName == nil {
				panic("data: alt vaal skill without baseTypeName " + secondary)
			}
			newGem.Name = "Vaal " + *altSkill.BaseTypeName
			for t, v := range gem.Tags {
				newGem.Tags[t] = v
			}
			setupGem(newGem, gemId+alt)
			toAdd[gemId+alt] = newGem
		}
	}
	for id, gem := range toAdd {
		Gems[id] = gem
	}

	// Lookups, rebuilt in sorted gem-id order.
	GemForSkill = map[*GrantedEffect]string{}
	GemForBaseName = map[string]string{}
	GemsByGameId = map[string]map[string]*Gem{}
	GemGrantedEffectIdForVaalGemId = map[string]string{}
	GemVaalGemIdForBaseGemId = map[string]string{}
	ids := sortedGemIds(Gems)
	for _, gemId := range ids {
		gem := Gems[gemId]
		GemForSkill[gem.GrantedEffect] = gemId
		if GemsByGameId[gem.GameId] == nil {
			GemsByGameId[gem.GameId] = map[string]*Gem{}
		}
		GemsByGameId[gem.GameId][gem.VariantId] = gem
		baseName := gem.Name
		if gem.GrantedEffect.Support && gem.GrantedEffectId != "SupportBarrage" {
			baseName = baseName + " Support"
		}
		GemForBaseName[luaLower(baseName)] = gemId
		if gem.BaseTypeName != nil && *gem.BaseTypeName != baseName {
			GemForBaseName[luaLower(*gem.BaseTypeName)] = gemId
		}
	}
	// The Vaal id lookups only ever process the original gems (the alt gems
	// are merged in afterwards in the reference); alt entries derive from
	// their base gem's mapping.
	originalIds := make([]string, 0, len(ids))
	for _, id := range ids {
		if toAdd[id] == nil {
			originalIds = append(originalIds, id)
		}
	}
	for _, gemId := range originalIds {
		gem := Gems[gemId]
		if !gem.VaalGem || gem.SecondaryGrantedEffectId == nil {
			continue
		}
		sec := *gem.SecondaryGrantedEffectId
		GemGrantedEffectIdForVaalGemId[sec] = gemId
		for _, otherId := range originalIds {
			if Gems[otherId].GrantedEffectId == sec {
				GemVaalGemIdForBaseGemId[gemId] = otherId
				break
			}
		}
		for _, alt := range []string{"AltX", "AltY"} {
			if Skills[sec+alt] == nil {
				continue
			}
			GemGrantedEffectIdForVaalGemId[sec+alt] = gemId + alt
			base, ok := GemVaalGemIdForBaseGemId[gemId]
			if !ok {
				panic("data: alt vaal gem without base mapping " + gemId)
			}
			GemVaalGemIdForBaseGemId[gemId+alt] = base + alt
		}
	}
}

func sortedGemIds(gems map[string]*Gem) []string {
	ids := make([]string, 0, len(gems))
	for id := range gems {
		ids = append(ids, id)
	}
	sortStrings(ids)
	return ids
}

// luaLower is string.lower (ASCII only, as Lua's default locale).
func luaLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// GemCanon builds a gem's plain-table shadow; granted effects appear as
// their skill ids (the dump normalises the same way to avoid inlining
// entire skills).
func GemCanon(g *Gem) map[string]any {
	m := map[string]any{
		"id":              g.Id,
		"name":            g.Name,
		"gameId":          g.GameId,
		"variantId":       g.VariantId,
		"grantedEffectId": g.GrantedEffectId,
		"tags":            g.Tags,
		"tagString":       g.TagString,
		"reqStr":          g.ReqStr,
		"reqDex":          g.ReqDex,
		"reqInt":          g.ReqInt,
		"naturalMaxLevel": g.NaturalMaxLevel,
	}
	if g.BaseTypeName != nil {
		m["baseTypeName"] = *g.BaseTypeName
	}
	if g.SecondaryGrantedEffectId != nil {
		m["secondaryGrantedEffectId"] = *g.SecondaryGrantedEffectId
	}
	if g.SecondaryEffectName != nil {
		m["secondaryEffectName"] = *g.SecondaryEffectName
	}
	if g.VaalGem {
		m["vaalGem"] = true
	}
	if g.GrantedEffect != nil {
		m["grantedEffect"] = "\x1bskill:" + g.GrantedEffect.Id
	}
	if g.SecondaryGrantedEffect != nil {
		m["secondaryGrantedEffect"] = "\x1bskill:" + g.SecondaryGrantedEffect.Id
	}
	list := map[string]any{}
	for i, ge := range g.GrantedEffectList {
		if ge != nil {
			list[itoa(i+1)] = "\x1bskill:" + ge.Id
		}
	}
	m["grantedEffectList"] = list
	return m
}

// GemForSkillCanon keys the lookup by skill id.
func GemForSkillCanon() map[string]string {
	out := map[string]string{}
	for ge, gemId := range GemForSkill {
		out[ge.Id] = gemId
	}
	return out
}
