#!/usr/bin/env bash
# Integration test: full pipeline bootstrap → iso → qemu boot validation
# Usage: ./scripts/test-boot-integration.sh [--uefi] [--timeout N]
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NO_COLOR='\033[0m'

UEFI=0
TIMEOUT=300
MODE="BIOS"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --uefi) UEFI=1; MODE="UEFI"; shift ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    *) echo -e "${RED}[ERROR]${NO_COLOR} Unknown option: $1"; exit 1 ;;
  esac
done

echo -e "${GREEN}================================================${NO_COLOR}"
echo -e "${GREEN}  Agnostikos Boot Integration Test (${MODE})${NO_COLOR}"
echo -e "${GREEN}================================================${NO_COLOR}"
echo ""

ISO="${BUILD_DIR:-build}/agnostikos-latest.iso"
BASE_DIR="/mnt/data/agnostikOS"
LFS="${BASE_DIR}/rootfs"

# Ensure build directory exists
mkdir -p build

# Step 1: Build the Go binary
echo -e "${YELLOW}[STEP 1]${NO_COLOR} Building Go binary..."
if ! make build; then
    echo -e "${RED}[FAIL]${NO_COLOR} Build failed"
    exit 1
fi
echo -e "${GREEN}[OK]${NO_COLOR} Build succeeded"
echo ""

# Step 2: Bootstrap (with GRUB, skip toolchain to save time)
echo -e "${YELLOW}[STEP 2]${NO_COLOR} Bootstrapping RootFS..."
if [[ "$UEFI" == "1" ]]; then
    ARGS="--force --skip-toolchain --uefi"
else
    ARGS="--force --skip-toolchain"
fi

if ! sudo make bootstrap ARGS="$ARGS"; then
    echo -e "${RED}[FAIL]${NO_COLOR} Bootstrap failed"
    exit 1
fi
echo -e "${GREEN}[OK]${NO_COLOR} Bootstrap succeeded"
echo ""

# Step 3: Build ISO
echo -e "${YELLOW}[STEP 3]${NO_COLOR} Building ISO..."
ISO_ARGS=""
if [[ "$UEFI" == "1" ]]; then
    ISO_ARGS="--uefi"
fi

if ! make iso ARGS="$ISO_ARGS"; then
    echo -e "${RED}[FAIL]${NO_COLOR} ISO build failed"
    exit 1
fi
echo -e "${GREEN}[OK]${NO_COLOR} ISO created at ${ISO}"
echo ""

# Step 4: Run QEMU and validate boot
echo -e "${YELLOW}[STEP 4]${NO_COLOR} Testing boot in QEMU (${MODE}, timeout: ${TIMEOUT}s)..."
echo -e "${YELLOW}[INFO]${NO_COLOR} Using TCG emulation (no KVM required)"

UEFI_FLAGS=""
if [[ "$UEFI" == "1" ]]; then
    UEFI_FLAGS="--uefi"
fi

if HEADLESS=1 BOOT_TIMEOUT="${TIMEOUT}" bash scripts/run-qemu.sh "$ISO" $UEFI_FLAGS; then
    echo -e "${GREEN}[STEP 4]${NO_COLOR} Boot test PASSED"
else
    echo -e "${RED}[STEP 4]${NO_COLOR} Boot test FAILED"
    exit 1
fi

echo ""
echo -e "${GREEN}================================================${NO_COLOR}"
echo -e "${GREEN}  Integration Test PASSED (${MODE})${NO_COLOR}"
echo -e "${GREEN}================================================${NO_COLOR}"
