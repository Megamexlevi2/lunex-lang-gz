// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package errfmt

import (
	"fmt"
	"os"
	"strings"
)

var useColor = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"

func c(code, t string) string {
	if !useColor {
		return t
	}
	return "\x1b[" + code + "m" + t + "\x1b[0m"
}

func bold(t string) string   { return c("1", t) }
func dim(t string) string    { return c("2", t) }
func red(t string) string    { return c("31", t) }
func green(t string) string  { return c("32", t) }
func yellow(t string) string { return c("33", t) }
func blue(t string) string   { return c("34", t) }
func cyan(t string) string   { return c("36", t) }
func white(t string) string  { return c("37", t) }
func gray(t string) string   { return c("90", t) }

type ErrorKind string

const (
	KindSyntax    ErrorKind = "SyntaxError"
	KindType      ErrorKind = "TypeError"
	KindReference ErrorKind = "ReferenceError"
	KindRuntime   ErrorKind = "RuntimeError"
	KindImport    ErrorKind = "ImportError"
	KindAssertion ErrorKind = "AssertionError"
	KindRange     ErrorKind = "RangeError"
	KindIO        ErrorKind = "IOError"
	KindLex       ErrorKind = "LexError"
	KindParse     ErrorKind = "ParseError"
)

var phaseLabel = map[ErrorKind]string{
	KindLex:       "error[lex]",
	KindParse:     "error[parse]",
	KindSyntax:    "error[syntax]",
	KindType:      "error[type]",
	KindRuntime:   "error[runtime]",
	KindReference: "error[scope]",
	KindImport:    "error[module]",
	KindIO:        "error[io]",
	KindAssertion: "error[assertion]",
	KindRange:     "error[range]",
}

type StackFrame struct {
	FnName string
	File   string
	Line   int
	Col    int
}

type LunexError struct {
	Message    string
	File       string
	Line       int
	Col        int
	Phase      string
	Kind       ErrorKind
	Suggestion string
	Source     string
	Lines      []string
	Stack      []StackFrame
	Notes      []string
	Similar    []string
	Code       string
	ExBad      string
	ExGood     string
}

func (e *LunexError) Error() string { return e.Message }

func (e *LunexError) WithNote(note string) *LunexError {
	e.Notes = append(e.Notes, note)
	return e
}

func (e *LunexError) WithStack(frames []StackFrame) *LunexError {
	e.Stack = frames
	return e
}

func levenshtein(a, b string) int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				d := dp[i-1][j]
				if dp[i][j-1] < d {
					d = dp[i][j-1]
				}
				if dp[i-1][j-1] < d {
					d = dp[i-1][j-1]
				}
				dp[i][j] = 1 + d
			}
		}
	}
	return dp[m][n]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func FindSimilar(name string, names []string) []string {
	nl := strings.ToLower(name)
	type scored struct {
		n    string
		dist int
	}
	var results []scored
	for _, n := range names {
		if n == name || len(n) <= 1 || strings.HasPrefix(n, "__") {
			continue
		}
		nl2 := strings.ToLower(n)
		dist := levenshtein(nl, nl2)
		maxLen := len(nl)
		if len(nl2) > maxLen {
			maxLen = len(nl2)
		}
		threshold := maxLen / 2
		if threshold < 3 {
			threshold = 3
		}
		shareStart := len(nl) >= 3 && len(nl2) >= 3 &&
			(strings.HasPrefix(nl, nl2[:minInt(4, len(nl2))]) ||
				strings.HasPrefix(nl2, nl[:minInt(4, len(nl))]))
		if dist <= threshold || shareStart {
			results = append(results, scored{n, dist})
		}
	}
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].dist < results[j-1].dist; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	out := make([]string, 0, 3)
	for i, r := range results {
		if i >= 3 {
			break
		}
		out = append(out, r.n)
	}
	return out
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

