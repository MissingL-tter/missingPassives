package export

import "math"

// Byte readers matching Common.lua's bytesTo* helpers. Offsets are 0-based
// (the Lua originals are 1-based; call sites here convert).

func bytesToUInt(b []byte, o int) uint32 {
	var v uint32
	for i := 0; i < 4; i++ {
		if o+i < len(b) {
			v |= uint32(b[o+i]) << (8 * i)
		}
	}
	return v
}

func bytesToInt(b []byte, o int) int32 {
	return int32(bytesToUInt(b, o))
}

func bytesToUShort(b []byte, o int) uint16 {
	var v uint16
	for i := 0; i < 2; i++ {
		if o+i < len(b) {
			v |= uint16(b[o+i]) << (8 * i)
		}
	}
	return v
}

func bytesToULong(b []byte, o int) uint64 {
	return uint64(bytesToUInt(b, o)) + uint64(bytesToUInt(b, o+4))<<32
}

// bytesToFloat matches Common.lua's manual float32 decoding, including its
// quirks: denormals collapse to signed zero, and exponent 128 (inf/NaN bits)
// decodes as a huge finite number.
func bytesToFloat(b []byte, o int) float64 {
	u := bytesToUInt(b, o)
	s := 1.0
	if u>>31 != 0 {
		s = -1.0
	}
	e := int((u>>23)&0xFF) - 127
	if e == -127 {
		return 0 * s
	}
	m := 1.0
	for i := 0; i < 23; i++ {
		if u>>uint(i)&1 != 0 {
			m += math.Ldexp(1, i-23)
		}
	}
	return s * m * math.Ldexp(1, e)
}

// codePointToUTF8 matches Common.lua's codePointToUTF8, including returning
// "?" for surrogate code points and anything above 0x10FFFF.
func codePointToUTF8(cp int) []byte {
	switch {
	case cp >= 0xD800 && cp <= 0xDFFF:
		return []byte{'?'}
	case cp <= 0x7F:
		return []byte{byte(cp)}
	case cp <= 0x07FF:
		return []byte{byte(0xC0 + cp>>6), byte(0x80 + cp&0x3F)}
	case cp <= 0xFFFF:
		return []byte{byte(0xE0 + cp>>12), byte(0x80 + cp>>6&0x3F), byte(0x80 + cp&0x3F)}
	case cp <= 0x10FFFF:
		return []byte{byte(0xF0 + cp>>18), byte(0x80 + cp>>12&0x3F), byte(0x80 + cp>>6&0x3F), byte(0x80 + cp&0x3F)}
	default:
		return []byte{'?'}
	}
}

// convertUTF16to8 matches Common.lua's convertUTF16to8: decode UTF-16LE code
// units starting at 0-based offset o until a zero unit or the end of the
// buffer, pairing surrogates the way the Lua does (a pending high surrogate
// survives intervening non-surrogate units).
func convertUTF16to8(b []byte, o int) string {
	var out []byte
	highSurr := -1
	for i := o; i+1 < len(b); i += 2 {
		cu := int(b[i]) + int(b[i+1])<<8
		switch {
		case cu == 0:
			return string(out)
		case cu >= 0xD800 && cu <= 0xDBFF:
			highSurr = cu - 0xD800
		case cu >= 0xDC00 && cu <= 0xDFFF:
			if highSurr >= 0 {
				out = append(out, codePointToUTF8(highSurr*1024+cu-0xDC00+0x010000)...)
				highSurr = -1
			}
		default:
			out = append(out, codePointToUTF8(cu)...)
		}
	}
	return string(out)
}
