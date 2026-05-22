# XML Module

XML parsing and generation utilities.

**Use case:** Parse and create XML documents.

---

## Import

```ntl
val xml = @import("std.xml")
```

---

## Available Functions

### `parse(data)`

Executes the `parse` operation with the given parameter (data).

**Signature:**
```ntl
fn parse(data)
```

### `stringify(obj, root, options)`

Executes the `stringify` operation with the given parameters (obj, root, options).

**Signature:**
```ntl
fn stringify(obj, root, options)
```

### `validate(data)`

Executes the `validate` operation with the given parameter (data).

**Signature:**
```ntl
fn validate(data)
```

### `query(data, xpath)`

Executes the `query` operation with the given parameters (data, xpath).

**Signature:**
```ntl
fn query(data, xpath)
```

### `getAttribute(element, name)`

Executes the `getAttribute` operation with the given parameters (element, name).

**Signature:**
```ntl
fn getAttribute(element, name)
```

### `getText(element)`

Executes the `getText` operation with the given parameter (element).

**Signature:**
```ntl
fn getText(element)
```

### `readFile(filePath)`

Executes the `readFile` operation with the given parameter (filePath).

**Signature:**
```ntl
fn readFile(filePath)
```

### `writeFile(filePath, obj, root, options)`

Executes the `writeFile` operation with the given parameters (filePath, obj, root, options).

**Signature:**
```ntl
fn writeFile(filePath, obj, root, options)
```

### `fromJSON(obj, root)`

Executes the `fromJSON` operation with the given parameters (obj, root).

**Signature:**
```ntl
fn fromJSON(obj, root)
```

### `toJSON(data)`

Executes the `toJSON` operation with the given parameter (data).

**Signature:**
```ntl
fn toJSON(data)
```

