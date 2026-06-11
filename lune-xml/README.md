# lunex-xml

XML module for the [Lunex](https://github.com/Megamexlevi2) programming language.  
---

## Installation

Copy the `lunex-xml` folder into your project and import:

```lx
val xml = @fimport("./lunex-xml/main.nax")   // compiled bundle (recommended)
val xml = @fimport("./lunex-xml/main.lx")    // source
```

---

## Quick Start

```lx
val io  = @import("std.io")
val xml = @fimport("./lunex-xml/main.nax")

fn main() {
  // Build a document
  val catalog = xml.createElement("catalog")
  xml.setAttribute(catalog, "version", "1.0")

  val book = xml.createElementWithAttrs("book", { id: "1", lang: "en" })
  xml.appendChild(book, xml.createTextElement("title", "Lunex Programming"))
  xml.appendChild(book, xml.createTextElement("author", "David Dev"))
  xml.appendChild(catalog, book)

  io.log(xml.build(catalog))

  // Parse it back and query it
  val doc   = xml.parse(xml.build(catalog))
  val title = xml.selectFirst(doc, "title")
  io.log("Title:", xml.text(title))
  io.log("Book id:", xml.attr(xml.selectFirst(doc, "book"), "id"))
}
```

Output:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<catalog version="1.0">
  <book id="1" lang="en">
    <title>Lunex Programming</title>
    <author>David Dev</author>
  </book>
</catalog>
Title: Lunex Programming
Book id: 1
```

---

## Node Structure

Every XML element is an `XmlNode` struct with the following fields and methods:

### Fields (direct access)

| Field      | Type   | Description                              |
|------------|--------|------------------------------------------|
| `tag`      | string | Element tag name, e.g. `"book"`          |
| `attrs`    | object | Key-value map of attributes              |
| `children` | array  | Child `XmlNode` structs                  |
| `text`     | string | Text content — set only on leaf nodes    |

### Methods (on the struct)

| Method                    | Description                                  |
|---------------------------|----------------------------------------------|
| `setAttribute(k, v)`      | Set attribute `k` to value `v`               |
| `getAttribute(k)`         | Get attribute value, or `null` if missing    |
| `hasAttr(k)`              | Returns `true` if attribute `k` exists       |
| `appendChild(child)`      | Add a child node                             |
| `hasChildren()`           | Returns `true` if there are child nodes      |
| `childAt(i)`              | Returns child at index `i`                   |
| `childCount()`            | Returns number of children                   |
| `setText(t)`              | Set text content                             |
| `getText()`               | Get text content                             |
| `getTag()`                | Get tag name                                 |
| `toString()`              | Returns `"XmlNode<tagname>"` for debugging   |

You can call these methods directly on any node returned by the API:

```lx
val n = xml.createElement("div")
n.setAttribute("class", "box")
io.log(n.getAttribute("class"))  // → box
io.log(n.hasAttr("id"))          // → false
io.log(n.childCount())           // → 0
```

---

## API Reference

### Building XML

#### `xml.createElement(tag)` → node
Creates an empty element with no attributes, children, or text.

```lx
val section = xml.createElement("section")
```

#### `xml.createTextElement(tag, text)` → node
Creates an element with text content.

```lx
val title = xml.createTextElement("title", "Hello World")
// → <title>Hello World</title>
```

#### `xml.createElementWithAttrs(tag, attrs)` → node
Creates an element and sets multiple attributes at once.

```lx
val img = xml.createElementWithAttrs("img", { src: "photo.png", alt: "photo" })
// → <img src="photo.png" alt="photo" />
```

#### `xml.appendChild(node, child)`
Appends a child node. Same as calling `node.appendChild(child)`.

```lx
xml.appendChild(root, child)
```

#### `xml.setAttribute(node, key, value)`
Sets an attribute. Same as `node.setAttribute(key, value)`.

```lx
xml.setAttribute(node, "id", "main")
```

#### `xml.getAttribute(node, key)` → string | null
Gets an attribute value. Returns `null` if the attribute is missing.

```lx
val id = xml.getAttribute(node, "id")
```

#### `xml.hasChildren(node)` → bool
Returns `true` if the node has at least one child.

#### `xml.build(root)` → string
Serializes the full tree to an XML string with a `<?xml ...?>` header.

```lx
val output = xml.build(root)
```

#### `xml.buildFragment(node)` → string
Serializes a node without the XML header. Good for embedding snippets.

```lx
val snippet = xml.buildFragment(node)
```

---

### Parsing

#### `xml.parse(src)` → node
Parses an XML string and returns the root `XmlNode`.  
Handles: attributes, text content, self-closing tags (`<br />`), and the `<?xml ...?>` declaration.

```lx
val doc = xml.parse("<users><user id=\"1\"><name>Alice</name></user></users>")
io.log(doc.tag)              // → users
io.log(doc.childCount())     // → 1
```

> **Round-trip:** `xml.parse(xml.build(node))` always produces an equivalent tree.

---

### Querying

#### `xml.selectFirst(root, tag)` → node | null
Returns the first node anywhere in the tree that matches `tag` (depth-first).  
Returns `null` if not found.

```lx
val title = xml.selectFirst(doc, "title")
```

#### `xml.selectAll(root, tag)` → array
Returns all nodes in the tree matching `tag`.

```lx
val items = xml.selectAll(doc, "item")
each item in items {
  io.log(xml.text(item))
}
```

#### `xml.selectByAttr(root, attrName, attrValue)` → array
Returns all nodes where `attrs[attrName] == attrValue`.

```lx
val admins = xml.selectByAttr(doc, "role", "admin")
```

#### `xml.selectPath(root, path)` → node | null
Navigates a dot-separated path starting from the root tag.  
Returns the first match at the end of the path.

```lx
// path must start with the root tag
val price = xml.selectPath(doc, "catalog.book.price")
```

#### `xml.children(node)` → array
Returns only the **direct** children of a node (not recursive).

```lx
val direct = xml.children(root)
```

#### `xml.childrenByTag(node, tag)` → array
Returns direct children that match a specific tag.

```lx
val chapters = xml.childrenByTag(book, "chapter")
```

#### `xml.text(node)` → string
Returns the text content of a node.

```lx
val content = xml.text(titleNode)
```

#### `xml.attr(node, name)` → string | null
Shorthand for `getAttribute`. Returns an attribute value.

```lx
val id = xml.attr(node, "id")
```

---

### Transforming

#### `xml.clone(node)` → node
Deep-clones a node and all its descendants. Mutations on the clone do not affect the original.

```lx
val copy = xml.clone(original)
```

#### `xml.walk(root, fn)`
Visits every node in the tree depth-first, calling `fn(node)` on each one.

```lx
xml.walk(doc, fn(n) {
  io.log(n.tag)
})
```

#### `xml.mapNodes(root, fn)` → node | null
Transforms every node by passing it through `fn(node)`.  
Return the node to keep it; return `null` to remove it from the tree.

```lx
// Remove all <draft> elements from the tree
val clean = xml.mapNodes(doc, fn(n) {
  if n.tag == "draft" { null } else { n }
})
```

#### `xml.filterNodes(root, predicate)` → node
Removes all children (recursively) for which `predicate(node)` returns `false`.  
Mutates the tree in place and returns root.

```lx
// Keep only <item> nodes with type="good"
xml.filterNodes(doc, fn(n) {
  n.tag == "list" or xml.attr(n, "type") == "good"
})
```

#### `xml.toObject(root)` → object
Converts an XML tree to a plain Lunex object.  
Text nodes become strings. Repeated sibling tags become arrays.

```lx
val obj = xml.toObject(doc)
// <users><user><name>Alice</name></user><user><name>Bob</name></user></users>
// → { user: [ { name: "Alice" }, { name: "Bob" } ] }
```

#### `xml.fromObject(tag, obj)` → node
Converts a plain Lunex object to an XML tree.  
Arrays produce repeated sibling elements with the same tag.

```lx
val node = xml.fromObject("config", {
  host: "localhost"
  port: "3000"
})
// → <config><host>localhost</host><port>3000</port></config>

val list = xml.fromObject("items", { item: ["a", "b", "c"] })
// → <items><item>a</item><item>b</item><item>c</item></items>
```

---

### Validating

All validation functions return `{ ok: bool, error: string }`.  
On success: `ok = true`, `error = ""`.  
On failure: `ok = false`, `error` contains a human-readable message.

#### `xml.isWellFormed(node)` → result
Checks that the node and all descendants have non-empty tag names.

```lx
val res = xml.isWellFormed(doc)
unless res.ok {
  io.error(res.error)
}
```

#### `xml.validateSchema(node, schema)` → result
Validates a single node against a schema object.

**Schema fields:**

| Field              | Type   | Description                                       |
|--------------------|--------|---------------------------------------------------|
| `tag`              | string | Node must have this exact tag name                |
| `requiredAttrs`    | array  | These attribute names must be present             |
| `requiredChildren` | array  | These child tag names must exist                  |
| `minChildren`      | number | Node must have at least this many children        |
| `maxChildren`      | number | Node must have at most this many children         |
| `requireText`      | bool   | Node must have non-empty text content             |
| `noText`           | bool   | Node must NOT have text content                   |

```lx
val res = xml.validateSchema(node, {
  tag:              "book"
  requiredAttrs:    ["id"]
  requiredChildren: ["title", "author"]
})

if res.ok {
  io.log("valid!")
} else {
  io.error(res.error)
}
```

#### `xml.validateTree(root, schemas)` → result
Validates every node in the tree using a map of `tag → schema`.  
Tags not present in the map are skipped.  
Stops and returns the first error found.

```lx
val schemas = {
  catalog: { requiredChildren: ["book"] }
  book:    { requiredAttrs: ["id"], requiredChildren: ["title"] }
  title:   { requireText: true }
}

val res = xml.validateTree(doc, schemas)
if res.ok {
  io.log("document is valid")
} else {
  io.error(res.error)
}
```

---

## Running Tests

```bash
lunex run test/run_all.lx
```

6 suites: **node**, **builder**, **parser**, **query**, **transform**, **validate**.

---

## Project Structure

```
lunex-xml/
├── main.lx            Public module entry point
├── main.nax           Compiled bundle — use this for @fimport
├── config.lx          Project manifest
├── README.md          This file
├── src/
│   ├── node.lx        XmlNode struct (tag, attrs, children, text + methods)
│   ├── builder.lx     XML serializer (build, buildFragment, createElement...)
│   ├── parser.lx      XML parser (parse)
│   ├── query.lx       Selectors (selectFirst, selectAll, selectPath...)
│   ├── transform.lx   clone, walk, mapNodes, filterNodes, toObject, fromObject
│   └── validate.lx    isWellFormed, validateSchema, validateTree
└── test/
    ├── run_all.lx
    ├── test_node.lx
    ├── test_builder.lx
    ├── test_parser.lx
    ├── test_query.lx
    ├── test_transform.lx
    └── test_validate.lx
```

---

## License

license: Apache-2.0


 — by David Dev · [github.com/Megamexlevi2](https://github.com/Megamexlevi2)
