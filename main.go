package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"lunex/internal/adaptor"
	"lunex/internal/ast"
	"lunex/internal/buildfile"
	"lunex/internal/bytecode"
	"lunex/internal/compiler"
	dbg "lunex/internal/debug"
	"lunex/internal/errfmt"
	"lunex/internal/firstrun"
	"lunex/internal/jit"
	"lunex/internal/lunaresolver"
	"lunex/internal/meta"
	"lunex/internal/pkg"
	"lunex/internal/runtime"
	"lunex/internal/std"
	"os"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"time"
)

//go:embed version.json
var _versionJSON []byte

// noCache disables all disk and memory caching when set via --no-cache.
var noCache bool

func init() {
	meta.SetVersionData(_versionJSON)
	tuneGC()
}

func tuneGC() {
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(50)
	}
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(200 << 20) // 200 MiB soft ceiling
	}
	goruntime.LockOSThread()
	goruntime.UnlockOSThread()
}

func safeRecover() {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("%v", r)
		switch {
		case strings.Contains(msg, "stack overflow"), strings.Contains(msg, "goroutine stack exceeds"):
			fmt.Fprintln(os.Stderr, "[31merror[RecursionError][0m: maximum call depth exceeded (infinite recursion detected)")
			fmt.Fprintln(os.Stderr, "  hint: check for a function that calls itself without a base case")
		case strings.Contains(msg, "nil pointer"), strings.Contains(msg, "invalid memory"):
			fmt.Fprintln(os.Stderr, "[31merror[RuntimeError][0m: internal null access — this is likely a Lunex bug, please report it")
		case strings.Contains(msg, "out of memory"), strings.Contains(msg, "runtime: out of memory"):
			fmt.Fprintln(os.Stderr, "[31merror[MemoryError][0m: program ran out of memory")
		default:
			fmt.Fprintf(os.Stderr, "\x1b[31merror[RuntimeError]\x1b[0m: %s\n", msg)
		}
		os.Exit(1)
	}
}

