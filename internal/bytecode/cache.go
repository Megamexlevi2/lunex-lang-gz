// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package bytecode

import (
        "fmt"
        "lunex/internal/adaptor"
)

// CacheDir returns the platform-resolved bytecode cache directory.
// Delegates entirely to the adaptor so that platform quirks (Termux, Android,
// noexec mounts, missing home directory) are handled in one place.
func CacheDir() string {
        return adaptor.CacheDir()
}

// CacheKey returns a deterministic hex key for the given source file,
// incorporating path + mtime + size.
func CacheKey(absPath string) (string, error) {
        key, err := adaptor.MemCacheKey(absPath)
        if err != nil {
                return "", fmt.Errorf("cache key error: %w", err)
        }
        return key, nil
}

// CacheLookup checks both disk and in-memory cache for compiled bytecode.
func CacheLookup(absPath string) ([]byte, bool) {
        return adaptor.CacheLookup(absPath)
}

// CacheStore writes bytecode to disk (best-effort) and always to the
// in-memory cache so execution is never blocked by filesystem restrictions.
func CacheStore(absPath string, ncData []byte) error {
        adaptor.CacheStore(absPath, ncData)
        return nil
}

// CacheInvalidate removes the entry for absPath from disk and memory.
func CacheInvalidate(absPath string) {
        adaptor.CacheInvalidate(absPath)
}

// overrideCacheDir holds a user-specified cache directory, or "" to use the
// platform default from adaptor.CacheDir().
var overrideCacheDir string

// SetCacheDir changes the on-disk bytecode cache directory for this process.
// Pass dir="" to revert to the platform default.  The caller is responsible for
// creating the directory before calling SetCacheDir.
func SetCacheDir(dir string) error {
        overrideCacheDir = dir
        return nil
}

// UnpackNAX decodes the .nax archive in data and writes every entry as a file
// under outDir.  It returns the number of files written and any error.
func UnpackNAX(data []byte, outDir string) (int, error) {
        return unpackNAXData(data, outDir)
}
