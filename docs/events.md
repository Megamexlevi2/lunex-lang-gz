# Events Module

Event-driven programming with publish-subscribe and event emitter patterns.

**Use case:** Implement reactive patterns and decouple components via events.

---

## Import

```lunex
val events = @import("std.events")
```

---

## Available Functions

### `create()`

Executes the `create` operation with the given no arguments.

**Signature:**
```lunex
fn create()
```

### `on(event, listener)`

Executes the `on` operation with the given parameters (event, listener).

**Signature:**
```lunex
fn on(event, listener)
```

### `off(event, listener)`

Executes the `off` operation with the given parameters (event, listener).

**Signature:**
```lunex
fn off(event, listener)
```

### `emit(event, ...args)`

Executes the `emit` operation with the given parameters (event, ...args).

**Signature:**
```lunex
fn emit(event, ...args)
```

### `once(event, listener)`

Executes the `once` operation with the given parameters (event, listener).

**Signature:**
```lunex
fn once(event, listener)
```

### `count(event)`

Executes the `count` operation with the given parameter (event).

**Signature:**
```lunex
fn count(event)
```

### `names()`

Executes the `names` operation with the given no arguments.

**Signature:**
```lunex
fn names()
```

