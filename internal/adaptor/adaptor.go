// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package adaptor provides platform detection, directory resolution, and
// cache management for the Lunex compiler. It is designed to operate
// correctly on Linux, macOS, Windows, FreeBSD, and Android/Termux, even
// when every writable directory on disk is blocked. In that case all cache
// data is held exclusively in the process-local in-memory store.
package adaptor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ─── Platform detection ────────────────────────────────────────────────────

// Platform identifies the host operating system with extra Termux awareness.
type Platform uint8

const (
	PlatformLinux   Platform = iota // standard Linux (not Android)
	PlatformMacOS                   // darwin
	PlatformWindows                 // windows
	PlatformFreeBSD                 // freebsd
	PlatformAndroid                 // Android bare (GOOS == "android")
	PlatformTermux                  // Android + $PREFIX set → Termux
	PlatformWASM                    // js/wasm – filesystem is virtual
	PlatformUnknown                 // any other GOOS
)

// Current is the Platform of the running process, resolved once at startup.
var Current = detectPlatform()

func detectPlatform() Platform {
	switch runtime.GOOS {
	case "linux":
		// Termux sets $PREFIX to its package root (e.g. /data/data/…/files/usr).
		if os.Getenv("PREFIX") != "" {
			return PlatformTermux
		}
		// Google's Android runtime sets $ANDROID_ROOT or $ANDROID_DATA.
		if os.Getenv("ANDROID_ROOT") != "" || os.Getenv("ANDROID_DATA") != "" {
			return PlatformAndroid
		}
		return PlatformLinux
	case "android":
		if os.Getenv("PREFIX") != "" {
			return PlatformTermux
		}
		return PlatformAndroid
	case "darwin":
		return PlatformMacOS
	case "windows":
		return PlatformWindows
	case "freebsd":
		return PlatformFreeBSD
	case "js":
		return PlatformWASM
	default:
		return PlatformUnknown
	}
}

// String returns a human-readable platform label used in diagnostics.
func (p Platform) String() string {
	switch p {
	case PlatformLinux:
		return "linux"
	case PlatformMacOS:
		return "macos"
	case PlatformWindows:
		return "windows"
	case PlatformFreeBSD:
		return "freebsd"
	case PlatformAndroid:
		return "android"
	case PlatformTermux:
		return "android/termux"
	case PlatformWASM:
		return "wasm"
	default:
		return "unknown"
	}
}

// IsAndroidLike returns true for both bare Android and Termux.
func IsAndroidLike() bool {
	return Current == PlatformAndroid || Current == PlatformTermux
}

// IsUnix returns true for all POSIX-like platforms (Linux, macOS, FreeBSD,
// Android, Termux). Windows and WASM return false.
func IsUnix() bool {
	switch Current {
	case PlatformLinux, PlatformMacOS, PlatformFreeBSD,
		PlatformAndroid, PlatformTermux:
		return true
	}
	return false
}

// ─── Root data-directory candidates ───────────────────────────────────────

// cwdCacheDir is the name of the local cache directory created inside the
// current working directory when no other writable location is found first.
const cwdCacheDir = "lunex-cache"