func buildSourceView(lines []string, line, col int, underlineLabel string) string {
	if len(lines) == 0 || line < 1 || line > len(lines) {
		return ""
	}
	w := len(fmt.Sprintf("%d", line+1))
	if w < 3 {
		w = 3
	}
	numFmt := func(n int, active bool) string {
		s := fmt.Sprintf("%*d", w, n)
		if active {
			return blue(" "+s+" │")
		}
		return gray(" "+s+" │")
	}
	blank := blue(" " + strings.Repeat(" ", w) + " │")

	var rows []string
	if line > 2 {
		rows = append(rows, dim(numFmt(line-2, false)+" "+lines[line-3]))
	}
	if line > 1 {
		rows = append(rows, dim(numFmt(line-1, false)+" "+lines[line-2]))
	}
	rows = append(rows, numFmt(line, true)+" "+white(lines[line-1]))

	safeCol := col - 1
	if safeCol < 0 {
		safeCol = 0
	}
	srcLine := lines[line-1]
	tokenLen := safeCol
	for tokenLen < len(srcLine) && isIdentChar(srcLine[tokenLen]) {
		tokenLen++
	}
	tokenLen = tokenLen - safeCol
	if tokenLen < 3 {
		tokenLen = 3
	}
	caret := red("┬" + strings.Repeat("─", tokenLen-1))
	label := red("╰── ") + yellow(underlineLabel)
	rows = append(rows, blank+" "+strings.Repeat(" ", safeCol)+caret)
	rows = append(rows, blank+" "+strings.Repeat(" ", safeCol)+label)

	if line < len(lines) {
		rows = append(rows, dim(numFmt(line+1, false)+" "+lines[line]))
	}

	return strings.Join(rows, "\n")
}

func getUnderlineLabel(code, msg, name string) string {
	switch code {
	case "E0001", "UNDEF_VAR":
		return "`" + name + "` not found in scope"
	case "E0002":
		return "method not found"
	case "E0003", "NOT_FUNCTION":
		return "not callable"
	case "E0004", "NULL_ACCESS":
		return "null/undefined — cannot read property"
	case "UNDEF_FUNC":
		return "not defined in this scope"
	case "CONST_REASSIGN":
		return "`" + name + "` is immutable (val)"
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "not defined") || strings.Contains(lower, "not declared") {
		return "not declared in this scope"
	}
	if strings.Contains(lower, "is not a function") || strings.Contains(lower, "not callable") {
		return "not a function"
	}
	if strings.Contains(lower, "type") {
		return "type error originates here"
	}
	return "error originates here"
}

