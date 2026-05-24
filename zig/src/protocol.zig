// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

const std = @import("std");
const builtin = @import("builtin");

pub const NCP_MAGIC = [4]u8{ 0x4E, 0x54, 0x4C, 0x01 }; 
pub const NCP_VERSION: u8 = 0x01;
pub const FRAME_HEADER_SIZE: usize = 20;

pub const MSG_EXEC_PIPE: u8 = 0x01;
pub const MSG_EXEC_FILE: u8 = 0x02;
pub const MSG_RT_INFO: u8 = 0x03;
pub const MSG_RT_MANAGE: u8 = 0x04;
pub const MSG_RT_CLEAN: u8 = 0x05;
pub const MSG_KILL: u8 = 0x06;

pub const MSG_RESP_OK: u8 = 0x80;
pub const MSG_RESP_ERR: u8 = 0x81;
pub const MSG_RESP_EXIT: u8 = 0x82;
pub const MSG_STDOUT: u8 = 0x83;
pub const MSG_STDERR: u8 = 0x84;
pub const MSG_RUNTIME_DIR_INFO: u8 = 0x85;
pub const MSG_RT_INFO_RESP: u8 = 0x86;
pub const MSG_END: u8 = 0xFF;

pub const FLAG_NONE: u8 = 0x00;
pub const FLAG_COMPRESSED: u8 = 0x01; 
pub const FLAG_DEBUG: u8 = 0x02;
pub const FLAG_NO_CACHE: u8 = 0x04;

pub const FrameHeader = struct {
    msg_type: u8,
    flags: u8,
    seq: u16,
    payload_len: u32,
    payload_crc: u32,
};

pub const ErrorFrame = struct {
    code: u16,
    line: u32,
    column: u16,
    msg: []const u8,
    hint: []const u8,
};

const CRC32_TABLE: [256]u32 = blk: {
    @setEvalBranchQuota(100000);
    var table: [256]u32 = undefined;
    var i: u32 = 0;
    while (i < 256) : (i += 1) {
        var crc: u32 = i;
        var j: u32 = 0;
        while (j < 8) : (j += 1) {
            crc = if (crc & 1 != 0) 0xEDB88320 ^ (crc >> 1) else crc >> 1;
        }
        table[i] = crc;
    }
    break :blk table;
};

pub fn crc32(data: []const u8) u32 {
    var c: u32 = 0xFFFFFFFF;
    for (data) |byte| c = CRC32_TABLE[(c ^ byte) & 0xFF] ^ (c >> 8);
    return c ^ 0xFFFFFFFF;
}

pub fn readFrame(
    reader: anytype,
    alloc: std.mem.Allocator,
) !struct { header: FrameHeader, payload: []u8 } {
    var hdr_buf: [FRAME_HEADER_SIZE]u8 = undefined;
    const n = try reader.readAll(&hdr_buf);
    if (n < FRAME_HEADER_SIZE) return error.ConnectionClosed;

    if (!std.mem.eql(u8, hdr_buf[0..4], &NCP_MAGIC)) return error.BadMagic;
    if (hdr_buf[4] != NCP_VERSION) return error.BadVersion;

    const msg_type = hdr_buf[5];
    const flags = hdr_buf[6];
    const seq = std.mem.readInt(u16, hdr_buf[8..10], .little);
    const payload_len = std.mem.readInt(u32, hdr_buf[12..16], .little);
    const payload_crc = std.mem.readInt(u32, hdr_buf[16..20], .little);

    var payload: []u8 = &[_]u8{};
    if (payload_len > 0) {
        payload = try alloc.alloc(u8, payload_len);
        const pn = try reader.readAll(payload);
        if (pn < payload_len) return error.TruncatedPayload;
        if (crc32(payload) != payload_crc) return error.CRCMismatch;
    }

    return .{
        .header = FrameHeader{
            .msg_type = msg_type,
            .flags = flags,
            .seq = seq,
            .payload_len = payload_len,
            .payload_crc = payload_crc,
        },
        .payload = payload,
    };
}

