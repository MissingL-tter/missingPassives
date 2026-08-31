// Package config ports the configuration tab's logic half:
// Modules/ConfigOptions.lua's option table and the parts of
// Classes/ConfigTab.lua that turn a saved <Config> element into the two
// modifier lists the calc consumes. The tab's widgets, undo, config-set
// management popups and ConfigVisibility's per-option relevance are view
// and are not ported.
//
// The option table is the enumeration: every configuration variable the
// application knows is one Option here, and nothing outside the table
// names one.
package config

// Var is a configuration variable's name, as it appears in the option
// table and in a saved build's <Input name="..."> elements.
type Var string

// Kind is an option's value type (the reference's varData.type). It
// decides how BuildModList reads the option and whether the placeholder
// stands in for a missing input.
type Kind uint8

const (
	// KindCheck is a boolean; it applies only when true.
	KindCheck Kind = iota
	// KindCount is a positive number; 0 and absent both skip the apply.
	KindCount
	// KindCountAllowZero is a number whose 0 is meaningful.
	KindCountAllowZero
	// KindInteger is a signed whole number; 0 skips the apply.
	KindInteger
	// KindFloat is a fractional number; 0 skips the apply.
	KindFloat
	// KindList is a choice among the option's List entries.
	KindList
	// KindText is free text.
	KindText
)

// numeric reports the kinds BuildModList reads through the placeholder
// fallback.
func (k Kind) numeric() bool {
	switch k {
	case KindCount, KindCountAllowZero, KindInteger, KindFloat:
		return true
	}
	return false
}

// Value is a stored configuration value: exactly one of Bool, Num or Str.
// The zero interface means the variable is unset, which the reference
// distinguishes from false and from 0.
type Value interface{ isConfigValue() }

// Bool is a check option's state, and the boolean form of a saved input.
type Bool bool

// Num is a numeric option's value.
type Num float64

// Str is a list option's selected value, or a text option's contents.
type Str string

func (Bool) isConfigValue() {}
func (Num) isConfigValue()  {}
func (Str) isConfigValue()  {}

// Truthy is the reference's `if input[var] then`: only an unset value and
// an explicit false are falsy. 0 and "" are true, as in Lua.
func Truthy(v Value) bool {
	if v == nil {
		return false
	}
	b, ok := v.(Bool)
	return !ok || bool(b)
}

// NumOf reads a value as a number, reporting whether it was one.
func NumOf(v Value) (float64, bool) {
	n, ok := v.(Num)
	return float64(n), ok
}

// StrOf reads a value as text, reporting whether it was text.
func StrOf(v Value) (string, bool) {
	s, ok := v.(Str)
	return string(s), ok
}

// ListEntry is one choice of a list option: the value stored when that
// choice is selected. The label the tab shows is view and is not carried.
type ListEntry struct {
	Val Value
}

// Option is one configuration variable.
type Option struct {
	Var  Var
	Kind Kind
	// List is the choices of a KindList option, in display order.
	List []ListEntry
	// DefaultIndex selects a List entry (1-based, as the reference
	// writes it); 0 means no default.
	DefaultIndex int
	// Default is the value a fresh config set starts with. A list
	// option's default comes from DefaultIndex instead.
	Default Value
	// DefaultPlaceholder is the placeholder a fresh config set starts
	// with, standing in for an unset numeric input.
	DefaultPlaceholder Value
	// Apply contributes this option's modifiers. Nil means the option
	// carries no modifiers of its own - the calc reads it straight off
	// ConfigInput, or it only steers other options.
	Apply func(v Value, t *Tab)
}
