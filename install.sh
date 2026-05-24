#!/usr/bin/env sh
set -eu

REPO="https://github.com/Megamexlevi2/lunex-lang-gz/releases/latest/download"

while :; do
  clear 2>/dev/null || true

  printf '%s\n' "Lunex Installer"
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
      asset="lunex-linux-amd64"
      target="${HOME}/.local/bin/lunex"
      ;;
    2)
      asset="lunex-linux-arm64"
      target="${HOME}/.local/bin/lunex"
      ;;
    3)
      asset="lunex-darwin-amd64"
      target="${HOME}/.local/bin/lunex"
      ;;
    4)
      asset="lunex-darwin-arm64"
      target="${HOME}/.local/bin/lunex"
      ;;
    5)
      asset="lunex-android-arm64"
      if [ -n "${PREFIX:-}" ]; then
        target="${PREFIX}/bin/lunex"
      else
        target="${HOME}/.local/bin/lunex"
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

  tmp="$(mktemp 2>/dev/null || printf '%s\n' "/tmp/lunex-install.$$")"
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
  printf '%s\n' "lunex help"
  break
done