# Build System Module

Build system utilities for project compilation, dependency management, and artifact generation.

**Use case:** Manage project builds and compilation workflows programmatically.

---

## Import

```lunex
val build = @import("std.build")
```

---

## Available Functions

### `hostTarget()`

Executes the `hostTarget` operation with the given no arguments.

**Signature:**
```lunex
fn hostTarget()
```

### `allTargets()`

Executes the `allTargets` operation with the given no arguments.

**Signature:**
```lunex
fn allTargets()
```

### `target(os, arch)`

Executes the `target` operation with the given parameters (os, arch).

**Signature:**
```lunex
fn target(os, arch)
```

### `executable(opts)`

Executes the `executable` operation with the given parameter (opts).

**Signature:**
```lunex
fn executable(opts)
```

### `setTarget(t)`

Executes the `setTarget` operation with the given parameter (t).

**Signature:**
```lunex
fn setTarget(t)
```

### `optimize(mode)`

Executes the `optimize` operation with the given parameter (mode).

**Signature:**
```lunex
fn optimize(mode)
```

### `output(path)`

Executes the `output` operation with the given parameter (path).

**Signature:**
```lunex
fn output(path)
```

### `install()`

Executes the `install` operation with the given no arguments.

**Signature:**
```lunex
fn install()
```

