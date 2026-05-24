// Lunex lang — Built-in functions for the Zig VM
// Created by David Dev · GitHub: https://github.com/Megamexlevi2

const std   = @import("std");
const value = @import("value.zig");
const Value = value.Value;

const extensions = @import("extensions_gen.zig");

const pa = std.heap.page_allocator;

pub fn registerAll(globals: *std.StringHashMap(Value), alloc: std.mem.Allocator) !void {
    try extensions.registerAll(globals, alloc);
    try globals.put("log",        Value{ .native = builtinPrint });
    try globals.put("parseInt",   Value{ .native = builtinParseInt });
    try globals.put("parseFloat", Value{ .native = builtinParseFloat });
    try globals.put("String",     Value{ .native = builtinString });
    try globals.put("Number",     Value{ .native = builtinNumber });
    try globals.put("Boolean",    Value{ .native = builtinBoolean });
    try globals.put("isNaN",      Value{ .native = builtinIsNaN });
    try globals.put("isFinite",   Value{ .native = builtinIsFinite });
    try globals.put("typeof",     Value{ .native = builtinTypeof });
    try globals.put("len",        Value{ .native = builtinLen });
    try globals.put("keys",       Value{ .native = builtinKeys });
    try globals.put("values",     Value{ .native = builtinValues });
    try globals.put("entries",    Value{ .native = builtinEntries });
    try globals.put("range",      Value{ .native = builtinRange });
    try globals.put("sleep",      Value{ .native = builtinSleep });
    try globals.put("abs",        Value{ .native = builtinAbs });
    try globals.put("floor",      Value{ .native = builtinFloor });
    try globals.put("ceil",       Value{ .native = builtinCeil });
    try globals.put("sqrt",       Value{ .native = builtinSqrt });
    try globals.put("min",        Value{ .native = builtinMin });
    try globals.put("max",        Value{ .native = builtinMax });
    try globals.put("push",       Value{ .native = builtinPush });
    try globals.put("pop",        Value{ .native = builtinPop });
    try globals.put("slice",      Value{ .native = builtinSlice });
    try globals.put("join",       Value{ .native = builtinJoin });
    try globals.put("split",      Value{ .native = builtinSplit });
    try globals.put("trim",       Value{ .native = builtinTrim });
    try globals.put("toUpperCase",Value{ .native = builtinToUpperCase });
    try globals.put("toLowerCase",Value{ .native = builtinToLowerCase });
    try globals.put("includes",   Value{ .native = builtinIncludes });
    try globals.put("indexOf",    Value{ .native = builtinIndexOf });
    try globals.put("replace",    Value{ .native = builtinReplace });
    try globals.put("startsWith", Value{ .native = builtinStartsWith });
    try globals.put("endsWith",   Value{ .native = builtinEndsWith });
    try globals.put("concat",     Value{ .native = builtinConcat });
    try globals.put("reverse",    Value{ .native = builtinReverse });
    try globals.put("sort",       Value{ .native = builtinSort });
}

fn builtinPrint(args: []const Value) anyerror!Value {
    var buf = std.ArrayList(u8).empty;
    defer buf.deinit(pa);
    for (args, 0..) |arg, i| {
        if (i > 0) try buf.append(pa, ' ');
        const s = try arg.toString(pa);
        try buf.appendSlice(pa, s);
    }
    try buf.append(pa, '\n');
    // Platform-agnostic stdout write via direct syscall.
    var done: usize = 0;
    while (done < buf.items.len) {
        const rc = std.os.linux.write(1, buf.items.ptr + done, buf.items.len - done);
        if (std.os.linux.errno(rc) != .SUCCESS) break;
        if (rc == 0) break;
        done += rc;
    }
    return Value.Null;
}

fn builtinParseInt(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value{ .float = std.math.nan(f64) };
    switch (args[0]) {
        .int    => |n| return Value{ .int = n },
        .float  => |f| return Value{ .int = @intFromFloat(f) },
        .string => |s| {
            const t = std.mem.trim(u8, s, &std.ascii.whitespace);
            const n = std.fmt.parseInt(i64, t, 10) catch return Value{ .float = std.math.nan(f64) };
            return Value{ .int = n };
        },
        else => return Value{ .float = std.math.nan(f64) },
    }
}

