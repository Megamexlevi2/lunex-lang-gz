package main

import (
        _ "embed"
        "errors"
        "fmt"
        "lunex/internal/aot"
        "lunex/internal/bridge"
        "lunex/internal/buildfile"
        "lunex/internal/builtin"
        "lunex/internal/bytecode"
        "lunex/internal/compiler"
        "lunex/internal/editor"
	"lunex/internal/ide"
        "lunex/internal/firstrun"
        "lunex/internal/jit"
        "lunex/internal/meta"
        "lunex/internal/pkg"
        "lunex/internal/zigrt"
        "os"
        "path/filepath"
        "strings"
        "time"
)

//go:embed version.json
var _versionJSON []byte

func init() {
        meta.SetVersionData(_versionJSON)
        zigrt.Init(embeddedZigRT)
}

func main() {
        defer zigrt.Shutdown()
        meta.Seal()
        firstrun.Check(meta.Version())
        if ncData, ok := aot.CheckEmbeddedNC(); ok {
                res, err := zigrt.RunPipe(ncData)
                if err != nil {
                        fmt.Fprintln(os.Stderr, "error:", err)
                        os.Exit(1)
                }
                if res.ExitCode != 0 {
                        os.Exit(res.ExitCode)
                }
                return
        }

        args := os.Args[1:]

        if len(args) > 0 && args[0] == "*debug" {
                os.Setenv("NTL_DEBUG", "1")
                args = args[1:]
        }

        if len(args) == 0 {
                printHelp()
                return
        }

        cmd := args[0]
        switch cmd {
        case "run":
                if len(args) < 2 {
                        fmt.Fprintln(os.Stderr, "usage: lunex run <file.lx|file.nc|file.nax>")
                        os.Exit(1)
                }
                runFile(args[1], args[2:])

        case "-e", "execute":
                if len(args) < 2 {
                        fmt.Fprintln(os.Stderr, "usage: lunex -e \"<code>\"")
                        os.Exit(1)
                }
                runString(args[1])


          case "ide":
                  subCmd := ""
                  if len(args) > 1 {
                          subCmd = args[1]
                  }
                  ideFile := ""
                  if subCmd == "run" {
                          if len(args) > 2 {
                                  ideFile = args[2]
                          }
                  } else {
                          if subCmd != "" {
                                  ideFile = subCmd
                          }
                  }
                  ide.Run(ideFile)

          case "edit":
                filePath := ""
                if len(args) > 1 {
                        filePath = args[1]
                }
                editor.Run(filePath)

        case "version", "--version", "-v":
                fmt.Printf("Lunex %s\n", meta.Version())

        case "help", "--help", "-h":
                printHelp()

        case "init":
                name := ""
                if len(args) > 1 {
                        name = args[1]
                }
                if name == "" {
                        cwd, _ := os.Getwd()
                        name = filepath.Base(cwd)
                }
                cwd, _ := os.Getwd()
                if err := pkg.InitManifest(cwd, name); err != nil {
                        fmt.Fprintln(os.Stderr, "error:", err)
                        os.Exit(1)
                }
                if err := buildfile.Generate("build.lx", name); err == nil {
                        fmt.Printf("created build.lx\n")
                }
                fmt.Printf("initialized lunex.mod for %q\n", name)

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
                fmt.Fprintln(os.Stderr, "error: REPL not available in this build")
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
                        os.Exit(1)
                }
                fmtFile(args[1])

        case "see_errors", "see-errors", "errors":
                if len(args) < 2 {
                        fmt.Fprintln(os.Stderr, "usage: lunex see_errors <file.lx>")
                        os.Exit(1)
                }
                seeErrors(args[1])

        case "check":
                if len(args) < 2 {
                        fmt.Fprintln(os.Stderr, "usage: lunex check <file.lx>")
                        os.Exit(1)
                }
                checkFile(args[1])

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

        case "rt-info":
                if err := zigrt.Info(); err != nil {
                        fmt.Fprintln(os.Stderr, "error:", err)
                        os.Exit(1)
                }

        default:
                ext := strings.ToLower(filepath.Ext(cmd))
                if ext == ".lx" || ext == ".nc" || ext == ".nax" {
                        runFile(cmd, args[1:])
                } else {
                        fmt.Fprintf(os.Stderr, "unknown command: %s\nRun 'lunex help' for usage.\n", cmd)
                        os.Exit(1)
                }
        }
}

