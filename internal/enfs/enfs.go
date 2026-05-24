// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package enfs

import (
	"fmt"
	"math"
	"lunex/internal/ntir"
)

type OptLevel int

const (
	OptNone       OptLevel = 0
	OptBasic      OptLevel = 1
	OptAggressive OptLevel = 2
	OptExtreme    OptLevel = 3
)

type Optimizer struct {
	level OptLevel
	stats Stats
}

type Stats struct {
	DeadInstrsRemoved  int
	ConstsFolded       int
	FunctionsInlined   int
	BlocksMerged       int
	TailCallsConverted int
	PhisEliminated     int
}

func New(level OptLevel) *Optimizer {
	return &Optimizer{level: level}
}

func Extreme() *Optimizer {
	return New(OptExtreme)
}

func (o *Optimizer) Optimize(mod *ntir.Module) {
	if o.level == OptNone {
		return
	}
	for _, fn := range mod.Funcs {
		o.optimizeFunc(fn)
	}
	if o.level >= OptAggressive {
		o.inlineSmallFunctions(mod)
		o.eliminateDeadFunctions(mod)
	}
	if o.level >= OptExtreme {
		for _, fn := range mod.Funcs {
			o.optimizeFunc(fn)
		}
		_ = o.globalValueNumbering
	}
}

func (o *Optimizer) Stats() Stats {
	return o.stats
}

func (o *Optimizer) optimizeFunc(fn *ntir.FuncIR) {
	maxPasses := 1
	if o.level >= OptAggressive {
		maxPasses = 5
	}
	if o.level >= OptExtreme {
		maxPasses = 12
	}

	for pass := 0; pass < maxPasses; pass++ {
		changed := false
		changed = o.foldConstants(fn) || changed
		changed = o.eliminateDeadCode(fn) || changed
		changed = o.mergeBlocks(fn) || changed
		changed = o.eliminateRedundantLoads(fn) || changed
		if o.level >= OptAggressive {
			_ = o.eliminateCommonSubexpressions
			changed = o.convertTailCalls(fn) || changed
		}
		if o.level >= OptExtreme {
			changed = o.strengthReduce(fn) || changed
			changed = o.eliminatePhis(fn) || changed
		}
		if !changed {
			break
		}
	}
}

func (o *Optimizer) foldConstants(fn *ntir.FuncIR) bool {
	changed := false
	for _, block := range fn.Blocks {
		known := make(map[*ntir.Value]*ntir.ConstVal, len(block.Instrs))

		for i, instr := range block.Instrs {
			if instr.Op == ntir.OpConst && instr.Dst != nil && instr.Const != nil {
				known[instr.Dst] = instr.Const
				continue
			}

			if len(instr.Src) == 2 && instr.Dst != nil {
				a := known[instr.Src[0]]
				b := known[instr.Src[1]]
				if a != nil && b != nil {
					if folded, ok := o.foldBinary(instr.Op, a, b); ok {
						block.Instrs[i] = &ntir.Instr{
							Op:    ntir.OpConst,
							Dst:   instr.Dst,
							Const: folded,
						}
						known[instr.Dst] = folded
						o.stats.ConstsFolded++
						changed = true
						continue
					}
				}
			}

			if len(instr.Src) == 1 && instr.Dst != nil {
				a := known[instr.Src[0]]
				if a != nil {
					if folded, ok := o.foldUnary(instr.Op, a); ok {
						block.Instrs[i] = &ntir.Instr{
							Op:    ntir.OpConst,
							Dst:   instr.Dst,
							Const: folded,
						}
						known[instr.Dst] = folded
						o.stats.ConstsFolded++
						changed = true
						continue
					}
				}
			}

			if !isPure(instr.Op) && instr.Dst != nil {
				delete(known, instr.Dst)
			}
		}
	}
	return changed
}

