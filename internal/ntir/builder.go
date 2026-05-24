// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ntir

import (
        "fmt"
        "lunex/internal/ast"
        "strconv"
        "strings"
)

type Builder struct {
        module        *Module
        fn            *FuncIR
        block         *BasicBlock
        scopes        []map[string]*Value
        fnStack       []*FuncIR
        blkStack      []*BasicBlock
        labelID       int
        errors        []string
        breakTarget   *BasicBlock
        continueTarget *BasicBlock
}

func NewBuilder(module *Module) *Builder {
        return &Builder{module: module}
}

func (b *Builder) error(format string, args ...interface{}) {
        b.errors = append(b.errors, fmt.Sprintf(format, args...))
}

func (b *Builder) Errors() []string {
        return b.errors
}

func (b *Builder) pushScope() {
        b.scopes = append(b.scopes, make(map[string]*Value))
}

func (b *Builder) popScope() {
        if len(b.scopes) > 0 {
                b.scopes = b.scopes[:len(b.scopes)-1]
        }
}

func (b *Builder) define(name string, val *Value) {
        if len(b.scopes) == 0 {
                return
        }
        b.scopes[len(b.scopes)-1][name] = val
}

func (b *Builder) lookup(name string) (*Value, bool) {
        for i := len(b.scopes) - 1; i >= 0; i-- {
                if v, ok := b.scopes[i][name]; ok {
                        return v, true
                }
        }
        return nil, false
}

func (b *Builder) emit(op Opcode, dst *Value, srcs ...*Value) *Instr {
        instr := &Instr{Op: op, Dst: dst, Src: srcs}
        b.block.Add(instr)
        return instr
}

func (b *Builder) emitConst(dst *Value, cv *ConstVal) *Instr {
        instr := &Instr{Op: OpConst, Dst: dst, Const: cv}
        b.block.Add(instr)
        return instr
}

func (b *Builder) emitJump(target *BasicBlock) {
        instr := &Instr{Op: OpJump, BlockID: target.ID}
        b.block.Add(instr)
}

func (b *Builder) emitJumpIf(cond *Value, thenBlk, elseBlk *BasicBlock) {
        instr := &Instr{Op: OpJumpIf, Src: []*Value{cond}, BlockID: thenBlk.ID}
        b.block.Add(instr)
        instr2 := &Instr{Op: OpJump, BlockID: elseBlk.ID}
        b.block.Add(instr2)
}

func (b *Builder) switchBlock(blk *BasicBlock) {
        b.block = blk
}

func (b *Builder) pushFunc(f *FuncIR) {
        b.fnStack = append(b.fnStack, b.fn)
        b.blkStack = append(b.blkStack, b.block)
        b.fn = f
        b.block = f.Entry
}

func (b *Builder) popFunc() {
        if len(b.fnStack) > 0 {
                b.fn = b.fnStack[len(b.fnStack)-1]
                b.fnStack = b.fnStack[:len(b.fnStack)-1]
                b.block = b.blkStack[len(b.blkStack)-1]
                b.blkStack = b.blkStack[:len(b.blkStack)-1]
        }
}

func (b *Builder) BuildModule(node *ast.Node) *Module {
        b.pushScope()
        b.buildProgram(node)
        b.popScope()
        return b.module
}

func (b *Builder) buildProgram(node *ast.Node) {
        if node == nil || node.Type != ast.Program {
                return
        }
        mainFn := NewFuncIR("__lunex_main__")
        mainFn.IsExport = true
        b.module.AddFunc(mainFn)
        b.pushFunc(mainFn)
        b.pushScope()

        for _, stmt := range node.Body_ {
                b.buildStmt(stmt)
        }

        retVal := b.fn.NewValue(TyVoidT)
        b.emit(OpReturn, retVal)

        b.popScope()
        b.popFunc()
}

