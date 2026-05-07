package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BusyboxConfig contém os parâmetros de compilação do Busybox
type BusyboxConfig struct {
	Version   string // ex: "1.36.1"
	TargetDir string // ex: "/mnt/lfs"
	NumCPUs   string // ex: "4" (opcional; usa nproc se vazio)
}

// BuildBusybox baixa, configura, compila e instala o Busybox no diretório alvo
func BuildBusybox(ctx context.Context, cfg BusyboxConfig) error {
	if cfg.Version == "" {
		return fmt.Errorf("busybox version is required")
	}
	if cfg.TargetDir == "" {
		return fmt.Errorf("target directory is required")
	}

	srcDir := filepath.Join(cfg.TargetDir, "sources")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("mkdir sources %s: %w", srcDir, err)
	}

	busyboxDir := filepath.Join(srcDir, "busybox-"+cfg.Version)
	tarball := fmt.Sprintf("busybox-%s.tar.bz2", cfg.Version)
	tarballPath := filepath.Join(srcDir, tarball)
	url := fmt.Sprintf("https://busybox.net/downloads/%s", tarball)

	// 1. Download
	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		fmt.Printf("[busybox] Downloading Busybox %s...\n", cfg.Version)
		dlCmd := exec.CommandContext(ctx, "wget", "-q", "--show-progress", "-O", tarballPath, url)
		dlCmd.Stdout, dlCmd.Stderr = os.Stdout, os.Stderr
		if err := dlCmd.Run(); err != nil {
			return fmt.Errorf("download busybox %s: %w", cfg.Version, err)
		}
	}

	// 2. Extract
	if _, err := os.Stat(busyboxDir); os.IsNotExist(err) {
		fmt.Println("[busybox] Extracting tarball...")
		extCmd := exec.CommandContext(ctx, "tar", "-xf", tarballPath, "-C", srcDir)
		extCmd.Stdout, extCmd.Stderr = os.Stdout, os.Stderr
		if err := extCmd.Run(); err != nil {
			return fmt.Errorf("extract busybox: %w", err)
		}
	}

	// 3. Defconfig
	fmt.Println("[busybox] Applying defconfig...")
	defCmd := exec.CommandContext(ctx, "make", "-C", busyboxDir, "defconfig")
	defCmd.Stdout, defCmd.Stderr = os.Stdout, os.Stderr
	if err := defCmd.Run(); err != nil {
		return fmt.Errorf("busybox defconfig: %w", err)
	}

	// 3b. Patch .config:
	//   - Disable CONFIG_TC (incompatible with kernel >= 6.1)
	//   - Enable CONFIG_STATIC for fully static binary (no runtime lib deps)
	dotConfig := filepath.Join(busyboxDir, ".config")
	fmt.Println("[busybox] Patching .config: disabling CONFIG_TC, enabling CONFIG_STATIC...")
	patchCmd := exec.CommandContext(ctx, "sed", "-i",
		"-e", "s/^CONFIG_TC=y/CONFIG_TC=n/",
		"-e", "s/^CONFIG_FEATURE_TC_INGRESS=y/CONFIG_FEATURE_TC_INGRESS=n/",
		"-e", "s/^# CONFIG_STATIC is not set/CONFIG_STATIC=y/",
		"-e", "s/^CONFIG_STATIC=n/CONFIG_STATIC=y/",
		dotConfig,
	)
	patchCmd.Stdout, patchCmd.Stderr = os.Stdout, os.Stderr
	if err := patchCmd.Run(); err != nil {
		return fmt.Errorf("busybox patch .config: %w", err)
	}
	// If CONFIG_STATIC line didn't exist in .config, add it
	// (check if the sed already enabled it; if not, append)
	needsAppend, err := exec.CommandContext(ctx, "sh", "-c",
		fmt.Sprintf("grep -q '^CONFIG_STATIC=y' %s || echo need_append", dotConfig)).CombinedOutput()
	if err == nil && string(needsAppend) == "need_append\n" {
		fmt.Println("[busybox] CONFIG_STATIC not in .config, appending...")
		appendCmd := exec.CommandContext(ctx, "sh", "-c",
			fmt.Sprintf("echo 'CONFIG_STATIC=y' >> %s", dotConfig))
		appendCmd.Stdout, appendCmd.Stderr = os.Stdout, os.Stderr
		if err := appendCmd.Run(); err != nil {
			return fmt.Errorf("busybox append CONFIG_STATIC: %w", err)
		}
	}

	// 4. Compile
	numCPUs := cfg.NumCPUs
	if numCPUs == "" {
		numCPUs = fmt.Sprintf("%d", runtime.NumCPU())
	}
	jobs := fmt.Sprintf("-j%s", numCPUs)
	fmt.Printf("[busybox] Compiling with %s...\n", jobs)
	makeCmd := exec.CommandContext(ctx, "make", "-C", busyboxDir, jobs)
	makeCmd.Stdout, makeCmd.Stderr = os.Stdout, os.Stderr
	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("busybox compile: %w", err)
	}

	// 5. Install
	installDir := filepath.Join(cfg.TargetDir, "busybox-install")
	fmt.Printf("[busybox] Installing to %s...\n", installDir)
	installCmd := exec.CommandContext(ctx, "make", "-C", busyboxDir,
		"install", "CONFIG_PREFIX="+installDir)
	installCmd.Stdout, installCmd.Stderr = os.Stdout, os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("busybox install: %w", err)
	}

	fmt.Printf("[busybox] Busybox %s installed to %s\n", cfg.Version, installDir)
	return nil
}
