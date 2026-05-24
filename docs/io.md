# Input/Output Module

The Input/Output module provides functions for console logging, user input, and terminal text formatting.

**Use case:** print messages, format console output, and read user input.

---

# Import

```lunex
val io = @import("std.io")
```

---

# Available Functions

## log(...args)

Prints values to the console.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log("Hello, world!")
}
```

---

## error(...args)

Prints an error message.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.error("Something went wrong!")
}
```

---

## warn(...args)

Prints a warning message.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.warn("Warning message!")
}
```

---

## success(...args)

Prints a success message.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.success("Operation completed!")
}
```

---

## info(...args)

Prints an informational message.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.info("Information message")
}
```

---

## table(data, columns)

Displays a table.

### Example

```lunex
val io = @import("std.io")

fn main() {
  val users = [
    { name: "David", age: 18 },
    { name: "Lucas", age: 20 }
  ]

  io.table(users, ["name", "age"])
}
```

---

## progress(current, total, label)

Displays a progress bar.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.progress(50, 100, "Loading")
}
```

---

## spinner(message)

Displays a loading spinner.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.spinner("Loading...")
}
```

---

## banner(text, color)

Displays a banner.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.banner("Lunex Lang", "cyan")
}
```

---

## hr(char, length)

Displays a horizontal line.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.hr("-", 30)
}
```

---

## readLine(prompt)

Reads text input from the user.

### Example

```lunex
val io = @import("std.io")

fn main() {
  val name = io.readLine("Your name: ")

  io.log("Hello " + name)
}
```

---

## readInt(prompt)

Reads an integer from the user.

### Example

```lunex
val io = @import("std.io")

fn main() {
  val age = io.readInt("Your age: ")

  io.log(age)
}
```

---

## clear()

Clears the console.

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.clear()
}
```

---

# Color Helpers

Color helpers should be used inside `io.log()`.

---

## red(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.red("Red text"))
}
```

---

## green(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.green("Green text"))
}
```

---

## yellow(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.yellow("Yellow text"))
}
```

---

## blue(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.blue("Blue text"))
}
```

---

## cyan(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.cyan("Cyan text"))
}
```

---

## magenta(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.magenta("Magenta text"))
}
```

---

## bold(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.bold("Bold text"))
}
```

---

## dim(text)

### Example

```lunex
val io = @import("std.io")

fn main() {
  io.log(io.dim("Dim text"))
}
```