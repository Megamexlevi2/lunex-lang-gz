// Lunex lang — NaxerIR emitter for the Go→Zig JIT pipeline.
//
// This file walks a Lunex function body (AST subtree) and emits the
// NXCH wire format that the Zig JIT service decodes and passes to
// naxer_compile().
//
// Architecture:
//   Go interpreter executes all Lunex code.
//   When the profiler marks a function as hot (>= HotThreshold calls),
//   Go calls CompileNaxerIR() to emit Naxer IR for that function.
//   The IR is sent to the Zig subprocess via MSG_JIT_COMPILE (0x01).
//   Zig runs naxer_compile() and returns native code via MSG_JIT_RESULT (0x81).
//
// NXCH wire format (what Go sends to Zig):
//
//   [4]  magic: 'N' 'X' 'C' 'H'
//   [4]  code_len: u32 LE
//   [code_len] opcode stream (see Naxer opcode constants below)
//   [4]  const_count: u32 LE
//   [const_count * 8] int64 constants, LE (indexed by NX_CONST_INT/NX_CONST_F64)
//   [4]  str_count: u32 LE
//   for each string:
//     [4] str_len u32 LE
//     [str_len] UTF-8 bytes
//   [4]  local_count: u32 LE
//   [4]  name_len: u32 LE
//   [name_len] function name bytes
//
// Naxer opcode encoding (each opcode is 1 byte followed by operands):
//   NX_CONST_INT  0x01  + [4] const_idx u32 LE
//   NX_CONST_F64  0x02  + [4] const_idx u32 LE  (value stored as bit-cast i64)
//   NX_CONST_BOOL 0x03  + [1] 0x00 or 0x01
//   NX_CONST_NULL 0x04
//   NX_LOAD       0x05  + [4] slot u32 LE
//   NX_STORE      0x06  + [4] slot u32 LE
//   NX_LOAD_GLOB  0x07  + [4] slot u32 LE
//   NX_STORE_GLOB 0x08  + [4] slot u32 LE
//   NX_ADD        0x10  (no operands)
//   NX_SUB        0x11
//   NX_MUL        0x12
//   NX_DIV        0x13
//   NX_MOD        0x14
//   NX_NEG        0x15
//   NX_EQ         0x20
//   NX_NEQ        0x21
//   NX_LT         0x22
//   NX_LTE        0x23
//   NX_GT         0x24
//   NX_GTE        0x25
//   NX_BIT_AND    0x28
//   NX_BIT_OR     0x29
//   NX_LNOT       0x2A
//   NX_BIT_NOT    0x2B
//   NX_SHL        0x2C
//   NX_SHR        0x2D
//   NX_JUMP       0x30  + [4] target_pc u32 LE
//   NX_JUMP_TRUE  0x31  + [4] target_pc u32 LE
//   NX_JUMP_FALSE 0x32  + [4] target_pc u32 LE
//   NX_CALL_RT    0x38  + [4] str_idx u32 LE   (string table index for built-in name)
//   NX_CALL       0x39  + [4] argc u32 LE
//   NX_RET        0x3A
//   NX_POP        0xFE
//   NX_HALT       0xFF

package jit

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"lunex/internal/ast"
)

// ─── Naxer opcode constants ────────────────────────────────────────────────────

const (
	nxConstInt  = uint8(0x01)
	nxConstF64  = uint8(0x02)
	nxConstBool = uint8(0x03)
	nxConstNull = uint8(0x04)
	nxLoad      = uint8(0x05)
	nxStore     = uint8(0x06)
	nxLoadGlob  = uint8(0x07)
	nxStoreGlob = uint8(0x08)
	nxAdd       = uint8(0x10)
	nxSub       = uint8(0x11)
	nxMul       = uint8(0x12)
	nxDiv       = uint8(0x13)
	nxMod       = uint8(0x14)
	nxNeg       = uint8(0x15)
	nxEq        = uint8(0x20)
	nxNeq       = uint8(0x21)
	nxLt        = uint8(0x22)
	nxLte       = uint8(0x23)
	nxGt        = uint8(0x24)
	nxGte       = uint8(0x25)
	nxBitAnd    = uint8(0x28)
	nxBitOr     = uint8(0x29)
	nxLnot      = uint8(0x2A)
	nxBitNot    = uint8(0x2B)
	nxShl       = uint8(0x2C)
	nxShr       = uint8(0x2D)
	nxJump      = uint8(0x30)
	nxJumpTrue  = uint8(0x31)
	nxJumpFalse = uint8(0x32)
	nxCallRT    = uint8(0x38)
	nxCall      = uint8(0x39)
	nxRet       = uint8(0x3A)
	nxPop       = uint8(0xFE)
	nxHalt      = uint8(0xFF)
)

