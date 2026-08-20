package modparser

import "testing"

func TestScan(t *testing.T) {
	tbl := newScanTable("test", map[string]any{
		`^(\d+)% increased`: "INC",
		`^(\d+)% reduced`:   "RED",
		`(\d+)% more`:       "MORE",
		`^(\d+)% increase`:  "SHORTER", // same start, shorter match must lose
	}, false)
	val, rest, caps := scan("35% increased Fire Damage", tbl)
	if val != "INC" {
		t.Fatalf("scan picked %v", val)
	}
	if rest != " Fire Damage" {
		t.Fatalf("scan remainder %q", rest)
	}
	if cap1(caps, 1) != "35" {
		t.Fatalf("scan cap %q", cap1(caps, 1))
	}
	// Case-insensitive matching, original-case remainder.
	val, rest, _ = scan("10% MORE Damage", tbl)
	if val != "MORE" || rest != " Damage" {
		t.Fatalf("case handling: %v %q", val, rest)
	}
}

func TestModConstructor(t *testing.T) {
	m := mod("Damage", "MORE", 10.0, nil, ModFlag.Attack, KeywordFlag.Bleed, Tag{"type": "Condition", "var": "X"})
	if m.Flags != ModFlag.Attack || m.KeywordFlags != KeywordFlag.Bleed {
		t.Fatalf("positional flags wrong: %+v", m)
	}
	if len(m.Tags) != 1 {
		t.Fatalf("tags: %v", m.Tags)
	}
	m2 := mod("X", "BASE", 1.0, "A Source", Tag{"type": "Condition"})
	if m2.Source != "A Source" || len(m2.Tags) != 1 {
		t.Fatalf("source parse wrong: %+v", m2)
	}
	// A nil among the tags is a Lua table hole: numbering skips it.
	m3 := mod("X", "BASE", 1.0, nil, Tag{"type": "Condition"})
	if len(m3.Tags) != 2 || m3.Tags[0] != nil {
		t.Fatalf("nil-hole tags wrong: %v", m3.Tags)
	}
	want := `{"2":{"type":"Condition"},"flags":0,"keywordFlags":0,"name":"X","type":"BASE","value":1}`
	if got := Canon(m3); got != want {
		t.Fatalf("hole canon:\n got %s\nwant %s", got, want)
	}
}

func TestCanon(t *testing.T) {
	m := mod("FireDamage", "INC", 10.0)
	want := `{"flags":0,"keywordFlags":0,"name":"FireDamage","type":"INC","value":10}`
	if got := Canon(m); got != want {
		t.Fatalf("canon:\n got %s\nwant %s", got, want)
	}
	l := []any{mod("A", "BASE", 1.5)}
	want = `{"1":{"flags":0,"keywordFlags":0,"name":"A","type":"BASE","value":1.5}}`
	if got := Canon(l); got != want {
		t.Fatalf("canon list:\n got %s\nwant %s", got, want)
	}
}
