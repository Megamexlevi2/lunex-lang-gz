// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.
// interpreter.go — tree-walking interpreter and bytecode executor for Lunex.
//
// Architecture:
//   The Go tree-walker handles all Lunex execution.
//   Pure-Go fast paths (O(1) math) accelerate recognised numeric loop patterns
//   on all platforms with no external dependencies.

package runtime

import (
	"fmt"
	"lunex/internal/ast"
	"lunex/internal/errfmt"
	"lunex/internal/jit"
	"lunex/internal/lexer"
	"lunex/internal/meta"
	"lunex/internal/parser"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type breakSignal struct{}
type continueSignal struct{}
type returnSignal struct{ value *Value }
type throwSignal struct{ value *Value }

type NTLLoader func(name string) (string, bool)

// NaxLoader is a callback that loads a local .nax or .nc binary file as a module Value.
// It is set by the bytecode layer (vm.go) to avoid a circular import dependency.
// Returns the exported module Value or an error.
type NaxLoader func(path string) (*Value, error)

// deferEntry pairs a deferred statement with the environment it was registered in,
// so each deferred call executes in the correct lexical scope.
type deferEntry struct {
	node *ast.Node
	env  *Environment
}

type Interpreter struct {
	globals       *Environment
	topEnv        *Environment
	profiler      *jit.Profiler
	modules       map[string]*Value
	mu            sync.RWMutex
	filename      string
	sourceLines   []string
	currentLine   int
	currentCol    int
	defers        []deferEntry
	ntlLoader     NTLLoader
	naxLoader     NaxLoader
	libLoadDepth  int32
	callDepth     int
	templateCache sync.Map
	numCache      sync.Map
}

func NewInterpreter() *Interpreter {
	interp := &Interpreter{
		globals:  NewEnvironment(nil),
		profiler: jit.NewProfiler(true),
		modules:  make(map[string]*Value),
	}
	interp.registerBuiltins()
	CallFunction = func(fn *Value, args []*Value, this ...*Value) (*Value, error) {
		var t *Value
		if len(this) > 0 {
			t = this[0]
		}
		return interp.callFunctionValue(fn, args, t)
	}
	return interp
}

func (interp *Interpreter) SetFilename(f string)          { interp.filename = f }
func (interp *Interpreter) SetSourceLines(lines []string) { interp.sourceLines = lines }
func (interp *Interpreter) getSourceLines() []string      { return interp.sourceLines }

// runtimeError builds a rich LunexError from a node with position info.
func (interp *Interpreter) runtimeError(kind errfmt.ErrorKind, code, msg string, node *ast.Node, similar []string) *errfmt.LunexError {
	line, col := 0, 0
	if node != nil {
		line = node.Line
		col = node.Col
		if line == 0 {
			line = interp.currentLine
			col = interp.currentCol
		}
	} else {
		line = interp.currentLine
		col = interp.currentCol
	}
	return &errfmt.LunexError{
		Message: msg,
		File:    interp.filename,
		Line:    line,
		Col:     col,
		Kind:    kind,
		Code:    code,
		Lines:   interp.sourceLines,
		Similar: similar,
	}
}

// visibleNames collects all names visible in the given environment.
func visibleNames(env *Environment) []string {
	seen := make(map[string]bool)
	var names []string
	for e := env; e != nil; e = e.parent {
		for k := range e.vars {
			if !seen[k] && !strings.HasPrefix(k, "__") {
				seen[k] = true
				names = append(names, k)
			}
		}
	}
	return names
}

// objKeys returns the keys of an object Value.
func objKeys(v *Value) []string {
	if v == nil || v.ObjVal == nil {
		return nil
	}
	keys := make([]string, 0, len(v.ObjVal))
	for k := range v.ObjVal {
		keys = append(keys, k)
	}
	return keys
}

func (interp *Interpreter) RegisterModule(name string, val *Value) {
	interp.mu.Lock()
	interp.modules[name] = val
	interp.mu.Unlock()
}

func (interp *Interpreter) RegisterNative(name string, val *Value) {
	interp.globals.Define("__native_"+name+"__", val, true)
}

func (interp *Interpreter) GetModule(name string) (*Value, bool) {
	interp.mu.RLock()
	v, ok := interp.modules[name]
	interp.mu.RUnlock()
	return v, ok
}

// SetGlobal writes a value into the interpreter's global scope.
// Supports dot-path notation (e.g. "io.log") to set a property on a nested
// object that already exists at the top level.
func (interp *Interpreter) SetGlobal(name string, val *Value) {
	dot := strings.IndexByte(name, '.')
	if dot >= 0 {
		parent := name[:dot]
		child := name[dot+1:]
		if obj, ok := interp.globals.Get(parent); ok && obj != nil && obj.Tag == TypeObject {
			if obj.ObjVal == nil {
				obj.ObjVal = make(map[string]*Value)
			}
			obj.ObjVal[child] = val
			return
		}
	}
	interp.globals.Define(name, val, false)
}

// GetGlobal reads a value from the interpreter's global scope.
// Supports dot-path notation (e.g. "io.log") to reach nested object properties.
// Returns (Undefined, false) when the name does not exist.
func (interp *Interpreter) GetGlobal(name string) (*Value, bool) {
	dot := strings.IndexByte(name, '.')
	if dot >= 0 {
		parent := name[:dot]
		child := name[dot+1:]
		if obj, ok := interp.globals.Get(parent); ok && obj != nil && obj.Tag == TypeObject {
			if v, exists := obj.ObjVal[child]; exists {
				return v, true
			}
			return Undefined, false
		}
		return Undefined, false
	}
	return interp.globals.Get(name)
}

// GetAllGlobalNames returns the names of every binding defined in the
// interpreter's global environment (excludes internal __ names).
func (interp *Interpreter) GetAllGlobalNames() []string {
	return interp.globals.AllNames()
}

func (interp *Interpreter) SetNTLLoader(loader NTLLoader) {
	interp.ntlLoader = loader
}

// NTLLoader returns the current source loader function (for chaining in wrappers).
func (interp *Interpreter) NTLLoader() NTLLoader {
	return interp.ntlLoader
}

// SetNaxLoader registers the callback used to load .nax/.nc binary archives as modules.
// Called by the bytecode layer so it can inject its decoder without a circular import.
func (interp *Interpreter) SetNaxLoader(loader NaxLoader) {
	interp.naxLoader = loader
}

func (interp *Interpreter) SetPkgLoader(loader NTLLoader) {
	prev := interp.ntlLoader
	interp.ntlLoader = func(name string) (string, bool) {
		if prev != nil {
			if src, ok := prev(name); ok {
				return src, true
			}
		}
		return loader(name)
	}
}

func (interp *Interpreter) Exec(program *ast.Node) (*Value, error) {
	env := NewEnvironment(interp.globals)
	interp.topEnv = env
	// Hoist class declarations so forward references resolve.
	for _, stmt := range program.Body_ {
		if stmt != nil && stmt.Type == ast.ClassDecl && stmt.Name != "" {
			if _, already := env.Get(stmt.Name); !already {
				stub := &Class{
					Name:          stmt.Name,
					Methods:       make(map[string]*Function),
					StaticMethods: make(map[string]*Function),
					Env:           env,
				}
				env.Define(stmt.Name, ClassVal(stub), false)
			}
		}
	}

	// Validate top-level statements before execution.
	// Only declarations and imports are allowed at the top level.
	// Bare calls, expressions, and explicit main() calls are rejected.
	for _, stmt := range program.Body_ {
		if stmt == nil {
			continue
		}
		if err := interp.checkTopLevelStatement(stmt); err != nil {
			return nil, err
		}
	}

	return interp.execBlock(program.Body_, env)
}

// checkTopLevelStatement enforces top-level restrictions.
// Allowed at the top level: fn, val, var, class, enum, namespace, import/fimport.
// Forbidden: bare function calls, bare expressions, explicit main() calls.
func (interp *Interpreter) checkTopLevelStatement(stmt *ast.Node) *errfmt.LunexError {
	switch stmt.Type {
	case ast.FnDecl, ast.ClassDecl, ast.EnumDecl, ast.NamespaceDecl,
		ast.ComponentDecl, ast.ExportDecl, ast.ImportDecl,
		ast.LunexRequire, ast.UseStmt, ast.ImmutableDecl, ast.UsingDecl:
		// Always allowed.
		return nil

	case ast.VarDecl:
		// val/var allowed — @import and @fimport assignments live here.
		return nil

	case ast.ExprStmt:
		// Bare expression statement at the top level — check what it is.
		expr := stmt.Expr
		if expr == nil {
			return nil
		}
		return interp.checkTopLevelExpr(expr, stmt)

	default:
		// Any other statement (if, while, for, return, etc.) at the top level.
		e := &errfmt.LunexError{
			Message:    fmt.Sprintf("statement of type `%s` is not allowed at the top level", stmt.Type),
			File:       interp.filename,
			Kind:       errfmt.KindSyntax,
			Code:       "E0071",
			Line:       stmt.Line,
			Col:        stmt.Col,
			Lines:      interp.sourceLines,
			Notes:      []string{"only declarations (`fn`, `val`, `var`, `class`) and imports are allowed outside `main`"},
			Suggestion: "move this code inside `fn main() { ... }`",
		}
		return e
	}
}

// checkTopLevelExpr checks an expression used as a bare top-level statement.
// Call expressions (e.g. test(), main()) are always forbidden at the top level.
func (interp *Interpreter) checkTopLevelExpr(expr *ast.Node, stmt *ast.Node) *errfmt.LunexError {
	if expr == nil {
		return nil
	}
	switch expr.Type {
	case ast.CallExpr:
		callee := expr.Callee
		calleeName := ""
		if callee != nil && callee.Type == ast.Identifier {
			calleeName = callee.Name
		}

		// Explicit main() call — special case with dedicated message.
		if calleeName == "main" {
			e := &errfmt.LunexError{
				Message: "explicit call to `main()` is not allowed",
				File:    interp.filename,
				Kind:    errfmt.KindSyntax,
				Code:    "E0072",
				Line:    stmt.Line,
				Col:     stmt.Col,
				Lines:   interp.sourceLines,
				Notes: []string{
					"`main` is the program entry point — Lunex calls it automatically",
					"calling `main()` manually creates unintended re-entry",
				},
				Suggestion: "remove the `main()` call — Lunex calls it for you when the program starts",
			}
			return e
		}

		// Generic bare call at the top level.
		suggestion := "move `" + calleeName + "(...)` inside `fn main() { ... }`"
		if calleeName == "" {
			suggestion = "move this call inside `fn main() { ... }`"
		}
		e := &errfmt.LunexError{
			Message: fmt.Sprintf("function call `%s(...)` is not allowed at the top level", calleeName),
			File:    interp.filename,
			Kind:    errfmt.KindSyntax,
			Code:    "E0071",
			Line:    stmt.Line,
			Col:     stmt.Col,
			Lines:   interp.sourceLines,
			Notes: []string{
				"top-level code is limited to declarations and imports",
				"all executable logic must live inside `fn main()`",
			},
			Suggestion: suggestion,
			ExBad:      "test()   // top-level call — not allowed",
			ExGood:     "fn main() {\n  test()   // call inside main — correct\n}",
		}
		return e

	case ast.AssignExpr:
		// Top-level assignment (e.g. x = 5) — not a declaration.
		e := &errfmt.LunexError{
			Message:    "top-level assignment is not allowed — use `val` or `var` to declare",
			File:       interp.filename,
			Kind:       errfmt.KindSyntax,
			Code:       "E0071",
			Line:       stmt.Line,
			Col:        stmt.Col,
			Lines:      interp.sourceLines,
			Suggestion: "use a declaration:  val name = <value>",
		}
		return e
	}
	// Other expressions (literals, etc.) — also not useful at top level, but
	// we let them through silently to avoid being overly strict about things
	// like string literals used as documentation comments in some styles.
	return nil
}

// ExecAsModule compiles and runs source as a module (no main() call, no KeepAliveWait).
// Returns the exported Value object — same as evalModuleSource but accessible from
// outside the runtime package (used by the bytecode NAX loader).
func (interp *Interpreter) ExecAsModule(source, filename string) (*Value, error) {
	prevFilename := interp.filename
	interp.filename = filename
	defer func() { interp.filename = prevFilename }()
	return interp.evalModuleSource(source, filename)
}

func (interp *Interpreter) CallMain() error {
	if interp.topEnv == nil {
		return nil
	}
	mainVal, ok := interp.topEnv.Get("main")
	if !ok || mainVal == nil {
		e := &errfmt.LunexError{
			Message: "entry point `main` is not defined",
			File:    interp.filename,
			Kind:    errfmt.KindReference,
			Code:    "E0070",
			Lines:   interp.sourceLines,
			Notes: []string{
				"every Lunex program requires a `fn main()` entry point",
				"top-level code outside `main` is not allowed in executable files",
			},
			Suggestion: "add a main function:\n\n  fn main() {\n    // your code here\n  }",
			ExGood:     "fn main() {\n  val io = @import(\"std.io\")\n  io.log(\"hello\")\n}",
			ExBad:      "val io = @import(\"std.io\")\nio.log(\"hello\")   // error: no main()",
		}
		return e
	}
	_, err := interp.callFunctionValue(mainVal, []*Value{}, nil)
	if err != nil {
		return err
	}
	return nil
}

// CallExport calls a named function that was defined in the last executed
// source. It converts Go values to Lunex *Value automatically, and converts
// the result back to a plain Go type (string, float64, bool, []interface{},
// map[string]interface{}, or nil). This is used by the self-hosted compiler
// bridge so Go code can call functions written in Lunex.
func (interp *Interpreter) CallExport(name string, args ...interface{}) (interface{}, error) {
	if interp.globals == nil {
		return nil, fmt.Errorf("interpreter not initialized")
	}
	fnVal, ok := interp.globals.Get(name)
	if !ok {
		return nil, fmt.Errorf("export %q not found", name)
	}
	lxArgs := make([]*Value, len(args))
	for i, a := range args {
		lxArgs[i] = goToValue(a)
	}
	result, err := interp.callFunctionValue(fnVal, lxArgs, nil)
	if err != nil {
		return nil, err
	}
	return valueToGo(result), nil
}

// goToValue converts a plain Go value to a Lunex *Value.
func goToValue(v interface{}) *Value {
	if v == nil {
		return Null
	}
	switch val := v.(type) {
	case bool:
		return BoolVal(val)
	case int:
		return NumberVal(float64(val))
	case int64:
		return NumberVal(float64(val))
	case float64:
		return NumberVal(val)
	case string:
		return StringVal(val)
	case []interface{}:
		arr := make([]*Value, len(val))
		for i, elem := range val {
			arr[i] = goToValue(elem)
		}
		return ArrayVal(arr)
	case map[string]interface{}:
		m := make(map[string]*Value, len(val))
		for k, elem := range val {
			m[k] = goToValue(elem)
		}
		return ObjectVal(m)
	default:
		return Null
	}
}

// valueToGo converts a Lunex *Value to a plain Go type.
func valueToGo(v *Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.Tag {
	case TypeNull, TypeUndefined:
		return nil
	case TypeBool:
		return v.BoolVal
	case TypeNumber:
		return v.NumVal
	case TypeString:
		return v.StrVal
	case TypeArray:
		out := make([]interface{}, len(v.ArrVal))
		for i, elem := range v.ArrVal {
			out[i] = valueToGo(elem)
		}
		return out
	case TypeObject:
		m := make(map[string]interface{}, len(v.ObjVal))
		for k, elem := range v.ObjVal {
			m[k] = valueToGo(elem)
		}
		return m
	default:
		return nil
	}
}

func (interp *Interpreter) execBlock(stmts []*ast.Node, env *Environment) (*Value, error) {
	var result *Value = Undefined
	for _, stmt := range stmts {
		val, err := interp.execNode(stmt, env)
		if err != nil {
			return nil, err
		}
		if val != nil {
			result = val
		}
	}
	return result, nil
}

func (interp *Interpreter) execNode(node *ast.Node, env *Environment) (*Value, error) {
	if node == nil {
		return Undefined, nil
	}
	if node.Line > 0 {
		interp.currentLine = node.Line
		interp.currentCol = node.Col
	}
	switch node.Type {
	case ast.Program:
		return interp.execBlock(node.Body_, env)
	case ast.Block:
		childEnv := NewEnvironment(env)
		return interp.execBlock(node.Body_, childEnv)
	case ast.VarDecl:
		return interp.execVarDecl(node, env)
	case ast.FnDecl:
		return interp.execFnDecl(node, env)
	case ast.ClassDecl:
		return interp.execClassDecl(node, env)
	case ast.EnumDecl:
		return interp.execEnumDecl(node, env)
	case ast.NamespaceDecl:
		return interp.execNamespace(node, env)
	case ast.ImportDecl:
		return interp.execImport(node, env)
	case ast.ExportDecl:
		return interp.execExport(node, env)
	case ast.LunexRequire:
		return interp.execLunexRequire(node, env)
	case ast.UseStmt:
		return interp.execUse(node, env)
	case ast.ImmutableDecl:
		return interp.execImmutable(node, env)
	case ast.UsingDecl:
		return interp.execUsing(node, env)
	case ast.ExprStmt:
		return interp.evalExpr(node.Expr, env)
	case ast.LogStmt:
		return interp.execLog(node, env)
	case ast.ReturnStmt:
		var val *Value = Undefined
		var err error
		if node.Value != nil {
			val, err = interp.evalExpr(node.Value.(*ast.Node), env)
			if err != nil {
				return nil, err
			}
		}
		return nil, &returnError{val: val}
	case ast.ThrowStmt, ast.RaiseStmt:
		val, err := interp.evalExpr(node.Value.(*ast.Node), env)
		if err != nil {
			return nil, err
		}
		return nil, &throwError{val: val}
	case ast.BreakStmt:
		return nil, &breakError{}
	case ast.ContinueStmt:
		return nil, &continueError{}
	case ast.IfStmt:
		return interp.execIf(node, env)
	case ast.UnlessStmt:
		return interp.execUnless(node, env)
	case ast.WhileStmt:
		return interp.execWhile(node, env)
	case ast.ForStmt:
		return interp.execFor(node, env)
	case ast.ForOfStmt, ast.EachInStmt:
		return interp.execForOf(node, env)
	case ast.RepeatStmt:
		return interp.execRepeat(node, env)
	case ast.LoopStmt:
		return interp.execLoop(node, env)
	case ast.MatchStmt:
		return interp.execMatch(node, env)
	case ast.TryStmt:
		return interp.execTry(node, env)
	case ast.GuardStmt:
		return interp.execGuard(node, env)
	case ast.DeferStmt:
		interp.defers = append(interp.defers, deferEntry{node: node, env: env})
		return Undefined, nil
	case ast.SpawnStmt:
		go func() {
			defer func() { recover() }()
			interp.evalExpr(node.Expr, env)
		}()
		return Undefined, nil
	case ast.AssertStmt:
		return interp.execAssert(node, env)
	case ast.HaveStmt:
		return interp.execHave(node, env)
	case ast.IfHaveStmt:
		return interp.execIfHave(node, env)
	case ast.IfSetStmt:
		return interp.execIfSet(node, env)
	case ast.DeleteStmt:
		return interp.execDelete(node, env)
	case ast.WithStmt:
		return interp.execWith(node, env)
	case ast.ComponentDecl:
		return interp.execComponent(node, env)
	case ast.SelectStmt:
		return interp.execSelect(node, env)
	case ast.DecoratedExpr:
		if node.Expr != nil {
			return interp.execNode(node.Expr, env)
		}
		return Undefined, nil
	default:
		return interp.evalExpr(node, env)
	}
}

func (interp *Interpreter) evalExpr(node *ast.Node, env *Environment) (*Value, error) {
	if node == nil {
		return Undefined, nil
	}
	if node.Line > 0 {
		interp.currentLine = node.Line
		interp.currentCol = node.Col
	}
	switch node.Type {
	case ast.NumberLit:
		return interp.evalNumber(node.Value)
	case ast.StringLit:
		s, _ := node.Value.(string)
		return StringVal(s), nil
	case ast.BoolLit:
		b, _ := node.Value.(bool)
		return BoolVal(b), nil
	case ast.NullLit:
		return Null, nil
	case ast.UndefinedLit:
		return Undefined, nil
	case ast.TemplateLit:
		return interp.evalTemplate(node, env)
	case ast.ArrayLit:
		return interp.evalArray(node, env)
	case ast.ObjectLit:
		return interp.evalObject(node, env)
	case ast.RegexLit:
		return interp.evalRegex(node)
	case ast.Identifier:
		return interp.evalIdentifier(node, env)
	case ast.ThisExpr:
		val, _ := env.Get("this")
		return val, nil
	case ast.SuperExpr:
		val, _ := env.Get("__super__")
		return val, nil
	case ast.VoidExpr:
		interp.evalExpr(node.Arg, env)
		return Undefined, nil
	case ast.TypeofExpr:
		return interp.evalTypeof(node, env)
	case ast.DeleteExpr:
		return interp.evalDelete(node, env)
	case ast.FnExpr, ast.FnDecl:
		return interp.evalFnExpr(node, env)
	case ast.ArrowFn:
		return interp.evalArrowFn(node, env)
	case ast.CallExpr:
		return interp.evalCall(node, env)
	case ast.NewExpr:
		return interp.evalNew(node, env)
	case ast.MemberExpr:
		return interp.evalMember(node, env)
	case ast.BinaryExpr:
		return interp.evalBinary(node, env)
	case ast.UnaryExpr:
		return interp.evalUnary(node, env)
	case ast.AssignExpr:
		return interp.evalAssign(node, env)
	case ast.TernaryExpr:
		return interp.evalTernary(node, env)
	case ast.SpreadExpr:
		return interp.evalExpr(node.Arg, env)
	case ast.PipelineExpr:
		return interp.evalPipeline(node, env)
	case ast.SequenceExpr:
		return interp.evalSequence(node, env)
	case ast.NotExpr:
		val, err := interp.evalExpr(node.Arg, env)
		if err != nil {
			return nil, err
		}
		return BoolVal(!val.IsTruthy()), nil
	case ast.HaveExpr:
		return interp.evalHaveExpr(node, env)
	case ast.TrySafeExpr:
		return interp.evalTrySafe(node, env)
	case ast.RangeExpr:
		return interp.evalRange(node, env)
	case ast.SleepExpr:
		return interp.evalSleep(node, env)
	case ast.ChannelExpr:
		return ChanV(NewChannel(64)), nil
	case ast.NaxImportExpr:
		return Null, nil
	case ast.AtImportExpr:
		return interp.evalAtImport(node, env)
	case ast.StructLit:
		return interp.evalStructLit(node, env)
	case ast.MatchStmt:
		return interp.evalMatchExpr(node, env)
	case ast.SatisfiesExpr:
		return interp.evalExpr(node.Expr, env)
	case ast.DecoratedExpr:
		if node.Expr != nil {
			return interp.evalExpr(node.Expr, env)
		}
		return Undefined, nil
	case ast.ExprStmt:
		return interp.evalExpr(node.Expr, env)
	default:
		return Undefined, nil
	}
}

func (interp *Interpreter) evalAtImport(node *ast.Node, env *Environment) (*Value, error) {
	path := node.Source
	resolved := resolveModulePath(path)
	if resolved == "native" && interp.libLoadDepth == 0 {
		e := interp.runtimeError(errfmt.KindImport, "E0014",
			fmt.Sprintf("module %q is internal and cannot be imported by user code — use a standard lib module like @import(\"std.io\")", path), node, nil)
		return nil, e
	}
	if forceLocalImport(node) {
		// @fimport: local files only — .lx source, .nax archive, or .nc bytecode.
		if localPath, ok := interp.resolveLocalFile(path); ok {
			return interp.loadLocalFile(localPath, node)
		}
		// Also let the ntlLoader serve bundled sources (e.g. inside a running .nax).
		if interp.ntlLoader != nil {
			if src, ok := interp.ntlLoader(path); ok {
				return interp.evalModuleSourceFile(src, path, path)
			}
		}
		e := interp.runtimeError(errfmt.KindImport, "E0010",
			fmt.Sprintf("local file %q not found", path), node, nil)
		e.Notes = append(e.Notes,
			"accepted extensions: .lx (source), .nax (archive), .nc (bytecode)",
			"path is resolved relative to the current file, then the working directory",
			"use @import(\"std.io\") for stdlib — @fimport is for local files only",
		)
		return nil, e
	}
	return interp.loadModule(path)
}

// loadLocalFile reads a local .lx, .nax, or .nc file and returns it as a module Value.
// absPath must already exist on disk (caller has verified via resolveLocalFile).
func (interp *Interpreter) loadLocalFile(localPath string, node *ast.Node) (*Value, error) {
	abs, _ := filepath.Abs(localPath)

	// Deduplication: if we already loaded this exact file, return cached copy.
	interp.mu.RLock()
	if mod, ok := interp.modules[abs]; ok {
		interp.mu.RUnlock()
		return mod, nil
	}
	interp.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(localPath))
	switch ext {
	case ".nax", ".nc":
		if interp.naxLoader == nil {
			e := interp.runtimeError(errfmt.KindImport, "E0015",
				fmt.Sprintf("cannot load %q: binary module loader is not available in this context", localPath), node, nil)
			return nil, e
		}
		mod, err := interp.naxLoader(abs)
		if err != nil {
			e := interp.runtimeError(errfmt.KindImport, "E0015",
				fmt.Sprintf("failed to load binary module %q: %v", localPath, err), node, nil)
			return nil, e
		}
		interp.mu.Lock()
		interp.modules[abs] = mod
		interp.mu.Unlock()
		return mod, nil

	default: // .lx or extensionless
		data, err := os.ReadFile(localPath)
		if err != nil {
			e := interp.runtimeError(errfmt.KindImport, "E0015",
				fmt.Sprintf("cannot read file %q: %v", localPath, err), node, nil)
			return nil, e
		}
		return interp.evalModuleSourceFile(string(data), abs, localPath)
	}
}
func (interp *Interpreter) evalStructLit(node *ast.Node, env *Environment) (*Value, error) {
	sEnv := NewEnvironment(env)

	for _, stmt := range node.Body_ {
		if stmt == nil {
			continue
		}
		if stmt.Type == ast.ExprStmt && stmt.Expr != nil && stmt.Expr.Type == ast.AssignExpr && stmt.Expr.Op == "=" {
			if left := stmt.Expr.Left; left != nil && left.Type == ast.Identifier && left.Name != "" {
				val, err := interp.evalExpr(stmt.Expr.Right, sEnv)
				if err != nil {
					return nil, err
				}
				sEnv.Define(left.Name, val, false)
				continue
			}
		}
		if _, err := interp.execNode(stmt, sEnv); err != nil {
			if _, ok := err.(*returnError); !ok {
				return nil, err
			}
		}
	}

	obj := make(map[string]*Value)
	hasFn := false
	for k, v := range sEnv.vars {
		if k == "self" || k == "this" || len(k) == 0 || k[0] == '_' {
			continue
		}
		if _, exists := obj[k]; exists {
			continue
		}
		obj[k] = v
		if v != nil && v.Tag == TypeFunction {
			hasFn = true
		}
	}

	structVal := ObjectVal(obj)
	sEnv.Define("self", structVal, false)
	sEnv.Define("this", structVal, false)

	if hasFn {
		MarkEscaped(sEnv)
	}
	return structVal, nil
}

