package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	for _, d := range []string{"bin", "dev", "etc", "proc", "sys", "mnt/root", "usr/bin", "usr/local/bin"} {
		path := filepath.Join(initDir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}

	// Criar /init script — monta sistema de arquivos virtual, configura rede,
	// verifica se há um RootFS real em /mnt/root para switch_root, e se não
	// houver, abre um shell interativo no VGA (tty1) com todas as ferramentas
	// do Busybox.
	initScript := `#!/bin/sh
export PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin

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

# Determine the best console device.
# Priority:
#   1. Consoles specified in kernel cmdline (console= parameter, in order)
#   2. /dev/tty1  — VGA / SPICE / virt-manager display
#   3. /dev/ttyS0 — serial console (used by virsh console, CI headless)
#   4. /dev/hvc0  — virtio console
#   5. fallback   — no controlling TTY (last resort)
#
# Kernel cmdline parsing: extract each console= parameter value,
# strip optional baud rate (e.g., "ttyS0,115200" -> "ttyS0"),
# and build the device path (e.g., "ttyS0" -> "/dev/ttyS0").
#
# setsid creates a new session; cttyhack assigns the controlling terminal
# so keyboard input goes to the correct console.
if [ -f /proc/cmdline ]; then
    CMDLINE=$(cat /proc/cmdline)
    for token in $CMDLINE; do
        case "$token" in
            console=*)
                # Extract value after "console=" and strip baud rate
                CON="${token#console=}"
                CON="${CON%%,*}"
                case "$CON" in
                    tty0)     CONS="/dev/tty1" ;;
                    ttyS0)    CONS="/dev/ttyS0" ;;
                    tty1)     CONS="/dev/tty1" ;;
                    hvc0)     CONS="/dev/hvc0" ;;
                    *)        CONS="/dev/$CON" ;;
                esac
                if [ -c "$CONS" ]; then
                    exec setsid cttyhack /bin/sh < "$CONS" > "$CONS" 2>&1
                fi
                ;;
        esac
    done
fi

# Fallback: try common console devices in order
for tty in /dev/tty1 /dev/ttyS0 /dev/hvc0; do
    if [ -c "$tty" ]; then
        exec setsid cttyhack /bin/sh < "$tty" > "$tty" 2>&1
    fi
done

# Last resort: no device console found
echo "Warning: no console device found, starting shell without controlling TTY."
exec /bin/sh
`
	initPath := filepath.Join(initDir, "init")
	if err := os.WriteFile(initPath, []byte(initScript), 0755); err != nil {
		return fmt.Errorf("write /init script: %w", err)
	}

	// Copiar binários do busybox para dentro do initramfs.
	// Copiamos apenas o binário principal busybox e criamos links simbólicos
	// para os applets necessários na inicialização.
	busyboxInstall := filepath.Join(rootfsDir, "busybox-install")
	busyboxBin := filepath.Join(busyboxInstall, "bin", "busybox")
	if _, err := os.Stat(busyboxBin); err == nil {
		fmt.Printf("[initramfs] using busybox binary from %s\n", busyboxBin)

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

		applets := []string{
			"sh", "mount", "poweroff", "sleep", "reboot", "halt",
			"dmesg", "cat", "echo", "udhcpc",
			// essenciais para shell no VGA
			"setsid", "cttyhack",
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
				if os.IsExist(err) {
					continue
				}
				return fmt.Errorf("symlink %s: %w", a, err)
			}
		}

		sbinDir := filepath.Join(initDir, "sbin")
		if err := os.MkdirAll(sbinDir, 0755); err != nil {
			return fmt.Errorf("mkdir sbin: %w", err)
		}
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

	// Instala o binário agnostic no initramfs.
	// O binário é copiado de rootfsDir/usr/bin/agnostic (instalado por installAgnosticBinary
	// durante o build do rootfs). Deve ser um binário estático (CGO_ENABLED=0).
	agnosticSrc := filepath.Join(rootfsDir, "usr", "bin", "agnostic")
	if _, err := os.Stat(agnosticSrc); err == nil {
		data, err := os.ReadFile(agnosticSrc)
		if err != nil {
			return fmt.Errorf("read agnostic binary: %w", err)
		}
		dest := filepath.Join(initDir, "usr", "bin", "agnostic")
		if err := os.WriteFile(dest, data, 0755); err != nil {
			return fmt.Errorf("write agnostic binary: %w", err)
		}
		// Symlink /usr/local/bin/agnostic -> /usr/bin/agnostic
		symlinkPath := filepath.Join(initDir, "usr", "local", "bin", "agnostic")
		_ = os.Remove(symlinkPath)
		if err := os.Symlink("/usr/bin/agnostic", symlinkPath); err != nil {
			return fmt.Errorf("symlink agnostic: %w", err)
		}
		fmt.Printf("[initramfs] installed agnostic binary from %s\n", agnosticSrc)

		// Fallback: se o binário for dinamicamente ligado, copia as bibliotecas
		// compartilhadas necessárias para dentro do initramfs.
		// Isso cobre o caso em que o binário veio de "make build" (CGO_ENABLED=default)
		// ou de uma release pré-compilada sem linkagem estática.
		if err := installAgnosticLibraries(dest, initDir); err != nil {
			fmt.Printf("[initramfs] warn: could not install shared libraries: %v\n", err)
		}
	} else {
		fmt.Printf("[initramfs] warn: agnostic binary not found at %s — run 'agnostic build' first\n", agnosticSrc)
	}

	// Empacotar com cpio | gzip
	cmd := exec.CommandContext(ctx, "sh", "-c",
		fmt.Sprintf("cd %s && find . | cpio -H newc -o | gzip > %s",
			initDir, outputPath))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pack initramfs: %w", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("initramfs was not created at %s", outputPath)
	}

	fmt.Printf("[initramfs] Initramfs created: %s\n", outputPath)
	return nil
}

