package config

import (
	"math"
	"strings"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/internal/util"
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

// Set is one saved configuration set: the values the user chose, the
// placeholders standing in for the ones they did not, and the custom
// modifier blocks.
type Set struct {
	ID          int
	Title       string
	Input       map[Var]Value
	Placeholder map[Var]Value
	CustomMods  []CustomModBlock
}

// CustomModBlock is one <CustomModifierBlock>: free modifier text the
// user typed, applied as its own source.
type CustomModBlock struct {
	Title   string
	Enabled bool
	Text    string
}

// Tab is the configuration tab's state for one build. Input and
// Placeholder alias the active set's tables, the way the reference's tab
// does, because the option table's apply functions read them by name.
type Tab struct {
	Sets      map[int]*Set
	SetOrder  []int
	ActiveSet int

	Input       map[Var]Value
	Placeholder map[Var]Value

	// EnemyLevel is the level the enemy-stat placeholders are computed
	// at; UpdateLevel settles it before any option applies.
	EnemyLevel float64
	// CharacterLevel is the build's level, which the enemy level falls
	// back to.
	CharacterLevel float64

	// Mods and EnemyMods are the lists BuildModList fills.
	Mods      *modstore.List
	EnemyMods *modstore.List

	// disabled records options an apply function switched off. The
	// reference keeps this on the widget, so it survives across
	// BuildModList calls rather than resetting per pass; a freshly loaded
	// tab starts with none.
	disabled map[Var]bool
}

// NewTab returns a tab with one empty config set, as the reference's
// constructor plus NewConfigSet(1) does.
func NewTab(characterLevel float64) *Tab {
	t := &Tab{
		Sets:           map[int]*Set{},
		ActiveSet:      1,
		CharacterLevel: characterLevel,
		disabled:       map[Var]bool{},
	}
	t.NewSet(1, "Default")
	t.SetOrder = []int{1}
	t.setActive(1)
	return t
}

// NewSet creates a config set carrying every option's default state and
// default placeholder. An option with neither leaves both absent, which
// the reference distinguishes from a stored false or 0.
func (t *Tab) NewSet(id int, title string) *Set {
	s := &Set{
		ID:          id,
		Title:       title,
		Input:       map[Var]Value{},
		Placeholder: map[Var]Value{},
		CustomMods:  []CustomModBlock{{Title: "Default", Enabled: true}},
	}
	for _, opt := range Options {
		if opt.Default != nil {
			s.Input[opt.Var] = opt.Default
		}
		if opt.DefaultPlaceholder != nil {
			s.Placeholder[opt.Var] = opt.DefaultPlaceholder
		}
		if opt.DefaultIndex > 0 {
			s.Input[opt.Var] = opt.List[opt.DefaultIndex-1].Val
		}
	}
	t.Sets[id] = s
	return s
}

func (t *Tab) setActive(id int) {
	if t.Sets[id] == nil {
		if len(t.SetOrder) > 0 {
			id = t.SetOrder[0]
		} else {
			id = 1
		}
	}
	t.ActiveSet = id
	t.Input = t.Sets[id].Input
	t.Placeholder = t.Sets[id].Placeholder
}

// UpdateLevel settles the enemy level: an explicit input wins, then the
// placeholder, then the character's own level, each capped at the data
// table's maximum.
func (t *Tab) UpdateLevel() {
	if n, ok := NumOf(t.Input["enemyLevel"]); ok && n > 0 {
		t.EnemyLevel = math.Min(data.Misc.MaxEnemyLevel, n)
		return
	}
	if n, ok := NumOf(t.Placeholder["enemyLevel"]); ok && n > 0 {
		t.EnemyLevel = math.Min(data.Misc.MaxEnemyLevel, n)
		return
	}
	t.EnemyLevel = math.Min(data.Misc.MaxEnemyLevel, t.CharacterLevel)
}

// SetPlaceholder is the apply functions' `varControls[var]:SetPlaceholder(v,
// true)`: the control stringifies the value and hands it back to its own
// change handler, which stores it. #EVAL: that round trip quantizes a
// number to Lua's tostring precision, so the stored placeholder is not
// always the value the caller passed.
func (t *Tab) SetPlaceholder(v Var, value float64) {
	n, _ := util.Tonumber(util.FormatG14(value))
	t.Placeholder[v] = Num(n)
}

// ClearPlaceholder is `SetPlaceholder("", true)`: the control hands the
// empty string to its change handler, tonumber returns nil, and assigning
// nil removes the key. Callers that mean "no default" go through here.
func (t *Tab) ClearPlaceholder(v Var) {
	delete(t.Placeholder, v)
}

// PlaceholderNum reads a numeric placeholder, 0 when unset.
func (t *Tab) PlaceholderNum(v Var) float64 {
	n, _ := NumOf(t.Placeholder[v])
	return n
}

// InputNum reads a numeric input, 0 when unset.
func (t *Tab) InputNum(v Var) float64 {
	n, _ := NumOf(t.Input[v])
	return n
}

// InputStr reads a text or list input, "" when unset.
func (t *Tab) InputStr(v Var) string {
	s, _ := StrOf(t.Input[v])
	return s
}

// SelIndex is `varControls[var].selIndex`: the 1-based position of the
// option's current value in its list, or 0 when nothing matches - which
// the reference's `or 1` fallbacks then read as 1.
func (t *Tab) SelIndex(v Var) int {
	opt := byVar[v]
	if opt == nil {
		return 0
	}
	cur := t.Input[v]
	for i, entry := range opt.List {
		if entry.Val == cur {
			return i + 1
		}
	}
	return 0
}

// SelByValue is `varControls[var]:SelByValue(value)`: select a list
// option's entry by its stored value.
func (t *Tab) SelByValue(v Var, value Value) {
	opt := byVar[v]
	if opt == nil {
		return
	}
	for _, entry := range opt.List {
		if entry.Val == value {
			t.Input[v] = value
			return
		}
	}
}

// SetEnabled and Enabled carry the one piece of control state an apply
// function writes for a later one to read.
func (t *Tab) SetEnabled(v Var, on bool) { t.disabled[v] = !on }

// Enabled reports whether an earlier option disabled this one this pass.
func (t *Tab) Enabled(v Var) bool { return !t.disabled[v] }

// BuildModList runs every option's apply function in table order and
// fills Mods and EnemyMods. It settles the enemy level first, because the
// boss options compute their placeholders from it.
func (t *Tab) BuildModList() {
	t.Mods = modstore.NewList(nil)
	t.EnemyMods = modstore.NewList(nil)
	t.UpdateLevel()
	for i := range Options {
		opt := &Options[i]
		if opt.Apply == nil {
			continue
		}
		switch {
		case opt.Kind == KindCheck:
			if Truthy(t.Input[opt.Var]) {
				opt.Apply(Bool(true), t)
			}
		case opt.Kind.numeric():
			if v, use := numericApplies(t.Input[opt.Var], opt.Kind); use {
				opt.Apply(v, t)
			} else if v, use := numericApplies(t.Placeholder[opt.Var], opt.Kind); use {
				opt.Apply(v, t)
			}
		default: // KindList, KindText
			if v := t.Input[opt.Var]; Truthy(v) {
				opt.Apply(v, t)
			}
		}
	}
	t.applyCustomMods()
}

// applyCustomMods parses each enabled custom modifier block and adds the
// lines that parse cleanly, sourced by block title. A build with no block
// text falls back to the legacy single customMods string.
func (t *Tab) applyCustomMods() {
	set := t.Sets[t.ActiveSet]
	hasBlockText := false
	if set != nil {
		for _, block := range set.CustomMods {
			if !block.Enabled || block.Text == "" {
				continue
			}
			hasBlockText = true
			title := block.Title
			if title == "" {
				title = "Default"
			}
			t.addCustomLines(block.Text, "Custom:"+title)
		}
	}
	if !hasBlockText {
		if text := t.InputStr("customMods"); text != "" {
			t.addCustomLines(text, "Custom")
		}
	}
}

func (t *Tab) addCustomLines(text, source string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(util.StripEscapes(line))
		mods, extra, _ := modparser.Parse(line)
		if extra != "" {
			continue
		}
		for _, mod := range mods {
			if mod == nil {
				continue
			}
			t.Mods.AddMod(modparser.SetSource(mod, source))
		}
	}
}

// numericApplies is the reference's `input[var] and (input[var] ~= 0 or
// type == "countAllowZero")`: an unset or false value never applies, and
// a zero applies only where zero is meaningful. A non-number stored in a
// numeric slot compares unequal to 0 in Lua, so it applies.
func numericApplies(v Value, k Kind) (Value, bool) {
	if !Truthy(v) {
		return nil, false
	}
	if n, ok := NumOf(v); ok && n == 0 && k != KindCountAllowZero {
		return nil, false
	}
	return v, true
}
