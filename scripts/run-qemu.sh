#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NO_COLOR='\033[0m'

# QEMU availability check
if ! command -v qemu-system-x86_64 &>/dev/null; then
    echo -e "${RED}[ERROR]${NO_COLOR} qemu-system-x86_64 not found. Install with: apt install qemu-system-x86"
    exit 1
fi

ISO="${1:-build/agnostikos-latest.iso}"
RAM="${RAM:-2G}"
CPUS="${CPUS:-2}"
HEADLESS="${HEADLESS:-0}"
BOOT_TIMEOUT="${BOOT_TIMEOUT:-120}"
SERIAL_LOG="/tmp/qemu-serial-$$.log"

echo -e "${GREEN}[QEMU]${NO_COLOR} Starting AgnosticOS test..."
echo -e "${GREEN}[QEMU]${NO_COLOR} ISO: $ISO | RAM: $RAM | CPUs: $CPUS | Headless: $HEADLESS | Timeout: ${BOOT_TIMEOUT}s"

[[ ! -f "$ISO" ]] && { echo -e "${RED}[ERROR]${NO_COLOR} ISO not found: $ISO"; exit 1; }

# KVM detection
KVM_FLAG=""
if [[ -e /dev/kvm ]]; then
    KVM_FLAG="-enable-kvm"
    echo -e "${GREEN}[QEMU]${NO_COLOR} KVM acceleration enabled"
else
    KVM_FLAG=""
    echo -e "${GREEN}[INFO]${NO_COLOR} KVM not available — using TCG emulation (slower)"
fi

# OVMF detection
OVMF_FLAG=""
for p in /usr/share/ovmf/OVMF.fd /usr/share/OVMF/OVMF_CODE.fd /usr/share/edk2-ovmf/x64/OVMF_CODE.fd; do
  [[ -f "$p" ]] && OVMF_FLAG="-bios $p" && echo -e "${GREEN}[QEMU]${NO_COLOR} UEFI firmware: $p" && break
done

# Display mode
if [[ "$HEADLESS" == "1" ]]; then
  DISPLAY_FLAGS="-display none -serial file:${SERIAL_LOG}"
  echo -e "${GREEN}[QEMU]${NO_COLOR} Running headless (CI mode) - serial log: $SERIAL_LOG"
else
  DISPLAY_FLAGS="-vga virtio -serial mon:stdio"
fi

# Cleanup: remover log apenas após validação (não no EXIT)
cleanup_log() {
  [[ "$HEADLESS" == "1" ]] && rm -f "$SERIAL_LOG"
}

# Roda QEMU com timeout
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
    if [[ $EXIT -eq 124 || $EXIT -eq 130 ]]; then
      echo -e "${RED}[ERROR]${NO_COLOR} Boot timeout after ${BOOT_TIMEOUT}s"
      if [[ "$HEADLESS" == "1" && -f "$SERIAL_LOG" ]]; then
        echo -e "${RED}[DEBUG]${NO_COLOR} Serial log (${SERIAL_LOG}):"
        echo "--- BEGIN SERIAL OUTPUT ---"
        cat "$SERIAL_LOG" || true
        echo "--- END SERIAL OUTPUT ---"
      fi
      cleanup_log
      exit 1
    fi
    echo -e "${GREEN}[QEMU]${NO_COLOR} VM stopped (exit $EXIT)"
  }

echo -e "${GREEN}[QEMU]${NO_COLOR} Done"

# Boot output validation (headless mode only)
if [[ "$HEADLESS" == "1" ]]; then
  echo ""
  echo -e "${GREEN}[BOOT TEST]${NO_COLOR} Analyzing serial output..."

  if [[ ! -f "$SERIAL_LOG" ]]; then
    echo -e "${RED}[BOOT TEST]${NO_COLOR} Serial log not found: $SERIAL_LOG"
    echo -e "${RED}[BOOT TEST]${NO_COLOR} FAIL"
    exit 1
  fi

  if grep -q "Welcome to Agnostikos" "$SERIAL_LOG"; then
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} Welcome message found in serial output"
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} PASS"
  else
    echo -e "${RED}[BOOT TEST]${NO_COLOR} Welcome message NOT found in serial output"
    echo -e "${RED}[BOOT TEST]${NO_COLOR} Last 50 lines of serial output:"
    tail -50 "$SERIAL_LOG"
    echo -e "${RED}[BOOT TEST]${NO_COLOR} FAIL"
    cleanup_log
    exit 1
  fi

  cleanup_log
fi
