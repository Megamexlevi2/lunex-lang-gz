// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package compiler

import (
	"fmt"
	"lunex/internal/ast"
	"lunex/internal/errfmt"
	"lunex/internal/formatter"
	"lunex/internal/lexer"
	"lunex/internal/meta"
	"lunex/internal/parser"
	"lunex/internal/runtime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Strict    bool
	TypeCheck bool // force type checking even without lunex.types = on
	LowLevel  bool // force low-level mode even without lunex.lowlevel = on
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

// fileFlags holds compile-time directives parsed from top-level source comments.
type fileFlags struct {
	typesEnabled    bool
	lowLevelEnabled bool
}

// parseFileFlags scans the leading directive lines of source without running the full parser.
func parseFileFlags(source string) fileFlags {
	var f fileFlags
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "lunex.types = on" || trimmed == "lunex.types=on" {
			f.typesEnabled = true
		}
		if trimmed == "lunex.lowlevel = on" || trimmed == "lunex.lowlevel=on" {
			f.lowLevelEnabled = true
		}
		if len(trimmed) > 0 && trimmed[0] != '/' && !strings.HasPrefix(trimmed, "lunex.") {
			break
		}
	}
	return f
}

func (c *Compiler) CompileSource(source, filename string) *CompileResult {
	start := time.Now()
	lines := strings.Split(source, "\n")
	result := &CompileResult{}

	// Parse top-level directives (lunex.types = on, lunex.lowlevel = on)
	// before tokenization so they affect the compile pipeline immediately.
	flags := parseFileFlags(source)
	typesEnabled := flags.typesEnabled || c.opts.TypeCheck
	_ = flags.lowLevelEnabled || c.opts.LowLevel // reserved for future low-level checks

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

	// Type checking is reserved for a future Go-native implementation.
	// lunex.types = on is accepted as a directive but currently a no-op.
	_ = typesEnabled

	result.AST = tree
	result.Success = true
	result.Time = time.Since(start)
	return result
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
	if lunexErr, ok := err.(*errfmt.LunexError); ok {
		if len(lunexErr.Lines) == 0 {
			lunexErr.Lines = srcLines
		}
		if lunexErr.File == "" {
			lunexErr.File = filename
		}
		fmt.Fprint(os.Stderr, errfmt.Format(lunexErr))
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
