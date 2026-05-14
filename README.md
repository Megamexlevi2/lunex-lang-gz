<div align="center">

# NTL — Native Typed Language

A modern typed programming language focused on performance, simplicity, and developer experience.

NTL compiles source code into portable `.nc` bytecode and includes a built-in terminal editor, package manager, formatter, and standard library.

```bash
ntl build file.ntl
```

</div>

---

# Installation

NTL is currently distributed through official installers.

<div align="center">

## Linux / macOS / Termux

```bash
bash install.sh
```

Or install directly using curl:

```bash
curl -fsSL https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.sh | bash
```

## Windows

```powershell
iwr https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.ps1 -useb | iex
```

Manual installers:

```text
install.ps1
install.sh
```

</div>

---

# Quick Start

```bash
ntl build hello.ntl
ntl run hello.nc

ntl hello.ntl

ntl pack ./dist -o app.nax
ntl run app.nax

ntl edit hello.ntl
```

---

# Example

```ntl
use io

fn greet(name) {
  return "Hello, " + name + "!"
}

class Counter {
  constructor(start) {
    this.value = start
  }

  inc() {
    this.value += 1
  }

  get() {
    return this.value
  }
}

val c = new Counter(0)

repeat 5 {
  c.inc()
}

io.log(greet("world"), c.get())

match c.get() {
  case 5 => io.log("five!")
  default => io.log("other")
}
```

Full language reference:

```text
docs/language.md
```

---

# CLI Reference

| Command | Description |
|---|---|
| `ntl run <file>` | Run `.ntl`, `.nc`, or `.nax` files |
| `ntl <file.ntl>` | Shortcut for running a source file |
| `ntl -e "<code>"` | Execute inline code |
| `ntl build <file.ntl>` | Compile source into `.nc` bytecode |
| `ntl build` | Execute the `build.ntl` project script |
| `ntl pack <dir> [-o out.nax]` | Create a distributable archive |
| `ntl fmt <file.ntl>` | Format source code |
| `ntl check <file.ntl>` | Run type-checking and linting |
| `ntl dis <file.nc>` | Disassemble bytecode |
| `ntl edit [file]` | Open the terminal editor |
| `ntl init [name]` | Create a new project |
| `ntl add <pkg>[@ver]` | Install a package |
| `ntl remove <pkg>` | Remove a package |
| `ntl list` | List installed packages |
| `ntl cache clear` | Clear bytecode cache |
| `ntl version` | Show current version |

---

# File Formats

| Extension | Description |
|---|---|
| `.ntl` | NTL source code |
| `.nc` | Compiled bytecode |
| `.nax` | Packaged application archive |
| `ntl.mod` | Project manifest |

---

# Execution Pipeline

```text
test.ntl
   ↓
Lexer / Parser
   ↓
AST
   ↓
Bytecode (.nc)
   ↓
ntl run test.nc
```

---

# Packaging Applications

```bash
ntl build src/main.ntl -o dist/main.nc
ntl build src/utils.ntl -o dist/utils.nc

ntl pack dist -o myapp.nax

ntl run myapp.nax
```

---

# Compiler Optimizations

Before execution, the compiler applies several optimization passes through ENFS (Extreme Native Fast System):

- Constant folding
- Dead code elimination
- Function inlining
- Tail call conversion
- Strength reduction
- Block merging
- Common subexpression elimination
- Global value numbering

---

# naxer — Terminal Editor

`naxer` is the built-in terminal editor for NTL.

It includes:

- Syntax highlighting
- Autocomplete
- Vim/nano-style controls
- Fast terminal-based editing

Open the editor with:

```bash
ntl edit hello.ntl
```

## Editor Modes

| Mode | Keys |
|---|---|
| Normal | `i` insert · `d` delete line · `w/b` move words · `:w` save · `:q` quit |
| Insert | `Tab` autocomplete · `Esc` return to normal mode |
| Command | `:wq` save and quit · `:e` open file |

---

# Standard Library

| Module | Description |
|---|---|
| `io` | Logging, printing, colors, tables |
| `fs` | File system utilities |
| `http` | HTTP client and server |
| `crypto` | Hashing, AES, HMAC, JWT |
| `db` | In-memory database |
| `env` | Environment variables |
| `events` | EventEmitter implementation |
| `cache` | TTL cache |
| `logger` | Structured logging |
| `queue` | Task queues |
| `validate` | Schema validation |
| `ws` | WebSocket support |
| `mail` | SMTP email |
| `ai` | AI / LLM client |
| `alloc` | Binary and memory utilities |
| `excel` | Excel `.xlsx` support |
| `formats` | CSV, YAML, TOML, Markdown |
| `graphql` | GraphQL tools |
| `jwt` | Token signing and verification |
| `mysql` | MySQL and MariaDB client |
| `oauth2` | OAuth 2.0 authentication |
| `pdf` | PDF generation |
| `rabbitmq` | RabbitMQ / AMQP |
| `redis` | Redis client |
| `stripe` | Stripe integration |
| `utils` | Utility helpers |
| `test` | Unit testing |

---

# Documentation

Module documentation is available inside the `docs/` folder.

```text
ai
alloc
commands
crypto
db
env
excel
formats
fs
graphql
http
io
jwt
language
mail
mysql
oauth2
os
pdf
rabbitmq
redis
stripe
test
utils
validate
ws
xml
```

---

# License

See `LICENSE` for details.

Copyright © 2026 David Dev (Megamexlevi2).
All rights reserved.