func (b *Builder) buildStmt(node *ast.Node) {
        if node == nil {
                return
        }
        switch node.Type {
        case ast.VarDecl:
                b.buildVarDecl(node)
        case ast.FnDecl:
                b.buildFnDecl(node)
        case ast.ExprStmt:
                b.buildExpr(node.Expr)
        case ast.ReturnStmt:
                b.buildReturn(node)
        case ast.IfStmt:
                b.buildIf(node)
        case ast.WhileStmt:
                b.buildWhile(node)
        case ast.ForStmt:
                b.buildFor(node)
        case ast.EachInStmt:
                b.buildEachIn(node)
        case ast.Block:
                b.buildBlock(node)
        case ast.LogStmt:
                b.buildLog(node)
        case ast.ThrowStmt, ast.RaiseStmt:
                b.buildThrow(node)
        case ast.TryStmt:
                b.buildTry(node)
        case ast.BreakStmt:
                b.buildBreak(node)
        case ast.ContinueStmt:
                b.buildContinue(node)
        case ast.ClassDecl:
                b.buildClassDecl(node)
        case ast.ImportDecl:
                b.buildImport(node)
        case ast.SpawnStmt:
                b.buildSpawn(node)
        default:
                b.buildExpr(node)
        }
}

func (b *Builder) buildVarDecl(node *ast.Node) {
        var initVal *Value
        if node.Init != nil {
                initVal = b.buildExpr(node.Init)
        } else {
                dst := b.fn.NewValue(TyAnyT)
                b.emitConst(dst, &ConstVal{Kind: TyNull})
                initVal = dst
        }
        named := b.fn.NewNamedValue(node.Name, TyAnyT)
        storeInstr := &Instr{Op: OpStore, Dst: named, Src: []*Value{initVal}, Symbol: node.Name}
        b.block.Add(storeInstr)
        b.define(node.Name, named)
}

func (b *Builder) buildFnDecl(node *ast.Node) {
        fn := b.buildFuncIR(node.Name, node.Params, node.Body)
        b.module.AddFunc(fn)
        fnVal := b.fn.NewNamedValue(node.Name, TyFuncT)
        b.emitConst(fnVal, &ConstVal{Kind: TyFunc, StrVal: node.Name})
        b.define(node.Name, fnVal)
}

func (b *Builder) buildFuncIR(name string, params []*ast.Param, body *ast.Node) *FuncIR {
        fn := NewFuncIR(name)
        b.pushFunc(fn)
        b.pushScope()

        for _, p := range params {
                paramVal := fn.NewNamedValue(p.Name, TyAnyT)
                fn.Params = append(fn.Params, paramVal)
                b.define(p.Name, paramVal)
        }

        if body != nil {
                for _, stmt := range body.Body_ {
                        b.buildStmt(stmt)
                }
        }

        lastBlock := fn.Blocks[len(fn.Blocks)-1]
        hasReturn := false
        for _, instr := range lastBlock.Instrs {
                if instr.Op == OpReturn {
                        hasReturn = true
                        break
                }
        }
        if !hasReturn {
                retVal := fn.NewValue(TyNullT)
                b.emitConst(retVal, &ConstVal{Kind: TyNull})
                b.emit(OpReturn, nil, retVal)
        }

        b.popScope()
        b.popFunc()
        return fn
}

func (b *Builder) buildBlock(node *ast.Node) {
        b.pushScope()
        for _, stmt := range node.Body_ {
                b.buildStmt(stmt)
        }
        b.popScope()
}

func (b *Builder) buildReturn(node *ast.Node) {
        var retVal *Value
        if node.Expr != nil {
                retVal = b.buildExpr(node.Expr)
        } else {
                retVal = b.fn.NewValue(TyNullT)
                b.emitConst(retVal, &ConstVal{Kind: TyNull})
        }
        b.emit(OpReturn, nil, retVal)
}

func (b *Builder) buildIf(node *ast.Node) {
        condVal := b.buildExpr(node.Test)
        thenBlk := b.fn.NewBlock("then")
        afterBlk := b.fn.NewBlock("after_if")

        if node.Alternate != nil {
                elseBlk := b.fn.NewBlock("else")
                b.emitJumpIf(condVal, thenBlk, elseBlk)
                b.switchBlock(thenBlk)
                b.buildStmt(node.Consequent)
                b.emitJump(afterBlk)
                b.switchBlock(elseBlk)
                b.buildStmt(node.Alternate)
                b.emitJump(afterBlk)
        } else {
                b.emitJumpIf(condVal, thenBlk, afterBlk)
                b.switchBlock(thenBlk)
                b.buildStmt(node.Consequent)
                b.emitJump(afterBlk)
        }
        b.switchBlock(afterBlk)
}

