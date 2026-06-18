# Language Reference

Complete reference for the Lunex programming language, version 0.7.1.

---

## Lexical structure

### Comments

```lx
// single-line comment
```

### Identifiers

Identifiers start with a letter or underscore and may contain letters,
digits, and underscores. They are case-sensitive.

```lx
myVar   _private   score99   HTTP_PORT
```

### Keywords

```
val var fn struct if else guard unless while each in break continue
match defer spawn channel null true false
```

> **Note:** Lunex does not have a `return` keyword. The last expression in a
> function body is the function's value automatically.

### Literals

| Kind | Examples |
|------|---------|
| Number | `0` `42` `3.14` `-1` `1e6` |
| String | `"hello"` `"line\nbreak"` `"tab\there"` |
| Template | `` `Hello, ${name}!` `` |
| Boolean | `true` `false` |
| Null | `null` |
| Array | `[1, 2, 3]` |
| Object | `{ key: value, other: 42 }` |

---

## Bindings

### `val` — immutable binding

```lx
val x    = 42
val name = "Lunex"
val pi   = 3.14159
```

`val` bindings cannot be reassigned. Attempting to reassign one is a
compile-time error.

### `var` — mutable binding

```lx
var count = 0
count = count + 1
```

### Destructuring

Object destructuring:

```lx
val { name, role, score } = getUser()
```

Array destructuring:

```lx
val [first, second, third] = getItems()
```

---

## Types

Lunex is dynamically typed. The runtime types are:

| Type | Description |
|------|-------------|
| `number` | 64-bit IEEE 754 floating point |
| `string` | UTF-8 string |
| `boolean` | `true` or `false` |
| `null` | absence of a value |
| `array` | ordered collection |
| `object` | key-value map |
| `struct` | object with methods and `this` binding |
| `function` | first-class callable |
| `channel` | concurrent message channel |

Use the global `typeof(v)` to inspect the type of a value at runtime:

```lx
typeof("hello")  // "string"
typeof(42)       // "number"
typeof(true)     // "boolean"
typeof(null)     // "null"
typeof([])       // "array"
typeof({})       // "object"
```

---

## Operators

### Arithmetic

```lx
a + b    // addition (also string concatenation)
a - b    // subtraction
a * b    // multiplication
a / b    // division
a % b    // modulo
```

### Comparison

```lx
a == b   // equal
a != b   // not equal
a < b    a > b    a <= b    a >= b
```

### Logical

```lx
a && b   // AND (short-circuits)
a || b   // OR (short-circuits)
!a       // NOT
```

### Assignment

```lx
x = expr          // reassign a var binding
obj.field = expr  // set an object or struct field
arr[i] = expr     // set an array element
```

---

## Global built-in functions

These functions are available everywhere without an import:

| Function | Description |
|----------|-------------|
| `str(v)` | Convert any value to its string representation |
| `num(v)` | Convert a value to a number |
| `typeof(v)` | Return the type name of a value as a string |
| `parseInt(s)` | Parse a string as an integer |
| `parseFloat(s)` | Parse a string as a floating-point number |
| `isNaN(v)` | Return `true` if the value is NaN |
| `isFinite(v)` | Return `true` if the value is a finite number |
| `channel()` | Create an unbuffered concurrent channel |

```lx
str(42)          // "42"
str(true)        // "true"
num("3.14")      // 3.14
typeof("hello")  // "string"
parseInt("10")   // 10
isNaN(0 / 0)     // true
```

---

## Functions

### Declaration

```lx
fn add(a, b) {
  a + b
}
```

The last expression in the body is the function's value. There is no
`return` keyword.

### Anonymous functions

```lx
val square = fn(x) { x * x }
```

### Immediately invoked

```lx
val result = fn(a, b) { a + b }(3, 4)
```

### First-class functions

Functions are values. They can be passed as arguments, returned from other
functions, and stored in arrays or objects.

```lx
fn apply(f, x) { f(x) }

fn compose(f, g) {
  fn(x) { f(g(x)) }
}

val ops = [
  fn(x) { x + 1 }
  fn(x) { x * 2 }
]
```

### Closures

