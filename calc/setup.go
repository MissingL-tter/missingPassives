// Port of .archive/src/Modules/CalcSetup.lua, staged: this file covers
// initModDB and initEnv through the tree merge (the stages a build without
// items or skills exercises). The items loop and the skill/support stages
// panic loudly when an input would reach them, instead of silently
// diverging from the archive.
package calc

import (
	"fmt"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// GrantedSkill is one env.grantedSkills entry (tree- or item-granted).
type GrantedSkill struct {
	SkillID    string
	Level      float64
	NoSupports bool
	Source     string
	SourceNode *NodeInput
	NameSpec   string
	SourceItem *Item
	SlotName   string
	// Triggered/TriggerChance come from the item's ExtraSkill modifier.
	Triggered     bool
	TriggerChance util.Opt[float64]
}

// Item is the runtime item: the fixture payload plus mutable state.
// Implements modstore.Item for the ItemCondition eval tag.
type Item struct {
	In *ItemInput
	// jewelData.limitDisabled
	JewelLimitDisabled bool
}

func (it *Item) Name() string     { return it.In.Name }
func (it *Item) ItemType() string { return it.In.Type }
func (it *Item) Rarity() string   { return it.In.Rarity }
func (it *Item) Corrupted() bool  { return it.In.Corrupted != nil && *it.In.Corrupted }
func (it *Item) Shaper() bool     { return it.In.Shaper != nil && *it.In.Shaper }
func (it *Item) Elder() bool      { return it.In.Elder != nil && *it.In.Elder }

// ExplodeKey implements ExplodeSource: the item's mod source. An item
// without one is unreachable from Item:BuildModList (the Lua errors).
func (it *Item) ExplodeKey() string {
	if it.In.ModSource == nil {
		panic("calc: explode-source item without modSource (the Lua errors)")
	}
	return *it.In.ModSource
}

// patFind reports whether pat matches s. The item-tag patterns are Go
// regex, like the rest of the shipped tables: the only syntax they use is
// a leading ^ / trailing $, which means the same in both dialects, and the
// searchCond argument comes from a [a-zA-Z \t\n\v\f\r]+ capture, so no
// metacharacter can reach here. TestItemTagPatternsAreLiteral guards that.
// The cache is fine single-threaded (the calc pipeline is sequential).
var patCache = map[string]*regexp.Regexp{}

func patFind(s, pat string) bool {
	re, ok := patCache[pat]
	if !ok {
		re = regexp.MustCompile(pat)
		patCache[pat] = re
	}
	return re.MatchString(s)
}

// FindModifierSubstring ports Item:FindModifierSubstring (Item.lua:284).
// The fixture's ExplicitLines/OtherLines are already filtered for disabled
// and variant, matching the reference's per-line checks.
func (it *Item) FindModifierSubstring(substring, itemSlotName string) bool {
	explicit := strings.Count(substring, "explicit ")
	substring = strings.ReplaceAll(substring, "explicit ", "")
	modLines := append([]string{}, it.In.ExplicitLines...)
	if explicit < 1 {
		modLines = append(modLines, it.In.OtherLines...)
	}
	for _, line := range modLines {
		lower := strings.ToLower(line)
		if patFind(lower, substring) && !patFind(lower, substring+" modifier") {
			excluded := false
			if slotPats := data.ItemTagSpecialExclusionPattern[substring]; slotPats != nil {
				for _, specialMod := range slotPats[itemSlotName] {
					if patFind(lower, strings.ToLower(specialMod)) {
						excluded = true
						break
					}
				}
			}
			if !excluded {
				return true
			}
		}
		if slotPats := data.ItemTagSpecial[substring]; slotPats != nil {
			for _, specialMod := range slotPats[itemSlotName] {
				if patFind(lower, strings.ToLower(specialMod)) {
					return true
				}
			}
		}
	}
	return false
}

// influence property map of CalcSetup L1105/L1190 (map iteration order is
// irrelevant: distinct multiplier keys).
var influenceMults = map[string]func(*ItemInput) bool{
	"CorruptedItem": func(in *ItemInput) bool { return in.Corrupted != nil && *in.Corrupted },
	"ShaperItem":    func(in *ItemInput) bool { return in.Shaper != nil && *in.Shaper },
	"ElderItem":     func(in *ItemInput) bool { return in.Elder != nil && *in.Elder },
	"WarlordItem":   func(in *ItemInput) bool { return in.Adjudicator != nil && *in.Adjudicator },
	"HunterItem":    func(in *ItemInput) bool { return in.Basilisk != nil && *in.Basilisk },
	"CrusaderItem":  func(in *ItemInput) bool { return in.Crusader != nil && *in.Crusader },
	"RedeemerItem":  func(in *ItemInput) bool { return in.Eyrie != nil && *in.Eyrie },
}

// stripSpaces is s:gsub("%s+", "") for the ASCII whitespace mod text uses.
func stripSpaces(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v' {
			if b == nil {
				b = []byte(s[:i])
			}
			continue
		}
		if b != nil {
			b = append(b, c)
		}
	}
	if b == nil {
		return s
	}
	return string(b)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func listFlagOf(mods []*modparser.Mod, name string) bool {
	l := modstore.NewList(nil)
	l.AddList(mods)
	return l.Flag(nil, name)
}

// CalcMode is the mode argument threaded through initEnv and the output
// driver, and the key of the reference's GlobalCache.cachedData bucket
// (Data/Global.lua L356). The reference's third mode, "CALCS", is the calcs
// tab's own buildOutput; only these two are ported.
type CalcMode string

const (
	ModeMain       CalcMode = "MAIN"
	ModeCalculator CalcMode = "CALCULATOR"
)

