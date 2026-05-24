const std = @import("std");
const builtin = @import("builtin");
const proto = @import("protocol.zig");
const rtmgr = @import("runtime_mgr.zig");
const errfmt = @import("errfmt.zig");
const vm_mod = @import("vm.zig");
const builtins = @import("builtins.zig");
const platform = @import("platform.zig");
const jit = @import("jit.zig");

const win = std.os.windows;

const VM = vm_mod.VM;
const RuntimeManager = rtmgr.RuntimeManager;

const fd_t = i32;
const STDIN_FD: fd_t = 0;
const STDOUT_FD: fd_t = 1;
const STDERR_FD: fd_t = 2;

const WinHandles = if (builtin.os.tag == .windows) struct {
    extern "kernel32" fn GetStdHandle(nStdHandle: win.DWORD) callconv(.winapi) win.HANDLE;

    fn of(fd: fd_t) win.HANDLE {
        return GetStdHandle(switch (fd) {
            STDIN_FD => @bitCast(@as(i32, -10)),
            STDOUT_FD => @bitCast(@as(i32, -11)),
            STDERR_FD => @bitCast(@as(i32, -12)),
            else => @bitCast(@as(i32, -12)),
        });
    }
} else struct {};

extern "kernel32" fn WriteFile(
    hFile: win.HANDLE,
    lpBuffer: [*]const u8,
    nNumberOfBytesToWrite: win.DWORD,
    lpNumberOfBytesWritten: ?*win.DWORD,
    lpOverlapped: ?*anyopaque,
) callconv(if (builtin.os.tag == .windows) .winapi else .c) win.BOOL;

extern "kernel32" fn ReadFile(
    hFile: win.HANDLE,
    lpBuffer: [*]u8,
    nNumberOfBytesToRead: win.DWORD,
    lpNumberOfBytesRead: ?*win.DWORD,
    lpOverlapped: ?*anyopaque,
) callconv(.winapi) win.BOOL;

fn sysWrite(fd: fd_t, data: []const u8) !void {
    if (comptime builtin.os.tag == .windows) {
        const h = WinHandles.of(fd);
        if (h == win.INVALID_HANDLE_VALUE) return error.InvalidHandle;
        var written: win.DWORD = 0;
        if (!WriteFile(h, data.ptr, @intCast(data.len), &written, null).toBool()) return error.WriteFailed;
    } else {
        var done: usize = 0;
        while (done < data.len) {
            const rc = std.os.linux.write(fd, data.ptr + done, data.len - done);
            const errno = std.os.linux.errno(rc);
            if (errno != .SUCCESS) return error.WriteFailed;
            const n = rc;
            if (n == 0) break;
            done += n;
        }
    }
}

fn sysRead(fd: fd_t, buf: []u8) !usize {
    if (comptime builtin.os.tag == .windows) {
        const h = WinHandles.of(fd);
        var n: win.DWORD = 0;
        if (!ReadFile(h, buf.ptr, @intCast(buf.len), &n, null).toBool()) return error.ReadFailed;
        return @intCast(n);
    } else {
        const rc = std.os.linux.read(fd, buf.ptr, buf.len);
        const errno = std.os.linux.errno(rc);
        if (errno == .AGAIN or errno == .INTR) return error.WouldBlock;
        if (errno != .SUCCESS) return error.ReadFailed;
        return rc;
    }
}

fn sysClose(fd: fd_t) void {
    if (comptime builtin.os.tag == .windows) return;
    _ = std.os.linux.close(fd);
}

fn sysDup(fd: fd_t) !fd_t {
    if (comptime builtin.os.tag == .windows) return error.UnsupportedOnWindows;
    const rc = std.os.linux.dup(fd);
    const errno = std.os.linux.errno(rc);
    if (errno != .SUCCESS) return error.DupFailed;
    return @intCast(rc);
}

fn sysDup2(old: fd_t, new: fd_t) !void {
    if (comptime builtin.os.tag == .windows) return error.UnsupportedOnWindows;
    const rc = std.os.linux.dup2(old, new);
    const errno = std.os.linux.errno(rc);
    if (errno != .SUCCESS) return error.Dup2Failed;
}

