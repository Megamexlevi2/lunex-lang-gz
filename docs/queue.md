# Queue Module

Message queue implementation with support for both in-memory and Redis-backed queues.

**Use case:** Implement asynchronous task processing and message passing.

---

## Import

```lunex
val queue = @import("std.queue")
```

---

## Available Functions

### `create()`

Executes the `create` operation with the given no arguments.

**Signature:**
```lunex
fn create()
```

### `push(item)`

Executes the `push` operation with the given parameter (item).

**Signature:**
```lunex
fn push(item)
```

### `pop()`

Executes the `pop` operation with the given no arguments.

**Signature:**
```lunex
fn pop()
```

### `peek()`

Executes the `peek` operation with the given no arguments.

**Signature:**
```lunex
fn peek()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```lunex
fn size()
```

### `isEmpty()`

Executes the `isEmpty` operation with the given no arguments.

**Signature:**
```lunex
fn isEmpty()
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```lunex
fn clear()
```

### `toArray()`

Executes the `toArray` operation with the given no arguments.

**Signature:**
```lunex
fn toArray()
```

### `drain(handler)`

Executes the `drain` operation with the given parameter (handler).

**Signature:**
```lunex
fn drain(handler)
```

### `stats()`

Executes the `stats` operation with the given no arguments.

**Signature:**
```lunex
fn stats()
```

### `createPriority()`

Executes the `createPriority` operation with the given no arguments.

**Signature:**
```lunex
fn createPriority()
```

### `push(item, priority)`

Executes the `push` operation with the given parameters (item, priority).

**Signature:**
```lunex
fn push(item, priority)
```

### `pop()`

Executes the `pop` operation with the given no arguments.

**Signature:**
```lunex
fn pop()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```lunex
fn size()
```

### `isEmpty()`

Executes the `isEmpty` operation with the given no arguments.

**Signature:**
```lunex
fn isEmpty()
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```lunex
fn clear()
```

### `createDeque()`

Executes the `createDeque` operation with the given no arguments.

**Signature:**
```lunex
fn createDeque()
```

### `pushFront(item)`

Executes the `pushFront` operation with the given parameter (item).

**Signature:**
```lunex
fn pushFront(item)
```

### `pushBack(item)`

Executes the `pushBack` operation with the given parameter (item).

**Signature:**
```lunex
fn pushBack(item)
```

### `popFront()`

Executes the `popFront` operation with the given no arguments.

**Signature:**
```lunex
fn popFront()
```

### `popBack()`

Executes the `popBack` operation with the given no arguments.

**Signature:**
```lunex
fn popBack()
```

### `peekFront()`

Executes the `peekFront` operation with the given no arguments.

**Signature:**
```lunex
fn peekFront()
```

### `peekBack()`

Executes the `peekBack` operation with the given no arguments.

**Signature:**
```lunex
fn peekBack()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```lunex
fn size()
```

### `isEmpty()`

Executes the `isEmpty` operation with the given no arguments.

**Signature:**
```lunex
fn isEmpty()
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```lunex
fn clear()
```

