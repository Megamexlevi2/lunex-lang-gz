# Environment Module

Environment variable management with support for loading from .env files and accessing configuration values.

**Use case:** Load configuration from environment variables and .env files.

---

## Import

```ntl
val env = @import("std.env")
```

---

## Available Functions

### `get(key, fallback)`

Executes the `get` operation with the given parameters (key, fallback).

**Signature:**
```ntl
fn get(key, fallback)
```

### `set(key, value)`

Executes the `set` operation with the given parameters (key, value).

**Signature:**
```ntl
fn set(key, value)
```

### `has(key)`

Executes the `has` operation with the given parameter (key).

**Signature:**
```ntl
fn has(key)
```

### `all()`

Executes the `all` operation with the given no arguments.

**Signature:**
```ntl
fn all()
```

### `load(path)`

Executes the `load` operation with the given parameter (path).

**Signature:**
```ntl
fn load(path)
```

### `mustGet(key)`

Executes the `mustGet` operation with the given parameter (key).

**Signature:**
```ntl
fn mustGet(key)
```

### `getNumber(key, fallback)`

Executes the `getNumber` operation with the given parameters (key, fallback).

**Signature:**
```ntl
fn getNumber(key, fallback)
```

### `getBool(key, fallback)`

Executes the `getBool` operation with the given parameters (key, fallback).

**Signature:**
```ntl
fn getBool(key, fallback)
```