fn builtinParseFloat(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value{ .float = std.math.nan(f64) };
    switch (args[0]) {
        .float  => return args[0],
        .int    => |n| return Value{ .float = @as(f64, @floatFromInt(n)) },
        .string => |s| {
            const t = std.mem.trim(u8, s, &std.ascii.whitespace);
            const f = std.fmt.parseFloat(f64, t) catch return Value{ .float = std.math.nan(f64) };
            return Value{ .float = f };
        },
        else => return Value{ .float = std.math.nan(f64) },
    }
}

fn builtinString(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value{ .string = "" };
    return Value{ .string = try args[0].toString(pa) };
}

fn builtinNumber(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    return switch (args[0]) {
        .int   => args[0],
        .float => args[0],
        else   => Value{ .float = args[0].toFloat() },
    };
}

fn builtinBoolean(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.False;
    return Value{ .boolean = args[0].isTruthy() };
}

fn builtinIsNaN(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.True;
    return switch (args[0]) {
        .float => |f| Value{ .boolean = std.math.isNan(f) },
        .int   => Value.False,
        else   => Value.True,
    };
}

fn builtinIsFinite(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.False;
    return switch (args[0]) {
        .float => |f| Value{ .boolean = std.math.isFinite(f) },
        .int   => Value.True,
        else   => Value.False,
    };
}

fn builtinTypeof(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value{ .string = "undefined" };
    return Value{ .string = args[0].typeName() };
}

fn builtinLen(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    return switch (args[0]) {
        .string => |s| Value{ .int = @intCast(s.len) },
        .array  => |a| Value{ .int = @intCast(a.items.len) },
        .object => |o| Value{ .int = @intCast(o.count()) },
        else    => Value.Zero,
    };
}

fn builtinKeys(args: []const Value) anyerror!Value {
    var arr = std.ArrayList(Value).empty;
    if (args.len == 0) return Value{ .array = arr };
    switch (args[0]) {
        .object => |obj| {
            var it = obj.keyIterator();
            while (it.next()) |k| try arr.append(pa, Value{ .string = k.* });
        },
        .array  => |a| {
            for (0..a.items.len) |i|
                try arr.append(pa, Value{ .int = @intCast(i) });
        },
        else => {},
    }
    return Value{ .array = arr };
}

fn builtinValues(args: []const Value) anyerror!Value {
    var arr = std.ArrayList(Value).empty;
    if (args.len == 0) return Value{ .array = arr };
    switch (args[0]) {
        .object => |obj| { var it = obj.valueIterator(); while (it.next()) |v| try arr.append(pa, v.*); },
        .array  => |a|   { for (a.items) |v| try arr.append(pa, v); },
        else    => {},
    }
    return Value{ .array = arr };
}

fn builtinEntries(args: []const Value) anyerror!Value {
    var arr = std.ArrayList(Value).empty;
    if (args.len == 0) return Value{ .array = arr };
    switch (args[0]) {
        .object => |obj| {
            var it = obj.iterator();
            while (it.next()) |entry| {
                var pair = std.ArrayList(Value).empty;
                try pair.append(pa, Value{ .string = entry.key_ptr.* });
                try pair.append(pa, entry.value_ptr.*);
                try arr.append(pa, Value{ .array = pair });
            }
        },
        .array  => |a| {
            for (a.items, 0..) |v, i| {
                var pair = std.ArrayList(Value).empty;
                try pair.append(pa, Value{ .int = @intCast(i) });
                try pair.append(pa, v);
                try arr.append(pa, Value{ .array = pair });
            }
        },
        else => {},
    }
    return Value{ .array = arr };
}

fn builtinRange(args: []const Value) anyerror!Value {
    var arr = std.ArrayList(Value).empty;
    if (args.len == 0) return Value{ .array = arr };

    const start: i64 = if (args.len >= 2) args[0].toInt() else 0;
    const end_v: i64 = if (args.len >= 2) args[1].toInt() else args[0].toInt();
    const step:  i64 = if (args.len >= 3) args[2].toInt() else 1;
    if (step == 0) return Value{ .array = arr };

    var i = start;
    if (step > 0) {
        while (i < end_v) : (i += step) {
            try arr.append(pa, Value{ .int = i });
            if (arr.items.len > 10_000_000) break;
        }
    } else {
        while (i > end_v) : (i += step) {
            try arr.append(pa, Value{ .int = i });
            if (arr.items.len > 10_000_000) break;
        }
    }
    return Value{ .array = arr };
}

