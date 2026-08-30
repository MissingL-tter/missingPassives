// Directive vocabularies of the template families (export/templates/).
// Each family is a small sum type discriminated by the JSON "directive"
// key; a document's directive order is load-bearing (emission order).

package export

import "encoding/json"

// kind carries the discriminator so strict decoding accepts it.
type kind struct {
	Directive string `json:"directive"`
}

// ---- minions (Minions/, Spectres/) ----

type minionDirective interface{ minionDirective() }

// monsterDirective: variety id, display name (defaults to the variety),
// extra granted-effect ids appended to the monster's skill list.
type monsterDirective struct {
	kind
	Variety string   `json:"variety"`
	Name    string   `json:"name,omitempty"`
	Skills  []string `json:"skills,omitempty"`
}

// spectreDirective is monster + emit in one line.
type spectreDirective struct {
	kind
	Variety string   `json:"variety"`
	Name    string   `json:"name,omitempty"`
	Skills  []string `json:"skills,omitempty"`
}

type limitDirective struct {
	kind
	Name string `json:"name"`
}

type hostileDirective struct {
	kind
	Value bool `json:"value"`
}

// extraSkillDirective adds one granted-effect id to the pending monster.
type extraSkillDirective struct {
	kind
	Name string `json:"name"`
}

type emitDirective struct{ kind }

// modDirective: hand-written extra mods (modparser codec JSON).
type modDirective struct {
	kind
	Mods json.RawMessage `json:"mods"`
}

func (*monsterDirective) minionDirective()    {}
func (*spectreDirective) minionDirective()    {}
func (*limitDirective) minionDirective()      {}
func (*hostileDirective) minionDirective()    {}
func (*extraSkillDirective) minionDirective() {}
func (*emitDirective) minionDirective()       {}
func (*modDirective) minionDirective()        {}

var minionDirectives = map[string]func() minionDirective{
	"monster": func() minionDirective { return &monsterDirective{} },
	"spectre": func() minionDirective { return &spectreDirective{} },
	"limit":   func() minionDirective { return &limitDirective{} },
	"hostile": func() minionDirective { return &hostileDirective{} },
	"skill":   func() minionDirective { return &extraSkillDirective{} },
	"emit":    func() minionDirective { return &emitDirective{} },
	"mod":     func() minionDirective { return &modDirective{} },
}

// ---- skills (Skills/) ----

type skillDirective interface{ skillDirective() }

// skillHeadDirective opens a skill: granted-effect id and display name
// (defaults to the id).
type skillHeadDirective struct {
	kind
	Granted string `json:"granted"`
	Name    string `json:"name,omitempty"`
}

type flagsDirective struct {
	kind
	Flags []string `json:"flags"`
}

// modsDirective closes a skill; Flags are the emission switches
// (noLevels, noStats, noBaseFlags, noBaseMods).
type modsDirective struct {
	kind
	Flags []string `json:"flags,omitempty"`
}

type noGemDirective struct{ kind }

type addSkillTypesDirective struct {
	kind
	Types []string `json:"types"`
}

// baseModDirective: hand-written base mods (modparser codec JSON).
type baseModDirective struct {
	kind
	Mods json.RawMessage `json:"mods"`
}

func (*skillHeadDirective) skillDirective()     {}
func (*flagsDirective) skillDirective()         {}
func (*modsDirective) skillDirective()          {}
func (*noGemDirective) skillDirective()         {}
func (*addSkillTypesDirective) skillDirective() {}
func (*baseModDirective) skillDirective()       {}

var skillDirectives = map[string]func() skillDirective{
	"skill":         func() skillDirective { return &skillHeadDirective{} },
	"flags":         func() skillDirective { return &flagsDirective{} },
	"mods":          func() skillDirective { return &modsDirective{} },
	"noGem":         func() skillDirective { return &noGemDirective{} },
	"addSkillTypes": func() skillDirective { return &addSkillTypesDirective{} },
	"baseMod":       func() skillDirective { return &baseModDirective{} },
}

// ---- bases (Bases/) ----

type baseDirective interface{ baseDirective() }

type typeDirective struct {
	kind
	Name string `json:"name"`
}

type subTypeDirective struct {
	kind
	Name string `json:"name"`
}

type influenceBaseTagDirective struct {
	kind
	Tag string `json:"tag"`
}

type forceShowDirective struct {
	kind
	Value bool `json:"value"`
}

type forceHideDirective struct {
	kind
	Value bool `json:"value"`
}

type socketLimitDirective struct {
	kind
	Limit float64 `json:"limit"`
}

