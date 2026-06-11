// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package compiler

import (
	"fmt"
	"io"
	"lunex/internal/ast"
	"lunex/internal/errfmt"
	"lunex/internal/formatter"
	"lunex/internal/lexer"
	"lunex/internal/meta"
	"lunex/internal/parser"
	"lunex/internal/runtime"
	"lunex/internal/selfhosted"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Strict    bool
	TypeCheck bool  // force type checking even without lunex.types = on
	LowLevel  bool  // force low-level mode even without lunex.lowlevel = on
	Silent    bool
	Profile   bool
	REPL      bool
}

var DefaultOptions = Options{
	Silent:  true,
	Profile: false,
}

type CompileResult struct {
	AST      *ast.Node
	Errors   []*errfmt.LunexError
	Warnings []*errfmt.LunexError
	Time     time.Duration
	Success  bool
}

type Compiler struct {
	opts        Options
	interp      *runtime.Interpreter
	moduleCache map[string]*runtime.Value
}

func New(opts Options) *Compiler {
	c := &Compiler{
		opts:        opts,
		interp:      runtime.NewInterpreter(),
		moduleCache: make(map[string]*runtime.Value),
	}
	c.registerStdlib()
	return c
}

func (c *Compiler) Interpreter() *runtime.Interpreter {
	return c.interp
}

func (c *Compiler) CompileSource(source, filename string) *CompileResult {
	start := time.Now()
	lines := strings.Split(source, "\n")
	result := &CompileResult{}

	// Parse top-level directives (lunex.types = on, lunex.lowlevel = on)
	// before tokenization so they affect the compile pipeline immediately.
	flags := selfhosted.ParseFileFlags(source)
	typesEnabled    := flags.TypesEnabled    || c.opts.TypeCheck
	_ = flags.LowLevelEnabled || c.opts.LowLevel // reserved for future low-level checks

	tokens, err := lexer.Tokenize(source, filename)
	if err != nil {
		hint := errfmt.SuggestForMessage(err.Error())
		result.Errors = append(result.Errors, &errfmt.LunexError{
			Message:    err.Error(),
			File:       filename,
			Kind:       errfmt.KindLex,
			Suggestion: hint,
			Lines:      lines,
		})
		result.Time = time.Since(start)
		return result
	}

	tree, err := parser.ParseWithLines(tokens, filename, lines)
	if err != nil {
		if pe, ok := err.(*errfmt.LunexError); ok {
			if pe.File == "" {
				pe.File = filename
			}
			if len(pe.Lines) == 0 {
				pe.Lines = lines
			}
			result.Errors = append(result.Errors, pe)
		} else {
			hint := errfmt.SuggestForMessage(err.Error())
			result.Errors = append(result.Errors, &errfmt.LunexError{
				Message:    err.Error(),
				File:       filename,
				Kind:       errfmt.KindParse,
				Suggestion: hint,
				Lines:      lines,
			})
		}
		result.Time = time.Since(start)
		return result
	}

	// ── Self-hosted type checker ────────────────────────────────────────────
	// Runs only when the source opts in with "lunex.types = on" or when the
	// compiler was started with --strict / TypeCheck option.
	// Type errors are treated as compile errors; they abort before codegen.
	if typesEnabled {
		astMap := astToMap(tree)
		typeErrs, tcErr := selfhosted.CheckTypes(astMap, filename)
		if tcErr == nil && len(typeErrs) > 0 {
			for _, te := range typeErrs {
				result.Errors = append(result.Errors, &errfmt.LunexError{
					Message:    te.Message,
					File:       te.File,
					Line:       te.Line,
					Col:        te.Col,
					Kind:       errfmt.KindType,
					Suggestion: errfmt.CodeSuggestion(te.Code),
					Lines:      lines,
				})
			}
			result.Time = time.Since(start)
			return result
		}
	}

	result.AST = tree
	result.Success = true
	result.Time = time.Since(start)
	return result
}

