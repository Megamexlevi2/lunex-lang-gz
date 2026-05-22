# Logger Module

Structured logging with multiple severity levels, formatting options, and output targets.

**Use case:** Log application events with different severity levels.

---

## Import

```ntl
val logger = @import("std.logger")
```

---

## Available Functions

### `_levelLabel(level)`

Executes the `_levelLabel` operation with the given parameter (level).

**Signature:**
```ntl
fn _levelLabel(level)
```

### `_formatMeta(meta)`

Executes the `_formatMeta` operation with the given parameter (meta).

**Signature:**
```ntl
fn _formatMeta(meta)
```

### `create(name, options)`

Executes the `create` operation with the given parameters (name, options).

**Signature:**
```ntl
fn create(name, options)
```

### `_write(lvl, msg, meta)`

Executes the `_write` operation with the given parameters (lvl, msg, meta).

**Signature:**
```ntl
fn _write(lvl, msg, meta)
```

### `_setLevel(lvl)`

Executes the `_setLevel` operation with the given parameter (lvl).

**Signature:**
```ntl
fn _setLevel(lvl)
```

### `_getLevel()`

Executes the `_getLevel` operation with the given no arguments.

**Signature:**
```ntl
fn _getLevel()
```

### `_addHandler(h)`

Executes the `_addHandler` operation with the given parameter (h).

**Signature:**
```ntl
fn _addHandler(h)
```

### `_clearHandlers()`

Executes the `_clearHandlers` operation with the given no arguments.

**Signature:**
```ntl
fn _clearHandlers()
```

### `_removeHandler(h)`

Executes the `_removeHandler` operation with the given parameter (h).

**Signature:**
```ntl
fn _removeHandler(h)
```

### `_count()`

Executes the `_count` operation with the given no arguments.

**Signature:**
```ntl
fn _count()
```

### `_reset()`

Executes the `_reset` operation with the given no arguments.

**Signature:**
```ntl
fn _reset()
```

### `_trace(msg, meta)`

Executes the `_trace` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _trace(msg, meta)
```

### `_debug(msg, meta)`

Executes the `_debug` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _debug(msg, meta)
```

### `_info(msg, meta)`

Executes the `_info` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _info(msg, meta)
```

### `_warn(msg, meta)`

Executes the `_warn` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _warn(msg, meta)
```

### `_error(msg, meta)`

Executes the `_error` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _error(msg, meta)
```

### `_fatal(msg, meta)`

Executes the `_fatal` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _fatal(msg, meta)
```

### `_log(msg, meta)`

Executes the `_log` operation with the given parameters (msg, meta).

**Signature:**
```ntl
fn _log(msg, meta)
```

### `_child(childName, extraCtx)`

Executes the `_child` operation with the given parameters (childName, extraCtx).

**Signature:**
```ntl
fn _child(childName, extraCtx)
```

### `_withContext(ctx)`

Executes the `_withContext` operation with the given parameter (ctx).

**Signature:**
```ntl
fn _withContext(ctx)
```

### `_timed(label)`

Executes the `_timed` operation with the given parameter (label).

**Signature:**
```ntl
fn _timed(label)
```

### `_group(label)`

Executes the `_group` operation with the given parameter (label).

**Signature:**
```ntl
fn _group(label)
```

### `_assert(condition, msg, meta)`

Executes the `_assert` operation with the given parameters (condition, msg, meta).

**Signature:**
```ntl
fn _assert(condition, msg, meta)
```

### `_fileHandler(path)`

Executes the `_fileHandler` operation with the given parameter (path).

**Signature:**
```ntl
fn _fileHandler(path)
```

### `_jsonHandler(path)`

Executes the `_jsonHandler` operation with the given parameter (path).

**Signature:**
```ntl
fn _jsonHandler(path)
```