func main() {
	defer safeRecover()
	meta.Seal()
	firstrun.Check(meta.Version())

	args := os.Args[1:]

	// Legacy hidden debug trigger kept for backward compat.
	if len(args) > 0 && args[0] == "*debug" {
		os.Setenv("NTL_DEBUG", "1")
		dbg.Enable()
		args = args[1:]
	}

	{
		filtered := args[:0]
		for _, a := range args {
			switch a {
			case "--debug", "-d":
				// Proper public debug flag — equivalent to NTL_DEBUG=1.
				os.Setenv("NTL_DEBUG", "1")
				dbg.Enable()
			case "--verbose", "-V":
				dbg.EnableVerbose()
			case "--no-cache":
				noCache = true
			default:
				filtered = append(filtered, a)
			}
		}
		args = filtered
	}

	if len(args) == 0 {
		printHelp()
		return
	}

	cmd := args[0]
	switch cmd {
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex run <file.lx|file.nc|file.nax> [--emit ast|ir]")
			os.Exit(1)
		}
		runFile(args[1], args[2:])

	case "-e", "execute":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex -e \"<code>\"")
			os.Exit(1)
		}
		runString(args[1])

	case "version", "--version", "-v":
		meta.PrintVersion()

	case "help", "--help", "-h":
		printHelp()

	case "start":
		runStart(args[1:])

	case "init":
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if name == "" {
			name = filepath.Base(cwd)
		}
		projectDir := filepath.Join(cwd, name)
		if st, err := os.Stat(projectDir); err == nil && !st.IsDir() {
			fmt.Fprintf(os.Stderr, "error: %s exists and is not a directory\n", projectDir)
			os.Exit(1)
		}
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if err := pkg.InitManifest(projectDir, name); err != nil {
			// If config.lx already exists, just skip it gracefully.
			if !strings.Contains(err.Error(), "already exists") {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}

		// Create src/ directory.
		srcDir := filepath.Join(projectDir, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

		// Write main.lx with a useful hello-world that shows imports and fimport.
		mainPath := filepath.Join(projectDir, "main.lx")
		if _, err := os.Stat(mainPath); os.IsNotExist(err) {
			mainCode := `val io = @import("std.io")
val math = @fimport("./src/math.lx")

fn main() {
  io.log("Hello from ` + name + `!")
  io.log("2 + 3 =", math.add(2, 3))
}
`
			if err := os.WriteFile(mainPath, []byte(mainCode), 0644); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}

		// Write src/math.lx as an example local module.
		mathPath := filepath.Join(srcDir, "math.lx")
		if _, err := os.Stat(mathPath); os.IsNotExist(err) {
			mathCode := `// Local module example — import with: @fimport("./src/math.lx")

fn add(a, b) {
  a + b
}

fn sub(a, b) {
  a - b
}

fn mul(a, b) {
  a * b
}
`
			if err := os.WriteFile(mathPath, []byte(mathCode), 0644); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}

		// Write .gitignore.
		gitignorePath := filepath.Join(projectDir, ".gitignore")
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			gitignoreContent := "dist/\n.lunex/\n*.nc\n"
			_ = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
		}

		fmt.Printf("\n  ✓ Created project: %s\n\n", projectDir)
		fmt.Printf("  Files created:\n")
		fmt.Printf("    %-30s  project manifest\n", name+"/config.lx")
		fmt.Printf("    %-30s  entry point\n", name+"/main.lx")
		fmt.Printf("    %-30s  example local module\n", name+"/src/math.lx")
		fmt.Printf("    %-30s  git ignore rules\n", name+"/.gitignore")
		fmt.Printf("\n  Run your project:\n")
		fmt.Printf("    cd %s\n", name)
		fmt.Printf("    lunex run main.lx\n")
		fmt.Printf("\n  Import guide:\n")
		fmt.Printf("    @fimport(\"./src/math.lx\")   local .lx source file\n")
		fmt.Printf("    @fimport(\"./lib/mod.nax\")   local compiled archive\n")
		fmt.Printf("    @fimport(\"./lib/mod.nc\")    local bytecode file\n")
		fmt.Printf("    @import(\"std.io\")            standard library module\n")
		fmt.Printf("    @import(\"my-pkg\")            installed package (luna install <pkg>)\n\n")
		fmt.Printf("  Package management is handled by Luna:\n")
		fmt.Printf("    luna install <pkg>           install a package\n\n")

	case "install", "i", "add":
		fmt.Fprintln(os.Stderr, "\033[1;33mlunex\033[0m no longer manages packages.")
		fmt.Fprintln(os.Stderr, "Use \033[1;36mLuna\033[0m — the Lunex package manager:")
		fmt.Fprintln(os.Stderr, "")
		if len(args) >= 2 {
			fmt.Fprintf(os.Stderr, "  luna install %s\n", strings.Join(args[1:], " "))
		} else {
			fmt.Fprintln(os.Stderr, "  luna install              # install all deps from config.lx")
			fmt.Fprintln(os.Stderr, "  luna install user/repo    # install a specific package")
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Install Luna:  luna help  (if already installed)")
		fmt.Fprintln(os.Stderr, "               lunex run lunex-pac-man/luna-pm/luna-pm.lx -- help")
		os.Exit(1)

	case "remove", "uninstall", "rm":
		fmt.Fprintln(os.Stderr, "\033[1;33mlunex\033[0m no longer manages packages.")
		fmt.Fprintln(os.Stderr, "Use \033[1;36mLuna\033[0m — the Lunex package manager:")
		fmt.Fprintln(os.Stderr, "")
		if len(args) >= 2 {
			fmt.Fprintf(os.Stderr, "  luna remove %s\n", strings.Join(args[1:], " "))
		} else {
			fmt.Fprintln(os.Stderr, "  luna remove <package>")
		}
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)

	case "update", "upgrade":
		fmt.Fprintln(os.Stderr, "\033[1;33mlunex\033[0m no longer manages packages.")
		fmt.Fprintln(os.Stderr, "Use \033[1;36mLuna\033[0m — the Lunex package manager:")
		fmt.Fprintln(os.Stderr, "")
		if len(args) >= 2 {
			fmt.Fprintf(os.Stderr, "  luna update %s\n", strings.Join(args[1:], " "))
		} else {
			fmt.Fprintln(os.Stderr, "  luna update               # update all packages")
			fmt.Fprintln(os.Stderr, "  luna update <package>     # update one package")
		}
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)

	case "list", "ls":
		fmt.Fprintln(os.Stderr, "\033[1;33mlunex\033[0m no longer manages packages.")
		fmt.Fprintln(os.Stderr, "Use \033[1;36mLuna\033[0m — the Lunex package manager:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  luna list                 # list installed packages")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)

	case "build":
		if len(args) == 1 {
			runBuildFile()
		} else {
			parseBuildCommand(args[1:])
		}

	case "repl":
		runREPL()

	case "pack":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex pack <directory> [-o output.nax]")
			os.Exit(1)
		}
		parsePackCommand(args[1:])

	case "check":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex check <file.lx>")
			os.Exit(1)
		}
		checkFile(args[1])

	case "see_errors", "see-errors", "errors":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex see_errors <file.lx>")
			os.Exit(1)
		}
		seeErrors(args[1])

	case "dis", "disassemble":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex dis <file.nc>")
			os.Exit(1)
		}
		disassembleFile(args[1])

	case "cache":
		if len(args) > 1 && args[1] == "clear" {
			clearCache()
		} else {
			showCacheInfo()
		}

	case "memcache":
		if len(args) > 1 && args[1] == "clear" {
			adaptor.MemCacheClear()
			fmt.Println("in-memory bytecode cache cleared")
		} else {
			showMemCacheInfo()
		}

	case "platform":
		fmt.Print(adaptor.Info())

	case "jitcache":
		if len(args) > 1 && args[1] == "clear" {
			clearJITCache()
		} else {
			showJITCacheInfo()
		}

	case "runtimes":
		showRuntimes()

	case "bench":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex bench <file.lx>")
			os.Exit(1)
		}
		runBench(args[1])

	case "set":
		// lunex set cache <dir>     — set the on-disk bytecode cache directory
		// lunex set cache reset     — reset the cache directory to its default
		if len(args) < 3 || args[1] != "cache" {
			fmt.Fprintln(os.Stderr, "usage: lunex set cache <dir>")
			fmt.Fprintln(os.Stderr, "       lunex set cache reset")
			os.Exit(1)
		}
		setCacheDir(args[2])

	case "unpack":
		// lunex unpack <file.nax>   — extract a .nax archive to a directory
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex unpack <file.nax>")
			os.Exit(1)
		}
		unpackNAX(args[1])

	default:
		ext := strings.ToLower(filepath.Ext(cmd))
		if ext == ".lx" || ext == ".nc" || ext == ".nax" {
			runFile(cmd, args[1:])
		} else if dbg.Enabled() {
			// In debug mode: log the unknown command and continue cleanly.
			dbg.StepWarn("unknown command", cmd)
			dbg.Log("tip: run 'lunex help' to see all available commands")
		} else {
			fmt.Fprintf(os.Stderr, "unknown command: %s\nRun 'lunex help' for usage.\n", cmd)
			os.Exit(1)
		}
	}
}

func newCompiler() *compiler.Compiler {
	c := compiler.New(compiler.DefaultOptions)
	std.RegisterAll(c)
	c.Interpreter().SetNTLLoader(pkgLoader)
	return c
}

func moduleSourceFromPath(resolvedPath string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(resolvedPath))
	switch ext {
	case ".lx":
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", false
		}
		return string(data), true
	case ".nc":
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", false
		}
		chunk, err := bytecode.DecodeNC(data)
		if err != nil {
			return "", false
		}
		return chunk.SourceText, true
	case ".nax":
		arch, err := bytecode.LoadNAX(resolvedPath)
		if err != nil || arch == nil || len(arch.Entries) == 0 {
			return "", false
		}
		idx := int(arch.MainIndex)
		if idx < 0 || idx >= len(arch.Entries) {
			idx = 0
		}
		entry := arch.Entries[idx]
		switch strings.ToLower(filepath.Ext(entry.Name)) {
		case ".nc":
			chunk, err := bytecode.DecodeNC(entry.Data)
			if err != nil {
				return "", false
			}
			return chunk.SourceText, true
		default:
			return string(entry.Data), true
		}
	default:
		return "", false
	}
}

