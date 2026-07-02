// Lunex lang — AST coercion fixer

// Package formatter — coerce.go
//
// FixCoercions walks the AST and rewrites BinaryExpr nodes where the two
// operands are known to be incompatible types at format time.
//
// Rules applied
//   - number + string  → str(number) + string
//   - string + number  → string + str(number)
//   - bool   + string  → str(bool)   + string
//   - string + bool    → string + str(bool)
//   - number - string  → number - num(string)   (and *, /)
//   - string - number  → num(string) - number
//
// "Known type" means one side is a literal (NumberLit, StringLit, BoolLit)
// or a wrapping call that already resolves to a type (str(), num()).
// Variable identifiers are left alone — we can't know their runtime type.
//
// The fix is conservative: it only rewrites when at least one side is a
// literal so the type is unambiguous.
package formatter

import "lunex/internal/ast"

// FixCoercions mutates the AST tree in-place, wrapping implicit-coercion
// operands with explicit conversion calls. Returns the same root node.
func FixCoercions(root *ast.Node) *ast.Node {
	walkFix(root)
	return root
}

// walkFix recursively visits every node and fixes BinaryExpr coercions.
func walkFix(n *ast.Node) {
	if n == nil {
		return
	}

	// Fix children first (bottom-up), so nested exprs are already clean.
	walkFix(n.Left)
	walkFix(n.Right)
	walkFix(n.Body)
	walkFix(n.Init)
	walkFix(n.Test)
	walkFix(n.Consequent)
	walkFix(n.Alternate)
	walkFix(n.Arg)
	walkFix(n.Expr)
	walkFix(n.Stmt)
	walkFix(n.Object)
	walkFix(n.Callee)
	walkFix(n.Subject)
	walkFix(n.Lo)
	walkFix(n.Hi)
	walkFix(n.Count)
	walkFix(n.Ms)
	walkFix(n.Channel)
	walkFix(n.Guard)
	walkFix(n.Declaration)
	walkFix(n.Extends)
	walkFix(n.CatchBlock)
	walkFix(n.FinallyBlock)
	for _, c := range n.Args {
		walkFix(c)
	}
	for _, c := range n.Elements {
		walkFix(c)
	}
	for _, c := range n.Body_ {
		walkFix(c)
	}
	for _, c := range n.Decorators {
		walkFix(c)
	}
	for _, c := range n.Exprs {
		walkFix(c)
	}
	for _, p := range n.Properties {
		walkFix(p.Value)
		walkFix(p.Arg)
		walkFix(p.Body)
	}
	for _, m := range n.Methods {
		walkFix(m.Body)
		walkFix(m.Init)
	}
	for _, c := range n.Cases {
		walkFix(c.Body)
		walkFix(c.Guard)
	}
	for _, sc := range n.SelectCases {
		walkFix(sc.Body)
		walkFix(sc.Channel)
	}
	for _, pm := range n.Params {
		walkFix(pm.DefaultVal)
	}

	// Fix this node if it's a BinaryExpr.
	if n.Type == ast.BinaryExpr {
		fixBinaryCoercion(n)
	}
}

// ── type inference ────────────────────────────────────────────────────────────

type staticType int

const (
	typeUnknown staticType = iota
	typeNumber
	typeString
	typeBool
)

// inferType returns the static type of a node if it can be determined without
// execution. Only literals and known conversion calls are classified.
func inferType(n *ast.Node) staticType {
	if n == nil {
		return typeUnknown
	}
	switch n.Type {
	case ast.NumberLit:
		return typeNumber
	case ast.StringLit, ast.TemplateLit:
		return typeString
	case ast.BoolLit:
		return typeBool
	case ast.CallExpr:
		// str(...) → string, num(...) → number
		if n.Callee != nil && n.Callee.Type == ast.Identifier {
			switch n.Callee.Name {
			case "str", "String":
				return typeString
			case "num", "Number", "parseInt", "parseFloat":
				return typeNumber
			case "bool", "Boolean":
				return typeBool
			}
		}
	case ast.BinaryExpr:
		// Propagate type from a same-type binary expression.
		l, r := inferType(n.Left), inferType(n.Right)
		if l == r && l != typeUnknown {
			return l
		}
		// string + anything with + stays string after our fix.
		if n.Op == "+" && (l == typeString || r == typeString) {
			return typeString
		}
	}
	return typeUnknown
}

// ── coercion wrapping ─────────────────────────────────────────────────────────

// wrapStr wraps node in a str() call. Returns a new CallExpr node.
func wrapStr(n *ast.Node) *ast.Node {
	return &ast.Node{
		Type:   ast.CallExpr,
		Line:   n.Line,
		Col:    n.Col,
		Callee: &ast.Node{Type: ast.Identifier, Name: "str", Line: n.Line, Col: n.Col},
		Args:   []*ast.Node{n},
	}
}

// wrapNum wraps node in a num() call. Returns a new CallExpr node.
func wrapNum(n *ast.Node) *ast.Node {
	return &ast.Node{
		Type:   ast.CallExpr,
		Line:   n.Line,
		Col:    n.Col,
		Callee: &ast.Node{Type: ast.Identifier, Name: "num", Line: n.Line, Col: n.Col},
		Args:   []*ast.Node{n},
	}
}

// isAlreadyWrapped returns true if n is already a str() or num() call,
// to avoid double-wrapping on repeated fmt passes.
func isAlreadyWrapped(n *ast.Node, fn string) bool {
	if n == nil || n.Type != ast.CallExpr {
		return false
	}
	if n.Callee == nil || n.Callee.Type != ast.Identifier {
		return false
	}
	return n.Callee.Name == fn
}

// ── rule application ──────────────────────────────────────────────────────────

// fixBinaryCoercion inspects a BinaryExpr and, if the two sides are
// statically known to be incompatible, rewrites the node in-place.
func fixBinaryCoercion(n *ast.Node) {
	if n.Left == nil || n.Right == nil {
		return
	}
	lType := inferType(n.Left)
	rType := inferType(n.Right)

	// Only act when at least one side is a known literal type.
	// If both are unknown (e.g. two identifier refs), leave it alone.
	if lType == typeUnknown && rType == typeUnknown {
		return
	}
	if lType == rType {
		return // same type, no coercion needed
	}

	op := n.Op
	switch op {
	case "+":
		// String concatenation: wrap the non-string side in str().
		if lType == typeString && rType != typeString && rType != typeUnknown {
			if !isAlreadyWrapped(n.Right, "str") {
				n.Right = wrapStr(n.Right)
			}
		} else if rType == typeString && lType != typeString && lType != typeUnknown {
			if !isAlreadyWrapped(n.Left, "str") {
				n.Left = wrapStr(n.Left)
			}
		}

	case "-", "*", "/", "%", "**":
		// Arithmetic: wrap the string side in num().
		if lType == typeString && rType == typeNumber {
			if !isAlreadyWrapped(n.Left, "num") {
				n.Left = wrapNum(n.Left)
			}
		} else if rType == typeString && lType == typeNumber {
			if !isAlreadyWrapped(n.Right, "num") {
				n.Right = wrapNum(n.Right)
			}
		} else if lType == typeBool && rType == typeNumber {
			if !isAlreadyWrapped(n.Left, "num") {
				n.Left = wrapNum(n.Left)
			}
		} else if rType == typeBool && lType == typeNumber {
			if !isAlreadyWrapped(n.Right, "num") {
				n.Right = wrapNum(n.Right)
			}
		}
	}
}
