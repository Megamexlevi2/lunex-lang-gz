// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package lunaresolver resolves @import("pkg") references to files installed
// by the Luna package manager (~/.luna/packages/).
package lunaresolver

import (
	"os"
	"path/filepath"
	"strings"
)

// LunaHome returns the path to the Luna home directory (~/.luna).
func LunaHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".luna")
}

// PackagesDir returns the path to the Luna global packages directory.
func PackagesDir() string {
	return filepath.Join(LunaHome(), "packages")
}

// Resolve looks up a package name in the Luna global package store
// (~/.luna/packages/<name>/) and returns the entry file path if found.
//
// Search order within a package directory:
//  1. .luna-entry  — explicit entry pointer written by Luna on install
//  2. index.lx     — conventional source entry
//  3. main.lx
//  4. <name>.lx    — package name as file
//  5. index.nax    — compiled archive entry
//  6. main.nax
func Resolve(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	// Direct file path — pass through unchanged.
	if info, err := os.Stat(name); err == nil && !info.IsDir() {
		return name, true
	}
	// Try appending known extensions.
	for _, ext := range []string{".lx", ".nax", ".nc"} {
		if !strings.HasSuffix(strings.ToLower(name), ext) {
			if info, err := os.Stat(name + ext); err == nil && !info.IsDir() {
				return name + ext, true
			}
		}
	}

	pkgsDir := PackagesDir()
	if pkgsDir == "" {
		return "", false
	}

	// Normalize: "github.com/user/repo" → last path segment for simple lookup.
	safeName := strings.ReplaceAll(name, "/", "__")

	// Candidate directories to check (exact match then prefix match).
	candidates := []string{
		filepath.Join(pkgsDir, name),    // exact: ~/.luna/packages/pkgname
		filepath.Join(pkgsDir, safeName), // safe: ~/.luna/packages/github.com__user__repo
	}

	// Also scan for versioned directories: pkgname@version.
	if entries, err := os.ReadDir(pkgsDir); err == nil {
		prefix := safeName + "@"
		suffix := "__" + safeName + "@"
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			d := e.Name()
			if strings.HasPrefix(d, prefix) || strings.Contains(d, suffix) {
				candidates = append(candidates, filepath.Join(pkgsDir, d))
			}
		}
	}

	baseName := filepath.Base(name)
	entryFiles := []string{
		".luna-entry",
		"index.lx", "main.lx", baseName + ".lx",
		"index.nax", "main.nax",
	}

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		// Check .luna-entry pointer first.
		if data, err := os.ReadFile(filepath.Join(dir, ".luna-entry")); err == nil {
			entryName := strings.TrimSpace(string(data))
			fp := filepath.Join(dir, entryName)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}

		// Try well-known entry names.
		for _, file := range entryFiles {
			if file == ".luna-entry" {
				continue
			}
			fp := filepath.Join(dir, file)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}

		// Last resort: any .lx file in the directory.
		if files, err := os.ReadDir(dir); err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".lx") {
					return filepath.Join(dir, f.Name()), true
				}
			}
		}
	}

	return "", false
}