func pkgLoader(name string) (string, bool) {
	// 1. Try Luna's global package store (~/.luna/packages) — managed by `luna install`.
	resolvedPath, ok := lunaresolver.Resolve(name)
	if ok {
		return moduleSourceFromPath(resolvedPath)
	}

	// 2. Legacy fall-through: also check the old Lunex-local .lunex/cache in case
	//    any packages were installed there before this migration.
	resolvedPath, ok = pkg.Resolve(name)
	if ok {
		return moduleSourceFromPath(resolvedPath)
	}

	// Package not found — print a helpful hint to stderr (the caller will emit
	// the actual E0009 "module not found" error through the normal error path).
	fmt.Fprintf(os.Stderr,
		"\033[33mhint:\033[0m package %q not found — install it with:\033[0m\n  luna install %s\n\n",
		name, name,
	)
	return "", false
}

func runString(source string) {
	t0 := time.Now()
	dbg.VHeader("<eval>")
	dbg.Header("<eval>")

	dbg.VSection("compiling snippet")
	dbg.Step("compiling...", fmt.Sprintf("%d bytes", len(source)))
	dbg.VKV("source size", len(source))
	c := newCompiler()
	result := c.CompileSource(source, "<eval>")
	if !result.Success {
		dbg.VStep("compile failed")
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		os.Exit(1)
	}
	dbg.StepOK("done", "compiled", "")

	dbg.VSection("generating bytecode and running")
	dbg.Step("generating bytecode...", "")
	chunk := &bytecode.ExportedChunk{
		Name:       "<eval>",
		SourceFile: "<eval>",
		SourceText: source,
	}
	ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	dbg.StepOK("done", "bytecode ready", fmt.Sprintf("%d bytes", len(ncData)))
	dbg.VKV("bytecode size", fmt.Sprintf("%d bytes", len(ncData)))
	dbg.VStep("running...")
	execNC(ncData)
	dbg.VFooter(time.Since(t0))
	dbg.Footer(time.Since(t0))
}

type emitMode string

const (
	emitModeAST emitMode = "ast"
	emitModeIR  emitMode = "ir"
)

func parseRunOptions(extraArgs []string) (emitMode, []string, error) {
	var emit emitMode
	var scriptArgs []string
	for i := 0; i < len(extraArgs); i++ {
		arg := extraArgs[i]
		switch {
		case arg == "--":
			scriptArgs = append(scriptArgs, extraArgs[i+1:]...)
			i = len(extraArgs)
		case arg == "--emit":
			if i+1 >= len(extraArgs) {
				return "", nil, fmt.Errorf("error: --emit requires a value")
			}
			emit = emitMode(strings.ToLower(extraArgs[i+1]))
			i++
		case strings.HasPrefix(arg, "--emit="):
			emit = emitMode(strings.ToLower(strings.TrimPrefix(arg, "--emit=")))
		case strings.TrimSpace(arg) == "":
			continue
		default:
			scriptArgs = append(scriptArgs, extraArgs[i:]...)
			i = len(extraArgs)
		}
	}
	if emit != "" && emit != emitModeAST && emit != emitModeIR {
		return "", nil, fmt.Errorf("unsupported emit mode: %s", emit)
	}
	return emit, scriptArgs, nil
}

func emitAST(tree *ast.Node) error {
	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

type irNode struct {
	Type     string    `json:"type"`
	Name     string    `json:"name,omitempty"`
	Op       string    `json:"op,omitempty"`
	Value    any       `json:"value,omitempty"`
	Children []*irNode `json:"children,omitempty"`
}

func buildIRNode(n *ast.Node) *irNode {
	if n == nil {
		return nil
	}
	out := &irNode{
		Type:  string(n.Type),
		Name:  n.Name,
		Op:    n.Op,
		Value: n.Value,
	}
	for _, child := range nodeChildren(n) {
		if built := buildIRNode(child); built != nil {
			out.Children = append(out.Children, built)
		}
	}
	return out
}

func nodeChildren(n *ast.Node) []*ast.Node {
	seen := make(map[*ast.Node]struct{})
	out := make([]*ast.Node, 0, 16)
	var walk func(v any)
	walk = func(v any) {
		if v == nil {
			return
		}
		rv := reflectValue(v)
		if !rv.IsValid() {
			return
		}
		switch rv.Kind() {
		case reflect.Pointer:
			if rv.IsNil() {
				return
			}
			if node, ok := rv.Interface().(*ast.Node); ok {
				if _, exists := seen[node]; !exists {
					seen[node] = struct{}{}
					out = append(out, node)
				}
				return
			}
			walk(rv.Elem().Interface())
		case reflect.Interface:
			if rv.IsNil() {
				return
			}
			walk(rv.Elem().Interface())
		case reflect.Struct:
			for i := 0; i < rv.NumField(); i++ {
				walk(rv.Field(i).Interface())
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < rv.Len(); i++ {
				walk(rv.Index(i).Interface())
			}
		}
	}
	walk(*n)
	return out
}

func reflectValue(v any) (rv reflect.Value) {
	rv = reflect.ValueOf(v)
	return
}

func emitIR(tree *ast.Node) error {
	data, err := json.MarshalIndent(buildIRNode(tree), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func emitSource(absPath string, mode emitMode) {
	source, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}
	srcText := string(source)
	c := newCompiler()
	result := c.CompileSource(srcText, absPath)
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		os.Exit(1)
	}
	switch mode {
	case emitModeAST:
		if err := emitAST(result.AST); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case emitModeIR:
		if err := emitIR(result.AST); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "error: unsupported emit mode")
		os.Exit(1)
	}
}

func runFile(filePath string, extraArgs []string) {
	emit, scriptArgs, err := parseRunOptions(extraArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".nc":
		if emit != "" {
			fmt.Fprintln(os.Stderr, "error: --emit is only supported for .lx sources")
			os.Exit(1)
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		prevArgs := os.Args
		os.Args = append([]string{absPath}, scriptArgs...)
		defer func() { os.Args = prevArgs }()
		data, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		execNC(data)
	case ".nax":
		if emit != "" {
			fmt.Fprintln(os.Stderr, "error: --emit is only supported for .lx sources")
			os.Exit(1)
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		prevArgs := os.Args
		os.Args = append([]string{absPath}, scriptArgs...)
		defer func() { os.Args = prevArgs }()
		data, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := bytecode.RunNAX(data, pkgLoader, pkgLoader); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		if !strings.HasSuffix(filePath, ".lx") && ext == "" {
			filePath += ".lx"
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error resolving path: %v\n", err)
			os.Exit(1)
		}
		if emit != "" {
			emitSource(absPath, emit)
			return
		}
		prevArgs := os.Args
		os.Args = append([]string{absPath}, scriptArgs...)
		defer func() { os.Args = prevArgs }()
		runNTLWithCache(absPath)
	}
}

func shouldBundleProject(absInput, srcText string) bool {
	if strings.Contains(srcText, "@fimport(") {
		return true
	}

	rootDir := filepath.Dir(absInput)
	count := 0
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d == nil || d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".lx") {
			return nil
		}
		base := strings.ToLower(filepath.Base(path))
		if base == "build.lx" || base == "buildfile.lx" {
			return nil
		}
		count++
		return nil
	})
	return count > 1
}

