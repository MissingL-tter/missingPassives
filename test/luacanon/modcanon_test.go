package luacanon

import (
	"testing"

	"github.com/MissingL-tter/missingPassives/modparser"
)

func TestCanonMods(t *testing.T) {
	m := &modparser.Mod{Name: "FireDamage", Type: modparser.Inc, Value: modparser.Num(10)}
	want := `{"flags":0,"keywordFlags":0,"name":"FireDamage","type":"INC","value":10}`
	if got := CanonMods(m); got != want {
		t.Fatalf("canon:\n got %s\nwant %s", got, want)
	}
	l := []any{&modparser.Mod{Name: "A", Type: modparser.Base, Value: modparser.Num(1.5)}}
	want = `{"1":{"flags":0,"keywordFlags":0,"name":"A","type":"BASE","value":1.5}}`
	if got := CanonMods(l); got != want {
		t.Fatalf("canon list:\n got %s\nwant %s", got, want)
	}
	// A nil among the tags is a Lua table hole: numbering skips it.
	hole := &modparser.Mod{Name: "X", Type: modparser.Base, Value: modparser.Num(1), Tags: []modparser.Tag{nil, &modparser.CondTag{}}}
	want = `{"2":{"type":"Condition"},"flags":0,"keywordFlags":0,"name":"X","type":"BASE","value":1}`
	if got := CanonMods(hole); got != want {
		t.Fatalf("hole canon:\n got %s\nwant %s", got, want)
	}
}
