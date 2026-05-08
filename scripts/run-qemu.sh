#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NO_COLOR='\033[0m'

if ! command -v qemu-system-x86_64 &>/dev/null; then
    echo -e "${RED}[ERROR]${NO_COLOR} qemu-system-x86_64 not found. Install with: apt install qemu-system-x86"
    exit 1
fi

ISO="${1:-build/agnostikos-latest.iso}"
RAM="${RAM:-2G}"
CPUS="${CPUS:-2}"
HEADLESS="${HEADLESS:-0}"
# Timeout defaults: 30s headless, 300s interactive
# Override via BOOT_TIMEOUT environment variable
if [[ "$HEADLESS" == "1" ]]; then
  BOOT_TIMEOUT="${BOOT_TIMEOUT:-30}"
else
  BOOT_TIMEOUT="${BOOT_TIMEOUT:-300}"
fi
SERIAL_LOG="/tmp/qemu-serial-$$.log"

echo -e "${GREEN}[QEMU]${NO_COLOR} Starting AgnosticOS test..."
echo -e "${GREEN}[QEMU]${NO_COLOR} ISO: $ISO | RAM: $RAM | CPUs: $CPUS | Headless: $HEADLESS | Timeout: ${BOOT_TIMEOUT}s"

[[ ! -f "$ISO" ]] && { echo -e "${RED}[ERROR]${NO_COLOR} ISO not found: $ISO"; exit 1; }

# KVM detection: check readable (not just exists) to avoid permission denied
KVM_FLAG=""
if [[ -r /dev/kvm ]]; then
    KVM_FLAG="-enable-kvm"
    echo -e "${GREEN}[QEMU]${NO_COLOR} KVM acceleration enabled"
else
    echo -e "${GREEN}[INFO]${NO_COLOR} KVM not available — using TCG emulation (slower)"
fi

# UEFI firmware detection
# Correct method: -drive if=pflash (not -bios which runs CSM/compat mode)
# Split OVMF (CODE+VARS) is preferred; monolithic OVMF.fd works too via pflash.
OVMF_FLAGS=""
VARS_TMP=""

if [[ -f /usr/share/OVMF/OVMF_CODE.fd && -f /usr/share/OVMF/OVMF_VARS.fd ]]; then
  VARS_TMP="$(mktemp /tmp/OVMF_VARS_XXXXXX.fd)"
  cp /usr/share/OVMF/OVMF_VARS.fd "$VARS_TMP"
  OVMF_FLAGS="-drive if=pflash,format=raw,readonly=on,file=/usr/share/OVMF/OVMF_CODE.fd \
-drive if=pflash,format=raw,file=${VARS_TMP}"
  echo -e "${GREEN}[QEMU]${NO_COLOR} UEFI firmware: OVMF split (CODE+VARS)"
elif [[ -f /usr/share/ovmf/OVMF.fd ]]; then
  VARS_TMP="$(mktemp /tmp/OVMF_XXXXXX.fd)"
  cp /usr/share/ovmf/OVMF.fd "$VARS_TMP"
  OVMF_FLAGS="-drive if=pflash,format=raw,readonly=on,file=/usr/share/ovmf/OVMF.fd \
-drive if=pflash,format=raw,file=${VARS_TMP}"
  echo -e "${GREEN}[QEMU]${NO_COLOR} UEFI firmware: OVMF monolithic (pflash)"
elif [[ -f /usr/share/edk2-ovmf/x64/OVMF_CODE.fd ]]; then
  VARS_TMP="$(mktemp /tmp/OVMF_VARS_XXXXXX.fd)"
  cp /usr/share/edk2-ovmf/x64/OVMF_VARS.fd "$VARS_TMP"
  OVMF_FLAGS="-drive if=pflash,format=raw,readonly=on,file=/usr/share/edk2-ovmf/x64/OVMF_CODE.fd \
-drive if=pflash,format=raw,file=${VARS_TMP}"
  echo -e "${GREEN}[QEMU]${NO_COLOR} UEFI firmware: edk2 OVMF (pflash)"
else
  echo -e "${RED}[WARN]${NO_COLOR} UEFI firmware not found — falling back to legacy BIOS"
fi

# CD-ROM via AHCI — OVMF enumerates AHCI devices properly for El Torito EFI
# -cdrom uses IDE legacy which OVMF may not register as a UEFI boot option
CDROM_FLAGS="-device ahci,id=ahci0 \
-drive id=cdrom0,if=none,format=raw,readonly=on,file=${ISO} \
-device ide-cd,bus=ahci0.0,drive=cdrom0"

# Display mode
if [[ "$HEADLESS" == "1" ]]; then
  DISPLAY_FLAGS="-display none -serial file:${SERIAL_LOG}"
  echo -e "${GREEN}[QEMU]${NO_COLOR} Running headless (CI mode) - serial log: $SERIAL_LOG"
