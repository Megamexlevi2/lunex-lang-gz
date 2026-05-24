# Lunex Module System

  Lunex lets you create, publish, and install modules (packages) directly from GitHub.

  ## Installing a Module

  ```sh
  lunex add github.com/user/module
  ```

  You can also install from a subfolder inside a repository:

  ```sh
  lunex add github.com/user/module/subfolder
  lunex add github.com/user/mytools/http
  lunex add github.com/user/mytools/xml
  ```

  This is useful when a single repository contains multiple independent modules.

  Or declare it in `lunex.mod`:

  ```
  [dependencies]
  mymodule = "github.com/user/module@main"
  submod   = "github.com/user/repo/subfolder@v1.2.0"
  ```

  ## Creating a Module

  ### Minimal Structure

  ```
  mymodule/
    index.lx     <- entry point, must export __module__
    lunex.json      <- package metadata
  ```

  ### lunex.json

  ```json
  {
    "name": "mymodule",
    "version": "1.0.0",
    "description": "A short description",
    "author": "Your Name",
    "license": "MIT",
    "main": "index.lx"
  }
  ```

  ### index.lx

  ```lunex
  fn greet(name) {
    "Hello, " + name + "!"
  }

  fn add(a, b) {
    a + b
  }

  val __module__ = {
    greet: greet,
    add:   add
  }
  ```

  The `__module__` object is the public API of your module.
  Everything else defined in the file is private.

  ## Using a Module

  The imported name is the last segment of the install path:

  ```lunex
  val mymodule = @import("mymodule")
  io.log(mymodule.greet("World"))
  io.log(mymodule.add(1, 2))
  ```

  For subfolders:

  ```lunex
  val http = @import("std.http")
  val xml = @import("std.xml")
  ```

  (installed from `github.com/user/repo/http` and `github.com/user/repo/xml`)

  ## Module with Dependencies

  Modules can use native modules or other installed modules:

  ```lunex
  val http = @import("std.http")

  fn fetch(url) {
    val res = http.get(url)
    JSON.parse(res.body)
  }

  val __module__ = { fetch: fetch }
  ```

  ## Privileged Author Modules

  If the module file contains `# author: David Dev` in the first 15 lines,
  all native modules are injected automatically — no `use native` needed:

  ```lunex
  # author: David Dev
  # mymodule v1.0.0

  fn readConfig(path) {
    val raw = fs.readFile(path)
    JSON.parse(raw)
  }

  val __module__ = { readConfig: readConfig }
  ```

  Available injected names: `fs`, `http`, `crypto`, `db`, `io`, `utils`,
  `mail`, `ws`, `validate`, `xml`, `env`, `test`, `ai`

  ## Multi-Module Repository

  A single GitHub repository can host many modules in subfolders:

  ```
  myrepo/
    math/
      index.lx
      lunex.json
    strings/
      index.lx
      lunex.json
    http/
      index.lx
      lunex.json
  ```

  Install individually:

  ```sh
  lunex add github.com/user/myrepo/math
  lunex add github.com/user/myrepo/strings
  lunex add github.com/user/myrepo/http
  ```

  ## Publishing

  Push your module to GitHub. That is all.
  Use tags for versioned releases:

  ```sh
  git tag v1.0.0
  git push origin v1.0.0
  ```

  Install a specific version:

  ```sh
  lunex add github.com/user/mymodule@v1.0.0
  ```
  
---

## Built-in modules reference

| Module | Import | Description |
|---|---|---|
| `lunex:ai` | `@import("std.ai")` | OpenAI / LLM API client |
| `lunex:alloc` | `@import("std.alloc")` | Low-level memory buffers and binary I/O |
| `lunex:crypto` | `@import("std.crypto")` | Hashing, HMAC, AES, RSA, UUID |
| `lunex:csv` | `@import("std.csv")` | CSV parsing and serialization |
| `lunex:db` | `@import("std.db")` | SQLite / embedded database |
| `lunex:env` | `@import("std.env")` | Environment variables |
| `lunex:excel` | `@import("std.excel")` | Excel (.xlsx) read/write |
| `lunex:fs` | `@import("std.fs")` | File system operations |
| `lunex:graphql` | `@import("std.graphql")` | GraphQL schema and executor |
| `lunex:http` | `@import("std.http")` | HTTP server and client |
| `lunex:io` | `@import("std.io")` | Console I/O and logging |
| `lunex:jwt` | `@import("std.jwt")` | JSON Web Token sign/verify |
| `lunex:mail` | `@import("std.mail")` | SMTP email sending |
| `lunex:markdown` | `@import("std.markdown")` | Markdown to HTML |
| `lunex:mustache` | `@import("std.mustache")` | Mustache template rendering |
| `lunex:mysql` | `@import("std.mysql")` | MySQL/MariaDB client |
| `lunex:oauth2` | `@import("std.oauth2")` | OAuth 2.0 (Google, GitHub, custom) |
| `lunex:os` | `@import("std.os")` | OS process, args, signals |
| `lunex:pdf` | `@import("std.pdf")` | PDF document generation |
| `lunex:postgres` | `@import("std.postgres")` | PostgreSQL client |
| `lunex:rabbitmq` | `@import("std.rabbitmq")` | RabbitMQ / AMQP message queue |
| `lunex:redis` | `@import("std.redis")` | Redis client |
| `lunex:stripe` | `@import("std.stripe")` | Stripe payments |
| `lunex:test` | `@import("std.test")` | Unit testing |
| `lunex:toml` | `@import("std.toml")` | TOML parsing and serialization |
| `lunex:utils` | `@import("std.utils")` | Math, string, array utilities |
| `lunex:validate` | `@import("std.validate")` | Input validation |
| `lunex:ws` | `@import("std.ws")` | WebSocket server |
| `lunex:xml` | `@import("std.xml")` | XML parsing and building |
| `lunex:yaml` | `@import("std.yaml")` | YAML parsing and serialization |