func getSuggestions(code, name, msg string, similar []string) []string {
	var sugs []string
	lower := strings.ToLower(msg)

	switch code {
	case "E0001":
		// If the undefined name matches a known stdlib module, suggest importing it.
		knownStdlib := map[string]string{
			"io": "std.io", "fs": "std.fs", "http": "std.http",
			"crypto": "std.crypto", "db": "std.db", "env": "std.env",
			"ws": "std.ws", "mail": "std.mail", "ai": "std.ai",
			"utils": "std.utils", "validate": "std.validate", "os": "std.os",
			"jwt": "std.jwt", "redis": "std.redis", "postgres": "std.postgres",
			"mysql": "std.mysql", "stripe": "std.stripe", "oauth2": "std.oauth2",
			"graphql": "std.graphql", "rabbitmq": "std.rabbitmq", "excel": "std.excel",
			"pdf": "std.pdf", "csv": "std.csv", "yaml": "std.yaml", "toml": "std.toml",
			"markdown": "std.markdown", "mustache": "std.mustache", "xml": "std.xml",
			"alloc": "std.alloc", "zip": "std.zip", "regex": "std.regex",
			"math": "std.math", "datetime": "std.datetime", "path": "std.path",
			"compress": "std.compress", "cache": "std.cache", "logger": "std.logger",
			"queue": "std.queue", "test": "std.test",
		}
		if stdlibName, ok := knownStdlib[name]; ok {
			sugs = append(sugs, "add this import at the top of your file:  val "+name+" = @import(\""+stdlibName+"\")")
			return sugs
		}
		sugs = append(sugs, "declare `"+name+"` before using it, or check for a typo in the name")
		if len(similar) > 0 {
			sugs = append(sugs, "did you mean `"+similar[0]+"`?")
		}
		return sugs
	case "E0002":
		if len(similar) > 0 {
			sugs = append(sugs, "did you mean `"+similar[0]+"`?")
		} else {
			sugs = append(sugs, "check the spelling and verify the module exports this method")
		}
		return sugs
	case "E0003":
		sugs = append(sugs, "only values declared with `fn` are callable")
		sugs = append(sugs, "check the value is not null/undefined before calling it")
		return sugs
	case "E0004":
		sugs = append(sugs, "guard against null before accessing properties: if "+name+" != null { ... }")
		sugs = append(sugs, "use optional chaining: "+name+"?.property")
		return sugs
	case "E0010", "E0011", "E0012", "E0013", "E0014":
		sugs = append(sugs, "for local files: use @import(\"./file.lx\") or @import(\"file.lx\") — the file must be in the same directory")
		sugs = append(sugs, "for stdlib modules: use @import(\"std.io\"), @import(\"std.fs\"), etc.")
		sugs = append(sugs, "for external packages: run `lunex add <module>` to install it")
		return sugs
	case "UNDEF_VAR":
		sugs = append(sugs, "declare it first:  val "+name+" = <value>")
		if len(similar) > 0 {
			quoted := make([]string, minInt(2, len(similar)))
			for i := range quoted {
				quoted[i] = "'" + similar[i] + "'"
			}
			sugs = append(sugs, "did you mean "+strings.Join(quoted, " or ")+"?")
		}
		return sugs
	case "CONST_REASSIGN":
		sugs = append(sugs, "use var instead:  var "+name+" = <value>")
		sugs = append(sugs, "or create a new binding:  val "+name+"2 = newValue")
		return sugs
	case "NOT_FUNCTION":
		sugs = append(sugs, "'"+name+"' is not a function — check what it actually holds")
		sugs = append(sugs, "if it comes from a module, make sure the module exports it as a function")
		return sugs
	case "NULL_ACCESS":
		sugs = append(sugs, "guard first:  if "+name+" != null { ... }")
		sugs = append(sugs, "use optional chaining:  "+name+"?.property")
		sugs = append(sugs, "provide a fallback:  "+name+" ?? defaultValue")
		return sugs
	}

	if strings.Contains(lower, "is not a function") || strings.Contains(lower, "not callable") {
		sugs = append(sugs, "make sure the value is declared as a function with 'fn' before calling it")
	} else if strings.Contains(lower, "is not defined") || strings.Contains(lower, "not declared") {
		sugs = append(sugs, "declare the variable before use, or check for typos in the name")
		if len(similar) > 0 {
			sugs = append(sugs, "did you mean '"+similar[0]+"'?")
		}
	} else if strings.Contains(lower, "cannot reassign") || strings.Contains(lower, "immutable") {
		sugs = append(sugs, "use 'var' instead of 'val' if you need a mutable variable")
	} else if strings.Contains(lower, "unexpected token") {
		sugs = append(sugs, "check for missing brackets, colons, or keywords near this position")
	} else if strings.Contains(lower, "division by zero") {
		sugs = append(sugs, "guard the denominator with an 'if' check before dividing")
	} else if strings.Contains(lower, "cannot resolve module") {
		sugs = append(sugs, "run 'lunex add <module>' to install, or verify the module name in the stdlib list")
	} else if strings.Contains(lower, "stack overflow") || strings.Contains(lower, "recursion") {
		sugs = append(sugs, "add a base case to your recursive function to stop infinite recursion")
	} else if strings.Contains(lower, "index out of range") || strings.Contains(lower, "out of bounds") {
		sugs = append(sugs, "check the array length before accessing an index: 'if i < arr.length'")
	} else if strings.Contains(lower, "null") || strings.Contains(lower, "undefined") {
		sugs = append(sugs, "guard the value:  if x != null { ... }")
		sugs = append(sugs, "use optional chaining:  x?.property")
	}
	return sugs
}

