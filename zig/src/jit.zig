// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
//
//
//   Tier 0 — Interpreter (vm.zig): always available, runs all bytecode.
//   Tier 1 — Baseline JIT (this file): compiles hot functions to native code.
//
// When the profiler (profiler.zig) detects that a bytecode chunk has been
// called more than HOT_THRESHOLD times, the VM dispatches here to compile it
// to native machine code.  On subsequent calls the native code runs directly
// without interpreter overhead.
//
// Architecture support
// --------------------
//   x86_64:  full emitter with 64-bit immediates, jumps, and arithmetic.
//   aarch64: full emitter with movz/movk, b.cond, and arithmetic.
//   riscv64: full emitter using RV64IMAC instruction set.
//   others:  compilation returns error.UnsupportedArch; the VM falls back to
//            the interpreter transparently.
//
// Calling convention
// ------------------
// Each compiled function matches this C-ABI signature:
//
//   i64 compiled_fn(i64 *locals, i64 *globals)
//
// The JIT registers allocate fixed roles so we do not need a register
// allocator for this baseline tier:
//
//   x86_64   : RBX = locals ptr, RAX = accumulator, RCX = rhs scratch
//   aarch64  : x19 = locals ptr, x0  = accumulator, x1  = rhs scratch
//   riscv64  : s1  = locals ptr, t0  = accumulator, t1  = rhs scratch

const std     = @import("std");
const builtin = @import("builtin");
const platform = @import("platform.zig");

// ─── Opcodes (mirrors vm.zig) ─────────────────────────────────────────────────

pub const Op = enum(u8) {
    LoadConst   = 0x01,
    LoadLocal   = 0x02,
    StoreLocal  = 0x03,
    Add         = 0x10,
    Sub         = 0x11,
    Mul         = 0x12,
    Div         = 0x13,
    Mod         = 0x14,
    Neg         = 0x15,
    Eq          = 0x20,
    Ne          = 0x21,
    Lt          = 0x22,
    Le          = 0x23,
    Gt          = 0x24,
    Ge          = 0x25,
    JumpIf      = 0x30,
    JumpIfFalse = 0x31,
    Jump        = 0x32,
    Ret         = 0x40,
    Call        = 0x41,
    Nop         = 0xFF,
    _,
};

// ─── CodeBuffer ───────────────────────────────────────────────────────────────
// Wraps executable memory.  Uses platform.zig for portable mmap/VirtualAlloc.

pub const CodeBuffer = struct {
    mem:  []u8,
    len:  usize,
    alloc: std.mem.Allocator,

    const DEFAULT_CAPACITY = 65536;

    pub fn init(alloc: std.mem.Allocator) !CodeBuffer {
        const mem = try platform.allocExecMem(DEFAULT_CAPACITY);
        return CodeBuffer{ .mem = mem, .len = 0, .alloc = alloc };
    }

    pub fn deinit(self: *CodeBuffer) void {
        platform.freeExecMem(self.mem);
    }

    /// Make the buffer executable (called after emitting all code).
    pub fn seal(self: *CodeBuffer) !void {
        try platform.makeExec(self.mem[0..self.len]);
    }

    pub fn emit(self: *CodeBuffer, byte: u8) !void {
        if (self.len >= self.mem.len) return error.CodeBufferFull;
        self.mem[self.len] = byte;
        self.len += 1;
    }

    pub fn emitSlice(self: *CodeBuffer, data: []const u8) !void {
        if (self.len + data.len > self.mem.len) return error.CodeBufferFull;
        @memcpy(self.mem[self.len..][0..data.len], data);
        self.len += data.len;
    }

    pub fn emitU32LE(self: *CodeBuffer, v: u32) !void {
        try self.emit(@truncate(v));
        try self.emit(@truncate(v >> 8));
        try self.emit(@truncate(v >> 16));
        try self.emit(@truncate(v >> 24));
    }

    pub fn emitU64LE(self: *CodeBuffer, v: u64) !void {
        try self.emitU32LE(@truncate(v));
        try self.emitU32LE(@truncate(v >> 32));
    }

    pub fn patchU32LE(self: *CodeBuffer, offset: usize, v: u32) void {
        self.mem[offset + 0] = @truncate(v);
        self.mem[offset + 1] = @truncate(v >> 8);
        self.mem[offset + 2] = @truncate(v >> 16);
        self.mem[offset + 3] = @truncate(v >> 24);
    }

    pub fn patchU8(self: *CodeBuffer, offset: usize, v: u8) void {
        self.mem[offset] = v;
    }

    pub fn currentOffset(self: *const CodeBuffer) usize { return self.len; }

    /// Return a callable function pointer to the beginning of emitted code.
    pub fn getFnPtr(self: *const CodeBuffer) *const fn ([*]i64, [*]i64) callconv(.c) i64 {
        return @ptrCast(@alignCast(self.mem.ptr));
    }

    /// Return a callable function pointer to an arbitrary offset.
    pub fn getFnPtrAt(self: *const CodeBuffer, offset: usize) *const fn ([*]i64, [*]i64) callconv(.c) i64 {
        return @ptrCast(@alignCast(self.mem[offset..].ptr));
    }
};

// ─── Patch ────────────────────────────────────────────────────────────────────

const Patch = struct {
    /// Offset of the field in the code buffer that needs patching.
    code_offset: usize,
    /// Bytecode offset this patch targets (the branch destination).
    bc_target:   usize,
    kind:        PatchKind,
};

const PatchKind = enum {
    rel32_x86,   // 32-bit relative offset for x86-64 Jcc/JMP
    imm19_a64,   // 19-bit immediate for AArch64 B.cond / CBZ / CBNZ
    imm26_a64,   // 26-bit immediate for AArch64 B / BL
    imm13_rv64,  // 13-bit (B-type) immediate for RISC-V branches
    imm21_rv64,  // 21-bit (J-type) immediate for RISC-V JAL
};

// ─── CompiledFunction ─────────────────────────────────────────────────────────

pub const CompiledFunction = struct {
    buf:         CodeBuffer,
    entry_point: usize,    // offset into buf where the function starts

    pub fn deinit(self: *CompiledFunction) void { self.buf.deinit(); }

    pub fn call(self: *const CompiledFunction, locals: [*]i64, globals: [*]i64) i64 {
        const fn_ptr = self.buf.getFnPtrAt(self.entry_point);
        return fn_ptr(locals, globals);
    }
};

// ─── JIT Compiler ─────────────────────────────────────────────────────────────

pub const JitCompiler = struct {
    alloc: std.mem.Allocator,

    pub fn init(alloc: std.mem.Allocator) JitCompiler {
        return JitCompiler{ .alloc = alloc };
    }

    /// Compile the bytecode slice `code` and return a CompiledFunction.
    /// Caller owns the result; call deinit() when done.
    pub fn compile(self: *JitCompiler, code: []const u8) !CompiledFunction {
        return switch (builtin.cpu.arch) {
            .x86_64  => compileX86_64(self.alloc, code),
            .aarch64 => compileAarch64(self.alloc, code),
            .riscv64 => compileRiscv64(self.alloc, code),
            else     => error.UnsupportedArch,
        };
    }
};

// ─── JIT Cache ────────────────────────────────────────────────────────────────

