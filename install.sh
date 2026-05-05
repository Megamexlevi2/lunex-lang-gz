#!/usr/bin/env bash
set -euo pipefail

while true; do
  clear
  echo "NTL Installer"
  echo
  echo "1) Linux amd64"
  echo "2) Linux arm64"
  echo "3) Darwin amd64"
  echo "4) Darwin arm64"
  echo "5) Android arm64"
  echo "0) Exit"
  echo
  read -r -p "Select an option: " choice

  case "$choice" in
    1) asset="ntl-linux-amd64"; target="$HOME/.local/bin/ntl" ;;
    2) asset="ntl-linux-arm64"; target="$HOME/.local/bin/ntl" ;;
    3) asset="ntl-darwin-amd64"; target="$HOME/.local/bin/ntl" ;;
    4) asset="ntl-darwin-arm64"; target="$HOME/.local/bin/ntl" ;;
    5)
      asset="ntl-android-arm64"
      if [ -n "${PREFIX:-}" ]; then
        target="$PREFIX/bin/ntl"
      else
        target="$HOME/.local/bin/ntl"
      fi
      ;;
    0)
      exit 0
      ;;
    *)
      echo "Invalid option."
      sleep 1
      continue
      ;;
  esac

  tmp="$(mktemp)"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://github.com/Megamexlevi2/ntl-go/releases/latest/download/$asset" -o "$tmp"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$tmp" "https://github.com/Megamexlevi2/ntl-go/releases/latest/download/$asset"
  else
    echo "curl or wget is required."
    exit 1
  fi

  mkdir -p "$(dirname "$target")"
  install -m 755 "$tmp" "$target"
  rm -f "$tmp"

  echo
  echo "Installed to: $target"
  break
done