func (interp *Interpreter) evalNumber(val interface{}) (*Value, error) {
	s, ok := val.(string)
	if !ok {
		if f, fok := val.(float64); fok {
			return NumberVal(f), nil
		}
		return NumberVal(0), nil
	}
	if cached, hit := interp.numCache.Load(s); hit {
		return NumberVal(cached.(float64)), nil
	}
	orig := s
	s = strings.ReplaceAll(s, "_", "")
	if strings.HasSuffix(s, "n") {
		s = s[:len(s)-1]
	}
	var f float64
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			interp.numCache.Store(orig, float64(0))
			return NumberVal(0), nil
		}
		f = float64(n)
	} else if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		n, err := strconv.ParseInt(s[2:], 8, 64)
		if err != nil {
			interp.numCache.Store(orig, float64(0))
			return NumberVal(0), nil
		}
		f = float64(n)
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		n, err := strconv.ParseInt(s[2:], 2, 64)
		if err != nil {
			interp.numCache.Store(orig, float64(0))
			return NumberVal(0), nil
		}
		f = float64(n)
	} else {
		var err error
		f, err = strconv.ParseFloat(s, 64)
		if err != nil {
			f = math.NaN()
		}
	}
	interp.numCache.Store(orig, f)
	return NumberVal(f), nil
}

var templateBuilderPool = sync.Pool{New: func() any { return new(strings.Builder) }}

func (interp *Interpreter) evalTemplate(node *ast.Node, env *Environment) (*Value, error) {
	raw, _ := node.Parts.(string)
	result := templateBuilderPool.Get().(*strings.Builder)
	result.Reset()
	defer templateBuilderPool.Put(result)
	i := 0
	for i < len(raw) {
		if raw[i] == '$' && i+1 < len(raw) && raw[i+1] == '{' {
			i += 2
			depth := 1
			start := i
			for i < len(raw) && depth > 0 {
				if raw[i] == '{' {
					depth++
				} else if raw[i] == '}' {
					depth--
				}
				if depth > 0 {
					i++
				}
			}
			exprStr := raw[start:i]
			i++
			val, err := interp.evalTemplateExpr(exprStr, env)
			if err != nil {
				result.WriteString("${error}")
			} else {
				result.WriteString(val.ToString())
			}
		} else if raw[i] == '\\' && i+1 < len(raw) {
			i++
			switch raw[i] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '`':
				result.WriteByte('`')
			default:
				result.WriteByte(raw[i])
			}
			i++
		} else {
			r, size := utf8.DecodeRuneInString(raw[i:])
			result.WriteRune(r)
			i += size
		}
	}
	return StringVal(result.String()), nil
}

func (interp *Interpreter) evalTemplateExpr(src string, env *Environment) (*Value, error) {
	if cached, ok := interp.templateCache.Load(src); ok {
		return interp.evalExpr(cached.(*ast.Node), env)
	}
	toks, err := lexer.Tokenize(src, "<template>")
	if err != nil {
		return StringVal(src), nil
	}
	prog, err := parser.Parse(toks, "<template>")
	if err != nil {
		return StringVal(src), nil
	}
	if len(prog.Body_) == 0 {
		return Undefined, nil
	}
	stmt := prog.Body_[0]
	var exprNode *ast.Node
	if stmt.Type == ast.ExprStmt && stmt.Expr != nil {
		exprNode = stmt.Expr
	} else {
		exprNode = stmt
	}
	interp.templateCache.Store(src, exprNode)
	return interp.evalExpr(exprNode, env)
}

func (interp *Interpreter) evalArray(node *ast.Node, env *Environment) (*Value, error) {
	var elements []*Value
	for _, el := range node.Elements {
		if el == nil {
			elements = append(elements, Undefined)
			continue
		}
		if el.Type == ast.SpreadExpr {
			val, err := interp.evalExpr(el.Arg, env)
			if err != nil {
				return nil, err
			}
			if suspErr := interp.CheckArraySpread(val, el); suspErr != nil {
				errfmt.Print(suspErr.(*errfmt.LunexError))
				// spread produces nothing; continue building the array
				continue
			}
			if val.Tag == TypeArray {
				elements = append(elements, val.ArrVal...)
			} else {
				elements = append(elements, val)
			}
			continue
		}
		val, err := interp.evalExpr(el, env)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}
	return ArrayVal(elements), nil
}

func (interp *Interpreter) evalObject(node *ast.Node, env *Environment) (*Value, error) {
	obj := make(map[string]*Value)
	for _, prop := range node.Properties {
		switch prop.Kind {
		case "spread":
			val, err := interp.evalExpr(prop.Arg, env)
			if err != nil {
				return nil, err
			}
			if suspErr := interp.CheckObjectSpread(val, prop.Arg); suspErr != nil {
				errfmt.Print(suspErr.(*errfmt.LunexError))
				// spread produces nothing; continue building the object
			} else if val.Tag == TypeObject {
				for k, v := range val.ObjVal {
					obj[k] = v
				}
			} else if val.Tag == TypeInstance {
				for k, v := range val.InstVal.Fields {
					obj[k] = v
				}
			}
		case "prop":
			var key string
			if prop.Computed {
				kv, err := interp.evalExpr(prop.Key.(*ast.Node), env)
				if err != nil {
					return nil, err
				}
				key = kv.ToString()
			} else {
				key, _ = prop.Key.(string)
			}
			val, err := interp.evalExpr(prop.Value, env)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		case "shorthand":
			key, _ := prop.Key.(string)
			val, _ := env.Get(key)
			obj[key] = val
		case "method":
			var key string
			if prop.Computed {
				if keyNode, ok := prop.Key.(*ast.Node); ok {
					kv, err := interp.evalExpr(keyNode, env)
					if err != nil {
						return nil, err
					}
					key = kv.ToString()
				}
			} else {
				key, _ = prop.Key.(string)
			}
			fn := &Function{
				Name:   key,
				Params: paramsToFnParams(prop.Params),
				Body:   prop.Body,
				Env:    env,
			}
			obj[key] = FuncVal(fn)
		}
	}
	return ObjectVal(obj), nil
}

