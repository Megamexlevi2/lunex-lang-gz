# Lunex — Language Reference

Lunex is a fast, statically-scoped scripting language. It is NOT a copy of JavaScript.

Key differences:
- No `return` — the last expression in a function is its result automatically.
- No `class` — use `val Name = struct { ... }` for named types.
- No `use` — use `val mod = @import("std.module")` to import modules.
- All code should run inside functions.
- Modules are explicit — nothing is imported by default.

## Quick Start

```lunex
val io = @import("std.io")

fn main() {
  io.log("Hello, Lunex!")
}
```

Run: `lunex hello.lx` or `lunex run hello.lx`

## Variables

### `val` — immutable binding

```lunex
val name = "Alice"
val count = 42
val active = true
val scores = [10, 20, 30]
val user = { name: "Bob", age: 25 }
```

### `var` — mutable binding

```lunex
var x = 0
x = x + 1
x += 10
```

## Types

Lunex is dynamically typed. Runtime types: `string`, `number`, `boolean`, `array`, `object`, `function`, `null`, `undefined`.

```lunex
typeof "hello"     // "string"
typeof 42          // "number"
typeof true        // "boolean"
typeof [1,2,3]     // "array"
typeof {}          // "object"
typeof null        // "null"
```

## Operators

```lunex
+  -  *  /  %  **          arithmetic
==  !=  <  >  <=  >=       comparison
===  !==                   strict equality
and  or  not               logical
?:                         ternary
??                         nullish coalescing
|>                         pipeline
```

## Control Flow

### if / else

```lunex
if x > 0 {
  io.log("positive")
} else if x < 0 {
  io.log("negative")
} else {
  io.log("zero")
}
```

### unless

```lunex
unless x == 0 {
  io.log("not zero")
}
```

### while

```lunex
var i = 0
while i < 10 {
  io.log(i)
  i += 1
}
```

### loop (infinite)

```lunex
loop {
  if done { break }
}
```

### repeat

```lunex
repeat 5 {
  io.log("hello")
}
```

### for / range

```lunex
each i in range(10) {
  io.log(i)
}

each i in range(2, 20, 2) {
  io.log(i)
}
```

### each

```lunex
val items = ["a", "b", "c"]
each item in items {
  io.log(item)
}
```

### break / continue

```lunex
while true {
  if done { break }
  if skip { continue }
}
```

### guard

```lunex
fn process(x) {
  guard x != null else { io.log("null input") }
  io.log(x)
}
```

### defer

```lunex
fn readFile(path) {
  val f = fs.open(path)
  defer f.close()
  f.read()
}
```

## Functions

The last expression in a function body is automatically its result.
There is **no** `return` keyword.

```lunex
fn add(a, b) {
  a + b
}

fn greet(name) {
  "Hello, " + name + "!"
}

val double = fn(x) { x * 2 }

fn sum(...nums) {
  var total = 0
  each n in nums { total += n }
  total
}
```

### Arrow-style

```lunex
val square = fn(x) { x * x }
val doubled = [1, 2, 3].map(fn(x) { x * 2 })
```

### Pipeline

```lunex
val result = [1, 2, 3]
  |> fn(arr) { arr.map(fn(x) { x * 2 }) }
  |> fn(arr) { arr.filter(fn(x) { x > 2 }) }
```

## Objects

```lunex
val person = {
  name: "Alice",
  age: 30,
  greet: fn() {
    "Hi, I am " + this.name
  }
}

io.log(person.name)
io.log(person["age"])
person.age = 31
```

## Arrays

```lunex
val arr = [1, 2, 3, 4, 5]

arr.push(6)
arr.pop()
arr.length

arr.map(fn(x) { x * 2 })
arr.filter(fn(x) { x > 2 })
arr.reduce(fn(acc, x) { acc + x }, 0)
arr.find(fn(x) { x > 3 })
arr.every(fn(x) { x > 0 })
arr.some(fn(x) { x > 4 })
arr.includes(3)
arr.slice(1, 3)
arr.join(", ")
arr.reverse()
arr.sort()
arr.forEach(fn(x) { io.log(x) })
```