func (b *Builder) buildWhile(node *ast.Node) {
        condBlk := b.fn.NewBlock("while_cond")
        bodyBlk := b.fn.NewBlock("while_body")
        afterBlk := b.fn.NewBlock("while_after")

        prevBreak := b.breakTarget
        prevCont := b.continueTarget
        b.breakTarget = afterBlk
        b.continueTarget = condBlk

        b.emitJump(condBlk)
        b.switchBlock(condBlk)
        condVal := b.buildExpr(node.Test)
        b.emitJumpIf(condVal, bodyBlk, afterBlk)
        b.switchBlock(bodyBlk)
        b.buildStmt(node.Body)
        b.emitJump(condBlk)
        b.switchBlock(afterBlk)

        b.breakTarget = prevBreak
        b.continueTarget = prevCont
}

func (b *Builder) buildFor(node *ast.Node) {
        b.pushScope()
        if node.Init != nil {
                b.buildStmt(node.Init)
        }
        condBlk := b.fn.NewBlock("for_cond")
        bodyBlk := b.fn.NewBlock("for_body")
        incrBlk := b.fn.NewBlock("for_incr")
        afterBlk := b.fn.NewBlock("for_after")

        prevBreak := b.breakTarget
        prevCont := b.continueTarget
        b.breakTarget = afterBlk
        b.continueTarget = incrBlk

        b.emitJump(condBlk)
        b.switchBlock(condBlk)
        if node.Test != nil {
                condVal := b.buildExpr(node.Test)
                b.emitJumpIf(condVal, bodyBlk, afterBlk)
        } else {
                b.emitJump(bodyBlk)
        }
        b.switchBlock(bodyBlk)
        b.buildStmt(node.Body)
        b.emitJump(incrBlk)
        b.switchBlock(incrBlk)
        if node.Right != nil {
                b.buildExpr(node.Right)
        }
        b.emitJump(condBlk)
        b.switchBlock(afterBlk)
        b.popScope()

        b.breakTarget = prevBreak
        b.continueTarget = prevCont
}

func (b *Builder) buildEachIn(node *ast.Node) {
        iterVal := b.buildExpr(node.Subject)
        idxVal := b.fn.NewValue(TyInt64T)
        b.emitConst(idxVal, &ConstVal{Kind: TyInt64, IntVal: 0})

        condBlk := b.fn.NewBlock("each_cond")
        bodyBlk := b.fn.NewBlock("each_body")
        afterBlk := b.fn.NewBlock("each_after")

        prevBreak := b.breakTarget
        prevCont := b.continueTarget
        b.breakTarget = afterBlk
        b.continueTarget = condBlk

        b.emitJump(condBlk)
        b.switchBlock(condBlk)
        lenVal := b.fn.NewValue(TyInt64T)
        b.emit(OpLen, lenVal, iterVal)
        cmpVal := b.fn.NewValue(TyBoolT)
        b.emit(OpLt, cmpVal, idxVal, lenVal)
        b.emitJumpIf(cmpVal, bodyBlk, afterBlk)
        b.switchBlock(bodyBlk)
        b.pushScope()

        elemVal := b.fn.NewValue(TyAnyT)
        b.emit(OpGetIndex, elemVal, iterVal, idxVal)
        if node.Name != "" {
                elemNamed := b.fn.NewNamedValue(node.Name, TyAnyT)
                b.emit(OpStore, elemNamed, elemVal)
                b.define(node.Name, elemNamed)
        }
        b.buildStmt(node.Body)
        b.popScope()

        one := b.fn.NewValue(TyInt64T)
        b.emitConst(one, &ConstVal{Kind: TyInt64, IntVal: 1})
        newIdx := b.fn.NewValue(TyInt64T)
        b.emit(OpAdd, newIdx, idxVal, one)
        b.emit(OpStore, idxVal, newIdx)
        b.emitJump(condBlk)
        b.switchBlock(afterBlk)

        b.breakTarget = prevBreak
        b.continueTarget = prevCont
}