pub const JitCache = struct {
    alloc:     std.mem.Allocator,
    functions: std.AutoHashMap(u64, CompiledFunction),

    pub fn init(alloc: std.mem.Allocator) JitCache {
        return JitCache{
            .alloc     = alloc,
            .functions = std.AutoHashMap(u64, CompiledFunction).init(alloc),
        };
    }

    pub fn deinit(self: *JitCache) void {
        var it = self.functions.valueIterator();
        while (it.next()) |fn_ptr| fn_ptr.deinit();
        self.functions.deinit();
    }

    pub fn get(self: *JitCache, id: u64) ?*const CompiledFunction {
        return self.functions.getPtr(id);
    }

    pub fn put(self: *JitCache, id: u64, cf: CompiledFunction) !void {
        try self.functions.put(id, cf);
    }

    pub fn evict(self: *JitCache, id: u64) void {
        if (self.functions.fetchRemove(id)) |kv| {
            var cf = kv.value;
            cf.deinit();
        }
    }
};

// ═══════════════════════════════════════════════════════════════════════════════
// x86-64 Backend
// ═══════════════════════════════════════════════════════════════════════════════
//
// Register assignments:
//   RBX = locals pointer (callee-saved; preserved across calls)
//   RAX = accumulator (lhs, result)
//   RCX = rhs scratch
//   RDX = div/mod scratch (RDX:RAX dividend)
//   RBP = frame pointer (saved/restored in prologue/epilogue)
//   RSP = stack pointer

fn compileX86_64(alloc: std.mem.Allocator, code: []const u8) !CompiledFunction {
    var cb   = try CodeBuffer.init(alloc);
    errdefer cb.deinit();

    // Forward-jump patches: patch back after we know destination offsets.
    var patches: std.ArrayList(Patch) = .empty;
    defer patches.deinit(alloc);

    // Maps bytecode offset → code buffer offset (for back-patches).
    var bc_to_code = std.AutoHashMap(usize, usize).init(alloc);
    defer bc_to_code.deinit();

    // ── Prologue ──────────────────────────────────────────────────────────────
    // push rbp; push rbx; mov rbp, rsp; mov rbx, rdi
    try cb.emitSlice(&[_]u8{ 0x55 });           // push rbp
    try cb.emitSlice(&[_]u8{ 0x53 });           // push rbx
    try cb.emitSlice(&[_]u8{ 0x48, 0x89, 0xE5 }); // mov rbp, rsp
    try cb.emitSlice(&[_]u8{ 0x48, 0x89, 0xFB }); // mov rbx, rdi  (locals ptr)
    try cb.emitSlice(&[_]u8{ 0x48, 0x31, 0xC0 }); // xor rax, rax  (zero acc)

    // ── Bytecode loop ─────────────────────────────────────────────────────────
    var pc: usize = 0;
    while (pc < code.len) {
        try bc_to_code.put(pc, cb.currentOffset());

        const op: Op = @enumFromInt(code[pc]);
        pc += 1;

        switch (op) {
            .Nop => {},

            .LoadConst => {
                if (pc + 8 > code.len) return error.UnexpectedEOF;
                const v = std.mem.readInt(i64, code[pc..][0..8], .little);
                pc += 8;
                // mov rax, imm64
                try cb.emitSlice(&[_]u8{ 0x48, 0xB8 });
                try cb.emitU64LE(@bitCast(v));
            },

            .LoadLocal => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const idx: u8 = code[pc]; pc += 1;
                // mov rax, [rbx + idx*8]
                try cb.emitSlice(&[_]u8{ 0x48, 0x8B, 0x43 });
                try cb.emit(@intCast(idx * 8));
            },

            .StoreLocal => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const idx: u8 = code[pc]; pc += 1;
                // mov [rbx + idx*8], rax
                try cb.emitSlice(&[_]u8{ 0x48, 0x89, 0x43 });
                try cb.emit(@intCast(idx * 8));
            },

            .Add, .Sub, .Mul, .Div, .Mod => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                const b: u8 = code[pc]; pc += 1;
                // mov rax, [rbx + a*8]
                try cb.emitSlice(&[_]u8{ 0x48, 0x8B, 0x43, @intCast(a * 8) });
                // mov rcx, [rbx + b*8]
                try cb.emitSlice(&[_]u8{ 0x48, 0x8B, 0x4B, @intCast(b * 8) });
                switch (op) {
                    .Add => try cb.emitSlice(&[_]u8{ 0x48, 0x01, 0xC8 }), // add rax, rcx
                    .Sub => try cb.emitSlice(&[_]u8{ 0x48, 0x29, 0xC8 }), // sub rax, rcx
                    .Mul => try cb.emitSlice(&[_]u8{ 0x48, 0x0F, 0xAF, 0xC1 }), // imul rax, rcx
                    .Div => {
                        // cqo; idiv rcx
                        try cb.emitSlice(&[_]u8{ 0x48, 0x99 });         // cqo
                        try cb.emitSlice(&[_]u8{ 0x48, 0xF7, 0xF9 });   // idiv rcx
                    },
                    .Mod => {
                        // cqo; idiv rcx; mov rax, rdx
                        try cb.emitSlice(&[_]u8{ 0x48, 0x99 });
                        try cb.emitSlice(&[_]u8{ 0x48, 0xF7, 0xF9 });
                        try cb.emitSlice(&[_]u8{ 0x48, 0x89, 0xD0 }); // mov rax, rdx
                    },
                    else => unreachable,
                }
            },

            .Neg => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                // mov rax, [rbx + a*8]; neg rax
                try cb.emitSlice(&[_]u8{ 0x48, 0x8B, 0x43, @intCast(a * 8) });
                try cb.emitSlice(&[_]u8{ 0x48, 0xF7, 0xD8 }); // neg rax
            },

            .Eq, .Ne, .Lt, .Le, .Gt, .Ge => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                const b: u8 = code[pc]; pc += 1;
                // mov rax, [rbx+a*8]; cmp rax, [rbx+b*8]; setcc al; movzx rax, al
                try cb.emitSlice(&[_]u8{ 0x48, 0x8B, 0x43, @intCast(a * 8) });
                try cb.emitSlice(&[_]u8{ 0x48, 0x3B, 0x43, @intCast(b * 8) });
                const setcc: u8 = switch (op) {
                    .Eq => 0x94, .Ne => 0x95, .Lt => 0x9C,
                    .Le => 0x9E, .Gt => 0x9F, .Ge => 0x9D,
                    else => unreachable,
                };
                try cb.emitSlice(&[_]u8{ 0x0F, setcc, 0xC0 });    // setcc al
                try cb.emitSlice(&[_]u8{ 0x48, 0x0F, 0xB6, 0xC0 }); // movzx rax, al
            },

            .JumpIf, .JumpIfFalse, .Jump => {
                if (op == .Jump) {
                    if (pc + 2 > code.len) return error.UnexpectedEOF;
                    const tgt = std.mem.readInt(u16, code[pc..][0..2], .little);
                    pc += 2;
                    // jmp rel32  (patch later if forward)
                    try cb.emitSlice(&[_]u8{ 0xE9 }); // JMP rel32
                    const patch_off = cb.currentOffset();
                    try cb.emitU32LE(0xDEADBEEF);
                    try patches.append(alloc, .{ .code_offset = patch_off, .bc_target = tgt, .kind = .rel32_x86 });
                } else {
                    if (pc + 3 > code.len) return error.UnexpectedEOF;
                    const cond_idx: u8 = code[pc]; pc += 1;
                    const tgt = std.mem.readInt(u16, code[pc..][0..2], .little);
                    pc += 2;
                    // mov rax, [rbx + cond*8]; test rax, rax; jz/jnz rel32
                    try cb.emitSlice(&[_]u8{ 0x48, 0x8B, 0x43, @intCast(cond_idx * 8) });
                    try cb.emitSlice(&[_]u8{ 0x48, 0x85, 0xC0 }); // test rax, rax
                    const jcc: u8 = if (op == .JumpIf) 0x85 else 0x84; // JNZ / JZ
                    try cb.emitSlice(&[_]u8{ 0x0F, jcc });
                    const patch_off = cb.currentOffset();
                    try cb.emitU32LE(0xDEADBEEF);
                    try patches.append(alloc, .{ .code_offset = patch_off, .bc_target = tgt, .kind = .rel32_x86 });
                }
            },

            .Ret => {
                // Epilogue: pop rbx; pop rbp; ret
                try cb.emitSlice(&[_]u8{ 0x5B, 0x5D, 0xC3 });
            },

            else => return error.UnknownOpcode,
        }
    }

    // Implicit return at end of function body.
    try cb.emitSlice(&[_]u8{ 0x5B, 0x5D, 0xC3 }); // pop rbx; pop rbp; ret

    // ── Patch forward jumps ───────────────────────────────────────────────────
    for (patches.items) |patch| {
        const dst_code = bc_to_code.get(patch.bc_target) orelse return error.BadJumpTarget;
        const after_patch = patch.code_offset + 4;
        const rel: i32 = @intCast(@as(i64, @intCast(dst_code)) - @as(i64, @intCast(after_patch)));
        cb.patchU32LE(patch.code_offset, @bitCast(rel));
    }

    try cb.seal();
    return CompiledFunction{ .buf = cb, .entry_point = 0 };
}

