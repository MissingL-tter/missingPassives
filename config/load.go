package config

import (
	"strconv"
	"strings"
	"unicode"
)

// XMLConfig is a saved build's <Config> element.
type XMLConfig struct {
	ActiveConfigSet string         `xml:"activeConfigSet,attr"`
	Sets            []XMLConfigSet `xml:"ConfigSet"`
	// Inputs, Placeholders and Blocks are the pre-config-set form, where
	// the elements sat directly under <Config>.
	Inputs       []XMLInput `xml:"Input"`
	Placeholders []XMLInput `xml:"Placeholder"`
	Blocks       []XMLBlock `xml:"CustomModifierBlock"`
}

// XMLConfigSet is one <ConfigSet>.
type XMLConfigSet struct {
	ID           string     `xml:"id,attr"`
	Title        string     `xml:"title,attr"`
	Inputs       []XMLInput `xml:"Input"`
	Placeholders []XMLInput `xml:"Placeholder"`
	Blocks       []XMLBlock `xml:"CustomModifierBlock"`
}

// XMLInput is one <Input> or <Placeholder>: a name plus exactly one of the
// three value attributes.
type XMLInput struct {
	Name    string `xml:"name,attr"`
	Number  string `xml:"number,attr"`
	String  string `xml:"string,attr"`
	Boolean string `xml:"boolean,attr"`
}

// XMLBlock is one <CustomModifierBlock>.
type XMLBlock struct {
	Title   string `xml:"title,attr"`
	Enabled string `xml:"enabled,attr"`
	Text    string `xml:",chardata"`
}

// Load builds the tab from a saved <Config> element. Every config set
// starts from the option defaults, then takes what the file stored.
func Load(x *XMLConfig, characterLevel float64) *Tab {
	t := &Tab{
		Sets:           map[int]*Set{},
		ActiveSet:      1,
		CharacterLevel: characterLevel,
		disabled:       map[Var]bool{},
	}
	if len(x.Sets) == 0 {
		s := t.NewSet(1, "Default")
		t.SetOrder = []int{1}
		applyElems(s, x.Inputs, x.Placeholders)
		s.CustomMods = append(s.CustomMods, blocks(x.Blocks)...)
	}
	for _, xs := range x.Sets {
		id, err := strconv.Atoi(strings.TrimSpace(xs.ID))
		if err != nil {
			id = len(t.Sets) + 1
		}
		title := xs.Title
		if title == "" {
			title = "Default"
		}
		s := t.NewSet(id, title)
		t.SetOrder = append(t.SetOrder, id)
		// A config set replaces the default block list outright.
		s.CustomMods = blocks(xs.Blocks)
		applyElems(s, xs.Inputs, xs.Placeholders)
	}
	// Legacy builds kept their custom modifiers in a single input.
	for _, s := range t.Sets {
		legacy, _ := StrOf(s.Input["customMods"])
		switch {
		case legacy != "" && emptyBlocks(s.CustomMods):
			s.CustomMods = []CustomModBlock{{Title: "Default", Enabled: true, Text: legacy}}
		case len(s.CustomMods) == 0:
			s.CustomMods = []CustomModBlock{{Title: "Default", Enabled: true}}
		}
		delete(s.Input, "customMods")
	}
	active := 1
	if n, err := strconv.Atoi(strings.TrimSpace(x.ActiveConfigSet)); err == nil {
		active = n
	}
	t.setActive(active)
	return t
}

// emptyBlocks reports the states the migration treats as "no blocks yet".
func emptyBlocks(blocks []CustomModBlock) bool {
	return len(blocks) == 0 || (len(blocks) == 1 && blocks[0].Text == "")
}

func blocks(xs []XMLBlock) []CustomModBlock {
	var out []CustomModBlock
	for _, b := range xs {
		title := b.Title
		if title == "" {
			title = "Default"
		}
		out = append(out, CustomModBlock{
			Title: title,
			// An absent attribute means enabled.
			Enabled: b.Enabled == "true" || b.Enabled == "",
			Text:    b.Text,
		})
	}
	return out
}

func applyElems(s *Set, inputs, placeholders []XMLInput) {
	for _, in := range inputs {
		if in.Name == "" {
			continue
		}
		if v, ok := inputValue(in); ok {
			s.Input[Var(in.Name)] = v
		}
	}
	for _, ph := range placeholders {
		if ph.Name == "" {
			continue
		}
		if ph.Number != "" {
			if n, err := strconv.ParseFloat(strings.TrimSpace(ph.Number), 64); err == nil {
				s.Placeholder[Var(ph.Name)] = Num(n)
			}
			continue
		}
		// A string-valued <Placeholder> is stored as an INPUT by
		// the reference, not as a placeholder.
		if ph.String != "" {
			s.Input[Var(ph.Name)] = Str(ph.String)
		}
	}
}

// inputValue reads an <Input>'s value, applying the two stored-value
// migrations the reference does on load.
func inputValue(in XMLInput) (Value, bool) {
	switch {
	case in.Number != "":
		n, err := strconv.ParseFloat(strings.TrimSpace(in.Number), 64)
		if err != nil {
			// tonumber failed: the reference stores nil, leaving the
			// option at its default.
			return nil, false
		}
		return Num(n), true
	case in.String != "":
		switch in.Name {
		case "enemyIsBoss":
			return Str(migrateEnemyIsBoss(in.String)), true
		case "presetBossSkills":
			// <=3.20 named the uber variants "Uber <skill>".
			return Str(strings.TrimPrefix(in.String, "Uber ")), true
		}
		return Str(in.String), true
	case in.Boolean != "":
		return Bool(in.Boolean == "true"), true
	}
	return nil, false
}

// migrateEnemyIsBoss lower-cases the stored value, title-cases each word,
// and folds the three renamed presets onto their current names.
func migrateEnemyIsBoss(s string) string {
	s = titleWords(strings.ToLower(s))
	s = strings.ReplaceAll(s, "Uber Atziri", "Boss")
	s = strings.ReplaceAll(s, "Shaper", "Pinnacle")
	s = strings.ReplaceAll(s, "Sirus", "Pinnacle")
	return s
}

// titleWords is the reference's gsub("(%l)(%w*)", upper(first)..rest): it
// upper-cases a lowercase letter that begins a run of word characters.
func titleWords(s string) string {
	out := []rune(s)
	prevWord := false
	for i, r := range out {
		isWord := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		if !prevWord && unicode.IsLower(r) {
			out[i] = unicode.ToUpper(r)
		}
		prevWord = isWord
	}
	return string(out)
}
