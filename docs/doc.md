# Lunex lang — Language Reference

  > **Lunex v0.4.0** — A modern, dynamically-typed scripting language built on Go, designed for rapid development with a batteries-included standard library.

  ---

  ## Table of Contents

  1. [Overview](#1-overview)
  2. [Installation and Build](#2-installation-and-build)
  3. [CLI Commands](#3-cli-commands)
  4. [Language Syntax](#4-language-syntax)
     - [Variables](#41-variables)
     - [Data Types](#42-data-types)
     - [Operators](#43-operators)
     - [Control Flow](#44-control-flow)
     - [Functions](#45-functions)
     - [Structs (No Classes)](#46-structs-no-classes)
     - [Pattern Matching](#47-pattern-matching)
     - [Error Handling](#48-error-handling)
     - [Async / Concurrency](#49-async--concurrency)
     - [Modules](#410-modules)
  5. [Standard Library](#5-standard-library)
     - [io — Console I/O and Colors](#51-io--console-io-and-colors)
     - [fs — File System](#52-fs--file-system)
     - [http — HTTP Client and Server](#53-http--http-client-and-server)
     - [crypto — Cryptography](#54-crypto--cryptography)
     - [db — In-Memory Database](#55-db--in-memory-database)
     - [env — Environment Variables](#56-env--environment-variables)
     - [ws — WebSocket](#57-ws--websocket)
     - [mail — SMTP Email](#58-mail--smtp-email)
     - [ai — AI / LLM Client](#59-ai--ai--llm-client)
     - [utils — Utilities and Helpers](#510-utils--utilities-and-helpers)
     - [validate — Validation and Schema](#511-validate--validation-and-schema)
     - [os — Operating System](#512-os--operating-system)
     - [xml — XML Parsing and Generation](#513-xml--xml-parsing-and-generation)
  6. [Data Format Modules](#6-data-format-modules)
     - [csv — CSV Parsing](#61-csv--csv-parsing)
     - [yaml — YAML Parsing](#62-yaml--yaml-parsing)
     - [toml — TOML Parsing](#63-toml--toml-parsing)
     - [markdown — Markdown Rendering](#64-markdown--markdown-rendering)
     - [mustache — Mustache Templates](#65-mustache--mustache-templates)
  7. [Database Modules](#7-database-modules)
     - [postgres — PostgreSQL](#71-postgres--postgresql)
     - [mysql — MySQL / MariaDB](#72-mysql--mysql--mariadb)
     - [redis — Redis](#73-redis--redis)
  8. [Authentication and Payments](#8-authentication-and-payments)
     - [jwt — JSON Web Tokens](#81-jwt--json-web-tokens)
     - [oauth2 — OAuth2](#82-oauth2--oauth2)
     - [stripe — Stripe Payments](#83-stripe--stripe-payments)
  9. [Messaging and APIs](#9-messaging-and-apis)
     - [rabbitmq — RabbitMQ / AMQP](#91-rabbitmq--rabbitmq--amqp)
     - [graphql — GraphQL](#92-graphql--graphql)
  10. [Document Generation](#10-document-generation)
      - [excel — Excel Spreadsheets](#101-excel--excel-spreadsheets)
      - [pdf — PDF Generation](#102-pdf--pdf-generation)
  11. [Runtime Internals](#11-runtime-internals)
      - [Bytecode Format (.nc / .nax)](#111-bytecode-format-nc--nax)
      - [REPL](#112-repl)
      - [Built-in Editor](#113-built-in-editor)
      - [JIT Profiler](#114-jit-profiler)
      - [Package Manager](#115-package-manager)

  ---

  ## 1. Overview

  Lunex is a general-purpose scripting language that:

  - Compiles to an internal bytecode format (**.nc** files) with automatic caching
  - Can pack entire directories into single portable archives (**.nax** files)
  - Embeds the Go standard library for I/O, networking, crypto, and more
  - Integrates first-class support for databases, HTTP servers, WebSockets, AI providers, payments, and document generation
  - Provides an interactive REPL and a built-in TUI code editor

  **Architecture summary**

  ```
  Lunex Source (.lx)
        │
    Lexer → Parser → AST
        │
    NTLIR + ENFS optimizer (up to 12-pass)
        │
    Bytecode (.nc)
        │  (Go writes → Zig reads via pipe)
    Lunex VM  →  JIT Profiler  →  Native machine code (x86_64 / AArch64)
  ```

  The runtime uses a single `Value` type (tagged union) with the following tags:

  | Tag | Description |
  |-----|-------------|
  | `null` | Null value |
  | `undefined` | Absent / uninitialized value |
  | `boolean` | `true` / `false` |
  | `number` | IEEE 754 double |
  | `string` | UTF-8 string |
  | `array` | Ordered list of values |
  | `object` | Key/value map |
  | `function` | First-class function |
  | `class` | Class definition |
  | `instance` | Class instance |
  | `regex` | Regular expression |
  | `channel` | Async channel |
  | `error` | Error value |

  ---

  ## 2. Installation and Build

  ### Prerequisites

  - Go 1.23 or later

  ### Build from source

  ```sh
  go run tools/build.go                  # Development build
  go run tools/build.go release          # Stripped release build
  go run tools/build.go --install        # Build and install to ~/local/bin
  ```

  ### Install script (Linux / macOS)

  ```sh
  curl -fsSL https://... | sh            # see install.sh
  ```

  ### Install script (Windows)

  ```powershell
  irm https://... | iex                  # see install.ps1
  ```

  ---

  ## 3. CLI Commands

  | Command | Description |
  |---------|-------------|
  | `lunex <file.lx>` | Run an Lunex file (bytecode cached automatically) |
  | `lunex run <file>` | Run a `.lx`, `.nc`, or `.nax` file |
  | `lunex repl` | Start interactive REPL |
  | `lunex edit [file]` | Open the built-in TUI editor |
  | `lunex build <file.lx> [-o out.nc]` | Compile to `.nc` bytecode |
  | `lunex pack <dir> [-o out.nax]` | Pack a directory to `.nax` archive |
  | `lunex fmt <file.lx>` | Format source file in-place |
  | `lunex check <file.lx>` | Syntax/semantic check without running |
  | `lunex dis <file.nc>` | Disassemble a `.nc` object |
  | `lunex cache` | Show bytecode cache info |
  | `lunex cache clear` | Clear bytecode cache |
  | `lunex init [name]` | Initialize a new `lunex.mod` project |
  | `lunex install [pkg]` | Install all packages from `lunex.mod` |
  | `lunex add <pkg>` | Add + install a package (updates `lunex.mod`) |
  | `lunex remove <pkg>` | Remove a package |
  | `lunex list` | List installed packages |
  | `lunex version` | Show version |

  ### Bytecode cache

  Lunex automatically caches compiled bytecode alongside sources. On second run the cached `.nc` is used, skipping parsing. The cache lives in `~/.lx/cache/`.

  ---

  ## 4. Language Syntax

  ### 4.1 Variables

  ```lunex
  var x = 10         // mutable binding
  val y = "hello"    // immutable binding
  val PI = 3.14159   // immutable binding
  ```

  ### 4.2 Data Types

  ```lunex
  // Primitives
  var n = 42
  var f = 3.14
  var s = "text"
  var b = true
  var nul = null
  var u = undefined

  // Arrays
  var arr = [1, 2, 3]
  arr.push(4)
  var len = arr.length

  // Objects
  var obj = { name: "Alice", age: 30 }
  obj.email = "alice@example.com"

  // Template strings
  var msg = `Hello, ${obj.name}!`

  // Regular expressions
  var re = /^[a-z]+$/i
  re.test("hello")        // true

  // Ranges
  var r = 1..10           // range from 1 to 9
  ```

  ### 4.3 Operators

  ```lunex
  // Arithmetic
  +  -  *  /  %  **    // standard + power

  // Comparison
  ==  !=  <  >  <=  >= // structural equality
  ===  !==              // strict equality (type + value)

  // Logical
  &&  ||  !

  // Null coalescing
  var val = x ?? "default"

  // Optional chaining
  var city = user?.address?.city

  // Spread / rest
  var merged = { ...a, ...b }
  var [first, ...rest] = arr

  // Bitwise
  &  |  ^  ~  <<  >>
  ```

  ### 4.4 Control Flow

  ```lunex
  // if / elif / else
  if x > 0 {
      io.log("positive")
  } elif x == 0 {
      io.log("zero")
  } else {
      io.log("negative")
  }

  // unless (negated if)
  unless x > 100 { io.log("in range") }

  // while
  while condition { }

  // for ... in (range / array)
  each i in 0..10 { io.log(i) }
  each item in list { io.log(item) }

  // for ... of (key-value)
  for key, value of obj { io.log(key, value) }

  // loop (infinite)
  loop {
      if done { break }
  }

  // each (functional)
  [1, 2, 3].each(fn(x) { io.log(x) })

  // repeat N times
  repeat 5 { io.log("hello") }

  // do ... while
  do { } while condition

  // guard (early return)
  guard x > 0 else { return }

  // defer
  defer io.log("cleanup")
  ```

  ### 4.5 Functions

  ```lunex
  // Named function
  fn add(a, b) {
      a + b
  }

  // Arrow / anonymous
  var double = fn(x) { return x * 2 }
  var greet = fn(name) { return "Hello, " + name }

  // Default parameters
  fn connect(host, port = 3000) { }

  // Rest parameters
  fn sum(...nums) {
      nums.reduce(fn(acc, n) { return acc + n }, 0)
  }

  // Closures
  fn counter() {
      var n = 0
      fn() { n += 1; return n }
  }

  // Immediately invoked
  var result = fn(x) { return x * x }(5)
  ```

  ### 4.6 Structs (No Classes)

  Lunex has **no** `class` keyword. Use constructor functions with `struct { ... }` to define named types with methods.

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
          fn speak()      { base.speak() }
          fn learn(trick) { self.tricks.push(trick) }
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

  val d = Dog("Rex")
  io.log(d.speak())
  d.learn("sit")
  io.log(d.perform())
  ```

  > **Removed:** `class`, `constructor`, `extends`, `super`, `abstract`, `interface`, `trait`, `implements`, `new`. Use the struct pattern above instead.

    ### 4.7 Pattern Matching

  ```lunex
  match value {
      case 0 => io.log("zero")
      case 1..10 => io.log("small")
      case "hello" => io.log("greeting")
      case [a, b] => io.log("two-element array")
      case { name } => io.log("object with name: " + name)
      default => io.log("no match")
  }

  // when guards
  match x {
      case n when n > 0 => io.log("positive")
      case n when n < 0 => io.log("negative")
      default => io.log("zero")
  }
  ```

  ### 4.8 Error Handling

  ```lunex
  // try / catch / finally
  try {
      var data = JSON.parse(bad)
  } catch err {
      io.log("Error:", err)
  } finally {
      io.log("Done")
  }

  // raise / throw
  raise "something went wrong"
  throw new Error("invalid input")

  // Result-style (functions returning null on error)
  var result = fs.readFile("missing.txt")  // returns null on error
  ```

  ### 4.9 Async / Concurrency

  ```lunex
  // Async functions
  async fn fetchUser(id) {
      var res = await http.get("https://api.example.com/users/" + id)
      res.json()
  }

  // Await
  var user = await fetchUser(1)

  // Channels
  var ch = channel()
  spawn fn() { ch.send("hello") }
  var msg = ch.receive()

  // Parallel execution
  var [a, b, c] = await parallel(
      fetchUser(1),
      fetchUser(2),
      fetchUser(3)
  )
  ```

  ### 4.10 Modules

  ```lunex
  // Import standard library modules
  val http     = @import("std.http")
  val fs       = @import("std.fs")
  val crypto   = @import("std.crypto")
  val io       = @import("std.io")
  val env      = @import("std.env")

  // Import third-party packages (installed via lunex add)
  val discord = @import("discordlunex")
  val github  = @import("lunex-github")
  ```

  ---

  ## 5. Standard Library

  Import any stdlib module with `use "<name>"`.

  ### 5.1 `io` — Console I/O and Colors

  Provides terminal I/O, colored output, and progress bars.

  ```lunex
  val io = @import("std.io")

  io.log("Hello")              // print to stdout
  io.error("something broke")  // print to stderr
  io.warn("watch out")
  io.success("done!")
  io.info("starting...")

  // Read input
  var line = io.readLine("Enter name: ")
  var n    = io.readInt("Enter number: ")

  // Colored output (returns colored string)
  io.red("error")
  io.green("ok")
  io.yellow("warning")
  io.blue("info")
  io.cyan("debug")
  io.magenta("special")
  io.bold("important")
  io.dim("subtle")

  // Table
  io.table([{ name: "Alice", age: 30 }, { name: "Bob", age: 25 }])

  // Progress bar
  io.progress(50, 100, "Loading")

  // Spinner
  io.spinner("Processing...")

  // Banner and separator
  io.banner("Lunex", "green")
  io.hr("-", 40)

  // Clear terminal
  io.clear()
  ```

  ---

  ### 5.2 `fs` — File System

  Full file system access: read, write, directory operations, and metadata.

  ```lunex
  val fs = @import("std.fs")

  // Read / Write
  var content = fs.readFile("file.txt")          // string or null
  fs.writeFile("out.txt", "content")
  fs.appendFile("log.txt", "new line\n")

  // Delete / Copy / Move
  fs.deleteFile("file.txt")
  fs.copyFile("src.txt", "dst.txt")
  fs.moveFile("old.txt", "new.txt")

  // Directories
  fs.mkdir("dir")
  fs.mkdir("deep/nested/dir", true)              // recursive
  fs.rmdir("dir")
  fs.rmdir("dir", true)                          // recursive

  // List
  var entries = fs.list(".")                     // array of names
  var dirs    = fs.listDir(".")                  // directories only

  // Existence / Stats
  var exists = fs.exists("path")                 // bool
  var stat   = fs.stat("path")                   // { name, size, isDir, isFile, modTime }
  var isDir  = fs.isDir("path")                  // bool
  var isFile = fs.isFile("path")                 // bool

  // Path helpers
  var abs    = fs.resolve("relative/path")
  var base   = fs.basename("/path/file.txt")     // "file.txt"
  var dir    = fs.dirname("/path/file.txt")      // "/path"
  var ext    = fs.extname("file.txt")            // ".txt"
  var joined = fs.join("dir", "sub", "file.txt")
  var ok     = fs.isAbsolute("/tmp/file")        // bool

  // Glob
  var files = fs.glob("src/**/*.lx")

  // JSON convenience
  var obj = fs.readJSON("config.json")           // parsed object or null
  fs.writeJSON("config.json", obj)
  fs.writeJSON("config.json", obj, true)         // pretty-print

  // Watch
  fs.watch("file.txt", fn(event) {
      io.log(event.type, event.path)
  })
  ```

  ---

  ### 5.3 `http` — HTTP Client and Server

  Full HTTP/1.1 client with fetch-style API and a full-featured server with routing.

  #### Client

  ```lunex
  val http = @import("std.http")

  // GET
  var res = await http.get("https://api.example.com/data")
  // res: { status, headers, body, json(), text() }

  io.log(res.status)         // 200
  var data = res.json()     // parsed JSON object
  var text = res.text()     // raw string

  // POST with JSON body
  var res = await http.post("https://api.example.com/users", {
      body: { name: "Alice", email: "alice@example.com" },
      headers: { "Authorization": "Bearer token123" }
  })

  // Other methods
  await http.put(url, opts)
  await http.patch(url, opts)
  await http.del(url, opts)
  await http.head(url, opts)
  ```

  #### Server

  ```lunex
  val http = @import("std.http")
  val io   = @import("std.io")

  // Simple server with http.serve(port, handler)
  http.serve(3000, fn(req, res) {
      http.json(res, { message: "Hello from Lunex" }, 200)
  })

  // Full server with createServer + listen
  val server = http.createServer(fn(req, res) {
      if req.url == "/" {
          http.text(res, "Hello", 200)
      } elif req.url == "/data" {
          http.json(res, { ok: true }, 200)
      } else {
          http.text(res, "Not Found", 404)
      }
  })

  http.listen(server, 3000, "0.0.0.0", fn() {
      io.log("Server running on port 3000")
  })

  // Response helpers
  http.json(res, obj, 200)        // send JSON response
  http.text(res, "ok", 200)       // send text response
  http.redirect(res, "/other")    // 302 redirect
  http.redirect(res, "/other", 301)
  ```

  ---

  ### 5.4 `crypto` — Cryptography

  Hashing, HMAC, symmetric encryption, and key generation.

  ```lunex
  val crypto = @import("std.crypto")

  // Hashing
  crypto.hash("sha256", "data")       // hex string
  crypto.hash("md5", "data")
  crypto.hash("sha1", "data")
  crypto.hash("sha512", "data")

  // HMAC
  crypto.hmac("sha256", "secret", "message")   // hex string

  // Base64
  crypto.base64Encode("hello world")       // "aGVsbG8gd29ybGQ="
  crypto.base64Decode("aGVsbG8gd29ybGQ=") // "hello world"
  crypto.base64UrlEncode("hello world")
  crypto.base64UrlDecode("aGVsbG8gd29ybGQ")

  // Hex
  crypto.toHex("hello")       // "68656c6c6f"
  crypto.fromHex("68656c6c6f") // "hello"

  // AES-256-GCM encryption
  var encrypted = crypto.encrypt("plaintext", "my-32-char-secret-key-for-aes256")
  var decrypted = crypto.decrypt(encrypted, "my-32-char-secret-key-for-aes256")

  // Random
  var bytes = crypto.randomBytes(32)   // random bytes
  var hex   = crypto.randomHex(16)     // random hex string
  var tok   = crypto.token(32)         // secure hex token
  var id    = crypto.randomUUID()      // UUID v4

  // Password hashing (bcrypt)
  var hash = crypto.hashPassword("mypassword")
  var ok   = crypto.verifyPassword("mypassword", hash)   // bool

  // Timing-safe compare
  crypto.compare(a, b)

  // PBKDF2
  var key = crypto.pbkdf2("password", "salt", 100000, 32)
  ```

  ---

  ### 5.5 `db` — In-Memory Database

  A full SQL-inspired in-memory database engine with collections, indexes, transactions, and aggregations.

  ```lunex
  val db = @import("std.db")

  // Open / create a named database
  var database = db.open("mydb")

  // Create a table (collection)
  var users = database.table("users")

  // Insert
  var id = users.insert({ name: "Alice", age: 30, role: "admin" })
  users.insertMany([
      { name: "Bob", age: 25 },
      { name: "Carol", age: 28 }
  ])

  // Find
  var all = users.find()                         // all records
  var alice = users.find({ name: "Alice" })      // filter
  var first = users.findOne({ name: "Alice" })   // first match
  var byId = users.findById(id)                  // by internal id

  // Query builder
  var results = users
      .where({ role: "admin" })
      .where(fn(row) => row.age >= 25)
      .orderBy("age", "asc")
      .limit(10)
      .offset(0)
      .select(["name", "age"])
      .toArray()

  // Update
  users.update({ name: "Alice" }, { age: 31 })
  users.updateById(id, { age: 31 })
  users.upsert({ name: "Alice" }, { name: "Alice", age: 31 })

  // Delete
  users.delete({ name: "Alice" })
  users.deleteById(id)

  // Count
  var count = users.count()
  var adminCount = users.count({ role: "admin" })

  // Aggregation
  var stats = users.aggregate([
      { $match: { role: "admin" } },
      { $group: { _id: "role", total: { $sum: 1 }, avgAge: { $avg: "age" } } }
  ])

  // Indexes for fast lookup
  users.createIndex("email", { unique: true })
  users.createIndex(["lastName", "firstName"])

  // Transactions
  var tx = database.transaction()
  tx.begin()
  try {
      users.insert({ name: "Dave" })
      tx.commit()
  } catch e {
      tx.rollback()
  }

  // Drop / info
  users.drop()
  database.drop()
  var info = users.info()   // { name, count, indexes }

  // List all databases / tables
  var dbs = db.list()
  var tables = database.tables()
  ```

  ---

  ### 5.6 `env` — Environment Variables

  ```lunex
  val env = @import("std.env")

  var value = env.get("DATABASE_URL")            // string or undefined
  var value = env.get("PORT", "3000")            // with default
  env.set("KEY", "value")
  var has = env.has("API_KEY")                   // bool
  env.delete("OLD_VAR")
  var all = env.all()                            // object of all env vars
  var parsed = env.parse(".env")                 // parse .env file
  ```

  ---

  ### 5.7 `ws` — WebSocket

  WebSocket client and server (no external library dependency — pure Go implementation).

  ```lunex
  val ws = @import("std.ws")

  // Server
  val server = ws.createServer(8080, {})

  server.on("connection", fn(conn) {
      io.log("Client connected")
      server.send(conn, "Welcome!")
  })

  server.on("message", fn(conn, msg) {
      io.log("Received:", msg)
      server.send(conn, "Echo: " + msg)
  })

  server.on("close", fn(conn) {
      io.log("Client disconnected")
  })

  server.start(fn() {
      io.log("WebSocket server on port 8080")
  })

  // Broadcast to all clients
  server.broadcast("Hello everyone")
  io.log(server.clientCount())

  server.stop()

  // Client
  val client = ws.createClient("ws://localhost:8080")

  client.on("open", fn() { io.log("Connected") })
  client.on("message", fn(msg) { io.log("Server says:", msg) })
  client.on("close", fn() { io.log("Disconnected") })

  client.connect()
  client.send("Hello server")
  client.close()
  ```

  ---

  ### 5.8 `mail` — SMTP Email

  ```lunex
  val mail = @import("std.mail")

  var mailer = mail.createMailer({
      host: "smtp.gmail.com",
      port: 587,
      user: "me@gmail.com",
      password: env.get("SMTP_PASS"),
      from: "me@gmail.com"
  })

  // Send plain text
  mailer.send({
      to: "recipient@example.com",
      subject: "Hello",
      text: "This is a test email."
  })

  // Send HTML
  mailer.send({
      to: "recipient@example.com",
      subject: "Welcome",
      html: "<h1>Welcome!</h1><p>Thanks for signing up.</p>",
      from: "noreply@example.com"
  })
  ```

  ---

  ### 5.9 `ai` — AI / LLM Client

  Connect to OpenAI, Anthropic, Ollama, or any OpenAI-compatible API.

  ```lunex
  val ai = @import("std.ai")

  // Create a client
  var client = ai.create({
      apiKey: env.get("OPENAI_API_KEY"),
      model: "gpt-4",
      provider: "openai",           // "openai" | "anthropic" | "ollama" | custom
      baseUrl: "https://api.openai.com/v1",
      timeout: 30000
  })

  // Single chat completion
  var reply = await client.chat("What is the capital of France?")
  io.log(reply)

  // Multi-turn conversation
  var response = await client.chat([
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Tell me a joke." }
  ], {
      model: "gpt-3.5-turbo",
      maxTokens: 256,
      temperature: 0.7
  })

  // Streaming
  await client.stream("Write a poem about rain", fn(chunk) {
      io.log(chunk)
  }, { model: "gpt-4" })

  // Embeddings
  var vec = await client.embed("some text to embed")

  // Ollama (local)
  var local = ai.create({
      provider: "ollama",
      baseUrl: "http://localhost:11434",
      model: "llama3"
  })
  var reply = await local.chat("Hello!")
  ```

  ---

  ### 5.10 `utils` — Utilities and Helpers

  A comprehensive functional utility library for arrays, objects, strings, numbers, and functions.

  ```lunex
  val utils = @import("std.utils")

  // ── Time ──────────────────────────────────────────────
  utils.sleep(1000)               // sleep N milliseconds
  utils.now()                     // current timestamp in ms
  utils.timestamp()               // Unix timestamp in seconds

  // ── Array Utilities ───────────────────────────────────
  utils.range(5)                  // [0,1,2,3,4]
  utils.range(1, 6)               // [1,2,3,4,5]
  utils.range(0, 10, 2)           // [0,2,4,6,8]

  utils.chunk([1,2,3,4,5], 2)    // [[1,2],[3,4],[5]]
  utils.flatten([[1,[2]],3])      // [1,2,3]
  utils.flatten([[1,[2]],3], 1)   // [1,[2],3] (depth 1)
  utils.flatMap([1,2], fn(x) => [x, x*2])  // [1,2,2,4]

  utils.zip([1,2,3], ["a","b","c"])         // [[1,"a"],[2,"b"],[3,"c"]]
  utils.unzip([[1,"a"],[2,"b"]])            // [[1,2],["a","b"]]

  utils.intersection([1,2,3], [2,3,4])     // [2,3]
  utils.difference([1,2,3], [2,3])         // [1]
  utils.union([1,2], [2,3])                // [1,2,3]

  utils.uniq([1,2,2,3,3])                  // [1,2,3]
  utils.uniqBy(arr, "id")                  // unique by field
  utils.uniqBy(arr, fn(x) => x.id)        // unique by function

  utils.groupBy(arr, "status")             // { active:[...], inactive:[...] }
  utils.countBy(arr, "status")             // { active:3, inactive:2 }

  utils.partition(arr, fn(x) => x > 0)    // [[positives], [negatives]]
  utils.sortBy(arr, "name")               // sort by field asc
  utils.sortBy(arr, fn(x) => x.age, "desc")

  // ── Object Utilities ──────────────────────────────────
  utils.pick(obj, ["a","b"])              // { a:..., b:... }
  utils.omit(obj, ["secret"])             // object without "secret"
  utils.merge(a, b, c)                    // deep merge
  utils.defaults(obj, { timeout: 5000 }) // fill missing keys
  utils.invert({ a: 1, b: 2 })           // { "1":"a", "2":"b" }

  utils.keys(obj)                         // array of keys
  utils.values(obj)                       // array of values
  utils.entries(obj)                      // [[key,val],...]
  utils.fromEntries([["a",1],["b",2]])    // { a:1, b:2 }
  utils.mapValues(obj, fn(v) => v * 2)   // map object values
  utils.filterValues(obj, fn(v) => v > 0)
  utils.has(obj, "key")                   // bool

  // ── String Utilities ──────────────────────────────────
  utils.camelCase("hello world")          // "helloWorld"
  utils.snakeCase("helloWorld")           // "hello_world"
  utils.kebabCase("Hello World")          // "hello-world"
  utils.pascalCase("hello world")         // "HelloWorld"
  utils.titleCase("hello world")          // "Hello World"
  utils.upperFirst("hello")              // "Hello"
  utils.lowerFirst("Hello")              // "hello"

  utils.pad("5", 3, "0")                 // "005"
  utils.padStart("5", 3, "0")            // "005"
  utils.padEnd("5", 3, "0")             // "500"

  utils.truncate("Hello World", 8)       // "Hello..."
  utils.truncate("Hello World", 8, "..") // "Hello.."

  utils.repeat("ab", 3)                  // "ababab"
  utils.reverse("hello")                 // "olleh"
  utils.trim("  hello  ")               // "hello"
  utils.trimStart("  hello")            // "hello"
  utils.trimEnd("hello  ")              // "hello"

  utils.startsWith("hello", "hel")       // true
  utils.endsWith("hello", "llo")         // true
  utils.includes("hello", "ell")         // true

  utils.count("banana", "a")             // 3
  utils.words("Hello World")             // ["Hello","World"]
  utils.escape("<div>")                  // "&lt;div&gt;"
  utils.unescape("&lt;div&gt;")          // "<div>"

  // ── Number Utilities ──────────────────────────────────
  utils.clamp(150, 0, 100)               // 100
  utils.clamp(-5, 0, 100)               // 0
  utils.lerp(0, 100, 0.5)               // 50
  utils.round(3.456, 2)                  // 3.46
  utils.floor(3.7)                       // 3
  utils.ceil(3.2)                        // 4
  utils.abs(-5)                          // 5

  utils.min([3,1,4,1,5])                 // 1
  utils.max([3,1,4,1,5])                 // 5
  utils.sum([1,2,3,4,5])                 // 15
  utils.mean([1,2,3,4,5])               // 3
  utils.median([1,2,3,4,5])             // 3

  utils.random()                         // float 0..1
  utils.random(10)                       // int 0..9
  utils.random(5, 10)                    // int 5..9
  utils.shuffle([1,2,3,4,5])            // random order
  utils.sample([1,2,3,4,5])             // random element
  utils.sample([1,2,3,4,5], 2)          // 2 random elements

  utils.formatNumber(1234567.89, 2)      // "1,234,567.89"
  utils.formatBytes(1048576)             // "1 MB"

  // ── Identity / Type Utilities ─────────────────────────
  utils.identity(x)                      // returns x unchanged
  utils.noop()                           // does nothing, returns undefined
  utils.type(value)                      // "string" | "number" | ...
  utils.isEmpty(value)                   // true if null/undefined/[]/{}/"" 
  utils.isNil(value)                     // true if null or undefined
  utils.isEmail("a@b.com")               // bool
  utils.isUrl("https://example.com")     // bool
  utils.isNumeric("42.5")                // bool
  utils.toNumber("42")                   // 42
  utils.toString(42)                     // "42"
  utils.toJSON(obj)                      // JSON string
  utils.fromJSON(str)                    // parsed value
  utils.clone(obj)                       // deep clone
  utils.equal(a, b)                      // deep equality

  // ── Function Utilities ────────────────────────────────
  utils.memoize(fn)                      // returns memoized version
  utils.once(fn)                         // call only first time
  utils.negate(fn)                       // returns negated predicate
  utils.compose(f, g, h)                 // h(g(f(x)))
  utils.pipe(f, g, h)                    // f(g(h(x))) (reverse compose)

  // ── UUID ──────────────────────────────────────────────
  utils.uuid()                           // "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
  ```

  ---

  ### 5.11 `validate` — Validation and Schema

  Provides regex-based validators and a composable schema validation system.

  ```lunex
  val validate = @import("std.validate")

  // Basic validators (return bool)
  validate.isEmail("user@example.com")       // true
  validate.isUrl("https://example.com")      // true
  validate.isPhone("+1 555-123-4567")        // true
  validate.isIPv4("192.168.1.1")             // true
  validate.isIPv6("2001:0db8::1")            // true
  validate.isIP("192.168.1.1")               // true (v4 or v6)
  validate.isUUID("550e8400-e29b-41d4-a716-446655440000")  // true
  validate.isAlpha("hello")                  // true
  validate.isAlphanumeric("hello123")        // true
  validate.isHex("deadbeef")                 // true
  validate.isNumeric("42.5")                 // true
  validate.isBase64("aGVsbG8=")             // true
  validate.isJSON('{"a":1}')                 // true
  validate.isCreditCard("4111111111111111")  // true (Luhn check)
  validate.isSlug("my-article-slug")         // true
  validate.isDate("2024-01-15")              // true
  validate.isStrongPassword("P@ssw0rd!")     // true (8+ chars, upper, lower, digit, special)

  // Regex match
  validate.matches("hello123", "[a-z]+[0-9]+")   // bool

  // Length / range
  validate.length("hello", 3, 10)           // true (between 3 and 10 chars)
  validate.range(7, 1, 10)                  // true (number between 1 and 10)

  // Schema builder
  var schema = validate.schema({
      type: "object",
      properties: {
          name:  { type: "string", minLength: 1, maxLength: 100, required: true },
          email: { type: "string", format: "email", required: true },
          age:   { type: "number", min: 0, max: 120 },
          role:  { type: "string", enum: ["admin", "user", "guest"] },
          tags:  { type: "array", minItems: 1, maxItems: 10,
                   items: { type: "string" } }
      }
  })

  var result = schema.validate({ name: "Alice", email: "alice@example.com" })
  // result: { valid: true, error: null }

  var result = schema.validate({ name: "" })
  // result: { valid: false, error: "field 'name': value must be at least 1 characters" }

  var ok = schema.check({ name: "Alice", email: "alice@example.com" })  // bool

  schema.assert(input)   // throws if invalid, returns input if valid

  // Validate an object against a field schema map
  var result = validate.validate(userInput, {
      name:  { type: "string", required: true },
      email: { type: "string", format: "email", required: true }
  })
  // result: { valid: bool, errors: [{ field, message }] }

  // Schema types
  // "string"  — minLength, maxLength, pattern, format ("email"|"url"|"uuid"|"ipv4")
  // "number"  — min, max
  // "boolean"
  // "array"   — minItems, maxItems, items (sub-schema)
  // "object"  — properties (field schemas)
  // "any"     — no type restriction
  // "date"    — accepts string or number
  ```

  ---

  ### 5.12 `os` — Operating System

  Process management, environment, file system metadata, and path helpers.

  ```lunex
  val os = @import("std.os")

  // Process execution (synchronous)
  var result = os.exec("ls -la /tmp")
  // result: { stdout, stderr, code, ok }

  var result = os.exec("npm install", {
      cwd: "/project",
      env: { NODE_ENV: "production" },
      timeout: 30000
  })

  // Spawn (async, non-blocking)
  var proc = os.spawn("python3 server.py")
  var code = proc.wait()       // wait for exit
  proc.kill()                  // kill process
  io.log(proc.pid)

  // Environment
  os.getenv("PATH")
  os.setenv("FOO", "bar")
  os.unsetenv("FOO")
  var all = os.environ()       // object of all env vars

  // Process info
  os.getpid()                  // current PID
  os.getppid()                 // parent PID
  os.getcwd()                  // current working directory
  os.chdir("/tmp")             // change directory
  os.hostname()                // machine hostname

  // Platform info
  os.platform()                // "linux" | "darwin" | "windows"
  os.arch()                    // "amd64" | "arm64" ...
  os.cpus()                    // number of CPUs

  // Exit
  os.exit(0)
  os.exit(1)

  // CLI args
  var args = os.args()         // array of strings

  // File system operations
  os.stat("file.txt")          // { name, size, isDir, isFile, mode, modTime }
  os.exists("file.txt")        // bool
  os.mkdir("dir")
  os.mkdir("deep/dir", true)   // recursive
  os.remove("file.txt")
  os.remove("dir", true)       // recursive
  os.rename("old.txt", "new.txt")
  var entries = os.listDir(".")  // [{ name, isDir, isFile, size, modTime }]
  var matches = os.glob("*.txt")  // array of matching paths

  // Temp
  os.tempDir()                 // system temp directory path
  os.tempFile("prefix-")       // creates a temp file, returns path

  // Paths
  os.join("dir", "sub", "file.txt")
  os.dirname("/path/file.txt")   // "/path"
  os.basename("/path/file.txt")  // "file.txt"
  os.extname("file.txt")         // ".txt"
  os.abs("./relative")           // absolute path
  os.expandEnv("$HOME/projects") // expand env vars

  // Constants
  os.sep      // path separator ("/" or "\")
  os.pathSep  // path list separator (":" or ";")
  os.eol      // line ending ("\n")
  os.homeDir  // user home directory
  ```

  ---

  ### 5.13 `xml` — XML Parsing and Generation

  ```lunex
  val xml = @import("std.xml")

  // Parse XML string to object
  var doc = xml.parse('<root><user id="1"><name>Alice</name></user></root>')
  // doc: { tag: "root", attrs: {}, children: [...], text: "" }

  // Stringify object to XML
  var str = xml.stringify(obj, "root")
  var pretty = xml.stringify(obj, "root", { pretty: true })

  // Read / Write XML files
  var doc = xml.readFile("config.xml")
  xml.writeFile("output.xml", obj, "root")
  xml.writeFile("output.xml", obj, "root", { pretty: true })

  // Parsed node structure:
  // {
  //   tag: "tagName",
  //   attrs: { id: "1", class: "example" },
  //   children: [ ...nodes ],
  //   text: "inner text"
  // }
  ```

  ---

  ## 6. Data Format Modules

  ### 6.1 `csv` — CSV Parsing

  ```lunex
  val csv = @import("std.csv")

  // Parse CSV string
  var rows = csv.parse("name,age\nAlice,30\nBob,25")
  // rows: [{ name: "Alice", age: "30" }, { name: "Bob", age: "25" }]

  var rows = csv.parse(data, { separator: ";", header: false })
  // Without header: returns arrays instead of objects

  // Stringify array to CSV
  var csvStr = csv.stringify(rows)
  var csvStr = csv.stringify(rows, { separator: ";", header: true })

  // Read / Write CSV files
  var rows = csv.readFile("data.csv")
  var rows = csv.readFile("data.csv", { separator: "\t" })
  csv.writeFile("output.csv", rows)
  csv.writeFile("output.csv", rows, { separator: ";" })
  ```

  ### 6.2 `yaml` — YAML Parsing

  ```lunex
  val yaml = @import("std.yaml")

  var obj = yaml.parse("name: Alice\nage: 30")
  var str = yaml.stringify({ name: "Alice", age: 30 })

  var obj = yaml.readFile("config.yaml")
  yaml.writeFile("output.yaml", obj)
  ```

  ### 6.3 `toml` — TOML Parsing

  ```lunex
  val toml = @import("std.toml")

  var obj = toml.parse('[database]\nhost = "localhost"')
  var str = toml.stringify({ database: { host: "localhost" } })

  var config = toml.readFile("config.toml")
  toml.writeFile("config.toml", config)
  ```

  ### 6.4 `markdown` — Markdown Rendering

  Converts Markdown to HTML using Goldmark (supports tables, strikethrough, autolinks, task lists).

  ```lunex
  val markdown = @import("std.markdown")

  var html = markdown.toHTML("# Hello\n\nThis is **bold**.")
  var html = markdown.readFile("README.md")      // parse and return HTML
  markdown.renderFile("template.md", "out.html") // write HTML file
  ```

  ### 6.5 `mustache` — Mustache Templates

  Logic-less templates following the Mustache spec.

  ```lunex
  val mustache = @import("std.mustache")

  var html = mustache.render("Hello, {{name}}!", { name: "Alice" })

  var result = mustache.renderFile("template.mustache", { title: "Home", items: [...] })

  // Compile template for reuse
  var tmpl = mustache.parse("Hello, {{name}}!")
  var out = tmpl.render({ name: "Bob" })
  ```

  ---

  ## 7. Database Modules

  ### 7.1 `postgres` — PostgreSQL

  Persistent connection pool backed by [pgx/v5](https://github.com/jackc/pgx).

  ```lunex
  val postgres = @import("std.postgres")

  var db = await postgres.connect("postgresql://user:pass@localhost:5432/mydb")
  // or
  var db = await postgres.connect({
      host: "localhost",
      port: 5432,
      database: "mydb",
      user: "user",
      password: "pass",
      maxConns: 10
  })

  // Query — returns array of objects
  var users = await db.query("SELECT * FROM users WHERE active = $1", [true])

  // Execute — returns { rowsAffected }
  await db.exec("UPDATE users SET active = $1 WHERE id = $2", [false, 5])

  // Single row
  var user = await db.queryOne("SELECT * FROM users WHERE id = $1", [1])

  // Insert with RETURNING
  var user = await db.insert("users", { name: "Alice", email: "alice@example.com" })
  // uses INSERT INTO ... RETURNING *

  // Transactions
  var tx = await db.begin()
  try {
      await tx.exec("INSERT INTO orders ...")
      await tx.exec("UPDATE inventory ...")
      await tx.commit()
  } catch e {
      await tx.rollback()
  }

  db.close()
  ```

  ### 7.2 `mysql` — MySQL / MariaDB

  Backed by [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql).

  ```lunex
  val mysql = @import("std.mysql")

  var db = await mysql.connect("user:pass@tcp(localhost:3306)/mydb")
  // or connection object
  var db = await mysql.connect({
      host: "localhost",
      port: 3306,
      database: "mydb",
      user: "user",
      password: "pass"
  })

  var rows = await db.query("SELECT * FROM users WHERE id = ?", [1])
  await db.exec("UPDATE users SET name = ? WHERE id = ?", ["Alice", 1])
  var row = await db.queryOne("SELECT * FROM users WHERE id = ?", [1])

  // Transactions
  var tx = await db.begin()
  await tx.exec("INSERT INTO users (name) VALUES (?)", ["Dave"])
  await tx.commit()

  db.close()
  ```

  ### 7.3 `redis` — Redis

  Backed by [go-redis/v9](https://github.com/redis/go-redis).

  ```lunex
  val redis = @import("std.redis")

  var client = redis.connect({ host: "localhost", port: 6379, db: 0 })
  // or URL
  var client = redis.connect("redis://:password@localhost:6379/0")

  // Key-Value
  await client.set("key", "value")
  await client.set("key", "value", 5000)     // TTL in ms
  var val = await client.get("key")          // string or null
  await client.delete("key")
  var exists = await client.exists("key")    // bool
  await client.expire("key", 3600)           // TTL in seconds
  var ttl = await client.ttl("key")          // seconds remaining

  // Hashes
  await client.hset("user:1", "name", "Alice")
  var name = await client.hget("user:1", "name")
  var all = await client.hgetall("user:1")   // object
  await client.hmset("user:1", { name: "Alice", age: "30" })
  await client.hdel("user:1", "age")
  var len = await client.hlen("user:1")

  // Lists
  await client.lpush("queue", "item1")
  await client.rpush("queue", "item2")
  var item = await client.lpop("queue")
  var item = await client.rpop("queue")
  var all = await client.lrange("queue", 0, -1)
  var len = await client.llen("queue")

  // Sets
  await client.sadd("myset", "a", "b", "c")
  var members = await client.smembers("myset")
  var isMember = await client.sismember("myset", "a")    // bool
  await client.srem("myset", "a")
  var len = await client.scard("myset")

  // Sorted Sets
  await client.zadd("scores", 100, "alice")
  await client.zadd("scores", 200, "bob")
  var members = await client.zrange("scores", 0, -1)
  var rank = await client.zrank("scores", "alice")
  var score = await client.zscore("scores", "alice")

  // Pub/Sub
  var sub = client.subscribe("channel")
  sub.onMessage(fn(msg) { io.log("Received:", msg) })
  client.publish("channel", "hello!")
  sub.unsubscribe()

  // Keys
  var keys = await client.keys("user:*")
  await client.flushdb()     // clear current database
  client.close()
  ```

  ---

  ## 8. Authentication and Payments

  ### 8.1 `jwt` — JSON Web Tokens

  Backed by [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt).

  ```lunex
  val jwt = @import("std.jwt")

  // Sign
  var token = jwt.sign({ userId: 123, role: "admin" }, "secret", {
      algorithm: "HS256",    // HS256 | HS384 | HS512 | RS256
      expiresIn: 3600,       // seconds
      issuer: "myapp",
      audience: "users",
      subject: "auth"
  })

  // Verify
  var payload = jwt.verify(token, "secret")
  // payload: { userId: 123, role: "admin", exp: ..., iat: ... }
  // throws if invalid or expired

  // Decode (no verification)
  var payload = jwt.decode(token)

  // Refresh
  var newToken = jwt.refresh(token, "secret", { expiresIn: 7200 })

  // Check expiry
  var expired = jwt.isExpired(token)    // bool
  ```

  ### 8.2 `oauth2` — OAuth2

  Backed by [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2).

  ```lunex
  val oauth2 = @import("std.oauth2")

  // Google
  var google = oauth2.google({
      clientId: env.get("GOOGLE_CLIENT_ID"),
      clientSecret: env.get("GOOGLE_CLIENT_SECRET"),
      redirectUrl: "https://app.example.com/auth/callback",
      scopes: ["profile", "email"]
  })

  // GitHub
  var github = oauth2.github({
      clientId: env.get("GITHUB_CLIENT_ID"),
      clientSecret: env.get("GITHUB_CLIENT_SECRET"),
      redirectUrl: "https://app.example.com/auth/callback",
      scopes: ["user:email"]
  })

  // Custom provider
  var custom = oauth2.custom({
      clientId: "...",
      clientSecret: "...",
      redirectUrl: "...",
      authUrl: "https://provider.com/oauth/authorize",
      tokenUrl: "https://provider.com/oauth/token",
      scopes: ["read"]
  })

  // Get authorization URL (redirect user here)
  var url = google.authURL("random-state-string")

  // Exchange code for token
  var token = await google.exchange(req.query.code)
  // token: { accessToken, refreshToken, tokenType, expiry }

  // Refresh token
  var newToken = await google.refresh(token.refreshToken)

  // Fetch user info with token
  var userInfo = await google.userInfo(token.accessToken)
  ```

  ### 8.3 `stripe` — Stripe Payments

  Backed by [stripe-go/v76](https://github.com/stripe/stripe-go).

  ```lunex
  val stripe = @import("std.stripe")

  var client = stripe.create(env.get("STRIPE_SECRET_KEY"))

  // Payment Intents
  var intent = await client.createPaymentIntent({
      amount: 1999,           // cents
      currency: "usd",
      customer: "cus_xxx",
      description: "Order #42"
  })
  // intent: { id, clientSecret, status, amount, currency }

  var intent = await client.getPaymentIntent("pi_xxx")
  var intent = await client.confirmPaymentIntent("pi_xxx")
  var intent = await client.cancelPaymentIntent("pi_xxx")

  // Refunds
  var refund = await client.createRefund({ paymentIntent: "pi_xxx", amount: 500 })

  // Customers
  var customer = await client.createCustomer({
      email: "alice@example.com",
      name: "Alice",
      phone: "+1234567890",
      metadata: { userId: "123" }
  })
  // customer: { id, email, name, ... }

  var customer = await client.getCustomer("cus_xxx")
  var updated  = await client.updateCustomer("cus_xxx", { name: "Alice Smith" })
  var deleted  = await client.deleteCustomer("cus_xxx")
  var list     = await client.listCustomers({ limit: 10 })

  // Products and Prices
  var product = await client.createProduct({ name: "Pro Plan", description: "..." })
  var price   = await client.createPrice({
      product: "prod_xxx",
      unitAmount: 999,
      currency: "usd",
      recurring: { interval: "month" }
  })

  // Subscriptions
  var sub = await client.createSubscription({
      customer: "cus_xxx",
      items: [{ price: "price_xxx" }],
      trialPeriodDays: 14
  })
  // sub: { id, status, currentPeriodEnd, ... }

  var sub  = await client.getSubscription("sub_xxx")
  var sub  = await client.cancelSubscription("sub_xxx")

  // Invoices
  var inv  = await client.getInvoice("in_xxx")
  var list = await client.listInvoices({ customer: "cus_xxx", limit: 5 })

  // Checkout Sessions
  var session = await client.createCheckoutSession({
      lineItems: [{ price: "price_xxx", quantity: 1 }],
      mode: "subscription",
      successUrl: "https://app.com/success",
      cancelUrl: "https://app.com/cancel",
      customer: "cus_xxx"
  })
  // session: { id, url }

  // Webhooks
  var event = stripe.constructEvent(rawBody, req.headers["stripe-signature"], env.get("STRIPE_WEBHOOK_SECRET"))
  // event: { type, data: { object } }
  ```

  ---

  ## 9. Messaging and APIs

  ### 9.1 `rabbitmq` — RabbitMQ / AMQP

  Backed by [amqp091-go](https://github.com/rabbitmq/amqp091-go).

  ```lunex
  val rabbitmq = @import("std.rabbitmq")

  var conn = await rabbitmq.connect("amqp://guest:guest@localhost:5672/")

  var ch = conn.createChannel()

  // Declare a queue
  var q = ch.declareQueue("my-queue", { durable: true })

  // Declare an exchange
  ch.declareExchange("my-exchange", "direct", { durable: true })

  // Bind queue to exchange
  ch.bindQueue("my-queue", "my-exchange", "routing-key")

  // Publish a message
  ch.publish("my-exchange", "routing-key", "Hello World!", {
      contentType: "text/plain",
      persistent: true
  })

  // Publish directly to a queue (default exchange)
  ch.publish("", "my-queue", "Hello Queue!")

  // Consume messages
  var consumer = ch.consume("my-queue", fn(delivery) {
      io.log("Received:", delivery.body)
      delivery.ack()          // acknowledge
      // delivery.nack()      // negative ack (requeue)
      // delivery.reject()    // reject (no requeue)
  })

  // Queue inspection
  var info = ch.queueInspect("my-queue")
  // info: { name, messages, consumers }

  ch.close()
  conn.close()
  io.log(conn.isClosed())    // bool
  ```

  ### 9.2 `graphql` — GraphQL

  Backed by [graphql-go/graphql](https://github.com/graphql-go/graphql).

  ```lunex
  val graphql = @import("std.graphql")

  // Build schema from Lunex functions
  var schema = graphql.buildSchema({
      query: {
          hello: fn(args) { return "Hello, " + (args.input ?? "world") },
          user:  fn(args) { return users.findOne({ id: args.input }) }
      },
      mutation: {
          createUser: fn(args) {
              var data = fromJSON(args.input)
              users.insert(data)
          }
      }
  })

  // Execute a query
  var result = graphql.execute(schema, "{ hello(input: \"Lunex\") }")
  // result: { data: { hello: "Hello, Lunex" }, errors: [] }

  // Integrate with HTTP server
  val http = @import("std.http")

  val server = http.createServer(fn(req, res) {
      if req.method == "POST" and req.url == "/graphql" {
          var body = JSON.parse(req.body)
          var result = graphql.execute(schema, body.query, body.variables)
          http.json(res, result, 200)
      } else {
          http.text(res, "Not Found", 404)
      }
  })

  http.listen(server, 4000, "0.0.0.0", fn() {
      io.log("GraphQL server on port 4000")
  })
  ```

  ---

  ## 10. Document Generation

  ### 10.1 `excel` — Excel Spreadsheets

  Backed by [excelize/v2](https://github.com/xuri/excelize).

  ```lunex
  val excel = @import("std.excel")

  // Create new workbook
  var wb = excel.create()

  // Open existing file
  var wb = excel.open("report.xlsx")

  // Work with sheets
  wb.setCell("Sheet1", "A1", "Name")
  wb.setCell("Sheet1", "B1", "Age")
  wb.setCell("Sheet1", "A2", "Alice")
  wb.setCell("Sheet1", "B2", 30)

  var val = wb.getCell("Sheet1", "A1")    // "Name"

  // New sheet
  wb.addSheet("Summary")
  wb.deleteSheet("OldSheet")
  var sheets = wb.sheets()    // array of sheet names

  // Styling
  wb.setStyle("Sheet1", "A1", {
      font: { bold: true, size: 14, color: "#000000" },
      fill: { type: "pattern", color: "#FFFF00" },
      alignment: { horizontal: "center" }
  })

  // Merge cells
  wb.merge("Sheet1", "A1", "C1")

  // Column width / row height
  wb.setColWidth("Sheet1", "A", "C", 20)
  wb.setRowHeight("Sheet1", 1, 30)

  // Column name helper
  var name = excel.columnName(1)   // "A"
  var name = excel.columnName(27)  // "AA"

  // Save
  wb.save("output.xlsx")
  wb.saveAs("copy.xlsx")
  ```

  ### 10.2 `pdf` — PDF Generation

  Backed by [jung-kurt/gofpdf](https://github.com/jung-kurt/gofpdf).

  ```lunex
  val pdf = @import("std.pdf")

  var doc = pdf.create({
      orientation: "portrait",    // "portrait" | "landscape"
      unit: "mm",
      size: "A4"
  })

  // Pages
  doc.addPage()
  doc.setMargins(10, 10, 10)

  // Font
  doc.setFont("Arial", "B", 16)   // family, style (B/I/U/BI), size
  doc.setTextColor(0, 0, 0)       // R, G, B
  doc.setFillColor(255, 255, 0)
  doc.setDrawColor(0, 0, 0)
  doc.setLineWidth(0.5)

  // Text cells
  doc.cell(0, 10, "Hello PDF World!")   // width, height, text
  doc.cell(40, 10, "Column 1")
  doc.cell(40, 10, "Column 2")

  // Multi-line text
  doc.multiCell(190, 10, "Long text that wraps automatically...", "1", "L", false)

  // Move position
  doc.ln(10)              // line break of 10mm
  doc.setX(20)
  doc.setY(50)
  doc.setXY(20, 50)

  // Images
  doc.image("logo.png", 10, 10, { width: 50 })

  // Shapes
  doc.rect(10, 10, 100, 50)       // x, y, w, h
  doc.rect(10, 10, 100, 50, "F")  // filled
  doc.line(10, 10, 100, 100)

  // Links
  doc.link(10, 10, 100, 10, "https://example.com")

  // Page info
  var w, h = doc.pageSize()
  var pageNum = doc.pageCount()

  // Save
  doc.save("output.pdf")
  ```

  ---

  ## 11. Runtime Internals

  ### 11.1 Bytecode Format (.nc / .nax)

  Lunex compiles source to an intermediate format:

  - **.nc** (Lunex Compiled) — single compiled module. Contains:
    - Source file path and source text (for error reporting)
    - Bytecode instructions
    - Constant pool

  - **.nax** (Lunex Archive) — directory packed into a single file. Requires `main.lx` as entry point. All `.lx` files are compiled before packing.

  ```sh
  lunex build app.lx             # → app.nc
  lunex build app.lx -o app.nax  # → app.nax

  lunex pack ./myproject          # → myproject.nax
  lunex run app.nax               # execute archive
  lunex dis app.nc                # disassemble and inspect
  ```

  ### 11.2 REPL

  ```sh
  lunex repl
  # or just:
  lunex
  ```

  - Maintains state across lines
  - Supports multi-line input
  - History (up/down arrows)
  - Commands: `.help`, `.exit`, `.clear`, `.history`

  ### 11.3 Built-in Editor

  A vim-style TUI editor built with [bubbletea](https://github.com/charmbracelet/bubbletea) and [lipgloss](https://github.com/charmbracelet/lipgloss).

  ```sh
  lunex edit file.lx
  lunex edit          # opens file browser
  ```

  | Key | Action |
  |-----|--------|
  | `i` / `a` / `o` / `O` | Enter insert mode |
  | `Esc` | Return to normal mode |
  | `h` `j` `k` `l` | Move cursor |
  | `Ctrl+S` | Save |
  | `Ctrl+O` | Open file browser |
  | `Ctrl+H` | Help panel |
  | `Tab` | Autocomplete |
  | `:w` `:q` `:wq` | Command mode: write / quit / write+quit |

  ### 11.4 JIT Profiler

  Lunex includes a lightweight JIT profiler that tracks hot code paths. The profiler runs transparently — no configuration needed. It collects call counts per function and can be used to identify optimization candidates in future runtime versions.

  ### 11.5 Package Manager

  Lunex has a built-in package manager that downloads packages from GitHub.

  **lunex.mod** — project manifest (JSON or simple KV format):

  ```json
  {
    "name": "my-project",
    "version": "1.0.0",
    "description": "An Lunex project",
    "main": "main.lx",
    "dependencies": {
      "discordlunex": "github.com/user/discordlunex@main",
      "lunex-github": "github.com/user/lunex-github@v1.0.0"
    }
  }
  ```

  Packages are stored in `~/.lx/modules/<name>@<version>/`.

  ```sh
  lunex init myapp           # create lunex.mod
  lunex add discordlunex       # install + add to lunex.mod
  lunex install              # install all from lunex.mod
  lunex remove discordlunex    # uninstall
  lunex list                 # list installed packages
  ```

  **Writing a package:**

  Any directory with a `lunex.json` manifest and an `index.lx` entry point is a valid package. Packages export functions and values using the `export` keyword.

  ---

  ## Global Built-ins

  These are always available without `use`:

  ```lunex
  io.log(...)              // print to stdout
  typeOf(value)           // "string" | "number" | "boolean" | "array" | "object" | "function" | "null" | "undefined"
  isString(v)
  isNumber(v)
  isBool(v)
  isArray(v)
  isObject(v)
  isNull(v)
  isUndefined(v)
  isFunction(v)
  toJSON(value)           // serialize to JSON string
  fromJSON(str)           // parse JSON string
  deepClone(value)        // deep copy
  deepEqual(a, b)         // structural equality