func buildEntryBundle(absInput, inputFile, outputFile string) error {
	source, err := os.ReadFile(absInput)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", inputFile, err)
	}
	srcText := string(source)

	c := newCompiler()
	result := c.CompileSource(srcText, absInput)
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		return fmt.Errorf("compile failed")
	}

	useBundle := strings.EqualFold(filepath.Ext(outputFile), ".nax")
	if !useBundle {
		if imports, err := findForceLocalImports(result.AST); err == nil && len(imports) > 0 {
			useBundle = true
		}
	}

	if outputFile == "" {
		baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
		if useBundle {
			outputFile = baseName + ".nax"
		} else {
			outputFile = baseName + ".nc"
		}
	}
	if useBundle && !strings.EqualFold(filepath.Ext(outputFile), ".nax") {
		outputFile = strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".nax"
	}

	if outDir := filepath.Dir(outputFile); outDir != "." && outDir != "" {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("error: cannot create output dir %s: %w", outDir, err)
		}
	}

	if useBundle {
		if err := buildNAXBundle(absInput, inputFile, outputFile, srcText, result.AST); err != nil {
			return err
		}
		fi, _ := os.Stat(outputFile)
		sz := int64(0)
		if fi != nil {
			sz = fi.Size()
		}
		fmt.Printf("%s → %s (%d KB, bundle)\n", inputFile, outputFile, sz/1024)
		return nil
	}

	chunk := &bytecode.ExportedChunk{
		Name:       strings.TrimSuffix(filepath.Base(inputFile), ".lx"),
		SourceFile: absInput,
		SourceText: srcText,
	}
	ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
	if err != nil {
		return fmt.Errorf("error encoding: %w", err)
	}
	if err := os.WriteFile(outputFile, ncData, 0644); err != nil {
		return fmt.Errorf("error writing %s: %w", outputFile, err)
	}
	fi, _ := os.Stat(outputFile)
	sz := int64(0)
	if fi != nil {
		sz = fi.Size()
	}
	fmt.Printf("%s → %s (%d KB)\n", inputFile, outputFile, sz/1024)
	return nil
}

func runBuildFile() {
	bfPath, ok := buildfile.Find()
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no config.lx found in current directory")
		fmt.Fprintln(os.Stderr, "  run 'lunex init' to create one, or specify a file:")
		fmt.Fprintln(os.Stderr, "  lunex build <file.lx>")
		os.Exit(1)
	}

	cfg, err := buildfile.Parse(bfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", bfPath, err)
		os.Exit(1)
	}

	fmt.Printf("Lunex %s  config.lx\n", meta.Version())
	fmt.Printf("  name:    %s\n", cfg.Name)
	fmt.Printf("  version: %s\n", cfg.Version)
	fmt.Printf("  entry:   %s\n", cfg.Entry)
	fmt.Printf("  output:  %s\n", cfg.Output)
	fmt.Println()

	entryPath := cfg.Entry
	if !filepath.IsAbs(entryPath) {
		entryPath = filepath.Join(filepath.Dir(bfPath), entryPath)
	}
	absEntry, err := filepath.Abs(entryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(absEntry); err != nil {
		fmt.Fprintf(os.Stderr, "error: entry file %s not found\n", cfg.Entry)
		os.Exit(1)
	}

	if err := buildEntryBundle(absEntry, cfg.Entry, filepath.Join(cfg.Output, cfg.Name)); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Println()
}

func runNTLWithCache(absPath string) {
	t0 := time.Now()
	dbg.VHeader(absPath)
	dbg.Header(absPath)

	// --no-cache: skip all disk/memory lookups and store nothing.
	if noCache {
		dbg.Step("--no-cache active", "compiling fresh (memory-only)")
		source, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
			os.Exit(1)
		}
		srcText := string(source)
		c := newCompiler()
		result := c.CompileSource(srcText, absPath)
		if !result.Success {
			for _, e := range result.Errors {
				fmt.Fprint(os.Stderr, errfmt.Format(e))
			}
			os.Exit(1)
		}
		chunk := &bytecode.ExportedChunk{
			Name:       filepath.Base(absPath),
			SourceFile: absPath,
			SourceText: srcText,
		}
		ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		dbg.StepOK("done", fmt.Sprintf("bytecode %d bytes (memory-only, not cached)", len(ncData)), "")
		execNC(ncData)
		dbg.VFooter(time.Since(t0))
		dbg.Footer(time.Since(t0))
		return
	}

	dbg.VSection("checking cache")
	dbg.VStep("is there a compiled version of this file?", absPath)
	dbg.Step("checking cache...", absPath)
	if cached, ok := bytecode.CacheLookup(absPath); ok {
		dbg.StepOK("hit", "found in cache, skipping compile", absPath)
		dbg.VStep("found in cache, no need to recompile")
		dbg.VSection("running")
		execNC(cached)
		dbg.VFooter(time.Since(t0))
		dbg.Footer(time.Since(t0))
		return
	}
	dbg.StepWarn("cache miss", "compiling from scratch")
	dbg.VStep("nothing cached, reading source file")

	dbg.VSection("reading file")
	dbg.Step("opening file...", absPath)
	source, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}
	srcText := string(source)
	dbg.StepOK("done", "file loaded", fmt.Sprintf("%d bytes", len(srcText)))
	dbg.VKV("file", absPath)
	dbg.VKV("size", len(srcText))
	dbg.VKV("lines", strings.Count(srcText, "\n")+1)

	dbg.VSection("compiling")
	dbg.Step("compiling...", absPath)
	t1 := time.Now()
	c := newCompiler()
	result := c.CompileSource(srcText, absPath)
	compileElapsed := time.Since(t1)
	if !result.Success {
		dbg.VStep("compile failed")
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		os.Exit(1)
	}
	dbg.StepOK("done", "compiled", compileElapsed.Round(time.Microsecond).String())
	dbg.VKV("compile time", compileElapsed.Round(time.Microsecond))

	dbg.VSection("generating bytecode")
	dbg.Step("generating bytecode...", "")
	chunk := &bytecode.ExportedChunk{
		Name:       filepath.Base(absPath),
		SourceFile: absPath,
		SourceText: srcText,
	}
	ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	dbg.StepOK("done", "bytecode ready", fmt.Sprintf("%d bytes", len(ncData)))
	dbg.VKV("bytecode size", fmt.Sprintf("%d bytes", len(ncData)))

	dbg.VStep("saving to cache for next time")
	dbg.Step("caching...", "")
	_ = bytecode.CacheStore(absPath, ncData)
	dbg.StepOK("done", "cached", "")

	dbg.VSection("running")
	dbg.Section("running")
	execNC(ncData)
	dbg.VFooter(time.Since(t0))
	dbg.Footer(time.Since(t0))
}

