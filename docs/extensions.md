# NTL lang — Writing Extensions

Extensions let you add new built-in functions to NTL in Zig, without changing any existing file.
The build system scans `zig/src/extensions/`, auto-generates `zig/src/extensions_gen.zig`, and your
functions are available in every NTL program after a rebuild.

## Step 1 — create `zig/src/extensions/myext.zig`

```zig
const std   = @import("std");
const value = @import("../value.zig");
const Value = value.Value;

pub fn register(globals: *std.StringHashMap(Value)) !void {
    try globals.put("square", Value{ .native = square });
    try globals.put("cube",   Value{ .native = cube   });
}

fn square(vm: anytype, args: []const Value) anyerror!Value {
    _ = vm;
    if (args.len == 0) return Value.Zero;
    const n = args[0].toInt();
    return Value{ .int = n * n };
}

fn cube(vm: anytype, args: []const Value) anyerror!Value {
    _ = vm;
    if (args.len == 0) return Value.Zero;
    const n = args[0].toInt();
    return Value{ .int = n * n * n };
}
```

## Step 2 — rebuild

```bash
./build.sh
```

## Value API

```zig
// Constructors
Value.mkInt(42)        // integer  (i64)
Value.mkFloat(3.14)    // float    (f64)
Value.mkBool(true)     // boolean
Value.mkStr("hello")   // string
Value.Null             // null
Value.True / False     // boolean constants
Value.Zero / One       // integer constants

// Inspect
v.isTruthy()           // bool
v.toInt()              // i64
v.toFloat()            // f64
v.typeName()           // "null"/"boolean"/"number"/"string"/"array"/"object"/"function"
v.equals(other)        // structural equality
v.toString(alloc)      // ![]const u8

// Pattern match
switch (v) {
    .null_val => {},
    .boolean  => |b| {},
    .int      => |n| {},
    .float    => |f| {},
    .string   => |s| {},
    .array    => |a| {},   // std.ArrayList(Value)
    .object   => |o| {},   // std.StringHashMap(Value)
    .native   => |f| {},
    .closure  => |p| {},
}
```