func (b *Builder) buildLog(node *ast.Node) {
        for _, arg := range node.Args {
                val := b.buildExpr(arg)
                strVal := b.fn.NewValue(TyStringT)
                b.emit(OpToString, strVal, val)
                b.emit(OpCallRuntime, nil, strVal)
                b.block.Instrs[len(b.block.Instrs)-1].Symbol = "ntl_rt_print"
        }
}

func (b *Builder) buildThrow(node *ast.Node) {
        var val *Value
        if node.Expr != nil {
                val = b.buildExpr(node.Expr)
        } else {
                val = b.fn.NewValue(TyNullT)
                b.emitConst(val, &ConstVal{Kind: TyNull})
        }
        b.emit(OpThrow, nil, val)
}

func (b *Builder) buildTry(node *ast.Node) {
        if node.Body != nil {
                b.buildStmt(node.Body)
        }
}

func (b *Builder) buildBreak(_ *ast.Node) {
        if b.breakTarget != nil {
                b.emitJump(b.breakTarget)
        }
}

func (b *Builder) buildContinue(_ *ast.Node) {
        if b.continueTarget != nil {
                b.emitJump(b.continueTarget)
        }
}

func (b *Builder) buildClassDecl(node *ast.Node) {
        name := node.Name
        fnName := "__class_" + name + "_new__"
        fn := NewFuncIR(fnName)
        b.module.AddFunc(fn)
        b.pushFunc(fn)
        b.pushScope()

        objVal := fn.NewValue(TyObjectT)
        b.emit(OpMakeObject, objVal)

        for _, member := range node.Methods {
                if member.Kind == "method" {
                        mfn := b.buildFuncIR(name+"_"+member.Name, member.Params, member.Body)
                        b.module.AddFunc(mfn)
                        mfnVal := fn.NewValue(TyFuncT)
                        b.emitConst(mfnVal, &ConstVal{Kind: TyFunc, StrVal: mfn.Name})
                        keyVal := fn.NewValue(TyStringT)
                        b.emitConst(keyVal, &ConstVal{Kind: TyString, StrVal: member.Name})
                        b.emit(OpSetField, nil, objVal, keyVal, mfnVal)
                } else if member.Kind == "field" && member.Init != nil {
                        fieldVal := b.buildExpr(member.Init)
                        keyVal := fn.NewValue(TyStringT)
                        b.emitConst(keyVal, &ConstVal{Kind: TyString, StrVal: member.Name})
                        b.emit(OpSetField, nil, objVal, keyVal, fieldVal)
                }
        }

        b.emit(OpReturn, nil, objVal)
        b.popScope()
        b.popFunc()

        classVal := b.fn.NewNamedValue(name, TyFuncT)
        b.emitConst(classVal, &ConstVal{Kind: TyFunc, StrVal: fnName})
        b.define(name, classVal)
}

func (b *Builder) buildImport(node *ast.Node) {
        b.module.Imports = append(b.module.Imports, node.Source)
}

func (b *Builder) buildSpawn(node *ast.Node) {
        if node.Expr != nil {
                val := b.buildExpr(node.Expr)
                b.emit(OpCallRuntime, nil, val)
                b.block.Instrs[len(b.block.Instrs)-1].Symbol = "ntl_rt_spawn"
        }
}

func (b *Builder) buildExpr(node *ast.Node) *Value {
        if node == nil {
                v := b.fn.NewValue(TyNullT)
                b.emitConst(v, &ConstVal{Kind: TyNull})
                return v
        }
        switch node.Type {
        case ast.NumberLit:
                return b.buildNumber(node)
        case ast.StringLit:
                return b.buildString(node)
        case ast.BoolLit:
                return b.buildBool(node)
        case ast.NullLit:
                v := b.fn.NewValue(TyNullT)
                b.emitConst(v, &ConstVal{Kind: TyNull})
                return v
        case ast.UndefinedLit:
                v := b.fn.NewValue(TyNullT)
                b.emitConst(v, &ConstVal{Kind: TyNull})
                return v
        case ast.Identifier:
                return b.buildIdentifier(node)
        case ast.BinaryExpr:
                return b.buildBinaryExpr(node)
        case ast.UnaryExpr:
                return b.buildUnaryExpr(node)
        case ast.AssignExpr:
                return b.buildAssign(node)
        case ast.CallExpr:
                return b.buildCall(node)
        case ast.MemberExpr:
                return b.buildMember(node)
        case ast.ArrayLit:
                return b.buildArray(node)
        case ast.ObjectLit:
                return b.buildObject(node)
        case ast.FnExpr, ast.ArrowFn:
                return b.buildFuncExpr(node)
        case ast.TemplateLit:
                return b.buildTemplate(node)
        case ast.TernaryExpr:
                return b.buildTernary(node)
        case ast.NewExpr:
                return b.buildNew(node)
        case ast.LogicalExpr:
                return b.buildLogical(node)
        default:
                v := b.fn.NewValue(TyNullT)
                b.emitConst(v, &ConstVal{Kind: TyNull})
                return v
        }
}