fn sysPipe() ![2]fd_t {
    if (comptime builtin.os.tag == .windows) return error.UnsupportedOnWindows;
    var fds: [2]fd_t = undefined;
    const rc = std.os.linux.pipe(&fds);
    const errno = std.os.linux.errno(rc);
    if (errno != .SUCCESS) return error.PipeFailed;
    return fds;
}

fn posixWrite(fd: fd_t, data: []const u8) void {
    sysWrite(fd, data) catch {};
}

fn posixWriteAlloc(
    fd: fd_t,
    alloc: std.mem.Allocator,
    comptime fmt: []const u8,
    args: anytype,
) void {
    const msg = std.fmt.allocPrint(alloc, fmt, args) catch return;
    defer alloc.free(msg);
    posixWrite(fd, msg);
}

fn posixReadAll(fd: fd_t, alloc: std.mem.Allocator, max_bytes: usize) ![]u8 {
    var list: std.ArrayListUnmanaged(u8) = .empty;
    errdefer list.deinit(alloc);
    var buf: [65536]u8 = undefined;
    while (true) {
        const n = try sysRead(fd, &buf);
        if (n == 0) break;
        try list.appendSlice(alloc, buf[0..n]);
        if (list.items.len >= max_bytes) break;
    }
    return try list.toOwnedSlice(alloc);
}

fn readFileAlloc(alloc: std.mem.Allocator, path: []const u8, max_bytes: usize) ![]u8 {
    // Open file using linux syscall directly (no Io context needed).
    const flags = std.os.linux.O{ .ACCMODE = .RDONLY };
    const path_z = try std.posix.toPosixPath(path);
    const fd_rc = std.os.linux.open(&path_z, flags, 0);
    if (std.os.linux.errno(fd_rc) != .SUCCESS) return error.FileOpenFailed;
    const fd: i32 = @intCast(fd_rc);
    defer _ = std.os.linux.close(fd);
    var list: std.ArrayListUnmanaged(u8) = .empty;
    errdefer list.deinit(alloc);
    var buf: [65536]u8 = undefined;
    while (true) {
        const rc = std.os.linux.read(fd, &buf, buf.len);
        if (std.os.linux.errno(rc) != .SUCCESS) return error.FileReadFailed;
        if (rc == 0) break;
        try list.appendSlice(alloc, buf[0..rc]);
        if (list.items.len >= max_bytes) break;
    }
    return try list.toOwnedSlice(alloc);
}

const NCPReader = struct {
    buf: [65536]u8 = undefined,
    pos: usize = 0,
    end: usize = 0,

    pub fn readAll(self: *NCPReader, dst: []u8) !usize {
        var total: usize = 0;
        while (total < dst.len) {
            if (self.pos >= self.end) {
                const n = sysRead(STDIN_FD, &self.buf) catch |e| {
                    if (total == 0) return e;
                    break;
                };
                if (n == 0) break;
                self.pos = 0;
                self.end = n;
            }
            const take = @min(self.end - self.pos, dst.len - total);
            @memcpy(dst[total..][0..take], self.buf[self.pos..][0..take]);
            self.pos += take;
            total += take;
        }
        return total;
    }
};

const NCPWriter = struct {
    buf: [65536]u8 = undefined,
    len: usize = 0,

    pub fn writeAll(self: *NCPWriter, data: []const u8) !void {
        var remaining = data;
        while (remaining.len > 0) {
            if (self.buf.len == self.len) try self.flush();
            const take = @min(remaining.len, self.buf.len - self.len);
            @memcpy(self.buf[self.len..][0..take], remaining[0..take]);
            self.len += take;
            remaining = remaining[take..];
        }
    }

    pub fn flush(self: *NCPWriter) !void {
        if (self.len == 0) return;
        try sysWrite(STDOUT_FD, self.buf[0..self.len]);
        self.len = 0;
    }
};

fn extractNTZFromNC(data: []const u8) ![]const u8 {
    if (data.len >= 48 and data[0] == 'n' and data[1] == 't' and data[2] == 'l' and data[3] == 'i') {
        const ntz_len: u32 =
            @as(u32, data[40]) | (@as(u32, data[41]) << 8) |
            (@as(u32, data[42]) << 16) | (@as(u32, data[43]) << 24);

        if (ntz_len > 0 and ntz_len <= data.len - 48) return data[data.len - ntz_len ..];
        return error.InvalidBytecodeFormat;
    }
    return data;
}

