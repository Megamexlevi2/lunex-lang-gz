# CSV Module

Parse and generate CSV (Comma-Separated Values) files with proper escaping and formatting.

**Use case:** Import/export tabular data from spreadsheets and data sources.

---

## Import

```lunex
val csv = @import("std.csv")
```

---

## Available Functions

### `parse(content, options)`

Executes the `parse` operation with the given parameters (content, options).

**Signature:**
```lunex
fn parse(content, options)
```

### `stringify(data, options)`

Executes the `stringify` operation with the given parameters (data, options).

**Signature:**
```lunex
fn stringify(data, options)
```

### `readFile(path, options)`

Executes the `readFile` operation with the given parameters (path, options).

**Signature:**
```lunex
fn readFile(path, options)
```

### `writeFile(path, data, options)`

Executes the `writeFile` operation with the given parameters (path, data, options).

**Signature:**
```lunex
fn writeFile(path, data, options)
```

