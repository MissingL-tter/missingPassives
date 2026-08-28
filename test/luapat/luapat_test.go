package luapat

import (
	"regexp"
	"testing"
)

func TestConvert(t *testing.T) {
	cases := []struct{ lua, want string }{
		{"^(%d+)%% increased", `^([0-9]+)% increased`},
		{"^([%+%-][%d%.]+)%%? to", `^([+\-][0-9.]+)%? to`},
		{"^removes? ([%d%.]+) ?o?f? ?y?o?u?r?", `^removes? ([0-9.]+) ?o?f? ?y?o?u?r?`},
		{"non%-fire", `non-fire`},
		{"l(.-)y", `l(.*?)y`},
		{"-1 strength per 1 strength", `\-1 strength per 1 strength`},
		{"a-b", `a*?b`}, // bare dash after an atom is Lua's lazy quantifier
		{"cost$", `cost$`},
		{"25%% chance", `25% chance`},
		{"[ct][ar][si][tg]g?e?r?s?", `[ct][ar][si][tg]g?e?r?s?`},
		{"x [special] y", `x [special] y`},
	}
	for _, c := range cases {
		got, err := Convert(c.lua)
		if err != nil {
			t.Errorf("Convert(%q): %v", c.lua, err)
			continue
		}
		if got != c.want {
			t.Errorf("Convert(%q) = %q, want %q", c.lua, got, c.want)
			continue
		}
		if _, err := regexp.Compile(got); err != nil {
			t.Errorf("Convert(%q) = %q does not compile: %v", c.lua, got, err)
		}
	}
}

func TestConvertRejects(t *testing.T) {
	for _, pat := range []string{"%b()", "%f[%a]x", "(%a+) %1"} {
		if _, err := Convert(pat); err == nil {
			t.Errorf("Convert(%q) should fail", pat)
		}
	}
}
