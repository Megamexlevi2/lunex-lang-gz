# NTL Language Reference

NTL v2.0 — a fast, expressive scripting language.

## Quick Start

```ntl
use io

fn main() {
  io.log("Hello, world!")
}
```

Run with: `ntl hello.ntl` or `ntl run hello.ntl`

## Variables

### `val` — immutable binding

```ntl
val name = "Alice"
val count = 42
val active = true
val scores = [10, 20, 30]
val user = { name: "Bob", age: 25 }
```

### `var` — mutable binding

```ntl
var x = 0
x = x + 1
x += 10
```

## Types

NTL is dynamically typed. Runtime types: `string`, `number`, `boolean`, `array`, `object`, `function`, `null`, `undefined`.

```ntl
typeof "hello"     // "string"
typeof 42          // "number"
typeof true        // "boolean"
typeof [1,2,3]     // "array"
typeof {}          // "object"
typeof null        // "null"
```

## Operators

```ntl
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

```ntl
if x > 0 {
  io.log("positive")
} else if x < 0 {
  io.log("negative")
} else {
  io.log("zero")
}
```

### unless

```ntl
unless x == 0 {
  io.log("not zero")
}
```

### while

```ntl
var i = 0
while i < 10 {
  io.log(i)
  i += 1
}
```

### loop (infinite)

```ntl
loop {
  if done { break }
}
```

### repeat

```ntl
repeat 5 {
  io.log("hello")
}
```

### for / range

```ntl
each i in range(10) {
  io.log(i)
}

each i in range(2, 20, 2) {
  io.log(i)
}
```

### each

```ntl
val items = ["a", "b", "c"]
each item in items {
  io.log(item)
}
```

### break / continue

```ntl
while true {
  if done { break }
  if skip { continue }
}
```

### guard

```ntl
fn process(x) {
  guard x != null else { return }
  io.log(x)
}
```

### defer

```ntl
fn readFile(path) {
  val f = fs.open(path)
  defer f.close()
  return f.read()
}
```

## Functions

```ntl
fn add(a, b) {
  return a + b
}

fn greet(name) {
  return "Hello, " + name + "!"
}

val double = fn(x) { return x * 2 }

fn sum(...nums) {
  var total = 0
  each n in nums { total += n }
  return total
}
```

### Arrow-style

```ntl
val square = fn(x) { return x * x }
val doubled = [1, 2, 3].map(fn(x) { return x * 2 })
```

### Pipeline

```ntl
val result = [1, 2, 3]
  |> fn(arr) { return arr.map(fn(x) { return x * 2 }) }
  |> fn(arr) { return arr.filter(fn(x) { return x > 2 }) }
```

## Objects

```ntl
val person = {
  name: "Alice",
  age: 30,
  greet: fn() {
    return "Hi, I am " + this.name
  }
}

io.log(person.name)
io.log(person["age"])
person.age = 31
```

## Arrays

```ntl
val arr = [1, 2, 3, 4, 5]

arr.push(6)
arr.pop()
arr.length

arr.map(fn(x) { return x * 2 })
arr.filter(fn(x) { return x > 2 })
arr.reduce(fn(acc, x) { return acc + x }, 0)
arr.find(fn(x) { return x > 3 })
arr.every(fn(x) { return x > 0 })
arr.some(fn(x) { return x > 4 })
arr.includes(3)
arr.slice(1, 3)
arr.join(", ")
arr.reverse()
arr.sort()
arr.forEach(fn(x) { io.log(x) })
```

## Classes

```ntl
class Animal {
  constructor(name, sound) {
    this.name = name
    this.sound = sound
  }

  speak() {
    return this.name + " says " + this.sound
  }
}

class Dog extends Animal {
  constructor(name) {
    super(name, "woof")
    this.tricks = []
  }

  learn(trick) {
    this.tricks.push(trick)
  }
}

val dog = new Dog("Rex")
io.log(dog.speak())
dog.learn("sit")
```

## Match

```ntl
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

```ntl
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

```ntl
val result = try? riskyOperation()
```

## Modules

### Stdlib (built-in)

```ntl
use io
use fs
use http
use crypto
use db
use env
use utils
use validate
use events
use cache
use logger
use queue
use ws
use mail
use ai
use test
use type
```

### Local files

```ntl
use "./utils"
use "./models/user"
```

### Packages (GitHub or registry)

```ntl
use "github.com/user/package"
use "colors"
```

## Concurrency

```ntl
spawn myFunction()

val ch = channel()

spawn fn() {
  ch.send(42)
}()

val value = ch.recv()
```

## Destructuring

```ntl
val { name, age } = person
val [first, second, ...rest] = items
```

## Spread

```ntl
val merged = { ...obj1, ...obj2 }
val combined = [...arr1, ...arr2]
fn call(fn, ...args) { fn(...args) }
```

## Template Strings

```ntl
val msg = `Hello, ${name}! You are ${age} years old.`
val math = `Result: ${2 + 2}`
```

## Optional Chaining & Nullish

```ntl
val value = maybeNull ?? "default"
val name = user?.profile?.name ?? "anonymous"
val len = arr?.length ?? 0
```

## String Operations

```ntl
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

```ntl
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

```ntl
val json = JSON.stringify({ name: "Alice" })
val obj = JSON.parse('{"name":"Alice"}')
val pretty = JSON.stringify(obj, null, 2)
```
