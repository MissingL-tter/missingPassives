package modparser

import (
	"testing"
)

func TestScan(t *testing.T) {
	tbl := newScanTable("test", map[string]string{
		`^(\d+)% increased`: "INC",
		`^(\d+)% reduced`:   "RED",
		`(\d+)% more`:       "MORE",
		`^(\d+)% increase`:  "SHORTER", // same start, shorter match must lose
	}, false)
	val, found, rest, caps := scan("35% increased Fire Damage", tbl)
	if !found || val != "INC" {
		t.Fatalf("scan picked %v", val)
	}
	if rest != " Fire Damage" {
		t.Fatalf("scan remainder %q", rest)
	}
	if cap1(caps, 1) != "35" {
		t.Fatalf("scan cap %q", cap1(caps, 1))
	}
	// Case-insensitive matching, original-case remainder.
	val, _, rest, _ = scan("10% MORE Damage", tbl)
	if val != "MORE" || rest != " Damage" {
		t.Fatalf("case handling: %v %q", val, rest)
	}
}

func TestModConstructor(t *testing.T) {
	m := modf("Damage", More, Num(10), FlagAttack, KeywordBleed, &CondTag{Var: "X"})
	if m.Flags != FlagAttack || m.KeywordFlags != KeywordBleed {
		t.Fatalf("positional flags wrong: %+v", m)
	}
	if len(m.Tags) != 1 {
		t.Fatalf("tags: %v", m.Tags)
	}
	m2 := mods("X", Base, Num(1), "A Source", &CondTag{})
	if m2.Source != "A Source" || !m2.SourceSet || len(m2.Tags) != 1 {
		t.Fatalf("source wrong: %+v", m2)
	}
	// A nil among the tags is a Lua table hole: numbering skips it, the
	// ipairs view stops before it.
	m3 := mod("X", Base, Num(1), nil, &CondTag{})
	if len(m3.Tags) != 2 || m3.Tags[0] != nil || len(ModTags(m3)) != 0 {
		t.Fatalf("nil-hole tags wrong: %v", m3.Tags)
	}
	// Trailing nils vanish.
	m4 := mod("X", Base, Num(1), &CondTag{}, nil)
	if len(m4.Tags) != 1 {
		t.Fatalf("trailing nil kept: %v", m4.Tags)
	}
}

func TestTagParamsRoundTrip(t *testing.T) {
	tags := []Tag{
		&MultiplierTag{Var: "Rage", Div: opt(5), Limit: opt(-30), LimitNegTotal: true, GlobalLimit: opt(100), GlobalLimitKey: "K"},
		&StatTag{StatKind: TagPercentStat, Stat: "Armour", Percent: opt(1)},
		&SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", Sockets: []float64{1, 2}},
		&SlotTag{SlotKind: TagSocketedIn, SlotName: "{SlotName}", SocketsAll: true},
		&ItemCondTag{ItemSlot: "Helmet", ElderCond: true},
		&DistanceRampTag{Ramp: Pairs{{35, 0}, {70, 1}}},
		&SkillTypeTag{SkillTypeList: []SkillTypeID{SkillTypeInstant, 0, SkillTypeTriggered}},
	}
	for _, tag := range tags {
		back, ok := TagFromParams(tag.Kind().String(), tag.Params())
		if !ok || FormatTag(back) != FormatTag(tag) {
			t.Fatalf("round trip lost %s: %s vs %s", tag.Kind(), FormatTag(tag), FormatTag(back))
		}
	}
}

func TestCodecRoundTrip(t *testing.T) {
	mods := []*Mod{
		mods("Damage", More, Num(-10), "Src", &CondTag{Var: "X", Neg: true}, nil, &MultiplierTag{Var: "Rage", Div: opt(5)}),
		mod("ExtraSkill", List, SkillRef{SkillID: "Fireball", Level: opt(20), Triggered: true}),
		mod("JewelData", List, DataRef{Key: "conqueredBy", Value: ConqueredBy{Seed: 1234, Conqueror: &Conqueror{Kind: ConquerorVaal, Index: 2, V2: true}}}),
		mod("ChainCountMax", Base, Num(m_huge), &MultiplierTag{Var: "X", Limit: opt(m_huge)}),
		mod("EnemyModifier", List, ModRef{Mod: flag("Condition:Blinded")}, &SkillPartTag{PartList: []float64{}}),
	}
	blob := EncodeMods(mods)
	back := DecodeMods(blob)
	if string(EncodeMods(back)) != string(blob) {
		t.Fatalf("codec not stable:\n%s\n%s", blob, EncodeMods(back))
	}
	for i := range mods {
		if FormatMod(back[i]) != FormatMod(mods[i]) {
			t.Fatalf("codec changed mod %d: %s vs %s", i, FormatMod(mods[i]), FormatMod(back[i]))
		}
	}
}
