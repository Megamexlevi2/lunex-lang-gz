// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package bytecode

import (
	"fmt"
	"strings"
)

type Chunk struct {
	Name       string
	SourceFile string
	SourceText string
	SubChunks  []*Chunk
}

func NewChunk(name string) *Chunk {
	return &Chunk{Name: name}
}

func (c *Chunk) Disassemble() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== module: %s ===\n", c.Name))
	sb.WriteString(fmt.Sprintf("source: %s\n", c.SourceFile))
	sb.WriteString(fmt.Sprintf("size: %d bytes (encoded)\n", len(c.SourceText)))
	return sb.String()
}
