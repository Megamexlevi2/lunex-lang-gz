# File System Module

File system operations including reading, writing, deleting, and directory listing.

**Use case:** Manage files and directories in your applications.

---

## Import

```ntl
val fs = @import("std.fs")
```

---

## Available Functions

### `readFile(path, encoding)`

Executes the `readFile` operation with the given parameters (path, encoding).

**Signature:**
```ntl
fn readFile(path, encoding)
```

### `writeFile(path, content, options)`

Executes the `writeFile` operation with the given parameters (path, content, options).

**Signature:**
```ntl
fn writeFile(path, content, options)
```

### `appendFile(path, content)`

Executes the `appendFile` operation with the given parameters (path, content).

**Signature:**
```ntl
fn appendFile(path, content)
```

### `deleteFile(path)`

Executes the `deleteFile` operation with the given parameter (path).

**Signature:**
```ntl
fn deleteFile(path)
```

### `exists(path)`

Executes the `exists` operation with the given parameter (path).

**Signature:**
```ntl
fn exists(path)
```

### `stat(path)`

Executes the `stat` operation with the given parameter (path).

**Signature:**
```ntl
fn stat(path)
```

### `mkdir(path, recursive)`

Executes the `mkdir` operation with the given parameters (path, recursive).

**Signature:**
```ntl
fn mkdir(path, recursive)
```

### `rmdir(path, recursive)`

Executes the `rmdir` operation with the given parameters (path, recursive).

**Signature:**
```ntl
fn rmdir(path, recursive)
```

### `list(path)`

Executes the `list` operation with the given parameter (path).

**Signature:**
```ntl
fn list(path)
```

### `listDir(path)`

Executes the `listDir` operation with the given parameter (path).

**Signature:**
```ntl
fn listDir(path)
```

### `copyFile(src, dest)`

Executes the `copyFile` operation with the given parameters (src, dest).

**Signature:**
```ntl
fn copyFile(src, dest)
```

### `moveFile(src, dest)`

Executes the `moveFile` operation with the given parameters (src, dest).

**Signature:**
```ntl
fn moveFile(src, dest)
```

### `join(...parts)`

Executes the `join` operation with the given parameter (...parts).

**Signature:**
```ntl
fn join(...parts)
```

### `basename(path, ext)`

Executes the `basename` operation with the given parameters (path, ext).

**Signature:**
```ntl
fn basename(path, ext)
```

### `dirname(path)`

Executes the `dirname` operation with the given parameter (path).

**Signature:**
```ntl
fn dirname(path)
```

### `extname(path)`

Executes the `extname` operation with the given parameter (path).

**Signature:**
```ntl
fn extname(path)
```

### `resolve(...parts)`

Executes the `resolve` operation with the given parameter (...parts).

**Signature:**
```ntl
fn resolve(...parts)
```

### `isAbsolute(path)`

Executes the `isAbsolute` operation with the given parameter (path).

**Signature:**
```ntl
fn isAbsolute(path)
```

### `isDir(path)`

Executes the `isDir` operation with the given parameter (path).

**Signature:**
```ntl
fn isDir(path)
```

### `isFile(path)`

Executes the `isFile` operation with the given parameter (path).

**Signature:**
```ntl
fn isFile(path)
```

### `readJSON(path)`

Executes the `readJSON` operation with the given parameter (path).

**Signature:**
```ntl
fn readJSON(path)
```

### `writeJSON(path, data, pretty)`

Executes the `writeJSON` operation with the given parameters (path, data, pretty).

**Signature:**
```ntl
fn writeJSON(path, data, pretty)
```

### `glob(pattern)`

Executes the `glob` operation with the given parameter (pattern).

**Signature:**
```ntl
fn glob(pattern)
```

### `watch(path, callback)`

Executes the `watch` operation with the given parameters (path, callback).

**Signature:**
```ntl
fn watch(path, callback)
```