func (b *Builder) buildNumber(node *ast.Node) *Value {
        v := b.fn.NewValue(TyAnyT)
        raw := fmt.Sprintf("%v", node.Value)
        raw = strings.ReplaceAll(raw, "_", "")
        if strings.ContainsAny(raw, ".eE") {
                f, err := strconv.ParseFloat(raw, 64)
                if err == nil {
                        b.emitConst(v, &ConstVal{Kind: TyFloat64, FltVal: f})
                        return v
                }
        }
        if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
                n, err := strconv.ParseInt(raw[2:], 16, 64)
                if err == nil {
                        b.emitConst(v, &ConstVal{Kind: TyInt64, IntVal: n})
                        return v
                }
        }
        n, err := strconv.ParseInt(raw, 10, 64)
        if err == nil {
                b.emitConst(v, &ConstVal{Kind: TyInt64, IntVal: n})
        } else {
                f, _ := strconv.ParseFloat(raw, 64)
                b.emitConst(v, &ConstVal{Kind: TyFloat64, FltVal: f})
        }
        return v
}

func (b *Builder) buildString(node *ast.Node) *Value {
        v := b.fn.NewValue(TyStringT)
        s := fmt.Sprintf("%v", node.Value)
        b.emitConst(v, &ConstVal{Kind: TyString, StrVal: s})
        return v
}

func (b *Builder) buildBool(node *ast.Node) *Value {
        v := b.fn.NewValue(TyBoolT)
        bv := node.Value == true || node.Value == "true"
        b.emitConst(v, &ConstVal{Kind: TyBool, BoolVal: bv})
        return v
}

func (b *Builder) buildIdentifier(node *ast.Node) *Value {
        name := node.Name
        if v, ok := b.lookup(name); ok {
                loaded := b.fn.NewValue(TyAnyT)
                b.emit(OpLoad, loaded, v)
                return loaded
        }
        v := b.fn.NewValue(TyAnyT)
        instr := &Instr{Op: OpCallRuntime, Dst: v, Symbol: "ntl_rt_load_global"}
        key := b.fn.NewValue(TyStringT)
        b.emitConst(key, &ConstVal{Kind: TyString, StrVal: name})
        instr.Src = []*Value{key}
        b.block.Add(instr)
        return v
}

func (b *Builder) buildBinaryExpr(node *ast.Node) *Value {
        left := b.buildExpr(node.Left)
        right := b.buildExpr(node.Right)
        dst := b.fn.NewValue(TyAnyT)
        var op Opcode
        switch node.Op {
        case "+":
                op = OpAdd
        case "-":
                op = OpSub
        case "*":
                op = OpMul
        case "/":
                op = OpDiv
        case "%":
                op = OpMod
        case "**":
                op = OpCallRuntime
                b.emit(OpCallRuntime, dst, left, right)
                b.block.Instrs[len(b.block.Instrs)-1].Symbol = "ntl_rt_pow"
                return dst
        case "==", "===":
                op = OpEq
        case "!=", "!==":
                op = OpNeq
        case "<":
                op = OpLt
        case "<=":
                op = OpLte
        case ">":
                op = OpGt
        case ">=":
                op = OpGte
        case "&":
                op = OpBitAnd
        case "|":
                op = OpBitOr
        case "^":
                op = OpBitXor
        case "<<":
                op = OpShl
        case ">>":
                op = OpShr
        default:
                op = OpAdd
        }
        b.emit(op, dst, left, right)
        return dst
}