// ═══════════════════════════════════════════════════════════════════════════════
// AArch64 Backend
// ═══════════════════════════════════════════════════════════════════════════════
//
// Register assignments:
//   x19 = locals pointer (callee-saved)
//   x0  = accumulator / return value
//   x1  = rhs scratch
//   x2  = scratch C
//   x29 = frame pointer (callee-saved)
//   x30 = link register  (callee-saved)

const A64_X0: u5  = 0;
const A64_X1: u5  = 1;
const A64_X2: u5  = 2;
const A64_X19: u5 = 19;
const A64_X29: u5 = 29; // frame pointer
const A64_X30: u5 = 30; // link register
const A64_SP: u5  = 31;
const A64_XZR: u5 = 31; // zero register (same encoding as SP in non-SP context)

fn a64Encode(op: u32) [4]u8 {
    return .{
        @truncate(op),
        @truncate(op >> 8),
        @truncate(op >> 16),
        @truncate(op >> 24),
    };
}

/// STP (store pair, pre-index).  offset is in units of 8 bytes.
fn a64STPPre(rt1: u5, rt2: u5, rn: u5, offset7: i7) u32 {
    const imm: u32 = @bitCast(@as(i32, offset7) & 0x7F);
    return 0xA9800000 | (imm << 15) | (@as(u32, rt2) << 10) | (@as(u32, rn) << 5) | rt1;
}

/// LDP (load pair, post-index).  offset is in units of 8 bytes.
fn a64LDPPost(rt1: u5, rt2: u5, rn: u5, offset7: i7) u32 {
    const imm: u32 = @bitCast(@as(i32, offset7) & 0x7F);
    return 0xA8C00000 | (imm << 15) | (@as(u32, rt2) << 10) | (@as(u32, rn) << 5) | rt1;
}

/// LDR (64-bit, unsigned offset).  offset is in bytes (must be aligned to 8).
fn a64LDRUnsigned(rt: u5, rn: u5, byte_offset: u16) u32 {
    const pimm = byte_offset / 8;
    return 0xF9400000 | (@as(u32, pimm) << 10) | (@as(u32, rn) << 5) | rt;
}

/// STR (64-bit, unsigned offset).
fn a64STRUnsigned(rt: u5, rn: u5, byte_offset: u16) u32 {
    const pimm = byte_offset / 8;
    return 0xF9000000 | (@as(u32, pimm) << 10) | (@as(u32, rn) << 5) | rt;
}

/// ADD (shifted register, 64-bit).
fn a64AddReg(rd: u5, rn: u5, rm: u5) u32 {
    return 0x8B000000 | (@as(u32, rm) << 16) | (@as(u32, rn) << 5) | rd;
}

/// SUB (shifted register, 64-bit).
fn a64SubReg(rd: u5, rn: u5, rm: u5) u32 {
    return 0xCB000000 | (@as(u32, rm) << 16) | (@as(u32, rn) << 5) | rd;
}

/// MUL = MADD with XZR.
fn a64Mul(rd: u5, rn: u5, rm: u5) u32 {
    return 0x9B007C00 | (@as(u32, rm) << 16) | (@as(u32, rn) << 5) | rd;
}

/// SDIV (signed divide).
fn a64SDiv(rd: u5, rn: u5, rm: u5) u32 {
    return 0x9AC00C00 | (@as(u32, rm) << 16) | (@as(u32, rn) << 5) | rd;
}

/// MSUB: rd = rn - rm*ra  → used for modulo: rd = dividend - (quot * divisor).
fn a64MSub(rd: u5, rn: u5, rm: u5, ra: u5) u32 {
    return 0x9B008000 | (@as(u32, rm) << 16) | (@as(u32, ra) << 10) | (@as(u32, rn) << 5) | rd;
}

/// NEG (= SUB from XZR).
fn a64Neg(rd: u5, rn: u5) u32 { return a64SubReg(rd, A64_XZR, rn); }

/// MOV (register, 64-bit).
fn a64Mov(rd: u5, rn: u5) u32 {
    return 0xAA0003E0 | (@as(u32, rn) << 16) | rd;
}

/// MOVZ: rd = imm16 << shift.
fn a64Movz(rd: u5, imm16: u16, shift: u2) u32 {
    return 0xD2800000 | (@as(u32, shift) << 21) | (@as(u32, imm16) << 5) | rd;
}

/// MOVK: rd[shift*16 +: 16] = imm16.
fn a64Movk(rd: u5, imm16: u16, shift: u2) u32 {
    return 0xF2800000 | (@as(u32, shift) << 21) | (@as(u32, imm16) << 5) | rd;
}

/// CMP (register, 64-bit).  Sets NZCV flags.
fn a64Cmp(rn: u5, rm: u5) u32 {
    return 0xEB000000 | (@as(u32, rm) << 16) | (@as(u32, rn) << 5) | 0x1F;
}

/// CSET: rd = (cond) ? 1 : 0.
fn a64Cset(rd: u5, cond: u4) u32 {
    return 0x9A9F07E0 | (@as(u32, cond) << 12) | rd;
}

/// B.cond (branch conditional, pc-relative imm19 in INSTRUCTIONS not bytes).
fn a64Bcond(cond: u4, imm19: i19) u32 {
    const imm: u32 = @bitCast(@as(i32, imm19) & 0x7_FFFF);
    return 0x54000000 | (imm << 5) | cond;
}

/// B (unconditional, pc-relative imm26).
fn a64B(imm26: i26) u32 {
    const imm: u32 = @bitCast(@as(i32, imm26) & 0x3FF_FFFF);
    return 0x14000000 | imm;
}

/// RET (branches to x30).
fn a64Ret() u32 { return 0xD65F03C0; }

/// Emit a 64-bit immediate into register rd using MOVZ + up to 3× MOVK.
fn a64EmitImm64(cb: *CodeBuffer, rd: u5, value: i64) !void {
    const v: u64 = @bitCast(value);
    const lo: u16 = @truncate(v);
    try cb.emitSlice(&a64Encode(a64Movz(rd, lo, 0)));
    var shift: u2 = 1;
    while (shift < 4) : (shift += 1) {
        const part: u16 = @truncate(v >> (@as(u6, shift) * 16));
        if (part != 0)
            try cb.emitSlice(&a64Encode(a64Movk(rd, part, shift)));
    }
}

