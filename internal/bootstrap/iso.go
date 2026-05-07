package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ISOConfig contém os parâmetros de geração da ISO
type ISOConfig struct {
	Name          string
	Version       string // versão do OS (label da ISO)
	KernelVersion string // versão do kernel (ex: "6.6") — usada para localizar vmlinuz-<KernelVersion>
	RootFS        string
	Output        string
	UEFI          bool
	BootLabel     string
}

// findVmlinuz localiza o vmlinuz dentro de rootfs/boot/.
// Se kernelVersion for informado, usa vmlinuz-<kernelVersion>.
// Caso contrário, faz glob em boot/vmlinuz-* e retorna o primeiro encontrado.
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

// GenerateISO cria uma imagem ISO bootável a partir do RootFS.
func GenerateISO(cfg ISOConfig) error {
	if cfg.RootFS == "" || cfg.Output == "" {
		return fmt.Errorf("RootFS and Output are required")
	}

	isoTmpBase := tmpDir()
	if err := os.MkdirAll(isoTmpBase, 0755); err != nil {
		return fmt.Errorf("mkdir iso tmp base: %w", err)
	}

	workDir, err := os.MkdirTemp(isoTmpBase, "iso-*")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	isoDir := filepath.Join(workDir, "iso")
	bootDir := filepath.Join(isoDir, "boot")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return fmt.Errorf("mkdir bootDir: %w", err)
	}

	// Localiza vmlinuz (por versão ou glob)
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

	// Copia initramfs real ou cria stub
	initramfsSrc := filepath.Join(cfg.RootFS, "boot", "initramfs.img")
	if data, err := os.ReadFile(initramfsSrc); err == nil {
		fmt.Printf("[iso] using real initramfs from %s (%d bytes)\n", initramfsSrc, len(data))
		if err := os.WriteFile(filepath.Join(bootDir, "initramfs.img"), data, 0644); err != nil {
			return fmt.Errorf("write initramfs: %w", err)
		}
	} else {
		fmt.Printf("[iso] real initramfs not found at %s, creating stub\n", initramfsSrc)
		if err := createinitramfs(filepath.Join(bootDir, "initramfs.img")); err != nil {
			return fmt.Errorf("create initramfs stub: %w", err)
		}
	}

	// Bootloader
	if cfg.UEFI {
		if err := setupGRUBUEFI(isoDir, bootDir, cfg); err != nil {
			return err
		}
	} else {
		if err := setupIsolinux(isoDir, cfg); err != nil {
			return err
		}
	}

	return runXorriso(isoDir, cfg)
}

func createinitramfs(output string) error {
	initTmpBase := tmpDir()
	if err := os.MkdirAll(initTmpBase, 0755); err != nil {
		return fmt.Errorf("mkdir initramfs tmp base: %w", err)
	}
	initDir, err := os.MkdirTemp(initTmpBase, "initramfs-*")
	if err != nil {
		return fmt.Errorf("create initramfs temp dir: %w", err)
	}
	defer os.RemoveAll(initDir)
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
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("cd %s && find . | cpio -o -H newc | gzip > %s", initDir, output))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cpio: %w", err)
	}
	return nil
}

func setupGRUBUEFI(isoDir, bootDir string, cfg ISOConfig) error {
	efiDir := filepath.Join(isoDir, "EFI", "BOOT")
	if err := os.MkdirAll(efiDir, 0755); err != nil {
		return fmt.Errorf("mkdir efiDir: %w", err)
	}
	grubDir := filepath.Join(bootDir, "grub")
	if err := os.MkdirAll(grubDir, 0755); err != nil {
		return fmt.Errorf("mkdir grubDir: %w", err)
	}
	// console=ttyS0,115200 garante output serial no QEMU headless
	grubCfg := fmt.Sprintf(`set timeout=5
set default=0

menuentry "%s %s" {
    linux /boot/vmlinuz root=/dev/sda1 console=ttyS0,115200 quiet
    initrd /boot/initramfs.img
}
`, cfg.Name, cfg.Version)
	if err := os.WriteFile(filepath.Join(grubDir, "grub.cfg"), []byte(grubCfg), 0644); err != nil {
		return fmt.Errorf("write grub.cfg: %w", err)
	}
	if err := exec.Command("grub-mkstandalone",
		"-O", "x86_64-efi",
		"--fonts=unicode",
		"-o", filepath.Join(efiDir, "BOOTX64.EFI"),
		"boot/grub/grub.cfg="+filepath.Join(grubDir, "grub.cfg"),
	).Run(); err != nil {
		return fmt.Errorf("grub-mkstandalone: %w", err)
	}
	return nil
}

func setupIsolinux(isoDir string, cfg ISOConfig) error {
	dir := filepath.Join(isoDir, "isolinux")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir isolinux: %w", err)
	}

	candidates := []string{
		"/usr/lib/ISOLINUX/isolinux.bin",
		"/usr/lib/syslinux/bios/isolinux.bin",
		"/usr/lib/syslinux/isolinux.bin",
		"/usr/share/syslinux/isolinux.bin",
		"/usr/lib/ISOLINUX/isolinux.bin",
	}
	var isolinuxBin string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			isolinuxBin = p
			break
		}
	}
	if isolinuxBin == "" {
		return fmt.Errorf("isolinux.bin not found — install syslinux")
	}
	data, err := os.ReadFile(isolinuxBin)
	if err != nil {
		return fmt.Errorf("read isolinux.bin: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "isolinux.bin"), data, 0644); err != nil {
		return fmt.Errorf("write isolinux.bin: %w", err)
	}

	// console=ttyS0,115200 garante output serial no QEMU headless
	cfgContent := `DEFAULT agnostic
TIMEOUT 50
LABEL agnostic
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=/dev/sda1 console=ttyS0,115200 quiet
`
	if err := os.WriteFile(filepath.Join(dir, "isolinux.cfg"), []byte(cfgContent), 0644); err != nil {
		return fmt.Errorf("write isolinux.cfg: %w", err)
	}
	return nil
}

func runXorriso(isoDir string, cfg ISOConfig) error {
	args := []string{"-as", "mkisofs", "-o", cfg.Output, "-V", cfg.BootLabel, "-J", "-R"}
	if cfg.UEFI {
		args = append(args, "-eltorito-alt-boot", "-e", "EFI/BOOT/BOOTX64.EFI", "-no-emul-boot")
	} else {
		args = append(args,
			"-b", "isolinux/isolinux.bin",
			"-c", "isolinux/boot.cat",
			"-no-emul-boot", "-boot-load-size", "4", "-boot-info-table")
	}
	args = append(args, isoDir)
	cmd := exec.Command("xorriso", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