func newCompiler() *compiler.Compiler {
        c := compiler.New(compiler.DefaultOptions)
        builtin.RegisterAll(c)
        c.Interpreter().SetNTLLoader(loadLib)
        c.Interpreter().SetPkgLoader(pkgLoader)
        return c
}

func pkgLoader(name string) (string, bool) {
        path, ok := pkg.Resolve(name)
        if !ok {
                return "", false
        }
        data, err := os.ReadFile(path)
        if err != nil {
                return "", false
        }
        return string(data), true
}

func loadLib(name string) (string, bool) {
        if src, ok := loadEmbeddedLib(name); ok {
                return src, true
        }
        return pkgLoader(name)
}

func runString(source string) {
        c := newCompiler()
        result := c.CompileSource(source, "<eval>")
        if !result.Success {
                for _, e := range result.Errors {
                        fmt.Fprintf(os.Stderr, "%v\n", e)
                }
                os.Exit(1)
        }
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
        execNC(ncData)
}

func runFile(filePath string, extraArgs []string) {
        _ = extraArgs
        ext := strings.ToLower(filepath.Ext(filePath))
        switch ext {
        case ".nc":
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
                entryData, err := bytecode.ExtractNAXEntry(data)
                if err != nil {
                        fmt.Fprintln(os.Stderr, "error:", err)
                        os.Exit(1)
                }
                execNC(entryData)
        default:
                if !strings.HasSuffix(filePath, ".lx") && ext == "" {
                        filePath += ".lx"
                }
                absPath, err := filepath.Abs(filePath)
                if err != nil {
                        fmt.Fprintf(os.Stderr, "error resolving path: %v\n", err)
                        os.Exit(1)
                }
                runNTLWithCache(absPath)
        }
}

func runNTLWithCache(absPath string) {
        if cached, ok := bytecode.CacheLookup(absPath); ok {
                execNC(cached)
                return
        }

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
                        fmt.Fprintf(os.Stderr, "%v\n", e)
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

        _ = bytecode.CacheStore(absPath, ncData)
        execNC(ncData)
}

func execNC(ncData []byte) {
        if zigrt.Available() {
                res, err := zigrt.RunPipe(ncData)
                if err == nil {
                        if res.ExitCode != 0 {
                                os.Exit(res.ExitCode)
                        }
                        return
                }
                var ze *bridge.ZigError
                if errors.As(err, &ze) && ze.Frame.Code != bridge.CodeBadBCFormat {
                        fmt.Fprintln(os.Stderr, "error:", err)
                        os.Exit(1)
                }
        }
        if err := bytecode.RunNC(ncData, loadLib, pkgLoader); err != nil {
                fmt.Fprintln(os.Stderr, "error:", err)
                os.Exit(1)
        }
}

func emitASM(filePath string) {
        if !strings.HasSuffix(filePath, ".lx") {
                filePath += ".lx"
        }
        source, err := os.ReadFile(filePath)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error reading %s: %v\n", filePath, err)
                os.Exit(1)
        }
        asmText, err := aot.EmitASM(string(source), filePath)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }
        fmt.Print(asmText)
}

func runBuildFile() {
        bfPath, ok := buildfile.Find()
        if !ok {
                fmt.Fprintln(os.Stderr, "error: no build.lx found in current directory")
                fmt.Fprintln(os.Stderr, "  run 'lunex init' to create one, or specify a file:")
                fmt.Fprintln(os.Stderr, "  lunex build <file.lx>")
                os.Exit(1)
        }

        cfg, err := buildfile.Parse(bfPath)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error reading %s: %v\n", bfPath, err)
                os.Exit(1)
        }

        fmt.Printf("Lunex %s  build.lx\n", meta.Version())
        fmt.Printf("  name:    %s\n", cfg.Name)
        fmt.Printf("  version: %s\n", cfg.Version)
        fmt.Printf("  entry:   %s\n", cfg.Entry)
        fmt.Printf("  output:  %s\n", cfg.Output)
        fmt.Println()

        absEntry, err := filepath.Abs(cfg.Entry)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }
        if _, err := os.Stat(absEntry); err != nil {
                fmt.Fprintf(os.Stderr, "error: entry file %s not found\n", cfg.Entry)
                os.Exit(1)
        }

        if err := os.MkdirAll(cfg.Output, 0755); err != nil {
                fmt.Fprintf(os.Stderr, "error: cannot create output dir %s\n", cfg.Output)
                os.Exit(1)
        }

        source, err := os.ReadFile(absEntry)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }
        srcText := string(source)

        fmt.Printf("  [1/2] parse + compile    %s\n", cfg.Entry)
        c := newCompiler()
        result := c.CompileSource(srcText, absEntry)
        if !result.Success {
                for _, e := range result.Errors {
                        fmt.Fprintf(os.Stderr, "%v\n", e)
                }
                os.Exit(1)
        }

        fmt.Printf("  [2/2] emit bytecode\n")
        chunk := &bytecode.ExportedChunk{
                Name:       cfg.Name,
                SourceFile: absEntry,
                SourceText: srcText,
        }
        ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }

        ncPath := filepath.Join(cfg.Output, cfg.Name+".nc")
        if err := os.WriteFile(ncPath, ncData, 0644); err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }

        fmt.Printf("\n  %s → %s\n", cfg.Entry, ncPath)
        fmt.Println()
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
                        fmt.Fprintln(os.Stderr, "  usage: lunex build <file.lx> [-o output.nc]")
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
        baseName := strings.TrimSuffix(filepath.Base(inputFile), ".lx")

        if outputFile == "" {
                outputFile = baseName + ".nc"
        }

        buildNC(absInput, inputFile, outputFile)
}

