# NTL Module System

  NTL lets you create, publish, and install modules (packages) directly from GitHub.

  ## Installing a Module

  ```sh
  ntl add github.com/user/module
  ```

  You can also install from a subfolder inside a repository:

  ```sh
  ntl add github.com/user/module/subfolder
  ntl add github.com/user/mytools/http
  ntl add github.com/user/mytools/xml
  ```

  This is useful when a single repository contains multiple independent modules.

  Or declare it in `ntl.mod`:

  ```
  [dependencies]
  mymodule = "github.com/user/module@main"
  submod   = "github.com/user/repo/subfolder@v1.2.0"
  ```

  ## Creating a Module

  ### Minimal Structure

  ```
  mymodule/
    index.ntl     <- entry point, must export __module__
    ntl.json      <- package metadata
  ```

  ### ntl.json

  ```json
  {
    "name": "mymodule",
    "version": "1.0.0",
    "description": "A short description",
    "author": "Your Name",
    "license": "MIT",
    "main": "index.ntl"
  }
  ```

  ### index.ntl

  ```ntl
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

  ```ntl
  val mymodule = @import("mymodule")
  io.log(mymodule.greet("World"))
  io.log(mymodule.add(1, 2))
  ```

  For subfolders:

  ```ntl
  val http = @import("std.http")
  val xml = @import("std.xml")
  ```

  (installed from `github.com/user/repo/http` and `github.com/user/repo/xml`)

  ## Module with Dependencies

  Modules can use native modules or other installed modules:

  ```ntl
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

  ```ntl
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
      index.ntl
      ntl.json
    strings/
      index.ntl
      ntl.json
    http/
      index.ntl
      ntl.json
  ```

  Install individually:

  ```sh
  ntl add github.com/user/myrepo/math
  ntl add github.com/user/myrepo/strings
  ntl add github.com/user/myrepo/http
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
  ntl add github.com/user/mymodule@v1.0.0
  ```
  
---

## Built-in modules reference

| Module | Import | Description |
|---|---|---|
| `ntl:ai` | `@import("std.ai")` | OpenAI / LLM API client |
| `ntl:alloc` | `@import("std.alloc")` | Low-level memory buffers and binary I/O |
| `ntl:crypto` | `@import("std.crypto")` | Hashing, HMAC, AES, RSA, UUID |
| `ntl:csv` | `@import("std.csv")` | CSV parsing and serialization |
| `ntl:db` | `@import("std.db")` | SQLite / embedded database |
| `ntl:env` | `@import("std.env")` | Environment variables |
| `ntl:excel` | `@import("std.excel")` | Excel (.xlsx) read/write |
| `ntl:fs` | `@import("std.fs")` | File system operations |
| `ntl:graphql` | `@import("std.graphql")` | GraphQL schema and executor |
| `ntl:http` | `@import("std.http")` | HTTP server and client |
| `ntl:io` | `@import("std.io")` | Console I/O and logging |
| `ntl:jwt` | `@import("std.jwt")` | JSON Web Token sign/verify |
| `ntl:mail` | `@import("std.mail")` | SMTP email sending |
| `ntl:markdown` | `@import("std.markdown")` | Markdown to HTML |
| `ntl:mustache` | `@import("std.mustache")` | Mustache template rendering |
| `ntl:mysql` | `@import("std.mysql")` | MySQL/MariaDB client |
| `ntl:oauth2` | `@import("std.oauth2")` | OAuth 2.0 (Google, GitHub, custom) |
| `ntl:os` | `@import("std.os")` | OS process, args, signals |
| `ntl:pdf` | `@import("std.pdf")` | PDF document generation |
| `ntl:postgres` | `@import("std.postgres")` | PostgreSQL client |
| `ntl:rabbitmq` | `@import("std.rabbitmq")` | RabbitMQ / AMQP message queue |
| `ntl:redis` | `@import("std.redis")` | Redis client |
| `ntl:stripe` | `@import("std.stripe")` | Stripe payments |
| `ntl:test` | `@import("std.test")` | Unit testing |
| `ntl:toml` | `@import("std.toml")` | TOML parsing and serialization |
| `ntl:utils` | `@import("std.utils")` | Math, string, array utilities |
| `ntl:validate` | `@import("std.validate")` | Input validation |
| `ntl:ws` | `@import("std.ws")` | WebSocket server |
| `ntl:xml` | `@import("std.xml")` | XML parsing and building |
| `ntl:yaml` | `@import("std.yaml")` | YAML parsing and serialization |