fn runNC(machine: *VM, data: []const u8) !vm_mod.Value {
    const code = try extractNTZFromNC(data);
    return machine.execBytecode(code);
}

pub fn main(init: std.process.Init) !void {
    const alloc = init.gpa;
    const args = try init.minimal.args.toSlice(init.arena.allocator());

    if (args.len < 2) {
        printUsage();
        std.process.exit(1);
    }

    const cmd = args[1];

    if (std.mem.eql(u8, cmd, "ncp-server")) {
        try runNCPServer(alloc, init.io, init.minimal.environ);
    } else if (std.mem.eql(u8, cmd, "exec")) {
        if (args.len < 3) {
            posixWrite(STDERR_FD, "Usage: lunex-rt exec <file.nc>\n");
            std.process.exit(1);
        }
        try runFileStandalone(alloc, args[2], false);
    } else if (std.mem.eql(u8, cmd, "pipe")) {
        try runPipeStandalone(alloc);
    } else if (std.mem.eql(u8, cmd, "info")) {
        try printRuntimeInfo(alloc, init.io, init.minimal.environ);
    } else if (std.mem.eql(u8, cmd, "bench")) {
        if (args.len < 3) {
            posixWrite(STDERR_FD, "Usage: lunex-rt bench <file.nc>\n");
            std.process.exit(1);
        }
        try runFileStandalone(alloc, args[2], true);
    } else if (std.mem.eql(u8, cmd, "jit-bench")) {
        try jit.benchmark(alloc);
    } else {
        try runFileStandalone(alloc, cmd, false);
    }
}

fn runNCPServer(alloc: std.mem.Allocator, io: std.Io, environ: std.process.Environ) !void {
    var rt = try RuntimeManager.init(alloc, io, environ);
    defer rt.deinit();

    var in = NCPReader{};
    var out = NCPWriter{};

    try proto.sendRuntimeDirInfo(&out, alloc, 0, rt.basePath());
    try out.flush();

    var seq: u16 = 0;

    while (true) {
        const frame = proto.readFrame(&in, alloc) catch |err| switch (err) {
            error.ConnectionClosed => break,
            error.BadMagic => {
                try ncpErr(&out, alloc, seq, 6001, "Bad NCP magic — version mismatch?", "");
                try out.flush();
                continue;
            },
            error.BadVersion => {
                try ncpErr(&out, alloc, seq, 6002, "NCP version mismatch — rebuild lunex", "");
                try out.flush();
                continue;
            },
            error.CRCMismatch => {
                try ncpErr(&out, alloc, seq, 6003, "Payload CRC mismatch — data corrupted", "");
                try out.flush();
                continue;
            },
            else => break,
        };
        defer if (frame.payload.len > 0) alloc.free(frame.payload);

        seq = frame.header.seq;
        const dbg = (frame.header.flags & proto.FLAG_DEBUG) != 0;

        switch (frame.header.msg_type) {
            proto.MSG_EXEC_PIPE => {
                if (dbg) std.debug.print("[lunex-rt] exec-pipe {d} bytes\n", .{frame.payload.len});
                try dispatchExecPipe(alloc, &out, frame.header.seq, frame.payload);
                try out.flush();
            },
            proto.MSG_EXEC_FILE => {
                const path = std.mem.sliceTo(frame.payload, 0);
                if (dbg) std.debug.print("[lunex-rt] exec-file {s}\n", .{path});
                try dispatchExecFile(alloc, &out, frame.header.seq, path);
                try out.flush();
            },
            proto.MSG_RT_INFO => {
                const s = try rt.info(alloc);
                defer alloc.free(s);
                try proto.writeFrame(&out, proto.MSG_RT_INFO_RESP, proto.FLAG_NONE, frame.header.seq, s);
                try out.flush();
            },
            proto.MSG_RT_CLEAN => {
                try rt.cleanJITCache(io);
                try proto.sendOK(&out, frame.header.seq);
                try out.flush();
            },
            proto.MSG_KILL => {
                try proto.sendEnd(&out, frame.header.seq);
                try out.flush();
                break;
            },
            else => {},
        }
    }
}

