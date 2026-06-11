// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package pkg

import (
        "encoding/json"
        "fmt"
        "io"
        "lunex/internal/adaptor"
        "lunex/internal/meta"
        "net/http"
        "os"
        "path/filepath"
        "strings"
        "time"

        "lunex/internal/compiler"
        "lunex/internal/runtime"
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
        GitHub       string            `json:"github"`
        URL          string            `json:"url"`
        Main         string            `json:"main"`
        Entry        string            `json:"entry"`
        Output       string            `json:"output"`
        Optimize     bool              `json:"optimize"`
        Targets      []string          `json:"targets"`
        Dependencies map[string]string `json:"dependencies"`
}

// CacheDir returns the platform-resolved modules cache directory.
// Delegates to the platform adaptor.
func CacheDir() string {
        return adaptor.ModuleDir("", "")
}

func ModuleDir(name, version string) string {
        return adaptor.ModuleDir(name, version)
}

func resolveConfigPath(path string) string {
        info, err := os.Stat(path)
        if err == nil && info.IsDir() {
                candidate := filepath.Join(path, "config.lx")
                if _, err := os.Stat(candidate); err == nil {
                        return candidate
                }
                return candidate
        }
        return path
}

func LoadManifest(path string) (*Manifest, error) {
        path = resolveConfigPath(path)
        data, err := os.ReadFile(path)
        if err != nil {
                return nil, err
        }

        trimmed := bytesTrimSpace(data)
        if len(trimmed) > 0 && trimmed[0] == '{' {
                var m Manifest
                if err := json.Unmarshal(trimmed, &m); err == nil {
                        if m.Dependencies == nil {
                                m.Dependencies = make(map[string]string)
                        }
                        if m.Main == "" && m.Entry != "" {
                                m.Main = m.Entry
                        }
                        if m.Entry == "" && m.Main != "" {
                                m.Entry = m.Main
                        }
                        return &m, nil
                }
        }

        if looksLikeConfigScript(string(data)) {
                m, err := loadManifestScript(path, string(data))
                if err != nil {
                        return nil, err
                }
                if m != nil {
                        return m, nil
                }
        }

        return parseConfigLX(string(data)), nil
}

func looksLikeConfigScript(content string) bool {
        for _, needle := range []string{"val ", "fn ", "const ", "@import(", "project = {", "config = {"} {
                if strings.Contains(content, needle) {
                        return true
                }
        }
        return false
}

func loadManifestScript(path, content string) (*Manifest, error) {
        c := compiler.New(compiler.DefaultOptions)
        result := c.CompileSource(content, path)
        if !result.Success || result.AST == nil {
                return nil, fmt.Errorf("manifest script compile failed")
        }

        interp := c.Interpreter()
        interp.SetFilename(path)
        interp.SetSourceLines(strings.Split(content, "\n"))
        if _, err := interp.Exec(result.AST); err != nil {
                return nil, err
        }

        for _, name := range []string{"project", "config"} {
                if v, ok := interp.GetGlobal(name); ok && v != nil {
                        if m, ok := manifestFromValue(v); ok {
                                return m, nil
                        }
                }
        }

        return nil, nil
}

