package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ToolchainConfig contém os parâmetros de compilação da toolchain
type ToolchainConfig struct {
	TargetDir string // diretório raiz do RootFS
	NumCPUs   string // número de CPUs para make -j
	Target    string // target triple (ex: x86_64-linux-gnu); vazio = auto-detect
}

// targetTriple detecta o target triple do sistema
func targetTriple() string {
	out, err := exec.CommandContext(context.Background(), "gcc", "-dumpmachine").CombinedOutput()
	if err != nil {
		return "x86_64-linux-gnu" // fallback
	}
	return strings.TrimSpace(string(out))
}

func toolchainNumCPUs(cfg ToolchainConfig) string {
	if cfg.NumCPUs != "" {
		return cfg.NumCPUs
	}
	return strconv.Itoa(runtime.NumCPU())
}

func toolchainTarget(cfg ToolchainConfig) string {
	if cfg.Target != "" {
		return cfg.Target
	}
	return targetTriple()
}

// extractIfNeeded extrai o tarball se o diretório de origem não existir
func extractIfNeeded(ctx context.Context, tarballPath, srcDir, expectedDir string) error {
	if _, err := os.Stat(expectedDir); err == nil {
		fmt.Printf("[toolchain] already extracted: %s\n", expectedDir)
		return nil
	}
	fmt.Printf("[toolchain] extracting %s...\n", filepath.Base(tarballPath))
	cmd := exec.CommandContext(ctx, "tar", "-xf", tarballPath, "-C", srcDir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

// BuildBinutils configura, compila e instala o binutils no diretório alvo
func BuildBinutils(ctx context.Context, cfg ToolchainConfig) error {
	target := toolchainTarget(cfg)
	jobs := toolchainNumCPUs(cfg)
	srcDir := sourcesDir(cfg.TargetDir)
	pkg := "binutils-2.42"
	tarball := pkg + ".tar.xz"
	tarballPath := filepath.Join(srcDir, tarball)
	srcPath := filepath.Join(srcDir, pkg)

	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		return fmt.Errorf("binutils tarball not found at %s — run download toolchain first", tarballPath)
	}

	if err := extractIfNeeded(ctx, tarballPath, srcDir, srcPath); err != nil {
		return fmt.Errorf("extract binutils: %w", err)
	}

	buildDir := filepath.Join(srcDir, "build-binutils")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("mkdir build-binutils: %w", err)
	}

	// Verificar se já foi compilado
	stampFile := filepath.Join(buildDir, ".build-complete")
	if _, err := os.Stat(stampFile); err == nil {
		fmt.Printf("[toolchain] binutils already built at %s\n", buildDir)
		return nil
	}

	fmt.Println("[toolchain] Configuring binutils...")
	configureCmd := exec.CommandContext(ctx, filepath.Join(srcPath, "configure"),
		"--prefix=/usr",
		"--target=" + target,
		"--disable-nls",
		"--disable-werror",
		"--enable-gprofng=no",
		"--enable-64-bit-bfd",
	)
	configureCmd.Dir = buildDir
	configureCmd.Stdout, configureCmd.Stderr = os.Stdout, os.Stderr
	if err := configureCmd.Run(); err != nil {
		return fmt.Errorf("binutils configure: %w", err)
	}

	fmt.Println("[toolchain] Compiling binutils...")
	makeCmd := exec.CommandContext(ctx, "make", "-j"+jobs)
	makeCmd.Dir = buildDir
	makeCmd.Stdout, makeCmd.Stderr = os.Stdout, os.Stderr
	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("binutils make: %w", err)
	}

	fmt.Println("[toolchain] Installing binutils...")
	installDir := cfg.TargetDir
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("mkdir install dir: %w", err)
	}
	installCmd := exec.CommandContext(ctx, "make", "install",
		"DESTDIR=" + installDir)
	installCmd.Dir = buildDir
	installCmd.Stdout, installCmd.Stderr = os.Stdout, os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("binutils install: %w", err)
	}

	// Stamp
	if err := os.WriteFile(stampFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("write stamp: %w", err)
	}

	fmt.Printf("[toolchain] binutils installed to %s\n", installDir)
	return nil
}

