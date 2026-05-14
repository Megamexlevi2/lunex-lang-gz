# ntl:csv / ntl:yaml / ntl:toml / ntl:markdown / ntl:mustache

Data format parsers and serializers. Each format is available as its own module.

---

## ntl:csv

```ntl
use csv
```

### `csv.parse(text, options?)`

Parses a CSV string into an array of row objects.

Options:

| Field | Default | Description |
|---|---|---|
| `separator` | `","` | Column separator character |
| `header` | `true` | Use the first row as object keys |

```ntl
val rows = csv.parse("name,age\nAlice,30\nBob,25")
io.log(rows[0].name)  // "Alice"
```

### `csv.stringify(rows, options?)`

Converts an array of objects or arrays into a CSV string.

```ntl
val text = csv.stringify([{ name: "Alice", age: 30 }])
```

### `csv.readFile(path, options?)`

Reads and parses a CSV file from disk.

```ntl
val rows = csv.readFile("data.csv")
```

### `csv.writeFile(path, rows, options?)`

Writes an array of rows to a CSV file on disk.

```ntl
csv.writeFile("output.csv", rows)
```

---

## ntl:yaml

```ntl
use yaml
```

### `yaml.parse(text)`

Parses a YAML string into an NTL value.

```ntl
val config = yaml.parse("name: Alice\nage: 30")
```

### `yaml.stringify(value)`

Serializes an NTL value to a YAML string.

### `yaml.readFile(path)`

Reads and parses a YAML file from disk.

### `yaml.writeFile(path, value)`

Serializes a value and writes it to a YAML file.

---

## ntl:toml

```ntl
use toml
```

### `toml.parse(text)`

Parses a TOML string into an NTL value.

```ntl
val config = toml.parse('[server]\nport = 8080')
io.log(config.server.port)  // 8080
```

### `toml.stringify(value)`

Serializes an NTL value to a TOML string.

### `toml.readFile(path)`

Reads and parses a TOML file from disk.

### `toml.writeFile(path, value)`

Serializes a value and writes it to a TOML file.

---

## ntl:markdown

```ntl
use markdown
```

### `markdown.toHTML(text, options?)`

Converts a Markdown string to HTML. Options:

| Field | Default | Description |
|---|---|---|
| `unsafe` | `false` | Allow raw HTML in input |
| `hardWraps` | `false` | Treat newlines as `<br>` |

```ntl
val html = markdown.toHTML("# Hello\n**World**")
```

### `markdown.readFile(path)`

Reads a Markdown file from disk and returns the raw string.

### `markdown.renderFile(path, options?)`

Reads a Markdown file and returns it as HTML.

---

## ntl:mustache

```ntl
use mustache
```

### `mustache.render(template, data)`

Renders a Mustache template string with the given data object.

```ntl
val html = mustache.render("Hello, {{name}}!", { name: "Alice" })
```

### `mustache.renderFile(path, data)`

Reads a `.mustache` file from disk and renders it with `data`.

```ntl
val page = mustache.renderFile("views/home.mustache", { title: "Home" })
```
