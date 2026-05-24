// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Lunex Source License — attribution required, copying prohibited.

// Package jit provides the Go-side profiler and pure-Go fallback
// implementations of hot loop patterns recognized by the interpreter.
//
// All machine-code JIT compilation is delegated to the embedded Zig runtime
// // (lunex-rt).  When the interpreter sends Lunex bytecode to lunex-rt via the NCP
// pipe, the Zig VM and JIT compiler (zig/src/jit.zig) handle native code
// generation automatically for hot functions.  This package keeps a separate
// Go-side call profile that feeds the human-readable profiling report.
package jit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"lunex/internal/jit/native"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HotThreshold is the call count above which a function is considered hot.
// // Kept for API compatibility with the interpreter; hot detection inside lunex-rt
// uses the Zig profiler's own threshold.
const HotThreshold int64 = 50

// Execution tier constants.
const (
	TierInterpret uint32 = iota // Interpreted by the Go-side interpreter.
	TierFastGo                  // Recognized loop pattern, pure-Go fast path.
	TierNative                  // Compiled to native code by the Zig JIT.
)

var tierLabel = [...]string{"interpret", "fast-go", "zig-jit"}

// FnProfile holds per-function call statistics collected by the Go profiler.
type FnProfile struct {
	Name    string
	calls   atomic.Int64
	totalNs atomic.Int64
	tier    atomic.Uint32
	mu      sync.Mutex
}

// NewProfile creates a new FnProfile for the named function.
// New profiles start at TierNative because all functions that reach the
// profiler have already been dispatched through the Zig runtime.
func NewProfile(name string) *FnProfile {
	p := &FnProfile{Name: name}
	p.tier.Store(TierNative)
	return p
}

// Tier returns the current execution tier for this function.
func (p *FnProfile) Tier() uint32 { return p.tier.Load() }

// RecordAndCheck records a call with its duration in nanoseconds.
func (p *FnProfile) RecordAndCheck(durationNs int64) bool {
	p.calls.Add(1)
	p.totalNs.Add(durationNs)
	return true
}

// Record is an alias for RecordAndCheck; the argSig parameter is ignored.
func (p *FnProfile) Record(durationNs int64, _ string) bool {
	return p.RecordAndCheck(durationNs)
}

// TotalCalls returns the total number of recorded calls.
func (p *FnProfile) TotalCalls() int64 { return p.calls.Load() }

// AvgNs returns the average call duration in nanoseconds.
func (p *FnProfile) AvgNs() float64 {
	c := p.calls.Load()
	if c == 0 {
		return 0
	}
	return float64(p.totalNs.Load()) / float64(c)
}

func (p *FnProfile) promoteToTier(newTier uint32) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if newTier > p.tier.Load() {
		p.tier.Store(newTier)
		return true
	}
	return false
}

// PromoteToFastGo promotes this function to the fast-Go execution tier.
func (p *FnProfile) PromoteToFastGo() bool { return p.promoteToTier(TierFastGo) }

// ShouldSample returns true every 32nd call to amortize sampling overhead.
func (p *FnProfile) ShouldSample() bool { return p.calls.Load()&31 == 0 }

// TryLoadFromDiskCache is a no-op stub.
// The JIT cache is managed inside the Zig runtime subprocess.
func (p *FnProfile) TryLoadFromDiskCache(_ string) bool { return false }

// Profiler tracks per-function call statistics for the Go-side interpreter.
type Profiler struct {
	profiles sync.Map
	start    time.Time
}

// NewProfiler creates a new Profiler.
func NewProfiler(_ bool) *Profiler {
	return &Profiler{start: time.Now()}
}

// GetOrCreate returns the FnProfile for the named function, creating it if
// it does not yet exist.
func (p *Profiler) GetOrCreate(name string) *FnProfile {
	if v, ok := p.profiles.Load(name); ok {
		return v.(*FnProfile)
	}
	np := NewProfile(name)
	actual, _ := p.profiles.LoadOrStore(name, np)
	return actual.(*FnProfile)
}

// Record records a call to the named function and promotes it to FastGo.
func (p *Profiler) Record(name string, durationNs int64, _ string) {
	prof := p.GetOrCreate(name)
	prof.RecordAndCheck(durationNs)
	prof.PromoteToFastGo()
}

// RecordAndCheckHot records a call and returns true (every call is accepted).
func (p *Profiler) RecordAndCheckHot(name string, durationNs int64) bool {
	prof := p.GetOrCreate(name)
	return prof.RecordAndCheck(durationNs)
}

// GetProfile returns the profile for the named function, if it exists.
func (p *Profiler) GetProfile(name string) (*FnProfile, bool) {
	v, ok := p.profiles.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*FnProfile), true
}

// IsHot returns true if the named function has a recorded profile.
func (p *Profiler) IsHot(name string) bool {
	_, ok := p.GetProfile(name)
	return ok
}

// Report returns a formatted profiling summary table.
func (p *Profiler) Report() string {
	var sb strings.Builder
	elapsed := time.Since(p.start)
	sb.WriteString(fmt.Sprintf(
		"\n\x1b[1m  Lunex JIT Profile\x1b[0m  \x1b[90m(runtime: %.2fs | arch: %s | jit: zig-native)\x1b[0m\n",
		elapsed.Seconds(), runtime.GOARCH,
	))
	sb.WriteString(fmt.Sprintf("  %-28s %-8s %-12s %-10s\n", "Symbol", "Calls", "Avg µs", "Tier"))
	sb.WriteString(strings.Repeat("\x1b[90m─\x1b[0m", 64) + "\n")
	p.profiles.Range(func(_, value any) bool {
		prof := value.(*FnProfile)
		c := prof.TotalCalls()
		if c == 0 {
			return true
		}
		avgUs := prof.AvgNs() / 1e3
		tier := tierLabel[min3(prof.Tier(), 2)]
		sb.WriteString(fmt.Sprintf("  %-28s %-8d %-12.2f %-10s\n",
			trunc(prof.Name, 27), c, avgUs, tier))
		return true
	})
	return sb.String()
}

func min3(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

func fnCacheKey(name, sourceText string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(sourceText))
	return hex.EncodeToString(h.Sum(nil))
}

// ExecCountSumNative counts from start to limit and returns (finalCounter, sum).
// Delegates to the pure-Go implementation; machine-code generation is handled
// // by the Zig JIT inside lunex-rt.
func ExecCountSumNative(start, limit int64) (int64, int64) {
	return native.RunCountSum(start, limit)
}

// ExecCountNative counts from start to limit and returns the final counter.
func ExecCountNative(start, limit int64) int64 {
	return native.RunCount(start, limit)
}

// ExecFib advances a Fibonacci sequence by count steps and returns (a, b).
func ExecFib(a, b, count int64) (int64, int64) {
	return native.RunFib(a, b, count)
}

// ExecCountAccum counts from start to limit by step, accumulating delta per
// step, and returns (finalCounter, finalAccum).
func ExecCountAccum(start, limit, step, accumStart, delta int64) (int64, int64) {
	return native.RunCountAccum(start, limit, step, accumStart, delta)
}
