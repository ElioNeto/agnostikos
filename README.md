# AgnosticOS

**A meta-wrapper package manager — unify Pacman, Nix, and Flatpak under a single CLI.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![CI](https://github.com/ElioNeto/agnostikos/actions/workflows/build.yml/badge.svg)](https://github.com/ElioNeto/agnostikos/actions/workflows/build.yml)
[![Release](https://img.shields.io/github/v/release/ElioNeto/agnostikos?include_prereleases&sort=semver)](https://github.com/ElioNeto/agnostikos/releases)

---

## 🚀 Installation

### Via install script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/ElioNeto/agnostikos/main/scripts/install.sh | sh
```

The script automatically detects your architecture (amd64 / arm64), fetches the latest release from GitHub, verifies the SHA256 checksum, and installs the `agnostic` binary to `/usr/local/bin` (or `~/.local/bin` as fallback).

### Manual download

Download the pre-built binary for your platform from the [Releases page](https://github.com/ElioNeto/agnostikos/releases), extract it, and place it in your `PATH`:

```bash
tar -xzf agnostikos_*.tar.gz
sudo install agnostic /usr/local/bin/
```

---

## 📖 Overview

AgnosticOS is a **meta-wrapper package manager** that abstracts multiple package ecosystems behind a single, unified command-line interface. Instead of learning `pacman -S`, `nix profile install`, and `flatpak install` separately, you use `agnostic install` — the tool dispatches to the right backend automatically.

Built from scratch in Go, AgnosticOS is designed for developers who work across distributions and want a consistent, scriptable package management experience.

**Key features:**
- 🔀 **Multi-backend dispatch** — Pacman, Nix, and Flatpak, all from one CLI
- 📋 **Config-driven installs** — declare your packages in `agnostic.yaml`
- 🔒 **Namespace isolation** — optional Linux mount namespace sandboxing
- 🏗️ **Build pipeline** — full ISO generation: RootFS → toolchain → kernel → busybox → initramfs → GRUB → ISO (BIOS or UEFI)
- 💿 **ISO builder** — standalone ISO generation from an existing RootFS (`agnostic iso`)

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

## 🏗️ Build pipeline

The `build` command runs the complete ISO generation pipeline — from creating the RootFS
to producing a bootable ISO image.

```bash
# Default build (auto-detects architecture, uses kernel 6.6, busybox 1.36.1)
sudo ./build/agnostic build --output /tmp/agnostikos.iso

# Build from a recipe file
./build/agnostic build recipes/base.yaml

# Quick build (skip lengthy toolchain compilation)
./build/agnostic build --skip-toolchain --output /tmp/agnostikos.iso

# Full build with UEFI
sudo ./build/agnostic build \
  --uefi \
  --output /tmp/agnostikos.iso
```

> **Note**: The `bootstrap` subcommand is still available internally for advanced use
> (`agnostic bootstrap --help`), but `build` is the recommended entry point.

### Key flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output` | `agnostikos-latest.iso` | Output ISO path |
| `-t, --target` | `$AGNOSTICOS_ROOT` | RootFS target directory |
| `--device` | — | Disk for BIOS `grub-install` |
| `--efi-partition` | — | ESP partition for UEFI `grub-install` |
| `--uefi` | `false` | Enable UEFI boot support |
| `--kernel-version` | `6.6` | Linux kernel version |
| `--busybox-version` | `1.36.1` | Busybox version |
| `--arch` | auto-detect | Target architecture (`amd64`, `arm64`) |
| `--recipe` | — | Load settings from a YAML recipe file |
| `--skip-toolchain` | `false` | Skip toolchain compilation |
| `--skip-kernel` | `false` | Skip kernel compilation |
| `--skip-busybox` | `false` | Skip busybox compilation |
| `--skip-initramfs` | `false` | Skip initramfs generation |
| `--skip-grub` | `false` | Skip GRUB installation |
| `--force` | `false` | Rebuild all steps from scratch |

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
make bootstrap          # (internal) RootFS bootstrap pipeline (requires root)
make dev                # Run in development mode
```

> **Note:** `make build` runs the full pipeline and does NOT require root unless you use
> `--device`/`--efi-partition` for GRUB installation into real hardware. `make bootstrap`
> is kept for backward compatibility but `make build [ARGS="..."]` is the recommended target.

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

## 🚀 Post-Boot Flow

After booting a custom AgnosticOS ISO, the system starts with **busybox init** as the PID 1 process.

### Init system

The kernel launches `/init` which is a symlink to busybox's `/sbin/init`. The init process reads `/etc/inittab` to determine what to run:

```
::sysinit:/etc/init.d/rcS       →  system initialization
::ctrlaltdel:/sbin/reboot       →  Ctrl+Alt+Del handling
::shutdown:/sbin/swapoff -a     →  swap teardown
::shutdown:/bin/umount -a -r    →  filesystem unmount
```

The boot script `/etc/init.d/rcS` mounts virtual filesystems, sets up device nodes via `mdev`, and configures the hostname.

### Login

- **Auto-login mode (default for live sessions):** If `AutoLoginUser` is set in the bootstrap config, the configured user is automatically logged in on `tty1` via `/bin/login -f <user>`.
- **Manual login mode:** When no auto-login user is configured, `init` presents a login prompt on the console (`::askfirst:-/bin/sh`). The default credentials depend on the contents of `/etc/passwd` and `/etc/shadow` in the rootfs.

### Available commands

Once logged in, the following `agnostic` commands are available:

| Command                       | Description                                      |
|-------------------------------|--------------------------------------------------|
| `agnostic tui`                | Launch the interactive terminal UI (TUI)         |
| `agnostic install <package>`  | Install a package via the default backend        |
| `agnostic install --config <file>` | Install packages declared in a YAML config  |
| `agnostic list`               | List installed packages across all backends      |
| `agnostic search <query>`     | Search for packages in configured backends       |
| `agnostic remove <package>`   | Remove an installed package                      |
| `agnostic backend list`       | Show available backends (pacman, nix, flatpak)   |

### Package installation

```bash
# Install a single package using the default backend
agnostic install firefox

# Install using a specific backend
agnostic install firefox --backend flatpak

# Install from a declarative config file
agnostic install --config /etc/agnostic.yaml
```

### Building a custom ISO

```bash
# Full pipeline: RootFS → toolchain → kernel → busybox → initramfs → GRUB → ISO
sudo make build ARGS="--output /tmp/custom.iso"

# Quick build (skip toolchain, use cached artifacts)
sudo make build ARGS="--skip-toolchain --output /tmp/custom.iso"

# Build with a recipe file
./build/agnostic build recipes/base.yaml
```

---

## ⬇️ Download

Pre-built binaries for Linux (amd64 and arm64) are available on the
[Releases page](https://github.com/ElioNeto/agnostikos/releases).

Each release includes:
- `agnostikos_<version>_linux_amd64.tar.gz`
- `agnostikos_<version>_linux_arm64.tar.gz`
- `checksums.txt` (SHA256)

### Man pages

Man pages are included in the release archive under `docs/man/`. To install them:

```bash
sudo install -d /usr/local/share/man/man1
sudo install -m 644 docs/man/* /usr/local/share/man/man1/
sudo mandb  # update the man database
```

You can also generate the latest man pages from source:

```bash
make docs
sudo install -m 644 docs/man/* /usr/local/share/man/man1/
sudo mandb
```

---

## 🤝 Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, development setup, and the PR process.

---

## 📄 License

MIT — see [LICENSE](LICENSE)

---

**Author:** [Elio Neto](https://github.com/ElioNeto) — Santa Catarina, Brasil