fn compileAarch64(alloc: std.mem.Allocator, code: []const u8) !CompiledFunction {
    var cb = try CodeBuffer.init(alloc);
    errdefer cb.deinit();

    var patches: std.ArrayList(Patch) = .empty;
    defer patches.deinit(alloc);

    var bc_to_code = std.AutoHashMap(usize, usize).init(alloc);
    defer bc_to_code.deinit();

    // ── Prologue ──────────────────────────────────────────────────────────────
    // stp x29, x30, [sp, #-32]!
    // stp x19, x0,  [sp, #16]    (save callee-saved x19 and incoming locals ptr)
    // mov x29, sp
    // mov x19, x0   (x0 = locals ptr from C ABI)
    try cb.emitSlice(&a64Encode(a64STPPre(A64_X29, A64_X30, A64_SP, -4))); // #-32
    try cb.emitSlice(&a64Encode(a64STRUnsigned(A64_X19, A64_SP, 16)));
    // MOV x29, SP
    try cb.emitSlice(&a64Encode(0x910003FD)); // ADD x29, sp, #0
    // MOV x19, x0
    try cb.emitSlice(&a64Encode(a64Mov(A64_X19, A64_X0)));

    var pc: usize = 0;
    while (pc < code.len) {
        try bc_to_code.put(pc, cb.currentOffset());
        const op: Op = @enumFromInt(code[pc]);
        pc += 1;

        switch (op) {
            .Nop => {},

            .LoadConst => {
                if (pc + 8 > code.len) return error.UnexpectedEOF;
                const v = std.mem.readInt(i64, code[pc..][0..8], .little);
                pc += 8;
                try a64EmitImm64(&cb, A64_X0, v);
            },

            .LoadLocal => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const idx: u8 = code[pc]; pc += 1;
                // ldr x0, [x19, idx*8]
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X0, A64_X19, @intCast(idx * 8))));
            },

            .StoreLocal => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const idx: u8 = code[pc]; pc += 1;
                // str x0, [x19, idx*8]
                try cb.emitSlice(&a64Encode(a64STRUnsigned(A64_X0, A64_X19, @intCast(idx * 8))));
            },

            .Add, .Sub, .Mul, .Div, .Mod => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                const b: u8 = code[pc]; pc += 1;
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X0, A64_X19, @intCast(a * 8))));
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X1, A64_X19, @intCast(b * 8))));
                switch (op) {
                    .Add => try cb.emitSlice(&a64Encode(a64AddReg(A64_X0, A64_X0, A64_X1))),
                    .Sub => try cb.emitSlice(&a64Encode(a64SubReg(A64_X0, A64_X0, A64_X1))),
                    .Mul => try cb.emitSlice(&a64Encode(a64Mul(A64_X0, A64_X0, A64_X1))),
                    .Div => try cb.emitSlice(&a64Encode(a64SDiv(A64_X0, A64_X0, A64_X1))),
                    .Mod => {
                        // x2 = sdiv(x0, x1); x0 = msub(x2, x1, x0) = x0 - x2*x1
                        try cb.emitSlice(&a64Encode(a64SDiv(A64_X2, A64_X0, A64_X1)));
                        try cb.emitSlice(&a64Encode(a64MSub(A64_X0, A64_X2, A64_X1, A64_X0)));
                    },
                    else => unreachable,
                }
            },

            .Neg => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X0, A64_X19, @intCast(a * 8))));
                try cb.emitSlice(&a64Encode(a64Neg(A64_X0, A64_X0)));
            },

            .Eq, .Ne, .Lt, .Le, .Gt, .Ge => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                const b: u8 = code[pc]; pc += 1;
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X0, A64_X19, @intCast(a * 8))));
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X1, A64_X19, @intCast(b * 8))));
                try cb.emitSlice(&a64Encode(a64Cmp(A64_X0, A64_X1)));
                // AArch64 condition codes: EQ=0, NE=1, LT=11, LE=13, GT=12, GE=10
                const cond: u4 = switch (op) {
                    .Eq => 0, .Ne => 1, .Lt => 11, .Le => 13, .Gt => 12, .Ge => 10,
                    else => unreachable,
                };
                try cb.emitSlice(&a64Encode(a64Cset(A64_X0, cond)));
            },

            .JumpIf, .JumpIfFalse, .Jump => {
                if (op == .Jump) {
                    if (pc + 2 > code.len) return error.UnexpectedEOF;
                    const tgt = std.mem.readInt(u16, code[pc..][0..2], .little);
                    pc += 2;
                    const patch_off = cb.currentOffset();
                    try cb.emitSlice(&a64Encode(a64B(0))); // placeholder
                    try patches.append(alloc, .{ .code_offset = patch_off, .bc_target = tgt, .kind = .imm26_a64 });
                } else {
                    if (pc + 3 > code.len) return error.UnexpectedEOF;
                    const cond_idx: u8 = code[pc]; pc += 1;
                    const tgt = std.mem.readInt(u16, code[pc..][0..2], .little);
                    pc += 2;
                    // ldr x0, [x19, cond*8]; cbnz/cbz x0, target
                    try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X0, A64_X19, @intCast(cond_idx * 8))));
                    // CBNZ x0 / CBZ x0 + imm19 placeholder
                    const base: u32 = if (op == .JumpIf) 0xB5000000 else 0xB4000000;
                    const patch_off = cb.currentOffset();
                    try cb.emitSlice(&a64Encode(base | A64_X0)); // imm19=0
                    try patches.append(alloc, .{ .code_offset = patch_off, .bc_target = tgt, .kind = .imm19_a64 });
                }
            },

            .Ret => {
                // Epilogue: restore, ret
                try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X19, A64_SP, 16)));
                try cb.emitSlice(&a64Encode(a64LDPPost(A64_X29, A64_X30, A64_SP, 4))); // #32
                try cb.emitSlice(&a64Encode(a64Ret()));
            },

            else => return error.UnknownOpcode,
        }
    }

    // Implicit epilogue.
    try cb.emitSlice(&a64Encode(a64LDRUnsigned(A64_X19, A64_SP, 16)));
    try cb.emitSlice(&a64Encode(a64LDPPost(A64_X29, A64_X30, A64_SP, 4)));
    try cb.emitSlice(&a64Encode(a64Ret()));

    // ── Patch branches ────────────────────────────────────────────────────────
    for (patches.items) |patch| {
        const dst = bc_to_code.get(patch.bc_target) orelse return error.BadJumpTarget;
        const delta_bytes = @as(i64, @intCast(dst)) - @as(i64, @intCast(patch.code_offset));
        const delta_insn  = @divExact(delta_bytes, 4);

        var insn = std.mem.readInt(u32, cb.mem[patch.code_offset..][0..4], .little);
        switch (patch.kind) {
            .imm26_a64 => {
                const imm26: i26 = @intCast(delta_insn);
                insn = (insn & 0xFC00_0000) | (@as(u32, @bitCast(@as(i32, imm26))) & 0x03FF_FFFF);
            },
            .imm19_a64 => {
                const imm19: i19 = @intCast(delta_insn);
                insn = (insn & 0xFF00_001F) | ((@as(u32, @bitCast(@as(i32, imm19))) & 0x7FFFF) << 5);
            },
            else => {},
        }
        cb.patchU32LE(patch.code_offset, insn);
    }

    try cb.seal();
    return CompiledFunction{ .buf = cb, .entry_point = 0 };
}