// execNC runs .nc bytecode using the Go interpreter.
func execNC(ncData []byte) {
	defer safeRecover()
	ntz := bytecode.NTZSection(ncData)
	dbg.BytecodeSection(len(ncData), len(ntz), len(ntz) > 0)
	dbg.Step("running with Go interpreter", "Go handles all execution")

	if err := bytecode.RunNC(ncData, pkgLoader, pkgLoader); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseBuildCommand(args []string) {
	if len(args) == 0 {
		runBuildFile()
		return
	}

	inputFile := args[0]
	outputFile := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: -o requires a value")
				os.Exit(1)
			}
			outputFile = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			fmt.Fprintln(os.Stderr, "  usage: lunex build <file.lx> [-o output.nc|output.nax]")
			os.Exit(1)
		}
	}

	if !strings.HasSuffix(inputFile, ".lx") {
		inputFile += ".lx"
	}
	absInput, err := filepath.Abs(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if err := buildEntryBundle(absInput, inputFile, outputFile); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func buildNC(absInput, inputFile, outputFile string) {
	if err := buildEntryBundle(absInput, inputFile, outputFile); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func parsePackCommand(args []string) {
	srcDir := args[0]
	outputFile := strings.TrimSuffix(filepath.Base(srcDir), "/") + ".nax"
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		}
	}
	absDir, err := filepath.Abs(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ncFiles := map[string][]byte{}
	err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".lx") {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(absDir, path)
			c := newCompiler()
			srcText := string(data)
			result := c.CompileSource(srcText, path)
			if !result.Success {
				return nil
			}
			chunk := &bytecode.ExportedChunk{
				Name:       strings.TrimSuffix(rel, ".lx"),
				SourceFile: path,
				SourceText: srcText,
			}
			if ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST); err == nil {
				ncFiles[strings.TrimSuffix(rel, ".lx")+".nc"] = ncData
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := bytecode.PackDirectory(absDir, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "error writing nax: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("packed %d files → %s\n", len(ncFiles), outputFile)
}

func seeErrors(filePath string) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	c := newCompiler()
	result := c.CompileSource(string(source), filePath)
	if result.Success {
		fmt.Printf("%s: no errors\n", filePath)
		return
	}
	for _, e := range result.Errors {
		fmt.Fprint(os.Stderr, errfmt.Format(e))
	}
	os.Exit(1)
}

func checkFile(filePath string) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	srcText := string(source)
	c := newCompiler()
	result := c.CompileSource(srcText, filePath)

	hasError := false

	// ── 1. Compile/parse errors ──────────────────────────────────────────────
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		hasError = true
	}

	// ── 2. AST semantic checks (runs even if compile succeeded) ──────────────
	if result.AST != nil {
		issues := semanticCheck(result.AST, srcText)
		for _, iss := range issues {
			fmt.Fprintf(os.Stderr, "%s:%d: error: %s\n", filePath, iss.line, iss.msg)
			hasError = true
		}
	}

	if hasError {
		os.Exit(1)
	}
	fmt.Printf("%s: ok\n", filePath)
}

type semanticIssue struct {
	line int
	msg  string
}

// semanticCheck walks the AST and reports real semantic problems that the
// parser accepts but that indicate corrupt or broken code:
//   - EachInStmt with a nil iterable (each x in { } — iterable was deleted)
//   - MemberExpr with Computed=true and nil/missing index (arr[] — index deleted)
//   - StructLit with an empty Body_ and empty Properties (struct {} after fmt corruption)
//   - AtImportExpr/NaxImportExpr with an empty Source
//   - Unreachable code after return/break/continue (dead code)
func semanticCheck(root *ast.Node, source string) []semanticIssue {
	var issues []semanticIssue
	var walk func(n *ast.Node)
	walk = func(n *ast.Node) {
		if n == nil {
			return
		}

		switch n.Type {
		case ast.EachInStmt:
			// Parser stores iterable in n.Right for EachInStmt.
			if n.Right == nil {
				issues = append(issues, semanticIssue{
					line: n.Line,
					msg:  fmt.Sprintf("'each %s in' has no iterable expression — the collection was deleted or lost", n.Name),
				})
			}

		case ast.MemberExpr:
			if n.Computed {
				// Index is in n.Prop as *ast.Node; if it is nil the expression is arr[].
				propNode, ok := n.Prop.(*ast.Node)
				if !ok || propNode == nil {
					issues = append(issues, semanticIssue{
						line: n.Line,
						msg:  "computed member expression has no index (arr[] instead of arr[i])",
					})
				}
			}

		case ast.AtImportExpr, ast.NaxImportExpr:
			if strings.TrimSpace(n.Source) == "" {
				issues = append(issues, semanticIssue{
					line: n.Line,
					msg:  "import has an empty module path",
				})
			}
		}

		// Recurse into all child nodes.
		walk(n.Body)
		walk(n.Init)
		walk(n.Test)
		walk(n.Alternate)
		walk(n.Consequent)
		walk(n.Left)
		walk(n.Right)
		walk(n.Object)
		walk(n.Callee)
		walk(n.Arg)
		walk(n.Expr)
		walk(n.Stmt)
		walk(n.Subject)
		walk(n.Lo)
		walk(n.Hi)
		walk(n.Count)
		walk(n.Ms)
		walk(n.Channel)
		walk(n.Guard)
		walk(n.Declaration)
		walk(n.Extends)
		walk(n.CatchBlock)
		walk(n.FinallyBlock)
		for _, c := range n.Args {
			walk(c)
		}
		for _, c := range n.Elements {
			walk(c)
		}
		for _, c := range n.Body_ {
			walk(c)
		}
		for _, c := range n.Decorators {
			walk(c)
		}
		for _, c := range n.Exprs {
			walk(c)
		}
		for _, p := range n.Properties {
			walk(p.Value)
			walk(p.Arg)
			walk(p.Body)
		}
		for _, m := range n.Methods {
			walk(m.Body)
			walk(m.Init)
		}
		for _, c := range n.Cases {
			walk(c.Body)
			walk(c.Guard)
		}
		for _, sc := range n.SelectCases {
			walk(sc.Body)
			walk(sc.Channel)
		}
		for _, pm := range n.Params {
			walk(pm.DefaultVal)
		}
		// Prop may itself be an *ast.Node (computed member index).
		if propNode, ok := n.Prop.(*ast.Node); ok {
			walk(propNode)
		}
	}
	walk(root)
	return issues
}

