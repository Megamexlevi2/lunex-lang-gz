# io — Console I/O Module

The `io` module provides all console input/output functions, colors, spinners, progress bars, and formatted output.

## Import

```ntl
use io
```

---

## Output

### `io.log(...args)`
Print values to stdout separated by spaces, followed by a newline.

```ntl
io.log("Hello", "World")       // Hello World
io.log(42, true, [1, 2])       // 42 true [1, 2]
```

### `io.io.log(...args)`
Print without a trailing newline.

```ntl
io.io.log("Name: ")
io.io.log("Alice")
```

### `io.println(...args)`
Alias for `io.log`.

### `io.write(...args)`
Alias for `io.print`.

### `io.error(...args)`
Print to stderr in red.

```ntl
io.error("Something went wrong")
```

### `io.warn(...args)`
Print to stderr in yellow.

```ntl
io.warn("Disk space low")
```

### `io.info(...args)`
Print to stdout in cyan.

```ntl
io.info("Server started on port 3000")
```

### `io.success(...args)`
Print to stdout in green with a ✔ prefix.

```ntl
io.success("Build complete")
```

---

## Colors

Each color function wraps a string in ANSI color codes.

```ntl
io.red("error!")
io.green("ok")
io.yellow("warning")
io.blue("info")
io.magenta("debug")
io.cyan("hint")
io.white("plain")
io.gray("dim")
io.bold("important")
io.dim("subtle")
io.italic("emphasis")
```

### `io.color(colorName, text)`
Apply a named color to text.

```ntl
io.log(io.color("cyan", "Hello"))
```

### `io.strip(text)`
Remove ANSI escape codes from a string.

```ntl
val plain = io.strip(io.red("hello"))
```

---

## Input

### `io.read([prompt])`
Read a line from stdin, optionally printing a prompt first.

```ntl
val name = io.read("Your name: ")
io.log("Hello,", name)
```

### `io.readLine([prompt])`
Alias for `io.read`.

### `io.readInt([prompt])`
Read an integer from stdin.

```ntl
val age = io.readInt("Your age: ")
```

---

## Formatting

### `io.format(template, ...args)`
Replace `{}` or `{0}`, `{1}` placeholders with arguments.

```ntl
val msg = io.format("Hello, {}! You are {} years old.", "Alice", 30)
io.log(msg)   // Hello, Alice! You are 30 years old.
```

---

## Structured Output

### `io.table(array)`
Render an array of objects as a formatted table.

```ntl
io.table([
  { name: "Alice", age: 30 },
  { name: "Bob",   age: 25 },
])
```

### `io.json(value)`
Print a value as formatted JSON.

```ntl
io.json({ key: "value", list: [1, 2, 3] })
```

### `io.banner(text)`
Print text inside a box border.

```ntl
io.banner("NTL v2.0")
```

### `io.hr([width], [char])`
Print a horizontal rule.

```ntl
io.hr()           // ────────────────────────────────────────────────────────────
io.hr(40, "═")    // ════════════════════════════════════════
```

### `io.newline([n])`
Print one or more blank lines.

```ntl
io.newline()     // one blank line
io.newline(3)    // three blank lines
```

---

## Progress

### `io.progress(current, total)`
Render an inline progress bar.

```ntl
for i in range(0, 100) {
  io.progress(i, 100)
  sleep(10)
}
```

### `io.spinner()`
Create a spinner object.

```ntl
val spin = io.spinner()
spin.tick("Loading...")
sleep(500)
spin.stop()
```

---

## Terminal

### `io.clear()`
Clear the terminal screen.

### `io.isTerminal()`
Returns `true` if stdout is a real terminal (not piped).

```ntl
if io.isTerminal() {
  io.log(io.green("Colors supported"))
}
```

---

## Example: Full Script

```ntl
use io

fn greet(name) {
  io.log(io.cyan("Hello,"), io.bold(name) + "!")
  io.hr(40)
}

fn main() {
  val name = io.read("Enter your name: ")
  greet(name)
  io.success("Done")
}
```