// astToMap converts an *ast.Node to a plain map[string]interface{} that the
// selfhost type checker can consume without depending on Go types.
func astToMap(n *ast.Node) map[string]interface{} {
	if n == nil {
		return nil
	}
	m := map[string]interface{}{
		"type": string(n.Type),
		"line": n.Line,
		"col":  n.Col,
		"name": n.Name,
	}
	if n.TypeAnn != nil {
		m["typeAnn"] = fmt.Sprintf("%v", n.TypeAnn)
	}
	if n.Op != "" {
		m["op"] = n.Op
	}
	if n.Body != nil {
		m["body"] = astToMap(n.Body)
	}
	if n.Init != nil {
		m["init"] = astToMap(n.Init)
	}
	if n.Left != nil {
		m["left"] = astToMap(n.Left)
	}
	if n.Right != nil {
		m["right"] = astToMap(n.Right)
	}
	if n.Test != nil {
		m["test"] = astToMap(n.Test)
	}
	if n.Alternate != nil {
		m["alternate"] = astToMap(n.Alternate)
	}
	if n.Consequent != nil {
		m["consequent"] = astToMap(n.Consequent)
	}
	if len(n.Body_) > 0 {
		children := make([]interface{}, len(n.Body_))
		for i, child := range n.Body_ {
			children[i] = astToMap(child)
		}
		m["body_"] = children
	}
	if len(n.Args) > 0 {
		args := make([]interface{}, len(n.Args))
		for i, arg := range n.Args {
			args[i] = astToMap(arg)
		}
		m["args"] = args
	}
	if len(n.Params) > 0 {
		params := make([]interface{}, len(n.Params))
		for i, p := range n.Params {
			pm := map[string]interface{}{"name": p.Name}
			if p.TypeAnn != nil {
				pm["typeAnn"] = fmt.Sprintf("%v", p.TypeAnn)
			}
			params[i] = pm
		}
		m["params"] = params
	}
	if n.Value != nil {
		switch v := n.Value.(type) {
		case string:
			m["value"] = v
		case float64:
			m["value"] = v
		case bool:
			m["value"] = v
		}
	}
	return m
}

func (c *Compiler) RunFile(filePath string) error {
	source, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprint(os.Stderr, errfmt.Format(&errfmt.LunexError{
			Message:    fmt.Sprintf("cannot read file '%s': %v", filePath, err),
			File:       filePath,
			Kind:       errfmt.KindIO,
			Suggestion: "check that the file exists and you have read permissions",
		}))
		return err
	}
	return c.RunSource(string(source), filePath)
}

func (c *Compiler) RunSource(source, filename string) error {
	result := c.CompileSource(source, filename)
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		return fmt.Errorf("compilation failed with %d error(s)", len(result.Errors))
	}
	return c.RunAST(result.AST, filename, source)
}

// RunSourceAsModule compiles and executes a Lunex source string as a module,
// returning the exported Value object. Used by the NAX loader so binary archives
// can be imported via @fimport without running main() or blocking on KeepAliveWait.
func (c *Compiler) RunSourceAsModule(source, filename string) (*runtime.Value, error) {
	return c.interp.ExecAsModule(source, filename)
}

func (c *Compiler) RunAST(tree *ast.Node, filename, source string) error {
	c.interp.SetFilename(filename)
	c.interp.SetSourceLines(strings.Split(source, "\n"))
	_ = filepath.Dir(filename)

	// Emit style warnings (spacing, semicolons, braces) — deduplicated by code.
	emitStyleHintIfNeeded(source, filename, os.Stderr)

	// Emit W0005 warnings for any implicit type coercions found in the AST.
	// These are the same issues that `lunex fmt` fixes automatically.
	emitCoercionWarnings(tree, filename, source, os.Stderr)

	_, execErr := c.interp.Exec(tree)
	if execErr != nil {
		c.printRuntimeError(execErr, filename, source)
		return execErr
	}
	if mainErr := c.interp.CallMain(); mainErr != nil {
		c.printRuntimeError(mainErr, filename, source)
		return mainErr
	}
	// Block until all long-running background tasks (HTTP server, WebSocket server,
	// RabbitMQ consumers, Redis subscribers, etc.) have finished.
	// Without this, the process exits immediately after main() returns,
	// killing all goroutines that were still serving requests.
	runtime.KeepAliveWait()
	return nil
}

