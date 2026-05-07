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
		if err := setupGRUBUEFI(isoDir, bootDir, workDir, cfg); err != nil {
			return err
		}
	} else {
		if err := setupIsolinux(isoDir, cfg); err != nil {
			return err
		}
	}

	return runXorriso(isoDir, workDir, cfg)
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

// setupGRUBUEFI cria a estrutura UEFI correta:
//  1. Gera BOOTX64.EFI via grub-mkstandalone com grub.cfg embutido
//  2. Cria efi.img (imagem FAT de 20MB) contendo EFI/BOOT/BOOTX64.EFI
//     grub-mkstandalone com --fonts=unicode gera ~3-4MB; 20MB dá folga segura
func setupGRUBUEFI(isoDir, bootDir, workDir string, cfg ISOConfig) error {
	for _, tool := range []string{"grub-mkstandalone", "mformat", "mcopy"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found — install grub-efi-amd64-bin and mtools", tool)
		}
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
	grubCfgPath := filepath.Join(grubDir, "grub.cfg")
	if err := os.WriteFile(grubCfgPath, []byte(grubCfg), 0644); err != nil {
		return fmt.Errorf("write grub.cfg: %w", err)
	}

	// Gera BOOTX64.EFI com grub.cfg embutido
	efiBin := filepath.Join(workDir, "BOOTX64.EFI")
	cmd := exec.Command("grub-mkstandalone",
		"-O", "x86_64-efi",
		"--fonts=unicode",
		"-o", efiBin,
		"boot/grub/grub.cfg="+grubCfgPath,
	)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("grub-mkstandalone: %w", err)
	}

	// Verifica tamanho real do EFI e dimensiona a imagem FAT com folga (2x + 2MB)
	efiBinInfo, err := os.Stat(efiBin)
	if err != nil {
		return fmt.Errorf("stat BOOTX64.EFI: %w", err)
	}
	efiSizeMB := (efiBinInfo.Size()/(1024*1024) + 1) * 2 + 2
	if efiSizeMB < 10 {
		efiSizeMB = 10
	}
	fmt.Printf("[iso] BOOTX64.EFI size: %d bytes — allocating %dMB FAT image\n", efiBinInfo.Size(), efiSizeMB)

	// Cria imagem FAT — mtools requer sintaxe ::path com -i
	efiImg := filepath.Join(workDir, "efi.img")
	run := func(name string, args ...string) error {
		c := exec.Command(name, args...)
		out, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %v: %w\noutput: %s", name, args, err, string(out))
		}
		return nil
	}

	if err := run("dd", "if=/dev/zero", "of="+efiImg, "bs=1M", fmt.Sprintf("count=%d", efiSizeMB)); err != nil {
		return err
	}
	if err := run("mformat", "-i", efiImg, "-F", "::"); err != nil {
		return err
	}
	if err := run("mmd", "-i", efiImg, "::EFI"); err != nil {
		return err
	}
	if err := run("mmd", "-i", efiImg, "::EFI/BOOT"); err != nil {
		return err
	}
	if err := run("mcopy", "-i", efiImg, efiBin, "::EFI/BOOT/BOOTX64.EFI"); err != nil {
		return err
	}

	// Copia efi.img para a árvore da ISO
	efiImgDest := filepath.Join(grubDir, "efi.img")
	imgData, err := os.ReadFile(efiImg)
	if err != nil {
		return fmt.Errorf("read efi.img: %w", err)
	}
	if err := os.WriteFile(efiImgDest, imgData, 0644); err != nil {
		return fmt.Errorf("write efi.img to iso tree: %w", err)
	}

	fmt.Printf("[iso] EFI image created: %s (%d bytes)\n", efiImgDest, len(imgData))
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

func runXorriso(isoDir, workDir string, cfg ISOConfig) error {
	args := []string{"-as", "mkisofs", "-o", cfg.Output, "-V", cfg.BootLabel, "-J", "-R"}
	if cfg.UEFI {
		args = append(args,
			"-eltorito-alt-boot",
			"-e", "boot/grub/efi.img",
			"-no-emul-boot",
			"-isohybrid-gpt-basdat",
		)
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
