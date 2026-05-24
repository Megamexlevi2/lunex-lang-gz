// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

const std = @import("std");

pub const BuiltinFn = *const fn (args: []const Value) anyerror!Value;

pub const Value = union(enum) {
    null_val: void,
    boolean:  bool,
    int:      i64,
    float:    f64,
    string:   []const u8,
    array:    std.ArrayList(Value),
    object:   std.StringHashMap(Value),
    native:   BuiltinFn,
    closure:  *const FuncProto,

    pub const Null  = Value{ .null_val = {} };
    pub const True  = Value{ .boolean  = true };
    pub const False = Value{ .boolean  = false };
    pub const Zero  = Value{ .int      = 0 };
    pub const One   = Value{ .int      = 1 };

    pub fn mkInt(n: i64) Value   { return Value{ .int   = n }; }
    pub fn mkFloat(f: f64) Value { return Value{ .float = f }; }
    pub fn mkBool(b: bool) Value { return Value{ .boolean = b }; }
    pub fn mkStr(s: []const u8) Value { return Value{ .string = s }; }

    pub fn isTruthy(self: Value) bool {
        return switch (self) {
            .null_val => false,
            .boolean  => |b| b,
            .int      => |n| n != 0,
            .float    => |f| f != 0.0 and !std.math.isNan(f),
            .string   => |s| s.len > 0,
            .array    => |a| a.items.len > 0,
            .object   => |o| o.count() > 0,
            .native, .closure => true,
        };
    }

    pub fn isNullish(self: Value) bool {
        return self == .null_val;
    }

    pub fn toFloat(self: Value) f64 {
        return switch (self) {
            .int     => |n| @as(f64, @floatFromInt(n)),
            .float   => |f| f,
            .boolean => |b| if (b) 1.0 else 0.0,
            .string  => |s| std.fmt.parseFloat(f64, s) catch std.math.nan(f64),
            .null_val => 0.0,
            else => std.math.nan(f64),
        };
    }

    pub fn toInt(self: Value) i64 {
        return switch (self) {
            .int     => |n| n,
            .float   => |f| @as(i64, @intFromFloat(f)),
            .boolean => |b| if (b) 1 else 0,
            .string  => |s| std.fmt.parseInt(i64, s, 10) catch 0,
            .null_val => 0,
            else => 0,
        };
    }

    pub fn toString(self: Value, alloc: std.mem.Allocator) ![]const u8 {
        return switch (self) {
            .null_val => "null",
            .boolean  => |b| if (b) "true" else "false",
            .int      => |n| try std.fmt.allocPrint(alloc, "{d}", .{n}),
            .float    => |f| blk: {
                if (std.math.isNan(f))      break :blk "NaN";
                if (std.math.isInf(f))      break :blk if (f > 0) "Infinity" else "-Infinity";
                const i: i64 = @intFromFloat(f);
                if (@as(f64, @floatFromInt(i)) == f) {
                    break :blk try std.fmt.allocPrint(alloc, "{d}", .{i});
                }
                break :blk try std.fmt.allocPrint(alloc, "{d}", .{f});
            },
            .string   => |s| s,
            .array    => |arr| blk: {
                
                var sb = std.ArrayList(u8).empty;
                try sb.append(alloc, '[');
                for (arr.items, 0..) |item, i| {
                    if (i > 0) try sb.appendSlice(alloc, ", ");
                    try sb.appendSlice(alloc, try item.toString(alloc));
                }
                try sb.append(alloc, ']');
                break :blk try sb.toOwnedSlice(alloc);
            },
            .object   => |obj| blk: {
                var sb = std.ArrayList(u8).empty;
                try sb.append(alloc, '{');
                var it    = obj.iterator();
                var first = true;
                while (it.next()) |entry| {
                    if (!first) try sb.appendSlice(alloc, ", ");
                    first = false;
                    try sb.appendSlice(alloc, entry.key_ptr.*);
                    try sb.appendSlice(alloc, ": ");
                    try sb.appendSlice(alloc, try entry.value_ptr.toString(alloc));
                }
                try sb.append(alloc, '}');
                break :blk try sb.toOwnedSlice(alloc);
            },
            .native  => "[native function]",
            .closure => "[function]",
        };
    }

    pub fn equals(self: Value, other: Value) bool {
        switch (self) {
            .null_val => return other == .null_val,
            .boolean  => |b|  return other == .boolean and other.boolean == b,
            .int      => |n| switch (other) {
                .int   => |m|  return n == m,
                .float => |f|  return @as(f64, @floatFromInt(n)) == f,
                else   => return false,
            },
            .float    => |f| switch (other) {
                .float => |g|  return f == g,
                .int   => |m|  return f == @as(f64, @floatFromInt(m)),
                else   => return false,
            },
            .string   => |s| return other == .string and std.mem.eql(u8, s, other.string),
            else      => return false,
        }
    }

    pub fn typeName(self: Value) []const u8 {
        return switch (self) {
            .null_val => "null",
            .boolean  => "boolean",
            .int      => "number",
            .float    => "number",
            .string   => "string",
            .array    => "array",
            .object   => "object",
            .native   => "function",
            .closure  => "function",
        };
    }
};

pub const FuncProto = struct {
    name:        []const u8,
    reg_count:   u32,
    code:        []u32,
    constants:   []Value,
    param_count: u32,
    children:    []FuncProto,
};

pub fn toString(v: Value, alloc: std.mem.Allocator) ![]const u8 {
    return v.toString(alloc);
}