// Env mirrors the slice of the Lua env the ported stages populate.
type Env struct {
	Build       *BuildInput
	Mode        CalcMode
	ConfigInput *ConfigInput

	ModDB     *modstore.DB
	EnemyDB   *modstore.DB
	ItemModDB *modstore.DB
	Player    *modstore.Actor
	Enemy     *modstore.Actor

	EnemyLevel                           float64
	ModeBuffs, ModeCombat, ModeEffective bool
	ClassID                              float64

	// AllocOrders holds the reference's pairs() order over env.allocNodes
	// for each buildModListForNodeList call, captured by the dump (LuaJIT
	// hash order is deterministic per table state but not derivable here,
	// and the table grows mid-initEnv when passives are granted).
	AllocOrders [][]int
	// ExtraOrders holds, per buildModListForNodeList call, the captured
	// pairs() order over env.extraRadiusNodeList (the node-call sequence
	// tail beyond the allocated nodes).
	ExtraOrders   [][]int
	Replay        *ReplayInput
	allocOrderIdx int
	// buildDepth counts nested BuildActiveSkill environments (defensive).
	buildDepth       int
	AllocNodes       map[int]*NodeInput
	InitialNodeModDB *modstore.List
	Keystone         modstore.KeystoneEnv

	// OverrideConditions is the reference's `override.conditions`, carried
	// so a copied env starts where this one ended up.
	OverrideConditions []string

	RadiusJewelList     []*RadiusJewel
	ExtraRadiusNodeList map[int]*NodeInput

	GrantedSkillsNodes []GrantedSkill
	GrantedSkillsItems []GrantedSkill
	GrantedSkills      []GrantedSkill
	ExplodeSources     []ExplodeSource
	GrantedPassives    map[int]bool

	// items stage state
	ItemPool               map[int]*Item    // by item id
	Items                  map[string]*Item // by slot name, post-disablers
	Flasks                 map[*Item]bool
	Tinctures              map[*Item]bool
	FlaskSlotMap           map[*Item]float64
	FlaskSlotOccupied      map[float64]bool
	AegisModList           *modstore.List
	TheIronMass            *modstore.List
	WeaponModList1         *modstore.List
	RequirementsTableItems []ItemRequirement

	// skills stage state
	MainSocketGroup          int
	CrossLinkedSupportGroups map[string][]string
	PlayerActiveSkills       []*ActiveSkill // env.player.activeSkillList
	PlayerMainSkill          *ActiveSkill   // env.player.mainSkill
	// LimitedSkills marks skill uuids the mirage machinery has taken over,
	// so the stage-cache paths skip them (CalcMirages L43).
	LimitedSkills map[string]bool
	// MirageHandled is calcs.mirages' return: true when the mirage
	// machinery took over and calcs.offence is skipped.
	MirageHandled         bool
	AuxSkillList          []*ActiveSkill
	RequirementsTableGems []GemRequirement
	geFromItemMark        map[*data.GrantedEffect]bool
	slotsByName           map[string]*SlotInput
	// statMapOverlay memoizes lazy skillStatMap copies per granted effect
	// (the reference memoizes into the shared skill tables; per-env keeps
	// the game-data canon pristine, same values).
	statMapOverlay map[*data.GrantedEffect]map[string]*data.StatMapEntry
	// globalEffectOverlay carries the hasGlobalEffect stamps the reference's
	// lazy statMap copies write onto the shared skill tables at calc time
	// (Data.lua metatable -> processMod). Kept per-env so the loaded data
	// stays immutable after Load; geHasGlobalEffect is the reader.
	globalEffectOverlay map[*data.GrantedEffect]bool
	// TriggeredCostWipes marks skill levels whose cost the reference wipes
	// for triggered granted skills (ProcessSocketGroup mutates the shared
	// table; kept per-env). The perform-stage cost reads must consult it.
	TriggeredCostWipes map[*data.SkillLevel]bool

	// perform-stage actor bundles
	playerPA, enemyPA, minionPA *performActor
	aegisItem                   modstore.Item
	Minion                      *Minion // env.minion (main skill's minion)
	Buffs                       map[string]*modstore.List
	MinionBuffsOut              map[string]*modstore.List
	Debuffs                     map[string]*modstore.List
	CurseSlots                  []*curseEntry

	// GlobalCache is GlobalCache.cachedData[mode] as the trigger stage
	// finds it, supplied by the replay (see calc/globalcache.go).
	GlobalCache map[string]*CachedSkill
}

// statMapLookup is grantedEffect.statMap[key] including the metatable's
// lazy copy from the shared skillStatMap.
func (env *Env) statMapLookup(ge *data.GrantedEffect, key string) *data.StatMapEntry {
	if e, ok := ge.StatMap[key]; ok {
		return e
	}
	if o := env.statMapOverlay[ge]; o != nil {
		if e, ok := o[key]; ok {
			return e
		}
	}
	e, setsGlobal := data.LazyStatMapCopy(ge, key)
	if setsGlobal {
		// The reference stamps hasGlobalEffect onto the SHARED granted
		// effect here; the overlay keeps the loaded data immutable, same
		// values (see globalEffectOverlay).
		if env.globalEffectOverlay == nil {
			env.globalEffectOverlay = map[*data.GrantedEffect]bool{}
		}
		owner := ge
		if ge.StatMapOwner != nil {
			owner = ge.StatMapOwner
		}
		env.globalEffectOverlay[owner] = true
	}
	if e != nil {
		if env.statMapOverlay == nil {
			env.statMapOverlay = map[*data.GrantedEffect]map[string]*data.StatMapEntry{}
		}
		if env.statMapOverlay[ge] == nil {
			env.statMapOverlay[ge] = map[string]*data.StatMapEntry{}
		}
		env.statMapOverlay[ge][key] = e
	}
	return e
}

// GemRequirement is one env.requirementsTableGems entry.
type GemRequirement struct {
	SourceGem     *SocketGemInput
	Str, Dex, Int float64
}

// ItemRequirement is one env.requirementsTableItems entry.
type ItemRequirement struct {
	SourceItem    *Item
	SourceSlot    string
	Str, Dex, Int float64
}

func newMod(name string, typ modparser.ModType, value modparser.Value, tags ...modparser.Tag) *modparser.Mod {
	return modparser.NewMod(name, typ, value, tags...)
}

// newModS is newMod with a source.
func newModS(name string, typ modparser.ModType, value modparser.Value, source string, tags ...modparser.Tag) *modparser.Mod {
	return modparser.NewModFull(name, typ, value, source, true, 0, 0, tags...)
}

// newModF is newMod with flags.
func newModF(name string, typ modparser.ModType, value modparser.Value, flags modparser.ModFlag, kw modparser.KeywordFlag, tags ...modparser.Tag) *modparser.Mod {
	return modparser.NewModFull(name, typ, value, "", false, flags, kw, tags...)
}

// newModSF is newMod with a source and flags.
func newModSF(name string, typ modparser.ModType, value modparser.Value, source string, flags modparser.ModFlag, kw modparser.KeywordFlag, tags ...modparser.Tag) *modparser.Mod {
	return modparser.NewModFull(name, typ, value, source, true, flags, kw, tags...)
}

// cloneMod deep-copies a mod (the reference's copyTable before mutation).
func cloneMod(m *modparser.Mod) *modparser.Mod { return m.Clone() }

func opt(v float64) util.Opt[float64] { return util.Some(v) }