// ErrNaxerUnsupported is returned when the AST contains features the
// NaxerIR emitter cannot handle. The caller should skip JIT compilation
// and let the Go interpreter handle execution.
var ErrNaxerUnsupported = fmt.Errorf("jit/naxer: feature not supported by NaxerIR emitter")

// naxPatch records a jump target placeholder that needs back-patching.
type naxPatch struct{ offset int }

// naxLoopCtx tracks break/continue destinations inside a loop.
type naxLoopCtx struct {
	continueTarget int
	breakSites     []int
}

// naxC is the NaxerIR compiler state for one function.
type naxC struct {
	code      bytes.Buffer
	constants []int64  // int64 and float64 (bit-cast to int64) constant table
	strings   []string // string table for built-in names
	vars      map[string]uint32
	nextSlot  uint32
	loops     []naxLoopCtx
}

// CompileNaxerIR walks a Lunex function body and emits the NXCH wire format.
//
// fnName is used for diagnostics and embedded in the NXCH header.
// body is the function body (ast.Block or ast.Program node).
//
// Returns ErrNaxerUnsupported when any unsupported AST node is encountered.
func CompileNaxerIR(fnName string, body *ast.Node) ([]byte, error) {
	if body == nil {
		return encodeNXCH(fnName, []byte{nxHalt}, nil, nil, 0), nil
	}
	c := &naxC{vars: make(map[string]uint32)}
	if err := c.stmt(body); err != nil {
		return nil, err
	}
	c.emit(nxHalt)
	return encodeNXCH(fnName, c.code.Bytes(), c.constants, c.strings, c.nextSlot), nil
}

// encodeNXCH serialises the NXCH wire format.
func encodeNXCH(name string, code []byte, constants []int64, strings []string, localCount uint32) []byte {
	var buf bytes.Buffer
	buf.WriteString("NXCH")
	writeU32(&buf, uint32(len(code)))
	buf.Write(code)
	writeU32(&buf, uint32(len(constants)))
	for _, v := range constants {
		writeI64(&buf, v)
	}
	writeU32(&buf, uint32(len(strings)))
	for _, s := range strings {
		writeU32(&buf, uint32(len(s)))
		buf.WriteString(s)
	}
	writeU32(&buf, localCount)
	writeU32(&buf, uint32(len(name)))
	buf.WriteString(name)
	return buf.Bytes()
}

// ─── emit helpers ──────────────────────────────────────────────────────────────

func (c *naxC) emit(b ...byte)      { c.code.Write(b) }
func (c *naxC) emitU8(v uint8)     { c.code.WriteByte(v) }
func (c *naxC) pos() int           { return c.code.Len() }

func (c *naxC) emitU32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	c.code.Write(b[:])
}

func (c *naxC) emitJump(op uint8) naxPatch {
	c.emitU8(op)
	pt := naxPatch{offset: c.code.Len()}
	c.emitU32(0) // placeholder
	return pt
}

func (c *naxC) patch(pt naxPatch, target int) {
	binary.LittleEndian.PutUint32(c.code.Bytes()[pt.offset:], uint32(target))
}

func (c *naxC) patchAt(offset, target int) {
	binary.LittleEndian.PutUint32(c.code.Bytes()[offset:], uint32(target))
}

func (c *naxC) addConst(v int64) uint32 {
	idx := uint32(len(c.constants))
	c.constants = append(c.constants, v)
	return idx
}

func (c *naxC) addString(s string) uint32 {
	for i, existing := range c.strings {
		if existing == s {
			return uint32(i)
		}
	}
	idx := uint32(len(c.strings))
	c.strings = append(c.strings, s)
	return idx
}

func (c *naxC) varSlot(name string) uint32 {
	if slot, ok := c.vars[name]; ok {
		return slot
	}
	slot := c.nextSlot
	c.vars[name] = slot
	c.nextSlot++
	return slot
}

// ─── statement compilation ─────────────────────────────────────────────────────

