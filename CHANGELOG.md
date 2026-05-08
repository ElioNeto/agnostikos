# Changelog

## v0.1.0 (2026-05-08)

### Features
- Meta-package manager with Pacman, Nix, and Flatpak backends
- Package installation via `agnostic install <pkg>`
- YAML configuration system (`agnostic.yaml`)
- Custom Linux distribution bootstrapper (`agnostic bootstrap`)
- ISO image builder with BIOS/UEFI support (`agnostic iso build`)
- QEMU integration for ISO testing (`make test-iso`)
- CI/CD pipeline with GoReleaser releases
- Documentation: README, CONTRIBUTING, architecture

### Backends
- **Pacman**: Arch Linux package manager integration (stable)
- **Nix**: NixOS/nix package manager integration (stable)
- **Flatpak**: Cross-distro flatpak integration (stable)

### Technical
- Written in Go 1.24
- Cobra CLI framework
- GoReleaser for multi-arch releases
- CI: GitHub Actions (lint, test, build, ISO test)
- Architecture: modular internal packages (config, manager, bootstrap, iso, isolation)

### Resolved Issues
- **#18** — RootFS bootável real: kernel, GRUB, init mínimo. Full bootstrap pipeline with Linux kernel compilation, Busybox, initramfs generation, and GRUB installation (BIOS + UEFI, auto ESP mount).
- **#19** — CI test-iso-headless with minimal ISO. Headless QEMU integration test for CI pipelines, using a minimal RootFS with host kernel and test initramfs.
