# NTL Command Reference

NTL — Native Typed Language. Compiled pipeline uses embedded TCC as the sole native backend.

## Running Code

```
ntl <file.ntl>             Run an NTL source file (TCC native if available, else interpreted)
ntl run <file>             Run .ntl, .nc, or .nax file
```

## Building Bytecode (default)

`ntl build` without a `--target` flag compiles to `.nc` bytecode. Bytecode files run via the
fast VM + JIT pipeline and are portable across platforms.

```
ntl build app.ntl                       Compile to app.nc (default)
ntl build app.ntl -o output.nc          Explicit output path
ntl build                               Read build.ntl config and build all targets
```

## Building Native Binaries (`--target`)

Passing `--target` compiles to a native binary via the embedded TCC backend (host) or a
system cross-compiler (non-host targets).

```
ntl build app.ntl --target linux/amd64          Host binary (uses embedded TCC)
ntl build app.ntl --target linux/arm64          Cross-compile (requires gcc-aarch64-linux-gnu or clang)
ntl build app.ntl --target windows/amd64 -o app.exe
ntl build app.ntl --target android/arm64        (requires Android NDK)
ntl build app.ntl --target darwin/arm64         (requires clang on macOS or cross-clang)
```

### How the compilation pipeline works

```
.ntl source
  -> Lexer -> Parser -> AST
  -> Bytecode (.nc)
  -> NTIR (NTL Intermediate Representation)
  -> ENFS optimizer (up to 12-pass extreme IR optimization)
  -> C backend (NTL C code generator)
  -> main.c
  -> Embedded TCC (Tiny C Compiler) — no external tools for host builds
  -> Native binary: ELF (Linux), PE (Windows), Mach-O (macOS)
```

TCC is the **only** native compilation backend. The custom amd64/arm64 machine code generator
has been removed. TCC handles all host-platform builds automatically with no setup. For
cross-compilation, install a system cross-compiler (see `ntl runtimes` for setup instructions).

## Building Bytecode Archives

```
ntl pack <directory>         Pack all .nc files into a .nax archive
ntl pack <directory> -o output.nax
```

## ENFS — Extreme Native Fast System

ENFS is NTL's built-in IR optimizer. It runs automatically before every pipeline — both
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

When running `.nc` files or falling back from TCC, NTL uses its fast interpreted pipeline:

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
ntl init [name]              Initialize a new project (creates ntl.mod)
ntl install                  Install packages listed in ntl.mod
ntl install [pkg]            Install a specific package
ntl add <pkg>                Add and install a package (updates ntl.mod)
ntl remove <pkg>             Remove an installed package
ntl list                     List installed packages
```

## Code Quality

```
ntl check app.ntl            Check for syntax/parse errors without running
ntl fmt app.ntl              Format NTL source code in-place
ntl dis app.nc               Disassemble bytecode module
ntl see_errors app.ntl       Deep static analysis — list every error/warning
```

## Version

```
ntl version
ntl --version
ntl -v
```

## File Types

| Extension | Description                                            |
|-----------|--------------------------------------------------------|
| `.ntl`    | NTL source code (human readable)                       |
| `.nc`     | NTL Compiled — bytecode (fast VM + JIT pipeline)       |
| `.nax`    | NTL Archive — bundled bytecode                         |
| `ntl.mod` | Project manifest (package dependencies)                |

Native binaries produced by `ntl build --target` have no extension on Linux/macOS and `.exe`
on Windows. They run standalone with no NTL runtime required.

## The naxer Editor

`naxer` (or `ntl edit`) is NTL's built-in terminal editor with NTL syntax highlighting.

```
ntl edit                     Open editor (new file)
ntl edit <file>              Open a file in the editor
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

Use with `use <module>` at the top of your file. All modules work in both compiled and
interpreted mode.

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
