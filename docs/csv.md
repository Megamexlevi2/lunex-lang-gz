# CSV Module

Parse and generate CSV (Comma-Separated Values) files with proper escaping and formatting.

**Use case:** Import/export tabular data from spreadsheets and data sources.

---

## Import

```ntl
val csv = @import("std.csv")
```

---

## Available Functions

### `parse(content, options)`

Executes the `parse` operation with the given parameters (content, options).

**Signature:**
```ntl
fn parse(content, options)
```

### `stringify(data, options)`

Executes the `stringify` operation with the given parameters (data, options).

**Signature:**
```ntl
fn stringify(data, options)
```

### `readFile(path, options)`

Executes the `readFile` operation with the given parameters (path, options).

**Signature:**
```ntl
fn readFile(path, options)
```

### `writeFile(path, data, options)`

Executes the `writeFile` operation with the given parameters (path, data, options).

**Signature:**
```ntl
fn writeFile(path, data, options)
```