func extractName(msg string) string {
	for _, q := range []byte{'\'', '"'} {
		start := strings.IndexByte(msg, q)
		if start >= 0 {
			end := strings.IndexByte(msg[start+1:], q)
			if end >= 0 {
				return msg[start+1 : start+1+end]
			}
		}
	}
	return ""
}

func Format(err *LunexError) string {
	if err == nil {
		return ""
	}
	label, ok := phaseLabel[err.Kind]
	if !ok {
		label = err.Phase
		if label == "" {
			label = "error"
		}
	}
	// Prepend error code (E0001, E0002, …) when present
	if err.Code != "" && len(err.Code) == 5 && err.Code[0] == 'E' {
		// Insert code into label: "error[scope]" → "error[E0001][scope]"
		bracketIdx := strings.Index(label, "[")
		if bracketIdx >= 0 {
			label = label[:bracketIdx] + "[" + err.Code + "]" + label[bracketIdx:]
		} else {
			label = label + "[" + err.Code + "]"
		}
	}
	msg := err.Message
	// 'name' is the identifier extracted from the message (used for suggestions/underlines)
	// It is NOT the error code (E0001 etc.) which is already embedded in the label above
	name := extractName(msg)
	if name == "" && err.Code != "" && len(err.Code) > 5 {
		name = err.Code // fallback for codes like "UNDEF_VAR"
	}

	sugs := getSuggestions(err.Code, name, msg, err.Similar)
	if err.Suggestion != "" {
		sugs = []string{err.Suggestion}
	}

	var out []string
	out = append(out, "")
	out = append(out, bold(red(label))+bold(": "+msg))

	if err.File != "" || err.Line > 0 {
		parts := []string{}
		if err.File != "" && err.File != "<unknown>" {
			parts = append(parts, err.File)
		}
		if err.Line > 0 {
			parts = append(parts, fmt.Sprintf("%d", err.Line))
		}
		if err.Col > 1 {
			parts = append(parts, fmt.Sprintf("%d", err.Col))
		}
		out = append(out, blue("  --> ")+strings.Join(parts, ":"))
	}

	if len(err.Lines) > 0 && err.Line > 0 {
		out = append(out, "")
		ulLabel := getUnderlineLabel(err.Code, msg, name)
		view := buildSourceView(err.Lines, err.Line, err.Col, ulLabel)
		if view != "" {
			out = append(out, view)
		}
	}

	if len(sugs) > 0 {
		out = append(out, "")
		for i, s := range sugs {
			prefix := green("  help: ")
			if i > 0 {
				prefix = green("     or: ")
			}
			out = append(out, prefix+s)
		}
	}

	if len(err.Similar) > 0 {
		out = append(out, "")
		out = append(out, yellow("  note: ")+"similar names in scope:")
		for _, s := range err.Similar {
			out = append(out, "        "+cyan(s))
		}
	}

	if err.ExBad != "" && err.ExGood != "" {
		out = append(out, "")
		out = append(out, red("  bad:  ")+dim(err.ExBad))
		out = append(out, green("  good: ")+green(err.ExGood))
	}

	if len(err.Stack) > 0 {
		out = append(out, "")
		out = append(out, gray("  stack trace:"))
		for i, frame := range err.Stack {
			if i >= 8 {
				out = append(out, fmt.Sprintf("    %s", gray(fmt.Sprintf("... %d more frames", len(err.Stack)-i))))
				break
			}
			fn := frame.FnName
			if fn == "" {
				fn = "<anonymous>"
			}
			line := "    " + gray("at") + " " + yellow(fn)
			if frame.File != "" {
				line += " " + gray("("+frame.File)
				if frame.Line > 0 {
					line += gray(fmt.Sprintf(":%d", frame.Line))
				}
				line += gray(")")
			}
			out = append(out, line)
		}
	}

	if len(err.Notes) > 0 {
		out = append(out, "")
		for _, note := range err.Notes {
			out = append(out, yellow("  note: ")+note)
		}
	}

	out = append(out, "")
	return strings.Join(out, "\n") + "\n"
}

func Print(err *LunexError) {
	fmt.Fprint(os.Stderr, Format(err))
}