// ═══════════════════════════════════════════════════════════════════════════════
// RISC-V 64 Backend  (RV64IMAC)
// ═══════════════════════════════════════════════════════════════════════════════
//
// Register assignments:
//   s1  (x9)  = locals pointer (callee-saved)
//   t0  (x5)  = accumulator / lhs
//   t1  (x6)  = rhs scratch
//   t2  (x7)  = PC-grab scratch (used only in emitLI64)
//   a0  (x10) = return value (populated from t0 in epilogue)
//   s0  (x8)  = frame pointer (callee-saved)
//   ra  (x1)  = link register (callee-saved in the sense that we save/restore)
//   sp  (x2)  = stack pointer
//
// Instruction encoding helpers use the standard RV32I / RV64I formats.

const RV_X0: u5  = 0;  // zero
const RV_RA: u5  = 1;  // return address
const RV_SP: u5  = 2;  // stack pointer
const RV_S0: u5  = 8;  // frame pointer (callee-saved)
const RV_S1: u5  = 9;  // locals ptr   (callee-saved)
const RV_A0: u5  = 10; // arg0 / return
const RV_A1: u5  = 11; // arg1 (globals ptr, not used by baseline JIT)
const RV_T0: u5  = 5;  // accumulator / lhs
const RV_T1: u5  = 6;  // rhs scratch
const RV_T2: u5  = 7;  // scratch for emitLI64 PC-grab

// ── Encoding helpers ──────────────────────────────────────────────────────────

fn rvRType(funct7: u7, rs2: u5, rs1: u5, funct3: u3, rd: u5, opcode: u7) u32 {
    return (@as(u32, funct7) << 25) | (@as(u32, rs2) << 20) |
           (@as(u32, rs1) << 15) | (@as(u32, funct3) << 12) |
           (@as(u32, rd) << 7)   | @as(u32, opcode);
}

fn rvIType(imm12: i12, rs1: u5, funct3: u3, rd: u5, opcode: u7) u32 {
    const imm: u32 = @as(u32, @bitCast(@as(i32, imm12))) & 0xFFF;
    return (imm << 20) | (@as(u32, rs1) << 15) | (@as(u32, funct3) << 12) |
           (@as(u32, rd) << 7) | @as(u32, opcode);
}

fn rvSType(imm12: i12, rs2: u5, rs1: u5, funct3: u3, opcode: u7) u32 {
    const imm: u32 = @as(u32, @bitCast(@as(i32, imm12))) & 0xFFF;
    return ((imm >> 5) << 25) | (@as(u32, rs2) << 20) | (@as(u32, rs1) << 15) |
           (@as(u32, funct3) << 12) | ((imm & 0x1F) << 7) | @as(u32, opcode);
}

fn rvBType(imm13: i13, rs2: u5, rs1: u5, funct3: u3, opcode: u7) u32 {
    const imm: u32 = @as(u32, @bitCast(@as(i32, imm13)));
    const b12     = (imm >> 12) & 1;
    const b11     = (imm >> 11) & 1;
    const b10_5   = (imm >> 5)  & 0x3F;
    const b4_1    = (imm >> 1)  & 0xF;
    return (b12 << 31) | (b10_5 << 25) | (@as(u32, rs2) << 20) |
           (@as(u32, rs1) << 15) | (@as(u32, funct3) << 12) |
           (b4_1 << 8) | (b11 << 7) | @as(u32, opcode);
}

fn rvJType(imm21: i21, rd: u5, opcode: u7) u32 {
    const imm: u32 = @as(u32, @bitCast(@as(i32, imm21)));
    const b20     = (imm >> 20) & 1;
    const b10_1   = (imm >> 1)  & 0x3FF;
    const b11     = (imm >> 11) & 1;
    const b19_12  = (imm >> 12) & 0xFF;
    return (b20 << 31) | (b10_1 << 21) | (b11 << 20) | (b19_12 << 12) |
           (@as(u32, rd) << 7) | @as(u32, opcode);
}

fn rvUType(imm20: u20, rd: u5, opcode: u7) u32 {
    return (@as(u32, imm20) << 12) | (@as(u32, rd) << 7) | @as(u32, opcode);
}

// ── Common instructions ───────────────────────────────────────────────────────

/// ADDI rd, rs1, imm12
fn rvADDI(rd: u5, rs1: u5, imm: i12) u32  { return rvIType(imm, rs1, 0, rd, 0x13); }
/// ADD  rd, rs1, rs2
fn rvADD(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(0, rs2, rs1, 0, rd, 0x33); }
/// SUB  rd, rs1, rs2
fn rvSUB(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(0x20, rs2, rs1, 0, rd, 0x33); }
/// MUL  rd, rs1, rs2    (M extension)
fn rvMUL(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(1, rs2, rs1, 0, rd, 0x33); }
/// DIV  rd, rs1, rs2    (M extension, signed)
fn rvDIV(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(1, rs2, rs1, 4, rd, 0x33); }
/// REM  rd, rs1, rs2    (M extension, signed)
fn rvREM(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(1, rs2, rs1, 6, rd, 0x33); }
/// SLT  rd, rs1, rs2    (set if less-than, signed)
fn rvSLT(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(0, rs2, rs1, 2, rd, 0x33); }
/// SLTU rd, rs1, rs2    (set if less-than, unsigned)
fn rvSLTU(rd: u5, rs1: u5, rs2: u5) u32   { return rvRType(0, rs2, rs1, 3, rd, 0x33); }
/// XOR  rd, rs1, rs2
fn rvXOR(rd: u5, rs1: u5, rs2: u5) u32    { return rvRType(0, rs2, rs1, 4, rd, 0x33); }
/// XORI rd, rs1, imm12
fn rvXORI(rd: u5, rs1: u5, imm: i12) u32  { return rvIType(imm, rs1, 4, rd, 0x13); }
/// SLTIU rd, rs1, imm12   (SEQZ when imm=1)
fn rvSLTIU(rd: u5, rs1: u5, imm: i12) u32 { return rvIType(imm, rs1, 3, rd, 0x13); }
/// LD   rd, offset(rs1)
fn rvLD(rd: u5, rs1: u5, off: i12) u32    { return rvIType(off, rs1, 3, rd, 0x03); }
/// SD   rs2, offset(rs1)
fn rvSD(rs2: u5, rs1: u5, off: i12) u32   { return rvSType(off, rs2, rs1, 3, 0x23); }
/// NEG  rd, rs1  (pseudo: SUB rd, x0, rs1)
fn rvNEG(rd: u5, rs: u5) u32              { return rvSUB(rd, RV_X0, rs); }
/// MV   rd, rs1  (pseudo: ADDI rd, rs1, 0)
fn rvMV(rd: u5, rs: u5) u32               { return rvADDI(rd, rs, 0); }
/// RET          (pseudo: JALR x0, ra, 0)
fn rvRET() u32                             { return rvIType(0, RV_RA, 0, RV_X0, 0x67); }
/// BEQ  rs1, rs2, offset13
fn rvBEQ(rs1: u5, rs2: u5, off: i13) u32  { return rvBType(off, rs2, rs1, 0, 0x63); }
/// BNE  rs1, rs2, offset13
fn rvBNE(rs1: u5, rs2: u5, off: i13) u32  { return rvBType(off, rs2, rs1, 1, 0x63); }
/// BLT  rs1, rs2, offset13  (signed)
fn rvBLT(rs1: u5, rs2: u5, off: i13) u32  { return rvBType(off, rs2, rs1, 4, 0x63); }
/// BGE  rs1, rs2, offset13  (signed)
fn rvBGE(rs1: u5, rs2: u5, off: i13) u32  { return rvBType(off, rs2, rs1, 5, 0x63); }
/// JAL  rd, offset21
fn rvJAL(rd: u5, off: i21) u32            { return rvJType(off, rd, 0x6F); }

