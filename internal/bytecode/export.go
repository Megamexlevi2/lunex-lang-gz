// Lunex lang — ExportedChunk and NC encoding helpers.
// ExportedChunk is the high-level structure used by the Go front-end to
// produce NC files.  When NTZOpcodes is populated the bytes are stored in
// the NC file's NTZ section so the Zig runtime can execute them directly,
// making Zig the primary execution engine for supported programs.

package bytecode

import "lunex/internal/ast"

// ExportedChunk describes the content of a compiled Lunex module.
type ExportedChunk struct {
	Name       string
	SourceFile string
	SourceText string
	// NTZOpcodes, when non-nil, holds Zig VM-compatible bytecode compiled from
	// the parsed AST.  The Zig runtime executes these directly; the Go
	// interpreter is used as a fallback when this field is empty.
	NTZOpcodes []byte
}

// EncodeExported encodes an ExportedChunk into the NC binary format.
// If e.NTZOpcodes is non-nil they are included as the NTZ section.
func EncodeExported(e *ExportedChunk) ([]byte, error) {
	chunk := &Chunk{
		Name:       e.Name,
		SourceFile: e.SourceFile,
		SourceText: e.SourceText,
	}
	return encodeNCWithNTZ(chunk, e.NTZOpcodes)
}

// EncodeExportedWithAST encodes an ExportedChunk into the NC binary format,
// attempting to compile the provided AST to NTZ opcodes so the Zig VM can
// execute them.  If NTZ compilation fails (unsupported feature) the NC file
// is produced without an NTZ section and Go's interpreter handles execution.
func EncodeExportedWithAST(e *ExportedChunk, tree *ast.Node) ([]byte, error) {
	if tree != nil && len(e.NTZOpcodes) == 0 {
		if ops, err := CompileNTZ(tree); err == nil {
			e.NTZOpcodes = ops
		}
	}
	return EncodeExported(e)
}