func (c *Compiler) printRuntimeError(err error, filename, source string) {
	srcLines := strings.Split(source, "\n")
	if ntlErr, ok := err.(*errfmt.LunexError); ok {
		if len(ntlErr.Lines) == 0 {
			ntlErr.Lines = srcLines
		}
		if ntlErr.File == "" {
			ntlErr.File = filename
		}
		fmt.Fprint(os.Stderr, errfmt.Format(ntlErr))
		return
	}

	msg := err.Error()
	hint := errfmt.SuggestForMessage(msg)

	kind := errfmt.KindRuntime
	if strings.HasPrefix(msg, "TypeError:") {
		kind = errfmt.KindType
		msg = strings.TrimPrefix(msg, "TypeError: ")
	} else if strings.HasPrefix(msg, "ReferenceError:") {
		kind = errfmt.KindReference
		msg = strings.TrimPrefix(msg, "ReferenceError: ")
	} else if strings.HasPrefix(msg, "ImportError:") {
		kind = errfmt.KindImport
		msg = strings.TrimPrefix(msg, "ImportError: ")
	}

	fmt.Fprint(os.Stderr, errfmt.Format(&errfmt.LunexError{
		Message:    msg,
		File:       filename,
		Kind:       kind,
		Lines:      srcLines,
		Suggestion: hint,
	}))
}

func Version() string {
	return meta.Version()
}

func (c *Compiler) Version() string {
	return Version()
}

func (c *Compiler) Check(source, filename string) error {
	result := c.CompileSource(source, filename)
	if !result.Success {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	return nil
}

// Format formats Lunex source by parsing it to an AST and pretty-printing.
// It also applies the coercion fixer pass (FixCoercions) which rewrites
// implicit type mismatches such as `10 + "10"` → `str(10) + "10"`.
// Falls back to the legacy indent-only formatter if parsing fails.
func Format(source string) string {
	tokens, err := lexer.Tokenize(source, "<fmt>")
	if err == nil {
		tree, err2 := parser.Parse(tokens, "<fmt>")
		if err2 == nil {
			// Apply coercion fixes before pretty-printing.
			formatter.FixCoercions(tree)
			return formatter.Format(tree)
		}
	}
	// Fallback: indent-only pass (safe for unparseable files).
	return formatLegacy(source)
}

// formatLegacy is the old text-based indent formatter used as a fallback.
func formatLegacy(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	indent := 0

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			out = append(out, "")
			continue
		}

		opens, closes := countBracesOutsideStrings(line)

		if closes > 0 && (strings.HasPrefix(line, "}") || strings.HasPrefix(line, ")") || strings.HasPrefix(line, "]")) {
			indent -= closes
			if indent < 0 {
				indent = 0
			}
		}

		out = append(out, strings.Repeat("  ", indent)+line)

		if opens > closes {
			indent += opens - closes
		}
	}
	return strings.Join(out, "\n")
}

// emitStyleHintIfNeeded emits all style warnings for the source, but
// deduplicates by warning code — each W-code appears only once per file.
// After all warnings, if any were found, a single action hint is appended
// pointing the user to `lunex fmt`.
func emitStyleHintIfNeeded(source, filename string, w io.Writer) {
	lines := strings.Split(source, "\n")
	var warns []lintWarning
	seen := map[string]bool{}

	collect := func(warn lintWarning) {
		if seen[warn.code] {
			return
		}
		seen[warn.code] = true
		warns = append(warns, warn)
	}

	// ── W0010: entire program on a single line ────────────────────────────
	nonEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	totalStmts := countStatementTokens(source)
	if nonEmpty <= 1 && totalStmts >= 2 {
		collect(lintWarning{
			code: "W0010", line: 1, col: 1,
			msg:        fmt.Sprintf("entire program written on a single line (%d statements)", totalStmts),
			suggestion: "each statement should be on its own line for readability and accurate error locations",
			exBad:      `val io=@import("std.io");fn main(){io.log("hi")}`,
			exGood:     "val io = @import(\"std.io\")\n\nfn main() {\n  io.log(\"hi\")\n}",
		})
	}

	for i, raw := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// ── W0009: semicolons as statement separators ─────────────────────
		if col := findSemicolonSeparator(trimmed); col >= 0 {
			collect(lintWarning{
				code: "W0009", line: lineNum, col: col + 1,
				msg:        "semicolon used as statement separator",
				suggestion: "replace `;` with a newline — Lunex uses line breaks, not semicolons",
				exBad:      "val x = 1; val y = 2; io.log(x)",
				exGood:     "val x = 1\nval y = 2\nio.log(x)",
			})
		}

		// ── W0006: multiple statements on one line ────────────────────────
		if stmts := countStatementTokens(trimmed); stmts >= 2 {
			collect(lintWarning{
				code: "W0006", line: lineNum, col: 1,
				msg:        fmt.Sprintf("multiple statements on one line (%d statements detected)", stmts),
				suggestion: "put each statement on its own line",
				exBad:      "fn soma(a,b){a+b}fn main(){io.log(soma(1,2))}",
				exGood:     "fn soma(a, b) {\n  a + b\n}\n\nfn main() {\n  io.log(soma(1, 2))\n}",
			})
		}

		// ── W0007: missing spaces around operators / after commas ─────────
		if issues := findSpacingIssues(trimmed); len(issues) > 0 {
			issue := issues[0]
			collect(lintWarning{
				code: "W0007", line: lineNum, col: issue.col,
				msg:        issue.msg,
				suggestion: issue.fix,
				exBad:      issue.bad,
				exGood:     issue.good,
			})
		}

		// ── W0008: missing space before `{` ───────────────────────────────
		if col := findMissingSpaceBeforeBrace(trimmed); col >= 0 {
			collect(lintWarning{
				code: "W0008", line: lineNum, col: col + 1,
				msg:        "missing space before `{`",
				suggestion: "add a space before `{`:  fn name() { ... }",
				exBad:      "fn main(){",
				exGood:     "fn main() {",
			})
		}
	}

	if len(warns) == 0 {
		return
	}

	// Emit one warning per distinct code.
	for _, warn := range warns {
		e := &errfmt.LunexError{
			Message:    warn.msg,
			File:       filename,
			Kind:       errfmt.KindStyle,
			Code:       warn.code,
			Lines:      lines,
			Line:       warn.line,
			Col:        warn.col,
			Suggestion: warn.suggestion,
			ExBad:      warn.exBad,
			ExGood:     warn.exGood,
		}
		fmt.Fprint(w, errfmt.Format(e))
	}

	// Final action hint — printed once after all warnings.
	fmt.Fprintf(w, "\n  \033[1;36m→\033[0m  run \033[1mlunex fmt %s\033[0m to auto-fix all style issues above\n\n", filename)
}

