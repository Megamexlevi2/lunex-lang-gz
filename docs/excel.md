# Excel Module

Read and write Excel/XLSX spreadsheet files with cell formatting and formula support.

**Use case:** Generate reports, import data, and work with spreadsheets programmatically.

---

## Import

```ntl
val excel = @import("std.excel")
```

---

## Available Functions

### `create()`

Executes the `create` operation with the given no arguments.

**Signature:**
```ntl
fn create()
```

### `open(path)`

Executes the `open` operation with the given parameter (path).

**Signature:**
```ntl
fn open(path)
```

### `columnName(number)`

Executes the `columnName` operation with the given parameter (number).

**Signature:**
```ntl
fn columnName(number)
```

