// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package aot

import (
	"fmt"
	"lunex/internal/codegen"
	"os"
	"path/filepath"
	"runtime"
)

type Target struct {
	OS   string
	Arch string
}

var KnownTargets = []Target{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
	{"android", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

func (t Target) String() string { return t.OS + "/" + t.Arch }

func (t Target) ExeSuffix() string {
	if t.OS == "windows" {
		return ".exe"
	}
	return ""
}

func (t Target) IsCurrentPlatform() bool {
	return t.OS == runtime.GOOS && t.Arch == runtime.GOARCH
}

func (t Target) ToCodegenTarget() codegen.Target {
	return codegen.Target{
		Arch: codegen.Arch(t.Arch),
		OS:   codegen.OS(t.OS),
	}
}

func ParseTarget(s string) (Target, error) {
	for _, kt := range KnownTargets {
		if s == kt.String() {
			return kt, nil
		}
	}
	return Target{}, fmt.Errorf("unknown target %q — supported: linux/amd64 linux/arm64 windows/amd64 windows/arm64 android/arm64 darwin/amd64 darwin/arm64", s)
}

func RuntimeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lx", "runtimes")
}

func RuntimeBinaryPath(t Target) string {
	name := "lunex-" + t.OS + "-" + t.Arch + t.ExeSuffix()
	return filepath.Join(RuntimeDir(), name)
}

func FindRuntime(t Target) (string, error) {
	if t.IsCurrentPlatform() {
		self, err := os.Executable()
		if err == nil {
			return self, nil
		}
	}
	p := RuntimeBinaryPath(t)
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf(
		"runtime for %s not found at %s\n  run 'lunex runtimes' to see install instructions",
		t, p,
	)
}
