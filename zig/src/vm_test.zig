// Lunex lang — VM + JIT tests.
// Covers:
//   - Value constructors and conversions
//   - VM raw NTZ opcode execution (arithmetic, branching, locals, stack)
//   - NC container format decoder
//   - JIT compilation + correctness on the host architecture
//   - Profiler hot-function detection and type-feedback
//
// Run with: zig build test

const std       = @import("std");
const value     = @import("value.zig");
const vm_mod    = @import("vm.zig");
const builtins  = @import("builtins.zig");
const jit_mod   = @import("jit.zig");
const prof_mod  = @import("profiler.zig");

const Value  = value.Value;
const VM     = vm_mod.VM;
const Opcode = vm_mod.Opcode;
const talloc = std.testing.allocator;

// ─── VM helpers ───────────────────────────────────────────────────────────────

fn runVM(code: []const u8) !Value {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();
    var machine = try VM.init(alloc);
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, alloc);
    return machine.execBytecode(code);
}

// Mirrors main.zig's extractNTZFromNC (to avoid importing main in tests).
fn extractNTZFromNC(data: []const u8) ![]const u8 {
    if (data.len >= 48 and
        data[0] == 'n' and data[1] == 't' and data[2] == 'l' and data[3] == 'i')
    {
        const ntz_len: u32 =
            @as(u32, data[40]) | (@as(u32, data[41]) << 8) |
            (@as(u32, data[42]) << 16) | (@as(u32, data[43]) << 24);
        if (ntz_len > 0 and ntz_len <= data.len - 48) return data[data.len - ntz_len..];
        return error.InvalidBytecodeFormat;
    }
    return data;
}

fn buildNCWithNTZ(alloc: std.mem.Allocator, ntz: []const u8) ![]u8 {
    var buf = std.ArrayList(u8).init(alloc);
    errdefer buf.deinit();
    try buf.appendSlice("ntli");
    try buf.appendSlice(&[_]u8{ 0x00, 0x05, 0x01, 0x00 }); // version, flags
    try buf.appendNTimes(0, 16); // hash
    try buf.appendNTimes(0, 4);  // src len
    try buf.appendNTimes(0, 4);  // chunk count
    try buf.appendNTimes(0, 4);  // name len
    try buf.appendSlice(&[_]u8{ 0xA7, 0x3E, 0xC1, 0x5B }); // sentinel
    // [40:44] NTZ section length
    const ntz_len: u32 = @intCast(ntz.len);
    try buf.append(@truncate(ntz_len));
    try buf.append(@truncate(ntz_len >> 8));
    try buf.append(@truncate(ntz_len >> 16));
    try buf.append(@truncate(ntz_len >> 24));
    try buf.appendNTimes(0, 4); // reserved
    try buf.appendSlice("DUMMY");
    try buf.appendSlice(ntz);
    return buf.toOwnedSlice();
}

fn runNC(data: []const u8) !Value {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();
    var machine = try VM.init(alloc);
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, alloc);
    const code = try extractNTZFromNC(data);
    return machine.execBytecode(code);
}

// ─── Value constructor tests ──────────────────────────────────────────────────

test "value: int" {
    const v = Value.mkInt(42);
    try std.testing.expect(v == .int);
    try std.testing.expectEqual(@as(i64, 42), v.int);
}

test "value: float" {
    const v = Value.mkFloat(3.14);
    try std.testing.expect(v == .float);
    try std.testing.expect(v.float == 3.14);
}

test "value: bool" {
    try std.testing.expect(Value.mkBool(true).boolean  == true);
    try std.testing.expect(Value.mkBool(false).boolean == false);
    try std.testing.expect(Value.True.boolean);
    try std.testing.expect(!Value.False.boolean);
}

test "value: null" {
    try std.testing.expect(Value.Null == .null_val);
    try std.testing.expect(!Value.Null.isTruthy());
}

test "value: isTruthy" {
    try std.testing.expect(Value.mkInt(1).isTruthy());
    try std.testing.expect(!Value.mkInt(0).isTruthy());
    try std.testing.expect(Value.mkFloat(0.1).isTruthy());
    try std.testing.expect(!Value.mkFloat(0.0).isTruthy());
    try std.testing.expect(Value.True.isTruthy());
    try std.testing.expect(!Value.False.isTruthy());
    try std.testing.expect(Value.mkStr("x").isTruthy());
    try std.testing.expect(!Value.mkStr("").isTruthy());
}

