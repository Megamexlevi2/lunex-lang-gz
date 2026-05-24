// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

const std = @import("std");

pub const Arena = struct {
    inner: std.heap.ArenaAllocator,

    pub fn init(backing: std.mem.Allocator) Arena {
        return Arena{ .inner = std.heap.ArenaAllocator.init(backing) };
    }

    pub fn deinit(self: *Arena) void {
        self.inner.deinit();
    }

    pub fn allocator(self: *Arena) std.mem.Allocator {
        return self.inner.allocator();
    }

    pub fn reset(self: *Arena) void {
        _ = self.inner.reset(.retain_capacity);
    }
};

pub const Pool = struct {
    free_list: ?*Node,
    alloc: std.mem.Allocator,
    item_size: usize,

    const Node = struct {
        next: ?*Node,
    };

    pub fn init(alloc: std.mem.Allocator, item_size: usize) Pool {
        return Pool{
            .free_list = null,
            .alloc = alloc,
            .item_size = @max(item_size, @sizeOf(Node)),
        };
    }

    pub fn alloc_item(self: *Pool) !*anyopaque {
        if (self.free_list) |node| {
            self.free_list = node.next;
            return @ptrCast(node);
        }
        const mem = try self.alloc.alloc(u8, self.item_size);
        return @ptrCast(mem.ptr);
    }

    pub fn free_item(self: *Pool, ptr: *anyopaque) void {
        const node: *Node = @ptrCast(@alignCast(ptr));
        node.next = self.free_list;
        self.free_list = node;
    }
};