func (interp *Interpreter) evalRegex(node *ast.Node) (*Value, error) {
	flags := ""
	pattern := node.Pattern
	if strings.Contains(node.Flags, "i") {
		flags += "(?i)"
	}
	if strings.Contains(node.Flags, "m") {
		flags += "(?m)"
	}
	// The 's' (dotAll) flag maps to Go's (?s) mode.
	if strings.Contains(node.Flags, "s") {
		flags += "(?s)"
	}
	// 'g' (global) has no Go equivalent; FindAll* methods already return all matches.
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return Null, nil
	}
	return RegexV(re), nil
}

func (interp *Interpreter) evalIdentifier(node *ast.Node, env *Environment) (*Value, error) {
	name := node.Name
	switch name {
	case "undefined":
		return Undefined, nil
	case "null":
		return Null, nil
	case "true":
		return True, nil
	case "false":
		return False, nil
	case "NaN":
		return NumberVal(math.NaN()), nil
	case "Infinity":
		return NumberVal(math.Inf(1)), nil
	}
	val, ok := env.Get(name)
	if !ok {
		allNames := visibleNames(env)
		similar := errfmt.FindSimilar(name, allNames)
		e := interp.runtimeError(errfmt.KindReference, "E0001",
			fmt.Sprintf("variable `%s` was not defined", name), node, similar)

		if len(similar) > 0 {
			e.Notes = append(e.Notes,
				fmt.Sprintf("did you mean `%s`? (closest match by name)", similar[0]))
		}

		// Show user-defined names only — filter out well-known built-ins
		// so the note is actually useful.
		noisy := map[string]bool{
			"Error": true, "Infinity": true, "NaN": true, "Math": true,
			"setInterval": true, "isFinite": true, "Number": true,
			"Map": true, "log": true, "str": true,
		}
		userNames := make([]string, 0, len(allNames))
		for _, n := range allNames {
			if !noisy[n] {
				userNames = append(userNames, n)
			}
		}
		if len(userNames) > 0 {
			visible := userNames
			if len(visible) > 6 {
				visible = visible[:6]
			}
			quoted := make([]string, len(visible))
			for i, n := range visible {
				quoted[i] = "`" + n + "`"
			}
			e.Notes = append(e.Notes, "names in scope: "+strings.Join(quoted, ", "))
		}
		return nil, e
	}
	return val, nil
}

func (interp *Interpreter) evalTypeof(node *ast.Node, env *Environment) (*Value, error) {
	val, _ := interp.evalExpr(node.Arg, env)
	if val == nil {
		return StringVal("undefined"), nil
	}
	return StringVal(val.TypeName()), nil
}

func (interp *Interpreter) evalDelete(node *ast.Node, env *Environment) (*Value, error) {
	if node.Arg.Type == ast.MemberExpr {
		obj, err := interp.evalExpr(node.Arg.Object, env)
		if err != nil {
			return nil, err
		}
		var key string
		if node.Arg.Computed {
			k, err := interp.evalExpr(node.Arg.Prop.(*ast.Node), env)
			if err != nil {
				return nil, err
			}
			key = k.ToString()
		} else {
			key, _ = node.Arg.Prop.(string)
		}
		if obj.Tag == TypeObject {
			delete(obj.ObjVal, key)
		}
	}
	return True, nil
}

func (interp *Interpreter) evalFnExpr(node *ast.Node, env *Environment) (*Value, error) {
	MarkEscaped(env)
	fn := &Function{
		Name:   node.Name,
		Params: paramsToFnParams(node.Params),
		Body:   node.Body,
		Env:    env,
	}
	return FuncVal(fn), nil
}

func (interp *Interpreter) evalArrowFn(node *ast.Node, env *Environment) (*Value, error) {
	MarkEscaped(env)
	// Capture 'this' from the enclosing lexical scope at creation time.
	capturedThis, _ := env.Get("this")
	fn := &Function{
		Name:         "",
		Params:       paramsToFnParams(node.Params),
		Body:         node.Body,
		Env:          env,
		IsArrow:      true,
		CapturedThis: capturedThis,
	}
	return FuncVal(fn), nil
}

func (interp *Interpreter) evalCall(node *ast.Node, env *Environment) (*Value, error) {
	var thisVal *Value = Undefined
	var fnVal *Value

	if node.Callee.Type == ast.MemberExpr {
		if node.Callee.Object != nil && node.Callee.Object.Type == ast.SuperExpr {
			superCls, _ := env.Get("__super_class__")
			if superCls != nil && superCls.Tag == TypeClass {
				key, _ := node.Callee.Prop.(string)
				if method, ok := superCls.ClsVal.Methods[key]; ok {
					thisVal, _ := env.Get("this")
					superArgs, err := interp.evalArgs(node.Args, env)
					if err != nil {
						return nil, err
					}
					return interp.callFunctionValue(FuncVal(method), superArgs, thisVal)
				}
			}
			return Undefined, nil
		}
		obj, err := interp.evalExpr(node.Callee.Object, env)
		if err != nil {
			return nil, err
		}
		if node.Optional && obj.IsNullish() {
			return Undefined, nil
		}
		thisVal = obj
		if node.Callee.Computed {
			k, err := interp.evalExpr(node.Callee.Prop.(*ast.Node), env)
			if err != nil {
				return nil, err
			}
			// For array subscript calls like arr[0](), use GetIndex so the
			// numeric index maps to the slot instead of a string property key.
			if obj.Tag == TypeArray && k.Tag == TypeNumber {
				fnVal = obj.GetIndex(int(k.NumVal))
			} else {
				fnVal = obj.Get(k.ToString())
			}
		} else {
			key, _ := node.Callee.Prop.(string)
			fnVal = obj.Get(key)
			// Channel method dispatch: send, recv, close
			if (fnVal == nil || fnVal.Tag == TypeUndefined) && obj.Tag == TypeChannel {
				ch := obj.ChanVal
				switch key {
				case "send":
					fnVal = FuncVal(&Function{Name: "send", Native: func(args []*Value, _ *Value) (*Value, error) {
						if len(args) > 0 {
							ch.Send(args[0])
						}
						return Undefined, nil
					}})
				case "recv":
					fnVal = FuncVal(&Function{Name: "recv", Native: func(args []*Value, _ *Value) (*Value, error) {
						return ch.Receive(), nil
					}})
				}
			}
			// Detect missing method and give a rich error with suggestions
			if fnVal == nil || fnVal.Tag == TypeUndefined {
				similar := errfmt.FindSimilar(key, objKeys(obj))
				objName := ""
				if node.Callee.Object != nil && node.Callee.Object.Type == ast.Identifier {
					objName = node.Callee.Object.Name
				}
				msg := fmt.Sprintf("method `%s` does not exist", key)
				if objName != "" {
					msg = fmt.Sprintf("method `%s` does not exist on `%s`", key, objName)
				}
				e := interp.runtimeError(errfmt.KindType, "E0002", msg, node, similar)
				avail := objKeys(obj)
				if len(avail) > 0 && len(avail) <= 12 {
					e.Notes = append(e.Notes, "available: "+strings.Join(avail, ", "))
				}
				return nil, e
			}
		}
	} else if node.Callee.Type == ast.SuperExpr {
		superCls, _ := env.Get("__super_class__")
		if superCls != nil && superCls.Tag == TypeClass {
			superArgs, err := interp.evalArgs(node.Args, env)
			if err != nil {
				return nil, err
			}
			childThis, hasThis := env.Get("this")
			if hasThis && childThis != nil && childThis.Tag == TypeInstance {
				return interp.runConstructorWithThis(superCls.ClsVal, superArgs, childThis, env)
			}
			return interp.callClass(superCls.ClsVal, superArgs, env)
		}
		return Undefined, nil
	} else {
		v, err := interp.evalExpr(node.Callee, env)
		if err != nil {
			return nil, err
		}
		fnVal = v
	}

	if node.Optional && fnVal.IsNullish() {
		return Undefined, nil
	}

	args, err := interp.evalArgs(node.Args, env)
	if err != nil {
		return nil, err
	}

	return interp.callFunctionValue(fnVal, args, thisVal)
}