test "value: equals" {
    try std.testing.expect(Value.mkInt(3).equals(Value.mkFloat(3.0)));
    try std.testing.expect(!Value.mkInt(3).equals(Value.mkInt(4)));
}

test "value: toString int" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const s = try Value.mkInt(99).toString(gpa.allocator());
    defer gpa.allocator().free(s);
    try std.testing.expectEqualStrings("99", s);
}

test "value: toInt / toFloat conversions" {
    try std.testing.expectEqual(@as(i64, 7),    Value.mkInt(7).toInt());
    try std.testing.expectEqual(@as(i64, 3),    Value.mkFloat(3.7).toInt());
    try std.testing.expect(Value.mkInt(5).toFloat() == 5.0);
}

test "value: typeName" {
    try std.testing.expectEqualStrings("null",    Value.Null.typeName());
    try std.testing.expectEqualStrings("boolean", Value.True.typeName());
    try std.testing.expectEqualStrings("number",  Value.mkInt(1).typeName());
    try std.testing.expectEqualStrings("string",  Value.mkStr("").typeName());
}

// ─── VM opcode tests ──────────────────────────────────────────────────────────

test "vm: halt returns null" {
    const code = [_]u8{ @intFromEnum(Opcode.Halt) };
    const r = try runVM(&code);
    try std.testing.expect(r == .null_val);
}

test "vm: add two ints → 42" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 10, .little); try code.appendSlice(&b8);

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 32, .little); try code.appendSlice(&b8);

    try code.append(@intFromEnum(Opcode.Add));
    try code.append(@intFromEnum(Opcode.Return));

    const r = try runVM(code.items);
    try std.testing.expectEqual(@as(i64, 42), r.int);
}

test "vm: 3*4 - 2 = 10" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 3, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 4, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Mul));

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 2, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Sub));
    try code.append(@intFromEnum(Opcode.Return));

    const r = try runVM(code.items);
    try std.testing.expectEqual(@as(i64, 10), r.int);
}

test "vm: 10 mod 3 = 1" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 10, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 3, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Mod));
    try code.append(@intFromEnum(Opcode.Return));

    const r = try runVM(code.items);
    try std.testing.expectEqual(@as(i64, 1), r.int);
}

test "vm: boolean not" {
    const code = [_]u8{
        @intFromEnum(Opcode.Const), 1, 1,
        @intFromEnum(Opcode.Not),
        @intFromEnum(Opcode.Return),
    };
    const r = try runVM(&code);
    try std.testing.expect(r == .boolean and !r.boolean);
}

test "vm: comparison 5 < 10 = true" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 5, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 10, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Lt));
    try code.append(@intFromEnum(Opcode.Return));

    const r = try runVM(code.items);
    try std.testing.expect(r == .boolean and r.boolean);
}

test "vm: unconditional jump" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b4: [4]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Jump));
    std.mem.writeInt(u32, &b4, @intCast(1 + 4), .little);
    try code.appendSlice(&b4);
    try code.append(@intFromEnum(Opcode.Halt));

    const r = try runVM(code.items);
    try std.testing.expect(r == .null_val);
}

test "vm: store and load local" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;
    var b2: [2]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 77, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Store));
    std.mem.writeInt(u16, &b2, 0, .little); try code.appendSlice(&b2);
    try code.append(@intFromEnum(Opcode.Load));
    std.mem.writeInt(u16, &b2, 0, .little); try code.appendSlice(&b2);
    try code.append(@intFromEnum(Opcode.Return));

    const r = try runVM(code.items);
    try std.testing.expectEqual(@as(i64, 77), r.int);
}

test "vm: pop discards top of stack" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 42, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Pop));
    try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
    std.mem.writeInt(i64, &b8, 99, .little); try code.appendSlice(&b8);
    try code.append(@intFromEnum(Opcode.Return));

    const r = try runVM(code.items);
    try std.testing.expectEqual(@as(i64, 99), r.int);
}