fn builtinSleep(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Null;
    const ms: u64 = @intCast(@max(0, args[0].toInt()));
    // Use nanosleep syscall directly (cross-platform via posix).
    const ns = ms * std.time.ns_per_ms;
    var req = std.os.linux.timespec{
        .sec = @intCast(ns / std.time.ns_per_s),
        .nsec = @intCast(ns % std.time.ns_per_s),
    };
    _ = std.os.linux.nanosleep(&req, null);
    return Value.Null;
}

fn builtinAbs(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    return switch (args[0]) {
        .int   => |n| Value{ .int   = if (n < 0) -n else n },
        .float => |f| Value{ .float = @abs(f) },
        else   => Value.Zero,
    };
}

fn builtinFloor(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    return switch (args[0]) {
        .int   => args[0],
        .float => |f| Value{ .int = @intFromFloat(@floor(f)) },
        else   => Value.Zero,
    };
}

fn builtinCeil(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    return switch (args[0]) {
        .int   => args[0],
        .float => |f| Value{ .int = @intFromFloat(@ceil(f)) },
        else   => Value.Zero,
    };
}

fn builtinSqrt(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    const f = args[0].toFloat();
    return Value{ .float = @sqrt(f) };
}

fn builtinMin(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    var best = args[0];
    for (args[1..]) |a| {
        if (a.toFloat() < best.toFloat()) best = a;
    }
    return best;
}

fn builtinMax(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;
    var best = args[0];
    for (args[1..]) |a| {
        if (a.toFloat() > best.toFloat()) best = a;
    }
    return best;
}

fn builtinPush(args: []const Value) anyerror!Value {
    if (args.len < 2) return Value.Null;
    if (args[0] != .array) return Value.Null;
    for (args[1..]) |a| try @constCast(&args[0].array).append(pa, a);
    return Value{ .int = @intCast(args[0].array.items.len) };
}

fn builtinPop(args: []const Value) anyerror!Value {
    if (args.len == 0 or args[0] != .array) return Value.Null;
    const arr = @constCast(&args[0].array);
    if (arr.items.len == 0) return Value.Null;
    const last = arr.items[arr.items.len - 1];
    arr.items = arr.items[0 .. arr.items.len - 1];
    return last;
}

fn builtinSlice(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value{ .array = .empty };
    switch (args[0]) {
        .array => |a| {
            const n    = a.items.len;
            const from: usize = if (args.len >= 2) @max(0, @as(usize, @intCast(@max(0, args[1].toInt())))) else 0;
            const to_i: usize = if (args.len >= 3) @as(usize, @intCast(@max(0, args[2].toInt()))) else n;
            const to   = @min(to_i, n);
            var result = std.ArrayList(Value).empty;
            if (from < to) try result.appendSlice(pa, a.items[from..to]);
            return Value{ .array = result };
        },
        .string => |s| {
            const n    = s.len;
            const from = if (args.len >= 2) @min(@as(usize, @intCast(@max(0, args[1].toInt()))), n) else 0;
            const to_i = if (args.len >= 3) @as(usize, @intCast(@max(0, args[2].toInt()))) else n;
            const to   = @min(to_i, n);
            return Value{ .string = if (from < to) s[from..to] else "" };
        },
        else => return Value.Null,
    }
}

fn builtinJoin(args: []const Value) anyerror!Value {
    if (args.len == 0 or args[0] != .array) return Value{ .string = "" };
    const sep = if (args.len >= 2 and args[1] == .string) args[1].string else ",";
    var sb = std.ArrayList(u8).empty;
    for (args[0].array.items, 0..) |item, i| {
        if (i > 0) try sb.appendSlice(pa, sep);
        try sb.appendSlice(pa, try item.toString(pa));
    }
    return Value{ .string = try sb.toOwnedSlice(pa) };
}