// baseItemDirective: one base item type by id, optional display name.
type baseItemDirective struct {
	kind
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// baseMatchDirective: every BaseItemTypes row whose Column (default Id)
// matches Pattern (Go regexp).
type baseMatchDirective struct {
	kind
	Column  string `json:"column,omitempty"`
	Pattern string `json:"pattern"`
}

// baseGroupDirective names a mod set later setBase lines fall back to.
type baseGroupDirective struct {
	kind
	Name string   `json:"name"`
	Mods []string `json:"mods"`
}

// setBestBaseDirective: rare on the best base of Class/SubType; Name
// overrides "SubType Class"; empty Mods fall back to the group.
type setBestBaseDirective struct {
	kind
	Class   string   `json:"class"`
	SubType string   `json:"subType"`
	Name    string   `json:"name,omitempty"`
	Mods    []string `json:"mods"`
}

// setBaseDirective: rare on a named base; Name is a "%s" pattern over the
// base class; empty Mods fall back to the group.
type setBaseDirective struct {
	kind
	Base string   `json:"base"`
	Name string   `json:"name,omitempty"`
	Mods []string `json:"mods"`
}

func (*typeDirective) baseDirective()             {}
func (*subTypeDirective) baseDirective()          {}
func (*influenceBaseTagDirective) baseDirective() {}
func (*forceShowDirective) baseDirective()        {}
func (*forceHideDirective) baseDirective()        {}
func (*socketLimitDirective) baseDirective()      {}
func (*baseItemDirective) baseDirective()         {}
func (*baseMatchDirective) baseDirective()        {}
func (*baseGroupDirective) baseDirective()        {}
func (*setBestBaseDirective) baseDirective()      {}
func (*setBaseDirective) baseDirective()          {}

var baseDirectives = map[string]func() baseDirective{
	"type":             func() baseDirective { return &typeDirective{} },
	"subType":          func() baseDirective { return &subTypeDirective{} },
	"influenceBaseTag": func() baseDirective { return &influenceBaseTagDirective{} },
	"forceShow":        func() baseDirective { return &forceShowDirective{} },
	"forceHide":        func() baseDirective { return &forceHideDirective{} },
	"socketLimit":      func() baseDirective { return &socketLimitDirective{} },
	"base":             func() baseDirective { return &baseItemDirective{} },
	"baseMatch":        func() baseDirective { return &baseMatchDirective{} },
	"baseGroup":        func() baseDirective { return &baseGroupDirective{} },
	"setBestBase":      func() baseDirective { return &setBestBaseDirective{} },
	"setBase":          func() baseDirective { return &setBaseDirective{} },
}

// ---- enemies (Enemies/Bosses) ----

type bossMonsterDirective interface{ bossMonsterDirective() }

type bossMonsterEntry struct {
	kind
	Name    string `json:"name"`
	Monster string `json:"monster"` // MonsterTypes id
	Uber    bool   `json:"uber,omitempty"`
}

func (*bossMonsterEntry) bossMonsterDirective() {}

var bossMonsterDirectives = map[string]func() bossMonsterDirective{
	"boss": func() bossMonsterDirective { return &bossMonsterEntry{} },
}

// ---- enemies (Enemies/BossSkills) ----

type bossSkillDirective interface{ bossSkillDirective() }

type bossHeadDirective struct {
	kind
	Name        string `json:"name"`
	Monster     string `json:"monster"` // MonsterVarieties id
	EarlierUber bool   `json:"earlierUber,omitempty"`
	MapBoss     bool   `json:"mapBoss,omitempty"`
}

// optIndex is a per-level index override: absent, explicit null (no
// index), or a number.
type optIndex struct {
	Set, Nil bool
	N        int
}

func (o *optIndex) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Nil = true
		return nil
	}
	return json.Unmarshal(b, &o.N)
}

func (o optIndex) MarshalJSON() ([]byte, error) {
	if o.Nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.N)
}

type bossSkillEntry struct {
	kind
	Name            string   `json:"name"`
	Granted         string   `json:"granted"`
	SkillIndex      optIndex `json:"skillIndex,omitzero"`
	SkillIndexUber  optIndex `json:"skillIndexUber,omitzero"`
	Granted2        string   `json:"grantedEffectId2,omitempty"`
	GrantedUber     string   `json:"grantedEffectIdUber,omitempty"`
	ExtraDamageMult *float64 `json:"extraDamageMult,omitempty"` // percent
	Stages          *float64 `json:"stages,omitempty"`
	SpeedMult       *float64 `json:"speedMult,omitempty"`
}

type tooltipDirective struct {
	kind
	Text string `json:"text"`
}

type skillListDirective struct{ kind }

func (*bossHeadDirective) bossSkillDirective()  {}
func (*bossSkillEntry) bossSkillDirective()     {}
func (*tooltipDirective) bossSkillDirective()   {}
func (*skillListDirective) bossSkillDirective() {}

var bossSkillDirectives = map[string]func() bossSkillDirective{
	"boss":      func() bossSkillDirective { return &bossHeadDirective{} },
	"skill":     func() bossSkillDirective { return &bossSkillEntry{} },
	"tooltip":   func() bossSkillDirective { return &tooltipDirective{} },
	"skillList": func() bossSkillDirective { return &skillListDirective{} },
}
