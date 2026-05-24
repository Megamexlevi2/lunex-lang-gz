// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ntir

import (
	"fmt"
	"strings"
)

type TypeKind uint8

const (
	TyAny     TypeKind = iota
	TyInt64
	TyFloat64
	TyBool
	TyString
	TyNull
	TyVoid
	TyPtr
	TyFunc
	TyObject
	TyArray
)

type Type struct {
	Kind    TypeKind
	ElemTy  *Type
	Params  []*Type
	RetTy   *Type
}

var (
	TyAnyT     = &Type{Kind: TyAny}
	TyInt64T   = &Type{Kind: TyInt64}
	TyFloat64T = &Type{Kind: TyFloat64}
	TyBoolT    = &Type{Kind: TyBool}
	TyStringT  = &Type{Kind: TyString}
	TyNullT    = &Type{Kind: TyNull}
	TyVoidT    = &Type{Kind: TyVoid}
	TyPtrT     = &Type{Kind: TyPtr}
	TyFuncT    = &Type{Kind: TyFunc}
	TyObjectT  = &Type{Kind: TyObject}
	TyArrayT   = &Type{Kind: TyArray}
)

func (t *Type) String() string {
	if t == nil {
		return "any"
	}
	switch t.Kind {
	case TyAny:
		return "any"
	case TyInt64:
		return "i64"
	case TyFloat64:
		return "f64"
	case TyBool:
		return "bool"
	case TyString:
		return "str"
	case TyNull:
		return "null"
	case TyVoid:
		return "void"
	case TyPtr:
		return "ptr"
	case TyFunc:
		return "fn"
	case TyObject:
		return "obj"
	case TyArray:
		return "arr"
	}
	return "unknown"
}

type Opcode uint8

const (
	OpConst     Opcode = iota
	OpLoad
	OpStore
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpNeg
	OpNot
	OpBitAnd
	OpBitOr
	OpBitXor
	OpBitNot
	OpShl
	OpShr
	OpEq
	OpNeq
	OpLt
	OpLte
	OpGt
	OpGte
	OpAnd
	OpOr
	OpCall
	OpCallRuntime
	OpReturn
	OpJump
	OpJumpIf
	OpJumpIfFalse
	OpPhi
	OpAlloc
	OpGetField
	OpSetField
	OpGetIndex
	OpSetIndex
	OpMakeArray
	OpMakeObject
	OpMakeClosure
	OpGetUpval
	OpSetUpval
	OpTypecheck
	OpBox
	OpUnbox
	OpConcat
	OpLen
	OpToString
	OpToNumber
	OpToBool
	OpThrow
	OpLandingPad
	OpNop
)

var opcodeNames = map[Opcode]string{
	OpConst:       "const",
	OpLoad:        "load",
	OpStore:       "store",
	OpAdd:         "add",
	OpSub:         "sub",
	OpMul:         "mul",
	OpDiv:         "div",
	OpMod:         "mod",
	OpNeg:         "neg",
	OpNot:         "not",
	OpBitAnd:      "band",
	OpBitOr:       "bor",
	OpBitXor:      "bxor",
	OpBitNot:      "bnot",
	OpShl:         "shl",
	OpShr:         "shr",
	OpEq:          "eq",
	OpNeq:         "neq",
	OpLt:          "lt",
	OpLte:         "lte",
	OpGt:          "gt",
	OpGte:         "gte",
	OpAnd:         "and",
	OpOr:          "or",
	OpCall:        "call",
	OpCallRuntime: "callrt",
	OpReturn:      "ret",
	OpJump:        "jmp",
	OpJumpIf:      "jif",
	OpJumpIfFalse: "jiff",
	OpPhi:         "phi",
	OpAlloc:       "alloc",
	OpGetField:    "getf",
	OpSetField:    "setf",
	OpGetIndex:    "geti",
	OpSetIndex:    "seti",
	OpMakeArray:   "mkarr",
	OpMakeObject:  "mkobj",
	OpMakeClosure: "mkcls",
	OpGetUpval:    "getupv",
	OpSetUpval:    "setupv",
	OpTypecheck:   "tyck",
	OpBox:         "box",
	OpUnbox:       "unbox",
	OpConcat:      "concat",
	OpLen:         "len",
	OpToString:    "tostr",
	OpToNumber:    "tonum",
	OpToBool:      "tobool",
	OpThrow:       "throw",
	OpLandingPad:  "landpad",
	OpNop:         "nop",
}

func (o Opcode) String() string {
	if s, ok := opcodeNames[o]; ok {
		return s
	}
	return fmt.Sprintf("op(%d)", o)
}

type Value struct {
	ID   int
	Type *Type
	Name string
}

func (v *Value) String() string {
	if v == nil {
		return "<nil>"
	}
	if v.Name != "" {
		return fmt.Sprintf("%%%s", v.Name)
	}
	return fmt.Sprintf("%%v%d", v.ID)
}