// DataDirCandidates returns an ordered list of directories that Lunex may
// use as its root data store (for the .lx tree). The list is ordered from
// most to least desirable; callers should walk it and use the first one
// for which EnsureWritable returns true.
//
// Priority order:
//  1. LUNEX_DATA_DIR env override — skips all other candidates.
//  2. Termux $PREFIX/var/cache/lunex — always exec-capable.
//  3. ~/.local/share/lunex (XDG_DATA_HOME) and ~/.lx (classic).
//  4. OS cache directory (platform-specific, see os.UserCacheDir).
//  5. Windows %AppData% / %LocalAppData%.
//  6. macOS ~/Library/Application Support/lunex.
//  7. ./lunex-cache (current working directory) — opt-in via
//     LUNEX_USE_CWD_CACHE=1, or automatically when all above fail.
//
// Temporary directories are never included. When no candidate is writable,
// callers must fall back to the in-memory cache.
func DataDirCandidates() []string {
	if override := os.Getenv("LUNEX_DATA_DIR"); override != "" {
		return []string{override}
	}

	var candidates []string

	// Termux: $PREFIX is always on an exec-capable filesystem.
	if prefix := os.Getenv("PREFIX"); prefix != "" {
		candidates = append(candidates,
			filepath.Join(prefix, "var", "cache", "lunex"),
		)
	}

	// XDG / home-based directories.
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "lunex"), // XDG_DATA_HOME
			filepath.Join(home, ".lx"),                      // classic lunex dir
		)
	}

	// OS cache directory.
	// os.UserCacheDir returns:
	//   Linux:   ~/.cache
	//   macOS:   ~/Library/Caches
	//   Windows: %LocalAppData%
	if cacheDir, err := os.UserCacheDir(); err == nil {
		candidates = append(candidates, filepath.Join(cacheDir, "lunex"))
	}

	// Windows-specific %AppData% directories.
	if Current == PlatformWindows {
		if appData := os.Getenv("APPDATA"); appData != "" {
			candidates = append(candidates, filepath.Join(appData, "lunex"))
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			candidates = append(candidates, filepath.Join(localAppData, "lunex"))
		}
	}

	// macOS: ~/Library/Application Support.
	if Current == PlatformMacOS {
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates,
				filepath.Join(home, "Library", "Application Support", "lunex"),
			)
		}
	}

	// Current working directory — ./lunex-cache.
	// Included when LUNEX_USE_CWD_CACHE=1 is set (explicit opt-in) or
	// always appended last so it acts as the final disk fallback before
	// the process falls through to the in-memory cache.
	if cwd, err := os.Getwd(); err == nil {
		cwdCache := filepath.Join(cwd, cwdCacheDir)
		if os.Getenv("LUNEX_USE_CWD_CACHE") == "1" {
			// Opt-in: prepend so it wins over all other candidates.
			candidates = append([]string{cwdCache}, candidates...)
		} else {
			// Always append as last-resort disk fallback.
			candidates = append(candidates, cwdCache)
		}
	}

	return candidates
}

// CWDCacheDir returns the absolute path of the ./lunex-cache directory inside
// the current working directory, regardless of whether it exists or is
// writable. Returns an empty string when os.Getwd() fails.
func CWDCacheDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, cwdCacheDir)
}

// UseCWDCache reports whether the current working directory cache is the
// active preferred location (i.e. LUNEX_USE_CWD_CACHE=1 is set).
func UseCWDCache() bool {
	return os.Getenv("LUNEX_USE_CWD_CACHE") == "1"
}

// ─── Sub-directory helpers ─────────────────────────────────────────────────

// CacheDir returns a writable path for bytecode cache files (.nc).
// Returns an empty string when no writable disk location is available;
// callers must use the in-memory cache in that case.
func CacheDir() string {
	return subDir("cache")
}

// JITCacheDir returns a writable path for JIT stub files (.bin / .mcx).
// Returns an empty string when no writable disk location is available.
func JITCacheDir() string {
	return subDir(filepath.Join("jit", runtime.GOOS+"_"+runtime.GOARCH))
}

// NativeCacheDir returns a writable path for native machine-code stubs.
// Returns an empty string when no writable disk location is available.
func NativeCacheDir() string {
	return subDir(filepath.Join("ncache", runtime.GOOS+"_"+runtime.GOARCH))
}

// ModuleDir returns a writable path for a downloaded package (name@version).
// Returns an empty string when no writable disk location is available.
func ModuleDir(name, version string) string {
	if version == "" {
		version = "main"
	}
	safe := strings.ReplaceAll(name, "/", "__")
	return subDir(filepath.Join("modules", safe+"@"+version))
}

// RuntimeDir returns the directory where lunex-rt should be extracted, keyed
// by the binary hash so different builds never collide.
// Returns an empty string when no exec-capable disk location is available.
func RuntimeDir(hash string) string {
	return subDirExec("rt-" + hash)
}

