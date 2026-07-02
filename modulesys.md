# Lunex Module System

Lunex has a unified module system that supports standard library modules,
local file imports, and external packages installed by the Go package manager.

---

## Standard Library Modules

The standard library is always available. Import any module with:

```lx
val io     = @import("std.io")
val http   = @import("std.http")
val fs     = @import("std.fs")
val crypto = @import("std.crypto")
val math   = @import("std.math")
val db     = @import("std.db")
val os     = @import("std.os")
val regex  = @import("std.regex")
val utils  = @import("std.utils")
val dt     = @import("std.datetime")
val env    = @import("std.env")
val ws     = @import("std.ws")
val jwt    = @import("std.jwt")
```

Standard library modules are embedded in the Lunex binary. No installation
needed, no `config.lx` entry needed.

---

## Local File Imports (`.lx` and `.nax`)

Use `@fimport` to import a local Lunex source file or compiled archive:

```lx
val utils  = @fimport("./src/utils.lx")
val mylib  = @fimport("./mylib.nax")
val shared = @fimport("../shared/utils.nax")
```

### How `.nax` files work

A `.nax` file is a compiled Lunex archive produced by `lunex build`. It bundles
one or more `.lx` source files along with their compiled bytecode into a single
binary archive.

If `math.lx` contains:

```lx
fn divide(a, b) {
  a / b
}

fn add(a, b) {
  a + b
}
```

Compile it:

```bash
lunex build math.lx -o math.nax
```

Then import and use it:

```lx
val io   = @import("std.io")
val math = @fimport("./math.nax")

fn main() {
  io.log(math.divide(50, 2))   // → 25
  io.log(math.add(10, 5))      // → 15
}
```

---

## External Packages (managed by Lunex)

External packages are installed and managed entirely by the Go-based package manager built into `lunex`. The runtime resolves imported packages from the local cache.

> **Rule**: External package names must not clash with standard library names
> (`io`, `fs`, `http`, `crypto`, `db`, `ws`, `jwt`, `math`, `datetime`,
> `os`, `regex`, `env`, `utils`).

### Installing packages

```bash
lunex install github.com/user/repo         # from GitHub (main branch)
lunex install github.com/user/repo@v1.2.3  # specific tag or branch
lunex install github.com/user/repo/subdir  # install a subdirectory package
```

### Importing installed packages

After installation, import the package by name:

```lx
val xml = @import("lune-xml")
xml.parse("<root/>")
```

The Lunex runtime resolves `@import("pkg-name")` by looking in `~/.lunex/cache/`.

### Offline use

Once installed, packages work without an internet connection:

```bash
lunex install github.com/Megamexlevi2/lune-xml      # install once
# later, offline:
lunex run main.lx                                    # @import("lune-xml") works from cache
```

### Managing packages

```bash
lunex list              # list installed packages
lunex remove lune-xml   # remove a package
lunex update            # update all packages
lunex update lune-xml   # update one package
```

---

## `config.lx` — Project Configuration

`config.lx` is the project manifest file. It declares external module
dependencies and project metadata. Standard library modules are automatic —
you never need to list them here.

```lx
val project = {
  name: "my-app"
  version: "1.0.0"
  description: "My Lunex application"
  author: "Your Name"
  main: "main.lx"
  entry: "main.lx"
  output: "dist"
  optimize: true
  targets: []

  // Only external modules go here — std modules are automatic
  dependencies: {
    "lune-xml": "main"
    "Megamexlevi2/util": "v1.0.0"
  }
}
```

### Installing all dependencies

```bash
lunex install
```

Reads `config.lx` from the current directory and installs all listed
dependencies. After this, `@import("lune-xml")` works without internet.

---

## Module Resolution Order

When you write `@import("name")`, Lunex resolves it in this order:

1. **Standard library** — `std.io`, `std.http`, etc. (built-in, always wins)
2. **Lunex package cache** — packages installed via `lunex install`
   (checked in `~/.lunex/cache/`)

When you write `@fimport("path")`, Lunex resolves it as:

1. **Relative path** — `.lx` source file or `.nax` archive relative to the
   current source file

### Package not found?

If `@import("pkg-name")` fails to resolve, Lunex will hint:

```
hint: package "pkg-name" not found — install it with:
  lunex install pkg-name
```

---

## `.nax` File Format

A `.nax` file uses a custom binary format — it is **not** a zip or tar archive.

- **Magic header**: `LXNAX ` (8 bytes) — intentionally unrecognisable to
  standard archive tools. Attempting to open it with zip/tar will fail by design.
- Only the Lunex runtime knows how to read `.nax` files.
- Running a `.nax` directly: `lunex run mylib.nax`

The format stores:
- One or more compiled bytecode chunks (one per source file)
- Source text embedded for debugging
- Module export table (function names accessible via `module.fn()`)

---

## Example: Complete Module Workflow

**Step 1**: Install a package with Lunex

```bash
lunex install github.com/Megamexlevi2/lune-xml
```

**Step 2**: Import and use it

```lx
val io  = @import("std.io")
val xml = @import("lune-xml")

fn main() {
  val doc = xml.parse("<greet>Hello, Lunex!</greet>")
  io.log(doc.root.text)   // → Hello, Lunex!
}
```

**Step 3**: Run

```bash
lunex run main.lx
```

---

## Local Library Workflow

**Step 1**: Write a library (`mathlib.lx`)

```lx
fn add(a, b) { a + b }
fn sub(a, b) { a - b }
fn mul(a, b) { a * b }
fn div(a, b) { a / b }
fn clamp(x, lo, hi) {
  if x < lo { lo }
  else if x > hi { hi }
  else { x }
}
```

**Step 2**: Build it as a `.nax` archive

```bash
lunex build mathlib.lx -o mathlib.nax
```

**Step 3**: Import and use it

```lx
val io      = @import("std.io")
val mathlib = @fimport("./mathlib.nax")

fn main() {
  io.log(mathlib.add(10, 5))          // → 15
  io.log(mathlib.mul(3, 7))           // → 21
  io.log(mathlib.clamp(150, 0, 100))  // → 100
}
```

**Step 4**: Run

```bash
lunex run main.lx
```

Output:
```
15
21
100
```
