package bytecode

import (
	"fmt"
	"lunex/internal/compiler"
	"lunex/internal/runtime"
	"lunex/internal/std"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func RunNC(data []byte, ntlLoader func(string) (string, bool), pkgLoader func(string) (string, bool)) error {
	chunk, err := DecodeNC(data)
	if err != nil {
		return fmt.Errorf("cannot load object: %w", err)
	}
	c := newRuntimeCompiler(ntlLoader, pkgLoader)
	return c.RunSource(chunk.SourceText, chunk.SourceFile)
}

func RunNAX(data []byte, ntlLoader func(string) (string, bool), pkgLoader func(string) (string, bool)) error {
	arch, err := decodeNAX(data)
	if err != nil {
		return fmt.Errorf("cannot load archive: %w", err)
	}

	if len(arch.Entries) == 0 {
		return fmt.Errorf("entry not found in archive: <empty>")
	}

	idx := int(arch.MainIndex)
	if idx < 0 || idx >= len(arch.Entries) {
		idx = 0
	}

	archiveSources := make(map[string]string)
	addArchiveSource := func(name, src string) {
		key := normalizeArchiveKey(name)
		if key != "" {
			archiveSources[key] = src
		}
		if strings.HasSuffix(strings.ToLower(key), ".lx") {
			archiveSources[normalizeArchiveKey(strings.TrimSuffix(key, ".lx"))] = src
		}
	}
	for _, e := range arch.Entries {
		lower := strings.ToLower(e.Name)
		switch {
		case strings.HasSuffix(lower, ".lx"):
			addArchiveSource(e.Name, string(e.Data))
		case strings.HasSuffix(lower, ".nc"):
			if chunk, err := DecodeNC(e.Data); err == nil {
				addArchiveSource(strings.TrimSuffix(e.Name, ".nc")+".lx", chunk.SourceText)
			}
		}
	}

	archiveLoader := func(name string) (string, bool) {
		if src, ok := archiveSources[normalizeArchiveKey(name)]; ok {
			return src, true
		}
		if !strings.HasSuffix(strings.ToLower(name), ".lx") {
			if src, ok := archiveSources[normalizeArchiveKey(name+".lx")]; ok {
				return src, true
			}
		}
		return "", false
	}

	combinedLoader := archiveLoader
	if ntlLoader != nil {
		combinedLoader = func(name string) (string, bool) {
			if src, ok := archiveLoader(name); ok {
				return src, true
			}
			return ntlLoader(name)
		}
	}

	return RunNC(arch.Entries[idx].Data, combinedLoader, pkgLoader)
}

func RunNAXFile(path string, ntlLoader func(string) (string, bool), pkgLoader func(string) (string, bool)) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	return RunNAX(data, ntlLoader, pkgLoader)
}

func RunNCFile(path string, ntlLoader func(string) (string, bool), pkgLoader func(string) (string, bool)) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}
	return RunNC(data, ntlLoader, pkgLoader)
}

