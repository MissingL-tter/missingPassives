// LuaJIT's math.random PRNG (lib_math.c): a combined Tausworthe generator
// with period 2^223, seeded exactly as LuaJIT seeds it at startup (seed 0.0).
// legionPassives.lua consumes math.random() without seeding, so its output
// depends on this exact sequence.
//
// #EVAL: archive parity only — once the generated data format is Go-owned,
// random layout offsets in it deserve replacing, and this file goes away.

package export

import "math"

type luaPRNG struct {
	gen [4]uint64
}

func (rs *luaPRNG) step() uint64 {
	var r uint64
	twGen := func(i int, k, q, s uint) {
		z := rs.gen[i]
		z = (((z << q) ^ z) >> (k - s)) ^ ((z & (^uint64(0) << (64 - k))) << s)
		r ^= z
		rs.gen[i] = z
	}
	twGen(0, 63, 31, 18)
	twGen(1, 58, 19, 28)
	twGen(2, 55, 24, 7)
	twGen(3, 47, 21, 8)
	return (r & 0x000fffffffffffff) | 0x3ff0000000000000
}

// random returns the next math.random() double in [0,1).
func (rs *luaPRNG) random() float64 {
	return math.Float64frombits(rs.step()) - 1.0
}

func newLuaPRNG() *luaPRNG {
	rs := &luaPRNG{}
	r := uint32(0x11090601) // 64-k[i] as four 8 bit constants
	d := 0.0
	for i := 0; i < 4; i++ {
		m := uint64(1) << (r & 255)
		r >>= 8
		d = d*3.14159265358979323846 + 2.7182818284590452354
		u := math.Float64bits(d)
		if u < m { // ensure k[i] MSB of gen[i] are non-zero
			u += m
		}
		rs.gen[i] = u
	}
	for i := 0; i < 10; i++ {
		rs.step()
	}
	return rs
}

// PRNGTest exposes the first n math.random() values for verification.
func PRNGTest(n int) []float64 {
	rs := newLuaPRNG()
	out := make([]float64, n)
	for i := range out {
		out[i] = rs.random()
	}
	return out
}
