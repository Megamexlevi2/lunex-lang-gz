// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

const std = @import("std");

pub const E_NULL_DEREF: u16 = 4001;
pub const E_DIV_ZERO: u16 = 4002;
pub const E_INDEX_OOB: u16 = 4003;
pub const E_KEY_NOT_FOUND: u16 = 4004;
pub const E_STACK_OVERFLOW: u16 = 4005;
pub const E_INVALID_CAST: u16 = 4006;
pub const E_BAD_BYTECODE: u16 = 4007;
pub const E_FILE_NOT_FOUND: u16 = 4009;
pub const E_PERMISSION: u16 = 4010;
pub const E_NETWORK: u16 = 4011;
pub const E_TIMEOUT: u16 = 4012;
pub const E_ASSERTION: u16 = 4013;
pub const E_USER_PANIC: u16 = 4014;
pub const E_INVALID_REGEX: u16 = 4015;
pub const E_JIT_ALLOC: u16 = 5001;
pub const E_JIT_UNSUPPORTED: u16 = 5002;
pub const E_JIT_CODEGEN: u16 = 5003;
pub const E_IO_READ: u16 = 7001;
pub const E_IO_WRITE: u16 = 7002;
pub const E_BAD_BC_FORMAT: u16 = 7003;
pub const E_UNKNOWN: u16 = 9999;

const Entry = struct {
    title: []const u8,
    hint: []const u8,
};

const catalog = std.StaticStringMap(Entry).initComptime(.{
    .{ "NullDeref", Entry{
        .title = "Null dereference",
        .hint  = "You tried to access something on a null value. Add a null check first.",
    } },
    .{ "DivisionByZero", Entry{
        .title = "Division by zero",
        .hint  = "The divisor is zero. Guard with 'if b != 0' before dividing.",
    } },
    .{ "IndexOutOfBounds", Entry{
        .title = "Array index out of bounds",
        .hint  = "Index is >= length. Check bounds before indexing.",
    } },
    .{ "KeyNotFound", Entry{
        .title = "Key not found in object",
        .hint  = "That key doesn't exist. Use keys(obj) to see what's available.",
    } },
    .{ "StackOverflow", Entry{
        .title = "Stack overflow",
        .hint  = "Looks like infinite recursion. Add a base case to stop it.",
    } },
    .{ "StackUnderflow", Entry{
        .title = "Stack underflow",
        .hint  = "Internal error — too many pops. Recompile with 'lunex build'.",
    } },
    .{ "InvalidCast", Entry{
        .title = "Invalid type cast",
        .hint  = "Can't convert that value to the target type. Use typeof() to check first.",
    } },
    .{ "TypeError", Entry{
        .title = "Type error",
        .hint  = "Wrong type for this operation. Check what the value actually holds.",
    } },
    .{ "BadBytecode", Entry{
        .title = "Corrupted bytecode",
        .hint  = "The .nc file looks broken. Recompile with 'lunex build'.",
    } },
    .{ "InvalidBytecodeFormat", Entry{
        .title = "Invalid bytecode format",
        .hint  = "Not a valid .nc file. Use 'lunex build' to produce one.",
    } },
    .{ "UndefinedVariable", Entry{
        .title = "Undefined variable",
        .hint  = "That name doesn't exist in this scope. Check for typos.",
    } },
    .{ "FileNotFound", Entry{
        .title = "File not found",
        .hint  = "Double-check the path and make sure the file exists.",
    } },
    .{ "AccessDenied", Entry{
        .title = "Permission denied",
        .hint  = "The process doesn't have permission to access this resource.",
    } },
    .{ "OutOfMemory", Entry{
        .title = "Out of memory",
        .hint  = "The runtime ran out of heap memory. Try reducing data size.",
    } },
    .{ "Timeout", Entry{
        .title = "Operation timed out",
        .hint  = "Took too long. Check for slow I/O or runaway loops.",
    } },
    .{ "AssertionFailed", Entry{
        .title = "Assertion failed",
        .hint  = "An assert expression was false. Review your invariants.",
    } },
    .{ "Throw", Entry{
        .title = "Unhandled throw",
        .hint  = "A throw wasn't caught. Wrap the call in a try/catch.",
    } },
    .{ "UserPanic", Entry{
        .title = "Explicit panic",
        .hint  = "The program called panic() intentionally. Check the message for details.",
    } },
    .{ "InvalidRegex", Entry{
        .title = "Invalid regular expression",
        .hint  = "The pattern is malformed. Check your special character escaping.",
    } },
    .{ "JITAllocFailed", Entry{
        .title = "JIT: executable memory allocation failed",
        .hint  = "System denied executable memory. Falling back to interpreter.",
    } },
    .{ "UnsupportedArch", Entry{
        .title = "JIT: CPU architecture not supported",
        .hint  = "JIT works on x86_64 and AArch64. Falling back to interpreter.",
    } },
    .{ "IOError", Entry{
        .title = "I/O error",
        .hint  = "Read or write failed. Check paths, permissions, and disk space.",
    } },
    .{ "InvalidOpcode", Entry{
        .title = "Invalid opcode in bytecode",
        .hint  = "Unknown instruction found. Recompile with 'lunex build'.",
    } },
    .{ "MaxFramesExceeded", Entry{
        .title = "Call stack depth limit exceeded",
        .hint  = "Too many nested calls. Max is 512 frames.",
    } },
});

