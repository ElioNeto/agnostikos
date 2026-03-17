package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// KernelConfig contém os parâmetros de compilação do kernel
type KernelConfig struct {
	Version    string // ex: "6.8.0"
	SourcesDir string // ex: "/mnt/lfs/sources"
	OutputDir  string // ex: "/mnt/lfs/boot"
	Defconfig  string // ex: "x86_64_defconfig"
}

// BuildKernel automatiza download, configuração e compilação do Linux kernel
func BuildKernel(cfg KernelConfig) error {
	major := strings.Split(cfg.Version, ".")[0]
	tarball := fmt.Sprintf("linux-%s.tar.xz", cfg.Version)
	srcPath := filepath.Join(cfg.SourcesDir, "linux-"+cfg.Version)
	tarballPath := filepath.Join(cfg.SourcesDir, tarball)

	os.MkdirAll(cfg.SourcesDir, 0755)
	os.MkdirAll(cfg.OutputDir, 0755)

	// 1. Download
	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		url := fmt.Sprintf("https://cdn.kernel.org/pub/linux/kernel/v%s.x/%s", major, tarball)
		fmt.Printf("[kernel] Downloading Linux %s...\n", cfg.Version)
		cmd := exec.Command("wget", "-q", "--show-progress", "-O", tarballPath, url)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("download: %w", err)
		}
	}

	// 2. Extract
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		fmt.Println("[kernel] Extracting tarball...")
		cmd := exec.Command("tar", "-xf", tarballPath, "-C", cfg.SourcesDir)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("extract: %w", err)
		}
	}

	// 3. mrproper
	exec.Command("make", "-C", srcPath, "mrproper").Run()

	// 4. defconfig
	fmt.Printf("[kernel] Applying %s...\n", cfg.Defconfig)
	cmd := exec.Command("make", "-C", srcPath, cfg.Defconfig)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("defconfig: %w", err)
	}

	// 5. Compile (parallel)
	jobs := fmt.Sprintf("-j%d", runtime.NumCPU())
	fmt.Printf("[kernel] Compiling with %s (this may take a while)...\n", jobs)
	cmd = exec.Command("make", "-C", srcPath, jobs, "bzImage", "modules")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	// 6. Install bzImage
	src := filepath.Join(srcPath, "arch/x86/boot/bzImage")
	dst := filepath.Join(cfg.OutputDir, "vmlinuz-"+cfg.Version)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read bzImage: %w", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("write vmlinuz: %w", err)
	}

	// 7. Install modules
	modPath := filepath.Dir(cfg.OutputDir)
	cmd = exec.Command("make", "-C", srcPath, "INSTALL_MOD_PATH="+modPath, "modules_install")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.Run()

	fmt.Printf("[kernel] ✅ Kernel ready: %s\n", dst)
	return nil
}
