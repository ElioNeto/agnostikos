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

	// Criar /init script — mostra a mensagem de boas-vindas e abre um shell
	// interativo. O usuário digita comandos. Ao digitar exit ou Ctrl+D,
	// o sistema desliga.
	initScript := `#!/bin/sh
export PATH=/bin:/sbin:/usr/bin:/usr/sbin
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
echo "Welcome to Agnostikos minimal system"
echo ""
echo "Type 'exit' or press Ctrl+D to power off."
echo ""

# Run interactive shell. On exit, power off.
/bin/sh
poweroff -f
`
	initPath := filepath.Join(initDir, "init")
	if err := os.WriteFile(initPath, []byte(initScript), 0755); err != nil {
		return fmt.Errorf("write /init script: %w", err)
	}

	// Copiar binários do busybox para dentro do initramfs.
	// Copiamos apenas o binário principal busybox e criamos links simbólicos
	// para os applets necessários na inicialização: sh, mount, poweroff, sleep.
	busyboxInstall := filepath.Join(rootfsDir, "busybox-install")
	busyboxBin := filepath.Join(busyboxInstall, "bin", "busybox")
	if _, err := os.Stat(busyboxBin); err == nil {
		fmt.Printf("[initramfs] using busybox binary from %s\n", busyboxBin)

		// Copia o binário busybox para bin/
		targetBin := filepath.Join(initDir, "bin")
		if err := os.MkdirAll(targetBin, 0755); err != nil {
			return fmt.Errorf("mkdir bin: %w", err)
		}
		data, err := os.ReadFile(busyboxBin)
		if err != nil {
			return fmt.Errorf("read busybox: %w", err)
		}
		destPath := filepath.Join(targetBin, "busybox")
		if err := os.WriteFile(destPath, data, 0755); err != nil {
			return fmt.Errorf("write busybox: %w", err)
		}

		// Cria links simbólicos para os applets essenciais de inicialização
		applets := []string{"sh", "mount", "poweroff", "sleep", "reboot", "halt", "dmesg", "cat", "echo"}
		for _, a := range applets {
			linkPath := filepath.Join(targetBin, a)
			if err := os.Symlink("busybox", linkPath); err != nil {
				return fmt.Errorf("symlink %s: %w", a, err)
			}
		}

		// Também copia sbin/init e switch_root se existirem
		// (busybox install cria sbin/init como symlink para ../bin/busybox)
		sbinDir := filepath.Join(initDir, "sbin")
		if err := os.MkdirAll(sbinDir, 0755); err != nil {
			return fmt.Errorf("mkdir sbin: %w", err)
		}
		// init, switch_root são essenciais
		for _, a := range []string{"init", "switch_root"} {
			srcPath := filepath.Join(busyboxInstall, "sbin", a)
			if _, err := os.Stat(srcPath); err == nil {
				linkPath := filepath.Join(sbinDir, a)
				if err := os.Symlink("../bin/busybox", linkPath); err != nil {
					return fmt.Errorf("symlink sbin/%s: %w", a, err)
				}
			}
		}
	} else {
		fmt.Printf("[initramfs] warn: busybox not found at %s — initramfs will have no shell\n", busyboxBin)
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