// BuildGCC configura, compila e instala o GCC (pass 1, linguagem C only)
func BuildGCC(ctx context.Context, cfg ToolchainConfig) error {
	target := toolchainTarget(cfg)
	jobs := toolchainNumCPUs(cfg)
	srcDir := sourcesDir(cfg.TargetDir)
	pkg := "gcc-14.1.0"
	tarball := pkg + ".tar.xz"
	tarballPath := filepath.Join(srcDir, tarball)
	srcPath := filepath.Join(srcDir, pkg)

	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		return fmt.Errorf("gcc tarball not found at %s — run download toolchain first", tarballPath)
	}

	if err := extractIfNeeded(ctx, tarballPath, srcDir, srcPath); err != nil {
		return fmt.Errorf("extract gcc: %w", err)
	}

	// Baixar dependências do GCC (GMP, MPFR, MPC, ISL)
	fmt.Println("[toolchain] Downloading GCC prerequisites...")
	dlCmd := exec.CommandContext(ctx, "make", "-C", srcPath, "graphite=", "fetch")
	dlCmd.Stdout, dlCmd.Stderr = os.Stdout, os.Stderr
	_ = dlCmd.Run() // best-effort

	buildDir := filepath.Join(srcDir, "build-gcc")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("mkdir build-gcc: %w", err)
	}

	stampFile := filepath.Join(buildDir, ".build-complete")
	if _, err := os.Stat(stampFile); err == nil {
		fmt.Printf("[toolchain] gcc already built at %s\n", buildDir)
		return nil
	}

	fmt.Println("[toolchain] Configuring GCC (pass 1, C only)...")
	configureCmd := exec.CommandContext(ctx, filepath.Join(srcPath, "configure"),
		"--prefix=/usr",
		"--target=" + target,
		"--enable-languages=c",
		"--disable-nls",
		"--disable-libstdcxx-pch",
		"--disable-multilib",
		"--disable-bootstrap",
		"--without-headers",
	)
	configureCmd.Dir = buildDir
	configureCmd.Stdout, configureCmd.Stderr = os.Stdout, os.Stderr
	if err := configureCmd.Run(); err != nil {
		return fmt.Errorf("gcc configure: %w", err)
	}

	fmt.Println("[toolchain] Compiling GCC...")
	makeCmd := exec.CommandContext(ctx, "make", "-j"+jobs)
	makeCmd.Dir = buildDir
	makeCmd.Stdout, makeCmd.Stderr = os.Stdout, os.Stderr
	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("gcc make: %w", err)
	}

	fmt.Println("[toolchain] Installing GCC...")
	installDir := cfg.TargetDir
	installCmd := exec.CommandContext(ctx, "make", "install",
		"DESTDIR=" + installDir)
	installCmd.Dir = buildDir
	installCmd.Stdout, installCmd.Stderr = os.Stdout, os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("gcc install: %w", err)
	}

	if err := os.WriteFile(stampFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("write stamp: %w", err)
	}

	fmt.Printf("[toolchain] gcc installed to %s\n", installDir)
	return nil
}

// BuildGLibc configura, compila e instala a glibc
func BuildGLibc(ctx context.Context, cfg ToolchainConfig) error {
	target := toolchainTarget(cfg)
	jobs := toolchainNumCPUs(cfg)
	srcDir := sourcesDir(cfg.TargetDir)
	pkg := "glibc-2.39"
	tarball := pkg + ".tar.xz"
	tarballPath := filepath.Join(srcDir, tarball)
	srcPath := filepath.Join(srcDir, pkg)

	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		return fmt.Errorf("glibc tarball not found at %s — run download toolchain first", tarballPath)
	}

	if err := extractIfNeeded(ctx, tarballPath, srcDir, srcPath); err != nil {
		return fmt.Errorf("extract glibc: %w", err)
	}

	buildDir := filepath.Join(srcDir, "build-glibc")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("mkdir build-glibc: %w", err)
	}

	stampFile := filepath.Join(buildDir, ".build-complete")
	if _, err := os.Stat(stampFile); err == nil {
		fmt.Printf("[toolchain] glibc already built at %s\n", buildDir)
		return nil
	}

	fmt.Println("[toolchain] Configuring glibc...")
	configureCmd := exec.CommandContext(ctx, filepath.Join(srcPath, "configure"),
		"--prefix=/usr",
		"--host=" + target,
		"--disable-nls",
		"--enable-kernel=4.19",
	)
	configureCmd.Dir = buildDir
	configureCmd.Stdout, configureCmd.Stderr = os.Stdout, os.Stderr
	if err := configureCmd.Run(); err != nil {
		return fmt.Errorf("glibc configure: %w", err)
	}

	fmt.Println("[toolchain] Compiling glibc...")
	makeCmd := exec.CommandContext(ctx, "make", "-j"+jobs)
	makeCmd.Dir = buildDir
	makeCmd.Stdout, makeCmd.Stderr = os.Stdout, os.Stderr
	if err := makeCmd.Run(); err != nil {
		return fmt.Errorf("glibc make: %w", err)
	}

	fmt.Println("[toolchain] Installing glibc...")
	installDir := cfg.TargetDir
	installCmd := exec.CommandContext(ctx, "make", "install",
		"DESTDIR=" + installDir)
	installCmd.Dir = buildDir
	installCmd.Stdout, installCmd.Stderr = os.Stdout, os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("glibc install: %w", err)
	}

	if err := os.WriteFile(stampFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("write stamp: %w", err)
	}

	fmt.Printf("[toolchain] glibc installed to %s\n", installDir)
	return nil
}
