# AgnosticOS

**A developer-focused Linux distribution built from scratch with a hybrid package manager written in Go.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)

## Overview

AgnosticOS is a Linux From Scratch (LFS) based distribution that unifies multiple package ecosystems (Pacman, Nix, Flatpak) through a meta-wrapper package manager written in Go. It provides container-based isolation for package operations and focuses on developer productivity.

## Architecture

```
┌─────────────────────────────────────────┐
│         agnostic CLI (Cobra)            │
├─────────────────────────────────────────┤
│      AgnosticManager (Orchestrator)     │
├──────────┬──────────┬───────────────────┤
│  Pacman  │   Nix    │    Flatpak        │
│  Backend │ Backend  │    Backend        │
├──────────┴──────────┴───────────────────┤
│   Linux Namespaces (CLONE_NEWNS)        │
├─────────────────────────────────────────┤
│      AgnosticOS Base (LFS Core)         │
└─────────────────────────────────────────┘
```

## Quick Start

```bash
git clone https://github.com/ElioNeto/agnostikos.git
cd agnostikos
make deps
make build
./build/agnostic --help
```

## Project Structure

```
agnostikos/
├── cmd/agnostic/          # CLI entry point (Cobra)
├── internal/
│   ├── manager/           # PackageService interface + backends
│   └── bootstrap/         # RootFS, Kernel, ISO build system
├── recipes/               # YAML image definitions
├── scripts/               # Host automation scripts
└── Makefile
```

## Makefile Targets

```bash
make build       # Compile CLI binary
make test        # Run unit tests
make test-iso    # Test ISO in QEMU
make lint        # Run golangci-lint
make fmt         # Format code
make install     # Install to /usr/local/bin
make clean       # Remove build artifacts
```

## Roadmap

- [x] CLI Bootstrap with Cobra
- [x] PackageService interface
- [ ] RootFS generator (FHS)
- [ ] Pacman backend
- [ ] Kernel compilation
- [ ] ISO generation (UEFI)
- [ ] Nix backend
- [ ] Flatpak backend
- [ ] Namespace isolation

## License

MIT — see [LICENSE](LICENSE)

**Author:** [Elio Neto](https://github.com/ElioNeto) — Santa Catarina, Brasil
