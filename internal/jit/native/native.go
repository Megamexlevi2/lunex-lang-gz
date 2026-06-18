// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Licensed under the Mozilla Public License, Version 2.0.

// Package native provides pure-Go fast-path implementations of common numeric
// loop patterns recognised by the interpreter.
//
// Design goals:
//   - O(1) closed-form math wherever possible (no loops at all).
//   - Fallback to tight Go loops only when no closed form exists.
//   - Zero allocations — all functions operate on scalars.
//
// Pattern catalogue:
//   RunCount         i from start to limit (empty body)               → O(1)
//   RunCountSum      sum of integers start..limit                      → O(1) Gauss
//   RunCountMul      product start..limit (factorial-style)            → O(n) loop
//   RunFib           advance Fibonacci by N steps                      → O(n) loop
//   RunCountAccum    loop with constant delta per step                 → O(1)
//   RunCountAccumMul loop with multiplicative factor per step          → O(1) geometric
//   RunSumSquares    sum of squares start..limit                       → O(1)
//   RunCountStep     loop with step != 1                               → O(1)
//   RunStepAccum     step loop + constant delta accumulator            → O(1)
package native

import "runtime"

// SupportsNative always returns false in this build.
// All acceleration is handled by pure-Go math, no mmap needed.
func SupportsNative() bool { return false }

// Arch returns the current CPU architecture string (e.g. "amd64", "arm64").
func Arch() string { return runtime.GOARCH }

// MmapExec / MunmapExec / CallNativeI64 / EmitCountSum / EmitCount are stubs.
func MmapExec(_ int) []byte       { return nil }
func MunmapExec(_ []byte)         {}
func CallNativeI64(_ []byte, _ *int64, _ *int64, _ int64) {}
func EmitCountSum() []byte        { return nil }
func EmitCount() []byte           { return nil }

// ─── O(1) closed-form paths ───────────────────────────────────────────────────

// RunCount counts from start up to (but not including) limit.
// Returns the final counter value (= limit when limit > start, else start).
// Body is empty so no accumulation — pure counter advance.  O(1).
func RunCount(start, limit int64) int64 {
	if limit <= start {
		return start
	}
	return limit
}

// RunCountInclusive counts from start up through limit (inclusive).  O(1).
func RunCountInclusive(start, limit int64) int64 {
	if limit < start {
		return start
	}
	return limit + 1
}

// RunCountSum computes the sum of integers in [start, limit] (inclusive)
// and returns (finalCounter, sum).
//
// Uses the Gauss formula: sum(a..b) = (b-a+1)*(a+b)/2.  O(1).
func RunCountSum(start, limit int64) (int64, int64) {
	if limit < start {
		return start, 0
	}
	n := limit - start + 1
	sum := n * (start + limit) / 2
	return limit + 1, sum
}

// RunCountSumExclusive computes sum of integers in [start, limit) (exclusive).
// Equivalent to RunCountSum(start, limit-1).  O(1).
func RunCountSumExclusive(start, limit int64) (int64, int64) {
	if limit <= start {
		return start, 0
	}
	return RunCountSum(start, limit-1)
}

// RunSumSquares computes sum of squares in [start, limit] (inclusive).
// Formula: Σi²(a..b) = b*(b+1)*(2b+1)/6 − (a-1)*a*(2a-1)/6.  O(1).
func RunSumSquares(start, limit int64) (int64, int64) {
	if limit < start {
		return start, 0
	}
	sumSquaresUpTo := func(n int64) int64 {
		if n <= 0 {
			return 0
		}
		return n * (n + 1) * (2*n + 1) / 6
	}
	sum := sumSquaresUpTo(limit) - sumSquaresUpTo(start-1)
	return limit + 1, sum
}

// RunCountAccum advances a counter from start to limit by step=1, adding
// delta to accum each iteration.  Returns (finalCounter, finalAccum).
//
// n = limit − start + 1 (inclusive) iterations, so:
//   finalAccum = accumStart + n * delta.  O(1).
func RunCountAccum(start, limit, step, accumStart, delta int64) (int64, int64) {
	if step <= 0 {
		step = 1
	}
	if limit < start {
		return start, accumStart
	}
	// Number of iterations: ceil((limit-start)/step) — exclusive upper bound variant.
	// The interpreter always calls with inclusive limit already adjusted.
	n := (limit-start)/step + 1
	return start + n*step, accumStart + n*delta
}

// RunCountAccumMul accumulates by multiplication (geometric series).
// Each iteration: accum *= factor.
// After n iters: accum = accumStart * factor^n.  O(1) via intPow.
// Returns (finalCounter, finalAccum).
func RunCountAccumMul(start, limit, step, accumStart, factor int64) (int64, int64) {
	if step <= 0 {
		step = 1
	}
	if limit < start {
		return start, accumStart
	}
	n := (limit-start)/step + 1
	return start + n*step, accumStart * intPow(factor, n)
}

// RunStepAccum counts from start to limit (exclusive) by step, adding delta
// each iteration.  O(1) closed form.
// Returns (finalCounter, finalAccum).
func RunStepAccum(start, limitExcl, step, accumStart, delta int64) (int64, int64) {
	if step <= 0 {
		step = 1
	}
	if limitExcl <= start {
		return start, accumStart
	}
	n := (limitExcl - start + step - 1) / step // ceil division
	finalI := start + n*step
	finalAccum := accumStart + n*delta
	return finalI, finalAccum
}

// RunCountStep counts from start to limitExcl by step (no body).  O(1).
func RunCountStep(start, limitExcl, step int64) int64 {
	if step <= 0 {
		step = 1
	}
	if limitExcl <= start {
		return start
	}
	n := (limitExcl - start + step - 1) / step
	return start + n*step
}

// ─── O(n) fallbacks (no closed form) ─────────────────────────────────────────

// RunFib advances a Fibonacci sequence (a, b) by count steps.
// Returns the resulting pair (a, b).  O(n) — no closed form for integers.
func RunFib(a, b, count int64) (int64, int64) {
	for k := int64(0); k < count; k++ {
		a, b = b, a+b
	}
	return a, b
}

// RunCountMul computes the product of integers in [start, limit] (inclusive).
// Returns (finalCounter, product).  O(n).
func RunCountMul(start, limit int64) (int64, int64) {
	if limit < start {
		return start, 1
	}
	product := int64(1)
	for i := start; i <= limit; i++ {
		product *= i
	}
	return limit + 1, product
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// intPow computes base^exp for non-negative exp without math.Pow (avoids float).
func intPow(base, exp int64) int64 {
	if exp == 0 {
		return 1
	}
	result := int64(1)
	b := base
	e := exp
	for e > 0 {
		if e&1 == 1 {
			result *= b
		}
		b *= b
		e >>= 1
	}
	return result
}
