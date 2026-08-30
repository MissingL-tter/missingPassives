// Package calc ports Modules/Calcs + CalcSetup + CalcPerform + CalcTools
// (and the CalcActiveSkill spine initEnv needs). The archive comparison
// replays fixtures dumped by tools/dump_calc.lua and diffs the resulting
// mod databases and outputs checkpoint by checkpoint.
package calc

import (
	"strconv"

	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/item"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/skills"
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
	ConfigInput        *ConfigInput     `lua:"configInput"`
	ConfigPlaceholder  *ConfigInput     `lua:"configPlaceholder"`
	ConfigModList      []*modparser.Mod `lua:"configModList"`
	ConfigEnemyModList []*modparser.Mod `lua:"configEnemyModList"`
	PartyEnemyModList  []*modparser.Mod `lua:"partyEnemyModList"`
	Spec               *SpecInput       `lua:"spec"`
}

// DamageCategory is configInput.enemyDamageType: which damage the EHP
// estimation is computed against. The seven values are the enemyDamageType
// option's list (ConfigOptions.lua L2332-2340); several become output-key
// prefixes ("Melee" -> "MeleeNotHitChance"), so the text is load-bearing.
type DamageCategory string

const (
	DamageAverage         DamageCategory = "Average"
	DamageUntyped         DamageCategory = "Untyped"
	DamageOverTime        DamageCategory = "DamageOverTime"
	DamageMelee           DamageCategory = "Melee"
	DamageProjectile      DamageCategory = "Projectile"
	DamageSpell           DamageCategory = "Spell"
	DamageSpellProjectile DamageCategory = "SpellProjectile"
)

// AilmentMode is configInput.ailmentMode: whether ailment base damage comes
// from the average application or from crits only (ConfigOptions.lua L210).
type AilmentMode string

const (
	AilmentAverage AilmentMode = "AVERAGE"
	AilmentCrit    AilmentMode = "CRIT"
)

// RepeatMode is configInput.repeatMode: how a repeating skill's repeats are
// counted (ConfigOptions.lua L945-949).
type RepeatMode string

const (
	RepeatNone     RepeatMode = "NONE"
	RepeatAverage  RepeatMode = "AVERAGE"
	RepeatFinal    RepeatMode = "FINAL"
	RepeatFinalDPS RepeatMode = "FINAL_DPS"
	// RepeatNoneMixedCase is the reference's own typo: the crit branch at
	// CalcOffence.lua L2993 tests "None" where every other site tests
	// "NONE". The option only ever stores "NONE", so that branch is dead
	// and its skill falls through to the elseif. Kept because the
	// fall-through is the observed behaviour.
	RepeatNoneMixedCase RepeatMode = "None"
)

// PhysMode is configInput.physMode: which element the "random element" mods
// pick, or all three at a third each (ConfigOptions.lua L211).
type PhysMode string

const (
	PhysAverage   PhysMode = "AVERAGE"
	PhysFire      PhysMode = "FIRE"
	PhysCold      PhysMode = "COLD"
	PhysLightning PhysMode = "LIGHTNING"
)

