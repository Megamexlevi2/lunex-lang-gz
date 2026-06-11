// Lunex lang — structured diagnostic output for the toolchain.
//
// Everything in this package goes to stderr so it never pollutes program output.
//
// How to enable:
//
//   lunex --debug run test.lx            — standard debug (every execution step)
//   lunex -d run test.lx                 — short form of --debug
//   NTL_DEBUG=1 ./lunex run test.lx      — same via environment variable
//   LUNEX_DEBUG=1 ./lunex run test.lx    — alternate env var name
//   lunex --verbose run test.lx          — verbose mode (even more detail)
//   LUNEX_VERBOSE=1 ./lunex run test.lx  — verbose via environment variable
//
// What you see in debug mode:
//
//   - Exactly where the file came from and how big it is
//   - Whether the cache was hit or missed
//   - How long each compile phase took
//   - How many bytes of bytecode Go produced
//   - Fast-Go path activations for recognized hot loop patterns
//   - Memory allocation totals and GC count at the end
//
// Architecture note:
//   Go interpreter handles ALL Lunex execution.
//   Pure-Go fast paths accelerate recognized numeric loop patterns.

package debug

import (
	"fmt"
	"lunex/internal/meta"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func envEnabled(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on", "debug":
		return true
	default:
		return false
	}
}

var enabled = envEnabled("NTL_DEBUG") || envEnabled("LUNEX_DEBUG") || envEnabled("DEBUG")
var verbose = envEnabled("LUNEX_VERBOSE")

func Enabled() bool { return enabled }
func Verbose() bool { return verbose }

func Enable() {
	enabled = true
	_ = os.Setenv("NTL_DEBUG", "1")
	_ = os.Setenv("LUNEX_DEBUG", "1")
}

func EnableVerbose() {
	verbose = true
	_ = os.Setenv("LUNEX_VERBOSE", "1")
}

// ─── ANSI colour codes ────────────────────────────────────────────────────────

const (
	cReset  = "\x1b[0m"
	cBold   = "\x1b[1m"
	cDim    = "\x1b[90m"
	cCyan   = "\x1b[36m"
	cGreen  = "\x1b[32m"
	cYellow = "\x1b[33m"
	cBlue   = "\x1b[34m"
	cMag    = "\x1b[35m"
	cWhite  = "\x1b[97m"
	cRed    = "\x1b[31m"
)

func stamp() string { return time.Now().Format("15:04:05.000") }

// ─── Timer ────────────────────────────────────────────────────────────────────

// Timer measures elapsed time for a labelled operation.
type Timer struct {
	label string
	start time.Time
}

// Start begins a named timer and logs that it started (debug mode only).
func Start(label string) *Timer {
	if enabled {
		fmt.Fprintf(os.Stderr, "%s  [dbg] %-28s %sstarted%s\n", cDim, label, cCyan, cReset)
	}
	return &Timer{label: label, start: time.Now()}
}

// Done stops the timer, logs the elapsed time, and returns it.
func (t *Timer) Done() time.Duration {
	d := time.Since(t.start)
	if enabled {
		fmt.Fprintf(os.Stderr, "%s  [dbg] %-28s %s%s%s\n", cDim, t.label, cGreen, d.Round(time.Microsecond), cReset)
	}
	return d
}

// ─── Basic log functions ──────────────────────────────────────────────────────

// Log writes a timestamped message to stderr (debug mode only).
func Log(format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  [dbg] %s%s  %s%s\n",
		cDim, cCyan, stamp(), cReset+fmt.Sprintf(format, args...), cReset)
}

// Section prints a separator with a title — use this to mark major phases.
func Section(title string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  -- %s%s%s\n", cDim, cWhite, title, cReset)
}

// ─── Header / Footer ─────────────────────────────────────────────────────────