test "vm: pop underflow returns StackUnderflow" {
    const code = [_]u8{ @intFromEnum(Opcode.Pop) };
    try std.testing.expectError(error.StackUnderflow, runVM(&code));
}

test "vm: string concat Hello + Lunex" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b4: [4]u8 = undefined;

    try code.append(@intFromEnum(Opcode.Const)); try code.append(4);
    std.mem.writeInt(u32, &b4, 5, .little); try code.appendSlice(&b4);
    try code.appendSlice("Hello");
    try code.append(@intFromEnum(Opcode.Const)); try code.append(4);
    std.mem.writeInt(u32, &b4, 3, .little); try code.appendSlice(&b4);
    try code.appendSlice("Lunex");
    try code.append(@intFromEnum(Opcode.Add));
    try code.append(@intFromEnum(Opcode.Return));

    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    var machine = try VM.init(gpa.allocator());
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, gpa.allocator());
    const r = try machine.execBytecode(code.items);
    try std.testing.expectEqualStrings("HelloNTL", r.string);
}

test "vm: make array [1, 2, 3]" {
    var code = std.ArrayList(u8).init(talloc);
    defer code.deinit();
    var b8: [8]u8 = undefined;
    var b2: [2]u8 = undefined;

    for ([_]i64{ 1, 2, 3 }) |n| {
        try code.append(@intFromEnum(Opcode.Const)); try code.append(2);
        std.mem.writeInt(i64, &b8, n, .little); try code.appendSlice(&b8);
    }
    try code.append(@intFromEnum(Opcode.MakeArray));
    std.mem.writeInt(u16, &b2, 3, .little); try code.appendSlice(&b2);
    try code.append(@intFromEnum(Opcode.Return));

    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    var machine = try VM.init(gpa.allocator());
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, gpa.allocator());
    const r = try machine.execBytecode(code.items);
    try std.testing.expect(r == .array);
    try std.testing.expectEqual(@as(usize, 3), r.array.items.len);
    try std.testing.expectEqual(@as(i64, 1), r.array.items[0].int);
    try std.testing.expectEqual(@as(i64, 3), r.array.items[2].int);
}

// ─── NC container format tests ────────────────────────────────────────────────

test "nc: raw bytecode passes through" {
    const code = [_]u8{ @intFromEnum(Opcode.Halt) };
    const out = try extractNTZFromNC(&code);
    try std.testing.expectEqualSlices(u8, &code, out);
}

test "nc: missing NTZ section → InvalidBytecodeFormat" {
    var hdr = [_]u8{0} ** 60;
    hdr[0] = 'n'; hdr[1] = 't'; hdr[2] = 'l'; hdr[3] = 'i';
    hdr[4] = 0; hdr[5] = 5; hdr[6] = 1;
    // hdr[40:44] = 0 → ntz_len = 0 → InvalidBytecodeFormat
    try std.testing.expectError(error.InvalidBytecodeFormat, extractNTZFromNC(&hdr));
}

test "nc: too short → pass through" {
    const tiny = [_]u8{ 'n', 't', 'l', 'i', 0, 0 };
    const out = try extractNTZFromNC(&tiny);
    try std.testing.expectEqualSlices(u8, &tiny, out);
}

test "nc: NC with Halt NTZ" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const ntz = [_]u8{ @intFromEnum(Opcode.Halt) };
    const nc = try buildNCWithNTZ(gpa.allocator(), &ntz);
    defer gpa.allocator().free(nc);
    const r = try runNC(nc);
    try std.testing.expect(r == .null_val);
}

test "nc: NC with Return 55" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();
    var ntz = std.ArrayList(u8).init(alloc);
    defer ntz.deinit();
    var b8: [8]u8 = undefined;
    try ntz.append(@intFromEnum(Opcode.Const)); try ntz.append(2);
    std.mem.writeInt(i64, &b8, 55, .little); try ntz.appendSlice(&b8);
    try ntz.append(@intFromEnum(Opcode.Return));
    const nc = try buildNCWithNTZ(alloc, ntz.items);
    defer alloc.free(nc);
    const r = try runNC(nc);
    try std.testing.expectEqual(@as(i64, 55), r.int);
}

// ─── JIT compiler tests ───────────────────────────────────────────────────────
// These tests exercise the JIT compiler's code-buffer and emitter directly.
// On unsupported architectures the tests gracefully skip.

