const std = @import("std");

pub fn build(b: *std.Build) void {
    const target   = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const update_sources = b.addUpdateSourceFiles();
    const write_files    = b.addWriteFiles();
    const generated      = generateExtensionsRegistry(b, write_files) catch @panic("failed to generate extensions registry");
    update_sources.addCopyFileToSource(generated, "src/extensions_gen.zig");

    const extensions_gen_module = b.createModule(.{
        .root_source_file = b.path("src/extensions_gen.zig"),
        .target   = target,
        .optimize = optimize,
    });

    // ── AsmJit C++ multi-arch JIT backend ────────────────────────────────────
    // In Zig 0.17.0-dev, addCSourceFile and link_libcpp live on the Module.
    const main_module = b.createModule(.{
        .root_source_file = b.path("src/main.zig"),
        .target   = target,
        .optimize = optimize,
        .link_libcpp = true,
        .imports  = &.{
            .{ .name = "extensions_gen", .module = extensions_gen_module },
        },
    });
    main_module.addCSourceFile(.{
        .file  = b.path("src/jit_asmjit.cpp"),
        .flags = &.{ "-std=c++17", "-O2", "-fno-exceptions", "-fno-rtti" },
    });
    main_module.addCSourceFile(.{
        .file  = b.path("src/crypto.cpp"),
        .flags = &.{ "-std=c++17", "-O3", "-fno-exceptions", "-fno-rtti" },
    });

    const exe = b.addExecutable(.{
        .name        = "lunex-rt",
        .root_module = main_module,
    });

    b.installArtifact(exe);

    const unit_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/vm_test.zig"),
            .target   = target,
            .optimize = optimize,
        }),
    });

    const run_tests = b.addRunArtifact(unit_tests);
    const test_step = b.step("test", "Run unit tests");
    test_step.dependOn(&update_sources.step);
    test_step.dependOn(&run_tests.step);

    const run_cmd = b.addRunArtifact(exe);
    run_cmd.step.dependOn(&update_sources.step);
    run_cmd.step.dependOn(b.getInstallStep());
    if (b.args) |args| run_cmd.addArgs(args);

    const run_step = b.step("run", "Run lunex-rt");
    run_step.dependOn(&run_cmd.step);

    const ext_step = b.step("ext", "Rescan extensions and regenerate registry");
    ext_step.dependOn(&update_sources.step);

    const bench_cmd = b.addRunArtifact(exe);
    bench_cmd.step.dependOn(&update_sources.step);
    bench_cmd.addArg("jit-bench");

    const bench_step = b.step("bench", "Run JIT benchmark");
    bench_step.dependOn(&bench_cmd.step);

    const cross_targets = [_]std.Target.Query{
        .{ .cpu_arch = .x86_64,  .os_tag = .linux,   .abi = .musl },
        .{ .cpu_arch = .aarch64, .os_tag = .linux,   .abi = .musl },
        .{ .cpu_arch = .riscv64, .os_tag = .linux,   .abi = .musl },
        .{ .cpu_arch = .x86_64,  .os_tag = .freebsd, .abi = .none },
    };

    const cross_step = b.step("cross", "Build for all supported platforms");
    cross_step.dependOn(&update_sources.step);

    for (cross_targets) |ct| {
        const cross_target = b.resolveTargetQuery(ct);

        const cross_ext_module = b.createModule(.{
            .root_source_file = b.path("src/extensions_gen.zig"),
            .target   = cross_target,
            .optimize = .ReleaseFast,
        });

        const cross_main_module = b.createModule(.{
            .root_source_file = b.path("src/main.zig"),
            .target   = cross_target,
            .optimize = .ReleaseFast,
            .link_libcpp = true,
            .imports  = &.{
                .{ .name = "extensions_gen", .module = cross_ext_module },
            },
        });
        cross_main_module.addCSourceFile(.{
            .file  = b.path("src/jit_asmjit.cpp"),
            .flags = &.{ "-std=c++17", "-O3", "-fno-exceptions", "-fno-rtti" },
        });
        cross_main_module.addCSourceFile(.{
            .file  = b.path("src/crypto.cpp"),
            .flags = &.{ "-std=c++17", "-O3", "-fno-exceptions", "-fno-rtti" },
        });

        const cross_exe = b.addExecutable(.{
            .name        = b.fmt("lunex-rt-{s}-{s}", .{
                @tagName(ct.cpu_arch.?),
                @tagName(ct.os_tag.?),
            }),
            .root_module = cross_main_module,
        });

        const cross_install = b.addInstallArtifact(cross_exe, .{});
        cross_step.dependOn(&cross_install.step);
    }
}

fn appendFmt(
    list:  *std.ArrayListUnmanaged(u8),
    alloc: std.mem.Allocator,
    comptime fmt: []const u8,
    args: anytype,
) !void {
    const s = try std.fmt.allocPrint(alloc, fmt, args);
    defer alloc.free(s);
    try list.appendSlice(alloc, s);
}

fn generateExtensionsRegistry(b: *std.Build, wf: anytype) !std.Build.LazyPath {
    const alloc = b.allocator;
    const io    = b.graph.io;

    var content: std.ArrayListUnmanaged(u8) = .empty;
    defer content.deinit(alloc);

    try content.appendSlice(alloc,
        "// Lunex lang\n" ++
        "// Created by David Dev - GitHub: https://github.com/Megamexlevi2\n" ++
        "// Auto-generated by build.zig — do not edit by hand.\n\n" ++
        "pub const Extension = struct {\n" ++
        "    name: []const u8,\n" ++
        "    path: []const u8,\n" ++
        "};\n\n" ++
        "pub const extensions: []const Extension = &.{\n");

    // In Zig 0.17.0-dev all Dir ops take io: std.Io as first or second param.
    var ext_dir = std.Io.Dir.openDir(b.build_root.handle, io, "src/extensions", .{ .iterate = true }) catch {
        try content.appendSlice(alloc, "};\n\n");
        try content.appendSlice(alloc,
            "pub fn registerAll(_: anytype, _: anytype) anyerror!void {}\n");
        return wf.add("src/extensions_gen.zig", try content.toOwnedSlice(alloc));
    };
    defer ext_dir.close(io);

    var it = ext_dir.iterate();
    while (try it.next(io)) |entry| {
        if (entry.kind != .file) continue;
        if (!std.mem.endsWith(u8, entry.name, ".zig")) continue;
        const stem = entry.name[0 .. entry.name.len - 4];
        try appendFmt(&content, alloc,
            "    .{{ .name = \"{s}\", .path = \"extensions/{s}.zig\" }},\n",
            .{ stem, stem });
    }

    try content.appendSlice(alloc, "};\n\n");
    try content.appendSlice(alloc,
        "pub fn registerAll(_: anytype, _: anytype) anyerror!void {}\n");

    return wf.add("src/extensions_gen.zig", try content.toOwnedSlice(alloc));
}
