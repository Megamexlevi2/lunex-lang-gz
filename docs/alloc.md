# ntl:alloc

Low-level memory buffer module. Gives direct control over raw byte regions, shared memory pools, and binary I/O. Bypasses NTL safety guarantees — reading or writing out of bounds will corrupt memory or crash the process.

## Import

```ntl
val alloc = @import("std.alloc")
```

## Buffer

### `alloc.create(size)`

Allocates a new buffer of `size` bytes (zero-initialized).

```ntl
val buf = alloc.create(1024)
```

### `alloc.fromString(str)`

Creates a buffer from a UTF-8 string.

```ntl
val buf = alloc.fromString("hello")
```

### `alloc.fromBytes(array)`

Creates a buffer from an array of byte values (0–255).

```ntl
val buf = alloc.fromBytes([0x48, 0x65, 0x6C, 0x6C, 0x6F])
```

## Buffer methods

All methods are on the object returned by `alloc.create`, `alloc.fromString`, or `alloc.fromBytes`.

| Method | Description |
|---|---|
| `buf.size()` | Returns the byte length of the buffer |
| `buf.cap()` | Returns the allocated capacity |
| `buf.readByte(index)` | Reads a single byte at `index` (0–255) |
| `buf.writeByte(index, value)` | Writes a byte at `index`; returns `true` on success |
| `buf.readU16LE(offset)` | Reads a 16-bit unsigned integer (little-endian) |
| `buf.readU32LE(offset)` | Reads a 32-bit unsigned integer (little-endian) |
| `buf.readU64LE(offset)` | Reads a 64-bit unsigned integer (little-endian) |
| `buf.writeU16LE(offset, value)` | Writes a 16-bit unsigned integer (little-endian) |
| `buf.writeU32LE(offset, value)` | Writes a 32-bit unsigned integer (little-endian) |
| `buf.writeU64LE(offset, value)` | Writes a 64-bit unsigned integer (little-endian) |
| `buf.readFloat32LE(offset)` | Reads a 32-bit float (little-endian) |
| `buf.readFloat64LE(offset)` | Reads a 64-bit float (little-endian) |
| `buf.writeFloat32LE(offset, value)` | Writes a 32-bit float (little-endian) |
| `buf.writeFloat64LE(offset, value)` | Writes a 64-bit float (little-endian) |
| `buf.toString()` | Converts the buffer to a UTF-8 string |
| `buf.toBytes()` | Returns an array of byte values |
| `buf.slice(start, end)` | Returns a new buffer from `start` to `end` (exclusive) |
| `buf.copy(src, dstOffset, srcOffset, length)` | Copies `length` bytes from `src` into this buffer |
| `buf.fill(value)` | Fills the entire buffer with `value` (0–255) |
| `buf.close()` | Releases the buffer and marks it as closed |

## Shared regions

Named memory regions shared across calls within the same process.

### `alloc.region(name, size)`

Creates or attaches to a named shared region.

```ntl
val region = alloc.region("mydata", 4096)
region.writeU32LE(0, 42)
```

### `alloc.dropRegion(name)`

Releases a named shared region.

```ntl
alloc.dropRegion("mydata")
```

## HTTP binary transfer

### `alloc.fetchBinary(url)`

Downloads a URL and returns the response body as a buffer.

```ntl
val buf = await alloc.fetchBinary("https://example.com/data.bin")
```

### `alloc.postBinary(url, buf)`

Posts a buffer as the request body (raw bytes). Returns the response as a buffer.

```ntl
val res = await alloc.postBinary("https://example.com/upload", buf)
```

## Notes

- Index bounds are checked at runtime — out-of-bounds reads return `false`, writes return `false`.
- `close()` must be called manually; buffers are not garbage collected automatically.
- Shared regions use reference counting; `dropRegion` decrements the ref count.
- On Android, this module runs entirely in user-space without any kernel privilege escalation.
