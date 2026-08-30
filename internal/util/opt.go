// Package util holds the reference's numeric semantics that the port keeps
// (number→text, %.14g quantisation, half-up rounding, tonumber) and Opt[T].
package util

// Opt is a value that may be absent, where the reference distinguishes
// absent from the zero value and the archive comparison sees the difference.
type Opt[T any] struct {
	V   T
	Set bool
}

// Some wraps a present value.
func Some[T any](v T) Opt[T] { return Opt[T]{V: v, Set: true} }

// Or returns the value, or def when absent.
func (o Opt[T]) Or(def T) T {
	if o.Set {
		return o.V
	}
	return def
}