func buildNC(absInput, inputFile, outputFile string) {
        source, err := os.ReadFile(absInput)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error reading %s: %v\n", inputFile, err)
                os.Exit(1)
        }
        srcText := string(source)
        c := newCompiler()
        result := c.CompileSource(srcText, absInput)
        if !result.Success {
                for _, e := range result.Errors {
                        fmt.Fprintf(os.Stderr, "%v\n", e)
                }
                os.Exit(1)
        }
        chunk := &bytecode.ExportedChunk{
                Name:       strings.TrimSuffix(filepath.Base(inputFile), ".lx"),
                SourceFile: absInput,
                SourceText: srcText,
        }
        ncData, err := bytecode.EncodeExportedWithAST(chunk, result.AST)
        if err != nil {
                fmt.Fprintf(os.Stderr, "error encoding: %v\n", err)
                os.Exit(1)
        }
        if err := os.WriteFile(outputFile, ncData, 0644); err != nil {
                fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outputFile, err)
                os.Exit(1)
        }
        fi, _ := os.Stat(outputFile)
        sz := int64(0)
        if fi != nil {
                sz = fi.Size()
        }
        fmt.Printf("%s → %s (%d KB)\n", inputFile, outputFile, sz/1024)
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
        formatted := compiler.Format(string(source))
        if err := os.WriteFile(filePath, []byte(formatted), 0644); err != nil {
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(1)
        }
        fmt.Printf("formatted %s\n", filePath)
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
                fmt.Fprintf(os.Stderr, "%v\n", e)
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
                        fmt.Fprintf(os.Stderr, "%v\n", e)
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
        fmt.Print(chunk.Disassemble())
}

func showCacheInfo() {
        dir := bytecode.CacheDir()
        fmt.Printf("cache dir:   %s\n", dir)
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
        fmt.Printf("Go interpreter:   available\n")
        if zigrt.Available() {
                fmt.Printf("Zig runtime:      available\n")
        } else {
                fmt.Printf("Zig runtime:      not available\n")
        }
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
                        fmt.Fprintf(os.Stderr, "%v\n", e)
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

func installAll() {
        mod, err := pkg.LoadManifest(".")
        if err != nil {
                fmt.Fprintln(os.Stderr, "error: no lunex.mod found. Run 'lunex init' first.")
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

func parseSpec(spec string) (name, version string) {
        idx := strings.LastIndex(spec, "@")
        if idx < 1 {
                return spec, ""
        }
        return spec[:idx], spec[idx+1:]
}

func printHelp() {
        fmt.Printf("Lunex %s\n\n", meta.Version())
        fmt.Print(`Usage:
  lunex ide run [file]       launch Lunex IDE (full terminal IDE)
  lunex run <file>           run a .lx, .nc, or .nax file
  lunex -e "<code>"          run a snippet directly
  lunex build [file] [-o]   compile to .nc bytecode
  lunex check <file>         type-check without running
  lunex fmt <file>           format source code
  lunex dis <file.nc>        disassemble bytecode
  lunex init [name]          create a new project
  lunex install / add / rm   manage packages
  lunex list                 show installed packages
  lunex pack <dir>           bundle a directory to .nax
  lunex cache [clear]        manage the bytecode cache
  lunex runtimes             show available execution engines
  lunex bench <file>         run with timing output
  lunex version              print version
  lunex help                 show this help

`)
        _ = jit.JITCacheDir
}
