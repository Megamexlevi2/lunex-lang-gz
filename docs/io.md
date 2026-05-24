# Input/Output Module

Input/output operations including console logging, reading user input, and text formatting utilities.

**Use case:** Log output, format console text, and read user input.

---

## Import

```lunex
val io = @import("std.io")
```

---

## Available Functions

### `log(...args)`

Executes the `log` operation with the given parameter (...args).

**Signature:**
```lunex
fn log(...args)
```

### `error(...args)`

Executes the `error` operation with the given parameter (...args).

**Signature:**
```lunex
fn error(...args)
```

### `warn(...args)`

Executes the `warn` operation with the given parameter (...args).

**Signature:**
```lunex
fn warn(...args)
```

### `success(...args)`

Executes the `success` operation with the given parameter (...args).

**Signature:**
```lunex
fn success(...args)
```

### `info(...args)`

Executes the `info` operation with the given parameter (...args).

**Signature:**
```lunex
fn info(...args)
```

### `table(data, columns)`

Executes the `table` operation with the given parameters (data, columns).

**Signature:**
```lunex
fn table(data, columns)
```

### `progress(current, total, label)`

Executes the `progress` operation with the given parameters (current, total, label).

**Signature:**
```lunex
fn progress(current, total, label)
```

### `spinner(message)`

Executes the `spinner` operation with the given parameter (message).

**Signature:**
```lunex
fn spinner(message)
```

### `banner(text, color)`

Executes the `banner` operation with the given parameters (text, color).

**Signature:**
```lunex
fn banner(text, color)
```

### `hr(char, length)`

Executes the `hr` operation with the given parameters (char, length).

**Signature:**
```lunex
fn hr(char, length)
```

### `readLine(prompt)`

Executes the `readLine` operation with the given parameter (prompt).

**Signature:**
```lunex
fn readLine(prompt)
```

### `readInt(prompt)`

Executes the `readInt` operation with the given parameter (prompt).

**Signature:**
```lunex
fn readInt(prompt)
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```lunex
fn clear()
```

### `red(text)`

Executes the `red` operation with the given parameter (text).

**Signature:**
```lunex
fn red(text)
```

### `green(text)`

Executes the `green` operation with the given parameter (text).

**Signature:**
```lunex
fn green(text)
```

### `yellow(text)`

Executes the `yellow` operation with the given parameter (text).

**Signature:**
```lunex
fn yellow(text)
```

### `blue(text)`

Executes the `blue` operation with the given parameter (text).

**Signature:**
```lunex
fn blue(text)
```

### `cyan(text)`

Executes the `cyan` operation with the given parameter (text).

**Signature:**
```lunex
fn cyan(text)
```

### `magenta(text)`

Executes the `magenta` operation with the given parameter (text).

**Signature:**
```lunex
fn magenta(text)
```

### `bold(text)`

Executes the `bold` operation with the given parameter (text).

**Signature:**
```lunex
fn bold(text)
```

### `dim(text)`

Executes the `dim` operation with the given parameter (text).

**Signature:**
```lunex
fn dim(text)
```

