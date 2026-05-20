# fs — File System Module

The `fs` module provides file and directory operations: reading, writing, listing, copying, moving, and watching files.

## Import

```ntl
val fs = @import("std.fs")
```

---

## Reading Files

### `fs.read(path)`
Read the entire content of a file as a string.

```ntl
val content = fs.read("config.txt")
io.log(content)
```

### `fs.readBytes(path)`
Read file as an array of bytes.

```ntl
val bytes = fs.readBytes("data.bin")
```

### `fs.readLines(path)`
Read file and return an array of lines.

```ntl
val lines = fs.readLines("data.csv")
each line in lines {
  io.log(line)
}
```

### `fs.readJSON(path)`
Read and parse a JSON file.

```ntl
val config = fs.readJSON("config.json")
io.log(config.name)
```

---

## Writing Files

### `fs.write(path, content)`
Write a string to a file (creates or overwrites).

```ntl
fs.write("output.txt", "Hello, World!\n")
```

### `fs.append(path, content)`
Append text to a file.

```ntl
fs.append("log.txt", "New entry\n")
```

### `fs.writeJSON(path, value)`
Serialize a value to JSON and write to a file.

```ntl
fs.writeJSON("config.json", { name: "app", version: "1.0" })
```

---

## File Information

### `fs.exists(path)`
Returns `true` if the path exists.

```ntl
if fs.exists("config.json") {
  val cfg = fs.readJSON("config.json")
}
```

### `fs.isFile(path)`
Returns `true` if the path is a regular file.

### `fs.isDir(path)`
Returns `true` if the path is a directory.

### `fs.stat(path)`
Returns an object with file metadata.

```ntl
val info = fs.stat("file.txt")
io.log(info.size)     // bytes
io.log(info.mtime)    // last modified timestamp
io.log(info.isFile)   // boolean
io.log(info.isDir)    // boolean
```

---

## Directory Operations

### `fs.list(dir)`
List entries in a directory (names only).

```ntl
val files = fs.list(".")
each f in files {
  io.log(f)
}
```

### `fs.listFull(dir)`
List entries with full metadata objects.

```ntl
val entries = fs.listFull(".")
each e in entries {
  io.log(e.name, e.size, e.isDir)
}
```

### `fs.mkdir(path)`
Create a directory (and intermediate directories).

```ntl
fs.mkdir("build/output")
```

### `fs.rmdir(path)`
Remove an empty directory.

```ntl
fs.rmdir("tmp")
```

---

## File Management

### `fs.copy(src, dest)`
Copy a file.

```ntl
fs.copy("template.txt", "output.txt")
```

### `fs.move(src, dest)`
Move (rename) a file.

```ntl
fs.move("draft.txt", "final.txt")
```

### `fs.delete(path)`
Delete a file.

```ntl
fs.delete("temp.txt")
```

---

## Path Utilities

### `fs.join(...parts)`
Join path segments.

```ntl
val p = fs.join("home", "user", "file.txt")
// → "home/user/file.txt"
```

### `fs.dirname(path)`
Get the directory component of a path.

```ntl
val dir = fs.dirname("/home/user/file.txt")  // "/home/user"
```

### `fs.basename(path)`
Get the filename component.

```ntl
val name = fs.basename("/home/user/file.txt")  // "file.txt"
```

### `fs.ext(path)`
Get the file extension (with dot).

```ntl
val e = fs.ext("file.txt")  // ".txt"
```

### `fs.abs(path)`
Get the absolute path.

```ntl
val abs = fs.abs("./config.json")
```

---

## Example: Config File Manager

```ntl
val fs = @import("std.fs")
val io = @import("std.io")

fn loadConfig(path) {
  if !fs.exists(path) {
    val defaults = { theme: "dark", lang: "en" }
    fs.writeJSON(path, defaults)
    defaults
  }
  fs.readJSON(path)
}

fn main() {
  val cfg = loadConfig("config.json")
  io.log("Theme:", cfg.theme)
  io.log("Language:", cfg.lang)
}
```
