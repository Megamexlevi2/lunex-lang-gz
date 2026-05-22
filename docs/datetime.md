# DateTime Module

Date and time manipulation, formatting, parsing, and timezone handling.

**Use case:** Work with dates, times, intervals, and schedule operations.

---

## Import

```ntl
val datetime = @import("std.datetime")
```

---

## Note on Internal Modules

**Important:** The `native` module is an internal implementation detail of NTL's standard library. It is not available for direct use in user code. This module provides native bindings that are only accessible from within the standard library modules.
