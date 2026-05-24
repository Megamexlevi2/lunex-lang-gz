# Mustache Module

Mustache template engine for string interpolation and template rendering.

**Use case:** Render dynamic content using Mustache templates.

---

## Import

```lunex
val mustache = @import("std.mustache")
```

---

## Available Functions

### `render(template, data)`

Executes the `render` operation with the given parameters (template, data).

**Signature:**
```lunex
fn render(template, data)
```

### `renderFile(path, data)`

Executes the `renderFile` operation with the given parameters (path, data).

**Signature:**
```lunex
fn renderFile(path, data)
```

### `parse(template)`

Executes the `parse` operation with the given parameter (template).

**Signature:**
```lunex
fn parse(template)
```