func (b *Builder) buildUnaryExpr(node *ast.Node) *Value {
        val := b.buildExpr(node.Arg)
        dst := b.fn.NewValue(TyAnyT)
        switch node.Op {
        case "-":
                b.emit(OpNeg, dst, val)
        case "!":
                b.emit(OpNot, dst, val)
        case "~":
                b.emit(OpBitNot, dst, val)
        case "typeof":
                b.emit(OpCallRuntime, dst, val)
                b.block.Instrs[len(b.block.Instrs)-1].Symbol = "ntl_rt_typeof"
        default:
                b.emit(OpNop, dst, val)
        }
        return dst
}

func (b *Builder) buildAssign(node *ast.Node) *Value {
        val := b.buildExpr(node.Right)
        if node.Left != nil {
                switch node.Left.Type {
                case ast.Identifier:
                        name := node.Left.Name
                        if v, ok := b.lookup(name); ok {
                                if node.Op == "=" {
                                        b.emit(OpStore, v, val)
                                } else {
                                        loaded := b.fn.NewValue(TyAnyT)
                                        b.emit(OpLoad, loaded, v)
                                        res := b.fn.NewValue(TyAnyT)
                                        switch node.Op {
                                        case "+=":
                                                b.emit(OpAdd, res, loaded, val)
                                        case "-=":
                                                b.emit(OpSub, res, loaded, val)
                                        case "*=":
                                                b.emit(OpMul, res, loaded, val)
                                        case "/=":
                                                b.emit(OpDiv, res, loaded, val)
                                        case "%=":
                                                b.emit(OpMod, res, loaded, val)
                                        default:
                                                b.emit(OpAdd, res, loaded, val)
                                        }
                                        b.emit(OpStore, v, res)
                                        return res
                                }
                        } else {
                                newV := b.fn.NewNamedValue(name, TyAnyT)
                                b.emit(OpStore, newV, val)
                                b.define(name, newV)
                        }
                case ast.MemberExpr:
                        obj := b.buildExpr(node.Left.Object)
                        keyNode := node.Left
                        keyVal := b.fn.NewValue(TyStringT)
                        if keyNode.Computed && keyNode.Prop != nil {
                                keyVal = b.buildExpr(keyNode.Prop.(*ast.Node))
                        } else {
                                propName := fmt.Sprintf("%v", keyNode.Prop)
                                b.emitConst(keyVal, &ConstVal{Kind: TyString, StrVal: propName})
                        }
                        b.emit(OpSetField, nil, obj, keyVal, val)
                }
        }
        return val
}

func moduleMethodRTSymbol(moduleName, methodName string) string {
        switch moduleName {
        case "io":
                switch methodName {
                case "log", "println":
                        return "ntl_rt_io_log"
                case "print":
                        return "ntl_rt_io_print"
                case "error", "err":
                        return "ntl_rt_io_error"
                case "warn":
                        return "ntl_rt_io_warn"
                case "read", "readline":
                        return "ntl_rt_io_readline"
                }
        case "math":
                switch methodName {
                case "sqrt":
                        return "ntl_rt_math_sqrt"
                case "floor":
                        return "ntl_rt_math_floor"
                case "ceil":
                        return "ntl_rt_math_ceil"
                case "round":
                        return "ntl_rt_math_round"
                case "abs":
                        return "ntl_rt_math_abs"
                case "pow":
                        return "ntl_rt_math_pow"
                case "log", "ln":
                        return "ntl_rt_math_log"
                case "max":
                        return "ntl_rt_math_max"
                case "min":
                        return "ntl_rt_math_min"
                case "random", "rand":
                        return "ntl_rt_math_random"
                case "sin":
                        return "ntl_rt_math_sin"
                case "cos":
                        return "ntl_rt_math_cos"
                case "tan":
                        return "ntl_rt_math_tan"
                }
        case "os":
                switch methodName {
                case "exit":
                        return "ntl_rt_os_exit"
                case "getenv", "env":
                        return "ntl_rt_os_getenv"
                case "args":
                        return "ntl_rt_os_args"
                case "pid":
                        return "ntl_rt_os_pid"
                case "cwd":
                        return "ntl_rt_os_cwd"
                case "time":
                        return "ntl_rt_os_time"
                case "sleep":
                        return "ntl_rt_os_sleep"
                }
        case "str":
                switch methodName {
                case "length", "len":
                        return "ntl_rt_str_length"
                case "upper", "toUpper":
                        return "ntl_rt_str_upper"
                case "lower", "toLower":
                        return "ntl_rt_str_lower"
                case "trim":
                        return "ntl_rt_str_trim"
                case "contains":
                        return "ntl_rt_str_contains"
                case "startsWith":
                        return "ntl_rt_str_startswith"
                case "endsWith":
                        return "ntl_rt_str_endswith"
                case "indexOf":
                        return "ntl_rt_str_indexof"
                case "slice", "substring":
                        return "ntl_rt_str_slice"
                case "replace":
                        return "ntl_rt_str_replace"
                case "split":
                        return "ntl_rt_str_split"
                case "repeat":
                        return "ntl_rt_str_repeat"
                case "format", "sprintf":
                        return "ntl_rt_str_format"
                }
        }
        return ""
}

