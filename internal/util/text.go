package util

import "strings"

// FoldText is Common.lua sanitiseText: strip <...> spans, fold the unicode
// hyphen family to "-" and a-/o-umlaut to ascii (UTF-8 and cp1252 forms;
// both cases, where the reference folds only lowercase), and replace any
// remaining high byte with "?" — but only when a byte 128-255 or '<' occurs.
func FoldText(text string) string {
	if !strings.ContainsFunc(text, func(r rune) bool { return r == '<' || r > 127 }) {
		// The reference returns nil here (the and-chain falls through);
		// every caller treats that as "unchanged".
		return text
	}
	text = StripBalanced(text, '<', '>')
	for _, rp := range [...][2]string{
		{"‐", "-"}, {"‑", "-"}, {"‒", "-"}, {"–", "-"},
		{"—", "-"}, {"―", "-"}, {"−", "-"},
		{"ä", "a"}, {"ö", "o"}, {"Ä", "A"}, {"Ö", "O"},
		// single-byte: Windows-1252 and similar
		{"\x96", "-"}, {"\x97", "-"}, {"\xe4", "a"}, {"\xf6", "o"},
		{"\xc4", "A"}, {"\xd6", "O"},
	} {
		text = strings.ReplaceAll(text, rp[0], rp[1])
	}
	// unsupported
	b := []byte(text)
	for i, c := range b {
		if c >= 128 {
			b[i] = '?'
		}
	}
	return string(b)
}

// StripBalanced is gsub("%b<>", ""): remove balanced open..close spans.
func StripBalanced(s string, open, close byte) string {
	var sb strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == open {
			depth := 1
			j := i + 1
			for j < len(s) && depth > 0 {
				if s[j] == open {
					depth++
				} else if s[j] == close {
					depth--
				}
				j++
			}
			if depth == 0 {
				i = j
				continue
			}
			// Unbalanced: no match, keep the byte.
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}
