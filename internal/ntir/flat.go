// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ntir

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type RegID = uint16

const MaxRegs = 256

type FlatOpcode uint8

const (
	FOP_NOP     FlatOpcode = iota
	FOP_CONST             // dst = kst[kst_idx]
	FOP_MOVE              // dst = src_a
	FOP_LOAD_G            // dst = globals[kst[kst_idx]]
	FOP_STORE_G           // globals[kst[kst_idx]] = src_a
	FOP_ADD               // dst = a + b
	FOP_SUB               // dst = a - b
	FOP_MUL               // dst = a * b
	FOP_DIV               // dst = a / b
	FOP_MOD               // dst = a % b
	FOP_NEG               // dst = -a
	FOP_NOT               // dst = !a
	FOP_BAND              // dst = a & b
	FOP_BOR               // dst = a | b
	FOP_BXOR              // dst = a ^ b
	FOP_BNOT              // dst = ~a
	FOP_SHL               // dst = a << b
	FOP_SHR               // dst = a >> b
	FOP_EQ                // dst = a == b
	FOP_NEQ               // dst = a != b
	FOP_LT                // dst = a < b
	FOP_LTE               // dst = a <= b
	FOP_GT                // dst = a > b
	FOP_GTE               // dst = a >= b
	FOP_AND               // dst = a && b
	FOP_OR                // dst = a || b
	FOP_CONCAT            // dst = a .. b (string concat)
	FOP_LEN               // dst = len(a)
	FOP_TO_STR            // dst = toString(a)
	FOP_TO_NUM            // dst = toNumber(a)
	FOP_TO_BOOL           // dst = toBool(a)
	FOP_GET_FIELD         // dst = obj[key_kst]
	FOP_SET_FIELD         // obj[key_kst] = src
	FOP_GET_INDEX         // dst = obj[idx]
	FOP_SET_INDEX         // obj[idx] = val
	FOP_MAKE_ARRAY        // dst = array(base_r, count)
	FOP_MAKE_OBJ          // dst = {}
	FOP_CALL              // dst = callee(args...)
	FOP_CALL_RT           // runtime call
	FOP_RETURN            // return src_a
	FOP_JMP               // ip = target
	FOP_JMP_TRUE          // if a: ip = target
	FOP_JMP_FALSE         // if !a: ip = target
	FOP_THROW             // throw a
	FOP_MAKE_CLOSURE      // dst = closure(proto_idx, upval_count)
	FOP_GET_UPVAL         // dst = upvals[idx]
	FOP_SET_UPVAL         // upvals[idx] = src
	FOP_DUP               // dst = a (alias for move)
	FOP_POP               // no-op for stack compat
)

var flatOpcodeNames = [...]string{
	"nop", "const", "move", "load_g", "store_g",
	"add", "sub", "mul", "div", "mod", "neg", "not",
	"band", "bor", "bxor", "bnot", "shl", "shr",
	"eq", "neq", "lt", "lte", "gt", "gte", "and", "or",
	"concat", "len", "to_str", "to_num", "to_bool",
	"get_field", "set_field", "get_idx", "set_idx",
	"mkarr", "mkobj", "call", "call_rt",
	"ret", "jmp", "jmp_true", "jmp_false",
	"throw", "closure", "get_upv", "set_upv",
	"dup", "pop",
}

func (op FlatOpcode) String() string {
	if int(op) < len(flatOpcodeNames) {
		return flatOpcodeNames[op]
	}
	return fmt.Sprintf("op%d", op)
}

type FlatInstr struct {
	Op  FlatOpcode
	Dst uint8
	A   uint8
	B   uint8
	Kst uint32
}

