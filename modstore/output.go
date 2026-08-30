// The actor output table (Lua actor.output) as mod evaluation reads it:
// GetStat's `output[stat] or cfg.skillStats[stat] or 0` distinguishes an
// absent key, a present-but-false one (falls through), and a value.

package modstore

// OutKind is which of the three scalar shapes an output entry holds.
type OutKind uint8

const (
	OutAbsent OutKind = iota
	OutNum
	OutBool
	OutStr
)

// OutValue is one output entry; the zero value is an absent key.
type OutValue struct {
	Kind OutKind
	N    float64
	B    bool
	S    string
}

// Truthy is Lua truthiness: absent and false are the only falsy entries.
func (v OutValue) Truthy() bool {
	switch v.Kind {
	case OutAbsent:
		return false
	case OutBool:
		return v.B
	}
	return true
}

// Num reads the entry as a number; non-numbers read 0 (Lua arithmetic on
// them would error before reaching here in practice).
func (v OutValue) Num() float64 {
	if v.Kind == OutNum {
		return v.N
	}
	return 0
}

// Output is the typed output table; absent key != present false. A nil
// Output reads as empty.
type Output map[string]OutValue

// N reads a numeric entry; absent or non-numeric reads 0.
func (o Output) N(key string) float64 { return o[key].Num() }

// Has reports key presence, whatever the value.
func (o Output) Has(key string) bool { _, ok := o[key]; return ok }

// Flag reads an entry's truthiness.
func (o Output) Flag(key string) bool { return o[key].Truthy() }

// Str reads a string entry ("" when absent or not a string).
func (o Output) Str(key string) string {
	if v := o[key]; v.Kind == OutStr {
		return v.S
	}
	return ""
}

// Get reads an entry (absent when the key is missing).
func (o Output) Get(key string) OutValue { return o[key] }

// Set is a plain Lua field assignment: storing an absent value removes the key.
func (o Output) Set(key string, v OutValue) {
	if v.Kind == OutAbsent {
		delete(o, key)
		return
	}
	o[key] = v
}

func (o Output) SetN(key string, n float64)  { o[key] = OutValue{Kind: OutNum, N: n} }
func (o Output) SetFlag(key string, b bool)  { o[key] = OutValue{Kind: OutBool, B: b} }
func (o Output) SetStr(key string, s string) { o[key] = OutValue{Kind: OutStr, S: s} }
func (o Output) Del(key string)              { delete(o, key) }
