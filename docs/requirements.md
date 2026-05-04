# Host Dependencies

This document lists the host system dependencies required to build and test the AgnosticOS project.

## Required Tools

- **Go** >= 1.22 — for building the CLI and running unit tests
- **GNU Make** — for running build targets
- **git** — for version control

## QEMU/KVM Testing

The `test-iso` and `test-iso-headless` Make targets run the built ISO inside QEMU. These require:

### Packages (Debian/Ubuntu)

```bash
sudo apt-get install -y qemu-system-x86 ovmf
```

- `qemu-system-x86` — QEMU x86_64 system emulator
- `ovmf` — UEFI firmware for QEMU (provides OVMF.fd / OVMF_CODE.fd)

### KVM Acceleration (optional)

For better performance, KVM is auto-detected at runtime:

```bash
# Verify KVM support
ls -l /dev/kvm

# On Ubuntu/Debian, install:
sudo apt-get install -y qemu-kvm
```

Without KVM, the ISO will run under software emulation (slower but functional).

### Firmware Search Paths

The script `scripts/run-qemu.sh` looks for UEFI firmware in these locations:

- `/usr/share/ovmf/OVMF.fd`
- `/usr/share/OVMF/OVMF_CODE.fd`
- `/usr/share/edk2-ovmf/x64/OVMF_CODE.fd`

## ISO Build

The `make iso` and `agnostic iso build` commands create a bootable ISO image from a RootFS directory. These require:

### Packages (Debian/Ubuntu)

```bash
sudo apt-get install -y xorriso isolinux grub-pc-bin grub-efi-amd64-bin
```

- **xorriso** (libisoburn) — ISO creation tool (provides `xorrisofs`); alternatives are `mkisofs` or `genisoimage`
- **isolinux** (syslinux) — BIOS bootloader files (`isolinux.bin`, `boot.cat`); must be present in the RootFS at `isolinux/isolinux.bin`
- **grub** (grub-mkrescue, grub-pc-bin, grub-efi-amd64-bin) — UEFI boot support; if `boot/grub/efi.img` exists in the RootFS, the ISO will include UEFI boot capability

### Detection Order

The builder probes for ISO creation tools in this order:
1. `xorrisofs` (from libisoburn)
2. `mkisofs` (from genisoimage/cdrtools)
3. `genisoimage` (from genisoimage)

## CI Environment

The GitHub Actions workflow defined in `.github/workflows/build.yml` installs
`qemu-system-x86` and `ovmf` automatically on `ubuntu-latest` runners.
