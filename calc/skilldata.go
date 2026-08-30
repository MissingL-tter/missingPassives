// SkillData is the Lua activeSkill.skillData table: an open key set of
// scalars, filled from the granted effect, SkillData list mods and the
// offence stage. Each key lives in exactly one of the three maps; Lua
// truthiness (0 and "" are true) is what Flag reads.
package calc

import (
	"github.com/MissingL-tter/missingPassives/modparser"
	"github.com/MissingL-tter/missingPassives/modstore"
)

type SkillData struct {
	Nums  map[string]float64
	Flags map[string]bool
	Strs  map[string]string
}

func newSkillData() *SkillData {
	return &SkillData{Nums: map[string]float64{}, Flags: map[string]bool{}, Strs: map[string]string{}}
}

// N reads a numeric entry; absent or non-numeric reads 0 (numeric text
// coerces, as Lua arithmetic does).
func (d *SkillData) N(key string) float64 {
	if n, ok := d.Nums[key]; ok {
		return n
	}
	if s, ok := d.Strs[key]; ok {
		if n, ok := modparser.NumOf(modparser.Str(s)); ok {
			return n
		}
	}
	return 0
}

// Has reports key presence, whatever the value.
func (d *SkillData) Has(key string) bool {
	_, n := d.Nums[key]
	_, f := d.Flags[key]
	_, s := d.Strs[key]
	return n || f || s
}

// Flag reads an entry's truthiness: absent and false are the only
// falsy entries.
func (d *SkillData) Flag(key string) bool {
	if b, ok := d.Flags[key]; ok {
		return b
	}
	return d.Has(key)
}

// Str reads a string entry ("" when absent or not a string).
func (d *SkillData) Str(key string) string { return d.Strs[key] }

// Get reads an entry as an output value (absent when the key is missing).
func (d *SkillData) Get(key string) modstore.OutValue {
	if n, ok := d.Nums[key]; ok {
		return modstore.OutValue{Kind: modstore.OutNum, N: n}
	}
	if b, ok := d.Flags[key]; ok {
		return modstore.OutValue{Kind: modstore.OutBool, B: b}
	}
	if s, ok := d.Strs[key]; ok {
		return modstore.OutValue{Kind: modstore.OutStr, S: s}
	}
	return modstore.OutValue{}
}

// Set is a plain Lua field assignment: an absent value removes the key.
func (d *SkillData) Set(key string, v modstore.OutValue) {
	switch v.Kind {
	case modstore.OutNum:
		d.SetN(key, v.N)
	case modstore.OutBool:
		d.SetFlag(key, v.B)
	case modstore.OutStr:
		d.SetStr(key, v.S)
	default:
		d.Del(key)
	}
}

func (d *SkillData) SetN(key string, n float64) {
	delete(d.Flags, key)
	delete(d.Strs, key)
	d.Nums[key] = n
}

func (d *SkillData) SetFlag(key string, b bool) {
	delete(d.Nums, key)
	delete(d.Strs, key)
	d.Flags[key] = b
}

func (d *SkillData) SetStr(key string, s string) {
	delete(d.Nums, key)
	delete(d.Flags, key)
	d.Strs[key] = s
}

func (d *SkillData) Del(key string) {
	delete(d.Nums, key)
	delete(d.Flags, key)
	delete(d.Strs, key)
}

// Clone is copyTable(skillData): an independent copy.
func (d *SkillData) Clone() *SkillData {
	c := newSkillData()
	for k, v := range d.Nums {
		c.Nums[k] = v
	}
	for k, v := range d.Flags {
		c.Flags[k] = v
	}
	for k, v := range d.Strs {
		c.Strs[k] = v
	}
	return c
}

// valueOfOut converts an output entry to a mod value (nil when absent).
func valueOfOut(v modstore.OutValue) modparser.Value {
	switch v.Kind {
	case modstore.OutNum:
		return modparser.Num(v.N)
	case modstore.OutBool:
		return modparser.Bool(v.B)
	case modstore.OutStr:
		return modparser.Str(v.S)
	}
	return nil
}

// outValueOf converts a mod value scalar to an output entry (absent for
// nil or a non-scalar).
func outValueOf(v modparser.Value) modstore.OutValue {
	switch t := v.(type) {
	case modparser.Num:
		return modstore.OutValue{Kind: modstore.OutNum, N: float64(t)}
	case modparser.Bool:
		return modstore.OutValue{Kind: modstore.OutBool, B: bool(t)}
	case modparser.Str:
		return modstore.OutValue{Kind: modstore.OutStr, S: string(t)}
	}
	return modstore.OutValue{}
}