const PipeCapture = struct {
    buf: std.ArrayListUnmanaged(u8) = .empty,
    fd: fd_t,
    alloc: std.mem.Allocator,

    fn threadFn(cap: *PipeCapture) void {
        var tmp: [4096]u8 = undefined;
        while (true) {
            const rc = std.os.linux.read(cap.fd, &tmp, tmp.len);
            const errno = std.os.linux.errno(rc);
            if (errno != .SUCCESS) break;
            const n = rc;
            if (n == 0) break;
            cap.buf.appendSlice(cap.alloc, tmp[0..n]) catch break;
        }
        sysClose(cap.fd);
    }
};

fn dispatchExecPipe(alloc: std.mem.Allocator, writer: anytype, seq: u16, ncData: []const u8) !void {
    if (comptime builtin.os.tag != .windows) {
        try dispatchExecPipePosix(alloc, writer, seq, ncData);
    } else {
        try dispatchExecPipeDirect(alloc, writer, seq, ncData);
    }
}

fn dispatchExecFile(alloc: std.mem.Allocator, writer: anytype, seq: u16, path: []const u8) !void {
    const data = readFileAlloc(alloc, path, 256 * 1024 * 1024) catch |err| {
        const msg = try std.fmt.allocPrint(alloc, "Cannot read '{s}': {s}", .{ path, @errorName(err) });
        defer alloc.free(msg);
        try ncpErr(writer, alloc, seq, 7001, msg, "Check path and permissions.");
        try proto.sendEnd(writer, seq);
        return;
    };
    defer alloc.free(data);
    try dispatchExecPipe(alloc, writer, seq, data);
}

fn runVM(alloc: std.mem.Allocator, ncData: []const u8) !?proto.ErrorFrame {
    var machine = try VM.init(alloc);
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, alloc);
    _ = runNC(&machine, ncData) catch |err| {
        const msg = errfmt.formatError(alloc, err) catch @errorName(err);
        return proto.ErrorFrame{
            .code = errfmt.errorCode(err),
            .line = 0,
            .column = 0,
            .msg = msg,
            .hint = errfmt.errorHint(err),
        };
    };
    return null;
}

fn dispatchExecPipePosix(alloc: std.mem.Allocator, writer: anytype, seq: u16, ncData: []const u8) !void {
    const saved_out = try sysDup(STDOUT_FD);
    const saved_err = try sysDup(STDERR_FD);
    defer {
        sysDup2(saved_out, STDOUT_FD) catch {};
        sysDup2(saved_err, STDERR_FD) catch {};
        sysClose(saved_out);
        sysClose(saved_err);
    }

    const out_pipe = try sysPipe();
    const err_pipe = try sysPipe();
    try sysDup2(out_pipe[1], STDOUT_FD);
    try sysDup2(err_pipe[1], STDERR_FD);
    sysClose(out_pipe[1]);
    sysClose(err_pipe[1]);

    var out_cap = PipeCapture{ .buf = .empty, .fd = out_pipe[0], .alloc = alloc };
    var err_cap = PipeCapture{ .buf = .empty, .fd = err_pipe[0], .alloc = alloc };
    defer out_cap.buf.deinit(alloc);
    defer err_cap.buf.deinit(alloc);

    const out_thread = try std.Thread.spawn(.{}, PipeCapture.threadFn, .{ &out_cap });
    const err_thread = try std.Thread.spawn(.{}, PipeCapture.threadFn, .{ &err_cap });

    var machine = try VM.init(alloc);
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, alloc);

    var vm_err: ?proto.ErrorFrame = null;
    _ = runNC(&machine, ncData) catch |err| {
        const msg = errfmt.formatError(alloc, err) catch @errorName(err);
        vm_err = .{
            .code = errfmt.errorCode(err),
            .line = 0,
            .column = 0,
            .msg = msg,
            .hint = errfmt.errorHint(err),
        };
    };

    sysDup2(saved_out, STDOUT_FD) catch {};
    sysDup2(saved_err, STDERR_FD) catch {};
    out_thread.join();
    err_thread.join();

    if (out_cap.buf.items.len > 0) try proto.sendStdout(writer, seq, out_cap.buf.items);
    if (err_cap.buf.items.len > 0) try proto.sendStderr(writer, seq, err_cap.buf.items);

    if (vm_err) |ef| {
        try proto.sendError(writer, alloc, seq, ef);
    } else {
        try proto.sendExit(writer, seq, 0);
    }
}

