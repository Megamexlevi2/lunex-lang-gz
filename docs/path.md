# Path Module

File path manipulation, normalization, and resolution utilities.

**Use case:** Work with file paths across different operating systems.

---

## Import

```lunex
val path = @import("std.path")
```

---

## Note on Internal Modules

**Important:** The `native` module is an internal implementation detail of Lunex's standard library. It is not available for direct use in user code. This module provides native bindings that are only accessible from within the standard library modules.
