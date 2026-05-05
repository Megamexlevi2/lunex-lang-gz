# NTL Command Reference

NTL v2.0 — a fast, expressive scripting language with bytecode compilation.

## Running Code

```
ntl <file.ntl>             Run an NTL source file directly
ntl run <file>             Run .ntl, .nc, or .nax file
ntl repl                   Start interactive REPL
```

## Building Bytecode

```
ntl build <file.ntl>       Compile source to .nc bytecode
ntl build <file.ntl> -o output.nc
```

The `.nc` format is NTL Compiled bytecode. It is a binary format that is scrambled
and cannot be read as plain text. It runs with `ntl run` exactly like source.

Example:
```
ntl build hello.ntl          # produces hello.nc
ntl run hello.nc             # runs the compiled file
```

## Packing Archives

```
ntl pack <directory>         Pack all .nc files into a .nax archive
ntl pack <directory> -o output.nax
```

The `.nax` format is a NTL Archive. It bundles multiple `.nc` files into a single
distributable binary package. The archive is fully scrambled and appears as binary
machine code — the source code cannot be extracted.

The entry point is determined automatically (looks for `main.nc` or `index.nc`,
falls back to the first file).

Example:
```
ntl build main.ntl -o dist/main.nc
ntl build utils.ntl -o dist/utils.nc
ntl pack dist -o app.nax
ntl run app.nax
```

## Project Management

```
ntl init [name]              Initialize a new project (creates ntl.mod)
ntl install                  Install packages from ntl.mod
ntl install [pkg]            Install a specific package
ntl add <pkg>                Add and install a package (updates ntl.mod)
ntl remove <pkg>             Remove an installed package
ntl list                     List installed packages
```

## Code Quality

```
ntl check <file.ntl>         Check for syntax/parse errors without running
ntl fmt <file.ntl>           Format NTL source code in-place
ntl dis <file.nc>            Show bytecode module info (non-destructive)
```

## Version

```
ntl version                  Show version
ntl --version
ntl -v
```

## File Types

| Extension | Description                                    |
|-----------|------------------------------------------------|
| `.ntl`    | NTL source code (human readable)               |
| `.nc`     | NTL Compiled — bytecode archive (binary)       |
| `.nax`    | NTL Archive — bundled bytecode (binary)        |
| `ntl.mod` | Project manifest (package dependencies)        |

## The naxer Editor

`naxer` is NTL's built-in terminal editor.

```
naxer                        Open editor (new file)
naxer <file>                 Open a file in the editor
```

### naxer Key Bindings

**Normal mode:**

| Key          | Action                    |
|--------------|---------------------------|
| `i`          | Enter insert mode         |
| `a`          | Insert after cursor       |
| `o` / `O`   | New line below / above    |
| `d`          | Delete current line       |
| `x`          | Delete character          |
| `h j k l`   | Move left/down/up/right   |
| `0` / `$`   | Start / end of line       |
| `g` / `G`   | First / last line         |
| `w` / `b`   | Next / prev word          |
| `Ctrl+D/U`  | Half page down / up       |
| `Ctrl+S`    | Save file                 |
| `Ctrl+O`    | Open file browser         |
| `Ctrl+H`    | Toggle help               |
| `:`          | Enter command mode        |

**Insert mode:**

| Key          | Action                    |
|--------------|---------------------------|
| `Tab`        | Accept autocomplete       |
| `Up/Down`   | Navigate autocomplete     |
| `Esc`        | Return to Normal mode     |

**Commands (type `:` then command):**

| Command      | Action                    |
|--------------|---------------------------|
| `w`          | Save                      |
| `q`          | Quit                      |
| `wq`         | Save and quit             |
| `w filename` | Save as new file          |
| `e filename` | Open file                 |
| `new`        | New empty file            |
| `help`       | Show help                 |

## Stdlib Modules

Use with `use <module>` at the top of your file.

| Module     | Description                              |
|------------|------------------------------------------|
| `io`       | Console output with colors and tables    |
| `fs`       | File system read/write                   |
| `http`     | HTTP client and server                   |
| `crypto`   | Hashing, JWT, AES-GCM encryption         |
| `db`       | In-memory database with schema           |
| `env`      | Environment variables, .env loading      |
| `events`   | Event emitter                            |
| `cache`    | TTL cache with LRU eviction              |
| `logger`   | Structured logging                       |
| `queue`    | Task queues with priority scheduling     |
| `validate` | Schema validation and format checking    |
| `ws`       | WebSocket server and client              |
| `mail`     | SMTP email with HTML                     |
| `ai`       | AI/LLM client (OpenAI compatible)        |
| `utils`    | Array, string, math utilities            |
| `test`     | Unit testing framework                   |
| `type`     | Type system utilities                    |
