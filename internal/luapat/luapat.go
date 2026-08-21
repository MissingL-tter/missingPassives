// Package luapat converts Lua patterns to Go regular expressions. It exists
// for the transition away from the reference's Lua tables: the shipped parser
// tables are plain Go regex, and this converter is used by the one-time source
// rewriter and by the table-archive test to map the reference's keys onto the
// converted ones. It is deleted together with the Lua.
package luapat

import (
	"fmt"
	"strings"
)

// Character class expansions, written as the inside of a bracket expression so
// they can be spliced into sets as well as used standalone. Go's \s lacks \v,
// so Lua's %s is spelled out.
var classes = map[byte]string{
	'a': "a-zA-Z",
	'c': `\x00-\x1f\x7f`,
	'd': "0-9",
	'l': "a-z",
	'p': `!-\/:-@\[-` + "`" + `{-~`,
	's': ` \t\n\v\f\r`,
	'u': "A-Z",
	'w': "0-9a-zA-Z",
	'x': "0-9a-fA-F",
}

const regexMeta = `\.+*?()|[]{}^$`

// Convert translates one Lua pattern into an equivalent Go regexp source
// string. The constructs Lua supports beyond regular languages (%b, %f,
// back-references) are reported as errors; the parser's tables use none.
func Convert(pat string) (string, error) {
	var sb strings.Builder
	i := 0
	prevAtom := false // whether a quantifiable atom precedes position i

	if strings.HasPrefix(pat, "^") {
		sb.WriteByte('^')
		i = 1
	}

	for i < len(pat) {
		c := pat[i]
		switch c {
		case '%':
			i++
			if i >= len(pat) {
				return "", fmt.Errorf("pattern ends with %%: %q", pat)
			}
			e := pat[i]
			switch {
			case e == 'b':
				return "", fmt.Errorf("%%b has no regexp equivalent: %q", pat)
			case e == 'f':
				return "", fmt.Errorf("%%f has no regexp equivalent: %q", pat)
			case e >= '1' && e <= '9':
				return "", fmt.Errorf("back-reference %%%c has no RE2 equivalent: %q", e, pat)
			case isLetter(e):
				set, ok := classes[lower(e)]
				if !ok {
					return "", fmt.Errorf("unknown class %%%c: %q", e, pat)
				}
				if isUpper(e) {
					sb.WriteString("[^" + set + "]")
				} else {
					sb.WriteString("[" + set + "]")
				}
			default:
				writeLiteral(&sb, e)
			}
			prevAtom = true
			i++
		case '[':
			end, err := setEnd(pat, i)
			if err != nil {
				return "", err
			}
			converted, err := convertSet(pat[i:end])
			if err != nil {
				return "", err
			}
			sb.WriteString(converted)
			prevAtom = true
			i = end
		case '(':
			if i+1 < len(pat) && pat[i+1] == ')' {
				return "", fmt.Errorf("position capture () has no regexp equivalent: %q", pat)
			}
			sb.WriteByte('(')
			prevAtom = false
			i++
		case ')':
			sb.WriteByte(')')
			prevAtom = true
			i++
		case '.':
			sb.WriteByte('.')
			prevAtom = true
			i++
		case '*', '+', '?':
			if prevAtom {
				sb.WriteByte(c)
				prevAtom = false // Lua does not allow stacked quantifiers
			} else {
				// A quantifier with nothing before it is a literal in Lua.
				writeLiteral(&sb, c)
				prevAtom = true
			}
			i++
		case '-':
			if prevAtom {
				sb.WriteString("*?") // Lua's lazy quantifier
				prevAtom = false
			} else {
				sb.WriteString(`\-`)
				prevAtom = true
			}
			i++
		case '$':
			if i == len(pat)-1 {
				sb.WriteByte('$')
			} else {
				sb.WriteString(`\$`)
				prevAtom = true
			}
			i++
		case '^':
			// Only anchors at position 0 (handled above); literal elsewhere.
			sb.WriteString(`\^`)
			prevAtom = true
			i++
		default:
			writeLiteral(&sb, c)
			prevAtom = true
			i++
		}
	}
	return sb.String(), nil
}

func writeLiteral(sb *strings.Builder, b byte) {
	if strings.IndexByte(regexMeta, b) >= 0 {
		sb.WriteByte('\\')
	}
	sb.WriteByte(b)
}

// setEnd returns the index just past the closing bracket, honouring Lua's rule
// that a ']' in the first position (after optional ^) is literal and that %x
// escapes one character.
func setEnd(pat string, start int) (int, error) {
	i := start + 1
	if i < len(pat) && pat[i] == '^' {
		i++
	}
	first := true
	for i < len(pat) {
		switch {
		case pat[i] == '%':
			i += 2
		case pat[i] == ']' && !first:
			return i + 1, nil
		default:
			i++
		}
		first = false
	}
	return 0, fmt.Errorf("unterminated set: %q", pat[start:])
}

// convertSet rewrites one bracket expression, expanding %classes inside it.
func convertSet(set string) (string, error) {
	body := set[1 : len(set)-1]
	var sb strings.Builder
	sb.WriteByte('[')
	i := 0
	if strings.HasPrefix(body, "^") {
		sb.WriteByte('^')
		i = 1
	}
	for i < len(body) {
		c := body[i]
		if c == '%' {
			i++
			if i >= len(body) {
				return "", fmt.Errorf("set ends with %%: %q", set)
			}
			e := body[i]
			if isLetter(e) {
				cls, ok := classes[lower(e)]
				if !ok || isUpper(e) {
					return "", fmt.Errorf("class %%%c inside a set cannot be spliced: %q", e, set)
				}
				sb.WriteString(cls)
			} else {
				writeSetLiteral(&sb, e)
			}
			i++
			continue
		}
		// A bare '-' between two characters is a range in both languages;
		// pass it through. Everything else is literal.
		if c == '-' && i > 0 && i < len(body)-1 {
			sb.WriteByte('-')
		} else {
			writeSetLiteral(&sb, c)
		}
		i++
	}
	sb.WriteByte(']')
	return sb.String(), nil
}

func writeSetLiteral(sb *strings.Builder, b byte) {
	if b == '\\' || b == ']' || b == '^' || b == '-' || b == '[' {
		sb.WriteByte('\\')
	}
	sb.WriteByte(b)
}

func isLetter(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
func isUpper(b byte) bool  { return b >= 'A' && b <= 'Z' }
func lower(b byte) byte {
	if isUpper(b) {
		return b + 32
	}
	return b
}
