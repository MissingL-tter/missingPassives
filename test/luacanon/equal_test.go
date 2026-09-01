package luacanon

import "testing"

// TestEqualWithinIgnoresOrder pins the property the differential relies
// on: a Lua array reaches the canon as an object keyed "1".."n", and two
// such objects holding the same elements agree however they are ordered.
// Without this a comparison fails on any reordering whatever the values
// are, which is a defect in the comparison rather than a disagreement
// between the programs.
func TestEqualWithinIgnoresOrder(t *testing.T) {
	cases := []struct {
		name, got, want string
		equal           bool
	}{
		{
			name:  "reversed array of objects",
			got:   `{"1":{"name":"A","value":1},"2":{"name":"B","value":2}}`,
			want:  `{"2":{"name":"A","value":1},"1":{"name":"B","value":2}}`,
			equal: true,
		},
		{
			name:  "reversed array of scalars",
			got:   `{"1":10,"2":20,"3":30}`,
			want:  `{"1":30,"2":20,"3":10}`,
			equal: true,
		},
		{
			name:  "same length, one element differs",
			got:   `{"1":10,"2":20}`,
			want:  `{"1":10,"2":21}`,
			equal: false,
		},
		{
			name:  "a duplicate is not a reordering",
			got:   `{"1":10,"2":10}`,
			want:  `{"1":10,"2":20}`,
			equal: false,
		},
		{
			name:  "lengths differ",
			got:   `{"1":10,"2":20,"3":30}`,
			want:  `{"1":10,"2":20}`,
			equal: false,
		},
		{
			name:  "keyed object still compares by key, not as a bag",
			got:   `{"Str":10,"Dex":20}`,
			want:  `{"Str":20,"Dex":10}`,
			equal: false,
		},
		{
			name:  "elements agreeing only past 14 digits still match",
			got:   `{"1":0.27462500000000006,"2":5}`,
			want:  `{"2":5,"1":0.27462500000000001}`,
			equal: true,
		},
		{
			name:  "nested arrays reorder independently",
			got:   `{"1":{"1":"a","2":"b"},"2":{"1":"c"}}`,
			want:  `{"2":{"1":"c"},"1":{"2":"a","1":"b"}}`,
			equal: true,
		},
		// The boundary. Two mods sharing every match field and differing
		// only in value read as the same bag either way round - which is
		// exactly right for a set, and exactly wrong for a list that
		// ReplaceModInternal/ConvertModInternal walk taking match #1
		// (modstore/list.go:22,36 - they compare name, type, flags and
		// source, never value). So mod-list SEQUENCES stay on a positional
		// comparison; this is what that decision rests on.
		{
			name:  "same-key mods differing in value reorder freely",
			got:   `{"1":{"name":"Armour","value":105},"2":{"name":"Armour","value":80}}`,
			want:  `{"1":{"name":"Armour","value":80},"2":{"name":"Armour","value":105}}`,
			equal: true,
		},
		// ...but a value that actually changes is still caught, whatever
		// the order. This is the property that lets the loosened
		// comparisons stay honest.
		{
			name:  "a changed value is caught despite reordering",
			got:   `{"1":{"name":"Armour","value":105},"2":{"name":"Evasion","value":80}}`,
			want:  `{"1":{"name":"Evasion","value":80},"2":{"name":"Armour","value":106}}`,
			equal: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diffs, _, err := EqualWithin(tc.got, tc.want)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := len(diffs) == 0; got != tc.equal {
				t.Errorf("equal = %v, want %v (diffs: %v)", got, tc.equal, diffs)
			}
		})
	}
}