func (i FlatInstr) String() string {
	switch i.Op {
	case FOP_CONST:
		return fmt.Sprintf("%-12s r%d, kst[%d]", i.Op, i.Dst, i.Kst)
	case FOP_LOAD_G, FOP_STORE_G:
		return fmt.Sprintf("%-12s r%d, g[%d]", i.Op, i.Dst, i.Kst)
	case FOP_JMP:
		return fmt.Sprintf("%-12s .L%d", i.Op, i.Kst)
	case FOP_JMP_TRUE, FOP_JMP_FALSE:
		return fmt.Sprintf("%-12s r%d, .L%d", i.Op, i.A, i.Kst)
	case FOP_GET_FIELD, FOP_SET_FIELD:
		return fmt.Sprintf("%-12s r%d, r%d, k[%d]", i.Op, i.Dst, i.A, i.Kst)
	case FOP_ADD, FOP_SUB, FOP_MUL, FOP_DIV, FOP_MOD,
		FOP_BAND, FOP_BOR, FOP_BXOR, FOP_SHL, FOP_SHR,
		FOP_EQ, FOP_NEQ, FOP_LT, FOP_LTE, FOP_GT, FOP_GTE,
		FOP_AND, FOP_OR, FOP_CONCAT:
		return fmt.Sprintf("%-12s r%d, r%d, r%d", i.Op, i.Dst, i.A, i.B)
	case FOP_NEG, FOP_NOT, FOP_BNOT, FOP_LEN, FOP_TO_STR, FOP_TO_NUM, FOP_TO_BOOL:
		return fmt.Sprintf("%-12s r%d, r%d", i.Op, i.Dst, i.A)
	case FOP_MOVE, FOP_DUP:
		return fmt.Sprintf("%-12s r%d, r%d", i.Op, i.Dst, i.A)
	case FOP_RETURN:
		return fmt.Sprintf("%-12s r%d", i.Op, i.A)
	case FOP_THROW:
		return fmt.Sprintf("%-12s r%d", i.Op, i.A)
	case FOP_CALL:
		return fmt.Sprintf("%-12s r%d, r%d, argc=%d", i.Op, i.Dst, i.A, i.B)
	case FOP_CALL_RT:
		return fmt.Sprintf("%-12s r%d, sym[%d], argc=%d", i.Op, i.Dst, i.Kst, i.A)
	case FOP_MAKE_ARRAY:
		return fmt.Sprintf("%-12s r%d, base=r%d, n=%d", i.Op, i.Dst, i.A, i.Kst)
	case FOP_MAKE_OBJ:
		return fmt.Sprintf("%-12s r%d", i.Op, i.Dst)
	}
	return fmt.Sprintf("%-12s r%d r%d r%d kst=%d", i.Op, i.Dst, i.A, i.B, i.Kst)
}

type KstKind uint8

const (
	KNull    KstKind = 0
	KBool    KstKind = 1
	KNumber  KstKind = 2
	KString  KstKind = 3
	KFunc    KstKind = 4
)

type KstEntry struct {
	Kind KstKind
	IVal int64
	FVal float64
	SVal string
	BVal bool
}

func (k KstEntry) String() string {
	switch k.Kind {
	case KNull:
		return "null"
	case KBool:
		if k.BVal {
			return "true"
		}
		return "false"
	case KNumber:
		if k.FVal == float64(int64(k.FVal)) {
			return fmt.Sprintf("%d", int64(k.FVal))
		}
		return fmt.Sprintf("%g", k.FVal)
	case KString:
		return fmt.Sprintf("%q", k.SVal)
	case KFunc:
		return fmt.Sprintf("fn<%s>", k.SVal)
	}
	return "?"
}

type FlatFunc struct {
	Name      string
	Instrs    []FlatInstr
	Constants []KstEntry
	RegCount  uint8
	ParamCount uint8
	IsExport  bool
	Children  []*FlatFunc
}

func NewFlatFunc(name string) *FlatFunc {
	return &FlatFunc{Name: name}
}

func (f *FlatFunc) Emit(instr FlatInstr) int {
	idx := len(f.Instrs)
	f.Instrs = append(f.Instrs, instr)
	return idx
}

func (f *FlatFunc) EmitNop() int {
	return f.Emit(FlatInstr{Op: FOP_NOP})
}

func (f *FlatFunc) PatchJump(idx int, target int) {
	if idx >= 0 && idx < len(f.Instrs) {
		f.Instrs[idx].Kst = uint32(target)
	}
}

func (f *FlatFunc) AddConst(k KstEntry) uint32 {
	for i, existing := range f.Constants {
		if existing.Kind == k.Kind && constEq(existing, k) {
			return uint32(i)
		}
	}
	idx := uint32(len(f.Constants))
	f.Constants = append(f.Constants, k)
	return idx
}