/// Load a 64-bit immediate into `rd` using the PC-grab trick.
/// Clobbers T2 (x7) temporarily.
/// Layout (16 bytes):
///   +0:  JAL  T2, 12    → T2 = addr of literal; jump to +12
///   +4:  <8-byte literal>
///   +12: LD   rd, 0(T2)
fn rvEmitLI64(cb: *CodeBuffer, rd: u5, value: i64) !void {
    // Try small immediate first (addi rd, x0, imm12).
    if (value >= -2048 and value <= 2047) {
        try cb.emitU32LE(rvADDI(rd, RV_X0, @intCast(value)));
        return;
    }
    // Try 32-bit (lui + addi): fits if upper bits are sign-extend of lower 12.
    const lo12: i64 = @as(i64, @truncate(@as(i32, @truncate(value))));
    const hi20: i64 = value - lo12;
    if (hi20 == (hi20 >> 12) << 12 and hi20 >> 31 == 0) {
        const upper: u20 = @truncate(@as(u64, @bitCast(hi20)) >> 12);
        try cb.emitU32LE(rvUType(upper, rd, 0x37)); // LUI rd, upper20
        const adj_lo: i12 = @intCast(lo12);
        if (adj_lo != 0) try cb.emitU32LE(rvADDI(rd, rd, adj_lo));
        return;
    }
    // Full 64-bit literal via JAL + inline data + LD.
    try cb.emitU32LE(rvJAL(RV_T2, 12));    // JAL t2, 12  (skip 8-byte literal)
    try cb.emitU64LE(@bitCast(value));      // .dword value
    try cb.emitU32LE(rvLD(rd, RV_T2, 0)); // LD rd, 0(t2)
}

// ── riscv64 local load/store ──────────────────────────────────────────────────

fn rvLoadLocal(cb: *CodeBuffer, rd: u5, idx: u8) !void {
    // ld rd, (idx*8)(s1)
    // If offset > 2047 we'd need a different approach, but in practice idx < 32.
    const off: i12 = @intCast(@as(i32, idx) * 8);
    try cb.emitU32LE(rvLD(rd, RV_S1, off));
}

fn rvStoreLocal(cb: *CodeBuffer, rs: u5, idx: u8) !void {
    const off: i12 = @intCast(@as(i32, idx) * 8);
    try cb.emitU32LE(rvSD(rs, RV_S1, off));
}

// ── riscv64 prologue / epilogue ───────────────────────────────────────────────

fn rvEmitPrologue(cb: *CodeBuffer) !void {
    // addi sp, sp, -32
    try cb.emitU32LE(rvADDI(RV_SP, RV_SP, -32));
    // sd ra, 24(sp)
    try cb.emitU32LE(rvSD(RV_RA, RV_SP, 24));
    // sd s0, 16(sp)
    try cb.emitU32LE(rvSD(RV_S0, RV_SP, 16));
    // sd s1, 8(sp)
    try cb.emitU32LE(rvSD(RV_S1, RV_SP, 8));
    // addi s0, sp, 32   (frame pointer)
    try cb.emitU32LE(rvADDI(RV_S0, RV_SP, 32));
    // mv s1, a0         (save locals ptr)
    try cb.emitU32LE(rvMV(RV_S1, RV_A0));
}

fn rvEmitEpilogue(cb: *CodeBuffer) !void {
    // mv a0, t0         (return accumulator)
    try cb.emitU32LE(rvMV(RV_A0, RV_T0));
    // ld s1, 8(sp)
    try cb.emitU32LE(rvLD(RV_S1, RV_SP, 8));
    // ld s0, 16(sp)
    try cb.emitU32LE(rvLD(RV_S0, RV_SP, 16));
    // ld ra, 24(sp)
    try cb.emitU32LE(rvLD(RV_RA, RV_SP, 24));
    // addi sp, sp, 32
    try cb.emitU32LE(rvADDI(RV_SP, RV_SP, 32));
    // ret
    try cb.emitU32LE(rvRET());
}

// ── riscv64 compiler ─────────────────────────────────────────────────────────

