package skills

// Gem lookup helpers shared with calc: name resolution (SkillsTab:FindSkillGem)
// and the in-game stat-requirement formula (calcLib.getGemStatRequirement).

import (
	"math"
	"sort"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
)

// GetGemStatRequirement ports calcLib.getGemStatRequirement (the in-game
// formula).
func GetGemStatRequirement(level float64, isSupport bool, multi float64) float64 {
	if multi == 0 {
		return 0
	}
	statType := 0.7
	if isSupport {
		statType = 0.5
	}
	req := util.RoundHalfUp((20+(level-3)*3)*math.Pow(multi/100, 0.9)*statType, 0)
	if req < 14 {
		return 0
	}
	return req
}

// FindSkillGem ports SkillsTab:FindSkillGem (L1076): match a gem by name
// through increasingly broad abbreviation patterns. Returns nil when the
// name is unrecognised or ambiguous (the reference reports an error message
// either way; nothing compared carries it). The reference iterates
// pairs(data.gems) -- hash order -- which only affects which two names an
// ambiguity error cites; the match result is order-independent, and this
// port scans in sorted id order.
func FindSkillGem(nameSpec string) *data.Gem {
	type matcher func(name string) bool
	isAlpha := func(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
	lowerRun := func(s string, i int) int { // len of [a-z]+ run at i
		n := 0
		for i+n < len(s) && s[i+n] >= 'a' && s[i+n] <= 'z' {
			n++
		}
		return n
	}
	noSpaces := strings.ReplaceAll(nameSpec, " ", "")

	// 1. Exact match, case-insensitive.
	exact := func(name string) bool { return strings.EqualFold(name, nameSpec) }

	// 2. Simple abbreviation ("CtF" -> "Cold to Fire"): each spec letter
	// starts a word and is followed by one or more lowercase letters;
	// non-letters in the spec match literally. Subject is " "+name, anchored
	// both ends.
	simpleAbbrev := func(name string) bool {
		s := " " + name
		i := 0
		for k := 0; k < len(nameSpec); k++ {
			c := nameSpec[k]
			if isAlpha(c) {
				if i >= len(s) || s[i] != ' ' {
					return false
				}
				i++
				if i >= len(s) || s[i] != c {
					return false
				}
				i++
				run := lowerRun(s, i)
				if run == 0 {
					return false
				}
				i += run
			} else {
				if i >= len(s) || s[i] != c {
					return false
				}
				i++
			}
		}
		return i == len(s)
	}

	// 3. Abbreviated words ("CldFr" -> "Cold to Fire"): lowercase spec
	// letters may be preceded by lowercase runs; anchored, must end in a
	// lowercase run. Greedy with backtracking over the optional runs is
	// needed for faithfulness; implement recursively.
	var wordAbbrevAt func(s string, i, k int) bool
	wordAbbrevAt = func(s string, i, k int) bool {
		if k == len(noSpaces) {
			run := lowerRun(s, i)
			return run > 0 && i+run == len(s)
		}
		c := noSpaces[k]
		if c >= 'a' && c <= 'z' {
			// "%l*c": try every split of a lowercase run before c.
			for skip := 0; ; skip++ {
				j := i + skip
				if j < len(s) && s[j] == c && wordAbbrevAt(s, j+1, k+1) {
					return true
				}
				if j >= len(s) || !(s[j] >= 'a' && s[j] <= 'z') {
					return false
				}
			}
		}
		if i < len(s) && s[i] == c {
			return wordAbbrevAt(s, i+1, k+1)
		}
		return false
	}
	wordAbbrev := func(name string) bool { return wordAbbrevAt(" "+name, 1, 0) }

	// 4. Global abbreviation ("CtoF" -> "Cold to Fire"): spec letters appear
	// in order anywhere (case-sensitive); unanchored tail.
	globalAbbrev := func(name string) bool {
		s := " " + name
		i := 0
		for k := 0; k < len(noSpaces); k++ {
			c := noSpaces[k]
			j := strings.IndexByte(s[i:], c)
			if j < 0 {
				return false
			}
			i += j + 1
		}
		return true
	}

	// 5. The same, case-insensitive.
	globalAbbrevFold := func(name string) bool {
		s := strings.ToLower(" " + name)
		spec := strings.ToLower(noSpaces)
		i := 0
		for k := 0; k < len(spec); k++ {
			j := strings.IndexByte(s[i:], spec[k])
			if j < 0 {
				return false
			}
			i += j + 1
		}
		return true
	}

	gemIds := make([]string, 0, len(data.Gems))
	for id := range data.Gems {
		gemIds = append(gemIds, id)
	}
	sort.Strings(gemIds)
	for _, match := range []matcher{exact, simpleAbbrev, wordAbbrev, globalAbbrev, globalAbbrevFold} {
		var found *data.Gem
		for _, id := range gemIds {
			g := data.Gems[id]
			if match(g.Name) {
				if found != nil {
					return nil // ambiguous
				}
				found = g
			}
		}
		if found != nil {
			return found
		}
	}
	return nil
}
