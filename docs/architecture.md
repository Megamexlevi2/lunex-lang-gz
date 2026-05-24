# Lunex lang — Architecture

The Lunex compiler is written in Go. The runtime and JIT are written in Zig.
Both are compiled into a **single binary** (`lunex`) — the Zig runtime is embedded at link time via `go:embed`.
There is no CGo, no shared memory, no FFI. The Go compiler writes bytecode to a pipe; the Zig runtime reads and executes it.

## Pipeline

```
Lunex Source (.lx)
      │
  ┌───▼───┐
  │ Lexer │   tokenize source text
  └───┬───┘
  ┌───▼────┐
  │ Parser │  recursive descent → AST
  └───┬────┘
  ┌───▼─────┐
  │  NTLIR  │  flat intermediate representation (SSA-like)
  │ + ENFS  │  up to 12-pass IR optimizer
  └───┬─────┘
  ┌───▼──────┐
  │ Bytecode │  compact .nc format
  └───┬──────┘
      │  subprocess pipe (Go writes → Zig reads)
  ┌───▼──────┐
  │  Lunex VM  │  register-based bytecode interpreter
  └───┬──────┘
  ┌───▼──────┐
  │ Profiler │  counts calls + loop back-edges per function
  └───┬──────┘
  ┌───▼─────┐
  │   JIT   │  emits real x86_64 / AArch64 machine code
  └───┬─────┘
  ┌───▼─────┐
  │   CPU   │  executes native machine code directly
  └─────────┘
```

## Why Go + Zig?

| Layer                     | Language | Reason                                                   |
|---------------------------|----------|----------------------------------------------------------|
| Frontend (lexer→bytecode) | Go       | Fast build, rich stdlib, easy string/AST work            |
| Backend (VM + JIT)        | Zig      | Direct memory control, inline ASM for JIT, zero hidden allocations |

## Tiered JIT

| Tier        | Threshold                          | What happens                                   |
|-------------|------------------------------------|------------------------------------------------|
| Interpreter | always                             | Full bytecode dispatch loop                    |
| Profiler    | every call & loop                  | Counts invocations and back-edges              |
| JIT Tier 1  | 100 calls OR 1 000 loop iterations | Function compiled to native machine code       |
| Native      | after JIT                          | Compiled unit runs directly on CPU             |

Type feedback is collected per call site. Monomorphic integer functions get specialized machine code with no boxing and no type checks.

## ENFS — Extreme Native Fast System

ENFS is the built-in IR optimizer. It runs automatically on every compilation.

- Constant folding
- Dead code elimination
- Common subexpression elimination
- Block merging
- Redundant load elimination
- Strength reduction (multiply-by-power-of-two → shift)
- Tail call conversion
- Phi elimination
- Global value numbering
- Function inlining
- Dead function elimination

## Source Layout

```
lunex/
├── main.go                 entry point — CLI dispatch
├── embed.go                embed lib/ into the binary
├── embed_unix.go           embed bin/lunex-rt  (non-Windows)
├── embed_windows.go        embed bin/lunex-rt.exe
├── internal/
│   ├── lexer/              tokenizer
│   ├── parser/             recursive descent parser → AST
│   ├── ast/                AST node types
│   ├── ntir/               flat IR (SSA-like)
│   ├── bytecode/           .nc format + Go reference VM
│   ├── compiler/           AST → IR → bytecode
│   ├── aot/                AOT / embedded bytecode support
│   ├── builtin/            Go stdlib module implementations
│   ├── runtime/            Go → Zig pipe bridge
│   ├── zigrt/              Zig runtime extraction + launch
│   └── repl/               interactive REPL
├── zig/
│   ├── build.zig           build system + extension scanner
│   ├── build.zig.zon       package manifest (Zig 0.14+)
│   └── src/
│       ├── main.zig        runtime entry point
│       ├── vm.zig          bytecode VM
│       ├── value.zig       Value tagged union
│       ├── jit.zig         tiered JIT — x86_64 + AArch64
│       ├── profiler.zig    call + loop profiler
│       ├── platform.zig    mmap, icache flush, OS/arch detection
│       ├── memory.zig      arena + pool allocators
│       ├── builtins.zig    built-in functions
│       ├── extensions_gen.zig   AUTO-GENERATED — do not edit
│       └── vm_test.zig     unit tests (zig build test)
├── lib/                    Lunex standard library (embedded)
├── tests/                  integration tests
├── examples/               example programs
└── build.sh                one-command build script
```