fn compileRiscv64(alloc: std.mem.Allocator, code: []const u8) !CompiledFunction {
    var cb = try CodeBuffer.init(alloc);
    errdefer cb.deinit();

    var patches: std.ArrayList(Patch) = .empty;
    defer patches.deinit(alloc);

    var bc_to_code = std.AutoHashMap(usize, usize).init(alloc);
    defer bc_to_code.deinit();

    try rvEmitPrologue(&cb);

    var pc: usize = 0;
    while (pc < code.len) {
        try bc_to_code.put(pc, cb.currentOffset());
        const op: Op = @enumFromInt(code[pc]);
        pc += 1;

        switch (op) {
            .Nop => {},

            .LoadConst => {
                if (pc + 8 > code.len) return error.UnexpectedEOF;
                const v = std.mem.readInt(i64, code[pc..][0..8], .little);
                pc += 8;
                try rvEmitLI64(&cb, RV_T0, v);
            },

            .LoadLocal => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const idx: u8 = code[pc]; pc += 1;
                try rvLoadLocal(&cb, RV_T0, idx);
            },

            .StoreLocal => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const idx: u8 = code[pc]; pc += 1;
                try rvStoreLocal(&cb, RV_T0, idx);
            },

            .Add, .Sub, .Mul, .Div, .Mod => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                const b: u8 = code[pc]; pc += 1;
                try rvLoadLocal(&cb, RV_T0, a); // t0 = locals[a]
                try rvLoadLocal(&cb, RV_T1, b); // t1 = locals[b]
                switch (op) {
                    .Add => try cb.emitU32LE(rvADD(RV_T0, RV_T0, RV_T1)),
                    .Sub => try cb.emitU32LE(rvSUB(RV_T0, RV_T0, RV_T1)),
                    .Mul => try cb.emitU32LE(rvMUL(RV_T0, RV_T0, RV_T1)),
                    .Div => try cb.emitU32LE(rvDIV(RV_T0, RV_T0, RV_T1)),
                    .Mod => try cb.emitU32LE(rvREM(RV_T0, RV_T0, RV_T1)),
                    else => unreachable,
                }
            },

            .Neg => {
                if (pc + 1 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                try rvLoadLocal(&cb, RV_T0, a);
                try cb.emitU32LE(rvNEG(RV_T0, RV_T0));
            },

            // Comparison: result → t0 (0 or 1)
            .Eq, .Ne, .Lt, .Le, .Gt, .Ge => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const a: u8 = code[pc]; pc += 1;
                const b: u8 = code[pc]; pc += 1;
                try rvLoadLocal(&cb, RV_T0, a); // t0 = lhs
                try rvLoadLocal(&cb, RV_T1, b); // t1 = rhs
                switch (op) {
                    .Eq => {
                        // xor t0, t0, t1; sltiu t0, t0, 1  (seqz)
                        try cb.emitU32LE(rvXOR(RV_T0, RV_T0, RV_T1));
                        try cb.emitU32LE(rvSLTIU(RV_T0, RV_T0, 1));
                    },
                    .Ne => {
                        // xor t0, t0, t1; sltu t0, x0, t0  (snez)
                        try cb.emitU32LE(rvXOR(RV_T0, RV_T0, RV_T1));
                        try cb.emitU32LE(rvSLTU(RV_T0, RV_X0, RV_T0));
                    },
                    .Lt => try cb.emitU32LE(rvSLT(RV_T0, RV_T0, RV_T1)),
                    .Ge => {
                        // t0 = (t1 < t0) ? 1 : 0; then xori t0, t0, 1
                        try cb.emitU32LE(rvSLT(RV_T0, RV_T0, RV_T1)); // t0 = (lhs < rhs)
                        try cb.emitU32LE(rvXORI(RV_T0, RV_T0, 1));     // flip bit
                    },
                    .Gt => try cb.emitU32LE(rvSLT(RV_T0, RV_T1, RV_T0)), // rhs < lhs
                    .Le => {
                        try cb.emitU32LE(rvSLT(RV_T0, RV_T1, RV_T0)); // t0 = (rhs < lhs)
                        try cb.emitU32LE(rvXORI(RV_T0, RV_T0, 1));     // flip
                    },
                    else => unreachable,
                }
            },

            // Branches
            .Jump => {
                if (pc + 2 > code.len) return error.UnexpectedEOF;
                const tgt = std.mem.readInt(u16, code[pc..][0..2], .little);
                pc += 2;
                const patch_off = cb.currentOffset();
                try cb.emitU32LE(rvJAL(RV_X0, 0)); // JAL x0, 0 (placeholder)
                try patches.append(alloc, .{ .code_offset = patch_off, .bc_target = tgt, .kind = .imm21_rv64 });
            },

            .JumpIf, .JumpIfFalse => {
                if (pc + 3 > code.len) return error.UnexpectedEOF;
                const cond_idx: u8 = code[pc]; pc += 1;
                const tgt = std.mem.readInt(u16, code[pc..][0..2], .little);
                pc += 2;
                // ld t0, cond*8(s1)
                try rvLoadLocal(&cb, RV_T0, cond_idx);
                // BNE/BEQ t0, x0, target
                const patch_off = cb.currentOffset();
                if (op == .JumpIf) {
                    try cb.emitU32LE(rvBNE(RV_T0, RV_X0, 0)); // placeholder
                } else {
                    try cb.emitU32LE(rvBEQ(RV_T0, RV_X0, 0)); // placeholder
                }
                try patches.append(alloc, .{ .code_offset = patch_off, .bc_target = tgt, .kind = .imm13_rv64 });
            },

            .Ret => try rvEmitEpilogue(&cb),

            else => return error.UnknownOpcode,
        }
    }

    // Implicit epilogue at end of bytecode.
    try rvEmitEpilogue(&cb);

    // ── Patch forward branches ────────────────────────────────────────────────
    for (patches.items) |patch| {
        const dst = bc_to_code.get(patch.bc_target) orelse return error.BadJumpTarget;
        const delta = @as(i64, @intCast(dst)) - @as(i64, @intCast(patch.code_offset));
        var insn = std.mem.readInt(u32, cb.mem[patch.code_offset..][0..4], .little);

        switch (patch.kind) {
            .imm21_rv64 => {
                const off: i21 = @intCast(delta);
                insn = rvJAL(RV_X0, off);
            },
            .imm13_rv64 => {
                // Reconstruct the B-type instruction with the correct offset.
                const off: i13 = @intCast(delta);
                // Preserve rs1/rs2/funct3 from placeholder, replace immediate.
                const rs1: u5  = @truncate((insn >> 15) & 0x1F);
                const rs2: u5  = @truncate((insn >> 20) & 0x1F);
                const f3:  u3  = @truncate((insn >> 12) & 0x7);
                insn = rvBType(off, rs2, rs1, f3, 0x63);
            },
            else => {},
        }
        cb.patchU32LE(patch.code_offset, insn);
    }

    try cb.seal();
    return CompiledFunction{ .buf = cb, .entry_point = 0 };
}

// ═══════════════════════════════════════════════════════════════════════════════
// Self-benchmark (reached via: lunex-rt jit-bench)
// ═══════════════════════════════════════════════════════════════════════════════

/// A simple fibonacci loop encoded as NTZ bytecode for JIT self-testing.
/// locals[0] = n, locals[1] = a, locals[2] = b, locals[3] = tmp, locals[4] = cond
///
/// Lunex pseudo-code:
///   n = 30
///   a = 0; b = 1
///   while n > 0:
///     tmp  = a + b
///     a    = b
///     b    = tmp
///     n   -= 1
///   return a
const FIB_BYTECODE = blk: {
    // Index constants for locals:
    //   0 = n, 1 = a, 2 = b, 3 = tmp, 4 = cond
    var bc: [128]u8 = undefined;
    var i: usize = 0;
    // Macro helpers (comptime)
    const S = struct {
        fn lc(b: *[128]u8, off: *usize, v: i64) void {
            b[off.*] = @intFromEnum(Op.LoadConst); off.* += 1;
            const bytes = @as([8]u8, @bitCast(v));
            var j: usize = 0; while (j < 8) : (j += 1) { b[off.* + j] = bytes[j]; } off.* += 8;
        }
        fn sl(b: *[128]u8, off: *usize, idx: u8) void {
            b[off.*] = @intFromEnum(Op.StoreLocal); off.* += 1; b[off.*] = idx; off.* += 1;
        }
        fn ll(b: *[128]u8, off: *usize, idx: u8) void {
            b[off.*] = @intFromEnum(Op.LoadLocal); off.* += 1; b[off.*] = idx; off.* += 1;
        }
        fn add(b: *[128]u8, off: *usize, a: u8, bb: u8) void {
            b[off.*] = @intFromEnum(Op.Add); off.* += 1; b[off.*] = a; off.* += 1; b[off.*] = bb; off.* += 1;
        }
        fn sub(b: *[128]u8, off: *usize, a: u8, bb: u8) void {
            b[off.*] = @intFromEnum(Op.Sub); off.* += 1; b[off.*] = a; off.* += 1; b[off.*] = bb; off.* += 1;
        }
        fn gt(b: *[128]u8, off: *usize, a: u8, bb: u8) void {
            b[off.*] = @intFromEnum(Op.Gt); off.* += 1; b[off.*] = a; off.* += 1; b[off.*] = bb; off.* += 1;
        }
        fn jf(b: *[128]u8, off: *usize, cond: u8, tgt_lo: u8, tgt_hi: u8) void {
            b[off.*] = @intFromEnum(Op.JumpIfFalse); off.* += 1;
            b[off.*] = cond; off.* += 1;
            b[off.*] = tgt_lo; off.* += 1; b[off.*] = tgt_hi; off.* += 1;
        }
        fn jmp(b: *[128]u8, off: *usize, tgt_lo: u8, tgt_hi: u8) void {
            b[off.*] = @intFromEnum(Op.Jump); off.* += 1;
            b[off.*] = tgt_lo; off.* += 1; b[off.*] = tgt_hi; off.* += 1;
        }
        fn ret(b: *[128]u8, off: *usize) void {
            b[off.*] = @intFromEnum(Op.Ret); off.* += 1;
        }
    };
    //  0: LoadConst 30 → StoreLocal 0  (n = 30)
    S.lc(&bc, &i, 30); S.sl(&bc, &i, 0);
    //  11: LoadConst 0 → StoreLocal 1  (a = 0)
    S.lc(&bc, &i, 0); S.sl(&bc, &i, 1);
    //  22: LoadConst 1 → StoreLocal 2  (b = 1)
    S.lc(&bc, &i, 1); S.sl(&bc, &i, 2);
    //  33: LoadConst 0 → StoreLocal 4  (cond = 0, placeholder)
    S.lc(&bc, &i, 0); S.sl(&bc, &i, 4);
    // loop_head = i (37):
    //  37: Gt 0, 4_zero → cond (n > 0)
    // Emit: lc 0→sl4; Gt 0, 4→sl4; JumpIfFalse 4, exit_lo, exit_hi
    // Simplification: reuse local 4 for zero comparison
    S.lc(&bc, &i, 0); S.sl(&bc, &i, 4);   // local4 = 0
    S.gt(&bc, &i, 0, 4);   // acc = n > 0
    S.sl(&bc, &i, 4);       // local4 = result
    // JumpIfFalse local4, exit (we'll patch exit below)
    const jf_off = i;
    S.jf(&bc, &i, 4, 0, 0); // placeholder target
    // tmp = a + b
    S.add(&bc, &i, 1, 2); S.sl(&bc, &i, 3);
    // a = b
    S.ll(&bc, &i, 2); S.sl(&bc, &i, 1);
    // b = tmp
    S.ll(&bc, &i, 3); S.sl(&bc, &i, 2);
    // n -= 1  (n = n - local4_one; but local4 holds 0 now... use lc 1)
    S.lc(&bc, &i, 1); S.sl(&bc, &i, 3);   // local3 = 1
    S.sub(&bc, &i, 0, 3); S.sl(&bc, &i, 0); // n = n - 1
    // JMP loop_head (offset of Gt instruction above)
    const loop_head_off = jf_off - 14; // offset of lc 0 before Gt
    S.jmp(&bc, &i, @truncate(loop_head_off), @truncate(loop_head_off >> 8));
    // exit:
    const exit_off = i;
    // Patch JumpIfFalse target.
    bc[jf_off + 2] = @truncate(exit_off);
    bc[jf_off + 3] = @truncate(exit_off >> 8);
    // return a (local 1)
    S.ll(&bc, &i, 1);
    S.sl(&bc, &i, 0);
    S.ret(&bc, &i);
    const len = i;
    break :blk bc[0..len].*;
};