// initModDB ports calcs.initModDB: stats and conditions common to all
// actors, in the reference's statement order.
func (env *Env) initModDB(modDB *modstore.DB) {
	cc := data.CharacterConstants
	add := func(m *modparser.Mod) { modDB.AddMod(m) }
	add(newModS("FireResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("ColdResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("LightningResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("ChaosResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("TotemFireResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("TotemColdResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("TotemLightningResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("TotemChaosResistMax", modparser.Base, modparser.Num(cc["base_maximum_all_resistances_%"]), "Base"))
	add(newModS("BlockChanceMax", modparser.Base, modparser.Num(cc["maximum_block_%"]), "Base"))
	add(newModS("SpellBlockChanceMax", modparser.Base, modparser.Num(cc["base_maximum_spell_block_%"]), "Base"))
	add(newModS("SpellDodgeChanceMax", modparser.Base, modparser.Num(75.0), "Base"))
	add(newModS("ChargeDuration", modparser.Base, modparser.Num(10.0), "Base"))
	add(newModS("PowerChargesMax", modparser.Base, modparser.Num(cc["max_power_charges"]), "Base"))
	add(newModS("FrenzyChargesMax", modparser.Base, modparser.Num(cc["max_frenzy_charges"]), "Base"))
	add(newModS("EnduranceChargesMax", modparser.Base, modparser.Num(cc["max_endurance_charges"]), "Base"))
	add(newModS("SiphoningChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("ChallengerChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("BlitzChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("InspirationChargesMax", modparser.Base, modparser.Num(cc["maximum_righteous_charges"]), "Base"))
	add(newModS("CrabBarriersMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("BrutalChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("BrineChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("PhysicalDamageGainAsCold", modparser.Base, modparser.Num(cc["physical_damage_%_to_add_as_cold_per_brine_charge"]), "Base", &modparser.MultiplierTag{Var: "BrineCharge"}))
	add(newModS("PhysicalDamageGainAsLightning", modparser.Base, modparser.Num(cc["physical_damage_%_to_add_as_lightning_per_brine_charge"]), "Base", &modparser.MultiplierTag{Var: "BrineCharge"}))
	add(newModS("AbsorptionChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("AfflictionChargesMax", modparser.Base, modparser.Num(0.0), "Base"))
	add(newModS("BloodChargesMax", modparser.Base, modparser.Num(cc["maximum_blood_scythe_charges"]), "Base"))
	add(newModS("MaxLifeLeechRate", modparser.Base, modparser.Num(cc["maximum_life_leech_rate_%_per_minute"]/60), "Base"))
	add(newModS("MaxManaLeechRate", modparser.Base, modparser.Num(cc["maximum_mana_leech_rate_%_per_minute"]/60), "Base"))
	add(newModS("ImpaleStacksMax", modparser.Base, modparser.Num(cc["impaled_debuff_number_of_reflected_hits"]), "Base"))
	add(newModS("SoulEaterMax", modparser.Base, modparser.Num(cc["soul_eater_maximum_stacks"]), "Base"))
	add(newModS("BleedStacksMax", modparser.Base, modparser.Num(1.0), "Base"))
	add(newModS("MaxEnergyShieldLeechRate", modparser.Base, modparser.Num(10.0), "Base"))
	add(newModS("MaxLifeLeechInstance", modparser.Base, modparser.Num(cc["maximum_life_leech_amount_per_leech_%_max_life"]), "Base"))
	add(newModS("MaxManaLeechInstance", modparser.Base, modparser.Num(cc["maximum_mana_leech_amount_per_leech_%_max_mana"]), "Base"))
	add(newModS("MaxEnergyShieldLeechInstance", modparser.Base, modparser.Num(cc["maximum_energy_shield_leech_amount_per_leech_%_max_energy_shield"]), "Base"))
	add(newModS("TrapThrowingTime", modparser.Base, modparser.Num(0.6), "Base"))
	add(newModS("MineLayingTime", modparser.Base, modparser.Num(0.3), "Base"))
	add(newModS("WarcryCastTime", modparser.Base, modparser.Num(0.8), "Base"))
	add(newModS("TotemPlacementTime", modparser.Base, modparser.Num(0.6), "Base"))
	add(newModS("BallistaPlacementTime", modparser.Base, modparser.Num(0.5), "Base"))
	add(newModS("ActiveTotemLimit", modparser.Base, modparser.Num(cc["base_number_of_totems_allowed"]), "Base"))
	add(newModS("ShockStacksMax", modparser.Base, modparser.Num(1.0), "Base"))
	add(newModS("ScorchStacksMax", modparser.Base, modparser.Num(1.0), "Base"))
	add(newModS("MovementSpeed", modparser.Inc, modparser.Num(-30.0), "Base", &modparser.CondTag{Var: "Maimed"}))
	add(newModSF("DamageTaken", modparser.Inc, modparser.Num(10.0), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.CondTag{Var: "Intimidated"}))
	add(newModSF("DamageTaken", modparser.Inc, modparser.Num(10.0), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.CondTag{Var: "Intimidated", Neg: true}, &modparser.CondTag{Var: "Party:Intimidated"}))
	add(newModSF("DamageTaken", modparser.Inc, modparser.Num(10.0), "Base", modparser.FlagSpell, modparser.KeywordNone, &modparser.CondTag{Var: "Unnerved"}))
	add(newModSF("DamageTaken", modparser.Inc, modparser.Num(10.0), "Base", modparser.FlagSpell, modparser.KeywordNone, &modparser.CondTag{Var: "Unnerved", Neg: true}, &modparser.CondTag{Var: "Party:Unnerved"}))
	add(newModS("Damage", modparser.More, modparser.Num(-10.0), "Base", &modparser.CondTag{Var: "Debilitated"}, &modparser.GlobalEffectTag{EffectName: "Debilitated", EffectType: "Debuff"}))
	add(newModS("MovementSpeed", modparser.More, modparser.Num(-20.0), "Base", &modparser.CondTag{Var: "Debilitated"}, &modparser.GlobalEffectTag{EffectName: "Debilitated", EffectType: "Debuff"}))
	add(newModS("Damage", modparser.More, modparser.Num(-10.0), "Base", &modparser.CondTag{Var: "MalignantMadness"}, &modparser.GlobalEffectTag{EffectName: "Malignant Madness", EffectType: "Debuff"}))
	add(newModS("ActionSpeed", modparser.More, modparser.Num(-10.0), "Base", &modparser.CondTag{Var: "MalignantMadness"}, &modparser.GlobalEffectTag{EffectName: "Malignant Madness", EffectType: "Debuff"}))
	add(newModS("Condition:Burning", modparser.Flag, modparser.Bool(true), "Base", &modparser.MarkerTag{Marker: modparser.TagIgnoreCond}, &modparser.CondTag{Var: "Ignited"}))
	add(newModS("Condition:Poisoned", modparser.Flag, modparser.Bool(true), "Base", &modparser.MarkerTag{Marker: modparser.TagIgnoreCond}, &modparser.MultiplierTag{IsThreshold: true, Var: "PoisonStack", Threshold: opt(1.0)}))
	add(newModS("Blind", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Blinded"}))
	add(newModS("Chill", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Chilled"}))
	add(newModS("Freeze", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Frozen"}))
	add(newModS("Fortify", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Fortify"}))
	add(newModS("Fortified", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Fortified"}))
	add(newModS("Excommunicated", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Excommunicated"}))
	add(newModS("Fanaticism", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Fanaticism"}))
	add(newModS("Onslaught", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Onslaught"}))
	add(newModS("UnholyMight", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "UnholyMight"}))
	add(newModS("ChaoticMight", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "ChaoticMight"}))
	add(newModS("Tailwind", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Tailwind"}))
	add(newModS("Adrenaline", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Adrenaline"}))
	add(newModS("AccelerationShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "AccelerationShrine"}))
	add(newModS("BrutalShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "BrutalShrine"}))
	add(newModS("DiamondShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "DiamondShrine"}))
	add(newModS("DivineShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "DivineShrine"}))
	add(newModS("EchoingShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "EchoingShrine"}))
	add(newModS("GloomShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "GloomShrine"}))
	add(newModS("GreaterFreezingShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "GreaterFreezingShrine"}))
	add(newModS("GreaterShockingShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "GreaterShockingShrine"}))
	add(newModS("GreaterSkeletalShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "GreaterSkeletalShrine"}))
	add(newModS("ImpenetrableShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "ImpenetrableShrine"}))
	add(newModS("MassiveShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "MassiveShrine"}))
	add(newModS("ReplenishingShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "ReplenishingShrine"}))
	add(newModS("ResistanceShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "ResistanceShrine"}))
	add(newModS("ResonatingShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "ResonatingShrine"}))
	add(newModS("LesserAccelerationShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LesserAccelerationShrine"}, &modparser.CondTag{Var: "AccelerationShrine", Neg: true}))
	add(newModS("LesserBrutalShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LesserBrutalShrine"}, &modparser.CondTag{Var: "BrutalShrine", Neg: true}))
	add(newModS("LesserImpenetrableShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LesserImpenetrableShrine"}, &modparser.CondTag{Var: "ImpenetrableShrine", Neg: true}))
	add(newModS("LesserMassiveShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LesserMassiveShrine"}, &modparser.CondTag{Var: "MassiveShrine", Neg: true}))
	add(newModS("LesserReplenishingShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LesserReplenishingShrine"}, &modparser.CondTag{Var: "ReplenishingShrine", Neg: true}))
	add(newModS("LesserResistanceShrine", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LesserResistanceShrine"}, &modparser.CondTag{Var: "ResistanceShrine", Neg: true}))
	add(newModS("AlchemistsGenius", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "AlchemistsGenius"}))
	add(newModS("LuckyHits", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "LuckyHits"}))
	add(newModS("Convergence", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "Convergence"}))
	add(newModS("PhysicalDamageReduction", modparser.Base, modparser.Num(-15.0), "Base", &modparser.CondTag{Var: "Crushed"}))
	add(newModS("CritChanceCap", modparser.Base, modparser.Num(100.0), "Base"))
	modDB.Conditions.Set("Buffed", env.ModeBuffs)
	modDB.Conditions.Set("Combat", env.ModeCombat)
	modDB.Conditions.Set("Effective", env.ModeEffective)
}

// applySoulMod ports pantheon.applySoulMod (PantheonTools.lua).
func applySoulMod(db *modstore.DB, god data.Pantheon) {
	soulKeys := make([]int, 0, len(god.Souls))
	for k := range god.Souls {
		soulKeys = append(soulKeys, k)
	}
	sort.Ints(soulKeys)
	for _, k := range soulKeys {
		for _, soulMod := range god.Souls[k].Mods {
			mods, extra, parsed := modparser.Parse(soulMod.Line)
			if parsed && extra == "" {
				list := make([]*modparser.Mod, len(mods))
				for i, mod := range mods {
					mod.Source = "Pantheon:" + god.Souls[1].Name
					mod.SourceSet = true
					list[i] = mod
				}
				db.AddList(list)
			}
		}
	}
}

// mergeDB ports Common.lua's mergeDB.
func mergeDB(dst, src *modstore.DB) {
	dst.AddDB(src)
	for k, v := range src.Conditions {
		dst.Conditions[k] = v
	}
	for k, v := range src.Multipliers {
		dst.Multipliers[k] = v
	}
}

// buildModListForNode ports calcs.buildModListForNode.
func (env *Env) buildModListForNode(node *NodeInput) (*modstore.List, ExplodeSource) {
	modList := modstore.NewList(nil)
	if node.Type == "Keystone" {
		modList.AddMod(node.KeystoneMod)
	} else {
		modList.AddList(node.ModList)
	}

	// Run first pass radius jewels
	for _, rad := range env.RadiusJewelList {
		if rad.Type == "Other" && rad.Nodes[int(node.ID)] != "" && rad.Nodes[int(node.ID)] != "Mastery" {
			rad.Fn(jewelNodeRef{node}, listWriter{modList}, rad.Data)
		}
	}

	if modList.Flag(nil, "PassiveSkillHasNoEffect") || (env.AllocNodes[int(node.ID)] != nil && modList.Flag(nil, "AllocatedPassiveSkillHasNoEffect")) {
		modList = modstore.NewList(nil) // wipeTable(modList)
	}

	// Apply effect scaling
	if scale := Mod(modList, nil, "PassiveSkillEffect"); scale != 1 {
		scaledList := modstore.NewList(nil)
		scaledList.ScaleAddList(modList.Mods, scale, false)
		modList = scaledList
	}

	// Run second pass radius jewels
	for _, rad := range env.RadiusJewelList {
		if nt := rad.Nodes[int(node.ID)]; nt != "" && nt != "Mastery" &&
			(rad.Type == "Threshold" ||
				(rad.Type == "Self" && env.AllocNodes[int(node.ID)] != nil) ||
				(rad.Type == "SelfUnalloc" && env.AllocNodes[int(node.ID)] == nil)) {
			rad.Fn(jewelNodeRef{node}, listWriter{modList}, rad.Data)
		}
	}

	if modList.Flag(nil, "PassiveSkillHasOtherEffect") {
		for i, v := range modList.List(nil, "NodeModifier") {
			if i == 0 {
				modList = modstore.NewList(nil) // wipeTable(modList)
			}
			modList.AddMod(v.(modparser.ModRef).Mod)
		}
	}

	node.GrantedSkills = nil
	for _, v := range modList.List(nil, "ExtraSkill") {
		skill := v.(modparser.SkillRef)
		{
			node.GrantedSkills = append(node.GrantedSkills, GrantedSkill{
				SkillID:    skill.SkillID,
				Level:      skill.Level.Or(0),
				NoSupports: true,
				Source:     fmt.Sprintf("Tree:%d", int64(node.ID)),
			})
		}
	}

	if modList.Flag(nil, "CanExplode") {
		return modList, node
	}
	// The reference's Flag() returns nil here, so t_insert(explodeSources,
	// nil) is a no-op — non-exploding nodes leave no entry at all.
	return modList, nil
}

// nextAllocOrder pops the captured iteration order for the next
// buildModListForNodeList call.
func (env *Env) nextAllocOrder() []int {
	if env.allocOrderIdx >= len(env.AllocOrders) {
		panic(fmt.Sprintf("calc: more buildModListForNodeList calls than captured allocOrders (have %d, depth %d, mode %s)", len(env.AllocOrders), env.buildDepth, env.Mode))
	}
	order := env.AllocOrders[env.allocOrderIdx]
	env.allocOrderIdx++
	return order
}

// buildModListForNodeList ports calcs.buildModListForNodeList over the
// captured pairs() orders.
func (env *Env) buildModListForNodeList(finishJewels bool) (*modstore.List, []ExplodeSource) {
	callIdx := env.allocOrderIdx
	// Initialise radius jewels
	for _, rad := range env.RadiusJewelList {
		rad.Data = &modparser.JewelFuncTag{ModSource: fmt.Sprintf("Tree:%d", rad.NodeID)}
	}

	// Add node modifiers
	modList := modstore.NewList(nil)
	explodeSources := []ExplodeSource{}
	for _, id := range env.nextAllocOrder() {
		node := env.AllocNodes[id]
		if node == nil {
			panic(fmt.Sprintf("calc: allocOrder id %d missing from allocNodes", id))
		}
		nodeModList, explode := env.buildModListForNode(node)
		if explode != nil {
			explodeSources = append(explodeSources, explode)
		}
		modList.AddList(nodeModList.Mods)
	}

	if finishJewels {
		// Process extra radius nodes (unallocated nodes near conversion or
		// threshold jewels), in the reference's captured pairs() order.
		if len(env.ExtraRadiusNodeList) > 0 {
			if callIdx >= len(env.ExtraOrders) {
				panic("calc: extraRadiusNodeList populated but no captured order for this call")
			}
			for _, id := range env.ExtraOrders[callIdx] {
				node := env.ExtraRadiusNodeList[id]
				if node == nil {
					panic(fmt.Sprintf("calc: extra order id %d missing from extraRadiusNodeList", id))
				}
				env.buildModListForNode(node)
			}
		}

		// Finalise radius jewels (jewelRadiusData is UI state, skipped)
		for _, rad := range env.RadiusJewelList {
			rad.Fn(nil, listWriter{modList}, rad.Data)
		}
	}

	return modList, explodeSources
}

// ReplayInput carries the dump-captured reference state a byte-exact
// replay needs beyond the build fixture itself.
type ReplayInput struct {
	// AllocOrders: pairs() order per buildModListForNodeList call.
	AllocOrders [][]int
	// NodeOrders: the full buildModListForNode call sequence per NodeList
	// call; the tail beyond AllocOrders is the extraRadiusNodeList order.
	NodeOrders [][]int
	// GrantedPassiveNodes: resolved notable/ascendancy nodes by the
	// GrantedPassive value (anoints etc.).
	GrantedPassiveNodes map[string]*NodeInput
	// GrantedAscendancyNodes: resolved nodes by GrantedAscendancyNode name
	// (Forbidden Flame/Flesh).
	GrantedAscendancyNodes map[string]*NodeInput
	// EnergyBladeItems: the synthesized Energy Blade weapons by slot name
	// (the reference constructs them via the Item machinery on re-entry).
	EnergyBladeItems map[string]*ItemInput
	// GlobalCache: GlobalCache.cachedData[mode] as the trigger stage finds
	// it. Filled by Calcs.lua's buildOutput driver, which is not one of the
	// stages ported here (see calc/globalcache.go).
	GlobalCache map[string]*CachedSkill
	// MirageAllocOrders / MirageNodeOrders: the same two sequences for the
	// second initEnv copyActiveSkill runs (CALCULATOR mode) when a mirage
	// path takes over. Empty for a build with no mirages.
	MirageAllocOrders [][]int
	MirageNodeOrders  [][]int
	// StubHandoff makes nested performs body-only, mirroring the archive
	// dump's checkpoint phase where calcs.defence/offence are stubbed out
	// (the reference's nested calls inherit whatever those functions
	// currently are). Fixture-mode only: the differential harness sets it
	// for its checkpoint replay; driver runs leave it false (full perform).
	StubHandoff bool
}

// InitEnv ports calcs.initEnv for the one-shot MAIN mode over a fixture
// BuildInput, including the Energy Blade re-entry (the reference restarts
// initEnv with override.conditions when an enabled Energy Blade gem is
// found).
func InitEnv(in *BuildInput, mode CalcMode, replay *ReplayInput) *Env {
	return initEnvOverride(in, mode, replay, nil)
}

// initEnvOverride is initEnv with the reference's `override` argument, which
// copyActiveSkill passes on from the env it copies (`calcs.initEnv(env.build,
// mode, env.override)`) so a sub-environment inherits an Energy Blade
// re-entry instead of rediscovering it.
func initEnvOverride(in *BuildInput, mode CalcMode, replay *ReplayInput, overrideConditions []string) *Env {
	orderStart := 0
	for {
		env, restart := initEnvPass(in, mode, replay, orderStart, overrideConditions)
		if !restart {
			return env
		}
		orderStart = env.allocOrderIdx
		overrideConditions = append(overrideConditions, "AffectedByEnergyBlade")
	}
}

func initEnvPass(in *BuildInput, mode CalcMode, replay *ReplayInput, orderStart int, overrideConditions []string) (*Env, bool) {
	// CALCULATOR is what copyActiveSkill's second initEnv uses. It differs
	// from MAIN only in skipping the write-backs onto the build objects
	// that exist for the UI (node.finalModList, gemInstance.displayEffect,
	// group.displayLabel, item.jewelRadiusData, superseded flags,
	// group.mainActiveSkill) -- of which this port keeps only the ones a
	// later stage reads.
	if mode != ModeMain && mode != ModeCalculator {
		panic("calc: only MAIN and CALCULATOR modes are ported")
	}
	if in.ConfigInput == nil {
		in.ConfigInput = &ConfigInput{}
	}
	if in.ConfigPlaceholder == nil {
		in.ConfigPlaceholder = &ConfigInput{}
	}
	env := &Env{
		Build:               in,
		Mode:                mode,
		ConfigInput:         in.ConfigInput,
		ClassID:             in.ClassID,
		AllocOrders:         replay.AllocOrders,
		allocOrderIdx:       orderStart,
		Replay:              replay,
		GlobalCache:         replay.GlobalCache,
		ExtraRadiusNodeList: map[int]*NodeInput{},
		OverrideConditions:  overrideConditions,
	}
	// GlobalCache is cachedData[mode][uuid] in the reference. This port
	// keeps only the MAIN bucket -- the one every ported reader looks in --
	// and gives any other mode its own, so a CALCULATOR sub-environment's
	// own cacheData cannot land where a MAIN reader would find it. The
	// collapse holds as long as no ported code reads across modes; the
	// "shattershard" trigger config, which branches on env.mode, is the one
	// place that would break it, and it is still guarded.
	if mode != ModeMain {
		env.GlobalCache = map[string]*CachedSkill{}
	}
	for i, seq := range replay.NodeOrders {
		if i < len(replay.AllocOrders) {
			env.ExtraOrders = append(env.ExtraOrders, seq[len(replay.AllocOrders[i]):])
		}
	}

	modDB := modstore.NewDB(nil)
	env.ModDB = modDB
	enemyDB := modstore.NewDB(nil)
	env.EnemyDB = enemyDB
	env.ItemModDB = modstore.NewDB(nil)

	if in.ConfigEnemyLevel != nil {
		env.EnemyLevel = *in.ConfigEnemyLevel
	} else {
		env.EnemyLevel = math.Min(data.Misc.MaxEnemyLevel, in.CharacterLevel)
	}

	// Create player/enemy actors
	env.Player = &modstore.Actor{DB: modDB, Level: in.CharacterLevel, ItemList: map[string]modstore.Item{}, Resolver: gemIds{}}
	modDB.Actor = env.Player
	env.Enemy = &modstore.Actor{DB: enemyDB, Level: env.EnemyLevel, Resolver: gemIds{}}
	enemyDB.Actor = env.Enemy
	env.Player.Enemy = env.Enemy
	env.Enemy.Enemy = env.Player
	env.Enemy.Player = env.Player
	env.Player.Player = env.Player

	// Set buff mode: MAIN is always EFFECTIVE
	env.ModeBuffs = true
	env.ModeCombat = true
	env.ModeEffective = true

	// Initialise modifier database with base values
	for _, s := range [3]struct {
		name string
		base float64
	}{{"Str", in.ClassStats.BaseStr}, {"Dex", in.ClassStats.BaseDex}, {"Int", in.ClassStats.BaseInt}} {
		modDB.AddMod(newModS(s.name, modparser.Base, modparser.Num(s.base), "Base"))
	}
	modDB.Multipliers["Level"] = math.Max(1, math.Min(100, in.CharacterLevel))
	env.initModDB(modDB)
	cc := data.CharacterConstants
	resistPenalty := -60.0
	if in.ConfigInput.ResistancePenalty.Set {
		resistPenalty = in.ConfigInput.ResistancePenalty.V
	}
	modDB.AddMod(newModS("Life", modparser.Base, modparser.Num(cc["life_per_level"]), "Base", &modparser.MultiplierTag{Var: "Level", Base: opt(38.0)}))
	modDB.AddMod(newModS("Mana", modparser.Base, modparser.Num(cc["mana_per_level"]), "Base", &modparser.MultiplierTag{Var: "Level", Base: opt(34.0)}))
	modDB.AddMod(newModS("ManaRegen", modparser.Base, modparser.Num(data.Misc.ManaRegenBase), "Base", &modparser.StatTag{StatKind: modparser.TagPerStat, Stat: "Mana", Div: opt(1.0)}))
	modDB.AddMod(newModS("Devotion", modparser.Base, modparser.Num(0.0), "Base"))
	modDB.AddMod(newModS("Evasion", modparser.Base, modparser.Num(cc["base_evasion_rating"]), "Base"))
	modDB.AddMod(newModS("Accuracy", modparser.Base, modparser.Num(cc["accuracy_rating_per_level"]), "Base", &modparser.MultiplierTag{Var: "Level", Base: opt(-cc["accuracy_rating_per_level"])}))
	modDB.AddMod(newModS("CritMultiplier", modparser.Base, modparser.Num(cc["base_critical_strike_multiplier"]-100), "Base"))
	modDB.AddMod(newModS("DotMultiplier", modparser.Base, modparser.Num(cc["critical_ailment_dot_multiplier_+"]), "Base", &modparser.CondTag{Var: "CriticalStrike"}))
	modDB.AddMod(newModS("FireResist", modparser.Base, modparser.Num(resistPenalty), "Base"))
	modDB.AddMod(newModS("ColdResist", modparser.Base, modparser.Num(resistPenalty), "Base"))
	modDB.AddMod(newModS("LightningResist", modparser.Base, modparser.Num(resistPenalty), "Base"))
	modDB.AddMod(newModS("ChaosResist", modparser.Base, modparser.Num(resistPenalty), "Base"))
	modDB.AddMod(newModS("TotemFireResist", modparser.Base, modparser.Num(40.0), "Base"))
	modDB.AddMod(newModS("TotemColdResist", modparser.Base, modparser.Num(40.0), "Base"))
	modDB.AddMod(newModS("TotemLightningResist", modparser.Base, modparser.Num(40.0), "Base"))
	modDB.AddMod(newModS("TotemChaosResist", modparser.Base, modparser.Num(20.0), "Base"))
	modDB.AddMod(newModS("CritChance", modparser.Inc, modparser.Num(cc["critical_strike_chance_+%_per_power_charge"]), "Base", &modparser.MultiplierTag{Var: "PowerCharge"}))
	modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(cc["base_attack_speed_+%_per_frenzy_charge"]), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.MultiplierTag{Var: "FrenzyCharge"}))
	modDB.AddMod(newModSF("Speed", modparser.Inc, modparser.Num(cc["base_cast_speed_+%_per_frenzy_charge"]), "Base", modparser.FlagCast, modparser.KeywordNone, &modparser.MultiplierTag{Var: "FrenzyCharge"}))
	modDB.AddMod(newModS("Damage", modparser.More, modparser.Num(cc["object_inherent_damage_+%_final_per_frenzy_charge"]), "Base", &modparser.MultiplierTag{Var: "FrenzyCharge"}))
	modDB.AddMod(newModS("PhysicalDamageReduction", modparser.Base, modparser.Num(cc["physical_damage_reduction_%_per_endurance_charge"]), "Base", &modparser.MultiplierTag{Var: "EnduranceCharge"}))
	modDB.AddMod(newModS("ElementalDamageReduction", modparser.Base, modparser.Num(cc["elemental_damage_reduction_%_per_endurance_charge"]), "Base", &modparser.MultiplierTag{Var: "EnduranceCharge"}))
	modDB.AddMod(newModS("MaximumRage", modparser.Base, modparser.Num(cc["maximum_rage"]), "Base"))
	modDB.AddMod(newModS("Multiplier:GaleForce", modparser.Base, modparser.Num(0.0), "Base"))
	modDB.AddMod(newModS("MaximumGaleForce", modparser.Base, modparser.Num(10.0), "Base"))
	modDB.AddMod(newModS("MaximumFortification", modparser.Base, modparser.Num(cc["base_max_fortification"]), "Base"))
	modDB.AddMod(newModS("MaximumValour", modparser.Base, modparser.Num(50.0), "Base"))
	modDB.AddMod(newModS("Multiplier:IntensityLimit", modparser.Base, modparser.Num(3.0), "Base"))
	modDB.AddMod(newModS("Damage", modparser.Inc, modparser.Num(cc["damage_+%_per_10_rampage_stacks"]), "Base", &modparser.MultiplierTag{Var: "Rampage", Limit: opt(cc["max_rampage_stacks"] / 20), Div: opt(20.0)}))
	modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(cc["movement_velocity_+%_per_10_rampage_stacks"]), "Base", &modparser.MultiplierTag{Var: "Rampage", Limit: opt(cc["max_rampage_stacks"] / 20), Div: opt(20.0)}))
	modDB.AddMod(newModS("ActiveTrapLimit", modparser.Base, modparser.Num(cc["base_number_of_traps_allowed"]), "Base"))
	modDB.AddMod(newModS("ActiveMineLimit", modparser.Base, modparser.Num(cc["base_number_of_remote_mines_allowed"]), "Base"))
	modDB.AddMod(newModS("MineThrowCount", modparser.Base, modparser.Num(1.0), "Base"))
	modDB.AddMod(newModS("TrapThrowCount", modparser.Base, modparser.Num(1.0), "Base"))
	modDB.AddMod(newModS("ActiveBrandLimit", modparser.Base, modparser.Num(3.0), "Base"))
	modDB.AddMod(newModS("EnemyCurseLimit", modparser.Base, modparser.Num(1.0), "Base"))
	modDB.AddMod(newModS("SocketedCursesHexLimitValue", modparser.Base, modparser.Num(1.0), "Base"))
	modDB.AddMod(newModS("ProjectileCount", modparser.Base, modparser.Num(1.0), "Base"))
	modDB.AddMod(newModSF("Speed", modparser.More, modparser.Num(cc["dual_wield_inherent_attack_speed_+%_final"]), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.CondTag{Var: "DualWielding"}, &modparser.CondTag{Var: "DoubledInherentDualWieldingSpeed", Neg: true}))
	modDB.AddMod(newModSF("Speed", modparser.More, modparser.Num(2*cc["dual_wield_inherent_attack_speed_+%_final"]), "Base", modparser.FlagAttack, modparser.KeywordNone, &modparser.CondTag{Var: "DualWielding"}, &modparser.CondTag{Var: "DoubledInherentDualWieldingSpeed"}))
	modDB.AddMod(newModS("BlockChance", modparser.Base, modparser.Num(cc["inherent_block_while_dual_wielding_%"]), "Base", &modparser.CondTag{Var: "DualWielding"}, &modparser.CondTag{Var: "NoInherentBlock", Neg: true}, &modparser.CondTag{Var: "DoubledInherentDualWieldingBlock", Neg: true}))
	modDB.AddMod(newModS("BlockChance", modparser.Base, modparser.Num(2*cc["inherent_block_while_dual_wielding_%"]), "Base", &modparser.CondTag{Var: "DualWielding"}, &modparser.CondTag{Var: "NoInherentBlock", Neg: true}, &modparser.CondTag{Var: "DoubledInherentDualWieldingBlock"}))
	modDB.AddMod(newModSF("Damage", modparser.More, modparser.Num(200.0), "Base", modparser.FlagNone, modparser.KeywordBleed, &modparser.CondTag{IsActor: true, Actor: "enemy", Var: "Moving"}, &modparser.CondTag{Var: "NoExtraBleedDamageToMovingEnemy", Neg: true}))
	modDB.AddMod(newModS("Condition:BloodStance", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "SandStance", Neg: true}))
	modDB.AddMod(newModS("Condition:PrideMinEffect", modparser.Flag, modparser.Bool(true), "Base", &modparser.CondTag{Var: "PrideMaxEffect", Neg: true}))
	modDB.AddMod(newModS("PerBrutalTripleDamageChance", modparser.Base, modparser.Num(cc["chance_to_deal_triple_damage_%_per_brutal_charge"]), "Base"))
	modDB.AddMod(newModS("PerAfflictionAilmentDamage", modparser.Base, modparser.Num(cc["ailment_damage_+%_final_per_affliction_charge"]), "Base"))
	modDB.AddMod(newModS("PerAfflictionNonDamageEffect", modparser.Base, modparser.Num(cc["non_damaging_ailment_effect_+%_final_per_affliction_charge"]), "Base"))
	modDB.AddMod(newModS("PerAbsorptionElementalEnergyShieldRecoup", modparser.Base, modparser.Num(cc["elemental_damage_taken_goes_to_energy_shield_over_4_seconds_%_per_absorption_charge"]), "Base"))
	modDB.AddMod(newModS("TinctureLimit", modparser.Base, modparser.Num(1.0), "Base"))
	modDB.AddMod(newModS("ManaDegenPercentTincture", modparser.Base, modparser.Num(1.0), "Base", &modparser.MultiplierTag{Var: "EffectiveManaBurnStacks"}))
	modDB.AddMod(newModS("LifeDegenPercentTincture", modparser.Base, modparser.Num(1.0), "Base", &modparser.MultiplierTag{Var: "WeepingWoundsStacks"}))
	modDB.AddMod(newModS("PresenceRadius", modparser.Base, modparser.Num(cc["base_presence_radius"]), "Base"))

	// Add bandit mods
	switch in.ConfigInput.Bandit {
	case "Alira":
		modDB.AddMod(newModS("ElementalResist", modparser.Base, modparser.Num(15.0), "Bandit"))
	case "Kraityn":
		modDB.AddMod(newModS("MovementSpeed", modparser.Inc, modparser.Num(8.0), "Bandit"))
	case "Oak":
		modDB.AddMod(newModS("Life", modparser.Base, modparser.Num(40.0), "Bandit"))
	default:
		modDB.AddMod(newModS("ExtraPoints", modparser.Base, modparser.Num(1.0), "Bandit"))
	}

	// Add Pantheon mods
	if god := in.ConfigInput.PantheonMajorGod; god != "None" && god != "" {
		applySoulMod(modDB, data.Pantheons[god])
	}
	if god := in.ConfigInput.PantheonMinorGod; god != "None" && god != "" {
		applySoulMod(modDB, data.Pantheons[god])
	}

	// Initialise enemy modifier database
	env.initModDB(enemyDB)
	enemyDB.AddMod(newModS("Accuracy", modparser.Base, modparser.Num(data.MonsterAccuracyTable[int(env.EnemyLevel)-1]), "Base"))
	enemyDB.AddMod(newModSF("Condition:AgainstDamageOverTime", modparser.Flag, modparser.Bool(true), "Base", modparser.FlagDot, modparser.KeywordNone, &modparser.CondTag{IsActor: true, Actor: "player", Var: "Combat"}))

	// Add mods from the config tab, then the party tab
	modDB.AddList(in.ConfigModList)
	enemyDB.AddList(in.ConfigEnemyModList)
	enemyDB.AddList(in.PartyEnemyModList)

	// (specCopy caching skipped: one-shot replay)
	for _, flag := range overrideConditions {
		modDB.Conditions.Set(flag, true)
	}

	allocatedNotableCount := in.Spec.AllocatedNotableCount
	allocatedKeystoneCount := in.Spec.AllocatedKeystoneCount
	allocatedMasteryCount := in.Spec.AllocatedMasteryCount
	allocatedMasteryTypeCount := in.Spec.AllocatedMasteryTypeCount
	allocatedMasteryTypes := make(map[string]float64, len(in.Spec.AllocatedMasteryTypes))
	for k, v := range in.Spec.AllocatedMasteryTypes {
		allocatedMasteryTypes[k] = v
	}
	allocatedTattooTypes := make(map[string]float64, len(in.Spec.AllocatedTattooTypes))
	for k, v := range in.Spec.AllocatedTattooTypes {
		allocatedTattooTypes[k] = v
	}

	// Build list of passive nodes (no overrides in one-shot MAIN)
	env.AllocNodes = make(map[int]*NodeInput, len(in.Spec.AllocNodes))
	for id, node := range in.Spec.AllocNodes {
		env.AllocNodes[id] = node
	}
	env.Keystone.KeystoneMods = in.Spec.KeystoneMap
	initialList, _ := env.buildModListForNodeList(true)
	env.InitialNodeModDB = initialList
	modstore.MergeKeystones(&env.Keystone, initialList)

	if allocatedNotableCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedNotable", modparser.Base, modparser.Num(allocatedNotableCount)))
	}
	if allocatedKeystoneCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedKeystone", modparser.Base, modparser.Num(allocatedKeystoneCount)))
	}
	if allocatedMasteryCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedMastery", modparser.Base, modparser.Num(allocatedMasteryCount)))
	}
	if allocatedMasteryTypeCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedMasteryType", modparser.Base, modparser.Num(allocatedMasteryTypeCount)))
	}
	if allocatedMasteryTypes["Life Mastery"] > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedLifeMastery", modparser.Base, modparser.Num(allocatedMasteryTypes["Life Mastery"])))
	}
	for typ, count := range allocatedTattooTypes {
		modDB.Multipliers[typ] = count
	}

	// Resolve socket group gem references (the fixture stores ids; the
	// items stage's socket counting already reads gemData)
	for _, group := range in.SkillsTab.SocketGroups {
		for _, gem := range group.GemList {
			if gem.GemDataID != nil {
				gem.GemData = data.Gems[*gem.GemDataID]
				if gem.GemData == nil {
					panic("calc: unknown gemDataId " + *gem.GemDataID)
				}
			}
			if gem.GrantedEffectID != nil {
				gem.GrantedEffect = data.Skills[*gem.GrantedEffectID]
				if gem.GrantedEffect == nil {
					panic("calc: unknown grantedEffectId " + *gem.GrantedEffectID)
				}
			}
		}
	}

	// Build and merge item modifiers (CalcSetup L697-1280)
	env.buildItems()

	// Merge env.itemModDB with env.ModDB
	mergeDB(modDB, env.ItemModDB)

	// Add granted passives (e.g., amulet anoints)
	env.GrantedPassives = map[int]bool{}
	for _, pv := range modDB.List(nil, "GrantedPassive") {
		passiveStr, ok := pv.(modparser.Str)
		if !ok {
			continue
		}
		passive := string(passiveStr)
		node := replay.GrantedPassiveNodes[passive]
		if node == nil {
			// name resolved through none of the tree maps
			continue
		}
		env.AllocNodes[int(node.ID)] = node
		env.GrantedPassives[int(node.ID)] = true
	}

	// Add granted ascendancy node (e.g., Forbidden Flame/Flesh combo)
	type ascMatch struct {
		side    string
		matched bool
	}
	matchedName := map[string]*ascMatch{}
	for _, v := range modDB.List(nil, "GrantedAscendancyNode") {
		ascTbl, ok := v.(modparser.AscendancyNodeRef)
		if !ok {
			panic("calc: non-AscendancyNodeRef value in GrantedAscendancyNode list (the Lua errors)")
		}
		name := ascTbl.Name
		if m := matchedName[name]; m != nil && m.side != ascTbl.Side && !m.matched {
			m.matched = true
			node := replay.GrantedAscendancyNodes[name]
			if node != nil {
				if condClass(env.ItemModDB.Conditions, "ForbiddenFlesh") == in.CurClassName &&
					condClass(env.ItemModDB.Conditions, "ForbiddenFlame") == in.CurClassName {
					env.AllocNodes[int(node.ID)] = node
					env.GrantedPassives[int(node.ID)] = true
				}
			}
		} else {
			matchedName[name] = &ascMatch{side: ascTbl.Side}
		}
	}

	// Merge modifiers for allocated passives
	{
		modList, explodeSources := env.buildModListForNodeList(true)
		modDB.AddList(modList.Mods)
		env.ExplodeSources = append(explodeSources, env.ExplodeSources...)
	}
	if got := modDB.Tabulate(modparser.List, nil, "ExtraJewelFunc"); len(got) > 0 {
		panic("calc: ExtraJewelFunc re-entry reached - items stage not ported")
	}

	// Find skills granted by tree nodes. pairs(env.allocNodes) over the
	// same table state as the tree merge — reuse that captured order.
	for _, id := range env.AllocOrders[len(env.AllocOrders)-1] {
		node := env.AllocNodes[id]
		for _, skill := range node.GrantedSkills {
			granted := skill
			granted.SourceNode = node
			env.GrantedSkillsNodes = append(env.GrantedSkillsNodes, granted)
		}
	}
	env.GrantedSkills = append(append([]GrantedSkill{}, env.GrantedSkillsNodes...), env.GrantedSkillsItems...)

	// Skill and support processing (CalcSetup L1349-1871)
	if env.buildSkillsStage() {
		return env, true // Energy Blade re-entry
	}

	return env, false
}

// geHasGlobalEffect is `grantedEffect.hasGlobalEffect` with the calc-time
// stamps the reference writes onto the shared tables read from the per-env
// overlay instead.
func (env *Env) geHasGlobalEffect(ge *data.GrantedEffect) bool {
	return ge.HasGlobalEffect || env.globalEffectOverlay[ge]
}
