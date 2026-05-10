package bootstrap

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GRUBConfig contém os parâmetros de instalação do GRUB
type GRUBConfig struct {
	RootfsDir    string // ex: "/mnt/data"
	Device       string // ex: "/dev/sda" — obrigatório para BIOS (disco base, sem número de partição)
	UEFI         bool   // true = x86_64-efi, false = i386-pc
	EFIPartition string // ex: "/dev/nvme0n1p1" — se definido, monta a ESP antes do grub-install
	Strict       bool   // true = falha do grub-install retorna erro; false = apenas warn (seguro para testes)
}

// findMountPoint retorna o ponto de montagem atual de um dispositivo lendo /proc/mounts.
// Retorna "" se o dispositivo não estiver montado.
func findMountPoint(device string) (string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("open /proc/mounts: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == device {
			return fields[1], nil
		}
	}
	return "", scanner.Err()
}

// mountESP monta a ESP em efiMountDir.
// Se o dispositivo já estiver montado em outro lugar, usa bind mount.
// Retorna true se o mount foi feito (e portanto o caller deve desmontar).
func mountESP(ctx context.Context, device, efiMountDir string) (bool, error) {
	existing, err := findMountPoint(device)
	if err != nil {
		return false, fmt.Errorf("check existing mounts: %w", err)
	}

	// Já montada exatamente no destino — não precisa fazer nada
	if existing == efiMountDir {
		fmt.Printf("[grub] ESP %s already mounted at %s, reusing\n", device, efiMountDir)
		return false, nil
	}

	// Já montada em outro ponto — bind mount
	if existing != "" {
		fmt.Printf("[grub] ESP %s already mounted at %s, bind-mounting to %s\n", device, existing, efiMountDir)
		if out, err := exec.CommandContext(ctx, "mount", "--bind", existing, efiMountDir).CombinedOutput(); err != nil {
			return false, fmt.Errorf("bind-mount %s -> %s: %s: %w", existing, efiMountDir, strings.TrimSpace(string(out)), err)
		}
		return true, nil
	}

	// Não montada — mount normal
	fmt.Printf("[grub] mounting ESP %s -> %s\n", device, efiMountDir)
	if out, err := exec.CommandContext(ctx, "mount", device, efiMountDir).CombinedOutput(); err != nil {
		return false, fmt.Errorf("mount ESP %s: %s: %w", device, strings.TrimSpace(string(out)), err)
	}
	return true, nil
}

// InstallGRUB configura o GRUB no RootFS (BIOS ou UEFI)
func InstallGRUB(ctx context.Context, cfg GRUBConfig) error {
	if cfg.RootfsDir == "" {
		return errors.New("rootfs directory is required")
	}
	if !cfg.UEFI && cfg.Device == "" {
		return errors.New("device is required for BIOS grub-install (e.g. /dev/sda)")
	}

	bootDir := filepath.Join(cfg.RootfsDir, "boot")
	grubDir := filepath.Join(bootDir, "grub")

	if err := os.MkdirAll(grubDir, 0755); err != nil {
		return fmt.Errorf("mkdir grub dir: %w", err)
	}

	// console=tty0     → saída vai para o VGA virtual da VM (virt-manager/SPICE/VNC)
	// quiet loglevel=3 → suprime mensagens de debug do kernel (kworker/dying etc.)
	grubCfg := `set timeout=5
set default=0

menuentry "Agnostikos Linux" {
  linux /boot/vmlinuz root=/dev/ram0 console=tty0 quiet loglevel=3
  initrd /boot/initramfs.img
}
`
	grubCfgPath := filepath.Join(grubDir, "grub.cfg")
	if err := os.WriteFile(grubCfgPath, []byte(grubCfg), 0644); err != nil {
		return fmt.Errorf("write grub.cfg: %w", err)
	}
	fmt.Printf("[grub] grub.cfg created at %s\n", grubCfgPath)

	if cfg.UEFI {
		efiMountDir := filepath.Join(bootDir, "efi")
		if err := os.MkdirAll(efiMountDir, 0755); err != nil {
			return fmt.Errorf("mkdir efi mount dir: %w", err)
		}

		efiMounted := false
		if cfg.EFIPartition != "" {
			var err error
			efiMounted, err = mountESP(ctx, cfg.EFIPartition, efiMountDir)
			if err != nil {
				return fmt.Errorf("mount ESP: %w", err)
			}
			if efiMounted {
				defer func() {
					fmt.Printf("[grub] unmounting ESP %s\n", efiMountDir)
					_ = exec.CommandContext(ctx, "umount", efiMountDir).Run()
				}()
			}
		}

		efiDir := filepath.Join(efiMountDir, "EFI", "BOOT")
		if err := os.MkdirAll(efiDir, 0755); err != nil {
			return fmt.Errorf("mkdir efi dir: %w", err)
		}

		if !efiMounted && cfg.EFIPartition == "" {
			efiStub := "#!/bin/sh\n# Placeholder EFI stub - replace with real grub-install output\n# Run: grub-install --target=x86_64-efi --root-directory=" + cfg.RootfsDir + "\necho \"This is a placeholder EFI binary. Run grub-install to create the real one.\"\n"
			efiPath := filepath.Join(efiDir, "BOOTx64.EFI")
			if err := os.WriteFile(efiPath, []byte(efiStub), 0755); err != nil {
				return fmt.Errorf("write BOOTx64.EFI: %w", err)
			}
		}
		fmt.Printf("[grub] UEFI directory structure ready at %s\n", efiDir)

		if hasGrubInstall() {
			fmt.Println("[grub] grub-install found, attempting UEFI installation...")
			grubInstCmd := exec.CommandContext(ctx, "grub-install",
				"--target=x86_64-efi",
				"--root-directory="+cfg.RootfsDir,
				"--boot-directory="+bootDir,
				"--efi-directory="+efiMountDir,
				"--bootloader-id=Agnostikos",
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
				if cfg.Strict {
					return fmt.Errorf("grub-install BIOS on %s: %w", cfg.Device, err)
				}
				fmt.Printf("[grub] warn: grub-install BIOS on %s failed: %v\n", cfg.Device, err)
			} else {
				fmt.Printf("[grub] grub-install (BIOS) completed on %s\n", cfg.Device)
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
