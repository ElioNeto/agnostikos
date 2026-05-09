#!/usr/bin/env bash
# prepare-minimal-rootfs-arm64.sh
#
# Creates a minimal ARM64 rootfs for ISO boot testing under QEMU.
# Downloads an ARM64 kernel and busybox static binary, then builds
# an initramfs with a shell-based init script.
#
# Usage:
#   bash scripts/prepare-minimal-rootfs-arm64.sh
#
# Output:
#   MINIMAL_ROOT pointing to /mnt/data/agnostikOS/minimal-rootfs-arm64
#   The boot/ directory will contain:
#     - Image-<version>  (ARM64 kernel)
#     - initramfs.img    (gzip-cpio archive with busybox + init script)
#
# Prerequisites:
#   - wget or curl
#   - cpio, gzip

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NO_COLOR='\033[0m'

AGNOSTICOS_BASE="${AGNOSTICOS_BASE:-/mnt/data/agnostikOS}"
MINIMAL_ROOT="${AGNOSTICOS_BASE}/minimal-rootfs-arm64"
BOOT_DIR="${MINIMAL_ROOT}/boot"

KERNEL_VERSION="${KERNEL_VERSION:-6.8}"
BUSYBOX_VERSION="${BUSYBOX_VERSION:-1.36.1}"

echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Creating minimal ARM64 rootfs at ${MINIMAL_ROOT}"

# Ensure base directory exists (may need sudo on CI runners)
if ! mkdir -p "${AGNOSTICOS_BASE}" 2>/dev/null; then
  sudo mkdir -p "${AGNOSTICOS_BASE}"
  sudo chown "$(id -u):$(id -g)" "${AGNOSTICOS_BASE}"
fi

rm -rf "${MINIMAL_ROOT}"
mkdir -p "${BOOT_DIR}" "${MINIMAL_ROOT}/bin" "${MINIMAL_ROOT}/dev" \
  "${MINIMAL_ROOT}/proc" "${MINIMAL_ROOT}/sys"

# ---------------------------------------------------------------------------
# Step 1: Download ARM64 kernel
# ---------------------------------------------------------------------------
KERNEL_DEST="${BOOT_DIR}/Image-${KERNEL_VERSION}"

if [ -f "${KERNEL_DEST}" ]; then
  echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} Kernel already exists at ${KERNEL_DEST}, skipping download"
else
  echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Downloading ARM64 kernel v${KERNEL_VERSION}..."

  # Try Ubuntu's cloud kernel first, then fall back to generic URLs
  KERNEL_URL="https://cloud-images.ubuntu.com/releases/noble/release/unpacked/ubuntu-24.04-server-cloudimg-arm64-vmlinuz-generic"

  if command -v wget &>/dev/null; then
    wget -q --timeout=30 -O "${KERNEL_DEST}" "${KERNEL_URL}" || {
      echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} Primary kernel URL failed, trying kernel.org..."
      # Download a generic ARM64 kernel from kernel.org (this is a compressed image)
      local KERNEL_TAR="/tmp/linux-${KERNEL_VERSION}-arm64.tar.gz"
      wget -q --timeout=30 \
        "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${KERNEL_VERSION}.tar.xz" \
        -O "/tmp/linux-${KERNEL_VERSION}.tar.xz" || {
        echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Failed to download kernel source"
        exit 1
      }
      echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} Kernel source downloaded but compilation is too heavy for CI"
      echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} Install a pre-built kernel or run this script on an ARM64 machine"
      exit 1
    }
  elif command -v curl &>/dev/null; then
    curl -sfL --connect-timeout 30 -o "${KERNEL_DEST}" "${KERNEL_URL}" || {
      echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Failed to download ARM64 kernel"
      exit 1
    }
  else
    echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Neither wget nor curl found"
    exit 1
  fi

  # Verify it's a valid kernel image
  KERNEL_SIZE=$(stat -c%s "${KERNEL_DEST}" 2>/dev/null || stat -f%z "${KERNEL_DEST}" 2>/dev/null)
  if [ "${KERNEL_SIZE}" -lt 1000000 ]; then
    echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} Kernel file suspiciously small (${KERNEL_SIZE} bytes), but continuing"
  else
    echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Kernel downloaded (${KERNEL_SIZE} bytes)"
  fi
fi

# ---------------------------------------------------------------------------
# Step 2: Download ARM64 busybox static binary
# ---------------------------------------------------------------------------
BUSYBOX_DEST="${MINIMAL_ROOT}/bin/busybox"

