# Testing Module

Unit testing framework with assertions, test suites, and test runners.

**Use case:** Write and run automated tests for your code.

---

## Import

```lunex
val test = @import("std.test")
```

---

## Available Functions

### `describe(name, block)`

Executes the `describe` operation with the given parameters (name, block).

**Signature:**
```lunex
fn describe(name, block)
```

### `it(name, testCallback)`

Executes the `it` operation with the given parameters (name, testCallback).

**Signature:**
```lunex
fn it(name, testCallback)
```

### `test(name, testCallback)`

Executes the `test` operation with the given parameters (name, testCallback).

**Signature:**
```lunex
fn test(name, testCallback)
```

### `skip(name, testCallback)`

Executes the `skip` operation with the given parameters (name, testCallback).

**Signature:**
```lunex
fn skip(name, testCallback)
```

### `beforeEach(hook)`

Executes the `beforeEach` operation with the given parameter (hook).

**Signature:**
```lunex
fn beforeEach(hook)
```

### `afterEach(hook)`

Executes the `afterEach` operation with the given parameter (hook).

**Signature:**
```lunex
fn afterEach(hook)
```

### `beforeAll(hook)`

Executes the `beforeAll` operation with the given parameter (hook).

**Signature:**
```lunex
fn beforeAll(hook)
```

### `afterAll(hook)`

Executes the `afterAll` operation with the given parameter (hook).

**Signature:**
```lunex
fn afterAll(hook)
```

### `assertionError(message)`

Executes the `assertionError` operation with the given parameter (message).

**Signature:**
```lunex
fn assertionError(message)
```

### `_fail(msg)`

Executes the `_fail` operation with the given parameter (msg).

**Signature:**
```lunex
fn _fail(msg)
```

### `_toStr(v)`

Executes the `_toStr` operation with the given parameter (v).

**Signature:**
```lunex
fn _toStr(v)
```

### `run()`

Executes the `run` operation with the given no arguments.

**Signature:**
```lunex
fn run()
```

