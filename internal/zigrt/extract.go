// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package zigrt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	rtBin     []byte
	rtBinOnce sync.Once
	rtBinPath string
	rtBinErr  error
)

// Init stores the embedded Zig binary. Call this from main's init().
func Init(data []byte) {
	rtBin = data
}

// ExtractOnce pulls the embedded binary onto disk the first time it's called.
// After that it just returns the cached path — no re-writing needed.
func ExtractOnce() (string, error) {
	rtBinOnce.Do(func() {
		rtBinPath, rtBinErr = extract(rtBin)
	})
	return rtBinPath, rtBinErr
}

func extract(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("lunex: no embedded Zig runtime — was the binary built correctly?")
	}

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:8])

	binName := "lunex-rt"
	if runtime.GOOS == "windows" {
		binName = "lunex-rt.exe"
	}

	for _, dir := range candidateDirs(hash) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}
		binPath := filepath.Join(dir, binName)

		// Already there and identical — verify it is still runnable.
		if existing, err := os.ReadFile(binPath); err == nil {
			if sha256.Sum256(existing) == sum && canExec(binPath) {
				return binPath, nil
			}
		}

		// Write binary.
		if err := os.WriteFile(binPath, data, 0755); err != nil {
			continue
		}

		// Explicit chmod in case umask stripped execute bits.
		if runtime.GOOS != "windows" {
			_ = os.Chmod(binPath, 0755)
		}

		if canExec(binPath) {
			return binPath, nil
		}

		// Filesystem is noexec — remove and try the next location.
		_ = os.Remove(binPath)
	}

	return "", fmt.Errorf(
		"lunex: could not install the Zig runtime in any writable+executable location; " +
			"set LUNEX_RT_DIR to a directory on an exec-capable filesystem",
	)
}

// candidateDirs returns directories to store the Zig runtime binary,
// ordered from most to least preferred.  Covers Linux, macOS, Windows,
// and Android / Termux.
func candidateDirs(hash string) []string {
	sub := filepath.Join("lunex", "rt-"+hash)
	var dirs []string

	// LUNEX_RT_DIR override — always tried first.
	if override := os.Getenv("LUNEX_RT_DIR"); override != "" {
		dirs = append(dirs, filepath.Join(override, "rt-"+hash))
	}

	if home, err := os.UserHomeDir(); err == nil {
		// XDG standard — works on Linux, macOS, Termux.
		dirs = append(dirs, filepath.Join(home, ".local", "share", sub))
		// Simple dot-directory fallback.
		dirs = append(dirs, filepath.Join(home, ".lx", "rt-"+hash))
	}

	// ~/.cache on Linux, ~/Library/Caches on macOS, %LocalAppData% on Windows.
	if cacheDir, err := os.UserCacheDir(); err == nil {
		dirs = append(dirs, filepath.Join(cacheDir, sub))
	}

	// Termux: $PREFIX/{var/cache,tmp} are always on an exec-capable FS.
	if prefix := os.Getenv("PREFIX"); prefix != "" {
		dirs = append(dirs,
			filepath.Join(prefix, "var", "cache", sub),
			filepath.Join(prefix, "tmp", sub),
		)
	}

	// System temp dir — last resort.
	dirs = append(dirs, filepath.Join(os.TempDir(), sub))

	return dirs
}

// canExec reports whether the binary at path can actually be executed.
// It checks the execute bit on Unix and probes the binary on all platforms
// to catch noexec mounts and architecturally incompatible binaries early.
func canExec(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		return false
	}
	// Run the binary with an unknown flag; any outcome except "exec failed"
	// means the OS was willing to launch it.
	cmd := exec.Command(path, "--probe")
	if err := cmd.Run(); err == nil {
		return true
	} else if _, ok := err.(*exec.ExitError); ok {
		return true
	}
	return false
}
