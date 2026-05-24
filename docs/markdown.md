# Markdown Module

Markdown parsing and conversion to HTML.

**Use case:** Convert Markdown content to HTML for rendering.

---

## Import

```lunex
val markdown = @import("std.markdown")
```

---

## Available Functions

### `toHTML(content)`

Executes the `toHTML` operation with the given parameter (content).

**Signature:**
```lunex
fn toHTML(content)
```

### `parse(content)`

Executes the `parse` operation with the given parameter (content).

**Signature:**
```lunex
fn parse(content)
```

### `readFile(path)`

Executes the `readFile` operation with the given parameter (path).

**Signature:**
```lunex
fn readFile(path)
```

### `renderFile(inputPath, outputPath)`

Executes the `renderFile` operation with the given parameters (inputPath, outputPath).

**Signature:**
```lunex
fn renderFile(inputPath, outputPath)
```