func (b *Builder) buildCall(node *ast.Node) *Value {
        dst := b.fn.NewValue(TyAnyT)
        args := make([]*Value, 0, len(node.Args))
        for _, arg := range node.Args {
                args = append(args, b.buildExpr(arg))
        }

        if node.Callee != nil && node.Callee.Type == ast.MemberExpr {
                if node.Callee.Object != nil && node.Callee.Object.Type == ast.Identifier {
                        moduleName := node.Callee.Object.Name
                        methodName := fmt.Sprintf("%v", node.Callee.Prop)
                        if rtSym := moduleMethodRTSymbol(moduleName, methodName); rtSym != "" {
                                instr := b.emit(OpCallRuntime, dst, args...)
                                instr.Symbol = rtSym
                                return dst
                        }
                }

                obj := b.buildExpr(node.Callee.Object)
                propName := fmt.Sprintf("%v", node.Callee.Prop)
                keyVal := b.fn.NewValue(TyStringT)
                b.emitConst(keyVal, &ConstVal{Kind: TyString, StrVal: propName})
                methodVal := b.fn.NewValue(TyAnyT)
                b.emit(OpGetField, methodVal, obj, keyVal)
                allArgs := append([]*Value{obj, methodVal}, args...)
                b.emit(OpCall, dst, allArgs...)
                return dst
        }

        if node.Callee != nil && node.Callee.Type == ast.Identifier {
                name := node.Callee.Name
                if v, ok := b.lookup(name); ok && v != nil && v.Type != nil && v.Type.Kind == TyFunc {
                        instr := b.emit(OpCall, dst, args...)
                        instr.Symbol = name
                        return dst
                }
        }

        callee := b.buildExpr(node.Callee)
        allArgs := append([]*Value{callee}, args...)
        b.emit(OpCall, dst, allArgs...)
        return dst
}

func (b *Builder) buildMember(node *ast.Node) *Value {
        obj := b.buildExpr(node.Object)
        dst := b.fn.NewValue(TyAnyT)
        if node.Computed {
                if node.Prop != nil {
                        if propNode, ok := node.Prop.(*ast.Node); ok {
                                idx := b.buildExpr(propNode)
                                b.emit(OpGetIndex, dst, obj, idx)
                        }
                }
        } else {
                propName := fmt.Sprintf("%v", node.Prop)
                keyVal := b.fn.NewValue(TyStringT)
                b.emitConst(keyVal, &ConstVal{Kind: TyString, StrVal: propName})
                b.emit(OpGetField, dst, obj, keyVal)
        }
        return dst
}

func (b *Builder) buildArray(node *ast.Node) *Value {
        dst := b.fn.NewValue(TyArrayT)
        elems := make([]*Value, 0, len(node.Elements))
        for _, el := range node.Elements {
                elems = append(elems, b.buildExpr(el))
        }
        b.emit(OpMakeArray, dst, elems...)
        return dst
}