// ConfigInput is the slice of build.configInput (and, for the placeholder
// defaults, build.configPlaceholder) the calc reads. Numbers the reference
// tests for presence before use are Opt (a present 0 is truthy in Lua);
// numbers read as `x or 0` are plain. Config values that the reference
// keeps as UI strings are parsed to numbers by the input decoder.
type ConfigInput struct {
	Bandit                                         string
	PantheonMajorGod, PantheonMinorGod             string
	EnemyDamageType                                DamageCategory
	AilmentMode                                    AilmentMode
	RepeatMode                                     RepeatMode
	PhysMode                                       PhysMode
	RuthlessSupportMode                            string
	ChanceToIgnoreEnemyPhysicalDamageReductionMode string
	DoomBlastSource                                string

	PvpScaling            bool
	DisableEHPGainOnBlock bool
	ConditionLowLife      bool
	ExcludeCullingDPS     bool
	EEIgnoreHitDamage     bool
	IgnoreJewelLimits     bool
	IgnoreItemDisablers   bool
	// ConditionLowEnergyShield's raw value is what output.CappingES gets
	// when neither armour nor evasion caps ES.
	ConditionLowEnergyShield util.Opt[bool]

	EHPUnluckyWorstOf           util.Opt[float64]
	EnemyCritChance             util.Opt[float64]
	EnemyCritDamage             util.Opt[float64]
	EnemySpeed                  util.Opt[float64]
	EnemyMultiplierPvpDamage    util.Opt[float64]
	MultiplierPvpTvalueOverride util.Opt[float64]
	ResistancePenalty           util.Opt[float64]
	MeleeDistance               util.Opt[float64]
	ProjectileDistance          util.Opt[float64]
	OverrideEmptyRedSockets     util.Opt[float64]
	OverrideEmptyGreenSockets   util.Opt[float64]
	OverrideEmptyBlueSockets    util.Opt[float64]
	OverrideEmptyWhiteSockets   util.Opt[float64]

	MultiplierPoisonOnEnemy  float64
	MultiplierSummonedMinion float64
	MultiplierManaBurnStacks float64

	// Per damage type (Physical/Lightning/Cold/Fire/Chaos): enemy<Type>Damage,
	// enemy<Type>Pen, enemy<Type>Overwhelm, enemy<Type>Resist.
	EnemyDamage    map[string]float64
	EnemyPen       map[string]float64
	EnemyOverwhelm map[string]float64
	EnemyResist    map[string]float64
	// BalanceOfTerrorSelfCast is keyed by the curse name without spaces
	// (balanceOfTerrorSelfCast<Curse>).
	BalanceOfTerrorSelfCast map[string]bool
}

// SkillsTabInput carries what the initEnv skills stage reads from
// build.skillsTab.
type SkillsTabInput struct {
	SocketGroups []*SocketGroupInput `lua:"socketGroupList"`
	// ImbuedSupportBySlot maps slot name to the imbued support's granted
	// effect id.
	ImbuedSupportBySlot map[string]string `lua:"imbuedSupportBySlot"`
}

// SocketGroupInput is one socket group: the skills tab's typed group plus
// the runtime references the granted-skill update sets. GemList shadows the
// embedded group's list with the same gems wrapped in their calc state.
type SocketGroupInput struct {
	*skills.SocketGroup
	GemList []*SocketGemInput `lua:"gemList"`

	SourceItem     *Item
	SourceNode     *NodeInput
	ExplodeSources []ExplodeSource
}

// ExplodeSource is what grants an on-kill explosion: an item or a tree
// node. ExplodeKey is the reference's `modSource or "Tree:"..id`, the mod
// source the explosion's ExplodeMod entries carry.
type ExplodeSource interface{ ExplodeKey() string }

// SocketGemInput is one gem instance: the skills tab's typed gem plus
// resolution ids (the fixture stores ids; InitEnv resolves GemData/
// GrantedEffect from them) and runtime state (gemInstance.reqOverride,
// the explode source).
type SocketGemInput struct {
	*skills.Gem
	GemDataID           *string  `lua:"gemDataId"`
	GrantedEffectID     *string  `lua:"grantedEffectId"`
	ExplodeSourceItemID *float64 `lua:"explodeSourceItemId"`
	ExplodeSourceNodeID *float64 `lua:"explodeSourceNodeId"`

	ReqOverride   *float64
	ExplodeSource ExplodeSource
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
	RadiusNodes      map[int]string     `lua:"radiusNodes"`
	RadiusAttributes map[string]float64 `lua:"radiusAttributes"`
}

type ItemBaseInput struct {
	SubType *string         `lua:"subType"`
	Type    *string         `lua:"type"`
	Flask   *FlaskBaseInput `lua:"flask"`
}

