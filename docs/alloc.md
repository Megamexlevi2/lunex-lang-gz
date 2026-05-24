# Memory Allocation Module

Low-level memory management and allocation utilities for fine-grained control over memory usage.

**Use case:** Advanced users: optimize memory for performance-critical applications.

---

## Import

```lunex
val alloc = @import("std.alloc")
```

---

## Available Functions

### `buffer(size)`

Executes the `buffer` operation with the given parameter (size).

**Signature:**
```lunex
fn buffer(size)
```

### `write(offset, ...bytes)`

Executes the `write` operation with the given parameters (offset, ...bytes).

**Signature:**
```lunex
fn write(offset, ...bytes)
```

### `writeString(offset, str)`

Executes the `writeString` operation with the given parameters (offset, str).

**Signature:**
```lunex
fn writeString(offset, str)
```

### `readByte(offset)`

Executes the `readByte` operation with the given parameter (offset).

**Signature:**
```lunex
fn readByte(offset)
```

### `readString(offset, length)`

Executes the `readString` operation with the given parameters (offset, length).

**Signature:**
```lunex
fn readString(offset, length)
```

### `readBytes(offset, length)`

Executes the `readBytes` operation with the given parameters (offset, length).

**Signature:**
```lunex
fn readBytes(offset, length)
```

### `fill(value)`

Executes the `fill` operation with the given parameter (value).

**Signature:**
```lunex
fn fill(value)
```

### `slice(start, end)`

Executes the `slice` operation with the given parameters (start, end).

**Signature:**
```lunex
fn slice(start, end)
```

### `copy(src, destOffset, srcOffset, length)`

Executes the `copy` operation with the given parameters (src, destOffset, srcOffset, length).

**Signature:**
```lunex
fn copy(src, destOffset, srcOffset, length)
```

### `byteSize()`

Executes the `byteSize` operation with the given no arguments.

**Signature:**
```lunex
fn byteSize()
```

### `raw()`

Executes the `raw` operation with the given no arguments.

**Signature:**
```lunex
fn raw()
```

### `free()`

Executes the `free` operation with the given no arguments.

**Signature:**
```lunex
fn free()
```

### `arena(capacity)`

Executes the `arena` operation with the given parameter (capacity).

**Signature:**
```lunex
fn arena(capacity)
```

### `alloc(size)`

Executes the `alloc` operation with the given parameter (size).

**Signature:**
```lunex
fn alloc(size)
```

### `allocZero(size)`

Executes the `allocZero` operation with the given parameter (size).

**Signature:**
```lunex
fn allocZero(size)
```

### `reset()`

Executes the `reset` operation with the given no arguments.

**Signature:**
```lunex
fn reset()
```

### `destroy()`

Executes the `destroy` operation with the given no arguments.

**Signature:**
```lunex
fn destroy()
```

### `used()`

Executes the `used` operation with the given no arguments.

**Signature:**
```lunex
fn used()
```

### `remaining()`

Executes the `remaining` operation with the given no arguments.

**Signature:**
```lunex
fn remaining()
```

### `fromString(str)`

Executes the `fromString` operation with the given parameter (str).

**Signature:**
```lunex
fn fromString(str)
```

### `fromBytes(arr)`

Executes the `fromBytes` operation with the given parameter (arr).

**Signature:**
```lunex
fn fromBytes(arr)
```

### `pageSize()`

Executes the `pageSize` operation with the given no arguments.

**Signature:**
```lunex
fn pageSize()
```

### `alignTo(size, align)`

Executes the `alignTo` operation with the given parameters (size, align).

**Signature:**
```lunex
fn alignTo(size, align)
```

### `sizeof(val)`

Executes the `sizeof` operation with the given parameter (val).

**Signature:**
```lunex
fn sizeof(val)
```

### `concat(...bufs)`

Executes the `concat` operation with the given parameter (...bufs).

**Signature:**
```lunex
fn concat(...bufs)
```

### `stats()`

Executes the `stats` operation with the given no arguments.

**Signature:**
```lunex
fn stats()
```

