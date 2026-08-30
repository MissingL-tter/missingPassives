package modparser

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Radius jewel functions — ModParser.lua:17-39, 6096-6538. The parser embeds
// these closures in JewelFunc/ExtraJewelFunc modifiers; the calculation engine
// runs them per tree node. They are ported against the narrow interfaces below
// so the package stays self-contained until the mod-store port exists.
//
// Keys are the jewels' exact mod text (matched by whole-line lookup); the
// parametric entries are keyed by a Go regular expression instead and wrapped
// in jewelFactory, which also marks them for pattern matching at parse time.

// JewelStoreWriter is the slice of ModStore/ModList the jewel functions
// use; the calc engine adapts its mod list to it.
type JewelStoreWriter interface {
	AddMod(m *Mod)
	MergeMod(m *Mod)
	AddList(list []*Mod)
	Sum(typ ModType, names ...string) float64
	Mods() []*Mod
}

// JewelNodeRef is what the functions read from a passive tree node. A nil
// node is the reference's final "apply" call after the radius walk.
type JewelNodeRef interface {
	ConqueredBy() bool
	Type() string
	IsTattoo() bool
	ModList() []*Mod
}

// JewelFuncTag is the per-jewel scratch table the reference threads through
// each call (data): the mod source, running attribute sums, the collected
// mod list of the grants-all-bonuses functions, and per-function tables
// for entries that combine several functions.
type JewelFuncTag struct {
	ModSource string
	Stats     map[string]float64
	ModList   []*Mod
	FuncData  []*JewelFuncTag
}

// AddStat accumulates a running attribute sum.
func (d *JewelFuncTag) AddStat(name string, v float64) {
	if d.Stats == nil {
		d.Stats = map[string]float64{}
	}
	d.Stats[name] += v
}

// Stat reads a running sum (0 when never accumulated).
func (d *JewelFuncTag) Stat(name string) float64 { return d.Stats[name] }

// HasStat reports whether a sum was ever accumulated (the reference's
// nil check on the data field).
func (d *JewelFuncTag) HasStat(name string) bool {
	_, ok := d.Stats[name]
	return ok
}

// JewelNodeFn mirrors the Lua signature function(node, out, data).
type JewelNodeFn func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag)

type jewelNodeFunc = JewelNodeFn

// jewelFactory is a parse-time factory: given the pattern's captures it
// returns the node function to embed.
type jewelFactory func(c caps) jewelNodeFunc

// jewelValue is a jewel-table value: a ready node function, a parse-time
// factory, or — for the one line that applies two independent threshold
// functions — a sequence of node functions.
type jewelValue interface{ isJewelValue() }

type jewelFuncSeq []JewelNodeFn

func (JewelNodeFn) isJewelValue()  {}
func (jewelFactory) isJewelValue() {}
func (jewelFuncSeq) isJewelValue() {}

// derived builds a mod from another's positional parts, as
// `mod.source, mod.flags, mod.keywordFlags, unpack(mod)` does.
func derived(name string, typ ModType, value Value, m *Mod, flags ModFlag, tags []Tag) *Mod {
	return NewModFull(name, typ, value, m.Source, m.SourceSet, flags, m.KeywordFlags, tags...)
}

// getSimpleConv — ModParser.lua:18. factor of 0 means no factor was given.
func getSimpleConv(srcList []string, dst string, typ ModType, remove bool, factor float64, srcType ModType) jewelNodeFunc {
	return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		attributes := map[string]bool{"Dex": true, "Int": true, "Str": true}
		if node == nil {
			return
		}
		for _, src := range srcList {
			for _, m := range node.ModList() {
				// do not convert stats from tattoos
				typeMatches := m.Type == typ
				if srcType != 0 {
					typeMatches = m.Type == srcType
				}
				if m.Name == src && typeMatches && !(node.IsTattoo() && attributes[src]) {
					if remove {
						out.MergeMod(derived(src, typ, negateValue(m.Value), m, m.Flags, m.Tags))
					}
					if factor != 0 {
						out.MergeMod(derived(dst, typ, Num(math.Floor(numValue(m.Value)*factor)), m, m.Flags, m.Tags))
					} else {
						out.MergeMod(derived(dst, typ, m.Value, m, m.Flags, m.Tags))
					}
				}
			}
		}
	}
}

func negateValue(v Value) Value {
	if f, ok := v.(Num); ok {
		return -f
	}
	return v
}

func numValue(v Value) float64 {
	n, _ := NumOf(v)
	return n
}

// getPerStat — ModParser.lua:6314.
func getPerStat(dst string, modType ModType, flags ModFlag, stat string, factor float64) jewelNodeFunc {
	return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if node != nil {
			data.AddStat(stat, out.Sum(Base, stat))
		} else if data.Stat(stat) != 0 {
			out.AddMod(modsf(dst, modType, Num(math.Floor(data.Stat(stat)*factor)), data.ModSource, flags, KeywordNone))
		}
	}
}

// getThreshold — ModParser.lua:6400. The threshold is on the sum of the
// listed attributes.
func getThreshold(attribs []string, name string, modType ModType, value Value, tags ...Tag) jewelNodeFunc {
	return getThresholdF(attribs, name, modType, value, FlagNone, KeywordNone, tags...)
}

func getThresholdF(attribs []string, name string, modType ModType, value Value, flags ModFlag, kw KeywordFlag, tags ...Tag) jewelNodeFunc {
	baseMod := modsf(name, modType, value, "", flags, kw, tags...) // source "" exactly as the reference
	return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if node != nil {
			for _, att := range attribs {
				nodeVal := out.Sum(Base, att)
				data.AddStat(att, nodeVal)
				data.AddStat("total", nodeVal)
			}
		} else if data.Stat("total") >= 40 {
			m := *baseMod
			m.Source = data.ModSource
			m.SourceSet = true
			if ref, ok := baseMod.Value.(ModRef); ok && ref.Mod != nil {
				// the reference mutates the SHARED inner mod's source
				ref.Mod.Source = data.ModSource
				ref.Mod.SourceSet = true
			}
			out.AddMod(&m)
		}
	}
}