// FlaskBaseInput carries base.flask.life/mana: recovery amounts whose
// presence the calc reads.
type FlaskBaseInput struct {
	Life util.Opt[float64] `lua:"life"`
	Mana util.Opt[float64] `lua:"mana"`
}

// ItemInput is the per-item slice of Item.lua state the calc stages read.
type ItemInput struct {
	Name                        string                   `lua:"name"`
	ModSource                   *string                  `lua:"modSource"`
	Title                       *string                  `lua:"title"`
	BaseName                    *string                  `lua:"baseName"`
	Type                        string                   `lua:"type"`
	Rarity                      string                   `lua:"rarity"`
	Corrupted                   *bool                    `lua:"corrupted"`
	Shaper                      *bool                    `lua:"shaper"`
	Elder                       *bool                    `lua:"elder"`
	Adjudicator                 *bool                    `lua:"adjudicator"`
	Basilisk                    *bool                    `lua:"basilisk"`
	Crusader                    *bool                    `lua:"crusader"`
	Eyrie                       *bool                    `lua:"eyrie"`
	Foulborn                    *bool                    `lua:"foulborn"`
	ClassRestriction            *string                  `lua:"classRestriction"`
	Limit                       *float64                 `lua:"limit"`
	Quality                     *float64                 `lua:"quality"`
	Base                        *ItemBaseInput           `lua:"base"`
	ModList                     []*modparser.Mod         `lua:"modList"`
	SlotModList                 map[int][]*modparser.Mod `lua:"slotModList"`
	BaseModList                 []*modparser.Mod         `lua:"baseModList"`
	BuffModList                 []*modparser.Mod         `lua:"buffModList"`
	GrantedSkills               []item.GrantedSkill      `lua:"grantedSkills"`
	Requirements                *item.Requirements       `lua:"requirements"`
	Sockets                     []SocketInput            `lua:"sockets"`
	AbyssalSocketCount          *float64                 `lua:"abyssalSocketCount"`
	SocketedJewelEffectModifier *float64                 `lua:"socketedJewelEffectModifier"`
	JewelRadiusIndex            *float64                 `lua:"jewelRadiusIndex"`
	FuncTypes                   []string                 `lua:"funcTypes"`
	JewelData                   *item.JewelData          `lua:"jewelData"`
	FlaskData                   *item.FlaskData          `lua:"flaskData"`
	TinctureData                *item.TinctureData       `lua:"tinctureData"`
	ArmourData                  *item.ArmourData         `lua:"armourData"`
	WeaponData                  map[int]*item.WeaponData `lua:"weaponData"`
	ExplicitLines               []string                 `lua:"explicitLines"`
	OtherLines                  []string                 `lua:"otherLines"`
}

// SocketInput is one item.sockets entry. Group carries the link group; the
// calc reads only Color, but the field is part of the item projection.
type SocketInput struct {
	Color string  `lua:"color"`
	Group float64 `lua:"group"`
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
	ID           float64 `lua:"id"`
	Type         string  `lua:"type"`
	Name         *string `lua:"name"`
	DN           *string `lua:"dn"`
	IsTattoo     *bool   `lua:"isTattoo"`
	OverrideType *string `lua:"overrideType"`
	ConqueredBy  *bool   `lua:"conqueredBy"`
	// DistanceToClassStart is spec-computed
	// (PassiveSpec:SetNodeDistanceToClassStart); Split Personality sockets
	// scale their effect by it.
	DistanceToClassStart *float64         `lua:"distanceToClassStart"`
	ModList              []*modparser.Mod `lua:"modList"`
	KeystoneMod          *modparser.Mod   `lua:"keystoneMod"`

	// GrantedSkills is runtime state buildModListForNode writes onto the
	// node (untagged: not part of the fixture echo).
	GrantedSkills []GrantedSkill
}

// ExplodeKey implements ExplodeSource.
func (n *NodeInput) ExplodeKey() string { return "Tree:" + strconv.FormatInt(int64(n.ID), 10) }