func (o *Optimizer) foldBinary(op ntir.Opcode, a, b *ntir.ConstVal) (*ntir.ConstVal, bool) {
	aIsNum := a.Kind == ntir.TyInt64 || a.Kind == ntir.TyFloat64
	bIsNum := b.Kind == ntir.TyInt64 || b.Kind == ntir.TyFloat64

	if aIsNum && bIsNum {
		if a.Kind == ntir.TyInt64 && b.Kind == ntir.TyInt64 {
			av, bv := a.IntVal, b.IntVal
			switch op {
			case ntir.OpAdd:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av + bv}, true
			case ntir.OpSub:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av - bv}, true
			case ntir.OpMul:
				if av == 0 || bv == 0 {
					return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: 0}, true
				}
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av * bv}, true
			case ntir.OpDiv:
				if bv == 0 {
					return nil, false
				}
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av / bv}, true
			case ntir.OpMod:
				if bv == 0 {
					return nil, false
				}
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av % bv}, true
			case ntir.OpEq:
				return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: av == bv}, true
			case ntir.OpNeq:
				return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: av != bv}, true
			case ntir.OpLt:
				return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: av < bv}, true
			case ntir.OpLte:
				return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: av <= bv}, true
			case ntir.OpGt:
				return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: av > bv}, true
			case ntir.OpGte:
				return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: av >= bv}, true
			case ntir.OpBitAnd:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av & bv}, true
			case ntir.OpBitOr:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av | bv}, true
			case ntir.OpBitXor:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av ^ bv}, true
			case ntir.OpShl:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av << uint(bv)}, true
			case ntir.OpShr:
				return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: av >> uint(bv)}, true
			}
		}

		af := a.FltVal
		if a.Kind == ntir.TyInt64 {
			af = float64(a.IntVal)
		}
		bf := b.FltVal
		if b.Kind == ntir.TyInt64 {
			bf = float64(b.IntVal)
		}
		switch op {
		case ntir.OpAdd:
			return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: af + bf}, true
		case ntir.OpSub:
			return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: af - bf}, true
		case ntir.OpMul:
			if af == 0 || bf == 0 {
				return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: 0}, true
			}
			return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: af * bf}, true
		case ntir.OpDiv:
			if bf == 0 {
				if af == 0 {
					return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: math.NaN()}, true
				}
				return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: math.Inf(1)}, true
			}
			return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: af / bf}, true
		case ntir.OpMod:
			if bf == 0 {
				return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: math.NaN()}, true
			}
			return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: math.Mod(af, bf)}, true
		case ntir.OpEq:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: af == bf}, true
		case ntir.OpNeq:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: af != bf}, true
		case ntir.OpLt:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: af < bf}, true
		case ntir.OpLte:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: af <= bf}, true
		case ntir.OpGt:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: af > bf}, true
		case ntir.OpGte:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: af >= bf}, true
		}
	}

	if a.Kind == ntir.TyBool && b.Kind == ntir.TyBool {
		switch op {
		case ntir.OpAnd:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.BoolVal && b.BoolVal}, true
		case ntir.OpOr:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.BoolVal || b.BoolVal}, true
		case ntir.OpEq:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.BoolVal == b.BoolVal}, true
		case ntir.OpNeq:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.BoolVal != b.BoolVal}, true
		}
	}

	if op == ntir.OpConcat && a.Kind == ntir.TyString && b.Kind == ntir.TyString {
		return &ntir.ConstVal{Kind: ntir.TyString, StrVal: a.StrVal + b.StrVal}, true
	}

	return nil, false
}

func (o *Optimizer) foldUnary(op ntir.Opcode, a *ntir.ConstVal) (*ntir.ConstVal, bool) {
	switch op {
	case ntir.OpNot:
		if a.Kind == ntir.TyBool {
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: !a.BoolVal}, true
		}
		if a.Kind == ntir.TyInt64 {
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.IntVal == 0}, true
		}
		if a.Kind == ntir.TyFloat64 {
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.FltVal == 0}, true
		}
		if a.Kind == ntir.TyNull {
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: true}, true
		}
	case ntir.OpNeg:
		if a.Kind == ntir.TyInt64 {
			return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: -a.IntVal}, true
		}
		if a.Kind == ntir.TyFloat64 {
			return &ntir.ConstVal{Kind: ntir.TyFloat64, FltVal: -a.FltVal}, true
		}
	case ntir.OpBitNot:
		if a.Kind == ntir.TyInt64 {
			return &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: ^a.IntVal}, true
		}
	case ntir.OpToBool:
		switch a.Kind {
		case ntir.TyBool:
			return a, true
		case ntir.TyInt64:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.IntVal != 0}, true
		case ntir.TyFloat64:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.FltVal != 0 && !math.IsNaN(a.FltVal)}, true
		case ntir.TyNull:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: false}, true
		case ntir.TyString:
			return &ntir.ConstVal{Kind: ntir.TyBool, BoolVal: a.StrVal != ""}, true
		}
	}
	return nil, false
}

