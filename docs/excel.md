# ntl:excel

Read, write, and create Excel (.xlsx) spreadsheets.

## Import

```ntl
val excel = @import("std.excel")
```

## Top-level

### `excel.create()`

Creates a new in-memory workbook with a default sheet named `"Sheet1"`.

```ntl
val wb = excel.create()
```

### `excel.open(path)`

Opens an existing `.xlsx` file from disk.

```ntl
val wb = excel.open("data.xlsx")
```

### `excel.columnName(number)`

Converts a 1-based column number to its Excel letter name (`1 → "A"`, `26 → "Z"`, `27 → "AA"`).

```ntl
val name = excel.columnName(3)  // "C"
```

## Workbook methods

All methods below are on the object returned by `excel.create()` or `excel.open()`.

### Reading

| Method | Description |
|---|---|
| `wb.getCell(sheet, cell)` | Returns the value of a cell (e.g. `"A1"`) as a string |
| `wb.getRow(sheet, row)` | Returns an array of cell values for the given 1-based row number |
| `wb.getRows(sheet)` | Returns all rows as an array of arrays |
| `wb.getRowsAsObjects(sheet)` | Returns all rows as objects using the first row as header keys |
| `wb.getSheets()` | Returns an array of sheet names |

### Writing

| Method | Description |
|---|---|
| `wb.setCell(sheet, cell, value)` | Sets the value of a cell (e.g. `"B2"`) |
| `wb.writeRow(sheet, row, values)` | Writes an array of values into a row |
| `wb.setFormula(sheet, cell, formula)` | Sets a formula string (e.g. `"=SUM(A1:A10)"`) |
| `wb.mergeCell(sheet, hCell, vCell)` | Merges cells from `hCell` to `vCell` |
| `wb.setColWidth(sheet, col, width)` | Sets the width of a column letter |

### Formatting

| Method | Description |
|---|---|
| `wb.setStyle(sheet, cell, style)` | Applies a style object to a cell |

Style object fields: `bold`, `italic`, `fontSize`, `fontColor`, `bgColor`, `align` (`"left"`, `"center"`, `"right"`), `border`.

### Sheet management

| Method | Description |
|---|---|
| `wb.newSheet(name)` | Creates a new sheet |
| `wb.deleteSheet(name)` | Deletes a sheet |

### Persistence

| Method | Description |
|---|---|
| `wb.save(path)` | Saves the workbook to disk |
| `wb.close()` | Closes and releases the workbook |

## Example

```ntl
val wb = excel.create()
wb.setCell("Sheet1", "A1", "Name")
wb.setCell("Sheet1", "B1", "Score")
wb.writeRow("Sheet1", 2, ["Alice", 98])
wb.writeRow("Sheet1", 3, ["Bob", 87])
wb.save("report.xlsx")
wb.close()
```
