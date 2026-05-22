# Validation Module

Data validation schemas and validators for input validation.

**Use case:** Validate user input and data structures.

---

## Import

```ntl
val validate = @import("std.validate")
```

---

## Available Functions

### `isString(v)`

Executes the `isString` operation with the given parameter (v).

**Signature:**
```ntl
fn isString(v)
```

### `isNumber(v)`

Executes the `isNumber` operation with the given parameter (v).

**Signature:**
```ntl
fn isNumber(v)
```

### `isBoolean(v)`

Executes the `isBoolean` operation with the given parameter (v).

**Signature:**
```ntl
fn isBoolean(v)
```

### `isArray(v)`

Executes the `isArray` operation with the given parameter (v).

**Signature:**
```ntl
fn isArray(v)
```

### `isObject(v)`

Executes the `isObject` operation with the given parameter (v).

**Signature:**
```ntl
fn isObject(v)
```

### `isNull(v)`

Executes the `isNull` operation with the given parameter (v).

**Signature:**
```ntl
fn isNull(v)
```

### `isDefined(v)`

Executes the `isDefined` operation with the given parameter (v).

**Signature:**
```ntl
fn isDefined(v)
```

### `isEmail(v)`

Executes the `isEmail` operation with the given parameter (v).

**Signature:**
```ntl
fn isEmail(v)
```

### `isUrl(v)`

Executes the `isUrl` operation with the given parameter (v).

**Signature:**
```ntl
fn isUrl(v)
```

### `isPhone(v)`

Executes the `isPhone` operation with the given parameter (v).

**Signature:**
```ntl
fn isPhone(v)
```

### `isUUID(v)`

Executes the `isUUID` operation with the given parameter (v).

**Signature:**
```ntl
fn isUUID(v)
```

### `isIPv4(v)`

Executes the `isIPv4` operation with the given parameter (v).

**Signature:**
```ntl
fn isIPv4(v)
```

### `isIPv6(v)`

Executes the `isIPv6` operation with the given parameter (v).

**Signature:**
```ntl
fn isIPv6(v)
```

### `isIP(v)`

Executes the `isIP` operation with the given parameter (v).

**Signature:**
```ntl
fn isIP(v)
```

### `isAlpha(v)`

Executes the `isAlpha` operation with the given parameter (v).

**Signature:**
```ntl
fn isAlpha(v)
```

### `isAlphanumeric(v)`

Executes the `isAlphanumeric` operation with the given parameter (v).

**Signature:**
```ntl
fn isAlphanumeric(v)
```

### `isNumeric(v)`

Executes the `isNumeric` operation with the given parameter (v).

**Signature:**
```ntl
fn isNumeric(v)
```

### `isHex(v)`

Executes the `isHex` operation with the given parameter (v).

**Signature:**
```ntl
fn isHex(v)
```

### `isBase64(v)`

Executes the `isBase64` operation with the given parameter (v).

**Signature:**
```ntl
fn isBase64(v)
```

### `isJSON(v)`

Executes the `isJSON` operation with the given parameter (v).

**Signature:**
```ntl
fn isJSON(v)
```

### `isCreditCard(v)`

Executes the `isCreditCard` operation with the given parameter (v).

**Signature:**
```ntl
fn isCreditCard(v)
```

### `isSlug(v)`

Executes the `isSlug` operation with the given parameter (v).

**Signature:**
```ntl
fn isSlug(v)
```

### `isDate(v)`

Executes the `isDate` operation with the given parameter (v).

**Signature:**
```ntl
fn isDate(v)
```

### `isStrongPassword(v)`

Executes the `isStrongPassword` operation with the given parameter (v).

**Signature:**
```ntl
fn isStrongPassword(v)
```

### `test(v, pattern)`

Executes the `test` operation with the given parameters (v, pattern).

**Signature:**
```ntl
fn test(v, pattern)
```

### `schema(definition)`

Executes the `schema` operation with the given parameter (definition).

**Signature:**
```ntl
fn schema(definition)
```

### `validate(data, rules)`

Executes the `validate` operation with the given parameters (data, rules).

**Signature:**
```ntl
fn validate(data, rules)
```

### `required(v)`

Executes the `required` operation with the given parameter (v).

**Signature:**
```ntl
fn required(v)
```

### `minLength(v, n)`

Executes the `minLength` operation with the given parameters (v, n).

**Signature:**
```ntl
fn minLength(v, n)
```

### `maxLength(v, n)`

Executes the `maxLength` operation with the given parameters (v, n).

**Signature:**
```ntl
fn maxLength(v, n)
```

### `inRange(n, lo, hi)`

Executes the `inRange` operation with the given parameters (n, lo, hi).

**Signature:**
```ntl
fn inRange(n, lo, hi)
```