func constEq(a, b KstEntry) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case KNull:
		return true
	case KBool:
		return a.BVal == b.BVal
	case KNumber:
		return a.FVal == b.FVal || (math.IsNaN(a.FVal) && math.IsNaN(b.FVal))
	case KString, KFunc:
		return a.SVal == b.SVal
	}
	return false
}

func (f *FlatFunc) AddString(s string) uint32 {
	return f.AddConst(KstEntry{Kind: KString, SVal: s})
}

func (f *FlatFunc) AddNumber(n float64) uint32 {
	return f.AddConst(KstEntry{Kind: KNumber, FVal: n})
}

func (f *FlatFunc) AddNull() uint32 {
	return f.AddConst(KstEntry{Kind: KNull})
}

func (f *FlatFunc) AddBool(b bool) uint32 {
	return f.AddConst(KstEntry{Kind: KBool, BVal: b})
}

func (f *FlatFunc) IP() int {
	return len(f.Instrs)
}

func (f *FlatFunc) AllocReg() uint8 {
	r := f.RegCount
	f.RegCount++
	return r
}

func (f *FlatFunc) Disassemble() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("fn %s [regs=%d, params=%d]:\n", f.Name, f.RegCount, f.ParamCount))
	sb.WriteString("  constants:\n")
	for i, k := range f.Constants {
		sb.WriteString(fmt.Sprintf("    [%d] %s\n", i, k))
	}
	sb.WriteString("  code:\n")
	for i, instr := range f.Instrs {
		sb.WriteString(fmt.Sprintf("  %04d  %s\n", i, instr))
	}
	for _, child := range f.Children {
		sb.WriteString(child.Disassemble())
	}
	return sb.String()
}

type FlatModule struct {
	Name    string
	Funcs   []*FlatFunc
	Imports []string
}

func NewFlatModule(name string) *FlatModule {
	return &FlatModule{Name: name}
}

func (m *FlatModule) AddFunc(f *FlatFunc) {
	m.Funcs = append(m.Funcs, f)
}

func (m *FlatModule) Main() *FlatFunc {
	for _, f := range m.Funcs {
		if f.Name == "__lunex_main__" || f.Name == "main" {
			return f
		}
	}
	if len(m.Funcs) > 0 {
		return m.Funcs[0]
	}
	return nil
}

func (m *FlatModule) Disassemble() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("; flat module: %s\n\n", m.Name))
	for _, f := range m.Funcs {
		sb.WriteString(f.Disassemble())
		sb.WriteRune('\n')
	}
	return sb.String()
}

func (m *FlatModule) Encode() []byte {
	var buf []byte
	buf = binary.LittleEndian.AppendUint32(buf, 0x4C544E03)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(m.Funcs)))
	for _, f := range m.Funcs {
		buf = encodeFunc(buf, f)
	}
	return buf
}

func encodeFunc(buf []byte, f *FlatFunc) []byte {
	nameBytes := []byte(f.Name)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(nameBytes)))
	buf = append(buf, nameBytes...)
	buf = append(buf, f.RegCount, f.ParamCount)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(f.Instrs)))
	for _, instr := range f.Instrs {
		raw := uint32(instr.Op) |
			(uint32(instr.Dst) << 8) |
			(uint32(instr.A) << 16) |
			(uint32(instr.B) << 24)
		buf = binary.LittleEndian.AppendUint32(buf, raw)
		buf = binary.LittleEndian.AppendUint32(buf, instr.Kst)
	}
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(f.Constants)))
	for _, k := range f.Constants {
		buf = append(buf, byte(k.Kind))
		switch k.Kind {
		case KNull:
		case KBool:
			if k.BVal {
				buf = append(buf, 1)
			} else {
				buf = append(buf, 0)
			}
		case KNumber:
			bits := math.Float64bits(k.FVal)
			buf = binary.LittleEndian.AppendUint64(buf, bits)
		case KString, KFunc:
			sb := []byte(k.SVal)
			buf = binary.LittleEndian.AppendUint32(buf, uint32(len(sb)))
			buf = append(buf, sb...)
		}
	}
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(f.Children)))
	for _, child := range f.Children {
		buf = encodeFunc(buf, child)
	}
	return buf
}