func (c *naxC) stmt(n *ast.Node) error {
	if n == nil {
		return nil
	}
	switch n.Type {
	case ast.Program, ast.Block:
		for _, s := range n.Body_ {
			if err := c.stmt(s); err != nil {
				return err
			}
		}

	case ast.VarDecl, ast.ImmutableDecl, ast.UsingDecl:
		if n.Init != nil {
			if err := c.expr(n.Init); err != nil {
				return err
			}
		} else {
			c.emitU8(nxConstNull)
		}
		slot := c.varSlot(n.Name)
		c.emitU8(nxStore)
		c.emitU32(slot)
		c.emitU8(nxPop)

	case ast.ExprStmt:
		if n.Expr != nil {
			if err := c.expr(n.Expr); err != nil {
				return err
			}
			c.emitU8(nxPop)
		}

	case ast.LogStmt:
		for _, arg := range n.Args {
			if err := c.expr(arg); err != nil {
				return err
			}
		}
		c.emitU8(nxCallRT)
		c.emitU32(c.addString("log"))
		c.emitU8(nxPop)

	case ast.ReturnStmt:
		if n.Expr != nil {
			if err := c.expr(n.Expr); err != nil {
				return err
			}
		} else {
			c.emitU8(nxConstNull)
		}
		c.emitU8(nxRet)

	case ast.IfStmt, ast.UnlessStmt:
		return c.compileIf(n)

	case ast.WhileStmt:
		return c.compileWhile(n)

	case ast.ForStmt:
		return c.compileFor(n)

	case ast.BreakStmt:
		if len(c.loops) == 0 {
			return fmt.Errorf("jit/naxer: break outside loop")
		}
		jmp := c.emitJump(nxJump)
		c.loops[len(c.loops)-1].breakSites = append(c.loops[len(c.loops)-1].breakSites, jmp.offset)

	case ast.ContinueStmt:
		if len(c.loops) == 0 {
			return fmt.Errorf("jit/naxer: continue outside loop")
		}
		target := c.loops[len(c.loops)-1].continueTarget
		c.emitU8(nxJump)
		c.emitU32(uint32(target))

	case ast.AssertStmt:
		if n.Expr != nil {
			if err := c.expr(n.Expr); err != nil {
				return err
			}
		} else {
			c.emitU8(nxConstBool)
			c.emitU8(0x01)
		}
		jmp := c.emitJump(nxJumpTrue)
		c.emitU8(nxConstInt)
		c.emitU32(c.addConst(0)) // "assertion failed" not representable as i64; use 0
		c.emitU8(nxRet)
		c.patch(jmp, c.pos())

	case ast.ThrowStmt, ast.RaiseStmt:
		if n.Expr != nil {
			if err := c.expr(n.Expr); err != nil {
				return err
			}
		} else {
			c.emitU8(nxConstNull)
		}
		c.emitU8(nxRet)

	// Unsupported — fall back to Go interpreter
	case ast.ClassDecl, ast.EnumDecl, ast.NamespaceDecl, ast.ComponentDecl,
		ast.ImportDecl, ast.ExportDecl, ast.LunexRequire, ast.UseStmt,
		ast.TryStmt, ast.SpawnStmt, ast.SelectStmt, ast.WithStmt,
		ast.HaveStmt, ast.IfHaveStmt, ast.IfSetStmt, ast.MatchStmt,
		ast.GuardStmt, ast.DeferStmt, ast.DeleteStmt, ast.RepeatStmt,
		ast.LoopStmt, ast.FnDecl:
		return ErrNaxerUnsupported

	default:
		if err := c.expr(n); err != nil {
			return err
		}
		c.emitU8(nxPop)
	}
	return nil
}

func (c *naxC) compileIf(n *ast.Node) error {
	if n.Test == nil {
		return ErrNaxerUnsupported
	}
	if err := c.expr(n.Test); err != nil {
		return err
	}
	if n.Type == ast.UnlessStmt {
		c.emitU8(nxLnot)
	}
	jmpFalse := c.emitJump(nxJumpFalse)
	if err := c.stmt(n.Consequent); err != nil {
		return err
	}
	if n.Alternate != nil {
		jmpEnd := c.emitJump(nxJump)
		c.patch(jmpFalse, c.pos())
		if err := c.stmt(n.Alternate); err != nil {
			return err
		}
		c.patch(jmpEnd, c.pos())
	} else {
		c.patch(jmpFalse, c.pos())
	}
	return nil
}

