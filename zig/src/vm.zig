// Lunex lang — Zig register-based virtual machine.
// Executes NTZ (Lunex Zig) bytecode produced by the Go NTZ compiler.
// The VM is a simple stack machine with support for integers, floats,
// booleans, strings, arrays, and objects.  A profiler tracks hot loops for
// future JIT compilation, and a lightweight arena allocator handles all
// value allocations within a single execution.

const std     = @import("std");
const value   = @import("value.zig");
const memory  = @import("memory.zig");
const builtins = @import("builtins.zig");
const platform = @import("platform.zig");
const profiler_mod = @import("profiler.zig");
const jit_mod = @import("jit.zig");

pub const Value    = value.Value;
pub const Allocator = std.mem.Allocator;

/// NTZ opcode set.  Values must stay in sync with the constants in
/// internal/bytecode/ntz.go on the Go side.
pub const Opcode = enum(u8) {
    Const,        //  0 — push a constant value
    Load,         //  1 — load local variable (slot u16)
    Store,        //  2 — store to local variable, leave value on stack (peek)
    Add,          //  3
    Sub,          //  4
    Mul,          //  5
    Div,          //  6
    Mod,          //  7
    Neg,          //  8 — unary negate
    Not,          //  9 — logical not
    BitAnd,       // 10
    BitOr,        // 11
    BitXor,       // 12
    BitNot,       // 13
    Shl,          // 14
    Shr,          // 15
    Eq,           // 16
    Neq,          // 17
    Lt,           // 18
    Lte,          // 19
    Gt,           // 20
    Gte,          // 21
    And,          // 22 — logical short-circuit and (both operands already evaluated)
    Or,           // 23 — logical short-circuit or (both operands already evaluated)
    Call,         // 24 — call function value on stack
    CallRT,       // 25 — call a named runtime built-in
    Return,       // 26 — return top-of-stack value from current frame
    Jump,         // 27 — unconditional jump (target u32)
    JumpIf,       // 28 — jump if top is truthy (target u32)
    JumpIfFalse,  // 29 — jump if top is falsy  (target u32)
    GetField,     // 30 — object field access
    SetField,     // 31 — object field assignment (peeks obj, pops value)
    GetIndex,     // 32 — array/string index access
    SetIndex,     // 33 — array element assignment (peeks array, pops value+index)
    MakeArray,    // 34 — pop n items and push an array (count u16)
    MakeObject,   // 35 — pop n key-value pairs and push an object (count u16)
    Spread,       // 36 — spread operator
    Iter,         // 37 — create iterator
    IterNext,     // 38 — advance iterator
    Throw,        // 39 — throw top-of-stack as exception
    Try,          // 40 — begin try block
    Catch,        // 41 — begin catch handler
    Halt,         // 42 — stop execution, return Null
    Pop,          // 43 — discard top of stack
    _,            // catch-all — unknown opcode → InvalidOpcode error
};

pub const FRAME_MAX: usize = 512;
pub const STACK_MAX: usize = 65536;
pub const CONST_MAX: usize = 65536;

pub const RuntimeError = error{
    StackOverflow,
    StackUnderflow,
    TypeError,
    DivisionByZero,
    UndefinedVariable,
    IndexOutOfBounds,
    InvalidOpcode,
    MaxFramesExceeded,
    OutOfMemory,
    Throw,
};

pub const CallFrame = struct {
    ip:      usize,
    base:    usize,
    func_id: u32,
};

