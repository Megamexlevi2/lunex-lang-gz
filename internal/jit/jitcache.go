// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Licensed under the Mozilla Public License, Version 2.0.

// This file provides on-disk JIT cache management for the Go side.
// Machine-code stub cache helpers — retained for API compatibility.


package jit

import (
	"lunex/internal/adaptor"
	"os"
	"path/filepath"
)

// JITCacheDir returns the directory used to store JIT cache entries.
// Delegates to the platform adaptor so all path resolution is centralised.
func JITCacheDir() string {
	return adaptor.JITCacheDir()
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
	return adaptor.MemCacheKey(absPath)
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