// MarkerPath returns the path to the first-run sentinel file.
// Returns ("", false) when no writable disk location is available.
func MarkerPath() (string, bool) {
	base := subDir("")
	if base == "" {
		return "", false
	}
	return filepath.Join(base, ".initialized"), true
}

// subDir returns the first resolvable writable path for sub within the Lunex
// data tree. Returns an empty string when every candidate is unwritable; the
// caller must then fall back to the in-memory cache. Temporary directories
// are never used.
func subDir(sub string) string {
	for _, base := range DataDirCandidates() {
		dir := base
		if sub != "" {
			dir = filepath.Join(base, sub)
		}
		if EnsureWritable(dir) {
			return dir
		}
	}
	// All disk candidates failed. Signal the caller to use memory only.
	return ""
}

// subDirExec is like subDir but also requires exec permission on the
// directory's filesystem (needed for executable binaries). Returns an
// empty string when no exec-capable location is available.
func subDirExec(sub string) string {
	for _, base := range DataDirCandidates() {
		dir := filepath.Join(base, sub)
		if EnsureWritable(dir) && FSSupportsExec(dir) {
			return dir
		}
	}
	// All exec-capable candidates failed.
	return ""
}

// ─── Runtime binary extraction ────────────────────────────────────────────

// RuntimeCandidateDirs returns every directory that may hold the lunex-rt
// binary, ordered from most to least preferred.
//
// Priority:
//  1. LUNEX_RT_DIR env override.
//  2. Termux $PREFIX — always exec-capable.
//  3. XDG / home fallbacks.
//  4. OS cache dir.
//  5. ./lunex-cache (current working directory) — last disk resort.
//
// Temporary directories are never included. When every candidate lacks exec
// capability, the caller must report that runtime extraction is unavailable.
func RuntimeCandidateDirs(hash string) []string {
	sub := filepath.Join("lunex", "rt-"+hash)
	var dirs []string

	if override := os.Getenv("LUNEX_RT_DIR"); override != "" {
		dirs = append(dirs, filepath.Join(override, "rt-"+hash))
	}

	if prefix := os.Getenv("PREFIX"); prefix != "" {
		dirs = append(dirs,
			filepath.Join(prefix, "var", "cache", sub),
		)
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			filepath.Join(home, ".local", "share", sub),
			filepath.Join(home, ".lx", "rt-"+hash),
		)
	}

	if cacheDir, err := os.UserCacheDir(); err == nil {
		dirs = append(dirs, filepath.Join(cacheDir, sub))
	}

	if Current == PlatformWindows {
		if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
			dirs = append(dirs, filepath.Join(appData, sub))
		}
	}

	// Current working directory — ./lunex-cache — last disk resort.
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, cwdCacheDir, "rt-"+hash))
	}

	return dirs
}

// ─── Filesystem capability probes ─────────────────────────────────────────

// EnsureWritable creates dir (and all parents) if needed and confirms
// that a file can actually be created inside it. Returns false on any
// failure without logging. Never falls back to a temp directory.
func EnsureWritable(dir string) bool {
	if dir == "" {
		return false
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	probe := filepath.Join(dir, ".lunex_probe_"+fmt.Sprintf("%d", os.Getpid()))
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(probe)
	return true
}

// FSSupportsExec reports whether the filesystem that contains dir allows
// execute permission on files. It writes a tiny probe shell script, sets
// the exec bit, and tries to launch it.
//
// On Windows execute permission is always implicit, so this returns true
// immediately. On WASM there is no filesystem exec, so this returns false.
func FSSupportsExec(dir string) bool {
	if Current == PlatformWindows {
		return true
	}
	if Current == PlatformWASM {
		return false
	}
	if dir == "" {
		return false
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}

	probe := filepath.Join(dir, ".lunex_exec_probe")
	// A minimal POSIX shell script that exits immediately. This works on all
	// Unix platforms (Linux, macOS, FreeBSD, Android/Termux) without needing
	// architecture-specific ELF/Mach-O bytes.
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(probe, []byte(script), 0755); err != nil {
		return false
	}
	defer os.Remove(probe)

	if err := os.Chmod(probe, 0755); err != nil {
		return false
	}

	cmd := exec.Command(probe)
	err := cmd.Run()
	if err == nil {
		return true
	}
	if _, ok := err.(*exec.ExitError); ok {
		return true // process ran but exited non-zero — exec still worked
	}
	return false
}

// ─── In-memory bytecode cache (no files needed) ───────────────────────────

// memCache stores compiled .nc bytecode keyed by a hash of the source path +
// mtime + size. This lets Lunex run even when every writable directory is on
// a noexec or read-only filesystem, and without ever touching a temp dir.
var memCache sync.Map // key: string → value: []byte

// MemCacheKey returns the in-memory cache key for a source file. The key
// incorporates the absolute path, modification time, and file size so stale
// entries are never returned after a file is updated.
func MemCacheKey(absPath string) (string, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		// Source is not a real file (e.g. eval string). Hash the path itself.
		h := sha256.Sum256([]byte(absPath))
		return hex.EncodeToString(h[:16]), nil
	}
	h := sha256.New()
	h.Write([]byte(absPath))
	fmt.Fprintf(h, "%d", info.ModTime().UnixNano())
	fmt.Fprintf(h, "%d", info.Size())
	return hex.EncodeToString(h.Sum(nil)[:16]), nil
}

