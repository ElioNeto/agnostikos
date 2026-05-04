# Architecture

---

## 📐 Module diagram

```
┌──────────────────────────────────────────────────────────────┐
│                     cmd/agnostic/ (CLI entry)                │
│                       Cobra command tree                     │
│  install  ·  remove  ·  update  ·  search  ·  build  ·  iso │
│  bootstrap · --config · --backend · --isolated               │
└──────────┬───────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│                  internal/config                             │
│            YAML parsing & validation                         │
│  Config { Version, Locale, Timezone, Packages, Backends,    │
│           User }                                             │
│  Load(path) → *Config, error                                │
│  Validate() → error                                         │
└──────────┬───────────────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│                  internal/manager                            │
│          PackageService interface + backends                 │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  PackageService  (interface)                         │   │
│  │    Install(pkgName) → error                          │   │
│  │    Remove(pkgName)  → error                          │   │
│  │    Update()          → error                         │   │
│  │    Search(query)     → ([]string, error)             │   │
│  └──────────┬──────────┬──────────┬─────────────────────┘   │
│             │          │          │                          │
│      ┌──────┴──────┐ ┌─┴──────┐ ┌─┴────────┐               │
│      │ Pacman      │ │ Nix    │ │ Flatpak  │               │
│      │ Backend     │ │ Backend│ │ Backend  │               │
│      └─────────────┘ └────────┘ └──────────┘               │
│                                                              │
│  AgnosticManager                                            │
│    Backends: map[string]PackageService                       │
│    RegisterBackend(name, svc)                                │
│    ListBackends() → []string                                 │
└──────────┬───────────────────────────────────────────────────┘
           │
     ┌─────┴───────────────────────────────────────────────────┐
     │                                                         │
     ▼                                                         ▼
┌──────────────────────────┐          ┌──────────────────────────┐
│   internal/bootstrap     │          │   internal/isolation     │
│                          │          │                          │
│  CreateRootFS(target)    │          │  RunIsolated(cmd, args)  │
│    → FHS directory tree  │          │    → CLONE_NEWNS         │
│    → usrmerge symlinks   │          │    → requires root       │
│    → mount VirtualFS     │          │                          │
│                          │          │  Used by:                │
│  BuildKernel(config)     │          │  agnostic install        │
│    → download sources    │          │    --isolated            │
│    → compile x86_64      │          │                          │
│    → output vmlinuz      │          └──────────────────────────┘
│                          │
│  DownloadToolchain()     │
│    → binutils, gcc, glibc│
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│                  internal/iso                                │
│                                                              │
│  ISOBuilder                                                  │
│    executor: Executor (abstrai exec.Command)                 │
│    rootfsPath, outputPath                                    │
│                                                              │
│  Build(ctx, rootfs, output) → error                         │
│    1. Detect tool: xorrisofs > mkisofs > genisoimage        │
│    2. Add BIOS boot args (isolinux)                          │
│    3. Add UEFI boot args (efi.img)                           │
│    4. Run tool → output ISO                                  │
│    5. Generate SHA256 checksum                               │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔄 End-to-end build flow

```
                     agnostic.yaml
                          │
                          ▼
              ┌──────────────────────┐
              │  internal/config     │
              │  Load + Validate     │
              └──────────┬───────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  internal/manager    │
              │  Dispatch backend    │
              │  Install packages    │
              └──────────┬───────────┘
                         │
          ──── optional ──┼──── isolate ──────────────────
                         ▼
              ┌──────────────────────────────┐
              │  internal/isolation          │
              │  RunIsolated(backend, args)  │
              │  → CLONE_NEWNS mount ns      │
              └──────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  internal/bootstrap  │
              │  CreateRootFS()      │
              │  → FHS tree          │
              │  → usrmerge symlinks │
              │  → mount VirtualFS   │
              │  → (optional) Kernel │
              └──────────┬───────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  internal/iso        │
              │  Build()             │
              │  → detect xorriso    │
              │  → add BIOS/UEFI     │
              │  → generate .iso     │
              │  → SHA256 checksum   │
              └──────────┬───────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │  QEMU (make test-iso)│
              │  run-qemu.sh *.iso   │
              │  UEFI: OVMF.fd       │
              │  Headless: -nographic│
              └──────────────────────┘
```

### Build command walkthrough

```bash
agnostic build recipes/base.yaml
```

1. **Parse recipe** — reads YAML, extracts name, kernel version, packages
2. **Create RootFS** — `bootstrap.CreateRootFS()` builds the FHS tree at `/mnt/lfs`
3. **Compile kernel** (if `kernel_version` set) — downloads sources, runs `x86_64_defconfig`, outputs `vmlinuz` to `/mnt/lfs/boot/`
4. **Generate ISO** — `bootstrap.GenerateISO()` calls `iso.Build()`, which detects `xorrisofs` and produces a bootable ISO with BIOS + optional UEFI support
5. **Checksum** — a `.sha256` file is written alongside the ISO

---

## 🎯 Design decisions

### Why Go?

- **Single binary** — no runtime dependencies; distribute a single static binary
- **Cross-compilation** — build for any platform from a single CI runner
- **Fast compilation** — iterate quickly during development
- **Strong standard library** — `os/exec`, `context`, `crypto/sha256`, `net/http` — no need for heavy frameworks
- **goroutines** — potential for parallel backend operations (not yet exploited, but the option is there)

### Why no SQLite / persistent state?

AgnosticOS is designed as a **stateless meta-wrapper**. It delegates all package state management to the underlying backends:

- `pacman` has its own database (`/var/lib/pacman/local/`)
- `nix` has its own store (`/nix/store/`)
- `flatpak` has its own installation directories

Adding another database layer (SQLite, BoltDB, etc.) would:
- Duplicate state that already exists in the backends
- Introduce sync and consistency problems
- Add complexity for no benefit

The only persisted configuration is the optional `agnostic.yaml` file, which is intentionally simple YAML.

### Why Cobra for CLI?

- De facto standard for Go CLIs
- Automatic help generation, completion, and flag parsing
- Easy subcommand tree (`install`, `remove`, `build`, `iso`, etc.)
- Well-documented with a large ecosystem

### Why namespace isolation is optional (not default)

Linux namespace isolation (`CLONE_NEWNS`) requires `CAP_SYS_ADMIN` (typically root). Making it opt-in via `--isolated` keeps the common case (unprivileged package installation) simple while offering sandboxing for those who need it.

### ISO builder architecture

Rather than shelling out to a complex build system, the ISO builder:
1. Probes for `xorrisofs` / `mkisofs` / `genisoimage` (in order of preference)
2. Constructs the correct CLI flags for BIOS + UEFI
3. Computes a SHA256 checksum of the result

This keeps the builder **lightweight** (no libisoburn C bindings) and **flexible** (works with any available mkisofs-compatible tool).

---

## 📦 PackageService interface

```go
type PackageService interface {
    Install(pkgName string) error
    Remove(pkgName string) error
    Update() error
    Search(query string) ([]string, error)
}
```

Every backend implements this interface. The `AgnosticManager` holds a registry of backends and dispatches calls to the selected one. This makes adding new backends trivial: implement the four methods and call `RegisterBackend("name", svc)`.
