# YAML Module

YAML configuration file parsing and serialization.

**Use case:** Parse and write YAML configuration files.

---

## Import

```ntl
val yaml = @import("std.yaml")
```

---

## Available Functions

### `parse(content)`

Executes the `parse` operation with the given parameter (content).

**Signature:**
```ntl
fn parse(content)
```

### `stringify(data)`

Executes the `stringify` operation with the given parameter (data).

**Signature:**
```ntl
fn stringify(data)
```

### `readFile(path)`

Executes the `readFile` operation with the given parameter (path).

**Signature:**
```ntl
fn readFile(path)
```

### `writeFile(path, data)`

Executes the `writeFile` operation with the given parameters (path, data).

**Signature:**
```ntl
fn writeFile(path, data)
```

