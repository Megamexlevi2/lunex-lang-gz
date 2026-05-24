# File System Module

File system operations including reading, writing, deleting, and directory listing.

**Use case:** Manage files and directories in your applications.

---

## Import

```lunex
val fs = @import("std.fs")
```

---

## Available Functions

### `readFile(path, encoding)`

Executes the `readFile` operation with the given parameters (path, encoding).

**Signature:**
```lunex
fn readFile(path, encoding)
```

### `writeFile(path, content, options)`

Executes the `writeFile` operation with the given parameters (path, content, options).

**Signature:**
```lunex
fn writeFile(path, content, options)
```

### `appendFile(path, content)`

Executes the `appendFile` operation with the given parameters (path, content).

**Signature:**
```lunex
fn appendFile(path, content)
```

### `deleteFile(path)`

Executes the `deleteFile` operation with the given parameter (path).

**Signature:**
```lunex
fn deleteFile(path)
```

### `exists(path)`

Executes the `exists` operation with the given parameter (path).

**Signature:**
```lunex
fn exists(path)
```

### `stat(path)`

Executes the `stat` operation with the given parameter (path).

**Signature:**
```lunex
fn stat(path)
```

### `mkdir(path, recursive)`

Executes the `mkdir` operation with the given parameters (path, recursive).

**Signature:**
```lunex
fn mkdir(path, recursive)
```

### `rmdir(path, recursive)`

Executes the `rmdir` operation with the given parameters (path, recursive).

**Signature:**
```lunex
fn rmdir(path, recursive)
```

### `list(path)`

Executes the `list` operation with the given parameter (path).

**Signature:**
```lunex
fn list(path)
```

### `listDir(path)`

Executes the `listDir` operation with the given parameter (path).

**Signature:**
```lunex
fn listDir(path)
```

### `copyFile(src, dest)`

Executes the `copyFile` operation with the given parameters (src, dest).

**Signature:**
```lunex
fn copyFile(src, dest)
```

### `moveFile(src, dest)`

Executes the `moveFile` operation with the given parameters (src, dest).

**Signature:**
```lunex
fn moveFile(src, dest)
```

### `join(...parts)`

Executes the `join` operation with the given parameter (...parts).

**Signature:**
```lunex
fn join(...parts)
```

### `basename(path, ext)`

Executes the `basename` operation with the given parameters (path, ext).

**Signature:**
```lunex
fn basename(path, ext)
```

### `dirname(path)`

Executes the `dirname` operation with the given parameter (path).

**Signature:**
```lunex
fn dirname(path)
```

### `extname(path)`

Executes the `extname` operation with the given parameter (path).

**Signature:**
```lunex
fn extname(path)
```

### `resolve(...parts)`

Executes the `resolve` operation with the given parameter (...parts).

**Signature:**
```lunex
fn resolve(...parts)
```

### `isAbsolute(path)`

Executes the `isAbsolute` operation with the given parameter (path).

**Signature:**
```lunex
fn isAbsolute(path)
```

### `isDir(path)`

Executes the `isDir` operation with the given parameter (path).

**Signature:**
```lunex
fn isDir(path)
```

### `isFile(path)`

Executes the `isFile` operation with the given parameter (path).

**Signature:**
```lunex
fn isFile(path)
```

### `readJSON(path)`

Executes the `readJSON` operation with the given parameter (path).

**Signature:**
```lunex
fn readJSON(path)
```

### `writeJSON(path, data, pretty)`

Executes the `writeJSON` operation with the given parameters (path, data, pretty).

**Signature:**
```lunex
fn writeJSON(path, data, pretty)
```

### `glob(pattern)`

Executes the `glob` operation with the given parameter (pattern).

**Signature:**
```lunex
fn glob(pattern)
```

### `watch(path, callback)`

Executes the `watch` operation with the given parameters (path, callback).

**Signature:**
```lunex
fn watch(path, callback)
```

