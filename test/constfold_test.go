package test

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"io/fs"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// productionDirs are the packages whose arithmetic has to match the
// reference's. Test code is exempt: it compares, it does not compute.
var productionDirs = []string{
	"build", "calc", "config", "data", "export", "internal",
	"item", "modparser", "modstore", "skills", "tree", "cmd",
}

// TestNoUnsafeConstantFolding proves the port has no arithmetic that Go
// folds at compile time where the reference divides at runtime.
//
// Go evaluates an expression made only of untyped constants exactly, at
// arbitrary precision, and rounds once at the end. Lua rounds each literal
// to a double first and rounds again at every operation. Where a literal
// is not exactly representable the two land on different doubles:
// 1/0.033 folds to 30.303030303030305, while the reference computes
// 30.303030303030301.
//
// The fix at a site is to give the literal a type - `const d float64 =
// 0.033` - so it is rounded before the operation, which is the order the
// reference rounds in.
//
// This replaces a regex over seven hand-picked candidates. It evaluates
// every constant arithmetic expression in production source two ways,
// using the same exact-arithmetic package the compiler uses, so a site
// cannot hide in a const block or a table literal.
func TestNoUnsafeConstantFolding(t *testing.T) {
	var findings []string
	examined := 0
	fset := token.NewFileSet()
	for _, dir := range productionDirs {
		root := filepath.Join("..", dir)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				t.Fatalf("parse %s: %v", path, perr)
			}
			untyped := untypedConsts(file)
			ast.Inspect(file, func(n ast.Node) bool {
				be, ok := n.(*ast.BinaryExpr)
				if !ok {
					return true
				}
				exact, folded := evalFoldedExact(be, untyped)
				if !folded {
					return true
				}
				stepwise, ok := evalFoldedStepwise(be, untyped)
				if !ok {
					return true
				}
				examined++
				if exact != stepwise && !(math.IsNaN(exact) && math.IsNaN(stepwise)) {
					pos := fset.Position(be.Pos())
					findings = append(findings, formatFinding(pos.Filename, pos.Line, exact, stepwise))
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	sort.Strings(findings)
	for _, f := range findings {
		t.Errorf("%s", f)
	}
	if examined == 0 {
		t.Fatal("examined no constant expressions at all - the sweep is not reaching the source")
	}
	t.Logf("constant folding: %d fractional constant expressions examined, %d unsafe", examined, len(findings))
}

func formatFinding(file string, line int, exact, stepwise float64) string {
	return fmt.Sprintf("%s:%d: constant folded to %s where rounding each operand first gives %s"+
		" (round the OPERAND: give the non-representable literal a float64 type, or compute at runtime."+
		" Typing the RESULT does not help - the fold has already happened by then)",
		filepath.ToSlash(file), line,
		strconv.FormatFloat(exact, 'g', 17, 64), strconv.FormatFloat(stepwise, 'g', 17, 64))
}

// evalFoldedExact evaluates an expression the way the compiler folds it:
// every leaf exact, one rounding at the very end. It reports false unless
// the whole expression is constant AND at least one leaf is fractional -
// integer-only arithmetic is exact either way.
func evalFoldedExact(e ast.Expr, untyped map[string]ast.Expr) (float64, bool) {
	v, fractional, ok := constExact(e, untyped)
	if !ok || !fractional {
		return 0, false
	}
	f, _ := constant.Float64Val(constant.ToFloat(v))
	return f, true
}

func constExact(e ast.Expr, untyped map[string]ast.Expr) (val constant.Value, fractional, ok bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		switch x.Kind {
		case token.INT:
			return constant.MakeFromLiteral(x.Value, token.INT, 0), false, true
		case token.FLOAT:
			return constant.MakeFromLiteral(x.Value, token.FLOAT, 0), true, true
		}
		return nil, false, false
	case *ast.Ident:
		if def, ok := untyped[x.Name]; ok {
			return constExact(def, untyped)
		}
		return nil, false, false
	case *ast.ParenExpr:
		return constExact(x.X, untyped)
	case *ast.UnaryExpr:
		if x.Op != token.SUB && x.Op != token.ADD {
			return nil, false, false
		}
		v, fr, ok := constExact(x.X, untyped)
		if !ok {
			return nil, false, false
		}
		return constant.UnaryOp(x.Op, v, 0), fr, true
	case *ast.BinaryExpr:
		if !isArith(x.Op) {
			return nil, false, false
		}
		l, lf, ok := constExact(x.X, untyped)
		if !ok {
			return nil, false, false
		}
		r, rf, ok := constExact(x.Y, untyped)
		if !ok {
			return nil, false, false
		}
		if x.Op == token.QUO && !lf && !rf {
			// Integer division: a different question, and not this one.
			return nil, false, false
		}
		return constant.BinaryOp(l, x.Op, r), lf || rf, true
	}
	return nil, false, false
}

// evalFoldedStepwise evaluates the same expression the way the reference
// does: every leaf rounded to a double first, and the result rounded again
// at every operation.
func evalFoldedStepwise(e ast.Expr, untyped map[string]ast.Expr) (float64, bool) {
	switch x := e.(type) {
	case *ast.BasicLit:
		if x.Kind != token.INT && x.Kind != token.FLOAT {
			return 0, false
		}
		f, _ := constant.Float64Val(constant.ToFloat(constant.MakeFromLiteral(x.Value, x.Kind, 0)))
		return f, true
	case *ast.Ident:
		if def, ok := untyped[x.Name]; ok {
			return evalFoldedStepwise(def, untyped)
		}
		return 0, false
	case *ast.ParenExpr:
		return evalFoldedStepwise(x.X, untyped)
	case *ast.UnaryExpr:
		v, ok := evalFoldedStepwise(x.X, untyped)
		if !ok {
			return 0, false
		}
		if x.Op == token.SUB {
			return -v, true
		}
		return v, true
	case *ast.BinaryExpr:
		l, ok := evalFoldedStepwise(x.X, untyped)
		if !ok {
			return 0, false
		}
		r, ok := evalFoldedStepwise(x.Y, untyped)
		if !ok {
			return 0, false
		}
		switch x.Op {
		case token.ADD:
			return l + r, true
		case token.SUB:
			return l - r, true
		case token.MUL:
			return l * r, true
		case token.QUO:
			return l / r, true
		}
	}
	return 0, false
}

func isArith(op token.Token) bool {
	switch op {
	case token.ADD, token.SUB, token.MUL, token.QUO:
		return true
	}
	return false
}

// untypedConsts maps every UNTYPED constant declared in the file to its
// expression. A typed constant (const x float64 = 0.033) is already
// rounded at its declaration, which is the safe form and the fix, so only
// untyped ones can carry the fault.
func untypedConsts(file *ast.File) map[string]ast.Expr {
	out := map[string]ast.Expr{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || vs.Type != nil {
				continue // typed: rounded at declaration
			}
			for i, name := range vs.Names {
				if i < len(vs.Values) {
					out[name.Name] = vs.Values[i]
				}
			}
		}
	}
	return out
}
