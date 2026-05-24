// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import (
	"lunex/internal/compiler"
	"strings"
)

// FormatSource runs the Lunex formatter on source code
func FormatSource(source string) (string, error) {
	formatted := compiler.Format(source)
	return formatted, nil
}

// AutoIndent calculates the correct indentation for a new line
// given the previous line and the current indentation level.
func AutoIndent(prevLine string, indent string) string {
	trimmed := strings.TrimRight(prevLine, " \t")
	// Increase indent after opening braces
	if strings.HasSuffix(trimmed, "{") || strings.HasSuffix(trimmed, "(") {
		return indent + "  "
	}
	return indent
}

// LeadingIndent returns the leading whitespace of a string
func LeadingIndent(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

// AutoCloseBracket returns the auto-closing character for a given char, or ""
func AutoCloseBracket(ch rune) string {
	switch ch {
	case '{':
		return "}"
	case '(':
		return ")"
	case '[':
		return "]"
	case '"':
		return `"`
	case '\'':
		return "'"
	case '`':
		return "`"
	}
	return ""
}

// ShouldAutoClose returns true if auto-close should be inserted for this char
func ShouldAutoClose(line string, cx int, ch rune) bool {
	runes := []rune(line)
	// Don't auto-close if next char is already the same closing char
	if cx < len(runes) && rune(runes[cx]) == ch {
		return false
	}
	// For quotes: only auto-close if the count of that quote character
	// BEFORE the cursor is even (meaning we're not already inside a string)
	if ch == '"' || ch == '\'' || ch == '`' {
		count := 0
		for _, r := range runes[:cx] {
			if r == ch {
				count++
			}
		}
		// If count is odd, we're inside a string — don't auto-close
		if count%2 != 0 {
			return false
		}
	}
	return true
}

// DefaultTemplate returns the default content for a new .lx file
func DefaultTemplate() string {
	return `val io = @import("std.io")

fn main() {
  
}
`
}
