# test — Unit Testing Module

The `test` module provides a lightweight testing framework with assertions, suites, and reporting.

## Import

```ntl
use test
```

---

## Defining Tests

### `test.run(name, fn)`
Register and immediately run a named test.

```ntl
test.run("addition works", fn() {
  test.eq(1 + 1, 2)
  test.eq(10 + 5, 15)
})
```

### `test.suite(name, fn)`
Group related tests in a suite.

```ntl
test.suite("Math", fn() {
  test.run("add", fn() {
    test.eq(1 + 2, 3)
  })
  test.run("subtract", fn() {
    test.eq(5 - 3, 2)
  })
})
```

---

## Assertions

### `test.eq(actual, expected)`
Assert strict equality.

```ntl
test.eq(42, 42)
test.eq("hello", "hello")
```

### `test.neq(actual, expected)`
Assert not equal.

```ntl
test.neq(1, 2)
```

### `test.ok(value)`
Assert value is truthy.

```ntl
test.ok(1 > 0)
test.ok(arr.length > 0)
```

### `test.fail(value)`
Assert value is falsy.

```ntl
test.fail(false)
test.fail(null)
```

### `test.deepEq(actual, expected)`
Assert deep equality (objects and arrays).

```ntl
test.deepEq([1, 2, 3], [1, 2, 3])
test.deepEq({ name: "Alice" }, { name: "Alice" })
```

### `test.throws(fn, [message])`
Assert that a function throws.

```ntl
test.throws(fn() {
  throw "oops"
})
```

### `test.notThrows(fn)`
Assert that a function does not throw.

### `test.match(str, pattern)`
Assert a string matches a regex pattern.

```ntl
test.match("hello world", "^hello")
```

### `test.type(value, typeName)`
Assert the type of a value.

```ntl
test.type(42, "number")
test.type("hi", "string")
test.type([], "array")
```

### `test.range(value, min, max)`
Assert a number is within a range.

```ntl
test.range(5, 0, 10)
```

### `test.null(value)`
Assert value is `null`.

### `test.defined(value)`
Assert value is not `undefined`.

---

## Setup & Teardown

### `test.beforeEach(fn)`
Run before every test in the current suite.

```ntl
test.suite("DB tests", fn() {
  val db = null

  test.beforeEach(fn() {
    db = createTestDB()
  })

  test.afterEach(fn() {
    db.clear()
  })

  test.run("insert works", fn() {
    db.insert({ name: "Alice" })
    test.eq(db.count(), 1)
  })
})
```

### `test.afterEach(fn)`
Run after every test in the current suite.

### `test.beforeAll(fn)`
Run once before all tests in the suite.

### `test.afterAll(fn)`
Run once after all tests in the suite.

---

## Reporting

### `test.report()`
Print a summary of all test results.

```ntl
test.report()
```

Output example:
```
  ✔ addition works            (0ms)
  ✔ Math / add                (0ms)
  ✔ Math / subtract           (0ms)
  ✗ divide by zero            ReferenceError: division by zero

  3 passed, 1 failed
```

### `test.summary()`
Return test stats as an object.

```ntl
val stats = test.summary()
io.log(stats.passed, "/", stats.total)
```

---

## Example: Testing a Function

```ntl
use test

fn add(a, b) { return a + b }
fn divide(a, b) {
  if b == 0 { throw "division by zero" }
  return a / b
}

test.suite("Math Functions", fn() {
  test.run("add returns correct sum", fn() {
    test.eq(add(2, 3), 5)
    test.eq(add(-1, 1), 0)
    test.eq(add(0, 0), 0)
  })

  test.run("divide works for non-zero denominators", fn() {
    test.eq(divide(10, 2), 5)
    test.eq(divide(9, 3), 3)
  })

  test.run("divide throws on zero", fn() {
    test.throws(fn() { divide(5, 0) })
  })
})

test.report()
```
