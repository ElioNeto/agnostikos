package bootstrap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BaseDir é o diretório raiz de todo o ambiente AgnosticOS.
// Tudo relacionado a build, rootfs, sources e ISO fica aqui dentro.
const BaseDir = "/mnt/data/agnostikOS"

// DefaultRoot é o diretório padrão do RootFS dentro do BaseDir.
const DefaultRoot = BaseDir + "/rootfs"

// ToolchainPackage descreve um pacote da toolchain
type ToolchainPackage struct {
	Name string
	URL  string
}

// DefaultToolchain lista os pacotes base
var DefaultToolchain = []ToolchainPackage{
	{"binutils-2.42", "https://sourceware.org/pub/binutils/releases/binutils-2.42.tar.xz"},
	{"gcc-14.1.0", "https://ftp.gnu.org/gnu/gcc/gcc-14.1.0/gcc-14.1.0.tar.xz"},
	{"glibc-2.39", "https://ftp.gnu.org/gnu/glibc/glibc-2.39.tar.xz"},
}

// FHSDirectories é a árvore de diretórios do Filesystem Hierarchy Standard
var FHSDirectories = []string{
	"boot",
	"dev",
	"etc",
	"home",
	"media",
	"mnt",
	"opt",
	"proc",
	"root",
	"run",
	"srv",
	"sys",
	"tmp",
	"usr/bin",
	"usr/lib",
	"usr/sbin",
	"usr/include",
	"usr/share",
	"usr/local",
	"usr/src",
	"var/cache",
	"var/lib",
	"var/log",
	"var/run",
	"var/tmp",
}

// resolveTarget retorna o target resolvido: arg > env AGNOSTICOS_ROOT > DefaultRoot
func resolveTarget(target string) string {
	if target != "" {
		return target
	}
	if v := os.Getenv("AGNOSTICOS_ROOT"); v != "" {
		return v
	}
	return DefaultRoot
}

// sourcesDir retorna /mnt/data/agnostikOS/sources (ou base derivado de AGNOSTICOS_ROOT)
func sourcesDir(rootfsDir string) string {
	base := BaseDir
	if v := os.Getenv("AGNOSTICOS_ROOT"); v != "" {
		base = filepath.Dir(v)
	}
	_ = rootfsDir
	return filepath.Join(base, "sources")
}

// tmpDir retorna /mnt/data/agnostikOS/tmp
func tmpDir() string {
	base := BaseDir
	if v := os.Getenv("AGNOSTICOS_ROOT"); v != "" {
		base = filepath.Dir(v)
	}
	return filepath.Join(base, "tmp")
}

// artifactExists retorna true se o caminho dado já existir no disco.
func artifactExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CreateRootFS monta a árvore FHS no diretório alvo e inicializa o VirtualFS
func CreateRootFS(target string) error {
	target = resolveTarget(target)
	fmt.Printf("[rootfs] Creating RootFS at: %s\n", target)

	for _, dir := range FHSDirectories {
		path := filepath.Join(target, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
	}

	symlinks := map[string]string{
		filepath.Join(target, "bin"):     "usr/bin",
		filepath.Join(target, "lib"):     "usr/lib",
		filepath.Join(target, "lib64"):   "usr/lib",
		filepath.Join(target, "sbin"):    "usr/sbin",
		filepath.Join(target, "var/run"): "../run",
	}
	for link, dest := range symlinks {
		os.Remove(link)
		if err := os.Symlink(dest, link); err != nil {
			fmt.Printf("[rootfs] warn: symlink %s -> %s: %v\n", link, dest, err)
		}
	}

	fmt.Println("[rootfs] FHS structure created")
	return mountVirtualFS(target)
}

// DownloadToolchain baixa os pacotes da toolchain para /mnt/data/agnostikOS/sources
func DownloadToolchain(rootfsDir string) error {
	rootfsDir = resolveTarget(rootfsDir)
	src := sourcesDir(rootfsDir)
	if err := os.MkdirAll(src, 0755); err != nil {
		return fmt.Errorf("mkdir sources: %w", err)
	}
	for _, pkg := range DefaultToolchain {
		dest := filepath.Join(src, filepath.Base(pkg.URL))
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("[toolchain] already exists: %s\n", pkg.Name)
			continue
		}
		fmt.Printf("[toolchain] downloading %s...\n", pkg.Name)
		if err := downloadFile(dest, pkg.URL); err != nil {
			return fmt.Errorf("download %s: %w", pkg.Name, err)
		}
		fmt.Printf("[toolchain] downloaded %s\n", pkg.Name)
	}
	return nil
}