// MemCacheGet retrieves previously compiled bytecode from the in-process
// cache. Returns (nil, false) on a miss.
func MemCacheGet(key string) ([]byte, bool) {
	v, ok := memCache.Load(key)
	if !ok {
		return nil, false
	}
	return v.([]byte), true
}

// MemCacheSet stores bytecode in the in-process cache under key.
func MemCacheSet(key string, data []byte) {
	if len(data) == 0 {
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	memCache.Store(key, cp)
}

// MemCacheDelete removes one entry from the in-process cache.
func MemCacheDelete(key string) {
	memCache.Delete(key)
}

// MemCacheStats returns the number of entries and total bytes held in the
// in-process cache.
func MemCacheStats() (count int, totalBytes int64) {
	memCache.Range(func(_, v any) bool {
		count++
		totalBytes += int64(len(v.([]byte)))
		return true
	})
	return
}

// MemCacheClear flushes the entire in-process cache.
func MemCacheClear() {
	memCache.Range(func(k, _ any) bool {
		memCache.Delete(k)
		return true
	})
}

// ─── Unified cache lookup (disk → memory) ─────────────────────────────────

// CacheLookup checks the disk cache first, then the in-memory cache.
// Returns the bytecode and true on a hit, or nil/false on a miss.
// Works correctly even when CacheDir() returns "".
func CacheLookup(absPath string) ([]byte, bool) {
	key, err := MemCacheKey(absPath)
	if err != nil {
		return nil, false
	}

	// 1. Try disk (only when a writable cache dir is available).
	if dir := CacheDir(); dir != "" {
		diskPath := filepath.Join(dir, key+".nc")
		if data, err := os.ReadFile(diskPath); err == nil && len(data) > 0 {
			return data, true
		}
	}

	// 2. Fall back to in-memory cache.
	return MemCacheGet(key)
}

// CacheStore writes bytecode to the disk cache when possible, and always
// writes it to the in-memory cache. A disk failure is silently ignored so
// execution is never blocked. When disk is unavailable, memory is the sole
// store and the data survives for the lifetime of the process.
func CacheStore(absPath string, ncData []byte) {
	key, err := MemCacheKey(absPath)
	if err != nil || len(ncData) == 0 {
		return
	}

	// Always populate in-memory cache first.
	MemCacheSet(key, ncData)

	// Best-effort disk write — skip silently when no writable dir exists.
	dir := CacheDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	diskPath := filepath.Join(dir, key+".nc")
	_ = os.WriteFile(diskPath, ncData, 0600)
}

// CacheInvalidate removes a source file's entry from both disk and memory.
func CacheInvalidate(absPath string) {
	key, err := MemCacheKey(absPath)
	if err != nil {
		return
	}
	MemCacheDelete(key)
	if dir := CacheDir(); dir != "" {
		_ = os.Remove(filepath.Join(dir, key+".nc"))
	}
}

// ─── Runtime execution helpers ────────────────────────────────────────────

// CanExecBinary reports whether the binary at path can actually be launched.
func CanExecBinary(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if Current != PlatformWindows && info.Mode()&0111 == 0 {
		return false
	}
	cmd := exec.Command(path, "--probe")
	if err := cmd.Run(); err == nil {
		return true
	} else if _, ok := err.(*exec.ExitError); ok {
		return true
	}
	return false
}

// SetExecBit attempts to set the executable bit on path. A no-op on Windows.
func SetExecBit(path string) error {
	if Current == PlatformWindows {
		return nil
	}
	return os.Chmod(path, 0755)
}

// BinaryName returns the platform-correct name for the Lunex runtime binary.
func BinaryName() string {
	if Current == PlatformWindows {
		return "lunex-rt.exe"
	}
	return "lunex-rt"
}

// ─── Path utilities ────────────────────────────────────────────────────────

// ShortenHome replaces the user's home directory prefix with "~" for
// display purposes.
func ShortenHome(path string) string {
	if path == "" {
		return "(memory only)"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// CleanStale removes every rt-<otherhash> directory that sits next to
// currentDir so old runtime versions do not accumulate.
func CleanStale(currentDir, currentHash string) {
	if currentDir == "" {
		return
	}
	parent := filepath.Dir(currentDir)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "rt-") {
			continue
		}
		if name == "rt-"+currentHash {
			continue
		}
		_ = os.RemoveAll(filepath.Join(parent, name))
	}
}

// ─── Environment / identity helpers ───────────────────────────────────────

// userToken returns a short, filesystem-safe identifier for the current user.
// Only used for non-temp directory naming; never used to construct /tmp paths.
func userToken() string {
	if u := os.Getenv("USER"); u != "" {
		return sanitize(u, 16)
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return sanitize(u, 16)
	}
	return fmt.Sprintf("pid%d", os.Getpid())
}

// sanitize keeps only alphanumeric characters and underscores, up to maxLen.
func sanitize(s string, maxLen int) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
		if b.Len() >= maxLen {
			break
		}
	}
	if b.Len() == 0 {
		return "user"
	}
	return b.String()
}