// installAgnosticLibraries copia as bibliotecas compartilhadas necessárias
// pelo binário agnostic para dentro do initramfs.
// Usa ldd para detectar dependências e copia cada uma para o diretório correto.
// É um fallback para quando o binário não foi compilado com CGO_ENABLED=0.
//
// Parâmetros:
//   - binaryPath: caminho completo para o binário dentro do initramfs (ex: <initDir>/usr/bin/agnostic)
//   - initDir: diretório raiz do initramfs (ex: /tmp/initramfs-123456)
func installAgnosticLibraries(binaryPath, initDir string) error {
	// Roda ldd para listar as dependências dinâmicas
	cmd := exec.Command("ldd", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		// ldd falhou — provavelmente o binário é estático
		return nil
	}

	linhas := strings.Split(string(output), "\n")
	copied := 0
	for _, linha := range linhas {
		linha = strings.TrimSpace(linha)
		// Pula linhas vazias, vdso (virtual) e binários estáticos
		if linha == "" || strings.Contains(linha, "linux-vdso") || strings.Contains(linha, "statically linked") {
			continue
		}
		// Formato típico: libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x...)
		// Ou: /lib64/ld-linux-x86-64.so.2 (0x...)
		parts := strings.Split(linha, "=>")
		var libPath string
		if len(parts) >= 2 {
			// Formato com "=>"
			libPath = strings.TrimSpace(parts[1])
			// Remove o endereço hexadecimal no final
			if idx := strings.LastIndex(libPath, "("); idx != -1 {
				libPath = strings.TrimSpace(libPath[:idx])
			}
		} else {
			// Formato sem "=>" (ex: /lib64/ld-linux-x86-64.so.2)
			if idx := strings.LastIndex(linha, "("); idx != -1 {
				libPath = strings.TrimSpace(linha[:idx])
			} else {
				libPath = linha
			}
		}

		if libPath == "" || !strings.HasPrefix(libPath, "/") {
			continue
		}

		// Verifica se o arquivo fonte existe
		if _, err := os.Stat(libPath); os.IsNotExist(err) {
			// Tenta com /lib64/ relativo
			altPath := "/lib64/" + filepath.Base(libPath)
			if _, err2 := os.Stat(altPath); err2 == nil {
				libPath = altPath
			} else {
				continue
			}
		}

		// Cria o diretório alvo no initramfs preservando o caminho absoluto
		// Ex: libPath=/lib/x86_64-linux-gnu/libc.so.6 -> initDir/lib/x86_64-linux-gnu/libc.so.6
		destPath := filepath.Join(initDir, libPath)
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("[initramfs] warn: mkdir %s: %v\n", destDir, err)
			continue
		}

		data, err := os.ReadFile(libPath)
		if err != nil {
			fmt.Printf("[initramfs] warn: read %s: %v\n", libPath, err)
			continue
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			fmt.Printf("[initramfs] warn: write %s: %v\n", destPath, err)
			continue
		}
		copied++
		fmt.Printf("[initramfs] copied shared library: %s\n", libPath)
	}

	if copied > 0 {
		fmt.Printf("[initramfs] installed %d shared libraries for agnostic binary\n", copied)
	}
	return nil
}
