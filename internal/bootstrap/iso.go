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
	os.MkdirAll(bootDir, 0755)

	// Copy kernel
	kernelSrc := filepath.Join(cfg.RootFS, "boot", "vmlinuz-"+cfg.Version)
	data, err := os.ReadFile(kernelSrc)
	if err != nil {
		return fmt.Errorf("kernel not found at %s: %w", kernelSrc, err)
	}
	os.WriteFile(filepath.Join(bootDir, "vmlinuz"), data, 0644)

	// Create initramfs stub
	createinitramfs(filepath.Join(bootDir, "initramfs.img"))

	// Bootloader setup
	if cfg.UEFI {
		setupGRUBUEFI(isoDir, bootDir, cfg)
	} else {
		setupIsolinux(isoDir, cfg)
	}

	// Generate ISO with xorriso
	return runXorriso(isoDir, cfg)
}

func createinitramfs(output string) {
	initDir, _ := os.MkdirTemp("", "initramfs-*")
	defer os.RemoveAll(initDir)
	for _, d := range []string{"bin", "dev", "etc", "proc", "sys", "mnt/root"} {
		os.MkdirAll(filepath.Join(initDir, d), 0755)
	}
	init := `#!/bin/sh
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
exec switch_root /mnt/root /sbin/init
`
	os.WriteFile(filepath.Join(initDir, "init"), []byte(init), 0755)
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("cd %s && find . | cpio -o -H newc | gzip > %s", initDir, output))
	cmd.Run()
}

func setupGRUBUEFI(isoDir, bootDir string, cfg ISOConfig) {
	efiDir := filepath.Join(isoDir, "EFI", "BOOT")
	os.MkdirAll(efiDir, 0755)
	grubDir := filepath.Join(bootDir, "grub")
	os.MkdirAll(grubDir, 0755)
	grubCfg := fmt.Sprintf(`set timeout=5
set default=0

menuentry "%s %s" {
    linux /boot/vmlinuz root=/dev/sda1 quiet
    initrd /boot/initramfs.img
}
`, cfg.Name, cfg.Version)
	os.WriteFile(filepath.Join(grubDir, "grub.cfg"), []byte(grubCfg), 0644)
	// grub-mkstandalone generates EFI/BOOT/BOOTX64.EFI
	exec.Command("grub-mkstandalone",
		"-O", "x86_64-efi",
		"--fonts=unicode",
		"-o", filepath.Join(efiDir, "BOOTX64.EFI"),
		"boot/grub/grub.cfg="+filepath.Join(grubDir, "grub.cfg"),
	).Run()
}

func setupIsolinux(isoDir string, cfg ISOConfig) {
	dir := filepath.Join(isoDir, "isolinux")
	os.MkdirAll(dir, 0755)
	cfgContent := fmt.Sprintf(`DEFAULT agnostic
TIMEOUT 50
LABEL agnostic
    KERNEL /boot/vmlinuz
    APPEND initrd=/boot/initramfs.img root=/dev/sda1 quiet
`)
	os.WriteFile(filepath.Join(dir, "isolinux.cfg"), []byte(cfgContent), 0644)
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
