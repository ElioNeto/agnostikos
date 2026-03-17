#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ISO="${1:-build/agnostikos-latest.iso}"
RAM="${RAM:-2G}"
CPUS="${CPUS:-2}"

echo -e "${GREEN}[QEMU]${NC} Starting AgnosticOS test..."
echo -e "${GREEN}[QEMU]${NC} ISO: $ISO | RAM: $RAM | CPUs: $CPUS"

[[ ! -f "$ISO" ]] && { echo -e "${RED}[ERROR]${NC} ISO not found: $ISO"; exit 1; }

# KVM detection
KVM=""
[[ -e /dev/kvm ]] && KVM="-enable-kvm" && echo -e "${GREEN}[QEMU]${NC} KVM enabled"

# OVMF detection
OVMF=""
for p in /usr/share/ovmf/OVMF.fd /usr/share/OVMF/OVMF_CODE.fd /usr/share/edk2-ovmf/x64/OVMF_CODE.fd; do
  [[ -f "$p" ]] && OVMF="-bios $p" && echo -e "${GREEN}[QEMU]${NC} UEFI firmware: $p" && break
done

qemu-system-x86_64 \
  $KVM \
  -m "$RAM" \
  -smp "$CPUS" \
  $OVMF \
  -cdrom "$ISO" \
  -boot d \
  -vga virtio \
  -serial mon:stdio \
  -device virtio-net-pci,netdev=net0 \
  -netdev user,id=net0 \
  -no-reboot

echo -e "${GREEN}[QEMU]${NC} VM stopped"
