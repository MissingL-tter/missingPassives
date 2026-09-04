// Package util holds the reference's numeric semantics that the port keeps
// (number→text, %.14g quantisation, half-up rounding, tonumber) and Opt.
package util

// Opt is a number that may be absent, for inputs that have an empty state
// of their own (an empty config box, an item line without {range:}, no
// item in a slot) and a reader that answers differently for empty than for
// zero. It wraps float64 only: no reference reader tells nil from false,
// so a bool is a bool and the comparisons read a missing key as false.
type Opt[T float64] struct {
	V   T
	Set bool
}

// Some wraps a present value.
func Some[T float64](v T) Opt[T] { return Opt[T]{V: v, Set: true} }

// Or returns the value, or def when absent.
func (o Opt[T]) Or(def T) T {
	if o.Set {
		return o.V
	}
	return def
}
