// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Module struct {
	Name    string
	Version string
	Source  string
	Path    string
}

type Manifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license"`
	Main         string            `json:"main"`
	Dependencies map[string]string `json:"dependencies"`
}

func CacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".lx", "modules")
	}
	return filepath.Join(home, ".lx", "modules")
}

func ModuleDir(name, version string) string {
	if version == "" {
		version = "main"
	}
	safe := strings.ReplaceAll(name, "/", "__")
	return filepath.Join(CacheDir(), safe+"@"+version)
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return parseNTLMod(string(data)), nil
	}
	return &m, nil
}

func parseNTLMod(content string) *Manifest {
	m := &Manifest{Dependencies: make(map[string]string)}
	lines := strings.Split(content, "\n")
	inDeps := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		if line == "[dependencies]" || line == "[deps]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 { continue }
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if inDeps {
			m.Dependencies[key] = val
		} else {
			switch key {
			case "name": m.Name = val
			case "version": m.Version = val
			case "main": m.Main = val
			}
		}
	}
	return m
}

func SaveManifest(path string, m *Manifest) error {
	var sb strings.Builder
	sb.WriteString("lunex 5.0.0\n\n")
	sb.WriteString(fmt.Sprintf("name = %q\nversion = %q\n", m.Name, m.Version))
	if len(m.Dependencies) > 0 {
		sb.WriteString("\n[dependencies]\n")
		for name, ver := range m.Dependencies {
			sb.WriteString(fmt.Sprintf("%-20s = %q\n", name, ver))
		}
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}

func resolveSource(spec string) (owner, repo, ref, subpath string) {
	spec = strings.TrimPrefix(spec, "github.com/")
	spec = strings.TrimPrefix(spec, "https://github.com/")

	atIdx := strings.Index(spec, "@")
	ref = "main"
	if atIdx > 0 {
		ref = spec[atIdx+1:]
		spec = spec[:atIdx]
	}

	parts := strings.Split(spec, "/")
	if len(parts) >= 2 {
		owner = parts[0]
		repo = parts[1]
	}
	if len(parts) > 2 {
		subpath = strings.Join(parts[2:], "/")
	}
	return
}

func downloadFile(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return nil, fmt.Errorf("HTTP %d", resp.StatusCode) }
	return io.ReadAll(resp.Body)
}

func InstallFromGitHub(name, owner, repo, ref, subpath string) (*Module, error) {
	dir := ModuleDir(name, ref)
	os.MkdirAll(dir, 0755)

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, subpath, ref)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var files []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"download_url"`
		Type        string `json:"type"`
		Path        string `json:"path"`
	}

	if resp.StatusCode == 200 {
		json.NewDecoder(resp.Body).Decode(&files)
		for _, file := range files {
			if file.Type == "file" {
				data, err := downloadFile(file.DownloadURL)
				if err == nil {
					targetPath := filepath.Join(dir, file.Name)
					os.WriteFile(targetPath, data, 0644)
				}
			}
		}
	} else {
		rawBase := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, subpath)
		urls := []string{rawBase + "/index.lx", rawBase + "/main.lx", rawBase}
		for _, u := range urls {
			data, err := downloadFile(u)
			if err == nil {
				os.WriteFile(filepath.Join(dir, "index.lx"), data, 0644)
				break
			}
		}
	}

	return &Module{
		Name:    name,
		Version: ref,
		Source:  fmt.Sprintf("github.com/%s/%s/%s@%s", owner, repo, subpath, ref),
		Path:    filepath.Join(dir, "index.lx"),
	}, nil
}

func Install(spec string) (*Module, error) {
	if strings.Contains(spec, "/") {
		owner, repo, ref, subpath := resolveSource(spec)
		return InstallFromGitHub(repo, owner, repo, ref, subpath)
	}

	atIdx := strings.Index(spec, "@")
	name, version := spec, "main"
	if atIdx > 0 {
		name = spec[:atIdx]
		version = spec[atIdx+1:]
	}

	owner := "Megamexlevi2"
	repo := "lunex-modules"
	subpath := name

	return InstallFromGitHub(name, owner, repo, version, subpath)
}

func Resolve(name string) (string, bool) {
	entries, _ := os.ReadDir(CacheDir())
	prefix := strings.ReplaceAll(name, "/", "__") + "@"
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			path := filepath.Join(CacheDir(), e.Name(), "index.lx")
			if _, err := os.Stat(path); err == nil { return path, true }
		}
	}
	return "", false
}

func List() []Module {
	entries, _ := os.ReadDir(CacheDir())
	var mods []Module
	for _, e := range entries {
		if !e.IsDir() { continue }
		parts := strings.Split(e.Name(), "@")
		if len(parts) == 2 {
			mods = append(mods, Module{
				Name:    strings.ReplaceAll(parts[0], "__", "/"),
				Version: parts[1],
				Path:    filepath.Join(CacheDir(), e.Name(), "index.lx"),
			})
		}
	}
	return mods
}

func Remove(name string) error {
	entries, _ := os.ReadDir(CacheDir())
	prefix := strings.ReplaceAll(name, "/", "__") + "@"
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			return os.RemoveAll(filepath.Join(CacheDir(), e.Name()))
		}
	}
	return fmt.Errorf("not found")
}
func InitManifest(dir string, name string) error {
	path := filepath.Join(dir, "lunex.mod")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("lunex.mod already exists")
	}
	m := &Manifest{
		Name:         name,
		Version:      "0.1.0",
		Dependencies: make(map[string]string),
	}
	return SaveManifest(path, m)
}

func AddToManifest(manifestPath, spec string, mod *Module) error {
	var m *Manifest
	if loaded, err := LoadManifest(manifestPath); err == nil {
		m = loaded
	} else {
		m = &Manifest{Dependencies: make(map[string]string)}
	}
	if m.Dependencies == nil {
		m.Dependencies = make(map[string]string)
	}
	m.Dependencies[mod.Name] = mod.Version
	return SaveManifest(manifestPath, m)
}
