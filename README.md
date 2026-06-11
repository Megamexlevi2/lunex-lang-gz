# Lunex

A programming language focused on being simple, fast, and useful.

Lunex runs on **Linux**, **macOS**, **Windows**, and **Android (Termux)**.

## Installation

Download the latest version from the release. 

## Quick Start

```bash
cat << 'PROG' > main.lx
val io = @import("std.io")

fn main() {
  io.log("Hello, World!")

  val name = io.read("Your name: ")
  io.log("Hello, " + name + "!")
}
PROG

lunex run main.lx
```

## CLI

```
lunex run <file> [--emit ast|ir]   run a .lx, .nc, or .nax file
lunex -e "<code>"                  run a snippet directly
lunex build [file] [-o]            compile to .nc bytecode
lunex check <file>                 check for errors without running
lunex fmt <file>                   format source code
lunex dis <file.nc>                disassemble bytecode
lunex init [name]                  create a new project
lunex install <url>                install a package (GitHub or any URL)
lunex add / remove                 manage packages
lunex list                         show installed packages
lunex bench <file>                 run with timing output
lunex version                      print version
lunex start                        run project entry from config.lx
```

---

## Language

### Variables

`val` declares an immutable binding. `var` declares a mutable one.

```lx
val name   = "Lunex"
val age    = 30
val height = 1.75
val active = true

var counter = 0
counter = counter + 1
```

Destructuring works on objects and arrays:

```lx
val { name, role } = user
val [first, second] = items
```

### Functions

```lx
fn add(a, b) {
  a + b
}

val square = fn(x) { x * x }

io.log(add(2, 3))
io.log(square(5))
```

The last expression in a function is its return value. You can also use `return` explicitly.

### Template strings

```lx
val msg = "Hello, " + name + "! You are " + str(age) + " years old."
```

### Structs

Lunex has no `class` keyword. Use factory functions that return a `struct`:

```lx
fn Animal(name, sound) {
  val self = struct {
    name  = name
    sound = sound

    fn speak() {
      this.name + " says " + this.sound
    }
  }
  self
}

val cat = Animal("Cat", "Meow")
io.log(cat.speak())
```

Inside a `struct`, plain assignments like `name = name` create fields on the struct itself, and methods can use `this` or `self` safely.

### Control flow

```lx
if n < 0 {
  "negative"
} else if n == 0 {
  "zero"
} else {
  "positive"
}
```

`guard` runs its block when the condition is false, useful for early exits:

```lx
guard user.loggedIn else {
  io.error("not authenticated")
}
```

`unless` is the opposite of `if`:

```lx
unless ready {
  io.log("not ready")
}
```

### Loops

```lx
var i = 0
while i < 10 {
  io.log(i)
  i = i + 1
}

each name in ["Alice", "Bob", "Charlie"] {
  io.log("Hello, " + name + "!")
}
```

### Pattern matching

```lx
match x {
  0      => "zero"
  1      => "one"
  2..10  => "small"
  _      => "other"
}
```

### Concurrency

```lx
val ch = channel()

spawn fn() {
  ch.send(fetchData())
}()

val result = ch.recv()
io.log(result)
```

### Defer

Schedules code to run when the current function exits:

```lx
fn readFile(path) {
  val fs = @import("std.fs")
  val f = fs.open(path)
  defer { f.close() }
  f.read()
}
```

---

## Module System

Lunex has a powerful module system. There are three kinds of imports:

### 1. Standard library

```lx
val io   = @import("std.io")
val http = @import("std.http")
val fs   = @import("std.fs")
```

### 2. Local `.nax` file

```lx
val module = @fimport("./mylib.nax")
fn main() {
module.doSomething()
}
```

A `.nax` file is a compiled Lunex archive (produced by `lunex build`).
If `mylib.nax` exports a function `dividir(a, b)` you can call it as:

```lx
val math = @fimport("./math.nax")
val io = @import("std.io")
fn main(){
io.log(math.dividir(50, 2))    // → 25
}
```

### 3. External module (installed via `lunex install`)

```lx
val module = @import("modulename")
```

The module name **must not** clash with any standard library name.

You can also import directly from a URL:

```lx
val module = @import("github.com/user/repo")
val module = @import("https://example.com/mylib")
```

