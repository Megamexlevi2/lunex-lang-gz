# Queue Module

Message queue implementation with support for both in-memory and Redis-backed queues.

**Use case:** Implement asynchronous task processing and message passing.

---

## Import

```ntl
val queue = @import("std.queue")
```

---

## Available Functions

### `create()`

Executes the `create` operation with the given no arguments.

**Signature:**
```ntl
fn create()
```

### `push(item)`

Executes the `push` operation with the given parameter (item).

**Signature:**
```ntl
fn push(item)
```

### `pop()`

Executes the `pop` operation with the given no arguments.

**Signature:**
```ntl
fn pop()
```

### `peek()`

Executes the `peek` operation with the given no arguments.

**Signature:**
```ntl
fn peek()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```ntl
fn size()
```

### `isEmpty()`

Executes the `isEmpty` operation with the given no arguments.

**Signature:**
```ntl
fn isEmpty()
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```ntl
fn clear()
```

### `toArray()`

Executes the `toArray` operation with the given no arguments.

**Signature:**
```ntl
fn toArray()
```

### `drain(handler)`

Executes the `drain` operation with the given parameter (handler).

**Signature:**
```ntl
fn drain(handler)
```

### `stats()`

Executes the `stats` operation with the given no arguments.

**Signature:**
```ntl
fn stats()
```

### `createPriority()`

Executes the `createPriority` operation with the given no arguments.

**Signature:**
```ntl
fn createPriority()
```

### `push(item, priority)`

Executes the `push` operation with the given parameters (item, priority).

**Signature:**
```ntl
fn push(item, priority)
```

### `pop()`

Executes the `pop` operation with the given no arguments.

**Signature:**
```ntl
fn pop()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```ntl
fn size()
```

### `isEmpty()`

Executes the `isEmpty` operation with the given no arguments.

**Signature:**
```ntl
fn isEmpty()
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```ntl
fn clear()
```

### `createDeque()`

Executes the `createDeque` operation with the given no arguments.

**Signature:**
```ntl
fn createDeque()
```

### `pushFront(item)`

Executes the `pushFront` operation with the given parameter (item).

**Signature:**
```ntl
fn pushFront(item)
```

### `pushBack(item)`

Executes the `pushBack` operation with the given parameter (item).

**Signature:**
```ntl
fn pushBack(item)
```

### `popFront()`

Executes the `popFront` operation with the given no arguments.

**Signature:**
```ntl
fn popFront()
```

### `popBack()`

Executes the `popBack` operation with the given no arguments.

**Signature:**
```ntl
fn popBack()
```

### `peekFront()`

Executes the `peekFront` operation with the given no arguments.

**Signature:**
```ntl
fn peekFront()
```

### `peekBack()`

Executes the `peekBack` operation with the given no arguments.

**Signature:**
```ntl
fn peekBack()
```

### `size()`

Executes the `size` operation with the given no arguments.

**Signature:**
```ntl
fn size()
```

### `isEmpty()`

Executes the `isEmpty` operation with the given no arguments.

**Signature:**
```ntl
fn isEmpty()
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```ntl
fn clear()
```