Functions capture variables from the enclosing scope. Mutations in the
outer scope are visible through the closure.

```lx
fn makeAdder(n) {
  fn(x) { x + n }
}

val add5 = makeAdder(5)
io.log(add5(10))  // 15
```

---

## Control flow

### `if` / `else if` / `else`

```lx
if score >= 90 {
  "A"
} else if score >= 80 {
  "B"
} else {
  "C"
}
```

`if` is an expression — its value can be assigned:

```lx
val label = if n > 0 { "positive" } else { "non-positive" }
```

### `guard`

Runs its `else` block when the condition is **false**. Execution continues
after the guard statement.

```lx
fn process(user) {
  guard user != null else {
    io.err("no user — skipping")
  }
  // continues here whether or not the else block ran
  io.log("processing:", user)
}
```

`guard` is most useful for early-out patterns where you set a flag or log a
warning before skipping the rest of the function body with a surrounding `if`.

### `unless`

Runs its block when the condition is **false**:

```lx
unless connected {
  io.warn("no connection — retrying")
}
```

### `match`

Tests a value against a list of exact arms. The first match wins.

```lx
val label = match status {
  "ok"      => "success"
  "pending" => "waiting"
  "fail"    => "error"
  _         => "unknown"
}
```

`match` is an expression and can be used as a value. The `_` wildcard
matches anything. Arms are checked top-to-bottom.

For range-based classification, use `if` / `else if` chains:

```lx
fn classify(n) {
  if n == 0       { "zero" }
  else if n <= 9  { "single digit" }
  else if n <= 99 { "double digit" }
  else            { "large" }
}
```

### `defer`

Schedules a block to run when the enclosing function exits, regardless of
how it exits. Multiple defers run in reverse order (last in, first out).

```lx
fn readAndProcess(path) {
  val fs = @import("std.fs")
  defer { io.log("done with", path) }
  fs.readFile(path)
}
```

---

## Loops

### `while`

```lx
var i = 0
while i < 10 {
  io.log(i)
  i = i + 1
}
```

Use `break` to exit early, `continue` to skip to the next iteration.

### `each … in`

Iterate over an array:

```lx
each name in ["Alice", "Bob", "Carol"] {
  io.log("Hello,", name)
}
```

Iterate over a string (character by character):

```lx
each ch in "lunex" {
  io.log(ch)
}
```

Iterate over an object (yields each key):

```lx
each key in config {
  io.log(key, "=", config[key])
}
```

---

## Native array methods

Arrays have built-in methods — no import required.

| Method | Returns | Description |
|--------|---------|-------------|
| `arr.length` | number | Number of elements |
| `arr.push(v)` | — | Append a value in place |
| `arr.pop()` | value | Remove and return the last element |
| `arr.map(fn)` | array | Return a new array with each element transformed |
| `arr.filter(fn)` | array | Return a new array of elements where fn is truthy |
| `arr.reduce(fn, init)` | value | Fold the array left to a single value |
| `arr.find(fn)` | value \| null | First element where fn is truthy |
| `arr.includes(v)` | boolean | True if the value is in the array |
| `arr.indexOf(v)` | number | Index of the first occurrence (−1 if absent) |
| `arr.sort()` | array | Sort in ascending order (returns new array) |
| `arr.reverse()` | array | Reverse the order (returns new array) |
| `arr.slice(start, end)` | array | Sub-array from start to end (exclusive) |
| `arr.join(sep)` | string | Concatenate elements with a separator |
| `arr.every(fn)` | boolean | True if all elements satisfy fn |
| `arr.some(fn)` | boolean | True if any element satisfies fn |
| `arr.flat()` | array | Flatten one level of nesting |

```lx
val nums = [3, 1, 4, 1, 5, 9, 2, 6]
io.log(nums.sort())                          // [1, 1, 2, 3, 4, 5, 6, 9]
io.log(nums.filter(fn(x) { x > 4 }))        // [5, 9, 6]
io.log(nums.map(fn(x) { x * 2 }))           // [6, 2, 8, 2, 10, 18, 4, 12]
io.log(nums.reduce(fn(acc, x) { acc + x }, 0))  // 31
```

---