func manifestFromValue(v *runtime.Value) (*Manifest, bool) {
        if v == nil || v.Tag != runtime.TypeObject || v.ObjVal == nil {
                return nil, false
        }

        m := &Manifest{Dependencies: make(map[string]string)}
        getString := func(keys ...string) string {
                for _, key := range keys {
                        if val, ok := v.ObjVal[key]; ok && val != nil {
                                if val.Tag == runtime.TypeString {
                                        return val.StrVal
                                }
                                return val.ToString()
                        }
                }
                return ""
        }

        m.Name = getString("name")
        m.Version = getString("version")
        m.Description = getString("description")
        m.Author = getString("author")
        m.License = getString("license")
        m.GitHub = getString("github")
        m.URL = getString("url")
        m.Main = getString("main")
        m.Entry = getString("entry")
        m.Output = getString("output", "out")

        if optimize, ok := v.ObjVal["optimize"]; ok && optimize != nil {
                m.Optimize = optimize.IsTruthy()
        }
        if targets, ok := v.ObjVal["targets"]; ok && targets != nil && targets.Tag == runtime.TypeArray {
                for _, item := range targets.ArrVal {
                        if item == nil {
                                continue
                        }
                        if item.Tag == runtime.TypeString {
                                m.Targets = append(m.Targets, item.StrVal)
                        } else {
                                m.Targets = append(m.Targets, item.ToString())
                        }
                }
        }
        if deps, ok := v.ObjVal["dependencies"]; ok && deps != nil && deps.Tag == runtime.TypeObject {
                for name, ver := range deps.ObjVal {
                        if ver == nil {
                                continue
                        }
                        if ver.Tag == runtime.TypeString {
                                m.Dependencies[name] = ver.StrVal
                        } else {
                                m.Dependencies[name] = ver.ToString()
                        }
                }
        }

        if m.Main == "" && m.Entry != "" {
                m.Main = m.Entry
        }
        if m.Entry == "" && m.Main != "" {
                m.Entry = m.Main
        }
        if m.Output == "" {
                m.Output = "dist"
        }
        if m.Dependencies == nil {
                m.Dependencies = make(map[string]string)
        }
        return m, true
}

func Resolve(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	// Direct file paths (.lx, .nax, .nc) — used when resolving local @imports.
	if info, err := os.Stat(name); err == nil && !info.IsDir() {
		return name, true
	}
	// Try appending known extensions if the caller omitted them.
	for _, ext := range []string{".lx", ".nax", ".nc"} {
		if !strings.HasSuffix(strings.ToLower(name), ext) {
			if info, err := os.Stat(name + ext); err == nil && !info.IsDir() {
				return name + ext, true
			}
		}
	}

	// Search the installed package cache. Priority order:
	//   1. index.nax  — pre-compiled archive (fastest to load)
	//   2. index.lx   — source entry point
	//   3. main.nax / main.lx / config.lx — alternate entry point names
	cache := CacheDir()
	entries, err := os.ReadDir(cache)
	if err != nil {
		return "", false
	}

	// Entry files to look for, in priority order.
	entryFiles := []string{
		"index.nax", "index.lx",
		"main.nax", "main.lx",
		"config.lx",
	}

	var candidate string
	prefix := strings.ReplaceAll(name, "/", "__") + "@"
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		p := filepath.Join(cache, e.Name())
		for _, file := range entryFiles {
			fp := filepath.Join(p, file)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}
		if candidate == "" {
			candidate = p
		}
	}

	if candidate != "" {
		for _, file := range entryFiles {
			fp := filepath.Join(candidate, file)
			if st, err := os.Stat(fp); err == nil && !st.IsDir() {
				return fp, true
			}
		}
	}

	return "", false
}

func bytesTrimSpace(b []byte) []byte {
        start := 0
        end := len(b)
        for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
                start++
        }
        for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
                end--
        }
        return b[start:end]
}

func parseConfigLX(content string) *Manifest {
        m := &Manifest{Dependencies: make(map[string]string)}
        lines := strings.Split(content, "\n")
        block := ""
        for _, line := range lines {
                line = strings.TrimSpace(line)
                if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
                        continue
                }

                switch line {
                case "project {", "config {", "build {":
                        block = "project"
                        continue
                case "dependencies {", "deps {", "[dependencies]", "[deps]":
                        block = "dependencies"
                        continue
                case "}":
                        block = ""
                        continue
                }

                key, val, ok := strings.Cut(line, "=")
                if !ok {
                        continue
                }
                key = strings.TrimSpace(key)
                val = strings.TrimSpace(val)
                val = strings.Trim(val, "\"'")

                if block == "dependencies" {
                        m.Dependencies[key] = val
                        continue
                }

                switch key {
                case "name":
                        m.Name = val
                case "version":
                        m.Version = val
                case "description":
                        m.Description = val
                case "author":
                        m.Author = val
                case "license":
                        m.License = val
                case "github":
                        m.GitHub = val
                case "url":
                        m.URL = val
                case "main":
                        m.Main = val
                case "entry":
                        m.Entry = val
                case "output", "out":
                        m.Output = val
                case "optimize":
                        m.Optimize = val == "true" || val == "1" || val == "yes"
                case "targets", "target":
                        val = strings.Trim(val, "[]")
                        for _, t := range strings.Split(val, ",") {
                                t = strings.TrimSpace(strings.Trim(t, "\"' "))
                                if t != "" {
                                        m.Targets = append(m.Targets, t)
                                }
                        }
                }
        }

        if m.Main == "" && m.Entry != "" {
                m.Main = m.Entry
        }
        if m.Entry == "" && m.Main != "" {
                m.Entry = m.Main
        }
        if m.Dependencies == nil {
                m.Dependencies = make(map[string]string)
        }
        return m
}

