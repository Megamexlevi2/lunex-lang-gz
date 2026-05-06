<div align="center">

<h1>NTL — Native Typed Language</h1>

<p>A fast, expressive scripting language with bytecode compilation, a TUI editor, and a rich standard library.<br>This is the first official release.</p>

</div>

---

## Installation

NTL is not open source. To use it, download the installer for your platform.

<div align="center">

**Linux / macOS / Termux**

```bash
bash install.sh
```

NTL will not be open source for now, only in the future. 

or via curl:

```bash
curl -fsSL https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.sh | bash
```

**Windows**

```powershell
iwr https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.ps1 -useb | iex
```

or download manually: [install.ps1](https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.ps1) · [install.sh](https://github.com/Megamexlevi2/ntl-go/releases/latest/download/install.sh)

</div>

---

## Quick Start

```bash
ntl hello.ntl               # run a source file
ntl build hello.ntl         # compile to bytecode (.nc)
ntl pack ./dist -o app.nax  # pack into a distributable archive
ntl run app.nax             # run it
ntl repl                    # open the interactive REPL
naxer hello.ntl             # open the terminal editor
```

---

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

Full language reference: [`docs/language.md`](docs/language.md)

---

## CLI Reference

| Command | Description |
|---|---|
| `ntl <file.ntl>` | Run an NTL source file |
| `ntl run <file>` | Run `.ntl`, `.nc`, or `.nax` |
| `ntl repl` | Interactive REPL |
| `ntl build <file.ntl>` | Compile to `.nc` bytecode |
| `ntl pack <dir>` | Pack `.nc` files into a `.nax` archive |
| `ntl fmt <file.ntl>` | Format source in-place |
| `ntl check <file.ntl>` | Check for errors without running |
| `ntl dis <file.nc>` | Show module info |
| `ntl init [name]` | Initialize a project (`ntl.mod`) |
| `ntl add <pkg>` | Install a package |
| `ntl list` | List installed packages |
| `ntl version` | Show version |

---

## File Formats

| Extension | Description |
|---|---|
| `.ntl` | NTL source code |
| `.nc` | Compiled bytecode (binary, scrambled) |
| `.nax` | Bundled bytecode archive |
| `ntl.mod` | Project manifest |

`.nc` and `.nax` files are binary — source code cannot be recovered from them.

---

## Build Pipeline

```bash
ntl build src/main.ntl -o dist/main.nc
ntl build src/utils.ntl -o dist/utils.nc

ntl pack dist -o myapp.nax

ntl run myapp.nax
```

---

## naxer — Terminal Editor

A vim/nano-style TUI editor with NTL syntax highlighting and autocomplete.

```bash
./ntl edit hello.ntl
```

| Mode | Keys |
|---|---|
| Normal | `i` insert · `d` delete line · `w`/`b` word jump · `:w` save · `:q` quit · `Ctrl+O` file browser |
| Insert | `Tab` autocomplete · `Esc` back to normal |
| Command | `:wq` save and quit · `:e` open file · `:w` save as |

---

## Standard Library

| Module | What it does |
|---|---|
| `io` | log, print, table output, colors |
| `fs` | read, write, mkdir, stat |
| `http` | get, post, serve, router |
| `crypto` | hash, hmac, jwt, aes |
| `db` | in-memory DB with schema |
| `env` | getenv, loadenv |
| `events` | EventEmitter |
| `cache` | TTL cache |
| `logger` | structured logging |
| `queue` | task queues |
| `validate` | schema validation |
| `ws` | WebSocket |
| `mail` | SMTP email |
| `ai` | LLM/AI client |
| `utils` | array, string, math helpers |
| `test` | unit testing |

Full module docs in the [`docs/`](docs/) folder:
[`ai`](docs/ai.md) · [`commands`](docs/commands.md) · [`crypto`](docs/crypto.md) · [`db`](docs/db.md) · [`env`](docs/env.md) · [`fs`](docs/fs.md) · [`http`](docs/http.md) · [`io`](docs/io.md) · [`language`](docs/language.md) · [`mail`](docs/mail.md) · [`test`](docs/test.md) · [`utils`](docs/utils.md) · [`validate`](docs/validate.md) · [`ws`](docs/ws.md)

## License

 — see [LICENSE](LICENSE) for details.

Copyright 2026 David Dev (Megamexlevi2). All rights reserved.