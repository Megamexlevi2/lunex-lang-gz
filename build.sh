#!/usr/bin/env bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

step() { printf '%b==>%b %b%s%b\n' "$BLUE" "$NC" "$BOLD" "$*" "$NC"; }
ok()   { printf '%b[ok]%b %s\n' "$GREEN" "$NC" "$*"; }
warn() { printf '%b[!]%b %s\n' "$YELLOW" "$NC" "$*"; }
fail() { printf '%b[error]%b %s\n' "$RED" "$NC" "$*" >&2; exit 1; }
info() { printf '%b[info]%b %s\n' "$CYAN" "$NC" "$*"; }

NTL_VERSION="0.4.0"
GO_MIN="1.23"
ZIG_MIN="0.17"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

banner() {
cat << 'EOF'

  Lunex lang  v0.4.0
  Created by David Dev
  GitHub: https://github.com/Megamexlevi2

EOF
}

detect_os() {
    case "$(uname -s)" in
        Linux*)
            if [ -f /proc/version ] && grep -qi android /proc/version 2>/dev/null; then
                echo "android"
            else
                echo "linux"
            fi
            ;;
        Darwin*)  echo "macos" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        FreeBSD*) echo "freebsd" ;;
        OpenBSD*) echo "openbsd" ;;
        NetBSD*)  echo "netbsd" ;;
        SunOS*)   echo "solaris" ;;
        *)        echo "unknown" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "x86_64" ;;
        aarch64|arm64) echo "aarch64" ;;
        riscv64)       echo "riscv64" ;;
        *)             echo "unknown" ;;
    esac
}

version_gte() {
    [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

normalize_arch() {
    case "$1" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        riscv64) echo "riscv64" ;;
        *) echo "$1" ;;
    esac
}

normalize_zig_arch() {
    case "$1" in
        x86_64|amd64) echo "x86_64" ;;
        aarch64|arm64) echo "aarch64" ;;
        riscv64) echo "riscv64" ;;
        *) echo "$1" ;;
    esac
}

normalize_go_os() {
    case "$1" in
        macos)   echo "darwin" ;;
        android) echo "linux" ;;
        *)       echo "$1" ;;
    esac
}

runtime_ext_for_os() {
    case "$1" in
        windows) echo ".exe" ;;
        *) echo "" ;;
    esac
}

runtime_name_for_os() {
    case "$1" in
        windows) echo "lunex-rt.exe" ;;
        *) echo "lunex-rt" ;;
    esac
}

install_go() {
    local os arch ver="1.24.3"
    os=$(detect_os)
    arch=$(detect_arch)

    if command -v go >/dev/null 2>&1 && version_gte "$(go version | awk '{print $3}' | tr -d 'go')" "$GO_MIN"; then
        ok "Go $(go version | awk '{print $3}') is already installed"
        return
    fi

    step "Installing Go $ver"

    local goarch goos
    case "$arch" in
        x86_64)  goarch="amd64" ;;
        aarch64) goarch="arm64" ;;
        *)       fail "Unsupported architecture: $arch" ;;
    esac

    case "$os" in
        linux|android) goos="linux" ;;
        macos)         goos="darwin" ;;
        freebsd)       goos="freebsd" ;;
        *)             fail "Automatic Go installation is not supported on $os. Install manually from https://go.dev/dl/" ;;
    esac

    local url="https://go.dev/dl/go${ver}.${goos}-${goarch}.tar.gz"
    local tmp
    tmp="$(mktemp -d)"
    curl -fsSL "$url" -o "$tmp/go.tar.gz"
    sudo tar -C /usr/local -xzf "$tmp/go.tar.gz" 2>/dev/null || tar -C "$HOME" -xzf "$tmp/go.tar.gz"
    rm -rf "$tmp"

    if [ -d /usr/local/go/bin ]; then
        export PATH="/usr/local/go/bin:$PATH"
        printf 'export PATH="/usr/local/go/bin:$PATH"\n' >> "$HOME/.bashrc" 2>/dev/null || true
    elif [ -d "$HOME/go/bin" ]; then
        export PATH="$HOME/go/bin:$PATH"
    fi

    ok "Go $ver installed"
}

