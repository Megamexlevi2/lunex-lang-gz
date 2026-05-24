// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package bytecode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func CacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".lx", "cache")
	}
	return filepath.Join(home, ".lx", "cache")
}

func CacheKey(absPath string) (string, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(absPath))
	h.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 16)))
	h.Write([]byte(strconv.FormatInt(info.Size(), 16)))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CacheLookup(absPath string) ([]byte, bool) {
	key, err := CacheKey(absPath)
	if err != nil {
		return nil, false
	}
	cachePath := filepath.Join(CacheDir(), key+".nc")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}
	return data, true
}

func CacheStore(absPath string, ncData []byte) error {
	key, err := CacheKey(absPath)
	if err != nil {
		return fmt.Errorf("cache key error: %w", err)
	}
	dir := CacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cachePath := filepath.Join(dir, key+".nc")
	return os.WriteFile(cachePath, ncData, 0600)
}

func CacheInvalidate(absPath string) {
	key, err := CacheKey(absPath)
	if err != nil {
		return
	}
	os.Remove(filepath.Join(CacheDir(), key+".nc"))
}
