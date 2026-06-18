# Lunex Module System

Lunex has a unified module system that supports standard library modules,
local file imports, and external packages from the internet.

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

Standard library modules are built into the Lunex binary. No installation
needed. No `config.lx` entry needed.

---

## Local File Imports (`.nax` archives)

Use `@fimport` to import a compiled Lunex archive (`.nax` file):

```lx
val module = @fimport("./mylib.nax")
val utils  = @fimport("../shared/utils.nax")
```

### How `.nax` files work

A `.nax` file is a compiled Lunex archive produced by `lunex build`. It bundles
one or more `.lx` source files along with their compiled bytecode into a single
binary archive.

```
my-project/
  main.lx
  math.lx        ← contains fn divide(a, b) { a / b }
  build → math.nax
```

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

Then import and use it from another file:

```lx
val math = @fimport("./math.nax")

fn main() {
  val io = @import("std.io")
  io.log(math.divide(50, 2))   // → 25
  io.log(math.add(10, 5))      // → 15
}
```

This works 100% — functions exported from `.nax` files are callable with
normal dot-notation after import.

---

## External Modules

External modules are packages installed from the internet. They are cached
locally so they work offline after the first install.

> **Rule**: External module names must not clash with standard library names
> (`io`, `fs`, `http`, `crypto`, `db`, `ws`, `jwt`, `math`, `datetime`,
> `os`, `regex`, `env`, `utils`).

### Installing packages

#### From GitHub

```bash
lunex install github.com/user/repo
lunex install https://github.com/user/repo
lunex install github.com/user/repo@v1.2.3   # specific tag/branch
```

#### From any URL

```bash
lunex install https://example.com/mypackage
```

Non-GitHub URLs are downloaded in a **sandboxed environment** with stronger
security verification before being cached. The sandbox restricts filesystem and
network access during package evaluation.

### Importing installed packages

After installation, import the package by name:

```lx
val mylib = @import("mylib")
mylib.doSomething()
```

Or import directly from a URL (installs on first use):

```lx
val mylib = @import("github.com/user/repo")
val mylib = @import("https://example.com/mylib")
```

**GitHub URLs** — fetched via the GitHub Contents API, verified, then cached.
**Other URLs** — fetched in a sandboxed environment with stricter verification.

### Offline use

Once installed, packages are available without an internet connection:

```bash
lunex install github.com/user/mathlib      # install once
# later, offline:
lunex run main.lx                           # @import("mathlib") works from cache
```

### Listing installed packages

```bash
lunex list
```

Output:
```
package                        version
──────────────────────────────────────
mathlib                        1.0.0
github.com/user/util           main
```

### Removing packages

```bash
lunex remove mathlib
```

---

## `config.lx` — Project Configuration

`config.lx` is the project manifest file. It is used **only** for declaring
external module dependencies. Lunex configures built-in modules automatically
— you never need to list standard library modules here.

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
    "mathlib": "1.0.0"
    "github.com/user/util": "main"
  }
}
```

### Installing all dependencies

```bash
lunex install
```

This reads `config.lx` from the current directory and installs all listed
dependencies. After this, `@import("mathlib")` and `@import("util")` work
without internet access.

---

## Module Resolution Order

When you write `@import("name")`, Lunex resolves it in this order:

1. **Standard library** — `std.io`, `std.http`, etc. (built-in, always wins)
2. **Local cache** — packages installed via `lunex install`
3. **URL install** — if `name` looks like a URL, download and cache it

When you write `@fimport("path")`, Lunex resolves it as:

1. **Relative path** — `.nax` file relative to the current source file

---

## Security

### GitHub packages

Packages from `github.com` are fetched using the GitHub Contents API:
- Package files are downloaded individually over HTTPS
- GitHub rate limits apply (unauthenticated: 60 requests/hour)
- No code is executed during installation

### Non-GitHub URLs

Packages from other URLs run through a stricter verification process:
- Downloaded over HTTPS only
- Content is scanned for dangerous patterns before caching
- Execution is sandboxed during first evaluation

### Best practice

Always review package source code before installing from untrusted sources.
Use `lunex list` to audit installed packages and `lunex remove` to uninstall
anything you no longer need.

---

## Example: Complete Module Workflow

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
  io.log(mathlib.add(10, 5))      // → 15
  io.log(mathlib.mul(3, 7))       // → 21
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

### Using external modules

Install an external module:

```bash
lunex install github.com/Megamexlevi2/lunex-lang-gz/lune-xml
```

Import it in your code:

```lx
val xml = @import("lune-xml")
```

---

## `.nax` File Format

A `.nax` file uses a custom binary format. It is **not** a zip or tar archive.

- **Magic header**: `LXNAX ` (8 bytes) — intentionally unrecognisable to
  standard archive tools. Attempting to open it with zip/tar will fail by design.
- Only the Lunex runtime knows how to read `.nax` files.
- Running a `.nax` directly: `lunex run mylib.nax`

The format stores:
- One or more compiled bytecode chunks (one per source file)
- Source text embedded for debugging
- Module export table (function names accessible via `module.fn()`)