install_zig() {
    local os arch ver="0.17.0"
    os=$(detect_os)
    arch=$(detect_arch)

    if [ "$os" = "windows" ] || [ "$os" = "macos" ]; then
        warn "Skipping Zig install on $os (runtime not used on this platform)"
        return
    fi

    if command -v zig >/dev/null 2>&1 && version_gte "$(zig version 2>/dev/null)" "$ZIG_MIN"; then
        ok "Zig $(zig version) is already installed"
        return
    fi

    step "Installing Zig $ver"

    local zigarch zigos
    zigarch=$(normalize_zig_arch "$arch")

    case "$os" in
        linux|android) zigos="linux" ;;
        freebsd)       zigos="freebsd" ;;
        *)             fail "Automatic Zig installation is not supported on $os. Install manually from https://ziglang.org/download/" ;;
    esac

    local name="zig-${zigos}-${zigarch}-${ver}"
    local url="https://ziglang.org/download/${ver}/${name}.tar.xz"
    local tmp
    tmp="$(mktemp -d)"
    curl -fsSL "$url" -o "$tmp/zig.tar.xz"
    tar -C "$tmp" -xJf "$tmp/zig.tar.xz"
    mkdir -p "$HOME/.local/bin"
    cp "$tmp/${name}/zig" "$HOME/.local/bin/zig"
    chmod +x "$HOME/.local/bin/zig"
    export PATH="$HOME/.local/bin:$PATH"
    printf 'export PATH="$HOME/.local/bin:$PATH"\n' >> "$HOME/.bashrc" 2>/dev/null || true
    rm -rf "$tmp"

    ok "Zig $ver installed to ~/.local/bin/zig"
}

build_zig_rt() {
    local os arch output_path zigarch zigos target
    os="${1:-$(detect_os)}"
    arch="${2:-$(detect_arch)}"
    output_path="${3:-$SCRIPT_DIR/bin/lunex-rt}"

    if [ "$os" = "windows" ] || [ "$os" = "macos" ]; then
        warn "Skipping Zig runtime build on $os (not supported on this platform)"
        return
    fi

    mkdir -p "$(dirname "$output_path")"

    if [ -f "$output_path" ] && ! find "$SCRIPT_DIR/zig" \( -name "*.zig" -o -name "build.zig" \) -newer "$output_path" 2>/dev/null | grep -q .; then
        ok "Zig runtime is up to date"
        return
    fi

    step "Building Zig runtime for $os/$(normalize_arch "$arch")"

    cd "$SCRIPT_DIR/zig"

    zigarch=$(normalize_zig_arch "$arch")

    case "$os" in
        freebsd)       zigos="freebsd" ;;
        linux|android) zigos="linux" ;;
        *)             fail "Unsupported Zig target OS: $os" ;;
    esac

    target="${zigarch}-${zigos}"
    zig build -Dtarget="$target" -Doptimize=ReleaseFast

    local built_bin="$SCRIPT_DIR/zig/zig-out/bin/lunex-rt"
    [ -f "$built_bin" ] || fail "Zig build finished without producing lunex-rt"

    cp "$built_bin" "$output_path"
    chmod +x "$output_path"

    ok "Zig runtime built at $output_path"
}

