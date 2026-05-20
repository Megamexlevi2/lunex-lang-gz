# ntl:pdf

Generate PDF documents from NTL code.

## Import

```ntl
val pdf = @import("std.pdf")
```

## `pdf.create(options?)`

Creates a new PDF document. Optional `options` object:

| Field | Default | Description |
|---|---|---|
| `orientation` | `"portrait"` | `"portrait"` or `"landscape"` |
| `unit` | `"mm"` | Units: `"mm"`, `"cm"`, `"in"`, `"pt"` |
| `size` | `"A4"` | Page size: `"A4"`, `"Letter"`, `"A3"`, etc. |

```ntl
val doc = pdf.create({ orientation: "landscape", size: "A4" })
```

## Document methods

### Pages

| Method | Description |
|---|---|
| `doc.addPage()` | Adds a new page |

### Fonts

| Method | Description |
|---|---|
| `doc.setFont(family, style?)` | Sets the font family (`"Arial"`, `"Helvetica"`, `"Courier"`, etc.) and optional style (`"B"`, `"I"`, `"BI"`) |
| `doc.setFontSize(size)` | Sets font size in points |

### Colors

| Method | Description |
|---|---|
| `doc.setTextColor(r, g, b)` | Sets text color (0–255 RGB) |
| `doc.setFillColor(r, g, b)` | Sets fill color for shapes |
| `doc.setDrawColor(r, g, b)` | Sets draw/stroke color |

### Text

| Method | Description |
|---|---|
| `doc.cell(w, h, text, border?, align?, fill?)` | Draws a cell with text |
| `doc.multiCell(w, h, text, border?, align?, fill?)` | Draws a multi-line cell with word wrapping |

`align`: `"L"` (left), `"C"` (center), `"R"` (right). `border`: `0` or `1`.

### Shapes

| Method | Description |
|---|---|
| `doc.line(x1, y1, x2, y2)` | Draws a line |
| `doc.rect(x, y, w, h, style?)` | Draws a rectangle; style: `"D"` (outline), `"F"` (fill), `"DF"` (both) |

### Position

| Method | Description |
|---|---|
| `doc.setXY(x, y)` | Sets the current cursor position |
| `doc.setX(x)` | Sets the horizontal cursor position |
| `doc.setY(y)` | Sets the vertical cursor position |
| `doc.getX()` | Returns current X position |
| `doc.getY()` | Returns current Y position |
| `doc.setMargins(left, top, right?)` | Sets page margins |

### Images

| Method | Description |
|---|---|
| `doc.image(path, x, y, w?, h?)` | Embeds an image (PNG/JPG) at the given coordinates |

### Metadata

| Method | Description |
|---|---|
| `doc.setTitle(title)` | Sets the PDF title metadata |
| `doc.setAuthor(author)` | Sets the PDF author metadata |

### Output

| Method | Description |
|---|---|
| `doc.save(path)` | Saves the PDF to disk |

## Example

```ntl
val doc = pdf.create()
doc.setFont("Helvetica", "B")
doc.setFontSize(20)
doc.cell(0, 10, "Sales Report", 0, "C")
doc.addPage()
doc.setFont("Arial")
doc.setFontSize(12)
doc.cell(40, 8, "Product", 1, "L")
doc.cell(40, 8, "Revenue", 1, "R")
doc.save("report.pdf")
```