// Header prints the debug banner at the start of a run.
func Header(file string) {
	if !enabled {
		return
	}
	v := meta.FullVersion()
	exe := "(unknown)"
	if path, err := os.Executable(); err == nil && path != "" {
		exe = filepath.Base(path)
	}
	fmt.Fprintf(os.Stderr, "\n%slunex %s%s  %s(debug mode on)%s\n", cBold, v, cReset, cCyan, cReset)
	fmt.Fprintf(os.Stderr, "%s  target file   %s%s%s\n", cDim, cWhite, file, cReset)
	fmt.Fprintf(os.Stderr, "%s  executable    %s%s%s\n", cDim, cWhite, exe, cReset)
	fmt.Fprintf(os.Stderr, "%s  platform      %s/%s%s\n", cDim, runtime.GOOS, runtime.GOARCH, cReset)
	fmt.Fprintf(os.Stderr, "%s  go version    %s%s%s\n", cDim, cWhite, runtime.Version(), cReset)
	fmt.Fprintf(os.Stderr, "%s  flags         debug=%t verbose=%t%s\n", cDim, enabled, verbose, cReset)
	fmt.Fprintf(os.Stderr, "%s  LUNEX_DEBUG=1 — you will see every step of the execution flow%s\n", cDim, cReset)
	fmt.Fprintf(os.Stderr, "%s  Go interpreter: active  fast-Go paths: enabled%s\n", cDim, cReset)
	fmt.Fprintf(os.Stderr, "%s  ----------------------------------------%s\n", cDim, cReset)
}

// Footer prints a summary at the end of a run.
func Footer(total time.Duration) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  ----------------------------------------%s\n", cDim, cReset)
	fmt.Fprintf(os.Stderr, "%s  total wall time  %s%s%s\n", cDim, cGreen, total.Round(time.Microsecond), cReset)
	MemStats()
	fmt.Fprintf(os.Stderr, "\n")
}

// ─── Pipeline steps ───────────────────────────────────────────────────────────

// Step prints a single execution step (what we are about to do).
func Step(label, detail string) {
	if !enabled {
		return
	}
	if detail == "" {
		fmt.Fprintf(os.Stderr, "%s  %s→%s  %s\n", cDim, cCyan, cReset, label)
	} else {
		fmt.Fprintf(os.Stderr, "%s  %s→%s  %-32s%s%s%s\n", cDim, cCyan, cReset, label, cDim, detail, cReset)
	}
}

// StepOK logs a successful step outcome.
func StepOK(tag, label, detail string) {
	if !enabled {
		return
	}
	if detail == "" {
		fmt.Fprintf(os.Stderr, "%s  %s%-4s%s  %s\n", cDim, cGreen, tag, cReset, label)
	} else {
		fmt.Fprintf(os.Stderr, "%s  %s%-4s%s  %-30s%s%s%s\n", cDim, cGreen, tag, cReset, label, cDim, detail, cReset)
	}
}

// StepWarn logs a step that completed with a non-fatal warning.
func StepWarn(label, detail string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  %s!%s   %-31s%s%s%s\n", cDim, cYellow, cReset, label, cDim, detail, cReset)
}

// StepFail logs a step that failed.
func StepFail(label, detail string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  %s✗%s   %-31s%s%s%s\n", cDim, cRed, cReset, label, cDim, detail, cReset)
}

// ─── JIT diagnostics (no-op stubs kept for API compat) ───────────────────────

func BridgeSend(_ string, _ uint16, _ int)          {}
func BridgeRecv(_ string, _ uint16, _ uint8, _ int) {}
func BridgeError(_ string)                           {}
func BridgeJITRequest(_ int)                         {}
func BridgeJITResult(_ int)                          {}
func BridgeZigStarted(_ string)                      {}
func BridgeZigFallback(_ string)                     {}
func BridgeZigSuccess(_ int, _ time.Duration)        {}

// BytecodeSection logs details about the bytecode container Go produced.
func BytecodeSection(totalBytes, ntzBytes int, hasNTZ bool) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "%s  %s[bc]%s  Go produced .nc container: %d bytes%s\n",
		cDim, cYellow, cReset, totalBytes, cReset)
	if hasNTZ && ntzBytes > 0 {
		fmt.Fprintf(os.Stderr, "%s         ├─ NTZ section: %d bytes%s\n", cDim, ntzBytes, cReset)
	} else {
		fmt.Fprintf(os.Stderr, "%s         ├─ NTZ section: absent%s\n", cDim, cReset)
	}
	fmt.Fprintf(os.Stderr, "%s         └─ Go interpreter executes source text; fast-Go loop optimizations active%s\n",
		cDim, cReset)
}