// coercionWarning describes an implicit type coercion found in the AST.
type coercionWarning struct {
	line    int
	col     int
	leftT   string
	rightT  string
	op      string
	fixHint string
}

// emitCoercionWarnings walks the AST and emits W0005 warnings for every
// BinaryExpr where the two operands are statically known to be incompatible.
// Only the first occurrence of each coercion pattern is reported per file
// (to avoid noise), because `lunex fmt` will fix them all at once.
func emitCoercionWarnings(tree *ast.Node, filename, source string, w io.Writer) {
	lines := strings.Split(source, "\n")
	seen := map[string]bool{}
	var found []coercionWarning
	collectCoercions(tree, &found, seen)
	if len(found) == 0 {
		return
	}
	for _, cw := range found {
		msg := fmt.Sprintf(
			"implicit type coercion: `%s %s %s` — operands have incompatible types",
			cw.leftT, cw.op, cw.rightT,
		)
		e := &errfmt.LunexError{
			Message:    msg,
			File:       filename,
			Kind:       errfmt.KindStyle,
			Code:       "W0005",
			Lines:      lines,
			Line:       cw.line,
			Col:        cw.col,
			Suggestion: cw.fixHint,
			ExBad:      fmt.Sprintf("val r = 10 %s \"10\"", cw.op),
			ExGood:     fmt.Sprintf("val r = str(10) %s \"10\"", cw.op),
		}
		fmt.Fprint(w, errfmt.Format(e))
	}
	fmt.Fprintf(w, "\n  \033[1;36m→\033[0m  run \033[1mlunex fmt %s\033[0m to auto-insert str()/num() conversions\n\n", filename)
}

// staticTypeOf returns a display name for the inferred type of n, or "" if unknown.
func staticTypeOf(n *ast.Node) string {
	if n == nil {
		return ""
	}
	switch n.Type {
	case ast.NumberLit:
		return "number"
	case ast.StringLit, ast.TemplateLit:
		return "string"
	case ast.BoolLit:
		return "bool"
	case ast.CallExpr:
		if n.Callee != nil && n.Callee.Type == ast.Identifier {
			switch n.Callee.Name {
			case "str", "String":
				return "string"
			case "num", "Number", "parseInt", "parseFloat":
				return "number"
			case "bool", "Boolean":
				return "bool"
			}
		}
	}
	return ""
}