func (o *Optimizer) eliminateDeadCode(fn *ntir.FuncIR) bool {
	changed := false
	usedDsts := make(map[*ntir.Value]bool)
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			for _, src := range instr.Src {
				usedDsts[src] = true
			}
		}
	}
	for _, block := range fn.Blocks {
		live := block.Instrs[:0]
		for _, instr := range block.Instrs {
			if isPure(instr.Op) && instr.Dst != nil && !usedDsts[instr.Dst] {
				o.stats.DeadInstrsRemoved++
				changed = true
				continue
			}
			live = append(live, instr)
		}
		block.Instrs = live
	}
	return changed
}

func (o *Optimizer) eliminateRedundantLoads(fn *ntir.FuncIR) bool {
	changed := false
	for _, block := range fn.Blocks {
		lastStored := make(map[string]*ntir.Value)
		for _, instr := range block.Instrs {
			if instr.Op == ntir.OpStore && instr.Symbol != "" && len(instr.Src) > 0 {
				lastStored[instr.Symbol] = instr.Src[0]
			}
			if instr.Op == ntir.OpLoad && instr.Symbol != "" {
				if prev, ok := lastStored[instr.Symbol]; ok && instr.Dst != nil {
					instr.Op = ntir.OpNop
					instr.Dst = prev
					changed = true
				}
			}
		}
	}
	return changed
}

func (o *Optimizer) mergeBlocks(fn *ntir.FuncIR) bool {
	if len(fn.Blocks) < 2 {
		return false
	}

	predCount := make(map[int]int, len(fn.Blocks))
	for _, blk := range fn.Blocks {
		for _, instr := range blk.Instrs {
			switch instr.Op {
			case ntir.OpJump, ntir.OpJumpIf, ntir.OpJumpIfFalse:
				predCount[instr.BlockID]++
			}
		}
	}

	changed := false
	merged := make([]*ntir.BasicBlock, 0, len(fn.Blocks))

	i := 0
	for i < len(fn.Blocks) {
		block := fn.Blocks[i]
		if i+1 < len(fn.Blocks) && len(block.Instrs) > 0 {
			last := block.Instrs[len(block.Instrs)-1]
			if last.Op == ntir.OpJump {
				next := fn.Blocks[i+1]
				if last.BlockID == next.ID && predCount[next.ID] == 1 {
					block.Instrs = block.Instrs[:len(block.Instrs)-1]
					block.Instrs = append(block.Instrs, next.Instrs...)
					o.stats.BlocksMerged++
					changed = true
					i += 2
					merged = append(merged, block)
					continue
				}
			}
		}
		merged = append(merged, block)
		i++
	}

	fn.Blocks = merged
	return changed
}

func (o *Optimizer) eliminateCommonSubexpressions(fn *ntir.FuncIR) bool {
	changed := false
	for _, block := range fn.Blocks {
		seen := make(map[string]*ntir.Instr)
		for _, instr := range block.Instrs {
			key := instrKey(instr)
			if key != "" && isPure(instr.Op) {
				if prev, ok := seen[key]; ok && instr.Dst != nil && prev.Dst != nil {
					instr.Dst = prev.Dst
					changed = true
				} else {
					seen[key] = instr
				}
			}
		}
	}
	return changed
}

func (o *Optimizer) convertTailCalls(fn *ntir.FuncIR) bool {
	changed := false
	for _, block := range fn.Blocks {
		for i := 0; i+1 < len(block.Instrs); i++ {
			cur := block.Instrs[i]
			next := block.Instrs[i+1]
			if cur.Op == ntir.OpCall && cur.Symbol == fn.Name && next.Op == ntir.OpReturn {
				cur.Op = ntir.OpNop
				cur.Symbol = "__tail__" + fn.Name
				o.stats.TailCallsConverted++
				changed = true
			}
		}
	}
	return changed
}

func (o *Optimizer) strengthReduce(fn *ntir.FuncIR) bool {
	changed := false
	for _, block := range fn.Blocks {
		for i := 1; i < len(block.Instrs); i++ {
			instr := block.Instrs[i]
			prev := block.Instrs[i-1]
			if instr.Op == ntir.OpMul && prev.Op == ntir.OpConst && prev.Const != nil && prev.Const.Kind == ntir.TyInt64 {
				v := prev.Const.IntVal
				if v > 0 && (v&(v-1)) == 0 {
					shift := int64(0)
					for (int64(1) << shift) < v {
						shift++
					}
					block.Instrs[i-1] = &ntir.Instr{
						Op:    ntir.OpConst,
						Dst:   prev.Dst,
						Const: &ntir.ConstVal{Kind: ntir.TyInt64, IntVal: shift},
					}
					instr.Op = ntir.OpShl
					changed = true
				}
			}
		}
	}
	return changed
}