/// Build a tiny NTZ-style JIT bytecode:
///   LoadConst 5 → StoreLocal 0
///   LoadConst 3 → StoreLocal 1
///   Add locals[0]+locals[1] → StoreLocal 2
///   (result = 8 in locals[2])
fn buildAddBytecode() [32]u8 {
    var bc: [32]u8 = undefined;
    var i: usize = 0;
    // Op tags match jit_mod.Op
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 5, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 0; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 3, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Add); i += 1; bc[i] = 0; i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Ret); i += 1;
    _ = i;
    return bc;
}

test "jit: compile and call on host arch" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    const bc_full = buildAddBytecode();
    // Slice only the filled-in bytes (up to Ret opcode):
    // LoadConst(9) + StoreLocal(2) + LoadConst(9) + StoreLocal(2) + Add(3) + Ret(1) = 27
    const bc = bc_full[0..27];

    var jc = jit_mod.JitCompiler.init(alloc);
    var cf = jc.compile(bc) catch |err| {
        if (err == error.UnsupportedArch) {
            std.debug.print("  [skip] JIT not supported on this arch\n", .{});
            return;
        }
        return err;
    };
    defer cf.deinit();

    var locals = [_]i64{0} ** 8;
    const result = cf.call(&locals, &locals);
    // Add: locals[0]=5, locals[1]=3 → acc=8
    try std.testing.expectEqual(@as(i64, 8), result);
}

test "jit: CodeBuffer alloc and seal" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    var cb = jit_mod.CodeBuffer.init(gpa.allocator()) catch |err| {
        if (err == error.OutOfMemory or err == error.PermissionDenied) return;
        return err;
    };
    defer cb.deinit();
    // Emit a NOP (x86: 0x90 / AArch64: NOP = 0xD503201F / RV64: NOP = ADDI x0,x0,0)
    try cb.emit(0x90);
    try cb.seal();
    try std.testing.expect(cb.currentOffset() == 1);
}

test "jit: LoadConst negative value" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    var bc: [10]u8 = undefined;
    bc[0] = @intFromEnum(jit_mod.Op.LoadConst);
    std.mem.writeInt(i64, bc[1..][0..8], -42, .little);
    bc[9] = @intFromEnum(jit_mod.Op.Ret);

    var jc = jit_mod.JitCompiler.init(alloc);
    var cf = jc.compile(&bc) catch |err| {
        if (err == error.UnsupportedArch) return;
        return err;
    };
    defer cf.deinit();

    var locals = [_]i64{0} ** 4;
    const result = cf.call(&locals, &locals);
    try std.testing.expectEqual(@as(i64, -42), result);
}

test "jit: large 64-bit immediate" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    const large_val: i64 = 0x1234_5678_9ABC_DEF0;
    var bc: [10]u8 = undefined;
    bc[0] = @intFromEnum(jit_mod.Op.LoadConst);
    std.mem.writeInt(i64, bc[1..][0..8], large_val, .little);
    bc[9] = @intFromEnum(jit_mod.Op.Ret);

    var jc = jit_mod.JitCompiler.init(alloc);
    var cf = jc.compile(&bc) catch |err| {
        if (err == error.UnsupportedArch) return;
        return err;
    };
    defer cf.deinit();

    var locals = [_]i64{0} ** 4;
    const result = cf.call(&locals, &locals);
    try std.testing.expectEqual(large_val, result);
}

test "jit: sub operation locals[0]=10 - locals[1]=3 = 7" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    var bc: [32]u8 = undefined;
    var i: usize = 0;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 10, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 0; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 3, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Sub); i += 1; bc[i] = 0; i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Ret); i += 1;

    var jc = jit_mod.JitCompiler.init(alloc);
    var cf = jc.compile(bc[0..i]) catch |err| {
        if (err == error.UnsupportedArch) return;
        return err;
    };
    defer cf.deinit();

    var locals = [_]i64{0} ** 8;
    const result = cf.call(&locals, &locals);
    try std.testing.expectEqual(@as(i64, 7), result);
}