func (b *Builder) buildObject(node *ast.Node) *Value {
        dst := b.fn.NewValue(TyObjectT)
        b.emit(OpMakeObject, dst)
        for _, prop := range node.Properties {
                var keyStr string
                switch k := prop.Key.(type) {
                case string:
                        keyStr = k
                default:
                        keyStr = fmt.Sprintf("%v", k)
                }
                keyVal := b.fn.NewValue(TyStringT)
                b.emitConst(keyVal, &ConstVal{Kind: TyString, StrVal: keyStr})
                var valV *Value
                if prop.Value != nil {
                        valV = b.buildExpr(prop.Value)
                } else {
                        valV = b.fn.NewValue(TyNullT)
                        b.emitConst(valV, &ConstVal{Kind: TyNull})
                }
                b.emit(OpSetField, nil, dst, keyVal, valV)
        }
        return dst
}

func (b *Builder) buildFuncExpr(node *ast.Node) *Value {
        name := node.Name
        if name == "" {
                name = fmt.Sprintf("__anon_%d__", b.fn.valueID)
        }
        fn := b.buildFuncIR(name, node.Params, node.Body)
        b.module.AddFunc(fn)
        v := b.fn.NewValue(TyFuncT)
        b.emitConst(v, &ConstVal{Kind: TyFunc, StrVal: fn.Name})
        return v
}

func (b *Builder) buildTemplate(node *ast.Node) *Value {
        raw := fmt.Sprintf("%v", node.Value)
        _ = raw
        dst := b.fn.NewValue(TyStringT)
        b.emitConst(dst, &ConstVal{Kind: TyString, StrVal: raw})
        return dst
}

func (b *Builder) buildTernary(node *ast.Node) *Value {
        condVal := b.buildExpr(node.Test)
        thenBlk := b.fn.NewBlock("tern_then")
        elseBlk := b.fn.NewBlock("tern_else")
        afterBlk := b.fn.NewBlock("tern_after")

        dst := b.fn.NewValue(TyAnyT)
        b.emitJumpIf(condVal, thenBlk, elseBlk)

        b.switchBlock(thenBlk)
        thenVal := b.buildExpr(node.Consequent)
        b.emit(OpStore, dst, thenVal)
        b.emitJump(afterBlk)

        b.switchBlock(elseBlk)
        elseVal := b.buildExpr(node.Alternate)
        b.emit(OpStore, dst, elseVal)
        b.emitJump(afterBlk)

        b.switchBlock(afterBlk)
        loaded := b.fn.NewValue(TyAnyT)
        b.emit(OpLoad, loaded, dst)
        return loaded
}

func (b *Builder) buildNew(node *ast.Node) *Value {
        dst := b.fn.NewValue(TyObjectT)
        args := make([]*Value, 0, len(node.Args)+1)
        callee := b.buildExpr(node.Callee)
        args = append(args, callee)
        for _, arg := range node.Args {
                args = append(args, b.buildExpr(arg))
        }
        b.emit(OpCallRuntime, dst, args...)
        b.block.Instrs[len(b.block.Instrs)-1].Symbol = "ntl_rt_new"
        return dst
}

func (b *Builder) buildLogical(node *ast.Node) *Value {
        left := b.buildExpr(node.Left)
        dst := b.fn.NewValue(TyAnyT)

        switch node.Op {
        case "&&":
                rhsBlk := b.fn.NewBlock("and_rhs")
                afterBlk := b.fn.NewBlock("and_after")
                b.emit(OpStore, dst, left)
                b.emitJumpIf(left, rhsBlk, afterBlk)
                b.switchBlock(rhsBlk)
                right := b.buildExpr(node.Right)
                b.emit(OpStore, dst, right)
                b.emitJump(afterBlk)
                b.switchBlock(afterBlk)
        case "||":
                rhsBlk := b.fn.NewBlock("or_rhs")
                afterBlk := b.fn.NewBlock("or_after")
                b.emit(OpStore, dst, left)
                b.emitJumpIf(left, afterBlk, rhsBlk)
                b.switchBlock(rhsBlk)
                right := b.buildExpr(node.Right)
                b.emit(OpStore, dst, right)
                b.emitJump(afterBlk)
                b.switchBlock(afterBlk)
        default:
                b.emit(OpStore, dst, left)
        }

        loaded := b.fn.NewValue(TyAnyT)
        b.emit(OpLoad, loaded, dst)
        return loaded
}
