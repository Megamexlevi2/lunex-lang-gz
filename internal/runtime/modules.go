// David Dev — (c) 2026. Licensed under the Mozilla Public License 2.0.

package runtime

import (
	"fmt"
	"lunex/internal/ast"
	"lunex/internal/errfmt"
	"lunex/internal/lexer"
	"lunex/internal/parser"
	"os"
	"path/filepath"
	"strings"
)

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
		// Bundled-source fallback (e.g. inside a running .nax): use ntlFileLoader
		// so interp.filename is set to the real path for nested relative imports.
		if interp.ntlFileLoader != nil {
			if src, realPath, ok := interp.ntlFileLoader(path); ok && realPath != "" {
				abs := realPath
				if !filepath.IsAbs(abs) {
					abs, _ = filepath.Abs(realPath)
				}
				return interp.evalModuleSourceFile(src, abs, abs)
			}
		}
		if interp.ntlLoader != nil {
			if src, ok := interp.ntlLoader(path); ok {
				return interp.evalModuleSourceFile(src, path, path)
			}
		}
		e := interp.runtimeError(errfmt.KindImport, "E0010F",
			fmt.Sprintf("local file %q not found", path), node, nil)
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

	// Try ntlFileLoader first: it returns (src, realAbsPath) so the interpreter
	// sets interp.filename to the real on-disk path. This is critical for
	// packages installed in ~/.lunex/packages — their @fimport("./x.lx") calls
	// must resolve relative to the package directory, not the working directory.
	if interp.ntlFileLoader != nil {
		src, realPath, ok := interp.ntlFileLoader(resolved)
		if ok && realPath != "" {
			abs := realPath
			if !filepath.IsAbs(abs) {
				abs, _ = filepath.Abs(realPath)
			}
			// Deduplication by real path.
			interp.mu.RLock()
			if mod, ok2 := interp.modules[abs]; ok2 {
				interp.mu.RUnlock()
				return mod, nil
			}
			interp.mu.RUnlock()
			return interp.evalModuleSourceFile(src, abs, abs)
		}
	}

	// Local file resolution: handles .lx files on disk (relative or absolute paths).
	// resolveLocalFile uses interp.filename as the base for relative paths.
	if localPath, ok := interp.resolveLocalFile(path); ok {
		abs, _ := filepath.Abs(localPath)
		interp.mu.RLock()
		if mod, ok := interp.modules[abs]; ok {
			interp.mu.RUnlock()
			return mod, nil
		}
		interp.mu.RUnlock()
		return interp.loadLocalFile(localPath, nil)
	}

	// Fallback source loader for bundled stdlib archives.
	if interp.ntlLoader != nil {
		src, ok := interp.ntlLoader(resolved)
		if ok {
			return interp.evalModuleSourceFile(src, resolved, resolved)
		}
	}

	e := interp.runtimeError(errfmt.KindImport, "E0010",
		fmt.Sprintf("module %q not found", path), nil, nil)
	return nil, e
}

// resolveLocalFile tries to find a local source or binary file for the given import path.
// It checks, in order:
//  1. path as-is (absolute or already has a known extension)
//  2. path + each known extension: .lx, .nax, .nc
//  3. Project-local package cache under .lunex/cache/modules
//  4. Same directory as the currently executing file
//  5. Current working directory
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

	// Project-local package cache under .lunex/cache/modules.
	// This lets @import("lune-xml") resolve to the installed package archive
	// instead of only to a source string, so nested relative imports keep working.
	if wd, err := os.Getwd(); err == nil {
		cacheRoot := filepath.Join(wd, ".lunex", "cache", "modules")
		if entries, err := os.ReadDir(cacheRoot); err == nil {
			nameKey := strings.ReplaceAll(path, "\\", "/")
			prefix := strings.ReplaceAll(nameKey, "/", "__") + "@"
			suffix := "__" + strings.ReplaceAll(nameKey, "/", "__") + "@"
			entryFiles := []string{"index.nax", "index.lx", "main.nax", "main.lx", "config.lx"}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				dirName := e.Name()
				if !strings.HasPrefix(dirName, prefix) && !strings.Contains(dirName, suffix) {
					continue
				}
				pkgDir := filepath.Join(cacheRoot, dirName)
				if data, err := os.ReadFile(filepath.Join(pkgDir, ".lunex-entry")); err == nil {
					entryName := strings.TrimSpace(string(data))
					if entryName != "" {
						fp := filepath.Join(pkgDir, entryName)
						if st, err := os.Stat(fp); err == nil && !st.IsDir() {
							return fp, true
						}
					}
				}
				for _, file := range entryFiles {
					fp := filepath.Join(pkgDir, file)
					if st, err := os.Stat(fp); err == nil && !st.IsDir() {
						return fp, true
					}
				}
				if files, err := os.ReadDir(pkgDir); err == nil {
					for _, f := range files {
						if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".lx") {
							return filepath.Join(pkgDir, f.Name()), true
						}
					}
				}
			}
		}
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
	prevLines := interp.sourceLines
	prevLine := interp.currentLine
	prevCol := interp.currentCol
	interp.filename = cacheKey
	defer func() {
		interp.filename = prevFilename
		interp.sourceLines = prevLines
		// Restore position so module-loading side effects do not corrupt the
		// caller's error position tracking.
		interp.currentLine = prevLine
		interp.currentCol = prevCol
	}()

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

	// Temporarily replace sourceLines with this module's lines so that parse
	// and load-time errors show the correct source context.
	prevLines := interp.sourceLines
	interp.sourceLines = lines
	defer func() { interp.sourceLines = prevLines }()

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
		_ = mod
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