func (interp *Interpreter) evalArgs(argNodes []*ast.Node, env *Environment) ([]*Value, error) {
	var args []*Value
	for _, argNode := range argNodes {
		if argNode.Type == ast.SpreadExpr {
			val, err := interp.evalExpr(argNode.Arg, env)
			if err != nil {
				return nil, err
			}
			if val.Tag == TypeArray {
				args = append(args, val.ArrVal...)
			} else {
				args = append(args, val)
			}
		} else {
			val, err := interp.evalExpr(argNode, env)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
	}
	return args, nil
}

func (interp *Interpreter) callFunctionValue(fnVal *Value, args []*Value, thisVal *Value) (*Value, error) {
	if fnVal == nil || fnVal.Tag == TypeNull || fnVal.Tag == TypeUndefined {
		typeName := "undefined"
		if fnVal != nil {
			typeName = fnVal.TypeName()
		}
		e := interp.runtimeError(errfmt.KindType, "E0003",
			fmt.Sprintf("value of type `%s` is not callable", typeName), nil, nil)
		e.Notes = append(e.Notes, "only values declared with `fn` can be called")
		return nil, e
	}
	if fnVal.Tag == TypeClass {
		return interp.callClass(fnVal.ClsVal, args, nil)
	}
	if fnVal.Tag != TypeFunction {
		e := interp.runtimeError(errfmt.KindType, "E0003",
			fmt.Sprintf("value of type `%s` is not callable (expected a function)", fnVal.TypeName()), nil, nil)
		e.Notes = append(e.Notes, fmt.Sprintf("the value is: %s", fnVal.ToString()))
		return nil, e
	}
	fn := fnVal.FnVal
	if fn.Native != nil {
		result, err := fn.Native(args, thisVal)
		return result, err
	}
	return interp.callUserFunction(fn, args, thisVal)
}

func (interp *Interpreter) callUserFunction(fn *Function, args []*Value, thisVal *Value) (*Value, error) {
	const maxCallDepth = 1000
	interp.callDepth++
	if interp.callDepth > maxCallDepth {
		interp.callDepth--
		fnName := fn.Name
		if fnName == "" {
			fnName = "<anonymous>"
		}
		return nil, interp.runtimeError(errfmt.KindRecursion, errfmt.ErrStackOverflow,
			"maximum call depth exceeded (infinite recursion in '"+fnName+"')",
			nil, []string{"check for a function that calls itself without a base case"})
	}
	defer func() { interp.callDepth-- }()

	// Per-function profiling with per-function sampling (fixes: inflated *32, global callCount).
	var fnProf *jit.FnProfile
	var t0 int64
	if fn.Name != "" {
		fnProf = interp.profiler.GetOrCreate(fn.Name)
		if fnProf.ShouldSample() {
			t0 = time.Now().UnixNano()
		}
	}

	// Save outer defer stack so this call frame gets its own clean slate.
	savedDefers := interp.defers
	interp.defers = nil

	fnEnv := NewEnvironment(fn.Env)
	// Return fnEnv to the pool once this call frame exits.
	// The defer fires after all local uses of fnEnv are done, including
	// the deferred-statement flush below.
	defer ReleaseEnvironment(fnEnv)
	// Arrow functions capture 'this' lexically; regular functions use the call-site 'this'.
	effectiveThis := thisVal
	if fn.IsArrow && fn.CapturedThis != nil {
		effectiveThis = fn.CapturedThis
	}
	if effectiveThis != nil {
		fnEnv.Define("this", effectiveThis, false)
	}
	if fn.DefClass != nil && fn.DefClass.Super != nil {
		fnEnv.Define("__super_class__", ClassVal(fn.DefClass.Super), false)
	}
	bodyNode, _ := fn.Body.(*ast.Node)
	if bodyNode == nil {
		interp.defers = savedDefers
		return Undefined, nil
	}
	err := interp.bindParams(fn.Params, args, fnEnv)
	if err != nil {
		interp.defers = savedDefers
		return nil, err
	}
	var result *Value = Undefined
	var execErr error
	if bodyNode.Type == ast.Block {
		stmts := bodyNode.Body_
		// Hoist all fn declarations inside the block so that functions
		// declared anywhere in the body are visible from the first statement.
		// This allows calling a nested fn before its declaration site.
		for _, stmt := range stmts {
			if stmt != nil && stmt.Type == ast.FnDecl && stmt.Name != "" {
				if _, already := fnEnv.GetLocal(stmt.Name); !already {
					MarkEscaped(fnEnv)
					fn := &Function{
						Name:   stmt.Name,
						Params: paramsToFnParams(stmt.Params),
						Body:   stmt.Body,
						Env:    fnEnv,
					}
					fnEnv.Define(stmt.Name, FuncVal(fn), false)
				}
			}
		}
		for i, stmt := range stmts {
			val, e := interp.execNode(stmt, fnEnv)
			if e != nil {
				if re, ok := e.(*returnError); ok {
					result = re.val
					break
				}
				execErr = e
				break
			}
			if i == len(stmts)-1 && (stmt.Type == ast.ExprStmt || stmt.Type == ast.MatchStmt ||
				stmt.Type == ast.IfStmt || stmt.Type == ast.UnlessStmt ||
				stmt.Type == ast.TryStmt || stmt.Type == ast.Block ||
				stmt.Type == ast.FnExpr || stmt.Type == ast.FnDecl) {
				result = val
			}
		}
	} else {
		result, execErr = interp.evalExpr(bodyNode, fnEnv)
	}

	// Execute deferred statements in LIFO order before returning.
	// The DeferStmt parser stores the deferred block in node.Body (*ast.Node),
	// not node.Expr — use execNode on the body node directly.
	localDefers := interp.defers
	interp.defers = savedDefers
	for i := len(localDefers) - 1; i >= 0; i-- {
		de := localDefers[i]
		if de.node.Body != nil {
			interp.execNode(de.node.Body, de.env)
		} else if de.node.Expr != nil {
			interp.evalExpr(de.node.Expr, de.env)
		}
	}

	// Record actual elapsed time without the erroneous *32 inflation.
	if fnProf != nil && t0 != 0 {
		elapsed := time.Now().UnixNano() - t0
		if interp.profiler.RecordAndCheckHot(fn.Name, elapsed) {
			fnProf.PromoteToFastGo()
		}
	}

	if execErr != nil {
		if re, ok := execErr.(*returnError); ok {
			return re.val, nil
		}
		return nil, execErr
	}
	return result, nil
}

func (interp *Interpreter) bindParams(params []FnParam, args []*Value, env *Environment) error {
	for i, param := range params {
		if param.Rest {
			var rest []*Value
			if i < len(args) {
				rest = args[i:]
			}
			env.Define(param.Name, ArrayVal(rest), false)
			break
		}
		var val *Value
		if i < len(args) {
			val = args[i]
		} else {
			if param.Default != nil {
				defNode, ok := param.Default.(*ast.Node)
				if ok {
					var err error
					val, err = interp.evalExpr(defNode, env)
					if err != nil {
						return err
					}
				}
			}
			if val == nil {
				val = Undefined
			}
		}
		if param.Destructure != nil {
			if err := interp.bindDestructure(param.Destructure, val, env); err != nil {
				return err
			}
		} else {
			env.Define(param.Name, val, false)
		}
	}
	return nil
}

func (interp *Interpreter) bindDestructure(pattern interface{}, val *Value, env *Environment) error {
	m, ok := pattern.(map[string]interface{})
	if !ok {
		return nil
	}
	kind, _ := m["kind"].(string)
	switch kind {
	case "object":
		props, _ := m["props"].([]map[string]interface{})
		for _, prop := range props {
			key, _ := prop["key"].(string)
			alias, _ := prop["alias"].(string)
			if alias == "" {
				alias = key
			}
			fieldVal := val.Get(key)
			if fieldVal.IsNullish() {
				if defNode, ok := prop["default"]; ok && defNode != nil {
					if dn, ok := defNode.(*ast.Node); ok {
						v, err := interp.evalExpr(dn, env)
						if err != nil {
							return err
						}
						fieldVal = v
					}
				}
			}
			env.Define(alias, fieldVal, false)
		}
	case "array":
		items, _ := m["items"].([]interface{})
		for i, item := range items {
			if item == nil {
				continue
			}
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if itemMap["kind"] == "rest" {
				name, _ := itemMap["name"].(string)
				var rest []*Value
				if val.Tag == TypeArray && i < len(val.ArrVal) {
					rest = val.ArrVal[i:]
				}
				env.Define(name, ArrayVal(rest), false)
				break
			}
			name, _ := itemMap["name"].(string)
			var fieldVal *Value
			if val.Tag == TypeArray && i < len(val.ArrVal) {
				fieldVal = val.ArrVal[i]
			}
			if fieldVal == nil || fieldVal.IsNullish() {
				if defNode, ok := itemMap["default"]; ok && defNode != nil {
					if dn, ok := defNode.(*ast.Node); ok {
						v, err := interp.evalExpr(dn, env)
						if err != nil {
							return err
						}
						fieldVal = v
					}
				}
			}
			if fieldVal == nil {
				fieldVal = Undefined
			}
			env.Define(name, fieldVal, false)
		}
	}
	return nil
}

func (interp *Interpreter) evalNew(node *ast.Node, env *Environment) (*Value, error) {
	calleeVal, err := interp.evalExpr(node.Callee, env)
	if err != nil {
		return nil, err
	}
	args, err := interp.evalArgs(node.Args, env)
	if err != nil {
		return nil, err
	}
	if calleeVal.Tag == TypeClass {
		return interp.callClass(calleeVal.ClsVal, args, env)
	}
	if calleeVal.Tag == TypeFunction {
		inst := &Instance{
			Class:  &Class{Name: calleeVal.FnVal.Name},
			Fields: make(map[string]*Value),
		}
		instVal := InstVal(inst)
		_, err := interp.callFunctionValue(calleeVal, args, instVal)
		if err != nil {
			return nil, err
		}
		return instVal, nil
	}
	return Null, nil
}

func (interp *Interpreter) callClass(cls *Class, args []*Value, outerEnv *Environment) (*Value, error) {
	inst := NewInstance(cls)
	instVal := InstVal(inst)
	// Super-class fields are initialized by the constructor via super() calls;
	// iterating a freshly-allocated superInst.Fields (always empty) is dead code.
	if initFn, ok := cls.Methods["constructor"]; ok {
		fnEnv := NewEnvironment(initFn.Env)
		fnEnv.Define("this", instVal, false)
		if cls.Super != nil {
			fnEnv.Define("__super_class__", ClassVal(cls.Super), false)
		}
		err := interp.bindParams(initFn.Params, args, fnEnv)
		if err != nil {
			return nil, err
		}
		bodyNode, ok := initFn.Body.(*ast.Node)
		if ok {
			for _, stmt := range bodyNode.Body_ {
				_, e := interp.execNode(stmt, fnEnv)
				if e != nil {
					if _, ok := e.(*returnError); ok {
						break
					}
					return nil, e
				}
			}
		}
	}
	return instVal, nil
}

func (interp *Interpreter) runConstructorWithThis(cls *Class, args []*Value, thisVal *Value, outerEnv *Environment) (*Value, error) {
	if initFn, ok := cls.Methods["constructor"]; ok {
		fnEnv := NewEnvironment(initFn.Env)
		fnEnv.Define("this", thisVal, false)
		if cls.Super != nil {
			fnEnv.Define("__super_class__", ClassVal(cls.Super), false)
		}
		err := interp.bindParams(initFn.Params, args, fnEnv)
		if err != nil {
			return nil, err
		}
		bodyNode, ok := initFn.Body.(*ast.Node)
		if ok {
			for _, stmt := range bodyNode.Body_ {
				_, e := interp.execNode(stmt, fnEnv)
				if e != nil {
					if _, ok := e.(*returnError); ok {
						break
					}
					return nil, e
				}
			}
		}
	}
	return thisVal, nil
}

func (interp *Interpreter) evalMember(node *ast.Node, env *Environment) (*Value, error) {
	obj, err := interp.evalExpr(node.Object, env)
	if err != nil {
		return nil, err
	}
	if node.Optional && obj.IsNullish() {
		return Undefined, nil
	}
	if node.Computed {
		propVal, err := interp.evalExpr(node.Prop.(*ast.Node), env)
		if err != nil {
			return nil, err
		}
		if obj.Tag == TypeArray {
			if propVal.Tag == TypeNumber {
				idx := int(propVal.NumVal)
				if v, suspErr := interp.CheckIndexBounds(obj, idx, node); suspErr != nil {
					errfmt.Print(suspErr.(*errfmt.LunexError))
					return v, nil
				}
				return obj.GetIndex(idx), nil
			}
			return obj.Get(propVal.ToString()), nil
		}
		if obj.Tag == TypeString && propVal.Tag == TypeNumber {
			return obj.GetIndex(int(propVal.NumVal)), nil
		}
		return obj.Get(propVal.ToString()), nil
	}
	key, _ := node.Prop.(string)
	// If object is null/undefined, give a clear error instead of panicking
	if obj.Tag == TypeNull || obj.Tag == TypeUndefined {
		objName := ""
		if node.Object != nil && node.Object.Type == ast.Identifier {
			objName = node.Object.Name
		}
		msg := fmt.Sprintf("cannot read property `%s` of %s", key, obj.TypeName())
		if objName != "" {
			msg = fmt.Sprintf("cannot read property `%s` of `%s` (which is %s)", key, objName, obj.TypeName())
		}
		e := interp.runtimeError(errfmt.KindType, "E0004", msg, node, nil)
		e.Notes = append(e.Notes, "guard with: if "+func() string {
			if objName != "" {
				return objName
			}
			return "value"
		}()+" != null { ... }")
		return nil, e
	}
	return obj.Get(key), nil
}

func (interp *Interpreter) evalBinary(node *ast.Node, env *Environment) (*Value, error) {
	op := node.Op
	if op == "&&" {
		left, err := interp.evalExpr(node.Left, env)
		if err != nil {
			return nil, err
		}
		if !left.IsTruthy() {
			return left, nil
		}
		return interp.evalExpr(node.Right, env)
	}
	if op == "||" {
		left, err := interp.evalExpr(node.Left, env)
		if err != nil {
			return nil, err
		}
		if left.IsTruthy() {
			return left, nil
		}
		return interp.evalExpr(node.Right, env)
	}
	if op == "??" {
		left, err := interp.evalExpr(node.Left, env)
		if err != nil {
			return nil, err
		}
		if !left.IsNullish() {
			return left, nil
		}
		return interp.evalExpr(node.Right, env)
	}

	left, err := interp.evalExpr(node.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := interp.evalExpr(node.Right, env)
	if err != nil {
		return nil, err
	}

	switch op {
	case "+":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return NumberVal(left.NumVal + right.NumVal), nil
		}
		if left.Tag == TypeString && right.Tag == TypeString {
			return StringVal(left.StrVal + right.StrVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '+' cannot combine %s and %s — use explicit conversion (e.g. str(x))", left.TypeName(), right.TypeName()), node, nil)
		}
		result := left.ToNumber() + right.ToNumber()
		if v, suspErr := interp.CheckNaNResult(result, "+", left, right, node); suspErr != nil {
			errfmt.Print(suspErr.(*errfmt.LunexError))
			return v, nil
		}
		return NumberVal(result), nil
	case "-":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return NumberVal(left.NumVal - right.NumVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '-' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), node, nil)
		}
		result := left.ToNumber() - right.ToNumber()
		if v, suspErr := interp.CheckNaNResult(result, "-", left, right, node); suspErr != nil {
			errfmt.Print(suspErr.(*errfmt.LunexError))
			return v, nil
		}
		return NumberVal(result), nil
	case "*":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return NumberVal(left.NumVal * right.NumVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '*' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), node, nil)
		}
		result := left.ToNumber() * right.ToNumber()
		if v, suspErr := interp.CheckNaNResult(result, "*", left, right, node); suspErr != nil {
			errfmt.Print(suspErr.(*errfmt.LunexError))
			return v, nil
		}
		return NumberVal(result), nil
	case "/":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			if right.NumVal == 0 {
				return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
					"division by zero", node, nil)
			}
			return NumberVal(left.NumVal / right.NumVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '/' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), node, nil)
		}
		r := right.ToNumber()
		if r == 0 {
			return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
				"division by zero", node, nil)
		}
		result := left.ToNumber() / r
		if v, suspErr := interp.CheckNaNResult(result, "/", left, right, node); suspErr != nil {
			errfmt.Print(suspErr.(*errfmt.LunexError))
			return v, nil
		}
		return NumberVal(result), nil
	case "%":
		if right.Tag == TypeNumber && right.NumVal == 0 {
			return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
				"modulo by zero", node, nil)
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '%%' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), node, nil)
		}
		r := right.ToNumber()
		if r == 0 {
			return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
				"modulo by zero", node, nil)
		}
		result := math.Mod(left.ToNumber(), r)
		if v, suspErr := interp.CheckNaNResult(result, "%", left, right, node); suspErr != nil {
			errfmt.Print(suspErr.(*errfmt.LunexError))
			return v, nil
		}
		return NumberVal(result), nil
	case "**":
		result := math.Pow(left.ToNumber(), right.ToNumber())
		if v, suspErr := interp.CheckNaNResult(result, "**", left, right, node); suspErr != nil {
			errfmt.Print(suspErr.(*errfmt.LunexError))
			return v, nil
		}
		return NumberVal(result), nil
	case "===":
		return BoolVal(!left.StrictEquals(right)), nil
	case "==":
		return BoolVal(left.Equals(right)), nil
	case "!=":
		return BoolVal(!left.Equals(right)), nil
	case "<":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return BoolVal(left.NumVal < right.NumVal), nil
		}
		if left.Tag == TypeString && right.Tag == TypeString {
			return BoolVal(left.StrVal < right.StrVal), nil
		}
		return BoolVal(left.ToNumber() < right.ToNumber()), nil
	case ">":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return BoolVal(left.NumVal > right.NumVal), nil
		}
		if left.Tag == TypeString && right.Tag == TypeString {
			return BoolVal(left.StrVal > right.StrVal), nil
		}
		return BoolVal(left.ToNumber() > right.ToNumber()), nil
	case "<=":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return BoolVal(left.NumVal <= right.NumVal), nil
		}
		if left.Tag == TypeString && right.Tag == TypeString {
			return BoolVal(left.StrVal <= right.StrVal), nil
		}
		return BoolVal(left.ToNumber() <= right.ToNumber()), nil
	case ">=":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return BoolVal(left.NumVal >= right.NumVal), nil
		}
		if left.Tag == TypeString && right.Tag == TypeString {
			return BoolVal(left.StrVal >= right.StrVal), nil
		}
		return BoolVal(left.ToNumber() >= right.ToNumber()), nil
	case "&":
		return NumberVal(float64(int64(left.ToNumber()) & int64(right.ToNumber()))), nil
	case "|":
		return NumberVal(float64(int64(left.ToNumber()) | int64(right.ToNumber()))), nil
	case "^":
		return NumberVal(float64(int64(left.ToNumber()) ^ int64(right.ToNumber()))), nil
	case "<<":
		return NumberVal(float64(int64(left.ToNumber()) << uint(right.ToNumber()))), nil
	case ">>":
		return NumberVal(float64(int64(left.ToNumber()) >> uint(right.ToNumber()))), nil
	case ">>>":
		return NumberVal(float64(uint64(left.ToNumber()) >> uint(right.ToNumber()))), nil
	case "instanceof":
		if right.Tag == TypeClass && left.Tag == TypeInstance {
			return BoolVal(isInstanceOf(left.InstVal, right.ClsVal)), nil
		}
		return False, nil
	case "in":
		key := left.ToString()
		switch right.Tag {
		case TypeObject:
			_, ok := right.ObjVal[key]
			return BoolVal(ok), nil
		case TypeArray:
			idx := int(left.ToNumber())
			if left.Tag == TypeNumber && idx >= 0 && idx < len(right.ArrVal) {
				return True, nil
			}
			return False, nil
		case TypeInstance:
			_, ok := right.InstVal.Fields[key]
			return BoolVal(ok), nil
		}
		return False, nil
	}
	return Undefined, nil
}

func (interp *Interpreter) evalUnary(node *ast.Node, env *Environment) (*Value, error) {
	if node.Op == "++" || node.Op == "--" {
		val, err := interp.evalExpr(node.Arg, env)
		if err != nil {
			return nil, err
		}
		num := val.ToNumber()
		var newNum float64
		if node.Op == "++" {
			newNum = num + 1
		} else {
			newNum = num - 1
		}
		newVal := NumberVal(newNum)
		interp.assignToNode(node.Arg, newVal, env)
		if node.Prefix {
			return newVal, nil
		}
		return val, nil
	}
	arg, err := interp.evalExpr(node.Arg, env)
	if err != nil {
		return nil, err
	}
	switch node.Op {
	case "!":
		return BoolVal(!arg.IsTruthy()), nil
	case "-":
		return NumberVal(-arg.ToNumber()), nil
	case "+":
		return NumberVal(arg.ToNumber()), nil
	case "~":
		return NumberVal(float64(^int64(arg.ToNumber()))), nil
	}
	return Undefined, nil
}

func (interp *Interpreter) evalAssign(node *ast.Node, env *Environment) (*Value, error) {
	right, err := interp.evalExpr(node.Right, env)
	if err != nil {
		return nil, err
	}
	if node.Op != "=" {
		left, err := interp.evalExpr(node.Left, env)
		if err != nil {
			return nil, err
		}
		op := node.Op[:len(node.Op)-1]
		right, err = interp.evalBinaryValues(left, right, op)
		if err != nil {
			return nil, err
		}
	}
	err = interp.assignToNode(node.Left, right, env)
	if err != nil {
		return nil, err
	}
	return right, nil
}

func (interp *Interpreter) evalBinaryValues(left, right *Value, op string) (*Value, error) {
	switch op {
	case "+":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return NumberVal(left.NumVal + right.NumVal), nil
		}
		if left.Tag == TypeString && right.Tag == TypeString {
			return StringVal(left.StrVal + right.StrVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '+' cannot combine %s and %s — use explicit conversion (e.g. str(x))", left.TypeName(), right.TypeName()), nil, nil)
		}
		return NumberVal(left.ToNumber() + right.ToNumber()), nil
	case "-":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return NumberVal(left.NumVal - right.NumVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '-' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), nil, nil)
		}
		return NumberVal(left.ToNumber() - right.ToNumber()), nil
	case "*":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			return NumberVal(left.NumVal * right.NumVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '*' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), nil, nil)
		}
		return NumberVal(left.ToNumber() * right.ToNumber()), nil
	case "/":
		if left.Tag == TypeNumber && right.Tag == TypeNumber {
			if right.NumVal == 0 {
				return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
					"division by zero", nil, nil)
			}
			return NumberVal(left.NumVal / right.NumVal), nil
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '/' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), nil, nil)
		}
		r2 := right.ToNumber()
		if r2 == 0 {
			return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
				"division by zero", nil, nil)
		}
		return NumberVal(left.ToNumber() / r2), nil
	case "%":
		if right.Tag == TypeNumber && right.NumVal == 0 {
			return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
				"modulo by zero", nil, nil)
		}
		if left.Tag == TypeString || right.Tag == TypeString {
			return nil, interp.runtimeError(errfmt.KindType, errfmt.ErrTypeMismatch,
				fmt.Sprintf("type error: '%%' cannot be applied to %s and %s", left.TypeName(), right.TypeName()), nil, nil)
		}
		r2 := right.ToNumber()
		if r2 == 0 {
			return nil, interp.runtimeError(errfmt.KindArithmetic, errfmt.ErrDivisionByZero,
				"modulo by zero", nil, nil)
		}
		return NumberVal(math.Mod(left.ToNumber(), r2)), nil
	case "**":
		return NumberVal(math.Pow(left.ToNumber(), right.ToNumber())), nil
	case "&&":
		if !left.IsTruthy() {
			return left, nil
		}
		return right, nil
	case "||":
		if left.IsTruthy() {
			return left, nil
		}
		return right, nil
	case "??":
		if !left.IsNullish() {
			return left, nil
		}
		return right, nil
	case "<<":
		return NumberVal(float64(int64(left.ToNumber()) << uint(right.ToNumber()))), nil
	case ">>":
		return NumberVal(float64(int64(left.ToNumber()) >> uint(right.ToNumber()))), nil
	}
	return right, nil
}

func (interp *Interpreter) assignToNode(target *ast.Node, val *Value, env *Environment) error {
	switch target.Type {
	case ast.Identifier:
		if err := env.Set(target.Name, val); err != nil {
			if le, ok := err.(*errfmt.LunexError); ok {
				if le.Line == 0 {
					le.File = interp.filename
					le.Lines = interp.sourceLines
					le.Line = target.Line
					le.Col = target.Col
					if le.Line == 0 {
						le.Line = interp.currentLine
						le.Col = interp.currentCol
					}
				}
			}
			return err
		}
		return nil
	case ast.MemberExpr:
		obj, err := interp.evalExpr(target.Object, env)
		if err != nil {
			return err
		}
		if target.Computed {
			keyVal, err := interp.evalExpr(target.Prop.(*ast.Node), env)
			if err != nil {
				return err
			}
			key := keyVal.ToString()
			if obj.Tag == TypeArray {
				idx := int(keyVal.ToNumber())
				for len(obj.ArrVal) <= idx {
					obj.ArrVal = append(obj.ArrVal, Undefined)
				}
				obj.ArrVal[idx] = val
			} else {
				obj.Set(key, val)
			}
		} else {
			key, _ := target.Prop.(string)
			obj.Set(key, val)
		}
	}
	return nil
}

