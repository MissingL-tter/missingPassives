// A small evaluator for the mod-constructor subset of Lua the skill and
// minion templates hand-write: mod(...) / flag(...) / skill(...) calls with
// literal arguments, table constructors, ModFlag/KeywordFlag/SkillType
// constants, and bit.bor. The templates' text reaches the runtime through
// the schema documents; this turns it into structured mods with no Lua
// involved. Every evaluated mod is verified against the booted archive by
// the game-data comparison.

package data

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/MissingL-tter/missingPassives/modparser"
)

// modConstants resolves ModFlag.X / KeywordFlag.X / SkillType.X.
var modConstants = map[string]int64{}

func init() {
	for prefix, s := range map[string]any{
		"ModFlag":     modparser.ModFlag,
		"KeywordFlag": modparser.KeywordFlag,
		"SkillType":   modparser.SkillType,
	} {
		rv := reflect.ValueOf(s)
		rt := rv.Type()
		for i := 0; i < rt.NumField(); i++ {
			modConstants[prefix+"."+rt.Field(i).Name] = rv.Field(i).Int()
		}
	}
}

type modExprParser struct {
	s   string
	pos int
}

// evalModLine evaluates one template mod line. Comment lines and empty
// lines yield (nil, false). The result is *modparser.Mod, or *modparser.D
// for the upstream typo where a tag sits in the flags slot.
func evalModLine(line string) (any, bool) {
	p := &modExprParser{s: line}
	p.skipSpace()
	if p.eof() || strings.HasPrefix(p.s[p.pos:], "--") {
		return nil, false
	}
	switch v := p.expr().(type) {
	case *modparser.Mod, *modparser.D:
		return v, true
	default:
		panic("data: template line did not evaluate to a mod: " + line)
	}
}

func (p *modExprParser) fail(msg string) {
	panic(fmt.Sprintf("data: mod expression %q at %d: %s", p.s, p.pos, msg))
}

func (p *modExprParser) eof() bool { return p.pos >= len(p.s) }

