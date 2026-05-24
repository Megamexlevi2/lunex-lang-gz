# Database Module

Generic database abstractions and SQL utilities for database connectivity and queries.

**Use case:** Build database query abstractions and manage connections.

---

## Import

```lunex
val db = @import("std.db")
```

---

## Available Functions

### `create(name)`

Executes the `create` operation with the given parameter (name).

**Signature:**
```lunex
fn create(name)
```

### `open(name)`

Executes the `open` operation with the given parameter (name).

**Signature:**
```lunex
fn open(name)
```

### `connect(name)`

Executes the `connect` operation with the given parameter (name).

**Signature:**
```lunex
fn connect(name)
```

### `table(name)`

Executes the `table` operation with the given parameter (name).

**Signature:**
```lunex
fn table(name)
```

### `collection(name)`

Executes the `collection` operation with the given parameter (name).

**Signature:**
```lunex
fn collection(name)
```

### `drop(name)`

Executes the `drop` operation with the given parameter (name).

**Signature:**
```lunex
fn drop(name)
```

### `list()`

Executes the `list` operation with the given no arguments.

**Signature:**
```lunex
fn list()
```

