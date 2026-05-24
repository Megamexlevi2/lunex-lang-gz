# Lunex lang

<p align="center">
  <img src="icon.svg" width="140">
</p>

**Created by David Dev · GitHub: https://github.com/Megamexlevi2 · (c) David Dev 2026**

# more comprehensive documentation 

<a href="https://megamexlevi2.github.io/lunex-lang-gz/docs/index.html" target="_blank" style="padding:12px 20px;background:#111;color:#fff;text-decoration:none;border-radius:10px;display:inline-block;font-weight:bold;">
  Open Documentation
</a>

# Installation

Lunex is currently distributed through official installers.

<div align="center">

## Linux / macOS / Termux

```bash
bash install.sh
```

Or install directly using curl:

```bash
curl -fsSL https://raw.githubusercontent.com/Megamexlevi2/lunex-lang-gz/main/install.sh | bash
```

## Windows

```powershell
iwr https://raw.githubusercontent.com/Megamexlevi2/lunex-lang-gz/main/install.ps1 -UseBasicParsing | iex
```

</div>

# Source Code

[Open Source Page](https://megamexlevi2.github.io/lunex-lang-gz/source/index.html)

---

## Usage

```bash
./lunex run file.lx              # run a source file
./lunex run file.nc               # run compiled bytecode
./lunex build file.lx            # compile to bytecode (.nc)
./lunex build file.lx -o file.nc # compile with explicit output path
```

## Language Notes

- **No `return`** — Lunex does not have a `return` keyword. The last expression in a function body is its result automatically. Using `return` is a parse error.
- **No `class`** — Lunex does not have a `class` keyword. Use constructor functions + `struct { ... }` for named types. Using `class` is a parse error.
- A `main()` function is the conventional entry point for programs.


---

## Hello World

```lunex
val io = @import("std.io")

fn main() {
  io.log("Hello, world!")
}
```

---

---

## example of http 

```lunex
val http = @import("std.http")
val io = @import("std.io")

fn main() {
  val server = http.createServer(fn(req, res) {
    http.json(res, {
      "message": "Hello from Lunex-lang"
    }, 200)
  })

  http.listen(server, 3000, "0.0.0.0", fn() {
    io.log("Server running on port 3000")
  })
}
```

---

## Variables

```lunex
val name  = "Alice"      // immutable
var count = 0            // mutable
count += 1
```

---

## Types

Lunex is dynamically typed. Runtime types: `string`, `number`, `boolean`, `array`, `object`, `function`, `null`.

```lunex
typeof "hello"   // "string"
typeof 42        // "number"
typeof true      // "boolean"
typeof [1, 2]    // "array"
typeof {}        // "object"
typeof null      // "null"
```

---

## Operators

```lunex
+  -  *  /  %  **        arithmetic
==  !=  <  >  <=  >=     comparison
===  !==                 strict equality
and  or  not             logical
?:                       ternary
??                       nullish coalescing
|>                       pipeline
```

---

## Control Flow

```lunex
if x > 0 {
  io.log("positive")
} else if x < 0 {
  io.log("negative")
} else {
  io.log("zero")
}

unless x == 0 {
  io.log("not zero")
}
```

---

## Loops

```lunex
// range
each i in range(10) {
  io.log(i)
}

each i in range(2, 20, 2) {
  io.log(i)
}

// iterate array
each item in items {
  io.log(item)
}

// while
var i = 0
while i < 10 {
  i += 1
}

// infinite loop
loop {
  if done { break }
}

// repeat N times
repeat 5 {
  io.log("hello")
}
```

---

## Functions

```lunex
fn add(a, b) {
  a + b
}

// variadic
fn sum(...nums) {
  var total = 0
  each n in nums { total += n }
  total
}

// first-class / closures
val double = fn(x) { x * 2 }

// pipeline
val result = [1, 2, 3]
  |> fn(arr) { arr.map(fn(x) { x * 2 }) }
  |> fn(arr) { arr.filter(fn(x) { x > 2 }) }
```

---

## Objects

```lunex
val person = {
  name: "Alice",
  age: 30,
  greet: fn() {
    "Hi, I am " + this.name
  }
}

io.log(person.name)
io.log(person["age"])
person.age = 31
```

---

## Arrays

```lunex
val arr = [1, 2, 3, 4, 5]

arr.push(6)
arr.pop()
arr.length

arr.map(fn(x) { x * 2 })
arr.filter(fn(x) { x > 2 })
arr.reduce(fn(acc, x) { acc + x }, 0)
arr.find(fn(x) { x > 3 })
arr.includes(3)
arr.slice(1, 3)
arr.join(", ")
arr.reverse()
arr.sort()
arr.forEach(fn(x) { io.log(x) })
```

---

## Structs (Replacing Classes)


```lunex
fn Animal(name, sound) {
  val self = struct {
    name  = name
    sound = sound
    fn speak() {
      self.name + " says " + self.sound
    }
  }
  self
}

fn Dog(name) {
  val base   = Animal(name, "woof")
  val tricks = []
  val self = struct {
    name   = base.name
    sound  = base.sound
    tricks = tricks
    fn speak()       { base.speak() }
    fn learn(trick)  { self.tricks.push(trick) }
    fn perform() {
      if self.tricks.length == 0 {
        self.name + " knows no tricks"
      } else {
        self.name + " can: " + self.tricks.join(", ")
      }
    }
  }
  self
}

val dog = Dog("Rex")
io.log(dog.speak())
dog.learn("sit")
io.log(dog.perform())
```

---

## Match

```lunex
fn describe(x) {
  match x {
    case null  => "nothing"
    case true  => "yes"
    case false => "no"
    case 0     => "zero"
    default    => "something else"
  }
}
```

---

## Error Handling

```lunex
try {
  val result = riskyOperation()
} catch err {
  io.log("error:", err)
} finally {
  io.log("always runs")
}

throw "something went wrong"

// safe call — returns null on error instead of throwing
val result = try? riskyOperation()
```

---

## Template Strings

```lunex
val msg = `Hello, ${name}! You are ${age} years old.`
```

---

## Destructuring

```lunex
val { name, age } = person
val [first, second, ...rest] = items
```

---

## Spread

```lunex
val merged   = { ...obj1, ...obj2 }
val combined = [...arr1, ...arr2]
```

---

## Optional Chaining & Nullish

```lunex
val value = maybeNull ?? "default"
val name  = user?.profile?.name ?? "anonymous"
val len   = arr?.length ?? 0
```

---

## Concurrency

```lunex
spawn myFunction()

val ch = channel()

spawn fn() {
  ch.send(42)
}()

val value = ch.recv()
```

---

## Modules

### Built-in stdlib

```lunex
val io       = @import("std.io")
val fs       = @import("std.fs")
val http     = @import("std.http")
val crypto   = @import("std.crypto")
val db       = @import("std.db")
val env      = @import("std.env")
val events   = @import("std.events")
val cache    = @import("std.cache")
val logger   = @import("std.logger")
val queue    = @import("std.queue")
val validate = @import("std.validate")
val ws       = @import("std.ws")
val mail     = @import("std.mail")
val ai       = @import("std.ai")
val utils    = @import("std.utils")
val test     = @import("std.test")
```

### Packages (installed via `lunex add`)

```lunex
val discord = @import("discordlunex")
val github  = @import("lunex-github")
```

---

## Stdlib Overview

| Module     | Import                           | Description                                   | Docs                                         |
|------------|----------------------------------|-----------------------------------------------|----------------------------------------------|
| `io`       | `@import("std.io")`      | Console output, colors, tables                | [docs/io.md](docs/io.md)                     |
| `fs`       | `@import("std.fs")`      | File system read/write                        | [docs/fs.md](docs/fs.md)                     |
| `http`     | `@import("std.http")`    | HTTP client and server                        | [docs/http.md](docs/http.md)                 |
| `crypto`   | `@import("std.crypto")`  | Hashing, HMAC, AES-GCM, bcrypt                | [docs/crypto.md](docs/crypto.md)             |
| `db`       | `@import("std.db")`      | In-memory database with schema                | [docs/db.md](docs/db.md)                     |
| `env`      | `@import("std.env")`     | Environment variables, .env loading           | [docs/env.md](docs/env.md)                   |
| `validate` | `@import("std.validate")`| Schema validation and format checking         | [docs/validate.md](docs/validate.md)         |
| `ws`       | `@import("std.ws")`      | WebSocket server and client                   | [docs/ws.md](docs/ws.md)                     |
| `mail`     | `@import("std.mail")`    | SMTP email with HTML                          | [docs/mail.md](docs/mail.md)                 |
| `ai`       | `@import("std.ai")`      | AI/LLM client (OpenAI-compatible)             | [docs/ai.md](docs/ai.md)                     |
| `utils`    | `@import("std.utils")`   | Array, string, math utilities                 | [docs/utils.md](docs/utils.md)               |
| `test`     | `@import("std.test")`    | Unit testing framework                        | [docs/test.md](docs/test.md)                 |
| `xml`      | `@import("std.xml")`     | XML parse / build                             | [docs/xml.md](docs/xml.md)                   |
| `os`       | `@import("std.os")`      | Process, signals, platform info               | [docs/os.md](docs/os.md)                     |
| `redis`    | `@import("std.redis")`   | Redis client                                  | [docs/redis.md](docs/redis.md)               |
| `postgres` | `@import("std.postgres")`| PostgreSQL client                             | [docs/postgres.md](docs/postgres.md)         |
| `mysql`    | `@import("std.mysql")`   | MySQL client                                  | [docs/mysql.md](docs/mysql.md)               |
| `jwt`      | `@import("std.jwt")`     | JSON Web Tokens                               | [docs/jwt.md](docs/jwt.md)                   |
| `stripe`   | `@import("std.stripe")`  | Stripe payments                               | [docs/stripe.md](docs/stripe.md)             |
| `oauth2`   | `@import("std.oauth2")`  | OAuth2 flows                                  | [docs/oauth2.md](docs/oauth2.md)             |
| `graphql`  | `@import("std.graphql")` | GraphQL client                                | [docs/graphql.md](docs/graphql.md)           |
| `rabbitmq` | `@import("std.rabbitmq")`| RabbitMQ client                               | [docs/rabbitmq.md](docs/rabbitmq.md)         |
| `excel`    | `@import("std.excel")`   | Excel file read/write                         | [docs/excel.md](docs/excel.md)               |
| `pdf`      | `@import("std.pdf")`     | PDF generation                                | [docs/pdf.md](docs/pdf.md)                   |
| `alloc`    | `@import("std.alloc")`   | Manual memory allocation                      | [docs/alloc.md](docs/alloc.md)               |

---

## Build Options

```bash
./build.sh              # build for current platform
./build.sh cross        # cross-compile for all platforms
./build.sh test         # run tests
./build.sh clean        # remove artifacts
```

### Requirements

Go 1.25+ and Zig 0.17 (installed automatically by `build.sh`).

## License

(c) David Dev 2026. See `License` file.
