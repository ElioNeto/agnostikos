package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BuildInitramfs cria um initramfs com /init script e empacota com cpio | gzip
func BuildInitramfs(ctx context.Context, rootfsDir, outputPath string) error {
	if rootfsDir == "" {
		return errors.New("rootfs directory is required")
	}
	if outputPath == "" {
		return errors.New("output path is required")
	}

	initDir, err := os.MkdirTemp("", "initramfs-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(initDir) }()

	// Criar diretórios essenciais
	for _, d := range []string{"bin", "dev", "etc", "proc", "sys", "mnt/root"} {
		path := filepath.Join(initDir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	// Criar /init script — monta sistema de arquivos virtual, configura rede,
	// verifica se há um RootFS real em /mnt/root para switch_root, e se não
	// houver, abre um shell interativo com todas as ferramentas do Busybox.
	initScript := `#!/bin/sh
export PATH=/bin:/sbin:/usr/bin:/usr/sbin

# Mount virtual filesystems
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
mkdir -p /dev/pts /dev/shm
mount -t devpts none /dev/pts
mount -t tmpfs none /dev/shm

# Configure loopback network interface
ip link set lo up

# Try to bring up eth0 with DHCP
ip link set eth0 up 2>/dev/null
udhcpc -i eth0 -q -n 2>/dev/null || true

echo ""
echo "================================================"
echo "  Welcome to Agnostikos minimal system"
echo "  Kernel: $(uname -r)"
echo "================================================"
echo ""

# Check if there's a real RootFS to switch to
if [ -x /mnt/root/sbin/init ]; then
    echo "[init] RootFS found at /mnt/root, switching root..."
    mount --move /proc /mnt/root/proc
    mount --move /sys /mnt/root/sys
    mount --move /dev /mnt/root/dev
    exec switch_root /mnt/root /sbin/init
fi

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

		// Cria links simbólicos para os applets do Busybox.
		// Estes são os comandos disponíveis no shell interativo de resgate.
		applets := []string{
			"sh", "mount", "poweroff", "sleep", "reboot", "halt",
			"dmesg", "cat", "echo", "udhcpc",
			// Navegação e arquivos
			"ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "rmdir", "touch",
			"ln", "find", "grep", "sed", "awk", "more", "less", "head", "tail",
			"sort", "cut", "tr", "uniq", "wc", "tee",
			// Sistema e processos
			"ps", "top", "free", "df", "du", "uname", "id", "whoami",
			"kill", "killall", "pgrep", "pkill", "uptime", "watch",
			"clear", "reset", "env", "printenv", "set",
			"chmod", "chown", "chroot", "mknod", "mkfifo",
			// Rede
			"ip", "ifconfig", "ping", "netstat", "nslookup",
			"nc", "telnet", "wget", "tftp",
			// Compressão e arquivamento
			"tar", "gzip", "gunzip", "bzip2", "bunzip2", "xz", "unxz",
			"losetup", "blkid",
			// Editores e utilitários
			"vi", "ed", "patch", "diff", "cmp",
			"md5sum", "sha1sum", "sha256sum", "sha512sum",
			"base64", "xxd", "hexdump", "od",
			// Montagem e disco
			"umount", "swapon", "swapoff", "fdisk", "fsck", "mkfs.ext2",
			"dd", "sync",
			// Shell e scripting
			"test", "expr", "xargs", "which", "dirname", "basename",
			"date", "cal", "time", "sleep",
			// Log e debug
			"logger", "logread", "sysctl", "stty",
		}
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
