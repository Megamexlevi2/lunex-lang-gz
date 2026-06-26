// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package bootstrap handles the installation of the Luna package manager
// from GitHub. It is used to bootstrap Luna before it is self-managing.
package bootstrap

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

// InstallLuna downloads and installs the Luna package manager from GitHub.
// It places the scripts in ~/.luna/bin/ and creates a `luna` shim in PATH.
func InstallLuna() error {
	fmt.Printf("%s%s Luna Package Manager — Installer%s\n\n", colorBold, colorCyan, colorReset)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	lunaHome := filepath.Join(home, ".luna")
	lunaBin  := filepath.Join(lunaHome, "bin")

	// ── Step 1: create directory structure ───────────────────────────────────
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

	// ── Step 2: download luna-pm.lx from GitHub ───────────────────────────────
	step("Downloading luna-pm from GitHub (%s/%s/%s)", lunaOwner, lunaRepo, lunaSubdir)
	pmDest := filepath.Join(lunaBin, "luna-pm.lx")

	rawURL := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/%s/%s/luna-pm.lx",
		lunaOwner, lunaRepo, lunaRef, lunaSubdir,
	)
	if err := downloadFile(rawURL, pmDest); err != nil {
		// Try the zip fallback.
		warn("direct download failed, trying archive fallback…")
		if err2 := downloadViaZip(lunaHome, lunaBin); err2 != nil {
			return fmt.Errorf("download failed: %w (zip fallback: %v)", err, err2)
		}
	}
	ok()

	// ── Step 3: write version file ─────────────────────────────────────────────
	vFile := filepath.Join(lunaHome, "VERSION")
	_ = os.WriteFile(vFile, []byte(time.Now().Format("2006-01-02")), 0644)

	// ── Step 4: write the `luna` shim ─────────────────────────────────────────
	step("Writing luna shim")
	shimPath, err := writeLunaShim(lunaBin, pmDest)
	if err != nil {
		return fmt.Errorf("cannot write luna shim: %w", err)
	}
	ok()

	// ── Step 5: detect PATH ────────────────────────────────────────────────────
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

// ── helpers ───────────────────────────────────────────────────────────────────

func step(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("  %s•%s %s… ", colorCyan, colorReset, msg)
}

func ok() {
	fmt.Printf("%s✔%s\n", colorGreen, colorReset)
}

func warn(msg string) {
	fmt.Printf("\n  %s!%s %s\n  ", colorYellow, colorReset, msg)
}

func printSuccess(msg string) {
	fmt.Printf("%s%s✔ %s%s\n", colorBold, colorGreen, msg, colorReset)
}

// downloadFile downloads url into destPath, following redirects.
func downloadFile(url, destPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0644)
}

// downloadViaZip downloads the entire luna repo as a zip and extracts luna-pm/.
func downloadViaZip(lunaHome, lunaBin string) error {
	zipURL := fmt.Sprintf(
		"https://github.com/%s/%s/archive/refs/heads/%s.zip",
		lunaOwner, lunaRepo, lunaRef,
	)
	tmpZip := filepath.Join(lunaHome, "luna-download.zip")
	if err := downloadFile(zipURL, tmpZip); err != nil {
		return err
	}
	defer os.Remove(tmpZip)

	// Extract luna-pm.lx from the zip.
	return extractLunaPMFromZip(tmpZip, lunaSubdir, lunaBin)
}

// extractLunaPMFromZip extracts files from subdir inside the zip into destDir.
func extractLunaPMFromZip(zipPath, subdir, destDir string) error {
	// Use the system unzip if available.
	if _, err := exec.LookPath("unzip"); err == nil {
		pattern := fmt.Sprintf("*/%s/*", subdir)
		cmd := exec.Command("unzip", "-j", "-o", zipPath, pattern, "-d", destDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("unzip failed: %v — %s", err, string(out))
		}
		return nil
	}
	return fmt.Errorf("unzip not found; please install unzip and retry")
}

// writeLunaShim creates the `luna` shim script and marks it executable.
// Returns the path of the shim that was written.
func writeLunaShim(lunaBin, pmScript string) (string, error) {
	// Prefer to place the shim alongside the lunex binary so it's already in PATH.
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
# Luna Package Manager shim — generated by: lunex run luna-pm/luna-pm.lx
# Do not edit by hand. Re-run the bootstrap to regenerate.
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

// shimInstallDir returns the preferred directory for the luna shim.
// Priority:
//  1. Same directory as the running lunex binary
//  2. ~/.local/bin  (XDG standard, works on Linux / Termux)
//  3. ~/bin
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

// isInPATH returns true if the directory of shimPath is in the current PATH.
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

// printPathHint prints shell-specific instructions for adding luna to PATH.
func printPathHint(shimPath string) {
	shimDir := filepath.Dir(shimPath)
	shell := os.Getenv("SHELL")

	fmt.Printf("%s! Add Luna to your PATH:%s\n\n", colorYellow, colorReset)

	switch {
	case strings.Contains(shell, "zsh"):
		fmt.Printf("  echo 'export PATH=\"%s:$PATH\"' >> ~/.zshrc\n", shimDir)
		fmt.Printf("  source ~/.zshrc\n")
	case strings.Contains(shell, "fish"):
		fmt.Printf("  fish_add_path %s\n", shimDir)
	default: // bash, sh, Termux default
		fmt.Printf("  echo 'export PATH=\"%s:$PATH\"' >> ~/.bashrc\n", shimDir)
		fmt.Printf("  source ~/.bashrc\n")
	}

	fmt.Printf("\nThen run %sluna --version%s to verify.\n", colorCyan, colorReset)
}
