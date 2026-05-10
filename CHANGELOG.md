# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.2.0] - 2026-05-10

### Added
- **CLI additions** — new commands for complete package lifecycle management:
  - `agnostic list` — list installed packages across all backends
  - `agnostic search <query>` — search packages in configured backends
  - `agnostic remove <package>` — remove an installed package
  - `agnostic update [package]` — update packages with `--all` and `--dry-run` flags
  - `agnostic tui` — interactive terminal UI for package search and management
  - `agnostic serve` — Web UI server for browser-based package management
  - `agnostic dotfiles` — manage dotfiles via `backup`, `restore`, `sync` subcommands
  - `agnostic validate <config>` — validate `agnostic.yaml` configuration files
- **Bootstrap/build unification** — `agnostic build` replaces `agnostic bootstrap` as the primary entry point for ISO generation; `bootstrap` kept for backward compatibility (#48)
- **14-step bootstrap pipeline** — `BootstrapAll` executa pipeline completo com `emitProgress` em cada step, suportando flags `Skip*`, `Force` e `chan<- string` para progresso em tempo real
- **Parallel toolchain download** — `DownloadToolchain` usa `errgroup` + semáforo com `maxConcurrent` (padrão 3, cap 10); respeita `--jobs`; exibe progresso por arquivo (#57)
- **SHA256/SHA512 integrity verification** — `downloadFile` verifica checksum após cada download e remove arquivo corrompido em caso de mismatch; SHA512 tem prioridade sobre SHA256 (#55, #61)
  - `binutils-2.42` — verificado via SHA512 (padrão do upstream)
  - `gcc-14.3.0` — verificado via SHA256
  - `glibc-2.39` — verificado via SHA256
- **HTTPS enforcement** — `enforceHTTPS = true` por padrão; qualquer URL sem prefixo `https://` é rejeitada com erro explícito (#55)
- **TUI BuildConfigView** — formulário configurável antes do build com campos `TargetDir`, `KernelVersion`, `Arch`, `OutputISO`, `BusyboxVersion`, `Jobs`, toggles `SkipToolchain`/`SkipKernel`; navegação Tab/Shift+Tab/Enter/Esc (#60)
- **TUI build progress** — barra de progresso `[███████░░░░] 7/14` com step atual em tempo real; `buildDone` exibe caminho da ISO ou erro com contexto (#60)
- **ARM64 CI with QEMU** — cross-architecture CI pipeline usando QEMU user-mode emulation para build e teste de artefatos ARM64 em runners amd64
- **Isolation tests** — boot headless automatizado via QEMU em CI com timeout configurável (300s para emulação TCG)
- **Man pages** — geradas automaticamente via `cobra/doc`; geradas com `make docs` e incluídas nos archives de release em `docs/man/`

### Changed
- **Project structure** — novos pacotes internos: `server/`, `tui/`, `dotfiles/` para Web UI, TUI e gerenciamento de dotfiles
- **Makefile** — adicionados targets `docs`, `test-minimal-iso-headless`, `test-boot-integration`, `test-boot-integration-uefi`, `package`
- **`ToolchainPackage`** — novo campo `SHA512` adicionado ao struct; `downloadFile` atualizado com assinatura `(ctx, dest, url, sha256, sha512)`
- **gcc toolchain** — versão atualizada de `14.1.0` para `14.3.0`

### Fixed
- **manager.Build** — corrigidas falhas de build na camada de abstração do package manager (#39)
- **root.go test import** — corrigido bug de import de testes no setup do comando root (#41)

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

[v0.2.0]: https://github.com/ElioNeto/agnostikos/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/ElioNeto/agnostikos/releases/tag/v0.1.0