pub const VM = struct {
    stack:     []Value,
    sp:        usize,
    frames:    [FRAME_MAX]CallFrame,
    fp:        usize,
    globals:   std.StringHashMap(Value),
    constants: []Value,
    n_consts:  usize,
    alloc:     Allocator,
    arena:     memory.Arena,
    profiler:  profiler_mod.Profiler,
    jit:       jit_mod.JITCompiler,
    interp_ns: u64,
    jit_ns:    u64,

    pub fn init(alloc: Allocator) !VM {
        const stack     = try alloc.alloc(Value, STACK_MAX);
        const constants = try alloc.alloc(Value, CONST_MAX);
        @memset(stack,     Value.Null);
        @memset(constants, Value.Null);
        return VM{
            .stack     = stack,
            .sp        = 0,
            .frames    = undefined,
            .fp        = 0,
            .globals   = std.StringHashMap(Value).init(alloc),
            .constants = constants,
            .n_consts  = 0,
            .alloc     = alloc,
            .arena     = memory.Arena.init(alloc),
            .profiler  = profiler_mod.Profiler.init(alloc),
            .jit       = jit_mod.JITCompiler.init(alloc),
            .interp_ns = 0,
            .jit_ns    = 0,
        };
    }

    pub fn deinit(self: *VM) void {
        self.alloc.free(self.stack);
        self.alloc.free(self.constants);
        self.globals.deinit();
        self.arena.deinit();
        self.profiler.deinit();
        self.jit.deinit();
    }

    inline fn push(self: *VM, v: Value) !void {
        if (self.sp >= STACK_MAX) return RuntimeError.StackOverflow;
        self.stack[self.sp] = v;
        self.sp += 1;
    }

    inline fn pop(self: *VM) !Value {
        if (self.sp == 0) return RuntimeError.StackUnderflow;
        self.sp -= 1;
        return self.stack[self.sp];
    }

    inline fn peek(self: *VM, offset: usize) Value {
        return self.stack[self.sp - 1 - offset];
    }

    inline fn readU8(code: []const u8, ip: *usize) u8 {
        const b = code[ip.*]; ip.* += 1; return b;
    }

    inline fn readU16(code: []const u8, ip: *usize) u16 {
        const lo: u16 = code[ip.*]; const hi: u16 = code[ip.* + 1];
        ip.* += 2; return lo | (hi << 8);
    }

    inline fn readU32(code: []const u8, ip: *usize) u32 {
        const a: u32 = code[ip.*]; const b2: u32 = code[ip.* + 1];
        const c: u32 = code[ip.* + 2]; const d: u32 = code[ip.* + 3];
        ip.* += 4; return a | (b2 << 8) | (c << 16) | (d << 24);
    }

    inline fn readI64(code: []const u8, ip: *usize) i64 {
        const v = std.mem.readInt(i64, code[ip.*..][0..8], .little);
        ip.* += 8; return v;
    }

    inline fn readF64(code: []const u8, ip: *usize) f64 {
        const bits = std.mem.readInt(u64, code[ip.*..][0..8], .little);
        ip.* += 8; return @bitCast(bits);
    }

    /// Execute a slice of raw NTZ opcodes.
    /// The data must already be the opcode slice — not a full NC container.
    /// Use extractNTZFromNC (main.zig) to decode the container first.
    pub fn execBytecode(self: *VM, code: []const u8) !Value {
        const t0 = platform.highResTimer();
        var ip: usize = 0;

        while (ip < code.len) {
            const op_byte = readU8(code, &ip);
            const op: Opcode = @enumFromInt(op_byte);

            switch (op) {
                .Const => {
                    const kind = readU8(code, &ip);
                    const val: Value = switch (kind) {
                        0 => Value.Null,
                        1 => blk: {
                            break :blk Value{ .boolean = readU8(code, &ip) != 0 };
                        },
                        2 => blk: {
                            break :blk Value{ .int = readI64(code, &ip) };
                        },
                        3 => blk: {
                            break :blk Value{ .float = readF64(code, &ip) };
                        },
                        4 => blk: {
                            const len = readU32(code, &ip);
                            const s = try self.arena.allocator().dupe(u8, code[ip .. ip + len]);
                            ip += len;
                            break :blk Value{ .string = s };
                        },
                        else => Value.Null,
                    };
                    try self.push(val);
                },

                .Load => {
                    const slot = readU16(code, &ip);
                    const base = if (self.fp > 0) self.frames[self.fp - 1].base else 0;
                    try self.push(self.stack[base + slot]);
                },

                .Store => {
                    const slot = readU16(code, &ip);
                    const base = if (self.fp > 0) self.frames[self.fp - 1].base else 0;
                    self.stack[base + slot] = self.peek(0);
                    // Store is a peek — value remains on stack as expression result.
                },

                .Pop => {
                    // Discard the top stack value.  Used after expression statements
                    // and variable declarations to keep the stack balanced.
                    _ = try self.pop();
                },

                .Add => {
                    const b2 = try self.pop();
                    const a  = try self.pop();
                    try self.push(try self.opAdd(a, b2));
                },

                .Sub => {
                    const b2 = try self.pop();
                    const a  = try self.pop();
                    try self.push(try self.opSub(a, b2));
                },

                .Mul => {
                    const b2 = try self.pop();
                    const a  = try self.pop();
                    try self.push(try self.opMul(a, b2));
                },

                .Div => {
                    const b2 = try self.pop();
                    const a  = try self.pop();
                    try self.push(try self.opDiv(a, b2));
                },

                .Mod => {
                    const b2 = try self.pop();
                    const a  = try self.pop();
                    try self.push(try self.opMod(a, b2));
                },

                .Neg => {
                    const a = try self.pop();
                    switch (a) {
                        .int   => |n| try self.push(Value{ .int   = -n }),
                        .float => |f| try self.push(Value{ .float = -f }),
                        else   => return RuntimeError.TypeError,
                    }
                },

                .Not => {
                    const a = try self.pop();
                    try self.push(Value{ .boolean = !a.isTruthy() });
                },

                .BitAnd => {
                    const b2 = try self.pop(); const a = try self.pop();
                    if (a != .int or b2 != .int) return RuntimeError.TypeError;
                    try self.push(Value{ .int = a.int & b2.int });
                },

                .BitOr => {
                    const b2 = try self.pop(); const a = try self.pop();
                    if (a != .int or b2 != .int) return RuntimeError.TypeError;
                    try self.push(Value{ .int = a.int | b2.int });
                },

                .BitXor => {
                    const b2 = try self.pop(); const a = try self.pop();
                    if (a != .int or b2 != .int) return RuntimeError.TypeError;
                    try self.push(Value{ .int = a.int ^ b2.int });
                },

                .BitNot => {
                    const a = try self.pop();
                    if (a != .int) return RuntimeError.TypeError;
                    try self.push(Value{ .int = ~a.int });
                },

                .Shl => {
                    const b2 = try self.pop(); const a = try self.pop();
                    if (a != .int or b2 != .int) return RuntimeError.TypeError;
                    try self.push(Value{ .int = a.int << @intCast(b2.int & 63) });
                },

                .Shr => {
                    const b2 = try self.pop(); const a = try self.pop();
                    if (a != .int or b2 != .int) return RuntimeError.TypeError;
                    try self.push(Value{ .int = a.int >> @intCast(b2.int & 63) });
                },

                .Eq => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(Value{ .boolean = a.equals(b2) });
                },

                .Neq => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(Value{ .boolean = !a.equals(b2) });
                },

                .Lt => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(Value{ .boolean = try self.opCmp(a, b2) < 0 });
                },

                .Lte => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(Value{ .boolean = try self.opCmp(a, b2) <= 0 });
                },

                .Gt => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(Value{ .boolean = try self.opCmp(a, b2) > 0 });
                },

                .Gte => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(Value{ .boolean = try self.opCmp(a, b2) >= 0 });
                },

                .And => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(if (a.isTruthy()) b2 else a);
                },

                .Or => {
                    const b2 = try self.pop(); const a = try self.pop();
                    try self.push(if (a.isTruthy()) a else b2);
                },

                .Jump => {
                    ip = readU32(code, &ip);
                },

                .JumpIf => {
                    const target = readU32(code, &ip);
                    const cond   = try self.pop();
                    if (cond.isTruthy()) ip = target;
                },

                .JumpIfFalse => {
                    const target  = readU32(code, &ip);
                    const cond    = try self.pop();
                    const func_id = if (self.fp > 0) self.frames[self.fp - 1].func_id else 0;
                    if (!cond.isTruthy()) {
                        const is_hot = self.profiler.recordLoopBack(func_id, @intCast(target)) catch false;
                        if (is_hot) {
                            // Hot loop detected: JIT-compile the full bytecode of this
                            // function and cache it.  compileHot is idempotent — if the
                            // function is already in the cache it returns immediately.
                            _ = self.jit.compileHot(@as(u64, func_id), code);
                        }
                        ip = target;
                    }
                },

                .Call => {
                    const n_args = readU8(code, &ip);
                    const callee = self.stack[self.sp - 1 - n_args];

                    switch (callee) {
                        .native => |nfn| {
                            const args   = self.stack[self.sp - n_args .. self.sp];
                            const result = try nfn(args);
                            self.sp -= n_args + 1;
                            try self.push(result);
                        },
                        else => {
                            self.sp -= n_args + 1;
                            try self.push(Value.Null);
                        },
                    }
                },

                .CallRT => {
                    const name_len = readU8(code, &ip);
                    const name     = code[ip .. ip + name_len];
                    ip += name_len;
                    const n_args = readU8(code, &ip);
                    if (self.globals.get(name)) |gval| {
                        switch (gval) {
                            .native => |nfn| {
                                const args   = self.stack[self.sp - n_args .. self.sp];
                                const result = try nfn(args);
                                self.sp -= n_args;
                                try self.push(result);
                            },
                            else => return RuntimeError.TypeError,
                        }
                    } else {
                        return RuntimeError.UndefinedVariable;
                    }
                },

                .Return => {
                    const result = try self.pop();
                    if (self.fp == 0) {
                        self.interp_ns += platform.highResTimer() - t0;
                        return result;
                    }
                    self.fp -= 1;
                    const frame = self.frames[self.fp];
                    self.sp  = frame.base;
                    ip       = frame.ip;
                    try self.push(result);
                },

                .GetField => {
                    const name_len = readU8(code, &ip);
                    const name     = code[ip .. ip + name_len];
                    ip += name_len;
                    const obj = try self.pop();
                    switch (obj) {
                        .object => |map| {
                            try self.push(map.get(name) orelse Value.Null);
                        },
                        .string => |s| {
                            if (std.mem.eql(u8, name, "length")) {
                                try self.push(Value{ .int = @intCast(s.len) });
                            } else {
                                try self.push(Value.Null);
                            }
                        },
                        .array  => |a| {
                            if (std.mem.eql(u8, name, "length")) {
                                try self.push(Value{ .int = @intCast(a.items.len) });
                            } else {
                                try self.push(Value.Null);
                            }
                        },
                        else => return RuntimeError.TypeError,
                    }
                },

                .SetField => {
                    // Stack before: [..., obj, value]   (value on top)
                    // Pops value, peeks obj, sets obj.field = value.
                    // Obj remains on stack as the expression result.
                    const name_len = readU8(code, &ip);
                    const name     = code[ip .. ip + name_len];
                    ip += name_len;
                    const val2 = try self.pop();
                    const obj  = self.peek(0);
                    switch (obj) {
                        .object => |*map| {
                            const owned = try self.arena.allocator().dupe(u8, name);
                            try @constCast(map).put(owned, val2);
                        },
                        else => return RuntimeError.TypeError,
                    }
                },

                .GetIndex => {
                    const idx = try self.pop();
                    const arr = try self.pop();
                    switch (arr) {
                        .array => |a| {
                            if (idx != .int) return RuntimeError.TypeError;
                            const i: usize = @intCast(idx.int);
                            if (i >= a.items.len) return RuntimeError.IndexOutOfBounds;
                            try self.push(a.items[i]);
                        },
                        .string => |s| {
                            if (idx != .int) return RuntimeError.TypeError;
                            const i: usize = @intCast(idx.int);
                            if (i >= s.len) return RuntimeError.IndexOutOfBounds;
                            const ch = try std.fmt.allocPrint(self.arena.allocator(), "{c}", .{s[i]});
                            try self.push(Value{ .string = ch });
                        },
                        .object => |map| {
                            if (idx != .string) return RuntimeError.TypeError;
                            try self.push(map.get(idx.string) orelse Value.Null);
                        },
                        else => return RuntimeError.TypeError,
                    }
                },

                .SetIndex => {
                    // Stack: [..., arr, idx, value]  (value on top)
                    // Pops value and idx, peeks arr, sets arr[idx] = value.
                    const val2 = try self.pop();
                    const idx  = try self.pop();
                    const arr  = self.peek(0);
                    switch (arr) {
                        .array => |a| {
                            if (idx != .int) return RuntimeError.TypeError;
                            const i: usize = @intCast(idx.int);
                            if (i >= a.items.len) return RuntimeError.IndexOutOfBounds;
                            @constCast(&a).items[i] = val2;
                        },
                        else => return RuntimeError.TypeError,
                    }
                },

                .MakeArray => {
                    const n = readU16(code, &ip);
                    var arr = try std.ArrayList(Value).initCapacity(self.arena.allocator(), n);
                    arr.items.len = n;
                    var i: usize = n;
                    while (i > 0) {
                        i -= 1;
                        arr.items[i] = try self.pop();
                    }
                    try self.push(Value{ .array = arr });
                },

                .MakeObject => {
                    // Pops n pairs of (key, value).  Keys must be strings.
                    const n = readU16(code, &ip);
                    var map = std.StringHashMap(Value).init(self.arena.allocator());
                    var i: usize = 0;
                    while (i < n) : (i += 1) {
                        const v2 = try self.pop();
                        const k2 = try self.pop();
                        if (k2 != .string) return RuntimeError.TypeError;
                        try map.put(k2.string, v2);
                    }
                    try self.push(Value{ .object = map });
                },

                .Throw => {
                    _ = try self.pop();
                    return RuntimeError.Throw;
                },

                .Halt => {
                    self.interp_ns += platform.highResTimer() - t0;
                    return Value.Null;
                },

                else => return RuntimeError.InvalidOpcode,
            }
        }

        self.interp_ns += platform.highResTimer() - t0;
        return Value.Null;
    }

    fn opAdd(self: *VM, a: Value, b2: Value) !Value {
        switch (a) {
            .int   => |n| switch (b2) {
                .int   => |m| return Value{ .int   = n +% m },
                .float => |f| return Value{ .float = @as(f64, @floatFromInt(n)) + f },
                else   => return RuntimeError.TypeError,
            },
            .float => |f| switch (b2) {
                .int   => |m| return Value{ .float = f + @as(f64, @floatFromInt(m)) },
                .float => |g| return Value{ .float = f + g },
                else   => return RuntimeError.TypeError,
            },
            .string => |s| {
                const bs = try b2.toString(self.arena.allocator());
                const r  = try std.mem.concat(self.arena.allocator(), u8, &.{ s, bs });
                return Value{ .string = r };
            },
            else => return RuntimeError.TypeError,
        }
    }

    fn opSub(_: *VM, a: Value, b2: Value) !Value {
        switch (a) {
            .int   => |n| switch (b2) {
                .int   => |m| return Value{ .int   = n -% m },
                .float => |f| return Value{ .float = @as(f64, @floatFromInt(n)) - f },
                else   => return RuntimeError.TypeError,
            },
            .float => |f| switch (b2) {
                .int   => |m| return Value{ .float = f - @as(f64, @floatFromInt(m)) },
                .float => |g| return Value{ .float = f - g },
                else   => return RuntimeError.TypeError,
            },
            else => return RuntimeError.TypeError,
        }
    }

    fn opMul(_: *VM, a: Value, b2: Value) !Value {
        switch (a) {
            .int   => |n| switch (b2) {
                .int   => |m| return Value{ .int   = n *% m },
                .float => |f| return Value{ .float = @as(f64, @floatFromInt(n)) * f },
                else   => return RuntimeError.TypeError,
            },
            .float => |f| switch (b2) {
                .int   => |m| return Value{ .float = f * @as(f64, @floatFromInt(m)) },
                .float => |g| return Value{ .float = f * g },
                else   => return RuntimeError.TypeError,
            },
            else => return RuntimeError.TypeError,
        }
    }

    fn opDiv(_: *VM, a: Value, b2: Value) !Value {
        switch (a) {
            .int   => |n| switch (b2) {
                .int   => |m| {
                    if (m == 0) return RuntimeError.DivisionByZero;
                    return Value{ .int = @divTrunc(n, m) };
                },
                .float => |f| return Value{ .float = @as(f64, @floatFromInt(n)) / f },
                else   => return RuntimeError.TypeError,
            },
            .float => |f| switch (b2) {
                .int   => |m| return Value{ .float = f / @as(f64, @floatFromInt(m)) },
                .float => |g| return Value{ .float = f / g },
                else   => return RuntimeError.TypeError,
            },
            else => return RuntimeError.TypeError,
        }
    }

    fn opMod(_: *VM, a: Value, b2: Value) !Value {
        if (a != .int or b2 != .int) return RuntimeError.TypeError;
        if (b2.int == 0) return RuntimeError.DivisionByZero;
        return Value{ .int = @mod(a.int, b2.int) };
    }

    fn opCmp(_: *VM, a: Value, b2: Value) !i8 {
        switch (a) {
            .int   => |n| switch (b2) {
                .int   => |m| return if (n < m) -1 else if (n > m) 1 else 0,
                .float => |f| {
                    const af: f64 = @floatFromInt(n);
                    return if (af < f) -1 else if (af > f) 1 else 0;
                },
                else => return RuntimeError.TypeError,
            },
            .float => |f| switch (b2) {
                .int   => |m| {
                    const bf: f64 = @floatFromInt(m);
                    return if (f < bf) -1 else if (f > bf) 1 else 0;
                },
                .float => |g| return if (f < g) -1 else if (f > g) 1 else 0,
                else   => return RuntimeError.TypeError,
            },
            .string => |s| {
                if (b2 != .string) return RuntimeError.TypeError;
                return switch (std.mem.order(u8, s, b2.string)) {
                    .lt => -1,
                    .gt => 1,
                    .eq => 0,
                };
            },
            else => return RuntimeError.TypeError,
        }
    }

    pub fn setGlobal(self: *VM, name: []const u8, val: Value) !void {
        const owned = try self.alloc.dupe(u8, name);
        try self.globals.put(owned, val);
    }

    pub fn getGlobal(self: *VM, name: []const u8) ?Value {
        return self.globals.get(name);
    }

    pub fn printStats(self: *VM) void {
        var buf: [256]u8 = undefined;
        const msg = std.fmt.bufPrint(&buf, "[vm] interp={d}us  jit_units={d}\n", .{
            self.interp_ns / 1000, self.jit.totalUnits(),
        }) catch "";
        platform.fdWrite(std.posix.STDERR_FILENO, msg);
        self.profiler.printStats();
        self.jit.printStats();
    }
};