fn builtinSplit(args: []const Value) anyerror!Value {
    var arr = std.ArrayList(Value).empty;
    if (args.len < 2 or args[0] != .string or args[1] != .string) return Value{ .array = arr };
    var it = std.mem.splitSequence(u8, args[0].string, args[1].string);
    while (it.next()) |part| try arr.append(pa, Value{ .string = part });
    return Value{ .array = arr };
}

fn builtinTrim(args: []const Value) anyerror!Value {
    if (args.len == 0 or args[0] != .string) return Value{ .string = "" };
    return Value{ .string = std.mem.trim(u8, args[0].string, &std.ascii.whitespace) };
}

fn builtinToUpperCase(args: []const Value) anyerror!Value {
    if (args.len == 0 or args[0] != .string) return Value{ .string = "" };
    const up = try pa.dupe(u8, args[0].string);
    for (up) |*c| c.* = std.ascii.toUpper(c.*);
    return Value{ .string = up };
}

fn builtinToLowerCase(args: []const Value) anyerror!Value {
    if (args.len == 0 or args[0] != .string) return Value{ .string = "" };
    const lo = try pa.dupe(u8, args[0].string);
    for (lo) |*c| c.* = std.ascii.toLower(c.*);
    return Value{ .string = lo };
}

fn builtinIncludes(args: []const Value) anyerror!Value {
    if (args.len < 2) return Value.False;
    return switch (args[0]) {
        .string => |s| Value{ .boolean = std.mem.indexOf(u8, s, args[1].string) != null },
        .array  => |a| blk: {
            for (a.items) |item| if (item.equals(args[1])) break :blk Value.True;
            break :blk Value.False;
        },
        else => Value.False,
    };
}

fn builtinIndexOf(args: []const Value) anyerror!Value {
    if (args.len < 2) return Value{ .int = -1 };
    return switch (args[0]) {
        .string => |s| {
            if (args[1] != .string) return Value{ .int = -1 };
            const idx = std.mem.indexOf(u8, s, args[1].string);
            return Value{ .int = if (idx) |i| @intCast(i) else -1 };
        },
        .array  => |a| {
            for (a.items, 0..) |item, i| if (item.equals(args[1])) return Value{ .int = @intCast(i) };
            return Value{ .int = -1 };
        },
        else => Value{ .int = -1 },
    };
}

fn builtinReplace(args: []const Value) anyerror!Value {
    if (args.len < 3 or args[0] != .string or args[1] != .string or args[2] != .string)
        return if (args.len > 0) args[0] else Value.Null;
    const result = try std.mem.replaceOwned(u8, pa, args[0].string, args[1].string, args[2].string);
    return Value{ .string = result };
}

fn builtinStartsWith(args: []const Value) anyerror!Value {
    if (args.len < 2 or args[0] != .string or args[1] != .string) return Value.False;
    return Value{ .boolean = std.mem.startsWith(u8, args[0].string, args[1].string) };
}

fn builtinEndsWith(args: []const Value) anyerror!Value {
    if (args.len < 2 or args[0] != .string or args[1] != .string) return Value.False;
    return Value{ .boolean = std.mem.endsWith(u8, args[0].string, args[1].string) };
}

fn builtinConcat(args: []const Value) anyerror!Value {
    var sb = std.ArrayList(u8).empty;
    for (args) |a| try sb.appendSlice(pa, try a.toString(pa));
    return Value{ .string = try sb.toOwnedSlice(pa) };
}

fn builtinReverse(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Null;
    return switch (args[0]) {
        .array => |a| { std.mem.reverse(Value, a.items); return args[0]; },
        .string => |s| blk: {
            const copy = try pa.dupe(u8, s);
            std.mem.reverse(u8, copy);
            break :blk Value{ .string = copy };
        },
        else => args[0],
    };
}

fn sortCompare(_: void, a: Value, b: Value) bool {
    return a.toFloat() < b.toFloat();
}

fn builtinSort(args: []const Value) anyerror!Value {
    if (args.len == 0 or args[0] != .array) return Value.Null;
    std.mem.sort(Value, @constCast(&args[0]).array.items, {}, sortCompare);
    return args[0];
}