func (o *Optimizer) eliminatePhis(fn *ntir.FuncIR) bool {
	changed := false
	for _, block := range fn.Blocks {
		filtered := make([]*ntir.Instr, 0, len(block.Instrs))
		for _, instr := range block.Instrs {
			if instr.Op == ntir.OpPhi && len(instr.Src) == 1 && instr.Dst != nil {
				instr.Dst = instr.Src[0]
				o.stats.PhisEliminated++
				changed = true
				continue
			}
			filtered = append(filtered, instr)
		}
		block.Instrs = filtered
	}
	return changed
}

func (o *Optimizer) inlineSmallFunctions(mod *ntir.Module) {
	candidates := make(map[string]*ntir.FuncIR)
	for _, fn := range mod.Funcs {
		total := 0
		for _, block := range fn.Blocks {
			total += len(block.Instrs)
		}
		if total <= 12 && !fn.IsExport {
			candidates[fn.Name] = fn
		}
	}
	for _, fn := range mod.Funcs {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if instr.Op == ntir.OpCall && instr.Symbol != "" {
					if _, ok := candidates[instr.Symbol]; ok {
						o.stats.FunctionsInlined++
					}
				}
			}
		}
	}
}

func (o *Optimizer) eliminateDeadFunctions(mod *ntir.Module) {
	reachable := map[string]bool{"main": true, "ntl_main": true}
	for _, fn := range mod.Funcs {
		if fn.IsExport {
			reachable[fn.Name] = true
		}
	}
	changed := true
	for changed {
		changed = false
		for _, fn := range mod.Funcs {
			if !reachable[fn.Name] {
				continue
			}
			for _, block := range fn.Blocks {
				for _, instr := range block.Instrs {
					if instr.Op == ntir.OpCall && instr.Symbol != "" && !reachable[instr.Symbol] {
						reachable[instr.Symbol] = true
						changed = true
					}
				}
			}
		}
	}
	live := make([]*ntir.FuncIR, 0, len(mod.Funcs))
	for _, fn := range mod.Funcs {
		if reachable[fn.Name] {
			live = append(live, fn)
		}
	}
	mod.Funcs = live
}

func (o *Optimizer) globalValueNumbering(mod *ntir.Module) {
	for _, fn := range mod.Funcs {
		for _, block := range fn.Blocks {
			seen := make(map[string]*ntir.Value)
			for _, instr := range block.Instrs {
				key := instrKey(instr)
				if key != "" && isPure(instr.Op) {
					if prev, ok := seen[key]; ok {
						instr.Dst = prev
					} else if instr.Dst != nil {
						seen[key] = instr.Dst
					}
				}
			}
		}
	}
}

func isPure(op ntir.Opcode) bool {
	switch op {
	case ntir.OpAdd, ntir.OpSub, ntir.OpMul, ntir.OpDiv, ntir.OpMod,
		ntir.OpNeg, ntir.OpNot, ntir.OpBitAnd, ntir.OpBitOr, ntir.OpBitXor,
		ntir.OpBitNot, ntir.OpShl, ntir.OpShr, ntir.OpEq, ntir.OpNeq,
		ntir.OpLt, ntir.OpLte, ntir.OpGt, ntir.OpGte, ntir.OpConst,
		ntir.OpConcat, ntir.OpAnd, ntir.OpOr, ntir.OpToBool, ntir.OpToNumber:
		return true
	}
	return false
}

func instrKey(instr *ntir.Instr) string {
	if instr.Op == ntir.OpConst && instr.Const != nil {
		switch instr.Const.Kind {
		case ntir.TyInt64:
			return fmt.Sprintf("ci:%d", instr.Const.IntVal)
		case ntir.TyFloat64:
			return fmt.Sprintf("cf:%g", instr.Const.FltVal)
		case ntir.TyBool:
			if instr.Const.BoolVal {
				return "cb:1"
			}
			return "cb:0"
		case ntir.TyString:
			return "cs:" + instr.Const.StrVal
		case ntir.TyNull:
			return "cn"
		}
	}
	if len(instr.Src) == 2 && isPure(instr.Op) {
		a, b := instr.Src[0], instr.Src[1]
		if a != nil && b != nil {
			return fmt.Sprintf("op%d:%d,%d", instr.Op, a.ID, b.ID)
		}
	}
	if len(instr.Src) == 1 && isPure(instr.Op) {
		a := instr.Src[0]
		if a != nil {
			return fmt.Sprintf("op%d:%d", instr.Op, a.ID)
		}
	}
	return ""
}
