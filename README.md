# AgnosticOS

**A meta-wrapper package manager — unify Pacman, Nix, and Flatpak under a single CLI.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![CI](https://github.com/ElioNeto/agnostikos/actions/workflows/build.yml/badge.svg)](https://github.com/ElioNeto/agnostikos/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/ElioNeto/agnostikos?include_prereleases&sort=semver)](https://github.com/ElioNeto/agnostikos/releases)

---

## 📖 Overview

AgnosticOS is a **meta-wrapper package manager** that abstracts multiple package ecosystems behind a single, unified command-line interface. Instead of learning `pacman -S`, `nix profile install`, and `flatpak install` separately, you use `agnostic install` — the tool dispatches to the right backend automatically.

Built from scratch in Go, AgnosticOS is designed for developers who work across distributions and want a consistent, scriptable package management experience.

**Key features:**
- 🔀 **Multi-backend dispatch** — Pacman, Nix, and Flatpak, all from one CLI
- 📋 **Config-driven installs** — declare your packages in `agnostic.yaml`
- 🔒 **Namespace isolation** — optional Linux mount namespace sandboxing
- 🏗️ **Bootstrap pipeline** — RootFS (FHS) → toolchain download → kernel → busybox → initramfs → GRUB (BIOS or UEFI)
- 💿 **ISO builder** — produce a bootable ISO and test it in QEMU

---

## 🧩 Supported backends

| Backend    | Scope          | Use case                       | Status |
|------------|----------------|--------------------------------|--------|
| **Pacman** | Arch Linux     | Native Arch packages           | ✅     |
| **Nix**    | NixOS / multi  | Reproducible, declarative pkgs | ✅     |
| **Flatpak**| Universal      | Sandboxed desktop apps         | ✅     |

---

## 🚀 Quick start

```bash
# Clone
git clone https://github.com/ElioNeto/agnostikos.git
cd agnostikos

# Install dependencies & build
make deps
make build

# See available commands
./build/agnostic --help
```

**Install a single package:**
```bash
./build/agnostic install firefox --backend flatpak
```

**Install from config file:**
```bash
./build/agnostic install --config agnostic.yaml
```

---

## 🏗️ Bootstrap pipeline

The `bootstrap` command builds a complete bootable RootFS from scratch.

### UEFI (recommended)

```bash
sudo ./build/agnostic bootstrap \
  --target /mnt/lfs \
  --efi-partition /dev/nvme0n1p1 \
  --uefi
```

> `--efi-partition` mounts the EFI System Partition (FAT32) at `<target>/boot/efi` automatically before running `grub-install`, then unmounts it.

### BIOS (MBR disks only)

> ⚠️ **GPT disks require a BIOS Boot Partition** (1 MB, type `EF02`) before using BIOS mode.
> If your disk is GPT without one, use `--uefi` instead.

```bash
sudo ./build/agnostic bootstrap \
  --target /mnt/lfs \
  --device /dev/sda
```

### Skip individual steps

```bash
sudo ./build/agnostic bootstrap \
  --target /mnt/lfs \
  --efi-partition /dev/nvme0n1p1 \
  --uefi \
  --skip-kernel \
  --skip-busybox \
  --skip-initramfs
```

### All `bootstrap` flags

| Flag | Default | Description |
|------|---------|-------------|
| `-t, --target` | `$LFS` or `/mnt/lfs` | Target RootFS directory |
| `--device` | — | Disk for BIOS `grub-install` (e.g. `/dev/sda`) |
| `--efi-partition` | — | ESP partition to auto-mount for UEFI (e.g. `/dev/nvme0n1p1`) |
| `--uefi` | `false` | Enable UEFI boot support |
| `--kernel-version` | `6.6` | Linux kernel version |
| `--busybox-version` | `1.36.1` | Busybox version |
| `--skip-kernel` | `false` | Skip kernel compilation |
| `--skip-busybox` | `false` | Skip busybox compilation |
| `--skip-initramfs` | `false` | Skip initramfs generation |
| `--skip-grub` | `false` | Skip GRUB installation |

---

## 📁 Project structure

```
agnostikos/
├── cmd/agnostic/          # CLI entry point (Cobra)
├── internal/
│   ├── config/            # YAML config parsing
│   ├── manager/           # PackageService interface + backends
│   ├── bootstrap/         # RootFS, kernel, busybox, initramfs, GRUB
│   ├── iso/               # ISO builder (xorriso/mkisofs)
│   └── isolation/         # Linux namespace isolation
├── recipes/               # YAML image definitions (base.yaml)
├── scripts/               # QEMU runner, CI helpers
├── docs/                  # Architecture & requirements docs
├── agnostic.yaml.example  # Example configuration
├── Makefile               # Build, test, lint, ISO targets
└── main.go                # Binary entry point
```

---

## ⚙️ Prerequisites

| Tool                   | Version / Notes                                          |
|------------------------|----------------------------------------------------------|
| **Go**                 | 1.22+                                                    |
| **GNU Make**           | Any recent version                                       |
| **grub-install**       | GRUB 2 (`apt install grub-efi-amd64` or `grub-pc`)       |
| **xorriso**            | ISO creation (`apt install xorriso`)                     |
| **qemu-system-x86**    | QEMU for ISO testing (`apt install qemu-system-x86`)     |
| **ovmf**               | UEFI firmware for QEMU (`apt install ovmf`)              |
| **git**                | Version control                                          |

See [docs/requirements.md](docs/requirements.md) for detailed setup instructions.

---

## 🔧 Makefile targets

```bash
make build              # Compile CLI binary
make test               # Run unit tests with race detector
make test-iso           # Test ISO in QEMU (graphical)
make test-iso-headless  # Test ISO in QEMU (headless, for CI)
make lint               # Run golangci-lint
make fmt                # Format Go code
make install            # Install to /usr/local/bin
make clean              # Remove build artifacts
make iso                # Build ISO from RootFS
make bootstrap          # Bootstrap RootFS into $(LFS) (requires root)
make dev                # Run in development mode
```

> **Note:** `make bootstrap` requires **root privileges** because it mounts virtual filesystems (`proc`, `sys`, `dev`) and optionally the EFI System Partition into the target directory.

---

## 📄 Example configuration (`agnostic.yaml`)

```yaml
version: "1.0"
locale: pt_BR.UTF-8
timezone: America/Sao_Paulo

packages:
  base:
    - vim
    - git
    - curl
  extra:
    - docker
    - neovim

backends:
  default: pacman
  fallback: nix

user:
  name: dev
  shell: /bin/zsh
```

Then run:
```bash
agnostic install --config agnostic.yaml
```

---

## 🗺️ Roadmap

- [x] CLI bootstrap with Cobra
- [x] PackageService interface
- [x] Pacman backend
- [x] Nix backend
- [x] Flatpak backend
- [x] Namespace isolation (CLONE_NEWNS)
- [x] ISO builder (BIOS + UEFI)
- [x] CI pipeline (build, test, lint)
- [x] RootFS generator (FHS + usrmerge)
- [x] Toolchain download (binutils, gcc, glibc)
- [x] GRUB installation (BIOS + UEFI, auto ESP mount)
- [ ] Kernel compilation
- [ ] Busybox compilation
- [ ] Initramfs generation
- [ ] Full LFS bootstrap recipe
- [ ] QEMU smoke test in CI
- [ ] `agnostic.yaml` schema validation
- [ ] Multi-architecture support (ARM64)

---

## ⬇️ Download

Pre-built binaries for Linux (amd64 and arm64) are available on the
[Releases page](https://github.com/ElioNeto/agnostikos/releases).

Each release includes:
- `agnostikos_<version>_linux_amd64.tar.gz`
- `agnostikos_<version>_linux_arm64.tar.gz`
- `checksums.txt` (SHA256)

---

## 🤝 Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, development setup, and the PR process.

---

## 📄 License

MIT — see [LICENSE](LICENSE)

---

**Author:** [Elio Neto](https://github.com/ElioNeto) — Santa Catarina, Brasil
