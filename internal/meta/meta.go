// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// Package meta provides runtime integrity and build metadata utilities.
package meta

import "encoding/json"

// _versionData holds the raw version.json bytes, set by main via SetVersionData().
var _versionData []byte

// SetVersionData receives the embedded version.json bytes from main.
// Must be called in main's init() before Version() is used.
func SetVersionData(data []byte) {
	_versionData = data
}

// Version parses and returns the version field from version.json.
// Falls back to the compiled default if the file cannot be parsed.
func Version() string {
	if len(_versionData) == 0 {
		return "0.3.0"
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(_versionData, &v); err != nil || v.Version == "" {
		return "0.3.0"
	}
	return v.Version
}
