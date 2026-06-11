// Package selfhosted previously contained a Lunex-in-Lunex encoder.
// It is now a no-op shim. All encoding is done directly by the Go
// encoder in internal/bytecode/format.go and nax.go.
// The embedded .lx files are kept as empty stubs for build compatibility.

package selfhosted

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed nc_encoder.lx
var ncEncoderSource string

//go:embed type_checker.lx
var typeCheckerSource string

// FileFlags holds compile-time directives parsed from a Lunex source file.
type FileFlags struct {
	TypesEnabled    bool
	LowLevelEnabled bool
}

// ParseFileFlags scans source for directive lines without running the full parser.
func ParseFileFlags(source string) FileFlags {
	var f FileFlags
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "lunex.types = on" || trimmed == "lunex.types=on" {
			f.TypesEnabled = true
		}
		if trimmed == "lunex.lowlevel = on" || trimmed == "lunex.lowlevel=on" {
			f.LowLevelEnabled = true
		}
		if len(trimmed) > 0 && trimmed[0] != '/' && !strings.HasPrefix(trimmed, "lunex.") {
			break
		}
	}
	return f
}

// NCEncoderSource returns the raw source (stub only).
func NCEncoderSource() string { return ncEncoderSource }

// TypeCheckerSource returns the raw source (stub only).
func TypeCheckerSource() string { return typeCheckerSource }

// Interpreter is the minimal interface needed from runtime.Interpreter.
type Interpreter interface {
	RunSource(source, filename string) error
	CallExport(name string, args ...interface{}) (interface{}, error)
}

// SetInterpreterFactory is a no-op; selfhosted encoding is disabled.
func SetInterpreterFactory(_ func() Interpreter) {}

// NCChunk describes a single source module to encode.
type NCChunk struct {
	Name       string
	SourceFile string
	SourceText string
	SubChunks  []NCChunk
	NTZOpcodes []byte
}

// EncodeNC always returns an error; the Go encoder in bytecode/format.go is used instead.
func EncodeNC(_ NCChunk) ([]byte, error) {
	return nil, fmt.Errorf("selfhosted: encoder disabled — pure Go encoder handles all NC encoding")
}

// NAXEntry is a single archive entry.
type NAXEntry struct {
	Name string
	Data []byte
}

// EncodeNAX always returns an error; the Go encoder in bytecode/nax.go is used instead.
func EncodeNAX(_ []NAXEntry, _ int, _ int64) ([]byte, error) {
	return nil, fmt.Errorf("selfhosted: NAX encoder disabled — pure Go encoder handles all NAX encoding")
}

// TypeError is a type error reported by the type checker.
type TypeError struct {
	Code    string
	Message string
	File    string
	Line    int
	Col     int
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("[%s] %s:%d:%d — %s", e.Code, e.File, e.Line, e.Col, e.Message)
}

// CheckTypes is a no-op; selfhosted type checking is disabled.
func CheckTypes(_ interface{}, _ string) ([]*TypeError, error) {
	return nil, nil
}
