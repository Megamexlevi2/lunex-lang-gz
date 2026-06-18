// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// Package meta provides runtime integrity and build metadata utilities.
// All version information is sourced exclusively from the embedded version.json.
package meta

import (
	"encoding/json"
	"fmt"
	"sync"
)

// _versionData holds the raw bytes of the embedded version.json.
// It is set once at program startup via SetVersionData, called from main.init().
var (
	_versionData []byte
	_once        sync.Once
	_cached      versionInfo
)

// SetVersionData stores the raw version.json bytes embedded by main.go.
// Must be called before any Version* functions are used (i.e., in main.init()).
func SetVersionData(data []byte) {
	_versionData = data
	_once = sync.Once{} // reset so next access re-parses fresh data
}

// versionInfo mirrors the shape of version.json exactly.
type versionInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	GitHub      string   `json:"github"`
	Repository  string   `json:"repository"`
	License     string   `json:"license"`
	Year        string   `json:"year"`
	BuildDate   string   `json:"buildDate"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
	MinGo       string   `json:"minGo"`
	Platforms   []string `json:"platforms"`
}

// loadVersion parses _versionData once and caches the result.
func loadVersion() versionInfo {
	_once.Do(func() {
		_ = json.Unmarshal(_versionData, &_cached)
	})
	return _cached
}

// ── Public accessors ──────────────────────────────────────────────────────────

// Version returns the version string (e.g. "0.5.2 alpha 5").
func Version() string {
	v := loadVersion()
	if v.Version == "" {
		return "unknown"
	}
	return v.Version
}

// Name returns the project name (e.g. "Lunex Lang").
func Name() string {
	v := loadVersion()
	if v.Name == "" {
		return "Lunex"
	}
	return v.Name
}

// Author returns the author field from version.json.
func Author() string {
	return loadVersion().Author
}

// Year returns the copyright year.
func Year() string {
	return loadVersion().Year
}

// BuildDate returns the build date string.
func BuildDate() string {
	return loadVersion().BuildDate
}

// Repository returns the repository URL.
func Repository() string {
	return loadVersion().Repository
}

// License returns the license name.
func License() string {
	return loadVersion().License
}

// Description returns the short project description.
func Description() string {
	return loadVersion().Description
}

// GitHub returns the author's GitHub URL.
func GitHub() string {
	return loadVersion().GitHub
}

// ── Compatibility alias ───────────────────────────────────────────────────────

// FullVersion is an alias for Version, kept for backward compatibility.
func FullVersion() string { return Version() }

// ── Display helpers ───────────────────────────────────────────────────────────

// PrintVersion prints a formatted version block to stdout.
func PrintVersion() {
	v := loadVersion()
	name := v.Name
	if name == "" {
		name = "Lunex"
	}
	fmt.Printf("%s %s\n", name, v.Version)
	if v.BuildDate != "" {
		fmt.Printf("build    %s\n", v.BuildDate)
	}
	if v.Author != "" {
		fmt.Printf("author   %s\n", v.Author)
	}
	if v.License != "" {
		fmt.Printf("license  %s\n", v.License)
	}
	if v.Repository != "" {
		fmt.Printf("repo     %s\n", v.Repository)
	}
}