// ─── Diagnostics ──────────────────────────────────────────────────────────

// Info returns a human-readable summary of the current platform adapter
// state. Useful for `lunex info` and debug output.
func Info() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "platform       : %s (%s/%s)\n", Current, runtime.GOOS, runtime.GOARCH)

	// CWD cache status.
	cwdDir := CWDCacheDir()
	if cwdDir != "" {
		active := ""
		if UseCWDCache() {
			active = " (active — LUNEX_USE_CWD_CACHE=1)"
		}
		fmt.Fprintf(&sb, "cwd cache      : %s%s\n", cwdDir, active)
	}

	cacheDir := CacheDir()
	if cacheDir == "" {
		fmt.Fprintf(&sb, "cache dir      : (unavailable — memory only)\n")
	} else {
		fmt.Fprintf(&sb, "cache dir      : %s\n", ShortenHome(cacheDir))
	}

	jitDir := JITCacheDir()
	if jitDir == "" {
		fmt.Fprintf(&sb, "jit cache dir  : (unavailable — memory only)\n")
	} else {
		fmt.Fprintf(&sb, "jit cache dir  : %s\n", ShortenHome(jitDir))
	}

	nativeDir := NativeCacheDir()
	if nativeDir == "" {
		fmt.Fprintf(&sb, "native cache   : (unavailable — memory only)\n")
	} else {
		fmt.Fprintf(&sb, "native cache   : %s\n", ShortenHome(nativeDir))
	}

	count, total := MemCacheStats()
	fmt.Fprintf(&sb, "mem cache      : %d entries, %d bytes\n", count, total)

	if mp, ok := MarkerPath(); ok {
		fmt.Fprintf(&sb, "marker path    : %s\n", ShortenHome(mp))
	} else {
		fmt.Fprintf(&sb, "marker path    : (unavailable)\n")
	}

	return sb.String()
}