if [ -f "${BUSYBOX_DEST}" ]; then
  echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} Busybox already exists, skipping download"
else
  echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Downloading ARM64 busybox ${BUSYBOX_VERSION}..."

  BUSYBOX_URL="https://busybox.net/downloads/binaries/${BUSYBOX_VERSION}/busybox-arm64"

  if command -v wget &>/dev/null; then
    wget -q --timeout=30 -O "${BUSYBOX_DEST}" "${BUSYBOX_URL}" || {
      echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Failed to download ARM64 busybox from ${BUSYBOX_URL}"
      # Try alternative: use busybox.net's list of binaries
      BUSYBOX_URL_ALT="https://busybox.net/downloads/binaries/${BUSYBOX_VERSION}/busybox-armv8l"
      wget -q --timeout=30 -O "${BUSYBOX_DEST}" "${BUSYBOX_URL_ALT}" || {
        echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Failed to download ARM64 busybox from fallback URL"
        exit 1
      }
    }
  elif command -v curl &>/dev/null; then
    curl -sfL --connect-timeout 30 -o "${BUSYBOX_DEST}" "${BUSYBOX_URL}" || {
      echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Failed to download ARM64 busybox"
      exit 1
    }
  else
    echo -e "${RED}[prepare-minimal-rootfs-arm64]${NO_COLOR} Neither wget nor curl found"
    exit 1
  fi

  chmod +x "${BUSYBOX_DEST}"
  echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Busybox downloaded and made executable"
fi

# ---------------------------------------------------------------------------
# Step 3: Create busybox symlinks for required applets
# ---------------------------------------------------------------------------
"${BUSYBOX_DEST}" --install -s "${MINIMAL_ROOT}/bin/" 2>/dev/null || {
  # If busybox --install fails (wrong arch emulation), create symlinks manually
  echo -e "${YELLOW}[prepare-minimal-rootfs-arm64]${NO_COLOR} busybox --install failed (expected without QEMU binfmt), creating symlinks manually"
  for applet in sh mount poweroff uname cat echo ls; do
    ln -sf busybox "${MINIMAL_ROOT}/bin/${applet}" 2>/dev/null || true
  done
}

# ---------------------------------------------------------------------------
# Step 4: Create init script
# ---------------------------------------------------------------------------
cat > "${MINIMAL_ROOT}/init" << 'INIT'
#!/bin/sh
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
echo ""
echo "================================================"
echo "  Welcome to Agnostikos (ARM64)"
echo "  Kernel: $(uname -r)"
echo "================================================"
echo ""
poweroff -f
INIT
chmod +x "${MINIMAL_ROOT}/init"

# ---------------------------------------------------------------------------
# Step 5: Create initramfs
# ---------------------------------------------------------------------------
INITRAMFS_DEST="${BOOT_DIR}/initramfs.img"
echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Creating initramfs at ${INITRAMFS_DEST}..."

cd "${MINIMAL_ROOT}"
# Exclude the boot directory itself from the initramfs (kernel is loaded separately)
find . -path ./boot -prune -o -print | cpio -o -H newc | gzip > "${INITRAMFS_DEST}"
cd - > /dev/null

INITRAMFS_SIZE=$(stat -c%s "${INITRAMFS_DEST}" 2>/dev/null || stat -f%z "${INITRAMFS_DEST}" 2>/dev/null)
echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Initramfs created (${INITRAMFS_SIZE} bytes)"

# ---------------------------------------------------------------------------
# Step 6: Verify
# ---------------------------------------------------------------------------
echo ""
echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} Minimal ARM64 rootfs ready at:"
echo "  ${MINIMAL_ROOT}"
echo "  boot/Image-${KERNEL_VERSION}"
echo "  boot/initramfs.img"
echo "  bin/busybox (ARM64 static)"
echo "  init        (shell script)"
echo ""
echo -e "${GREEN}[prepare-minimal-rootfs-arm64]${NO_COLOR} To build an ISO and test:"
echo "  ./build/agnostic iso --rootfs ${MINIMAL_ROOT} --output /tmp/agnostikos-arm64.iso --uefi"
echo "  qemu-system-aarch64 -machine virt -cpu cortex-a57 -m 2G -smp 2 \\"
echo "    -drive if=pflash,format=raw,readonly=on,file=/usr/share/AAVMF/AAVMF_CODE.fd \\"
echo "    -drive if=pflash,format=raw,file=/tmp/arm64-vars.fd \\"
echo "    -cdrom /tmp/agnostikos-arm64.iso -nographic"
echo ""
