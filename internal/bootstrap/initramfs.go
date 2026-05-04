package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BuildInitramfs cria um initramfs com /init script e empacota com cpio | gzip
func BuildInitramfs(ctx context.Context, rootfsDir, outputPath string) error {
	if rootfsDir == "" {
		return fmt.Errorf("rootfs directory is required")
	}
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	initDir, err := os.MkdirTemp("", "initramfs-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(initDir)

	// Criar diretórios essenciais
	for _, d := range []string{"bin", "dev", "etc", "proc", "sys", "mnt/root"} {
		path := filepath.Join(initDir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	// Criar /init script
	initScript := `#!/bin/sh
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
echo "Welcome to Agnostikos minimal system"
exec /bin/sh
`
	initPath := filepath.Join(initDir, "init")
	if err := os.WriteFile(initPath, []byte(initScript), 0755); err != nil {
		return fmt.Errorf("write /init script: %w", err)
	}

	// Copiar binários do busybox se existirem
	busyboxInstall := filepath.Join(rootfsDir, "busybox-install")
	if info, err := os.Stat(busyboxInstall); err == nil && info.IsDir() {
		fmt.Println("[initramfs] copying busybox binaries into initramfs...")
		copyCmd := exec.CommandContext(ctx, "cp", "-ra",
			filepath.Join(busyboxInstall, "."),
			initDir)
		copyCmd.Stdout, copyCmd.Stderr = os.Stdout, os.Stderr
		if err := copyCmd.Run(); err != nil {
			return fmt.Errorf("copy busybox to initramfs: %w", err)
		}
	}

	// Empacotar com cpio | gzip
	cmd := exec.CommandContext(ctx, "sh", "-c",
		fmt.Sprintf("cd %s && find . | cpio -H newc -o | gzip > %s",
			initDir, outputPath))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pack initramfs: %w", err)
	}

	// Verificar se o arquivo foi criado
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("initramfs was not created at %s", outputPath)
	}

	fmt.Printf("[initramfs] Initramfs created: %s\n", outputPath)
	return nil
}
