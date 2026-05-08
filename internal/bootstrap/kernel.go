package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// kernelConfigFragment is a minimal kernel config fragment that enables essential
// features for a bootable initramfs-based system: initrd, devtmpfs, serial console, ext4.
const kernelConfigFragment = `CONFIG_BLK_DEV_INITRD=y
CONFIG_DEVTMPFS=y
CONFIG_DEVTMPFS_MOUNT=y
CONFIG_SERIAL_8250_CONSOLE=y
CONFIG_EXT4_FS=y
`

// KernelConfig contém os parâmetros de compilação do kernel
type KernelConfig struct {
	Version    string // ex: "6.8.0"
	SourcesDir string // ex: "/mnt/lfs/sources"
	OutputDir  string // ex: "/mnt/lfs/boot"
	Defconfig  string // ex: "x86_64_defconfig" — auto-detected from arch if empty
	Arch       string // target arch: "amd64" or "arm64" — auto-detected if empty
}

// kernelArch mapeia Go arch names para nomes de arquitetura do kernel Linux.
// Retorna (kernel_arch, kernel_defconfig, bzImage_path).
func kernelArch(arch string) (karch, defconfig, imagePath string) {
	switch arch {
	case "arm64":
		return "arm64", "defconfig", "arch/arm64/boot/Image"
	default:
		return "x86_64", "x86_64_defconfig", "arch/x86/boot/bzImage"
	}
}

// autoDetectArch returns the host's Go arch if cfg.Arch is empty
func autoDetectArch(cfg KernelConfig) string {
	if cfg.Arch != "" {
		return cfg.Arch
	}
	// Detect from runtime
	switch runtime.GOARCH {
	case "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

// applyKernelConfigFragment writes a minimal config fragment and merges it into
// the kernel .config using scripts/kconfig/merge_config.sh.
func applyKernelConfigFragment(srcPath, karch string) error {
	fragmentPath := filepath.Join(srcPath, "kernel-config-minimal.config")
	if err := os.WriteFile(fragmentPath, []byte(kernelConfigFragment), 0644); err != nil {
		return fmt.Errorf("write config fragment: %w", err)
	}
	fmt.Printf("[kernel] Applying kernel config fragment (%s)...\n", fragmentPath)

	// Run merge_config.sh directly (not via make) since it's a shell script,
	// not a make target. -m means "only merge, don't run olddefconfig",
	// -O specifies the output directory for the merged .config.
	cmd := exec.CommandContext(context.Background(), "sh", "-c",
		fmt.Sprintf("cd %s && ARCH=%s scripts/kconfig/merge_config.sh -m -O %s %s %s",
			srcPath, karch, srcPath,
			filepath.Join(srcPath, ".config"), fragmentPath))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("merge config fragment: %w", err)
	}
	return nil
}

// BuildKernel automatiza download, configuração e compilação do Linux kernel
func BuildKernel(cfg KernelConfig) error {
	major := strings.Split(cfg.Version, ".")[0]
	tarball := fmt.Sprintf("linux-%s.tar.xz", cfg.Version)
	srcPath := filepath.Join(cfg.SourcesDir, "linux-"+cfg.Version)
	tarballPath := filepath.Join(cfg.SourcesDir, tarball)

	arch := autoDetectArch(cfg)
	karch, defconfig, imageRelPath := kernelArch(arch)

	if err := os.MkdirAll(cfg.SourcesDir, 0755); err != nil {
		return fmt.Errorf("mkdir sourcesDir: %w", err)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("mkdir outputDir: %w", err)
	}

	// 1. Download
	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		url := fmt.Sprintf("https://cdn.kernel.org/pub/linux/kernel/v%s.x/%s", major, tarball)
		fmt.Printf("[kernel] Downloading Linux %s...\n", cfg.Version)
		cmd := exec.CommandContext(context.Background(), "wget", "-q", "--show-progress", "-O", tarballPath, url)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("download: %w", err)
		}
	}

	// 2. Extract
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		fmt.Println("[kernel] Extracting tarball...")
		cmd := exec.CommandContext(context.Background(), "tar", "-xf", tarballPath, "-C", cfg.SourcesDir)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("extract: %w", err)
		}
	}

	// 3. mrproper (best-effort: ignora erro)
	_ = exec.CommandContext(context.Background(), "make", "-C", srcPath, "mrproper").Run()

	// 4. defconfig — use cfg.Defconfig if set, otherwise auto-detect from arch
	if cfg.Defconfig == "" {
		cfg.Defconfig = defconfig
	}
	fmt.Printf("[kernel] Applying %s (arch: %s)...\n", cfg.Defconfig, arch)
	cmd := exec.CommandContext(context.Background(), "make", "-C", srcPath, "ARCH="+karch, cfg.Defconfig)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("defconfig: %w", err)
	}

	// 4b. Apply minimal kernel config fragment
	if err := applyKernelConfigFragment(srcPath, karch); err != nil {
		return fmt.Errorf("apply config fragment: %w", err)
	}

	// 5. Compile (parallel)
	imageTarget := filepath.Base(imageRelPath) // "bzImage" for x86, "Image" for arm64
	jobs := fmt.Sprintf("-j%d", runtime.NumCPU())
	fmt.Printf("[kernel] Compiling with %s (arch: %s, target: %s)...\n", jobs, arch, imageTarget)
	cmd = exec.CommandContext(context.Background(), "make", "-C", srcPath, "ARCH="+karch, jobs, imageTarget, "modules")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	// 6. Install kernel image
	src := filepath.Join(srcPath, imageRelPath)
	imageName := "vmlinuz-" + cfg.Version
	if arch == "arm64" {
		imageName = "Image-" + cfg.Version
	}
	dst := filepath.Join(cfg.OutputDir, imageName)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read kernel image from %s: %w", imageRelPath, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write kernel image: %w", err)
	}

	// 7. Install modules (best-effort: ignora erro)
	modPath := filepath.Dir(cfg.OutputDir)
	cmd = exec.CommandContext(context.Background(), "make", "-C", srcPath, "ARCH="+karch, "INSTALL_MOD_PATH="+modPath, "modules_install")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	_ = cmd.Run()

	fmt.Printf("[kernel] ✅ Kernel ready: %s\n", dst)
	return nil
}
