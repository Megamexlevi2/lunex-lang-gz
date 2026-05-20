# NTL v0.4.0 — Credits & Dependencies

**Created by David Dev**
GitHub: https://github.com/Megamexlevi2
© David Dev 2026

---

## Languages

| Language | Role |
|----------|------|
| **NTL** | The language itself — source files, stdlib modules, compiler library written in NTL |
| **Go** | Host compiler, parser, AST, IR builder, interpreter, CLI, package manager |
| **Zig** | Native runtime (`ntl-rt`) — VM, JIT backend, memory manager, bytecode executor |

---

## Go Dependencies

### Direct

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/BurntSushi/toml` | v1.3.2 | TOML config parsing |
| `github.com/cbroglie/mustache` | v1.4.0 | Mustache template rendering |
| `github.com/charmbracelet/bubbletea` | v0.26.6 | TUI framework (REPL, editor) |
| `github.com/charmbracelet/lipgloss` | v0.12.1 | Terminal style / layout |
| `github.com/go-sql-driver/mysql` | v1.7.1 | MySQL driver |
| `github.com/golang-jwt/jwt/v5` | v5.2.0 | JWT authentication |
| `github.com/graphql-go/graphql` | v0.8.1 | GraphQL server |
| `github.com/jackc/pgx/v5` | v5.5.0 | PostgreSQL driver |
| `github.com/jung-kurt/gofpdf` | v1.16.2 | PDF generation |
| `github.com/rabbitmq/amqp091-go` | v1.9.0 | RabbitMQ / AMQP 0-9-1 client |
| `github.com/redis/go-redis/v9` | v9.3.0 | Redis client |
| `github.com/stripe/stripe-go/v76` | v76.25.0 | Stripe payments API |
| `github.com/xuri/excelize/v2` | v2.8.0 | Excel (.xlsx) read/write |
| `github.com/yuin/goldmark` | v1.6.0 | Markdown parser |
| `golang.org/x/arch` | v0.8.0 | CPU architecture introspection (JIT) |
| `golang.org/x/oauth2` | v0.16.0 | OAuth 2.0 client |
| `golang.org/x/sys` | v0.26.0 | Low-level OS / syscall bindings |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing |

### Indirect

| Package | Version | Purpose |
|---------|---------|---------|
| `cloud.google.com/go/compute` | v1.20.1 | Google Cloud compute metadata (oauth2 dep) |
| `cloud.google.com/go/compute/metadata` | v0.2.3 | GCP metadata client |
| `github.com/aymanbagabas/go-osc52/v2` | v2.0.1 | OSC 52 clipboard escape sequences |
| `github.com/cespare/xxhash/v2` | v2.2.0 | Fast 64-bit hash (redis dep) |
| `github.com/charmbracelet/x/ansi` | v0.1.4 | ANSI escape sequence helpers |
| `github.com/charmbracelet/x/input` | v0.1.0 | Terminal input handling |
| `github.com/charmbracelet/x/term` | v0.1.1 | Terminal state management |
| `github.com/charmbracelet/x/windows` | v0.1.0 | Windows terminal support |
| `github.com/dgryski/go-rendezvous` | v0.0.0-20200823014737 | Rendezvous hashing (redis dep) |
| `github.com/erikgeiser/coninput` | v0.0.0-20211004153227 | Windows console input |
| `github.com/golang/protobuf` | v1.5.3 | Protocol Buffers (oauth2 dep) |
| `github.com/jackc/pgpassfile` | v1.0.0 | PostgreSQL .pgpass file parser |
| `github.com/jackc/pgservicefile` | v0.0.0-20221227161230 | PostgreSQL service file parser |
| `github.com/jackc/puddle/v2` | v2.2.1 | Generic connection pool (pgx dep) |
| `github.com/lucasb-eyer/go-colorful` | v1.2.0 | Color space conversions |
| `github.com/mattn/go-isatty` | v0.0.20 | TTY detection |
| `github.com/mattn/go-localereader` | v0.0.1 | Locale-aware reader |
| `github.com/mattn/go-runewidth` | v0.0.15 | Unicode rune display width |
| `github.com/mohae/deepcopy` | v0.0.0-20170929034955 | Deep copy (excelize dep) |
| `github.com/muesli/ansi` | v0.0.0-20230316100256 | ANSI sequence parsing |
| `github.com/muesli/cancelreader` | v0.2.2 | Cancellable io.Reader |
| `github.com/muesli/termenv` | v0.15.2 | Terminal environment detection |
| `github.com/richardlehane/mscfb` | v1.0.4 | Microsoft Compound File Binary (excelize dep) |
| `github.com/richardlehane/msoleps` | v1.0.3 | OLE property sets (excelize dep) |
| `github.com/rivo/uniseg` | v0.4.7 | Unicode segmentation |
| `github.com/rogpeppe/go-internal` | v1.14.1 | Internal Go tooling helpers |
| `github.com/xo/terminfo` | v0.0.0-20220910002029 | terminfo database |
| `github.com/xuri/efp` | v0.0.0-20230802181842 | Excel formula parser (excelize dep) |
| `github.com/xuri/nfp` | v0.0.0-20230819163627 | Number format parser (excelize dep) |
| `golang.org/x/crypto` | v0.18.0 | Cryptographic primitives |
| `golang.org/x/net` | v0.20.0 | Extended networking (HTTP/2, websocket) |
| `golang.org/x/sync` | v0.7.0 | Synchronization primitives |
| `golang.org/x/text` | v0.14.0 | Text processing, Unicode normalization |
| `google.golang.org/appengine` | v1.6.7 | App Engine support (oauth2 dep) |
| `google.golang.org/protobuf` | v1.31.0 | Protocol Buffers runtime |

---

## Zig Dependencies

NTL's Zig runtime (`ntl-rt`) has **zero external Zig dependencies** — it uses only the Zig standard library.

| Component | Source |
|-----------|--------|
| `std.mem` | Memory utilities |
| `std.ArrayList` / `std.ArrayListUnmanaged` | Dynamic arrays |
| `std.StringHashMap` | Hash map |
| `std.fs` | Filesystem access |
| `std.Build` | Build system API |
| `std.fmt` | Formatting |

Minimum Zig version: **0.17.0**

---

## Build Toolchain

| Tool | Version | Role |
|------|---------|------|
| Go | ≥ 1.21 | Compiles the `ntl` host binary |
| Zig | ≥ 0.17.0 | Compiles `ntl-rt` (native VM) |

Both are auto-installed by `build.sh` if not present on Linux, macOS, Android/Termux, and BSD.
