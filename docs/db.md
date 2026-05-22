# Database Module

Generic database abstractions and SQL utilities for database connectivity and queries.

**Use case:** Build database query abstractions and manage connections.

---

## Import

```ntl
val db = @import("std.db")
```

---

## Available Functions

### `create(name)`

Executes the `create` operation with the given parameter (name).

**Signature:**
```ntl
fn create(name)
```

### `open(name)`

Executes the `open` operation with the given parameter (name).

**Signature:**
```ntl
fn open(name)
```

### `connect(name)`

Executes the `connect` operation with the given parameter (name).

**Signature:**
```ntl
fn connect(name)
```

### `table(name)`

Executes the `table` operation with the given parameter (name).

**Signature:**
```ntl
fn table(name)
```

### `collection(name)`

Executes the `collection` operation with the given parameter (name).

**Signature:**
```ntl
fn collection(name)
```

### `drop(name)`

Executes the `drop` operation with the given parameter (name).

**Signature:**
```ntl
fn drop(name)
```

### `list()`

Executes the `list` operation with the given no arguments.

**Signature:**
```ntl
fn list()
```