func (p *modExprParser) skipSpace() {
	for !p.eof() {
		c := p.s[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
			continue
		}
		// comments run to end of line
		if c == '-' && p.pos+1 < len(p.s) && p.s[p.pos+1] == '-' {
			for !p.eof() && p.s[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		break
	}
}

func (p *modExprParser) peek() byte {
	p.skipSpace()
	if p.eof() {
		p.fail("unexpected end")
	}
	return p.s[p.pos]
}

func (p *modExprParser) accept(c byte) bool {
	p.skipSpace()
	if !p.eof() && p.s[p.pos] == c {
		p.pos++
		return true
	}
	return false
}

func (p *modExprParser) expect(c byte) {
	if !p.accept(c) {
		p.fail("expected " + string(c))
	}
}

func isIdentByte(c byte) bool {
	return c == '_' || c == '.' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func (p *modExprParser) ident() string {
	p.skipSpace()
	start := p.pos
	for !p.eof() && isIdentByte(p.s[p.pos]) {
		p.pos++
	}
	if start == p.pos {
		p.fail("expected identifier")
	}
	return p.s[start:p.pos]
}

// expr handles the template subset's arithmetic: `1/3`, `-2/3 * 100`.
func (p *modExprParser) expr() any {
	left := p.primary()
	for {
		p.skipSpace()
		if p.eof() {
			return left
		}
		var op byte
		switch c := p.s[p.pos]; c {
		case '*', '/', '+':
			op = c
		case '-':
			// binary minus only when a number follows a numeric left side
			if _, ok := left.(float64); !ok {
				return left
			}
			op = c
		default:
			return left
		}
		ln, ok := left.(float64)
		if !ok {
			p.fail("arithmetic on non-number")
		}
		p.pos++
		rv := p.primary()
		rn, ok := rv.(float64)
		if !ok {
			p.fail("arithmetic on non-number")
		}
		switch op {
		case '*':
			left = ln * rn
		case '/':
			left = ln / rn
		case '+':
			left = ln + rn
		case '-':
			left = ln - rn
		}
	}
}

func (p *modExprParser) primary() any {
	switch c := p.peek(); {
	case c == '"':
		return p.stringLit()
	case c == '{':
		return p.table()
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	default:
		name := p.ident()
		switch name {
		case "nil":
			return nil
		case "true":
			return true
		case "false":
			return false
		}
		if p.accept('(') {
			return p.call(name)
		}
		if v, ok := modConstants[name]; ok {
			return v
		}
		p.fail("unknown identifier " + name)
		return nil
	}
}

func (p *modExprParser) stringLit() string {
	p.expect('"')
	var b strings.Builder
	for {
		if p.eof() {
			p.fail("unterminated string")
		}
		c := p.s[p.pos]
		p.pos++
		if c == '"' {
			return b.String()
		}
		if c == '\\' {
			if p.eof() {
				p.fail("bad escape")
			}
			e := p.s[p.pos]
			p.pos++
			switch e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\', '"', '\'':
				b.WriteByte(e)
			default:
				p.fail("unhandled escape \\" + string(e))
			}
			continue
		}
		b.WriteByte(c)
	}
}

func (p *modExprParser) number() float64 {
	p.skipSpace()
	start := p.pos
	if !p.eof() && p.s[p.pos] == '-' {
		p.pos++
	}
	for !p.eof() {
		c := p.s[p.pos]
		if (c >= '0' && c <= '9') || c == '.' || c == 'e' || c == 'E' || c == 'x' ||
			(c == '-' && p.pos > start && (p.s[p.pos-1] == 'e' || p.s[p.pos-1] == 'E')) ||
			(c >= 'a' && c <= 'f' && strings.Contains(p.s[start:p.pos], "x")) ||
			(c >= 'A' && c <= 'F' && strings.Contains(p.s[start:p.pos], "x")) {
			p.pos++
			continue
		}
		break
	}
	n, err := parseLuaNumber(p.s[start:p.pos])
	if err != nil {
		p.fail("bad number " + p.s[start:p.pos])
	}
	return n
}

// call evaluates the known constructor functions.
func (p *modExprParser) call(name string) any {
	var args []any
	if !p.accept(')') {
		for {
			args = append(args, p.expr())
			if p.accept(')') {
				break
			}
			p.expect(',')
		}
	}
	switch name {
	case "mod":
		if len(args) < 3 {
			p.fail("mod() needs at least 3 arguments")
		}
		return makeSkillMod(args[0].(string), args[1].(string), args[2], args[3:]...)
	case "flag":
		if len(args) < 1 {
			p.fail("flag() needs a name")
		}
		return makeFlagMod(args[0].(string), args[1:]...)
	case "skill":
		if len(args) < 2 {
			p.fail("skill() needs key and value")
		}
		return makeSkillDataMod(args[0].(string), args[1], args[2:]...)
	case "bit.bor", "bor":
		var out int64
		for _, a := range args {
			out |= toFlagInt(a)
		}
		return out
	}
	p.fail("unknown function " + name)
	return nil
}

// table evaluates a constructor: hash-only tables become map[string]any,
// array-only become []any (mixed tables do not occur in this subset).
func (p *modExprParser) table() any {
	p.expect('{')
	kv := map[string]any{}
	var arr []any
	for {
		if p.accept('}') {
			break
		}
		if c := p.peek(); c == '[' {
			p.pos++
			key := p.stringLit()
			p.expect(']')
			p.expect('=')
			kv[key] = p.expr()
		} else if c != '"' && c != '{' && c != '-' && !(c >= '0' && c <= '9') {
			// identifier: either `name = value` or a bare expression
			save := p.pos
			name := p.ident()
			if p.accept('=') {
				kv[name] = p.expr()
			} else {
				p.pos = save
				arr = append(arr, p.expr())
			}
		} else {
			arr = append(arr, p.expr())
		}
		if !p.accept(',') {
			p.expect('}')
			break
		}
	}
	if len(arr) > 0 && len(kv) > 0 {
		p.fail("mixed table constructor")
	}
	if len(arr) > 0 {
		return arr
	}
	return kv
}

func toFlagInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	}
	panic(fmt.Sprintf("data: non-numeric flag value %T", v))
}

// makeSkillMod mirrors Data.lua's makeSkillMod. When a non-number sits in
// the flags/keywordFlags slot (an upstream typo the reference keeps), the
// result is a raw *modparser.D table instead of a structured mod.
func makeSkillMod(name, typ string, value any, rest ...any) any {
	numeric := func(v any) bool {
		switch v.(type) {
		case nil, int64, float64:
			return true
		}
		return false
	}
	if (len(rest) > 0 && !numeric(rest[0])) || (len(rest) > 1 && !numeric(rest[1])) {
		kv := map[string]any{"name": name, "type": typ, "flags": int64(0), "keywordFlags": int64(0)}
		if value != nil {
			kv["value"] = value
		}
		var arr []any
		if len(rest) > 0 && rest[0] != nil {
			kv["flags"] = rest[0]
		}
		if len(rest) > 1 && rest[1] != nil {
			kv["keywordFlags"] = rest[1]
		}
		if len(rest) > 2 {
			arr = append(arr, rest[2:]...)
		}
		return &modparser.D{Arr: arr, KV: kv}
	}
	m := &modparser.Mod{Name: name, Type: typ, Value: value}
	if len(rest) > 0 && rest[0] != nil {
		m.Flags = toFlagInt(rest[0])
	}
	if len(rest) > 1 && rest[1] != nil {
		m.KeywordFlags = toFlagInt(rest[1])
	}
	if len(rest) > 2 {
		m.Tags = append(m.Tags, rest[2:]...)
	}
	return m
}

func makeFlagMod(name string, tags ...any) any {
	return makeSkillMod(name, "FLAG", true, append([]any{int64(0), int64(0)}, tags...)...)
}

func makeSkillDataMod(dataKey string, dataValue any, tags ...any) any {
	value := map[string]any{"key": dataKey}
	if dataValue != nil {
		value["value"] = dataValue
	}
	return makeSkillMod("SkillData", "LIST", value, append([]any{int64(0), int64(0)}, tags...)...)
}
