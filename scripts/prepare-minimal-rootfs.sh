#!/usr/bin/env bash
# prepare-minimal-rootfs.sh
#
# Cria um rootfs mínimo para teste de boot ISO (sem toolchain completa).
# Detecta o kernel do host em /boot/vmlinuz-* e o copia/symlinka para
# /mnt/data/agnostikOS/minimal-rootfs/boot/vmlinuz-<version>.
#
# Uso:
#   bash scripts/prepare-minimal-rootfs.sh
#
# Saída:
#   MINIMAL_ROOT apontando para /mnt/data/agnostikOS/minimal-rootfs
#   O diretório boot/ conterá o kernel detectado.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NO_COLOR='\033[0m'

AGNOSTICOS_BASE="${AGNOSTICOS_BASE:-/mnt/data/agnostikOS}"
MINIMAL_ROOT="${AGNOSTICOS_BASE}/minimal-rootfs"
BOOT_DIR="${MINIMAL_ROOT}/boot"

echo -e "${GREEN}[prepare-minimal-rootfs]${NO_COLOR} Creating minimal rootfs at ${MINIMAL_ROOT}"

# Ensure base directory exists
mkdir -p "${AGNOSTICOS_BASE}"

# Ensure boot directory exists
mkdir -p "${BOOT_DIR}"

# Detect host kernel
KERNEL_SRC=""
# Try common kernel paths
for pattern in "/boot/vmlinuz-"* "/boot/vmlinux-"*; do
  for f in $pattern; do
    if [ -f "$f" ]; then
      KERNEL_SRC="$f"
      break 2
    fi
  done
done

if [ -z "$KERNEL_SRC" ]; then
  echo -e "${RED}[prepare-minimal-rootfs]${NO_COLOR} No kernel found in /boot/vmlinuz-*"
  echo -e "${RED}[prepare-minimal-rootfs]${NO_COLOR} Cannot create minimal rootfs without a kernel."
  echo -e "${YELLOW}[prepare-minimal-rootfs]${NO_COLOR} Install a kernel first or run 'make bootstrap'."
  exit 1
fi

# Extract kernel version from filename
KERNEL_FILENAME=$(basename "$KERNEL_SRC")
# Strip vmlinuz- or vmlinux- prefix to get version
KERNEL_VERSION="${KERNEL_FILENAME#vmlinuz-}"
KERNEL_VERSION="${KERNEL_VERSION#vmlinux-}"

echo -e "${GREEN}[prepare-minimal-rootfs]${NO_COLOR} Host kernel detected: ${KERNEL_SRC}"
echo -e "${GREEN}[prepare-minimal-rootfs]${NO_COLOR} Kernel version: ${KERNEL_VERSION}"

# Copy kernel to minimal rootfs boot directory
DEST_KERNEL="${BOOT_DIR}/vmlinuz-${KERNEL_VERSION}"

if [ -f "$DEST_KERNEL" ]; then
  echo -e "${YELLOW}[prepare-minimal-rootfs]${NO_COLOR} Kernel already exists at ${DEST_KERNEL}, skipping copy"
else
  echo -e "${GREEN}[prepare-minimal-rootfs]${NO_COLOR} Copying kernel to ${DEST_KERNEL}"
  cp -f "$KERNEL_SRC" "$DEST_KERNEL"
  chmod 644 "$DEST_KERNEL"
fi

# Verify the kernel was placed correctly
if [ ! -f "$DEST_KERNEL" ]; then
  echo -e "${RED}[prepare-minimal-rootfs]${NO_COLOR} Failed to place kernel at ${DEST_KERNEL}"
  exit 1
fi

# Print summary
echo ""
echo -e "${GREEN}[prepare-minimal-rootfs]${NO_COLOR} Minimal rootfs ready at:"
echo "  ${MINIMAL_ROOT}"
echo "  boot/vmlinuz-${KERNEL_VERSION}"
echo ""
echo -e "${GREEN}[prepare-minimal-rootfs]${NO_COLOR} Use it with:"
echo "  make minimal-iso"
echo "  make minimal-iso ARGS=\"--kernel-version ${KERNEL_VERSION}\""
echo ""

# Export the minimal root path for easy use
echo "MINIMAL_ROOT=${MINIMAL_ROOT}"
