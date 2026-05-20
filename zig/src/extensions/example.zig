// NTL lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Standard Zig library.
const std = @import("std");

// Import the runtime value system used by the VM.
const value = @import("../value.zig");
const Value = value.Value;

// Registers all math helper functions into the global runtime table.
// After this, scripts can directly call functions like:
//
// factorial(5)
// fibonacci(10)
// isPrime(97)
pub fn register(globals: *std.StringHashMap(Value)) !void {
    try globals.put("factorial", Value{ .native = factorial });
    try globals.put("fibonacci", Value{ .native = fibonacci });
    try globals.put("isPrime",   Value{ .native = isPrime   });
    try globals.put("clamp",     Value{ .native = clamp     });
    try globals.put("lerp",      Value{ .native = lerp      });
    try globals.put("randInt",   Value{ .native = randInt   });
}

// Calculates the factorial of a number.
//
// Example:
// factorial(5) -> 120
//
// Negative numbers are clamped to 0.
// The loop also stops after 20 to avoid huge overflow values.
fn factorial(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.One;

    var n = args[0].toInt();

    if (n < 0) n = 0;

    var result: i64 = 1;
    var i: i64 = 2;

    while (i <= n) : (i += 1) {
        result *%= i;

        // Prevents extremely large overflow explosions.
        if (i > 20) break;
    }

    return Value{ .int = result };
}

// Calculates a Fibonacci sequence value.
//
// Example:
// fibonacci(7) -> 13
//
// Uses an iterative approach instead of recursion
// because it is much faster and avoids stack usage.
fn fibonacci(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.Zero;

    const n = args[0].toInt();

    if (n <= 0) return Value.Zero;
    if (n == 1) return Value.One;

    var a: i64 = 0;
    var b2: i64 = 1;
    var i: i64 = 2;

    while (i <= n) : (i += 1) {
        const tmp = a + b2;

        a = b2;
        b2 = tmp;
    }

    return Value{ .int = b2 };
}

// Checks if a number is prime.
//
// Example:
// isPrime(11) -> true
// isPrime(12) -> false
//
// The algorithm skips even numbers after 2
// for better performance.
fn isPrime(args: []const Value) anyerror!Value {
    if (args.len == 0) return Value.False;

    const n = args[0].toInt();

    if (n < 2) return Value.False;
    if (n == 2) return Value.True;

    // Even numbers greater than 2 are never prime.
    if (@mod(n, 2) == 0) return Value.False;

    var i: i64 = 3;

    // Only checks divisors up to sqrt(n).
    while (i * i <= n) : (i += 2) {
        if (@mod(n, i) == 0) {
            return Value.False;
        }
    }

    return Value.True;
}

// Restricts a value between a minimum and maximum range.
//
// Example:
// clamp(15, 0, 10) -> 10
// clamp(-5, 0, 10) -> 0
fn clamp(args: []const Value) anyerror!Value {
    if (args.len < 3) {
        return if (args.len > 0) args[0] else Value.Zero;
    }

    const v  = args[0].toFloat();
    const lo = args[1].toFloat();
    const hi = args[2].toFloat();

    return Value{
        .float = @max(lo, @min(hi, v)),
    };
}

// Performs linear interpolation between two values.
//
// Example:
// lerp(0, 100, 0.5) -> 50
//
// Commonly used in animations, games,
// camera movement, and smooth transitions.
fn lerp(args: []const Value) anyerror!Value {
    if (args.len < 3) return Value.Zero;

    const a  = args[0].toFloat();
    const b2 = args[1].toFloat();
    const t  = args[2].toFloat();

    return Value{
        .float = a + (b2 - a) * t,
    };
}

// Internal random generator state.
//
// This uses a simple XORSHIFT algorithm.
// It is extremely fast and lightweight.
var rng_state: u64 = 0xDEADBEEFCAFEBABE;

// Generates a pseudo-random integer.
//
// Example:
// randInt()         -> random raw number
// randInt(1, 10)    -> number between 1 and 9
//
// The generator updates its internal state
// every time the function is called.
fn randInt(args: []const Value) anyerror!Value {
    // XORSHIFT randomization steps.
    rng_state ^= rng_state << 13;
    rng_state ^= rng_state >> 7;
    rng_state ^= rng_state << 17;

    const r: i64 = @bitCast(rng_state >> 1);

    // Range mode.
    if (args.len >= 2) {
        const lo = args[0].toInt();
        const hi = args[1].toInt();

        if (hi > lo) {
            return Value{
                .int = lo + @mod(r, hi - lo),
            };
        }
    }

    // Raw random integer.
    return Value{ .int = r };
}