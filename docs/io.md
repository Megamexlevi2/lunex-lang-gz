# Input/Output Module

Input/output operations including console logging, reading user input, and text formatting utilities.

**Use case:** Log output, format console text, and read user input.

---

## Import

```ntl
val io = @import("std.io")
```

---

## Available Functions

### `log(...args)`

Executes the `log` operation with the given parameter (...args).

**Signature:**
```ntl
fn log(...args)
```

### `error(...args)`

Executes the `error` operation with the given parameter (...args).

**Signature:**
```ntl
fn error(...args)
```

### `warn(...args)`

Executes the `warn` operation with the given parameter (...args).

**Signature:**
```ntl
fn warn(...args)
```

### `success(...args)`

Executes the `success` operation with the given parameter (...args).

**Signature:**
```ntl
fn success(...args)
```

### `info(...args)`

Executes the `info` operation with the given parameter (...args).

**Signature:**
```ntl
fn info(...args)
```

### `table(data, columns)`

Executes the `table` operation with the given parameters (data, columns).

**Signature:**
```ntl
fn table(data, columns)
```

### `progress(current, total, label)`

Executes the `progress` operation with the given parameters (current, total, label).

**Signature:**
```ntl
fn progress(current, total, label)
```

### `spinner(message)`

Executes the `spinner` operation with the given parameter (message).

**Signature:**
```ntl
fn spinner(message)
```

### `banner(text, color)`

Executes the `banner` operation with the given parameters (text, color).

**Signature:**
```ntl
fn banner(text, color)
```

### `hr(char, length)`

Executes the `hr` operation with the given parameters (char, length).

**Signature:**
```ntl
fn hr(char, length)
```

### `readLine(prompt)`

Executes the `readLine` operation with the given parameter (prompt).

**Signature:**
```ntl
fn readLine(prompt)
```

### `readInt(prompt)`

Executes the `readInt` operation with the given parameter (prompt).

**Signature:**
```ntl
fn readInt(prompt)
```

### `clear()`

Executes the `clear` operation with the given no arguments.

**Signature:**
```ntl
fn clear()
```

### `red(text)`

Executes the `red` operation with the given parameter (text).

**Signature:**
```ntl
fn red(text)
```

### `green(text)`

Executes the `green` operation with the given parameter (text).

**Signature:**
```ntl
fn green(text)
```

### `yellow(text)`

Executes the `yellow` operation with the given parameter (text).

**Signature:**
```ntl
fn yellow(text)
```

### `blue(text)`

Executes the `blue` operation with the given parameter (text).

**Signature:**
```ntl
fn blue(text)
```

### `cyan(text)`

Executes the `cyan` operation with the given parameter (text).

**Signature:**
```ntl
fn cyan(text)
```

### `magenta(text)`

Executes the `magenta` operation with the given parameter (text).

**Signature:**
```ntl
fn magenta(text)
```

### `bold(text)`

Executes the `bold` operation with the given parameter (text).

**Signature:**
```ntl
fn bold(text)
```

### `dim(text)`

Executes the `dim` operation with the given parameter (text).

**Signature:**
```ntl
fn dim(text)
```

