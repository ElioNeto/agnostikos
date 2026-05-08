# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2026-05-08

### 🎉 First public release

#### Added
- **Bootstrap pipeline** — multi-step build system for generating a minimal Linux ISO from scratch:
  - Toolchain build (binutils, GCC cross-compiler, glibc)
  - BusyBox compilation and applet symlink setup
  - Kernel compilation with minimal config
  - RootFS assembly with essential directories and init scripts
  - Initramfs creation (cpio + gzip) with full BusyBox applet set and interactive rescue shell
  - GRUB bootloader integration
  - ISO image generation via `xorriso`
- **`agnostic` CLI** (Cobra-based):
  - `agnostic bootstrap` — full ISO build pipeline with `--skip-toolchain` flag
  - `agnostic install` — package installation via configurable backends (pacman, nix, flatpak)
  - `agnostic install --config agnostic.yaml` — declarative package installation from YAML
  - `agnostic tui` — interactive TUI for package search and management
- **Multi-backend package manager** — unified interface for pacman, nix-env and flatpak
- **`agnostic.yaml` config format** — declarative package lists with backend selection, locale, timezone, user and dotfiles settings
- **QEMU smoke test** — automated boot validation via `scripts/run-qemu.sh`
- **CI/CD** — GitHub Actions workflows for lint, test, build and release via GoReleaser
- **GoReleaser** — automated release pipeline producing binaries for `linux/amd64` and `linux/arm64`

#### Artifacts

| File | Architecture |
|---|---|
| `agnostikos_0.1.0_linux_amd64.tar.gz` | amd64 |
| `agnostikos_0.1.0_linux_arm64.tar.gz` | arm64 |
| `agnostikos_0.1.0_amd64.deb` | amd64 (Debian/Ubuntu) |
| `agnostikos_0.1.0_arm64.deb` | arm64 (Debian/Ubuntu) |
| `agnostikos_0.1.0_amd64.rpm` | amd64 (Fedora/RHEL) |
| `agnostikos_0.1.0_arm64.rpm` | arm64 (Fedora/RHEL) |
| `checksums.txt` | SHA256 checksums |

#### Bug Fixes
- Fixed BusyBox applet symlink creation failing with `file exists` error when symlinks already existed in the initramfs `bin/` directory — now uses `os.IsExist` guard to skip existing symlinks idempotently

#### Contributors
- @ElioNeto — initial project setup and full implementation

**Full Changelog**: https://github.com/ElioNeto/agnostikos/commits/v0.1.0

[v0.1.0]: https://github.com/ElioNeto/agnostikos/releases/tag/v0.1.0
