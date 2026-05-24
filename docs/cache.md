# Cache Module

In-memory caching with configurable TTL (Time To Live) for storing and retrieving frequently accessed data.

**Use case:** Improve performance by caching expensive computations or API responses.

---

## Import

```lunex
val cache = @import("std.cache")
```

---

## Available Functions

### `create(maxSize, ttl)`

Creates a new cache instance with optional max size and TTL (in milliseconds).

**Signature:**
```lunex
fn create(maxSize, ttl)
```

### `set(key, value)`

Stores a value by key. Evicts the oldest entry if the cache is full.

**Signature:**
```lunex
fn set(key, value)
```

### `get(key)`

Retrieves a value by key. Returns `null` if not found or expired.

**Signature:**
```lunex
fn get(key)
```

### `has(key)`

Returns `true` if the key exists and has not expired.

**Signature:**
```lunex
fn has(key)
```

### `del(key)`

Removes a key from the cache.

**Signature:**
```lunex
fn del(key)
```

### `clear()`

Removes all entries from the cache.

**Signature:**
```lunex
fn clear()
```

### `size()`

Returns the number of active (non-expired) entries.

**Signature:**
```lunex
fn size()
```

### `keys()`

Returns an array of all active keys.

**Signature:**
```lunex
fn keys()
```

### `stats()`

Returns cache statistics: `{ hits, misses, size, hitRate }`.

**Signature:**
```lunex
fn stats()
```

---

## Example

```lunex
val cache = @import("std.cache")

val c = cache.create(500, 60000)  // max 500 entries, 60s TTL

c.set("user:1", { name: "Alice" })

val user = c.get("user:1")        // { name: "Alice" }
val miss = c.get("user:99")       // null

io.log(c.has("user:1"))           // true
io.log(c.size())                  // 1

val s = c.stats()
io.log(s.hits, s.misses, s.hitRate)

c.del("user:1")
c.clear()
```
