// Lunex lang — Platform Abstraction Layer
// Created by David Dev · GitHub: https://github.com/Megamexlevi2

const std     = @import("std");
const builtin = @import("builtin");

pub const OS = enum {
    linux,
    macos,
    windows,
    android,
    freebsd,
    openbsd,
    netbsd,
    solaris,
    unknown,
};

pub const ARCH = enum {
    x86_64,
    aarch64,
    riscv64,
    unknown,
};

pub fn currentOS() OS {
    return switch (builtin.os.tag) {
        .linux   => if (isAndroid()) OS.android else OS.linux,
        .macos   => OS.macos,
        .windows => OS.windows,
        .freebsd => OS.freebsd,
        .openbsd => OS.openbsd,
        .netbsd  => OS.netbsd,
        .illumos => OS.solaris,
        else     => OS.unknown,
    };
}

pub fn currentArch() ARCH {
    return switch (builtin.cpu.arch) {
        .x86_64               => ARCH.x86_64,
        .aarch64, .aarch64_be => ARCH.aarch64,
        .riscv64              => ARCH.riscv64,
        else                  => ARCH.unknown,
    };
}

fn isAndroid() bool {
    if (comptime builtin.os.tag != .linux) return false;
    const fd = std.os.linux.open("/proc/version", .{ .ACCMODE = .RDONLY }, 0);
    const open_errno = std.os.linux.errno(fd);
    if (open_errno != .SUCCESS) return false;
    defer _ = std.os.linux.close(@intCast(fd));
    var buf: [256]u8 = undefined;
    const rc = std.os.linux.read(@intCast(fd), &buf, buf.len);
    const read_errno = std.os.linux.errno(rc);
    if (read_errno != .SUCCESS) return false;
    return std.mem.indexOf(u8, buf[0..rc], "Android") != null;
}

// ─── Windows API declarations ─────────────────────────────────────────────────

const win = std.os.windows;

const PAGE_EXECUTE_READWRITE: win.DWORD = 0x40;
const PAGE_EXECUTE_READ:      win.DWORD = 0x20;
const MEM_COMMIT:             win.DWORD = 0x00001000;
const MEM_RESERVE:            win.DWORD = 0x00002000;
const MEM_RELEASE:            win.DWORD = 0x00008000;
const STD_OUTPUT_HANDLE:      win.DWORD = @bitCast(@as(i32, -11));
const STD_ERROR_HANDLE:       win.DWORD = @bitCast(@as(i32, -12));

extern "kernel32" fn VirtualAlloc(
    lpAddress:        ?win.LPVOID,
    dwSize:           win.SIZE_T,
    flAllocationType: win.DWORD,
    flProtect:        win.DWORD,
) callconv(.winapi) ?win.LPVOID;

extern "kernel32" fn VirtualFree(
    lpAddress:  win.LPVOID,
    dwSize:     win.SIZE_T,
    dwFreeType: win.DWORD,
) callconv(.winapi) win.BOOL;

extern "kernel32" fn VirtualProtect(
    lpAddress:      win.LPVOID,
    dwSize:         win.SIZE_T,
    flNewProtect:   win.DWORD,
    lpflOldProtect: *win.DWORD,
) callconv(.winapi) win.BOOL;

extern "kernel32" fn WriteFile(
    hFile:                  win.HANDLE,
    lpBuffer:               [*]const u8,
    nNumberOfBytesToWrite:  win.DWORD,
    lpNumberOfBytesWritten: ?*win.DWORD,
    lpOverlapped:           ?*anyopaque,
) callconv(.winapi) win.BOOL;

extern "kernel32" fn GetStdHandle(nStdHandle: win.DWORD) callconv(.winapi) win.HANDLE;

extern "kernel32" fn QueryPerformanceFrequency(lpFrequency: *i64) callconv(.winapi) win.BOOL;
extern "kernel32" fn QueryPerformanceCounter(lpPerformanceCount: *i64) callconv(.winapi) win.BOOL;

// ─── Executable memory ────────────────────────────────────────────────────────

pub const ExecMemory = struct {
    ptr: [*]u8,
    len: usize,

    pub fn alloc(size: usize) !ExecMemory {
        const aligned = std.mem.alignForward(usize, size, 4096);

        if (comptime builtin.os.tag == .windows) {
            const base = VirtualAlloc(
                null,
                aligned,
                MEM_COMMIT | MEM_RESERVE,
                PAGE_EXECUTE_READWRITE,
            ) orelse return error.OutOfMemory;
            return ExecMemory{ .ptr = @ptrCast(base), .len = aligned };
        } else {
            const prot  = std.posix.PROT{ .READ = true, .WRITE = true, .EXEC = true };
            const flags = std.posix.MAP{ .TYPE = .PRIVATE, .ANONYMOUS = true };
            const mem   = try std.posix.mmap(null, aligned, prot, flags, -1, 0);
            return ExecMemory{ .ptr = mem.ptr, .len = aligned };
        }
    }

    pub fn free(self: ExecMemory) void {
        if (comptime builtin.os.tag == .windows) {
            _ = VirtualFree(@ptrCast(self.ptr), 0, MEM_RELEASE);
        } else {
            std.posix.munmap(@alignCast(self.ptr[0..self.len]));
        }
    }

    pub fn makeExecutable(self: ExecMemory) !void {
        if (comptime builtin.os.tag == .windows) {
            var old: win.DWORD = undefined;
            if (VirtualProtect(@ptrCast(self.ptr), self.len, PAGE_EXECUTE_READ, &old) == 0)
                return error.PermissionDenied;
        } else {
            const prot = std.os.linux.PROT{ .READ = true, .EXEC = true };
            const rc = std.os.linux.mprotect(self.ptr, self.len, prot);
            if (std.os.linux.errno(rc) != .SUCCESS) return error.PermissionDenied;
            if (comptime builtin.cpu.arch == .aarch64) flushICache(self.ptr, self.len);
        }
    }
};



