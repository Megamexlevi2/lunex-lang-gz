// Lunex lang — internal/modules
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package modules manages external Lunex module metadata at runtime.
//
// External modules are distributed as .nax archives and installed via
//   lunex install <github-url-or-other-url>
//
// The module system only works via .nax files. When you install a package
// it is compiled to a .nax archive and cached locally. Subsequent imports
// work without an internet connection.
//
// To add a new built-in standard module:
//  1. Implement the Go module in internal/builtin/builtin_<name>.go
//  2. Register it in internal/builtin/register.go
//  3. Add metadata to internal/std/modcfg.go

package modules

import (
	"fmt"
	"strings"
	"sync"
)

// Info holds the metadata of one external module.
type Info struct {
	// Name is the @import("name") string used in Lunex source code.
	Name string

	// Description is a short human-readable description of the module.
	Description string

	// SourceFile is the primary Go source file implementing this module.
	SourceFile string

	// Functions maps exported function names to their description strings.
	Functions map[string]string
}

var (
	once    sync.Once
	catalog map[string]*Info
)

func loadAll() {
	// External module catalog is populated at runtime from installed packages.
	// There are no embedded config files — all metadata comes from the
	// installed .nax archives in the Lunex cache directory.
	catalog = make(map[string]*Info)
}

// Get returns the metadata for the given module name.
// Returns nil, false if the module is not found.
func Get(name string) (*Info, bool) {
	once.Do(loadAll)
	m, ok := catalog[name]
	return m, ok
}

// Register adds a module to the in-memory catalog.
// Called by the package loader when it installs a new module.
func Register(info *Info) {
	once.Do(loadAll)
	if info != nil && info.Name != "" {
		catalog[info.Name] = info
	}
}

// All returns a snapshot map of all documented modules, keyed by name.
func All() map[string]*Info {
	once.Do(loadAll)
	result := make(map[string]*Info, len(catalog))
	for k, v := range catalog {
		result[k] = v
	}
	return result
}

// Describe returns a one-line description for the given module name,
// or an empty string if the module is not found.
func Describe(name string) string {
	m, ok := Get(name)
	if !ok {
		return ""
	}
	return m.Description
}

// IsStdName returns true if the given name is a reserved standard library name.
// External modules must not use the same name as any std module.
func IsStdName(name string) bool {
	stdNames := []string{
		"io", "fs", "http", "crypto", "db", "ws", "jwt",
		"math", "datetime", "os", "regex", "env", "utils",
	}
	lower := strings.ToLower(name)
	for _, s := range stdNames {
		if lower == s {
			return true
		}
	}
	return false
}

// ValidateExternalName returns an error if name clashes with a std module.
func ValidateExternalName(name string) error {
	if IsStdName(name) {
		return fmt.Errorf("module name %q conflicts with a standard library module; choose a different name", name)
	}
	return nil
}