// jewelOtherFuncs — ModParser.lua:6096. Values are either node functions (keys
// are exact mod text) or parse-time factories (keys are regex).
var jewelOtherFuncs = map[string]jewelValue{
	"Strength from Passives in Radius is Transformed to Dexterity":                            getSimpleConv([]string{"Str"}, "Dex", Base, true, 0, 0),
	"Dexterity from Passives in Radius is Transformed to Strength":                            getSimpleConv([]string{"Dex"}, "Str", Base, true, 0, 0),
	"Strength from Passives in Radius is Transformed to Intelligence":                         getSimpleConv([]string{"Str"}, "Int", Base, true, 0, 0),
	"Intelligence from Passives in Radius is Transformed to Strength":                         getSimpleConv([]string{"Int"}, "Str", Base, true, 0, 0),
	"Dexterity from Passives in Radius is Transformed to Intelligence":                        getSimpleConv([]string{"Dex"}, "Int", Base, true, 0, 0),
	"Intelligence from Passives in Radius is Transformed to Dexterity":                        getSimpleConv([]string{"Int"}, "Dex", Base, true, 0, 0),
	"Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield":    getSimpleConv([]string{"Life"}, "EnergyShield", Inc, true, 0, 0),
	"Increases and Reductions to Evasion Rating in Radius are Transformed to apply to Armour": getSimpleConv([]string{"Evasion"}, "Armour", Inc, true, 0, 0),
	`increases and reductions to energy shield in radius are transformed to apply to armour at ([0-9]+)% of their value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"EnergyShield"}, "Armour", Inc, true, c.n(1)/100, 0)
	}),
	`increases and reductions to life in radius are transformed to apply to mana at ([0-9]+)% of their value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"Life"}, "Mana", Inc, true, c.n(1)/100, 0)
	}),
	"Increases and Reductions to Physical Damage in Radius are Transformed to apply to Cold Damage":    getSimpleConv([]string{"PhysicalDamage"}, "ColdDamage", Inc, true, 0, 0),
	"Increases and Reductions to Cold Damage in Radius are Transformed to apply to Physical Damage":    getSimpleConv([]string{"ColdDamage"}, "PhysicalDamage", Inc, true, 0, 0),
	"Increases and Reductions to other Damage Types in Radius are Transformed to apply to Fire Damage": getSimpleConv([]string{"PhysicalDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"}, "FireDamage", Inc, true, 0, 0),
	`passives granting lightning resistance or all elemental resistances in radius also grant chance to block spells? ?d?a?m?a?g?e? at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"LightningResist", "ElementalResist"}, "SpellBlockChance", Base, false, c.n(1)/100, 0)
	}),
	`passives granting lightning resistance or all elemental resistances in radius also grant increased maximum energy shield at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"LightningResist", "ElementalResist"}, "EnergyShield", Inc, false, c.n(1)/100, Base)
	}),
	`passives granting lightning resistance or all elemental resistances in radius also grant lightning damage converted to chaos damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"LightningResist", "ElementalResist"}, "LightningDamageConvertToChaos", Base, false, c.n(1)/100, 0)
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant chance to dodge attacks? ?h?i?t?s? at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "AttackDodgeChance", Base, false, c.n(1)/100, 0)
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant chance to suppress spell damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "SpellSuppressionChance", Base, false, c.n(1)/100, 0)
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant increased maximum mana at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "Mana", Inc, false, c.n(1)/100, Base)
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant cold damage converted to chaos damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "ColdDamageConvertToChaos", Base, false, c.n(1)/100, 0)
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant chance to block attack damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "BlockChance", Base, false, c.n(1)/100, 0)
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant chance to block at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "BlockChance", Base, false, c.n(1)/100, 0)
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant increased maximum life at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "Life", Inc, false, c.n(1)/100, Base)
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant fire damage converted to chaos damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "FireDamageConvertToChaos", Base, false, c.n(1)/100, 0)
	}),
	// ModParser.lua:6147 — melee-to-bow transform.
	"Melee and Melee Weapon Type modifiers in Radius are Transformed to Bow Modifiers": jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if node == nil {
			return
		}
		mask1 := FlagAxe | FlagClaw | FlagDagger | FlagMace | FlagStaff | FlagSword | FlagMelee
		mask2 := FlagWeapon1H | FlagWeaponMelee
		mask3 := FlagWeapon2H | FlagWeaponMelee
		using := map[string]bool{"UsingAxe": true, "UsingClaw": true, "UsingDagger": true, "UsingMace": true, "UsingStaff": true, "UsingSword": true, "UsingMeleeWeapon": true}
		usingCond := func(t Tag) bool {
			ct, ok := t.(*CondTag)
			return ok && !ct.IsActor && using[ct.Var]
		}
		for _, m := range node.ModList() {
			if m.Flags&mask1 != 0 || m.Flags&mask2 == mask2 || m.Flags&mask3 == mask3 {
				out.MergeMod(derived(m.Name, m.Type, negateValue(m.Value), m, m.Flags, m.Tags))
				out.MergeMod(derived(m.Name, m.Type, m.Value, m, (m.Flags&^(mask1|mask2|mask3))|FlagBow, m.Tags))
			} else if len(m.Tags) > 0 {
				for _, tag := range m.Tags {
					if tag != nil && usingCond(tag) {
						newTags := CloneTags(m.Tags)
						for _, t := range newTags {
							if t != nil && usingCond(t) {
								t.(*CondTag).Var = "UsingBow"
								break
							}
						}
						out.MergeMod(derived(m.Name, m.Type, negateValue(m.Value), m, m.Flags, m.Tags))
						out.MergeMod(derived(m.Name, m.Type, m.Value, m, m.Flags, newTags))
						break
					}
				}
			}
		}
	}),
	`([0-9]+)% increased effect of non-keystone passive skills in radius`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() != "Keystone" && n.Type() != "ClassStart" {
				out.AddMod(mods("PassiveSkillEffect", Inc, Num(num), data.ModSource))
			}
		}
	}),
	"Notable Passive Skills in Radius grant nothing": jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if n := node; n != nil && n.Type() == "Notable" {
			out.AddMod(mods("PassiveSkillHasNoEffect", Flag, Bool(true), data.ModSource))
		}
	}),
	`([0-9]+)% increased effect of tattoos in radius`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.IsTattoo() {
				out.AddMod(mods("PassiveSkillEffect", Inc, Num(num), data.ModSource))
			}
		}
	}),
	"Allocated Small Passive Skills in Radius grant nothing": jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if n := node; n != nil && n.Type() == "Normal" {
			out.AddMod(mods("AllocatedPassiveSkillHasNoEffect", Flag, Bool(true), data.ModSource))
		}
	}),
	"Allocated Notable Passive Skills in Radius grant nothing": jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if n := node; n != nil && n.Type() == "Notable" {
			out.AddMod(mods("AllocatedPassiveSkillHasNoEffect", Flag, Bool(true), data.ModSource))
		}
	}),
	`passive skills in radius also grant: traps and mines deal ([0-9]+) to ([0-9]+) added physical damage`: jewelFactory(func(c caps) jewelNodeFunc {
		min, max := c.n(1), c.n(2)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() != "Keystone" && n.Type() != "ClassStart" {
				out.AddMod(modsf("PhysicalMin", Base, Num(min), data.ModSource, FlagNone, KeywordTrap|KeywordMine))
				out.AddMod(modsf("PhysicalMax", Base, Num(max), data.ModSource, FlagNone, KeywordTrap|KeywordMine))
			}
		}
	}),
	`passive skills in radius also grant: ([0-9]+)% increased unarmed attack speed with melee skills`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() != "Keystone" && n.Type() != "ClassStart" {
				out.AddMod(modsf("Speed", Inc, Num(num), data.ModSource, FlagUnarmed|FlagAttack|FlagMelee, KeywordNone))
			}
		}
	}),
	`passive skills in radius also grant ([0-9]+)% increased global critical strike chance`: radiusGrantFactory("CritChance", Inc),
	`passive skills in radius also grant \+([0-9]+) to maximum life`:                        radiusGrantFactory("Life", Base),
	`passive skills in radius also grant \+([0-9]+) to maximum mana`:                        radiusGrantFactory("Mana", Base),
	`passive skills in radius also grant ([0-9]+)% increased energy shield`:                 radiusGrantFactory("EnergyShield", Inc),
	`passive skills in radius also grant ([0-9]+)% increased armour`:                        radiusGrantFactory("Armour", Inc),
	`passive skills in radius also grant ([0-9]+)% increased evasion rating`:                radiusGrantFactory("Evasion", Inc),
	`passive skills in radius also grant \+([0-9]+) to all attributes`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() != "Keystone" && n.Type() != "Socket" && n.Type() != "ClassStart" {
				out.AddMod(mods("Str", Base, Num(num), data.ModSource))
				out.AddMod(mods("Dex", Base, Num(num), data.ModSource))
				out.AddMod(mods("Int", Base, Num(num), data.ModSource))
				out.AddMod(mods("All", Base, Num(num), data.ModSource))
			}
		}
	}),
	`passive skills in radius also grant \+([0-9]+)% to chaos resistance`: radiusGrantFactory("ChaosResist", Base),
	`passive skills in radius also grant ([0-9]+)% increased ([0-9a-zA-Z]+) damage`: jewelFactory(func(c caps) jewelNodeFunc {
		num, typ := c.n(1), c.s(2)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() != "Keystone" && n.Type() != "Socket" && n.Type() != "ClassStart" {
				out.AddMod(mods(firstToUpper(typ)+"Damage", Inc, Num(num), data.ModSource))
			}
		}
	}),
	`notable passive skills in radius are transformed to instead grant: ([0-9]+)% increased mana cost of skills and ([0-9]+)% increased spell damage`: jewelFactory(func(c caps) jewelNodeFunc {
		num1, num2 := c.n(1), c.n(2)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() == "Notable" {
				out.AddMod(mods("PassiveSkillHasOtherEffect", Flag, Bool(true), data.ModSource))
				out.AddMod(mods("NodeModifier", List, ModRef{Mod: mods("ManaCost", Inc, Num(num1), data.ModSource)}, data.ModSource))
				out.AddMod(mods("NodeModifier", List, ModRef{Mod: modsf("Damage", Inc, Num(num2), data.ModSource, FlagSpell, KeywordNone)}, data.ModSource))
			}
		}
	}),
	`notable passive skills in radius are transformed to instead grant: minions take ([0-9]+)% increased damage`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() == "Notable" {
				out.AddMod(mods("PassiveSkillHasOtherEffect", Flag, Bool(true), data.ModSource))
				out.AddMod(mods("NodeModifier", List, ModRef{Mod: mod("MinionModifier", List, ModRef{Mod: mods("DamageTaken", Inc, Num(num), data.ModSource)})}, data.ModSource))
			}
		}
	}),
	`notable passive skills in radius are transformed to instead grant: minions have ([0-9]+)% reduced movement speed`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() == "Notable" {
				out.AddMod(mods("PassiveSkillHasOtherEffect", Flag, Bool(true), data.ModSource))
				out.AddMod(mods("NodeModifier", List, ModRef{Mod: mod("MinionModifier", List, ModRef{Mod: mods("MovementSpeed", Inc, Num(-num), data.ModSource)})}, data.ModSource))
			}
		}
	}),
}

// radiusGrantFactory covers the "Passive Skills in Radius also grant X" family
// that excludes Keystone/Socket/ClassStart nodes.
func radiusGrantFactory(name string, typ ModType) jewelFactory {
	return func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.Type() != "Keystone" && n.Type() != "Socket" && n.Type() != "ClassStart" {
				out.AddMod(mods(name, typ, Num(num), data.ModSource))
			}
		}
	}
}

func stripSpaces(s string) string      { return strings.ReplaceAll(s, " ", "") }
func replaceAll(s, a, b string) string { return strings.ReplaceAll(s, a, b) }
func splitWords(s string) []string     { return strings.Fields(s) }

// jewelSelfFuncs — ModParser.lua:6323: radius jewels that modify the jewel
// itself based on nearby allocated nodes.
var jewelSelfFuncs = map[string]jewelValue{
	"Adds 1 to maximum Life per 3 Intelligence in Radius":                                                    getPerStat("Life", Base, 0, "Int", 1.0/3),
	"Adds 1 to Maximum Life per 3 Intelligence Allocated in Radius":                                          getPerStat("Life", Base, 0, "Int", 1.0/3),
	"1% increased Evasion Rating per 3 Dexterity Allocated in Radius":                                        getPerStat("Evasion", Inc, 0, "Dex", 1.0/3),
	"1% increased Claw Physical Damage per 3 Dexterity Allocated in Radius":                                  getPerStat("PhysicalDamage", Inc, FlagClaw, "Dex", 1.0/3),
	"1% increased Melee Physical Damage while Unarmed per 3 Dexterity Allocated in Radius":                   getPerStat("PhysicalDamage", Inc, FlagUnarmed, "Dex", 1.0/3),
	"3% increased Totem Life per 10 Strength in Radius":                                                      getPerStat("TotemLife", Inc, 0, "Str", 3.0/10),
	"3% increased Totem Life per 10 Strength Allocated in Radius":                                            getPerStat("TotemLife", Inc, 0, "Str", 3.0/10),
	"Adds 1 maximum Lightning Damage to Attacks per 1 Dexterity Allocated in Radius":                         getPerStat("LightningMax", Base, FlagAttack, "Dex", 1),
	"5% increased Chaos damage per 10 Intelligence from Allocated Passives in Radius":                        getPerStat("ChaosDamage", Inc, 0, "Int", 5.0/10),
	"-1 Strength per 1 Strength on Allocated Passives in Radius":                                             getPerStat("Str", Base, 0, "Str", -1),
	"1% additional Physical Damage Reduction per 10 Strength on Allocated Passives in Radius":                getPerStat("PhysicalDamageReduction", Base, 0, "Str", 1.0/10),
	"2% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", Inc, 0, "Str", 2.0/10),
	"3% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", Inc, 0, "Str", 3.0/10),
	"-1 Intelligence per 1 Intelligence on Allocated Passives in Radius":                                     getPerStat("Int", Base, 0, "Int", -1),
	"0.4% of Energy Shield Regenerated per Second for every 10 Intelligence on Allocated Passives in Radius": getPerStat("EnergyShieldRegenPercent", Base, 0, "Int", 0.4/10),
	"regenerate 0.4% of energy shield per second for every 10 Intelligence on Allocated Passives in Radius":  getPerStat("EnergyShieldRegenPercent", Base, 0, "Int", 0.4/10),
	"2% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", Inc, 0, "Int", 2.0/10),
	"3% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", Inc, 0, "Int", 3.0/10),
	"-1 Dexterity per 1 Dexterity on Allocated Passives in Radius":                                           getPerStat("Dex", Base, 0, "Dex", -1),
	"2% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", Inc, 0, "Dex", 2.0/10),
	"3% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", Inc, 0, "Dex", 3.0/10),
	// ModParser.lua:6333
	"Dexterity and Intelligence from passives in Radius count towards Strength Melee Damage bonus": jewelNodeFunc(func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if n := node; n != nil {
			data.AddStat("Dex", sumModList(n.ModList(), Base, "Dex"))
			data.AddStat("Int", sumModList(n.ModList(), Base, "Int"))
		} else if data.HasStat("Dex") || data.HasStat("Int") {
			out.AddMod(mods("DexIntToMeleeBonus", Base, Num(data.Stat("Dex")+data.Stat("Int")), data.ModSource))
		}
	}),
}

// sumModList mirrors node.modList:Sum("BASE", nil, name) for a plain list.
func sumModList(list []*Mod, typ ModType, name string) float64 {
	total := 0.0
	for _, m := range list {
		if m.Type == typ && m.Name == name && len(m.Tags) == 0 {
			total += numValue(m.Value)
		}
	}
	return total
}

// jewelSelfUnallocFuncs — ModParser.lua:6354.
var jewelSelfUnallocFuncs = map[string]jewelValue{
	"+5% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius":      getPerStat("CritMultiplier", Base, 0, "Str", 5.0/10),
	"+7% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius":      getPerStat("CritMultiplier", Base, 0, "Str", 7.0/10),
	"2% reduced Life Recovery Rate per 10 Strength on Unallocated Passives in Radius":          getPerStat("LifeRecoveryRate", Inc, 0, "Str", -2.0/10),
	"+15 to maximum Mana per 10 Dexterity on Unallocated Passives in Radius":                   getPerStat("Mana", Base, 0, "Dex", 15.0/10),
	"+100 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius":            getPerStat("Accuracy", Base, 0, "Int", 100.0/10),
	"+125 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius":            getPerStat("Accuracy", Base, 0, "Int", 125.0/10),
	"2% reduced Mana Recovery Rate per 10 Intelligence on Unallocated Passives in Radius":      getPerStat("ManaRecoveryRate", Inc, 0, "Int", -2.0/10),
	"+3% to Damage over Time Multiplier per 10 Intelligence on Unallocated Passives in Radius": getPerStat("DotMultiplier", Base, 0, "Int", 3.0/10),
	"2% reduced Movement Speed per 10 Dexterity on Unallocated Passives in Radius":             getPerStat("MovementSpeed", Inc, 0, "Dex", -2.0/10),
	"+125 to Accuracy Rating per 10 Dexterity on Unallocated Passives in Radius":               getPerStat("Accuracy", Base, 0, "Dex", 125.0/10),
	// ModParser.lua:6365,6381
	"Grants all bonuses of Unallocated Small Passive Skills in Radius":   grantsUnallocated("Normal"),
	"Grants all bonuses of Unallocated Notable Passive Skills in Radius": grantsUnallocated("Notable"),
}

func grantsUnallocated(nodeType string) jewelNodeFunc {
	return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
		if node != nil {
			if node.Type() == nodeType {
				if data.ModList == nil {
					data.ModList = []*Mod{} // the reference's new("ModList")
				}
				// Filter out "Condition:ConnectedTo" mods as these nodes are
				// not technically allocated by this jewel func
				for _, m := range out.Mods() {
					if !strings.HasPrefix(m.Name, "Condition:ConnectedTo") {
						data.ModList = append(data.ModList, m)
					}
				}
			}
		} else if data.ModList != nil {
			out.AddList(data.ModList)
		}
	}
}

// jewelThresholdFuncs — ModParser.lua:6425.
var jewelThresholdFuncs = map[string]jewelValue{
	"With at least 40 Dexterity in Radius, Frost Blades Melee Damage Penetrates 15% Cold Resistance":                      getThresholdF([]string{"Dex"}, "ColdPenetration", Base, Num(15), FlagMelee, KeywordNone, &SkillNameTag{SkillName: "Frost Blades", IncludeTransfigured: true}),
	"With at least 40 Dexterity in Radius, Melee Damage dealt by Frost Blades Penetrates 15% Cold Resistance":             getThresholdF([]string{"Dex"}, "ColdPenetration", Base, Num(15), FlagMelee, KeywordNone, &SkillNameTag{SkillName: "Frost Blades", IncludeTransfigured: true}),
	"With at least 40 Dexterity in Radius, Frost Blades has 25% increased Projectile Speed":                               getThreshold([]string{"Dex"}, "ProjectileSpeed", Inc, Num(25), &SkillNameTag{SkillName: "Frost Blades", IncludeTransfigured: true}),
	"With at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect":                                     getThreshold([]string{"Dex"}, "AreaOfEffect", Inc, Num(25), &SkillNameTag{SkillName: "Ice Shot"}),
	"Ice Shot Pierces 5 additional Targets with 40 Dexterity in Radius":                                                   getThreshold([]string{"Dex"}, "PierceCount", Base, Num(5), &SkillNameTag{SkillName: "Ice Shot"}),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 3 additional Targets":                                         getThreshold([]string{"Dex"}, "PierceCount", Base, Num(3), &SkillNameTag{SkillName: "Ice Shot"}),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 5 additional Targets":                                         getThreshold([]string{"Dex"}, "PierceCount", Base, Num(5), &SkillNameTag{SkillName: "Ice Shot"}),
	"With at least 40 Intelligence in Radius, Frostbolt fires 2 additional Projectiles":                                   getThreshold([]string{"Int"}, "ProjectileCount", Base, Num(2), &SkillNameTag{SkillName: "Frostbolt"}),
	"With at least 40 Intelligence in Radius, Rolling Magma fires an additional Projectile":                               getThreshold([]string{"Int"}, "ProjectileCount", Base, Num(1), &SkillNameTag{SkillName: "Rolling Magma"}),
	"With at least 40 Intelligence in Radius, Rolling Magma has 10% increased Area of Effect per Chain":                   getThreshold([]string{"Int"}, "AreaOfEffect", Inc, Num(10), &SkillNameTag{SkillName: "Rolling Magma"}, &StatTag{StatKind: TagPerStat, Stat: "Chain"}),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 40% more damage per chain":                              getThreshold([]string{"Int"}, "Damage", More, Num(40), &SkillNameTag{SkillName: "Rolling Magma"}, &StatTag{StatKind: TagPerStat, Stat: "Chain"}),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 50% less damage":                                        getThreshold([]string{"Int"}, "Damage", More, Num(-50), &SkillNameTag{SkillName: "Rolling Magma"}),
	"With at least 40 Dexterity in Radius, Shrapnel Shot has 25% increased Area of Effect":                                getThreshold([]string{"Dex"}, "AreaOfEffect", Inc, Num(25), &SkillNameTag{SkillName: "Shrapnel Shot"}),
	"With at least 40 Dexterity in Radius, Shrapnel Shot's cone has a 50% chance to deal Double Damage":                   getThreshold([]string{"Dex"}, "DoubleDamageChance", Base, Num(50), &SkillNameTag{SkillName: "Shrapnel Shot"}, &SkillPartTag{Part: opt(2)}),
	"With at least 40 Dexterity in Radius, Galvanic Arrow deals 50% increased Area Damage":                                getThreshold([]string{"Dex"}, "Damage", Inc, Num(50), &SkillNameTag{SkillName: "Galvanic Arrow", IncludeTransfigured: true}, &SkillPartTag{Part: opt(2)}),
	"With at least 40 Dexterity in Radius, Galvanic Arrow has 25% increased Area of Effect":                               getThreshold([]string{"Dex"}, "AreaOfEffect", Inc, Num(25), &SkillNameTag{SkillName: "Galvanic Arrow", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Freezing Pulse fires 2 additional Projectiles":                              getThreshold([]string{"Int"}, "ProjectileCount", Base, Num(2), &SkillNameTag{SkillName: "Freezing Pulse"}),
	"With at least 40 Intelligence in Radius, 25% increased Freezing Pulse Damage if you've Shattered an Enemy Recently":  getThreshold([]string{"Int"}, "Damage", Inc, Num(25), &SkillNameTag{SkillName: "Freezing Pulse"}, &CondTag{Var: "ShatteredEnemyRecently"}),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 10 additional Projectiles":                               getThreshold([]string{"Dex"}, "ProjectileCount", Base, Num(10), &SkillNameTag{SkillName: "Ethereal Knives", IncludeTransfigured: true}),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 5 additional Projectiles":                                getThreshold([]string{"Dex"}, "ProjectileCount", Base, Num(5), &SkillNameTag{SkillName: "Ethereal Knives", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Molten Strike fires 2 additional Projectiles":                                   getThreshold([]string{"Str"}, "ProjectileCount", Base, Num(2), &SkillNameTag{SkillName: "Molten Strike", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Molten Strike has 25% increased Area of Effect":                                 getThreshold([]string{"Str"}, "AreaOfEffect", Inc, Num(25), &SkillNameTag{SkillName: "Molten Strike", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Molten Strike Projectiles Chain +1 time":                                        getThreshold([]string{"Str"}, "ChainCountMax", Base, Num(1), &SkillNameTag{SkillName: "Molten Strike", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Molten Strike fires 50% less Projectiles":                                       getThreshold([]string{"Str"}, "ProjectileCount", More, Num(-50), &SkillNameTag{SkillName: "Molten Strike", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, 25% of Glacial Hammer Physical Damage converted to Cold Damage":                 getThreshold([]string{"Str"}, "SkillPhysicalDamageConvertToCold", Base, Num(25), &SkillNameTag{SkillName: "Glacial Hammer", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage":                            getThreshold([]string{"Str"}, "DoubleDamageChance", Base, Num(20), &SkillNameTag{SkillName: "Heavy Strike"}),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage.":                           getThreshold([]string{"Str"}, "DoubleDamageChance", Base, Num(20), &SkillNameTag{SkillName: "Heavy Strike"}),
	"With at least 40 Strength in Radius, Cleave has +1 to Radius per Nearby Enemy, up to +10":                            getThreshold([]string{"Str"}, "AreaOfEffect", Base, Num(1), &MultiplierTag{Var: "NearbyEnemies", Limit: opt(10)}, &SkillNameTag{SkillName: "Cleave", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Cleave has +0.1 metres to Radius per Nearby Enemy, up to a maximum of +1 metre": getThreshold([]string{"Str"}, "AreaOfEffect", Base, Num(1), &MultiplierTag{Var: "NearbyEnemies", Limit: opt(10)}, &SkillNameTag{SkillName: "Cleave", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Cleave grants Fortify on Hit":                                                   getThreshold([]string{"Str"}, "ExtraSkillMod", List, ModRef{Mod: flag("Condition:Fortified")}, &SkillNameTag{SkillName: "Cleave", IncludeTransfigured: true}),
	"With at least 40 Strength in Radius, Hits with Cleave Fortify":                                                       getThreshold([]string{"Str"}, "ExtraSkillMod", List, ModRef{Mod: flag("Condition:Fortified")}, &SkillNameTag{SkillName: "Cleave", IncludeTransfigured: true}),
	"With at least 40 Dexterity in Radius, Dual Strike has a 20% chance to deal Double Damage with the Main-Hand Weapon":  getThreshold([]string{"Dex"}, "DoubleDamageChance", Base, Num(20), &SkillNameTag{SkillName: "Dual Strike", IncludeTransfigured: true}, &CondTag{Var: "MainHandAttack"}),
	`with at least 40 dexterity in radius, dual strike has ([0-9]+)% increased attack speed while wielding a claw`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Dex"}, "Speed", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Dual Strike", IncludeTransfigured: true}, &CondTag{Var: "UsingClaw"})
	}),
	`with at least 40 dexterity in radius, dual strike has \+([0-9]+)% to critical strike multiplier while wielding a dagger`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Dex"}, "CritMultiplier", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Dual Strike", IncludeTransfigured: true}, &CondTag{Var: "UsingDagger"})
	}),
	`with at least 40 dexterity in radius, dual strike has ([0-9]+)% increased accuracy rating while wielding a sword`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Dex"}, "Accuracy", Inc, Num(c.n(1)), &SkillNameTag{SkillName: "Dual Strike", IncludeTransfigured: true}, &CondTag{Var: "UsingSword"})
	}),
	"With at least 40 Dexterity in Radius, Dual Strike Hits Intimidate Enemies for 4 seconds while wielding an Axe":                  getThreshold([]string{"Dex"}, "EnemyModifier", List, ModRef{Mod: flag("Condition:Intimidated")}, &CondTag{Var: "UsingAxe"}),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack has 100% increased Cooldown Recovery Speed":                getThreshold([]string{"Int"}, "MinionModifier", List, ModRef{Mod: mod("CooldownRecovery", Inc, Num(100), &SkillIDTag{SkillID: "ZombieSlam"})}),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack deals 30% increased Damage":                                getThreshold([]string{"Int"}, "MinionModifier", List, ModRef{Mod: mod("Damage", Inc, Num(30), &SkillIDTag{SkillID: "ZombieSlam"})}),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Attack Damage for each Poison on the Enemy":               getThresholdF([]string{"Dex"}, "Damage", Inc, Num(2), FlagAttack, KeywordNone, &SkillNameTag{SkillName: "Viper Strike", IncludeTransfigured: true}, &MultiplierTag{Actor: "enemy", Var: "PoisonStack"}),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Damage with Hits and Poison for each Poison on the Enemy": getThresholdF([]string{"Dex"}, "Damage", Inc, Num(2), FlagNone, KeywordHit|KeywordPoison, &SkillNameTag{SkillName: "Viper Strike", IncludeTransfigured: true}, &MultiplierTag{Actor: "enemy", Var: "PoisonStack"}),
	"With at least 40 Intelligence in Radius, Spark fires 2 additional Projectiles":                                                  getThreshold([]string{"Int"}, "ProjectileCount", Base, Num(2), &SkillNameTag{SkillName: "Spark", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Blight has 50% increased Hinder Duration":                                              getThreshold([]string{"Int"}, "SecondaryDuration", Inc, Num(50), &SkillNameTag{SkillName: "Blight", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Enemies Hindered by Blight take 25% increased Chaos Damage":                            getThreshold([]string{"Int"}, "ExtraSkillMod", List, ModRef{Mod: mod("ChaosDamageTaken", Inc, Num(25), &GlobalEffectTag{EffectType: "Debuff", EffectName: "Hinder"})}, &SkillNameTag{SkillName: "Blight", IncludeTransfigured: true}, &CondTag{IsActor: true, Actor: "enemy", Var: "Hindered"}),
	"With 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":                                getThreshold([]string{"Int"}, "SkillPhysicalDamageConvertToCold", Base, Num(20), &SkillNameTag{SkillName: "Glacial Cascade", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":                       getThreshold([]string{"Int"}, "SkillPhysicalDamageConvertToCold", Base, Num(20), &SkillNameTag{SkillName: "Glacial Cascade", IncludeTransfigured: true}),
	"With 40 total Intelligence and Dexterity in Radius, Elemental Hit and Wild Strike deal 50% less Fire Damage":                    getThreshold([]string{"Int", "Dex"}, "FireDamage", More, Num(-50), &SkillNameTag{SkillNameList: []string{"Elemental Hit", "Wild Strike"}, IncludeTransfigured: true}),
	"With 40 total Strength and Intelligence in Radius, Elemental Hit and Wild Strike deal 50% less Cold Damage":                     getThreshold([]string{"Str", "Int"}, "ColdDamage", More, Num(-50), &SkillNameTag{SkillNameList: []string{"Elemental Hit", "Wild Strike"}, IncludeTransfigured: true}),
	"With 40 total Dexterity and Strength in Radius, Elemental Hit and Wild Strike deal 50% less Lightning Damage":                   getThreshold([]string{"Dex", "Str"}, "LightningDamage", More, Num(-50), &SkillNameTag{SkillNameList: []string{"Elemental Hit", "Wild Strike"}, IncludeTransfigured: true}),
	"With 40 total Intelligence and Dexterity in Radius, Prismatic Skills deal 50% less Fire Damage":                                 getThreshold([]string{"Int", "Dex"}, "FireDamage", More, Num(-50), &SkillTypeTag{SkillType: SkillTypeRandomElement}),
	"With 40 total Strength and Intelligence in Radius, Prismatic Skills deal 50% less Cold Damage":                                  getThreshold([]string{"Str", "Int"}, "ColdDamage", More, Num(-50), &SkillTypeTag{SkillType: SkillTypeRandomElement}),
	"With 40 total Dexterity and Strength in Radius, Prismatic Skills deal 50% less Lightning Damage":                                getThreshold([]string{"Dex", "Str"}, "LightningDamage", More, Num(-50), &SkillTypeTag{SkillType: SkillTypeRandomElement}),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw Chains +4 times":                                          getThreshold([]string{"Dex", "Str"}, "ChainCountMax", Base, Num(4), &SkillNameTag{SkillName: "Spectral Shield Throw", IncludeTransfigured: true}),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw fires 75% less Shard Projectiles":                         getThreshold([]string{"Dex", "Str"}, "ProjectileCount", More, Num(-75), &SkillNameTag{SkillName: "Spectral Shield Throw", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Blight inflicts Withered for 2 seconds":                                                getThreshold([]string{"Int"}, "ExtraSkillMod", List, ModRef{Mod: mod("Condition:CanWither", Flag, Bool(true))}, &SkillNameTag{SkillName: "Blight", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Blight has 30% reduced Cast Speed":                                                     getThreshold([]string{"Int"}, "Speed", Inc, Num(-30), &SkillNameTag{SkillName: "Blight", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Fireball cannot ignite":                                                                getThreshold([]string{"Int"}, "ExtraSkillMod", List, ModRef{Mod: flag("CannotIgnite")}, &SkillNameTag{SkillName: "Fireball"}),
	`with at least 40 intelligence in radius, fireball has \+([0-9]+)% chance to inflict scorch`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Int"}, "EnemyScorchChance", Base, Num(c.n(1)), &SkillNameTag{SkillName: "Fireball"})
	}),
	"With at least 40 Intelligence in Radius, Discharge has 60% less Area of Effect": getThreshold([]string{"Int"}, "AreaOfEffect", More, Num(-60), &SkillNameTag{SkillName: "Discharge", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Discharge Cooldown is 250 ms":          getThreshold([]string{"Int"}, "CooldownRecovery", Override, Num(0.25), &SkillNameTag{SkillName: "Discharge", IncludeTransfigured: true}),
	"With at least 40 Intelligence in Radius, Discharge deals 60% less Damage":       getThreshold([]string{"Int"}, "Damage", More, Num(-60), &SkillNameTag{SkillName: "Discharge", IncludeTransfigured: true}),
	`with at least 40 intelligence in radius, ([0-9]+)% of damage taken recouped as mana if you've warcried recently`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Int"}, "ManaRecoup", Base, Num(c.n(1)), &CondTag{Var: "UsedWarcryRecently"})
	}),
	`with at least 40 intelligence in radius, fireball projectiles gain radius as they travel farther, up to \+([0-9]+) radius`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Int"}, "AreaOfEffect", Base, Num(c.n(1)), &DistanceRampTag{Ramp: Pairs{{0, 0}, {50, 1}}})
	}),
	`with at least 40 intelligence in radius, projectiles gain radius as they travel farther, up to a maximum of \+([0-9.]+) metres? to radius`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold([]string{"Int"}, "AreaOfEffect", Base, Num(c.n(1)*10), &DistanceRampTag{Ramp: Pairs{{0, 0}, {50, 1}}})
	}),
	// ModParser.lua:6489 — one line applying two independent threshold funcs.
	"With at least 40 Intelligence in Radius, Raised Spectres have a 50% chance to gain Soul Eater for 20 seconds on Kill": jewelFuncSeq{
		getThreshold([]string{"Int"}, "MinionModifier", List, ModRef{Mod: mod("Condition:CanHaveSoulEater", Flag, Bool(true))}, &SkillNameTag{SkillName: "Raise Spectre", IncludeTransfigured: true}),
		getThreshold([]string{"Int"}, "Condition:MinionCanHaveSoulEater", Flag, Bool(true)),
	},
}

