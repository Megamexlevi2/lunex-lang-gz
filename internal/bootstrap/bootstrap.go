package bootstrap

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	lunaOwner  = "Megamexlevi2"
	lunaRepo   = "luna"
	lunaSubdir = "luna-pm"
	lunaRef    = "main"

	colorCyan   = "\033[96m"
	colorGreen  = "\033[92m"
	colorYellow = "\033[93m"
	colorRed    = "\033[91m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
	colorDim    = "\033[2m"
)

func InstallLuna() error {
	fmt.Printf("%s%s Luna Package Manager — Installer%s\n\n", colorBold, colorCyan, colorReset)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	lunaHome := filepath.Join(home, ".luna")
	lunaBin := filepath.Join(lunaHome, "bin")

	step("Creating Luna directory structure")
	for _, dir := range []string{
		lunaHome,
		lunaBin,
		filepath.Join(lunaHome, "packages"),
		filepath.Join(lunaHome, "cache"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("cannot create %s: %w", dir, err)
		}
	}
	ok()

	step("Downloading luna-pm from GitHub (%s/%s/%s)", lunaOwner, lunaRepo, lunaSubdir)
	pmDest := filepath.Join(lunaBin, "luna-pm.lx")
	if err := downloadViaZip(lunaHome, lunaBin); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	ok()

	vFile := filepath.Join(lunaHome, "VERSION")
	_ = os.WriteFile(vFile, []byte(time.Now().Format("2006-01-02")), 0644)

	step("Writing luna shim")
	shimPath, err := writeLunaShim(lunaBin, pmDest)
	if err != nil {
		return fmt.Errorf("cannot write luna shim: %w", err)
	}
	ok()

	fmt.Println()
	printSuccess("Luna installed successfully!")
	fmt.Printf("  Scripts : %s%s%s\n", colorDim, lunaBin, colorReset)
	fmt.Printf("  Shim    : %s%s%s\n", colorDim, shimPath, colorReset)
	fmt.Println()

	if !isInPATH(shimPath) {
		printPathHint(shimPath)
	} else {
		fmt.Printf("%s✔ luna is already in your PATH%s\n", colorGreen, colorReset)
		fmt.Printf("\nRun %sluna --version%s to verify the installation.\n", colorCyan, colorReset)
	}

	return nil
}

func step(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("  %s•%s %s… ", colorCyan, colorReset, msg)
}

func ok() {
	fmt.Printf("%s✔%s\n", colorGreen, colorReset)
}

func printSuccess(msg string) {
	fmt.Printf("%s%s✔ %s%s\n", colorBold, colorGreen, msg, colorReset)
}

func downloadBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// downloadViaZip downloads the GitHub repo zip and extracts luna-pm/ into lunaBin.
// Uses Go's archive/zip so it works on any platform including Termux/BusyBox.
func downloadViaZip(lunaHome, lunaBin string) error {
	zipURL := fmt.Sprintf(
		"https://github.com/%s/%s/archive/refs/heads/%s.zip",
		lunaOwner, lunaRepo, lunaRef,
	)

	data, err := downloadBytes(zipURL)
	if err != nil {
		return fmt.Errorf("download error: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("cannot read zip: %w", err)
	}

	// Zip entries look like: "luna-main/luna-pm/luna-pm.lx"
	//                        "luna-main/luna-pm/src/commands/install.lx"
	// We find the prefix "luna-main/luna-pm/" and strip it, writing
	// the rest into lunaBin.
	// The prefix varies by branch name so we detect it dynamically.
	subdirSuffix := "/" + lunaSubdir + "/"
	extracted := 0

	for _, f := range zr.File {
		// Find the index of "/luna-pm/" in the path.
		idx := strings.Index(f.Name, subdirSuffix)
		if idx < 0 {
			continue
		}
		// rel is the path inside luna-pm/, e.g. "luna-pm.lx" or "src/commands/install.lx"
		rel := f.Name[idx+len(subdirSuffix):]
		if rel == "" {
			continue // the luna-pm/ directory entry itself
		}

		dest := filepath.Join(lunaBin, filepath.FromSlash(rel))

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("cannot open zip entry %s: %w", f.Name, err)
		}
		buf, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("cannot read zip entry %s: %w", f.Name, err)
		}
		if err := os.WriteFile(dest, buf, 0644); err != nil {
			return fmt.Errorf("cannot write %s: %w", dest, err)
		}
		extracted++
	}

	if extracted == 0 {
		return fmt.Errorf("no files found under %q in the GitHub archive", lunaSubdir)
	}
	return nil
}

