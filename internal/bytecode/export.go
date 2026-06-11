// Lunex lang — ExportedChunk and .nc encoding helpers.
//
// ExportedChunk is the high-level type the Go compiler fills in after
// parsing a .lx file.  EncodeExportedWithAST then:
//
//   1. Packs source metadata into a .nc container (Go's job).
//   2. Returns the raw bytes — ready to be cached on disk or executed by
//      the Go interpreter.
//
// The NTZ bytecode system has been removed.  Go's tree-walking interpreter
// handles ALL Lunex execution.  Zig is a background JIT optimization service
// that compiles NaxerIR (emitted by jit/naxer_ir.go) for hot functions;
// it never executes Lunex code and never receives .nc containers.
//
// The NTZOpcodes field is kept for backward compatibility with .nc files
// produced by older Lunex versions (read path only; never written).

package bytecode

import "lunex/internal/ast"

// ExportedChunk describes the content of a compiled Lunex module.
type ExportedChunk struct {
	Name       string
	SourceFile string
	SourceText string
	// NTZOpcodes is kept for reading legacy .nc files only.
	// New .nc files are produced without an NTZ section.
	NTZOpcodes []byte
}

// EncodeExported encodes an ExportedChunk into the NC binary format.
// The NTZ section is omitted — the Go interpreter handles all execution.
func EncodeExported(e *ExportedChunk) ([]byte, error) {
	chunk := &Chunk{
		Name:       e.Name,
		SourceFile: e.SourceFile,
		SourceText: e.SourceText,
	}
	// Always pass nil for the NTZ section.
	// Zig no longer executes Lunex code; the NTZ section is not needed.
	return encodeNCWithNTZ(chunk, nil)
}

// EncodeExportedWithAST encodes an ExportedChunk into the NC binary format.
// The AST is accepted for API compatibility but NTZ compilation is skipped —
// Go's interpreter uses the embedded source text, not NTZ opcodes.
func EncodeExportedWithAST(e *ExportedChunk, _ *ast.Node) ([]byte, error) {
	return EncodeExported(e)
}