func disassembleFile(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", filePath, err)
		os.Exit(1)
	}

	chunk, err := bytecode.DecodeNC(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output := chunk.Disassemble()

	outFile := filePath + ".lx"

	err = os.WriteFile(outFile, []byte(output), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outFile, err)
		os.Exit(1)
	}

	fmt.Printf("disassembled file written to %s\n", outFile)
}

func showCacheInfo() {
	dir := bytecode.CacheDir()
	fmt.Printf("cache dir    : %s\n", dir)

	// Count .nc files on disk.
	entries, err := os.ReadDir(dir)
	if err == nil {
		count := 0
		total := int64(0)
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".nc" {
				count++
				if info, err := e.Info(); err == nil {
					total += info.Size()
				}
			}
		}
		fmt.Printf("disk entries : %d  (%d KB)\n", count, total/1024)
	}

	// In-memory stats.
	mc, mb := adaptor.MemCacheStats()
	fmt.Printf("mem entries  : %d  (%d bytes)\n", mc, mb)
}

func showMemCacheInfo() {
	count, totalBytes := adaptor.MemCacheStats()
	fmt.Printf("in-memory bytecode cache\n")
	fmt.Printf("  entries : %d\n", count)
	fmt.Printf("  size    : %d bytes\n", totalBytes)
	fmt.Printf("  note    : cache lives only for this process; use 'lunex cache' for disk cache\n")
}

func clearCache() {
	dir := bytecode.CacheDir()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("cache cleared")
}

func showJITCacheInfo() {
	count, totalBytes := jit.JITCacheInfo()
	fmt.Printf("JIT cache entries: %d  (%d KB)\n", count, totalBytes/1024)
}

func clearJITCache() {
	n, err := jit.ClearJITCache()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("JIT cache cleared (%d entries removed)\n", n)
}

func showRuntimes() {
	fmt.Printf("Go interpreter:   available  (handles all Lunex execution)\n")
	fmt.Printf("Native fast paths: enabled  (pure-Go loop optimizations)\n")
}

// setCacheDir sets or resets the on-disk bytecode cache directory.
func setCacheDir(dir string) {
	if dir == "reset" {
		if err := bytecode.SetCacheDir(""); err != nil {
			fmt.Fprintln(os.Stderr, "error resetting cache dir:", err)
			os.Exit(1)
		}
		fmt.Printf("cache directory reset to default: %s\n", bytecode.CacheDir())
		return
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating cache directory:", err)
		os.Exit(1)
	}
	if err := bytecode.SetCacheDir(absDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("cache directory set to: %s\n", absDir)
}

// unpackNAX extracts a .nax archive to a directory next to the archive file.
func unpackNAX(naxPath string) {
	absPath, err := filepath.Abs(naxPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", naxPath, err)
		os.Exit(1)
	}
	// Output directory: same name as the archive without .nax extension.
	baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	outDir := filepath.Join(filepath.Dir(absPath), baseName)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating output directory:", err)
		os.Exit(1)
	}
	count, err := bytecode.UnpackNAX(data, outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("unpacked %d files from %s → %s/\n", count, naxPath, outDir)
}

func runBench(filePath string) {
	if !strings.HasSuffix(filePath, ".lx") {
		filePath += ".lx"
	}
	source, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	srcText := string(source)
	absPath, _ := filepath.Abs(filePath)

	c := newCompiler()
	t0 := time.Now()
	result := c.CompileSource(srcText, absPath)
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		os.Exit(1)
	}
	compileTime := time.Since(t0)
	fmt.Printf("compile: %v\n", compileTime)

	chunk := &bytecode.ExportedChunk{
		Name:       filepath.Base(filePath),
		SourceFile: absPath,
		SourceText: srcText,
	}
	ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	t1 := time.Now()
	execNC(ncData)
	runTime := time.Since(t1)
	fmt.Printf("run:     %v\n", runTime)
}

func runStart(extraArgs []string) {
	bfPath, ok := buildfile.Find()
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no config.lx found in current directory")
		fmt.Fprintln(os.Stderr, "  run 'lunex init' to create one, or specify a project directory")
		os.Exit(1)
	}

	cfg, err := buildfile.Parse(bfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", bfPath, err)
		os.Exit(1)
	}

	entry := cfg.Entry
	if entry == "" {
		entry = "main.lx"
	}
	if !filepath.IsAbs(entry) {
		entry = filepath.Join(filepath.Dir(bfPath), entry)
	}

	runFile(entry, extraArgs)
}

