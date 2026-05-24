# Compression Module

Data compression and decompression utilities supporting multiple compression algorithms.

**Use case:** Reduce file size, bandwidth usage, or storage requirements.

---

## Import

```lunex
val compress = @import("std.compress")
```

---

## Note on Internal Modules

**Important:** The `native` module is an internal implementation detail of Lunex's standard library. It is not available for direct use in user code. This module provides native bindings that are only accessible from within the standard library modules.
