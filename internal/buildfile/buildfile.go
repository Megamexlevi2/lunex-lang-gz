// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package buildfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Name     string
	Version  string
	Entry    string
	Output   string
	Targets  []string
	Optimize bool
}

func DefaultConfig() Config {
	return Config{
		Name:     "app",
		Version:  "0.1.0",
		Entry:    "main.lx",
		Output:   ".",
		Optimize: true,
	}
}

func Find() (string, bool) {
	for _, name := range []string{"build.lx", "Build.lx"} {
		if _, err := os.Stat(name); err == nil {
			return name, true
		}
	}
	return "", false
}

func Parse(path string) (Config, error) {
	cfg := DefaultConfig()
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, "\"'")

		switch key {
		case "name":
			cfg.Name = val
		case "version":
			cfg.Version = val
		case "entry":
			cfg.Entry = val
		case "output", "out":
			cfg.Output = val
		case "optimize":
			cfg.Optimize = val == "true" || val == "1" || val == "yes"
		case "targets", "target":
			val = strings.Trim(val, "[]")
			for _, t := range strings.Split(val, ",") {
				t = strings.TrimSpace(strings.Trim(t, "\"' "))
				if t != "" {
					cfg.Targets = append(cfg.Targets, t)
				}
			}
		}
	}
	return cfg, sc.Err()
}

func Generate(path string, name string) error {
	content := fmt.Sprintf(`# Lunex build configuration

name    = "%s"
version = "0.1.0"
entry   = "main.lx"
output  = "dist"
optimize = true

# Cross-compilation targets (uncomment what you need)
# targets = ["linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64", "android/arm64"]
targets = []
`, name)
	return os.WriteFile(path, []byte(content), 0644)
}