pub fn errorCode(err: anyerror) u16 {
    return switch (err) {
        error.NullDeref                        => E_NULL_DEREF,
        error.DivisionByZero                   => E_DIV_ZERO,
        error.IndexOutOfBounds                 => E_INDEX_OOB,
        error.KeyNotFound                      => E_KEY_NOT_FOUND,
        error.StackOverflow, error.StackUnderflow, error.MaxFramesExceeded => E_STACK_OVERFLOW,
        error.InvalidCast, error.TypeError     => E_INVALID_CAST,
        error.BadBytecode, error.InvalidBytecodeFormat, error.InvalidOpcode => E_BAD_BC_FORMAT,
        error.FileNotFound                     => E_FILE_NOT_FOUND,
        error.AccessDenied                     => E_PERMISSION,
        error.OutOfMemory                      => 4099,
        error.AssertionFailed                  => E_ASSERTION,
        error.JITAllocFailed, error.MMapFailed => E_JIT_ALLOC,
        error.UnsupportedArch                  => E_JIT_UNSUPPORTED,
        else                                   => E_UNKNOWN,
    };
}

pub fn errorHint(err: anyerror) []const u8 {
    if (catalog.get(@errorName(err))) |entry| return entry.hint;
    return "No extra info. Run with NTL_DEBUG=1 for details.";
}

pub fn formatError(alloc: std.mem.Allocator, err: anyerror) ![]const u8 {
    const name = @errorName(err);
    if (catalog.get(name)) |entry| {
        return std.fmt.allocPrint(alloc, "{s}", .{entry.title});
    }
    return std.fmt.allocPrint(alloc, "Runtime error: {s}", .{name});
}

pub fn formatFull(
    alloc: std.mem.Allocator,
    err: anyerror,
    line: u32,
    col: u16,
    extra: ?[]const u8,
) ![]const u8 {
    const name = @errorName(err);
    const code = errorCode(err);

    var title: []const u8 = name;
    var hint: []const u8 = "";
    if (catalog.get(name)) |entry| {
        title = entry.title;
        hint = entry.hint;
    }

    var buf = std.ArrayList(u8).init(alloc);
    const w = buf.writer();

    try w.print("[E{d:0>4}] {s}\n", .{ code, title });

    if (line > 0) {
        if (col > 0) {
            try w.print("  --> line {d}, column {d}\n", .{ line, col });
        } else {
            try w.print("  --> line {d}\n", .{line});
        }
    }

    if (extra) |e| try w.print("     = {s}\n", .{e});
    if (hint.len > 0) try w.print("  hint: {s}\n", .{hint});

    return buf.toOwnedSlice();
}
