package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ISOConfig contém os parâmetros de geração da ISO
type ISOConfig struct {
	Name      string
	Version   string
	RootFS    string
	Output    string
	UEFI      bool
	BootLabel string
}

// GenerateISO cria uma imagem ISO bootável a partir do RootFS
func GenerateISO(cfg ISOConfig) error {
	if cfg.RootFS == "" || cfg.Output == "" {
		return fmt.Errorf("RootFS and Output are required")
	}

	workDir, _ := os.MkdirTemp("", "agnostikos-iso-*")
	defer os.RemoveAll(workDir)

	isoDir := filepath.Join(workDir, "iso")
	bootDir := filepath.Join(isoDir, "boot")
	if err := os.MkdirAll(bootDir, 0755); err != nil {
		return fmt.Errorf("mkdir bootDir: %w", err)
	}

	// Copy kernel
	kernelSrc := filepath.Join(cfg.RootFS, "boot", "vmlinuz-"+cfg.Version)
	data, err := os.ReadFile(kernelSrc)
	if err != nil {
		return fmt.Errorf("kernel not found at %s: %w", kernelSrc, err)
	}
	if err := os.WriteFile(filepath.Join(bootDir, "vmlinuz"), data, 0644); err != nil {
		return fmt.Errorf("write vmlinuz: %w", err)
	}

	// Create initramfs stub
	if err := createinitramfs(filepath.Join(bootDir, "initramfs.img")); err != nil {
		return fmt.Errorf("create initramfs: %w", err)
	}

	// Bootloader setup
	if cfg.UEFI {
		if err := setupGRUBUEFI(isoDir, bootDir, cfg); err != nil {
			return err
		}
	} else {
		if err := setupIsolinux(isoDir, cfg); err != nil {
			return err
		}
	}

	// Generate ISO with xorriso
	return runXorriso(isoDir, cfg)
}

func createinitramfs(output string) error {
	initDir, _ := os.MkdirTemp("", "initramfs-*")
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
	grubCfg := fmt.Sprintf(`set timeout=5
set default=0

menuentry "%s %s" {
    linux /boot/vmlinuz root=/dev/sda1 quiet
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
	cfgContent := `DEFAULT agnostic
TIMEOUT 50
LABEL agnostic
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=/dev/sda1 quiet
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
