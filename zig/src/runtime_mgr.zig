// Lunex lang — Runtime directory manager.
// Manages the JIT cache directory and runtime metadata.
// Created by David Dev · GitHub: https://github.com/Megamexlevi2

const std     = @import("std");
const builtin = @import("builtin");

pub const RuntimeManager = struct {
    alloc:         std.mem.Allocator,
    io:            std.Io,
    dir:           []const u8,
    jit_cache_dir: []const u8,

    const VERSION_FILE     = "runtime.version";
    const JIT_CACHE_SUBDIR = "jit";
    const CURRENT_VERSION  = "lunex-rt-4.0";

    pub fn init(alloc: std.mem.Allocator, io: std.Io, environ: std.process.Environ) !RuntimeManager {
        const dir = try resolveRuntimeDir(alloc, environ);
        errdefer alloc.free(dir);

        try mkdirAll(io, dir);

        const jit_dir = try std.fs.path.join(alloc, &[_][]const u8{ dir, JIT_CACHE_SUBDIR });
        errdefer alloc.free(jit_dir);
        try mkdirAll(io, jit_dir);

        writeVersionManifest(io, dir);

        return RuntimeManager{
            .alloc         = alloc,
            .io            = io,
            .dir           = dir,
            .jit_cache_dir = jit_dir,
        };
    }

    pub fn deinit(self: *RuntimeManager) void {
        self.alloc.free(self.dir);
        self.alloc.free(self.jit_cache_dir);
    }

    pub fn basePath(self: *const RuntimeManager) []const u8 { return self.dir; }
    pub fn jitCachePath(self: *const RuntimeManager) []const u8 { return self.jit_cache_dir; }

    pub fn cleanJITCache(self: *RuntimeManager, io: std.Io) !void {
        var dir = std.Io.Dir.openDirAbsolute(io, self.jit_cache_dir, .{ .iterate = true }) catch return;
        defer dir.close(io);
        var it = dir.iterate();
        while (try it.next(io)) |entry| {
            if (entry.kind == .file) dir.deleteFile(io, entry.name) catch {};
        }
    }

    pub fn info(self: *const RuntimeManager, alloc: std.mem.Allocator) ![]const u8 {
        var jit_count: usize = 0;
        var jit_bytes: u64   = 0;

        if (std.Io.Dir.openDirAbsolute(self.io, self.jit_cache_dir, .{ .iterate = true })) |d| {
            var jdir = d;
            defer jdir.close(self.io);
            var it = jdir.iterate();
            while (it.next(self.io) catch null) |entry| {
                if (entry.kind == .file) {
                    jit_count += 1;
                    if (jdir.statFile(self.io, entry.name, .{})) |st| {
                        jit_bytes += st.size;
                    } else |_| {}
                }
            }
        } else |_| {}

        return std.fmt.allocPrint(alloc,
            \\Lunex Zig Runtime  {s}
            \\  OS:            {s}
            \\  Arch:          {s}
            \\  Runtime dir:   {s}
            \\  JIT cache:     {d} units ({d} KB)
            \\  JIT cache dir: {s}
            \\
        , .{
            CURRENT_VERSION,
            @tagName(builtin.os.tag),
            @tagName(builtin.cpu.arch),
            self.dir,
            jit_count,
            jit_bytes / 1024,
            self.jit_cache_dir,
        });
    }
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

fn mkdirAll(io: std.Io, path: []const u8) !void {
    std.Io.Dir.createDirAbsolute(io, path, .default_dir) catch |err| switch (err) {
        error.PathAlreadyExists => {},
        error.FileNotFound => {
            const parent = std.fs.path.dirname(path) orelse return err;
            try mkdirAll(io, parent);
            std.Io.Dir.createDirAbsolute(io, path, .default_dir) catch |e| switch (e) {
                error.PathAlreadyExists => {},
                else => return e,
            };
        },
        else => return err,
    };
}

fn getEnv(environ: std.process.Environ, key: []const u8) ?[]const u8 {
    if (comptime builtin.os.tag == .windows) return null;
    const val = std.process.Environ.getPosix(environ, key) orelse return null;
    if (val.len == 0) return null;
    return val;
}

fn resolveRuntimeDir(alloc: std.mem.Allocator, environ: std.process.Environ) ![]const u8 {
    if (comptime builtin.os.tag == .windows) {
        return alloc.dupe(u8, "C:\\lunex\\runtime");
    }

    if (comptime builtin.os.tag == .macos) {
        if (getEnv(environ, "HOME")) |home| {
            return std.fs.path.join(alloc, &[_][]const u8{ home, "Library", "Caches", "lunex", "runtime" });
        }
    }

    if (getEnv(environ, "XDG_CACHE_HOME")) |xdg| {
        return std.fs.path.join(alloc, &[_][]const u8{ xdg, "lunex", "runtime" });
    }

    if (getEnv(environ, "HOME")) |home| {
        return std.fs.path.join(alloc, &[_][]const u8{ home, ".cache", "lunex", "runtime" });
    }

    return alloc.dupe(u8, "/tmp/lunex-runtime");
}

fn writeVersionManifest(io: std.Io, dir_path: []const u8) void {
    var dir = std.Io.Dir.openDirAbsolute(io, dir_path, .{}) catch return;
    defer dir.close(io);
    dir.writeFile(io, .{
        .sub_path = RuntimeManager.VERSION_FILE,
        .data     = RuntimeManager.CURRENT_VERSION,
    }) catch {};
}