test "jit: mul 6 * 7 = 42" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    var bc: [32]u8 = undefined;
    var i: usize = 0;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 6, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 0; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 7, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Mul); i += 1; bc[i] = 0; i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Ret); i += 1;

    var jc = jit_mod.JitCompiler.init(alloc);
    var cf = jc.compile(bc[0..i]) catch |err| {
        if (err == error.UnsupportedArch) return;
        return err;
    };
    defer cf.deinit();

    var locals = [_]i64{0} ** 8;
    const result = cf.call(&locals, &locals);
    try std.testing.expectEqual(@as(i64, 42), result);
}

test "jit: eq comparison 5 == 5 → 1" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    var bc: [32]u8 = undefined;
    var i: usize = 0;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 5, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 0; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.LoadConst); i += 1;
    std.mem.writeInt(i64, bc[i..][0..8], 5, .little); i += 8;
    bc[i] = @intFromEnum(jit_mod.Op.StoreLocal); i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Eq); i += 1; bc[i] = 0; i += 1; bc[i] = 1; i += 1;
    bc[i] = @intFromEnum(jit_mod.Op.Ret); i += 1;

    var jc = jit_mod.JitCompiler.init(alloc);
    var cf = jc.compile(bc[0..i]) catch |err| {
        if (err == error.UnsupportedArch) return;
        return err;
    };
    defer cf.deinit();

    var locals = [_]i64{0} ** 8;
    const result = cf.call(&locals, &locals);
    try std.testing.expectEqual(@as(i64, 1), result);
}

test "jit: JitCache put and get" {
    var gpa = std.heap.DebugAllocator(.{}){};
    defer _ = gpa.deinit();
    const alloc = gpa.allocator();

    var bc: [10]u8 = undefined;
    bc[0] = @intFromEnum(jit_mod.Op.LoadConst);
    std.mem.writeInt(i64, bc[1..][0..8], 99, .little);
    bc[9] = @intFromEnum(jit_mod.Op.Ret);

    var jc = jit_mod.JitCompiler.init(alloc);
    const cf = jc.compile(&bc) catch |err| {
        if (err == error.UnsupportedArch) return;
        return err;
    };

    var cache = jit_mod.JitCache.init(alloc);
    defer cache.deinit();

    try cache.put(0xDEAD_BEEF, cf);
    const found = cache.get(0xDEAD_BEEF);
    try std.testing.expect(found != null);

    var locals = [_]i64{0} ** 4;
    const result = found.?.call(&locals, &locals);
    try std.testing.expectEqual(@as(i64, 99), result);
}

// ─── Profiler tests ───────────────────────────────────────────────────────────

test "profiler: hot function detection" {
    var prof = prof_mod.Profiler.init(talloc);
    defer prof.deinit();

    var i: u32 = 0;
    while (i < prof_mod.HOT_FUNCTION_THRESHOLD - 1) : (i += 1) {
        const hot = try prof.recordCall(42);
        try std.testing.expect(!hot);
    }
    const hot = try prof.recordCall(42);
    try std.testing.expect(hot);
}

test "profiler: different function IDs are independent" {
    var prof = prof_mod.Profiler.init(talloc);
    defer prof.deinit();

    var i: u32 = 0;
    while (i < prof_mod.HOT_FUNCTION_THRESHOLD) : (i += 1) _ = try prof.recordCall(1);
    // Function 2 should not be hot yet (it has 0 calls).
    const hot2 = try prof.recordCall(2);
    try std.testing.expect(!hot2);
}

test "profiler: type feedback monomorphic" {
    var prof = prof_mod.Profiler.init(talloc);
    defer prof.deinit();

    try prof.observeType(10, 2); // type 2 = int
    try prof.observeType(10, 2);
    try prof.observeType(10, 2);

    const fb = prof.type_feedback.get(10) orelse return error.MissingFeedback;
    try std.testing.expect(fb.isMonomorphic());
    try std.testing.expect(fb.seen_int);
    try std.testing.expect(!fb.seen_float);
}

test "profiler: type feedback polymorphic" {
    var prof = prof_mod.Profiler.init(talloc);
    defer prof.deinit();

    try prof.observeType(20, 2); // int
    try prof.observeType(20, 3); // float
    const fb = prof.type_feedback.get(20) orelse return error.MissingFeedback;
    try std.testing.expect(!fb.isMonomorphic());
}