// jewelFuncList — ModParser.lua:6497-6538: the unified lookup keyed by
// lowercased line. Each value carries the type label and either a ready node
// function or a parse-time factory.
type jewelFuncEntry struct {
	typ     string
	nodeFn  jewelNodeFunc  // ready function (exact-lookup entries)
	factory jewelFactory   // called with the captures at parse time
	re      *regexp.Regexp // compiled key, set for parametric (factory) entries
}

var jewelFuncList = buildJewelFuncList()

func buildJewelFuncList() map[string]jewelFuncEntry {
	out := map[string]jewelFuncEntry{}

	// Jewels that modify nodes: wrap so nodes already conquered by timeless
	// jewels are never modified — ModParser.lua:6500-6515.
	wrapOther := func(inner jewelNodeFunc) jewelNodeFunc {
		return func(node JewelNodeRef, out JewelStoreWriter, data *JewelFuncTag) {
			if n := node; n != nil && n.ConqueredBy() {
				return
			}
			inner(node, out, data)
		}
	}
	for k, v := range jewelOtherFuncs {
		switch fv := v.(type) {
		case jewelNodeFunc:
			inner := fv
			out[strings.ToLower(k)] = jewelFuncEntry{typ: "Other", factory: func(c caps) jewelNodeFunc {
				return wrapOther(inner)
			}}
		case jewelFactory:
			factory := fv
			// Factory keys are pre-lowered regex; lowercasing again would wreck
			// their character classes.
			out[k] = jewelFuncEntry{typ: "Other", re: compileJewelKey(k), factory: func(c caps) jewelNodeFunc {
				return wrapOther(factory(c))
			}}
		default:
			panic(fmt.Sprintf("modparser: jewelOtherFuncs[%q] holds an unhandled %T", k, v))
		}
	}
	// nodeFn recovers an entry of a table that holds ready node functions
	// only; like the switch defaults, it can only fire on a new jewelValue
	// inhabitant.
	nodeFn := func(table, k string, v jewelValue) jewelNodeFunc {
		fn, ok := v.(jewelNodeFunc)
		if !ok {
			panic(fmt.Sprintf("modparser: %s[%q] is %T, not a node function", table, k, v))
		}
		return fn
	}
	for k, v := range jewelSelfFuncs {
		out[strings.ToLower(k)] = jewelFuncEntry{typ: "Self", nodeFn: nodeFn("jewelSelfFuncs", k, v)}
	}
	for k, v := range jewelSelfUnallocFuncs {
		out[strings.ToLower(k)] = jewelFuncEntry{typ: "SelfUnalloc", nodeFn: nodeFn("jewelSelfUnallocFuncs", k, v)}
	}
	for k, v := range jewelThresholdFuncs {
		switch fv := v.(type) {
		case jewelNodeFunc:
			out[strings.ToLower(k)] = jewelFuncEntry{typ: "Threshold", nodeFn: fv}
		case jewelFactory:
			out[k] = jewelFuncEntry{typ: "Threshold", re: compileJewelKey(k), factory: fv}
		case jewelFuncSeq:
			funcs := fv
			out[strings.ToLower(k)] = jewelFuncEntry{typ: "Threshold", nodeFn: func(node JewelNodeRef, w JewelStoreWriter, data *JewelFuncTag) {
				if data.FuncData == nil {
					data.FuncData = make([]*JewelFuncTag, len(funcs))
					for i := range data.FuncData {
						data.FuncData[i] = &JewelFuncTag{}
					}
				}
				for i, f := range funcs {
					data.FuncData[i].ModSource = data.ModSource
					f(node, w, data.FuncData[i])
				}
			}}
		default:
			panic(fmt.Sprintf("modparser: jewelThresholdFuncs[%q] holds an unhandled %T", k, v))
		}
	}
	return out
}

// compileJewelKey compiles a parametric jewel key (written pre-lowered so it
// matches the lowered line).
func compileJewelKey(k string) *regexp.Regexp {
	return regexp.MustCompile(k)
}
