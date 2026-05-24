// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package zigrt manages the embedded Zig runtime subprocess.
// After the process starts, Zig owns its own runtime directory — Go stays out of it.
package zigrt

import (
	"fmt"
	"io"
	"lunex/internal/bridge"
	"os"
	"os/exec"
	"sync"
)

type session struct {
	cmd    *exec.Cmd
	client *bridge.Client
	stdin  io.WriteCloser
	stdout io.ReadCloser
	once   sync.Once
}

var (
	globalSession *session
	sessionMu     sync.Mutex
)

// acquire returns the running Zig process, starting it on the first call.
func acquire() (*session, error) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if globalSession != nil {
		return globalSession, nil
	}
	s, err := startSession()
	if err != nil {
		return nil, err
	}
	globalSession = s
	return s, nil
}

func startSession() (*session, error) {
	binPath, err := ExtractOnce()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(binPath, "ncp-server")
	cmd.Stderr = os.Stderr // let Zig errors land in our terminal

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lunex: stdin pipe failed: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("lunex: stdout pipe failed: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("lunex: can't start Zig runtime (%s): %w", binPath, err)
	}

	client := bridge.NewClient(stdin, stdout)
	return &session{cmd: cmd, client: client, stdin: stdin, stdout: stdout}, nil
}

func (s *session) close() {
	s.once.Do(func() {
		_ = s.client.Kill()
		_ = s.stdin.Close()
		_ = s.cmd.Wait()
	})
}

// RunPipe sends bytecode directly to the Zig runtime and waits for the result.
// No temp files, no cache — this is the hot path.
func RunPipe(ncData []byte) (bridge.Result, error) {
	s, err := acquire()
	if err != nil {
		return bridge.Result{}, err
	}
	return s.client.ExecPipe(ncData)
}

// RunFile tells Zig to execute a .nc file from disk.
func RunFile(absPath string) (bridge.Result, error) {
	s, err := acquire()
	if err != nil {
		return bridge.Result{}, err
	}
	return s.client.ExecFile(absPath)
}

// RunBench runs a file and the runtime will print timing details.
func RunBench(absPath string) (bridge.Result, error) {
	return RunFile(absPath)
}

// Info prints the runtime version and platform info to stdout.
func Info() error {
	s, err := acquire()
	if err != nil {
		return err
	}
	info, err := s.client.RuntimeInfo()
	if err != nil {
		return err
	}
	fmt.Print(info)
	return nil
}

// Available reports whether the embedded runtime can be extracted successfully.
func Available() bool {
	_, err := ExtractOnce()
	return err == nil
}

// Shutdown gracefully stops the Zig process. Call via defer in main().
func Shutdown() {
	sessionMu.Lock()
	s := globalSession
	globalSession = nil
	sessionMu.Unlock()
	if s != nil {
		s.close()
	}
}