## Structs

Lunex has no `class` keyword. Use `struct { ... }` to create a named type with methods.

```lunex
val Animal = struct {
  fn new(name, sound) {
    { name: name, sound: sound }
  }
  fn speak(self) {
    self.name + " says " + self.sound
  }
}

val Dog = struct {
  fn new(name) {
    { name: name, sound: "woof", tricks: [] }
  }
  fn learn(self, trick) {
    self.tricks.push(trick)
  }
}

val dog = Dog.new("Rex")
io.log(Animal.speak(dog))
Dog.learn(dog, "sit")
```

## Match

```lunex
fn describe(x) {
  match x {
    case null    => "nothing"
    case true    => "yes"
    case false   => "no"
    case 0       => "zero"
    default      => "something else"
  }
}
```

## Error Handling

```lunex
try {
  val result = riskyOperation()
} catch err {
  io.log("error:", err)
} finally {
  io.log("always runs")
}

throw "something went wrong"
```

### Safe call (returns null on error)

```lunex
val result = try? riskyOperation()
```

## Modules

Use `@import("std.module")` to load any standard library module.

```lunex
val io       = @import("std.io")
val fs       = @import("std.fs")
val http     = @import("std.http")
val crypto   = @import("std.crypto")
val db       = @import("std.db")
val env      = @import("std.env")
val validate = @import("std.validate")
val events   = @import("std.events")
val cache    = @import("std.cache")
val logger   = @import("std.logger")
val queue    = @import("std.queue")
val ws       = @import("std.ws")
val mail     = @import("std.mail")
val ai       = @import("std.ai")
val test     = @import("std.test")
val alloc    = @import("std.alloc")
```

### Removed — do NOT use these

```
use std/io       ← old syntax, causes an error
use "./utils"    ← old syntax, causes an error
```

### Removed keywords table

| Removed  | Reason                      | Replacement                                 |
|----------|-----------------------------|---------------------------------------------|
| `return` | Not needed                  | Last expression is the function result      |
| `class`  | No OO classes in Lunex        | `val Name = struct { fn new() { ... } }`    |
| `use`    | Replaced by `@import`       | `val mod = @import("std.module")`           |

## Concurrency

```lunex
spawn myFunction()

val ch = channel()

spawn fn() {
  ch.send(42)
}()

val value = ch.recv()
```

## Destructuring

```lunex
val { name, age } = person
val [first, second, ...rest] = items
```

## Spread

```lunex
val merged = { ...obj1, ...obj2 }
val combined = [...arr1, ...arr2]
fn call(fn, ...args) { fn(...args) }
```

## Template Strings

```lunex
val msg = `Hello, ${name}! You are ${age} years old.`
val math = `Result: ${2 + 2}`
```

## Optional Chaining & Nullish

```lunex
val value = maybeNull ?? "default"
val name = user?.profile?.name ?? "anonymous"
val len = arr?.length ?? 0
```

## String Operations

```lunex
val s = "Hello, World!"
s.length
s.toUpperCase()
s.toLowerCase()
s.trim()
s.split(", ")
s.replace("Hello", "Hi")
s.includes("World")
s.startsWith("Hello")
s.endsWith("!")
s.indexOf("o")
s.slice(0, 5)
s.repeat(3)
```

## Math

```lunex
Math.abs(-5)
Math.ceil(4.1)
Math.floor(4.9)
Math.round(4.5)
Math.max(1, 2, 3)
Math.min(1, 2, 3)
Math.sqrt(16)
Math.pow(2, 10)
Math.random()
Math.PI
```

## JSON

```lunex
val json = JSON.stringify({ name: "Alice" })
val obj = JSON.parse('{"name":"Alice"}')
val pretty = JSON.stringify(obj, null, 2)
```