// BytecodeNaxerHit logs that the Zig/Naxer engine JIT-compiled a hot function.
func BytecodeNaxerHit(fnName string, codeBytes int, report string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr,
		"%s  %s[naxer]%s compiled hot fn %s%q%s  code=%d B  %s%s%s\n",
		cDim, cGreen, cReset, cCyan, fnName, cReset, codeBytes, cDim, report, cReset)
}

// ─── Memory ───────────────────────────────────────────────────────────────────

// MemStats prints current heap allocation and GC stats.
func MemStats() {
	if !enabled {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(os.Stderr,
		"%s  memory  %s%d KB%s alloc   gc %s%d%s   goroutines %s%d%s\n",
		cDim, cCyan, m.Alloc/1024, cDim, cCyan, m.NumGC, cDim, cCyan, runtime.NumGoroutine(), cReset,
	)
}

// ─── Verbose helpers ──────────────────────────────────────────────────────────

// V writes a verbose-only timestamped message.
func V(format string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "%s%s%s  %s\n", cDim, stamp(), cReset, fmt.Sprintf(format, args...))
}

// VSection prints a verbose section heading.
func VSection(title string) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s%s%s\n", cCyan, title, cReset)
}

// VStep prints a verbose step detail.
func VStep(label string, args ...any) {
	if !verbose {
		return
	}
	suffix := ""
	if len(args) > 0 {
		parts := make([]string, 0, len(args))
		for _, a := range args {
			parts = append(parts, fmt.Sprintf("%v", a))
		}
		suffix = "  " + cDim + strings.Join(parts, "  ") + cReset
	}
	fmt.Fprintf(os.Stderr, "  %s>%s  %s%s\n", cCyan, cReset, label, suffix)
}

// VKV prints a verbose key-value pair.
func VKV(key string, val any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "  %s.%s  %-24s  %s%v%s\n", cDim, cReset, key, cWhite, val, cReset)
}

// VHeader prints the verbose startup banner.
func VHeader(file string) {
	if !verbose {
		return
	}
	v := meta.FullVersion()
	fmt.Fprintf(os.Stderr, "\n%slunex %s%s\n", cBold, v, cReset)
	fmt.Fprintf(os.Stderr, "%s  file     %s%s\n", cDim, cWhite, file)
	fmt.Fprintf(os.Stderr, "%s  pid      %d\n", cDim, os.Getpid())
	fmt.Fprintf(os.Stderr, "  os/arch  %s/%s%s\n\n", runtime.GOOS, runtime.GOARCH, cReset)
}

// VFooter prints the verbose completion summary.
func VFooter(total time.Duration) {
	if !verbose {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(os.Stderr, "\n%s  done in  %s%s%s\n", cDim, cGreen, total.Round(time.Microsecond), cReset)
	fmt.Fprintf(os.Stderr, "%s  memory   %s%d KB%s   gc %s%d%s\n\n", cDim, cCyan, m.Alloc/1024, cDim, cCyan, m.NumGC, cReset)
}

// ─── Command stubs ────────────────────────────────────────────────────────────

// CommandStub is called from command handlers that are not yet fully implemented.
func CommandStub(cmd, description string) bool {
	if !enabled {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s  %s[stub]%s %-20s %s(debug mode — not fully implemented)%s\n",
		cDim, cYellow, cReset, cmd, cDim, cReset)
	if description != "" {
		fmt.Fprintf(os.Stderr, "%s         → %s%s\n", cDim, description, cReset)
	}
	return true
}

// CommandDebugHeader prints a banner when a command starts in debug mode.
func CommandDebugHeader(cmd string, args []string) {
	if !enabled {
		return
	}
	argsStr := strings.Join(args, " ")
	if argsStr == "" {
		argsStr = "(no args)"
	}
	fmt.Fprintf(os.Stderr, "%s  %s[cmd]%s  lunex %s  %s%s%s\n",
		cDim, cBlue, cReset, cmd, cDim, argsStr, cReset)
}