// LoadNAXAsModule decodes a .nax archive and executes its main entry as a module,
// returning the exported Value. Used by the interpreter's NaxLoader callback.
func LoadNAXAsModule(filePath string, c *compiler.Compiler) (*runtime.Value, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", filePath, err)
	}

	arch, err := decodeNAX(data)
	if err != nil {
		return nil, fmt.Errorf("cannot decode archive %s: %w", filePath, err)
	}

	if len(arch.Entries) == 0 {
		return nil, fmt.Errorf("archive %s has no entries", filePath)
	}

	// Build a source map so inner @imports within the archive work.
	archiveSources := make(map[string]string)
	for _, e := range arch.Entries {
		lower := strings.ToLower(e.Name)
		switch {
		case strings.HasSuffix(lower, ".lx"):
			key := normalizeArchiveKey(e.Name)
			archiveSources[key] = string(e.Data)
			// Also index without extension for bare-name imports.
			archiveSources[normalizeArchiveKey(strings.TrimSuffix(e.Name, ".lx"))] = string(e.Data)
		case strings.HasSuffix(lower, ".nc"):
			if chunk, err2 := DecodeNC(e.Data); err2 == nil {
				key := normalizeArchiveKey(strings.TrimSuffix(e.Name, ".nc"))
				archiveSources[key] = chunk.SourceText
				archiveSources[normalizeArchiveKey(strings.TrimSuffix(e.Name, ".nc")+".lx")] = chunk.SourceText
			}
		}
	}

	archiveLoader := func(name string) (string, bool) {
		k := normalizeArchiveKey(name)
		if src, ok := archiveSources[k]; ok {
			return src, true
		}
		if !strings.HasSuffix(strings.ToLower(name), ".lx") {
			if src, ok := archiveSources[normalizeArchiveKey(name+".lx")]; ok {
				return src, true
			}
		}
		return "", false
	}

	// Merge archive loader with the compiler's existing ntl loader.
	prev := c.Interpreter().NTLLoader()
	combinedLoader := archiveLoader
	if prev != nil {
		combinedLoader = func(name string) (string, bool) {
			if src, ok := archiveLoader(name); ok {
				return src, true
			}
			return prev(name)
		}
	}
	c.Interpreter().SetNTLLoader(combinedLoader)
	defer func() { c.Interpreter().SetNTLLoader(prev) }()

	// Execute the main entry as a module and return the exports.
	idx := int(arch.MainIndex)
	if idx < 0 || idx >= len(arch.Entries) {
		idx = 0
	}
	mainEntry := arch.Entries[idx]
	var src string
	switch strings.ToLower(filepath.Ext(mainEntry.Name)) {
	case ".nc":
		chunk, err2 := DecodeNC(mainEntry.Data)
		if err2 != nil {
			return nil, fmt.Errorf("cannot decode main entry in %s: %w", filePath, err2)
		}
		src = chunk.SourceText
	default:
		src = string(mainEntry.Data)
	}

	// Use a synthetic filename rooted at the module directory (not the .nax
	// path itself) so that resolveLocalFile resolves @fimport("./src/node.lx")
	// relative to the module's own directory, where the sub-files live.
	moduleFilename := filepath.Join(filepath.Dir(filePath), mainEntry.Name)
	return c.RunSourceAsModule(src, moduleFilename)
}

// LoadNCAsModule decodes a .nc bytecode file and executes it as a module.
func LoadNCAsModule(filePath string, c *compiler.Compiler) (*runtime.Value, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", filePath, err)
	}
	chunk, err := DecodeNC(data)
	if err != nil {
		return nil, fmt.Errorf("cannot decode %s: %w", filePath, err)
	}
	return c.RunSourceAsModule(chunk.SourceText, filePath)
}

func newRuntimeCompiler(ntlLoader func(string) (string, bool), pkgLoader func(string) (string, bool)) *compiler.Compiler {
	c := compiler.New(compiler.DefaultOptions)
	std.RegisterAll(c)
	if ntlLoader != nil {
		c.Interpreter().SetNTLLoader(ntlLoader)
	}
	if pkgLoader != nil {
		c.Interpreter().SetPkgLoader(pkgLoader)
	}
	// Wire the NaxLoader so @fimport("./file.nax") and @fimport("./file.nc") work.
	c.Interpreter().SetNaxLoader(func(absPath string) (*runtime.Value, error) {
		ext := strings.ToLower(filepath.Ext(absPath))
		switch ext {
		case ".nax":
			return LoadNAXAsModule(absPath, c)
		case ".nc":
			return LoadNCAsModule(absPath, c)
		default:
			return nil, fmt.Errorf("unsupported binary module extension: %s", ext)
		}
	})
	return c
}

func normalizeArchiveKey(name string) string {
	key := strings.TrimSpace(name)
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "./")
	key = strings.TrimPrefix(key, "/")
	key = path.Clean(key)
	if key == "." {
		return ""
	}
	return key
}

func BuildNCFile(sourcePath string, outputPath string) error {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", sourcePath, err)
	}
	srcText := string(source)

	absSource, _ := filepath.Abs(sourcePath)

	c := newRuntimeCompiler(nil, nil)
	result := c.CompileSource(srcText, absSource)
	if !result.Success {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("compile error: %s", strings.Join(msgs, "; "))
	}

	chunk := &Chunk{
		Name:       filepath.Base(sourcePath),
		SourceFile: absSource,
		SourceText: srcText,
	}

	nc, err := EncodeNC(chunk)
	if err != nil {
		return fmt.Errorf("encode error: %w", err)
	}

	return os.WriteFile(outputPath, nc, 0644)
}