- **GitHub URLs** (`github.com/...`) are fetched via the GitHub API and cached locally.
- **Other URLs** are fetched in a **sandboxed environment** with stronger security checks before being cached.

After the first import the module is cached on disk — subsequent runs work without internet access.

### Installing packages

```bash
# From GitHub
lunex install github.com/user/repo
lunex install https://github.com/user/repo

# From any URL (sandboxed verification)
lunex install https://example.com/mypackage
```

Installed packages are saved in the Lunex cache directory. Once installed you
can use them with `@import("packagename")` without internet.

### `config.lx`

`config.lx` is your project's build manifest. It is **only** used for managing
external module dependencies — the Lunex runtime configures built-in modules
automatically; you never need to declare them here.

```lx
val project = {
  name: "my-app"
  version: "1.0.0"
  main: "main.lx"
  entry: "main.lx"
  output: "dist"
  optimize: true
  targets: []
  dependencies: {
    "mathlib": "1.0.0"
    "github.com/user/example": "main"
  }
}
```

Run `lunex install` (no arguments) inside the project directory to install all
dependencies listed in `config.lx`.

---

## Compatibility note

The examples and module names in this README are written to match the current Lunex runtime. If a module is shown here, it should be importable with the same name in the current build, such as `@import("std.io")`, `@import("std.fs")`, `@import("std.http")`, and `@import("runtime")`.

The return types and short descriptions below are the ones exposed by the current implementation, so you can inspect a module with `io.log(io)` and code against it without guessing.

## Standard Library

Lunex modules are objects. You can store them in variables, print them, and call their members with dot notation.

### Module inspection

```lx
val io = @import("std.io")
fn main() {
io.log(io)
}
```

That pattern is useful when you want to see what a module exposes before coding against it.  

A quick rule of thumb for returns:

- printing helpers like `log`, `warn`, `err`, `info`, `success`, `table`, `clear`, `spinner`, `tick`, and `stop` return `undefined`
- input helpers like `read` and `readLine` return `string`
- numeric input like `readInt` returns `number`
- runtime helpers usually return the value they read, a boolean, or `undefined` depending on the operation

- `readInt` retorna `number`

### Runtime introspection

```lx
val runtime = @import("runtime")
val io = @import("std.io")
fn main() {
val originalLog = runtime.getGlobal("io.log")
io.log(runtime.hasGlobal("io.log"))
io.log(runtime.globals())
}
```

`runtime.getGlobal("name")` reads a global value, `runtime.setGlobal("name", value)` writes one, `runtime.hasGlobal("name")` checks if it exists, `runtime.globals()` lists visible names, and `runtime.version()` returns the runtime version string.

There is no separate `runtime.keywords` module in the current build. For discovery, use `runtime.globals()` or print the module object itself.

### Common modules

| Module | Purpose | Typical return |
|--------|---------|----------------|
| `io` | Console I/O | `undefined`, `string`, or `number` depending on the function |
| `fs` | Files and folders | `string`, `array`, `object`, or `undefined` |
| `http` | HTTP client/server | response/server objects |
| `crypto` | Hashing, encryption, JWT | `string` or `bool` |
| `db` | In-memory database | query/result objects |
| `math` | Math functions and constants | `number` and constants |
| `datetime` | Dates, timestamps, sleep | `number` or `string` |
| `os` | Process and platform helpers | `string`, `number`, or `undefined` |
| `regex` | Pattern matching and replacement | `bool`, `string`, or `array` |
| `env` | Environment variables | `string` and `object` |
| `utils` | General helpers | mixed |
| `ws` | WebSocket client/server | connection objects |
| `jwt` | JWT sign and verify | `string` or `bool` |
| `runtime` | Global inspection and patching | `value`, `bool`, `array`, `string`, or `undefined` |

### Example style

```lx
val io = @import("std.io")
val fs = @import("std.fs")
val math = @import("std.math")
fn main() {
io.log("hello")
val text = fs.read("data.txt")
io.log(math.PI)
}
```

For local compiled archives, use `@fimport`:

```lx
val lib = @fimport("./mylib.nax")
lib.doSomething()
```

For installed packages, use `@import("name")`:

```lx
val pkg = @import("my-package")
fn main() {
pkg.run()
}
```

That is enough for most day-to-day code without needing a full reference manual.


### license

See the LICENSE file to view the current license