package asm

import (
	"fmt"
	"lunex/internal/ntir"
	"strings"
)

type Generator struct {
	sb      *strings.Builder
	regMap  map[int]string
	regCnt  int
	labelFn func(int) string
}

func NewGenerator() *Generator {
	return &Generator{
		sb:     &strings.Builder{},
		regMap: make(map[int]string),
	}
}

func reg(v *ntir.Value) string {
	if v == nil {
		return "r0"
	}
	if v.Name != "" {
		return "r" + sanitize(v.Name)
	}
	return fmt.Sprintf("r%d", v.ID)
}

func sanitize(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func label(id int) string {
	return fmt.Sprintf("L%d", id)
}

func constStr(c *ntir.ConstVal) string {
	if c == nil {
		return "0"
	}
	return c.String()
}

func EmitModule(module *ntir.Module) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("; NTASM module: %s\n\n", module.Name))
	for _, fn := range module.Funcs {
		sb.WriteString(EmitFunc(fn))
		sb.WriteString("\n")
	}
	return sb.String()
}

func EmitFunc(fn *ntir.FuncIR) string {
	var sb strings.Builder

	params := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		params[i] = reg(p)
	}
	retStr := "void"
	if fn.RetType != nil {
		retStr = fn.RetType.String()
	}
	sb.WriteString(fmt.Sprintf("fn %s(%s) -> %s {\n", fn.Name, strings.Join(params, ", "), retStr))

	for _, blk := range fn.Blocks {
		sb.WriteString(fmt.Sprintf("%s:\n", label(blk.ID)))
		for _, instr := range blk.Instrs {
			line := emitInstr(instr)
			if line != "" {
				sb.WriteString("  " + line + "\n")
			}
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

func emitInstr(i *ntir.Instr) string {
	dst := ""
	if i.Dst != nil {
		dst = reg(i.Dst) + " = "
	}

	switch i.Op {
	case ntir.OpConst:
		return fmt.Sprintf("%sMOV %s", dst, constStr(i.Const))

	case ntir.OpLoad:
		if i.Symbol != "" {
			return fmt.Sprintf("%sLOAD %s", dst, i.Symbol)
		}
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sLOAD [%s]", dst, reg(i.Src[0]))
		}
		return fmt.Sprintf("%sLOAD ?", dst)

	case ntir.OpStore:
		if i.Symbol != "" && len(i.Src) > 0 {
			return fmt.Sprintf("STORE %s, %s", i.Symbol, reg(i.Src[0]))
		}
		if len(i.Src) >= 2 {
			return fmt.Sprintf("STORE [%s], %s", reg(i.Src[0]), reg(i.Src[1]))
		}
		return "STORE ?"

	case ntir.OpAdd:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sADD %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpSub:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sSUB %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpMul:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sMUL %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpDiv:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sDIV %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpMod:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sMOD %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpNeg:
		if len(i.Src) >= 1 {
			return fmt.Sprintf("%sNEG %s", dst, reg(i.Src[0]))
		}

	case ntir.OpNot:
		if len(i.Src) >= 1 {
			return fmt.Sprintf("%sNOT %s", dst, reg(i.Src[0]))
		}

	case ntir.OpBitAnd:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sAND %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpBitOr:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sOR %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpBitXor:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sXOR %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpBitNot:
		if len(i.Src) >= 1 {
			return fmt.Sprintf("%sBNOT %s", dst, reg(i.Src[0]))
		}

	case ntir.OpShl:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sSHL %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpShr:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sSHR %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpEq:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCMP_EQ %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpNeq:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCMP_NEQ %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpLt:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCMP_LT %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpLte:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCMP_LTE %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpGt:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCMP_GT %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpGte:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCMP_GTE %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpAnd:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sLAND %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpOr:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sLOR %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpCall:
		args := make([]string, len(i.Src))
		for k, s := range i.Src {
			args[k] = reg(s)
		}
		sym := i.Symbol
		if sym == "" {
			sym = "?"
		}
		return fmt.Sprintf("%sCALL %s(%s)", dst, sym, strings.Join(args, ", "))

	case ntir.OpCallRuntime:
		args := make([]string, len(i.Src))
		for k, s := range i.Src {
			args[k] = reg(s)
		}
		return fmt.Sprintf("%sRTCALL %s(%s)", dst, i.Symbol, strings.Join(args, ", "))

	case ntir.OpReturn:
		if len(i.Src) > 0 {
			return fmt.Sprintf("RET %s", reg(i.Src[0]))
		}
		return "RET"

	case ntir.OpJump:
		return fmt.Sprintf("JMP %s", label(i.BlockID))

	case ntir.OpJumpIf:
		if len(i.Src) > 0 {
			return fmt.Sprintf("JNZ %s, %s", reg(i.Src[0]), label(i.BlockID))
		}

	case ntir.OpJumpIfFalse:
		if len(i.Src) > 0 {
			return fmt.Sprintf("JZ %s, %s", reg(i.Src[0]), label(i.BlockID))
		}

	case ntir.OpGetField:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sGETF %s.%s", dst, reg(i.Src[0]), i.Symbol)
		}

	case ntir.OpSetField:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("SETF %s.%s, %s", reg(i.Src[0]), i.Symbol, reg(i.Src[1]))
		}

	case ntir.OpGetIndex:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sGETI %s[%s]", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpSetIndex:
		if len(i.Src) >= 3 {
			return fmt.Sprintf("SETI %s[%s], %s", reg(i.Src[0]), reg(i.Src[1]), reg(i.Src[2]))
		}

	case ntir.OpMakeArray:
		args := make([]string, len(i.Src))
		for k, s := range i.Src {
			args[k] = reg(s)
		}
		return fmt.Sprintf("%sMKARR [%s]", dst, strings.Join(args, ", "))

	case ntir.OpMakeObject:
		return fmt.Sprintf("%sMKOBJ", dst)

	case ntir.OpMakeClosure:
		return fmt.Sprintf("%sMKCLS %s", dst, i.Symbol)

	case ntir.OpAlloc:
		return fmt.Sprintf("%sALLOC", dst)

	case ntir.OpLen:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sLEN %s", dst, reg(i.Src[0]))
		}

	case ntir.OpConcat:
		if len(i.Src) >= 2 {
			return fmt.Sprintf("%sCONCAT %s, %s", dst, reg(i.Src[0]), reg(i.Src[1]))
		}

	case ntir.OpToString:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sTOSTR %s", dst, reg(i.Src[0]))
		}

	case ntir.OpToNumber:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sTONUM %s", dst, reg(i.Src[0]))
		}

	case ntir.OpToBool:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sTOBOOL %s", dst, reg(i.Src[0]))
		}

	case ntir.OpBox:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sBOX %s", dst, reg(i.Src[0]))
		}

	case ntir.OpUnbox:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sUNBOX %s", dst, reg(i.Src[0]))
		}

	case ntir.OpThrow:
		if len(i.Src) > 0 {
			return fmt.Sprintf("THROW %s", reg(i.Src[0]))
		}
		return "THROW"

	case ntir.OpLandingPad:
		return fmt.Sprintf("%sLAND", dst)

	case ntir.OpNop:
		return "NOP"

	case ntir.OpPhi:
		args := make([]string, len(i.Src))
		for k, s := range i.Src {
			args[k] = reg(s)
		}
		return fmt.Sprintf("%sPHI [%s]", dst, strings.Join(args, ", "))

	case ntir.OpGetUpval:
		return fmt.Sprintf("%sGETUPV %s", dst, i.Symbol)

	case ntir.OpSetUpval:
		if len(i.Src) > 0 {
			return fmt.Sprintf("SETUPV %s, %s", i.Symbol, reg(i.Src[0]))
		}

	case ntir.OpTypecheck:
		if len(i.Src) > 0 {
			return fmt.Sprintf("%sTYCK %s", dst, reg(i.Src[0]))
		}
	}

	return fmt.Sprintf("; unhandled op: %s", i.Op.String())
}

func LoopExample() string {
	return `; Lunex loop:
;   while i <= 1000000 {
;     sum = sum + i
;     i = i + 1
;   }

fn count_sum(limit) -> i64 {
L0:
  rsum = MOV 0
  ri = MOV 1
L1:
  CMP ri, rlimit
  JG L2
  ADD rsum, rsum, ri
  ADD ri, ri, 1
  JMP L1
L2:
  RET rsum
}
`
}