func SaveManifest(path string, m *Manifest) error {
        path = resolveConfigPath(path)
        if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
                return err
        }

        entry := m.Entry
        if entry == "" {
                entry = m.Main
        }
        if entry == "" {
                entry = "main.lx"
        }
        if m.Main == "" {
                m.Main = entry
        }
        if m.Entry == "" {
                m.Entry = entry
        }

        output := m.Output
        if output == "" {
                output = "dist"
        }

        if m.Dependencies == nil {
                m.Dependencies = make(map[string]string)
        }

        // description falls back to empty placeholder so the field is always present
        description := m.Description
        if description == "" {
                description = "A Lunex project"
        }
        author := m.Author
        if author == "" {
                author = "unknown"
        }
        license := m.License
        if license == "" {
                license = "MIT"
        }
        github := m.GitHub
        url := m.URL

        var sb strings.Builder

        // File-level comment header (mirrors Lunex source file conventions)
        sb.WriteString("// config.lx — project manifest\n")
        sb.WriteString(fmt.Sprintf("// %s\n", description))
        sb.WriteString(fmt.Sprintf("// Author : %s\n", author))
        if github != "" {
                sb.WriteString(fmt.Sprintf("// GitHub : %s\n", github))
        }
        if url != "" {
                sb.WriteString(fmt.Sprintf("// URL    : %s\n", url))
        }
        sb.WriteString(fmt.Sprintf("// License: %s\n", license))
        sb.WriteString("\n")

        // Project block
        sb.WriteString("val project = {\n")

        // Identity
        sb.WriteString("  // --- identity ---\n")
        sb.WriteString(fmt.Sprintf("  name:        %q\n", m.Name))
        sb.WriteString(fmt.Sprintf("  version:     %q\n", m.Version))
        sb.WriteString(fmt.Sprintf("  description: %q\n", description))
        sb.WriteString(fmt.Sprintf("  author:      %q\n", author))
        sb.WriteString(fmt.Sprintf("  license:     %q\n", license))
        if github != "" {
                sb.WriteString(fmt.Sprintf("  github:      %q\n", github))
        } else {
                sb.WriteString("  github:      \"\"  // e.g. \"https://github.com/you/project\"\n")
        }
        if url != "" {
                sb.WriteString(fmt.Sprintf("  url:         %q\n", url))
        } else {
                sb.WriteString("  url:         \"\"  // project homepage or docs URL\n")
        }

        // Build
        sb.WriteString("\n  // --- build ---\n")
        sb.WriteString(fmt.Sprintf("  main:        %q\n", m.Main))
        sb.WriteString(fmt.Sprintf("  entry:       %q\n", m.Entry))
        sb.WriteString(fmt.Sprintf("  output:      %q\n", output))
        if m.Optimize {
                sb.WriteString("  optimize:    true\n")
        } else {
                sb.WriteString("  optimize:    false\n")
        }

        // Dependencies
        sb.WriteString("\n  // --- dependencies: \"pkg-name\": \"version\" ---\n")
        sb.WriteString("  dependencies: {\n")
        for name, ver := range m.Dependencies {
                sb.WriteString(fmt.Sprintf("    %q: %q\n", name, ver))
        }
        sb.WriteString("  }\n")
        sb.WriteString("}\n\n")

        // build() function — evaluated by the manifest loader
        sb.WriteString("// build() is called by `lunex build`. Must return the project object.\n")
        sb.WriteString("fn build() {\n")
        sb.WriteString("  project\n")
        sb.WriteString("}\n")

        return os.WriteFile(path, []byte(sb.String()), 0644)
}