func (interp *Interpreter) evalTernary(node *ast.Node, env *Environment) (*Value, error) {
	cond, err := interp.evalExpr(node.Test, env)
	if err != nil {
		return nil, err
	}
	if cond.IsTruthy() {
		return interp.evalExpr(node.Consequent, env)
	}
	return interp.evalExpr(node.Alternate, env)
}

func (interp *Interpreter) evalPipeline(node *ast.Node, env *Environment) (*Value, error) {
	left, err := interp.evalExpr(node.Left, env)
	if err != nil {
		return nil, err
	}
	fn, err := interp.evalExpr(node.Right, env)
	if err != nil {
		return nil, err
	}
	return interp.callFunctionValue(fn, []*Value{left}, nil)
}

func (interp *Interpreter) evalSequence(node *ast.Node, env *Environment) (*Value, error) {
	var result *Value = Undefined
	for _, e := range node.Exprs {
		val, err := interp.evalExpr(e, env)
		if err != nil {
			return nil, err
		}
		result = val
	}
	return result, nil
}

func (interp *Interpreter) evalHaveExpr(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return nil, err
	}
	return BoolVal(interp.testHaveCondition(val, node, env)), nil
}

func (interp *Interpreter) testHaveCondition(val *Value, node *ast.Node, env *Environment) bool {
	if node.InExpr == nil && node.MatchMode == "" {
		return val != nil && !val.IsNullish() && val.IsTruthy()
	}
	var inVal *Value
	if node.InExpr != nil {
		if inNode, ok := node.InExpr.(*ast.Node); ok {
			inVal, _ = interp.evalExpr(inNode, env)
		}
	}
	switch node.MatchMode {
	case "in":
		if inVal == nil {
			return false
		}
		if inVal.Tag == TypeArray {
			for _, e := range inVal.ArrVal {
				if e != nil && e.StrictEquals(val) {
					return true
				}
			}
			return false
		}
		if inVal.Tag == TypeObject {
			_, ok := inVal.ObjVal[val.ToString()]
			return ok
		}
		if inVal.Tag == TypeString {
			return strings.Contains(inVal.StrVal, val.ToString())
		}
		return false
	case "not_in":
		if inVal == nil {
			return true
		}
		if inVal.Tag == TypeArray {
			for _, e := range inVal.ArrVal {
				if e != nil && e.StrictEquals(val) {
					return false
				}
			}
			return true
		}
		return true
	case "matches":
		if inVal != nil && inVal.Tag == TypeRegex {
			return inVal.RegexVal.MatchString(val.ToString())
		}
		return false
	case "is":
		if inVal == nil {
			return false
		}
		typeName := inVal.ToString()
		switch strings.ToLower(typeName) {
		case "string":
			return val.Tag == TypeString
		case "number":
			return val.Tag == TypeNumber
		case "boolean":
			return val.Tag == TypeBool
		case "null":
			return val.Tag == TypeNull
		case "undefined":
			return val.Tag == TypeUndefined
		case "array":
			return val.Tag == TypeArray
		case "object":
			return val.Tag == TypeObject || val.Tag == TypeInstance
		case "function":
			return val.Tag == TypeFunction
		}
		if inVal.Tag == TypeClass && val.Tag == TypeInstance {
			return isInstanceOf(val.InstVal, inVal.ClsVal)
		}
		return false
	case "is_not":
		if inVal == nil {
			return true
		}
		return !interp.testHaveCondition(val, &ast.Node{InExpr: node.InExpr, MatchMode: "is"}, env)
	case "between":
		lo, _ := interp.evalExpr(node.Lo, env)
		hi, _ := interp.evalExpr(node.Hi, env)
		n := val.ToNumber()
		return n >= lo.ToNumber() && n <= hi.ToNumber()
	case "startsWith":
		if inVal != nil {
			return strings.HasPrefix(val.ToString(), inVal.ToString())
		}
		return false
	case "endsWith":
		if inVal != nil {
			return strings.HasSuffix(val.ToString(), inVal.ToString())
		}
		return false
	default:
		return !val.IsNullish() && val.IsTruthy()
	}
}

func (interp *Interpreter) evalTrySafe(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return Null, nil
	}
	return val, nil
}

func (interp *Interpreter) evalRange(node *ast.Node, env *Environment) (*Value, error) {
	if len(node.Args) == 0 {
		return ArrayVal(nil), nil
	}
	if len(node.Args) == 1 {
		n, err := interp.evalExpr(node.Args[0], env)
		if err != nil {
			return nil, err
		}
		count := int(n.ToNumber())
		result := make([]*Value, count)
		for i := 0; i < count; i++ {
			result[i] = NumberVal(float64(i))
		}
		return ArrayVal(result), nil
	}
	startVal, _ := interp.evalExpr(node.Args[0], env)
	endVal, _ := interp.evalExpr(node.Args[1], env)
	step := 1.0
	if len(node.Args) > 2 {
		sv, _ := interp.evalExpr(node.Args[2], env)
		step = sv.ToNumber()
	}
	start := startVal.ToNumber()
	end := endVal.ToNumber()
	if step == 0 {
		return ArrayVal(nil), nil
	}
	count := int(math.Max(0, math.Ceil((end-start)/step)))
	result := make([]*Value, count)
	for i := 0; i < count; i++ {
		result[i] = NumberVal(start + float64(i)*step)
	}
	return ArrayVal(result), nil
}

func (interp *Interpreter) evalSleep(node *ast.Node, env *Environment) (*Value, error) {
	ms, err := interp.evalExpr(node.Ms, env)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Duration(ms.ToNumber()) * time.Millisecond)
	return Undefined, nil
}

func (interp *Interpreter) evalMatchExpr(node *ast.Node, env *Environment) (*Value, error) {
	subject, err := interp.evalExpr(node.Subject, env)
	if err != nil {
		return nil, err
	}
	for _, mc := range node.Cases {
		if mc.IsDefault {
			return interp.execNode(mc.Body, env)
		}
		for _, pat := range mc.Patterns {
			bindings := make(map[string]*Value)
			if interp.matchPattern(subject, pat, bindings) {
				if mc.Guard != nil {
					caseEnv := NewEnvironment(env)
					for k, v := range bindings {
						caseEnv.Define(k, v, false)
					}
					guardVal, err := interp.evalExpr(mc.Guard, caseEnv)
					if err != nil {
						return nil, err
					}
					if !guardVal.IsTruthy() {
						continue
					}
				}
				caseEnv := NewEnvironment(env)
				for k, v := range bindings {
					caseEnv.Define(k, v, false)
				}
				result, err := interp.execNode(mc.Body, caseEnv)
				if err != nil {
					if re, ok := err.(*returnError); ok {
						return re.val, nil
					}
					return nil, err
				}
				return result, nil
			}
		}
	}
	// No arm matched — emit S0002 and return undefined
	if v, suspErr := interp.CheckMatchResult(subject, node); suspErr != nil {
		errfmt.Print(suspErr.(*errfmt.LunexError))
		return v, nil
	}
	return Undefined, nil
}

func (interp *Interpreter) matchPattern(val *Value, pat *ast.MatchPattern, bindings map[string]*Value) bool {
	switch pat.Kind {
	case "wildcard":
		return true
	case "binding":
		bindings[pat.Name] = val
		return true
	case "literal":
		switch pv := pat.Value.(type) {
		case nil:
			return val.Tag == TypeNull
		case bool:
			return val.Tag == TypeBool && val.BoolVal == pv
		case string:
			if pv == "undefined" {
				return val.Tag == TypeUndefined
			}
			f, err := strconv.ParseFloat(pv, 64)
			if err == nil {
				return val.Tag == TypeNumber && val.NumVal == f
			}
			return val.Tag == TypeString && val.StrVal == pv
		}
		return false
	case "array":
		if val.Tag != TypeArray {
			return false
		}
		for i, item := range pat.Items {
			if item.Kind == "rest" {
				bindings[item.Name] = ArrayVal(val.ArrVal[i:])
				return true
			}
			if i >= len(val.ArrVal) {
				return false
			}
			if !interp.matchPattern(val.ArrVal[i], item, bindings) {
				return false
			}
		}
		return true
	case "object":
		if val.Tag != TypeObject && val.Tag != TypeInstance {
			return false
		}
		for _, prop := range pat.Props {
			fieldVal := val.Get(prop.Key)
			bindings[prop.Alias] = fieldVal
		}
		return true
	case "enumVal":
		if val.Tag == TypeString && val.StrVal == pat.Path {
			return true
		}
		if val.Tag == TypeNumber {
			return false
		}
		return false
	default:
		return false
	}
}

func (interp *Interpreter) execVarDecl(node *ast.Node, env *Environment) (*Value, error) {
	var val *Value = Undefined
	if node.Init != nil {
		var err error
		val, err = interp.evalExpr(node.Init, env)
		if err != nil {
			return nil, err
		}
	}
	if node.Destructure != nil {
		return Undefined, interp.bindDestructure(node.Destructure, val, env)
	}
	env.Define(node.Name, val, node.IsConst)
	return Undefined, nil
}

func (interp *Interpreter) execFnDecl(node *ast.Node, env *Environment) (*Value, error) {
	MarkEscaped(env)
	fn := &Function{
		Name:   node.Name,
		Params: paramsToFnParams(node.Params),
		Body:   node.Body,
		Env:    env,
	}
	fnVal := FuncVal(fn)
	if node.Name != "" {
		env.Define(node.Name, fnVal, false)
	}
	return fnVal, nil
}

func (interp *Interpreter) execClassDecl(node *ast.Node, env *Environment) (*Value, error) {
	MarkEscaped(env)
	cls := &Class{
		Name:          node.Name,
		Methods:       make(map[string]*Function),
		StaticMethods: make(map[string]*Function),
		Env:           env,
	}
	if node.SuperClass != nil {
		superVal, err := interp.evalExpr(node.SuperClass, env)
		if err != nil {
			return nil, err
		}
		if superVal.Tag == TypeClass {
			cls.Super = superVal.ClsVal
		}
	}
	for _, member := range node.Methods {
		fn := &Function{
			Name:     member.Name,
			Params:   paramsToFnParams(member.Params),
			Body:     member.Body,
			Env:      env,
			DefClass: cls,
		}
		if member.Init != nil {
			fn.Body = member.Init
		}
		if member.IsStatic {
			cls.StaticMethods[member.Name] = fn
		} else {
			cls.Methods[member.Name] = fn
		}
	}
	clsVal := ClassVal(cls)
	if node.Name != "" {
		env.Define(node.Name, clsVal, false)
	}
	return clsVal, nil
}

func (interp *Interpreter) execEnumDecl(node *ast.Node, env *Environment) (*Value, error) {
	obj := make(map[string]*Value)
	for i, member := range node.Members {
		var val *Value
		if member.Init != nil {
			v, err := interp.evalExpr(member.Init, env)
			if err != nil {
				return nil, err
			}
			val = v
		} else {
			val = NumberVal(float64(i))
		}
		obj[member.Name] = val
	}
	enumVal := ObjectVal(obj)
	if node.Name != "" {
		env.Define(node.Name, enumVal, false)
	}
	return enumVal, nil
}

func (interp *Interpreter) execNamespace(node *ast.Node, env *Environment) (*Value, error) {
	nsEnv := NewEnvironment(env)
	for _, stmt := range node.Body_ {
		_, err := interp.execNode(stmt, nsEnv)
		if err != nil {
			return nil, err
		}
	}
	obj := make(map[string]*Value)
	for k, v := range nsEnv.vars {
		obj[k] = v
	}
	nsVal := ObjectVal(obj)
	if node.Name != "" {
		env.Define(node.Name, nsVal, false)
	}
	return nsVal, nil
}

func (interp *Interpreter) execImport(node *ast.Node, env *Environment) (*Value, error) {
	if node.TypeOnly {
		return Undefined, nil
	}
	modVal, err := interp.loadModule(node.Source)
	if err != nil {
		return nil, err
	}
	if node.Namespace != "" {
		env.Define(node.Namespace, modVal, true)
	} else if node.DefaultImport != "" && len(node.Specifiers) == 0 {
		env.Define(node.DefaultImport, modVal, true)
	} else {
		if node.DefaultImport != "" {
			def := modVal.Get("default")
			if def.IsNullish() {
				def = modVal
			}
			env.Define(node.DefaultImport, def, true)
		}
		for _, spec := range node.Specifiers {
			val := modVal.Get(spec.Imported)
			env.Define(spec.Local, val, true)
		}
	}
	return Undefined, nil
}

// resolveModulePath normalises module paths to their canonical name.
// Supports both dot notation ("std.io") and slash notation ("std/io").
// "std.io" -> "io", "std/io" -> "io", "internal.native" -> "native".
//
// Local file paths ("hello.lx", "./utils/math.lx") are returned unchanged
// so that the local-file resolution in loadModule can handle them.
func resolveModulePath(path string) string {
	// Preserve local file paths: anything with a .lx extension or a
	// relative/absolute prefix must not be dot-to-slash converted.
	if strings.HasSuffix(path, ".lx") ||
		strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../") ||
		strings.HasPrefix(path, "/") {
		return path
	}

	// Convert dot notation to slash notation for module names only.
	slashPath := strings.ReplaceAll(path, ".", "/")
	for _, prefix := range []string{"std/", "core/", "internal/"} {
		if strings.HasPrefix(slashPath, prefix) {
			rest := slashPath[len(prefix):]
			if rest != "" {
				return rest
			}
		}
	}
	return slashPath
}

func forceLocalImport(node *ast.Node) bool {
	if node == nil {
		return false
	}
	if s, ok := node.Prop.(string); ok {
		return strings.EqualFold(s, "force-local") || strings.EqualFold(s, "fimport")
	}
	return false
}

func (interp *Interpreter) loadModule(path string) (*Value, error) {
	resolved := resolveModulePath(path)

	interp.mu.RLock()
	if mod, ok := interp.modules[resolved]; ok {
		interp.mu.RUnlock()
		return mod, nil
	}
	interp.mu.RUnlock()

	// Try source loaders first (bundled archives, installed .lx packages).
	if interp.ntlLoader != nil {
		src, ok := interp.ntlLoader(resolved)
		if ok {
			return interp.evalModuleSourceFile(src, resolved, resolved)
		}
	}

	// Try resolving as a local file path. Handles .lx, .nax, and .nc.
	if localPath, ok := interp.resolveLocalFile(path); ok {
		abs, _ := filepath.Abs(localPath)
		// Deduplication check.
		interp.mu.RLock()
		if mod, ok := interp.modules[abs]; ok {
			interp.mu.RUnlock()
			return mod, nil
		}
		interp.mu.RUnlock()
		return interp.loadLocalFile(localPath, nil)
	}

	e := interp.runtimeError(errfmt.KindImport, "E0010",
		fmt.Sprintf("module %q not found", path), nil, nil)
	return nil, e
}

// resolveLocalFile tries to find a local source or binary file for the given import path.
// It checks, in order:
//  1. path as-is (absolute or already has a known extension)
//  2. path + each known extension: .lx, .nax, .nc
//  3. Same directory as the currently executing file
//  4. Current working directory
//
// Returns the resolved path and whether it was found.
func (interp *Interpreter) resolveLocalFile(path string) (string, bool) {
	var candidates []string

	// Build base names to try: the raw path plus fallback extensions.
	bases := []string{path}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".lx" && ext != ".nax" && ext != ".nc" {
		bases = append(bases, path+".lx", path+".nax", path+".nc")
	}

	// Relative to current file's directory.
	if interp.filename != "" {
		dir := filepath.Dir(interp.filename)
		for _, b := range bases {
			candidates = append(candidates,
				filepath.Join(dir, b),
				filepath.Join(dir, filepath.FromSlash(b)),
			)
		}
	}

	// Relative to working directory.
	wd, _ := os.Getwd()
	for _, b := range bases {
		candidates = append(candidates, filepath.Join(wd, b))
	}

	// Absolute path fall-through.
	candidates = append(candidates, bases...)

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, true
		}
	}
	return "", false
}

// evalModuleSourceFile compiles and runs a local .lx file as a module.
// cacheKey is the absolute path used for deduplication.
// displayPath is used in error messages.
func (interp *Interpreter) evalModuleSourceFile(src, cacheKey, displayPath string) (*Value, error) {
	// Temporarily set filename so nested @imports inside the module resolve
	// relative to the module's own directory.
	prevFilename := interp.filename
	interp.filename = cacheKey
	defer func() { interp.filename = prevFilename }()

	return interp.evalModuleSource(src, cacheKey)
}

