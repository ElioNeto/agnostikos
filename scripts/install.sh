#!/bin/bash
#
# install.sh — AgnosticOS install script
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ElioNeto/agnostikos/main/scripts/install.sh | sh
#
# This script detects the OS and architecture, fetches the latest release
# from GitHub, verifies the SHA256 checksum, and installs the 'agnostic'
# binary to /usr/local/bin (or ~/.local/bin as fallback).
#

set -euo pipefail

REPO="ElioNeto/agnostikos"
BINARY="agnostic"

# ---------------------------------------------------------------------------
# Temporary directory — cleaned up on exit
# ---------------------------------------------------------------------------
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# ---------------------------------------------------------------------------
# Utility functions
# ---------------------------------------------------------------------------

info() { printf 'ℹ️  %s\n' "$*"; }
ok()   { printf '✅ %s\n' "$*"; }
err()  { printf '❌ %s\n' "$*" >&2; exit 1; }

require_cmd() {
    for cmd in "$@"; do
        command -v "$cmd" >/dev/null 2>&1 || err "Required command not found: $cmd"
    done
}

# ---------------------------------------------------------------------------
# detect_os — ensure we are on Linux
# ---------------------------------------------------------------------------
detect_os() {
    os=$(uname -s)
    case "$os" in
        Linux) return 0 ;;
        *) err "Unsupported operating system: $os. This script only supports Linux." ;;
    esac
}

# ---------------------------------------------------------------------------
# detect_arch — map uname machine to Go arch
#   x86_64  → amd64
#   aarch64 → arm64
# ---------------------------------------------------------------------------
detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64)  echo "amd64" ;;
        aarch64) echo "arm64" ;;
        *) err "Unsupported architecture: $arch. Only x86_64 and aarch64 are supported." ;;
    esac
}

# ---------------------------------------------------------------------------
# fetch_latest_version — query GitHub API for the latest release tag
# ---------------------------------------------------------------------------
fetch_latest_version() {
    info "Fetching latest release version..."
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | sed 's/.*"tag_name": "//;s/",//')
    if [ -z "$version" ]; then
        err "Failed to fetch latest version from GitHub API."
        err "Check your internet connection or try again later."
    fi
    echo "$version"
}

# ---------------------------------------------------------------------------
# download_release — download binary tarball and checksums.txt
# ---------------------------------------------------------------------------
download_release() {
    version=$1
    arch=$2
    ver_no_v=${version#v}
    tarball="agnostikos_${ver_no_v}_linux_${arch}.tar.gz"
    tarball_url="https://github.com/${REPO}/releases/download/${version}/${tarball}"
    checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    info "Downloading ${tarball}..."
    curl -fsSL "$tarball_url" -o "${TMP_DIR}/${tarball}"

    info "Downloading checksums.txt..."
    curl -fsSL "$checksums_url" -o "${TMP_DIR}/checksums.txt"

    echo "${TMP_DIR}/${tarball}"
}

# ---------------------------------------------------------------------------
# verify_checksum — compare sha256 of downloaded tarball against checksums.txt
# ---------------------------------------------------------------------------
verify_checksum() {
    tarball_path=$1
    tarball_name=$(basename "$tarball_path")
    checksums_file="${TMP_DIR}/checksums.txt"

    info "Verifying SHA256 checksum..."
    expected_hash=$(grep "$tarball_name" "$checksums_file" | awk '{print $1}')
    if [ -z "$expected_hash" ]; then
        err "Checksum for ${tarball_name} not found in checksums.txt."
        err "The release may be corrupted. Try again later."
    fi
    computed_hash=$(sha256sum "$tarball_path" | awk '{print $1}')
    if [ "$expected_hash" != "$computed_hash" ]; then
        err "Checksum mismatch!"
        err "Expected: ${expected_hash}"
        err "Computed: ${computed_hash}"
        err "The downloaded file may be corrupted. Try again."
    fi
    ok "Checksum verified."
}

# ---------------------------------------------------------------------------
# install_binary — extract binary and install to /usr/local/bin
#                  with fallback to ~/.local/bin
# ---------------------------------------------------------------------------
install_binary() {
    tarball_path=$1

    info "Extracting binary..."
    tar -xzf "$tarball_path" -C "$TMP_DIR" "$BINARY"
    if [ ! -f "${TMP_DIR}/${BINARY}" ]; then
        err "Binary '${BINARY}' not found in the archive."
        err "The release may be malformed."
    fi

    # Determine installation directory
    if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
        dest="/usr/local/bin"
    elif mkdir -p /usr/local/bin 2>/dev/null && [ -w /usr/local/bin ]; then
        dest="/usr/local/bin"
    else
        dest="${HOME}/.local/bin"
        mkdir -p "$dest"
    fi

    info "Installing to ${dest}/${BINARY}..."
    install -m 755 "${TMP_DIR}/${BINARY}" "${dest}/${BINARY}"
    ok "Installed to ${dest}/${BINARY}"

    # Remind user if destination is not in PATH
    case ":${PATH}:" in
        *":${dest}:"*) ;;
        *) info "Note: ${dest} is not in your PATH. Add 'export PATH=\"${dest}:\$PATH\"' to your ~/.profile or ~/.bashrc." ;;
    esac
}

# ---------------------------------------------------------------------------
# confirm_installation — run `agnostic --version` to verify
# ---------------------------------------------------------------------------
confirm_installation() {
    info "Verifying installation..."
    if command -v "$BINARY" >/dev/null 2>&1; then
        $BINARY --version
    elif [ -x "/usr/local/bin/${BINARY}" ]; then
        /usr/local/bin/${BINARY} --version
    elif [ -x "${HOME}/.local/bin/${BINARY}" ]; then
        "${HOME}/.local/bin/${BINARY}" --version
    else
        err "Binary not found in PATH. Installation may have failed."
    fi
    ok "${BINARY} is ready to use!"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    require_cmd mktemp curl tar sha256sum uname grep sed awk install

    info "Installing ${BINARY} from ${REPO}..."

    detect_os
    arch=$(detect_arch)
    version=$(fetch_latest_version)
    tarball_path=$(download_release "$version" "$arch")
    verify_checksum "$tarball_path"
    install_binary "$tarball_path"
    confirm_installation

    ok "Installation complete!"
    info "Run '${BINARY} --help' to get started."
}

main "$@"
