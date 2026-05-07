#!/usr/bin/env bash
# Package build script for AgnosticOS
# Produces .deb and .rpm packages using GoReleaser's nfpm or manual fpm.
# Prerequisites: gem install fpm
set -euo pipefail

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="${2:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
ARCH="${3:-amd64}"
BUILD_DIR="/mnt/data/agnostikOS/build"
PKG_DIR="${BUILD_DIR}/packaging"
BINARY_NAME="agnostic"

echo "[package] Building AgnosticOS ${VERSION} (commit: ${COMMIT}, arch: ${ARCH})"

# Ensure fpm is available
if ! command -v fpm &>/dev/null; then
    echo "[package] fpm not found. Install with: gem install fpm"
    echo "[package] Falling back to manual deb/rpm structure..."
    # Build .deb manually using dpkg-deb
    build_deb_manual() {
        local deb_dir="${PKG_DIR}/deb"
        mkdir -p "${deb_dir}/DEBIAN"
        mkdir -p "${deb_dir}/usr/local/bin"

        cat > "${deb_dir}/DEBIAN/control" <<CONTROL
Package: agnostikos
Version: ${VERSION#v}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: ElioNeto <elioneto@users.noreply.github.com>
Description: AgnosticOS Hybrid Package Manager
 A meta-wrapper for Pacman, Nix, and Flatpak backends
 under a unified CLI.
Homepage: https://github.com/ElioNeto/agnostikos
CONTROL

        cp "${BUILD_DIR}/${BINARY_NAME}" "${deb_dir}/usr/local/bin/agnostic"
        chmod 755 "${deb_dir}/usr/local/bin/agnostic"

        dpkg-deb --build "${deb_dir}" "${BUILD_DIR}/agnostikos_${VERSION#v}_${ARCH}.deb"
        echo "[package] .deb created: ${BUILD_DIR}/agnostikos_${VERSION#v}_${ARCH}.deb"
    }

    build_rpm_manual() {
        local rpm_dir="${PKG_DIR}/rpm"
        mkdir -p "${rpm_dir}/usr/local/bin"

        cp "${BUILD_DIR}/${BINARY_NAME}" "${rpm_dir}/usr/local/bin/agnostic"
        chmod 755 "${rpm_dir}/usr/local/bin/agnostic"

        # Use rpmbuild if available
        if command -v rpmbuild &>/dev/null; then
            local spec_dir="${PKG_DIR}/spec"
            mkdir -p "${spec_dir}"
            cat > "${spec_dir}/agnostikos.spec" <<SPEC
Name: agnostikos
Version: ${VERSION#v}
Release: 1
Summary: AgnosticOS Hybrid Package Manager
License: MIT
URL: https://github.com/ElioNeto/agnostikos
Group: System Environment/Base
BuildArch: ${ARCH}

%description
A meta-wrapper for Pacman, Nix, and Flatpak backends
under a unified CLI.

%install
mkdir -p %{buildroot}/usr/local/bin
cp ${BUILD_DIR}/${BINARY_NAME} %{buildroot}/usr/local/bin/agnostic
chmod 755 %{buildroot}/usr/local/bin/agnostic

%files
/usr/local/bin/agnostic
SPEC
            rpmbuild --define "_builddir ${rpm_dir}" \
                     --define "_rpmdir ${BUILD_DIR}" \
                     -bb "${spec_dir}/agnostikos.spec"
            echo "[package] .rpm created (check ${BUILD_DIR} for x86_64/*.rpm)"
        else
            echo "[package] rpmbuild not found. Cannot build .rpm manually."
            echo "[package] Install rpm-build or use: gem install fpm"
        fi
    }

    build_deb_manual
    build_rpm_manual
else
    # Use fpm for both .deb and .rpm
    echo "[package] Building .deb with fpm..."
    fpm -s dir -t deb \
        -n agnostikos \
        -v "${VERSION#v}" \
        --architecture "${ARCH}" \
        --description "AgnosticOS Hybrid Package Manager" \
        --url "https://github.com/ElioNeto/agnostikos" \
        --maintainer "ElioNeto <elioneto@users.noreply.github.com>" \
        --license "MIT" \
        "${BUILD_DIR}/${BINARY_NAME}"=/usr/local/bin/agnostic

    echo "[package] Building .rpm with fpm..."
    fpm -s dir -t rpm \
        -n agnostikos \
        -v "${VERSION#v}" \
        --architecture "${ARCH}" \
        --description "AgnosticOS Hybrid Package Manager" \
        --url "https://github.com/ElioNeto/agnostikos" \
        --maintainer "ElioNeto <elioneto@users.noreply.github.com>" \
        --license "MIT" \
        "${BUILD_DIR}/${BINARY_NAME}"=/usr/local/bin/agnostic
fi

echo "[package] Done."