prepare_runtime_for_build() {
    local os runtime_source runtime_dest runtime_dest_exe
    os="$1"
    runtime_source="${2:-}"

    mkdir -p "$SCRIPT_DIR/bin"
    runtime_dest="$SCRIPT_DIR/bin/lunex-rt"
    runtime_dest_exe="$SCRIPT_DIR/bin/lunex-rt.exe"

    if [ -n "$runtime_source" ]; then
        [ -f "$runtime_source" ] || fail "Runtime file not found: $runtime_source"
        
        # Only copy when the paths differ to avoid a cp error on identical src/dst.
        if [ "$runtime_source" != "$runtime_dest" ]; then
            cp "$runtime_source" "$runtime_dest"
        fi
        chmod +x "$runtime_dest"
        
        if [ "$os" = "windows" ]; then
            if [ "$runtime_source" != "$runtime_dest_exe" ]; then
                cp "$runtime_source" "$runtime_dest_exe"
            fi
            chmod +x "$runtime_dest_exe"
        fi
        return
    fi

    case "$os" in
        windows)
            : > "$runtime_dest"
            : > "$runtime_dest_exe"
            ;;
        macos|darwin)
            : > "$runtime_dest"
            ;;
        *)
            fail "Runtime file is required for $os builds"
            ;;
    esac
}

build_go_binary() {
    local os arch runtime_source target_bin goos goarch ext log_file skip_uptodate
    os="${1:-$(detect_os)}"
    arch="${2:-$(detect_arch)}"
    runtime_source="${3:-}"
    target_bin="${4:-}"
    skip_uptodate="${5:-}"

    goos=$(normalize_go_os "$os")
    goarch=$(normalize_arch "$arch")
    ext="$(runtime_ext_for_os "$os")"

    if [ -z "$target_bin" ]; then
        target_bin="$SCRIPT_DIR/lunex$ext"
    fi

    mkdir -p "$(dirname "$target_bin")"

    if [ -z "$skip_uptodate" ] && [ -f "$target_bin" ] && ! find "$SCRIPT_DIR" -maxdepth 1 \( -name "*.go" -o -name "go.mod" -o -name "go.sum" \) -newer "$target_bin" 2>/dev/null | grep -q .; then
        if [ -n "$runtime_source" ] && [ -f "$runtime_source" ] && find "$runtime_source" -newer "$target_bin" 2>/dev/null | grep -q .; then
            :
        elif [ "$os" != "windows" ] && [ "$os" != "macos" ] && [ "$os" != "darwin" ] && find "$SCRIPT_DIR/bin/lunex-rt" -newer "$target_bin" 2>/dev/null | grep -q .; then
            :
        else
            ok "Go binary is up to date"
            return
        fi
    fi

    step "Building Go compiler for $goos/$goarch"

    prepare_runtime_for_build "$os" "$runtime_source"

    cd "$SCRIPT_DIR"
    log_file="$(mktemp)"

    if ! env GONOSUMDB='*' GOFLAGS='-mod=mod' GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -tags netgo -o "$target_bin" . >"$log_file" 2>&1; then
        sed '/^go: downloading/d' "$log_file" >&2
        rm -f "$log_file"
        fail "Go compiler build failed"
    fi

    rm -f "$log_file"
    [ -f "$target_bin" ] || fail "Build completed without producing $target_bin"

    ok "$(basename "$target_bin") built: $(du -sh "$target_bin" | cut -f1)"
}

build_release() {
    step "Starting multi-platform release build"

    local out_dir runtime_root
    out_dir="$SCRIPT_DIR/release"
    runtime_root="$out_dir/.runtime"
    mkdir -p "$out_dir" "$runtime_root"

    local targets=(
        "linux:amd64"
        "linux:arm64"
        "windows:amd64"
        "darwin:amd64"
        "darwin:arm64"
        "android:arm64"
    )

    local t goos goarch release_name runtime_src runtime_target ext
    for t in "${targets[@]}"; do
        IFS=':' read -r goos goarch <<< "$t"
        info "Building target $goos/$goarch"

        ext="$(runtime_ext_for_os "$goos")"
        release_name="lunex-${goos}-${goarch}${ext}"

        runtime_src=""
        case "$goos" in
            linux|android)
                runtime_target="$runtime_root/${goos}-${goarch}/lunex-rt"
                build_zig_rt "$goos" "$goarch" "$runtime_target"
                runtime_src="$runtime_target"
                ;;
            windows|darwin)
                runtime_src=""
                ;;
            *)
                fail "Unsupported release target: $goos/$goarch"
                ;;
        esac

        build_go_binary "$goos" "$goarch" "$runtime_src" "$out_dir/$release_name" "release"
        ok "Done: $out_dir/$release_name ($(du -sh "$out_dir/$release_name" | cut -f1))"
    done

    build_zig_rt "$(detect_os)" "$(detect_arch)" "$SCRIPT_DIR/bin/lunex-rt"
    ok "All targets built successfully in ./$out_dir/"
}