func InitManifest(dir string, name string) error {
        path := filepath.Join(dir, "config.lx")
        if _, err := os.Stat(path); err == nil {
                return fmt.Errorf("config.lx already exists")
        }
        m := &Manifest{
                Name:         name,
                Version:      meta.Version(),
                Description:  "A Lunex project",
                Author:       "unknown",
                License:      "MIT",
                GitHub:       "",
                URL:          "",
                Main:         "main.lx",
                Entry:        "main.lx",
                Output:       "dist",
                Optimize:     true,
                Targets:      []string{},
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
        return SaveManifest(resolveConfigPath(manifestPath), m)
}

func List() []Module {
        entries, _ := os.ReadDir(CacheDir())
        var mods []Module
        for _, e := range entries {
                if !e.IsDir() {
                        continue
                }
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

// githubEntry represents a single file or directory entry from the GitHub API.
type githubEntry struct {
        Name        string `json:"name"`
        Path        string `json:"path"`
        DownloadURL string `json:"download_url"`
        Type        string `json:"type"` // "file" or "dir"
        Size        int    `json:"size"`
        HTMLURL     string `json:"html_url"`
}

// githubClient is a thin HTTP client for GitHub API calls with retry and timeout.
type githubClient struct {
        http *http.Client
}

func newGitHubClient() *githubClient {
        return &githubClient{
                http: &http.Client{Timeout: 60 * time.Second},
        }
}

func (c *githubClient) getJSON(url string, v interface{}) error {
        var lastErr error
        for attempt := 0; attempt < 3; attempt++ {
                if attempt > 0 {
                        time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
                }
                resp, err := c.http.Get(url)
                if err != nil {
                        lastErr = err
                        continue
                }
                defer resp.Body.Close()
                if resp.StatusCode == 403 {
                        return fmt.Errorf("GitHub rate limit exceeded — wait a moment and try again")
                }
                if resp.StatusCode == 404 {
                        return fmt.Errorf("not found: %s", url)
                }
                if resp.StatusCode != 200 {
                        return fmt.Errorf("GitHub API returned HTTP %d for %s", resp.StatusCode, url)
                }
                if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
                        lastErr = err
                        continue
                }
                return nil
        }
        return fmt.Errorf("GitHub API request failed after 3 attempts: %w", lastErr)
}

func (c *githubClient) downloadFile(url string) ([]byte, error) {
        var lastErr error
        for attempt := 0; attempt < 3; attempt++ {
                if attempt > 0 {
                        time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
                }
                resp, err := c.http.Get(url)
                if err != nil {
                        lastErr = err
                        continue
                }
                defer resp.Body.Close()
                if resp.StatusCode == 404 {
                        return nil, fmt.Errorf("file not found: %s", url)
                }
                if resp.StatusCode != 200 {
                        lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
                        continue
                }
                data, err := io.ReadAll(resp.Body)
                if err != nil {
                        lastErr = err
                        continue
                }
                return data, nil
        }
        return nil, fmt.Errorf("download failed after 3 attempts: %w", lastErr)
}

// fetchDirRecursive downloads all files under a GitHub repo path into localDir.
// It walks subdirectories using the GitHub Contents API.
func (c *githubClient) fetchDirRecursive(owner, repo, ref, remotePath, localDir string) error {
        apiURL := fmt.Sprintf(
                "https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
                owner, repo, remotePath, ref)

        var entries []githubEntry
        if err := c.getJSON(apiURL, &entries); err != nil {
                return err
        }

        for _, entry := range entries {
                localPath := filepath.Join(localDir, filepath.FromSlash(entry.Name))
                switch entry.Type {
                case "file":
                        if entry.DownloadURL == "" {
                                continue
                        }
                        if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
                                return err
                        }
                        data, err := c.downloadFile(entry.DownloadURL)
                        if err != nil {
                                return fmt.Errorf("downloading %s: %w", entry.Path, err)
                        }
                        if err := os.WriteFile(localPath, data, 0644); err != nil {
                                return err
                        }
                case "dir":
                        if err := os.MkdirAll(localPath, 0755); err != nil {
                                return err
                        }
                        if err := c.fetchDirRecursive(owner, repo, ref, entry.Path, localPath); err != nil {
                                return err
                        }
                }
        }
        return nil
}

// InstallFromGitHub downloads a Lunex package from a GitHub repository.
// It recursively fetches all files and subdirectories under subpath.
func InstallFromGitHub(name, owner, repo, ref, subpath string) (*Module, error) {
        if owner == "" || repo == "" {
                return nil, fmt.Errorf("invalid GitHub source: owner and repo are required")
        }
        if ref == "" {
                ref = "main"
        }

        dir := ModuleDir(name, ref)
        if err := os.MkdirAll(dir, 0755); err != nil {
                return nil, fmt.Errorf("creating module directory: %w", err)
        }

        client := newGitHubClient()

        remotePath := subpath
        if remotePath == "" {
                remotePath = ""
        }

        err := client.fetchDirRecursive(owner, repo, ref, remotePath, dir)
        if err != nil {
                // Fallback: try raw.githubusercontent.com for single-file packages
                rawBase := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", owner, repo, ref)
                if subpath != "" {
                        rawBase += "/" + subpath
                }
                for _, filename := range []string{"index.lx", "main.lx"} {
                        data, dlErr := client.downloadFile(rawBase + "/" + filename)
                        if dlErr == nil {
                                if writeErr := os.WriteFile(filepath.Join(dir, filename), data, 0644); writeErr == nil {
                                        fmt.Printf("  fetched %s/%s (single-file fallback)\n", subpath, filename)
                                        goto found
                                }
                        }
                }
                // Try the subpath itself as a direct .lx file
                if strings.HasSuffix(subpath, ".lx") {
                        data, dlErr := client.downloadFile(rawBase)
                        if dlErr == nil {
                                filename := filepath.Base(subpath)
                                if writeErr := os.WriteFile(filepath.Join(dir, filename), data, 0644); writeErr == nil {
                                        goto found
                                }
                        }
                }
                return nil, fmt.Errorf("could not download %s/%s@%s: %w", owner, repo, ref, err)
        found:
        }

        // Locate the package entry point.
        entry := "index.lx"
        for _, candidate := range []string{"index.lx", "main.lx"} {
                if _, err := os.Stat(filepath.Join(dir, candidate)); err == nil {
                        entry = candidate
                        break
                }
        }

        return &Module{
                Name:    name,
                Version: ref,
                Source:  fmt.Sprintf("github.com/%s/%s/%s@%s", owner, repo, subpath, ref),
                Path:    filepath.Join(dir, entry),
        }, nil
}

func Install(spec string) (*Module, error) {
        spec = strings.TrimSpace(spec)
        if spec == "" {
                return nil, fmt.Errorf("empty package spec")
        }

        // GitHub URL or owner/repo path
        if strings.Contains(spec, "/") {
                owner, repo, ref, subpath := resolveSource(spec)
                if owner == "" || repo == "" {
                        return nil, fmt.Errorf("invalid package spec %q: expected owner/repo[@ref]", spec)
                }
                pkgName := repo
                if subpath != "" {
                        pkgName = repo + "/" + subpath
                }
                return InstallFromGitHub(pkgName, owner, repo, ref, subpath)
        }

        // Simple name[@version] spec — not a GitHub path, treat as named package
        atIdx := strings.Index(spec, "@")
        name := spec
        ver := "latest"
        if atIdx > 0 {
                name = spec[:atIdx]
                ver = spec[atIdx+1:]
        }

        return &Module{
                Name:    name,
                Version: ver,
                Path:    filepath.Join(ModuleDir(name, ver), "index.lx"),
        }, nil
}
