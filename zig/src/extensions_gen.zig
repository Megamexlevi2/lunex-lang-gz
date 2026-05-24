// Lunex lang
// Created by David Dev - GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.
// AUTO-GENERATED - do not edit by hand.

const std = @import("std");
const value = @import("value.zig");
const Value = value.Value;

const ext_example = @import("extensions/example.zig");

pub fn registerAll(globals: *std.StringHashMap(Value), alloc: std.mem.Allocator) !void {
    _ = alloc;
    try ext_example.register(globals);
}
