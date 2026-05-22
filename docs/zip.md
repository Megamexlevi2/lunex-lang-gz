# ZIP Module

ZIP archive creation and extraction utilities.

**Use case:** Create and extract ZIP files.

---

## Import

```ntl
val zip = @import("std.zip")
```

---

## Available Functions

### `create(output_path, files)`

Executes the `create` operation with the given parameters (output_path, files).

**Signature:**
```ntl
fn create(output_path, files)
```

### `extract(zip_path, dest_dir)`

Executes the `extract` operation with the given parameters (zip_path, dest_dir).

**Signature:**
```ntl
fn extract(zip_path, dest_dir)
```

### `list(zip_path)`

Executes the `list` operation with the given parameter (zip_path).

**Signature:**
```ntl
fn list(zip_path)
```

### `read(zip_path, entry_name)`

Executes the `read` operation with the given parameters (zip_path, entry_name).

**Signature:**
```ntl
fn read(zip_path, entry_name)
```

### `addFile(zip_path, file_path, entry_name)`

Executes the `addFile` operation with the given parameters (zip_path, file_path, entry_name).

**Signature:**
```ntl
fn addFile(zip_path, file_path, entry_name)
```

### `packDir(src_dir, output_path)`

Executes the `packDir` operation with the given parameters (src_dir, output_path).

**Signature:**
```ntl
fn packDir(src_dir, output_path)
```

### `fromBytes(raw)`

Executes the `fromBytes` operation with the given parameter (raw).

**Signature:**
```ntl
fn fromBytes(raw)
```