func (interp *Interpreter) evalModuleSource(src, name string) (*Value, error) {
	// Use name as-is for display if it already ends with .lx (local file path),
	// otherwise append .lx for stdlib module names.
	displayName := name
	if !strings.HasSuffix(name, ".lx") {
		displayName = name + ".lx"
	}
	lines := strings.Split(src, "\n")
	toks, err := lexer.Tokenize(src, displayName)
	if err != nil {
		return nil, interp.runtimeError(errfmt.KindImport, "E0011",
			fmt.Sprintf("failed to tokenize module '%s': %v", name, err), nil, nil)
	}
	prog, err := parser.ParseWithLines(toks, displayName, lines)
	if err != nil {
		// E0011 = module parse/tokenize failed. E0012 is reserved for circular imports.
		return nil, interp.runtimeError(errfmt.KindImport, "E0011",
			fmt.Sprintf("failed to parse module '%s': %v", name, err), nil, nil)
	}
	interp.libLoadDepth++
	modEnv := NewEnvironment(interp.globals)
	_, execErr := interp.execBlock(prog.Body_, modEnv)
	interp.libLoadDepth--
	if execErr != nil {
		if _, ok := execErr.(*returnError); !ok {
			return nil, interp.runtimeError(errfmt.KindImport, "E0013",
				fmt.Sprintf("error while executing module '%s': %v", name, execErr), nil, nil)
		}
	}
	mod, ok := modEnv.vars["__module__"]
	if ok {
		fmt.Fprintln(os.Stderr, "warning [deprecated]: val __module__ = {} is no longer needed — all public bindings are exported automatically. Remove it.")
	} else {
		exports := make(map[string]*Value)
		for k, v := range modEnv.vars {
			if len(k) == 0 || k[0] == '_' {
				continue
			}
			exports[k] = v
		}
		mod = ObjectVal(exports)
	}
	interp.mu.Lock()
	interp.modules[name] = mod
	interp.mu.Unlock()
	return mod, nil
}

func (interp *Interpreter) execExport(node *ast.Node, env *Environment) (*Value, error) {
	if node.Declaration != nil {
		return interp.execNode(node.Declaration, env)
	}
	return Undefined, nil
}

func (interp *Interpreter) execUse(node *ast.Node, env *Environment) (*Value, error) {
	// 'use' has been removed from Lunex — this path should only be reached by
	// compiled bytecode from an older version; give a clear diagnostic.
	modName := ""
	if len(node.Modules) > 0 {
		modName = node.Modules[0]
	}
	suggestion := "std." + modName
	if modName == "native" {
		suggestion = "internal.native"
	}
	return nil, interp.runtimeError(errfmt.KindImport, "E0014",
		fmt.Sprintf("'use %s' is no longer valid — replace with: val %s = @import(%q)", modName, modName, suggestion), node, nil)
}

func (interp *Interpreter) execLunexRequire(node *ast.Node, env *Environment) (*Value, error) {
	for _, mod := range node.Modules {
		modVal, err := interp.loadModule(mod)
		if err != nil {
			return nil, err
		}
		env.Define(mod, modVal, true)
	}
	return Undefined, nil
}

func (interp *Interpreter) execImmutable(node *ast.Node, env *Environment) (*Value, error) {
	return interp.execNode(node.Body, env)
}

func (interp *Interpreter) execUsing(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Init, env)
	if err != nil {
		return nil, err
	}
	env.Define(node.Name, val, false)
	return Undefined, nil
}

func (interp *Interpreter) execLog(node *ast.Node, env *Environment) (*Value, error) {
	var parts []string
	for _, arg := range node.Args {
		val, err := interp.evalExpr(arg, env)
		if err != nil {
			parts = append(parts, fmt.Sprintf("<error: %v>", err))
		} else {
			parts = append(parts, val.Inspect())
		}
	}
	fmt.Println(strings.Join(parts, " "))
	return Undefined, nil
}

func (interp *Interpreter) execIf(node *ast.Node, env *Environment) (*Value, error) {
	test, err := interp.evalExpr(node.Test, env)
	if err != nil {
		return nil, err
	}
	if test.IsTruthy() {
		return interp.execNode(node.Consequent, env)
	}
	if node.Alternate != nil {
		return interp.execNode(node.Alternate, env)
	}
	return Undefined, nil
}

func (interp *Interpreter) execUnless(node *ast.Node, env *Environment) (*Value, error) {
	test, err := interp.evalExpr(node.Test, env)
	if err != nil {
		return nil, err
	}
	if !test.IsTruthy() {
		return interp.execNode(node.Consequent, env)
	}
	if node.Alternate != nil {
		return interp.execNode(node.Alternate, env)
	}
	return Undefined, nil
}

func (interp *Interpreter) execWhile(node *ast.Node, env *Environment) (*Value, error) {
	for {
		test, err := interp.evalExpr(node.Test, env)
		if err != nil {
			return nil, err
		}
		if !test.IsTruthy() {
			break
		}
		_, err = interp.execNode(node.Body, env)
		if err != nil {
			if _, ok := err.(*breakError); ok {
				break
			}
			if _, ok := err.(*continueError); ok {
				continue
			}
			return nil, err
		}
	}
	return Undefined, nil
}

func (interp *Interpreter) execForOf(node *ast.Node, env *Environment) (*Value, error) {
	iterVal, err := interp.evalExpr(node.Right, env)
	if err != nil {
		return nil, err
	}
	idx := 0
	iterLoop := func(val *Value) error {
		iterEnv := NewEnvironment(env)
		if node.Destructure != nil {
			if err := interp.bindDestructure(node.Destructure, val, iterEnv); err != nil {
				return err
			}
		} else {
			iterEnv.Define(node.Name, val, node.IsConst)
		}
		if node.Alias != "" {
			iterEnv.Define(node.Alias, NumberVal(float64(idx)), node.IsConst)
		}
		idx++
		_, err := interp.execNode(node.Body, iterEnv)
		return err
	}
	if v, err := interp.CheckForOfIterable(iterVal, node); err != nil {
		errfmt.Print(err.(*errfmt.LunexError))
		return v, nil
	}
	switch iterVal.Tag {
	case TypeArray:
		for _, el := range iterVal.ArrVal {
			if el == nil {
				el = Undefined
			}
			if err := iterLoop(el); err != nil {
				if _, ok := err.(*breakError); ok {
					return Undefined, nil
				}
				if _, ok := err.(*continueError); ok {
					continue
				}
				return nil, err
			}
		}
	case TypeString:
		for _, r := range iterVal.StrVal {
			if err := iterLoop(StringVal(string(r))); err != nil {
				if _, ok := err.(*breakError); ok {
					return Undefined, nil
				}
				if _, ok := err.(*continueError); ok {
					continue
				}
				return nil, err
			}
		}
	case TypeObject:
		for k := range iterVal.ObjVal {
			if err := iterLoop(StringVal(k)); err != nil {
				if _, ok := err.(*breakError); ok {
					return Undefined, nil
				}
				if _, ok := err.(*continueError); ok {
					continue
				}
				return nil, err
			}
		}
	}
	return Undefined, nil
}

func (interp *Interpreter) execFor(node *ast.Node, env *Environment) (*Value, error) {
	if node.Body == nil && node.Init == nil {
		return Undefined, nil
	}

	forEnv := NewEnvironment(env)

	if node.Init != nil {
		if _, err := interp.execNode(node.Init, forEnv); err != nil {
			return nil, err
		}
	}

	for {
		if node.Test != nil {
			test, err := interp.evalExpr(node.Test, forEnv)
			if err != nil {
				return nil, err
			}
			if !test.IsTruthy() {
				break
			}
		}

		if node.Body != nil {
			if _, err := interp.execNode(node.Body, forEnv); err != nil {
				if _, ok := err.(*breakError); ok {
					break
				}
				if _, ok := err.(*continueError); ok {
				} else {
					return nil, err
				}
			}
		}

		if node.Right != nil {
			if _, err := interp.evalExpr(node.Right, forEnv); err != nil {
				return nil, err
			}
		}
	}
	return Undefined, nil
}

func (interp *Interpreter) execRepeat(node *ast.Node, env *Environment) (*Value, error) {
	count := -1
	if node.Count != nil {
		n, err := interp.evalExpr(node.Count, env)
		if err != nil {
			return nil, err
		}
		count = int(n.ToNumber())
	}
	for i := 0; count < 0 || i < count; i++ {
		_, err := interp.execNode(node.Body, env)
		if err != nil {
			if _, ok := err.(*breakError); ok {
				break
			}
			if _, ok := err.(*continueError); ok {
				continue
			}
			return nil, err
		}
	}
	return Undefined, nil
}

func (interp *Interpreter) execLoop(node *ast.Node, env *Environment) (*Value, error) {
	for {
		_, err := interp.execNode(node.Body, env)
		if err != nil {
			if _, ok := err.(*breakError); ok {
				break
			}
			if _, ok := err.(*continueError); ok {
				continue
			}
			return nil, err
		}
	}
	return Undefined, nil
}

func (interp *Interpreter) execMatch(node *ast.Node, env *Environment) (*Value, error) {
	return interp.evalMatchExpr(node, env)
}

func (interp *Interpreter) execTry(node *ast.Node, env *Environment) (*Value, error) {
	tryResult, err := interp.execNode(node.Body, env)
	if tryResult == nil {
		tryResult = Undefined
	}
	result := tryResult
	if err != nil {
		if te, ok := err.(*throwError); ok {
			if node.CatchBlock != nil {
				catchEnv := NewEnvironment(env)
				if node.CatchParam != "" {
					catchEnv.Define(node.CatchParam, te.val, false)
				}
				catchResult, catchErr := interp.execNode(node.CatchBlock, catchEnv)
				if catchErr != nil {
					// Propagate the catch error even if there is a finally block.
					if node.FinallyBlock != nil {
						interp.execNode(node.FinallyBlock, env)
					}
					return nil, catchErr
				}
				if catchResult != nil {
					result = catchResult
				}
			}
		} else if re, ok := err.(*returnError); ok {
			if node.FinallyBlock != nil {
				interp.execNode(node.FinallyBlock, env)
			}
			return nil, re
		} else {
			if node.CatchBlock != nil {
				catchEnv := NewEnvironment(env)
				if node.CatchParam != "" {
					errMsg := err.Error()
					errObj := ObjectVal(map[string]*Value{
						"message": StringVal(errMsg),
						"name":    StringVal("Error"),
						"stack":   StringVal("Error: " + errMsg),
					})
					catchEnv.Define(node.CatchParam, errObj, false)
				}
				catchResult, _ := interp.execNode(node.CatchBlock, catchEnv)
				if catchResult != nil {
					result = catchResult
				}
			}
		}
	}
	if node.FinallyBlock != nil {
		interp.execNode(node.FinallyBlock, env)
	}
	return result, nil
}

func (interp *Interpreter) execGuard(node *ast.Node, env *Environment) (*Value, error) {
	test, err := interp.evalExpr(node.Test, env)
	if err != nil {
		return nil, err
	}
	if !test.IsTruthy() {
		return interp.execNode(node.Alternate, env)
	}
	return Undefined, nil
}

func (interp *Interpreter) execAssert(node *ast.Node, env *Environment) (*Value, error) {
	test, err := interp.evalExpr(node.Test, env)
	if err != nil {
		return nil, err
	}
	if !test.IsTruthy() {
		msg := "Assertion failed"
		if node.Arg != nil {
			msgVal, err := interp.evalExpr(node.Arg, env)
			if err == nil {
				msg = msgVal.ToString()
			}
		}
		return nil, &throwError{val: ObjectVal(map[string]*Value{
			"message": StringVal(msg),
		})}
	}
	return Undefined, nil
}

func (interp *Interpreter) execHave(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return nil, err
	}
	cond := interp.testHaveCondition(val, node, env)
	if node.IsGuard {
		if !cond {
			if node.Alternate != nil {
				return interp.execNode(node.Alternate, env)
			}
			return nil, &returnError{val: Undefined}
		}
		return Undefined, nil
	}
	haveEnv := NewEnvironment(env)
	if node.Alias != "" {
		haveEnv.Define(node.Alias, val, false)
	}
	if cond {
		if node.Consequent != nil {
			return interp.execNode(node.Consequent, haveEnv)
		}
	} else {
		if node.Alternate != nil {
			return interp.execNode(node.Alternate, env)
		}
	}
	return Undefined, nil
}

func (interp *Interpreter) execIfHave(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return nil, err
	}
	cond := interp.testHaveCondition(val, node, env)
	ifEnv := NewEnvironment(env)
	if node.Alias != "" {
		ifEnv.Define(node.Alias, val, false)
	}
	if cond {
		return interp.execNode(node.Consequent, ifEnv)
	}
	if node.Alternate != nil {
		return interp.execNode(node.Alternate, env)
	}
	return Undefined, nil
}

func (interp *Interpreter) execIfSet(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return nil, err
	}
	ifEnv := NewEnvironment(env)
	alias := node.Alias
	if alias == "" {
		alias = fmt.Sprintf("_ifset_%d", node.ID)
	}
	ifEnv.Define(alias, val, false)
	if !val.IsNullish() {
		return interp.execNode(node.Consequent, ifEnv)
	}
	if node.Alternate != nil {
		return interp.execNode(node.Alternate, env)
	}
	return Undefined, nil
}

func (interp *Interpreter) execDelete(node *ast.Node, env *Environment) (*Value, error) {
	if node.Expr.Type == ast.MemberExpr {
		obj, err := interp.evalExpr(node.Expr.Object, env)
		if err != nil {
			return nil, err
		}
		var key string
		if node.Expr.Computed {
			k, err := interp.evalExpr(node.Expr.Prop.(*ast.Node), env)
			if err != nil {
				return nil, err
			}
			key = k.ToString()
		} else {
			key, _ = node.Expr.Prop.(string)
		}
		if obj.Tag == TypeObject {
			delete(obj.ObjVal, key)
		}
	}
	return True, nil
}

func (interp *Interpreter) execWith(node *ast.Node, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(node.Expr, env)
	if err != nil {
		return nil, err
	}
	withEnv := NewEnvironment(env)
	if val.Tag == TypeObject {
		for k, v := range val.ObjVal {
			withEnv.Define(k, v, false)
		}
	}
	return interp.execNode(node.Body, withEnv)
}

func (interp *Interpreter) execComponent(node *ast.Node, env *Environment) (*Value, error) {
	fn := &Function{
		Name:   node.Name,
		Params: paramsToFnParams(node.Params),
		Body:   node.Body,
		Env:    env,
	}
	fnVal := FuncVal(fn)
	if node.Name != "" {
		env.Define(node.Name, fnVal, false)
	}
	return fnVal, nil
}

func (interp *Interpreter) execSelect(node *ast.Node, env *Environment) (*Value, error) {
	type result struct {
		idx int
		val *Value
	}
	ch := make(chan result, len(node.SelectCases))
	for i, sc := range node.SelectCases {
		i, sc := i, sc
		go func() {
			chanVal, err := interp.evalExpr(sc.Channel, env)
			if err != nil {
				ch <- result{idx: i, val: Null}
				return
			}
			var val *Value
			if chanVal.Tag == TypeChannel {
				val = chanVal.ChanVal.Receive()
			} else {
				val = chanVal
			}
			ch <- result{idx: i, val: val}
		}()
	}
	r := <-ch
	if r.idx < len(node.SelectCases) {
		sc := node.SelectCases[r.idx]
		caseEnv := NewEnvironment(env)
		if sc.Binding != "" {
			caseEnv.Define(sc.Binding, r.val, false)
		}
		interp.execNode(sc.Body, caseEnv)
	}
	return Undefined, nil
}

