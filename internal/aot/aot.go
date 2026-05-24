package aot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"lunex/internal/asm"
	"lunex/internal/ast"
	"lunex/internal/enfs"
	"lunex/internal/lexer"
	"lunex/internal/ntir"
	"lunex/internal/parser"
	"os"
	"time"
)
// ahh bugs Don't use this, it's very buggy.  

const embeddedMarker = "Lunex\xffCOMPILED\x00"

type BuildOptions struct {
	OutputPath string
	Optimize   bool
	Debug      bool
	UseCache   bool
	Verbose    bool
	DumpASM    string
}

type BuildResult struct {
	OutputPath string
	Size       int64
	Duration   time.Duration
	FuncCount  int
}

func DefaultBuildOptions(output string) BuildOptions {
	return BuildOptions{
		OutputPath: output,
		Optimize:   true,
		UseCache:   true,
		Verbose:    false,
	}
}

func Build(source, sourceFile string, opts BuildOptions) (*BuildResult, error) {
	t0 := time.Now()

	tokens, err := lexer.Tokenize(source, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("lex error: %w", err)
	}
	tree, err := parser.Parse(tokens, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	module, err := LowerToIR(tree, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("IR lowering error: %w", err)
	}

	opt := enfs.Extreme()
	opt.Optimize(module)

	asmText := asm.EmitModule(module)

	if opts.DumpASM != "" {
		_ = os.WriteFile(opts.DumpASM, []byte(asmText), 0644)
	}

	outPath := opts.OutputPath
	if outPath == "" {
		outPath = sourceFile + ".nasm"
	}

	if err := os.WriteFile(outPath, []byte(asmText), 0644); err != nil {
		return nil, fmt.Errorf("write asm: %w", err)
	}

	info, _ := os.Stat(outPath)
	sz := int64(0)
	if info != nil {
		sz = info.Size()
	}

	return &BuildResult{
		OutputPath: outPath,
		Size:       sz,
		Duration:   time.Since(t0),
		FuncCount:  len(module.Funcs),
	}, nil
}

func BuildFile(sourcePath string, opts BuildOptions) (*BuildResult, error) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", sourcePath, err)
	}
	return Build(string(source), sourcePath, opts)
}

func LowerToIR(tree *ast.Node, sourceFile string) (*ntir.Module, error) {
	module := ntir.NewModule(sourceFile)
	module.SourceFile = sourceFile
	builder := ntir.NewBuilder(module)
	builder.BuildModule(tree)
	if errs := builder.Errors(); len(errs) > 0 {
		return nil, fmt.Errorf("IR build errors: %v", errs)
	}
	return module, nil
}

func EmitASM(source, sourceFile string) (string, error) {
	tokens, err := lexer.Tokenize(source, sourceFile)
	if err != nil {
		return "", fmt.Errorf("lex: %w", err)
	}
	tree, err := parser.Parse(tokens, sourceFile)
	if err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	module, err := LowerToIR(tree, sourceFile)
	if err != nil {
		return "", fmt.Errorf("IR: %w", err)
	}
	opt := enfs.Extreme()
	opt.Optimize(module)
	return asm.EmitModule(module), nil
}

func CheckEmbeddedNC() ([]byte, bool) {
	selfPath, err := os.Executable()
	if err != nil {
		return nil, false
	}
	return readEmbedded(selfPath)
}

func readEmbedded(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	marker := []byte(embeddedMarker)
	idx := bytes.LastIndex(data, marker)
	if idx < 0 {
		return nil, false
	}
	hdrEnd := idx + len(marker)
	if hdrEnd+8 > len(data) {
		return nil, false
	}
	ncLen := binary.BigEndian.Uint64(data[hdrEnd : hdrEnd+8])
	start := hdrEnd + 8
	if uint64(start)+ncLen > uint64(len(data)) {
		return nil, false
	}
	out := make([]byte, ncLen)
	copy(out, data[start:start+int(ncLen)])
	return out, true
}

func BuildBinary(ncData []byte, outputPath string) error {
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate lunex runtime: %w", err)
	}
	return BuildBinaryFrom(ncData, outputPath, selfPath)
}

func BuildBinaryFrom(ncData []byte, outputPath string, basePath string) error {
	base, err := os.ReadFile(basePath)
	if err != nil {
		return fmt.Errorf("cannot read runtime: %w", err)
	}
	marker := []byte(embeddedMarker)
	if idx := bytes.LastIndex(base, marker); idx >= 0 {
		base = base[:idx]
	}
	var buf bytes.Buffer
	buf.Grow(len(base) + len(marker) + 8 + len(ncData))
	buf.Write(base)
	buf.Write(marker)
	lenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBuf, uint64(len(ncData)))
	buf.Write(lenBuf)
	buf.Write(ncData)
	if err := os.WriteFile(outputPath, buf.Bytes(), 0755); err != nil {
		return fmt.Errorf("cannot write binary: %w", err)
	}
	_ = os.Chmod(outputPath, 0755)
	return nil
}

func StripEmbedded(binaryPath string) error {
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return err
	}
	marker := []byte(embeddedMarker)
	idx := bytes.LastIndex(data, marker)
	if idx < 0 {
		return nil
	}
	return os.WriteFile(binaryPath, data[:idx], 0755)
}
