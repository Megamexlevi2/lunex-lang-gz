#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/bin"
OUT_DIR="$SCRIPT_DIR/release/android"

GOOS="linux"
ZIG_TARGET_BASE="linux"

detect_arch() {
case "$(uname -m)" in
aarch64|arm64) echo "arm64" ;;
armv7l|armv8l) echo "arm" ;;
x86_64) echo "amd64" ;;
*) echo "arm64" ;;
esac
}

normalize_go_arch() {
case "$1" in
arm64) echo "arm64" ;;
arm) echo "arm" ;;
amd64) echo "amd64" ;;
*) echo "arm64" ;;
esac
}

normalize_zig_arch() {
case "$1" in
arm64) echo "aarch64" ;;
arm) echo "arm" ;;
amd64) echo "x86_64" ;;
*) echo "aarch64" ;;
esac
}

prepare_bin_runtime() {
mkdir -p "$BIN_DIR"
cp "$1" "$BIN_DIR/lunex-rt"
chmod +x "$BIN_DIR/lunex-rt"
}

build_zig_rt() {
arch="$1"
target="$BIN_DIR/lunex-rt-$arch"

mkdir -p "$BIN_DIR"

cd "$SCRIPT_DIR/zig"

zig build -Dtarget="$(normalize_zig_arch "$arch")-$ZIG_TARGET_BASE" -Doptimize=ReleaseFast

cp zig-out/bin/lunex-rt "$target"
chmod +x "$target"

cp "$target" "$BIN_DIR/lunex-rt"

cd "$SCRIPT_DIR"
}

build_go() {
arch="$1"
out="$OUT_DIR/$arch/lunex"

mkdir -p "$(dirname "$out")"

GOARCH="$(normalize_go_arch "$arch")"

GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 \
go build -trimpath -ldflags "-s -w" -o "$out" .

chmod +x "$out"
}

build_all() {
archs=("arm64" "arm" "amd64")

for a in "${archs[@]}"; do
build_zig_rt "$a"
build_go "$a"
done
}

clean() {
rm -rf "$OUT_DIR" "$BIN_DIR"
}

case "${1:-build}" in
build)
mkdir -p "$OUT_DIR"
build_all
;;
clean)
clean
;;
*)
echo "Usage: build | clean"
;;
esac