pub fn writeFrame(
    writer: anytype,
    msg_type: u8,
    flags: u8,
    seq: u16,
    payload: []const u8,
) !void {
    var hdr: [FRAME_HEADER_SIZE]u8 = undefined;
    hdr[0] = NCP_MAGIC[0];
    hdr[1] = NCP_MAGIC[1];
    hdr[2] = NCP_MAGIC[2];
    hdr[3] = NCP_MAGIC[3];
    hdr[4] = NCP_VERSION;
    hdr[5] = msg_type;
    hdr[6] = flags;
    hdr[7] = 0x00;
    std.mem.writeInt(u16, hdr[8..10], seq, .little);
    hdr[10] = 0x00;
    hdr[11] = 0x00;
    std.mem.writeInt(u32, hdr[12..16], @intCast(payload.len), .little);
    std.mem.writeInt(u32, hdr[16..20], if (payload.len > 0) crc32(payload) else 0, .little);
    try writer.writeAll(&hdr);
    if (payload.len > 0) try writer.writeAll(payload);
}

pub fn sendOK(writer: anytype, seq: u16) !void {
    try writeFrame(writer, MSG_RESP_OK, FLAG_NONE, seq, &[_]u8{});
}

pub fn sendExit(writer: anytype, seq: u16, code: u8) !void {
    try writeFrame(writer, MSG_RESP_EXIT, FLAG_NONE, seq, &[_]u8{code});
}

pub fn sendEnd(writer: anytype, seq: u16) !void {
    try writeFrame(writer, MSG_END, FLAG_NONE, seq, &[_]u8{});
}

pub fn sendStdout(writer: anytype, seq: u16, data: []const u8) !void {
    try writeFrame(writer, MSG_STDOUT, FLAG_NONE, seq, data);
}

pub fn sendStderr(writer: anytype, seq: u16, data: []const u8) !void {
    try writeFrame(writer, MSG_STDERR, FLAG_NONE, seq, data);
}

pub fn sendError(
    writer: anytype,
    alloc: std.mem.Allocator,
    seq: u16,
    ef: ErrorFrame,
) !void {
    const msg_len: u16 = @intCast(ef.msg.len);
    const hint_len: u16 = @intCast(ef.hint.len);
    const size = 2 + 4 + 2 + 2 + ef.msg.len + 2 + ef.hint.len;
    var buf = try alloc.alloc(u8, size);
    defer alloc.free(buf);

    var off: usize = 0;
    std.mem.writeInt(u16, buf[off..][0..2], ef.code, .little);
    off += 2;
    std.mem.writeInt(u32, buf[off..][0..4], ef.line, .little);
    off += 4;
    std.mem.writeInt(u16, buf[off..][0..2], ef.column, .little);
    off += 2;
    std.mem.writeInt(u16, buf[off..][0..2], msg_len, .little);
    off += 2;
    @memcpy(buf[off..][0..ef.msg.len], ef.msg);
    off += ef.msg.len;
    std.mem.writeInt(u16, buf[off..][0..2], hint_len, .little);
    off += 2;
    @memcpy(buf[off..][0..ef.hint.len], ef.hint);
    try writeFrame(writer, MSG_RESP_ERR, FLAG_NONE, seq, buf);
}

pub fn sendRuntimeDirInfo(
    writer: anytype,
    alloc: std.mem.Allocator,
    seq: u16,
    dir: []const u8,
) !void {
    var buf = try alloc.alloc(u8, 4 + dir.len);
    defer alloc.free(buf);
    std.mem.writeInt(u32, buf[0..4], @intCast(dir.len), .little);
    @memcpy(buf[4..], dir);
    try writeFrame(writer, MSG_RUNTIME_DIR_INFO, FLAG_NONE, seq, buf);
}
