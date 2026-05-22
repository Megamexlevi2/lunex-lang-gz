# PostgreSQL Module

PostgreSQL database client with connection pooling and advanced query features.

**Use case:** Connect to and query PostgreSQL databases.

---

## Import

```ntl
val postgres = @import("std.postgres")
```

---

## Available Functions

### `connect(dsn)`

Executes the `connect` operation with the given parameter (dsn).

**Signature:**
```ntl
fn connect(dsn)
```

