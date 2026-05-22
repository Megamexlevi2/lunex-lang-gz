# Cache Module

In-memory caching with configurable TTL (Time To Live) for storing and retrieving frequently accessed data.

**Use case:** Improve performance by caching expensive computations or API responses.

---

## Import

```ntl
val cache = @import("std.cache")
```

---

## Available Functions

### `create(maxSize, ttl)`

Executes the `create` operation with the given parameters (maxSize, ttl).

**Signature:**
```ntl
fn create(maxSize, ttl)
```

### `put(key, value)`

Executes the `put` operation with the given parameters (key, value).

**Signature:**
```ntl
fn put(key, value)
```

### `lookup(key)`

Executes the `lookup` operation with the given parameter (key).

**Signature:**
```ntl
fn lookup(key)
```

### `has(key)`

Executes the `has` operation with the given parameter (key).

**Signature:**
```ntl
fn has(key)
```

### `del(key)`

Executes the `del` operation with the given parameter (key).

**Signature:**
```ntl
fn del(key)
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```ntl
fn clear()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```ntl
fn size()
```

### `keys()`

Executes the `keys` operation with the given no arguments.

**Signature:**
```ntl
fn keys()
```

### `stats()`

Executes the `stats` operation with the given no arguments.

**Signature:**
```ntl
fn stats()
```

