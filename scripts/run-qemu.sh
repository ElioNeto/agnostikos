#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NO_COLOR='\033[0m'

ISO="${1:-build/agnostikos-latest.iso}"
RAM="${RAM:-2G}"
CPUS="${CPUS:-2}"
HEADLESS="${HEADLESS:-0}"
BOOT_TIMEOUT="${BOOT_TIMEOUT:-120}"

echo -e "${GREEN}[QEMU]${NO_COLOR} Starting AgnosticOS test..."
echo -e "${GREEN}[QEMU]${NO_COLOR} ISO: $ISO | RAM: $RAM | CPUs: $CPUS | Headless: $HEADLESS | Timeout: ${BOOT_TIMEOUT}s"

[[ ! -f "$ISO" ]] && { echo -e "${RED}[ERROR]${NO_COLOR} ISO not found: $ISO"; exit 1; }

# KVM detection
KVM_FLAG=""
[[ -e /dev/kvm ]] && KVM_FLAG="-enable-kvm" && echo -e "${GREEN}[QEMU]${NO_COLOR} KVM enabled"

# OVMF detection
OVMF_FLAG=""
for p in /usr/share/ovmf/OVMF.fd /usr/share/OVMF/OVMF_CODE.fd /usr/share/edk2-ovmf/x64/OVMF_CODE.fd; do
  [[ -f "$p" ]] && OVMF_FLAG="-bios $p" && echo -e "${GREEN}[QEMU]${NO_COLOR} UEFI firmware: $p" && break
done

# Display mode: headless for CI, graphical otherwise
if [[ "$HEADLESS" == "1" ]]; then
  DISPLAY_FLAGS="-display none -serial stdio"
  echo -e "${GREEN}[QEMU]${NO_COLOR} Running headless (CI mode)"
else
  DISPLAY_FLAGS="-vga virtio -serial mon:stdio"
fi

# Run with timeout to avoid hanging CI
timeout "${BOOT_TIMEOUT}" qemu-system-x86_64 \
  $KVM_FLAG \
  -m "$RAM" \
  -smp "$CPUS" \
  $OVMF_FLAG \
  -cdrom "$ISO" \
  -boot d \
  $DISPLAY_FLAGS \
  -device virtio-net-pci,netdev=net0 \
  -netdev user,id=net0 \
  -no-reboot || {
    EXIT=$?
    if [[ $EXIT -eq 124 ]]; then
      echo -e "${RED}[ERROR]${NO_COLOR} Boot timeout after ${BOOT_TIMEOUT}s"
      exit 1
    fi
    echo -e "${GREEN}[QEMU]${NO_COLOR} VM stopped (exit $EXIT)"
  }

echo -e "${GREEN}[QEMU]${NO_COLOR} Done"