func writeLunaShim(lunaBin, pmScript string) (string, error) {
	shimDir := shimInstallDir()
	shimName := "luna"
	if runtime.GOOS == "windows" {
		shimName = "luna.cmd"
	}
	shimPath := filepath.Join(shimDir, shimName)

	if err := os.MkdirAll(shimDir, 0755); err != nil {
		return "", err
	}

	var shimContent string
	if runtime.GOOS == "windows" {
		shimContent = fmt.Sprintf("@echo off\r\n"+
			"setlocal\r\n"+
			"set LUNA_PM=%s\r\n"+
			"set CMD=%%1\r\n"+
			"if /i \"%%CMD%%\"==\"install\"  goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"remove\"   goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"rm\"       goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"list\"     goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"ls\"       goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"update\"   goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"search\"   goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"version\"  goto :pm\r\n"+
			"if /i \"%%CMD%%\"==\"help\"     goto :pm\r\n"+
			":run\r\n"+
			"lunex run %%*\r\n"+
			"goto :eof\r\n"+
			":pm\r\n"+
			"lunex run \"%%LUNA_PM%%\" %%*\r\n"+
			":eof\r\n",
			pmScript)
	} else {
		shimContent = fmt.Sprintf(`#!/bin/sh
LUNA_PM=%q

case "$1" in
  install|remove|rm|list|ls|update|upgrade|search|version|help|--version|-v|--help|-h)
    exec lunex run "$LUNA_PM" "$@"
    ;;
  *)
    exec lunex run "$@"
    ;;
esac
`, pmScript)
	}

	if err := os.WriteFile(shimPath, []byte(shimContent), 0755); err != nil {
		return "", err
	}
	return shimPath, nil
}

func shimInstallDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if canWrite(dir) {
			return dir
		}
	}
	home, _ := os.UserHomeDir()
	for _, candidate := range []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
	} {
		if canWrite(candidate) || os.MkdirAll(candidate, 0755) == nil {
			return candidate
		}
	}
	return filepath.Join(home, ".local", "bin")
}

func canWrite(dir string) bool {
	probe := filepath.Join(dir, ".luna_probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(probe)
	return true
}

func isInPATH(shimPath string) bool {
	shimDir := filepath.Dir(shimPath)
	pathEnv := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	for _, dir := range strings.Split(pathEnv, sep) {
		if filepath.Clean(dir) == filepath.Clean(shimDir) {
			return true
		}
	}
	return false
}

func printPathHint(shimPath string) {
	shimDir := filepath.Dir(shimPath)
	shell := os.Getenv("SHELL")

	fmt.Printf("%s! Add Luna to your PATH:%s\n\n", colorYellow, colorReset)

	switch {
	case strings.Contains(shell, "zsh"):
		fmt.Printf("  echo 'export PATH=\"%s:$PATH\"'  >> ~/.zshrc\n", shimDir)
		fmt.Printf("  source ~/.zshrc\n")
	case strings.Contains(shell, "fish"):
		fmt.Printf("  fish_add_path %s\n", shimDir)
	default:
		fmt.Printf("  echo 'export PATH=\"%s:$PATH\"'  >> ~/.bashrc\n", shimDir)
		fmt.Printf("  source ~/.bashrc\n")
	}

	fmt.Printf("\nThen run %sluna --version%s to verify.\n", colorCyan, colorReset)
}


