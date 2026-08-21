package modparser

import (
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

// modStoreWriter is the slice of ModStore/ModList the jewel functions use.
type modStoreWriter interface {
	NewMod(name, typ string, value any, rest ...any)
	MergeNewMod(name, typ string, value any, rest ...any)
	Sum(typ string, cfg any, names ...string) float64
	AddMod(m *Mod)
	AddList(list any)
	ModsList() []*Mod
}

// jewelNode is what the functions read from a passive tree node.
type jewelNode interface {
	ConqueredBy() bool
	Type() string
	IsTattoo() bool
	ModList() []*Mod
}

// jewelNodeFunc mirrors the Lua signature function(node, out, data).
type jewelNodeFunc func(node any, out modStoreWriter, data Tag)

// Exported aliases so the calc engine can implement the interfaces and
// call the functions carried in JewelFunc modifier values.
type (
	JewelStoreWriter = modStoreWriter
	JewelNodeRef     = jewelNode
	JewelNodeFn      = jewelNodeFunc
)

// jewelFactory is a parse-time factory: given the pattern's captures it
// returns the node function to embed.
type jewelFactory func(c caps) jewelNodeFunc

func asJewelNode(node any) (jewelNode, bool) {
	n, ok := node.(jewelNode)
	return n, ok && n != nil
}

// getSimpleConv — ModParser.lua:18. factor of 0 means no factor was given.
func getSimpleConv(srcList []string, dst, typ string, remove bool, factor float64, srcType string) jewelNodeFunc {
	return func(node any, out modStoreWriter, data Tag) {
		attributes := map[string]bool{"Dex": true, "Int": true, "Str": true}
		n, ok := asJewelNode(node)
		if !ok {
			return
		}
		for _, src := range srcList {
			for _, m := range n.ModList() {
				// do not convert stats from tattoos
				typeMatches := m.Type == typ
				if srcType != "" {
					typeMatches = m.Type == srcType
				}
				if m.Name == src && typeMatches && !(n.IsTattoo() && attributes[src]) {
					if remove {
						out.MergeNewMod(src, typ, negateValue(m.Value), append([]any{m.Source, m.Flags, m.KeywordFlags}, m.Tags...)...)
					}
					if factor != 0 {
						out.MergeNewMod(dst, typ, math.Floor(numValue(m.Value)*factor), append([]any{m.Source, m.Flags, m.KeywordFlags}, m.Tags...)...)
					} else {
						out.MergeNewMod(dst, typ, m.Value, append([]any{m.Source, m.Flags, m.KeywordFlags}, m.Tags...)...)
					}
				}
			}
		}
	}
}

func negateValue(v any) any {
	if f, ok := v.(float64); ok {
		return -f
	}
	if i, ok := v.(int); ok {
		return -i
	}
	return v
}

func numValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	}
	return 0
}

// getPerStat — ModParser.lua:6314.
func getPerStat(dst, modType string, flags int64, stat string, factor float64) jewelNodeFunc {
	return func(node any, out modStoreWriter, data Tag) {
		if _, ok := asJewelNode(node); ok {
			data[stat] = numValue(data[stat]) + out.Sum("BASE", nil, stat)
		} else if numValue(data[stat]) != 0 {
			out.NewMod(dst, modType, math.Floor(numValue(data[stat])*factor), data["modSource"], flags)
		}
	}
}

// getThreshold — ModParser.lua:6400. attrib is a string or a []string.
func getThreshold(attrib any, name, modType string, value any, rest ...any) jewelNodeFunc {
	args := append([]any{""}, rest...) // source "" exactly as the reference
	baseMod := mod(name, modType, value, args...)
	return func(node any, out modStoreWriter, data Tag) {
		if _, ok := asJewelNode(node); ok {
			if list, isList := attrib.([]string); isList {
				for _, att := range list {
					nodeVal := out.Sum("BASE", nil, att)
					data[att] = numValue(data[att]) + nodeVal
					data["total"] = numValue(data["total"]) + nodeVal
				}
			} else {
				att := attrib.(string)
				nodeVal := out.Sum("BASE", nil, att)
				data[att] = numValue(data[att]) + nodeVal
				data["total"] = numValue(data["total"]) + nodeVal
			}
		} else if numValue(data["total"]) >= 40 {
			m := *baseMod
			m.Source, _ = data["modSource"].(string)
			m.SourceSet = true
			if valueTag, ok := baseMod.Value.(Tag); ok {
				if inner, ok := valueTag["mod"].(*Mod); ok {
					// the reference mutates the SHARED inner mod's source
					inner.Source, _ = data["modSource"].(string)
					inner.SourceSet = true
				}
			}
			out.AddMod(&m)
		}
	}
}

