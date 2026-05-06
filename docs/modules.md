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
    return "Hello, " + name + "!"
  }

  fn add(a, b) {
    return a + b
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
  use "mymodule"
  io.log(mymodule.greet("World"))
  io.log(mymodule.add(1, 2))
  ```

  For subfolders:

  ```ntl
  use "http"
  use "xml"
  ```

  (installed from `github.com/user/repo/http` and `github.com/user/repo/xml`)

  ## Module with Dependencies

  Modules can use native modules or other installed modules:

  ```ntl
  use native

  fn fetch(url) {
    val res = native.http.get(url, {})
    return JSON.parse(res.body)
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
    return JSON.parse(raw)
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
  