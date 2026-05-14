<div align="center">
<h1>NTL — Native Typed Language</h1>
<p>A fast programming language with an ultra-aggressive JIT,
a built-in TUI editor, and a rich standard library.

<code>ntl build file.ntl</code> produces fast .nc bytecode.</p>
</div>
## Installation
NTL is not open source. To use it, download the installer for your platform.
<div align="center">
**Linux / macOS / Termux**
```bash
bash install.sh

```
or via curl:
```bash
curl -fsSL [https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.sh](https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.sh) | bash

```
**Windows**
```powershell
iwr [https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.ps1](https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.ps1) -useb | iex

```
or download manually: install.ps1 · install.sh
</div>
## Quick Start
```bash
ntl build hello.ntl                    # generate hello.nc bytecode
ntl run hello.nc                       # run .nc bytecode (fast VM + JIT)
ntl hello.ntl                          # shorthand run
ntl pack ./dist -o app.nax             # pack bytecode into a distributable archive
ntl run app.nax                        # run archive
ntl edit hello.ntl                     # open the built-in terminal editor

```
## Language
```ntl
use io

fn greet(name) {
  return "Hello, " + name + "!"
}

class Counter {
  constructor(start) {
    this.value = start
  }

  inc() { this.value += 1 }
  get() { return this.value }
}

val c = new Counter(0)
repeat 5 { c.inc() }
io.log(greet("world"), c.get())

match c.get() {
  case 5  => io.log("five!")
  default => io.log("other")
}

```
Full language reference: docs/language.md
## CLI Reference
| Command | Description |
|---|---|
| ntl run <file> | Run .ntl source, .nc bytecode, or .nax archive |
| ntl <file.ntl> | Shorthand — run an NTL source file |
| ntl -e "<code>" | Execute a code string inline |
| ntl build <file.ntl> | Generate .nc bytecode |
| ntl build | Run build.ntl project build script |
| ntl pack <dir> [-o out.nax] | Pack .nc files into a .nax archive |
| ntl fmt <file.ntl> | Format source in-place |
| ntl check <file.ntl> | Type-check and lint without running |
| ntl dis <file.nc> | Disassemble bytecode module |
| ntl edit [file] | Open the built-in terminal editor |
| ntl init [name] | Initialize a project (ntl.mod) |
| ntl add <pkg>[@ver] | Install a package |
| ntl remove <pkg> | Remove a package |
| ntl list | List installed packages |
| ntl cache clear | Clear the bytecode cache |
| ntl version | Show version |
## File Formats
| Extension | Description |
|---|---|
| .ntl | NTL source code |
| .nc | NTL bytecode (fast VM + JIT) |
| .nax | Bundled bytecode archive |
| ntl.mod | Project manifest |
## Execution Pipeline
### Bytecode (ntl build file.ntl)
```
test.ntl
   ↓
Lexer / Parser
   ↓
AST
   ↓
Bytecode (.nc)   ← output of 'ntl build file.ntl'
   ↓
ntl run test.nc  (fast VM + JIT)

```
### Distributable Archive
```bash
ntl build src/main.ntl -o dist/main.nc
ntl build src/utils.ntl -o dist/utils.nc
ntl pack dist -o myapp.nax
ntl run myapp.nax

```
## Performance
**Bytecode** (.nc): The JIT hot threshold is 0 — every function is promoted to machine code on its very first call. No warm-up. Significantly faster than Node.js V8 for CPU-bound workloads.
The pipeline runs ENFS (Extreme Native Fast System) before execution: constant folding, dead code elimination, CSE, block merging, strength reduction, tail call conversion, GVN, and function inlining.
## naxer — Terminal Editor
A vim/nano-style TUI editor with NTL syntax highlighting and autocomplete.
```bash
ntl edit hello.ntl

```
| Mode | Keys |
|---|---|
| Normal | i insert · d delete line · w/b word jump · :w save · :q quit · Ctrl+O file browser |
| Insert | Tab autocomplete · Esc back to normal |
| Command | :wq save and quit · :e open file · :w save as |
## Standard Library
| Module | What it does |
|---|---|
| io | log, print, table output, colors |
| fs | read, write, mkdir, stat |
| http | get, post, serve, router |
| crypto | hash, hmac, jwt, aes |
| db | in-memory DB with schema |
| env | getenv, loadenv |
| events | EventEmitter |
| cache | TTL cache |
| logger | structured logging |
| queue | task queues |
| validate | schema validation |
| ws | WebSocket |
| mail | SMTP email |
| ai | LLM/AI client |
| alloc | low-level memory buffers and binary I/O |
| excel | Excel (.xlsx) read/write |
| formats | CSV, YAML, TOML, Markdown, Mustache |
| graphql | GraphQL schema builder and executor |
| jwt | JSON Web Token sign/verify/refresh |
| mysql | MySQL/MariaDB client |
| oauth2 | OAuth 2.0 (Google, GitHub, custom) |
| pdf | PDF document generation |
| rabbitmq | RabbitMQ / AMQP message queues |
| redis | Redis client |
| stripe | Stripe payments |
| utils | array, string, math helpers |
| test | unit testing |
Full module docs in the docs/ folder:
ai · alloc · commands · crypto · db · env · excel · formats · fs · graphql · http · io · jwt · language · mail · mysql · oauth2 · os · pdf · rabbitmq · redis · stripe · test · utils · validate · ws · xml
## License
— see LICENSE for details.
Copyright 2026 David Dev (Megamexlevi2). All rights reserved.
