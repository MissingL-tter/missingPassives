// Package calc ports Modules/Calcs + CalcSetup + CalcPerform + CalcTools
// (and the CalcActiveSkill spine initEnv needs). The archive comparison
// replays fixtures dumped by tools/dump_calc.lua and diffs the resulting
// mod databases and outputs checkpoint by checkpoint.
package calc

import (
	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

// BuildInput is the fixture boundary: everything initEnv reads from the
// Lua `build` object, in the shapes tools/dump_calc.lua emits. The lua
// tags make luacanon.Encode reproduce the dump's fixture canon byte for
// byte (the input-echo check in test/calc_test.go).
type BuildInput struct {
	CharacterLevel     float64          `lua:"characterLevel"`
	ClassID            float64          `lua:"classId"`
	ConfigEnemyLevel   *float64         `lua:"configEnemyLevel"`
	CurClassName       string           `lua:"curClassName"`
	TreeVersion        string           `lua:"treeVersion"`
	MainSocketGroup    float64          `lua:"mainSocketGroup"`
	ClassStats         ClassStats       `lua:"classStats"`
	ItemsTab           *ItemsTabInput   `lua:"itemsTab"`
	SkillsTab          *SkillsTabInput  `lua:"skillsTab"`
	SpectreList        []string         `lua:"spectreList"`
	ConfigInput        map[string]any   `lua:"configInput"`
	ConfigPlaceholder  map[string]any   `lua:"configPlaceholder"`
	ConfigModList      []*modparser.Mod `lua:"configModList"`
	ConfigEnemyModList []*modparser.Mod `lua:"configEnemyModList"`
	PartyEnemyModList  []*modparser.Mod `lua:"partyEnemyModList"`
	Spec               *SpecInput       `lua:"spec"`
}

// SkillsTabInput carries what the initEnv skills stage reads from
// build.skillsTab.
type SkillsTabInput struct {
	SocketGroups []*SocketGroupInput `lua:"socketGroupList"`
	// ImbuedSupportBySlot maps slot name to the imbued support's granted
	// effect id.
	ImbuedSupportBySlot map[string]string `lua:"imbuedSupportBySlot"`
}

// SocketGroupInput is one socket group: its scalar fields as a bag (the
// runtime mutates the same keys the Lua group table holds) plus the gems.
type SocketGroupInput struct {
	KV      map[string]any    `lua:"kv"`
	GemList []*SocketGemInput `lua:"gemList"`

	// runtime references set by the granted-skill group update
	SourceItem     *Item
	SourceNode     *NodeInput
	ExplodeSources []any
}

// SocketGemInput is one gem instance: scalar bag plus resolution refs.
// GemData/GrantedEffect are resolved from the ids by InitEnv; ReqOverride
// is runtime state (gemInstance.reqOverride).
type SocketGemInput struct {
	KV                  map[string]any `lua:"kv"`
	GemDataID           *string        `lua:"gemDataId"`
	GrantedEffectID     *string        `lua:"grantedEffectId"`
	ExplodeSourceItemID *float64       `lua:"explodeSourceItemId"`
	ExplodeSourceNodeID *float64       `lua:"explodeSourceNodeId"`

	GemData       *data.Gem
	GrantedEffect *data.GrantedEffect
	ReqOverride   *float64
	ExplodeSource any // *Item or *NodeInput
}

// ItemsTabInput carries what the initEnv items stage reads from
// build.itemsTab: the fixed slot list plus the item pool.
type ItemsTabInput struct {
	UseSecondWeaponSet *bool                 `lua:"useSecondWeaponSet"`
	Slots              []*SlotInput          `lua:"slots"`
	Items              map[int]*ItemInput    `lua:"items"`
	ItemSets           map[int]*ItemSetInput `lua:"itemSets"`
	ItemSetOrderList   []float64             `lua:"itemSetOrderList"`
}

// ItemSetInput is one itemsTab.itemSets entry: slot name -> selected item
// id (zero/absent slots elided).
type ItemSetInput struct {
	UseSecondWeaponSet *bool              `lua:"useSecondWeaponSet"`
	Slots              map[string]float64 `lua:"slots"`
}

type SlotInput struct {
	SlotName       string   `lua:"slotName"`
	Label          string   `lua:"label"`
	SlotNum        *float64 `lua:"slotNum"`
	WeaponSet      *float64 `lua:"weaponSet"`
	NodeID         *float64 `lua:"nodeId"`
	Active         *bool    `lua:"active"`
	ParentSlotName *string  `lua:"parentSlotName"`
	ItemID         *float64 `lua:"itemId"`
	// ContainJewelSocket marks cluster-jewel sockets (spec node flag).
	ContainJewelSocket *bool `lua:"containJewelSocket"`
	// RadiusNodes: for jewel sockets holding a radius jewel, the in-radius
	// node ids mapped to their node type (spec nodesInRadius[radiusIndex]).
	RadiusNodes      map[int]string `lua:"radiusNodes"`
	RadiusAttributes map[string]any `lua:"radiusAttributes"`
}

type ItemBaseInput struct {
	SubType *string `lua:"subType"`
	Type    *string `lua:"type"`
}

// ItemInput is the per-item slice of Item.lua state the calc stages read.
// The map-typed fields are scalar bags dumped as-is (flaskData, jewelData,
// weaponData sides, ...).
type ItemInput struct {
	Name                        string                    `lua:"name"`
	ModSource                   *string                   `lua:"modSource"`
	Title                       *string                   `lua:"title"`
	BaseName                    *string                   `lua:"baseName"`
	Type                        string                    `lua:"type"`
	Rarity                      string                    `lua:"rarity"`
	Corrupted                   *bool                     `lua:"corrupted"`
	Shaper                      *bool                     `lua:"shaper"`
	Elder                       *bool                     `lua:"elder"`
	Adjudicator                 *bool                     `lua:"adjudicator"`
	Basilisk                    *bool                     `lua:"basilisk"`
	Crusader                    *bool                     `lua:"crusader"`
	Eyrie                       *bool                     `lua:"eyrie"`
	Foulborn                    *bool                     `lua:"foulborn"`
	ClassRestriction            *string                   `lua:"classRestriction"`
	Limit                       *float64                  `lua:"limit"`
	Base                        *ItemBaseInput            `lua:"base"`
	ModList                     []*modparser.Mod          `lua:"modList"`
	SlotModList                 map[int][]*modparser.Mod  `lua:"slotModList"`
	BaseModList                 []*modparser.Mod          `lua:"baseModList"`
	BuffModList                 []*modparser.Mod          `lua:"buffModList"`
	GrantedSkills               []map[string]any          `lua:"grantedSkills"`
	Requirements                map[string]float64        `lua:"requirements"`
	Sockets                     []map[string]any          `lua:"sockets"`
	AbyssalSocketCount          *float64                  `lua:"abyssalSocketCount"`
	SocketedJewelEffectModifier *float64                  `lua:"socketedJewelEffectModifier"`
	JewelRadiusIndex            *float64                  `lua:"jewelRadiusIndex"`
	FuncTypes                   []string                  `lua:"funcTypes"`
	JewelData                   map[string]any            `lua:"jewelData"`
	FlaskData                   map[string]any            `lua:"flaskData"`
	TinctureData                map[string]any            `lua:"tinctureData"`
	ArmourData                  map[string]any            `lua:"armourData"`
	WeaponData                  map[int]map[string]any    `lua:"weaponData"`
	ExplicitLines               []string                  `lua:"explicitLines"`
	OtherLines                  []string                  `lua:"otherLines"`
}

type ClassStats struct {
	BaseStr float64 `lua:"base_str"`
	BaseDex float64 `lua:"base_dex"`
	BaseInt float64 `lua:"base_int"`
}

type SpecInput struct {
	AllocNodes                map[int]*NodeInput          `lua:"allocNodes"`
	KeystoneMap               map[string][]*modparser.Mod `lua:"keystoneMap"`
	RadiusNodeData            map[int]*NodeInput          `lua:"radiusNodeData"`
	AllocatedNotableCount     float64                     `lua:"allocatedNotableCount"`
	AllocatedKeystoneCount    float64                     `lua:"allocatedKeystoneCount"`
	AllocatedMasteryCount     float64                     `lua:"allocatedMasteryCount"`
	AllocatedMasteryTypeCount float64                     `lua:"allocatedMasteryTypeCount"`
	AllocatedMasteryTypes     map[string]float64          `lua:"allocatedMasteryTypes"`
	AllocatedTattooTypes      map[string]float64          `lua:"allocatedTattooTypes"`
}

// NodeInput carries the allocated-node fields buildModListForNodeList
// consumes. Pointer fields are keys the dump only sets sometimes.
type NodeInput struct {
	ID           float64          `lua:"id"`
	Type         string           `lua:"type"`
	Name         *string          `lua:"name"`
	DN           *string          `lua:"dn"`
	IsTattoo     *bool            `lua:"isTattoo"`
	OverrideType *string          `lua:"overrideType"`
	ConqueredBy  *bool            `lua:"conqueredBy"`
	ModList      []*modparser.Mod `lua:"modList"`
	KeystoneMod  *modparser.Mod   `lua:"keystoneMod"`

	// GrantedSkills is runtime state buildModListForNode writes onto the
	// node (untagged: not part of the fixture echo).
	GrantedSkills []GrantedSkill
}
