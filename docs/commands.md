# Lunex Command Reference

Lunex — Native Typed Language. Compiled pipeline uses embedded TCC as the sole native backend.

## Running Code

```
lunex <file.lx>             Run an Lunex source file (TCC native if available, else interpreted)
lunex run <file>             Run .lx, .nc, or .nax file
```

## Building Bytecode (default)

`lunex build` without a `--target` flag compiles to `.nc` bytecode. Bytecode files run via the
fast VM + JIT pipeline and are portable across platforms.

```
lunex build app.lx                       Compile to app.nc (default)
lunex build app.lx -o output.nc          Explicit output path
lunex build                               Read build.lx config and build all targets
```

## Building Native Binaries (`--target`)

Passing `--target` compiles to a native binary via the embedded TCC backend (host) or a
system cross-compiler (non-host targets).

```
lunex build app.lx --target linux/amd64          Host binary (uses embedded TCC)
lunex build app.lx --target linux/arm64          Cross-compile (requires gcc-aarch64-linux-gnu or clang)
lunex build app.lx --target windows/amd64 -o app.exe
lunex build app.lx --target android/arm64        (requires Android NDK)
lunex build app.lx --target darwin/arm64         (requires clang on macOS or cross-clang)
```

### How the compilation pipeline works

```
.lx source
  -> Lexer -> Parser -> AST
  -> Bytecode (.nc)
  -> NTIR (Lunex Intermediate Representation)
  -> ENFS optimizer (up to 12-pass extreme IR optimization)
  -> C backend (Lunex C code generator)
  -> main.c
  -> Embedded TCC (Tiny C Compiler) — no external tools for host builds
  -> Native binary: ELF (Linux), PE (Windows), Mach-O (macOS)
```

TCC is the **only** native compilation backend. The custom amd64/arm64 machine code generator
has been removed. TCC handles all host-platform builds automatically with no setup. For
cross-compilation, install a system cross-compiler (see `lunex runtimes` for setup instructions).

## Building Bytecode Archives

```
lunex pack <directory>         Pack all .nc files into a .nax archive
lunex pack <directory> -o output.nax
```

## ENFS — Extreme Native Fast System

ENFS is Lunex's built-in IR optimizer. It runs automatically before every pipeline — both
compiled and interpreted. You never need to invoke it manually.

Optimization passes:

- **Constant folding** — arithmetic on literals is resolved at compile time
- **Dead code elimination** — unreachable instructions and unused values are removed
- **Common subexpression elimination** — repeated computations are computed once
- **Block merging** — chains of single-successor blocks are collapsed
- **Redundant load elimination** — loads after stores to the same variable are removed
- **Strength reduction** — multiply-by-power-of-two becomes a shift
- **Tail call conversion** — recursive tail calls are marked for optimization
- **Phi elimination** — single-source phi nodes are folded
- **Global value numbering** — equivalent values across a function are unified
- **Function inlining** — small functions are inlined at call sites
- **Dead function elimination** — unreachable functions are removed before codegen

## Interpreted / Bytecode Execution

When running `.nc` files or falling back from TCC, Lunex uses its fast interpreted pipeline:

```
.nc bytecode
  -> NTIR instruction dispatch (no tree-walking, no line-by-line execution)
  -> ENFS optimizer
  -> Ultra-aggressive JIT (threshold=0, immediate native promotion)
  -> Native amd64/arm64 machine code
```

The JIT hot threshold is 0: every function is promoted to native machine code on its very
first call. No warm-up. Significantly faster than Node.js V8 for CPU-bound workloads.

## Project Management

```
lunex init [name]              Initialize a new project (creates lunex.mod)
lunex install                  Install packages listed in lunex.mod
lunex install [pkg]            Install a specific package
lunex add <pkg>                Add and install a package (updates lunex.mod)
lunex remove <pkg>             Remove an installed package
lunex list                     List installed packages
```

## Code Quality

```
lunex check app.lx            Check for syntax/parse errors without running
lunex fmt app.lx              Format Lunex source code in-place
lunex dis app.nc               Disassemble bytecode module
lunex see_errors app.lx       Deep static analysis — list every error/warning
```

## Version

```
lunex version
lunex --version
lunex -v
```

## File Types

| Extension | Description                                            |
|-----------|--------------------------------------------------------|
| `.lx`    | Lunex source code (human readable)                       |
| `.nc`     | Lunex Compiled — bytecode (fast VM + JIT pipeline)       |
| `.nax`    | Lunex Archive — bundled bytecode                         |
| `lunex.mod` | Project manifest (package dependencies)                |

Native binaries produced by `lunex build --target` have no extension on Linux/macOS and `.exe`
on Windows. They run standalone with no Lunex runtime required.

## The naxer Editor

`naxer` (or `lunex edit`) is Lunex's built-in terminal editor with Lunex syntax highlighting.

```
lunex edit                     Open editor (new file)
lunex edit <file>              Open a file in the editor
```

### Key Bindings

**Normal mode:**

| Key          | Action                    |
|--------------|---------------------------|
| `i`          | Enter insert mode         |
| `a`          | Insert after cursor       |
| `o` / `O`   | New line below / above    |
| `d`          | Delete current line       |
| `x`          | Delete character          |
| `h j k l`   | Move left/down/up/right   |
| `0` / `$`   | Start / end of line       |
| `g` / `G`   | First / last line         |
| `w` / `b`   | Next / prev word          |
| `Ctrl+D/U`  | Half page down / up       |
| `Ctrl+S`    | Save file                 |
| `Ctrl+O`    | Open file browser         |
| `Ctrl+H`    | Toggle help               |
| `:`          | Enter command mode        |

**Insert mode:**

| Key          | Action                    |
|--------------|---------------------------|
| `Tab`        | Accept autocomplete       |
| `Up/Down`   | Navigate autocomplete     |
| `Esc`        | Return to Normal mode     |

**Commands (type `:` then command):**

| Command      | Action                    |
|--------------|---------------------------|
| `w`          | Save                      |
| `q`          | Quit                      |
| `wq`         | Save and quit             |
| `w filename` | Save as new file          |
| `e filename` | Open file                 |
| `new`        | New empty file            |
| `help`       | Show help                 |

## Stdlib Modules

Import with `val mod = @import("std.module")` at the top of your file. All modules work in both compiled and interpreted mode.

| Module     | Description                              |
|------------|------------------------------------------|
| `io`       | Console output with colors and tables    |
| `fs`       | File system read/write                   |
| `http`     | HTTP client and server                   |
| `crypto`   | Hashing, JWT, AES-GCM encryption         |
| `db`       | In-memory database with schema           |
| `env`      | Environment variables, .env loading      |
| `events`   | Event emitter                            |
| `cache`    | TTL cache with LRU eviction              |
| `logger`   | Structured logging                       |
| `queue`    | Task queues with priority scheduling     |
| `validate` | Schema validation and format checking    |
| `ws`       | WebSocket server and client              |
| `mail`     | SMTP email with HTML                     |
| `ai`       | AI/LLM client (OpenAI compatible)        |
| `utils`    | Array, string, math utilities            |
| `test`     | Unit testing framework                   |
| `type`     | Type system utilities                    |