func FormatSimple(msg, file string, line, col int) string {
	return Format(&LunexError{
		Message: msg,
		File:    file,
		Line:    line,
		Col:     col,
		Kind:    KindRuntime,
	})
}

func New(kind ErrorKind, msg, file string, line, col int, lines []string) *LunexError {
	return &LunexError{
		Message: msg,
		File:    file,
		Line:    line,
		Col:     col,
		Kind:    kind,
		Lines:   lines,
	}
}

func SyntaxError(msg, suggestion, file string, line, col int, lines []string) *LunexError {
	return &LunexError{
		Message:    msg,
		File:       file,
		Line:       line,
		Col:        col,
		Kind:       KindSyntax,
		Suggestion: suggestion,
		Lines:      lines,
	}
}

func NewTypeError(msg, suggestion string) *LunexError {
	return &LunexError{
		Message:    msg,
		Kind:       KindType,
		Suggestion: suggestion,
	}
}

func ReferenceError(name, file string, line, col int, lines []string) *LunexError {
	return &LunexError{
		Message:    fmt.Sprintf("'%s' is not defined", name),
		File:       file,
		Line:       line,
		Col:        col,
		Kind:       KindReference,
		Code:       "UNDEF_VAR",
		Suggestion: fmt.Sprintf("declare it with 'val %s = ...' or 'var %s = ...' before use", name, name),
		Lines:      lines,
	}
}

func ReferenceErrorWithSimilar(name, file string, line, col int, lines []string, similar []string) *LunexError {
	return &LunexError{
		Message:    fmt.Sprintf("'%s' is not defined", name),
		File:       file,
		Line:       line,
		Col:        col,
		Kind:       KindReference,
		Code:       "UNDEF_VAR",
		Suggestion: fmt.Sprintf("declare it with 'val %s = ...' or 'var %s = ...' before use", name, name),
		Lines:      lines,
		Similar:    similar,
	}
}

func ImportError(mod, file string, line int) *LunexError {
	return &LunexError{
		Message:    fmt.Sprintf("cannot resolve module '%s'", mod),
		File:       file,
		Line:       line,
		Kind:       KindImport,
		Code:       "UNDEF_MOD",
		Suggestion: fmt.Sprintf("check that '%s' is installed with 'lunex add %s' or is a valid stdlib module", mod, mod),
	}
}

func AssertionError(expr, file string, line int, lines []string) *LunexError {
	return &LunexError{
		Message: fmt.Sprintf("assertion failed: %s", expr),
		File:    file,
		Line:    line,
		Kind:    KindAssertion,
		Lines:   lines,
	}
}

func SuggestForMessage(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "is not a function") || strings.Contains(lower, "not callable"):
		return "make sure the value is declared as a function with 'fn' before calling it"
	case strings.Contains(lower, "is not defined") || strings.Contains(lower, "not declared"):
		return "declare the variable before use, or check for typos in the name"
	case strings.Contains(lower, "cannot reassign") || strings.Contains(lower, "immutable"):
		return "use 'var' instead of 'val' if you need a mutable variable"
	case strings.Contains(lower, "unexpected token"):
		return "check for missing brackets, colons, or keywords near this position"
	case strings.Contains(lower, "expected"):
		return "review the syntax here — a keyword or delimiter may be missing"
	case strings.Contains(lower, "division by zero"):
		return "guard the denominator with an 'if' check before dividing"
	case strings.Contains(lower, "null") || strings.Contains(lower, "undefined"):
		return "use 'if x != null' or optional chaining 'x?.prop' to safely access nullable values"
	case strings.Contains(lower, "cannot resolve module"):
		return "run 'lunex add <module>' to install, or check the module name in the stdlib list"
	case strings.Contains(lower, "stack overflow") || strings.Contains(lower, "recursion"):
		return "add a base case to your recursive function to stop infinite recursion"
	case strings.Contains(lower, "index out of range") || strings.Contains(lower, "out of bounds"):
		return "check the array length before accessing an index: 'if i < arr.length'"
	case strings.Contains(lower, "cannot read"):
		return "the value may be null or undefined — use optional chaining (?.) or a guard check"
	}
	return ""
}