run_tests() {
    step "Running Zig unit tests"
    cd "$SCRIPT_DIR/zig"
    zig build test 2>&1

    if [ -f "$SCRIPT_DIR/tests/run_tests.sh" ]; then
        step "Running Lunex integration tests"
        bash "$SCRIPT_DIR/tests/run_tests.sh"
    fi

    ok "All tests passed"
}

clean() {
    step "Cleaning build artifacts"
    cd "$SCRIPT_DIR"
    rm -rf lunex lunex.exe release
    rm -f bin/lunex-rt bin/lunex-rt.exe
    rm -rf zig/zig-out zig/zig-cache zig/.zig-cache
    rm -f zig/src/extensions_gen.zig
    ok "Cleaned"
}

cmd="${1:-build}"
shift || true

case "$cmd" in
    build|"")
        banner
        target_os="${1:-$(detect_os)}"
        target_arch="${2:-$(detect_arch)}"
        install_go
        
        runtime_src=""
        if [ "$target_os" != "windows" ] && [ "$target_os" != "macos" ] && [ "$target_os" != "darwin" ]; then
            install_zig
            runtime_src="$SCRIPT_DIR/bin/lunex-rt"
        fi
        
        build_zig_rt "$target_os" "$target_arch" "$SCRIPT_DIR/bin/lunex-rt"
        build_go_binary "$target_os" "$target_arch" "$runtime_src" "$SCRIPT_DIR/lunex$(runtime_ext_for_os "$target_os")"
        
        echo
        ok "Lunex v${NTL_VERSION} built successfully"
        info "Run:      ./lunex run <file.lx>"
        info "RT info:  ./lunex rt-info"
        ;;
    release|--release)
        banner
        install_go
        install_zig
        build_release
        ;;
    test)
        target_os="$(detect_os)"
        target_arch="$(detect_arch)"
        install_go
        
        runtime_src=""
        if [ "$target_os" != "windows" ] && [ "$target_os" != "macos" ] && [ "$target_os" != "darwin" ]; then
            install_zig
            runtime_src="$SCRIPT_DIR/bin/lunex-rt"
        fi
        
        build_zig_rt "$target_os" "$target_arch" "$SCRIPT_DIR/bin/lunex-rt"
        build_go_binary "$target_os" "$target_arch" "$runtime_src" "$SCRIPT_DIR/lunex$(runtime_ext_for_os "$target_os")"
        run_tests
        ;;
    clean)
        clean
        ;;
    go-only)
        target_os="$(detect_os)"
        target_arch="$(detect_arch)"
        install_go
        
        runtime_src=""
        if [ "$target_os" != "windows" ] && [ "$target_os" != "macos" ] && [ "$target_os" != "darwin" ]; then
            runtime_src="$SCRIPT_DIR/bin/lunex-rt"
        fi
        
        build_go_binary "$target_os" "$target_arch" "$runtime_src" "$SCRIPT_DIR/lunex$(runtime_ext_for_os "$target_os")"
        ;;
    zig-only)
        install_zig
        build_zig_rt "$(detect_os)" "$(detect_arch)" "$SCRIPT_DIR/bin/lunex-rt"
        ;;
    *)
        echo "Usage: $0 [build|release|--release|test|clean|go-only|zig-only]"
        exit 1
        ;;
esac