// downloadFile faz o download de uma URL para um arquivo local
func downloadFile(dest, url string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// mountVirtualFS monta proc/sys/dev dentro do chroot
func mountVirtualFS(target string) error {
	type mountSpec struct {
		fstype, source, target, opts string
	}
	mounts := []mountSpec{
		{"proc", "proc", filepath.Join(target, "proc"), ""},
		{"sysfs", "sysfs", filepath.Join(target, "sys"), ""},
		{"devtmpfs", "devtmpfs", filepath.Join(target, "dev"), "mode=0755"},
		{"devpts", "devpts", filepath.Join(target, "dev/pts"), "gid=5,mode=0620"},
		{"tmpfs", "tmpfs", filepath.Join(target, "run"), ""},
	}
	for _, m := range mounts {
		args := []string{"-t", m.fstype}
		if m.opts != "" {
			args = append(args, "-o", m.opts)
		}
		args = append(args, m.source, m.target)
		if out, err := exec.Command("mount", args...).CombinedOutput(); err != nil {
			fmt.Printf("[rootfs] warn: mount %s: %s\n", m.fstype, string(out))
		} else {
			fmt.Printf("[rootfs] mounted %s -> %s\n", m.fstype, m.target)
		}
	}
	return nil
}

// UnmountVirtualFS desmonta os filesystems virtuais do chroot
func UnmountVirtualFS(target string) error {
	for _, p := range []string{"dev/pts", "dev", "run", "proc", "sys"} {
		_ = exec.Command("umount", filepath.Join(target, p)).Run()
	}
	return nil
}

// BootstrapConfig contém todos os parâmetros para a construção completa do RootFS
type BootstrapConfig struct {
	TargetDir      string // diretório raiz do RootFS (ex: /mnt/data/agnostikOS/rootfs)
	Device         string // disco base para grub-install BIOS (ex: /dev/sda)
	EFIPartition   string // partição ESP para grub-install UEFI (ex: /dev/nvme0n1p1)
	KernelVersion  string // versão do kernel Linux (ex: "6.6")
	BusyboxVersion string // versão do Busybox (ex: "1.36.1")
	Arch           string // target arch: "amd64" ou "arm64" (vazio = auto-detect de runtime.GOARCH)
	UEFI           bool   // gerar estrutura UEFI
	SkipToolchain  bool   // pular compilação da toolchain (binutils, gcc, glibc)
	SkipKernel     bool   // pular compilação do kernel
	SkipBusybox    bool   // pular compilação do busybox
	SkipInitramfs  bool   // pular geração do initramfs
	SkipGRUB       bool   // pular instalação do GRUB
	Force          bool   // ignorar cache e recompilar tudo
}

// kernelImageName retorna o nome do arquivo da imagem do kernel de acordo com a arquitetura.
func kernelImageName(arch, version string) string {
	if arch == "arm64" {
		return "Image-" + version
	}
	return "vmlinuz-" + version
}

// BootstrapAll executa o pipeline completo de construção do RootFS
func BootstrapAll(ctx context.Context, cfg BootstrapConfig) error {
	if cfg.TargetDir == "" {
		cfg.TargetDir = resolveTarget("")
	}

	fmt.Printf("[bootstrap] Starting full bootstrap at %s\n", cfg.TargetDir)
	arch := cfg.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	fmt.Printf("[bootstrap] Config: kernel=%s busybox=%s arch=%s uefi=%v force=%v\n",
		cfg.KernelVersion, cfg.BusyboxVersion, arch, cfg.UEFI, cfg.Force)

	// Step 1: RootFS — idempotente, MkdirAll é no-op se já existe
	fmt.Println("\n=== Step 1/9: Create RootFS ===")
	if err := CreateRootFS(cfg.TargetDir); err != nil {
		return fmt.Errorf("create rootfs: %w", err)
	}

	// Step 2: Toolchain — download dos tarballs
	fmt.Println("\n=== Step 2/9: Download Toolchain ===")
	if err := DownloadToolchain(cfg.TargetDir); err != nil {
		return fmt.Errorf("download toolchain: %w", err)
	}

	tcCfg := ToolchainConfig{
		TargetDir: cfg.TargetDir,
	}

	// Step 3: Build binutils
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 3/9: Build binutils ===")
		if err := BuildBinutils(ctx, tcCfg); err != nil {
			return fmt.Errorf("build binutils: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 3/9: Build binutils (skipped) ===")
	}

	// Step 4: Build GCC (pass 1, C only)
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 4/9: Build GCC (pass 1) ===")
		if err := BuildGCC(ctx, tcCfg); err != nil {
			return fmt.Errorf("build gcc: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 4/9: Build GCC (skipped) ===")
	}

	// Step 5: Build glibc
	if !cfg.SkipToolchain {
		fmt.Println("\n=== Step 5/9: Build glibc ===")
		if err := BuildGLibc(ctx, tcCfg); err != nil {
			return fmt.Errorf("build glibc: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 5/9: Build glibc (skipped) ===")
	}

	// Step 6: Kernel
	kernelImage := kernelImageName(arch, cfg.KernelVersion)
	kernelArtifact := filepath.Join(cfg.TargetDir, "boot", kernelImage)
	if !cfg.SkipKernel {
		if !cfg.Force && artifactExists(kernelArtifact) {
			fmt.Printf("\n=== Step 6/9: Build Kernel (cached: %s) ===\n", kernelArtifact)
		} else {
			fmt.Println("\n=== Step 6/9: Build Kernel ===")
			kernelCfg := KernelConfig{
				Version:    cfg.KernelVersion,
				SourcesDir: sourcesDir(cfg.TargetDir),
				OutputDir:  filepath.Join(cfg.TargetDir, "boot"),
				Defconfig:  "", // auto-detect from arch
				Arch:       arch,
			}
			if err := BuildKernel(kernelCfg); err != nil {
				return fmt.Errorf("build kernel: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 6/9: Build Kernel (skipped by flag) ===")
	}

	// Step 4: Busybox
	busyboxArtifact := filepath.Join(cfg.TargetDir, "busybox-install", "bin", "busybox")
	if !cfg.SkipBusybox {
		if !cfg.Force && artifactExists(busyboxArtifact) {
			fmt.Printf("\n=== Step 7/9: Build Busybox (cached: %s) ===\n", busyboxArtifact)
		} else {
			fmt.Println("\n=== Step 7/9: Build Busybox ===")
			busyboxCfg := BusyboxConfig{
				Version:   cfg.BusyboxVersion,
				TargetDir: cfg.TargetDir,
			}
			if err := BuildBusybox(ctx, busyboxCfg); err != nil {
				return fmt.Errorf("build busybox: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 7/9: Build Busybox (skipped by flag) ===")
	}

	// Step 5: Initramfs
	initramfsArtifact := filepath.Join(cfg.TargetDir, "boot", "initramfs.img")
	if !cfg.SkipInitramfs {
		if !cfg.Force && artifactExists(initramfsArtifact) {
			fmt.Printf("\n=== Step 8/9: Build Initramfs (cached: %s) ===\n", initramfsArtifact)
		} else {
			fmt.Println("\n=== Step 8/9: Build Initramfs ===")
			outputDir := filepath.Join(cfg.TargetDir, "boot")
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("mkdir boot: %w", err)
			}
			if err := BuildInitramfs(ctx, cfg.TargetDir, initramfsArtifact); err != nil {
				return fmt.Errorf("build initramfs: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 8/9: Build Initramfs (skipped by flag) ===")
	}

	// Step 6: GRUB
	grubArtifact := filepath.Join(cfg.TargetDir, "boot", "grub", "grub.cfg")
	if !cfg.SkipGRUB {
		if !cfg.Force && artifactExists(grubArtifact) {
			fmt.Printf("\n=== Step 9/9: Install GRUB (cached: %s) ===\n", grubArtifact)
		} else {
			fmt.Println("\n=== Step 9/9: Install GRUB ===")
			if err := InstallGRUB(ctx, GRUBConfig{
				RootfsDir:    cfg.TargetDir,
				Device:       cfg.Device,
				UEFI:         cfg.UEFI,
				EFIPartition: cfg.EFIPartition,
				Strict:       true,
			}); err != nil {
				return fmt.Errorf("install grub: %w", err)
			}
		}
	} else {
		fmt.Println("\n=== Step 9/9: Install GRUB (skipped by flag) ===")
	}

	fmt.Printf("\n[bootstrap] ✅ Bootstrap complete at %s\n", cfg.TargetDir)
	return nil
}
