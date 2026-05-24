// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package bytecode

import (
	"fmt"
	"lunex/internal/builtin"
	"lunex/internal/compiler"
	"os"
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

	entryName := arch.MainEntry
	if entryName == "" && len(arch.Entries) > 0 {
		entryName = arch.Entries[0].Name
	}

	for _, entry := range arch.Entries {
		if entry.Name == entryName {
			return RunNC(entry.Data, ntlLoader, pkgLoader)
		}
	}
	return fmt.Errorf("entry not found in archive: %s", entryName)
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

func newRuntimeCompiler(ntlLoader func(string) (string, bool), pkgLoader func(string) (string, bool)) *compiler.Compiler {
	c := compiler.New(compiler.DefaultOptions)
	builtin.RegisterAll(c)
	if ntlLoader != nil {
		c.Interpreter().SetNTLLoader(ntlLoader)
	}
	if pkgLoader != nil {
		c.Interpreter().SetPkgLoader(pkgLoader)
	}
	return c
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
