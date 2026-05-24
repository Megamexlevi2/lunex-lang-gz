// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

const std = @import("std");
const platform = @import("platform.zig");
const Allocator = std.mem.Allocator;

pub const HOT_FUNCTION_THRESHOLD: u32 = 100;
pub const HOT_LOOP_THRESHOLD: u32 = 1_000;
pub const MEGA_MORPHIC_THRESHOLD: u32 = 10_000;

pub const FunctionProfile = struct {
    invocations: u32 = 0,
    total_ns: u64 = 0,
    jit_compiled: bool = false,
    jit_addr: ?[*]u8 = null,
    inline_cache_hits: u32 = 0,
    inline_cache_misses: u32 = 0,
};

pub const LoopProfile = struct {
    iterations: u32 = 0,
    func_id: u32 = 0,
    pc_start: u32 = 0,
    jit_compiled: bool = false,
    jit_addr: ?[*]u8 = null,
};

pub const TypeFeedback = struct {
    seen_int: bool = false,
    seen_float: bool = false,
    seen_bool: bool = false,
    seen_string: bool = false,
    seen_object: bool = false,
    seen_null: bool = false,
    observation_count: u32 = 0,

    pub fn observe(self: *TypeFeedback, tag: u8) void {
        self.observation_count += 1;
        switch (tag) {
            0 => self.seen_null = true,
            1 => self.seen_bool = true,
            2 => self.seen_int = true,
            3 => self.seen_float = true,
            4 => self.seen_string = true,
            5 => self.seen_object = true,
            else => {},
        }
    }

    pub fn isMonomorphic(self: *const TypeFeedback) bool {
        var count: u8 = 0;
        if (self.seen_int) count += 1;
        if (self.seen_float) count += 1;
        if (self.seen_bool) count += 1;
        if (self.seen_string) count += 1;
        if (self.seen_object) count += 1;
        if (self.seen_null) count += 1;
        return count == 1;
    }

    pub fn dominantTag(self: *const TypeFeedback) u8 {
        if (self.seen_int) return 2;
        if (self.seen_float) return 3;
        if (self.seen_bool) return 1;
        if (self.seen_string) return 4;
        if (self.seen_object) return 5;
        return 0;
    }
};

pub const Profiler = struct {
    functions: std.AutoHashMap(u32, FunctionProfile),
    loops: std.AutoHashMap(u64, LoopProfile),
    type_feedback: std.AutoHashMap(u64, TypeFeedback),
    allocator: Allocator,
    total_compilations: u32,
    time_in_jit_ns: u64,
    time_in_interpreter_ns: u64,

    pub fn init(allocator: Allocator) Profiler {
        return .{
            .functions = std.AutoHashMap(u32, FunctionProfile).init(allocator),
            .loops = std.AutoHashMap(u64, LoopProfile).init(allocator),
            .type_feedback = std.AutoHashMap(u64, TypeFeedback).init(allocator),
            .allocator = allocator,
            .total_compilations = 0,
            .time_in_jit_ns = 0,
            .time_in_interpreter_ns = 0,
        };
    }

    pub fn deinit(self: *Profiler) void {
        self.functions.deinit();
        self.loops.deinit();
        self.type_feedback.deinit();
    }

    pub fn recordCall(self: *Profiler, func_id: u32) !bool {
        const entry = try self.functions.getOrPut(func_id);
        if (!entry.found_existing) {
            entry.value_ptr.* = FunctionProfile{};
        }
        entry.value_ptr.invocations += 1;
        if (entry.value_ptr.jit_compiled) return false;
        return entry.value_ptr.invocations >= HOT_FUNCTION_THRESHOLD;
    }

    pub fn recordLoopBack(self: *Profiler, func_id: u32, pc: u32) !bool {
        const key = (@as(u64, func_id) << 32) | @as(u64, pc);
        const entry = try self.loops.getOrPut(key);
        if (!entry.found_existing) {
            entry.value_ptr.* = LoopProfile{ .func_id = func_id, .pc_start = pc };
        }
        entry.value_ptr.iterations += 1;
        if (entry.value_ptr.jit_compiled) return false;
        return entry.value_ptr.iterations >= HOT_LOOP_THRESHOLD;
    }

    pub fn observeType(self: *Profiler, site_id: u64, tag: u8) !void {
        const entry = try self.type_feedback.getOrPut(site_id);
        if (!entry.found_existing) {
            entry.value_ptr.* = TypeFeedback{};
        }
        entry.value_ptr.observe(tag);
    }

    pub fn getFunctionProfile(self: *Profiler, func_id: u32) ?*FunctionProfile {
        return self.functions.getPtr(func_id);
    }

    pub fn markJITCompiled(self: *Profiler, func_id: u32, addr: [*]u8) void {
        if (self.functions.getPtr(func_id)) |p| {
            p.jit_compiled = true;
            p.jit_addr = addr;
            self.total_compilations += 1;
        }
    }

    pub fn getJITAddr(self: *Profiler, func_id: u32) ?[*]u8 {
        if (self.functions.getPtr(func_id)) |p| {
            if (p.jit_compiled) return p.jit_addr;
        }
        return null;
    }

    pub fn printStats(self: *Profiler) void {
        
        
        var buf: [512]u8 = undefined;
        const hdr = std.fmt.bufPrint(&buf,
            "[lunex-profiler] JIT compilations: {d}  JIT time: {d}us  Interp time: {d}us\n",
            .{ self.total_compilations, self.time_in_jit_ns / 1000, self.time_in_interpreter_ns / 1000 },
        ) catch return;
        platform.fdWrite(std.posix.STDERR_FILENO, hdr);
        var it = self.functions.iterator();
        while (it.next()) |entry| {
            if (entry.value_ptr.invocations > 10) {
                var line_buf: [128]u8 = undefined;
                const line = std.fmt.bufPrint(&line_buf,
                    "  fn#{d}: {d} calls, jit={}\n",
                    .{ entry.key_ptr.*, entry.value_ptr.invocations, entry.value_ptr.jit_compiled },
                ) catch continue;
                platform.fdWrite(std.posix.STDERR_FILENO, line);
            }
        }
    }
};
