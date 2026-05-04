package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallGRUB configura o GRUB no RootFS (BIOS ou UEFI)
func InstallGRUB(ctx context.Context, rootfsDir string, uefi bool) error {
	if rootfsDir == "" {
		return fmt.Errorf("rootfs directory is required")
	}

	bootDir := filepath.Join(rootfsDir, "boot")
	grubDir := filepath.Join(bootDir, "grub")

	// Criar diretório do GRUB
	if err := os.MkdirAll(grubDir, 0755); err != nil {
		return fmt.Errorf("mkdir grub dir: %w", err)
	}

	// Criar grub.cfg
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

	// Se UEFI, criar diretório EFI e placeholder BOOTx64.EFI
	if uefi {
		efiDir := filepath.Join(bootDir, "efi", "EFI", "BOOT")
		if err := os.MkdirAll(efiDir, 0755); err != nil {
			return fmt.Errorf("mkdir efi dir: %w", err)
		}

		// Placeholder BOOTx64.EFI (script stub que indica a necessidade de grub-install real)
		efiStub := `#!/bin/sh
# Placeholder EFI stub - replace with real grub-install output
# Run: grub-install --target=x86_64-efi --root-directory=` + rootfsDir + `
echo "This is a placeholder EFI binary. Run grub-install to create the real one."
`
		efiPath := filepath.Join(efiDir, "BOOTx64.EFI")
		if err := os.WriteFile(efiPath, []byte(efiStub), 0755); err != nil {
			return fmt.Errorf("write BOOTx64.EFI: %w", err)
		}
		fmt.Printf("[grub] UEFI directory structure created at %s\n", efiDir)

		// Tentar grub-install se disponível (não fatal se falhar)
		if hasGrubInstall() {
			fmt.Println("[grub] grub-install found, attempting UEFI installation...")
			grubInstCmd := exec.CommandContext(ctx, "grub-install",
				"--target=x86_64-efi",
				"--root-directory="+rootfsDir,
				"--boot-directory="+bootDir,
			)
			grubInstCmd.Stdout, grubInstCmd.Stderr = os.Stdout, os.Stderr
			if err := grubInstCmd.Run(); err != nil {
				fmt.Printf("[grub] warn: grub-install (UEFI) failed: %v\n", err)
				fmt.Println("[grub] grub.cfg and EFI structure created; grub-install requires root/device")
			} else {
				fmt.Println("[grub] grub-install (UEFI) completed")
			}
		} else {
			fmt.Println("[grub] grub-install not found; BOOTx64.EFI is a placeholder")
		}
	} else {
		// BIOS mode: tentar grub-install se disponível (não fatal se falhar)
		if hasGrubInstall() {
			fmt.Println("[grub] grub-install found, attempting BIOS installation...")
			grubInstCmd := exec.CommandContext(ctx, "grub-install",
				"--target=i386-pc",
				"--root-directory="+rootfsDir,
				"--boot-directory="+bootDir,
			)
			grubInstCmd.Stdout, grubInstCmd.Stderr = os.Stdout, os.Stderr
			if err := grubInstCmd.Run(); err != nil {
				fmt.Printf("[grub] warn: grub-install (BIOS) failed: %v\n", err)
				fmt.Println("[grub] grub.cfg created; grub-install requires root/device")
			} else {
				fmt.Println("[grub] grub-install (BIOS) completed")
			}
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
