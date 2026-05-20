#!/usr/bin/env sh
set -eu

REPO="https://github.com/Megamexlevi2/ntl-lang-gz/releases/latest/download"

while :; do
  clear 2>/dev/null || true

  printf '%s\n' "NTL Installer"
  printf '\n'
  printf '%s\n' "1) Linux amd64"
  printf '%s\n' "2) Linux arm64"
  printf '%s\n' "3) macOS amd64 (Intel)"
  printf '%s\n' "4) macOS arm64 (Apple Silicon)"
  printf '%s\n' "5) Android arm64 (Termux)"
  printf '%s\n' "0) Exit"
  printf '\n'

  printf '%s' "Select an option: "
  IFS= read -r choice || exit 0

  case "$choice" in
    1)
      asset="ntl-linux-amd64"
      target="${HOME}/.local/bin/ntl"
      ;;
    2)
      asset="ntl-linux-arm64"
      target="${HOME}/.local/bin/ntl"
      ;;
    3)
      asset="ntl-darwin-amd64"
      target="${HOME}/.local/bin/ntl"
      ;;
    4)
      asset="ntl-darwin-arm64"
      target="${HOME}/.local/bin/ntl"
      ;;
    5)
      asset="ntl-android-arm64"
      if [ -n "${PREFIX:-}" ]; then
        target="${PREFIX}/bin/ntl"
      else
        target="${HOME}/.local/bin/ntl"
      fi
      ;;
    0)
      exit 0
      ;;
    *)
      printf '%s\n' "Invalid option."
      sleep 1
      continue
      ;;
  esac

  tmp="$(mktemp 2>/dev/null || printf '%s\n' "/tmp/ntl-install.$$")"
  url="${REPO}/${asset}"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$tmp"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$tmp" "$url"
  else
    printf '%s\n' "Error: curl or wget is required."
    rm -f "$tmp"
    exit 1
  fi

  mkdir -p "$(dirname "$target")"
  install -m 755 "$tmp" "$target"
  rm -f "$tmp"

  printf '\n%s\n' "Installed successfully:"
  printf '%s\n' "$target"
  printf '\n%s\n' "Run:"
  printf '%s\n' "ntl help"
  break
done