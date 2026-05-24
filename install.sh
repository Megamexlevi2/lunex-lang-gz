#!/usr/bin/env sh
set -eu

REPO="https://github.com/Megamexlevi2/lunex-lang-gz/releases/latest/download"

read_choice() {
  if [ -r /dev/tty ]; then
    IFS= read -r choice </dev/tty || exit 0
  else
    IFS= read -r choice || exit 0
  fi
  printf '%s' "$choice"
}

format_bytes() {
  bytes="$1"
  awk -v b="$bytes" 'BEGIN {
    split("B KB MB GB TB", u, " ")
    i = 1
    while (b >= 1024 && i < 5) {
      b /= 1024
      i++
    }
    printf "%.2f %s", b, u[i]
  }'
}

download_file() {
  url="$1"
  out="$2"

  if command -v curl >/dev/null 2>&1; then
    stats="$(curl -L --fail --progress-bar -o "$out" "$url" -w '%{size_download} %{speed_download}')"
    set -- $stats
    bytes="${1:-0}"
    speed_bps="${2:-0}"

    downloaded="$(format_bytes "$bytes")"
    speed="$(format_bytes "$speed_bps")/s"

    printf '\n%s\n' "Downloaded: $downloaded"
    printf '%s\n' "Speed: $speed"
  elif command -v wget >/dev/null 2>&1; then
    start="$(date +%s)"
    wget --show-progress -O "$out" "$url"
    end="$(date +%s)"

    bytes="$(wc -c < "$out" | tr -d ' ')"
    elapsed="$((end - start))"

    if [ "$elapsed" -le 0 ]; then
      elapsed=1
    fi

    speed_bps=$((bytes / elapsed))
    downloaded="$(format_bytes "$bytes")"
    speed="$(format_bytes "$speed_bps")/s"

    printf '\n%s\n' "Downloaded: $downloaded"
    printf '%s\n' "Speed: $speed"
  else
    printf '%s\n' "Error: curl or wget is required."
    exit 1
  fi
}

while :; do
  clear 2>/dev/null || true

  printf '%s\n\n' "Lunex Installer"
  printf '%s\n' "1) Linux amd64"
  printf '%s\n' "2) Linux arm64"
  printf '%s\n' "3) macOS amd64 (Intel)"
  printf '%s\n' "4) macOS arm64 (Apple Silicon)"
  printf '%s\n' "5) Android arm64 (Termux)"
  printf '%s\n' "0) Exit"
  printf '\n%s' "Select an option: "

  choice="$(read_choice)"

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

  tmpdir="${TMPDIR:-/tmp}"
  tmp="$(mktemp "$tmpdir/lunex-install.XXXXXX" 2>/dev/null || printf '%s\n' "$tmpdir/lunex-install.$$")"
  url="${REPO}/${asset}"

  printf '\n%s\n' "Downloading $asset..."
  download_file "$url" "$tmp"

  mkdir -p "$(dirname "$target")"
  cp "$tmp" "$target"
  chmod 755 "$target"
  rm -f "$tmp"

  printf '\n%s\n' "Installed successfully:"
  printf '%s\n' "$target"
  printf '\n%s\n' "Run:"
  printf '%s\n' "lunex help"
  break
done