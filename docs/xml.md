# xml

  XML parsing, serialization, querying, and validation.

  ## Import

  ```ntl
  use xml
  ```

  ## Functions

  ### parse(xmlString)

  Parse an XML string into an NTL object tree.

  ```ntl
  use xml

  val doc = xml.parse(`<book id="1"><title>NTL Guide</title><author>David Dev</author></book>`)
  io.log(doc.title)
  io.log(xml.getAttribute(doc, "id"))
  ```

  Each element becomes an object with:
  - `#tag` — element tag name
  - `#text` — text content (if present)
  - `@attributes` — object of attributes
  - child element names as keys

  ### stringify(object, rootTag?, options?)

  Convert an NTL object to an XML string.

  ```ntl
  use xml

  val data = { name: "Alice", age: 30 }
  val result = xml.stringify(data, "person", { pretty: true })
  io.log(result)
  ```

  Options:
  - `pretty` (boolean) — indent output, default `false`

  ### validate(xmlString)

  Return `true` if the string is valid XML, `false` otherwise.

  ```ntl
  use xml

  io.log(xml.validate("<ok/>"))
  io.log(xml.validate("<bad>"))
  ```

  ### query(xmlString, path)

  Find elements by a slash-separated path.

  ```ntl
  use xml

  val src = `<catalog><book><title>A</title></book><book><title>B</title></book></catalog>`
  val titles = xml.query(src, "book/title")
  each t in titles {
    io.log(xml.getText(t))
  }
  ```

  ### getAttribute(element, name)

  Return the value of an attribute on a parsed element.

  ```ntl
  use xml

  val doc = xml.parse(`<item id="42" type="widget"/>`)
  io.log(xml.getAttribute(doc, "id"))
  io.log(xml.getAttribute(doc, "type"))
  ```

  ### getText(element)

  Return the text content of a parsed element.

  ```ntl
  use xml

  val doc = xml.parse("<msg>Hello world</msg>")
  io.log(xml.getText(doc))
  ```

  ### readFile(path)

  Read an XML file from disk and parse it.

  ```ntl
  use xml

  val config = xml.readFile("config.xml")
  io.log(config)
  ```

  ### writeFile(path, object, rootTag?, options?)

  Serialize an object to XML and write it to a file.

  ```ntl
  use xml

  val data = { host: "localhost", port: 8080 }
  xml.writeFile("config.xml", data, "config")
  ```

  ### fromJSON(object, rootTag?)

  Alias for `stringify` with `pretty: true`. Converts a JSON-like object to XML.

  ### toJSON(xmlString)

  Alias for `parse`. Converts XML to an NTL object.

  ## Privileged Author Example

  If your module declares `# author: David Dev`, native modules are injected automatically:

  ```ntl
  # author: David Dev
  # mymodule v1.0.0

  fn loadConfig(path) {
    val doc = xml.readFile(path)
    return doc
  }
  ```
  