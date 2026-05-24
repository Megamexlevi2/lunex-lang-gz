// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Lunex Source License — attribution required, copying prohibited.

// Package native provides pure-Go fallback implementations of built-in loop
// patterns that the Go-side interpreter can recognize and fast-path.
//
// All machine-code JIT compilation is handled exclusively by the embedded
// // Zig runtime (lunex-rt) via the NCP protocol.  This package no longer performs
// any mmap / VirtualAlloc or unsafe code generation; it exists solely to keep
// the interpreter's calling API stable.
package native

import "runtime"

// SupportsNative always returns false.
// Machine-code JIT is managed entirely by the Zig runtime.
func SupportsNative() bool { return false }

// Arch returns the current CPU architecture string (e.g. "amd64", "arm64").
func Arch() string { return runtime.GOARCH }

// MmapExec is a no-op stub.
// Executable memory allocation is performed by the Zig runtime.
func MmapExec(_ int) []byte { return nil }

// MunmapExec is a no-op stub.
func MunmapExec(_ []byte) {}

// CallNativeI64 is a no-op stub; it is never reached because SupportsNative
// returns false before this path is taken.
func CallNativeI64(_ []byte, _ *int64, _ *int64, _ int64) {}

// EmitCountSum is a no-op stub.
// // Code generation is performed by the Zig JIT inside lunex-rt.
func EmitCountSum() []byte { return nil }

// EmitCount is a no-op stub.
// // Code generation is performed by the Zig JIT inside lunex-rt.
func EmitCount() []byte { return nil }

// RunCount counts from start up to (but not including) limit and returns the
// final counter value.
func RunCount(start, limit int64) int64 {
	if limit <= start {
		return start
	}
	return limit
}

// RunCountSum counts from start to limit (inclusive) and returns
// (finalCounter, sum).
func RunCountSum(start, limit int64) (int64, int64) {
	var sum int64
	for i := start; i <= limit; i++ {
		sum += i
	}
	return limit + 1, sum
}

// RunFib advances a Fibonacci sequence (a, b) by count steps and returns the
// resulting pair (a, b).
func RunFib(a, b, count int64) (int64, int64) {
	for k := int64(0); k < count; k++ {
		a, b = b, a+b
	}
	return a, b
}

// RunCountAccum counts from start to limit by step, adding delta to accum on
// each iteration, and returns (finalCounter, finalAccum).
func RunCountAccum(start, limit, step, accumStart, delta int64) (int64, int64) {
	accum := accumStart
	i := start
	for i < limit {
		accum += delta
		i += step
	}
	return i, accum
}