// collectCoercions walks node n and appends any coercion warnings to out.
func collectCoercions(n *ast.Node, out *[]coercionWarning, seen map[string]bool) {
	if n == nil {
		return
	}
	if n.Type == ast.BinaryExpr {
		lT := staticTypeOf(n.Left)
		rT := staticTypeOf(n.Right)
		if lT != "" && rT != "" && lT != rT {
			key := lT + n.Op + rT
			if !seen[key] {
				seen[key] = true
				fix := ""
				switch n.Op {
				case "+":
					if lT == "number" {
						fix = fmt.Sprintf("wrap the number: str(%v) + ...", nodeDisplayValue(n.Left))
					} else if rT == "number" {
						fix = fmt.Sprintf("wrap the number: ... + str(%v)", nodeDisplayValue(n.Right))
					} else if lT == "bool" {
						fix = "wrap the bool: str(boolValue) + ..."
					} else if rT == "bool" {
						fix = "wrap the bool: ... + str(boolValue)"
					}
				case "-", "*", "/", "%":
					if lT == "string" {
						fix = "wrap the string: num(stringValue) " + n.Op + " ..."
					} else if rT == "string" {
						fix = "wrap the string: ... " + n.Op + " num(stringValue)"
					}
				}
				if fix == "" {
					fix = "use str() or num() to make the conversion explicit"
				}
				*out = append(*out, coercionWarning{
					line:    n.Line,
					col:     n.Col,
					leftT:   lT,
					rightT:  rT,
					op:      n.Op,
					fixHint: fix,
				})
			}
		}
	}

	// Recurse into all child nodes.
	collectCoercions(n.Left, out, seen)
	collectCoercions(n.Right, out, seen)
	collectCoercions(n.Body, out, seen)
	collectCoercions(n.Init, out, seen)
	collectCoercions(n.Test, out, seen)
	collectCoercions(n.Consequent, out, seen)
	collectCoercions(n.Alternate, out, seen)
	collectCoercions(n.Arg, out, seen)
	collectCoercions(n.Expr, out, seen)
	collectCoercions(n.Stmt, out, seen)
	collectCoercions(n.Object, out, seen)
	collectCoercions(n.Callee, out, seen)
	collectCoercions(n.Subject, out, seen)
	collectCoercions(n.Lo, out, seen)
	collectCoercions(n.Hi, out, seen)
	collectCoercions(n.Declaration, out, seen)
	collectCoercions(n.Extends, out, seen)
	collectCoercions(n.CatchBlock, out, seen)
	collectCoercions(n.FinallyBlock, out, seen)
	for _, c := range n.Args {
		collectCoercions(c, out, seen)
	}
	for _, c := range n.Elements {
		collectCoercions(c, out, seen)
	}
	for _, c := range n.Body_ {
		collectCoercions(c, out, seen)
	}
	for _, c := range n.Exprs {
		collectCoercions(c, out, seen)
	}
	for _, p := range n.Properties {
		collectCoercions(p.Value, out, seen)
	}
	for _, m := range n.Methods {
		collectCoercions(m.Body, out, seen)
		collectCoercions(m.Init, out, seen)
	}
	for _, c := range n.Cases {
		collectCoercions(c.Body, out, seen)
		collectCoercions(c.Guard, out, seen)
	}
}

// nodeDisplayValue returns a short string representation of a literal node value.
func nodeDisplayValue(n *ast.Node) string {
	if n == nil {
		return "?"
	}
	if n.Value != nil {
		return fmt.Sprintf("%v", n.Value)
	}
	if n.Name != "" {
		return n.Name
	}
	return "..."
}

func countBracesOutsideStrings(line string) (opens, closes int) {
	inStr := false
	var strChar rune
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inStr {
			if r == '\\' {
				i++
				continue
			}
			if r == strChar {
				inStr = false
			}
			continue
		}
		if r == '"' || r == '\'' || r == '`' {
			inStr = true
			strChar = r
			continue
		}
		if r == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			break
		}
		switch r {
		case '{', '(':
			opens++
		case '}', ')':
			closes++
		}
	}
	return opens, closes
}

func (c *Compiler) registerStdlib() {}

// lintWarning is a single readability issue found in a source file.
type lintWarning struct {
	code       string
	line       int
	col        int
	msg        string
	suggestion string
	exBad      string
	exGood     string
}


// spacingIssue describes a single spacing problem found on a line.
type spacingIssue struct {
	col  int
	msg  string
	fix  string
	bad  string
	good string
}

