package main

import (
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
	"lunex/internal/meta"
	"lunex/internal/pkg"
	"lunex/internal/std"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
	_ "embed"
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
	runtime.LockOSThread()
	runtime.UnlockOSThread()
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
			gitignoreContent := "dist/\n.lunex-cache/\n*.nc\n"
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
		fmt.Printf("    @import(\"my-pkg\")            installed package (lunex add <pkg>)\n\n")

	case "install", "i":
		if len(args) < 2 {
			installAll()
		} else {
			for _, spec := range args[1:] {
				installPkg(spec, false)
			}
		}

	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex add <package>[@version]")
			os.Exit(1)
		}
		for _, spec := range args[1:] {
			installPkg(spec, true)
		}

	case "remove", "uninstall", "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex remove <package>")
			os.Exit(1)
		}
		for _, name := range args[1:] {
			if err := pkg.Remove(name); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
			} else {
				fmt.Printf("removed %s\n", name)
			}
		}

	case "update", "upgrade":
		if len(args) < 2 {
			// No module specified — update all deps from config.lx
			updateAll()
		} else {
			for _, spec := range args[1:] {
				updatePkg(spec)
			}
		}

	case "list", "ls":
		mods := pkg.List()
		if len(mods) == 0 {
			fmt.Println("no packages installed")
			return
		}
		fmt.Printf("%-30s %s\n", "package", "version")
		fmt.Println(strings.Repeat("─", 50))
		for _, m := range mods {
			fmt.Printf("%-30s %s\n", m.Name, m.Version)
		}

	case "build":
		if len(args) == 1 {
			runBuildFile()
		} else {
			parseBuildCommand(args[1:])
		}

	case "repl":
		if dbg.Enabled() {
			// In debug mode: show what would happen instead of hard-exiting.
			dbg.Step("repl", "debug stub — REPL is not compiled into this binary")
			dbg.StepWarn("repl skipped", "build with -tags repl to enable the interactive REPL")
			return
		}
		fmt.Fprintln(os.Stderr, "error: REPL not available in this build (build with -tags repl)")
		os.Exit(1)

	case "pack":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex pack <directory> [-o output.nax]")
			os.Exit(1)
		}
		parsePackCommand(args[1:])

	case "fmt", "format":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: lunex fmt <file.lx>")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "  Auto-formats a Lunex source file using the AST-based pretty-printer.")
			fmt.Fprintln(os.Stderr, "  Fixes: spacing, comma gaps, missing spaces before '{', semicolons, indentation.")
			fmt.Fprintln(os.Stderr, "  The file is rewritten in-place. If it is already clean, nothing changes.")
			os.Exit(1)
		}
		fmtFile(args[1])

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