type ConstVal struct {
	Kind    TypeKind
	IntVal  int64
	FltVal  float64
	StrVal  string
	BoolVal bool
}

func (c *ConstVal) String() string {
	switch c.Kind {
	case TyInt64:
		return fmt.Sprintf("%d", c.IntVal)
	case TyFloat64:
		return fmt.Sprintf("%g", c.FltVal)
	case TyBool:
		if c.BoolVal {
			return "true"
		}
		return "false"
	case TyString:
		return fmt.Sprintf("%q", c.StrVal)
	case TyNull:
		return "null"
	}
	return "?"
}

type Instr struct {
	Op      Opcode
	Dst     *Value
	Src     []*Value
	Const   *ConstVal
	Symbol  string
	BlockID int
	Type    *Type
	Extra   interface{}
}

func (i *Instr) String() string {
	var sb strings.Builder
	if i.Dst != nil {
		sb.WriteString(i.Dst.String())
		sb.WriteString(" = ")
	}
	sb.WriteString(i.Op.String())
	if i.Const != nil {
		sb.WriteString(" ")
		sb.WriteString(i.Const.String())
	}
	if i.Symbol != "" {
		sb.WriteString(" @")
		sb.WriteString(i.Symbol)
	}
	for _, s := range i.Src {
		sb.WriteString(" ")
		sb.WriteString(s.String())
	}
	if i.BlockID != 0 {
		sb.WriteString(fmt.Sprintf(" .L%d", i.BlockID))
	}
	return sb.String()
}

type BasicBlock struct {
	ID     int
	Label  string
	Instrs []*Instr
	Preds  []*BasicBlock
	Succs  []*BasicBlock
}

func NewBasicBlock(id int, label string) *BasicBlock {
	return &BasicBlock{ID: id, Label: label}
}

func (b *BasicBlock) Add(instr *Instr) {
	b.Instrs = append(b.Instrs, instr)
}

func (b *BasicBlock) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(".L%d:", b.ID))
	if b.Label != "" {
		sb.WriteString(" ; ")
		sb.WriteString(b.Label)
	}
	sb.WriteString("\n")
	for _, instr := range b.Instrs {
		sb.WriteString("  ")
		sb.WriteString(instr.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

type UpvalDesc struct {
	Name  string
	Index int
	FromParent bool
}

type FuncIR struct {
	Name      string
	Params    []*Value
	RetType   *Type
	Blocks    []*BasicBlock
	Entry     *BasicBlock
	Upvals    []UpvalDesc
	IsExport  bool
	IsInline  bool
	SourcePos int
	valueID   int
}

func NewFuncIR(name string) *FuncIR {
	f := &FuncIR{Name: name}
	entry := NewBasicBlock(0, "entry")
	f.Blocks = append(f.Blocks, entry)
	f.Entry = entry
	return f
}

func (f *FuncIR) NewValue(ty *Type) *Value {
	f.valueID++
	return &Value{ID: f.valueID, Type: ty}
}

func (f *FuncIR) NewNamedValue(name string, ty *Type) *Value {
	f.valueID++
	return &Value{ID: f.valueID, Name: name, Type: ty}
}

func (f *FuncIR) NewBlock(label string) *BasicBlock {
	id := len(f.Blocks)
	b := NewBasicBlock(id, label)
	f.Blocks = append(f.Blocks, b)
	return b
}

func (f *FuncIR) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("fn %s(", f.Name))
	for i, p := range f.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.String())
		if p.Type != nil {
			sb.WriteString(": ")
			sb.WriteString(p.Type.String())
		}
	}
	sb.WriteString(")")
	if f.RetType != nil {
		sb.WriteString(" -> ")
		sb.WriteString(f.RetType.String())
	}
	sb.WriteString(" {\n")
	for _, b := range f.Blocks {
		sb.WriteString(b.String())
	}
	sb.WriteString("}\n")
	return sb.String()
}

type Module struct {
	Name     string
	Funcs    []*FuncIR
	Globals  map[string]*ConstVal
	Strings  []string
	Imports  []string
	Exports  []string
	SourceFile string
}

func NewModule(name string) *Module {
	return &Module{
		Name:    name,
		Globals: make(map[string]*ConstVal),
	}
}

func (m *Module) AddFunc(f *FuncIR) {
	m.Funcs = append(m.Funcs, f)
}

func (m *Module) InternString(s string) int {
	for i, existing := range m.Strings {
		if existing == s {
			return i
		}
	}
	idx := len(m.Strings)
	m.Strings = append(m.Strings, s)
	return idx
}

func (m *Module) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("; module: %s\n; source: %s\n\n", m.Name, m.SourceFile))
	for _, f := range m.Funcs {
		sb.WriteString(f.String())
		sb.WriteString("\n")
	}
	return sb.String()
}
