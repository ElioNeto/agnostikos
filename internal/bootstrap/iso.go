package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ISOConfig contém os parâmetros de geração da ISO
type ISOConfig struct {
	Name          string
	Version       string
	KernelVersion string
	RootFS        string
	Output        string
	InitramfsPath string // caminho opcional para initramfs; vazio = RootFS/boot/initramfs.img
	UEFI          bool
	BootLabel     string
	TestMode      bool // quando true, gera initramfs mínimo sem busybox para teste
}

func findVmlinuz(rootfs, kernelVersion string) (string, error) {
	bootDir := filepath.Join(rootfs, "boot")
	if kernelVersion != "" {
		p := filepath.Join(bootDir, "vmlinuz-"+kernelVersion)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("kernel not found at %s: %w", p, err)
		}
		return p, nil
	}
	matches, err := filepath.Glob(filepath.Join(bootDir, "vmlinuz-*"))
	if err != nil {
		return "", fmt.Errorf("glob vmlinuz: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no vmlinuz-* found in %s — run 'make bootstrap' first", bootDir)
	}
	fmt.Printf("[iso] using kernel: %s\n", matches[0])
	return matches[0], nil
}

func GenerateISO(cfg ISOConfig) error {
	if cfg.RootFS == "" || cfg.Output == "" {
		return errors.New("RootFS and Output are required")
	}

	isoTmpBase := tmpDir()
	if err := os.MkdirAll(isoTmpBase, 0755); err != nil {
		return fmt.Errorf("mkdir iso tmp base: %w", err)
	}
	workDir, err := os.MkdirTemp(isoTmpBase, "iso-*")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	isoDir := filepath.Join(workDir, "iso")
	bootDir := filepath.Join(isoDir, "boot")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return fmt.Errorf("mkdir bootDir: %w", err)
	}

	kernelSrc, err := findVmlinuz(cfg.RootFS, cfg.KernelVersion)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(kernelSrc)
	if err != nil {
		return fmt.Errorf("read kernel: %w", err)
	}
	if err := os.WriteFile(filepath.Join(bootDir, "vmlinuz"), data, 0644); err != nil {
		return fmt.Errorf("write vmlinuz: %w", err)
	}

	initramfsSrc := cfg.InitramfsPath
	if initramfsSrc == "" {
		initramfsSrc = filepath.Join(cfg.RootFS, "boot", "initramfs.img")
	}
	if data, err := os.ReadFile(initramfsSrc); err == nil {
		fmt.Printf("[iso] using real initramfs from %s (%d bytes)\n", initramfsSrc, len(data))
		if err := os.WriteFile(filepath.Join(bootDir, "initramfs.img"), data, 0644); err != nil {
			return fmt.Errorf("write initramfs: %w", err)
		}
	} else {
		fmt.Printf("[iso] real initramfs not found at %s, creating stub\n", initramfsSrc)
		if err := createinitramfs(filepath.Join(bootDir, "initramfs.img"), cfg.TestMode); err != nil {
			return fmt.Errorf("create initramfs stub: %w", err)
		}
	}

	// Criar grub.cfg básico que será usado pelo grub-mkrescue
	grubDir := filepath.Join(bootDir, "grub")
	if err := os.MkdirAll(grubDir, 0755); err != nil {
		return fmt.Errorf("mkdir grubDir: %w", err)
	}
	grubCfg := fmt.Sprintf(`set timeout=5
set default=0

menuentry "%s %s" {
    linux /boot/vmlinuz console=ttyS0,115200
    initrd /boot/initramfs.img
}
`, cfg.Name, cfg.Version)
	grubCfgPath := filepath.Join(grubDir, "grub.cfg")
	if err := os.WriteFile(grubCfgPath, []byte(grubCfg), 0644); err != nil {
		return fmt.Errorf("write grub.cfg: %w", err)
	}

	return runGrubMkrescue(isoDir, cfg)
}

func createinitramfs(output string, testMode bool) error {
	initTmpBase := tmpDir()
	if err := os.MkdirAll(initTmpBase, 0755); err != nil {
		return fmt.Errorf("mkdir initramfs tmp base: %w", err)
	}
	initDir, err := os.MkdirTemp(initTmpBase, "initramfs-*")
	if err != nil {
		return fmt.Errorf("create initramfs temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(initDir) }()

	if testMode {
		// Initramfs mínimo de teste: inclui busybox estático para o shell.
		// Monta VFS, imprime "Welcome to Agnostikos" e desliga.
		for _, d := range []string{"bin", "dev", "proc", "sys"} {
			if err := os.MkdirAll(filepath.Join(initDir, d), 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", d, err)
			}
		}
		// Copia busybox do host para o initramfs (precisa ser estático)
		busyboxPath, err := exec.LookPath("busybox")
		if err != nil {
			return fmt.Errorf("busybox not found on host — required for test initramfs: %w", err)
		}
		bbData, err := os.ReadFile(busyboxPath)
		if err != nil {
			return fmt.Errorf("read busybox binary: %w", err)
		}
		bbDest := filepath.Join(initDir, "bin", "busybox")
		if err := os.WriteFile(bbDest, bbData, 0755); err != nil {
			return fmt.Errorf("write busybox: %w", err)
		}
		// Symlinks para applets do busybox (cada applet é um symlink para busybox;
		// busybox detecta argv[0] e executa o applet correspondente)
		for _, applet := range []string{"sh", "mount", "poweroff", "uname"} {
			if err := os.Symlink("busybox", filepath.Join(initDir, "bin", applet)); err != nil {
				return fmt.Errorf("symlink bin/%s: %w", applet, err)
			}
		}
		init := `#!/bin/sh
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
echo ""
echo "================================================"
echo "  Welcome to Agnostikos"
echo "  Kernel: $(uname -r)"
echo "================================================"
echo ""
poweroff -f
`
		if err := os.WriteFile(filepath.Join(initDir, "init"), []byte(init), 0755); err != nil {
			return fmt.Errorf("write test init: %w", err)
		}
	} else {
		for _, d := range []string{"bin", "dev", "etc", "proc", "sys", "mnt/root"} {
			if err := os.MkdirAll(filepath.Join(initDir, d), 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", d, err)
			}
		}
		init := `#!/bin/sh
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
exec switch_root /mnt/root /sbin/init
`
		if err := os.WriteFile(filepath.Join(initDir, "init"), []byte(init), 0755); err != nil {
			return fmt.Errorf("write init: %w", err)
		}
	}

	cmd := exec.CommandContext(context.Background(), "sh", "-c",
		fmt.Sprintf("cd %s && find . | cpio -o -H newc | gzip > %s", initDir, output))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cpio: %w", err)
	}
	return nil
}

// runGrubMkrescue cria a ISO usando grub-mkrescue, que gera corretamente
// uma ISO híbrida com suporte a BIOS e UEFI, incluindo System Area (GPT/MBR)
// necessária para boot OVMF.
func runGrubMkrescue(isoDir string, cfg ISOConfig) error {
	if _, err := exec.LookPath("grub-mkrescue"); err != nil {
		return errors.New("grub-mkrescue not found — install grub-common and grub-efi-amd64-bin")
	}

	label := cfg.BootLabel
	if label == "" {
		label = "AgnostikOS"
	}

	args := []string{
		"-o", cfg.Output,
		"-V", label,
		isoDir,
	}

	fmt.Printf("[iso] running grub-mkrescue to create hybrid ISO: %s\n", cfg.Output)
	cmd := exec.CommandContext(context.Background(), "grub-mkrescue", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("grub-mkrescue failed: %w", err)
	}
	return nil
}
