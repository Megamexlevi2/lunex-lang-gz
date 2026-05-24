// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Lunex Source License — attribution required, copying prohibited.

// This file provides on-disk JIT cache management for the Go side.
// Machine-code stubs are no longer written by Go; the Zig JIT runtime manages
// its own code cache internally.  The functions here are retained for API
// compatibility (e.g. the "jit clear-cache" CLI command).
package jit

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// JITCacheDir returns the directory used to store JIT cache entries.
func JITCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".lx", "jit")
	}
	return filepath.Join(home, ".lx", "jit")
}

func stubPath(key string) string {
	return filepath.Join(JITCacheDir(), key+".bin")
}

// SaveStub writes raw bytes to the on-disk JIT cache under the given key.
func SaveStub(key string, code []byte) error {
	dir := JITCacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(stubPath(key), code, 0600)
}

// LoadStub reads bytes from the on-disk JIT cache.
// Returns (nil, false) when the entry does not exist.
// Note: the bytes are not executed in-process; execution is handled by the
// Zig runtime.
func LoadStub(key string) ([]byte, bool) {
	code, err := os.ReadFile(stubPath(key))
	if err != nil || len(code) == 0 {
		return nil, false
	}
	return code, true
}

// FileJITKey returns a deterministic cache key for a file based on its path,
// content, and modification time.
func FileJITKey(absPath string) (string, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(absPath))
	h.Write(data)
	h.Write([]byte(info.ModTime().String()))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ClearJITCache removes all .bin files from the on-disk JIT cache directory
// and returns the number of files removed.
func ClearJITCache() (int, error) {
	dir := JITCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, nil
	}
	removed := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".bin" {
			os.Remove(filepath.Join(dir, e.Name()))
			removed++
		}
	}
	return removed, nil
}

// JITCacheInfo returns the number of cached stubs and their total size in bytes.
func JITCacheInfo() (count int, totalBytes int64) {
	dir := JITCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".bin" {
			count++
			if info, err := e.Info(); err == nil {
				totalBytes += info.Size()
			}
		}
	}
	return
}
