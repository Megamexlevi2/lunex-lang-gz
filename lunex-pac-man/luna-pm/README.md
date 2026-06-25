# Luna — Lunex Package Manager

Luna is the official package manager for the Lunex language

## Install

```sh
lunex install luna
```

## Commands

```sh
luna install user/repo          Install from GitHub
luna install user/repo@v1.0.0  Install a specific version
luna install github.com/u/r    Full GitHub URL
luna install                   Install all deps from config.lx

luna remove  mypackage         Remove a package
luna list                      List installed packages
luna update  [pkg]             Update one or all packages
luna run     build             Run a script from config.lx
luna init    [name]            Create a new config.lx
luna search  query             Search GitHub for packages
luna version                   Show Luna version
```

## Importing packages

After `luna install user/repo`:

```lx
val pkg = @import("repo")
```

## Shebang support

```lx
#!/usr/bin/env luna
val io = @import("std.io")
io.log("Hello!")
```

```sh
chmod +x myscript.lx
./myscript.lx
```

## Running tests

```sh
lunex run tests/run-all.lx
```

## Packages location

`~/.luna/packages/`
