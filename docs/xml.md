# XML Module

XML parsing and generation utilities.

**Use case:** Parse and create XML documents.

---

## Import

```lunex
val xml = @import("std.xml")
```

---

## Available Functions

### `parse(data)`

Executes the `parse` operation with the given parameter (data).

**Signature:**
```lunex
fn parse(data)
```

### `stringify(obj, root, options)`

Executes the `stringify` operation with the given parameters (obj, root, options).

**Signature:**
```lunex
fn stringify(obj, root, options)
```

### `validate(data)`

Executes the `validate` operation with the given parameter (data).

**Signature:**
```lunex
fn validate(data)
```

### `query(data, xpath)`

Executes the `query` operation with the given parameters (data, xpath).

**Signature:**
```lunex
fn query(data, xpath)
```

### `getAttribute(element, name)`

Executes the `getAttribute` operation with the given parameters (element, name).

**Signature:**
```lunex
fn getAttribute(element, name)
```

### `getText(element)`

Executes the `getText` operation with the given parameter (element).

**Signature:**
```lunex
fn getText(element)
```

### `readFile(filePath)`

Executes the `readFile` operation with the given parameter (filePath).

**Signature:**
```lunex
fn readFile(filePath)
```

### `writeFile(filePath, obj, root, options)`

Executes the `writeFile` operation with the given parameters (filePath, obj, root, options).

**Signature:**
```lunex
fn writeFile(filePath, obj, root, options)
```

### `fromJSON(obj, root)`

Executes the `fromJSON` operation with the given parameters (obj, root).

**Signature:**
```lunex
fn fromJSON(obj, root)
```

### `toJSON(data)`

Executes the `toJSON` operation with the given parameter (data).

**Signature:**
```lunex
fn toJSON(data)
```