func (c *naxC) compileWhile(n *ast.Node) error {
	loopStart := c.pos()
	if err := c.expr(n.Test); err != nil {
		return err
	}
	jmpEnd := c.emitJump(nxJumpFalse)
	c.loops = append(c.loops, naxLoopCtx{continueTarget: loopStart})
	if err := c.stmt(n.Body); err != nil {
		return err
	}
	c.emitU8(nxJump)
	c.emitU32(uint32(loopStart))
	c.patch(jmpEnd, c.pos())
	c.patchBreaks()
	return nil
}

func (c *naxC) compileFor(n *ast.Node) error {
	if n.Init != nil {
		if err := c.stmt(n.Init); err != nil {
			return err
		}
	}
	loopStart := c.pos()
	var jmpEnd naxPatch
	hasTest := n.Test != nil
	if hasTest {
		if err := c.expr(n.Test); err != nil {
			return err
		}
		jmpEnd = c.emitJump(nxJumpFalse)
	}
	c.loops = append(c.loops, naxLoopCtx{continueTarget: loopStart})
	if err := c.stmt(n.Body); err != nil {
		return err
	}
	if n.Right != nil {
		if err := c.expr(n.Right); err != nil {
			return err
		}
		c.emitU8(nxPop)
	}
	c.emitU8(nxJump)
	c.emitU32(uint32(loopStart))
	if hasTest {
		c.patch(jmpEnd, c.pos())
	}
	c.patchBreaks()
	return nil
}

func (c *naxC) patchBreaks() {
	if len(c.loops) == 0 {
		return
	}
	end := c.pos()
	for _, site := range c.loops[len(c.loops)-1].breakSites {
		c.patchAt(site, end)
	}
	c.loops = c.loops[:len(c.loops)-1]
}

// ─── expression compilation ────────────────────────────────────────────────────

func (c *naxC) expr(n *ast.Node) error {
	if n == nil {
		c.emitU8(nxConstNull)
		return nil
	}
	switch n.Type {
	case ast.NumberLit:
		return c.emitNumber(n.Value)

	case ast.StringLit:
		// Strings are not representable in Naxer i64 IR — unsupported.
		return ErrNaxerUnsupported

	case ast.BoolLit:
		c.emitU8(nxConstBool)
		if n.Value == true {
			c.emitU8(0x01)
		} else {
			c.emitU8(0x00)
		}

	case ast.NullLit, ast.UndefinedLit:
		c.emitU8(nxConstNull)

	case ast.Identifier:
		slot := c.varSlot(n.Name)
		c.emitU8(nxLoad)
		c.emitU32(slot)

	case ast.BinaryExpr:
		return c.compileBinary(n)

	case ast.UnaryExpr, ast.NotExpr:
		return c.compileUnary(n)

	case ast.LogicalExpr:
		return c.compileLogical(n)

	case ast.AssignExpr:
		return c.compileAssign(n)

	case ast.TernaryExpr:
		return c.compileTernary(n)

	case ast.VoidExpr:
		if n.Expr != nil {
			if err := c.expr(n.Expr); err != nil {
				return err
			}
			c.emitU8(nxPop)
		}
		c.emitU8(nxConstNull)

	// Unsupported expressions — Go interpreter handles these
	case ast.CallExpr, ast.MemberExpr, ast.ArrayLit, ast.ObjectLit,
		ast.SpreadExpr, ast.TypeofExpr, ast.FnExpr, ast.ArrowFn,
		ast.AtImportExpr, ast.NewExpr, ast.TemplateLit, ast.PipelineExpr,
		ast.SequenceExpr, ast.HaveExpr, ast.TrySafeExpr, ast.RangeExpr,
		ast.SleepExpr, ast.ChannelExpr, ast.NaxImportExpr, ast.ThisExpr,
		ast.SuperExpr, ast.SatisfiesExpr, ast.DecoratedExpr,
		ast.RegexLit, ast.DeleteExpr:
		return ErrNaxerUnsupported

	default:
		return ErrNaxerUnsupported
	}
	return nil
}

func (c *naxC) emitNumber(v interface{}) error {
	switch val := v.(type) {
	case int64:
		c.emitU8(nxConstInt)
		c.emitU32(c.addConst(val))
	case int:
		c.emitU8(nxConstInt)
		c.emitU32(c.addConst(int64(val)))
	case float64:
		if val == math.Trunc(val) && val >= -9e15 && val <= 9e15 {
			c.emitU8(nxConstInt)
			c.emitU32(c.addConst(int64(val)))
		} else {
			// Store float64 as bit-cast int64 in constant table.
			c.emitU8(nxConstF64)
			c.emitU32(c.addConst(int64(math.Float64bits(val))))
		}
	default:
		c.emitU8(nxConstInt)
		c.emitU32(c.addConst(0))
	}
	return nil
}

