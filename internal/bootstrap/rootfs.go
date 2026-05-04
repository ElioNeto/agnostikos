package bootstrap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const DefaultLFSRoot = "/mnt/lfs"

// ToolchainPackage descreve um pacote da toolchain LFS
type ToolchainPackage struct {
	Name string
	URL  string
}

// DefaultToolchain lista os pacotes base do LFS 12.x
var DefaultToolchain = []ToolchainPackage{
	{"binutils-2.42", "https://sourceware.org/pub/binutils/releases/binutils-2.42.tar.xz"},
	{"gcc-14.1.0", "https://ftp.gnu.org/gnu/gcc/gcc-14.1.0/gcc-14.1.0.tar.xz"},
	{"glibc-2.39", "https://ftp.gnu.org/gnu/glibc/glibc-2.39.tar.xz"},
}

// FHSDirectories é a árvore de diretórios do Filesystem Hierarchy Standard
var FHSDirectories = []string{
	"bin", "boot", "dev", "etc", "home", "lib", "lib64",
	"media", "mnt", "opt", "proc", "root", "run", "sbin",
	"srv", "sys", "tmp",
	"usr/bin", "usr/lib", "usr/sbin", "usr/include", "usr/share", "usr/local", "usr/src",
	"var/cache", "var/lib", "var/log", "var/run", "var/tmp",
	"tools", "sources",
}

// resolveLFSTarget retorna o target resolvido: arg > env LFS > default
func resolveLFSTarget(target string) string {
	if target != "" {
		return target
	}
	if lfs := os.Getenv("LFS"); lfs != "" {
		return lfs
	}
	return DefaultLFSRoot
}

// CreateRootFS monta a árvore FHS no diretório alvo e inicializa o VirtualFS
func CreateRootFS(target string) error {
	if target == "" {
		if lfs := os.Getenv("LFS"); lfs != "" {
			target = lfs
		} else {
			target = DefaultLFSRoot
		}
	}
	fmt.Printf("[rootfs] Creating RootFS at: %s\n", target)

	for _, dir := range FHSDirectories {
		path := filepath.Join(target, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}
	}

	// Symlinks modernos (usrmerge)
	symlinks := map[string]string{
		filepath.Join(target, "lib"):     "usr/lib",
		filepath.Join(target, "lib64"):   "usr/lib",
		filepath.Join(target, "bin"):     "usr/bin",
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

// DownloadToolchain baixa os pacotes da toolchain LFS para o diretório sources
func DownloadToolchain(target string) error {
	target = resolveLFSTarget(target)
	sourcesDir := filepath.Join(target, "sources")
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return fmt.Errorf("mkdir sources: %w", err)
	}
	for _, pkg := range DefaultToolchain {
		dest := filepath.Join(sourcesDir, filepath.Base(pkg.URL))
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
	TargetDir      string // diretório raiz do RootFS (ex: /mnt/lfs)
	KernelVersion  string // versão do kernel Linux (ex: "6.6")
	BusyboxVersion string // versão do Busybox (ex: "1.36.1")
	UEFI           bool   // gerar estrutura UEFI
	SkipKernel     bool   // pular compilação do kernel
	SkipBusybox    bool   // pular compilação do busybox
	SkipInitramfs  bool   // pular geração do initramfs
	SkipGRUB       bool   // pular instalação do GRUB
}

// BootstrapAll executa o pipeline completo de construção do RootFS
func BootstrapAll(ctx context.Context, cfg BootstrapConfig) error {
	if cfg.TargetDir == "" {
		cfg.TargetDir = resolveLFSTarget("")
	}

	fmt.Printf("[bootstrap] Starting full bootstrap at %s\n", cfg.TargetDir)
	fmt.Printf("[bootstrap] Config: kernel=%s busybox=%s uefi=%v\n",
		cfg.KernelVersion, cfg.BusyboxVersion, cfg.UEFI)

	// 1. Create RootFS structure (FHS directories)
	fmt.Println("\n=== Step 1/6: Create RootFS ===")
	if err := CreateRootFS(cfg.TargetDir); err != nil {
		return fmt.Errorf("create rootfs: %w", err)
	}

	// 2. Download toolchain
	fmt.Println("\n=== Step 2/6: Download Toolchain ===")
	if err := DownloadToolchain(cfg.TargetDir); err != nil {
		return fmt.Errorf("download toolchain: %w", err)
	}

	// 3. Build kernel
	if !cfg.SkipKernel {
		fmt.Println("\n=== Step 3/6: Build Kernel ===")
		kernelCfg := KernelConfig{
			Version:    cfg.KernelVersion,
			SourcesDir: filepath.Join(cfg.TargetDir, "sources"),
			OutputDir:  filepath.Join(cfg.TargetDir, "boot"),
			Defconfig:  "x86_64_defconfig",
		}
		if err := BuildKernel(kernelCfg); err != nil {
			return fmt.Errorf("build kernel: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 3/6: Build Kernel (skipped) ===")
	}

	// 4. Build busybox
	if !cfg.SkipBusybox {
		fmt.Println("\n=== Step 4/6: Build Busybox ===")
		busyboxCfg := BusyboxConfig{
			Version:   cfg.BusyboxVersion,
			TargetDir: cfg.TargetDir,
		}
		if err := BuildBusybox(ctx, busyboxCfg); err != nil {
			return fmt.Errorf("build busybox: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 4/6: Build Busybox (skipped) ===")
	}

	// 5. Build initramfs
	if !cfg.SkipInitramfs {
		fmt.Println("\n=== Step 5/6: Build Initramfs ===")
		outputDir := filepath.Join(cfg.TargetDir, "boot")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("mkdir boot: %w", err)
		}
		initramfsPath := filepath.Join(outputDir, "initramfs.img")
		if err := BuildInitramfs(ctx, cfg.TargetDir, initramfsPath); err != nil {
			return fmt.Errorf("build initramfs: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 5/6: Build Initramfs (skipped) ===")
	}

	// 6. Install GRUB
	if !cfg.SkipGRUB {
		fmt.Println("\n=== Step 6/6: Install GRUB ===")
		if err := InstallGRUB(ctx, cfg.TargetDir, cfg.UEFI); err != nil {
			return fmt.Errorf("install grub: %w", err)
		}
	} else {
		fmt.Println("\n=== Step 6/6: Install GRUB (skipped) ===")
	}

	fmt.Printf("\n[bootstrap] ✅ Bootstrap complete at %s\n", cfg.TargetDir)
	return nil
}