else
  # Interactive mode: serial no terminal, sem janela gráfica.
  # O usuário vê as mensagens de boot e o shell do sistema, podendo digitar
  # comandos diretamente. 'exit' no shell encerra a VM.
  # Usar -serial stdio (não -nographic) para evitar problemas de terminal raw.
  # Ctrl+C no terminal mata o QEMU se necessário.
  DISPLAY_FLAGS="-display none -serial stdio"
  echo -e "${GREEN}[QEMU]${NO_COLOR} Running interactively (serial via stdio) - type 'exit' in guest to power off"
fi

cleanup() {
  [[ -n "$VARS_TMP" && -f "$VARS_TMP" ]] && rm -f "$VARS_TMP"
  [[ "$HEADLESS" == "1" && -f "$SERIAL_LOG" ]] && rm -f "$SERIAL_LOG"
  return 0
}

# check_boot_output checks serial log for welcome message and exits accordingly
check_boot_output() {
  if [[ "$HEADLESS" != "1" ]]; then
    return 0
  fi
  echo ""
  echo -e "${GREEN}[BOOT TEST]${NO_COLOR} Analyzing serial output..."

  if [[ ! -f "$SERIAL_LOG" ]]; then
    echo -e "${RED}[BOOT TEST]${NO_COLOR} Serial log not found: $SERIAL_LOG"
    echo -e "${RED}[BOOT TEST]${NO_COLOR} FAIL"
    return 1
  fi

  # Check for kernel panic first (always a failure)
  if grep -qi "kernel panic\|Kernel Panic" "$SERIAL_LOG"; then
    echo -e "${RED}[BOOT TEST]${NO_COLOR} KERNEL PANIC detected in serial output"
    echo -e "${RED}[BOOT TEST]${NO_COLOR} FAIL"
    tail -50 "$SERIAL_LOG" || true
    return 1
  fi

  # Primary check: welcome message
  if grep -q "Welcome to Agnostikos" "$SERIAL_LOG"; then
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} Welcome message found in serial output"
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} PASS"
    return 0
  fi

  # Fallback check: look for kernel boot messages indicating successful boot
  if grep -q "Linux version\|init started\|Freeing unused kernel memory" "$SERIAL_LOG"; then
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} Kernel boot messages detected (fallback)"
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} PASS"
    return 0
  fi

  # Fallback check: look for shell prompt or busybox init
  if grep -q "/bin/sh\|init started\|/ #" "$SERIAL_LOG"; then
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} Shell/interactive prompt detected (fallback)"
    echo -e "${GREEN}[BOOT TEST]${NO_COLOR} PASS"
    return 0
  fi

  echo -e "${RED}[BOOT TEST]${NO_COLOR} No boot indicators found in serial output"
  echo -e "${RED}[BOOT TEST]${NO_COLOR} Last 50 lines of serial output:"
  tail -50 "$SERIAL_LOG" || true
  echo -e "${RED}[BOOT TEST]${NO_COLOR} FAIL"
  return 1
}

# Roda QEMU
if [[ "$HEADLESS" == "1" ]]; then
  # Headless: timeout + serial log capture + boot validation
  timeout "${BOOT_TIMEOUT}" qemu-system-x86_64 \
    $KVM_FLAG \
    -m "$RAM" \
    -smp "$CPUS" \
    $OVMF_FLAGS \
    $CDROM_FLAGS \
    $DISPLAY_FLAGS \
    -device virtio-net-pci,netdev=net0 \
    -netdev user,id=net0 \
    -no-reboot || {
      EXIT=$?
      if [[ $EXIT -eq 124 || $EXIT -eq 130 ]]; then
        echo -e "${RED}[QEMU]${NO_COLOR} Boot timeout after ${BOOT_TIMEOUT}s"
        if [[ -f "$SERIAL_LOG" ]]; then
          echo -e "${RED}[DEBUG]${NO_COLOR} Serial log (${SERIAL_LOG}):"
          echo "--- BEGIN SERIAL OUTPUT ---"
          cat "$SERIAL_LOG" || true
          echo "--- END SERIAL OUTPUT ---"
        fi
        # Check boot output even on timeout — boot may have been successful
        # but the shell keeps running (no poweroff in initramfs)
        if check_boot_output; then
          cleanup
          exit 0
        fi
        cleanup
        exit 1
      fi
      echo -e "${GREEN}[QEMU]${NO_COLOR} VM stopped (exit $EXIT)"
    }

  echo -e "${GREEN}[QEMU]${NO_COLOR} QEMU exited normally"

  # Boot output validation (headless mode, on normal exit)
  check_boot_output && RESULT=0 || RESULT=$?
  cleanup
  exit $RESULT
else
  # Interactive: no timeout, user controls exit via 'exit' or Ctrl+A X
  # Padrão &&/|| para capturar exit code sem disparar set -e
  qemu-system-x86_64 \
    $KVM_FLAG \
    -m "$RAM" \
    -smp "$CPUS" \
    $OVMF_FLAGS \
    $CDROM_FLAGS \
    $DISPLAY_FLAGS \
    -device virtio-net-pci,netdev=net0 \
    -netdev user,id=net0 \
    -no-reboot && QEMU_EXIT=0 || QEMU_EXIT=$?
  cleanup
  exit $QEMU_EXIT
fi
