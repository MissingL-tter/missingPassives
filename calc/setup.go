// Port of .archive/src/Modules/CalcSetup.lua, staged: this file covers
// initModDB and initEnv through the tree merge (the stages a build without
// items or skills exercises). The items loop and the skill/support stages
// panic loudly when an input would reach them, instead of silently
// diverging from the archive.
package calc

import (
	"fmt"
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
	Raw        map[string]any // the item's raw grantedSkills entry
}

// Item is the runtime item: the fixture payload plus mutable state.
// Implements modstore.Item for the ItemCondition eval tag.
type Item struct {
	In *ItemInput
	D  *data.Data // itemTagSpecial pools for FindModifierSubstring
	// jewelData.limitDisabled
	JewelLimitDisabled bool
}

func (it *Item) Name() string     { return it.In.Name }
func (it *Item) ItemType() string { return it.In.Type }
func (it *Item) Rarity() string   { return it.In.Rarity }
func (it *Item) Corrupted() bool  { return it.In.Corrupted != nil && *it.In.Corrupted }
func (it *Item) Shaper() bool     { return it.In.Shaper != nil && *it.In.Shaper }
func (it *Item) Elder() bool      { return it.In.Elder != nil && *it.In.Elder }

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
			if slotPats := it.D.ItemTagSpecialExclusionPattern[substring]; slotPats != nil {
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
		if slotPats := it.D.ItemTagSpecial[substring]; slotPats != nil {
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

// Env mirrors the slice of the Lua env the ported stages populate.
type Env struct {
	Build       *BuildInput
	Data        *data.Data
	Mode        string
	ConfigInput map[string]any

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
	AllocOrders      [][]int
	// ExtraOrders holds, per buildModListForNodeList call, the captured
	// pairs() order over env.extraRadiusNodeList (the node-call sequence
	// tail beyond the allocated nodes).
	ExtraOrders      [][]int
	Replay           *ReplayInput
	allocOrderIdx    int
	AllocNodes       map[int]*NodeInput
	InitialNodeModDB *modstore.List
	Keystone         modstore.KeystoneEnv

	RadiusJewelList     []*RadiusJewel
	ExtraRadiusNodeList map[int]*NodeInput

	GrantedSkillsNodes []GrantedSkill
	GrantedSkillsItems []GrantedSkill
	GrantedSkills      []GrantedSkill
	ExplodeSources     []any
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
	AuxSkillList             []*ActiveSkill
	RequirementsTableGems    []GemRequirement
	geFromItemMark           map[*data.GrantedEffect]bool
	slotsByName              map[string]*SlotInput
	// statMapOverlay memoizes lazy skillStatMap copies per granted effect
	// (the reference memoizes into the shared skill tables; per-env keeps
	// the game-data canon pristine, same values).
	statMapOverlay map[*data.GrantedEffect]map[string]*data.StatMapEntry
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
	CurseSlots                  []any

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
	e := data.LazyStatMapCopy(ge, key)
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

func newMod(name, typ string, value any, rest ...any) *modparser.Mod {
	return modparser.NewMod(name, typ, value, rest...)
}

// initModDB ports calcs.initModDB: stats and conditions common to all
// actors, in the reference's statement order.
func (env *Env) initModDB(modDB *modstore.DB) {
	cc := env.Data.CharacterConstants
	tag := func(kv modparser.Tag) modparser.Tag { return kv }
	add := func(m *modparser.Mod) { modDB.AddMod(m) }
	add(newMod("FireResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("ColdResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("LightningResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("ChaosResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("TotemFireResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("TotemColdResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("TotemLightningResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("TotemChaosResistMax", "BASE", cc["base_maximum_all_resistances_%"], "Base"))
	add(newMod("BlockChanceMax", "BASE", cc["maximum_block_%"], "Base"))
	add(newMod("SpellBlockChanceMax", "BASE", cc["base_maximum_spell_block_%"], "Base"))
	add(newMod("SpellDodgeChanceMax", "BASE", 75.0, "Base"))
	add(newMod("ChargeDuration", "BASE", 10.0, "Base"))
	add(newMod("PowerChargesMax", "BASE", cc["max_power_charges"], "Base"))
	add(newMod("FrenzyChargesMax", "BASE", cc["max_frenzy_charges"], "Base"))
	add(newMod("EnduranceChargesMax", "BASE", cc["max_endurance_charges"], "Base"))
	add(newMod("SiphoningChargesMax", "BASE", 0.0, "Base"))
	add(newMod("ChallengerChargesMax", "BASE", 0.0, "Base"))
	add(newMod("BlitzChargesMax", "BASE", 0.0, "Base"))
	add(newMod("InspirationChargesMax", "BASE", cc["maximum_righteous_charges"], "Base"))
	add(newMod("CrabBarriersMax", "BASE", 0.0, "Base"))
	add(newMod("BrutalChargesMax", "BASE", 0.0, "Base"))
	add(newMod("BrineChargesMax", "BASE", 0.0, "Base"))
	add(newMod("PhysicalDamageGainAsCold", "BASE", cc["physical_damage_%_to_add_as_cold_per_brine_charge"], "Base", tag(modparser.Tag{"type": "Multiplier", "var": "BrineCharge"})))
	add(newMod("PhysicalDamageGainAsLightning", "BASE", cc["physical_damage_%_to_add_as_lightning_per_brine_charge"], "Base", tag(modparser.Tag{"type": "Multiplier", "var": "BrineCharge"})))
	add(newMod("AbsorptionChargesMax", "BASE", 0.0, "Base"))
	add(newMod("AfflictionChargesMax", "BASE", 0.0, "Base"))
	add(newMod("BloodChargesMax", "BASE", cc["maximum_blood_scythe_charges"], "Base"))
	add(newMod("MaxLifeLeechRate", "BASE", cc["maximum_life_leech_rate_%_per_minute"]/60, "Base"))
	add(newMod("MaxManaLeechRate", "BASE", cc["maximum_mana_leech_rate_%_per_minute"]/60, "Base"))
	add(newMod("ImpaleStacksMax", "BASE", cc["impaled_debuff_number_of_reflected_hits"], "Base"))
	add(newMod("SoulEaterMax", "BASE", cc["soul_eater_maximum_stacks"], "Base"))
	add(newMod("BleedStacksMax", "BASE", 1.0, "Base"))
	add(newMod("MaxEnergyShieldLeechRate", "BASE", 10.0, "Base"))
	add(newMod("MaxLifeLeechInstance", "BASE", cc["maximum_life_leech_amount_per_leech_%_max_life"], "Base"))
	add(newMod("MaxManaLeechInstance", "BASE", cc["maximum_mana_leech_amount_per_leech_%_max_mana"], "Base"))
	add(newMod("MaxEnergyShieldLeechInstance", "BASE", cc["maximum_energy_shield_leech_amount_per_leech_%_max_energy_shield"], "Base"))
	add(newMod("TrapThrowingTime", "BASE", 0.6, "Base"))
	add(newMod("MineLayingTime", "BASE", 0.3, "Base"))
	add(newMod("WarcryCastTime", "BASE", 0.8, "Base"))
	add(newMod("TotemPlacementTime", "BASE", 0.6, "Base"))
	add(newMod("BallistaPlacementTime", "BASE", 0.5, "Base"))
	add(newMod("ActiveTotemLimit", "BASE", cc["base_number_of_totems_allowed"], "Base"))
	add(newMod("ShockStacksMax", "BASE", 1.0, "Base"))
	add(newMod("ScorchStacksMax", "BASE", 1.0, "Base"))
	add(newMod("MovementSpeed", "INC", -30.0, "Base", tag(modparser.Tag{"type": "Condition", "var": "Maimed"})))
	add(newMod("DamageTaken", "INC", 10.0, "Base", modparser.ModFlag.Attack, tag(modparser.Tag{"type": "Condition", "var": "Intimidated"})))
	add(newMod("DamageTaken", "INC", 10.0, "Base", modparser.ModFlag.Attack, tag(modparser.Tag{"type": "Condition", "var": "Intimidated", "neg": true}), tag(modparser.Tag{"type": "Condition", "var": "Party:Intimidated"})))
	add(newMod("DamageTaken", "INC", 10.0, "Base", modparser.ModFlag.Spell, tag(modparser.Tag{"type": "Condition", "var": "Unnerved"})))
	add(newMod("DamageTaken", "INC", 10.0, "Base", modparser.ModFlag.Spell, tag(modparser.Tag{"type": "Condition", "var": "Unnerved", "neg": true}), tag(modparser.Tag{"type": "Condition", "var": "Party:Unnerved"})))
	add(newMod("Damage", "MORE", -10.0, "Base", tag(modparser.Tag{"type": "Condition", "var": "Debilitated"}), tag(modparser.Tag{"type": "GlobalEffect", "effectName": "Debilitated", "effectType": "Debuff"})))
	add(newMod("MovementSpeed", "MORE", -20.0, "Base", tag(modparser.Tag{"type": "Condition", "var": "Debilitated"}), tag(modparser.Tag{"type": "GlobalEffect", "effectName": "Debilitated", "effectType": "Debuff"})))
	add(newMod("Damage", "MORE", -10.0, "Base", tag(modparser.Tag{"type": "Condition", "var": "MalignantMadness"}), tag(modparser.Tag{"type": "GlobalEffect", "effectName": "Malignant Madness", "effectType": "Debuff"})))
	add(newMod("ActionSpeed", "MORE", -10.0, "Base", tag(modparser.Tag{"type": "Condition", "var": "MalignantMadness"}), tag(modparser.Tag{"type": "GlobalEffect", "effectName": "Malignant Madness", "effectType": "Debuff"})))
	add(newMod("Condition:Burning", "FLAG", true, "Base", tag(modparser.Tag{"type": "IgnoreCond"}), tag(modparser.Tag{"type": "Condition", "var": "Ignited"})))
	add(newMod("Condition:Poisoned", "FLAG", true, "Base", tag(modparser.Tag{"type": "IgnoreCond"}), tag(modparser.Tag{"type": "MultiplierThreshold", "var": "PoisonStack", "threshold": 1.0})))
	add(newMod("Blind", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Blinded"})))
	add(newMod("Chill", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Chilled"})))
	add(newMod("Freeze", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Frozen"})))
	add(newMod("Fortify", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Fortify"})))
	add(newMod("Fortified", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Fortified"})))
	add(newMod("Excommunicated", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Excommunicated"})))
	add(newMod("Fanaticism", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Fanaticism"})))
	add(newMod("Onslaught", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Onslaught"})))
	add(newMod("UnholyMight", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "UnholyMight"})))
	add(newMod("ChaoticMight", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "ChaoticMight"})))
	add(newMod("Tailwind", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Tailwind"})))
	add(newMod("Adrenaline", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Adrenaline"})))
	add(newMod("AccelerationShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "AccelerationShrine"})))
	add(newMod("BrutalShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "BrutalShrine"})))
	add(newMod("DiamondShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "DiamondShrine"})))
	add(newMod("DivineShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "DivineShrine"})))
	add(newMod("EchoingShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "EchoingShrine"})))
	add(newMod("GloomShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "GloomShrine"})))
	add(newMod("GreaterFreezingShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "GreaterFreezingShrine"})))
	add(newMod("GreaterShockingShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "GreaterShockingShrine"})))
	add(newMod("GreaterSkeletalShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "GreaterSkeletalShrine"})))
	add(newMod("ImpenetrableShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "ImpenetrableShrine"})))
	add(newMod("MassiveShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "MassiveShrine"})))
	add(newMod("ReplenishingShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "ReplenishingShrine"})))
	add(newMod("ResistanceShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "ResistanceShrine"})))
	add(newMod("ResonatingShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "ResonatingShrine"})))
	add(newMod("LesserAccelerationShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LesserAccelerationShrine"}), tag(modparser.Tag{"type": "Condition", "var": "AccelerationShrine", "neg": true})))
	add(newMod("LesserBrutalShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LesserBrutalShrine"}), tag(modparser.Tag{"type": "Condition", "var": "BrutalShrine", "neg": true})))
	add(newMod("LesserImpenetrableShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LesserImpenetrableShrine"}), tag(modparser.Tag{"type": "Condition", "var": "ImpenetrableShrine", "neg": true})))
	add(newMod("LesserMassiveShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LesserMassiveShrine"}), tag(modparser.Tag{"type": "Condition", "var": "MassiveShrine", "neg": true})))
	add(newMod("LesserReplenishingShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LesserReplenishingShrine"}), tag(modparser.Tag{"type": "Condition", "var": "ReplenishingShrine", "neg": true})))
	add(newMod("LesserResistanceShrine", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LesserResistanceShrine"}), tag(modparser.Tag{"type": "Condition", "var": "ResistanceShrine", "neg": true})))
	add(newMod("AlchemistsGenius", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "AlchemistsGenius"})))
	add(newMod("LuckyHits", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "LuckyHits"})))
	add(newMod("Convergence", "FLAG", true, "Base", tag(modparser.Tag{"type": "Condition", "var": "Convergence"})))
	add(newMod("PhysicalDamageReduction", "BASE", -15.0, "Base", tag(modparser.Tag{"type": "Condition", "var": "Crushed"})))
	add(newMod("CritChanceCap", "BASE", 100.0, "Base"))
	modDB.Conditions["Buffed"] = env.ModeBuffs
	modDB.Conditions["Combat"] = env.ModeCombat
	modDB.Conditions["Effective"] = env.ModeEffective
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
			mods, extra := modparser.Parse(soulMod.Line)
			if mods != nil && extra == "" {
				list := make([]*modparser.Mod, len(mods))
				for i, m := range mods {
					mod := m.(*modparser.Mod)
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
func (env *Env) buildModListForNode(node *NodeInput) (*modstore.List, any) {
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
			modList.AddMod(v.(modparser.Tag)["mod"].(*modparser.Mod))
		}
	}

	node.GrantedSkills = nil
	for _, v := range modList.List(nil, "ExtraSkill") {
		skill := v.(modparser.Tag)
		if skill["name"] != "Unknown" {
			node.GrantedSkills = append(node.GrantedSkills, GrantedSkill{
				SkillID:    skill["skillId"].(string),
				Level:      anyNum(skill["level"]),
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
		panic("calc: more buildModListForNodeList calls than captured allocOrders")
	}
	order := env.AllocOrders[env.allocOrderIdx]
	env.allocOrderIdx++
	return order
}

// buildModListForNodeList ports calcs.buildModListForNodeList over the
// captured pairs() orders.
func (env *Env) buildModListForNodeList(finishJewels bool) (*modstore.List, []any) {
	callIdx := env.allocOrderIdx
	// Initialise radius jewels
	for _, rad := range env.RadiusJewelList {
		for k := range rad.Data {
			delete(rad.Data, k)
		}
		rad.Data["modSource"] = fmt.Sprintf("Tree:%d", rad.NodeID)
	}

	// Add node modifiers
	modList := modstore.NewList(nil)
	explodeSources := []any{}
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
}

// InitEnv ports calcs.initEnv for the one-shot MAIN mode over a fixture
// BuildInput, including the Energy Blade re-entry (the reference restarts
// initEnv with override.conditions when an enabled Energy Blade gem is
// found).
func InitEnv(d *data.Data, in *BuildInput, mode string, replay *ReplayInput) *Env {
	var overrideConditions []string
	orderStart := 0
	for {
		env, restart := initEnvPass(d, in, mode, replay, orderStart, overrideConditions)
		if !restart {
			return env
		}
		orderStart = env.allocOrderIdx
		overrideConditions = append(overrideConditions, "AffectedByEnergyBlade")
	}
}

func initEnvPass(d *data.Data, in *BuildInput, mode string, replay *ReplayInput, orderStart int, overrideConditions []string) (*Env, bool) {
	if mode != "MAIN" {
		panic("calc: only MAIN mode is ported")
	}
	// Wire the calcLib externals mod evaluation reaches into.
	modstore.Externals.GemIsType = func(gem any, keyword string) bool {
		g, ok := gem.(*data.Gem)
		if !ok {
			panic("calc: non-gem skillGem in SocketedIn eval")
		}
		return GemIsType(d, g, keyword, false)
	}
	modstore.Externals.GetGameIdFromGemName = func(name string, dropVaal bool) (string, bool) {
		id := GetGameIdFromGemName(d, name, dropVaal)
		return id, id != ""
	}
	env := &Env{
		Build:               in,
		Data:                d,
		Mode:                mode,
		ConfigInput:         in.ConfigInput,
		ClassID:             in.ClassID,
		AllocOrders:         replay.AllocOrders,
		allocOrderIdx:       orderStart,
		Replay:              replay,
		GlobalCache:         replay.GlobalCache,
		ExtraRadiusNodeList: map[int]*NodeInput{},
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
		env.EnemyLevel = math.Min(d.Misc.MaxEnemyLevel, in.CharacterLevel)
	}

	// Create player/enemy actors
	env.Player = &modstore.Actor{DB: modDB, Level: in.CharacterLevel, ItemList: map[string]modstore.Item{}}
	modDB.Actor = env.Player
	env.Enemy = &modstore.Actor{DB: enemyDB, Level: env.EnemyLevel}
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
		modDB.AddMod(newMod(s.name, "BASE", s.base, "Base"))
	}
	modDB.Multipliers["Level"] = math.Max(1, math.Min(100, in.CharacterLevel))
	env.initModDB(modDB)
	cc := d.CharacterConstants
	resistPenalty := -60.0
	if v, ok := in.ConfigInput["resistancePenalty"]; ok && truthy(v) {
		resistPenalty = anyNum(v)
	}
	modDB.AddMod(newMod("Life", "BASE", cc["life_per_level"], "Base", modparser.Tag{"type": "Multiplier", "var": "Level", "base": 38.0}))
	modDB.AddMod(newMod("Mana", "BASE", cc["mana_per_level"], "Base", modparser.Tag{"type": "Multiplier", "var": "Level", "base": 34.0}))
	modDB.AddMod(newMod("ManaRegen", "BASE", d.Misc.ManaRegenBase, "Base", modparser.Tag{"type": "PerStat", "stat": "Mana", "div": 1.0}))
	modDB.AddMod(newMod("Devotion", "BASE", 0.0, "Base"))
	modDB.AddMod(newMod("Evasion", "BASE", cc["base_evasion_rating"], "Base"))
	modDB.AddMod(newMod("Accuracy", "BASE", cc["accuracy_rating_per_level"], "Base", modparser.Tag{"type": "Multiplier", "var": "Level", "base": -cc["accuracy_rating_per_level"]}))
	modDB.AddMod(newMod("CritMultiplier", "BASE", cc["base_critical_strike_multiplier"]-100, "Base"))
	modDB.AddMod(newMod("DotMultiplier", "BASE", cc["critical_ailment_dot_multiplier_+"], "Base", modparser.Tag{"type": "Condition", "var": "CriticalStrike"}))
	modDB.AddMod(newMod("FireResist", "BASE", resistPenalty, "Base"))
	modDB.AddMod(newMod("ColdResist", "BASE", resistPenalty, "Base"))
	modDB.AddMod(newMod("LightningResist", "BASE", resistPenalty, "Base"))
	modDB.AddMod(newMod("ChaosResist", "BASE", resistPenalty, "Base"))
	modDB.AddMod(newMod("TotemFireResist", "BASE", 40.0, "Base"))
	modDB.AddMod(newMod("TotemColdResist", "BASE", 40.0, "Base"))
	modDB.AddMod(newMod("TotemLightningResist", "BASE", 40.0, "Base"))
	modDB.AddMod(newMod("TotemChaosResist", "BASE", 20.0, "Base"))
	modDB.AddMod(newMod("CritChance", "INC", cc["critical_strike_chance_+%_per_power_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "PowerCharge"}))
	modDB.AddMod(newMod("Speed", "INC", cc["base_attack_speed_+%_per_frenzy_charge"], "Base", modparser.ModFlag.Attack, modparser.Tag{"type": "Multiplier", "var": "FrenzyCharge"}))
	modDB.AddMod(newMod("Speed", "INC", cc["base_cast_speed_+%_per_frenzy_charge"], "Base", modparser.ModFlag.Cast, modparser.Tag{"type": "Multiplier", "var": "FrenzyCharge"}))
	modDB.AddMod(newMod("Damage", "MORE", cc["object_inherent_damage_+%_final_per_frenzy_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "FrenzyCharge"}))
	modDB.AddMod(newMod("PhysicalDamageReduction", "BASE", cc["physical_damage_reduction_%_per_endurance_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "EnduranceCharge"}))
	modDB.AddMod(newMod("ElementalDamageReduction", "BASE", cc["elemental_damage_reduction_%_per_endurance_charge"], "Base", modparser.Tag{"type": "Multiplier", "var": "EnduranceCharge"}))
	modDB.AddMod(newMod("MaximumRage", "BASE", cc["maximum_rage"], "Base"))
	modDB.AddMod(newMod("Multiplier:GaleForce", "BASE", 0.0, "Base"))
	modDB.AddMod(newMod("MaximumGaleForce", "BASE", 10.0, "Base"))
	modDB.AddMod(newMod("MaximumFortification", "BASE", cc["base_max_fortification"], "Base"))
	modDB.AddMod(newMod("MaximumValour", "BASE", 50.0, "Base"))
	modDB.AddMod(newMod("Multiplier:IntensityLimit", "BASE", 3.0, "Base"))
	modDB.AddMod(newMod("Damage", "INC", cc["damage_+%_per_10_rampage_stacks"], "Base", modparser.Tag{"type": "Multiplier", "var": "Rampage", "limit": cc["max_rampage_stacks"] / 20, "div": 20.0}))
	modDB.AddMod(newMod("MovementSpeed", "INC", cc["movement_velocity_+%_per_10_rampage_stacks"], "Base", modparser.Tag{"type": "Multiplier", "var": "Rampage", "limit": cc["max_rampage_stacks"] / 20, "div": 20.0}))
	modDB.AddMod(newMod("ActiveTrapLimit", "BASE", cc["base_number_of_traps_allowed"], "Base"))
	modDB.AddMod(newMod("ActiveMineLimit", "BASE", cc["base_number_of_remote_mines_allowed"], "Base"))
	modDB.AddMod(newMod("MineThrowCount", "BASE", 1.0, "Base"))
	modDB.AddMod(newMod("TrapThrowCount", "BASE", 1.0, "Base"))
	modDB.AddMod(newMod("ActiveBrandLimit", "BASE", 3.0, "Base"))
	modDB.AddMod(newMod("EnemyCurseLimit", "BASE", 1.0, "Base"))
	modDB.AddMod(newMod("SocketedCursesHexLimitValue", "BASE", 1.0, "Base"))
	modDB.AddMod(newMod("ProjectileCount", "BASE", 1.0, "Base"))
	modDB.AddMod(newMod("Speed", "MORE", cc["dual_wield_inherent_attack_speed_+%_final"], "Base", modparser.ModFlag.Attack, modparser.Tag{"type": "Condition", "var": "DualWielding"}, modparser.Tag{"type": "Condition", "var": "DoubledInherentDualWieldingSpeed", "neg": true}))
	modDB.AddMod(newMod("Speed", "MORE", 2*cc["dual_wield_inherent_attack_speed_+%_final"], "Base", modparser.ModFlag.Attack, modparser.Tag{"type": "Condition", "var": "DualWielding"}, modparser.Tag{"type": "Condition", "var": "DoubledInherentDualWieldingSpeed"}))
	modDB.AddMod(newMod("BlockChance", "BASE", cc["inherent_block_while_dual_wielding_%"], "Base", modparser.Tag{"type": "Condition", "var": "DualWielding"}, modparser.Tag{"type": "Condition", "var": "NoInherentBlock", "neg": true}, modparser.Tag{"type": "Condition", "var": "DoubledInherentDualWieldingBlock", "neg": true}))
	modDB.AddMod(newMod("BlockChance", "BASE", 2*cc["inherent_block_while_dual_wielding_%"], "Base", modparser.Tag{"type": "Condition", "var": "DualWielding"}, modparser.Tag{"type": "Condition", "var": "NoInherentBlock", "neg": true}, modparser.Tag{"type": "Condition", "var": "DoubledInherentDualWieldingBlock"}))
	modDB.AddMod(newMod("Damage", "MORE", 200.0, "Base", int64(0), modparser.KeywordFlag.Bleed, modparser.Tag{"type": "ActorCondition", "actor": "enemy", "var": "Moving"}, modparser.Tag{"type": "Condition", "var": "NoExtraBleedDamageToMovingEnemy", "neg": true}))
	modDB.AddMod(newMod("Condition:BloodStance", "FLAG", true, "Base", modparser.Tag{"type": "Condition", "var": "SandStance", "neg": true}))
	modDB.AddMod(newMod("Condition:PrideMinEffect", "FLAG", true, "Base", modparser.Tag{"type": "Condition", "var": "PrideMaxEffect", "neg": true}))
	modDB.AddMod(newMod("PerBrutalTripleDamageChance", "BASE", cc["chance_to_deal_triple_damage_%_per_brutal_charge"], "Base"))
	modDB.AddMod(newMod("PerAfflictionAilmentDamage", "BASE", cc["ailment_damage_+%_final_per_affliction_charge"], "Base"))
	modDB.AddMod(newMod("PerAfflictionNonDamageEffect", "BASE", cc["non_damaging_ailment_effect_+%_final_per_affliction_charge"], "Base"))
	modDB.AddMod(newMod("PerAbsorptionElementalEnergyShieldRecoup", "BASE", cc["elemental_damage_taken_goes_to_energy_shield_over_4_seconds_%_per_absorption_charge"], "Base"))
	modDB.AddMod(newMod("TinctureLimit", "BASE", 1.0, "Base"))
	modDB.AddMod(newMod("ManaDegenPercentTincture", "BASE", 1.0, "Base", modparser.Tag{"type": "Multiplier", "var": "EffectiveManaBurnStacks"}))
	modDB.AddMod(newMod("LifeDegenPercentTincture", "BASE", 1.0, "Base", modparser.Tag{"type": "Multiplier", "var": "WeepingWoundsStacks"}))
	modDB.AddMod(newMod("PresenceRadius", "BASE", cc["base_presence_radius"], "Base"))

	// Add bandit mods
	switch in.ConfigInput["bandit"] {
	case "Alira":
		modDB.AddMod(newMod("ElementalResist", "BASE", 15.0, "Bandit"))
	case "Kraityn":
		modDB.AddMod(newMod("MovementSpeed", "INC", 8.0, "Bandit"))
	case "Oak":
		modDB.AddMod(newMod("Life", "BASE", 40.0, "Bandit"))
	default:
		modDB.AddMod(newMod("ExtraPoints", "BASE", 1.0, "Bandit"))
	}

	// Add Pantheon mods
	if god, _ := in.ConfigInput["pantheonMajorGod"].(string); god != "None" && god != "" {
		applySoulMod(modDB, d.Pantheons[god])
	}
	if god, _ := in.ConfigInput["pantheonMinorGod"].(string); god != "None" && god != "" {
		applySoulMod(modDB, d.Pantheons[god])
	}

	// Initialise enemy modifier database
	env.initModDB(enemyDB)
	enemyDB.AddMod(newMod("Accuracy", "BASE", d.MonsterAccuracyTable[int(env.EnemyLevel)-1], "Base"))
	enemyDB.AddMod(newMod("Condition:AgainstDamageOverTime", "FLAG", true, "Base", modparser.ModFlag.Dot, modparser.Tag{"type": "ActorCondition", "actor": "player", "var": "Combat"}))

	// Add mods from the config tab, then the party tab
	modDB.AddList(in.ConfigModList)
	enemyDB.AddList(in.ConfigEnemyModList)
	enemyDB.AddList(in.PartyEnemyModList)

	// (specCopy caching skipped: one-shot replay)
	for _, flag := range overrideConditions {
		modDB.Conditions[flag] = true
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
		modDB.AddMod(newMod("Multiplier:AllocatedNotable", "BASE", allocatedNotableCount))
	}
	if allocatedKeystoneCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedKeystone", "BASE", allocatedKeystoneCount))
	}
	if allocatedMasteryCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedMastery", "BASE", allocatedMasteryCount))
	}
	if allocatedMasteryTypeCount > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedMasteryType", "BASE", allocatedMasteryTypeCount))
	}
	if allocatedMasteryTypes["Life Mastery"] > 0 {
		modDB.AddMod(newMod("Multiplier:AllocatedLifeMastery", "BASE", allocatedMasteryTypes["Life Mastery"]))
	}
	for typ, count := range allocatedTattooTypes {
		modDB.Multipliers[typ] = count
	}

	// Resolve socket group gem references (the fixture stores ids; the
	// items stage's socket counting already reads gemData)
	for _, group := range in.SkillsTab.SocketGroups {
		for _, gem := range group.GemList {
			if gem.GemDataID != nil {
				gem.GemData = d.Gems[*gem.GemDataID]
				if gem.GemData == nil {
					panic("calc: unknown gemDataId " + *gem.GemDataID)
				}
			}
			if gem.GrantedEffectID != nil {
				gem.GrantedEffect = d.Skills[*gem.GrantedEffectID]
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
		passive, ok := pv.(string)
		if !ok {
			continue
		}
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
		ascTbl, _ := v.(modparser.Tag)
		name := str(ascTbl["name"])
		if m := matchedName[name]; m != nil && m.side != str(ascTbl["side"]) && !m.matched {
			m.matched = true
			node := replay.GrantedAscendancyNodes[name]
			if node != nil {
				if env.ItemModDB.Conditions["ForbiddenFlesh"] == in.CurClassName &&
					env.ItemModDB.Conditions["ForbiddenFlame"] == in.CurClassName {
					env.AllocNodes[int(node.ID)] = node
					env.GrantedPassives[int(node.ID)] = true
				}
			}
		} else {
			matchedName[name] = &ascMatch{side: str(ascTbl["side"])}
		}
	}

	// Merge modifiers for allocated passives
	{
		modList, explodeSources := env.buildModListForNodeList(true)
		modDB.AddList(modList.Mods)
		env.ExplodeSources = append(explodeSources, env.ExplodeSources...)
	}
	if got := modDB.Tabulate("LIST", nil, "ExtraJewelFunc"); len(got) > 0 {
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
