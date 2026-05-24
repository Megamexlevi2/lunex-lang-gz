// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import (
	"lunex/internal/compiler"
	"strings"
)

// DiagSeverity indicates error vs warning
type DiagSeverity int

const (
	SevError DiagSeverity = iota
	SevWarning
	SevInfo
)

// Diagnostic is a single error/warning from the compiler
type Diagnostic struct {
	Line     int
	Col      int
	EndLine  int
	EndCol   int
	Message  string
	Severity DiagSeverity
	Code     string
}

// DiagnosticsResult holds all diagnostics for the current file
type DiagnosticsResult struct {
	Diagnostics []Diagnostic
	HasErrors   bool
}

// CheckSource runs the Lunex compiler/parser on the source and returns diagnostics
func CheckSource(source, filePath string) DiagnosticsResult {
	if filePath == "" {
		filePath = "<untitled>"
	}

	c := compiler.New(compiler.DefaultOptions)
	result := c.CompileSource(source, filePath)

	var diags []Diagnostic
	hasErrors := false

	for _, e := range result.Errors {
		errStr := e.Error()
		d := Diagnostic{
			Line:     1,
			Col:      1,
			Message:  errStr,
			Severity: SevError,
		}

		// Try to parse line/col from error string: "file:line:col: message" or "line:col: message"
		parseLineCol(errStr, &d)

		diags = append(diags, d)
		hasErrors = true
	}

	return DiagnosticsResult{
		Diagnostics: diags,
		HasErrors:   hasErrors,
	}
}

// parseLineCol attempts to extract line/col info from a compiler error string
func parseLineCol(errStr string, d *Diagnostic) {
	// Format: "filename:line:col: message" or "line:col: message"
	// Try to find pattern like ":N:N:"
	parts := strings.Split(errStr, ":")
	if len(parts) < 3 {
		return
	}

	// Find numeric parts
	for i := 0; i < len(parts)-1; i++ {
		p := strings.TrimSpace(parts[i])
		var line, col int
		if n, err := parseInt(p); err == nil && n > 0 {
			line = n
			if i+1 < len(parts) {
				if c, err := parseInt(strings.TrimSpace(parts[i+1])); err == nil && c > 0 {
					col = c
					d.Line = line
					d.Col = col
					// Message is everything after line:col:
					if i+2 < len(parts) {
						d.Message = strings.TrimSpace(strings.Join(parts[i+2:], ":"))
					}
					return
				}
			}
			_ = line
		}
	}
}

func parseInt(s string) (int, error) {
	n := 0
	if len(s) == 0 {
		return 0, &parseError{}
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, &parseError{}
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

type parseError struct{}
func (e *parseError) Error() string { return "not a number" }

// GetDiagnosticsForLine returns diagnostics that belong to a specific line (0-indexed)
func GetDiagnosticsForLine(diags []Diagnostic, line int) []Diagnostic {
	var result []Diagnostic
	for _, d := range diags {
		if d.Line-1 == line {
			result = append(result, d)
		}
	}
	return result
}

// FormatDiagnostics formats diagnostics for display in the error panel
func FormatDiagnostics(diags []Diagnostic) []string {
	lines := make([]string, 0, len(diags))
	for _, d := range diags {
		sev := "error"
		if d.Severity == SevWarning {
			sev = "warning"
		}
		lines = append(lines, sev+": "+d.Message+" (line "+itoa(d.Line)+")")
	}
	return lines
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