// ─── JITCompiler (VM-facing facade) ───────────────────────────────────────────
//
// This struct is the single JIT object held by the VM.  It combines the
// one-shot compiler (JitCompiler) with the function cache (JitCache) so the
// VM only needs one field to manage the entire JIT subsystem.
//
// API used by vm.zig
// ------------------
//   init(alloc)                      — allocate
//   deinit(self)                     — free all cached native code
//   compileHot(func_id, code) bool   — compile + cache a hot function (idempotent)
//   callCached(func_id, l, g) ?i64   — execute cached native code, or null
//   totalUnits() u32                 — number of functions compiled so far
//   printStats(self)                 — write a one-line summary to stderr

pub const JITCompiler = struct {
    compiler:    JitCompiler,
    cache:       JitCache,
    n_compiled:  u32,

    pub fn init(alloc: std.mem.Allocator) JITCompiler {
        return JITCompiler{
            .compiler   = JitCompiler.init(alloc),
            .cache      = JitCache.init(alloc),
            .n_compiled = 0,
        };
    }

    pub fn deinit(self: *JITCompiler) void {
        self.cache.deinit();
    }

    /// JIT-compile `code` for `func_id` if not already in the cache.
    /// Returns true when a compiled version is available (cached or just built).
    /// Transparent fall-through on unsupported architectures.
    pub fn compileHot(self: *JITCompiler, func_id: u64, code: []const u8) bool {
        // Already cached — nothing to do.
        if (self.cache.get(func_id) != null) return true;
        // Try to compile; silently swallow unsupported-arch errors so the
        // interpreter continues running on non-JIT platforms.
        const cf = self.compiler.compile(code) catch return false;
        self.cache.put(func_id, cf) catch {
            var cf_mut = cf;
            cf_mut.deinit();
            return false;
        };
        self.n_compiled += 1;
        return true;
    }

    /// Execute the cached native function for `func_id` with the given locals
    /// and globals arrays.  Returns null when the function is not yet compiled.
    pub fn callCached(
        self:    *const JITCompiler,
        func_id: u64,
        locals:  [*]i64,
        globals: [*]i64,
    ) ?i64 {
        const cf = self.cache.get(func_id) orelse return null;
        return cf.call(locals, globals);
    }

    /// Returns the total number of functions compiled by this JIT instance.
    pub fn totalUnits(self: *const JITCompiler) u32 {
        return self.n_compiled;
    }

    /// Writes a one-line JIT summary to stderr.
    pub fn printStats(self: *const JITCompiler) void {
        var buf: [128]u8 = undefined;
        const msg = std.fmt.bufPrint(&buf,
            "[jit] compiled={d}  cached={d}\n",
            .{ self.n_compiled, self.cache.functions.count() },
        ) catch return;
        platform.fdWrite(std.posix.STDERR_FILENO, msg);
    }
};

pub fn benchmark(alloc: std.mem.Allocator) !void {
    const arch_name = @tagName(builtin.cpu.arch);
    const os_name   = @tagName(builtin.os.tag);

    std.debug.print(
        \\
        \\┌─────────────────────────────────────────────────────────────┐
        \\│  Lunex JIT Self-Benchmark  (arch={s:<8} os={s:<8})  │
        \\└─────────────────────────────────────────────────────────────┘
        \\
    , .{ arch_name, os_name });

    // ── Compilation benchmark ─────────────────────────────────────────────────
    const COMPILE_ITERS = 1_000;
    var compile_ns: u64 = 0;

    var last_cf: ?CompiledFunction = null;
    var iter: usize = 0;
    while (iter < COMPILE_ITERS) : (iter += 1) {
        if (last_cf) |*cf| cf.deinit();
        const t0 = platform.highResTimer();
        var jc = JitCompiler.init(alloc);
        last_cf = jc.compile(&FIB_BYTECODE) catch |e| {
            if (e == error.UnsupportedArch) {
                std.debug.print("  JIT not supported on arch={s} — interpreter fallback is used.\n", .{arch_name});
                return;
            }
            return e;
        };
        compile_ns += platform.highResTimer() - t0;
    }
    defer if (last_cf) |*cf| cf.deinit();

    const avg_compile_us = @as(f64, @floatFromInt(compile_ns)) / @as(f64, COMPILE_ITERS) / 1000.0;
    std.debug.print("  Compile:   {d:.2} µs/fn  ({d} iterations)\n", .{ avg_compile_us, COMPILE_ITERS });

    // ── Execution benchmark ───────────────────────────────────────────────────
    if (last_cf) |cf| {
        var locals: [8]i64 = undefined;
@memset(&locals, 0);
        const EXEC_ITERS = 100_000;
        const t_exec = platform.highResTimer();
        var result: i64 = 0;
        var ei: usize = 0;
        while (ei < EXEC_ITERS) : (ei += 1) {
            @memset(&locals, 0);
            result = cf.call(&locals, &locals);
        }
        const exec_ns = (platform.highResTimer() - t_exec);
        const ns_per_call = @as(f64, @floatFromInt(exec_ns)) / @as(f64, EXEC_ITERS);

        // fib(30) = 832040
        const expected: i64 = 832040;
        std.debug.print(
            "  Execute:   {d:.1} ns/call  ({d} iterations)\n" ++
            "  fib(30):   {d}  {s}\n",
            .{ ns_per_call, EXEC_ITERS, result,
               if (result == expected) "✓ correct" else "✗ WRONG — check JIT emitter" },
        );
    }

    std.debug.print("\n", .{});
}