// jewelOtherFuncs — ModParser.lua:6096. Values are either node functions (keys
// are exact mod text) or parse-time factories (keys are regex).
var jewelOtherFuncs = map[string]any{
	"Strength from Passives in Radius is Transformed to Dexterity":                            getSimpleConv([]string{"Str"}, "Dex", "BASE", true, 0, ""),
	"Dexterity from Passives in Radius is Transformed to Strength":                            getSimpleConv([]string{"Dex"}, "Str", "BASE", true, 0, ""),
	"Strength from Passives in Radius is Transformed to Intelligence":                         getSimpleConv([]string{"Str"}, "Int", "BASE", true, 0, ""),
	"Intelligence from Passives in Radius is Transformed to Strength":                         getSimpleConv([]string{"Int"}, "Str", "BASE", true, 0, ""),
	"Dexterity from Passives in Radius is Transformed to Intelligence":                        getSimpleConv([]string{"Dex"}, "Int", "BASE", true, 0, ""),
	"Intelligence from Passives in Radius is Transformed to Dexterity":                        getSimpleConv([]string{"Int"}, "Dex", "BASE", true, 0, ""),
	"Increases and Reductions to Life in Radius are Transformed to apply to Energy Shield":    getSimpleConv([]string{"Life"}, "EnergyShield", "INC", true, 0, ""),
	"Increases and Reductions to Evasion Rating in Radius are Transformed to apply to Armour": getSimpleConv([]string{"Evasion"}, "Armour", "INC", true, 0, ""),
	`increases and reductions to energy shield in radius are transformed to apply to armour at ([0-9]+)% of their value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"EnergyShield"}, "Armour", "INC", true, c.n(1)/100, "")
	}),
	`increases and reductions to life in radius are transformed to apply to mana at ([0-9]+)% of their value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"Life"}, "Mana", "INC", true, c.n(1)/100, "")
	}),
	"Increases and Reductions to Physical Damage in Radius are Transformed to apply to Cold Damage":    getSimpleConv([]string{"PhysicalDamage"}, "ColdDamage", "INC", true, 0, ""),
	"Increases and Reductions to Cold Damage in Radius are Transformed to apply to Physical Damage":    getSimpleConv([]string{"ColdDamage"}, "PhysicalDamage", "INC", true, 0, ""),
	"Increases and Reductions to other Damage Types in Radius are Transformed to apply to Fire Damage": getSimpleConv([]string{"PhysicalDamage", "ColdDamage", "LightningDamage", "ChaosDamage", "ElementalDamage"}, "FireDamage", "INC", true, 0, ""),
	`passives granting lightning resistance or all elemental resistances in radius also grant chance to block spells? ?d?a?m?a?g?e? at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"LightningResist", "ElementalResist"}, "SpellBlockChance", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting lightning resistance or all elemental resistances in radius also grant increased maximum energy shield at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"LightningResist", "ElementalResist"}, "EnergyShield", "INC", false, c.n(1)/100, "BASE")
	}),
	`passives granting lightning resistance or all elemental resistances in radius also grant lightning damage converted to chaos damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"LightningResist", "ElementalResist"}, "LightningDamageConvertToChaos", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant chance to dodge attacks? ?h?i?t?s? at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "AttackDodgeChance", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant chance to suppress spell damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "SpellSuppressionChance", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant increased maximum mana at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "Mana", "INC", false, c.n(1)/100, "BASE")
	}),
	`passives granting cold resistance or all elemental resistances in radius also grant cold damage converted to chaos damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"ColdResist", "ElementalResist"}, "ColdDamageConvertToChaos", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant chance to block attack damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "BlockChance", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant chance to block at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "BlockChance", "BASE", false, c.n(1)/100, "")
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant increased maximum life at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "Life", "INC", false, c.n(1)/100, "BASE")
	}),
	`passives granting fire resistance or all elemental resistances in radius also grant fire damage converted to chaos damage at ([0-9]+)% of its value`: jewelFactory(func(c caps) jewelNodeFunc {
		return getSimpleConv([]string{"FireResist", "ElementalResist"}, "FireDamageConvertToChaos", "BASE", false, c.n(1)/100, "")
	}),
	// ModParser.lua:6147 — melee-to-bow transform.
	"Melee and Melee Weapon Type modifiers in Radius are Transformed to Bow Modifiers": jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
		n, ok := asJewelNode(node)
		if !ok {
			return
		}
		mask1 := ModFlag.Axe | ModFlag.Claw | ModFlag.Dagger | ModFlag.Mace | ModFlag.Staff | ModFlag.Sword | ModFlag.Melee
		mask2 := ModFlag.Weapon1H | ModFlag.WeaponMelee
		mask3 := ModFlag.Weapon2H | ModFlag.WeaponMelee
		using := map[string]bool{"UsingAxe": true, "UsingClaw": true, "UsingDagger": true, "UsingMace": true, "UsingStaff": true, "UsingSword": true, "UsingMeleeWeapon": true}
		for _, m := range n.ModList() {
			if m.Flags&mask1 != 0 || m.Flags&mask2 == mask2 || m.Flags&mask3 == mask3 {
				out.MergeNewMod(m.Name, m.Type, negateValue(m.Value), append([]any{m.Source, m.Flags, m.KeywordFlags}, m.Tags...)...)
				out.MergeNewMod(m.Name, m.Type, m.Value, append([]any{m.Source, (m.Flags &^ (mask1 | mask2 | mask3)) | ModFlag.Bow, m.KeywordFlags}, m.Tags...)...)
			} else if len(m.Tags) > 0 {
				for _, tagAny := range m.Tags {
					tag, _ := tagAny.(Tag)
					if tag != nil && tag["type"] == "Condition" && using[stringField(tag, "var")] {
						newTags := make([]any, len(m.Tags))
						for i, t := range m.Tags {
							if tt, ok := t.(Tag); ok {
								ct := Tag{}
								for k, v := range tt {
									ct[k] = v
								}
								newTags[i] = ct
							} else {
								newTags[i] = t
							}
						}
						for _, t := range newTags {
							if tt, ok := t.(Tag); ok && tt["type"] == "Condition" && using[stringField(tt, "var")] {
								tt["var"] = "UsingBow"
								break
							}
						}
						out.MergeNewMod(m.Name, m.Type, negateValue(m.Value), append([]any{m.Source, m.Flags, m.KeywordFlags}, m.Tags...)...)
						out.MergeNewMod(m.Name, m.Type, m.Value, append([]any{m.Source, m.Flags, m.KeywordFlags}, newTags...)...)
						break
					}
				}
			}
		}
	}),
	`([0-9]+)% increased effect of non-keystone passive skills in radius`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() != "Keystone" && n.Type() != "ClassStart" {
				out.NewMod("PassiveSkillEffect", "INC", num, data["modSource"])
			}
		}
	}),
	"Notable Passive Skills in Radius grant nothing": jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
		if n, ok := asJewelNode(node); ok && n.Type() == "Notable" {
			out.NewMod("PassiveSkillHasNoEffect", "FLAG", true, data["modSource"])
		}
	}),
	`([0-9]+)% increased effect of tattoos in radius`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.IsTattoo() {
				out.NewMod("PassiveSkillEffect", "INC", num, data["modSource"])
			}
		}
	}),
	"Allocated Small Passive Skills in Radius grant nothing": jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
		if n, ok := asJewelNode(node); ok && n.Type() == "Normal" {
			out.NewMod("AllocatedPassiveSkillHasNoEffect", "FLAG", true, data["modSource"])
		}
	}),
	"Allocated Notable Passive Skills in Radius grant nothing": jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
		if n, ok := asJewelNode(node); ok && n.Type() == "Notable" {
			out.NewMod("AllocatedPassiveSkillHasNoEffect", "FLAG", true, data["modSource"])
		}
	}),
	`passive skills in radius also grant: traps and mines deal ([0-9]+) to ([0-9]+) added physical damage`: jewelFactory(func(c caps) jewelNodeFunc {
		min, max := c.n(1), c.n(2)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() != "Keystone" && n.Type() != "ClassStart" {
				out.NewMod("PhysicalMin", "BASE", min, data["modSource"], int64(0), KeywordFlag.Trap|KeywordFlag.Mine)
				out.NewMod("PhysicalMax", "BASE", max, data["modSource"], int64(0), KeywordFlag.Trap|KeywordFlag.Mine)
			}
		}
	}),
	`passive skills in radius also grant: ([0-9]+)% increased unarmed attack speed with melee skills`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() != "Keystone" && n.Type() != "ClassStart" {
				out.NewMod("Speed", "INC", num, data["modSource"], ModFlag.Unarmed|ModFlag.Attack|ModFlag.Melee)
			}
		}
	}),
	`passive skills in radius also grant ([0-9]+)% increased global critical strike chance`: radiusGrantFactory("CritChance", "INC"),
	`passive skills in radius also grant \+([0-9]+) to maximum life`:                        radiusGrantFactory("Life", "BASE"),
	`passive skills in radius also grant \+([0-9]+) to maximum mana`:                        radiusGrantFactory("Mana", "BASE"),
	`passive skills in radius also grant ([0-9]+)% increased energy shield`:                 radiusGrantFactory("EnergyShield", "INC"),
	`passive skills in radius also grant ([0-9]+)% increased armour`:                        radiusGrantFactory("Armour", "INC"),
	`passive skills in radius also grant ([0-9]+)% increased evasion rating`:                radiusGrantFactory("Evasion", "INC"),
	`passive skills in radius also grant \+([0-9]+) to all attributes`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() != "Keystone" && n.Type() != "Socket" && n.Type() != "ClassStart" {
				out.NewMod("Str", "BASE", num, data["modSource"])
				out.NewMod("Dex", "BASE", num, data["modSource"])
				out.NewMod("Int", "BASE", num, data["modSource"])
				out.NewMod("All", "BASE", num, data["modSource"])
			}
		}
	}),
	`passive skills in radius also grant \+([0-9]+)% to chaos resistance`: radiusGrantFactory("ChaosResist", "BASE"),
	`passive skills in radius also grant ([0-9]+)% increased ([0-9a-zA-Z]+) damage`: jewelFactory(func(c caps) jewelNodeFunc {
		num, typ := c.n(1), c.s(2)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() != "Keystone" && n.Type() != "Socket" && n.Type() != "ClassStart" {
				out.NewMod(firstToUpper(typ)+"Damage", "INC", num, data["modSource"])
			}
		}
	}),
	`notable passive skills in radius are transformed to instead grant: ([0-9]+)% increased mana cost of skills and ([0-9]+)% increased spell damage`: jewelFactory(func(c caps) jewelNodeFunc {
		num1, num2 := c.n(1), c.n(2)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() == "Notable" {
				out.NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data["modSource"])
				out.NewMod("NodeModifier", "LIST", Tag{"mod": mod("ManaCost", "INC", num1, stringField(data, "modSource"))}, data["modSource"])
				out.NewMod("NodeModifier", "LIST", Tag{"mod": mod("Damage", "INC", num2, stringField(data, "modSource"), ModFlag.Spell)}, data["modSource"])
			}
		}
	}),
	`notable passive skills in radius are transformed to instead grant: minions take ([0-9]+)% increased damage`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() == "Notable" {
				out.NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data["modSource"])
				out.NewMod("NodeModifier", "LIST", Tag{"mod": mod("MinionModifier", "LIST", Tag{"mod": mod("DamageTaken", "INC", num, stringField(data, "modSource"))})}, data["modSource"])
			}
		}
	}),
	`notable passive skills in radius are transformed to instead grant: minions have ([0-9]+)% reduced movement speed`: jewelFactory(func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() == "Notable" {
				out.NewMod("PassiveSkillHasOtherEffect", "FLAG", true, data["modSource"])
				out.NewMod("NodeModifier", "LIST", Tag{"mod": mod("MinionModifier", "LIST", Tag{"mod": mod("MovementSpeed", "INC", -num, stringField(data, "modSource"))})}, data["modSource"])
			}
		}
	}),
}

// radiusGrantFactory covers the "Passive Skills in Radius also grant X" family
// that excludes Keystone/Socket/ClassStart nodes.
func radiusGrantFactory(name, typ string) jewelFactory {
	return func(c caps) jewelNodeFunc {
		num := c.n(1)
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.Type() != "Keystone" && n.Type() != "Socket" && n.Type() != "ClassStart" {
				out.NewMod(name, typ, num, data["modSource"])
			}
		}
	}
}

func stringField(t Tag, k string) string {
	s, _ := t[k].(string)
	return s
}

func stripSpaces(s string) string      { return strings.ReplaceAll(s, " ", "") }
func replaceAll(s, a, b string) string { return strings.ReplaceAll(s, a, b) }
func splitWords(s string) []string     { return strings.Fields(s) }

// jewelSelfFuncs — ModParser.lua:6323: radius jewels that modify the jewel
// itself based on nearby allocated nodes.
var jewelSelfFuncs = map[string]any{
	"Adds 1 to maximum Life per 3 Intelligence in Radius":                                                    getPerStat("Life", "BASE", 0, "Int", 1.0/3),
	"Adds 1 to Maximum Life per 3 Intelligence Allocated in Radius":                                          getPerStat("Life", "BASE", 0, "Int", 1.0/3),
	"1% increased Evasion Rating per 3 Dexterity Allocated in Radius":                                        getPerStat("Evasion", "INC", 0, "Dex", 1.0/3),
	"1% increased Claw Physical Damage per 3 Dexterity Allocated in Radius":                                  getPerStat("PhysicalDamage", "INC", ModFlag.Claw, "Dex", 1.0/3),
	"1% increased Melee Physical Damage while Unarmed per 3 Dexterity Allocated in Radius":                   getPerStat("PhysicalDamage", "INC", ModFlag.Unarmed, "Dex", 1.0/3),
	"3% increased Totem Life per 10 Strength in Radius":                                                      getPerStat("TotemLife", "INC", 0, "Str", 3.0/10),
	"3% increased Totem Life per 10 Strength Allocated in Radius":                                            getPerStat("TotemLife", "INC", 0, "Str", 3.0/10),
	"Adds 1 maximum Lightning Damage to Attacks per 1 Dexterity Allocated in Radius":                         getPerStat("LightningMax", "BASE", ModFlag.Attack, "Dex", 1),
	"5% increased Chaos damage per 10 Intelligence from Allocated Passives in Radius":                        getPerStat("ChaosDamage", "INC", 0, "Int", 5.0/10),
	"-1 Strength per 1 Strength on Allocated Passives in Radius":                                             getPerStat("Str", "BASE", 0, "Str", -1),
	"1% additional Physical Damage Reduction per 10 Strength on Allocated Passives in Radius":                getPerStat("PhysicalDamageReduction", "BASE", 0, "Str", 1.0/10),
	"2% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", "INC", 0, "Str", 2.0/10),
	"3% increased Life Recovery Rate per 10 Strength on Allocated Passives in Radius":                        getPerStat("LifeRecoveryRate", "INC", 0, "Str", 3.0/10),
	"-1 Intelligence per 1 Intelligence on Allocated Passives in Radius":                                     getPerStat("Int", "BASE", 0, "Int", -1),
	"0.4% of Energy Shield Regenerated per Second for every 10 Intelligence on Allocated Passives in Radius": getPerStat("EnergyShieldRegenPercent", "BASE", 0, "Int", 0.4/10),
	"regenerate 0.4% of energy shield per second for every 10 Intelligence on Allocated Passives in Radius":  getPerStat("EnergyShieldRegenPercent", "BASE", 0, "Int", 0.4/10),
	"2% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", "INC", 0, "Int", 2.0/10),
	"3% increased Mana Recovery Rate per 10 Intelligence on Allocated Passives in Radius":                    getPerStat("ManaRecoveryRate", "INC", 0, "Int", 3.0/10),
	"-1 Dexterity per 1 Dexterity on Allocated Passives in Radius":                                           getPerStat("Dex", "BASE", 0, "Dex", -1),
	"2% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", "INC", 0, "Dex", 2.0/10),
	"3% increased Movement Speed per 10 Dexterity on Allocated Passives in Radius":                           getPerStat("MovementSpeed", "INC", 0, "Dex", 3.0/10),
	// ModParser.lua:6333
	"Dexterity and Intelligence from passives in Radius count towards Strength Melee Damage bonus": jewelNodeFunc(func(node any, out modStoreWriter, data Tag) {
		if n, ok := asJewelNode(node); ok {
			data["Dex"] = numValue(data["Dex"]) + sumModList(n.ModList(), "BASE", "Dex")
			data["Int"] = numValue(data["Int"]) + sumModList(n.ModList(), "BASE", "Int")
		} else if data["Dex"] != nil || data["Int"] != nil {
			out.NewMod("DexIntToMeleeBonus", "BASE", numValue(data["Dex"])+numValue(data["Int"]), data["modSource"])
		}
	}),
}

// sumModList mirrors node.modList:Sum("BASE", nil, name) for a plain list.
func sumModList(mods []*Mod, typ, name string) float64 {
	total := 0.0
	for _, m := range mods {
		if m.Type == typ && m.Name == name && len(m.Tags) == 0 {
			total += numValue(m.Value)
		}
	}
	return total
}

// jewelSelfUnallocFuncs — ModParser.lua:6354.
var jewelSelfUnallocFuncs = map[string]any{
	"+5% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius":      getPerStat("CritMultiplier", "BASE", 0, "Str", 5.0/10),
	"+7% to Critical Strike Multiplier per 10 Strength on Unallocated Passives in Radius":      getPerStat("CritMultiplier", "BASE", 0, "Str", 7.0/10),
	"2% reduced Life Recovery Rate per 10 Strength on Unallocated Passives in Radius":          getPerStat("LifeRecoveryRate", "INC", 0, "Str", -2.0/10),
	"+15 to maximum Mana per 10 Dexterity on Unallocated Passives in Radius":                   getPerStat("Mana", "BASE", 0, "Dex", 15.0/10),
	"+100 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius":            getPerStat("Accuracy", "BASE", 0, "Int", 100.0/10),
	"+125 to Accuracy Rating per 10 Intelligence on Unallocated Passives in Radius":            getPerStat("Accuracy", "BASE", 0, "Int", 125.0/10),
	"2% reduced Mana Recovery Rate per 10 Intelligence on Unallocated Passives in Radius":      getPerStat("ManaRecoveryRate", "INC", 0, "Int", -2.0/10),
	"+3% to Damage over Time Multiplier per 10 Intelligence on Unallocated Passives in Radius": getPerStat("DotMultiplier", "BASE", 0, "Int", 3.0/10),
	"2% reduced Movement Speed per 10 Dexterity on Unallocated Passives in Radius":             getPerStat("MovementSpeed", "INC", 0, "Dex", -2.0/10),
	"+125 to Accuracy Rating per 10 Dexterity on Unallocated Passives in Radius":               getPerStat("Accuracy", "BASE", 0, "Dex", 125.0/10),
	// ModParser.lua:6365,6381
	"Grants all bonuses of Unallocated Small Passive Skills in Radius":   grantsUnallocated("Normal"),
	"Grants all bonuses of Unallocated Notable Passive Skills in Radius": grantsUnallocated("Notable"),
}

// modListCollector is the small stand-in for new("ModList") the two
// grants-all-bonuses functions build up.
type modListCollector struct{ mods []*Mod }

func grantsUnallocated(nodeType string) jewelNodeFunc {
	return func(node any, out modStoreWriter, data Tag) {
		if n, ok := asJewelNode(node); ok {
			if n.Type() == nodeType {
				coll, _ := data["modList"].(*modListCollector)
				if coll == nil {
					coll = &modListCollector{}
					data["modList"] = coll
				}
				// Filter out "Condition:ConnectedTo" mods as these nodes are
				// not technically allocated by this jewel func
				for _, m := range out.ModsList() {
					if !strings.HasPrefix(m.Name, "Condition:ConnectedTo") {
						coll.mods = append(coll.mods, m)
					}
				}
			}
		} else if coll, ok := data["modList"].(*modListCollector); ok {
			out.AddList(coll)
		}
	}
}

// jewelThresholdFuncs — ModParser.lua:6425.
var jewelThresholdFuncs = map[string]any{
	"With at least 40 Dexterity in Radius, Frost Blades Melee Damage Penetrates 15% Cold Resistance":                      getThreshold("Dex", "ColdPenetration", "BASE", 15, ModFlag.Melee, Tag{"type": "SkillName", "skillName": "Frost Blades", "includeTransfigured": true}),
	"With at least 40 Dexterity in Radius, Melee Damage dealt by Frost Blades Penetrates 15% Cold Resistance":             getThreshold("Dex", "ColdPenetration", "BASE", 15, ModFlag.Melee, Tag{"type": "SkillName", "skillName": "Frost Blades", "includeTransfigured": true}),
	"With at least 40 Dexterity in Radius, Frost Blades has 25% increased Projectile Speed":                               getThreshold("Dex", "ProjectileSpeed", "INC", 25, Tag{"type": "SkillName", "skillName": "Frost Blades", "includeTransfigured": true}),
	"With at least 40 Dexterity in Radius, Ice Shot has 25% increased Area of Effect":                                     getThreshold("Dex", "AreaOfEffect", "INC", 25, Tag{"type": "SkillName", "skillName": "Ice Shot"}),
	"Ice Shot Pierces 5 additional Targets with 40 Dexterity in Radius":                                                   getThreshold("Dex", "PierceCount", "BASE", 5, Tag{"type": "SkillName", "skillName": "Ice Shot"}),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 3 additional Targets":                                         getThreshold("Dex", "PierceCount", "BASE", 3, Tag{"type": "SkillName", "skillName": "Ice Shot"}),
	"With at least 40 Dexterity in Radius, Ice Shot Pierces 5 additional Targets":                                         getThreshold("Dex", "PierceCount", "BASE", 5, Tag{"type": "SkillName", "skillName": "Ice Shot"}),
	"With at least 40 Intelligence in Radius, Frostbolt fires 2 additional Projectiles":                                   getThreshold("Int", "ProjectileCount", "BASE", 2, Tag{"type": "SkillName", "skillName": "Frostbolt"}),
	"With at least 40 Intelligence in Radius, Rolling Magma fires an additional Projectile":                               getThreshold("Int", "ProjectileCount", "BASE", 1, Tag{"type": "SkillName", "skillName": "Rolling Magma"}),
	"With at least 40 Intelligence in Radius, Rolling Magma has 10% increased Area of Effect per Chain":                   getThreshold("Int", "AreaOfEffect", "INC", 10, Tag{"type": "SkillName", "skillName": "Rolling Magma"}, Tag{"type": "PerStat", "stat": "Chain"}),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 40% more damage per chain":                              getThreshold("Int", "Damage", "MORE", 40, Tag{"type": "SkillName", "skillName": "Rolling Magma"}, Tag{"type": "PerStat", "stat": "Chain"}),
	"With at least 40 Intelligence in Radius, Rolling Magma deals 50% less damage":                                        getThreshold("Int", "Damage", "MORE", -50, Tag{"type": "SkillName", "skillName": "Rolling Magma"}),
	"With at least 40 Dexterity in Radius, Shrapnel Shot has 25% increased Area of Effect":                                getThreshold("Dex", "AreaOfEffect", "INC", 25, Tag{"type": "SkillName", "skillName": "Shrapnel Shot"}),
	"With at least 40 Dexterity in Radius, Shrapnel Shot's cone has a 50% chance to deal Double Damage":                   getThreshold("Dex", "DoubleDamageChance", "BASE", 50, Tag{"type": "SkillName", "skillName": "Shrapnel Shot"}, Tag{"type": "SkillPart", "skillPart": 2}),
	"With at least 40 Dexterity in Radius, Galvanic Arrow deals 50% increased Area Damage":                                getThreshold("Dex", "Damage", "INC", 50, Tag{"type": "SkillName", "skillName": "Galvanic Arrow", "includeTransfigured": true}, Tag{"type": "SkillPart", "skillPart": 2}),
	"With at least 40 Dexterity in Radius, Galvanic Arrow has 25% increased Area of Effect":                               getThreshold("Dex", "AreaOfEffect", "INC", 25, Tag{"type": "SkillName", "skillName": "Galvanic Arrow", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Freezing Pulse fires 2 additional Projectiles":                              getThreshold("Int", "ProjectileCount", "BASE", 2, Tag{"type": "SkillName", "skillName": "Freezing Pulse"}),
	"With at least 40 Intelligence in Radius, 25% increased Freezing Pulse Damage if you've Shattered an Enemy Recently":  getThreshold("Int", "Damage", "INC", 25, Tag{"type": "SkillName", "skillName": "Freezing Pulse"}, Tag{"type": "Condition", "var": "ShatteredEnemyRecently"}),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 10 additional Projectiles":                               getThreshold("Dex", "ProjectileCount", "BASE", 10, Tag{"type": "SkillName", "skillName": "Ethereal Knives", "includeTransfigured": true}),
	"With at least 40 Dexterity in Radius, Ethereal Knives fires 5 additional Projectiles":                                getThreshold("Dex", "ProjectileCount", "BASE", 5, Tag{"type": "SkillName", "skillName": "Ethereal Knives", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Molten Strike fires 2 additional Projectiles":                                   getThreshold("Str", "ProjectileCount", "BASE", 2, Tag{"type": "SkillName", "skillName": "Molten Strike", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Molten Strike has 25% increased Area of Effect":                                 getThreshold("Str", "AreaOfEffect", "INC", 25, Tag{"type": "SkillName", "skillName": "Molten Strike", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Molten Strike Projectiles Chain +1 time":                                        getThreshold("Str", "ChainCountMax", "BASE", 1, Tag{"type": "SkillName", "skillName": "Molten Strike", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Molten Strike fires 50% less Projectiles":                                       getThreshold("Str", "ProjectileCount", "MORE", -50, Tag{"type": "SkillName", "skillName": "Molten Strike", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, 25% of Glacial Hammer Physical Damage converted to Cold Damage":                 getThreshold("Str", "SkillPhysicalDamageConvertToCold", "BASE", 25, Tag{"type": "SkillName", "skillName": "Glacial Hammer", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage":                            getThreshold("Str", "DoubleDamageChance", "BASE", 20, Tag{"type": "SkillName", "skillName": "Heavy Strike"}),
	"With at least 40 Strength in Radius, Heavy Strike has a 20% chance to deal Double Damage.":                           getThreshold("Str", "DoubleDamageChance", "BASE", 20, Tag{"type": "SkillName", "skillName": "Heavy Strike"}),
	"With at least 40 Strength in Radius, Cleave has +1 to Radius per Nearby Enemy, up to +10":                            getThreshold("Str", "AreaOfEffect", "BASE", 1, Tag{"type": "Multiplier", "var": "NearbyEnemies", "limit": 10}, Tag{"type": "SkillName", "skillName": "Cleave", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Cleave has +0.1 metres to Radius per Nearby Enemy, up to a maximum of +1 metre": getThreshold("Str", "AreaOfEffect", "BASE", 1, Tag{"type": "Multiplier", "var": "NearbyEnemies", "limit": 10}, Tag{"type": "SkillName", "skillName": "Cleave", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Cleave grants Fortify on Hit":                                                   getThreshold("Str", "ExtraSkillMod", "LIST", Tag{"mod": flag("Condition:Fortified")}, Tag{"type": "SkillName", "skillName": "Cleave", "includeTransfigured": true}),
	"With at least 40 Strength in Radius, Hits with Cleave Fortify":                                                       getThreshold("Str", "ExtraSkillMod", "LIST", Tag{"mod": flag("Condition:Fortified")}, Tag{"type": "SkillName", "skillName": "Cleave", "includeTransfigured": true}),
	"With at least 40 Dexterity in Radius, Dual Strike has a 20% chance to deal Double Damage with the Main-Hand Weapon":  getThreshold("Dex", "DoubleDamageChance", "BASE", 20, Tag{"type": "SkillName", "skillName": "Dual Strike", "includeTransfigured": true}, Tag{"type": "Condition", "var": "MainHandAttack"}),
	`with at least 40 dexterity in radius, dual strike has ([0-9]+)% increased attack speed while wielding a claw`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Dex", "Speed", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Dual Strike", "includeTransfigured": true}, Tag{"type": "Condition", "var": "UsingClaw"})
	}),
	`with at least 40 dexterity in radius, dual strike has \+([0-9]+)% to critical strike multiplier while wielding a dagger`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Dex", "CritMultiplier", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Dual Strike", "includeTransfigured": true}, Tag{"type": "Condition", "var": "UsingDagger"})
	}),
	`with at least 40 dexterity in radius, dual strike has ([0-9]+)% increased accuracy rating while wielding a sword`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Dex", "Accuracy", "INC", c.n(1), Tag{"type": "SkillName", "skillName": "Dual Strike", "includeTransfigured": true}, Tag{"type": "Condition", "var": "UsingSword"})
	}),
	"With at least 40 Dexterity in Radius, Dual Strike Hits Intimidate Enemies for 4 seconds while wielding an Axe":                  getThreshold("Dex", "EnemyModifier", "LIST", Tag{"mod": flag("Condition:Intimidated")}, Tag{"type": "Condition", "var": "UsingAxe"}),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack has 100% increased Cooldown Recovery Speed":                getThreshold("Int", "MinionModifier", "LIST", Tag{"mod": mod("CooldownRecovery", "INC", 100, Tag{"type": "SkillId", "skillId": "ZombieSlam"})}),
	"With at least 40 Intelligence in Radius, Raised Zombies' Slam Attack deals 30% increased Damage":                                getThreshold("Int", "MinionModifier", "LIST", Tag{"mod": mod("Damage", "INC", 30, Tag{"type": "SkillId", "skillId": "ZombieSlam"})}),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Attack Damage for each Poison on the Enemy":               getThreshold("Dex", "Damage", "INC", 2, ModFlag.Attack, Tag{"type": "SkillName", "skillName": "Viper Strike", "includeTransfigured": true}, Tag{"type": "Multiplier", "actor": "enemy", "var": "PoisonStack"}),
	"With at least 40 Dexterity in Radius, Viper Strike deals 2% increased Damage with Hits and Poison for each Poison on the Enemy": getThreshold("Dex", "Damage", "INC", 2, int64(0), KeywordFlag.Hit|KeywordFlag.Poison, Tag{"type": "SkillName", "skillName": "Viper Strike", "includeTransfigured": true}, Tag{"type": "Multiplier", "actor": "enemy", "var": "PoisonStack"}),
	"With at least 40 Intelligence in Radius, Spark fires 2 additional Projectiles":                                                  getThreshold("Int", "ProjectileCount", "BASE", 2, Tag{"type": "SkillName", "skillName": "Spark", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Blight has 50% increased Hinder Duration":                                              getThreshold("Int", "SecondaryDuration", "INC", 50, Tag{"type": "SkillName", "skillName": "Blight", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Enemies Hindered by Blight take 25% increased Chaos Damage":                            getThreshold("Int", "ExtraSkillMod", "LIST", Tag{"mod": mod("ChaosDamageTaken", "INC", 25, Tag{"type": "GlobalEffect", "effectType": "Debuff", "effectName": "Hinder"})}, Tag{"type": "SkillName", "skillName": "Blight", "includeTransfigured": true}, Tag{"type": "ActorCondition", "actor": "enemy", "var": "Hindered"}),
	"With 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":                                getThreshold("Int", "SkillPhysicalDamageConvertToCold", "BASE", 20, Tag{"type": "SkillName", "skillName": "Glacial Cascade", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, 20% of Glacial Cascade Physical Damage Converted to Cold Damage":                       getThreshold("Int", "SkillPhysicalDamageConvertToCold", "BASE", 20, Tag{"type": "SkillName", "skillName": "Glacial Cascade", "includeTransfigured": true}),
	"With 40 total Intelligence and Dexterity in Radius, Elemental Hit and Wild Strike deal 50% less Fire Damage":                    getThreshold([]string{"Int", "Dex"}, "FireDamage", "MORE", -50, Tag{"type": "SkillName", "skillNameList": []any{"Elemental Hit", "Wild Strike"}, "includeTransfigured": true}),
	"With 40 total Strength and Intelligence in Radius, Elemental Hit and Wild Strike deal 50% less Cold Damage":                     getThreshold([]string{"Str", "Int"}, "ColdDamage", "MORE", -50, Tag{"type": "SkillName", "skillNameList": []any{"Elemental Hit", "Wild Strike"}, "includeTransfigured": true}),
	"With 40 total Dexterity and Strength in Radius, Elemental Hit and Wild Strike deal 50% less Lightning Damage":                   getThreshold([]string{"Dex", "Str"}, "LightningDamage", "MORE", -50, Tag{"type": "SkillName", "skillNameList": []any{"Elemental Hit", "Wild Strike"}, "includeTransfigured": true}),
	"With 40 total Intelligence and Dexterity in Radius, Prismatic Skills deal 50% less Fire Damage":                                 getThreshold([]string{"Int", "Dex"}, "FireDamage", "MORE", -50, Tag{"type": "SkillType", "skillType": SkillType.RandomElement}),
	"With 40 total Strength and Intelligence in Radius, Prismatic Skills deal 50% less Cold Damage":                                  getThreshold([]string{"Str", "Int"}, "ColdDamage", "MORE", -50, Tag{"type": "SkillType", "skillType": SkillType.RandomElement}),
	"With 40 total Dexterity and Strength in Radius, Prismatic Skills deal 50% less Lightning Damage":                                getThreshold([]string{"Dex", "Str"}, "LightningDamage", "MORE", -50, Tag{"type": "SkillType", "skillType": SkillType.RandomElement}),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw Chains +4 times":                                          getThreshold([]string{"Dex", "Str"}, "ChainCountMax", "BASE", 4, Tag{"type": "SkillName", "skillName": "Spectral Shield Throw", "includeTransfigured": true}),
	"With 40 total Dexterity and Strength in Radius, Spectral Shield Throw fires 75% less Shard Projectiles":                         getThreshold([]string{"Dex", "Str"}, "ProjectileCount", "MORE", -75, Tag{"type": "SkillName", "skillName": "Spectral Shield Throw", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Blight inflicts Withered for 2 seconds":                                                getThreshold("Int", "ExtraSkillMod", "LIST", Tag{"mod": mod("Condition:CanWither", "FLAG", true)}, Tag{"type": "SkillName", "skillName": "Blight", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Blight has 30% reduced Cast Speed":                                                     getThreshold("Int", "Speed", "INC", -30, Tag{"type": "SkillName", "skillName": "Blight", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Fireball cannot ignite":                                                                getThreshold("Int", "ExtraSkillMod", "LIST", Tag{"mod": flag("CannotIgnite")}, Tag{"type": "SkillName", "skillName": "Fireball"}),
	`with at least 40 intelligence in radius, fireball has \+([0-9]+)% chance to inflict scorch`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Int", "EnemyScorchChance", "BASE", c.n(1), Tag{"type": "SkillName", "skillName": "Fireball"})
	}),
	"With at least 40 Intelligence in Radius, Discharge has 60% less Area of Effect": getThreshold("Int", "AreaOfEffect", "MORE", -60, Tag{"type": "SkillName", "skillName": "Discharge", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Discharge Cooldown is 250 ms":          getThreshold("Int", "CooldownRecovery", "OVERRIDE", 0.25, Tag{"type": "SkillName", "skillName": "Discharge", "includeTransfigured": true}),
	"With at least 40 Intelligence in Radius, Discharge deals 60% less Damage":       getThreshold("Int", "Damage", "MORE", -60, Tag{"type": "SkillName", "skillName": "Discharge", "includeTransfigured": true}),
	`with at least 40 intelligence in radius, ([0-9]+)% of damage taken recouped as mana if you've warcried recently`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Int", "ManaRecoup", "BASE", c.n(1), Tag{"type": "Condition", "var": "UsedWarcryRecently"})
	}),
	`with at least 40 intelligence in radius, fireball projectiles gain radius as they travel farther, up to \+([0-9]+) radius`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Int", "AreaOfEffect", "BASE", c.n(1), Tag{"type": "DistanceRamp", "ramp": []any{[]any{0, 0}, []any{50, 1}}})
	}),
	`with at least 40 intelligence in radius, projectiles gain radius as they travel farther, up to a maximum of \+([0-9.]+) metres? to radius`: jewelFactory(func(c caps) jewelNodeFunc {
		return getThreshold("Int", "AreaOfEffect", "BASE", c.n(1)*10, Tag{"type": "DistanceRamp", "ramp": []any{[]any{0, 0}, []any{50, 1}}})
	}),
	// ModParser.lua:6489 — one line applying two independent threshold funcs.
	"With at least 40 Intelligence in Radius, Raised Spectres have a 50% chance to gain Soul Eater for 20 seconds on Kill": []jewelNodeFunc{
		getThreshold("Int", "MinionModifier", "LIST", Tag{"mod": mod("Condition:CanHaveSoulEater", "FLAG", true)}, Tag{"type": "SkillName", "skillName": "Raise Spectre", "includeTransfigured": true}),
		getThreshold("Int", "Condition:MinionCanHaveSoulEater", "FLAG", true),
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
		return func(node any, out modStoreWriter, data Tag) {
			if n, ok := asJewelNode(node); ok && n.ConqueredBy() {
				return
			}
			inner(node, out, data)
		}
	}
	for k, v := range jewelOtherFuncs {
		switch fv := v.(type) {
		case jewelNodeFunc:
			inner := fv
			out[asciiLower(k)] = jewelFuncEntry{typ: "Other", factory: func(c caps) jewelNodeFunc {
				return wrapOther(inner)
			}}
		case jewelFactory:
			factory := fv
			// Factory keys are pre-lowered regex; lowercasing again would wreck
			// their character classes.
			out[k] = jewelFuncEntry{typ: "Other", re: compileJewelKey(k), factory: func(c caps) jewelNodeFunc {
				return wrapOther(factory(c))
			}}
		}
	}
	for k, v := range jewelSelfFuncs {
		out[asciiLower(k)] = jewelFuncEntry{typ: "Self", nodeFn: v.(jewelNodeFunc)}
	}
	for k, v := range jewelSelfUnallocFuncs {
		out[asciiLower(k)] = jewelFuncEntry{typ: "SelfUnalloc", nodeFn: v.(jewelNodeFunc)}
	}
	for k, v := range jewelThresholdFuncs {
		switch fv := v.(type) {
		case jewelNodeFunc:
			out[asciiLower(k)] = jewelFuncEntry{typ: "Threshold", nodeFn: fv}
		case jewelFactory:
			out[k] = jewelFuncEntry{typ: "Threshold", re: compileJewelKey(k), factory: fv}
		case []jewelNodeFunc:
			funcs := fv
			out[asciiLower(k)] = jewelFuncEntry{typ: "Threshold", nodeFn: func(node any, w modStoreWriter, data Tag) {
				funcData, _ := data["funcData"].([]Tag)
				if funcData == nil {
					funcData = make([]Tag, len(funcs))
					for i := range funcData {
						funcData[i] = Tag{}
					}
					data["funcData"] = funcData
				}
				for i, f := range funcs {
					funcData[i]["modSource"] = data["modSource"]
					f(node, w, funcData[i])
				}
			}}
		}
	}
	return out
}

// compileJewelKey compiles a parametric jewel key (written pre-lowered so it
// matches the lowered line).
func compileJewelKey(k string) *regexp.Regexp {
	return regexp.MustCompile(k)
}