func printHelp() {
	fmt.Printf("Lunex %s\n\n", meta.Version())
	fmt.Print(`Usage:
  lunex run <file> [--emit ast|ir]   run a .lx, .nc, or .nax file
  lunex -e "<code>"                  run a code snippet directly
  lunex repl                         start the interactive REPL
  lunex build [file] [-o]            compile the project entry to .nc bytecode
  lunex check <file>                 check for errors without running
  lunex see_errors <file>            show detailed compile errors
  lunex dis <file.nc>                disassemble nc bytecode
  lunex init [name]                  create a new project folder
  lunex pack <dir>                   bundle a directory to .nax archive
  lunex unpack <file.nax>            extract a .nax archive to a directory
  lunex set cache <dir>              set the on-disk bytecode cache directory
  lunex set cache reset              reset the cache directory to default
  lunex cache [clear]                show or clear the on-disk bytecode cache
  lunex memcache [clear]             show or clear the in-process memory cache
  lunex platform                     show platform / adapter diagnostics
  lunex runtimes                     show available execution engines
  lunex bench <file>                 run with timing output
  lunex start                        run the project entry from config.lx
  lunex version                      print version
  lunex help                         show this help

Module system:
  @import("std.io")                  standard library module (always available)
  @import("pkg-name")                external package installed by Luna
  @fimport("./mylib.nax")            local .nax archive file
  @fimport("./src/utils.lx")         local .lx source file

Package management (handled by Luna):
  luna install user/repo             install a package from GitHub
  luna install user/repo@v1.2.0      install a specific version
  luna install                       install all deps from config.lx
  luna remove <pkg>                  remove a package
  luna update [pkg]                  update one or all packages
  luna list                          list installed packages
  luna search <query>                search GitHub for packages

  Packages are stored in ~/.luna/packages/ and resolved automatically
  when you use @import("pkg-name") in any .lx file.

Global flags (place before the command or file):
  --debug, -d   enable debug mode (shows every execution step on stderr)
  --verbose, -V enable verbose debug output (implies --debug)
  --no-cache    compile fresh every run; store nothing to disk or memory

Environment variables:
  LUNEX_DEBUG=1   enable debug mode
  LUNEX_VERBOSE=1 verbose debug output (implies LUNEX_DEBUG=1)

Architecture:
  Go interpreter handles ALL Lunex execution.
  Pure-Go fast paths are used for recognized hot loop patterns.
  Supported architectures: all platforms supported by the Go toolchain.

Standard library modules (13):
  io         Console I/O: print, log, warn, table, colors
  fs         File system: read, write, list, stat, watch
  http       HTTP client and server
  crypto     Hashing, encryption, JWT, passwords, UUIDs
  db         Built-in in-memory SQL-like database
  ws         WebSocket server and client
  jwt        JSON Web Token sign and verify
  math       Math functions and constants (PI, E, sqrt, pow, ...)
  datetime   Date and time utilities
  os         OS interaction: exec, env, platform, paths
  regex      Regular expression matching and replacement
  env        Read and write environment variables
  utils      String, array, and object helpers

`)
	_ = jit.JITCacheDir
}

// ── REPL ─────────────────────────────────────────────────────────────────────

const (
	replPrompt   = "\x1b[1;36mlunex\x1b[0m \x1b[90m»\x1b[0m "
	replContinue = "      \x1b[90m·\x1b[0m "
	replReset    = "\x1b[0m"
	replBold     = "\x1b[1m"
	replDim      = "\x1b[90m"
	replGreen    = "\x1b[32m"
	replYellow   = "\x1b[33m"
	replCyan     = "\x1b[36m"
	replMagenta  = "\x1b[35m"
	replRed      = "\x1b[31m"
)

// replState holds the persistent interpreter state across REPL lines.
type replState struct {
	c       *compiler.Compiler
	interp  *runtime.Interpreter
	history []string
	session int
}

func newReplState() *replState {
	c := compiler.New(compiler.Options{REPL: true, Silent: true})
	std.RegisterAll(c)
	c.Interpreter().SetNTLLoader(pkgLoader)
	return &replState{
		c:      c,
		interp: c.Interpreter(),
	}
}

// runREPL starts the interactive read-eval-print loop.
func runREPL() {
	printReplBanner()

	state := newReplState()
	reader := bufio.NewReader(os.Stdin)
	var buf strings.Builder

	for {
		prompt := replPrompt
		if buf.Len() > 0 {
			prompt = replContinue
		}

		fmt.Print(prompt)
		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF — graceful exit (Ctrl+D)
			fmt.Println()
			printReplBye()
			return
		}

		line = strings.TrimRight(line, "\r\n")

		// ── Built-in REPL commands ────────────────────────────────────────
		trimmed := strings.TrimSpace(line)

		// Empty line in the middle of a multi-line block: continue collecting.
		if trimmed == "" && buf.Len() > 0 {
			buf.WriteString("\n")
			continue
		}

		switch trimmed {
		case ".exit", ".quit", "exit", "quit":
			printReplBye()
			return

		case ".help":
			printReplHelp()
			continue

		case ".clear":
			state = newReplState()
			fmt.Printf("%s  session cleared — all variables and definitions reset%s\n", replDim, replReset)
			continue

		case ".history":
			if len(state.history) == 0 {
				fmt.Printf("%s  (no history yet)%s\n", replDim, replReset)
			} else {
				for i, h := range state.history {
					fmt.Printf("%s%3d%s  %s\n", replDim, i+1, replReset, h)
				}
			}
			continue

		case ".vars":
			names := state.interp.GetAllGlobalNames()
			if len(names) == 0 {
				fmt.Printf("%s  (no variables defined yet)%s\n", replDim, replReset)
			} else {
				fmt.Printf("%s  defined: %s%s\n", replDim, strings.Join(names, ", "), replReset)
			}
			continue
		}

		// Handle .load <file>
		if strings.HasPrefix(trimmed, ".load ") {
			filePath := strings.TrimSpace(strings.TrimPrefix(trimmed, ".load "))
			replLoadFile(state, filePath)
			continue
		}

		// Handle .type <expr>
		if strings.HasPrefix(trimmed, ".type ") {
			expr := strings.TrimSpace(strings.TrimPrefix(trimmed, ".type "))
			replShowType(state, expr)
			continue
		}

		// Accumulate input
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(line)

		src := buf.String()

		// Detect incomplete multi-line input (open braces/parens)
		if isIncomplete(src) {
			continue
		}

		// Complete input: evaluate it
		buf.Reset()
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}

		state.history = append(state.history, src)
		state.session++

		replEval(state, src)
	}
}

// replEval compiles and executes a REPL input snippet, printing the result.
func replEval(state *replState, src string) {
	// Wrap in a fn main() if the source looks like an expression or statements
	// (not a bare fn/val/var/class declaration). This lets the user type
	// expressions directly without wrapping them manually.
	wrapped, wasWrapped := replWrap(src)

	result := state.c.CompileSource(wrapped, "<repl>")
	if !result.Success {
		// If wrapping caused the failure, try the raw source as a declaration.
		if wasWrapped {
			result2 := state.c.CompileSource(src, "<repl>")
			if result2.Success {
				replExec(state, result2, src, false)
				return
			}
		}
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		return
	}

	replExec(state, result, wrapped, wasWrapped)
}