fn dispatchExecPipeDirect(alloc: std.mem.Allocator, writer: anytype, seq: u16, ncData: []const u8) !void {
    if (try runVM(alloc, ncData)) |ef| {
        try ncpErr(writer, alloc, seq, ef.code, ef.msg, ef.hint);
        return;
    }
    try proto.sendExit(writer, seq, 0);
}

fn runFileStandalone(alloc: std.mem.Allocator, path: []const u8, bench_mode: bool) !void {
    const t0 = platform.highResTimer();

    const data = readFileAlloc(alloc, path, 256 * 1024 * 1024) catch |err| {
        posixWriteAlloc(STDERR_FD, alloc, "lunex-rt: can't read '{s}': {s}\n", .{ path, @errorName(err) });
        std.process.exit(1);
    };
    defer alloc.free(data);

    var machine = try VM.init(alloc);
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, alloc);

    _ = runNC(&machine, data) catch |err| {
        posixWriteAlloc(STDERR_FD, alloc, "lunex-rt: {s}\n", .{ @errorName(err) });
        std.process.exit(1);
    };

    if (bench_mode) {
        const elapsed_ns = platform.highResTimer() - t0;
        posixWriteAlloc(STDERR_FD, alloc,
            "\n[bench] {d:.3}ms  arch={s}  os={s}\n",
            .{ @as(f64, @floatFromInt(elapsed_ns)) / 1_000_000.0,
               @tagName(platform.currentArch()), @tagName(platform.currentOS()) });
        machine.printStats();
    }
}

fn runPipeStandalone(alloc: std.mem.Allocator) !void {
    const data = try posixReadAll(STDIN_FD, alloc, 512 * 1024 * 1024);
    defer alloc.free(data);
    if (data.len == 0) return;

    var machine = try VM.init(alloc);
    defer machine.deinit();
    try builtins.registerAll(&machine.globals, alloc);

    _ = runNC(&machine, data) catch |err| {
        posixWriteAlloc(STDERR_FD, alloc, "lunex-rt: {s}\n", .{ @errorName(err) });
        std.process.exit(1);
    };
}

fn printRuntimeInfo(alloc: std.mem.Allocator, io: std.Io, environ: std.process.Environ) !void {
    var rt = try RuntimeManager.init(alloc, io, environ);
    defer rt.deinit();
    const s = try rt.info(alloc);
    defer alloc.free(s);
    posixWrite(STDOUT_FD, s);

    const feats = platform.cpuFeatures();
    posixWriteAlloc(STDOUT_FD, alloc,
        "  Arch:   {s}\n  OS:     {s}\n  CPU:    avx2={} avx512={} neon={}\n" ++
        "  JIT:    {}\n",
        .{ @tagName(platform.currentArch()), @tagName(platform.currentOS()),
           feats.avx2, feats.avx512, feats.neon,
           platform.currentArch() != .unknown });
}

fn printUsage() void {
    std.debug.print(
        \\lunex-rt — Lunex Zig Runtime  (x86_64 / aarch64 / riscv64 JIT)
        \\
        \\Usage:
        \\  lunex-rt ncp-server          start the NCP server (used by ntl binary)
        \\  lunex-rt exec <file.nc>      run a .nc bytecode file
        \\  lunex-rt pipe                read .nc from stdin and run it
        \\  lunex-rt info                print platform and runtime info
        \\  lunex-rt bench <file.nc>     run and print timing
        \\  lunex-rt jit-bench           run the built-in JIT self-benchmark
        \\
    , .{});
}

fn ncpErr(
    writer: anytype,
    alloc: std.mem.Allocator,
    seq: u16,
    code: u16,
    msg: []const u8,
    hint: []const u8,
) !void {
    try proto.sendError(writer, alloc, seq, .{
        .code = code,
        .line = 0,
        .column = 0,
        .msg = msg,
        .hint = hint,
    });
}