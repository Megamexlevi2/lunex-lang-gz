// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package debug

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var enabled = os.Getenv("NTL_DEBUG") == "1"

func Enabled() bool { return enabled }

func Enable() { enabled = true; os.Setenv("NTL_DEBUG", "1") }

type Timer struct {
	label string
	start time.Time
}

func Start(label string) *Timer {
	if enabled {
		fmt.Fprintf(os.Stderr, "\x1b[90m  [dbg] %-22s ...\x1b[0m\n", label)
	}
	return &Timer{label: label, start: time.Now()}
}

func (t *Timer) Done() time.Duration {
	d := time.Since(t.start)
	if enabled {
		fmt.Fprintf(os.Stderr, "\x1b[90m  [dbg] %-22s \x1b[32m%v\x1b[0m\n", t.label, d)
	}
	return d
}

func Log(format string, args ...any) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "\x1b[90m  [dbg] "+format+"\x1b[0m\n", args...)
}

func Section(title string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "\x1b[90m  [dbg] ── %s\x1b[0m\n", title)
}

func MemStats() {
	if !enabled {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(os.Stderr,
		"\x1b[90m  [dbg] memory: alloc=%dKB  sys=%dKB  gc=%d  goroutines=%d\x1b[0m\n",
		m.Alloc/1024, m.Sys/1024, m.NumGC, runtime.NumGoroutine(),
	)
}

func Header(file, version, arch string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "\x1b[1m\x1b[36m  *debug Lunex %s [%s]\x1b[0m\n", version, arch)
	fmt.Fprintf(os.Stderr, "\x1b[90m  file: %s\x1b[0m\n", file)
	fmt.Fprintf(os.Stderr, "\x1b[90m  ────────────────────────────────\x1b[0m\n")
}

func Footer(total time.Duration) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "\x1b[90m  ────────────────────────────────\x1b[0m\n")
	fmt.Fprintf(os.Stderr, "\x1b[90m  [dbg] total: \x1b[32m%v\x1b[0m\n", total)
	MemStats()
}