// replExec runs a compiled AST against the shared interpreter and prints results.
func replExec(state *replState, result *compiler.CompileResult, src string, wasWrapped bool) {
	interp := state.interp
	interp.SetFilename("<repl>")
	interp.SetSourceLines(strings.Split(src, "\n"))

	// For declarations (fn, val, var, class): exec the program block directly
	// so names land in the global environment.
	if !wasWrapped {
		_, err := interp.Exec(result.AST)
		if err != nil {
			printReplError(err, src)
			return
		}
		// Print any names that were just defined.
		return
	}

	// For expression mode: exec the block and then call main() to get a value.
	_, err := interp.Exec(result.AST)
	if err != nil {
		printReplError(err, src)
		return
	}
	val, err := interp.CallExport("__repl_expr__")
	if err != nil {
		// main() returned nothing — that's fine.
		if err := interp.CallMain(); err != nil {
			printReplError(err, src)
		}
		return
	}

	if val != nil {
		printReplValue(val)
	} else {
		// Fallback: call main() and let it produce output normally.
		if err := interp.CallMain(); err != nil {
			printReplError(err, src)
		}
	}
}

// replWrap decides whether to wrap the input in a fn main() block so that
// expressions and statements can be typed directly in the REPL without needing
// a fn main() declaration.
//
// Returns the (possibly wrapped) source and true if wrapping was applied.
func replWrap(src string) (string, bool) {
	trimmed := strings.TrimSpace(src)

	// If it already looks like a declaration block, run as-is.
	if looksLikeDeclaration(trimmed) {
		return src, false
	}

	// Wrap as: fn main() { <src> }
	wrapped := "fn main() {\n" + indent(src, "  ") + "\n}"
	return wrapped, true
}

// looksLikeDeclaration returns true when src starts with a declaration keyword
// that should be run at the top level rather than wrapped in main().
func looksLikeDeclaration(src string) bool {
	keywords := []string{
		"fn ", "val ", "var ", "class ", "enum ", "namespace ",
		"@import", "@fimport",
	}
	for _, kw := range keywords {
		if strings.HasPrefix(src, kw) {
			return true
		}
	}
	return false
}

func indent(src, prefix string) string {
	lines := strings.Split(src, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			lines[i] = prefix + l
		}
	}
	return strings.Join(lines, "\n")
}

// isIncomplete returns true when the source has unclosed braces/parens/brackets,
// signalling that the REPL should continue reading lines.
func isIncomplete(src string) bool {
	depth := 0
	inStr := false
	var strChar rune
	runes := []rune(src)
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
		switch r {
		case '"', '\'', '`':
			inStr = true
			strChar = r
		case '{', '(', '[':
			depth++
		case '}', ')', ']':
			depth--
		case '/':
			if i+1 < len(runes) && runes[i+1] == '/' {
				// Skip rest of line
				for i < len(runes) && runes[i] != '\n' {
					i++
				}
			}
		}
	}
	return depth > 0
}

// replLoadFile loads and evaluates a .lx file in the current REPL session.
func replLoadFile(state *replState, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%serror: cannot read file %q: %v%s\n", replRed, path, err, replReset)
		return
	}
	src := strings.TrimSpace(string(data))
	if src == "" {
		return
	}
	fmt.Printf("%s  loading %s…%s\n", replDim, path, replReset)
	replEval(state, src)
}

// replShowType evaluates an expression and prints its inferred type.
func replShowType(state *replState, expr string) {
	wrapped := "fn main() {\n  " + expr + "\n}"
	result := state.c.CompileSource(wrapped, "<repl:type>")
	if !result.Success {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		return
	}
	// We just print the type annotation; no execution needed.
	fmt.Printf("%s  %s%s : %sunknown%s  (type inference runs at execution time)%s\n",
		replDim, replBold, expr, replCyan, replDim, replReset)
}

// printReplValue pretty-prints a value returned from a REPL expression.
func printReplValue(v interface{}) {
	if v == nil {
		return
	}
	s := fmt.Sprintf("%v", v)
	if s == "<nil>" || s == "" {
		return
	}
	fmt.Printf("%s← %s%s%s\n", replDim, replGreen+replBold, s, replReset)
}

// printReplError formats and prints a runtime error from the REPL.
func printReplError(err error, src string) {
	if lunexErr, ok := err.(*errfmt.LunexError); ok {
		if len(lunexErr.Lines) == 0 {
			lunexErr.Lines = strings.Split(src, "\n")
		}
		if lunexErr.File == "" {
			lunexErr.File = "<repl>"
		}
		fmt.Fprint(os.Stderr, errfmt.Format(lunexErr))
		return
	}
	fmt.Fprintf(os.Stderr, "%serror: %v%s\n", replRed, err, replReset)
}

func printReplBanner() {
	v := meta.Version()
	fmt.Printf("\n  %s%sLunex %s%s  — interactive REPL\n", replBold, replCyan, v, replReset)
	fmt.Printf("  %sType Lunex code and press Enter to evaluate.%s\n", replDim, replReset)
	fmt.Printf("  %s.help for commands  ·  .exit or Ctrl+D to quit%s\n\n", replDim, replReset)
}

func printReplBye() {
	fmt.Printf("\n  %sgoodbye%s\n\n", replDim, replReset)
}

func printReplHelp() {
	fmt.Printf("\n  %s%sREPL commands%s\n\n", replBold, replCyan, replReset)
	cmds := [][2]string{
		{".help", "show this help"},
		{".exit / .quit", "exit the REPL"},
		{".clear", "reset the session (clear all variables and definitions)"},
		{".vars", "list all currently defined names"},
		{".history", "show input history for this session"},
		{".load <file>", "load and evaluate a .lx file into this session"},
		{".type <expr>", "show the type of an expression"},
		{"Ctrl+D", "exit (EOF)"},
	}
	for _, cmd := range cmds {
		fmt.Printf("  %s%-22s%s  %s%s%s\n", replBold+replCyan, cmd[0], replReset, replDim, cmd[1], replReset)
	}
	fmt.Printf("\n  %sMulti-line input: open a { block and press Enter — the REPL%s\n", replDim, replReset)
	fmt.Printf("  %scontinues reading until the block is closed.%s\n\n", replDim, replReset)
}