// findSpacingIssues checks for missing spaces after commas and around `=`.
// It skips content inside string literals.
func findSpacingIssues(line string) []spacingIssue {
	var issues []spacingIssue
	runes := []rune(line)
	n := len(runes)
	inStr := false
	var strChar rune

	for i := 0; i < n; i++ {
		r := runes[i]
		if inStr {
			if r == '\\' {
				i++
				continue
			}
			if r == strChar {
				inStr = false
			}
			continue
		}
		if r == '"' || r == '\'' || r == '`' {
			inStr = true
			strChar = r
			continue
		}
		if r == '/' && i+1 < n && runes[i+1] == '/' {
			break
		}

		// Missing space after comma: "a,b" → "a, b"
		if r == ',' && i+1 < n && runes[i+1] != ' ' && runes[i+1] != '\n' {
			issues = append(issues, spacingIssue{
				col:  i + 1,
				msg:  "missing space after `,`",
				fix:  "add a space after each comma:  fn soma(a, b)",
				bad:  "fn soma(a,b) { ... }",
				good: "fn soma(a, b) { ... }",
			})
		}

		// Missing space around `=` in assignments: "val x=1" → "val x = 1"
		// Skip `==`, `!=`, `<=`, `>=`, `=>`, `@import=` patterns
		if r == '=' {
			prev := rune(0)
			if i > 0 {
				prev = runes[i-1]
			}
			next := rune(0)
			if i+1 < n {
				next = runes[i+1]
			}
			isComparison := next == '=' || prev == '!' || prev == '<' || prev == '>' || prev == '='
			isArrow := next == '>'
			if !isComparison && !isArrow {
				missingBefore := prev != ' ' && prev != 0
				missingAfter := next != ' ' && next != 0 && next != '\n'
				if missingBefore || missingAfter {
					issues = append(issues, spacingIssue{
						col:  i + 1,
						msg:  "missing space around `=`",
						fix:  "add spaces around `=`:  val x = value",
						bad:  "val x=1",
						good: "val x = 1",
					})
				}
			}
		}
	}
	return issues
}

// findSemicolonSeparator returns the column (0-based) of the first semicolon
// used as a statement separator, or -1 if none found.
// Semicolons inside strings are ignored. A semicolon at the very end of a
// line (trailing) is also reported.
func findSemicolonSeparator(line string) int {
	runes := []rune(line)
	n := len(runes)
	inStr := false
	var strChar rune
	for i := 0; i < n; i++ {
		r := runes[i]
		if inStr {
			if r == '\\' {
				i++
				continue
			}
			if r == strChar {
				inStr = false
			}
			continue
		}
		if r == '"' || r == '\'' || r == '`' {
			inStr = true
			strChar = r
			continue
		}
		if r == '/' && i+1 < n && runes[i+1] == '/' {
			break
		}
		if r == ';' {
			return i
		}
	}
	return -1
}

// findMissingSpaceBeforeBrace returns the column (0-based) of a `{` that is
// not preceded by a space, or -1 if all braces are properly spaced.
func findMissingSpaceBeforeBrace(line string) int {
	runes := []rune(line)
	n := len(runes)
	inStr := false
	var strChar rune
	for i := 0; i < n; i++ {
		r := runes[i]
		if inStr {
			if r == '\\' {
				i++
				continue
			}
			if r == strChar {
				inStr = false
			}
			continue
		}
		if r == '"' || r == '\'' || r == '`' {
			inStr = true
			strChar = r
			continue
		}
		if r == '{' && i > 0 && runes[i-1] != ' ' && runes[i-1] != '\t' {
			return i
		}
	}
	return -1
}

// countStatementTokens counts how many statement-starting keywords
// appear in src outside of string literals and comments.
func countStatementTokens(src string) int {
	keywords := []string{"fn ", "val ", "var ", "if ", "for ", "while ", "return "}
	runes := []rune(src)
	n := len(runes)
	count := 0
	inStr := false
	var strChar rune
	for i := 0; i < n; i++ {
		r := runes[i]
		if inStr {
			if r == '\\' {
				i++
				continue
			}
			if r == strChar {
				inStr = false
			}
			continue
		}
		if r == '"' || r == '\'' || r == '`' {
			inStr = true
			strChar = r
			continue
		}
		if r == '/' && i+1 < n && runes[i+1] == '/' {
			// skip to end of line
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}
		sub := string(runes[i:])
		for _, kw := range keywords {
			if strings.HasPrefix(sub, kw) {
				count++
				break
			}
		}
	}
	return count
}
