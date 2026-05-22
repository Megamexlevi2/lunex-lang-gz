# Memory Allocation Module

Low-level memory management and allocation utilities for fine-grained control over memory usage.

**Use case:** Advanced users: optimize memory for performance-critical applications.

---

## Import

```ntl
val alloc = @import("std.alloc")
```

---

## Available Functions

### `buffer(size)`

Executes the `buffer` operation with the given parameter (size).

**Signature:**
```ntl
fn buffer(size)
```

### `write(offset, ...bytes)`

Executes the `write` operation with the given parameters (offset, ...bytes).

**Signature:**
```ntl
fn write(offset, ...bytes)
```

### `writeString(offset, str)`

Executes the `writeString` operation with the given parameters (offset, str).

**Signature:**
```ntl
fn writeString(offset, str)
```

### `readByte(offset)`

Executes the `readByte` operation with the given parameter (offset).

**Signature:**
```ntl
fn readByte(offset)
```

### `readString(offset, length)`

Executes the `readString` operation with the given parameters (offset, length).

**Signature:**
```ntl
fn readString(offset, length)
```

### `readBytes(offset, length)`

Executes the `readBytes` operation with the given parameters (offset, length).

**Signature:**
```ntl
fn readBytes(offset, length)
```

### `fill(value)`

Executes the `fill` operation with the given parameter (value).

**Signature:**
```ntl
fn fill(value)
```

### `slice(start, end)`

Executes the `slice` operation with the given parameters (start, end).

**Signature:**
```ntl
fn slice(start, end)
```

### `copy(src, destOffset, srcOffset, length)`

Executes the `copy` operation with the given parameters (src, destOffset, srcOffset, length).

**Signature:**
```ntl
fn copy(src, destOffset, srcOffset, length)
```

### `byteSize()`

Executes the `byteSize` operation with the given no arguments.

**Signature:**
```ntl
fn byteSize()
```

### `raw()`

Executes the `raw` operation with the given no arguments.

**Signature:**
```ntl
fn raw()
```

### `free()`

Executes the `free` operation with the given no arguments.

**Signature:**
```ntl
fn free()
```

### `arena(capacity)`

Executes the `arena` operation with the given parameter (capacity).

**Signature:**
```ntl
fn arena(capacity)
```

### `alloc(size)`

Executes the `alloc` operation with the given parameter (size).

**Signature:**
```ntl
fn alloc(size)
```

### `allocZero(size)`

Executes the `allocZero` operation with the given parameter (size).

**Signature:**
```ntl
fn allocZero(size)
```

### `reset()`

Executes the `reset` operation with the given no arguments.

**Signature:**
```ntl
fn reset()
```

### `destroy()`

Executes the `destroy` operation with the given no arguments.

**Signature:**
```ntl
fn destroy()
```

### `used()`

Executes the `used` operation with the given no arguments.

**Signature:**
```ntl
fn used()
```

### `remaining()`

Executes the `remaining` operation with the given no arguments.

**Signature:**
```ntl
fn remaining()
```

### `fromString(str)`

Executes the `fromString` operation with the given parameter (str).

**Signature:**
```ntl
fn fromString(str)
```

### `fromBytes(arr)`

Executes the `fromBytes` operation with the given parameter (arr).

**Signature:**
```ntl
fn fromBytes(arr)
```

### `pageSize()`

Executes the `pageSize` operation with the given no arguments.

**Signature:**
```ntl
fn pageSize()
```

### `alignTo(size, align)`

Executes the `alignTo` operation with the given parameters (size, align).

**Signature:**
```ntl
fn alignTo(size, align)
```

### `sizeof(val)`

Executes the `sizeof` operation with the given parameter (val).

**Signature:**
```ntl
fn sizeof(val)
```

### `concat(...bufs)`

Executes the `concat` operation with the given parameter (...bufs).

**Signature:**
```ntl
fn concat(...bufs)
```

### `stats()`

Executes the `stats` operation with the given no arguments.

**Signature:**
```ntl
fn stats()
```

