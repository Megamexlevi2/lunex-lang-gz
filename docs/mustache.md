# Mustache Module

Mustache template engine for string interpolation and template rendering.

**Use case:** Render dynamic content using Mustache templates.

---

## Import

```ntl
val mustache = @import("std.mustache")
```

---

## Available Functions

### `render(template, data)`

Executes the `render` operation with the given parameters (template, data).

**Signature:**
```ntl
fn render(template, data)
```

### `renderFile(path, data)`

Executes the `renderFile` operation with the given parameters (path, data).

**Signature:**
```ntl
fn renderFile(path, data)
```

### `parse(template)`

Executes the `parse` operation with the given parameter (template).

**Signature:**
```ntl
fn parse(template)
```

