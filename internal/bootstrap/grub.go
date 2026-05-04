package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GRUBConfig contém os parâmetros de instalação do GRUB
type GRUBConfig struct {
	RootfsDir string // ex: "/mnt/data"
	Device    string // ex: "/dev/sda" — obrigatório para BIOS (disco base, sem número de partição)
	UEFI      bool   // true = x86_64-efi, false = i386-pc
}

// InstallGRUB configura o GRUB no RootFS (BIOS ou UEFI)
func InstallGRUB(ctx context.Context, cfg GRUBConfig) error {
	if cfg.RootfsDir == "" {
		return fmt.Errorf("rootfs directory is required")
	}
	if !cfg.UEFI && cfg.Device == "" {
		return fmt.Errorf("device is required for BIOS grub-install (e.g. /dev/sda)")
	}

	bootDir := filepath.Join(cfg.RootfsDir, "boot")
	grubDir := filepath.Join(bootDir, "grub")

	if err := os.MkdirAll(grubDir, 0755); err != nil {
		return fmt.Errorf("mkdir grub dir: %w", err)
	}

	grubCfg := `set timeout=5
set default=0

menuentry "Agnostikos Linux" {
  linux /boot/vmlinuz root=/dev/ram0 quiet
  initrd /boot/initramfs.img
}
`
	grubCfgPath := filepath.Join(grubDir, "grub.cfg")
	if err := os.WriteFile(grubCfgPath, []byte(grubCfg), 0644); err != nil {
		return fmt.Errorf("write grub.cfg: %w", err)
	}
	fmt.Printf("[grub] grub.cfg created at %s\n", grubCfgPath)

	if cfg.UEFI {
		efiDir := filepath.Join(bootDir, "efi", "EFI", "BOOT")
		if err := os.MkdirAll(efiDir, 0755); err != nil {
			return fmt.Errorf("mkdir efi dir: %w", err)
		}

		efiStub := "#!/bin/sh\n# Placeholder EFI stub - replace with real grub-install output\n# Run: grub-install --target=x86_64-efi --root-directory=" + cfg.RootfsDir + "\necho \"This is a placeholder EFI binary. Run grub-install to create the real one.\"\n"
		efiPath := filepath.Join(efiDir, "BOOTx64.EFI")
		if err := os.WriteFile(efiPath, []byte(efiStub), 0755); err != nil {
			return fmt.Errorf("write BOOTx64.EFI: %w", err)
		}
		fmt.Printf("[grub] UEFI directory structure created at %s\n", efiDir)

		if hasGrubInstall() {
			fmt.Println("[grub] grub-install found, attempting UEFI installation...")
			grubInstCmd := exec.CommandContext(ctx, "grub-install",
				"--target=x86_64-efi",
				"--root-directory="+cfg.RootfsDir,
				"--boot-directory="+bootDir,
				"--efi-directory="+filepath.Join(bootDir, "efi"),
				"--no-nvram",
			)
			grubInstCmd.Stdout, grubInstCmd.Stderr = os.Stdout, os.Stderr
			if err := grubInstCmd.Run(); err != nil {
				fmt.Printf("[grub] warn: grub-install UEFI failed: %v\n", err)
			} else {
				fmt.Println("[grub] grub-install (UEFI) completed")
			}
		} else {
			fmt.Println("[grub] grub-install not found; BOOTx64.EFI is a placeholder")
		}
	} else {
		if hasGrubInstall() {
			fmt.Printf("[grub] grub-install found, attempting BIOS installation on %s...\n", cfg.Device)
			grubInstCmd := exec.CommandContext(ctx, "grub-install",
				"--target=i386-pc",
				"--root-directory="+cfg.RootfsDir,
				"--boot-directory="+bootDir,
				cfg.Device,
			)
			grubInstCmd.Stdout, grubInstCmd.Stderr = os.Stdout, os.Stderr
			if err := grubInstCmd.Run(); err != nil {
				return fmt.Errorf("grub-install BIOS on %s: %w", cfg.Device, err)
			}
			fmt.Printf("[grub] grub-install (BIOS) completed on %s\n", cfg.Device)
		} else {
			fmt.Println("[grub] grub-install not found; grub.cfg created without bootloader installation")
		}
	}

	return nil
}

// hasGrubInstall verifica se grub-install está disponível no PATH
func hasGrubInstall() bool {
	_, err := exec.LookPath("grub-install")
	return err == nil
}
