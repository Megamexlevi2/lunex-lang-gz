# utils — Utilities Module

The `utils` module provides general-purpose helpers: type checks, deep operations, date/time, string utilities, math helpers, and more.

## Import

```ntl
val utils = @import("std.utils")
```

---

## Type Checking

```ntl
utils.isString(x)     // true if x is a string
utils.isNumber(x)     // true if x is a number
utils.isBool(x)       // true if x is a boolean
utils.isArray(x)      // true if x is an array
utils.isObject(x)     // true if x is an object
utils.isFunction(x)   // true if x is a function
utils.isNull(x)       // true if x is null
utils.isUndefined(x)  // true if x is undefined
utils.typeOf(x)       // returns type name as string
```

---

## JSON

### `utils.toJSON(value)`
Serialize a value to a JSON string.

```ntl
val s = utils.toJSON({ name: "Alice", age: 30 })
// {"name":"Alice","age":30}
```

### `utils.fromJSON(str)`
Parse a JSON string into a value.

```ntl
val obj = utils.fromJSON('{"name":"Alice"}')
io.log(obj.name)
```

---

## Deep Operations

### `utils.deepClone(value)`
Create a deep copy of an object or array.

```ntl
val copy = utils.deepClone(original)
```

### `utils.deepEqual(a, b)`
Check if two values are deeply equal.

```ntl
if utils.deepEqual(a, b) {
  io.log("They are equal")
}
```

---

## Date & Time

### `utils.now()`
Returns the current Unix timestamp in milliseconds.

```ntl
val ts = utils.now()
```

### `utils.date([timestamp])`
Create a date object from a timestamp (or current time).

```ntl
val d = utils.date()
io.log(d.year, d.month, d.day)
io.log(d.format("YYYY-MM-DD"))
```

### `utils.sleep(ms)`
Pause execution for `ms` milliseconds.

```ntl
utils.sleep(1000)    // wait 1 second
```

---

## String Utilities

### `utils.trim(s)`
Remove leading and trailing whitespace.

### `utils.trimLeft(s)` / `utils.trimRight(s)`
Trim from one side only.

### `utils.pad(s, width, [char])`
Pad a string to a given width.

```ntl
utils.pad("42", 5, "0")    // "00042"
```

### `utils.repeat(s, n)`
Repeat a string `n` times.

```ntl
utils.repeat("ab", 3)    // "ababab"
```

### `utils.truncate(s, maxLen, [suffix])`
Truncate a string.

```ntl
utils.truncate("Hello World", 8)       // "Hello..."
utils.truncate("Hello World", 8, "…")  // "Hello W…"
```

### `utils.capitalize(s)`
Capitalize the first character.

```ntl
utils.capitalize("hello")    // "Hello"
```

### `utils.camelCase(s)`
Convert to camelCase.

```ntl
utils.camelCase("hello-world")    // "helloWorld"
```

### `utils.snakeCase(s)`
Convert to snake_case.

```ntl
utils.snakeCase("helloWorld")    // "hello_world"
```

---

## Array Utilities

### `utils.range(start, end, [step])`
Generate a range of numbers.

```ntl
val r = utils.range(0, 5)     // [0, 1, 2, 3, 4]
val r2 = utils.range(0, 10, 2) // [0, 2, 4, 6, 8]
```

### `utils.shuffle(arr)`
Randomly shuffle an array (returns a new array).

```ntl
val shuffled = utils.shuffle([1, 2, 3, 4, 5])
```

### `utils.unique(arr)`
Remove duplicates from an array.

```ntl
val u = utils.unique([1, 1, 2, 3, 2])    // [1, 2, 3]
```

### `utils.chunk(arr, size)`
Split an array into chunks of a given size.

```ntl
val chunks = utils.chunk([1, 2, 3, 4, 5], 2)
// [[1, 2], [3, 4], [5]]
```

### `utils.flatten(arr, [depth])`
Flatten a nested array.

```ntl
val flat = utils.flatten([[1, 2], [3, [4, 5]]])
// [1, 2, 3, 4, 5]
```

---

## Math

### `utils.clamp(value, min, max)`
Clamp a value between min and max.

```ntl
utils.clamp(15, 0, 10)    // 10
utils.clamp(-5, 0, 10)    // 0
```

### `utils.lerp(a, b, t)`
Linear interpolation.

```ntl
utils.lerp(0, 100, 0.5)    // 50
```

### `utils.round(n, [decimals])`
Round to a number of decimal places.

```ntl
utils.round(3.14159, 2)    // 3.14
```

---

## Example

```ntl
val utils = @import("std.utils")
val io = @import("std.io")

fn main() {
  val data = [5, 2, 8, 1, 9, 3]
  val sorted = data.sort()
  val top3 = sorted.slice(0, 3)
  io.log("Top 3:", top3)

  val name = "hello world"
  io.log(utils.capitalize(name))    // Hello world
  io.log(utils.camelCase(name))     // helloWorld

  val cloned = utils.deepClone({ x: 1, y: [2, 3] })
  io.log(utils.deepEqual(cloned, { x: 1, y: [2, 3] }))    // true
}
```