func (interp *Interpreter) registerBuiltins() {
	g := interp.globals

	g.Define("undefined", Undefined, false)
	g.Define("null", Null, false)
	g.Define("true", True, false)
	g.Define("false", False, false)
	g.Define("NaN", NumberVal(math.NaN()), false)
	g.Define("Infinity", NumberVal(math.Inf(1)), false)

	g.Define("lunex", ObjectVal(map[string]*Value{
		"getversion": FuncVal(&Function{Name: "getversion", Native: func(args []*Value, this *Value) (*Value, error) {
			return StringVal(meta.Version()), nil
		}}),
	}), false)

	g.Define("parseInt", FuncVal(&Function{Name: "parseInt", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return NumberVal(math.NaN()), nil
		}
		base := 10
		if len(args) > 1 {
			base = int(args[1].ToNumber())
		}
		s := strings.TrimSpace(args[0].ToString())
		n, err := strconv.ParseInt(s, base, 64)
		if err != nil {
			return NumberVal(math.NaN()), nil
		}
		return NumberVal(float64(n)), nil
	}}), false)

	g.Define("parseFloat", FuncVal(&Function{Name: "parseFloat", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return NumberVal(math.NaN()), nil
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(args[0].ToString()), 64)
		if err != nil {
			return NumberVal(math.NaN()), nil
		}
		return NumberVal(f), nil
	}}), false)

	g.Define("isNaN", FuncVal(&Function{Name: "isNaN", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return True, nil
		}
		return BoolVal(math.IsNaN(args[0].ToNumber())), nil
	}}), false)

	g.Define("isFinite", FuncVal(&Function{Name: "isFinite", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return False, nil
		}
		n := args[0].ToNumber()
		return BoolVal(!math.IsNaN(n) && !math.IsInf(n, 0)), nil
	}}), false)

	g.Define("String", FuncVal(&Function{Name: "String", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		return StringVal(args[0].ToString()), nil
	}}), false)

	g.Define("str", FuncVal(&Function{Name: "str", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		return StringVal(args[0].ToString()), nil
	}}), false)

	g.Define("Number", FuncVal(&Function{Name: "Number", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return NumberVal(0), nil
		}
		return NumberVal(args[0].ToNumber()), nil
	}}), false)

	g.Define("num", FuncVal(&Function{Name: "num", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return NumberVal(0), nil
		}
		return NumberVal(args[0].ToNumber()), nil
	}}), false)

	g.Define("Boolean", FuncVal(&Function{Name: "Boolean", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return False, nil
		}
		return BoolVal(args[0].IsTruthy()), nil
	}}), false)

	g.Define("print", FuncVal(&Function{Name: "print", Native: func(args []*Value, this *Value) (*Value, error) {
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = a.ToString()
		}
		fmt.Println(strings.Join(parts, " "))
		return Null, nil
	}}), false)

	g.Define("log", FuncVal(&Function{Name: "log", Native: func(args []*Value, this *Value) (*Value, error) {
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = a.ToString()
		}
		fmt.Println(strings.Join(parts, " "))
		return Null, nil
	}}), false)

	g.Define("Array", ObjectVal(map[string]*Value{
		"isArray": FuncVal(&Function{Name: "isArray", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return False, nil
			}
			return BoolVal(args[0].Tag == TypeArray), nil
		}}),
		"from": FuncVal(&Function{Name: "from", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ArrayVal(nil), nil
			}
			src := args[0]
			if src.Tag == TypeArray {
				result := make([]*Value, len(src.ArrVal))
				copy(result, src.ArrVal)
				return ArrayVal(result), nil
			}
			if src.Tag == TypeString {
				runes := []rune(src.StrVal)
				result := make([]*Value, len(runes))
				for i, r := range runes {
					result[i] = StringVal(string(r))
				}
				return ArrayVal(result), nil
			}
			return ArrayVal(nil), nil
		}}),
		"of": FuncVal(&Function{Name: "of", Native: func(args []*Value, this *Value) (*Value, error) {
			return ArrayVal(args), nil
		}}),
	}), false)

	g.Define("Object", ObjectVal(map[string]*Value{
		"keys": FuncVal(&Function{Name: "keys", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ArrayVal(nil), nil
			}
			obj := args[0]
			var keys []*Value
			if obj.Tag == TypeObject {
				for k := range obj.ObjVal {
					keys = append(keys, StringVal(k))
				}
			} else if obj.Tag == TypeInstance {
				for k := range obj.InstVal.Fields {
					keys = append(keys, StringVal(k))
				}
			}
			return ArrayVal(keys), nil
		}}),
		"values": FuncVal(&Function{Name: "values", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ArrayVal(nil), nil
			}
			obj := args[0]
			var vals []*Value
			if obj.Tag == TypeObject {
				for _, v := range obj.ObjVal {
					vals = append(vals, v)
				}
			}
			return ArrayVal(vals), nil
		}}),
		"entries": FuncVal(&Function{Name: "entries", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ArrayVal(nil), nil
			}
			obj := args[0]
			var entries []*Value
			if obj.Tag == TypeObject {
				for k, v := range obj.ObjVal {
					entries = append(entries, ArrayVal([]*Value{StringVal(k), v}))
				}
			}
			return ArrayVal(entries), nil
		}}),
		"assign": FuncVal(&Function{Name: "assign", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ObjectVal(nil), nil
			}
			target := args[0]
			if target.Tag != TypeObject {
				return target, nil
			}
			for _, src := range args[1:] {
				if src.Tag == TypeObject {
					for k, v := range src.ObjVal {
						target.ObjVal[k] = v
					}
				}
			}
			return target, nil
		}}),
		"freeze": FuncVal(&Function{Name: "freeze", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Undefined, nil
			}
			return args[0], nil
		}}),
		"create": FuncVal(&Function{Name: "create", Native: func(args []*Value, this *Value) (*Value, error) {
			obj := ObjectVal(nil)
			if len(args) > 0 && args[0].Tag == TypeObject {
				for k, v := range args[0].ObjVal {
					obj.ObjVal[k] = v
				}
			}
			return obj, nil
		}}),
		"fromEntries": FuncVal(&Function{Name: "fromEntries", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ObjectVal(nil), nil
			}
			obj := make(map[string]*Value)
			if args[0].Tag == TypeArray {
				for _, entry := range args[0].ArrVal {
					if entry != nil && entry.Tag == TypeArray && len(entry.ArrVal) >= 2 {
						key := entry.ArrVal[0].ToString()
						obj[key] = entry.ArrVal[1]
					}
				}
			}
			return ObjectVal(obj), nil
		}}),
	}), false)

	g.Define("Math", ObjectVal(map[string]*Value{
		"PI":     NumberVal(math.Pi),
		"E":      NumberVal(math.E),
		"LN2":    NumberVal(math.Ln2),
		"LN10":   NumberVal(math.Log(10)),
		"LOG2E":  NumberVal(math.Log2E),
		"LOG10E": NumberVal(math.Log10E),
		"SQRT2":  NumberVal(math.Sqrt2),
		"abs":    mathFn1("abs", math.Abs),
		"ceil":   mathFn1("ceil", math.Ceil),
		"floor":  mathFn1("floor", math.Floor),
		"round":  mathFn1("round", math.Round),
		"sqrt":   mathFn1("sqrt", math.Sqrt),
		"cbrt":   mathFn1("cbrt", math.Cbrt),
		"sin":    mathFn1("sin", math.Sin),
		"cos":    mathFn1("cos", math.Cos),
		"tan":    mathFn1("tan", math.Tan),
		"asin":   mathFn1("asin", math.Asin),
		"acos":   mathFn1("acos", math.Acos),
		"atan":   mathFn1("atan", math.Atan),
		"log":    mathFn1("log", math.Log),
		"log2":   mathFn1("log2", math.Log2),
		"log10":  mathFn1("log10", math.Log10),
		"exp":    mathFn1("exp", math.Exp),
		"sign":   mathFn1("sign", mathSign),
		"trunc":  mathFn1("trunc", math.Trunc),
		"hypot":  mathFn1("hypot", math.Abs),
		"max": FuncVal(&Function{Name: "max", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return NumberVal(math.Inf(-1)), nil
			}
			max := args[0].ToNumber()
			for _, a := range args[1:] {
				if n := a.ToNumber(); n > max {
					max = n
				}
			}
			return NumberVal(max), nil
		}}),
		"min": FuncVal(&Function{Name: "min", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return NumberVal(math.Inf(1)), nil
			}
			min := args[0].ToNumber()
			for _, a := range args[1:] {
				if n := a.ToNumber(); n < min {
					min = n
				}
			}
			return NumberVal(min), nil
		}}),
		"pow": FuncVal(&Function{Name: "pow", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) < 2 {
				return NumberVal(math.NaN()), nil
			}
			return NumberVal(math.Pow(args[0].ToNumber(), args[1].ToNumber())), nil
		}}),
		"atan2": FuncVal(&Function{Name: "atan2", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) < 2 {
				return NumberVal(math.NaN()), nil
			}
			return NumberVal(math.Atan2(args[0].ToNumber(), args[1].ToNumber())), nil
		}}),
		"random": FuncVal(&Function{Name: "random", Native: func(args []*Value, this *Value) (*Value, error) {
			return NumberVal(pseudoRandom()), nil
		}}),
		"imul": FuncVal(&Function{Name: "imul", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) < 2 {
				return NumberVal(0), nil
			}
			return NumberVal(float64(int32(args[0].ToNumber()) * int32(args[1].ToNumber()))), nil
		}}),
	}), false)

	g.Define("JSON", ObjectVal(map[string]*Value{
		"stringify": FuncVal(&Function{Name: "stringify", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Undefined, nil
			}
			indent := ""
			if len(args) > 2 {
				if args[2].Tag == TypeNumber {
					indent = strings.Repeat(" ", int(args[2].ToNumber()))
				} else if args[2].Tag == TypeString {
					indent = args[2].StrVal
				}
			}
			result := jsonStringify(args[0], indent, 0)
			return StringVal(result), nil
		}}),
		"parse": FuncVal(&Function{Name: "parse", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Null, nil
			}
			val, err := jsonParse(args[0].ToString())
			if err != nil {
				return nil, &throwError{val: ObjectVal(map[string]*Value{"message": StringVal(err.Error())})}
			}
			return val, nil
		}}),
	}), false)

	g.Define("Promise", ObjectVal(map[string]*Value{
		"resolve": FuncVal(&Function{Name: "resolve", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Undefined, nil
			}
			return args[0], nil
		}}),
		"reject": FuncVal(&Function{Name: "reject", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Null, nil
			}
			return nil, &throwError{val: args[0]}
		}}),
		"all": FuncVal(&Function{Name: "all", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return ArrayVal(nil), nil
			}
			arr := args[0]
			if arr.Tag != TypeArray {
				return ArrayVal(nil), nil
			}
			return arr, nil
		}}),
	}), false)

	g.Define("Error", FuncVal(&Function{Name: "Error", Native: func(args []*Value, this *Value) (*Value, error) {
		msg := ""
		if len(args) > 0 {
			msg = args[0].ToString()
		}
		return ObjectVal(map[string]*Value{
			"message": StringVal(msg),
			"name":    StringVal("Error"),
			"stack":   StringVal("Error: " + msg),
		}), nil
	}}), false)

	g.Define("TypeError", FuncVal(&Function{Name: "TypeError", Native: func(args []*Value, this *Value) (*Value, error) {
		msg := ""
		if len(args) > 0 {
			msg = args[0].ToString()
		}
		return ObjectVal(map[string]*Value{
			"message": StringVal(msg),
			"name":    StringVal("TypeError"),
		}), nil
	}}), false)

	g.Define("RangeError", FuncVal(&Function{Name: "RangeError", Native: func(args []*Value, this *Value) (*Value, error) {
		msg := ""
		if len(args) > 0 {
			msg = args[0].ToString()
		}
		return ObjectVal(map[string]*Value{
			"message": StringVal(msg),
			"name":    StringVal("RangeError"),
		}), nil
	}}), false)

	g.Define("Map", FuncVal(&Function{Name: "Map", Native: func(args []*Value, this *Value) (*Value, error) {
		m := &ntlMap{data: make(map[string]*Value), keyOrder: nil}
		if this != nil && this.Tag == TypeInstance {
			this.InstVal.Fields["__map__"] = ObjectVal(nil)
			this.InstVal.Fields["__map__"].ObjVal["_m"] = FuncVal(&Function{Native: func(a []*Value, t *Value) (*Value, error) {
				return ObjectVal(m.data), nil
			}})
		}
		return mapObject(m), nil
	}}), false)

	g.Define("Set", FuncVal(&Function{Name: "Set", Native: func(args []*Value, this *Value) (*Value, error) {
		s := &ntlSet{items: make(map[string]*Value)}
		if len(args) > 0 && args[0].Tag == TypeArray {
			for _, item := range args[0].ArrVal {
				if item != nil {
					s.items[item.ToString()] = item
				}
			}
		}
		return setObject(s), nil
	}}), false)

	g.Define("setTimeout", FuncVal(&Function{Name: "setTimeout", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) < 2 {
			return NumberVal(0), nil
		}
		fn := args[0]
		ms := int(args[1].ToNumber())
		if ms < 0 {
			ms = 0
		}
		go func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			interp.callFunctionValue(fn, nil, nil)
		}()
		return NumberVal(0), nil
	}}), false)

	var intervalMu sync.Mutex
	intervalMap := make(map[float64]*time.Ticker)
	var intervalIDCounter float64

	g.Define("setInterval", FuncVal(&Function{Name: "setInterval", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) < 2 {
			return NumberVal(0), nil
		}
		fn := args[0]
		ms := int(args[1].ToNumber())
		if ms < 1 {
			ms = 1
		}
		ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
		intervalMu.Lock()
		intervalIDCounter++
		id := intervalIDCounter
		intervalMap[id] = ticker
		intervalMu.Unlock()
		go func() {
			for range ticker.C {
				interp.callFunctionValue(fn, nil, nil)
			}
		}()
		return NumberVal(id), nil
	}}), false)

	g.Define("clearTimeout", FuncVal(&Function{Name: "clearTimeout", Native: func(args []*Value, this *Value) (*Value, error) {
		return Undefined, nil
	}}), false)

	g.Define("clearInterval", FuncVal(&Function{Name: "clearInterval", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return Undefined, nil
		}
		id := args[0].ToNumber()
		intervalMu.Lock()
		if ticker, ok := intervalMap[id]; ok {
			ticker.Stop()
			delete(intervalMap, id)
		}
		intervalMu.Unlock()
		return Undefined, nil
	}}), false)

	g.Define("performance", ObjectVal(map[string]*Value{
		"now": FuncVal(&Function{Name: "now", Native: func(args []*Value, this *Value) (*Value, error) {
			return NumberVal(float64(time.Now().UnixNano()) / 1e6), nil
		}}),
	}), false)

	g.Define("process", ObjectVal(map[string]*Value{
		"env":  ObjectVal(nil),
		"argv": ArrayVal(nil),
		"exit": FuncVal(&Function{Name: "exit", Native: func(args []*Value, this *Value) (*Value, error) {
			code := 0
			if len(args) > 0 {
				code = int(args[0].ToNumber())
			}
			os.Exit(code)
			return Undefined, nil
		}}),
		"stdout": ObjectVal(map[string]*Value{
			"write": FuncVal(&Function{Name: "write", Native: func(args []*Value, this *Value) (*Value, error) {
				if len(args) > 0 {
					fmt.Print(args[0].ToString())
				}
				return Undefined, nil
			}}),
		}),
		"stderr": ObjectVal(map[string]*Value{
			"write": FuncVal(&Function{Name: "write", Native: func(args []*Value, this *Value) (*Value, error) {
				if len(args) > 0 {
					fmt.Fprint(os.Stderr, args[0].ToString())
				}
				return Undefined, nil
			}}),
		}),
	}), false)

	g.Define("encodeURIComponent", FuncVal(&Function{Name: "encodeURIComponent", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal("undefined"), nil
		}
		return StringVal(encodeURIComponent(args[0].ToString())), nil
	}}), false)

	g.Define("decodeURIComponent", FuncVal(&Function{Name: "decodeURIComponent", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		result, err := decodeURIComponent(args[0].ToString())
		if err != nil {
			return StringVal(args[0].ToString()), nil
		}
		return StringVal(result), nil
	}}), false)

	g.Define("encodeURI", FuncVal(&Function{Name: "encodeURI", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal("undefined"), nil
		}
		return StringVal(encodeURI(args[0].ToString())), nil
	}}), false)

	g.Define("decodeURI", FuncVal(&Function{Name: "decodeURI", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		result, _ := decodeURIComponent(args[0].ToString())
		return StringVal(result), nil
	}}), false)

	g.Define("btoa", FuncVal(&Function{Name: "btoa", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		encoded := base64Encode([]byte(args[0].ToString()))
		return StringVal(encoded), nil
	}}), false)

	g.Define("atob", FuncVal(&Function{Name: "atob", Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return StringVal(""), nil
		}
		decoded, err := base64Decode(args[0].ToString())
		if err != nil {
			return StringVal(""), nil
		}
		return StringVal(string(decoded)), nil
	}}), false)
}

type returnError struct{ val *Value }
type throwError struct{ val *Value }
type breakError struct{}
type continueError struct{}

func (e *returnError) Error() string { return "return" }
func (e *throwError) Error() string {
	if e.val != nil {
		if e.val.Tag == TypeObject {
			if msg, ok := e.val.ObjVal["message"]; ok {
				return msg.ToString()
			}
		}
		return e.val.ToString()
	}
	return "thrown"
}
func (e *breakError) Error() string    { return "break" }
func (e *continueError) Error() string { return "continue" }