func pkgLoader(name string) (string, bool) {
	resolvedPath, ok := pkg.Resolve(name)
	if !ok {
		return "", false
	}
	// pkgLoader only serves .lx source text.
	// .nax and .nc binary packages are handled by the naxLoader callback
	// that is wired up in bytecode.newRuntimeCompiler.
	ext := strings.ToLower(filepath.Ext(resolvedPath))
	if ext == ".nax" || ext == ".nc" {
		return "", false
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", false
	}
	return string(data), true
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

func parseRunOptions(extraArgs []string) (emitMode, error) {
	var emit emitMode
	for i := 0; i < len(extraArgs); i++ {
		arg := extraArgs[i]
		switch {
		case arg == "--emit":
			if i+1 >= len(extraArgs) {
				return "", fmt.Errorf("error: --emit requires a value")
			}
			emit = emitMode(strings.ToLower(extraArgs[i+1]))
			i++
		case strings.HasPrefix(arg, "--emit="):
			emit = emitMode(strings.ToLower(strings.TrimPrefix(arg, "--emit=")))
		case strings.TrimSpace(arg) == "":
			continue
		default:
			return "", fmt.Errorf("unknown flag: %s", arg)
		}
	}
	if emit != "" && emit != emitModeAST && emit != emitModeIR {
		return "", fmt.Errorf("unsupported emit mode: %s", emit)
	}
	return emit, nil
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
	emit, err := parseRunOptions(extraArgs)
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

func fmtFile(filePath string) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	original := string(source)
	formatted := compiler.Format(original)
	if formatted == original {
		fmt.Printf("%s  already formatted\n", filePath)
		return
	}
	if err := os.WriteFile(filePath, []byte(formatted), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s  reformatted  (spacing · coercions · indentation)\n", filePath)
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
	c := newCompiler()
	result := c.CompileSource(string(source), filePath)
	if result.Success {
		fmt.Printf("%s: ok\n", filePath)
	} else {
		for _, e := range result.Errors {
			fmt.Fprint(os.Stderr, errfmt.Format(e))
		}
		os.Exit(1)
	}
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

func installAll() {
	mod, err := pkg.LoadManifest(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: no config.lx found. Run 'lunex init' first.")
		os.Exit(1)
	}
	if len(mod.Dependencies) == 0 {
		fmt.Println("no dependencies to install")
		return
	}
	for name, ver := range mod.Dependencies {
		spec := name + "@" + ver
		installPkg(spec, false)
	}
}

func installPkg(spec string, _ bool) {
	mod, err := pkg.Install(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error installing %s: %v\n", spec, err)
		os.Exit(1)
	}
	fmt.Printf("installed %s@%s\n", mod.Name, mod.Version)
}

func updatePkg(spec string) {
	// Strip any pinned version so we always pull the latest.
	name := spec
	if idx := strings.Index(spec, "@"); idx > 0 {
		name = spec[:idx]
	}
	// Remove the cached copy first.
	if err := pkg.Remove(name); err != nil && !strings.Contains(err.Error(), "not found") {
		fmt.Fprintf(os.Stderr, "error removing %s: %v\n", name, err)
		os.Exit(1)
	}
	// Re-install from source (no pinned ref → pulls default branch).
	mod, err := pkg.Install(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error updating %s: %v\n", name, err)
		os.Exit(1)
	}
	fmt.Printf("updated %s@%s\n", mod.Name, mod.Version)
}

func updateAll() {
	manifest, err := pkg.LoadManifest(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: no config.lx found. Run 'lunex init' first.")
		os.Exit(1)
	}
	if len(manifest.Dependencies) == 0 {
		fmt.Println("no dependencies to update")
		return
	}
	for name := range manifest.Dependencies {
		updatePkg(name)
	}
}

func printHelp() {
	fmt.Printf("Lunex %s\n\n", meta.Version())
	fmt.Print(`Usage:
  lunex run <file> [--emit ast|ir]   run a .lx, .nc, or .nax file
  lunex -e "<code>"                  run a code snippet directly
  lunex build [file] [-o]            compile the project entry to .nc bytecode
  lunex check <file>                 check for errors without running
  lunex see_errors <file>            show detailed compile errors
  lunex fmt <file>                   format source code
  lunex dis <file.nc>                disassemble nc bytecode
  lunex init [name]                  create a new project folder
  lunex install <url>                install a package from GitHub or any URL
  lunex add <url>                    alias for install
  lunex update <name>                update an installed package to latest
  lunex update                       update all dependencies from config.lx
  lunex remove / rm <name>           remove an installed package
  lunex list                         show installed packages
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
  @import("std.io")                  standard library module
  @import("modulename")              installed external module
  @import("github.com/user/repo")    import directly from GitHub (installs on first use)
  @import("https://example.com/pkg") import from URL (sandboxed, installs on first use)
  @fimport("./mylib.nax")            import from a local .nax archive file

Installing packages:
  lunex install github.com/user/repo          from GitHub
  lunex install https://github.com/user/repo  full URL, same result
  lunex install https://example.com/mypkg     any HTTPS URL (sandboxed)
  lunex install                               install all deps from config.lx

Global flags (place before the command or file):
  --debug, -d   enable debug mode (shows every execution step on stderr)
  --verbose, -V enable verbose debug output (implies --debug)
  --no-cache    compile fresh every run; store nothing to disk or memory

Environment variables:
  NTL_DEBUG=1     enable debug mode
  LUNEX_DEBUG=1   alias for NTL_DEBUG
  LUNEX_VERBOSE=1 verbose debug output (implies NTL_DEBUG=1)

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
