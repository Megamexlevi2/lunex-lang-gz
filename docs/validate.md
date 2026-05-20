# validate — Validation Module

The `validate` module provides schema-based validation for objects, strings, numbers, and arrays.

## Import

```ntl
val validate = @import("std.validate")
```

---

## Validating Values

### `validate.check(schema, value)`
Validate a value against a schema. Returns `{ ok: bool, errors: [...] }`.

```ntl
val result = validate.check({
  name:  { type: "string", required: true, minLen: 2 },
  age:   { type: "number", min: 0, max: 150 },
  email: { type: "string", format: "email" },
}, {
  name: "Alice",
  age: 30,
  email: "alice@example.com",
})

if !result.ok {
  each err in result.errors {
    io.error(err)
  }
}
```

### `validate.assert(schema, value)`
Like `check` but throws if validation fails.

```ntl
validate.assert({ name: { type: "string", required: true } }, body)
```

---

## Schema Fields

### Common Options

| Option | Type | Description |
|---|---|---|
| `type` | string | `"string"`, `"number"`, `"boolean"`, `"array"`, `"object"`, `"any"` |
| `required` | boolean | Value must be present and not null |
| `default` | any | Default value if not provided |
| `nullable` | boolean | Allow `null` values |

### String Options

| Option | Description |
|---|---|
| `minLen` | Minimum string length |
| `maxLen` | Maximum string length |
| `pattern` | Regex pattern the value must match |
| `format` | Built-in format: `"email"`, `"url"`, `"uuid"`, `"date"`, `"phone"` |
| `enum` | Array of allowed values |

```ntl
{ type: "string", format: "email", required: true }
{ type: "string", enum: ["admin", "user", "guest"] }
{ type: "string", pattern: "^[a-z]+$", maxLen: 20 }
```

### Number Options

| Option | Description |
|---|---|
| `min` | Minimum value |
| `max` | Maximum value |
| `integer` | Must be an integer |
| `positive` | Must be positive |

### Array Options

| Option | Description |
|---|---|
| `minItems` | Minimum array length |
| `maxItems` | Maximum array length |
| `items` | Schema for each element |
| `unique` | All items must be unique |

```ntl
{ type: "array", items: { type: "number", positive: true }, minItems: 1 }
```

### Object Options

| Option | Description |
|---|---|
| `fields` | Map of field name → schema |
| `strict` | Reject unknown fields |

---

## Built-in Validators

### `validate.isEmail(s)`
Returns `true` if `s` is a valid email address.

```ntl
validate.isEmail("alice@example.com")    // true
```

### `validate.isURL(s)`
Returns `true` if `s` is a valid URL.

### `validate.isUUID(s)`
Returns `true` if `s` is a valid UUID.

### `validate.isPhone(s)`
Returns `true` if `s` looks like a phone number.

### `validate.isDate(s)`
Returns `true` if `s` is a parseable date string.

---

## Composing Schemas

```ntl
val userSchema = {
  id:       { type: "number", integer: true, positive: true },
  name:     { type: "string", required: true, minLen: 1, maxLen: 100 },
  email:    { type: "string", required: true, format: "email" },
  role:     { type: "string", enum: ["admin", "user"], default: "user" },
  tags:     { type: "array", items: { type: "string" } },
  active:   { type: "boolean", default: true },
}

fn createUser(data) {
  val result = validate.check(userSchema, data)
  if !result.ok {
    throw "ValidationError: " + result.errors.join(", ")
  }
  // ...
}
```

---

## Example: API Request Validation

```ntl
val validate = @import("std.validate")
val http = @import("std.http")
val io = @import("std.io")

val loginSchema = {
  username: { type: "string", required: true, minLen: 3 },
  password: { type: "string", required: true, minLen: 8 },
}

val app = http.router()

app.post("/login", fn(req, res) {
  val result = validate.check(loginSchema, req.body)
  if !result.ok {
    res.status(400).json({ errors: result.errors })
  }
  res.json({ token: "abc123" })
})

fn main() {
  http.serve(3000, app.handler())
}
```