pub fn flushInstructionCache(ptr: [*]u8, len: usize) void {
    switch (builtin.cpu.arch) {
        .aarch64, .aarch64_be => {
            const cache_line_size: usize = 64;

            const start = @intFromPtr(ptr);
            const aligned_start = start & ~(cache_line_size - 1);
            const aligned_end = (start + len + cache_line_size - 1) & ~(cache_line_size - 1);

            var addr = aligned_start;
            while (addr < aligned_end) : (addr += cache_line_size) {
                asm volatile ("dc cvau, %[addr]"
                    :
                    : [addr] "r" (addr),
                    : .{ .memory = true });
            }

            asm volatile ("dsb ish"
                :
                :
                : .{ .memory = true });

            addr = aligned_start;
            while (addr < aligned_end) : (addr += cache_line_size) {
                asm volatile ("ic ivau, %[addr]"
                    :
                    : [addr] "r" (addr),
                    : .{ .memory = true });
            }

            asm volatile ("dsb ish"
                :
                :
                : .{ .memory = true });

            asm volatile ("isb"
                :
                :
                : .{ .memory = true });
        },
        else => {},
    }
}

pub fn flushICache(ptr: [*]u8, len: usize) void {
    flushInstructionCache(ptr, len);
}

pub fn pageSize() usize {
    return std.heap.pageSize();
}

// ─── High-resolution timer ────────────────────────────────────────────────────

pub fn highResTimer() u64 {
    if (comptime builtin.os.tag == .windows) {
        var freq: i64 = 0;
        var count: i64 = 0;
        _ = QueryPerformanceFrequency(&freq);
        _ = QueryPerformanceCounter(&count);
        if (freq == 0) return 0;
        return @as(u64, @intCast(count)) * 1_000_000_000 / @as(u64, @intCast(freq));
    } else {
        var ts: std.os.linux.timespec = undefined;
        const rc = std.os.linux.clock_gettime(.MONOTONIC, &ts);
        if (std.os.linux.errno(rc) != .SUCCESS) return 0;
        return @as(u64, @intCast(ts.sec)) * 1_000_000_000 +
               @as(u64, @intCast(ts.nsec));
    }
}

// ─── CPU feature detection ────────────────────────────────────────────────────

pub fn cpuFeatures() struct { avx2: bool, avx512: bool, neon: bool } {
    return .{
        .avx2 = switch (builtin.cpu.arch) {
            .x86_64 => std.Target.x86.featureSetHas(builtin.cpu.features, .avx2),
            else    => false,
        },
        .avx512 = switch (builtin.cpu.arch) {
            .x86_64 => std.Target.x86.featureSetHas(builtin.cpu.features, .avx512f),
            else    => false,
        },
        .neon = switch (builtin.cpu.arch) {
            .aarch64, .aarch64_be => true,
            else                  => false,
        },
    };
}

// ─── Low-level file-descriptor write ─────────────────────────────────────────

pub fn fdWrite(fd: i32, data: []const u8) void {
    if (data.len == 0) return;

    if (comptime builtin.os.tag == .windows) {
        const handle = GetStdHandle(if (fd == 2) STD_ERROR_HANDLE else STD_OUTPUT_HANDLE);
        if (handle == win.INVALID_HANDLE_VALUE) return;
        var written: win.DWORD = 0;
        _ = WriteFile(handle, data.ptr, @intCast(data.len), &written, null);
    } else {
        var done: usize = 0;
        while (done < data.len) {
            const rc = std.os.linux.write(fd, data.ptr + done, data.len - done);
            if (std.os.linux.errno(rc) != .SUCCESS) break;
            if (rc == 0) break;
            done += rc;
        }
    }
}

// ─── JIT memory helpers ────────────────────────────────────────────────────────

pub fn allocExecMem(size: usize) ![]u8 {
    const em = try ExecMemory.alloc(size);
    return em.ptr[0..em.len];
}

pub fn freeExecMem(mem: []u8) void {
    const em = ExecMemory{ .ptr = mem.ptr, .len = mem.len };
    em.free();
}

pub fn makeExec(code: []u8) !void {
    const em = ExecMemory{ .ptr = code.ptr, .len = code.len };
    try em.makeExecutable();
}
