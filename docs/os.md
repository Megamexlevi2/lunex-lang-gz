# Operating System Module

Operating system utilities including process management, signal handling, and environment information.

**Use case:** Interact with the operating system and manage processes.

---

## Import

```ntl
val os = @import("std.os")
```

---

## Available Functions

### `_exec(command, options)`

Executes the `_exec` operation with the given parameters (command, options).

**Signature:**
```ntl
fn _exec(command, options)
```

### `_spawn(command, options)`

Executes the `_spawn` operation with the given parameters (command, options).

**Signature:**
```ntl
fn _spawn(command, options)
```

### `_getenv(key)`

Executes the `_getenv` operation with the given parameter (key).

**Signature:**
```ntl
fn _getenv(key)
```

### `_setenv(key, value)`

Executes the `_setenv` operation with the given parameters (key, value).

**Signature:**
```ntl
fn _setenv(key, value)
```

### `_unsetenv(key)`

Executes the `_unsetenv` operation with the given parameter (key).

**Signature:**
```ntl
fn _unsetenv(key)
```

### `_environ()`

Executes the `_environ` operation with the given no arguments.

**Signature:**
```ntl
fn _environ()
```

### `_getpid()`

Executes the `_getpid` operation with the given no arguments.

**Signature:**
```ntl
fn _getpid()
```

### `_getppid()`

Executes the `_getppid` operation with the given no arguments.

**Signature:**
```ntl
fn _getppid()
```

### `_getcwd()`

Executes the `_getcwd` operation with the given no arguments.

**Signature:**
```ntl
fn _getcwd()
```

### `_chdir(path)`

Executes the `_chdir` operation with the given parameter (path).

**Signature:**
```ntl
fn _chdir(path)
```

### `_hostname()`

Executes the `_hostname` operation with the given no arguments.

**Signature:**
```ntl
fn _hostname()
```

### `_platform()`

Executes the `_platform` operation with the given no arguments.

**Signature:**
```ntl
fn _platform()
```

### `_arch()`

Executes the `_arch` operation with the given no arguments.

**Signature:**
```ntl
fn _arch()
```

### `_cpus()`

Executes the `_cpus` operation with the given no arguments.

**Signature:**
```ntl
fn _cpus()
```

### `_exit(code)`

Executes the `_exit` operation with the given parameter (code).

**Signature:**
```ntl
fn _exit(code)
```

### `_args()`

Executes the `_args` operation with the given no arguments.

**Signature:**
```ntl
fn _args()
```

### `_stat(path)`

Executes the `_stat` operation with the given parameter (path).

**Signature:**
```ntl
fn _stat(path)
```

### `_exists(path)`

Executes the `_exists` operation with the given parameter (path).

**Signature:**
```ntl
fn _exists(path)
```

### `_mkdir(path, recursive)`

Executes the `_mkdir` operation with the given parameters (path, recursive).

**Signature:**
```ntl
fn _mkdir(path, recursive)
```

### `_remove(path, recursive)`

Executes the `_remove` operation with the given parameters (path, recursive).

**Signature:**
```ntl
fn _remove(path, recursive)
```

### `_rename(src, dst)`

Executes the `_rename` operation with the given parameters (src, dst).

**Signature:**
```ntl
fn _rename(src, dst)
```

### `_listDir(path)`

Executes the `_listDir` operation with the given parameter (path).

**Signature:**
```ntl
fn _listDir(path)
```

### `_glob(pattern)`

Executes the `_glob` operation with the given parameter (pattern).

**Signature:**
```ntl
fn _glob(pattern)
```

### `_tempDir()`

Executes the `_tempDir` operation with the given no arguments.

**Signature:**
```ntl
fn _tempDir()
```

### `_tempFile(prefix)`

Executes the `_tempFile` operation with the given parameter (prefix).

**Signature:**
```ntl
fn _tempFile(prefix)
```

### `_expandEnv(str)`

Executes the `_expandEnv` operation with the given parameter (str).

**Signature:**
```ntl
fn _expandEnv(str)
```

### `_join(...parts)`

Executes the `_join` operation with the given parameter (...parts).

**Signature:**
```ntl
fn _join(...parts)
```

### `_dirname(path)`

Executes the `_dirname` operation with the given parameter (path).

**Signature:**
```ntl
fn _dirname(path)
```

### `_basename(path)`

Executes the `_basename` operation with the given parameter (path).

**Signature:**
```ntl
fn _basename(path)
```

### `_extname(path)`

Executes the `_extname` operation with the given parameter (path).

**Signature:**
```ntl
fn _extname(path)
```

### `_abs(path)`

Executes the `_abs` operation with the given parameter (path).

**Signature:**
```ntl
fn _abs(path)
```

