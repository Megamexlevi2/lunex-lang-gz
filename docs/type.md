# Type Module

Type checking and runtime type utilities for type validation.

**Use case:** Check types at runtime and validate data types.

---

## Import

```ntl
val type = @import("std.type")
```

---

## Available Functions

### `isString(x)`

Executes the `isString` operation with the given parameter (x).

**Signature:**
```ntl
fn isString(x)
```

### `isNumber(x)`

Executes the `isNumber` operation with the given parameter (x).

**Signature:**
```ntl
fn isNumber(x)
```

### `isBool(x)`

Executes the `isBool` operation with the given parameter (x).

**Signature:**
```ntl
fn isBool(x)
```

### `isBoolean(x)`

Executes the `isBoolean` operation with the given parameter (x).

**Signature:**
```ntl
fn isBoolean(x)
```

### `isArray(x)`

Executes the `isArray` operation with the given parameter (x).

**Signature:**
```ntl
fn isArray(x)
```

### `isObject(x)`

Executes the `isObject` operation with the given parameter (x).

**Signature:**
```ntl
fn isObject(x)
```

### `isNull(x)`

Executes the `isNull` operation with the given parameter (x).

**Signature:**
```ntl
fn isNull(x)
```

### `isUndefined(x)`

Executes the `isUndefined` operation with the given parameter (x).

**Signature:**
```ntl
fn isUndefined(x)
```

### `isFunction(x)`

Executes the `isFunction` operation with the given parameter (x).

**Signature:**
```ntl
fn isFunction(x)
```

### `typeOf(x)`

Executes the `typeOf` operation with the given parameter (x).

**Signature:**
```ntl
fn typeOf(x)
```

### `isInt(x)`

Executes the `isInt` operation with the given parameter (x).

**Signature:**
```ntl
fn isInt(x)
```

### `isFloat(x)`

Executes the `isFloat` operation with the given parameter (x).

**Signature:**
```ntl
fn isFloat(x)
```

### `isNaN(x)`

Executes the `isNaN` operation with the given parameter (x).

**Signature:**
```ntl
fn isNaN(x)
```

### `isFinite(x)`

Executes the `isFinite` operation with the given parameter (x).

**Signature:**
```ntl
fn isFinite(x)
```

### `isDate(x)`

Executes the `isDate` operation with the given parameter (x).

**Signature:**
```ntl
fn isDate(x)
```

### `toInt(v)`

Executes the `toInt` operation with the given parameter (v).

**Signature:**
```ntl
fn toInt(v)
```

### `toFloat(v)`

Executes the `toFloat` operation with the given parameter (v).

**Signature:**
```ntl
fn toFloat(v)
```

### `toBool(v)`

Executes the `toBool` operation with the given parameter (v).

**Signature:**
```ntl
fn toBool(v)
```

### `toString(v)`

Executes the `toString` operation with the given parameter (v).

**Signature:**
```ntl
fn toString(v)
```

### `toArray(v)`

Executes the `toArray` operation with the given parameter (v).

**Signature:**
```ntl
fn toArray(v)
```

### `toObject(v)`

Executes the `toObject` operation with the given parameter (v).

**Signature:**
```ntl
fn toObject(v)
```

### `cast(value, targetType)`

Executes the `cast` operation with the given parameters (value, targetType).

**Signature:**
```ntl
fn cast(value, targetType)
```

### `assertString(v, name)`

Executes the `assertString` operation with the given parameters (v, name).

**Signature:**
```ntl
fn assertString(v, name)
```

### `assertNumber(v, name)`

Executes the `assertNumber` operation with the given parameters (v, name).

**Signature:**
```ntl
fn assertNumber(v, name)
```

### `assertArray(v, name)`

Executes the `assertArray` operation with the given parameters (v, name).

**Signature:**
```ntl
fn assertArray(v, name)
```

### `assertObject(v, name)`

Executes the `assertObject` operation with the given parameters (v, name).

**Signature:**
```ntl
fn assertObject(v, name)
```

### `assertBool(v, name)`

Executes the `assertBool` operation with the given parameters (v, name).

**Signature:**
```ntl
fn assertBool(v, name)
```

### `nullable(v, defaultVal)`

Executes the `nullable` operation with the given parameters (v, defaultVal).

**Signature:**
```ntl
fn nullable(v, defaultVal)
```