func (c *naxC) compileBinary(n *ast.Node) error {
	if err := c.expr(n.Left); err != nil {
		return err
	}
	if err := c.expr(n.Right); err != nil {
		return err
	}
	switch n.Op {
	case "+":
		c.emitU8(nxAdd)
	case "-":
		c.emitU8(nxSub)
	case "*":
		c.emitU8(nxMul)
	case "/":
		c.emitU8(nxDiv)
	case "%":
		c.emitU8(nxMod)
	case "==", "===":
		c.emitU8(nxEq)
	case "!=", "!==":
		c.emitU8(nxNeq)
	case "<":
		c.emitU8(nxLt)
	case "<=":
		c.emitU8(nxLte)
	case ">":
		c.emitU8(nxGt)
	case ">=":
		c.emitU8(nxGte)
	case "&":
		c.emitU8(nxBitAnd)
	case "|":
		c.emitU8(nxBitOr)
	case "<<":
		c.emitU8(nxShl)
	case ">>":
		c.emitU8(nxShr)
	default:
		return fmt.Errorf("jit/naxer: unsupported binary op %q", n.Op)
	}
	return nil
}

func (c *naxC) compileUnary(n *ast.Node) error {
	arg := n.Arg
	if arg == nil {
		arg = n.Expr
	}
	if err := c.expr(arg); err != nil {
		return err
	}
	switch n.Op {
	case "-":
		c.emitU8(nxNeg)
	case "!", "not":
		c.emitU8(nxLnot)
	case "~":
		c.emitU8(nxBitNot)
	case "+":
		// unary plus is a no-op for numbers
	default:
		return fmt.Errorf("jit/naxer: unsupported unary op %q", n.Op)
	}
	return nil
}

func (c *naxC) compileLogical(n *ast.Node) error {
	switch n.Op {
	case "&&", "and":
		if err := c.expr(n.Left); err != nil {
			return err
		}
		skipRight := c.emitJump(nxJumpFalse)
		c.emitU8(nxPop)
		if err := c.expr(n.Right); err != nil {
			return err
		}
		c.patch(skipRight, c.pos())
	case "||", "or":
		if err := c.expr(n.Left); err != nil {
			return err
		}
		skipRight := c.emitJump(nxJumpTrue)
		c.emitU8(nxPop)
		if err := c.expr(n.Right); err != nil {
			return err
		}
		c.patch(skipRight, c.pos())
	default:
		return fmt.Errorf("jit/naxer: unsupported logical op %q", n.Op)
	}
	return nil
}

func (c *naxC) compileAssign(n *ast.Node) error {
	if n.Left == nil || n.Left.Type != ast.Identifier {
		return ErrNaxerUnsupported
	}
	rhs := n.Right
	if rhs == nil {
		rhs = n.Value.(*ast.Node)
	}
	if n.Op != "" && n.Op != "=" {
		// Compound assignment (+=, -=, …) — load LHS first.
		slot := c.varSlot(n.Left.Name)
		c.emitU8(nxLoad)
		c.emitU32(slot)
		if err := c.expr(rhs); err != nil {
			return err
		}
		var op uint8
		switch n.Op {
		case "+=":
			op = nxAdd
		case "-=":
			op = nxSub
		case "*=":
			op = nxMul
		case "/=":
			op = nxDiv
		case "%=":
			op = nxMod
		default:
			return fmt.Errorf("jit/naxer: unsupported compound assign %q", n.Op)
		}
		c.emitU8(op)
	} else {
		if err := c.expr(rhs); err != nil {
			return err
		}
	}
	slot := c.varSlot(n.Left.Name)
	c.emitU8(nxStore)
	c.emitU32(slot)
	return nil
}

func (c *naxC) compileTernary(n *ast.Node) error {
	if err := c.expr(n.Test); err != nil {
		return err
	}
	jmpFalse := c.emitJump(nxJumpFalse)
	if err := c.expr(n.Consequent); err != nil {
		return err
	}
	jmpEnd := c.emitJump(nxJump)
	c.patch(jmpFalse, c.pos())
	if err := c.expr(n.Alternate); err != nil {
		return err
	}
	c.patch(jmpEnd, c.pos())
	return nil
}

// ─── Wire format helpers ───────────────────────────────────────────────────────

func writeU32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

func writeI64(buf *bytes.Buffer, v int64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	buf.Write(b[:])
}