func paramsToFnParams(params []*ast.Param) []FnParam {
	result := make([]FnParam, len(params))
	for i, p := range params {
		result[i] = FnParam{
			Name:        p.Name,
			Default:     p.DefaultVal,
			Rest:        p.Rest,
			Destructure: p.Destructure,
		}
	}
	return result
}

func isInstanceOf(inst *Instance, cls *Class) bool {
	if inst.Class == cls {
		return true
	}
	if inst.Class != nil && inst.Class.Super != nil {
		return isInstanceOf(&Instance{Class: inst.Class.Super}, cls)
	}
	return false
}

func buildArgSig(args []*Value) string {
	types := make([]string, len(args))
	for i, a := range args {
		types[i] = a.TypeName()
	}
	return strings.Join(types, ",")
}

func mathFn1(name string, fn func(float64) float64) *Value {
	return FuncVal(&Function{Name: name, Native: func(args []*Value, this *Value) (*Value, error) {
		if len(args) == 0 {
			return NumberVal(math.NaN()), nil
		}
		return NumberVal(fn(args[0].ToNumber())), nil
	}})
}

func mathSign(x float64) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

var randState = uint64(time.Now().UnixNano()) | 1 // seeded from wall clock; never zero

func pseudoRandom() float64 {
	randState ^= randState << 13
	randState ^= randState >> 7
	randState ^= randState << 17
	return float64(randState&0x7FFFFFFFFFFFFFFF) / float64(0x7FFFFFFFFFFFFFFF)
}

type ntlMap struct {
	data     map[string]*Value
	keyOrder []string
	mu       sync.RWMutex
}

type ntlSet struct {
	items map[string]*Value
	mu    sync.RWMutex
}

func mapObject(m *ntlMap) *Value {
	obj := ObjectVal(map[string]*Value{
		"size": NumberVal(0),
		"set": FuncVal(&Function{Name: "set", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) < 2 {
				return this, nil
			}
			key := args[0].ToString()
			m.mu.Lock()
			if _, exists := m.data[key]; !exists {
				m.keyOrder = append(m.keyOrder, key)
			}
			m.data[key] = args[1]
			m.mu.Unlock()
			this.ObjVal["size"] = NumberVal(float64(len(m.data)))
			return this, nil
		}}),
		"get": FuncVal(&Function{Name: "get", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Undefined, nil
			}
			m.mu.RLock()
			val, ok := m.data[args[0].ToString()]
			m.mu.RUnlock()
			if !ok {
				return Undefined, nil
			}
			return val, nil
		}}),
		"has": FuncVal(&Function{Name: "has", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return False, nil
			}
			m.mu.RLock()
			_, ok := m.data[args[0].ToString()]
			m.mu.RUnlock()
			return BoolVal(ok), nil
		}}),
		"delete": FuncVal(&Function{Name: "delete", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return False, nil
			}
			key := args[0].ToString()
			m.mu.Lock()
			_, ok := m.data[key]
			delete(m.data, key)
			m.mu.Unlock()
			if ok {
				this.ObjVal["size"] = NumberVal(float64(len(m.data)))
			}
			return BoolVal(ok), nil
		}}),
		"clear": FuncVal(&Function{Name: "clear", Native: func(args []*Value, this *Value) (*Value, error) {
			m.mu.Lock()
			m.data = make(map[string]*Value)
			m.keyOrder = nil
			m.mu.Unlock()
			this.ObjVal["size"] = NumberVal(0)
			return Undefined, nil
		}}),
		"keys": FuncVal(&Function{Name: "keys", Native: func(args []*Value, this *Value) (*Value, error) {
			m.mu.RLock()
			result := make([]*Value, len(m.keyOrder))
			for i, k := range m.keyOrder {
				result[i] = StringVal(k)
			}
			m.mu.RUnlock()
			return ArrayVal(result), nil
		}}),
		"values": FuncVal(&Function{Name: "values", Native: func(args []*Value, this *Value) (*Value, error) {
			m.mu.RLock()
			result := make([]*Value, 0, len(m.data))
			for _, k := range m.keyOrder {
				if v, ok := m.data[k]; ok {
					result = append(result, v)
				}
			}
			m.mu.RUnlock()
			return ArrayVal(result), nil
		}}),
		"entries": FuncVal(&Function{Name: "entries", Native: func(args []*Value, this *Value) (*Value, error) {
			m.mu.RLock()
			result := make([]*Value, 0, len(m.data))
			for _, k := range m.keyOrder {
				if v, ok := m.data[k]; ok {
					result = append(result, ArrayVal([]*Value{StringVal(k), v}))
				}
			}
			m.mu.RUnlock()
			return ArrayVal(result), nil
		}}),
		"forEach": FuncVal(&Function{Name: "forEach", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Undefined, nil
			}
			fn := args[0]
			m.mu.RLock()
			keys := make([]string, len(m.keyOrder))
			copy(keys, m.keyOrder)
			m.mu.RUnlock()
			for _, k := range keys {
				m.mu.RLock()
				v, ok := m.data[k]
				m.mu.RUnlock()
				if ok {
					CallFunction(fn, []*Value{v, StringVal(k), this}, nil)
				}
			}
			return Undefined, nil
		}}),
	})
	return obj
}

func setObject(s *ntlSet) *Value {
	obj := ObjectVal(map[string]*Value{
		"size": NumberVal(float64(len(s.items))),
		"add": FuncVal(&Function{Name: "add", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return this, nil
			}
			key := args[0].ToString()
			s.mu.Lock()
			s.items[key] = args[0]
			s.mu.Unlock()
			this.ObjVal["size"] = NumberVal(float64(len(s.items)))
			return this, nil
		}}),
		"has": FuncVal(&Function{Name: "has", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return False, nil
			}
			s.mu.RLock()
			_, ok := s.items[args[0].ToString()]
			s.mu.RUnlock()
			return BoolVal(ok), nil
		}}),
		"delete": FuncVal(&Function{Name: "delete", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return False, nil
			}
			key := args[0].ToString()
			s.mu.Lock()
			_, ok := s.items[key]
			delete(s.items, key)
			s.mu.Unlock()
			if ok {
				this.ObjVal["size"] = NumberVal(float64(len(s.items)))
			}
			return BoolVal(ok), nil
		}}),
		"clear": FuncVal(&Function{Name: "clear", Native: func(args []*Value, this *Value) (*Value, error) {
			s.mu.Lock()
			s.items = make(map[string]*Value)
			s.mu.Unlock()
			this.ObjVal["size"] = NumberVal(0)
			return Undefined, nil
		}}),
		"forEach": FuncVal(&Function{Name: "forEach", Native: func(args []*Value, this *Value) (*Value, error) {
			if len(args) == 0 {
				return Undefined, nil
			}
			fn := args[0]
			s.mu.RLock()
			vals := make([]*Value, 0, len(s.items))
			for _, v := range s.items {
				vals = append(vals, v)
			}
			s.mu.RUnlock()
			for _, v := range vals {
				CallFunction(fn, []*Value{v, v, this}, nil)
			}
			return Undefined, nil
		}}),
		"values": FuncVal(&Function{Name: "values", Native: func(args []*Value, this *Value) (*Value, error) {
			s.mu.RLock()
			vals := make([]*Value, 0, len(s.items))
			for _, v := range s.items {
				vals = append(vals, v)
			}
			s.mu.RUnlock()
			return ArrayVal(vals), nil
		}}),
	})
	return obj
}

func jsonStringify(val *Value, indent string, depth int) string {
	if val == nil {
		return "null"
	}
	switch val.Tag {
	case TypeNull, TypeUndefined:
		return "null"
	case TypeBool:
		if val.BoolVal {
			return "true"
		}
		return "false"
	case TypeNumber:
		if math.IsNaN(val.NumVal) || math.IsInf(val.NumVal, 0) {
			return "null"
		}
		if val.NumVal == math.Trunc(val.NumVal) {
			return fmt.Sprintf("%.0f", val.NumVal)
		}
		return strconv.FormatFloat(val.NumVal, 'f', -1, 64)
	case TypeString:
		return fmt.Sprintf("%q", val.StrVal)
	case TypeArray:
		if len(val.ArrVal) == 0 {
			return "[]"
		}
		var parts []string
		for _, el := range val.ArrVal {
			if el == nil {
				parts = append(parts, "null")
			} else {
				parts = append(parts, jsonStringify(el, indent, depth+1))
			}
		}
		if indent == "" {
			return "[" + strings.Join(parts, ",") + "]"
		}
		pad := strings.Repeat(indent, depth+1)
		return "[\n" + pad + strings.Join(parts, ",\n"+pad) + "\n" + strings.Repeat(indent, depth) + "]"
	case TypeObject:
		if len(val.ObjVal) == 0 {
			return "{}"
		}
		// Sort keys for stable, deterministic output.
		keys := make([]string, 0, len(val.ObjVal))
		for k, v := range val.ObjVal {
			if v == nil || v.Tag == TypeFunction {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			v := val.ObjVal[k]
			key := fmt.Sprintf("%q", k)
			parts = append(parts, key+":"+jsonStringify(v, indent, depth+1))
		}
		if indent == "" {
			return "{" + strings.Join(parts, ",") + "}"
		}
		pad := strings.Repeat(indent, depth+1)
		return "{\n" + pad + strings.Join(parts, ",\n"+pad) + "\n" + strings.Repeat(indent, depth) + "}"
	case TypeInstance:
		obj := ObjectVal(nil)
		if val.InstVal != nil {
			obj.ObjVal = val.InstVal.Fields
		}
		return jsonStringify(obj, indent, depth)
	default:
		return "null"
	}
}

func jsonParse(s string) (*Value, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return Null, nil
	}
	switch s {
	case "null":
		return Null, nil
	case "true":
		return True, nil
	case "false":
		return False, nil
	}
	if s[0] == '"' {
		var str string
		if err := jsonUnquote(s, &str); err != nil {
			return nil, err
		}
		return StringVal(str), nil
	}
	if s[0] == '[' {
		return jsonParseArray(s)
	}
	if s[0] == '{' {
		return jsonParseObject(s)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, &errfmt.LunexError{
			Message: fmt.Sprintf("invalid JSON value: %s", s),
			Kind:    errfmt.KindEncoding,
			Code:    "E0067",
		}
	}
	return NumberVal(f), nil
}

func jsonUnquote(s string, out *string) error {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return &errfmt.LunexError{
			Message: "invalid JSON string: value is not a quoted string",
			Kind:    errfmt.KindEncoding,
			Code:    "E0067",
		}
	}
	inner := s[1 : len(s)-1]
	// Fast path: no escape sequences.
	if !strings.ContainsRune(inner, '\\') {
		*out = inner
		return nil
	}
	var buf strings.Builder
	buf.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		if inner[i] != '\\' || i+1 >= len(inner) {
			buf.WriteByte(inner[i])
			continue
		}
		i++
		switch inner[i] {
		case '"':
			buf.WriteByte('"')
		case '\\':
			buf.WriteByte('\\')
		case '/':
			buf.WriteByte('/')
		case 'n':
			buf.WriteByte('\n')
		case 'r':
			buf.WriteByte('\r')
		case 't':
			buf.WriteByte('\t')
		case 'b':
			buf.WriteByte('\b')
		case 'f':
			buf.WriteByte('\f')
		case 'u':
			if i+4 < len(inner) {
				r, err := strconv.ParseInt(inner[i+1:i+5], 16, 32)
				if err == nil {
					buf.WriteRune(rune(r))
					i += 4
					continue
				}
			}
			buf.WriteString(`\u`)
		default:
			buf.WriteByte('\\')
			buf.WriteByte(inner[i])
		}
	}
	*out = buf.String()
	return nil
}

func jsonParseArray(s string) (*Value, error) {
	if s == "[]" {
		return ArrayVal(nil), nil
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return ArrayVal(nil), nil
	}
	parts := jsonSplit(inner)
	result := make([]*Value, len(parts))
	for i, p := range parts {
		v, err := jsonParse(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return ArrayVal(result), nil
}

func jsonParseObject(s string) (*Value, error) {
	if s == "{}" {
		return ObjectVal(nil), nil
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return ObjectVal(nil), nil
	}
	obj := make(map[string]*Value)
	parts := jsonSplit(inner)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Find the colon that separates key from value by scanning past the
		// closing quote of the key — avoids splitting on colons inside values
		// like {"url": "https://example.com"}.
		colonIdx := -1
		if len(part) > 0 && part[0] == '"' {
			for i := 1; i < len(part); i++ {
				if part[i] == '\\' {
					i++ // skip escaped character
				} else if part[i] == '"' {
					// Scan whitespace then expect ':'
					for j := i + 1; j < len(part); j++ {
						if part[j] == ':' {
							colonIdx = j
							break
						} else if part[j] != ' ' && part[j] != '\t' {
							break
						}
					}
					break
				}
			}
		}
		if colonIdx < 0 {
			colonIdx = strings.Index(part, ":") // fallback for unquoted keys
		}
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:colonIdx])
		val := strings.TrimSpace(part[colonIdx+1:])
		var keyStr string
		if err := jsonUnquote(key, &keyStr); err != nil {
			keyStr = key
		}
		v, err := jsonParse(val)
		if err != nil {
			continue
		}
		obj[keyStr] = v
	}
	return ObjectVal(obj), nil
}

func jsonSplit(s string) []string {
	var parts []string
	depth := 0
	start := 0
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
		} else {
			switch c {
			case '"':
				inStr = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			case ',':
				if depth == 0 {
					parts = append(parts, s[start:i])
					start = i + 1
				}
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func encodeURIComponent(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '!' || r == '~' || r == '*' || r == '\'' || r == '(' || r == ')' {
			buf.WriteRune(r)
		} else {
			for _, b := range []byte(string(r)) {
				buf.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
	}
	return buf.String()
}

// encodeURI encodes a full URI, preserving characters that are legal URI syntax.
func encodeURI(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '!' || r == '~' || r == '*' || r == '\'' || r == '(' || r == ')' ||
			r == ';' || r == ',' || r == '/' || r == '?' || r == ':' || r == '@' || r == '&' ||
			r == '=' || r == '+' || r == '$' || r == '#' {
			buf.WriteRune(r)
		} else {
			for _, b := range []byte(string(r)) {
				buf.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
	}
	return buf.String()
}

func decodeURIComponent(s string) (string, error) {
	var buf strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) {
			hex := s[i+1 : i+3]
			b, err := strconv.ParseUint(hex, 16, 8)
			if err != nil {
				buf.WriteByte(s[i])
				i++
				continue
			}
			buf.WriteByte(byte(b))
			i += 3
		} else if s[i] == '+' {
			buf.WriteByte(' ')
			i++
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String(), nil
}

const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func base64Encode(data []byte) string {
	var buf strings.Builder
	for i := 0; i < len(data); i += 3 {
		b0 := data[i]
		b1 := byte(0)
		b2 := byte(0)
		if i+1 < len(data) {
			b1 = data[i+1]
		}
		if i+2 < len(data) {
			b2 = data[i+2]
		}
		buf.WriteByte(base64Table[b0>>2])
		buf.WriteByte(base64Table[((b0&3)<<4)|(b1>>4)])
		if i+1 < len(data) {
			buf.WriteByte(base64Table[((b1&0xf)<<2)|(b2>>6)])
		} else {
			buf.WriteByte('=')
		}
		if i+2 < len(data) {
			buf.WriteByte(base64Table[b2&0x3f])
		} else {
			buf.WriteByte('=')
		}
	}
	return buf.String()
}

func base64Decode(s string) ([]byte, error) {
	decode := func(c byte) (byte, bool) {
		switch {
		case c >= 'A' && c <= 'Z':
			return c - 'A', true
		case c >= 'a' && c <= 'z':
			return c - 'a' + 26, true
		case c >= '0' && c <= '9':
			return c - '0' + 52, true
		case c == '+':
			return 62, true
		case c == '/':
			return 63, true
		}
		return 0, false
	}
	var result []byte
	for i := 0; i+3 < len(s); i += 4 {
		b0, ok0 := decode(s[i])
		b1, ok1 := decode(s[i+1])
		if !ok0 || !ok1 {
			continue
		}
		result = append(result, (b0<<2)|(b1>>4))
		if s[i+2] != '=' {
			b2, ok2 := decode(s[i+2])
			if ok2 {
				result = append(result, (b1<<4)|(b2>>2))
			}
		}
		if s[i+3] != '=' {
			b2, _ := decode(s[i+2])
			b3, ok3 := decode(s[i+3])
			if ok3 {
				result = append(result, (b2<<6)|b3)
			}
		}
	}
	return result, nil
}
