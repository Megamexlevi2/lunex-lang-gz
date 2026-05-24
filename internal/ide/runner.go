// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// RunResult holds the result of running an Lunex file
type RunResult struct {
	Output   []string
	ExitCode int
	Duration time.Duration
	Err      error
	Done     bool
}

// runMsg is a tea message containing run output
type runMsg struct {
	result RunResult
}

// diagMsg is a tea message containing diagnostic results
type diagMsg struct {
	result DiagnosticsResult
}

// tickMsg is used for debouncing diagnostics
type tickMsg struct{}

// GetLunexExecutable returns the path to the lunex binary
func GetLunexExecutable() string {
	// First try the same executable (self)
	self, err := os.Executable()
	if err == nil {
		return self
	}

	// Then try lunex in PATH
	if path, err := exec.LookPath("lunex"); err == nil {
		return path
	}

	// Windows
	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("lunex.exe"); err == nil {
			return path
		}
	}

	return "lunex"
}

// RunFile saves and runs the given file, returning the result
func RunFile(filePath string, content string) RunResult {
	start := time.Now()
	result := RunResult{}

	// Ensure file is saved
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		result.Err = err
		result.Output = []string{"Error saving file: " + err.Error()}
		result.Done = true
		return result
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	ntlBin := GetLunexExecutable()
	cmd := exec.Command(ntlBin, "run", absPath)
	cmd.Dir = filepath.Dir(absPath)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	result.Duration = time.Since(start)
	result.Done = true

	// Combine stdout and stderr
	output := outBuf.String()
	errOutput := errBuf.String()

	allOutput := output
	if errOutput != "" {
		if allOutput != "" {
			allOutput += "\n"
		}
		allOutput += errOutput
	}

	// Split into lines
	if allOutput == "" {
		allOutput = "(no output)"
	}
	lines := strings.Split(strings.TrimRight(allOutput, "\n"), "\n")
	result.Output = lines

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Err = runErr
		}
	}

	return result
}

// FormatFile runs the Lunex formatter and returns the formatted source
func FormatFile(source string) string {
	// Use the compiler's Format function directly
	// This is called from the model after import
	return source
}
