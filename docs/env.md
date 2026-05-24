# Environment Module

Environment variable management with support for loading from .env files and accessing configuration values.

**Use case:** Load configuration from environment variables and .env files.

---

## Import

```lunex
val env = @import("std.env")
```

---

## Available Functions

### `get(key, fallback)`

Executes the `get` operation with the given parameters (key, fallback).

**Signature:**
```lunex
fn get(key, fallback)
```

### `set(key, value)`

Executes the `set` operation with the given parameters (key, value).

**Signature:**
```lunex
fn set(key, value)
```

### `has(key)`

Executes the `has` operation with the given parameter (key).

**Signature:**
```lunex
fn has(key)
```

### `all()`

Executes the `all` operation with the given no arguments.

**Signature:**
```lunex
fn all()
```

### `load(path)`

Executes the `load` operation with the given parameter (path).

**Signature:**
```lunex
fn load(path)
```

### `mustGet(key)`

Executes the `mustGet` operation with the given parameter (key).

**Signature:**
```lunex
fn mustGet(key)
```

### `getNumber(key, fallback)`

Executes the `getNumber` operation with the given parameters (key, fallback).

**Signature:**
```lunex
fn getNumber(key, fallback)
```

### `getBool(key, fallback)`

Executes the `getBool` operation with the given parameters (key, fallback).

**Signature:**
```lunex
fn getBool(key, fallback)
```