## Native string methods

Strings have built-in methods — no import required.

| Method | Returns | Description |
|--------|---------|-------------|
| `s.length` | number | Number of UTF-8 characters |
| `s.toUpperCase()` | string | Convert to uppercase |
| `s.toLowerCase()` | string | Convert to lowercase |
| `s.trim()` | string | Remove leading and trailing whitespace |
| `s.trimStart()` | string | Remove leading whitespace |
| `s.trimEnd()` | string | Remove trailing whitespace |
| `s.startsWith(prefix)` | boolean | True if string starts with prefix |
| `s.endsWith(suffix)` | boolean | True if string ends with suffix |
| `s.includes(sub)` | boolean | True if string contains sub |
| `s.indexOf(sub)` | number | Index of first occurrence (−1 if absent) |
| `s.split(sep)` | array | Split on separator |
| `s.slice(start, end)` | string | Substring from start to end (exclusive) |
| `s.replace(old, new)` | string | Replace first occurrence |
| `s.replaceAll(old, new)` | string | Replace all occurrences |
| `s.repeat(n)` | string | Repeat the string n times |

```lx
"  hello  ".trim()              // "hello"
"lunex".toUpperCase()           // "LUNEX"
"hello world".includes("world") // true
"hello".split("")               // ["h","e","l","l","o"]
```

---

## Structs

A `struct` is an object with named fields and methods. Methods can use
`this` to refer to the struct instance.

```lx
fn Counter(start) {
  var count = start
  struct {
    fn inc()   { count = count + 1 }
    fn dec()   { count = count - 1 }
    fn value() { count }
    fn reset() { count = 0 }
  }
}

val c = Counter(0)
c.inc()
c.inc()
io.log(c.value())  // 2
```

Plain assignments in a `struct` body become fields:

```lx
fn Point(x, y) {
  struct {
    x = x
    y = y
    fn length() {
      val math = @import("std.math")
      math.sqrt(this.x * this.x + this.y * this.y)
    }
  }
}
```

---

## Concurrency

### `channel()`

Creates an unbuffered FIFO channel.

```lx
val ch = channel()
```

### `ch.send(value)` / `ch.recv()`

`send` blocks until a receiver is ready. `recv` blocks until a sender sends.

```lx
spawn fn() {
  ch.send(compute())
}()

val result = ch.recv()
```

### `spawn`

Launches a function call in a new goroutine.

```lx
spawn fn() {
  heavyWork()
}()
```

---

## Modules

### Standard library

```lx
val io   = @import("std.io")
val math = @import("std.math")
val http = @import("std.http")
```

See [stdlib.md](stdlib.md) for the full module reference.

### Local files

```lx
val utils = @fimport("./src/utils.lx")
val lib   = @fimport("./build/math.nax")
```

### External packages

```lx
val pkg = @import("github.com/user/repo")
val pkg = @import("https://example.com/mylib")
```

---

## Error messages

Lunex error messages include a source window, a caret pointing at the
problem, and an automatic fix suggestion where possible. Every error carries
a code (`E0001`–`E0071`) for reference in the error documentation.

```
error[E0021] UndefinedVariable: 'usr' is not defined
   → main.lx:12:3
 10 │
 11 │  fn greet(user) {
 12 │    io.log(usr.name)
    │           ^^^
    │           did you mean 'user'?
```

---

## Type conversions

Lunex does not implicitly coerce between types. Use global built-in
conversion functions for explicit conversions:

```lx
str(42)           // "42"
str(true)         // "true"
num("3.14")       // 3.14
parseInt("10")    // 10
parseFloat("1.5") // 1.5
```

String concatenation with `+` converts the right-hand operand to a string
when the left-hand operand is a string:

```lx
"score: " + 99    // "score: 99"
"items: " + 5     // "items: 5"
```

---

## Scope rules

- Variables are lexically scoped to the block in which they are declared.
- `fn` declarations at the top level of a file are visible throughout the file.
- Closures capture variables by reference — mutations are visible through the closure.
- `struct` methods defined with `this` receive the struct as their implicit receiver.
- `@import` and `@fimport` are scoped to the binding they are assigned to.
